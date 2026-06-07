package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CheckinHandler 签到 HTTP 处理器
type CheckinHandler struct {
	checkinSvc *service.CheckinService
	log        *zap.Logger
}

// NewCheckinHandler 创建 CheckinHandler
func NewCheckinHandler(checkinSvc *service.CheckinService, log *zap.Logger) *CheckinHandler {
	return &CheckinHandler{checkinSvc: checkinSvc, log: log}
}

// DoCheckin 执行签到
// POST /api/v1/checkin
func (h *CheckinHandler) DoCheckin(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	result, err := h.checkinSvc.Checkin(req.PlayerID)
	if err != nil {
		h.log.Warn("签到失败", zap.Int64("player", req.PlayerID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "签到成功",
		"data": result,
	})
}

// DoMakeupCheckin 补签
// POST /api/v1/checkin/makeup
func (h *CheckinHandler) DoMakeupCheckin(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	result, err := h.checkinSvc.MakeupCheckin(req.PlayerID)
	if err != nil {
		h.log.Warn("补签失败", zap.Int64("player", req.PlayerID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "补签成功",
		"data": result,
	})
}

// GetStatus 获取签到状态
// GET /api/v1/checkin/status
func (h *CheckinHandler) GetStatus(c *gin.Context) {
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

	status, err := h.checkinSvc.GetStatus(playerID)
	if err != nil {
		h.log.Warn("查询签到状态失败", zap.Int64("player", playerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询签到状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": status,
	})
}
