package model

import "time"

// ============================================================
// 签到系统模型
// ============================================================

// CheckinRecord 玩家签到记录
type CheckinRecord struct {
	PlayerID          int64     `json:"player_id" gorm:"primaryKey"`
	LastCheckinDate   string    `json:"last_checkin_date" gorm:"size:10"`            // 最后签到日期 YYYY-MM-DD
	ConsecutiveDays   int32     `json:"consecutive_days" gorm:"default:0"`           // 连续签到天数
	WeekStartDate     string    `json:"week_start_date" gorm:"size:10"`              // 本周周一日期 YYYY-MM-DD
	WeekClaimedMask   int32     `json:"week_claimed_mask" gorm:"default:0"`          // 本周签到位掩码(bit0=周一..bit6=周日)
	MonthTotal        int32     `json:"month_total" gorm:"default:0"`                // 本月累计签到天数
	MonthStr          string    `json:"month_str" gorm:"size:7"`                     // 本月标识 YYYY-MM
	MonthRewardClaimed bool    `json:"month_reward_claimed" gorm:"default:false"`    // 本月满签奖励是否已领取
	MakeupDate        string    `json:"makeup_date" gorm:"size:10"`                  // 补签日期
	MakeupUsedToday   int32     `json:"makeup_used_today" gorm:"default:0"`          // 今日是否已补签(0/1)
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CheckinReward 签到奖励定义(连续第N天)
type CheckinReward struct {
	Day      int    `json:"day"`                // 连续签到天数(1-7)
	Gold     int64  `json:"gold,omitempty"`     // 灵石
	ItemID   int64  `json:"item_id,omitempty"`  // 物品ID
	ItemName string `json:"item_name,omitempty"`
	Quantity int32  `json:"quantity,omitempty"` // 物品数量
	Luck     int32  `json:"luck,omitempty"`     // 气运值
}

// CheckinRewards 每日签到奖励配置(连续1-7天)
var CheckinRewards = []CheckinReward{
	{Day: 1, Gold: 100, ItemName: "灵石", Quantity: 100},
	{Day: 2, Gold: 150, ItemName: "灵石", Quantity: 150},
	{Day: 3, ItemID: 1001, ItemName: "聚气丹", Quantity: 1},
	{Day: 4, Gold: 200, ItemName: "灵石", Quantity: 200},
	{Day: 5, ItemID: 1002, ItemName: "培元丹", Quantity: 1},
	{Day: 6, Gold: 300, ItemName: "灵石", Quantity: 300},
	{Day: 7, Luck: 10, ItemName: "气运符", Quantity: 1},
}

// MakeupCheckinCost 每次补签消耗灵石
const MakeupCheckinCost int64 = 50

// MonthlyFullCheckinDays 月满签天数
const MonthlyFullCheckinDays int32 = 28

// CheckinStatus 签到状态响应
type CheckinStatus struct {
	CheckedInToday  bool   `json:"checked_in_today"`    // 今日是否已签到
	ConsecutiveDays int32  `json:"consecutive_days"`    // 当前连续签到天数
	WeekClaimed     []bool `json:"week_claimed"`        // 本周签到情况[7]bool
	MonthTotal      int32  `json:"month_total"`         // 本月签到天数
	CanMakeup       bool   `json:"can_makeup"`          // 今日是否可补签
	MakeupCost      int64  `json:"makeup_cost"`         // 补签消耗灵石
	FullMonthReward bool   `json:"full_month_reward"`   // 本月是否已满签
}

// CheckinResult 签到结果
type CheckinResult struct {
	Reward     *CheckinReward `json:"reward"`      // 本次奖励
	Makeup     bool           `json:"makeup"`      // 是否补签
	CostGold   int64          `json:"cost_gold"`    // 消耗灵石(补签时)
	FullMonth  bool           `json:"full_month"`   // 是否达成满签
	MonthTotal int32          `json:"month_total"`  // 本月累计
	Streak     int32          `json:"streak"`       // 当前连续
}
