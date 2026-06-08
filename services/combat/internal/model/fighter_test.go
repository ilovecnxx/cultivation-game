package model

import (
	"testing"
)

// helper to create a pre-configured fighter for tests
func newTestFighter(id string, hp int64) *Fighter {
	f := NewFighter(id, "Test-"+id, FighterTypePlayer, ElementMetal, 10)
	f.BaseAttack = 100
	f.BaseDefense = 50
	f.BaseSpeed = 20
	f.BaseHP = hp
	f.BaseMaxHP = hp
	f.CritRate = 0
	f.ApplyPassiveStats()
	f.ResetBattleStats()
	return f
}

// ---------------------------------------------------------------------------
// IsAlive
// ---------------------------------------------------------------------------

func TestIsAlive(t *testing.T) {
	f := newTestFighter("p1", 1000)
	if !f.IsAlive() {
		t.Error("fighter should be alive when HP > 0 and Status is Alive")
	}

	f.TakeDamage(1000)
	if f.IsAlive() {
		t.Error("fighter should be dead after taking fatal damage")
	}
	if f.HP != 0 {
		t.Errorf("HP should be 0, got %v", f.HP)
	}
	if f.Status != StatusDead {
		t.Errorf("status should be Dead, got %s", f.Status)
	}
}

func TestIsAlive_ZeroHPOnCreation(t *testing.T) {
	f := NewFighter("p1", "Ghost", FighterTypePlayer, ElementMetal, 10)
	// Base fields zero — ApplyPassiveStats would set HP = 0
	f.ApplyPassiveStats()
	if f.IsAlive() {
		t.Error("fighter with 0 HP should not be alive")
	}
}

// ---------------------------------------------------------------------------
// TakeDamage
// ---------------------------------------------------------------------------

func TestTakeDamage(t *testing.T) {
	t.Run("normal damage", func(t *testing.T) {
		f := newTestFighter("p1", 1000)
		actual := f.TakeDamage(100)
		if actual != 100 {
			t.Errorf("returned damage %v, want 100", actual)
		}
		if f.HP != 900 {
			t.Errorf("HP = %v, want 900", f.HP)
		}
		if f.TotalDamageTaken != 100 {
			t.Errorf("TotalDamageTaken = %v, want 100", f.TotalDamageTaken)
		}
	})

	t.Run("exact lethal damage", func(t *testing.T) {
		f := newTestFighter("p1", 500)
		f.TakeDamage(500)
		if f.HP != 0 {
			t.Errorf("HP = %v, want 0", f.HP)
		}
		if f.Status != StatusDead {
			t.Errorf("status = %s, want Dead", f.Status)
		}
	})

	t.Run("overkill clamps to zero", func(t *testing.T) {
		f := newTestFighter("p1", 100)
		f.TakeDamage(9999)
		if f.HP != 0 {
			t.Errorf("HP = %v, want 0", f.HP)
		}
		if f.Status != StatusDead {
			t.Errorf("status = %s, want Dead", f.Status)
		}
	})

	t.Run("negative damage clamped to 0", func(t *testing.T) {
		f := newTestFighter("p1", 1000)
		hpBefore := f.HP
		actual := f.TakeDamage(-50)
		if actual != 0 {
			t.Errorf("returned damage %v, want 0", actual)
		}
		if f.HP != hpBefore {
			t.Errorf("HP changed from %v to %v", hpBefore, f.HP)
		}
	})
}

// ---------------------------------------------------------------------------
// Heal
// ---------------------------------------------------------------------------

