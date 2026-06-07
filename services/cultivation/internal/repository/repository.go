// Package repository 提供修炼系统数据存储整合层
package repository

import (
	"log/slog"
	"time"

	"cultivation-game/services/cultivation/internal/model"
	"cultivation-game/services/cultivation/internal/repository/mysql"
	"cultivation-game/services/cultivation/internal/repository/redis"
)

// PlayerRepository 实现 handler.PlayerStore，整合 MySQL + Redis
// 读写分离策略：
//   - 读取：先查 Redis 缓存，未命中则查 MySQL 并回填缓存
//   - 写入：先写入 Redis（同步），再异步写入 MySQL
type PlayerRepository struct {
	logger *slog.Logger
	mysql *mysql.PlayerRepo
	cache *redis.PlayerCache
}

// NewPlayerRepository 创建 PlayerRepository
func NewPlayerRepository(logger *slog.Logger, mysqlRepo *mysql.PlayerRepo, cache *redis.PlayerCache) *PlayerRepository {
	return &PlayerRepository{
		mysql:  mysqlRepo,
		cache:  cache,
		logger: logger,
	}
}

// GetPlayer 获取玩家数据（缓存优先）
func (r *PlayerRepository) GetPlayer(id uint64) (*model.Player, error) {
	// 1. 尝试从缓存读取
	p, err := r.cache.GetCachedPlayer(id)
	if err != nil {
		r.logger.Warn("读取缓存失败，回查 MySQL", "error", err)
	} else if p != nil {
		return p, nil
	}

	// 2. 缓存未命中，从 MySQL 读取
	p, err = r.mysql.GetPlayer(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}

	// 3. 回填缓存（异步，不阻塞）
	go func() {
		if err := r.cache.CachePlayer(p); err != nil {
			r.logger.Error("回填玩家缓存失败", "player_id", id, "error", err)
		}
	}()

	return p, nil
}

// SavePlayer 保存玩家数据（先写缓存，再异步写 MySQL）
func (r *PlayerRepository) SavePlayer(player *model.Player) error {
	// 1. 同步写入缓存
	if err := r.cache.CachePlayer(player); err != nil {
		return err
	}

	// 2. 异步写入 MySQL
	go func() {
		if err := r.mysql.SavePlayer(player); err != nil {
			r.logger.Error("异步保存玩家到MySQL失败", "player_id", player.ID, "error", err)
		}
	}()

	return nil
}

// CreatePlayer 创建新玩家
// 先在 MySQL 创建以获取自增 ID，再缓存到 Redis
func (r *PlayerRepository) CreatePlayer(name string, spiritRoots map[string]float64) *model.Player {
	p := &model.Player{
		Name:         name,
		RealmID:      1,
		RealmLevel:   1,
		Experience:   0,
		SpiritRoots:  spiritRoots,
		PillBonuses:  make(map[string]float64),
		ArtifactBonuses: make(map[string]float64),
		AlchemyLevel: 1,
		Ingredients:  make(map[int]int),
		Pills:        []model.Pill{},
		Status:       "idle",
	}

	if p.SpiritRoots == nil {
		p.SpiritRoots = make(map[string]float64)
	}

	// MySQL 创建获取自增 ID
	id, err := r.mysql.CreatePlayer(p)
	if err != nil {
		r.logger.Error("MySQL创建玩家失败，使用时间戳ID降级", "error", err)
		p.ID = uint64(time.Now().UnixMilli())
		return p
	}

	p.ID = id

	// 缓存到 Redis（异步）
	go func() {
		if err := r.cache.CachePlayer(p); err != nil {
			r.logger.Error("缓存新玩家失败", "player_id", id, "error", err)
		}
	}()

	return p
}

// Ping 健康检查：检查 MySQL 和 Redis 连接
func (r *PlayerRepository) Ping() error {
	if err := r.mysql.Ping(); err != nil {
		return err
	}
	return r.cache.Ping()
}
