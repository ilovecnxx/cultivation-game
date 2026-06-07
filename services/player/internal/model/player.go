package model

import (
	"math/rand"
	"time"
)

// ============================================================
// 境界系统（10境）
// ============================================================

const (
	RealmForge   = 1  // 锻体
	RealmQiRef   = 2  // 练气
	RealmBase    = 3  // 筑基
	RealmGolden  = 4  // 金丹
	RealmNascent = 5  // 元婴
	RealmSpirit  = 6  // 化神
	RealmVoid    = 7  // 炼虚
	RealmMerge   = 8  // 合体
	RealmAscend  = 9  // 大乘
	RealmTrib    = 10 // 渡劫
)

var RealmNames = map[int32]string{
	RealmForge:   "锻体",
	RealmQiRef:   "练气",
	RealmBase:    "筑基",
	RealmGolden:  "金丹",
	RealmNascent: "元婴",
	RealmSpirit:  "化神",
	RealmVoid:    "炼虚",
	RealmMerge:   "合体",
	RealmAscend:  "大乘",
	RealmTrib:    "渡劫",
}

// RealmCoefficient 境界系数（修炼速度）
var RealmCoefficient = map[int32]int64{
	RealmForge:   1,
	RealmQiRef:   2,
	RealmBase:    3,
	RealmGolden:  5,
	RealmNascent: 8,
	RealmSpirit:  12,
	RealmVoid:    18,
	RealmMerge:   25,
	RealmAscend:  35,
	RealmTrib:    50,
}

// BreakthroughBaseRate 突破基础成功率(%) — 真实修仙界比例
var BreakthroughBaseRate = map[int32]int32{
	RealmForge:   7000,   // 锻体→练气: 70%（凡人→修士）
	RealmQiRef:   2500,   // 练气→筑基: 25%（外门→内门）
	RealmBase:    400,    // 筑基→金丹: 4%（百里挑一）
	RealmGolden:  80,     // 金丹→元婴: 0.8%（万中无一）
	RealmNascent: 12,     // 元婴→化神: 0.12%（十万中一）
	RealmSpirit:  1,      // 化神→炼虚: 0.015%（百万中一）
	RealmVoid:    0,      // 炼虚→合体: 0.002%（亿中无一）
	RealmMerge:   0,      // 合体→大乘: 0.0002%
	RealmAscend:  0,      // 大乘→渡劫: 0.00002%
	RealmTrib:    0,      // 渡劫→飞升: 0.000002%
}
// 注意：基础值存储为 ×100 倍（如 7000=70.00%, 80=0.80%, 1=0.01%）。高境界极低，依赖灵根品质倍率拉升。

// QualityBreakthroughMultiplier 灵根品质对突破率的倍率（核心：好灵根才能冲击高境界）
var QualityBreakthroughMultiplier = map[int32]float64{
	RootQualityNone:    1.0,
	RootQualityLow:     5.0,
	RootQualityMedium:  15.0,
	RootQualityHigh:    40.0,
	RootQualityPerfect: 100.0,
}

// AgeBreakthroughMultiplier 年龄对突破率的影响
// 1 现实天 = 1 寿元年
func AgeBreakthroughMultiplier(ageDays int64, totalLifespan int64) float64 {
	if totalLifespan <= 0 {
		return 1.0
	}
	ratio := float64(ageDays) / float64(totalLifespan)
	switch {
	case ratio < 0.25:
		return 1.5 // 少年: ×1.5
	case ratio < 0.50:
		return 1.2 // 青年: ×1.2
	case ratio < 0.75:
		return 0.8 // 中年: ×0.8
	default:
		return 0.3 // 老年: ×0.3
	}
}

// AgeBracketName 年龄段中文名
func AgeBracketName(ageDays int64, totalLifespan int64) string {
	if totalLifespan <= 0 {
		return "少年"
	}
	ratio := float64(ageDays) / float64(totalLifespan)
	switch {
	case ratio < 0.25:
		return "少年"
	case ratio < 0.50:
		return "青年"
	case ratio < 0.75:
		return "中年"
	default:
		return "老年"
	}
}

