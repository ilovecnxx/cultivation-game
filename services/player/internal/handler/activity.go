package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ActivityHandler 运营活动 HTTP 处理器
type ActivityHandler struct {
	activitySvc *service.ActivityService
	log         *zap.Logger
}

// NewActivityHandler 创建 ActivityHandler
func NewActivityHandler(activitySvc *service.ActivityService, log *zap.Logger) *ActivityHandler {
	return &ActivityHandler{activitySvc: activitySvc, log: log}
}

// ============================================================
// 限时活动
// ============================================================

// ListEvents 获取当前活跃活动列表
// GET /api/v1/activity/events
func (h *ActivityHandler) ListEvents(c *gin.Context) {
	events, err := h.activitySvc.GetActiveEvents()
	if err != nil {
		h.log.Error("获取活动列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取活动列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": events,
	})
}

// GetEventDetail 获取活动详情及玩家进度
// GET /api/v1/activity/events/:eventID
func (h *ActivityHandler) GetEventDetail(c *gin.Context) {
	eventID := c.Param("eventID")
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

	resp, err := h.activitySvc.GetEventDetail(eventID, playerID)
	if err != nil {
		h.log.Error("获取活动详情失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// ClaimEventReward 领取活动奖励
// POST /api/v1/activity/events/:eventID/claim
func (h *ActivityHandler) ClaimEventReward(c *gin.Context) {
	eventID := c.Param("eventID")
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	rewards, err := h.activitySvc.ClaimEventReward(eventID, req.PlayerID)
	if err != nil {
		h.log.Warn("领取活动奖励失败", zap.Int64("player", req.PlayerID), zap.String("event", eventID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "领取成功",
		"data": rewards,
	})
}

// ============================================================
// 战令系统
// ============================================================

// GetBattlePass 获取战令赛季及玩家进度
// GET /api/v1/activity/battlepass
func (h *ActivityHandler) GetBattlePass(c *gin.Context) {
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

	resp, err := h.activitySvc.GetBattlePass(playerID)
	if err != nil {
		h.log.Error("获取战令状态失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// BuyPremiumBP 购买高级战令
// POST /api/v1/activity/battlepass/buy
func (h *ActivityHandler) BuyPremiumBP(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	if err := h.activitySvc.BuyPremiumBP(req.PlayerID); err != nil {
		h.log.Warn("购买高级战令失败", zap.Int64("player", req.PlayerID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "购买高级战令成功",
	})
}

// ClaimBPReward 领取战令等级奖励
// POST /api/v1/activity/battlepass/claim/:level
func (h *ActivityHandler) ClaimBPReward(c *gin.Context) {
	levelStr := c.Param("level")
	level, err := strconv.Atoi(levelStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的等级"})
		return
	}

	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	tier, err := h.activitySvc.ClaimBPReward(req.PlayerID, level)
	if err != nil {
		h.log.Warn("领取战令奖励失败", zap.Int64("player", req.PlayerID), zap.Int("level", level), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "领取成功",
		"data": tier,
	})
}

// ============================================================
// 签到增强
// ============================================================

// GetMonthlyCheckin 获取每月签到状态
// GET /api/v1/activity/checkin/month
func (h *ActivityHandler) GetMonthlyCheckin(c *gin.Context) {
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

	status, err := h.activitySvc.GetEnhancedCheckinStatus(playerID)
	if err != nil {
		h.log.Error("获取签到状态失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取签到状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": status,
	})
}

// DoCheckin 每日签到
// POST /api/v1/activity/checkin
func (h *ActivityHandler) DoCheckin(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	result, err := h.activitySvc.DoEnhancedCheckin(req.PlayerID)
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
// POST /api/v1/activity/checkin/makeup
func (h *ActivityHandler) DoMakeupCheckin(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	result, err := h.activitySvc.DoMakeupCheckinEnhanced(req.PlayerID)
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

// ============================================================
// 成就系统增强
// ============================================================

// GetPlayerAchievements 获取玩家成就列表
// GET /api/v1/activity/achievements/:playerID
func (h *ActivityHandler) GetPlayerAchievements(c *gin.Context) {
	playerIDStr := c.Param("playerID")
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	achievements, progress, err := h.activitySvc.GetPlayerAchievements(playerID)
	if err != nil {
		h.log.Error("查询成就列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询成就列表失败"})
		return
	}

	// 构建带进度的响应
	type AchievementWithProgress struct {
		Achievement  *model.AchievementReq         `json:"achievement"`
		Progress     *model.PlayerAchievementTier   `json:"progress"`
		Tiers        []*model.AchievementTier       `json:"tiers"`
	}

	resp := make([]*AchievementWithProgress, 0, len(achievements))
	for i, ach := range achievements {
		item := &AchievementWithProgress{
			Achievement: ach,
			Progress:    progress[i],
		}
		tiers, _ := h.activitySvc.GetAchievementTiers(ach.ID)
		item.Tiers = tiers
		resp = append(resp, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// ClaimAchievement 领取成就奖励
// POST /api/v1/activity/achievements/claim
func (h *ActivityHandler) ClaimAchievement(c *gin.Context) {
	var req struct {
		PlayerID      int64  `json:"player_id" binding:"required"`
		AchievementID string `json:"achievement_id" binding:"required"`
		Tier          int    `json:"tier" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	reward, err := h.activitySvc.ClaimAchievementTier(req.PlayerID, req.AchievementID, req.Tier)
	if err != nil {
		h.log.Warn("领取成就奖励失败", zap.Int64("player", req.PlayerID), zap.String("achievement", req.AchievementID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "领取成功",
		"data": reward,
	})
}

// ============================================================
// 称号系统
// ============================================================

// GetPlayerTitles 获取玩家称号列表
// GET /api/v1/activity/titles/:playerID
func (h *ActivityHandler) GetPlayerTitles(c *gin.Context) {
	playerIDStr := c.Param("playerID")
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.activitySvc.GetPlayerTitles(playerID)
	if err != nil {
		h.log.Error("查询称号列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询称号列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// EquipTitle 装备/卸下称号
// POST /api/v1/activity/titles/equip
func (h *ActivityHandler) EquipTitle(c *gin.Context) {
	var req struct {
		PlayerID int64  `json:"player_id" binding:"required"`
		TitleID  string `json:"title_id" binding:"required"`
		Equip    bool   `json:"equip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	if err := h.activitySvc.EquipTitle(req.PlayerID, req.TitleID, req.Equip); err != nil {
		h.log.Warn("装备称号失败", zap.Int64("player", req.PlayerID), zap.String("title", req.TitleID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	msg := "装备称号成功"
	if !req.Equip {
		msg = "卸下称号成功"
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  msg,
	})
}
