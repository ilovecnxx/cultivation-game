// Package config 提供社交服务的配置管理
package config

import (
	"encoding/json"
	"os"
	"time"
)

// Config 社交服务配置
type Config struct {
	Server   ServerConfig   `json:"server"`
	MongoDB  MongoDBConfig  `json:"mongodb"`
	Redis    RedisConfig    `json:"redis"`
	Services ServicesConfig `json:"services"`
	SectWar  SectWarConfig  `json:"sect_war"`
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
	UserServiceAddr    string `json:"user_service_addr"`
	WorldServiceAddr   string `json:"world_service_addr"`
	PlayerServiceAddr  string `json:"player_service_addr"`
}

// SectWarConfig 宗门战配置
type SectWarConfig struct {
	SeasonDays     int   `json:"season_days"`      // 赛季天数(默认30)
	MatchWeekdays  []int `json:"match_weekdays"`   // 比赛日星期(周三=3,周六=6)
	MatchHour      int   `json:"match_hour"`        // 比赛小时(20:00)
	MinSectLevel   int   `json:"min_sect_level"`    // 参赛最低宗门等级(默认3)
	MinOnlineMembers int `json:"min_online_members"` // 最低在线成员数(默认10)
	PlayersPerSect int   `json:"players_per_sect"`   // 每方出战人数(默认3)
	WinScore       int   `json:"win_score"`          // 胜利积分(默认3)
	DrawScore      int   `json:"draw_score"`         // 平局积分(默认1)
	LoseScore      int   `json:"lose_score"`         // 失败积分(默认0)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8082,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		MongoDB: MongoDBConfig{
			URI:      "mongodb://localhost:27017",
			Database: "cultivation_social",
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
		Services: ServicesConfig{
			UserServiceAddr:    "http://localhost:8081",
			WorldServiceAddr:   "http://localhost:8083",
			PlayerServiceAddr:  "http://localhost:8080",
		},
		SectWar: SectWarConfig{
			SeasonDays:       30,
			MatchWeekdays:    []int{3, 6},
			MatchHour:        20,
			MinSectLevel:     3,
			MinOnlineMembers: 10,
			PlayersPerSect:   3,
			WinScore:         3,
			DrawScore:        1,
			LoseScore:        0,
		},
	}
}

// LoadConfig 从文件加载配置，若文件不存在则返回默认配置。
// 支持环境变量覆盖：SOCIAL_PORT, REDIS_ADDR, REDIS_PASSWORD
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// 环境变量覆盖
	if v := os.Getenv("SOCIAL_PORT"); v != "" {
		if p, err := parseInt(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}

	return cfg, nil
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
