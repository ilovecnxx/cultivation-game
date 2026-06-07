// Package handler 世界Boss HTTP处理器
//
// 路由:
//   旧接口(前端兼容): /api/v1/boss/*
//   新接口(任务要求): /api/v1/world-boss/*
package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/world/internal/model"
	"cultivation-game/services/world/internal/service"

	"github.com/gin-gonic/gin"
)

// WorldBossHandler 世界Boss HTTP处理器
type WorldBossHandler struct {
	bossSvc *service.WorldBossService
}

// NewWorldBossHandler 创建WorldBossHandler
func NewWorldBossHandler(bossSvc *service.WorldBossService) *WorldBossHandler {
	return &WorldBossHandler{bossSvc: bossSvc}
}

// RegisterRoutes 注册Boss相关路由
func (h *WorldBossHandler) RegisterRoutes(r *gin.Engine) {
	// ---- 旧接口 (前端兼容) ----
	r.GET("/api/v1/boss/list", h.handleList)
	r.GET("/api/v1/boss/status", h.handleStatus)
	r.POST("/api/v1/boss/attack", h.handleAttack)
	r.GET("/api/v1/boss/kill/:boss_id", h.handleKillRecord)
	r.GET("/api/v1/boss/ranking/:boss_id", h.handleRanking)

	// ---- 新接口 (任务要求) ----
	r.GET("/api/v1/world-boss/list", h.handleWorldBossList)
	r.GET("/api/v1/world-boss/active", h.handleWorldBossActive)
	r.GET("/api/v1/world-boss/upcoming", h.handleWorldBossUpcoming)
	r.GET("/api/v1/world-boss/buff", h.handleWorldBossBuff)
	r.GET("/api/v1/world-boss/:bossID", h.handleWorldBossDetail)
	r.POST("/api/v1/world-boss/:bossID/attack", h.handleWorldBossAttack)
	r.GET("/api/v1/world-boss/:bossID/rankings", h.handleWorldBossRankings)
	r.GET("/api/v1/world-boss/:bossID/rewards", h.handleWorldBossRewards)
}

// ============================================================
// 旧接口 (前端兼容)
// ============================================================

// handleList 获取所有Boss列表
// GET /api/v1/boss/list
func (h *WorldBossHandler) handleList(c *gin.Context) {
	bosses := h.bossSvc.ListBosses()
	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取Boss列表成功",
		Data: map[string]interface{}{
			"bosses": bosses,
		},
	})
}

// handleStatus 获取单个Boss详细状态(含排行榜)
// GET /api/v1/boss/status?boss_id=xxx
func (h *WorldBossHandler) handleStatus(c *gin.Context) {
	bossID := c.Query("boss_id")
	if bossID == "" {
		writeError(c, http.StatusBadRequest, "缺少 boss_id 参数")
		return
	}

	detail, err := h.bossSvc.GetBossDetail(bossID)
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取Boss状态成功",
		Data:    detail,
	})
}

// handleAttack 玩家攻击Boss
// POST /api/v1/boss/attack
func (h *WorldBossHandler) handleAttack(c *gin.Context) {
	var req model.BossAttackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.PlayerID == "" || req.BossID == "" {
		writeError(c, http.StatusBadRequest, "player_id 和 boss_id 不能为空")
		return
	}
	if req.AttackVal <= 0 {
		req.AttackVal = 100
	}
	if req.PlayerName == "" {
		req.PlayerName = "修士" + req.PlayerID
	}

	result, err := h.bossSvc.AttackBoss(&req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "攻击成功",
		Data:    result,
	})
}

// handleKillRecord 获取Boss击杀记录
// GET /api/v1/boss/kill/{boss_id}
func (h *WorldBossHandler) handleKillRecord(c *gin.Context) {
	bossID := c.Param("boss_id")
	if bossID == "" {
		writeError(c, http.StatusBadRequest, "boss_id 不能为空")
		return
	}

	record := h.bossSvc.GetKillRecord(bossID)
	if record == nil {
		writeJSON(c, http.StatusOK, &apiResponse{
			Code:    0,
			Message: "暂无击杀记录",
			Data:    nil,
		})
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取击杀记录成功",
		Data:    record,
	})
}

// handleRanking 获取Boss伤害排行
// GET /api/v1/boss/ranking/{boss_id}
func (h *WorldBossHandler) handleRanking(c *gin.Context) {
	bossID := c.Param("boss_id")
	if bossID == "" {
		writeError(c, http.StatusBadRequest, "boss_id 不能为空")
		return
	}

	rankings, err := h.bossSvc.GetDamageRanking(bossID)
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	if rankings == nil {
		rankings = []model.WorldBossDamage{}
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取排行成功",
		Data:    rankings,
	})
}

// ============================================================
// 新接口 (任务要求)
// ============================================================

