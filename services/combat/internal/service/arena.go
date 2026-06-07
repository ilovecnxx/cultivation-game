package service

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"sync"
	"time"

	"cultivation-game/services/combat/internal/model"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// ---------- 常量 ----------

const (
	BaseScore          = 1000 // 初始积分
	WinBaseScore       = 25   // 胜利基础加分
	LoseBaseScore      = -10  // 失败扣分
	DailyFirstWinBonus = 50   // 每日首胜额外加分

	Streak2Bonus = 5  // 2连胜额外加分
	Streak3Bonus = 10 // 3连胜(及以上)额外加分
	Streak5Bonus = 20 // 5连胜(及以上)额外加分

	MatchBaseRange      = 100                // 基础匹配积分差范围
	MatchExpandInterval = 30                 // 多少秒后开始扩大匹配范围
	MatchExpandStep     = 50                 // 每次扩大多少分
	MatchProcessSeconds = 3                  // 匹配处理间隔(秒)
	SeasonCheckMinutes  = 1                  // 赛季检查间隔(分钟)
	MaxRankingsReturn   = 100                // 排行榜最多返回
)

// ---------- 段位配置 ----------

// tierThresholds 按积分确定段位和子段位
// 每个元素: {rank名称, 最低分, 最高分, 子段位数量}
var tierThresholds = []struct {
	Name      string
	MinScore  int
	MaxScore  int
	TierCount int
}{
	{"bronze", 1000, 1299, 3},
	{"silver", 1300, 1599, 3},
	{"gold", 1600, 1899, 3},
	{"diamond", 1900, 2199, 3},
	{"legend", 2200, math.MaxInt32, 1},
}

// ---------- 匹配队列条目 ----------

type queueEntry struct {
	PlayerID  string
	Score     int
	EnteredAt time.Time
}

// ArenaMatch 匹配成功结果
type ArenaMatch struct {
	PlayerA string
	PlayerB string
	ScoreA  int
	ScoreB  int
}

// ---------- ArenaService ----------

// ArenaService 竞技场服务: 匹配、积分、段位、排行
type ArenaService struct {
	mu            sync.RWMutex
	players       map[string]*model.ArenaPlayer // playerID -> 玩家数据
	queue         []*queueEntry
	history       []*model.MatchRecord // 最近对战记录(最多保留1000条)
	seasonSvc     *SeasonService
	matchCallback func(match *ArenaMatch)
	stopCh        chan struct{}
	redisClient   *redis.Client // Redis 持久化（可选）
}

// SetRedis 设置 Redis 客户端用于持久化
func (s *ArenaService) SetRedis(client *redis.Client) {
	s.redisClient = client
}

