// Package service 灵鱼垂钓系统
//
// 修仙者可在各灵池/仙湖垂钓不同品级的灵鱼仙鱼
// 支持开始垂钓→抛竿→等待→收杆的完整流程
// 张力控制小游戏提升捕获成功率
package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"cultivation-game/services/world/internal/model"
)

// FishingService 钓鱼业务逻辑
type FishingService struct {
	mu        sync.RWMutex
	spots     map[string]*model.FishingSpot   // key: spotID
	fishTypes map[int64]*model.FishType        // key: fishID
	sessions  map[int64]*model.FishingSession  // key: playerID
	playerDB  map[int64]*model.PlayerFishing   // key: playerID (内存存储，生产应替换为 DB)
	records   []model.FishingRecord            // 钓鱼记录
	recordsMu sync.Mutex
}

// NewFishingService 创建 FishingService，从 JSON 文件加载数据
func NewFishingService(dataPath string) (*FishingService, error) {
	svc := &FishingService{
		spots:     make(map[string]*model.FishingSpot),
		fishTypes: make(map[int64]*model.FishType),
		sessions:  make(map[int64]*model.FishingSession),
		playerDB:  make(map[int64]*model.PlayerFishing),
		records:   make([]model.FishingRecord, 0),
	}

	if err := svc.loadData(dataPath); err != nil {
		return nil, fmt.Errorf("加载钓鱼数据失败: %w", err)
	}

	return svc, nil
}

// loadData 从 JSON 文件加载钓鱼数据
func (s *FishingService) loadData(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取数据文件失败: %w", err)
	}

	var fishingData model.FishingSpotsData
	if err := json.Unmarshal(data, &fishingData); err != nil {
		return fmt.Errorf("解析数据文件失败: %w", err)
	}

	for i := range fishingData.Spots {
		s.spots[fishingData.Spots[i].ID] = &fishingData.Spots[i]
	}

	for i := range fishingData.FishTypes {
		s.fishTypes[fishingData.FishTypes[i].ID] = &fishingData.FishTypes[i]
	}

	return nil
}

// GetSpots 获取所有钓鱼点
func (s *FishingService) GetSpots() []*model.FishingSpot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*model.FishingSpot, 0, len(s.spots))
	for _, spot := range s.spots {
		result = append(result, spot)
	}
	return result
}

// GetSpotByID 根据ID获取钓鱼点
func (s *FishingService) GetSpotByID(spotID string) *model.FishingSpot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.spots[spotID]
}

// GetFishType 根据ID获取鱼类型
func (s *FishingService) GetFishType(fishID int64) *model.FishType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fishTypes[fishID]
}

// GetFishBySpot 获取某个钓鱼点的所有可钓鱼类（含详细信息）
func (s *FishingService) GetFishBySpot(spotID string) []*model.FishType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	spot, ok := s.spots[spotID]
	if !ok {
		return nil
	}

	result := make([]*model.FishType, 0, len(spot.FishIDs))
	for _, fishID := range spot.FishIDs {
		if fish, ok := s.fishTypes[fishID]; ok {
			result = append(result, fish)
		}
	}
	return result
}

// getAllFishTypes 返回所有鱼类（内部使用，已持锁）
func (s *FishingService) getAllFishTypes() []*model.FishType {
	result := make([]*model.FishType, 0, len(s.fishTypes))
	for _, fish := range s.fishTypes {
		result = append(result, fish)
	}
	return result
}

// StartFishing 开始钓鱼会话
// POST /api/v1/fishing/start
func (s *FishingService) StartFishing(playerID int64, spotID string) (*model.FishingSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查钓鱼点是否存在
	spot, ok := s.spots[spotID]
	if !ok {
		return nil, fmt.Errorf("钓鱼点不存在")
	}

	// 检查玩家是否已有进行中的钓鱼会话
	if _, ok := s.sessions[playerID]; ok {
		return nil, fmt.Errorf("你正在进行一次垂钓，请先完成当前垂钓")
	}

	// 确保玩家有钓鱼数据
	playerFish, ok := s.playerDB[playerID]
	if !ok {
		playerFish = &model.PlayerFishing{
			PlayerID:         playerID,
			FishingSkillLevel: 1,
			FishingExp:       0,
			TotalCaught:      0,
			BaitCount:        10,
		}
		s.playerDB[playerID] = playerFish
	}

	// 检查鱼饵
	if playerFish.BaitCount <= 0 {
		return nil, fmt.Errorf("鱼饵不足，请前往仙坊购买")
	}

	now := time.Now().Unix()
	session := &model.FishingSession{
		PlayerID:  playerID,
		SpotID:    spotID,
		SpotName:  spot.Name,
		Phase:     "casting",
		StartedAt: now,
	}

	s.sessions[playerID] = session
	return session, nil
}

