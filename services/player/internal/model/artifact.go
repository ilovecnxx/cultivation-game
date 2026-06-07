package model

import "time"

// ============================================================
// 品质枚举（保持原有）
// ============================================================
const (
	ArtifactQualityMortal   = 1 // 凡品
	ArtifactQualitySpirit   = 2 // 灵品
	ArtifactQualityImmortal = 3 // 仙品
	ArtifactQualityDivine   = 4 // 神品
	ArtifactQualityChaos    = 5 // 混沌
)

var ArtifactQualityNames = map[int]string{
	ArtifactQualityMortal:   "凡品",
	ArtifactQualitySpirit:   "灵品",
	ArtifactQualityImmortal: "仙品",
	ArtifactQualityDivine:   "神品",
	ArtifactQualityChaos:    "混沌",
}

var ArtifactQualityUpgradeRates = map[int]float64{
	ArtifactQualityMortal:   0.50,
	ArtifactQualitySpirit:   0.30,
	ArtifactQualityImmortal: 0.15,
	ArtifactQualityDivine:   0.05,
}

// ============================================================
// 法宝类型 (6 types)
// ============================================================
const (
	ArtifactTypeSword  = 1 // 飞剑   — 攻击提升，远程攻击
	ArtifactTypeShield = 2 // 护盾   — 防御，伤害吸收
	ArtifactTypePagoda = 3 // 宝塔   — 群体控制，敌人减益
	ArtifactTypeOrb    = 4 // 灵珠   — 修炼加速，灵力回复
	ArtifactTypeWings  = 5 // 羽翼   — 速度，闪避
	ArtifactTypeSeal   = 6 // 印玺   — 全属性，突破辅助
)

var ArtifactTypeNames = map[int]string{
	ArtifactTypeSword:  "飞剑",
	ArtifactTypeShield: "护盾",
	ArtifactTypePagoda: "宝塔",
	ArtifactTypeOrb:    "灵珠",
	ArtifactTypeWings:  "羽翼",
	ArtifactTypeSeal:   "印玺",
}

var ArtifactTypeDescriptions = map[int]string{
	ArtifactTypeSword:  "攻击提升，远程攻击",
	ArtifactTypeShield: "防御，伤害吸收",
	ArtifactTypePagoda: "群体控制，敌人减益",
	ArtifactTypeOrb:    "修炼加速，灵力回复",
	ArtifactTypeWings:  "速度，闪避",
	ArtifactTypeSeal:   "全属性，突破辅助",
}

var ArtifactTypeIcons = map[int]string{
	ArtifactTypeSword:  "🗡️",
	ArtifactTypeShield: "🛡️",
	ArtifactTypePagoda: "🗼",
	ArtifactTypeOrb:    "💎",
	ArtifactTypeWings:  "🪽",
	ArtifactTypeSeal:   "👑",
}

// ArtifactTypeBonusMultipliers 各类型属性倍率 [攻击, 防御, HP, 灵力, 速度, 闪避]
var ArtifactTypeBonusMultipliers = map[int][6]float64{
	ArtifactTypeSword:  {1.8, 0.6, 0.8, 0.6, 1.0, 0.4},
	ArtifactTypeShield: {0.6, 2.0, 1.5, 0.5, 0.6, 0.3},
	ArtifactTypePagoda: {0.8, 1.2, 1.0, 1.0, 0.8, 0.5},
	ArtifactTypeOrb:    {0.5, 0.6, 0.8, 2.0, 0.6, 0.4},
	ArtifactTypeWings:  {0.7, 0.5, 0.6, 0.6, 2.0, 1.8},
	ArtifactTypeSeal:   {1.2, 1.2, 1.2, 1.2, 1.2, 1.0},
}

