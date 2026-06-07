// Package service 黑市系统
//
// 功能要点：
//   - 每次刷新 8 件物品，从 8 类物品池中随机抽取
//   - 每 6 小时自动刷新（0:00 / 6:00 / 12:00 / 18:00）
//   - 支持仙玉强制刷新
//   - 价格随机浮动：市场价的 70%~150%（-30% ~ +50%）
//   - VIP 等级折扣：VIP 越高折扣越大（0%~25%）
//   - 特殊事件：双倍库存、全场折扣、稀有物品出现
//   - 每位玩家购买记录追踪
package service

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"cultivation-game/services/trade/internal/model"
)

// ============================================================================
// 常量
// ============================================================================

const (
	// BlackMarketItemCount 每次刷新的物品数量
	BlackMarketItemCount = 8

	// DefaultRefreshJadeCost 强制刷新消耗的仙玉
	DefaultRefreshJadeCost = 50

	// EventTriggerChance 触发特殊事件的概率（百分比）
	EventTriggerChance = 15

	// PriceMinPercent 最低价格百分比（相对于市场价）
	PriceMinPercent = 70

	// PriceMaxPercent 最高价格百分比（相对于市场价）
	PriceMaxPercent = 150

	// RefreshIntervalHours 自动刷新间隔（小时）
	RefreshIntervalHours = 6
)

// vipDiscountRates VIP 等级对应的额外折扣百分比
// index = VIP 等级，value = 折扣百分比
var vipDiscountRates = []int{0, 5, 10, 15, 20, 25}

// ============================================================================
// VipDiscountRate 获取指定 VIP 等级的折扣率
// ============================================================================

func VipDiscountRate(vipLevel int) int {
	if vipLevel < 0 {
		return 0
	}
	if vipLevel >= len(vipDiscountRates) {
		return vipDiscountRates[len(vipDiscountRates)-1]
	}
	return vipDiscountRates[vipLevel]
}

// ============================================================================
// BlackMarketService
// ============================================================================

// BlackMarketService 黑市业务逻辑
type BlackMarketService struct {
	mu              sync.RWMutex
	items           []*model.BlackMarketItem // 当前黑市物品列表
	lastRefresh     time.Time                // 上次刷新时间
	activeEvent     *model.BlackMarketEvent  // 当前特殊事件（nil=无）
	purchaseRecords map[uint64][]*model.BlackMarketPurchaseRecord // 玩家购买记录
	refreshJadeCost int64                    // 强制刷新消耗仙玉
}

// NewBlackMarketService 创建黑市服务并执行首次刷新
func NewBlackMarketService() *BlackMarketService {
	svc := &BlackMarketService{
		purchaseRecords: make(map[uint64][]*model.BlackMarketPurchaseRecord),
		refreshJadeCost: DefaultRefreshJadeCost,
	}
	svc.refresh()
	return svc
}

// ============================================================================
// 内部刷新逻辑
// ============================================================================

// refresh 刷新黑市物品。如果距上次刷新不足 6 小时且未到下一个整点则不刷新。
func (s *BlackMarketService) refresh() {
	now := time.Now()
	if !s.needsRefresh(now) {
		return
	}

	// 从物品池中按权重选取
	selected := s.pickItemsFromPool(model.BlackMarketPool, BlackMarketItemCount)
	if len(selected) == 0 {
		return
	}

	// 计算下次过期时间
	expiresAt := s.nextScheduledTime(now).Unix()

	// 生成最终物品（随机定价）
	items := make([]*model.BlackMarketItem, 0, len(selected))
	seq := 0
	for _, p := range selected {
		seq++

		// 价格随机浮动 70%~150%
		priceFactor := PriceMinPercent + rand.Intn(PriceMaxPercent-PriceMinPercent+1)
		priceStone := p.BasePriceStone * int64(priceFactor) / 100
		priceJade := p.BasePriceJade * int64(priceFactor) / 100

		// 库存随机
		stock := p.MinStock
		if p.MaxStock > p.MinStock {
			stock += rand.Intn(p.MaxStock - p.MinStock + 1)
		}

		item := &model.BlackMarketItem{
			ID:            fmt.Sprintf("bm_%d_%d", seq, now.Unix()),
			ItemID:        p.ItemID,
			Name:          p.Name,
			Type:          p.Type,
			Description:   p.Description,
			PriceStone:    priceStone,
			PriceJade:     priceJade,
			Stock:         stock,
			OriginalPrice: p.BasePriceStone,
			Discount:      priceFactor,
			ExpiresAt:     expiresAt,
			MaxPerPlayer:  p.MaxPerPlayer,
			Rarity:        p.Rarity,
		}
		items = append(items, item)
	}

	// 尝试触发特殊事件
	var activeEvent *model.BlackMarketEvent
	if rand.Intn(100) < EventTriggerChance {
		activeEvent = s.generateEvent(items, now, expiresAt)
	}

	// 注意：refresh() 始终在调用方已持有 mu.Lock() 的上下文中执行，
	// 因此不需要在此处再加锁。
	s.items = items
	s.lastRefresh = now
	s.activeEvent = activeEvent
}