// RemainingLifespanBonus 剩余寿元加成（满寿元时 +15%）
func RemainingLifespanBonus(ageDays int64, totalLifespan int64) float64 {
	if totalLifespan <= 0 {
		return 0
	}
	remaining := totalLifespan - ageDays
	if remaining < 0 {
		remaining = 0
	}
	return float64(remaining) / float64(totalLifespan) * 15.0
}

// ============================================================
// 灵根系统
// ============================================================

const (
	SpiritRootNone  = 0 // 无灵根（锻体期）
	SpiritRootMetal = 1 // 金
	SpiritRootWood  = 2 // 木
	SpiritRootWater = 3 // 水
	SpiritRootFire  = 4 // 火
	SpiritRootEarth = 5 // 土
	SpiritRootDi    = 6 // 地
	SpiritRootTian  = 7 // 天
)

var SpiritRootNames = map[int32]string{
	SpiritRootNone:  "无灵根",
	SpiritRootMetal: "金灵根",
	SpiritRootWood:  "木灵根",
	SpiritRootWater: "水灵根",
	SpiritRootFire:  "火灵根",
	SpiritRootEarth: "土灵根",
	SpiritRootDi:    "地灵根",
	SpiritRootTian:  "天灵根",
}

// 旧灵根常量（向后兼容）
const (
	SpiritRootWind    = SpiritRootDi
	SpiritRootThunder = SpiritRootTian
	SpiritRootIce     = 8
)

// 灵根品质
const (
	RootQualityNone    = 0 // 无品
	RootQualityLow     = 1 // 下品
	RootQualityMedium  = 2 // 中品
	RootQualityHigh    = 3 // 上品
	RootQualityPerfect = 4 // 极品
)

var RootQualityNames = map[int32]string{
	RootQualityNone:    "无品",
	RootQualityLow:     "下品",
	RootQualityMedium:  "中品",
	RootQualityHigh:    "上品",
	RootQualityPerfect: "极品",
}

var RootSpeedMultiplier = map[int32]float64{
	RootQualityNone:    0.7,
	RootQualityLow:     1.0,
	RootQualityMedium:  1.3,
	RootQualityHigh:    1.6,
	RootQualityPerfect: 2.0,
}

var RootBreakthroughBonus = map[int32]int32{
	RootQualityNone:    -5,
	RootQualityLow:     0,
	RootQualityMedium:  6,
	RootQualityHigh:    12,
	RootQualityPerfect: 20,
}

// ============================================================
// 灵根属性加成 — 不同灵根类型偏重不同属性
// ============================================================

// SpiritRootAttrBonus 灵根类型对属性的百分比加成
type SpiritRootAttrBonus struct {
	AttackPct  float64 // 攻击加成%
	DefensePct float64 // 防御加成%
	MaxHPPct   float64 // 生命加成%
	MaxMPPct   float64 // 灵力加成%
	CritRatePct float64 // 暴击率加成%
	CritDmgPct float64 // 暴击伤害加成%
	DodgePct   float64 // 闪避加成%
	MPRegenPct float64 // 灵力恢复加成%
}

var SpiritRootAttrBonuses = map[int32]SpiritRootAttrBonus{
	SpiritRootNone:  {},
	SpiritRootMetal: {AttackPct: 0.12, CritRatePct: 0.05},                         // 金：主攻伐，暴击
	SpiritRootWood:  {MaxHPPct: 0.15, MPRegenPct: 0.10},                            // 木：主生机，恢复
	SpiritRootWater: {MaxMPPct: 0.12, MPRegenPct: 0.15, DodgePct: 0.03},            // 水：主灵力，灵动
	SpiritRootFire:  {AttackPct: 0.08, CritRatePct: 0.10, CritDmgPct: 0.10},        // 火：主爆发，暴伤
	SpiritRootEarth: {DefensePct: 0.15, MaxHPPct: 0.08},                            // 土：主防御，厚实
	SpiritRootDi:    {AttackPct: 0.08, DefensePct: 0.08, MaxHPPct: 0.08, MaxMPPct: 0.08}, // 地：均衡+8%
	SpiritRootTian:  {AttackPct: 0.12, DefensePct: 0.10, MaxHPPct: 0.12, MaxMPPct: 0.10, CritRatePct: 0.08, MPRegenPct: 0.08}, // 天：全属性
}

