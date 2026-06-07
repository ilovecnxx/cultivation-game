package model

import (
	"testing"
)

// ---------------------------------------------------------------------------
// NewBuff
// ---------------------------------------------------------------------------

func TestNewBuff(t *testing.T) {
	b := NewBuff(BuffTypeAttack, "Power Up", 50.0, 3, false)

	if b.Type != BuffTypeAttack {
		t.Errorf("Type = %s, want 'attack'", b.Type)
	}
	if b.Name != "Power Up" {
		t.Errorf("Name = %s, want 'Power Up'", b.Name)
	}
	if b.Value != 50.0 {
		t.Errorf("Value = %v, want 50.0", b.Value)
	}
	if b.Duration != 3 {
		t.Errorf("Duration = %d, want 3", b.Duration)
	}
	if b.Remaining != 3 {
		t.Errorf("Remaining = %d, want 3", b.Remaining)
	}
	if b.Stackable {
		t.Error("NewBuff should create non-stackable buff by default")
	}
	if b.Stacks != 1 {
		t.Errorf("Stacks = %d, want 1", b.Stacks)
	}
	if b.Effect != BuffEffectOnTurn {
		t.Errorf("Effect = %s, want 'on_turn'", b.Effect)
	}
	if b.IsDebuff {
		t.Error("NewBuff should create non-debuff by default")
	}
	if b.MaxStacks != 1 {
		t.Errorf("MaxStacks = %d, want 1", b.MaxStacks)
	}
}

func TestNewBuff_Debuff(t *testing.T) {
	b := NewBuff(BuffTypeDamage, "Poison", 10.0, 3, true)

	if b.Type != BuffTypeDamage {
		t.Errorf("Type = %s, want 'damage'", b.Type)
	}
	if !b.IsDebuff {
		t.Error("expected IsDebuff = true")
	}
}

// ---------------------------------------------------------------------------
// Tick
// ---------------------------------------------------------------------------

func TestTick_NormalDecrement(t *testing.T) {
	b := NewBuff(BuffTypeAttack, "Buff", 50, 3, false)

	expired := b.Tick()
	if expired {
		t.Error("buff with Remaining=3 should not expire after 1 tick")
	}
	if b.Remaining != 2 {
		t.Errorf("Remaining = %d, want 2", b.Remaining)
	}

	b.Tick()
	if b.Remaining != 1 {
		t.Errorf("Remaining = %d, want 1", b.Remaining)
	}
}

func TestTick_ExpiresWhenRemainingReachesZero(t *testing.T) {
	b := NewBuff(BuffTypeAttack, "Buff", 50, 2, false)
	b.Tick() // Remaining = 1
	b.Tick() // Remaining = 0 -> expired

	if b.Remaining != 0 {
		t.Errorf("Remaining = %d, want 0", b.Remaining)
	}
}

func TestTick_ExpiresWhenRemainingNegative(t *testing.T) {
	b := NewBuff(BuffTypeAttack, "Buff", 50, 1, false)
	b.Tick() // Remaining = 0 -> expired

	if b.Remaining != 0 {
		t.Errorf("Remaining = %d, want 0", b.Remaining)
	}
}

func TestTick_PermanentBuff(t *testing.T) {
	b := NewBuff(BuffTypeAttack, "Permanent", 50, -1, false)

	for i := 0; i < 100; i++ {
		expired := b.Tick()
		if expired {
			t.Fatal("permanent buff (Duration=-1) should never expire")
		}
	}
	if b.Remaining != -1 {
		t.Errorf("Remaining should stay -1, got %d", b.Remaining)
	}
}

// ---------------------------------------------------------------------------
// Clone
// ---------------------------------------------------------------------------

func TestClone_Independence(t *testing.T) {
	original := NewBuff(BuffTypeAttack, "Power Up", 50.0, 3, false)
	original.Stacks = 3
	original.MaxStacks = 5
	original.FromSkillID = "skill_1"

	clone := original.Clone()

	// Modify original
	original.Value = 100
	original.Stacks = 1
	original.FromSkillID = "skill_2"

	// Clone must remain untouched
	if clone.Value != 50.0 {
		t.Errorf("clone.Value = %v, want 50.0", clone.Value)
	}
	if clone.Stacks != 3 {
		t.Errorf("clone.Stacks = %d, want 3", clone.Stacks)
	}
	if clone.FromSkillID != "skill_1" {
		t.Errorf("clone.FromSkillID = %s, want 'skill_1'", clone.FromSkillID)
	}
}

// ---------------------------------------------------------------------------
// Buff types constant validation
// ---------------------------------------------------------------------------

func TestBuffTypes(t *testing.T) {
	tests := []struct {
		buffType BuffType
		expected string
	}{
		{BuffTypeAttack, "attack"},
		{BuffTypeDefense, "defense"},
		{BuffTypeSpeed, "speed"},
		{BuffTypeHeal, "heal"},
		{BuffTypeDamage, "damage"},
		{BuffTypeStun, "stun"},
		{BuffTypeShield, "shield"},
		{BuffTypeSilence, "silence"},
		{BuffTypeInvincible, "invincible"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.buffType) != tt.expected {
				t.Errorf("BuffType = %q, want %q", tt.buffType, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BuffEffect values
// ---------------------------------------------------------------------------

func TestBuffEffects(t *testing.T) {
	tests := []struct {
		effect   BuffEffect
		expected string
	}{
		{BuffEffectInstant, "instant"},
		{BuffEffectOnTurn, "on_turn"},
		{BuffEffectOnAttack, "on_attack"},
		{BuffEffectOnDamaged, "on_damaged"},
		{BuffEffectOnExpire, "on_expire"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.effect) != tt.expected {
				t.Errorf("BuffEffect = %q, want %q", tt.effect, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Fighter buff integration (lightweight cross-checks)
// ---------------------------------------------------------------------------

func TestBuff_DurationDecreasesOnTick(t *testing.T) {
	b := NewBuff(BuffTypeHeal, "Regen", 20, 3, false)

	if b.Duration != 3 {
		t.Errorf("Duration = %d, want 3", b.Duration)
	}

	b.Tick()
	if b.Remaining != 2 {
		t.Errorf("after 1 tick: Remaining = %d, want 2", b.Remaining)
	}

	b.Tick()
	if b.Remaining != 1 {
		t.Errorf("after 2 ticks: Remaining = %d, want 1", b.Remaining)
	}

	expired := b.Tick()
	if !expired {
		t.Error("buff should be expired after 3 ticks")
	}
}

func TestBuff_ExpiredAfterDuration(t *testing.T) {
	b := NewBuff(BuffTypeAttack, "Short", 10, 1, false)
	if b.Tick() != true {
		t.Error("1-duration buff should expire after one Tick()")
	}
}

func TestBuff_ZeroDurationBuff(t *testing.T) {
	b := NewBuff(BuffTypeAttack, "Instant", 10, 0, false)
	expired := b.Tick()
	if !expired {
		t.Error("0-duration buff should expire immediately")
	}
}
