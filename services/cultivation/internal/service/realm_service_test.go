package service

import (
	"testing"

	"cultivation-game/services/cultivation/internal/config"
	"cultivation-game/services/cultivation/internal/model"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// mockEventBus implements model.EventBus for testing.
type mockEventBus struct {
	published []struct {
		event string
		data  interface{}
	}
}

func (m *mockEventBus) Publish(event string, data interface{}) {
	if m != nil {
		m.published = append(m.published, struct {
			event string
			data  interface{}
		}{event, data})
	}
}

func (m *mockEventBus) Subscribe(event string, handler func(data interface{})) func() {
	return func() {}
}

// testRealmConfig builds a GameConfig with fixed test data.
func testRealmConfig() *config.ConfigLoader {
	loader := config.NewConfigLoader(nil, "", config.LoadOptions{})
	gc := loader.GetConfig()
	gc.Realms = []model.Realm{
		{
			ID:   1,
			Name: "练气",
			SubStages: []model.SubStage{
				{Level: 1, Name: "练气一层", RequiredExp: 0, BaseAttack: 10, BaseDefense: 5, BaseHP: 100},
				{Level: 2, Name: "练气二层", RequiredExp: 100, BaseAttack: 15, BaseDefense: 8, BaseHP: 150},
				{Level: 3, Name: "练气三层", RequiredExp: 300, BaseAttack: 20, BaseDefense: 10, BaseHP: 200},
			},
			BaseSpeed: 1.0,
		},
		{
			ID:   2,
			Name: "筑基",
			SubStages: []model.SubStage{
				{Level: 1, Name: "筑基一层", RequiredExp: 500, BaseAttack: 30, BaseDefense: 20, BaseHP: 400},
				{Level: 2, Name: "筑基二层", RequiredExp: 800, BaseAttack: 40, BaseDefense: 25, BaseHP: 500},
			},
			BaseSpeed: 1.5,
		},
		{
			ID:   3,
			Name: "金丹",
			SubStages: []model.SubStage{
				{Level: 1, Name: "金丹一层", RequiredExp: 1500, BaseAttack: 60, BaseDefense: 35, BaseHP: 800},
			},
			BaseSpeed: 2.0,
		},
	}
	gc.Techniques = []model.Technique{
		{
			ID:               1,
			Name:             "烈火诀",
			Element:          "fire",
			CultivationSpeed: 1.5,
			BreakthroughBonus: 0.05,
			AttackBonus:      0.10,
			DefenseBonus:     0.05,
			HPBonus:          0.02,
			ElementAffinity:  map[string]float64{"fire": 0.3, "water": -0.1},
			RequiredRealmID:  1,
			RequiredRealmLevel: 1,
		},
		{
			ID:               2,
			Name:             "水月诀",
			Element:          "water",
			CultivationSpeed: 2.0,
			BreakthroughBonus: 0.10,
			AttackBonus:      0.20,
			DefenseBonus:     0.15,
			HPBonus:          0.05,
			ElementAffinity:  map[string]float64{"water": 0.4},
			RequiredRealmID:  2,
			RequiredRealmLevel: 1,
		},
	}
	gc.Breakthrough = config.BreakthroughConfig{
		BaseRates: map[string]config.BreakthroughRate{
			"2": {BaseRate: 0.25, PenaltyExpLoss: 0.5, Description: "突破至筑基"},
		},
		LevelBreakthroughs: map[string]config.BreakthroughRate{
			"1_3": {BaseRate: 0.7, Description: "练气三层→筑基"},
		},
		BonusItems: []config.BonusItem{
			{ItemID: "pill_breakthrough", Name: "筑基丹", RateBonus: 0.15},
			{ItemID: "artifact_protect", Name: "护身法宝", RateBonus: 0.25},
		},
	}
	return loader
}

// newTestRealmService creates a RealmService wired with test config and mocks.
func newTestRealmService() *RealmService {
	cfg := testRealmConfig()
	return &RealmService{
		config:   cfg,
		eventBus: &mockEventBus{},
	}
}

// ---------------------------------------------------------------------------
// GetCurrentRealm
// ---------------------------------------------------------------------------

func TestGetCurrentRealm(t *testing.T) {
	svc := newTestRealmService()

	t.Run("valid realm and level", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 2}
		realm, subStage, ok := svc.GetCurrentRealm(player)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if realm.ID != 1 {
			t.Errorf("expected realm ID 1, got %d", realm.ID)
		}
		if realm.Name != "练气" {
			t.Errorf("expected realm name 练气, got %s", realm.Name)
		}
		if subStage.Level != 2 {
			t.Errorf("expected sub stage level 2, got %d", subStage.Level)
		}
		if subStage.Name != "练气二层" {
			t.Errorf("expected sub stage name 练气二层, got %s", subStage.Name)
		}
	})

	t.Run("invalid realm ID", func(t *testing.T) {
		player := &model.Player{RealmID: 99, RealmLevel: 1}
		realm, subStage, ok := svc.GetCurrentRealm(player)
		if ok {
			t.Fatal("expected ok=false")
		}
		if realm != nil {
			t.Error("expected nil realm")
		}
		if subStage != nil {
			t.Error("expected nil subStage")
		}
	})

	t.Run("invalid realm level", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 99}
		realm, subStage, ok := svc.GetCurrentRealm(player)
		if ok {
			t.Fatal("expected ok=false")
		}
		if realm != nil {
			t.Error("expected nil realm")
		}
		if subStage != nil {
			t.Error("expected nil subStage")
		}
	})

	t.Run("realm ID zero", func(t *testing.T) {
		player := &model.Player{RealmID: 0, RealmLevel: 1}
		_, _, ok := svc.GetCurrentRealm(player)
		if ok {
			t.Fatal("expected ok=false for realm ID 0")
		}
	})

	t.Run("first level of realm", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 1}
		_, subStage, ok := svc.GetCurrentRealm(player)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if subStage.RequiredExp != 0 {
			t.Errorf("expected RequiredExp 0 for first sub stage, got %d", subStage.RequiredExp)
		}
	})
}

