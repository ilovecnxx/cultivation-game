package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RebirthHandler 轮回转世 HTTP 处理器
type RebirthHandler struct {
	rebirthService *service.RebirthService
	log            *zap.Logger
}

// NewRebirthHandler 创建 RebirthHandler
func NewRebirthHandler(rebirthService *service.RebirthService, log *zap.Logger) *RebirthHandler {
	return &RebirthHandler{
		rebirthService: rebirthService,
		log:            log,
	}
}

// Check 检查轮回状态和条件
// GET /api/v1/player/:id/rebirth/check
func (h *RebirthHandler) Check(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.rebirthService.CheckRebirth(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("检查轮回失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// Execute 执行轮回转世
// POST /api/v1/player/:id/rebirth/execute
func (h *RebirthHandler) Execute(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.rebirthService.ExecuteRebirth(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("执行轮回失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "轮回转世成功",
		"data": resp,
	})
}

// Benefits 获取轮回福利列表
// GET /api/v1/player/:id/rebirth/benefits
func (h *RebirthHandler) Benefits(c *gin.Context) {
	benefits := h.rebirthService.GetRebirthBenefits(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": benefits,
	})
}

// List 获取轮回历史
// GET /api/v1/player/:id/rebirth/list
func (h *RebirthHandler) List(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	histories, err := h.rebirthService.ListRebirthHistory(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询轮回历史失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": histories,
	})
}

// ============================================================
// 天赋树 API
// ============================================================

// GetTalentInfo 获取天赋信息
// GET /api/v1/player/:id/rebirth/talent
func (h *RebirthHandler) GetTalentInfo(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.rebirthService.GetTalentInfo(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("获取天赋信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// LearnTalent 学习天赋
// POST /api/v1/player/:id/rebirth/talent/learn
func (h *RebirthHandler) LearnTalent(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		TalentID string `json:"talent_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	if err := h.rebirthService.LearnTalent(c.Request.Context(), playerID, req.TalentID); err != nil {
		h.log.Warn("学习天赋失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "学习天赋成功",
	})
}

// ResetTalents 重置天赋
// POST /api/v1/player/:id/rebirth/talent/reset
func (h *RebirthHandler) ResetTalents(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	if err := h.rebirthService.ResetTalents(c.Request.Context(), playerID); err != nil {
		h.log.Warn("重置天赋失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "重置天赋成功",
	})
}

// ============================================================
// 轮回商店 API
// ============================================================

// GetRebirthShop 获取轮回商店信息
// GET /api/v1/player/:id/rebirth/shop
func (h *RebirthHandler) GetRebirthShop(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.rebirthService.GetRebirthShop(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("获取轮回商店失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// BuyRebirthShopItem 购买轮回商店物品
// POST /api/v1/player/:id/rebirth/shop/buy
func (h *RebirthHandler) BuyRebirthShopItem(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		ShopID string `json:"shop_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	if err := h.rebirthService.BuyRebirthShopItem(c.Request.Context(), playerID, req.ShopID); err != nil {
		h.log.Warn("购买轮回商店物品失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "购买成功",
	})
}

// ============================================================
// 称号 API
// ============================================================

// ListTitles 获取所有称号列表
// GET /api/v1/player/:id/rebirth/titles
func (h *RebirthHandler) ListTitles(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	rebirth, err := h.rebirthService.GetPlayerRebirth(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("获取玩家轮回记录失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	var titles []gin.H
	for i := 0; i <= model.MaxRebirthCount; i++ {
		bonus := model.RebirthTitleBonuses[i]
		unlocked := i <= rebirth.RebirthCount
		titles = append(titles, gin.H{
			"rebirth_count": i,
			"name":          model.RebirthTitleNames[i],
			"attack_pct":    bonus.AttackPct,
			"defense_pct":   bonus.DefensePct,
			"hp_pct":        bonus.HPPct,
			"speed_pct":     bonus.SpeedPct,
			"unlocked":      unlocked,
			"active":        i == rebirth.RebirthCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": titles,
	})
}
