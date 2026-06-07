package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// EquipmentHandler 装备 HTTP 处理器
type EquipmentHandler struct {
	inventoryService *service.InventoryService
	log              *zap.Logger
}

// NewEquipmentHandler 创建 EquipmentHandler
func NewEquipmentHandler(inventoryService *service.InventoryService, log *zap.Logger) *EquipmentHandler {
	return &EquipmentHandler{inventoryService: inventoryService, log: log}
}

// ListEquipment 查询已装备列表
// GET /api/v1/player/:id/equipment
func (h *EquipmentHandler) ListEquipment(c *gin.Context) {
	playerID := h.equipParseID(c)
	if playerID == 0 {
		return
	}

	equipments, err := h.inventoryService.GetEquipment(c.Request.Context(), playerID)
	if err != nil {
		h.log.Error("查询装备失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询装备失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": equipments})
}

// Equip 穿戴装备
// POST /api/v1/player/:id/equipment/equip
func (h *EquipmentHandler) Equip(c *gin.Context) {
	playerID := h.equipParseID(c)
	if playerID == 0 {
		return
	}

	var req model.EquipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	equipment, err := h.inventoryService.EquipItem(c.Request.Context(), playerID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "穿戴成功", "data": equipment})
}

// Unequip 卸下装备
// POST /api/v1/player/:id/equipment/unequip
func (h *EquipmentHandler) Unequip(c *gin.Context) {
	playerID := h.equipParseID(c)
	if playerID == 0 {
		return
	}

	var req model.UnequipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	invItem, err := h.inventoryService.UnequipItem(c.Request.Context(), playerID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "卸下成功", "data": invItem})
}

// Strengthen 强化装备
// POST /api/v1/player/:id/equipment/strengthen
func (h *EquipmentHandler) Strengthen(c *gin.Context) {
	playerID := h.equipParseID(c)
	if playerID == 0 {
		return
	}

	var req model.StrengthenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	equipment, err := h.inventoryService.StrengthenEquipment(c.Request.Context(), playerID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "强化成功", "data": equipment})
}

func (h *EquipmentHandler) equipParseID(c *gin.Context) int64 {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return 0
	}
	return playerID
}
