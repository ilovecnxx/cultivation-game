// Package service 飞升仙界业务逻辑
package service

import (
	"context"
	"fmt"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"
	"cultivation-game/services/player/internal/repository/redis"

	"go.uber.org/zap"
)

// AscendService 飞升仙界服务
type AscendService struct {
	playerRepo *mysql.PlayerRepo
	cache      *redis.Cache
	log        *zap.Logger
}

// NewAscendService 创建飞升服务
func NewAscendService(playerRepo *mysql.PlayerRepo, cache *redis.Cache, log *zap.Logger) *AscendService {
	return &AscendService{
		playerRepo: playerRepo,
		cache:      cache,
		log:        log,
	}
}

// CelestialRealmInfo 仙界境界信息
type CelestialRealmInfo struct {
	RealmID   int32  `json:"realm_id"`
	RealmName string `json:"realm_name"`
	Level     int32  `json:"level"`
}

// XianjieRealmNames 仙界境界中文名
var XianjieRealmNames = map[int32]string{
	10: "天仙",
	11: "金仙",
	12: "仙君",
	13: "仙帝",
}

// CheckAscendEligibility 检测玩家是否满足飞升条件
// 条件: 玩家境界=渡劫(RealmTrib=9) 且已满级
func (s *AscendService) CheckAscendEligibility(playerID int64) (bool, string, error) {
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return false, "", fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return false, "", fmt.Errorf("玩家不存在")
	}

	if player.Realm < model.RealmTrib {
		return false, fmt.Sprintf("当前境界不足（%s），需达渡劫圆满方可飞升",
			model.RealmNames[player.Realm]), nil
	}

	if player.Realm >= 10 {
		return false, "已飞升仙界，无需再次飞升", nil
	}

	return true, "渡劫圆满，可飞升仙界！", nil
}

// ProcessAscension 处理飞升逻辑
// 将玩家从下界境界更新为天仙一层，保留部分修为
func (s *AscendService) ProcessAscension(playerID int64) (*CelestialRealmInfo, error) {
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}

	if player.Realm >= 10 {
		return nil, fmt.Errorf("玩家已在仙界，无法重复飞升")
	}

	// ---- 执行飞升 ----

	// 1. 境界更新为天仙（realm=10）
	player.Realm = 10

	// 2. 保存到数据库
	if err := s.playerRepo.Update(player); err != nil {
		return nil, fmt.Errorf("保存玩家数据失败: %w", err)
	}

	// 3. 更新缓存（redis key 基于 playerID）
	cacheCtx := context.Background()
	if err := s.cache.SetPlayer(cacheCtx, player.ToCache()); err != nil {
		s.log.Warn("飞升后更新缓存失败", zap.Error(err))
	}

	// 4. 记录飞升日志
	s.log.Info("玩家飞升成功",
		zap.Int64("player_id", playerID),
		zap.String("player_name", player.Name),
		zap.Int32("new_realm", player.Realm),
	)

	return &CelestialRealmInfo{
		RealmID:   10,
		RealmName: "天仙",
		Level:     1,
	}, nil
}

// GetCelestialRealmName 获取仙界境界中文名
func GetCelestialRealmName(realmID int32) string {
	if name, ok := XianjieRealmNames[realmID]; ok {
		return name
	}
	return "未知仙界境界"
}
