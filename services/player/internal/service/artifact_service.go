package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// ArtifactService 法宝业务逻辑
type ArtifactService struct {
	artifactRepo  *mysql.ArtifactRepo
	playerRepo    *mysql.PlayerRepo
	inventoryRepo *mysql.InventoryRepo
	log           *zap.Logger
}

// NewArtifactService 创建 ArtifactService
func NewArtifactService(artifactRepo *mysql.ArtifactRepo, playerRepo *mysql.PlayerRepo, inventoryRepo *mysql.InventoryRepo, log *zap.Logger) *ArtifactService {
	return &ArtifactService{
		artifactRepo:  artifactRepo,
		playerRepo:    playerRepo,
		inventoryRepo: inventoryRepo,
		log:           log,
	}
}

// ============================================================
// 1. 基础操作
// ============================================================

// BindArtifact 绑定本命法宝
// 条件：玩家达到金丹期(21层+)，且未绑定过法宝
func (s *ArtifactService) BindArtifact(ctx context.Context, playerID int64, name string, artType int) (*model.Artifact, error) {
	// 验证类型
	if artType < 1 || artType > 6 {
		return nil, fmt.Errorf("无效的法宝类型（1-6）")
	}

	// 1. 检查玩家是否存在
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}

	// 2. 检查境界——金丹期要求等级 >= 21
	if player.Level < 21 {
		return nil, fmt.Errorf("金丹期(21层)以上才能绑定本命法宝，当前等级 %d", player.Level)
	}

	// 3. 检查是否已绑定同类型法宝
	existing, err := s.artifactRepo.GetMultipleByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询法宝失败: %w", err)
	}
	// 最多6件（每种类型1件）
	if len(existing) >= 6 {
		return nil, fmt.Errorf("法宝已达上限（最多6件）")
	}
	for _, e := range existing {
		if e.Type == artType {
			return nil, fmt.Errorf("已拥有%s类型法宝，不可重复", model.ArtifactTypeNames[artType])
		}
	}

	// 4. 计算初始属性
	artifact := &model.Artifact{
		PlayerID: playerID,
		Name:     name,
		Type:     artType,
		Quality:  model.ArtifactQualityMortal,
		Level:    1,
	}
	s.recalcBonuses(artifact)

	if err := s.artifactRepo.Create(artifact); err != nil {
		return nil, err
	}

	s.log.Info("绑定法宝成功",
		zap.Int64("player_id", playerID),
		zap.String("name", name),
		zap.Int("type", artType),
	)
	return artifact, nil
}

// ListArtifacts 获取玩家所有法宝列表
func (s *ArtifactService) ListArtifacts(ctx context.Context, playerID int64) (*model.ArtifactListResponse, error) {
	artifacts, err := s.artifactRepo.GetMultipleByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询法宝列表失败: %w", err)
	}
	if artifacts == nil {
		artifacts = []*model.Artifact{}
	}

	resp := &model.ArtifactListResponse{
		Artifacts: artifacts,
	}
	if len(artifacts) > 0 {
		resp.MainSlotID = artifacts[0].ID
	}

	// 计算共鸣
	resp.Resonance = s.calcResonance(artifacts)

	return resp, nil
}

// UpgradeArtifact 升级法宝（消耗灵石）
func (s *ArtifactService) UpgradeArtifact(ctx context.Context, playerID int64, artifactID int64) (*model.Artifact, error) {
	// 1. 获取法宝
	artifact, err := s.artifactRepo.GetByID(artifactID)
	if err != nil {
		return nil, fmt.Errorf("查询法宝失败: %w", err)
	}
	if artifact == nil {
		return nil, fmt.Errorf("法宝不存在")
	}
	if artifact.PlayerID != playerID {
		return nil, fmt.Errorf("法宝不属于该玩家")
	}

	// 2. 检查等级上限
	if artifact.Level >= 100 {
		return nil, fmt.Errorf("法宝已达最高等级(100级)")
	}

	// 3. 计算消耗并扣除灵石
	cost := int64(artifact.Level * 200)
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}
	if player.Gold < cost {
		return nil, fmt.Errorf("灵石不足，升级需要 %d 灵石，当前 %d", cost, player.Gold)
	}
	player.Gold -= cost

	// 4. 升级
	artifact.Level++
	s.recalcBonuses(artifact)

	// 5. 写库
	if err := s.playerRepo.Update(player); err != nil {
		return nil, fmt.Errorf("扣除灵石失败: %w", err)
	}
	if err := s.artifactRepo.Update(artifact); err != nil {
		return nil, err
	}

	s.log.Info("法宝升级成功",
		zap.Int64("player_id", playerID),
		zap.Int64("artifact_id", artifactID),
		zap.Int("level", artifact.Level),
	)
	return artifact, nil
}

