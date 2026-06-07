package model

import "time"

// 轮回次数上限
const (
	MaxRebirthCount        = 9  // 最多轮回9次
	SpeedBonusPerRebirth   = 20 // 每轮回修炼速度+20%
	MaxSpiritRootQuality   = 4  // 灵根品质上限（极品）
	CarryOverRatio         = 5  // 每次轮回继承5%前世修为
	TalentPointPerRebirth  = 3  // 每次轮回获得3天赋点
)

// 天赋树分支
const (
	TalentBranchAttack      = "attack"      // 攻击系
	TalentBranchDefense     = "defense"     // 防御系
	TalentBranchCultivation = "cultivation" // 修炼系
	TalentBranchLuck        = "luck"        // 气运系
)

// TalentBranchNames 天赋分支中文名
var TalentBranchNames = map[string]string{
	TalentBranchAttack:      "杀戮之道",
	TalentBranchDefense:     "不动明王",
	TalentBranchCultivation: "天道感悟",
	TalentBranchLuck:        "气运加身",
}

// TalentTreeNode 天赋节点定义
type TalentTreeNode struct {
	ID          string  `json:"id"`           // 唯一标识 branch_slot
	Branch      string  `json:"branch"`       // 所属分支
	Slot        int     `json:"slot"`         // 分支内序号 0-4
	Name        string  `json:"name"`         // 名称
	Description string  `json:"description"`  // 描述
	MaxLevel    int     `json:"max_level"`    // 最大等级(大部分1级)
	EffectType  string  `json:"effect_type"`  // 效果类型: attack_pct / defense_pct / speed_pct / luck_pct / crit_pct / dodge_pct
	EffectValue float64 `json:"effect_value"` // 每级效果值(百分比)
	PreReqSlot  int     `json:"pre_req_slot"` // 前置天赋slot(-1表示无前置)
	CostPoints  int     `json:"cost_points"`  // 消耗天赋点数
}

