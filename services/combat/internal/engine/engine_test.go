package engine

import (
	"strings"
	"testing"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/model"
)

// createTestFighter is a helper that builds a Fighter with deterministic stats.
// Sets both Base* and current fields so the fighter is immediately usable.
func createTestFighter(id, name string, fType model.FighterType, element model.ElementType, level int, atk, def, hp, speed float64) *model.Fighter {
	f := model.NewFighter(id, name, fType, element, level)
	f.BaseAttack = atk
	f.Attack = atk
	f.BaseDefense = def
	f.Defense = def
	f.BaseHP = hp
	f.HP = hp
	f.BaseMaxHP = hp
	f.MaxHP = hp
	f.BaseSpeed = speed
	f.Speed = speed
	f.CritRate = 0 // deterministic
	return f
}

// ---------------------------------------------------------------------------
// Battle creation
// ---------------------------------------------------------------------------

func TestNewBattle(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Player", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	enemy := createTestFighter("e1", "Enemy", model.FighterTypeMonster, model.ElementWood, 10, 10, 10, 100, 50)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)

	if battle.ID == "" {
		t.Error("battle ID must not be empty")
	}
	if battle.State != BattleStateInit {
		t.Errorf("expected state Init, got %s", battle.State)
	}
	if battle.MaxTurns != 30 {
		t.Errorf("expected MaxTurns 30, got %d", battle.MaxTurns)
	}
	if len(battle.PlayerTeam) != 1 {
		t.Errorf("expected 1 player, got %d", len(battle.PlayerTeam))
	}
	if len(battle.EnemyTeam) != 1 {
		t.Errorf("expected 1 enemy, got %d", len(battle.EnemyTeam))
	}
	if battle.IsPVP {
		t.Error("PVE battle must not be marked PVP")
	}
	if battle.TurnNumber != 0 {
		t.Errorf("expected TurnNumber 0, got %d", battle.TurnNumber)
	}
	if len(battle.TurnLogs) != 0 {
		t.Errorf("expected empty TurnLogs, got %d entries", len(battle.TurnLogs))
	}
}

func TestNewPVPBattle(t *testing.T) {
	cfg := config.DefaultConfig()
	p1 := createTestFighter("p1", "Player1", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	p2 := createTestFighter("p2", "Player2", model.FighterTypePlayer, model.ElementWood, 10, 100, 50, 1000, 100)

	battle := NewPVPBattle("player1", "player2", []*model.Fighter{p1}, []*model.Fighter{p2}, &cfg.Game)

	if !battle.IsPVP {
		t.Error("PVP battle must have IsPVP true")
	}
	if battle.Player1ID != "player1" {
		t.Errorf("expected Player1ID 'player1', got '%s'", battle.Player1ID)
	}
	if battle.Player2ID != "player2" {
		t.Errorf("expected Player2ID 'player2', got '%s'", battle.Player2ID)
	}
}

// ---------------------------------------------------------------------------
// Battle flow: player win
// ---------------------------------------------------------------------------

func TestBattle_PlayerWin(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 1000, 100, 1000, 100)
	enemy := createTestFighter("e1", "Slime", model.FighterTypeMonster, model.ElementWood, 10, 10, 10, 100, 50)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	result := battle.Start()

	if result.State != BattleStatePlayerWin {
		t.Errorf("expected PlayerWin, got %s", result.State)
	}
	if result.Rewards == nil {
		t.Fatal("expected non-nil rewards on player win")
	}
	if result.Rewards.Exp <= 0 {
		t.Errorf("expected positive exp reward, got %d", result.Rewards.Exp)
	}
	if result.Rewards.Gold <= 0 {
		t.Errorf("expected positive gold reward, got %d", result.Rewards.Gold)
	}
	if result.TotalTurns < 1 {
		t.Errorf("expected at least 1 turn, got %d", result.TotalTurns)
	}
	if len(result.TurnLogs) < 1 {
		t.Error("expected at least 1 turn log")
	}
}

// ---------------------------------------------------------------------------
// Battle flow: enemy win
// ---------------------------------------------------------------------------

func TestBattle_EnemyWin(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 10, 10, 100, 50)
	enemy := createTestFighter("e1", "Boss", model.FighterTypeMonster, model.ElementWood, 10, 1000, 100, 1000, 100)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	result := battle.Start()

	if result.State != BattleStateEnemyWin {
		t.Errorf("expected EnemyWin, got %s", result.State)
	}
	if result.Rewards != nil {
		t.Error("expected nil rewards on enemy win")
	}
}

