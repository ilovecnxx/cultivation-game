package service

import (
	"sync"
	"testing"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// ---------- mock implementations for inventory ----------

type mockInventoryRepo struct {
	mu        sync.Mutex
	items     map[int64]*model.InventoryItem
	itemDefs  map[int64]*model.Item // item templates
	equipment map[int64]*model.Equipment
	nextItemID int64
	nextEquipID int64
}

func newMockInventoryRepo() *mockInventoryRepo {
	return &mockInventoryRepo{
		items:       make(map[int64]*model.InventoryItem),
		itemDefs:    make(map[int64]*model.Item),
		equipment:   make(map[int64]*model.Equipment),
		nextItemID:  100,
		nextEquipID: 200,
	}
}

func (r *mockInventoryRepo) InsertItem(inv *model.InventoryItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv.ID = r.nextItemID
	r.nextItemID++
	clone := *inv
	clone.Item = nil // don't store the item reference in the map
	r.items[inv.ID] = &clone
	// Also assign back the item ref for the caller
	inv.Item = r.itemDefs[inv.ItemID]
	return nil
}

func (r *mockInventoryRepo) GetInventoryByPlayer(playerID int64) ([]*model.InventoryItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.InventoryItem
	for _, inv := range r.items {
		if inv.PlayerID == playerID {
			clone := *inv
			clone.Item = r.itemDefs[inv.ItemID]
			if clone.Item == nil {
				clone.Item = &model.Item{ID: inv.ItemID, MaxStack: 1}
			}
			result = append(result, &clone)
		}
	}
	if result == nil {
		return []*model.InventoryItem{}, nil
	}
	return result, nil
}

func (r *mockInventoryRepo) GetInventoryItem(inventoryItemID int64) (*model.InventoryItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.items[inventoryItemID]
	if !ok {
		return nil, nil
	}
	clone := *inv
	clone.Item = r.itemDefs[inv.ItemID]
	if clone.Item == nil {
		clone.Item = &model.Item{ID: inv.ItemID, MaxStack: 1}
	}
	return &clone, nil
}

func (r *mockInventoryRepo) FindStackableItem(playerID, itemID int64) (*model.InventoryItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	def, hasDef := r.itemDefs[itemID]
	for _, inv := range r.items {
		if inv.PlayerID == playerID && inv.ItemID == itemID && !inv.IsEquipped {
			maxStack := int32(1)
			if hasDef {
				maxStack = def.MaxStack
			}
			if inv.Quantity < maxStack {
				clone := *inv
				clone.Item = def
				return &clone, nil
			}
		}
	}
	return nil, nil
}

func (r *mockInventoryRepo) UpdateItemQuantity(id int64, quantity int32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.items[id]
	if !ok {
		return errMockNotFound
	}
	inv.Quantity = quantity
	return nil
}

func (r *mockInventoryRepo) UpdateItemSlot(id int64, slotIndex int32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.items[id]
	if !ok {
		return errMockNotFound
	}
	inv.SlotIndex = slotIndex
	return nil
}

func (r *mockInventoryRepo) UpdateItemEquipStatus(id int64, isEquipped bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.items[id]
	if !ok {
		return errMockNotFound
	}
	inv.IsEquipped = isEquipped
	return nil
}

func (r *mockInventoryRepo) DeleteItem(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, id)
	return nil
}

func (r *mockInventoryRepo) GetInventoryCount(playerID int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, inv := range r.items {
		if inv.PlayerID == playerID && !inv.IsEquipped {
			count++
		}
	}
	return count, nil
}

func (r *mockInventoryRepo) InsertEquipment(eq *model.Equipment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	eq.ID = r.nextEquipID
	r.nextEquipID++
	r.equipment[eq.ID] = eq
	return nil
}

func (r *mockInventoryRepo) GetEquipmentByPlayer(playerID int64) ([]*model.Equipment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.Equipment
	for _, eq := range r.equipment {
		if eq.PlayerID == playerID {
			clone := *eq
			result = append(result, &clone)
		}
	}
	return result, nil
}

func (r *mockInventoryRepo) GetEquipmentBySlot(playerID, slot int64) (*model.Equipment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, eq := range r.equipment {
		if eq.PlayerID == playerID && int64(eq.Slot) == slot {
			clone := *eq
			return &clone, nil
		}
	}
	return nil, nil
}

func (r *mockInventoryRepo) DeleteEquipment(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.equipment, id)
	return nil
}

func (r *mockInventoryRepo) DeleteEquipmentBySlot(playerID, slot int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, eq := range r.equipment {
		if eq.PlayerID == playerID && int64(eq.Slot) == slot {
			delete(r.equipment, id)
			return nil
		}
	}
	return nil
}

