// Package service 炼丹系统增强业务逻辑 - 丹方研究、炼丹小游戏、丹毒、丹炉
package service

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"cultivation-game/services/cultivation/internal/model"
)

// EnhancedAlchemyService 炼丹增强服务
type EnhancedAlchemyService struct {
	mu           sync.RWMutex
	rng          *rand.Rand
	rngMu        sync.Mutex

	// 基础丹方（所有玩家初始知晓）
	baseFormulas map[int]*model.Formula
	// 所有可研究丹方（full recipe book）
	allFormulas map[int]*model.Formula

	// 玩家已研究的丹方 playerID -> formulaID -> PlayerFormula
	playerFormulas map[uint64]map[int]*model.PlayerFormula

	// 玩家研究尝试记录
	researchAttempts map[uint64]*model.ResearchAttempt

	// 炼丹小游戏会话
	sessions map[string]*model.AlchemySession // session key: "playerID:formulaID"

	// 玩家丹毒
	playerToxicity map[uint64]*model.PlayerToxicity

	// 玩家丹炉
	playerFurnaces map[uint64]*model.PlayerFurnace

	// 研究记录
	researchLog []*model.ResearchRecord
}

// NewEnhancedAlchemyService 创建炼丹增强服务
func NewEnhancedAlchemyService() *EnhancedAlchemyService {
	s := &EnhancedAlchemyService{
		rng:              rand.New(rand.NewSource(time.Now().UnixNano())),
		baseFormulas:     make(map[int]*model.Formula),
		allFormulas:      make(map[int]*model.Formula),
		playerFormulas:   make(map[uint64]map[int]*model.PlayerFormula),
		researchAttempts: make(map[uint64]*model.ResearchAttempt),
		sessions:         make(map[string]*model.AlchemySession),
		playerToxicity:   make(map[uint64]*model.PlayerToxicity),
		playerFurnaces:   make(map[uint64]*model.PlayerFurnace),
		researchLog:      make([]*model.ResearchRecord, 0),
	}
	s.initFormulas()
	return s
}

