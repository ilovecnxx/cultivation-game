package engine

import (
	"fmt"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/model"
)

// SkillResult 技能执行结果
type SkillResult struct {
	Skill       *model.Skill `json:"skill"`
	CasterID    string      `json:"caster_id"`
	Targets     []*TargetResult `json:"targets"`
	IsCrit      bool        `json:"is_crit"`
	ElementMult float64     `json:"element_multiplier"`
	Log         string      `json:"log"`
}

// TargetResult 单个目标受击结果
type TargetResult struct {
	TargetID     string  `json:"target_id"`
	TargetName   string  `json:"target_name"`
	Damage       float64 `json:"damage"`
	Heal         float64 `json:"heal"`
	IsCrit       bool    `json:"is_crit"`
	IsDead       bool    `json:"is_dead"`
	AppliedBuffs []*model.Buff `json:"applied_buffs,omitempty"`
	ElementMult  float64 `json:"element_multiplier"`
	IsDodged     bool    `json:"is_dodged"`
	Blocked      bool    `json:"blocked"`
	Log         string  `json:"log,omitempty"`
}

// ExecuteSkill 执行技能
//
// 处理流程:
//  1. 检查冷却和灵力消耗
//  2. 确定目标
//  3. 计算伤害/治疗
//  4. 应用 buff 效果
//  5. 处理吸血
//  6. 返回执行结果
func ExecuteSkill(caster *model.Fighter, skill *model.Skill, allies, enemies []*model.Fighter, cfg *config.GameConfig) *SkillResult {
	result := &SkillResult{
		Skill:    skill,
		CasterID: caster.ID,
		Targets:  make([]*TargetResult, 0),
	}

	// 获取目标列表
	targets := resolveTargets(caster, skill, allies, enemies)

	// 计算五行倍率(只用第一个目标的属性来产生 log 信息)
	if len(targets) > 0 {
		result.ElementMult = GetElementMultiplier(
			skill.Element, targets[0].Element,
			cfg.ElementAdvantageMultiplier,
			cfg.ElementDisadvantageMultiplier,
		)
	}

	// 对每个目标执行效果
	for _, target := range targets {
		targetRes := applySkillToTarget(caster, target, skill, cfg)
		result.Targets = append(result.Targets, targetRes)
		if targetRes.IsCrit {
			result.IsCrit = true
		}
	}

	return result
}

// resolveTargets 解析技能目标列表
func resolveTargets(caster *model.Fighter, skill *model.Skill, allies, enemies []*model.Fighter) []*model.Fighter {
	switch skill.TargetType {
	case model.TargetSingleEnemy:
		// 选择第一个存活的敌方
		for _, e := range enemies {
			if e.IsAlive() {
				return []*model.Fighter{e}
			}
		}
		return nil
	case model.TargetAllEnemy:
		alive := make([]*model.Fighter, 0, len(enemies))
		for _, e := range enemies {
			if e.IsAlive() {
				alive = append(alive, e)
			}
		}
		return alive
	case model.TargetSelf:
		return []*model.Fighter{caster}
	case model.TargetSingleAlly:
		// 选择第一个存活的友方(不包含自己)
		for _, a := range allies {
			if a.IsAlive() && a.ID != caster.ID {
				return []*model.Fighter{a}
			}
		}
		// 没有其他友方则治疗自己
		if caster.IsAlive() {
			return []*model.Fighter{caster}
		}
		return nil
	case model.TargetAllAlly:
		alive := make([]*model.Fighter, 0, len(allies))
		for _, a := range allies {
			if a.IsAlive() {
				alive = append(alive, a)
			}
		}
		return alive
	case model.TargetRandomEnemy:
		alive := make([]*model.Fighter, 0, len(enemies))
		for _, e := range enemies {
			if e.IsAlive() {
				alive = append(alive, e)
			}
		}
		if len(alive) == 0 {
			return nil
		}
		// 随机选一个
		idx := int(caster.TotalDamageDealt) % len(alive) // 伪随机, 测试用
		return []*model.Fighter{alive[idx]}
	default:
		// 默认单体敌方
		for _, e := range enemies {
			if e.IsAlive() {
				return []*model.Fighter{e}
			}
		}
		return nil
	}
}