func (r *mockInventoryRepo) UpdateEquipmentLevel(id int64, level int32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	eq, ok := r.equipment[id]
	if !ok {
		return errMockNotFound
	}
	eq.Level = level
	return nil
}

func (r *mockInventoryRepo) GetItem(itemID int64) (*model.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	def, ok := r.itemDefs[itemID]
	if !ok {
		return nil, nil
	}
	clone := *def
	return &clone, nil
}

func (r *mockInventoryRepo) ListItems(ids []int64) (map[int64]*model.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[int64]*model.Item)
	for _, id := range ids {
		if def, ok := r.itemDefs[id]; ok {
			result[id] = def
		}
	}
	return result, nil
}

// ---------- helpers ----------

func newInventoryService() (*InventoryService, *mockPlayerRepo, *mockInventoryRepo, *mockCache) {
	mpr := newMockPlayerRepo()
	mir := newMockInventoryRepo()
	mc := newMockCache()
	ps := &PlayerService{
		playerRepo: mpr,
		cache:      mc,
		log:        zap.NewNop(),
	}
	is := &InventoryService{
		inventoryRepo: mir,
		playerService: ps,
		cache:         mc,
		log:           zap.NewNop(),
	}
	return is, mpr, mir, mc
}

func registerTestItem(repo *mockInventoryRepo, item *model.Item) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.itemDefs[item.ID] = item
}

func setupTestPlayer(repo *mockPlayerRepo, player *model.Player) int64 {
	_ = repo.Create(player)
	return player.ID
}

func setupInventoryItem(repo *mockInventoryRepo, pID int64, itemID int64, qty int32, slot int32) int64 {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	inv := &model.InventoryItem{
		ID:        repo.nextItemID,
		PlayerID:  pID,
		ItemID:    itemID,
		Quantity:  qty,
		SlotIndex: slot,
	}
	repo.nextItemID++
	repo.items[inv.ID] = inv
	return inv.ID
}

// resultQties is a helper that returns a slice of quantities from inventory results for error messages.
func resultQties(results []*model.InventoryItem) []int32 {
	qties := make([]int32, len(results))
	for i, r := range results {
		qties[i] = r.Quantity
	}
	return qties
}

// ---------- AddItem tests ----------

var pillItem = &model.Item{
	ID: 101, Name: "回血丹", Type: model.ItemTypePill, Quality: model.QualityCommon,
	MaxStack: 99, BaseHP: 50,
}

var swordItem = &model.Item{
	ID: 201, Name: "铁剑", Type: model.ItemTypeWeapon, Quality: model.QualityLow,
	MaxStack: 1, BaseAttack: 10,
}

var materialItem = &model.Item{
	ID: 301, Name: "灵石碎块", Type: model.ItemTypeMaterial, Quality: model.QualityCommon,
	MaxStack: 999,
}

func TestAddItem_NormalStackable(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u1", Name: "p1"})

	results, err := is.AddItem(testCtx, pid, 101, 5)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Quantity != 5 {
		t.Errorf("Quantity = %d, want 5", results[0].Quantity)
	}
	if results[0].SlotIndex != 1 {
		t.Errorf("SlotIndex = %d, want 1", results[0].SlotIndex)
	}
}

func TestAddItem_NonStackable(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, swordItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u2", Name: "p2"})

	results, err := is.AddItem(testCtx, pid, 201, 2)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (one per slot), got %d", len(results))
	}
	// Each sword occupies its own slot
	if results[0].Quantity != 1 {
		t.Errorf("result[0] Quantity = %d, want 1", results[0].Quantity)
	}
	if results[1].Quantity != 1 {
		t.Errorf("result[1] Quantity = %d, want 1", results[1].Quantity)
	}
}

func TestAddItem_StackOverflowCreatesNewSlot(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem) // max stack = 99
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u3", Name: "p3"})

	// Pre-fill a stack with 97 items
	existingID := setupInventoryItem(mir, pid, 101, 97, 1)

	_, _ = mir.GetInventoryItem(existingID)

	// Adding 5 should overflow: 97+5=102, max=99, so 2 get added to existing (filling to 99)
	// and remaining 3 go to a new slot
	results, err := is.AddItem(testCtx, pid, 101, 5)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results (fill + new slot), got %d", len(results))
	}

	// One result should be the filled stack (99), the other the overflow (3)
	found99 := false
	found3 := false
	for _, r := range results {
		if r.Quantity == 99 {
			found99 = true
		}
		if r.Quantity == 3 {
			found3 = true
		}
	}
	if !found99 {
		t.Errorf("expected one result with Quantity 99 (filled stack), got %v", resultQties(results))
	}
	if !found3 {
		t.Errorf("expected one result with Quantity 3 (overflow), got %v", resultQties(results))
	}

	// Total inventory count should be 2 slots
	count, _ := mir.GetInventoryCount(pid)
	if count != 2 {
		t.Errorf("expected 2 inventory slots used, got %d", count)
	}

	// Total quantity across all inventory items should be 102
	allItems, _ := mir.GetInventoryByPlayer(pid)
	totalQty := int32(0)
	for _, item := range allItems {
		totalQty += item.Quantity
	}
	if totalQty != 102 {
		t.Errorf("total quantity in inventory = %d, want 102", totalQty)
	}
}

