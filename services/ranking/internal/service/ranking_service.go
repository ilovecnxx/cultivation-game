// Package service 实现排行榜业务逻辑：实时排名、批量更新、分数衰减、快照缓存。
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"cultivation-game/services/ranking/internal/config"
	"cultivation-game/services/ranking/internal/model"
)

// RankingRepository defines the data-access methods needed by RankingService.
// Implemented by *redis.RankingRepo in production; can be mocked in tests.
type RankingRepository interface {
	UpdateScore(ctx context.Context, rankingType model.RankingType, playerID uint64, score float64, nickname, realmName string) error
	GetTopN(ctx context.Context, rankingType model.RankingType, n int64) ([]*model.RankingEntry, error)
	GetRankingByPage(ctx context.Context, rankingType model.RankingType, page *model.PageRequest) ([]*model.RankingEntry, int64, error)
	GetPlayerRank(ctx context.Context, rankingType model.RankingType, playerID uint64) (int64, float64, error)
	GetPlayerInfo(ctx context.Context, rankingType model.RankingType, playerID uint64) (*model.RankingEntry, error)
	GetNeighbors(ctx context.Context, rankingType model.RankingType, playerID uint64, neighborCount int64) (above, below []*model.RankingEntry, err error)
	BatchUpdateScores(ctx context.Context, rankingType model.RankingType, entries []*model.RankingEntry) error
	UpdateActivity(ctx context.Context, rankingType model.RankingType, playerID uint64) error
	SetSnapshot(ctx context.Context, rankingType model.RankingType, entries []*model.RankingEntry, ttl time.Duration) error
	GetInactivePlayers(ctx context.Context, rankingType model.RankingType, deadline time.Time) ([]uint64, error)
	ApplyDecayToPlayer(ctx context.Context, rankingType model.RankingType, playerID uint64, decayRate float64) error
}

// 错误定义。
var (
	ErrInvalidRankingType = fmt.Errorf("无效的排行榜类型")
	ErrPlayerNotFound     = fmt.Errorf("玩家不在排行榜中")
)

// UpdateTask 分数更新任务（异步队列）。
type UpdateTask struct {
	RankingType model.RankingType
	PlayerID    uint64
	Score       float64
	Nickname    string
	RealmName   string
}

// RankingService 排行榜服务，处理所有排行榜业务逻辑。
type RankingService struct {
	repo RankingRepository
	cfg  *config.Config
	log  *slog.Logger

	// 异步更新通道
	updateCh chan *UpdateTask

	// 快照缓存（Top N）
	cacheMu     sync.RWMutex
	snapshots   map[model.RankingType]*model.Snapshot

	// 停止信号
	stopCh  chan struct{}
	wg      sync.WaitGroup
	stopped bool // 防止重复关闭 stopCh
}

// NewRankingService 创建 RankingService 并启动后台 Worker。
func NewRankingService(repo RankingRepository, cfg *config.Config, log *slog.Logger) *RankingService {
	svc := &RankingService{
		repo:      repo,
		cfg:       cfg,
		log:       log,
		updateCh:  make(chan *UpdateTask, cfg.UpdateBufferSize),
		snapshots: make(map[model.RankingType]*model.Snapshot),
		stopCh:    make(chan struct{}),
	}

	// 启动异步更新 Worker
	for i := 0; i < cfg.UpdateWorkerCount; i++ {
		svc.wg.Add(1)
		go svc.updateWorker(i)
	}

	// 启动快照缓存刷新
	svc.wg.Add(1)
	go svc.snapshotRefreshLoop()

	// 启动分数衰减检查
	if cfg.DecayCheckInterval > 0 {
		svc.wg.Add(1)
		go svc.decayCheckLoop()
	}

	log.Info("排行榜服务初始化完成",
		"update_workers", cfg.UpdateWorkerCount,
		"update_buffer", cfg.UpdateBufferSize,
		"cache_interval", cfg.CacheRefreshInterval,
		"decay_interval", cfg.DecayCheckInterval)

	return svc
}

// Stop 优雅关闭服务，等待所有后台 Worker 完成。
func (s *RankingService) Stop() {
	if s.stopped {
		return
	}
	s.stopped = true
	close(s.stopCh)
	s.wg.Wait()
	s.log.Info("排行榜服务已停止")
}

// ---- 公开 API ----

