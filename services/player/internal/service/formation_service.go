package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// FormationService 阵法业务逻辑
type FormationService struct {
	formationRepo *mysql.FormationRepo
	playerRepo    *mysql.PlayerRepo
	petRepo       *mysql.PetRepo
	templates     []*model.FormationTemplate
	log           *zap.Logger
}

// NewFormationService 创建 FormationService
func NewFormationService(formationRepo *mysql.FormationRepo, playerRepo *mysql.PlayerRepo, petRepo *mysql.PetRepo, log *zap.Logger) *FormationService {
	return &FormationService{
		formationRepo: formationRepo,
		playerRepo:    playerRepo,
		petRepo:       petRepo,
		log:           log,
	}
}

// LoadTemplates 加载阵法图谱（启动时调用）
func (s *FormationService) LoadTemplates(path string) error {
	templates, err := s.formationRepo.LoadTemplates(path)
	if err != nil {
		return err
	}
	s.templates = templates
	s.log.Info("阵法图谱加载完成", zap.Int("count", len(templates)))
	return nil
}

// GetTemplateByID 根据 ID 获取阵法模板
func (s *FormationService) GetTemplateByID(tmplID int) *model.FormationTemplate {
	for _, t := range s.templates {
		if t.ID == tmplID {
			return t
		}
	}
	return nil
}

// GetAllTemplates 获取所有阵法图谱
func (s *FormationService) GetAllTemplates() []*model.FormationTemplate {
	return s.templates
}

// EnsureTemplatesLoaded 确保阵法图谱已加载
func (s *FormationService) EnsureTemplatesLoaded() error {
	if s.templates == nil {
		return fmt.Errorf("阵法图谱尚未加载，请先调用 LoadTemplates")
	}
	return nil
}

// ============================================================
// 基础 CRUD（保留原有 + 增强）
// ============================================================

// LearnFormation 学习阵法图谱
func (s *FormationService) LearnFormation(ctx context.Context, playerID int64, tmplID int) (*model.Formation, error) {
	tmpl := s.GetTemplateByID(tmplID)
	if tmpl == nil {
		return nil, fmt.Errorf("阵法图谱不存在")
	}

	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}
	if player.Level < 21 {
		return nil, fmt.Errorf("金丹期(21层)以上才能学习阵法，当前等级 %d", player.Level)
	}

	formations, err := s.formationRepo.ListByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询已有阵法失败: %w", err)
	}
	for _, f := range formations {
		if f.TmplID == tmplID {
			return nil, fmt.Errorf("已学习过阵法[%s]，不可重复学习", tmpl.Name)
		}
	}

	if player.Gold < tmpl.LearnCost {
		return nil, fmt.Errorf("灵石不足，学习需要 %d 灵石，当前 %d", tmpl.LearnCost, player.Gold)
	}

	player.Gold -= tmpl.LearnCost
	if err := s.playerRepo.Update(player); err != nil {
		return nil, fmt.Errorf("扣除灵石失败: %w", err)
	}

	effects := make([]model.FormationEffect, len(tmpl.Effects))
	copy(effects, tmpl.Effects)

	formation := &model.Formation{
		PlayerID:     playerID,
		TmplID:       tmpl.ID,
		Name:         tmpl.Name,
		Type:         tmpl.Type,
		Level:        model.FormationLevelMin,
		Quality:      tmpl.Quality,
		Deployed:     false,
		Guardian:     false,
		Exp:          0,
		Effects:      effects,
		MasteryExp:   0,
		MasteryLevel: 0,
		LinkGroup:    0,
	}

	if err := s.formationRepo.Create(formation); err != nil {
		return nil, err
	}

	s.log.Info("学习阵法成功",
		zap.Int64("player_id", playerID),
		zap.String("name", tmpl.Name),
		zap.Int("tmpl_id", tmplID),
	)
	return formation, nil
}