func TestAddItem_FullInventory(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u4", Name: "p4"})

	// Fill all 60 slots
	for i := 1; i <= 60; i++ {
		setupInventoryItem(mir, pid, int64(900+i), 1, int32(i))
	}

	_, err := is.AddItem(testCtx, pid, 101, 1)
	if err == nil {
		t.Fatal("expected '背包已满' error, got nil")
	}
}

func TestAddItem_InvalidQuantity(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u5", Name: "p5"})

	_, err := is.AddItem(testCtx, pid, 101, 0)
	if err == nil {
		t.Fatal("expected error for quantity <= 0, got nil")
	}
}

func TestAddItem_ItemNotFound(t *testing.T) {
	is, _, _, _ := newInventoryService()
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u6", Name: "p6"})

	_, err := is.AddItem(testCtx, pid, 999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent item, got nil")
	}
}

// ---------- RemoveItem tests ----------

func TestRemoveItem_Partial(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u7", Name: "p7"})
	invID := setupInventoryItem(mir, pid, 101, 50, 1)

	err := is.RemoveItem(testCtx, pid, invID, 10)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}

	inv, _ := mir.GetInventoryItem(invID)
	if inv == nil {
		t.Fatal("inventory item should still exist after partial removal")
	}
	if inv.Quantity != 40 {
		t.Errorf("Quantity = %d, want 40", inv.Quantity)
	}
}

func TestRemoveItem_FullRemovalDeletes(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u8", Name: "p8"})
	invID := setupInventoryItem(mir, pid, 101, 10, 1)

	err := is.RemoveItem(testCtx, pid, invID, 10)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}

	// Item should be deleted
	inv, _ := mir.GetInventoryItem(invID)
	if inv != nil {
		t.Error("inventory item should have been deleted after full removal")
	}
}

func TestRemoveItem_InsufficientQuantity(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u9", Name: "p9"})
	invID := setupInventoryItem(mir, pid, 101, 3, 1)

	err := is.RemoveItem(testCtx, pid, invID, 5)
	if err == nil {
		t.Fatal("expected insufficient quantity error, got nil")
	}
}

func TestRemoveItem_NotFound(t *testing.T) {
	is, _, _, _ := newInventoryService()
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u10", Name: "p10"})

	err := is.RemoveItem(testCtx, pid, 999, 1)
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
}

func TestRemoveItem_WrongPlayer(t *testing.T) {
	is, mpr, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	// Use the shared mpr so player IDs are consistent with the service's repo
	pid1 := setupTestPlayer(mpr, &model.Player{UserID: "u11a", Name: "p11a"})
	pid2 := setupTestPlayer(mpr, &model.Player{UserID: "u11b", Name: "p11b"})
	invID := setupInventoryItem(mir, pid1, 101, 10, 1)

	// Try to remove from wrong player
	err := is.RemoveItem(testCtx, pid2, invID, 1)
	if err == nil {
		t.Fatal("expected error for wrong player, got nil")
	}
}

func TestRemoveItem_Equipped(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, swordItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u12", Name: "p12"})

	// Create equipped item
	mir.mu.Lock()
	invID := mir.nextItemID
	mir.items[invID] = &model.InventoryItem{
		ID: invID, PlayerID: pid, ItemID: 201, Quantity: 1,
		SlotIndex: 1, IsEquipped: true,
	}
	mir.nextItemID++
	mir.mu.Unlock()

	err := is.RemoveItem(testCtx, pid, invID, 1)
	if err == nil {
		t.Fatal("expected error for equipped item, got nil")
	}
}

func TestRemoveItem_InvalidQuantity(t *testing.T) {
	is, _, _, _ := newInventoryService()

	err := is.RemoveItem(testCtx, 1, 1, 0)
	if err == nil {
		t.Fatal("expected error for quantity <= 0, got nil")
	}
}

// ---------- TransferItem tests ----------

func TestTransferItem_MoveToEmptySlot(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u13", Name: "p13"})
	_ = setupInventoryItem(mir, pid, 101, 5, 1)

	req := &model.InventoryTransferRequest{FromSlot: 1, ToSlot: 5, Quantity: 1}
	err := is.TransferItem(testCtx, pid, req)
	if err != nil {
		t.Fatalf("TransferItem failed: %v", err)
	}

	inv, _ := mir.GetInventoryItem(100)
	if inv.SlotIndex != 5 {
		t.Errorf("SlotIndex = %d, want 5", inv.SlotIndex)
	}
}

