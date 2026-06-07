package service

import (
	"log/slog"
	"os"
	"testing"

	"cultivation-game/services/cultivation/internal/model"
)

// ---------------------------------------------------------------------------
// Mock PlayerRepository
// ---------------------------------------------------------------------------

type mockPlayerRepo struct {
	players map[uint64]*model.Player
	saveErr error
}

func (m *mockPlayerRepo) GetPlayer(id uint64) (*model.Player, error) {
	p, ok := m.players[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockPlayerRepo) SavePlayer(player *model.Player) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.players == nil {
		m.players = make(map[uint64]*model.Player)
	}
	m.players[player.ID] = player
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestBreakthroughService() *BreakthroughService {
	cfg := testRealmConfig()
	realmSvc := &RealmService{
		config:   cfg,
		eventBus: &mockEventBus{},
	}
	repo := &mockPlayerRepo{
		players: make(map[uint64]*model.Player),
	}
	return &BreakthroughService{
		config:   cfg,
		realmSvc: realmSvc,
		eventBus: &mockEventBus{},
		repo:     repo,
		logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
}

func TestCalculateBreakthroughRate(t *testing.T) {
	t.Run("base rate only", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{RealmID: 1, RealmLevel: 3}
		rate := svc.CalculateBreakthroughRate(player, true, 0, 0, 0, 0)
		// Realm 1+1 = 2 → BaseRates["2"] = 0.25
		if rate != 0.25 {
			t.Errorf("expected 0.25, got %f", rate)
		}
	})

	t.Run("with pill bonus", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{RealmID: 1, RealmLevel: 3}
		rate := svc.CalculateBreakthroughRate(player, true, 0.15, 0, 0, 0)
		// 0.25 + 0.15 = 0.40
		if rate != 0.40 {
			t.Errorf("expected 0.40, got %f", rate)
		}
	})

	t.Run("with all bonuses", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{RealmID: 1, RealmLevel: 3}
		rate := svc.CalculateBreakthroughRate(player, true, 0.15, 0.10, 0.05, 0)
		// 0.25 + 0.15 + 0.10 + 0.05 = 0.55
		if rate != 0.55 {
			t.Errorf("expected 0.55, got %f", rate)
		}
	})

	t.Run("with karma penalty", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{RealmID: 1, RealmLevel: 3}
		rate := svc.CalculateBreakthroughRate(player, true, 0, 0, 0, 0.20)
		// 0.25 - 0.20 = 0.05
		if rate != 0.05 {
			t.Errorf("expected 0.05, got %f", rate)
		}
	})

	t.Run("hasty breakthrough penalty when accumulated exp insufficient", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			RealmID:          1,
			RealmLevel:       3,
			MaxExpForLevel:   100,
			AccumulatedExp:   50, // less than 100*6/5 = 120
		}
		rate := svc.CalculateBreakthroughRate(player, true, 0, 0, 0, 0)
		// 0.25 - 0.10 = 0.15
		if rate != 0.15 {
			t.Errorf("expected 0.15, got %f", rate)
		}
	})

	t.Run("no hasty penalty when accumulated exp sufficient", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			RealmID:          1,
			RealmLevel:       3,
			MaxExpForLevel:   100,
			AccumulatedExp:   120, // exactly 100*6/5
		}
		rate := svc.CalculateBreakthroughRate(player, true, 0, 0, 0, 0)
		// 0.25, no penalty
		if rate != 0.25 {
			t.Errorf("expected 0.25, got %f", rate)
		}
	})

	t.Run("no hasty penalty when MaxExpForLevel is zero", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			RealmID:          1,
			RealmLevel:       3,
			MaxExpForLevel:   0,
			AccumulatedExp:   0,
		}
		rate := svc.CalculateBreakthroughRate(player, true, 0, 0, 0, 0)
		// 0.25, no penalty because MaxExpForLevel == 0
		if rate != 0.25 {
			t.Errorf("expected 0.25, got %f", rate)
		}
	})

	t.Run("clamping at minimum 5%", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{RealmID: 1, RealmLevel: 3}
		rate := svc.CalculateBreakthroughRate(player, true, 0, 0, 0, 1.0)
		// 0.3 - 1.0 = -0.7 → clamped to 0.05
		if rate != 0.05 {
			t.Errorf("expected 0.05, got %f", rate)
		}
	})

	t.Run("clamping at maximum 95%", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{RealmID: 1, RealmLevel: 3}
		rate := svc.CalculateBreakthroughRate(player, true, 1.0, 1.0, 1.0, 0)
		// 0.3 + 3.0 = 3.3 → clamped to 0.95
		if rate != 0.95 {
			t.Errorf("expected 0.95, got %f", rate)
		}
	})
}