// ============================================================
// 法宝技能枚举（保持原有，扩充觉醒技能）
// ============================================================
const (
	ArtifactSkillSwordShield = 1 // 剑气护体：减伤30%
	ArtifactSkillArmorBreak  = 2 // 破甲一击：无视防御20%
	ArtifactSkillHeal        = 3 // 回春术：回合恢复15%HP
	ArtifactSkillIronBody    = 4 // 金刚不坏：格挡率+25%
	ArtifactSkillChaosForce  = 5 // 混沌之力：全属性+10%

	// 觉醒技能 (6-15)
	ArtifactAwakenSwordRain    = 6  // 万剑归宗：全体攻击
	ArtifactAwakenAbsoluteWall = 7  // 绝对壁垒：免疫伤害1回合
	ArtifactAwakenSoulBind     = 8  // 灵魂锁链：控制+2回合
	ArtifactAwakenManaSpring   = 9  // 灵力源泉：灵力回复翻倍
	ArtifactAwakenWindWalk     = 10 // 风之步：闪避+50%持续2回合
	ArtifactAwakenSealBreak    = 11 // 破印之力：突破成功率+20%
	ArtifactAwakenVoidSlash    = 12 // 虚空斩：必中无视防御
	ArtifactAwakenIronFortress = 13 // 铁壁：伤害吸收+50%
	ArtifactAwakenHeavenSupp   = 14 // 镇狱：敌人攻防-30%
	ArtifactAwakenSoulMerge    = 15 // 神魂合一：全属性+25%
)

var ArtifactAwakenSkillNames = map[int]string{
	ArtifactAwakenSwordRain:    "万剑归宗",
	ArtifactAwakenAbsoluteWall: "绝对壁垒",
	ArtifactAwakenSoulBind:     "灵魂锁链",
	ArtifactAwakenManaSpring:   "灵力源泉",
	ArtifactAwakenWindWalk:     "风之步",
	ArtifactAwakenSealBreak:    "破印之力",
	ArtifactAwakenVoidSlash:    "虚空斩",
	ArtifactAwakenIronFortress: "铁壁",
	ArtifactAwakenHeavenSupp:   "镇狱",
	ArtifactAwakenSoulMerge:    "神魂合一",
}

var ArtifactSkillNames = map[int]string{
	ArtifactSkillSwordShield:   "剑气护体",
	ArtifactSkillArmorBreak:    "破甲一击",
	ArtifactSkillHeal:          "回春术",
	ArtifactSkillIronBody:      "金刚不坏",
	ArtifactSkillChaosForce:    "混沌之力",
	ArtifactAwakenSwordRain:    "万剑归宗",
	ArtifactAwakenAbsoluteWall: "绝对壁垒",
	ArtifactAwakenSoulBind:     "灵魂锁链",
	ArtifactAwakenManaSpring:   "灵力源泉",
	ArtifactAwakenWindWalk:     "风之步",
	ArtifactAwakenSealBreak:    "破印之力",
	ArtifactAwakenVoidSlash:    "虚空斩",
	ArtifactAwakenIronFortress: "铁壁",
	ArtifactAwakenHeavenSupp:   "镇狱",
	ArtifactAwakenSoulMerge:    "神魂合一",
}

// ArtifactAwakenMilestones 觉醒里程碑
type ArtifactAwakenMilestone struct {
	Level     int `json:"level"`    // 20, 40, 60, 80, 100
	SkillID   int `json:"skill_id"` // 解锁的技能ID
	SlotIndex int `json:"slot"`     // 技能槽位 0-4
}

