// Package tradeadapter wraps trade service internals for monolith consumption.
package tradeadapter

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"

	"cultivation-game/services/trade/internal/config"
	"cultivation-game/services/trade/internal/handler"
	mysqlRepo "cultivation-game/services/trade/internal/repository/mysql"
	redisRepo "cultivation-game/services/trade/internal/repository/redis"
	"cultivation-game/services/trade/internal/service"
	shopHandler "cultivation-game/services/trade/internal/shop/handler"
	shopService "cultivation-game/services/trade/internal/shop/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Components holds initialized trade service components.
type Components struct {
	Handler          *handler.TradeHandler
	BlackMarket      *handler.BlackMarketHandler
	ShopHandler      *shopHandler.ShopHandler
	AuctionCancel    context.CancelFunc
}

// Bootstrap initializes the trade service layer and returns components.
func Bootstrap(db *sql.DB, rdb *redis.Client) *Components {
	log := slog.Default()
	cfg := config.Load()
	tradeRepo := mysqlRepo.NewTradeRepo(db, log)
	cache := redisRepo.NewCacheRepo(rdb, log)
	marketSvc := service.NewMarketService(tradeRepo, cache, cfg, log)
	auctionSvc := service.NewAuctionService(tradeRepo, cache, cfg, log)
	blackMarketSvc := service.NewBlackMarketService()

	// Shop service needs a data directory
	dataDir := os.Getenv("TRADE_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join("..", "trade", "internal", "data")
	}
	shopSvc, err := shopService.NewShopService(dataDir, log)
	if err != nil {
		log.Warn("failed to initialize shop service, using degraded mode", "error", err)
	}

	tradeHandler := handler.NewTradeHandler(marketSvc, auctionSvc, log)
	blackMarketHandler := handler.NewBlackMarketHandler(blackMarketSvc, log)
	shopH := shopHandler.NewShopHandler(shopSvc, log)

	// Start auction expiry loop
	auctionCtx, auctionCancel := context.WithCancel(context.Background())
	go auctionSvc.StartAuctionExpiryLoop(auctionCtx)

	return &Components{
		Handler:       tradeHandler,
		BlackMarket:   blackMarketHandler,
		ShopHandler:   shopH,
		AuctionCancel: auctionCancel,
	}
}

// RegisterShopRoutes registers shop routes on the given router group.
func (c *Components) RegisterShopRoutes(rg *gin.RouterGroup) {
	c.ShopHandler.RegisterRoutes(rg)
}
