// Package model 定义世界服务的数据模型
package model

import "time"

// ============================================================
// 天机阁推演系统模型
// ============================================================

// DivinationType 推演类型
type DivinationType string

const (
	DivinationBreakthrough DivinationType = "breakthrough" // 突破最佳时机
	DivinationTreasure     DivinationType = "treasure"     // 寻宝方位
	DivinationWeather      DivinationType = "weather"      // 天气/灵气潮汐预告
)

// DivinationTypeNames 推演类型中文名
var DivinationTypeNames = map[DivinationType]string{
	DivinationBreakthrough: "突破时机",
	DivinationTreasure:     "寻宝方位",
	DivinationWeather:      "灵气潮汐",
}

// DivinationTypeDescriptions 推演类型描述
var DivinationTypeDescriptions = map[DivinationType]string{
	DivinationBreakthrough: "推演最佳突破时机，突破成功率+5%",
	DivinationTreasure:     "推演天地灵宝方位，获得藏宝图",
	DivinationWeather:      "推演未来天气与灵气潮汐变化",
}

// Cost 推演消耗
const (
	DivineCostGoldBase  = 1000  // 基础消耗 1000 灵石
	DivineCostJadeBase  = 10    // 基础消耗 10 仙玉
	DivineFreeDaily     = 1     // 每日免费推演次数
	DivineMaxLevel      = 10    // 天机阁最高等级
	DivineExpPerUse     = 100   // 每次推演获得经验
	DivineExpPerLevel   = 500   // 每级所需经验
)

// AccuracyRange 准确度范围
const (
	DivineAccuracyMin = 0.50 // 最低准确度 50%
	DivineAccuracyMax = 0.95 // 最高准确度 95%
)

// DivinationLevel 天机阁等级
type DivinationLevel struct {
	Level     int   `json:"level"`      // 当前等级 1-10
	Exp       int   `json:"exp"`        // 当前经验
	ExpMax    int   `json:"exp_max"`    // 升级所需经验
	TotalUsed int64 `json:"total_used"` // 总使用次数
}

// DivinationResult 推演结果
type DivinationResult struct {
	ID         string        `json:"id"`          // 结果ID
	PlayerID   int64         `json:"player_id"`   // 玩家ID
	Type       DivinationType `json:"type"`        // 推演类型
	Content    string        `json:"content"`     // 推演内容文本
	Accuracy   float64       `json:"accuracy"`    // 本次准确度 0.5-0.95
	CostGold   int64         `json:"cost_gold"`   // 消耗灵石
	CostJade   int64         `json:"cost_jade"`   // 消耗仙玉
	IsFree     bool          `json:"is_free"`     // 是否免费
	Data       interface{}   `json:"data,omitempty"` // 结构化数据（藏宝图等）
	CreatedAt  time.Time     `json:"created_at"`
}

// DivinationRecord 玩家推演记录与状态
type DivinationRecord struct {
	PlayerID    int64              `json:"player_id"`
	Level       int                `json:"level"`        // 天机阁等级
	Exp         int                `json:"exp"`          // 当前经验
	TotalUsed   int64              `json:"total_used"`   // 总使用次数
	LastFreeAt  *time.Time         `json:"last_free_at,omitempty"` // 上次免费推演时间
	Results     []*DivinationResult `json:"results,omitempty"`     // 最近推演结果
}
