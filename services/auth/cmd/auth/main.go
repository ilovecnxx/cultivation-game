// Command auth 修仙游戏认证服务入口。
//
// 启动流程：
//   1. 加载配置（环境变量）
//   2. 初始化结构化日志
//   3. 连接 MySQL，建表检查
//   4. 连接 Redis
//   5. 初始化 Repository、Service、Handler 三层
//   6. 启动 gRPC 服务器
//   7. 监听系统信号，优雅关闭
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cultivation-game/services/auth/api"
	"cultivation-game/services/auth/internal/config"
	"cultivation-game/services/auth/internal/handler"
	"cultivation-game/services/auth/internal/repository"
	"cultivation-game/services/auth/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
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
	slog.Info("认证服务启动中", "listen_addr", cfg.ListenAddr)

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
	slog.Info("Redis 连接成功")

	// =========================================================
	// 5. 初始化三层架构
	// =========================================================
	userRepo := repository.NewUserRepo(db, slog.Default())
	authService := service.NewAuthService(userRepo, rdb, cfg, slog.Default())
	authHandler := handler.NewAuthHandler(authService, slog.Default())

	// ---- GM 管理后台 ----
	gmRepo := repository.NewGMRepo(db, slog.Default())
	gmService := service.NewGMService(gmRepo, userRepo, cfg, slog.Default())
	gmService.SetJWTSecret(cfg.GMJWTSecret)
	gmHandler := handler.NewGMHandler(gmService, slog.Default())

	// 创建默认管理员账号
	if err := gmService.SeedDefaultAdmin(context.Background()); err != nil {
		slog.Error("创建默认 GM 管理员失败", "error", err)
		os.Exit(1)
	}
	slog.Info("GM 管理员检查完毕")

	// =========================================================
	// 6. 启动 gRPC 服务器
	// =========================================================
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  30 * time.Second,
			Timeout:               5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.MaxRecvMsgSize(64 * 1024),
		grpc.MaxSendMsgSize(64 * 1024),
	)

	// 注册认证服务
	api.RegisterAuthServiceServer(grpcServer, authHandler)

	// 注册 gRPC 反射服务（便于调试和 grpcurl 使用）
	reflection.Register(grpcServer)

	// 启动监听
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		slog.Error("监听端口失败", "error", err, "addr", cfg.ListenAddr)
		os.Exit(1)
	}

	go func() {
		slog.Info("gRPC 服务器已启动", "addr", cfg.ListenAddr)
		if err := grpcServer.Serve(listener); err != nil {
			slog.Error("gRPC 服务器异常退出", "error", err)
			os.Exit(1)
		}
	}()

	// =========================================================
	// 7. 启动 GM HTTP 管理服务器
	// =========================================================
	gin.SetMode(gin.ReleaseMode)
	gmRouter := gin.New()
	gmRouter.Use(gin.Recovery())

	// 健康检查
	gmRouter.GET("/health", func(c *gin.Context) {
		dbStatus := "ok"
		if err := db.Ping(); err != nil {
			dbStatus = "degraded: " + err.Error()
		}
		redisStatus := "ok"
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			redisStatus = "degraded: " + err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "auth-gm",
			"mysql":   dbStatus,
			"redis":   redisStatus,
		})
	})

	// GM API v1 路由
	gmV1 := gmRouter.Group("/api/v1/gm")
	{
		// GM 登录（无需认证）
		gmV1.POST("/login", gmHandler.Login)

		// 以下路由需要 GM 认证
		authed := gmV1.Group("")
		authed.Use(gmService.GMAuthMiddleware())
		{
			// 玩家管理
			authed.GET("/players", gmHandler.GetPlayerList)
			authed.GET("/players/:id", gmHandler.GetPlayerDetail)

			// 写操作需要更高权限
			write := authed.Group("")
			write.Use(gmService.GMPermissionMiddleware())
			{
				write.PUT("/players/:id/attribute", gmHandler.EditPlayerAttribute)
				write.POST("/players/:id/ban", gmHandler.BanPlayer)
				write.DELETE("/players/:id/ban", gmHandler.UnbanPlayer)
				write.POST("/announcements", gmHandler.SendAnnouncement)
				write.POST("/players/:id/items", gmHandler.SendItem)
			}

			// 统计和日志（所有 GM 角色可查看）
			authed.GET("/stats", gmHandler.GetServerStats)
			authed.GET("/logs", gmHandler.GetOperationLogs)
		}
	}

	gmHTTPServer := &http.Server{
		Addr:         cfg.GMListenAddr,
		Handler:      gmRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("GM HTTP 管理服务器启动", "addr", cfg.GMListenAddr)
		if err := gmHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("GM HTTP 服务器异常退出", "error", err)
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

	// 优雅关闭所有服务器（最大等待 10 秒）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 关闭 GM HTTP 服务器
	if err := gmHTTPServer.Shutdown(shutdownCtx); err != nil {
		slog.Warn("GM HTTP 服务器关闭异常", "error", err)
	} else {
		slog.Info("GM HTTP 服务器已关闭")
	}

	// 关闭 gRPC 服务器
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("gRPC 服务器已优雅关闭")
	case <-shutdownCtx.Done():
		slog.Warn("gRPC 服务器关闭超时，强制停止")
		grpcServer.Stop()
	}

	// 关闭数据库连接
	db.Close()
	rdb.Close()

	slog.Info("认证服务已停止")
}

