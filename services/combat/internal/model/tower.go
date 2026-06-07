// Package model 定义心魔塔（幻境爬塔）核心数据结构。
package model

// HeartDemonType 心魔类型枚举。
type HeartDemonType string

const (
	DemonGreed  HeartDemonType = "greed"  // 贪：损失灵石心境
	DemonWrath  HeartDemonType = "wrath"  // 嗔：被激怒，攻性提升
	DemonIgnor  HeartDemonType = "ignor"  // 痴：执着不悟，需破解执念
	DemonSlay   HeartDemonType = "slay"   // 杀：直接战斗心魔
)

// TowerFloorConfig 心魔塔单层配置。
type TowerFloorConfig struct {
	Floor       int             `json:"floor"`        // 层数（1-100）
	DemonType   HeartDemonType  `json:"demon_type"`   // 心魔类型
	IsBoss      bool            `json:"is_boss"`       // 是否为Boss层（每5层）
	Name        string          `json:"name"`          // 心魔名称
	Description string          `json:"description"`   // 心魔描述

	// 战斗属性（杀心魔使用）
	MonsterHP   int64   `json:"monster_hp"`   // 心魔气血
	MonsterAtk  int64   `json:"monster_atk"`  // 心魔攻击
	MonsterDef  int64   `json:"monster_def"`  // 心魔防御
	MonsterSpeed int64  `json:"monster_speed"` // 心魔速度

	// 贪心魔：需要扣除灵石
	GreedCost int64 `json:"greed_cost,omitempty"` // 贪心魔需扣除灵石数

	// 嗔心魔：激怒倍率
	WrathMultiplier float64 `json:"wrath_multiplier,omitempty"` // 嗔心魔攻击倍率

	// 痴心魔：需回答的执念问题
	IgnorQuestion string   `json:"ignor_question,omitempty"`  // 执念问题
	IgnorAnswer   string   `json:"ignor_answer,omitempty"`    // 正确答案（前端比对用哈希）
	IgnorChoices  []string `json:"ignor_choices,omitempty"`   // 选项列表

	// 奖励
	RewardExp       int64  `json:"reward_exp"`         // 修为奖励
	RewardMoney     int64  `json:"reward_money"`       // 灵石奖励
	RewardItems     []int  `json:"reward_items"`       // 物品ID列表（随机掉落）
	RewardTitle     string `json:"reward_title"`       // 通关称号（里程碑层）
}

// TowerConfig 心魔塔整体配置。
type TowerConfig struct {
	TotalFloors    int                          `json:"total_floors"`     // 总层数 100
	RealmRequired  uint32                       `json:"realm_required"`   // 需要境界ID（金丹期）
	TimeLimitSec   int                          `json:"time_limit_sec"`   // 限时秒数（180秒=3分钟）
	DailyFree      int                          `json:"daily_free"`       // 每日免费次数 1
	MaxBuyTimes    int                          `json:"max_buy_times"`    // 最多购买次数 3
	BuyCost        int64                        `json:"buy_cost"`         // 每次购买所需灵石
	Floors         map[int]*TowerFloorConfig    `json:"floors"`           // floor -> 配置
	MilestoneRewards map[int]*MilestoneReward   `json:"milestone_rewards"` // 里程碑层 -> 奖励
}

// MilestoneReward 里程碑奖励（首次通关特定层数时发放）。
type MilestoneReward struct {
	Floor       int    `json:"floor"`        // 里程碑层数
	Exp         int64  `json:"exp"`          // 修为
	Money       int64  `json:"money"`        // 灵石
	Items       []int  `json:"items"`        // 物品ID
	Title       string `json:"title"`        // 称号
	Description string `json:"description"`  // 描述
}

// TowerPlayer 玩家心魔塔持久数据。
type TowerPlayer struct {
	PlayerID         uint64 `json:"player_id"`
	HighestFloor     int    `json:"highest_floor"`      // 历史最高通关层数
	BestTimeSec      int    `json:"best_time_sec"`      // 最快通关时间（秒）
	DailyFreeUsed    int    `json:"daily_free_used"`    // 今日免费已用次数
	DailyBuyUsed     int    `json:"daily_buy_used"`     // 今日购买次数
	LastDailyDate    string `json:"last_daily_date"`    // 最后记录日期 YYYY-MM-DD
	ClaimedMilestones []int `json:"claimed_milestones"` // 已领取的里程碑层数列表
	TitlesEarned     []string `json:"titles_earned"`    // 已获得的称号列表
}

// TowerSession 玩家当前爬塔会话（运行时）。
type TowerSession struct {
	PlayerID     uint64 `json:"player_id"`
	CurrentFloor int    `json:"current_floor"`   // 当前层数（从1开始）
	State        string `json:"state"`           // idle / fighting / completed
	EnteredAt    int64  `json:"entered_at"`      // 进入时间戳（秒）
	Completed    bool   `json:"completed"`       // 是否已通关全部100层
	Failed       bool   `json:"failed"`          // 是否已失败
}

// TowerFightRequest 战斗请求参数。
type TowerFightRequest struct {
	PlayerID     uint64 `json:"player_id"`
	UseItemID    uint32 `json:"use_item_id,omitempty"`   // 使用道具ID（可选）
	IgnorChoice  int    `json:"ignor_choice,omitempty"`  // 痴心魔选择的选项索引
}

// TowerFightResult 单层战斗结果。
type TowerFightResult struct {
	Floor        int             `json:"floor"`
	DemonType    HeartDemonType  `json:"demon_type"`
	IsBoss       bool            `json:"is_boss"`
	Win          bool            `json:"win"`
	GreedCost    int64           `json:"greed_cost,omitempty"`    // 贪心魔扣除灵石
	BattleResult interface{}     `json:"battle_result,omitempty"` // 杀心魔战斗详情
	RewardExp    int64           `json:"reward_exp"`
	RewardMoney  int64           `json:"reward_money"`
	RewardItems  []int           `json:"reward_items,omitempty"`
	RewardTitle  string          `json:"reward_title,omitempty"`  // 获得的称号
	TimeUsedSec  int             `json:"time_used_sec"`           // 本层耗时
}

// TowerStatus 玩家心魔塔状态响应。
type TowerStatus struct {
	InSession      bool   `json:"in_session"`
	CurrentFloor   int    `json:"current_floor"`
	HighestFloor   int    `json:"highest_floor"`
	BestTimeSec    int    `json:"best_time_sec"`
	DailyFreeUsed  int    `json:"daily_free_used"`
	DailyBuyUsed   int    `json:"daily_buy_used"`
	DailyFreeMax   int    `json:"daily_free_max"`
	DailyBuyMax    int    `json:"daily_buy_max"`
	BuyCost        int64  `json:"buy_cost"`
	TimeLimitSec   int    `json:"time_limit_sec"`
	RemainingTime  int    `json:"remaining_time,omitempty"`  // 剩余时间（秒）
	Completed      bool   `json:"completed"`
	Failed         bool   `json:"failed"`
}

// TowerRankEntry 排行榜条目。
type TowerRankEntry struct {
	Rank         int    `json:"rank"`
	PlayerID     uint64 `json:"player_id"`
	Nickname     string `json:"nickname"`
	HighestFloor int    `json:"highest_floor"`
	BestTimeSec  int    `json:"best_time_sec"`
	RealmName    string `json:"realm_name"`
}

// TowerRankingResponse 排行榜响应。
type TowerRankingResponse struct {
	Rankings []TowerRankEntry `json:"rankings"`
}
