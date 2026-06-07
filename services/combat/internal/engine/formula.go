// Package engine 战斗引擎核心算法
package engine

import (
	"math"
	"math/rand"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/model"
)

// ---------------------------------------------------------------------------
// 伤害公式核心函数
// ---------------------------------------------------------------------------

// Skill 伤害计算使用的技能简略定义（仅提取倍率字段）
type Skill struct {
	DamageMultiplier float64 // 技能伤害倍率（1.0 = 100%）
}

// CalculateBaseDamage 计算基础伤害
//
// 公式: 基础伤害 = 攻击力 - 目标防御力 × 0.5
//   - 防御不能完全抵消攻击，最多减免 50% 攻击力对应部分
//   - 返回 float64 而非 int64，允许后续乘法累积精度
func CalculateBaseDamage(attack, defense int64) float64 {
	return float64(attack) - float64(defense)*0.5
}

// CalculateElementMultiplier 计算五行克制修正倍率（整数索引版）
//
// 参数使用 int 索引: 0=金, 1=木, 2=水, 3=火, 4=土
//
//	克制关系（5x5 矩阵，行=攻击方，列=防御方）:
//	  金 木 水 火 土
//	金 1.0 1.3 0.7 1.0 1.0   // 金克木
//	木 0.7 1.0 1.0 1.0 1.3   // 木克土
//	水 1.0 1.0 1.0 1.3 0.7   // 水克火
//	火 1.3 1.0 0.7 1.0 1.0   // 火克金
//	土 1.0 0.7 1.3 1.0 1.0   // 土克水
func CalculateElementMultiplier(attackerElement, defenderElement int) float64 {
	multipliers := [5][5]float64{
		// 金   木   水   火   土
		{1.0, 1.3, 0.7, 1.0, 1.0}, // 金（攻击方）
		{0.7, 1.0, 1.0, 1.0, 1.3}, // 木（攻击方）
		{1.0, 1.0, 1.0, 1.3, 0.7}, // 水（攻击方）
		{1.3, 1.0, 0.7, 1.0, 1.0}, // 火（攻击方）
		{1.0, 0.7, 1.3, 1.0, 1.0}, // 土（攻击方）
	}
	if attackerElement < 0 || attackerElement > 4 || defenderElement < 0 || defenderElement > 4 {
		return 1.0 // 越界返回无克制
	}
	return multipliers[attackerElement][defenderElement]
}

// CalculateCritDamage 计算暴击修正
//
//   - 暴击时最终伤害 × 2.0
//   - 非暴击时不变
//
// 注意: baseDamage 是经过基础伤害和倍率计算后的值，此处仅做倍率修正。
func CalculateCritDamage(baseDamage float64, isCrit bool) float64 {
	if isCrit {
		return baseDamage * 2.0
	}
	return baseDamage
}

// CalculateRealmMultiplier 计算境界等级压制倍率
//
// 公式: 每高 1 个境界等级伤害 +5%，每低 1 个境界等级伤害 -5%
//   - 境界等级 = realmID * 10 + realmLevel（简化处理，直接用 realmID 代表大境界级别）
//   - 差值上限 ±5，避免极端压制
//   - 返回值: 1.0 + diff × 0.05
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

// CalculateFinalDamage 计算最终伤害（汇总所有修正因子）
//
// 公式:
//
//	最终伤害 = 基础伤害 × 技能倍率 × 五行修正 × 暴击修正 × 等级压制
//
// 参数:
//   - attack: 攻击方攻击力
//   - defense: 防御方防御力
//   - skill: 技能模板（包含伤害倍率）
//   - attackerElement / defenderElement: 五行索引（0=金,1=木,2=水,3=火,4=土）
//   - playerRealm / monsterRealm: 境界等级ID
//   - isCrit: 是否触发暴击
//
// 最终伤害最低为 1（保证任何攻击都能造成保底伤害）
func CalculateFinalDamage(attack, defense int64, skill Skill, attackerElement, defenderElement, playerRealm, monsterRealm int, isCrit bool) int64 {
	base := CalculateBaseDamage(attack, defense)
	skillMult := skill.DamageMultiplier
	elementMult := CalculateElementMultiplier(attackerElement, defenderElement)
	realmMult := CalculateRealmMultiplier(playerRealm, monsterRealm)
	critMult := CalculateCritDamage(1.0, isCrit) // critMult = 2.0 or 1.0

	final := base * skillMult * elementMult * realmMult * critMult
	if final < 1 {
		final = 1
	}
	return int64(math.Round(final))
}

