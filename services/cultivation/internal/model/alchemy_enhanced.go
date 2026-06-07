// Package model 炼丹系统增强数据结构定义
package model

import "time"

// AlchemySession 炼丹会话（小游戏状态）
type AlchemySession struct {
	PlayerID    string    `json:"player_id"`
	FormulaID   int       `json:"formula_id"`
	FormulaName string    `json:"formula_name"`
	FurnaceID   string    `json:"furnace_id"`
	StartTime   time.Time `json:"start_time"`
	HeatZone    int       `json:"heat_zone"`   // 0=low, 1=medium, 2=high
	HeatTimer   float64   `json:"heat_timer"`   // 0-1 progress within zone
	Phase       string    `json:"phase"`        // "heating", "adding", "condensing", "completed"
	Ingredients []string  `json:"ingredients"`  // material IDs already added
	Score       int       `json:"score"`        // 0-100 mini-game score
	BaseQuality int       `json:"base_quality"` // base quality tier from skill
	FinalQuality int      `json:"final_quality"` // final quality tier after adjustments
	Toxicity    int       `json:"toxicity"`     // calculated toxicity
	Success     bool      `json:"success"`
	Completed   bool      `json:"completed"`
}

// Formula 丹方定义（增强版）
type Formula struct {
	ID                int      `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Materials         []string `json:"materials"`         // required material IDs
	BaseQuality       int      `json:"base_quality"`       // minimum quality tier when crafted
	MinLevel          int      `json:"min_level"`          // minimum alchemy level
	RealmRequired     int      `json:"realm_required"`     // minimum realm ID
	ResearchDifficulty float64 `json:"research_difficulty"` // 0-1 difficulty for research
	IsRare            bool     `json:"is_rare"`             // requires special materials (boss drops)
	Effect            string   `json:"effect"`             // description of pill effect
	CraftTime         int      `json:"craft_time"`          // base craft time in seconds
	ExpValue          int      `json:"exp_value"`           // base exp gained
}

// QualityMultiplier 品质倍率（效果倍率）
func QualityMultiplier(q int) float64 {
	switch q {
	case 0: // junk
		return 0.5
	case 1: // common
		return 1.0
	case 2: // good
		return 1.5
	case 3: // superior
		return 2.0
	case 4: // premium
		return 2.5
	case 5: // immortal
		return 3.0
	default:
		return 1.0
	}
}

// QualityToxicity 品质对应毒性值
func QualityToxicity(q int) int {
	switch q {
	case 0:
		return 5
	case 1:
		return 10
	case 2:
		return 15
	case 3:
		return 20
	case 4:
		return 25
	case 5:
		return 10 // immortal pills are pure, less toxic
	default:
		return 10
	}
}

// PlayerFormula 玩家已研究的丹方
type PlayerFormula struct {
	PlayerID   uint64    `json:"player_id"`
	FormulaID  int       `json:"formula_id"`
	Name       string    `json:"name"`
	Discovered bool      `json:"discovered"`
	DiscoveredAt time.Time `json:"discovered_at"`
	CraftCount int       `json:"craft_count"`
}

// PlayerToxicity 玩家丹毒
type PlayerToxicity struct {
	PlayerID  uint64    `json:"player_id"`
	Value     int       `json:"value"`      // 0-100
	UpdatedAt time.Time `json:"updated_at"` // last toxicity update time
}

// FurnaceQuality 丹炉品质等级
type FurnaceQuality int

const (
	FurnaceBronze  FurnaceQuality = 0 // 青铜丹炉
	FurnaceSilver  FurnaceQuality = 1 // 白银丹炉
	FurnaceGold    FurnaceQuality = 2 // 黄金丹炉
	FurnaceImmortal FurnaceQuality = 3 // 仙品丹炉
)

// FurnaceQualityName 丹炉品质中文名
var FurnaceQualityNames = map[FurnaceQuality]string{
	FurnaceBronze:   "青铜丹炉",
	FurnaceSilver:   "白银丹炉",
	FurnaceGold:     "黄金丹炉",
	FurnaceImmortal: "仙品丹炉",
}

func (f FurnaceQuality) String() string {
	if name, ok := FurnaceQualityNames[f]; ok {
		return name
	}
	return "未知丹炉"
}

// FurnaceQualityFloor 丹炉品质对应的最低品质档位
var FurnaceQualityFloor = map[FurnaceQuality]int{
	FurnaceBronze:   0, // junk floor
	FurnaceSilver:   1, // common floor
	FurnaceGold:     2, // good floor
	FurnaceImmortal: 3, // superior floor
}

// FurnaceQualityBonus 丹炉品质对应的小游戏分数加成
var FurnaceQualityBonus = map[FurnaceQuality]int{
	FurnaceBronze:   0,
	FurnaceSilver:   5,
	FurnaceGold:     10,
	FurnaceImmortal: 20,
}

// Furnace 玩家丹炉
type PlayerFurnace struct {
	PlayerID   uint64         `json:"player_id"`
	ID         string         `json:"id"`
	Quality    FurnaceQuality `json:"quality"`
	Durability int            `json:"durability"`    // remaining uses
	MaxDurability int         `json:"max_durability"` // max uses
}

// NewPlayerFurnace 创建初始青铜丹炉
func NewPlayerFurnace(playerID uint64) *PlayerFurnace {
	return &PlayerFurnace{
		PlayerID:      playerID,
		ID:            "furnace_default",
		Quality:       FurnaceBronze,
		Durability:    100,
		MaxDurability: 100,
	}
}

// ResearchRecord 研究记录
type ResearchRecord struct {
	PlayerID    uint64    `json:"player_id"`
	FormulaID   int       `json:"formula_id"`
	FormulaName string    `json:"formula_name"`
	Success     bool      `json:"success"`
	Timestamp   time.Time `json:"timestamp"`
}

// ResearchAttempt 研究尝试信息
type ResearchAttempt struct {
	PlayerID   uint64    `json:"player_id"`
	DailyCount int       `json:"daily_count"`   // today's free attempts used
	LastReset  time.Time `json:"last_reset"`     // last daily reset time
	NextFreeAt time.Time `json:"next_free_at"`   // next free attempt available (3/day)
}

// CraftResultEnhanced 增强版炼制结果
type CraftResultEnhanced struct {
	Success      bool   `json:"success"`
	Quality      int    `json:"quality"`
	QualityName  string `json:"quality_name"`
	PillID       string `json:"pill_id,omitempty"`
	PillName     string `json:"pill_name,omitempty"`
	Score        int    `json:"score"`
	Toxicity     int    `json:"toxicity"`
	TotalToxicity int   `json:"total_toxicity"` // after adding this pill's toxicity
	ExpGained    int64  `json:"exp_gained"`
	AlchemyExp   int64  `json:"alchemy_exp"`
	QualityUp    bool   `json:"quality_up,omitempty"`    // critical success upgrade
	DurabilityUsed int  `json:"durability_used"`          // furnace durability consumed
	Message      string `json:"message,omitempty"`
}

// DetoxResult 解毒结果
type DetoxResult struct {
	Success      bool   `json:"success"`
	ToxicityReduced int `json:"toxicity_reduced"`
	CurrentToxicity int `json:"current_toxicity"`
	Message      string `json:"message"`
}

// FurnaceUpgradeResult 丹炉升级结果
type FurnaceUpgradeResult struct {
	Success     bool            `json:"success"`
	OldQuality FurnaceQuality   `json:"old_quality"`
	NewQuality FurnaceQuality   `json:"new_quality"`
	OldQualityName string       `json:"old_quality_name"`
	NewQualityName string       `json:"new_quality_name"`
	Message     string          `json:"message"`
}