// initFormulas 初始化丹方数据
func (s *EnhancedAlchemyService) initFormulas() {
	// ---- 基础丹方（3个，所有玩家初始可知） ----
	baseRecipes := []*model.Formula{
		{
			ID: 1, Name: "聚气丹", Description: "基础修炼丹药，提升灵气吸收速度",
			Materials: []string{"101"}, BaseQuality: 1, MinLevel: 1, RealmRequired: 1,
			ResearchDifficulty: 0.0, IsRare: false, Effect: "修炼速度 +10%，持续30分钟",
			CraftTime: 30, ExpValue: 100,
		},
		{
			ID: 2, Name: "引气丹", Description: "简单引气入体丹药",
			Materials: []string{"101"}, BaseQuality: 1, MinLevel: 1, RealmRequired: 1,
			ResearchDifficulty: 0.0, IsRare: false, Effect: "修为 +100",
			CraftTime: 10, ExpValue: 50,
		},
		{
			ID: 3, Name: "金创丹", Description: "疗伤止血的基础丹药",
			Materials: []string{"101", "102"}, BaseQuality: 1, MinLevel: 2, RealmRequired: 1,
			ResearchDifficulty: 0.0, IsRare: false, Effect: "HP恢复 +200",
			CraftTime: 20, ExpValue: 80,
		},
	}

	for _, f := range baseRecipes {
		s.baseFormulas[f.ID] = f
		s.allFormulas[f.ID] = f
	}

	// ---- 可研究丹方 ----
	researchable := []*model.Formula{
		{
			ID: 4, Name: "培元丹", Description: "固本培元，提升基础修为",
			Materials: []string{"101", "102"}, BaseQuality: 1, MinLevel: 3, RealmRequired: 1,
			ResearchDifficulty: 0.3, IsRare: false, Effect: "修为 +500，HP/MP +50",
			CraftTime: 60, ExpValue: 200,
		},
		{
			ID: 5, Name: "清心丹", Description: "清心凝神，提升闭关效率",
			Materials: []string{"101", "105"}, BaseQuality: 1, MinLevel: 4, RealmRequired: 1,
			ResearchDifficulty: 0.35, IsRare: false, Effect: "闭关效率 +30%，持续1小时",
			CraftTime: 45, ExpValue: 250,
		},
		{
			ID: 6, Name: "炼体丹", Description: "锤炼肉身，增加气血",
			Materials: []string{"102", "103"}, BaseQuality: 2, MinLevel: 5, RealmRequired: 1,
			ResearchDifficulty: 0.4, IsRare: false, Effect: "HP上限 +100，防御 +20",
			CraftTime: 60, ExpValue: 300,
		},
		{
			ID: 7, Name: "筑基丹", Description: "突破筑基期必备丹药",
			Materials: []string{"102", "103", "104"}, BaseQuality: 2, MinLevel: 7, RealmRequired: 1,
			ResearchDifficulty: 0.5, IsRare: false, Effect: "筑基突破成功率 +30%",
			CraftTime: 120, ExpValue: 500,
		},
		{
			ID: 8, Name: "凝神丹", Description: "凝神静气，突破时增加时限",
			Materials: []string{"103", "105"}, BaseQuality: 2, MinLevel: 1, RealmRequired: 2,
			ResearchDifficulty: 0.45, IsRare: false, Effect: "突破时限 +30秒",
			CraftTime: 90, ExpValue: 400,
		},
		{
			ID: 9, Name: "回元丹", Description: "恢复灵力，加快修炼",
			Materials: []string{"102", "104"}, BaseQuality: 2, MinLevel: 2, RealmRequired: 2,
			ResearchDifficulty: 0.4, IsRare: false, Effect: "MP +100，修炼速度 +30%",
			CraftTime: 60, ExpValue: 350,
		},
		{
			ID: 10, Name: "护脉丹", Description: "突破失败时保护经脉",
			Materials: []string{"103", "106"}, BaseQuality: 2, MinLevel: 3, RealmRequired: 2,
			ResearchDifficulty: 0.5, IsRare: false, Effect: "突破失败修为损失 -15%",
			CraftTime: 90, ExpValue: 450,
		},
		{
			ID: 11, Name: "续骨丹", Description: "治愈重伤的灵丹",
			Materials: []string{"103", "106"}, BaseQuality: 2, MinLevel: 4, RealmRequired: 2,
			ResearchDifficulty: 0.5, IsRare: false, Effect: "HP恢复 +500，HP上限 +300",
			CraftTime: 90, ExpValue: 450,
		},
		{
			ID: 12, Name: "聚灵丹", Description: "扩大节点判定范围",
			Materials: []string{"105", "104"}, BaseQuality: 3, MinLevel: 5, RealmRequired: 2,
			ResearchDifficulty: 0.55, IsRare: false, Effect: "节点判定范围 +20%",
			CraftTime: 120, ExpValue: 500,
		},
		// ---- 稀有丹方（需要Boss掉落材料） ----
		{
			ID: 13, Name: "蕴神丹", Description: "蕴含神魂之力，大幅提升修为",
			Materials: []string{"106", "107"}, BaseQuality: 3, MinLevel: 6, RealmRequired: 2,
			ResearchDifficulty: 0.7, IsRare: true, Effect: "修为 +8000，突破 +10%",
			CraftTime: 120, ExpValue: 800,
		},
		{
			ID: 14, Name: "金丹破境丹", Description: "突破金丹期的关键丹药",
			Materials: []string{"106", "107", "108"}, BaseQuality: 3, MinLevel: 1, RealmRequired: 3,
			ResearchDifficulty: 0.75, IsRare: true, Effect: "金丹突破成功率 +25%",
			CraftTime: 180, ExpValue: 1200,
		},
		{
			ID: 15, Name: "太清丹", Description: "金丹期修炼圣药",
			Materials: []string{"108", "105"}, BaseQuality: 3, MinLevel: 2, RealmRequired: 3,
			ResearchDifficulty: 0.7, IsRare: true, Effect: "修为 +30000，修炼速度 +100%",
			CraftTime: 180, ExpValue: 1500,
		},
		{
			ID: 16, Name: "清虚丹", Description: "提升闭关效率的宝丹",
			Materials: []string{"108", "107"}, BaseQuality: 3, MinLevel: 3, RealmRequired: 3,
			ResearchDifficulty: 0.7, IsRare: false, Effect: "修为 +20000，闭关效率 +100%",
			CraftTime: 150, ExpValue: 1000,
		},
		{
			ID: 17, Name: "玉灵丹", Description: "全面提升属性的珍品丹药",
			Materials: []string{"107", "109"}, BaseQuality: 4, MinLevel: 4, RealmRequired: 3,
			ResearchDifficulty: 0.8, IsRare: true, Effect: "HP/MP +500，防御 +100",
			CraftTime: 180, ExpValue: 2000,
		},
		{
			ID: 18, Name: "九转丹", Description: "九转凝练的无上丹药",
			Materials: []string{"109", "110"}, BaseQuality: 4, MinLevel: 6, RealmRequired: 3,
			ResearchDifficulty: 0.85, IsRare: true, Effect: "修为 +50000，突破 +20%",
			CraftTime: 240, ExpValue: 3000,
		},
	}

	for _, f := range researchable {
		s.allFormulas[f.ID] = f
	}
}

