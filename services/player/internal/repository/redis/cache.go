package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	playerKeyPrefix = "player:"
	playerTTL       = 30 * time.Minute // 玩家缓存 TTL
)

// Cache Redis 缓存层
type Cache struct {
	client *redis.Client
	log    *zap.Logger
}

// NewCache 创建 Redis 缓存
func NewCache(client *redis.Client, log *zap.Logger) *Cache {
	return &Cache{client: client, log: log}
}

// playerKey 生成玩家缓存 key
func playerKey(playerID int64) string {
	return fmt.Sprintf("%s%d", playerKeyPrefix, playerID)
}

// SetPlayer 写入玩家缓存（Write-Behind：写入缓存即成功，异步刷 DB）
func (c *Cache) SetPlayer(ctx context.Context, p *model.PlayerCache) error {
	data, err := json.Marshal(p)
	if err != nil {
		c.log.Error("序列化玩家缓存失败", zap.Error(err))
		return fmt.Errorf("序列化缓存失败: %w", err)
	}

	if err := c.client.Set(ctx, playerKey(p.ID), data, playerTTL).Err(); err != nil {
		c.log.Error("写入玩家缓存失败", zap.Error(err))
		return fmt.Errorf("写入缓存失败: %w", err)
	}
	return nil
}

// GetPlayer 读取玩家缓存
func (c *Cache) GetPlayer(ctx context.Context, playerID int64) (*model.PlayerCache, error) {
	data, err := c.client.Get(ctx, playerKey(playerID)).Bytes()
	if err == redis.Nil {
		return nil, nil // 缓存未命中
	}
	if err != nil {
		c.log.Error("读取玩家缓存失败", zap.Error(err))
		return nil, fmt.Errorf("读取缓存失败: %w", err)
	}

	cache := &model.PlayerCache{}
	if err := json.Unmarshal(data, cache); err != nil {
		c.log.Error("反序列化玩家缓存失败", zap.Error(err))
		return nil, fmt.Errorf("反序列化缓存失败: %w", err)
	}
	return cache, nil
}

// DelPlayer 删除玩家缓存
func (c *Cache) DelPlayer(ctx context.Context, playerID int64) error {
	if err := c.client.Del(ctx, playerKey(playerID)).Err(); err != nil {
		c.log.Error("删除玩家缓存失败", zap.Error(err))
		return fmt.Errorf("删除缓存失败: %w", err)
	}
	return nil
}

// RefreshTTL 刷新玩家缓存 TTL
func (c *Cache) RefreshTTL(ctx context.Context, playerID int64) error {
	if err := c.client.Expire(ctx, playerKey(playerID), playerTTL).Err(); err != nil {
		c.log.Warn("刷新缓存TTL失败", zap.Error(err))
		return err
	}
	return nil
}

// Exists 检查缓存是否存在
func (c *Cache) Exists(ctx context.Context, playerID int64) (bool, error) {
	n, err := c.client.Exists(ctx, playerKey(playerID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// SetInventoryCache 写入背包缓存（简化为 JSON 字符串）
func (c *Cache) SetInventoryCache(ctx context.Context, playerID int64, items []*model.InventoryItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("序列化背包缓存失败: %w", err)
	}
	key := fmt.Sprintf("inventory:%d", playerID)
	return c.client.Set(ctx, key, data, 15*time.Minute).Err()
}

// GetInventoryCache 读取背包缓存
func (c *Cache) GetInventoryCache(ctx context.Context, playerID int64) ([]*model.InventoryItem, error) {
	key := fmt.Sprintf("inventory:%d", playerID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var items []*model.InventoryItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Ping 健康检查
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
