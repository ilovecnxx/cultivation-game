// Package handler 实现 GM 管理后台 HTTP 处理器。
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"cultivation-game/services/auth/internal/model"
	"cultivation-game/services/auth/internal/service"

	"github.com/gin-gonic/gin"
)

// GMHandler GM 管理后台 HTTP 处理器。
type GMHandler struct {
	gmSvc *service.GMService
	log   *slog.Logger
}

// NewGMHandler 创建 GMHandler。
func NewGMHandler(gmSvc *service.GMService, log *slog.Logger) *GMHandler {
	return &GMHandler{gmSvc: gmSvc, log: log}
}

// ---- 辅助 ----

// success 返回成功响应。
func success(c *gin.Context, data interface{}) {
	if data == nil {
		c.JSON(http.StatusOK, model.GMAPIResponse{Code: 0, Message: "success"})
		return
	}
	c.JSON(http.StatusOK, model.GMAPIResponse{Code: 0, Message: "success", Data: data})
}

// fail 返回失败响应。
func fail(c *gin.Context, httpCode int, msg string) {
	c.JSON(httpCode, model.GMAPIResponse{Code: -1, Message: msg})
}

// parseID 从路径参数解析 uint64 ID。
func parseID(c *gin.Context, param string) (uint64, error) {
	return strconv.ParseUint(c.Param(param), 10, 64)
}

// getGMToken 从请求头中提取 GM Token。
func getGMToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return auth
}

// ---- 认证 ----

// GM GM 登录。
func (h *GMHandler) Login(c *gin.Context) {
	var req model.GMLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if req.Username == "" || req.Password == "" {
		fail(c, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	resp, err := h.gmSvc.AuthenticateGM(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.log.WarnContext(c.Request.Context(), "GM 登录失败", "username", req.Username, "error", err)
		if errors.Is(err, service.ErrGMInvalidCredentials) {
			fail(c, http.StatusUnauthorized, "用户名或密码错误")
			return
		}
		if errors.Is(err, service.ErrGMDisabled) {
			fail(c, http.StatusForbidden, "管理员账号已被禁用")
			return
		}
		fail(c, http.StatusInternalServerError, "登录失败")
		return
	}

	success(c, resp)
}

// ---- 玩家管理 ----

// GetPlayerList 获取玩家列表/搜索玩家。
func (h *GMHandler) GetPlayerList(c *gin.Context) {
	search := c.DefaultQuery("search", "")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	resp, err := h.gmSvc.GetPlayerList(c.Request.Context(), search, page, limit)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "查询玩家列表失败", "error", err)
		fail(c, http.StatusInternalServerError, "查询玩家列表失败")
		return
	}

	success(c, resp)
}

// GetPlayerDetail 获取玩家详情。
func (h *GMHandler) GetPlayerDetail(c *gin.Context) {
	playerID, err := parseID(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的玩家 ID")
		return
	}

	player, err := h.gmSvc.GetPlayerDetail(c.Request.Context(), playerID)
	if err != nil {
		if errors.Is(err, service.ErrGMPLayerNotFound) {
			fail(c, http.StatusNotFound, "玩家不存在")
			return
		}
		h.log.ErrorContext(c.Request.Context(), "查询玩家详情失败", "error", err, "player_id", playerID)
		fail(c, http.StatusInternalServerError, "查询玩家详情失败")
		return
	}

	success(c, player)
}

// EditPlayerAttribute 修改玩家属性。
func (h *GMHandler) EditPlayerAttribute(c *gin.Context) {
	playerID, err := parseID(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的玩家 ID")
		return
	}

	var req model.GMEditAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if req.Field == "" {
		fail(c, http.StatusBadRequest, "属性名不能为空")
		return
	}

	adminID := c.GetUint64("admin_id")
	if err := h.gmSvc.EditPlayerAttribute(c.Request.Context(), playerID, adminID, req.Field, req.Value); err != nil {
		h.log.ErrorContext(c.Request.Context(), "修改玩家属性失败", "error", err, "player_id", playerID, "field", req.Field)
		if errors.Is(err, service.ErrGMPermissionDenied) {
			fail(c, http.StatusForbidden, "权限不足")
			return
		}
		fail(c, http.StatusInternalServerError, "修改属性失败")
		return
	}

	success(c, nil)
}