// ============================================================
// 2. 进化系统 (Evolution)
// 消耗材料 + 灵石，每次进化提升1级，到达关键等级有概率升品
// ============================================================

// EvolveArtifact 进化法宝（消耗材料升级）
func (s *ArtifactService) EvolveArtifact(ctx context.Context, playerID int64, artifactID int64) (*model.ArtifactEvolveResult, error) {
	artifact, err := s.artifactRepo.GetByID(artifactID)
	if err != nil {
		return nil, fmt.Errorf("查询法宝失败: %w", err)
	}
	if artifact == nil || artifact.PlayerID != playerID {
		return nil, fmt.Errorf("法宝不存在")
	}
	if artifact.Level >= 100 {
		return nil, fmt.Errorf("法宝已达最高等级(100级)")
	}

	// 获取当前等级段的进化消耗
	cost := model.GetEvolveCost(artifact.Level)

	// 检查灵石
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}
	if player.Gold < cost.Gold {
		return nil, fmt.Errorf("灵石不足，进化需要 %d 灵石，当前 %d", cost.Gold, player.Gold)
	}

	// 检查材料
	invItem, err := s.inventoryRepo.FindStackableItem(playerID, cost.MaterialID)
	if err != nil || invItem == nil || invItem.Quantity < cost.MaterialQty {
		return nil, fmt.Errorf("%s不足，需要 %d 个", cost.MaterialName, cost.MaterialQty)
	}

	// 扣除灵石
	player.Gold -= cost.Gold
	if err := s.playerRepo.Update(player); err != nil {
		return nil, fmt.Errorf("扣除灵石失败: %w", err)
	}

	// 扣除材料
	newQty := invItem.Quantity - cost.MaterialQty
	if newQty <= 0 {
		if err := s.inventoryRepo.DeleteItem(invItem.ID); err != nil {
			return nil, fmt.Errorf("扣除材料失败: %w", err)
		}
	} else {
		if err := s.inventoryRepo.UpdateItemQuantity(invItem.ID, newQty); err != nil {
			return nil, fmt.Errorf("扣除材料失败: %w", err)
		}
	}

	// 升级
	artifact.Level++
	levelUp := false

	// 每10级触发一次品质升品判定
	if artifact.Level%10 == 0 && artifact.Quality < model.ArtifactQualityChaos {
		rate, ok := model.ArtifactQualityUpgradeRates[artifact.Quality]
		if !ok {
			rate = 0.5
		}
		if rand.Float64() < rate {
			artifact.Quality++
			s.unlockSkill(artifact)
			levelUp = true
		}
	}

	// 检查觉醒里程碑
	s.checkAwakenMilestones(artifact)

	s.recalcBonuses(artifact)
	if err := s.artifactRepo.Update(artifact); err != nil {
		return nil, err
	}

	// 器灵好感增加
	s.addSpiritBondXP(ctx, artifactID, 10)

	result := &model.ArtifactEvolveResult{
		Artifact: artifact,
		Success:  true,
		LevelUp:  levelUp,
		Msg:      fmt.Sprintf("进化成功！法宝等级 %d", artifact.Level),
	}
	if levelUp {
		result.Msg = fmt.Sprintf("进化成功！品质提升至%s！", model.ArtifactQualityNames[artifact.Quality])
	}

	s.log.Info("法宝进化成功",
		zap.Int64("player_id", playerID),
		zap.Int64("artifact_id", artifactID),
		zap.Int("new_level", artifact.Level),
		zap.Bool("quality_up", levelUp),
	)
	return result, nil
}