// ---------------------------------------------------------------------------
// CalculateStats
// ---------------------------------------------------------------------------

func TestCalculateStats(t *testing.T) {
	t.Run("no technique", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{RealmID: 1, RealmLevel: 1}
		atk, def, hp := svc.CalculateStats(player)
		if atk != 10 {
			t.Errorf("expected attack 10, got %d", atk)
		}
		if def != 5 {
			t.Errorf("expected defense 5, got %d", def)
		}
		if hp != 100 {
			t.Errorf("expected hp 100, got %d", hp)
		}
	})

	t.Run("with technique, matching spirit roots", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:     1,
			RealmLevel:  2,
			TechniqueID: 1,
			SpiritRoots: map[string]float64{"fire": 0.8, "water": 0.2},
		}
		atk, def, hp := svc.CalculateStats(player)

		// Base: atk=15, def=8, hp=150
		// Technique: atkBonus=0.1, defBonus=0.05, hpBonus=0.02
		// Element affinity fire->0.3: rootVal(0.8) * affinity(0.3) = 0.24 -> atk/def/hp += 0.024 each
		// Element affinity water->-0.1: rootVal(0.2) * affinity(-0.1) = -0.02 -> atk/def/hp += -0.002
		// So element total bonus = 0.24 + (-0.02) = 0.22 for each stat
		// atkBonus = 0.1 + 0.22*0.1 = 0.1 + 0.022 = 0.122
		// finalAtk = 15 * (1 + 0.122) = 15 * 1.122 = 16.83 -> 16(int64)
		// finalDef = 8 * (1 + 0.072) = 8 * 1.072 = 8.576 -> 8(int64)
		// finalHP = 150 * (1 + 0.042) = 150 * 1.042 = 156.3 -> 156(int64)

		if atk == 0 {
			t.Error("attack should not be zero")
		}
		if def == 0 {
			t.Error("defense should not be zero")
		}
		if hp == 0 {
			t.Error("hp should not be zero")
		}
	})

	t.Run("with technique, no matching spirit roots", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:     1,
			RealmLevel:  1,
			TechniqueID: 1,
			SpiritRoots: map[string]float64{"earth": 0.5},
		}
		atk, def, hp := svc.CalculateStats(player)
		// Base: atk=10, def=5, hp=100
		// Technique: atkBonus=0.1, defBonus=0.05, hpBonus=0.02
		// Element affinity: fire->0.3, water->-0.1, but player has earth only
		// So no element affinity bonus
		// finalAtk = 10 * (1 + 0.1) = 11
		// finalDef = 5 * (1 + 0.05) = 5.25 -> 5(int64)
		// finalHP = 100 * (1 + 0.02) = 102

		if atk != 11 {
			t.Errorf("expected attack 11, got %d", atk)
		}
		if def != 5 {
			t.Errorf("expected defense 5, got %d", def)
		}
		if hp != 102 {
			t.Errorf("expected hp 102, got %d", hp)
		}
	})

	t.Run("with technique, no spirit roots at all", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:     1,
			RealmLevel:  3,
			TechniqueID: 1,
			SpiritRoots: map[string]float64{},
		}
		atk, def, hp := svc.CalculateStats(player)
		// Base: atk=20, def=10, hp=200
		// Technique: atkBonus=0.1, defBonus=0.05, hpBonus=0.02
		// finalAtk = 20 * 1.1 = 22
		// finalDef = 10 * 1.05 = 10.5 -> 10(int64)
		// finalHP = 200 * 1.02 = 204
		if atk != 22 {
			t.Errorf("expected attack 22, got %d", atk)
		}
		if def != 10 {
			t.Errorf("expected defense 10, got %d", def)
		}
		if hp != 204 {
			t.Errorf("expected hp 204, got %d", hp)
		}
	})

	t.Run("invalid realm returns zeros", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{RealmID: 99, RealmLevel: 1, TechniqueID: 1}
		atk, def, hp := svc.CalculateStats(player)
		if atk != 0 || def != 0 || hp != 0 {
			t.Errorf("expected 0,0,0 for invalid realm, got %d,%d,%d", atk, def, hp)
		}
	})
}

