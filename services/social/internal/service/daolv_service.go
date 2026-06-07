// Package service 提供道侣双修系统的业务逻辑 (金丹期解锁)
package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"cultivation-game/services/social/internal/model"
	"cultivation-game/services/social/internal/repository"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// DaoLvService 道侣业务服务
type DaoLvService struct {
	repo *repository.DaoLvRepo
	db   *mongo.Database // 用于直接查询玩家数据
}

// NewDaoLvService 创建道侣服务
func NewDaoLvService(repo *repository.DaoLvRepo, db *mongo.Database) *DaoLvService {
	return &DaoLvService{
		repo: repo,
		db:   db,
	}
}

// ============================================================
// 辅助常量
// ============================================================

const (
	ProposalCooldown     = 24 * time.Hour // 求婚冷却时间
	DailyCultivateLimit  = 2 * time.Hour  // 每日双修上限
	BaseCultivateRate    = 10.0           // 基础修炼速率(修为/秒)
	MaxDurationBonus     = 0.50           // 伴侣时长最大加成(30天)
	DurationBonusDays    = 30             // 达到最大加成的天数
	RealmGapPenaltyStart = 3              // 境界差超过3开始惩罚
	StatSharePercent     = 0.05           // 属性共享百分比5%
	TeleportCooldown     = 30 * time.Minute
)

// ============================================================
// 1. 求婚系统
// ============================================================

// ProposeRequest 求婚请求
type ProposeRequest struct {
	FromID       uint64
	FromName     string
	ToID         uint64
	ToName       string
	Message      string
	GiftItemID   string
	GiftItemName string
}

// Propose 发送求婚申请
func (s *DaoLvService) Propose(ctx context.Context, req *ProposeRequest) error {
	if req.FromID == req.ToID {
		return fmt.Errorf("不能向自己求婚")
	}

	// 检查双方是否已有道侣
	for _, pid := range []uint64{req.FromID, req.ToID} {
		exists, err := s.repo.IsPlayerInRelation(ctx, pid)
		if err != nil {
			return fmt.Errorf("检查道侣关系失败: %w", err)
		}
		if exists {
			return fmt.Errorf("对方已有道侣")
		}
	}

	// 检查是否有待处理的申请(双向)
	existing, err := s.repo.FindPendingProposal(ctx, req.FromID, req.ToID)
	if err == nil && existing != nil {
		return fmt.Errorf("已有待处理的求婚申请")
	}

	// 检查求婚冷却(24小时)
	since := time.Now().Add(-ProposalCooldown)
	rejected, err := s.repo.FindRecentRejectedProposal(ctx, req.FromID, since)
	if err == nil && rejected != nil {
		return fmt.Errorf("求婚被拒后需等待24小时才能再次求婚")
	}

	proposal := &model.DaolvProposal{
		ID:           uuid.New().String(),
		FromID:       req.FromID,
		FromName:     req.FromName,
		ToID:         req.ToID,
		ToName:       req.ToName,
		Message:      req.Message,
		GiftItemID:   req.GiftItemID,
		GiftItemName: req.GiftItemName,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}
	if err := s.repo.InsertProposal(ctx, proposal); err != nil {
		return fmt.Errorf("发送求婚申请失败: %w", err)
	}
	return nil
}

// HandleProposal 处理求婚(接受/拒绝)
func (s *DaoLvService) HandleProposal(ctx context.Context, proposalID string, accept bool) error {
	proposal, err := s.repo.FindProposalByID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("求婚申请不存在")
	}
	if proposal.Status != "pending" {
		return fmt.Errorf("该申请已被处理")
	}

	if accept {
		return s.acceptProposal(ctx, proposal)
	}
	return s.rejectProposal(ctx, proposal)
}

// acceptProposal 接受求婚
func (s *DaoLvService) acceptProposal(ctx context.Context, proposal *model.DaolvProposal) error {
	// 再次检查双方是否已有道侣
	for _, pid := range []uint64{proposal.FromID, proposal.ToID} {
		exists, err := s.repo.IsPlayerInRelation(ctx, pid)
		if err != nil {
			return fmt.Errorf("检查道侣关系失败: %w", err)
		}
		if exists {
			return fmt.Errorf("对方已有道侣")
		}
	}

	// 计算契合度
	compatibility := s.GetCompatibility(ctx, proposal.FromID, proposal.ToID)

	// 确定初始等级
	level := model.GetDaolvLevel(0)
	initialSkills := s.getInitialSkills(level)

	// 创建道侣关系
	rel := &model.DaolvRelation{
		ID:                 uuid.New().String(),
		PlayerA:            proposal.FromID,
		PlayerB:            proposal.ToID,
		Intimacy:           10, // 结为道侣初始亲密度
		Compatibility:      compatibility,
		Level:              level,
		Skills:             initialSkills,
		GiftItemA:          proposal.GiftItemID,
		DailyCultivated:    0,
		DailyCultivateDate: time.Now().Format("2006-01-02"),
		StartedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		Status:             "normal",
	}
	if err := s.repo.InsertRelation(ctx, rel); err != nil {
		return fmt.Errorf("结为道侣失败: %w", err)
	}

	// 更新申请状态
	if err := s.repo.UpdateProposalStatus(ctx, proposal.ID, "accepted"); err != nil {
		return fmt.Errorf("更新申请状态失败: %w", err)
	}

	// 生成每日任务
	if err := s.refreshDailyTasks(ctx, rel.ID); err != nil {
		// 非致命错误
		_ = err
	}

	return nil
}