// ==================== 丹方研究系统 ====================

// GetBaseFormulas 获取基础丹方列表
func (s *EnhancedAlchemyService) GetBaseFormulas() []*model.Formula {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.Formula, 0, len(s.baseFormulas))
	for _, f := range s.baseFormulas {
		result = append(result, f)
	}
	return result
}

// GetAllFormulas 获取所有丹方
func (s *EnhancedAlchemyService) GetAllFormulas() []*model.Formula {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.Formula, 0, len(s.allFormulas))
	for _, f := range s.allFormulas {
		result = append(result, f)
	}
	return result
}

// GetFormulaByID 按ID获取丹方
func (s *EnhancedAlchemyService) GetFormulaByID(id int) (*model.Formula, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.allFormulas[id]
	return f, ok
}

// GetPlayerFormulas 获取玩家已研究的丹方
func (s *EnhancedAlchemyService) GetPlayerFormulas(playerID uint64) []*model.PlayerFormula {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if formulas, ok := s.playerFormulas[playerID]; ok {
		result := make([]*model.PlayerFormula, 0, len(formulas))
		for _, pf := range formulas {
			result = append(result, pf)
		}
		return result
	}
	return nil
}

// HasPlayerResearchedFormula 检查玩家是否已研究某丹方
func (s *EnhancedAlchemyService) HasPlayerResearchedFormula(playerID uint64, formulaID int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if formulas, ok := s.playerFormulas[playerID]; ok {
		_, found := formulas[formulaID]
		return found
	}
	return false
}

// InitPlayerFormulas 初始化玩家基础丹方（首次使用炼丹系统时调用）
func (s *EnhancedAlchemyService) InitPlayerFormulas(playerID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.playerFormulas[playerID]; !exists {
		s.playerFormulas[playerID] = make(map[int]*model.PlayerFormula)
		now := time.Now()
		for _, f := range s.baseFormulas {
			s.playerFormulas[playerID][f.ID] = &model.PlayerFormula{
				PlayerID:    playerID,
				FormulaID:   f.ID,
				Name:        f.Name,
				Discovered:  true,
				DiscoveredAt: now,
				CraftCount:  0,
			}
		}
	}
}

// GetAvailableFormulasForResearch 获取玩家可研究的丹方（未研究且满足条件）
func (s *EnhancedAlchemyService) GetAvailableFormulasForResearch(playerID uint64, alchemyLevel int, realmID int) []*model.Formula {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerKnown := s.playerFormulas[playerID]
	if playerKnown == nil {
		playerKnown = make(map[int]*model.PlayerFormula)
	}

	result := make([]*model.Formula, 0)
	for _, f := range s.allFormulas {
		// Skip if already known
		if _, known := playerKnown[f.ID]; known {
			continue
		}
		// Check level and realm requirements
		if alchemyLevel >= f.MinLevel && realmID >= f.RealmRequired {
			result = append(result, f)
		}
	}
	return result
}

// GetResearchAttempts 获取玩家今日研究尝试信息
func (s *EnhancedAlchemyService) GetResearchAttempts(playerID uint64) *model.ResearchAttempt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	attempt, exists := s.researchAttempts[playerID]
	if !exists {
		return &model.ResearchAttempt{
			PlayerID:   playerID,
			DailyCount: 0,
			LastReset:  time.Now(),
		}
	}
	return attempt
}

