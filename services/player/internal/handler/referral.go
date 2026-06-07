package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ReferralHandler 推荐系统 HTTP 处理器
type ReferralHandler struct {
	referralService *service.ReferralService
	log             *zap.Logger
}

// NewReferralHandler 创建 ReferralHandler
func NewReferralHandler(referralService *service.ReferralService, log *zap.Logger) *ReferralHandler {
	return &ReferralHandler{
		referralService: referralService,
		log:             log,
	}
}

// GetReferralInfo 查询邀请信息
// GET /api/v1/referral/info?player_id=123
func (h *ReferralHandler) GetReferralInfo(c *gin.Context) {
	playerID, err := getPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	info, err := h.referralService.GetReferralInfo(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询邀请信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": info,
	})
}

// ApplyInviteCode 使用邀请码
// POST /api/v1/referral/apply
func (h *ReferralHandler) ApplyInviteCode(c *gin.Context) {
	var req model.ApplyInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	if err := h.referralService.ApplyInviteCode(c.Request.Context(), req.InviteeID, req.Code); err != nil {
		h.log.Warn("使用邀请码失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "邀请绑定成功",
	})
}

// ClaimReferralReward 领取推荐奖励
// POST /api/v1/referral/claim/:inviteeId?player_id=123
func (h *ReferralHandler) ClaimReferralReward(c *gin.Context) {
	playerID, err := getPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	inviteeIDStr := c.Param("inviteeId")
	inviteeID, err := strconv.ParseInt(inviteeIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的被邀请者ID"})
		return
	}

	if err := h.referralService.ClaimReferralReward(c.Request.Context(), playerID, inviteeID); err != nil {
		h.log.Warn("领取推荐奖励失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "奖励领取成功",
	})
}

// ---------- 内部方法 ----------

// getPlayerID 从请求中提取玩家ID（支持 query param）
func getPlayerID(c *gin.Context) (int64, error) {
	idStr := c.Query("player_id")
	if idStr == "" {
		idStr = c.Param("player_id")
	}
	if idStr == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(idStr, 10, 64)
}
