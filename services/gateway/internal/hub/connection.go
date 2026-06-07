// Package hub 连接管理器，管理所有 WebSocket 连接的生命周期。
package hub

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"cultivation-game/services/gateway/internal/ratelimit"
	"cultivation-game/services/gateway/pkg/protocol"

	"github.com/gorilla/websocket"
)

// 默认值
const (
	writeTimeout    = 10 * time.Second  // 单次写入超时
	sendBufferSize  = 256               // 发送缓冲通道大小
	readWaitTimeout = 120 * time.Second // 读等待超时
)

var (
	// connIDCounter 连接 ID 生成器
	connIDCounter atomic.Uint64
)

// ConnectionState 连接状态
type ConnectionState int32

const (
	StateNew          ConnectionState = iota // 新建
	StateAuthenticated                       // 已认证
	StateDisconnected                        // 已断线
	StateClosed                              // 已关闭
)

// Connection 代表一个 WebSocket 连接。
// 每个连接运行 ReadPump 和 WritePump 两个 goroutine。
type Connection struct {
	ID        string                     // 连接唯一 ID
	PlayerID  uint64                     // 玩家 ID（认证后设置）
	Account   string                     // 账号
	State     ConnectionState            // 连接状态
	Conn      *websocket.Conn            // WebSocket 连接
	hub       *Hub                       // 所属连接池
	SendCh    chan []byte                // 发送缓冲通道
	rateLimit *ratelimit.TokenBucket     // 令牌桶限流器
	lastPong  time.Time                  // 最近一次 pong 时间
	closeOnce sync.Once                  // 确保只关闭一次
	done      chan struct{}              // 关闭通知
	mu        sync.RWMutex               // 保护 PlayerID、Account、State 等字段
}

// generateConnID 生成全局唯一的连接 ID。
func generateConnID() string {
	id := connIDCounter.Add(1)
	return fmt.Sprintf("conn_%d_%d", time.Now().UnixMilli(), id)
}

// NewConnection 创建新的连接实例，由 Hub 调用。
func (h *Hub) NewConnection(conn *websocket.Conn, rateLimit *ratelimit.TokenBucket) *Connection {
	return &Connection{
		ID:        generateConnID(),
		State:     StateNew,
		Conn:      conn,
		hub:       h,
		SendCh:    make(chan []byte, sendBufferSize),
		rateLimit: rateLimit,
		lastPong:  time.Now(),
		done:      make(chan struct{}),
	}
}

// msgIDToAction 将消息 ID 映射为动作名称（用于反作弊校验）。
func msgIDToAction(msgID uint32) string {
	switch msgID {
	case 300:
		return "combat_attack"
	case 301:
		return "combat_skill"
	case 302:
		return "combat_defend"
	case 303:
		return "combat_flee"
	case 600:
		return "shop_buy"
	case 601:
		return "shop_sell"
	case 700:
		return "inventory_use"
	case 701:
		return "inventory_discard"
	case 400:
		return "chat_message"
	case 500:
		return "mail_send"
	case 800:
		return "quest_action"
	case 200:
		return "scene_move"
	case 100:
		return "player_info"
	case 1:
		return "auth_action"
	default:
		// 在 1-999 系统消息和 1000+ 业务消息间判断
		if msgID < 100 {
			return "system"
		}
		if msgID >= 1000 {
			return "breakthrough"
		}
		return "unknown"
	}
}

// ReadPump 从 WebSocket 读取消息的循环。
// goroutine 入口：负责读取消息、限流检查、反作弊校验、解码、路由。
func (c *Connection) ReadPump() {
	defer func() {
		// 连接关闭时清理反作弊中的战斗速度记录
		if c.hub.anticheat != nil {
			c.mu.RLock()
			pid := c.PlayerID
			c.mu.RUnlock()
			if pid > 0 {
				c.hub.anticheat.CombatSpeed.CleanupPlayer(pid)
			}
		}
		slog.Info("read pump exit", "player_id", c.PlayerID, "conn_id", c.ID)
		c.Close()
	}()

	// 设置读取限制和超时
	c.Conn.SetReadLimit(c.hub.config.WSMaxMessageSize)

	// 设置 Pong 处理：重置读取超时 + 记录 lastPong
	c.Conn.SetPongHandler(func(pongData string) error {
		c.mu.Lock()
		c.lastPong = time.Now()
		c.mu.Unlock()
		// 延长读取超时
		c.Conn.SetReadDeadline(time.Now().Add(readWaitTimeout))
		return nil
	})

	// 初始读取超时
	c.Conn.SetReadDeadline(time.Now().Add(readWaitTimeout))

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			// 正常关闭错误不视为异常
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				slog.Warn("unexpected websocket close",
					"error", err,
					"player_id", c.PlayerID,
					"conn_id", c.ID,
				)
			}
			break
		}

		// 限流检查（基于连接级别的令牌桶）
		if !c.rateLimit.Allow() {
			slog.Warn("rate limit exceeded",
				"player_id", c.PlayerID,
				"conn_id", c.ID,
			)
			c.sendErrorPacket(protocol.ErrRateLimited, "请求过于频繁，请稍后重试")
			continue
		}

		// 解码消息包
		packet, err := protocol.Decode(message)
		if err != nil {
			slog.Warn("decode packet error",
				"error", err,
				"player_id", c.PlayerID,
				"conn_id", c.ID,
			)
			c.sendErrorPacket(protocol.ErrInvalidPacket, "消息格式错误")
			continue
		}

		// 注入玩家 ID
		c.mu.RLock()
		packet.PlayerID = c.PlayerID
		pid := c.PlayerID
		c.mu.RUnlock()

		// 反作弊检查（仅已认证玩家）
		if pid > 0 && c.hub.anticheat != nil {
			action := msgIDToAction(packet.MsgID)
			result := c.hub.anticheat.ValidateAction(c.hub.ctx, pid, packet.MsgID, action, 0, "", "", c.ID)

			if !result.Allowed {
				// 构造详细的限流/封禁错误响应
				errBody := map[string]interface{}{
					"code":        429,
					"reason":      result.Reason,
					"retry_after": result.RetryAfterSec,
				}
				if result.BanInfo != nil {
					errBody["ban_expires_at"] = result.BanInfo.ExpiresAt.Unix()
					errBody["ban_duration"] = result.BanInfo.Duration
				}
				bodyBytes, _ := json.Marshal(errBody)
				c.SendPacket(&protocol.Packet{
					MsgID:     uint32(protocol.ErrRateLimited),
					PlayerID:  pid,
					Body:      bodyBytes,
					Timestamp: time.Now().UnixMilli(),
				})

				slog.Warn("anticheat blocked",
					"player_id", pid,
					"msg_id", packet.MsgID,
					"action", action,
					"reason", result.Reason,
					"conn_id", c.ID,
				)
				continue
			}
		}

		// 路由到后端服务
		if err := c.hub.router.Route(packet); err != nil {
			slog.Error("route error",
				"error", err,
				"msg_id", packet.MsgID,
				"player_id", c.PlayerID,
				"conn_id", c.ID,
			)
			c.sendErrorPacket(protocol.ErrInternal, "消息处理失败")
		}
	}
}