// ============================================================
// 3. 觉醒系统 (Awakening)
// 在等级20/40/60/80/100解锁特殊技能
// ============================================================

// GetAwakenInfo 获取觉醒信息
func (s *ArtifactService) GetAwakenInfo(ctx context.Context, playerID int64, artifactID int64) ([]model.ArtifactAwakenMilestone, []int, error) {
	artifact, err := s.artifactRepo.GetByID(artifactID)
	if err != nil {
		return nil, nil, fmt.Errorf("查询法宝失败: %w", err)
	}
	if artifact == nil || artifact.PlayerID != playerID {
		return nil, nil, fmt.Errorf("法宝不存在")
	}

	milestones, ok := model.AwakenMilestonesByType[artifact.Type]
	if !ok {
		return nil, nil, fmt.Errorf("未知法宝类型")
	}

	return milestones, artifact.AwakenSkills, nil
}

// AwakenArtifact 强制觉醒（消耗潜力点）
func (s *ArtifactService) AwakenArtifact(ctx context.Context, playerID int64, artifactID int64, slotIndex int) (*model.ArtifactAwakenResult, error) {
	artifact, err := s.artifactRepo.GetByID(artifactID)
	if err != nil {
		return nil, fmt.Errorf("查询法宝失败: %w", err)
	}
	if artifact == nil || artifact.PlayerID != playerID {
		return nil, fmt.Errorf("法宝不存在")
	}

	milestones, ok := model.AwakenMilestonesByType[artifact.Type]
	if !ok {
		return nil, fmt.Errorf("未知法宝类型")
	}

	// 找到对应槽位的里程碑
	var targetMilestone *model.ArtifactAwakenMilestone
	for _, m := range milestones {
		if m.SlotIndex == slotIndex {
			targetMilestone = &m
			break
		}
	}
	if targetMilestone == nil {
		return nil, fmt.Errorf("无效的觉醒槽位 %d", slotIndex)
	}

	// 检查等级是否达标
	if artifact.Level < targetMilestone.Level {
		return nil, fmt.Errorf("等级不足，需要%d级才能觉醒该技能（当前%d级）", targetMilestone.Level, artifact.Level)
	}

	// 检查是否已解锁
	for _, sid := range artifact.AwakenSkills {
		if sid == targetMilestone.SkillID {
			return nil, fmt.Errorf("该技能已觉醒")
		}
	}

	// 检查潜力点
	if artifact.Potential < 1 {
		return nil, fmt.Errorf("潜力点不足，需要1点潜力")
	}
	artifact.Potential--

	// 解锁技能
	artifact.AwakenSkills = append(artifact.AwakenSkills, targetMilestone.SkillID)

	if err := s.artifactRepo.Update(artifact); err != nil {
		return nil, err
	}

	// 器灵好感增加
	s.addSpiritBondXP(ctx, artifactID, 20)

	skillName := model.ArtifactAwakenSkillNames[targetMilestone.SkillID]
	if skillName == "" {
		skillName = model.ArtifactSkillNames[targetMilestone.SkillID]
	}

	s.log.Info("法宝觉醒成功",
		zap.Int64("player_id", playerID),
		zap.Int64("artifact_id", artifactID),
		zap.Int("skill_id", targetMilestone.SkillID),
		zap.String("skill_name", skillName),
	)

	return &model.ArtifactAwakenResult{
		Artifact:      artifact,
		UnlockedSkill: targetMilestone.SkillID,
		SkillName:     skillName,
		SlotIndex:     slotIndex,
	}, nil
}