// ListFormations 查询玩家所有阵法（含增强响应字段）
func (s *FormationService) ListFormations(ctx context.Context, playerID int64) ([]*model.FormationResponse, error) {
	formations, err := s.formationRepo.ListByPlayerID(playerID)
	if err != nil {
		return nil, err
	}

	// 获取所有灵兽用于填充守护守护名称
	allPets, _ := s.petRepo.ListByPlayerID(playerID)
	petMap := make(map[int64]*model.Pet)
	for _, p := range allPets {
		petMap[p.ID] = p
	}

	var resp []*model.FormationResponse
	deployIdx := 0
	for _, f := range formations {
		masteryName := model.MasteryLevelNames[f.MasteryLevel]
		if masteryName == "" {
			masteryName = "初窥门径"
		}
		progress := 0.0
		needExp := model.CalcMasteryExpRequired(f.MasteryLevel)
		if needExp > 0 {
			progress = float64(f.MasteryExp) / float64(needExp)
			if progress > 1.0 {
				progress = 1.0
			}
		}

		r := &model.FormationResponse{
			Formation:        f,
			TypeName:         model.FormationTypeNames[f.Type],
			QualityName:      model.FormationQualityNames[f.Quality],
			MasteryLevelName: masteryName,
			MasteryProgress:  progress,
		}

		if f.Deployed {
			deployIdx++
			r.DeployIdx = deployIdx
		}

		// 联动加成描述
		if f.LinkGroup > 0 {
			// 查找同组的其他阵法
			for _, other := range formations {
				if other.ID != f.ID && other.LinkGroup == f.LinkGroup && other.Deployed {
					if syn := model.FindSynergy(f.Type, other.Type); syn != nil {
						r.SynergyBonus = syn.Bonus
						break
					}
				}
			}
		}

		// 守护灵兽信息
		if f.GuardianPetID != nil {
			r.HasGuardian = true
			if pet, ok := petMap[*f.GuardianPetID]; ok {
				r.GuardianPetName = pet.Name
				r.GuardianContribution = model.CalcGuardianPetContribution(pet.Atk, pet.Def, pet.Star, pet.Level)
			}
		}

		resp = append(resp, r)
	}
	return resp, nil
}

// DeployFormation 部署阵法
func (s *FormationService) DeployFormation(ctx context.Context, playerID, formationID int64) (*model.Formation, error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil {
		return nil, fmt.Errorf("阵法不存在")
	}
	if formation.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此阵法")
	}
	if formation.Deployed {
		return nil, fmt.Errorf("阵法[%s]已处于部署状态", formation.Name)
	}

	deployed, err := s.formationRepo.GetDeployedByPlayerID(playerID)
	if err != nil {
		return nil, err
	}
	if len(deployed) >= model.MaxDeployedFormations {
		return nil, fmt.Errorf("最多同时部署 %d 个阵法，请先撤销其他部署", model.MaxDeployedFormations)
	}

	formation.Deployed = true
	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}

	s.log.Info("部署阵法成功",
		zap.Int64("player_id", playerID),
		zap.Int64("formation_id", formationID),
		zap.String("name", formation.Name),
	)
	return formation, nil
}

// UndeployFormation 撤销阵法部署
func (s *FormationService) UndeployFormation(ctx context.Context, playerID, formationID int64) (*model.Formation, error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil {
		return nil, fmt.Errorf("阵法不存在")
	}
	if formation.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此阵法")
	}
	if !formation.Deployed {
		return nil, fmt.Errorf("阵法[%s]未部署", formation.Name)
	}

	// 撤销部署时清除联动组
	formation.Deployed = false
	formation.LinkGroup = 0
	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}
	return formation, nil
}