// AttemptResearch 尝试研究丹方
// playerID: 玩家ID
// formulaID: 要研究的丹方ID
// useStones: 是否使用灵石（增加额外尝试次数）
// luck: 玩家气运值
// alchemyLevel: 玩家炼丹等级
func (s *EnhancedAlchemyService) AttemptResearch(playerID uint64, formulaID int, useStones bool, luck int64, alchemyLevel int) *model.ResearchRecord {
	formula, ok := s.GetFormulaByID(formulaID)
	if !ok {
		return &model.ResearchRecord{
			PlayerID: playerID, FormulaID: formulaID, FormulaName: "未知",
			Success: false,
		}
	}

	// Check if already researched
	if s.HasPlayerResearchedFormula(playerID, formulaID) {
		return &model.ResearchRecord{
			PlayerID: playerID, FormulaID: formulaID, FormulaName: formula.Name,
			Success: false,
		}
	}

	s.mu.Lock()
	// Check research attempts
	attempt, exists := s.researchAttempts[playerID]
	now := time.Now()
	if !exists {
		attempt = &model.ResearchAttempt{
			PlayerID:   playerID,
			DailyCount: 0,
			LastReset:  now,
		}
		s.researchAttempts[playerID] = attempt
	}

	// Reset daily count if last reset was before today
	if attempt.LastReset.Before(truncateToDay(now)) {
		attempt.DailyCount = 0
		attempt.LastReset = now
	}

	// Check free attempts
	maxFree := 3
	if useStones {
		maxFree = 10 // spending stones allows more attempts
	}

	if attempt.DailyCount >= maxFree {
		s.mu.Unlock()
		return &model.ResearchRecord{
			PlayerID: playerID, FormulaID: formulaID, FormulaName: formula.Name,
			Success: false,
		}
	}

	attempt.DailyCount++

	// Calculate success rate
	// Base: 1 - researchDifficulty (e.g., difficulty 0.3 = 70% base)
	// Level bonus: alchemyLevel * 0.03 (each level +3%)
	// Luck bonus: luck * 0.0005 (each luck point +0.05%)
	baseRate := 1.0 - formula.ResearchDifficulty
	levelBonus := float64(alchemyLevel) * 0.03
	luckBonus := float64(luck) * 0.0005
	successRate := baseRate + levelBonus + luckBonus
	if successRate > 0.95 {
		successRate = 0.95
	}
	if successRate < 0.05 {
		successRate = 0.05
	}

	s.rngMu.Lock()
	roll := s.rng.Float64()
	s.rngMu.Unlock()
	success := roll < successRate

	record := &model.ResearchRecord{
		PlayerID:    playerID,
		FormulaID:   formulaID,
		FormulaName: formula.Name,
		Success:     success,
		Timestamp:   now,
	}
	s.researchLog = append(s.researchLog, record)

	if success {
		// Add to player's known formulas
		if s.playerFormulas[playerID] == nil {
			s.playerFormulas[playerID] = make(map[int]*model.PlayerFormula)
		}
		s.playerFormulas[playerID][formulaID] = &model.PlayerFormula{
			PlayerID:    playerID,
			FormulaID:   formulaID,
			Name:        formula.Name,
			Discovered:  true,
			DiscoveredAt: now,
			CraftCount:  0,
		}
	}
	s.mu.Unlock()

	return record
}

// ==================== 炼丹小游戏系统 ====================

// StartCraftSession 开始炼丹小游戏会话
func (s *EnhancedAlchemyService) StartCraftSession(playerID uint64, formulaID int, alchemyLevel int) (*model.AlchemySession, error) {
	formula, ok := s.GetFormulaByID(formulaID)
	if !ok {
		return nil, fmt.Errorf("丹方不存在: %d", formulaID)
	}

	// Check player has furnace
	furnace := s.getOrCreateFurnace(playerID)
	if furnace.Durability <= 0 {
		return nil, fmt.Errorf("丹炉耐久度不足，请修复丹炉")
	}

	sessionKey := fmt.Sprintf("%d:%d", playerID, formulaID)

	s.mu.Lock()
	defer s.mu.Unlock()

	// If there's an active session, return it
	if existing, ok := s.sessions[sessionKey]; ok && !existing.Completed {
		return existing, nil
	}

	// Calculate base quality from alchemy level
	baseQuality := 0
	switch {
	case alchemyLevel >= 50:
		baseQuality = 4 // premium
	case alchemyLevel >= 30:
		baseQuality = 3 // superior
	case alchemyLevel >= 15:
		baseQuality = 2 // good
	case alchemyLevel >= 5:
		baseQuality = 1 // common
	default:
		baseQuality = 0 // junk floor
	}

	// Quality floor from furnace
	qualityFloor := model.FurnaceQualityFloor[furnace.Quality]
	if baseQuality < qualityFloor {
		baseQuality = qualityFloor
	}

	session := &model.AlchemySession{
		PlayerID:    fmt.Sprintf("%d", playerID),
		FormulaID:   formulaID,
		FormulaName: formula.Name,
		FurnaceID:   furnace.ID,
		StartTime:   time.Now(),
		HeatZone:    1, // start at medium
		HeatTimer:   0.5,
		Phase:       "heating",
		Ingredients: make([]string, 0),
		Score:       50, // base score
		BaseQuality: baseQuality,
		Toxicity:    model.QualityToxicity(formula.BaseQuality),
		Success:     false,
		Completed:   false,
	}

	s.sessions[sessionKey] = session
	return session, nil
}

