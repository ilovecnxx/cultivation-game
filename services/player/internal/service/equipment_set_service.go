package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// EquipmentSetService 装备套装/附魔/觉醒业务逻辑
type EquipmentSetService struct {
	repo EquipSetRepository
	log  *zap.Logger
}

// NewEquipmentSetService 创建 EquipmentSetService
func NewEquipmentSetService(repo EquipSetRepository, log *zap.Logger) *EquipmentSetService {
	return &EquipmentSetService{repo: repo, log: log}
}

// ============================================================
// 套装检测
// ============================================================

// GetActiveSetBonuses 获取所有已激活的套装效果
func (s *EquipmentSetService) GetActiveSetBonuses(ctx context.Context, playerID int64) ([]*model.ActiveSetBonus, error) {
	equipments, err := s.repo.GetEquipmentByPlayer(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取玩家装备失败: %w", err)
	}

	// 构建已装备item ID -> equipment映射
	equippedItemIDs := make(map[int64]bool)
	for _, eq := range equipments {
		if eq.Item != nil {
			equippedItemIDs[eq.Item.ID] = true
		}
	}

	var activeBonuses []*model.ActiveSetBonus

	for _, set := range model.EquipmentSetList {
		matchedPieces := 0
		for _, pieceID := range set.Pieces {
			if equippedItemIDs[pieceID] {
				matchedPieces++
			}
		}

		if matchedPieces == 0 {
			continue
		}

		activeBonus := &model.ActiveSetBonus{
			SetID:          set.ID,
			SetName:        set.Name,
			SetQuality:     set.Quality,
			SetElement:     set.Element,
			SetIcon:        set.Icon,
			PiecesEquipped: matchedPieces,
			PiecesTotal:    len(set.Pieces),
			IsActive:       false,
		}

		// 找出所有满足条件的套装效果
		for _, bonus := range set.Bonuses {
			if matchedPieces >= bonus.PiecesRequired {
				activeBonus.Bonuses = append(activeBonus.Bonuses, bonus)
			}
		}

		// 如果集齐了至少2件，有激活的效果
		if matchedPieces >= 2 {
			activeBonus.IsActive = true
		}

		activeBonuses = append(activeBonuses, activeBonus)
	}

	if activeBonuses == nil {
		activeBonuses = []*model.ActiveSetBonus{}
	}

	return activeBonuses, nil
}

// CalculateSetStats 计算套装提供的战斗属性加成
func (s *EquipmentSetService) CalculateSetStats(ctx context.Context, playerID int64) (*model.CombatStats, error) {
	activeBonuses, err := s.GetActiveSetBonuses(ctx, playerID)
	if err != nil {
		return nil, err
	}

	stats := &model.CombatStats{}

	// 检查是否激活了鸿蒙套装5件套效果
	primordialDoubled := false
	for _, bonus := range activeBonuses {
		if bonus.SetID == "primordial" && bonus.PiecesEquipped >= 5 {
			primordialDoubled = true
		}
	}

	multiplier := 1.0
	if primordialDoubled {
		multiplier = 2.0
	}

	for _, active := range activeBonuses {
		for _, bonus := range active.Bonuses {
			for _, effect := range bonus.Effects {
				val := effect.Value * multiplier

				switch effect.Stat {
				case "all_stats":
					stats.Attack += val
					stats.Defense += val
					stats.HP += val
					stats.Speed += val
					stats.CritRate += val
					stats.CritDmg += val
					stats.DodgeRate += val
				case "attack":
					stats.Attack += val
				case "defense":
					stats.Defense += val
				case "hp":
					stats.HP += val
				case "speed":
					stats.Speed += val
				case "crit_rate":
					stats.CritRate += val
				case "crit_dmg":
					stats.CritDmg += val
				case "dodge_rate":
					stats.DodgeRate += val
				case "lifesteal":
					stats.Lifesteal += val
				case "armor_pen":
					stats.ArmorPen += val
				case "damage_bonus":
					stats.DamageBonus += val
				case "damage_reduction":
					stats.DamageReduction += val
				case "fire_dmg":
					stats.FireDmg += val
				case "breakthrough":
					stats.Breakthrough += val
				case "luck":
					stats.Luck += val
				case "exp_bonus":
					stats.ExpBonus += val
				case "extra_actions":
					stats.ExtraActions += int(val)
				case "reflect_dmg":
					stats.ReflectDmg += val
				case "hp_regen":
					stats.HpRegen += val
				}
			}
		}
	}

	return stats, nil
}

