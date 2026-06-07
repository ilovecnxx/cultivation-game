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
	mysqlRepo "cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// EnergyService 体力/能量业务逻辑（修炼打坐恢复版）
// 核心设计：体力通过修炼打坐恢复和丹药回复，随境界提高而增长
type EnergyService struct {
	energyRepo  *mysqlRepo.EnergyRepo
	playerSvc   *PlayerService
	config      *model.EnergyCostConfig
	configMutex sync.RWMutex
	log         *zap.Logger

	// 玩家已装备功法的体力回复加成（由外部服务同步更新）
	techniqueBonues     map[int64]float64 // playerID -> total regen bonus
	techniqueBonusMutex sync.RWMutex

	// 丹药冷却追踪：playerID -> last pill usage time
	pillCooldowns     map[int64]time.Time
	pillCooldownMutex sync.RWMutex
}

// NewEnergyService 创建 EnergyService
func NewEnergyService(energyRepo *mysqlRepo.EnergyRepo, playerSvc *PlayerService, log *zap.Logger) *EnergyService {
	return &EnergyService{
		energyRepo:      energyRepo,
		playerSvc:       playerSvc,
		config:          &model.EnergyCostConfig{},
		techniqueBonues: make(map[int64]float64),
		pillCooldowns:   make(map[int64]time.Time),
		log:             log,
	}
}

// LoadConfig 从 JSON 文件加载能量配置
func (s *EnergyService) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取能量配置文件失败: %w", err)
	}

	cfg := &model.EnergyCostConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("解析能量配置文件失败: %w", err)
	}

	s.configMutex.Lock()
	s.config = cfg
	s.configMutex.Unlock()

	s.log.Info("能量配置加载完成",
		zap.Int("realms", len(cfg.RealmMaxEnergy)),
		zap.Int("pill_tiers", len(cfg.PillTiers)),
		zap.Int("action_types", len(cfg.ActionCosts)),
	)
	return nil
}

// GetConfig 获取能量配置（线程安全）
func (s *EnergyService) GetConfig() *model.EnergyCostConfig {
	s.configMutex.RLock()
	defer s.configMutex.RUnlock()
	return s.config
}

// ============================================================
// 修炼打坐恢复 (Meditation Recovery)
// ============================================================

// RecoverFromMeditation 修炼打坐恢复体力
// 公式: energy_per_minute = base_regen(10) * realm_multiplier(1.0 + realm*0.3) * technique_bonus(1.0 + technique_level*0.1)
func (s *EnergyService) RecoverFromMeditation(ctx context.Context, playerID int64, durationMinutes int) (*model.MeditateResponse, error) {
	player, err := s.playerSvc.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("获取玩家信息失败: %w", err)
	}

	energy, err := s.GetOrCreateEnergy(ctx, playerID)
	if err != nil {
		return nil, err
	}

	// 计算每分钟回复量
	regenPerMin := s.CalculateMeditationRegen(player.Realm)

	// 功法加成
	techBonus := s.getTechniqueBonus(playerID)
	regenPerMin = int(float64(regenPerMin) * (1.0 + techBonus))

	// 计算本次修炼回复总量
	energyGained := regenPerMin * durationMinutes

	// 获取当前最大能量
	maxEnergy := s.GetRealmMaxEnergy(player.Realm, 0)

	// 应用回复
	energy.CurrentEnergy += energyGained
	if energy.CurrentEnergy > maxEnergy {
		energy.CurrentEnergy = maxEnergy
		energyGained = maxEnergy - (energy.CurrentEnergy - energyGained)
		if energyGained < 0 {
			energyGained = 0
		}
	}
	energy.MaxEnergy = maxEnergy

	// 更新最后修炼时间
	now := time.Now()
	energy.LastMeditationAt = &now

	if err := s.energyRepo.Update(energy); err != nil {
		return nil, fmt.Errorf("保存修炼回复数据失败: %w", err)
	}

	s.log.Info("修炼打坐恢复体力",
		zap.Int64("player_id", playerID),
		zap.Int("duration_min", durationMinutes),
		zap.Int("energy_gained", energyGained),
		zap.Int("regen_per_min", regenPerMin),
		zap.Int("current", energy.CurrentEnergy),
	)

	return &model.MeditateResponse{
		EnergyGained:  energyGained,
		CurrentEnergy: energy.CurrentEnergy,
		MaxEnergy:     maxEnergy,
		MeditationMin: durationMinutes,
		RegenPerMin:   regenPerMin,
	}, nil
}

