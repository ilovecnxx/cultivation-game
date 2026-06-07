package models

import "time"

// DamageType 伤害类型枚举。
type DamageType int32

const (
	DamageTypePhysical DamageType = 0 // 物理伤害
	DamageTypeMagical  DamageType = 1 // 法术伤害
	DamageTypeTrue     DamageType = 2 // 真实伤害（无视防御）
)

// SkillTarget 技能目标类型。
type SkillTarget int32

const (
	SkillTargetSingleEnemy  SkillTarget = 0 // 单体敌方
	SkillTargetAllEnemy     SkillTarget = 1 // 全体敌方
	SkillTargetSelf         SkillTarget = 2 // 自身
	SkillTargetSingleAlly   SkillTarget = 3 // 单体友方
	SkillTargetAllAlly      SkillTarget = 4 // 全体友方
)

// Skill 技能定义（功法/法术/武技）。
type Skill struct {
	ID         uint32      `json:"id"`          // 技能ID
	Name       string      `json:"name"`        // 技能名称
	DamageType DamageType  `json:"damage_type"` // 伤害类型
	Target     SkillTarget `json:"target"`      // 目标类型
	BaseDamage int64       `json:"base_damage"` // 基础伤害值
	// 伤害系数，与攻击者属性相乘：伤害 = BaseDamage + Coefficient * AttrValue
	Coefficient float64 `json:"coefficient"`
	AttrScale   string  `json:"attr_scale"`    // 受哪个属性加成（"strength"/"spirit"/"agility"）
	Cooldown    uint32  `json:"cooldown"`       // 冷却回合数
	CostMana    int64   `json:"cost_mana"`      // 灵力消耗
	CritBonus   float64 `json:"crit_bonus"`    // 暴击时额外倍率（如 0.5 表示暴击伤害倍率 = 1.5）
	LevelRequire uint32 `json:"level_require"` // 所需境界等级
	Description string  `json:"description"`   // 技能描述
}

// Fighter 战斗单元（玩家或怪物/NPC）。
type Fighter struct {
	ID        uint64          `json:"id"`         // 唯一ID
	Name      string          `json:"name"`       // 显示名称
	IsPlayer  bool            `json:"is_player"`  // 是否为玩家
	Attr      PlayerAttribute `json:"attr"`       // 当前属性（已含装备和Buff加成）
	HP        int64           `json:"hp"`         // 当前气血
	MaxHP     int64           `json:"max_hp"`     // 最大气血
	MP        int64           `json:"mp"`         // 当前灵力
	MaxMP     int64           `json:"max_mp"`     // 最大灵力
	Skills    []Skill         `json:"skills"`     // 可用技能列表
	Cooldowns map[uint32]uint32 `json:"cooldowns"` // 技能ID -> 剩余冷却回合数
	Buffs     []Buff          `json:"buffs"`      // 当前生效的Buff
}

// Buff 战斗状态效果（增益/减益）。
type Buff struct {
	ID          uint32 `json:"id"`           // Buff模板ID
	Name        string `json:"name"`         // Buff名称
	RemainRounds uint32 `json:"remain_rounds"` // 剩余回合数
	// 属性修正，格式同 PlayerAttribute，表示加减值
	AttrMod PlayerAttribute `json:"attr_mod"`
	// 每回合效果：如中毒每回合扣血
	DamagePerRound int64 `json:"damage_per_round"`
	HealPerRound   int64 `json:"heal_per_round"`
}

// CombatRoundResult 单回合战斗结果。
type CombatRoundResult struct {
	RoundNum  uint32       `json:"round_num"`  // 回合序号
	Actions   []CombatActionDetail `json:"actions"` // 本回合中的行动详情
}

// CombatActionDetail 一次行动的详细信息。
type CombatActionDetail struct {
	AttackerID uint64  `json:"attacker_id"` // 攻击者ID
	SkillID    uint32  `json:"skill_id"`    // 使用的技能ID
	TargetID   uint64  `json:"target_id"`   // 目标ID
	Damage     int64   `json:"damage"`      // 造成的伤害（负数为治疗）
	IsCritical bool    `json:"is_critical"` // 是否暴击
	IsDodged   bool    `json:"is_dodged"`   // 是否被闪避
	IsBlocked  bool    `json:"is_blocked"`  // 是否被格挡
	BuffAdded  []uint32 `json:"buff_added"` // 添加的BuffID列表
	Description string `json:"description"` // 行动描述文本
}

// CombatResult 整场战斗的最终结果。
type CombatResult struct {
	CombatID     string               `json:"combat_id"`      // 战斗唯一ID（UUID）
	AttackerID   uint64               `json:"attacker_id"`    // 发起方ID
	DefenderID   uint64               `json:"defender_id"`    // 防守方ID
	WinnerID     uint64               `json:"winner_id"`      // 胜利方ID（0=平局）
	TotalRounds  uint32               `json:"total_rounds"`    // 总回合数
	Rounds       []CombatRoundResult  `json:"rounds"`         // 各回合详情
	ExpReward    uint64               `json:"exp_reward"`      // 战斗获得经验
	ItemRewards  []uint32             `json:"item_rewards"`    // 掉落物品ID列表
	StartTime    time.Time            `json:"start_time"`      // 战斗开始时间
	EndTime      time.Time            `json:"end_time"`        // 战斗结束时间
	DurationMS   int64                `json:"duration_ms"`     // 战斗耗时（毫秒）
}

// IsAlive 判断战斗单元是否存活。
func (f *Fighter) IsAlive() bool {
	return f.HP > 0
}

// CanUseSkill 判断技能是否可用（冷却中/灵力不足）。
func (f *Fighter) CanUseSkill(skill *Skill) bool {
	if f.MP < skill.CostMana {
		return false
	}
	if cd, ok := f.Cooldowns[skill.ID]; ok && cd > 0 {
		return false
	}
	return true
}

// ApplyDamage 对战斗单元造成伤害，返回实际扣血量。
func (f *Fighter) ApplyDamage(damage int64, damageType DamageType) int64 {
	if damage <= 0 {
		return 0
	}
	var actualDamage int64
	switch damageType {
	case DamageTypeTrue:
		actualDamage = damage
	case DamageTypeMagical:
		reduction := f.Attr.Spirit / 10 // 神识提供法术减伤
		if reduction >= damage {
			actualDamage = 1 // 至少造成1点伤害
		} else {
			actualDamage = damage - reduction
		}
	default: // Physical
		reduction := f.Attr.Defense / 10
		if reduction >= damage {
			actualDamage = 1
		} else {
			actualDamage = damage - reduction
		}
	}
	if actualDamage > f.HP {
		actualDamage = f.HP
	}
	f.HP -= actualDamage
	return actualDamage
}

// Heal 治疗战斗单元，返回实际治疗量。
func (f *Fighter) Heal(amount int64) int64 {
	if amount <= 0 || f.HP >= f.MaxHP {
		return 0
	}
	maxHeal := f.MaxHP - f.HP
	if amount > maxHeal {
		amount = maxHeal
	}
	f.HP += amount
	return amount
}
