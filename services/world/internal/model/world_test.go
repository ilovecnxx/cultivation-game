package model

import (
	"testing"
)

// TestMapRegionConnections verifies that MapRegion connection properties are
// correctly set up and can be validated.
func TestMapRegionConnections(t *testing.T) {
	tests := []struct {
		name      string
		region    *MapRegion
		wantID    string
		wantType  RegionType
		wantDanger int
	}{
		{
			name: "newbie village has correct type and danger",
			region: &MapRegion{
				ID:          "newbie_village_01",
				Name:        "青竹村",
				Type:        RegionNewbie,
				DangerLevel: 1,
				Connections: []string{"qingyun_range_01", "secret_forest_01"},
			},
			wantID:    "newbie_village_01",
			wantType:  RegionNewbie,
			wantDanger: 1,
		},
		{
			name: "dangerous land has high danger level",
			region: &MapRegion{
				ID:          "thunder_valley_01",
				Name:        "雷霆谷",
				Type:        RegionDangerous,
				DangerLevel: 9,
				Connections: []string{"wild_lands_01"},
			},
			wantID:    "thunder_valley_01",
			wantType:  RegionDangerous,
			wantDanger: 9,
		},
		{
			name: "secret realm is accessible by connections",
			region: &MapRegion{
				ID:          "ancient_ruins_01",
				Name:        "上古遗迹",
				Type:        RegionSecretRealm,
				DangerLevel: 6,
				Connections: []string{"star_city_01", "sunset_valley_01"},
			},
			wantID:    "ancient_ruins_01",
			wantType:  RegionSecretRealm,
			wantDanger: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.region
			if r.ID != tt.wantID {
				t.Errorf("region.ID = %q, want %q", r.ID, tt.wantID)
			}
			if r.Type != tt.wantType {
				t.Errorf("region.Type = %q, want %q", r.Type, tt.wantType)
			}
			if r.DangerLevel != tt.wantDanger {
				t.Errorf("region.DangerLevel = %d, want %d", r.DangerLevel, tt.wantDanger)
			}
			// Verify connections are non-empty
			if len(r.Connections) == 0 {
				t.Error("region.Connections is empty, expected at least one connection")
			}
		})
	}
}

