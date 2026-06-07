package service

import (
	"fmt"
	"sync"
	"time"

	"cultivation-game/services/combat/internal/model"
	"github.com/rs/zerolog/log"
)

// SeasonReward 赛季奖励定义
type SeasonReward struct {
	Stone          int    `json:"stone"`           // 灵石
	Pill           int    `json:"pill"`            // 修为丹
	Title          string `json:"title,omitempty"`            // 称号
	Artifact       string `json:"artifact,omitempty"`         // 法宝
	Frame          string `json:"frame,omitempty"`            // 头像框
	TitlePermanent bool   `json:"title_permanent"` // 称号是否永久
}

// SeasonRewardsByRank 各段位对应的赛季奖励
var SeasonRewardsByRank = map[string]SeasonReward{
	"bronze": {
		Stone: 5000,
		Pill:  10,
	},
	"silver": {
		Stone: 10000,
		Pill:  20,
		Title: "白银斗士",
	},
	"gold": {
		Stone:     20000,
		Pill:      50,
		Title:     "黄金斗士",
		Artifact:  "gold_artifact",
	},
	"diamond": {
		Stone:          50000,
		Pill:           100,
		Title:          "钻石尊者",
		Artifact:       "diamond_artifact",
		TitlePermanent: true,
	},
	"legend": {
		Stone:          100000,
		Pill:           200,
		Title:          "传说至尊",
		Artifact:       "legend_artifact",
		Frame:          "legend_frame",
		TitlePermanent: true,
	},
}

// SeasonService 赛季管理服务
type SeasonService struct {
	mu       sync.RWMutex
	Season   *model.SeasonInfo
	duration time.Duration
	number   int
}

// NewSeasonService 创建赛季服务
//
//  seasonDurationDays: 每个赛季持续的天数
func NewSeasonService(seasonDurationDays int) *SeasonService {
	now := time.Now()
	duration := time.Duration(seasonDurationDays) * 24 * time.Hour

	return &SeasonService{
		duration: duration,
		number:   1,
		Season: &model.SeasonInfo{
			SeasonID:  1,
			Name:      "第一赛季",
			StartTime: now.Unix(),
			EndTime:   now.Add(duration).Unix(),
			Status:    "active",
		},
	}
}

// GetCurrentSeason 获取当前赛季信息
func (s *SeasonService) GetCurrentSeason() *model.SeasonInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	season := *s.Season
	return &season
}

// GetReward 根据段位获取赛季奖励(返回副本)
func (s *SeasonService) GetReward(rank string) *SeasonReward {
	reward, ok := SeasonRewardsByRank[rank]
	if !ok {
		r := SeasonRewardsByRank["bronze"]
		return &r
	}
	return &reward
}

// CheckAndEndSeason 检查赛季是否结束, 是则执行赛季切换并重置玩家数据
//
//  返回 true 表示赛季已切换
func (s *SeasonService) CheckAndEndSeason(players map[string]*model.ArenaPlayer) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	if now < s.Season.EndTime {
		return false
	}

	// 赛季结束, 发放奖励(记录)并重置玩家
	s.endSeason(players)

	// 启动新赛季
	s.number++
	s.Season = &model.SeasonInfo{
		SeasonID:  s.number,
		Name:      fmt.Sprintf("第%d赛季", s.number),
		StartTime: now,
		EndTime:   time.Now().Add(s.duration).Unix(),
		Status:    "active",
	}

	log.Info().
		Int("season_id", s.Season.SeasonID).
		Str("name", s.Season.Name).
		Int("player_count", len(players)).
		Msg("新赛季开始, 玩家数据已重置")
	return true
}

// endSeason 结束当前赛季: 记录上赛季段位, 重置积分/段位/胜场
func (s *SeasonService) endSeason(players map[string]*model.ArenaPlayer) {
	for _, p := range players {
		p.LastSeasonRank = p.Rank
		p.Score = 1000
		p.Rank = "bronze"
		p.Tier = 3
		p.SeasonWin = 0
		p.SeasonLose = 0
		p.Streak = 0
		p.DailyWinCount = 0
	}
}
