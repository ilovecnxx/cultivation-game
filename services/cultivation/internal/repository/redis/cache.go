// Package redis 提供修炼系统 Redis 缓存层
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cultivation-game/services/cultivation/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	playerKeyPrefix = "cultivation:player:"
	playerTTL       = 15 * time.Minute // 玩家缓存 TTL
)

// PlayerCache Redis 缓存层
type PlayerCache struct {
	client *redis.Client
}

// NewPlayerCache 创建 PlayerCache
func NewPlayerCache(client *redis.Client) *PlayerCache {
	return &PlayerCache{client: client}
}

// playerKey 生成玩家缓存 key
func playerKey(id uint64) string {
	return fmt.Sprintf("%s%d", playerKeyPrefix, id)
}

// PlayerCacheData 缓存用的玩家数据结构（轻量级）
type PlayerCacheData struct {
	ID               uint64             `json:"id"`
	Name             string             `json:"name"`
	RealmID          int               `json:"realm_id"`
	RealmLevel       int               `json:"realm_level"`
	Experience       int64             `json:"experience"`
	BaseAttack       int64             `json:"base_attack"`
	BaseDefense      int64             `json:"base_defense"`
	BaseHP           int64             `json:"base_hp"`
	TechniqueID      int               `json:"technique_id"`
	TechniqueLevel   int               `json:"technique_level"`
	SpiritRoots      map[string]float64 `json:"spirit_roots"`
	IsMeditating     bool              `json:"is_meditating"`
	MeditationStart  int64             `json:"meditation_start"`
	AccumulatedExp   int64             `json:"accumulated_exp"`
	PillBonuses      map[string]float64 `json:"pill_bonuses"`
	ArtifactBonuses  map[string]float64 `json:"artifact_bonuses"`
}

// toCacheData 将 Player 转为缓存数据
func toCacheData(p *model.Player) *PlayerCacheData {
	return &PlayerCacheData{
		ID:              p.ID,
		Name:            p.Name,
		RealmID:         p.RealmID,
		RealmLevel:      p.RealmLevel,
		Experience:      p.Experience,
		BaseAttack:      p.BaseAttack,
		BaseDefense:     p.BaseDefense,
		BaseHP:          p.BaseHP,
		TechniqueID:     p.TechniqueID,
		TechniqueLevel:  p.TechniqueLevel,
		SpiritRoots:     p.SpiritRoots,
		IsMeditating:    p.IsMeditating,
		MeditationStart: p.MeditationStart,
		AccumulatedExp:  p.AccumulatedExp,
		PillBonuses:     p.PillBonuses,
		ArtifactBonuses: p.ArtifactBonuses,
	}
}

// toPlayer 将缓存数据还原为 Player
func toPlayer(d *PlayerCacheData) *model.Player {
	p := &model.Player{
		ID:              d.ID,
		Name:            d.Name,
		RealmID:         d.RealmID,
		RealmLevel:      d.RealmLevel,
		Experience:      d.Experience,
		BaseAttack:      d.BaseAttack,
		BaseDefense:     d.BaseDefense,
		BaseHP:          d.BaseHP,
		TechniqueID:     d.TechniqueID,
		TechniqueLevel:  d.TechniqueLevel,
		IsMeditating:    d.IsMeditating,
		MeditationStart: d.MeditationStart,
		AccumulatedExp:  d.AccumulatedExp,
	}
	if d.SpiritRoots != nil {
		p.SpiritRoots = d.SpiritRoots
	} else {
		p.SpiritRoots = make(map[string]float64)
	}
	if d.PillBonuses != nil {
		p.PillBonuses = d.PillBonuses
	} else {
		p.PillBonuses = make(map[string]float64)
	}
	if d.ArtifactBonuses != nil {
		p.ArtifactBonuses = d.ArtifactBonuses
	} else {
		p.ArtifactBonuses = make(map[string]float64)
	}
	return p
}

// CachePlayer 缓存玩家数据（TTL 15 分钟）
func (c *PlayerCache) CachePlayer(p *model.Player) error {
	ctx := context.Background()
	data := toCacheData(p)
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化玩家%d缓存失败: %w", p.ID, err)
	}

	if err := c.client.Set(ctx, playerKey(p.ID), payload, playerTTL).Err(); err != nil {
		return fmt.Errorf("写入玩家%d缓存失败: %w", p.ID, err)
	}
	return nil
}

// GetCachedPlayer 获取缓存玩家，返回 nil 表示缓存未命中
func (c *PlayerCache) GetCachedPlayer(id uint64) (*model.Player, error) {
	ctx := context.Background()
	data, err := c.client.Get(ctx, playerKey(id)).Bytes()
	if err == redis.Nil {
		return nil, nil // 缓存未命中
	}
	if err != nil {
		return nil, fmt.Errorf("读取玩家%d缓存失败: %w", id, err)
	}

	var cacheData PlayerCacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return nil, fmt.Errorf("反序列化玩家%d缓存失败: %w", id, err)
	}

	return toPlayer(&cacheData), nil
}

// InvalidatePlayer 清除玩家缓存
func (c *PlayerCache) InvalidatePlayer(id uint64) error {
	ctx := context.Background()
	if err := c.client.Del(ctx, playerKey(id)).Err(); err != nil {
		return fmt.Errorf("清除玩家%d缓存失败: %w", id, err)
	}
	return nil
}

// Ping 健康检查
func (c *PlayerCache) Ping() error {
	ctx := context.Background()
	return c.client.Ping(ctx).Err()
}
