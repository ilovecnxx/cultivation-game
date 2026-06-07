// Package model 定义仙玉商城系统的数据模型，包括商品、VIP信息和购买请求。
package model

// ============================================================================
// 商品模型
// ============================================================================

// ShopItem 商城商品。
type ShopItem struct {
	ID          uint32  `json:"id"`                    // 商品 ID
	Name        string  `json:"name"`                  // 商品名称
	Category    string  `json:"category"`              // 分类：丹药/材料/时装/功法/宝箱
	Description string  `json:"description"`           // 商品描述
	PriceType   string  `json:"price_type"`            // 货币类型：jade（仙玉）/spirit_stone（灵石）
	Price       uint64  `json:"price"`                 // 价格
	Stock       int32   `json:"stock"`                 // 库存（-1 表示不限量）
	LimitBuy    int32   `json:"limit_buy"`             // 每日限购次数（0 表示不限）
	VipLevel    int32   `json:"vip_level"`             // 所需最低 VIP 等级（0 表示无限制）
	Icon        string  `json:"icon"`                  // 图标标识
	Discount    float64 `json:"discount"`               // 折扣（1.0 为原价）
	SortOrder   int32   `json:"sort_order"`            // 排序序号
}

// DiscountedPrice 返回折扣后的价格，向下取整。
func (s *ShopItem) DiscountedPrice() uint64 {
	if s.Discount <= 0 || s.Discount >= 1.0 {
		return s.Price
	}
	return uint64(float64(s.Price) * s.Discount)
}

// ============================================================================
// VIP 系统
// ============================================================================

// VIPInfo VIP 信息。
type VIPInfo struct {
	Level      int32           `json:"level"`       // VIP 等级（0 表示非 VIP）
	Exp        uint64          `json:"exp"`         // 当前 VIP 经验
	SpeedBonus int32           `json:"speed_bonus"` // 修炼速度加成百分比（VIP1=5, VIP10=50）
	DailyClaim *VIPDailyReward `json:"daily_claim"` // 每日可领取奖励（已领取时为 nil）
}

// VIPConfig VIP 等级配置。
type VIPConfig struct {
	Level      int32            `json:"level"`       // VIP 等级
	NeedExp    uint64           `json:"need_exp"`    // 升级所需经验
	SpeedBonus int32            `json:"speed_bonus"` // 修炼速度加成百分比
	DailyItems []VIPRewardItem  `json:"daily_items"` // 每日可领取物品
}

// VIPRewardItem VIP 每日奖励物品定义。
type VIPRewardItem struct {
	ItemID   uint32 `json:"item_id"`   // 物品 ID
	ItemName string `json:"item_name"` // 物品名称
	Quantity uint32 `json:"quantity"`  // 数量
}

// VIPDailyReward 玩家每日 VIP 领取记录。
type VIPDailyReward struct {
	PlayerID  uint64   `json:"player_id"`  // 玩家 ID
	VIPLevel  int32    `json:"vip_level"`  // 领取时的 VIP 等级
	Date      string   `json:"date"`       // 领取日期（YYYY-MM-DD）
	Items     []Reward `json:"items"`      // 实际领取的物品
}

// Reward 领取奖励条目。
type Reward struct {
	ItemID   uint32 `json:"item_id"`   // 物品 ID
	ItemName string `json:"item_name"` // 物品名称
	Quantity uint32 `json:"quantity"`  // 数量
}

// ============================================================================
// 请求/响应
// ============================================================================

// BuyReq 购买请求。
type BuyReq struct {
	PlayerID uint64 `json:"player_id"` // 玩家 ID
	ItemID   uint32 `json:"item_id"`   // 商品 ID
	Quantity uint32 `json:"quantity"`  // 购买数量
}

// BuyResp 购买响应。
type BuyResp struct {
	Success      bool   `json:"success"`       // 是否成功
	TotalCost    uint64 `json:"total_cost"`     // 总花费
	RemainingJade uint64 `json:"remaining_jade"` // 剩余仙玉
	RemainingStone uint64 `json:"remaining_stone"` // 剩余灵石
}

// RechargeReq 充值请求（模拟）。
type RechargeReq struct {
	PlayerID uint64 `json:"player_id"` // 玩家 ID
	Amount   uint64 `json:"amount"`    // 充值金额（元）
}

// RechargeResp 充值响应。
type RechargeResp struct {
	Success      bool   `json:"success"`        // 是否成功
	Amount       uint64 `json:"amount"`         // 充值金额（元）
	ObtainedJade uint64 `json:"obtained_jade"`  // 获得仙玉（amount * 10）
	TotalJade    uint64 `json:"total_jade"`     // 当前仙玉总数
}

// ClaimVIPReq 领取 VIP 每日奖励请求。
type ClaimVIPReq struct {
	PlayerID uint64 `json:"player_id"` // 玩家 ID
}

// ClaimVIPResp 领取 VIP 每日奖励响应。
type ClaimVIPResp struct {
	Success bool     `json:"success"`
	Items   []Reward `json:"items"`
}
