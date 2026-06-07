// Package service 宗门副本系统
//
// 宗门成员共同挑战首领，按伤害排名发放奖励
package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// ============================================================
// 常量
// ============================================================

const (
	// SectDungeonStatusPreparing  准备中
	SectDungeonStatusPreparing = 0
	// SectDungeonStatusInProgress 进行中
	SectDungeonStatusInProgress = 1
	// SectDungeonStatusCompleted  已完成(击败首领)
	SectDungeonStatusCompleted = 2
	// SectDungeonStatusFailed     已失败(超时)
	SectDungeonStatusFailed = 3

	// SectDungeonAttackCooldownSec 攻击冷却时间(秒)
	SectDungeonAttackCooldownSec = 3

	// SectDungeonDamageVariance 伤害浮动范围
	SectDungeonDamageVariance = 0.15
	// SectDungeonCritChance 暴击概率
	SectDungeonCritChance = 0.12
	// SectDungeonCritMultiplier 暴击倍率
	SectDungeonCritMultiplier = 2.0

	// SectDungeonTimerIntervalSec 定时器检查间隔(秒)
	SectDungeonTimerIntervalSec = 10
)

// ============================================================
// 静态配置类型
// ============================================================

// SectDungeonConfig 宗门副本配置
type SectDungeonConfig struct {
	ID               int                    `json:"id"`
	Name             string                 `json:"name"`
	RealmRequired    int                    `json:"realm_required"`
	BossHP           int64                  `json:"boss_hp"`
	BossAtk          int                    `json:"boss_atk"`
	BossDef          int                    `json:"boss_def"`
	BossSkills       []SectDungeonBossSkill `json:"boss_skills"`
	MaxParticipants  int                    `json:"max_participants"`
	DurationMinutes  int                    `json:"duration_minutes"`
	UnlockCost       int                    `json:"unlock_cost"`
	Rewards          []SectDungeonRewardTier `json:"rewards"`
}

// SectDungeonBossSkill 首领技能
type SectDungeonBossSkill struct {
	Name        string  `json:"name"`
	DamageMult  float64 `json:"damage_mult"`
	Description string  `json:"description"`
}

// SectDungeonRewardTier 奖励档位
type SectDungeonRewardTier struct {
	RankFrom          int                    `json:"rank_from"`
	RankTo            int                    `json:"rank_to"`
	SpiritStones      int64                  `json:"spirit_stones"`
	SectContribution  int64                  `json:"sect_contribution"`
	Items             []SectDungeonRewardItem `json:"items"`
}

// SectDungeonRewardItem 奖励物品
type SectDungeonRewardItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ============================================================
// 运行时数据类型
// ============================================================

// SectDungeonSession 宗门副本会话
type SectDungeonSession struct {
	ID             uint64                       `json:"id"`
	SectID         uint64                       `json:"sect_id"`
	ConfigID       int                          `json:"config_id"`
	Status         int                          `json:"status"`
	StartedAt      time.Time                    `json:"started_at"`
	EndedAt        time.Time                    `json:"ended_at"`
	TotalDamage    int64                        `json:"total_damage"`
	BossRemainingHP int64                       `json:"boss_remaining_hp"`
	Participants   []*SectDungeonParticipant    `json:"participants"`
}

// SectDungeonParticipant 参与者数据
type SectDungeonParticipant struct {
	ID            uint64    `json:"id"`
	SessionID     uint64    `json:"session_id"`
	PlayerID      uint64    `json:"player_id"`
	DamageDealt   int64     `json:"damage_dealt"`
	Rank          int       `json:"rank"`
	RewardsClaimed bool     `json:"rewards_claimed"`
	LastAttackAt  time.Time `json:"last_attack_at"`
	JoinedAt      time.Time `json:"joined_at"`
}

// ============================================================
// 请求/响应类型
// ============================================================

