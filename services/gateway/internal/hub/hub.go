package hub

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"cultivation-game/services/gateway/internal/anticheat"
	"cultivation-game/services/gateway/internal/config"
	"cultivation-game/services/gateway/internal/ratelimit"
	"cultivation-game/services/gateway/internal/router"
	"cultivation-game/services/gateway/internal/session"
	"cultivation-game/services/gateway/pkg/protocol"
)

// Hub 连接池管理器。
// 职责：
//   1. 连接注册/注销/查找
//   2. 按 PlayerID 索引，支持快速查找
//   3. 断线重连窗口管理
//   4. 广播消息
//   5. 优雅关闭
//   6. 管理在线玩家 Redis 会话
//   7. 反作弊检查
type Hub struct {
	config      *config.Config
	router      *router.Router
	rateLimiter *ratelimit.RateLimiter
	sessionMgr  *session.Manager    // Redis 会话管理器（可能为 nil）
	anticheat   *anticheat.Manager  // 反作弊管理器
	ctx         context.Context     // 全局上下文

	// 连接存储
	connections map[string]*Connection   // connID -> Connection（所有连接）
	players     map[uint64]*Connection   // playerID -> Connection（已认证连接）
	mu          sync.RWMutex
	closed      bool

	// 重连窗口管理
	reconnectTimers map[uint64]*time.Timer // playerID -> 清理定时器
	reconnectMu     sync.Mutex

	logger *slog.Logger
}

// NewHub 创建连接池。
func NewHub(cfg *config.Config, r *router.Router, rl *ratelimit.RateLimiter, sessMgr *session.Manager, ac *anticheat.Manager) *Hub {
	h := &Hub{
		config:          cfg,
		router:          r,
		rateLimiter:     rl,
		sessionMgr:      sessMgr,
		anticheat:       ac,
		ctx:			context.Background(),
		connections:     make(map[string]*Connection),
		players:         make(map[uint64]*Connection),
		reconnectTimers: make(map[uint64]*time.Timer),
		logger:          slog.Default(),
	}

	// 如果 Redis 可用，启动时清理上次遗留的在线玩家标记（节点重启后）
	if sessMgr != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if count, err := sessMgr.Client().Del(ctx, session.OnlinePlayersKey).Result(); err == nil && count > 0 {
			slog.Info("启动时清理了 Redis 在线玩家标记", "count", count)
		}
		cancel()
	}

	return h
}

// Register 注册连接到连接池。
// 如果玩家已有旧连接，会关闭旧连接并用新连接替换。
func (h *Hub) Register(c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		slog.Warn("hub is closed, rejecting new connection", "conn_id", c.ID)
		c.Close()
		return
	}

	h.connections[c.ID] = c
	slog.Info("connection registered", "conn_id", c.ID)

	// 如果已认证，建立玩家索引并同步到 Redis
	if c.PlayerID > 0 {
		h.bindPlayerLocked(c)
		h.syncPlayerOnline(c)
	}
}

// syncPlayerOnline 将玩家在线状态同步到 Redis。
func (h *Hub) syncPlayerOnline(c *Connection) {
	if h.sessionMgr == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := h.sessionMgr.PlayerOnline(ctx, c.PlayerID, c.ID, c.Account); err != nil {
		slog.Warn("同步玩家在线状态到 Redis 失败",
			"error", err, "player_id", c.PlayerID, "conn_id", c.ID,
		)
	}
}

// syncPlayerOffline 从 Redis 中清除玩家在线状态。
func (h *Hub) syncPlayerOffline(playerID uint64) {
	if h.sessionMgr == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := h.sessionMgr.PlayerOffline(ctx, playerID); err != nil {
		slog.Warn("清除 Redis 在线状态失败",
			"error", err, "player_id", playerID,
		)
	}
}

// bindPlayerLocked 将连接绑定到玩家索引（调用者需持有 mu 写锁）。
func (h *Hub) bindPlayerLocked(c *Connection) {
	playerID := c.PlayerID

	// 关闭该玩家的旧连接
	if old, ok := h.players[playerID]; ok && old != c {
		slog.Info("player reconnecting, close old connection",
			"player_id", playerID,
			"old_conn", old.ID,
			"new_conn", c.ID,
		)
		// 取消重连定时器（如果有）
		h.cancelReconnectTimer(playerID)
		// 异步关闭旧连接（可能被 WritePump 阻塞）
		go old.Close()
	}

	h.players[playerID] = c

	// 订阅该玩家的后端响应通道
	if err := h.router.SubscribePlayer(playerID, func(data []byte) {
		c.Send(data)
	}); err != nil {
		slog.Error("subscribe player error",
			"error", err,
			"player_id", playerID,
		)
	}

	slog.Info("player bound to connection",
		"player_id", playerID,
		"conn_id", c.ID,
		"account", c.Account,
	)
}