// checkAwakenMilestones 自动检查并解锁觉醒里程碑（当等级达到时）
func (s *ArtifactService) checkAwakenMilestones(artifact *model.Artifact) {
	milestones, ok := model.AwakenMilestonesByType[artifact.Type]
	if !ok {
		return
	}

	for _, m := range milestones {
		if artifact.Level >= m.Level {
			// 检查是否已解锁
			already := false
			for _, sid := range artifact.AwakenSkills {
				if sid == m.SkillID {
					already = true
					break
				}
			}
			if !already {
				artifact.AwakenSkills = append(artifact.AwakenSkills, m.SkillID)
				s.log.Info("法宝自动觉醒技能",
					zap.Int64("artifact_id", artifact.ID),
					zap.Int("skill_id", m.SkillID),
					zap.Int("level", artifact.Level),
				)
			}
		}
	}
}

// ============================================================
// 4. 器灵系统 (Artifact Spirit)
// ============================================================

// ActivateSpirit 激活器灵
func (s *ArtifactService) ActivateSpirit(ctx context.Context, playerID int64, artifactID int64, spiritName string) (*model.ArtifactSpirit, error) {
	artifact, err := s.artifactRepo.GetByID(artifactID)
	if err != nil {
		return nil, fmt.Errorf("查询法宝失败: %w", err)
	}
	if artifact == nil || artifact.PlayerID != playerID {
		return nil, fmt.Errorf("法宝不存在")
	}
	if artifact.SpiritID != 0 {
		// 已有器灵，检查是否已激活
		existing, _ := s.artifactRepo.GetSpiritByArtifactID(artifactID)
		if existing != nil {
			return nil, fmt.Errorf("该法宝已激活器灵[%s]", existing.Name)
		}
	}

	// 品质要求至少灵品
	if artifact.Quality < model.ArtifactQualitySpirit {
		return nil, fmt.Errorf("灵品以上法宝才能激活器灵")
	}

	// 随机性格
	personality := rand.Intn(6) + 1

	spirit := &model.ArtifactSpirit{
		ArtifactID:  artifactID,
		PlayerID:    playerID,
		Name:        spiritName,
		Personality: personality,
		BondLevel:   1,
		BondExp:     0,
	}

	if err := s.artifactRepo.CreateSpirit(spirit); err != nil {
		return nil, err
	}

	// 关联到法宝
	artifact.SpiritID = spirit.ID
	if err := s.artifactRepo.Update(artifact); err != nil {
		return nil, err
	}

	// 初始对话
	dialogue := s.getSpiritDialogue(personality, "bind")
	spirit.LastDialogue = dialogue
	spirit.LastEvent = "bind"
	s.artifactRepo.UpdateSpirit(spirit)

	s.log.Info("器灵激活成功",
		zap.Int64("player_id", playerID),
		zap.Int64("artifact_id", artifactID),
		zap.String("spirit_name", spiritName),
		zap.Int("personality", personality),
	)

	return spirit, nil
}

// GetSpirit 获取器灵信息
func (s *ArtifactService) GetSpirit(ctx context.Context, playerID int64, artifactID int64) (*model.ArtifactSpirit, error) {
	if artifactID > 0 {
		return s.artifactRepo.GetSpiritByArtifactID(artifactID)
	}
	return s.artifactRepo.GetSpiritByPlayerID(playerID)
}

// InteractSpirit 与器灵互动（增加好感度）
func (s *ArtifactService) InteractSpirit(ctx context.Context, playerID int64, spiritID int64) (*model.ArtifactSpirit, string, error) {
	spirit, err := s.artifactRepo.GetSpiritByArtifactID(spiritID)
	if err != nil {
		return nil, "", fmt.Errorf("查询器灵失败: %w", err)
	}
	if spirit == nil || spirit.PlayerID != playerID {
		return nil, "", fmt.Errorf("器灵不存在")
	}

	// 好感度增加（每日有上限）
	bonus := int64(rand.Intn(20) + 10)
	spirit.BondExp += bonus

	// 升级判定（每100经验升1级）
	for spirit.BondExp >= 100 && spirit.BondLevel < 100 {
		spirit.BondExp -= 100
		spirit.BondLevel++
		spirit.BondUnlocked++
	}

	// 随机对话
	events := []string{"idle", "combat", "idle"}
	event := events[rand.Intn(len(events))]
	dialogue := s.getSpiritDialogue(spirit.Personality, event)
	spirit.LastDialogue = dialogue
	spirit.LastEvent = event

	if err := s.artifactRepo.UpdateSpirit(spirit); err != nil {
		return nil, "", err
	}

	return spirit, dialogue, nil
}