// Save 将竞技场数据持久化到 Redis
func (s *ArenaService) Save(ctx context.Context) error {
	if s.redisClient == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	pipe := s.redisClient.Pipeline()

	// 保存玩家数据
	playersData := make(map[string]string)
	for id, p := range s.players {
		data, err := json.Marshal(p)
		if err != nil {
			continue
		}
		playersData[id] = string(data)
	}
	if len(playersData) > 0 {
		pipe.HSet(ctx, "arena:players", playersData)
	}

	// 保存对战历史
	if len(s.history) > 0 {
		historyData, err := json.Marshal(s.history)
		if err == nil {
			pipe.Set(ctx, "arena:history", string(historyData), 30*24*time.Hour)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("保存竞技场数据失败")
		return err
	}
	return nil
}

// Load 从 Redis 恢复竞技场数据
func (s *ArenaService) Load(ctx context.Context) error {
	if s.redisClient == nil {
		return nil
	}

	// 恢复玩家数据
	playersData, err := s.redisClient.HGetAll(ctx, "arena:players").Result()
	if err == nil && len(playersData) > 0 {
		s.mu.Lock()
		for id, raw := range playersData {
			var p model.ArenaPlayer
			if err := json.Unmarshal([]byte(raw), &p); err == nil {
				s.players[id] = &p
			}
		}
		s.mu.Unlock()
		log.Info().Int("count", len(playersData)).Msg("从 Redis 恢复竞技场玩家数据")
	}

	// 恢复对战历史
	historyRaw, err := s.redisClient.Get(ctx, "arena:history").Result()
	if err == nil && historyRaw != "" {
		var history []*model.MatchRecord
		if err := json.Unmarshal([]byte(historyRaw), &history); err == nil {
			s.mu.Lock()
			s.history = history
			s.mu.Unlock()
			log.Info().Int("count", len(history)).Msg("从 Redis 恢复竞技场对战历史")
		}
	}

	return nil
}

// StartAutoSave 启动定期自动保存
func (s *ArenaService) StartAutoSave(ctx context.Context) {
	if s.redisClient == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = s.Save(ctx)
			case <-s.stopCh:
				_ = s.Save(context.Background())
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// NewArenaService 创建竞技场服务
//
//	seasonDays: 赛季持续天数
//	callback: 匹配成功回调(在独立goroutine中调用)
func NewArenaService(seasonDays int, callback func(match *ArenaMatch)) *ArenaService {
	svc := &ArenaService{
		players:       make(map[string]*model.ArenaPlayer),
		queue:         make([]*queueEntry, 0),
		history:       make([]*model.MatchRecord, 0, 1000),
		seasonSvc:     NewSeasonService(seasonDays),
		matchCallback: callback,
		stopCh:        make(chan struct{}),
	}
	return svc
}

// StartLoop 启动后台协程: 匹配处理 + 赛季检查
func (s *ArenaService) StartLoop() {
	// 匹配协程
	go func() {
		matchTicker := time.NewTicker(MatchProcessSeconds * time.Second)
		defer matchTicker.Stop()
		for {
			select {
			case <-matchTicker.C:
				s.processMatches()
			case <-s.stopCh:
				return
			}
		}
	}()

	// 赛季检查由 ArenaTickerService.checkSeason 统一处理
	// 此处不再运行重复的赛季检查协程，避免与奖励发放逻辑竞争
}

// Stop 停止后台协程
func (s *ArenaService) Stop() {
	close(s.stopCh)
}

// ---------- 玩家管理 ----------

// GetOrCreatePlayer 获取或创建玩家竞技场数据
func (s *ArenaService) GetOrCreatePlayer(playerID string) *model.ArenaPlayer {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.players[playerID]
	if !ok {
		p = &model.ArenaPlayer{
			PlayerID: playerID,
			Score:    BaseScore,
			Rank:     "bronze",
			Tier:     3,
		}
		s.players[playerID] = p
	}
	return p
}

// GetPlayer 获取玩家数据(只读, 不创建)
func (s *ArenaService) GetPlayer(playerID string) *model.ArenaPlayer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.players[playerID]
}

// ---------- 匹配队列 ----------

// EnqueueMatch 加入竞技场匹配队列
func (s *ArenaService) EnqueueMatch(playerID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 已在队列中
	for _, e := range s.queue {
		if e.PlayerID == playerID {
			return false
		}
	}

	player, ok := s.players[playerID]
	if !ok {
		player = &model.ArenaPlayer{
			PlayerID: playerID,
			Score:    BaseScore,
			Rank:     "bronze",
			Tier:     3,
		}
		s.players[playerID] = player
	}

	s.queue = append(s.queue, &queueEntry{
		PlayerID:  playerID,
		Score:     player.Score,
		EnteredAt: time.Now(),
	})
	return true
}

// DequeueMatch 移出竞技场匹配队列
func (s *ArenaService) DequeueMatch(playerID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, e := range s.queue {
		if e.PlayerID == playerID {
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
			return true
		}
	}
	return false
}

// IsInQueue 检查玩家是否在匹配队列中
func (s *ArenaService) IsInQueue(playerID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.queue {
		if e.PlayerID == playerID {
			return true
		}
	}
	return false
}

// QueueSize 队列长度
func (s *ArenaService) QueueSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.queue)
}

// ---------- 匹配逻辑 ----------

// processMatches 处理一轮匹配(由定时器调用)
func (s *ArenaService) processMatches() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) < 2 {
		return
	}

	// 按等待时间排序(最久的先匹配)
	sort.Slice(s.queue, func(i, j int) bool {
		return s.queue[i].EnteredAt.Before(s.queue[j].EnteredAt)
	})

	now := time.Now()
	matched := make(map[string]bool)
	callbackList := make([]*ArenaMatch, 0)

	for i := 0; i < len(s.queue); i++ {
		if matched[s.queue[i].PlayerID] {
			continue
		}

		for j := i + 1; j < len(s.queue); j++ {
			if matched[s.queue[j].PlayerID] {
				continue
			}

			p1 := s.queue[i]
			p2 := s.queue[j]

			if s.canMatch(p1, p2, now) {
				callbackList = append(callbackList, &ArenaMatch{
					PlayerA: p1.PlayerID,
					PlayerB: p2.PlayerID,
					ScoreA:  p1.Score,
					ScoreB:  p2.Score,
				})
				matched[p1.PlayerID] = true
				matched[p2.PlayerID] = true
				break
			}
		}
	}

	// 移除已匹配的玩家
	newQueue := make([]*queueEntry, 0, len(s.queue))
	for _, e := range s.queue {
		if !matched[e.PlayerID] {
			newQueue = append(newQueue, e)
		}
	}
	s.queue = newQueue

	// 释放锁后执行回调(避免死锁)
	s.mu.Unlock()
	for _, m := range callbackList {
		if s.matchCallback != nil {
			s.matchCallback(m)
		}
	}
	s.mu.Lock()
}

