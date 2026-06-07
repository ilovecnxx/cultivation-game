package service

import (
	"sync"
	"testing"
	"time"

	"cultivation-game/services/world/internal/model"
)

// mockStateStore implements playerStateStore for testing.
type mockStateStore struct {
	mu     sync.Mutex
	states map[string]*PlayerState
}

func newMockStateStore() *mockStateStore {
	return &mockStateStore{
		states: make(map[string]*PlayerState),
	}
}

func (m *mockStateStore) Load(userID string) (*PlayerState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[userID]
	if !ok {
		return nil, nil
	}
	return state, nil
}

func (m *mockStateStore) Save(userID string, state *PlayerState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[userID] = state
	return nil
}

func (m *mockStateStore) Delete(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.states, userID)
	return nil
}

func (m *mockStateStore) Ping() error { return nil }

// newTestExploreService creates an ExploreService with test data and a mock state store.
func newTestExploreService() *ExploreService {
	regions := map[string]*model.MapRegion{
		"newbie_village_01": {
			ID:          "newbie_village_01",
			Name:        "青竹村",
			Type:        model.RegionNewbie,
			Description: "新手村",
			LevelMin:    1,
			LevelMax:    10,
			DangerLevel: 1,
			Connections: []string{"qingyun_range_01", "secret_forest_01"},
		},
		"qingyun_range_01": {
			ID:          "qingyun_range_01",
			Name:        "青云山脉",
			Type:        model.RegionNewbie,
			Description: "山脉",
			LevelMin:    5,
			LevelMax:    20,
			DangerLevel: 3,
			Connections: []string{"newbie_village_01", "star_city_01"},
		},
		"star_city_01": {
			ID:          "star_city_01",
			Name:        "星辉城",
			Type:        model.RegionTown,
			Description: "城池",
			LevelMin:    10,
			LevelMax:    40,
			DangerLevel: 3,
			Connections: []string{"qingyun_range_01", "ancient_ruins_01"},
		},
		"ancient_ruins_01": {
			ID:          "ancient_ruins_01",
			Name:        "上古遗迹",
			Type:        model.RegionSecretRealm,
			Description: "秘境",
			LevelMin:    20,
			LevelMax:    60,
			DangerLevel: 6,
			Connections: []string{"star_city_01"},
		},
		"secret_forest_01": {
			ID:          "secret_forest_01",
			Name:        "秘境森林",
			Type:        model.RegionNewbie,
			LevelMin:    10,
			LevelMax:    30,
			DangerLevel: 4,
			Connections: []string{"newbie_village_01"},
		},
	}

	npcs := []*model.NPC{
		{
			ID:        "npc_elder",
			Name:      "长老",
			Type:      model.NPCQuestGiver,
			Title:     "村长",
			RegionID:  "newbie_village_01",
			Dialogues: []string{"你好", "加油"},
		},
		{
			ID:        "npc_merchant",
			Name:      "商人",
			Type:      model.NPCShop,
			Title:     "掌柜",
			RegionID:  "star_city_01",
			Dialogues: []string{"欢迎光临", "好买卖"},
		},
	}

	spots := []*model.GatheringSpot{
		{
			ID:         "spot_herb_01",
			RegionID:   "newbie_village_01",
			Name:       "药草丛",
			Type:       "herb",
			ItemID:     "herb_yinchen_01",
			ItemName:   "茵陈草",
			MinAmount:  2,
			MaxAmount:  5,
			Difficulty: 1,
			LevelReq:   1,
			RespawnSec: 300,
		},
		{
			ID:         "spot_ore_01",
			RegionID:   "qingyun_range_01",
			Name:       "铜矿脉",
			Type:       "ore",
			ItemID:     "ore_copper_01",
			ItemName:   "铜矿石",
			MinAmount:  1,
			MaxAmount:  3,
			Difficulty: 2,
			LevelReq:   3,
			RespawnSec: 600,
		},
	}

	return &ExploreService{
		regions:      regions,
		playerStates: make(map[string]*PlayerState),
		stateStore:   newMockStateStore(),
		npcList:      npcs,
		npcMap:       makeNPCIndex(npcs),
		spotList:     spots,
		spotMap:      makeSpotIndex(spots),
	}
}

// initPlayerForMove initializes a player state and sets LastMoveAt far enough in the past
// to bypass the MoveTo 2-second cooldown check in the production code.
func initPlayerForMove(svc *ExploreService, playerID string) *PlayerState {
	state := svc.GetPlayerExploreInfo(playerID)
	state.LastMoveAt = time.Now().Add(-3 * time.Second)
	state.ActionPoints = ActionPointMax
	return state
}

