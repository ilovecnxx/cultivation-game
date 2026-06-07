// main 是世界服务的HTTP入口
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"cultivation-game/services/world/internal/config"
	"cultivation-game/services/world/internal/handler"
	"cultivation-game/services/world/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// 数据文件路径(相对于可执行文件或工作目录)
var (
	defaultRegionsPath    = "internal/data/map_regions.json"
	defaultNPCsPath       = "internal/data/npcs.json"
	defaultSpotsPath      = "internal/data/gathering_spots.json"
	defaultEncountersPath = "internal/data/encounters.json"
	defaultQuestsPath     = "internal/data/quests.json"
	defaultFishingPath  = "internal/data/fishing_spots.json"
	defaultDailyTasksPath = "internal/data/daily_tasks.json"
)

// findDataPath 查找数据文件路径
func findDataPath(relativePath string) string {
	// 如果文件存在直接返回
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath
	}

	// 尝试从可执行文件所在目录查找
	exe, err := os.Executable()
	if err == nil {
		altPath := filepath.Join(filepath.Dir(exe), relativePath)
		if _, err := os.Stat(altPath); err == nil {
			return altPath
		}
	}

	// 尝试从父目录查找
	parentPath := filepath.Join("..", "..", relativePath)
	if _, err := os.Stat(parentPath); err == nil {
		return parentPath
	}

	return relativePath
}

func main() {
	// 加载配置
	cfg := config.DefaultConfig()
	log.Printf("世界服务启动中，端口: %d", cfg.Server.Port)

	// 解析数据文件路径
	regionsPath := findDataPath(defaultRegionsPath)
	npcsPath := findDataPath(defaultNPCsPath)
	spotsPath := findDataPath(defaultSpotsPath)
	encountersPath := findDataPath(defaultEncountersPath)
	questsPath := findDataPath(defaultQuestsPath)
	fishingPath := findDataPath(defaultFishingPath)
	dailyTasksPath := findDataPath(defaultDailyTasksPath)

	log.Printf("加载数据文件:")
	log.Printf("  区域配置: %s", regionsPath)
	log.Printf("  NPC配置: %s", npcsPath)
	log.Printf("  采集点配置: %s", spotsPath)
	log.Printf("  奇遇配置: %s", encountersPath)
	log.Printf("  任务配置: %s", questsPath)
	log.Printf("  钓鱼配置: %s", fishingPath)
	log.Printf("  每日任务配置: %s", dailyTasksPath)

	// 连接 Redis（用于持久化玩家探索状态）
	var rdb *redis.Client
	rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("Redis 连接失败（将回退到文件存储）: %v", err)
		rdb = nil
	} else {
		log.Println("Redis 连接成功")
		defer rdb.Close()
	}

	// 确定数据目录（用于文件存储回退）
	dataDir := filepath.Dir(regionsPath)

	// 创建探索服务
	exploreSvc, err := service.NewExploreService(regionsPath, npcsPath, spotsPath, rdb, dataDir)
	if err != nil {
		log.Fatalf("创建探索服务失败: %v", err)
	}
	log.Printf("探索服务初始化完成，已加载 %d 个区域", len(exploreSvc.GetAllRegions()))

	// 创建奇遇服务
	encounterSvc, err := service.NewEncounterService(encountersPath, exploreSvc)
	if err != nil {
		log.Fatalf("创建奇遇服务失败: %v", err)
	}
	log.Printf("奇遇服务初始化完成，已加载 %d 个奇遇事件", len(encounterSvc.GetAllEncounters()))

	// 创建任务服务
	questSvc, err := service.NewQuestService(questsPath)
	if err != nil {
		log.Fatalf("创建任务服务失败: %v", err)
	}
	log.Printf("任务服务初始化完成，已加载 %d 个任务", len(questSvc.GetAllQuests()))

	// 加载每日任务配置
	if err := questSvc.LoadDailyTasks(dailyTasksPath); err != nil {
		log.Printf("加载每日任务配置失败: %v", err)
	} else {
		log.Printf("每日任务配置加载完成，已加载 %d 个每日任务", len(questSvc.GetDailyTaskDefs()))
	}

	// 创建灵气浓度服务
	spiritDensitySvc, err := service.NewSpiritDensityService(regionsPath, 50)
	if err != nil {
		log.Printf("创建灵气浓度服务失败（使用默认值）: %v", err)
	}
	log.Printf("灵气浓度服务初始化完成")

	// 创建灵脉争夺服务
	veinSvc, err := service.NewSpiritVeinService(rdb, dataDir)
	if err != nil {
		log.Fatalf("创建灵脉争夺服务失败: %v", err)
	}
	log.Printf("灵脉争夺服务初始化完成")

	// 创建钓鱼服务
	fishingSvc, err := service.NewFishingService(fishingPath)
	if err != nil {
		log.Fatalf("创建钓鱼服务失败: %v", err)
	}
	log.Printf("钓鱼服务初始化完成，已加载 %d 个钓鱼点", len(fishingSvc.GetSpots()))

	// 创建HTTP处理器
	worldHandler := handler.NewWorldHandler(exploreSvc, encounterSvc, spiritDensitySvc)
	questHandler := handler.NewQuestHandler(questSvc)
	veinHandler := handler.NewVeinHandler(veinSvc)
	fishingHandler := handler.NewFishingHandler(fishingSvc)
	bossSvc := service.NewWorldBossService()
	bossHandler := handler.NewWorldBossHandler(bossSvc)
	bossSvc.Start()
	defer bossSvc.Stop()

	// 创建路由
	r := gin.Default()
	r.Use(corsMiddleware())
	worldHandler.RegisterRoutes(r)
	questHandler.RegisterRoutes(r)
	veinHandler.RegisterRoutes(r)
	bossHandler.RegisterRoutes(r)
	fishingHandler.RegisterRoutes(r)

	// 创建HTTP服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器(goroutine)
	go func() {
		log.Printf("HTTP 服务器启动在 %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务器强制关闭: %v", err)
	}

	log.Println("服务器已安全退出")
}

// corsMiddleware 跨域中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}
