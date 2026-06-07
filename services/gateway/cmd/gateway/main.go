// Command gateway 修仙游戏网关服务入口。
//
// 启动流程：
//   1. 加载配置
//   2. 初始化日志、JWT、限流器、路由器、连接池、gRPC 客户端
//   3. 注册 HTTP 路由（WebSocket 升级、Token 刷新、健康检查）
//   4. 启动 HTTP 服务器
//   5. 监听系统信号，优雅关闭
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cultivation-game/services/gateway/internal/analytics"
	"cultivation-game/services/gateway/internal/anticheat"
	"cultivation-game/services/gateway/internal/auth"
	"cultivation-game/services/gateway/internal/config"
	"cultivation-game/services/gateway/internal/hub"
	"cultivation-game/services/gateway/internal/ratelimit"
	"cultivation-game/services/gateway/internal/router"
	"cultivation-game/services/gateway/internal/server"
	"cultivation-game/services/gateway/internal/session"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	slog.Info("gateway starting", "config", cfg)

	// =========================================================
	// 3. 初始化 JWT 管理器
	// =========================================================
	jwtManager := auth.NewJWTManager(
		cfg.JWTAccessSecret,
		cfg.JWTRefreshSecret,
		cfg.JWTAccessExpire,
		cfg.JWTRefreshExpire,
		cfg.JWTIssuer,
	)

	// =========================================================
	// 4. 初始化限流器
	// =========================================================
	rateLimiter := ratelimit.NewRateLimiter(cfg.RateLimitRate, cfg.RateLimitCapacity)

	// =========================================================
	// 5. 初始化消息路由器（NATS）
	// =========================================================
	msgRouter, err := router.NewRouter(cfg.NATSURL, cfg.NATSConnectTimeout)
	if err != nil {
		slog.Error("failed to create router", "error", err)
		os.Exit(1)
	}
	defer msgRouter.Close()

	// =========================================================
	// 6. 初始化 Redis 会话管理器
	// =========================================================
	sessionMgr, err := session.NewManager(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		slog.Warn("Redis 连接失败，在线会话管理将降级为本地模式", "error", err)
		// 非致命：允许无 Redis 降级运行
	}
	if sessionMgr != nil {
		defer sessionMgr.Close()
	}

	// =========================================================
	// 7. 初始化反作弊管理器
	// =========================================================
	anticheatOpts := anticheat.Options{
		BanDuration: time.Hour,
	}
	if sessionMgr != nil {
		anticheatOpts.RedisClient = sessionMgr.Client()
	}

	// 尝试连接 MongoDB 用于反作弊记录持久化
	if cfg.MongoURI != "" {
		mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
		mongoClient, mongoErr := mongo.Connect(mongoCtx, options.Client().ApplyURI(cfg.MongoURI))
		mongoCancel()
		if mongoErr == nil {
			anticheatOpts.MongoCollection = mongoClient.Database(cfg.MongoDatabase).Collection("anticheat_reports")
			slog.Info("反作弊 MongoDB 连接成功", "collection", "anticheat_reports")
			// MongoDB 连接将在进程退出时自动清理
		} else {
			slog.Warn("反作弊 MongoDB 连接失败，使用纯内存模式", "error", mongoErr)
		}
	}

	acManager := anticheat.NewManager(anticheatOpts)

	// =========================================================
	// 8. 初始化连接池
	// =========================================================
	hub := hub.NewHub(cfg, msgRouter, rateLimiter, sessionMgr, acManager)
	defer hub.Close()

	// =========================================================
	// 9. 初始化 gRPC 客户端（连接到后端服务）
	// =========================================================
	grpcClient, err := server.NewGRPCClient(cfg.BackendServiceAddr, cfg.GRPCDialTimeout)
	if err != nil {
		slog.Warn("failed to connect to backend service (non-fatal)",
			"error", err,
			"addr", cfg.BackendServiceAddr,
		)
		// 网关不依赖后端 gRPC 启动
	} else {
		defer grpcClient.Close()
	}

	// =========================================================
	// 10. 初始化分析引擎
	// =========================================================
	analyticsEngine := analytics.NewAnalytics(analytics.Options{
		MongoURI:        cfg.MongoURI,
		MongoDatabase:   cfg.MongoDatabase,
		MongoCollection: "analytics_events",
		BufferCapacity:  cfg.AnalyticsBufferCapacity,
		FlushInterval:   cfg.AnalyticsFlushInterval,
		FlushBatchSize:  cfg.AnalyticsBufferCapacity,
		Mode:            analytics.FlushModeBatch,
	})
	defer analyticsEngine.Close()

	// =========================================================
	// 11. 注册 HTTP 路由
	// =========================================================
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(RequestLogger())
	r.Use(CORSMiddleware())

	r.GET("/ws", wsHandler(hub, jwtManager, cfg))
	r.POST("/auth/refresh", refreshTokenHandler(jwtManager))
	r.POST("/auth/login", loginHandler(jwtManager, grpcClient))
	r.POST("/auth/register", registerHandler(jwtManager, grpcClient, hub))
	r.GET("/health", healthHandler(hub, msgRouter))

	// 注册分析引擎路由
	{
		analyticsHandler := analytics.NewHandler(analyticsEngine, nil)
		analyticsHandler.RegisterRoutes(r)
	}

	// =========================================================
	// 12. 启动 HTTP 服务器
	// =========================================================
	addr := ":" + os.Getenv("SERVER_PORT")
	if addr == ":" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("HTTP server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// =========================================================
	// 13. 等待信号，优雅关闭
	// =========================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig)

	// 先停止接受新连接
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	// 关闭连接池（会断开所有 WebSocket 连接）
	hub.Close()

	// 关闭路由器
	msgRouter.Close()

	// 关闭 gRPC 连接
	if grpcClient != nil {
		grpcClient.Close()
	}

	slog.Info("gateway stopped")
}

