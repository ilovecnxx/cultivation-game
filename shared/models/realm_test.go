package models

import (
	"testing"
)

func TestRealmConfig_DefaultValues(t *testing.T) {
	cfg := RealmConfig{}
	if cfg.ID != 0 || cfg.Name != "" || cfg.LevelCount != 0 {
		t.Errorf("Default RealmConfig has unexpected non-zero fields: %+v", cfg)
	}
	if cfg.BaseExpPerLevel != 0 || cfg.ExpMultiplier != 0 {
		t.Errorf("Default exp fields should be zero: %+v", cfg)
	}
}

func TestRealmConfig_ExpForLevel_LevelOne(t *testing.T) {
	cfg := RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.5,
	}
	// Level 1: no multiplier applied, just base
	exp := cfg.ExpForLevel(1)
	if exp != 1000 {
		t.Errorf("ExpForLevel(1) = %d, want 1000", exp)
	}
}

func TestRealmConfig_ExpForLevel_MultiplierApplied(t *testing.T) {
	cfg := RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   2.0,
	}
	// Level 2: 1000 * 2.0 = 2000
	exp := cfg.ExpForLevel(2)
	if exp != 2000 {
		t.Errorf("ExpForLevel(2) = %d, want 2000", exp)
	}
	// Level 3: 1000 * 2.0 * 2.0 = 4000
	exp = cfg.ExpForLevel(3)
	if exp != 4000 {
		t.Errorf("ExpForLevel(3) = %d, want 4000", exp)
	}
	// Level 4: 1000 * 2.0^3 = 8000
	exp = cfg.ExpForLevel(4)
	if exp != 8000 {
		t.Errorf("ExpForLevel(4) = %d, want 8000", exp)
	}
}

func TestRealmConfig_ExpForLevel_FractionalMultiplier(t *testing.T) {
	cfg := RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.1,
	}
	// Level 3: 1000 * 1.1 * 1.1 = 1210
	exp := cfg.ExpForLevel(3)
	if exp != 1210 {
		t.Errorf("ExpForLevel(3) = %d, want 1210", exp)
	}
}

func TestRealmConfig_ExpForLevel_ZeroLevel(t *testing.T) {
	cfg := RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.5,
	}
	exp := cfg.ExpForLevel(0)
	if exp != 0 {
		t.Errorf("ExpForLevel(0) = %d, want 0", exp)
	}
}

func TestRealmConfig_ExpForLevel_AboveMax(t *testing.T) {
	cfg := RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.5,
	}
	exp := cfg.ExpForLevel(10)
	if exp != 0 {
		t.Errorf("ExpForLevel(10) = %d, want 0 (above max)", exp)
	}
}

func TestRealmConfig_ExpForLevel_MultiplierOne(t *testing.T) {
	cfg := RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.0,
	}
	// All levels should return the same base
	for level := uint32(1); level <= 9; level++ {
		exp := cfg.ExpForLevel(level)
		if exp != 1000 {
			t.Errorf("ExpForLevel(%d) with multiplier 1.0 = %d, want 1000", level, exp)
		}
	}
}

func TestRealmConfig_ExpForLevel_LargeMultiplier(t *testing.T) {
	cfg := RealmConfig{
		LevelCount:      5,
		BaseExpPerLevel: 10,
		ExpMultiplier:   10.0,
	}
	// Level 5: 10 * 10^4 = 100000
	exp := cfg.ExpForLevel(5)
	if exp != 100000 {
		t.Errorf("ExpForLevel(5) = %d, want 100000", exp)
	}
}

func TestBreakthroughConfig_Defaults(t *testing.T) {
	bc := BreakthroughConfig{}
	if bc.FromRealmID != 0 || bc.ToRealmID != 0 {
		t.Errorf("Default FromRealmID/ToRealmID should be 0: %+v", bc)
	}
	if bc.BaseRate != 0 {
		t.Errorf("Default BaseRate = %f, want 0", bc.BaseRate)
	}
	if bc.MaxRate != 0 {
		t.Errorf("Default MaxRate = %f, want 0", bc.MaxRate)
	}
	if bc.SpiritRootBonus != nil {
		t.Errorf("Default SpiritRootBonus should be nil")
	}
	if bc.ItemBonus != nil {
		t.Errorf("Default ItemBonus should be nil")
	}
}