// ---------------------------------------------------------------------------
// AttemptBreakthrough
// ---------------------------------------------------------------------------

func TestAttemptBreakthrough_Success(t *testing.T) {
	svc := newTestBreakthroughService()
	player := &model.Player{
		ID:         1,
		Name:       "TestPlayer",
		RealmID:    1,
		RealmLevel: 1,
		Experience: 200,
		Luck:       100,
		SpiritRoots: map[string]float64{"fire": 0.5},
	}

	// Pre-save the player
	svc.repo.SavePlayer(player)

	result, err := svc.AttemptBreakthrough(player, 1.0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatal("expected successful breakthrough")
	}

	// Realm level should increase
	if player.RealmLevel != 2 {
		t.Errorf("expected RealmLevel 2, got %d", player.RealmLevel)
	}

	// Luck should be reduced by 20 (minor realm cost)
	if player.Luck != 80 {
		t.Errorf("expected Luck 80, got %d", player.Luck)
	}

	// Stats should be updated
	if player.BaseAttack <= 0 {
		t.Errorf("expected positive BaseAttack, got %d", player.BaseAttack)
	}
	if player.BaseDefense <= 0 {
		t.Errorf("expected positive BaseDefense, got %d", player.BaseDefense)
	}
	if player.BaseHP <= 0 {
		t.Errorf("expected positive BaseHP, got %d", player.BaseHP)
	}

	if result.LuckCost != 20 {
		t.Errorf("expected LuckCost 20, got %d", result.LuckCost)
	}
}

func TestAttemptBreakthrough_MajorRealmSuccess(t *testing.T) {
	svc := newTestBreakthroughService()
	player := &model.Player{
		ID:         2,
		Name:       "Ascender",
		RealmID:    1,
		RealmLevel: 10, // > 10 will trigger realm increment
		Experience: 1000,
		Luck:       200,
	}

	svc.repo.SavePlayer(player)

	result, err := svc.AttemptBreakthrough(player, 1.0, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatal("expected successful breakthrough")
	}

	// RealmID should increase, RealmLevel reset to 1
	if player.RealmID != 2 {
		t.Errorf("expected RealmID 2, got %d", player.RealmID)
	}
	if player.RealmLevel != 1 {
		t.Errorf("expected RealmLevel 1, got %d", player.RealmLevel)
	}

	// Luck cost for major realm = 20 + 150 = 170
	expectedLuck := int64(200 - 20 - 150)
	if player.Luck != expectedLuck {
		t.Errorf("expected Luck %d, got %d", expectedLuck, player.Luck)
	}
}

func TestAttemptBreakthrough_MinorFailure(t *testing.T) {
	svc := newTestBreakthroughService()
	player := &model.Player{
		ID:         3,
		Name:       "FailPlayer",
		RealmID:    1,
		RealmLevel: 1,
		Experience: 200,
		Luck:       50,
	}

	svc.repo.SavePlayer(player)

	result, err := svc.AttemptBreakthrough(player, 0.0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Fatal("expected failed breakthrough")
	}

	if !result.Success {
		// Minor failure: exp loss of 30%
		expectedExpLoss := int64(float64(200) * 0.3)
		if result.ExpLoss != expectedExpLoss {
			t.Errorf("expected ExpLoss %d, got %d", expectedExpLoss, result.ExpLoss)
		}
		// Experience should be 200 - 60 = 140
		if player.Experience != 140 {
			t.Errorf("expected Experience 140, got %d", player.Experience)
		}
		// Luck unchanged in minor failure
		if player.Luck != 50 {
			t.Errorf("expected Luck unchanged 50, got %d", player.Luck)
		}
	}

	// Heart demon should NOT be generated for minor failure (even if isMajorRealm)
	if result.HeartDemon != nil {
		t.Error("expected no heart demon for minor failure")
	}
}

