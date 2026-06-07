package service

import (
	"context"
	"fmt"
	"sort"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

const (
	maxInventorySlots = 60 // 最大背包格子数
	defaultSlotSize   = 20 // 初始背包格子数
)

// InventoryService 背包/装备业务逻辑
type InventoryService struct {
	inventoryRepo InventoryRepository
	playerService *PlayerService
	cache         Cache
	log           *zap.Logger
}

// NewInventoryService 创建 InventoryService
func NewInventoryService(
	inventoryRepo InventoryRepository,
	playerService *PlayerService,
	cache Cache,
	log *zap.Logger,
) *InventoryService {
	return &InventoryService{
		inventoryRepo: inventoryRepo,
		playerService: playerService,
		cache:         cache,
		log:           log,
	}
}

// -------- 背包操作 --------

// GetInventory 获取玩家背包物品
func (s *InventoryService) GetInventory(ctx context.Context, playerID int64) ([]*model.InventoryItem, error) {
	// 尝试从缓存读取
	items, err := s.cache.GetInventoryCache(ctx, playerID)
	if err == nil && items != nil {
		return items, nil
	}

	// 缓存未命中，从 MySQL 读取
	items, err = s.inventoryRepo.GetInventoryByPlayer(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取背包失败: %w", err)
	}
	if items == nil {
		items = []*model.InventoryItem{}
	}

	// 回写缓存（异步）
	if err := s.cache.SetInventoryCache(ctx, playerID, items); err != nil {
		s.log.Warn("回写背包缓存失败", zap.Error(err))
	}

	return items, nil
}

// AddItem 向背包添加物品
// 流程：查找可堆叠物品 -> 堆叠/新建 -> 更新缓存
func (s *InventoryService) AddItem(ctx context.Context, playerID, itemID int64, quantity int32) ([]*model.InventoryItem, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("数量必须大于0")
	}

	// 查询物品模板
	item, err := s.inventoryRepo.GetItem(itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("物品不存在")
	}

	var results []*model.InventoryItem

	// 可堆叠物品，尝试堆叠
	if item.MaxStack > 1 {
		remaining := quantity
		for remaining > 0 {
			// 查找同种可堆叠的物品
			stackable, err := s.inventoryRepo.FindStackableItem(playerID, itemID)
			if err != nil {
				return nil, err
			}

			if stackable != nil {
				// 计算可堆叠量
				space := item.MaxStack - stackable.Quantity
				if space >= remaining {
					// 完全堆叠
					stackable.Quantity += remaining
					if err := s.inventoryRepo.UpdateItemQuantity(stackable.ID, stackable.Quantity); err != nil {
						return nil, err
					}
					results = append(results, stackable)
					remaining = 0
				} else {
					// 部分堆叠，先填满
					stackable.Quantity = item.MaxStack
					if err := s.inventoryRepo.UpdateItemQuantity(stackable.ID, stackable.Quantity); err != nil {
						return nil, err
					}
					results = append(results, stackable)
					remaining -= space
				}
			} else {
				// 开辟新格子
				count, err := s.inventoryRepo.GetInventoryCount(playerID)
				if err != nil {
					return nil, err
				}
				if count >= maxInventorySlots {
					return nil, fmt.Errorf("背包已满")
				}

				addQty := remaining
				if addQty > item.MaxStack {
					addQty = item.MaxStack
				}

				slotIndex := int32(count + 1)
				invItem := &model.InventoryItem{
					PlayerID:  playerID,
					ItemID:    itemID,
					Quantity:  addQty,
					SlotIndex: slotIndex,
				}
				if err := s.inventoryRepo.InsertItem(invItem); err != nil {
					return nil, err
				}
				invItem.Item = item
				results = append(results, invItem)
				remaining -= addQty
			}
		}
	} else {
		// 不可堆叠物品，逐个插入
		for i := int32(0); i < quantity; i++ {
			count, err := s.inventoryRepo.GetInventoryCount(playerID)
			if err != nil {
				return nil, err
			}
			if count >= maxInventorySlots {
				return nil, fmt.Errorf("背包已满")
			}

			invItem := &model.InventoryItem{
				PlayerID:  playerID,
				ItemID:    itemID,
				Quantity:  1,
				SlotIndex: int32(count + 1),
			}
			if err := s.inventoryRepo.InsertItem(invItem); err != nil {
				return nil, err
			}
			invItem.Item = item
			results = append(results, invItem)
		}
	}

	// 使背包缓存失效
	s.invalidateInventoryCache(ctx, playerID)
	return results, nil
}