// SetHeatZone 设置火候区域（小游戏操作）
func (s *EnhancedAlchemyService) SetHeatZone(playerID uint64, formulaID int, zone int) (*model.AlchemySession, error) {
	if zone < 0 || zone > 2 {
		return nil, fmt.Errorf("无效的火候区域: %d (0=低火, 1=中火, 2=高火)", zone)
	}

	sessionKey := fmt.Sprintf("%d:%d", playerID, formulaID)
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionKey]
	if !ok || session.Completed {
		return nil, fmt.Errorf("没有进行中的炼丹会话")
	}

	if session.Phase != "heating" {
		return nil, fmt.Errorf("当前阶段不能调整火候")
	}

	prevZone := session.HeatZone

	// Score adjustment based on zone change timing
	// Moving to correct zone gives points; wrong zone or too frequent changes penalize
	if zone == prevZone {
		// No change - slight penalty (wasting time)
		session.Score = max(0, session.Score-2)
	} else {
		// Zone change - reward good judgment
		// The "ideal" zone depends on the phase progress
		progress := session.HeatTimer
		var idealZone int
		switch {
		case progress < 0.33:
			idealZone = 0 // low heat for early stage
		case progress < 0.66:
			idealZone = 1 // medium heat for mid stage
		default:
			idealZone = 2 // high heat for late stage
		}

		if zone == idealZone {
			session.Score = min(100, session.Score+10)
		} else {
			session.Score = max(0, session.Score-3)
		}
	}

	session.HeatZone = zone
	// Advance timer based on zone
	switch zone {
	case 0:
		session.HeatTimer -= 0.05
		if session.HeatTimer < 0 {
			session.HeatTimer = 0
		}
	case 1:
		session.HeatTimer += 0.08
		if session.HeatTimer > 1.0 {
			session.HeatTimer = 1.0
		}
	case 2:
		session.HeatTimer += 0.15
		if session.HeatTimer > 1.0 {
			session.HeatTimer = 1.0
		}
	}

	// Auto-advance phase when timer is near complete
	if session.HeatTimer >= 0.95 && session.Phase == "heating" {
		session.Phase = "adding"
	}

	return session, nil
}

// AddMaterial 添加材料（小游戏操作）
func (s *EnhancedAlchemyService) AddMaterial(playerID uint64, formulaID int, materialID string) (*model.AlchemySession, error) {
	sessionKey := fmt.Sprintf("%d:%d", playerID, formulaID)
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionKey]
	if !ok || session.Completed {
		return nil, fmt.Errorf("没有进行中的炼丹会话")
	}

	if session.Phase != "adding" && session.Phase != "heating" {
		return nil, fmt.Errorf("当前阶段不能添加材料")
	}

	// Check if material already added
	for _, m := range session.Ingredients {
		if m == materialID {
			return nil, fmt.Errorf("材料 %s 已添加", materialID)
		}
	}

	// Timing bonus: materials added at the right heat zone get bonus
	progress := session.HeatTimer
	var idealZone int
	switch {
	case progress < 0.33:
		idealZone = 0
	case progress < 0.66:
		idealZone = 1
	default:
		idealZone = 2
	}

	// Score for adding material
	if session.HeatZone == idealZone {
		session.Score = min(100, session.Score+15) // Perfect timing!
	} else {
		session.Score = max(0, session.Score+5) // Added but not optimal
	}

	session.Ingredients = append(session.Ingredients, materialID)

	// Auto-advance to condensing when all ingredients are in
	formula, _ := s.GetFormulaByID(session.FormulaID)
	if formula != nil && len(session.Ingredients) >= len(formula.Materials) {
		session.Phase = "condensing"
	}

	return session, nil
}