// getSpiritDialogue 根据性格和事件获取对话
func (s *ArtifactService) getSpiritDialogue(personality int, event string) string {
	var candidates []string
	for _, d := range model.DefaultSpiritDialogues {
		if d.Personality == personality && d.Event == event {
			candidates = append(candidates, d.Dialogue)
		}
	}
	if len(candidates) == 0 {
		return "……"
	}
	return candidates[rand.Intn(len(candidates))]
}

// addSpiritBondXP 增加器灵好感经验（内部）
func (s *ArtifactService) addSpiritBondXP(ctx context.Context, artifactID int64, xp int64) {
	spirit, err := s.artifactRepo.GetSpiritByArtifactID(artifactID)
	if err != nil || spirit == nil {
		return
	}

	spirit.BondExp += xp
	for spirit.BondExp >= 100 && spirit.BondLevel < 100 {
		spirit.BondExp -= 100
		spirit.BondLevel++
		spirit.BondUnlocked++
	}
	s.artifactRepo.UpdateSpirit(spirit)
}

// ============================================================
// 5. 共鸣系统 (Resonance)
// ============================================================

// calcResonance 计算玩家法宝共鸣加成
func (s *ArtifactService) calcResonance(artifacts []*model.Artifact) *model.ArtifactResonance {
	if len(artifacts) == 0 {
		return &model.ArtifactResonance{
			OwnedTypes:    []int{},
			ActiveSets:    []int{},
			ActiveBonuses: map[string]int64{},
		}
	}

	// 收集已拥有的类型
	typeSet := make(map[int]bool)
	for _, a := range artifacts {
		typeSet[a.Type] = true
	}
	var ownedTypes []int
	for t := 1; t <= 6; t++ {
		if typeSet[t] {
			ownedTypes = append(ownedTypes, t)
		}
	}

	ownedCount := len(ownedTypes)

	// 计算激活的套装
	var activeSets []int
	totalBonuses := map[string]int64{"attack": 0, "defense": 0, "hp": 0, "mp": 0, "speed": 0, "dodge": 0}

	for _, set := range model.ResonanceSets {
		if ownedCount >= set.Count {
			activeSets = append(activeSets, set.ID)
			for k, v := range set.Bonuses {
				totalBonuses[k] += v
			}
		}
	}

	return &model.ArtifactResonance{
		OwnedTypes:    ownedTypes,
		ActiveSets:    activeSets,
		ActiveBonuses: totalBonuses,
	}
}

// GetResonance 获取共鸣信息
func (s *ArtifactService) GetResonance(ctx context.Context, playerID int64) (*model.ArtifactResonance, error) {
	artifacts, err := s.artifactRepo.GetMultipleByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询法宝列表失败: %w", err)
	}
	return s.calcResonance(artifacts), nil
}

// ============================================================
// 6. 试炼系统 (Trials)
// ============================================================

// GetTrialStages 获取可挑战的试炼关卡
func (s *ArtifactService) GetTrialStages(ctx context.Context, playerID int64, artifactID int64) ([]model.ArtifactTrialStage, *model.ArtifactTrialProgress, error) {
	artifact, err := s.artifactRepo.GetByID(artifactID)
	if err != nil {
		return nil, nil, fmt.Errorf("查询法宝失败: %w", err)
	}
	if artifact == nil || artifact.PlayerID != playerID {
		return nil, nil, fmt.Errorf("法宝不存在")
	}

	// 获取进度
	progress, err := s.artifactRepo.GetTrialProgress(artifactID)
	if err != nil || progress == nil {
		progress = &model.ArtifactTrialProgress{
			PlayerID:        playerID,
			ArtifactID:      artifactID,
			CompletedStages: []int{},
		}
	}

	// 计算可挑战的关卡
	nextStageID := len(progress.CompletedStages) + 1
	var available []model.ArtifactTrialStage
	for _, stage := range model.ArtifactTrials {
		if stage.StageID <= nextStageID && artifact.Level >= stage.MinLevel && artifact.Quality >= stage.MinQuality {
			available = append(available, stage)
		}
	}

	return available, progress, nil
}

