// Package service 灵根进化系统
//
// 灵根品质: 杂品(灰) -> 人品(白) -> 地品(绿) -> 天品(蓝) -> 混沌(紫) -> 鸿蒙(金)
// 进化方式: 使用「灵根进化石」 + 境界要求 + 概率判定
// 灵根进化石来源: 势力争霸奖励 / 心魔塔高层 / 世界BOSS掉落
//
// 进化机制:
//   - 基础成功率 30%，每次轮回 +5% 成功率
//   - 失败时消耗进化石，有 10% 概率降级
//   - 进化成功时修炼速度 +20%/级，突破成功率 +2%/级
//   - 达到混沌(4)时解锁元素觉醒，获得元素伤害加成
package service

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SpiritEvolutionService 灵根进化业务
type SpiritEvolutionService struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewSpiritEvolutionService 创建 SpiritEvolutionService
func NewSpiritEvolutionService(db *gorm.DB, log *zap.Logger) *SpiritEvolutionService {
	return &SpiritEvolutionService{db: db, log: log}
}

// =============================================================================
//  核心业务方法
// =============================================================================

// GetSpiritStatus 获取玩家灵根状态（如不存在则初始化）
func (s *SpiritEvolutionService) GetSpiritStatus(ctx context.Context, playerID int64) (*model.PlayerSpirit, error) {
	var spirit model.PlayerSpirit
	err := s.db.WithContext(ctx).First(&spirit, playerID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			spirit = model.PlayerSpirit{
				PlayerID:      playerID,
				Quality:       model.SpiritQualityMisc,
				Reincarnations: 0,
				UpdatedAt:     time.Now().Unix(),
			}
			if createErr := s.db.WithContext(ctx).Create(&spirit).Error; createErr != nil {
				return nil, fmt.Errorf("初始化灵根失败: %w", createErr)
			}
		} else {
			return nil, fmt.Errorf("查询灵根状态失败: %w", err)
		}
	}
	return &spirit, nil
}

// EvolveByStone 使用灵根进化石尝试进化
//
// 流程:
//  1. 校验灵根未满
//  2. 校验境界要求
//  3. 扣减进化石
//  4. 计算成功率 (基础30% + 轮回次数*5%)
//  5. 随机判定成功/失败
//  6. 失败时判定是否降级 (10%)
//  7. 记录进化历史
//  8. 达到混沌后解锁元素觉醒
func (s *SpiritEvolutionService) EvolveByStone(ctx context.Context, playerID int64) (*model.EvolutionResult, error) {
	spirit, err := s.GetSpiritStatus(ctx, playerID)
	if err != nil {
		return nil, err
	}

	// --- 1. 校验最高品质 ---
	if spirit.Quality >= model.SpiritQualityHongMeng {
		return nil, fmt.Errorf("灵根已达最高品质「鸿蒙灵根」，无法继续进化")
	}

	target := spirit.GetEvolutionTarget()
	needStones := model.SpiritEvolutionCost[spirit.Quality]

	// --- 2. 境界要求 ---
	reqRealm, hasRealmReq := model.EvolutionRealmRequirement[spirit.Quality]
	if hasRealmReq {
		var player model.Player
		if err := s.db.WithContext(ctx).First(&player, playerID).Error; err != nil {
			return nil, fmt.Errorf("查询玩家信息失败: %w", err)
		}
		if player.Realm < reqRealm {
			return nil, fmt.Errorf("境界不足，当前【%s】，需要【%s】以上",
				model.RealmNames[player.Realm],
				model.RealmNames[reqRealm])
		}
	}

	// --- 3. 查询玩家境界（供后续使用）---
	var player model.Player
	var realmAtTime int32
	if err := s.db.WithContext(ctx).First(&player, playerID).Error; err == nil {
		realmAtTime = player.Realm
	}

	now := time.Now().Unix()

	// --- 4. 事务: 扣石头 + 更新灵根 + 保存历史 ---
	tx := s.db.WithContext(ctx).Begin()

	// 扣减进化石
	stonesErr := s.deductEvolutionStones(tx, playerID, needStones)
	if stonesErr != nil {
		tx.Rollback()
		return nil, stonesErr
	}

	// --- 5. 计算成功率 ---
	baseRate := model.EvolutionBaseSuccessRate
	reincarnationBonus := float64(spirit.Reincarnations) * model.ReincarnationSuccessBonus
	successRate := baseRate + reincarnationBonus
	if successRate > 1.0 {
		successRate = 1.0
	}

	// --- 6. 随机判定 ---
	rng := rand.Float64()
	success := rng < successRate

	var newQuality model.SpiritQuality
	var degraded bool

	if success {
		newQuality = target

		s.log.Info("灵根进化成功",
			zap.Int64("player_id", playerID),
			zap.String("from", model.SpiritQualityNames[spirit.Quality]),
			zap.String("to", model.SpiritQualityNames[newQuality]),
			zap.Float64("success_rate", successRate),
			zap.Float64("rand", rng),
		)
	} else {
		// 失败时判定降级
		degraded = rand.Float64() < model.DegradationChanceOnFail
		if degraded && spirit.Quality > model.SpiritQualityMisc {
			newQuality = spirit.Quality - 1

			s.log.Warn("灵根进化失败且降级",
				zap.Int64("player_id", playerID),
				zap.String("from", model.SpiritQualityNames[spirit.Quality]),
				zap.String("to", model.SpiritQualityNames[newQuality]),
				zap.Float64("success_rate", successRate),
				zap.Float64("rand", rng),
			)
		} else {
			newQuality = spirit.Quality

			s.log.Warn("灵根进化失败",
				zap.Int64("player_id", playerID),
				zap.String("quality", model.SpiritQualityNames[spirit.Quality]),
				zap.Float64("success_rate", successRate),
				zap.Float64("rand", rng),
			)
		}
	}

	// --- 7. 更新灵根状态 ---
	spirit.Quality = newQuality
	spirit.UpdatedAt = now
	if err := tx.Save(&spirit).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新灵根状态失败: %w", err)
	}

	// --- 8. 记录进化历史 ---
	history := model.EvolutionHistory{
		PlayerID:    playerID,
		FromQuality: spirit.Quality,
		ToQuality:   newQuality,
		Success:     success,
		Degraded:    degraded && !success,
		StonesUsed:  needStones,
		RealmAtTime: realmAtTime,
		SuccessRate: successRate,
		CreatedAt:   now,
	}
	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("保存进化历史失败: %w", err)
	}

	tx.Commit()

	// --- 9. 元素觉醒判定（在事务外执行，只读操作可接受）---
	var elementAwakening *model.PlayerElementAwakening
	if success && newQuality >= model.SpiritQualityChaos {
		awakening, awakenErr := s.tryUnlockElementAwakening(ctx, playerID, &player, now)
		if awakenErr == nil {
			elementAwakening = awakening
		}
	}

	result := &model.EvolutionResult{
		Spirit:           spirit,
		Success:          success,
		Degraded:         degraded && !success,
		SuccessRate:      successRate,
		StonesUsed:       needStones,
		FromQuality:      spirit.Quality,
		ToQuality:        newQuality,
		ElementAwakening: elementAwakening,
	}

	return result, nil
}