func TestHeal(t *testing.T) {
	t.Run("normal heal", func(t *testing.T) {
		f := newTestFighter("p1", 1000)
		f.TakeDamage(300) // HP = 700

		actual := f.Heal(200)
		if actual != 200 {
			t.Errorf("returned heal %v, want 200", actual)
		}
		if f.HP != 900 {
			t.Errorf("HP = %v, want 900", f.HP)
		}
	})

	t.Run("overheal prevention", func(t *testing.T) {
		f := newTestFighter("p1", 1000)
		f.TakeDamage(100) // HP = 900

		actual := f.Heal(500) // would exceed MaxHP
		if actual != 100 {
			t.Errorf("returned heal %v, want 100", actual)
		}
		if f.HP != 1000 {
			t.Errorf("HP = %v, want 1000 (max)", f.HP)
		}
	})

	t.Run("full HP returns 0", func(t *testing.T) {
		f := newTestFighter("p1", 1000)
		actual := f.Heal(100)
		if actual != 0 {
			t.Errorf("returned heal %v, want 0", actual)
		}
		if f.HP != 1000 {
			t.Errorf("HP = %v, want 1000", f.HP)
		}
	})

	t.Run("heal on dead fighter returns 0", func(t *testing.T) {
		f := newTestFighter("p1", 1000)
		f.TakeDamage(1000) // HP = 0, Status = Dead

		actual := f.Heal(500)
		if actual != 0 {
			t.Errorf("returned heal %v, want 0", actual)
		}
		if f.HP != 0 {
			t.Errorf("HP = %v, want 0", f.HP)
		}
		if f.Status != StatusDead {
			t.Errorf("status should remain Dead after heal attempt")
		}
	})
}

// ---------------------------------------------------------------------------
// AddBuff / RemoveBuff / HasBuff / GetBuffStacks / RemoveBuffByID
// ---------------------------------------------------------------------------

func TestAddBuff_Stackable(t *testing.T) {
	f := newTestFighter("p1", 1000)

	buff := &Buff{
		Type:        BuffTypeDamage,
		Name:        "Bleed",
		Value:       10,
		Duration:    3,
		Remaining:   3,
		Stackable:   true,
		Stacks:      1,
		MaxStacks:   5,
		FromSkillID: "skill_bleed",
	}

	// First addition
	f.AddBuff(buff)
	if len(f.Buffs) != 1 {
		t.Fatalf("expected 1 buff, got %d", len(f.Buffs))
	}
	if f.Buffs[0].Stacks != 1 {
		t.Errorf("expected 1 stack, got %d", f.Buffs[0].Stacks)
	}

	// Second addition — should increment stack
	f.AddBuff(buff)
	if len(f.Buffs) != 1 {
		t.Fatalf("expected 1 buff after stacking, got %d", len(f.Buffs))
	}
	if f.Buffs[0].Stacks != 2 {
		t.Errorf("expected 2 stacks, got %d", f.Buffs[0].Stacks)
	}
	// Duration should be refreshed
	if f.Buffs[0].Remaining != 3 {
		t.Errorf("expected remaining 3 (refreshed), got %d", f.Buffs[0].Remaining)
	}
}

func TestAddBuff_StackableMaxStacks(t *testing.T) {
	f := newTestFighter("p1", 1000)

	buff := &Buff{
		Type:        BuffTypeDamage,
		Name:        "Bleed",
		Value:       10,
		Duration:    3,
		Remaining:   3,
		Stackable:   true,
		Stacks:      1,
		MaxStacks:   3,
		FromSkillID: "skill_bleed",
	}

	// Add 5 times — should cap at 3
	for i := 0; i < 5; i++ {
		f.AddBuff(buff)
	}
	if f.Buffs[0].Stacks != 3 {
		t.Errorf("expected max 3 stacks, got %d", f.Buffs[0].Stacks)
	}
}

func TestAddBuff_NonStackableReplaces(t *testing.T) {
	f := newTestFighter("p1", 1000)

	buff1 := &Buff{
		Type:        BuffTypeAttack,
		Name:        "Attack Up",
		Value:       50,
		Duration:    3,
		Remaining:   3,
		Stackable:   false,
		Stacks:      1,
		MaxStacks:   1,
		FromSkillID: "skill_buff",
	}

	buff2 := &Buff{
		Type:        BuffTypeAttack,
		Name:        "Attack Up+",
		Value:       100,
		Duration:    5,
		Remaining:   5,
		Stackable:   false,
		Stacks:      1,
		MaxStacks:   1,
		FromSkillID: "skill_buff",
	}

	f.AddBuff(buff1)
	if f.Buffs[0].Value != 50 {
		t.Errorf("expected value 50, got %v", f.Buffs[0].Value)
	}

	// Replace with stronger version
	f.AddBuff(buff2)
	if len(f.Buffs) != 1 {
		t.Fatalf("expected 1 buff after replacement, got %d", len(f.Buffs))
	}
	if f.Buffs[0].Value != 100 {
		t.Errorf("expected value 100 after replace, got %v", f.Buffs[0].Value)
	}
	if f.Buffs[0].Remaining != 5 {
		t.Errorf("expected remaining 5 after replace, got %d", f.Buffs[0].Remaining)
	}
}