// CalculateMeditationRegen 计算修炼打坐时的每分钟体力回复量
// 公式: energy_per_minute = base_regen(10) * (1.0 + realm * 0.3)
func (s *EnergyService) CalculateMeditationRegen(realm int32) int {
	cfg := s.GetConfig().MeditationConfig
	base := cfg.BaseRegenPerMinute
	if base <= 0 {
		base = 10
	}
	multiplier := cfg.RealmMultiplierBase + float64(realm)*cfg.RealmMultiplierPerLevel
	return int(float64(base) * multiplier)
}

// GetOfflineMeditationRecovery 计算离线期间的修炼回复量（从last_meditation_at至今）
func (s *EnergyService) GetOfflineMeditationRecovery(realm int32, lastMeditationAt time.Time) (int, int) {
	elapsed := time.Since(lastMeditationAt)
	elapsedMinutes := int(elapsed.Minutes())
	if elapsedMinutes <= 0 {
		return 0, 0
	}

	regenPerMin := s.CalculateMeditationRegen(realm)
	regenAmount := regenPerMin * elapsedMinutes

	return regenAmount, regenPerMin
}

// ============================================================
// 丹药回复 (Pill Recovery)
// ============================================================

// RecoverFromPill 使用体力丹药恢复能量
// 支持多品阶丹药：回体丹(低)、续命丹(中)、大还丹(高)、太乙回元丹(极)、九转续命丹(传说)
func (s *EnergyService) RecoverFromPill(ctx context.Context, playerID int64, pillID int, quantity int) (*model.PillRecoveryResponse, error) {
	player, err := s.playerSvc.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("获取玩家信息失败: %w", err)
	}

	energy, err := s.GetOrCreateEnergy(ctx, playerID)
	if err != nil {
		return nil, err
	}

	cfg := s.GetConfig()

	// 查找丹药配置
	var pillTier *model.PillTier
	for _, pt := range cfg.PillTiers {
		if pt.ID == pillID {
			pillTier = pt
			break
		}
	}
	if pillTier == nil {
		return nil, fmt.Errorf("未知的体力丹药ID: %d", pillID)
	}

	// 检查境界要求
	if int(player.Realm) < pillTier.RealmRequired {
		return nil, fmt.Errorf("境界不足，需要 %d 级境界才能使用 %s", pillTier.RealmRequired, pillTier.Name)
	}

	// 检查每日限制
	if energy.EnergyPillsUsedToday+quantity > cfg.PillUsageConfig.MaxPillsPerDay {
		return nil, fmt.Errorf("每日最多使用 %d 次体力丹药，今日已用 %d 次",
			cfg.PillUsageConfig.MaxPillsPerDay, energy.EnergyPillsUsedToday)
	}

	// 检查冷却
	s.pillCooldownMutex.RLock()
	lastUse, exists := s.pillCooldowns[playerID]
	s.pillCooldownMutex.RUnlock()
	if exists {
		elapsed := time.Since(lastUse)
		cooldown := time.Duration(cfg.PillUsageConfig.CooldownSeconds) * time.Second
		if elapsed < cooldown {
			remaining := int((cooldown - elapsed).Seconds())
			return nil, fmt.Errorf("丹药冷却中，请等待 %d 秒", remaining)
		}
	}

	// 计算回复量
	totalRestore := pillTier.RestoreAmount * quantity
	maxEnergy := s.GetRealmMaxEnergy(player.Realm, 0)

	// 应用回复（不超过上限）
	beforeRestore := energy.CurrentEnergy
	energy.CurrentEnergy += totalRestore
	if energy.CurrentEnergy > maxEnergy {
		energy.CurrentEnergy = maxEnergy
	}
	actualGained := energy.CurrentEnergy - beforeRestore
	if actualGained <= 0 {
		return nil, fmt.Errorf("体力已满，无需使用丹药")
	}

	energy.MaxEnergy = maxEnergy
	energy.EnergyPillsUsedToday += quantity

	if err := s.energyRepo.Update(energy); err != nil {
		return nil, fmt.Errorf("使用体力丹药失败: %w", err)
	}

	// 设置冷却
	s.pillCooldownMutex.Lock()
	s.pillCooldowns[playerID] = time.Now()
	s.pillCooldownMutex.Unlock()

	s.log.Info("使用体力丹药",
		zap.Int64("player_id", playerID),
		zap.Int("pill_id", pillID),
		zap.String("pill_name", pillTier.Name),
		zap.Int("tier", pillTier.Tier),
		zap.Int("quantity", quantity),
		zap.Int("energy_gained", actualGained),
	)

	return &model.PillRecoveryResponse{
		PillName:      pillTier.Name,
		PillTier:      pillTier.Tier,
		EnergyGained:  actualGained,
		CurrentEnergy: energy.CurrentEnergy,
		MaxEnergy:     maxEnergy,
		PillsUsed:     energy.EnergyPillsUsedToday,
		MaxPills:      cfg.PillUsageConfig.MaxPillsPerDay,
	}, nil
}

