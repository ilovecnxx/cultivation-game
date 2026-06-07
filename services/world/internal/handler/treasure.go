// Package handler 藏宝图系统 HTTP 处理器
package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/world/internal/service"

	"github.com/gin-gonic/gin"
)

// TreasureHandler 藏宝图 HTTP 处理器
type TreasureHandler struct {
	treasureSvc *service.TreasureService
}

// NewTreasureHandler 创建 TreasureHandler
func NewTreasureHandler(treasureSvc *service.TreasureService) *TreasureHandler {
	return &TreasureHandler{treasureSvc: treasureSvc}
}

// RegisterRoutes 注册藏宝图路由
func (h *TreasureHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v1/treasure/fragments", h.handleGetFragments)
	r.POST("/api/v1/treasure/combine", h.handleCombine)
	r.POST("/api/v1/treasure/dig", h.handleDig)
}

// writeTreasureJSON 写入 JSON 响应
func writeTreasureJSON(c *gin.Context, statusCode int, resp interface{}) {
	c.JSON(statusCode, resp)
}

// handleGetFragments 获取碎片状态
// GET /api/v1/treasure/fragments?player_id=xxx
func (h *TreasureHandler) handleGetFragments(c *gin.Context) {
	pidStr := c.Query("player_id")
	playerID, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil || playerID <= 0 {
		writeTreasureJSON(c, http.StatusBadRequest, map[string]interface{}{
			"code": 400, "msg": "无效的玩家ID",
		})
		return
	}

	data := h.treasureSvc.GetFragments(playerID)
	writeTreasureJSON(c, http.StatusOK, map[string]interface{}{
		"code": 0, "msg": "success", "data": data,
	})
}

// handleCombine 拼合藏宝图
// POST /api/v1/treasure/combine
func (h *TreasureHandler) handleCombine(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 {
		writeTreasureJSON(c, http.StatusBadRequest, map[string]interface{}{
			"code": 400, "msg": "参数错误",
		})
		return
	}

	result, err := h.treasureSvc.CombineFragments(req.PlayerID)
	if err != nil {
		writeTreasureJSON(c, http.StatusBadRequest, map[string]interface{}{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeTreasureJSON(c, http.StatusOK, map[string]interface{}{
		"code": 0, "msg": "藏宝图拼合成功", "data": result,
	})
}

// handleDig 挖宝
// POST /api/v1/treasure/dig
func (h *TreasureHandler) handleDig(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 {
		writeTreasureJSON(c, http.StatusBadRequest, map[string]interface{}{
			"code": 400, "msg": "参数错误",
		})
		return
	}

	result, err := h.treasureSvc.DigTreasure(req.PlayerID)
	if err != nil {
		writeTreasureJSON(c, http.StatusBadRequest, map[string]interface{}{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeTreasureJSON(c, http.StatusOK, map[string]interface{}{
		"code": 0, "msg": "挖宝成功", "data": result,
	})
}
