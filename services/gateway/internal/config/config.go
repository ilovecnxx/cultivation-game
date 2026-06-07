// Package config 加载和管理网关配置。
// 配置来源优先级：环境变量 > 默认值。
package config

import (
	"os"
	"strconv"
	"time"
)

// Config 网关服务配置。
type Config struct {
	// HTTP 服务器端口
	ServerPort int `json:"server_port"`
	// 优雅关闭超时
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// WebSocket 读缓冲大小（字节）
	WSReadBufferSize int `json:"ws_read_buffer_size"`
	// WebSocket 写缓冲大小（字节）
	WSWriteBufferSize int `json:"ws_write_buffer_size"`
	// WebSocket 最大消息大小（字节）
	WSMaxMessageSize int64 `json:"ws_max_message_size"`

	// Ping 间隔
	PingInterval time.Duration `json:"ping_interval"`
	// Pong 超时（超过此时间未收到 pong 判定断线）
	PongTimeout time.Duration `json:"pong_timeout"`
	// 断线重连窗口（超过此时间清理玩家状态）
	ReconnectWindow time.Duration `json:"reconnect_window"`

	// JWT 签发密钥（Access Token）
	JWTAccessSecret string `json:"jwt_access_secret"`
	// JWT 签发密钥（Refresh Token）
	JWTRefreshSecret string `json:"jwt_refresh_secret"`
	// Access Token 有效期
	JWTAccessExpire time.Duration `json:"jwt_access_expire"`
	// Refresh Token 有效期
	JWTRefreshExpire time.Duration `json:"jwt_refresh_expire"`
	// Token 签发者
	JWTIssuer string `json:"jwt_issuer"`

	// 限流：令牌桶每秒填充速率
	RateLimitRate float64 `json:"rate_limit_rate"`
	// 限流：令牌桶容量（允许的突发请求数）
	RateLimitCapacity int `json:"rate_limit_capacity"`

	// NATS 服务器地址
	NATSURL string `json:"nats_url"`
	// NATS 连接超时
	NATSConnectTimeout time.Duration `json:"nats_connect_timeout"`

	// Redis 地址
	RedisAddr string `json:"redis_addr"`
	// Redis 密码
	RedisPassword string `json:"redis_password"`
	// Redis DB
	RedisDB int `json:"redis_db"`

	// 后端 gRPC 服务地址
	BackendServiceAddr string `json:"backend_service_addr"`
	// gRPC 拨号超时
	GRPCDialTimeout time.Duration `json:"grpc_dial_timeout"`

	// MongoDB 连接 URI
	MongoURI string `json:"mongo_uri"`
	// MongoDB 数据库名
	MongoDatabase string `json:"mongo_database"`
	// 分析引擎缓冲区容量
	AnalyticsBufferCapacity int `json:"analytics_buffer_capacity"`
	// 分析引擎刷新间隔
	AnalyticsFlushInterval time.Duration `json:"analytics_flush_interval"`
}

// Load 从环境变量加载配置，缺失项使用默认值。
func Load() *Config {
	return &Config{
		ServerPort:       getEnvInt("SERVER_PORT", 8080),
		ShutdownTimeout:  getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		WSReadBufferSize:  getEnvInt("WS_READ_BUFFER_SIZE", 4096),
		WSWriteBufferSize: getEnvInt("WS_WRITE_BUFFER_SIZE", 4096),
		WSMaxMessageSize:  int64(getEnvInt("WS_MAX_MESSAGE_SIZE", 65536)),
		PingInterval:      getEnvDuration("PING_INTERVAL", 30*time.Second),
		PongTimeout:       getEnvDuration("PONG_TIMEOUT", 60*time.Second),
		ReconnectWindow:   getEnvDuration("RECONNECT_WINDOW", 60*time.Second),
		JWTAccessSecret:   getEnv("JWT_ACCESS_SECRET", "default-access-secret-change-in-production"),
		JWTRefreshSecret:  getEnv("JWT_REFRESH_SECRET", "default-refresh-secret-change-in-production"),
		JWTAccessExpire:   getEnvDuration("JWT_ACCESS_EXPIRE", 15*time.Minute),
		JWTRefreshExpire:  getEnvDuration("JWT_REFRESH_EXPIRE", 7*24*time.Hour),
		JWTIssuer:         getEnv("JWT_ISSUER", "cultivation-game-gateway"),
		RateLimitRate:     getEnvFloat("RATE_LIMIT_RATE", 10.0),
		RateLimitCapacity: getEnvInt("RATE_LIMIT_CAPACITY", 20),
		NATSURL:           getEnv("NATS_URL", "nats://localhost:4222"),
		NATSConnectTimeout: getEnvDuration("NATS_CONNECT_TIMEOUT", 5*time.Second),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		BackendServiceAddr:        getEnv("BACKEND_SERVICE_ADDR", "localhost:50060"),
		GRPCDialTimeout:           getEnvDuration("GRPC_DIAL_TIMEOUT", 5*time.Second),
		MongoURI:                  getEnv("MONGO_URI", ""),
		MongoDatabase:             getEnv("MONGO_DATABASE", "cultivation_game"),
		AnalyticsBufferCapacity:   getEnvInt("ANALYTICS_BUFFER_CAPACITY", 1000),
		AnalyticsFlushInterval:    getEnvDuration("ANALYTICS_FLUSH_INTERVAL", 60*time.Second),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