// needsRefresh 检查是否需要自动刷新
func (s *BlackMarketService) needsRefresh(now time.Time) bool {
	// 首次启动或上次刷新时间为空时刷新
	next := s.nextScheduledTime(s.lastRefresh)
	return now.After(next) || now.Equal(next)
}

// nextScheduledTime 计算 lastRefresh 之后的下一个计划刷新时间点
// 根据 0/6/12/18 整点计算
func (s *BlackMarketService) nextScheduledTime(ref time.Time) time.Time {
	if ref.IsZero() {
		ref = time.Now()
	}
	for _, h := range model.BlackMarketRefreshTimes {
		t := time.Date(ref.Year(), ref.Month(), ref.Day(), h, 0, 0, 0, ref.Location())
		if t.After(ref) {
			return t
		}
	}
	// 次日 0 点
	return time.Date(ref.Year(), ref.Month(), ref.Day()+1, 0, 0, 0, 0, ref.Location())
}

// pickItemsFromPool 从物品池中按权重选取 N 件不重复物品
func (s *BlackMarketService) pickItemsFromPool(pool []model.BlackMarketItemPool, count int) []model.BlackMarketItemPool {
	if count > len(pool) {
		count = len(pool)
	}

	// 计算总权重，排除已选
	selected := make([]model.BlackMarketItemPool, 0, count)
	used := make([]bool, len(pool))

	for len(selected) < count {
		totalWeight := 0
		for i, p := range pool {
			if !used[i] {
				totalWeight += p.Weight
			}
		}
		if totalWeight == 0 {
			break
		}

		roll := rand.Intn(totalWeight)
		cumulative := 0
		chosen := -1
		for i, p := range pool {
			if used[i] {
				continue
			}
			cumulative += p.Weight
			if roll < cumulative {
				chosen = i
				break
			}
		}
		if chosen == -1 {
			break
		}
		used[chosen] = true
		selected = append(selected, pool[chosen])
	}

	return selected
}

// ============================================================================
// 特殊事件
// ============================================================================

