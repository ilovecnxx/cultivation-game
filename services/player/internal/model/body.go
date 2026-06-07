package model

import "time"

// 炼体大境界枚举
const (
	BodyRealmCopper    = 1 // 铜皮
	BodyRealmIronBone  = 2 // 铁骨
	BodyRealmDiamond   = 3 // 金刚
	BodyRealmUndying   = 4 // 不灭
	BodyRealmChaos     = 5 // 混沌体
)

// BodyRealmNames 炼体境界中文名
var BodyRealmNames = map[int32]string{
	BodyRealmCopper:   "铜皮",
	BodyRealmIronBone: "铁骨",
	BodyRealmDiamond:  "金刚",
	BodyRealmUndying:  "不灭",
	BodyRealmChaos:    "混沌体",
}

// BodyInfo 玩家炼体信息
// 每个玩家只有一条炼体记录，与 Player 一一对应
type BodyInfo struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	PlayerID  int64     `json:"player_id" gorm:"uniqueIndex;not null"`
	Realm     int32     `json:"realm" gorm:"default:0"`       // 当前炼体大境界(0=未开启)
	Level     int32     `json:"level" gorm:"default:0"`       // 当前炼体小层(1-10)
	Exp       int64     `json:"exp" gorm:"default:0"`         // 当前炼体经验
	MaxHPLost int64     `json:"max_hp_lost" gorm:"default:0"` // 突破失败累计扣减的HP上限
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BodyRealmConfig 炼体境界配置
type BodyRealmConfig struct {
	ID                 int32   `json:"id"`
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	LevelCap           int32   `json:"level_cap"`
	HPPerLevel         int64   `json:"hp_per_level"`
	DefensePerLevel    int64   `json:"defense_per_level"`
	DamageReduction    float64 `json:"damage_reduction"`
	BaseBreakthroughRate float64 `json:"base_breakthrough_rate"`
	UnlockRealmRequired int32   `json:"unlock_realm_required"`
	ExpPerTrain        int64   `json:"exp_per_train"`
}

// BodyRealmsData JSON 配置文件根结构
type BodyRealmsData struct {
	Realms      []BodyRealmConfig `json:"realms"`
	TrainConfig TrainConfig       `json:"train_config"`
}

// TrainConfig 炼体训练配置
type TrainConfig struct {
	DamageToExpRatio float64 `json:"damage_to_exp_ratio"`
	DailyDamageCap   int64   `json:"daily_damage_cap"`
	PillExpBase      int64   `json:"pill_exp_base"`
	DungeonExpBase   int64   `json:"dungeon_exp_base"`
	FailureHPPenalty int64   `json:"failure_hp_penalty"`
	RecoverHPPerHour int64   `json:"recover_hp_per_hour"`
}

// BodyBonuses 炼体提供的属性加成（计算结果）
type BodyBonuses struct {
	HP              int64   `json:"hp"`
	Defense         int64   `json:"defense"`
	DamageReduction float64 `json:"damage_reduction"`
}

// ---------- 请求/响应结构 ----------

// BodyTrainRequest 炼体训练请求
type BodyTrainRequest struct {
	TrainType string `json:"train_type" binding:"required,oneof=damage pill dungeon"`
	Amount    int64  `json:"amount" binding:"min=0"`          // 伤害量/丹药经验
	PillID    int64  `json:"pill_id,omitempty"`               // 丹药ID（train_type=pill时使用）
}

// BodyBreakthroughRequest 炼体突破请求
type BodyBreakthroughRequest struct {
	PillID int64 `json:"pill_id,omitempty"` // 辅助突破的丹药ID(可选)
}

// BodyStatusResponse 炼体状态响应
type BodyStatusResponse struct {
	BodyInfo   *BodyInfo    `json:"body_info"`
	Bonuses    *BodyBonuses `json:"bonuses"`
	NextRealm  string       `json:"next_realm,omitempty"`   // 下级境界名
	NextLevel  int32        `json:"next_level"`             // 下级小层
	MaxExp     int64        `json:"max_exp"`                // 当前经验上限
	Rate       float64      `json:"breakthrough_rate"`      // 当前突破成功率
	RealmName  string       `json:"realm_name,omitempty"`   // 当前境界名
}

// ---------- 辅助方法 ----------

// CalcBonuses 根据当前炼体境界和等级计算属性加成
func (b *BodyInfo) CalcBonuses(configs []BodyRealmConfig) *BodyBonuses {
	if b.Realm <= 0 || b.Level <= 0 {
		return &BodyBonuses{HP: 0, Defense: 0, DamageReduction: 0}
	}

	var totalHP, totalDef int64
	var totalDR float64

	for i := int32(1); i <= b.Realm; i++ {
		cfg := getRealmConfig(configs, i)
		if cfg == nil {
			continue
		}

		level := b.Level
		if i < b.Realm {
			level = cfg.LevelCap // 已满级的前置境界取满层
		}

		totalHP += cfg.HPPerLevel * int64(level)
		totalDef += cfg.DefensePerLevel * int64(level)
		totalDR += cfg.DamageReduction
	}

	// 扣除突破失败损失的HP上限
	effectiveHP := totalHP - b.MaxHPLost
	if effectiveHP < 0 {
		effectiveHP = 0
	}

	return &BodyBonuses{
		HP:              effectiveHP,
		Defense:         totalDef,
		DamageReduction: totalDR,
	}
}

// getRealmConfig 工具函数：根据ID查找境界配置
func getRealmConfig(configs []BodyRealmConfig, id int32) *BodyRealmConfig {
	for i := range configs {
		if configs[i].ID == id {
			return &configs[i]
		}
	}
	return nil
}
