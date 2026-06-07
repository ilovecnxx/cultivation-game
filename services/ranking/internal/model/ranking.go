// Package model 定义排行榜核心数据模型。
package model

import "time"

// RankingType 排行榜类型枚举。
type RankingType string

const (
	RankingTypeRealm       RankingType = "realm"        // 境界榜（realm_id + level 计算评分）
	RankingTypeCombatPower RankingType = "combat_power" // 战力榜
	RankingTypeWealth      RankingType = "wealth"       // 财富榜
	RankingTypeSect        RankingType = "sect"         // 宗门榜
)

// 排行榜类型对应的中文名。
var RankingTypeNames = map[RankingType]string{
	RankingTypeRealm:       "境界榜",
	RankingTypeCombatPower: "战力榜",
	RankingTypeWealth:      "财富榜",
	RankingTypeSect:        "宗门榜",
}

// RankingEntry 排行榜单个条目。
type RankingEntry struct {
	PlayerID  uint64  `json:"player_id"`  // 玩家 ID
	Nickname  string  `json:"nickname"`   // 昵称
	RealmName string  `json:"realm_name"` // 境界名称（如"筑基三层"）
	Score     float64 `json:"score"`      // 排名分数
	Rank      int32   `json:"rank"`       // 当前排名（从1开始）
	UpdatedAt int64   `json:"updated_at"` // 最后更新时间（Unix 时间戳）
}

// RankingEntryJSON 用于 Redis Hash 存储的扁平结构。
type RankingEntryJSON struct {
	PlayerID  uint64 `json:"player_id"`
	Nickname  string `json:"nickname"`
	RealmName string `json:"realm_name"`
	Score     string `json:"score"`      // Redis 中存为 string
	UpdatedAt int64  `json:"updated_at"`
}

// Leaderboard 排行榜元数据。
type Leaderboard struct {
	ID             string       `json:"id"`              // 排行榜唯一标识（同 RankingType）
	Name           string       `json:"name"`            // 排行榜中文名
	RankingType    RankingType  `json:"ranking_type"`    // 排行榜类型
	UpdateInterval int          `json:"update_interval"` // 更新间隔（秒）
	CacheTTL       int          `json:"cache_ttl"`       // Top 100 缓存 TTL（秒）
	DecayEnabled   bool         `json:"decay_enabled"`   // 是否启用分数衰减
	DecayRate      float64      `json:"decay_rate"`      // 每日衰减率（如 0.05 = 5%）
	DecayAfterDays int          `json:"decay_after_days"` // 多少天不活跃开始衰减
}

// DefaultLeaderboards 默认排行榜定义。
var DefaultLeaderboards = []*Leaderboard{
	{
		ID:             string(RankingTypeRealm),
		Name:           "境界榜",
		RankingType:    RankingTypeRealm,
		UpdateInterval: 30,
		CacheTTL:       30,
		DecayEnabled:   false, // 境界不衰减
		DecayRate:      0,
		DecayAfterDays: 0,
	},
	{
		ID:             string(RankingTypeCombatPower),
		Name:           "战力榜",
		RankingType:    RankingTypeCombatPower,
		UpdateInterval: 30,
		CacheTTL:       30,
		DecayEnabled:   true,
		DecayRate:      0.03, // 每日衰减 3%
		DecayAfterDays: 7,
	},
	{
		ID:             string(RankingTypeWealth),
		Name:           "财富榜",
		RankingType:    RankingTypeWealth,
		UpdateInterval: 60,
		CacheTTL:       60,
		DecayEnabled:   true,
		DecayRate:      0.02, // 每日衰减 2%
		DecayAfterDays: 14,
	},
	{
		ID:             string(RankingTypeSect),
		Name:           "宗门榜",
		RankingType:    RankingTypeSect,
		UpdateInterval: 60,
		CacheTTL:       60,
		DecayEnabled:   false, // 宗门等级不衰减
		DecayRate:      0,
		DecayAfterDays: 0,
	},
}

