// Package model 定义世界服务的数据模型
package model

import "time"

// ============================================================
// 地图探索模型
// ============================================================

// RegionType 区域类型
type RegionType string

const (
	RegionNewbie      RegionType = "newbie_village"   // 新手村
	RegionTown        RegionType = "town"             // 城镇
	RegionSecretRealm RegionType = "secret_realm"     // 秘境
	RegionDangerous   RegionType = "dangerous_land"   // 险地
)

// MapRegion 地图区域
type MapRegion struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        RegionType `json:"type"`
	Description string     `json:"description"`
	LevelMin    int        `json:"level_min"`    // 最低进入等级
	LevelMax    int        `json:"level_max"`    // 最高进入等级
	DangerLevel int        `json:"danger_level"` // 危险等级 1-10
	Connections []string   `json:"connections"`  // 相邻区域ID列表
	// 资源产出
	Resources struct {
		SpiritQi float64 `json:"spirit_qi"` // 灵气浓度(修炼速度加成)
		Items    []struct {
			ItemID string  `json:"item_id"`
			Rate   float64 `json:"rate"` // 出现概率 0-1
		} `json:"items"`
	} `json:"resources"`
}

// PlayerExploreInfo 玩家探索状态
type PlayerExploreInfo struct {
	UserID     string    `bson:"user_id" json:"user_id"`
	RegionID   string    `bson:"region_id" json:"region_id"`
	PositionX  int       `bson:"position_x" json:"position_x"`
	PositionY  int       `bson:"position_y" json:"position_y"`
	LastMoveAt time.Time `bson:"last_move_at" json:"last_move_at"`
	// 已发现区域列表
	DiscoveredRegions []string `bson:"discovered_regions" json:"discovered_regions"`
}

// ============================================================
// 奇遇系统模型
// ============================================================

// EncounterCondition 奇遇触发条件
type EncounterCondition struct {
	Type     string      `json:"type"`     // level / cultivation / item / quest / probability
	Operator string      `json:"operator"` // gt / gte / lt / lte / eq / has
	Value    interface{} `json:"value"`
}

// EncounterOutcome 奇遇结果
type EncounterOutcome struct {
	Type        string `json:"type"`        // item / exp / spirit_stone / damage / teleport / buff
	TargetID    string `json:"target_id"`   // 物品ID/区域ID等
	Amount      int64  `json:"amount"`      // 数量/数值
	Description string `json:"description"` // 结果描述
}

// EncounterChoice 奇遇选项
type EncounterChoice struct {
	Text    string             `json:"text"`
	Outcome *EncounterOutcome  `json:"outcome"`
}

// Encounter 奇遇配置
type Encounter struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Regions     []string            `json:"regions"`     // 可触发区域
	Probability float64             `json:"probability"` // 触发概率 0-1
	MinLevel    int                 `json:"min_level"`
	MaxLevel    int                 `json:"max_level"`
	Conditions  []EncounterCondition `json:"conditions"`
	Choices     []EncounterChoice   `json:"choices"`
	AutoOutcome *EncounterOutcome   `json:"auto_outcome,omitempty"` // 自动触发(无需选择)
	CooldownSec int64               `json:"cooldown_sec"`           // 冷却时间(秒)
}

// PlayerEncounter 玩家奇遇记录
type PlayerEncounter struct {
	UserID      string    `bson:"user_id" json:"user_id"`
	EncounterID string    `bson:"encounter_id" json:"encounter_id"`
	LastTrigger time.Time `bson:"last_trigger" json:"last_trigger"` // 上次触发时间(用于冷却)
}

// ============================================================
// NPC 系统模型
// ============================================================

// NPCType NPC 类型
type NPCType string

const (
	NPCQuestGiver  NPCType = "quest_giver"   // 任务发布者
	NPCShop        NPCType = "shop"          // 商人
	NPCTrainer     NPCType = "trainer"       // 技能训练师
	NPCCultivator  NPCType = "cultivator"    // 散修
)

// NPC 配置
type NPC struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        NPCType  `json:"type"`
	Title       string   `json:"title"`       // 称号
	Description string   `json:"description"`
	RegionID    string   `json:"region_id"`   // 所在区域
	Dialogues   []string `json:"dialogues"`   // 对白列表(随机)
	// 商店配置(商人类型)
	ShopItems []NPCShopItem `json:"shop_items,omitempty"`
	// 任务配置(任务发布者类型)
	Quests []string `json:"quests,omitempty"`
}

// NPCShopItem 商店物品
type NPCShopItem struct {
	ItemID    string `json:"item_id"`
	Name      string `json:"name"`
	Price     int64  `json:"price"`
	Currency  string `json:"currency"`   // spirit_stone / contribution
	Stock     int    `json:"stock"`      // 库存(-1为不限)
	LevelReq  int    `json:"level_req"`  // 等级要求
}

