package config

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// DefaultConfig values
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Server defaults
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want '0.0.0.0'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}

	// Game defaults
	if cfg.Game.ElementAdvantageMultiplier != 1.3 {
		t.Errorf("ElementAdvantageMultiplier = %v, want 1.3", cfg.Game.ElementAdvantageMultiplier)
	}
	if cfg.Game.ElementDisadvantageMultiplier != 0.7 {
		t.Errorf("ElementDisadvantageMultiplier = %v, want 0.7", cfg.Game.ElementDisadvantageMultiplier)
	}
	if cfg.Game.CritDamageMultiplier != 2.0 {
		t.Errorf("CritDamageMultiplier = %v, want 2.0", cfg.Game.CritDamageMultiplier)
	}
	if cfg.Game.MatchmakingRange != 3 {
		t.Errorf("MatchmakingRange = %d, want 3", cfg.Game.MatchmakingRange)
	}
	if cfg.Game.MatchmakingTimeout != 60 {
		t.Errorf("MatchmakingTimeout = %d, want 60", cfg.Game.MatchmakingTimeout)
	}
	if cfg.Game.MaxCooldownReduction != 1 {
		t.Errorf("MaxCooldownReduction = %d, want 1", cfg.Game.MaxCooldownReduction)
	}
	if cfg.Game.ArenaMatchScoreRange != 100 {
		t.Errorf("ArenaMatchScoreRange = %d, want 100", cfg.Game.ArenaMatchScoreRange)
	}
	if cfg.Game.ArenaMatchExpandSeconds != 30 {
		t.Errorf("ArenaMatchExpandSeconds = %d, want 30", cfg.Game.ArenaMatchExpandSeconds)
	}
	if cfg.Game.SeasonDurationDays != 30 {
		t.Errorf("SeasonDurationDays = %d, want 30", cfg.Game.SeasonDurationDays)
	}

	// DataPath defaults
	if cfg.DataPath.Monsters != "internal/data/monsters.json" {
		t.Errorf("DataPath.Monsters = %q, want 'internal/data/monsters.json'", cfg.DataPath.Monsters)
	}
	if cfg.DataPath.Skills != "internal/data/skills.json" {
		t.Errorf("DataPath.Skills = %q, want 'internal/data/skills.json'", cfg.DataPath.Skills)
	}
	if cfg.DataPath.Instances != "internal/data/instances.json" {
		t.Errorf("DataPath.Instances = %q, want 'internal/data/instances.json'", cfg.DataPath.Instances)
	}
	if cfg.DataPath.Dungeons != "internal/data/dungeons.json" {
		t.Errorf("DataPath.Dungeons = %q, want 'internal/data/dungeons.json'", cfg.DataPath.Dungeons)
	}
}

// ---------------------------------------------------------------------------
// Load — file not found returns defaults without error
// ---------------------------------------------------------------------------

func TestLoad_FileNotFound(t *testing.T) {
	cfg, err := Load("/tmp/__nonexistent_config_file_12345.json")
	if err != nil {
		t.Fatalf("Load on missing file returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load on missing file returned nil config")
	}
	// Should return default values
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want default 8080", cfg.Server.Port)
	}
}

