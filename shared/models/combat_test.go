package models

import (
	"testing"
	"time"
)

func TestDamageTypeValues(t *testing.T) {
	tests := []struct {
		name string
		dt   DamageType
		want int32
	}{
		{"Physical", DamageTypePhysical, 0},
		{"Magical", DamageTypeMagical, 1},
		{"True", DamageTypeTrue, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.dt) != tt.want {
				t.Errorf("DamageType = %d, want %d", int32(tt.dt), tt.want)
			}
		})
	}
}

func TestSkillTargetValues(t *testing.T) {
	tests := []struct {
		name string
		st   SkillTarget
		want int32
	}{
		{"SingleEnemy", SkillTargetSingleEnemy, 0},
		{"AllEnemy", SkillTargetAllEnemy, 1},
		{"Self", SkillTargetSelf, 2},
		{"SingleAlly", SkillTargetSingleAlly, 3},
		{"AllAlly", SkillTargetAllAlly, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.st) != tt.want {
				t.Errorf("SkillTarget = %d, want %d", int32(tt.st), tt.want)
			}
		})
	}
}

func TestSkillStructure(t *testing.T) {
	s := Skill{
		ID:           1001,
		Name:         "烈焰斩",
		DamageType:   DamageTypeMagical,
		Target:       SkillTargetSingleEnemy,
		BaseDamage:   500,
		Coefficient:  1.5,
		AttrScale:    "spirit",
		Cooldown:     3,
		CostMana:     50,
		CritBonus:    0.5,
		LevelRequire: 3,
		Description:  "猛烈的火焰攻击",
	}
	if s.ID != 1001 || s.Name != "烈焰斩" || s.BaseDamage != 500 {
		t.Errorf("Skill field mismatch: %+v", s)
	}
	if s.Coefficient != 1.5 {
		t.Errorf("Coefficient = %f, want 1.5", s.Coefficient)
	}
}

func TestFighter_ApplyDamage_Physical(t *testing.T) {
	f := &Fighter{
		HP:    1000,
		MaxHP: 1000,
		Attr: PlayerAttribute{
			Defense: 200,
		},
	}
	// damage 500, reduction = 200/10 = 20, actual = 480
	actual := f.ApplyDamage(500, DamageTypePhysical)
	if actual != 480 {
		t.Errorf("ApplyDamage returned %d, want 480", actual)
	}
	if f.HP != 520 {
		t.Errorf("HP = %d, want 520", f.HP)
	}
}

func TestFighter_ApplyDamage_Magical(t *testing.T) {
	f := &Fighter{
		HP:    1000,
		MaxHP: 1000,
		Attr: PlayerAttribute{
			Spirit: 300,
		},
	}
	// damage 500, reduction = 300/10 = 30, actual = 470
	actual := f.ApplyDamage(500, DamageTypeMagical)
	if actual != 470 {
		t.Errorf("ApplyDamage returned %d, want 470", actual)
	}
	if f.HP != 530 {
		t.Errorf("HP = %d, want 530", f.HP)
	}
}

func TestFighter_ApplyDamage_True(t *testing.T) {
	f := &Fighter{
		HP:    1000,
		MaxHP: 1000,
		Attr: PlayerAttribute{
			Defense: 99999,
			Spirit:  99999,
		},
	}
	// True damage ignores all reduction
	actual := f.ApplyDamage(500, DamageTypeTrue)
	if actual != 500 {
		t.Errorf("ApplyDamage returned %d, want 500", actual)
	}
	if f.HP != 500 {
		t.Errorf("HP = %d, want 500", f.HP)
	}
}

func TestFighter_ApplyDamage_Zero(t *testing.T) {
	f := &Fighter{
		HP:    1000,
		MaxHP: 1000,
	}
	actual := f.ApplyDamage(0, DamageTypePhysical)
	if actual != 0 {
		t.Errorf("ApplyDamage(0) returned %d, want 0", actual)
	}
	if f.HP != 1000 {
		t.Errorf("HP should remain 1000, got %d", f.HP)
	}
	actual = f.ApplyDamage(-100, DamageTypePhysical)
	if actual != 0 {
		t.Errorf("ApplyDamage(-100) returned %d, want 0", actual)
	}
}

func TestFighter_ApplyDamage_MinimumOne(t *testing.T) {
	f := &Fighter{
		HP:    100,
		MaxHP: 100,
		Attr: PlayerAttribute{
			Defense: 1000, // reduction = 100
		},
	}
	// damage 50, reduction 100 >= 50, so actual = 1
	actual := f.ApplyDamage(50, DamageTypePhysical)
	if actual != 1 {
		t.Errorf("ApplyDamage returned %d, want 1 (minimum)", actual)
	}
	if f.HP != 99 {
		t.Errorf("HP = %d, want 99", f.HP)
	}
}

