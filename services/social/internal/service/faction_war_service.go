// Package service 势力争霸系统
//
// 合体期解锁，宗门级PVP
// 每月1次全服宗门争霸
// 争夺「灵脉福地」控制权(灵气浓度5.0的公共区域)
// 占领福地的宗门: 全员修炼速度+30%, 持续到下个月
// 战斗形式: 宗门派出最强5人参战, 5v5车轮战
// 积分制: 击败1人+10分, 存活到最后+50分
// 奖励: 仙玉/稀有材料/称号/战旗
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"time"

	"cultivation-game/services/social/internal/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ============================================================
// 势力争霸 - 模型定义
// ============================================================

// FactionWarPhase 势力争霸阶段
type FactionWarPhase string

const (
	PhaseRegistration FactionWarPhase = "registration" // 报名阶段
	PhaseBattle       FactionWarPhase = "battle"       // 战斗阶段
	PhaseSettled      FactionWarPhase = "settled"      // 结算完成
)

// FactionWarRecord 势力争霸记录
type FactionWarRecord struct {
	ID            string                 `bson:"_id" json:"id"`
	Season        int                    `bson:"season" json:"season"`                 // 赛季编号
	Phase         FactionWarPhase        `bson:"phase" json:"phase"`                   // 当前阶段
	SectID        string                 `bson:"sect_id" json:"sect_id"`               // 报名宗门
	SectName      string                 `bson:"sect_name" json:"sect_name"`
	Players       []FactionWarPlayer     `bson:"players" json:"players"`               // 出战的5名弟子
	TotalScore    int                    `bson:"total_score" json:"total_score"`        // 宗门总积分
	TotalKills    int                    `bson:"total_kills" json:"total_kills"`        // 总击败数
	Rank          int                    `bson:"rank" json:"rank"`                      // 最终排名(0=未排名)
	IsWinner      bool                   `bson:"is_winner" json:"is_winner"`            // 是否占领福地
	Battles       []FactionBattleLog     `bson:"battles,omitempty" json:"battles,omitempty"` // 战斗日志
	EnrolledAt    time.Time              `bson:"enrolled_at" json:"enrolled_at"`
	UpdatedAt     time.Time              `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// FactionWarPlayer 参战弟子
type FactionWarPlayer struct {
	UserID   string `bson:"user_id" json:"user_id"`
	UserName string `bson:"user_name" json:"user_name"`
	Level    int    `bson:"level" json:"level"`           // 等级(用于战力计算)
	Realm    string `bson:"realm" json:"realm"`           // 境界
	Score    int    `bson:"score" json:"score"`           // 个人积分
	Kills    int    `bson:"kills" json:"kills"`           // 击败数
	Alive    bool   `bson:"alive" json:"alive"`           // 是否存活到最终
	Power    int    `bson:"power" json:"power"`           // 战力值(计算用)
}

// FactionBattleLog 单场战斗日志
type FactionBattleLog struct {
	Round      int    `bson:"round" json:"round"`           // 第几轮
	AttackerID string `bson:"attacker_id" json:"attacker_id"`
	AttackerName string `bson:"attacker_name" json:"attacker_name"`
	DefenderID string `bson:"defender_id" json:"defender_id"`
	DefenderName string `bson:"defender_name" json:"defender_name"`
	WinnerID   string `bson:"winner_id" json:"winner_id"`
	RoundDesc  string `bson:"round_desc" json:"round_desc"` // 战斗描述
}

// FactionSeason 势力争霸赛季
type FactionSeason struct {
	Season      int       `bson:"season" json:"season"`
	StartDate   time.Time `bson:"start_date" json:"start_date"`
	BattleDate  time.Time `bson:"battle_date" json:"battle_date"`   // 战斗日
	EndDate     time.Time `bson:"end_date" json:"end_date"`         // 结束日(下月1号)
	VeinOwnerID string    `bson:"vein_owner_id,omitempty" json:"vein_owner_id,omitempty"` // 福地占领宗门
	VeinOwnerName string  `bson:"vein_owner_name,omitempty" json:"vein_owner_name,omitempty"`
	Status      string    `bson:"status" json:"status"`             // pending / active / finished
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}

// FactionWarRanking 排行榜条目
type FactionWarRanking struct {
	Rank       int    `bson:"rank" json:"rank"`
	SectID     string `bson:"sect_id" json:"sect_id"`
	SectName   string `bson:"sect_name" json:"sect_name"`
	TotalScore int    `bson:"total_score" json:"total_score"`
	TotalKills int    `bson:"total_kills" json:"total_kills"`
	Wins       int    `bson:"wins" json:"wins"`             // 历史夺冠次数
}

// SpiritVeinBonus 灵脉加成属性
type SpiritVeinBonus struct {
	CultivationSpeed float64 `bson:"cultivation_speed" json:"cultivation_speed"` // 修炼速度加成(百分比)
	BreakthroughRate float64 `bson:"breakthrough_rate" json:"breakthrough_rate"` // 突破概率加成(百分比)
	SpiritStoneYield int64   `bson:"spirit_stone_yield" json:"spirit_stone_yield"` // 每日灵石产量
}

// SpiritVein 灵脉福地信息
type SpiritVein struct {
	ID          string          `bson:"_id" json:"id"`
	Name        string          `bson:"name" json:"name"`
	Quality     int             `bson:"quality" json:"quality"`               // 1-5星品质
	QiDensity   float64         `bson:"qi_density" json:"qi_density"`         // 灵气浓度
	RegionID    string          `bson:"region_id,omitempty" json:"region_id,omitempty"`
	RegionName  string          `bson:"region_name,omitempty" json:"region_name,omitempty"`
	OwnerSectID string          `bson:"owner_sect_id,omitempty" json:"owner_sect_id,omitempty"`
	OwnerSectName string        `bson:"owner_sect_name,omitempty" json:"owner_sect_name,omitempty"`
	OccupiedAt  time.Time       `bson:"occupied_at,omitempty" json:"occupied_at,omitempty"`
	PositionX   int             `bson:"position_x,omitempty" json:"position_x,omitempty"`
	PositionY   int             `bson:"position_y,omitempty" json:"position_y,omitempty"`
	Description string          `bson:"description,omitempty" json:"description,omitempty"`
	Status      string          `bson:"status" json:"status"`                 // idle / contested / occupied
	BoostBuff   string          `bson:"boost_buff" json:"boost_buff"`         // 占领加成描述
	Bonus       SpiritVeinBonus `bson:"bonus,omitempty" json:"bonus,omitempty"`
	CreatedAt   time.Time       `bson:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt   time.Time       `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// FactionWarConfig 势力争霸配置
type FactionWarConfig struct {
	MinSectLevel    int // 最低宗门等级(合体期)
	PlayersPerSect  int // 每宗出战人数
	KillScore       int // 击败1人得分
	SurviveBonus    int // 存活到最后加分
	QiDensity       float64 // 福地灵气浓度
	CultivationBuff float64 // 修炼速度加成(30%)
}

// DefaultFactionWarConfig 默认配置
func DefaultFactionWarConfig() FactionWarConfig {
	return FactionWarConfig{
		MinSectLevel:    6,  // 合体期对应宗门等级6
		PlayersPerSect:  5,
		KillScore:       10,
		SurviveBonus:    50,
		QiDensity:       5.0,
		CultivationBuff: 0.30,
	}
}

// ============================================================
// FactionWarService 势力争霸服务
// ============================================================

// FactionWarService 势力争霸业务
type FactionWarService struct {
	logger    *slog.Logger
	db        *mongo.Database
	cfg       FactionWarConfig
}

// NewFactionWarService 创建势力争霸服务
func NewFactionWarService(logger *slog.Logger, db *mongo.Database) *FactionWarService {
	return &FactionWarService{
		logger: logger,
		db:     db,
		cfg:    DefaultFactionWarConfig(),
	}
}

func (s *FactionWarService) factionColl() *mongo.Collection { return s.db.Collection("faction_wars") }
func (s *FactionWarService) seasonColl() *mongo.Collection  { return s.db.Collection("faction_seasons") }
func (s *FactionWarService) veinColl() *mongo.Collection    { return s.db.Collection("spirit_veins") }
func (s *FactionWarService) sectColl() *mongo.Collection    { return s.db.Collection("sects") }
func (s *FactionWarService) memberColl() *mongo.Collection  { return s.db.Collection("sect_members") }

// ============================================================
// 赛季管理
// ============================================================

// GetOrCreateSeason 获取当前赛季，不存在则创建
func (s *FactionWarService) GetOrCreateSeason(ctx context.Context) (*FactionSeason, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "season", Value: -1}})
	var season FactionSeason
	err := s.seasonColl().FindOne(ctx, bson.M{}, opts).Decode(&season)
	if err == nil {
		// 检查是否仍在有效期内(从 battle_date 到下个月)
		if time.Now().Before(season.EndDate) {
			return &season, nil
		}
		// 赛季已结束，自动开新赛季
	}

	return s.startNewSeason(ctx)
}

// startNewSeason 开启新赛季
func (s *FactionWarService) startNewSeason(ctx context.Context) (*FactionSeason, error) {
	var lastSeason FactionSeason
	opts := options.FindOne().SetSort(bson.D{{Key: "season", Value: -1}})
	_ = s.seasonColl().FindOne(ctx, bson.M{}, opts).Decode(&lastSeason)

	newSeasonNum := lastSeason.Season + 1
	now := time.Now()

	// 战斗日：每月28号(如果已过则下个月28号)
	battleDate := time.Date(now.Year(), now.Month(), 28, 20, 0, 0, 0, now.Location())
	if now.After(battleDate) {
		battleDate = time.Date(now.Year(), now.Month()+1, 28, 20, 0, 0, 0, now.Location())
	}

	// 结束日：下个月1号(福地buff持续到下月)
	endDate := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	season := &FactionSeason{
		Season:     newSeasonNum,
		StartDate:  now,
		BattleDate: battleDate,
		EndDate:    endDate,
		Status:     "pending",
		CreatedAt:  now,
	}

	_, err := s.seasonColl().InsertOne(ctx, season)
	if err != nil {
		return nil, fmt.Errorf("创建新赛季失败: %w", err)
	}

	// 确保灵脉福地记录存在
	s.initSpiritVein(ctx)

	return season, nil
}

// initSpiritVein 初始化/确保灵脉福地存在
func (s *FactionWarService) initSpiritVein(ctx context.Context) {
	count, _ := s.veinColl().CountDocuments(ctx, bson.M{})
	if count > 0 {
		return
	}

	vein := &SpiritVein{
		ID:        "spirit_vein_01",
		Name:      "天罡灵脉",
		QiDensity: s.cfg.QiDensity,
		BoostBuff: fmt.Sprintf("修炼速度+%.0f%%", s.cfg.CultivationBuff*100),
	}
	_, _ = s.veinColl().InsertOne(ctx, vein)
}

// ============================================================
// 报名
// ============================================================

// Enroll 报名势力争霸(宗主/长老操作)
func (s *FactionWarService) Enroll(ctx context.Context, sectID, userID string, playerIDs []string) (*FactionWarRecord, error) {
	// --- 赛季检查 ---
	season, err := s.GetOrCreateSeason(ctx)
	if err != nil {
		return nil, err
	}

	// 报名截止到战斗日前
	if time.Now().After(season.BattleDate) {
		return nil, fmt.Errorf("本争霸报名已截止，下次请早")
	}

	// --- 宗门等级校验 ---
	var sect model.Sect
	err = s.sectColl().FindOne(ctx, bson.M{"_id": sectID}).Decode(&sect)
	if err != nil {
		return nil, fmt.Errorf("宗门不存在")
	}
	if sect.Level < s.cfg.MinSectLevel {
		return nil, fmt.Errorf("宗门等级不足，需要 %d 级(合体期)，当前 %d 级", s.cfg.MinSectLevel, sect.Level)
	}

	// --- 权限校验(仅宗主/长老) ---
	var member model.SectMember
	err = s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": userID}).Decode(&member)
	if err != nil {
		return nil, fmt.Errorf("成员不存在")
	}
	if member.Rank != model.SectLeader && member.Rank != model.SectElder {
		return nil, fmt.Errorf("仅宗主和长老可报名势力争霸")
	}

	// --- 人数校验 ---
	if len(playerIDs) != s.cfg.PlayersPerSect {
		return nil, fmt.Errorf("需要派出 %d 名弟子参战", s.cfg.PlayersPerSect)
	}

	// --- 校验玩家都是宗门成员 ---
	for _, pid := range playerIDs {
		var pm model.SectMember
		err := s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": pid}).Decode(&pm)
		if err != nil {
			return nil, fmt.Errorf("弟子 %s 不是宗门成员", pid)
		}
	}

	// --- 检查是否重复报名 ---
	count, _ := s.factionColl().CountDocuments(ctx, bson.M{
		"sect_id": sectID,
		"season":  season.Season,
		"phase":   bson.M{"$ne": string(PhaseSettled)},
	})
	if count > 0 {
		return nil, fmt.Errorf("本宗门已在当前赛季报名")
	}

	// --- 构建参战弟子列表 ---
	players := s.buildPlayers(ctx, sectID, playerIDs)

	record := &FactionWarRecord{
		ID:         uuid.New().String(),
		Season:     season.Season,
		Phase:      PhaseRegistration,
		SectID:     sectID,
		SectName:   sect.Name,
		Players:    players,
		TotalScore: 0,
		TotalKills: 0,
		Rank:       0,
		IsWinner:   false,
		EnrolledAt: time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err = s.factionColl().InsertOne(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("报名失败: %w", err)
	}

	return record, nil
}

// buildPlayers 构建参战弟子信息
func (s *FactionWarService) buildPlayers(ctx context.Context, sectID string, playerIDs []string) []FactionWarPlayer {
	players := make([]FactionWarPlayer, 0, len(playerIDs))
	for _, pid := range playerIDs {
		// 简化版：从member集合获取成员信息
		var member model.SectMember
		_ = s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": pid}).Decode(&member)

		// 随机生成战力(实际应从玩家系统获取真实战力)
		power := 800 + rand.Intn(400) // 800~1200
		realm := "合体期"
		level := 60 + rand.Intn(20) // 60~79

		players = append(players, FactionWarPlayer{
			UserID:   pid,
			UserName: pid, // 实际应从user表获取名称
			Level:    level,
			Realm:    realm,
			Score:    0,
			Kills:    0,
			Alive:    true,
			Power:    power,
		})
	}
	return players
}

// ============================================================
// 战斗模拟
// ============================================================

// SimulateFactionWar 模拟势力争霸(所有报名宗门参与)
func (s *FactionWarService) SimulateFactionWar(ctx context.Context, seasonID int) ([]*FactionWarRecord, error) {
	season, err := s.GetOrCreateSeason(ctx)
	if err != nil {
		return nil, err
	}

	// 查询所有报名宗门
	cursor, err := s.factionColl().Find(ctx, bson.M{
		"season": season.Season,
		"phase":  string(PhaseRegistration),
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []*FactionWarRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("报名宗门不足，无法开始争霸")
	}

	// 随机打乱顺序
	rand.Shuffle(len(records), func(i, j int) {
		records[i], records[j] = records[j], records[i]
	})

	// 淘汰赛制：两两对战
	for len(records) > 1 {
		var nextRound []*FactionWarRecord
		for i := 0; i+1 < len(records); i += 2 {
			winner, loser := s.battle5v5(ctx, records[i], records[i+1])
			nextRound = append(nextRound, winner)
			s.saveBattleResult(ctx, winner)
			s.saveBattleResult(ctx, loser)
		}
		// 奇数宗门轮空自动晋级
		if len(records)%2 == 1 {
			nextRound = append(nextRound, records[len(records)-1])
		}
		records = nextRound
	}

	// 冠军
	champion := records[0]
	champion.IsWinner = true
	champion.Rank = 1
	champion.Phase = PhaseSettled
	champion.UpdatedAt = time.Now()
	s.saveBattleResult(ctx, champion)

	// 结算：分配福地、发放奖励
	s.settleFactionWar(ctx, champion, season)

	return records, nil
}

// battle5v5 5v5车轮战(返回胜者、败者)
func (s *FactionWarService) battle5v5(ctx context.Context, a, b *FactionWarRecord) (winner, loser *FactionWarRecord) {
	// 深拷贝玩家列表以便模拟
	playersA := make([]FactionWarPlayer, len(a.Players))
	playersB := make([]FactionWarPlayer, len(b.Players))
	copy(playersA, a.Players)
	copy(playersB, b.Players)

	// 重置存活状态
	for i := range playersA {
		playersA[i].Alive = true
		playersA[i].Kills = 0
		playersA[i].Score = 0
	}
	for i := range playersB {
		playersB[i].Alive = true
		playersB[i].Kills = 0
		playersB[i].Score = 0
	}

	var battleLogs []FactionBattleLog
	round := 0
	idxA, idxB := 0, 0

	// 5v5车轮战：按战力排序，强者先出
	sort.Slice(playersA, func(i, j int) bool { return playersA[i].Power > playersA[j].Power })
	sort.Slice(playersB, func(i, j int) bool { return playersB[i].Power > playersB[j].Power })

	for idxA < len(playersA) && idxB < len(playersB) {
		round++
		attacker := &playersA[idxA]
		defender := &playersB[idxB]

		// 战力计算：基础战力 + 随机波动
		aPower := float64(attacker.Power) * (0.85 + 0.3*rand.Float64())
		bPower := float64(defender.Power) * (0.85 + 0.3*rand.Float64())

		var log FactionBattleLog
		log.Round = round
		log.AttackerID = attacker.UserID
		log.AttackerName = attacker.UserName
		log.DefenderID = defender.UserID
		log.DefenderName = defender.UserName

		if aPower >= bPower {
			// A方胜
			attacker.Kills++
			attacker.Score += s.cfg.KillScore
			defender.Alive = false
			log.WinnerID = attacker.UserID
			log.RoundDesc = fmt.Sprintf("第%d轮: %s 击败 %s", round, attacker.UserName, defender.UserName)
			idxB++
		} else {
			// B方胜
			defender.Kills++
			defender.Score += s.cfg.KillScore
			attacker.Alive = false
			log.WinnerID = defender.UserID
			log.RoundDesc = fmt.Sprintf("第%d轮: %s 击败 %s", round, defender.UserName, attacker.UserName)
			// 攻击方败了，换下一个人
			idxA++
			// 但防守方(赢的人)留在场上继续战
			// 我们需要交换角色：idxB的人赢了，留在场上，idxA的下一个人上场
			// 重新排列：当前防守方赢了，他成为新的攻击方
			// 简单处理：让赢的人继续 vs A的下一个人
			// 但我们需要调整索引，因为defender赢了但我们在循环中用attacker和defender
			// 实际上这里idxA已经++了，而下轮循环中attacker=playersA[idxA], defender不动
			// 这正好实现了车轮战效果
		}
		battleLogs = append(battleLogs, log)
	}

	// 存活者加分
	for i := idxA; i < len(playersA); i++ {
		playersA[i].Alive = true
		playersA[i].Score += s.cfg.SurviveBonus
	}
	for i := idxB; i < len(playersB); i++ {
		playersB[i].Alive = true
		playersB[i].Score += s.cfg.SurviveBonus
	}

	// 统计总分
	totalScoreA := 0
	totalKillsA := 0
	for i := range playersA {
		totalScoreA += playersA[i].Score
		totalKillsA += playersA[i].Kills
	}
	totalScoreB := 0
	totalKillsB := 0
	for i := range playersB {
		totalScoreB += playersB[i].Score
		totalKillsB += playersB[i].Kills
	}

	// 判定胜负
	var winRecord, loseRecord *FactionWarRecord
	if totalScoreA >= totalScoreB {
		winRecord = a
		loseRecord = b
	} else {
		winRecord = b
		loseRecord = a
	}

	// 更新胜者
	winRecord.Players = playersA
	if winRecord == b {
		winRecord.Players = playersB
	}
	winRecord.TotalScore = totalScoreA
	winRecord.TotalKills = totalKillsA
	if winRecord == b {
		winRecord.TotalScore = totalScoreB
		winRecord.TotalKills = totalKillsB
	}
	winRecord.Battles = append(winRecord.Battles, battleLogs...)
	winRecord.Phase = PhaseBattle

	// 更新败者
	loseRecord.Players = playersB
	if loseRecord == b {
		loseRecord.Players = playersA
	}
	loseRecord.TotalScore = totalScoreB
	loseRecord.TotalKills = totalKillsB
	if loseRecord == b {
		loseRecord.TotalScore = totalScoreA
		loseRecord.TotalKills = totalKillsA
	}
	loseRecord.Battles = append(loseRecord.Battles, battleLogs...)
	loseRecord.Phase = PhaseBattle

	return winRecord, loseRecord
}

// saveBattleResult 保存战斗结果
func (s *FactionWarService) saveBattleResult(ctx context.Context, record *FactionWarRecord) {
	_, _ = s.factionColl().UpdateOne(ctx, bson.M{"_id": record.ID}, bson.M{"$set": bson.M{
		"phase":        record.Phase,
		"total_score":  record.TotalScore,
		"total_kills":  record.TotalKills,
		"rank":         record.Rank,
		"is_winner":    record.IsWinner,
		"players":      record.Players,
		"battles":      record.Battles,
		"updated_at":   record.UpdatedAt,
	}})
}

// settleFactionWar 结算势力争霸
func (s *FactionWarService) settleFactionWar(ctx context.Context, champion *FactionWarRecord, season *FactionSeason) {
	// 1. 更新灵脉福地主
	_, _ = s.veinColl().UpdateOne(ctx, bson.M{"_id": "spirit_vein_01"}, bson.M{"$set": bson.M{
		"owner_sect_id":   champion.SectID,
		"owner_sect_name": champion.SectName,
		"occupied_at":     time.Now(),
	}})

	// 2. 更新赛季占领信息
	_, _ = s.seasonColl().UpdateOne(ctx, bson.M{"season": season.Season}, bson.M{"$set": bson.M{
		"vein_owner_id":   champion.SectID,
		"vein_owner_name": champion.SectName,
		"status":          "finished",
	}})

	// 3. 发放奖励(简化：更新宗门资金和声望)
	//    冠军宗门奖励
	baseFunds := int64(50000)
	baseReputation := int64(1000)
	_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": champion.SectID},
		bson.M{"$inc": bson.M{
			"funds":      baseFunds,
			"reputation": baseReputation,
		}})

	// 参战弟子个人奖励
	for _, p := range champion.Players {
		if p.UserID == "" {
			continue
		}
		// 增加宗门贡献
		_, _ = s.memberColl().UpdateOne(ctx,
			bson.M{"sect_id": champion.SectID, "user_id": p.UserID},
			bson.M{"$inc": bson.M{"contribution": int64(200 + p.Kills*50)}})
	}

	// 4. 记录日志
	s.logger.Info("赛季结束，冠军占领灵脉福地，全员修炼速度提升", "season",
		season.Season, "champion", champion.SectName, "buff", s.cfg.CultivationBuff*100)
}

// ============================================================
// 查询接口
// ============================================================

// GetStatus 获取势力争霸状态
func (s *FactionWarService) GetStatus(ctx context.Context, sectID string) (map[string]interface{}, error) {
	season, err := s.GetOrCreateSeason(ctx)
	if err != nil {
		return nil, err
	}

	// 获取灵脉信息
	var vein SpiritVein
	_ = s.veinColl().FindOne(ctx, bson.M{"_id": "spirit_vein_01"}).Decode(&vein)

	// 获取宗门报名情况
	var myRecord FactionWarRecord
	err = s.factionColl().FindOne(ctx, bson.M{
		"sect_id": sectID,
		"season":  season.Season,
	}, options.FindOne().SetSort(bson.D{{Key: "enrolled_at", Value: -1}})).Decode(&myRecord)

	enrolled := err == nil

	// 报名宗门总数
	totalEnrolled, _ := s.factionColl().CountDocuments(ctx, bson.M{
		"season": season.Season,
	})

	result := map[string]interface{}{
		"season":          season.Season,
		"phase":           s.determinePhase(season),
		"battle_date":     season.BattleDate,
		"end_date":        season.EndDate,
		"vein":            vein,
		"enrolled":        enrolled,
		"my_record":       nil,
		"total_enrolled":  totalEnrolled,
		"min_sect_level":  s.cfg.MinSectLevel,
		"players_per_sect": s.cfg.PlayersPerSect,
	}

	if enrolled {
		result["my_record"] = &myRecord
	}

	if season.VeinOwnerID != "" {
		result["current_owner"] = map[string]string{
			"sect_id":   season.VeinOwnerID,
			"sect_name": season.VeinOwnerName,
		}
	}

	return result, nil
}

// determinePhase 判断当前所处阶段
func (s *FactionWarService) determinePhase(season *FactionSeason) string {
	now := time.Now()
	if now.Before(season.BattleDate) {
		return "registration"
	}
	if now.Before(season.EndDate) {
		return "battle"
	}
	return "settled"
}

// GetRanking 获取势力争霸排行榜(按积分降序)
func (s *FactionWarService) GetRanking(ctx context.Context, page, pageSize int64) ([]*FactionWarRecord, int64, error) {
	season, err := s.GetOrCreateSeason(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, _ := s.factionColl().CountDocuments(ctx, bson.M{
		"season": season.Season,
	})

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{Key: "total_score", Value: -1}}).
		SetSkip(skip).
		SetLimit(pageSize)

	cursor, err := s.factionColl().Find(ctx, bson.M{"season": season.Season}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var records []*FactionWarRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, 0, err
	}

	// 设置排名序号
	for i := range records {
		records[i].Rank = int(skip) + i + 1
	}

	return records, total, nil
}

// GetHistory 获取宗门势力争霸历史
func (s *FactionWarService) GetHistory(ctx context.Context, sectID string, limit int64) ([]*FactionWarRecord, error) {
	filter := bson.M{"sect_id": sectID}
	if limit <= 0 {
		limit = 10
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "enrolled_at", Value: -1}}).
		SetLimit(limit)

	cursor, err := s.factionColl().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []*FactionWarRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}
