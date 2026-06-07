package model

import "time"

// AchievementCategory 成就分类
const (
	AchievementCatCultivation = "cultivation"
	AchievementCatCombat      = "combat"
	AchievementCatExplore     = "explore"
	AchievementCatSocial      = "social"
	AchievementCatWealth      = "wealth"
)

// Achievement 成就定义（静态配置）
type Achievement struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"` // cultivation/combat/explore/social/wealth
	Target      int               `json:"target"`   // 需要达到的值
	Reward      AchievementReward `json:"reward"`
}

// AchievementReward 成就奖励
type AchievementReward struct {
	Title     string  `json:"title"`      // 称号（空字符串表示无称号）
	Exp       int64   `json:"exp"`        // 经验奖励
	Money     int64   `json:"money"`      // 灵石奖励
	AttrBonus float64 `json:"attr_bonus"` // 属性加成（百分比，0.01=1%）
}

// PlayerAchievement 玩家成就进度
type PlayerAchievement struct {
	PlayerID      uint64    `json:"player_id" gorm:"primaryKey"`
	AchievementID int       `json:"achievement_id" gorm:"primaryKey"`
	Progress      int       `json:"progress" gorm:"default:0"`
	Completed     bool      `json:"completed" gorm:"default:false"`
	CompletedAt   time.Time `json:"completed_at"`
	Claimed       bool      `json:"claimed" gorm:"default:false"`
}

// PlayerTitle 玩家当前称号
type PlayerTitle struct {
	PlayerID  uint64    `json:"player_id" gorm:"primaryKey"`
	Title     string    `json:"title" gorm:"size:32;default:''"`
	UpdatedAt time.Time `json:"updated_at"`
}

// -------- 请求/响应结构体 --------

// AchievementProgress 成就进度响应
type AchievementProgress struct {
	Achievement Achievement `json:"achievement"`
	Progress    int         `json:"progress"`
	Completed   bool        `json:"completed"`
	Claimed     bool        `json:"claimed"`
}

// ClaimAchievementRequest 领取成就奖励请求
type ClaimAchievementRequest struct {
	AchievementID int `json:"achievement_id" binding:"required"`
}

// ClaimResult 领取奖励结果
type ClaimResult struct {
	AchievementID int     `json:"achievement_id"`
	Name          string  `json:"name"`
	Title         string  `json:"title,omitempty"`
	Exp           int64   `json:"exp"`
	Money         int64   `json:"money"`
	AttrBonus     float64 `json:"attr_bonus,omitempty"`
}

// TitleResponse 当前称号响应
type TitleResponse struct {
	PlayerID uint64 `json:"player_id"`
	Title    string `json:"title"`
}
