// Package service 组队副本系统
//
// 玩家组队(3-5人)进入独立副本, 依次挑战怪物波次+最终Boss
// 团队协作机制: 同元素连击, 阵型加成, 贡献度奖励
package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ============================================================
// 常量
// ============================================================

const (
	// TeamDungeonStatusRecruiting  招募中
	TeamDungeonStatusRecruiting = 0
	// TeamDungeonStatusReady      已就绪
	TeamDungeonStatusReady = 1
	// TeamDungeonStatusInProgress 进行中
	TeamDungeonStatusInProgress = 2
	// TeamDungeonStatusCompleted  已完成
	TeamDungeonStatusCompleted = 3
	// TeamDungeonStatusFailed     已失败
	TeamDungeonStatusFailed = 4

	// TeamDungeonInvitePending  待处理
	TeamDungeonInvitePending = 0
	// TeamDungeonInviteAccepted 已接受
	TeamDungeonInviteAccepted = 1
	// TeamDungeonInviteDeclined 已拒绝
	TeamDungeonInviteDeclined = 2
	// TeamDungeonInviteExpired  已过期
	TeamDungeonInviteExpired = 3

	// TeamDungeonPositionTank    坦克
	TeamDungeonPositionTank = 1
	// TeamDungeonPositionDPS     输出
	TeamDungeonPositionDPS = 2
	// TeamDungeonPositionSupport 辅助
	TeamDungeonPositionSupport = 3

	// TeamDungeonInviteExpireSec 邀请过期时间(秒)
	TeamDungeonInviteExpireSec = 120

	// TeamDungeonFormationBonusTank    坦克位减伤加成
	TeamDungeonFormationBonusTank = 0.8
	// TeamDungeonFormationBonusDPS     输出位伤害加成
	TeamDungeonFormationBonusDPS = 1.2
	// TeamDungeonFormationBonusSupport 辅助位治疗加成
	TeamDungeonFormationBonusSupport = 1.3

	// TeamDungeonComboMultiplier 同元素连击倍率
	TeamDungeonComboMultiplier = 1.5
	// TeamDungeonComboCountMin   触发连击最少同元素数
	TeamDungeonComboCountMin = 2
)

// ============================================================
// 静态配置类型(对应JSON)
// ============================================================

