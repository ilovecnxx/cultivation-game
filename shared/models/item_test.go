package models

import (
	"testing"
)

func TestItemQualityValues(t *testing.T) {
	tests := []struct {
		name string
		q    ItemQuality
		want int32
	}{
		{"Common", ItemQualityCommon, 0},
		{"Uncommon", ItemQualityUncommon, 1},
		{"Rare", ItemQualityRare, 2},
		{"Epic", ItemQualityEpic, 3},
		{"Legendary", ItemQualityLegendary, 4},
		{"Mythical", ItemQualityMythical, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.q) != tt.want {
				t.Errorf("ItemQuality = %d, want %d", int32(tt.q), tt.want)
			}
		})
	}
}

func TestItemTypeValues(t *testing.T) {
	tests := []struct {
		name string
		it   ItemType
		want int32
	}{
		{"Consumable", ItemTypeConsumable, 0},
		{"Equipment", ItemTypeEquipment, 1},
		{"Material", ItemTypeMaterial, 2},
		{"Quest", ItemTypeQuest, 3},
		{"Secret", ItemTypeSecret, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.it) != tt.want {
				t.Errorf("ItemType = %d, want %d", int32(tt.it), tt.want)
			}
		})
	}
}

func TestEquipmentSlotValues(t *testing.T) {
	tests := []struct {
		name string
		es   EquipmentSlot
		want int32
	}{
		{"Weapon", EquipSlotWeapon, 0},
		{"Armor", EquipSlotArmor, 1},
		{"Ring", EquipSlotRing, 2},
		{"Amulet", EquipSlotAmulet, 3},
		{"Boots", EquipSlotBoots, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.es) != tt.want {
				t.Errorf("EquipmentSlot = %d, want %d", int32(tt.es), tt.want)
			}
		})
	}
}

func TestItemStructure(t *testing.T) {
	item := Item{
		ID:          101,
		Name:        "回元丹",
		Type:        ItemTypeConsumable,
		Quality:     ItemQualityRare,
		StackMax:    99,
		UseLevel:    3,
		SellPrice:   500,
		Description: "恢复灵力",
		UseEffect:   `{"mana": 200}`,
	}
	if item.ID != 101 || item.Name != "回元丹" || item.Type != ItemTypeConsumable {
		t.Errorf("Item field mismatch: %+v", item)
	}
	if item.Quality != ItemQualityRare {
		t.Errorf("Quality = %d, want %d", item.Quality, ItemQualityRare)
	}
	if item.SellPrice != 500 {
		t.Errorf("SellPrice = %d, want 500", item.SellPrice)
	}
}

func TestNewInventory(t *testing.T) {
	inv := NewInventory(1001, 20)
	if inv == nil {
		t.Fatal("NewInventory returned nil")
	}
	if inv.PlayerID != 1001 {
		t.Errorf("PlayerID = %d, want 1001", inv.PlayerID)
	}
	if inv.Capacity != 20 {
		t.Errorf("Capacity = %d, want 20", inv.Capacity)
	}
	if len(inv.Slots) != 0 {
		t.Errorf("Expected empty slots, got %d", len(inv.Slots))
	}
	if inv.Gold != 0 {
		t.Errorf("Expected Gold = 0, got %d", inv.Gold)
	}
}

func TestInventory_AddItem_NewSlot(t *testing.T) {
	inv := NewInventory(1, 10)
	item := &Item{ID: 101, Name: "回元丹", StackMax: 1}

	added := inv.AddItem(item, 1)
	if added != 1 {
		t.Errorf("AddItem returned %d, want 1", added)
	}
	if len(inv.Slots) != 1 {
		t.Errorf("Expected 1 slot, got %d", len(inv.Slots))
	}
	if inv.Slots[0].ItemID != 101 || inv.Slots[0].Count != 1 {
		t.Errorf("Slot mismatch: %+v", inv.Slots[0])
	}
}

func TestInventory_AddItem_Stackable(t *testing.T) {
	inv := NewInventory(1, 10)
	item := &Item{ID: 102, Name: "灵石", StackMax: 999}

	added := inv.AddItem(item, 100)
	if added != 100 {
		t.Errorf("First add returned %d, want 100", added)
	}
	// Stack on existing
	added = inv.AddItem(item, 50)
	if added != 50 {
		t.Errorf("Second add returned %d, want 50", added)
	}
	if len(inv.Slots) != 1 {
		t.Errorf("Expected 1 slot, got %d", len(inv.Slots))
	}
	if inv.Slots[0].Count != 150 {
		t.Errorf("Count = %d, want 150", inv.Slots[0].Count)
	}
}

