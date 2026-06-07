package models

import (
	"testing"
	"time"
)

func TestPlayer_TotalAttr_AllZero(t *testing.T) {
	p := &Player{
		BaseAttr:  PlayerAttribute{},
		EquipAttr: PlayerAttribute{},
	}
	total := p.TotalAttr()
	expected := PlayerAttribute{}
	if total != expected {
		t.Errorf("TotalAttr with zero base/equip = %+v, want %+v", total, expected)
	}
}

func TestPlayer_TotalAttr_BaseOnly(t *testing.T) {
	p := &Player{
		BaseAttr: PlayerAttribute{
			Health:   1000,
			Mana:     500,
			Strength: 100,
			Agility:  50,
			Spirit:   80,
			Defense:  200,
			Critical: 500,
			Dodge:    300,
		},
		EquipAttr: PlayerAttribute{},
	}
	total := p.TotalAttr()
	if total.Health != 1000 || total.Strength != 100 || total.Defense != 200 {
		t.Errorf("TotalAttr base-only mismatch: %+v", total)
	}
	if total.Critical != 500 || total.Dodge != 300 {
		t.Errorf("TotalAttr percentage fields mismatch: crit=%d dodge=%d", total.Critical, total.Dodge)
	}
}

func TestPlayer_TotalAttr_EquipOnly(t *testing.T) {
	p := &Player{
		BaseAttr: PlayerAttribute{},
		EquipAttr: PlayerAttribute{
			Health:   500,
			Mana:     200,
			Strength: 50,
			Agility:  30,
			Spirit:   20,
			Defense:  100,
			Critical: 200,
			Dodge:    100,
		},
	}
	total := p.TotalAttr()
	if total.Health != 500 || total.Strength != 50 || total.Defense != 100 {
		t.Errorf("TotalAttr equip-only mismatch: %+v", total)
	}
}

func TestPlayer_TotalAttr_Combined(t *testing.T) {
	p := &Player{
		BaseAttr: PlayerAttribute{
			Health:   1000,
			Mana:     500,
			Strength: 100,
			Agility:  50,
			Spirit:   80,
			Defense:  200,
			Critical: 500,
			Dodge:    300,
		},
		EquipAttr: PlayerAttribute{
			Health:   300,
			Mana:     100,
			Strength: 20,
			Agility:  10,
			Spirit:   15,
			Defense:  50,
			Critical: 100,
			Dodge:    50,
		},
	}
	total := p.TotalAttr()
	expected := PlayerAttribute{
		Health:   1300,
		Mana:     600,
		Strength: 120,
		Agility:  60,
		Spirit:   95,
		Defense:  250,
		Critical: 600,
		Dodge:    350,
	}
	if total != expected {
		t.Errorf("TotalAttr combined = %+v, want %+v", total, expected)
	}
}

func TestPlayer_TotalAttr_LargeValues(t *testing.T) {
	p := &Player{
		BaseAttr: PlayerAttribute{
			Health:   999999999999,
			Mana:     888888888888,
			Strength: 777777777777,
			Agility:  666666666666,
			Spirit:   555555555555,
			Defense:  444444444444,
			Critical: 333333333333,
			Dodge:    222222222222,
		},
		EquipAttr: PlayerAttribute{
			Health:   111111111111,
			Mana:     222222222222,
			Strength: 333333333333,
			Agility:  444444444444,
			Spirit:   555555555555,
			Defense:  666666666666,
			Critical: 777777777777,
			Dodge:    888888888888,
		},
	}
	total := p.TotalAttr()
	if total.Health != 1111111111110 {
		t.Errorf("Health = %d, want 1111111111110", total.Health)
	}
	if total.Spirit != 1111111111110 {
		t.Errorf("Spirit = %d, want 1111111111110", total.Spirit)
	}
}

func TestPlayer_TotalAttr_AllFieldsSummed(t *testing.T) {
	// Verify ALL fields are summed
	base := PlayerAttribute{
		Health:   1,
		Mana:     2,
		Strength: 3,
		Agility:  4,
		Spirit:   5,
		Defense:  6,
		Critical: 7,
		Dodge:    8,
	}
	equip := PlayerAttribute{
		Health:   10,
		Mana:     20,
		Strength: 30,
		Agility:  40,
		Spirit:   50,
		Defense:  60,
		Critical: 70,
		Dodge:    80,
	}
	p := &Player{BaseAttr: base, EquipAttr: equip}
	total := p.TotalAttr()

	if total.Health != 11 || total.Mana != 22 {
		t.Errorf("Health/Mana sum: %d/%d, want 11/22", total.Health, total.Mana)
	}
	if total.Strength != 33 || total.Agility != 44 {
		t.Errorf("Strength/Agility sum: %d/%d, want 33/44", total.Strength, total.Agility)
	}
	if total.Spirit != 55 || total.Defense != 66 {
		t.Errorf("Spirit/Defense sum: %d/%d, want 55/66", total.Spirit, total.Defense)
	}
	if total.Critical != 77 || total.Dodge != 88 {
		t.Errorf("Critical/Dodge sum: %d/%d, want 77/88", total.Critical, total.Dodge)
	}
}

