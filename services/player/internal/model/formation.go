package model

import "time"

// ============================================================
// 阵法类型 — 8 种特定阵法
// ============================================================
const (
	FormationTypeWuXing        = 1 // 五行阵：五行伤害 +20%
	FormationTypeBaGua         = 2 // 八卦阵：闪避 +15%，反击
	FormationTypeZhuXian       = 3 // 诛仙阵：对单体巨量伤害
	FormationTypeHuShan        = 4 // 护山大阵：全队防御 +25%
	FormationTypeJuLing        = 5 // 聚灵阵：修炼速度 +10%
	FormationTypeMiHun         = 6 // 迷魂阵：敌方命中 -20%
	FormationTypeChuanSong     = 7 // 传送阵：区域间快速传送
	FormationTypeTianGangBeiDou = 8 // 天罡北斗阵：全属性 +10%
)

// FormationTypeNames 阵法类型中文名
var FormationTypeNames = map[int]string{
	FormationTypeWuXing:        "五行阵",
	FormationTypeBaGua:         "八卦阵",
	FormationTypeZhuXian:       "诛仙阵",
	FormationTypeHuShan:        "护山大阵",
	FormationTypeJuLing:        "聚灵阵",
	FormationTypeMiHun:         "迷魂阵",
	FormationTypeChuanSong:     "传送阵",
	FormationTypeTianGangBeiDou: "天罡北斗阵",
}

// FormationTypeIcons 阵法的图标或简短描述
var FormationTypeIcons = map[int]string{
	FormationTypeWuXing:        "五行伤害 +20%",
	FormationTypeBaGua:         "闪避 +15% / 反击",
	FormationTypeZhuXian:       "单体巨量伤害",
	FormationTypeHuShan:        "全队防御 +25%",
	FormationTypeJuLing:        "修炼速度 +10%",
	FormationTypeMiHun:         "敌方命中 -20%",
	FormationTypeChuanSong:     "快速传送",
	FormationTypeTianGangBeiDou: "全属性 +10%",
}

// ============================================================
// 品质枚举
// ============================================================
const (
	FormationQualityOne   = 1 // 一品
	FormationQualityTwo   = 2 // 二品
	FormationQualityThree = 3 // 三品
	FormationQualityFour  = 4 // 四品
	FormationQualityFive  = 5 // 五品
)

// FormationQualityNames 品质中文名
var FormationQualityNames = map[int]string{
	FormationQualityOne:   "一品",
	FormationQualityTwo:   "二品",
	FormationQualityThree: "三品",
	FormationQualityFour:  "四品",
	FormationQualityFive:  "五品",
}

// ============================================================
// 等级 & 部署常量
// ============================================================
const (
	FormationLevelMin = 1
	FormationLevelMax = 10
	// 最大同时部署阵法数
	MaxDeployedFormations = 3
	// 每格阵法最多一个守护灵兽
	MaxGuardianPets = 1
)

// ============================================================
// 阵法熟练度（精通）常量
// ============================================================
const (
	MasteryLevelMin    = 0
	MasteryLevelMax    = 10
	MasteryExpBase    = 100  // 每次使用获得的熟练度基数
	MasteryExpPerLevel = 500 // 每级所需熟练度基数
)

// MasteryLevelNames 熟练度等级名称
var MasteryLevelNames = map[int]string{
	0:  "初窥门径",
	1:  "略知一二",
	2:  "渐入佳境",
	3:  "融会贯通",
	4:  "炉火纯青",
	5:  "登堂入室",
	6:  "出神入化",
	7:  "超凡入圣",
	8:  "登峰造极",
	9:  "返璞归真",
	10: "天人合一",
}

// MasteryLevelThresholds 每级所需熟练度
var MasteryLevelThresholds = func() map[int]int64 {
	m := make(map[int]int64, MasteryLevelMax+1)
	for i := 0; i <= MasteryLevelMax; i++ {
		m[i] = int64(MasteryExpPerLevel * (i + 1))
	}
	return m
}()

// ============================================================
// 阵法联动（联结）— 预设联动组合
// ============================================================

// FormationSynergy 两个阵法配合时产生的额外效果
type FormationSynergy struct {
	TypeA    int     `json:"type_a"`    // 阵法类型 A
	TypeB    int     `json:"type_b"`    // 阵法类型 B
	Name     string  `json:"name"`      // 联动名称
	Bonus    string  `json:"bonus"`     // 联动加成描述
	AtkMult  float64 `json:"atk_mult,omitempty"`
	DefMult  float64 `json:"def_mult,omitempty"`
	OtherPct float64 `json:"other_pct,omitempty"` // 其他百分比加成
}