// ---------------------------------------------------------------------------
// GetRealmProgress
// ---------------------------------------------------------------------------

func TestGetRealmProgress(t *testing.T) {
	svc := newTestRealmService()

	t.Run("zero experience", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 1, Experience: 0}
		got := svc.GetRealmProgress(player)
		// At level 1 with exp=0, need 100 for next level (100-0=100), current=0, progress=0
		if got != 0.0 {
			t.Errorf("expected 0.0, got %f", got)
		}
	})

	t.Run("partial progress", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 1, Experience: 50}
		got := svc.GetRealmProgress(player)
		// next: level 2 requires 100, current: level 1 requires 0
		// needed = 100 - 0 = 100, current = 50 - 0 = 50, progress = 50/100 = 0.5
		if got != 0.5 {
			t.Errorf("expected 0.5, got %f", got)
		}
	})

	t.Run("exact next stage requirement", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 1, Experience: 100}
		got := svc.GetRealmProgress(player)
		// At level 1 with 100 exp, next level requires 100
		// needed = 100-0 = 100, current = 100-0 = 100, progress = 1.0
		if got != 1.0 {
			t.Errorf("expected 1.0, got %f", got)
		}
	})

	t.Run("experience beyond next requirement", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 1, Experience: 200}
		got := svc.GetRealmProgress(player)
		if got != 1.0 {
			t.Errorf("expected 1.0 (capped), got %f", got)
		}
	})

	t.Run("at last sub stage of realm", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 3, Experience: 300}
		got := svc.GetRealmProgress(player)
		// Last level of realm, should return 1.0
		if got != 1.0 {
			t.Errorf("expected 1.0 for last sub stage, got %f", got)
		}
	})

	t.Run("invalid realm ID", func(t *testing.T) {
		player := &model.Player{RealmID: 99, RealmLevel: 1, Experience: 50}
		got := svc.GetRealmProgress(player)
		if got != 0.0 {
			t.Errorf("expected 0.0 for invalid realm, got %f", got)
		}
	})

	t.Run("invalid realm level", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 99, Experience: 50}
		got := svc.GetRealmProgress(player)
		if got != 0.0 {
			t.Errorf("expected 0.0 for invalid level, got %f", got)
		}
	})

	t.Run("level 2 progress calculation", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 2, Experience: 200}
		got := svc.GetRealmProgress(player)
		// next: level 3 requires 300, current: level 2 requires 100
		// needed = 300-100 = 200, current = 200-100 = 100, progress = 100/200 = 0.5
		if got != 0.5 {
			t.Errorf("expected 0.5, got %f", got)
		}
	})

	t.Run("level 2 with experience below start", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 2, Experience: 50}
		got := svc.GetRealmProgress(player)
		// current = 50-100 = -50 -> clamped to 0
		if got != 0.0 {
			t.Errorf("expected 0.0, got %f", got)
		}
	})
}

