// Package config 提供认证服务的配置结构，支持从环境变量加载。
package config

import (
	"os"
	"strconv"
	"time"
)

// Config 认证服务配置。
type Config struct {
	// 监听地址（gRPC 端口）
	ListenAddr string

	// GM HTTP 监听地址
	GMListenAddr string

	// GM JWT 签名密钥
	GMJWTSecret string

	// MySQL 数据源
	MySQLDSN string

	// Redis 地址
	RedisAddr string

	// Redis 密码
	RedisPassword string

	// Redis 数据库编号
	RedisDB int

	// JWT 签名密钥（Access Token）
	JWTAccessSecret string

	// JWT 签名密钥（Refresh Token）
	JWTRefreshSecret string

	// Access Token 有效期
	JWTAccessExpire time.Duration

	// Refresh Token 有效期
	JWTRefreshExpire time.Duration

	// 会话在 Redis 中的过期时间
	SessionTTL time.Duration
}

// Load 从环境变量加载配置，缺失项使用默认值。
func Load() *Config {
	return &Config{
		ListenAddr:       getEnv("LISTEN_ADDR", ":50060"),
		GMListenAddr:     getEnv("GM_LISTEN_ADDR", ":18082"),
		GMJWTSecret:      getEnv("GM_JWT_SECRET", "gm-jwt-secret-change-in-production"),
		MySQLDSN:         getEnv("MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/cultivation?charset=utf8mb4&parseTime=True&loc=Local"),
		RedisAddr:        getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		RedisDB:          getInt("REDIS_DB", 0),
		JWTAccessSecret:  getEnv("JWT_ACCESS_SECRET", "auth-access-secret-change-in-production"),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", "auth-refresh-secret-change-in-production"),
		JWTAccessExpire:  getDuration("JWT_ACCESS_EXPIRE", 1*time.Hour),
		JWTRefreshExpire: getDuration("JWT_REFRESH_EXPIRE", 7*24*time.Hour),
		SessionTTL:       getDuration("SESSION_TTL", 24*time.Hour),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