// GetMissingPieces 获取指定套装缺少的部件及获取提示
func (s *EquipmentSetService) GetMissingPieces(ctx context.Context, playerID int64, setName string) (*model.SetProgress, error) {
	// 查找套装定义
	var targetSet *model.EquipmentSet
	for i := range model.EquipmentSetList {
		if model.EquipmentSetList[i].ID == setName {
			targetSet = &model.EquipmentSetList[i]
			break
		}
	}
	if targetSet == nil {
		return nil, fmt.Errorf("未找到套装: %s", setName)
	}

	// 获取玩家装备和背包
	equipments, err := s.repo.GetEquipmentByPlayer(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取装备失败: %w", err)
	}

	inventory, err := s.repo.GetInventoryByPlayer(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取背包失败: %w", err)
	}

	// 收集玩家拥有的所有item IDs
	ownedItemIDs := make(map[int64]bool)
	for _, eq := range equipments {
		if eq.Item != nil {
			ownedItemIDs[eq.Item.ID] = true
		}
	}
	for _, inv := range inventory {
		ownedItemIDs[inv.ItemID] = true
	}

	// 加载所有set pieces的物品信息
	allPieceIDs := targetSet.Pieces
	itemMap, err := s.repo.ListItems(allPieceIDs)
	if err != nil {
		s.log.Warn("加载套装物品信息失败", zap.Error(err))
	}

	progress := &model.SetProgress{
		SetID:      targetSet.ID,
		SetName:    targetSet.Name,
		SetElement: targetSet.Element,
		SetIcon:    targetSet.Icon,
		TotalPieces: len(targetSet.Pieces),
	}

	for _, pieceID := range targetSet.Pieces {
		if ownedItemIDs[pieceID] {
			progress.Equipped = append(progress.Equipped, pieceID)
			if item, ok := itemMap[pieceID]; ok {
				progress.EquippedNames = append(progress.EquippedNames, item.Name)
			} else {
				progress.EquippedNames = append(progress.EquippedNames, fmt.Sprintf("物品#%d", pieceID))
			}
		} else {
			progress.Missing = append(progress.Missing, pieceID)
			if item, ok := itemMap[pieceID]; ok {
				progress.MissingNames = append(progress.MissingNames, item.Name)
			} else {
				progress.MissingNames = append(progress.MissingNames, fmt.Sprintf("物品#%d", pieceID))
			}
		}
	}

	progress.PiecesCount = len(progress.Equipped)

	// 获取提示
	if hints, ok := model.SetAcquisitionHints[targetSet.ID]; ok {
		progress.MissingHints = hints
	} else {
		progress.MissingHints = []string{}
	}

	return progress, nil
}

// GetAllSetProgress 获取所有套装的收集进度
func (s *EquipmentSetService) GetAllSetProgress(ctx context.Context, playerID int64) ([]*model.SetProgress, error) {
	var allProgress []*model.SetProgress
	for _, set := range model.EquipmentSetList {
		progress, err := s.GetMissingPieces(ctx, playerID, set.ID)
		if err != nil {
			s.log.Warn("获取套装进度失败", zap.String("set", set.ID), zap.Error(err))
			continue
		}
		allProgress = append(allProgress, progress)
	}
	return allProgress, nil
}

// ============================================================
// 附魔系统
// ============================================================

// GetAvailableEnchantments 获取装备位可用的附魔列表
func (s *EquipmentSetService) GetAvailableEnchantments(slot int32) []model.EnchantmentDef {
	return model.GetEnchantmentsForSlot(slot)
}

// GetAllEnchantmentGroups 获取所有附魔分组
func (s *EquipmentSetService) GetAllEnchantmentGroups() []model.EnchantSlotGroup {
	return model.EnchantmentSlotGroups
}

