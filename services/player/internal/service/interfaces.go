package service

import (
	"context"

	"cultivation-game/services/player/internal/model"
)

// PlayerRepository 玩家数据存储接口
type PlayerRepository interface {
	Create(p *model.Player) error
	GetByID(id int64) (*model.Player, error)
	GetByUserID(userID string) (*model.Player, error)
	GetByName(name string) (*model.Player, error)
	Update(p *model.Player) error
	UpdateCurrency(playerID int64, gold, boundGold, jade int64) error
	Delete(id int64) error
}

// InventoryRepository 背包/装备数据存储接口
type InventoryRepository interface {
	InsertItem(inv *model.InventoryItem) error
	GetInventoryByPlayer(playerID int64) ([]*model.InventoryItem, error)
	GetInventoryItem(inventoryItemID int64) (*model.InventoryItem, error)
	FindStackableItem(playerID, itemID int64) (*model.InventoryItem, error)
	UpdateItemQuantity(id int64, quantity int32) error
	UpdateItemSlot(id int64, slotIndex int32) error
	UpdateItemEquipStatus(id int64, isEquipped bool) error
	DeleteItem(id int64) error
	GetInventoryCount(playerID int64) (int, error)
	InsertEquipment(eq *model.Equipment) error
	GetEquipmentByPlayer(playerID int64) ([]*model.Equipment, error)
	GetEquipmentBySlot(playerID, slot int64) (*model.Equipment, error)
	DeleteEquipment(id int64) error
	DeleteEquipmentBySlot(playerID, slot int64) error
	UpdateEquipmentLevel(id int64, level int32) error
	GetItem(itemID int64) (*model.Item, error)
	ListItems(ids []int64) (map[int64]*model.Item, error)
}

// EquipSetRepository 装备套装/附魔/觉醒数据存储接口
type EquipSetRepository interface {
	GetEquipmentByID(equipmentID int64) (*model.Equipment, error)
	GetEquipmentByPlayer(playerID int64) ([]*model.Equipment, error)
	GetEquipmentBySlot(playerID, slot int64) (*model.Equipment, error)
	GetInventoryByPlayer(playerID int64) ([]*model.InventoryItem, error)
	GetItem(itemID int64) (*model.Item, error)
	ListItems(ids []int64) (map[int64]*model.Item, error)
	UpdateEquipmentLevel(id int64, level int32) error
	UpdateEquipmentMaxLevel(id int64, maxLevel int32) error

	// 附魔
	GetPlayerEnchantments(playerID int64) ([]*model.PlayerEnchantment, error)
	GetEquipmentEnchantments(equipmentID int64) ([]*model.PlayerEnchantment, error)
	SaveEnchantment(enchant *model.PlayerEnchantment) error
	RemoveEnchantment(id int64) error

	// 觉醒
	GetEquipmentAwakening(equipmentID int64) (*model.EquipmentAwakening, error)
	SaveAwakening(awaken *model.EquipmentAwakening) error
	UpdateAwakening(awaken *model.EquipmentAwakening) error
}

// Cache 缓存接口
type Cache interface {
	SetPlayer(ctx context.Context, p *model.PlayerCache) error
	GetPlayer(ctx context.Context, playerID int64) (*model.PlayerCache, error)
	DelPlayer(ctx context.Context, playerID int64) error
	RefreshTTL(ctx context.Context, playerID int64) error
	SetInventoryCache(ctx context.Context, playerID int64, items []*model.InventoryItem) error
	GetInventoryCache(ctx context.Context, playerID int64) ([]*model.InventoryItem, error)
}