// PredefinedSynergies 预定义联动组合
var PredefinedSynergies = []FormationSynergy{
	{TypeA: FormationTypeWuXing, TypeB: FormationTypeTianGangBeiDou, Name: "周天星斗", Bonus: "攻击 +15%，五行伤害 +10%", AtkMult: 0.15, OtherPct: 0.10},
	{TypeA: FormationTypeBaGua, TypeB: FormationTypeHuShan, Name: "固若金汤", Bonus: "防御 +20%，闪避 +10%", DefMult: 0.20, OtherPct: 0.10},
	{TypeA: FormationTypeZhuXian, TypeB: FormationTypeMiHun, Name: "杀机四伏", Bonus: "暴击 +15%，敌方命中 -10%", AtkMult: 0.15, OtherPct: -0.10},
	{TypeA: FormationTypeJuLing, TypeB: FormationTypeChuanSong, Name: "天地通达", Bonus: "修炼速度 +15%，体耗 -20%", OtherPct: 0.15},
	{TypeA: FormationTypeWuXing, TypeB: FormationTypeBaGua, Name: "五行八卦", Bonus: "全属性 +8%", OtherPct: 0.08},
	{TypeA: FormationTypeZhuXian, TypeB: FormationTypeTianGangBeiDou, Name: "天诛地灭", Bonus: "对单体伤害 +30%", AtkMult: 0.30},
	{TypeA: FormationTypeHuShan, TypeB: FormationTypeJuLing, Name: "厚德载物", Bonus: "气血 +20%，灵力恢复 +10%", DefMult: 0.20, OtherPct: 0.10},
	{TypeA: FormationTypeMiHun, TypeB: FormationTypeChuanSong, Name: "迷踪幻影", Bonus: "闪避 +10%，移动速度 +15%", OtherPct: 0.10},
}

// ============================================================
// 阵法相克（破阵）
// ============================================================

// FormationCounter 阵法克制关系
type FormationCounter struct {
	AttackerType int     `json:"attacker_type"` // 攻击方阵法
	DefenderType int     `json:"defender_type"` // 防守方阵法
	BreakPct     float64 `json:"break_pct"`     // 克制时降低防守方效果百分比
}

// PredefinedCounters 预定义克制关系
// 五行阵 > 诛仙阵 > 护山大阵 > 八卦阵 > 迷魂阵 > 聚灵阵 > 五行阵 (循环)
// 天罡北斗阵 被所有阵法微弱克制，但克制所有阵法（微弱）
var PredefinedCounters = []FormationCounter{
	{AttackerType: FormationTypeWuXing, DefenderType: FormationTypeZhuXian, BreakPct: 0.30},
	{AttackerType: FormationTypeZhuXian, DefenderType: FormationTypeHuShan, BreakPct: 0.30},
	{AttackerType: FormationTypeHuShan, DefenderType: FormationTypeBaGua, BreakPct: 0.25},
	{AttackerType: FormationTypeBaGua, DefenderType: FormationTypeMiHun, BreakPct: 0.25},
	{AttackerType: FormationTypeMiHun, DefenderType: FormationTypeJuLing, BreakPct: 0.25},
	{AttackerType: FormationTypeJuLing, DefenderType: FormationTypeWuXing, BreakPct: 0.20},
	// 天罡北斗相关
	{AttackerType: FormationTypeTianGangBeiDou, DefenderType: FormationTypeWuXing, BreakPct: 0.15},
	{AttackerType: FormationTypeTianGangBeiDou, DefenderType: FormationTypeBaGua, BreakPct: 0.15},
	{AttackerType: FormationTypeTianGangBeiDou, DefenderType: FormationTypeZhuXian, BreakPct: 0.15},
	{AttackerType: FormationTypeTianGangBeiDou, DefenderType: FormationTypeHuShan, BreakPct: 0.15},
	{AttackerType: FormationTypeTianGangBeiDou, DefenderType: FormationTypeJuLing, BreakPct: 0.15},
	{AttackerType: FormationTypeTianGangBeiDou, DefenderType: FormationTypeMiHun, BreakPct: 0.15},
	{AttackerType: FormationTypeTianGangBeiDou, DefenderType: FormationTypeChuanSong, BreakPct: 0.15},
	{AttackerType: FormationTypeWuXing, DefenderType: FormationTypeTianGangBeiDou, BreakPct: 0.10},
	{AttackerType: FormationTypeBaGua, DefenderType: FormationTypeTianGangBeiDou, BreakPct: 0.10},
	{AttackerType: FormationTypeZhuXian, DefenderType: FormationTypeTianGangBeiDou, BreakPct: 0.10},
	{AttackerType: FormationTypeHuShan, DefenderType: FormationTypeTianGangBeiDou, BreakPct: 0.10},
	{AttackerType: FormationTypeJuLing, DefenderType: FormationTypeTianGangBeiDou, BreakPct: 0.10},
	{AttackerType: FormationTypeMiHun, DefenderType: FormationTypeTianGangBeiDou, BreakPct: 0.10},
	{AttackerType: FormationTypeChuanSong, DefenderType: FormationTypeTianGangBeiDou, BreakPct: 0.10},
	// 传送阵没有强克，但被微弱克
	{AttackerType: FormationTypeChuanSong, DefenderType: FormationTypeMiHun, BreakPct: 0.15},
}

