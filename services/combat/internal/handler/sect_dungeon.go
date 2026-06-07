// Package handler 宗门副本 HTTP 处理器
//
// 路由:
//   POST /api/v1/sect-dungeon/start    - 开启副本
//   POST /api/v1/sect-dungeon/join     - 加入副本
//   POST /api/v1/sect-dungeon/attack   - 攻击首领
//   GET  /api/v1/sect-dungeon/status   - 获取副本状态
//   GET  /api/v1/sect-dungeon/configs  - 获取副本配置列表
//   POST /api/v1/sect-dungeon/complete - 完成副本
//   POST /api/v1/sect-dungeon/claim    - 领取奖励
package handler

import (
	"net/http"
	"strconv"
	"time"

	"cultivation-game/services/combat/internal/service"

	"github.com/gin-gonic/gin"
)

// SectDungeonHandler 宗门副本 HTTP 处理器
type SectDungeonHandler struct {
	svc *service.SectDungeonService
}

// NewSectDungeonHandler 创建 SectDungeonHandler
func NewSectDungeonHandler(svc *service.SectDungeonService) *SectDungeonHandler {
	return &SectDungeonHandler{svc: svc}
}

// RegisterRoutes 注册宗门副本路由
func (h *SectDungeonHandler) RegisterRoutes(r *gin.Engine) {
	sectDungeonGroup := r.Group("/api/v1/sect-dungeon")
	{
		sectDungeonGroup.GET("/configs", h.HandleGetConfigs)
		sectDungeonGroup.POST("/start", h.HandleStart)
		sectDungeonGroup.POST("/join", h.HandleJoin)
		sectDungeonGroup.POST("/attack", h.HandleAttack)
		sectDungeonGroup.GET("/status", h.HandleStatus)
		sectDungeonGroup.POST("/complete", h.HandleComplete)
		sectDungeonGroup.POST("/claim", h.HandleClaim)
	}
}

// HandleGetConfigs 获取副本配置列表
// GET /api/v1/sect-dungeon/configs
func (h *SectDungeonHandler) HandleGetConfigs(c *gin.Context) {
	configs := h.svc.GetConfigs()
	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
	})
}

// HandleStart 开启宗门副本
// POST /api/v1/sect-dungeon/start
// Body: { sect_id, dungeon_config_id, leader_id }
func (h *SectDungeonHandler) HandleStart(c *gin.Context) {
	var req service.SectDungeonStartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.SectID == 0 || req.DungeonConfigID == 0 || req.LeaderID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: sect_id, dungeon_config_id, leader_id"})
		return
	}

	session, err := h.svc.StartSectDungeon(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "副本已开启",
		"session_id":  session.ID,
		"config_id":   session.ConfigID,
		"status":      session.Status,
	})
}

// HandleJoin 加入宗门副本
// POST /api/v1/sect-dungeon/join
// Body: { session_id, player_id }
func (h *SectDungeonHandler) HandleJoin(c *gin.Context) {
	var req service.SectDungeonJoinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.SessionID == 0 || req.PlayerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: session_id, player_id"})
		return
	}

	if err := h.svc.JoinSectDungeon(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取副本状态用于返回
	session := h.svc.GetSession(req.SessionID)
	status := h.buildStatusResponse(session)

	c.JSON(http.StatusOK, gin.H{
		"message":  "已加入副本",
		"session":  status,
	})
}

// HandleAttack 攻击首领
// POST /api/v1/sect-dungeon/attack
// Body: { session_id, player_id, attack_val }
func (h *SectDungeonHandler) HandleAttack(c *gin.Context) {
	var req service.SectDungeonAttackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.SessionID == 0 || req.PlayerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: session_id, player_id"})
		return
	}
	if req.AttackVal <= 0 {
		req.AttackVal = 100
	}

	result, err := h.svc.AttackBoss(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "攻击成功",
		"result":  result,
	})
}

// HandleStatus 获取宗门副本状态
// GET /api/v1/sect-dungeon/status?sect_id=xxx
func (h *SectDungeonHandler) HandleStatus(c *gin.Context) {
	sectIDStr := c.Query("sect_id")
	if sectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数: sect_id"})
		return
	}
	sectID, err := strconv.ParseUint(sectIDStr, 10, 64)
	if err != nil || sectID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 sect_id"})
		return
	}

	status := h.svc.GetActiveSectDungeon(sectID)

	c.JSON(http.StatusOK, gin.H{
		"dungeon": status,
	})
}

// HandleComplete 完成副本(手动结束)
// POST /api/v1/sect-dungeon/complete
// Body: { session_id }
func (h *SectDungeonHandler) HandleComplete(c *gin.Context) {
	var req struct {
		SessionID uint64 `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SessionID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数: session_id"})
		return
	}

	result, err := h.svc.CompleteSectDungeon(req.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := "副本已结束"
	if result.BossDefeated {
		message = "首领已被击败!"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"result":  result,
	})
}

// HandleClaim 领取奖励
// POST /api/v1/sect-dungeon/claim
// Body: { session_id, player_id }
func (h *SectDungeonHandler) HandleClaim(c *gin.Context) {
	var req struct {
		SessionID uint64 `json:"session_id"`
		PlayerID  uint64 `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SessionID == 0 || req.PlayerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数: session_id, player_id"})
		return
	}

	rewards, err := h.svc.ClaimRewards(req.SessionID, req.PlayerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "奖励已领取",
		"rewards": rewards,
	})
}

// buildStatusResponse 构建副本状态响应
func (h *SectDungeonHandler) buildStatusResponse(session *service.SectDungeonSession) *service.SectDungeonStatus {
	if session == nil {
		return &service.SectDungeonStatus{Active: false}
	}
	cfg := h.svc.GetConfig(session.ConfigID)
	remaining := 0
	if cfg != nil {
		elapsed := time.Since(session.StartedAt)
		total := time.Duration(cfg.DurationMinutes) * time.Minute
		rem := total - elapsed
		if rem > 0 {
			remaining = int(rem.Seconds())
		}
	}

	return &service.SectDungeonStatus{
		Session:          session,
		Config:           cfg,
		RemainingSeconds: remaining,
		Participants:     session.Participants,
		Active:           session.Status == service.SectDungeonStatusInProgress,
	}
}
