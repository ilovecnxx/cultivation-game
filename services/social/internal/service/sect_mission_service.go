// Package service 宗门每日任务系统
//
// 每个成员每日可领取 3 个随机任务，类型包括采集/战斗/捐献/修炼。
// 任务进度由外部服务回调更新。每日 0 点自动刷新。
package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"cultivation-game/services/social/internal/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// SectMissionService 宗门任务业务
type SectMissionService struct {
	db *mongo.Database
}

// NewSectMissionService 创建宗门任务服务
func NewSectMissionService(db *mongo.Database) *SectMissionService {
	return &SectMissionService{db: db}
}

func (s *SectMissionService) missionColl() *mongo.Collection      { return s.db.Collection("sect_missions") }
func (s *SectMissionService) memberMissionColl() *mongo.Collection { return s.db.Collection("member_missions") }
func (s *SectMissionService) memberColl() *mongo.Collection       { return s.db.Collection("sect_members") }
func (s *SectMissionService) sectColl() *mongo.Collection         { return s.db.Collection("sects") }

// missionsPerDay 每日任务数
const missionsPerDay = 3

// missionTemplate 任务模板
type missionTemplate struct {
	MissionType         model.MissionType
	DescriptionTmpl     string
	Requirement         int32
	RewardContribution  int64
	RewardExp           int64
	RewardFunds         int64
}

var missionTemplates = []missionTemplate{
	// 采集类：提交指定材料
	{MissionType: model.MissionGathering, DescriptionTmpl: "提交灵草 x10", Requirement: 10, RewardContribution: 50, RewardExp: 100, RewardFunds: 20},
	{MissionType: model.MissionGathering, DescriptionTmpl: "提交矿石 x15", Requirement: 15, RewardContribution: 60, RewardExp: 120, RewardFunds: 25},
	{MissionType: model.MissionGathering, DescriptionTmpl: "提交木材 x20", Requirement: 20, RewardContribution: 40, RewardExp: 80, RewardFunds: 15},

	// 战斗类：击败指定怪物
	{MissionType: model.MissionCombat, DescriptionTmpl: "击败妖兽 x5", Requirement: 5, RewardContribution: 80, RewardExp: 200, RewardFunds: 30},
	{MissionType: model.MissionCombat, DescriptionTmpl: "击败魔修 x3", Requirement: 3, RewardContribution: 100, RewardExp: 250, RewardFunds: 40},
	{MissionType: model.MissionCombat, DescriptionTmpl: "击败精英怪 x1", Requirement: 1, RewardContribution: 150, RewardExp: 300, RewardFunds: 50},

	// 贡献类：捐献灵石
	{MissionType: model.MissionDonation, DescriptionTmpl: "向宗门捐献灵石 x1000", Requirement: 1000, RewardContribution: 200, RewardExp: 50, RewardFunds: 0},
	{MissionType: model.MissionDonation, DescriptionTmpl: "向宗门捐献灵石 x2000", Requirement: 2000, RewardContribution: 400, RewardExp: 100, RewardFunds: 0},
	{MissionType: model.MissionDonation, DescriptionTmpl: "向宗门捐献灵石 x5000", Requirement: 5000, RewardContribution: 1000, RewardExp: 250, RewardFunds: 0},

	// 修炼类：完成修炼时长
	{MissionType: model.MissionCultivate, DescriptionTmpl: "修炼 30 分钟", Requirement: 30, RewardContribution: 60, RewardExp: 150, RewardFunds: 10},
	{MissionType: model.MissionCultivate, DescriptionTmpl: "修炼 60 分钟", Requirement: 60, RewardContribution: 100, RewardExp: 250, RewardFunds: 20},
	{MissionType: model.MissionCultivate, DescriptionTmpl: "修炼 120 分钟", Requirement: 120, RewardContribution: 180, RewardExp: 450, RewardFunds: 35},
}

// todayDate 返回 yyyy-mm-dd 格式的今天日期
func todayDate() string {
	return time.Now().Format("2006-01-02")
}

