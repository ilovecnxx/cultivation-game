package engine

import "cultivation-game/services/combat/internal/model"

// elementCycle 五行相克关系: 金 -> 木 -> 土 -> 水 -> 火 -> 金
var elementCycle = map[model.ElementType]model.ElementType{
	model.ElementMetal: model.ElementWood,  // 金克木
	model.ElementWood:  model.ElementEarth, // 木克土
	model.ElementEarth: model.ElementWater, // 土克水
	model.ElementWater: model.ElementFire,  // 水克火
	model.ElementFire:  model.ElementMetal, // 火克金
}

// GetElementMultiplier 获取五行克制倍率
//   - 攻击方属性克制防御方属性: 返回 config 中的克制倍率(默认1.3)
//   - 攻击方属性被防御方属性克制: 返回 config 中的被克倍率(默认0.7)
//   - 无克制关系: 返回1.0
func GetElementMultiplier(attackElement, defenseElement model.ElementType, advantage, disadvantage float64) float64 {
	if attackElement == defenseElement {
		return 1.0 // 同属性无加成
	}

	// 检查攻击方是否克制防御方
	if countered, ok := elementCycle[attackElement]; ok && countered == defenseElement {
		return advantage
	}

	// 检查攻击方是否被防御方克制
	if countered, ok := elementCycle[defenseElement]; ok && countered == attackElement {
		return disadvantage
	}

	return 1.0
}

// IsCountered 检查attackElement是否克制defenseElement
func IsCountered(attackElement, defenseElement model.ElementType) bool {
	countered, ok := elementCycle[attackElement]
	return ok && countered == defenseElement
}

// IsWeakAgainst 检查attackElement是否被defenseElement克制
func IsWeakAgainst(attackElement, defenseElement model.ElementType) bool {
	countered, ok := elementCycle[defenseElement]
	return ok && countered == attackElement
}