// CompleteCraft 完成炼丹并结算
func (s *EnhancedAlchemyService) CompleteCraft(playerID uint64, formulaID int, luck int64, alchemyLevel int) *model.CraftResultEnhanced {
	sessionKey := fmt.Sprintf("%d:%d", playerID, formulaID)

	s.mu.Lock()
	session, ok := s.sessions[sessionKey]
	if !ok || session.Completed {
		s.mu.Unlock()
		return &model.CraftResultEnhanced{
			Success: false, Message: "没有进行中的炼丹会话",
		}
	}

	session.Completed = true
	score := session.Score

	// Furnace bonus
	furnace := s.getOrCreateFurnace(playerID)
	furnaceBonus := model.FurnaceQualityBonus[furnace.Quality]
	score += furnaceBonus

	// Consume furnace durability
	furnace.Durability--
	if furnace.Durability < 0 {
		furnace.Durability = 0
	}

	s.mu.Unlock()

	// ---- Calculate final quality ----
	finalQuality := session.BaseQuality

	// Score-based quality adjustment
	switch {
	case score >= 95:
		finalQuality += 2 // excellent performance
	case score >= 80:
		finalQuality += 1 // good performance
	case score >= 60:
		// no change
	case score >= 40:
		finalQuality -= 0 // slight penalty if score is low... actually no penalty
	default:
		finalQuality -= 1 // poor performance
	}

	// Cap at immortal (5)
	if finalQuality > 5 {
		finalQuality = 5
	}
	if finalQuality < 0 {
		finalQuality = 0
	}

	// Critical success: 5% chance of +1 quality tier
	s.rngMu.Lock()
	critRoll := s.rng.Float64()
	s.rngMu.Unlock()

	qualityUp := false
	if critRoll < 0.05 && finalQuality < 5 {
		finalQuality++
		qualityUp = true
	}

	if finalQuality > 5 {
		finalQuality = 5
	}

	// Determine success
	success := finalQuality > 0

	// Calculate toxicity
	toxicity := model.QualityToxicity(finalQuality)
	if !success {
		toxicity = 2 // failed craft still creates minimal toxicity
	}

	// Update player toxicity
	s.mu.Lock()
	playerTox := s.getOrCreateToxicity(playerID)
	playerTox.Value += toxicity
	if playerTox.Value > 100 {
		playerTox.Value = 100
	}
	playerTox.UpdatedAt = time.Now()
	currentTox := playerTox.Value
	s.mu.Unlock()

	// Calculate exp gained
	expGained := int64(0)
	alchemyExp := int64(0)
	if success {
		expGained = int64(float64(100) * model.QualityMultiplier(finalQuality))
		alchemyExp = int64(float64(10) * model.QualityMultiplier(finalQuality))
	} else {
		alchemyExp = int64(5)
	}

	qualityName := model.QualityNames[model.Quality(finalQuality)]

	// Update craft count
	s.mu.Lock()
	if formulas, ok := s.playerFormulas[playerID]; ok {
		if pf, found := formulas[formulaID]; found {
			pf.CraftCount++
		}
	}
	s.mu.Unlock()

	return &model.CraftResultEnhanced{
		Success:       success,
		Quality:       finalQuality,
		QualityName:   qualityName,
		PillID:        fmt.Sprintf("enh_pill_%d_%d_%d", formulaID, finalQuality, time.Now().Unix()),
		PillName:      session.FormulaName,
		Score:         score,
		Toxicity:      toxicity,
		TotalToxicity: currentTox,
		ExpGained:     expGained,
		AlchemyExp:    alchemyExp,
		QualityUp:     qualityUp,
		DurabilityUsed: 1,
		Message:       fmt.Sprintf("炼丹%s！品质：%s，评分：%d", map[bool]string{true: "成功", false: "失败"}[success], qualityName, score),
	}
}

// GetActiveSession 获取玩家当前活跃的炼丹会话
func (s *EnhancedAlchemyService) GetActiveSession(playerID uint64, formulaID int) *model.AlchemySession {
	sessionKey := fmt.Sprintf("%d:%d", playerID, formulaID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionKey]
	if !ok || session.Completed {
		return nil
	}
	return session
}

// ==================== 丹毒系统 ====================

// GetPlayerToxicity 获取玩家当前丹毒值
func (s *EnhancedAlchemyService) GetPlayerToxicity(playerID uint64) *model.PlayerToxicity {
	s.mu.Lock()
	defer s.mu.Unlock()
	tox := s.getOrCreateToxicity(playerID)
	// Apply natural decay based on time passed
	s.applyToxicityDecay(tox)
	return tox
}

