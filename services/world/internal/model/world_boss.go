package model

import "time"

// ============================================================
// 世界 Boss 扩展模型
// ============================================================

// BossConfig Boss 静态配置(14个区域各1个)
type BossConfig struct {
	BossID      string  `json:"boss_id"`      // 唯一ID, 如 "boss_region_01"
	RegionID    string  `json:"region_id"`    // 所在区域ID
	RegionName  string  `json:"region_name"`  // 区域中文名
	Name        string  `json:"name"`         // Boss名称
	Level       int     `json:"level"`        // Boss等级
	MaxHP       float64 `json:"max_hp"`       // 最大血量
	Attack      float64 `json:"attack"`       // 攻击力
	Defense     float64 `json:"defense"`      // 防御力
	GoldReward  int64   `json:"gold_reward"`  // 参与灵石奖励(基础)
	ExpReward   int64   `json:"exp_reward"`   // 参与修为奖励(基础)
	Description string  `json:"description"`  // Boss描述
}

// BossRewardRank 伤害排名奖励配置
type BossRewardRank struct {
	RankFrom    int   `json:"rank_from"`
	RankTo      int   `json:"rank_to"`
	GoldBonus   int64 `json:"gold_bonus"`    // 额外灵石
	ExpBonus    int64 `json:"exp_bonus"`     // 额外修为
	ItemID      int64 `json:"item_id,omitempty"` // 额外物品ID
	ItemName    string `json:"item_name,omitempty"`
	ItemQuantity int32 `json:"item_quantity,omitempty"`
}

// BossKillRecord Boss击杀记录
type BossKillRecord struct {
	BossID      string    `json:"boss_id"`
	BossName    string    `json:"boss_name"`
	RegionID    string    `json:"region_id"`
	RegionName  string    `json:"region_name"`
	KilledAt    time.Time `json:"killed_at"`
	KillerID    string    `json:"killer_id"`    // 最后一击玩家ID
	KillerName  string    `json:"killer_name"`
	Participants int      `json:"participants"` // 参与人数
	TopDamage   []WorldBossDamage `json:"top_damage"` // Top10
	ReplayLog   []string  `json:"replay_log,omitempty"` // 击杀回放简讯
}

// BossAttackRequest 攻击Boss请求
type BossAttackRequest struct {
	PlayerID   string  `json:"player_id"`
	PlayerName string  `json:"player_name"`
	BossID     string  `json:"boss_id"`
	AttackVal  float64 `json:"attack_val"` // 玩家攻击力(由客户端传入或服务端计算)
}

// BossAttackResult 攻击Boss结果
type BossAttackResult struct {
	Damage         float64 `json:"damage"`          // 实际造成的伤害
	BossRemainHP   float64 `json:"boss_remain_hp"` // Boss剩余血量
	BossMaxHP      float64 `json:"boss_max_hp"`
	BossStatus     string  `json:"boss_status"`     // alive / defeated
	Critical       bool    `json:"critical"`        // 是否暴击
	LastHit        bool    `json:"last_hit"`         // 是否最后一击
	GoldReward     int64   `json:"gold_reward"`      // 本次获得灵石
	ExpReward      int64   `json:"exp_reward"`       // 本次获得修为
}

// BossListResponse Boss列表响应
type BossListResponse struct {
	Bosses []*BossStatusBrief `json:"bosses"`
}

// BossStatusBrief Boss简要状态
type BossStatusBrief struct {
	BossConfig
	Status    string    `json:"status"`     // alive / defeated / dormant
	HP        float64   `json:"hp"`         // 当前血量
	HPPct     float64   `json:"hp_pct"`     // 血量百分比 0-100
	SpawnedAt time.Time `json:"spawned_at"` // 刷新时间
	SpawnLeft string    `json:"spawn_left"` // 下次刷新倒计时(可读)
}

// BossStatusDetail Boss详细状态(含排行榜)
type BossStatusDetail struct {
	BossStatusBrief
	DamageRank  []*WorldBossDamage `json:"damage_rank"`
	LastKill    *BossKillRecord    `json:"last_kill,omitempty"`
}

// 奖励排名配置
// 排名区间: 使用RankFrom/RankTo表示绝对排名范围
// 百分比区间: RankFrom=101表示1%+, RankFrom=151表示51%+, RankTo=0表示开放区间
var BossRewardRanks = []BossRewardRank{
	{RankFrom: 1, RankTo: 1, GoldBonus: 10000, ExpBonus: 100000, ItemID: 3001, ItemName: "BOSS击杀礼盒", ItemQuantity: 1},
	{RankFrom: 2, RankTo: 10, GoldBonus: 5000, ExpBonus: 50000, ItemID: 3002, ItemName: "高级灵石包", ItemQuantity: 1},
	{RankFrom: 11, RankTo: 50, GoldBonus: 2000, ExpBonus: 20000},
	{RankFrom: 51, RankTo: 100, GoldBonus: 500, ExpBonus: 5000},
}

// LastHitExtraReward 最后一击额外奖励
const (
	LastHitGoldBonus = 10000
	LastHitExpBonus  = 100000
)

// BaseParticipateGold 参与基础灵石
const BaseParticipateGold int64 = 200

// BaseParticipateExp 参与基础修为
const BaseParticipateExp int64 = 5000

// WorldBoss 世界BOSS
type WorldBoss struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Region    string     `json:"region"`
	MaxHP     int64      `json:"max_hp"`
	CurrentHP int64      `json:"current_hp"`
	Level     int        `json:"level"`
	Status    string     `json:"status"`
	SpawnedAt time.Time  `json:"spawned_at,omitempty"`
	KilledAt  time.Time  `json:"killed_at,omitempty"`
	Rewards   BossReward `json:"rewards"`
}

type BossReward struct {
	Exp   int64 `json:"exp"`
	Money int64 `json:"money"`
	Items []int `json:"items"`
}

type WorldBossDamage struct {
	PlayerID   uint64    `json:"player_id"`
	PlayerName string    `json:"player_name"`
	Damage     int64     `json:"damage"`
	Time       time.Time `json:"time"`
}

type AttackResult struct {
	Damage     int64  `json:"damage"`
	BossHP     int64  `json:"boss_hp"`
	BossMaxHP  int64  `json:"boss_max_hp"`
	Killed     bool   `json:"killed"`
	KillerName string `json:"killer_name,omitempty"`
}

type BossSession struct {
	BossID    string    `json:"boss_id"`
	StartedAt time.Time `json:"started_at"`
	Active    bool      `json:"active"`
}
