// Package handler 提供排行榜 HTTP 传输层。
// 将 HTTP 请求转换为 Service 层调用，返回 JSON 响应。
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"cultivation-game/services/ranking/internal/model"
	"cultivation-game/services/ranking/internal/service"

	"github.com/gin-gonic/gin"
)

// RankingHandler HTTP 请求处理器。
type RankingHandler struct {
	svc *service.RankingService
	log *slog.Logger
}

// NewRankingHandler 创建 RankingHandler。
func NewRankingHandler(svc *service.RankingService, log *slog.Logger) *RankingHandler {
	return &RankingHandler{svc: svc, log: log}
}

// RegisterRoutes 注册所有 HTTP 路由。
func (h *RankingHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v1/ranking/:type", h.handleGetRanking)
	r.GET("/api/v1/ranking/:type/player/:id", h.handleGetPlayerRank)
	r.GET("/api/v1/ranking/:type/top", h.handleGetTopPlayers)
}

// ============================================================
// 请求/响应结构体
// ============================================================

// apiResponse 通用 API 响应。
type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// rankingListData 排行榜分页响应数据。
type rankingListData struct {
	Entries []*model.RankingEntry `json:"entries"`
	Total   int32                 `json:"total"`
}

// playerRankData 玩家排名响应数据。
type playerRankData struct {
	Entry *model.RankingEntry   `json:"entry"`
	Above []*model.RankingEntry `json:"above"`
	Below []*model.RankingEntry `json:"below"`
}

// topPlayersData Top N 排行榜响应数据。
type topPlayersData struct {
	Entries []*model.RankingEntry `json:"entries"`
}

// healthData 健康检查响应数据。
type healthData struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

// ============================================================
// 辅助函数
// ============================================================

// writeJSON 写入 JSON 响应。
func writeJSON(c *gin.Context, statusCode int, resp *apiResponse) {
	c.JSON(statusCode, resp)
}

// writeError 写入错误响应。
func writeError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, &apiResponse{
		Code:    statusCode,
		Message: message,
	})
}

// ============================================================
// 处理器实现
// ============================================================

// handleGetRanking 获取排行榜（分页）。
// GET /api/v1/ranking/{type}?page=1&page_size=20
func (h *RankingHandler) handleGetRanking(c *gin.Context) {
	rankingType := model.RankingType(c.Param("type"))

	if !model.IsValidType(string(rankingType)) {
		writeError(c, http.StatusBadRequest, "无效的排行榜类型，可选值: realm, combat_power, wealth, sect")
		return
	}

	// 解析分页参数
	page := parsePageParams(c)

	entries, total, err := h.svc.GetRanking(c.Request.Context(), rankingType, page)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "获取排行榜失败", "type", rankingType, "error", err)
		writeError(c, http.StatusInternalServerError, "获取排行榜失败")
		return
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "success",
		Data: &rankingListData{
			Entries: entries,
			Total:   total,
		},
	})
}

// handleGetPlayerRank 获取玩家排名及周围邻居。
// GET /api/v1/ranking/{type}/player/{id}
func (h *RankingHandler) handleGetPlayerRank(c *gin.Context) {
	rankingType := model.RankingType(c.Param("type"))

	if !model.IsValidType(string(rankingType)) {
		writeError(c, http.StatusBadRequest, "无效的排行榜类型，可选值: realm, combat_power, wealth, sect")
		return
	}

	playerID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "无效的玩家 ID")
		return
	}

	entry, above, below, err := h.svc.GetPlayerRank(c.Request.Context(), rankingType, playerID)
	if err != nil {
		if errors.Is(err, service.ErrPlayerNotFound) {
			writeError(c, http.StatusNotFound, "玩家未上榜")
			return
		}
		h.log.ErrorContext(c.Request.Context(), "查询玩家排名失败",
			"type", rankingType, "player_id", playerID, "error", err)
		writeError(c, http.StatusInternalServerError, "查询玩家排名失败")
		return
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "success",
		Data: &playerRankData{
			Entry: entry,
			Above: above,
			Below: below,
		},
	})
}

// handleGetTopPlayers 获取 Top 100 缓存排行。
// GET /api/v1/ranking/{type}/top
func (h *RankingHandler) handleGetTopPlayers(c *gin.Context) {
	rankingType := model.RankingType(c.Param("type"))

	if !model.IsValidType(string(rankingType)) {
		writeError(c, http.StatusBadRequest, "无效的排行榜类型，可选值: realm, combat_power, wealth, sect")
		return
	}

	entries, err := h.svc.GetTopPlayers(c.Request.Context(), rankingType, 100)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "获取 Top 玩家失败", "type", rankingType, "error", err)
		writeError(c, http.StatusInternalServerError, "获取 Top 玩家失败")
		return
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "success",
		Data: &topPlayersData{
			Entries: entries,
		},
	})
}

// handleHealth 健康检查。
// GET /api/v1/health
func (h *RankingHandler) handleHealth(c *gin.Context) {
	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "ok",
		Data: &healthData{
			Service: "ranking-service",
			Status:  "running",
		},
	})
}

// ============================================================
// 工具函数
// ============================================================

// parsePageParams 从查询参数解析分页参数。
func parsePageParams(c *gin.Context) *model.PageRequest {
	page := int32(1)
	pageSize := int32(20)

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = int32(v)
		}
	}

	return &model.PageRequest{
		Page:     page,
		PageSize: pageSize,
	}
}