// CastLine 抛竿 - 进入等待咬钩阶段
// POST /api/v1/fishing/cast
func (s *FishingService) CastLine(playerID int64) (*model.FishingSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[playerID]
	if !ok {
		return nil, fmt.Errorf("还没有开始钓鱼，请先开始垂钓")
	}

	if session.Phase != "casting" {
		return nil, fmt.Errorf("当前状态不可抛竿")
	}

	// 选取一条鱼（根据玩家技能和钓鱼点鱼池）
	fish := s.selectFish(playerID, session.SpotID)
	if fish == nil {
		return nil, fmt.Errorf("该钓鱼点暂无鱼可钓")
	}

	// 计算等待时间（受钓鱼技能影响）
	playerFish := s.playerDB[playerID]
	waitTime := s.calculateBiteTime(playerFish.FishingSkillLevel, fish.Rarity)

	// 生成鱼的实际重量（基础重量上下浮动）
	weightVariation := 0.7 + rand.Float64()*0.6 // 0.7~1.3
	weight := fish.BaseWeight * weightVariation
	weight = math.Round(weight*100) / 100

	biteAt := time.Now().Unix() + int64(waitTime.Seconds())

	session.Phase = "waiting"
	session.Fish = fish
	session.Weight = weight
	session.BiteAt = biteAt
	session.Tension = 50.0

	return session, nil
}

// selectFish 根据玩家技能和钓鱼点选取可钓的鱼
func (s *FishingService) selectFish(playerID int64, spotID string) *model.FishType {
	spot, ok := s.spots[spotID]
	if !ok {
		return nil
	}

	playerFish, ok := s.playerDB[playerID]
	if !ok {
		return nil
	}

	skillLevel := playerFish.FishingSkillLevel

	// 收集该钓鱼点所有可钓的鱼
	var availableFish []*model.FishType
	for _, fishID := range spot.FishIDs {
		fish, ok := s.fishTypes[fishID]
		if !ok {
			continue
		}

		// 检查玩家修为是否满足（简化：技能等级 >= 鱼类需求等级-2）
		requiredLevel := fish.MinRealm / 5
		if requiredLevel < 1 {
			requiredLevel = 1
		}
		if skillLevel >= requiredLevel-2 {
			availableFish = append(availableFish, fish)
		}
	}

	if len(availableFish) == 0 {
		return nil
	}

	// 根据稀有度概率抽取
	// 稀有度越高，概率越低
	// 技能等级提升可提高高稀有度鱼类的概率
	totalWeight := 0.0
	rarityProb := make([]float64, len(availableFish))
	for i, fish := range availableFish {
		prob := float64(6 - fish.Rarity) // rarity 1 → 5, rarity 5 → 1
		// 技能加成：每5级提升一点稀有鱼概率
		if fish.Rarity > 1 {
			skillBonus := float64(skillLevel) / 50.0
			prob += skillBonus
		}
		if prob < 0.5 {
			prob = 0.5
		}
		rarityProb[i] = prob
		totalWeight += prob
	}

	r := rand.Float64() * totalWeight
	cumulative := 0.0
	for i, prob := range rarityProb {
		cumulative += prob
		if r <= cumulative {
			return availableFish[i]
		}
	}

	return availableFish[len(availableFish)-1]
}

// calculateBiteTime 计算咬钩等待时间
func (s *FishingService) calculateBiteTime(skillLevel int, rarity int) time.Duration {
	// 基础等待时间：5~15秒
	baseSec := 5.0 + rand.Float64()*10.0

	// 稀有度越高，等待时间越长
	rarityFactor := 1.0 + float64(rarity)*0.3

	// 技能减少等待时间
	skillFactor := 1.0 - float64(skillLevel-1)*0.03
	if skillFactor < 0.4 {
		skillFactor = 0.4
	}

	totalSec := baseSec * rarityFactor * skillFactor
	return time.Duration(totalSec * float64(time.Second))
}

// CheckBite 检查是否有鱼咬钩（前端轮询）
func (s *FishingService) CheckBite(playerID int64) (*model.FishingSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[playerID]
	if !ok {
		return nil, fmt.Errorf("没有进行中的钓鱼")
	}

	if session.Phase != "waiting" {
		return nil, fmt.Errorf("当前状态不可检查咬钩")
	}

	now := time.Now().Unix()
	if now >= session.BiteAt {
		session.Phase = "hooking"
		return session, nil
	}

	// 返回剩余等待时间
	return session, nil
}