// rejectProposal 拒绝求婚
func (s *DaoLvService) rejectProposal(ctx context.Context, proposal *model.DaolvProposal) error {
	if err := s.repo.UpdateProposalStatus(ctx, proposal.ID, "rejected"); err != nil {
		return fmt.Errorf("拒绝求婚失败: %w", err)
	}
	return nil
}

// getInitialSkills 根据等级获取初始技能
func (s *DaoLvService) getInitialSkills(level string) []string {
	// 初识阶段只有双修，没有额外技能
	return []string{}
}

// ============================================================
// 2. 双修系统
// ============================================================

// DualCultivateRequest 双修请求
type DualCultivateRequest struct {
	PlayerID  uint64
	Duration  time.Duration // 修炼时长(最大2小时)
	Technique string        // 双修功法(可选)
}

// DualCultivateResult 双修结果
type DualCultivateResult struct {
	CultivationGained int64   `json:"cultivation_gained"`
	IntimacyGained    int64   `json:"intimacy_gained"`
	BonusMultiplier   float64 `json:"bonus_multiplier"`
	DurationSeconds   int64   `json:"duration_seconds"`
	PartnerName       string  `json:"partner_name"`
}

// StartDualCultivate 开始双修
func (s *DaoLvService) StartDualCultivate(ctx context.Context, req *DualCultivateRequest) (*DualCultivateResult, error) {
	rel, err := s.repo.FindRelationByPlayer(ctx, req.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("未找到道侣关系，请先结为道侣")
	}
	if rel.Status != "normal" {
		return nil, fmt.Errorf("道侣关系已解除")
	}

	partnerID := s.repo.GetPartnerID(rel, req.PlayerID)

	// 检查每日双修上限
	if err := s.checkDailyLimit(ctx, rel); err != nil {
		return nil, err
	}

	// 限制最大时长
	if req.Duration > DailyCultivateLimit {
		req.Duration = DailyCultivateLimit
	}

	// 计算剩余可修炼时间
	remaining := DailyCultivateLimit - time.Duration(rel.DailyCultivated)*time.Second
	if req.Duration > remaining {
		req.Duration = remaining
	}
	if req.Duration <= 0 {
		return nil, fmt.Errorf("今日双修时间已达上限")
	}

	// 计算加成倍率
	bonusMultiplier := s.calculateCultivateBonus(ctx, rel, req.PlayerID, req.Technique)

	// 计算修为收益
	totalSeconds := req.Duration.Seconds()
	effectiveRate := BaseCultivateRate * bonusMultiplier
	gained := int64(math.Round(effectiveRate * totalSeconds))

	// 亲密度增加（每分钟+1，双修功法额外加成）
	intimacyGain := int64(totalSeconds / 60)
	if intimacyGain < 1 {
		intimacyGain = 1
	}
	if req.Technique != "" {
		intimacyGain = int64(float64(intimacyGain) * 1.5)
	}

	// 更新每日修炼时间和亲密度
	today := time.Now().Format("2006-01-02")
	newDaily := rel.DailyCultivated + int64(totalSeconds)
	if newDaily > int64(DailyCultivateLimit.Seconds()) {
		newDaily = int64(DailyCultivateLimit.Seconds())
	}

	if err := s.repo.UpdateRelation(ctx, rel.ID, bson.M{
		"$inc": bson.M{
			"intimacy":         int(intimacyGain),
			"daily_cultivated": int64(totalSeconds),
		},
	}); err != nil {
		return nil, fmt.Errorf("更新双修记录失败: %w", err)
	}

	// 更新日期(跨天重置)
	if err := s.repo.SetRelationField(ctx, rel.ID, "daily_cultivate_date", today); err != nil {
		_ = err
	}

	// 检查是否需要升级
	s.checkAndUpdateLevel(ctx, rel.ID)

	// 更新道侣状态中的双修字段
	s.repo.SetRelationField(ctx, rel.ID, "last_cultivate_at", time.Now())

	partnerName := s.fetchPlayerName(ctx, partnerID)

	return &DualCultivateResult{
		CultivationGained: gained,
		IntimacyGained:    intimacyGain,
		BonusMultiplier:   bonusMultiplier,
		DurationSeconds:   int64(totalSeconds),
		PartnerName:       partnerName,
	}, nil
}

// checkDailyLimit 检查每日双修上限
func (s *DaoLvService) checkDailyLimit(ctx context.Context, rel *model.DaolvRelation) error {
	today := time.Now().Format("2006-01-02")
	if rel.DailyCultivateDate != today {
		// 跨天重置
		_ = s.repo.SetRelationField(ctx, rel.ID, "daily_cultivate_date", today)
		_ = s.repo.SetRelationField(ctx, rel.ID, "daily_cultivated", 0)
		rel.DailyCultivated = 0
		return nil
	}

	if rel.DailyCultivated >= int64(DailyCultivateLimit.Seconds()) {
		return fmt.Errorf("今日双修时间已达上限（2小时）")
	}
	return nil
}