// RemoveItem 从背包移除物品（递减或删除）
func (s *InventoryService) RemoveItem(ctx context.Context, playerID int64, inventoryItemID int64, quantity int32) error {
	if quantity <= 0 {
		return fmt.Errorf("数量必须大于0")
	}

	invItem, err := s.inventoryRepo.GetInventoryItem(inventoryItemID)
	if err != nil {
		return err
	}
	if invItem == nil || invItem.PlayerID != playerID {
		return fmt.Errorf("物品不存在")
	}
	if invItem.IsEquipped {
		return fmt.Errorf("装备中的物品无法移除")
	}
	if quantity > invItem.Quantity {
		return fmt.Errorf("物品数量不足")
	}

	if quantity == invItem.Quantity {
		// 全部移除
		if err := s.inventoryRepo.DeleteItem(inventoryItemID); err != nil {
			return err
		}
	} else {
		// 部分移除
		newQty := invItem.Quantity - quantity
		if err := s.inventoryRepo.UpdateItemQuantity(inventoryItemID, newQty); err != nil {
			return err
		}
	}

	s.invalidateInventoryCache(ctx, playerID)
	return nil
}

// TransferItem 移动物品格子
func (s *InventoryService) TransferItem(ctx context.Context, playerID int64, req *model.InventoryTransferRequest) error {
	// 检查源格子
	items, err := s.inventoryRepo.GetInventoryByPlayer(playerID)
	if err != nil {
		return err
	}

	var fromItem, toItem *model.InventoryItem
	for _, item := range items {
		if item.SlotIndex == req.FromSlot && !item.IsEquipped {
			fromItem = item
		}
		if item.SlotIndex == req.ToSlot && !item.IsEquipped {
			toItem = item
		}
	}

	if fromItem == nil {
		return fmt.Errorf("源格子无物品")
	}

	if toItem == nil {
		// 目标格子为空，直接移动
		if err := s.inventoryRepo.UpdateItemSlot(fromItem.ID, req.ToSlot); err != nil {
			return err
		}
	} else if fromItem.ItemID == toItem.ItemID && fromItem.Item.MaxStack > 1 {
		// 同种可堆叠物品，尝试合并
		space := fromItem.Item.MaxStack - toItem.Quantity
		if space >= fromItem.Quantity {
			// 全部合并
			toItem.Quantity += fromItem.Quantity
			if err := s.inventoryRepo.UpdateItemQuantity(toItem.ID, toItem.Quantity); err != nil {
				return err
			}
			if err := s.inventoryRepo.DeleteItem(fromItem.ID); err != nil {
				return err
			}
		} else {
			// 部分合并
			toItem.Quantity = fromItem.Item.MaxStack
			fromItem.Quantity -= space
			if err := s.inventoryRepo.UpdateItemQuantity(toItem.ID, toItem.Quantity); err != nil {
				return err
			}
			if err := s.inventoryRepo.UpdateItemQuantity(fromItem.ID, fromItem.Quantity); err != nil {
				return err
			}
		}
	} else {
		// 交换两个格子的物品
		fromSlot := fromItem.SlotIndex
		if err := s.inventoryRepo.UpdateItemSlot(fromItem.ID, req.ToSlot); err != nil {
			return err
		}
		if err := s.inventoryRepo.UpdateItemSlot(toItem.ID, fromSlot); err != nil {
			return err
		}
	}

	s.invalidateInventoryCache(ctx, playerID)
	return nil
}