// AwakenMilestonesByType 每类法宝的觉醒里程碑
var AwakenMilestonesByType = map[int][]ArtifactAwakenMilestone{
	ArtifactTypeSword: {
		{Level: 20, SkillID: ArtifactAwakenSwordRain, SlotIndex: 0},
		{Level: 40, SkillID: ArtifactAwakenVoidSlash, SlotIndex: 1},
		{Level: 60, SkillID: ArtifactAwakenSoulMerge, SlotIndex: 2},
		{Level: 80, SkillID: ArtifactAwakenWindWalk, SlotIndex: 3},
		{Level: 100, SkillID: ArtifactSkillChaosForce, SlotIndex: 4},
	},
	ArtifactTypeShield: {
		{Level: 20, SkillID: ArtifactAwakenAbsoluteWall, SlotIndex: 0},
		{Level: 40, SkillID: ArtifactAwakenIronFortress, SlotIndex: 1},
		{Level: 60, SkillID: ArtifactAwakenSoulMerge, SlotIndex: 2},
		{Level: 80, SkillID: ArtifactAwakenHeavenSupp, SlotIndex: 3},
		{Level: 100, SkillID: ArtifactSkillChaosForce, SlotIndex: 4},
	},
	ArtifactTypePagoda: {
		{Level: 20, SkillID: ArtifactAwakenSoulBind, SlotIndex: 0},
		{Level: 40, SkillID: ArtifactAwakenHeavenSupp, SlotIndex: 1},
		{Level: 60, SkillID: ArtifactAwakenSoulMerge, SlotIndex: 2},
		{Level: 80, SkillID: ArtifactAwakenSwordRain, SlotIndex: 3},
		{Level: 100, SkillID: ArtifactSkillChaosForce, SlotIndex: 4},
	},
	ArtifactTypeOrb: {
		{Level: 20, SkillID: ArtifactAwakenManaSpring, SlotIndex: 0},
		{Level: 40, SkillID: ArtifactSkillHeal, SlotIndex: 1},
		{Level: 60, SkillID: ArtifactAwakenSoulMerge, SlotIndex: 2},
		{Level: 80, SkillID: ArtifactAwakenSealBreak, SlotIndex: 3},
		{Level: 100, SkillID: ArtifactSkillChaosForce, SlotIndex: 4},
	},
	ArtifactTypeWings: {
		{Level: 20, SkillID: ArtifactAwakenWindWalk, SlotIndex: 0},
		{Level: 40, SkillID: ArtifactSkillIronBody, SlotIndex: 1},
		{Level: 60, SkillID: ArtifactAwakenSoulMerge, SlotIndex: 2},
		{Level: 80, SkillID: ArtifactAwakenVoidSlash, SlotIndex: 3},
		{Level: 100, SkillID: ArtifactSkillChaosForce, SlotIndex: 4},
	},
	ArtifactTypeSeal: {
		{Level: 20, SkillID: ArtifactAwakenSealBreak, SlotIndex: 0},
		{Level: 40, SkillID: ArtifactSkillArmorBreak, SlotIndex: 1},
		{Level: 60, SkillID: ArtifactAwakenSoulMerge, SlotIndex: 2},
		{Level: 80, SkillID: ArtifactAwakenAbsoluteWall, SlotIndex: 3},
		{Level: 100, SkillID: ArtifactSkillChaosForce, SlotIndex: 4},
	},
}

// ============================================================
// 进化材料配置
// ============================================================
type ArtifactEvolveCost struct {
	Level        int    `json:"level"`         // 当前等级段
	Gold         int64  `json:"gold"`          // 灵石消耗
	MaterialID   int64  `json:"material_id"`   // 材料物品ID
	MaterialQty  int32  `json:"material_qty"`  // 材料数量
	MaterialName string `json:"material_name"` // 材料显示名称
}

// EvolveCostTable 进化消耗表（按等级段）
// 每10级一个档次
var EvolveCostTable = []ArtifactEvolveCost{
	{Level: 1, Gold: 500, MaterialID: 1001, MaterialQty: 2, MaterialName: "碧落石"},
	{Level: 11, Gold: 2000, MaterialID: 1001, MaterialQty: 5, MaterialName: "碧落石"},
	{Level: 21, Gold: 5000, MaterialID: 1002, MaterialQty: 3, MaterialName: "天星砂"},
	{Level: 31, Gold: 10000, MaterialID: 1002, MaterialQty: 6, MaterialName: "天星砂"},
	{Level: 41, Gold: 20000, MaterialID: 1003, MaterialQty: 3, MaterialName: "混沌石"},
	{Level: 51, Gold: 40000, MaterialID: 1003, MaterialQty: 6, MaterialName: "混沌石"},
	{Level: 61, Gold: 80000, MaterialID: 1004, MaterialQty: 2, MaterialName: "鸿蒙紫气"},
	{Level: 71, Gold: 150000, MaterialID: 1004, MaterialQty: 4, MaterialName: "鸿蒙紫气"},
	{Level: 81, Gold: 300000, MaterialID: 1005, MaterialQty: 3, MaterialName: "天道碎片"},
	{Level: 91, Gold: 500000, MaterialID: 1005, MaterialQty: 5, MaterialName: "天道碎片"},
}

func GetEvolveCost(level int) ArtifactEvolveCost {
	idx := (level - 1) / 10
	if idx >= len(EvolveCostTable) {
		idx = len(EvolveCostTable) - 1
	}
	// clamp to valid index
	if level < 1 {
		idx = 0
	}
	return EvolveCostTable[idx]
}

// ============================================================
// 法宝灵宠（器灵）
// ============================================================
const (
	ArtifactSpiritPersonalityReserved = 1 // 内敛
	ArtifactSpiritPersonalityEarnest  = 2 // 热忱
	ArtifactSpiritPersonalityHaughty  = 3 // 高傲
	ArtifactSpiritPersonalityCheerful = 4 // 活泼
	ArtifactSpiritPersonalitySinister = 5 // 阴沉
	ArtifactSpiritPersonalityAncient  = 6 // 古板
)