// calculateCultivateBonus 计算双修加成倍率
//
//	加成 = 基础20% + 契合度加成 + 伴侣时长加成 - 境界差惩罚
func (s *DaoLvService) calculateCultivateBonus(ctx context.Context, rel *model.DaolvRelation, playerID uint64, technique string) float64 {
	// 基础倍率: 1.0 (100%)
	multiplier := 1.0

	// 1. 道侣双修基础加成 20%
	daolvBonus := 0.20

	// 2. 契合度加成: affinity * 0.5%
	affinityBonus := rel.Compatibility * 0.005
	daolvBonus += affinityBonus

	// 3. 伴侣时长加成 (结为道侣越久加成越高, 最多+50%)
	daysSinceStart := time.Since(rel.StartedAt).Hours() / 24
	durationBonus := math.Min(float64(daysSinceStart)/float64(DurationBonusDays), 1.0) * MaxDurationBonus
	daolvBonus += durationBonus

	// 4. 境界差惩罚
	partnerID := s.repo.GetPartnerID(rel, playerID)
	playerStage := s.fetchCultivationStage(context.Background(), playerID)
	partnerStage := s.fetchCultivationStage(context.Background(), partnerID)
	stageDiff := math.Abs(float64(playerStage - partnerStage))

	if stageDiff > float64(RealmGapPenaltyStart) {
		// 境界差超过3级，每多1级惩罚5%
		gapPenalty := (stageDiff - float64(RealmGapPenaltyStart)) * 0.05
		daolvBonus -= gapPenalty
	}

	// 5. 特殊双修功法加成
	if technique != "" {
		techBonus := s.getTechniqueBonus(technique)
		daolvBonus += techBonus
	}

	// 确保加成不低于0
	if daolvBonus < 0 {
		daolvBonus = 0
	}

	// 确保加成不超过上限
	if daolvBonus > 1.0 {
		daolvBonus = 1.0
	}

	return multiplier + daolvBonus
}

// getTechniqueBonus 获取双修功法加成
func (s *DaoLvService) getTechniqueBonus(technique string) float64 {
	bonuses := map[string]float64{
		"阴阳调和": 0.10,
		"龙凤呈祥": 0.15,
		"天地交泰": 0.20,
		"五行合气": 0.12,
		"心意相通": 0.08,
	}
	if bonus, ok := bonuses[technique]; ok {
		return bonus
	}
	return 0
}

// ============================================================
// 3. 道侣技能
// ============================================================

// UseSkill 使用道侣技能
func (s *DaoLvService) UseSkill(ctx context.Context, playerID uint64, skillName string) (interface{}, error) {
	rel, err := s.repo.FindRelationByPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("未找到道侣关系")
	}
	if rel.Status != "normal" {
		return nil, fmt.Errorf("道侣关系已解除")
	}

	// 检查技能是否已解锁
	if !s.hasSkill(rel, skillName) {
		return nil, fmt.Errorf("技能「%s」尚未解锁", skillName)
	}

	partnerID := s.repo.GetPartnerID(rel, playerID)

	switch skillName {
	case "传送":
		return s.useTeleport(ctx, playerID, partnerID, rel)
	case "复活":
		return s.useRevive(ctx, playerID, partnerID, rel)
	case "属性共享":
		return s.getStatShareBonus(ctx, playerID, partnerID)
	case "心有灵犀":
		return s.checkHpAlert(ctx, partnerID)
	default:
		return nil, fmt.Errorf("未知技能: %s", skillName)
	}
}

// hasSkill 检查是否已解锁技能
func (s *DaoLvService) hasSkill(rel *model.DaolvRelation, skill string) bool {
	for _, sk := range rel.Skills {
		if sk == skill {
			return true
		}
	}
	return false
}

// useTeleport 传送技能: 传送至道侣身边
func (s *DaoLvService) useTeleport(ctx context.Context, playerID, partnerID uint64, rel *model.DaolvRelation) (*TeleportResult, error) {
	// 获取道侣位置
	partnerRegion := s.fetchPlayerRegion(ctx, partnerID)
	partnerName := s.fetchPlayerName(ctx, partnerID)

	// 更新玩家位置到道侣所在区域（直接更新 MongoDB）
	if err := s.updatePlayerRegion(ctx, playerID, partnerRegion); err != nil {
		return nil, fmt.Errorf("传送失败，更新位置出错: %w", err)
	}

	return &TeleportResult{
		Message:     fmt.Sprintf("已传送至 %s 身边", partnerName),
		PartnerName: partnerName,
		Region:      partnerRegion,
	}, nil
}

// TeleportResult 传送结果
type TeleportResult struct {
	Message     string `json:"message"`
	PartnerName string `json:"partner_name"`
	Region      string `json:"region"`
}

