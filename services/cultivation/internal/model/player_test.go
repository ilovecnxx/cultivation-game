package model

import (
	"testing"
)

func TestSpiritRootMultiplier(t *testing.T) {
	tests := []struct {
		name       string
		spiritRoots map[string]float64
		want       float64
	}{
		{
			name:       "天灵根 - single root >= 0.9",
			spiritRoots: map[string]float64{"fire": 0.95},
			want:       2.0,
		},
		{
			name:       "天灵根 - highest is >= 0.9 among multiple",
			spiritRoots: map[string]float64{"fire": 0.4, "water": 0.92, "earth": 0.3},
			want:       2.0,
		},
		{
			name:       "地灵根 - highest >= 0.7",
			spiritRoots: map[string]float64{"fire": 0.75},
			want:       1.5,
		},
		{
			name:       "地灵根 - highest exactly 0.7",
			spiritRoots: map[string]float64{"water": 0.7},
			want:       1.5,
		},
		{
			name:       "人灵根 - highest >= 0.4",
			spiritRoots: map[string]float64{"earth": 0.5},
			want:       1.0,
		},
		{
			name:       "人灵根 - highest exactly 0.4",
			spiritRoots: map[string]float64{"wood": 0.4},
			want:       1.0,
		},
		{
			name:       "杂灵根 - highest < 0.4",
			spiritRoots: map[string]float64{"fire": 0.2, "water": 0.35},
			want:       0.7,
		},
		{
			name:       "杂灵根 - no spirit roots",
			spiritRoots: map[string]float64{},
			want:       0.7,
		},
		{
			name:       "杂灵根 - all zero",
			spiritRoots: map[string]float64{"fire": 0.0, "water": 0.0},
			want:       0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Player{SpiritRoots: tt.spiritRoots}
			got := p.SpiritRootMultiplier()
			if got != tt.want {
				t.Errorf("SpiritRootMultiplier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBreakthroughBonus(t *testing.T) {
	tests := []struct {
		name            string
		pillBonuses     map[string]float64
		artifactBonuses map[string]float64
		want            float64
	}{
		{
			name:            "no bonuses",
			pillBonuses:     nil,
			artifactBonuses: nil,
			want:            0.0,
		},
		{
			name:            "empty maps",
			pillBonuses:     map[string]float64{},
			artifactBonuses: map[string]float64{},
			want:            0.0,
		},
		{
			name:            "single pill bonus",
			pillBonuses:     map[string]float64{"pill_breakthrough": 0.15},
			artifactBonuses: nil,
			want:            0.15,
		},
		{
			name:            "multiple pill bonuses",
			pillBonuses:     map[string]float64{"pill_a": 0.125, "pill_b": 0.125},
			artifactBonuses: nil,
			want:            0.25,
		},
		{
			name:            "single artifact bonus",
			pillBonuses:     nil,
			artifactBonuses: map[string]float64{"artifact_protect": 0.25},
			want:            0.25,
		},
		{
			name:            "pill and artifact bonuses combined",
			pillBonuses:     map[string]float64{"pill_breakthrough": 0.15},
			artifactBonuses: map[string]float64{"artifact_protect": 0.25},
			want:            0.40,
		},
		{
			name:            "multiple of both types",
			pillBonuses:     map[string]float64{"pill_a": 0.125, "pill_b": 0.125},
			artifactBonuses: map[string]float64{"artifact_a": 0.25, "artifact_b": 0.125},
			want:            0.625,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Player{
				PillBonuses:     tt.pillBonuses,
				ArtifactBonuses: tt.artifactBonuses,
			}
			got := p.GetBreakthroughBonus()
			if got != tt.want {
				t.Errorf("GetBreakthroughBonus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlayerStatusManagement(t *testing.T) {
	t.Run("default status is idle", func(t *testing.T) {
		p := &Player{ID: 1, Name: "TestPlayer"}
		if p.Status != "" {
			t.Errorf("expected empty default status, got %q", p.Status)
		}
	})

	t.Run("set and get status", func(t *testing.T) {
		p := &Player{ID: 1, Status: "idle"}
		if p.Status != "idle" {
			t.Errorf("expected idle, got %q", p.Status)
		}

		p.Status = "cultivating"
		if p.Status != "cultivating" {
			t.Errorf("expected cultivating, got %q", p.Status)
		}

		p.Status = "adventuring"
		if p.Status != "adventuring" {
			t.Errorf("expected adventuring, got %q", p.Status)
		}

		p.Status = "exploring"
		if p.Status != "exploring" {
			t.Errorf("expected exploring, got %q", p.Status)
		}
	})

	t.Run("cultivation mode defaults to empty", func(t *testing.T) {
		p := &Player{ID: 1}
		if p.CultivationMode != "" {
			t.Errorf("expected empty cultivation mode, got %q", p.CultivationMode)
		}
	})

	t.Run("set cultivation mode", func(t *testing.T) {
		p := &Player{ID: 1, CultivationMode: "online"}
		if p.CultivationMode != "online" {
			t.Errorf("expected online, got %q", p.CultivationMode)
		}
	})
}

func TestPlayerMeditationState(t *testing.T) {
	t.Run("default meditation state", func(t *testing.T) {
		p := &Player{ID: 1}
		if p.IsMeditating {
			t.Error("expected IsMeditating to be false")
		}
		if p.MeditationStart != 0 {
			t.Errorf("expected MeditationStart to be 0, got %d", p.MeditationStart)
		}
		if p.AccumulatedExp != 0 {
			t.Errorf("expected AccumulatedExp to be 0, got %d", p.AccumulatedExp)
		}
	})

	t.Run("set meditation state", func(t *testing.T) {
		p := &Player{ID: 1, IsMeditating: true, MeditationStart: 1000, AccumulatedExp: 500}
		if !p.IsMeditating {
			t.Error("expected IsMeditating to be true")
		}
		if p.MeditationStart != 1000 {
			t.Errorf("expected MeditationStart 1000, got %d", p.MeditationStart)
		}
		if p.AccumulatedExp != 500 {
			t.Errorf("expected AccumulatedExp 500, got %d", p.AccumulatedExp)
		}
	})
}

func TestPlayerRealmFields(t *testing.T) {
	t.Run("default realm values", func(t *testing.T) {
		p := &Player{ID: 1}
		if p.RealmID != 0 {
			t.Errorf("expected RealmID 0, got %d", p.RealmID)
		}
		if p.RealmLevel != 0 {
			t.Errorf("expected RealmLevel 0, got %d", p.RealmLevel)
		}
		if p.Experience != 0 {
			t.Errorf("expected Experience 0, got %d", p.Experience)
		}
	})

	t.Run("set realm values", func(t *testing.T) {
		p := &Player{ID: 1, RealmID: 1, RealmLevel: 3, Experience: 500}
		if p.RealmID != 1 {
			t.Errorf("expected RealmID 1, got %d", p.RealmID)
		}
		if p.RealmLevel != 3 {
			t.Errorf("expected RealmLevel 3, got %d", p.RealmLevel)
		}
		if p.Experience != 500 {
			t.Errorf("expected Experience 500, got %d", p.Experience)
		}
	})
}
