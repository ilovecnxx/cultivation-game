package model

import "time"

// ============================================================
// 装备套装系统 (Equipment Set System)
// ============================================================

// EquipmentSet 装备套装模板（静态配置）
type EquipmentSet struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Pieces   []int64    `json:"pieces"`    // item template IDs
	Bonuses  []SetBonus `json:"bonuses"`   // bonuses per piece count
	Quality  int        `json:"quality"`   // 1-5 stars
	Element  string     `json:"element"`   // 五行属性
	Icon     string     `json:"icon"`
	Lore     string     `json:"lore"`
}

// SetBonus 套装效果
type SetBonus struct {
	PiecesRequired int         `json:"pieces_required"`
	Effects        []SetEffect `json:"effects"`
	Description    string      `json:"description"`
}

// SetEffect 单条效果
type SetEffect struct {
	Stat      string  `json:"stat"`
	Value     float64 `json:"value"`
	IsPercent bool    `json:"is_percent"`
	Special   string  `json:"special,omitempty"`
}

// ActiveSetBonus 已激活的套装效果
type ActiveSetBonus struct {
	SetID          string     `json:"set_id"`
	SetName        string     `json:"set_name"`
	SetQuality     int        `json:"set_quality"`
	SetElement     string     `json:"set_element"`
	SetIcon        string     `json:"set_icon"`
	PiecesEquipped int        `json:"pieces_equipped"`
	PiecesTotal    int        `json:"pieces_total"`
	Bonuses        []SetBonus `json:"bonuses"`
	IsActive       bool       `json:"is_active"`
}

// CombatStats 战斗属性（含套装加成）
type CombatStats struct {
	Attack          float64 `json:"attack"`
	Defense         float64 `json:"defense"`
	HP              float64 `json:"hp"`
	Speed           float64 `json:"speed"`
	CritRate        float64 `json:"crit_rate"`
	CritDmg         float64 `json:"crit_dmg"`
	DodgeRate       float64 `json:"dodge_rate"`
	Lifesteal       float64 `json:"lifesteal"`
	ArmorPen        float64 `json:"armor_pen"`
	DamageBonus     float64 `json:"damage_bonus"`
	DamageReduction float64 `json:"damage_reduction"`
	FireDmg         float64 `json:"fire_dmg"`
	Breakthrough    float64 `json:"breakthrough"`
	Luck            float64 `json:"luck"`
	ExpBonus        float64 `json:"exp_bonus"`
	ExtraActions    int     `json:"extra_actions"`
	ReflectDmg      float64 `json:"reflect_dmg"`
	HpRegen         float64 `json:"hp_regen"`
}

// SetProgress 套装收集进度
type SetProgress struct {
	SetID        string   `json:"set_id"`
	SetName      string   `json:"set_name"`
	SetElement   string   `json:"set_element"`
	SetIcon      string   `json:"set_icon"`
	Equipped     []int64  `json:"equipped"`      // 已装备的item IDs
	EquippedNames []string `json:"equipped_names"`
	Missing      []int64  `json:"missing"`       // 缺少的item IDs
	MissingNames  []string `json:"missing_names"`
	MissingHints []string `json:"missing_hints"`
	PiecesCount  int      `json:"pieces_count"`
	TotalPieces  int      `json:"total_pieces"`
}

// ============================================================
// 附魔系统 (Enchantment System)
// ============================================================

// EnchantmentDef 附魔模板
type EnchantmentDef struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	TargetSlots []int32        `json:"target_slots"` // equipment slots
	Stats       []StatBonus    `json:"stats"`
	Cost        int64          `json:"cost"`           // spirit stones
	Materials   []MaterialCost `json:"materials"`      // required items
	SuccessRate float64        `json:"success_rate"`   // base 30-90%
	MaxLevel    int            `json:"max_level"`
	Icon        string         `json:"icon"`
	Description string         `json:"description"`
}

// StatBonus 属性加成
type StatBonus struct {
	Stat     string  `json:"stat"`
	Value    float64 `json:"value"`
	PerLevel float64 `json:"per_level"`
}

// MaterialCost 材料消耗
type MaterialCost struct {
	ItemID   int64  `json:"item_id"`
	Quantity int32  `json:"quantity"`
	Name     string `json:"name"`
}

