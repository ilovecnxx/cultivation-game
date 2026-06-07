package model

import "time"

// 物品类型
const (
	ItemTypeWeapon     = 1  // 武器
	ItemTypeHelmet     = 2  // 头盔
	ItemTypeArmor      = 3  // 衣服
	ItemTypeBracers    = 4  // 护腕
	ItemTypeBelt       = 5  // 腰带
	ItemTypeLegs       = 6  // 裤子
	ItemTypeBoots      = 7  // 鞋子
	ItemTypeNecklace   = 8  // 项链
	ItemTypeRing       = 9  // 戒指
	ItemTypePill       = 10 // 丹药
	ItemTypeMaterial   = 11 // 材料
	ItemTypeSkillBook  = 12 // 功法
	ItemTypeConsumable = 13 // 消耗品
)

// 装备品质
const (
	QualityCommon  = 1 // 凡品
	QualityLow     = 2 // 下品
	QualityMedium  = 3 // 中品
	QualityHigh    = 4 // 上品
	QualityEpic    = 5 // 极品
	QualityMythic  = 6 // 仙品
)

// QualityNames 品质中文名
var QualityNames = map[int32]string{
	QualityCommon: "凡品",
	QualityLow:    "下品",
	QualityMedium: "中品",
	QualityHigh:   "上品",
	QualityEpic:   "极品",
	QualityMythic: "仙品",
}

// 装备槽位（与物品类型一一映射）
const (
	EquipSlotWeapon   = 1
	EquipSlotHelmet   = 2
	EquipSlotArmor    = 3
	EquipSlotBracers  = 4
	EquipSlotBelt     = 5
	EquipSlotLegs     = 6
	EquipSlotBoots    = 7
	EquipSlotNecklace = 8
	EquipSlotRing     = 9
)

// ItemTypeToEquipSlot 物品类型到装备槽位的映射
var ItemTypeToEquipSlot = map[int32]int32{
	ItemTypeWeapon:   EquipSlotWeapon,
	ItemTypeHelmet:   EquipSlotHelmet,
	ItemTypeArmor:    EquipSlotArmor,
	ItemTypeBracers:  EquipSlotBracers,
	ItemTypeBelt:     EquipSlotBelt,
	ItemTypeLegs:     EquipSlotLegs,
	ItemTypeBoots:    EquipSlotBoots,
	ItemTypeNecklace: EquipSlotNecklace,
	ItemTypeRing:     EquipSlotRing,
}

// Item 物品模板（静态配置）
type Item struct {
	ID             int64  `json:"id" gorm:"primaryKey"`
	Name           string `json:"name" gorm:"size:64;not null"`
	Type           int32  `json:"type" gorm:"not null;index"`
	Quality        int32  `json:"quality" gorm:"default:1"`
	RequiredLevel  int32  `json:"required_level" gorm:"default:1"`
	RequiredRealm  int32  `json:"required_realm" gorm:"default:1"`
	Description    string `json:"description" gorm:"size:256"`
	MaxStack       int32  `json:"max_stack" gorm:"default:1"` // 最大堆叠数，1=不可堆叠
	BaseAttack     int64  `json:"base_attack" gorm:"default:0"`
	BaseDefense    int64  `json:"base_defense" gorm:"default:0"`
	BaseHP         int64  `json:"base_hp" gorm:"default:0"`
	BaseMP         int64  `json:"base_mp" gorm:"default:0"`
	UseEffect      string `json:"use_effect" gorm:"size:128"`          // 使用效果 JSON 配置
	SellPrice      int64  `json:"sell_price" gorm:"default:0"`         // 出售价格（灵石）
	SellPriceBound int64  `json:"sell_price_bound" gorm:"default:0"`   // 出售价格（绑定灵石）
	CreatedAt      time.Time `json:"created_at"`
}

// InventoryItem 玩家背包中的物品
type InventoryItem struct {
	ID         int64     `json:"id" gorm:"primaryKey"`
	PlayerID   int64     `json:"player_id" gorm:"index;not null"`
	ItemID     int64     `json:"item_id" gorm:"not null;index"`
	Quantity   int32     `json:"quantity" gorm:"default:1"`
	SlotIndex  int32     `json:"slot_index" gorm:"default:0"` // 背包格子索引，0=未分配
	IsEquipped bool      `json:"is_equipped" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// 关联的物品模板（查询时填充）
	Item *Item `json:"item,omitempty" gorm:"foreignKey:ItemID"`
}

// Equipment 玩家装备（强化信息）
type Equipment struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	PlayerID        int64     `json:"player_id" gorm:"index;not null"`
	Slot            int32     `json:"slot" gorm:"not null"`            // 装备槽位
	InventoryItemID int64     `json:"inventory_item_id" gorm:"uniqueIndex;not null"`
	ItemID          int64     `json:"item_id" gorm:"not null"`
	Level           int32     `json:"level" gorm:"default:0"`          // 强化等级
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Item *Item `json:"item,omitempty" gorm:"foreignKey:ItemID"`
}

// -------- 请求/响应结构体 --------

// UseItemRequest 使用物品请求
type UseItemRequest struct {
	InventoryItemID int64 `json:"inventory_item_id" binding:"required"`
	Quantity        int32 `json:"quantity" binding:"required,min=1"`
}

// SortInventoryRequest 整理背包请求
type SortInventoryRequest struct {
	SortBy string `json:"sort_by" binding:"required,oneof=type quality level"` // 排序字段
	Desc   bool   `json:"desc"`                                                // 是否降序
}

// EquipRequest 穿戴/卸下装备请求
type EquipRequest struct {
	InventoryItemID int64 `json:"inventory_item_id" binding:"required"` // 背包物品ID
}

// UnequipRequest 卸下装备请求
type UnequipRequest struct {
	Slot int32 `json:"slot" binding:"required,min=1,max=9"` // 装备槽位
}

// StrengthenRequest 强化装备请求
type StrengthenRequest struct {
	Slot int32 `json:"slot" binding:"required,min=1,max=9"`
}

// CurrencyChangeRequest 货币变更请求
type CurrencyChangeRequest struct {
	Gold      int64 `json:"gold"`
	BoundGold int64 `json:"bound_gold"`
	Jade      int64 `json:"jade"`
}

// InventoryTransferRequest 物品转移请求
type InventoryTransferRequest struct {
	FromSlot int32 `json:"from_slot" binding:"required,min=1"`
	ToSlot   int32 `json:"to_slot" binding:"required,min=1"`
	Quantity int32 `json:"quantity" binding:"required,min=1"`
}