func TestAddBuff_SameTypeDifferentSkillID(t *testing.T) {
	f := newTestFighter("p1", 1000)

	buff1 := &Buff{
		Type:        BuffTypeAttack,
		Name:        "Buff A",
		Duration:    3,
		Remaining:   3,
		Stackable:   false,
		FromSkillID: "skill_a",
	}
	buff2 := &Buff{
		Type:        BuffTypeAttack,
		Name:        "Buff B",
		Duration:    3,
		Remaining:   3,
		Stackable:   false,
		FromSkillID: "skill_b",
	}

	f.AddBuff(buff1)
	f.AddBuff(buff2)

	if len(f.Buffs) != 2 {
		t.Errorf("expected 2 separate buffs (different FromSkillID), got %d", len(f.Buffs))
	}
}

func TestHasBuff(t *testing.T) {
	f := newTestFighter("p1", 1000)

	if f.HasBuff(BuffTypeStun) {
		t.Error("fighter should not have stun initially")
	}

	// Use the AddBuff method (it clones, so we need to pass a properly created buff)
	stunBuff := &Buff{
		Type:        BuffTypeStun,
		Name:        "Stun",
		Duration:    2,
		Remaining:   2,
		Stackable:   false,
		FromSkillID: "skill_stun",
	}
	f.AddBuff(stunBuff)

	if !f.HasBuff(BuffTypeStun) {
		t.Error("fighter should have stun after adding")
	}
}

func TestGetBuffStacks(t *testing.T) {
	f := newTestFighter("p1", 1000)

	stacks := f.GetBuffStacks(BuffTypeShield)
	if stacks != 0 {
		t.Errorf("expected 0 stacks for missing buff, got %d", stacks)
	}

	shieldBuff := &Buff{
		Type:        BuffTypeShield,
		Name:        "Shield",
		Duration:    3,
		Remaining:   3,
		Stackable:   true,
		Stacks:      3,
		MaxStacks:   10,
		FromSkillID: "skill_shield",
	}
	f.AddBuff(shieldBuff)

	stacks = f.GetBuffStacks(BuffTypeShield)
	if stacks != 3 {
		t.Errorf("expected 3 stacks, got %d", stacks)
	}
}

func TestRemoveBuff(t *testing.T) {
	f := newTestFighter("p1", 1000)

	f.AddBuff(&Buff{Type: BuffTypeStun, Name: "Stun", Duration: 2, Remaining: 2, FromSkillID: "s1"})
	f.AddBuff(&Buff{Type: BuffTypeAttack, Name: "Atk Up", Duration: 3, Remaining: 3, FromSkillID: "s2"})

	if len(f.Buffs) != 2 {
		t.Fatalf("expected 2 buffs, got %d", len(f.Buffs))
	}

	f.RemoveBuff(BuffTypeStun)

	if len(f.Buffs) != 1 {
		t.Fatalf("expected 1 buff after removal, got %d", len(f.Buffs))
	}
	if f.Buffs[0].Type != BuffTypeAttack {
		t.Errorf("remaining buff should be Attack, got %s", f.Buffs[0].Type)
	}

	// Removing a non-existent buff should be a no-op
	f.RemoveBuff(BuffTypeSpeed)
	if len(f.Buffs) != 1 {
		t.Errorf("expected 1 buff after removing non-existent, got %d", len(f.Buffs))
	}
}

