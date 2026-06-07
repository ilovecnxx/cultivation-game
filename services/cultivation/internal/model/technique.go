// Package model 功法数据结构定义
package model

// Technique 功法定义
type Technique struct {
	ID              int                `json:"id"`               // 功法唯一ID
	Name            string             `json:"name"`             // 功法名称
	Element         string             `json:"element"`          // 五行属性（金木水火土/无）
	Description     string             `json:"description"`      // 功法描述
	CultivationSpeed float64           `json:"cultivation_speed"` // 修炼速度倍数（1.0=基础速度）
	BreakthroughBonus float64          `json:"breakthrough_bonus"` // 突破概率加成
	AttackBonus     float64            `json:"attack_bonus"`     // 攻击力百分比加成
	DefenseBonus    float64            `json:"defense_bonus"`    // 防御力百分比加成
	HPBonus         float64            `json:"hp_bonus"`         // 生命值百分比加成
	ElementAffinity map[string]float64 `json:"element_affinity"` // 五行亲和作用（正数=亲和，负数=排斥）
	RequiredRealmID    int             `json:"required_realm_id"`    // 需求大境界ID
	RequiredRealmLevel int             `json:"required_realm_level"` // 需求小境界等级
}

// PlayerTechnique 玩家已学习的功法
type PlayerTechnique struct {
	TechniqueID int   `json:"technique_id"`
	Level       int   `json:"level"`       // 功法等级
	Experience  int64 `json:"experience"`  // 功法经验
}

// TechniqueLearnResult 学习功法结果
type TechniqueLearnResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Technique *Technique `json:"technique,omitempty"`
}

// CultivationEfficiency 修炼效率计算结果
type CultivationEfficiency struct {
	BaseSpeed     float64           `json:"base_speed"`     // 基础速度
	TechniqueSpeed float64          `json:"technique_speed"` // 功法加成速度
	SpiritRootBonus float64         `json:"spirit_root_bonus"` // 灵根加成
	PillBonus     float64            `json:"pill_bonus"`    // 丹药加成
	FinalSpeed    float64            `json:"final_speed"`   // 最终速度
	ExpPerSecond  int64              `json:"exp_per_second"` // 每秒修为获取(前端显示)
	ExpPerMinute  float64            `json:"exp_per_minute"` // 每分钟修为(精确)
}