// QualityAttrMultiplier 品质对属性加成的倍率
var QualityAttrMultiplier = map[int32]float64{
	RootQualityNone:    0.5,  // 无品：加成打对折
	RootQualityLow:     1.0,  // 下品：全额
	RootQualityMedium:  1.5,  // 中品：1.5倍
	RootQualityHigh:    2.0,  // 上品：2倍
	RootQualityPerfect: 3.0,  // 极品：3倍
}

// ApplySpiritRootBonuses 将灵根类型+品质加成应用到属性上
func ApplySpiritRootBonuses(spiritRoot, rootQuality int32, attack, defense, maxHP, maxMP, critRate, critDmg, dodge, mpRegen int64) (int64, int64, int64, int64, int64, int64, int64, int64) {
	bonus, ok := SpiritRootAttrBonuses[spiritRoot]
	if !ok {
		return attack, defense, maxHP, maxMP, critRate, critDmg, dodge, mpRegen
	}
	mult, ok := QualityAttrMultiplier[rootQuality]
	if !ok {
		mult = 1.0
	}
	attack = int64(float64(attack) * (1 + bonus.AttackPct*mult))
	defense = int64(float64(defense) * (1 + bonus.DefensePct*mult))
	maxHP = int64(float64(maxHP) * (1 + bonus.MaxHPPct*mult))
	maxMP = int64(float64(maxMP) * (1 + bonus.MaxMPPct*mult))
	critRate = int64(float64(critRate) * (1 + bonus.CritRatePct*mult))
	critDmg = int64(float64(critDmg) * (1 + bonus.CritDmgPct*mult))
	dodge = int64(float64(dodge) * (1 + bonus.DodgePct*mult))
	mpRegen = int64(float64(mpRegen) * (1 + bonus.MPRegenPct*mult))
	return attack, defense, maxHP, maxMP, critRate, critDmg, dodge, mpRegen
}

// SpiritRequired 每期所需修为
var SpiritRequired = map[int32][]int64{
	RealmForge:   {100, 120, 150, 180, 220, 270, 330, 400, 500},
	RealmQiRef:   {600, 700, 850, 1000, 1200, 1450, 1750, 2100, 2600},
	RealmBase:    {3000, 3600, 4300, 5200, 6300, 7600, 9200, 11000, 13500},
	RealmGolden:  {16000, 19000, 23000, 28000, 34000, 41000, 50000, 60000, 73000},
	RealmNascent: {88000, 105000, 125000, 150000, 180000, 215000, 260000, 310000, 375000},
	RealmSpirit:  {450000, 540000, 650000, 780000, 935000, 1120000, 1350000, 1620000, 1950000},
	RealmVoid:    {2340000, 2810000, 3370000, 4050000, 4860000, 5830000, 7000000, 8400000, 10080000},
	RealmMerge:   {12100000, 14500000, 17400000, 20900000, 25100000, 30100000, 36100000, 43300000, 52000000},
	RealmAscend:  {62400000, 74900000, 89900000, 107900000, 129500000, 155400000, 186500000, 223800000, 268600000},
	RealmTrib:    {322300000, 386800000, 464200000, 557000000, 668400000, 802100000, 962500000, 1155000000, 1386000000},
}

func GetRequiredSpirit(realm int32, stage int32) int64 {
	if stages, ok := SpiritRequired[realm]; ok {
		idx := int(stage - 1)
		if idx >= 0 && idx < len(stages) {
			return stages[idx]
		}
	}
	return 999999999999
}

