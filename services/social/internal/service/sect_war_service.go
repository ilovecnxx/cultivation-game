// Package service 宗门战争系统 (PVP guild warfare)
//
// 赛季制 (30天): 报名期(7d) → 战争期(21d) → 休战期(2d)
// 比赛形式: 3轮制淘汰赛, 每轮 best-of-3
// 领地系统: 灵脉争夺 (1-5星品质)
// 匹配机制: 按宗门等级+平均境界匹配
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"cultivation-game/services/social/internal/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ============================================================
// 类型定义
// ============================================================

// WarPhase 战争阶段
type WarPhase string

const (
	WarPhaseRegistration WarPhase = "registration" // 报名期
	WarPhaseWar          WarPhase = "war"          // 战争期
	WarPhaseRest         WarPhase = "rest"         // 休战期
)

// MatchStatus 比赛状态
type MatchStatus string

const (
	MatchPending  MatchStatus = "pending"
	MatchFighting MatchStatus = "fighting"
	MatchFinished MatchStatus = "finished"
)

// MatchResult 比赛结果
type MatchResult string

const (
	MatchResultAWin MatchResult = "sect_a_win"
	MatchResultBWin MatchResult = "sect_b_win"
	MatchResultDraw MatchResult = "draw"
)

// BracketStatus 赛程状态
type BracketStatus string

const (
	BracketPending   BracketStatus = "pending"
	BracketActive    BracketStatus = "active"
	BracketFinished  BracketStatus = "finished"
)

// WarSeason 战争赛季
type WarSeason struct {
	ID              string        `bson:"_id" json:"id"`
	SeasonNumber    int           `bson:"season_number" json:"season_number"`
	Phase           WarPhase      `bson:"phase" json:"phase"`
	StartTime       time.Time     `bson:"start_time" json:"start_time"`
	RegistrationEnd time.Time     `bson:"registration_end" json:"registration_end"`
	WarEndTime      time.Time     `bson:"war_end_time" json:"war_end_time"`
	EndTime         time.Time     `bson:"end_time" json:"end_time"`
	RegisteredSects []string      `bson:"registered_sects,omitempty" json:"registered_sects,omitempty"`
	BracketIDs      []string      `bson:"bracket_ids,omitempty" json:"bracket_ids,omitempty"`
	Status          string        `bson:"status" json:"status"` // upcoming / active / settled
	CreatedAt       time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time     `bson:"updated_at" json:"updated_at"`
}