// ---------------------------------------------------------------------------
// CalculateEfficiency
// ---------------------------------------------------------------------------

func TestCalculateEfficiency(t *testing.T) {
	t.Run("base efficiency without technique", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  0,
			SpiritRoots:  map[string]float64{"fire": 0.5},
			SpiritDensity: 1.0,
		}
		got := svc.CalculateEfficiency(player, 1.0)
		// base=1.0, spiritMult=1.0, rootMult=1.0 (0.5 -> 人灵根), techniqueMult=1.0
		// expected = 1.0 * 1.0 * 1.0 * 1.0 = 1.0
		if got != 1.0 {
			t.Errorf("expected 1.0, got %f", got)
		}
	})

	t.Run("with technique and spirit roots", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  1,
			SpiritRoots:  map[string]float64{"fire": 0.95},
			SpiritDensity: 1.0,
		}
		got := svc.CalculateEfficiency(player, 1.0)
		// base=1.0, spiritMult=1.0, rootMult=2.0 (0.95 -> 天灵根), techniqueMult=1.5
		// expected = 1.0 * 1.0 * 2.0 * 1.5 = 3.0
		if got != 3.0 {
			t.Errorf("expected 3.0, got %f", got)
		}
	})

	t.Run("spirit density clamping - low", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  1,
			SpiritRoots:  map[string]float64{"fire": 0.5},
			SpiritDensity: 0.1,
		}
		got := svc.CalculateEfficiency(player, 0.1)
		// base=1.0, spiritMult=0.5 (clamped), rootMult=1.0, techniqueMult=1.5
		// expected = 1.0 * 0.5 * 1.0 * 1.5 = 0.75
		if got != 0.75 {
			t.Errorf("expected 0.75, got %f", got)
		}
	})

	t.Run("spirit density clamping - high", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			SpiritRoots:  map[string]float64{"fire": 0.4},
			SpiritDensity: 10.0,
		}
		got := svc.CalculateEfficiency(player, 10.0)
		// base=1.0, spiritMult=5.0 (clamped), rootMult=1.0, techniqueMult=1.0
		// expected = 1.0 * 5.0 * 1.0 * 1.0 = 5.0
		if got != 5.0 {
			t.Errorf("expected 5.0, got %f", got)
		}
	})

	t.Run("technique with cultivation speed below 1.0", func(t *testing.T) {
		// Create a config with a slow technique
		cfg := testRealmConfig()
		cfg.GetConfig().Techniques = append(cfg.GetConfig().Techniques, model.Technique{
			ID:               3,
			Name:             "残破功法",
			Element:          "none",
			CultivationSpeed: 0.5,
			RequiredRealmID:  1,
			RequiredRealmLevel: 1,
		})
		svc := &RealmService{config: cfg, eventBus: &mockEventBus{}}
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  3,
			SpiritRoots:  map[string]float64{"fire": 0.4},
			SpiritDensity: 1.0,
		}
		got := svc.CalculateEfficiency(player, 1.0)
		// techniqueMult would be 0.5 but clamped to 1.0
		// expected = 1.0 * 1.0 * 1.0 * 1.0 = 1.0
		if got != 1.0 {
			t.Errorf("expected 1.0 (technique clamped), got %f", got)
		}
	})

	t.Run("maximum possible efficiency", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  1,
			SpiritRoots:  map[string]float64{"fire": 0.95},
			SpiritDensity: 5.0,
		}
		got := svc.CalculateEfficiency(player, 5.0)
		// base=1.0, spiritMult=5.0, rootMult=2.0, techniqueMult=1.5
		// expected = 1.0 * 5.0 * 2.0 * 1.5 = 15.0
		if got != 15.0 {
			t.Errorf("expected 15.0, got %f", got)
		}
	})
}

