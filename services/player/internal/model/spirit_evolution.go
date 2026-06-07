// Package model 灵根进化数据模型
package model

// SpiritQuality 灵根品质
type SpiritQuality int

const (
	SpiritQualityMisc     SpiritQuality = 0 // 杂品(灰)
	SpiritQualityHuman    SpiritQuality = 1 // 人品(白)
	SpiritQualityEarth    SpiritQuality = 2 // 地品(绿)
	SpiritQualitySky      SpiritQuality = 3 // 天品(蓝)
	SpiritQualityChaos    SpiritQuality = 4 // 混沌(紫)
	SpiritQualityHongMeng SpiritQuality = 5 // 鸿蒙(金)
)

// SpiritQualityNames 灵根品质名称
var SpiritQualityNames = map[SpiritQuality]string{
	SpiritQualityMisc:     "杂品灵根",
	SpiritQualityHuman:    "人品灵根",
	SpiritQualityEarth:    "地品灵根",
	SpiritQualitySky:      "天品灵根",
	SpiritQualityChaos:    "混沌灵根",
	SpiritQualityHongMeng: "鸿蒙灵根",
}

// SpiritQualityColors 灵根品质颜色
var SpiritQualityColors = map[SpiritQuality]string{
	SpiritQualityMisc:     "#9e9e9e",
	SpiritQualityHuman:    "#ffffff",
	SpiritQualityEarth:    "#4caf50",
	SpiritQualitySky:      "#42a5f5",
	SpiritQualityChaos:    "#ab47bc",
	SpiritQualityHongMeng: "#ffd700",
}

// SpiritSpeedBonusPerLevel 每档修炼速度加成（百分比）
const SpiritSpeedBonusPerLevel = 20

// SpiritBreakthroughBonusPerLevel 每档突破成功率加成（百分比）
const SpiritBreakthroughBonusPerLevel = 2

// SpiritEvolutionCost 进化所需灵根进化石数量
var SpiritEvolutionCost = map[SpiritQuality]int{
	SpiritQualityMisc:   1,
	SpiritQualityHuman:  2,
	SpiritQualityEarth:  3,
	SpiritQualitySky:    5,
	SpiritQualityChaos:  10,
}

// SpiritEvolutionItemID 灵根进化石物品ID
const SpiritEvolutionItemID = 301

// ==================== 进化成功率 / 降级 ====================

// EvolutionBaseSuccessRate 基础成功率 30%
const EvolutionBaseSuccessRate = 0.30

// ReincarnationSuccessBonus 每次轮回增加的成功率 5%
const ReincarnationSuccessBonus = 0.05

// DegradationChanceOnFail 进化失败时降级概率 10%
const DegradationChanceOnFail = 0.10

// EvolutionRealmRequirement 进化所需最低境界
var EvolutionRealmRequirement = map[SpiritQuality]int32{
	SpiritQualityMisc:  RealmQiRef,   // 杂品→人品: 练气
	SpiritQualityHuman: RealmBase,    // 人品→地品: 筑基
	SpiritQualityEarth: RealmGolden,  // 地品→天品: 金丹
	SpiritQualitySky:   RealmNascent, // 天品→混沌: 元婴
	SpiritQualityChaos: RealmSpirit,  // 混沌→鸿蒙: 化神
}

// ==================== 元素觉醒 ====================

// ElementDamageBonus 元素觉醒额外伤害加成 15%
const ElementDamageBonus = 0.15

// PlayerElementAwakening 玩家元素觉醒
type PlayerElementAwakening struct {
	PlayerID    int64   `json:"player_id" gorm:"primaryKey"`
	Element     int32   `json:"element"`                        // 对应 player.go SpiritRoot 枚举
	DamageBonus float64 `json:"damage_bonus" gorm:"default:0.15"` // 元素伤害加成百分比
	ActivatedAt int64   `json:"activated_at"`
}

// ==================== 进化历史 ====================

// EvolutionHistory 进化历史记录
type EvolutionHistory struct {
	ID          int64         `json:"id" gorm:"primaryKey"`
	PlayerID    int64         `json:"player_id" gorm:"index;not null"`
	FromQuality SpiritQuality `json:"from_quality"`
	ToQuality   SpiritQuality `json:"to_quality"`
	Success     bool          `json:"success"`
	Degraded    bool          `json:"degraded"`
	StonesUsed  int           `json:"stones_used"`
	RealmAtTime int32         `json:"realm_at_time"` // 进化时的境界
	SuccessRate float64       `json:"success_rate"`  // 进化时的成功率
	CreatedAt   int64         `json:"created_at"`
}