// handleWorldBossList 获取所有Boss列表(包含状态)
// GET /api/v1/world-boss/list
func (h *WorldBossHandler) handleWorldBossList(c *gin.Context) {
	bosses := h.bossSvc.ListBosses()

	// 支持筛选: ?status=alive 或 ?region=region_01
	statusFilter := c.Query("status")
	regionFilter := c.Query("region")

	if statusFilter != "" || regionFilter != "" {
		filtered := make([]*model.BossStatusBrief, 0, len(bosses))
		for _, b := range bosses {
			if statusFilter != "" && b.Status != statusFilter {
				continue
			}
			if regionFilter != "" && b.RegionID != regionFilter {
				continue
			}
			filtered = append(filtered, b)
		}
		bosses = filtered
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取Boss列表成功",
		Data: map[string]interface{}{
			"bosses": bosses,
			"total":  len(bosses),
		},
	})
}

// handleWorldBossActive 获取当前活跃Boss
// GET /api/v1/world-boss/active
func (h *WorldBossHandler) handleWorldBossActive(c *gin.Context) {
	active := h.bossSvc.GetActiveBoss()
	if active == nil {
		writeJSON(c, http.StatusOK, &apiResponse{
			Code:    0,
			Message: "当前没有活跃的Boss",
			Data:    nil,
		})
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取活跃Boss成功",
		Data:    active,
	})
}

// handleWorldBossUpcoming 获取即将刷新的Boss
// GET /api/v1/world-boss/upcoming
func (h *WorldBossHandler) handleWorldBossUpcoming(c *gin.Context) {
	bosses := h.bossSvc.GetUpcomingBosses()

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取即将刷新Boss成功",
		Data: map[string]interface{}{
			"bosses": bosses,
		},
	})
}

// handleWorldBossBuff 获取全服Buff信息
// GET /api/v1/world-boss/buff
func (h *WorldBossHandler) handleWorldBossBuff(c *gin.Context) {
	buff := h.bossSvc.GetGlobalBuffInfo()

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取Buff信息成功",
		Data:    buff,
	})
}

// handleWorldBossDetail 获取Boss详情(含当前HP、排行榜)
// GET /api/v1/world-boss/:bossID
func (h *WorldBossHandler) handleWorldBossDetail(c *gin.Context) {
	bossID := c.Param("bossID")
	if bossID == "" {
		writeError(c, http.StatusBadRequest, "bossID 不能为空")
		return
	}

	detail, err := h.bossSvc.GetBossDetail(bossID)
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取Boss详情成功",
		Data:    detail,
	})
}

// handleWorldBossAttack 攻击Boss
// POST /api/v1/world-boss/:bossID/attack
// Body: { player_id, player_name, attack_val }
func (h *WorldBossHandler) handleWorldBossAttack(c *gin.Context) {
	bossID := c.Param("bossID")
	if bossID == "" {
		writeError(c, http.StatusBadRequest, "bossID 不能为空")
		return
	}

	var req struct {
		PlayerID   string  `json:"player_id"`
		PlayerName string  `json:"player_name"`
		AttackVal  float64 `json:"attack_val"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.PlayerID == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}
	if req.AttackVal <= 0 {
		req.AttackVal = 100
	}
	if req.PlayerName == "" {
		req.PlayerName = "修士" + req.PlayerID
	}

	attackReq := &model.BossAttackRequest{
		PlayerID:   req.PlayerID,
		PlayerName: req.PlayerName,
		BossID:     bossID,
		AttackVal:  req.AttackVal,
	}

	result, err := h.bossSvc.AttackBoss(attackReq)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "攻击成功",
		Data:    result,
	})
}

// handleWorldBossRankings 获取Boss伤害排行
// GET /api/v1/world-boss/:bossID/rankings
func (h *WorldBossHandler) handleWorldBossRankings(c *gin.Context) {
	bossID := c.Param("bossID")
	if bossID == "" {
		writeError(c, http.StatusBadRequest, "bossID 不能为空")
		return
	}

	// 支持分页: ?page=1&page_size=10
	page := 1
	pageSize := 50
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	rankings, err := h.bossSvc.GetDamageRanking(bossID)
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	// 分页
	total := len(rankings)
	start := (page - 1) * pageSize
	if start >= total {
		rankings = []model.WorldBossDamage{}
	} else {
		end := start + pageSize
		if end > total {
			end = total
		}
		rankings = rankings[start:end]
	}

	if rankings == nil {
		rankings = []model.WorldBossDamage{}
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取排行成功",
		Data: map[string]interface{}{
			"rankings":   rankings,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
		},
	})
}

// handleWorldBossRewards 获取玩家在指定Boss中的奖励信息
// GET /api/v1/world-boss/:bossID/rewards?player_id=xxx
func (h *WorldBossHandler) handleWorldBossRewards(c *gin.Context) {
	bossID := c.Param("bossID")
	if bossID == "" {
		writeError(c, http.StatusBadRequest, "bossID 不能为空")
		return
	}

	playerID := c.Query("player_id")
	if playerID == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	rewards := h.bossSvc.GetPlayerRewards(bossID, playerID)
	if rewards == nil {
		writeError(c, http.StatusNotFound, "Boss不存在")
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取奖励信息成功",
		Data:    rewards,
	})
}