// =========================================================
// HTTP Handlers
// =========================================================

// upgrader WebSocket 升级器。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// 允许所有来源（生产环境应限制）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// RequestLogger 请求日志中间件，使用 slog 记录每个 HTTP 请求。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		slog.Info("request",
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"status", status,
			"latency", latency,
			"remote", c.Request.RemoteAddr,
		)
	}
}

// CORSMiddleware CORS 中间件，允许所有来源（生产环境应限制）。
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// wsHandler 处理 WebSocket 升级请求。
// 路径: GET /ws?token=<access_token>
func wsHandler(hub *hub.Hub, jwtManager *auth.JWTManager, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 验证 Token
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing token"})
			return
		}

		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			slog.Warn("ws auth failed", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid token"})
			return
		}

		// 升级为 WebSocket 连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "error", err)
			return
		}

		// 创建连接对象
		playerID := claims.PlayerID
		account := claims.Account
		bucket := hub.RateLimiter().GetBucket("conn_" + c.Request.RemoteAddr)
		connection := hub.NewConnection(conn, bucket)
		connection.SetPlayer(playerID, account)

		// 注册到连接池
		hub.Register(connection)

		slog.Info("new websocket connection",
			"player_id", playerID,
			"account", account,
			"conn_id", connection.ID,
			"remote", c.Request.RemoteAddr,
		)

		// 启动读写协程
		go connection.ReadPump()
		go connection.WritePump()
	}
}

// refreshTokenHandler 刷新 Access Token。
// 路径: POST /auth/refresh
// Body: {"refresh_token": "..."}
// 响应: {"access_token": "...", "refresh_token": "..."}
func refreshTokenHandler(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid body"})
			return
		}

		if req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "refresh_token is required"})
			return
		}

		// 验证 Refresh Token 并签发新的 Access Token
		claims, err := jwtManager.ValidateRefreshToken(req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid refresh token"})
			return
		}

		accessToken, err := jwtManager.GenerateAccessToken(claims.PlayerID, claims.Account)
		if err != nil {
			slog.Error("generate access token error", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "internal error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": req.RefreshToken,
		})
	}
}