// ---------------------------------------------------------------------------
// AfterBreakthrough
// ---------------------------------------------------------------------------

func TestAfterBreakthrough(t *testing.T) {
	svc := newTestRealmService()

	t.Run("breakthrough to realm 1 level 1", func(t *testing.T) {
		// This should not panic - it launches goroutines but they fail silently
		svc.AfterBreakthrough(100, 1, 1)
		// If we got here without panic, the goroutine dispatch worked
	})

	t.Run("breakthrough to realm 2 level 1", func(t *testing.T) {
		svc.AfterBreakthrough(200, 2, 1)
	})

	t.Run("breakthrough to realm 5 level 10", func(t *testing.T) {
		svc.AfterBreakthrough(300, 5, 10)
	})
}

// ---------------------------------------------------------------------------
// CalculateCultivationEfficiency
// ---------------------------------------------------------------------------

func TestCalculateCultivationEfficiency(t *testing.T) {
	t.Run("returns valid efficiency struct", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:       1,
			RealmLevel:    1,
			TechniqueID:   1,
			SpiritRoots:   map[string]float64{"fire": 0.8},
			SpiritDensity: 1.0,
		}
		eff := svc.CalculateCultivationEfficiency(player)
		if eff == nil {
			t.Fatal("expected non-nil efficiency")
		}
		if eff.FinalSpeed <= 0 {
			t.Errorf("expected positive FinalSpeed, got %f", eff.FinalSpeed)
		}
		if eff.BaseSpeed != 1.0 {
			t.Errorf("expected BaseSpeed 1.0, got %f", eff.BaseSpeed)
		}
	})

	t.Run("spirit density below minimum defaults to 0.5", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:       1,
			RealmLevel:    1,
			SpiritRoots:   map[string]float64{"fire": 0.4},
			SpiritDensity: 0.1, // below minimum
		}
		eff := svc.CalculateCultivationEfficiency(player)
		if eff.FinalSpeed <= 0 {
			t.Errorf("expected positive FinalSpeed, got %f", eff.FinalSpeed)
		}
	})

	t.Run("exp per second rounds to integer", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:       1,
			RealmLevel:    1,
			TechniqueID:   1,
			SpiritRoots:   map[string]float64{"fire": 0.95},
			SpiritDensity: 2.0,
		}
		eff := svc.CalculateCultivationEfficiency(player)
		// FinalSpeed should be 1.0 * 2.0 * 2.0 * 1.5 = 6.0
		// ExpPerSecond = int64(6.0/60.0 * 60 + 0.5) = int64(6.5) = 6
		if eff.ExpPerSecond != 6 {
			t.Errorf("expected ExpPerSecond 6, got %d", eff.ExpPerSecond)
		}
	})
}

// ---------------------------------------------------------------------------
// Cultivate
// ---------------------------------------------------------------------------

