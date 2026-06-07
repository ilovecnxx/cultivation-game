// Package model defines daolv-specific data types
package model

import "time"

// ============================================================
// 道侣等级系统
// ============================================================

const (
	DaolvLevelChuShi   = "初识"  // 0-99
	DaolvLevelZhiJi    = "知己"  // 100-499
	DaolvLevelQingShen = "情深"  // 500-999
	DaolvLevelTongXin  = "同心"  // 1000-2999
	DaolvLevelXianLv   = "仙侣"  // 3000+
)

// DaolvLevelInfo 道侣等级信息
type DaolvLevelInfo struct {
	Name       string `json:"name"`
	MinIntimacy int64 `json:"min_intimacy"`
	MaxIntimacy int64 `json:"max_intimacy,omitempty"` // 0 表示最高等级无上限
	UnlockSkill string `json:"unlock_skill,omitempty"`
}

// GetDaolvLevels 返回所有道侣等级定义
func GetDaolvLevels() []DaolvLevelInfo {
	return []DaolvLevelInfo{
		{Name: DaolvLevelChuShi, MinIntimacy: 0, MaxIntimacy: 99},
		{Name: DaolvLevelZhiJi, MinIntimacy: 100, MaxIntimacy: 499, UnlockSkill: "传送"},
		{Name: DaolvLevelQingShen, MinIntimacy: 500, MaxIntimacy: 999, UnlockSkill: "属性共享"},
		{Name: DaolvLevelTongXin, MinIntimacy: 1000, MaxIntimacy: 2999, UnlockSkill: "复活"},
		{Name: DaolvLevelXianLv, MinIntimacy: 3000, MaxIntimacy: 0, UnlockSkill: "心有灵犀"},
	}
}

// GetDaolvLevel 根据亲密度获取等级
func GetDaolvLevel(intimacy int) string {
	switch {
	case intimacy >= 3000:
		return DaolvLevelXianLv
	case intimacy >= 1000:
		return DaolvLevelTongXin
	case intimacy >= 500:
		return DaolvLevelQingShen
	case intimacy >= 100:
		return DaolvLevelZhiJi
	default:
		return DaolvLevelChuShi
	}
}

// GetDaolvMaxIntimacy 获取当前等级最高亲密度
func GetDaolvMaxIntimacy(intimacy int) int64 {
	for _, l := range GetDaolvLevels() {
		if l.MaxIntimacy == 0 {
			return 0 // 无上限
		}
		if intimacy >= int(l.MinIntimacy) && intimacy < int(l.MaxIntimacy) {
			return l.MaxIntimacy
		}
	}
	return 99
}

// ============================================================
// 道侣任务
// ============================================================

// DaolvTaskType 任务类型
type DaolvTaskType string

const (
	TaskDualCultivate DaolvTaskType = "dual_cultivate" // 双修
	TaskSendGift      DaolvTaskType = "send_gift"      // 赠送礼物
	TaskAdventure     DaolvTaskType = "adventure"      // 共同冒险
	TaskBoss          DaolvTaskType = "boss"           // 世界BOSS
	TaskDungeon       DaolvTaskType = "dungeon"        // 秘境副本
)

// DaolvTask 道侣任务
type DaolvTask struct {
	ID          string       `bson:"_id" json:"id"`
	RelationID  string       `bson:"relation_id" json:"relation_id"`
	Type        DaolvTaskType `bson:"type" json:"type"`
	Description string       `bson:"description" json:"description"`
	Target      int64        `bson:"target" json:"target"`
	Progress    int64        `bson:"progress" json:"progress"`
	Completed   bool         `bson:"completed" json:"completed"`
	Claimed     bool         `bson:"claimed" json:"claimed"`
	Period      string       `bson:"period" json:"period"` // "daily" / "weekly"
	Date        string       `bson:"date" json:"date"`     // 日期 yyyy-mm-dd
	Reward      *DaolvReward `bson:"reward" json:"reward"`
	CreatedAt   time.Time    `bson:"created_at" json:"created_at"`
}

// DaolvReward 任务奖励
type DaolvReward struct {
	Intimacy int64        `json:"intimacy"`
	Items    []ItemReward `json:"items,omitempty"`
}

// ItemReward 物品奖励
type ItemReward struct {
	ItemID   string `json:"item_id"`
	ItemName string `json:"item_name"`
	Quantity int64  `json:"quantity"`
}