// SectDungeonStartReq 开启副本请求
type SectDungeonStartReq struct {
	SectID         uint64 `json:"sect_id"`
	DungeonConfigID int    `json:"dungeon_config_id"`
	LeaderID       uint64 `json:"leader_id"`
}

// SectDungeonJoinReq 加入副本请求
type SectDungeonJoinReq struct {
	SessionID uint64 `json:"session_id"`
	PlayerID  uint64 `json:"player_id"`
}

// SectDungeonAttackReq 攻击请求
type SectDungeonAttackReq struct {
	SessionID uint64 `json:"session_id"`
	PlayerID  uint64 `json:"player_id"`
	AttackVal int64  `json:"attack_val"`
}

// SectDungeonAttackResult 攻击结果
type SectDungeonAttackResult struct {
	Damage         int64  `json:"damage"`
	Critical       bool   `json:"critical"`
	BossRemainingHP int64 `json:"boss_remaining_hp"`
	BossMaxHP      int64  `json:"boss_max_hp"`
	BossAlive      bool   `json:"boss_alive"`
	TotalDamage    int64  `json:"total_damage"`
	YourDamage     int64  `json:"your_damage"`
	YourRank       int    `json:"your_rank"`
	SkillUsed      string `json:"skill_used"`
}

// SectDungeonStatus 副本状态响应
type SectDungeonStatus struct {
	Session          *SectDungeonSession      `json:"session,omitempty"`
	Config           *SectDungeonConfig       `json:"config,omitempty"`
	RemainingSeconds int                      `json:"remaining_seconds"`
	Participants     []*SectDungeonParticipant `json:"participants"`
	Active           bool                      `json:"active"`
}

// SectDungeonCompleteResult 完成结果
type SectDungeonCompleteResult struct {
	SessionID   uint64                  `json:"session_id"`
	ConfigName  string                  `json:"config_name"`
	BossDefeated bool                   `json:"boss_defeated"`
	TotalDamage int64                   `json:"total_damage"`
	Rankings    []*SectDungeonRankEntry `json:"rankings"`
}

// SectDungeonRankEntry 排名条目
type SectDungeonRankEntry struct {
	Rank         int    `json:"rank"`
	PlayerID     uint64 `json:"player_id"`
	DamageDealt  int64  `json:"damage_dealt"`
	Rewards      *SectDungeonRewardInfo `json:"rewards,omitempty"`
}

// SectDungeonRewardInfo 奖励信息
type SectDungeonRewardInfo struct {
	SpiritStones     int64                    `json:"spirit_stones"`
	SectContribution int64                    `json:"sect_contribution"`
	Items            []SectDungeonRewardItem  `json:"items"`
}

// ============================================================
// SectDungeonService
// ============================================================

// SectDungeonService 宗门副本业务逻辑
type SectDungeonService struct {
	mu       sync.RWMutex
	configs  map[int]*SectDungeonConfig   // dungeonConfigID -> config
	sessions map[uint64]*SectDungeonSession // sessionID -> session
	nextID   uint64                        // 自增ID

	stopCh  chan struct{}
	running bool
}

// NewSectDungeonService 创建宗门副本服务
func NewSectDungeonService() *SectDungeonService {
	svc := &SectDungeonService{
		configs:  make(map[int]*SectDungeonConfig),
		sessions: make(map[uint64]*SectDungeonSession),
		nextID:   1,
		stopCh:   make(chan struct{}),
	}
	svc.loadConfigs()
	return svc
}

// Start 启动后台定时器(检查过期会话)
func (s *SectDungeonService) Start() {
	if s.running {
		return
	}
	s.running = true
	log.Info().Msg("[宗门副本] 宗门副本系统启动")

	go func() {
		ticker := time.NewTicker(SectDungeonTimerIntervalSec * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.checkExpiredSessions()
			case <-s.stopCh:
				log.Info().Msg("[宗门副本] 定时器已停止")
				return
			}
		}
	}()
}

