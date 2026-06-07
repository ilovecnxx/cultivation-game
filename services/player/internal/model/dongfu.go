package model

import "time"

// 房间类型常量
const (
	RoomTypeTraining = 1 // 修炼室：修炼效率+5%/级
	RoomTypeAlchemy  = 2 // 炼丹室：炼丹成功率+3%/级
	RoomTypeTreasure = 3 // 藏宝阁：存储物品上限+10/级
	RoomTypeArena    = 4 // 演武场：被动战斗经验+2/小时/级
	RoomTypeSpring   = 5 // 灵泉：被动灵石产出+5/小时/级
)

// RoomTypeNames 房间类型中文名
var RoomTypeNames = map[int]string{
	RoomTypeTraining: "修炼室",
	RoomTypeAlchemy:  "炼丹室",
	RoomTypeTreasure: "藏宝阁",
	RoomTypeArena:    "演武场",
	RoomTypeSpring:   "灵泉",
}

// RoomTypeMaxLevel 各房间类型最大等级（1-10）
var RoomTypeMaxLevel = map[int]int{
	RoomTypeTraining: 10,
	RoomTypeAlchemy:  10,
	RoomTypeTreasure: 10,
	RoomTypeArena:    10,
	RoomTypeSpring:   10,
}

// RoomTypeBonusPerLevel 每级加成值
// 修炼室/炼丹室/藏宝阁是数值加成；演武场/灵泉是每小时产出
var RoomTypeBonusPerLevel = map[int]float64{
	RoomTypeTraining: 5.0,  // 修炼效率+5%/级
	RoomTypeAlchemy:  3.0,  // 炼丹成功率+3%/级
	RoomTypeTreasure: 10.0, // 存储上限+10/级
	RoomTypeArena:    2.0,  // 战斗经验+2/小时/级
	RoomTypeSpring:   5.0,  // 灵石产出+5/小时/级
}

// RoomTypeEffectDesc 效果描述模板
var RoomTypeEffectDesc = map[int]string{
	RoomTypeTraining: "修炼效率+%.0f%%",
	RoomTypeAlchemy:  "炼丹成功率+%.0f%%",
	RoomTypeTreasure: "存储物品上限+%.0f",
	RoomTypeArena:    "战斗经验+%.0f/小时",
	RoomTypeSpring:   "灵石产出+%.0f/小时",
}

// RoomTypeIcon 房间图标（用于前端）
var RoomTypeIcon = map[int]string{
	RoomTypeTraining: "🧘",
	RoomTypeAlchemy:  "⚗️",
	RoomTypeTreasure: "🏺",
	RoomTypeArena:    "⚔️",
	RoomTypeSpring:   "💧",
}

// RoomTypeDetailDesc 房间详细说明
var RoomTypeDetailDesc = map[int]string{
	RoomTypeTraining: "在灵气充沛的密室中打坐修炼，大幅提升修为获取速度。",
	RoomTypeAlchemy:  "配备上品丹炉与灵火，提高丹药炼制成功率和品质。",
	RoomTypeTreasure: "设有禁制防护的宝库，可存放更多天材地宝与法器。",
	RoomTypeArena:    "演武切磋之地，即使静坐观摩也能领悟战斗真谛，获得战斗经验。",
	RoomTypeSpring:   "洞府灵脉汇聚之泉，会自动产出灵石供主人使用。",
}

// DongFuGatheringStatus 洞府灵气汇聚状态
const (
	GatheringStatusIdle    = 0 // 空闲中
	GatheringStatusActive  = 1 // 汇聚中
)