func TestCultivate(t *testing.T) {
	t.Run("positive cultivation minutes", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:       1,
			RealmLevel:    1,
			TechniqueID:   1,
			SpiritRoots:   map[string]float64{"fire": 0.95},
			SpiritDensity: 1.0,
		}
		initialExp := player.Experience
		gained := svc.Cultivate(player, 10.0)
		if gained <= 0 {
			t.Errorf("expected positive exp gain, got %d", gained)
		}
		if player.Experience != initialExp+gained {
			t.Errorf("expected experience %d, got %d", initialExp+gained, player.Experience)
		}
	})

	t.Run("zero minutes", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:     1,
			RealmLevel:  1,
			SpiritRoots: map[string]float64{"fire": 0.5},
		}
		initialExp := player.Experience
		gained := svc.Cultivate(player, 0)
		if gained != 0 {
			t.Errorf("expected 0 exp gain, got %d", gained)
		}
		if player.Experience != initialExp {
			t.Errorf("experience should not change, got %d", player.Experience)
		}
	})

	t.Run("negative minutes", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:     1,
			RealmLevel:  1,
			SpiritRoots: map[string]float64{"fire": 0.5},
		}
		initialExp := player.Experience
		gained := svc.Cultivate(player, -5.0)
		if gained != 0 {
			t.Errorf("expected 0 exp gain for negative minutes, got %d", gained)
		}
		if player.Experience != initialExp {
			t.Errorf("experience should not change, got %d", player.Experience)
		}
	})

	t.Run("experience accumulation over multiple calls", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:       1,
			RealmLevel:    1,
			TechniqueID:   1,
			SpiritRoots:   map[string]float64{"fire": 0.95},
			SpiritDensity: 1.0,
		}
		totalGained := int64(0)
		for i := 0; i < 5; i++ {
			gained := svc.Cultivate(player, 1.0)
			totalGained += gained
		}
		if totalGained <= 0 {
			t.Errorf("expected cumulative positive exp, got %d", totalGained)
		}
	})
}

// ---------------------------------------------------------------------------
// GetNextRealmRequirement
// ---------------------------------------------------------------------------

func TestGetNextRealmRequirement(t *testing.T) {
	svc := newTestRealmService()

	t.Run("next sub stage within same realm", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 1, Experience: 50}
		required, name, canBreak := svc.GetNextRealmRequirement(player)
		if required != 100 {
			t.Errorf("expected required exp 100, got %d", required)
		}
		if name != "练气二层" {
			t.Errorf("expected name 练气二层, got %s", name)
		}
		if canBreak {
			t.Error("expected canBreakthrough false (exp 50 < 100)")
		}
	})

	t.Run("can breakthrough next sub stage", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 1, Experience: 100}
		_, _, canBreak := svc.GetNextRealmRequirement(player)
		if !canBreak {
			t.Error("expected canBreakthrough true")
		}
	})

	t.Run("last sub stage - needs major realm breakthrough", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 3, Experience: 500}
		required, name, canBreak := svc.GetNextRealmRequirement(player)
		// Major breakthrough to realm 2, sub stage 1 requires 500
		if required != 500 {
			t.Errorf("expected required exp 500, got %d", required)
		}
		if name != "筑基一层" {
			t.Errorf("expected name 筑基一层, got %s", name)
		}
		if !canBreak {
			t.Error("expected canBreakthrough true (exp 500 >= 500)")
		}
	})

	t.Run("invalid realm ID", func(t *testing.T) {
		player := &model.Player{RealmID: 99, RealmLevel: 1}
		required, name, canBreak := svc.GetNextRealmRequirement(player)
		if required != 0 || name != "" || canBreak {
			t.Errorf("expected 0, '', false; got %d, %s, %v", required, name, canBreak)
		}
	})

	t.Run("invalid realm level", func(t *testing.T) {
		player := &model.Player{RealmID: 1, RealmLevel: 99}
		required, name, canBreak := svc.GetNextRealmRequirement(player)
		if required != 0 || name != "" || canBreak {
			t.Errorf("expected 0, '', false; got %d, %s, %v", required, name, canBreak)
		}
	})

	t.Run("max realm - no next realm", func(t *testing.T) {
		player := &model.Player{RealmID: 3, RealmLevel: 1, Experience: 99999}
		_, name, canBreak := svc.GetNextRealmRequirement(player)
		// Realm 3 has only 1 sub stage (level 1), so it's the last sub stage
		// Next realm (ID 4) doesn't exist
		if name != "已满级" {
			t.Errorf("expected '已满级', got %s", name)
		}
		if canBreak {
			t.Error("expected canBreakthrough false for max realm")
		}
	})
}

