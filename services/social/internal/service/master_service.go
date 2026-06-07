// Package service 提供师徒系统的核心业务逻辑
package service

import (
	"context"
	"fmt"
	"time"

	"cultivation-game/services/social/internal/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ============================================================
// 境界常量(用于权限校验)
// ============================================================

const (
	RealmQiRefining      = "qi_refining"      // 炼气期
	RealmFoundationBuild = "foundation_build"  // 筑基期
	RealmGoldenCore      = "golden_core"       // 金丹期
	RealmNascentSoul     = "nascent_soul"      // 元婴期
)

// MaxStudents 一个师父最大收徒数
const MaxStudents = 3

// GraduateMasterExpRatio 师父获得徒弟突破修为的比例
const GraduateMasterExpRatio = 0.1

// MentorshipUpgradeRatio 师徒值折算为升级进度的比例
const MentorshipUpgradeRatio = 1.0

// BetrayalPenaltyExpRatio 叛离惩罚修为扣除比例
const BetrayalPenaltyExpRatio = 0.2

// DungeonMaxWaves 师徒副本最大波次
const DungeonMaxWaves = 10

// ============================================================
// 玩家修为接口(由 player 服务实现)
// ============================================================

// PlayerRealmGetter 获取玩家境界与修为的接口
type PlayerRealmGetter interface {
	GetPlayerRealm(ctx context.Context, userID string) (realm string, err error)
	AddPlayerExp(ctx context.Context, userID string, exp int64) error
	GetPlayerExp(ctx context.Context, userID string) (int64, error)
	GetPlayerAttrs(ctx context.Context, userID string) (map[string]int64, error)
}

// ============================================================
// MasterService 师徒服务
// ============================================================

// MasterService 师徒系统业务逻辑
type MasterService struct {
	db          *mongo.Database
	realmGetter PlayerRealmGetter
}

// NewMasterService 创建师徒服务
func NewMasterService(db *mongo.Database, realmGetter PlayerRealmGetter) *MasterService {
	return &MasterService{
		db:          db,
		realmGetter: realmGetter,
	}
}

// ============================================================
// MongoDB 集合辅助方法
// ============================================================

func (s *MasterService) relationColl() *mongo.Collection {
	return s.db.Collection("master_relations")
}

func (s *MasterService) applyColl() *mongo.Collection {
	return s.db.Collection("master_applies")
}

func (s *MasterService) missionColl() *mongo.Collection {
	return s.db.Collection("master_missions")
}

func (s *MasterService) teachColl() *mongo.Collection {
	return s.db.Collection("master_teach_records")
}

func (s *MasterService) rewardColl() *mongo.Collection {
	return s.db.Collection("master_breakthrough_rewards")
}

func (s *MasterService) dungeonColl() *mongo.Collection {
	return s.db.Collection("master_dungeon_instances")
}

func (s *MasterService) betrayalColl() *mongo.Collection {
	return s.db.Collection("master_betrayal_records")
}

func (s *MasterService) trainingColl() *mongo.Collection {
	return s.db.Collection("master_daily_trainings")
}

// ============================================================
// 通用校验
// ============================================================

// canBeMaster 检查玩家是否满足收徒条件(筑基期+)
func (s *MasterService) canBeMaster(ctx context.Context, userID string) error {
	realm, err := s.realmGetter.GetPlayerRealm(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取玩家境界失败: %w", err)
	}
	allowed := map[string]bool{
		RealmFoundationBuild: true,
		RealmGoldenCore:      true,
		RealmNascentSoul:     true,
	}
	if !allowed[realm] {
		return fmt.Errorf("境界不足: 需要筑基期及以上才能收徒")
	}
	return nil
}

// canBeStudent 检查玩家是否满足拜师条件(炼气期)
func (s *MasterService) canBeStudent(ctx context.Context, userID string) error {
	realm, err := s.realmGetter.GetPlayerRealm(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取玩家境界失败: %w", err)
	}
	if realm != RealmQiRefining {
		return fmt.Errorf("仅炼气期玩家可以拜师")
	}
	return nil
}

// countActiveStudents 统计师父当前活跃徒弟数量
func (s *MasterService) countActiveStudents(ctx context.Context, masterID string) (int64, error) {
	return s.relationColl().CountDocuments(ctx, bson.M{
		"master_id": masterID,
		"status":    model.MasterStatusActive,
	})
}

// hasActiveMaster 检查玩家是否已有师父
func (s *MasterService) hasActiveMaster(ctx context.Context, studentID string) (bool, error) {
	count, err := s.relationColl().CountDocuments(ctx, bson.M{
		"student_id": studentID,
		"status":     model.MasterStatusActive,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ============================================================
// 申请(拜师/收徒)
// ============================================================

// Apply 发起师徒申请
func (s *MasterService) Apply(ctx context.Context, fromID, fromName, toID, toName string, applyType model.MasterApplyType, message string) error {
	if fromID == toID {
		return fmt.Errorf("不能向自己申请师徒关系")
	}

	switch applyType {
	case model.ApplyAsStudent:
		if err := s.canBeStudent(ctx, fromID); err != nil {
			return err
		}
		if err := s.canBeMaster(ctx, toID); err != nil {
			return err
		}
		hasMaster, err := s.hasActiveMaster(ctx, fromID)
		if err != nil {
			return fmt.Errorf("检查拜师状态失败: %w", err)
		}
		if hasMaster {
			return fmt.Errorf("你已有师父，无法重复拜师")
		}
		count, err := s.countActiveStudents(ctx, toID)
		if err != nil {
			return fmt.Errorf("检查收徒数量失败: %w", err)
		}
		if count >= MaxStudents {
			return fmt.Errorf("该师父已收满 %d 个徒弟", MaxStudents)
		}

	case model.ApplyAsMaster:
		if err := s.canBeMaster(ctx, fromID); err != nil {
			return err
		}
		if err := s.canBeStudent(ctx, toID); err != nil {
			return err
		}
		hasMaster, err := s.hasActiveMaster(ctx, toID)
		if err != nil {
			return fmt.Errorf("检查拜师状态失败: %w", err)
		}
		if hasMaster {
			return fmt.Errorf("该玩家已有师父")
		}
		count, err := s.countActiveStudents(ctx, fromID)
		if err != nil {
			return fmt.Errorf("检查收徒数量失败: %w", err)
		}
		if count >= MaxStudents {
			return fmt.Errorf("你已收满 %d 个徒弟", MaxStudents)
		}
	}

	existing, err := s.applyColl().CountDocuments(ctx, bson.M{
		"from_id": fromID,
		"to_id":   toID,
		"status":  "pending",
	})
	if err != nil {
		return fmt.Errorf("检查重复申请失败: %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("已发送过申请，请等待对方处理")
	}

	apply := &model.MasterApply{
		ID:        uuid.New().String(),
		FromID:    fromID,
		FromName:  fromName,
		ToID:      toID,
		ToName:    toName,
		ApplyType: applyType,
		Message:   message,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	_, err = s.applyColl().InsertOne(ctx, apply)
	if err != nil {
		return fmt.Errorf("提交申请失败: %w", err)
	}
	return nil
}

// ============================================================
// 同意/拒绝申请
// ============================================================

// Accept 同意师徒申请
func (s *MasterService) Accept(ctx context.Context, applyID string) error {
	var apply model.MasterApply
	err := s.applyColl().FindOneAndUpdate(
		ctx,
		bson.M{"_id": applyID, "status": "pending"},
		bson.M{"$set": bson.M{"status": "accepted", "handled_at": time.Now()}},
	).Decode(&apply)
	if err != nil {
		return fmt.Errorf("申请不存在或已被处理: %w", err)
	}

	var masterID, masterName, studentID, studentName string
	switch apply.ApplyType {
	case model.ApplyAsStudent:
		masterID, masterName = apply.ToID, apply.ToName
		studentID, studentName = apply.FromID, apply.FromName
	case model.ApplyAsMaster:
		masterID, masterName = apply.FromID, apply.FromName
		studentID, studentName = apply.ToID, apply.ToName
	}

	count, err := s.countActiveStudents(ctx, masterID)
	if err != nil {
		return fmt.Errorf("校验收徒数量失败: %w", err)
	}
	if count >= MaxStudents {
		return fmt.Errorf("师父已收满 %d 个徒弟，无法继续收徒", MaxStudents)
	}

	hasMaster, err := s.hasActiveMaster(ctx, studentID)
	if err != nil {
		return fmt.Errorf("校验拜师状态失败: %w", err)
	}
	if hasMaster {
		return fmt.Errorf("该玩家已有师父")
	}

	// 创建师徒关系(默认为记名弟子)
	relation := &model.MasterRelation{
		ID:              uuid.New().String(),
		MasterID:        masterID,
		MasterName:      masterName,
		StudentID:       studentID,
		StudentName:     studentName,
		MasterValue:     0,
		MentorshipLevel: model.MentorshipLevelRegistered,
		Status:          model.MasterStatusActive,
		CreatedAt:       time.Now(),
	}
	_, err = s.relationColl().InsertOne(ctx, relation)
	if err != nil {
		return fmt.Errorf("创建师徒关系失败: %w", err)
	}

	if err := s.generateDailyMissions(ctx, relation.ID); err != nil {
		_ = err
	}

	return nil
}

// Reject 拒绝师徒申请
func (s *MasterService) Reject(ctx context.Context, applyID string) error {
	result, err := s.applyColl().UpdateOne(
		ctx,
		bson.M{"_id": applyID, "status": "pending"},
		bson.M{"$set": bson.M{"status": "rejected", "handled_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("处理申请失败: %w", err)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("申请不存在或已被处理")
	}
	return nil
}

// GetPendingApplies 获取待处理的申请列表
func (s *MasterService) GetPendingApplies(ctx context.Context, userID string) ([]*model.MasterApply, error) {
	cursor, err := s.applyColl().Find(ctx, bson.M{
		"to_id":  userID,
		"status": "pending",
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var applies []*model.MasterApply
	if err := cursor.All(ctx, &applies); err != nil {
		return nil, err
	}
	return applies, nil
}

// ============================================================
// 师徒关系查询
// ============================================================

// GetMyMaster 获取玩家的师父
func (s *MasterService) GetMyMaster(ctx context.Context, userID string) (*model.MasterRelation, error) {
	var rel model.MasterRelation
	err := s.relationColl().FindOne(ctx, bson.M{
		"student_id": userID,
		"status":     model.MasterStatusActive,
	}).Decode(&rel)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rel, nil
}

// GetMyStudents 获取玩家的徒弟列表
func (s *MasterService) GetMyStudents(ctx context.Context, masterID string) ([]*model.MasterRelation, error) {
	cursor, err := s.relationColl().Find(ctx, bson.M{
		"master_id": masterID,
		"status":    model.MasterStatusActive,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var relations []*model.MasterRelation
	if err := cursor.All(ctx, &relations); err != nil {
		return nil, err
	}
	return relations, nil
}

// ============================================================
// 师徒等级(亲密度)
// ============================================================

// GetMentorshipLevelInfo 获取师徒等级信息
func (s *MasterService) GetMentorshipLevelInfo(ctx context.Context, relationID string) (*ginH, error) {
	var rel model.MasterRelation
	err := s.relationColl().FindOne(ctx, bson.M{"_id": relationID}).Decode(&rel)
	if err != nil {
		return nil, fmt.Errorf("师徒关系不存在: %w", err)
	}

	currentLevel := rel.MentorshipLevel
	currentBenefits := model.MentorshipLevelBenefitsMap[currentLevel]

	// 计算下一级信息
	var nextLevel *model.MentorshipLevelBenefit
	var nextLevelName string
	var nextLevelID model.MentorshipLevel
	if currentLevel < model.MentorshipLevelSuccessor {
		nextLevelID = currentLevel + 1
		benefit := model.MentorshipLevelBenefitsMap[nextLevelID]
		nextLevel = &benefit
		nextLevelName = model.MentorshipLevelNames[nextLevelID]
	}

	// 计算升级进度
	var progressPct float64
	var mvNeeded int64
	if currentLevel < model.MentorshipLevelSuccessor {
		mvNeeded = model.MentorshipLevelBenefitsMap[currentLevel+1].MvRequired
		if mvNeeded > 0 {
			progressPct = float64(rel.MasterValue) / float64(mvNeeded) * 100
			if progressPct > 100 {
				progressPct = 100
			}
		}
	}

	return &ginH{
		"relation_id":       rel.ID,
		"current_level":     int(currentLevel),
		"current_level_name": model.MentorshipLevelNames[currentLevel],
		"current_benefits":  currentBenefits,
		"master_value":      rel.MasterValue,
		"next_level":        nextLevelID,
		"next_level_name":   nextLevelName,
		"next_benefits":     nextLevel,
		"mv_needed":         mvNeeded,
		"progress_pct":      progressPct,
	}, nil
}

// TryUpgradeMentorshipLevel 尝试提升师徒等级
func (s *MasterService) TryUpgradeMentorshipLevel(ctx context.Context, relationID string) error {
	var rel model.MasterRelation
	err := s.relationColl().FindOne(ctx, bson.M{"_id": relationID, "status": model.MasterStatusActive}).Decode(&rel)
	if err != nil {
		return fmt.Errorf("师徒关系不存在或已失效: %w", err)
	}

	if rel.MentorshipLevel >= model.MentorshipLevelSuccessor {
		return fmt.Errorf("已达最高师徒等级(衣钵传人)")
	}

	nextLevel := rel.MentorshipLevel + 1
	requiredMV := model.MentorshipLevelBenefitsMap[nextLevel].MvRequired

	if rel.MasterValue < requiredMV {
		return fmt.Errorf("师徒值不足: 需要%d点，当前%d点", requiredMV, rel.MasterValue)
	}

	// 扣除升级所需师徒值,提升等级
	_, err = s.relationColl().UpdateOne(
		ctx,
		bson.M{"_id": relationID, "master_value": bson.M{"$gte": requiredMV}},
		bson.M{
			"$set": bson.M{"mentorship_level": nextLevel},
			"$inc": bson.M{"master_value": -requiredMV},
		},
	)
	if err != nil {
		return fmt.Errorf("升级师徒等级失败: %w", err)
	}
	return nil
}

// ============================================================
// 传授功法(增强版:支持折扣)
// ============================================================

// Teach 师父传授功法给徒弟(消耗师徒值，根据师徒等级享受折扣)
func (s *MasterService) Teach(ctx context.Context, relationID, skillID, skillName string, costMV int64) error {
	var rel model.MasterRelation
	err := s.relationColl().FindOne(ctx, bson.M{
		"_id":    relationID,
		"status": model.MasterStatusActive,
	}).Decode(&rel)
	if err != nil {
		return fmt.Errorf("师徒关系不存在或已失效: %w", err)
	}

	if costMV <= 0 {
		return fmt.Errorf("消耗师徒值必须大于0")
	}

	// 根据师徒等级计算折扣
	levelBenefits := model.MentorshipLevelBenefitsMap[rel.MentorshipLevel]
	discountPct := levelBenefits.TeachDiscountPct
	actualCost := costMV - int64(float64(costMV)*discountPct/100.0)
	if actualCost < 1 {
		actualCost = 1
	}

	if rel.MasterValue < actualCost {
		levelName := model.MentorshipLevelNames[rel.MentorshipLevel]
		return fmt.Errorf("师徒值不足(当前%s等级享%.0f%%折扣): 需要%d点，当前%d点",
			levelName, discountPct, actualCost, rel.MasterValue)
	}

	_, err = s.relationColl().UpdateOne(
		ctx,
		bson.M{"_id": relationID, "master_value": bson.M{"$gte": actualCost}},
		bson.M{"$inc": bson.M{"master_value": -actualCost}},
	)
	if err != nil {
		return fmt.Errorf("扣除师徒值失败: %w", err)
	}

	record := &model.MasterTeachRecord{
		ID:           uuid.New().String(),
		RelationID:   relationID,
		MasterID:     rel.MasterID,
		StudentID:    rel.StudentID,
		SkillID:      skillID,
		SkillName:    skillName,
		CostMV:       costMV,
		ActualCostMV: actualCost,
		CreatedAt:    time.Now(),
	}
	_, err = s.teachColl().InsertOne(ctx, record)
	if err != nil {
		return fmt.Errorf("记录传授日志失败: %w", err)
	}

	return nil
}

// ============================================================
// 每日训练任务
// ============================================================

// AssignDailyTraining 师父给徒弟指定今日训练任务
func (s *MasterService) AssignDailyTraining(ctx context.Context, relationID, taskType string, target int32) error {
	var rel model.MasterRelation
	err := s.relationColl().FindOne(ctx, bson.M{
		"_id":    relationID,
		"status": model.MasterStatusActive,
	}).Decode(&rel)
	if err != nil {
		return fmt.Errorf("师徒关系不存在或已失效: %w", err)
	}

	today := time.Now().Format("2006-01-02")

	// 检查是否已有今日训练
	var existing model.DailyTraining
	err = s.trainingColl().FindOne(ctx, bson.M{
		"relation_id": relationID,
		"date":        today,
	}).Decode(&existing)
	if err == nil {
		return fmt.Errorf("今日已有训练任务，请明日再来")
	}

	// 任务描述
	descriptions := map[string]string{
		"cultivate": "完成修炼%次",
		"combat":    "击败%d个怪物",
		"alchemy":   "炼制丹药%d次",
		"explore":   "探索地图%d次",
		"dungeon":   "通关副本%d次",
	}

	if target <= 0 {
		target = 1
	}
	if target > 20 {
		target = 20
	}

	desc := descriptions[taskType]
	if desc == "" {
		desc = "完成任务%d次"
	}

	// 计算奖励
	rewardMV := int64(target) * 10
	rewardExp := int64(target) * 100

	training := &model.DailyTraining{
		ID:             uuid.New().String(),
		RelationID:     relationID,
		TaskType:       model.TrainingTaskType(taskType),
		Description:    fmt.Sprintf(desc, target),
		Target:         target,
		Progress:       0,
		RewardMV:       rewardMV,
		RewardExp:      rewardExp,
		Completed:      false,
		MasterClaimed:  false,
		StudentClaimed: false,
		Date:           today,
		CreatedAt:      time.Now(),
	}

	// 写入关系表
	_, err = s.trainingColl().InsertOne(ctx, training)
	if err != nil {
		return fmt.Errorf("创建训练任务失败: %w", err)
	}

	_, err = s.relationColl().UpdateOne(
		ctx,
		bson.M{"_id": relationID},
		bson.M{
			"$set": bson.M{
				"daily_training_id": training.ID,
				"training_progress": 0,
				"training_target":   target,
				"training_date":     today,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("更新关系训练信息失败: %w", err)
	}

	return nil
}

// GetDailyTraining 获取今日训练任务
func (s *MasterService) GetDailyTraining(ctx context.Context, relationID string) (*model.DailyTraining, error) {
	today := time.Now().Format("2006-01-02")

	var training model.DailyTraining
	err := s.trainingColl().FindOne(ctx, bson.M{
		"relation_id": relationID,
		"date":        today,
	}).Decode(&training)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询训练任务失败: %w", err)
	}
	return &training, nil
}

// UpdateTrainingProgress 更新训练任务进度
func (s *MasterService) UpdateTrainingProgress(ctx context.Context, missionID string, addProgress int32) error {
	if addProgress <= 0 {
		return fmt.Errorf("进度增量必须大于0")
	}

	result, err := s.trainingColl().UpdateOne(
		ctx,
		bson.M{"_id": missionID, "completed": false},
		bson.M{"$inc": bson.M{"progress": addProgress}},
	)
	if err != nil {
		return fmt.Errorf("更新训练进度失败: %w", err)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("训练任务不存在或已完成")
	}

	var training model.DailyTraining
	err = s.trainingColl().FindOne(ctx, bson.M{"_id": missionID}).Decode(&training)
	if err != nil {
		return err
	}

	if training.Progress >= training.Target {
		_, err := s.trainingColl().UpdateOne(
			ctx,
			bson.M{"_id": missionID},
			bson.M{"$set": bson.M{"completed": true}},
		)
		if err != nil {
			return fmt.Errorf("标记训练完成失败: %w", err)
		}
	}
	return nil
}

// ClaimTrainingReward 领取训练任务奖励
func (s *MasterService) ClaimTrainingReward(ctx context.Context, missionID, claimerID string) error {
	var training model.DailyTraining
	err := s.trainingColl().FindOne(ctx, bson.M{
		"_id":       missionID,
		"completed": true,
	}).Decode(&training)
	if err != nil {
		return fmt.Errorf("训练任务未完成或不存在: %w", err)
	}

	// 检查师徒关系以确认领取者身份
	var rel model.MasterRelation
	err = s.relationColl().FindOne(ctx, bson.M{"_id": training.RelationID}).Decode(&rel)
	if err != nil {
		return fmt.Errorf("师徒关系不存在: %w", err)
	}

	isMaster := claimerID == rel.MasterID
	isStudent := claimerID == rel.StudentID

	if !isMaster && !isStudent {
		return fmt.Errorf("您不是该师徒关系成员")
	}

	// 检查是否已领取
	if isMaster && training.MasterClaimed {
		return fmt.Errorf("师父已领取奖励")
	}
	if isStudent && training.StudentClaimed {
		return fmt.Errorf("徒弟已领取奖励")
	}

	// 标记领取
	setField := "master_claimed"
	if isStudent {
		setField = "student_claimed"
	}

	_, err = s.trainingColl().UpdateOne(
		ctx,
		bson.M{"_id": missionID},
		bson.M{"$set": bson.M{setField: true}},
	)
	if err != nil {
		return fmt.Errorf("标记奖励领取失败: %w", err)
	}

	// 发奖励
	if err := s.realmGetter.AddPlayerExp(ctx, claimerID, training.RewardExp); err != nil {
		return fmt.Errorf("发放修为奖励失败: %w", err)
	}

	// 给关系加师徒值
	_, err = s.relationColl().UpdateOne(
		ctx,
		bson.M{"_id": training.RelationID},
		bson.M{"$inc": bson.M{"master_value": training.RewardMV}},
	)
	if err != nil {
		return fmt.Errorf("发放师徒值奖励失败: %w", err)
	}

	return nil
}

// ============================================================
// 每日师徒任务
// ============================================================

// generateDailyMissions 为新的师徒关系生成当日任务
func (s *MasterService) generateDailyMissions(ctx context.Context, relationID string) error {
	today := time.Now().Format("2006-01-02")

	type missionDef struct {
		missionType model.MasterMissionType
		required    int32
		rewardMV    int64
	}
	defs := []missionDef{
		{missionType: model.MasterMissionCultivate, required: 1, rewardMV: 50},
		{missionType: model.MasterMissionCombat, required: 5, rewardMV: 30},
		{missionType: model.MasterMissionTribute, required: 1, rewardMV: 20},
		{missionType: model.MasterMissionDungeon, required: 1, rewardMV: 80},
	}

	for _, d := range defs {
		mission := &model.MasterMission{
			ID:          uuid.New().String(),
			RelationID:  relationID,
			MissionType: d.missionType,
			Required:    d.required,
			Progress:    0,
			Completed:   false,
			Claimed:     false,
			Date:        today,
			RewardMV:    d.rewardMV,
		}
		if _, err := s.missionColl().InsertOne(ctx, mission); err != nil {
			return err
		}
	}
	return nil
}

// GetDailyMissions 获取师徒关系的今日任务列表
func (s *MasterService) GetDailyMissions(ctx context.Context, relationID string) ([]*model.MasterMission, error) {
	today := time.Now().Format("2006-01-02")

	cursor, err := s.missionColl().Find(ctx, bson.M{
		"relation_id": relationID,
		"date":        today,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var missions []*model.MasterMission
	if err := cursor.All(ctx, &missions); err != nil {
		return nil, err
	}

	if len(missions) == 0 {
		if err := s.generateDailyMissions(ctx, relationID); err != nil {
			return nil, err
		}
		cursor, err := s.missionColl().Find(ctx, bson.M{
			"relation_id": relationID,
			"date":        today,
		})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)
		if err := cursor.All(ctx, &missions); err != nil {
			return nil, err
		}
	}

	return missions, nil
}

// UpdateMissionProgress 更新师徒任务进度
func (s *MasterService) UpdateMissionProgress(ctx context.Context, missionID string, addProgress int32) error {
	if addProgress <= 0 {
		return fmt.Errorf("进度增量必须大于0")
	}

	result, err := s.missionColl().UpdateOne(
		ctx,
		bson.M{"_id": missionID, "completed": false, "claimed": false},
		bson.M{"$inc": bson.M{"progress": addProgress}},
	)
	if err != nil {
		return fmt.Errorf("更新任务进度失败: %w", err)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("任务不存在或已完成")
	}

	var mission model.MasterMission
	err = s.missionColl().FindOne(ctx, bson.M{"_id": missionID}).Decode(&mission)
	if err != nil {
		return err
	}
	if mission.Progress >= mission.Required {
		_, err := s.missionColl().UpdateOne(
			ctx,
			bson.M{"_id": missionID},
			bson.M{"$set": bson.M{"completed": true}},
		)
		if err != nil {
			return fmt.Errorf("标记任务完成失败: %w", err)
		}
	}
	return nil
}

// ClaimMission 领取师徒任务奖励
func (s *MasterService) ClaimMission(ctx context.Context, missionID string) error {
	var mission model.MasterMission
	err := s.missionColl().FindOneAndUpdate(
		ctx,
		bson.M{"_id": missionID, "completed": true, "claimed": false},
		bson.M{"$set": bson.M{"claimed": true}},
	).Decode(&mission)
	if err != nil {
		return fmt.Errorf("任务未完成或已领奖: %w", err)
	}

	_, err = s.relationColl().UpdateOne(
		ctx,
		bson.M{"_id": mission.RelationID},
		bson.M{"$inc": bson.M{"master_value": mission.RewardMV}},
	)
	if err != nil {
		return fmt.Errorf("发放师徒值奖励失败: %w", err)
	}
	return nil
}

// ============================================================
// 徒弟突破奖励
// ============================================================

// OnStudentBreakthrough 徒弟突破时调用,师父获得徒弟获得修为的10%
func (s *MasterService) OnStudentBreakthrough(ctx context.Context, studentID, newRealm string, studentExpGained int64) error {
	var rel model.MasterRelation
	err := s.relationColl().FindOne(ctx, bson.M{
		"student_id": studentID,
		"status":     model.MasterStatusActive,
	}).Decode(&rel)
	if err == mongo.ErrNoDocuments {
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询师徒关系失败: %w", err)
	}

	// 师徒等级加成
	benefits := model.MentorshipLevelBenefitsMap[rel.MentorshipLevel]
	masterExpRatio := GraduateMasterExpRatio * (1 + benefits.ExpBonusPct/100.0)
	masterExp := int64(float64(studentExpGained) * masterExpRatio)
	if masterExp <= 0 {
		return nil
	}

	if err := s.realmGetter.AddPlayerExp(ctx, rel.MasterID, masterExp); err != nil {
		return fmt.Errorf("给师父发放修为奖励失败: %w", err)
	}

	record := &model.MasterBreakthroughReward{
		ID:           uuid.New().String(),
		MasterID:     rel.MasterID,
		StudentID:    studentID,
		StudentRealm: newRealm,
		MasterExp:    masterExp,
		CreatedAt:    time.Now(),
	}
	_, err = s.rewardColl().InsertOne(ctx, record)
	if err != nil {
		return fmt.Errorf("记录突破奖励失败: %w", err)
	}
	return nil
}

// ============================================================
// 出师(增强版:根据师徒等级加成)
// ============================================================

// Graduate 徒弟出师(达到金丹期自动触发或手动)
func (s *MasterService) Graduate(ctx context.Context, relationID string) error {
	var rel model.MasterRelation
	err := s.relationColl().FindOne(ctx, bson.M{
		"_id":    relationID,
		"status": model.MasterStatusActive,
	}).Decode(&rel)
	if err != nil {
		return fmt.Errorf("师徒关系不存在或已失效: %w", err)
	}

	studentRealm, err := s.realmGetter.GetPlayerRealm(ctx, rel.StudentID)
	if err != nil {
		return fmt.Errorf("获取徒弟境界失败: %w", err)
	}
	if studentRealm != RealmGoldenCore {
		return fmt.Errorf("徒弟未达到金丹期，无法出师(当前: %s)", studentRealm)
	}

	// 根据师徒等级计算加成
	benefits := model.MentorshipLevelBenefitsMap[rel.MentorshipLevel]
	bonusPct := 1.0 + benefits.GraduationBonusPct/100.0

	reward := s.getGraduateReward(rel.MasterValue, bonusPct)

	// 发放奖励
	if err := s.realmGetter.AddPlayerExp(ctx, rel.MasterID, reward.MasterExp); err != nil {
		return fmt.Errorf("给师父发放出师修为奖励失败: %w", err)
	}
	if err := s.realmGetter.AddPlayerExp(ctx, rel.StudentID, reward.StudentExp); err != nil {
		return fmt.Errorf("给徒弟发放出师修为奖励失败: %w", err)
	}

	now := time.Now()
	_, err = s.relationColl().UpdateOne(
		ctx,
		bson.M{"_id": relationID},
		bson.M{
			"$set": bson.M{
				"status":       model.MasterStatusGraduated,
				"graduated_at": now,
			},
			"$inc": bson.M{"master_value": reward.MasterMV + reward.StudentMV},
		},
	)
	if err != nil {
		return fmt.Errorf("更新出师状态失败: %w", err)
	}

	return nil
}

// getGraduateReward 根据师徒值和等级加成计算出师奖励
func (s *MasterService) getGraduateReward(masterValue int64, bonusPct float64) model.GraduateReward {
	base := model.GraduateReward{
		MasterExp:   int64(float64(5000) * bonusPct),
		StudentExp:  int64(float64(10000) * bonusPct),
		MasterMV:    0,
		StudentMV:   0,
		MasterItems: []model.ItemReward{
			{ItemID: "master_graduate_token", ItemName: "桃李令", Quantity: 1},
		},
		StudentItems: []model.ItemReward{
			{ItemID: "graduate_certificate", ItemName: "出师凭证", Quantity: 1},
			{ItemID: "spirit_stone", ItemName: "灵石", Quantity: 5000},
		},
	}

	bonus := (masterValue / 100) * int64(float64(500) * bonusPct)
	base.MasterExp += bonus
	base.StudentExp += bonus * 2

	if masterValue > 0 {
		base.MasterMV = int64(float64(masterValue/2) * bonusPct)
		base.StudentMV = int64(float64(masterValue/2) * bonusPct)
	}
	return base
}

// ============================================================
// 逐出师门
// ============================================================

// Kick 师父将徒弟逐出师门
func (s *MasterService) Kick(ctx context.Context, relationID string) error {
	result, err := s.relationColl().UpdateOne(
		ctx,
		bson.M{
			"_id":    relationID,
			"status": model.MasterStatusActive,
		},
		bson.M{"$set": bson.M{"status": model.MasterStatusKicked}},
	)
	if err != nil {
		return fmt.Errorf("逐出师门失败: %w", err)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("师徒关系不存在或已失效")
	}
	return nil
}

// ============================================================
// 叛离师门
// ============================================================

// Betray 徒弟叛离师门
func (s *MasterService) Betray(ctx context.Context, relationID string) error {
	var rel model.MasterRelation
	err := s.relationColl().FindOneAndUpdate(
		ctx,
		bson.M{
			"_id":    relationID,
			"status": model.MasterStatusActive,
		},
		bson.M{"$set": bson.M{"status": model.MasterStatusBetrayed}},
	).Decode(&rel)
	if err != nil {
		return fmt.Errorf("师徒关系不存在或已失效: %w", err)
	}

	// 扣除徒弟修为(惩罚)
	studentExp, err := s.realmGetter.GetPlayerExp(ctx, rel.StudentID)
	if err != nil {
		return fmt.Errorf("获取徒弟修为失败: %w", err)
	}

	penaltyExp := int64(float64(studentExp) * BetrayalPenaltyExpRatio)
	if penaltyExp > 0 {
		// 扣修为(通过加负值)
		if err := s.realmGetter.AddPlayerExp(ctx, rel.StudentID, -penaltyExp); err != nil {
			return fmt.Errorf("扣除徒弟修为失败: %w", err)
		}
	}

	// 师徒值损失
	mvLost := rel.MasterValue
	levelLost := rel.MentorshipLevel

	// 记录叛离
	record := &model.MasterBetrayalRecord{
		ID:                 uuid.New().String(),
		RelationID:         relationID,
		MasterID:           rel.MasterID,
		MasterName:         rel.MasterName,
		StudentID:          rel.StudentID,
		StudentName:        rel.StudentName,
		MasterValueLost:    mvLost,
		MentorshipLevelLost: levelLost,
		PenaltyExp:         penaltyExp,
		BetrayedAt:         time.Now(),
	}
	_, err = s.betrayalColl().InsertOne(ctx, record)
	if err != nil {
		return fmt.Errorf("记录叛离失败: %w", err)
	}

	return nil
}

// GetBetrayalHistory 获取玩家的叛离历史
func (s *MasterService) GetBetrayalHistory(ctx context.Context, userID string) ([]*model.MasterBetrayalRecord, error) {
	cursor, err := s.betrayalColl().Find(ctx, bson.M{
		"$or": []bson.M{
			{"master_id": userID},
			{"student_id": userID},
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []*model.MasterBetrayalRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// ============================================================
// 师徒副本
// ============================================================

// CreateDungeonInstance 创建师徒副本实例
func (s *MasterService) CreateDungeonInstance(ctx context.Context, relationID string, dungeonLevel int) (*model.MasterDungeonInstance, error) {
	var rel model.MasterRelation
	err := s.relationColl().FindOne(ctx, bson.M{
		"_id":    relationID,
		"status": model.MasterStatusActive,
	}).Decode(&rel)
	if err != nil {
		return nil, fmt.Errorf("师徒关系不存在或已失效: %w", err)
	}

	if dungeonLevel < 1 {
		dungeonLevel = 1
	}
	if dungeonLevel > DungeonMaxWaves {
		dungeonLevel = DungeonMaxWaves
	}

	// 获取双方属性
	masterAttrs, err := s.realmGetter.GetPlayerAttrs(ctx, rel.MasterID)
	if err != nil {
		return nil, fmt.Errorf("获取师父属性失败: %w", err)
	}
	studentAttrs, err := s.realmGetter.GetPlayerAttrs(ctx, rel.StudentID)
	if err != nil {
		return nil, fmt.Errorf("获取徒弟属性失败: %w", err)
	}

	maxWave := dungeonLevel * 2

	instance := &model.MasterDungeonInstance{
		ID:              uuid.New().String(),
		RelationID:      relationID,
		MasterID:        rel.MasterID,
		StudentID:       rel.StudentID,
		DungeonLevel:    dungeonLevel,
		MasterHP:        getAttrOrDefault(masterAttrs, "hp", 1000),
		MasterMaxHP:     getAttrOrDefault(masterAttrs, "hp", 1000),
		StudentHP:       getAttrOrDefault(studentAttrs, "hp", 500),
		StudentMaxHP:    getAttrOrDefault(studentAttrs, "hp", 500),
		CurrentWave:     1,
		MaxWave:         maxWave,
		TotalMasterDmg:  0,
		TotalStudentDmg: 0,
		Status:          "pending",
		RewardClaimed:   false,
		CreatedAt:       time.Now(),
	}

	_, err = s.dungeonColl().InsertOne(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("创建副本实例失败: %w", err)
	}

	return instance, nil
}

// EnterDungeon 进入副本(开始挑战)
func (s *MasterService) EnterDungeon(ctx context.Context, instanceID string) error {
	result, err := s.dungeonColl().UpdateOne(
		ctx,
		bson.M{"_id": instanceID, "status": "pending"},
		bson.M{"$set": bson.M{"status": "active"}},
	)
	if err != nil {
		return fmt.Errorf("进入副本失败: %w", err)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("副本不存在或已在战斗中")
	}
	return nil
}

// DungeonWaveComplete 完成一波战斗
func (s *MasterService) DungeonWaveComplete(ctx context.Context, instanceID string, masterDmg, studentDmg int64) (*model.MasterDungeonInstance, error) {
	var instance model.MasterDungeonInstance
	err := s.dungeonColl().FindOne(ctx, bson.M{"_id": instanceID, "status": "active"}).Decode(&instance)
	if err != nil {
		return nil, fmt.Errorf("副本不存在或未激活: %w", err)
	}

	nextWave := instance.CurrentWave + 1
	isLastWave := nextWave > instance.MaxWave

	update := bson.M{
		"$inc": bson.M{
			"current_wave":      1,
			"total_master_dmg":  masterDmg,
			"total_student_dmg": studentDmg,
		},
	}

	if isLastWave {
		update["$set"] = bson.M{
			"status":       "completed",
			"completed_at": time.Now(),
		}
	} else {
		update["$set"] = bson.M{
			"current_wave": nextWave,
		}
	}

	_, err = s.dungeonColl().UpdateOne(
		ctx,
		bson.M{"_id": instanceID},
		update,
	)
	if err != nil {
		return nil, fmt.Errorf("更新副本进度失败: %w", err)
	}

	if isLastWave {
		instance.Status = "completed"
		instance.CompletedAt = time.Now()
	} else {
		instance.CurrentWave = nextWave
	}
	instance.TotalMasterDmg += masterDmg
	instance.TotalStudentDmg += studentDmg

	return &instance, nil
}

// ClaimDungeonReward 领取副本奖励
func (s *MasterService) ClaimDungeonReward(ctx context.Context, instanceID, claimerID string) (*model.MasterDungeonReward, error) {
	var instance model.MasterDungeonInstance
	err := s.dungeonColl().FindOneAndUpdate(
		ctx,
		bson.M{
			"_id":           instanceID,
			"status":        "completed",
			"reward_claimed": false,
		},
		bson.M{"$set": bson.M{"reward_claimed": true}},
	).Decode(&instance)
	if err != nil {
		return nil, fmt.Errorf("副本未完成或奖励已领取: %w", err)
	}

	// 计算奖励
	levelBonus := 1.0 + float64(instance.DungeonLevel)*0.2
	if levelBonus > 3.0 {
		levelBonus = 3.0
	}

	reward := &model.MasterDungeonReward{
		MasterMV:   int64(50 * levelBonus),
		StudentMV:  int64(50 * levelBonus),
		MasterExp:  int64(200 * levelBonus),
		StudentExp: int64(500 * levelBonus),
		Items: []model.ItemReward{
			{ItemID: "spirit_stone", ItemName: "灵石", Quantity: int64(200 * levelBonus)},
		},
	}

	// 给师徒关系加师徒值
	_, err = s.relationColl().UpdateOne(
		ctx,
		bson.M{"_id": instance.RelationID},
		bson.M{"$inc": bson.M{"master_value": reward.MasterMV + reward.StudentMV}},
	)
	if err != nil {
		return nil, fmt.Errorf("发放副本师徒值失败: %w", err)
	}

	// 给个人发修为
	if err := s.realmGetter.AddPlayerExp(ctx, instance.MasterID, reward.MasterExp); err != nil {
		return nil, fmt.Errorf("发放副本修为奖励失败: %w", err)
	}
	if err := s.realmGetter.AddPlayerExp(ctx, instance.StudentID, reward.StudentExp); err != nil {
		return nil, fmt.Errorf("发放副本修为奖励失败: %w", err)
	}

	return reward, nil
}

// GetDungeonInstance 获取副本实例状态
func (s *MasterService) GetDungeonInstance(ctx context.Context, instanceID string) (*model.MasterDungeonInstance, error) {
	var instance model.MasterDungeonInstance
	err := s.dungeonColl().FindOne(ctx, bson.M{"_id": instanceID}).Decode(&instance)
	if err != nil {
		return nil, fmt.Errorf("副本不存在: %w", err)
	}
	return &instance, nil
}

// ============================================================
// 师徒值查询
// ============================================================

// GetMasterValue 获取师徒关系的当前师徒值
func (s *MasterService) GetMasterValue(ctx context.Context, relationID string) (int64, error) {
	var rel model.MasterRelation
	err := s.relationColl().FindOne(
		ctx,
		bson.M{"_id": relationID},
		options.FindOne().SetProjection(bson.M{"master_value": 1}),
	).Decode(&rel)
	if err != nil {
		return 0, fmt.Errorf("查询师徒值失败: %w", err)
	}
	return rel.MasterValue, nil
}

// ============================================================
// 辅助方法
// ============================================================

// ginH is a shortcut for map[string]interface{}
type ginH map[string]interface{}

func getAttrOrDefault(attrs map[string]int64, key string, defaultVal int64) int64 {
	if v, ok := attrs[key]; ok {
		return v
	}
	return defaultVal
}
