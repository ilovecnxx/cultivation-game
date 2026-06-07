// Package model 炼丹系统数据结构定义
package model

import "fmt"

// Quality 丹药品质枚举
type Quality int

const (
	QualityJunk     Quality = 0 // 废丹
	QualityMortal   Quality = 1 // 凡品
	QualityGood     Quality = 2 // 良品
	QualitySuperior Quality = 3 // 上品
	QualityPremium  Quality = 4 // 极品
	QualityImmortal Quality = 5 // 仙品

	// Deprecated aliases（保留编译兼容）
	QualityExcellent Quality = 3 // 旧版"极品"→新版对应"上品"
	QualitySupreme   Quality = 4 // 旧版"绝品"→新版对应"极品"
)

// QualityName 品质中文名映射
var QualityNames = map[Quality]string{
	QualityJunk:     "废丹",
	QualityMortal:   "凡品",
	QualityGood:     "良品",
	QualitySuperior: "上品",
	QualityPremium:  "极品",
	QualityImmortal: "仙品",
}

func (q Quality) String() string {
	if name, ok := QualityNames[q]; ok {
		return name
	}
	return "未知"
}

// PillEffects 丹药效果
type PillEffects struct {
	ExpBonus            int64   `json:"exp_bonus,omitempty"`
	HpBonus             int64   `json:"hp_bonus,omitempty"`
	MpBonus             int64   `json:"mp_bonus,omitempty"`
	HealHp              int64   `json:"heal_hp,omitempty"`
	DefenseBonus        int64   `json:"defense_bonus,omitempty"`
	AttackBonus         int64   `json:"attack_bonus,omitempty"`
	CultivationSpeed    float64 `json:"cultivation_speed,omitempty"`
	MeditationEfficiency float64 `json:"meditation_efficiency,omitempty"`
	// 突破小游戏加成（替代旧版概率加成）
	BreakthroughTimeBonus  int64   `json:"breakthrough_time_bonus,omitempty"`  // 突破时限增加（秒），如凝神丹+30s
	BreakthroughRangeBonus float64 `json:"breakthrough_range_bonus,omitempty"` // 节点判定范围扩大（百分比），如聚灵丹+20%
	// 旧版突破概率加成（已废弃，保留兼容）
	BreakthroughBonus   float64 `json:"breakthrough_bonus,omitempty"`
	Duration            int64   `json:"duration,omitempty"` // 持续秒数，0表示立即生效/永久
}

// HasInstantEffect 是否有即时效果（修为/属性增加等）
func (e *PillEffects) HasInstantEffect() bool {
	return e.ExpBonus > 0 || e.HealHp > 0 || e.HpBonus > 0 || e.MpBonus > 0 || e.DefenseBonus > 0 || e.AttackBonus > 0
}

// HasBuffEffect 是否有持续增益效果（修炼速度/闭关效率等）
func (e *PillEffects) HasBuffEffect() bool {
	return e.CultivationSpeed > 0 || e.MeditationEfficiency > 0
}

// HasBreakthroughEffect 是否有突破加成（新旧字段皆算）
func (e *PillEffects) HasBreakthroughEffect() bool {
	return e.BreakthroughBonus > 0 || e.BreakthroughTimeBonus > 0 || e.BreakthroughRangeBonus > 0
}

// RecipeIngredient 配方所需材料
type RecipeIngredient struct {
	ItemID int    `json:"item_id"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
}

// Recipe 丹方定义
type Recipe struct {
	ID            int                `json:"id"`
	Name          string             `json:"name"`
	RealmRequired int                `json:"realm_required"`
	LevelRequired int                `json:"level_required"`
	Ingredients   []RecipeIngredient `json:"ingredients"`
	Effects       PillEffects        `json:"effects"`
	CraftTime     int                `json:"craft_time"` // 炼制耗时（秒）
}

// Ingredient 灵药材料定义
type Ingredient struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Rarity      int    `json:"rarity"`
	Description string `json:"description"`
}

// Pill 丹药实例（玩家持有的成品丹药）
type Pill struct {
	ID          string      `json:"id"`           // 唯一标识
	RecipeID    int         `json:"recipe_id"`    // 来源配方ID
	Name        string      `json:"name"`         // 丹药名称
	Quality     Quality     `json:"quality"`      // 品质枚举值
	QualityName string      `json:"quality_name"` // 品质中文名
	Effects     PillEffects `json:"effects"`      // 效果
	Count       int         `json:"count"`        // 堆叠数量
	CreatedAt   int64       `json:"created_at"`   // 创建时间戳
	Expiry      int64       `json:"expiry"`       // 过期时间戳，0=永不过期
}

// NewPillID 生成丹药唯一ID
func NewPillID(recipeID int, quality Quality, createdAt int64) string {
	return fmt.Sprintf("pill_%d_%d_%d", recipeID, quality, createdAt)
}

// CraftResult 炼制结果
type CraftResult struct {
	Success     bool    `json:"success"`      // 是否成功
	Quality     Quality `json:"quality"`      // 品质
	QualityName string  `json:"quality_name"` // 品质中文名
	Pill        *Pill   `json:"pill,omitempty"` // 成品丹药
	ExpGained   int64   `json:"exp_gained"`   // 修为值获取
	AlchemyExp  int64   `json:"alchemy_exp"`  // 炼丹经验值
}

// CollectResult 采集结果
type CollectResult struct {
	Success      bool   `json:"success"`       // 是否成功
	IngredientID int    `json:"ingredient_id"` // 材料ID
	Name         string `json:"name"`          // 材料名称
	Count        int    `json:"count"`         // 获得数量
	Message      string `json:"message"`       // 提示信息
}

// AlchemyConfig 炼丹配置（从JSON加载）
type AlchemyConfig struct {
	Recipes     []Recipe     `json:"recipes"`
	Ingredients []Ingredient `json:"ingredients"`
}
