package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// ProtectionService 新手保护业务逻辑
type ProtectionService struct {
	repo       *mysql.ProtectionRepo
	playerRepo *mysql.PlayerRepo
	config     model.ProtectionConfig
	configMu   sync.RWMutex
	log        *zap.Logger
}

// NewProtectionService 创建 ProtectionService
func NewProtectionService(repo *mysql.ProtectionRepo, playerRepo *mysql.PlayerRepo, log *zap.Logger) *ProtectionService {
	s := &ProtectionService{
		repo:       repo,
		playerRepo: playerRepo,
		log:        log,
	}
	s.loadConfig()
	return s
}

// loadConfig 从 JSON 文件加载保护配置
func (s *ProtectionService) loadConfig() {
	data, err := os.ReadFile("internal/data/protection.json")
	if err != nil {
		s.log.Warn("读取保护配置文件失败, 使用默认配置", zap.Error(err))
		s.configMu.Lock()
		s.config = model.ProtectionConfig{
			NewbieProtectionHours:              72,
			PvpProtectionHours:                 168,
			ProtectionRealmMax:                 2,
			BreakthroughGraceCount:             3,
			BreakthroughGracePenaltyReduction:  0.8,
			FreeResurrectionCount:              5,
			LevelGapMax:                        3,
		}
		s.configMu.Unlock()
		return
	}
	var cfg model.ProtectionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		s.log.Warn("解析保护配置文件失败, 使用默认配置", zap.Error(err))
		s.configMu.Lock()
		s.config = model.ProtectionConfig{
			NewbieProtectionHours:              72,
			PvpProtectionHours:                 168,
			ProtectionRealmMax:                 2,
			BreakthroughGraceCount:             3,
			BreakthroughGracePenaltyReduction:  0.8,
			FreeResurrectionCount:              5,
			LevelGapMax:                        3,
		}
		s.configMu.Unlock()
		return
	}
	s.configMu.Lock()
	s.config = cfg
	s.configMu.Unlock()
	s.log.Info("保护配置加载完成", zap.Any("config", cfg))
}

// GetConfig 获取当前保护配置
func (s *ProtectionService) GetConfig() model.ProtectionConfig {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config
}

// CreateProtection 为新玩家创建保护记录（在帐号创建时调用）
func (s *ProtectionService) CreateProtection(ctx context.Context, playerID int64) error {
	cfg := s.GetConfig()
	now := time.Now()

	protectionUntil := now.Add(time.Duration(cfg.NewbieProtectionHours) * time.Hour)
	pvpProtectionUntil := now.Add(time.Duration(cfg.PvpProtectionHours) * time.Hour)

	rec := &model.PlayerProtection{
		PlayerID:               playerID,
		ProtectionUntil:        &protectionUntil,
		PvpProtectionUntil:     &pvpProtectionUntil,
		BreakthroughGraceCount: int32(cfg.BreakthroughGraceCount),
		FreeResurrectionCount:  int32(cfg.FreeResurrectionCount),
	}

	if err := s.repo.Create(rec); err != nil {
		return fmt.Errorf("创建新手保护记录失败: %w", err)
	}

	s.log.Info("新手保护已创建",
		zap.Int64("player_id", playerID),
		zap.Time("protection_until", protectionUntil),
		zap.Time("pvp_protection_until", pvpProtectionUntil),
	)
	return nil
}

// IsProtected 检查玩家是否处于通用保护状态
func (s *ProtectionService) IsProtected(playerID int64) (bool, error) {
	rec, err := s.repo.GetByPlayerID(playerID)
	if err != nil {
		return false, err
	}
	if rec == nil {
		return false, nil
	}

	// 检查境界是否超过保护上限
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return false, fmt.Errorf("查询玩家信息失败: %w", err)
	}
	if player == nil {
		return false, nil
	}

	cfg := s.GetConfig()
	if player.Realm >= int32(cfg.ProtectionRealmMax) {
		return false, nil
	}

	// 检查保护时间是否过期
	if rec.ProtectionUntil == nil || time.Now().After(*rec.ProtectionUntil) {
		return false, nil
	}

	return true, nil
}

// IsPvpProtected 检查玩家是否处于 PVP 保护状态
func (s *ProtectionService) IsPvpProtected(playerID int64) (bool, error) {
	rec, err := s.repo.GetByPlayerID(playerID)
	if err != nil {
		return false, err
	}
	if rec == nil {
		return false, nil
	}

	// 检查境界是否超过保护上限
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return false, fmt.Errorf("查询玩家信息失败: %w", err)
	}
	if player == nil {
		return false, nil
	}

	cfg := s.GetConfig()
	if player.Realm >= int32(cfg.ProtectionRealmMax) {
		return false, nil
	}

	// 检查 PVP 保护时间是否过期
	if rec.PvpProtectionUntil == nil || time.Now().After(*rec.PvpProtectionUntil) {
		return false, nil
	}

	return true, nil
}

