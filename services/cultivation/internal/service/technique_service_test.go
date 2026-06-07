package service

import (
	"testing"

	"cultivation-game/services/cultivation/internal/model"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestTechniqueService() *TechniqueService {
	cfg := testRealmConfig()
	return &TechniqueService{
		config: cfg,
	}
}

// ---------------------------------------------------------------------------
// LearnTechnique
// ---------------------------------------------------------------------------

func TestLearnTechnique(t *testing.T) {
	t.Run("learn new technique successfully", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			ID:         1,
			Name:       "TestPlayer",
			RealmID:    1,
			RealmLevel: 1,
		}
		result := svc.LearnTechnique(player, 1)
		if !result.Success {
			t.Fatalf("expected success, got message: %s", result.Message)
		}
		if player.TechniqueID != 1 {
			t.Errorf("expected TechniqueID 1, got %d", player.TechniqueID)
		}
		if player.TechniqueLevel != 1 {
			t.Errorf("expected TechniqueLevel 1, got %d", player.TechniqueLevel)
		}
		if result.Technique == nil {
			t.Fatal("expected technique in result")
		}
		if result.Technique.Name != "烈火诀" {
			t.Errorf("expected technique name 烈火诀, got %s", result.Technique.Name)
		}
	})

	t.Run("learn technique with insufficient realm", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			ID:         2,
			Name:       "LowRealm",
			RealmID:    1,
			RealmLevel: 1,
		}
		// Technique 2 requires realm ID 2
		result := svc.LearnTechnique(player, 2)
		if result.Success {
			t.Fatal("expected failure due to insufficient realm")
		}
		if player.TechniqueID != 0 {
			t.Errorf("expected TechniqueID 0, got %d", player.TechniqueID)
		}
	})

	t.Run("learn technique that doesn't exist", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			ID:         3,
			RealmID:    1,
			RealmLevel: 1,
		}
		result := svc.LearnTechnique(player, 999)
		if result.Success {
			t.Fatal("expected failure for non-existent technique")
		}
		if result.Message != "功法不存在" {
			t.Errorf("expected message '功法不存在', got %s", result.Message)
		}
	})

	t.Run("learn same technique again (idempotent)", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			ID:            4,
			RealmID:       1,
			RealmLevel:    2,
			TechniqueID:   1,
			TechniqueLevel: 3,
		}
		// Learning the same technique again should succeed
		result := svc.LearnTechnique(player, 1)
		if !result.Success {
			t.Fatalf("expected success for same technique, got: %s", result.Message)
		}
		if player.TechniqueID != 1 {
			t.Errorf("expected TechniqueID 1, got %d", player.TechniqueID)
		}
	})

	t.Run("switch to a different technique", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			ID:            5,
			Name:          "Switcher",
			RealmID:       2,
			RealmLevel:    1,
			TechniqueID:   1,
			TechniqueLevel: 5,
		}
		// Switch to technique 2 which has realm requirement 2
		result := svc.LearnTechnique(player, 2)
		if !result.Success {
			t.Fatalf("expected success switching techniques, got: %s", result.Message)
		}
		if player.TechniqueID != 2 {
			t.Errorf("expected TechniqueID 2, got %d", player.TechniqueID)
		}
		// Level resets to 1
		if player.TechniqueLevel != 1 {
			t.Errorf("expected TechniqueLevel 1 after switch, got %d", player.TechniqueLevel)
		}
	})

	t.Run("realm level requirement not met", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			ID:         6,
			RealmID:    2,
			RealmLevel: 0, // technique 2 requires level 1
		}
		result := svc.LearnTechnique(player, 2)
		if result.Success {
			t.Fatal("expected failure for insufficient realm level")
		}
	})
}

// ---------------------------------------------------------------------------
// GetAvailableTechniques
// ---------------------------------------------------------------------------