// generateEvent 生成一个随机特殊事件并应用到物品列表上
func (s *BlackMarketService) generateEvent(items []*model.BlackMarketItem, now time.Time, expiresAt int64) *model.BlackMarketEvent {
	eventTypes := []model.BlackMarketEventType{
		model.EventDoubleStock,
		model.EventAllDiscount,
		model.EventRareItem,
	}
	eventType := eventTypes[rand.Intn(len(eventTypes))]

	switch eventType {
	case model.EventDoubleStock:
		for _, item := range items {
			item.Stock *= 2
		}
		return &model.BlackMarketEvent{
			Type:        model.EventDoubleStock,
			Description: "双倍库存活动：所有物品库存翻倍！",
		}

	case model.EventAllDiscount:
		extraDiscount := 10 + rand.Intn(16) // 10%~25% 额外折扣
		for _, item := range items {
			// 额外折扣仅适用于灵石价格
			discountFactor := 100 - extraDiscount
			item.PriceStone = item.PriceStone * int64(discountFactor) / 100
		}
		return &model.BlackMarketEvent{
			Type:        model.EventAllDiscount,
			Description: fmt.Sprintf("全场特惠：所有物品额外 %d%% 折扣！", extraDiscount),
			Discount:    extraDiscount,
		}

	case model.EventRareItem:
		// 从稀有池中选取 1~2 件
		picked := s.pickItemsFromPool(model.BlackMarketRarePool, 1+rand.Intn(2))
		extraItems := make([]model.BlackMarketItem, 0, len(picked))
		for _, p := range picked {
			priceFactor := PriceMinPercent + rand.Intn(PriceMaxPercent-PriceMinPercent+1)
			priceStone := p.BasePriceStone * int64(priceFactor) / 100
			priceJade := p.BasePriceJade * int64(priceFactor) / 100
			stock := p.MinStock
			if p.MaxStock > p.MinStock {
				stock += rand.Intn(p.MaxStock - p.MinStock + 1)
			}
			extraItems = append(extraItems, model.BlackMarketItem{
				ID:            fmt.Sprintf("bm_event_%d_%d", len(items)+len(extraItems)+1, now.Unix()),
				ItemID:        p.ItemID,
				Name:          p.Name,
				Type:          p.Type,
				Description:   p.Description,
				PriceStone:    priceStone,
				PriceJade:     priceJade,
				Stock:         stock,
				OriginalPrice: p.BasePriceStone,
				Discount:      priceFactor,
				ExpiresAt:     expiresAt,
				MaxPerPlayer:  p.MaxPerPlayer,
				Rarity:        p.Rarity,
			})
		}

		// 将稀有物品追加到当前物品列表
		for i := range extraItems {
			items = append(items, &extraItems[i])
		}

		// 同时更新 s.items（因为 s.items 是新 slice，但底层指向的 slice 非 s.items）
		// 这里我们需要额外同步写入 s.items，由调用方在加锁状态下处理
		return &model.BlackMarketEvent{
			Type:        model.EventRareItem,
			Description: fmt.Sprintf("稀有物品出现！%d 件传说中的物品限时出售！", len(extraItems)),
			ExtraItems:  extraItems,
		}
	}

	return nil
}

// ============================================================================
// 公开方法
// ============================================================================

// GetItemList 获取当前黑市物品列表。
// 自动检查是否需要刷新。
func (s *BlackMarketService) GetItemList() []*model.BlackMarketItem {
	// 先检查刷新
	s.mu.Lock()
	s.refresh()
	s.mu.Unlock()

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*model.BlackMarketItem, len(s.items))
	for i, item := range s.items {
		cp := *item
		result[i] = &cp
	}
	return result
}

// GetActiveEvent 获取当前特殊事件
func (s *BlackMarketService) GetActiveEvent() *model.BlackMarketEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.activeEvent == nil {
		return nil
	}
	cp := *s.activeEvent
	// 深拷贝 ExtraItems
	if cp.ExtraItems != nil {
		cp.ExtraItems = make([]model.BlackMarketItem, len(s.activeEvent.ExtraItems))
		copy(cp.ExtraItems, s.activeEvent.ExtraItems)
	}
	return &cp
}

// GetRefreshTime 获取下次自动刷新时间
func (s *BlackMarketService) GetRefreshTime() time.Time {
	s.mu.RLock()
	last := s.lastRefresh
	s.mu.RUnlock()
	return s.nextScheduledTime(last)
}

// ============================================================================
// 购买
// ============================================================================

