// Package service 实现仙玉商城业务逻辑：商品查询、购买、VIP 与充值。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cultivation-game/services/trade/internal/shop/model"
)

// ============================================================================
// 错误定义
// ============================================================================

var (
	ErrItemNotFound       = errors.New("商品不存在")
	ErrInsufficientJade  = errors.New("仙玉不足")
	ErrInsufficientStone = errors.New("灵石不足")
	ErrOutOfStock        = errors.New("库存不足")
	ErrLimitReached      = errors.New("已达到每日限购次数")
	ErrVipLevelRequired  = errors.New("VIP 等级不足")
	ErrInvalidQuantity   = errors.New("购买数量无效")
	ErrNotVip            = errors.New("非 VIP 用户")
	ErrAlreadyClaimed    = errors.New("今日已领取过 VIP 奖励")
)

// ============================================================================
// 玩家资产（内存模拟）
// ============================================================================

// playerAssets 玩家资产状态。
type playerAssets struct {
	Jade        uint64                  // 仙玉
	Stones      uint64                  // 灵石
	VipLevel    int32                   // VIP 等级
	VipExp      uint64                  // VIP 经验
	DailyBuy    map[uint32]int32        // 商品 ID -> 今日已购买次数
	LastClaim   string                  // 上次领取 VIP 奖励日期（YYYY-MM-DD）
}

// ============================================================================
// ShopService 定义
// ============================================================================

// ShopService 仙玉商城服务，处理商品查询、购买、VIP 信息展示和充值。
type ShopService struct {
	items     []model.ShopItem
	itemIndex map[uint32]*model.ShopItem
	vipCfgs   []model.VIPConfig
	vipMap    map[int32]*model.VIPConfig

	players sync.Map // map[uint64]*playerAssets

	dataDir string
	log     *slog.Logger
}

// NewShopService 创建 ShopService，从 dataDir 加载商品和 VIP 配置。
func NewShopService(dataDir string, log *slog.Logger) (*ShopService, error) {
	s := &ShopService{
		itemIndex: make(map[uint32]*model.ShopItem),
		vipMap:    make(map[int32]*model.VIPConfig),
		dataDir:   dataDir,
		log:       log,
	}

	if err := s.loadData(); err != nil {
		return nil, fmt.Errorf("加载商城数据失败: %w", err)
	}

	// 初始化默认玩家资产（方便测试）
	s.ensurePlayer(1)

	return s, nil
}

