package service

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// ---------- 常量 ----------

const (
	DungeonDailyFree     = 3  // 每日免费进入次数
	DungeonMaxBuyTimes   = 3  // 每日最多购买次数
	DungeonBuyCostJade   = 50 // 每次购买消耗玉璧
	TeamMaxMembers       = 3  // 队伍最大人数(含队长)
	InviteExpireSec      = 60 // 邀请过期时间(秒)
	ThreeStarTimeRatio   = 0.5 // 3星所需时间 <= 时限*该比例
	TwoStarTimeRatio     = 0.8 // 2星所需时间 <= 时限*该比例
)

// ---------- 静态配置类型 ----------

// DungeonConfig 秘境静态配置
type DungeonConfig struct {
	ID               int                  `json:"id"`
	Name             string               `json:"name"`
	Color1           string               `json:"color1"`
	Color2           string               `json:"color2"`
	RecommendLevel   int                  `json:"recommend_level"`
	TotalFloors      int                  `json:"total_floors"`
	UnlockRealm      int                  `json:"unlock_realm"`
	UnlockCondition  string               `json:"unlock_condition"`
	Floors           []DungeonFloorConfig `json:"floors"`
	EntryCost        EntryCost            `json:"entry_cost"`
	DailyFree        int                  `json:"daily_free"`
	MaxBuyTimes      int                  `json:"max_buy_times"`
	BuyCostJade      int64                `json:"buy_cost_jade"`
	TeamMaxSize      int                  `json:"team_max_size"`
	TimeLimitSec     int                  `json:"time_limit_sec"`
	FirstClearReward []RewardItem         `json:"first_clear_reward"`
	CompletionBonus  float64              `json:"completion_bonus"`
}

// DungeonFloorConfig 层配置
type DungeonFloorConfig struct {
	Floor        int            `json:"floor"`
	IsBoss       bool           `json:"is_boss"`
	Name         string         `json:"name"`
	Monsters     []MonsterConfig `json:"monsters"`
	Boss         *MonsterConfig `json:"boss,omitempty"`
	Rewards      FloorRewards   `json:"rewards"`
	TimeLimitSec int            `json:"time_limit_sec"`
}

// MonsterConfig 怪物配置
type MonsterConfig struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	HP    int64  `json:"hp"`
	Atk   int64  `json:"atk"`
	Def   int64  `json:"def"`
	Speed int64  `json:"speed"`
	Level int    `json:"level"`
}

// EntryCost 进入消耗
type EntryCost struct {
	SpiritStones int64 `json:"spirit_stones"`
	Stamina      int   `json:"stamina"`
}

// FloorRewards 层奖励
type FloorRewards struct {
	Exp   int64        `json:"exp"`
	Money int64        `json:"money"`
	Items []RewardItem `json:"items"`
}

// RewardItem 奖励物品
type RewardItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ---------- 运行时数据类型 ----------

// DungeonPlayerData 玩家的秘境进度
type DungeonPlayerData struct {
	PlayerID       string                    `json:"player_id"`
	Progress       map[int]*DungeonProgress  `json:"progress"`   // dungeonID -> progress
	DailyFreeUsed  map[string]int            `json:"daily_free_used"` // "dungeonID:date" -> count
	DailyBuyUsed   map[string]int            `json:"daily_buy_used"`  // "dungeonID:date" -> count
	FirstCleared   map[int]bool              `json:"first_cleared"`   // dungeonIDs claimed first-clear bonus
}

// DungeonProgress 单个秘境进度
type DungeonProgress struct {
	DungeonID    int   `json:"dungeon_id"`
	HighestFloor int   `json:"highest_floor"`
	BestTime     int   `json:"best_time"`
	Stars        int   `json:"stars"`
	Completed    bool  `json:"completed"`
}

// DungeonSessionData 当前秘境会话
type DungeonSessionData struct {
	PlayerID     string   `json:"player_id"`
	DungeonID    int      `json:"dungeon_id"`
	CurrentFloor int      `json:"current_floor"`
	MaxFloor     int      `json:"max_floor"`
	Team         []string `json:"team"`
	StartedAt    int64    `json:"started_at"`
	FloorTimes   []int    `json:"floor_times"`
	Completed    bool     `json:"completed"`
	Failed       bool     `json:"failed"`
	State        string   `json:"state"` // "exploring", "fighting", "done"
	TotalTimeSec int      `json:"total_time_sec"`
	Rating       int      `json:"rating"` // 0=unrated, 1-3 stars
}

// TeamInvite 组队邀请
type TeamInvite struct {
	ID        string   `json:"id"`
	DungeonID int      `json:"dungeon_id"`
	HostID    string   `json:"host_id"`
	HostName  string   `json:"host_name"`
	TargetID  string   `json:"target_id"`
	Status    string   `json:"status"` // "pending","accepted","declined","expired"
	CreatedAt int64    `json:"created_at"`
	ExpiresAt int64    `json:"expires_at"`
	Members   []string `json:"members"` // 当前已确认成员
}

// DungeonFightResult 战斗结果
type DungeonFightResult struct {
	Win          bool           `json:"win"`
	Floor        int            `json:"floor"`
	IsBoss       bool           `json:"is_boss"`
	Rewards      *FloorRewards  `json:"rewards,omitempty"`
	Rating       int            `json:"rating,omitempty"`
	Completed    bool           `json:"completed"`
	TimeUsedSec  int            `json:"time_used_sec"`
	Logs         []string       `json:"logs"`
}

// DungeonListEntry 列表条目
type DungeonListEntry struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Color1          string `json:"color1"`
	Color2          string `json:"color2"`
	RecommendLevel  int    `json:"recommend_level"`
	TotalFloors     int    `json:"total_floors"`
	CurrentFloor    int    `json:"current_floor"`
	Unlocked        bool   `json:"unlocked"`
	UnlockCondition string `json:"unlock_condition"`
}

// DungeonDetail 秘境详情
type DungeonDetail struct {
	ID              int                  `json:"id"`
	Name            string               `json:"name"`
	Color1          string               `json:"color1"`
	Color2          string               `json:"color2"`
	RecommendLevel  int                  `json:"recommend_level"`
	TotalFloors     int                  `json:"total_floors"`
	EntryCost       EntryCost            `json:"entry_cost"`
	DailyFree       int                  `json:"daily_free"`
	DailyUsed       int                  `json:"daily_used"`
	BuyUsed         int                  `json:"buy_used"`
	MaxBuyTimes     int                  `json:"max_buy_times"`
	BuyCostJade     int64                `json:"buy_cost_jade"`
	HighestFloor    int                  `json:"highest_floor"`
	BestStars       int                  `json:"best_stars"`
	Completed       bool                 `json:"completed"`
	FirstCleared    bool                 `json:"first_cleared"`
	UnlockCondition string               `json:"unlock_condition"`
	Floors          []DungeonFloorConfig `json:"floors"`
}

// FloorRewardClaim 领取奖励结果
type FloorRewardClaim struct {
	Exp   int64        `json:"exp"`
	Money int64        `json:"money"`
	Items []RewardItem `json:"items"`
}

