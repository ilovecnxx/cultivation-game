package model

import "time"

// VipPlayer 玩家VIP信息（映射 vip_players 表）
type VipPlayer struct {
	ID                  int64      `json:"id" gorm:"primaryKey"`
	PlayerID            int64      `json:"player_id" gorm:"uniqueIndex;not null"`
	VipLevel            int        `json:"vip_level" gorm:"default:0"`
	VipExp              int64      `json:"vip_exp" gorm:"default:0"`
	TotalRecharge       int64      `json:"total_recharge" gorm:"default:0"`
	MonthlyCardExpiresAt *time.Time `json:"monthly_card_expires_at"`
	MonthlyCardType     int8       `json:"monthly_card_type" gorm:"default:0"`
	LastDailyClaimDate  *string    `json:"last_daily_claim_date"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// VipRechargeRecord 充值记录（映射 vip_recharge_records 表）
type VipRechargeRecord struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	PlayerID  int64     `json:"player_id" gorm:"index;not null"`
	AmountJade int     `json:"amount_jade"`
	AmountRmb  int     `json:"amount_rmb"` // 单位:分（100分=1元）
	OrderID   string   `json:"order_id" gorm:"uniqueIndex;size:64"`
	Status    int8     `json:"status" gorm:"default:0"` // 0=待支付 1=已完成 2=已失败
	CreatedAt time.Time `json:"created_at"`
}

// VipRechargeStatus 充值状态常量
const (
	RechargeStatusPending   = 0 // 待支付
	RechargeStatusCompleted = 1 // 已完成
	RechargeStatusFailed    = 2 // 已失败
)

// VipLevelConfig VIP等级配置（从 vip.json 加载）
type VipLevelConfig struct {
	Level               int               `json:"level"`
	RequiredExp         int64             `json:"required_exp"`
	SpeedBonus          int               `json:"speed_bonus"`          // 修炼速度加成百分比
	DailyRewardItems    []VipDailyReward  `json:"daily_reward_items"`   // 每日奖励物品列表
	AuctionFeeDiscount  int               `json:"auction_fee_discount"` // 拍卖手续费折扣百分比
	ExtraSweepTickets   int               `json:"extra_sweep_tickets"`  // 额外扫荡券
	ExtraDungeonAttempts int              `json:"extra_dungeon_attempts"` // 额外副本次数
}

// VipDailyReward VIP每日奖励物品
type VipDailyReward struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

// -------- 请求/响应结构体 --------

// VipInfoResponse VIP信息响应
type VipInfoResponse struct {
	PlayerID            int64            `json:"player_id"`
	VipLevel            int              `json:"vip_level"`
	VipExp              int64            `json:"vip_exp"`
	TotalRecharge       int64            `json:"total_recharge"`
	NextLevelExp        int64            `json:"next_level_exp,omitempty"` // 升级到下一级所需经验，满级时为0
	SpeedBonus          int              `json:"speed_bonus"`
	AuctionFeeDiscount  int              `json:"auction_fee_discount"`
	ExtraSweepTickets   int              `json:"extra_sweep_tickets"`
	ExtraDungeonAttempts int             `json:"extra_dungeon_attempts"`
	DailyRewardItems    []VipDailyReward `json:"daily_reward_items"`
	CanClaimDaily       bool             `json:"can_claim_daily"`
	MonthlyCardType     int8             `json:"monthly_card_type"`
	MonthlyCardActive   bool             `json:"monthly_card_active"`
	MonthlyCardExpiresAt *string         `json:"monthly_card_expires_at,omitempty"`
}

// RechargeRequest 充值请求
type RechargeRequest struct {
	PlayerID int64  `json:"player_id" binding:"required"`
	AmountRmb int   `json:"amount_rmb" binding:"required,min=100"` // 单位:分，最低1元
	OrderID  string `json:"order_id" binding:"required"`
}

// RechargeResponse 充值响应
type RechargeResponse struct {
	OrderID    string `json:"order_id"`
	AmountJade int    `json:"amount_jade"`
	AddedExp   int64  `json:"added_exp"`
	VipLevel   int    `json:"vip_level"`
	NewJade    int64  `json:"new_jade"`
}

// ActivateMonthlyCardRequest 激活月卡请求
type ActivateMonthlyCardRequest struct {
	PlayerID int64 `json:"player_id" binding:"required"`
	CardType int8  `json:"card_type" binding:"required,oneof=1 2"`
}

// MonthlyCardStatusResponse 月卡状态响应
type MonthlyCardStatusResponse struct {
	CardType          int8   `json:"card_type"`
	Active            bool   `json:"active"`
	ExpiresAt         string `json:"expires_at,omitempty"`
	RemainingDays     int    `json:"remaining_days,omitempty"`
}
