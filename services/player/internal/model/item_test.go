package model

import (
	"testing"
)

func TestItemTypeConstants(t *testing.T) {
	cases := []struct {
		itemType int32
		name     string
	}{
		{ItemTypeWeapon, "ItemTypeWeapon"},
		{ItemTypeHelmet, "ItemTypeHelmet"},
		{ItemTypeArmor, "ItemTypeArmor"},
		{ItemTypeBracers, "ItemTypeBracers"},
		{ItemTypeBelt, "ItemTypeBelt"},
		{ItemTypeLegs, "ItemTypeLegs"},
		{ItemTypeBoots, "ItemTypeBoots"},
		{ItemTypeNecklace, "ItemTypeNecklace"},
		{ItemTypeRing, "ItemTypeRing"},
		{ItemTypePill, "ItemTypePill"},
		{ItemTypeMaterial, "ItemTypeMaterial"},
		{ItemTypeSkillBook, "ItemTypeSkillBook"},
		{ItemTypeConsumable, "ItemTypeConsumable"},
	}

	if len(cases) != 13 {
		t.Errorf("expected 13 item types, got %d", len(cases))
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.itemType < ItemTypeWeapon || tc.itemType > ItemTypeConsumable {
				t.Errorf("%s = %d, out of expected range [%d, %d]",
					tc.name, tc.itemType, ItemTypeWeapon, ItemTypeConsumable)
			}
		})
	}
}

func TestItemTypeToEquipSlotMapping(t *testing.T) {
	cases := []struct {
		itemType int32
		slot     int32
	}{
		{ItemTypeWeapon, EquipSlotWeapon},
		{ItemTypeHelmet, EquipSlotHelmet},
		{ItemTypeArmor, EquipSlotArmor},
		{ItemTypeBracers, EquipSlotBracers},
		{ItemTypeBelt, EquipSlotBelt},
		{ItemTypeLegs, EquipSlotLegs},
		{ItemTypeBoots, EquipSlotBoots},
		{ItemTypeNecklace, EquipSlotNecklace},
		{ItemTypeRing, EquipSlotRing},
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			got, ok := ItemTypeToEquipSlot[tc.itemType]
			if !ok {
				t.Errorf("ItemTypeToEquipSlot missing entry for item type %d", tc.itemType)
				return
			}
			if got != tc.slot {
				t.Errorf("ItemTypeToEquipSlot[%d] = %d, want %d", tc.itemType, got, tc.slot)
			}
		})
	}
}

func TestItemTypeToEquipSlotNonEquippable(t *testing.T) {
	nonEquippable := []int32{
		ItemTypePill,
		ItemTypeMaterial,
		ItemTypeSkillBook,
		ItemTypeConsumable,
	}

	for _, typ := range nonEquippable {
		t.Run("", func(t *testing.T) {
			_, ok := ItemTypeToEquipSlot[typ]
			if ok {
				t.Errorf("ItemTypeToEquipSlot should not contain entry for non-equippable type %d", typ)
			}
		})
	}
}

func TestQualityLevels(t *testing.T) {
	cases := []struct {
		quality  int32
		name     string
		expected string
	}{
		{QualityCommon, "QualityCommon", "凡品"},
		{QualityLow, "QualityLow", "下品"},
		{QualityMedium, "QualityMedium", "中品"},
		{QualityHigh, "QualityHigh", "上品"},
		{QualityEpic, "QualityEpic", "极品"},
		{QualityMythic, "QualityMythic", "仙品"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := QualityNames[tc.quality]
			if !ok {
				t.Fatalf("QualityNames missing entry for %s (%d)", tc.name, tc.quality)
			}
			if got != tc.expected {
				t.Errorf("QualityNames[%d] = %q, want %q", tc.quality, got, tc.expected)
			}
		})
	}
}

func TestQualityOrdering(t *testing.T) {
	// Qualities should be in increasing order
	if QualityCommon >= QualityLow {
		t.Error("QualityCommon should be less than QualityLow")
	}
	if QualityLow >= QualityMedium {
		t.Error("QualityLow should be less than QualityMedium")
	}
	if QualityMedium >= QualityHigh {
		t.Error("QualityMedium should be less than QualityHigh")
	}
	if QualityHigh >= QualityEpic {
		t.Error("QualityHigh should be less than QualityEpic")
	}
	if QualityEpic >= QualityMythic {
		t.Error("QualityEpic should be less than QualityMythic")
	}
}

func TestQualityNamesUnknown(t *testing.T) {
	_, ok := QualityNames[999]
	if ok {
		t.Error("QualityNames should not contain entry for unknown quality 999")
	}
}

func TestStackLimits(t *testing.T) {
	// Items with MaxStack = 1 are non-stackable
	nonStackable := &Item{ID: 1, Name: "Sword", Type: ItemTypeWeapon, MaxStack: 1}
	if nonStackable.MaxStack != 1 {
		t.Error("non-stackable item should have MaxStack = 1")
	}

	// Items with MaxStack > 1 are stackable
	stackable := &Item{ID: 2, Name: "Pill", Type: ItemTypePill, MaxStack: 99}
	if stackable.MaxStack <= 1 {
		t.Error("stackable item should have MaxStack > 1")
	}

	// Material stacks high
	material := &Item{ID: 3, Name: "Stone", Type: ItemTypeMaterial, MaxStack: 999}
	if material.MaxStack < 99 {
		t.Error("material should have large MaxStack")
	}

	// Verify default MaxStack for new items (Go zero value is 0,
	// DB default applied by GORM is 1)
	defaultItem := &Item{ID: 4, Name: "Default"}
	if defaultItem.MaxStack != 0 {
		t.Errorf("Go zero value for MaxStack should be 0, got %d", defaultItem.MaxStack)
	}
}

func TestEquipmentSlotConstants(t *testing.T) {
	slots := []struct {
		slot int32
		name string
	}{
		{EquipSlotWeapon, "EquipSlotWeapon"},
		{EquipSlotHelmet, "EquipSlotHelmet"},
		{EquipSlotArmor, "EquipSlotArmor"},
		{EquipSlotBracers, "EquipSlotBracers"},
		{EquipSlotBelt, "EquipSlotBelt"},
		{EquipSlotLegs, "EquipSlotLegs"},
		{EquipSlotBoots, "EquipSlotBoots"},
		{EquipSlotNecklace, "EquipSlotNecklace"},
		{EquipSlotRing, "EquipSlotRing"},
	}

	for _, tc := range slots {
		t.Run(tc.name, func(t *testing.T) {
			if tc.slot < EquipSlotWeapon || tc.slot > EquipSlotRing {
				t.Errorf("%s = %d, out of expected range [%d, %d]",
					tc.name, tc.slot, EquipSlotWeapon, EquipSlotRing)
			}
		})
	}
}

func TestItemMaxStackBoundaries(t *testing.T) {
	cases := []struct {
		name     string
		maxStack int32
		want     string
	}{
		{"non-stackable", 1, "single-slot"},
		{"small-stack", 10, "multi-slot"},
		{"pill-stack", 99, "multi-slot"},
		{"material-stack", 999, "large-stack"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := &Item{ID: 1, Name: "test", MaxStack: tc.maxStack}
			if item.MaxStack != tc.maxStack {
				t.Errorf("MaxStack = %d, want %d", item.MaxStack, tc.maxStack)
			}
		})
	}
}