var ArtifactSpiritPersonalityNames = map[int]string{
	ArtifactSpiritPersonalityReserved: "内敛",
	ArtifactSpiritPersonalityEarnest:  "热忱",
	ArtifactSpiritPersonalityHaughty:  "高傲",
	ArtifactSpiritPersonalityCheerful: "活泼",
	ArtifactSpiritPersonalitySinister: "阴沉",
	ArtifactSpiritPersonalityAncient:  "古板",
}

// SpiritDialogueTemplate 器灵对话模板
type SpiritDialogueEntry struct {
	Personality int    `json:"personality"`
	Event       string `json:"event"` // 触发事件: awake/evolve/combat/bind/idle
	Dialogue    string `json:"dialogue"`
}

// DefaultSpiritDialogues 默认器灵对话库
var DefaultSpiritDialogues = []SpiritDialogueEntry{
	{Personality: ArtifactSpiritPersonalityReserved, Event: "bind", Dialogue: "……有缘。"},
	{Personality: ArtifactSpiritPersonalityReserved, Event: "awake", Dialogue: "力量，苏醒了。"},
	{Personality: ArtifactSpiritPersonalityReserved, Event: "evolve", Dialogue: "蜕变。"},
	{Personality: ArtifactSpiritPersonalityReserved, Event: "combat", Dialogue: "……出手了。"},
	{Personality: ArtifactSpiritPersonalityReserved, Event: "idle", Dialogue: "静默如初。"},
	{Personality: ArtifactSpiritPersonalityEarnest, Event: "bind", Dialogue: "主人！一起变强吧！"},
	{Personality: ArtifactSpiritPersonalityEarnest, Event: "awake", Dialogue: "我感觉浑身充满了力量！"},
	{Personality: ArtifactSpiritPersonalityEarnest, Event: "evolve", Dialogue: "太好了，又进步了！"},
	{Personality: ArtifactSpiritPersonalityEarnest, Event: "combat", Dialogue: "交给我吧！"},
	{Personality: ArtifactSpiritPersonalityEarnest, Event: "idle", Dialogue: "主人，我随时待命！"},
	{Personality: ArtifactSpiritPersonalityHaughty, Event: "bind", Dialogue: "哼，勉强认可你。"},
	{Personality: ArtifactSpiritPersonalityHaughty, Event: "awake", Dialogue: "这才像话。"},
	{Personality: ArtifactSpiritPersonalityHaughty, Event: "evolve", Dialogue: "本应如此。"},
	{Personality: ArtifactSpiritPersonalityHaughty, Event: "combat", Dialogue: "蝼蚁，退下。"},
	{Personality: ArtifactSpiritPersonalityHaughty, Event: "idle", Dialogue: "无聊。"},
	{Personality: ArtifactSpiritPersonalityCheerful, Event: "bind", Dialogue: "哇！你好呀！"},
	{Personality: ArtifactSpiritPersonalityCheerful, Event: "awake", Dialogue: "好棒好棒！新力量！"},
	{Personality: ArtifactSpiritPersonalityCheerful, Event: "evolve", Dialogue: "耶！又变强了！"},
	{Personality: ArtifactSpiritPersonalityCheerful, Event: "combat", Dialogue: "看我厉害！"},
	{Personality: ArtifactSpiritPersonalityCheerful, Event: "idle", Dialogue: "嘿嘿，我们出去玩吧~"},
	{Personality: ArtifactSpiritPersonalitySinister, Event: "bind", Dialogue: "终于找到你了……"},
	{Personality: ArtifactSpiritPersonalitySinister, Event: "awake", Dialogue: "力量……还不够……"},
	{Personality: ArtifactSpiritPersonalitySinister, Event: "evolve", Dialogue: "有意思。"},
	{Personality: ArtifactSpiritPersonalitySinister, Event: "combat", Dialogue: "死吧。"},
	{Personality: ArtifactSpiritPersonalitySinister, Event: "idle", Dialogue: "……盯着你看。"},
	{Personality: ArtifactSpiritPersonalityAncient, Event: "bind", Dialogue: "万载岁月，今朝有主。"},
	{Personality: ArtifactSpiritPersonalityAncient, Event: "awake", Dialogue: "古法复苏。"},
	{Personality: ArtifactSpiritPersonalityAncient, Event: "evolve", Dialogue: "道法自然，循序渐进。"},
	{Personality: ArtifactSpiritPersonalityAncient, Event: "combat", Dialogue: "尔等，不知天高地厚。"},
	{Personality: ArtifactSpiritPersonalityAncient, Event: "idle", Dialogue: "修炼不可懈怠。"},
}