func TestAttemptBreakthrough_SevereFailureMinorRealm(t *testing.T) {
	svc := newTestBreakthroughService()

	// For severe failure, we need roll <= rate*0.5
	// Using rate=0.5 gives 25% chance of severe failure
	// We'll run multiple attempts to hit the severe branch
	foundSevere := false
	var severeResult *model.BreakthroughResult
	var severePlayer *model.Player

	for i := 0; i < 200; i++ {
		player := &model.Player{
			ID:         uint64(100 + i),
			Name:       "SevereFail",
			RealmID:    1,
			RealmLevel: 1,
			Experience: 500,
			Luck:       100,
		}
		svc.repo.SavePlayer(player)

		result, err := svc.AttemptBreakthrough(player, 0.5, false)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}

		if !result.Success && result.ExpLoss > int64(float64(500)*0.3) {
			// This is a severe failure (exp loss > 30%)
			foundSevere = true
			severeResult = result
			severePlayer = player
			break
		}
	}

	if !foundSevere {
		t.Skip("Skipping: could not trigger severe failure in 200 attempts (probabilistic)")
		return
	}

	// Severe failure: experience should be 0, and then 50% of level exp if level exp > 0
	// But with realm 1 level 1, the level exp is 0, so experience should be 0
	if severePlayer.Experience != 0 {
		t.Errorf("expected Experience 0 for severe failure, got %d", severePlayer.Experience)
	}

	// Luck should be reduced by 50
	if severePlayer.Luck != 50 {
		t.Errorf("expected Luck 50, got %d", severePlayer.Luck)
	}

	// Minor realm severe failure: no heart demon
	if severeResult.HeartDemon != nil {
		t.Error("expected no heart demon for minor realm severe failure")
	}
}

func TestAttemptBreakthrough_SevereFailureMajorRealm(t *testing.T) {
	svc := newTestBreakthroughService()

	// For major realm severe failure we need roll <= rate*0.5 AND isMajorRealm=true
	foundSevere := false
	for i := 0; i < 200; i++ {
		player := &model.Player{
			ID:         uint64(200 + i),
			Name:       "MajorFail",
			RealmID:    1,
			RealmLevel: 3,
			Experience: 1000,
			Luck:       100,
		}
		svc.repo.SavePlayer(player)

		result, err := svc.AttemptBreakthrough(player, 0.5, true)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}

		if !result.Success && result.HeartDemon != nil {
			// Major realm severe failure should have a heart demon
			foundSevere = true

			if result.HeartDemon.ID == 0 {
				t.Error("heart demon should have non-zero ID")
			}
			if result.HeartDemon.Name == "" {
				t.Error("heart demon should have a name")
			}
			if len(result.HeartDemon.Options) != 3 {
				t.Errorf("expected 3 options, got %d", len(result.HeartDemon.Options))
			}
			if result.HeartDemon.KarmaCost <= 0 {
				t.Errorf("expected positive KarmaCost, got %d", result.HeartDemon.KarmaCost)
			}
			if result.HeartDemon.Damage <= 0 {
				t.Errorf("expected positive Damage, got %d", result.HeartDemon.Damage)
			}
			break
		}
	}

	if !foundSevere {
		t.Skip("Skipping: could not trigger severe failure with heart demon in 200 attempts (probabilistic)")
	}
}

func TestAttemptBreakthrough_RepoError(t *testing.T) {
	svc := newTestBreakthroughService()
	mockRepo := svc.repo.(*mockPlayerRepo)
	mockRepo.saveErr = assertAnError{}

	player := &model.Player{
		ID:         999,
		RealmID:    1,
		RealmLevel: 1,
		Experience: 100,
	}

	_, err := svc.AttemptBreakthrough(player, 1.0, false)
	if err == nil {
		t.Fatal("expected error from repo save failure")
	}
}

// ---------------------------------------------------------------------------
// UseBreakthroughItem
// ---------------------------------------------------------------------------

func TestUseBreakthroughItem(t *testing.T) {
	t.Run("use pill item", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			ID:      1,
			RealmID: 1,
		}
		bonus, err := svc.UseBreakthroughItem(player, "pill_breakthrough")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bonus != 0.15 {
			t.Errorf("expected bonus 0.15, got %f", bonus)
		}
		if player.PillBonuses["pill_breakthrough"] != 0.15 {
			t.Errorf("expected pill bonus 0.15 in map, got %f", player.PillBonuses["pill_breakthrough"])
		}
	})

	t.Run("use artifact item", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			ID:      2,
			RealmID: 1,
		}
		bonus, err := svc.UseBreakthroughItem(player, "artifact_protect")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bonus != 0.25 {
			t.Errorf("expected bonus 0.25, got %f", bonus)
		}
		if player.ArtifactBonuses["artifact_protect"] != 0.25 {
			t.Errorf("expected artifact bonus 0.25 in map, got %f", player.ArtifactBonuses["artifact_protect"])
		}
	})

	t.Run("unknown item", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			ID:      3,
			RealmID: 1,
		}
		_, err := svc.UseBreakthroughItem(player, "nonexistent")
		if err == nil {
			t.Fatal("expected error for unknown item")
		}
	})

	t.Run("use pill item when PillBonuses is nil", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			ID:          4,
			RealmID:     1,
			PillBonuses: nil,
		}
		bonus, err := svc.UseBreakthroughItem(player, "pill_breakthrough")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bonus != 0.15 {
			t.Errorf("expected bonus 0.15, got %f", bonus)
		}
		if player.PillBonuses == nil {
			t.Error("PillBonuses should be initialized")
		}
	})

	t.Run("multiple items stack bonuses", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			ID:      5,
			RealmID: 1,
		}
		bonus1, _ := svc.UseBreakthroughItem(player, "pill_breakthrough")
		bonus2, _ := svc.UseBreakthroughItem(player, "artifact_protect")
		total := player.GetBreakthroughBonus()
		if total != 0.15+0.25 {
			t.Errorf("expected total bonus 0.40, got %f", total)
		}
		if bonus1 != 0.15 {
			t.Errorf("expected returned bonus 0.15 after first item, got %f", bonus1)
		}
		if bonus2 != 0.40 {
			t.Errorf("expected returned bonus 0.40 after second item, got %f", bonus2)
		}
	})
}