// PlayerEnchantment 玩家附魔记录
type PlayerEnchantment struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	PlayerID    int64     `json:"player_id" gorm:"index;not null"`
	EquipmentID int64     `json:"equipment_id" gorm:"index;not null"`
	EnchantID   string    `json:"enchant_id" gorm:"size:32;not null"`
	Level       int       `json:"level" gorm:"default:1"`
	SlotIndex   int       `json:"slot_index" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EnchantmentInstance 附魔实例（前端展示用）
type EnchantmentInstance struct {
	ID          int64       `json:"id"`
	EnchantID   string      `json:"enchant_id"`
	Name        string      `json:"name"`
	Level       int         `json:"level"`
	SlotIndex   int         `json:"slot_index"`
	Stats       []StatBonus `json:"stats"`
	Description string      `json:"description"`
	Icon        string      `json:"icon"`
}

// ============================================================
// 装备觉醒 (Equipment Awakening)
// ============================================================

// EquipmentAwakening 装备觉醒记录
type EquipmentAwakening struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	PlayerID    int64     `json:"player_id" gorm:"index;not null"`
	EquipmentID int64     `json:"equipment_id" gorm:"uniqueIndex;not null"`
	AwakenLevel int       `json:"awaken_level" gorm:"default:1"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AwakenCost 觉醒消耗
type AwakenCost struct {
	SpiritStones int64          `json:"spirit_stones"`
	Materials    []MaterialCost `json:"materials"`
	Level        int            `json:"level"`
}

// ============================================================
// 请求/响应
// ============================================================

// EnchantRequest 附魔请求
type EnchantRequest struct {
	PlayerID    int64  `json:"player_id"`
	EquipmentID int64  `json:"equipment_id" binding:"required"`
	EnchantID   string `json:"enchant_id" binding:"required"`
}

// EnchantRemoveRequest 移除附魔请求
type EnchantRemoveRequest struct {
	PlayerID    int64 `json:"player_id"`
	EquipmentID int64 `json:"equipment_id" binding:"required"`
	EnchantSlot int   `json:"enchant_slot" binding:"required,min=0,max=2"`
}

// AwakenRequest 觉醒请求
type AwakenRequest struct {
	PlayerID    int64 `json:"player_id"`
	EquipmentID int64 `json:"equipment_id" binding:"required"`
}

// EquipmentDetail 装备详细信息
type EquipmentDetail struct {
	Equipment    *Equipment            `json:"equipment"`
	SetInfo      *ActiveSetBonus       `json:"set_info,omitempty"`
	Enchantments []*EnchantmentInstance `json:"enchantments,omitempty"`
	Awakening    *EquipmentAwakening   `json:"awakening,omitempty"`
	TotalStats   *CombatStats          `json:"total_stats,omitempty"`
}

// EnchantSlotGroup 附魔槽位分组
type EnchantSlotGroup struct {
	GroupID  string  `json:"group_id"`
	Name     string  `json:"name"`
	Slots    []int32 `json:"slots"`
	Enchants []EnchantmentDef `json:"enchants"`
}

// ============================================================
// 预定义装备套装
// ============================================================

