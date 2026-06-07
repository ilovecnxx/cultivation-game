package engine

import (
	"testing"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/model"
)

// ---------------------------------------------------------------------------
// CalculateBaseDamage
// ---------------------------------------------------------------------------

func TestCalculateBaseDamage(t *testing.T) {
	tests := []struct {
		name     string
		attack   int64
		defense  int64
		expected float64
	}{
		{"normal: 100 atk vs 50 def", 100, 50, 75.0},
		{"zero defense", 100, 0, 100.0},
		{"zero attack", 0, 50, -25.0},
		{"high defense nullifies attack", 50, 200, -50.0},
		{"both zero", 0, 0, 0.0},
		{"large values", 1_000_000, 500_000, 750_000.0},
		{"attack exactly half of defense", 100, 200, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBaseDamage(tt.attack, tt.defense)
			if got != tt.expected {
				t.Errorf("CalculateBaseDamage(%d, %d) = %v; want %v",
					tt.attack, tt.defense, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateCritDamage
// ---------------------------------------------------------------------------

func TestCalculateCritDamage(t *testing.T) {
	tests := []struct {
		name   string
		base   float64
		isCrit bool
		want   float64
	}{
		{"crit: doubles damage", 100.0, true, 200.0},
		{"no crit: unchanged", 100.0, false, 100.0},
		{"zero with crit", 0, true, 0},
		{"negative with crit", -50.0, true, -100.0},
		{"fractional", 33.3, true, 66.6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCritDamage(tt.base, tt.isCrit)
			if got != tt.want {
				t.Errorf("CalculateCritDamage(%v, %v) = %v; want %v",
					tt.base, tt.isCrit, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateRealmMultiplier
// ---------------------------------------------------------------------------

func TestCalculateRealmMultiplier(t *testing.T) {
	tests := []struct {
		name     string
		player   int
		monster  int
		expected float64
	}{
		{"same realm", 5, 5, 1.0},
		{"player +1", 6, 5, 1.05},
		{"player +3", 8, 5, 1.15},
		{"player +5 (cap)", 10, 5, 1.25},
		{"player +7 (capped to +5)", 12, 5, 1.25},
		{"player -1", 5, 6, 0.95},
		{"player -3", 5, 8, 0.85},
		{"player -5 (cap)", 5, 10, 0.75},
		{"player -7 (capped to -5)", 5, 12, 0.75},
		{"zero diff", 0, 0, 1.0},
		{"large equal", 100, 100, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRealmMultiplier(tt.player, tt.monster)
			if got != tt.expected {
				t.Errorf("CalculateRealmMultiplier(%d, %d) = %v; want %v",
					tt.player, tt.monster, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateElementMultiplier (integer-index version)
// ---------------------------------------------------------------------------

func TestCalculateElementMultiplier(t *testing.T) {
	tests := []struct {
		name string
		atk  int
		def  int
		want float64
	}{
		// Same element
		{"same-metal (0,0)", 0, 0, 1.0},
		{"same-wood (1,1)", 1, 1, 1.0},
		{"same-water (2,2)", 2, 2, 1.0},
		{"same-fire (3,3)", 3, 3, 1.0},
		{"same-earth (4,4)", 4, 4, 1.0},

		// Advantage (attacker controls defender per the 5x5 matrix)
		{"metal->wood (0,1)", 0, 1, 1.3},
		{"wood->earth (1,4)", 1, 4, 1.3},
		{"water->fire (2,3)", 2, 3, 1.3},
		{"fire->metal (3,0)", 3, 0, 1.3},
		{"earth->water (4,2)", 4, 2, 1.3},

		// Disadvantage (attacker is controlled by defender per the 5x5 matrix)
		{"metal<-water (0,2)", 0, 2, 0.7},
		{"wood<-metal (1,0)", 1, 0, 0.7},
		{"water<-earth (2,4)", 2, 4, 0.7},
		{"fire<-water (3,2)", 3, 2, 0.7},
		{"earth<-wood (4,1)", 4, 1, 0.7},

		// Neutral (no direct relationship in the 5x5 matrix)
		{"metal-fire (0,3)", 0, 3, 1.0},
		{"metal-earth (0,4)", 0, 4, 1.0},
		{"wood-water (1,2)", 1, 2, 1.0},
		{"wood-fire (1,3)", 1, 3, 1.0},
		{"water-metal (2,0)", 2, 0, 1.0},
		{"water-wood (2,1)", 2, 1, 1.0},
		{"fire-wood (3,1)", 3, 1, 1.0},
		{"fire-earth (3,4)", 3, 4, 1.0},
		{"earth-metal (4,0)", 4, 0, 1.0},
		{"earth-fire (4,3)", 4, 3, 1.0},

		// Out of bounds
		{"invalid negative attacker", -1, 0, 1.0},
		{"invalid too-high attacker", 5, 0, 1.0},
		{"invalid negative defender", 0, -1, 1.0},
		{"invalid too-high defender", 0, 5, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateElementMultiplier(tt.atk, tt.def)
			if got != tt.want {
				t.Errorf("CalculateElementMultiplier(%d, %d) = %v; want %v",
					tt.atk, tt.def, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateFinalDamage (all factors combined)
// ---------------------------------------------------------------------------

func TestCalculateFinalDamage(t *testing.T) {
	tests := []struct {
		name         string
		attack       int64
		defense      int64
		skill        Skill
		atkElem      int
		defElem      int
		playerRealm  int
		monsterRealm int
		isCrit       bool
		want         int64
	}{
		{
			name: "normal no-crit",
			attack: 100, defense: 50,
			skill:        Skill{DamageMultiplier: 1.5},
			atkElem:      0, defElem: 1, // metal counters wood = 1.3
			playerRealm:  5, monsterRealm: 5,
			isCrit: false,
			// 75 * 1.5 * 1.3 * 1.0 * 1.0 = 146.25 -> 146
			want: 146,
		},
		{
			name: "with crit",
			attack: 100, defense: 50,
			skill:        Skill{DamageMultiplier: 1.5},
			atkElem:      0, defElem: 1,
			playerRealm:  5, monsterRealm: 5,
			isCrit: true,
			// 75 * 1.5 * 1.3 * 1.0 * 2.0 = 292.5 -> 293
			want: 293,
		},
		{
			name: "realm advantage +3",
			attack: 100, defense: 50,
			skill:        Skill{DamageMultiplier: 1.0},
			atkElem:      0, defElem: 0,
			playerRealm:  8, monsterRealm: 5,
			isCrit: false,
			// 75 * 1.0 * 1.0 * 1.15 * 1.0 = 86.25 -> 86
			want: 86,
		},
		{
			name: "realm disadvantage -3",
			attack: 100, defense: 50,
			skill:        Skill{DamageMultiplier: 1.0},
			atkElem:      0, defElem: 0,
			playerRealm:  5, monsterRealm: 8,
			isCrit: false,
			// 75 * 1.0 * 1.0 * 0.85 * 1.0 = 63.75 -> 64
			want: 64,
		},
		{
			name: "min damage floor",
			attack: 1, defense: 100,
			skill:        Skill{DamageMultiplier: 1.0},
			atkElem:      0, defElem: 0,
			playerRealm:  5, monsterRealm: 5,
			isCrit: false,
			// 1 - 50 = -49, floor at 1
			want: 1,
		},
		{
			name: "all factors max",
			attack: 1000, defense: 100,
			skill:        Skill{DamageMultiplier: 2.0},
			atkElem:      0, defElem: 1, // advantage
			playerRealm:  10, monsterRealm: 5, // +5 capped
			isCrit: true,
			// 950 * 2.0 * 1.3 * 1.25 * 2.0 = 6175
			want: 6175,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateFinalDamage(tt.attack, tt.defense, tt.skill,
				tt.atkElem, tt.defElem, tt.playerRealm, tt.monsterRealm, tt.isCrit)
			if got != tt.want {
				t.Errorf("CalculateFinalDamage(...) = %d; want %d", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateHeal
// ---------------------------------------------------------------------------

func TestCalculateHeal(t *testing.T) {
	tests := []struct {
		name   string
		attack float64
		power  float64
		want   float64
	}{
		{"positive power", 100, 1.5, 150},
		{"negative power (treated as 1.0 per code)", 100, -0.5, 100},
		{"zero power (treated as 1.0)", 100, 0, 100},
		{"power=1.0", 100, 1.0, 100},
		{"large heal", 10000, 2.0, 20000},
		{"zero attack with power", 0, 1.5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healer := &model.Fighter{Attack: tt.attack}
			skill := &model.Skill{Power: tt.power}
			got := CalculateHeal(healer, skill)
			if got != tt.want {
				t.Errorf("CalculateHeal(...) = %v; want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateExpReward
// ---------------------------------------------------------------------------

func TestCalculateExpReward(t *testing.T) {
	tests := []struct {
		name      string
		level     int
		atk, def  float64
		partySize int
		want      int
	}{
		{"single player", 10, 100, 50, 1, 515},            // 10*50 + 150*0.1 = 515
		{"party of 2", 10, 100, 50, 2, 567},               // 515 * 1.1 = 566.5 -> 567
		{"party of 5", 10, 100, 50, 5, 721},               // 515 * 1.4 = 721
		{"low level", 1, 10, 5, 1, 52},                    // 1*50 + 15*0.1 = 51.5 -> 52
		{"zero everything", 0, 0, 0, 1, 0},
		{"high level", 100, 1000, 500, 1, 5150},           // 100*50 + 1500*0.1 = 5150
		{"solo vs party of 1", 5, 20, 10, 1, 253},         // 5*50+30*0.1 = 253
		{"large party bonus", 1, 1, 1, 10, 95},            // 1*50+2*0.1=50.2, * (1+9*0.1)=1.9 -> 95.38 -> 95
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateExpReward(tt.level, tt.atk, tt.def, tt.partySize)
			if got != tt.want {
				t.Errorf("CalculateExpReward(%d, %v, %v, %d) = %d; want %d",
					tt.level, tt.atk, tt.def, tt.partySize, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateDropRate
// ---------------------------------------------------------------------------

func TestCalculateDropRate(t *testing.T) {
	tests := []struct {
		name string
		base float64
		luck int
		want float64
	}{
		{"no luck", 0.5, 0, 0.5},
		{"with luck 100", 0.5, 100, 0.55},        // 0.5 * 1.1 = 0.55
		{"high luck 500", 0.1, 500, 0.15},         // 0.1 * 1.5 = 0.15
		{"very high luck 1000", 0.2, 1000, 0.4},   // 0.2 * 2.0 = 0.4
		{"zero base rate", 0, 500, 0},
		{"100% base", 1.0, 0, 1.0},
		{"negative luck", 0.5, -100, 0.45},         // 0.5 * 0.9 = 0.45
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateDropRate(tt.base, tt.luck)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.0001 {
				t.Errorf("CalculateDropRate(%v, %d) = %v; want %v (epsilon 0.0001)",
					tt.base, tt.luck, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateSpeedOrder
// ---------------------------------------------------------------------------

func TestCalculateSpeedOrder(t *testing.T) {
	t.Run("all different speeds", func(t *testing.T) {
		fighters := []*model.Fighter{
			{ID: "slow", Speed: 10},
			{ID: "fast", Speed: 100},
			{ID: "medium", Speed: 50},
		}
		order := CalculateSpeedOrder(fighters)

		if len(order) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(order))
		}
		// Indexes: slow=0, fast=1, medium=2 -> sorted descending: [1, 2, 0]
		if order[0] != 1 {
			t.Errorf("expected fastest (idx 1) first, got %d", order[0])
		}
		if order[1] != 2 {
			t.Errorf("expected medium (idx 2) second, got %d", order[1])
		}
		if order[2] != 0 {
			t.Errorf("expected slowest (idx 0) last, got %d", order[2])
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		order := CalculateSpeedOrder([]*model.Fighter{})
		if len(order) != 0 {
			t.Errorf("expected empty, got %v", order)
		}
	})

	t.Run("single fighter", func(t *testing.T) {
		fighters := []*model.Fighter{{ID: "only", Speed: 50}}
		order := CalculateSpeedOrder(fighters)
		if len(order) != 1 || order[0] != 0 {
			t.Errorf("expected [0], got %v", order)
		}
	})

	t.Run("tied speeds all present", func(t *testing.T) {
		fighters := []*model.Fighter{
			{ID: "a", Speed: 50},
			{ID: "b", Speed: 50},
			{ID: "c", Speed: 50},
		}
		order := CalculateSpeedOrder(fighters)

		if len(order) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(order))
		}
		// All indices must appear exactly once (order may vary due to random tie-break)
		seen := make(map[int]bool)
		for _, idx := range order {
			if seen[idx] {
				t.Errorf("duplicate index %d", idx)
			}
			seen[idx] = true
		}
		for i := 0; i < 3; i++ {
			if !seen[i] {
				t.Errorf("missing fighter index %d in order %v", i, order)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// CalculateDamage (full integration)
// ---------------------------------------------------------------------------

func TestCalculateDamage(t *testing.T) {
	cfg := &config.GameConfig{
		ElementAdvantageMultiplier:   1.3,
		ElementDisadvantageMultiplier: 0.7,
	}

	t.Run("normal damage no crit", func(t *testing.T) {
		attacker := &model.Fighter{
			Attack:     100,
			CritRate:   0,
			CritDamage: 2.0,
			Level:      5,
			Element:    model.ElementMetal,
		}
		defender := &model.Fighter{
			Defense: 50,
			Element: model.ElementWood,
			Level:   5,
		}
		skill := &model.Skill{
			Power:   1.5,
			Element: model.ElementMetal,
		}

		// 75 * 1.5 * 1.3 * 1.0 * 1.0 = 146.25 -> round 146
		got := CalculateDamage(attacker, defender, skill, cfg)
		if got != 146 {
			t.Errorf("expected 146, got %v", got)
		}
	})

	t.Run("min damage floor at 1", func(t *testing.T) {
		attacker := &model.Fighter{
			Attack:     1,
			CritRate:   0,
			CritDamage: 2.0,
			Level:      5,
			Element:    model.ElementMetal,
		}
		defender := &model.Fighter{
			Defense: 100,
			Element: model.ElementMetal,
			Level:   5,
		}
		skill := &model.Skill{
			Power:   1.0,
			Element: model.ElementMetal,
		}

		got := CalculateDamage(attacker, defender, skill, cfg)
		if got != 1 {
			t.Errorf("expected min damage 1, got %v", got)
		}
	})

	t.Run("ignore defense", func(t *testing.T) {
		attacker := &model.Fighter{
			Attack:     100,
			CritRate:   0,
			CritDamage: 2.0,
			Level:      5,
			Element:    model.ElementMetal,
		}
		defender := &model.Fighter{
			Defense: 50,
			Element: model.ElementWood,
			Level:   5,
		}
		skill := &model.Skill{
			Power:         1.0,
			Element:       model.ElementMetal,
			IgnoreDefense: true,
		}

		// base=75, *1.0*1.3*1.0*1.0 = 97.5
		// defPenalty=25, *1.0*1.3*1.0*1.0 = 32.5
		// total = 97.5 + 32.5 = 130
		got := CalculateDamage(attacker, defender, skill, cfg)
		if got != 130 {
			t.Errorf("expected 130 (ignore defense), got %v", got)
		}
	})

	t.Run("realm disadvantage", func(t *testing.T) {
		attacker := &model.Fighter{
			Attack:     100,
			CritRate:   0,
			CritDamage: 2.0,
			Level:      5,
			Element:    model.ElementMetal,
		}
		defender := &model.Fighter{
			Defense: 50,
			Element: model.ElementMetal,
			Level:   10, // 5 level diff -> 0.75 realm mult
		}
		skill := &model.Skill{
			Power:   1.0,
			Element: model.ElementMetal,
		}

		// 75 * 1.0 * 1.0 * 0.75 * 1.0 = 56.25 -> round 56
		got := CalculateDamage(attacker, defender, skill, cfg)
		if got != 56 {
			t.Errorf("expected 56 (realm disadvantage), got %v", got)
		}
	})
}

// ---------------------------------------------------------------------------
// IsCrit
// ---------------------------------------------------------------------------

func TestIsCrit(t *testing.T) {
	t.Run("zero rate never crits", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			if IsCrit(0) {
				t.Fatal("IsCrit(0) returned true on iteration", i)
			}
		}
	})

	t.Run("100% rate always crits", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			if !IsCrit(1.0) {
				t.Fatal("IsCrit(1.0) returned false on iteration", i)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// FloatToStr
// ---------------------------------------------------------------------------

func TestFloatToStr(t *testing.T) {
	tests := []struct {
		val  float64
		want string
	}{
		{100.5, "100.5"},
		{100.0, "100.0"},
		{0.1, "0.1"},
		{0, "0.0"},
		{-10.5, "-10.5"},
		{3.14159, "3.1"},
		{1.0 / 3.0, "0.3"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FloatToStr(tt.val)
			if got != tt.want {
				t.Errorf("FloatToStr(%v) = %q; want %q", tt.val, got, tt.want)
			}
		})
	}
}