func TestBreakthroughConfig_SpiritRootBonus(t *testing.T) {
	bc := BreakthroughConfig{
		FromRealmID: 1,
		ToRealmID:   2,
		BaseRate:    0.3,
		MaxRate:     0.9,
		SpiritRootBonus: map[string]float64{
			"金灵根": 0.1,
			"变异雷灵根": 0.2,
		},
		ItemBonus: map[uint32]float64{
			101: 0.05,
			102: 0.10,
		},
		FailedDropLevels:  1,
		FailedCooldownSec: 3600,
	}
	if bc.FromRealmID != 1 || bc.ToRealmID != 2 {
		t.Errorf("Realm IDs: from=%d to=%d", bc.FromRealmID, bc.ToRealmID)
	}
	if bc.BaseRate != 0.3 {
		t.Errorf("BaseRate = %f, want 0.3", bc.BaseRate)
	}
	if bonus, ok := bc.SpiritRootBonus["金灵根"]; !ok || bonus != 0.1 {
		t.Errorf("金灵根 bonus = %f, want 0.1", bonus)
	}
	if bonus, ok := bc.ItemBonus[101]; !ok || bonus != 0.05 {
		t.Errorf("Item 101 bonus = %f, want 0.05", bonus)
	}
	if bc.MaxRate != 0.9 {
		t.Errorf("MaxRate = %f, want 0.9", bc.MaxRate)
	}
	if bc.FailedDropLevels != 1 {
		t.Errorf("FailedDropLevels = %d, want 1", bc.FailedDropLevels)
	}
	if bc.FailedCooldownSec != 3600 {
		t.Errorf("FailedCooldownSec = %d, want 3600", bc.FailedCooldownSec)
	}
}

func TestRealmProgress_ExpToNextLevel_Normal(t *testing.T) {
	rp := RealmProgress{
		CurrentRealm: 1,
		CurrentLevel: 3,
		CurrentExp:   500,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.0,
	}
	// Level 4 needs 1000 exp, current 500, so need 500 more
	needed := rp.ExpToNextLevel(cfg)
	if needed != 500 {
		t.Errorf("ExpToNextLevel = %d, want 500", needed)
	}
}

func TestRealmProgress_ExpToNextLevel_AlreadyEnough(t *testing.T) {
	rp := RealmProgress{
		CurrentRealm: 1,
		CurrentLevel: 3,
		CurrentExp:   1500,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.0,
	}
	// Have more than needed for next level
	needed := rp.ExpToNextLevel(cfg)
	if needed != 0 {
		t.Errorf("ExpToNextLevel when already enough = %d, want 0", needed)
	}
}

func TestRealmProgress_ExpToNextLevel_MaxLevel(t *testing.T) {
	rp := RealmProgress{
		CurrentRealm: 1,
		CurrentLevel: 9,
		CurrentExp:   0,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
	}
	needed := rp.ExpToNextLevel(cfg)
	if needed != 0 {
		t.Errorf("ExpToNextLevel at max level = %d, want 0", needed)
	}
}

func TestRealmProgress_ExpToNextLevel_ZeroExp(t *testing.T) {
	rp := RealmProgress{
		CurrentRealm: 1,
		CurrentLevel: 1,
		CurrentExp:   0,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.0,
	}
	needed := rp.ExpToNextLevel(cfg)
	if needed != 1000 {
		t.Errorf("ExpToNextLevel with zero exp = %d, want 1000", needed)
	}
}