// ============================================================
// 法宝共鸣（套装加成）
// ============================================================
type ArtifactResonanceSet struct {
	ID         int              `json:"id"`
	Name       string           `json:"name"`
	Desc       string           `json:"desc"`
	Count      int              `json:"count"`       // 需求法宝数量
	Bonuses    map[string]int64 `json:"bonuses"`     // 属性加成
	UnlockHint string           `json:"unlock_hint"` // 解锁条件提示
}

// ResonanceSets 共鸣套装配置
var ResonanceSets = []ArtifactResonanceSet{
	{
		ID: 1, Name: "五行初识", Desc: "收集5种不同属性的法宝，激活基础五行之力",
		Count: 2, Bonuses: map[string]int64{"attack": 500, "defense": 300},
		UnlockHint: "持有任意2种不同类型法宝",
	},
	{
		ID: 2, Name: "五行汇聚", Desc: "五行之力汇聚，全属性大幅提升",
		Count: 3, Bonuses: map[string]int64{"attack": 1500, "defense": 1000, "hp": 5000},
		UnlockHint: "持有任意3种不同类型法宝",
	},
	{
		ID: 3, Name: "混沌初开", Desc: "混沌之力涌动，解锁隐藏潜能",
		Count: 4, Bonuses: map[string]int64{"attack": 3000, "defense": 2000, "hp": 10000, "mp": 500},
		UnlockHint: "持有任意4种不同类型法宝",
	},
	{
		ID: 4, Name: "万法归一", Desc: "集齐全部6种法宝，天地之力加身",
		Count: 5, Bonuses: map[string]int64{"attack": 5000, "defense": 4000, "hp": 20000, "mp": 1000, "speed": 200},
		UnlockHint: "持有任意5种不同类型法宝",
	},
	{
		ID: 5, Name: "天道圆满", Desc: "六道齐聚，万法不侵",
		Count: 6, Bonuses: map[string]int64{"attack": 10000, "defense": 8000, "hp": 50000, "mp": 3000, "speed": 500, "dodge": 300},
		UnlockHint: "持有全部6种不同类型法宝",
	},
}

// ============================================================
// 法宝试炼
// ============================================================
type ArtifactTrialStage struct {
	StageID            int    `json:"stage_id"`
	Name               string `json:"name"`
	Desc               string `json:"desc"`
	MinLevel           int    `json:"min_level"`   // 最小法宝等级
	MinQuality         int    `json:"min_quality"` // 最小法宝品质
	MonsterAttack      int64  `json:"monster_attack"`
	MonsterDefense     int64  `json:"monster_defense"`
	MonsterHP          int64  `json:"monster_hp"`
	RewardGold         int64  `json:"reward_gold"`
	RewardExp          int64  `json:"reward_exp"`
	RewardMaterialID   int64  `json:"reward_material_id"`
	RewardMaterialQty  int32  `json:"reward_material_qty"`
	RewardMaterialName string `json:"reward_material_name"`
	UnlockPotential    int    `json:"unlock_potential"` // 解锁的潜力点数
}