// ApplyToxicityDecay 应用丹毒自然衰减
func (s *EnhancedAlchemyService) ApplyToxicityDecay(playerID uint64, hoursOnline float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tox := s.getOrCreateToxicity(playerID)

	if tox.Value <= 0 {
		return
	}

	// Online decay: -10 per hour, offline: -5 per hour
	decayPerHour := 5.0
	if hoursOnline > 0 {
		// Mix of online and offline
		decayPerHour = 5.0 + hoursOnline*5.0
	}

	decay := int(decayPerHour * (1.0 / 60.0)) // per minute-ish
	if decay < 1 {
		decay = 1
	}

	tox.Value -= decay
	if tox.Value < 0 {
		tox.Value = 0
	}
	tox.UpdatedAt = time.Now()
}

// applyToxicityDecay 内部应用衰减（必须有写锁）
func (s *EnhancedAlchemyService) applyToxicityDecay(tox *model.PlayerToxicity) {
	if tox.Value <= 0 {
		tox.Value = 0
		return
	}

	elapsed := time.Since(tox.UpdatedAt)
	hoursElapsed := elapsed.Hours()
	if hoursElapsed < 0.1 { // less than 6 minutes, skip
		return
	}

	// Offline decay: -5 per hour
	decay := int(hoursElapsed * 5)
	if decay < 1 {
		decay = 1
	}

	tox.Value -= decay
	if tox.Value < 0 {
		tox.Value = 0
	}
}

// UseDetox 使用解毒丹药
func (s *EnhancedAlchemyService) UseDetox(playerID uint64) *model.DetoxResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	tox := s.getOrCreateToxicity(playerID)

	if tox.Value <= 0 {
		return &model.DetoxResult{
			Success: true, ToxicityReduced: 0, CurrentToxicity: 0,
			Message: "丹毒已清零，无需解毒",
		}
	}

	// Detox pill removes 30 toxicity
	reduced := 30
	if reduced > tox.Value {
		reduced = tox.Value
	}

	tox.Value -= reduced
	tox.UpdatedAt = time.Now()

	return &model.DetoxResult{
		Success:         true,
		ToxicityReduced: reduced,
		CurrentToxicity: tox.Value,
		Message:         fmt.Sprintf("解毒成功！丹毒-%d，当前丹毒值：%d", reduced, tox.Value),
	}
}

// GetToxicityEffect 获取当前丹毒负面效果
func (s *EnhancedAlchemyService) GetToxicityEffect(playerID uint64) map[string]interface{} {
	s.mu.RLock()
	tox, exists := s.playerToxicity[playerID]
	s.mu.RUnlock()

	value := 0
	if exists {
		// Apply natural decay even in read
		s.mu.Lock()
		s.applyToxicityDecay(tox)
		value = tox.Value
		s.mu.Unlock()
	}

	effects := map[string]interface{}{
		"toxicity":     value,
		"cultivation_penalty": 0.0,
		"breakthrough_penalty": 0.0,
		"pill_effect_reduction": 0.0,
		"hp_stopped":   false,
		"warning":      "无异常",
	}

	switch {
	case value > 90:
		effects["cultivation_penalty"] = 0.0
		effects["breakthrough_penalty"] = 0.0
		effects["pill_effect_reduction"] = 0.0
		effects["hp_stopped"] = true
		effects["warning"] = "丹毒已入骨髓！HP恢复停止，需立即解毒！"
	case value > 75:
		effects["breakthrough_penalty"] = 0.15
		effects["pill_effect_reduction"] = 0.5
		effects["warning"] = "丹毒严重！突破率-15%，丹药效果减半"
	case value > 50:
		effects["cultivation_penalty"] = 0.10
		effects["warning"] = "丹毒积累！修炼效率-10%"
	default:
		// no significant effect below 50
	}

	return effects
}

// ==================== 丹炉系统 ====================

// GetFurnace 获取玩家丹炉信息
func (s *EnhancedAlchemyService) GetFurnace(playerID uint64) *model.PlayerFurnace {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getOrCreateFurnace(playerID)
}

// RepairFurnace 修复丹炉耐久度
func (s *EnhancedAlchemyService) RepairFurnace(playerID uint64) *model.PlayerFurnace {
	s.mu.Lock()
	defer s.mu.Unlock()
	furnace := s.getOrCreateFurnace(playerID)
	furnace.Durability = furnace.MaxDurability
	return furnace
}