// EnterTrial 进入试炼
func (s *ArtifactService) EnterTrial(ctx context.Context, playerID int64, artifactID int64, stageID int) (*model.ArtifactTrialResult, error) {
	artifact, err := s.artifactRepo.GetByID(artifactID)
	if err != nil {
		return nil, fmt.Errorf("查询法宝失败: %w", err)
	}
	if artifact == nil || artifact.PlayerID != playerID {
		return nil, fmt.Errorf("法宝不存在")
	}

	// 查找关卡配置
	var stage *model.ArtifactTrialStage
	for _, st := range model.ArtifactTrials {
		if st.StageID == stageID {
			stage = &st
			break
		}
	}
	if stage == nil {
		return nil, fmt.Errorf("试炼关卡不存在")
	}

	// 检查进度
	progress, err := s.artifactRepo.GetTrialProgress(artifactID)
	if err != nil {
		progress = &model.ArtifactTrialProgress{
			PlayerID:        playerID,
			ArtifactID:      artifactID,
			CompletedStages: []int{},
		}
	}

	// 检查是否已完成
	for _, cs := range progress.CompletedStages {
		if cs == stageID {
			return nil, fmt.Errorf("该试炼关卡已完成")
		}
	}

	// 检查条件
	if artifact.Level < stage.MinLevel {
		return nil, fmt.Errorf("法宝等级不足（需要%d级，当前%d级）", stage.MinLevel, artifact.Level)
	}
	if artifact.Quality < stage.MinQuality {
		return nil, fmt.Errorf("法宝品质不足（需要%s，当前%s）",
			model.ArtifactQualityNames[stage.MinQuality],
			model.ArtifactQualityNames[artifact.Quality])
	}

	// 检查每日次数
	today := time.Now().Format("2006-01-02")
	if progress.LastAttemptDate != today {
		progress.TodayAttempts = 0
		progress.LastAttemptDate = today
	}
	if progress.TodayAttempts >= 3 {
		return nil, fmt.Errorf("今日试炼次数已用完（每日最多3次）")
	}

	// 战斗模拟
	victory := s.simulateTrialBattle(artifact, stage)

	// 更新尝试次数
	progress.TodayAttempts++

	if victory {
		// 记录完成
		progress.CompletedStages = append(progress.CompletedStages, stageID)
		progress.LastCompletedStage = stageID

		// 发放奖励
		rewards := map[string]int64{
			"gold": stage.RewardGold,
			"exp":  stage.RewardExp,
		}

		// 灵石奖励
		player, _ := s.playerRepo.GetByID(playerID)
		if player != nil {
			player.Gold += stage.RewardGold
			s.playerRepo.Update(player)
		}

		// 材料奖励
		if stage.RewardMaterialQty > 0 {
			if _, err := s.inventoryRepo.FindStackableItem(playerID, stage.RewardMaterialID); err != nil {
				// 尝试添加物品 (简化: 直接数据库插入)
				// 实际应使用库存服务
			}
		}

		// 增加潜力点
		artifact.Potential += stage.UnlockPotential

		// 器灵好感
		s.addSpiritBondXP(ctx, artifactID, int64(stage.UnlockPotential*5))

		s.recalcBonuses(artifact)
		s.artifactRepo.Update(artifact)

		// 更新或创建进度
		if progress.LastAttemptDate == "" {
			s.artifactRepo.CreateTrialProgress(progress)
		} else {
			s.artifactRepo.UpdateTrialProgress(progress)
		}

		return &model.ArtifactTrialResult{
			Stage:        stage,
			Victory:      true,
			Rewards:      rewards,
			NewPotential: stage.UnlockPotential,
			SpiritBond:   int64(stage.UnlockPotential * 5),
		}, nil
	}

	// 失败
	if progress.LastAttemptDate != "" {
		s.artifactRepo.UpdateTrialProgress(progress)
	}

	return &model.ArtifactTrialResult{
		Stage:   stage,
		Victory: false,
		Rewards: map[string]int64{},
	}, nil
}