func TestInventory_AddItem_StackMaxLimit(t *testing.T) {
	inv := NewInventory(1, 10)
	item := &Item{ID: 103, Name: "丹药", StackMax: 10}

	// Add up to stack max
	added := inv.AddItem(item, 5)
	if added != 5 {
		t.Errorf("First add returned %d, want 5", added)
	}
	// Add more, should fill to max
	added = inv.AddItem(item, 10)
	if added != 5 {
		t.Errorf("Second add returned %d, want 5 (filled to max)", added)
	}
	if inv.Slots[0].Count != 10 {
		t.Errorf("Count = %d, want 10", inv.Slots[0].Count)
	}
}

func TestInventory_AddItem_OverflowToNewSlot(t *testing.T) {
	inv := NewInventory(1, 10)
	item := &Item{ID: 104, Name: "符箓", StackMax: 5}

	added := inv.AddItem(item, 5)
	if added != 5 {
		t.Errorf("First add returned %d, want 5", added)
	}
	// Stack max reached, next add goes to new slot
	added = inv.AddItem(item, 3)
	if added != 3 {
		t.Errorf("Second add returned %d, want 3", added)
	}
	if len(inv.Slots) != 2 {
		t.Errorf("Expected 2 slots, got %d", len(inv.Slots))
	}
	if inv.Slots[0].Count != 5 || inv.Slots[1].Count != 3 {
		t.Errorf("Slot counts: [%d, %d], want [5, 3]", inv.Slots[0].Count, inv.Slots[1].Count)
	}
}

func TestInventory_AddItem_FullInventory(t *testing.T) {
	item := &Item{ID: 105, Name: "材料", StackMax: 1}
	inv := NewInventory(1, 2)
	inv.AddItem(item, 1)
	inv.AddItem(item, 1)

	// Inventory full
	added := inv.AddItem(item, 1)
	if added != 0 {
		t.Errorf("AddItem on full inventory returned %d, want 0", added)
	}
}

func TestInventory_AddItem_NonStackable(t *testing.T) {
	inv := NewInventory(1, 10)
	item := &Item{ID: 106, Name: "武器", StackMax: 1}

	added := inv.AddItem(item, 1)
	if added != 1 {
		t.Errorf("First add returned %d, want 1", added)
	}
	// Non-stackable: same ID but new slot
	added = inv.AddItem(item, 1)
	if added != 1 {
		t.Errorf("Second add returned %d, want 1 (new slot)", added)
	}
	if len(inv.Slots) != 2 {
		t.Errorf("Expected 2 slots for non-stackable items, got %d", len(inv.Slots))
	}
}

func TestInventory_AddItem_ExceedStackMax(t *testing.T) {
	inv := NewInventory(1, 10)
	item := &Item{ID: 107, Name: "限量药", StackMax: 5}

	// Attempt to add more than stack max at once
	added := inv.AddItem(item, 8)
	if added != 5 {
		t.Errorf("AddItem(8) on StackMax=5 returned %d, want 5", added)
	}
	if inv.Slots[0].Count != 5 {
		t.Errorf("Count = %d, want 5", inv.Slots[0].Count)
	}
}

func TestInventory_RemoveItem_Normal(t *testing.T) {
	inv := NewInventory(1, 10)
	inv.AddItem(&Item{ID: 101, StackMax: 99}, 50)

	removed := inv.RemoveItem(101, 30)
	if removed != 30 {
		t.Errorf("RemoveItem returned %d, want 30", removed)
	}
	if inv.Slots[0].Count != 20 {
		t.Errorf("Remaining count = %d, want 20", inv.Slots[0].Count)
	}
}

func TestInventory_RemoveItem_Exact(t *testing.T) {
	inv := NewInventory(1, 10)
	inv.AddItem(&Item{ID: 101, StackMax: 99}, 50)

	removed := inv.RemoveItem(101, 50)
	if removed != 50 {
		t.Errorf("RemoveItem returned %d, want 50", removed)
	}
	if len(inv.Slots) != 0 {
		t.Errorf("Expected slot to be removed, got %d slots", len(inv.Slots))
	}
}

func TestInventory_RemoveItem_NotFound(t *testing.T) {
	inv := NewInventory(1, 10)
	inv.AddItem(&Item{ID: 101, StackMax: 99}, 50)

	removed := inv.RemoveItem(999, 10)
	if removed != 0 {
		t.Errorf("RemoveItem for non-existent ID returned %d, want 0", removed)
	}
	if len(inv.Slots) != 1 {
		t.Errorf("Expected 1 slot unchanged, got %d", len(inv.Slots))
	}
}