// ============================================================
// 数据结构
// ============================================================

// FormationEffect 阵法效果
type FormationEffect struct {
	Type  string  `json:"type"`  // atk/def/hp/crit_rate/dodge/cultivation_speed/regen_hp/...
	Value float64 `json:"value"` // 效果数值（百分比，如 0.15 表示 15%）
}

// FormationTemplate 阵法图谱（静态模板数据）
type FormationTemplate struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Type        int               `json:"type"`        // 1-8
	Quality     int               `json:"quality"`     // 基础品质 1-5
	Description string            `json:"description"` // 阵法描述
	Effects     []FormationEffect `json:"effects"`     // 基础效果
	LearnCost   int64             `json:"learn_cost"`  // 学习消耗灵石
}

// Formation 玩家已习得的阵法
type Formation struct {
	ID           int64             `json:"id" gorm:"primaryKey"`
	PlayerID     int64             `json:"player_id"`
	TmplID       int               `json:"tmpl_id"`
	Name         string            `json:"name"`
	Type         int               `json:"type"`                    // 1-8
	Level        int               `json:"level" gorm:"default:1"`  // 等级 1-10
	Quality      int               `json:"quality" gorm:"default:1"` // 品质 1-5
	Deployed     bool              `json:"deployed"`
	Guardian     bool              `json:"guardian"`
	Exp          int64             `json:"exp"`
	Effects      []FormationEffect `json:"effects" gorm:"-"`
	LearnedAt    time.Time         `json:"learned_at"`
	// === 新增字段 ===
	MasteryExp   int64 `json:"mastery_exp"`    // 熟练度经验
	MasteryLevel int   `json:"mastery_level"`  // 熟练度等级 0-10
	GuardianPetID *int64 `json:"guardian_pet_id,omitempty"` // 守护灵兽ID（nil=未指派）
	LinkGroup    int   `json:"link_group"`      // 联动组编号（0=未联动）
}

// FormationResponse 阵法响应（含额外展示字段）
type FormationResponse struct {
	Formation            *Formation `json:"formation"`
	TypeName             string     `json:"type_name"`
	QualityName          string     `json:"quality_name"`
	DeployIdx            int        `json:"deploy_idx,omitempty"`
	MasteryLevelName    string     `json:"mastery_level_name,omitempty"`
	MasteryProgress     float64    `json:"mastery_progress,omitempty"` // 0.0-1.0
	SynergyBonus        string     `json:"synergy_bonus,omitempty"`   // 联动加成描述
	HasGuardian         bool       `json:"has_guardian,omitempty"`
	GuardianPetName     string     `json:"guardian_pet_name,omitempty"`
	GuardianContribution float64   `json:"guardian_contribution,omitempty"` // 守护灵兽贡献的额外效果百分比
}

// ============================================================
// 守护灵兽相关
// ============================================================

// FormationGuardian 阵法守护灵兽映射
type FormationGuardian struct {
	FormationID int64  `json:"formation_id"`
	PetID       int64  `json:"pet_id"`
	PetName     string `json:"pet_name"`
	PetStar     int    `json:"pet_star"`
	PetLevel    int    `json:"pet_level"`
	// 基于灵兽属性贡献的额外效果值
	AtkBonus    float64 `json:"atk_bonus"`
	DefBonus    float64 `json:"def_bonus"`
	HPBonus     float64 `json:"hp_bonus"`
	OtherBonus  float64 `json:"other_bonus"`
}

// ============================================================
// 联动（Linking）相关
// ============================================================