// GetDailyMissions 获取成员当日任务（无任务则自动生成）
func (s *SectMissionService) GetDailyMissions(ctx context.Context, sectID, userID string) ([]*model.MemberMission, error) {
	date := todayDate()

	// 查询成员当日已有任务
	cursor, err := s.memberMissionColl().Find(ctx, bson.M{
		"member_id": userID,
		"date":      date,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var existing []*model.MemberMission
	if err := cursor.All(ctx, &existing); err != nil {
		return nil, err
	}

	if len(existing) > 0 {
		return existing, nil
	}

	// 无任务 -> 生成当天的宗门任务并分配
	sectMissions, err := s.generateDailyMissions(ctx, sectID, date)
	if err != nil {
		return nil, err
	}

	// 随机选取 missionsPerDay 个任务分配给成员
	perm := rand.Perm(len(sectMissions))
	for i := 0; i < missionsPerDay && i < len(perm); i++ {
		sm := sectMissions[perm[i]]
		mm := &model.MemberMission{
			ID:        uuid.New().String(),
			MemberID:  userID,
			MissionID: sm.ID,
			Progress:  0,
			Completed: false,
			Claimed:   false,
			Date:      date,
		}
		if _, err := s.memberMissionColl().InsertOne(ctx, mm); err != nil {
			return nil, err
		}
		existing = append(existing, mm)
	}

	return existing, nil
}

// generateDailyMissions 生成宗门当日任务（同宗门共享任务池）
func (s *SectMissionService) generateDailyMissions(ctx context.Context, sectID, date string) ([]*model.SectMission, error) {
	// 检查是否已生成过
	cursor, err := s.missionColl().Find(ctx, bson.M{"sect_id": sectID, "date": date})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var existing []*model.SectMission
	if err := cursor.All(ctx, &existing); err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return existing, nil
	}

	// 从模板池随机选 6 个任务作为今日宗门任务池
	perm := rand.Perm(len(missionTemplates))
	if len(perm) > 6 {
		perm = perm[:6]
	}

	for _, idx := range perm {
		tmpl := missionTemplates[idx]
		m := &model.SectMission{
			ID:                 uuid.New().String(),
			SectID:             sectID,
			MissionType:        tmpl.MissionType,
			Description:        tmpl.DescriptionTmpl,
			Requirement:        tmpl.Requirement,
			RewardContribution: tmpl.RewardContribution,
			RewardExp:          tmpl.RewardExp,
			RewardFunds:        tmpl.RewardFunds,
			Date:               date,
		}
		if _, err := s.missionColl().InsertOne(ctx, m); err != nil {
			return nil, err
		}
		existing = append(existing, m)
	}
	return existing, nil
}

// UpdateMissionProgress 更新任务进度（由外部模块调用，如采集/战斗事件）
func (s *SectMissionService) UpdateMissionProgress(ctx context.Context, memberMissionID string, delta int32) error {
	// 查找任务
	var mm model.MemberMission
	err := s.memberMissionColl().FindOne(ctx, bson.M{"_id": memberMissionID}).Decode(&mm)
	if err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}
	if mm.Completed || mm.Claimed {
		return nil // 已完成的不再更新
	}

	var sm model.SectMission
	err = s.missionColl().FindOne(ctx, bson.M{"_id": mm.MissionID}).Decode(&sm)
	if err != nil {
		return fmt.Errorf("任务定义不存在: %w", err)
	}

	newProgress := mm.Progress + delta
	if newProgress > sm.Requirement {
		newProgress = sm.Requirement
	}
	completed := newProgress >= sm.Requirement

	_, err = s.memberMissionColl().UpdateOne(ctx, bson.M{"_id": memberMissionID},
		bson.M{"$set": bson.M{
			"progress":  newProgress,
			"completed": completed,
		}},
	)
	return err
}

// ClaimMissionReward 领取任务奖励
func (s *SectMissionService) ClaimMissionReward(ctx context.Context, memberMissionID, sectID, userID string) error {
	var mm model.MemberMission
	err := s.memberMissionColl().FindOne(ctx, bson.M{"_id": memberMissionID}).Decode(&mm)
	if err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}
	if mm.Claimed {
		return fmt.Errorf("该任务奖励已领取")
	}
	if !mm.Completed {
		return fmt.Errorf("任务尚未完成")
	}

	var sm model.SectMission
	err = s.missionColl().FindOne(ctx, bson.M{"_id": mm.MissionID}).Decode(&sm)
	if err != nil {
		return fmt.Errorf("任务定义不存在: %w", err)
	}

	// 事务：发放奖励
	session, err := s.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		// 标记已领取
		if _, err := s.memberMissionColl().UpdateOne(sc, bson.M{"_id": memberMissionID},
			bson.M{"$set": bson.M{"claimed": true}}); err != nil {
			return nil, err
		}
		// 加宗门贡献
		if _, err := s.memberColl().UpdateOne(sc,
			bson.M{"sect_id": sectID, "user_id": userID},
			bson.M{"$inc": bson.M{"contribution": sm.RewardContribution}}); err != nil {
			return nil, err
		}
		// 加宗门资金
		if sm.RewardFunds > 0 {
			if _, err := s.sectColl().UpdateOne(sc, bson.M{"_id": sectID},
				bson.M{"$inc": bson.M{"funds": sm.RewardFunds}}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("领取奖励失败: %w", err)
	}

	return nil
}

// GetMissionProgress 获取成员当天任务总进度
func (s *SectMissionService) GetMissionProgress(ctx context.Context, userID string) (int, int, error) {
	date := todayDate()
	cursor, err := s.memberMissionColl().Find(ctx, bson.M{"member_id": userID, "date": date})
	if err != nil {
		return 0, 0, err
	}
	defer cursor.Close(ctx)

	var missions []*model.MemberMission
	if err := cursor.All(ctx, &missions); err != nil {
		return 0, 0, err
	}

	completed := 0
	for _, m := range missions {
		if m.Completed {
			completed++
		}
	}
	return len(missions), completed, nil
}

// CheckMissionComplete 检查任务是否满足完成条件（供外部回调）
func (s *SectMissionService) CheckMissionComplete(ctx context.Context, userID string, mType model.MissionType, delta int32) error {
	date := todayDate()

	cursor, err := s.memberMissionColl().Find(ctx, bson.M{
		"member_id": userID,
		"date":      date,
		"completed": false,
		"claimed":   false,
	})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var missions []*model.MemberMission
	if err := cursor.All(ctx, &missions); err != nil {
		return err
	}

	for _, mm := range missions {
		var sm model.SectMission
		err := s.missionColl().FindOne(ctx, bson.M{"_id": mm.MissionID, "mission_type": mType}).Decode(&sm)
		if err != nil {
			continue
		}
		_ = s.UpdateMissionProgress(ctx, mm.ID, delta)
	}
	return nil
}