var EquipmentSetList = []EquipmentSet{
	{
		ID: "azure_dragon", Name: "青龙套装", Quality: 5, Element: "木",
		Icon: "🐉", Pieces: []int64{2001, 2002, 2003, 2004, 2005},
		Lore: "东方青龙，四象之首，蕴含无穷生机与力量。传说为天帝镇守东方的神兽之鳞所铸。",
		Bonuses: []SetBonus{
			{PiecesRequired: 2, Description: "攻击力 +10%", Effects: []SetEffect{{Stat: "attack", Value: 10, IsPercent: true}}},
			{PiecesRequired: 3, Description: "暴击伤害 +15%", Effects: []SetEffect{{Stat: "crit_dmg", Value: 15, IsPercent: true}}},
			{PiecesRequired: 5, Description: "青龙之魂：攻击时10%几率吸取造成伤害的50%为生命值", Effects: []SetEffect{{Stat: "lifesteal", Value: 10, IsPercent: true, Special: "青龙之魂"}}},
		},
	},
	{
		ID: "vermillion_bird", Name: "朱雀套装", Quality: 5, Element: "火",
		Icon: "🦅", Pieces: []int64{2006, 2007, 2008, 2009, 2010},
		Lore: "南方朱雀，浴火重生。蕴含南明离火之力的上古神装，得之可焚尽万物。",
		Bonuses: []SetBonus{
			{PiecesRequired: 2, Description: "速度 +10%", Effects: []SetEffect{{Stat: "speed", Value: 10, IsPercent: true}}},
			{PiecesRequired: 3, Description: "火焰伤害 +15%", Effects: []SetEffect{{Stat: "fire_dmg", Value: 15, IsPercent: true, Special: "朱雀之炎"}}},
			{PiecesRequired: 5, Description: "涅槃：每场战斗可复活一次，恢复50%生命值", Effects: []SetEffect{{Stat: "hp", Value: 50, IsPercent: true, Special: "涅槃重生"}}},
		},
	},
	{
		ID: "black_tortoise", Name: "玄武套装", Quality: 5, Element: "水",
		Icon: "🐢", Pieces: []int64{2011, 2012, 2013, 2014, 2015},
		Lore: "北方玄武，以防御称著。玄冰之甲，坚不可摧，乃天地间最坚固的防护。",
		Bonuses: []SetBonus{
			{PiecesRequired: 2, Description: "防御力 +15%", Effects: []SetEffect{{Stat: "defense", Value: 15, IsPercent: true}}},
			{PiecesRequired: 3, Description: "生命值上限 +20%", Effects: []SetEffect{{Stat: "hp", Value: 20, IsPercent: true}}},
			{PiecesRequired: 5, Description: "玄武盾：战斗开始获得30%伤害减免，持续3回合", Effects: []SetEffect{{Stat: "damage_reduction", Value: 30, IsPercent: true, Special: "玄武盾"}}},
		},
	},
	{
		ID: "white_tiger", Name: "白虎套装", Quality: 5, Element: "金",
		Icon: "🐯", Pieces: []int64{2016, 2017, 2018, 2019, 2020},
		Lore: "西方白虎，杀伐之气冠绝四方。庚金之气凝聚而成，锐不可当。",
		Bonuses: []SetBonus{
			{PiecesRequired: 2, Description: "暴击率 +10%", Effects: []SetEffect{{Stat: "crit_rate", Value: 10, IsPercent: true}}},
			{PiecesRequired: 3, Description: "暴击伤害 +20%", Effects: []SetEffect{{Stat: "crit_dmg", Value: 20, IsPercent: true}}},
			{PiecesRequired: 5, Description: "虎威：暴击时有30%几率眩晕敌人一回合", Effects: []SetEffect{{Stat: "crit_rate", Value: 0, IsPercent: false, Special: "虎威震慑"}}},
		},
	},
	{
		ID: "qilin", Name: "麒麟套装", Quality: 5, Element: "土",
		Icon: "🦄", Pieces: []int64{2021, 2022, 2023, 2024, 2025},
		Lore: "祥瑞之兽麒麟所化，蕴含大地之力与无尽祥瑞之气。得之者福缘深厚。",
		Bonuses: []SetBonus{
			{PiecesRequired: 2, Description: "全属性 +8%", Effects: []SetEffect{{Stat: "all_stats", Value: 8, IsPercent: true, Special: "祥瑞之兆"}}},
			{PiecesRequired: 3, Description: "突破成功率 +15%", Effects: []SetEffect{{Stat: "breakthrough", Value: 15, IsPercent: true}}},
			{PiecesRequired: 5, Description: "祥瑞：机缘值 +50", Effects: []SetEffect{{Stat: "luck", Value: 50, IsPercent: false, Special: "麒麟祥瑞"}}},
		},
	},
	{
		ID: "chaos", Name: "混沌套装", Quality: 5, Element: "无",
		Icon: "🌀", Pieces: []int64{2026, 2027, 2028, 2029, 2030},
		Lore: "混沌初开，鸿蒙未判之时诞生的原始力量。蕴藏无尽的毁灭与创造之力。",
		Bonuses: []SetBonus{
			{PiecesRequired: 2, Description: "伤害 +5%", Effects: []SetEffect{{Stat: "damage_bonus", Value: 5, IsPercent: true}}},
			{PiecesRequired: 3, Description: "护甲穿透 +10%", Effects: []SetEffect{{Stat: "armor_pen", Value: 10, IsPercent: true}}},
			{PiecesRequired: 5, Description: "混沌之力：每回合有20%几率发动一次额外攻击", Effects: []SetEffect{{Stat: "damage_bonus", Value: 0, IsPercent: false, Special: "混沌之力"}}},
		},
	},
	{
		ID: "void", Name: "太虚套装", Quality: 5, Element: "无",
		Icon: "🌌", Pieces: []int64{2031, 2032, 2033, 2034, 2035},
		Lore: "太虚之境，无形无相。蕴含空间法则的至宝，能令佩戴者穿梭虚空。",
		Bonuses: []SetBonus{
			{PiecesRequired: 2, Description: "闪避率 +12%", Effects: []SetEffect{{Stat: "dodge_rate", Value: 12, IsPercent: true}}},
			{PiecesRequired: 3, Description: "每回合额外行动1次", Effects: []SetEffect{{Stat: "extra_actions", Value: 1, IsPercent: false, Special: "太虚步"}}},
			{PiecesRequired: 5, Description: "虚空：免疫受到的第一次攻击", Effects: []SetEffect{{Stat: "dodge_rate", Value: 0, IsPercent: false, Special: "虚空之体"}}},
		},
	},
	{
		ID: "primordial", Name: "鸿蒙套装", Quality: 5, Element: "全",
		Icon: "☯️", Pieces: []int64{2036, 2037, 2038, 2039, 2040},
		Lore: "鸿蒙初开，大道之始。传说为开天辟地之前就已存在的至高神器。",
		Bonuses: []SetBonus{
			{PiecesRequired: 2, Description: "全属性 +15%", Effects: []SetEffect{{Stat: "all_stats", Value: 15, IsPercent: true, Special: "鸿蒙之气"}}},
			{PiecesRequired: 3, Description: "修炼经验获取 +25%", Effects: []SetEffect{{Stat: "exp_bonus", Value: 25, IsPercent: true}}},
			{PiecesRequired: 5, Description: "鸿蒙初开：所有套装效果翻倍", Effects: []SetEffect{{Stat: "all_stats", Value: 0, IsPercent: false, Special: "鸿蒙初开"}}},
		},
	},
}