// TestMoveToRegion tests the MoveTo method for valid and invalid moves.
func TestMoveToRegion(t *testing.T) {
	svc := newTestExploreService()

	t.Run("valid move to adjacent region", func(t *testing.T) {
		initPlayerForMove(svc, "player1")
		result, err := svc.MoveTo("player1", "qingyun_range_01")
		if err != nil {
			t.Fatalf("MoveTo() unexpected error: %v", err)
		}
		if !result.Success {
			t.Error("result.Success should be true")
		}
		if result.CurrentRegion == nil {
			t.Fatal("result.CurrentRegion should not be nil")
		}
		if result.CurrentRegion.ID != "qingyun_range_01" {
			t.Errorf("current region = %q, want qingyun_range_01", result.CurrentRegion.ID)
		}
		// Verify state was updated
		state := svc.GetPlayerExploreInfo("player1")
		if state.RegionID != "qingyun_range_01" {
			t.Errorf("state region = %q, want qingyun_range_01", state.RegionID)
		}
	})

	t.Run("move to non-adjacent region fails", func(t *testing.T) {
		// player2 starts in newbie_village_01; star_city_01 is not directly adjacent
		_, err := svc.MoveTo("player2", "star_city_01")
		if err == nil {
			t.Error("expected error for non-adjacent move")
		}
	})

	t.Run("move to non-existent region fails", func(t *testing.T) {
		_, err := svc.MoveTo("player3", "nonexistent_region")
		if err == nil {
			t.Error("expected error for non-existent region")
		}
	})

	t.Run("multiple sequential valid moves", func(t *testing.T) {
		initPlayerForMove(svc, "player4")

		// First move: newbie -> qingyun
		result, err := svc.MoveTo("player4", "qingyun_range_01")
		if err != nil {
			t.Fatalf("first move: %v", err)
		}
		if !result.Success {
			t.Error("first move should succeed")
		}

		// Second move: qingyun -> star_city is connected
		// Need to bypass cooldown again
		state := svc.GetPlayerExploreInfo("player4")
		state.LastMoveAt = time.Now().Add(-3 * time.Second)

		result, err = svc.MoveTo("player4", "star_city_01")
		if err != nil {
			t.Fatalf("second move: %v", err)
		}
		if !result.Success {
			t.Error("second move should succeed")
		}
		if result.CurrentRegion.ID != "star_city_01" {
			t.Errorf("current region = %q, want star_city_01", result.CurrentRegion.ID)
		}
	})
}

// TestMoveTo_InsufficientAP tests the AP check in MoveTo.
func TestMoveTo_InsufficientAP(t *testing.T) {
	svc := newTestExploreService()

	state := initPlayerForMove(svc, "low_ap_player")
	state.ActionPoints = 3 // Less than ActionPointCost (10)

	_, err := svc.MoveTo("low_ap_player", "qingyun_range_01")
	if err == nil {
		t.Error("expected insufficient AP error")
	}
}

// TestMoveTo_Cooldown tests the move cooldown check.
func TestMoveTo_Cooldown(t *testing.T) {
	svc := newTestExploreService()

	state := initPlayerForMove(svc, "cooldown_player")

	// First move should succeed
	_, err := svc.MoveTo("cooldown_player", "qingyun_range_01")
	if err != nil {
		t.Fatalf("first move: %v", err)
	}

	// After the move, LastMoveAt was set to time.Now() by MoveTo.
	// Immediate second move should fail due to cooldown.
	_, err = svc.MoveTo("cooldown_player", "secret_forest_01")
	if err == nil {
		t.Error("expected cooldown error for immediate second move")
	}

	// Verify state is still in qingyun_range_01 (previous move took effect)
	state = svc.GetPlayerExploreInfo("cooldown_player")
	if state.RegionID != "qingyun_range_01" {
		t.Errorf("expected region qingyun_range_01 after first move, got %q", state.RegionID)
	}
}

