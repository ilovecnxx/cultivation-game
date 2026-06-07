package handler

import (
	"net/http"

	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// FriendHandler 好友 HTTP 处理器
type FriendHandler struct {
	svc *service.FriendService
}

// NewFriendHandler 创建好友处理器
func NewFriendHandler(svc *service.FriendService) *FriendHandler {
	return &FriendHandler{svc: svc}
}

// GetFriendList 获取好友列表
// @Router GET /api/v1/friends [get]
func (h *FriendHandler) GetFriendList(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	friends, err := h.svc.GetFriendList(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": friends})
}

// GetBlacklist 获取黑名单
// @Router GET /api/v1/friends/blacklist [get]
func (h *FriendHandler) GetBlacklist(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	list, err := h.svc.GetBlacklist(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ApplyFriend 发送好友申请
// @Router POST /api/v1/friends/apply [post]
func (h *FriendHandler) ApplyFriend(c *gin.Context) {
	var req struct {
		FromID  string `json:"from_id"`
		FromName string `json:"from_name"`
		ToID    string `json:"to_id"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.ApplyFriend(c.Request.Context(), req.FromID, req.FromName, req.ToID, req.Message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "申请已发送"})
}

// HandleApply 处理好友申请
// @Router POST /api/v1/friends/handle-apply [post]
func (h *FriendHandler) HandleApply(c *gin.Context) {
	var req struct {
		ApplyID string `json:"apply_id"`
		Accept  bool   `json:"accept"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.HandleApply(c.Request.Context(), req.ApplyID, req.Accept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "操作成功"})
}

// GetPendingApplies 获取待处理的申请
// @Router GET /api/v1/friends/pending-applies [get]
func (h *FriendHandler) GetPendingApplies(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	applies, err := h.svc.GetPendingApplies(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": applies})
}

// RemoveFriend 删除好友
// @Router DELETE /api/v1/friends [delete]
func (h *FriendHandler) RemoveFriend(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id"`
		FriendID string `json:"friend_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.RemoveFriend(c.Request.Context(), req.UserID, req.FriendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// BlockUser 拉黑用户
// @Router POST /api/v1/friends/block [post]
func (h *FriendHandler) BlockUser(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id"`
		BlockID string `json:"block_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.BlockUser(c.Request.Context(), req.UserID, req.BlockID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "拉黑成功"})
}

// UnblockUser 取消拉黑
// @Router POST /api/v1/friends/unblock [post]
func (h *FriendHandler) UnblockUser(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id"`
		BlockID string `json:"block_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.UnblockUser(c.Request.Context(), req.UserID, req.BlockID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已取消拉黑"})
}
