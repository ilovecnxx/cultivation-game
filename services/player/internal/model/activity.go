package model

import "time"

// ============================================================
// 限时活动 (Limited-Time Events)
// ============================================================

// EventType 活动类型
type EventType string

const (
	EventTypeExpBoost    EventType = "exp_boost"     // 双倍修炼
	EventTypeDropBoost   EventType = "drop_boost"    // 双倍掉落
	EventTypeSpecialBoss EventType = "special_boss"  // 限时BOSS
	EventTypeCollection  EventType = "collection"    // 收集活动
	EventTypeRanking     EventType = "ranking"       // 仙魔大战(排行)
	EventTypeRecharge    EventType = "recharge"      // 充值返利
	EventTypeFortune     EventType = "fortune"       // 天降鸿福
)

// LimitedEvent 限时活动定义
type LimitedEvent struct {
	ID          string          `json:"id" gorm:"primaryKey"`
	Name        string          `json:"name" gorm:"size:64"`
	Type        EventType       `json:"type" gorm:"size:32"`
	Description string          `json:"description" gorm:"size:512"`
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time"`
	MinRealm    int             `json:"min_realm" gorm:"default:1"`
	Rewards     []*EventReward   `json:"rewards" gorm:"-"`
	Conditions  []*EventCondition `json:"conditions" gorm:"-"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// EventReward 活动奖励
type EventReward struct {
	ID         string `json:"id"`
	EventID    string `json:"event_id" gorm:"size:64"`
	ItemID     int64  `json:"item_id"`
	ItemName   string `json:"item_name" gorm:"size:64"`
	Quantity   int32  `json:"quantity"`
	Probability float64 `json:"probability"` // 概率(0-1)
	IsGuaranteed bool `json:"is_guaranteed"`
}

// EventCondition 活动条件/进度
type EventCondition struct {
	ID       string `json:"id"`
	EventID  string `json:"event_id" gorm:"size:64"`
	Type     string `json:"type" gorm:"size:32"`      // "kill_monsters", "collect_items", "cultivate_time", "spend_stones"
	Target   int64  `json:"target"`
	Progress int64  `json:"progress"`
	Priority int    `json:"priority" gorm:"default:0"`
}

// EventProgress 玩家活动进度
type EventProgress struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	PlayerID  int64     `json:"player_id"`
	EventID   string    `json:"event_id" gorm:"size:64"`
	Progress  int64     `json:"progress" gorm:"default:0"`
	Claimed   bool      `json:"claimed" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EventRewardRecord 玩家奖励领取记录
type EventRewardRecord struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	PlayerID  int64     `json:"player_id"`
	EventID   string    `json:"event_id" gorm:"size:64"`
	RewardID  string    `json:"reward_id" gorm:"size:64"`
	ClaimedAt time.Time `json:"claimed_at"`
}

// ============================================================
// 战令系统 (Battle Pass)
// ============================================================

