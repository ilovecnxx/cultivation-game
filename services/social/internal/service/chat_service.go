// Package service 提供社交服务的业务逻辑
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"cultivation-game/services/social/internal/model"
	"cultivation-game/services/social/internal/repository"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// ============================================================
// WebSocket 客户端连接管理
// ============================================================

// WSClient 表示一个 WebSocket 客户端连接
type WSClient struct {
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
	SectID   string // 所属宗门ID(用于宗门频道)
}

// ChatService 聊天业务逻辑
type ChatService struct {
	repo        *repository.ChatRepo
	redis       *redis.Client
	clients     map[string]*WSClient // key: userID
	clientsLock sync.RWMutex
	// 敏感词列表(实际应从配置/管理后台加载)
	sensitiveWords []string
}

// NewChatService 创建聊天服务
func NewChatService(repo *repository.ChatRepo, rdb *redis.Client) *ChatService {
	return &ChatService{
		repo:           repo,
		redis:          rdb,
		clients:        make(map[string]*WSClient),
		sensitiveWords: loadSensitiveWords(),
	}
}

// loadSensitiveWords 加载敏感词列表(示例)
func loadSensitiveWords() []string {
	return []string{
		"敏感词1", "敏感词2", "违禁词",
		// 实际项目中应从配置文件或管理后台动态加载
	}
}

// RegisterClient 注册 WebSocket 客户端
func (s *ChatService) RegisterClient(client *WSClient) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	s.clients[client.UserID] = client
	_ = s.redis.Set(context.Background(), "online:"+client.UserID, "1", 30*time.Minute).Err()
}

// UnregisterClient 注销 WebSocket 客户端
func (s *ChatService) UnregisterClient(userID string) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	if _, ok := s.clients[userID]; ok {
		delete(s.clients, userID)
		_ = s.redis.Del(context.Background(), "online:"+userID).Err()
	}
}

// GetOnlineUsers 批量查询用户在线状态
func (s *ChatService) GetOnlineUsers(ctx context.Context, userIDs []string) map[string]bool {
	result := make(map[string]bool)
	for _, uid := range userIDs {
		val, err := s.redis.Exists(ctx, "online:"+uid).Result()
		result[uid] = err == nil && val > 0
	}
	return result
}

// IsOnline 查询单个用户是否在线
func (s *ChatService) IsOnline(ctx context.Context, userID string) bool {
	val, err := s.redis.Exists(ctx, "online:"+userID).Result()
	return err == nil && val > 0
}

// ============================================================
// 敏感词过滤
// ============================================================

// FilterSensitive 过滤敏感词，将敏感词替换为 *
func (s *ChatService) FilterSensitive(content string) string {
	result := content
	for _, word := range s.sensitiveWords {
		if strings.Contains(result, word) {
			result = strings.ReplaceAll(result, word, strings.Repeat("*", len([]rune(word))))
		}
	}
	return result
}

// ============================================================
// 消息处理
// ============================================================

// HandleMessage 处理收到的聊天消息
func (s *ChatService) HandleMessage(senderID, senderName string, channel model.ChatChannel, targetID, content string) (*model.ChatMessage, error) {
	// 1. 敏感词过滤
	filtered := s.FilterSensitive(content)

	// 2. 构建消息
	msg := &model.ChatMessage{
		ID:         uuid.New().String(),
		Channel:    channel,
		SenderID:   senderID,
		SenderName: senderName,
		TargetID:   targetID,
		Content:    filtered,
		CreatedAt:  time.Now(),
	}

	// 3. 持久化
	ctx := context.Background()
	if err := s.repo.Insert(ctx, msg); err != nil {
		return nil, fmt.Errorf("保存聊天消息失败: %w", err)
	}

	// 4. 广播
	s.broadcastMessage(msg)

	return msg, nil
}

// HandleSystemMessage 发送系统消息(如世界事件公告)
func (s *ChatService) HandleSystemMessage(channel model.ChatChannel, content string, targetID string) (*model.ChatMessage, error) {
	msg := &model.ChatMessage{
		ID:        uuid.New().String(),
		Channel:   channel,
		SenderID:  "system",
		SenderName: "系统",
		TargetID:  targetID,
		Content:   content,
		IsSystem:  true,
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	if err := s.repo.Insert(ctx, msg); err != nil {
		return nil, fmt.Errorf("保存系统消息失败: %w", err)
	}

	s.broadcastMessage(msg)
	return msg, nil
}

// broadcastMessage 向频道内所有在线客户端广播消息
func (s *ChatService) broadcastMessage(msg *model.ChatMessage) {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()

	data := []byte(fmt.Sprintf(`{"type":"chat","data":%s}`, toJSON(msg)))

	for _, client := range s.clients {
		switch msg.Channel {
		case model.ChannelWorld:
			// 世界频道发给所有人
			client.Send <- data

		case model.ChannelSect:
			// 宗门频道只发给同宗门成员
			if client.SectID == msg.TargetID {
				client.Send <- data
			}

		case model.ChannelPrivate:
			// 私聊只发给收发双方
			if client.UserID == msg.SenderID || client.UserID == msg.TargetID {
				client.Send <- data
			}

		case model.ChannelSystem:
			// 系统频道发给所有人
			client.Send <- data
		}
	}
}

// GetHistory 获取聊天历史
func (s *ChatService) GetHistory(ctx context.Context, channel model.ChatChannel, targetID string, limit int64, before time.Time) ([]*model.ChatMessage, error) {
	if channel == model.ChannelPrivate {
		return nil, fmt.Errorf("私聊历史请使用 GetPrivateHistory")
	}
	return s.repo.FindByChannel(ctx, channel, targetID, limit, before)
}

// GetPrivateHistory 获取私聊历史
func (s *ChatService) GetPrivateHistory(ctx context.Context, userA, userB string, limit int64, before time.Time) ([]*model.ChatMessage, error) {
	return s.repo.FindPrivateMessages(ctx, userA, userB, limit, before)
}

// toJSON 简单序列化为 JSON 字符串(避免引入 json.Marshal 错误处理)
func toJSON(v interface{}) string {
	// 实际项目中使用 json.Marshal
	b, _ := json.Marshal(v)
	return string(b)
}