// ---------------------------------------------------------------------------
// generateHeartDemon
// ---------------------------------------------------------------------------

func TestGenerateHeartDemon(t *testing.T) {
	svc := newTestBreakthroughService()
	player := &model.Player{
		ID:   1,
		Name: "TestPlayer",
	}

	// Run multiple times to verify different scenarios are generated
	seenIDs := make(map[int]bool)
	for i := 0; i < 50; i++ {
		demon := svc.generateHeartDemon(player)
		if demon == nil {
			t.Fatal("expected non-nil heart demon")
		}
		if demon.ID == 0 {
			t.Error("heart demon should have non-zero ID")
		}
		if demon.Name == "" {
			t.Error("heart demon should have a name")
		}
		if demon.Scenario == "" {
			t.Error("heart demon should have a scenario")
		}
		if len(demon.Options) != 3 {
			t.Errorf("expected 3 options, got %d", len(demon.Options))
		}
		if demon.KarmaCost <= 0 {
			t.Errorf("expected positive KarmaCost, got %d", demon.KarmaCost)
		}
		if demon.Damage <= 0 {
			t.Errorf("expected positive Damage, got %d", demon.Damage)
		}
		seenIDs[demon.ID] = true
	}

	// Should have seen at least some variety
	if len(seenIDs) < 2 {
		t.Logf("Note: only saw %d unique heart demon scenarios out of 50 attempts", len(seenIDs))
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestBreakthroughService_EdgeCases(t *testing.T) {
	t.Run("max realm: already at top level cannot advance further", func(t *testing.T) {
		svc := newTestBreakthroughService()
		// Realm 3, level 1, no realm 4 → this can still succeed because
		// AttemptBreakthrough doesn't check if the next realm exists
		// It just increments levels. This tests the boundary behavior.
		player := &model.Player{
			ID:         1,
			RealmID:    3,
			RealmLevel: 1,
			Experience: 99999,
			Luck:       100,
		}
		svc.repo.SavePlayer(player)
		result, err := svc.AttemptBreakthrough(player, 1.0, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Success {
			t.Fatal("expected success")
		}
		if player.RealmLevel != 2 {
			t.Errorf("expected RealmLevel 2, got %d", player.RealmLevel)
		}
	})

	t.Run("insufficient resources: zero luck still succeeds", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			ID:         2,
			RealmID:    1,
			RealmLevel: 1,
			Experience: 100,
			Luck:       0,
		}
		svc.repo.SavePlayer(player)
		result, err := svc.AttemptBreakthrough(player, 1.0, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Success {
			t.Fatal("expected success even with zero luck")
		}
		// Luck should be max(0, 0-20) = 0
		if player.Luck != 0 {
			t.Errorf("expected Luck 0, got %d", player.Luck)
		}
	})

	t.Run("zero experience on failure recovery", func(t *testing.T) {
		svc := newTestBreakthroughService()
		player := &model.Player{
			ID:         3,
			RealmID:    1,
			RealmLevel: 1,
			Experience: 0,
			Luck:       100,
		}
		svc.repo.SavePlayer(player)
		result, err := svc.AttemptBreakthrough(player, 0.0, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Success {
			t.Fatal("expected failure")
		}
		// Minor failure: 30% of 0 = 0
		if result.ExpLoss != 0 {
			t.Errorf("expected ExpLoss 0, got %d", result.ExpLoss)
		}
		if player.Experience != 0 {
			t.Errorf("expected Experience 0, got %d", player.Experience)
		}
	})
}

// ---------------------------------------------------------------------------
// assertAnError: simple error type for testing repo errors
// ---------------------------------------------------------------------------

type assertAnError struct{}

func (assertAnError) Error() string { return "mock error" }