func CalcMaxSpirit(realm int32, stage int32) int64 {
	if stages, ok := SpiritRequired[realm]; ok {
		idx := int(stage - 1)
		if idx >= 0 && idx < len(stages) {
			return stages[idx]
		}
		return stages[len(stages)-1] * 150 / 100
	}
	return 999999999999
}

func CalcCultivationRate(realm int32, stage int32, rootQuality int32) float64 {
	base := 10.0
	if coef, ok := RealmCoefficient[realm]; ok {
		base = 10.0 + float64(coef)*float64(stage)
	}
	mult := 1.0
	if realm >= RealmQiRef {
		if m, ok := RootSpeedMultiplier[rootQuality]; ok {
			mult = m
		}
	}
	return base * mult
}

// CalcBreakthroughRate 计算突破成功率
// 公式: 基础率×品质倍率×年龄倍率 + 剩余寿元 + 气运×0.08 + 悟性×0.03
func CalcBreakthroughRate(realm int32, rootQuality int32, ageDays int64, lifespan int64, luck int64, comprehension int64) int32 {
	base := float64(int32(50))
	if r, ok := BreakthroughBaseRate[realm]; ok {
		base = float64(r)
	}
	mult := QualityBreakthroughMultiplier[rootQuality]
	if mult <= 0 {
		mult = 1.0
	}
	ageBonus := AgeBreakthroughMultiplier(ageDays, lifespan)
	remaining := RemainingLifespanBonus(ageDays, lifespan)
	luckBonus := float64(luck) * 0.08
	compBonus := float64(comprehension) * 0.03
	rate := int32(base*mult*ageBonus/100.0 + remaining + luckBonus + compBonus)
	if rate > 95 {
		rate = 95
	}
	if rate < 2 {
		rate = 2
	}
	return rate
}

// ============================================================
// 先天属性：悟性·气运·神识
// ============================================================

// RealmComprehensionBonus 每个大境界的悟性加成
var RealmComprehensionBonus = map[int32]int64{
	RealmForge:   0,
	RealmQiRef:   5,
	RealmBase:    10,
	RealmGolden:  20,
	RealmNascent: 35,
	RealmSpirit:  55,
	RealmVoid:    80,
	RealmMerge:   110,
	RealmAscend:  150,
	RealmTrib:    200,
}

// RootComprehensionBase 灵根品质对应的基础悟性范围 [min, max]
var RootComprehensionBase = map[int32][2]int64{
	RootQualityNone:    {5, 15},
	RootQualityLow:     {15, 35},
	RootQualityMedium:  {35, 55},
	RootQualityHigh:    {55, 75},
	RootQualityPerfect: {75, 100},
}

// CalcComprehension 计算悟性 = 灵根基础 + 境界加成 + 功法加成
func CalcComprehension(rootQuality int32, realm int32, techniques int64) int64 {
	base := int64(10)
	if r, ok := RootComprehensionBase[rootQuality]; ok {
		base = r[0] + (r[1]-r[0])/2 // 取中值，创角时随机
	}
	bonus := int64(0)
	if b, ok := RealmComprehensionBonus[realm]; ok {
		bonus = b
	}
	return base + bonus + techniques
}

// RandomComprehension 创角/觉醒灵根时随机生成悟性
func RandomComprehension(rootQuality int32) int64 {
	r, ok := RootComprehensionBase[rootQuality]
	if !ok {
		r = [2]int64{5, 15}
	}
	return r[0] + int64(rand.Int63())%(r[1]-r[0]+1)
}

// LuckRange 气运每日随机范围（按灵根品质分层）
var LuckRange = map[int32][2]int64{
	RootQualityNone:    {0, 30},
	RootQualityLow:     {0, 50},
	RootQualityMedium:  {0, 70},
	RootQualityHigh:    {0, 85},
	RootQualityPerfect: {0, 100},
}