// useRevive 复活技能: 复活道侣
func (s *DaoLvService) useRevive(ctx context.Context, playerID, partnerID uint64, rel *model.DaolvRelation) (*ReviveResult, error) {
	partnerName := s.fetchPlayerName(ctx, partnerID)

	// 检查道侣是否在战斗中死亡
	hp := s.fetchPlayerHP(ctx, partnerID)
	if hp > 0 {
		return nil, fmt.Errorf("%s 当前不需要复活", partnerName)
	}

	// 复活道侣：恢复满血（直接更新 MongoDB）
	maxHP := s.fetchPlayerMaxHP(ctx, partnerID)
	if err := s.revivePlayer(ctx, partnerID, maxHP); err != nil {
		return nil, fmt.Errorf("复活 %s 失败: %w", partnerName, err)
	}

	return &ReviveResult{
		Message:     fmt.Sprintf("已复活 %s", partnerName),
		PartnerName: partnerName,
		HPRestored:  true,
	}, nil
}

// ReviveResult 复活结果
type ReviveResult struct {
	Message     string `json:"message"`
	PartnerName string `json:"partner_name"`
	HPRestored  bool   `json:"hp_restored"`
}

// getStatShareBonus 属性共享: 获取道侣属性加成
type StatShareResult struct {
	BonusAttack  int64  `json:"bonus_attack"`
	BonusDefense int64  `json:"bonus_defense"`
	BonusMaxHP   int64  `json:"bonus_max_hp"`
	PartnerName  string `json:"partner_name"`
}

func (s *DaoLvService) getStatShareBonus(ctx context.Context, playerID, partnerID uint64) (*StatShareResult, error) {
	partner := s.fetchPlayerStats(ctx, partnerID)
	partnerName := s.fetchPlayerName(ctx, partnerID)

	return &StatShareResult{
		BonusAttack:  int64(float64(partner.Attack) * StatSharePercent),
		BonusDefense: int64(float64(partner.Defense) * StatSharePercent),
		BonusMaxHP:   int64(float64(partner.MaxHP) * StatSharePercent),
		PartnerName:  partnerName,
	}, nil
}

// checkHpAlert 心有灵犀: 检查道侣血量警报
type HpAlertResult struct {
	Alert        bool   `json:"alert"`
	PartnerHP    int64  `json:"partner_hp"`
	PartnerMaxHP int64  `json:"partner_max_hp"`
	PartnerName  string `json:"partner_name"`
	Message      string `json:"message"`
}

func (s *DaoLvService) checkHpAlert(ctx context.Context, partnerID uint64) (*HpAlertResult, error) {
	hp := s.fetchPlayerHP(ctx, partnerID)
	maxHP := s.fetchPlayerMaxHP(ctx, partnerID)
	partnerName := s.fetchPlayerName(ctx, partnerID)

	hpPercent := float64(hp) / float64(maxHP)
	alert := hpPercent < 0.20

	message := ""
	if alert {
		message = fmt.Sprintf("你的道侣 %s 生命值低于20%%，请速去救援！", partnerName)
	}

	return &HpAlertResult{
		Alert:        alert,
		PartnerHP:    hp,
		PartnerMaxHP: maxHP,
		PartnerName:  partnerName,
		Message:      message,
	}, nil
}

// ============================================================
// 4. 道侣等级
// ============================================================

// checkAndUpdateLevel 检查并更新道侣等级
func (s *DaoLvService) checkAndUpdateLevel(ctx context.Context, relationID string) {
	rel, err := s.repo.FindRelationByID(ctx, relationID)
	if err != nil {
		return
	}

	newLevel := model.GetDaolvLevel(rel.Intimacy)
	if newLevel != rel.Level {
		// 更新等级
		_ = s.repo.SetRelationField(ctx, relationID, "level", newLevel)

		// 检查是否有新技能解锁
		newSkills := s.getSkillsForLevel(rel, newLevel)
		if len(newSkills) > len(rel.Skills) {
			_ = s.repo.SetRelationField(ctx, relationID, "skills", newSkills)
		}
	}
}

// getSkillsForLevel 获取指定等级的所有可用技能
func (s *DaoLvService) getSkillsForLevel(rel *model.DaolvRelation, level string) []string {
	skills := make([]string, 0)
	// 保留已有技能
	skills = append(skills, rel.Skills...)

	// 根据等级解锁新技能
	levels := model.GetDaolvLevels()
	for _, l := range levels {
		if l.UnlockSkill != "" {
			switch level {
			case model.DaolvLevelZhiJi:
				if l.Name == model.DaolvLevelZhiJi && l.UnlockSkill == "传送" {
					skills = append(skills, "传送")
				}
			case model.DaolvLevelQingShen:
				if l.UnlockSkill == "传送" || l.UnlockSkill == "属性共享" {
					skills = appendIfMissing(skills, l.UnlockSkill)
				}
			case model.DaolvLevelTongXin:
				if l.UnlockSkill == "传送" || l.UnlockSkill == "属性共享" || l.UnlockSkill == "复活" {
					skills = appendIfMissing(skills, l.UnlockSkill)
				}
			case model.DaolvLevelXianLv:
				skills = appendIfMissing(skills, l.UnlockSkill)
			}
		}
	}

	// 去重
	return uniqueStrings(skills)
}