// UpdateScore 提交分数更新任务（异步）。
// 玩家升级/提升战力/获得财富时调用此方法，不阻塞调用方。
func (s *RankingService) UpdateScore(ctx context.Context, rankingType model.RankingType, playerID uint64, score float64, nickname, realmName string) error {
	if !model.IsValidType(string(rankingType)) {
		return ErrInvalidRankingType
	}

	select {
	case s.updateCh <- &UpdateTask{
		RankingType: rankingType,
		PlayerID:    playerID,
		Score:       score,
		Nickname:    nickname,
		RealmName:   realmName,
	}:
		return nil
	default:
		s.log.WarnContext(ctx, "更新通道已满，丢弃任务",
			"type", rankingType, "player_id", playerID)
		return fmt.Errorf("更新队列已满，请稍后重试")
	}
}

// GetRanking 分页获取排行榜。
func (s *RankingService) GetRanking(ctx context.Context, rankingType model.RankingType, page *model.PageRequest) ([]*model.RankingEntry, int32, error) {
	if !model.IsValidType(string(rankingType)) {
		return nil, 0, ErrInvalidRankingType
	}

	page.Normalize()
	entries, total, err := s.repo.GetRankingByPage(ctx, rankingType, page)
	if err != nil {
		return nil, 0, fmt.Errorf("获取排行榜失败: %w", err)
	}

	return entries, int32(total), nil
}

// GetTopPlayers 获取 Top N 玩家。
// 优先返回内存缓存，缓存过期时直接查询 Redis。
func (s *RankingService) GetTopPlayers(ctx context.Context, rankingType model.RankingType, limit int32) ([]*model.RankingEntry, error) {
	if !model.IsValidType(string(rankingType)) {
		return nil, ErrInvalidRankingType
	}

	if limit < 1 {
		limit = 10
	}
	if limit > int32(s.cfg.CacheTopN) {
		limit = int32(s.cfg.CacheTopN)
	}

	// 尝试从缓存获取
	s.cacheMu.RLock()
	snapshot, ok := s.snapshots[rankingType]
	s.cacheMu.RUnlock()

	if ok && snapshot != nil && time.Since(snapshot.RefreshedAt) < s.cfg.CacheRefreshInterval {
		topN := int(limit)
		if topN > len(snapshot.Entries) {
			topN = len(snapshot.Entries)
		}
		return snapshot.Entries[:topN], nil
	}

	// 缓存未命中，直接查询 Redis
	entries, err := s.repo.GetTopN(ctx, rankingType, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("获取 Top 玩家失败: %w", err)
	}

	// 被动更新缓存
	if len(entries) > 0 {
		s.cacheMu.Lock()
		s.snapshots[rankingType] = &model.Snapshot{
			Entries:     entries,
			RefreshedAt: time.Now(),
		}
		s.cacheMu.Unlock()
	}

	return entries, nil
}

// GetPlayerRank 获取玩家排名及周围邻居。
func (s *RankingService) GetPlayerRank(ctx context.Context, rankingType model.RankingType, playerID uint64) (*model.RankingEntry, []*model.RankingEntry, []*model.RankingEntry, error) {
	if !model.IsValidType(string(rankingType)) {
		return nil, nil, nil, ErrInvalidRankingType
	}

	rank, score, err := s.repo.GetPlayerRank(ctx, rankingType, playerID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("查询玩家排名失败: %w", err)
	}
	if rank < 0 {
		return nil, nil, nil, ErrPlayerNotFound
	}

	// 获取玩家信息
	info, err := s.repo.GetPlayerInfo(ctx, rankingType, playerID)
	if err != nil {
		s.log.WarnContext(ctx, "查询玩家信息失败", "player_id", playerID, "error", err)
		info = &model.RankingEntry{}
	}

	entry := &model.RankingEntry{
		PlayerID:  playerID,
		Nickname:  info.Nickname,
		RealmName: info.RealmName,
		Score:     score,
		Rank:      int32(rank),
		UpdatedAt: time.Now().Unix(),
	}

	// 获取邻居
	above, below, err := s.repo.GetNeighbors(ctx, rankingType, playerID, model.NeighborCount)
	if err != nil {
		s.log.WarnContext(ctx, "查询邻居信息失败", "player_id", playerID, "error", err)
		above = []*model.RankingEntry{}
		below = []*model.RankingEntry{}
	}

	return entry, above, below, nil
}

// BatchUpdate 批量同步更新分数（用于初始化或定时批量同步）。
func (s *RankingService) BatchUpdate(ctx context.Context, rankingType model.RankingType, entries []*model.RankingEntry) error {
	if !model.IsValidType(string(rankingType)) {
		return ErrInvalidRankingType
	}

	if len(entries) == 0 {
		return nil
	}

	return s.repo.BatchUpdateScores(ctx, rankingType, entries)
}

// ---- 后台 Workers ----