// SpiritRootLuckBonus 灵根类型对气运的固定加成
var SpiritRootLuckBonus = map[int32]int64{
	SpiritRootNone:  0,
	SpiritRootMetal: 0,
	SpiritRootWood:  3,
	SpiritRootWater: 2,
	SpiritRootFire:  0,
	SpiritRootEarth: 3,
	SpiritRootDi:    5,
	SpiritRootTian:  10,
}

// RollDailyLuck 每日随机气运值
func RollDailyLuck(rootQuality, spiritRoot int32) int64 {
	r, ok := LuckRange[rootQuality]
	if !ok {
		r = [2]int64{0, 30}
	}
	val := r[0] + int64(rand.Int63())%(r[1]-r[0]+1)
	if bonus, ok := SpiritRootLuckBonus[spiritRoot]; ok {
		val += bonus
	}
	if val > 100 {
		val = 100
	}
	return val
}

// LuckDropMultiplier 气运→掉落倍率: 1 + luck/200
func LuckDropMultiplier(luck int64) float64 {
	return 1.0 + float64(luck)/200.0
}

// LuckEncounterChance 气运→奇遇概率: luck/100 (如 luck=80 → 80%额外奇遇概率加成)
func LuckEncounterChance(luck int64) float64 {
	return float64(luck) / 100.0
}

// RealmSpiritSense 每个境界的神识基础值
var RealmSpiritSense = map[int32]int64{
	RealmForge:   100,
	RealmQiRef:   200,
	RealmBase:    300,
	RealmGolden:  500,
	RealmNascent: 800,
	RealmSpirit:  1200,
	RealmVoid:    1800,
	RealmMerge:   2500,
	RealmAscend:  3500,
	RealmTrib:    5000,
}

// RootSpiritSenseMultiplier 灵根品质对神识的倍率
var RootSpiritSenseMultiplier = map[int32]float64{
	RootQualityNone:    0.7,
	RootQualityLow:     1.0,
	RootQualityMedium:  1.3,
	RootQualityHigh:    1.6,
	RootQualityPerfect: 2.0,
}

// CalcSpiritSense 计算神识 = 境界基础 × 灵根倍率
func CalcSpiritSense(realm int32, rootQuality int32) int64 {
	base := int64(100)
	if b, ok := RealmSpiritSense[realm]; ok {
		base = b
	}
	mult := 1.0
	if m, ok := RootSpiritSenseMultiplier[rootQuality]; ok {
		mult = m
	}
	return int64(float64(base) * mult)
}

// SpiritSenseProfessionBonus 神识→副职加成
// 炼丹/画符/炼器成功率加成 = 神识/50 %
// 优良品质概率 = 神识/200 作为额外概率
func SpiritSenseSuccessBonus(ss int64) float64 {
	return float64(ss) / 50.0
}
func SpiritSenseQualityChance(ss int64) float64 {
	return float64(ss) / 200.0
}

// ============================================================
// 境界属性系统
// ============================================================

type RealmBaseAttr struct {
	Attack     int64
	Defense    int64
	MaxHP      int64
	MaxMP      int64
	Speed      int64
	CritRate   int64 // ×100: 500=5.00%
	CritDmg    int64 // ×100: 15000=150.00%
	Dodge      int64 // ×100: 300=3.00%
	Hit        int64 // ×100: 9500=95.00%
	CultBonus  int64 // ×100: 修炼加成
	BreakBonus int64 // ×100: 突破加成
	MPRegen    int64 // ×100: 每回合灵力恢复%
	Lifespan   int64 // 寿元(年)
}