// initSchema 初始化数据库表结构。
// 如果表不存在则自动创建，确保服务启动时数据库结构正确。
func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '用户唯一 ID',
		username VARCHAR(64) NOT NULL COMMENT '用户名',
		password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希（bcrypt）',
		player_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关联的玩家角色 ID',
		email VARCHAR(128) NOT NULL DEFAULT '' COMMENT '电子邮箱',
		status TINYINT NOT NULL DEFAULT 0 COMMENT '账号状态：0=正常 1=封禁 2=冻结 3=已删除',
		last_login_at DATETIME NULL COMMENT '最后登录时间',
		last_ip VARCHAR(64) NOT NULL DEFAULT '' COMMENT '最后登录 IP',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
		UNIQUE KEY uk_username (username),
		INDEX idx_player_id (player_id),
		INDEX idx_status (status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户账号表';
	`

	_, err := db.ExecContext(context.Background(), schema)
	if err != nil {
		return err
	}

	// GM 管理后台表（逐条执行，避免多语句兼容性问题）
	gmTables := []string{
		`CREATE TABLE IF NOT EXISTS gm_admins (
			id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '管理员唯一ID',
			username VARCHAR(64) NOT NULL COMMENT '管理员用户名（唯一）',
			password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希（bcrypt）',
			role TINYINT NOT NULL DEFAULT 3 COMMENT '角色：1=超级管理员 2=运营 3=观察者',
			status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1=启用 0=禁用',
			last_login_at TIMESTAMP NULL DEFAULT NULL COMMENT '最后登录时间',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
			UNIQUE KEY uk_username (username),
			INDEX idx_role (role),
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 管理员账号表'`,

		`CREATE TABLE IF NOT EXISTS gm_operation_logs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '日志ID',
			admin_id BIGINT NOT NULL COMMENT '操作管理员ID',
			action_type VARCHAR(64) NOT NULL COMMENT '操作类型（如 ban_player, send_item, edit_attribute）',
			target_type VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作目标类型（如 player, announcement, system）',
			target_id BIGINT NOT NULL DEFAULT 0 COMMENT '操作目标ID',
			detail JSON DEFAULT NULL COMMENT '操作详情（JSON格式）',
			ip_address VARCHAR(45) NOT NULL DEFAULT '' COMMENT '操作IP地址',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '操作时间',
			INDEX idx_admin_id (admin_id),
			INDEX idx_action_type (action_type),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 操作日志表'`,

		`CREATE TABLE IF NOT EXISTS gm_announcements (
			id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '公告ID',
			admin_id BIGINT NOT NULL COMMENT '发布管理员ID',
			title VARCHAR(128) NOT NULL COMMENT '公告标题',
			content TEXT NOT NULL COMMENT '公告内容',
			type TINYINT NOT NULL DEFAULT 1 COMMENT '公告类型：1=系统公告 2=世界公告 3=个人消息',
			target_player_id BIGINT NULL DEFAULT NULL COMMENT '目标玩家ID（个人消息时使用）',
			sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发送时间',
			expire_at TIMESTAMP NULL DEFAULT NULL COMMENT '过期时间（NULL表示永不过期）',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
			INDEX idx_admin_id (admin_id),
			INDEX idx_type (type),
			INDEX idx_sent_at (sent_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 公告表'`,

		`CREATE TABLE IF NOT EXISTS gm_bans (
			id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '封禁记录ID',
			player_id BIGINT NOT NULL COMMENT '被封禁玩家ID',
			admin_id BIGINT NOT NULL COMMENT '操作管理员ID',
			reason TEXT NOT NULL COMMENT '封禁原因',
			ban_type TINYINT NOT NULL DEFAULT 1 COMMENT '封禁类型：1=禁言 2=临时封号 3=永久封号',
			starts_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '封禁开始时间',
			ends_at TIMESTAMP NULL DEFAULT NULL COMMENT '封禁结束时间（NULL表示永久）',
			status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1=生效中 0=已解封',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
			INDEX idx_player_id (player_id),
			INDEX idx_admin_id (admin_id),
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 封禁记录表'`,
	}

	for _, ddl := range gmTables {
		if _, err := db.ExecContext(context.Background(), ddl); err != nil {
			return fmt.Errorf("初始化 GM 表失败: %w", err)
		}
	}
	return nil
}
