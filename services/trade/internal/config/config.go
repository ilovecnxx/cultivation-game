// Package config 提供交易服务的配置结构，支持从环境变量加载。
package config

import (
	"os"
	"strconv"
	"time"
)

// Config 交易服务配置。
type Config struct {
	// 监听地址（HTTP 端口）
	ListenAddr string

	// MySQL 数据源
	MySQLDSN string

	// Redis 地址
	RedisAddr string

	// Redis 密码
	RedisPassword string

	// Redis 数据库编号
	RedisDB int

	// 挂单默认有效期
	ListingDefaultDuration time.Duration

	// 拍卖默认持续时间
	AuctionDefaultDuration time.Duration

	// 拍卖过期检查间隔
	AuctionCheckInterval time.Duration

	// 每次购买的最少数量
	MinBuyQuantity uint32

	// 每次购买的最大数量
	MaxBuyQuantity uint32
}

// Load 从环境变量加载配置，缺失项使用默认值。
func Load() *Config {
	return &Config{
		ListenAddr:             getEnv("LISTEN_ADDR", ":"+getEnv("TRADE_PORT", "8087")),
		MySQLDSN:               getEnv("MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/cultivation?charset=utf8mb4&parseTime=True&loc=Local"),
		RedisAddr:              getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		RedisDB:                getInt("REDIS_DB", 0),
		ListingDefaultDuration: getDuration("LISTING_DEFAULT_DURATION", 72*time.Hour), // 默认挂单 3 天
		AuctionDefaultDuration: getDuration("AUCTION_DEFAULT_DURATION", 24*time.Hour), // 默认拍卖 24 小时
		AuctionCheckInterval:   getDuration("AUCTION_CHECK_INTERVAL", 30*time.Second), // 每 30 秒检查过期拍卖
		MinBuyQuantity:         uint32(getInt("MIN_BUY_QUANTITY", 1)),
		MaxBuyQuantity:         uint32(getInt("MAX_BUY_QUANTITY", 9999)),
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