// ApplyEnchantment 为装备附魔
func (s *EquipmentSetService) ApplyEnchantment(ctx context.Context, playerID int64, req *model.EnchantRequest) (*model.EnchantmentInstance, error) {
	// 验证装备
	equipment, err := s.repo.GetEquipmentByID(req.EquipmentID)
	if err != nil {
		return nil, fmt.Errorf("获取装备失败: %w", err)
	}
	if equipment == nil {
		return nil, fmt.Errorf("装备不存在")
	}
	if equipment.PlayerID != playerID {
		return nil, fmt.Errorf("装备不属于该玩家")
	}

	// 检查觉醒等级（决定最大附魔槽位）
	awakenLevel := 0
	awakening, err := s.repo.GetEquipmentAwakening(req.EquipmentID)
	if err == nil && awakening != nil {
		awakenLevel = awakening.AwakenLevel
	}

	maxEnchants := 3 + awakenLevel // 基础3个，每次觉醒+1
	_ = maxEnchants

	// 获取已有附魔
	existingEnchants, err := s.repo.GetEquipmentEnchantments(req.EquipmentID)
	if err != nil {
		return nil, fmt.Errorf("获取已有附魔失败: %w", err)
	}

	// 检查是否有空余槽位
	if len(existingEnchants) >= 3 {
		return nil, fmt.Errorf("附魔槽位已满（最大3个），可通过装备觉醒增加槽位")
	}

	// 检查是否已有相同类型的附魔
	for _, e := range existingEnchants {
		if e.EnchantID == req.EnchantID {
			return nil, fmt.Errorf("该装备已附魔相同效果，不能重复附魔")
		}
	}

	// 获取附魔定义
	enchantDef := model.GetEnchantmentDef(req.EnchantID)
	if enchantDef == nil {
		return nil, fmt.Errorf("未知的附魔类型: %s", req.EnchantID)
	}

	// 验证附魔是否适用于该装备位
	validSlot := false
	for _, slot := range enchantDef.TargetSlots {
		if slot == equipment.Slot {
			validSlot = true
			break
		}
	}
	if !validSlot {
		return nil, fmt.Errorf("该附魔不能用于此装备位")
	}

	// 计算成功率（随已有附魔数量递减）
	successRate := enchantDef.SuccessRate
	for i := 0; i < len(existingEnchants); i++ {
		successRate *= 0.8 // 每个已有附魔降低20%
	}

	// 模拟成功率判定
	if rand.Float64() > successRate {
		// 失败：材料扣除但装备不销毁
		return nil, fmt.Errorf("附魔失败，材料已消耗，装备完好无损（成功率: %.1f%%）", successRate*100)
	}

	// 创建附魔记录
	slotIndex := len(existingEnchants)
	enchant := &model.PlayerEnchantment{
		PlayerID:    playerID,
		EquipmentID: req.EquipmentID,
		EnchantID:   req.EnchantID,
		Level:       1,
		SlotIndex:   slotIndex,
	}

	if err := s.repo.SaveEnchantment(enchant); err != nil {
		return nil, fmt.Errorf("保存附魔失败: %w", err)
	}

	// 构建返回结果
	instance := &model.EnchantmentInstance{
		ID:          enchant.ID,
		EnchantID:   enchant.EnchantID,
		Name:        enchantDef.Name,
		Level:       enchant.Level,
		SlotIndex:   enchant.SlotIndex,
		Description: enchantDef.Description,
		Icon:        enchantDef.Icon,
	}
	for _, stat := range enchantDef.Stats {
		instance.Stats = append(instance.Stats, model.StatBonus{
			Stat:  stat.Stat,
			Value: stat.Value + stat.PerLevel*float64(enchant.Level-1),
		})
	}

	return instance, nil
}