// UpgradeFormation 升级阵法等级
func (s *FormationService) UpgradeFormation(ctx context.Context, playerID, formationID int64) (*model.Formation, error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil {
		return nil, fmt.Errorf("阵法不存在")
	}
	if formation.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此阵法")
	}
	if formation.Level >= model.FormationLevelMax {
		return nil, fmt.Errorf("阵法已达最高等级(%d级)", model.FormationLevelMax)
	}

	goldCost := int64(formation.Level * 2000)
	expRequired := int64(formation.Level * 500)

	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}
	if player.Gold < goldCost {
		return nil, fmt.Errorf("灵石不足，升级需要 %d 灵石，当前 %d", goldCost, player.Gold)
	}
	if formation.Exp < expRequired {
		return nil, fmt.Errorf("阵法经验不足，需要 %d 经验，当前 %d", expRequired, formation.Exp)
	}

	player.Gold -= goldCost
	formation.Exp -= expRequired
	formation.Level++

	s.recalcEffects(formation)

	if err := s.playerRepo.Update(player); err != nil {
		return nil, fmt.Errorf("扣除灵石失败: %w", err)
	}
	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}

	s.log.Info("阵法升级成功",
		zap.Int64("player_id", playerID),
		zap.String("name", formation.Name),
		zap.Int("new_level", formation.Level),
	)
	return formation, nil
}

// AddFormationExp 增加阵法经验
func (s *FormationService) AddFormationExp(ctx context.Context, playerID, formationID int64, exp int64) (*model.Formation, error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, err
	}
	if formation == nil || formation.PlayerID != playerID {
		return nil, fmt.Errorf("阵法不存在或无权操作")
	}

	formation.Exp += exp
	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}
	return formation, nil
}

// ============================================================
// 护法（突破加持）— 保留原有
// ============================================================

// SetGuardian 设置护法阵法
func (s *FormationService) SetGuardian(ctx context.Context, playerID, formationID int64) (*model.Formation, error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil {
		return nil, fmt.Errorf("阵法不存在")
	}
	if formation.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此阵法")
	}

	if err := s.formationRepo.ClearGuardianByPlayerID(playerID); err != nil {
		return nil, err
	}

	formation.Guardian = true
	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}

	s.log.Info("设置护法阵法成功",
		zap.Int64("player_id", playerID),
		zap.Int64("formation_id", formationID),
		zap.String("name", formation.Name),
	)
	return formation, nil
}

// UnsetGuardian 取消护法阵法
func (s *FormationService) UnsetGuardian(ctx context.Context, playerID int64) error {
	return s.formationRepo.ClearGuardianByPlayerID(playerID)
}

// GetGuardianBonus 计算护法加成（保留原算法）
func GetGuardianBonus(formation *model.Formation) float64 {
	if formation == nil || !formation.Guardian {
		return 0
	}
	bonus := 0.05 + float64(formation.Level-1)*0.01 + float64(formation.Quality-1)*0.005
	return math.Min(bonus, 0.15)
}

// PerformGuardian 执行护法突破
func (s *FormationService) PerformGuardian(ctx context.Context, guardianPlayerID, beneficiaryPlayerID int64, baseBreakRate float64) (float64, bool, error) {
	formation, err := s.formationRepo.GetGuardianByPlayerID(guardianPlayerID)
	if err != nil {
		return 0, false, fmt.Errorf("查询护法阵法失败: %w", err)
	}
	if formation == nil {
		return 0, false, fmt.Errorf("护法方未设置护法阵法")
	}

	bonus := GetGuardianBonus(formation)
	totalRate := baseBreakRate + bonus

	success := rand.Float64() < totalRate

	task := &model.GuardianTask{
		GuardianID:    guardianPlayerID,
		BeneficiaryID: beneficiaryPlayerID,
		FormationID:   formation.ID,
		BonusRate:     bonus,
		Success:       success,
	}
	if err := s.formationRepo.CreateGuardianTask(task); err != nil {
		s.log.Error("创建护法记录失败", zap.Error(err))
	}

	s.log.Info("护法突破完成",
		zap.Int64("guardian", guardianPlayerID),
		zap.Int64("beneficiary", beneficiaryPlayerID),
		zap.Float64("bonus", bonus),
		zap.Bool("success", success),
	)
	return bonus, success, nil
}

