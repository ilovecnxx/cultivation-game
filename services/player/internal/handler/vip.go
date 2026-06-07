package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VipHandler VIP系统 HTTP 处理器
type VipHandler struct {
	vipService *service.VipService
	log        *zap.Logger
}

// NewVipHandler 创建 VipHandler
func NewVipHandler(vipService *service.VipService, log *zap.Logger) *VipHandler {
	return &VipHandler{
		vipService: vipService,
		log:        log,
	}
}

// GetVipInfo 查询玩家VIP信息
// GET /api/v1/vip/info?player_id=123
func (h *VipHandler) GetVipInfo(c *gin.Context) {
	playerID, err := getPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	info, err := h.vipService.GetVipInfo(c.Request.Context(), playerID)
	if err != nil {
		h.log.Error("查询VIP信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": info,
	})
}

// ClaimDailyReward 领取VIP每日奖励
// POST /api/v1/vip/claim-daily
func (h *VipHandler) ClaimDailyReward(c *gin.Context) {
	playerID, err := getPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	rewards, err := h.vipService.ClaimDailyReward(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("领取VIP每日奖励失败", zap.Int64("player", playerID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "领取成功",
		"data": gin.H{
			"rewards": rewards,
		},
	})
}

// ProcessRecharge 处理充值
// POST /api/v1/vip/recharge
func (h *VipHandler) ProcessRecharge(c *gin.Context) {
	var req model.RechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	result, err := h.vipService.ProcessRecharge(c.Request.Context(), req.PlayerID, req.AmountRmb, req.OrderID)
	if err != nil {
		h.log.Warn("充值处理失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "充值成功",
		"data": result,
	})
}

// GetRechargeHistory 查询充值历史
// GET /api/v1/vip/recharge-history?player_id=123&limit=20&offset=0
func (h *VipHandler) GetRechargeHistory(c *gin.Context) {
	playerID, err := getPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	records, err := h.vipService.GetRechargeHistory(c.Request.Context(), playerID, limit, offset)
	if err != nil {
		h.log.Error("查询充值历史失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"records": records,
			"limit":   limit,
			"offset":  offset,
		},
	})
}

// ActivateMonthlyCard 激活月卡
// POST /api/v1/vip/activate-monthly-card
func (h *VipHandler) ActivateMonthlyCard(c *gin.Context) {
	var req model.ActivateMonthlyCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	result, err := h.vipService.ActivateMonthlyCard(c.Request.Context(), req.PlayerID, req.CardType)
	if err != nil {
		h.log.Warn("激活月卡失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "月卡激活成功",
		"data": result,
	})
}

// GetMonthlyCardStatus 查询月卡状态
// GET /api/v1/vip/monthly-card-status?player_id=123
func (h *VipHandler) GetMonthlyCardStatus(c *gin.Context) {
	playerID, err := getPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	status, err := h.vipService.GetMonthlyCardStatus(c.Request.Context(), playerID)
	if err != nil {
		h.log.Error("查询月卡状态失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": status,
	})
}