// ---------------------------------------------------------------------------
// Battle flow: timeout draw
// ---------------------------------------------------------------------------

func TestBattle_Draw(t *testing.T) {
	// Both sides deal only 1 damage per hit — cannot kill each other in 30 turns.
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 1, 100, 10000, 50)
	enemy := createTestFighter("e1", "Slime", model.FighterTypeMonster, model.ElementWood, 10, 1, 100, 10000, 50)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	result := battle.Start()

	if result.State != BattleStateDraw {
		t.Errorf("expected Draw, got %s", result.State)
	}
	if result.TotalTurns != 30 {
		t.Errorf("expected exactly 30 turns (max), got %d", result.TotalTurns)
	}
	if result.Rewards != nil {
		t.Error("expected nil rewards on draw")
	}
}

// ---------------------------------------------------------------------------
// OnComplete callback
// ---------------------------------------------------------------------------

func TestBattle_OnComplete(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 1000, 100, 1000, 100)
	enemy := createTestFighter("e1", "Slime", model.FighterTypeMonster, model.ElementWood, 10, 10, 10, 100, 50)

	var called bool
	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	battle.OnComplete = func(result *BattleResult) {
		called = true
		if result.State != BattleStatePlayerWin {
			t.Errorf("callback expected PlayerWin, got %s", result.State)
		}
		if result.BattleID != battle.ID {
			t.Errorf("callback BattleID mismatch: %s vs %s", result.BattleID, battle.ID)
		}
	}

	battle.Start()
	if !called {
		t.Fatal("OnComplete callback was not invoked")
	}
}

// ---------------------------------------------------------------------------
// BuildResult
// ---------------------------------------------------------------------------

func TestBattle_BuildResult(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 1000, 100, 1000, 100)
	enemy := createTestFighter("e1", "Slime", model.FighterTypeMonster, model.ElementWood, 10, 10, 10, 100, 50)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	battle.State = BattleStatePlayerWin
	battle.TurnNumber = 5

	result := battle.BuildResult()

	if result.BattleID != battle.ID {
		t.Errorf("expected BattleID %s, got %s", battle.ID, result.BattleID)
	}
	if result.State != BattleStatePlayerWin {
		t.Errorf("expected PlayerWin, got %s", result.State)
	}
	if result.TotalTurns != 5 {
		t.Errorf("expected 5 turns, got %d", result.TotalTurns)
	}
	if result.Rewards == nil {
		t.Fatal("expected rewards for player win")
	}
	if result.Rewards.Exp <= 0 {
		t.Errorf("expected positive exp, got %d", result.Rewards.Exp)
	}
}

// ---------------------------------------------------------------------------
// PVP ProcessTurnAction
// ---------------------------------------------------------------------------

func TestBattle_ProcessTurnAction(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 1000, 100, 1000, 100)
	enemy := createTestFighter("e1", "Slime", model.FighterTypeMonster, model.ElementWood, 10, 10, 10, 100, 50)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	battle.State = BattleStateRunning

	actions := map[string]*TurnAction{
		player.ID: {FighterID: player.ID, ActionType: "normal_attack"},
	}

	turnResult := battle.ProcessTurnAction(actions)
	if turnResult == nil {
		t.Fatal("expected non-nil turn result")
	}
	if turnResult.TurnNumber != 1 {
		t.Errorf("expected turn number 1, got %d", turnResult.TurnNumber)
	}

	if enemy.IsAlive() {
		t.Error("enemy should be dead after player normal attack")
	}
	if battle.State != BattleStatePlayerWin {
		t.Errorf("expected PlayerWin, got %s", battle.State)
	}
}

func TestBattle_ProcessTurnAction_NotRunning(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	enemy := createTestFighter("e1", "Slime", model.FighterTypeMonster, model.ElementWood, 10, 100, 50, 1000, 100)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	battle.State = BattleStateDraw

	result := battle.ProcessTurnAction(nil)
	if result != nil {
		t.Error("expected nil when battle state is not Running")
	}
}

// ---------------------------------------------------------------------------
// GetBattleResult
// ---------------------------------------------------------------------------

