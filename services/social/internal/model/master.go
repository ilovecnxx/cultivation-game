// Package model 定义师徒系统数据模型
package model

import "time"

// ============================================================
// 师徒关系状态常量
// ============================================================

// MasterRelationStatus 师徒关系状态
type MasterRelationStatus string

const (
	MasterStatusActive    MasterRelationStatus = "active"    // 正常
	MasterStatusGraduated MasterRelationStatus = "graduated" // 已出师
	MasterStatusKicked    MasterRelationStatus = "kicked"    // 被逐出师门
	MasterStatusBetrayed  MasterRelationStatus = "betrayed"  // 徒弟叛离
)

// MasterApplyType 申请发起方类型
type MasterApplyType string

const (
	ApplyAsStudent MasterApplyType = "as_student" // 徒弟申请拜师
	ApplyAsMaster  MasterApplyType = "as_master"  // 师父申请收徒
)

// MasterMissionType 每日师徒任务类型
type MasterMissionType string

const (
	MasterMissionCultivate MasterMissionType = "cultivate" // 共同修炼30分钟
	MasterMissionCombat    MasterMissionType = "combat"    // 师父帮徒弟击败怪物
	MasterMissionTribute   MasterMissionType = "tribute"   // 徒弟向师父进贡
	MasterMissionDungeon   MasterMissionType = "dungeon"   // 通关师徒副本
)

// ============================================================
// 师徒等级(关系亲密度)
// ============================================================

// MentorshipLevel 师徒等级
type MentorshipLevel int

const (
	MentorshipLevelRegistered  MentorshipLevel = 1 // 记名弟子
	MentorshipLevelApprentice  MentorshipLevel = 2 // 入门弟子
	MentorshipLevelCore        MentorshipLevel = 3 // 亲传弟子
	MentorshipLevelSuccessor   MentorshipLevel = 4 // 衣钵传人
)

// MentorshipLevelNames 师徒等级中文名
var MentorshipLevelNames = map[MentorshipLevel]string{
	MentorshipLevelRegistered: "记名弟子",
	MentorshipLevelApprentice: "入门弟子",
	MentorshipLevelCore:       "亲传弟子",
	MentorshipLevelSuccessor:  "衣钵传人",
}

// MentorshipLevelBenefits 各师徒等级加成
type MentorshipLevelBenefit struct {
	ExpBonusPct        float64 `json:"exp_bonus_pct"`         // 修炼经验加成
	CombatBonusPct     float64 `json:"combat_bonus_pct"`      // 战斗加成
	TeachDiscountPct   float64 `json:"teach_discount_pct"`    // 传授功法折扣
	DailyRewardMVPct   float64 `json:"daily_reward_mv_pct"`   // 每日奖励师徒值加成
	GraduationBonusPct float64 `json:"graduation_bonus_pct"`  // 出师奖励加成
	MvRequired         int64   `json:"mv_required"`           // 升级所需师徒值
}

// MentorshipLevelBenefitsMap 各师徒等级对应加成
var MentorshipLevelBenefitsMap = map[MentorshipLevel]MentorshipLevelBenefit{
	MentorshipLevelRegistered: {
		ExpBonusPct:        5,
		CombatBonusPct:     0,
		TeachDiscountPct:   0,
		DailyRewardMVPct:   0,
		GraduationBonusPct: 0,
		MvRequired:         0,
	},
	MentorshipLevelApprentice: {
		ExpBonusPct:        10,
		CombatBonusPct:     5,
		TeachDiscountPct:   10,
		DailyRewardMVPct:   10,
		GraduationBonusPct: 5,
		MvRequired:         500,
	},
	MentorshipLevelCore: {
		ExpBonusPct:        20,
		CombatBonusPct:     10,
		TeachDiscountPct:   20,
		DailyRewardMVPct:   20,
		GraduationBonusPct: 15,
		MvRequired:         2000,
	},
	MentorshipLevelSuccessor: {
		ExpBonusPct:        35,
		CombatBonusPct:     20,
		TeachDiscountPct:   30,
		DailyRewardMVPct:   30,
		GraduationBonusPct: 30,
		MvRequired:         5000,
	},
}

// ============================================================
// 师徒关系
// ============================================================