// TestExplore tests the Explore method's event probability distribution.
func TestExplore(t *testing.T) {
	svc := newTestExploreService()

	t.Run("explore with sufficient AP returns a result", func(t *testing.T) {
		state := svc.GetPlayerExploreInfo("explorer1")
		state.ActionPoints = 100

		result := svc.Explore("explorer1", 5, nil)
		if result == nil {
			t.Fatal("Explore() returned nil")
		}
		if result.Message == "" {
			t.Error("explore result should have a message")
		}
		// Without EncounterService, the possible event types are:
		// monster, resource, nothing (encounter falls through to resource/nothing)
		validTypes := map[string]bool{"monster": true, "resource": true, "nothing": true}
		if !validTypes[result.EventType] {
			t.Errorf("unexpected event type: %q (expected monster/resource/nothing without EncounterService)", result.EventType)
		}
	})

	t.Run("explore with insufficient AP returns nothing", func(t *testing.T) {
		state := svc.GetPlayerExploreInfo("tired_player")
		state.ActionPoints = 0

		result := svc.Explore("tired_player", 5, nil)
		if result == nil {
			t.Fatal("Explore() returned nil")
		}
		if result.EventType != "nothing" {
			t.Errorf("expected nothing event for low AP, got %q", result.EventType)
		}
	})

	t.Run("explore event distribution over many attempts", func(t *testing.T) {
		state := svc.GetPlayerExploreInfo("stat_player")
		state.ActionPoints = 10000 // Plenty of AP

		counts := map[string]int{"monster": 0, "resource": 0, "nothing": 0}
		attempts := 500

		for i := 0; i < attempts; i++ {
			result := svc.Explore("stat_player", 10, nil)
			counts[result.EventType]++
			state.ActionPoints += ExploreAPCost // Replenish AP
		}

		// Verify all event types that can appear without EncounterService appeared
		for eventType, count := range counts {
			if count == 0 {
				t.Errorf("event type %q never occurred in %d attempts", eventType, attempts)
			}
		}

		// Verify nothing events are not the only type
		if counts["nothing"] == attempts {
			t.Error("all events were 'nothing', expected variety")
		}
	})
}

// TestGather tests the Gather method for success and failure cases.
func TestGather(t *testing.T) {
	svc := newTestExploreService()

	t.Run("gather in wrong region fails", func(t *testing.T) {
		state := svc.GetPlayerExploreInfo("wrong_region_player")
		state.ActionPoints = 100
		// Player is in newbie_village_01, but spot_ore_01 is in qingyun_range_01
		_, msg, err := svc.Gather("wrong_region_player", "spot_ore_01")
		if err == nil {
			t.Error("expected error for gathering in wrong region")
		}
		if msg != "" {
			t.Errorf("expected empty message on error, got %q", msg)
		}
	})

	t.Run("gather non-existent spot fails", func(t *testing.T) {
		state := svc.GetPlayerExploreInfo("no_spot_player")
		state.ActionPoints = 100
		_, _, err := svc.Gather("no_spot_player", "nonexistent_spot")
		if err == nil {
			t.Error("expected error for non-existent spot")
		}
	})

	t.Run("gather with insufficient AP fails", func(t *testing.T) {
		state := svc.GetPlayerExploreInfo("low_ap_gatherer")
		state.ActionPoints = 0
		_, _, err := svc.Gather("low_ap_gatherer", "spot_herb_01")
		if err == nil {
			t.Error("expected insufficient AP error")
		}
	})

	t.Run("gather in same region proceeds", func(t *testing.T) {
		state := svc.GetPlayerExploreInfo("gatherer1")
		state.ActionPoints = 100
		state.RegionID = "newbie_village_01"

		drops, msg, err := svc.Gather("gatherer1", "spot_herb_01")
		if err != nil {
			t.Fatalf("Gather() unexpected error: %v", err)
		}

		if drops != nil && len(drops) > 0 {
			if msg == "" {
				t.Error("successful gather should have a message")
			}
			if drops[0].ItemID != "herb_yinchen_01" {
				t.Errorf("item ID = %q, want herb_yinchen_01", drops[0].ItemID)
			}
		} else if drops == nil || len(drops) == 0 {
			if msg == "" {
				t.Error("failed gather should have a message")
			}
		}
	})
}

// TestGetRegionConnections verifies adjacency checks.
func TestGetRegionConnections(t *testing.T) {
	svc := newTestExploreService()

	t.Run("get connections for valid region", func(t *testing.T) {
		conns := svc.GetRegionConnections("newbie_village_01")
		if len(conns) == 0 {
			t.Fatal("expected connections, got none")
		}
		connIDs := make(map[string]bool)
		for _, c := range conns {
			connIDs[c.ID] = true
		}
		if !connIDs["qingyun_range_01"] {
			t.Error("expected connection to qingyun_range_01")
		}
		if !connIDs["secret_forest_01"] {
			t.Error("expected connection to secret_forest_01")
		}
	})

	t.Run("get connections for non-existent region", func(t *testing.T) {
		conns := svc.GetRegionConnections("nonexistent")
		if conns != nil {
			t.Errorf("expected nil for non-existent region, got %d connections", len(conns))
		}
	})

	t.Run("verify bidirectional connections", func(t *testing.T) {
		nbConns := svc.GetRegionConnections("newbie_village_01")
		qyConns := svc.GetRegionConnections("qingyun_range_01")

		nbConnIDs := make(map[string]bool)
		for _, c := range nbConns {
			nbConnIDs[c.ID] = true
		}
		qyConnIDs := make(map[string]bool)
		for _, c := range qyConns {
			qyConnIDs[c.ID] = true
		}

		if !nbConnIDs["qingyun_range_01"] {
			t.Error("newbie_village should connect to qingyun_range")
		}
		if !qyConnIDs["newbie_village_01"] {
			t.Error("qingyun_range should connect back to newbie_village")
		}
	})
}

