package model

import "time"

// ============================================================
// 新手保护系统模型
// ============================================================

// PlayerProtection 玩家保护记录
type PlayerProtection struct {
	ID                       int64      `json:"id" gorm:"primaryKey"`
	PlayerID                 int64      `json:"player_id" gorm:"uniqueIndex;size:64;not null"`
	ProtectionUntil          *time.Time `json:"protection_until"`           // 通用保护截止时间（nil 表示已过期或不存在）
	PvpProtectionUntil       *time.Time `json:"pvp_protection_until"`       // PVP 保护截止时间（nil 表示已过期或不存在）
	BreakthroughGraceCount   int32      `json:"breakthrough_grace_count" gorm:"default:3"`    // 突破免罚剩余次数
	FreeResurrectionCount    int32      `json:"free_resurrection_count" gorm:"default:5"`     // 免费复活剩余次数
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// ProtectionStatus 保护状态响应
type ProtectionStatus struct {
	Protected                bool   `json:"protected"`                   // 是否处于通用保护状态
	PvpProtected             bool   `json:"pvp_protected"`               // 是否处于 PVP 保护状态
	ProtectionRemainingSec   int64  `json:"protection_remaining_sec"`    // 通用保护剩余秒数
	PvpProtectionRemainingSec int64 `json:"pvp_protection_remaining_sec"` // PVP 保护剩余秒数
	BreakthroughGraceRemaining int32 `json:"breakthrough_grace_remaining"` // 突破免罚剩余次数
	FreeResurrectionRemaining int32 `json:"free_resurrection_remaining"`   // 免费复活剩余次数
}

// ProtectionConfig 保护配置（从 JSON 加载）
type ProtectionConfig struct {
	NewbieProtectionHours        int     `json:"newbie_protection_hours"`
	PvpProtectionHours           int     `json:"pvp_protection_hours"`
	ProtectionRealmMax           int     `json:"protection_realm_max"`
	BreakthroughGraceCount       int     `json:"breakthrough_grace_count"`
	BreakthroughGracePenaltyReduction float64 `json:"breakthrough_grace_penalty_reduction"`
	FreeResurrectionCount        int     `json:"free_resurrection_count"`
	LevelGapMax                  int     `json:"level_gap_max"`
}