// loadData 从 JSON 文件加载商品和 VIP 配置。
func (s *ShopService) loadData() error {
	path := filepath.Join(s.dataDir, "shop_items.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var raw struct {
		Items     []model.ShopItem   `json:"items"`
		VIPConfigs []model.VIPConfig `json:"vip_configs"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for i := range raw.Items {
		item := raw.Items[i]
		s.itemIndex[item.ID] = &item
		s.items = append(s.items, item)
	}

	for i := range raw.VIPConfigs {
		cfg := raw.VIPConfigs[i]
		s.vipMap[cfg.Level] = &cfg
		s.vipCfgs = append(s.vipCfgs, cfg)
	}

	s.log.InfoContext(context.Background(), "商城数据加载完成",
		"items", len(s.items),
		"vip_configs", len(s.vipCfgs),
	)
	return nil
}

// ensurePlayer 确保玩家资产存在，不存在则初始化。
func (s *ShopService) ensurePlayer(playerID uint64) *playerAssets {
	v, _ := s.players.LoadOrStore(playerID, &playerAssets{
		Jade:      1000,  // 初始赠送 1000 仙玉
		Stones:    10000, // 初始赠送 10000 灵石
		VipLevel:  0,
		DailyBuy:  make(map[uint32]int32),
		LastClaim: "",
	})
	return v.(*playerAssets)
}

// ============================================================================
// 商品查询
// ============================================================================

// GetItems 获取商品列表，可选按分类筛选。
func (s *ShopService) GetItems(_ context.Context, category string) []model.ShopItem {
	if category == "" {
		result := make([]model.ShopItem, len(s.items))
		copy(result, s.items)
		return result
	}
	var result []model.ShopItem
	for _, item := range s.items {
		if item.Category == category {
			result = append(result, item)
		}
	}
	return result
}

// GetItemByID 根据 ID 获取单个商品。
func (s *ShopService) GetItemByID(_ context.Context, itemID uint32) (*model.ShopItem, bool) {
	item, ok := s.itemIndex[itemID]
	return item, ok
}

// GetCategories 获取所有商品分类列表。
func (s *ShopService) GetCategories() []string {
	seen := make(map[string]bool)
	var cats []string
	for _, item := range s.items {
		if !seen[item.Category] {
			seen[item.Category] = true
			cats = append(cats, item.Category)
		}
	}
	return cats
}

// ============================================================================
// 购买逻辑
// ============================================================================

// Buy 购买商品。包含：商品存在性 -> 库存 -> VIP 等级 -> 每日限购 -> 货币余额 -> 扣款。
func (s *ShopService) Buy(_ context.Context, req *model.BuyReq) (*model.BuyResp, error) {
	if req.Quantity == 0 {
		return nil, ErrInvalidQuantity
	}

	// 1. 商品存在性
	item, ok := s.itemIndex[req.ItemID]
	if !ok {
		return nil, ErrItemNotFound
	}

	// 2. 库存检查
	if item.Stock >= 0 && item.Stock < int32(req.Quantity) {
		return nil, ErrOutOfStock
	}

	// 3. VIP 等级检查
	player := s.ensurePlayer(req.PlayerID)
	if item.VipLevel > 0 && player.VipLevel < item.VipLevel {
		return nil, ErrVipLevelRequired
	}

	// 4. 每日限购检查
	if item.LimitBuy > 0 {
		existing := player.DailyBuy[item.ID]
		if existing+int32(req.Quantity) > item.LimitBuy {
			return nil, ErrLimitReached
		}
	}

	totalCost := item.DiscountedPrice() * uint64(req.Quantity)

	// 5. 货币余额检查
	switch item.PriceType {
	case "jade":
		if player.Jade < totalCost {
			return nil, ErrInsufficientJade
		}
	case "spirit_stone":
		if player.Stones < totalCost {
			return nil, ErrInsufficientStone
		}
	default:
		return nil, fmt.Errorf("不支持的货币类型: %s", item.PriceType)
	}

	// 6. 扣款
	switch item.PriceType {
	case "jade":
		player.Jade -= totalCost
	case "spirit_stone":
		player.Stones -= totalCost
	}

	// 7. 更新库存和限购
	if item.Stock > 0 {
		item.Stock -= int32(req.Quantity)
	}
	if item.LimitBuy > 0 {
		player.DailyBuy[item.ID] += int32(req.Quantity)
	}

	// 8. 购买仙玉商品时增加 VIP 经验（每消费 10 仙玉 = 1 点 VIP 经验）
	if item.PriceType == "jade" && totalCost > 0 {
		expGain := totalCost / 10
		player.VipExp += expGain
		s.tryUpgradeVIP(player)
	}

	s.log.InfoContext(context.Background(), "购买成功",
		"player_id", req.PlayerID,
		"item_id", req.ItemID,
		"quantity", req.Quantity,
		"total_cost", totalCost,
		"price_type", item.PriceType,
	)

	return &model.BuyResp{
		Success:        true,
		TotalCost:      totalCost,
		RemainingJade:  player.Jade,
		RemainingStone: player.Stones,
	}, nil
}

// tryUpgradeVIP 检查并升级玩家 VIP 等级。
func (s *ShopService) tryUpgradeVIP(player *playerAssets) {
	for _, cfg := range s.vipCfgs {
		if cfg.Level > player.VipLevel && player.VipExp >= cfg.NeedExp {
			player.VipLevel = cfg.Level
			s.log.InfoContext(context.Background(), "VIP 升级",
				"new_level", cfg.Level,
			)
		}
	}
}

// ============================================================================
// VIP 信息
// ============================================================================

// GetVIPInfo 获取玩家 VIP 信息，包括每日领取状态。
func (s *ShopService) GetVIPInfo(_ context.Context, playerID uint64) (*model.VIPInfo, error) {
	player := s.ensurePlayer(playerID)

	cfg, ok := s.vipMap[player.VipLevel]
	if !ok {
		// 非 VIP
		return &model.VIPInfo{
			Level:      0,
			SpeedBonus: 0,
			DailyClaim: nil,
		}, nil
	}

	today := time.Now().Format("2006-01-02")
	var claim *model.VIPDailyReward
	if player.LastClaim != today {
		var rewards []model.Reward
		for _, di := range cfg.DailyItems {
			rewards = append(rewards, model.Reward{
				ItemID:   di.ItemID,
				ItemName: di.ItemName,
				Quantity: di.Quantity,
			})
		}
		claim = &model.VIPDailyReward{
			VIPLevel: cfg.Level,
			Items:    rewards,
		}
	}

	return &model.VIPInfo{
		Level:      player.VipLevel,
		Exp:        player.VipExp,
		SpeedBonus: cfg.SpeedBonus,
		DailyClaim: claim,
	}, nil
}

// ClaimVIPDaily 领取 VIP 每日奖励。
func (s *ShopService) ClaimVIPDaily(_ context.Context, playerID uint64) (*model.ClaimVIPResp, error) {
	player := s.ensurePlayer(playerID)

	if player.VipLevel <= 0 {
		return nil, ErrNotVip
	}

	cfg, ok := s.vipMap[player.VipLevel]
	if !ok {
		return nil, ErrNotVip
	}

	today := time.Now().Format("2006-01-02")
	if player.LastClaim == today {
		return nil, ErrAlreadyClaimed
	}

	var items []model.Reward
	for _, di := range cfg.DailyItems {
		items = append(items, model.Reward{
			ItemID:   di.ItemID,
			ItemName: di.ItemName,
			Quantity: di.Quantity,
		})
		// 发放灵石到玩家资产
		if di.ItemID == 2001 {
			player.Stones += uint64(di.Quantity)
		}
	}

	player.LastClaim = today

	s.log.InfoContext(context.Background(), "VIP 每日奖励领取",
		"player_id", playerID,
		"vip_level", player.VipLevel,
	)

	return &model.ClaimVIPResp{
		Success: true,
		Items:   items,
	}, nil
}

// ============================================================================
// 充值（模拟）
// ============================================================================

// Recharge 模拟充值：1 元 = 10 仙玉。
func (s *ShopService) Recharge(_ context.Context, req *model.RechargeReq) (*model.RechargeResp, error) {
	if req.Amount == 0 {
		return nil, errors.New("充值金额必须大于 0")
	}
	if req.Amount > 999999 {
		return nil, errors.New("单次充值金额不能超过 999999 元")
	}

	player := s.ensurePlayer(req.PlayerID)
	obtainedJade := req.Amount * 10
	player.Jade += obtainedJade

	// 充值也增加 VIP 经验（1 元 = 1 点 VIP 经验）
	player.VipExp += req.Amount
	s.tryUpgradeVIP(player)

	s.log.InfoContext(context.Background(), "充值成功",
		"player_id", req.PlayerID,
		"amount", req.Amount,
		"obtained_jade", obtainedJade,
	)

	return &model.RechargeResp{
		Success:      true,
		Amount:       req.Amount,
		ObtainedJade: obtainedJade,
		TotalJade:    player.Jade,
	}, nil
}

// ============================================================================
// VIP 等级配置查询
// ============================================================================

// GetVIPConfig 获取某个 VIP 等级的配置。
func (s *ShopService) GetVIPConfig(level int32) (*model.VIPConfig, bool) {
	cfg, ok := s.vipMap[level]
	return cfg, ok
}

// GetAllVIPConfigs 获取所有 VIP 等级配置。
func (s *ShopService) GetAllVIPConfigs() []model.VIPConfig {
	result := make([]model.VIPConfig, len(s.vipCfgs))
	copy(result, s.vipCfgs)
	return result
}

// ============================================================================
// 随机宝箱开箱
// ============================================================================

// OpenChest 打开宝箱，根据宝箱 ID 返回随机奖励。
// 仅供演示，实际开箱逻辑由游戏核心服务实现。
func (s *ShopService) OpenChest(ctx context.Context, playerID uint64, itemID uint32) ([]model.Reward, error) {
	item, ok := s.itemIndex[itemID]
	if !ok {
		return nil, ErrItemNotFound
	}
	if item.Category != "宝箱" {
		return nil, errors.New("该商品不是宝箱")
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	switch itemID {
	case 5002: // 灵石宝箱
		amount := uint64(5000 + rng.Intn(5001))
		player := s.ensurePlayer(playerID)
		player.Stones += amount
		return []model.Reward{{ItemID: 2001, ItemName: "灵石", Quantity: uint32(amount)}}, nil

	case 5003: // 修炼加速宝箱
		return []model.Reward{
			{ItemID: 6001, ItemName: "修炼加速符", Quantity: 3},
		}, nil

	case 5004: // VIP 周卡
		player := s.ensurePlayer(playerID)
		player.VipExp += 5000
		s.tryUpgradeVIP(player)
		return []model.Reward{
			{ItemID: 2001, ItemName: "灵石", Quantity: 1000},
		}, nil

	case 5005: // 至尊礼包
		player := s.ensurePlayer(playerID)
		player.Stones += 50000
		return []model.Reward{
			{ItemID: 1010, ItemName: "九转金丹", Quantity: 3},
			{ItemID: 2001, ItemName: "灵石", Quantity: 50000},
			{ItemID: 7001, ItemName: "限定称号·至尊", Quantity: 1},
		}, nil

	default: // 新手礼包等
		player := s.ensurePlayer(playerID)
		player.Stones += 1000
		return []model.Reward{
			{ItemID: 1001, ItemName: "聚气丹", Quantity: 5},
			{ItemID: 2001, ItemName: "灵石", Quantity: 1000},
		}, nil
	}
}