func TestFighter_ApplyDamage_Fatal(t *testing.T) {
	f := &Fighter{
		HP:    100,
		MaxHP: 100,
		Attr: PlayerAttribute{
			Defense: 0,
		},
	}
	actual := f.ApplyDamage(1000, DamageTypePhysical)
	if actual != 100 {
		t.Errorf("ApplyDamage returned %d, want 100 (clamped to HP)", actual)
	}
	if f.HP != 0 {
		t.Errorf("HP = %d, want 0", f.HP)
	}
	if f.IsAlive() {
		t.Error("Fighter should be dead after fatal damage")
	}
}

func TestFighter_ApplyDamage_ExactDeath(t *testing.T) {
	f := &Fighter{
		HP:    100,
		MaxHP: 100,
		Attr: PlayerAttribute{
			Defense: 0,
		},
	}
	actual := f.ApplyDamage(100, DamageTypePhysical)
	if actual != 100 {
		t.Errorf("ApplyDamage returned %d, want 100", actual)
	}
	if f.HP != 0 {
		t.Errorf("HP = %d, want 0", f.HP)
	}
}

func TestFighter_Heal_Normal(t *testing.T) {
	f := &Fighter{
		HP:    500,
		MaxHP: 1000,
	}
	actual := f.Heal(300)
	if actual != 300 {
		t.Errorf("Heal returned %d, want 300", actual)
	}
	if f.HP != 800 {
		t.Errorf("HP = %d, want 800", f.HP)
	}
}

func TestFighter_Heal_Overheal(t *testing.T) {
	f := &Fighter{
		HP:    800,
		MaxHP: 1000,
	}
	actual := f.Heal(500)
	if actual != 200 {
		t.Errorf("Heal returned %d, want 200 (clamped)", actual)
	}
	if f.HP != 1000 {
		t.Errorf("HP = %d, want 1000 (max)", f.HP)
	}
}

func TestFighter_Heal_FullHP(t *testing.T) {
	f := &Fighter{
		HP:    1000,
		MaxHP: 1000,
	}
	actual := f.Heal(500)
	if actual != 0 {
		t.Errorf("Heal at full HP returned %d, want 0", actual)
	}
	if f.HP != 1000 {
		t.Errorf("HP should remain 1000, got %d", f.HP)
	}
}

func TestFighter_Heal_Zero(t *testing.T) {
	f := &Fighter{
		HP:    500,
		MaxHP: 1000,
	}
	actual := f.Heal(0)
	if actual != 0 {
		t.Errorf("Heal(0) returned %d, want 0", actual)
	}
	actual = f.Heal(-50)
	if actual != 0 {
		t.Errorf("Heal(-50) returned %d, want 0", actual)
	}
}

func TestFighter_IsAlive(t *testing.T) {
	tests := []struct {
		name string
		hp   int64
		alive bool
	}{
		{"Positive HP", 100, true},
		{"Zero HP", 0, false},
		{"Negative HP", -10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Fighter{HP: tt.hp}
			if got := f.IsAlive(); got != tt.alive {
				t.Errorf("IsAlive() = %v, want %v", got, tt.alive)
			}
		})
	}
}