// ---------- DungeonService ----------

// DungeonService 秘境系统服务
type DungeonService struct {
	mu          sync.RWMutex
	dungeons    map[int]*DungeonConfig           // 静态配置
	players     map[string]*DungeonPlayerData    // playerID -> 数据
	sessions    map[string]*DungeonSessionData   // playerID -> 当前会话
	invites     map[string]*TeamInvite           // inviteID -> invite
	playerNames map[string]string                // playerID -> nickname
	redisClient *redis.Client
}

// NewDungeonService 创建秘境服务并加载内置数据
func NewDungeonService() *DungeonService {
	svc := &DungeonService{
		dungeons:    make(map[int]*DungeonConfig),
		players:     make(map[string]*DungeonPlayerData),
		sessions:    make(map[string]*DungeonSessionData),
		invites:     make(map[string]*TeamInvite),
		playerNames: make(map[string]string),
	}
	svc.initDefaultDungeons()
	return svc
}

// SetRedis 设置 Redis 客户端
func (s *DungeonService) SetRedis(client *redis.Client) {
	s.redisClient = client
}

// initDefaultDungeons 初始化6个默认秘境
func (s *DungeonService) initDefaultDungeons() {
	dungeons := []*DungeonConfig{
		s.newTrialDungeon(),
		s.newBeastLairDungeon(),
		s.newDemonRealmDungeon(),
		s.newImmortalRuinsDungeon(),
		s.newDragonPalaceDungeon(),
		s.newChaosVoidDungeon(),
	}
	for _, d := range dungeons {
		s.dungeons[d.ID] = d
	}
	log.Info().Int("count", len(dungeons)).Msg("秘境默认数据初始化完成")
}

// ---------- 6个秘境定义 ----------

func (s *DungeonService) newTrialDungeon() *DungeonConfig {
	id := 1
	return &DungeonConfig{
		ID:              id,
		Name:            "新手试炼",
		Color1:          "#2d5016",
		Color2:          "#0d2818",
		RecommendLevel:  1,
		TotalFloors:     3,
		UnlockRealm:     0,
		UnlockCondition: "",
		DailyFree:       DungeonDailyFree,
		MaxBuyTimes:     DungeonMaxBuyTimes,
		BuyCostJade:     DungeonBuyCostJade,
		TeamMaxSize:     1,
		TimeLimitSec:    120,
		CompletionBonus: 0.5,
		EntryCost:       EntryCost{SpiritStones: 0, Stamina: 5},
		FirstClearReward: []RewardItem{{ID: 101, Name: "灵石", Count: 100}},
		Floors: []DungeonFloorConfig{
			{
				Floor: 1, IsBoss: false, Name: "试炼之路",
				Monsters: []MonsterConfig{{ID: 1, Name: "试炼木人", HP: 100, Atk: 10, Def: 5, Speed: 10, Level: 1}},
				Rewards:  FloorRewards{Exp: 50, Money: 10, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 5}}},
			},
			{
				Floor: 2, IsBoss: false, Name: "灵气试炼",
				Monsters: []MonsterConfig{{ID: 2, Name: "灵气傀儡", HP: 150, Atk: 15, Def: 8, Speed: 12, Level: 2}},
				Rewards:  FloorRewards{Exp: 80, Money: 20, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 10}}},
			},
			{
				Floor: 3, IsBoss: true, Name: "试炼守卫",
				Boss:   &MonsterConfig{ID: 3, Name: "试炼守卫", HP: 300, Atk: 25, Def: 12, Speed: 15, Level: 3},
				Monsters: []MonsterConfig{
					{ID: 101, Name: "试炼之灵", HP: 80, Atk: 8, Def: 4, Speed: 8, Level: 1},
				},
				Rewards: FloorRewards{Exp: 200, Money: 50, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 30}, {ID: 201, Name: "灵草", Count: 1}}},
			},
		},
	}
}

func (s *DungeonService) newBeastLairDungeon() *DungeonConfig {
	id := 2
	return &DungeonConfig{
		ID:              id,
		Name:            "妖兽巢穴",
		Color1:          "#6b1a1a",
		Color2:          "#2e0a0a",
		RecommendLevel:  10,
		TotalFloors:     6,
		UnlockRealm:     1,
		UnlockCondition: "修为达到练气中期",
		DailyFree:       DungeonDailyFree,
		MaxBuyTimes:     DungeonMaxBuyTimes,
		BuyCostJade:     DungeonBuyCostJade,
		TeamMaxSize:     TeamMaxMembers,
		TimeLimitSec:    180,
		CompletionBonus: 0.5,
		EntryCost:       EntryCost{SpiritStones: 100, Stamina: 10},
		FirstClearReward: []RewardItem{{ID: 102, Name: "妖兽内丹", Count: 1}},
		Floors: []DungeonFloorConfig{
			{Floor: 1, IsBoss: false, Name: "巢穴外围", Monsters: []MonsterConfig{{ID: 10, Name: "妖狼", HP: 300, Atk: 30, Def: 15, Speed: 20, Level: 8}}, Rewards: FloorRewards{Exp: 200, Money: 50, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 15}}}},
			{Floor: 2, IsBoss: false, Name: "密林深处", Monsters: []MonsterConfig{{ID: 11, Name: "妖狐", HP: 350, Atk: 40, Def: 12, Speed: 30, Level: 9}}, Rewards: FloorRewards{Exp: 280, Money: 70, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 20}}}},
			{Floor: 3, IsBoss: true, Name: "狼王巢穴", Monsters: []MonsterConfig{{ID: 10, Name: "妖狼", HP: 300, Atk: 30, Def: 15, Speed: 20, Level: 8}}, Boss: &MonsterConfig{ID: 12, Name: "狼王", HP: 800, Atk: 60, Def: 25, Speed: 25, Level: 10}, Rewards: FloorRewards{Exp: 600, Money: 150, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 50}, {ID: 202, Name: "妖兽皮毛", Count: 1}}}},
			{Floor: 4, IsBoss: false, Name: "山腹洞穴", Monsters: []MonsterConfig{{ID: 13, Name: "巨熊", HP: 500, Atk: 50, Def: 30, Speed: 15, Level: 10}}, Rewards: FloorRewards{Exp: 400, Money: 100, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 25}}}},
			{Floor: 5, IsBoss: false, Name: "毒瘴沼泽", Monsters: []MonsterConfig{{ID: 14, Name: "毒蛇", HP: 400, Atk: 55, Def: 10, Speed: 35, Level: 11}}, Rewards: FloorRewards{Exp: 500, Money: 120, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 30}}}},
			{Floor: 6, IsBoss: true, Name: "妖兽之王", Monsters: []MonsterConfig{{ID: 13, Name: "巨熊", HP: 500, Atk: 50, Def: 30, Speed: 15, Level: 10}, {ID: 14, Name: "毒蛇", HP: 400, Atk: 55, Def: 10, Speed: 35, Level: 11}}, Boss: &MonsterConfig{ID: 15, Name: "妖兽之王", HP: 1500, Atk: 80, Def: 40, Speed: 30, Level: 12}, Rewards: FloorRewards{Exp: 1200, Money: 300, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 100}, {ID: 203, Name: "妖兽内丹", Count: 1}}}},
		},
	}
}