// WritePump 向 WebSocket 写入消息的循环。
// goroutine 入口：负责从 SendCh 读取数据并写入 WebSocket，同时负责心跳 Ping。
func (c *Connection) WritePump() {
	// 心跳定时器
	pingTicker := time.NewTicker(c.hub.config.PingInterval)
	defer func() {
		pingTicker.Stop()
		slog.Info("write pump exit", "player_id", c.PlayerID, "conn_id", c.ID)
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.SendCh:
			if !ok {
				// 发送通道已关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.writeMessage(websocket.TextMessage, message); err != nil {
				slog.Error("write message error",
					"error", err,
					"player_id", c.PlayerID,
					"conn_id", c.ID,
				)
				return
			}

		case <-pingTicker.C:
			// 发送 Ping
			if err := c.writeMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("ping error",
					"error", err,
					"player_id", c.PlayerID,
					"conn_id", c.ID,
				)
				return
			}

			// 检查 Pong 超时
			c.mu.RLock()
			lastPong := c.lastPong
			c.mu.RUnlock()
			if time.Since(lastPong) > c.hub.config.PongTimeout {
				slog.Warn("pong timeout, closing",
					"player_id", c.PlayerID,
					"conn_id", c.ID,
					"last_pong_ago", time.Since(lastPong),
				)
				return
			}

		case <-c.done:
			return
		}
	}
}

// writeMessage 带超时写入消息到 WebSocket。
func (c *Connection) writeMessage(messageType int, data []byte) error {
	c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return c.Conn.WriteMessage(messageType, data)
}

// Send 发送数据到发送缓冲通道（异步）。
// 返回 true 表示发送成功，false 表示缓冲区满。
func (c *Connection) Send(data []byte) bool {
	select {
	case c.SendCh <- data:
		return true
	default:
		slog.Warn("send buffer full, dropping message",
			"player_id", c.PlayerID,
			"conn_id", c.ID,
		)
		return false
	}
}

// SendPacket 编码并发送消息包。
func (c *Connection) SendPacket(packet *protocol.Packet) bool {
	data, err := protocol.Encode(packet)
	if err != nil {
		slog.Error("encode packet error",
			"error", err,
			"msg_id", packet.MsgID,
			"player_id", c.PlayerID,
		)
		return false
	}
	return c.Send(data)
}

// sendErrorPacket 发送错误包（内部辅助方法）。
func (c *Connection) sendErrorPacket(code int, msg string) {
	c.SendPacket(protocol.ErrorPacket(code, msg))
}

// SetPlayer 设置玩家信息，标记为已认证。
func (c *Connection) SetPlayer(playerID uint64, account string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.PlayerID = playerID
	c.Account = account
	c.State = StateAuthenticated
}

// IsAuthenticated 是否已认证。
func (c *Connection) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State >= StateAuthenticated
}

// IsClosed 连接是否已关闭。
func (c *Connection) IsClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

// Close 关闭连接。
// 安全：使用 sync.Once 确保只执行一次。
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		// 标记关闭
		c.mu.Lock()
		c.State = StateClosed
		c.mu.Unlock()

		// 通知所有 goroutine 退出
		close(c.done)

		// 关闭 WebSocket
		c.Conn.Close()

		// 从连接池注销
		c.hub.unregister(c)

		// 清理限流器
		c.hub.rateLimiter.RemoveBucket(c.ID)

		slog.Info("connection closed",
			"player_id", c.PlayerID,
			"conn_id", c.ID,
			"account", c.Account,
		)
	})
}