func TestTransferItem_SwapDifferentItems(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	registerTestItem(mir, swordItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u14", Name: "p14"})
	id1 := setupInventoryItem(mir, pid, 101, 5, 1)
	id2 := setupInventoryItem(mir, pid, 201, 1, 2)

	req := &model.InventoryTransferRequest{FromSlot: 1, ToSlot: 2, Quantity: 1}
	err := is.TransferItem(testCtx, pid, req)
	if err != nil {
		t.Fatalf("TransferItem failed: %v", err)
	}

	// After swap, item1 should be at slot 2, item2 at slot 1
	inv1, _ := mir.GetInventoryItem(id1)
	inv2, _ := mir.GetInventoryItem(id2)
	if inv1.SlotIndex != 2 {
		t.Errorf("item1 SlotIndex = %d, want 2", inv1.SlotIndex)
	}
	if inv2.SlotIndex != 1 {
		t.Errorf("item2 SlotIndex = %d, want 1", inv2.SlotIndex)
	}
}

func TestTransferItem_MergeStackable(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem) // max stack = 99
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u15", Name: "p15"})
	_ = setupInventoryItem(mir, pid, 101, 40, 1) // from: 40 pills
	_ = setupInventoryItem(mir, pid, 101, 30, 2) // to: 30 pills, has space for 69

	req := &model.InventoryTransferRequest{FromSlot: 1, ToSlot: 2, Quantity: 1}
	err := is.TransferItem(testCtx, pid, req)
	if err != nil {
		t.Fatalf("TransferItem merge failed: %v", err)
	}

	// Items should be merged: 40 + 30 = 70 total
	// Find the remaining item
	inv2, _ := mir.GetInventoryItem(101) // id 101 = second item
	inv1, _ := mir.GetInventoryItem(100) // id 100 = first item
	// One of these should be nil (deleted)
	if inv1 != nil && inv2 != nil {
		// Both exist means partial merge
		if inv1.Quantity+inv2.Quantity != 70 {
			t.Errorf("total quantity = %d, want 70", inv1.Quantity+inv2.Quantity)
		}
	} else if inv1 != nil {
		if inv1.Quantity != 70 {
			t.Errorf("merged quantity = %d, want 70", inv1.Quantity)
		}
	} else if inv2 != nil {
		if inv2.Quantity != 70 {
			t.Errorf("merged quantity = %d, want 70", inv2.Quantity)
		}
	} else {
		t.Fatal("both items were deleted")
	}
}

func TestTransferItem_SourceSlotEmpty(t *testing.T) {
	is, _, _, _ := newInventoryService()
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u16", Name: "p16"})

	req := &model.InventoryTransferRequest{FromSlot: 1, ToSlot: 2, Quantity: 1}
	err := is.TransferItem(testCtx, pid, req)
	if err == nil {
		t.Fatal("expected error for empty source slot, got nil")
	}
}

// ---------- SortInventory tests ----------

func TestSortInventory_ByType(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{ID: 1, Name: "Sword", Type: model.ItemTypeWeapon, MaxStack: 1})
	registerTestItem(mir, &model.Item{ID: 2, Name: "Pill", Type: model.ItemTypePill, MaxStack: 99})
	registerTestItem(mir, &model.Item{ID: 3, Name: "Armor", Type: model.ItemTypeArmor, MaxStack: 1})

	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u17", Name: "p17"})
	setupInventoryItem(mir, pid, 1, 1, 3) // slot 3
	setupInventoryItem(mir, pid, 2, 5, 1) // slot 1
	setupInventoryItem(mir, pid, 3, 1, 2) // slot 2

	req := &model.SortInventoryRequest{SortBy: "type", Desc: false}
	results, err := is.SortInventory(testCtx, pid, req)
	if err != nil {
		t.Fatalf("SortInventory failed: %v", err)
	}

	// Sorted by type ascending: Weapon (1) < Armor (3) < Pill (10)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	types := make([]int32, len(results))
	for i, r := range results {
		types[i] = r.Item.Type
	}

	if types[0] != model.ItemTypeWeapon || types[1] != model.ItemTypeArmor || types[2] != model.ItemTypePill {
		t.Errorf("sort order by type asc: got %v, want [1 3 10]", types)
	}
}

