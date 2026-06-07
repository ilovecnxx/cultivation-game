package engine

import (
	"testing"

	"cultivation-game/services/combat/internal/model"
)

// TestElementCycleMap verifies the five-element cycle:
// Metal -> Wood -> Earth -> Water -> Fire -> Metal
func TestElementCycleMap(t *testing.T) {
	tests := []struct {
		attacker model.ElementType
		expected model.ElementType
	}{
		{model.ElementMetal, model.ElementWood},
		{model.ElementWood, model.ElementEarth},
		{model.ElementEarth, model.ElementWater},
		{model.ElementWater, model.ElementFire},
		{model.ElementFire, model.ElementMetal},
	}

	for _, tt := range tests {
		t.Run(string(tt.attacker)+" counters ", func(t *testing.T) {
			countered, ok := elementCycle[tt.attacker]
			if !ok {
				t.Fatalf("elementCycle missing entry for %s", tt.attacker)
			}
			if countered != tt.expected {
				t.Errorf("elementCycle[%s] = %s; want %s", tt.attacker, countered, tt.expected)
			}
		})
	}
}

// TestGetElementMultiplier tests all five advantage pairs, all five disadvantage
// pairs, same-element returns 1.0, and neutral pairs return 1.0.
func TestGetElementMultiplier(t *testing.T) {
	adv := 1.3
	disadv := 0.7

	tests := []struct {
		name     string
		attack   model.ElementType
		defense  model.ElementType
		expected float64
	}{
		// Same element = always 1.0
		{"Same-Metal", model.ElementMetal, model.ElementMetal, 1.0},
		{"Same-Wood", model.ElementWood, model.ElementWood, 1.0},
		{"Same-Earth", model.ElementEarth, model.ElementEarth, 1.0},
		{"Same-Water", model.ElementWater, model.ElementWater, 1.0},
		{"Same-Fire", model.ElementFire, model.ElementFire, 1.0},

		// Advantage pairs (attacker counters defender)
		{"Advantage-Metal-Wood", model.ElementMetal, model.ElementWood, adv},
		{"Advantage-Wood-Earth", model.ElementWood, model.ElementEarth, adv},
		{"Advantage-Earth-Water", model.ElementEarth, model.ElementWater, adv},
		{"Advantage-Water-Fire", model.ElementWater, model.ElementFire, adv},
		{"Advantage-Fire-Metal", model.ElementFire, model.ElementMetal, adv},

		// Disadvantage pairs (attacker is countered by defender)
		{"Disadvantage-Wood-Metal", model.ElementWood, model.ElementMetal, disadv},
		{"Disadvantage-Earth-Wood", model.ElementEarth, model.ElementWood, disadv},
		{"Disadvantage-Water-Earth", model.ElementWater, model.ElementEarth, disadv},
		{"Disadvantage-Fire-Water", model.ElementFire, model.ElementWater, disadv},
		{"Disadvantage-Metal-Fire", model.ElementMetal, model.ElementFire, disadv},

		// Neutral pairs (no direct counter relationship)
		{"Neutral-Metal-Water", model.ElementMetal, model.ElementWater, 1.0},
		{"Neutral-Metal-Earth", model.ElementMetal, model.ElementEarth, 1.0},
		{"Neutral-Wood-Water", model.ElementWood, model.ElementWater, 1.0},
		{"Neutral-Wood-Fire", model.ElementWood, model.ElementFire, 1.0},
		{"Neutral-Water-Metal", model.ElementWater, model.ElementMetal, 1.0},
		{"Neutral-Water-Wood", model.ElementWater, model.ElementWood, 1.0},
		{"Neutral-Fire-Wood", model.ElementFire, model.ElementWood, 1.0},
		{"Neutral-Fire-Earth", model.ElementFire, model.ElementEarth, 1.0},
		{"Neutral-Earth-Metal", model.ElementEarth, model.ElementMetal, 1.0},
		{"Neutral-Earth-Fire", model.ElementEarth, model.ElementFire, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetElementMultiplier(tt.attack, tt.defense, adv, disadv)
			if got != tt.expected {
				t.Errorf("GetElementMultiplier(%s, %s, %.1f, %.1f) = %v; want %v",
					tt.attack, tt.defense, adv, disadv, got, tt.expected)
			}
		})
	}
}

// TestGetElementMultiplier_CustomValues verifies custom advantage/disadvantage ratios.
func TestGetElementMultiplier_CustomValues(t *testing.T) {
	got := GetElementMultiplier(model.ElementMetal, model.ElementWood, 2.0, 0.5)
	if got != 2.0 {
		t.Errorf("expected custom advantage 2.0, got %v", got)
	}

	got = GetElementMultiplier(model.ElementWood, model.ElementMetal, 2.0, 0.5)
	if got != 0.5 {
		t.Errorf("expected custom disadvantage 0.5, got %v", got)
	}

	// Both 1.0 should behave as neutral for non-cycled pairs
	got = GetElementMultiplier(model.ElementMetal, model.ElementEarth, 1.0, 1.0)
	if got != 1.0 {
		t.Errorf("expected neutral 1.0, got %v", got)
	}
}

// TestIsCountered confirms IsCountered returns true when the attacker's element
// counters the defender's element.
func TestIsCountered(t *testing.T) {
	tests := []struct {
		attack  model.ElementType
		defense model.ElementType
		want    bool
	}{
		{model.ElementMetal, model.ElementWood, true},
		{model.ElementWood, model.ElementEarth, true},
		{model.ElementEarth, model.ElementWater, true},
		{model.ElementWater, model.ElementFire, true},
		{model.ElementFire, model.ElementMetal, true},

		{model.ElementWood, model.ElementMetal, false},
		{model.ElementMetal, model.ElementFire, false},
		{model.ElementWood, model.ElementWater, false},
		{model.ElementMetal, model.ElementMetal, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.attack)+"->"+string(tt.defense), func(t *testing.T) {
			got := IsCountered(tt.attack, tt.defense)
			if got != tt.want {
				t.Errorf("IsCountered(%s, %s) = %v; want %v", tt.attack, tt.defense, got, tt.want)
			}
		})
	}
}

// TestIsWeakAgainst confirms IsWeakAgainst returns true when the attacker's element
// is countered by the defender's element.
func TestIsWeakAgainst(t *testing.T) {
	tests := []struct {
		attack  model.ElementType
		defense model.ElementType
		want    bool
	}{
		{model.ElementWood, model.ElementMetal, true},
		{model.ElementEarth, model.ElementWood, true},
		{model.ElementWater, model.ElementEarth, true},
		{model.ElementFire, model.ElementWater, true},
		{model.ElementMetal, model.ElementFire, true},

		{model.ElementMetal, model.ElementWood, false},
		{model.ElementMetal, model.ElementMetal, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.attack)+"<-"+string(tt.defense), func(t *testing.T) {
			got := IsWeakAgainst(tt.attack, tt.defense)
			if got != tt.want {
				t.Errorf("IsWeakAgainst(%s, %s) = %v; want %v", tt.attack, tt.defense, got, tt.want)
			}
		})
	}
}