func (s *DungeonService) newDemonRealmDungeon() *DungeonConfig {
	id := 3
	return &DungeonConfig{
		ID:              id,
		Name:            "魔道秘境",
		Color1:          "#4a2e6b",
		Color2:          "#1a0a2e",
		RecommendLevel:  25,
		TotalFloors:     9,
		UnlockRealm:     2,
		UnlockCondition: "修为达到筑基期",
		DailyFree:       DungeonDailyFree,
		MaxBuyTimes:     DungeonMaxBuyTimes,
		BuyCostJade:     DungeonBuyCostJade,
		TeamMaxSize:     TeamMaxMembers,
		TimeLimitSec:    300,
		CompletionBonus: 0.5,
		EntryCost:       EntryCost{SpiritStones: 500, Stamina: 20},
		FirstClearReward: []RewardItem{{ID: 301, Name: "魔晶碎片", Count: 1}},
		Floors: []DungeonFloorConfig{
			{Floor: 1, IsBoss: false, Name: "魔道入口", Monsters: []MonsterConfig{{ID: 20, Name: "魔化守卫", HP: 800, Atk: 70, Def: 35, Speed: 25, Level: 20}}, Rewards: FloorRewards{Exp: 800, Money: 200, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 40}}}},
			{Floor: 2, IsBoss: false, Name: "暗影回廊", Monsters: []MonsterConfig{{ID: 21, Name: "暗影刺客", HP: 600, Atk: 90, Def: 20, Speed: 50, Level: 21}}, Rewards: FloorRewards{Exp: 1000, Money: 250, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 50}}}},
			{Floor: 3, IsBoss: true, Name: "魔将厅", Monsters: []MonsterConfig{{ID: 20, Name: "魔化守卫", HP: 800, Atk: 70, Def: 35, Speed: 25, Level: 20}, {ID: 21, Name: "暗影刺客", HP: 600, Atk: 90, Def: 20, Speed: 50, Level: 21}}, Boss: &MonsterConfig{ID: 22, Name: "魔将", HP: 2500, Atk: 120, Def: 50, Speed: 35, Level: 22}, Rewards: FloorRewards{Exp: 2500, Money: 600, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 150}, {ID: 302, Name: "魔核", Count: 1}}}},
			{Floor: 4, IsBoss: false, Name: "幻境迷宫", Monsters: []MonsterConfig{{ID: 23, Name: "幻影魔", HP: 1000, Atk: 85, Def: 30, Speed: 40, Level: 22}}, Rewards: FloorRewards{Exp: 1400, Money: 350, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 60}}}},
			{Floor: 5, IsBoss: false, Name: "血池", Monsters: []MonsterConfig{{ID: 24, Name: "血魔", HP: 1500, Atk: 75, Def: 45, Speed: 20, Level: 23}}, Rewards: FloorRewards{Exp: 1800, Money: 400, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 70}}}},
			{Floor: 6, IsBoss: true, Name: "血魔统领", Monsters: []MonsterConfig{{ID: 24, Name: "血魔", HP: 1500, Atk: 75, Def: 45, Speed: 20, Level: 23}, {ID: 23, Name: "幻影魔", HP: 1000, Atk: 85, Def: 30, Speed: 40, Level: 22}}, Boss: &MonsterConfig{ID: 25, Name: "血魔统领", HP: 4000, Atk: 150, Def: 60, Speed: 30, Level: 24}, Rewards: FloorRewards{Exp: 4000, Money: 1000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 250}, {ID: 303, Name: "血玉", Count: 1}}}},
			{Floor: 7, IsBoss: false, Name: "魔渊之底", Monsters: []MonsterConfig{{ID: 26, Name: "深渊魔", HP: 2000, Atk: 100, Def: 50, Speed: 30, Level: 24}}, Rewards: FloorRewards{Exp: 2200, Money: 500, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 80}}}},
			{Floor: 8, IsBoss: false, Name: "魔气核心", Monsters: []MonsterConfig{{ID: 27, Name: "魔气聚合体", HP: 2500, Atk: 110, Def: 55, Speed: 25, Level: 25}}, Rewards: FloorRewards{Exp: 2600, Money: 600, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 100}}}},
			{Floor: 9, IsBoss: true, Name: "魔王降临", Monsters: []MonsterConfig{{ID: 26, Name: "深渊魔", HP: 2000, Atk: 100, Def: 50, Speed: 30, Level: 24}, {ID: 27, Name: "魔气聚合体", HP: 2500, Atk: 110, Def: 55, Speed: 25, Level: 25}}, Boss: &MonsterConfig{ID: 28, Name: "魔王", HP: 6000, Atk: 200, Def: 80, Speed: 40, Level: 26}, Rewards: FloorRewards{Exp: 6000, Money: 1500, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 400}, {ID: 304, Name: "魔王之心", Count: 1}}}},
		},
	}
}