func TestSortInventory_ByQualityDesc(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{ID: 1, Name: "Common", Type: 1, Quality: model.QualityCommon, MaxStack: 1})
	registerTestItem(mir, &model.Item{ID: 2, Name: "Epic", Type: 2, Quality: model.QualityEpic, MaxStack: 1})
	registerTestItem(mir, &model.Item{ID: 3, Name: "Rare", Type: 3, Quality: model.QualityHigh, MaxStack: 1})

	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u18", Name: "p18"})
	setupInventoryItem(mir, pid, 1, 1, 1)
	setupInventoryItem(mir, pid, 2, 1, 2)
	setupInventoryItem(mir, pid, 3, 1, 3)

	req := &model.SortInventoryRequest{SortBy: "quality", Desc: true}
	results, err := is.SortInventory(testCtx, pid, req)
	if err != nil {
		t.Fatalf("SortInventory failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	quals := make([]int32, len(results))
	for i, r := range results {
		quals[i] = r.Item.Quality
	}

	// Descending: Epic (5) > High (4) > Common (1)
	if quals[0] != model.QualityEpic || quals[1] != model.QualityHigh || quals[2] != model.QualityCommon {
		t.Errorf("sort order by quality desc: got %v, want [5 4 1]", quals)
	}
}

func TestSortInventory_FiltersEquipped(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{ID: 1, Name: "Sword", Type: model.ItemTypeWeapon, MaxStack: 1})
	registerTestItem(mir, &model.Item{ID: 2, Name: "Pill", Type: model.ItemTypePill, MaxStack: 99})

	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u19", Name: "p19"})
	// Unequipped item
	setupInventoryItem(mir, pid, 1, 1, 1)
	// Equipped item (should be filtered out)
	mir.mu.Lock()
	eqID := mir.nextItemID
	mir.items[eqID] = &model.InventoryItem{
		ID: eqID, PlayerID: pid, ItemID: 2, Quantity: 5,
		SlotIndex: 2, IsEquipped: true,
	}
	mir.nextItemID++
	mir.mu.Unlock()

	req := &model.SortInventoryRequest{SortBy: "type", Desc: false}
	results, err := is.SortInventory(testCtx, pid, req)
	if err != nil {
		t.Fatalf("SortInventory failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 non-equipped item, got %d", len(results))
	}
}

// ---------- UseItem tests ----------

func TestUseItem_Pill(t *testing.T) {
	is, mpr, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{
		ID: 101, Name: "回血丹", Type: model.ItemTypePill,
		MaxStack: 99, BaseHP: 50, BaseMP: 0, BaseAttack: 0,
	})
	pid := setupTestPlayer(mpr, &model.Player{
		UserID:     "u20", Name: "p20",
		HP: 100, MaxHP: 200, MP: 50, MaxMP: 100, SpiritPower: 0,
	})
	invID := setupInventoryItem(mir, pid, 101, 5, 1)

	req := &model.UseItemRequest{InventoryItemID: invID, Quantity: 2}
	result, err := is.UseItem(testCtx, pid, req)
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	// Pill: BaseHP*quantity = 100 (HP), BaseAttack*quantity = 0 (spirit)
	// But ItemTypePill overrides spirit to 10*quantity = 20
	if result["hp"] != 100 {
		t.Errorf("hp effect = %d, want 100", result["hp"])
	}
	if result["spirit_power"] != 20 {
		t.Errorf("spirit_power effect = %d, want 20", result["spirit_power"])
	}

	// Player HP should have increased (capped at MaxHP)
	player, _ := mpr.GetByID(pid)
	if player.HP != 200 {
		t.Errorf("player HP = %d, want 200 (capped)", player.HP)
	}
	if player.SpiritPower != 20 {
		t.Errorf("player SpiritPower = %d, want 20", player.SpiritPower)
	}

	// Inventory item should have been decremented
	inv, _ := mir.GetInventoryItem(invID)
	if inv.Quantity != 3 {
		t.Errorf("remaining quantity = %d, want 3", inv.Quantity)
	}
}

func TestUseItem_Consumable(t *testing.T) {
	is, mpr, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{
		ID: 401, Name: "经验丹", Type: model.ItemTypeConsumable,
		MaxStack: 99, BaseHP: 0, BaseMP: 10, BaseAttack: 5,
	})
	pid := setupTestPlayer(mpr, &model.Player{
		UserID: "u21", Name: "p21",
		HP: 150, MaxHP: 200, MP: 30, MaxMP: 100, Experience: 0, SpiritPower: 0,
	})
	invID := setupInventoryItem(mir, pid, 401, 3, 1)

	req := &model.UseItemRequest{InventoryItemID: invID, Quantity: 2}
	result, err := is.UseItem(testCtx, pid, req)
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	// Consumable: BaseHP*qty = 0, BaseMP*qty = 20, BaseAttack*qty used as spirit = 10
	if result["mp"] != 20 {
		t.Errorf("mp effect = %d, want 20", result["mp"])
	}
	if result["spirit_power"] != 10 {
		t.Errorf("spirit_power effect = %d, want 10", result["spirit_power"])
	}

	player, _ := mpr.GetByID(pid)
	if player.MP != 50 {
		t.Errorf("player MP = %d, want 50", player.MP)
	}
	if player.SpiritPower != 10 {
		t.Errorf("player SpiritPower = %d, want 10", player.SpiritPower)
	}
}