// GetLeaderboard 根据类型获取排行榜元数据。
func GetLeaderboard(typ RankingType) *Leaderboard {
	for _, lb := range DefaultLeaderboards {
		if lb.RankingType == typ {
			return lb
		}
	}
	return nil
}

// IsValidType 检查排行榜类型是否合法。
func IsValidType(typ string) bool {
	switch RankingType(typ) {
	case RankingTypeRealm, RankingTypeCombatPower, RankingTypeWealth, RankingTypeSect:
		return true
	}
	return false
}

// PageRequest 分页请求通用参数。
type PageRequest struct {
	Page     int32 // 页码，从1开始
	PageSize int32 // 每页条数（默认20，最大100）
}

// Normalize 规范化分页参数。
func (p *PageRequest) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
}

// Offset 计算 Redis ZREVRANGE 的起始偏移。
func (p *PageRequest) Offset() int64 {
	return int64((p.Page - 1) * p.PageSize)
}

// Count 计算 Redis ZREVRANGE 的返回数量。
func (p *PageRequest) Count() int64 {
	return int64(p.PageSize)
}

// NeighborCount 获取玩家排名时，上下各取多少个邻居。
const NeighborCount = 5

// Redis key 模板常量。
const (
	// RedisKeyScoreZSet 玩家分数有序集合。
	// key = ranking:zset:{type}
	RedisKeyScoreZSet = "ranking:zset:%s"

	// RedisKeyPlayerInfo 玩家附属信息 Hash。
	// key = ranking:info:{type}
	RedisKeyPlayerInfo = "ranking:info:%s"

	// RedisKeySnapshot Top 100 快照缓存。
	// key = ranking:snapshot:{type}
	RedisKeySnapshot = "ranking:snapshot:%s"

	// RedisKeyLastActivity 玩家最后活跃时间 Hash。
	// key = ranking:last_active:{type}
	RedisKeyLastActivity = "ranking:last_active:%s"

	// RedisLockScoreUpdate 分数更新分布式锁。
	// key = ranking:lock:update:{type}:{player_id}
	RedisLockScoreUpdate = "ranking:lock:update:%s:%d"

	// LuaScriptAtomicUpdate Lua 脚本名（用于原子更新）。
	LuaScriptAtomicUpdate = `
-- KEYS[1] = zset key
-- KEYS[2] = info hash key
-- KEYS[3] = last_active hash key
-- ARGV[1] = player_id
-- ARGV[2] = score
-- ARGV[3] = nickname
-- ARGV[4] = realm_name
-- ARGV[5] = now (unix timestamp)
redis.call('ZADD', KEYS[1], ARGV[2], ARGV[1])
redis.call('HSET', KEYS[2], ARGV[1], cjson.encode({
	player_id = tonumber(ARGV[1]),
	nickname = ARGV[3],
	realm_name = ARGV[4],
	score = ARGV[2],
	updated_at = tonumber(ARGV[5])
}))
redis.call('HSET', KEYS[3], ARGV[1], ARGV[5])
return 1
`
)

// ScoreForRealm 计算境界评分 = realm_id * 10000 + realm_level。
// 这样可以保证境界越高分数越高。
func ScoreForRealm(realmID, realmLevel uint32) float64 {
	return float64(realmID)*10000 + float64(realmLevel)
}

// NewEntry 构造一个排行榜条目。
func NewEntry(playerID uint64, nickname, realmName string, score float64, rank int32) *RankingEntry {
	return &RankingEntry{
		PlayerID:  playerID,
		Nickname:  nickname,
		RealmName: realmName,
		Score:     score,
		Rank:      rank,
		UpdatedAt: time.Now().Unix(),
	}
}

// Snapshot Top 100 缓存快照。
type Snapshot struct {
	Entries  []*RankingEntry `json:"entries"`
	RefreshedAt time.Time    `json:"refreshed_at"`
}