// loginHandler 用户登录，调用 Auth 服务的 gRPC API 验证凭证并签发 JWT Token。
// 路径: POST /auth/login
// Body: {"account": "...", "password": "..."}
// 响应: {"access_token": "...", "refresh_token": "...", "player_id": ..., "user_id": ...}
func loginHandler(jwtManager *auth.JWTManager, grpcClient *server.GRPCClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Account  string `json:"account"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid body"})
			return
		}

		if req.Account == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "account and password required"})
			return
		}

		// 检查 gRPC 连接是否可用
		if grpcClient == nil {
			slog.Error("auth service gRPC client not available")
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "auth service unavailable"})
			return
		}

		// 调用 Auth 服务的 Login RPC 验证账号密码
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		loginResp, err := grpcClient.AuthLogin(ctx, req.Account, req.Password)
		if err != nil {
			slog.Warn("auth login failed", "account", req.Account, "error", err)

			// 将 gRPC 状态码映射为 HTTP 状态码
			if st, ok := status.FromError(err); ok {
				switch st.Code() {
				case codes.NotFound, codes.Unauthenticated:
					c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "账号或密码错误"})
					return
				case codes.PermissionDenied:
					c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "账号已被封禁"})
					return
				}
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "认证服务内部错误"})
			return
		}

		// 使用 Auth 服务返回的身份信息签发网关自身的 JWT Token 对
		accessToken, refreshToken, err := jwtManager.GenerateTokenPair(loginResp.PlayerId, req.Account)
		if err != nil {
			slog.Error("generate token pair error", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "internal error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"player_id":     loginResp.PlayerId,
			"user_id":       loginResp.UserId,
			"account":       req.Account,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

// registerHandler 用户注册，调用 Auth 服务的 gRPC API 创建账号并签发 JWT Token。
// 路径: POST /auth/register
// Body: {"account": "...", "password": "...", "nickname": "..."}
// 响应: {"access_token": "...", "refresh_token": "...", "player_id": ..., "user_id": ...}
func registerHandler(jwtManager *auth.JWTManager, grpcClient *server.GRPCClient, hub *hub.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Account  string `json:"account"`
			Password string `json:"password"`
			Nickname string `json:"nickname"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid body"})
			return
		}

		if req.Account == "" || req.Password == "" || req.Nickname == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "account, password and nickname required"})
			return
		}

		if grpcClient == nil {
			slog.Error("auth service gRPC client not available")
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "auth service unavailable"})
			return
		}

		// 调用 Auth 服务的 Register RPC 创建账号
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		registerResp, err := grpcClient.AuthRegister(ctx, req.Account, req.Password, req.Nickname)
		if err != nil {
			slog.Warn("auth register failed", "account", req.Account, "error", err)

			if st, ok := status.FromError(err); ok {
				switch st.Code() {
				case codes.AlreadyExists:
					c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": "用户名已存在"})
					return
				case codes.InvalidArgument:
					c.JSON(http.StatusBadRequest, gin.H{
						"code": 400,
						"msg":  st.Message(),
					})
					return
				}
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "认证服务内部错误"})
			return
		}

		// 使用 Auth 服务返回的身份信息签发网关自身的 JWT Token 对
		accessToken, refreshToken, err := jwtManager.GenerateTokenPair(registerResp.PlayerId, req.Account)
		if err != nil {
			slog.Error("generate token pair error", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "internal error"})
			return
		}

		// 递增注册用户计数（Redis）
		if hub.SessionMgr() != nil {
			hub.SessionMgr().Client().Incr(c.Request.Context(), "gateway:registered_users")
		}

		c.JSON(http.StatusOK, gin.H{
			"player_id":     registerResp.PlayerId,
			"user_id":       registerResp.UserId,
			"account":       req.Account,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

// healthHandler 健康检查端点。
// 路径: GET /health
func healthHandler(hub *hub.Hub, msgRouter *router.Router) gin.HandlerFunc {
	return func(c *gin.Context) {
		redisStatus := "disconnected"
		if hub.SessionMgr() != nil {
			if err := hub.SessionMgr().Ping(c.Request.Context()); err != nil {
				redisStatus = "degraded: " + err.Error()
			} else {
				redisStatus = "connected"
			}
		}

		online := hub.OnlineCount()
		// 如果 Redis 可用，也从 Redis 获取在线数
		if hub.SessionMgr() != nil {
			if redisCount, err := hub.SessionMgr().OnlineCount(c.Request.Context()); err == nil {
				online = int(redisCount)
			}
		}

		// 收集反作弊统计信息
		acStats := map[string]interface{}{"enabled": false}
		if hub.AntiCheat() != nil {
			acStats = hub.AntiCheat().Stats()
		}

		// 注册用户数（从 Redis 读取，auth 服务写入）
		registered := 0
		if hub.SessionMgr() != nil {
			if n, err := hub.SessionMgr().Client().Get(c.Request.Context(), "gateway:registered_users").Int(); err == nil {
				registered = n
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":         "ok",
			"time":           time.Now().Unix(),
			"online":         online,
			"registered":     registered,
			"conns":          hub.TotalConnections(),
			"nats_connected": msgRouter.IsConnected(),
			"redis":          redisStatus,
			"anticheat":      acStats,
		})
	}
}
