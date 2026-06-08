package engine

import (
	"fmt"
	"time"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/model"
	"github.com/google/uuid"
)

// BattleState 战斗状态
type BattleState string

const (
	BattleStateInit    BattleState = "init"     // 初始化
	BattleStateRunning BattleState = "running"  // 进行中
	BattleStatePaused  BattleState = "paused"   // 暂停
	BattleStatePlayerWin BattleState = "player_win"  // 玩家胜利
	BattleStateEnemyWin  BattleState = "enemy_win"   // 敌人胜利
	BattleStateDraw   BattleState = "draw"      // 平局
)

// Battle 战斗实例
type Battle struct {
	ID          string           `json:"id"`
	State       BattleState      `json:"state"`
	TurnNumber  int              `json:"turn_number"`
	MaxTurns    int              `json:"max_turns"`    // 最大回合数, 超时平局
	PlayerTeam  []*model.Fighter `json:"player_team"`
	EnemyTeam   []*model.Fighter `json:"enemy_team"`
	TurnLogs    []*TurnResult    `json:"turn_logs"`
	Config      *config.GameConfig `json:"-"`

	// PVP 相关
	IsPVP       bool   `json:"is_pvp"`
	Player1ID   string `json:"player1_id,omitempty"`
	Player2ID   string `json:"player2_id,omitempty"`

	// 回调
	OnComplete func(result *BattleResult) `json:"-"`

	CreatedAt time.Time `json:"created_at"`
}

// BattleResult 战斗结算结果
type BattleResult struct {
	BattleID     string           `json:"battle_id"`
	State        BattleState      `json:"state"`
	TotalTurns   int              `json:"total_turns"`
	TurnLogs     []*TurnResult    `json:"turn_logs"`
	PlayerTeam   []*model.Fighter `json:"player_team"`
	EnemyTeam    []*model.Fighter `json:"enemy_team"`
	Rewards      *BattleRewards   `json:"rewards,omitempty"`
}

// BattleRewards 战斗奖励
type BattleRewards struct {
	Exp         int              `json:"exp"`
	Gold        int              `json:"gold"`
	Items       []*DropItem      `json:"items,omitempty"`
}

// DropItem 掉落物品
type DropItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Rarity   string `json:"rarity"`
}

// NewBattle 创建新的战斗实例
func NewBattle(playerTeam, enemyTeam []*model.Fighter, cfg *config.GameConfig) *Battle {
	return &Battle{
		ID:         uuid.New().String(),
		State:      BattleStateInit,
		MaxTurns:   30,
		TurnNumber: 0,
		PlayerTeam: playerTeam,
		EnemyTeam:  enemyTeam,
		TurnLogs:   make([]*TurnResult, 0),
		Config:     cfg,
		CreatedAt:  time.Now(),
	}
}

// NewPVPBattle 创建 PVP 战斗实例
func NewPVPBattle(player1ID, player2ID string, team1, team2 []*model.Fighter, cfg *config.GameConfig) *Battle {
	battle := NewBattle(team1, team2, cfg)
	battle.IsPVP = true
	battle.Player1ID = player1ID
	battle.Player2ID = player2ID
	return battle
}

// Start 开始战斗, 执行完整战斗流程
func (b *Battle) Start() *BattleResult {
	b.State = BattleStateRunning

	// 对所有参战者应用被动技能
	for _, f := range b.PlayerTeam {
		f.ApplyPassiveStats()
		f.ResetBattleStats()
	}
	for _, f := range b.EnemyTeam {
		f.ApplyPassiveStats()
		f.ResetBattleStats()
	}

	// 回合循环
	for b.TurnNumber < b.MaxTurns {
		b.TurnNumber++

		// 处理回合
		turnResult := ProcessTurn(
			b.TurnNumber,
			b.PlayerTeam,
			b.EnemyTeam,
			nil, // PVE 无玩家指令, 自动战斗
			b.Config,
		)

		b.TurnLogs = append(b.TurnLogs, turnResult)

		// 检查战斗是否结束
		playerAlive, enemyAlive := GetBattleResult(b.PlayerTeam, b.EnemyTeam)
		if !playerAlive && !enemyAlive {
			b.State = BattleStateDraw
			break
		}
		if !enemyAlive {
			b.State = BattleStatePlayerWin
			break
		}
		if !playerAlive {
			b.State = BattleStateEnemyWin
			break
		}
	}

	if b.State == BattleStateRunning {
		b.State = BattleStateDraw // 超时平局
	}

	result := b.BuildResult()
	if b.OnComplete != nil {
		b.OnComplete(result)
	}
	return result
}