// MasterRelation 师徒关系
type MasterRelation struct {
	ID               string               `bson:"_id" json:"id"`
	MasterID         string               `bson:"master_id" json:"master_id"`
	MasterName       string               `bson:"master_name" json:"master_name"`
	StudentID        string               `bson:"student_id" json:"student_id"`
	StudentName      string               `bson:"student_name" json:"student_name"`
	MasterValue      int64                `bson:"master_value" json:"master_value"`             // 师徒值
	MentorshipLevel  MentorshipLevel      `bson:"mentorship_level" json:"mentorship_level"`     // 师徒等级 1-4
	Status           MasterRelationStatus `bson:"status" json:"status"`
	DailyTrainingID  string               `bson:"daily_training_id,omitempty" json:"daily_training_id,omitempty"` // 今日训练任务ID
	TrainingProgress int32                `bson:"training_progress" json:"training_progress"`   // 训练任务进度
	TrainingTarget   int32                `bson:"training_target" json:"training_target"`       // 训练任务目标
	TrainingDate     string               `bson:"training_date,omitempty" json:"training_date,omitempty"`         // 训练日期
	CreatedAt        time.Time            `bson:"created_at" json:"created_at"`
	GraduatedAt      time.Time            `bson:"graduated_at,omitempty" json:"graduated_at,omitempty"`
}

// ============================================================
// 师徒申请
// ============================================================

// MasterApply 师徒申请
type MasterApply struct {
	ID        string          `bson:"_id" json:"id"`
	FromID    string          `bson:"from_id" json:"from_id"`
	FromName  string          `bson:"from_name" json:"from_name"`
	ToID      string          `bson:"to_id" json:"to_id"`
	ToName    string          `bson:"to_name" json:"to_name"`
	ApplyType MasterApplyType `bson:"apply_type" json:"apply_type"`
	Message   string          `bson:"message,omitempty" json:"message,omitempty"`
	Status    string          `bson:"status" json:"status"` // pending / accepted / rejected
	CreatedAt time.Time       `bson:"created_at" json:"created_at"`
	HandledAt time.Time       `bson:"handled_at,omitempty" json:"handled_at,omitempty"`
}

// ============================================================
// 每日师徒任务
// ============================================================

// MasterMission 师徒每日任务
type MasterMission struct {
	ID          string             `bson:"_id" json:"id"`
	RelationID  string             `bson:"relation_id" json:"relation_id"`
	MissionType MasterMissionType  `bson:"mission_type" json:"mission_type"`
	Required    int32              `bson:"required" json:"required"`
	Progress    int32              `bson:"progress" json:"progress"`
	Completed   bool               `bson:"completed" json:"completed"`
	Claimed     bool               `bson:"claimed" json:"claimed"`
	Date        string             `bson:"date" json:"date"` // yyyy-mm-dd
	RewardMV    int64              `bson:"reward_mv" json:"reward_mv"` // 奖励师徒值
}

// ============================================================
// 传授功法记录
// ============================================================

// MasterTeachRecord 传授功法记录
type MasterTeachRecord struct {
	ID          string    `bson:"_id" json:"id"`
	RelationID  string    `bson:"relation_id" json:"relation_id"`
	MasterID    string    `bson:"master_id" json:"master_id"`
	StudentID   string    `bson:"student_id" json:"student_id"`
	SkillID     string    `bson:"skill_id" json:"skill_id"`
	SkillName   string    `bson:"skill_name" json:"skill_name"`
	CostMV      int64     `bson:"cost_mv" json:"cost_mv"`
	ActualCostMV int64    `bson:"actual_cost_mv" json:"actual_cost_mv"` // 折扣后实际消耗
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}

// ============================================================
// 徒弟突破奖励记录
// ============================================================

// MasterBreakthroughReward 徒弟突破时师父获得的修为奖励记录
type MasterBreakthroughReward struct {
	ID           string    `bson:"_id" json:"id"`
	MasterID     string    `bson:"master_id" json:"master_id"`
	StudentID    string    `bson:"student_id" json:"student_id"`
	StudentRealm string    `bson:"student_realm" json:"student_realm"` // 徒弟突破到的境界
	MasterExp    int64     `bson:"master_exp" json:"master_exp"`       // 师父获得的修为值
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
}

// ============================================================
// 出师奖励定义
// ============================================================

// GraduateReward 出师奖励
type GraduateReward struct {
	MasterExp   int64 `json:"master_exp"`  // 师父获取修为
	StudentExp  int64 `json:"student_exp"` // 徒弟获取修为
	MasterMV    int64 `json:"master_mv"`   // 师父获取师徒值
	StudentMV   int64 `json:"student_mv"`  // 徒弟获取师徒值
	MasterItems []ItemReward `json:"master_items,omitempty"`  // 师父额外物品
	StudentItems []ItemReward `json:"student_items,omitempty"` // 徒弟额外物品
}

