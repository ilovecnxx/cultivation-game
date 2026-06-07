// Package handler 实现仙玉商城的 HTTP 传输层，将 REST 请求转换为 Service 调用。
package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"cultivation-game/services/trade/internal/shop/model"
	"cultivation-game/services/trade/internal/shop/service"
)

// ============================================================================
// Handler 定义
// ============================================================================

// ShopHandler 处理商城 HTTP 请求。
type ShopHandler struct {
	svc *service.ShopService
	log *slog.Logger
}

// NewShopHandler 创建 ShopHandler。
func NewShopHandler(svc *service.ShopService, log *slog.Logger) *ShopHandler {
	return &ShopHandler{svc: svc, log: log}
}

// RegisterRoutes 在路由器上注册商城路由。
func (h *ShopHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/api/v1/shop/items", h.GetItems)
	r.POST("/api/v1/shop/buy", h.BuyItem)
	r.GET("/api/v1/shop/vip", h.GetVIPInfo)
	r.POST("/api/v1/shop/vip/claim", h.ClaimVIP)
	r.POST("/api/v1/shop/recharge", h.Recharge)
	r.POST("/api/v1/shop/chest/open", h.OpenChest)
}

// ============================================================================
// 请求/响应类型
// ============================================================================

type errorResponse struct {
	Error string `json:"error"`
}

type itemsResponse struct {
	Items      []model.ShopItem `json:"items"`
	Categories []string         `json:"categories"`
	Total      int              `json:"total"`
}

type buyRequest struct {
	PlayerID uint64 `json:"player_id"`
	ItemID   uint32 `json:"item_id"`
	Quantity uint32 `json:"quantity"`
}

type vipRequest struct {
	PlayerID uint64 `json:"player_id"`
}

type rechargeRequest struct {
	PlayerID uint64 `json:"player_id"`
	Amount   uint64 `json:"amount"`
}

type chestOpenRequest struct {
	PlayerID uint64 `json:"player_id"`
	ItemID   uint32 `json:"item_id"`
}

// ============================================================================
// API 实现
// ============================================================================

// GetItems 获取商品列表。
// GET /api/v1/shop/items?category=丹药
func (h *ShopHandler) GetItems(c *gin.Context) {
	category := c.Query("category")

	items := h.svc.GetItems(c.Request.Context(), category)
	categories := h.svc.GetCategories()

	writeJSON(c, http.StatusOK, itemsResponse{
		Items:      items,
		Categories: categories,
		Total:      len(items),
	})
}

// BuyItem 购买商品。
// POST /api/v1/shop/buy
func (h *ShopHandler) BuyItem(c *gin.Context) {
	var req buyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}

	if req.PlayerID == 0 {
		writeError(c, http.StatusBadRequest, "玩家 ID 不能为空")
		return
	}
	if req.ItemID == 0 {
		writeError(c, http.StatusBadRequest, "商品 ID 不能为空")
		return
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}

	buyReq := &model.BuyReq{
		PlayerID: req.PlayerID,
		ItemID:   req.ItemID,
		Quantity: req.Quantity,
	}

	resp, err := h.svc.Buy(c.Request.Context(), buyReq)
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "购买失败",
			"player_id", req.PlayerID,
			"item_id", req.ItemID,
			"error", err,
		)
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, resp)
}

// GetVIPInfo 获取 VIP 信息。
// GET /api/v1/shop/vip?player_id=1
func (h *ShopHandler) GetVIPInfo(c *gin.Context) {
	playerIDStr := c.Query("player_id")
	if playerIDStr == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}
	playerID, err := strconv.ParseUint(playerIDStr, 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "无效的 player_id")
		return
	}

	info, err := h.svc.GetVIPInfo(c.Request.Context(), playerID)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "查询 VIP 信息失败", "error", err)
		writeError(c, http.StatusInternalServerError, "查询 VIP 信息失败")
		return
	}

	// 同时返回 VIP 等级配置列表供前端展示升级进度
	vipCfgs := h.svc.GetAllVIPConfigs()

	writeJSON(c, http.StatusOK, map[string]interface{}{
		"vip_info":    info,
		"vip_configs": vipCfgs,
	})
}

// ClaimVIP 领取 VIP 每日奖励。
// POST /api/v1/shop/vip/claim
func (h *ShopHandler) ClaimVIP(c *gin.Context) {
	var req vipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}
	if req.PlayerID == 0 {
		writeError(c, http.StatusBadRequest, "玩家 ID 不能为空")
		return
	}

	resp, err := h.svc.ClaimVIPDaily(c.Request.Context(), req.PlayerID)
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "领取 VIP 奖励失败",
			"player_id", req.PlayerID,
			"error", err,
		)
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, resp)
}

// Recharge 模拟充值。
// POST /api/v1/shop/recharge
func (h *ShopHandler) Recharge(c *gin.Context) {
	var req rechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}
	if req.PlayerID == 0 {
		writeError(c, http.StatusBadRequest, "玩家 ID 不能为空")
		return
	}
	if req.Amount == 0 {
		writeError(c, http.StatusBadRequest, "充值金额必须大于 0")
		return
	}

	resp, err := h.svc.Recharge(c.Request.Context(), &model.RechargeReq{
		PlayerID: req.PlayerID,
		Amount:   req.Amount,
	})
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "充值失败",
			"player_id", req.PlayerID,
			"amount", req.Amount,
			"error", err,
		)
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, resp)
}

// OpenChest 打开宝箱。
// POST /api/v1/shop/chest/open
func (h *ShopHandler) OpenChest(c *gin.Context) {
	var req chestOpenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}
	if req.PlayerID == 0 {
		writeError(c, http.StatusBadRequest, "玩家 ID 不能为空")
		return
	}
	if req.ItemID == 0 {
		writeError(c, http.StatusBadRequest, "商品 ID 不能为空")
		return
	}

	rewards, err := h.svc.OpenChest(c.Request.Context(), req.PlayerID, req.ItemID)
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "开箱失败",
			"player_id", req.PlayerID,
			"item_id", req.ItemID,
			"error", err,
		)
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, map[string]interface{}{
		"success": true,
		"rewards": rewards,
	})
}

// ============================================================================
// 辅助函数
// ============================================================================

func writeJSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

func writeError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, &errorResponse{Error: message})
}

// writeServiceError 将 Service 层错误映射为 HTTP 状态码并写入响应。
func writeServiceError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case isError(err, service.ErrItemNotFound):
		status = http.StatusNotFound
	case isError(err, service.ErrInsufficientJade),
		isError(err, service.ErrInsufficientStone):
		status = http.StatusPaymentRequired
	case isError(err, service.ErrOutOfStock),
		isError(err, service.ErrLimitReached),
		isError(err, service.ErrInvalidQuantity),
		isError(err, service.ErrVipLevelRequired),
		isError(err, service.ErrNotVip),
		isError(err, service.ErrAlreadyClaimed):
		status = http.StatusBadRequest
	}
	c.JSON(status, &errorResponse{Error: err.Error()})
}

func isError(err, target error) bool {
	return err.Error() == target.Error()
}
