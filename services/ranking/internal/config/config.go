// Package config 提供排行榜服务的配置结构，支持从环境变量加载。
package config

import (
	"os"
	"strconv"
	"time"
)

// Config 排行榜服务配置。
type Config struct {
	// HTTP 服务端口
	Port int

	// Redis 地址
	RedisAddr string

	// Redis 密码
	RedisPassword string

	// Redis 数据库编号
	RedisDB int

	// 玩家服务地址（用于数据预热和定时同步）
	PlayerServiceAddr string

	// 排行榜每页默认大小
	DefaultPageSize int

	// 排行榜最大每页数量
	MaxPageSize int

	// Top N 缓存刷新间隔
	CacheRefreshInterval time.Duration

	// Top N 缓存条目数
	CacheTopN int

	// 分数衰减检查间隔
	DecayCheckInterval time.Duration

	// 分数衰减：不活跃天数阈值
	DecayAfterDays int

	// 分数衰减：每日衰减比例（万分比，如 300 = 3%）
	DecayRatePerMille int

	// 异步更新缓冲区大小
	UpdateBufferSize int

	// 异步更新 Worker 数量
	UpdateWorkerCount int
}

// Load 从环境变量加载配置，缺失项使用默认值。
func Load() *Config {
	return &Config{
		Port:                 getInt("RANKING_PORT", 8088),
		RedisAddr:            getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:        getEnv("REDIS_PASSWORD", ""),
		RedisDB:              getInt("REDIS_DB", 1),
		PlayerServiceAddr:    getEnv("PLAYER_SERVICE_ADDR", "http://localhost:8081"),
		DefaultPageSize:      getInt("RANKING_DEFAULT_PAGE_SIZE", 20),
		MaxPageSize:          getInt("RANKING_MAX_PAGE_SIZE", 100),
		CacheRefreshInterval: getDuration("RANKING_CACHE_REFRESH_INTERVAL", 30*time.Second),
		CacheTopN:            getInt("RANKING_CACHE_TOP_N", 100),
		DecayCheckInterval:   getDuration("RANKING_DECAY_CHECK_INTERVAL", 10*time.Minute),
		DecayAfterDays:       getInt("RANKING_DECAY_AFTER_DAYS", 7),
		DecayRatePerMille:    getInt("RANKING_DECAY_RATE_PER_MILLE", 30),
		UpdateBufferSize:     getInt("RANKING_UPDATE_BUFFER_SIZE", 10000),
		UpdateWorkerCount:    getInt("RANKING_UPDATE_WORKER_COUNT", 4),
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