// ==================== 请求/响应结构体 ====================

// SpiritInfoResponse 灵根状态响应
type SpiritInfoResponse struct {
	PlayerID          int64   `json:"player_id"`
	Quality           int     `json:"quality"`
	QualityName       string  `json:"quality_name"`
	QualityColor      string  `json:"quality_color"`
	NextQuality       int     `json:"next_quality"`
	NextQualityName   string  `json:"next_quality_name"`
	NextQualityColor  string  `json:"next_quality_color"`
	CanEvolve         bool    `json:"can_evolve"`
	EvolutionStones   int     `json:"evolution_stones"`
	CultBonus         int     `json:"cult_bonus"`
	BreakBonus        int     `json:"break_bonus"`
	Reincarnations    int     `json:"reincarnations"`
	ReincarnationChance float64 `json:"reincarnation_chance"`
	RealmRequirement  int32   `json:"realm_requirement"`
	RealmName         string  `json:"realm_name"`
	SuccessRate       float64 `json:"success_rate"`
	ElementAwakened   bool    `json:"element_awakened"`
	ElementName       string  `json:"element_name,omitempty"`
	ElementDamageBonus float64 `json:"element_damage_bonus,omitempty"`
}

// EvolutionResult 进化结果（供 handler 使用）
type EvolutionResult struct {
	Spirit           *PlayerSpirit          `json:"spirit"`
	Success          bool                   `json:"success"`
	Degraded         bool                   `json:"degraded"`
	SuccessRate      float64                `json:"success_rate"`
	StonesUsed       int                    `json:"stones_used"`
	FromQuality      SpiritQuality          `json:"from_quality"`
	ToQuality        SpiritQuality          `json:"to_quality"`
	ElementAwakening *PlayerElementAwakening `json:"element_awakening,omitempty"`
}

// StoneTierInfo 每档进化石需求信息
type StoneTierInfo struct {
	Quality     int    `json:"quality"`
	QualityName string `json:"quality_name"`
	QualityColor string `json:"quality_color"`
	Stones      int    `json:"stones"`
	RealmReq    int32  `json:"realm_requirement"`
	RealmName   string `json:"realm_name"`
}

// StoneInfoResponse 进化石信息响应
type StoneInfoResponse struct {
	ItemID              int64           `json:"item_id"`
	ItemName            string          `json:"item_name"`
	BaseSuccessRate     float64         `json:"base_success_rate"`
	ReincarnationBonus  float64         `json:"reincarnation_bonus"`
	DegradationChance   float64         `json:"degradation_chance"`
	TierRequirements    []StoneTierInfo `json:"tier_requirements"`
}

// EvolveRequest 进化请求
type EvolveRequest struct {
	PlayerID int64 `json:"player_id" binding:"required"`
}

// HistoryQuery 历史查询参数
type HistoryQuery struct {
	PlayerID int64 `form:"player_id" binding:"required"`
	Page     int   `form:"page" default:"1"`
	PageSize int   `form:"page_size" default:"20"`
}

// ==================== 原有核心方法 ====================

// PlayerSpirit 玩家灵根状态
type PlayerSpirit struct {
	PlayerID      int64         `json:"player_id" gorm:"primaryKey"`
	Quality       SpiritQuality `json:"quality"`        // 当前品质
	Reincarnations int          `json:"reincarnations"` // 轮回次数（自动提升概率）
	UpdatedAt     int64         `json:"updated_at"`
}

// GetEvolutionTarget 获取下一品质的目标
func (s *PlayerSpirit) GetEvolutionTarget() SpiritQuality {
	if s.Quality >= SpiritQualityHongMeng {
		return SpiritQualityHongMeng // 已满
	}
	return s.Quality + 1
}

// ReincarnationUpgradeChance 轮回自动提升概率（每轮回一次+5%）
func (s *PlayerSpirit) ReincarnationUpgradeChance() float64 {
	base := 0.05 // 基础5%
	return base + float64(s.Reincarnations)*0.05
}
