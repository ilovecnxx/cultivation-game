package models

import "time"

// ItemQuality 物品品质枚举。
type ItemQuality int32

const (
	ItemQualityCommon    ItemQuality = 0 // 凡品
	ItemQualityUncommon  ItemQuality = 1 // 良品
	ItemQualityRare      ItemQuality = 2 // 上品
	ItemQualityEpic      ItemQuality = 3 // 极品
	ItemQualityLegendary ItemQuality = 4 // 传说
	ItemQualityMythical  ItemQuality = 5 // 神话
)

// ItemType 物品种类。
type ItemType int32

const (
	ItemTypeConsumable ItemType = 0 // 消耗品（丹药、符箓）
	ItemTypeEquipment  ItemType = 1 // 装备
	ItemTypeMaterial   ItemType = 2 // 材料
	ItemTypeQuest      ItemType = 3 // 任务道具
	ItemTypeSecret     ItemType = 4 // 功法秘籍
)

// Item 游戏物品的基础定义（模板/配置），对应配置表中的一行。
type Item struct {
	ID          uint32    `json:"id"`           // 物品模板ID
	Name        string    `json:"name"`         // 物品名称
	Type        ItemType  `json:"type"`         // 物品种类
	Quality     ItemQuality `json:"quality"`    // 品质
	StackMax    uint32    `json:"stack_max"`    // 最大堆叠数量
	UseLevel    uint32    `json:"use_level"`    // 使用所需境界等级
	SellPrice   uint64    `json:"sell_price"`   // 出售价格（灵石）
	Description string    `json:"description"`  // 物品描述
	// UseEffect 使用效果，JSON 格式，不同物品种类结构不同。
	// 如丹药: {"exp": 1000, "attr": "spirit", "value": 5}
	UseEffect string `json:"use_effect,omitempty"`
}

// InventorySlot 背包中的一个格子。
type InventorySlot struct {
	ItemID  uint32 // 物品模板ID
	Count   uint32 // 数量
	SlotIdx uint32 // 格子索引（从0开始）
}

// Inventory 玩家背包。
type Inventory struct {
	PlayerID  uint64          `json:"player_id"` // 所属玩家ID
	Capacity  uint32          `json:"capacity"`  // 背包容量
	Slots     []InventorySlot `json:"slots"`     // 物品列表
	Gold      uint64          `json:"gold"`      // 灵石数量
}

// NewInventory 创建一个指定容量的空背包。
func NewInventory(playerID uint64, capacity uint32) *Inventory {
	return &Inventory{
		PlayerID: playerID,
		Capacity: capacity,
		Slots:    make([]InventorySlot, 0, capacity),
		Gold:     0,
	}
}

// AddItem 向背包添加物品。如果已存在同类可堆叠物品则叠加，否则占用新格子。
// 返回实际添加的数量。背包满时返回 0。
func (inv *Inventory) AddItem(item *Item, count uint32) uint32 {
	if item.StackMax > 1 {
		// 尝试合并到已有格子
		for i, slot := range inv.Slots {
			if slot.ItemID == item.ID && slot.Count < item.StackMax {
				space := item.StackMax - slot.Count
				toAdd := minU32(count, space)
				inv.Slots[i].Count += toAdd
				return toAdd
			}
		}
	}

	// 需要新格子
	if uint32(len(inv.Slots)) >= inv.Capacity {
		return 0
	}
	toAdd := minU32(count, item.StackMax)
	inv.Slots = append(inv.Slots, InventorySlot{
		ItemID: item.ID,
		Count:  toAdd,
	})
	return toAdd
}

// RemoveItem 从背包移除指定数量的物品，返回实际移除数量。
func (inv *Inventory) RemoveItem(itemID uint32, count uint32) uint32 {
	removed := uint32(0)
	for i := 0; i < len(inv.Slots); i++ {
		if inv.Slots[i].ItemID == itemID {
			if inv.Slots[i].Count <= count-removed {
				removed += inv.Slots[i].Count
				// 移除该格子
				inv.Slots = append(inv.Slots[:i], inv.Slots[i+1:]...)
				i-- // 因为切片长度变了
			} else {
				inv.Slots[i].Count -= count - removed
				removed = count
			}
			if removed >= count {
				break
			}
		}
	}
	return removed
}

// ItemCount 返回背包中指定物品的总数量。
func (inv *Inventory) ItemCount(itemID uint32) uint32 {
	total := uint32(0)
	for _, slot := range inv.Slots {
		if slot.ItemID == itemID {
			total += slot.Count
		}
	}
	return total
}

// EquipmentSlot 装备槽位枚举。
type EquipmentSlot int32

const (
	EquipSlotWeapon EquipmentSlot = 0 // 武器
	EquipSlotArmor  EquipmentSlot = 1 // 护甲
	EquipSlotRing   EquipmentSlot = 2 // 戒指
	EquipSlotAmulet EquipmentSlot = 3 // 项链
	EquipSlotBoots  EquipmentSlot = 4 // 靴子
)

// Equipment 玩家装备栏。
type Equipment struct {
	PlayerID uint64                    `json:"player_id"`
	Equipped map[EquipmentSlot]uint32 `json:"equipped"` // 槽位 -> 物品模板ID
	UpdatedAt time.Time               `json:"updated_at"`
}

// NewEquipment 创建空的装备栏。
func NewEquipment(playerID uint64) *Equipment {
	return &Equipment{
		PlayerID: playerID,
		Equipped: make(map[EquipmentSlot]uint32),
	}
}

// Equip 装备物品到指定槽位。返回之前在该槽位的物品ID（0表示空）。
func (e *Equipment) Equip(slot EquipmentSlot, itemID uint32) uint32 {
	old := e.Equipped[slot]
	e.Equipped[slot] = itemID
	e.UpdatedAt = time.Now()
	return old
}

// Unequip 卸下指定槽位的装备。返回卸下的物品ID（0表示空）。
func (e *Equipment) Unequip(slot EquipmentSlot) uint32 {
	old := e.Equipped[slot]
	if old != 0 {
		delete(e.Equipped, slot)
		e.UpdatedAt = time.Now()
	}
	return old
}

func minU32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
