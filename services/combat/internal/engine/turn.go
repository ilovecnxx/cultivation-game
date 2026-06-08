package engine

import (
	"fmt"
	"math/rand"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/model"
)

// TurnAction 回合行动
type TurnAction struct {
	FighterID string        `json:"fighter_id"`
	SkillID   string        `json:"skill_id"`
	ActionType string       `json:"action_type"` // "skill" 或 "normal_attack"
	TargetID  string        `json:"target_id,omitempty"`
}

// TurnResult 回合执行结果
type TurnResult struct {
	TurnNumber int            `json:"turn_number"`
	Actions    []*ActionLog   `json:"actions"`
}

// ActionLog 行动日志
type ActionLog struct {
	ActorName   string          `json:"actor_name"`
	ActorID     string          `json:"actor_id"`
	ActionType  string          `json:"action_type"`
	SkillName   string          `json:"skill_name,omitempty"`
	SkillResult *SkillResult    `json:"skill_result,omitempty"`
	LogEntries  []string        `json:"log_entries"`
	IsPass      bool            `json:"is_pass"` // 是否跳过回合(眩晕等)
}

// ProcessTurn 处理一个完整的回合
//
// 流程:
//  1. 检查双方存活状态
//  2. 需要处理的目标(去除非存活)
//  3. 按速度排序出手顺序
//  4. 每个参战者执行行动(技能/普攻)
//  5. 处理 buff 持续效果(每回合伤害/治疗)
//  6. buff 持续时间减少
//  7. 技能冷却减少
func ProcessTurn(
	turnNum int,
	playerTeam, enemyTeam []*model.Fighter,
	playerActions map[string]*TurnAction,
	cfg *config.GameConfig,
) *TurnResult {
	result := &TurnResult{
		TurnNumber: turnNum,
		Actions:    make([]*ActionLog, 0),
	}

	// 收集所有活着的参战者
	allFighters := make([]*model.Fighter, 0)
	allFighters = append(allFighters, playerTeam...)
	allFighters = append(allFighters, enemyTeam...)

	aliveFighters := make([]*model.Fighter, 0)
	for _, f := range allFighters {
		if f.IsAlive() {
			aliveFighters = append(aliveFighters, f)
		}
	}

	if len(aliveFighters) == 0 {
		return result
	}

	// 按照速度排序出手顺序
	order := CalculateSpeedOrder(aliveFighters)

	// 按顺序执行行动
	for _, idx := range order {
		fighter := aliveFighters[idx]
		if !fighter.IsAlive() {
			continue
		}

		actionLog := &ActionLog{
			ActorName:  fighter.Name,
			ActorID:    fighter.ID,
			LogEntries: make([]string, 0),
		}

		// 检查是否眩晕
		if fighter.HasBuff(model.BuffTypeStun) {
			actionLog.IsPass = true
			actionLog.LogEntries = append(actionLog.LogEntries, fighter.Name+"处于眩晕状态, 跳过本回合")
			result.Actions = append(result.Actions, actionLog)
			continue
		}

		// 检查是否被沉默(只能普攻)
		isSilenced := fighter.HasBuff(model.BuffTypeSilence)

		// 决定使用的技能
		var selectedSkill *model.Skill

		if !isSilenced && len(fighter.Skills) > 0 {
			// 优先使用玩家指定的技能
			if action, ok := playerActions[fighter.ID]; ok && action.ActionType == "skill" {
				for _, s := range fighter.Skills {
					if s.ID == action.SkillID && s.CurrentCD <= 0 {
						selectedSkill = s
						break
					}
				}
			}

			// 如果没有指定或不可用, 自动选择可用技能
			if selectedSkill == nil {
				availableSkills := make([]*model.Skill, 0)
				for _, s := range fighter.Skills {
					if s.Type == model.SkillTypeActive && s.CurrentCD <= 0 {
						availableSkills = append(availableSkills, s)
					}
				}
				if len(availableSkills) > 0 {
					// AI 选择: 随机选一个
					selectedSkill = availableSkills[rand.Intn(len(availableSkills))]
				}
			}
		}

		if selectedSkill != nil {
			actionLog.SkillName = selectedSkill.Name
			actionLog.ActionType = "skill"

			// 执行技能
			var allies, enemies []*model.Fighter
			if isPlayerTeam(fighter, playerTeam) {
				allies = playerTeam
				enemies = enemyTeam
			} else {
				allies = enemyTeam
				enemies = playerTeam
			}

			skillResult := ExecuteSkill(fighter, selectedSkill, allies, enemies, cfg)
			actionLog.SkillResult = skillResult

			// 设置冷却
			selectedSkill.CurrentCD = selectedSkill.Cooldown

			// 消耗灵力
			if fighter.MP >= selectedSkill.Cost {
				fighter.MP -= selectedSkill.Cost
			}

			// 构建日志
			for _, tr := range skillResult.Targets {
				if tr.Damage > 0 {
					critTag := ""
					if tr.IsCrit {
						critTag = " [暴击]"
					}
					actionLog.LogEntries = append(actionLog.LogEntries,
						fmt.Sprintf("%s 对 %s 使用 %s, 造成 %d 点伤害%s",
							fighter.Name, tr.TargetName, selectedSkill.Name, tr.Damage, critTag))
				}
				if tr.Heal > 0 {
					actionLog.LogEntries = append(actionLog.LogEntries,
						fmt.Sprintf("%s 对 %s 使用 %s, 恢复 %d 点生命",
							fighter.Name, tr.TargetName, selectedSkill.Name, tr.Heal))
				}
				if tr.IsDead {
					actionLog.LogEntries = append(actionLog.LogEntries,
						fmt.Sprintf("%s 被 %s 击杀!", tr.TargetName, fighter.Name))
				}
				if tr.Blocked {
					actionLog.LogEntries = append(actionLog.LogEntries,
						fmt.Sprintf("%s 的攻击被 %s 的无敌效果阻挡", fighter.Name, tr.TargetName))
				}
			}
		} else {
			// 普攻
			actionLog.ActionType = "normal_attack"
			var enemies []*model.Fighter
			if isPlayerTeam(fighter, playerTeam) {
				enemies = enemyTeam
			} else {
				enemies = playerTeam
			}

			// 找第一个存活的敌人
			var target *model.Fighter
			for _, e := range enemies {
				if e.IsAlive() {
					target = e
					break
				}
			}

			if target != nil {
				// 普攻视为无属性技能, 倍率1.0
				normalSkill := &model.Skill{
					ID:      "normal_attack",
					Name:    "普攻",
					Element: fighter.Element,
					Power:   1.0,
					Type:    model.SkillTypeActive,
				}
				skillResult := ExecuteSkill(fighter, normalSkill, nil, []*model.Fighter{target}, cfg)
				actionLog.SkillResult = skillResult

				for _, tr := range skillResult.Targets {
					critTag := ""
					if tr.IsCrit {
						critTag = " [暴击]"
					}
					actionLog.LogEntries = append(actionLog.LogEntries,
						fmt.Sprintf("%s 对 %s 发起普攻, 造成 %d 点伤害%s",
							fighter.Name, tr.TargetName, tr.Damage, critTag))
					if tr.IsDead {
						actionLog.LogEntries = append(actionLog.LogEntries,
							fmt.Sprintf("%s 被 %s 击杀!", tr.TargetName, fighter.Name))
					}
				}
			}
		}

		// 处理自身 buff 效果(每回合伤害/治疗)
		buffLogs := ProcessBuffEffects(fighter)
		actionLog.LogEntries = append(actionLog.LogEntries, buffLogs...)

		result.Actions = append(result.Actions, actionLog)

		// 检查是否战斗结束
		if isBattleOver(playerTeam, enemyTeam) {
			break
		}
	}

	// 回合结束处理: buff 持续时间减少 & 技能冷却减少
	for _, f := range allFighters {
		if !f.IsAlive() {
			continue
		}

		// buff 过期处理
		expiredBuffs := make([]*model.Buff, 0)
		remainingBuffs := make([]*model.Buff, 0)
		for _, b := range f.Buffs {
			expired := b.Tick()
			if expired {
				expiredBuffs = append(expiredBuffs, b)
			} else {
				remainingBuffs = append(remainingBuffs, b)
			}
		}
		f.Buffs = remainingBuffs

		// 技能冷却减少
		ReduceCooldowns(f, cfg.MaxCooldownReduction)
	}

	return result
}

// isPlayerTeam 判断 fighter 是否属于玩家队伍
func isPlayerTeam(fighter *model.Fighter, playerTeam []*model.Fighter) bool {
	for _, p := range playerTeam {
		if p.ID == fighter.ID {
			return true
		}
	}
	return false
}

// isBattleOver 检查战斗是否结束
func isBattleOver(playerTeam, enemyTeam []*model.Fighter) bool {
	playerAlive := false
	for _, p := range playerTeam {
		if p.IsAlive() {
			playerAlive = true
			break
		}
	}
	enemyAlive := false
	for _, e := range enemyTeam {
		if e.IsAlive() {
			enemyAlive = true
			break
		}
	}
	return !playerAlive || !enemyAlive
}

// GetBattleResult 获取战斗结果
func GetBattleResult(playerTeam, enemyTeam []*model.Fighter) (bool, bool) {
	playerAlive := false
	for _, p := range playerTeam {
		if p.IsAlive() {
			playerAlive = true
			break
		}
	}
	enemyAlive := false
	for _, e := range enemyTeam {
		if e.IsAlive() {
			enemyAlive = true
			break
		}
	}
	return playerAlive, enemyAlive
}