func (s *DungeonService) newImmortalRuinsDungeon() *DungeonConfig {
	id := 4
	return &DungeonConfig{
		ID:              id,
		Name:            "仙府遗址",
		Color1:          "#1a3a5c",
		Color2:          "#0a1a2e",
		RecommendLevel:  40,
		TotalFloors:     10,
		UnlockRealm:     3,
		UnlockCondition: "修为达到结丹期",
		DailyFree:       DungeonDailyFree,
		MaxBuyTimes:     DungeonMaxBuyTimes,
		BuyCostJade:     DungeonBuyCostJade,
		TeamMaxSize:     TeamMaxMembers,
		TimeLimitSec:    360,
		CompletionBonus: 0.5,
		EntryCost:       EntryCost{SpiritStones: 2000, Stamina: 30},
		FirstClearReward: []RewardItem{{ID: 401, Name: "古宝碎片", Count: 1}},
		Floors: []DungeonFloorConfig{
			{Floor: 1, IsBoss: false, Name: "残垣断壁", Monsters: []MonsterConfig{{ID: 30, Name: "石傀儡", HP: 3000, Atk: 120, Def: 80, Speed: 15, Level: 35}}, Rewards: FloorRewards{Exp: 3000, Money: 500, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 80}}}},
			{Floor: 2, IsBoss: false, Name: "灵药园", Monsters: []MonsterConfig{{ID: 31, Name: "药园守卫", HP: 2800, Atk: 130, Def: 60, Speed: 30, Level: 36}}, Rewards: FloorRewards{Exp: 3500, Money: 600, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 100}, {ID: 402, Name: "灵草", Count: 1}}}},
			{Floor: 3, IsBoss: true, Name: "藏经阁", Monsters: []MonsterConfig{{ID: 30, Name: "石傀儡", HP: 3000, Atk: 120, Def: 80, Speed: 15, Level: 35}, {ID: 31, Name: "药园守卫", HP: 2800, Atk: 130, Def: 60, Speed: 30, Level: 36}}, Boss: &MonsterConfig{ID: 32, Name: "书灵", HP: 6000, Atk: 200, Def: 70, Speed: 40, Level: 37}, Rewards: FloorRewards{Exp: 8000, Money: 1500, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 300}, {ID: 403, Name: "功法残页", Count: 1}}}},
			{Floor: 4, IsBoss: false, Name: "炼丹房", Monsters: []MonsterConfig{{ID: 33, Name: "丹炉之灵", HP: 3200, Atk: 140, Def: 65, Speed: 25, Level: 37}}, Rewards: FloorRewards{Exp: 4000, Money: 700, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 120}}}},
			{Floor: 5, IsBoss: false, Name: "演武场", Monsters: []MonsterConfig{{ID: 34, Name: "武道残影", HP: 3500, Atk: 160, Def: 50, Speed: 45, Level: 38}}, Rewards: FloorRewards{Exp: 4500, Money: 800, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 150}}}},
			{Floor: 6, IsBoss: true, Name: "阵眼", Monsters: []MonsterConfig{{ID: 33, Name: "丹炉之灵", HP: 3200, Atk: 140, Def: 65, Speed: 25, Level: 37}, {ID: 34, Name: "武道残影", HP: 3500, Atk: 160, Def: 50, Speed: 45, Level: 38}}, Boss: &MonsterConfig{ID: 35, Name: "阵灵", HP: 8000, Atk: 250, Def: 90, Speed: 35, Level: 39}, Rewards: FloorRewards{Exp: 10000, Money: 2000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 400}, {ID: 404, Name: "阵旗", Count: 1}}}},
			{Floor: 7, IsBoss: false, Name: "主殿", Monsters: []MonsterConfig{{ID: 36, Name: "守卫之魂", HP: 4000, Atk: 170, Def: 70, Speed: 30, Level: 39}}, Rewards: FloorRewards{Exp: 5000, Money: 900, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 180}}}},
			{Floor: 8, IsBoss: false, Name: "后山禁地", Monsters: []MonsterConfig{{ID: 37, Name: "禁地守卫", HP: 4500, Atk: 180, Def: 75, Speed: 35, Level: 40}}, Rewards: FloorRewards{Exp: 5500, Money: 1000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 200}}}},
			{Floor: 9, IsBoss: false, Name: "秘境深处", Monsters: []MonsterConfig{{ID: 38, Name: "上古怨灵", HP: 5000, Atk: 190, Def: 65, Speed: 40, Level: 41}}, Rewards: FloorRewards{Exp: 6000, Money: 1100, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 250}}}},
			{Floor: 10, IsBoss: true, Name: "仙府之主", Monsters: []MonsterConfig{{ID: 37, Name: "禁地守卫", HP: 4500, Atk: 180, Def: 75, Speed: 35, Level: 40}, {ID: 38, Name: "上古怨灵", HP: 5000, Atk: 190, Def: 65, Speed: 40, Level: 41}}, Boss: &MonsterConfig{ID: 39, Name: "仙府残魂", HP: 12000, Atk: 300, Def: 100, Speed: 45, Level: 42}, Rewards: FloorRewards{Exp: 15000, Money: 3000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 600}, {ID: 405, Name: "仙府密宝", Count: 1}}}},
		},
	}
}

func (s *DungeonService) newDragonPalaceDungeon() *DungeonConfig {
	id := 5
	return &DungeonConfig{
		ID:              id,
		Name:            "龙宫探宝",
		Color1:          "#0a4a6b",
		Color2:          "#041a2e",
		RecommendLevel:  60,
		TotalFloors:     10,
		UnlockRealm:     4,
		UnlockCondition: "修为达到元婴期",
		DailyFree:       DungeonDailyFree,
		MaxBuyTimes:     DungeonMaxBuyTimes,
		BuyCostJade:     DungeonBuyCostJade,
		TeamMaxSize:     TeamMaxMembers,
		TimeLimitSec:    420,
		CompletionBonus: 0.5,
		EntryCost:       EntryCost{SpiritStones: 5000, Stamina: 40},
		FirstClearReward: []RewardItem{{ID: 501, Name: "龙鳞", Count: 1}},
		Floors: []DungeonFloorConfig{
			{Floor: 1, IsBoss: false, Name: "龙宫入口", Monsters: []MonsterConfig{{ID: 40, Name: "虾兵", HP: 5000, Atk: 200, Def: 100, Speed: 30, Level: 50}}, Rewards: FloorRewards{Exp: 6000, Money: 1000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 150}}}},
			{Floor: 2, IsBoss: false, Name: "珊瑚道", Monsters: []MonsterConfig{{ID: 41, Name: "蟹将", HP: 5500, Atk: 220, Def: 120, Speed: 25, Level: 51}}, Rewards: FloorRewards{Exp: 7000, Money: 1200, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 180}}}},
			{Floor: 3, IsBoss: true, Name: "夜叉殿", Monsters: []MonsterConfig{{ID: 40, Name: "虾兵", HP: 5000, Atk: 200, Def: 100, Speed: 30, Level: 50}, {ID: 41, Name: "蟹将", HP: 5500, Atk: 220, Def: 120, Speed: 25, Level: 51}}, Boss: &MonsterConfig{ID: 42, Name: "夜叉王", HP: 12000, Atk: 350, Def: 130, Speed: 40, Level: 52}, Rewards: FloorRewards{Exp: 16000, Money: 3000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 500}, {ID: 502, Name: "夜叉骨", Count: 1}}}},
			{Floor: 4, IsBoss: false, Name: "水晶宫外", Monsters: []MonsterConfig{{ID: 43, Name: "龙宫禁卫", HP: 6000, Atk: 250, Def: 110, Speed: 35, Level: 52}}, Rewards: FloorRewards{Exp: 8000, Money: 1400, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 200}}}},
			{Floor: 5, IsBoss: false, Name: "藏宝阁", Monsters: []MonsterConfig{{ID: 44, Name: "宝库守卫", HP: 6500, Atk: 260, Def: 115, Speed: 30, Level: 53}}, Rewards: FloorRewards{Exp: 9000, Money: 1600, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 250}, {ID: 503, Name: "珍珠", Count: 1}}}},
			{Floor: 6, IsBoss: true, Name: "龙将殿", Monsters: []MonsterConfig{{ID: 43, Name: "龙宫禁卫", HP: 6000, Atk: 250, Def: 110, Speed: 35, Level: 52}, {ID: 44, Name: "宝库守卫", HP: 6500, Atk: 260, Def: 115, Speed: 30, Level: 53}}, Boss: &MonsterConfig{ID: 45, Name: "龙将", HP: 16000, Atk: 400, Def: 150, Speed: 45, Level: 54}, Rewards: FloorRewards{Exp: 20000, Money: 4000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 700}, {ID: 504, Name: "龙角", Count: 1}}}},
			{Floor: 7, IsBoss: false, Name: "龙渊", Monsters: []MonsterConfig{{ID: 46, Name: "深渊水妖", HP: 7000, Atk: 280, Def: 100, Speed: 40, Level: 54}}, Rewards: FloorRewards{Exp: 10000, Money: 1800, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 300}}}},
			{Floor: 8, IsBoss: false, Name: "龙脉", Monsters: []MonsterConfig{{ID: 47, Name: "龙脉守护者", HP: 7500, Atk: 300, Def: 130, Speed: 35, Level: 55}}, Rewards: FloorRewards{Exp: 11000, Money: 2000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 350}}}},
			{Floor: 9, IsBoss: false, Name: "龙王殿前", Monsters: []MonsterConfig{{ID: 48, Name: "龙王亲卫", HP: 8000, Atk: 320, Def: 140, Speed: 40, Level: 56}}, Rewards: FloorRewards{Exp: 12000, Money: 2200, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 400}}}},
			{Floor: 10, IsBoss: true, Name: "龙王殿", Monsters: []MonsterConfig{{ID: 47, Name: "龙脉守护者", HP: 7500, Atk: 300, Def: 130, Speed: 35, Level: 55}, {ID: 48, Name: "龙王亲卫", HP: 8000, Atk: 320, Def: 140, Speed: 40, Level: 56}}, Boss: &MonsterConfig{ID: 49, Name: "龙王", HP: 20000, Atk: 500, Def: 180, Speed: 50, Level: 58}, Rewards: FloorRewards{Exp: 25000, Money: 5000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 1000}, {ID: 505, Name: "龙珠", Count: 1}}}},
		},
	}
}

