package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/handler"
	"cultivation-game/services/combat/internal/repository"
	"cultivation-game/services/combat/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 初始化日志
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// 加载配置
	cfgPath := os.Getenv("COMBAT_CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.json"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg("加载配置失败")
	}
	log.Info().Msg("配置加载完成")

	// 初始化跨服务客户端
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://localhost:8083"
	}
	playerClient := repository.NewPlayerClient(playerAddr)

	// 初始化处理器
	pveHandler := handler.NewPVEHandler(cfg)
	pvpHandler := handler.NewPVPHandler(cfg, playerClient)
	arenaHandler := handler.NewArenaHandler(cfg)
	dungeonHandler := handler.NewDungeonHandler(cfg)
	towerHandler := handler.NewTowerHandler(cfg)
	sectDungeonSvc := service.NewSectDungeonService()
	sectDungeonSvc.Start()
	defer sectDungeonSvc.Stop()
	sectDungeonHandler := handler.NewSectDungeonHandler(sectDungeonSvc)

	teamDungeonSvc := service.NewTeamDungeonService()
	teamDungeonSvc.StartCleanupTask()
	teamDungeonHandler := handler.NewTeamDungeonHandler(teamDungeonSvc)

	// 加载 JSON 数据文件
	loadDataFiles(cfg)

	// 加载秘境副本数据
	if err := dungeonHandler.LoadDungeonData(cfg.DataPath.Dungeons); err != nil {
		log.Warn().Err(err).Str("path", cfg.DataPath.Dungeons).Msg("加载秘境数据失败")
	}

	// 设置路由
	r := gin.Default()

	// CORS 中间件
	r.Use(corsMiddleware())

	// PVE 路由
	pveGroup := r.Group("/api/v1/pve")
	{
		pveGroup.POST("/battle", pveHandler.StartBattle)
		pveGroup.GET("/monsters", pveHandler.GetMonsters)
		pveGroup.GET("/instances", pveHandler.GetInstances)
	}

	// Combat 路由（历练/扫荡）
	combatGroup := r.Group("/api/v1/combat")
	{
		combatGroup.GET("/monsters", pveHandler.GetMonsters)
		combatGroup.POST("/start", pveHandler.StartBattle)
		combatGroup.POST("/sweep", pveHandler.Sweep)
		combatGroup.GET("/dungeons", dungeonHandler.HandleListDungeons)
		combatGroup.POST("/dungeon/enter", dungeonHandler.HandleEnterDungeon)
		combatGroup.POST("/dungeon/fight", dungeonHandler.HandleFight)
		combatGroup.POST("/dungeon/claim", dungeonHandler.HandleClaimReward)
		combatGroup.GET("/dungeon/status", dungeonHandler.HandleStatus)
	}

	// PVP 路由
	pvpGroup := r.Group("/api/v1/pvp")
	{
		pvpGroup.POST("/join", pvpHandler.JoinQueue)
		pvpGroup.POST("/leave", pvpHandler.LeaveQueue)
		pvpGroup.GET("/queue-status", pvpHandler.QueueStatus)
		pvpGroup.POST("/action", pvpHandler.SubmitAction)
		pvpGroup.GET("/status", pvpHandler.GetBattleStatus)
		pvpGroup.GET("/rankings", pvpHandler.GetRankings)
	}

	// 竞技场路由
	arenaGroup := r.Group("/api/v1/arena")
	{
		arenaGroup.POST("/match", arenaHandler.HandleMatch)
		arenaGroup.GET("/status", arenaHandler.HandleStatus)
		arenaGroup.GET("/rankings", arenaHandler.HandleRankings)
		arenaGroup.POST("/cancel-match", arenaHandler.HandleCancelMatch)
		arenaGroup.GET("/season", arenaHandler.HandleSeason)
		arenaGroup.GET("/history", arenaHandler.HandleHistory)
	}

	// 心魔塔路由
	towerGroup := r.Group("/api/v1/tower")
	{
		towerGroup.POST("/enter", towerHandler.HandleEnter)
		towerGroup.POST("/fight", towerHandler.HandleFight)
		towerGroup.GET("/status", towerHandler.HandleStatus)
		towerGroup.GET("/ranking", towerHandler.HandleRanking)
	}

	// 宗门副本路由
	sectDungeonHandler.RegisterRoutes(r)

	// 组队副本路由
	teamDungeonHandler.RegisterRoutes(r)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "combat",
		})
	})

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info().Str("addr", addr).Msg("战斗服务启动中")

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 优雅关闭
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Info().Msg("收到关闭信号, 正在关闭服务...")
		srv.Close()
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("服务启动失败")
	}

	log.Info().Msg("战斗服务已关闭")
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// loadDataFiles 加载 JSON 数据文件
func loadDataFiles(cfg *config.Config) {
	// 加载怪物数据
	monsterData, err := os.ReadFile(cfg.DataPath.Monsters)
	if err != nil {
		log.Warn().Err(err).Str("path", cfg.DataPath.Monsters).Msg("加载怪物数据失败, 使用默认空数据")
	} else {
		var monsterMap struct {
			Monsters map[string]interface{} `json:"monsters"`
		}
		if err := json.Unmarshal(monsterData, &monsterMap); err != nil {
			log.Warn().Err(err).Msg("解析怪物数据失败")
		} else {
			log.Info().Int("count", len(monsterMap.Monsters)).Msg("怪物数据加载完成")
		}
	}

	// 加载技能数据
	skillData, err := os.ReadFile(cfg.DataPath.Skills)
	if err != nil {
		log.Warn().Err(err).Str("path", cfg.DataPath.Skills).Msg("加载技能数据失败")
	} else {
		var skillList []interface{}
		if err := json.Unmarshal(skillData, &skillList); err != nil {
			log.Warn().Err(err).Msg("解析技能数据失败")
		} else {
			log.Info().Int("count", len(skillList)).Msg("技能数据加载完成")
		}
	}

	// 加载副本数据
	instanceData, err := os.ReadFile(cfg.DataPath.Instances)
	if err != nil {
		log.Warn().Err(err).Str("path", cfg.DataPath.Instances).Msg("加载副本数据失败")
	} else {
		var instanceMap struct {
			Instances map[string]interface{} `json:"instances"`
		}
		if err := json.Unmarshal(instanceData, &instanceMap); err != nil {
			log.Warn().Err(err).Msg("解析副本数据失败")
		} else {
			log.Info().Int("count", len(instanceMap.Instances)).Msg("副本数据加载完成")
		}
	}
}