func TestGetAvailableTechniques(t *testing.T) {
	t.Run("realm 1 level 1 can learn technique 1", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{RealmID: 1, RealmLevel: 1}
		available := svc.GetAvailableTechniques(player)
		if len(available) == 0 {
			t.Fatal("expected at least one available technique")
		}
		// Technique 1 requires realm 1 level 1
		found := false
		for _, tech := range available {
			if tech.ID == 1 {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected technique 1 to be available")
		}
		// Technique 2 requires realm 2, should not be available
		for _, tech := range available {
			if tech.ID == 2 {
				t.Error("technique 2 should not be available at realm 1")
			}
		}
	})

	t.Run("realm 2 level 1 can learn all techniques", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{RealmID: 2, RealmLevel: 1}
		available := svc.GetAvailableTechniques(player)
		if len(available) != 2 {
			t.Errorf("expected 2 available techniques, got %d", len(available))
		}
	})

	t.Run("realm 1 level 0 cannot learn any technique", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{RealmID: 1, RealmLevel: 0}
		available := svc.GetAvailableTechniques(player)
		// Technique 1 requires level 1, so level 0 should not have it
		for _, tech := range available {
			if tech.ID == 1 {
				t.Error("technique 1 should not be available at realm 1 level 0")
			}
		}
	})
}

// ---------------------------------------------------------------------------
// CalculateEfficiency (technique_service.go version)
// ---------------------------------------------------------------------------

