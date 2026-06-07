package model

import (
	"testing"
)

func TestSubStageDefinition(t *testing.T) {
	t.Run("create sub stage with all fields", func(t *testing.T) {
		s := SubStage{
			Level:       1,
			Name:        "练气一层",
			RequiredExp: 0,
			BaseAttack:  10,
			BaseDefense: 5,
			BaseHP:      100,
		}
		if s.Level != 1 {
			t.Errorf("expected Level 1, got %d", s.Level)
		}
		if s.Name != "练气一层" {
			t.Errorf("expected Name 练气一层, got %s", s.Name)
		}
		if s.RequiredExp != 0 {
			t.Errorf("expected RequiredExp 0, got %d", s.RequiredExp)
		}
		if s.BaseAttack != 10 {
			t.Errorf("expected BaseAttack 10, got %d", s.BaseAttack)
		}
		if s.BaseDefense != 5 {
			t.Errorf("expected BaseDefense 5, got %d", s.BaseDefense)
		}
		if s.BaseHP != 100 {
			t.Errorf("expected BaseHP 100, got %d", s.BaseHP)
		}
	})

	t.Run("sub stage with non-zero required exp", func(t *testing.T) {
		s := SubStage{
			Level:       2,
			Name:        "练气二层",
			RequiredExp: 100,
			BaseAttack:  15,
			BaseDefense: 8,
			BaseHP:      150,
		}
		if s.RequiredExp != 100 {
			t.Errorf("expected RequiredExp 100, got %d", s.RequiredExp)
		}
	})
}

func TestRealmDefinition(t *testing.T) {
	t.Run("create realm with sub stages", func(t *testing.T) {
		r := Realm{
			ID:   1,
			Name: "练气",
			SubStages: []SubStage{
				{Level: 1, Name: "练气一层", RequiredExp: 0, BaseAttack: 10, BaseDefense: 5, BaseHP: 100},
				{Level: 2, Name: "练气二层", RequiredExp: 100, BaseAttack: 15, BaseDefense: 8, BaseHP: 150},
				{Level: 3, Name: "练气三层", RequiredExp: 300, BaseAttack: 20, BaseDefense: 10, BaseHP: 200},
			},
			HasTribulation:      false,
			TribulationBaseRate: 0.0,
			TribulationDamage:   0.0,
			ElementBonus:        0.0,
			BaseSpeed:           1.0,
		}
		if r.ID != 1 {
			t.Errorf("expected ID 1, got %d", r.ID)
		}
		if r.Name != "练气" {
			t.Errorf("expected Name 练气, got %s", r.Name)
		}
		if len(r.SubStages) != 3 {
			t.Errorf("expected 3 sub stages, got %d", len(r.SubStages))
		}
		if r.SubStages[2].Name != "练气三层" {
			t.Errorf("expected last sub stage 练气三层, got %s", r.SubStages[2].Name)
		}
	})

	t.Run("realm with tribulation", func(t *testing.T) {
		r := Realm{
			ID:                  9,
			Name:                "渡劫",
			SubStages:           []SubStage{{Level: 1, Name: "渡劫一层", RequiredExp: 99999, BaseAttack: 1000, BaseDefense: 500, BaseHP: 10000}},
			HasTribulation:      true,
			TribulationBaseRate: 0.3,
			TribulationDamage:   0.8,
			ElementBonus:        0.5,
			BaseSpeed:           5.0,
		}
		if !r.HasTribulation {
			t.Error("expected HasTribulation to be true")
		}
		if r.TribulationBaseRate != 0.3 {
			t.Errorf("expected TribulationBaseRate 0.3, got %f", r.TribulationBaseRate)
		}
		if r.TribulationDamage != 0.8 {
			t.Errorf("expected TribulationDamage 0.8, got %f", r.TribulationDamage)
		}
		if r.ElementBonus != 0.5 {
			t.Errorf("expected ElementBonus 0.5, got %f", r.ElementBonus)
		}
		if r.BaseSpeed != 5.0 {
			t.Errorf("expected BaseSpeed 5.0, got %f", r.BaseSpeed)
		}
	})
}

func TestBreakthroughEvent(t *testing.T) {
	t.Run("create breakthrough event", func(t *testing.T) {
		e := BreakthroughEvent{
			PlayerID:   12345,
			NewRealmID: 2,
		}
		if e.PlayerID != 12345 {
			t.Errorf("expected PlayerID 12345, got %d", e.PlayerID)
		}
		if e.NewRealmID != 2 {
			t.Errorf("expected NewRealmID 2, got %d", e.NewRealmID)
		}
	})

	t.Run("breakthrough event with zero values", func(t *testing.T) {
		e := BreakthroughEvent{}
		if e.PlayerID != 0 {
			t.Errorf("expected PlayerID 0, got %d", e.PlayerID)
		}
		if e.NewRealmID != 0 {
			t.Errorf("expected NewRealmID 0, got %d", e.NewRealmID)
		}
	})
}

