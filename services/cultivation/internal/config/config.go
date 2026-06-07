// Package config 管理游戏配置的加载与热重载
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"

	"cultivation-game/services/cultivation/internal/model"
)

// GameConfig 游戏总配置
type GameConfig struct {
	mu           sync.RWMutex
	Realms       []model.Realm                     `json:"realms"`
	Techniques   []model.Technique                 `json:"techniques"`
	Breakthrough BreakthroughConfig                `json:"breakthrough"`
}

// ServerConfig 服务器基础设施配置（环境变量加载）
type ServerConfig struct {
	MySQL MySQLConfig `json:"mysql"`
	Redis RedisConfig `json:"redis"`
}

// MySQLConfig MySQL 连接配置
type MySQLConfig struct {
	DSN string `json:"dsn"` // 连接字符串，如 user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true
}

// RedisConfig Redis 连接配置
type RedisConfig struct {
	Addr     string `json:"addr"`     // 地址，如 localhost:6379
	Password string `json:"password"` // 密码
	DB       int    `json:"db"`       // 数据库编号
}

// LoadServerConfig 从环境变量加载服务器配置
func LoadServerConfig() *ServerConfig {
	return &ServerConfig{
		MySQL: MySQLConfig{
			DSN: getEnv("CULTIVATION_MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/cultivation_game?charset=utf8mb4&parseTime=true"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("CULTIVATION_REDIS_ADDR", "127.0.0.1:6379"),
			Password: getEnv("CULTIVATION_REDIS_PASSWORD", ""),
			DB:       getEnvInt("CULTIVATION_REDIS_DB", 0),
		},
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

// BreakthroughConfig 突破概率配置
type BreakthroughConfig struct {
	BaseRates        map[string]BreakthroughRate `json:"base_rates"`         // 大境界突破配置 key=realmID
	BonusItems       []BonusItem                  `json:"bonus_items"`        // 辅助物品
	LevelBreakthroughs map[string]BreakthroughRate `json:"realm_level_breakthroughs"` // 小境界突破配置
}

// BreakthroughRate 突破概率与惩罚
type BreakthroughRate struct {
	BaseRate          float64 `json:"base_rate"`
	PenaltyRealmDrop  bool    `json:"penalty_realm_drop"`
	PenaltyExpLoss    float64 `json:"penalty_exp_loss"`
	Description       string  `json:"description"`
}

// BonusItem 突破辅助物品
type BonusItem struct {
	ItemID      string  `json:"item_id"`
	Name        string  `json:"name"`
	RateBonus   float64 `json:"rate_bonus"`
	Description string  `json:"description"`
}

// ConfigLoader 配置加载器
type ConfigLoader struct {
	config   *GameConfig
	filePath string
	opts     LoadOptions
	logger   *slog.Logger
}

// LoadOptions 加载选项
type LoadOptions struct {
	HotReload bool   // 是否启用热重载
	DataDir   string // 数据目录路径
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(logger *slog.Logger, dataDir string, opts LoadOptions) *ConfigLoader {
	if opts.DataDir == "" {
		opts.DataDir = dataDir
	}
	return &ConfigLoader{
		config:   &GameConfig{},
		filePath: opts.DataDir,
		opts:     opts,
		logger:   logger,
	}
}

// Load 加载所有配置文件
func (cl *ConfigLoader) Load() error {
	// 加载境界配置
	realms, err := loadJSON[[]model.Realm](cl.opts.DataDir + "/realms.json")
	if err != nil {
		return fmt.Errorf("加载境界配置失败: %w", err)
	}

	// 加载功法配置
	techniques, err := loadJSON[[]model.Technique](cl.opts.DataDir + "/techniques.json")
	if err != nil {
		return fmt.Errorf("加载功法配置失败: %w", err)
	}

	// 手动解析突破配置（包含嵌套结构较复杂）
	breakCfg, err := loadBreakthroughConfig(cl.opts.DataDir + "/breakthrough.json")
	if err != nil {
		return fmt.Errorf("加载突破配置失败: %w", err)
	}

	cl.config.mu.Lock()
	cl.config.Realms = realms
	cl.config.Techniques = techniques
	cl.config.Breakthrough = *breakCfg
	cl.config.mu.Unlock()

	return nil
}

// GetConfig 线程安全获取配置
func (cl *ConfigLoader) GetConfig() *GameConfig {
	cl.config.mu.RLock()
	defer cl.config.mu.RUnlock()
	return cl.config
}

// GetRealm 按ID获取境界配置
func (gc *GameConfig) GetRealm(id int) (*model.Realm, bool) {
	for i := range gc.Realms {
		if gc.Realms[i].ID == id {
			return &gc.Realms[i], true
		}
	}
	return nil, false
}

// GetRealmByLevel 通过大境界ID和小境界等级获取具体子境界
func (gc *GameConfig) GetRealmByLevel(realmID, level int) (*model.SubStage, bool) {
	realm, ok := gc.GetRealm(realmID)
	if !ok {
		return nil, false
	}
	for i := range realm.SubStages {
		if realm.SubStages[i].Level == level {
			return &realm.SubStages[i], true
		}
	}
	return nil, false
}

// GetTechnique 按ID获取功法
func (gc *GameConfig) GetTechnique(id int) (*model.Technique, bool) {
	for i := range gc.Techniques {
		if gc.Techniques[i].ID == id {
			return &gc.Techniques[i], true
		}
	}
	return nil, false
}

// GetBreakthroughRate 获取大境界突破基础概率
func (gc *GameConfig) GetBreakthroughRate(realmID, realmLevel int) float64 {
	// 先检查是否为大境界突破（小境界满级才突破大境界）
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	realm, ok := gc.GetRealm(realmID)
	if !ok {
		return 0
	}

	// 如果是突破大境界（realmID 是当前大境界，突破到下一个）
	key := fmt.Sprintf("%d", realmID+1)
	if rate, ok := gc.Breakthrough.BaseRates[key]; ok {
		return rate.BaseRate
	}

	// 如果是小境界突破
	levelKey := fmt.Sprintf("%d_%d", realmID, len(realm.SubStages))
	if rate, ok := gc.Breakthrough.LevelBreakthroughs[levelKey]; ok {
		return rate.BaseRate
	}

	return 0.5 // 默认50%
}

// GetBreakthroughPenalty 获取突破失败惩罚
func (gc *GameConfig) GetBreakthroughPenalty(realmID int) (bool, float64) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	key := fmt.Sprintf("%d", realmID+1)
	if rate, ok := gc.Breakthrough.BaseRates[key]; ok {
		return rate.PenaltyRealmDrop, rate.PenaltyExpLoss
	}

	// 小境界突破
	realm, ok := gc.GetRealm(realmID)
	if !ok {
		return false, 0.05
	}
	levelKey := fmt.Sprintf("%d_%d", realmID, len(realm.SubStages))
	if rate, ok := gc.Breakthrough.LevelBreakthroughs[levelKey]; ok {
		return rate.PenaltyRealmDrop, rate.PenaltyExpLoss
	}

	return false, 0.05
}

// GetBonusItem 获取辅助物品
func (gc *GameConfig) GetBonusItem(itemID string) (*BonusItem, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	for i := range gc.Breakthrough.BonusItems {
		if gc.Breakthrough.BonusItems[i].ItemID == itemID {
			return &gc.Breakthrough.BonusItems[i], true
		}
	}
	return nil, false
}

// --- 内部辅助函数 ---

func loadJSON[T any](filePath string) (T, error) {
	var result T
	data, err := os.ReadFile(filePath)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	return result, nil
}

func loadBreakthroughConfig(filePath string) (*BreakthroughConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var raw struct {
		BaseRates         map[string]json.RawMessage `json:"base_rates"`
		BonusItems        []BonusItem                `json:"bonus_items"`
		LevelBreakthroughs map[string]json.RawMessage `json:"realm_level_breakthroughs"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	cfg := &BreakthroughConfig{
		BaseRates:         make(map[string]BreakthroughRate),
		BonusItems:        raw.BonusItems,
		LevelBreakthroughs: make(map[string]BreakthroughRate),
	}

	for k, v := range raw.BaseRates {
		var rate struct {
			BaseRate          float64 `json:"base_rate"`
			PenaltyRealmDrop  bool    `json:"penalty_realm_drop"`
			PenaltyExpLoss    float64 `json:"penalty_exp_loss"`
			Description       string  `json:"description"`
		}
		if err := json.Unmarshal(v, &rate); err != nil {
			return nil, fmt.Errorf("解析突破率配置[%s]失败: %w", k, err)
		}
		cfg.BaseRates[k] = BreakthroughRate{
			BaseRate:         rate.BaseRate,
			PenaltyRealmDrop: rate.PenaltyRealmDrop,
			PenaltyExpLoss:   rate.PenaltyExpLoss,
			Description:      rate.Description,
		}
	}

	for k, v := range raw.LevelBreakthroughs {
		var rate struct {
			BaseRate          float64 `json:"base_rate"`
			PenaltyRealmDrop  bool    `json:"penalty_realm_drop"`
			PenaltyExpLoss    float64 `json:"penalty_exp_loss"`
			Description       string  `json:"description"`
		}
		if err := json.Unmarshal(v, &rate); err != nil {
			return nil, fmt.Errorf("解析小境界突破率配置[%s]失败: %w", k, err)
		}
		cfg.LevelBreakthroughs[k] = BreakthroughRate{
			BaseRate:         rate.BaseRate,
			PenaltyRealmDrop: rate.PenaltyRealmDrop,
			PenaltyExpLoss:   rate.PenaltyExpLoss,
			Description:      rate.Description,
		}
	}

	return cfg, nil
}
