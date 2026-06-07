package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PlayerHandler 玩家 HTTP 处理器
type PlayerHandler struct {
	playerService    *service.PlayerService
	inventoryService *service.InventoryService
	log              *zap.Logger
}

// NewPlayerHandler 创建 PlayerHandler
func NewPlayerHandler(playerService *service.PlayerService, inventoryService *service.InventoryService, log *zap.Logger) *PlayerHandler {
	return &PlayerHandler{
		playerService:    playerService,
		inventoryService: inventoryService,
		log:              log,
	}
}

// Register 注册并创建角色
// POST /api/v1/player/register
func (h *PlayerHandler) Register(c *gin.Context) {
	var req model.CreatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	player, err := h.playerService.CreatePlayer(c.Request.Context(), &req)
	if err != nil {
		h.log.Warn("创建角色失败", zap.Error(err))
		c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "创建成功",
		"data": player,
	})
}

// GetProfile 查询玩家属性
// GET /api/v1/player/:id
func (h *PlayerHandler) GetProfile(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.playerService.GetPlayerWithDetails(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询玩家失败", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// UpdateProfile 更新玩家属性
// PUT /api/v1/player/:id
func (h *PlayerHandler) UpdateProfile(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var updates map[string]any
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 获取当前玩家
	player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "玩家不存在"})
		return
	}

	// 更新允许的字段
	if v, ok := updates["name"].(string); ok && v != "" {
		player.Name = v
	}
	if v, ok := updates["spirit_root"]; ok {
		if vr, ok := v.(float64); ok {
			player.SpiritRoot = int32(vr)
		}
	}

	if err := h.playerService.UpdatePlayer(c.Request.Context(), player); err != nil {
		h.log.Error("更新玩家失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新成功"})
}

// GetByUserID 根据用户ID获取玩家
// GET /api/v1/player/user/:user_id
func (h *PlayerHandler) GetByUserID(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "用户ID为空"})
		return
	}

	player, err := h.playerService.GetPlayerByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": player})
}

// UpdateRealm 更新境界和属性（由修炼服务突破成功后调用）
// POST /api/v1/player/:id/update-realm
func (h *PlayerHandler) UpdateRealm(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		RealmID    int32 `json:"realm_id"`
		RealmLevel int32 `json:"realm_level"`
		Attack     int64 `json:"attack"`
		Defense    int64 `json:"defense"`
		MaxHP      int64 `json:"max_hp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	if err := h.playerService.UpdateRealm(c.Request.Context(), playerID, req.RealmID, req.RealmLevel, req.Attack, req.Defense, req.MaxHP); err != nil {
		h.log.Error("更新境界失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新境界失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "境界更新成功"})
}

// AddExp 增加经验（由战斗/世界等服务调用）
// POST /api/v1/player/:id/add-exp
func (h *PlayerHandler) AddExp(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		Exp int64 `json:"exp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	player, err := h.playerService.AddExp(c.Request.Context(), playerID, req.Exp)
	if err != nil {
		h.log.Error("增加经验失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "增加经验失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "经验增加成功",
		"data": gin.H{"total_exp": player.Experience},
	})
}

// UpdateCurrency 货币变更
// POST /api/v1/player/:id/currency
func (h *PlayerHandler) UpdateCurrency(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.CurrencyChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	player, err := h.playerService.UpdateCurrency(c.Request.Context(), playerID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"gold": player.Gold, "bound_gold": player.BoundGold, "jade": player.Jade,
		},
	})
}

// UpdateExp 增加修为（由修炼服务修炼成功后调用）
// POST /api/v1/player/:id/update-exp
func (h *PlayerHandler) UpdateExp(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		SpiritPower int64 `json:"spirit_power"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	if req.SpiritPower <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "修为值必须大于0"})
		return
	}

	player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "玩家不存在"})
		return
	}

	player.SpiritPower += req.SpiritPower
	if err := h.playerService.UpdatePlayer(c.Request.Context(), player); err != nil {
		h.log.Error("更新修为失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新修为失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "修为增加成功",
		"data": gin.H{
			"spirit_power": player.SpiritPower,
			"added":        req.SpiritPower,
		},
	})
}

// UpdateAttributes 更新属性（境界/攻防血速，由修炼服务突破成功后调用）
// POST /api/v1/player/:id/update-attributes
func (h *PlayerHandler) UpdateAttributes(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		RealmID    int32 `json:"realm_id"`
		RealmLevel int32 `json:"realm_level"`
		Attack     int64 `json:"attack"`
		Defense    int64 `json:"defense"`
		MaxHP      int64 `json:"max_hp"`
		Speed      int64 `json:"speed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	if err := h.playerService.UpdateRealm(c.Request.Context(), playerID, req.RealmID, req.RealmLevel, req.Attack, req.Defense, req.MaxHP); err != nil {
		h.log.Error("更新属性失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新属性失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "属性更新成功"})
}

// AddItem 添加物品到背包（由战斗/世界/炼丹等服务调用）
// POST /api/v1/player/:id/add-item
func (h *PlayerHandler) AddItem(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		ItemID   int64 `json:"item_id" binding:"required"`
		Quantity int32 `json:"quantity" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	items, err := h.inventoryService.AddItem(c.Request.Context(), playerID, req.ItemID, req.Quantity)
	if err != nil {
		h.log.Warn("添加物品失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "添加物品成功", "data": items})
}

// RemoveItem 从背包移除物品
// POST /api/v1/player/:id/remove-item
func (h *PlayerHandler) RemoveItem(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		InventoryItemID int64 `json:"inventory_item_id" binding:"required"`
		Quantity        int32 `json:"quantity" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	if err := h.inventoryService.RemoveItem(c.Request.Context(), playerID, req.InventoryItemID, req.Quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "移除物品成功"})
}

// GetAttributes 获取玩家完整属性
// GET /api/v1/player/:id/attributes
func (h *PlayerHandler) GetAttributes(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.playerService.GetPlayerWithDetails(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询玩家属性失败", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// MeditationAction 修炼打坐动作（开始/结束）
// POST /api/v1/player/:id/meditation
// TODO: 所有权校验 — 应从 JWT claims 提取 player_id 并与 URL :id 比对
func (h *PlayerHandler) MeditationAction(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		Action      string `json:"action" binding:"required"`
		DurationMin int    `json:"duration_min"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	if req.Action == "stop" && req.DurationMin > 0 {
		player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "玩家不存在"})
			return
		}

		gain := int64(req.DurationMin) * int64(player.Level) * 12
		player.SpiritPower += gain

		if err := h.playerService.UpdatePlayer(c.Request.Context(), player); err != nil {
			h.log.Error("更新修为失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新修为失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "success",
			"data": gin.H{
				"spirit_power": player.SpiritPower,
				"gained":       gain,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
