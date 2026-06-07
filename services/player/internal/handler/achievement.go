package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AchievementHandler 成就 HTTP 处理器
type AchievementHandler struct {
	achievementService *service.AchievementService
	log                *zap.Logger
}

// NewAchievementHandler 创建 AchievementHandler
func NewAchievementHandler(achievementService *service.AchievementService, log *zap.Logger) *AchievementHandler {
	return &AchievementHandler{
		achievementService: achievementService,
		log:                log,
	}
}

// GetAchievements 查询玩家成就列表（含进度）
// GET /api/v1/player/:id/achievements
func (h *AchievementHandler) GetAchievements(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	progress, err := h.achievementService.GetProgress(c.Request.Context(), playerID)
	if err != nil {
		h.log.Error("查询成就列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询成就列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": progress,
	})
}

// ClaimAchievement 领取已完成成就的奖励
// POST /api/v1/player/:id/achievements/claim
func (h *AchievementHandler) ClaimAchievement(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		AchievementID int `json:"achievement_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	result, err := h.achievementService.Claim(c.Request.Context(), playerID, req.AchievementID)
	if err != nil {
		h.log.Warn("领取成就奖励失败", zap.Uint64("player", playerID), zap.Int("achievement", req.AchievementID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "领取成功",
		"data": result,
	})
}

// GetTitle 获取玩家当前称号
// GET /api/v1/player/:id/title
func (h *AchievementHandler) GetTitle(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.achievementService.GetTitle(c.Request.Context(), playerID)
	if err != nil {
		h.log.Error("查询称号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询称号失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// UpdateAchievementProgress 外部服务调用：更新成就进度
// POST /api/v1/player/:id/achievements/progress
func (h *AchievementHandler) UpdateAchievementProgress(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		AchievementID int `json:"achievement_id" binding:"required"`
		Progress      int `json:"progress" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	if err := h.achievementService.UpdateProgress(c.Request.Context(), playerID, req.AchievementID, req.Progress); err != nil {
		h.log.Error("更新成就进度失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新成功"})
}
