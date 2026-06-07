package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"cultivation-game/services/combat/internal/model"
	"github.com/rs/zerolog/log"
)

// ArenaTickerService 竞技场赛季定时器
// 周期检查赛季是否结束：
//  1. 计算最终排名
//  2. 发放赛季奖励（通过邮件系统通知）
//  3. 重置段位和积分
type ArenaTickerService struct {
	mu          sync.Mutex
	arenaSvc    *ArenaService
	seasonSvc   *SeasonService
	mailSvcAddr string // 邮件服务HTTP地址
	ticker      *time.Ticker
	stopCh      chan struct{}
	running     bool
}

// NewArenaTickerService 创建竞技场赛季定时器
//
//	checkInterval: 赛季检查间隔（默认1分钟）
//	mailServiceAddr: 邮件服务地址，用于发送赛季奖励邮件
func NewArenaTickerService(arenaSvc *ArenaService, seasonSvc *SeasonService, mailServiceAddr string) *ArenaTickerService {
	if mailServiceAddr == "" {
		mailServiceAddr = os.Getenv("MAIL_SERVICE_ADDR")
		if mailServiceAddr == "" {
			mailServiceAddr = "http://127.0.0.1:8084"
		}
	}

	return &ArenaTickerService{
		arenaSvc:    arenaSvc,
		seasonSvc:   seasonSvc,
		mailSvcAddr: mailServiceAddr,
		stopCh:      make(chan struct{}),
	}
}

// Start 启动赛季检查循环
func (s *ArenaTickerService) Start() {
	if s.running {
		return
	}
	s.running = true
	s.ticker = time.NewTicker(1 * time.Minute)

	log.Info().Msg("[赛季] 竞技场赛季检查服务启动，间隔1分钟")

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.checkSeason()
			case <-s.stopCh:
				s.ticker.Stop()
				log.Info().Msg("[赛季] 竞技场赛季检查服务已停止")
				return
			}
		}
	}()
}

// Stop 停止赛季检查循环
func (s *ArenaTickerService) Stop() {
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

// checkSeason 检查赛季是否结束
func (s *ArenaTickerService) checkSeason() {
	currentSeason := s.seasonSvc.GetCurrentSeason()
	if currentSeason == nil {
		return
	}

	now := time.Now().Unix()
	if now < currentSeason.EndTime {
		return // 赛季未结束
	}

	log.Info().
		Int("season_id", currentSeason.SeasonID).
		Str("name", currentSeason.Name).
		Msg("[赛季] 赛季结束，开始结算")

	// 1. 获取玩家排名列表
	rankings := s.arenaSvc.GetRankings()

	// 2. 为每个玩家计算并发送赛季奖励
	s.distributeRewards(rankings)

	// 3. 重置赛季（积分、段位等）
	allPlayers := s.arenaSvc.GetAllPlayers()
	s.seasonSvc.CheckAndEndSeason(allPlayers)

	log.Info().
		Int("season_id", currentSeason.SeasonID).
		Str("name", currentSeason.Name).
		Int("player_count", len(allPlayers)).
		Msg("[赛季] 赛季结算完成：奖励已发放，段位已重置")
}

// distributeRewards 发放赛季奖励
//
//	按排名段发放：
//	  Top 10:  传说段位奖励
//	  Top 50:  钻石段位奖励
//	  Top 100: 黄金段位奖励
//	  Top 500: 白银段位奖励
//	  其余:    青铜段位奖励
func (s *ArenaTickerService) distributeRewards(rankings []*model.ArenaPlayer) {
	season := s.seasonSvc.GetCurrentSeason()
	if season == nil {
		return
	}

	log.Info().
		Int("player_count", len(rankings)).
		Int("season_id", season.SeasonID).
		Msg("[赛季] 开始发放赛季奖励")

	for i, player := range rankings {
		rank := i + 1
		var rankTier string

		switch {
		case rank <= 10:
			rankTier = "legend"
		case rank <= 50:
			rankTier = "diamond"
		case rank <= 100:
			rankTier = "gold"
		case rank <= 500:
			rankTier = "silver"
		default:
			rankTier = "bronze"
		}

		reward := s.seasonSvc.GetReward(rankTier)

		// 异步发送奖励邮件
		go s.sendRewardMail(player.PlayerID, rank, rankTier, reward, season.SeasonID)
	}

	log.Info().Msg("[赛季] 赛季奖励发放完成")
}

// sendRewardMail 发送赛季奖励邮件
//
//	通过邮件服务HTTP接口发送系统邮件
func (s *ArenaTickerService) sendRewardMail(playerID string, rank int, rankTier string, reward *SeasonReward, seasonID int) {
	if reward == nil {
		return
	}

	title := fmt.Sprintf("第%d赛季结算奖励", seasonID)
	content := fmt.Sprintf("恭喜你在第%d赛季中获得第%d名，段位: %s。\n\n赛季奖励:\n- 灵石: %d\n- 修为丹: %d",
		seasonID, rank, rankTier, reward.Stone, reward.Pill)

	if reward.Title != "" {
		content += fmt.Sprintf("\n- 称号: %s%s", reward.Title, map[bool]string{true: "（永久）", false: "（限时）"}[reward.TitlePermanent])
	}
	if reward.Artifact != "" {
		content += fmt.Sprintf("\n- 法宝: %s", reward.Artifact)
	}
	if reward.Frame != "" {
		content += fmt.Sprintf("\n- 头像框: %s", reward.Frame)
	}

	// 构造附件列表
	attachments := []map[string]interface{}{
		{"item_id": "spirit_stone", "quantity": reward.Stone},
		{"item_id": "cultivation_pill", "quantity": reward.Pill},
	}
	if reward.Artifact != "" {
		attachments = append(attachments, map[string]interface{}{
			"item_id": reward.Artifact, "quantity": 1,
		})
	}

	body, _ := json.Marshal(map[string]interface{}{
		"receiver_id":  playerID,
		"title":        title,
		"content":      content,
		"sender_id":    "system",
		"sender_name":  "系统",
		"attachments":  attachments,
		"expire_days":  30,
	})

	url := fmt.Sprintf("%s/api/v1/mail/send", s.mailSvcAddr)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Warn().Err(err).Str("player", playerID).Msg("[赛季] 构造奖励邮件失败")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Warn().Err(err).Str("player", playerID).Msg("[赛季] 发送奖励邮件失败")
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	log.Debug().Str("player", playerID).Int("rank", rank).Str("tier", rankTier).Msg("[赛季] 奖励邮件已发送")
}

// FinalizeSeason 强制结束当前赛季（管理员调用）
func (s *ArenaTickerService) FinalizeSeason() {
	s.mu.Lock()
	defer s.mu.Unlock()

	season := s.seasonSvc.GetCurrentSeason()
	if season == nil {
		return
	}

	log.Warn().Int("season_id", season.SeasonID).Msg("[赛季] 手动强制结算赛季")

	rankings := s.arenaSvc.GetRankings()
	s.distributeRewards(rankings)
}