// SetAcquisitionHints 套装获取提示
var SetAcquisitionHints = map[string][]string{
	"azure_dragon": {
		"青龙剑：通关青龙秘境第10层获得",
		"青龙战甲：青龙秘境首领掉落",
		"青龙头盔：完成青龙镇支线任务",
		"青龙战靴：青龙秘境宝箱概率开出",
		"青龙戒指：合成获得（需5枚青龙碎片）",
	},
	"vermillion_bird": {
		"朱雀羽扇：朱雀秘境第10层首通奖励",
		"朱雀法袍：朱雀秘境精英怪掉落",
		"朱雀头饰：天工阁声望达到尊敬兑换",
		"朱雀灵靴：朱雀秘境隐藏BOSS掉落",
		"朱雀项链：集齐5枚朱雀之羽合成",
	},
	"black_tortoise": {
		"玄武重剑：玄武秘境第10层首领掉落",
		"玄武甲胄：完成玄武城守卫系列任务",
		"玄武头盔：玄武秘境宝箱概率开出",
		"玄武护腿：世界BOSS玄武分身掉落",
		"玄武腰带：竞技场赛季奖励",
	},
	"white_tiger": {
		"白虎利爪：白虎秘境首领掉落",
		"白虎战甲：白虎秘境第10层首通奖励",
		"白虎战盔：猎杀榜积分达到5000兑换",
		"白虎战靴：白虎秘境疾风挑战通关",
		"白虎指环：集齐5枚白虎之牙合成",
	},
	"qilin": {
		"麒麟臂：完成千机阁所有机关挑战",
		"麒麟甲：洞天福地福缘事件获得",
		"麒麟角：世界事件祥瑞降临排名奖励",
		"麒麟之足：游历四方触发奇遇获得",
		"麒麟玉佩：集齐10枚祥瑞令牌兑换",
	},
	"chaos": {
		"混沌之刃：混沌深渊第10层首通奖励",
		"混沌战甲：混沌深渊精英怪掉落",
		"混沌头盔：混沌深渊隐藏房间宝箱",
		"混沌战靴：混沌深渊竞速榜第一奖励",
		"混沌指环：集齐5枚混沌精华合成",
	},
	"void": {
		"虚空之刃：虚空裂缝第10层通关奖励",
		"虚空法袍：虚空裂缝首领掉落",
		"虚空兜帽：虚空裂缝探索度100%奖励",
		"虚空之靴：虚空裂缝限时挑战奖励",
		"虚空吊坠：集齐5枚虚空结晶合成",
	},
	"primordial": {
		"鸿蒙剑：飞升试炼通关奖励",
		"鸿蒙甲：天道榜排名前三奖励",
		"鸿蒙冠：完成所有天书奇谭任务",
		"鸿蒙履：渡劫成功时有概率获得",
		"鸿蒙戒：集齐七件上古神器兑换",
	},
}