// canMatch 判断两名玩家是否可以匹配
func (s *ArenaService) canMatch(p1, p2 *queueEntry, now time.Time) bool {
	scoreDiff := p1.Score - p2.Score
	if scoreDiff < 0 {
		scoreDiff = -scoreDiff
	}

	// 计算有效匹配范围: 基础 ±100, 等待 >=30s 后逐步扩大
	effectiveRange := MatchBaseRange

	// 取两人中较长的等待时间
	waitP1 := now.Sub(p1.EnteredAt)
	waitP2 := now.Sub(p2.EnteredAt)
	maxWait := waitP1
	if waitP2 > maxWait {
		maxWait = waitP2
	}

	if maxWait.Seconds() >= float64(MatchExpandInterval) {
		extraSeconds := maxWait.Seconds() - float64(MatchExpandInterval)
		bonusSteps := int(extraSeconds) / MatchExpandInterval
		effectiveRange += bonusSteps * MatchExpandStep
	}

	return scoreDiff <= effectiveRange
}

// ---------- 积分结算 ----------

// ScoreChangeResult 积分变化结果
type ScoreChangeResult struct {
	WinnerID       string `json:"winner_id"`
	LoserID        string `json:"loser_id"`
	WinnerChange   int    `json:"winner_change"`
	LoserChange    int    `json:"loser_change"`
	StreakBonus    int    `json:"streak_bonus"`
	DailyFirstWin  bool   `json:"daily_first_win"`
	WinnerNewScore int    `json:"winner_new_score"`
	LoserNewScore  int    `json:"loser_new_score"`
}

// RecordResult 记录对战结果, 计算积分变化
func (s *ArenaService) RecordResult(winnerID, loserID string, rounds int) *ScoreChangeResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	winner := s.getOrCreatePlayerLocked(winnerID)
	loser := s.getOrCreatePlayerLocked(loserID)

	// 保存对战前数据
	prevWinnerScore := winner.Score
	prevLoserScore := loser.Score
	winnerRank := winner.Rank
	loserRank := loser.Rank
	winnerTier := winner.Tier
	loserTier := loser.Tier

	// 1. 更新连胜/连负
	winner.Streak++
	loser.Streak = 0

	// 2. 计算胜者加分
	streakBonus := calcStreakBonus(winner.Streak)
	winnerChange := WinBaseScore + streakBonus

	// 3. 每日首胜检查
	today := time.Now().Format("2006-01-02")
	dailyFirstWin := false
	if winner.LastDailyDate != today {
		winner.LastDailyDate = today
		winner.DailyWinCount = 0
	}
	if winner.DailyWinCount == 0 {
		winnerChange += DailyFirstWinBonus
		dailyFirstWin = true
	}
	winner.DailyWinCount++

	// 4. 计算败者扣分
	loserChange := LoseBaseScore

	// 段位保护: 青铜/白银不掉分
	if loser.Rank == "bronze" || loser.Rank == "silver" {
		loserChange = 0
	}

	// 5. 应用积分变化
	winner.Score += winnerChange
	loser.Score += loserChange

	// 积分不低于下限
	if winner.Score < BaseScore {
		winner.Score = BaseScore
	}
	if loser.Score < BaseScore {
		loser.Score = BaseScore
	}

	// 6. 更新段位
	winner.Rank, winner.Tier = calcRankAndTier(winner.Score)
	loser.Rank, loser.Tier = calcRankAndTier(loser.Score)

	// 7. 更新赛季胜/负场
	winner.SeasonWin++
	loser.SeasonLose++

	// 8. 保存对战记录
	record := &model.MatchRecord{
		ID:           uuid.New().String(),
		PlayerA:      winnerID,
		PlayerB:      loserID,
		Winner:       winnerID,
		ScoreChangeA: winnerChange,
		ScoreChangeB: loserChange,
		RankA:        winnerRank,
		RankB:        loserRank,
		TierA:        winnerTier,
		TierB:        loserTier,
		PlayerAScore: prevWinnerScore,
		PlayerBScore: prevLoserScore,
		Rounds:       rounds,
		Timestamp:    time.Now().Unix(),
	}
	s.addHistoryLocked(record)

	log.Info().
		Str("winner", winnerID).
		Str("loser", loserID).
		Int("winner_change", winnerChange).
		Int("loser_change", loserChange).
		Bool("daily_first", dailyFirstWin).
		Int("winner_score", winner.Score).
		Int("loser_score", loser.Score).
		Msg("竞技场对战结算")

	return &ScoreChangeResult{
		WinnerID:      winnerID,
		LoserID:       loserID,
		WinnerChange:  winnerChange,
		LoserChange:   loserChange,
		StreakBonus:   streakBonus,
		DailyFirstWin: dailyFirstWin,
		WinnerNewScore: winner.Score,
		LoserNewScore:  loser.Score,
	}
}

