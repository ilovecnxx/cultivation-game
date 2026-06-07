package service

import (
	"testing"
	"time"

	"cultivation-game/services/world/internal/model"
)

// newTestQuestService creates a QuestService with predefined test quests.
func newTestQuestService() *QuestService {
	quests := map[string]*model.Quest{
		"quest_first": {
			ID:   "quest_first",
			Name: "First Quest",
			Type: model.QuestMain,
			Requirements: []model.QuestRequirement{
				{Type: "talk_to_npc", TargetID: "npc_elder", Count: 1},
			},
			Rewards: []model.QuestReward{
				{Type: "exp", Quantity: 100},
			},
			LevelRequired: 1,
		},
		"quest_second": {
			ID:   "quest_second",
			Name: "Second Quest",
			Type: model.QuestMain,
			Requirements: []model.QuestRequirement{
				{Type: "kill_monster", TargetID: "monster_wolf", Count: 3},
			},
			Rewards: []model.QuestReward{
				{Type: "exp", Quantity: 200},
				{Type: "money", Quantity: 50},
			},
			Prerequisites:  []string{"quest_first"},
			LevelRequired:  2,
		},
		"quest_high_level": {
			ID:   "quest_high_level",
			Name: "High Level Quest",
			Type: model.QuestMain,
			Requirements: []model.QuestRequirement{
				{Type: "kill_monster", TargetID: "monster_boss", Count: 1},
			},
			Rewards: []model.QuestReward{
				{Type: "exp", Quantity: 500},
			},
			LevelRequired: 10,
		},
		"quest_daily_gather": {
			ID:   "quest_daily_gather",
			Name: "Daily Gathering",
			Type: model.QuestDaily,
			Requirements: []model.QuestRequirement{
				{Type: "gather_item", TargetID: "any", Count: 5},
			},
			Rewards: []model.QuestReward{
				{Type: "exp", Quantity: 150},
				{Type: "item", ID: "pill_qi", Quantity: 1},
			},
			LevelRequired: 1,
		},
		"quest_daily_kill": {
			ID:   "quest_daily_kill",
			Name: "Daily Killing",
			Type: model.QuestDaily,
			Requirements: []model.QuestRequirement{
				{Type: "kill_monster", TargetID: "any", Count: 10},
			},
			Rewards: []model.QuestReward{
				{Type: "exp", Quantity: 200},
				{Type: "money", Quantity: 50},
			},
			LevelRequired: 1,
		},
		"quest_side_help": {
			ID:   "quest_side_help",
			Name: "Side Help",
			Type: model.QuestSide,
			Requirements: []model.QuestRequirement{
				{Type: "talk_to_npc", TargetID: "npc_villager", Count: 2},
			},
			Rewards: []model.QuestReward{
				{Type: "exp", Quantity: 50},
			},
			Prerequisites: []string{"quest_first"},
			LevelRequired: 1,
		},
	}

	return &QuestService{
		quests:         quests,
		playerQuests:   make(map[string]map[string]*model.PlayerQuest),
		dailyCompleted: make(map[string]map[string]string),
	}
}

