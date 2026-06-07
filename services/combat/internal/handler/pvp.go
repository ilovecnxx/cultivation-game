package handler

import (
	"net/http"
	"sync"
	"time"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/engine"
	"cultivation-game/services/combat/internal/model"
	"cultivation-game/services/combat/internal/repository"
	"cultivation-game/services/combat/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// PVPHandler PVP 处理器
type PVPHandler struct {
	cfg          *config.Config
	matching     *service.MatchmakingService
	playerClient *repository.PlayerClient
	// 活跃战斗
	activeBattles sync.Map // map[string]*engine.Battle
	// 玩家当前战斗ID
	playerBattles sync.Map // map[string]string
}

// NewPVPHandler 创建 PVP 处理器
func NewPVPHandler(cfg *config.Config, playerClient *repository.PlayerClient) *PVPHandler {
	h := &PVPHandler{
		cfg:          cfg,
		playerClient: playerClient,
		matching:     service.NewMatchmakingService(cfg.Game.MatchmakingRange, time.Duration(cfg.Game.MatchmakingTimeout)*time.Second),
	}

	// 启动自动匹配协程
	h.matching.AutoMatch(5*time.Second, func(result *service.MatchResult) {
		h.startPVPBattle(result)
	})

	return h
}

// JoinQueueRequest 加入匹配队列请求
type JoinQueueRequest struct {
	PlayerID   string  `json:"player_id"`
	Name       string  `json:"name"`
	Level      int     `json:"level"`
	Rank       string  `json:"rank"`
	Score      int     `json:"score"`
	PowerLevel float64 `json:"power_level"`
}

// JoinQueue 加入匹配队列
//
// POST /api/v1/pvp/join
func (h *PVPHandler) JoinQueue(c *gin.Context) {
	var req JoinQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 检查是否已在战斗中
	if _, ok := h.playerBattles.Load(req.PlayerID); ok {
		c.JSON(http.StatusConflict, gin.H{"error": "您已在战斗中"})
		return
	}

	// 新手保护检查：PVP 保护期内不能参与匹配
	if h.playerClient != nil {
		protected, err := h.playerClient.IsPvpProtected(req.PlayerID)
		if err != nil {
			log.Warn().Err(err).Str("player_id", req.PlayerID).Msg("检查PVP保护失败，允许匹配")
		} else if protected {
			c.JSON(http.StatusForbidden, gin.H{"error": "您处于新手保护期，暂不能参与PVP"})
			return
		}
	}

	// 创建玩家档案
	profile := service.NewPlayerProfile(req.PlayerID, req.Name, req.Level, req.PowerLevel)
	profile.Rank = service.Rank(req.Rank)
	profile.Score = req.Score

	h.matching.Enqueue(profile)

	log.Info().Str("player_id", req.PlayerID).Str("name", req.Name).Msg("加入PVP匹配队列")

	c.JSON(http.StatusOK, gin.H{
		"message":    "已加入匹配队列",
		"queue_size": h.matching.QueueSize(),
	})
}

// LeaveQueue 离开匹配队列
//
// POST /api/v1/pvp/leave
func (h *PVPHandler) LeaveQueue(c *gin.Context) {
	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	h.matching.Dequeue(req.PlayerID)

	c.JSON(http.StatusOK, gin.H{
		"message": "已退出匹配队列",
	})
}

// QueueStatus 查询匹配队列状态
//
// GET /api/v1/pvp/queue-status?player_id=xxx
func (h *PVPHandler) QueueStatus(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少player_id参数"})
		return
	}

	inQueue := h.matching.IsInQueue(playerID)

	c.JSON(http.StatusOK, gin.H{
		"in_queue":   inQueue,
		"queue_size": h.matching.QueueSize(),
	})
}

