// Package main 修仙游戏 - 修炼服务入口
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cultivation-game/services/cultivation/internal/config"
	"cultivation-game/services/cultivation/internal/handler"
	"cultivation-game/services/cultivation/internal/model"
	"cultivation-game/services/cultivation/internal/repository"
	"cultivation-game/services/cultivation/internal/repository/mysql"
	"cultivation-game/services/cultivation/internal/repository/redis"
	"cultivation-game/services/cultivation/internal/service"
	_ "github.com/go-sql-driver/mysql"
	redisCli "github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger.Info("修仙游戏修炼服务启动中")

	// ---- 初始化配置 ----
	dataDir := config.DefaultConfigDir()
	logger.Info("配置数据目录", "path", dataDir)

	cfgLoader := config.NewConfigLoader(logger, dataDir, config.LoadOptions{
		HotReload: true,
		DataDir:   dataDir,
	})

	if err := cfgLoader.Load(); err != nil {
		logger.Error("配置加载失败", "error", err)
		os.Exit(1)
	}
	logger.Info("配置加载成功",
		"realms", len(cfgLoader.GetConfig().Realms),
		"techniques", len(cfgLoader.GetConfig().Techniques))

	// ---- 加载服务器配置（MySQL/Redis 环境变量） ----
	svrCfg := config.LoadServerConfig()
	logger.Info("服务器配置", "mysql_dsn", maskDSN(svrCfg.MySQL.DSN), "redis_addr", svrCfg.Redis.Addr)

	// ---- 初始化 MySQL 连接 ----
	db, err := sql.Open("mysql", svrCfg.MySQL.DSN)
	if err != nil {
		logger.Error("MySQL打开连接失败", "error", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		logger.Warn("MySQL连接测试失败，服务将以降级模式运行", "error", err)
	} else {
		logger.Info("MySQL数据库连接成功")
	}

	// ---- 初始化 Redis 连接 ----
	rdb := redisCli.NewClient(&redisCli.Options{
		Addr:         svrCfg.Redis.Addr,
		Password:     svrCfg.Redis.Password,
		DB:           svrCfg.Redis.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Warn("Redis连接测试失败，服务将以降级模式运行", "error", err)
	} else {
		logger.Info("Redis缓存连接成功")
	}

	// ---- 初始化 Repository 层 ----
	mysqlRepo := mysql.NewPlayerRepo(db, logger)
	cache := redis.NewPlayerCache(rdb)
	playerStore := repository.NewPlayerRepository(logger, mysqlRepo, cache)

	// ---- 初始化事件总线 ----
	eventBus := handler.NewSimpleEventBus()

	// 订阅突破事件日志
	eventBus.Subscribe("player.breakthrough", func(data interface{}) {
		if evt, ok := data.(*model.BreakthroughEvent); ok {
			logger.Info("玩家突破事件", "player_id", evt.PlayerID, "new_realm_id", evt.NewRealmID)
		}
	})

	// 订阅渡劫事件日志
	eventBus.Subscribe("tribulation.*", func(data interface{}) {
		if evt, ok := data.(*model.TribulationEvent); ok {
			switch evt.Status {
			case "started":
				logger.Info("玩家开始渡劫公告", "player_name", evt.PlayerName, "type", evt.TypeName)
			case "success":
				logger.Info("玩家渡劫成功公告", "player_name", evt.PlayerName, "type", evt.TypeName)
			case "failed":
				logger.Info("玩家渡劫失败公告", "player_name", evt.PlayerName, "type", evt.TypeName)
			}
		}
	})

	// ---- 初始化服务 ----
	realmSvc := service.NewRealmService(cfgLoader, eventBus, rdb)
	techniqueSvc := service.NewTechniqueService(cfgLoader)
	tribulationMgr := service.NewTribulationManager(logger, cfgLoader, realmSvc, eventBus)
	tribulationSvc := service.NewTribulationService(logger, cfgLoader, realmSvc)
	breakthroughSvc := service.NewBreakthroughService(logger, cfgLoader, realmSvc, eventBus, playerStore, tribulationMgr)

	// ---- 设置新手保护检查器（跨服务调用 Player 服务） ----
	playerSvcAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerSvcAddr == "" {
		playerSvcAddr = "http://localhost:8083"
	}
	playerClient := repository.NewPlayerClient(playerSvcAddr)
	breakthroughSvc.SetProtectionChecker(playerClient)
	logger.Info("新手保护检查器已设置", "player_addr", playerSvcAddr)

	// ---- 初始化炼丹服务 ----
	alchemySvc := service.NewAlchemyService(dataDir)
	enhancedAlchemySvc := service.NewEnhancedAlchemyService()
	logger.Info("炼丹已加载",
		"recipes", len(alchemySvc.GetRecipes(&model.Player{RealmID: 99, AlchemyLevel: 99})),
		"ingredients", len(alchemySvc.GetAllIngredients()))
	logger.Info("炼丹增强已初始化",
		"formulas", len(enhancedAlchemySvc.GetAllFormulas()),
		"rare_formulas", func() int {
			rare := 0
			for _, f := range enhancedAlchemySvc.GetAllFormulas() {
				if f.IsRare {
					rare++
				}
			}
			return rare
		}())

	// ---- 初始化心魔系统服务 ----
	heartDemonSvc := service.NewHeartDemonService(logger, playerStore)
	breakthroughSvc.SetHeartDemonService(heartDemonSvc)
	logger.Info("心魔系统服务初始化完成")

	// ---- 初始化突破节点小游戏服务 ----
	nodeBreakthroughSvc := service.NewNodeBreakthroughService()
	logger.Info("突破节点小游戏引擎初始化完成")

	// ---- 初始化离线修炼服务 ----
	meditateSvc := service.NewMeditateService(logger, realmSvc, rdb)
	logger.Info("离线修炼服务初始化完成")

	// ---- 初始化游戏心跳服务 ----
	tickerSvc := service.NewTickerService(logger, meditateSvc, realmSvc)
	tickerSvc.Start()
	logger.Info("游戏心跳循环已启动", "interval_seconds", 60)

	// ---- 初始化HTTP处理器 ----
	cultivationHandler := handler.NewCultivationHandler(
		logger, realmSvc, techniqueSvc, breakthroughSvc, tribulationSvc, tribulationMgr, meditateSvc, nodeBreakthroughSvc, playerStore,
	)
	alchemyHandler := handler.NewAlchemyHandler(alchemySvc, enhancedAlchemySvc, playerStore)

	mux := http.NewServeMux()
	cultivationHandler.RegisterRoutes(mux)
	alchemyHandler.RegisterRoutes(mux)
	heartDemonHandler := handler.NewHeartDemonHandler(logger, heartDemonSvc, playerStore)
	heartDemonHandler.RegisterRoutes(mux)

	// 用认证中间件包装所有路由（/health 端点免认证）
	authMux := handler.AuthMiddleware(mux)

	// ---- 启动热重载 ----
	stopCh := make(chan struct{})
	go cfgLoader.WatchConfig(stopCh)

	// ---- 创建演示用角色（如果数据库不存在则创建） ----
	demoPlayer, err := playerStore.GetPlayer(1)
	if err != nil || demoPlayer == nil {
		demoPlayer = playerStore.CreatePlayer("散修张三", map[string]float64{
			"金": 0.4,
			"木": 0.6,
			"水": 0.3,
			"火": 0.2,
			"土": 0.5,
		})
		// 学习初始功法
		techniqueSvc.LearnTechnique(demoPlayer, 1) // 焚天诀
		atk, def, hp := realmSvc.CalculateStats(demoPlayer)
		demoPlayer.BaseAttack = atk
		demoPlayer.BaseDefense = def
		demoPlayer.BaseHP = hp
		playerStore.SavePlayer(demoPlayer)
		logger.Info("创建演示角色", "name", demoPlayer.Name, "player_id", demoPlayer.ID)
	} else {
		logger.Info("加载已有演示角色", "name", demoPlayer.Name, "player_id", demoPlayer.ID, "realm_id", demoPlayer.RealmID, "realm_level", demoPlayer.RealmLevel)
	}

	// ---- 启动HTTP服务器 ----
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      authMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅关闭
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("收到关闭信号，正在优雅关闭")
		close(stopCh)
		_ = rdb.Close()
		_ = db.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	logger.Info("HTTP服务启动", "port", port)
	logger.Info("API端点:")
		logger.Info("  POST /api/v1/player/create             创建角色")
		logger.Info("  GET  /api/v1/player/{id}                 查询角色")
		logger.Info("  POST /api/v1/cultivate                   修炼")
		logger.Info("  POST /api/v1/breakthrough                 突破")
		logger.Info("  GET  /api/v1/tribulation/info            天劫预览(V1)")
		logger.Info("  POST /api/v1/tribulation/start           开始渡劫(V2)")
		logger.Info("  POST /api/v1/tribulation/action          渡劫行动(V2)")
		logger.Info("  GET  /api/v1/tribulation/status          渡劫状态(V2)")
		logger.Info("  POST /api/v1/tribulation/guardian        护法加入(V2)")
		logger.Info("  POST /api/v1/tribulation/complete        渡劫完成(V2)")
		logger.Info("  GET  /api/v1/tribulation/preview         渡劫预览(V2)")
		logger.Info("  POST /api/v1/breakthrough/tribulation    大境界渡劫突破(V2)")
		logger.Info("  POST /api/v1/technique/learn             学习功法")
		logger.Info("  GET  /api/v1/techniques/available        可学功法")
		logger.Info("  POST /api/v1/cultivate (mode=offline)       离线修炼")
		logger.Info("  POST /api/v1/player/status                查询状态")
		logger.Info("  POST /api/v1/sync-exp                     同步修为(内部)")
		logger.Info("  POST /api/v1/alchemy/recipes             炼丹配方")
		logger.Info("  POST /api/v1/alchemy/craft               炼制丹药")
		logger.Info("  POST /api/v1/alchemy/collect             采集灵药")
		logger.Info("  GET  /api/v1/alchemy/ingredients         查看材料")
		logger.Info("  POST /api/v1/alchemy/research            研究丹方")
		logger.Info("  GET  /api/v1/alchemy/formulas            已研究丹方")
		logger.Info("  GET  /api/v1/alchemy/formulas/available  可研究丹方")
		logger.Info("  POST /api/v1/alchemy/start-craft         开始炼丹（小游戏）")
		logger.Info("  POST /api/v1/alchemy/minigame/heat       调整火候")
		logger.Info("  POST /api/v1/alchemy/minigame/add-material 添加材料")
		logger.Info("  POST /api/v1/alchemy/complete-craft      完成炼丹")
		logger.Info("  GET  /api/v1/alchemy/toxicity            丹毒值")
		logger.Info("  POST /api/v1/alchemy/detox               解毒")
		logger.Info("  GET  /api/v1/alchemy/furnace             丹炉信息")
		logger.Info("  POST /api/v1/alchemy/furnace/upgrade     升级丹炉")
		logger.Info("  GET  /api/v1/health                      健康检查")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("HTTP服务异常", "error", err); os.Exit(1)
	}

	logger.Info("修炼服务已停止")
}

// maskDSN 隐藏密码用于日志输出
func maskDSN(dsn string) string {
	// 简单实现：替换 password 部分
	// 完整的 DSN 格式: user:password@tcp(host:port)/dbname
	cleaned := []byte(dsn)
	atPos := -1
	colonPos := -1
	for i, c := range cleaned {
		if c == '@' {
			atPos = i
			break
		}
		if c == ':' {
			colonPos = i
		}
	}
	if colonPos >= 0 && atPos > colonPos {
		for i := colonPos + 1; i < atPos; i++ {
			cleaned[i] = '*'
		}
	}
	return string(cleaned)
}