func (s *DungeonService) newChaosVoidDungeon() *DungeonConfig {
	id := 6
	return &DungeonConfig{
		ID:              id,
		Name:            "混沌虚空",
		Color1:          "#2a0a4a",
		Color2:          "#0a001a",
		RecommendLevel:  80,
		TotalFloors:     10,
		UnlockRealm:     5,
		UnlockCondition: "修为达到化神期",
		DailyFree:       DungeonDailyFree,
		MaxBuyTimes:     DungeonMaxBuyTimes,
		BuyCostJade:     DungeonBuyCostJade,
		TeamMaxSize:     TeamMaxMembers,
		TimeLimitSec:    480,
		CompletionBonus: 0.5,
		EntryCost:       EntryCost{SpiritStones: 10000, Stamina: 50},
		FirstClearReward: []RewardItem{{ID: 601, Name: "混沌灵石", Count: 1}},
		Floors: []DungeonFloorConfig{
			{Floor: 1, IsBoss: false, Name: "虚空裂隙", Monsters: []MonsterConfig{{ID: 50, Name: "虚空兽", HP: 8000, Atk: 300, Def: 150, Speed: 40, Level: 70}}, Rewards: FloorRewards{Exp: 10000, Money: 2000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 300}}}},
			{Floor: 2, IsBoss: false, Name: "混沌迷雾", Monsters: []MonsterConfig{{ID: 51, Name: "迷雾之灵", HP: 7500, Atk: 350, Def: 120, Speed: 55, Level: 71}}, Rewards: FloorRewards{Exp: 12000, Money: 2400, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 350}}}},
			{Floor: 3, IsBoss: true, Name: "虚空守卫", Monsters: []MonsterConfig{{ID: 50, Name: "虚空兽", HP: 8000, Atk: 300, Def: 150, Speed: 40, Level: 70}, {ID: 51, Name: "迷雾之灵", HP: 7500, Atk: 350, Def: 120, Speed: 55, Level: 71}}, Boss: &MonsterConfig{ID: 52, Name: "虚空守卫", HP: 18000, Atk: 500, Def: 200, Speed: 50, Level: 72}, Rewards: FloorRewards{Exp: 28000, Money: 5000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 1000}, {ID: 602, Name: "虚空碎片", Count: 1}}}},
			{Floor: 4, IsBoss: false, Name: "时间乱流", Monsters: []MonsterConfig{{ID: 53, Name: "时间残影", HP: 9000, Atk: 380, Def: 140, Speed: 60, Level: 72}}, Rewards: FloorRewards{Exp: 14000, Money: 2800, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 400}}}},
			{Floor: 5, IsBoss: false, Name: "混沌核心", Monsters: []MonsterConfig{{ID: 54, Name: "混沌之灵", HP: 10000, Atk: 400, Def: 160, Speed: 45, Level: 73}}, Rewards: FloorRewards{Exp: 16000, Money: 3200, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 450}}}},
			{Floor: 6, IsBoss: true, Name: "虚空领主", Monsters: []MonsterConfig{{ID: 53, Name: "时间残影", HP: 9000, Atk: 380, Def: 140, Speed: 60, Level: 72}, {ID: 54, Name: "混沌之灵", HP: 10000, Atk: 400, Def: 160, Speed: 45, Level: 73}}, Boss: &MonsterConfig{ID: 55, Name: "虚空领主", HP: 24000, Atk: 600, Def: 220, Speed: 55, Level: 74}, Rewards: FloorRewards{Exp: 36000, Money: 6500, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 1500}, {ID: 603, Name: "领主核心", Count: 1}}}},
			{Floor: 7, IsBoss: false, Name: "毁灭之渊", Monsters: []MonsterConfig{{ID: 56, Name: "毁灭魔像", HP: 11000, Atk: 420, Def: 180, Speed: 35, Level: 74}}, Rewards: FloorRewards{Exp: 18000, Money: 3600, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 500}}}},
			{Floor: 8, IsBoss: false, Name: "创世之痕", Monsters: []MonsterConfig{{ID: 57, Name: "创世残影", HP: 12000, Atk: 450, Def: 170, Speed: 50, Level: 75}}, Rewards: FloorRewards{Exp: 20000, Money: 4000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 550}}}},
			{Floor: 9, IsBoss: false, Name: "虚无之境", Monsters: []MonsterConfig{{ID: 58, Name: "虚无之主", HP: 13000, Atk: 480, Def: 190, Speed: 45, Level: 76}}, Rewards: FloorRewards{Exp: 22000, Money: 4400, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 600}}}},
			{Floor: 10, IsBoss: true, Name: "混沌之主", Monsters: []MonsterConfig{{ID: 57, Name: "创世残影", HP: 12000, Atk: 450, Def: 170, Speed: 50, Level: 75}, {ID: 58, Name: "虚无之主", HP: 13000, Atk: 480, Def: 190, Speed: 45, Level: 76}}, Boss: &MonsterConfig{ID: 59, Name: "混沌之主", HP: 30000, Atk: 800, Def: 300, Speed: 60, Level: 78}, Rewards: FloorRewards{Exp: 50000, Money: 10000, Items: []RewardItem{{ID: 101, Name: "灵石", Count: 2000}, {ID: 604, Name: "混沌之源", Count: 1}}}},
		},
	}
}

// ---------- 玩家数据管理 ----------