func TestBreakthroughResult(t *testing.T) {
	t.Run("successful breakthrough result", func(t *testing.T) {
		r := BreakthroughResult{
			Success:       true,
			FinalRate:     0.75,
			NewRealmID:    2,
			NewRealmLevel: 1,
			LuckCost:      20,
		}
		if !r.Success {
			t.Error("expected Success to be true")
		}
		if r.FinalRate != 0.75 {
			t.Errorf("expected FinalRate 0.75, got %f", r.FinalRate)
		}
		if r.NewRealmID != 2 {
			t.Errorf("expected NewRealmID 2, got %d", r.NewRealmID)
		}
		if r.NewRealmLevel != 1 {
			t.Errorf("expected NewRealmLevel 1, got %d", r.NewRealmLevel)
		}
	})

	t.Run("failed breakthrough with heart demon", func(t *testing.T) {
		r := BreakthroughResult{
			Success:   false,
			FinalRate: 0.3,
			ExpLoss:   500,
			HeartDemon: &HeartDemon{
				ID:        1,
				Name:      "贪欲之魔",
				Scenario:  "test scenario",
				Options:   []string{"A", "B", "C"},
				KarmaCost: 30,
				Damage:    500,
			},
		}
		if r.Success {
			t.Error("expected Success to be false")
		}
		if r.HeartDemon == nil {
			t.Fatal("expected HeartDemon to be non-nil")
		}
		if r.HeartDemon.ID != 1 {
			t.Errorf("expected HeartDemon ID 1, got %d", r.HeartDemon.ID)
		}
		if r.HeartDemon.Name != "贪欲之魔" {
			t.Errorf("expected HeartDemon Name 贪欲之魔, got %s", r.HeartDemon.Name)
		}
		if len(r.HeartDemon.Options) != 3 {
			t.Errorf("expected 3 options, got %d", len(r.HeartDemon.Options))
		}
		if r.HeartDemon.KarmaCost != 30 {
			t.Errorf("expected KarmaCost 30, got %d", r.HeartDemon.KarmaCost)
		}
		if r.HeartDemon.Damage != 500 {
			t.Errorf("expected Damage 500, got %d", r.HeartDemon.Damage)
		}
	})
}

func TestTribulationResult(t *testing.T) {
	t.Run("tribulation triggered and survived", func(t *testing.T) {
		tr := TribulationResult{
			Triggered:     true,
			Success:       true,
			Rate:          0.5,
			Damage:        1000,
			Survived:      true,
			ThunderCount:  9,
			ThunderPassed: 6,
		}
		if !tr.Triggered {
			t.Error("expected Triggered to be true")
		}
		if !tr.Survived {
			t.Error("expected Survived to be true")
		}
		if tr.ThunderCount != 9 {
			t.Errorf("expected ThunderCount 9, got %d", tr.ThunderCount)
		}
		if tr.ThunderPassed != 6 {
			t.Errorf("expected ThunderPassed 6, got %d", tr.ThunderPassed)
		}
	})
}

func TestHeartDemonScenario(t *testing.T) {
	t.Run("create heart demon scenario", func(t *testing.T) {
		s := HeartDemonScenario{
			ID:        1,
			Name:      "贪欲之魔",
			Scenario:  "test scenario",
			OptionA:   "拒绝诱惑",
			OptionB:   "交换部分修为",
			OptionC:   "全部交换",
			KarmaCost: 30,
			Damage:    500,
		}
		if s.ID != 1 {
			t.Errorf("expected ID 1, got %d", s.ID)
		}
		if s.OptionA != "拒绝诱惑" {
			t.Errorf("expected OptionA 拒绝诱惑, got %s", s.OptionA)
		}
	})
}

func TestProgressionValidation(t *testing.T) {
	t.Run("valid realm progression: RequiredExp increases with level", func(t *testing.T) {
		stages := []SubStage{
			{Level: 1, RequiredExp: 0},
			{Level: 2, RequiredExp: 100},
			{Level: 3, RequiredExp: 300},
			{Level: 4, RequiredExp: 600},
		}
		for i := 1; i < len(stages); i++ {
			if stages[i].RequiredExp <= stages[i-1].RequiredExp {
				t.Errorf("RequiredExp should increase: stage %d (%d) <= stage %d (%d)",
					i, stages[i].RequiredExp, i-1, stages[i-1].RequiredExp)
			}
		}
	})

	t.Run("valid realm progression: BaseAttack increases with level", func(t *testing.T) {
		stages := []SubStage{
			{Level: 1, BaseAttack: 10},
			{Level: 2, BaseAttack: 15},
			{Level: 3, BaseAttack: 20},
		}
		for i := 1; i < len(stages); i++ {
			if stages[i].BaseAttack <= stages[i-1].BaseAttack {
				t.Errorf("BaseAttack should increase: stage %d (%d) <= stage %d (%d)",
					i, stages[i].BaseAttack, i-1, stages[i-1].BaseAttack)
			}
		}
	})

	t.Run("realm ID should be positive", func(t *testing.T) {
		realms := []Realm{
			{ID: 1, Name: "练气"},
			{ID: 2, Name: "筑基"},
			{ID: 3, Name: "金丹"},
		}
		for _, r := range realms {
			if r.ID <= 0 {
				t.Errorf("Realm ID %d should be positive for %s", r.ID, r.Name)
			}
		}
	})
}