// ---------------------------------------------------------------------------
// CalculateDamage 全量战斗伤害计算（二合一版本，兼容外部调用）
//
// 集成: 基础攻击 × 技能倍率 × 五行修正 × 暴击修正 - 防御减免
//
// 与 CalculateFinalDamage 的区别:
//   - 使用 model.Fighter / model.Skill / config.GameConfig 作为输入
//   - 内部包含随机暴击判定
//   - 防御减免采用减法形式（与新公式的 50% 防御折减不同，这是加权减法）
//
// 新公式设计（防御按 50% 折减）:
//
//	最终伤害 = (攻击 - 防御 × 0.5) × 技能倍率 × 五行修正 × 暴击修正 × 境界压制
// ---------------------------------------------------------------------------
func CalculateDamage(attacker, defender *model.Fighter, skill *model.Skill, cfg *config.GameConfig) float64 {
	// ---- 基础攻击与防御折减 ----
	baseAtk := attacker.Attack
	skillMultiplier := skill.Power
	if skillMultiplier <= 0 {
		skillMultiplier = 1.0
	}

	// 五行修正（使用现有的 ElementType 版）
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

	// ---- 新公式核心 ----
	// 基础伤害 = 攻击 - 防御 × 0.5
	baseDamage := float64(baseAtk) - float64(defender.Defense)*0.5

	// 境界压制修正（使用 Fighter 的 Level 作为境界等级）
	realmMult := CalculateRealmMultiplier(attacker.Level, defender.Level)

	// 最终伤害 = 基础伤害 × 技能倍率 × 五行修正 × 暴击修正 × 境界压制
	finalDamage := baseDamage * skillMultiplier * elementMultiplier * critMultiplier * realmMult

	// 技能无视防御时，防御折减部分无效（额外追加折减部分）
	if skill.IgnoreDefense {
		defensePenalty := float64(defender.Defense) * 0.5
		finalDamage += defensePenalty * skillMultiplier * elementMultiplier * critMultiplier * realmMult
	}

	// 最低伤害保底
	if finalDamage < 1 {
		finalDamage = 1
	}

	return math.Round(finalDamage)
}

// ---------------------------------------------------------------------------
// 治疗 / 经验 / 掉落 / 速度 / 暴击判定
// ---------------------------------------------------------------------------

// CalculateHeal 计算治疗量
//
// 公式: 治疗量 = 基础攻击力 × 技能倍率
func CalculateHeal(healer *model.Fighter, skill *model.Skill) float64 {
	baseAtk := healer.Attack
	skillMultiplier := skill.Power
	if skillMultiplier <= 0 {
		skillMultiplier = 1.0
	}
	healAmount := baseAtk * skillMultiplier
	return math.Round(healAmount)
}

// CalculateExpReward 计算战斗结束后的经验值
//
// 公式: 基础经验 = 怪物等级 × 50 + 怪物总属性 × 0.1
//
//	组队加成: 每多一人 +10%
func CalculateExpReward(monsterLevel int, monsterAttack, monsterDefense float64, partySize int) int {
	baseExp := float64(monsterLevel)*50 + (monsterAttack+monsterDefense)*0.1
	partyBonus := 1.0 + float64(partySize-1)*0.1
	return int(math.Round(baseExp * partyBonus))
}

// CalculateDropRate 计算掉落概率
//
// 公式: 基础掉落率 × (1 + 幸运值/1000)
func CalculateDropRate(baseRate float64, luck int) float64 {
	return baseRate * (1.0 + float64(luck)/1000.0)
}

// CalculateSpeedOrder 根据速度排序参战者, 返回排序后的下标
//
// 速度相同则随机排序
func CalculateSpeedOrder(fighters []*model.Fighter) []int {
	type speedEntry struct {
		index int
		speed float64
	}
	entries := make([]speedEntry, len(fighters))
	for i, f := range fighters {
		entries[i] = speedEntry{index: i, speed: f.Speed}
	}

	// 冒泡排序(从高到低)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].speed > entries[i].speed {
				entries[i], entries[j] = entries[j], entries[i]
			} else if entries[j].speed == entries[i].speed {
				// 速度相同, 随机决定顺序
				if rand.Intn(2) == 0 {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}
	}

	result := make([]int, len(entries))
	for i, e := range entries {
		result[i] = e.index
	}
	return result
}

// IsCrit 判定暴击 (用于外部检查是否暴击)
func IsCrit(critRate float64) bool {
	return rand.Float64() < critRate
}
