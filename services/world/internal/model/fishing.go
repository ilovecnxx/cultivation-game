// Package model 提供灵鱼垂钓系统的数据模型
package model

// FishingSpot 钓鱼点配置
type FishingSpot struct {
	ID          string  `json:"id"`
	RegionID    string  `json:"region_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	MinRealm    int     `json:"min_realm"`
	MaxRealm    int     `json:"max_realm"`
	FishIDs     []int64 `json:"fish_ids"`
}

// FishType 鱼类配置
type FishType struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Rarity          int     `json:"rarity"`
	MinRealm        int     `json:"min_realm"`
	BaseWeight      float64 `json:"base_weight"`
	ExpReward       int     `json:"exp_reward"`
	SpiritStoneValue int    `json:"spirit_stone_value"`
	Description     string  `json:"description"`
}

// PlayerFishing 玩家钓鱼信息
type PlayerFishing struct {
	ID               int64   `json:"id"`
	PlayerID         int64   `json:"player_id"`
	FishingSkillLevel int    `json:"fishing_skill_level"`
	FishingExp       int     `json:"fishing_exp"`
	TotalCaught      int     `json:"total_caught"`
	BestCatchID      *int64  `json:"best_catch_id"`
	BestCatchWeight  float64 `json:"best_catch_weight"`
	BaitCount        int     `json:"bait_count"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// FishingRecord 钓鱼记录
type FishingRecord struct {
	ID        int64   `json:"id"`
	PlayerID  int64   `json:"player_id"`
	FishID    int64   `json:"fish_id"`
	SpotID    int64   `json:"spot_id"`
	Weight    float64 `json:"weight"`
	ExpGained int     `json:"exp_gained"`
	CaughtAt  string  `json:"caught_at"`
}

// FishingSpotsData JSON 数据文件根结构
type FishingSpotsData struct {
	Spots     []FishingSpot `json:"spots"`
	FishTypes []FishType    `json:"fish_types"`
}

// FishingResult 钓鱼结果
type FishingResult struct {
	Fish        FishType `json:"fish"`
	Weight      float64  `json:"weight"`
	ExpGained   int      `json:"exp_gained"`
	IsBestCatch bool     `json:"is_best_catch"`
	Message     string   `json:"message"`
}

// FishingSession 当前钓鱼会话状态
type FishingSession struct {
	PlayerID   int64  `json:"player_id"`
	SpotID     string `json:"spot_id"`
	SpotName   string `json:"spot_name"`
	Phase      string `json:"phase"` // "casting", "waiting", "hooking", "reeling", "done"
	Fish       *FishType  `json:"fish"`
	Weight     float64    `json:"weight"`
	StartedAt  int64  `json:"started_at"`
	BiteAt     int64  `json:"bite_at"`     // 鱼上钩的时间戳
	Tension    float64 `json:"tension"`    // 当前张力(0-100)
}

// FishingInfo 玩家钓鱼信息响应
type FishingInfo struct {
	PlayerID          int64            `json:"player_id"`
	FishingSkillLevel  int             `json:"fishing_skill_level"`
	FishingExp        int              `json:"fishing_exp"`
	ExpToNextLevel    int              `json:"exp_to_next_level"`
	TotalCaught       int              `json:"total_caught"`
	BestCatch         *BestCatchInfo   `json:"best_catch"`
	BaitCount         int              `json:"bait_count"`
	History           []FishingRecord  `json:"history"`
}

// BestCatchInfo 最佳捕获信息
type BestCatchInfo struct {
	FishName string  `json:"fish_name"`
	Weight   float64 `json:"weight"`
}