// RemoveEnchantment 移除装备附魔
func (s *EquipmentSetService) RemoveEnchantment(ctx context.Context, playerID int64, req *model.EnchantRemoveRequest) error {
	// 获取装备
	equipment, err := s.repo.GetEquipmentByID(req.EquipmentID)
	if err != nil {
		return fmt.Errorf("获取装备失败: %w", err)
	}
	if equipment == nil {
		return fmt.Errorf("装备不存在")
	}
	if equipment.PlayerID != playerID {
		return fmt.Errorf("装备不属于该玩家")
	}

	// 获取该装备的附魔
	existingEnchants, err := s.repo.GetEquipmentEnchantments(req.EquipmentID)
	if err != nil {
		return fmt.Errorf("获取附魔失败: %w", err)
	}

	// 找到指定槽位的附魔
	var targetEnchant *model.PlayerEnchantment
	for _, e := range existingEnchants {
		if e.SlotIndex == req.EnchantSlot {
			targetEnchant = e
			break
		}
	}
	if targetEnchant == nil {
		return fmt.Errorf("该槽位无附魔")
	}

	// 移除附魔
	if err := s.repo.RemoveEnchantment(targetEnchant.ID); err != nil {
		return fmt.Errorf("移除附魔失败: %w", err)
	}

	return nil
}

// GetEquipmentEnchantments 获取装备所有附魔
func (s *EquipmentSetService) GetEquipmentEnchantments(ctx context.Context, equipmentID int64) ([]*model.EnchantmentInstance, error) {
	enchants, err := s.repo.GetEquipmentEnchantments(equipmentID)
	if err != nil {
		return nil, err
	}

	var instances []*model.EnchantmentInstance
	for _, e := range enchants {
		def := model.GetEnchantmentDef(e.EnchantID)
		if def == nil {
			continue
		}
		instance := &model.EnchantmentInstance{
			ID:          e.ID,
			EnchantID:   e.EnchantID,
			Name:        def.Name,
			Level:       e.Level,
			SlotIndex:   e.SlotIndex,
			Description: def.Description,
			Icon:        def.Icon,
		}
		for _, stat := range def.Stats {
			instance.Stats = append(instance.Stats, model.StatBonus{
				Stat:  stat.Stat,
				Value: stat.Value + stat.PerLevel*float64(e.Level-1),
			})
		}
		instances = append(instances, instance)
	}

	if instances == nil {
		instances = []*model.EnchantmentInstance{}
	}

	return instances, nil
}

// ============================================================
// 装备觉醒
// ============================================================

// GetAwakeningInfo 获取装备觉醒信息
func (s *EquipmentSetService) GetAwakeningInfo(ctx context.Context, equipmentID int64) (*model.EquipmentAwakening, error) {
	awakening, err := s.repo.GetEquipmentAwakening(equipmentID)
	if err != nil {
		return nil, nil // 没有觉醒记录不算错误
	}
	return awakening, nil
}

// CanAwaken 检查装备能否觉醒
func (s *EquipmentSetService) CanAwaken(ctx context.Context, playerID int64, equipmentID int64) (bool, string, *model.AwakenCost) {
	equipment, err := s.repo.GetEquipmentByID(equipmentID)
	if err != nil || equipment == nil {
		return false, "装备不存在", nil
	}
	if equipment.PlayerID != playerID {
		return false, "装备不属于该玩家", nil
	}

	// 获取当前觉醒等级
	awakenLevel := 0
	awakening, err := s.repo.GetEquipmentAwakening(equipmentID)
	if err == nil && awakening != nil {
		awakenLevel = awakening.AwakenLevel
	}

	if awakenLevel >= 3 {
		return false, "已达到最大觉醒次数（3次）", nil
	}

	// 检查装备等级是否达到最大
	currentMaxLevel := int32(20 + awakenLevel*20)
	if equipment.Level < currentMaxLevel {
		return false, fmt.Sprintf("装备等级不足，需要%d级（当前%d级）", currentMaxLevel, equipment.Level), nil
	}

	cost := model.GetAwakenCost(awakenLevel + 1)
	return true, "", &cost
}