func TestFighter_CanUseSkill(t *testing.T) {
	skill := &Skill{ID: 1, CostMana: 100}

	tests := []struct {
		name string
		mp   int64
		cooldowns map[uint32]uint32
		want bool
	}{
		{"Enough MP, no cooldown", 200, nil, true},
		{"Exactly enough MP", 100, nil, true},
		{"Not enough MP", 50, nil, false},
		{"On cooldown", 200, map[uint32]uint32{1: 2}, false},
		{"Cooldown expired", 200, map[uint32]uint32{1: 0}, true},
		{"Different skill on cooldown", 200, map[uint32]uint32{2: 1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Fighter{MP: tt.mp, Cooldowns: tt.cooldowns}
			if got := f.CanUseSkill(skill); got != tt.want {
				t.Errorf("CanUseSkill() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFighter_BuffsDirectManipulation(t *testing.T) {
	f := &Fighter{}
	if len(f.Buffs) != 0 {
		t.Error("New fighter should have no buffs")
	}

	// Add buffs via direct slice manipulation
	f.Buffs = append(f.Buffs, Buff{
		ID:            1,
		Name:          "攻击增强",
		RemainRounds:  3,
		AttrMod:       PlayerAttribute{Strength: 50},
		DamagePerRound: 0,
	})
	f.Buffs = append(f.Buffs, Buff{
		ID:            2,
		Name:          "中毒",
		RemainRounds:  5,
		DamagePerRound: 30,
	})

	if len(f.Buffs) != 2 {
		t.Errorf("Expected 2 buffs, got %d", len(f.Buffs))
	}

	// HasBuff check
	hasBuff1 := false
	hasBuff3 := false
	for _, b := range f.Buffs {
		if b.ID == 1 {
			hasBuff1 = true
		}
		if b.ID == 3 {
			hasBuff3 = true
		}
	}
	if !hasBuff1 {
		t.Error("Expected buff ID 1 to be present")
	}
	if hasBuff3 {
		t.Error("Buff ID 3 should not be present")
	}

	// Count stacks
	stackCount := 0
	for _, b := range f.Buffs {
		if b.ID == 1 {
			stackCount++
		}
	}
	if stackCount != 1 {
		t.Errorf("Expected 1 stack of buff ID 1, got %d", stackCount)
	}

	// Remove buff by filtering
	var remaining []Buff
	for _, b := range f.Buffs {
		if b.ID != 1 {
			remaining = append(remaining, b)
		}
	}
	f.Buffs = remaining
	if len(f.Buffs) != 1 {
		t.Errorf("Expected 1 buff after removal, got %d", len(f.Buffs))
	}
	if f.Buffs[0].ID != 2 {
		t.Errorf("Remaining buff should be ID 2, got %d", f.Buffs[0].ID)
	}
}

func TestBuffStructure(t *testing.T) {
	b := Buff{
		ID:             10,
		Name:           "铁骨",
		RemainRounds:   5,
		AttrMod:        PlayerAttribute{Defense: 100},
		DamagePerRound: 0,
		HealPerRound:   10,
	}
	if b.ID != 10 || b.RemainRounds != 5 || b.HealPerRound != 10 {
		t.Errorf("Buff field mismatch: %+v", b)
	}
}

func TestCombatResultStructure(t *testing.T) {
	now := time.Now()
	cr := CombatResult{
		CombatID:    "abc-123",
		AttackerID:  1,
		DefenderID:  2,
		WinnerID:    1,
		TotalRounds: 5,
		Rounds: []CombatRoundResult{
			{
				RoundNum: 1,
				Actions: []CombatActionDetail{
					{
						AttackerID: 1,
						SkillID:    100,
						TargetID:   2,
						Damage:     150,
						IsCritical: true,
						Description: "暴击！造成150点伤害",
					},
				},
			},
		},
		ExpReward:   1000,
		ItemRewards: []uint32{101, 102},
		StartTime:   now,
		EndTime:     now.Add(30 * time.Second),
		DurationMS:  30000,
	}

	if cr.CombatID != "abc-123" || cr.WinnerID != 1 || cr.TotalRounds != 5 {
		t.Errorf("CombatResult fields mismatch: %+v", cr)
	}
	if len(cr.Rounds) != 1 {
		t.Errorf("Expected 1 round, got %d", len(cr.Rounds))
	}
	if len(cr.Rounds[0].Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(cr.Rounds[0].Actions))
	}
	if cr.Rounds[0].Actions[0].Damage != 150 || !cr.Rounds[0].Actions[0].IsCritical {
		t.Error("Action detail fields mismatch")
	}
	if len(cr.ItemRewards) != 2 {
		t.Errorf("Expected 2 item rewards, got %d", len(cr.ItemRewards))
	}
	if cr.DurationMS != 30000 {
		t.Errorf("DurationMS = %d, want 30000", cr.DurationMS)
	}
}

func TestCombatActionDetail_Flags(t *testing.T) {
	// Test IsDodged and IsBlocked flags
	action := CombatActionDetail{
		AttackerID: 1,
		TargetID:   2,
		Damage:     0,
		IsDodged:   true,
		IsBlocked:  false,
	}
	if !action.IsDodged {
		t.Error("IsDodged should be true")
	}
	if action.IsBlocked {
		t.Error("IsBlocked should be false")
	}
}

func TestFighter_ZeroDefense(t *testing.T) {
	f := &Fighter{
		HP:    100,
		MaxHP: 100,
		Attr:  PlayerAttribute{},
	}
	actual := f.ApplyDamage(50, DamageTypePhysical)
	if actual != 50 {
		t.Errorf("With zero defense, ApplyDamage(50) = %d, want 50", actual)
	}
	if f.HP != 50 {
		t.Errorf("HP = %d, want 50", f.HP)
	}
}

func TestFighter_ApplyDamage_PhysicalDefault(t *testing.T) {
	// DamageTypePhysical (0) should be treated as the default case (physical)
	f := &Fighter{
		HP:    200,
		MaxHP: 200,
		Attr: PlayerAttribute{
			Defense: 100, // reduction = 10
		},
	}
	actual := f.ApplyDamage(100, 0) // 0 = DamageTypePhysical
	if actual != 90 {
		t.Errorf("ApplyDamage(100, 0) = %d, want 90", actual)
	}
}