// TalentTree 天赋树预制数据
var TalentTree = map[string][]TalentTreeNode{
	TalentBranchAttack: {
		{ID: "attack_0", Branch: TalentBranchAttack, Slot: 0, Name: "破甲", Description: "攻击时无视目标5%防御", MaxLevel: 1, EffectType: "armor_pierce_pct", EffectValue: 5, PreReqSlot: -1, CostPoints: 1},
		{ID: "attack_1", Branch: TalentBranchAttack, Slot: 1, Name: "狂暴", Description: "暴击伤害提升10%", MaxLevel: 3, EffectType: "crit_dmg_pct", EffectValue: 10, PreReqSlot: 0, CostPoints: 1},
		{ID: "attack_2", Branch: TalentBranchAttack, Slot: 2, Name: "连击", Description: "攻击时有15%概率额外攻击一次", MaxLevel: 1, EffectType: "double_attack_pct", EffectValue: 15, PreReqSlot: 0, CostPoints: 2},
		{ID: "attack_3", Branch: TalentBranchAttack, Slot: 3, Name: "会心", Description: "暴击率提升8%", MaxLevel: 3, EffectType: "crit_rate_pct", EffectValue: 8, PreReqSlot: 1, CostPoints: 1},
		{ID: "attack_4", Branch: TalentBranchAttack, Slot: 4, Name: "灭世", Description: "最终伤害提升15%", MaxLevel: 1, EffectType: "final_dmg_pct", EffectValue: 15, PreReqSlot: 3, CostPoints: 3},
	},
	TalentBranchDefense: {
		{ID: "defense_0", Branch: TalentBranchDefense, Slot: 0, Name: "铁骨", Description: "基础防御提升10%", MaxLevel: 3, EffectType: "defense_pct", EffectValue: 10, PreReqSlot: -1, CostPoints: 1},
		{ID: "defense_1", Branch: TalentBranchDefense, Slot: 1, Name: "护体", Description: "受到伤害降低5%", MaxLevel: 1, EffectType: "dmg_reduce_pct", EffectValue: 5, PreReqSlot: 0, CostPoints: 1},
		{ID: "defense_2", Branch: TalentBranchDefense, Slot: 2, Name: "反震", Description: "受到攻击时反弹8%伤害", MaxLevel: 1, EffectType: "thorns_pct", EffectValue: 8, PreReqSlot: 0, CostPoints: 2},
		{ID: "defense_3", Branch: TalentBranchDefense, Slot: 3, Name: "不灭", Description: "生命值低于30%时减伤提升10%", MaxLevel: 2, EffectType: "low_hp_protect_pct", EffectValue: 10, PreReqSlot: 1, CostPoints: 1},
		{ID: "defense_4", Branch: TalentBranchDefense, Slot: 4, Name: "金刚", Description: "每回合获得等同防御5%的护盾", MaxLevel: 1, EffectType: "shield_pct", EffectValue: 5, PreReqSlot: 3, CostPoints: 3},
	},
	TalentBranchCultivation: {
		{ID: "cultivation_0", Branch: TalentBranchCultivation, Slot: 0, Name: "悟性", Description: "修炼速度提升10%", MaxLevel: 3, EffectType: "cult_speed_pct", EffectValue: 10, PreReqSlot: -1, CostPoints: 1},
		{ID: "cultivation_1", Branch: TalentBranchCultivation, Slot: 1, Name: "明悟", Description: "突破时消耗修为减少5%", MaxLevel: 3, EffectType: "breakthrough_cost_reduce_pct", EffectValue: 5, PreReqSlot: 0, CostPoints: 1},
		{ID: "cultivation_2", Branch: TalentBranchCultivation, Slot: 2, Name: "顿悟", Description: "修炼时有10%概率获得双倍修为", MaxLevel: 1, EffectType: "double_cult_rate_pct", EffectValue: 10, PreReqSlot: 0, CostPoints: 2},
		{ID: "cultivation_3", Branch: TalentBranchCultivation, Slot: 3, Name: "天资", Description: "灵根修炼效率提升15%", MaxLevel: 2, EffectType: "root_efficiency_pct", EffectValue: 15, PreReqSlot: 1, CostPoints: 1},
		{ID: "cultivation_4", Branch: TalentBranchCultivation, Slot: 4, Name: "天人合一", Description: "所有修炼效率提升25%", MaxLevel: 1, EffectType: "all_cult_rate_pct", EffectValue: 25, PreReqSlot: 3, CostPoints: 3},
	},
	TalentBranchLuck: {
		{ID: "luck_0", Branch: TalentBranchLuck, Slot: 0, Name: "福缘", Description: "探索获得奖励提升10%", MaxLevel: 3, EffectType: "explore_reward_pct", EffectValue: 10, PreReqSlot: -1, CostPoints: 1},
		{ID: "luck_1", Branch: TalentBranchLuck, Slot: 1, Name: "寻宝", Description: "稀有物品掉率提升8%", MaxLevel: 2, EffectType: "rare_drop_rate_pct", EffectValue: 8, PreReqSlot: 0, CostPoints: 1},
		{ID: "luck_2", Branch: TalentBranchLuck, Slot: 2, Name: "避凶", Description: "心魔劫难度降低10%", MaxLevel: 1, EffectType: "demon_trib_reduce_pct", EffectValue: 10, PreReqSlot: 0, CostPoints: 2},
		{ID: "luck_3", Branch: TalentBranchLuck, Slot: 3, Name: "天眷", Description: "天劫伤害降低12%", MaxLevel: 2, EffectType: "tribulation_dmg_reduce_pct", EffectValue: 12, PreReqSlot: 1, CostPoints: 1},
		{ID: "luck_4", Branch: TalentBranchLuck, Slot: 4, Name: "逆天改命", Description: "每次轮回额外保留10%修为", MaxLevel: 1, EffectType: "extra_carry_over_pct", EffectValue: 10, PreReqSlot: 3, CostPoints: 3},
	},
}

// 轮回称号
const (
	RebirthTitleMortal       = "一世散修" // 0次
	RebirthTitleOne          = "二世转生" // 1次
	RebirthTitleTwo          = "三世轮回" // 2次
	RebirthTitleThree        = "四世重生" // 3次
	RebirthTitleFour         = "五世道者" // 4次
	RebirthTitleFive         = "六世真人" // 5次
	RebirthTitleSix          = "七世天君" // 6次
	RebirthTitleSeven        = "八世至尊" // 7次
	RebirthTitleEight        = "九世仙人" // 8次
	RebirthTitleNine         = "十世圆满" // 9次
)

