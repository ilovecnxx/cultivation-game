package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProtectionHandler 新手保护 HTTP 处理器
type ProtectionHandler struct {
	protectionService *service.ProtectionService
	log               *zap.Logger
}

// NewProtectionHandler 创建 ProtectionHandler
func NewProtectionHandler(protectionService *service.ProtectionService, log *zap.Logger) *ProtectionHandler {
	return &ProtectionHandler{
		protectionService: protectionService,
		log:               log,
	}
}

// GetStatus 获取玩家保护状态
// GET /api/v1/protection/status?player_id=xxx
func (h *ProtectionHandler) GetStatus(c *gin.Context) {
	playerIDStr := c.Query("player_id")
	if playerIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "缺少 player_id 参数"})
		return
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的 player_id"})
		return
	}

	status, err := h.protectionService.GetProtectionInfo(playerID)
	if err != nil {
		h.log.Error("查询保护状态失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询保护状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": status,
	})
}

// UseBreakthroughGrace 使用一次突破免罚次数
// POST /api/v1/protection/breakthrough-grace/use
func (h *ProtectionHandler) UseBreakthroughGrace(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	used, err := h.protectionService.UseBreakthroughGrace(req.PlayerID)
	if err != nil {
		h.log.Error("使用突破免罚次数失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "使用突破免罚次数失败"})
		return
	}
	if !used {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "没有可用的突破免罚次数"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "突破免罚次数已使用",
	})
}

// UseFreeResurrection 使用一次免费复活
// POST /api/v1/protection/free-resurrection/use
func (h *ProtectionHandler) UseFreeResurrection(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	used, err := h.protectionService.UseFreeResurrection(req.PlayerID)
	if err != nil {
		h.log.Error("使用免费复活次数失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "使用免费复活次数失败"})
		return
	}
	if !used {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "没有可用的免费复活次数"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "免费复活次数已使用",
	})
}

// CheckBreakthroughGrace 查询突破免罚减免比例
// GET /api/v1/protection/breakthrough-grace?player_id=xxx
func (h *ProtectionHandler) CheckBreakthroughGrace(c *gin.Context) {
	playerIDStr := c.Query("player_id")
	if playerIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "缺少 player_id 参数"})
		return
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的 player_id"})
		return
	}

	reduction, err := h.protectionService.GetBreakthroughGraceReduction(playerID)
	if err != nil {
		h.log.Error("查询突破免罚减免失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"reduction":          reduction,
			"has_grace":          reduction > 0,
			"penalty_multiplier": 1.0 - reduction,
		},
	})
}
