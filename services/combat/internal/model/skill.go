package model

// ElementType 五行属性
type ElementType string

const (
	ElementMetal ElementType = "metal" // 金
	ElementWood  ElementType = "wood"  // 木
	ElementWater ElementType = "water" // 水
	ElementFire  ElementType = "fire"  // 火
	ElementEarth ElementType = "earth" // 土
)

// SkillType 技能类型
type SkillType string

const (
	SkillTypeActive  SkillType = "active"  // 主动技能
	SkillTypePassive SkillType = "passive" // 被动技能
)

// SkillTarget 技能目标类型
type SkillTarget string

const (
	TargetSingleEnemy  SkillTarget = "single_enemy"   // 单体敌方
	TargetAllEnemy     SkillTarget = "all_enemy"      // 全体敌方
	TargetSelf         SkillTarget = "self"           // 自身
	TargetSingleAlly   SkillTarget = "single_ally"    // 单体友方
	TargetAllAlly      SkillTarget = "all_ally"       // 全体友方
	TargetRandomEnemy  SkillTarget = "random_enemy"   // 随机敌方
)

// Skill 技能定义
type Skill struct {
	ID          string      `json:"id"`           // 唯一ID
	Name        string      `json:"name"`         // 名称
	Type        SkillType   `json:"type"`         // 技能类型
	Element     ElementType `json:"element"`      // 五行属性
	TargetType  SkillTarget `json:"target_type"`  // 目标类型
	Power       float64     `json:"power"`        // 技能倍率(百分比, 1.0=100%)
	Cost        int         `json:"cost"`         // 灵力消耗
	Cooldown    int         `json:"cooldown"`     // 冷却回合数
	CurrentCD   int         `json:"current_cd"`   // 当前冷却
	Description string      `json:"description"`  // 描述
	Level       int         `json:"level"`        // 技能等级

	// 被动技能属性
	PassiveStats *PassiveStats `json:"passive_stats,omitempty"` // 被动属性加成

	// buff 效果
	Buffs []BuffEffectConfig `json:"buffs,omitempty"` // 技能附加的 buff

	// 特殊效果
	IgnoreDefense bool `json:"ignore_defense"` // 是否无视防御
	LifeSteal     float64 `json:"life_steal"`  // 吸血比例(0~1)
}

// PassiveStats 被动属性加成
type PassiveStats struct {
	AttackBonus  float64 `json:"attack_bonus"`  // 攻击力加成
	DefenseBonus float64 `json:"defense_bonus"` // 防御力加成
	SpeedBonus   float64 `json:"speed_bonus"`   // 速度加成
	HpBonus      float64 `json:"hp_bonus"`      // 生命加成
	CritRate     float64 `json:"crit_rate"`     // 暴击率加成
	CritDamage   float64 `json:"crit_damage"`   // 暴击伤害加成
}

// BuffEffectConfig 技能携带的 buff 配置
type BuffEffectConfig struct {
	Type      BuffType   `json:"type"`
	Name      string     `json:"name"`
	Value     float64    `json:"value"`
	Duration  int        `json:"duration"`
	Chance    float64    `json:"chance"`    // 触发概率(0~1)
	IsDebuff  bool       `json:"is_debuff"`
	Effect    BuffEffect `json:"effect"`
}