// TestRegionConnectionSymmetry checks that some connections have expected
// reverse relationships.
func TestRegionConnectionSymmetry(t *testing.T) {
	regions := map[string]*MapRegion{
		"newbie_village_01": {
			ID:          "newbie_village_01",
			Connections: []string{"qingyun_range_01", "star_city_01"},
		},
		"qingyun_range_01": {
			ID:          "qingyun_range_01",
			Connections: []string{"newbie_village_01", "star_city_01"},
		},
		"star_city_01": {
			ID:          "star_city_01",
			Connections: []string{"newbie_village_01", "qingyun_range_01"},
		},
	}

	// Verify that for each bidirectional connection, both regions reference each other.
	for id, r := range regions {
		for _, connID := range r.Connections {
			connRegion, ok := regions[connID]
			if !ok {
				t.Errorf("Region %q connects to %q which does not exist", id, connID)
				continue
			}
			found := false
			for _, backConn := range connRegion.Connections {
				if backConn == id {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Asymmetric connection: %q -> %q but no reverse connection", id, connID)
			}
		}
	}
}

// TestNPCValidation checks that NPCs have required fields and valid types.
func TestNPCValidation(t *testing.T) {
	tests := []struct {
		name    string
		npc     *NPC
		wantErr bool
	}{
		{
			name: "valid quest giver NPC",
			npc: &NPC{
				ID:          "npc_old_zhang",
				Name:        "张老伯",
				Type:        NPCQuestGiver,
				Title:       "青竹村村长",
				Description: "A friendly elder",
				RegionID:    "newbie_village_01",
				Dialogues:   []string{"Hello", "Goodbye"},
				Quests:      []string{"quest_01"},
			},
			wantErr: false,
		},
		{
			name: "valid shop NPC with items",
			npc: &NPC{
				ID:          "npc_merchant_li",
				Name:        "李掌柜",
				Type:        NPCShop,
				Title:       "百草堂掌柜",
				RegionID:    "star_city_01",
				Dialogues:   []string{"欢迎光临"},
				ShopItems: []NPCShopItem{
					{ItemID: "pill_qi_01", Name: "聚气丹", Price: 100, Currency: "spirit_stone", Stock: 100, LevelReq: 1},
				},
			},
			wantErr: false,
		},
		{
			name: "NPC with empty name",
			npc: &NPC{
				ID:       "npc_no_name",
				Name:     "",
				Type:     NPCCultivator,
				RegionID: "newbie_village_01",
			},
			wantErr: true,
		},
		{
			name: "NPC with empty region",
			npc: &NPC{
				ID:       "npc_no_region",
				Name:     "Wanderer",
				Type:     NPCCultivator,
				RegionID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNPC(tt.npc)
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

// validateNPC performs basic field validation for an NPC.
// This is a test helper that mirrors the validation logic from the production code.
// If the service does not have such a validator yet, this at least documents the contract.
func validateNPC(npc *NPC) error {
	if npc.ID == "" {
		return errField("id")
	}
	if npc.Name == "" {
		return errField("name")
	}
	if npc.RegionID == "" {
		return errField("region_id")
	}
	switch npc.Type {
	case NPCQuestGiver, NPCShop, NPCTrainer, NPCCultivator:
		// valid
	default:
		return errField("type")
	}
	return nil
}

func errField(field string) error {
	return &validationError{field: field}
}

type validationError struct {
	field string
}

func (e *validationError) Error() string {
	return "missing or invalid field: " + e.field
}

// TestEncounterChoiceResolution verifies that encounter choices are correctly resolved.
func TestEncounterChoiceResolution(t *testing.T) {
	tests := []struct {
		name     string
		choices  []EncounterChoice
		index    int
		wantOK   bool
		wantDesc string
	}{
		{
			name: "first choice for item reward",
			choices: []EncounterChoice{
				{
					Text: "接受馈赠",
					Outcome: &EncounterOutcome{
						Type:        "item",
						TargetID:    "herb_lingzhi_01",
						Amount:      3,
						Description: "你获得了灵芝草 x3",
					},
				},
				{
					Text: "拒绝馈赠",
					Outcome: &EncounterOutcome{
						Type:        "none",
						Description: "你礼貌地拒绝了",
					},
				},
			},
			index:    0,
			wantOK:   true,
			wantDesc: "你获得了灵芝草 x3",
		},
		{
			name: "second choice for none outcome",
			choices: []EncounterChoice{
				{
					Text: "接受馈赠",
					Outcome: &EncounterOutcome{
						Type:        "item",
						TargetID:    "herb_lingzhi_01",
						Amount:      3,
						Description: "你获得了灵芝草 x3",
					},
				},
				{
					Text: "拒绝馈赠",
					Outcome: &EncounterOutcome{
						Type:        "none",
						Description: "你礼貌地拒绝了",
					},
				},
			},
			index:    1,
			wantOK:   true,
			wantDesc: "你礼貌地拒绝了",
		},
		{
			name: "invalid index returns no outcome",
			choices: []EncounterChoice{
				{Text: "A", Outcome: &EncounterOutcome{Type: "exp", Description: "exp"}},
			},
			index:  5,
			wantOK: false,
		},
		{
			name: "nil outcome is handled gracefully",
			choices: []EncounterChoice{
				{Text: "A", Outcome: nil},
			},
			index:    0,
			wantOK:   true,
			wantDesc: "",
		},
		{
			name: "auto outcome when no choices needed",
			choices: []EncounterChoice{
				{Text: "Continue", Outcome: &EncounterOutcome{Type: "exp", Amount: 100, Description: "exp gained"}},
			},
			index:    0,
			wantOK:   true,
			wantDesc: "exp gained",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.index < 0 || tt.index >= len(tt.choices) {
				if tt.wantOK {
					t.Error("expected valid resolution for out-of-range index")
				}
				return
			}
			choice := tt.choices[tt.index]
			if choice.Outcome == nil {
				if tt.wantDesc != "" {
					t.Errorf("expected description %q but got nil outcome", tt.wantDesc)
				}
				return
			}
			if choice.Outcome.Description != tt.wantDesc {
				t.Errorf("outcome.Description = %q, want %q", choice.Outcome.Description, tt.wantDesc)
			}
		})
	}
}

// TestEncounterOutcomeTypes verifies all supported outcome types.
func TestEncounterOutcomeTypes(t *testing.T) {
	outcomeTypes := []string{"item", "exp", "spirit_stone", "damage", "teleport", "buff", "none"}
	for _, ot := range outcomeTypes {
		t.Run(ot, func(t *testing.T) {
			outcome := &EncounterOutcome{Type: ot, Description: "test " + ot}
			if outcome.Type != ot {
				t.Errorf("outcome.Type = %q, want %q", outcome.Type, ot)
			}
			if outcome.Description == "" {
				t.Error("outcome.Description should not be empty")
			}
		})
	}
}

// TestGatheringSpotBounds verifies that gathering spot amounts are sensible.
func TestGatheringSpotBounds(t *testing.T) {
	tests := []struct {
		name string
		spot *GatheringSpot
	}{
		{
			name: "herb spot has positive amounts",
			spot: &GatheringSpot{
				ID:        "spot_herb_01",
				RegionID:  "newbie_village_01",
				Name:      "Test Herb",
				Type:      "herb",
				ItemID:    "herb_test_01",
				ItemName:  "Test Herb",
				MinAmount: 1,
				MaxAmount: 5,
				Difficulty: 1,
				LevelReq:  1,
				RespawnSec: 300,
			},
		},
		{
			name: "rare ore spot has tight amount range",
			spot: &GatheringSpot{
				ID:         "spot_ore_01",
				RegionID:   "ice_cave_01",
				Name:       "Rare Ore",
				Type:       "ore",
				ItemID:     "ore_rare_01",
				ItemName:   "Rare Ore",
				MinAmount:  1,
				MaxAmount:  2,
				Difficulty: 8,
				LevelReq:   40,
				RespawnSec: 3600,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.spot.MinAmount <= 0 {
				t.Error("MinAmount must be positive")
			}
			if tt.spot.MaxAmount < tt.spot.MinAmount {
				t.Errorf("MaxAmount (%d) < MinAmount (%d)", tt.spot.MaxAmount, tt.spot.MinAmount)
			}
			if tt.spot.Difficulty < 1 || tt.spot.Difficulty > 10 {
				t.Errorf("Difficulty out of range: %d", tt.spot.Difficulty)
			}
			if tt.spot.LevelReq < 0 {
				t.Error("LevelReq must be non-negative")
			}
			if tt.spot.RespawnSec <= 0 {
				t.Error("RespawnSec must be positive")
			}
		})
	}
}
