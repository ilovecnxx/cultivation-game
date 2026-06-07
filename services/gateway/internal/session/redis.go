// Package session 提供基于 Redis 的在线玩家会话管理。
//
// 功能：
//   - 连接 Redis
//   - 在线玩家列表（Redis Set）
//   - 跨服务消息发布/订阅（Redis Pub/Sub）
package session

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// OnlinePlayersKey Redis 中在线玩家集合的 key
	OnlinePlayersKey = "gateway:online_players"
	// PlayerSessionKeyPrefix 玩家会话 Hash 的前缀
	PlayerSessionKeyPrefix = "gateway:session:"
)

// Manager Redis 会话管理器。
type Manager struct {
	rdb *redis.Client
}

// NewManager 创建会话管理器并连接 Redis。
func NewManager(addr, password string, db int) (*Manager, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}

	slog.Info("Redis 会话管理器初始化成功", "addr", addr)
	return &Manager{rdb: rdb}, nil
}

// Client 返回底层 Redis 客户端。
func (m *Manager) Client() *redis.Client {
	return m.rdb
}

// Ping 检查 Redis 连接是否正常。
func (m *Manager) Ping(ctx context.Context) error {
	return m.rdb.Ping(ctx).Err()
}

// PlayerOnline 标记玩家在线，并记录会话信息。
func (m *Manager) PlayerOnline(ctx context.Context, playerID uint64, connID, account string) error {
	pipe := m.rdb.Pipeline()
	pipe.SAdd(ctx, OnlinePlayersKey, playerID)
	sessionKey := fmt.Sprintf("%s%d", PlayerSessionKeyPrefix, playerID)
	pipe.HSet(ctx, sessionKey, map[string]interface{}{
		"conn_id":  connID,
		"account":  account,
		"online_since": time.Now().Unix(),
	})
	pipe.Expire(ctx, sessionKey, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// PlayerOffline 标记玩家离线。
func (m *Manager) PlayerOffline(ctx context.Context, playerID uint64) error {
	pipe := m.rdb.Pipeline()
	pipe.SRem(ctx, OnlinePlayersKey, playerID)
	sessionKey := fmt.Sprintf("%s%d", PlayerSessionKeyPrefix, playerID)
	pipe.Del(ctx, sessionKey)
	_, err := pipe.Exec(ctx)
	return err
}

// OnlineCount 返回当前在线玩家数。
func (m *Manager) OnlineCount(ctx context.Context) (int64, error) {
	return m.rdb.SCard(ctx, OnlinePlayersKey).Result()
}

// IsPlayerOnline 检查指定玩家是否在线。
func (m *Manager) IsPlayerOnline(ctx context.Context, playerID uint64) (bool, error) {
	return m.rdb.SIsMember(ctx, OnlinePlayersKey, playerID).Result()
}

// Publish 通过 Redis Pub/Sub 发布消息。
func (m *Manager) Publish(ctx context.Context, channel string, data []byte) error {
	return m.rdb.Publish(ctx, channel, data).Err()
}

// Subscribe 订阅 Redis Pub/Sub 频道，返回订阅对象和关闭函数。
func (m *Manager) Subscribe(ctx context.Context, channel string, handler func([]byte)) (func() error, error) {
	pubsub := m.rdb.Subscribe(ctx, channel)

	// 等待订阅确认
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("订阅 Redis 频道 %s 失败: %w", channel, err)
	}

	go func() {
		for msg := range pubsub.Channel() {
			handler([]byte(msg.Payload))
		}
	}()

	closeFn := func() error {
		return pubsub.Close()
	}
	return closeFn, nil
}

// Close 关闭 Redis 连接。
func (m *Manager) Close() error {
	return m.rdb.Close()
}