// GetOrCreatePlayer 获取或创建玩家秘境数据
func (s *DungeonService) GetOrCreatePlayer(playerID string) *DungeonPlayerData {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getOrCreatePlayerLocked(playerID)
}

func (s *DungeonService) getOrCreatePlayerLocked(playerID string) *DungeonPlayerData {
	p, ok := s.players[playerID]
	if !ok {
		p = &DungeonPlayerData{
			PlayerID:       playerID,
			Progress:       make(map[int]*DungeonProgress),
			DailyFreeUsed:  make(map[string]int),
			DailyBuyUsed:   make(map[string]int),
			FirstCleared:   make(map[int]bool),
		}
		s.players[playerID] = p
	}
	return p
}

// SetPlayerName 设置玩家昵称(用于邀请显示)
func (s *DungeonService) SetPlayerName(playerID, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playerNames[playerID] = name
}

// ---------- 秘境查询 ----------

// GetDungeonList 获取秘境列表(含玩家进度)
func (s *DungeonService) GetDungeonList(playerID string) []*DungeonListEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	player := s.getOrCreatePlayerLocked(playerID)
	ids := make([]int, 0, len(s.dungeons))
	for id := range s.dungeons {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	list := make([]*DungeonListEntry, 0, len(ids))
	for _, id := range ids {
		d := s.dungeons[id]
		prog := player.Progress[id]

		currentFloor := 1
		if prog != nil {
			currentFloor = prog.HighestFloor + 1
			if currentFloor > d.TotalFloors {
				currentFloor = d.TotalFloors
			}
		}

		unlocked := true
		cond := d.UnlockCondition
		if d.UnlockRealm > 0 && playerID != "" {
			// 解锁判定依赖外部传入, 默认已解锁
			// 实际应由调用方根据玩家境界判断
		}

		list = append(list, &DungeonListEntry{
			ID:              d.ID,
			Name:            d.Name,
			Color1:          d.Color1,
			Color2:          d.Color2,
			RecommendLevel:  d.RecommendLevel,
			TotalFloors:     d.TotalFloors,
			CurrentFloor:    currentFloor,
			Unlocked:        unlocked,
			UnlockCondition: cond,
		})
	}
	return list
}

// GetDungeonConfig 获取秘境配置
func (s *DungeonService) GetDungeonConfig(dungeonID int) *DungeonConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dungeons[dungeonID]
}

// GetDungeonDetail 获取秘境详情(含玩家进度)
func (s *DungeonService) GetDungeonDetail(playerID string, dungeonID int) *DungeonDetail {
	s.mu.RLock()
	defer s.mu.RUnlock()

d, ok := s.dungeons[dungeonID]
	if !ok {
		return nil
	}

	player := s.getOrCreatePlayerLocked(playerID)
	prog := player.Progress[dungeonID]

	highestFloor := 0
	bestStars := 0
	completed := false
	firstCleared := false
	if prog != nil {
		highestFloor = prog.HighestFloor
		bestStars = prog.Stars
		completed = prog.Completed
		firstCleared = player.FirstCleared[dungeonID]
	}

	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%d:%s", dungeonID, today)
	dailyUsed := player.DailyFreeUsed[key]
	buyUsed := player.DailyBuyUsed[key]

	return &DungeonDetail{
		ID:              d.ID,
		Name:            d.Name,
		Color1:          d.Color1,
		Color2:          d.Color2,
		RecommendLevel:  d.RecommendLevel,
		TotalFloors:     d.TotalFloors,
		EntryCost:       d.EntryCost,
		DailyFree:       d.DailyFree,
		DailyUsed:       dailyUsed,
		BuyUsed:         buyUsed,
		MaxBuyTimes:     d.MaxBuyTimes,
		BuyCostJade:     d.BuyCostJade,
		HighestFloor:    highestFloor,
		BestStars:       bestStars,
		Completed:       completed,
		FirstCleared:    firstCleared,
		UnlockCondition: d.UnlockCondition,
		Floors:          d.Floors,
	}
}

// ---------- 进入秘境 ----------

// CanEnter 检查能否进入秘境
func (s *DungeonService) CanEnter(playerID string, dungeonID int) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

d, ok := s.dungeons[dungeonID]
	if !ok {
		return false, "秘境不存在"
	}

	// 检查会话冲突
	if s.sessions[playerID] != nil {
		return false, "已有进行中的秘境, 请先完成或退出"
	}

	player := s.getOrCreatePlayerLocked(playerID)
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%d:%s", dungeonID, today)

	if player.DailyFreeUsed[key] >= d.DailyFree {
		if player.DailyBuyUsed[key] >= d.MaxBuyTimes {
			return false, "今日挑战次数已达上限"
		}
	}

	return true, ""
}

// EnterDungeon 进入秘境
func (s *DungeonService) EnterDungeon(playerID string, dungeonID int, team []string) (*DungeonSessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

d, ok := s.dungeons[dungeonID]
	if !ok {
		return nil, fmt.Errorf("秘境不存在")
	}

	// 检查会话冲突
	if existing := s.sessions[playerID]; existing != nil {
		return nil, fmt.Errorf("已有进行中的秘境: %d", existing.DungeonID)
	}

	// 构建队伍
	if len(team) == 0 {
		team = []string{playerID}
	}
	if len(team) > TeamMaxMembers {
		team = team[:TeamMaxMembers]
	}

	// 检查每日次数
	player := s.getOrCreatePlayerLocked(playerID)
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%d:%s", dungeonID, today)

	usingBuy := false
	if player.DailyFreeUsed[key] >= d.DailyFree {
		usingBuy = true
		if player.DailyBuyUsed[key] >= d.MaxBuyTimes {
			return nil, fmt.Errorf("今日挑战次数已达上限")
		}
		player.DailyBuyUsed[key]++
	} else {
		player.DailyFreeUsed[key]++
	}

	// 创建会话
	session := &DungeonSessionData{
		PlayerID:     playerID,
		DungeonID:    dungeonID,
		CurrentFloor: 1,
		MaxFloor:     d.TotalFloors,
		Team:         team,
		StartedAt:    time.Now().Unix(),
		FloorTimes:   make([]int, 0),
		Completed:    false,
		Failed:       false,
		State:        "exploring",
		TotalTimeSec: 0,
		Rating:       0,
	}
	s.sessions[playerID] = session

	log.Info().Str("player_id", playerID).Int("dungeon_id", dungeonID).
		Strs("team", team).Bool("using_buy", usingBuy).Msg("进入秘境")

	return session, nil
}

// ---------- 战斗 ----------