// GetDeployedBonuses 获取玩家所有已部署阵法的战斗加成
func (s *FormationService) GetDeployedBonuses(ctx context.Context, playerID int64) ([]model.FormationEffect, error) {
	deployed, err := s.formationRepo.GetDeployedByPlayerID(playerID)
	if err != nil {
		return nil, err
	}

	var allEffects []model.FormationEffect
	for _, f := range deployed {
		// 应用熟练度倍率
		masteryMult := model.CalcMasteryMultiplier(f.MasteryLevel)
		for _, e := range f.Effects {
			modified := model.FormationEffect{
				Type:  e.Type,
				Value: e.Value * masteryMult,
			}
			allEffects = append(allEffects, modified)
		}

		// 如果有守护灵兽，加入灵兽贡献效果
		if f.GuardianPetID != nil {
			pet, err := s.petRepo.GetByID(*f.GuardianPetID)
			if err == nil && pet != nil && pet.PlayerID == playerID {
				contrib := model.CalcGuardianPetContribution(pet.Atk, pet.Def, pet.Star, pet.Level)
				if contrib > 0 {
					allEffects = append(allEffects, model.FormationEffect{
						Type:  "guardian_bonus",
						Value: contrib,
					})
				}
			}
		}
	}

	// 处理联动加成
	linkResult := s.calcLinkBonuses(deployed)
	if linkResult.TotalAtkBonus > 0 {
		allEffects = append(allEffects, model.FormationEffect{Type: "link_atk", Value: linkResult.TotalAtkBonus})
	}
	if linkResult.TotalDefBonus > 0 {
		allEffects = append(allEffects, model.FormationEffect{Type: "link_def", Value: linkResult.TotalDefBonus})
	}
	if linkResult.TotalOtherBonus != 0 {
		allEffects = append(allEffects, model.FormationEffect{Type: "link_other", Value: linkResult.TotalOtherBonus})
	}

	return allEffects, nil
}

// GetGuardianHistory 查询护法历史记录
func (s *FormationService) GetGuardianHistory(ctx context.Context, playerID int64, limit int) ([]*model.GuardianTask, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.formationRepo.ListGuardianTasksByPlayer(playerID, limit)
}

// recalcEffects 根据等级、品质、熟练度重新计算阵法效果
// 效果 = 基础效果 * (1 + 等级*0.1) * (1 + (品质-1)*0.05) * 熟练度倍率
func (s *FormationService) recalcEffects(f *model.Formation) {
	tmpl := s.GetTemplateByID(f.TmplID)
	if tmpl == nil {
		return
	}

	levelMult := 1.0 + float64(f.Level)*0.1
	qualityMult := 1.0 + float64(f.Quality-1)*0.05
	masteryMult := model.CalcMasteryMultiplier(f.MasteryLevel)

	effects := make([]model.FormationEffect, len(tmpl.Effects))
	for i, e := range tmpl.Effects {
		effects[i] = model.FormationEffect{
			Type:  e.Type,
			Value: e.Value * levelMult * qualityMult * masteryMult,
		}
	}
	f.Effects = effects
}

// ============================================================
// ========== 新增功能 ==========
// ============================================================

// ============================================================
// 1. 熟练度系统
// ============================================================