// GetProtectionInfo 获取玩家保护状态详情
func (s *ProtectionService) GetProtectionInfo(playerID int64) (*model.ProtectionStatus, error) {
	rec, err := s.repo.GetByPlayerID(playerID)
	if err != nil {
		return nil, err
	}

	status := &model.ProtectionStatus{}

	if rec == nil {
		return status, nil
	}

	// 检查境界
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家信息失败: %w", err)
	}
	cfg := s.GetConfig()
	belowRealmCap := player != nil && player.Realm < int32(cfg.ProtectionRealmMax)

	now := time.Now()

	// 通用保护
	if belowRealmCap && rec.ProtectionUntil != nil && now.Before(*rec.ProtectionUntil) {
		status.Protected = true
		status.ProtectionRemainingSec = int64(math.Ceil(rec.ProtectionUntil.Sub(now).Seconds()))
	}

	// PVP 保护
	if belowRealmCap && rec.PvpProtectionUntil != nil && now.Before(*rec.PvpProtectionUntil) {
		status.PvpProtected = true
		status.PvpProtectionRemainingSec = int64(math.Ceil(rec.PvpProtectionUntil.Sub(now).Seconds()))
	}

	status.BreakthroughGraceRemaining = rec.BreakthroughGraceCount
	status.FreeResurrectionRemaining = rec.FreeResurrectionCount

	return status, nil
}

// UseBreakthroughGrace 消耗一次突破免罚次数，返回是否成功
func (s *ProtectionService) UseBreakthroughGrace(playerID int64) (bool, error) {
	rec, err := s.repo.GetByPlayerID(playerID)
	if err != nil {
		return false, err
	}
	if rec == nil {
		return false, nil
	}
	if rec.BreakthroughGraceCount <= 0 {
		return false, nil
	}

	newCount := rec.BreakthroughGraceCount - 1
	if err := s.repo.UpdateGraceCount(playerID, newCount); err != nil {
		return false, err
	}

	s.log.Info("突破免罚次数已消耗",
		zap.Int64("player_id", playerID),
		zap.Int32("remaining", newCount),
	)
	return true, nil
}

// GetBreakthroughGraceReduction 获取突破免罚减免比例（0.0-1.0）
func (s *ProtectionService) GetBreakthroughGraceReduction(playerID int64) (float64, error) {
	rec, err := s.repo.GetByPlayerID(playerID)
	if err != nil {
		return 0, err
	}
	if rec == nil {
		return 0, nil
	}

	cfg := s.GetConfig()
	if rec.BreakthroughGraceCount > 0 {
		return cfg.BreakthroughGracePenaltyReduction, nil
	}
	return 0, nil
}

// UseFreeResurrection 使用一次免费复活，返回是否成功
func (s *ProtectionService) UseFreeResurrection(playerID int64) (bool, error) {
	rec, err := s.repo.GetByPlayerID(playerID)
	if err != nil {
		return false, err
	}
	if rec == nil {
		return false, nil
	}
	if rec.FreeResurrectionCount <= 0 {
		return false, nil
	}

	newCount := rec.FreeResurrectionCount - 1
	if err := s.repo.UpdateResurrectionCount(playerID, newCount); err != nil {
		return false, err
	}

	s.log.Info("免费复活次数已消耗",
		zap.Int64("player_id", playerID),
		zap.Int32("remaining", newCount),
	)
	return true, nil
}

// GetFreeResurrectionCount 获取玩家剩余的免费复活次数
func (s *ProtectionService) GetFreeResurrectionCount(playerID int64) (int32, error) {
	rec, err := s.repo.GetByPlayerID(playerID)
	if err != nil {
		return 0, err
	}
	if rec == nil {
		return 0, nil
	}
	return rec.FreeResurrectionCount, nil
}

// CanBeTargetedBy 检查攻击者是否可以 PVP 攻击目标玩家
// 返回是否可以攻击，以及拒绝原因（空字符串表示允许）
func (s *ProtectionService) CanBeTargetedBy(attackerPlayerID, targetPlayerID int64) (bool, string, error) {
	// 检查目标是否有 PVP 保护
	targetProtected, err := s.IsPvpProtected(targetPlayerID)
	if err != nil {
		return false, "", err
	}
	if targetProtected {
		return false, "目标玩家处于新手保护期，无法进行PVP攻击", nil
	}

	// 获取双方境界信息
	targetPlayer, err := s.playerRepo.GetByID(targetPlayerID)
	if err != nil {
		return false, "", fmt.Errorf("查询目标玩家信息失败: %w", err)
	}
	attackerPlayer, err := s.playerRepo.GetByID(attackerPlayerID)
	if err != nil {
		return false, "", fmt.Errorf("查询攻击者信息失败: %w", err)
	}

	if targetPlayer == nil || attackerPlayer == nil {
		return false, "玩家信息不存在", nil
	}

	// 检查境界差距：高境界玩家不能攻击低境界太多
	cfg := s.GetConfig()
	realmGap := int(attackerPlayer.Realm - targetPlayer.Realm)
	if realmGap > cfg.LevelGapMax {
		return false, fmt.Sprintf("境界差距过大(%d > %d)，无法攻击", realmGap, cfg.LevelGapMax), nil
	}

	return true, "", nil
}

// ExpireGeneralProtection 主动过期通用保护（如玩家达到保护上限境界时调用）
func (s *ProtectionService) ExpireGeneralProtection(playerID int64) error {
	return s.repo.ExpireGeneralProtection(playerID)
}

// ExpirePvpProtection 主动过期 PVP 保护
func (s *ProtectionService) ExpirePvpProtection(playerID int64) error {
	return s.repo.ExpirePvpProtection(playerID)
}
