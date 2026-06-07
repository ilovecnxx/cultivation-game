// Package model 定义世界服务的数据模型
package model

import "time"

// ============================================================
// 任务系统模型
// ============================================================

// QuestType 任务类型
type QuestType string

const (
	QuestMain  QuestType = "main"  // 主线任务
	QuestSide  QuestType = "side"  // 支线任务
	QuestDaily QuestType = "daily" // 每日任务
)

// QuestStatus 任务状态
type QuestStatus string

const (
	QuestNotAccepted QuestStatus = "not_accepted" // 未接取
	QuestInProgress  QuestStatus = "in_progress"  // 进行中
	QuestCompleted   QuestStatus = "completed"    // 条件已满足，待提交
	QuestSubmitted   QuestStatus = "submitted"    // 已提交，领取奖励完毕
)

// QuestRequirement 任务需求
type QuestRequirement struct {
	Type     string `json:"type"`      // kill_monster / gather_item / reach_realm / cultivate_time / craft_pill / talk_to_npc / explore_region / arena_win
	TargetID string `json:"target_id"` // 目标ID(怪物ID/物品ID/区域ID等)
	Count    int    `json:"count"`     // 需求数量
	Current  int    `json:"current"`   // 当前进度
}

// QuestReward 任务奖励
type QuestReward struct {
	Type     string `json:"type"`     // exp / money / item / skill / reputation
	ID       string `json:"id,omitempty"`       // 物品/技能ID
	Quantity int64  `json:"quantity"`           // 数量
}

// Quest 任务配置
type Quest struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          QuestType         `json:"type"`
	Description   string            `json:"description"`
	Requirements  []QuestRequirement `json:"requirements"`
	Rewards       []QuestReward     `json:"rewards"`
	Prerequisites []string          `json:"prerequisites"`   // 前置任务ID列表
	RealmRequired string            `json:"realm_required"`  // 所需境界描述
	LevelRequired int               `json:"level_required"`  // 所需等级
	NpcID         string            `json:"npc_id"`          // 发布/提交NPC
	DialogueStart string            `json:"dialogue_start"`  // 接任务时的对话
	DialogueEnd   string            `json:"dialogue_end"`    // 提交任务时的对话
}

// PlayerQuest 玩家任务进度
type PlayerQuest struct {
	PlayerID    string             `json:"player_id"`
	QuestID     string             `json:"quest_id"`
	Status      QuestStatus        `json:"status"`
	Progress    []QuestRequirement `json:"progress"`     // 当前进度(每个需求的当前值)
	AcceptedAt  time.Time          `json:"accepted_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
}

// QuestEvent 任务进度更新事件
// 由其他服务(战斗/采集/修炼等)触发，驱动任务进度更新
type QuestEvent struct {
	Type     string `json:"type"`      // kill_monster / gather_item / reach_realm / cultivate_time / craft_pill / talk_to_npc / explore_region / arena_win
	TargetID string `json:"target_id"` // 目标ID
	Count    int    `json:"count"`     // 数量(默认1)
}

// ============================================================
// 每日任务系统模型
// ============================================================

// DailyTaskStatus 每日任务状态
type DailyTaskStatus int

const (
	DailyTaskInProgress DailyTaskStatus = 0 // 进行中
	DailyTaskCompleted  DailyTaskStatus = 1 // 已完成
	DailyTaskClaimed    DailyTaskStatus = 2 // 已领取
)

// DailyTaskCategory 每日任务分类
type DailyTaskCategory string

const (
	DailyTaskCultivation DailyTaskCategory = "cultivation" // 修炼任务
	DailyTaskCombat      DailyTaskCategory = "combat"      // 战斗任务
	DailyTaskSocial      DailyTaskCategory = "social"      // 社交任务
	DailyTaskEconomy     DailyTaskCategory = "economy"     // 经济任务
)

// DailyTaskDef 每日任务定义(从JSON加载)
type DailyTaskDef struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Category        DailyTaskCategory `json:"category"`
	Type            string            `json:"type"`
	TargetID        string            `json:"target_id"`
	RequiredCount   int               `json:"required_count"`
	ActivityPoints  int               `json:"activity_points"`
	Rewards         []QuestReward     `json:"rewards"`
	SortOrder       int               `json:"sort_order"`
}

// DailyTaskProgress 玩家每日任务进度
type DailyTaskProgress struct {
	PlayerID      string           `json:"player_id"`
	TaskDate      string           `json:"task_date"`      // "2006-01-02"
	TaskID        string           `json:"task_id"`
	TaskType      string           `json:"task_type"`
	CurrentCount  int              `json:"current_count"`
	RequiredCount int              `json:"required_count"`
	Status        DailyTaskStatus  `json:"status"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
	ClaimedAt     *time.Time       `json:"claimed_at,omitempty"`
}

// DailyTaskWithProgress 带进度信息的每日任务
type DailyTaskWithProgress struct {
	Def      *DailyTaskDef    `json:"def"`
	Progress *DailyTaskProgress `json:"progress,omitempty"`
}

// ActivityPoints 每日活跃度积分信息
type ActivityPoints struct {
	PlayerID        string `json:"player_id"`
	Date            string `json:"date"`
	TotalPoints     int    `json:"total_points"`
	Chest25Claimed  bool   `json:"chest_25_claimed"`
	Chest50Claimed  bool   `json:"chest_50_claimed"`
	Chest75Claimed  bool   `json:"chest_75_claimed"`
	Chest100Claimed bool   `json:"chest_100_claimed"`
}

// ChestTier 活跃度宝箱档位
type ChestTier int

const (
	ChestTier25  ChestTier = 25
	ChestTier50  ChestTier = 50
	ChestTier75  ChestTier = 75
	ChestTier100 ChestTier = 100
)

// ChestReward 宝箱奖励定义
type ChestReward struct {
	Tier        int          `json:"tier"`
	PointsRequired int       `json:"points_required"`
	Rewards     []QuestReward `json:"rewards"`
}