func TestUseItem_InvalidType(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, swordItem) // ItemTypeWeapon, not consumable
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u22", Name: "p22"})
	invID := setupInventoryItem(mir, pid, 201, 1, 1)

	req := &model.UseItemRequest{InventoryItemID: invID, Quantity: 1}
	_, err := is.UseItem(testCtx, pid, req)
	if err == nil {
		t.Fatal("expected error for non-usable item type, got nil")
	}
}

func TestUseItem_NotFound(t *testing.T) {
	is, _, _, _ := newInventoryService()
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u23", Name: "p23"})

	req := &model.UseItemRequest{InventoryItemID: 999, Quantity: 1}
	_, err := is.UseItem(testCtx, pid, req)
	if err == nil {
		t.Fatal("expected error for non-existent item, got nil")
	}
}

func TestUseItem_InsufficientQuantity(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u24", Name: "p24"})
	invID := setupInventoryItem(mir, pid, 101, 1, 1)

	req := &model.UseItemRequest{InventoryItemID: invID, Quantity: 5}
	_, err := is.UseItem(testCtx, pid, req)
	if err == nil {
		t.Fatal("expected insufficient quantity error, got nil")
	}
}

// ---------- GetInventory tests ----------

func TestGetInventory_ReturnsItems(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, pillItem)
	registerTestItem(mir, swordItem)
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u25", Name: "p25"})
	setupInventoryItem(mir, pid, 101, 10, 1)
	setupInventoryItem(mir, pid, 201, 1, 2)

	items, err := is.GetInventory(testCtx, pid)
	if err != nil {
		t.Fatalf("GetInventory failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestGetInventory_Empty(t *testing.T) {
	is, _, _, _ := newInventoryService()
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{UserID: "u26", Name: "p26"})

	items, err := is.GetInventory(testCtx, pid)
	if err != nil {
		t.Fatalf("GetInventory failed: %v", err)
	}
	if items == nil || len(items) != 0 {
		t.Errorf("expected empty slice, got %v", items)
	}
}

// ---------- Equip / Unequip tests ----------

func TestEquipItem(t *testing.T) {
	is, mpr, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{
		ID: 201, Name: "铁剑", Type: model.ItemTypeWeapon, Quality: model.QualityLow,
		MaxStack: 1, BaseAttack: 10, RequiredLevel: 1, RequiredRealm: 1,
	})
	pid := setupTestPlayer(mpr, &model.Player{
		UserID: "u27", Name: "p27", Level: 5, Realm: model.RealmQiRef,
		Attack: 10, MaxHP: 100,
	})
	invID := setupInventoryItem(mir, pid, 201, 1, 1)

	req := &model.EquipRequest{InventoryItemID: invID}
	equip, err := is.EquipItem(testCtx, pid, req)
	if err != nil {
		t.Fatalf("EquipItem failed: %v", err)
	}
	if equip == nil {
		t.Fatal("expected non-nil equipment")
	}
	if equip.Slot != 1 {
		t.Errorf("Slot = %d, want 1 (weapon slot)", equip.Slot)
	}

	// Player stats should have been updated (BaseAttack added)
	player, _ := mpr.GetByID(pid)
	if player.Attack != 20 {
		t.Errorf("Attack = %d, want 20 (10 base + 10 weapon)", player.Attack)
	}
}

func TestEquipItem_LevelRequirement(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{
		ID: 501, Name: "神剑", Type: model.ItemTypeWeapon,
		RequiredLevel: 50, RequiredRealm: 1, MaxStack: 1,
	})
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{
		UserID: "u28", Name: "p28", Level: 1, Realm: model.RealmMortal,
	})
	invID := setupInventoryItem(mir, pid, 501, 1, 1)

	req := &model.EquipRequest{InventoryItemID: invID}
	_, err := is.EquipItem(testCtx, pid, req)
	if err == nil {
		t.Fatal("expected level requirement error, got nil")
	}
}