// ============================================================
// 预定义附魔
// ============================================================

// EnchantmentSlotGroups 附魔槽位分组
var EnchantmentSlotGroups = []EnchantSlotGroup{
	{
		GroupID: "weapon", Name: "武器附魔", Slots: []int32{EquipSlotWeapon, EquipSlotBracers},
		Enchants: []EnchantmentDef{
			{ID: "sharpen", Name: "锋利", TargetSlots: []int32{EquipSlotWeapon, EquipSlotBracers},
				Stats: []StatBonus{{Stat: "attack", Value: 10, PerLevel: 8}},
				Cost: 1000, SuccessRate: 0.75, MaxLevel: 10, Icon: "⚔️",
				Description: "提升武器锋利度，增加攻击力",
				Materials: []MaterialCost{{ItemID: 3001, Quantity: 2, Name: "磨刀石"}}},
			{ID: "gale", Name: "疾风", TargetSlots: []int32{EquipSlotWeapon, EquipSlotBracers},
				Stats: []StatBonus{{Stat: "speed", Value: 5, PerLevel: 3}},
				Cost: 1200, SuccessRate: 0.70, MaxLevel: 8, Icon: "💨",
				Description: "灌注风系灵力，提升攻击速度",
				Materials: []MaterialCost{{ItemID: 3002, Quantity: 2, Name: "风灵石"}}},
			{ID: "bloodthirst", Name: "嗜血", TargetSlots: []int32{EquipSlotWeapon, EquipSlotBracers},
				Stats: []StatBonus{{Stat: "lifesteal", Value: 1.5, PerLevel: 0.5}},
				Cost: 2000, SuccessRate: 0.50, MaxLevel: 5, Icon: "🩸",
				Description: "汲取敌人生命为己用，获得吸血效果",
				Materials: []MaterialCost{{ItemID: 3003, Quantity: 3, Name: "血魂石"}}},
			{ID: "armor_break", Name: "破甲", TargetSlots: []int32{EquipSlotWeapon, EquipSlotBracers},
				Stats: []StatBonus{{Stat: "armor_pen", Value: 2, PerLevel: 1.5}},
				Cost: 1500, SuccessRate: 0.60, MaxLevel: 6, Icon: "🔨",
				Description: "击破敌人护甲，提升护甲穿透",
				Materials: []MaterialCost{{ItemID: 3004, Quantity: 2, Name: "破甲锥"}}},
			{ID: "berserk", Name: "狂暴", TargetSlots: []int32{EquipSlotWeapon, EquipSlotBracers},
				Stats: []StatBonus{{Stat: "crit_rate", Value: 2, PerLevel: 1}},
				Cost: 1800, SuccessRate: 0.55, MaxLevel: 6, Icon: "🔥",
				Description: "激发狂暴战意，提升暴击率",
				Materials: []MaterialCost{{ItemID: 3005, Quantity: 3, Name: "狂暴符"}}},
			{ID: "deadly", Name: "致命", TargetSlots: []int32{EquipSlotWeapon, EquipSlotBracers},
				Stats: []StatBonus{{Stat: "crit_dmg", Value: 5, PerLevel: 3}},
				Cost: 2200, SuccessRate: 0.45, MaxLevel: 5, Icon: "💀",
				Description: "直击要害，大幅提升暴击伤害",
				Materials: []MaterialCost{{ItemID: 3006, Quantity: 3, Name: "致命毒液"}}},
		},
	},
	{
		GroupID: "armor", Name: "防具附魔", Slots: []int32{EquipSlotHelmet, EquipSlotArmor, EquipSlotBelt, EquipSlotLegs},
		Enchants: []EnchantmentDef{
			{ID: "toughness", Name: "坚韧", TargetSlots: []int32{EquipSlotHelmet, EquipSlotArmor, EquipSlotBelt, EquipSlotLegs},
				Stats: []StatBonus{{Stat: "defense", Value: 12, PerLevel: 8}},
				Cost: 1000, SuccessRate: 0.75, MaxLevel: 10, Icon: "🛡️",
				Description: "增强装备坚固度，提升防御力",
				Materials: []MaterialCost{{ItemID: 3011, Quantity: 2, Name: "强化纤维"}}},
			{ID: "vitality", Name: "活力", TargetSlots: []int32{EquipSlotHelmet, EquipSlotArmor, EquipSlotBelt, EquipSlotLegs},
				Stats: []StatBonus{{Stat: "hp", Value: 50, PerLevel: 30}},
				Cost: 1200, SuccessRate: 0.70, MaxLevel: 10, Icon: "❤️",
				Description: "注入生命灵力，提升生命值上限",
				Materials: []MaterialCost{{ItemID: 3012, Quantity: 2, Name: "生命精华"}}},
			{ID: "iron_wall", Name: "铁壁", TargetSlots: []int32{EquipSlotHelmet, EquipSlotArmor, EquipSlotBelt, EquipSlotLegs},
				Stats: []StatBonus{{Stat: "damage_reduction", Value: 2, PerLevel: 1.5}},
				Cost: 2000, SuccessRate: 0.50, MaxLevel: 5, Icon: "🏰",
				Description: "构筑灵力屏障，获得伤害减免",
				Materials: []MaterialCost{{ItemID: 3013, Quantity: 3, Name: "玄铁锭"}}},
			{ID: "regeneration", Name: "再生", TargetSlots: []int32{EquipSlotHelmet, EquipSlotArmor, EquipSlotBelt, EquipSlotLegs},
				Stats: []StatBonus{{Stat: "hp_regen", Value: 5, PerLevel: 3}},
				Cost: 1500, SuccessRate: 0.60, MaxLevel: 8, Icon: "💚",
				Description: "提升生命自动恢复速度",
				Materials: []MaterialCost{{ItemID: 3014, Quantity: 2, Name: "回春草"}}},
			{ID: "body_guard", Name: "护体", TargetSlots: []int32{EquipSlotHelmet, EquipSlotArmor, EquipSlotBelt, EquipSlotLegs},
				Stats: []StatBonus{{Stat: "defense", Value: 8, PerLevel: 5}, {Stat: "hp", Value: 30, PerLevel: 20}},
				Cost: 1600, SuccessRate: 0.55, MaxLevel: 6, Icon: "✨",
				Description: "灵力护体，全面提升防御能力",
				Materials: []MaterialCost{{ItemID: 3015, Quantity: 2, Name: "护体灵符"}}},
			{ID: "spiritual", Name: "通灵", TargetSlots: []int32{EquipSlotHelmet, EquipSlotArmor, EquipSlotBelt, EquipSlotLegs},
				Stats: []StatBonus{{Stat: "mp", Value: 30, PerLevel: 20}},
				Cost: 1300, SuccessRate: 0.65, MaxLevel: 8, Icon: "🔮",
				Description: "打通灵脉，提升灵力上限",
				Materials: []MaterialCost{{ItemID: 3016, Quantity: 2, Name: "灵玉"}}},
		},
	},
	{
		GroupID: "boots", Name: "靴子附魔", Slots: []int32{EquipSlotBoots},
		Enchants: []EnchantmentDef{
			{ID: "divine_step", Name: "神行", TargetSlots: []int32{EquipSlotBoots},
				Stats: []StatBonus{{Stat: "speed", Value: 8, PerLevel: 5}},
				Cost: 1000, SuccessRate: 0.75, MaxLevel: 8, Icon: "👟",
				Description: "神行千里，大幅提升移动速度",
				Materials: []MaterialCost{{ItemID: 3021, Quantity: 2, Name: "风之羽"}}},
			{ID: "light_body", Name: "轻身", TargetSlots: []int32{EquipSlotBoots},
				Stats: []StatBonus{{Stat: "dodge_rate", Value: 2, PerLevel: 1.5}},
				Cost: 1500, SuccessRate: 0.60, MaxLevel: 6, Icon: "🕊️",
				Description: "身轻如燕，提升闪避率",
				Materials: []MaterialCost{{ItemID: 3022, Quantity: 2, Name: "轻身散"}}},
			{ID: "cloud_step", Name: "踏云", TargetSlots: []int32{EquipSlotBoots},
				Stats: []StatBonus{{Stat: "speed", Value: 3, PerLevel: 2}},
				Cost: 1200, SuccessRate: 0.65, MaxLevel: 6, Icon: "☁️",
				Description: "脚踏祥云，提升速度",
				Materials: []MaterialCost{{ItemID: 3023, Quantity: 2, Name: "云母石"}}},
			{ID: "stable", Name: "稳固", TargetSlots: []int32{EquipSlotBoots},
				Stats: []StatBonus{{Stat: "defense", Value: 8, PerLevel: 5}},
				Cost: 1000, SuccessRate: 0.70, MaxLevel: 8, Icon: "⛰️",
				Description: "稳如泰山，提升防御力",
				Materials: []MaterialCost{{ItemID: 3024, Quantity: 2, Name: "泰山石"}}},
			{ID: "dash", Name: "疾驰", TargetSlots: []int32{EquipSlotBoots},
				Stats: []StatBonus{{Stat: "speed", Value: 10, PerLevel: 6}, {Stat: "dodge_rate", Value: 1, PerLevel: 0.8}},
				Cost: 2000, SuccessRate: 0.50, MaxLevel: 5, Icon: "⚡",
				Description: "疾如闪电，速度与闪避双重提升",
				Materials: []MaterialCost{{ItemID: 3025, Quantity: 3, Name: "闪电石"}}},
			{ID: "traceless", Name: "无踪", TargetSlots: []int32{EquipSlotBoots},
				Stats: []StatBonus{{Stat: "dodge_rate", Value: 3, PerLevel: 2}},
				Cost: 1800, SuccessRate: 0.50, MaxLevel: 5, Icon: "🌫️",
				Description: "来去无踪，极大提升闪避率",
				Materials: []MaterialCost{{ItemID: 3026, Quantity: 3, Name: "隐形粉"}}},
		},
	},
	{
		GroupID: "accessory", Name: "饰品附魔", Slots: []int32{EquipSlotNecklace, EquipSlotRing},
		Enchants: []EnchantmentDef{
			{ID: "brilliance", Name: "辉煌", TargetSlots: []int32{EquipSlotNecklace, EquipSlotRing},
				Stats: []StatBonus{{Stat: "attack", Value: 5, PerLevel: 3}, {Stat: "defense", Value: 5, PerLevel: 3}, {Stat: "hp", Value: 20, PerLevel: 15}},
				Cost: 3000, SuccessRate: 0.40, MaxLevel: 5, Icon: "💎",
				Description: "绽放辉煌之光，全面提升属性",
				Materials: []MaterialCost{{ItemID: 3031, Quantity: 3, Name: "辉煌宝石"}}},
			{ID: "luck", Name: "幸运", TargetSlots: []int32{EquipSlotNecklace, EquipSlotRing},
				Stats: []StatBonus{{Stat: "luck", Value: 5, PerLevel: 3}},
				Cost: 2500, SuccessRate: 0.45, MaxLevel: 8, Icon: "🍀",
				Description: "凝聚幸运之力，提升机缘值",
				Materials: []MaterialCost{{ItemID: 3032, Quantity: 2, Name: "四叶草"}}},
			{ID: "enlightenment", Name: "悟道", TargetSlots: []int32{EquipSlotNecklace, EquipSlotRing},
				Stats: []StatBonus{{Stat: "exp_bonus", Value: 3, PerLevel: 2}},
				Cost: 2000, SuccessRate: 0.50, MaxLevel: 8, Icon: "📖",
				Description: "悟道明心，提升修炼经验获取",
				Materials: []MaterialCost{{ItemID: 3033, Quantity: 2, Name: "悟道茶"}}},
			{ID: "fortune", Name: "富贵", TargetSlots: []int32{EquipSlotNecklace, EquipSlotRing},
				Stats: []StatBonus{{Stat: "luck", Value: 3, PerLevel: 2}, {Stat: "exp_bonus", Value: 2, PerLevel: 1}},
				Cost: 1800, SuccessRate: 0.55, MaxLevel: 6, Icon: "💰",
				Description: "招财进宝，提升机缘与修炼速度",
				Materials: []MaterialCost{{ItemID: 3034, Quantity: 2, Name: "招财符"}}},
			{ID: "charm", Name: "魅力", TargetSlots: []int32{EquipSlotNecklace, EquipSlotRing},
				Stats: []StatBonus{{Stat: "luck", Value: 4, PerLevel: 2.5}},
				Cost: 2200, SuccessRate: 0.50, MaxLevel: 6, Icon: "💫",
				Description: "魅力四射，提升机缘与NPC好感",
				Materials: []MaterialCost{{ItemID: 3035, Quantity: 2, Name: "魅惑香"}}},
			{ID: "resistance", Name: "抗性", TargetSlots: []int32{EquipSlotNecklace, EquipSlotRing},
				Stats: []StatBonus{{Stat: "damage_reduction", Value: 1.5, PerLevel: 1}, {Stat: "hp", Value: 40, PerLevel: 25}},
				Cost: 2000, SuccessRate: 0.50, MaxLevel: 6, Icon: "🛡️",
				Description: "提升状态抗性与生存能力",
				Materials: []MaterialCost{{ItemID: 3036, Quantity: 3, Name: "抗性宝石"}}},
		},
	},
}

