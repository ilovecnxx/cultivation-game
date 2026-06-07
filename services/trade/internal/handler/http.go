// Package handler 实现 HTTP 传输层，将 HTTP REST 请求转换为 Service 层调用。
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"cultivation-game/services/trade/internal/model"
	"cultivation-game/services/trade/internal/service"
)

// ============================================================================
// Handler 定义
// ============================================================================

// TradeHandler 处理 HTTP 请求，调用 MarketService 和 AuctionService。
type TradeHandler struct {
	marketSvc  *service.MarketService
	auctionSvc *service.AuctionService
	log        *slog.Logger
}

// NewTradeHandler 创建 TradeHandler。
func NewTradeHandler(marketSvc *service.MarketService, auctionSvc *service.AuctionService, log *slog.Logger) *TradeHandler {
	return &TradeHandler{
		marketSvc:  marketSvc,
		auctionSvc: auctionSvc,
		log:        log,
	}
}

// RegisterRoutes 在路由器上注册所有交易服务路由。
func (h *TradeHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/v1/trade/listings", h.CreateListing)
	r.GET("/api/v1/trade/listings", h.GetListings)
	r.POST("/api/v1/trade/buy", h.BuyItem)
	r.POST("/api/v1/trade/cancel", h.CancelListing)
	r.GET("/api/v1/trade/my-listings", h.GetMyListings)
	r.GET("/api/v1/trade/transactions", h.GetTransactions)
	r.POST("/api/v1/trade/auction/start", h.StartAuction)
	r.POST("/api/v1/trade/auction/bid", h.PlaceBid)
	r.GET("/api/v1/trade/auctions", h.GetAuctions)
}

// ============================================================================
// 请求/响应类型
// ============================================================================

type createListingReq struct {
	SellerID     uint64 `json:"seller_id"`
	SellerName   string `json:"seller_name"`
	ItemID       uint32 `json:"item_id"`
	ItemName     string `json:"item_name"`
	Quantity     uint32 `json:"quantity"`
	UnitPrice    uint64 `json:"unit_price"`
	CurrencyType string `json:"currency_type"`
	ExpiresAt    int64  `json:"expires_at"`
}

type buyItemReq struct {
	ListingID uint64 `json:"listing_id"`
	BuyerID   uint64 `json:"buyer_id"`
	Quantity  uint32 `json:"quantity"`
}

type cancelListingReq struct {
	ListingID uint64 `json:"listing_id"`
	SellerID  uint64 `json:"seller_id"`
}

type startAuctionReq struct {
	ItemID          uint32 `json:"item_id"`
	SellerID        uint64 `json:"seller_id"`
	ReservePrice    uint64 `json:"reserve_price"`
	DurationSeconds uint32 `json:"duration_seconds"`
}

type placeBidReq struct {
	AuctionID uint64 `json:"auction_id"`
	BidderID  uint64 `json:"bidder_id"`
	BidAmount uint64 `json:"bid_amount"`
}

type listingResponse struct {
	Listing *model.Listing `json:"listing"`
}

type listingsListResponse struct {
	Listings []*model.Listing `json:"listings"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type transactionResponse struct {
	Transaction *model.Transaction `json:"transaction"`
}

type transactionsListResponse struct {
	Transactions []*model.Transaction `json:"transactions"`
	Total        int                  `json:"total"`
	Page         int                  `json:"page"`
	PageSize     int                  `json:"page_size"`
}

type auctionResponse struct {
	Auction *model.Auction `json:"auction"`
}

type auctionsListResponse struct {
	Auctions []*model.Auction `json:"auctions"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// ============================================================================
// 市场挂单
// ============================================================================

// CreateListing 创建市场挂单。
// POST /api/v1/trade/listings
func (h *TradeHandler) CreateListing(c *gin.Context) {
	var req createListingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}

	if req.SellerID == 0 {
		writeError(c, http.StatusBadRequest, "卖家 ID 不能为空")
		return
	}
	if req.ItemID == 0 {
		writeError(c, http.StatusBadRequest, "物品 ID 不能为空")
		return
	}
	if req.Quantity == 0 {
		writeError(c, http.StatusBadRequest, "数量必须大于 0")
		return
	}
	if req.UnitPrice == 0 {
		writeError(c, http.StatusBadRequest, "单价必须大于 0")
		return
	}

	var expiresAt time.Time
	if req.ExpiresAt > 0 {
		expiresAt = time.Unix(req.ExpiresAt, 0)
	}

	currencyType := model.CurrencyType(req.CurrencyType)
	if currencyType == "" {
		currencyType = model.CurrencySpiritStone
	}

	listing, err := h.marketSvc.CreateListing(c.Request.Context(), req.SellerID, req.SellerName, req.ItemID, req.ItemName, req.Quantity, req.UnitPrice, currencyType, expiresAt)
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "创建挂单失败", "seller_id", req.SellerID, "item_id", req.ItemID, "error", err)
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, &listingResponse{Listing: listing})
}

