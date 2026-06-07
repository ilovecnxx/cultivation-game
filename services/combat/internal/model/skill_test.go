package model

import (
	"testing"
)

// ---------------------------------------------------------------------------
// SkillType constants
// ---------------------------------------------------------------------------

func TestSkillTypeValues(t *testing.T) {
	tests := []struct {
		skillType SkillType
		expected  string
	}{
		{SkillTypeActive, "active"},
		{SkillTypePassive, "passive"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.skillType) != tt.expected {
				t.Errorf("SkillType = %q, want %q", tt.skillType, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SkillTarget constants
// ---------------------------------------------------------------------------

func TestSkillTargetValues(t *testing.T) {
	tests := []struct {
		target   SkillTarget
		expected string
	}{
		{TargetSingleEnemy, "single_enemy"},
		{TargetAllEnemy, "all_enemy"},
		{TargetSelf, "self"},
		{TargetSingleAlly, "single_ally"},
		{TargetAllAlly, "all_ally"},
		{TargetRandomEnemy, "random_enemy"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.target) != tt.expected {
				t.Errorf("SkillTarget = %q, want %q", tt.target, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Skill creation
// ---------------------------------------------------------------------------

func TestSkill_Creation(t *testing.T) {
	skill := &Skill{
		ID:          "fireball_1",
		Name:        "Fireball",
		Type:        SkillTypeActive,
		Element:     ElementFire,
		TargetType:  TargetSingleEnemy,
		Power:       2.5,
		Cost:        20,
		Cooldown:    3,
		CurrentCD:   0,
		Description: "A powerful fire spell",
		Level:       2,
	}

	if skill.ID != "fireball_1" {
		t.Errorf("ID = %s, want 'fireball_1'", skill.ID)
	}
	if skill.Name != "Fireball" {
		t.Errorf("Name = %s, want 'Fireball'", skill.Name)
	}
	if skill.Type != SkillTypeActive {
		t.Errorf("Type = %s, want 'active'", skill.Type)
	}
	if skill.Element != ElementFire {
		t.Errorf("Element = %s, want 'fire'", skill.Element)
	}
	if skill.TargetType != TargetSingleEnemy {
		t.Errorf("TargetType = %s, want 'single_enemy'", skill.TargetType)
	}
	if skill.Power != 2.5 {
		t.Errorf("Power = %v, want 2.5", skill.Power)
	}
	if skill.Cost != 20 {
		t.Errorf("Cost = %d, want 20", skill.Cost)
	}
	if skill.Cooldown != 3 {
		t.Errorf("Cooldown = %d, want 3", skill.Cooldown)
	}
	if skill.CurrentCD != 0 {
		t.Errorf("CurrentCD = %d, want 0", skill.CurrentCD)
	}
	if skill.Description != "A powerful fire spell" {
		t.Errorf("Description = %s, want 'A powerful fire spell'", skill.Description)
	}
	if skill.Level != 2 {
		t.Errorf("Level = %d, want 2", skill.Level)
	}
}

// ---------------------------------------------------------------------------
// PassiveSkill
// ---------------------------------------------------------------------------

func TestPassiveSkill_Stats(t *testing.T) {
	stats := &PassiveStats{
		AttackBonus:  0.2,
		DefenseBonus: 0.15,
		SpeedBonus:   0.1,
		HpBonus:      0.5,
		CritRate:     0.05,
		CritDamage:   0.25,
	}

	skill := &Skill{
		ID:           "passive_str",
		Name:         "Strength of the Ox",
		Type:         SkillTypePassive,
		PassiveStats: stats,
	}

	if skill.Type != SkillTypePassive {
		t.Errorf("Type = %s, want 'passive'", skill.Type)
	}
	if skill.PassiveStats.AttackBonus != 0.2 {
		t.Errorf("AttackBonus = %v, want 0.2", skill.PassiveStats.AttackBonus)
	}
	if skill.PassiveStats.DefenseBonus != 0.15 {
		t.Errorf("DefenseBonus = %v, want 0.15", skill.PassiveStats.DefenseBonus)
	}
	if skill.PassiveStats.SpeedBonus != 0.1 {
		t.Errorf("SpeedBonus = %v, want 0.1", skill.PassiveStats.SpeedBonus)
	}
	if skill.PassiveStats.HpBonus != 0.5 {
		t.Errorf("HpBonus = %v, want 0.5", skill.PassiveStats.HpBonus)
	}
	if skill.PassiveStats.CritRate != 0.05 {
		t.Errorf("CritRate = %v, want 0.05", skill.PassiveStats.CritRate)
	}
	if skill.PassiveStats.CritDamage != 0.25 {
		t.Errorf("CritDamage = %v, want 0.25", skill.PassiveStats.CritDamage)
	}
}

// ---------------------------------------------------------------------------
// BuffEffectConfig embedded in Skill
// ---------------------------------------------------------------------------

func TestSkill_BuffEffectConfig(t *testing.T) {
	skill := &Skill{
		ID: "buff_skill",
		Buffs: []BuffEffectConfig{
			{
				Type:     BuffTypeAttack,
				Name:     "Attack Up",
				Value:    50,
				Duration: 3,
				Chance:   1.0,
				IsDebuff: false,
				Effect:   BuffEffectInstant,
			},
			{
				Type:     BuffTypeDamage,
				Name:     "Burn",
				Value:    15,
				Duration: 3,
				Chance:   0.5,
				IsDebuff: true,
				Effect:   BuffEffectOnTurn,
			},
		},
	}

	if len(skill.Buffs) != 2 {
		t.Fatalf("expected 2 buff configs, got %d", len(skill.Buffs))
	}

	buff0 := skill.Buffs[0]
	if buff0.Type != BuffTypeAttack {
		t.Errorf("buff[0].Type = %s, want 'attack'", buff0.Type)
	}
	if buff0.Chance != 1.0 {
		t.Errorf("buff[0].Chance = %v, want 1.0", buff0.Chance)
	}
	if buff0.IsDebuff {
		t.Error("buff[0] should not be a debuff")
	}

	buff1 := skill.Buffs[1]
	if buff1.Type != BuffTypeDamage {
		t.Errorf("buff[1].Type = %s, want 'damage'", buff1.Type)
	}
	if buff1.Chance != 0.5 {
		t.Errorf("buff[1].Chance = %v, want 0.5", buff1.Chance)
	}
	if !buff1.IsDebuff {
		t.Error("buff[1] should be a debuff")
	}
}

// ---------------------------------------------------------------------------
// Special skill flags
// ---------------------------------------------------------------------------

func TestSkill_SpecialFlags(t *testing.T) {
	skill := &Skill{
		ID:            "pierce",
		Name:          "Armor Pierce",
		IgnoreDefense: true,
		LifeSteal:     0.3,
	}

	if !skill.IgnoreDefense {
		t.Error("expected IgnoreDefense = true")
	}
	if skill.LifeSteal != 0.3 {
		t.Errorf("LifeSteal = %v, want 0.3", skill.LifeSteal)
	}
}

// ---------------------------------------------------------------------------
// ElementType constants (re-used from fighter context, tested here for coverage)
// ---------------------------------------------------------------------------

func TestElementTypeValues(t *testing.T) {
	tests := []struct {
		elem     ElementType
		expected string
	}{
		{ElementMetal, "metal"},
		{ElementWood, "wood"},
		{ElementWater, "water"},
		{ElementFire, "fire"},
		{ElementEarth, "earth"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.elem) != tt.expected {
				t.Errorf("ElementType = %q, want %q", tt.elem, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FighterType constants
// ---------------------------------------------------------------------------

func TestFighterTypeValues(t *testing.T) {
	tests := []struct {
		fType    FighterType
		expected string
	}{
		{FighterTypePlayer, "player"},
		{FighterTypeMonster, "monster"},
		{FighterTypePet, "pet"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.fType) != tt.expected {
				t.Errorf("FighterType = %q, want %q", tt.fType, tt.expected)
			}
		})
	}
}

func TestFighterStatusValues(t *testing.T) {
	tests := []struct {
		status   FighterStatus
		expected string
	}{
		{StatusAlive, "alive"},
		{StatusDead, "dead"},
		{StatusStunned, "stunned"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("FighterStatus = %q, want %q", tt.status, tt.expected)
			}
		})
	}
}