// TestNPCInteraction verifies NPC query methods.
func TestNPCInteraction(t *testing.T) {
	svc := newTestExploreService()

	t.Run("get NPCs in a region", func(t *testing.T) {
		npcs := svc.GetRegionNPCs("newbie_village_01")
		if len(npcs) == 0 {
			t.Fatal("expected NPCs in newbie_village_01")
		}
		found := false
		for _, n := range npcs {
			if n.ID == "npc_elder" {
				found = true
				if len(n.Dialogues) == 0 {
					t.Error("NPC should have dialogues")
				}
				break
			}
		}
		if !found {
			t.Error("expected npc_elder in newbie_village_01")
		}
	})

	t.Run("get NPCs in region with no NPCs returns nil/empty", func(t *testing.T) {
		npcs := svc.GetRegionNPCs("ancient_ruins_01")
		// The method returns nil when no NPCs match (var result []*model.NPC).
		// Both nil and empty slices are acceptable empty results.
		if npcs != nil && len(npcs) != 0 {
			t.Errorf("expected no NPCs, got %d", len(npcs))
		}
	})

	t.Run("get specific NPC by ID", func(t *testing.T) {
		npc, ok := svc.GetNPC("npc_merchant")
		if !ok {
			t.Fatal("expected to find npc_merchant")
		}
		if npc.Type != model.NPCShop {
			t.Errorf("expected shop type, got %q", npc.Type)
		}
	})

	t.Run("get non-existent NPC returns false", func(t *testing.T) {
		_, ok := svc.GetNPC("nonexistent_npc")
		if ok {
			t.Error("expected false for non-existent NPC")
		}
	})
}

// TestGetGatheringSpot tests the gathering spot query methods.
func TestGetGatheringSpot(t *testing.T) {
	svc := newTestExploreService()

	t.Run("get specific spot by ID", func(t *testing.T) {
		spot, ok := svc.GetGatheringSpot("spot_herb_01")
		if !ok {
			t.Fatal("expected to find spot_herb_01")
		}
		if spot.ItemID != "herb_yinchen_01" {
			t.Errorf("expected herb_yinchen_01, got %q", spot.ItemID)
		}
	})

	t.Run("get non-existent spot returns false", func(t *testing.T) {
		_, ok := svc.GetGatheringSpot("nonexistent_spot")
		if ok {
			t.Error("expected false for non-existent spot")
		}
	})

	t.Run("get spots in region", func(t *testing.T) {
		spots := svc.GetRegionGatheringSpots("newbie_village_01")
		if len(spots) == 0 {
			t.Fatal("expected spots in newbie_village_01")
		}
		if spots[0].RegionID != "newbie_village_01" {
			t.Errorf("expected newbie_village_01 region, got %q", spots[0].RegionID)
		}
	})
}

// TestPlayerState_APRegen tests action point regeneration logic.
func TestPlayerState_APRegen(t *testing.T) {
	svc := newTestExploreService()

	oldTime := time.Now().Add(-3 * time.Hour)
	playerState := &PlayerState{
		UserID:       "regen_player",
		RegionID:     DefaultRegionID,
		ActionPoints: 0,
		LastAPUpdate: oldTime,
		LastMoveAt:   oldTime,
	}
	svc.playerStates["regen_player"] = playerState

	state := svc.GetPlayerExploreInfo("regen_player")
	if state.ActionPoints <= 0 {
		t.Errorf("AP should have regenerated, got %d", state.ActionPoints)
	}
	// Should have gained at least 2 hours of regen = 40 AP
	if state.ActionPoints < 40 {
		t.Errorf("expected at least 40 AP after 2+ hours, got %d", state.ActionPoints)
	}
	// Cap at max
	if state.ActionPoints > ActionPointMax {
		t.Errorf("AP should not exceed max %d, got %d", ActionPointMax, state.ActionPoints)
	}
}

