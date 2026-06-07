package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// EnergyHandler 体力/能量 HTTP 处理器
type EnergyHandler struct {
	energyService *service.EnergyService
	log           *zap.Logger
}

// NewEnergyHandler 创建 EnergyHandler
func NewEnergyHandler(energyService *service.EnergyService, log *zap.Logger) *EnergyHandler {
	return &EnergyHandler{energyService: energyService, log: log}
}

// GetStatus 获取体力状态
// GET /api/v1/player/:id/energy/status
func (h *EnergyHandler) GetStatus(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	status, err := h.energyService.GetEnergy(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("获取体力状态失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": status,
	})
}

// UseEnergyPill 使用体力丹药（支持多品阶）
// POST /api/v1/player/:id/energy/use-pill
func (h *EnergyHandler) UseEnergyPill(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.UseEnergyPillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}
	req.PlayerID = playerID

	if req.Quantity < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "数量必须大于0"})
		return
	}

	result, err := h.energyService.RecoverFromPill(c.Request.Context(), playerID, req.PillID, req.Quantity)
	if err != nil {
		h.log.Warn("使用体力丹药失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "使用" + result.PillName + "成功",
		"data": result,
	})
}

// Meditate 修炼打坐恢复体力
// POST /api/v1/player/:id/energy/meditate
func (h *EnergyHandler) Meditate(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.MeditateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}
	req.PlayerID = playerID

	if req.DurationMin < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "修炼时间必须大于0分钟"})
		return
	}

	result, err := h.energyService.RecoverFromMeditation(c.Request.Context(), playerID, req.DurationMin)
	if err != nil {
		h.log.Warn("修炼打坐恢复体力失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "修炼打坐结束，体力恢复" + strconv.Itoa(result.EnergyGained),
		"data": result,
	})
}

// CheckEnergy 检查是否有足够能量（供其他服务调用）
// GET /api/v1/player/:id/energy/check/:action
func (h *EnergyHandler) CheckEnergy(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	actionType := c.Param("action")
	if actionType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "缺少行动类型"})
		return
	}

	ok, deficit, err := h.energyService.CheckEnergy(c.Request.Context(), playerID, actionType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"msg":     "体力不足",
			"data":    gin.H{"sufficient": false, "deficit": deficit},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{"sufficient": true},
	})
}

// ConsumeEnergy 消耗体力（供其他服务调用）
// POST /api/v1/player/:id/energy/consume
func (h *EnergyHandler) ConsumeEnergy(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.ConsumeEnergyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}
	req.PlayerID = playerID

	remaining, err := h.energyService.ConsumeEnergy(c.Request.Context(), playerID, req.ActionType)
	if err != nil {
		h.log.Warn("消耗体力失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"current_energy": remaining,
			"action_type":    req.ActionType,
		},
	})
}

// SetTechniqueBonus 设置功法体力回复加成（由修炼服务调用）
// POST /api/v1/player/:id/energy/technique-bonus
func (h *EnergyHandler) SetTechniqueBonus(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		Bonus float64 `json:"bonus" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	h.energyService.SetTechniqueBonus(playerID, req.Bonus)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "功法体力回复加成已更新",
	})
}