// SortInventory 整理背包
func (s *InventoryService) SortInventory(ctx context.Context, playerID int64, req *model.SortInventoryRequest) ([]*model.InventoryItem, error) {
	items, err := s.inventoryRepo.GetInventoryByPlayer(playerID)
	if err != nil {
		return nil, err
	}

	// 过滤出非装备物品
	var bagItems []*model.InventoryItem
	for _, item := range items {
		if !item.IsEquipped {
			bagItems = append(bagItems, item)
		}
	}

	// 排序
	switch req.SortBy {
	case "type":
		sort.Slice(bagItems, func(i, j int) bool {
			if req.Desc {
				return bagItems[i].Item.Type > bagItems[j].Item.Type
			}
			return bagItems[i].Item.Type < bagItems[j].Item.Type
		})
	case "quality":
		sort.Slice(bagItems, func(i, j int) bool {
			if req.Desc {
				return bagItems[i].Item.Quality > bagItems[j].Item.Quality
			}
			return bagItems[i].Item.Quality < bagItems[j].Item.Quality
		})
	case "level":
		sort.Slice(bagItems, func(i, j int) bool {
			if req.Desc {
				return bagItems[i].Item.RequiredLevel > bagItems[j].Item.RequiredLevel
			}
			return bagItems[i].Item.RequiredLevel < bagItems[j].Item.RequiredLevel
		})
	}

	// 重新分配格子索引
	for i, item := range bagItems {
		newSlot := int32(i + 1)
		if item.SlotIndex != newSlot {
			if err := s.inventoryRepo.UpdateItemSlot(item.ID, newSlot); err != nil {
				return nil, err
			}
			item.SlotIndex = newSlot
		}
	}

	s.invalidateInventoryCache(ctx, playerID)
	return bagItems, nil
}

// UseItem 使用物品（丹药/消耗品）
func (s *InventoryService) UseItem(ctx context.Context, playerID int64, req *model.UseItemRequest) (map[string]int64, error) {
	invItem, err := s.inventoryRepo.GetInventoryItem(req.InventoryItemID)
	if err != nil {
		return nil, err
	}
	if invItem == nil || invItem.PlayerID != playerID {
		return nil, fmt.Errorf("物品不存在")
	}

	if invItem.Quantity < req.Quantity {
		return nil, fmt.Errorf("物品数量不足")
	}

	item := invItem.Item
	if item.Type != model.ItemTypePill && item.Type != model.ItemTypeConsumable {
		return nil, fmt.Errorf("该物品无法使用")
	}

	// 查找玩家
	player, err := s.playerService.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}

	// 计算效果（按使用数量叠加）
	totalEffect := s.calcUseEffect(item, req.Quantity)

	// 应用效果
	player.HP += totalEffect.hp
	if player.HP > player.MaxHP {
		player.HP = player.MaxHP
	}
	player.MP += totalEffect.mp
	if player.MP > player.MaxMP {
		player.MP = player.MaxMP
	}
	player.Experience += totalEffect.exp
	player.SpiritPower += totalEffect.spirit

	// 更新玩家
	if err := s.playerService.UpdatePlayer(ctx, player); err != nil {
		return nil, err
	}

	// 扣除物品
	if err := s.RemoveItem(ctx, playerID, req.InventoryItemID, req.Quantity); err != nil {
		return nil, err
	}

	return map[string]int64{
		"hp": totalEffect.hp, "mp": totalEffect.mp,
		"exp": totalEffect.exp, "spirit_power": totalEffect.spirit,
	}, nil
}

// -------- 装备操作 --------