// AwakenEquipment 执行装备觉醒
func (s *EquipmentSetService) AwakenEquipment(ctx context.Context, playerID int64, req *model.AwakenRequest) (*model.EquipmentAwakening, error) {
	canAwaken, msg, cost := s.CanAwaken(ctx, playerID, req.EquipmentID)
	if !canAwaken {
		return nil, fmt.Errorf("%s", msg)
	}

	equipment, err := s.repo.GetEquipmentByID(req.EquipmentID)
	if err != nil {
		return nil, fmt.Errorf("获取装备失败: %w", err)
	}

	awakenLevel := 0
	awakening, err := s.repo.GetEquipmentAwakening(req.EquipmentID)
	if err == nil && awakening != nil {
		awakenLevel = awakening.AwakenLevel
	}

	newLevel := awakenLevel + 1

	// 重置装备等级为1，新的最大等级为 20 + newLevel*20
	newMaxLevel := int32(20 + newLevel*20)

	if err := s.repo.UpdateEquipmentLevel(equipment.ID, 1); err != nil {
		return nil, fmt.Errorf("重置装备等级失败: %w", err)
	}
	if err := s.repo.UpdateEquipmentMaxLevel(equipment.ID, newMaxLevel); err != nil {
		return nil, fmt.Errorf("更新最大等级失败: %w", err)
	}

	if awakening == nil {
		awakening = &model.EquipmentAwakening{
			PlayerID:    playerID,
			EquipmentID: req.EquipmentID,
			AwakenLevel: newLevel,
		}
		if err := s.repo.SaveAwakening(awakening); err != nil {
			return nil, fmt.Errorf("保存觉醒记录失败: %w", err)
		}
	} else {
		awakening.AwakenLevel = newLevel
		if err := s.repo.UpdateAwakening(awakening); err != nil {
			return nil, fmt.Errorf("更新觉醒记录失败: %w", err)
		}
	}

	s.log.Info("装备觉醒成功",
		zap.Int64("player_id", playerID),
		zap.Int64("equipment_id", req.EquipmentID),
		zap.Int("awaken_level", newLevel),
		zap.Int64("cost", cost.SpiritStones),
	)

	return awakening, nil
}

// ============================================================
// 装备详情
// ============================================================

// GetEquipmentDetail 获取装备详细信息（含套装、附魔、觉醒）
func (s *EquipmentSetService) GetEquipmentDetail(ctx context.Context, playerID int64, equipmentID int64) (*model.EquipmentDetail, error) {
	equipment, err := s.repo.GetEquipmentByID(equipmentID)
	if err != nil {
		return nil, fmt.Errorf("获取装备失败: %w", err)
	}
	if equipment == nil {
		return nil, fmt.Errorf("装备不存在")
	}

	detail := &model.EquipmentDetail{
		Equipment: equipment,
	}

	// 获取套装信息
	activeBonuses, err := s.GetActiveSetBonuses(ctx, playerID)
	if err == nil && equipment.Item != nil {
		// 找出该装备所属的套装
		for _, bonus := range activeBonuses {
			for _, set := range model.EquipmentSetList {
				if set.ID == bonus.SetID {
					for _, pieceID := range set.Pieces {
						if pieceID == equipment.ItemID {
							detail.SetInfo = bonus
							break
						}
					}
					if detail.SetInfo != nil {
						break
					}
				}
			}
			if detail.SetInfo != nil {
				break
			}
		}
	}

	// 获取附魔信息
	enchants, err := s.GetEquipmentEnchantments(ctx, equipmentID)
	if err == nil {
		detail.Enchantments = enchants
	}

	// 获取觉醒信息
	awakening, err := s.GetAwakeningInfo(ctx, equipmentID)
	if err == nil {
		detail.Awakening = awakening
	}

	// 计算总属性
	stats, err := s.CalculateSetStats(ctx, playerID)
	if err == nil {
		detail.TotalStats = stats
	}

	return detail, nil
}

// ============================================================
// 辅助函数
// ============================================================

// GetSetByID 根据ID获取套装定义
func (s *EquipmentSetService) GetSetByID(id string) *model.EquipmentSet {
	for _, set := range model.EquipmentSetList {
		if set.ID == id {
			return &set
		}
	}
	return nil
}

// CalculateEnchantSuccessRate 计算实际附魔成功率
func (s *EquipmentSetService) CalculateEnchantSuccessRate(baseRate float64, existingCount int) float64 {
	rate := baseRate
	for i := 0; i < existingCount; i++ {
		rate *= 0.8
	}
	return math.Max(rate, 0.05) // 最低5%
}
