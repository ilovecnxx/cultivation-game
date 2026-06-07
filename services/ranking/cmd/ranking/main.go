// Command ranking 修仙游戏排行榜服务 HTTP 入口。
//
// 启动流程：
//   1. 加载配置（环境变量）
//   2. 初始化结构化日志
//   3. 连接 Redis
//   4. 初始化 Repository、Service、Handler 三层
//   5. 从玩家服务拉取初始数据，预热排行榜
//   6. 启动定时同步（每 5 分钟从玩家服务更新数据）
//   7. 启动 HTTP 服务器
//   8. 监听系统信号，优雅关闭
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"cultivation-game/services/ranking/internal/config"
	"cultivation-game/services/ranking/internal/handler"
	"cultivation-game/services/ranking/internal/model"
	redisrepo "cultivation-game/services/ranking/internal/repository/redis"
	"cultivation-game/services/ranking/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// syncInterval 定时同步间隔（5 分钟）。
const syncInterval = 5 * time.Minute

// playerData 从玩家服务获取的玩家基础数据（响应中的 data.player 字段映射）。
type playerData struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Level       int32  `json:"level"`
	Realm       int32  `json:"realm"`
	Attack      int64  `json:"attack"`
	Defense     int64  `json:"defense"`
	SpiritPower int64  `json:"spirit_power"`
	Gold        int64  `json:"gold"`
}

// playerListResponse 玩家列表接口的通用响应格式。
type playerListResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data *playerList `json:"data,omitempty"`
}

// playerList 玩家列表分页数据。
type playerList struct {
	Players []*playerData `json:"players"`
	Total   int           `json:"total"`
	Page    int           `json:"page"`
}

// playerResponse 单个玩家接口的响应格式。
type playerResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data *playerWrapData `json:"data,omitempty"`
}

// playerWrapData 单玩家接口的 data 字段。
type playerWrapData struct {
	Player *playerData `json:"player"`
}

func main() {
	// =========================================================
	// 1. 加载配置
	// =========================================================
	cfg := config.Load()

	// =========================================================
	// 2. 初始化日志
	// =========================================================
	logLevel := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})))
	addr := fmt.Sprintf(":%d", cfg.Port)
	slog.Info("排行榜服务启动中", "listen_addr", addr, "player_service", cfg.PlayerServiceAddr)

	// =========================================================
	// 3. 连接 Redis
	// =========================================================
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer rdb.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		pingCancel()
		slog.Error("Ping Redis 失败", "error", err)
		os.Exit(1)
	}
	pingCancel()
	slog.Info("Redis 连接成功", "addr", cfg.RedisAddr, "db", cfg.RedisDB)

	// =========================================================
	// 4. 初始化三层架构
	// =========================================================
	rankingRepo := redisrepo.NewRankingRepo(rdb, slog.Default())
	rankingService := service.NewRankingService(rankingRepo, cfg, slog.Default())
	rankingHandler := handler.NewRankingHandler(rankingService, slog.Default())

	// =========================================================
	// 5. 数据预热：从玩家服务拉取初始数据
	// =========================================================
	slog.Info("开始排行榜数据预热")
	preheatCtx, preheatCancel := context.WithTimeout(context.Background(), 30*time.Second)
	loadPlayersFromService(preheatCtx, cfg.PlayerServiceAddr, rankingService)
	preheatCancel()
	slog.Info("排行榜数据预热完成")

	// =========================================================
	// 6. 启动定时同步 goroutine
	// =========================================================
	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		syncLoop(stopCh, cfg.PlayerServiceAddr, rankingService)
	}()

	// =========================================================
	// 7. 注册路由并启动 HTTP 服务器
	// =========================================================
	r := gin.Default()
	r.Use(corsMiddleware())
	rankingHandler.RegisterRoutes(r)

	// 注册额外的健康检查路由（兼容旧路径）
	r.GET("/health", func(c *gin.Context) {
		redisStatus := "ok"
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			redisStatus = "degraded: " + err.Error()
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
			"data": gin.H{
				"service": "ranking-service",
				"status":  "running",
				"redis":   redisStatus,
			},
		})
	})

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("HTTP 服务器已启动", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP 服务器异常退出", "error", err)
			os.Exit(1)
		}
	}()

	// =========================================================
	// 8. 等待信号，优雅关闭
	// =========================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("正在关闭服务", "signal", sig)

	// 停止后台 Worker
	rankingService.Stop()
	close(stopCh)
	wg.Wait()

	// 优雅关闭 HTTP 服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("HTTP 服务器关闭超时", "error", err)
	}

	rdb.Close()
	slog.Info("排行榜服务已停止")
}