// updateWorker 异步分数更新 Worker。
// 从通道消费 UpdateTask，写入 Redis。
func (s *RankingService) updateWorker(id int) {
	defer s.wg.Done()
	s.log.Info("排行榜更新 Worker 已启动", "worker_id", id)

	for {
		select {
		case task := <-s.updateCh:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			err := s.repo.UpdateScore(ctx, task.RankingType, task.PlayerID, task.Score, task.Nickname, task.RealmName)
			if err != nil {
				s.log.ErrorContext(ctx, "异步更新排行榜失败",
					"worker_id", id,
					"type", task.RankingType,
					"player_id", task.PlayerID,
					"error", err)
			}

			// 更新活跃时间
			_ = s.repo.UpdateActivity(ctx, task.RankingType, task.PlayerID)

			cancel()

		case <-s.stopCh:
			s.log.Info("排行榜更新 Worker 已停止", "worker_id", id)
			return
		}
	}
}

// snapshotRefreshLoop 定期刷新 Top N 快照缓存。
func (s *RankingService) snapshotRefreshLoop() {
	s.log.Info("排行榜快照刷新循环已启动")
	ticker := time.NewTicker(s.cfg.CacheRefreshInterval)
	defer ticker.Stop()
	defer s.wg.Done()

	rankingTypes := []model.RankingType{
		model.RankingTypeRealm,
		model.RankingTypeCombatPower,
		model.RankingTypeWealth,
		model.RankingTypeSect,
	}

	for {
		select {
		case <-ticker.C:
			for _, rt := range rankingTypes {
				s.refreshSnapshot(context.Background(), rt)
			}

		case <-s.stopCh:
			s.log.Info("排行榜快照刷新循环已停止")
			return
		}
	}
}

// refreshSnapshot 刷新单个排行榜的 Top N 快照。
func (s *RankingService) refreshSnapshot(ctx context.Context, rt model.RankingType) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	entries, err := s.repo.GetTopN(ctx, rt, int64(s.cfg.CacheTopN))
	if err != nil {
		s.log.ErrorContext(ctx, "刷新排行榜快照失败", "type", rt, "error", err)
		return
	}

	s.cacheMu.Lock()
	s.snapshots[rt] = &model.Snapshot{
		Entries:     entries,
		RefreshedAt: time.Now(),
	}
	s.cacheMu.Unlock()

	// 同步缓存到 Redis（供其他实例读取）
	_ = s.repo.SetSnapshot(ctx, rt, entries, s.cfg.CacheRefreshInterval*2)
}

// decayCheckLoop 定期检查并应用分数衰减。
func (s *RankingService) decayCheckLoop() {
	s.log.Info("排行榜分数衰减检查循环已启动")
	ticker := time.NewTicker(s.cfg.DecayCheckInterval)
	defer ticker.Stop()
	defer s.wg.Done()

	for {
		select {
		case <-ticker.C:
			s.applyDecay(context.Background())

		case <-s.stopCh:
			s.log.Info("排行榜分数衰减检查循环已停止")
			return
		}
	}
}

// applyDecay 对所有启用衰减的排行榜应用分数衰减。
func (s *RankingService) applyDecay(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for _, rt := range []model.RankingType{
		model.RankingTypeCombatPower,
		model.RankingTypeWealth,
	} {
		lb := model.GetLeaderboard(rt)
		if lb == nil || !lb.DecayEnabled {
			continue
		}

		deadline := time.Now().AddDate(0, 0, -lb.DecayAfterDays)
		inactivePlayers, err := s.repo.GetInactivePlayers(ctx, rt, deadline)
		if err != nil {
			s.log.ErrorContext(ctx, "获取不活跃玩家失败", "type", rt, "error", err)
			continue
		}

		if len(inactivePlayers) == 0 {
			continue
		}

		// 每次检查只处理最多 500 名玩家，避免长时间阻塞
		batchSize := 500
		start := 0
		for start < len(inactivePlayers) {
			end := start + batchSize
			if end > len(inactivePlayers) {
				end = len(inactivePlayers)
			}

			for _, pid := range inactivePlayers[start:end] {
				decayRate := math.Min(lb.DecayRate, 0.5) // 单次衰减不超过 50%
				if err := s.repo.ApplyDecayToPlayer(ctx, rt, pid, decayRate); err != nil {
					s.log.WarnContext(ctx, "玩家分数衰减失败",
						"type", rt, "player_id", pid, "error", err)
				}
			}

			start = end
		}

		s.log.Info("分数衰减处理完成",
			"type", rt,
			"count", len(inactivePlayers))
	}
}