// simulateTrialBattle 模拟试炼战斗
func (s *ArtifactService) simulateTrialBattle(artifact *model.Artifact, stage *model.ArtifactTrialStage) bool {
	// 计算法宝战力
	artifactPower := float64(artifact.AttackBonus)*2 + float64(artifact.DefenseBonus)*1.5 + float64(artifact.HPBonus)*0.5 + float64(artifact.MpBonus)*0.8
	// 觉醒技能加成
	for _, sid := range artifact.AwakenSkills {
		switch sid {
		case model.ArtifactAwakenSoulMerge:
			artifactPower *= 1.25
		case model.ArtifactAwakenSwordRain, model.ArtifactAwakenVoidSlash:
			artifactPower *= 1.15
		}
	}

	// 怪物战力
	monsterPower := float64(stage.MonsterAttack)*2 + float64(stage.MonsterDefense)*1.5 + float64(stage.MonsterHP)*0.3

	// 基础胜率
	winRate := artifactPower / (artifactPower + monsterPower)

	// 加入随机因素
	roll := rand.Float64()
	return roll < winRate
}

// ============================================================
// 7. 查询
// ============================================================

// GetArtifactBonus 获取法宝战斗属性加成（增强版）
func (s *ArtifactService) GetArtifactBonus(ctx context.Context, playerID int64, artifactID int64) (*model.ArtifactResponse, error) {
	var artifact *model.Artifact
	var err error

	if artifactID > 0 {
		artifact, err = s.artifactRepo.GetByID(artifactID)
	} else {
		artifact, err = s.artifactRepo.GetByPlayerID(playerID)
	}
	if err != nil {
		return nil, fmt.Errorf("查询法宝失败: %w", err)
	}

	resp := &model.ArtifactResponse{
		TotalBonus: map[string]int64{
			"attack": 0, "defense": 0, "hp": 0, "mp": 0, "speed": 0, "dodge": 0, "power": 0,
		},
	}

	if artifact == nil {
		return resp, nil
	}

	resp.Artifact = artifact
	resp.QualityName = model.ArtifactQualityNames[artifact.Quality]
	resp.TypeName = model.ArtifactTypeNames[artifact.Type]
	resp.TypeIcon = model.ArtifactTypeIcons[artifact.Type]
	resp.SkillName = model.ArtifactSkillNames[artifact.SkillID]
	if resp.SkillName == "" {
		resp.SkillName = "无"
	}

	// 觉醒技能
	resp.AwakenSkills = make(map[int]string)
	for _, sid := range artifact.AwakenSkills {
		name := model.ArtifactSkillNames[sid]
		if name != "" {
			resp.AwakenSkills[sid] = name
		}
	}

	// 基础加成
	resp.TotalBonus["attack"] = artifact.AttackBonus
	resp.TotalBonus["defense"] = artifact.DefenseBonus
	resp.TotalBonus["hp"] = artifact.HPBonus
	resp.TotalBonus["mp"] = artifact.MpBonus
	resp.TotalBonus["speed"] = artifact.SpeedBonus
	resp.TotalBonus["dodge"] = artifact.DodgeBonus
	resp.TotalBonus["power"] = artifact.PowerBonus

	// 获取器灵
	spirit, _ := s.artifactRepo.GetSpiritByArtifactID(artifact.ID)
	if spirit != nil {
		resp.Spirit = spirit
	}

	// 获取所有法宝计算共鸣
	allArtifacts, _ := s.artifactRepo.GetMultipleByPlayerID(playerID)
	if len(allArtifacts) > 1 {
		resp.Resonance = s.calcResonance(allArtifacts)
		// 加上共鸣加成
		if resp.Resonance != nil {
			for k, v := range resp.Resonance.ActiveBonuses {
				resp.TotalBonus[k] += v
			}
		}
	}

	// 获取试炼进度
	trialProgress, _ := s.artifactRepo.GetTrialProgress(artifact.ID)
	if trialProgress != nil {
		resp.TrialProgress = trialProgress
	}

	return resp, nil
}