func TestInventory_RemoveItem_MultipleSlots(t *testing.T) {
	inv := NewInventory(1, 10)
	item := &Item{ID: 101, StackMax: 10}
	// Fill first slot
	inv.AddItem(item, 10)
	// Add to second slot (since first is full)
	inv.AddItem(item, 5)

	removed := inv.RemoveItem(101, 12)
	if removed != 12 {
		t.Errorf("RemoveItem returned %d, want 12", removed)
	}
	// First slot should be removed (10 consumed), second slot should have 3 left
	if len(inv.Slots) != 1 {
		t.Errorf("Expected 1 slot remaining, got %d", len(inv.Slots))
	}
	if inv.Slots[0].Count != 3 {
		t.Errorf("Remaining count = %d, want 3", inv.Slots[0].Count)
	}
}

func TestInventory_RemoveItem_MoreThanAvailable(t *testing.T) {
	inv := NewInventory(1, 10)
	inv.AddItem(&Item{ID: 101, StackMax: 99}, 30)

	removed := inv.RemoveItem(101, 100)
	if removed != 30 {
		t.Errorf("RemoveItem returned %d, want 30 (all available)", removed)
	}
	if len(inv.Slots) != 0 {
		t.Errorf("Expected all slots removed, got %d", len(inv.Slots))
	}
}

func TestInventory_ItemCount(t *testing.T) {
	inv := NewInventory(1, 10)
	item := &Item{ID: 101, StackMax: 10}

	inv.AddItem(item, 7)
	// AddItem only fills to stack max in one call; AddItem(5) with 7 existing
	// adds only 3 (filling to 10) and returns 3. Remaining 2 must be re-added.
	inv.AddItem(item, 3) // fills from 7 to 10

	count := inv.ItemCount(101)
	if count != 10 {
		t.Errorf("ItemCount = %d, want 10", count)
	}

	count = inv.ItemCount(999)
	if count != 0 {
		t.Errorf("ItemCount for non-existent = %d, want 0", count)
	}
}

func TestInventory_ItemCount_Empty(t *testing.T) {
	inv := NewInventory(1, 10)
	if count := inv.ItemCount(101); count != 0 {
		t.Errorf("ItemCount on empty inventory = %d, want 0", count)
	}
}

func TestNewEquipment(t *testing.T) {
	eq := NewEquipment(1001)
	if eq == nil {
		t.Fatal("NewEquipment returned nil")
	}
	if eq.PlayerID != 1001 {
		t.Errorf("PlayerID = %d, want 1001", eq.PlayerID)
	}
	if len(eq.Equipped) != 0 {
		t.Errorf("Expected empty equipped map, got %d items", len(eq.Equipped))
	}
}

func TestEquipment_Equip(t *testing.T) {
	eq := NewEquipment(1)
	old := eq.Equip(EquipSlotWeapon, 2001)
	if old != 0 {
		t.Errorf("First equip should return 0, got %d", old)
	}
	if eq.Equipped[EquipSlotWeapon] != 2001 {
		t.Errorf("Slot should contain item 2001, got %d", eq.Equipped[EquipSlotWeapon])
	}
}

func TestEquipment_Equip_Replace(t *testing.T) {
	eq := NewEquipment(1)
	eq.Equip(EquipSlotWeapon, 2001)

	old := eq.Equip(EquipSlotWeapon, 2002)
	if old != 2001 {
		t.Errorf("Replace should return old item 2001, got %d", old)
	}
	if eq.Equipped[EquipSlotWeapon] != 2002 {
		t.Errorf("Slot should contain item 2002, got %d", eq.Equipped[EquipSlotWeapon])
	}
}

func TestEquipment_Unequip(t *testing.T) {
	eq := NewEquipment(1)
	eq.Equip(EquipSlotArmor, 3001)

	old := eq.Unequip(EquipSlotArmor)
	if old != 3001 {
		t.Errorf("Unequip should return 3001, got %d", old)
	}
	if _, ok := eq.Equipped[EquipSlotArmor]; ok {
		t.Error("Slot should be removed after unequip")
	}
}

func TestEquipment_Unequip_Empty(t *testing.T) {
	eq := NewEquipment(1)

	old := eq.Unequip(EquipSlotRing)
	if old != 0 {
		t.Errorf("Unequip empty slot should return 0, got %d", old)
	}
}

func TestInventory_Gold(t *testing.T) {
	inv := NewInventory(1, 10)
	if inv.Gold != 0 {
		t.Errorf("Initial Gold = %d, want 0", inv.Gold)
	}
	inv.Gold = 99999
	if inv.Gold != 99999 {
		t.Errorf("Gold = %d, want 99999", inv.Gold)
	}
}

func TestMinU32(t *testing.T) {
	tests := []struct {
		a, b, want uint32
	}{
		{1, 2, 1},
		{5, 3, 3},
		{0, 100, 0},
		{100, 100, 100},
	}
	for _, tt := range tests {
		got := minU32(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("minU32(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
