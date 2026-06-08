package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/engine"
	"cultivation-game/services/combat/internal/model"
	"cultivation-game/services/combat/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ArenaHandler 竞技场 HTTP 处理器
type ArenaHandler struct {
	cfg      *config.Config
	arenaSvc *service.ArenaService
}

// NewArenaHandler 创建竞技场处理器并启动后台匹配
func NewArenaHandler(cfg *config.Config) *ArenaHandler {
	h := &ArenaHandler{
		cfg: cfg,
		arenaSvc: service.NewArenaService(
			cfg.Game.SeasonDurationDays,
			nil, // 先创建, 下面设置回调
		),
	}

	// 设置匹配成功回调(自动战斗)
	h.arenaSvc = service.NewArenaService(
		cfg.Game.SeasonDurationDays,
		func(match *service.ArenaMatch) {
			h.onArenaMatch(match)
		},
	)

	h.arenaSvc.StartLoop()
	return h
}

// HandleMatch 发起竞技场匹配
//
// POST /api/v1/arena/match
func (h *ArenaHandler) HandleMatch(c *gin.Context) {
	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}

	if h.arenaSvc.IsInQueue(req.PlayerID) {
		c.JSON(http.StatusConflict, gin.H{"error": "已在匹配队列中"})
		return
	}

	// 确保玩家数据存在
	h.arenaSvc.GetOrCreatePlayer(req.PlayerID)

	ok := h.arenaSvc.EnqueueMatch(req.PlayerID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加入队列失败"})
		return
	}

	log.Info().Str("player_id", req.PlayerID).Msg("加入竞技场匹配队列")

	c.JSON(http.StatusOK, gin.H{
		"message":    "已加入竞技场匹配队列",
		"queue_size": h.arenaSvc.QueueSize(),
	})
}

// HandleStatus 查看我的段位/积分/排名
//
// GET /api/v1/arena/status?player_id=xxx
func (h *ArenaHandler) HandleStatus(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id 参数"})
		return
	}

	player := h.arenaSvc.GetPlayer(playerID)
	if player == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "玩家不存在, 请先参与竞技场"})
		return
	}

	ranking := h.arenaSvc.GetPlayerRanking(playerID)
	season := h.arenaSvc.GetSeasonService().GetCurrentSeason()

	c.JSON(http.StatusOK, gin.H{
		"player_id":        player.PlayerID,
		"rank":             player.Rank,
		"tier":             player.Tier,
		"score":            player.Score,
		"season_win":       player.SeasonWin,
		"season_lose":      player.SeasonLose,
		"streak":           player.Streak,
		"last_season_rank": player.LastSeasonRank,
		"daily_win_count":  player.DailyWinCount,
		"ranking":          ranking,
		"season_id":        season.SeasonID,
	})
}

// HandleRankings 获取竞技场排行榜
//
// GET /api/v1/arena/rankings
func (h *ArenaHandler) HandleRankings(c *gin.Context) {
	rankings := h.arenaSvc.GetRankings()

	type rankEntry struct {
		Rank     int    `json:"rank"`
		PlayerID string `json:"player_id"`
		Score    int    `json:"score"`
		RankName string `json:"rank_name"`
		Tier     int    `json:"tier"`
		WinCount int    `json:"win_count"`
		Streak   int    `json:"streak"`
	}

	entries := make([]rankEntry, 0, len(rankings))
	for i, p := range rankings {
		entries = append(entries, rankEntry{
			Rank:     i + 1,
			PlayerID: p.PlayerID,
			Score:    p.Score,
			RankName: p.Rank,
			Tier:     p.Tier,
			WinCount: p.SeasonWin,
			Streak:   p.Streak,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"rankings": entries,
		"total":    len(entries),
	})
}

// HandleCancelMatch 取消竞技场匹配
//
// POST /api/v1/arena/cancel-match
func (h *ArenaHandler) HandleCancelMatch(c *gin.Context) {
	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}

	if h.arenaSvc.DequeueMatch(req.PlayerID) {
		log.Info().Str("player_id", req.PlayerID).Msg("取消竞技场匹配")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已取消匹配",
	})
}

