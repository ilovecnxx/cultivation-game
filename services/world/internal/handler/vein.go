// Package handler 提供灵脉争夺系统的HTTP处理器
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"cultivation-game/services/world/internal/service"

	"github.com/gin-gonic/gin"
)

// VeinHandler 灵脉争夺HTTP处理器
type VeinHandler struct {
	veinSvc *service.SpiritVeinService
}

// NewVeinHandler 创建灵脉争夺处理器
func NewVeinHandler(veinSvc *service.SpiritVeinService) *VeinHandler {
	return &VeinHandler{veinSvc: veinSvc}
}

// RegisterRoutes 注册灵脉相关路由
func (h *VeinHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1/world/veins")
	{
		v1.GET("", h.handleListVeins)
		v1.GET("/:veinID", h.handleGetVein)
		v1.GET("/region/:regionID", h.handleGetRegionVeins)
		v1.GET("/my", h.handleGetMyVeins)
		v1.POST("/contest", h.handleInitiateContest)
		v1.POST("/contest/action", h.handleContestAction)
		v1.GET("/contest/:veinID", h.handleContestStatus)
		v1.POST("/abandon", h.handleAbandonVein)
		v1.POST("/upgrade", h.handleUpgradeVein)
		v1.POST("/discover", h.handleDiscoverVein)
		v1.POST("/occupy", h.handleOccupyVein)
		v1.POST("/collect", h.handleCollectYield)
	}
}

// ============================================================
// Request / Response types
// ============================================================

type veinContestRequest struct {
	UserID       string `json:"user_id"`
	UserName     string `json:"user_name"`
	VeinID       string `json:"vein_id"`
}

type contestActionRequest struct {
	UserID string `json:"user_id"`
	VeinID string `json:"vein_id"`
	Action string `json:"action"` // "attack", "skill", "heal"
	Damage int64  `json:"damage"`
}

type abandonRequest struct {
	UserID string `json:"user_id"`
	VeinID string `json:"vein_id"`
}

type upgradeRequest struct {
	OwnerID string `json:"owner_id"`
	VeinID  string `json:"vein_id"`
}

type discoverRequest struct {
	UserID string `json:"user_id"`
	VeinID string `json:"vein_id"`
	Method string `json:"method"` // "explore", "divination", "map_purchase"
}

type occupyRequest struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	VeinID   string `json:"vein_id"`
}

type collectRequest struct {
	VeinID string `json:"vein_id"`
	UserID string `json:"user_id"`
}

// ============================================================
// Handlers
// ============================================================

// handleListVeins 获取所有灵脉
// GET /api/v1/world/veins
func (h *VeinHandler) handleListVeins(c *gin.Context) {
	userID := c.Query("user_id")
	veins := h.veinSvc.GetAllVeins(userID)
	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取灵脉列表成功",
		Data:    veins,
	})
}

// handleGetVein 获取灵脉详情
// GET /api/v1/world/veins/:veinID
func (h *VeinHandler) handleGetVein(c *gin.Context) {
	veinID := c.Param("veinID")
	userID := c.Query("user_id")

	vein, err := h.veinSvc.GetVein(veinID, userID)
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取灵脉详情成功",
		Data:    vein,
	})
}

// handleGetRegionVeins 获取指定区域的灵脉
// GET /api/v1/world/veins/region/:regionID
func (h *VeinHandler) handleGetRegionVeins(c *gin.Context) {
	regionID := c.Param("regionID")
	userID := c.Query("user_id")

	veins := h.veinSvc.GetVeinsByRegion(regionID, userID)
	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: fmt.Sprintf("获取区域 %s 的灵脉成功", regionID),
		Data:    veins,
	})
}

// handleGetMyVeins 获取我的灵脉
// GET /api/v1/world/veins/my
func (h *VeinHandler) handleGetMyVeins(c *gin.Context) {
	userID := c.Query("user_id")
	sectID := c.Query("sect_id")

	veins := h.veinSvc.GetMyVeins(userID, sectID)
	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取我的灵脉成功",
		Data:    veins,
	})
}