func appendIfMissing(slice []string, s string) []string {
	for _, ele := range slice {
		if ele == s {
			return slice
		}
	}
	return append(slice, s)
}

func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// ============================================================
// 5. 道侣任务
// ============================================================

// GetTasks 获取道侣任务列表
func (s *DaoLvService) GetTasks(ctx context.Context, playerID uint64) ([]*model.DaolvTask, error) {
	rel, err := s.repo.FindRelationByPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("未找到道侣关系，请先结为道侣")
	}

	// 尝试获取现有任务
	tasks, err := s.repo.FindTasksByRelation(ctx, rel.ID, "")
	if err != nil || len(tasks) == 0 {
		// 没有任务，生成
		if err := s.refreshDailyTasks(ctx, rel.ID); err != nil {
			return nil, fmt.Errorf("生成任务失败: %w", err)
		}
		tasks, _ = s.repo.FindTasksByRelation(ctx, rel.ID, "")
	}

	return tasks, nil
}

// refreshDailyTasks 刷新每日任务
func (s *DaoLvService) refreshDailyTasks(ctx context.Context, relationID string) error {
	// 删除旧任务
	_ = s.repo.DeleteTasksByRelation(ctx, relationID)

	today := time.Now().Format("2006-01-02")
	weekday := time.Now().Weekday()

	tasks := make([]*model.DaolvTask, 0)

	// 每日任务
	dailyTasks := []struct {
		taskType model.DaolvTaskType
		target   int64
		desc     string
		intimacy int64
	}{
		{model.TaskDualCultivate, 1800, "与道侣双修30分钟", 15},
		{model.TaskSendGift, 1, "赠送道侣一份礼物", 10},
		{model.TaskAdventure, 1, "与道侣共同冒险一次", 20},
	}

	for _, dt := range dailyTasks {
		tasks = append(tasks, &model.DaolvTask{
			ID:          uuid.New().String(),
			RelationID:  relationID,
			Type:        dt.taskType,
			Description: dt.desc,
			Target:      dt.target,
			Progress:    0,
			Completed:   false,
			Claimed:     false,
			Period:      "daily",
			Date:        today,
			Reward: &model.DaolvReward{
				Intimacy: dt.intimacy,
				Items: []model.ItemReward{
					{ItemID: "gift_box", ItemName: "同心结", Quantity: 1},
				},
			},
		})
	}

	// 每周任务(仅周一刷新)
	if weekday == time.Monday {
		weeklyTasks := []struct {
			taskType model.DaolvTaskType
			target   int64
			desc     string
			intimacy int64
		}{
			{model.TaskBoss, 1, "与道侣共同击败世界BOSS", 50},
			{model.TaskDungeon, 3, "与道侣共同完成秘境副本(3次)", 40},
		}

		for _, wt := range weeklyTasks {
			tasks = append(tasks, &model.DaolvTask{
				ID:          uuid.New().String(),
				RelationID:  relationID,
				Type:        wt.taskType,
				Description: wt.desc,
				Target:      wt.target,
				Progress:    0,
				Completed:   false,
				Claimed:     false,
				Period:      "weekly",
				Date:        today,
				Reward: &model.DaolvReward{
					Intimacy: wt.intimacy,
					Items: []model.ItemReward{
						{ItemID: "spirit_stone", ItemName: "灵石", Quantity: 500},
						{ItemID: "intimacy_token", ItemName: "姻缘令", Quantity: 1},
					},
				},
			})
		}
	}

	if len(tasks) > 0 {
		return s.repo.InsertTasks(ctx, tasks)
	}
	return nil
}

// ClaimTask 领取任务奖励
func (s *DaoLvService) ClaimTask(ctx context.Context, taskID, relationID string) (*model.DaolvReward, error) {
	task, err := s.repo.FindTaskByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("任务不存在")
	}

	if task.RelationID != relationID {
		return nil, fmt.Errorf("非本人的任务")
	}

	if !task.Completed {
		return nil, fmt.Errorf("任务尚未完成")
	}

	if task.Claimed {
		return nil, fmt.Errorf("奖励已领取")
	}

	// 标记已领取
	if err := s.repo.UpdateTask(ctx, taskID, bson.M{"claimed": true}); err != nil {
		return nil, fmt.Errorf("领取失败: %w", err)
	}

	// 发放亲密度奖励
	if task.Reward != nil && task.Reward.Intimacy > 0 {
		_ = s.repo.UpdateRelation(ctx, relationID, bson.M{
			"$inc": bson.M{"intimacy": int(task.Reward.Intimacy)},
		})
	}

	// 检查等级更新
	s.checkAndUpdateLevel(ctx, relationID)

	return task.Reward, nil
}

// UpdateTaskProgress 更新任务进度(供外部调用)
func (s *DaoLvService) UpdateTaskProgress(ctx context.Context, relationID string, taskType model.DaolvTaskType, progressIncrement int64) error {
	tasks, err := s.repo.FindTasksByRelation(ctx, relationID, "")
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.Type == taskType && !task.Completed && !task.Claimed {
			newProgress := task.Progress + progressIncrement
			if newProgress > task.Target {
				newProgress = task.Target
			}
			completed := newProgress >= task.Target
			if err := s.repo.UpdateTaskProgress(ctx, task.ID, newProgress, completed); err != nil {
				return err
			}
		}
	}
	return nil
}

