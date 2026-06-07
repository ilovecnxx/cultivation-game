// Package handler 提供 HTTP/WebSocket 接口处理
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"cultivation-game/services/social/internal/model"
	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ChatHandler 聊天 HTTP/WebSocket 处理器
type ChatHandler struct {
	logger *slog.Logger
	svc *service.ChatService
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(logger *slog.Logger, svc *service.ChatService) *ChatHandler {
	return &ChatHandler{logger: logger, svc: svc}
}

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 实际项目应校验来源
	},
}

// HandleWebSocket 处理 WebSocket 连接
// @Description 聊天 WebSocket 连接, 通过 query 参数传递 user_id 和 sect_id
func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	userID := c.Query("user_id")
	sectID := c.Query("sect_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket 升级失败", "error", err)
		return
	}

	client := &service.WSClient{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		SectID: sectID,
	}

	h.svc.RegisterClient(client)

	// 启动写 goroutine
	go h.writePump(client)
	// 读循环 (阻塞)
	h.readPump(client)
}

// writePump 向客户端写入消息
func (h *ChatHandler) writePump(client *service.WSClient) {
	defer func() {
		client.Conn.Close()
		h.svc.UnregisterClient(client.UserID)
	}()

	for message := range client.Send {
		if err := client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return
		}
		if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

// readPump 读取客户端消息
func (h *ChatHandler) readPump(client *service.WSClient) {
	defer func() {
		client.Conn.Close()
		h.svc.UnregisterClient(client.UserID)
	}()

	client.Conn.SetReadLimit(512) // 单条消息最大 512 字节
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				h.logger.Error("WebSocket 读错误", "error", err)
			}
			break
		}

		// 解析消息格式: {"channel":"world","target_id":"","content":"hello"}
		var req struct {
			Channel  string `json:"channel"`
			TargetID string `json:"target_id"`
			Content  string `json:"content"`
		}
		// 简化解析
		if err := json.Unmarshal(message, &req); err != nil {
			h.logger.Warn("消息格式错误", "error", err)
			continue
		}

		channel := model.ChatChannel(req.Channel)
		if channel == "" {
			channel = model.ChannelWorld
		}

		_, err = h.svc.HandleMessage(client.UserID, "", channel, req.TargetID, req.Content)
		if err != nil {
			h.logger.Error("处理消息失败", "error", err)
		}
	}
}

// GetHistory 获取聊天历史 HTTP 接口
// @Router GET /api/v1/chat/history
func (h *ChatHandler) GetHistory(c *gin.Context) {
	channel := model.ChatChannel(c.Query("channel"))
	targetID := c.Query("target_id")
	limitStr := c.DefaultQuery("limit", "50")
	beforeStr := c.Query("before")

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50
	}

	var before time.Time
	if beforeStr != "" {
		if t, err := time.Parse(time.RFC3339, beforeStr); err == nil {
			before = t
		}
	}

	ctx := c.Request.Context()
	var messages []*model.ChatMessage

	if channel == model.ChannelPrivate {
		userA := c.Query("user_a")
		userB := c.Query("user_b")
		if userA == "" || userB == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "私聊需要 user_a 和 user_b"})
			return
		}
		messages, err = h.svc.GetPrivateHistory(ctx, userA, userB, limit, before)
	} else {
		messages, err = h.svc.GetHistory(ctx, channel, targetID, limit, before)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": messages})
}

// SendSystemNotification 发送系统通知(供世界事件等服务调用)
// @Router POST /api/v1/chat/system-notify [post]
func (h *ChatHandler) SendSystemNotification(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	_, err := h.svc.HandleSystemMessage(model.ChannelSystem, req.Content, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "系统消息已发送"})
}

// GetOnlineStatus 查询在线状态 HTTP 接口
// @Router POST /api/v1/chat/online
func (h *ChatHandler) GetOnlineStatus(c *gin.Context) {
	var req struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	status := h.svc.GetOnlineUsers(c.Request.Context(), req.UserIDs)
	c.JSON(http.StatusOK, gin.H{"data": status})
}
