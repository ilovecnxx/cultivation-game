// Package cultivationadapter wraps cultivation service internals for monolith.
package cultivationadapter

import (
	"database/sql"
	"log/slog"
	"net/http"

	"cultivation-game/services/cultivation/internal/config"
	"cultivation-game/services/cultivation/internal/handler"
	"cultivation-game/services/cultivation/internal/repository"
	mysqlRepo "cultivation-game/services/cultivation/internal/repository/mysql"
	redisRepo "cultivation-game/services/cultivation/internal/repository/redis"
	"cultivation-game/services/cultivation/internal/service"

	"github.com/redis/go-redis/v9"
)

// Components holds initialized cultivation components.
type Components struct {
	Mux           http.Handler
	CfgLoader     *config.ConfigLoader
	RealmSvc      *service.RealmService
	TechniqueSvc  *service.TechniqueService
	TickerSvc     *service.TickerService
	MeditateSvc   *service.MeditateService
}

// Bootstrap initializes the cultivation service.
func Bootstrap(db *sql.DB, rdb *redis.Client, dataDir string, playerAddr string) *Components {
	log := slog.Default()

	cfgLoader := config.NewConfigLoader(log, dataDir, config.LoadOptions{
		HotReload: true,
		DataDir:   dataDir,
	})
	if err := cfgLoader.Load(); err != nil {
		slog.Warn("failed to load cultivation config", "error", err)
	}

	mysqlRepo := mysqlRepo.NewPlayerRepo(db, log)
	cache := redisRepo.NewPlayerCache(rdb)
	playerStore := repository.NewPlayerRepository(log, mysqlRepo, cache)

	eventBus := handler.NewSimpleEventBus()

	realmSvc := service.NewRealmService(cfgLoader, eventBus, rdb)
	techniqueSvc := service.NewTechniqueService(cfgLoader)
	tribulationMgr := service.NewTribulationManager(log, cfgLoader, realmSvc, eventBus)
	tribulationSvc := service.NewTribulationService(log, cfgLoader, realmSvc)
	breakthroughSvc := service.NewBreakthroughService(log, cfgLoader, realmSvc, eventBus, playerStore, tribulationMgr)

	playerClient := repository.NewPlayerClient(playerAddr)
	breakthroughSvc.SetProtectionChecker(playerClient)

	alchemySvc := service.NewAlchemyService(dataDir)
	enhancedAlchemySvc := service.NewEnhancedAlchemyService()

	heartDemonSvc := service.NewHeartDemonService(log, playerStore)
	breakthroughSvc.SetHeartDemonService(heartDemonSvc)

	nodeBreakthroughSvc := service.NewNodeBreakthroughService()
	meditateSvc := service.NewMeditateService(log, realmSvc, rdb)
	tickerSvc := service.NewTickerService(log, meditateSvc, realmSvc)
	tickerSvc.Start()

	cultHandler := handler.NewCultivationHandler(
		log, realmSvc, techniqueSvc, breakthroughSvc,
		tribulationSvc, tribulationMgr, meditateSvc, nodeBreakthroughSvc, playerStore,
	)
	alchemyHandler := handler.NewAlchemyHandler(alchemySvc, enhancedAlchemySvc, playerStore)
	heartDemonHandler := handler.NewHeartDemonHandler(log, heartDemonSvc, playerStore)

	mux := http.NewServeMux()
	cultHandler.RegisterRoutes(mux)
	alchemyHandler.RegisterRoutes(mux)
	heartDemonHandler.RegisterRoutes(mux)
	authMux := handler.AuthMiddleware(mux)

	return &Components{
		Mux:          authMux,
		CfgLoader:    cfgLoader,
		RealmSvc:     realmSvc,
		TechniqueSvc: techniqueSvc,
		TickerSvc:    tickerSvc,
		MeditateSvc:  meditateSvc,
	}
}