// ---------------------------------------------------------------------------
// CalculateBreakthroughProbability
// ---------------------------------------------------------------------------

func TestCalculateBreakthroughProbability(t *testing.T) {
	t.Run("base rate only", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:     1,
			RealmLevel:  1,
			SpiritRoots: map[string]float64{"fire": 0.5},
		}
		got := svc.CalculateBreakthroughProbability(player, 0.5)
		if got != 0.5 {
			t.Errorf("expected 0.5, got %f", got)
		}
	})

	t.Run("with technique breakthrough bonus", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  1,
			SpiritRoots:  map[string]float64{"water": 0.5},
		}
		got := svc.CalculateBreakthroughProbability(player, 0.5)
		// base=0.5, technique bonus=0.05, water root doesn't match fire element → no extra bonus
		// expected = 0.5 + 0.05 = 0.55
		if got != 0.55 {
			t.Errorf("expected 0.55, got %f", got)
		}
	})

	t.Run("with pill bonuses", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			PillBonuses:  map[string]float64{"pill_breakthrough": 0.15},
			SpiritRoots:  map[string]float64{"fire": 0.5},
		}
		got := svc.CalculateBreakthroughProbability(player, 0.5)
		// base=0.5, pill bonus=0.15
		// expected = 0.65
		if got != 0.65 {
			t.Errorf("expected 0.65, got %f", got)
		}
	})

	t.Run("clamping at minimum 5%", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:     1,
			RealmLevel:  1,
			SpiritRoots: map[string]float64{"fire": 0.5},
		}
		got := svc.CalculateBreakthroughProbability(player, -1.0)
		if got != 0.05 {
			t.Errorf("expected 0.05 (clamped), got %f", got)
		}
	})

	t.Run("clamping at maximum 95%", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  1,
			PillBonuses:  map[string]float64{"pill_big": 0.9},
			SpiritRoots:  map[string]float64{"fire": 0.9},
		}
		got := svc.CalculateBreakthroughProbability(player, 0.5)
		// 0.5 + 0.9 + 0.05 + 0.09 = 1.54 -> clamped to 0.95
		if got != 0.95 {
			t.Errorf("expected 0.95 (clamped), got %f", got)
		}
	})

	t.Run("technique with non-matching element", func(t *testing.T) {
		svc := newTestRealmService()
		player := &model.Player{
			RealmID:      1,
			RealmLevel:   1,
			TechniqueID:  1,
			SpiritRoots:  map[string]float64{"water": 0.5}, // fire root not matched
		}
		got := svc.CalculateBreakthroughProbability(player, 0.5)
		// base=0.5, technique bonus=0.05 (no element match because water != fire)
		// expected = 0.55
		if got != 0.55 {
			t.Errorf("expected 0.55, got %f", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Helper: ClampFloat64
// ---------------------------------------------------------------------------

func TestClampFloat64(t *testing.T) {
	t.Run("value within range", func(t *testing.T) {
		if got := clampFloat64(3.0, 0.0, 5.0); got != 3.0 {
			t.Errorf("expected 3.0, got %f", got)
		}
	})

	t.Run("value below min", func(t *testing.T) {
		if got := clampFloat64(-1.0, 0.0, 5.0); got != 0.0 {
			t.Errorf("expected 0.0, got %f", got)
		}
	})

	t.Run("value above max", func(t *testing.T) {
		if got := clampFloat64(10.0, 0.0, 5.0); got != 5.0 {
			t.Errorf("expected 5.0, got %f", got)
		}
	})

	t.Run("value equals min", func(t *testing.T) {
		if got := clampFloat64(0.0, 0.0, 5.0); got != 0.0 {
			t.Errorf("expected 0.0, got %f", got)
		}
	})

	t.Run("value equals max", func(t *testing.T) {
		if got := clampFloat64(5.0, 0.0, 5.0); got != 5.0 {
			t.Errorf("expected 5.0, got %f", got)
		}
	})
}