func TestRemoveBuffByID(t *testing.T) {
	f := newTestFighter("p1", 1000)

	b1 := &Buff{ID: "buff_atk", Type: BuffTypeAttack, Name: "Atk", Duration: 3, Remaining: 3, FromSkillID: "s1"}
	b2 := &Buff{ID: "buff_def", Type: BuffTypeDefense, Name: "Def", Duration: 3, Remaining: 3, FromSkillID: "s2"}
	f.AddBuff(b1)
	f.AddBuff(b2)

	if len(f.Buffs) != 2 {
		t.Fatalf("expected 2 buffs, got %d", len(f.Buffs))
	}

	// Remove by the ID assigned during AddBuff (cloned buffs)
	idToRemove := f.Buffs[0].ID
	f.RemoveBuffByID(idToRemove)

	if len(f.Buffs) != 1 {
		t.Fatalf("expected 1 buff after removal by ID, got %d", len(f.Buffs))
	}
	if f.Buffs[0].Type != BuffTypeDefense {
		t.Errorf("remaining buff should be Defense, got %s", f.Buffs[0].Type)
	}
}

func TestHasBuffByName(t *testing.T) {
	f := newTestFighter("p1", 1000)

	f.AddBuff(&Buff{Type: BuffTypeAttack, Name: "Power Surge", Duration: 3, Remaining: 3, FromSkillID: "s1"})

	if !f.HasBuffByName("Power Surge") {
		t.Error("should find buff by name 'Power Surge'")
	}
	if f.HasBuffByName("Nonexistent") {
		t.Error("should not find nonexistent buff name")
	}
}

// ---------------------------------------------------------------------------
// ResetBattleStats
// ---------------------------------------------------------------------------

func TestResetBattleStats(t *testing.T) {
	f := newTestFighter("p1", 1000)
	f.TakeDamage(300)   // TotalDamageTaken += 300
	f.Heal(100)         // TotalHealingDone += 100

	// TotalDamageDealt isn't tracked by TakeDamage/Heal — set it manually
	f.TotalDamageDealt = 500

	f.ResetBattleStats()

	if f.TotalDamageDealt != 0 {
		t.Errorf("TotalDamageDealt = %v, want 0", f.TotalDamageDealt)
	}
	if f.TotalDamageTaken != 0 {
		t.Errorf("TotalDamageTaken = %v, want 0", f.TotalDamageTaken)
	}
	if f.TotalHealingDone != 0 {
		t.Errorf("TotalHealingDone = %v, want 0", f.TotalHealingDone)
	}
}

// ---------------------------------------------------------------------------
// ApplyPassiveStats
// ---------------------------------------------------------------------------

func TestApplyPassiveStats_NoPassives(t *testing.T) {
	f := NewFighter("p1", "Test", FighterTypePlayer, ElementMetal, 10)
	f.BaseAttack = 200
	f.BaseDefense = 100
	f.BaseSpeed = 50
	f.BaseHP = 5000
	f.BaseMaxHP = 5000

	f.ApplyPassiveStats()

	if f.Attack != 200 {
		t.Errorf("Attack = %v, want 200", f.Attack)
	}
	if f.Defense != 100 {
		t.Errorf("Defense = %v, want 100", f.Defense)
	}
	if f.Speed != 50 {
		t.Errorf("Speed = %v, want 50", f.Speed)
	}
	if f.HP != 5000 {
		t.Errorf("HP = %v, want 5000", f.HP)
	}
	if f.MaxHP != 5000 {
		t.Errorf("MaxHP = %v, want 5000", f.MaxHP)
	}
}

