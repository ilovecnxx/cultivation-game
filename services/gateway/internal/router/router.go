// Package router 消息路由器，通过 NATS 将客户端消息分发到后端服务。
//
// 路由策略：
//   - 客户端消息 -> 根据 MsgID 映射到 NATS Subject: game.svc.<service>.<msgID>
//   - 后端响应 -> 发布到 NATS Subject: gateway.player.<playerID>
//   - 网关订阅 gateway.player.<playerID> 将响应转发回对应 WebSocket 连接。
package router

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cultivation-game/services/gateway/pkg/protocol"

	"github.com/nats-io/nats.go"
)

// MsgID 到服务的映射表（可根据实际业务扩展）。
// 约定：1-999 为系统消息，1000+ 为业务消息。
var msgIDToService = map[uint32]string{
	1:   "auth",    // 认证相关
	2:   "heartbeat", // 心跳
	100: "player",  // 玩家信息
	200: "scene",   // 场景/地图
	300: "combat",  // 战斗
	400: "chat",    // 聊天
	500: "mail",    // 邮件
	600: "shop",    // 商店
	700: "inventory", // 背包
	800: "quest",   // 任务
	900: "guild",   // 宗门
}

// Router 消息路由器。
type Router struct {
	natsConn *nats.Conn
	subs     map[uint64]*nats.Subscription // playerID -> NATS subscription
	subsMu   sync.RWMutex
	logger   *slog.Logger
}

// NewRouter 创建路由器并连接 NATS。
func NewRouter(natsURL string, connectTimeout time.Duration) (*Router, error) {
	nc, err := nats.Connect(natsURL,
		nats.Timeout(connectTimeout),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),   // 无限重连
		nats.ReconnectWait(2*time.Second),
		nats.Name("gateway-router"),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}

	slog.Info("connected to NATS", "url", natsURL)

	return &Router{
		natsConn: nc,
		subs:     make(map[uint64]*nats.Subscription),
		logger:   slog.Default(),
	}, nil
}

// Route 将客户端消息路由到后端服务。
func (r *Router) Route(packet *protocol.Packet) error {
	service, ok := msgIDToService[packet.MsgID]
	if !ok {
		service = "default"
	}

	subject := fmt.Sprintf("game.svc.%s.%d", service, packet.MsgID)
	data, err := protocol.Encode(packet)
	if err != nil {
		return fmt.Errorf("encode packet: %w", err)
	}

	if err := r.natsConn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish to nats: %w", err)
	}

	slog.Debug("route message",
		"subject", subject,
		"msg_id", packet.MsgID,
		"player_id", packet.PlayerID,
	)
	return nil
}

// SubscribePlayer 订阅指定玩家的后端响应通道。
// handler 收到消息后应将其发送到对应 WebSocket 连接。
func (r *Router) SubscribePlayer(playerID uint64, handler func([]byte)) error {
	r.subsMu.Lock()
	defer r.subsMu.Unlock()

	// 如果已订阅，先取消旧的
	if sub, ok := r.subs[playerID]; ok {
		if err := sub.Unsubscribe(); err != nil {
			slog.Warn("unsubscribe old player subject", "error", err, "player_id", playerID)
		}
		delete(r.subs, playerID)
	}

	subject := fmt.Sprintf("gateway.player.%d", playerID)
	sub, err := r.natsConn.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	if err != nil {
		return fmt.Errorf("subscribe player subject %s: %w", subject, err)
	}

	// 设置自动取消订阅（连接断开时自动清理）
	if err := sub.AutoUnsubscribe(1); err != nil {
		// 不阻塞，只是 log
		slog.Warn("set auto unsubscribe", "error", err)
	}

	r.subs[playerID] = sub
	slog.Debug("subscribed player subject", "subject", subject, "player_id", playerID)
	return nil
}

// UnsubscribePlayer 取消订阅玩家的后端响应通道。
func (r *Router) UnsubscribePlayer(playerID uint64) {
	r.subsMu.Lock()
	defer r.subsMu.Unlock()

	if sub, ok := r.subs[playerID]; ok {
		if err := sub.Unsubscribe(); err != nil {
			slog.Warn("unsubscribe player subject", "error", err, "player_id", playerID)
		}
		delete(r.subs, playerID)
		slog.Debug("unsubscribed player subject", "player_id", playerID)
	}
}

// PublishToPlayer 直接向玩家发消息（后端服务通过此通道响应）。
func (r *Router) PublishToPlayer(playerID uint64, data []byte) error {
	subject := fmt.Sprintf("gateway.player.%d", playerID)
	return r.natsConn.Publish(subject, data)
}

// IsConnected 返回 NATS 连接是否正常。
func (r *Router) IsConnected() bool {
	return r.natsConn != nil && r.natsConn.IsConnected()
}

// Close 关闭路由器，断开 NATS 连接。
func (r *Router) Close() {
	r.subsMu.Lock()
	defer r.subsMu.Unlock()

	for playerID, sub := range r.subs {
		if err := sub.Unsubscribe(); err != nil {
			slog.Warn("unsubscribe error on close", "error", err, "player_id", playerID)
		}
		delete(r.subs, playerID)
	}
	r.natsConn.Close()
	slog.Info("router closed")
}
