package config

import (
	"encoding/json"
	"os"
)

// Config 战斗服务配置
type Config struct {
	Server   ServerConfig   `json:"server"`
	Game     GameConfig     `json:"game"`
	DataPath DataPathConfig `json:"data_path"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// GameConfig 游戏核心参数
type GameConfig struct {
	// 五行克制倍率
	ElementAdvantageMultiplier  float64 `json:"element_advantage_multiplier"`  // 克制时倍率，默认1.3
	ElementDisadvantageMultiplier float64 `json:"element_disadvantage_multiplier"` // 被克时倍率，默认0.7
	// 暴击相关
	CritDamageMultiplier float64 `json:"crit_damage_multiplier"` // 暴击伤害倍率，默认2.0
	// 匹配相关
	MatchmakingRange   int `json:"matchmaking_range"`    // 匹配段位范围
	MatchmakingTimeout int `json:"matchmaking_timeout"` // 匹配超时(秒)
	// 每回合最大冷却缩减
	MaxCooldownReduction int `json:"max_cooldown_reduction"` // 默认1
	// 竞技场/赛季配置
	ArenaMatchScoreRange    int `json:"arena_match_score_range"`    // 竞技场匹配积分范围, 默认100
	ArenaMatchExpandSeconds int `json:"arena_match_expand_seconds"` // 匹配扩大范围等待秒数, 默认30
	SeasonDurationDays      int `json:"season_duration_days"`       // 赛季持续天数, 默认30
}

// DataPathConfig 数据文件路径
type DataPathConfig struct {
	Monsters  string `json:"monsters"`
	Skills    string `json:"skills"`
	Instances string `json:"instances"`
	Dungeons  string `json:"dungeons"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Game: GameConfig{
			ElementAdvantageMultiplier:    1.3,
			ElementDisadvantageMultiplier: 0.7,
			CritDamageMultiplier:          2.0,
			MatchmakingRange:              3,
			MatchmakingTimeout:            60,
			MaxCooldownReduction:          1,
			ArenaMatchScoreRange:          100,
			ArenaMatchExpandSeconds:       30,
			SeasonDurationDays:            30,
		},
		DataPath: DataPathConfig{
			Monsters:  "internal/data/monsters.json",
			Skills:    "internal/data/skills.json",
			Instances: "internal/data/instances.json",
			Dungeons:  "internal/data/dungeons.json",
		},
	}
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil // 文件不存在使用默认
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