// ============================================================
// 数据加载与同步
// ============================================================

// loadPlayersFromService 从玩家服务拉取所有玩家数据，写入排行榜。
// 先尝试分页列表接口，回退到逐个 ID 查询。
func loadPlayersFromService(ctx context.Context, playerSvcAddr string, svc *service.RankingService) {
	players := fetchAllPlayers(ctx, playerSvcAddr)
	if len(players) == 0 {
		slog.Warn("玩家服务未返回数据，排行榜从空数据开始")
		return
	}

	batchUpdateRankings(ctx, svc, players)
	slog.Info("排行榜数据加载完成", "player_count", len(players))
}

// syncLoop 定时同步循环，每 syncInterval 从玩家服务拉取一次数据。
func syncLoop(stopCh chan struct{}, playerSvcAddr string, svc *service.RankingService) {
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			slog.Info("开始定时同步排行榜数据")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			players := fetchAllPlayers(ctx, playerSvcAddr)
			if len(players) > 0 {
				batchUpdateRankings(ctx, svc, players)
				slog.Info("定时同步完成", "player_count", len(players))
			} else {
				slog.Warn("定时同步未获取到玩家数据")
			}
			cancel()

		case <-stopCh:
			slog.Info("定时同步已停止")
			return
		}
	}
}

// fetchAllPlayers 分页拉取所有玩家数据。
// 优先使用 /api/v1/player/list 分页接口，若不可用则回退到单个玩家查询。
func fetchAllPlayers(ctx context.Context, playerSvcAddr string) []*playerData {
	// 优先尝试列表接口
	players := fetchPlayerList(ctx, playerSvcAddr)
	if len(players) > 0 {
		return players
	}

	// 回退策略：尝试逐个查询 ID 1~500
	slog.Info("玩家列表接口不可用，回退到逐个查询模式")
	players = fetchPlayersByRange(ctx, playerSvcAddr, 1, 500)
	return players
}

// fetchPlayerList 调用玩家服务列表接口获取分页数据。
// GET /api/v1/player/list?page=1&page_size=1000
func fetchPlayerList(ctx context.Context, baseURL string) []*playerData {
	client := &http.Client{Timeout: 10 * time.Second}
	var allPlayers []*playerData

	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/api/v1/player/list?page=%d&page_size=1000", baseURL, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			slog.Warn("创建列表请求失败", "error", err)
			return allPlayers
		}

		resp, err := client.Do(req)
		if err != nil {
			slog.Warn("请求玩家列表失败", "page", page, "error", err)
			return allPlayers // 接口可能不存在，直接返回
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			slog.Warn("玩家列表接口返回非 200", "page", page, "status", resp.StatusCode)
			return allPlayers
		}

		var listResp playerListResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			resp.Body.Close()
			slog.Warn("解析玩家列表响应失败", "page", page, "error", err)
			return allPlayers
		}
		resp.Body.Close()

		if listResp.Data == nil || len(listResp.Data.Players) == 0 {
			break // 无更多数据
		}

		allPlayers = append(allPlayers, listResp.Data.Players...)

		// 判断是否已拉取全部
		if listResp.Data.Total > 0 && len(allPlayers) >= listResp.Data.Total {
			break
		}
		// 如果返回数量不足 page_size，说明是最后一页
		if len(listResp.Data.Players) < 1000 {
			break
		}
	}

	return allPlayers
}