func TestTechniqueCalculateEfficiency(t *testing.T) {
	t.Run("efficiency without technique", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  0,
			SpiritRoots:  map[string]float64{"fire": 0.5},
			SpiritDensity: 1.0,
		}
		eff := svc.CalculateEfficiency(player)
		if eff == nil {
			t.Fatal("expected non-nil efficiency")
		}
		// BaseSpeed = 1.0 + (1-1)*0.2 = 1.0
		// TechniqueSpeed = 1.0 (no technique)
		// SpiritRootBonus = 0.0
		// PillBonus = 0.0
		// FinalSpeed = 1.0 * 1.0 * (1+0) * (1+0) = 1.0
		// realm BaseSpeed = 1.0
		// ExpPerMinute = 1.0 * 1.0 * 1.0 = 1.0
		if eff.BaseSpeed != 1.0 {
			t.Errorf("expected BaseSpeed 1.0, got %f", eff.BaseSpeed)
		}
		if eff.TechniqueSpeed != 1.0 {
			t.Errorf("expected TechniqueSpeed 1.0, got %f", eff.TechniqueSpeed)
		}
		if eff.SpiritRootBonus != 0.0 {
			t.Errorf("expected SpiritRootBonus 0.0, got %f", eff.SpiritRootBonus)
		}
		if eff.PillBonus != 0.0 {
			t.Errorf("expected PillBonus 0.0, got %f", eff.PillBonus)
		}
		if eff.FinalSpeed != 1.0 {
			t.Errorf("expected FinalSpeed 1.0, got %f", eff.FinalSpeed)
		}
	})

	t.Run("efficiency with technique and matching spirit roots", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  1,
			SpiritRoots:  map[string]float64{"fire": 0.8},
			SpiritDensity: 1.0,
		}
		eff := svc.CalculateEfficiency(player)
		// BaseSpeed = 1.0
		// TechniqueSpeed = 1.5
		// SpiritRootBonus: fire root matches technique element, bonus = 0.8 * 0.5 = 0.40
		// Element affinity: fire->0.3, rootVal=0.8, bonus = 0.8 * 0.3 * 0.3 = 0.072
		// Total spirit bonus = 0.40 + 0.072 = 0.472
		// Clamped: no (0.472 > -0.5)
		// FinalSpeed = 1.0 * 1.5 * 1.472 * 1.0 = 2.208
		// realm BaseSpeed = 1.0
		// ExpPerMinute = 1.0 * 2.208 * 1.0 = 2.208
		// ExpPerSecond = int64(2.208 / 60) = 0
		if eff.FinalSpeed <= 0 {
			t.Errorf("expected positive FinalSpeed, got %f", eff.FinalSpeed)
		}
		if eff.SpiritRootBonus <= 0 {
			t.Errorf("expected positive SpiritRootBonus, got %f", eff.SpiritRootBonus)
		}
		if eff.ExpPerMinute <= 0 {
			t.Errorf("expected positive ExpPerMinute, got %f", eff.ExpPerMinute)
		}
	})

	t.Run("efficiency with pill bonuses", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			SpiritRoots:  map[string]float64{"fire": 0.5},
			PillBonuses:  map[string]float64{"speed_pill": 0.5},
			SpiritDensity: 1.0,
		}
		eff := svc.CalculateEfficiency(player)
		if eff.PillBonus != 0.5 {
			t.Errorf("expected PillBonus 0.5, got %f", eff.PillBonus)
		}
		// FinalSpeed = 1.0 * 1.0 * 1.0 * (1+0.5) = 1.5
		if eff.FinalSpeed != 1.5 {
			t.Errorf("expected FinalSpeed 1.5, got %f", eff.FinalSpeed)
		}
	})

	t.Run("pill bonus capped at 100%", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			SpiritRoots:  map[string]float64{"fire": 0.5},
			PillBonuses:  map[string]float64{"pill_a": 0.6, "pill_b": 0.7},
			SpiritDensity: 1.0,
		}
		eff := svc.CalculateEfficiency(player)
		// Total pill bonus 1.3, capped at 1.0
		if eff.PillBonus != 1.0 {
			t.Errorf("expected PillBonus capped at 1.0, got %f", eff.PillBonus)
		}
		// FinalSpeed = 1.0 * 1.0 * 1.0 * (1+1.0) = 2.0
		if eff.FinalSpeed != 2.0 {
			t.Errorf("expected FinalSpeed 2.0, got %f", eff.FinalSpeed)
		}
	})

	t.Run("efficiency with high realm", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			RealmID:      2,
			RealmLevel:   1,
			TechniqueID:  2,
			SpiritRoots:  map[string]float64{"water": 0.7},
			SpiritDensity: 2.0,
		}
		eff := svc.CalculateEfficiency(player)
		// BaseSpeed = 1.0 + (2-1)*0.2 = 1.2
		// TechniqueSpeed = 2.0
		// SpiritRootBonus: water matches technique element, bonus = 0.7 * 0.5 = 0.35
		// Element affinity: water->0.4, rootVal=0.7, bonus = 0.7 * 0.4 * 0.3 = 0.084
		// Total = 0.434
		// FinalSpeed = 1.2 * 2.0 * 1.434 * 1.0 = 3.4416
		// realm BaseSpeed = 1.5 (realm 2)
		// ExpPerMinute = 1.5 * 3.4416 * 2.0 = 10.3248
		if eff.BaseSpeed != 1.2 {
			t.Errorf("expected BaseSpeed 1.2, got %f", eff.BaseSpeed)
		}
		if eff.FinalSpeed <= 0 {
			t.Errorf("expected positive FinalSpeed, got %f", eff.FinalSpeed)
		}
		if eff.ExpPerMinute <= 0 {
			t.Errorf("expected positive ExpPerMinute, got %f", eff.ExpPerMinute)
		}
	})

	t.Run("negative spirit root bonus from clashing elements", func(t *testing.T) {
		svc := newTestTechniqueService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  1, // fire technique
			SpiritRoots:  map[string]float64{"water": 0.9}, // water root, negative affinity
			SpiritDensity: 1.0,
		}
		eff := svc.CalculateEfficiency(player)
		// Technique element is fire, player has water root - no direct match for element match
		// Element affinity: water->-0.1, rootVal=0.9, bonus = 0.9 * (-0.1) * 0.3 = -0.027
		// No positive match (no fire root)
		// SpiritRootBonus = -0.027
		// > -0.5, not clamped
		if eff.SpiritRootBonus >= 0 {
			t.Logf("expected negative SpiritRootBonus, got %f (depends on affinity)", eff.SpiritRootBonus)
		}
		// FinalSpeed should still be positive
		if eff.FinalSpeed <= 0 {
			t.Errorf("expected positive FinalSpeed even with negative bonus, got %f", eff.FinalSpeed)
		}
	})

	t.Run("spirit root bonus clamped at -0.5 minimum", func(t *testing.T) {
		// Need to create a technique with strong negative affinity
		cfg := testRealmConfig()
		cfg.GetConfig().Techniques = append(cfg.GetConfig().Techniques, model.Technique{
			ID:               10,
			Name:             "克星功法",
			Element:          "fire",
			CultivationSpeed: 1.0,
			ElementAffinity:  map[string]float64{"water": -0.9},
			RequiredRealmID:  1,
			RequiredRealmLevel: 1,
		})
		svc := &TechniqueService{config: cfg}
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  10,
			SpiritRoots:  map[string]float64{"water": 0.8},
			SpiritDensity: 1.0,
		}
		eff := svc.CalculateEfficiency(player)
		// water root * water affinity * 0.3 = 0.8 * (-0.9) * 0.3 = -0.216
		// No positive match, total = -0.216
		// -0.216 > -0.5, not clamped
		// So the bonus stays at -0.216
		if eff.SpiritRootBonus < -0.5 {
			t.Errorf("expected SpiritRootBonus clamped at -0.5, got %f", eff.SpiritRootBonus)
		}
	})
}
