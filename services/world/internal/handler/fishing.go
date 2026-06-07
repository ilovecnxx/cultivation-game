// Package handler 灵鱼垂钓系统 HTTP 处理器
package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/world/internal/service"

	"github.com/gin-gonic/gin"
)

// FishingHandler 钓鱼 HTTP 处理器
type FishingHandler struct {
	fishingSvc *service.FishingService
}

// NewFishingHandler 创建 FishingHandler
func NewFishingHandler(fishingSvc *service.FishingService) *FishingHandler {
	return &FishingHandler{fishingSvc: fishingSvc}
}

// RegisterRoutes 注册钓鱼相关路由
func (h *FishingHandler) RegisterRoutes(r *gin.Engine) {
	fishingGroup := r.Group("/api/v1/fishing")
	{
		fishingGroup.POST("/start", h.handleStartFishing)
		fishingGroup.POST("/cast", h.handleCastLine)
		fishingGroup.GET("/check-bite", h.handleCheckBite)
		fishingGroup.POST("/hook", h.handleHookFish)
		fishingGroup.POST("/tension", h.handleUpdateTension)
		fishingGroup.GET("/info", h.handleGetFishingInfo)
		fishingGroup.POST("/upgrade", h.handleUpgradeSkill)
		fishingGroup.POST("/buy-bait", h.handleBuyBait)
		fishingGroup.GET("/spots", h.handleGetSpots)
		fishingGroup.GET("/spot/:id", h.handleGetSpotDetail)
		fishingGroup.GET("/collection", h.handleGetFishCollection)
		fishingGroup.POST("/cancel", h.handleCancelFishing)
	}
}

// ============================================================
// 处理器实现
// ============================================================

// handleGetSpots 获取所有钓鱼点
// GET /api/v1/fishing/spots
func (h *FishingHandler) handleGetSpots(c *gin.Context) {
	spots := h.fishingSvc.GetSpots()
	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "success", "data": spots,
	})
}

// handleGetSpotDetail 获取钓鱼点详情（含可钓鱼类）
// GET /api/v1/fishing/spot/:id
func (h *FishingHandler) handleGetSpotDetail(c *gin.Context) {
	spotID := c.Param("id")
	spot := h.fishingSvc.GetSpotByID(spotID)
	if spot == nil {
		writeFishingJSON(c, http.StatusNotFound, gin.H{
			"code": 404, "msg": "钓鱼点不存在",
		})
		return
	}

	fish := h.fishingSvc.GetFishBySpot(spotID)
	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "success", "data": gin.H{
			"spot": spot,
			"fish": fish,
		},
	})
}

// handleStartFishing 开始钓鱼
// POST /api/v1/fishing/start
func (h *FishingHandler) handleStartFishing(c *gin.Context) {
	var req struct {
		PlayerID int64  `json:"player_id"`
		SpotID   string `json:"spot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 || req.SpotID == "" {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "参数错误",
		})
		return
	}

	session, err := h.fishingSvc.StartFishing(req.PlayerID, req.SpotID)
	if err != nil {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "开始垂钓", "data": session,
	})
}

// handleCastLine 抛竿
// POST /api/v1/fishing/cast
func (h *FishingHandler) handleCastLine(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "参数错误",
		})
		return
	}

	session, err := h.fishingSvc.CastLine(req.PlayerID)
	if err != nil {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "抛竿成功，等待鱼儿上钩", "data": session,
	})
}

// handleCheckBite 检查是否有鱼咬钩
// GET /api/v1/fishing/check-bite?player_id=xxx
func (h *FishingHandler) handleCheckBite(c *gin.Context) {
	pidStr := c.Query("player_id")
	playerID, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil || playerID <= 0 {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "无效的玩家ID",
		})
		return
	}

	session, err := h.fishingSvc.CheckBite(playerID)
	if err != nil {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "success", "data": session,
	})
}

// handleHookFish 收杆捕获
// POST /api/v1/fishing/hook
func (h *FishingHandler) handleHookFish(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "参数错误",
		})
		return
	}

	result, err := h.fishingSvc.HookFish(req.PlayerID)
	if err != nil {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": result.Message, "data": result,
	})
}

// handleUpdateTension 调整张力
// POST /api/v1/fishing/tension
func (h *FishingHandler) handleUpdateTension(c *gin.Context) {
	var req struct {
		PlayerID int64  `json:"player_id"`
		Action   string `json:"action"` // "pull", "release", "hold"
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "参数错误",
		})
		return
	}

	session, err := h.fishingSvc.UpdateTension(req.PlayerID, req.Action)
	if err != nil {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "success", "data": session,
	})
}

// handleGetFishingInfo 获取钓鱼信息
// GET /api/v1/fishing/info?player_id=xxx
func (h *FishingHandler) handleGetFishingInfo(c *gin.Context) {
	pidStr := c.Query("player_id")
	playerID, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil || playerID <= 0 {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "无效的玩家ID",
		})
		return
	}

	info := h.fishingSvc.GetFishingInfo(playerID)
	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "success", "data": info,
	})
}

// handleUpgradeSkill 升级钓鱼技能
// POST /api/v1/fishing/upgrade
func (h *FishingHandler) handleUpgradeSkill(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "参数错误",
		})
		return
	}

	newLevel, err := h.fishingSvc.UpgradeSkill(req.PlayerID)
	if err != nil {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "技能升级成功", "data": gin.H{
			"fishing_skill_level": newLevel,
		},
	})
}

// handleBuyBait 购买鱼饵
// POST /api/v1/fishing/buy-bait
func (h *FishingHandler) handleBuyBait(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id"`
		Amount   int   `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "参数错误",
		})
		return
	}

	if req.Amount <= 0 {
		req.Amount = 10 // 默认购买10个
	}

	playerFish, err := h.fishingSvc.BuyBait(req.PlayerID, req.Amount)
	if err != nil {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "购买成功", "data": playerFish,
	})
}

// handleGetFishCollection 获取鱼类图鉴
// GET /api/v1/fishing/collection?player_id=xxx
func (h *FishingHandler) handleGetFishCollection(c *gin.Context) {
	pidStr := c.Query("player_id")
	playerID, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil || playerID <= 0 {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "无效的玩家ID",
		})
		return
	}

	collection := h.fishingSvc.GetFishCollection(playerID)
	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "success", "data": collection,
	})
}

// handleCancelFishing 取消钓鱼
// POST /api/v1/fishing/cancel
func (h *FishingHandler) handleCancelFishing(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": "参数错误",
		})
		return
	}

	if err := h.fishingSvc.CancelFishing(req.PlayerID); err != nil {
		writeFishingJSON(c, http.StatusBadRequest, gin.H{
			"code": 400, "msg": err.Error(),
		})
		return
	}

	writeFishingJSON(c, http.StatusOK, gin.H{
		"code": 0, "msg": "已取消垂钓",
	})
}

// writeFishingJSON 写入 JSON 响应
func writeFishingJSON(c *gin.Context, statusCode int, resp interface{}) {
	c.JSON(statusCode, resp)
}