// RebirthTitleNames 轮回称号映射
var RebirthTitleNames = map[int]string{
	0: RebirthTitleMortal,
	1: RebirthTitleOne,
	2: RebirthTitleTwo,
	3: RebirthTitleThree,
	4: RebirthTitleFour,
	5: RebirthTitleFive,
	6: RebirthTitleSix,
	7: RebirthTitleSeven,
	8: RebirthTitleEight,
	9: RebirthTitleNine,
}

// RebirthTitleBonuses 称号属性加成
var RebirthTitleBonuses = map[int]struct {
	AttackPct  float64 `json:"attack_pct"`
	DefensePct float64 `json:"defense_pct"`
	HPPct      float64 `json:"hp_pct"`
	SpeedPct   float64 `json:"speed_pct"`
}{
	0: {AttackPct: 0, DefensePct: 0, HPPct: 0, SpeedPct: 0},
	1: {AttackPct: 2, DefensePct: 2, HPPct: 2, SpeedPct: 5},
	2: {AttackPct: 5, DefensePct: 5, HPPct: 5, SpeedPct: 10},
	3: {AttackPct: 8, DefensePct: 8, HPPct: 8, SpeedPct: 15},
	4: {AttackPct: 12, DefensePct: 12, HPPct: 12, SpeedPct: 20},
	5: {AttackPct: 16, DefensePct: 16, HPPct: 16, SpeedPct: 25},
	6: {AttackPct: 20, DefensePct: 20, HPPct: 20, SpeedPct: 30},
	7: {AttackPct: 25, DefensePct: 25, HPPct: 25, SpeedPct: 35},
	8: {AttackPct: 30, DefensePct: 30, HPPct: 30, SpeedPct: 40},
	9: {AttackPct: 40, DefensePct: 40, HPPct: 40, SpeedPct: 50},
}

// 灵根品质
const (
	SpiritRootQualityLow      = 1 // 下品
	SpiritRootQualityMedium   = 2 // 中品
	SpiritRootQualityHigh     = 3 // 上品
	SpiritRootQualityPerfect  = 4 // 极品
)

// SpiritRootQualityNames 灵根品质中文名
var SpiritRootQualityNames = map[int]string{
	SpiritRootQualityLow:     "下品",
	SpiritRootQualityMedium:  "中品",
	SpiritRootQualityHigh:    "上品",
	SpiritRootQualityPerfect: "极品",
}

// RebirthShopItem 轮回商店物品
type RebirthShopItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int    `json:"price"`        // 轮回币价格
	ItemType    string `json:"item_type"`    // item/skill/effect
	ItemID      string `json:"item_id"`      // 对应物品/技能ID
	EffectValue int    `json:"effect_value"` // 效果值
	MaxPurchase int    `json:"max_purchase"` // 限购次数(0=不限)
	RebirthReq  int    `json:"rebirth_req"`  // 最低轮回次数要求
}