// GetStoneInfo 获取各品质所需进化石信息
func (s *SpiritEvolutionService) GetStoneInfo() *model.StoneInfoResponse {
	tiers := make([]model.StoneTierInfo, 0, len(model.SpiritEvolutionCost))
	for quality, stones := range model.SpiritEvolutionCost {
		realmReq := model.EvolutionRealmRequirement[quality]
		tiers = append(tiers, model.StoneTierInfo{
			Quality:      int(quality),
			QualityName:  model.SpiritQualityNames[quality],
			QualityColor: model.SpiritQualityColors[quality],
			Stones:       stones,
			RealmReq:     realmReq,
			RealmName:    model.RealmNames[realmReq],
		})
	}
	// 按品质升序排列
	sort.Slice(tiers, func(i, j int) bool {
		return tiers[i].Quality < tiers[j].Quality
	})

	return &model.StoneInfoResponse{
		ItemID:             model.SpiritEvolutionItemID,
		ItemName:           "灵根进化石",
		BaseSuccessRate:    model.EvolutionBaseSuccessRate,
		ReincarnationBonus: model.ReincarnationSuccessBonus,
		DegradationChance:  model.DegradationChanceOnFail,
		TierRequirements:   tiers,
	}
}

// GetEvolutionHistory 获取进化历史（分页）
func (s *SpiritEvolutionService) GetEvolutionHistory(ctx context.Context, playerID int64, page, pageSize int) ([]model.EvolutionHistory, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := s.db.WithContext(ctx).Model(&model.EvolutionHistory{}).
		Where("player_id = ?", playerID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询进化历史总数失败: %w", err)
	}

	var records []model.EvolutionHistory
	if err := s.db.WithContext(ctx).
		Where("player_id = ?", playerID).
		Order("id DESC").
		Limit(pageSize).Offset(offset).
		Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("查询进化历史失败: %w", err)
	}
	if records == nil {
		records = []model.EvolutionHistory{}
	}

	return records, total, nil
}

// GetElementAwakening 获取玩家元素觉醒信息
func (s *SpiritEvolutionService) GetElementAwakening(ctx context.Context, playerID int64) (*model.PlayerElementAwakening, error) {
	var awakening model.PlayerElementAwakening
	err := s.db.WithContext(ctx).First(&awakening, playerID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询元素觉醒信息失败: %w", err)
	}
	return &awakening, nil
}

// =============================================================================
//  内部辅助方法
// =============================================================================