// applySkillToTarget 对单个目标应用技能效果
func applySkillToTarget(caster, target *model.Fighter, skill *model.Skill, cfg *config.GameConfig) *TargetResult {
	res := &TargetResult{
		TargetID:   target.ID,
		TargetName: target.Name,
	}

	// 检查目标是否被沉默(不能释放技能)
	if skill.Type == model.SkillTypeActive && target.HasBuff(model.BuffTypeSilence) {
		res.Log = target.Name + "被沉默, 技能释放失败"
		return res
	}

	// 检查目标是否无敌
	if target.HasBuff(model.BuffTypeInvincible) {
		res.Damage = 0
		res.Blocked = true
		return res
	}

	// 伤害类技能
	if skill.Power > 0 {
		damage := CalculateDamage(caster, target, skill, cfg)
		res.Damage = damage
		res.IsCrit = IsCrit(caster.CritRate)
		res.ElementMult = GetElementMultiplier(
			skill.Element, target.Element,
			cfg.ElementAdvantageMultiplier,
			cfg.ElementDisadvantageMultiplier,
		)

		// 处理目标护盾
		if target.HasBuff(model.BuffTypeShield) {
			shieldReduction := float64(target.GetBuffStacks(model.BuffTypeShield)) * target.Defense * 0.5
			damage -= shieldReduction
			if damage < 0 {
				damage = 0
			}
		}

		// 实际扣血
		target.TakeDamage(damage)
		caster.TotalDamageDealt += damage

		// 吸血效果
		if skill.LifeSteal > 0 && damage > 0 {
			healAmount := damage * skill.LifeSteal
			caster.Heal(healAmount)
		}

		if !target.IsAlive() {
			res.IsDead = true
		}
	}

	// 治疗类技能(倍率为负表示治疗)
	if skill.Power < 0 {
		healAmount := CalculateHeal(caster, skill)
		actualHeal := target.Heal(healAmount)
		res.Heal = actualHeal
	}

	// 应用 buff 效果
	for _, buffCfg := range skill.Buffs {
		// 概率判定
		if buffCfg.Chance > 0 && buffCfg.Chance < 1.0 {
			if !IsCrit(buffCfg.Chance) {
				continue
			}
		}

		buff := &model.Buff{
			Type:        buffCfg.Type,
			Name:        buffCfg.Name,
			Value:       buffCfg.Value,
			Duration:    buffCfg.Duration,
			Remaining:   buffCfg.Duration,
			Stackable:   buffCfg.Type == model.BuffTypeDamage || buffCfg.Type == model.BuffTypeHeal,
			Stacks:      1,
			Effect:      buffCfg.Effect,
			FromSkillID: skill.ID,
			IsDebuff:    buffCfg.IsDebuff,
			MaxStacks:   5,
		}

		// buff 加给谁: debuff 加给目标, buff 加给施法者
		if buffCfg.IsDebuff {
			target.AddBuff(buff)
		} else {
			caster.AddBuff(buff)
		}
		res.AppliedBuffs = append(res.AppliedBuffs, buff)
	}

	return res
}

// ReduceCooldowns 减少所有技能的冷却
func ReduceCooldowns(fighter *model.Fighter, reduction int) {
	for _, skill := range fighter.Skills {
		if skill.CurrentCD > 0 {
			skill.CurrentCD -= reduction
			if skill.CurrentCD < 0 {
				skill.CurrentCD = 0
			}
		}
	}
}

// ProcessBuffEffects 处理每回合开始的 buff 效果(伤害/治疗)
func ProcessBuffEffects(fighter *model.Fighter) []string {
	logs := make([]string, 0)
	if !fighter.IsAlive() {
		return logs
	}

	for _, b := range fighter.Buffs {
		switch b.Type {
		case model.BuffTypeDamage:
			// 持续伤害
			damage := b.Value * float64(b.Stacks)
			fighter.TakeDamage(damage)
			logs = append(logs, fighter.Name+"受到"+b.Name+"伤害: "+FloatToStr(damage))
		case model.BuffTypeHeal:
			// 持续治疗
			heal := b.Value * float64(b.Stacks)
			fighter.Heal(heal)
			logs = append(logs, fighter.Name+"受到"+b.Name+"治疗: "+FloatToStr(heal))
		}
	}
	return logs
}

// FloatToStr float64 转字符串(保留一位小数)
func FloatToStr(v float64) string {
	return fmt.Sprintf("%.1f", v)
}