// DongFu 洞府
type DongFu struct {
	ID                  int64     `json:"id" gorm:"primaryKey"`
	PlayerID            int64     `json:"player_id" gorm:"uniqueIndex;not null"`
	Level               int       `json:"level" gorm:"default:1"`                        // 洞府等级 = 所有房间等级之和
	Name                string    `json:"name" gorm:"size:32;default:'洞府'"`
	CultivationBonus    float64   `json:"cultivation_bonus" gorm:"type:decimal(10,2);default:0"` // 修炼加成%
	AlchemyBonus        float64   `json:"alchemy_bonus" gorm:"type:decimal(10,2);default:0"`     // 炼丹加成%
	StorageBonus        float64   `json:"storage_bonus" gorm:"type:decimal(10,2);default:0"`     // 存储加成
	CombatExpPerHour    float64   `json:"combat_exp_per_hour" gorm:"type:decimal(10,2);default:0"` // 战斗经验/小时
	SpiritStonesPerHour float64   `json:"spirit_stones_per_hour" gorm:"type:decimal(10,2);default:0"` // 灵石/小时
	SpiritEnergy        float64   `json:"spirit_energy" gorm:"type:decimal(10,2);default:0"`   // 洞府灵气值
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Rooms               []Room    `json:"rooms,omitempty" gorm:"-"`
	Decorations         []Decoration `json:"decorations,omitempty" gorm:"-"`
	ActiveGathering     *SpiritGathering `json:"active_gathering,omitempty" gorm:"-"`
	Guests              []Guest   `json:"guests,omitempty" gorm:"-"`
}

// Room 洞府房间
type Room struct {
	ID       int64   `json:"id" gorm:"primaryKey"`
	DongFuID int64   `json:"dongfu_id" gorm:"index;not null"`
	RoomType int     `json:"room_type" gorm:"not null"`                // 房间类型 1-5
	Level    int     `json:"level" gorm:"default:1"`                  // 房间等级 1-10
	Name     string  `json:"name" gorm:"size:32;not null"`            // 房间名称
	Effect   string  `json:"effect" gorm:"size:128;default:''"`       // 效果描述
	Bonus    float64 `json:"bonus" gorm:"type:decimal(10,2);default:0"` // 当前加成值
}

// SpiritGathering 洞府灵气汇聚（挂机修炼）
type SpiritGathering struct {
	ID            int64     `json:"id" gorm:"primaryKey"`
	DongFuID      int64     `json:"dongfu_id" gorm:"index;not null"`
	PlayerID      int64     `json:"player_id" gorm:"index;not null"`
	Status        int       `json:"status" gorm:"default:0"`                             // 0=空闲 1=汇聚中
	StartTime     time.Time `json:"start_time"`                                          // 开始时间
	Duration      int       `json:"duration" gorm:"default:0"`                           // 计划时长(秒)
	BonusCultivation float64 `json:"bonus_cultivation" gorm:"type:decimal(10,2);default:0"` // 已累积修为
	ElapsedSeconds int      `json:"elapsed_seconds" gorm:"default:0"`                    // 已持续秒数
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Decoration 洞府装饰
type Decoration struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	DongFuID        int64     `json:"dongfu_id" gorm:"index;not null"`
	PlayerID        int64     `json:"player_id" gorm:"index;not null"`
	ItemID          string    `json:"item_id" gorm:"size:64;not null"`
	Name            string    `json:"name" gorm:"size:64;not null"`
	DecorationType  int       `json:"decoration_type" gorm:"default:0"`               // 类型 0=装饰 1=家具 2=盆景 3=挂画 4=奇石
	BonusType       string    `json:"bonus_type" gorm:"size:32;default:''"`           // 加成类型: cultivation/alchemy/defense
	BonusValue      float64   `json:"bonus_value" gorm:"type:decimal(10,2);default:0"` // 加成值
	Description     string    `json:"description" gorm:"size:256;default:''"`
	IsPlaced        bool      `json:"is_placed" gorm:"default:true"`                  // 是否已摆放
	PositionX       int       `json:"position_x" gorm:"default:0"`
	PositionY       int       `json:"position_y" gorm:"default:0"`
	CreatedAt       time.Time `json:"created_at"`
}