func TestPlayer_CanBreakthrough_Valid(t *testing.T) {
	p := &Player{
		RealmID:    1,
		RealmLevel: 5,
		Exp:        5000,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
	}
	// neededExp = 1000 * 5 = 5000, Exp = 5000 >= 5000 => true
	if !p.CanBreakthrough(cfg) {
		t.Error("CanBreakthrough should return true when Exp meets requirement")
	}
}

func TestPlayer_CanBreakthrough_NotEnoughExp(t *testing.T) {
	p := &Player{
		RealmID:    1,
		RealmLevel: 5,
		Exp:        4999,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
	}
	// neededExp = 1000 * 5 = 5000, Exp = 4999 < 5000 => false
	if p.CanBreakthrough(cfg) {
		t.Error("CanBreakthrough should return false when Exp is insufficient")
	}
}

func TestPlayer_CanBreakthrough_MaxLevel(t *testing.T) {
	p := &Player{
		RealmID:    1,
		RealmLevel: 9,
		Exp:        999999,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
	}
	// RealmLevel >= LevelCount => false
	if p.CanBreakthrough(cfg) {
		t.Error("CanBreakthrough should return false at max level")
	}
}

func TestPlayer_CanBreakthrough_LevelOne(t *testing.T) {
	p := &Player{
		RealmID:    1,
		RealmLevel: 1,
		Exp:        1000,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
	}
	// neededExp = 1000 * 1 = 1000, Exp = 1000 >= 1000 => true
	if !p.CanBreakthrough(cfg) {
		t.Error("CanBreakthrough should return true at level 1 with enough exp")
	}
}

func TestPlayer_CanBreakthrough_ZeroExp(t *testing.T) {
	p := &Player{
		RealmID:    1,
		RealmLevel: 3,
		Exp:        0,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
	}
	// neededExp = 1000 * 3 = 3000, Exp = 0 < 3000 => false
	if p.CanBreakthrough(cfg) {
		t.Error("CanBreakthrough should return false with zero exp")
	}
}

func TestPlayer_CanBreakthrough_ZeroLevelCount(t *testing.T) {
	p := &Player{
		RealmID:    1,
		RealmLevel: 1,
		Exp:        99999,
	}
	cfg := &RealmConfig{
		LevelCount:      0,
		BaseExpPerLevel: 1000,
	}
	// RealmLevel(1) >= LevelCount(0) => false
	if p.CanBreakthrough(cfg) {
		t.Error("CanBreakthrough should return false when LevelCount is 0")
	}
}

func TestPlayer_CanBreakthrough_ZeroBaseExp(t *testing.T) {
	p := &Player{
		RealmID:    1,
		RealmLevel: 5,
		Exp:        0,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 0,
	}
	// neededExp = 0, Exp = 0 >= 0 => true
	if !p.CanBreakthrough(cfg) {
		t.Error("CanBreakthrough should return true when BaseExpPerLevel is 0")
	}
}

func TestPlayer_CanBreakthrough_HighLevel_EnoughExp(t *testing.T) {
	p := &Player{
		RealmID:    5,
		RealmLevel: 8,
		Exp:        80000,
	}
	cfg := &RealmConfig{
		ID:              5,
		LevelCount:      9,
		BaseExpPerLevel: 10000,
	}
	// neededExp = 10000 * 8 = 80000, Exp = 80000 >= 80000 => true
	if !p.CanBreakthrough(cfg) {
		t.Error("CanBreakthrough should return true at high level with enough exp")
	}
}

func TestPlayer_CreatedAt(t *testing.T) {
	now := time.Now()
	p := &Player{
		CreatedAt: now,
	}
	if !p.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch: %v vs %v", p.CreatedAt, now)
	}
}

func TestPlayer_Defaults(t *testing.T) {
	p := &Player{}
	if p.ID != 0 || p.Nickname != "" || p.RealmID != 0 || p.RealmLevel != 0 {
		t.Errorf("Default Player has non-zero fields: %+v", p)
	}
	if p.SpiritRoot != "" {
		t.Errorf("Default SpiritRoot should be empty, got %q", p.SpiritRoot)
	}
}

func TestPlayerAttribute_Fields(t *testing.T) {
	attr := PlayerAttribute{
		Health:   1000,
		Mana:     500,
		Strength: 100,
		Agility:  50,
		Spirit:   80,
		Defense:  200,
		Critical: 500,
		Dodge:    300,
	}
	if attr.Health != 1000 || attr.Mana != 500 || attr.Strength != 100 {
		t.Errorf("Core attr mismatch: %+v", attr)
	}
	if attr.Agility != 50 || attr.Spirit != 80 || attr.Defense != 200 {
		t.Errorf("Secondary attr mismatch: %+v", attr)
	}
	if attr.Critical != 500 || attr.Dodge != 300 {
		t.Errorf("Percentage attr mismatch: %+v", attr)
	}
}