// unregister 从连接池注销连接（由 Connection.Close 调用）。
func (h *Hub) unregister(c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}

	// 从连接索引移除
	delete(h.connections, c.ID)

	// 从玩家索引移除（仅当是该玩家的当前连接时）
	if c.PlayerID > 0 {
		if current, ok := h.players[c.PlayerID]; ok && current == c {
			delete(h.players, c.PlayerID)

			// 取消订阅玩家消息通道
			h.router.UnsubscribePlayer(c.PlayerID)

			// 从 Redis 清除在线状态
			h.syncPlayerOffline(c.PlayerID)

			// 启动断线重连定时器
			h.startReconnectTimer(c.PlayerID)

			slog.Info("player disconnected, reconnect timer started",
				"player_id", c.PlayerID,
				"window", h.config.ReconnectWindow,
			)
		}
	}
}

// startReconnectTimer 启动断线重连定时器。
// 在重连窗口内如果玩家未重连，清理玩家状态。
func (h *Hub) startReconnectTimer(playerID uint64) {
	h.reconnectMu.Lock()
	defer h.reconnectMu.Unlock()

	// 取消旧的定时器
	if timer, ok := h.reconnectTimers[playerID]; ok {
		timer.Stop()
	}

	h.reconnectTimers[playerID] = time.AfterFunc(h.config.ReconnectWindow, func() {
		h.onReconnectTimeout(playerID)
	})
}

// cancelReconnectTimer 取消断线重连定时器（玩家重连成功时调用）。
func (h *Hub) cancelReconnectTimer(playerID uint64) {
	h.reconnectMu.Lock()
	defer h.reconnectMu.Unlock()

	if timer, ok := h.reconnectTimers[playerID]; ok {
		timer.Stop()
		delete(h.reconnectTimers, playerID)
	}
}

// onReconnectTimeout 重连超时回调，清理玩家状态。
func (h *Hub) onReconnectTimeout(playerID uint64) {
	h.reconnectMu.Lock()
	delete(h.reconnectTimers, playerID)
	h.reconnectMu.Unlock()

	slog.Info("reconnect window expired, cleanup player",
		"player_id", playerID,
	)

	// 通过 Redis Pub/Sub 通知后端服务清理玩家游戏状态
	if h.sessionMgr != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		msg := map[string]interface{}{
			"type":      "player_disconnect_timeout",
			"player_id": playerID,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(msg)
		if err := h.sessionMgr.Publish(ctx, "gateway:events", data); err != nil {
			slog.Warn("发布玩家超时事件失败", "error", err, "player_id", playerID)
		}

		// 确保 Redis 中该玩家已被标记离线
		if err := h.sessionMgr.PlayerOffline(ctx, playerID); err != nil {
			slog.Warn("清理 Redis 中玩家在线状态失败", "error", err, "player_id", playerID)
		}
	}
}

// GetConnection 根据连接 ID 查找连接。
func (h *Hub) GetConnection(connID string) *Connection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connections[connID]
}

// GetPlayerConnection 根据玩家 ID 查找连接。
func (h *Hub) GetPlayerConnection(playerID uint64) *Connection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.players[playerID]
}

// OnlineCount 返回当前在线玩家数。
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.players)
}

// TotalConnections 返回总连接数（含未认证）。
func (h *Hub) TotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

// Broadcast 向所有已认证玩家广播消息。
func (h *Hub) Broadcast(packet *protocol.Packet) {
	data, err := protocol.Encode(packet)
	if err != nil {
		slog.Error("broadcast encode error", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, c := range h.players {
		c.Send(data)
	}
}

// BroadcastByPlayerIDs 向指定玩家列表广播消息。
func (h *Hub) BroadcastByPlayerIDs(playerIDs []uint64, packet *protocol.Packet) {
	data, err := protocol.Encode(packet)
	if err != nil {
		slog.Error("broadcast encode error", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, pid := range playerIDs {
		if c, ok := h.players[pid]; ok {
			c.Send(data)
		}
	}
}

// PlayerIDs 返回所有在线玩家 ID 列表。
func (h *Hub) PlayerIDs() []uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]uint64, 0, len(h.players))
	for pid := range h.players {
		ids = append(ids, pid)
	}
	return ids
}

// RateLimiter 返回限流管理器。
func (h *Hub) RateLimiter() *ratelimit.RateLimiter {
	return h.rateLimiter
}

// AntiCheat 返回反作弊管理器。
func (h *Hub) AntiCheat() *anticheat.Manager {
	return h.anticheat
}

// SessionMgr 返回 Redis 会话管理器，可能为 nil。
func (h *Hub) SessionMgr() *session.Manager {
	return h.sessionMgr
}

// Close 关闭连接池，断开所有连接。
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	h.mu.Unlock()

	slog.Info("hub closing, disconnecting all connections",
		"total", len(h.connections),
		"players", len(h.players),
	)

	// 关闭所有连接
	h.mu.RLock()
	for _, c := range h.connections {
		c.Close()
	}
	h.mu.RUnlock()

	// 清理所有重连定时器
	h.reconnectMu.Lock()
	for _, timer := range h.reconnectTimers {
		timer.Stop()
	}
	h.reconnectTimers = make(map[uint64]*time.Timer)
	h.reconnectMu.Unlock()

	slog.Info("hub closed")
}