// GetListings 查询挂单列表（分页+筛选）。
// GET /api/v1/trade/listings?page=1&page_size=20&seller_id=&item_id=&status=&currency_type=
func (h *TradeHandler) GetListings(c *gin.Context) {


	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}

	var sellerID uint64
	if v := c.Query("seller_id"); v != "" {
		sellerID, _ = strconv.ParseUint(v, 10, 64)
	}
	var itemID uint32
	if v := c.Query("item_id"); v != "" {
		id, _ := strconv.ParseUint(v, 10, 32)
		itemID = uint32(id)
	}

	filter := model.ListingFilter{
		SellerID:     sellerID,
		ItemID:       itemID,
		Status:       model.ListingStatus(c.Query("status")),
		CurrencyType: model.CurrencyType(c.Query("currency_type")),
		Page:         page,
		PageSize:     pageSize,
	}

	listings, total, err := h.marketSvc.GetListings(c.Request.Context(), filter)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "查询挂单列表失败", "error", err)
		writeError(c, http.StatusInternalServerError, "查询挂单列表失败")
		return
	}

	writeJSON(c, http.StatusOK, &listingsListResponse{
		Listings: listings,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// BuyItem 购买物品。
// POST /api/v1/trade/buy
func (h *TradeHandler) BuyItem(c *gin.Context) {
	var req buyItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}

	if req.ListingID == 0 {
		writeError(c, http.StatusBadRequest, "挂单 ID 不能为空")
		return
	}
	if req.BuyerID == 0 {
		writeError(c, http.StatusBadRequest, "买家 ID 不能为空")
		return
	}
	if req.Quantity == 0 {
		writeError(c, http.StatusBadRequest, "购买数量必须大于 0")
		return
	}

	transaction, err := h.marketSvc.BuyItem(c.Request.Context(), req.ListingID, req.BuyerID, req.Quantity)
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "购买失败", "listing_id", req.ListingID, "buyer_id", req.BuyerID, "quantity", req.Quantity, "error", err)
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, &transactionResponse{Transaction: transaction})
}

// CancelListing 取消挂单。
// POST /api/v1/trade/cancel
func (h *TradeHandler) CancelListing(c *gin.Context) {
	var req cancelListingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}

	if req.ListingID == 0 {
		writeError(c, http.StatusBadRequest, "挂单 ID 不能为空")
		return
	}
	if req.SellerID == 0 {
		writeError(c, http.StatusBadRequest, "卖家 ID 不能为空")
		return
	}

	listing, err := h.marketSvc.CancelListing(c.Request.Context(), req.ListingID, req.SellerID)
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "取消挂单失败", "listing_id", req.ListingID, "seller_id", req.SellerID, "error", err)
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, &listingResponse{Listing: listing})
}

// GetMyListings 查询当前玩家的挂单。
// GET /api/v1/trade/my-listings?seller_id=&page=1&page_size=20
func (h *TradeHandler) GetMyListings(c *gin.Context) {


	sellerIDStr := c.Query("seller_id")
	if sellerIDStr == "" {
		writeError(c, http.StatusBadRequest, "seller_id 不能为空")
		return
	}
	sellerID, err := strconv.ParseUint(sellerIDStr, 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "无效的 seller_id")
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}

	filter := model.ListingFilter{
		SellerID: sellerID,
		Page:     page,
		PageSize: pageSize,
	}

	listings, total, err := h.marketSvc.GetListings(c.Request.Context(), filter)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "查询我的挂单失败", "error", err)
		writeError(c, http.StatusInternalServerError, "查询挂单失败")
		return
	}

	writeJSON(c, http.StatusOK, &listingsListResponse{
		Listings: listings,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetTransactions 查询交易记录。
// GET /api/v1/trade/transactions?buyer_id=&seller_id=&page=1&page_size=20
func (h *TradeHandler) GetTransactions(c *gin.Context) {


	var buyerID, sellerID uint64
	if v := c.Query("buyer_id"); v != "" {
		buyerID, _ = strconv.ParseUint(v, 10, 64)
	}
	if v := c.Query("seller_id"); v != "" {
		sellerID, _ = strconv.ParseUint(v, 10, 64)
	}

	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}

	transactions, total, err := h.marketSvc.GetTransactions(c.Request.Context(), buyerID, sellerID, page, pageSize)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "查询交易记录失败", "error", err)
		writeError(c, http.StatusInternalServerError, "查询交易记录失败")
		return
	}

	writeJSON(c, http.StatusOK, &transactionsListResponse{
		Transactions: transactions,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	})
}