// BanPlayer 封禁玩家。
func (h *GMHandler) BanPlayer(c *gin.Context) {
	playerID, err := parseID(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的玩家 ID")
		return
	}

	var req model.GMBanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if req.Reason == "" {
		fail(c, http.StatusBadRequest, "封禁原因不能为空")
		return
	}

	if req.BanType < 1 || req.BanType > 3 {
		fail(c, http.StatusBadRequest, "无效的封禁类型")
		return
	}

	adminID := c.GetUint64("admin_id")
	if err := h.gmSvc.BanPlayer(c.Request.Context(), playerID, adminID, req.Reason, req.BanType, req.Duration); err != nil {
		h.log.ErrorContext(c.Request.Context(), "封禁玩家失败", "error", err, "player_id", playerID)
		if errors.Is(err, service.ErrGMPermissionDenied) {
			fail(c, http.StatusForbidden, "权限不足")
			return
		}
		fail(c, http.StatusInternalServerError, "封禁失败")
		return
	}

	success(c, nil)
}

// UnbanPlayer 解封玩家。
func (h *GMHandler) UnbanPlayer(c *gin.Context) {
	playerID, err := parseID(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的玩家 ID")
		return
	}

	adminID := c.GetUint64("admin_id")
	if err := h.gmSvc.UnbanPlayer(c.Request.Context(), playerID, adminID); err != nil {
		h.log.ErrorContext(c.Request.Context(), "解封玩家失败", "error", err, "player_id", playerID)
		if errors.Is(err, service.ErrGMPermissionDenied) {
			fail(c, http.StatusForbidden, "权限不足")
			return
		}
		fail(c, http.StatusInternalServerError, "解封失败")
		return
	}

	success(c, nil)
}

// ---- 公告 ----

// SendAnnouncement 发送公告。
func (h *GMHandler) SendAnnouncement(c *gin.Context) {
	var req model.GMAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if req.Title == "" || req.Content == "" {
		fail(c, http.StatusBadRequest, "标题和内容不能为空")
		return
	}

	if req.Type < 1 || req.Type > 3 {
		req.Type = 1
	}

	adminID := c.GetUint64("admin_id")
	if err := h.gmSvc.SendAnnouncement(c.Request.Context(), adminID, req.Title, req.Content, req.Type, req.TargetPlayerID); err != nil {
		h.log.ErrorContext(c.Request.Context(), "发送公告失败", "error", err)
		if errors.Is(err, service.ErrGMPermissionDenied) {
			fail(c, http.StatusForbidden, "权限不足")
			return
		}
		fail(c, http.StatusInternalServerError, "发送公告失败")
		return
	}

	success(c, nil)
}

// ---- 物品 ----

// SendItem 发送物品。
func (h *GMHandler) SendItem(c *gin.Context) {
	playerID, err := parseID(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的玩家 ID")
		return
	}

	var req model.GMSendItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if req.ItemID == "" || req.Quantity <= 0 {
		fail(c, http.StatusBadRequest, "物品 ID 和数量不能为空")
		return
	}

	adminID := c.GetUint64("admin_id")
	if err := h.gmSvc.SendItem(c.Request.Context(), playerID, adminID, req.ItemID, req.Quantity); err != nil {
		h.log.ErrorContext(c.Request.Context(), "发送物品失败", "error", err, "player_id", playerID, "item_id", req.ItemID)
		if errors.Is(err, service.ErrGMPermissionDenied) {
			fail(c, http.StatusForbidden, "权限不足")
			return
		}
		fail(c, http.StatusInternalServerError, "发送物品失败")
		return
	}

	success(c, nil)
}

// ---- 统计 ----

// GetServerStats 获取服务器统计。
func (h *GMHandler) GetServerStats(c *gin.Context) {
	stats, err := h.gmSvc.GetServerStats(c.Request.Context())
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "获取服务器统计失败", "error", err)
		fail(c, http.StatusInternalServerError, "获取服务器统计失败")
		return
	}

	success(c, stats)
}

// ---- 操作日志 ----

// GetOperationLogs 获取操作日志。
func (h *GMHandler) GetOperationLogs(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.gmSvc.GetOperationLogs(c.Request.Context(), page, limit)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "获取操作日志失败", "error", err)
		fail(c, http.StatusInternalServerError, "获取操作日志失败")
		return
	}

	success(c, logs)
}
