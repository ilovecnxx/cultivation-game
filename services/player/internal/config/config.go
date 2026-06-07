package config

import (
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig
	MySQL    MySQLConfig
	Redis    RedisConfig
	Log      LogConfig
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// MySQLConfig 数据库配置
type MySQLConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig 缓存配置
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string
	Output string // "stdout" or file path
}

// Load 从环境变量加载配置，提供默认值
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Addr:         getEnv("SERVER_ADDR", ":8080"),
			ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		MySQL: MySQLConfig{
			DSN:             getEnv("MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/cultivation?charset=utf8mb4&parseTime=True&loc=Local"),
			MaxOpenConns:    getInt("MYSQL_MAX_OPEN", 50),
			MaxIdleConns:    getInt("MYSQL_MAX_IDLE", 10),
			ConnMaxLifetime: getDuration("MYSQL_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
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