var RealmAttributes = map[int32]RealmBaseAttr{
	RealmForge:   {Attack: 10, Defense: 5, MaxHP: 100, MaxMP: 50, Speed: 100, CritRate: 300, CritDmg: 15000, Dodge: 200, Hit: 9500, CultBonus: 0, BreakBonus: 0, MPRegen: 300, Lifespan: 100},
	RealmQiRef:   {Attack: 25, Defense: 12, MaxHP: 250, MaxMP: 125, Speed: 110, CritRate: 500, CritDmg: 15500, Dodge: 300, Hit: 9600, CultBonus: 0, BreakBonus: 0, MPRegen: 400, Lifespan: 150},
	RealmBase:    {Attack: 60, Defense: 30, MaxHP: 600, MaxMP: 300, Speed: 125, CritRate: 800, CritDmg: 16000, Dodge: 500, Hit: 9700, CultBonus: 0, BreakBonus: 0, MPRegen: 500, Lifespan: 200},
	RealmGolden:  {Attack: 140, Defense: 70, MaxHP: 1400, MaxMP: 700, Speed: 140, CritRate: 1200, CritDmg: 17000, Dodge: 700, Hit: 9750, CultBonus: 0, BreakBonus: 0, MPRegen: 600, Lifespan: 300},
	RealmNascent: {Attack: 300, Defense: 150, MaxHP: 3000, MaxMP: 1500, Speed: 160, CritRate: 1800, CritDmg: 18000, Dodge: 1000, Hit: 9800, CultBonus: 0, BreakBonus: 0, MPRegen: 800, Lifespan: 500},
	RealmSpirit:  {Attack: 600, Defense: 300, MaxHP: 6000, MaxMP: 3000, Speed: 185, CritRate: 2500, CritDmg: 19000, Dodge: 1300, Hit: 9850, CultBonus: 0, BreakBonus: 0, MPRegen: 1000, Lifespan: 800},
	RealmVoid:    {Attack: 1200, Defense: 600, MaxHP: 12000, MaxMP: 6000, Speed: 210, CritRate: 3200, CritDmg: 20000, Dodge: 1600, Hit: 9900, CultBonus: 0, BreakBonus: 0, MPRegen: 1200, Lifespan: 1300},
	RealmMerge:   {Attack: 2400, Defense: 1200, MaxHP: 24000, MaxMP: 12000, Speed: 240, CritRate: 4000, CritDmg: 21500, Dodge: 2000, Hit: 9925, CultBonus: 0, BreakBonus: 0, MPRegen: 1500, Lifespan: 2000},
	RealmAscend:  {Attack: 4500, Defense: 2250, MaxHP: 45000, MaxMP: 22500, Speed: 275, CritRate: 5000, CritDmg: 23000, Dodge: 2500, Hit: 9950, CultBonus: 0, BreakBonus: 0, MPRegen: 1800, Lifespan: 3500},
	RealmTrib:    {Attack: 8000, Defense: 4000, MaxHP: 80000, MaxMP: 40000, Speed: 310, CritRate: 6000, CritDmg: 25000, Dodge: 3000, Hit: 9975, CultBonus: 0, BreakBonus: 0, MPRegen: 2200, Lifespan: 5000},
}

func CalcRealmAttributes(realm int32, stage int32) (attack, defense, maxHP, maxMP, speed, critRate, critDmg, dodge, hit, cultBonus, breakBonus, mpRegen, lifespan int64) {
	attr, ok := RealmAttributes[realm]
	if !ok {
		attr = RealmAttributes[RealmForge]
	}
	bonus := float64(stage-1) * 0.08
	bonusPct := float64(stage-1) * 0.05
	attack = attr.Attack + int64(float64(attr.Attack)*bonus)
	defense = attr.Defense + int64(float64(attr.Defense)*bonus)
	maxHP = attr.MaxHP + int64(float64(attr.MaxHP)*bonus)
	maxMP = attr.MaxMP + int64(float64(attr.MaxMP)*bonus)
	speed = attr.Speed
	critRate = attr.CritRate + int64(float64(attr.CritRate)*bonusPct)
	critDmg = attr.CritDmg + int64(float64(attr.CritDmg)*bonusPct)
	dodge = attr.Dodge + int64(float64(attr.Dodge)*bonusPct)
	hit = attr.Hit + int64(float64(attr.Hit)*bonusPct)
	cultBonus = attr.CultBonus
	breakBonus = attr.BreakBonus
	mpRegen = attr.MPRegen + int64(float64(attr.MPRegen)*bonusPct)
	lifespan = attr.Lifespan + int64(float64(attr.Lifespan)*bonus)
	return
}