// RebirthShopItems 轮回商店商品列表
var RebirthShopItems = []RebirthShopItem{
	{ID: "shop_rebirth_stone", Name: "转生石", Description: "减少一次轮回CD", Price: 50, ItemType: "effect", ItemID: "rebirth_cd_reset", EffectValue: 1, MaxPurchase: 1, RebirthReq: 1},
	{ID: "shop_soul_jade", Name: "魂玉", Description: "永久提升攻击力5%", Price: 100, ItemType: "effect", ItemID: "perm_attack_pct", EffectValue: 5, MaxPurchase: 3, RebirthReq: 1},
	{ID: "shop_soul_shield", Name: "魂盾", Description: "永久提升防御力5%", Price: 100, ItemType: "effect", ItemID: "perm_defense_pct", EffectValue: 5, MaxPurchase: 3, RebirthReq: 1},
	{ID: "shop_soul_essence", Name: "精粹", Description: "永久提升修炼速度3%", Price: 80, ItemType: "effect", ItemID: "perm_speed_pct", EffectValue: 3, MaxPurchase: 5, RebirthReq: 1},
	{ID: "shop_talent_reset", Name: "洗髓丹", Description: "重置所有天赋点", Price: 200, ItemType: "effect", ItemID: "talent_reset", EffectValue: 1, MaxPurchase: 0, RebirthReq: 2},
	{ID: "shop_ancient_skill", Name: "上古秘术·残卷", Description: "获得一本随机上古功法", Price: 500, ItemType: "item", ItemID: "ancient_skill_scroll", EffectValue: 1, MaxPurchase: 1, RebirthReq: 3},
	{ID: "shop_carry_amulet", Name: "轮回护符", Description: "下次轮回额外继承10%修为", Price: 300, ItemType: "effect", ItemID: "carry_over_boost", EffectValue: 10, MaxPurchase: 3, RebirthReq: 2},
	{ID: "shop_destiny_fragment", Name: "命运碎片", Description: "永久提升气运(暴击率+2%,闪避率+2%)", Price: 150, ItemType: "effect", ItemID: "perm_luck_pct", EffectValue: 2, MaxPurchase: 3, RebirthReq: 2},
	{ID: "shop_trib_amulets", Name: "渡劫符箓", Description: "渡劫成功率+15%", Price: 120, ItemType: "item", ItemID: "tribulation_amulet", EffectValue: 15, MaxPurchase: 5, RebirthReq: 1},
	{ID: "shop_mythical_blood", Name: " mythical 血脉", Description: "永久提升生命上限10%", Price: 200, ItemType: "effect", ItemID: "perm_hp_pct", EffectValue: 10, MaxPurchase: 2, RebirthReq: 3},
}