func TestUnequipItem(t *testing.T) {
	is, mpr, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{
		ID: 201, Name: "铁剑", Type: model.ItemTypeWeapon,
		MaxStack: 1, BaseAttack: 10,
	})
	pid := setupTestPlayer(mpr, &model.Player{
		UserID: "u29", Name: "p29", Level: 10, Realm: model.RealmBase,
		Attack: 20, MaxHP: 100,
	})
	invID := setupInventoryItem(mir, pid, 201, 1, 1)

	// First equip
	_, _ = is.EquipItem(testCtx, pid, &model.EquipRequest{InventoryItemID: invID})

	// Then unequip
	unequipReq := &model.UnequipRequest{Slot: 1}
	invItem, err := is.UnequipItem(testCtx, pid, unequipReq)
	if err != nil {
		t.Fatalf("UnequipItem failed: %v", err)
	}
	if invItem == nil {
		t.Fatal("expected non-nil inventory item")
	}

	// Player stats should be decremented back to starting value (20 - 10 + 10 - 10 = 20)
	// Starting attack was 20, weapon gave +10, unequip removes +10
	player, _ := mpr.GetByID(pid)
	if player.Attack != 20 {
		t.Errorf("Attack after unequip = %d, want 20", player.Attack)
	}
}

// ---------- StrengthenEquipment tests ----------

func TestStrengthenEquipment(t *testing.T) {
	is, mpr, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{
		ID: 201, Name: "铁剑", Type: model.ItemTypeWeapon,
		MaxStack: 1, BaseAttack: 10,
	})
	pid := setupTestPlayer(mpr, &model.Player{
		UserID: "u30", Name: "p30", Level: 10, Realm: model.RealmBase,
		Gold: 1000, Attack: 20, MaxHP: 100,
	})
	invID := setupInventoryItem(mir, pid, 201, 1, 1)
	_, _ = is.EquipItem(testCtx, pid, &model.EquipRequest{InventoryItemID: invID})

	strengthenReq := &model.StrengthenRequest{Slot: 1}
	equip, err := is.StrengthenEquipment(testCtx, pid, strengthenReq)
	if err != nil {
		t.Fatalf("StrengthenEquipment failed: %v", err)
	}
	if equip == nil {
		t.Fatal("expected non-nil equipment")
	}
	if equip.Level != 1 {
		t.Errorf("Level = %d, want 1", equip.Level)
	}

	// Gold should be deducted (100 + 0*50 = 100)
	player, _ := mpr.GetByID(pid)
	if player.Gold >= 1000 {
		t.Errorf("Gold was not deducted: %d", player.Gold)
	}
}

func TestStrengthenEquipment_MaxLevel(t *testing.T) {
	is, _, mir, _ := newInventoryService()
	registerTestItem(mir, &model.Item{
		ID: 201, Name: "铁剑", Type: model.ItemTypeWeapon, MaxStack: 1,
	})
	pid := setupTestPlayer(newMockPlayerRepo(), &model.Player{
		UserID: "u31", Name: "p31", Gold: 99999,
	})
	invID := setupInventoryItem(mir, pid, 201, 1, 1)

	// Insert equipment at max level
	mir.mu.Lock()
	eqID := mir.nextEquipID
	mir.equipment[eqID] = &model.Equipment{
		ID: eqID, PlayerID: pid, Slot: 1, InventoryItemID: invID,
		ItemID: 201, Level: 20,
	}
	mir.nextEquipID++
	mir.mu.Unlock()

	req := &model.StrengthenRequest{Slot: 1}
	_, err := is.StrengthenEquipment(testCtx, pid, req)
	if err == nil {
		t.Fatal("expected max level error, got nil")
	}
}

// ---------- calcUseEffect / enhanceCost / successRate (pure logic) ----------

func TestCalcUseEffect_Pill(t *testing.T) {
	is, _, _, _ := newInventoryService()
	item := &model.Item{ID: 1, Type: model.ItemTypePill, BaseHP: 50, BaseMP: 30, BaseAttack: 0}
	// Pill: uses BaseHP*quantity for HP, BaseAttack*quantity for spirit,
	// but ItemTypePill overrides spirit to 10*quantity
	effect := is.calcUseEffect(item, 3)
	if effect.hp != 150 {
		t.Errorf("hp = %d, want 150", effect.hp)
	}
	if effect.mp != 90 {
		t.Errorf("mp = %d, want 90", effect.mp)
	}
	if effect.spirit != 30 {
		t.Errorf("spirit = %d, want 30", effect.spirit)
	}
}

func TestCalcUseEffect_Consumable(t *testing.T) {
	is, _, _, _ := newInventoryService()
	item := &model.Item{ID: 2, Type: model.ItemTypeConsumable, BaseHP: 10, BaseMP: 20, BaseAttack: 5}
	effect := is.calcUseEffect(item, 2)
	if effect.hp != 20 {
		t.Errorf("hp = %d, want 20", effect.hp)
	}
	if effect.mp != 40 {
		t.Errorf("mp = %d, want 40", effect.mp)
	}
	// For non-pill: spirit = BaseAttack * quantity
	if effect.spirit != 10 {
		t.Errorf("spirit = %d, want 10", effect.spirit)
	}
}