func TestGetBattleResult(t *testing.T) {
	t.Run("both alive", func(t *testing.T) {
		p := createTestFighter("p1", "P", model.FighterTypePlayer, model.ElementMetal, 1, 100, 50, 1000, 100)
		e := createTestFighter("e1", "E", model.FighterTypeMonster, model.ElementWood, 1, 100, 50, 1000, 100)
		pAlive, eAlive := GetBattleResult([]*model.Fighter{p}, []*model.Fighter{e})
		if !pAlive || !eAlive {
			t.Errorf("expected both alive, got player=%v enemy=%v", pAlive, eAlive)
		}
	})

	t.Run("player dead", func(t *testing.T) {
		p := createTestFighter("p1", "P", model.FighterTypePlayer, model.ElementMetal, 1, 100, 50, 0, 100)
		p.Status = model.StatusDead
		e := createTestFighter("e1", "E", model.FighterTypeMonster, model.ElementWood, 1, 100, 50, 1000, 100)
		pAlive, eAlive := GetBattleResult([]*model.Fighter{p}, []*model.Fighter{e})
		if pAlive {
			t.Error("expected player dead")
		}
		if !eAlive {
			t.Error("expected enemy alive")
		}
	})

	t.Run("enemy dead", func(t *testing.T) {
		p := createTestFighter("p1", "P", model.FighterTypePlayer, model.ElementMetal, 1, 100, 50, 1000, 100)
		e := createTestFighter("e1", "E", model.FighterTypeMonster, model.ElementWood, 1, 100, 50, 0, 100)
		e.Status = model.StatusDead
		pAlive, eAlive := GetBattleResult([]*model.Fighter{p}, []*model.Fighter{e})
		if !pAlive {
			t.Error("expected player alive")
		}
		if eAlive {
			t.Error("expected enemy dead")
		}
	})

	t.Run("both dead", func(t *testing.T) {
		p := createTestFighter("p1", "P", model.FighterTypePlayer, model.ElementMetal, 1, 100, 50, 0, 100)
		p.Status = model.StatusDead
		e := createTestFighter("e1", "E", model.FighterTypeMonster, model.ElementWood, 1, 100, 50, 0, 100)
		e.Status = model.StatusDead
		pAlive, eAlive := GetBattleResult([]*model.Fighter{p}, []*model.Fighter{e})
		if pAlive || eAlive {
			t.Errorf("expected both dead, got player=%v enemy=%v", pAlive, eAlive)
		}
	})

	t.Run("empty team", func(t *testing.T) {
		pAlive, eAlive := GetBattleResult([]*model.Fighter{}, []*model.Fighter{})
		if pAlive || eAlive {
			t.Errorf("expected false for empty, got player=%v enemy=%v", pAlive, eAlive)
		}
	})
}

// ---------------------------------------------------------------------------
// GetSummary
// ---------------------------------------------------------------------------

func TestBattle_GetSummary(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	enemy := createTestFighter("e1", "Slime", model.FighterTypeMonster, model.ElementWood, 5, 50, 30, 500, 50)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	summary := battle.GetSummary()

	if !strings.Contains(summary, "Hero") {
		t.Error("summary should contain player name")
	}
	if !strings.Contains(summary, "Slime") {
		t.Error("summary should contain enemy name")
	}
	if !strings.Contains(summary, battle.ID) {
		t.Error("summary should contain battle ID")
	}
}

// ---------------------------------------------------------------------------
// ExecuteSkill (engine-level integration)
// ---------------------------------------------------------------------------