// FightFloor 挑战当前层, 返回战斗结果
// 使用模拟战斗逻辑(可替换为真实战斗引擎)
func (s *DungeonService) FightFloor(playerID string) (*DungeonFightResult, error) {
	s.mu.Lock()

	session, ok := s.sessions[playerID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("未进入秘境")
	}
	if session.Completed || session.Failed {
		s.mu.Unlock()
		return nil, fmt.Errorf("秘境已结束")
	}

	d, ok := s.dungeons[session.DungeonID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("秘境配置不存在")
	}

	floorIdx := session.CurrentFloor - 1
	if floorIdx < 0 || floorIdx >= len(d.Floors) {
		s.mu.Unlock()
		return nil, fmt.Errorf("层数异常")
	}

	floor := d.Floors[floorIdx]
	session.State = "fighting"

	// 复制一份用于后续计算(避免长时间持锁)
	dCopy := *d
	sessionCopy := *session
	s.mu.Unlock()

	// 模拟战斗耗时(2~6秒)
	fightTime := 2 + rand.Intn(5)
	time.Sleep(time.Duration(fightTime) * time.Millisecond) // 仅模拟短暂延迟

	// 模拟战斗胜负(随层数增加难度)
	winChance := 0.85 - float64(floorIdx)*0.03
	if winChance < 0.4 {
		winChance = 0.4
	}
	if floor.IsBoss {
		winChance -= 0.15
	}
	win := rand.Float64() < winChance

	// 构建日志
	logs := s.buildFightLogs(&floor, &dCopy, win)

	// 生成结果
	result := &DungeonFightResult{
		Win:         win,
		Floor:       sessionCopy.CurrentFloor,
		IsBoss:      floor.IsBoss,
		Completed:   false,
		TimeUsedSec: fightTime,
		Logs:        logs,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 重新获取会话(防止并发修改)
	currentSession := s.sessions[playerID]
	if currentSession == nil || currentSession.CurrentFloor != sessionCopy.CurrentFloor {
		return nil, fmt.Errorf("会话状态已变更")
	}

	currentSession.TotalTimeSec += fightTime
	currentSession.FloorTimes = append(currentSession.FloorTimes, fightTime)

	if win {
		result.Rewards = &floor.Rewards
		currentSession.CurrentFloor++
		if currentSession.CurrentFloor > dCopy.TotalFloors {
			currentSession.Completed = true
			currentSession.State = "done"
			result.Completed = true

			// 计算评级
			rating := s.calculateRating(currentSession.TotalTimeSec, &dCopy)
			currentSession.Rating = rating
			result.Rating = rating

			// 更新玩家进度
			s.updateProgress(playerID, &dCopy, rating, currentSession.TotalTimeSec)
		} else {
			currentSession.State = "exploring"
		}
	} else {
		currentSession.Failed = true
		currentSession.State = "done"
		result.Completed = true
	}

	return result, nil
}

// buildFightLogs 构建模拟战斗日志
func (s *DungeonService) buildFightLogs(floor *DungeonFloorConfig, d *DungeonConfig, win bool) []string {
	logs := make([]string, 0)
	logs = append(logs, fmt.Sprintf("进入 %s 第 %d 层: %s", d.Name, floor.Floor, floor.Name))

	for _, m := range floor.Monsters {
		logs = append(logs, fmt.Sprintf("遭遇 %s(Lv.%d)", m.Name, m.Level))
		logs = append(logs, fmt.Sprintf("你对 %s 发动攻击, 造成 %d 点伤害", m.Name, int64(float64(m.HP)*0.3)))
		logs = append(logs, fmt.Sprintf("%s 反击, 你受到 %d 点伤害", m.Name, int64(float64(m.Atk)*0.8)))
		logs = append(logs, fmt.Sprintf("%s 被击败!", m.Name))
	}

	if floor.IsBoss && floor.Boss != nil {
		b := floor.Boss
		logs = append(logs, fmt.Sprintf("--- Boss战: %s(Lv.%d) ---", b.Name, b.Level))
		logs = append(logs, fmt.Sprintf("你使用强力技能, 造成 %d 点伤害", int64(float64(b.HP)*0.15)))
		logs = append(logs, fmt.Sprintf("%s 发动大招, 你受到 %d 点伤害", b.Name, int64(float64(b.Atk)*1.2)))
		logs = append(logs, fmt.Sprintf("你抓住破绽, 给予致命一击!"))
	}

	if win {
		logs = append(logs, fmt.Sprintf("第 %d 层通过!", floor.Floor))
		if floor.Rewards.Exp > 0 {
			logs = append(logs, fmt.Sprintf("获得修为 +%d", floor.Rewards.Exp))
		}
		if floor.Rewards.Money > 0 {
			logs = append(logs, fmt.Sprintf("获得灵石 +%d", floor.Rewards.Money))
		}
		for _, item := range floor.Rewards.Items {
			logs = append(logs, fmt.Sprintf("获得 %s x%d", item.Name, item.Count))
		}
	} else {
		logs = append(logs, "挑战失败...")
	}

	return logs
}

// calculateRating 根据通关时间计算星级(1-3)
func (s *DungeonService) calculateRating(totalTimeSec int, d *DungeonConfig) int {
	limit := d.TimeLimitSec
	if limit <= 0 {
		return 3
	}

	ratio := float64(totalTimeSec) / float64(limit)
	switch {
	case ratio <= ThreeStarTimeRatio:
		return 3
	case ratio <= TwoStarTimeRatio:
		return 2
	default:
		return 1
	}
}

// updateProgress 更新玩家通关进度
func (s *DungeonService) updateProgress(playerID string, d *DungeonConfig, rating int, timeSec int) {
	player := s.getOrCreatePlayerLocked(playerID)
	prog, ok := player.Progress[d.ID]
	if !ok {
		prog = &DungeonProgress{DungeonID: d.ID}
		player.Progress[d.ID] = prog
	}

	prog.HighestFloor = d.TotalFloors
	prog.Completed = true
	if rating > prog.Stars {
		prog.Stars = rating
	}
	if prog.BestTime == 0 || timeSec < prog.BestTime {
		prog.BestTime = timeSec
	}
}

// ---------- 退出秘境 ----------

// ExitDungeon 退出秘境(放弃进度)
func (s *DungeonService) ExitDungeon(playerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[playerID]
	if !ok {
		return fmt.Errorf("未进入秘境")
	}

	delete(s.sessions, playerID)
	log.Info().Str("player_id", playerID).Int("dungeon_id", session.DungeonID).
		Int("current_floor", session.CurrentFloor).Msg("退出秘境")

	return nil
}

// ---------- 领取奖励 ----------

// ClaimFloorRewards 领取当前已完成层的奖励(由战斗胜利自动发放)
// 实际调用方(handler)负责通知 player/cultivation 服务
func (s *DungeonService) ClaimFloorRewards(playerID string) (*FloorRewardClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[playerID]
	if !ok {
		return nil, fmt.Errorf("未进入秘境")
	}

	d, ok := s.dungeons[session.DungeonID]
	if !ok {
		return nil, fmt.Errorf("秘境配置不存在")
	}

	if !session.Completed && !session.Failed {
		return nil, fmt.Errorf("秘境尚未结束")
	}

	// 累计所有已通关层的奖励
	total := &FloorRewardClaim{}
	clearedCount := len(session.FloorTimes)
	if clearedCount > len(d.Floors) {
		clearedCount = len(d.Floors)
	}

	for i := 0; i < clearedCount; i++ {
		floor := d.Floors[i]
		total.Exp += floor.Rewards.Exp
		total.Money += floor.Rewards.Money
		total.Items = append(total.Items, floor.Rewards.Items...)
	}

	// 通关加成(首次通关)
	if session.Completed && session.Rating > 0 {
		bonusExp := int64(math.Round(float64(total.Exp) * d.CompletionBonus))
		bonusMoney := int64(math.Round(float64(total.Money) * d.CompletionBonus))
		total.Exp += bonusExp
		total.Money += bonusMoney

		// 首次通关额外奖励
		player := s.getOrCreatePlayerLocked(playerID)
		if !player.FirstCleared[session.DungeonID] {
			player.FirstCleared[session.DungeonID] = true
			total.Items = append(total.Items, d.FirstClearReward...)
		}
	}

	// 合并同名物品
	total.Items = mergeRewardItems(total.Items)

	return total, nil
}

// ClearSession 清理会话(奖励领取后调用)
func (s *DungeonService) ClearSession(playerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, playerID)
}