// TeamDungeonConfig 组队副本配置
type TeamDungeonConfig struct {
	ID               int                 `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	RealmRequired    int                 `json:"realm_required"`
	MinPlayers       int                 `json:"min_players"`
	MaxPlayers       int                 `json:"max_players"`
	TimeLimitMinutes int                 `json:"time_limit_minutes"`
	Waves            []TeamDungeonWave   `json:"waves"`
	Boss             TeamDungeonBoss     `json:"boss"`
	Rewards          TeamDungeonReward   `json:"rewards"`
}

// TeamDungeonWave 波次配置
type TeamDungeonWave struct {
	Wave     int                    `json:"wave"`
	Name     string                 `json:"name"`
	Monsters []TeamDungeonMonster   `json:"monsters"`
}

// TeamDungeonMonster 怪物配置
type TeamDungeonMonster struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	HP      int64  `json:"hp"`
	Atk     int64  `json:"atk"`
	Def     int64  `json:"def"`
	Speed   int64  `json:"speed"`
	Level   int    `json:"level"`
	Element string `json:"element"`
}

// TeamDungeonBoss Boss配置
type TeamDungeonBoss struct {
	ID      int                    `json:"id"`
	Name    string                 `json:"name"`
	HP      int64                  `json:"hp"`
	Atk     int64                  `json:"atk"`
	Def     int64                  `json:"def"`
	Speed   int64                  `json:"speed"`
	Level   int                    `json:"level"`
	Element string                 `json:"element"`
	Skills  []TeamDungeonBossSkill `json:"skills"`
}

// TeamDungeonBossSkill Boss技能
type TeamDungeonBossSkill struct {
	Name        string  `json:"name"`
	DamageMult  float64 `json:"damage_mult"`
	Description string  `json:"description"`
}

// TeamDungeonReward 奖励配置
type TeamDungeonReward struct {
	BaseExp           int64                       `json:"base_exp"`
	BaseSpiritStones  int64                       `json:"base_spirit_stones"`
	Items             []TeamDungeonRewardItem     `json:"items"`
	SpecialBonus      TeamDungeonSpecialBonus     `json:"special_bonus"`
	ContributionRewards TeamDungeonContribution   `json:"contribution_rewards"`
}

// TeamDungeonRewardItem 奖励物品
type TeamDungeonRewardItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// TeamDungeonSpecialBonus 特殊奖励加成
type TeamDungeonSpecialBonus struct {
	NoDeathBonusExp int64 `json:"no_death_bonus_exp"`
	TimeBonusExp    int64 `json:"time_bonus_exp"`
}

// TeamDungeonContribution 贡献度权重
type TeamDungeonContribution struct {
	DamageWeight  float64 `json:"damage_weight"`
	HealingWeight float64 `json:"healing_weight"`
	SupportWeight float64 `json:"support_weight"`
}

// ============================================================
// 运行时数据类型
// ============================================================

// TeamDungeonTeam 队伍
type TeamDungeonTeam struct {
	ID             string                  `json:"id"`
	DungeonConfigID int                   `json:"dungeon_config_id"`
	LeaderID       string                  `json:"leader_id"`
	Status         int                     `json:"status"`
	Members        []*TeamDungeonTeamMember `json:"members"`
	CurrentWave    int                     `json:"current_wave"`
	StartedAt      time.Time               `json:"started_at"`
	CompletedAt    time.Time               `json:"completed_at"`
	TotalDamage    int64                   `json:"total_damage"`
	TimeLimitSec   int                     `json:"time_limit_sec"`
	CreatedAt      time.Time               `json:"created_at"`

	// 副本会话中的运行时状态
	Instance *TeamDungeonInstance `json:"-"`

	// 完成详情(结算用)
	Completion *TeamDungeonCompletion `json:"-"`
}

// TeamDungeonTeamMember 队伍成员
type TeamDungeonTeamMember struct {
	PlayerID        string    `json:"player_id"`
	PlayerName      string    `json:"player_name"`
	Position        int       `json:"position"`
	Ready           bool      `json:"ready"`
	DamageDealt     int64     `json:"damage_dealt"`
	HealingDone     int64     `json:"healing_done"`
	SupportProvided int64     `json:"support_provided"`
	Contribution     int64     `json:"-"`
	RewardsClaimed  bool      `json:"rewards_claimed"`
	JoinedAt        time.Time `json:"joined_at"`
}

// TeamDungeonInvite 邀请
type TeamDungeonInvite struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	InviterID string    `json:"inviter_id"`
	InviteeID string    `json:"invitee_id"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TeamDungeonInstance 副本实例(运行时)
type TeamDungeonInstance struct {
	TeamID       string                      `json:"team_id"`
	CurrentWave  int                         `json:"current_wave"`
	TotalWaves   int                         `json:"total_waves"`
	WaveMonsters []*TeamDungeonWaveMonster   `json:"-"`
	Boss         *TeamDungeonBossInstance    `json:"-"`
	StartedAt    time.Time                   `json:"started_at"`
	ElapsedSec   int                         `json:"elapsed_sec"`
	AllDead      bool                        `json:"all_dead"`
	BossDefeated bool                        `json:"boss_defeated"`
	Completed    bool                        `json:"completed"`
}

// TeamDungeonWaveMonster 波次怪物运行时
type TeamDungeonWaveMonster struct {
	Config    TeamDungeonMonster `json:"config"`
	CurrentHP int64              `json:"current_hp"`
	Alive     bool               `json:"alive"`
}

// TeamDungeonBossInstance Boss运行时
type TeamDungeonBossInstance struct {
	Config    TeamDungeonBoss   `json:"config"`
	CurrentHP int64             `json:"current_hp"`
	Alive     bool              `json:"alive"`
	Skills    []TeamDungeonBossSkill `json:"skills"`
}

// TeamDungeonCompletion 完成结算
type TeamDungeonCompletion struct {
	Completed        bool                              `json:"completed"`
	BossDefeated     bool                              `json:"boss_defeated"`
	TotalDamage      int64                             `json:"total_damage"`
	TimeUsedSec      int                               `json:"time_used_sec"`
	NoDeaths         bool                              `json:"no_deaths"`
	MemberRewards    []*TeamDungeonMemberReward        `json:"member_rewards"`
}

// TeamDungeonMemberReward 成员奖励
type TeamDungeonMemberReward struct {
	PlayerID       string `json:"player_id"`
	Exp            int64  `json:"exp"`
	SpiritStones   int64  `json:"spirit_stones"`
	Contribution   int64  `json:"contribution"`
	Items          []TeamDungeonRewardItem `json:"items"`
	ContributionPct float64 `json:"contribution_pct"`
}

// ============================================================
// 请求/响应类型
// ============================================================

// TeamDungeonCreateReq 创建队伍请求
type TeamDungeonCreateReq struct {
	PlayerID       string `json:"player_id"`
	PlayerName     string `json:"player_name"`
	DungeonConfigID int    `json:"dungeon_config_id"`
	Position       int    `json:"position"`
}

// TeamDungeonJoinReq 加入队伍请求
type TeamDungeonJoinReq struct {
	TeamID     string `json:"team_id"`
	PlayerID   string `json:"player_id"`
	PlayerName string `json:"player_name"`
	Position   int    `json:"position"`
}

// TeamDungeonLeaveReq 离开队伍请求
type TeamDungeonLeaveReq struct {
	TeamID   string `json:"team_id"`
	PlayerID string `json:"player_id"`
}

// TeamDungeonInviteReq 邀请请求
type TeamDungeonInviteReq struct {
	TeamID    string `json:"team_id"`
	InviterID string `json:"inviter_id"`
	InviteeID string `json:"invitee_id"`
}

// TeamDungeonReadyReq 就绪请求
type TeamDungeonReadyReq struct {
	TeamID   string `json:"team_id"`
	PlayerID string `json:"player_id"`
	Ready    bool   `json:"ready"`
}

// TeamDungeonStartReq 开始副本请求
type TeamDungeonStartReq struct {
	TeamID   string `json:"team_id"`
	PlayerID string `json:"player_id"`
}

// TeamDungeonAttackReq 攻击请求(玩家行动)
type TeamDungeonAttackReq struct {
	TeamID    string `json:"team_id"`
	PlayerID  string `json:"player_id"`
	SkillID   int    `json:"skill_id"`
	Element   string `json:"element"`
	TargetID  int    `json:"target_id"`
	IsHeal    bool   `json:"is_heal"`
	IsSupport bool   `json:"is_support"`
}

// TeamDungeonAttackResult 攻击结果
type TeamDungeonAttackResult struct {
	PlayerID      string `json:"player_id"`
	DamageDealt   int64  `json:"damage_dealt"`
	HealingDone   int64  `json:"healing_done"`
	SupportValue  int64  `json:"support_value"`
	Critical      bool   `json:"critical"`
	ComboActive   bool   `json:"combo_active"`
	ComboCount    int    `json:"combo_count"`
	MonsterKilled bool   `json:"monster_killed"`
	TargetID      int    `json:"target_id"`
	TargetName    string `json:"target_name"`
	Log           string `json:"log"`
}

// TeamDungeonWaveResult 波次结果
type TeamDungeonWaveResult struct {
	Wave        int                       `json:"wave"`
	WaveName    string                    `json:"wave_name"`
	Cleared     bool                      `json:"cleared"`
	IsBossWave  bool                      `json:"is_boss_wave"`
	BossDefeated bool                     `json:"boss_defeated"`
	Monsters    []TeamDungeonMonsterStatus `json:"monsters"`
	Actions     []*TeamDungeonAttackResult `json:"actions"`
	Logs        []string                   `json:"logs"`
}

// TeamDungeonMonsterStatus 怪物状态
type TeamDungeonMonsterStatus struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	HP        int64  `json:"hp"`
	MaxHP     int64  `json:"max_hp"`
	Alive     bool   `json:"alive"`
}

// TeamDungeonTeamStatus 队伍状态
type TeamDungeonTeamStatus struct {
	Team          *TeamDungeonTeam        `json:"team"`
	Config        *TeamDungeonConfig      `json:"config,omitempty"`
	Members       []*TeamDungeonTeamMember `json:"members"`
	CurrentWave   int                     `json:"current_wave"`
	TotalWaves    int                     `json:"total_waves"`
	Status        int                     `json:"status"`
	StatusText    string                  `json:"status_text"`
	RemainingSec  int                     `json:"remaining_sec"`
}

// TeamDungeonInfo 队伍信息(公开)
type TeamDungeonInfo struct {
	TeamID        string                  `json:"team_id"`
	DungeonName   string                  `json:"dungeon_name"`
	LeaderID      string                  `json:"leader_id"`
	MemberCount   int                     `json:"member_count"`
	MaxPlayers    int                     `json:"max_players"`
	Status        int                     `json:"status"`
	StatusText    string                  `json:"status_text"`
	RealmRequired int                     `json:"realm_required"`
}

// ============================================================
// TeamDungeonService
// ============================================================