var ArtifactTrials = []ArtifactTrialStage{
	{StageID: 1, Name: "灵气试炼·初", Desc: "感受天地灵气的洗礼",
		MinLevel: 10, MinQuality: ArtifactQualityMortal,
		MonsterAttack: 500, MonsterDefense: 200, MonsterHP: 3000,
		RewardGold: 1000, RewardExp: 500, RewardMaterialID: 1001, RewardMaterialQty: 1, RewardMaterialName: "碧落石",
		UnlockPotential: 1},
	{StageID: 2, Name: "灵气试炼·中", Desc: "引导灵气锤炼法宝",
		MinLevel: 20, MinQuality: ArtifactQualitySpirit,
		MonsterAttack: 1500, MonsterDefense: 600, MonsterHP: 8000,
		RewardGold: 3000, RewardExp: 1500, RewardMaterialID: 1001, RewardMaterialQty: 3, RewardMaterialName: "碧落石",
		UnlockPotential: 2},
	{StageID: 3, Name: "五行试炼", Desc: "以五行之力淬炼法宝之躯",
		MinLevel: 30, MinQuality: ArtifactQualitySpirit,
		MonsterAttack: 4000, MonsterDefense: 1500, MonsterHP: 20000,
		RewardGold: 8000, RewardExp: 3000, RewardMaterialID: 1002, RewardMaterialQty: 2, RewardMaterialName: "天星砂",
		UnlockPotential: 3},
	{StageID: 4, Name: "天雷淬炼", Desc: "引天雷之力，锻造无上法宝",
		MinLevel: 40, MinQuality: ArtifactQualityImmortal,
		MonsterAttack: 10000, MonsterDefense: 4000, MonsterHP: 50000,
		RewardGold: 20000, RewardExp: 6000, RewardMaterialID: 1002, RewardMaterialQty: 4, RewardMaterialName: "天星砂",
		UnlockPotential: 4},
	{StageID: 5, Name: "心魔试炼", Desc: "直面内心之魔，突破自我",
		MinLevel: 50, MinQuality: ArtifactQualityImmortal,
		MonsterAttack: 25000, MonsterDefense: 10000, MonsterHP: 120000,
		RewardGold: 50000, RewardExp: 12000, RewardMaterialID: 1003, RewardMaterialQty: 2, RewardMaterialName: "混沌石",
		UnlockPotential: 5},
	{StageID: 6, Name: "虚空试炼", Desc: "踏入虚空，捕捉混沌之力",
		MinLevel: 60, MinQuality: ArtifactQualityDivine,
		MonsterAttack: 60000, MonsterDefense: 25000, MonsterHP: 300000,
		RewardGold: 100000, RewardExp: 25000, RewardMaterialID: 1003, RewardMaterialQty: 4, RewardMaterialName: "混沌石",
		UnlockPotential: 6},
	{StageID: 7, Name: "鸿蒙试炼", Desc: "鸿蒙初开，万法之源",
		MinLevel: 70, MinQuality: ArtifactQualityDivine,
		MonsterAttack: 150000, MonsterDefense: 60000, MonsterHP: 800000,
		RewardGold: 200000, RewardExp: 50000, RewardMaterialID: 1004, RewardMaterialQty: 2, RewardMaterialName: "鸿蒙紫气",
		UnlockPotential: 8},
	{StageID: 8, Name: "天道试炼", Desc: "直面天道法则，登临绝顶",
		MinLevel: 80, MinQuality: ArtifactQualityChaos,
		MonsterAttack: 400000, MonsterDefense: 150000, MonsterHP: 2000000,
		RewardGold: 500000, RewardExp: 100000, RewardMaterialID: 1005, RewardMaterialQty: 2, RewardMaterialName: "天道碎片",
		UnlockPotential: 10},
	{StageID: 9, Name: "混沌试炼·极", Desc: "混沌之巅，万法归一",
		MinLevel: 90, MinQuality: ArtifactQualityChaos,
		MonsterAttack: 1000000, MonsterDefense: 400000, MonsterHP: 5000000,
		RewardGold: 1000000, RewardExp: 200000, RewardMaterialID: 1005, RewardMaterialQty: 4, RewardMaterialName: "天道碎片",
		UnlockPotential: 15},
	{StageID: 10, Name: "超脱试炼", Desc: "超脱轮回，成就无上法宝",
		MinLevel: 100, MinQuality: ArtifactQualityChaos,
		MonsterAttack: 3000000, MonsterDefense: 1000000, MonsterHP: 15000000,
		RewardGold: 2000000, RewardExp: 500000, RewardMaterialID: 1005, RewardMaterialQty: 6, RewardMaterialName: "天道碎片",
		UnlockPotential: 20},
}

// ============================================================
// 核心数据模型
// ============================================================

// Artifact 本命法宝（主法宝）
type Artifact struct {
	ID           int64     `json:"id"`
	PlayerID     int64     `json:"player_id"`
	Type         int       `json:"type"` // 法宝类型 1-6
	Name         string    `json:"name"`
	Quality      int       `json:"quality"` // 品质 1-5
	Level        int       `json:"level"`   // 等级 1-100
	Exp          int64     `json:"exp"`
	AttackBonus  int64     `json:"attack_bonus"`
	DefenseBonus int64     `json:"defense_bonus"`
	HPBonus      int64     `json:"hp_bonus"`
	MpBonus      int64     `json:"mp_bonus"`      // 灵力加成
	SpeedBonus   int64     `json:"speed_bonus"`   // 速度加成
	DodgeBonus   int64     `json:"dodge_bonus"`   // 闪避加成
	SkillID      int       `json:"skill_id"`      // 主技能ID
	AwakenSkills []int     `json:"awaken_skills"` // 已解锁的觉醒技能ID列表
	Potential    int       `json:"potential"`     // 潜力点数（通过试炼获得）
	SpiritID     int64     `json:"spirit_id"`     // 关联的器灵ID (0=未激活)
	PowerBonus   int64     `json:"power_bonus"`   // 总战力加成
	BoundAt      time.Time `json:"bound_at"`
}