// handleInitiateContest 发起灵脉争夺
// POST /api/v1/world/veins/contest
func (h *VeinHandler) handleInitiateContest(c *gin.Context) {
	var req veinContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" || req.VeinID == "" {
		writeError(c, http.StatusBadRequest, "user_id 和 vein_id 不能为空")
		return
	}
	userName := req.UserName
	if userName == "" {
		userName = "修士" + req.UserID
	}

	contest, err := h.veinSvc.InitiateContest(req.UserID, userName, req.VeinID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "发起灵脉争夺成功",
		Data:    contest,
	})
}

// handleContestAction 提交争夺行动
// POST /api/v1/world/veins/contest/action
func (h *VeinHandler) handleContestAction(c *gin.Context) {
	var req contestActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" || req.VeinID == "" || req.Action == "" {
		writeError(c, http.StatusBadRequest, "user_id, vein_id, action 不能为空")
		return
	}

	contest, err := h.veinSvc.SubmitContestAction(req.VeinID, req.UserID, req.Action, req.Damage)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "行动提交成功",
		Data:    contest,
	})
}

// handleContestStatus 获取争夺状态
// GET /api/v1/world/veins/contest/:veinID
func (h *VeinHandler) handleContestStatus(c *gin.Context) {
	veinID := c.Param("veinID")
	if veinID == "" {
		writeError(c, http.StatusBadRequest, "vein_id 不能为空")
		return
	}

	contest, err := h.veinSvc.GetContestStatus(veinID)
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取争夺状态成功",
		Data:    contest,
	})
}

// handleAbandonVein 放弃灵脉
// POST /api/v1/world/veins/abandon
func (h *VeinHandler) handleAbandonVein(c *gin.Context) {
	var req abandonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" || req.VeinID == "" {
		writeError(c, http.StatusBadRequest, "user_id 和 vein_id 不能为空")
		return
	}

	err := h.veinSvc.AbandonVein(req.UserID, req.VeinID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "放弃灵脉成功",
	})
}

// handleUpgradeVein 升级灵脉
// POST /api/v1/world/veins/upgrade
func (h *VeinHandler) handleUpgradeVein(c *gin.Context) {
	var req upgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.OwnerID == "" || req.VeinID == "" {
		writeError(c, http.StatusBadRequest, "owner_id 和 vein_id 不能为空")
		return
	}

	costStones, durationHours, endTime, err := h.veinSvc.UpgradeVein(req.OwnerID, req.VeinID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "开始灵脉升级",
		Data: map[string]interface{}{
			"cost_stones":    costStones,
			"duration_hours": durationHours,
			"end_time":       endTime,
		},
	})
}

// handleDiscoverVein 发现灵脉
// POST /api/v1/world/veins/discover
func (h *VeinHandler) handleDiscoverVein(c *gin.Context) {
	var req discoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" || req.VeinID == "" {
		writeError(c, http.StatusBadRequest, "user_id 和 vein_id 不能为空")
		return
	}
	method := req.Method
	if method == "" {
		method = "explore"
	}

	vein, rewardStones, err := h.veinSvc.DiscoverVein(req.UserID, req.VeinID, method)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: fmt.Sprintf("发现灵脉成功，获得 %d 灵石奖励", rewardStones),
		Data: map[string]interface{}{
			"vein":          vein,
			"reward_stones": rewardStones,
		},
	})
}

// handleOccupyVein 占领无人灵脉
// POST /api/v1/world/veins/occupy
func (h *VeinHandler) handleOccupyVein(c *gin.Context) {
	var req occupyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" || req.VeinID == "" {
		writeError(c, http.StatusBadRequest, "user_id 和 vein_id 不能为空")
		return
	}
	userName := req.UserName
	if userName == "" {
		userName = "修士" + req.UserID
	}

	err := h.veinSvc.OccupyVein(req.UserID, userName, req.VeinID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "占领灵脉成功",
	})
}

// handleCollectYield 收取灵脉产出
// POST /api/v1/world/veins/collect
func (h *VeinHandler) handleCollectYield(c *gin.Context) {
	var req collectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.VeinID == "" {
		writeError(c, http.StatusBadRequest, "vein_id 不能为空")
		return
	}

	yield, err := h.veinSvc.CollectYield(req.VeinID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: fmt.Sprintf("收取灵石成功，获得 %d 灵石", yield),
		Data: map[string]interface{}{
			"yield":     yield,
			"vein_id":   req.VeinID,
		},
	})
}

// getQueryInt 获取查询参数中的整数
func getQueryInt(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}