func TestRealmProgress_ExpToNextLevel_WithMultiplier(t *testing.T) {
	rp := RealmProgress{
		CurrentRealm: 1,
		CurrentLevel: 2,
		CurrentExp:   500,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.5,
	}
	// Level 3 needs: 1000 * 1.5 * 1.5 = 2250. Current 500, need 1750
	needed := rp.ExpToNextLevel(cfg)
	if needed != 1750 {
		t.Errorf("ExpToNextLevel = %d, want 1750", needed)
	}
}

func TestRealmProgress_ExpToNextLevel_ExactlyNextLevel(t *testing.T) {
	rp := RealmProgress{
		CurrentRealm: 1,
		CurrentLevel: 4,
		CurrentExp:   1000,
	}
	cfg := &RealmConfig{
		LevelCount:      9,
		BaseExpPerLevel: 1000,
		ExpMultiplier:   1.0,
	}
	// Level 5 needs 1000, current 1000 >= 1000, return 0
	needed := rp.ExpToNextLevel(cfg)
	if needed != 0 {
		t.Errorf("ExpToNextLevel when exactly at threshold = %d, want 0", needed)
	}
}

func TestRealmConfig_ExpForLevel_WithMultiplierOneValue(t *testing.T) {
	cfg := RealmConfig{
		LevelCount:      3,
		BaseExpPerLevel: 500,
		ExpMultiplier:   1.5,
	}
	exp1 := cfg.ExpForLevel(1) // 500
	exp2 := cfg.ExpForLevel(2) // 500 * 1.5 = 750
	exp3 := cfg.ExpForLevel(3) // 500 * 1.5 * 1.5 = 1125

	if exp1 != 500 {
		t.Errorf("Level 1 exp = %d, want 500", exp1)
	}
	if exp2 != 750 {
		t.Errorf("Level 2 exp = %d, want 750", exp2)
	}
	if exp3 != 1125 {
		t.Errorf("Level 3 exp = %d, want 1125", exp3)
	}
}

func TestRealmProgress_Fields(t *testing.T) {
	rp := RealmProgress{
		PlayerID:      1001,
		CurrentRealm:  1,
		CurrentLevel:  5,
		CurrentExp:    5000,
		Breakthroughs: 2,
		FailedCount:   1,
	}
	if rp.PlayerID != 1001 {
		t.Errorf("PlayerID = %d, want 1001", rp.PlayerID)
	}
	if rp.Breakthroughs != 2 || rp.FailedCount != 1 {
		t.Errorf("Breakthrough/Failed counts: %d/%d, want 2/1", rp.Breakthroughs, rp.FailedCount)
	}
}

func TestRealmConfig_Fields(t *testing.T) {
	cfg := RealmConfig{
		ID:              3,
		Name:            "金丹",
		LevelCount:      9,
		BaseExpPerLevel: 50000,
		ExpMultiplier:   1.2,
		BaseHealth:      5000,
		BaseMana:        3000,
		BaseAttack:      400,
		BaseDefense:     300,
		NextRealmID:     4,
		Icon:            "realm/golden_core.png",
		Description:     "金丹大道，脱胎换骨",
	}
	if cfg.ID != 3 || cfg.Name != "金丹" {
		t.Errorf("ID/Name mismatch: %d/%s", cfg.ID, cfg.Name)
	}
	if cfg.BaseHealth != 5000 || cfg.BaseMana != 3000 {
		t.Errorf("Stat bonuses mismatch: HP=%d MP=%d", cfg.BaseHealth, cfg.BaseMana)
	}
	if cfg.BaseAttack != 400 || cfg.BaseDefense != 300 {
		t.Errorf("Combat stats mismatch: atk=%d def=%d", cfg.BaseAttack, cfg.BaseDefense)
	}
	if cfg.NextRealmID != 4 {
		t.Errorf("NextRealmID = %d, want 4", cfg.NextRealmID)
	}
	if cfg.Icon != "realm/golden_core.png" {
		t.Errorf("Icon = %q, want 'realm/golden_core.png'", cfg.Icon)
	}
}