// SubmitAction 提交回合行动
//
// POST /api/v1/pvp/action
func (h *PVPHandler) SubmitAction(c *gin.Context) {
	var req struct {
		PlayerID string            `json:"player_id"`
		BattleID string            `json:"battle_id"`
		Action   *engine.TurnAction `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 获取战斗
	battleInterface, ok := h.activeBattles.Load(req.BattleID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "战斗不存在或已结束"})
		return
	}
	battle := battleInterface.(*engine.Battle)

	if battle.State != engine.BattleStateRunning {
		c.JSON(http.StatusConflict, gin.H{"error": "战斗已结束"})
		return
	}

	// 处理行动
	turnResult := battle.ProcessTurnAction(map[string]*engine.TurnAction{
		req.PlayerID: req.Action,
	})

	// 如果战斗结束, 清理状态
	if battle.State != engine.BattleStateRunning {
		h.activeBattles.Delete(req.BattleID)
		h.playerBattles.Delete(battle.Player1ID)
		h.playerBattles.Delete(battle.Player2ID)

		log.Info().
			Str("battle_id", battle.ID).
			Str("result", string(battle.State)).
			Str("player1", battle.Player1ID).
			Str("player2", battle.Player2ID).
			Msg("PVP战斗结束")

		// 返回完整结果
		result := battle.BuildResult()
		c.JSON(http.StatusOK, result)
		return
	}

	c.JSON(http.StatusOK, turnResult)
}

// GetBattleStatus 获取战斗状态
//
// GET /api/v1/pvp/status?battle_id=xxx
func (h *PVPHandler) GetBattleStatus(c *gin.Context) {
	battleID := c.Query("battle_id")
	if battleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少battle_id参数"})
		return
	}

	battleInterface, ok := h.activeBattles.Load(battleID)
	if !ok {
		// 战斗可能已结束, 返回简单信息
		c.JSON(http.StatusOK, gin.H{
			"battle_id": battleID,
			"state":     "ended",
		})
		return
	}

	battle := battleInterface.(*engine.Battle)

	c.JSON(http.StatusOK, gin.H{
		"battle_id":  battle.ID,
		"state":      battle.State,
		"turn":       battle.TurnNumber,
		"max_turns":  battle.MaxTurns,
		"player1_id": battle.Player1ID,
		"player2_id": battle.Player2ID,
	})
}

// startPVPBattle 创建并开始 PVP 战斗
func (h *PVPHandler) startPVPBattle(matchResult *service.MatchResult) {
	// 这里需要完整的玩家队伍数据, 实际项目中通过 playerID 从游戏服务获取
	// 此处简化, 创建模拟数据
	team1 := h.createMockPlayerTeam(matchResult.Player1)
	team2 := h.createMockPlayerTeam(matchResult.Player2)

	battle := engine.NewPVPBattle(
		matchResult.Player1.PlayerID,
		matchResult.Player2.PlayerID,
		team1,
		team2,
		&h.cfg.Game,
	)

	battleID := uuid.New().String()
	battle.ID = battleID

	h.activeBattles.Store(battleID, battle)
	h.playerBattles.Store(matchResult.Player1.PlayerID, battleID)
	h.playerBattles.Store(matchResult.Player2.PlayerID, battleID)

	battle.State = engine.BattleStateRunning

	log.Info().
		Str("battle_id", battleID).
		Str("player1", matchResult.Player1.Name).
		Str("player2", matchResult.Player2.Name).
		Msg("PVP战斗创建成功, 等待玩家提交行动")
}

// createMockPlayerTeam 创建模拟玩家队伍(生产环境从数据库加载)
func (h *PVPHandler) createMockPlayerTeam(profile *service.PlayerProfile) []*model.Fighter {
	fighter := model.NewFighter(profile.PlayerID, profile.Name, model.FighterTypePlayer, model.ElementWater, profile.Level)
	fighter.BaseAttack = float64(profile.Level) * 15
	fighter.BaseDefense = float64(profile.Level) * 8
	fighter.BaseSpeed = float64(profile.Level) * 5
	fighter.BaseHP = float64(profile.Level) * 100
	fighter.BaseMaxHP = float64(profile.Level) * 100
	fighter.MP = 100
	fighter.MaxMP = 100
	fighter.ApplyPassiveStats()
	return []*model.Fighter{fighter}
}

// GetRankings 获取段位排行榜(模拟)
//
// GET /api/v1/pvp/rankings
func (h *PVPHandler) GetRankings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "排行榜功能需要对接数据服务",
	})
}