func TestExecuteSkill(t *testing.T) {
	cfg := &config.GameConfig{
		ElementAdvantageMultiplier:   1.3,
		ElementDisadvantageMultiplier: 0.7,
	}

	caster := createTestFighter("p1", "Mage", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	caster.ApplyPassiveStats()
	caster.ResetBattleStats()

	target := createTestFighter("e1", "Goblin", model.FighterTypeMonster, model.ElementWood, 10, 10, 10, 500, 50)
	target.ApplyPassiveStats()
	target.ResetBattleStats()

	skill := &model.Skill{
		ID:         "fireball",
		Name:       "Fireball",
		Type:       model.SkillTypeActive,
		Element:    model.ElementMetal,
		TargetType: model.TargetSingleEnemy,
		Power:      1.5,
	}

	result := ExecuteSkill(caster, skill, nil, []*model.Fighter{target}, cfg)

	if result == nil {
		t.Fatal("expected non-nil SkillResult")
	}
	if result.Skill.ID != "fireball" {
		t.Errorf("expected skill ID 'fireball', got '%s'", result.Skill.ID)
	}
	if len(result.Targets) == 0 {
		t.Fatal("expected at least one target result")
	}
	if result.Targets[0].Damage <= 0 {
		t.Errorf("expected positive damage, got %v", result.Targets[0].Damage)
	}
	if result.Targets[0].TargetID != "e1" {
		t.Errorf("expected target ID 'e1', got '%s'", result.Targets[0].TargetID)
	}
}

func TestExecuteSkill_Healing(t *testing.T) {
	cfg := &config.GameConfig{
		ElementAdvantageMultiplier:   1.3,
		ElementDisadvantageMultiplier: 0.7,
	}

	healer := createTestFighter("p1", "Cleric", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	healer.ApplyPassiveStats()
	healer.ResetBattleStats()

	ally := createTestFighter("p2", "Warrior", model.FighterTypePlayer, model.ElementEarth, 10, 80, 80, 1000, 60)
	ally.ApplyPassiveStats()
	ally.ResetBattleStats()
	ally.TakeDamage(300) // HP is now 700

	enemies := []*model.Fighter{
		createTestFighter("e1", "Orc", model.FighterTypeMonster, model.ElementFire, 10, 50, 30, 500, 40),
	}

	healSkill := &model.Skill{
		ID:         "heal",
		Name:       "Heal",
		Type:       model.SkillTypeActive,
		Element:    model.ElementMetal,
		TargetType: model.TargetSingleAlly,
		Power:      -0.5, // negative indicates healing
	}

	result := ExecuteSkill(healer, healSkill, []*model.Fighter{healer, ally}, enemies, cfg)

	if result == nil {
		t.Fatal("expected non-nil SkillResult")
	}
	if len(result.Targets) == 0 {
		t.Fatal("expected at least one target")
	}
	if result.Targets[0].Heal <= 0 {
		t.Errorf("expected positive heal, got %v", result.Targets[0].Heal)
	}
}

// ---------------------------------------------------------------------------
// ProcessBuffEffects
// ---------------------------------------------------------------------------

func TestProcessBuffEffects_Damage(t *testing.T) {
	fighter := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	fighter.ApplyPassiveStats()
	fighter.ResetBattleStats()

	fighter.Buffs = append(fighter.Buffs, &model.Buff{
		Type:   model.BuffTypeDamage,
		Name:   "Bleed",
		Value:  10,
		Stacks: 1,
	})

	initialHP := fighter.HP
	logs := ProcessBuffEffects(fighter)

	if len(logs) == 0 {
		t.Error("expected log entries from damage buff")
	}
	if fighter.HP >= initialHP {
		t.Errorf("expected HP to decrease from %v, got %v", initialHP, fighter.HP)
	}
}

func TestProcessBuffEffects_Heal(t *testing.T) {
	fighter := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	fighter.ApplyPassiveStats()
	fighter.ResetBattleStats()
	fighter.TakeDamage(300) // HP = 700

	fighter.Buffs = append(fighter.Buffs, &model.Buff{
		Type:   model.BuffTypeHeal,
		Name:   "Regen",
		Value:  50,
		Stacks: 1,
	})

	initialHP := fighter.HP
	logs := ProcessBuffEffects(fighter)

	if len(logs) == 0 {
		t.Error("expected log entries from heal buff")
	}
	if fighter.HP <= initialHP {
		t.Errorf("expected HP to increase from %v, got %v", initialHP, fighter.HP)
	}
}

func TestProcessBuffEffects_DeadFighter(t *testing.T) {
	fighter := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 0, 100)
	fighter.Status = model.StatusDead

	fighter.Buffs = append(fighter.Buffs, &model.Buff{
		Type:   model.BuffTypeDamage,
		Name:   "Bleed",
		Value:  10,
		Stacks: 1,
	})

	logs := ProcessBuffEffects(fighter)
	if len(logs) != 0 {
		t.Error("expected no log entries for dead fighter")
	}
}

func TestProcessBuffEffects_StacksMultiplier(t *testing.T) {
	fighter := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	fighter.ApplyPassiveStats()
	fighter.ResetBattleStats()

	// 3 stacks of damage buff, each worth 10 -> 30 damage per tick
	fighter.Buffs = append(fighter.Buffs, &model.Buff{
		Type:   model.BuffTypeDamage,
		Name:   "Poison",
		Value:  10,
		Stacks: 3,
	})

	initialHP := fighter.HP
	ProcessBuffEffects(fighter)

	expectedLoss := 10.0 * 3.0 // Value * Stacks
	if fighter.HP != initialHP-expectedLoss {
		t.Errorf("expected HP %v (lost %v), got %v", initialHP-expectedLoss, expectedLoss, fighter.HP)
	}
}

// ---------------------------------------------------------------------------
// ReduceCooldowns
// ---------------------------------------------------------------------------

func TestReduceCooldowns(t *testing.T) {
	fighter := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	fighter.Skills = []*model.Skill{
		{ID: "skill1", CurrentCD: 3, Cooldown: 5},
		{ID: "skill2", CurrentCD: 0, Cooldown: 2}, // already off cooldown
	}

	ReduceCooldowns(fighter, 1)

	if fighter.Skills[0].CurrentCD != 2 {
		t.Errorf("expected skill1 CD 2, got %d", fighter.Skills[0].CurrentCD)
	}
	if fighter.Skills[1].CurrentCD != 0 {
		t.Errorf("expected skill2 CD 0, got %d", fighter.Skills[1].CurrentCD)
	}

	// Reduce below zero — should clamp to 0
	ReduceCooldowns(fighter, 5)
	if fighter.Skills[0].CurrentCD != 0 {
		t.Errorf("expected skill1 CD 0 after clamping, got %d", fighter.Skills[0].CurrentCD)
	}
	if fighter.Skills[1].CurrentCD != 0 {
		t.Errorf("expected skill2 CD 0 after clamping, got %d", fighter.Skills[1].CurrentCD)
	}
}

// ---------------------------------------------------------------------------
// isPlayerTeam (unexported — tested indirectly via ProcessTurnAction)
// ---------------------------------------------------------------------------

func TestIsPlayerTeam(t *testing.T) {
	p1 := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 1, 100, 50, 1000, 100)
	p2 := createTestFighter("p2", "Ally", model.FighterTypePlayer, model.ElementWood, 1, 100, 50, 1000, 100)
	enemy := createTestFighter("e1", "Enemy", model.FighterTypeMonster, model.ElementWater, 1, 100, 50, 1000, 100)

	team := []*model.Fighter{p1, p2}

	if !isPlayerTeam(p1, team) {
		t.Error("p1 should be identified as player team")
	}
	if !isPlayerTeam(p2, team) {
		t.Error("p2 should be identified as player team")
	}
	if isPlayerTeam(enemy, team) {
		t.Error("enemy should NOT be identified as player team")
	}
}

