// Package handler 黑市系统 HTTP 处理器
package handler

import (
	"log/slog"
	"net/http"
	"time"

	"cultivation-game/services/trade/internal/model"
	"cultivation-game/services/trade/internal/service"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// BlackMarketHandler
// ============================================================================

// BlackMarketHandler 黑市 HTTP 处理器
type BlackMarketHandler struct {
	svc *service.BlackMarketService
	log *slog.Logger
}

// NewBlackMarketHandler 创建 BlackMarketHandler
func NewBlackMarketHandler(svc *service.BlackMarketService, log *slog.Logger) *BlackMarketHandler {
	return &BlackMarketHandler{svc: svc, log: log}
}

// RegisterRoutes 注册黑市路由（兼容新旧路径）
func (h *BlackMarketHandler) RegisterRoutes(r *gin.RouterGroup) {
	// ---- 向后兼容的旧路径（前端使用的路径） ----
	r.GET("/api/v1/blackmarket/list", h.handleList)
	r.POST("/api/v1/blackmarket/buy", h.handleBuy)

	// ---- 新路径（带中划线） ----
	r.GET("/api/v1/black-market/items", h.handleList)
	r.POST("/api/v1/black-market/buy", h.handleBuy)
	r.POST("/api/v1/black-market/refresh", h.handleRefresh)
	r.GET("/api/v1/black-market/refresh-time", h.handleRefreshTime)
}

// ============================================================================
// 请求/响应类型
// ============================================================================

type buyItemRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
	PlayerID uint64 `json:"player_id"`
	VIPLevel int    `json:"vip_level"`
}

type refreshRequest struct {
	PlayerID uint64 `json:"player_id"`
}

// ============================================================================
// 处理器实现
// ============================================================================

// handleList 获取黑市物品列表
// GET /api/v1/blackmarket/list
// GET /api/v1/black-market/items
func (h *BlackMarketHandler) handleList(c *gin.Context) {
	items := h.svc.GetItemList()
	event := h.svc.GetActiveEvent()

	resp := gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"items":         items,
			"count":         len(items),
			"refresh_times": model.BlackMarketRefreshTimes,
		},
	}

	if event != nil {
		data := resp["data"].(gin.H)
		data["event"] = event
	}

	c.JSON(http.StatusOK, resp)
}

// handleBuy 购买黑市物品
// POST /api/v1/blackmarket/buy
// POST /api/v1/black-market/buy
func (h *BlackMarketHandler) handleBuy(c *gin.Context) {
	var req buyItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数格式错误",
		})
		return
	}
	if req.ItemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "物品 ID 不能为空",
		})
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	item, totalStone, totalJade, vipBonus, err := h.svc.BuyItem(req.ItemID, req.Quantity, req.PlayerID, req.VIPLevel)
	if err != nil {
		h.log.Warn("黑市购买失败",
			"item_id", req.ItemID,
			"player_id", req.PlayerID,
			"quantity", req.Quantity,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}

	resp := gin.H{
		"code": 0,
		"msg":  "购买成功",
		"data": gin.H{
			"item":         item,
			"quantity":     req.Quantity,
			"total_stone":  totalStone,
			"total_jade":   totalJade,
			"vip_bonus":    vipBonus,
			"vip_level":    req.VIPLevel,
			"vip_discount": service.VipDiscountRate(req.VIPLevel),
		},
	}

	c.JSON(http.StatusOK, resp)
}

// handleRefresh 强制刷新黑市（消耗仙玉）
// POST /api/v1/black-market/refresh
func (h *BlackMarketHandler) handleRefresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数格式错误",
		})
		return
	}

	items, cost, err := h.svc.ForceRefresh(req.PlayerID)
	if err != nil {
		h.log.Warn("黑市强制刷新失败", "player_id", req.PlayerID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}

	event := h.svc.GetActiveEvent()

	resp := gin.H{
		"code": 0,
		"msg":  "刷新成功",
		"data": gin.H{
			"items":           items,
			"count":           len(items),
			"cost":            cost,
			"jade_deducted":   cost,
			"refresh_times":   model.BlackMarketRefreshTimes,
		},
	}

	if event != nil {
		data := resp["data"].(gin.H)
		data["event"] = event
	}

	c.JSON(http.StatusOK, resp)
}

// handleRefreshTime 获取下次自动刷新时间
// GET /api/v1/black-market/refresh-time
func (h *BlackMarketHandler) handleRefreshTime(c *gin.Context) {
	nextRefresh := h.svc.GetRefreshTime()
	now := time.Now()

	remaining := nextRefresh.Unix() - now.Unix()
	if remaining < 0 {
		remaining = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"next_refresh":      nextRefresh.Unix(),
			"remaining_seconds": remaining,
			"refresh_times":     model.BlackMarketRefreshTimes,
			"refresh_interval":  service.RefreshIntervalHours * 3600,
		},
	})
}