func TestApplyPassiveStats_WithPassives(t *testing.T) {
	f := NewFighter("p1", "Test", FighterTypePlayer, ElementMetal, 10)
	f.BaseAttack = 200
	f.BaseDefense = 100
	f.BaseSpeed = 50
	f.BaseHP = 5000
	f.BaseMaxHP = 5000

	f.Passives = []*Skill{
		{
			PassiveStats: &PassiveStats{
				AttackBonus: 0.2,   // +40
				DefenseBonus: 0.1,  // +10
				SpeedBonus:  0.5,   // +25
				HpBonus:     0.5,   // +2500
				CritRate:    0.1,   // +0.1
				CritDamage:  0.3,   // +0.3
			},
		},
	}

	f.ApplyPassiveStats()

	if f.Attack != 240 { // 200 + (200*0.2)
		t.Errorf("Attack = %v, want 240", f.Attack)
	}
	if f.Defense != 110 { // 100 + (100*0.1)
		t.Errorf("Defense = %v, want 110", f.Defense)
	}
	if f.Speed != 75 { // 50 + (50*0.5)
		t.Errorf("Speed = %v, want 75", f.Speed)
	}
	if f.MaxHP != 7500 { // 5000 + (5000*0.5)
		t.Errorf("MaxHP = %v, want 7500", f.MaxHP)
	}
	if f.CritRate < 0.1499 || f.CritRate > 0.1501 {
		t.Errorf("CritRate = %v, want ~0.15", f.CritRate)
	}
	if f.CritDamage != 2.3 { // 2.0 (default) + 0.3
		t.Errorf("CritDamage = %v, want 2.3", f.CritDamage)
	}
}

func TestApplyPassiveStats_HPCapping(t *testing.T) {
	f := NewFighter("p1", "Test", FighterTypePlayer, ElementMetal, 10)
	f.BaseHP = 5000
	f.BaseMaxHP = 3000 // MaxHP smaller than HP after passives

	f.ApplyPassiveStats()

	if f.HP > f.MaxHP {
		t.Errorf("HP %v should not exceed MaxHP %v", f.HP, f.MaxHP)
	}
}

func TestApplyPassiveStats_MultiplePassives(t *testing.T) {
	f := NewFighter("p1", "Test", FighterTypePlayer, ElementMetal, 10)
	f.BaseAttack = 100

	// Two passives, each giving 10% attack
	f.Passives = []*Skill{
		{PassiveStats: &PassiveStats{AttackBonus: 0.1}},
		{PassiveStats: &PassiveStats{AttackBonus: 0.1}},
	}

	f.ApplyPassiveStats()

	// Attack = 100 + 100*0.1 + 100*0.1 = 120
	if f.Attack != 120 {
		t.Errorf("Attack = %v, want 120", f.Attack)
	}
}

func TestApplyPassiveStats_NilPassiveStats(t *testing.T) {
	f := NewFighter("p1", "Test", FighterTypePlayer, ElementMetal, 10)
	f.BaseAttack = 100
	f.Passives = []*Skill{
		{ID: "bad_skill"}, // no PassiveStats
	}

	f.ApplyPassiveStats()
	// Should still have base attack, not panic
	if f.Attack != 100 {
		t.Errorf("Attack = %v, want 100", f.Attack)
	}
}

// ---------------------------------------------------------------------------
// NewFighter defaults
// ---------------------------------------------------------------------------

func TestNewFighterDefaults(t *testing.T) {
	f := NewFighter("p1", "Hero", FighterTypePlayer, ElementFire, 5)

	if f.ID != "p1" {
		t.Errorf("ID = %s, want 'p1'", f.ID)
	}
	if f.Name != "Hero" {
		t.Errorf("Name = %s, want 'Hero'", f.Name)
	}
	if f.Type != FighterTypePlayer {
		t.Errorf("Type = %s, want 'player'", f.Type)
	}
	if f.Element != ElementFire {
		t.Errorf("Element = %s, want 'fire'", f.Element)
	}
	if f.Level != 5 {
		t.Errorf("Level = %d, want 5", f.Level)
	}
	if f.Status != StatusAlive {
		t.Errorf("Status = %s, want 'alive'", f.Status)
	}
	if f.CritRate != 0.05 {
		t.Errorf("CritRate = %v, want 0.05", f.CritRate)
	}
	if f.CritDamage != 2.0 {
		t.Errorf("CritDamage = %v, want 2.0", f.CritDamage)
	}
	if f.Buffs == nil {
		t.Error("Buffs slice should be initialized (non-nil)")
	}
	if f.Skills == nil {
		t.Error("Skills slice should be initialized (non-nil)")
	}
	if f.Passives == nil {
		t.Error("Passives slice should be initialized (non-nil)")
	}
}