// GetSession 获取当前会话
func (s *DungeonService) GetSession(playerID string) *DungeonSessionData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[playerID]
}

// mergeRewardItems 合并同名物品
func mergeRewardItems(items []RewardItem) []RewardItem {
	if len(items) <= 1 {
		return items
	}
	merged := make(map[int]*RewardItem)
	ordered := make([]int, 0)
	for i := range items {
		item := items[i]
		if existing, ok := merged[item.ID]; ok {
			existing.Count += item.Count
		} else {
			merged[item.ID] = &RewardItem{ID: item.ID, Name: item.Name, Count: item.Count}
			ordered = append(ordered, item.ID)
		}
	}
	result := make([]RewardItem, 0, len(ordered))
	for _, id := range ordered {
		result = append(result, *merged[id])
	}
	return result
}

// ---------- 组队系统 ----------

// CreateInvite 创建组队邀请
func (s *DungeonService) CreateInvite(hostID, hostName string, dungeonID int, targetID string) (*TeamInvite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

_, ok := s.dungeons[dungeonID]
	if !ok {
		return nil, fmt.Errorf("秘境不存在")
	}

	// 检查是否有已有邀请
	for _, inv := range s.invites {
		if inv.HostID == hostID && inv.DungeonID == dungeonID && inv.Status == "pending" {
			// 已有邀请, 直接返回
			return inv, nil
		}
	}

	now := time.Now()
	invite := &TeamInvite{
		ID:        uuid.New().String(),
		DungeonID: dungeonID,
		HostID:    hostID,
		HostName:  hostName,
		TargetID:  targetID,
		Status:    "pending",
		CreatedAt: now.Unix(),
		ExpiresAt: now.Add(InviteExpireSec * time.Second).Unix(),
		Members:   []string{hostID},
	}

	// 如果targetID为空, 则视为公开邀请
	if targetID == "" {
		invite.TargetID = ""
	}

	s.invites[invite.ID] = invite
	log.Info().Str("invite_id", invite.ID).Str("host", hostID).
		Int("dungeon_id", dungeonID).Msg("创建组队邀请")

	return invite, nil
}

// AcceptInvite 接受组队邀请
func (s *DungeonService) AcceptInvite(playerID, inviteID string) (*TeamInvite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invites[inviteID]
	if !ok {
		return nil, fmt.Errorf("邀请不存在")
	}

	if inv.Status != "pending" {
		return nil, fmt.Errorf("邀请已%s", inv.Status)
	}

	if time.Now().Unix() > inv.ExpiresAt {
		inv.Status = "expired"
		return nil, fmt.Errorf("邀请已过期")
	}

	// 检查队伍是否已满
	if len(inv.Members) >= TeamMaxMembers {
		inv.Status = "expired"
		return nil, fmt.Errorf("队伍已满")
	}

	// 检查玩家是否已在其他队伍中
	for _, m := range inv.Members {
		if m == playerID {
			return nil, fmt.Errorf("已在队伍中")
		}
	}

	// 检查目标限制
	if inv.TargetID != "" && inv.TargetID != playerID {
		return nil, fmt.Errorf("该邀请不是发给你的")
	}

	inv.Members = append(inv.Members, playerID)
	inv.Status = "accepted"

	log.Info().Str("invite_id", inviteID).Str("player", playerID).
		Msg("接受组队邀请")

	return inv, nil
}

// DeclineInvite 拒绝组队邀请
func (s *DungeonService) DeclineInvite(playerID, inviteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invites[inviteID]
	if !ok {
		return fmt.Errorf("邀请不存在")
	}

	if inv.Status != "pending" {
		return fmt.Errorf("邀请已%s", inv.Status)
	}

	inv.Status = "declined"
	log.Info().Str("invite_id", inviteID).Str("player", playerID).Msg("拒绝组队邀请")
	return nil
}

// GetPendingInvites 获取玩家的待处理邀请
func (s *DungeonService) GetPendingInvites(playerID string) []*TeamInvite {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().Unix()
	result := make([]*TeamInvite, 0)
	for _, inv := range s.invites {
		if inv.Status != "pending" {
			continue
		}
		if now > inv.ExpiresAt {
			inv.Status = "expired"
			continue
		}
		if inv.TargetID == "" || inv.TargetID == playerID {
			result = append(result, inv)
		}
	}
	return result
}

// ---------- 每日重置 ----------

// ResetDaily 重置玩家每日次数(由定时任务或凌晨调用)
func (s *DungeonService) ResetDaily(playerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, ok := s.players[playerID]
	if !ok {
		return
	}
	player.DailyFreeUsed = make(map[string]int)
	player.DailyBuyUsed = make(map[string]int)
	log.Info().Str("player_id", playerID).Msg("秘境每日次数已重置")
}

// ResetAllDaily 重置所有玩家每日次数
func (s *DungeonService) ResetAllDaily() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, player := range s.players {
		player.DailyFreeUsed = make(map[string]int)
		player.DailyBuyUsed = make(map[string]int)
	}
	log.Info().Int("count", len(s.players)).Msg("所有玩家秘境每日次数已重置")
}

// CleanExpiredInvites 清理过期邀请
func (s *DungeonService) CleanExpiredInvites() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	for _, inv := range s.invites {
		if inv.Status == "pending" && now > inv.ExpiresAt {
			inv.Status = "expired"
		}
	}
}

// ---------- 持久化 ----------

// Save 保存秘境数据到 Redis
func (s *DungeonService) Save(ctx interface{}) error {
	if s.redisClient == nil {
		return nil
	}
	// 持久化实现略(与项目其他服务一致)
	return nil
}

// Load 从 Redis 加载秘境数据
func (s *DungeonService) Load(ctx interface{}) error {
	if s.redisClient == nil {
		return nil
	}
	return nil
}

// ---------- 工具方法 ----------

// GetDungeonIDs 获取所有秘境ID(排序)
func (s *DungeonService) GetDungeonIDs() []int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]int, 0, len(s.dungeons))
	for id := range s.dungeons {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}
