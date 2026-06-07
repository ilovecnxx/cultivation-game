package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// EquipmentSetHandler 装备套装/附魔/觉醒 HTTP 处理器
type EquipmentSetHandler struct {
	setService *service.EquipmentSetService
	inventoryService *service.InventoryService
	log        *zap.Logger
}

// NewEquipmentSetHandler 创建 EquipmentSetHandler
func NewEquipmentSetHandler(setService *service.EquipmentSetService, inventoryService *service.InventoryService, log *zap.Logger) *EquipmentSetHandler {
	return &EquipmentSetHandler{
		setService:       setService,
		inventoryService: inventoryService,
		log:              log,
	}
}

// ListSets 获取所有装备套装列表
// GET /api/v1/equipment/sets
func (h *EquipmentSetHandler) ListSets(c *gin.Context) {
	sets := model.EquipmentSetList
	if sets == nil {
		sets = []model.EquipmentSet{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": sets})
}

// GetActiveBonuses 获取玩家已激活的套装效果
// GET /api/v1/equipment/sets/active/:playerID
func (h *EquipmentSetHandler) GetActiveBonuses(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("playerID"), 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	bonuses, err := h.setService.GetActiveSetBonuses(c.Request.Context(), playerID)
	if err != nil {
		h.log.Error("获取套装效果失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取套装效果失败", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": bonuses})
}

// GetSetProgress 获取指定套装收集进度
// GET /api/v1/equipment/sets/progress/:playerID
func (h *EquipmentSetHandler) GetSetProgress(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("playerID"), 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	progress, err := h.setService.GetAllSetProgress(c.Request.Context(), playerID)
	if err != nil {
		h.log.Error("获取套装进度失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取套装进度失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": progress})
}

// GetMissingPieces 获取指定套装缺少的部件
// GET /api/v1/equipment/sets/missing/:playerID/:setName
func (h *EquipmentSetHandler) GetMissingPieces(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("playerID"), 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	setName := c.Param("setName")
	if setName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "套装名称不能为空"})
		return
	}

	progress, err := h.setService.GetMissingPieces(c.Request.Context(), playerID, setName)
	if err != nil {
		h.log.Error("获取缺少部件失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": progress})
}

// GetEnchantmentList 获取所有附魔类型
// GET /api/v1/equipment/enchants
func (h *EquipmentSetHandler) GetEnchantmentList(c *gin.Context) {
	groups := h.setService.GetAllEnchantmentGroups()
	if groups == nil {
		groups = []model.EnchantSlotGroup{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": groups})
}

// ApplyEnchant 为装备附魔
// POST /api/v1/equipment/enchant
func (h *EquipmentSetHandler) ApplyEnchant(c *gin.Context) {
	var req model.EnchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	// 从路径或请求体中获取playerID
	playerIDStr := c.Param("playerID")
	if playerIDStr != "" {
		pid, err := strconv.ParseInt(playerIDStr, 10, 64)
		if err == nil && pid > 0 {
			req.PlayerID = pid
		}
	}
	if req.PlayerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	instance, err := h.setService.ApplyEnchantment(c.Request.Context(), req.PlayerID, &req)
	if err != nil {
		h.log.Error("附魔失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "附魔成功", "data": instance})
}

// RemoveEnchant 移除装备附魔
// POST /api/v1/equipment/enchant/remove
func (h *EquipmentSetHandler) RemoveEnchant(c *gin.Context) {
	var req model.EnchantRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	playerIDStr := c.Param("playerID")
	if playerIDStr != "" {
		pid, err := strconv.ParseInt(playerIDStr, 10, 64)
		if err == nil && pid > 0 {
			req.PlayerID = pid
		}
	}
	if req.PlayerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	if err := h.setService.RemoveEnchantment(c.Request.Context(), req.PlayerID, &req); err != nil {
		h.log.Error("移除附魔失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "移除附魔成功"})
}

// GetEquipmentEnchants 获取装备附魔列表
// GET /api/v1/equipment/:equipmentID/enchants
func (h *EquipmentSetHandler) GetEquipmentEnchants(c *gin.Context) {
	equipmentID, err := strconv.ParseInt(c.Param("equipmentID"), 10, 64)
	if err != nil || equipmentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的装备ID"})
		return
	}

	enchants, err := h.setService.GetEquipmentEnchantments(c.Request.Context(), equipmentID)
	if err != nil {
		h.log.Error("获取装备附魔失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取附魔失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": enchants})
}

// AwakenEquipment 装备觉醒
// POST /api/v1/equipment/awaken
func (h *EquipmentSetHandler) AwakenEquipment(c *gin.Context) {
	var req model.AwakenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	playerIDStr := c.Param("playerID")
	if playerIDStr != "" {
		pid, err := strconv.ParseInt(playerIDStr, 10, 64)
		if err == nil && pid > 0 {
			req.PlayerID = pid
		}
	}
	if req.PlayerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	awakening, err := h.setService.AwakenEquipment(c.Request.Context(), req.PlayerID, &req)
	if err != nil {
		h.log.Error("装备觉醒失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "觉醒成功", "data": awakening})
}

// GetAwakeningInfo 获取装备觉醒信息
// GET /api/v1/equipment/:equipmentID/awakening
func (h *EquipmentSetHandler) GetAwakeningInfo(c *gin.Context) {
	equipmentID, err := strconv.ParseInt(c.Param("equipmentID"), 10, 64)
	if err != nil || equipmentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的装备ID"})
		return
	}

	awakening, err := h.setService.GetAwakeningInfo(c.Request.Context(), equipmentID)
	if err != nil {
		h.log.Error("获取觉醒信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取觉醒信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": awakening})
}

// GetEquipmentDetail 获取装备详细信息
// GET /api/v1/equipment/details/:equipmentID
func (h *EquipmentSetHandler) GetEquipmentDetail(c *gin.Context) {
	equipmentID, err := strconv.ParseInt(c.Param("equipmentID"), 10, 64)
	if err != nil || equipmentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的装备ID"})
		return
	}

	// 从query中获取playerID
	playerIDStr := c.Query("player_id")
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	detail, err := h.setService.GetEquipmentDetail(c.Request.Context(), playerID, equipmentID)
	if err != nil {
		h.log.Error("获取装备详情失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": detail})
}

// CheckCanAwaken 检查装备能否觉醒
// GET /api/v1/equipment/:equipmentID/can-awaken
func (h *EquipmentSetHandler) CheckCanAwaken(c *gin.Context) {
	equipmentID, err := strconv.ParseInt(c.Param("equipmentID"), 10, 64)
	if err != nil || equipmentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的装备ID"})
		return
	}

	playerIDStr := c.Query("player_id")
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	canAwaken, msg, cost := h.setService.CanAwaken(c.Request.Context(), playerID, equipmentID)
	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"msg":        "success",
		"data": gin.H{
			"can_awaken": canAwaken,
			"message":    msg,
			"cost":       cost,
		},
	})
}
