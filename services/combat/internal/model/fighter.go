package model

// FighterType 战斗实体类型
type FighterType string

const (
	FighterTypePlayer  FighterType = "player"  // 玩家
	FighterTypeMonster FighterType = "monster" // 怪物
	FighterTypePet     FighterType = "pet"     // 灵宠
)

// FighterStatus 战斗实体状态
type FighterStatus string

const (
	StatusAlive   FighterStatus = "alive"
	StatusDead    FighterStatus = "dead"
	StatusStunned FighterStatus = "stunned"
)

// Fighter 战斗实体
type Fighter struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        FighterType  `json:"type"`
	Element     ElementType  `json:"element"` // 五行属性
	Level       int          `json:"level"`
	Status      FighterStatus `json:"status"`
	RegionID    string       `json:"region_id,omitempty"` // 所属区域ID(用于按区域筛选怪物列表)

	// 基础属性
	BaseAttack  float64 `json:"base_attack"`
	BaseDefense float64 `json:"base_defense"`
	BaseSpeed   float64 `json:"base_speed"`
	BaseHP      float64 `json:"base_hp"`
	BaseMaxHP   float64 `json:"base_max_hp"`

	// 当前属性(装备+被动加成后)
	Attack  float64 `json:"attack"`
	Defense float64 `json:"defense"`
	Speed   float64 `json:"speed"`
	HP      float64 `json:"hp"`
	MaxHP   float64 `json:"max_hp"`

	// 战斗状态
	MP          int     `json:"mp"`           // 当前灵力
	MaxMP       int     `json:"max_mp"`       // 最大灵力
	CritRate    float64 `json:"crit_rate"`    // 暴击率(0~1)
	CritDamage  float64 `json:"crit_damage"`  // 暴击伤害倍率, 默认2.0

	// 技能
	Skills      []*Skill `json:"skills"`       // 已装备技能
	Passives    []*Skill `json:"passives"`     // 被动技能

	// buff
	Buffs       []*Buff  `json:"buffs"`        // 当前生效的 buff

	// 战斗统计
	TotalDamageDealt   float64 `json:"total_damage_dealt"`
	TotalDamageTaken   float64 `json:"total_damage_taken"`
	TotalHealingDone   float64 `json:"total_healing_done"`
	IsPlayerControlled bool    `json:"is_player_controlled"`
}

// NewFighter 创建新战斗实体
func NewFighter(id, name string, fType FighterType, element ElementType, level int) *Fighter {
	return &Fighter{
		ID:          id,
		Name:        name,
		Type:        fType,
		Element:     element,
		Level:       level,
		Status:      StatusAlive,
		CritRate:    0.05,  // 基础暴击5%
		CritDamage:  2.0,   // 基础暴伤200%
		Buffs:       make([]*Buff, 0),
		Skills:      make([]*Skill, 0),
		Passives:    make([]*Skill, 0),
	}
}

// IsAlive 是否存活
func (f *Fighter) IsAlive() bool {
	return f.Status == StatusAlive && f.HP > 0
}

// TakeDamage 承受伤害
func (f *Fighter) TakeDamage(damage float64) float64 {
	if damage < 0 {
		damage = 0
	}
	f.HP -= damage
	f.TotalDamageTaken += damage
	if f.HP <= 0 {
		f.HP = 0
		f.Status = StatusDead
	}
	return damage
}

// Heal 恢复生命
func (f *Fighter) Heal(amount float64) float64 {
	if !f.IsAlive() {
		return 0
	}
	actual := amount
	if f.HP+actual > f.MaxHP {
		actual = f.MaxHP - f.HP
	}
	f.HP += actual
	f.TotalHealingDone += actual
	return actual
}

// AddBuff 添加 buff
func (f *Fighter) AddBuff(b *Buff) {
	if b.Stackable {
		// 可叠加: 检查是否已有同类型buff, 叠加层数
		for _, existing := range f.Buffs {
			if existing.Type == b.Type && existing.FromSkillID == b.FromSkillID {
				existing.Stacks++
				if existing.Stacks > existing.MaxStacks {
					existing.Stacks = existing.MaxStacks
				}
				existing.Remaining = b.Duration // 刷新持续时间
				return
			}
		}
	} else {
		// 不可叠加: 替换已有同类型buff
		for i, existing := range f.Buffs {
			if existing.Type == b.Type && existing.FromSkillID == b.FromSkillID {
				f.Buffs[i] = b.Clone()
				return
			}
		}
	}
	f.Buffs = append(f.Buffs, b.Clone())
}

// RemoveBuff 移除指定类型的buff
func (f *Fighter) RemoveBuff(buffType BuffType) {
	removed := make([]*Buff, 0, len(f.Buffs))
	for _, b := range f.Buffs {
		if b.Type != buffType {
			removed = append(removed, b)
		}
	}
	f.Buffs = removed
}

// RemoveBuffByID 根据ID移除buff
func (f *Fighter) RemoveBuffByID(id string) {
	removed := make([]*Buff, 0, len(f.Buffs))
	for _, b := range f.Buffs {
		if b.ID != id {
			removed = append(removed, b)
		}
	}
	f.Buffs = removed
}

// HasBuff 检查是否有指定类型的buff
func (f *Fighter) HasBuff(buffType BuffType) bool {
	for _, b := range f.Buffs {
		if b.Type == buffType {
			return true
		}
	}
	return false
}

// HasBuffByName 检查是否有指定名称的buff
func (f *Fighter) HasBuffByName(name string) bool {
	for _, b := range f.Buffs {
		if b.Name == name {
			return true
		}
	}
	return false
}

// GetBuffStacks 获取指定类型buff的层数
func (f *Fighter) GetBuffStacks(buffType BuffType) int {
	for _, b := range f.Buffs {
		if b.Type == buffType {
			return b.Stacks
		}
	}
	return 0
}

// ResetBattleStats 重置战斗统计
func (f *Fighter) ResetBattleStats() {
	f.TotalDamageDealt = 0
	f.TotalDamageTaken = 0
	f.TotalHealingDone = 0
}

// ApplyPassiveStats 应用被动属性加成
func (f *Fighter) ApplyPassiveStats() {
	// 从基础值开始
	f.Attack = f.BaseAttack
	f.Defense = f.BaseDefense
	f.Speed = f.BaseSpeed
	f.HP = f.BaseHP
	f.MaxHP = f.BaseMaxHP

	for _, p := range f.Passives {
		if p.PassiveStats == nil {
			continue
		}
		stats := p.PassiveStats
		f.Attack += f.BaseAttack * stats.AttackBonus
		f.Defense += f.BaseDefense * stats.DefenseBonus
		f.Speed += f.BaseSpeed * stats.SpeedBonus
		f.MaxHP += f.BaseMaxHP * stats.HpBonus
		f.CritRate += stats.CritRate
		f.CritDamage += stats.CritDamage
	}

	// HP 不能超过 MaxHP
	if f.HP > f.MaxHP {
		f.HP = f.MaxHP
	}
}