// ArtifactSpirit 器灵
type ArtifactSpirit struct {
	ID           int64     `json:"id"`
	ArtifactID   int64     `json:"artifact_id"`
	PlayerID     int64     `json:"player_id"`
	Name         string    `json:"name"`
	Personality  int       `json:"personality"`   // 性格 1-6
	BondLevel    int       `json:"bond_level"`    // 好感等级 1-100
	BondExp      int64     `json:"bond_exp"`      // 好感经验
	BondUnlocked int       `json:"bond_unlocked"` // 已解锁好感事件数
	LastDialogue string    `json:"last_dialogue"` // 最近一句对话
	LastEvent    string    `json:"last_event"`    // 最近触发事件
	CreatedAt    time.Time `json:"created_at"`
}

// ArtifactResonance 法宝共鸣进度
type ArtifactResonance struct {
	PlayerID      int64            `json:"player_id"`
	OwnedTypes    []int            `json:"owned_types"`    // 已拥有的法宝类型列表
	ActiveSets    []int            `json:"active_sets"`    // 已激活的共鸣套装ID
	ActiveBonuses map[string]int64 `json:"active_bonuses"` // 总加成
}

// ArtifactTrialProgress 试炼进度
type ArtifactTrialProgress struct {
	PlayerID           int64  `json:"player_id"`
	ArtifactID         int64  `json:"artifact_id"`
	CompletedStages    []int  `json:"completed_stages"`
	LastCompletedStage int    `json:"last_completed_stage"`
	TodayAttempts      int    `json:"today_attempts"`
	LastAttemptDate    string `json:"last_attempt_date"` // YYYY-MM-DD
}

// ArtifactResponse 法宝详情响应
type ArtifactResponse struct {
	Artifact      *Artifact              `json:"artifact"`
	QualityName   string                 `json:"quality_name"`
	TypeName      string                 `json:"type_name"`
	TypeIcon      string                 `json:"type_icon"`
	SkillName     string                 `json:"skill_name"`
	AwakenSkills  map[int]string         `json:"awaken_skills"`  // skillID -> name
	UnlockedAwake []int                  `json:"unlocked_awake"` // slots 0-4
	Spirit        *ArtifactSpirit        `json:"spirit,omitempty"`
	Resonance     *ArtifactResonance     `json:"resonance,omitempty"`
	TrialProgress *ArtifactTrialProgress `json:"trial_progress,omitempty"`
	TotalBonus    map[string]int64       `json:"total_bonus"`
}

// ArtifactListResponse 所有法宝列表响应
type ArtifactListResponse struct {
	Artifacts  []*Artifact        `json:"artifacts"`
	MainSlotID int64              `json:"main_slot_id"` // 主法宝ID
	Resonance  *ArtifactResonance `json:"resonance"`
}

// ArtifactEvolveResult 进化结果
type ArtifactEvolveResult struct {
	Artifact *Artifact `json:"artifact"`
	Success  bool      `json:"success"`
	LevelUp  bool      `json:"level_up"` // 是否升品
	Msg      string    `json:"msg"`
}

// ArtifactAwakenResult 觉醒结果
type ArtifactAwakenResult struct {
	Artifact      *Artifact `json:"artifact"`
	UnlockedSkill int       `json:"unlocked_skill"`
	SkillName     string    `json:"skill_name"`
	SlotIndex     int       `json:"slot_index"`
}

// ArtifactTrialResult 试炼战斗结果
type ArtifactTrialResult struct {
	Stage        *ArtifactTrialStage `json:"stage"`
	Victory      bool                `json:"victory"`
	Rewards      map[string]int64    `json:"rewards"`
	NewPotential int                 `json:"new_potential"`
	SpiritBond   int64               `json:"spirit_bond_exp"` // 器灵好感经验
}
