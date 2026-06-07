package handler

import (
	"net/http"
	"strconv"
	"time"

	"cultivation-game/services/social/internal/model"
	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// MailHandler 邮件 HTTP 处理器
type MailHandler struct {
	svc *service.MailService
}

// NewMailHandler 创建邮件处理器
func NewMailHandler(svc *service.MailService) *MailHandler {
	return &MailHandler{svc: svc}
}

// SendSystemMail 发送系统邮件(管理员接口)
// @Router POST /api/v1/mail/system [post]
func (h *MailHandler) SendSystemMail(c *gin.Context) {
	var req struct {
		ReceiverID string                  `json:"receiver_id"`
		Title      string                  `json:"title"`
		Content    string                  `json:"content"`
		Attachments []model.MailAttachment `json:"attachments"`
		ExpireDays int                     `json:"expire_days"` // 过期天数, 0为不过期
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	expireDuration := time.Duration(req.ExpireDays) * 24 * time.Hour
	mail, err := h.svc.SendSystemMail(c.Request.Context(), req.ReceiverID, req.Title, req.Content, req.Attachments, expireDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mail})
}

// SendPlayerMail 发送玩家邮件
// @Router POST /api/v1/mail/player [post]
func (h *MailHandler) SendPlayerMail(c *gin.Context) {
	var req struct {
		SenderID   string                  `json:"sender_id"`
		SenderName string                  `json:"sender_name"`
		ReceiverID string                  `json:"receiver_id"`
		Title      string                  `json:"title"`
		Content    string                  `json:"content"`
		Attachments []model.MailAttachment `json:"attachments,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	mail, err := h.svc.SendPlayerMail(c.Request.Context(), req.SenderID, req.SenderName, req.ReceiverID, req.Title, req.Content, req.Attachments)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mail})
}

// GetInbox 获取收件箱
// @Router GET /api/v1/mail/inbox [get]
func (h *MailHandler) GetInbox(c *gin.Context) {
	userID := c.Query("user_id")
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

	mails, total, err := h.svc.GetInbox(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  mails,
		"total": total,
		"page":  page,
	})
}

// ReadMail 阅读邮件
// @Router GET /api/v1/mail/read [get]
func (h *MailHandler) ReadMail(c *gin.Context) {
	mailID := c.Query("mail_id")
	userID := c.Query("user_id")

	mail, err := h.svc.ReadMail(c.Request.Context(), mailID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mail})
}

// ClaimAttachment 领取附件
// @Router POST /api/v1/mail/claim [post]
func (h *MailHandler) ClaimAttachment(c *gin.Context) {
	var req struct {
		MailID string `json:"mail_id"`
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	mail, err := h.svc.ClaimAttachment(c.Request.Context(), req.MailID, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mail})
}

// DeleteMail 删除邮件
// @Router DELETE /api/v1/mail [delete]
func (h *MailHandler) DeleteMail(c *gin.Context) {
	var req struct {
		MailID string `json:"mail_id"`
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.DeleteMail(c.Request.Context(), req.MailID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// CountUnread 统计未读邮件数
// @Router GET /api/v1/mail/unread-count [get]
func (h *MailHandler) CountUnread(c *gin.Context) {
	userID := c.Query("user_id")
	count, err := h.svc.CountUnread(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}
