// Package handler 宗门仓库 HTTP 处理器
package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// SectWarehouseHandler 宗门仓库 HTTP 处理器
type SectWarehouseHandler struct {
	svc *service.SectWarehouseService
}

// NewSectWarehouseHandler 创建宗门仓库处理器
func NewSectWarehouseHandler(svc *service.SectWarehouseService) *SectWarehouseHandler {
	return &SectWarehouseHandler{svc: svc}
}

// DonateItem 捐献物品到仓库
// @Router POST /api/v1/sect/warehouse/donate [post]
func (h *SectWarehouseHandler) DonateItem(c *gin.Context) {
	var req struct {
		SectID      string `json:"sect_id"`
		UserID      string `json:"user_id"`
		UserName    string `json:"user_name"`
		ItemName    string `json:"item_name"`
		ItemType    string `json:"item_type"`
		ItemIcon    string `json:"item_icon"`
		Quantity    int    `json:"quantity"`
		MarketValue int64  `json:"market_value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	item, err := h.svc.DonateItem(c.Request.Context(), req.SectID, req.UserID, req.UserName,
		req.ItemName, req.ItemType, req.ItemIcon, req.Quantity, req.MarketValue)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item, "message": "捐献成功"})
}

// GetWarehouseItems 获取仓库物品列表
// @Router GET /api/v1/sect/warehouse/list [get]
func (h *SectWarehouseHandler) GetWarehouseItems(c *gin.Context) {
	sectID := c.Query("sect_id")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.ParseInt(pageStr, 10, 64)
	pageSize, _ := strconv.ParseInt(pageSizeStr, 10, 64)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	items, total, err := h.svc.GetWarehouseItems(c.Request.Context(), sectID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "total": total})
}

// BuyItem 购买仓库物品
// @Router POST /api/v1/sect/warehouse/buy [post]
func (h *SectWarehouseHandler) BuyItem(c *gin.Context) {
	var req struct {
		ItemID   string `json:"item_id"`
		SectID   string `json:"sect_id"`
		UserID   string `json:"user_id"`
		Currency string `json:"currency"` // "spirit" or "contribution"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.BuyItem(c.Request.Context(), req.ItemID, req.SectID, req.UserID, req.Currency); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "购买成功"})
}

// DonateFunds 捐献灵石给宗门
// @Router POST /api/v1/sect/warehouse/donate-funds [post]
func (h *SectWarehouseHandler) DonateFunds(c *gin.Context) {
	var req struct {
		SectID string `json:"sect_id"`
		UserID string `json:"user_id"`
		Amount int64  `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	amount, err := h.svc.DonateFunds(c.Request.Context(), req.SectID, req.UserID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "捐献成功", "data": gin.H{"funds_added": amount}})
}