func TestEnhanceCost(t *testing.T) {
	is, _, _, _ := newInventoryService()

	gold1, mat1 := is.enhanceCost(0) // level 0
	if gold1 != 100 {
		t.Errorf("gold cost for level 0 = %d, want 100", gold1)
	}
	if mat1 != 0 {
		t.Errorf("material for level 0 = %d, want 0", mat1)
	}

	gold5, mat5 := is.enhanceCost(5) // level 5
	if mat5 != 1001 {
		t.Errorf("material for level 5 = %d, want 1001", mat5)
	}

	gold10, mat10 := is.enhanceCost(10) // level 10
	if mat10 != 1002 {
		t.Errorf("material for level 10 = %d, want 1002", mat10)
	}
	if gold10 <= gold5 {
		t.Errorf("gold cost at lv10 (%d) should be > lv5 (%d)", gold10, gold5)
	}
}

func TestSuccessRate(t *testing.T) {
	is, _, _, _ := newInventoryService()

	cases := []struct {
		level int32
		want  float64
	}{
		{0, 1.0},
		{2, 1.0},
		{3, 0.8},
		{4, 0.8},
		{5, 0.6},
		{7, 0.6},
		{8, 0.4},
		{11, 0.4},
		{12, 0.2},
		{15, 0.2},
		{16, 0.1},
		{19, 0.1},
		{20, 0.1},
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			got := is.successRate(tc.level)
			if got != tc.want {
				t.Errorf("successRate(%d) = %f, want %f", tc.level, got, tc.want)
			}
		})
	}
}

// ---------- applyEquipmentStats (pure logic) ----------

func TestApplyEquipmentStats_Equip(t *testing.T) {
	is, _, _, _ := newInventoryService()
	player := &model.Player{MaxHP: 100, MaxMP: 50, Attack: 10, Defense: 5, HP: 80, MP: 30}
	item := &model.Item{BaseHP: 20, BaseMP: 10, BaseAttack: 5, BaseDefense: 3}

	is.applyEquipmentStats(player, item, 0, true)

	if player.MaxHP != 120 {
		t.Errorf("MaxHP = %d, want 120", player.MaxHP)
	}
	if player.MaxMP != 60 {
		t.Errorf("MaxMP = %d, want 60", player.MaxMP)
	}
	if player.Attack != 15 {
		t.Errorf("Attack = %d, want 15", player.Attack)
	}
	if player.Defense != 8 {
		t.Errorf("Defense = %d, want 8", player.Defense)
	}
}

func TestApplyEquipmentStats_Unequip(t *testing.T) {
	is, _, _, _ := newInventoryService()
	player := &model.Player{MaxHP: 120, MaxMP: 60, Attack: 15, Defense: 8, HP: 80, MP: 30}
	item := &model.Item{BaseHP: 20, BaseMP: 10, BaseAttack: 5, BaseDefense: 3}

	is.applyEquipmentStats(player, item, 0, false)

	if player.MaxHP != 100 {
		t.Errorf("MaxHP = %d, want 100", player.MaxHP)
	}
	if player.Attack != 10 {
		t.Errorf("Attack = %d, want 10", player.Attack)
	}
}

func TestApplyEquipmentStats_NegativePrevention(t *testing.T) {
	is, _, _, _ := newInventoryService()
	player := &model.Player{MaxHP: 5, MaxMP: 3, Attack: 2, Defense: 1}
	item := &model.Item{BaseHP: 10, BaseMP: 5, BaseAttack: 5, BaseDefense: 3}

	is.applyEquipmentStats(player, item, 0, false)

	if player.MaxHP != 1 {
		t.Errorf("MaxHP = %d, want 1 (min)", player.MaxHP)
	}
	if player.Attack != 1 {
		t.Errorf("Attack = %d, want 1 (min)", player.Attack)
	}
}

func TestApplyEquipmentStats_EnhanceBonus(t *testing.T) {
	is, _, _, _ := newInventoryService()
	player := &model.Player{MaxHP: 100, Attack: 10}
	item := &model.Item{BaseHP: 20, BaseAttack: 5}

	is.applyEquipmentStats(player, item, 10, true)

	// enhanceBonus = 1 + 10*10/100 = 2 (100% bonus at level 10)
	// MaxHP += 20 * 2 = 40
	// Attack += 5 * 2 = 10
	if player.MaxHP != 140 {
		t.Errorf("MaxHP = %d, want 140 (100 + 20*2)", player.MaxHP)
	}
	if player.Attack != 20 {
		t.Errorf("Attack = %d, want 20 (10 + 5*2)", player.Attack)
	}
}
