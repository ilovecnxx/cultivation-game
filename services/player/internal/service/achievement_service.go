package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// AchievementService 成就业务逻辑
type AchievementService struct {
	repo     *mysql.AchievementRepo
	configs  []*model.Achievement // 静态成就配置，按 ID 索引
	configMu sync.RWMutex
	log      *zap.Logger
}

// NewAchievementService 创建 AchievementService
func NewAchievementService(repo *mysql.AchievementRepo, log *zap.Logger) *AchievementService {
	s := &AchievementService{
		repo:    repo,
		configs: make([]*model.Achievement, 0),
		log:     log,
	}
	s.loadConfig()
	return s
}

// loadConfig 从 JSON 文件加载成就配置
func (s *AchievementService) loadConfig() {
	data, err := os.ReadFile("internal/data/achievements.json")
	if err != nil {
		s.log.Warn("读取成就配置文件失败, 使用空配置", zap.Error(err))
		return
	}
	var list []*model.Achievement
	if err := json.Unmarshal(data, &list); err != nil {
		s.log.Warn("解析成就配置文件失败", zap.Error(err))
		return
	}
	s.configMu.Lock()
	s.configs = list
	s.configMu.Unlock()
	s.log.Info("成就配置加载完成", zap.Int("count", len(list)))
}

// ReloadConfig 重新加载配置（可用作配置热更新）
func (s *AchievementService) ReloadConfig() {
	s.loadConfig()
}

// GetAllConfigs 获取所有成就配置
func (s *AchievementService) GetAllConfigs() []*model.Achievement {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	cp := make([]*model.Achievement, len(s.configs))
	copy(cp, s.configs)
	return cp
}

// GetConfig 根据 ID 获取单个成就配置
func (s *AchievementService) GetConfig(id int) *model.Achievement {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	for _, c := range s.configs {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// GetAchievementIDs 获取所有成就配置 ID
func (s *AchievementService) GetAchievementIDs() []int {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	ids := make([]int, len(s.configs))
	for i, c := range s.configs {
		ids[i] = c.ID
	}
	return ids
}

// GetProgress 获取玩家所有成就进度（含配置信息）
func (s *AchievementService) GetProgress(ctx context.Context, playerID uint64) ([]*model.AchievementProgress, error) {
	records, err := s.repo.GetByPlayer(playerID)
	if err != nil {
		return nil, err
	}

	configs := s.GetAllConfigs()
	progressMap := make(map[int]*model.PlayerAchievement, len(records))
	for _, r := range records {
		progressMap[r.AchievementID] = r
	}

	result := make([]*model.AchievementProgress, 0, len(configs))
	for _, cfg := range configs {
		ap := &model.AchievementProgress{
			Achievement: *cfg,
			Progress:    0,
		}
		if pa, ok := progressMap[cfg.ID]; ok {
			ap.Progress = pa.Progress
			ap.Completed = pa.Completed
			ap.Claimed = pa.Claimed
		}
		result = append(result, ap)
	}
	return result, nil
}

// Claim 领取成就奖励
// 返回领取的奖励详情，如果未完成或已领取则返回错误
func (s *AchievementService) Claim(ctx context.Context, playerID uint64, achievementID int) (*model.ClaimResult, error) {
	cfg := s.GetConfig(achievementID)
	if cfg == nil {
		return nil, fmt.Errorf("成就配置不存在: %d", achievementID)
	}

	pa, err := s.repo.GetOne(playerID, achievementID)
	if err != nil {
		return nil, err
	}
	if pa == nil {
		return nil, fmt.Errorf("成就记录不存在")
	}
	if !pa.Completed {
		return nil, fmt.Errorf("成就未完成")
	}
	if pa.Claimed {
		return nil, fmt.Errorf("成就奖励已领取")
	}

	if err := s.repo.MarkClaimed(playerID, achievementID); err != nil {
		return nil, err
	}

	result := &model.ClaimResult{
		AchievementID: achievementID,
		Name:          cfg.Name,
		Title:         cfg.Reward.Title,
		Exp:           cfg.Reward.Exp,
		Money:         cfg.Reward.Money,
		AttrBonus:     cfg.Reward.AttrBonus,
	}

	// 如果奖励包含称号，自动装备
	if cfg.Reward.Title != "" {
		if err := s.repo.SaveTitle(playerID, cfg.Reward.Title); err != nil {
			s.log.Error("保存称号失败", zap.Uint64("player", playerID), zap.String("title", cfg.Reward.Title), zap.Error(err))
		}
	}

	return result, nil
}

// GetTitle 获取玩家当前称号
func (s *AchievementService) GetTitle(ctx context.Context, playerID uint64) (*model.TitleResponse, error) {
	pt, err := s.repo.GetTitle(playerID)
	if err != nil {
		return nil, err
	}
	if pt == nil {
		return &model.TitleResponse{PlayerID: playerID, Title: ""}, nil
	}
	return &model.TitleResponse{PlayerID: pt.PlayerID, Title: pt.Title}, nil
}

// UpdateProgress 外部调用：更新成就进度
// 检查是否达到目标值，若达到则标记完成
func (s *AchievementService) UpdateProgress(ctx context.Context, playerID uint64, achievementID int, newProgress int) error {
	cfg := s.GetConfig(achievementID)
	if cfg == nil {
		return fmt.Errorf("成就配置不存在: %d", achievementID)
	}

	completed := newProgress >= cfg.Target
	return s.repo.UpdateProgress(playerID, achievementID, newProgress, completed)
}

// InitPlayerAchievements 初始化新玩家的成就记录
func (s *AchievementService) InitPlayerAchievements(ctx context.Context, playerID uint64) error {
	ids := s.GetAchievementIDs()
	return s.repo.BatchInit(playerID, ids)
}