// TeamDungeonService 组队副本业务逻辑
type TeamDungeonService struct {
	mu        sync.RWMutex
	configs   map[int]*TeamDungeonConfig     // configID -> config
	teams     map[string]*TeamDungeonTeam    // teamID -> team
	invites   map[string]*TeamDungeonInvite  // inviteID -> invite
	playerTeams map[string]string            // playerID -> teamID (当前所在队伍)
	nextID    uint64
}

// NewTeamDungeonService 创建服务
func NewTeamDungeonService() *TeamDungeonService {
	svc := &TeamDungeonService{
		configs:     make(map[int]*TeamDungeonConfig),
		teams:       make(map[string]*TeamDungeonTeam),
		invites:     make(map[string]*TeamDungeonInvite),
		playerTeams: make(map[string]string),
		nextID:      1,
	}
	svc.loadConfigs()
	return svc
}

// loadConfigs 从JSON加载副本配置
func (s *TeamDungeonService) loadConfigs() {
	data, err := os.ReadFile("internal/data/team_dungeons.json")
	if err != nil {
		log.Warn().Err(err).Msg("[组队副本] 加载配置文件失败, 使用内置默认配置")
		s.initDefaultConfigs()
		return
	}

	var raw struct {
		Dungeons []*TeamDungeonConfig `json:"dungeons"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Warn().Err(err).Msg("[组队副本] 解析配置文件失败, 使用内置默认配置")
		s.initDefaultConfigs()
		return
	}

	for _, d := range raw.Dungeons {
		s.configs[d.ID] = d
	}
	log.Info().Int("count", len(s.configs)).Msg("[组队副本] 配置加载完成")
}

// initDefaultConfigs 内置默认配置(兜底)
func (s *TeamDungeonService) initDefaultConfigs() {
	s.configs = map[int]*TeamDungeonConfig{
		1: {
			ID: 1, Name: "青龙秘径", RealmRequired: 1,
			MinPlayers: 3, MaxPlayers: 3, TimeLimitMinutes: 10,
			Waves: []TeamDungeonWave{
				{Wave: 1, Name: "秘径入口", Monsters: []TeamDungeonMonster{
					{ID: 1001, Name: "青鳞蛇", HP: 500, Atk: 30, Def: 10, Speed: 20, Level: 5, Element: "wood"},
				}},
			},
			Boss: TeamDungeonBoss{
				ID: 1100, Name: "青龙兽", HP: 5000, Atk: 80, Def: 30, Speed: 25, Level: 10, Element: "wood",
			},
			Rewards: TeamDungeonReward{
				BaseExp: 500, BaseSpiritStones: 200,
			},
		},
	}
}

// ============================================================
// 配置查询
// ============================================================

// GetConfigs 获取所有副本配置
func (s *TeamDungeonService) GetConfigs() []*TeamDungeonConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]int, 0, len(s.configs))
	for id := range s.configs {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	result := make([]*TeamDungeonConfig, 0, len(ids))
	for _, id := range ids {
		result = append(result, s.configs[id])
	}
	return result
}

// GetConfig 获取指定配置
func (s *TeamDungeonService) GetConfig(configID int) *TeamDungeonConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configs[configID]
}

// GetActiveTeams 获取招募中的队伍列表
func (s *TeamDungeonService) GetActiveTeams(dungeonConfigID int) []*TeamDungeonInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*TeamDungeonInfo, 0)
	for _, team := range s.teams {
		if team.Status != TeamDungeonStatusRecruiting {
			continue
		}
		if dungeonConfigID > 0 && team.DungeonConfigID != dungeonConfigID {
			continue
		}
		cfg := s.configs[team.DungeonConfigID]
		maxPlayers := 3
		name := "未知副本"
		realmReq := 0
		if cfg != nil {
			maxPlayers = cfg.MaxPlayers
			name = cfg.Name
			realmReq = cfg.RealmRequired
		}
		result = append(result, &TeamDungeonInfo{
			TeamID:        team.ID,
			DungeonName:   name,
			LeaderID:      team.LeaderID,
			MemberCount:   len(team.Members),
			MaxPlayers:    maxPlayers,
			Status:        team.Status,
			StatusText:    "招募中",
			RealmRequired: realmReq,
		})
	}
	return result
}

// GetTeamInfo 获取队伍详细信息
func (s *TeamDungeonService) GetTeamInfo(teamID string) *TeamDungeonTeamStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, ok := s.teams[teamID]
	if !ok {
		return nil
	}

	cfg := s.configs[team.DungeonConfigID]
	var remainingSec int
	if team.Status == TeamDungeonStatusInProgress && !team.StartedAt.IsZero() {
		elapsed := int(time.Since(team.StartedAt).Seconds())
		remainingSec = team.TimeLimitSec - elapsed
		if remainingSec < 0 {
			remainingSec = 0
		}
	}

	totalWaves := 0
	if cfg != nil {
		totalWaves = len(cfg.Waves) + 1 // waves + boss
	}

	statusText := s.statusText(team.Status)

	return &TeamDungeonTeamStatus{
		Team:         team,
		Config:       cfg,
		Members:      team.Members,
		CurrentWave:  team.CurrentWave,
		TotalWaves:   totalWaves,
		Status:       team.Status,
		StatusText:   statusText,
		RemainingSec: remainingSec,
	}
}

// ============================================================
// 队伍管理
// ============================================================

// CreateTeam 创建队伍
func (s *TeamDungeonService) CreateTeam(req *TeamDungeonCreateReq) (*TeamDungeonTeam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查配置是否存在
	cfg, ok := s.configs[req.DungeonConfigID]
	if !ok {
		return nil, fmt.Errorf("副本配置不存在")
	}

	// 检查玩家是否已在队伍中
	if existingTeamID, ok := s.playerTeams[req.PlayerID]; ok {
		if team, exists := s.teams[existingTeamID]; exists && team.Status <= TeamDungeonStatusReady {
			return nil, fmt.Errorf("你已在其他队伍中, 请先退出")
		}
		// 清理过期引用
		delete(s.playerTeams, req.PlayerID)
	}

	// 校验位置
	position := req.Position
	if position < TeamDungeonPositionTank || position > TeamDungeonPositionSupport {
		position = TeamDungeonPositionDPS
	}

	teamID := uuid.New().String()
	now := time.Now()

	member := &TeamDungeonTeamMember{
		PlayerID:   req.PlayerID,
		PlayerName: req.PlayerName,
		Position:   position,
		Ready:      true, // 创建者默认就绪
		JoinedAt:   now,
	}

	team := &TeamDungeonTeam{
		ID:              teamID,
		DungeonConfigID: req.DungeonConfigID,
		LeaderID:        req.PlayerID,
		Status:          TeamDungeonStatusRecruiting,
		Members:         []*TeamDungeonTeamMember{member},
		CurrentWave:     0,
		TimeLimitSec:    cfg.TimeLimitMinutes * 60,
		CreatedAt:       now,
	}

	s.teams[teamID] = team
	s.playerTeams[req.PlayerID] = teamID

	log.Info().Str("team_id", teamID).Str("leader", req.PlayerID).
		Int("config_id", req.DungeonConfigID).Msg("[组队副本] 队伍已创建")

	return team, nil
}

// JoinTeam 加入队伍
func (s *TeamDungeonService) JoinTeam(req *TeamDungeonJoinReq) (*TeamDungeonTeam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[req.TeamID]
	if !ok {
		return nil, fmt.Errorf("队伍不存在")
	}

	if team.Status >= TeamDungeonStatusInProgress {
		return nil, fmt.Errorf("副本已开始, 无法加入")
	}

	cfg, ok := s.configs[team.DungeonConfigID]
	if !ok {
		return nil, fmt.Errorf("副本配置不存在")
	}

	// 检查人数上限
	if len(team.Members) >= cfg.MaxPlayers {
		return nil, fmt.Errorf("队伍已满(上限%d人)", cfg.MaxPlayers)
	}

	// 检查玩家是否已在队伍中
	if existingTeamID, ok := s.playerTeams[req.PlayerID]; ok {
		if existingTeamID == req.TeamID {
			return nil, fmt.Errorf("你已在该队伍中")
		}
		if eTeam, exists := s.teams[existingTeamID]; exists && eTeam.Status <= TeamDungeonStatusReady {
			return nil, fmt.Errorf("你已在其他队伍中, 请先退出")
		}
		delete(s.playerTeams, req.PlayerID)
	}

	// 检查是否重复加入
	for _, m := range team.Members {
		if m.PlayerID == req.PlayerID {
			return nil, fmt.Errorf("已在队伍中")
		}
	}

	// 校验位置
	position := req.Position
	if position < TeamDungeonPositionTank || position > TeamDungeonPositionSupport {
		position = TeamDungeonPositionDPS
	}

	member := &TeamDungeonTeamMember{
		PlayerID:   req.PlayerID,
		PlayerName: req.PlayerName,
		Position:   position,
		Ready:      false,
		JoinedAt:   time.Now(),
	}

	team.Members = append(team.Members, member)
	s.playerTeams[req.PlayerID] = team.ID

	log.Info().Str("team_id", req.TeamID).Str("player", req.PlayerID).
		Int("position", position).Msg("[组队副本] 玩家加入队伍")

	return team, nil
}

// LeaveTeam 离开队伍
func (s *TeamDungeonService) LeaveTeam(req *TeamDungeonLeaveReq) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[req.TeamID]
	if !ok {
		return fmt.Errorf("队伍不存在")
	}

	if team.Status >= TeamDungeonStatusInProgress {
		return fmt.Errorf("副本已开始, 无法退出")
	}

	// 检查成员资格
	found := false
	for i, m := range team.Members {
		if m.PlayerID == req.PlayerID {
			team.Members = append(team.Members[:i], team.Members[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("你不在该队伍中")
	}

	delete(s.playerTeams, req.PlayerID)

	// 如果队长离开, 转让队长
	if req.PlayerID == team.LeaderID {
		if len(team.Members) > 0 {
			team.LeaderID = team.Members[0].PlayerID
			team.Members[0].Ready = true
			log.Info().Str("team_id", req.TeamID).Str("new_leader", team.LeaderID).
				Msg("[组队副本] 队长已转让")
		} else {
			// 队伍空了, 删除
			delete(s.teams, req.TeamID)
			log.Info().Str("team_id", req.TeamID).Msg("[组队副本] 队伍已解散")
			return nil
		}
	}

	// 检查人数是否足够
	cfg, ok := s.configs[team.DungeonConfigID]
	if ok && len(team.Members) < cfg.MinPlayers {
		team.Status = TeamDungeonStatusRecruiting
	}

	// 重置所有成员的ready状态
	for _, m := range team.Members {
		m.Ready = false
	}
	team.Members[0].Ready = true // 队长默认就绪

	log.Info().Str("team_id", req.TeamID).Str("player", req.PlayerID).
		Msg("[组队副本] 玩家离开队伍")

	return nil
}

// ============================================================
// 邀请系统
// ============================================================

// InviteToTeam 邀请玩家加入
func (s *TeamDungeonService) InviteToTeam(req *TeamDungeonInviteReq) (*TeamDungeonInvite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[req.TeamID]
	if !ok {
		return nil, fmt.Errorf("队伍不存在")
	}

	if team.Status >= TeamDungeonStatusInProgress {
		return nil, fmt.Errorf("副本已开始, 无法邀请")
	}

	// 检查邀请者是否为队长或成员
	isMember := false
	for _, m := range team.Members {
		if m.PlayerID == req.InviterID {
			isMember = true
			break
		}
	}
	if !isMember {
		return nil, fmt.Errorf("你不在该队伍中")
	}

	// 检查是否已邀请
	for _, inv := range s.invites {
		if inv.TeamID == req.TeamID && inv.InviteeID == req.InviteeID && inv.Status == TeamDungeonInvitePending {
			return nil, fmt.Errorf("已向该玩家发送过邀请")
		}
	}

	// 检查被邀请人是否已在队伍中
	if _, exists := s.playerTeams[req.InviteeID]; exists {
		for _, t := range s.teams {
			if t.ID == s.playerTeams[req.InviteeID] && t.Status <= TeamDungeonStatusReady {
				return nil, fmt.Errorf("该玩家已在其他队伍中")
			}
		}
	}

	inviteID := uuid.New().String()
	now := time.Now()

	invite := &TeamDungeonInvite{
		ID:        inviteID,
		TeamID:    req.TeamID,
		InviterID: req.InviterID,
		InviteeID: req.InviteeID,
		Status:    TeamDungeonInvitePending,
		CreatedAt: now,
		ExpiresAt: now.Add(TeamDungeonInviteExpireSec * time.Second),
	}

	s.invites[inviteID] = invite

	log.Info().Str("invite_id", inviteID).Str("team_id", req.TeamID).
		Str("inviter", req.InviterID).Str("invitee", req.InviteeID).
		Msg("[组队副本] 邀请已发送")

	return invite, nil
}

// AcceptInvite 接受邀请
func (s *TeamDungeonService) AcceptInvite(inviteID, playerID string) (*TeamDungeonTeam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invites[inviteID]
	if !ok {
		return nil, fmt.Errorf("邀请不存在")
	}

	if inv.Status != TeamDungeonInvitePending {
		return nil, fmt.Errorf("邀请已处理")
	}

	if time.Now().After(inv.ExpiresAt) {
		inv.Status = TeamDungeonInviteExpired
		return nil, fmt.Errorf("邀请已过期")
	}

	if inv.InviteeID != playerID {
		return nil, fmt.Errorf("该邀请不是给你的")
	}

	team, ok := s.teams[inv.TeamID]
	if !ok {
		inv.Status = TeamDungeonInviteExpired
		return nil, fmt.Errorf("队伍已解散")
	}

	if team.Status >= TeamDungeonStatusInProgress {
		return nil, fmt.Errorf("副本已开始")
	}

	cfg := s.configs[team.DungeonConfigID]
	if cfg != nil && len(team.Members) >= cfg.MaxPlayers {
		return nil, fmt.Errorf("队伍已满")
	}

	// 检查是否已在其他队伍
	if existingTeamID, ok := s.playerTeams[playerID]; ok {
		if existingTeamID == inv.TeamID {
			return nil, fmt.Errorf("已在该队伍中")
		}
		if eTeam, exists := s.teams[existingTeamID]; exists && eTeam.Status <= TeamDungeonStatusReady {
			return nil, fmt.Errorf("你已在其他队伍中")
		}
	}

	inv.Status = TeamDungeonInviteAccepted

	member := &TeamDungeonTeamMember{
		PlayerID: playerID,
		Position: TeamDungeonPositionDPS,
		JoinedAt: time.Now(),
	}

	team.Members = append(team.Members, member)
	s.playerTeams[playerID] = team.ID

	log.Info().Str("invite_id", inviteID).Str("player", playerID).
		Str("team_id", inv.TeamID).Msg("[组队副本] 接受邀请")

	return team, nil
}

// DeclineInvite 拒绝邀请
func (s *TeamDungeonService) DeclineInvite(inviteID, playerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invites[inviteID]
	if !ok {
		return fmt.Errorf("邀请不存在")
	}

	if inv.InviteeID != playerID {
		return fmt.Errorf("该邀请不是给你的")
	}

	if inv.Status != TeamDungeonInvitePending {
		return nil // 已经处理过
	}

	inv.Status = TeamDungeonInviteDeclined

	log.Info().Str("invite_id", inviteID).Str("player", playerID).
		Msg("[组队副本] 拒绝邀请")

	return nil
}

// GetPendingInvites 获取玩家的待处理邀请
func (s *TeamDungeonService) GetPendingInvites(playerID string) []*TeamDungeonInvite {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	result := make([]*TeamDungeonInvite, 0)
	for _, inv := range s.invites {
		if inv.InviteeID != playerID || inv.Status != TeamDungeonInvitePending {
			continue
		}
		if now.After(inv.ExpiresAt) {
			inv.Status = TeamDungeonInviteExpired
			continue
		}
		result = append(result, inv)
	}
	return result
}

// ============================================================
// 就绪/开始
// ============================================================

// SetReady 设置就绪状态
func (s *TeamDungeonService) SetReady(req *TeamDungeonReadyReq) (*TeamDungeonTeam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[req.TeamID]
	if !ok {
		return nil, fmt.Errorf("队伍不存在")
	}

	if team.Status >= TeamDungeonStatusInProgress {
		return nil, fmt.Errorf("副本已开始")
	}

	for _, m := range team.Members {
		if m.PlayerID == req.PlayerID {
			m.Ready = req.Ready
			break
		}
	}

	// 检查是否所有人就绪
	allReady := len(team.Members) >= s.configs[team.DungeonConfigID].MinPlayers
	for _, m := range team.Members {
		if !m.Ready {
			allReady = false
			break
		}
	}

	if allReady {
		team.Status = TeamDungeonStatusReady
	} else {
		team.Status = TeamDungeonStatusRecruiting
	}

	log.Info().Str("team_id", req.TeamID).Str("player", req.PlayerID).
		Bool("ready", req.Ready).Msg("[组队副本] 就绪状态更新")

	return team, nil
}

// StartDungeon 开始副本
func (s *TeamDungeonService) StartDungeon(req *TeamDungeonStartReq) (*TeamDungeonTeam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[req.TeamID]
	if !ok {
		return nil, fmt.Errorf("队伍不存在")
	}

	if req.PlayerID != team.LeaderID {
		return nil, fmt.Errorf("只有队长才能开始副本")
	}

	if team.Status == TeamDungeonStatusInProgress {
		return nil, fmt.Errorf("副本已在进行中")
	}

	cfg, ok := s.configs[team.DungeonConfigID]
	if !ok {
		return nil, fmt.Errorf("副本配置不存在")
	}

	// 检查人数
	if len(team.Members) < cfg.MinPlayers {
		return nil, fmt.Errorf("人数不足, 需要至少%d人", cfg.MinPlayers)
	}

	// 检查是否都就绪
	for _, m := range team.Members {
		if !m.Ready {
			return nil, fmt.Errorf("成员 %s 未就绪", m.PlayerID)
		}
	}

	// 初始化副本实例
	waveMonsters := make([]*TeamDungeonWaveMonster, 0)
	for _, w := range cfg.Waves {
		if w.Wave == 1 {
			for _, m := range w.Monsters {
				mCopy := m
				waveMonsters = append(waveMonsters, &TeamDungeonWaveMonster{
					Config:    mCopy,
					CurrentHP: mCopy.HP,
					Alive:     true,
				})
			}
		}
	}

	bossCopy := cfg.Boss
	bossInstance := &TeamDungeonBossInstance{
		Config:    bossCopy,
		CurrentHP: bossCopy.HP,
		Alive:     true,
		Skills:    bossCopy.Skills,
	}

	team.Instance = &TeamDungeonInstance{
		TeamID:       team.ID,
		CurrentWave:  1,
		TotalWaves:   len(cfg.Waves),
		WaveMonsters: waveMonsters,
		Boss:         bossInstance,
		StartedAt:    time.Now(),
	}
	team.CurrentWave = 1
	team.Status = TeamDungeonStatusInProgress
	team.StartedAt = time.Now()

	log.Info().Str("team_id", team.ID).Int("config_id", team.DungeonConfigID).
		Int("members", len(team.Members)).Msg("[组队副本] 副本开始!")

	return team, nil
}

// ============================================================
// 副本战斗
// ============================================================

// ProcessWave 处理当前波次(所有成员行动)
func (s *TeamDungeonService) ProcessWave(teamID string, memberActions []*TeamDungeonAttackReq) (*TeamDungeonWaveResult, error) {
	s.mu.Lock()
	team, ok := s.teams[teamID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("队伍不存在")
	}

	if team.Status != TeamDungeonStatusInProgress {
		s.mu.Unlock()
		return nil, fmt.Errorf("副本不在进行中")
	}

	instance := team.Instance
	if instance == nil || instance.Completed {
		s.mu.Unlock()
		return nil, fmt.Errorf("副本实例无效或已完成")
	}

	// 检查是否超时
	elapsed := int(time.Since(team.StartedAt).Seconds())
	if elapsed >= team.TimeLimitSec {
		team.Status = TeamDungeonStatusFailed
		team.CompletedAt = time.Now()
		s.mu.Unlock()
		return nil, fmt.Errorf("副本超时, 挑战失败")
	}

	cfg := s.configs[team.DungeonConfigID]
	if cfg == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("配置不存在")
	}

	currentWave := instance.CurrentWave

	// 复制一份用于战斗计算
	memberActionsCopy := make([]*TeamDungeonAttackReq, len(memberActions))
	copy(memberActionsCopy, memberActions)
	waveMonstersCopy := make([]*TeamDungeonWaveMonster, len(instance.WaveMonsters))
	for i, m := range instance.WaveMonsters {
		mCopy := *m
		waveMonstersCopy[i] = &mCopy
	}
	bossCopy := *instance.Boss

	s.mu.Unlock()

	// 执行战斗
	result := s.executeWave(currentWave, cfg, team.Members, memberActionsCopy, waveMonstersCopy, &bossCopy)

	// 更新状态
	s.mu.Lock()
	defer s.mu.Unlock()

	// 重新获取实例(防止并发修改)
	currentInstance := team.Instance
	if currentInstance == nil || currentInstance.CurrentWave != currentWave {
		return nil, fmt.Errorf("波次状态已变更")
	}

	// 更新怪物状态
	endBoss := result.IsBossWave
	for i, m := range result.Monsters {
		if i < len(currentInstance.WaveMonsters) {
			currentInstance.WaveMonsters[i].CurrentHP = m.HP
			currentInstance.WaveMonsters[i].Alive = m.Alive
		}
	}

	if endBoss {
		currentInstance.Boss.CurrentHP = bossCopy.CurrentHP
		currentInstance.Boss.Alive = bossCopy.Alive
		if !bossCopy.Alive {
			currentInstance.BossDefeated = true
		}
	}

	// 更新成员贡献
	for _, action := range result.Actions {
		for _, m := range team.Members {
			if m.PlayerID == action.PlayerID {
				m.DamageDealt += action.DamageDealt
				m.HealingDone += action.HealingDone
				m.SupportProvided += action.SupportValue
				team.TotalDamage += action.DamageDealt
				break
			}
		}
	}

	elapsed = int(time.Since(team.StartedAt).Seconds())
	currentInstance.ElapsedSec = elapsed

	if result.Cleared {
		if endBoss && result.BossDefeated {
			// Boss击败, 副本完成
			currentInstance.Completed = true
			currentInstance.BossDefeated = true
			team.Status = TeamDungeonStatusCompleted
			team.CompletedAt = time.Now()
			log.Info().Str("team_id", teamID).Msg("[组队副本] 副本通关!")
		} else {
			// 进入下一波
			nextWave := currentWave + 1
			if nextWave <= len(cfg.Waves) {
				currentInstance.CurrentWave = nextWave
				team.CurrentWave = nextWave
				// 加载下一波怪物
				currentInstance.WaveMonsters = s.loadWaveMonsters(cfg.Waves, nextWave)
			} else if nextWave > len(cfg.Waves) {
				// 所有波次完成, 进入Boss战
				currentInstance.CurrentWave = nextWave
				team.CurrentWave = nextWave
				result.IsBossWave = true
				result.WaveName = "Boss战: " + cfg.Boss.Name
			}
		}
	}

	result.Logs = s.buildWaveLogs(result, cfg)

	return result, nil
}

// executeWave 执行波次战斗(模拟)
func (s *TeamDungeonService) executeWave(
	wave int,
	cfg *TeamDungeonConfig,
	members []*TeamDungeonTeamMember,
	actions []*TeamDungeonAttackReq,
	monsters []*TeamDungeonWaveMonster,
	boss *TeamDungeonBossInstance,
) *TeamDungeonWaveResult {
	isBossWave := wave > len(cfg.Waves)
	waveName := ""
	if isBossWave {
		waveName = "Boss战: " + cfg.Boss.Name
	} else if wave >= 1 && wave <= len(cfg.Waves) {
		waveName = cfg.Waves[wave-1].Name
	}

	result := &TeamDungeonWaveResult{
		Wave:       wave,
		WaveName:   waveName,
		Cleared:    false,
		IsBossWave: isBossWave,
		Actions:    make([]*TeamDungeonAttackResult, 0),
		Logs:       make([]string, 0),
	}

	// 统计各元素使用(用于连击判定)
	elementCounts := make(map[string]int)

	// 执行每个成员的行动
	for _, action := range actions {
		// 查找成员
		var member *TeamDungeonTeamMember
		for _, m := range members {
			if m.PlayerID == action.PlayerID {
				member = m
				break
			}
		}
		if member == nil {
			continue
		}

		actionResult := &TeamDungeonAttackResult{
			PlayerID: action.PlayerID,
		}

		// 应用阵型加成
		formationMult := s.getFormationMultiplier(member.Position)

		if action.IsHeal {
			// 治疗行动
			baseHeal := int64(50 + rand.Int63n(30))
			healing := int64(float64(baseHeal) * formationMult)
			actionResult.HealingDone = healing
			actionResult.Log = fmt.Sprintf("%s 进行治疗, 恢复 %d 点生命", member.PlayerName, healing)
		} else if action.IsSupport {
			// 辅助行动(加buff)
			baseSupport := int64(30 + rand.Int63n(20))
			support := int64(float64(baseSupport) * formationMult)
			actionResult.SupportValue = support
			actionResult.Log = fmt.Sprintf("%s 释放辅助技能, 提供 %d 点增益", member.PlayerName, support)
		} else {
			// 攻击行动
			// 统计同元素
			if action.Element != "" {
				elementCounts[action.Element]++
			}

			// 找活的怪物作为目标
			var target *TeamDungeonWaveMonster
			for _, m := range monsters {
				if m.Alive {
					target = m
					break
				}
			}

			if target == nil && isBossWave && boss.Alive {
				// 没有小怪, 攻击Boss
				baseDmg := int64(80 + rand.Int63n(40))
				dmg := int64(float64(baseDmg) * formationMult)

				// 暴击判定
				critical := rand.Float64() < 0.15
				if critical {
					dmg = int64(float64(dmg) * 1.8)
				}

				if dmg > boss.CurrentHP {
					dmg = boss.CurrentHP
				}
				boss.CurrentHP -= dmg
				if boss.CurrentHP <= 0 {
					boss.CurrentHP = 0
					boss.Alive = false
				}

				actionResult.DamageDealt = dmg
				actionResult.Critical = critical
				actionResult.TargetID = boss.Config.ID
				actionResult.TargetName = boss.Config.Name
				actionResult.MonsterKilled = !boss.Alive
				actionResult.Log = fmt.Sprintf("%s 对 %s 造成 %d 点伤害", member.PlayerName, boss.Config.Name, dmg)

				// Boss反击
				for _, m := range members {
					if m.PlayerID != action.PlayerID {
						// 简单模拟Boss反击伤害
						bossSkill := ""
						if len(boss.Skills) > 0 && rand.Float64() < 0.3 {
							skill := boss.Skills[rand.Intn(len(boss.Skills))]
							if skill.DamageMult > 0 {
								bossSkill = fmt.Sprintf("Boss 使用 %s!", skill.Name)
							}
						}
						if bossSkill != "" {
							actionResult.Log += " | " + bossSkill
						}
						break
					}
				}
			} else if target != nil && target.Alive {
				// 攻击小怪
				baseDmg := int64(80 + rand.Int63n(40))
				dmg := int64(float64(baseDmg) * formationMult)

				// 暴击判定
				critical := rand.Float64() < 0.15
				if critical {
					dmg = int64(float64(dmg) * 1.8)
				}

				// 计算防御减伤
				dmg = int64(float64(dmg) * (1.0 - float64(target.Config.Def)/200.0))
				if dmg < 1 {
					dmg = 1
				}

				if dmg > target.CurrentHP {
					dmg = target.CurrentHP
				}
				target.CurrentHP -= dmg
				if target.CurrentHP <= 0 {
					target.CurrentHP = 0
					target.Alive = false
				}

				actionResult.DamageDealt = dmg
				actionResult.Critical = critical
				actionResult.TargetID = target.Config.ID
				actionResult.TargetName = target.Config.Name
				actionResult.MonsterKilled = !target.Alive
				actionResult.Log = fmt.Sprintf("%s 对 %s 造成 %d 点伤害", member.PlayerName, target.Config.Name, dmg)
			} else {
				actionResult.Log = fmt.Sprintf("%s 没有可攻击的目标", member.PlayerName)
			}
		}

		result.Actions = append(result.Actions, actionResult)
	}

	// 计算连击加成
	for idx, action := range result.Actions {
		comboCount := 0
		comboElement := ""
		if action.DamageDealt > 0 && idx < len(actions) {
			reqElement := actions[idx].Element
			if reqElement != "" {
				comboElement = reqElement
				if elementCounts[reqElement] >= TeamDungeonComboCountMin {
					comboCount = elementCounts[reqElement]
				}
			}
		}

		if comboCount >= TeamDungeonComboCountMin {
			comboBonus := int64(float64(action.DamageDealt) * (TeamDungeonComboMultiplier - 1.0))
			action.DamageDealt += comboBonus
			action.ComboActive = true
			action.ComboCount = comboCount
			action.Log += fmt.Sprintf(" [连击! %s元素x%d, 额外伤害+%d]", comboElement, comboCount, comboBonus)
		}
	}

	// 判定波次是否通过
	allDead := true
	for _, m := range monsters {
		if m.Alive {
			allDead = false
			break
		}
	}
	if isBossWave {
		allDead = allDead && !boss.Alive
		result.BossDefeated = !boss.Alive
	}

	// 构建怪物状态
	monsterStatuses := make([]TeamDungeonMonsterStatus, 0)
	for _, m := range monsters {
		monsterStatuses = append(monsterStatuses, TeamDungeonMonsterStatus{
			ID: m.Config.ID, Name: m.Config.Name,
			HP: m.CurrentHP, MaxHP: m.Config.HP, Alive: m.Alive,
		})
	}

	result.Monsters = monsterStatuses
	result.Cleared = allDead

	return result
}

// loadWaveMonsters 加载指定波次的怪物
func (s *TeamDungeonService) loadWaveMonsters(waves []TeamDungeonWave, wave int) []*TeamDungeonWaveMonster {
	monsters := make([]*TeamDungeonWaveMonster, 0)
	for _, w := range waves {
		if w.Wave == wave {
			for _, m := range w.Monsters {
				mCopy := m
				monsters = append(monsters, &TeamDungeonWaveMonster{
					Config:    mCopy,
					CurrentHP: mCopy.HP,
					Alive:     true,
				})
			}
			break
		}
	}
	return monsters
}

// getFormationMultiplier 获取阵型加成
func (s *TeamDungeonService) getFormationMultiplier(position int) float64 {
	switch position {
	case TeamDungeonPositionTank:
		return TeamDungeonFormationBonusTank
	case TeamDungeonPositionDPS:
		return TeamDungeonFormationBonusDPS
	case TeamDungeonPositionSupport:
		return TeamDungeonFormationBonusSupport
	default:
		return 1.0
	}
}

// buildWaveLogs 构建波次日志
func (s *TeamDungeonService) buildWaveLogs(result *TeamDungeonWaveResult, cfg *TeamDungeonConfig) []string {
	logs := make([]string, 0)
	logs = append(logs, fmt.Sprintf("--- 第%d波: %s ---", result.Wave, result.WaveName))

	for _, action := range result.Actions {
		logs = append(logs, action.Log)
	}

	if result.Cleared {
		logs = append(logs, fmt.Sprintf("第%d波通过!", result.Wave))
		if result.IsBossWave && result.BossDefeated {
			logs = append(logs, fmt.Sprintf("Boss %s 已被击败!", cfg.Boss.Name))
			logs = append(logs, "副本通关!")
		}
	} else {
		logs = append(logs, "波次进行中, 继续战斗...")
	}

	return logs
}

// ============================================================
// 副本完成/结算
// ============================================================

// CompleteDungeon 完成副本并结算奖励
func (s *TeamDungeonService) CompleteDungeon(teamID string) (*TeamDungeonCompletion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("队伍不存在")
	}

	if team.Status != TeamDungeonStatusInProgress && team.Status != TeamDungeonStatusCompleted {
		return nil, fmt.Errorf("副本状态不正确")
	}

	instance := team.Instance
	if instance == nil {
		return nil, fmt.Errorf("副本实例不存在")
	}

	cfg, ok := s.configs[team.DungeonConfigID]
	if !ok {
		return nil, fmt.Errorf("副本配置不存在")
	}

	// 标记完成
	if team.Status != TeamDungeonStatusCompleted {
		if instance.BossDefeated || instance.Completed {
			team.Status = TeamDungeonStatusCompleted
		} else {
			team.Status = TeamDungeonStatusFailed
		}
	}
	team.CompletedAt = time.Now()

	// 计算用时
	elapsed := int(team.CompletedAt.Sub(team.StartedAt).Seconds())
	if instance.ElapsedSec > 0 {
		elapsed = instance.ElapsedSec
	}

	// 检查是否有阵亡 (简化: 如果某成员伤害为0且治疗为0且辅助为0, 视为阵亡)
	noDeaths := true
	for _, m := range team.Members {
		if m.DamageDealt == 0 && m.HealingDone == 0 && m.SupportProvided == 0 {
			noDeaths = false
			break
		}
	}

	completed := team.Status == TeamDungeonStatusCompleted
	bossDefeated := instance.BossDefeated

	// 计算成员奖励
	memberRewards := s.calculateMemberRewards(team, cfg, completed, bossDefeated, noDeaths, elapsed)

	completion := &TeamDungeonCompletion{
		Completed:     completed,
		BossDefeated:  bossDefeated,
		TotalDamage:   team.TotalDamage,
		TimeUsedSec:   elapsed,
		NoDeaths:      noDeaths,
		MemberRewards: memberRewards,
	}

	team.Completion = completion

	// 清理playerTeams引用(完成/失败后玩家可以加入新队伍)
	for _, m := range team.Members {
		delete(s.playerTeams, m.PlayerID)
	}

	log.Info().Str("team_id", teamID).Bool("completed", completed).
		Bool("boss_defeated", bossDefeated).Int64("total_damage", team.TotalDamage).
		Msg("[组队副本] 副本结算完成")

	return completion, nil
}

// calculateMemberRewards 计算每个成员的奖励
func (s *TeamDungeonService) calculateMemberRewards(
	team *TeamDungeonTeam,
	cfg *TeamDungeonConfig,
	completed bool,
	bossDefeated bool,
	noDeaths bool,
	elapsedSec int,
) []*TeamDungeonMemberReward {
	if !completed || !bossDefeated {
		// 失败只有少量参与奖励
		rewards := make([]*TeamDungeonMemberReward, len(team.Members))
		for i, m := range team.Members {
			rewards[i] = &TeamDungeonMemberReward{
				PlayerID: m.PlayerID,
				Exp:      cfg.Rewards.BaseExp / 4,
				SpiritStones: cfg.Rewards.BaseSpiritStones / 4,
				Items:    make([]TeamDungeonRewardItem, 0),
			}
		}
		return rewards
	}

	// 计算总贡献
	totalContribution := int64(0)
	for _, m := range team.Members {
		contribution := int64(float64(m.DamageDealt)*cfg.Rewards.ContributionRewards.DamageWeight) +
			int64(float64(m.HealingDone)*cfg.Rewards.ContributionRewards.HealingWeight) +
			int64(float64(m.SupportProvided)*cfg.Rewards.ContributionRewards.SupportWeight)
		m.Contribution = contribution
		totalContribution += contribution
	}

	if totalContribution == 0 {
		totalContribution = 1
	}

	// 基础奖励
	baseExp := cfg.Rewards.BaseExp
	baseStones := cfg.Rewards.BaseSpiritStones

	// 特殊奖励加成
	if noDeaths {
		baseExp += cfg.Rewards.SpecialBonus.NoDeathBonusExp
	}
	if completed && elapsedSec <= team.TimeLimitSec/2 {
		baseExp += cfg.Rewards.SpecialBonus.TimeBonusExp
	}

	rewards := make([]*TeamDungeonMemberReward, len(team.Members))
	for i, m := range team.Members {
		contributionPct := float64(m.Contribution) / float64(totalContribution)

		// 按贡献分配
		memberExp := int64(float64(baseExp) * contributionPct)
		memberStones := int64(float64(baseStones) * contributionPct)
		// 确保最少奖励
		if memberExp < 10 {
			memberExp = 10
		}
		if memberStones < 5 {
			memberStones = 5
		}

		// 复制物品(每个成员都获得一份)
		items := make([]TeamDungeonRewardItem, len(cfg.Rewards.Items))
		copy(items, cfg.Rewards.Items)

		rewards[i] = &TeamDungeonMemberReward{
			PlayerID:         m.PlayerID,
			Exp:              memberExp,
			SpiritStones:     memberStones,
			Items:            items,
			ContributionPct:  math.Round(contributionPct*10000) / 100,
		}
	}

	return rewards
}

// ClaimMemberRewards 标记成员奖励已领取
func (s *TeamDungeonService) ClaimMemberRewards(teamID string, playerID string) (*TeamDungeonMemberReward, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("队伍不存在")
	}

	if team.Completion == nil {
		return nil, fmt.Errorf("副本尚未结算")
	}

	for _, mr := range team.Completion.MemberRewards {
		if mr.PlayerID == playerID {
			for _, m := range team.Members {
				if m.PlayerID == playerID {
					if m.RewardsClaimed {
						return nil, fmt.Errorf("奖励已领取")
					}
					m.RewardsClaimed = true
					return mr, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("未参与此副本")
}

// ============================================================
// 工具方法
// ============================================================

func (s *TeamDungeonService) statusText(status int) string {
	switch status {
	case TeamDungeonStatusRecruiting:
		return "招募中"
	case TeamDungeonStatusReady:
		return "已就绪"
	case TeamDungeonStatusInProgress:
		return "进行中"
	case TeamDungeonStatusCompleted:
		return "已完成"
	case TeamDungeonStatusFailed:
		return "已失败"
	default:
		return "未知"
	}
}

// CleanExpiredInvites 清理过期邀请
func (s *TeamDungeonService) CleanExpiredInvites() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, inv := range s.invites {
		if inv.Status == TeamDungeonInvitePending && now.After(inv.ExpiresAt) {
			inv.Status = TeamDungeonInviteExpired
			log.Debug().Str("invite_id", id).Msg("[组队副本] 邀请已过期")
		}
	}
}

// StartCleanupTask 启动定时清理任务
func (s *TeamDungeonService) StartCleanupTask() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.CleanExpiredInvites()
			s.cleanupStaleTeams()
		}
	}()
	log.Info().Msg("[组队副本] 定时清理任务已启动")
}

// cleanupStaleTeams 清理过期的招募队伍(超过30分钟无人加入)
func (s *TeamDungeonService) cleanupStaleTeams() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, team := range s.teams {
		if team.Status == TeamDungeonStatusRecruiting && now.Sub(team.CreatedAt) > 30*time.Minute {
			for _, m := range team.Members {
				delete(s.playerTeams, m.PlayerID)
			}
			delete(s.teams, id)
			log.Debug().Str("team_id", id).Msg("[组队副本] 招募超时, 队伍已清理")
		}
		// 清理已完成/失败的队伍(2小时后)
		if (team.Status == TeamDungeonStatusCompleted || team.Status == TeamDungeonStatusFailed) &&
			!team.CompletedAt.IsZero() && now.Sub(team.CompletedAt) > 2*time.Hour {
			delete(s.teams, id)
			log.Debug().Str("team_id", id).Msg("[组队副本] 历史记录已清理")
		}
	}
}
