package model

import "time"

// PlayerEnergy 玩家体力/能量（修炼打坐恢复版）
type PlayerEnergy struct {
	ID                   int64      `json:"id" gorm:"primaryKey"`
	PlayerID             int64      `json:"player_id" gorm:"uniqueIndex;not null"`
	CurrentEnergy        int        `json:"current_energy" gorm:"default:100"`
	MaxEnergy            int        `json:"max_energy" gorm:"default:100"`
	LastMeditationAt     *time.Time `json:"last_meditation_at"`
	EnergyPillsUsedToday int        `json:"energy_pills_used_today" gorm:"default:0"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// MeditationConfig 修炼打坐恢复配置
type MeditationConfig struct {
	BaseRegenPerMinute        int     `json:"base_regen_per_minute"`
	RealmMultiplierBase       float64 `json:"realm_multiplier_base"`
	RealmMultiplierPerLevel   float64 `json:"realm_multiplier_per_level"`
	TechniqueBonusMultiplier  float64 `json:"technique_bonus_multiplier"`
	BodyCultivationBonus      float64 `json:"body_cultivation_bonus_per_level"`
}

// PillTier 体力丹药品阶配置
type PillTier struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Tier          int    `json:"tier"`
	RestoreAmount int    `json:"restore_amount"`
	RealmRequired int    `json:"realm_required"`
	Description   string `json:"description"`
}

// PillUsageConfig 体力丹药使用限制
type PillUsageConfig struct {
	MaxPillsPerDay  int `json:"max_pills_per_day"`
	CooldownSeconds int `json:"cooldown_seconds"`
}

// EnergyCostConfig 能量消耗/恢复配置（从JSON加载）
type EnergyCostConfig struct {
	MeditationConfig           MeditationConfig        `json:"meditation_config"`
	RealmMaxEnergy             map[int]int             `json:"realm_max_energy"`
	BodyCultivationMaxEnergy   map[int]int             `json:"body_cultivation_max_energy_bonus"`
	ActionCosts                map[string]int          `json:"action_costs"`
	PillTiers                  []*PillTier             `json:"pill_tiers"`
	PillUsageConfig            PillUsageConfig         `json:"pill_usage_config"`
	MeditationDurationMinutes  int                     `json:"meditation_duration_minutes"`
}

// EnergyStatus 能量状态响应
type EnergyStatus struct {
	CurrentEnergy      int     `json:"current_energy"`
	MaxEnergy          int     `json:"max_energy"`
	MeditationRegenMin int     `json:"meditation_regen_per_min"`
	RegenPerHour       int     `json:"regen_per_hour"`
	HoursToFull        float64 `json:"hours_to_full"`
	PillsUsed          int     `json:"pills_used"`
	MaxPills           int     `json:"max_pills"`
	PillCooldownLeft   int     `json:"pill_cooldown_left,omitempty"`
	TechniqueBonus     float64 `json:"technique_bonus"`
	LastMeditationAt   *int64  `json:"last_meditation_at,omitempty"`
}

// UseEnergyPillRequest 使用体力丹药请求
type UseEnergyPillRequest struct {
	PlayerID int64 `json:"player_id" binding:"required"`
	PillID   int   `json:"pill_id" binding:"required"`
	Quantity int   `json:"quantity" binding:"required,min=1"`
}

// ConsumeEnergyRequest 消耗体力请求
type ConsumeEnergyRequest struct {
	PlayerID   int64  `json:"player_id" binding:"required"`
	ActionType string `json:"action_type" binding:"required"`
}

// MeditateRequest 修炼打坐请求
type MeditateRequest struct {
	PlayerID     int64 `json:"player_id" binding:"required"`
	DurationMin  int   `json:"duration_min" binding:"required,min=1"`
}

// MeditateResponse 修炼打坐响应
type MeditateResponse struct {
	EnergyGained    int   `json:"energy_gained"`
	CurrentEnergy   int   `json:"current_energy"`
	MaxEnergy       int   `json:"max_energy"`
	MeditationMin   int   `json:"meditation_minutes"`
	RegenPerMin     int   `json:"regen_per_min"`
}

// PillRecoveryResponse 丹药回复响应
type PillRecoveryResponse struct {
	PillName      string `json:"pill_name"`
	PillTier      int    `json:"pill_tier"`
	EnergyGained  int    `json:"energy_gained"`
	CurrentEnergy int    `json:"current_energy"`
	MaxEnergy     int    `json:"max_energy"`
	PillsUsed     int    `json:"pills_used"`
	MaxPills      int    `json:"max_pills"`
}
