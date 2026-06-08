package engine

import (
	"testing"

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
		expected int64
	}{
		{"normal: 100 atk vs 50 def", 100, 50, 75},
		{"zero defense", 100, 0, 100},
		{"zero attack", 0, 50, 1},  // floor at 1
		{"high defense nullifies attack", 50, 200, 1},  // floor at 1
		{"both zero", 0, 0, 1},  // floor at 1
		{"large values", 1_000_000, 500_000, 750_000},
		{"attack exactly half of defense", 100, 200, 1},  // floor at 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBaseDamage(tt.attack, tt.defense)
			if got != tt.expected {
				t.Errorf("CalculateBaseDamage(%d, %d) = %d; want %d",
					tt.attack, tt.defense, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateCritMultiplier
// ---------------------------------------------------------------------------

func TestCalculateCritMultiplier(t *testing.T) {
	tests := []struct {
		name   string
		isCrit bool
		want   float64
	}{
		{"crit: doubles damage", true, 2.0},
		{"no crit: unchanged", false, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCritMultiplier(tt.isCrit)
			if got != tt.want {
				t.Errorf("CalculateCritMultiplier(%v) = %v; want %v",
					tt.isCrit, got, tt.want)
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
		{"same-metal (0,0)", 0, 0, 1.0},
		{"same-wood (1,1)", 1, 1, 1.0},
		{"same-water (2,2)", 2, 2, 1.0},
		{"same-fire (3,3)", 3, 3, 1.0},
		{"same-earth (4,4)", 4, 4, 1.0},
		{"metal->wood (0,1)", 0, 1, 1.3},
		{"wood->earth (1,4)", 1, 4, 1.3},
		{"water->fire (2,3)", 2, 3, 1.3},
		{"fire->metal (3,0)", 3, 0, 1.3},
		{"earth->water (4,2)", 4, 2, 1.3},
		{"metal<-water (0,2)", 0, 2, 0.7},
		{"wood<-metal (1,0)", 1, 0, 0.7},
		{"water<-earth (2,4)", 2, 4, 0.7},
		{"fire<-water (3,2)", 3, 2, 0.7},
		{"earth<-wood (4,1)", 4, 1, 0.7},
		{"neutral metal-fire (0,3)", 0, 3, 1.0},
		{"invalid negative attacker", -1, 0, 1.0},
		{"invalid too-high attacker", 5, 0, 1.0},
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
			atkElem:      0, defElem: 1,
			playerRealm:  5, monsterRealm: 5,
			isCrit: false,
			want: 146,
		},
		{
			name: "with crit",
			attack: 100, defense: 50,
			skill:        Skill{DamageMultiplier: 1.5},
			atkElem:      0, defElem: 1,
			playerRealm:  5, monsterRealm: 5,
			isCrit: true,
			want: 293,
		},
		{
			name: "realm advantage +3",
			attack: 100, defense: 50,
			skill:        Skill{DamageMultiplier: 1.0},
			atkElem:      0, defElem: 0,
			playerRealm:  8, monsterRealm: 5,
			isCrit: false,
			want: 86,
		},
		{
			name: "realm disadvantage -3",
			attack: 100, defense: 50,
			skill:        Skill{DamageMultiplier: 1.0},
			atkElem:      0, defElem: 0,
			playerRealm:  5, monsterRealm: 8,
			isCrit: false,
			want: 64,
		},
		{
			name: "min damage floor",
			attack: 1, defense: 100,
			skill:        Skill{DamageMultiplier: 1.0},
			atkElem:      0, defElem: 0,
			playerRealm:  5, monsterRealm: 5,
			isCrit: false,
			want: 1,
		},
		{
			name: "all factors max",
			attack: 1000, defense: 100,
			skill:        Skill{DamageMultiplier: 2.0},
			atkElem:      0, defElem: 1,
			playerRealm:  10, monsterRealm: 5,
			isCrit: true,
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
		atk    int64
		power  float64
		want   int64
	}{
		{"normal heal", 100, 0.5, 50},
		{"full heal multiplier", 200, 1.0, 200},
		{"zero power defaults to 1.0", 150, 0, 150},
		{"zero attack", 0, 0.5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fighter := &model.Fighter{Attack: tt.atk}
			skill := &model.Skill{Power: tt.power}
			got := CalculateHeal(fighter, skill)
			if got != tt.want {
				t.Errorf("CalculateHeal() = %d; want %d", got, tt.want)
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
		monsterLv int
		atk       int64
		def       int64
		partySize int
		want      int
	}{
		{"solo kill, low level", 1, 10, 5, 1, 52},
		{"solo kill, medium level", 10, 50, 30, 1, 508},
		{"party of 2", 10, 50, 30, 2, 559},
		{"party of 4", 10, 50, 30, 4, 660},
		{"high level monster", 50, 200, 100, 1, 2530},
		{"zero stats", 0, 0, 0, 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateExpReward(tt.monsterLv, tt.atk, tt.def, tt.partySize)
			if got != tt.want {
				t.Errorf("CalculateExpReward(%d, %d, %d, %d) = %d; want %d",
					tt.monsterLv, tt.atk, tt.def, tt.partySize, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateSpeedOrder
// ---------------------------------------------------------------------------

func TestCalculateSpeedOrder(t *testing.T) {
	tests := []struct {
		name  string
		speeds []int64
	}{
		{"three fighters descending", []int64{100, 80, 60}},
		{"same speed", []int64{50, 50}},
		{"single fighter", []int64{100}},
		{"two equal, one different", []int64{80, 80, 60}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fighters := make([]*model.Fighter, len(tt.speeds))
			for i, s := range tt.speeds {
				fighters[i] = &model.Fighter{Speed: s}
			}
			order := CalculateSpeedOrder(fighters)
			if len(order) != len(tt.speeds) {
				t.Fatalf("expected %d results, got %d", len(tt.speeds), len(order))
			}
		})
	}
}