// GetAllEnchantmentDefs 获取所有附魔定义
func GetAllEnchantmentDefs() []EnchantmentDef {
	var all []EnchantmentDef
	for _, group := range EnchantmentSlotGroups {
		all = append(all, group.Enchants...)
	}
	return all
}

// GetEnchantmentDef 根据ID获取附魔定义
func GetEnchantmentDef(id string) *EnchantmentDef {
	for _, group := range EnchantmentSlotGroups {
		for _, e := range group.Enchants {
			if e.ID == id {
				return &e
			}
		}
	}
	return nil
}

// GetAwakenCost 获取觉醒消耗
func GetAwakenCost(level int) AwakenCost {
	switch level {
	case 1:
		return AwakenCost{
			SpiritStones: 5000,
			Level:        1,
			Materials: []MaterialCost{
				{ItemID: 4001, Quantity: 5, Name: "天灵石"},
				{ItemID: 4002, Quantity: 3, Name: "星辰砂"},
			},
		}
	case 2:
		return AwakenCost{
			SpiritStones: 20000,
			Level:        2,
			Materials: []MaterialCost{
				{ItemID: 4001, Quantity: 10, Name: "天灵石"},
				{ItemID: 4002, Quantity: 8, Name: "星辰砂"},
				{ItemID: 4003, Quantity: 3, Name: "混沌石"},
			},
		}
	case 3:
		return AwakenCost{
			SpiritStones: 80000,
			Level:        3,
			Materials: []MaterialCost{
				{ItemID: 4001, Quantity: 20, Name: "天灵石"},
				{ItemID: 4002, Quantity: 15, Name: "星辰砂"},
				{ItemID: 4003, Quantity: 8, Name: "混沌石"},
				{ItemID: 4004, Quantity: 1, Name: "鸿蒙碎片"},
			},
		}
	default:
		return AwakenCost{}
	}
}

// GetEnchantmentsForSlot 获取指定装备位可用的附魔
func GetEnchantmentsForSlot(slot int32) []EnchantmentDef {
	var result []EnchantmentDef
	for _, group := range EnchantmentSlotGroups {
		for _, s := range group.Slots {
			if s == slot {
				result = append(result, group.Enchants...)
				break
			}
		}
	}
	return result
}