// HandleSeason 获取赛季信息
//
// GET /api/v1/arena/season
func (h *ArenaHandler) HandleSeason(c *gin.Context) {
	season := h.arenaSvc.GetSeasonService().GetCurrentSeason()

	// 构建奖励预览
	rewardsPreview := make(map[string]interface{})
	for rank, reward := range service.SeasonRewardsByRank {
		rewardsPreview[rank] = reward
	}

	c.JSON(http.StatusOK, gin.H{
		"season_id":   season.SeasonID,
		"name":        season.Name,
		"start_time":  season.StartTime,
		"end_time":    season.EndTime,
		"status":      season.Status,
		"rewards":     rewardsPreview,
	})
}

// HandleHistory 获取对战记录
//
// GET /api/v1/arena/history?player_id=xxx&limit=10&before=1234567890
func (h *ArenaHandler) HandleHistory(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id 参数"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	var before int64
	if b := c.Query("before"); b != "" {
		if parsed, err := strconv.ParseInt(b, 10, 64); err == nil {
			before = parsed
		}
	}

	records := h.arenaSvc.GetHistory(playerID, limit, before)
	if records == nil {
		records = make([]*model.MatchRecord, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"count":   len(records),
	})
}

// ---------- 内部: 匹配成功回调 ----------

// onArenaMatch 匹配成功回调, 自动进行战斗并结算
func (h *ArenaHandler) onArenaMatch(match *service.ArenaMatch) {
	log.Info().
		Str("player_a", match.PlayerA).
		Str("player_b", match.PlayerB).
		Int("score_a", match.ScoreA).
		Int("score_b", match.ScoreB).
		Msg("竞技场匹配成功, 开始自动战斗")

	// 使用模拟队伍进行战斗(生产环境从数据库加载真实数据)
	team1 := h.createArenaMockTeam(match.PlayerA, match.ScoreA)
	team2 := h.createArenaMockTeam(match.PlayerB, match.ScoreB)

	battle := engine.NewPVPBattle(match.PlayerA, match.PlayerB, team1, team2, &h.cfg.Game)
	result := battle.Start()

	// 根据结果结算
	rounds := result.TotalTurns
	switch result.State {
	case engine.BattleStatePlayerWin:
		// PlayerA 胜利
		h.arenaSvc.RecordResult(match.PlayerA, match.PlayerB, rounds)
	case engine.BattleStateEnemyWin:
		// PlayerB 胜利(PVP中用EnemyWin表示Player2胜利)
		h.arenaSvc.RecordResult(match.PlayerB, match.PlayerA, rounds)
	default:
		// 平局
		h.arenaSvc.RecordDraw(match.PlayerA, match.PlayerB, rounds)
	}
}

// createArenaMockTeam 创建竞技场模拟队伍
func (h *ArenaHandler) createArenaMockTeam(playerID string, score int) []*model.Fighter {
	// 根据积分估算等级和战力
	estimatedLevel := (score / 100) + 1
	if estimatedLevel < 1 {
		estimatedLevel = 1
	}
	if estimatedLevel > 100 {
		estimatedLevel = 100
	}

	fighter := model.NewFighter(playerID, "玩家", model.FighterTypePlayer, model.ElementWater, estimatedLevel)
	fighter.BaseAttack = int64(estimatedLevel) * 15
	fighter.BaseDefense = int64(estimatedLevel) * 8
	fighter.BaseSpeed = int64(estimatedLevel) * 5
	fighter.BaseHP = int64(estimatedLevel) * 100
	fighter.BaseMaxHP = int64(estimatedLevel) * 100
	fighter.MP = 100
	fighter.MaxMP = 100
	fighter.ApplyPassiveStats()
	return []*model.Fighter{fighter}
}