// ---------------------------------------------------------------------------
// calculateRewards (unexported)
// ---------------------------------------------------------------------------

func TestCalculateRewards(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	enemy := createTestFighter("e1", "Slime", model.FighterTypeMonster, model.ElementWood, 10, 50, 30, 500, 30)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	battle.State = BattleStatePlayerWin

	rewards := battle.calculateRewards()
	if rewards == nil {
		t.Fatal("expected non-nil rewards")
	}
	// Exp = CalculateExpReward(10, 50, 30, 1) = 10*50 + 80*0.1 = 508
	if rewards.Exp != 508 {
		t.Errorf("expected exp 508, got %d", rewards.Exp)
	}
	// Gold = level * 10 = 100
	if rewards.Gold != 100 {
		t.Errorf("expected gold 100, got %d", rewards.Gold)
	}
}

func TestCalculateRewards_NoHumanoidRewards(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	// Fighter type is Player, not Monster — no rewards should be generated
	humanoid := createTestFighter("e1", "Rival", model.FighterTypePlayer, model.ElementWood, 10, 100, 50, 1000, 50)

	battle := NewBattle([]*model.Fighter{player}, []*model.Fighter{humanoid}, &cfg.Game)
	battle.State = BattleStatePlayerWin

	rewards := battle.calculateRewards()
	if rewards == nil {
		t.Fatal("expected non-nil rewards")
	}
	// No monster-type enemies -> no experience reward
	if rewards.Exp != 0 {
		t.Errorf("expected 0 exp for non-monster enemy, got %d", rewards.Exp)
	}
	// No gold for PVP-style enemies
	if rewards.Gold != 0 {
		t.Errorf("expected 0 gold for non-monster enemy, got %d", rewards.Gold)
	}
}

func TestCalculateRewards_PVPNoRewards(t *testing.T) {
	cfg := config.DefaultConfig()
	player := createTestFighter("p1", "Hero", model.FighterTypePlayer, model.ElementMetal, 10, 100, 50, 1000, 100)
	enemy := createTestFighter("e1", "Slime", model.FighterTypeMonster, model.ElementWood, 10, 50, 30, 500, 30)

	battle := NewPVPBattle("player1", "player2", []*model.Fighter{player}, []*model.Fighter{enemy}, &cfg.Game)
	battle.State = BattleStatePlayerWin

	rewards := battle.calculateRewards()
	if rewards == nil {
		t.Fatal("expected non-nil rewards")
	}
	// Exp is still calculated (monster types are checked separately)
	// But PVP battles skip gold drops
	if rewards.Exp != 508 {
		t.Errorf("expected exp 508, got %d", rewards.Exp)
	}
	if rewards.Gold != 0 {
		t.Errorf("expected 0 gold for PVP battle, got %d", rewards.Gold)
	}
}
