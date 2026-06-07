package service

import (
	"math/rand"
	"sync"
	"time"

	"cultivation-game/services/combat/internal/model"
)

// Rank 段位
type Rank string

const (
	RankBronze   Rank = "bronze"   // 青铜
	RankSilver   Rank = "silver"   // 白银
	RankGold     Rank = "gold"     // 黄金
	RankPlatinum Rank = "platinum" // 铂金
	RankDiamond  Rank = "diamond"  // 钻石
	RankLegend   Rank = "legend"   // 传说
)

// RankOrder 段位顺序(用于比较)
var RankOrder = map[Rank]int{
	RankBronze:   0,
	RankSilver:   1,
	RankGold:     2,
	RankPlatinum: 3,
	RankDiamond:  4,
	RankLegend:   5,
}

// PlayerProfile 玩家竞技信息
type PlayerProfile struct {
	PlayerID   string  `json:"player_id"`
	Name       string  `json:"name"`
	Rank       Rank    `json:"rank"`
	Score      int     `json:"score"`      // 段位分
	Level      int     `json:"level"`
	PowerLevel float64 `json:"power_level"` // 战力评估
	IsSearching bool   `json:"is_searching"`
	EnteredAt  time.Time `json:"entered_at"`
}

// MatchResult 匹配结果
type MatchResult struct {
	Player1 *PlayerProfile `json:"player1"`
	Player2 *PlayerProfile `json:"player2"`
	Team1   []*model.Fighter `json:"team1"`
	Team2   []*model.Fighter `json:"team2"`
}

// MatchmakingService 匹配服务
type MatchmakingService struct {
	mu         sync.RWMutex
	queue      []*PlayerProfile
	rankRange  int           // 匹配段位范围
	timeout    time.Duration // 匹配超时
}

// NewMatchmakingService 创建匹配服务
func NewMatchmakingService(rankRange int, timeout time.Duration) *MatchmakingService {
	return &MatchmakingService{
		queue:     make([]*PlayerProfile, 0),
		rankRange: rankRange,
		timeout:   timeout,
	}
}

// Enqueue 加入匹配队列
func (s *MatchmakingService) Enqueue(player *PlayerProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player.IsSearching = true
	player.EnteredAt = time.Now()
	s.queue = append(s.queue, player)
}

// Dequeue 移出匹配队列
func (s *MatchmakingService) Dequeue(playerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.queue {
		if p.PlayerID == playerID {
			p.IsSearching = false
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
			return
		}
	}
}

// Match 执行一轮匹配
//
// 匹配规则:
//  1. 段位差在 rankRange 以内
//  2. 等待时间越长, 匹配范围越大
//  3. 优先匹配战力相近的对手
func (s *MatchmakingService) Match() []*MatchResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) < 2 {
		return nil
	}

	results := make([]*MatchResult, 0)
	matched := make(map[string]bool)

	for i := 0; i < len(s.queue); i++ {
		if matched[s.queue[i].PlayerID] {
			continue
		}

		for j := i + 1; j < len(s.queue); j++ {
			if matched[s.queue[j].PlayerID] {
				continue
			}

			p1 := s.queue[i]
			p2 := s.queue[j]

			if s.canMatch(p1, p2) {
				result := &MatchResult{
					Player1: p1,
					Player2: p2,
				}
				results = append(results, result)
				matched[p1.PlayerID] = true
				matched[p2.PlayerID] = true
				p1.IsSearching = false
				p2.IsSearching = false
				break
			}
		}
	}

	// 移除已匹配的玩家
	newQueue := make([]*PlayerProfile, 0)
	for _, p := range s.queue {
		if !matched[p.PlayerID] {
			newQueue = append(newQueue, p)
		}
	}
	s.queue = newQueue

	return results
}

// canMatch 检查两名玩家是否可以匹配
func (s *MatchmakingService) canMatch(p1, p2 *PlayerProfile) bool {
	rankDiff := RankOrder[p1.Rank] - RankOrder[p2.Rank]
	if rankDiff < 0 {
		rankDiff = -rankDiff
	}

	// 基础匹配范围
	effectiveRange := s.rankRange

	// 等待时间补偿: 每多等10秒, 匹配范围+1
	waitTime1 := time.Since(p1.EnteredAt)
	waitTime2 := time.Since(p2.EnteredAt)
	maxWait := waitTime1
	if waitTime2 > maxWait {
		maxWait = waitTime2
	}
	bonusRange := int(maxWait.Seconds()) / 10
	effectiveRange += bonusRange

	// 段位差在有效范围内
	if rankDiff > effectiveRange {
		return false
	}

	return true
}

// IsInQueue 检查玩家是否在队列中
func (s *MatchmakingService) IsInQueue(playerID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.queue {
		if p.PlayerID == playerID {
			return true
		}
	}
	return false
}

// QueueSize 队列大小
func (s *MatchmakingService) QueueSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.queue)
}

// CalculateScoreChange 计算段位分变化
//
// 胜方: 基础 +20, 如果对手段位更高则额外加分
// 负方: 基础 -15, 如果对手段位更低则额外扣分
func CalculateScoreChange(winnerRank, loserRank Rank) (winnerGain, loserLoss int) {
	winnerGain = 20
	loserLoss = 15

	rankDiff := RankOrder[winnerRank] - RankOrder[loserRank]
	if rankDiff < 0 {
		// 胜方段位低于负方, 额外奖励
		winnerGain += (-rankDiff) * 5
		loserLoss += (-rankDiff) * 3
	} else if rankDiff > 0 {
		// 胜方段位高于负方, 减少奖励
		winnerGain -= rankDiff * 3
		if winnerGain < 5 {
			winnerGain = 5
		}
		loserLoss -= rankDiff * 2
		if loserLoss < 5 {
			loserLoss = 5
		}
	}

	return
}

// GetRankByScore 根据分数获取段位
func GetRankByScore(score int) Rank {
	switch {
	case score >= 3000:
		return RankLegend
	case score >= 2000:
		return RankDiamond
	case score >= 1500:
		return RankPlatinum
	case score >= 1000:
		return RankGold
	case score >= 500:
		return RankSilver
	default:
		return RankBronze
	}
}

// NewPlayerProfile 创建新玩家竞技档案
func NewPlayerProfile(playerID, name string, level int, powerLevel float64) *PlayerProfile {
	return &PlayerProfile{
		PlayerID:   playerID,
		Name:       name,
		Rank:       RankBronze,
		Score:      0,
		Level:      level,
		PowerLevel: powerLevel,
	}
}

// AutoMatch 自动匹配协程 (在后台运行)
func (s *MatchmakingService) AutoMatch(interval time.Duration, callback func(*MatchResult)) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			results := s.Match()
			for _, res := range results {
				callback(res)
			}
		}
	}()
}

// RandomOpponent 随机选择对手(用于快速匹配)
func (s *MatchmakingService) RandomOpponent(profiles []*PlayerProfile) *PlayerProfile {
	if len(profiles) == 0 {
		return nil
	}
	return profiles[rand.Intn(len(profiles))]
}