// UpgradeFurnace 升级丹炉
func (s *EnhancedAlchemyService) UpgradeFurnace(playerID uint64, alchemyLevel int, hasRareMaterial bool) *model.FurnaceUpgradeResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	furnace := s.getOrCreateFurnace(playerID)

	if furnace.Quality >= model.FurnaceImmortal {
		return &model.FurnaceUpgradeResult{
			Success:       false,
			OldQuality:    furnace.Quality,
			NewQuality:    furnace.Quality,
			OldQualityName: furnace.Quality.String(),
			NewQualityName: furnace.Quality.String(),
			Message:       "已达最高品质，无法继续升级",
		}
	}

	// Requirements:
	// Bronze -> Silver: alchemy level >= 10
	// Silver -> Gold: alchemy level >= 25, rare material
	// Gold -> Immortal: alchemy level >= 45, rare material
	var requiredLevel int
	requiresRare := false
	successChance := 0.0

	switch furnace.Quality {
	case model.FurnaceBronze:
		requiredLevel = 10
		successChance = 0.9
	case model.FurnaceSilver:
		requiredLevel = 25
		requiresRare = true
		successChance = 0.6
	case model.FurnaceGold:
		requiredLevel = 45
		requiresRare = true
		successChance = 0.3
	default:
		return &model.FurnaceUpgradeResult{
			Success: false, Message: "无法升级当前品质的丹炉",
		}
	}

	if alchemyLevel < requiredLevel {
		return &model.FurnaceUpgradeResult{
			Success:       false,
			OldQuality:    furnace.Quality,
			NewQuality:    furnace.Quality,
			OldQualityName: furnace.Quality.String(),
			NewQualityName: furnace.Quality.String(),
			Message:       fmt.Sprintf("炼丹等级不足，需要Lv.%d", requiredLevel),
		}
	}

	if requiresRare && !hasRareMaterial {
		return &model.FurnaceUpgradeResult{
			Success:       false,
			OldQuality:    furnace.Quality,
			NewQuality:    furnace.Quality,
			OldQualityName: furnace.Quality.String(),
			NewQualityName: furnace.Quality.String(),
			Message:       "需要稀有材料（如Boss掉落物）才能升级",
		}
	}

	// Roll for success
	s.rngMu.Lock()
	roll := s.rng.Float64()
	s.rngMu.Unlock()

	if roll >= successChance {
		return &model.FurnaceUpgradeResult{
			Success:        false,
			OldQuality:     furnace.Quality,
			NewQuality:     furnace.Quality,
			OldQualityName: furnace.Quality.String(),
			NewQualityName: furnace.Quality.String(),
			Message:        "丹炉升级失败！材料已消耗",
		}
	}

	oldQuality := furnace.Quality
	oldName := furnace.Quality.String()
	furnace.Quality++
	newName := furnace.Quality.String()
	furnace.MaxDurability += 50
	furnace.Durability = furnace.MaxDurability

	return &model.FurnaceUpgradeResult{
		Success:        true,
		OldQuality:     oldQuality,
		NewQuality:     furnace.Quality,
		OldQualityName: oldName,
		NewQualityName: newName,
		Message:        fmt.Sprintf("丹炉升级成功！%s → %s", oldName, newName),
	}
}

// ==================== 内部辅助方法 ====================

// getOrCreateToxicity 获取或创建玩家丹毒（需持有写锁）
func (s *EnhancedAlchemyService) getOrCreateToxicity(playerID uint64) *model.PlayerToxicity {
	tox, exists := s.playerToxicity[playerID]
	if !exists {
		tox = &model.PlayerToxicity{
			PlayerID:  playerID,
			Value:     0,
			UpdatedAt: time.Now(),
		}
		s.playerToxicity[playerID] = tox
	}
	return tox
}

// getOrCreateFurnace 获取或创建玩家丹炉（需持有写锁）
func (s *EnhancedAlchemyService) getOrCreateFurnace(playerID uint64) *model.PlayerFurnace {
	furnace, exists := s.playerFurnaces[playerID]
	if !exists {
		furnace = model.NewPlayerFurnace(playerID)
		s.playerFurnaces[playerID] = furnace
	}
	return furnace
}

// truncateToDay 截断到当天0点
func truncateToDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// max helper
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min helper
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