// ============================================================
// 6. 解除道侣关系
// ============================================================

// Dissolve 解除道侣关系
func (s *DaoLvService) Dissolve(ctx context.Context, relationID string, initiatorID uint64) error {
	rel, err := s.repo.FindRelationByID(ctx, relationID)
	if err != nil {
		return fmt.Errorf("道侣关系不存在")
	}
	if rel.Status != "normal" {
		return fmt.Errorf("道侣关系已解除")
	}

	// 标记解除
	if err := s.repo.UpdateRelation(ctx, relationID, bson.M{
		"$set": bson.M{
			"status":       "divorced",
			"dissolved_at": time.Now(),
		},
	}); err != nil {
		return fmt.Errorf("解除道侣关系失败: %w", err)
	}

	// 清理任务
	_ = s.repo.DeleteTasksByRelation(ctx, relationID)

	return nil
}

// ============================================================
// 7. 道侣状态查询
// ============================================================

// DaolvStatus 道侣状态响应
type DaolvStatus struct {
	HasPartner      bool           `json:"has_partner"`
	RelationID      string         `json:"relation_id,omitempty"`
	PartnerID       uint64         `json:"partner_id,omitempty"`
	PartnerName     string         `json:"partner_name,omitempty"`
	PartnerRealm    string         `json:"partner_realm,omitempty"`
	Level           string         `json:"level,omitempty"`
	Intimacy        int            `json:"intimacy,omitempty"`
	Compatibility   float64        `json:"compatibility,omitempty"`
	Skills          []string       `json:"skills,omitempty"`
	DailyCultivated int64          `json:"daily_cultivated,omitempty"`
	DailyLimit      int64          `json:"daily_limit,omitempty"`
	DurationDays    int            `json:"duration_days,omitempty"`
	NextLevel       *NextLevelInfo `json:"next_level,omitempty"`
	GiftItemID      string         `json:"gift_item_id,omitempty"`
}

// NextLevelInfo 下一级信息
type NextLevelInfo struct {
	LevelName      string `json:"level_name"`
	IntimacyNeeded int64  `json:"intimacy_needed"`
	UnlockSkill    string `json:"unlock_skill,omitempty"`
}

// GetStatus 获取玩家道侣状态
func (s *DaoLvService) GetStatus(ctx context.Context, playerID uint64) (*DaolvStatus, error) {
	rel, err := s.repo.FindRelationByPlayer(ctx, playerID)
	if err != nil {
		return &DaolvStatus{HasPartner: false}, nil
	}

	partnerID := s.repo.GetPartnerID(rel, playerID)
	partnerName := s.fetchPlayerName(ctx, partnerID)
	partnerRealm := s.fetchPlayerRealmName(ctx, partnerID)

	durationDays := int(time.Since(rel.StartedAt).Hours() / 24)

	// 计算下一级信息
	var nextLevel *NextLevelInfo
	levels := model.GetDaolvLevels()
	currentIdx := -1
	for i, l := range levels {
		if l.Name == rel.Level {
			currentIdx = i
			break
		}
	}
	if currentIdx >= 0 && currentIdx < len(levels)-1 {
		nextLevelInfo := levels[currentIdx+1]
		needed := nextLevelInfo.MinIntimacy - int64(rel.Intimacy)
		if needed < 0 {
			needed = 0
		}
		nextLevel = &NextLevelInfo{
			LevelName:      nextLevelInfo.Name,
			IntimacyNeeded: needed,
			UnlockSkill:    nextLevelInfo.UnlockSkill,
		}
	}

	// 确定定情信物
	giftItemID := rel.GiftItemA
	if rel.PlayerB == playerID {
		giftItemID = rel.GiftItemB
	}

	status := &DaolvStatus{
		HasPartner:      true,
		RelationID:      rel.ID,
		PartnerID:       partnerID,
		PartnerName:     partnerName,
		PartnerRealm:    partnerRealm,
		Level:           rel.Level,
		Intimacy:        rel.Intimacy,
		Compatibility:   rel.Compatibility,
		Skills:          rel.Skills,
		DailyCultivated: rel.DailyCultivated,
		DailyLimit:      int64(DailyCultivateLimit.Seconds()),
		DurationDays:    durationDays,
		NextLevel:       nextLevel,
		GiftItemID:      giftItemID,
	}

	return status, nil
}

// ============================================================
// 契合度计算
// ============================================================

// GetCompatibility 计算两名玩家契合度
// 公式: 50% + (灵根相似度 x 20%) + (境界差补正 x 30%), 最高 100%
func (s *DaoLvService) GetCompatibility(ctx context.Context, a, b uint64) float64 {
	// 基准值 50%
	compatibility := 0.50

	// 灵根相似度
	rootSimilarity := s.calcRootSimilarity(ctx, a, b)
	compatibility += rootSimilarity * 0.20

	// 境界差补正
	stageBonus := s.calcStageBonus(ctx, a, b)
	compatibility += stageBonus * 0.30

	// 随机波动 ±5%
	fluctuation := (rand.Float64() - 0.5) * 0.10
	compatibility += fluctuation

	// 范围限制
	if compatibility > 1.0 {
		compatibility = 1.0
	}
	if compatibility < 0.0 {
		compatibility = 0.0
	}
	return math.Round(compatibility*100) / 100
}