// PlayerRebirth 轮回记录（每个玩家一条）
type PlayerRebirth struct {
	ID                    int64     `json:"id" gorm:"primaryKey"`
	PlayerID              int64     `json:"player_id" gorm:"uniqueIndex;not null"`
	RebirthCount          int       `json:"rebirth_count" gorm:"default:0"`            // 轮回次数 0-9
	Enlightenment         int       `json:"enlightenment" gorm:"default:0"`            // 天道感悟层数（每轮回+1）
	SpiritRootQuality     int       `json:"spirit_root_quality" gorm:"default:1"`      // 灵根品质 1-4
	CultivationSpeedBonus int       `json:"cultivation_speed_bonus" gorm:"default:0"`  // 修炼速度加成%
	Title                 string    `json:"title" gorm:"size:32;default:'一世散修'"`
	RebirthJade           int64     `json:"rebirth_jade" gorm:"default:0"`             // 轮回币
	TalentPoints          int       `json:"talent_points" gorm:"default:0"`            // 可用天赋点数
	CarryOverPower        int64     `json:"carry_over_power" gorm:"default:0"`         // 继承的修为
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// PlayerTalent 已学习天赋
type PlayerTalent struct {
	ID         int64     `json:"id" gorm:"primaryKey"`
	PlayerID   int64     `json:"player_id" gorm:"index;not null"`
	TalentID   string    `json:"talent_id" gorm:"size:32;not null"` // 对应TalentTreeNode.ID
	Level      int       `json:"level" gorm:"default:1"`            // 当前等级
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// RebirthShopPurchase 轮回商店购买记录
type RebirthShopPurchase struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	PlayerID  int64     `json:"player_id" gorm:"index;not null"`
	ShopID    string    `json:"shop_id" gorm:"size:32;not null"` // 对应RebirthShopItem.ID
	Count     int       `json:"count" gorm:"default:1"`          // 已购买次数
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RebirthHistory 轮回历史（每次轮回一条）
type RebirthHistory struct {
	ID                 int64     `json:"id" gorm:"primaryKey"`
	PlayerID           int64     `json:"player_id" gorm:"index;not null"`
	RebirthNumber      int       `json:"rebirth_number"`           // 第几次轮回
	OldRealm           int32     `json:"old_realm"`                // 轮回前境界ID
	OldRealmLevel      int32     `json:"old_realm_level"`          // 轮回前等级
	OldAttack          int64     `json:"old_attack"`               // 轮回前攻击力
	OldDefense         int64     `json:"old_defense"`              // 轮回前防御力
	OldHP              int64     `json:"old_hp"`                   // 轮回前气血
	CarryOverAttack    int64     `json:"carry_over_attack"`        // 继承攻击
	CarryOverDefense   int64     `json:"carry_over_defense"`       // 继承防御
	CarryOverHP        int64     `json:"carry_over_hp"`            // 继承气血
	GoldKept           int64     `json:"gold_kept"`                // 保留灵石数
	SpiritRootBefore   int32     `json:"spirit_root_before"`       // 轮回前灵根
	SpiritRootAfter    int32     `json:"spirit_root_after"`        // 轮回后灵根
	QualityBefore      int       `json:"quality_before"`           // 轮回前灵根品质
	QualityAfter       int       `json:"quality_after"`            // 轮回后灵根品质
	RebirthJadeEarned  int64     `json:"rebirth_jade_earned"`      // 获得轮回币
	TitleEarned        string    `json:"title_earned"`             // 获得称号
	CreatedAt          time.Time `json:"created_at"`
}

// RebirthBenefits 轮回福利展示
type RebirthBenefits struct {
	EnlightenmentPerRebirth  int      `json:"enlightenment_per_rebirth"`  // 每次轮回获得的天道感悟层数
	SpeedBonusPerRebirth     int      `json:"speed_bonus_per_rebirth"`    // 每次轮回修炼速度加成%
	MaxRebirthCount          int      `json:"max_rebirth_count"`          // 最大轮回次数
	GoldRetentionRate        string   `json:"gold_retention_rate"`        // 灵石保留比例
	DongFuLevelAfter         int      `json:"dongfu_level_after"`         // 轮回后洞府等级
	ArtifactLevelAfter       int      `json:"artifact_level_after"`       // 轮回后法宝等级
	SpiritRootQualityUpgrade string   `json:"spirit_root_quality_upgrade"` // 灵根品质提升描述
	Titles                   []string `json:"titles"`                     // 各轮回称号
	TalentPointPerRebirth    int      `json:"talent_point_per_rebirth"`    // 每次轮回获得天赋点
	CarryOverPercent         int      `json:"carry_over_percent"`         // 继承比例
}

// RebirthCheckResponse 检查轮回响应
type RebirthCheckResponse struct {
	CanRebirth           bool            `json:"can_rebirth"`           // 能否轮回
	RebirthCount         int             `json:"rebirth_count"`         // 当前轮回次数
	MaxRebirthCount      int             `json:"max_rebirth_count"`     // 最大轮回次数
	CurrentTitle         string          `json:"current_title"`         // 当前称号
	CurrentTitleBonuses  struct {
		AttackPct  float64 `json:"attack_pct"`
		DefensePct float64 `json:"defense_pct"`
		HPPct      float64 `json:"hp_pct"`
		SpeedPct   float64 `json:"speed_pct"`
	} `json:"current_title_bonuses"`
	Enlightenment          int    `json:"enlightenment"`           // 天道感悟层数
	SpiritRootQuality      int    `json:"spirit_root_quality"`     // 当前灵根品质
	SpiritRootQualityName  string `json:"spirit_root_quality_name"`
	CultivationSpeedBonus  int    `json:"cultivation_speed_bonus"` // 修炼速度加成%
	RebirthJade            int64  `json:"rebirth_jade"`            // 轮回币
	TalentPoints           int    `json:"talent_points"`           // 可用天赋点
	Condition              string `json:"condition"`               // 轮回条件说明
	CooldownUntil          *time.Time `json:"cooldown_until,omitempty"` // 冷却到期时间
	Benefits               *RebirthBenefits `json:"benefits"`      // 轮回福利预览
}

// TalentInfoResponse 天赋信息响应
type TalentInfoResponse struct {
	TalentPoints     int                        `json:"talent_points"`     // 可用天赋点
	TalentTree       map[string][]TalentTreeNode  `json:"talent_tree"`      // 天赋树定义
	LearnedTalents   []PlayerTalent              `json:"learned_talents"`  // 已学天赋
	TalentBonuses    map[string]float64          `json:"talent_bonuses"`   // 天赋总加成
}

// RebirthShopResponse 轮回商店响应
type RebirthShopResponse struct {
	RebirthJade int64            `json:"rebirth_jade"` // 持有轮回币
	Items       []RebirthShopItem `json:"items"`       // 商品列表
	Purchases   []RebirthShopPurchase `json:"purchases"` // 已购买记录
}