// PlayerNPCInteraction NPC交互记录
type PlayerNPCInteraction struct {
	UserID     string    `bson:"user_id" json:"user_id"`
	NPCID      string    `bson:"npc_id" json:"npc_id"`
	Dialogues  []string  `bson:"dialogues" json:"dialogues"` // 已触发的对话
	LastTalkAt time.Time `bson:"last_talk_at" json:"last_talk_at"`
}

// ============================================================
// 采集系统模型
// ============================================================

// GatheringSpot 采集点
type GatheringSpot struct {
	ID          string  `json:"id"`
	RegionID    string  `json:"region_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"` // herb / ore / wood
	ItemID      string  `json:"item_id"`
	ItemName    string  `json:"item_name"`
	MinAmount   int64   `json:"min_amount"`
	MaxAmount   int64   `json:"max_amount"`
	Difficulty  int     `json:"difficulty"`  // 采集难度 1-10
	LevelReq    int     `json:"level_req"`   // 等级要求
	RespawnSec  int64   `json:"respawn_sec"` // 刷新时间(秒)
}

// GatheringRecord 采集记录
type GatheringRecord struct {
	UserID      string    `bson:"user_id" json:"user_id"`
	SpotID      string    `bson:"spot_id" json:"spot_id"`
	LastGather  time.Time `bson:"last_gather" json:"last_gather"`
}

// ============================================================
// 世界事件模型
// ============================================================

// WorldEventType 世界事件类型
type WorldEventType string

const (
	EventWorldBoss     WorldEventType = "world_boss"      // 世界 Boss
	EventTreasureRain  WorldEventType = "treasure_rain"   // 天降异宝
	EventSectWarNotice WorldEventType = "sect_war_notice" // 宗门战公告
	EventMysticMist    WorldEventType = "mystic_mist"     // 神秘迷雾
)