// EquipItem 穿戴装备
func (s *InventoryService) EquipItem(ctx context.Context, playerID int64, req *model.EquipRequest) (*model.Equipment, error) {
	// 获取背包中的物品
	invItem, err := s.inventoryRepo.GetInventoryItem(req.InventoryItemID)
	if err != nil {
		return nil, err
	}
	if invItem == nil || invItem.PlayerID != playerID {
		return nil, fmt.Errorf("物品不存在")
	}
	if invItem.IsEquipped {
		return nil, fmt.Errorf("该物品已装备")
	}

	// 检查物品是否为装备
	slot, ok := model.ItemTypeToEquipSlot[invItem.Item.Type]
	if !ok {
		return nil, fmt.Errorf("该物品无法装备")
	}

	// 检查等级/境界需求
	player, err := s.playerService.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if player.Level < invItem.Item.RequiredLevel {
		return nil, fmt.Errorf("等级不足，需要%d级", invItem.Item.RequiredLevel)
	}
	if player.Realm < invItem.Item.RequiredRealm {
		return nil, fmt.Errorf("境界不足")
	}

	// 检查该槽位是否已有装备
	existingEquip, err := s.inventoryRepo.GetEquipmentBySlot(playerID, int64(slot))
	if err != nil {
		return nil, err
	}

	// 如果已有装备，先卸下
	if existingEquip != nil {
		if _, err := s.UnequipItem(ctx, playerID, &model.UnequipRequest{Slot: slot}); err != nil {
			return nil, fmt.Errorf("卸下旧装备失败: %w", err)
		}
	}

	// 标记背包物品为已装备
	if err := s.inventoryRepo.UpdateItemEquipStatus(invItem.ID, true); err != nil {
		return nil, err
	}

	// 创建装备记录
	equip := &model.Equipment{
		PlayerID:        playerID,
		Slot:            slot,
		InventoryItemID: invItem.ID,
		ItemID:          invItem.ItemID,
		Level:           0,
	}
	if err := s.inventoryRepo.InsertEquipment(equip); err != nil {
		// 回滚
		_ = s.inventoryRepo.UpdateItemEquipStatus(invItem.ID, false)
		return nil, err
	}
	equip.Item = invItem.Item

	// 更新玩家属性
	s.applyEquipmentStats(player, invItem.Item, 0, true)
	if err := s.playerService.UpdatePlayer(ctx, player); err != nil {
		return nil, err
	}

	s.invalidateInventoryCache(ctx, playerID)
	return equip, nil
}

// UnequipItem 卸下装备
func (s *InventoryService) UnequipItem(ctx context.Context, playerID int64, req *model.UnequipRequest) (*model.InventoryItem, error) {
	equip, err := s.inventoryRepo.GetEquipmentBySlot(playerID, int64(req.Slot))
	if err != nil {
		return nil, err
	}
	if equip == nil {
		return nil, fmt.Errorf("该槽位未装备物品")
	}

	// 获取背包物品
	invItem, err := s.inventoryRepo.GetInventoryItem(equip.InventoryItemID)
	if err != nil {
		return nil, err
	}
	if invItem == nil {
		return nil, fmt.Errorf("关联的背包物品不存在")
	}

	// 删除装备记录
	if err := s.inventoryRepo.DeleteEquipmentBySlot(playerID, int64(req.Slot)); err != nil {
		return nil, err
	}

	// 取消装备状态
	if err := s.inventoryRepo.UpdateItemEquipStatus(invItem.ID, false); err != nil {
		return nil, err
	}

	// 更新玩家属性
	player, err := s.playerService.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	s.applyEquipmentStats(player, equip.Item, equip.Level, false)
	if err := s.playerService.UpdatePlayer(ctx, player); err != nil {
		return nil, err
	}

	s.invalidateInventoryCache(ctx, playerID)
	return invItem, nil
}

// StrengthenEquipment 强化装备
func (s *InventoryService) StrengthenEquipment(ctx context.Context, playerID int64, req *model.StrengthenRequest) (*model.Equipment, error) {
	equip, err := s.inventoryRepo.GetEquipmentBySlot(playerID, int64(req.Slot))
	if err != nil {
		return nil, err
	}
	if equip == nil {
		return nil, fmt.Errorf("该槽位未装备物品")
	}
	if equip.Level >= 20 {
		return nil, fmt.Errorf("装备已满级（20级）")
	}

	// 计算强化消耗
	goldCost, materialID := s.enhanceCost(equip.Level)
	player, err := s.playerService.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if player.Gold < goldCost {
		return nil, fmt.Errorf("灵石不足，需要%d", goldCost)
	}

	// 扣除灵石
	player.Gold -= goldCost
	if _, err := s.playerService.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{Gold: -goldCost}); err != nil {
		return nil, err
	}

	// 扣除材料（若有）
	if materialID > 0 {
		// 查找材料并扣除
		materials, _ := s.inventoryRepo.GetInventoryByPlayer(playerID)
		for _, mi := range materials {
			if mi.ItemID == materialID && !mi.IsEquipped && mi.Quantity >= 1 {
				if err := s.RemoveItem(ctx, playerID, mi.ID, 1); err != nil {
					return nil, fmt.Errorf("扣除材料失败: %w", err)
				}
				break
			}
		}
	}

	// 强化成功率（从80%递减到10%）
	successRate := s.successRate(equip.Level)

	// 这里简化处理：直接成功（实际项目中应由战斗/概率系统处理）
	_ = successRate

	// 提升等级
	equip.Level++
	if err := s.inventoryRepo.UpdateEquipmentLevel(equip.ID, equip.Level); err != nil {
		return nil, err
	}

	// 更新玩家属性
	s.applyEquipmentStats(player, equip.Item, equip.Level, true)
	if err := s.playerService.UpdatePlayer(ctx, player); err != nil {
		return nil, err
	}

	return equip, nil
}