// HookFish 收杆 - 尝试捕获鱼
// POST /api/v1/fishing/hook
func (s *FishingService) HookFish(playerID int64) (*model.FishingResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[playerID]
	if !ok {
		return nil, fmt.Errorf("没有进行中的钓鱼")
	}

	if session.Phase != "hooking" && session.Phase != "reeling" {
		return nil, fmt.Errorf("当前状态不可收杆")
	}

	playerFish, ok := s.playerDB[playerID]
	if !ok {
		return nil, fmt.Errorf("玩家数据异常")
	}

	// 扣除鱼饵
	playerFish.BaitCount--

	if session.Fish == nil {
		delete(s.sessions, playerID)
		return nil, fmt.Errorf("鱼已经跑掉了")
	}

	fish := session.Fish
	weight := session.Weight

	// 计算捕获成功率（基于张力、技能和鱼稀有度）
	successRate := s.calculateCatchRate(playerFish.FishingSkillLevel, fish.Rarity, session.Tension)
	caught := rand.Float64() < successRate

	var result model.FishingResult

	if caught {
		// 计算经验
		expGained := fish.ExpReward + (playerFish.FishingSkillLevel * 2)

		// 更新玩家数据
		playerFish.TotalCaught++
		playerFish.FishingExp += expGained

		// 检查是否是最佳捕获
		isBest := false
		if weight > playerFish.BestCatchWeight {
			playerFish.BestCatchWeight = weight
			fishID := fish.ID
			playerFish.BestCatchID = &fishID
			isBest = true
		}

		// 检查是否升级
		expToNext := expForLevel(playerFish.FishingSkillLevel)
		var leveledUp bool
		for playerFish.FishingExp >= expToNext {
			playerFish.FishingExp -= expToNext
			playerFish.FishingSkillLevel++
			expToNext = expForLevel(playerFish.FishingSkillLevel)
			leveledUp = true
		}

		msg := fmt.Sprintf("收获%s！重%.1f斤", fish.Name, weight)
		if leveledUp {
			msg += fmt.Sprintf(" 钓鱼技能提升至Lv.%d！", playerFish.FishingSkillLevel)
		}

		// 保存记录
		s.addRecord(playerID, fish.ID, 0, weight, expGained)

		spotIDStr := session.SpotID
		_ = spotIDStr

		result = model.FishingResult{
			Fish:        *fish,
			Weight:      weight,
			ExpGained:   expGained,
			IsBestCatch: isBest,
			Message:     msg,
		}
	} else {
		result = model.FishingResult{
			Message: fmt.Sprintf("%s挣脱了鱼钩...", fish.Name),
		}
	}

	delete(s.sessions, playerID)
	return &result, nil
}

// calculateCatchRate 计算捕获成功率
func (s *FishingService) calculateCatchRate(skillLevel int, rarity int, tension float64) float64 {
	// 基础成功率
	baseRate := 0.7

	// 稀有度降低成功率
	rarityPenalty := float64(rarity-1) * 0.12

	// 技能提升成功率
	skillBonus := float64(skillLevel-1) * 0.03

	// 张力影响：50为最佳，偏离越远成功率越低
	tensionFactor := 1.0 - math.Abs(tension-50.0)/100.0
	if tensionFactor < 0.3 {
		tensionFactor = 0.3
	}

	rate := (baseRate - rarityPenalty + skillBonus) * tensionFactor
	if rate > 0.95 {
		rate = 0.95
	}
	if rate < 0.1 {
		rate = 0.1
	}

	return rate
}

// UpdateTension 更新张力值（收杆小游戏）
// POST /api/v1/fishing/tension
func (s *FishingService) UpdateTension(playerID int64, action string) (*model.FishingSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[playerID]
	if !ok {
		return nil, fmt.Errorf("没有进行中的钓鱼")
	}

	if session.Phase != "hooking" && session.Phase != "reeling" {
		return nil, fmt.Errorf("当前无法调整张力")
	}

	session.Phase = "reeling"

	switch action {
	case "pull":
		session.Tension += 8.0 + rand.Float64()*4.0
	case "release":
		session.Tension -= 6.0 + rand.Float64()*4.0
	case "hold":
		// 保持不动，张力缓慢回归中心
		if session.Tension > 50 {
			session.Tension -= 2.0
		} else if session.Tension < 50 {
			session.Tension += 2.0
		}
	default:
		return nil, fmt.Errorf("无效操作")
	}

	// 限制范围
	if session.Tension < 0 {
		session.Tension = 0
	}
	if session.Tension > 100 {
		session.Tension = 100
	}

	return session, nil
}