// BattlePassSeason 战令赛季
type BattlePassSeason struct {
	SeasonID    string    `json:"season_id" gorm:"primaryKey"`
	SeasonName  string    `json:"season_name" gorm:"size:64"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	PremiumCost int64     `json:"premium_cost"` // 解锁高级战令花费(灵石或仙玉)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BPTier 战令等级定义
type BPTier struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	SeasonID   string    `json:"season_id" gorm:"size:64"`
	Level      int       `json:"level"`
	ExpRequired int64   `json:"exp_required"`
	IsPremium  bool      `json:"is_premium"`
	RewardItemID   int64 `json:"reward_item_id"`
	RewardName     string `json:"reward_name" gorm:"size:64"`
	RewardQuantity int32 `json:"reward_quantity"`
	RewardType     string `json:"reward_type" gorm:"size:32"` // "item", "title", "outfit", "mount", "artifact"
}

// BPProgress 玩家战令进度
type BPProgress struct {
	ID           string `json:"id" gorm:"primaryKey"`
	PlayerID     int64  `json:"player_id"`
	SeasonID     string `json:"season_id" gorm:"size:64"`
	CurrentLevel int    `json:"current_level" gorm:"default:1"`
	CurrentExp   int64  `json:"current_exp" gorm:"default:0"`
	HasPremium   bool   `json:"has_premium" gorm:"default:false"`
	ClaimedLevels string `json:"claimed_levels" gorm:"size:512;default:''"` // CSV of claimed level numbers
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BPRewardClaimLog 奖励领取日志
type BPRewardClaimLog struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	PlayerID  int64     `json:"player_id"`
	SeasonID  string    `json:"season_id" gorm:"size:64"`
	Level     int       `json:"level"`
	ClaimedAt time.Time `json:"claimed_at"`
}

// ============================================================
// 签到增强 (Enhanced Check-in)
// ============================================================

// MonthlyCheckinConfig 每月签到配置
type MonthlyCheckinConfig struct {
	Day         int    `json:"day"`
	ItemID      int64  `json:"item_id"`
	ItemName    string `json:"item_name" gorm:"size:64"`
	Quantity    int32  `json:"quantity"`
	IsMilestone bool   `json:"is_milestone"` // 是否里程碑奖励(第7/14/21/28天)
}

// MonthlyCheckinRewards 28天签到奖励配置
var MonthlyCheckinRewards = []MonthlyCheckinConfig{
	{Day: 1, ItemID: 0, ItemName: "灵石", Quantity: 100},
	{Day: 2, ItemID: 0, ItemName: "灵石", Quantity: 150},
	{Day: 3, ItemID: 1001, ItemName: "聚气丹", Quantity: 1},
	{Day: 4, ItemID: 0, ItemName: "灵石", Quantity: 200},
	{Day: 5, ItemID: 1002, ItemName: "培元丹", Quantity: 1},
	{Day: 6, ItemID: 0, ItemName: "灵石", Quantity: 300},
	{Day: 7, ItemID: 2001, ItemName: "法宝碎片", Quantity: 1, IsMilestone: true},
	{Day: 8, ItemID: 0, ItemName: "灵石", Quantity: 350},
	{Day: 9, ItemID: 1003, ItemName: "凝神丹", Quantity: 1},
	{Day: 10, ItemID: 0, ItemName: "灵石", Quantity: 400},
	{Day: 11, ItemID: 1001, ItemName: "聚气丹", Quantity: 2},
	{Day: 12, ItemID: 0, ItemName: "灵石", Quantity: 450},
	{Day: 13, ItemID: 1002, ItemName: "培元丹", Quantity: 2},
	{Day: 14, ItemID: 3001, ItemName: "灵兽蛋", Quantity: 1, IsMilestone: true},
	{Day: 15, ItemID: 0, ItemName: "灵石", Quantity: 500},
	{Day: 16, ItemID: 1003, ItemName: "凝神丹", Quantity: 2},
	{Day: 17, ItemID: 0, ItemName: "灵石", Quantity: 550},
	{Day: 18, ItemID: 1001, ItemName: "聚气丹", Quantity: 3},
	{Day: 19, ItemID: 0, ItemName: "灵石", Quantity: 600},
	{Day: 20, ItemID: 1002, ItemName: "培元丹", Quantity: 3},
	{Day: 21, ItemID: 4001, ItemName: "随机功法残页", Quantity: 1, IsMilestone: true},
	{Day: 22, ItemID: 0, ItemName: "灵石", Quantity: 650},
	{Day: 23, ItemID: 1003, ItemName: "凝神丹", Quantity: 3},
	{Day: 24, ItemID: 0, ItemName: "灵石", Quantity: 700},
	{Day: 25, ItemID: 1001, ItemName: "聚气丹", Quantity: 4},
	{Day: 26, ItemID: 0, ItemName: "灵石", Quantity: 800},
	{Day: 27, ItemID: 1002, ItemName: "培元丹", Quantity: 4},
	{Day: 28, ItemID: 5001, ItemName: "仙缘宝箱", Quantity: 1, IsMilestone: true},
}

// MakeupCost 补签花费(按次数递增)
func GetMakeupCost(makeupCount int) int64 {
	costs := []int64{50, 80, 120, 180, 250, 350, 500, 800}
	if makeupCount < 0 {
		makeupCount = 0
	}
	if makeupCount >= len(costs) {
		return costs[len(costs)-1] * 2
	}
	return costs[makeupCount]
}

// EnhancedCheckinStatus 增强签到状态响应
type EnhancedCheckinStatus struct {
	CheckedInToday  bool   `json:"checked_in_today"`
	ConsecutiveDays int32  `json:"consecutive_days"`
	MonthTotal      int32  `json:"month_total"`
	MonthDays       []bool `json:"month_days"`        // 28-length array for calendar view
	CanMakeup       bool   `json:"can_makeup"`
	MakeupCost      int64  `json:"makeup_cost"`
	MakeupCount     int    `json:"makeup_count"`      // 本月已补签次数
	StreakBonus     float64 `json:"streak_bonus"`     // 连续签到倍率
	MilestoneClaimed []bool `json:"milestone_claimed"` // [7,14,21,28]已领取
}

// ============================================================
// 成就系统增强 (Enhanced Achievement & Title)
// ============================================================

// AchievementTierLevel 成就等级
const (
	TierBronze  = 1 // 初窥门径
	TierSilver  = 2 // 登堂入室
	TierGold    = 3 // 炉火纯青
	TierDiamond = 4 // 出神入化
)

// Additional achievement categories (beyond those in achievement.go)
const (
	AchievementCatCollectionNew = "collection" // 收集
	AchievementCatActivityNew  = "activity"   // 活动
	AchievementCatHiddenNew    = "hidden"     // 隐藏
)

// AchievementTier 成就等级定义
type AchievementTier struct {
	ID        string `json:"id" gorm:"primaryKey"`
	AchievementID string `json:"achievement_id" gorm:"size:64"`
	Level     int    `json:"level"`         // 1-4
	Name      string `json:"name" gorm:"size:32"`
	Condition int64  `json:"condition"`     // 目标值
	TitleID   string `json:"title_id" gorm:"size:64"` // 可解锁的称号ID
	RewardExp int64  `json:"reward_exp"`
	RewardMoney int64 `json:"reward_money"`
}

// AchievementReq 成就定义(扩展)
type AchievementReq struct {
	ID          string            `json:"id" gorm:"primaryKey"`
	Category    string            `json:"category" gorm:"size:32"`
	Name        string            `json:"name" gorm:"size:64"`
	Description string            `json:"description" gorm:"size:256"`
	IsHidden    bool              `json:"is_hidden" gorm:"default:false"`
	Hint        string            `json:"hint" gorm:"size:128"`    // 隐藏成就提示
	Icon        string            `json:"icon" gorm:"size:32"`
	SortOrder   int               `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PlayerAchievementTier 玩家成就等级进度
type PlayerAchievementTier struct {
	PlayerID      int64     `json:"player_id" gorm:"primaryKey"`
	AchievementID string    `json:"achievement_id" gorm:"primaryKey;size:64"`
	CurrentTier   int       `json:"current_tier" gorm:"default:0"`   // 当前最高等级
	Progress      int64     `json:"progress" gorm:"default:0"`       // 当前进度
	Completed     bool      `json:"completed" gorm:"default:false"`
	ClaimedTiers  string    `json:"claimed_tiers" gorm:"size:32;default:''"` // CSV
	CompletedAt   time.Time `json:"completed_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Title 称号定义
type Title struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:32"`
	Description string    `json:"description" gorm:"size:128"`
	Color       string    `json:"color" gorm:"size:16"`     // 显示颜色, e.g. "#FFD700"
	Source      string    `json:"source" gorm:"size:64"`    // 来源说明
	StatBonusHP     int64   `json:"stat_bonus_hp" gorm:"default:0"`
	StatBonusAttack int64   `json:"stat_bonus_attack" gorm:"default:0"`
	StatBonusDefense int64  `json:"stat_bonus_defense" gorm:"default:0"`
	StatBonusSpeed  float64 `json:"stat_bonus_speed" gorm:"default:0"`
	Rarity      int       `json:"rarity" gorm:"default:1"` // 1-5
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PlayerTitleEnhanced 玩家已获得的称号(增强版)
type PlayerTitleEnhanced struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	PlayerID  int64     `json:"player_id"`
	TitleID   string    `json:"title_id" gorm:"size:64"`
	IsEquipped bool     `json:"is_equipped" gorm:"default:false"`
	ObtainedAt time.Time `json:"obtained_at"`
}

// PlayerTitleResponse 称号响应(增强版)
type PlayerTitleResponse struct {
	PlayerID    int64     `json:"player_id"`
	CurrentTitle *Title   `json:"current_title"`
	Titles      []*Title  `json:"titles"`
	TotalPoints int       `json:"total_points"` // 成就点总数
}

// ============================================================
// 响应结构体
// ============================================================

// ActivityCenterResponse 活动中心响应
type ActivityCenterResponse struct {
	Events      []*LimitedEvent       `json:"events"`
	BattlePass  *BattlePassStatus     `json:"battle_pass"`
	Checkin     *EnhancedCheckinStatus `json:"checkin"`
}

// BattlePassStatus 战令状态响应
type BattlePassStatus struct {
	Season      *BattlePassSeason `json:"season"`
	Progress    *BPProgress       `json:"progress"`
	FreeTiers   []*BPTier         `json:"free_tiers"`
	PremiumTiers []*BPTier        `json:"premium_tiers"`
}

// EventDetailResponse 活动详情响应
type EventDetailResponse struct {
	Event    *LimitedEvent     `json:"event"`
	Progress *EventProgress   `json:"progress"`
	Rewards  []*EventReward   `json:"rewards"`
}