// fetchPlayersByRange 逐个查询玩家（回退方案）。
// 遍历指定 ID 范围，调用 GET /api/v1/player/{id} 获取每个玩家。
func fetchPlayersByRange(ctx context.Context, baseURL string, startID, endID int) []*playerData {
	client := &http.Client{Timeout: 5 * time.Second}
	var players []*playerData

	for id := startID; id <= endID; id++ {
		select {
		case <-ctx.Done():
			return players
		default:
		}

		url := fmt.Sprintf("%s/api/v1/player/%d", baseURL, id)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		var playerResp playerResponse
		if err := json.NewDecoder(resp.Body).Decode(&playerResp); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if playerResp.Data != nil && playerResp.Data.Player != nil {
			players = append(players, playerResp.Data.Player)
		}
	}

	return players
}

// batchUpdateRankings 将玩家数据批量写入各个排行榜。
func batchUpdateRankings(ctx context.Context, svc *service.RankingService, players []*playerData) {
	if len(players) == 0 {
		return
	}

	// 按排行榜类型分组构建条目
	realmEntries := make([]*model.RankingEntry, 0, len(players))
	combatEntries := make([]*model.RankingEntry, 0, len(players))
	wealthEntries := make([]*model.RankingEntry, 0, len(players))

	for _, p := range players {
		if p == nil {
			continue
		}

		playerID := uint64(p.ID)
		rName := realmName(p.Realm)

		// 境界榜评分：境界 ID * 10000 + 等级
		realmScore := model.ScoreForRealm(uint32(p.Realm), uint32(p.Level))
		realmEntries = append(realmEntries, &model.RankingEntry{
			PlayerID:  playerID,
			Nickname:  p.Name,
			RealmName: rName,
			Score:     realmScore,
		})

		// 战力榜评分：攻击 + 防御 + 修为
		combatScore := float64(p.Attack + p.Defense + p.SpiritPower)
		combatEntries = append(combatEntries, &model.RankingEntry{
			PlayerID: playerID,
			Nickname: p.Name,
			Score:    combatScore,
		})

		// 财富榜评分：灵石数量
		wealthEntries = append(wealthEntries, &model.RankingEntry{
			PlayerID: playerID,
			Nickname: p.Name,
			Score:    float64(p.Gold),
		})
	}

	// 批量写入 Redis
	if err := svc.BatchUpdate(ctx, model.RankingTypeRealm, realmEntries); err != nil {
		slog.Error("批量更新境界榜失败", "error", err)
	} else {
		slog.Debug("境界榜更新完成", "count", len(realmEntries))
	}

	if err := svc.BatchUpdate(ctx, model.RankingTypeCombatPower, combatEntries); err != nil {
		slog.Error("批量更新战力榜失败", "error", err)
	} else {
		slog.Debug("战力榜更新完成", "count", len(combatEntries))
	}

	if err := svc.BatchUpdate(ctx, model.RankingTypeWealth, wealthEntries); err != nil {
		slog.Error("批量更新财富榜失败", "error", err)
	} else {
		slog.Debug("财富榜更新完成", "count", len(wealthEntries))
	}
}

// realmName 根据境界 ID 返回中文名称。
func realmName(id int32) string {
	names := map[int32]string{
		1: "凡人",
		2: "练气",
		3: "筑基",
		4: "金丹",
		5: "元婴",
		6: "化神",
		7: "合体",
		8: "大乘",
		9: "渡劫",
	}
	if name, ok := names[id]; ok {
		return name
	}
	return "未知"
}

// ============================================================
// HTTP 中间件
// ============================================================

// corsMiddleware 跨域中间件。
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