// TestDefaultPlayerState verifies default state creation.
func TestDefaultPlayerState(t *testing.T) {
	svc := newTestExploreService()

	state := svc.GetPlayerExploreInfo("new_player")
	if state == nil {
		t.Fatal("GetPlayerExploreInfo returned nil")
	}
	if state.UserID != "new_player" {
		t.Errorf("user ID = %q, want new_player", state.UserID)
	}
	if state.RegionID != DefaultRegionID {
		t.Errorf("region ID = %q, want %q", state.RegionID, DefaultRegionID)
	}
	if state.ActionPoints != ActionPointMax {
		t.Errorf("AP = %d, want %d", state.ActionPoints, ActionPointMax)
	}
	if len(state.DiscoveredRegions) != 1 || state.DiscoveredRegions[0] != DefaultRegionID {
		t.Errorf("discovered regions should start with %q", DefaultRegionID)
	}
}

// TestGetAllRegions tests the GetAllRegions method.
func TestGetAllRegions(t *testing.T) {
	svc := newTestExploreService()

	regions := svc.GetAllRegions()
	if len(regions) != 5 {
		t.Errorf("expected 5 regions, got %d", len(regions))
	}

	regionIDs := make(map[string]bool)
	for _, r := range regions {
		regionIDs[r.ID] = true
	}
	for _, id := range []string{"newbie_village_01", "qingyun_range_01", "star_city_01", "ancient_ruins_01", "secret_forest_01"} {
		if !regionIDs[id] {
			t.Errorf("missing region: %s", id)
		}
	}
}

// TestGetPlayerActionPoints tests the AP query method.
func TestGetPlayerActionPoints(t *testing.T) {
	svc := newTestExploreService()

	current, max := svc.GetPlayerActionPoints("ap_player")
	if current <= 0 {
		t.Errorf("expected positive AP, got %d", current)
	}
	if max != ActionPointMax {
		t.Errorf("max AP = %d, want %d", max, ActionPointMax)
	}
}

// TestMoveTo_DiscoverNewRegion tests the discovery of new regions.
func TestMoveTo_DiscoverNewRegion(t *testing.T) {
	svc := newTestExploreService()

	state := svc.GetPlayerExploreInfo("discoverer")
	state.LastMoveAt = time.Now().Add(-3 * time.Second)
	state.ActionPoints = ActionPointMax

	if len(state.DiscoveredRegions) != 1 {
		t.Fatalf("expected 1 discovered region, got %d", len(state.DiscoveredRegions))
	}

	// Move to a new region
	result, err := svc.MoveTo("discoverer", "qingyun_range_01")
	if err != nil {
		t.Fatalf("MoveTo: %v", err)
	}
	if !result.DiscoveredNew {
		t.Error("expected new discovery for first visit to qingyun_range")
	}

	state = svc.GetPlayerExploreInfo("discoverer")
	if len(state.DiscoveredRegions) != 2 {
		t.Errorf("expected 2 discovered regions, got %d", len(state.DiscoveredRegions))
	}

	// Move back to known region - should not add duplicate to discovered list
	state.LastMoveAt = time.Now().Add(-3 * time.Second)
	result, err = svc.MoveTo("discoverer", "newbie_village_01")
	if err != nil {
		t.Fatalf("MoveTo back: %v", err)
	}
	// Verify the discovered regions list did not grow (returning to known region)
	state = svc.GetPlayerExploreInfo("discoverer")
	if len(state.DiscoveredRegions) != 2 {
		t.Errorf("expected 2 discovered regions (no duplicate), got %d: %v",
			len(state.DiscoveredRegions), state.DiscoveredRegions)
	}
}

// TestGather_SuccessRate verifies gather success rates at different difficulties.
func TestGather_SuccessRate(t *testing.T) {
	svc := newTestExploreService()

	t.Run("low difficulty gather succeeds often", func(t *testing.T) {
		successes := 0
		attempts := 50

		for i := 0; i < attempts; i++ {
			playerID := "gather_stat_1"
			state := svc.GetPlayerExploreInfo(playerID)
			state.ActionPoints = 100
			state.RegionID = "newbie_village_01"

			drops, _, err := svc.Gather(playerID, "spot_herb_01")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if drops != nil && len(drops) > 0 {
				successes++
			}
			state.ActionPoints += GatherAPCost // Replenish
		}

		// Difficulty 1 => success rate = 0.92, so we expect many successes
		if successes == 0 {
			t.Errorf("expected at least some successes out of %d attempts, got 0", attempts)
		}
	})
}