// Stop 停止后台定时器
func (s *SectDungeonService) Stop() {
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

// checkExpiredSessions 检查过期的副本会话
func (s *SectDungeonService) checkExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, session := range s.sessions {
		if session.Status != SectDungeonStatusInProgress {
			continue
		}
		elapsed := now.Sub(session.StartedAt)
		cfg, ok := s.configs[session.ConfigID]
		if !ok {
			continue
		}
		if elapsed >= time.Duration(cfg.DurationMinutes)*time.Minute {
			session.Status = SectDungeonStatusFailed
			session.EndedAt = now
			log.Info().Uint64("session_id", session.ID).
				Uint64("sect_id", session.SectID).
				Int("config_id", session.ConfigID).
				Msg("[宗门副本] 副本超时结束")
		}
	}
}

// loadConfigs 从 JSON 加载宗门副本配置
func (s *SectDungeonService) loadConfigs() {
	data, err := os.ReadFile("internal/data/sect_dungeons.json")
	if err != nil {
		log.Warn().Err(err).Msg("[宗门副本] 加载配置文件失败, 使用内置默认配置")
		s.initDefaultConfigs()
		return
	}

	var raw struct {
		Dungeons []*SectDungeonConfig `json:"dungeons"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Warn().Err(err).Msg("[宗门副本] 解析配置文件失败, 使用内置默认配置")
		s.initDefaultConfigs()
		return
	}

	for _, d := range raw.Dungeons {
		s.configs[d.ID] = d
	}
	log.Info().Int("count", len(s.configs)).Msg("[宗门副本] 配置加载完成")
}

// initDefaultConfigs 内置默认配置(兜底)
func (s *SectDungeonService) initDefaultConfigs() {
	s.configs = map[int]*SectDungeonConfig{
		1: {
			ID: 1, Name: "守护山门", RealmRequired: 1,
			BossHP: 50000, BossAtk: 200, BossDef: 50,
			MaxParticipants: 10, DurationMinutes: 30, UnlockCost: 5000,
			BossSkills: []SectDungeonBossSkill{
				{Name: "山门震荡", DamageMult: 1.5, Description: "以山门之力震荡周围敌人"},
			},
			Rewards: []SectDungeonRewardTier{
				{RankFrom: 1, RankTo: 1, SpiritStones: 5000, SectContribution: 200},
				{RankFrom: 2, RankTo: 3, SpiritStones: 3000, SectContribution: 150},
				{RankFrom: 4, RankTo: 10, SpiritStones: 1500, SectContribution: 100},
			},
		},
	}
}

// ============================================================
// 公开API
// ============================================================

// GetConfigs 获取所有副本配置
func (s *SectDungeonService) GetConfigs() []*SectDungeonConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]int, 0, len(s.configs))
	for id := range s.configs {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	result := make([]*SectDungeonConfig, 0, len(ids))
	for _, id := range ids {
		result = append(result, s.configs[id])
	}
	return result
}

// GetConfig 获取指定配置
func (s *SectDungeonService) GetConfig(configID int) *SectDungeonConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configs[configID]
}

// GetActiveSectDungeon 获取宗门当前活跃的副本
func (s *SectDungeonService) GetActiveSectDungeon(sectID uint64) *SectDungeonStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.sessions {
		if session.SectID == sectID && session.Status == SectDungeonStatusInProgress {
			cfg := s.configs[session.ConfigID]
			remaining := 0
			if cfg != nil {
				elapsed := time.Since(session.StartedAt)
				total := time.Duration(cfg.DurationMinutes) * time.Minute
				rem := total - elapsed
				if rem > 0 {
					remaining = int(rem.Seconds())
				}
			}

			// 排序参与者
			sorted := make([]*SectDungeonParticipant, len(session.Participants))
			copy(sorted, session.Participants)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].DamageDealt > sorted[j].DamageDealt
			})

			return &SectDungeonStatus{
				Session:          session,
				Config:           cfg,
				RemainingSeconds: remaining,
				Participants:     sorted,
				Active:           true,
			}
		}
	}
	return &SectDungeonStatus{Active: false}
}

// GetSession 获取指定会话
func (s *SectDungeonService) GetSession(sessionID uint64) *SectDungeonSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[sessionID]
}

// StartSectDungeon 开启宗门副本
func (s *SectDungeonService) StartSectDungeon(req *SectDungeonStartReq) (*SectDungeonSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查配置是否存在
	cfg, ok := s.configs[req.DungeonConfigID]
	if !ok {
		return nil, fmt.Errorf("副本配置不存在")
	}

	// 检查宗门是否有进行中的副本
	for _, session := range s.sessions {
		if session.SectID == req.SectID &&
			(session.Status == SectDungeonStatusPreparing || session.Status == SectDungeonStatusInProgress) {
			return nil, fmt.Errorf("宗门已有进行中的副本")
		}
	}

	// 创建新会话
	session := &SectDungeonSession{
		ID:              s.nextID,
		SectID:          req.SectID,
		ConfigID:        req.DungeonConfigID,
		Status:          SectDungeonStatusInProgress,
		StartedAt:       time.Now(),
		TotalDamage:     0,
		BossRemainingHP: cfg.BossHP,
		Participants:    make([]*SectDungeonParticipant, 0),
	}
	s.sessions[s.nextID] = session
	s.nextID++

	log.Info().Uint64("session_id", session.ID).
		Uint64("sect_id", req.SectID).
		Int("config_id", req.DungeonConfigID).
		Str("dungeon_name", cfg.Name).
		Msg("[宗门副本] 副本已开启")

	return session, nil
}

// JoinSectDungeon 加入宗门副本
func (s *SectDungeonService) JoinSectDungeon(req *SectDungeonJoinReq) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[req.SessionID]
	if !ok {
		return fmt.Errorf("副本会话不存在")
	}

	if session.Status != SectDungeonStatusInProgress {
		return fmt.Errorf("副本不在进行中")
	}

	cfg, ok := s.configs[session.ConfigID]
	if !ok {
		return fmt.Errorf("副本配置不存在")
	}

	// 检查人数上限
	if len(session.Participants) >= cfg.MaxParticipants {
		return fmt.Errorf("参与人数已满(上限%d人)", cfg.MaxParticipants)
	}

	// 检查是否已加入
	for _, p := range session.Participants {
		if p.PlayerID == req.PlayerID {
			return fmt.Errorf("已在副本中")
		}
	}

	// 添加参与者
	participant := &SectDungeonParticipant{
		ID:        uint64(len(session.Participants)) + 1,
		SessionID: req.SessionID,
		PlayerID:  req.PlayerID,
		JoinedAt:  time.Now(),
	}
	session.Participants = append(session.Participants, participant)

	log.Info().Uint64("session_id", req.SessionID).
		Uint64("player_id", req.PlayerID).
		Msg("[宗门副本] 玩家加入副本")

	return nil
}

// AttackBoss 攻击首领
func (s *SectDungeonService) AttackBoss(req *SectDungeonAttackReq) (*SectDungeonAttackResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[req.SessionID]
	if !ok {
		return nil, fmt.Errorf("副本会话不存在")
	}

	if session.Status != SectDungeonStatusInProgress {
		return nil, fmt.Errorf("副本不在进行中")
	}

	cfg, ok := s.configs[session.ConfigID]
	if !ok {
		return nil, fmt.Errorf("副本配置不存在")
	}

	// 查找参与者
	var participant *SectDungeonParticipant
	for _, p := range session.Participants {
		if p.PlayerID == req.PlayerID {
			participant = p
			break
		}
	}
	if participant == nil {
		return nil, fmt.Errorf("未加入副本, 请先加入")
	}

	// 检查攻击冷却
	if !participant.LastAttackAt.IsZero() {
		elapsed := time.Since(participant.LastAttackAt)
		if elapsed < SectDungeonAttackCooldownSec*time.Second {
			rem := SectDungeonAttackCooldownSec - int(elapsed.Seconds())
			return nil, fmt.Errorf("攻击冷却中, 剩余%d秒", rem)
		}
	}

	// 计算伤害
	baseDmg := req.AttackVal
	if baseDmg <= 0 {
		baseDmg = 100
	}

	// 伤害浮动
	variance := 1.0 + (rand.Float64()*2-1)*SectDungeonDamageVariance
	rawDmg := int64(float64(baseDmg) * variance)
	if rawDmg < 1 {
		rawDmg = 1
	}

	// 暴击判定
	critical := rand.Float64() < SectDungeonCritChance
	if critical {
		rawDmg = int64(float64(rawDmg) * SectDungeonCritMultiplier)
	}

	// 不超过剩余血量
	if rawDmg >= session.BossRemainingHP {
		rawDmg = session.BossRemainingHP
	}

	// 随机选择一个技能
	skillName := "普通攻击"
	if len(cfg.BossSkills) > 0 {
		skill := cfg.BossSkills[rand.Intn(len(cfg.BossSkills))]
		skillName = fmt.Sprintf("施展[%s]", skill.Name)
	}

	// 更新数据
	session.BossRemainingHP -= rawDmg
	session.TotalDamage += rawDmg
	participant.DamageDealt += rawDmg
	participant.LastAttackAt = time.Now()

	// 检查首领是否被击败
	bossAlive := session.BossRemainingHP > 0
	if !bossAlive {
		session.Status = SectDungeonStatusCompleted
		session.EndedAt = time.Now()

		// 计算排名
		s.calculateRanks(session)

		log.Info().Uint64("session_id", session.ID).
			Uint64("sect_id", session.SectID).
			Msg("[宗门副本] 首领被击败!")
	}

	// 计算当前排名
	yourRank := s.calculatePlayerRank(session, participant.PlayerID)

	result := &SectDungeonAttackResult{
		Damage:          rawDmg,
		Critical:        critical,
		BossRemainingHP: session.BossRemainingHP,
		BossMaxHP:       cfg.BossHP,
		BossAlive:       bossAlive,
		TotalDamage:     session.TotalDamage,
		YourDamage:      participant.DamageDealt,
		YourRank:        yourRank,
		SkillUsed:       skillName,
	}

	return result, nil
}

// CalculateRanks 计算所有参与者的排名(按伤害降序)
func (s *SectDungeonService) calculateRanks(session *SectDungeonSession) {
	sort.Slice(session.Participants, func(i, j int) bool {
		return session.Participants[i].DamageDealt > session.Participants[j].DamageDealt
	})

	for i, p := range session.Participants {
		p.Rank = i + 1
	}
}

// calculatePlayerRank 计算单个玩家的排名
func (s *SectDungeonService) calculatePlayerRank(session *SectDungeonSession, playerID uint64) int {
	rank := 1
	for _, p := range session.Participants {
		if p.PlayerID == playerID {
			// 统计伤害高于该玩家的数量
			count := 0
			for _, other := range session.Participants {
				if other.DamageDealt > p.DamageDealt {
					count++
				}
			}
			rank = count + 1
			break
		}
	}
	return rank
}

// CompleteSectDungeon 完成副本(手动结束/领取奖励)
func (s *SectDungeonService) CompleteSectDungeon(sessionID uint64) (*SectDungeonCompleteResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("副本会话不存在")
	}

	if session.Status != SectDungeonStatusInProgress &&
		session.Status != SectDungeonStatusCompleted {
		return nil, fmt.Errorf("副本状态不正确")
	}

	cfg, ok := s.configs[session.ConfigID]
	if !ok {
		return nil, fmt.Errorf("副本配置不存在")
	}

	// 更新状态
	bossDefeated := session.BossRemainingHP <= 0 || session.Status == SectDungeonStatusCompleted
	if session.Status == SectDungeonStatusInProgress {
		if bossDefeated {
			session.Status = SectDungeonStatusCompleted
		} else {
			session.Status = SectDungeonStatusFailed
		}
		session.EndedAt = time.Now()
	}

	// 计算排名
	s.calculateRanks(session)

	// 构建完成结果
	rankings := make([]*SectDungeonRankEntry, 0, len(session.Participants))
	for _, p := range session.Participants {
		entry := &SectDungeonRankEntry{
			Rank:        p.Rank,
			PlayerID:    p.PlayerID,
			DamageDealt: p.DamageDealt,
		}

		// 计算奖励
		if bossDefeated && p.Rank > 0 {
			entry.Rewards = s.calculateParticipantRewards(p.Rank, cfg)
		}

		rankings = append(rankings, entry)
	}

	result := &SectDungeonCompleteResult{
		SessionID:    sessionID,
		ConfigName:   cfg.Name,
		BossDefeated: bossDefeated,
		TotalDamage:  session.TotalDamage,
		Rankings:     rankings,
	}

	return result, nil
}

// calculateParticipantRewards 根据排名计算奖励
func (s *SectDungeonService) calculateParticipantRewards(rank int, cfg *SectDungeonConfig) *SectDungeonRewardInfo {
	for _, tier := range cfg.Rewards {
		if rank >= tier.RankFrom && rank <= tier.RankTo {
			return &SectDungeonRewardInfo{
				SpiritStones:     tier.SpiritStones,
				SectContribution: tier.SectContribution,
				Items:            tier.Items,
			}
		}
	}

	// 默认参与奖: 使用最后一档
	if len(cfg.Rewards) > 0 {
		last := cfg.Rewards[len(cfg.Rewards)-1]
		return &SectDungeonRewardInfo{
			SpiritStones:     last.SpiritStones,
			SectContribution: last.SectContribution,
			Items:            last.Items,
		}
	}

	return &SectDungeonRewardInfo{}
}

// ClaimRewards 领取奖励(标记已领取)
func (s *SectDungeonService) ClaimRewards(sessionID uint64, playerID uint64) (*SectDungeonRewardInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("副本会话不存在")
	}

	if session.Status != SectDungeonStatusCompleted {
		return nil, fmt.Errorf("副本未完成, 无法领取奖励")
	}

	cfg, ok := s.configs[session.ConfigID]
	if !ok {
		return nil, fmt.Errorf("副本配置不存在")
	}

	for _, p := range session.Participants {
		if p.PlayerID == playerID {
			if p.RewardsClaimed {
				return nil, fmt.Errorf("奖励已领取")
			}

			p.RewardsClaimed = true
			rewards := s.calculateParticipantRewards(p.Rank, cfg)

			log.Info().Uint64("session_id", sessionID).
				Uint64("player_id", playerID).
				Int("rank", p.Rank).
				Msg("[宗门副本] 玩家领取奖励")

			return rewards, nil
		}
	}

	return nil, fmt.Errorf("未参与此副本")
}

// GetSectDungeonHistory 获取宗门最近的历史记录
func (s *SectDungeonService) GetSectDungeonHistory(sectID uint64, limit int) []*SectDungeonSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var history []*SectDungeonSession
	for _, session := range s.sessions {
		if session.SectID == sectID &&
			(session.Status == SectDungeonStatusCompleted || session.Status == SectDungeonStatusFailed) {
			history = append(history, session)
		}
	}

	// 按结束时间降序
	sort.Slice(history, func(i, j int) bool {
		return history[i].EndedAt.After(history[j].EndedAt)
	})

	if limit <= 0 || limit > len(history) {
		limit = len(history)
	}
	return history[:limit]
}
