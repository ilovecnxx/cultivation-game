// Package rankingadapter wraps ranking service internals for monolith consumption.
package rankingadapter

import (
	"log/slog"

	"cultivation-game/services/ranking/internal/config"
	"cultivation-game/services/ranking/internal/handler"
	redisRepo "cultivation-game/services/ranking/internal/repository/redis"
	"cultivation-game/services/ranking/internal/service"

	"github.com/redis/go-redis/v9"
)

// Components holds initialized ranking service components.
type Components struct {
	Handler *handler.RankingHandler
	Service *service.RankingService
}

// Bootstrap initializes the ranking service layer.
// The constructor auto-starts background workers.
func Bootstrap(rdb *redis.Client, playerAddr string) *Components {
	log := slog.Default()
	cfg := config.Load()
	cfg.PlayerServiceAddr = playerAddr
	rankingRepo := redisRepo.NewRankingRepo(rdb, log)
	rankingSvc := service.NewRankingService(rankingRepo, cfg, log)
	rankingHandler := handler.NewRankingHandler(rankingSvc, log)

	return &Components{
		Handler: rankingHandler,
		Service: rankingSvc,
	}
}