// AddFormationMastery 增加阵法熟练度
// 每次战斗/使用阵法后调用，获得熟练度经验，自动升级
func (s *FormationService) AddFormationMastery(ctx context.Context, playerID, formationID int64, baseExp int64) (*model.Formation, error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil || formation.PlayerID != playerID {
		return nil, fmt.Errorf("阵法不存在或无权操作")
	}
	if formation.MasteryLevel >= model.MasteryLevelMax {
		return formation, nil // 满级不再增加
	}

	// 基础经验 * 品质系数 * 等级系数
	qMult := 1.0 + float64(formation.Quality-1)*0.1
	lMult := 1.0 + float64(formation.Level)*0.05
	gain := int64(float64(baseExp) * qMult * lMult)
	if gain < 1 {
		gain = 1
	}

	formation.MasteryExp += gain

	// 检查是否升级
	upgraded := false
	for formation.MasteryLevel < model.MasteryLevelMax {
		needExp := model.CalcMasteryExpRequired(formation.MasteryLevel)
		if needExp <= 0 {
			break
		}
		if formation.MasteryExp < needExp {
			break
		}
		formation.MasteryExp -= needExp
		formation.MasteryLevel++
		upgraded = true
	}

	if upgraded {
		// 升级后重新计算效果
		s.recalcEffects(formation)
		s.log.Info("阵法熟练度提升",
			zap.Int64("player_id", playerID),
			zap.String("name", formation.Name),
			zap.Int("new_mastery_level", formation.MasteryLevel),
		)
	}

	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}
	return formation, nil
}

// GetFormationMasteryInfo 获取阵法熟练度信息
func (s *FormationService) GetFormationMasteryInfo(ctx context.Context, playerID, formationID int64) (currentExp, needExp int64, level int, levelName string, err error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return 0, 0, 0, "", fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil || formation.PlayerID != playerID {
		return 0, 0, 0, "", fmt.Errorf("阵法不存在或无权操作")
	}

	currentExp = formation.MasteryExp
	level = formation.MasteryLevel
	levelName = model.MasteryLevelNames[level]
	if level >= model.MasteryLevelMax {
		needExp = 0
	} else {
		needExp = model.CalcMasteryExpRequired(level)
	}
	return
}

// ============================================================
// 2. 守护灵兽系统
// ============================================================

// AssignGuardianPet 指派灵兽作为阵法的守护灵兽
func (s *FormationService) AssignGuardianPet(ctx context.Context, playerID, formationID, petID int64) (*model.Formation, error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil || formation.PlayerID != playerID {
		return nil, fmt.Errorf("阵法不存在或无权操作")
	}

	pet, err := s.petRepo.GetByID(petID)
	if err != nil {
		return nil, fmt.Errorf("查询灵兽失败: %w", err)
	}
	if pet == nil || pet.PlayerID != playerID {
		return nil, fmt.Errorf("灵兽不存在或无权操作")
	}

	// 检查这只灵兽是否已被其他阵法作为守护
	existing, err := s.formationRepo.GetByGuardianPetID(petID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID != formationID {
		return nil, fmt.Errorf("该灵兽已是阵法[%s]的守护，请先解除", existing.Name)
	}

	// 如果该阵法已有守护灵兽，先清除
	if formation.GuardianPetID != nil {
		oldPetID := *formation.GuardianPetID
		if oldPetID != petID {
			s.log.Info("替换守护灵兽",
				zap.Int64("formation_id", formationID),
				zap.Int64("old_pet_id", oldPetID),
				zap.Int64("new_pet_id", petID),
			)
		}
	}

	formation.GuardianPetID = &petID
	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}

	// 获得一些熟练度
	s.AddFormationMastery(ctx, playerID, formationID, model.MasteryExpBase/2)

	s.log.Info("指派守护灵兽成功",
		zap.Int64("player_id", playerID),
		zap.Int64("formation_id", formationID),
		zap.Int64("pet_id", petID),
		zap.String("pet_name", pet.Name),
	)
	return formation, nil
}

// RemoveGuardianPet 解除阵法守护灵兽
func (s *FormationService) RemoveGuardianPet(ctx context.Context, playerID, formationID int64) (*model.Formation, error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil || formation.PlayerID != playerID {
		return nil, fmt.Errorf("阵法不存在或无权操作")
	}
	if formation.GuardianPetID == nil {
		return nil, fmt.Errorf("该阵法没有守护灵兽")
	}

	formation.GuardianPetID = nil
	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}

	s.log.Info("解除守护灵兽成功",
		zap.Int64("player_id", playerID),
		zap.Int64("formation_id", formationID),
	)
	return formation, nil
}

