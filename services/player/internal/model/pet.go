package model

// PetSkill 灵兽技能
type PetSkill struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Type  string `json:"type"`  // attack=攻击, heal=治疗, buff=增益
	Value int64  `json:"value"` // 数值
}

// 灵兽技能预定义
const (
	PetSkillFlameStrike  = 1 // 烈焰冲击：额外攻击
	PetSkillFrostShield  = 2 // 冰霜护盾：减伤
	PetSkillThunderClap  = 3 // 雷霆一击：高额攻击
	PetSkillWindDodge    = 4 // 疾风闪避：闪避提升
	PetSkillEarthArmor   = 5 // 大地之铠：防御提升
	PetSkillShadowStrike = 6 // 暗影突袭：暴击攻击
	PetSkillHolyLight    = 7 // 圣光术：治疗
	PetSkillVenomSting   = 8 // 毒刺：持续伤害
	PetSkillMoonHowl     = 9 // 月嚎：全属性增益
	PetSkillPhoenixFire  = 10 // 凤凰涅槃：复活+治疗
)

// PetSkillDefs 技能定义
var PetSkillDefs = map[int]PetSkill{
	PetSkillFlameStrike:  {ID: PetSkillFlameStrike, Name: "烈焰冲击", Desc: "喷吐烈焰攻击敌人，造成额外伤害", Type: "attack", Value: 50},
	PetSkillFrostShield:  {ID: PetSkillFrostShield, Name: "冰霜护盾", Desc: "凝结冰霜护盾，减少受到的伤害", Type: "buff", Value: 20},
	PetSkillThunderClap:  {ID: PetSkillThunderClap, Name: "雷霆一击", Desc: "引动天雷之力，造成大量伤害", Type: "attack", Value: 80},
	PetSkillWindDodge:    {ID: PetSkillWindDodge, Name: "疾风闪避", Desc: "借风势闪避敌人攻击", Type: "buff", Value: 15},
	PetSkillEarthArmor:   {ID: PetSkillEarthArmor, Name: "大地之铠", Desc: "引大地之力护体，提升防御", Type: "buff", Value: 30},
	PetSkillShadowStrike: {ID: PetSkillShadowStrike, Name: "暗影突袭", Desc: "从暗影中突袭敌人，必定暴击", Type: "attack", Value: 100},
	PetSkillHolyLight:    {ID: PetSkillHolyLight, Name: "圣光术", Desc: "释放圣光治疗主人", Type: "heal", Value: 200},
	PetSkillVenomSting:   {ID: PetSkillVenomSting, Name: "毒刺", Desc: "注入毒素持续伤害敌人", Type: "attack", Value: 30},
	PetSkillMoonHowl:     {ID: PetSkillMoonHowl, Name: "月嚎", Desc: "对月长嚎，全面提升战力", Type: "buff", Value: 10},
	PetSkillPhoenixFire:  {ID: PetSkillPhoenixFire, Name: "凤凰涅槃", Desc: "浴火重生，恢复大量生命", Type: "heal", Value: 500},
}

// PetSpecies 灵兽物种配置
type PetSpecies struct {
	ID          string `json:"id"`          // 物种标识
	Name        string `json:"name"`        // 名称
	Star        int    `json:"star"`        // 基础星级 1-5
	BaseHP      int64  `json:"base_hp"`     // 基础气血
	BaseAtk     int64  `json:"base_atk"`    // 基础攻击
	BaseDef     int64  `json:"base_def"`    // 基础防御
	SkillID     int    `json:"skill_id"`    // 天赋技能ID
	Description string `json:"description"` // 描述
}

// PetStarFactor 星级成长系数（影响每级属性增长）
var PetStarFactor = map[int]float64{
	1: 1.0,
	2: 1.2,
	3: 1.5,
	4: 1.8,
	5: 2.2,
}

// PetLevelUpExp 计算灵兽升级所需经验
// 公式：baseExp * starFactor * level
// baseExp = 100
func PetLevelUpExp(star, level int) int64 {
	factor, ok := PetStarFactor[star]
	if !ok {
		factor = 1.0
	}
	return int64(float64(100) * factor * float64(level))
}

// EncounterStarWeights 游历遭遇灵兽的星级概率分布
var EncounterStarWeights = map[int]float64{
	1: 0.40, // 40% 概率遇到1星
	2: 0.30, // 30% 概率遇到2星
	3: 0.18, // 18% 概率遇到3星
	4: 0.10, // 10% 概率遇到4星
	5: 0.02, // 2%  概率遇到5星
}

// CaptureRateByStar 捕捉基础成功率（按星级）
var CaptureRateByStar = map[int]float64{
	1: 0.60,
	2: 0.45,
	3: 0.30,
	4: 0.20,
	5: 0.10,
}

// Pet 灵兽
type Pet struct {
	ID        int64    `json:"id"`
	PlayerID  int64    `json:"player_id"`
	Name      string   `json:"name"`
	Species   string   `json:"species"`
	Star      int      `json:"star"`    // 1-5星
	Level     int      `json:"level"`   // 1-100
	Exp       int64    `json:"exp"`
	HP        int64    `json:"hp"`
	Atk       int64    `json:"atk"`
	Def       int64    `json:"def"`
	Skill     PetSkill `json:"skill"`
	Active    bool     `json:"active"` // 是否出战
}

// CalcPetStats 根据物种、星级、等级重新计算灵兽属性
func CalcPetStats(species *PetSpecies, star, level int) (hp, atk, def int64) {
	factor, ok := PetStarFactor[star]
	if !ok {
		factor = 1.0
	}
	hp = species.BaseHP + int64(float64(star)*factor*float64(level)*10)
	atk = species.BaseAtk + int64(float64(star)*factor*float64(level)*5)
	def = species.BaseDef + int64(float64(star)*factor*float64(level)*3)
	return
}