// WarBracket 战争赛程( bracket )
type WarBracket struct {
	ID        string        `bson:"_id" json:"id"`
	SeasonID  string        `bson:"season_id" json:"season_id"`
	Name      string        `bson:"name" json:"name"`
	Sects     []string      `bson:"sects" json:"sects"`
	Rounds    []*WarRound   `bson:"rounds,omitempty" json:"rounds,omitempty"`
	Status    BracketStatus `bson:"status" json:"status"`
	Winner    string        `bson:"winner,omitempty" json:"winner,omitempty"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}

// WarRound 战争轮次
type WarRound struct {
	RoundNumber int         `bson:"round_number" json:"round_number"`
	Matches     []*WarMatch `bson:"matches" json:"matches"`
	StartTime   time.Time   `bson:"start_time" json:"start_time"`
	EndTime     time.Time   `bson:"end_time,omitempty" json:"end_time,omitempty"`
}

// WarMatch 对抗赛
type WarMatch struct {
	ID          string       `bson:"_id" json:"id"`
	SeasonID    string       `bson:"season_id" json:"season_id"`
	BracketID   string       `bson:"bracket_id" json:"bracket_id"`
	RoundNumber int          `bson:"round_number" json:"round_number"`
	SectA       string       `bson:"sect_a" json:"sect_a"`
	SectAName   string       `bson:"sect_a_name" json:"sect_a_name"`
	SectB       string       `bson:"sect_b" json:"sect_b"`
	SectBName   string       `bson:"sect_b_name" json:"sect_b_name"`
	ScoreA      int          `bson:"score_a" json:"score_a"`
	ScoreB      int          `bson:"score_b" json:"score_b"`
	Status      MatchStatus  `bson:"status" json:"status"`
	Result      MatchResult  `bson:"result,omitempty" json:"result,omitempty"`
	Territory   string       `bson:"territory,omitempty" json:"territory,omitempty"` // 争夺的灵脉ID
	StartTime   time.Time    `bson:"start_time" json:"start_time"`
	EndTime     time.Time    `bson:"end_time,omitempty" json:"end_time,omitempty"`
	CreatedAt   time.Time    `bson:"created_at" json:"created_at"`
	// 回合详细记录(3轮制)
	Rounds []*MatchRound `bson:"rounds,omitempty" json:"rounds,omitempty"`
}

// MatchRound 单场比赛的回合记录
type MatchRound struct {
	RoundIndex   int    `bson:"round_index" json:"round_index"`
	SectAPlayer  string `bson:"sect_a_player" json:"sect_a_player"`
	SectBPlayer  string `bson:"sect_b_player" json:"sect_b_player"`
	SectARole    string `bson:"sect_a_role" json:"sect_a_role"` // attack / defend
	SectBRole    string `bson:"sect_b_role" json:"sect_b_role"`
	WinnerSect   string `bson:"winner_sect,omitempty" json:"winner_sect,omitempty"`
	WinnerPlayer string `bson:"winner_player,omitempty" json:"winner_player,omitempty"`
	Status       string `bson:"status" json:"status"` // pending / fighting / finished
}

// SectRanking 宗门排名
type SectRanking struct {
	SectID       string `bson:"sect_id" json:"sect_id"`
	SectName     string `bson:"sect_name" json:"sect_name"`
	SectLevel    int    `bson:"sect_level" json:"sect_level"`
	Score        int    `bson:"score" json:"score"`
	Wins         int    `bson:"wins" json:"wins"`
	Losses       int    `bson:"losses" json:"losses"`
	TotalMatches int    `bson:"total_matches" json:"total_matches"`
	Rank         int    `bson:"rank" json:"rank"`
	VeinCount    int    `bson:"vein_count" json:"vein_count"`
	MVPScore     int    `bson:"mvp_score" json:"mvp_score"`
}

// SeasonReward 赛季奖励配置
type SeasonReward struct {
	RankMin        int    `json:"rank_min"`
	RankMax        int    `json:"rank_max"`
	SectFunds      int64  `json:"sect_funds"`
	Reputation     int64  `json:"reputation"`
	Contribution   int64  `json:"contribution"`
	TitleID        string `json:"title_id,omitempty"`
	TitleName      string `json:"title_name,omitempty"`
	ArtifactID     string `json:"artifact_id,omitempty"`
	SectLevelBoost int    `json:"sect_level_boost,omitempty"`
}

// WarConfig 战争系统配置
type WarConfig struct {
	SeasonDays         int
	RegistrationDays   int
	WarDays            int
	RestDays           int
	MinMembersForWar   int
	MinSectLevel       int
	PlayersPerMatch    int
	MatchIntervalHours int
	BracketSizes       []int
	WinScore           int
	LoseScore          int
	DrawScore          int
}

// DefaultWarConfig 默认战争配置
func DefaultWarConfig() WarConfig {
	return WarConfig{
		SeasonDays:         30,
		RegistrationDays:   7,
		WarDays:            21,
		RestDays:           2,
		MinMembersForWar:   5,
		MinSectLevel:       3,
		PlayersPerMatch:    3,
		MatchIntervalHours: 24,
		BracketSizes:       []int{8, 16, 32},
		WinScore:           3,
		LoseScore:          0,
		DrawScore:          1,
	}
}

// SeasonRewards 赛季奖励表
var SeasonRewards = []SeasonReward{
	{RankMin: 1, RankMax: 1, SectFunds: 100000, Reputation: 5000, Contribution: 1000, TitleID: "title_war_1", TitleName: "万宗至尊", ArtifactID: "artifact_war_1", SectLevelBoost: 2},
	{RankMin: 2, RankMax: 2, SectFunds: 50000, Reputation: 3000, Contribution: 600, TitleID: "title_war_2", TitleName: "九州霸主", ArtifactID: "artifact_war_2", SectLevelBoost: 1},
	{RankMin: 3, RankMax: 3, SectFunds: 30000, Reputation: 2000, Contribution: 400, TitleID: "title_war_3", TitleName: "一方枭雄", SectLevelBoost: 1},
	{RankMin: 4, RankMax: 8, SectFunds: 15000, Reputation: 1000, Contribution: 200},
	{RankMin: 9, RankMax: 16, SectFunds: 8000, Reputation: 500, Contribution: 100},
	{RankMin: 17, RankMax: 999, SectFunds: 3000, Reputation: 200, Contribution: 50},
}

// ============================================================
// 灵脉品质加成表
// ============================================================

var VeinBonusesByQuality = map[int]SpiritVeinBonus{
	1: {CultivationSpeed: 5.0, BreakthroughRate: 0, SpiritStoneYield: 500},
	2: {CultivationSpeed: 10.0, BreakthroughRate: 0, SpiritStoneYield: 1000},
	3: {CultivationSpeed: 15.0, BreakthroughRate: 3.0, SpiritStoneYield: 2000},
	4: {CultivationSpeed: 20.0, BreakthroughRate: 6.0, SpiritStoneYield: 3000},
	5: {CultivationSpeed: 25.0, BreakthroughRate: 10.0, SpiritStoneYield: 5000},
}

// ============================================================
// SectWarService 宗门战争服务
// ============================================================

// SectWarService 宗门战争业务
type SectWarService struct {
	logger    *slog.Logger
	db        *mongo.Database
	cfg       WarConfig
	mu        sync.RWMutex
}

// NewSectWarService 创建宗门战争服务
func NewSectWarService(logger *slog.Logger, db *mongo.Database) *SectWarService {
	return &SectWarService{
		logger: logger,
		db:     db,
		cfg:    DefaultWarConfig(),
	}
}

// NewSectWarServiceWithConfig 使用自定义配置创建
func NewSectWarServiceWithConfig(logger *slog.Logger, db *mongo.Database, cfg WarConfig) *SectWarService {
	return &SectWarService{
		logger: logger,
		db:     db,
		cfg:    cfg,
	}
}

// collection helpers
func (s *SectWarService) seasonColl() *mongo.Collection  { return s.db.Collection("war_seasons") }
func (s *SectWarService) bracketColl() *mongo.Collection { return s.db.Collection("war_brackets") }
func (s *SectWarService) matchColl() *mongo.Collection   { return s.db.Collection("war_matches") }
func (s *SectWarService) veinColl() *mongo.Collection    { return s.db.Collection("spirit_veins") }
func (s *SectWarService) rankingColl() *mongo.Collection { return s.db.Collection("war_rankings") }
func (s *SectWarService) sectColl() *mongo.Collection    { return s.db.Collection("sects") }
func (s *SectWarService) memberColl() *mongo.Collection  { return s.db.Collection("sect_members") }

// ============================================================
// 1. 赛季管理
// ============================================================

// GetCurrentSeason 获取当前赛季，必要时自动创建
func (s *SectWarService) GetCurrentSeason(ctx context.Context) (*WarSeason, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "season_number", Value: -1}})
	var season WarSeason
	err := s.seasonColl().FindOne(ctx, bson.M{}, opts).Decode(&season)
	if err == nil {
		// 赛季在进行中
		if time.Now().Before(season.EndTime) {
			// 更新阶段
			season.Phase = s.determinePhase(season)
			return &season, nil
		}
		// 赛季已结束
	}

	return s.StartNewSeason(ctx)
}

// StartNewSeason 开启新赛季
func (s *SectWarService) StartNewSeason(ctx context.Context) (*WarSeason, error) {
	var lastSeason WarSeason
	opts := options.FindOne().SetSort(bson.D{{Key: "season_number", Value: -1}})
	_ = s.seasonColl().FindOne(ctx, bson.M{}, opts).Decode(&lastSeason)

	newSeasonNum := lastSeason.SeasonNumber + 1
	now := time.Now()

	season := &WarSeason{
		ID:              uuid.New().String(),
		SeasonNumber:    newSeasonNum,
		Phase:           WarPhaseRegistration,
		StartTime:       now,
		RegistrationEnd: now.AddDate(0, 0, s.cfg.RegistrationDays),
		WarEndTime:      now.AddDate(0, 0, s.cfg.RegistrationDays+s.cfg.WarDays),
		EndTime:         now.AddDate(0, 0, s.cfg.SeasonDays),
		RegisteredSects: []string{},
		BracketIDs:      []string{},
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err := s.seasonColl().InsertOne(ctx, season)
	if err != nil {
		return nil, fmt.Errorf("开启新赛季失败: %w", err)
	}

	s.logger.Info("新赛季开启", "season",
		season.SeasonNumber, season.RegistrationEnd.Format("2006-01-02"),
		season.WarEndTime.Format("2006-01-02"))

	return season, nil
}

// determinePhase 根据当前时间判定赛季阶段
func (s *SectWarService) determinePhase(season WarSeason) WarPhase {
	now := time.Now()
	if now.Before(season.RegistrationEnd) {
		return WarPhaseRegistration
	}
	if now.Before(season.WarEndTime) {
		return WarPhaseWar
	}
	return WarPhaseRest
}

// GetSeasonInfo 获取赛季详细信息
func (s *SectWarService) GetSeasonInfo(ctx context.Context) (*WarSeason, error) {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return nil, err
	}

	// 获取注册宗门数量
	if season.Phase == WarPhaseRegistration {
		total, _ := s.seasonColl().CountDocuments(ctx, bson.M{
			"_id": season.ID,
			"registered_sects": bson.M{"$exists": true},
		})
		_ = total
	}

	return season, nil
}

// GetSeasonByNumber 按编号获取赛季
func (s *SectWarService) GetSeasonByNumber(ctx context.Context, seasonNumber int) (*WarSeason, error) {
	var season WarSeason
	err := s.seasonColl().FindOne(ctx, bson.M{"season_number": seasonNumber}).Decode(&season)
	if err != nil {
		return nil, fmt.Errorf("赛季不存在: %w", err)
	}
	return &season, nil
}

// SetSeasonStatus 更新赛季状态
func (s *SectWarService) SetSeasonStatus(ctx context.Context, seasonID, status string) error {
	_, err := s.seasonColl().UpdateOne(ctx, bson.M{"_id": seasonID},
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}})
	return err
}

// ============================================================
// 2. 报名与匹配
// ============================================================

// RegisterSect 宗门报名参战
func (s *SectWarService) RegisterSect(ctx context.Context, sectID, operatorID string, memberIDs []string) error {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return fmt.Errorf("获取赛季失败: %w", err)
	}

	// 仅在报名期可报名
	if season.Phase != WarPhaseRegistration {
		return fmt.Errorf("当前不在报名期(当前阶段: %s)", season.Phase)
	}

	// 验证宗门存在
	var sect model.Sect
	err = s.sectColl().FindOne(ctx, bson.M{"_id": sectID}).Decode(&sect)
	if err != nil {
		return fmt.Errorf("宗门不存在")
	}

	// 检查宗门等级
	if sect.Level < s.cfg.MinSectLevel {
		return fmt.Errorf("宗门等级不足，需要 %d 级(当前 %d 级)", s.cfg.MinSectLevel, sect.Level)
	}

	// 检查宗门成员数
	if sect.MemberCount < s.cfg.MinMembersForWar {
		return fmt.Errorf("宗门成员不足 %d 人，无法参战", s.cfg.MinMembersForWar)
	}

	// 权限校验: 仅宗主/长老可报名
	var operator model.SectMember
	err = s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": operatorID}).Decode(&operator)
	if err != nil {
		return fmt.Errorf("操作者不是宗门成员")
	}
	if operator.Rank != model.SectLeader && operator.Rank != model.SectElder {
		return fmt.Errorf("仅宗主和长老可报名宗门战争")
	}

	// 验证参战成员数量
	if len(memberIDs) < s.cfg.PlayersPerMatch {
		return fmt.Errorf("需要至少 %d 名参战成员", s.cfg.PlayersPerMatch)
	}

	// 验证所有成员属于该宗门
	for _, mid := range memberIDs {
		count, _ := s.memberColl().CountDocuments(ctx, bson.M{"sect_id": sectID, "user_id": mid})
		if count == 0 {
			return fmt.Errorf("成员 %s 不是宗门成员", mid)
		}
	}

	// 检查是否重复报名
	for _, rs := range season.RegisteredSects {
		if rs == sectID {
			return fmt.Errorf("宗门已在当前赛季报名")
		}
	}

	// 将宗门加入注册列表
	_, err = s.seasonColl().UpdateOne(ctx, bson.M{"_id": season.ID},
		bson.M{"$push": bson.M{"registered_sects": sectID}, "$set": bson.M{"updated_at": time.Now()}})
	if err != nil {
		return fmt.Errorf("报名失败: %w", err)
	}

	// 保存参战成员信息到专用集合
	registration := bson.M{
		"_id":        uuid.New().String(),
		"season_id":  season.ID,
		"sect_id":    sectID,
		"sect_name":  sect.Name,
		"sect_level": sect.Level,
		"member_ids": memberIDs,
		"created_at": time.Now(),
	}
	_, err = s.db.Collection("war_registrations").InsertOne(ctx, registration)
	if err != nil {
		return fmt.Errorf("保存报名信息失败: %w", err)
	}

	s.logger.Info("宗门报名赛季成功", "sect_name", sect.Name,
		"sect_level", sect.Level, "season", season.SeasonNumber, "members", len(memberIDs))

	return nil
}

// GenerateBrackets 生成赛程分组(报名截止后自动调用)
func (s *SectWarService) GenerateBrackets(ctx context.Context, seasonID string) ([]*WarBracket, error) {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return nil, err
	}
	if season.ID != seasonID {
		return nil, fmt.Errorf("赛季不匹配")
	}

	if season.Phase != WarPhaseRegistration {
		return nil, fmt.Errorf("当前不在报名期，无法生成赛程")
	}

	if len(season.RegisteredSects) < 2 {
		return nil, fmt.Errorf("报名宗门不足(当前 %d 个)，无法生成赛程", len(season.RegisteredSects))
	}

	// 获取各个宗门的等级信息用于匹配
	type sectInfo struct {
		ID    string
		Name  string
		Level int
	}
	var registeredSects []sectInfo
	for _, sid := range season.RegisteredSects {
		var sect model.Sect
		err := s.sectColl().FindOne(ctx, bson.M{"_id": sid}).Decode(&sect)
		if err != nil {
			continue
		}
		registeredSects = append(registeredSects, sectInfo{
			ID:    sect.ID,
			Name:  sect.Name,
			Level: sect.Level,
		})
	}

	// 按宗门等级排序(高级别先排)
	sort.Slice(registeredSects, func(i, j int) bool {
		return registeredSects[i].Level > registeredSects[j].Level
	})

	// 确定合适的分组大小(最接近的 2^N)
	bracketSize := s.nearestPowerOfTwo(len(registeredSects))
	if bracketSize < 4 {
		bracketSize = 4
	}
	if bracketSize > 32 {
		bracketSize = 32
	}

	// 种子选手分配: 强队分散在不同赛区
	numBrackets := 1
	if bracketSize >= 16 {
		numBrackets = 2
	} else if bracketSize >= 32 {
		numBrackets = 4
	}

	// 将宗门均匀分配到各赛区
	brackets := make([]*WarBracket, numBrackets)
	for i := range brackets {
		brackets[i] = &WarBracket{
			ID:        uuid.New().String(),
			SeasonID:  seasonID,
			Name:      fmt.Sprintf("赛区-%c", 'A'+i),
			Sects:     []string{},
			Rounds:    []*WarRound{},
			Status:    BracketPending,
			CreatedAt: time.Now(),
		}
	}

	// 蛇形分配: 种子选手分散
	for idx, sInfo := range registeredSects {
		bIdx := idx % numBrackets
		brackets[bIdx].Sects = append(brackets[bIdx].Sects, sInfo.ID)
	}

	// 补齐空位(轮空)
	sectsPerBracket := bracketSize / numBrackets
	for _, bracket := range brackets {
		for len(bracket.Sects) < sectsPerBracket {
			bracket.Sects = append(bracket.Sects, "") // 轮空
		}
	}

	// 生成对阵图
	for _, bracket := range brackets {
		s.generateBracketRounds(ctx, bracket)
	}

	// 存储赛程
	for _, bracket := range brackets {
		_, err := s.bracketColl().InsertOne(ctx, bracket)
		if err != nil {
			return nil, fmt.Errorf("保存赛程失败: %w", err)
		}
		season.BracketIDs = append(season.BracketIDs, bracket.ID)
	}

	// 更新赛季
	season.Phase = WarPhaseWar
	season.UpdatedAt = time.Now()
	_, err = s.seasonColl().UpdateOne(ctx, bson.M{"_id": season.ID},
		bson.M{"$set": bson.M{
			"phase":        season.Phase,
			"bracket_ids":  season.BracketIDs,
			"updated_at":   season.UpdatedAt,
		}})
	if err != nil {
		return nil, fmt.Errorf("更新赛季状态失败: %w", err)
	}

	s.logger.Info("赛程生成完成", "season", season.SeasonNumber,
		"brackets", numBrackets, "sects", len(registeredSects))

	return brackets, nil
}

// generateBracketRounds 为赛区生成轮次和比赛
func (s *SectWarService) generateBracketRounds(ctx context.Context, bracket *WarBracket) {
	numSects := len(bracket.Sects)
	if numSects < 2 {
		return
	}

	numRounds := int(math.Ceil(math.Log2(float64(numSects))))
	now := time.Now()

	for round := 1; round <= numRounds; round++ {
		// 本轮比赛数
		matchesInRound := numSects / (1 << uint(round))
		if matchesInRound == 0 {
			matchesInRound = 1
		}

		warRound := &WarRound{
			RoundNumber: round,
			Matches:     make([]*WarMatch, 0, matchesInRound),
			StartTime:   now.AddDate(0, 0, (round-1)*1), // 每轮间隔1天
		}

		// 生成比赛
		for m := 0; m < matchesInRound; m++ {
			sectAIdx := m * 2
			sectBIdx := m*2 + 1

			match := &WarMatch{
				ID:          uuid.New().String(),
				SeasonID:    bracket.SeasonID,
				BracketID:   bracket.ID,
				RoundNumber: round,
				SectA:       bracket.Sects[sectAIdx],
				SectB:       bracket.Sects[sectBIdx],
				ScoreA:      0,
				ScoreB:      0,
				Status:      MatchPending,
				StartTime:   warRound.StartTime,
				CreatedAt:   time.Now(),
				Rounds:      make([]*MatchRound, s.cfg.PlayersPerMatch),
			}

			// 初始化3轮制
			for ri := 0; ri < s.cfg.PlayersPerMatch; ri++ {
				role := "attack"
				if ri%2 == 1 {
					role = "defend"
				}
				match.Rounds[ri] = &MatchRound{
					RoundIndex: ri,
					SectARole:  role,
					SectBRole:  oppositeRole(role),
					Status:     "pending",
				}
			}

			warRound.Matches = append(warRound.Matches, match)
		}

		bracket.Rounds = append(bracket.Rounds, warRound)
	}
}

// nearestPowerOfTwo 获取最接近的2的幂
func (s *SectWarService) nearestPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	power := 1
	for power < n {
		power <<= 1
	}
	// 取最接近的(向上取整)
	return power
}

// oppositeRole 返回相反角色
func oppositeRole(role string) string {
	if role == "attack" {
		return "defend"
	}
	return "attack"
}

// ============================================================
// 3. 战斗系统
// ============================================================

// StartMatch 开始比赛(由定时器或手动触发)
func (s *SectWarService) StartMatch(ctx context.Context, matchID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var match WarMatch
	err := s.matchColl().FindOne(ctx, bson.M{"_id": matchID}).Decode(&match)
	if err != nil {
		return fmt.Errorf("比赛不存在")
	}
	if match.Status != MatchPending {
		return fmt.Errorf("比赛已开始或已结束")
	}

	// 检查双方是否都已就绪(对方不能是轮空)
	if match.SectA == "" || match.SectB == "" {
		// 轮空自动晋级
		winner := match.SectA
		if winner == "" {
			winner = match.SectB
		}
		match.Status = MatchFinished
		if winner == match.SectA {
			match.ScoreA = s.cfg.WinScore
			match.Result = MatchResultAWin
		} else {
			match.ScoreB = s.cfg.WinScore
			match.Result = MatchResultBWin
		}
		match.EndTime = time.Now()
		_, _ = s.matchColl().UpdateOne(ctx, bson.M{"_id": matchID}, bson.M{"$set": bson.M{
			"status":    match.Status,
			"score_a":   match.ScoreA,
			"score_b":   match.ScoreB,
			"result":    match.Result,
			"end_time":  match.EndTime,
		}})
		return nil
	}

	// 设置比赛为进行中
	match.Status = MatchFighting
	match.StartTime = time.Now()

	// 初始化回合状态
	for _, round := range match.Rounds {
		round.Status = "pending"
	}

	_, err = s.matchColl().UpdateOne(ctx, bson.M{"_id": matchID},
		bson.M{"$set": bson.M{
			"status":     match.Status,
			"start_time": match.StartTime,
		}})
	if err != nil {
		return fmt.Errorf("开始比赛失败: %w", err)
	}

	s.logger.Info("比赛开始", "match_id", match.ID[:8], "sect_a", match.SectAName, "sect_b", match.SectBName)

	return nil
}

// SubmitBattleResult 提交比赛回合结果(由战斗系统回调)
func (s *SectWarService) SubmitBattleResult(ctx context.Context, matchID string, scoreA, scoreB int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var match WarMatch
	err := s.matchColl().FindOne(ctx, bson.M{"_id": matchID}).Decode(&match)
	if err != nil {
		return fmt.Errorf("比赛不存在")
	}
	if match.Status != MatchFighting {
		return fmt.Errorf("比赛未在进行中")
	}

	match.ScoreA = scoreA
	match.ScoreB = scoreB

	// 判定胜负: 3轮制, 先赢2轮者胜
	if scoreA >= 2 {
		match.Status = MatchFinished
		match.Result = MatchResultAWin
	} else if scoreB >= 2 {
		match.Status = MatchFinished
		match.Result = MatchResultBWin
	}

	if match.Status == MatchFinished {
		match.EndTime = time.Now()

		// 发放基础奖励
		s.awardMatchRewards(ctx, &match)

		s.logger.Info("比赛结束", "sect_a", match.SectAName,
			"sect_b", match.SectBName, "result", match.Result, "score_a", scoreA, "score_b", scoreB)
	}

	_, err = s.matchColl().UpdateOne(ctx, bson.M{"_id": matchID},
		bson.M{"$set": bson.M{
			"status":   match.Status,
			"score_a":  match.ScoreA,
			"score_b":  match.ScoreB,
			"result":   match.Result,
			"end_time": match.EndTime,
		}})
	if err != nil {
		return fmt.Errorf("更新比赛结果失败: %w", err)
	}

	// 如果比赛结束, 推进赛程
	if match.Status == MatchFinished {
		s.advanceBracket(ctx, match.BracketID, match.RoundNumber, &match)
	}

	return nil
}

// advanceBracket 比赛结束后推进赛程
func (s *SectWarService) advanceBracket(ctx context.Context, bracketID string, roundNumber int, match *WarMatch) {
	var bracket WarBracket
	err := s.bracketColl().FindOne(ctx, bson.M{"_id": bracketID}).Decode(&bracket)
	if err != nil {
		return
	}

	// 确定胜者
	var winnerSect string
	if match.Result == MatchResultAWin {
		winnerSect = match.SectA
	} else if match.Result == MatchResultBWin {
		winnerSect = match.SectB
	} else {
		return
	}

	// 如果是决赛(最后一轮)
	nextRound := roundNumber
	if nextRound >= len(bracket.Rounds) {
		// 这是决赛, 记录胜者
		bracket.Winner = winnerSect
		bracket.Status = BracketFinished
		_, _ = s.bracketColl().UpdateOne(ctx, bson.M{"_id": bracketID},
			bson.M{"$set": bson.M{
				"winner": bracket.Winner,
				"status": bracket.Status,
			}})

		s.logger.Info("赛区决赛结束", "bracket", bracket.Name, "winner", winnerSect)
		return
	}

	// 在下一轮中找到胜者的位置
	nextRoundMatches := bracket.Rounds[nextRound].Matches
	for _, nextMatch := range nextRoundMatches {
		if nextMatch.SectA == "" {
			nextMatch.SectA = winnerSect
			_, _ = s.matchColl().UpdateOne(ctx, bson.M{"_id": nextMatch.ID},
				bson.M{"$set": bson.M{"sect_a": winnerSect}})
			break
		} else if nextMatch.SectB == "" {
			nextMatch.SectB = winnerSect
			_, _ = s.matchColl().UpdateOne(ctx, bson.M{"_id": nextMatch.ID},
				bson.M{"$set": bson.M{"sect_b": winnerSect}})
			break
		}
	}
}

// awardMatchRewards 发放单场比赛奖励
func (s *SectWarService) awardMatchRewards(ctx context.Context, match *WarMatch) {
	var winnerSect, loserSect, winnerName, loserName string
	if match.Result == MatchResultAWin {
		winnerSect, loserSect = match.SectA, match.SectB
		winnerName, loserName = match.SectAName, match.SectBName
	} else if match.Result == MatchResultBWin {
		winnerSect, loserSect = match.SectB, match.SectA
		winnerName, loserName = match.SectBName, match.SectAName
	} else {
		return
	}

	// 胜者奖励
	winFunds := int64(5000)
	winReputation := int64(200)
	_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": winnerSect},
		bson.M{"$inc": bson.M{"funds": winFunds, "reputation": winReputation}})

	// 败者参与奖
	loseFunds := int64(2000)
	loseReputation := int64(50)
	_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": loserSect},
		bson.M{"$inc": bson.M{"funds": loseFunds, "reputation": loseReputation}})

	s.logger.Info("比赛奖励发放", "winner", winnerName, "win_funds", winFunds, "win_reputation", winReputation, "loser", loserName, "lose_funds", loseFunds, "lose_reputation", loseReputation)
}

// ============================================================
// 4. 领地系统
// ============================================================

// ClaimTerritory 占领灵脉(胜者占领)
func (s *SectWarService) ClaimTerritory(ctx context.Context, winnerSectID, veinID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 验证宗门存在
	var sect model.Sect
	err := s.sectColl().FindOne(ctx, bson.M{"_id": winnerSectID}).Decode(&sect)
	if err != nil {
		return fmt.Errorf("宗门不存在")
	}

	// 验证灵脉存在
	var vein SpiritVein
	err = s.veinColl().FindOne(ctx, bson.M{"_id": veinID}).Decode(&vein)
	if err != nil {
		return fmt.Errorf("灵脉不存在")
	}

	if vein.Status == "contested" {
		return fmt.Errorf("灵脉正在被争夺中")
	}

	// 检查宗门已占领的灵脉数量
	ownedCount, _ := s.veinColl().CountDocuments(ctx, bson.M{"owner_sect_id": winnerSectID})
	maxVeins := 1
	if sect.Level >= 10 {
		maxVeins = 2
	}
	if ownedCount >= int64(maxVeins) {
		return fmt.Errorf("宗门已达最大灵脉持有数(%d), 需要先放弃已有灵脉", maxVeins)
	}

	now := time.Now()

	// 如果灵脉正被其他宗门占领, 记录历史
	if vein.OwnerSectID != "" && vein.OwnerSectID != winnerSectID {
		s.recordVeinOccupation(ctx, &vein, now)
	}

	// 占领灵脉
	vein.OwnerSectID = winnerSectID
	vein.OwnerSectName = sect.Name
	vein.OccupiedAt = now
	vein.Status = "occupied"
	vein.UpdatedAt = now

	_, err = s.veinColl().UpdateOne(ctx, bson.M{"_id": veinID},
		bson.M{"$set": bson.M{
			"owner_sect_id":   vein.OwnerSectID,
			"owner_sect_name": vein.OwnerSectName,
			"occupied_at":     vein.OccupiedAt,
			"status":          vein.Status,
			"updated_at":      vein.UpdatedAt,
		}})
	if err != nil {
		return fmt.Errorf("占领灵脉失败: %w", err)
	}

	// 记录占领历史
	s.recordVeinOccupation(ctx, &vein, now)

	s.logger.Info("宗门成功占领灵脉", "sect_name", sect.Name,
		"vein_name", vein.Name, "quality", vein.Quality)

	return nil
}

// recordVeinOccupation 记录灵脉占领历史
func (s *SectWarService) recordVeinOccupation(ctx context.Context, vein *SpiritVein, now time.Time) {
	history := bson.M{
		"_id":          uuid.New().String(),
		"vein_id":      vein.ID,
		"vein_name":    vein.Name,
		"vein_quality": vein.Quality,
		"sect_id":      vein.OwnerSectID,
		"sect_name":    vein.OwnerSectName,
		"occupied_at":  now,
		"created_at":   now,
	}
	_, _ = s.db.Collection("vein_occupation_history").InsertOne(ctx, history)
}

// ReleaseTerritory 放弃灵脉
func (s *SectWarService) ReleaseTerritory(ctx context.Context, sectID, veinID string) error {
	var vein SpiritVein
	err := s.veinColl().FindOne(ctx, bson.M{"_id": veinID, "owner_sect_id": sectID}).Decode(&vein)
	if err != nil {
		return fmt.Errorf("灵脉不属于该宗门")
	}

	now := time.Now()
	// 记录失去占领
	history := bson.M{
		"_id":            uuid.New().String(),
		"vein_id":        vein.ID,
		"vein_name":      vein.Name,
		"sect_id":        sectID,
		"sect_name":      vein.OwnerSectName,
		"occupied_at":    vein.OccupiedAt,
		"lost_at":        now,
		"duration_hours": int(now.Sub(vein.OccupiedAt).Hours()),
	}
	_, _ = s.db.Collection("vein_occupation_history").InsertOne(ctx, history)

	// 释放灵脉
	_, err = s.veinColl().UpdateOne(ctx, bson.M{"_id": veinID},
		bson.M{"$set": bson.M{
			"owner_sect_id":   "",
			"owner_sect_name": "",
			"occupied_at":     nil,
			"status":          "idle",
			"updated_at":      now,
		}})
	return err
}

// ContestVein 争夺灵脉(发起挑战)
func (s *SectWarService) ContestVein(ctx context.Context, sectID, veinID string) error {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return fmt.Errorf("获取赛季失败: %w", err)
	}

	// 仅在战争期可争夺
	if season.Phase != WarPhaseWar {
		return fmt.Errorf("当前不在战争期")
	}

	var sect model.Sect
	err = s.sectColl().FindOne(ctx, bson.M{"_id": sectID}).Decode(&sect)
	if err != nil {
		return fmt.Errorf("宗门不存在")
	}

	var vein SpiritVein
	err = s.veinColl().FindOne(ctx, bson.M{"_id": veinID}).Decode(&vein)
	if err != nil {
		return fmt.Errorf("灵脉不存在")
	}

	if vein.OwnerSectID == "" {
		return fmt.Errorf("灵脉未被占领，请使用占领功能")
	}
	if vein.OwnerSectID == sectID {
		return fmt.Errorf("灵脉已被宗门占领，无需争夺")
	}
	if vein.Status == "contested" {
		return fmt.Errorf("灵脉正在被争夺中，请等待结果")
	}

	// 检查宗门已有灵脉数
	ownedCount, _ := s.veinColl().CountDocuments(ctx, bson.M{"owner_sect_id": sectID})
	maxVeins := 1
	if sect.Level >= 10 {
		maxVeins = 2
	}
	if ownedCount >= int64(maxVeins) {
		return fmt.Errorf("宗门已达最大灵脉持有数(%d)", maxVeins)
	}

	// 标记灵脉为争夺状态
	vein.Status = "contested"
	_, err = s.veinColl().UpdateOne(ctx, bson.M{"_id": veinID},
		bson.M{"$set": bson.M{"status": "contested", "updated_at": time.Now()}})
	if err != nil {
		return fmt.Errorf("争夺灵脉失败: %w", err)
	}

	s.logger.Info("宗门对灵脉发起争夺", "sect_name", sect.Name,
		"vein_name", vein.Name, "quality", vein.Quality, "current_owner", vein.OwnerSectName)

	return nil
}

// GetSpiritVeins 获取所有灵脉列表
func (s *SectWarService) GetSpiritVeins(ctx context.Context) ([]*SpiritVein, error) {
	cursor, err := s.veinColl().Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var veins []*SpiritVein
	if err := cursor.All(ctx, &veins); err != nil {
		return nil, err
	}

	if veins == nil {
		veins = []*SpiritVein{}
	}
	return veins, nil
}

// GetSectVeins 获取宗门占领的灵脉
func (s *SectWarService) GetSectVeins(ctx context.Context, sectID string) ([]*SpiritVein, error) {
	cursor, err := s.veinColl().Find(ctx, bson.M{"owner_sect_id": sectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var veins []*SpiritVein
	if err := cursor.All(ctx, &veins); err != nil {
		return nil, err
	}
	return veins, nil
}

// ============================================================
// 5. 排名与奖励
// ============================================================

// GetSeasonRankings 获取赛季排名
func (s *SectWarService) GetSeasonRankings(ctx context.Context) ([]*SectRanking, error) {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return nil, err
	}

	// 尝试从排名集合读取
	cursor, err := s.rankingColl().Find(ctx, bson.M{"season_id": season.ID},
		options.Find().SetSort(bson.D{{Key: "score", Value: -1}}))
	if err == nil {
		defer cursor.Close(ctx)
		var rankings []*SectRanking
		if err := cursor.All(ctx, &rankings); err == nil && len(rankings) > 0 {
			// 设置排名序号
			for i := range rankings {
				rankings[i].Rank = i + 1
			}
			return rankings, nil
		}
	}

	// 如果没有排名数据，从比赛记录计算
	return s.calculateRankings(ctx, season)
}

// calculateRankings 从比赛记录计算排名
func (s *SectWarService) calculateRankings(ctx context.Context, season *WarSeason) ([]*SectRanking, error) {
	// 获取所有比赛
	cursor, err := s.matchColl().Find(ctx, bson.M{
		"season_id": season.ID,
		"status":    string(MatchFinished),
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var matches []*WarMatch
	if err := cursor.All(ctx, &matches); err != nil {
		return nil, err
	}

	// 按宗门聚合
	rankMap := make(map[string]*SectRanking)
	for _, m := range matches {
		s.addMatchToRanking(rankMap, m, m.SectA, m.SectAName)
		s.addMatchToRanking(rankMap, m, m.SectB, m.SectBName)
	}

	// 转为切片并排序
	rankings := make([]*SectRanking, 0, len(rankMap))
	for _, r := range rankMap {
		rankings = append(rankings, r)
	}

	sort.Slice(rankings, func(i, j int) bool {
		if rankings[i].Score != rankings[j].Score {
			return rankings[i].Score > rankings[j].Score
		}
		return rankings[i].Wins > rankings[j].Wins
	})

	for i := range rankings {
		rankings[i].Rank = i + 1
	}

	return rankings, nil
}

// addMatchToRanking 将比赛结果聚合到宗门排名
func (s *SectWarService) addMatchToRanking(rankMap map[string]*SectRanking, match *WarMatch, sectID, sectName string) {
	if sectID == "" {
		return
	}
	r, exists := rankMap[sectID]
	if !exists {
		r = &SectRanking{
			SectID:    sectID,
			SectName:  sectName,
		}
		rankMap[sectID] = r
	}

	r.TotalMatches++

	// 获取宗门等级
	var sect model.Sect
	err := s.sectColl().FindOne(context.Background(), bson.M{"_id": sectID}).Decode(&sect)
	if err == nil {
		r.SectLevel = sect.Level
	}

	if match.Result == MatchResultAWin && match.SectA == sectID {
		r.Score += s.cfg.WinScore
		r.Wins++
	} else if match.Result == MatchResultBWin && match.SectB == sectID {
		r.Score += s.cfg.WinScore
		r.Wins++
	} else if match.Result == MatchResultDraw {
		r.Score += s.cfg.DrawScore
	} else {
		r.Losses++
	}
}

// SettleSeason 赛季结算
func (s *SectWarService) SettleSeason(ctx context.Context, seasonID string) error {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return err
	}

	rankings, err := s.GetSeasonRankings(ctx)
	if err != nil {
		return err
	}

	// 发放赛季奖励
	for i, rank := range rankings {
		rank.Rank = i + 1
		s.awardSeasonRewards(ctx, season, rank)
	}

	// 更新赛季状态
	season.Status = "settled"
	season.Phase = WarPhaseRest
	season.UpdatedAt = time.Now()
	_, err = s.seasonColl().UpdateOne(ctx, bson.M{"_id": seasonID},
		bson.M{"$set": bson.M{
			"status":     season.Status,
			"phase":      season.Phase,
			"updated_at": season.UpdatedAt,
		}})
	if err != nil {
		return fmt.Errorf("赛季结算失败: %w", err)
	}

	s.logger.Info("赛季结算完成", "season", season.SeasonNumber,
		"rankings", len(rankings))

	return nil
}

// awardSeasonRewards 发放赛季排名奖励
func (s *SectWarService) awardSeasonRewards(ctx context.Context, season *WarSeason, ranking *SectRanking) {
	var reward *SeasonReward
	for _, r := range SeasonRewards {
		if ranking.Rank >= r.RankMin && ranking.Rank <= r.RankMax {
			reward = &r
			break
		}
	}
	if reward == nil {
		return
	}

	// 宗门资金和声望
	_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": ranking.SectID},
		bson.M{"$inc": bson.M{
			"funds":      reward.SectFunds,
			"reputation": reward.Reputation,
		}})

	// 宗门等级提升
	if reward.SectLevelBoost > 0 {
		s.boostSectLevel(ctx, ranking.SectID, reward.SectLevelBoost)
	}

	// 成员贡献奖励(所有参战成员)
	cursor, _ := s.memberColl().Find(ctx, bson.M{"sect_id": ranking.SectID})
	if cursor != nil {
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var member model.SectMember
			if err := cursor.Decode(&member); err == nil {
				_, _ = s.memberColl().UpdateOne(ctx,
					bson.M{"sect_id": ranking.SectID, "user_id": member.UserID},
					bson.M{"$inc": bson.M{"contribution": reward.Contribution}})
			}
		}
	}

	s.logger.Info("赛季奖励发放", "sect_id", ranking.SectID,
		"rank", ranking.Rank, "funds", reward.SectFunds, "reputation", reward.Reputation, "contribution", reward.Contribution)
}

// boostSectLevel 提升宗门等级
func (s *SectWarService) boostSectLevel(ctx context.Context, sectID string, levels int) {
	for i := 0; i < levels; i++ {
		var sect model.Sect
		err := s.sectColl().FindOne(ctx, bson.M{"_id": sectID}).Decode(&sect)
		if err != nil || sect.Level >= 50 {
			return
		}
		_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": sectID},
			bson.M{"$inc": bson.M{"level": 1, "max_members": 5}})
	}
}

// ============================================================
// 6. 查询接口
// ============================================================

// GetActiveWarStatus 获取宗门当前的战争状态
func (s *SectWarService) GetActiveWarStatus(ctx context.Context, sectID string) (*WarMatch, error) {
	var match WarMatch
	err := s.matchColl().FindOne(ctx, bson.M{
		"$or": []bson.M{
			{"sect_a": sectID},
			{"sect_b": sectID},
		},
		"status": bson.M{"$in": []string{
			string(MatchPending),
			string(MatchFighting),
		}},
	}, options.FindOne().SetSort(bson.D{{Key: "start_time", Value: -1}})).Decode(&match)
	if err != nil {
		return nil, fmt.Errorf("暂无进行中的战争")
	}
	return &match, nil
}

// GetSectMatches 获取宗门的所有比赛记录
func (s *SectWarService) GetSectMatches(ctx context.Context, sectID string, limit int64) ([]*WarMatch, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"sect_a": sectID},
			{"sect_b": sectID},
		},
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit)

	cursor, err := s.matchColl().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var matches []*WarMatch
	if err := cursor.All(ctx, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

// GetBracketsBySeason 获取赛季的所有赛程
func (s *SectWarService) GetBracketsBySeason(ctx context.Context, seasonID string) ([]*WarBracket, error) {
	cursor, err := s.bracketColl().Find(ctx, bson.M{"season_id": seasonID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var brackets []*WarBracket
	if err := cursor.All(ctx, &brackets); err != nil {
		return nil, err
	}
	return brackets, nil
}

// GetBracketByID 获取单个赛程详情
func (s *SectWarService) GetBracketByID(ctx context.Context, bracketID string) (*WarBracket, error) {
	var bracket WarBracket
	err := s.bracketColl().FindOne(ctx, bson.M{"_id": bracketID}).Decode(&bracket)
	if err != nil {
		return nil, fmt.Errorf("赛程不存在")
	}
	return &bracket, nil
}

// GetMatchByID 获取比赛详情
func (s *SectWarService) GetMatchByID(ctx context.Context, matchID string) (*WarMatch, error) {
	var match WarMatch
	err := s.matchColl().FindOne(ctx, bson.M{"_id": matchID}).Decode(&match)
	if err != nil {
		return nil, fmt.Errorf("比赛不存在")
	}
	return &match, nil
}

// ============================================================
// 7. 赛季转换(自动调度)
// ============================================================

// AutoTransition 自动推进赛季阶段(由调度器定时调用)
func (s *SectWarService) AutoTransition(ctx context.Context) error {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	updated := false

	switch season.Phase {
	case WarPhaseRegistration:
		// 报名结束 -> 生成赛程,进入战争期
		if now.After(season.RegistrationEnd) && len(season.RegisteredSects) >= 2 {
			_, err := s.GenerateBrackets(ctx, season.ID)
			if err != nil {
				s.logger.Error("自动生成赛程失败", "error", err)
				return err
			}
			updated = true
			s.logger.Info("赛季报名结束, 自动生成赛程", "season", season.SeasonNumber)
		}

	case WarPhaseWar:
		// 战争结束 -> 结算
		if now.After(season.WarEndTime) {
			err := s.SettleSeason(ctx, season.ID)
			if err != nil {
				s.logger.Error("自动结算失败", "error", err)
				return err
			}
			updated = true
			s.logger.Info("赛季战争结束, 自动结算", "season", season.SeasonNumber)
		}

	case WarPhaseRest:
		// 休战结束 -> 开启新赛季
		if now.After(season.EndTime) {
			_, err := s.StartNewSeason(ctx)
			if err != nil {
				return err
			}
			updated = true
			s.logger.Info("赛季休战期结束, 自动开启新赛季", "season", season.SeasonNumber)
		}
	}

	if !updated {
		// 更新阶段
		newPhase := s.determinePhase(*season)
		if newPhase != season.Phase {
			_, _ = s.seasonColl().UpdateOne(ctx, bson.M{"_id": season.ID},
				bson.M{"$set": bson.M{"phase": newPhase, "updated_at": now}})
		}
	}

	return nil
}

// ============================================================
// 8. 赛季奖励查询
// ============================================================

// GetSeasonRewards 获取赛季奖励配置
func (s *SectWarService) GetSeasonRewards() []SeasonReward {
	return SeasonRewards
}