// GetFishingInfo 获取玩家钓鱼信息
// GET /api/v1/fishing/info
func (s *FishingService) GetFishingInfo(playerID int64) *model.FishingInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerFish, ok := s.playerDB[playerID]
	if !ok {
		return &model.FishingInfo{
			PlayerID:         playerID,
			FishingSkillLevel: 1,
			FishingExp:       0,
			ExpToNextLevel:   expForLevel(1),
			TotalCaught:      0,
			BaitCount:        10,
			History:          []model.FishingRecord{},
		}
	}

	expToNext := expForLevel(playerFish.FishingSkillLevel)

	var bestCatch *model.BestCatchInfo
	if playerFish.BestCatchID != nil && playerFish.BestCatchWeight > 0 {
		fishName := "未知"
		if fish, ok := s.fishTypes[*playerFish.BestCatchID]; ok {
			fishName = fish.Name
		}
		bestCatch = &model.BestCatchInfo{
			FishName: fishName,
			Weight:   playerFish.BestCatchWeight,
		}
	}

	// 获取最近的钓鱼记录
	history := s.getPlayerRecords(playerID)

	return &model.FishingInfo{
		PlayerID:          playerID,
		FishingSkillLevel:  playerFish.FishingSkillLevel,
		FishingExp:        playerFish.FishingExp,
		ExpToNextLevel:    expToNext,
		TotalCaught:       playerFish.TotalCaught,
		BestCatch:         bestCatch,
		BaitCount:         playerFish.BaitCount,
		History:           history,
	}
}

// UpgradeSkill 升级钓鱼技能（消耗经验）
// POST /api/v1/fishing/upgrade
func (s *FishingService) UpgradeSkill(playerID int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerFish, ok := s.playerDB[playerID]
	if !ok {
		return 0, fmt.Errorf("玩家还没有钓鱼数据")
	}

	expNeeded := expForLevel(playerFish.FishingSkillLevel)
	if playerFish.FishingExp < expNeeded {
		return 0, fmt.Errorf("经验不足，需要 %d 点经验，当前 %d 点", expNeeded, playerFish.FishingExp)
	}

	playerFish.FishingExp -= expNeeded
	playerFish.FishingSkillLevel++

	return playerFish.FishingSkillLevel, nil
}

// BuyBait 购买鱼饵
// POST /api/v1/fishing/buy-bait
func (s *FishingService) BuyBait(playerID int64, amount int) (*model.PlayerFishing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerFish, ok := s.playerDB[playerID]
	if !ok {
		playerFish = &model.PlayerFishing{
			PlayerID:         playerID,
			FishingSkillLevel: 1,
			FishingExp:       0,
			TotalCaught:      0,
			BaitCount:        10,
		}
		s.playerDB[playerID] = playerFish
	}

	if amount <= 0 {
		return nil, fmt.Errorf("购买数量无效")
	}

	// 上限 999 个鱼饵
	maxBait := 999
	if playerFish.BaitCount+amount > maxBait {
		return nil, fmt.Errorf("鱼饵数量已达上限（最多%d个）", maxBait)
	}

	playerFish.BaitCount += amount
	return playerFish, nil
}

// CancelFishing 取消当前钓鱼
func (s *FishingService) CancelFishing(playerID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[playerID]; !ok {
		return fmt.Errorf("没有进行中的钓鱼")
	}

	delete(s.sessions, playerID)
	return nil
}

// addRecord 添加钓鱼记录
func (s *FishingService) addRecord(playerID int64, fishID int64, spotID int64, weight float64, expGained int) {
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()

	s.records = append(s.records, model.FishingRecord{
		ID:        int64(len(s.records) + 1),
		PlayerID:  playerID,
		FishID:    fishID,
		Weight:    weight,
		ExpGained: expGained,
		CaughtAt:  time.Now().Format("2006-01-02 15:04:05"),
	})
}

// getPlayerRecords 获取玩家的钓鱼记录
func (s *FishingService) getPlayerRecords(playerID int64) []model.FishingRecord {
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()

	var result []model.FishingRecord
	for i := len(s.records) - 1; i >= 0; i-- {
		if s.records[i].PlayerID == playerID {
			result = append(result, s.records[i])
			if len(result) >= 20 {
				break
			}
		}
	}
	return result
}

// expForLevel 计算升级所需经验
func expForLevel(level int) int {
	return 50 + level*30 + (level*level)*2
}

// GetFishCollection 获取玩家的鱼类图鉴
func (s *FishingService) GetFishCollection(playerID int64) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	allFish := s.getAllFishTypes()

	// 获取玩家的钓鱼记录以判断哪些鱼已捕获
	s.recordsMu.Lock()
	caughtFish := make(map[int64]bool)
	for _, record := range s.records {
		if record.PlayerID == playerID {
			caughtFish[record.FishID] = true
		}
	}
	s.recordsMu.Unlock()

	type collectionItem struct {
		Fish    *model.FishType `json:"fish"`
		Caught  bool            `json:"caught"`
		Count   int             `json:"count"`
	}

	items := make([]collectionItem, 0, len(allFish))
	for _, fish := range allFish {
		items = append(items, collectionItem{
			Fish:   fish,
			Caught: caughtFish[fish.ID],
		})
	}

	return map[string]interface{}{
		"total":   len(allFish),
		"caught":  len(caughtFish),
		"collection": items,
	}
}