// ============================================================================
// 拍卖
// ============================================================================

// StartAuction 发起拍卖。
// POST /api/v1/trade/auction/start
func (h *TradeHandler) StartAuction(c *gin.Context) {
	var req startAuctionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}

	if req.SellerID == 0 {
		writeError(c, http.StatusBadRequest, "卖家 ID 不能为空")
		return
	}
	if req.ItemID == 0 {
		writeError(c, http.StatusBadRequest, "物品 ID 不能为空")
		return
	}
	if req.ReservePrice == 0 {
		writeError(c, http.StatusBadRequest, "保留价必须大于 0")
		return
	}

	auction, err := h.auctionSvc.StartAuction(c.Request.Context(), req.ItemID, req.SellerID, req.ReservePrice, req.DurationSeconds)
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "创建拍卖失败", "seller_id", req.SellerID, "item_id", req.ItemID, "error", err)
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, &auctionResponse{Auction: auction})
}

// PlaceBid 出价。
// POST /api/v1/trade/auction/bid
func (h *TradeHandler) PlaceBid(c *gin.Context) {
	var req placeBidReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}

	if req.AuctionID == 0 {
		writeError(c, http.StatusBadRequest, "拍卖 ID 不能为空")
		return
	}
	if req.BidderID == 0 {
		writeError(c, http.StatusBadRequest, "出价者 ID 不能为空")
		return
	}
	if req.BidAmount == 0 {
		writeError(c, http.StatusBadRequest, "出价金额必须大于 0")
		return
	}

	auction, err := h.auctionSvc.PlaceBid(c.Request.Context(), req.AuctionID, req.BidderID, req.BidAmount)
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "出价失败", "auction_id", req.AuctionID, "bidder_id", req.BidderID, "bid_amount", req.BidAmount, "error", err)
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, &auctionResponse{Auction: auction})
}

// GetAuctions 获取拍卖列表。
// GET /api/v1/trade/auctions?page=1&page_size=20&item_id=
func (h *TradeHandler) GetAuctions(c *gin.Context) {


	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}

	var itemID uint32
	if v := c.Query("item_id"); v != "" {
		id, _ := strconv.ParseUint(v, 10, 32)
		itemID = uint32(id)
	}

	filter := model.AuctionFilter{
		ItemID:   itemID,
		Page:     page,
		PageSize: pageSize,
	}

	auctions, total, err := h.auctionSvc.GetActiveAuctions(c.Request.Context(), filter)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "查询拍卖列表失败", "error", err)
		writeError(c, http.StatusInternalServerError, "查询拍卖列表失败")
		return
	}

	writeJSON(c, http.StatusOK, &auctionsListResponse{
		Auctions: auctions,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// ============================================================================
// 辅助函数
// ============================================================================

// writeJSON 写入 JSON 响应。
func writeJSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// writeError 写入错误 JSON 响应。
func writeError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, &errorResponse{Error: message})
}

// writeServiceError 将 Service 层错误映射为 HTTP 状态码并写入响应。
func writeServiceError(c *gin.Context, err error) {
	c.JSON(httpStatusFromError(err), &errorResponse{Error: err.Error()})
}

// httpStatusFromError 将 Service 层业务错误映射为 HTTP 状态码。
func httpStatusFromError(err error) int {
	switch {
	case errors.Is(err, service.ErrListingNotFound), errors.Is(err, service.ErrAuctionNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrInvalidQuantity), errors.Is(err, service.ErrInvalidPrice),
		errors.Is(err, service.ErrBidTooLow), errors.Is(err, service.ErrBidBelowReserve),
		errors.Is(err, service.ErrBidBelowMinIncrement),
		errors.Is(err, service.ErrAuctionReserveTooLow),
		errors.Is(err, service.ErrAuctionDurationTooShort),
		errors.Is(err, service.ErrAuctionDurationTooLong):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrNotListingOwner):
		return http.StatusForbidden
	case errors.Is(err, service.ErrInsufficientGold):
		return http.StatusPaymentRequired
	case errors.Is(err, service.ErrConcurrentConflict):
		return http.StatusConflict
	case errors.Is(err, service.ErrListingNotActive),
		errors.Is(err, service.ErrListingAlreadySold),
		errors.Is(err, service.ErrListingAlreadyCancelled),
		errors.Is(err, service.ErrBuyOwnListing),
		errors.Is(err, service.ErrAuctionNotActive),
		errors.Is(err, service.ErrBidOwnAuction):
		return http.StatusConflict
	case errors.Is(err, service.ErrListingExpired), errors.Is(err, service.ErrAuctionAlreadyEnded):
		return http.StatusGone
	default:
		return http.StatusInternalServerError
	}
}