// MasterDungeonInstance 师徒副本实例
type MasterDungeonInstance struct {
	ID                  string    `bson:"_id" json:"id"`
	RelationID          string    `bson:"relation_id" json:"relation_id"`
	MasterID            string    `bson:"master_id" json:"master_id"`
	StudentID           string    `bson:"student_id" json:"student_id"`
	DungeonLevel        int       `bson:"dungeon_level" json:"dungeon_level"`               // 副本层数
	MasterHP            int64     `bson:"master_hp" json:"master_hp"`                       // 师父当前血量
	MasterMaxHP         int64     `bson:"master_max_hp" json:"master_max_hp"`               // 师父最大血量
	StudentHP           int64     `bson:"student_hp" json:"student_hp"`                     // 徒弟当前血量
	StudentMaxHP        int64     `bson:"student_max_hp" json:"student_max_hp"`             // 徒弟最大血量
	CurrentWave         int       `bson:"current_wave" json:"current_wave"`                 // 当前波次
	MaxWave             int       `bson:"max_wave" json:"max_wave"`                         // 总波次
	TotalMasterDmg      int64     `bson:"total_master_dmg" json:"total_master_dmg"`         // 师父总伤害
	TotalStudentDmg     int64     `bson:"total_student_dmg" json:"total_student_dmg"`       // 徒弟总伤害
	Status              string    `bson:"status" json:"status"`                             // pending / active / completed / failed
	RewardClaimed       bool      `bson:"reward_claimed" json:"reward_claimed"`             // 奖励是否已领取
	CreatedAt           time.Time `bson:"created_at" json:"created_at"`
	CompletedAt         time.Time `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
}

// MasterDungeonReward 师徒副本奖励
type MasterDungeonReward struct {
	MasterMV      int64       `json:"master_mv"`      // 师父师徒值
	StudentMV     int64       `json:"student_mv"`     // 徒弟师徒值
	MasterExp     int64       `json:"master_exp"`     // 师父修为
	StudentExp    int64       `json:"student_exp"`    // 徒弟修为
	Items         []ItemReward `json:"items,omitempty"` // 共同掉落
}

// ============================================================
// 叛离记录
// ============================================================

// MasterBetrayalRecord 叛离记录
type MasterBetrayalRecord struct {
	ID             string    `bson:"_id" json:"id"`
	RelationID     string    `bson:"relation_id" json:"relation_id"`
	MasterID       string    `bson:"master_id" json:"master_id"`
	MasterName     string    `bson:"master_name" json:"master_name"`
	StudentID      string    `bson:"student_id" json:"student_id"`
	StudentName    string    `bson:"student_name" json:"student_name"`
	MasterValueLost int64    `bson:"master_value_lost" json:"master_value_lost"`    // 损失的师徒值
	MentorshipLevelLost MentorshipLevel `bson:"mentorship_level_lost" json:"mentorship_level_lost"` // 损失的师徒等级
	PenaltyExp     int64     `bson:"penalty_exp" json:"penalty_exp"`                // 惩罚扣除修为
	BetrayedAt     time.Time `bson:"betrayed_at" json:"betrayed_at"`
}

// ============================================================
// 训练任务
// ============================================================

// TrainingTaskType 训练任务类型
type TrainingTaskType string

const (
	TrainingTaskCultivate   TrainingTaskType = "cultivate"   // 修炼
	TrainingTaskCombat      TrainingTaskType = "combat"      // 战斗
	TrainingTaskAlchemy     TrainingTaskType = "alchemy"     // 炼丹
	TrainingTaskExplore     TrainingTaskType = "explore"     // 探索
	TrainingTaskDungeon     TrainingTaskType = "dungeon"     // 副本
)

// DailyTraining 每日训练任务
type DailyTraining struct {
	ID            string           `bson:"_id" json:"id"`
	RelationID    string           `bson:"relation_id" json:"relation_id"`
	TaskType      TrainingTaskType `bson:"task_type" json:"task_type"`
	Description   string           `bson:"description" json:"description"`
	Target        int32            `bson:"target" json:"target"`
	Progress      int32            `bson:"progress" json:"progress"`
	RewardMV      int64            `bson:"reward_mv" json:"reward_mv"`
	RewardExp     int64            `bson:"reward_exp" json:"reward_exp"`
	Completed     bool             `bson:"completed" json:"completed"`
	MasterClaimed bool             `bson:"master_claimed" json:"master_claimed"`
	StudentClaimed bool            `bson:"student_claimed" json:"student_claimed"`
	Date          string           `bson:"date" json:"date"`
	CreatedAt     time.Time        `bson:"created_at" json:"created_at"`
}