// ProcessTurnAction 处理带玩家指令的回合(PVP)
func (b *Battle) ProcessTurnAction(playerActions map[string]*TurnAction) *TurnResult {
	if b.State != BattleStateRunning {
		return nil
	}

	b.TurnNumber++
	turnResult := ProcessTurn(
		b.TurnNumber,
		b.PlayerTeam,
		b.EnemyTeam,
		playerActions,
		b.Config,
	)

	b.TurnLogs = append(b.TurnLogs, turnResult)

	// 检查战斗结果
	playerAlive, enemyAlive := GetBattleResult(b.PlayerTeam, b.EnemyTeam)
	if !playerAlive && !enemyAlive {
		b.State = BattleStateDraw
	} else if !enemyAlive {
		b.State = BattleStatePlayerWin
	} else if !playerAlive {
		b.State = BattleStateEnemyWin
	} else if b.TurnNumber >= b.MaxTurns {
		b.State = BattleStateDraw
	}

	return turnResult
}

// BuildResult 构建战斗结算
func (b *Battle) BuildResult() *BattleResult {
	result := &BattleResult{
		BattleID:   b.ID,
		State:      b.State,
		TotalTurns: b.TurnNumber,
		TurnLogs:   b.TurnLogs,
		PlayerTeam: b.PlayerTeam,
		EnemyTeam:  b.EnemyTeam,
	}

	// 玩家胜利时计算奖励
	if b.State == BattleStatePlayerWin {
		result.Rewards = b.calculateRewards()
	}

	return result
}

// calculateRewards 计算战斗奖励
func (b *Battle) calculateRewards() *BattleRewards {
	rewards := &BattleRewards{
		Exp:  0,
		Gold: 0,
	}

	// 计算经验
	for _, enemy := range b.EnemyTeam {
		if enemy.Type == model.FighterTypeMonster {
			partySize := 0
			for _, p := range b.PlayerTeam {
				if p.IsAlive() {
					partySize++
				}
			}
			exp := CalculateExpReward(enemy.Level, enemy.Attack, enemy.Defense, partySize)
			rewards.Exp += exp
		}
	}

	// PVE 掉落
	if !b.IsPVP {
		for _, enemy := range b.EnemyTeam {
			if enemy.Type == model.FighterTypeMonster {
				// 简单金币掉落: 等级 * 10
				rewards.Gold += enemy.Level * 10
			}
		}
	}

	return rewards
}

// GetSummary 获取战斗概要
func (b *Battle) GetSummary() string {
	summary := fmt.Sprintf("===== 战斗 %s =====\n", b.ID)
	summary += fmt.Sprintf("状态: %s\n", b.State)
	summary += fmt.Sprintf("回合数: %d\n", b.TurnNumber)
	summary += fmt.Sprintf("创建时间: %s\n", b.CreatedAt.Format("2006-01-02 15:04:05"))

	if b.IsPVP {
		summary += fmt.Sprintf("PVP: %s vs %s\n", b.Player1ID, b.Player2ID)
	}

	// 玩家队伍状态
	summary += "\n--- 玩家队伍 ---\n"
	for _, f := range b.PlayerTeam {
		status := "存活"
		if !f.IsAlive() {
			status = "死亡"
		}
		summary += fmt.Sprintf("%s(Lv.%d) HP: %d/%d MP: %d %s\n",
			f.Name, f.Level, f.HP, f.MaxHP, f.MP, status)
	}

	// 敌人队伍状态
	summary += "\n--- 敌人队伍 ---\n"
	for _, f := range b.EnemyTeam {
		status := "存活"
		if !f.IsAlive() {
			status = "死亡"
		}
		summary += fmt.Sprintf("%s(Lv.%d) HP: %d/%d %s\n",
			f.Name, f.Level, f.HP, f.MaxHP, status)
	}

	return summary
}
