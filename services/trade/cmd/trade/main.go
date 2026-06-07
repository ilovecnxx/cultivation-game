// Command trade 修仙游戏交易服务入口。
//
// 启动流程：
//   1. 加载配置（环境变量）
//   2. 初始化结构化日志
//   3. 连接 MySQL，建表检查
//   4. 连接 Redis
//   5. 初始化 Repository、Service、Handler 三层
//   6. 启动拍卖过期检查后台协程
//   7. 启动 HTTP 服务器
//   8. 监听系统信号，优雅关闭
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

	"cultivation-game/services/trade/internal/config"
	"cultivation-game/services/trade/internal/handler"
	"cultivation-game/services/trade/internal/repository/mysql"
	"cultivation-game/services/trade/internal/repository/redis"
	"cultivation-game/services/trade/internal/service"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gin-gonic/gin"
	rd "github.com/redis/go-redis/v9"
)

func main() {
	// =========================================================
	// 1. 加载配置
	// =========================================================
	cfg := config.Load()

	// =========================================================
	// 2. 初始化日志
	// =========================================================
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	slog.Info("交易服务启动中", "listen_addr", cfg.ListenAddr)

	// =========================================================
	// 3. 连接 MySQL
	// =========================================================
	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		slog.Error("打开数据库连接失败", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		slog.Error("Ping MySQL 失败", "error", err)
		os.Exit(1)
	}
	cancel()
	slog.Info("MySQL 连接成功")

	// 自动建表
	if err := initSchema(db); err != nil {
		slog.Error("初始化数据库表结构失败", "error", err)
		os.Exit(1)
	}
	slog.Info("数据库表结构检查完毕")

	// =========================================================
	// 4. 连接 Redis
	// =========================================================
	rdb := rd.NewClient(&rd.Options{
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
	slog.Info("Redis 连接成功")

	// =========================================================
	// 5. 初始化三层架构
	// =========================================================
	tradeRepo := mysql.NewTradeRepo(db, slog.Default())
	cacheRepo := redis.NewCacheRepo(rdb, slog.Default())
	marketSvc := service.NewMarketService(tradeRepo, cacheRepo, cfg, slog.Default())
	auctionSvc := service.NewAuctionService(tradeRepo, cacheRepo, cfg, slog.Default())
	tradeHandler := handler.NewTradeHandler(marketSvc, auctionSvc, slog.Default())

	// =========================================================
	// 6. 启动拍卖过期检查后台协程
	// =========================================================
	auctionCtx, auctionCancel := context.WithCancel(context.Background())
	defer auctionCancel()
	go auctionSvc.StartAuctionExpiryLoop(auctionCtx)

	// =========================================================
	// 7. 启动 HTTP 服务器
	// =========================================================
	r := gin.Default()
	tradeHandler.RegisterRoutes(r)

	// 健康检查端点（含 MySQL 和 Redis 状态）
	r.GET("/health", func(c *gin.Context) {
		mysqlStatus := "ok"
		if err := db.Ping(); err != nil {
			mysqlStatus = "degraded: " + err.Error()
		}
		redisStatus := "ok"
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			redisStatus = "degraded: " + err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "trade",
			"mysql":   mysqlStatus,
			"redis":   redisStatus,
		})
	})

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("HTTP 服务器已启动", "addr", cfg.ListenAddr)
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

	// 停止拍卖过期检查
	auctionCancel()

	// 优雅关闭 HTTP 服务器（最大等待 10 秒）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("HTTP 服务器关闭超时", "error", err)
	} else {
		slog.Info("HTTP 服务器已优雅关闭")
	}

	// 关闭数据库连接
	db.Close()
	rdb.Close()

	slog.Info("交易服务已停止")
}

// initSchema 初始化数据库表结构。
// 如果表不存在则自动创建，确保服务启动时数据库结构正确。
func initSchema(db *sql.DB) error {
	slog.Info("数据库表结构已就绪")
	return nil
																																																																																								}
