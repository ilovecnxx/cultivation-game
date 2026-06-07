// Package handler 势力争霸 HTTP 处理器
//
// API:
//   GET  /api/v1/sect/war/status   - 获取争霸状态
//   POST /api/v1/sect/war/enroll   - 报名争霸
//   GET  /api/v1/sect/war/ranking  - 排行榜
//   GET  /api/v1/sect/war/history  - 历史记录
package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// FactionWarHandler 势力争霸 HTTP 处理器
type FactionWarHandler struct {
	svc *service.FactionWarService
}

// NewFactionWarHandler 创建势力争霸处理器
func NewFactionWarHandler(svc *service.FactionWarService) *FactionWarHandler {
	return &FactionWarHandler{svc: svc}
}

// GetWarStatus 获取势力争霸状态
// @Router GET /api/v1/sect/war/status [get]
func (h *FactionWarHandler) GetWarStatus(c *gin.Context) {
	sectID := c.Query("sect_id")
	if sectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sect_id 不能为空"})
		return
	}

	status, err := h.svc.GetStatus(c.Request.Context(), sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": status})
}

// EnrollWar 报名势力争霸
// @Router POST /api/v1/sect/war/enroll [post]
func (h *FactionWarHandler) EnrollWar(c *gin.Context) {
	var req struct {
		SectID    string   `json:"sect_id"`
		UserID    string   `json:"user_id"`
		PlayerIDs []string `json:"player_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if len(req.SectID) == 0 || len(req.UserID) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sect_id 和 user_id 不能为空"})
		return
	}
	if len(req.PlayerIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择参战弟子"})
		return
	}

	record, err := h.svc.Enroll(c.Request.Context(), req.SectID, req.UserID, req.PlayerIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": record, "message": "报名成功"})
}

// GetRanking 获取势力争霸排行榜
// @Router GET /api/v1/sect/war/ranking [get]
func (h *FactionWarHandler) GetRanking(c *gin.Context) {
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

	records, total, err := h.svc.GetRanking(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  records,
		"total": total,
		"page":  page,
	})
}

// GetHistory 获取宗门势力争霸历史
// @Router GET /api/v1/sect/war/history [get]
func (h *FactionWarHandler) GetHistory(c *gin.Context) {
	sectID := c.Query("sect_id")
	if sectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sect_id 不能为空"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.ParseInt(limitStr, 10, 64)
	if limit < 1 || limit > 50 {
		limit = 10
	}

	records, err := h.svc.GetHistory(c.Request.Context(), sectID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": records})
}