// TestAcceptQuest tests accepting quests under various conditions.
func TestAcceptQuest(t *testing.T) {
	svc := newTestQuestService()

	// Set up: accept quest_first for setup_player.
	// This player is used only for the "already accepted" and "prerequisites" checks.
	err := svc.AcceptQuest("setup_player", "quest_first", 1)
	if err != nil {
		t.Fatalf("setup: failed to accept quest_first: %v", err)
	}

	tests := []struct {
		name        string
		playerID    string
		questID     string
		playerLevel int
		wantErr     bool
		errContains string
	}{
		{
			name:        "accept valid quest with sufficient level",
			playerID:    "fresh_player",
			questID:     "quest_first",
			playerLevel: 1,
			wantErr:     false,
		},
		{
			name:        "reject non-existent quest",
			playerID:    "fresh_player",
			questID:     "quest_nonexistent",
			playerLevel: 1,
			wantErr:     true,
			errContains: "任务不存在",
		},
		{
			name:        "reject quest with insufficient level",
			playerID:    "fresh_player",
			questID:     "quest_high_level",
			playerLevel: 5,
			wantErr:     true,
			errContains: "境界不足",
		},
		{
			name:        "reject already in-progress quest",
			playerID:    "setup_player",
			questID:     "quest_first",
			playerLevel: 1,
			wantErr:     true,
			errContains: "任务已接取",
		},
		{
			name:        "reject quest when prerequisites not met",
			playerID:    "setup_player",
			questID:     "quest_second",
			playerLevel: 2,
			wantErr:     true,
			errContains: "前置任务",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.AcceptQuest(tt.playerID, tt.questID, tt.playerLevel)
			if tt.wantErr {
				if err == nil {
					t.Errorf("AcceptQuest() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("AcceptQuest() error = %q, want containing %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AcceptQuest() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestAcceptQuest_Prerequisites verifies prerequisite chain resolution.
func TestAcceptQuest_Prerequisites(t *testing.T) {
	svc := newTestQuestService()

	// Complete quest_first for player2
	if err := svc.AcceptQuest("player2", "quest_first", 1); err != nil {
		t.Fatalf("setup: AcceptQuest(quest_first): %v", err)
	}

	// Simulate completing the quest by updating progress and completing
	svc.UpdateProgress("player2", model.QuestEvent{
		Type: "talk_to_npc", TargetID: "npc_elder", Count: 1,
	})

	// Verify it completed
	pq, ok := svc.GetPlayerQuest("player2", "quest_first")
	if !ok {
		t.Fatal("player2 should have quest_first")
	}
	if pq.Status != model.QuestCompleted {
		t.Fatalf("quest_first status = %q, want %q", pq.Status, model.QuestCompleted)
	}

	// Submit quest_first
	if _, err := svc.CompleteQuest("player2", "quest_first"); err != nil {
		t.Fatalf("CompleteQuest(quest_first): %v", err)
	}

	// Now quest_second should be available - prerequisites met
	err := svc.AcceptQuest("player2", "quest_second", 2)
	if err != nil {
		t.Errorf("AcceptQuest(quest_second) should succeed when prerequisites are met, got: %v", err)
	}
}

// TestUpdateProgress verifies quest progress tracking via events.
func TestUpdateProgress(t *testing.T) {
	svc := newTestQuestService()

	// Prerequisite: complete quest_first before accepting quest_second
	if err := svc.AcceptQuest("player3", "quest_first", 1); err != nil {
		t.Fatalf("setup: AcceptQuest(quest_first): %v", err)
	}
	svc.UpdateProgress("player3", model.QuestEvent{
		Type: "talk_to_npc", TargetID: "npc_elder", Count: 1,
	})
	if _, err := svc.CompleteQuest("player3", "quest_first"); err != nil {
		t.Fatalf("setup: CompleteQuest(quest_first): %v", err)
	}

	// Accept a quest with kill requirement (prerequisites now met)
	if err := svc.AcceptQuest("player3", "quest_second", 2); err != nil {
		t.Fatalf("setup: AcceptQuest(quest_second): %v", err)
	}

	// Accept a daily quest
	if err := svc.AcceptQuest("player3", "quest_daily_gather", 1); err != nil {
		t.Fatalf("setup: AcceptQuest(quest_daily_gather): %v", err)
	}

	// Update progress for killing - should only affect quest_second
	svc.UpdateProgress("player3", model.QuestEvent{
		Type: "kill_monster", TargetID: "monster_wolf", Count: 2,
	})

	pq, ok := svc.GetPlayerQuest("player3", "quest_second")
	if !ok {
		t.Fatal("missing quest_second for player3")
	}

	if len(pq.Progress) != 1 {
		t.Fatalf("expected 1 progress entry, got %d", len(pq.Progress))
	}
	if pq.Progress[0].Current != 2 {
		t.Errorf("progress = %d, want 2", pq.Progress[0].Current)
	}
	if pq.Status != model.QuestInProgress {
		t.Errorf("status = %q, want in_progress", pq.Status)
	}

	// Complete the kill requirement
	svc.UpdateProgress("player3", model.QuestEvent{
		Type: "kill_monster", TargetID: "monster_wolf", Count: 1,
	})

	pq, _ = svc.GetPlayerQuest("player3", "quest_second")
	if pq.Progress[0].Current != 3 {
		t.Errorf("progress = %d, want 3", pq.Progress[0].Current)
	}
	if pq.Status != model.QuestCompleted {
		t.Errorf("status should be completed, got %q", pq.Status)
	}

	// Verify daily quest was not affected by kill event
	dailyPQ, ok := svc.GetPlayerQuest("player3", "quest_daily_gather")
	if !ok {
		t.Fatal("missing quest_daily_gather")
	}
	if dailyPQ.Progress[0].Current != 0 {
		t.Errorf("daily gather progress should be 0, got %d", dailyPQ.Progress[0].Current)
	}

	// Update with a matching gather event for daily quest
	svc.UpdateProgress("player3", model.QuestEvent{
		Type: "gather_item", TargetID: "herb_test", Count: 6,
	})

	dailyPQ, _ = svc.GetPlayerQuest("player3", "quest_daily_gather")
	if dailyPQ.Progress[0].Current != 5 {
		t.Errorf("daily gather progress capped at 5, got %d", dailyPQ.Progress[0].Current)
	}
	if dailyPQ.Status != model.QuestCompleted {
		t.Errorf("daily gather should be completed, got %q", dailyPQ.Status)
	}
}

// TestUpdateProgress_EventMatching verifies event matching with special cases.
func TestUpdateProgress_EventMatching(t *testing.T) {
	tests := []struct {
		name     string
		req      model.QuestRequirement
		event    model.QuestEvent
		want     bool
	}{
		{
			name:     "exact match",
			req:      model.QuestRequirement{Type: "kill_monster", TargetID: "monster_wolf"},
			event:    model.QuestEvent{Type: "kill_monster", TargetID: "monster_wolf"},
			want:     true,
		},
		{
			name:     "type mismatch",
			req:      model.QuestRequirement{Type: "kill_monster", TargetID: "monster_wolf"},
			event:    model.QuestEvent{Type: "gather_item", TargetID: "monster_wolf"},
			want:     false,
		},
		{
			name:     "target any wildcard",
			req:      model.QuestRequirement{Type: "kill_monster", TargetID: "any"},
			event:    model.QuestEvent{Type: "kill_monster", TargetID: "monster_any"},
			want:     true,
		},
		{
			name:     "reach_realm higher match",
			req:      model.QuestRequirement{Type: "reach_realm", TargetID: "qi_refining_3"},
			event:    model.QuestEvent{Type: "reach_realm", TargetID: "higher"},
			want:     true,
		},
		{
			name:     "reach_realm exact match",
			req:      model.QuestRequirement{Type: "reach_realm", TargetID: "qi_refining_3"},
			event:    model.QuestEvent{Type: "reach_realm", TargetID: "qi_refining_3"},
			want:     true,
		},
		{
			name:     "reach_realm wrong realm no match",
			req:      model.QuestRequirement{Type: "reach_realm", TargetID: "qi_refining_3"},
			event:    model.QuestEvent{Type: "reach_realm", TargetID: "foundation"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchRequirement(tt.req, tt.event)
			if got != tt.want {
				t.Errorf("matchRequirement(%+v, %+v) = %v, want %v", tt.req, tt.event, got, tt.want)
			}
		})
	}
}

// TestCompleteQuest verifies quest submission and reward generation.
func TestCompleteQuest(t *testing.T) {
	svc := newTestQuestService()

	// Setup: create a player with quest_first completed AND quest_second accepted but not completed.
	// Step 1: Accept and complete quest_first for player4
	if err := svc.AcceptQuest("player4", "quest_first", 1); err != nil {
		t.Fatalf("setup: AcceptQuest(quest_first): %v", err)
	}
	svc.UpdateProgress("player4", model.QuestEvent{
		Type: "talk_to_npc", TargetID: "npc_elder", Count: 1,
	})
	// quest_first is now "completed"

	// Step 2: Create a player for "submit not yet completed" test.
	// Accept quest_first for player5, complete it, submit it, then accept quest_second.
	if err := svc.AcceptQuest("player5", "quest_first", 1); err != nil {
		t.Fatalf("setup2: AcceptQuest(quest_first): %v", err)
	}
	svc.UpdateProgress("player5", model.QuestEvent{
		Type: "talk_to_npc", TargetID: "npc_elder", Count: 1,
	})
	if _, err := svc.CompleteQuest("player5", "quest_first"); err != nil {
		t.Fatalf("setup2: CompleteQuest(quest_first): %v", err)
	}
	if err := svc.AcceptQuest("player5", "quest_second", 2); err != nil {
		t.Fatalf("setup2: AcceptQuest(quest_second): %v", err)
	}
	// player5 now has quest_second in progress (not completed)

	tests := []struct {
		name        string
		playerID    string
		questID     string
		wantErr     bool
		errContains string
		wantRewards int
	}{
		{
			name:        "submit completed quest",
			playerID:    "player4",
			questID:     "quest_first",
			wantErr:     false,
			wantRewards: 1,
		},
		{
			name:        "submit non-existent quest",
			playerID:    "player4",
			questID:     "quest_nonexistent",
			wantErr:     true,
			errContains: "未接取",
		},
		{
			name:        "submit quest not yet completed",
			playerID:    "player5",
			questID:     "quest_second",
			wantErr:     true,
			errContains: "条件未满足",
		},
		{
			name:        "submit already submitted quest",
			playerID:    "player5",
			questID:     "quest_first",
			wantErr:     true,
			errContains: "条件未满足",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewards, err := svc.CompleteQuest(tt.playerID, tt.questID)
			if tt.wantErr {
				if err == nil {
					t.Error("CompleteQuest() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("CompleteQuest() error = %q, want containing %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CompleteQuest() unexpected error: %v", err)
				}
				if len(rewards) != tt.wantRewards {
					t.Errorf("got %d rewards, want %d", len(rewards), tt.wantRewards)
				}
			}
		})
	}
}

// TestDailyQuestReset verifies daily quest completion tracking.
func TestDailyQuestReset(t *testing.T) {
	svc := newTestQuestService()

	// Accept and complete daily quest
	if err := svc.AcceptQuest("player5", "quest_daily_gather", 1); err != nil {
		t.Fatalf("setup: AcceptQuest: %v", err)
	}
	svc.UpdateProgress("player5", model.QuestEvent{
		Type: "gather_item", TargetID: "any", Count: 5,
	})

	if _, err := svc.CompleteQuest("player5", "quest_daily_gather"); err != nil {
		t.Fatalf("setup: CompleteQuest: %v", err)
	}

	// Verify daily completed is recorded
	if svc.dailyCompleted["player5"] == nil {
		t.Fatal("dailyCompleted should be recorded")
	}
	today := time.Now().Format("2006-01-02")
	if svc.dailyCompleted["player5"]["quest_daily_gather"] != today {
		t.Errorf("dailyCompleted date = %q, want %q",
			svc.dailyCompleted["player5"]["quest_daily_gather"], today)
	}

	// Trying to accept again today should fail
	err := svc.AcceptQuest("player5", "quest_daily_gather", 1)
	if err == nil {
		t.Error("AcceptQuest should fail for already completed daily")
	}
	if !contains(err.Error(), "已完成该每日") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestGetAvailableQuests verifies the available quest filtering logic.
func TestGetAvailableQuests(t *testing.T) {
	svc := newTestQuestService()

	t.Run("returns quests within level range", func(t *testing.T) {
		available := svc.GetAvailableQuests("player6", 1)
		// At level 1: quest_first (L1), quest_daily_gather (L1), quest_daily_kill (L1)
		// quest_second is L2, quest_high_level is L10, quest_side_help needs prerequisite quest_first done
		if len(available) != 3 {
			t.Errorf("expected 3 available quests at level 1, got %d", len(available))
		}
	})

	t.Run("level 5 unlocks more quests", func(t *testing.T) {
		available := svc.GetAvailableQuests("player7", 5)
		// At level 5: quests with LevelRequired <= 5 that don't need prereqs:
		// quest_first (L1), quest_daily_gather (L1), quest_daily_kill (L1) = 3
		// quest_second (L2) and quest_side_help (L1) require quest_first completed.
		// quest_high_level (L10) is still excluded.
		if len(available) != 3 {
			t.Errorf("expected 3 available quests at level 5, got %d: %+v", len(available), available)
		}
	})

	t.Run("in-progress quests are excluded", func(t *testing.T) {
		svc.AcceptQuest("player8", "quest_first", 1)
		available := svc.GetAvailableQuests("player8", 1)
		for _, q := range available {
			if q.ID == "quest_first" {
				t.Error("in-progress quest should not be in available list")
			}
		}
	})

	t.Run("quests with unmet prerequisites are excluded", func(t *testing.T) {
		available := svc.GetAvailableQuests("player9", 2)
		// quest_second requires quest_first completed, which player9 hasn't done
		for _, q := range available {
			if q.ID == "quest_second" {
				t.Error("quest_second requires quest_first as prerequisite")
			}
		}
	})
}

// TestGetPlayerQuests verifies the player quest query methods.
func TestGetPlayerQuests(t *testing.T) {
	svc := newTestQuestService()

	// No quests yet
	quests := svc.GetPlayerQuests("player10")
	if quests != nil {
		t.Errorf("expected nil, got %d quests", len(quests))
	}

	// Accept a quest
	svc.AcceptQuest("player10", "quest_first", 1)
	quests = svc.GetPlayerQuests("player10")
	if len(quests) != 1 {
		t.Fatalf("expected 1 quest, got %d", len(quests))
	}
	if quests[0].QuestID != "quest_first" {
		t.Errorf("quest ID = %q, want quest_first", quests[0].QuestID)
	}
	if quests[0].Status != model.QuestInProgress {
		t.Errorf("status = %q, want in_progress", quests[0].Status)
	}
}

// Helper: contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