// ============================================================
// 辅助方法
// ============================================================

// recalcBonuses 根据品质、等级、类型重新计算法宝属性加成
// 基础值 = 品质系数 * 等级 * 类型倍率
func (s *ArtifactService) recalcBonuses(a *model.Artifact) {
	qualityMult := float64(a.Quality) // 1-5
	levelMult := float64(a.Level)     // 1-100
	multipliers := model.ArtifactTypeBonusMultipliers[a.Type]

	baseAttack := qualityMult * levelMult * 10
	baseDefense := qualityMult * levelMult * 8
	baseHP := qualityMult * levelMult * 50
	baseMP := qualityMult * levelMult * 5
	baseSpeed := qualityMult * levelMult * 2
	baseDodge := qualityMult * levelMult * 1

	a.AttackBonus = int64(baseAttack * multipliers[0])
	a.DefenseBonus = int64(baseDefense * multipliers[1])
	a.HPBonus = int64(baseHP * multipliers[2])
	a.MpBonus = int64(baseMP * multipliers[3])
	a.SpeedBonus = int64(baseSpeed * multipliers[4])
	a.DodgeBonus = int64(baseDodge * multipliers[5])

	// 觉醒技能加成
	for _, sid := range a.AwakenSkills {
		switch sid {
		case model.ArtifactAwakenSoulMerge:
			a.AttackBonus = int64(float64(a.AttackBonus) * 1.25)
			a.DefenseBonus = int64(float64(a.DefenseBonus) * 1.25)
			a.HPBonus = int64(float64(a.HPBonus) * 1.25)
			a.MpBonus = int64(float64(a.MpBonus) * 1.25)
		}
	}

	// 战力计算
	a.PowerBonus = a.AttackBonus*2 + a.DefenseBonus*2 + a.HPBonus + a.MpBonus*3 + a.SpeedBonus*5 + a.DodgeBonus*8
}

// unlockSkill 升品时概率解锁技能
func (s *ArtifactService) unlockSkill(a *model.Artifact) {
	if a.SkillID != 0 {
		return
	}
	if a.Quality < model.ArtifactQualitySpirit {
		return
	}

	var unlockRate float64
	switch a.Quality {
	case model.ArtifactQualitySpirit:
		unlockRate = 0.30
	case model.ArtifactQualityImmortal:
		unlockRate = 0.50
	case model.ArtifactQualityDivine:
		unlockRate = 0.80
	case model.ArtifactQualityChaos:
		unlockRate = 1.00
	default:
		return
	}

	if rand.Float64() < unlockRate {
		skillIDs := []int{
			model.ArtifactSkillSwordShield,
			model.ArtifactSkillArmorBreak,
			model.ArtifactSkillHeal,
			model.ArtifactSkillIronBody,
		}
		if a.Quality >= model.ArtifactQualityChaos {
			skillIDs = append(skillIDs, model.ArtifactSkillChaosForce)
		}
		a.SkillID = skillIDs[rand.Intn(len(skillIDs))]

		s.log.Info("法宝领悟技能",
			zap.Int64("artifact_id", a.ID),
			zap.Int("skill_id", a.SkillID),
			zap.String("skill_name", model.ArtifactSkillNames[a.SkillID]),
		)
	}
}

// CalcBattleSkillTrigger 战斗时判断法宝技能是否触发（兼容旧接口）
func CalcBattleSkillTrigger(artifact *model.Artifact) (bool, int, string) {
	if artifact == nil || artifact.SkillID == 0 {
		return false, 0, ""
	}

	triggerRate := 0.15 + float64(artifact.Level)*0.001
	triggerRate = math.Min(triggerRate, 0.25)

	if rand.Float64() < triggerRate {
		return true, artifact.SkillID, model.ArtifactSkillNames[artifact.SkillID]
	}
	return false, 0, ""
}
