package service

import (
	"context"
	"encoding/json"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type TowerRankingEntry struct {
	PlayerID     uint64 `json:"player_id"`
	Nickname     string `json:"nickname"`
	HighestFloor int    `json:"highest_floor"`
	Rank         int    `json:"rank"`
	BestTimeSec  int    `json:"best_time_sec"`
	RealmName    string `json:"realm_name"`
}

type TowerPlayerData struct {
	PlayerID          uint64
	HighestFloor      int
	BestTimeSec       int
	DailyFreeUsed     int
	DailyBuyUsed      int
	LastDailyDate     string
	ClaimedMilestones []int
	TitlesEarned      []string
}

type TowerSessionData struct {
	PlayerID     uint64
	CurrentFloor int
	StartedAt    int64
	State        string
	Completed    bool
	Failed       bool
	InSession      bool `json:"in_session"`
	RemainingTime int  `json:"remaining_time"`
}

type MilestoneRewardConfig struct {
	Floor       int
	Exp         int64
	Money       int64
	Items       []int
	Title       string
	Description string
}

type TowerFloorConfig struct {
	Floor       int    `json:"floor"`
	DemonType   string `json:"demon_type"`
	IsBoss      bool   `json:"is_boss"`
	Name        string `json:"name"`
	MonsterAtk  int64  `json:"monster_atk"`
	MonsterHP   int64  `json:"monster_hp"`
	RewardExp   int64  `json:"reward_exp"`
	RewardMoney int64  `json:"reward_money"`
	RewardItems []int  `json:"reward_items"`
	Description  string `json:"description"`
	MonsterDef   int64  `json:"monster_def"`
	MonsterSpeed int64    `json:"monster_speed"`
	GreedCost    int64  `json:"greed_cost"`
	WrathMultiplier float64 `json:"wrath_multiplier"`
	IgnorPenalty    float64 `json:"ignor_penalty"`
	IgnorQuestion  string  `json:"ignor_question"`
	IgnorAnswer    string     `json:"ignor_answer"`
	IgnorChoices   []string `json:"ignor_choices"`
	RewardTitle    string  `json:"reward_title"`
	TimeLimitSec   int     `json:"time_limit_sec"`
	MonsterCrit    float64 `json:"monster_crit"`
}

type TowerFightResult struct {
	Floor       int    `json:"floor"`
	DemonType   string `json:"demon_type"`
	IsBoss      bool   `json:"is_boss"`
	Win         bool   `json:"win"`
	TimeUsedSec int    `json:"time_used_sec"`
	RewardExp   int64  `json:"reward_exp"`
	RewardMoney int64  `json:"reward_money"`
	RewardItems []int  `json:"reward_items"`
	RewardTitle string `json:"reward_title"`
}

type TowerService struct {
	mu         sync.RWMutex
	players    map[uint64]*TowerPlayerData
	sessions   map[uint64]*TowerSessionData
	milestones map[int]*MilestoneRewardConfig
	rankings   []*TowerRankingEntry
	dirty      bool
	redisClient *redis.Client // Redis 持久化（可选）
}

func newTowerServiceInternal() *TowerService {
	ts := &TowerService{
		players:    make(map[uint64]*TowerPlayerData),
		sessions:   make(map[uint64]*TowerSessionData),
		milestones: make(map[int]*MilestoneRewardConfig),
	}
	ts.initMilestones()
	return ts
}

func (ts *TowerService) initMilestones() {
	ms := []MilestoneRewardConfig{
		{10, 5000, 2000, []int{301}, "心魔初破", "突破10层"},
		{20, 15000, 5000, []int{302}, "幻境行者", "突破20层"},
		{30, 30000, 10000, []int{303}, "明心见性", "突破30层"},
		{50, 100000, 40000, []int{305}, "心魔克星", "突破50层"},
		{100, 2000000, 1000000, []int{310}, "心魔至尊", "登顶100层"},
	}
	for i := range ms {
		ts.milestones[ms[i].Floor] = &ms[i]
	}
}

func (ts *TowerService) GetOrCreatePlayer(pid uint64) *TowerPlayerData {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	p, ok := ts.players[pid]
	if !ok {
		p = &TowerPlayerData{PlayerID: pid, LastDailyDate: time.Now().Format("2006-01-02")}
		ts.players[pid] = p
	}
	today := time.Now().Format("2006-01-02")
	if p.LastDailyDate != today {
		p.DailyFreeUsed, p.DailyBuyUsed, p.LastDailyDate = 0, 0, today
	}
	return p
}

func (ts *TowerService) EnterTower(pid uint64) (bool, string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if s, ok := ts.sessions[pid]; ok && s.State == "fighting" {
		return false, "已有进行中的挑战"
	}
	ts.sessions[pid] = &TowerSessionData{PlayerID: pid, CurrentFloor: 1, StartedAt: time.Now().Unix(), State: "fighting"}
	return true, ""
}

func (ts *TowerService) GetSession(pid uint64) *TowerSessionData {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.sessions[pid]
}

func (ts *TowerService) FightFloor(pid uint64, config *TowerFloorConfig, floor int, timeUsedSec int) (*TowerFightResult, string) {
	ts.mu.RLock()
	sess := ts.sessions[pid]
	ts.mu.RUnlock()
	if sess == nil || sess.State != "fighting" {
		return nil, "未进入心魔塔"
	}
	if sess.Completed || sess.Failed {
		return nil, "挑战已结束"
	}
	if floor != sess.CurrentFloor {
		return nil, "层数不匹配"
	}

	result := &TowerFightResult{Floor: floor, DemonType: config.DemonType, IsBoss: config.IsBoss, Win: true, TimeUsedSec: timeUsedSec}

	switch config.DemonType {
	case "greed":
		result.Win = rand.Float64() > 0.3
	case "wrath":
		result.Win = rand.Float64() > 0.25
	case "ignor":
		result.Win = rand.Float64() > 0.2
	case "slay":
		pwr := float64(1000 + floor*50)
		mwr := float64(config.MonsterHP/10 + config.MonsterAtk)
		result.Win = rand.Float64() < pwr/(pwr+mwr)
	}

	if !result.Win {
		ts.mu.Lock()
		sess.Failed, sess.State = true, "idle"
		ts.mu.Unlock()
		return result, ""
	}

	result.RewardExp = config.RewardExp
	result.RewardMoney = config.RewardMoney
	if len(config.RewardItems) > 0 {
		result.RewardItems = append(result.RewardItems, config.RewardItems[0])
	}

	ts.mu.Lock()
	p := ts.players[pid]
	if config.IsBoss {
		if ms, ok := ts.milestones[floor]; ok {
			claimed := false
			for _, c := range p.ClaimedMilestones {
				if c == floor { claimed = true; break }
			}
			if !claimed {
				result.RewardTitle = ms.Title
				p.ClaimedMilestones = append(p.ClaimedMilestones, floor)
			}
		}
	}
	if floor > p.HighestFloor {
		p.HighestFloor = floor
	}
	sess.CurrentFloor++
	sess.Completed = floor >= 100
	ts.dirty = true
	ts.mu.Unlock()

	return result, ""
}

func (ts *TowerService) GetFloorConfig(floor int) *TowerFloorConfig {
	demonTypes := []string{"greed", "wrath", "ignor", "slay"}
	cfg := &TowerFloorConfig{
		Floor:       floor,
		DemonType:   demonTypes[rand.Intn(len(demonTypes))],
		IsBoss:      floor%5 == 0,
		RewardExp:   int64(floor * 100),
		RewardMoney: int64(floor * 50),
	}
	if cfg.IsBoss {
		cfg.RewardExp *= 5
		cfg.RewardMoney *= 5
		cfg.RewardItems = []int{300 + floor/5}
	}
	return cfg
}

func (ts *TowerService) GetRankings() []*TowerRankingEntry {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	if ts.dirty { ts.sortRankingsLocked() }
	return ts.rankings
}

func (ts *TowerService) sortRankingsLocked() {
	ts.rankings = make([]*TowerRankingEntry, 0, len(ts.players))
	for _, p := range ts.players {
		ts.rankings = append(ts.rankings, &TowerRankingEntry{
			PlayerID: p.PlayerID, HighestFloor: p.HighestFloor, BestTimeSec: p.BestTimeSec,
		})
	}
	sort.Slice(ts.rankings, func(i, j int) bool {
		if ts.rankings[i].HighestFloor != ts.rankings[j].HighestFloor {
			return ts.rankings[i].HighestFloor > ts.rankings[j].HighestFloor
		}
		return ts.rankings[i].BestTimeSec < ts.rankings[j].BestTimeSec
	})
	ts.dirty = false
}

func (ts *TowerService) ResetDaily(pid uint64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if p, ok := ts.players[pid]; ok {
		p.DailyFreeUsed, p.DailyBuyUsed = 0, 0
		p.LastDailyDate = time.Now().Format("2006-01-02")
	}
}

func NewTowerService(_, _ string) *TowerService { return newTowerServiceInternal() }

func (ts *TowerService) CanEnter(pid uint64) (bool, string) {
	p := ts.GetOrCreatePlayer(pid)
	if p.DailyFreeUsed >= 5 { return false, "今日挑战次数已用尽" }
	if _, ok := ts.sessions[pid]; ok {
		if ts.sessions[pid].State == "fighting" { return false, "已有进行中的挑战" }
	}
	return true, ""
}

func (ts *TowerService) GetBuyCost() int64 { return 500 }

func (ts *TowerService) UseDailyAttempt(pid uint64, useBuy bool) error {
	p := ts.GetOrCreatePlayer(pid)
	p.DailyFreeUsed++
	return nil
}

func (ts *TowerService) GetStatus(pid uint64) *TowerSessionData {
	return ts.GetSession(pid)
}

// SetRedis 设置 Redis 客户端用于持久化
func (ts *TowerService) SetRedis(client *redis.Client) {
	ts.redisClient = client
}

// Save 将心魔塔数据持久化到 Redis
func (ts *TowerService) Save(ctx context.Context) error {
	if ts.redisClient == nil {
		return nil
	}
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	pipe := ts.redisClient.Pipeline()

	// 保存玩家数据
	playersData := make(map[string]string)
	for id, p := range ts.players {
		data, err := json.Marshal(p)
		if err != nil {
			continue
		}
		playersData[string(rune(id))] = string(data)
	}
	if len(playersData) > 0 {
		data, _ := json.Marshal(ts.players)
		pipe.Set(ctx, "tower:players", string(data), 30*24*time.Hour)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Load 从 Redis 恢复心魔塔数据
func (ts *TowerService) Load(ctx context.Context) error {
	if ts.redisClient == nil {
		return nil
	}

	data, err := ts.redisClient.Get(ctx, "tower:players").Result()
	if err != nil || data == "" {
		return nil
	}

	var players map[uint64]*TowerPlayerData
	if err := json.Unmarshal([]byte(data), &players); err != nil {
		return err
	}

	ts.mu.Lock()
	ts.players = players
	ts.dirty = true
	ts.mu.Unlock()

	return nil
}

// StartAutoSave 启动定期自动保存
func (ts *TowerService) StartAutoSave(ctx context.Context) {
	if ts.redisClient == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = ts.Save(ctx)
			case <-ctx.Done():
				_ = ts.Save(context.Background())
				return
			}
		}
	}()
}