// calcRootSimilarity 计算灵根相似度 (0~1)
func (s *DaoLvService) calcRootSimilarity(ctx context.Context, a, b uint64) float64 {
	rootsA := s.fetchSpiritualRoots(ctx, a)
	rootsB := s.fetchSpiritualRoots(ctx, b)

	if len(rootsA) == 0 || len(rootsB) == 0 {
		return 0
	}

	// Jaccard 相似度
	intersection := 0
	for r := range rootsA {
		if rootsB[r] {
			intersection++
		}
	}
	union := len(rootsA)
	if len(rootsB) > union {
		union = len(rootsB)
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// calcStageBonus 计算境界差补正 (0~1)
func (s *DaoLvService) calcStageBonus(ctx context.Context, a, b uint64) float64 {
	stageA := s.fetchCultivationStage(ctx, a)
	stageB := s.fetchCultivationStage(ctx, b)

	diff := math.Abs(float64(stageA - stageB))
	bonus := 1.0 - diff/6.0
	if bonus < 0 {
		bonus = 0
	}
	return bonus
}

// ============================================================
// 跨服务数据获取
// ============================================================

// PlayerStats 玩家属性
type PlayerStats struct {
	Attack  int64
	Defense int64
	MaxHP   int64
}

func (s *DaoLvService) fetchPlayerStats(ctx context.Context, playerID uint64) *PlayerStats {
	coll := s.db.Collection("players")
	var result struct {
		Attack  int64 `bson:"attack"`
		Defense int64 `bson:"defense"`
		MaxHP   int64 `bson:"max_hp"`
	}
	err := coll.FindOne(ctx, bson.M{"_id": playerID}).Decode(&result)
	if err != nil {
		return &PlayerStats{Attack: 10, Defense: 5, MaxHP: 100}
	}
	return &PlayerStats{
		Attack:  result.Attack,
		Defense: result.Defense,
		MaxHP:   result.MaxHP,
	}
}

func (s *DaoLvService) fetchPlayerHP(ctx context.Context, playerID uint64) int64 {
	coll := s.db.Collection("players")
	var result struct {
		HP int64 `bson:"hp"`
	}
	err := coll.FindOne(ctx, bson.M{"_id": playerID}).Decode(&result)
	if err != nil {
		return 100
	}
	return result.HP
}

func (s *DaoLvService) fetchPlayerMaxHP(ctx context.Context, playerID uint64) int64 {
	coll := s.db.Collection("players")
	var result struct {
		MaxHP int64 `bson:"max_hp"`
	}
	err := coll.FindOne(ctx, bson.M{"_id": playerID}).Decode(&result)
	if err != nil {
		return 100
	}
	return result.MaxHP
}

func (s *DaoLvService) fetchPlayerRegion(ctx context.Context, playerID uint64) string {
	coll := s.db.Collection("players")
	var result struct {
		Region string `bson:"region"`
	}
	err := coll.FindOne(ctx, bson.M{"_id": playerID}).Decode(&result)
	if err != nil {
		return "unknown"
	}
	return result.Region
}

// updatePlayerRegion 更新玩家在 MongoDB 中的位置信息
func (s *DaoLvService) updatePlayerRegion(ctx context.Context, playerID uint64, region string) error {
	coll := s.db.Collection("players")
	_, err := coll.UpdateOne(ctx, bson.M{"_id": playerID}, bson.M{
		"$set": bson.M{"region": region},
	})
	if err != nil {
		return fmt.Errorf("更新玩家位置到 MongoDB 失败: %w", err)
	}
	return nil
}

// revivePlayer 复活道侣：将 HP 恢复至满血
func (s *DaoLvService) revivePlayer(ctx context.Context, playerID uint64, maxHP int64) error {
	coll := s.db.Collection("players")
	_, err := coll.UpdateOne(ctx, bson.M{"_id": playerID}, bson.M{
		"$set": bson.M{"hp": maxHP},
	})
	if err != nil {
		return fmt.Errorf("复活玩家写入 MongoDB 失败: %w", err)
	}
	return nil
}

func (s *DaoLvService) fetchPlayerName(ctx context.Context, playerID uint64) string {
	coll := s.db.Collection("players")
	var result struct {
		Name string `bson:"name"`
	}
	err := coll.FindOne(ctx, bson.M{"_id": playerID}).Decode(&result)
	if err != nil {
		return fmt.Sprintf("玩家%d", playerID)
	}
	return result.Name
}

func (s *DaoLvService) fetchPlayerRealmName(ctx context.Context, playerID uint64) string {
	coll := s.db.Collection("players")
	var result struct {
		Realm int32 `bson:"realm"`
	}
	err := coll.FindOne(ctx, bson.M{"_id": playerID}).Decode(&result)
	if err != nil {
		return "未知"
	}
	realmNames := map[int32]string{
		1: "凡人", 2: "练气", 3: "筑基", 4: "金丹",
		5: "元婴", 6: "化神", 7: "合体", 8: "大乘", 9: "渡劫",
	}
	if name, ok := realmNames[result.Realm]; ok {
		return name
	}
	return "未知"
}

func (s *DaoLvService) fetchSpiritualRoots(ctx context.Context, playerID uint64) map[string]bool {
	coll := s.db.Collection("players")
	var result struct {
		SpiritualRoots []string `bson:"spiritual_roots"`
	}
	err := coll.FindOne(ctx, bson.M{"_id": playerID}).Decode(&result)
	if err != nil || len(result.SpiritualRoots) == 0 {
		n := rand.Intn(5) + 1
		roots := make(map[string]bool, n)
		defaultRoots := []string{"金", "木", "水", "火", "土"}
		for i := 0; i < n && i < len(defaultRoots); i++ {
			roots[defaultRoots[i]] = true
		}
		return roots
	}
	roots := make(map[string]bool, len(result.SpiritualRoots))
	for _, r := range result.SpiritualRoots {
		roots[r] = true
	}
	return roots
}

func (s *DaoLvService) fetchCultivationStage(ctx context.Context, playerID uint64) int {
	coll := s.db.Collection("players")
	var result struct {
		Stage int `bson:"cultivation_stage"`
	}
	err := coll.FindOne(ctx, bson.M{"_id": playerID}).Decode(&result)
	if err != nil {
		return 4 // 默认金丹
	}
	return result.Stage
}

// ============================================================
// 兼容方法(保持旧接口)
// ============================================================

// Propose (旧接口, 适配老调用方)
func (s *DaoLvService) ProposeOld(ctx context.Context, fromID, toID uint64, gift []model.Item) error {
	return s.Propose(ctx, &ProposeRequest{
		FromID: fromID,
		ToID:   toID,
	})
}

// GetRelation 获取玩家的道侣关系信息
func (s *DaoLvService) GetRelation(ctx context.Context, playerID uint64) (*model.DaolvRelation, error) {
	return s.repo.FindRelationByPlayer(ctx, playerID)
}

// GetProposals 获取玩家相关的所有求婚申请
func (s *DaoLvService) GetProposals(ctx context.Context, playerID uint64) ([]*model.DaolvProposal, error) {
	return s.repo.FindProposalsByPlayer(ctx, playerID)
}

// GetPendingProposals 获取玩家收到的待处理申请
func (s *DaoLvService) GetPendingProposals(ctx context.Context, playerID uint64) ([]*model.DaolvProposal, error) {
	return s.repo.FindPendingProposalsByTarget(ctx, playerID)
}

// SendGift 赠送礼物: fromID 送给 toID 物品, 增加亲密度
func (s *DaoLvService) SendGift(ctx context.Context, fromID, toID uint64, itemID uint64, qty int) error {
	rel, err := s.repo.FindRelationByPlayer(ctx, fromID)
	if err != nil {
		return fmt.Errorf("未找到道侣关系, 请先结为道侣: %w", err)
	}
	if (rel.PlayerA != fromID || rel.PlayerB != toID) && (rel.PlayerA != toID || rel.PlayerB != fromID) {
		return fmt.Errorf("对方不是你的道侣")
	}

	intimacyGain := qty * 2
	if intimacyGain < 1 {
		intimacyGain = 1
	}
	if err := s.repo.UpdateRelation(ctx, rel.ID, bson.M{
		"$inc": bson.M{"intimacy": intimacyGain},
	}); err != nil {
		return fmt.Errorf("更新亲密度失败: %w", err)
	}

	// 更新任务进度
	go func() {
		_ = s.UpdateTaskProgress(context.Background(), rel.ID, model.TaskSendGift, 1)
	}()

	return nil
}

// Teleport 传送到道侣身边（旧接口保持兼容）
func (s *DaoLvService) Teleport(ctx context.Context, playerID uint64) (*model.DaolvRelation, error) {
	rel, err := s.repo.FindRelationByPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("未找到道侣关系, 请先结为道侣: %w", err)
	}
	return rel, nil
}

// DualCultivate 双修(旧接口保持兼容, 默认30分钟)
func (s *DaoLvService) DualCultivate(ctx context.Context, daoLvID string, duration time.Duration) (int64, error) {
	rel, err := s.repo.FindRelationByID(ctx, daoLvID)
	if err != nil {
		return 0, fmt.Errorf("道侣关系不存在: %w", err)
	}
	if rel.Status != "normal" {
		return 0, fmt.Errorf("道侣关系已解除")
	}

	playerID := rel.PlayerA
	result, err := s.StartDualCultivate(ctx, &DualCultivateRequest{
		PlayerID: playerID,
		Duration: duration,
	})
	if err != nil {
		return 0, err
	}
	return result.CultivationGained, nil
}

// Divorce 解除道侣关系(旧接口)
func (s *DaoLvService) Divorce(ctx context.Context, id string) error {
	return s.Dissolve(ctx, id, 0)
}