// GetPillCooldown 获取丹药剩余冷却时间（秒）
func (s *EnergyService) GetPillCooldown(playerID int64) int {
	s.pillCooldownMutex.RLock()
	defer s.pillCooldownMutex.RUnlock()
	lastUse, exists := s.pillCooldowns[playerID]
	if !exists {
		return 0
	}
	cfg := s.GetConfig().PillUsageConfig
	cooldown := time.Duration(cfg.CooldownSeconds) * time.Second
	elapsed := time.Since(lastUse)
	if elapsed >= cooldown {
		return 0
	}
	return int((cooldown - elapsed).Seconds())
}

// ============================================================
// 功法体力回复加成 (Technique Bonus)
// ============================================================

// SetTechniqueBonus 设置玩家装备功法的体力回复加成（由外部服务调用）
func (s *EnergyService) SetTechniqueBonus(playerID int64, bonus float64) {
	s.techniqueBonusMutex.Lock()
	defer s.techniqueBonusMutex.Unlock()
	s.techniqueBonues[playerID] = bonus
	s.log.Debug("设置功法体力回复加成",
		zap.Int64("player_id", playerID),
		zap.Float64("bonus", bonus))
}

// RemoveTechniqueBonus 移除玩家功法加成（卸下功法时调用）
func (s *EnergyService) RemoveTechniqueBonus(playerID int64) {
	s.techniqueBonusMutex.Lock()
	defer s.techniqueBonusMutex.Unlock()
	delete(s.techniqueBonues, playerID)
}

// GetTechniqueBonus 获取玩家当前功法体力回复加成
// 返回: 加成系数（如0.2表示+20%）
func (s *EnergyService) GetTechniqueBonus(playerID int64) float64 {
	s.techniqueBonusMutex.RLock()
	defer s.techniqueBonusMutex.RUnlock()
	return s.techniqueBonues[playerID]
}

// getTechniqueBonus 内部获取功法加成（线程安全）
func (s *EnergyService) getTechniqueBonus(playerID int64) float64 {
	s.techniqueBonusMutex.RLock()
	defer s.techniqueBonusMutex.RUnlock()
	return s.techniqueBonues[playerID]
}

// ============================================================
// 境界成长 (Realm Growth)
// ============================================================

// GetRealmMaxEnergy 根据修炼境界和炼体境界计算最大体力
// 修炼境界提供基础体力上限，炼体境界提供额外加成
func (s *EnergyService) GetRealmMaxEnergy(realm int32, bodyRealm int32) int {
	cfg := s.GetConfig()

	// 基础体力上限（按修炼境界）
	maxEnergy := 100 // 默认值
	if val, ok := cfg.RealmMaxEnergy[int(realm)]; ok {
		maxEnergy = val
	}

	// 炼体加成
	if bodyRealm > 0 {
		if bonus, ok := cfg.BodyCultivationMaxEnergy[int(bodyRealm)]; ok {
			maxEnergy += bonus
		}
	}

	return maxEnergy
}

// ============================================================
// 体力消耗 (Energy Costs)
// ============================================================

// ConsumeEnergy 消耗体力
// actionType: explore, dungeon, team_dungeon, gather, pvp, boss_fight, breakthrough_attempt
func (s *EnergyService) ConsumeEnergy(ctx context.Context, playerID int64, actionType string) (int, error) {
	cfg := s.GetConfig()
	cost, ok := cfg.ActionCosts[actionType]
	if !ok {
		return 0, fmt.Errorf("未知的行动类型: %s", actionType)
	}

	// 先确保记录存在
	_, err := s.GetOrCreateEnergy(ctx, playerID)
	if err != nil {
		return 0, err
	}

	// 原子扣减
	energy, err := s.energyRepo.UseEnergy(playerID, cost)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", actionType, err)
	}

	return energy.CurrentEnergy, nil
}

// CheckEnergy 检查是否有足够能量执行某个行动
func (s *EnergyService) CheckEnergy(ctx context.Context, playerID int64, actionType string) (bool, int, error) {
	cost, ok := s.GetConfig().ActionCosts[actionType]
	if !ok {
		return false, 0, fmt.Errorf("未知的行动类型: %s", actionType)
	}

	status, err := s.GetEnergy(ctx, playerID)
	if err != nil {
		return false, 0, err
	}

	if status.CurrentEnergy < cost {
		return false, cost - status.CurrentEnergy, nil
	}
	return true, 0, nil
}