// RecordDraw 记录平局(双方不扣分)
func (s *ArenaService) RecordDraw(playerA, playerB string, rounds int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pA := s.getOrCreatePlayerLocked(playerA)
	pB := s.getOrCreatePlayerLocked(playerB)

	record := &model.MatchRecord{
		ID:        uuid.New().String(),
		PlayerA:   playerA,
		PlayerB:   playerB,
		Winner:    "draw",
		RankA:     pA.Rank,
		RankB:     pB.Rank,
		TierA:     pA.Tier,
		TierB:     pB.Tier,
		Rounds:    rounds,
		Timestamp: time.Now().Unix(),
	}
	s.addHistoryLocked(record)

	log.Info().
		Str("playerA", playerA).
		Str("playerB", playerB).
		Msg("竞技场平局")
}

// ---------- 查询 ----------

// GetRankings 获取排行榜(Top 100)
func (s *ArenaService) GetRankings() []*model.ArenaPlayer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*model.ArenaPlayer, 0, len(s.players))
	for _, p := range s.players {
		list = append(list, p)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Score > list[j].Score
	})

	if len(list) > MaxRankingsReturn {
		list = list[:MaxRankingsReturn]
	}
	return list
}

// GetPlayerRanking 获取玩家排名(1-based), 未上榜返回 -1
func (s *ArenaService) GetPlayerRanking(playerID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type kv struct {
		id    string
		score int
	}
	list := make([]kv, 0, len(s.players))
	for _, p := range s.players {
		list = append(list, kv{id: p.PlayerID, score: p.Score})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].score > list[j].score
	})
	for i, entry := range list {
		if entry.id == playerID {
			return i + 1
		}
	}
	return -1
}

// GetHistory 获取对战记录(分页)
//
//	limit: 最多返回条数, 默认20
//	before: 可选, 只返回早于此时间戳的记录(用于翻页)
func (s *ArenaService) GetHistory(playerID string, limit int, before int64) []*model.MatchRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	result := make([]*model.MatchRecord, 0, limit)
	for i := len(s.history) - 1; i >= 0; i-- {
		r := s.history[i]
		if r.PlayerA != playerID && r.PlayerB != playerID {
			continue
		}
		if before > 0 && r.Timestamp >= before {
			continue
		}
		result = append(result, r)
		if len(result) >= limit {
			break
		}
	}
	return result
}

// GetSeasonService 获取赛季服务
func (s *ArenaService) GetSeasonService() *SeasonService {
	return s.seasonSvc
}

// GetAllPlayers 获取所有玩家数据的副本（用于赛季重置等场景）
func (s *ArenaService) GetAllPlayers() map[string]*model.ArenaPlayer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	players := make(map[string]*model.ArenaPlayer, len(s.players))
	for k, v := range s.players {
		players[k] = v
	}
	return players
}

// ---------- 内部辅助 ----------

func (s *ArenaService) getOrCreatePlayerLocked(playerID string) *model.ArenaPlayer {
	p, ok := s.players[playerID]
	if !ok {
		p = &model.ArenaPlayer{
			PlayerID: playerID,
			Score:    BaseScore,
			Rank:     "bronze",
			Tier:     3,
		}
		s.players[playerID] = p
	}
	return p
}

func (s *ArenaService) addHistoryLocked(record *model.MatchRecord) {
	s.history = append(s.history, record)
	// 最多保留 1000 条历史
	if len(s.history) > 1000 {
		s.history = s.history[len(s.history)-1000:]
	}
}

// calcRankAndTier 根据积分计算段位和子段位
func calcRankAndTier(score int) (string, int) {
	for _, t := range tierThresholds {
		if score >= t.MinScore && score <= t.MaxScore {
			if t.TierCount <= 1 {
				return t.Name, 1
			}
			// 每个子段位所占分数区间
			rangeSize := (t.MaxScore - t.MinScore + 1) / t.TierCount
			tierIndex := (score - t.MinScore) / rangeSize
			// tier: 3(低) -> 1(高)
			tier := t.TierCount - tierIndex
			if tier < 1 {
				tier = 1
			}
			if tier > t.TierCount {
				tier = t.TierCount
			}
			return t.Name, tier
		}
	}
	return "bronze", 3
}

// calcStreakBonus 根据连胜数计算额外加分
func calcStreakBonus(streak int) int {
	// streak 是已更新后的值(包含本场胜利)
	switch {
	case streak >= 5:
		return Streak5Bonus
	case streak >= 3:
		return Streak3Bonus
	case streak >= 2:
		return Streak2Bonus
	default:
		return 0
	}
}
