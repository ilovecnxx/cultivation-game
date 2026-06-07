// Package config 提供世界服务的配置管理
package config

import (
	"encoding/json"
	"os"
	"time"
)

// Config 世界服务配置
type Config struct {
	Server   ServerConfig   `json:"server"`
	MongoDB  MongoDBConfig  `json:"mongodb"`
	Redis    RedisConfig    `json:"redis"`
	Services ServicesConfig `json:"services"`
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

// MongoDBConfig 数据库配置
type MongoDBConfig struct {
	URI      string `json:"uri"`
	Database string `json:"database"`
}

// RedisConfig 缓存配置
type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// ServicesConfig 依赖的其他微服务
type ServicesConfig struct {
	UserServiceAddr   string `json:"user_service_addr"`
	SocialServiceAddr string `json:"social_service_addr"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	port := 8086
	if p := os.Getenv("WORLD_PORT"); p != "" {
		if parsed, err := parseInt(p); err == nil {
			port = parsed
		}
	}
	return &Config{
		Server: ServerConfig{
			Port:         port,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		MongoDB: MongoDBConfig{
			URI:      "mongodb://localhost:27017",
			Database: "cultivation_world",
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
		Services: ServicesConfig{
			UserServiceAddr:   "http://localhost:8081",
			SocialServiceAddr: "http://localhost:8082",
		},
	}
}

// parseInt 解析字符串为整数
func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