// ============================================================
// 数据模型
// ============================================================

type Player struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"uniqueIndex;size:64;not null"`
	Name        string    `json:"name" gorm:"uniqueIndex;size:32;not null"`
	Gender            string    `json:"gender" gorm:"default:male"`
	Profession        string    `json:"profession" gorm:"default:''"`
	ProfessionLevel   int32     `json:"profession_level" gorm:"default:0"`
	ProfessionExp     int64     `json:"profession_exp" gorm:"default:0"`
	Level       int32     `json:"level" gorm:"default:1"`
	Realm       int32     `json:"realm_id" gorm:"default:1"`
	RealmStage  int32     `json:"realm_stage" gorm:"default:1"`
	SpiritRoot  int32     `json:"spirit_root" gorm:"default:0"`
	RootQuality int32     `json:"root_quality" gorm:"default:0"`
	HP          int64     `json:"hp" gorm:"default:100"`
	MaxHP       int64     `json:"max_hp" gorm:"default:100"`
	MP          int64     `json:"mp" gorm:"default:50"`
	MaxMP       int64     `json:"max_mp" gorm:"default:50"`
	Attack      int64     `json:"attack" gorm:"default:10"`
	Defense     int64     `json:"defense" gorm:"default:5"`
	Speed       int64     `json:"speed" gorm:"default:100"`
	CritRate    int64     `json:"crit_rate" gorm:"default:300"`
	CritDmg     int64     `json:"crit_dmg" gorm:"default:15000"`
	Dodge       int64     `json:"dodge" gorm:"default:200"`
	Hit         int64     `json:"hit" gorm:"default:9500"`
	CultBonus   int64     `json:"cult_bonus" gorm:"default:0"`
	BreakBonus  int64     `json:"break_bonus" gorm:"default:0"`
	MPRegen     int64     `json:"mp_regen" gorm:"default:300"`
	Lifespan       int64     `json:"lifespan" gorm:"default:100"`
	Comprehension  int64     `json:"comprehension" gorm:"default:10"`
	Luck           int64     `json:"luck" gorm:"default:10"`
	SpiritSense    int64     `json:"spirit_sense" gorm:"default:100"`
	LastLuckDate   string    `json:"last_luck_date" gorm:"default:"`
	SpiritPower int64     `json:"spirit_power" gorm:"default:0"`
	MaxSpirit   int64     `json:"max_spirit" gorm:"default:100"`
	Experience  int64     `json:"experience" gorm:"default:0"`
	Gold        int64     `json:"gold" gorm:"default:0"`
	BoundGold   int64     `json:"bound_gold" gorm:"default:0"`
	Jade        int64     `json:"jade" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PlayerCache struct {
	ID          int64  `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Gender            string `json:"gender"`
	Profession        string `json:"profession"`
	ProfessionLevel   int32  `json:"profession_level"`
	ProfessionExp     int64  `json:"profession_exp"`
	Level       int32  `json:"level"`
	Realm       int32  `json:"realm"`
	RealmStage  int32  `json:"realm_stage"`
	SpiritRoot  int32  `json:"spirit_root"`
	RootQuality int32  `json:"root_quality"`
	HP          int64  `json:"hp"`
	MaxHP       int64  `json:"max_hp"`
	MP          int64  `json:"mp"`
	MaxMP       int64  `json:"max_mp"`
	Attack      int64  `json:"attack"`
	Defense     int64  `json:"defense"`
	Speed       int64  `json:"speed"`
	CritRate    int64  `json:"crit_rate"`
	CritDmg     int64  `json:"crit_dmg"`
	Dodge       int64  `json:"dodge"`
	Hit         int64  `json:"hit"`
	CultBonus   int64  `json:"cult_bonus"`
	BreakBonus  int64  `json:"break_bonus"`
	MPRegen     int64  `json:"mp_regen"`
	Lifespan       int64  `json:"lifespan"`
	Comprehension  int64  `json:"comprehension"`
	Luck           int64  `json:"luck"`
	SpiritSense    int64  `json:"spirit_sense"`
	LastLuckDate   string `json:"last_luck_date"`
	SpiritPower int64  `json:"spirit_power"`
	MaxSpirit   int64  `json:"max_spirit"`
	Experience  int64  `json:"experience"`
	Gold        int64  `json:"gold"`
	BoundGold   int64  `json:"bound_gold"`
	Jade        int64  `json:"jade"`
}

func (p *Player) ToCache() *PlayerCache {
	return &PlayerCache{
		ID:          p.ID,
		UserID:      p.UserID,
		Name:        p.Name,
		Gender:            p.Gender,
		Profession:        p.Profession,
		ProfessionLevel:   p.ProfessionLevel,
		ProfessionExp:     p.ProfessionExp,
		Level:       p.Level,
		Realm:       p.Realm,
		RealmStage:  p.RealmStage,
		SpiritRoot:  p.SpiritRoot,
		RootQuality: p.RootQuality,
		HP:          p.HP,
		MaxHP:       p.MaxHP,
		MP:          p.MP,
		MaxMP:       p.MaxMP,
		Attack:      p.Attack,
		Defense:     p.Defense,
		Speed:       p.Speed,
		CritRate:    p.CritRate,
		CritDmg:     p.CritDmg,
		Dodge:       p.Dodge,
		Hit:         p.Hit,
		CultBonus:   p.CultBonus,
		BreakBonus:  p.BreakBonus,
		MPRegen:     p.MPRegen,
		Lifespan:       p.Lifespan,
		Comprehension:  p.Comprehension,
		Luck:           p.Luck,
		SpiritSense:    p.SpiritSense,
		LastLuckDate:   p.LastLuckDate,
		SpiritPower:    p.SpiritPower,
		MaxSpirit:   p.MaxSpirit,
		Experience:  p.Experience,
		Gold:        p.Gold,
		BoundGold:   p.BoundGold,
		Jade:        p.Jade,
	}
}

func (p *Player) FromCache(c *PlayerCache) {
	p.ID = c.ID
	p.UserID = c.UserID
	p.Name = c.Name
	p.Gender = c.Gender
	p.Profession = c.Profession
	p.ProfessionLevel = c.ProfessionLevel
	p.ProfessionExp = c.ProfessionExp
	p.Level = c.Level
	p.Realm = c.Realm
	p.RealmStage = c.RealmStage
	p.SpiritRoot = c.SpiritRoot
	p.RootQuality = c.RootQuality
	p.HP = c.HP
	p.MaxHP = c.MaxHP
	p.MP = c.MP
	p.MaxMP = c.MaxMP
	p.Attack = c.Attack
	p.Defense = c.Defense
	p.Speed = c.Speed
	p.CritRate = c.CritRate
	p.CritDmg = c.CritDmg
	p.Dodge = c.Dodge
	p.Hit = c.Hit
	p.CultBonus = c.CultBonus
	p.BreakBonus = c.BreakBonus
	p.MPRegen = c.MPRegen
	p.Lifespan = c.Lifespan
	p.Comprehension = c.Comprehension
	p.Luck = c.Luck
	p.SpiritSense = c.SpiritSense
	p.LastLuckDate = c.LastLuckDate
	p.SpiritPower = c.SpiritPower
	p.MaxSpirit = c.MaxSpirit
	p.Experience = c.Experience
	p.Gold = c.Gold
	p.BoundGold = c.BoundGold
	p.Jade = c.Jade
}

type CreatePlayerRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Name   string `json:"name" binding:"required,min=2,max=16"`
	Gender string `json:"gender"`
}

type PlayerResponse struct {
	Player      *Player `json:"player"`
	RealmName   string  `json:"realm_name"`
	RealmStage  int32   `json:"realm_stage"`
	SpiritName  string  `json:"spirit_name"`
	QualityName string  `json:"quality_name"`
	CultRate    float64 `json:"cult_rate"`
	BreakRate   int32   `json:"break_rate"`
}