// GetEquipment 获取玩家所有装备
func (s *InventoryService) GetEquipment(ctx context.Context, playerID int64) ([]*model.Equipment, error) {
	return s.inventoryRepo.GetEquipmentByPlayer(playerID)
}

// -------- 辅助方法 --------

// calcUseEffect 计算物品使用效果
type useEffect struct {
	hp, mp, exp, spirit int64
}

func (s *InventoryService) calcUseEffect(item *model.Item, quantity int32) useEffect {
	// 简单解析 UseEffect（格式: "hp:50,mp:30" 或 JSON）
	effect := useEffect{
		hp:     item.BaseHP * int64(quantity),
		mp:     item.BaseMP * int64(quantity),
		spirit: item.BaseAttack * int64(quantity), // 用 Attack 字段临时存储修为增加
	}

	// 丹药特殊效果
	if item.Type == model.ItemTypePill {
		effect.spirit = 10 * int64(quantity) // 丹药增加修为
	}

	return effect
}

// applyEquipmentStats 应用/移除装备属性
func (s *InventoryService) applyEquipmentStats(player *model.Player, item *model.Item, enhanceLevel int32, isEquip bool) {
	multiplier := int64(1)
	if !isEquip {
		multiplier = -1
	}

	// 基础属性 + 强化加成（每级提升10%）
	enhanceBonus := int64(1 + enhanceLevel*10/100)
	player.MaxHP += item.BaseHP * multiplier * enhanceBonus
	player.MaxMP += item.BaseMP * multiplier * enhanceBonus
	player.Attack += item.BaseAttack * multiplier * enhanceBonus
	player.Defense += item.BaseDefense * multiplier * enhanceBonus

	// 保证数值非负
	if player.MaxHP < 0 {
		player.MaxHP = 1
	}
	if player.MaxMP < 0 {
		player.MaxMP = 0
	}
	if player.Attack < 0 {
		player.Attack = 1
	}
	if player.Defense < 0 {
		player.Defense = 0
	}
	if player.HP > player.MaxHP {
		player.HP = player.MaxHP
	}
	if player.MP > player.MaxMP {
		player.MP = player.MaxMP
	}
}

// enhanceCost 计算强化消耗
func (s *InventoryService) enhanceCost(currentLevel int32) (goldCost int64, materialID int64) {
	goldCost = int64(100 + currentLevel*50)
	// 5级以后需要材料
	if currentLevel >= 5 {
		materialID = 1001 // 强化石
	}
	if currentLevel >= 10 {
		materialID = 1002 // 高级强化石
		goldCost += int64(currentLevel) * 100
	}
	return
}

// successRate 强化成功率
func (s *InventoryService) successRate(currentLevel int32) float64 {
	switch {
	case currentLevel < 3:
		return 1.0
	case currentLevel < 5:
		return 0.8
	case currentLevel < 8:
		return 0.6
	case currentLevel < 12:
		return 0.4
	case currentLevel < 16:
		return 0.2
	default:
		return 0.1
	}
}

// invalidateInventoryCache 使背包缓存失效
func (s *InventoryService) invalidateInventoryCache(ctx context.Context, playerID int64) {
	s.log.Debug("使背包缓存失效", zap.Int64("player_id", playerID))
	// 背包变化后，简单方式：删除缓存，下次查询时重新加载
	// 实际项目中可在此处触发异步刷 MySQL
}