// FormationLinkResult 当前已部署阵法的联动计算结果
type FormationLinkResult struct {
	Deployed        []*Formation       `json:"deployed"`
	Synergies       []ActiveSynergy    `json:"synergies"`
	TotalAtkBonus   float64            `json:"total_atk_bonus"`
	TotalDefBonus   float64            `json:"total_def_bonus"`
	TotalOtherBonus float64            `json:"total_other_bonus"`
}

// ActiveSynergy 已激活的联动
type ActiveSynergy struct {
	TypeA    int    `json:"type_a"`
	TypeB    int    `json:"type_b"`
	Name     string `json:"name"`
	Bonus    string `json:"bonus"`
	Level    int    `json:"level"` // 联动等级（取两个阵法熟练度等级之和）
	Mult     float64 `json:"mult"`
}

// ============================================================
// 破阵相关
// ============================================================

// FormationBreakResult PVP 破阵结果
type FormationBreakResult struct {
	AttackerID    int64              `json:"attacker_id"`
	DefenderID    int64              `json:"defender_id"`
	AttackerFms   []*Formation       `json:"attacker_formations"`
	DefenderFms   []*Formation       `json:"defender_formations"`
	Breaks        []SingleBreak      `json:"breaks"`
	TotalReduction float64           `json:"total_reduction"` // 防守方总效果降低百分比
	BonusActive   bool               `json:"bonus_active"`    // 攻击方是否获得加成
}

// SingleBreak 单次破阵记录
type SingleBreak struct {
	AttackerType int     `json:"attacker_type"`
	DefenderType int     `json:"defender_type"`
	BreakPct     float64 `json:"break_pct"`
	DefenderName string  `json:"defender_name"`
}

// ============================================================
// 护法（突破加持）— 已有，保留
// ============================================================

// GuardianTask 护法任务记录
type GuardianTask struct {
	ID            int64     `json:"id" gorm:"primaryKey"`
	GuardianID    int64     `json:"guardian_id"`
	BeneficiaryID int64     `json:"beneficiary_id"`
	FormationID   int64     `json:"formation_id"`
	BonusRate     float64   `json:"bonus_rate"`
	Success       bool      `json:"success"`
	CreatedAt     time.Time `json:"created_at"`
}

// ============================================================
// 辅助计算函数
// ============================================================

// CalcMasteryExpRequired 计算升到下一级熟练度所需经验
func CalcMasteryExpRequired(currentLevel int) int64 {
	if currentLevel >= MasteryLevelMax {
		return 0
	}
	return int64(MasteryExpPerLevel * (currentLevel + 1))
}

// CalcMasteryMultiplier 根据熟练度计算效果倍率
// 每级提升 5%，从 1.0 开始，最高 1.5
func CalcMasteryMultiplier(masteryLevel int) float64 {
	if masteryLevel < 0 {
		masteryLevel = 0
	}
	if masteryLevel > MasteryLevelMax {
		masteryLevel = MasteryLevelMax
	}
	return 1.0 + float64(masteryLevel)*0.05
}

// CalcSynergyLevel 计算两个阵法的联动等级（平均熟练度等级向下取整）
func CalcSynergyLevel(mlA, mlB int) int {
	return (mlA + mlB) / 2
}

// CalcSynergyMultiplier 根据联动等级计算联动效果倍率（每级 10%）
func CalcSynergyMultiplier(synergyLevel int) float64 {
	return 1.0 + float64(synergyLevel)*0.10
}

// FindSynergy 查找两个阵法类型是否有预定义的联动
func FindSynergy(typeA, typeB int) *FormationSynergy {
	for _, s := range PredefinedSynergies {
		if (s.TypeA == typeA && s.TypeB == typeB) ||
			(s.TypeA == typeB && s.TypeB == typeA) {
			return &s
		}
	}
	return nil
}

// FindCounter 查找克制关系
func FindCounter(attackerType, defenderType int) *FormationCounter {
	for _, c := range PredefinedCounters {
		if c.AttackerType == attackerType && c.DefenderType == defenderType {
			return &c
		}
	}
	return nil
}

// CalcGuardianPetContribution 计算灵兽作为阵法守护者的贡献值
// 基于灵兽的攻击、防御、星级和等级
func CalcGuardianPetContribution(petAtk, petDef int64, petStar, petLevel int) float64 {
	base := float64(petAtk+petDef) * 0.0001
	starMult := 1.0 + float64(petStar-1)*0.2
	levelMult := 1.0 + float64(petLevel)*0.005
	return base * starMult * levelMult
}