// ---------------------------------------------------------------------------
// Load — valid JSON overrides specific fields
// ---------------------------------------------------------------------------

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "combat_config.json")

	content := `{
		"game": {
			"element_advantage_multiplier": 2.0,
			"element_disadvantage_multiplier": 0.5,
			"crit_damage_multiplier": 2.5,
			"matchmaking_range": 5,
			"matchmaking_timeout": 120,
			"max_cooldown_reduction": 2
		},
		"server": {
			"host": "127.0.0.1",
			"port": 9090
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	// Overridden values
	if cfg.Game.ElementAdvantageMultiplier != 2.0 {
		t.Errorf("ElementAdvantageMultiplier = %v, want 2.0", cfg.Game.ElementAdvantageMultiplier)
	}
	if cfg.Game.ElementDisadvantageMultiplier != 0.5 {
		t.Errorf("ElementDisadvantageMultiplier = %v, want 0.5", cfg.Game.ElementDisadvantageMultiplier)
	}
	if cfg.Game.CritDamageMultiplier != 2.5 {
		t.Errorf("CritDamageMultiplier = %v, want 2.5", cfg.Game.CritDamageMultiplier)
	}
	if cfg.Game.MatchmakingRange != 5 {
		t.Errorf("MatchmakingRange = %d, want 5", cfg.Game.MatchmakingRange)
	}
	if cfg.Game.MatchmakingTimeout != 120 {
		t.Errorf("MatchmakingTimeout = %d, want 120", cfg.Game.MatchmakingTimeout)
	}
	if cfg.Game.MaxCooldownReduction != 2 {
		t.Errorf("MaxCooldownReduction = %d, want 2", cfg.Game.MaxCooldownReduction)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %q, want '127.0.0.1'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}

	// Non-overridden fields stay at default
	if cfg.Game.ArenaMatchScoreRange != 100 {
		t.Errorf("ArenaMatchScoreRange should remain default 100, got %d", cfg.Game.ArenaMatchScoreRange)
	}
	if cfg.Game.SeasonDurationDays != 30 {
		t.Errorf("SeasonDurationDays should remain default 30, got %d", cfg.Game.SeasonDurationDays)
	}
	if cfg.DataPath.Monsters != "internal/data/monsters.json" {
		t.Errorf("DataPath.Monsters should remain default, got %q", cfg.DataPath.Monsters)
	}
}

// ---------------------------------------------------------------------------
// Load — invalid JSON returns error
// ---------------------------------------------------------------------------

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad_config.json")

	content := `{ this is not valid json }`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
	if cfg != nil {
		t.Error("expected nil config for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// GameConfig field-level sanity
// ---------------------------------------------------------------------------

func TestGameConfig_ElementMultipliers(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Game.ElementAdvantageMultiplier <= 0 {
		t.Error("ElementAdvantageMultiplier must be positive")
	}
	if cfg.Game.ElementDisadvantageMultiplier <= 0 {
		t.Error("ElementDisadvantageMultiplier must be positive")
	}
	if cfg.Game.ElementAdvantageMultiplier < cfg.Game.ElementDisadvantageMultiplier {
		t.Error("advantage multiplier should normally exceed disadvantage multiplier")
	}
}

func TestGameConfig_CritDamage(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Game.CritDamageMultiplier < 1.0 {
		t.Errorf("CritDamageMultiplier = %v, should be at least 1.0", cfg.Game.CritDamageMultiplier)
	}
}

func TestGameConfig_Matchmaking(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Game.MatchmakingRange < 1 {
		t.Errorf("MatchmakingRange = %d, should be >= 1", cfg.Game.MatchmakingRange)
	}
	if cfg.Game.MatchmakingTimeout < 1 {
		t.Errorf("MatchmakingTimeout = %d, should be >= 1", cfg.Game.MatchmakingTimeout)
	}
}

func TestGameConfig_CooldownReduction(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Game.MaxCooldownReduction < 0 {
		t.Errorf("MaxCooldownReduction = %d, should be >= 0", cfg.Game.MaxCooldownReduction)
	}
}

func TestGameConfig_Arena(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Game.ArenaMatchScoreRange <= 0 {
		t.Errorf("ArenaMatchScoreRange = %d, should be > 0", cfg.Game.ArenaMatchScoreRange)
	}
	if cfg.Game.ArenaMatchExpandSeconds <= 0 {
		t.Errorf("ArenaMatchExpandSeconds = %d, should be > 0", cfg.Game.ArenaMatchExpandSeconds)
	}
	if cfg.Game.SeasonDurationDays <= 0 {
		t.Errorf("SeasonDurationDays = %d, should be > 0", cfg.Game.SeasonDurationDays)
	}
}