// Guest 洞府访客
type Guest struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	DongFuID        int64     `json:"dongfu_id" gorm:"index;not null"`
	GuestPlayerID   int64     `json:"guest_player_id" gorm:"not null"`
	HostPlayerID    int64     `json:"host_player_id" gorm:"not null"`
	Status          string    `json:"status" gorm:"size:16;default:'pending'"` // pending/visiting/completed
	VisitStart      *time.Time `json:"visit_start,omitempty"`
	VisitEnd        *time.Time `json:"visit_end,omitempty"`
	HostBonusType   string    `json:"host_bonus_type" gorm:"size:32;default:''"`
	HostBonusValue  float64   `json:"host_bonus_value" gorm:"type:decimal(10,2);default:0"`
	GuestBonusType  string    `json:"guest_bonus_type" gorm:"size:32;default:''"`
	GuestBonusValue float64   `json:"guest_bonus_value" gorm:"type:decimal(10,2);default:0"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	// 填充字段（不存库）
	GuestName   string `json:"guest_name,omitempty" gorm:"-"`
	GuestLevel  int    `json:"guest_level,omitempty" gorm:"-"`
	GuestRealm  string `json:"guest_realm,omitempty" gorm:"-"`
	HostName    string `json:"host_name,omitempty" gorm:"-"`
}

// ====================== 请求/响应结构体 ======================

// DongFuResponse 洞府响应
type DongFuResponse struct {
	DongFu           *DongFu `json:"dongfu"`
	CanBuild         bool    `json:"can_build"`
	RequiredLevel    int     `json:"required_level"`
	BuildCost        int64   `json:"build_cost"`
	RoomBuildCost    int64   `json:"room_build_cost"`
	RoomUpgradeCost  int64   `json:"room_upgrade_cost"`
}

// BuildDongFuRequest 建造洞府请求
type BuildDongFuRequest struct {
	Name string `json:"name" binding:"required,min=1,max=16"`
}

// BuildRoomRequest 建造房间请求
type BuildRoomRequest struct {
	RoomType int `json:"room_type" binding:"required,oneof=1 2 3 4 5"`
}

// UpgradeRoomRequest 升级房间请求
type UpgradeRoomRequest struct {
	RoomID int64 `json:"room_id" binding:"required"`
}

// StartGatheringRequest 开始灵气汇聚
type StartGatheringRequest struct {
	Duration int `json:"duration" binding:"required,min=60,max=86400"` // 秒
}

// PlaceDecorationRequest 摆放装饰
type PlaceDecorationRequest struct {
	DecorationType int    `json:"decoration_type" binding:"required,min=0,max=4"`
	Name           string `json:"name" binding:"required,min=1,max=16"`
	ItemID         string `json:"item_id" binding:"required"`
	PositionX      int    `json:"position_x"`
	PositionY      int    `json:"position_y" default:"0"`
}

// RemoveDecorationRequest 移除装饰
type RemoveDecorationRequest struct {
	DecorationID int64 `json:"decoration_id" binding:"required"`
}

// InviteGuestRequest 邀请访客
type InviteGuestRequest struct {
	GuestPlayerID int64 `json:"guest_player_id" binding:"required"`
}

// GuestActionRequest 访客动作（接受/拒绝/结束）
type GuestActionRequest struct {
	GuestID int64  `json:"guest_id" binding:"required"`
	Action  string `json:"action" binding:"required,oneof=accept reject complete"`
}

// RoomDetail 房间详情（含升级信息，用于前端展示）
type RoomDetail struct {
	Room
	Icon         string  `json:"icon"`
	Description  string  `json:"description"`
	MaxLevel     int     `json:"max_level"`
	BonusPerLevel float64 `json:"bonus_per_level"`
	NextBonus    float64 `json:"next_bonus"`
	BuildCost    int64   `json:"build_cost"`
	UpgradeCost  int64   `json:"upgrade_cost"`
	IsMaxLevel   bool    `json:"is_max_level"`
}