// ============================================================
// 查询与通用方法
// ============================================================

// GetOrCreateEnergy 获取或创建玩家能量记录
func (s *EnergyService) GetOrCreateEnergy(ctx context.Context, playerID int64) (*model.PlayerEnergy, error) {
	energy, err := s.energyRepo.GetByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取能量记录失败: %w", err)
	}
	if energy != nil {
		return energy, nil
	}

	// 没有能量记录，创建默认（满能量）
	player, err := s.playerSvc.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("获取玩家信息失败: %w", err)
	}

	maxEnergy := s.GetRealmMaxEnergy(player.Realm, 0)
	energy = &model.PlayerEnergy{
		PlayerID:      playerID,
		CurrentEnergy: maxEnergy,
		MaxEnergy:     maxEnergy,
	}
	if err := s.energyRepo.Create(energy); err != nil {
		return nil, fmt.Errorf("创建能量记录失败: %w", err)
	}
	return energy, nil
}

// GetEnergy 获取当前能量（含离线修炼回复计算）
func (s *EnergyService) GetEnergy(ctx context.Context, playerID int64) (*model.EnergyStatus, error) {
	player, err := s.playerSvc.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("获取玩家信息失败: %w", err)
	}

	energy, err := s.GetOrCreateEnergy(ctx, playerID)
	if err != nil {
		return nil, err
	}

	// 获取最大体力（含境界加成）
	maxEnergy := s.GetRealmMaxEnergy(player.Realm, 0)

	// 计算离线修炼回复（如果玩家上次修炼后离线了，自动获得回复）
	if energy.LastMeditationAt != nil {
		offlineRegen, regenPerMin := s.GetOfflineMeditationRecovery(player.Realm, *energy.LastMeditationAt)
		if offlineRegen > 0 {
			// 功法加成
			techBonus := s.getTechniqueBonus(playerID)
			regenPerMin = int(float64(regenPerMin) * (1.0 + techBonus))
			offlineRegen = int(float64(offlineRegen) * (1.0 + techBonus))

			energy.CurrentEnergy += offlineRegen
			if energy.CurrentEnergy > maxEnergy {
				energy.CurrentEnergy = maxEnergy
			}
			energy.MaxEnergy = maxEnergy
			now := time.Now()
			energy.LastMeditationAt = &now
			_ = s.energyRepo.Update(energy)
		}
	}

	// 计算恢复到满所需时间
	remaining := maxEnergy - energy.CurrentEnergy
	regenPerMin := s.CalculateMeditationRegen(player.Realm)
	techBonus := s.getTechniqueBonus(playerID)
	effectiveRegenPerMin := int(float64(regenPerMin) * (1.0 + techBonus))

	hoursToFull := 0.0
	if effectiveRegenPerMin > 0 && remaining > 0 {
		minutesToFull := float64(remaining) / float64(effectiveRegenPerMin)
		hoursToFull = minutesToFull / 60.0
		hoursToFull = math.Round(hoursToFull*100) / 100
	}

	cfg := s.GetConfig()

	// 计算丹药冷却
	cooldownLeft := s.GetPillCooldown(playerID)

	var lastMedAt *int64
	if energy.LastMeditationAt != nil {
		ts := energy.LastMeditationAt.Unix()
		lastMedAt = &ts
	}

	return &model.EnergyStatus{
		CurrentEnergy:      energy.CurrentEnergy,
		MaxEnergy:          maxEnergy,
		MeditationRegenMin: effectiveRegenPerMin,
		RegenPerHour:       effectiveRegenPerMin * 60,
		HoursToFull:        hoursToFull,
		PillsUsed:          energy.EnergyPillsUsedToday,
		MaxPills:           cfg.PillUsageConfig.MaxPillsPerDay,
		PillCooldownLeft:   cooldownLeft,
		TechniqueBonus:     techBonus,
		LastMeditationAt:   lastMedAt,
	}, nil
}

// GetMaxEnergy 获取玩家最大能量
func (s *EnergyService) GetMaxEnergy(ctx context.Context, playerID int64) (int, error) {
	player, err := s.playerSvc.GetPlayer(ctx, playerID)
	if err != nil {
		return 0, fmt.Errorf("获取玩家信息失败: %w", err)
	}
	return s.GetRealmMaxEnergy(player.Realm, 0), nil
}