// deductEvolutionStones 从背包扣减灵根进化石
func (s *SpiritEvolutionService) deductEvolutionStones(tx *gorm.DB, playerID int64, need int) error {
	var items []model.InventoryItem
	if err := tx.Where("player_id = ? AND item_id = ?", playerID, model.SpiritEvolutionItemID).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return fmt.Errorf("查询背包进化石失败: %w", err)
	}

	total := int32(0)
	for _, item := range items {
		total += item.Quantity
	}
	if total < int32(need) {
		return fmt.Errorf("灵根进化石不足，需要 %d 个，当前 %d 个", need, total)
	}

	remaining := int32(need)
	for _, item := range items {
		if remaining <= 0 {
			break
		}
		if item.Quantity > remaining {
			item.Quantity -= remaining
			remaining = 0
			if err := tx.Save(&item).Error; err != nil {
				return fmt.Errorf("扣减进化石失败: %w", err)
			}
		} else {
			remaining -= item.Quantity
			if err := tx.Delete(&item).Error; err != nil {
				return fmt.Errorf("删除进化石记录失败: %w", err)
			}
		}
	}

	return nil
}

// tryUnlockElementAwakening 尝试解锁元素觉醒（达到混沌品质后自动激活）
func (s *SpiritEvolutionService) tryUnlockElementAwakening(ctx context.Context, playerID int64, player *model.Player, now int64) (*model.PlayerElementAwakening, error) {
	// 检查是否已觉醒
	var existing model.PlayerElementAwakening
	err := s.db.WithContext(ctx).First(&existing, playerID).Error
	if err == nil {
		// 已经觉醒，直接返回
		return &existing, nil
	}

	if player.SpiritRoot <= model.SpiritRootNone {
		// 没有灵根类型，无法觉醒
		return nil, fmt.Errorf("玩家没有灵根类型，无法觉醒元素")
	}

	awakening := model.PlayerElementAwakening{
		PlayerID:    playerID,
		Element:     player.SpiritRoot,
		DamageBonus: model.ElementDamageBonus,
		ActivatedAt: now,
	}
	if err := s.db.WithContext(ctx).Create(&awakening).Error; err != nil {
		return nil, fmt.Errorf("激活元素觉醒失败: %w", err)
	}

	s.log.Info("元素觉醒激活",
		zap.Int64("player_id", playerID),
		zap.Int32("element", player.SpiritRoot),
		zap.Float64("damage_bonus", model.ElementDamageBonus),
	)

	return &awakening, nil
}

// =============================================================================
//  外部系统集成方法
// =============================================================================

// GetCultivationBonus 获取修炼速度加成（供修炼系统调用）
// 返回百分比加成，如 quality=2 返回 40.0 表示 +40%
func (s *SpiritEvolutionService) GetCultivationBonus(ctx context.Context, playerID int64) float64 {
	spirit, err := s.GetSpiritStatus(ctx, playerID)
	if err != nil {
		return 0
	}
	return float64(int(spirit.Quality)) * float64(model.SpiritSpeedBonusPerLevel)
}

// GetBreakthroughBonus 获取突破成功率加成（供突破系统调用）
// 返回百分比加成，如 quality=2 返回 4.0 表示 +4%
func (s *SpiritEvolutionService) GetBreakthroughBonus(ctx context.Context, playerID int64) float64 {
	spirit, err := s.GetSpiritStatus(ctx, playerID)
	if err != nil {
		return 0
	}
	return float64(int(spirit.Quality)) * float64(model.SpiritBreakthroughBonusPerLevel)
}

// TryReincarnationUpgrade 轮回后尝试自动提升灵根品质
// 每次轮回后调用，基础 5% + 已轮回次数 * 5%
func (s *SpiritEvolutionService) TryReincarnationUpgrade(ctx context.Context, playerID int64) (*model.PlayerSpirit, bool, error) {
	spirit, err := s.GetSpiritStatus(ctx, playerID)
	if err != nil {
		return nil, false, err
	}

	if spirit.Quality >= model.SpiritQualityHongMeng {
		return spirit, false, nil
	}

	chance := spirit.ReincarnationUpgradeChance()
	upgraded := rand.Float64() < chance

	now := time.Now().Unix()

	if upgraded {
		newQuality := spirit.GetEvolutionTarget()
		spirit.Quality = newQuality
		spirit.Reincarnations++
		spirit.UpdatedAt = now
		s.db.WithContext(ctx).Save(&spirit)

		s.log.Info("轮回自动提升灵根",
			zap.Int64("player_id", playerID),
			zap.String("quality", model.SpiritQualityNames[spirit.Quality]),
			zap.Float64("chance", chance),
		)

		// 自动元素觉醒
		if newQuality >= model.SpiritQualityChaos {
			var player model.Player
			if err := s.db.WithContext(ctx).First(&player, playerID).Error; err == nil {
				s.tryUnlockElementAwakening(ctx, playerID, &player, now)
			}
		}
	} else {
		spirit.Reincarnations++
		spirit.UpdatedAt = now
		s.db.WithContext(ctx).Save(&spirit)
	}

	return spirit, upgraded, nil
}

// GetElementDamageBonus 获取元素伤害加成（供战斗系统调用）
func (s *SpiritEvolutionService) GetElementDamageBonus(ctx context.Context, playerID int64) float64 {
	awakening, err := s.GetElementAwakening(ctx, playerID)
	if err != nil || awakening == nil {
		return 0
	}
	return awakening.DamageBonus
}