// WorldEvent 世界事件
type WorldEvent struct {
	ID          string         `bson:"_id" json:"id"`
	Type        WorldEventType `bson:"type" json:"type"`
	Title       string         `bson:"title" json:"title"`
	Description string         `bson:"description" json:"description"`
	RegionID    string         `bson:"region_id,omitempty" json:"region_id,omitempty"`
	// 事件参数
	Params map[string]interface{} `bson:"params,omitempty" json:"params,omitempty"`
	// 调度
	Status     string    `bson:"status" json:"status"` // scheduled / active / finished
	StartAt    time.Time `bson:"start_at" json:"start_at"`
	EndAt      time.Time `bson:"end_at,omitempty" json:"end_at,omitempty"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	// 重复调度
	ScheduleCron string `bson:"schedule_cron,omitempty" json:"schedule_cron,omitempty"`
}

// WorldBossState 世界 Boss 状态
type WorldBossState struct {
	EventID    string  `bson:"event_id" json:"event_id"`
	BossID     string  `bson:"boss_id" json:"boss_id"`
	Name       string  `bson:"name" json:"name"`
	Level      int     `bson:"level" json:"level"`
	HP         float64 `bson:"hp" json:"hp"`
	MaxHP      float64 `bson:"max_hp" json:"max_hp"`
	Attack     float64 `bson:"attack" json:"attack"`
	Defense    float64 `bson:"defense" json:"defense"`
	RegionID   string  `bson:"region_id" json:"region_id"`
	Status     string  `bson:"status" json:"status"` // alive / defeated
	SpawnedAt  time.Time `bson:"spawned_at" json:"spawned_at"`
	DefeatedAt time.Time `bson:"defeated_at,omitempty" json:"defeated_at,omitempty"`
}

// WorldBossDamage 玩家伤害排行

// ============================================================
// 灵脉争夺系统模型
// ============================================================

// SpiritVein 灵脉资源点
type SpiritVein struct {
	ID               string    `bson:"_id" json:"id"`
	Name             string    `bson:"name" json:"name"`                             // "九天灵脉", "地脉灵泉"
	Quality          int       `bson:"quality" json:"quality"`                       // 1-5 stars
	RegionID         string    `bson:"region_id" json:"region_id"`
	RegionName       string    `bson:"region_name,omitempty" json:"region_name,omitempty"`
	Position         [2]float64 `bson:"position" json:"position"`                    // map coordinates [x, y]
	OwnerType        string    `bson:"owner_type" json:"owner_type"`                 // "none", "player", "sect"
	OwnerID          string    `bson:"owner_id,omitempty" json:"owner_id,omitempty"`
	OwnerName        string    `bson:"owner_name,omitempty" json:"owner_name,omitempty"`
	OccupiedSince    time.Time `bson:"occupied_since,omitempty" json:"occupied_since,omitempty"`
	LastYieldTime    time.Time `bson:"last_yield_time,omitempty" json:"last_yield_time,omitempty"`
	YieldInterval    int64     `bson:"yield_interval" json:"yield_interval"`         // resource generation interval (seconds)
	YieldAmount      int64     `bson:"yield_amount" json:"yield_amount"`             // spirit stones per interval
	CultivationBonus float64   `bson:"cultivation_bonus" json:"cultivation_bonus"`   // cultivation speed bonus for owner (percent)
	Defenders        []string  `bson:"defenders,omitempty" json:"defenders,omitempty"`             // players guarding
	ContestedBy      []string  `bson:"contested_by,omitempty" json:"contested_by,omitempty"`       // players attacking
	Status           string    `bson:"status" json:"status"`                         // "idle", "contested", "occupied"
	Discovered       bool      `bson:"discovered" json:"discovered"`                 // whether this vein is discovered
	Description      string    `bson:"description,omitempty" json:"description,omitempty"`
	UpgradeLevel     int       `bson:"upgrade_level" json:"upgrade_level"`            // current upgrade level (0 = base)
	CreatedAt        time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// VeinContest 灵脉争夺战
type VeinContest struct {
	ID          string    `bson:"_id" json:"id"`
	VeinID      string    `bson:"vein_id" json:"vein_id"`
	VeinName    string    `bson:"vein_name,omitempty" json:"vein_name,omitempty"`
	AttackerID  string    `bson:"attacker_id" json:"attacker_id"`
	AttackerName string   `bson:"attacker_name,omitempty" json:"attacker_name,omitempty"`
	DefenderID  string    `bson:"defender_id" json:"defender_id"`
	DefenderName string   `bson:"defender_name,omitempty" json:"defender_name,omitempty"`
	StartTime   time.Time `bson:"start_time" json:"start_time"`
	EndTime     time.Time `bson:"end_time" json:"end_time"`                // 30 minute time limit
	AttackerHP  int64     `bson:"attacker_hp" json:"attacker_hp"`
	AttackerMaxHP int64   `bson:"attacker_max_hp" json:"attacker_max_hp"`
	DefenderHP  int64     `bson:"defender_hp" json:"defender_hp"`
	DefenderMaxHP int64   `bson:"defender_max_hp" json:"defender_max_hp"`
	Status      string    `bson:"status" json:"status"`                    // "active", "attacker_win", "defender_win", "timeout"
	Spectators  []string  `bson:"spectators,omitempty" json:"spectators,omitempty"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// VeinUpgrade 灵脉升级记录
type VeinUpgrade struct {
	ID         string    `bson:"_id" json:"id"`
	VeinID     string    `bson:"vein_id" json:"vein_id"`
	OwnerID    string    `bson:"owner_id" json:"owner_id"`
	OwnerType  string    `bson:"owner_type" json:"owner_type"`
	OldQuality int       `bson:"old_quality" json:"old_quality"`
	NewQuality int       `bson:"new_quality" json:"new_quality"`
	CostStones int64     `bson:"cost_stones" json:"cost_stones"`
	Duration   int64     `bson:"duration" json:"duration"` // seconds
	StartTime  time.Time `bson:"start_time" json:"start_time"`
	EndTime    time.Time `bson:"end_time" json:"end_time"`
	Status     string    `bson:"status" json:"status"` // "in_progress", "completed", "cancelled"
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
}

// VeinDiscovery 灵脉发现记录
type VeinDiscovery struct {
	ID         string    `bson:"_id" json:"id"`
	UserID     string    `bson:"user_id" json:"user_id"`
	VeinID     string    `bson:"vein_id" json:"vein_id"`
	Method     string    `bson:"method" json:"method"` // "explore", "divination", "map_purchase"
	RewardStones int64   `bson:"reward_stones" json:"reward_stones"`
	DiscoveredAt time.Time `bson:"discovered_at" json:"discovered_at"`
}

// VeinOccupationHistory 灵脉占领历史
type VeinOccupationHistory struct {
	ID          string    `bson:"_id" json:"id"`
	VeinID      string    `bson:"vein_id" json:"vein_id"`
	VeinName    string    `bson:"vein_name" json:"vein_name"`
	VeinQuality int       `bson:"vein_quality" json:"vein_quality"`
	OwnerType   string    `bson:"owner_type" json:"owner_type"`
	OwnerID     string    `bson:"owner_id" json:"owner_id"`
	OwnerName   string    `bson:"owner_name" json:"owner_name"`
	OccupiedAt  time.Time `bson:"occupied_at" json:"occupied_at"`
	LostAt      time.Time `bson:"lost_at,omitempty" json:"lost_at,omitempty"`
	Duration    int64     `bson:"duration,omitempty" json:"duration,omitempty"` // hours
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}