// BuyItem 购买黑市物品。
// 参数:
//   - itemBMID: 黑市物品 ID (e.g. "bm_1_1234567890")
//   - quantity: 购买数量
//   - playerID: 玩家 ID（用于购买记录和限购检查）
//   - vipLevel: VIP 等级（用于折扣）
//
// 返回值:
//   - item: 购买后的物品快照
//   - totalStone: 实际消耗灵石总数（已扣 VIP 折扣）
//   - totalJade: 实际消耗仙玉总数
//   - vipBonus: VIP 折扣减免的灵石数
//   - err: 错误信息
func (s *BlackMarketService) BuyItem(itemBMID string, quantity int, playerID uint64, vipLevel int) (*model.BlackMarketItem, int64, int64, int64, error) {
	if quantity <= 0 {
		return nil, 0, 0, 0, fmt.Errorf("购买数量必须大于 0")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 先尝试刷新（确保库存是最新的）
	s.refresh()

	// 查找物品
	var found *model.BlackMarketItem
	for _, item := range s.items {
		if item.ID == itemBMID {
			found = item
			break
		}
	}
	if found == nil {
		return nil, 0, 0, 0, fmt.Errorf("物品不存在或已下架")
	}
	if found.Stock < quantity {
		return nil, 0, 0, 0, fmt.Errorf("库存不足，剩余 %d", found.Stock)
	}

	// 检查玩家限购
	if found.MaxPerPlayer > 0 {
		totalBought := 0
		if records, ok := s.purchaseRecords[playerID]; ok {
			for _, r := range records {
				if r.ItemBMID == itemBMID {
					totalBought += r.Quantity
				}
			}
		}
		if totalBought+quantity > found.MaxPerPlayer {
			remaining := found.MaxPerPlayer - totalBought
			if remaining <= 0 {
				return nil, 0, 0, 0, fmt.Errorf("已达购买上限（最多 %d 件）", found.MaxPerPlayer)
			}
			return nil, 0, 0, 0, fmt.Errorf("超出限购数量，最多还能购买 %d 件", remaining)
		}
	}

	// 计算总价（含 VIP 折扣）
	vipRate := VipDiscountRate(vipLevel)
	baseStone := found.PriceStone * int64(quantity)
	baseJade := found.PriceJade * int64(quantity)

	vipReduction := baseStone * int64(vipRate) / 100
	totalStone := baseStone - vipReduction
	totalJade := baseJade

	// 扣减库存
	found.Stock -= quantity

	// 记录购买
	record := &model.BlackMarketPurchaseRecord{
		PlayerID:   playerID,
		ItemBMID:   itemBMID,
		ItemID:     found.ItemID,
		ItemName:   found.Name,
		ItemType:   found.Type,
		Quantity:   quantity,
		TotalStone: totalStone,
		TotalJade:  totalJade,
		Discount:   found.Discount,
		VipBonus:   vipRate,
		Timestamp:  time.Now().Unix(),
	}
	s.purchaseRecords[playerID] = append(s.purchaseRecords[playerID], record)

	// 返回购买后的快照
	itemCopy := *found
	return &itemCopy, totalStone, totalJade, vipReduction, nil
}

// ============================================================================
// 强制刷新
// ============================================================================

// ForceRefresh 强制刷新黑市。
// 玩家消耗仙玉立即刷新所有物品。
//
// 参数:
//   - playerID: 玩家 ID
//
// 返回值:
//   - items: 刷新后的物品列表
//   - cost: 消耗的仙玉数
//   - err: 错误信息
func (s *BlackMarketService) ForceRefresh(playerID uint64) ([]*model.BlackMarketItem, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cost := s.refreshJadeCost

	// 执行刷新
	s.items = nil
	s.activeEvent = nil
	s.lastRefresh = time.Time{} // 置零以确保下次访问时触发刷新
	s.refresh()

	result := make([]*model.BlackMarketItem, len(s.items))
	for i, item := range s.items {
		cp := *item
		result[i] = &cp
	}

	return result, cost, nil
}

// SetRefreshJadeCost 设置强制刷新消耗的仙玉
func (s *BlackMarketService) SetRefreshJadeCost(cost int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshJadeCost = cost
}

// GetRefreshJadeCost 获取强制刷新消耗的仙玉
func (s *BlackMarketService) GetRefreshJadeCost() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.refreshJadeCost
}

// ============================================================================
// 购买记录
// ============================================================================

// GetPurchaseRecords 获取指定玩家的购买记录
func (s *BlackMarketService) GetPurchaseRecords(playerID uint64) []*model.BlackMarketPurchaseRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, ok := s.purchaseRecords[playerID]
	if !ok || len(records) == 0 {
		return []*model.BlackMarketPurchaseRecord{}
	}

	result := make([]*model.BlackMarketPurchaseRecord, len(records))
	for i, r := range records {
		cp := *r
		result[i] = &cp
	}
	return result
}

// GetPlayerPurchaseCount 获取玩家对某物品的购买总数
func (s *BlackMarketService) GetPlayerPurchaseCount(playerID uint64, itemBMID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	if records, ok := s.purchaseRecords[playerID]; ok {
		for _, r := range records {
			if r.ItemBMID == itemBMID {
				total += r.Quantity
			}
		}
	}
	return total
}