// ============================================================
// 3. 阵法联动（Linking）系统
// ============================================================

// SetFormationLink 设置联动组
// 将指定阵法加入某个联动组，同组的已部署阵法之间会产生联动效果
func (s *FormationService) SetFormationLink(ctx context.Context, playerID, formationID int64, group int) (*model.Formation, error) {
	if group < 1 || group > model.MaxDeployedFormations {
		return nil, fmt.Errorf("联动组编号必须在 1-%d 之间", model.MaxDeployedFormations)
	}

	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil || formation.PlayerID != playerID {
		return nil, fmt.Errorf("阵法不存在或无权操作")
	}
	if !formation.Deployed {
		return nil, fmt.Errorf("只有已部署的阵法才能设置联动")
	}

	// 检查该联动组是否已有其他阵法
	deployed, err := s.formationRepo.GetDeployedByPlayerID(playerID)
	if err != nil {
		return nil, err
	}
	groupCount := 0
	for _, f := range deployed {
		if f.LinkGroup == group {
			groupCount++
		}
	}
	// 当前阵法可能已是该组，先不算它
	if formation.LinkGroup == group {
		groupCount--
	}
	if groupCount >= 2 {
		return nil, fmt.Errorf("联动组 %d 已有两个阵法，无法再加入", group)
	}

	formation.LinkGroup = group
	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}

	s.log.Info("设置阵法联动成功",
		zap.Int64("player_id", playerID),
		zap.Int64("formation_id", formationID),
		zap.Int("group", group),
	)
	return formation, nil
}

// ClearFormationLink 清除阵法联动
func (s *FormationService) ClearFormationLink(ctx context.Context, playerID, formationID int64) (*model.Formation, error) {
	formation, err := s.formationRepo.GetByID(formationID)
	if err != nil {
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	if formation == nil || formation.PlayerID != playerID {
		return nil, fmt.Errorf("阵法不存在或无权操作")
	}

	formation.LinkGroup = 0
	if err := s.formationRepo.Update(formation); err != nil {
		return nil, err
	}
	return formation, nil
}

// ClearAllLinks 清除所有联动组
func (s *FormationService) ClearAllLinks(ctx context.Context, playerID int64) error {
	return s.formationRepo.ClearLinkGroup(playerID)
}

// GetLinkBonuses 获取当前联动加成详情
func (s *FormationService) GetLinkBonuses(ctx context.Context, playerID int64) (*model.FormationLinkResult, error) {
	deployed, err := s.formationRepo.GetDeployedByPlayerID(playerID)
	if err != nil {
		return nil, err
	}

	return s.calcLinkBonuses(deployed), nil
}

// calcLinkBonuses 计算所有已部署阵法的联动加成（内部）
func (s *FormationService) calcLinkBonuses(deployed []*model.Formation) *model.FormationLinkResult {
	result := &model.FormationLinkResult{
		Deployed: deployed,
	}

	// 按联动组分组
	groups := make(map[int][]*model.Formation)
	for _, f := range deployed {
		if f.LinkGroup > 0 {
			groups[f.LinkGroup] = append(groups[f.LinkGroup], f)
		}
	}

	// 在每个组内查找联动组合
	seen := make(map[string]bool)
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				a, b := group[i], group[j]
				syn := model.FindSynergy(a.Type, b.Type)
				if syn == nil {
					continue
				}
				// 避免重复
				key := fmt.Sprintf("%d-%d", a.ID, b.ID)
				if seen[key] {
					continue
				}
				seen[key] = true

				synLevel := model.CalcSynergyLevel(a.MasteryLevel, b.MasteryLevel)
				mult := model.CalcSynergyMultiplier(synLevel)

				active := model.ActiveSynergy{
					TypeA: syn.TypeA,
					TypeB: syn.TypeB,
					Name:  syn.Name,
					Bonus: syn.Bonus,
					Level: synLevel,
					Mult:  mult,
				}

				// 累加效果
				if syn.AtkMult > 0 {
					result.TotalAtkBonus += syn.AtkMult * mult
				}
				if syn.DefMult > 0 {
					result.TotalDefBonus += syn.DefMult * mult
				}
				if syn.OtherPct != 0 {
					result.TotalOtherBonus += syn.OtherPct * mult
				}

				result.Synergies = append(result.Synergies, active)
			}
		}
	}

	if result.Synergies == nil {
		result.Synergies = []model.ActiveSynergy{}
	}
	return result
}

