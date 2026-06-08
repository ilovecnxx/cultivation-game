// Package engine 战斗引擎核心算法
package engine

import (
	"math"
	"math/rand"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/model"
)

// ---------------------------------------------------------------------------
// 伤害公式核心函数（全 int64 定点运算）
// ---------------------------------------------------------------------------

// Skill 伤害计算使用的技能简略定义（仅提取倍率字段）
type Skill struct {
	DamageMultiplier float64 // 技能伤害倍率（1.0 = 100%）
}

// CalculateBaseDamage 计算基础伤害
//
// 公式: 基础伤害 = 攻击力 - 目标防御力 × 0.5
//   - 防御不能完全抵消攻击，最多减免 50% 攻击力对应部分
//   - 使用 int64 定点运算，防御折减 0.5 通过 ×1/2 实现
func CalculateBaseDamage(attack, defense int64) int64 {
	dmg := attack - defense/2
	if dmg < 1 {
		dmg = 1
	}
	return dmg
}

// CalculateElementMultiplier 计算五行克制修正倍率（整数索引版）
//
// 参数使用 int 索引: 0=金, 1=木, 2=水, 3=火, 4=土
func CalculateElementMultiplier(attackerElement, defenderElement int) float64 {
	multipliers := [5][5]float64{
		{1.0, 1.3, 0.7, 1.0, 1.0},
		{0.7, 1.0, 1.0, 1.0, 1.3},
		{1.0, 1.0, 1.0, 1.3, 0.7},
		{1.3, 1.0, 0.7, 1.0, 1.0},
		{1.0, 0.7, 1.3, 1.0, 1.0},
	}
	if attackerElement < 0 || attackerElement > 4 || defenderElement < 0 || defenderElement > 4 {
		return 1.0
	}
	return multipliers[attackerElement][defenderElement]
}

// CalculateCritMultiplier 计算暴击修正倍率
//
//	暴击时 ×2.0，非暴击 ×1.0
func CalculateCritMultiplier(isCrit bool) float64 {
	if isCrit {
		return 2.0
	}
	return 1.0
}

// CalculateRealmMultiplier 计算境界等级压制倍率
func CalculateRealmMultiplier(playerRealm, monsterRealm int) float64 {
	diff := playerRealm - monsterRealm
	if diff < -5 {
		diff = -5
	}
	if diff > 5 {
		diff = 5
	}
	return 1.0 + float64(diff)*0.05
}

// CalculateFinalDamage 计算最终伤害（全 int64 核心路径）
//
//	最终伤害 = 基础伤害 × 技能倍率 × 五行修正 × 暴击修正 × 等级压制
//	所有乘法因子累乘后统一 round，避免中间精度损失
//	最终伤害最低为 1
func CalculateFinalDamage(attack, defense int64, skill Skill, attackerElement, defenderElement, playerRealm, monsterRealm int, isCrit bool) int64 {
	base := CalculateBaseDamage(attack, defense)

	// 累乘所有修正因子
	multiplier := skill.DamageMultiplier
	multiplier *= CalculateElementMultiplier(attackerElement, defenderElement)
	multiplier *= CalculateRealmMultiplier(playerRealm, monsterRealm)
	multiplier *= CalculateCritMultiplier(isCrit)

	// 一次 float64 乘法 + round，而非逐层 float64 运算
	final := int64(math.Round(float64(base) * multiplier))
	if final < 1 {
		final = 1
	}
	return final
}

// ---------------------------------------------------------------------------
// CalculateDamage 全量战斗伤害计算（使用 model.Fighter 的版本）
//
// 全 int64 核心路径：
//
//	最终伤害 = (攻击 - 防御 × 0.5) × 技能倍率 × 五行修正 × 暴击修正 × 境界压制
// ---------------------------------------------------------------------------
func CalculateDamage(attacker, defender *model.Fighter, skill *model.Skill, cfg *config.GameConfig) int64 {
	baseAtk := attacker.Attack
	skillMultiplier := skill.Power
	if skillMultiplier <= 0 {
		skillMultiplier = 1.0
	}

	// 五行修正
	elementMultiplier := GetElementMultiplier(
		skill.Element, defender.Element,
		cfg.ElementAdvantageMultiplier,
		cfg.ElementDisadvantageMultiplier,
	)

	// 暴击判定与修正
	critMultiplier := 1.0
	if rand.Float64() < attacker.CritRate {
		critMultiplier = attacker.CritDamage
	}

	// int64 核心：基础伤害 = 攻击 - 防御 × 0.5
	baseDamage := baseAtk - defender.Defense/2
	if baseDamage < 1 {
		baseDamage = 1
	}

	// 境界压制修正
	realmMult := CalculateRealmMultiplier(attacker.Level, defender.Level)

	// 累乘到 multiplier 后一次乘法
	totalMult := skillMultiplier * elementMultiplier * critMultiplier * realmMult
	finalDamage := int64(math.Round(float64(baseDamage) * totalMult))

	// 技能无视防御时，补回防御折减部分
	if skill.IgnoreDefense {
		defensePart := int64(math.Round(float64(defender.Defense/2) * totalMult))
		finalDamage += defensePart
	}

	// 最低伤害保底
	if finalDamage < 1 {
		finalDamage = 1
	}

	return finalDamage
}

// ---------------------------------------------------------------------------
// 治疗 / 经验 / 掉落 / 速度 / 暴击判定
// ---------------------------------------------------------------------------

// CalculateHeal 计算治疗量
func CalculateHeal(healer *model.Fighter, skill *model.Skill) int64 {
	skillMultiplier := skill.Power
	if skillMultiplier <= 0 {
		skillMultiplier = 1.0
	}
	return int64(math.Round(float64(healer.Attack) * skillMultiplier))
}

// CalculateExpReward 计算战斗结束后的经验值
func CalculateExpReward(monsterLevel int, monsterAttack, monsterDefense int64, partySize int) int {
	baseExp := float64(monsterLevel)*50 + float64(monsterAttack+monsterDefense)*0.1
	partyBonus := 1.0 + float64(partySize-1)*0.1
	return int(math.Round(baseExp * partyBonus))
}

// CalculateDropRate 计算掉落概率
func CalculateDropRate(baseRate float64, luck int) float64 {
	return baseRate * (1.0 + float64(luck)/1000.0)
}

// CalculateSpeedOrder 根据速度排序参战者, 返回排序后的下标
func CalculateSpeedOrder(fighters []*model.Fighter) []int {
	type speedEntry struct {
		index int
		speed int64
	}
	entries := make([]speedEntry, len(fighters))
	for i, f := range fighters {
		entries[i] = speedEntry{index: i, speed: f.Speed}
	}

	// 插入排序(从高到低，n 通常很小)
	for i := 1; i < len(entries); i++ {
		key := entries[i]
		j := i - 1
		for j >= 0 && entries[j].speed < key.speed {
			entries[j+1] = entries[j]
			j--
		}
		// 速度相同时随机排序
		if j >= 0 && entries[j].speed == key.speed {
			if rand.Intn(2) == 0 {
				entries[j+1] = key
				continue
			}
		}
		entries[j+1] = key
	}

	result := make([]int, len(entries))
	for i, e := range entries {
		result[i] = e.index
	}
	return result
}

// IsCrit 判定暴击
func IsCrit(critRate float64) bool {
	return rand.Float64() < critRate
}
