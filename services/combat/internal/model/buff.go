package model

// BuffType buff 类型
type BuffType string

const (
	BuffTypeAttack    BuffType = "attack"     // 攻击力增减
	BuffTypeDefense   BuffType = "defense"    // 防御力增减
	BuffTypeSpeed     BuffType = "speed"      // 速度增减
	BuffTypeHeal      BuffType = "heal"       // 持续回血
	BuffTypeDamage    BuffType = "damage"     // 持续伤害
	BuffTypeStun      BuffType = "stun"       // 眩晕
	BuffTypeShield    BuffType = "shield"     // 护盾
	BuffTypeSilence   BuffType = "silence"    // 沉默(无法释放技能)
	BuffTypeInvincible BuffType = "invincible" // 无敌
)

// BuffEffect buff 生效时机
type BuffEffect string

const (
	BuffEffectInstant   BuffEffect = "instant"   // 立即生效
	BuffEffectOnTurn    BuffEffect = "on_turn"    // 每回合生效
	BuffEffectOnAttack  BuffEffect = "on_attack"  // 攻击时生效
	BuffEffectOnDamaged BuffEffect = "on_damaged" // 受伤时生效
	BuffEffectOnExpire  BuffEffect = "on_expire"  // 消失时生效
)

// Buff buff/debuff 实例
type Buff struct {
	ID          string      `json:"id"`           // 唯一ID
	Type        BuffType    `json:"type"`         // 类型
	Name        string      `json:"name"`         // 名称
	Value       float64     `json:"value"`        // 数值(攻击增减量、每回合伤害量等)
	Duration    int         `json:"duration"`     // 持续回合数
	Remaining   int         `json:"remaining"`    // 剩余回合数
	Stackable   bool        `json:"stackable"`    // 是否可叠加
	Stacks      int         `json:"stacks"`       // 当前层数
	Effect      BuffEffect  `json:"effect"`       // 生效时机
	FromSkillID string      `json:"from_skill_id"` // 来源技能ID
	IsDebuff    bool        `json:"is_debuff"`    // 是否为减益效果
	MaxStacks   int         `json:"max_stacks"`   // 最大层数
}

// NewBuff 创建新的 buff 实例
func NewBuff(t BuffType, name string, value float64, duration int, isDebuff bool) *Buff {
	return &Buff{
		Type:      t,
		Name:      name,
		Value:     value,
		Duration:  duration,
		Remaining: duration,
		Stackable: false,
		Stacks:    1,
		Effect:    BuffEffectOnTurn,
		IsDebuff:  isDebuff,
		MaxStacks: 1,
	}
}

// Tick 每回合减少持续时间, 返回是否过期
func (b *Buff) Tick() bool {
	if b.Duration == -1 { // -1 表示永久
		return false
	}
	b.Remaining--
	return b.Remaining <= 0
}

// Clone 深拷贝
func (b *Buff) Clone() *Buff {
	cp := *b
	return &cp
}