// ============================================================
// 4. 阵法相克（破阵）系统 — PVP
// ============================================================

// CalcFormationBreak 计算 PVP 中攻击方对防守方的阵法克制
// 返回：防守方效果削减比例列表、总削减比例、攻击方是否获得额外加成
func (s *FormationService) CalcFormationBreak(ctx context.Context, attackerID, defenderID int64) (*model.FormationBreakResult, error) {
	attackerFms, err := s.formationRepo.GetDeployedByPlayerID(attackerID)
	if err != nil {
		return nil, fmt.Errorf("查询攻击方阵法失败: %w", err)
	}
	defenderFms, err := s.formationRepo.GetDeployedByPlayerID(defenderID)
	if err != nil {
		return nil, fmt.Errorf("查询防守方阵法失败: %w", err)
	}

	result := &model.FormationBreakResult{
		AttackerID:  attackerID,
		DefenderID:  defenderID,
		AttackerFms: attackerFms,
		DefenderFms: defenderFms,
	}

	var totalReduction float64
	var breaks []model.SingleBreak

	// 攻击方的每个阵法尝试克制防守方的每个阵法
	for _, atk := range attackerFms {
		for _, def := range defenderFms {
			counter := model.FindCounter(atk.Type, def.Type)
			if counter == nil {
				continue
			}

			// 基础削弱的百分比受攻击方熟练度加成
			masteryMult := model.CalcMasteryMultiplier(atk.MasteryLevel)
			effectiveBreak := counter.BreakPct * masteryMult
			if effectiveBreak > 0.50 {
				effectiveBreak = 0.50 // 上限50%
			}

			totalReduction += effectiveBreak

			brk := model.SingleBreak{
				AttackerType: atk.Type,
				DefenderType: def.Type,
				BreakPct:     effectiveBreak,
				DefenderName: def.Name,
			}
			breaks = append(breaks, brk)
		}
	}

	// 总削减上限 80%
	if totalReduction > 0.80 {
		totalReduction = 0.80
	}

	result.Breaks = breaks
	result.TotalReduction = totalReduction

	// 如果总削减 > 0，攻击方获得额外加成（破阵后气势如虹）
	result.BonusActive = totalReduction > 0.10

	return result, nil
}

// ApplyFormationBreak 在 PVP 战斗中使用阵法相克
// 返回防守方效果修正系数（1.0 - totalReduction）
func (s *FormationService) ApplyFormationBreak(ctx context.Context, attackerID, defenderID int64) (float64, bool, error) {
	breakResult, err := s.CalcFormationBreak(ctx, attackerID, defenderID)
	if err != nil {
		return 1.0, false, err
	}

	// 防守方效果修正系数
	reduction := breakResult.TotalReduction
	modifier := 1.0 - reduction

	// 给攻击方出战的阵法增加熟练度（破阵经验）
	attackerFms, _ := s.formationRepo.GetDeployedByPlayerID(attackerID)
	for _, f := range attackerFms {
		// 破阵获得的熟练度更多
		bonusExp := int64(model.MasteryExpBase * 2)
		s.AddFormationMastery(ctx, attackerID, f.ID, bonusExp)
	}

	s.log.Info("阵法相克计算完成",
		zap.Int64("attacker", attackerID),
		zap.Int64("defender", defenderID),
		zap.Float64("reduction", reduction),
		zap.Bool("bonus_active", breakResult.BonusActive),
	)

	return modifier, breakResult.BonusActive, nil
}
