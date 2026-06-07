// Package service 提供世界服务的业务逻辑
package service

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"cultivation-game/services/world/internal/model"
)

// QuestService 任务服务
// 负责任务配置加载、任务进度追踪、条件检查和奖励发放
type QuestService struct {
	mu             sync.RWMutex
	quests         map[string]*model.Quest               // questID -> 任务配置
	playerQuests   map[string]map[string]*model.PlayerQuest // playerID -> questID -> 玩家任务进度
	dailyCompleted map[string]map[string]string          // playerID -> questID -> 完成日期 "2006-01-02"

	// 每日任务系统
	dailyTaskDefs  map[string]*model.DailyTaskDef          // taskID -> 每日任务定义
	dailyProgress  map[string]map[string]*model.DailyTaskProgress // playerID -> taskID -> 今日进度
	activityPoints map[string]*model.ActivityPoints        // playerID -> 今日活跃度
}

// NewQuestService 创建任务服务，从 JSON 加载任务配置
func NewQuestService(questsPath string) (*QuestService, error) {
	quests, err := loadQuests(questsPath)
	if err != nil {
		return nil, fmt.Errorf("加载任务配置失败: %w", err)
	}

	return &QuestService{
		quests:          quests,
		playerQuests:    make(map[string]map[string]*model.PlayerQuest),
		dailyCompleted:  make(map[string]map[string]string),
		dailyTaskDefs:   make(map[string]*model.DailyTaskDef),
		dailyProgress:   make(map[string]map[string]*model.DailyTaskProgress),
		activityPoints:  make(map[string]*model.ActivityPoints),
	}, nil
}

// LoadDailyTasks 从 JSON 文件加载每日任务定义
func (s *QuestService) LoadDailyTasks(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("加载每日任务配置失败: %w", err)
	}
	var taskList []*model.DailyTaskDef
	if err := json.Unmarshal(data, &taskList); err != nil {
		return fmt.Errorf("解析每日任务配置失败: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range taskList {
		s.dailyTaskDefs[t.ID] = t
	}
	return nil
}

// loadQuests 从 JSON 文件加载任务配置
func loadQuests(path string) (map[string]*model.Quest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var questList []*model.Quest
	if err := json.Unmarshal(data, &questList); err != nil {
		return nil, err
	}
	result := make(map[string]*model.Quest, len(questList))
	for _, q := range questList {
		result[q.ID] = q
	}
	return result, nil
}

// GetQuest 获取任务配置
func (s *QuestService) GetQuest(questID string) (*model.Quest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.quests[questID]
	return q, ok
}

// GetAllQuests 获取所有任务配置(可选按类型过滤)
func (s *QuestService) GetAllQuests() []*model.Quest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.Quest, 0, len(s.quests))
	for _, q := range s.quests {
		result = append(result, q)
	}
	return result
}

// GetQuestsByType 按类型获取任务配置
func (s *QuestService) GetQuestsByType(qt model.QuestType) []*model.Quest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*model.Quest
	for _, q := range s.quests {
		if q.Type == qt {
			result = append(result, q)
		}
	}
	return result
}

// GetDailyQuests 获取今日每日任务
func (s *QuestService) GetDailyQuests() []*model.Quest {
	return s.GetQuestsByType(model.QuestDaily)
}

// ============================================================
// 玩家任务操作
// ============================================================

// AcceptQuest 玩家接取任务
// 检查: 前置任务、境界要求、是否已接取/已完成
func (s *QuestService) AcceptQuest(playerID, questID string, playerLevel int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	quest, ok := s.quests[questID]
	if !ok {
		return fmt.Errorf("任务不存在")
	}

	// 检查等级要求
	if playerLevel < quest.LevelRequired {
		return fmt.Errorf("境界不足，需要等级 %d，当前等级 %d", quest.LevelRequired, playerLevel)
	}

	// 获取或初始化该玩家的任务映射
	playerQuestMap, ok := s.playerQuests[playerID]
	if !ok {
		playerQuestMap = make(map[string]*model.PlayerQuest)
		s.playerQuests[playerID] = playerQuestMap
	}

	// 检查是否已接取或已完成
	if existing, ok := playerQuestMap[questID]; ok {
		switch existing.Status {
		case model.QuestInProgress:
			return fmt.Errorf("任务已接取，请先完成任务")
		case model.QuestCompleted:
			return fmt.Errorf("任务已完成，请先提交")
		case model.QuestSubmitted:
			// 每日任务隔天可重新接取
			if quest.Type == model.QuestDaily {
				today := time.Now().Format("2006-01-02")
				if s.dailyCompleted[playerID][questID] == today {
					return fmt.Errorf("今日已完成该每日任务，请明天再来")
				}
				// 隔天了，允许重新接取
				break
			}
			return fmt.Errorf("任务已完成")
		}
	}

	// 检查前置任务
	for _, prereqID := range quest.Prerequisites {
		prereqPQ, ok := playerQuestMap[prereqID]
		if !ok || prereqPQ.Status != model.QuestSubmitted {
			prereqQuest, hasName := s.quests[prereqID]
			name := "未知"
			if hasName {
				name = prereqQuest.Name
			}
			return fmt.Errorf("前置任务 '%s' 未完成，请先完成前置任务", name)
		}
	}

	// 对每日任务: 检查是否已记录完成(今天的已完成)
	if quest.Type == model.QuestDaily {
		today := time.Now().Format("2006-01-02")
		if s.dailyCompleted[playerID] != nil && s.dailyCompleted[playerID][questID] == today {
			return fmt.Errorf("今日已完成该每日任务")
		}
	}

	// 创建进度副本(所有需求初始当前值为0)
	progress := make([]model.QuestRequirement, len(quest.Requirements))
	for i, req := range quest.Requirements {
		progress[i] = model.QuestRequirement{
			Type:     req.Type,
			TargetID: req.TargetID,
			Count:    req.Count,
			Current:  0,
		}
	}

	now := time.Now()
	pq := &model.PlayerQuest{
		PlayerID:   playerID,
		QuestID:    questID,
		Status:     model.QuestInProgress,
		Progress:   progress,
		AcceptedAt: now,
	}

	playerQuestMap[questID] = pq
	return nil
}

// UpdateProgress 根据事件更新玩家所有进行中任务的进度
// 由其他服务(战斗、采集、修炼等)触发
func (s *QuestService) UpdateProgress(playerID string, event model.QuestEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerQuestMap, ok := s.playerQuests[playerID]
	if !ok {
		return
	}

	for _, pq := range playerQuestMap {
		if pq.Status != model.QuestInProgress {
			continue
		}

		updated := false
		for i, req := range pq.Progress {
			if matchRequirement(req, event) {
				pq.Progress[i].Current += event.Count
				if pq.Progress[i].Current > pq.Progress[i].Count {
					pq.Progress[i].Current = pq.Progress[i].Count
				}
				updated = true
			}
		}

		// 检查是否所有需求已满足
		if updated {
			allMet := true
			for _, req := range pq.Progress {
				if req.Current < req.Count {
					allMet = false
					break
				}
			}
			if allMet {
				now := time.Now()
				pq.Status = model.QuestCompleted
				pq.CompletedAt = &now
			}
		}
	}
}

// matchRequirement 判断事件是否匹配某个需求
func matchRequirement(req model.QuestRequirement, event model.QuestEvent) bool {
	// 类型必须匹配
	if req.Type != event.Type {
		return false
	}

	// 目标匹配: 精确匹配或通配符"any"
	if req.TargetID == "any" {
		return true
	}

	// reach_realm 类型: 事件中到达更高境界也算完成需求
	// 例如需求是 qi_refining_3，事件目标 higher 也应匹配
	if req.Type == "reach_realm" && event.TargetID == "higher" {
		return true
	}

	return req.TargetID == event.TargetID
}

// CompleteQuest 提交已完成的任务，发放奖励
// 返回任务配置的奖励列表(调用方负责实际发放)
func (s *QuestService) CompleteQuest(playerID, questID string) ([]model.QuestReward, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerQuestMap, ok := s.playerQuests[playerID]
	if !ok {
		return nil, fmt.Errorf("玩家未接取此任务")
	}

	pq, ok := playerQuestMap[questID]
	if !ok {
		return nil, fmt.Errorf("玩家未接取此任务")
	}

	if pq.Status != model.QuestCompleted {
		return nil, fmt.Errorf("任务条件未满足，无法提交")
	}

	quest, ok := s.quests[questID]
	if !ok {
		return nil, fmt.Errorf("任务配置不存在")
	}

	// 标记为已提交
	now := time.Now()
	pq.Status = model.QuestSubmitted
	pq.CompletedAt = &now

	// 每日任务记录完成日期
	if quest.Type == model.QuestDaily {
		if s.dailyCompleted[playerID] == nil {
			s.dailyCompleted[playerID] = make(map[string]string)
		}
		s.dailyCompleted[playerID][questID] = now.Format("2006-01-02")
	}

	// 返回奖励列表(副本)
	rewards := make([]model.QuestReward, len(quest.Rewards))
	copy(rewards, quest.Rewards)
	return rewards, nil
}

// GetPlayerQuests 获取玩家的所有任务状态
func (s *QuestService) GetPlayerQuests(playerID string) []*model.PlayerQuest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerQuestMap, ok := s.playerQuests[playerID]
	if !ok {
		return nil
	}

	result := make([]*model.PlayerQuest, 0, len(playerQuestMap))
	for _, pq := range playerQuestMap {
		result = append(result, pq)
	}
	return result
}

// GetPlayerQuest 获取玩家的单个任务状态
func (s *QuestService) GetPlayerQuest(playerID, questID string) (*model.PlayerQuest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerQuestMap, ok := s.playerQuests[playerID]
	if !ok {
		return nil, false
	}
	pq, ok := playerQuestMap[questID]
	return pq, ok
}

// GetAvailableQuests 获取玩家可接取的任务列表
// 过滤条件: 等级要求、前置任务、是否已完成(非每日)
func (s *QuestService) GetAvailableQuests(playerID string, playerLevel int) []*model.Quest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerQuestMap := s.playerQuests[playerID]
	today := time.Now().Format("2006-01-02")

	var result []*model.Quest
	for _, q := range s.quests {
		// 等级检查
		if playerLevel < q.LevelRequired {
			continue
		}

		// 检查玩家已有的任务状态
		if playerQuestMap != nil {
			if existing, ok := playerQuestMap[q.ID]; ok {
				// 进行中 / 已完成待提交 的状态不显示在可接取列表
				if existing.Status == model.QuestInProgress || existing.Status == model.QuestCompleted {
					continue
				}
				// 非每日任务已提交 => 不再显示
				if existing.Status == model.QuestSubmitted && q.Type != model.QuestDaily {
					continue
				}
				// 每日任务今日已提交 => 不显示
				if existing.Status == model.QuestSubmitted && q.Type == model.QuestDaily {
					if s.dailyCompleted[playerID] != nil && s.dailyCompleted[playerID][q.ID] == today {
						continue
					}
				}
			}
		}

		// 前置任务检查
		prereqMet := true
		for _, prereqID := range q.Prerequisites {
			if playerQuestMap == nil {
				prereqMet = false
				break
			}
			prereqPQ, ok := playerQuestMap[prereqID]
			if !ok || prereqPQ.Status != model.QuestSubmitted {
				prereqMet = false
				break
			}
		}
		if !prereqMet {
			continue
		}

		result = append(result, q)
	}
	return result
}

// ============================================================
// 每日任务系统
// ============================================================

// todayDate 获取今日日期字符串
func todayDate() string {
	return time.Now().Format("2006-01-02")
}

// GetDailyTaskDefs 获取所有每日任务定义
func (s *QuestService) GetDailyTaskDefs() []*model.DailyTaskDef {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.DailyTaskDef, 0, len(s.dailyTaskDefs))
	for _, t := range s.dailyTaskDefs {
		result = append(result, t)
	}
	return result
}

// GetDailyTaskDef 根据ID获取每日任务定义
func (s *QuestService) GetDailyTaskDef(taskID string) (*model.DailyTaskDef, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.dailyTaskDefs[taskID]
	return t, ok
}

// EnsureDailyTasks 确保玩家今日的每日任务已生成
func (s *QuestService) EnsureDailyTasks(playerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	date := todayDate()

	if s.dailyProgress[playerID] != nil {
		if ap, ok := s.activityPoints[playerID]; ok && ap.Date == date {
			return
		}
	}

	if s.dailyProgress[playerID] == nil {
		s.dailyProgress[playerID] = make(map[string]*model.DailyTaskProgress)
	}

	for _, def := range s.dailyTaskDefs {
		if _, exists := s.dailyProgress[playerID][def.ID]; !exists {
			s.dailyProgress[playerID][def.ID] = &model.DailyTaskProgress{
				PlayerID:      playerID,
				TaskDate:      date,
				TaskID:        def.ID,
				TaskType:      def.Type,
				CurrentCount:  0,
				RequiredCount: def.RequiredCount,
				Status:        model.DailyTaskInProgress,
			}
		}
	}

	if s.activityPoints[playerID] == nil || s.activityPoints[playerID].Date != date {
		s.activityPoints[playerID] = &model.ActivityPoints{
			PlayerID: playerID,
			Date:     date,
		}
	}
}

// GetDailyTasksWithProgress 获取玩家今日所有每日任务及进度
func (s *QuestService) GetDailyTasksWithProgress(playerID string) []*model.DailyTaskWithProgress {
	s.EnsureDailyTasks(playerID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*model.DailyTaskWithProgress, 0, len(s.dailyTaskDefs))
	for _, def := range s.dailyTaskDefs {
		item := &model.DailyTaskWithProgress{Def: def}
		if progress, ok := s.dailyProgress[playerID][def.ID]; ok {
			item.Progress = progress
		}
		result = append(result, item)
	}
	return result
}

// GetDailyTaskProgress 获取玩家指定每日任务的进度
func (s *QuestService) GetDailyTaskProgress(playerID, taskID string) (*model.DailyTaskWithProgress, error) {
	s.EnsureDailyTasks(playerID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	def, ok := s.dailyTaskDefs[taskID]
	if !ok {
		return nil, fmt.Errorf("每日任务不存在: %s", taskID)
	}

	item := &model.DailyTaskWithProgress{Def: def}
	if progress, ok := s.dailyProgress[playerID][taskID]; ok {
		item.Progress = progress
	}
	return item, nil
}

// UpdateDailyTaskProgress 更新玩家每日任务进度
func (s *QuestService) UpdateDailyTaskProgress(playerID, taskType, targetID string, count int) {
	s.EnsureDailyTasks(playerID)

	s.mu.Lock()
	defer s.mu.Unlock()

	date := todayDate()

	for _, progress := range s.dailyProgress[playerID] {
		if progress.Status != model.DailyTaskInProgress {
			continue
		}
		if progress.TaskType != taskType {
			continue
		}
		if progress.TaskDate != date {
			continue
		}

		progress.CurrentCount += count
		if progress.CurrentCount > progress.RequiredCount {
			progress.CurrentCount = progress.RequiredCount
		}
		if progress.CurrentCount >= progress.RequiredCount {
			progress.Status = model.DailyTaskCompleted
			now := time.Now()
			progress.CompletedAt = &now

			if def, ok := s.dailyTaskDefs[progress.TaskID]; ok {
				if s.activityPoints[playerID] == nil || s.activityPoints[playerID].Date != date {
					s.activityPoints[playerID] = &model.ActivityPoints{
						PlayerID: playerID,
						Date:     date,
					}
				}
				s.activityPoints[playerID].TotalPoints += def.ActivityPoints
				if s.activityPoints[playerID].TotalPoints > 100 {
					s.activityPoints[playerID].TotalPoints = 100
				}
			}
		}
	}
}

// ClaimDailyTaskReward 领取每日任务奖励
func (s *QuestService) ClaimDailyTaskReward(playerID, taskID string) ([]model.QuestReward, error) {
	s.EnsureDailyTasks(playerID)

	s.mu.Lock()
	defer s.mu.Unlock()

	progress, ok := s.dailyProgress[playerID][taskID]
	if !ok {
		return nil, fmt.Errorf("任务进度不存在")
	}
	if progress.Status != model.DailyTaskCompleted {
		return nil, fmt.Errorf("任务未完成，无法领取")
	}
	if progress.Status == model.DailyTaskClaimed {
		return nil, fmt.Errorf("奖励已领取")
	}

	def, ok := s.dailyTaskDefs[taskID]
	if !ok {
		return nil, fmt.Errorf("任务配置不存在")
	}

	now := time.Now()
	progress.Status = model.DailyTaskClaimed
	progress.ClaimedAt = &now

	rewards := make([]model.QuestReward, len(def.Rewards))
	copy(rewards, def.Rewards)
	return rewards, nil
}

// GetActivityPoints 获取玩家今日活跃度信息
func (s *QuestService) GetActivityPoints(playerID string) *model.ActivityPoints {
	s.EnsureDailyTasks(playerID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	if ap, ok := s.activityPoints[playerID]; ok {
		cp := *ap
		return &cp
	}
	return &model.ActivityPoints{
		PlayerID: playerID,
		Date:     todayDate(),
	}
}

// GetActivityChests 获取活跃度宝箱状态
func (s *QuestService) GetActivityChests(playerID string) *model.ActivityPoints {
	ap := s.GetActivityPoints(playerID)
	return ap
}

// ClaimActivityChest 领取活跃度宝箱
func (s *QuestService) ClaimActivityChest(playerID string, tier model.ChestTier) ([]model.QuestReward, error) {
	s.EnsureDailyTasks(playerID)

	s.mu.Lock()
	defer s.mu.Unlock()

	ap, ok := s.activityPoints[playerID]
	if !ok {
		return nil, fmt.Errorf("活跃度数据不存在")
	}

	var claimed *bool
	switch tier {
	case model.ChestTier25:
		claimed = &ap.Chest25Claimed
	case model.ChestTier50:
		claimed = &ap.Chest50Claimed
	case model.ChestTier75:
		claimed = &ap.Chest75Claimed
	case model.ChestTier100:
		claimed = &ap.Chest100Claimed
	default:
		return nil, fmt.Errorf("无效的宝箱档位: %d", tier)
	}

	if *claimed {
		return nil, fmt.Errorf("该宝箱已领取")
	}

	if ap.TotalPoints < int(tier) {
		return nil, fmt.Errorf("活跃度不足，需要%d点，当前%d点", tier, ap.TotalPoints)
	}

	*claimed = true

	rewards := getChestRewards(int(tier))
	return rewards, nil
}

// chestRewardsConfig 活跃度宝箱奖励配置
var chestRewardsConfig = map[int][]model.QuestReward{
	25: {
		{Type: "exp", ID: "", Quantity: 500},
		{Type: "money", ID: "", Quantity: 200},
	},
	50: {
		{Type: "exp", ID: "", Quantity: 1000},
		{Type: "money", ID: "", Quantity: 500},
		{Type: "item", ID: "pill_juqi_dan", Quantity: 3},
	},
	75: {
		{Type: "exp", ID: "", Quantity: 2000},
		{Type: "money", ID: "", Quantity: 1000},
		{Type: "item", ID: "pill_peiyuan_dan", Quantity: 2},
	},
	100: {
		{Type: "exp", ID: "", Quantity: 5000},
		{Type: "money", ID: "", Quantity: 2000},
		{Type: "item", ID: "pill_breakthrough_01", Quantity: 1},
		{Type: "reputation", ID: "", Quantity: 100},
	},
}

// getChestRewards 获取指定档位的宝箱奖励
func getChestRewards(tier int) []model.QuestReward {
	if rewards, ok := chestRewardsConfig[tier]; ok {
		result := make([]model.QuestReward, len(rewards))
		copy(result, rewards)
		return result
	}
	return nil
}

// GetChestTiers 获取所有宝箱档位配置
func (s *QuestService) GetChestTiers() []struct {
	Tier    int                 `json:"tier"`
	Points  int                 `json:"points"`
	Rewards []model.QuestReward `json:"rewards"`
} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []struct {
		Tier    int                 `json:"tier"`
		Points  int                 `json:"points"`
		Rewards []model.QuestReward `json:"rewards"`
	}

	for _, tier := range []int{25, 50, 75, 100} {
		rewards := chestRewardsConfig[tier]
		if rewards == nil {
			continue
		}
		rw := make([]model.QuestReward, len(rewards))
		copy(rw, rewards)
		result = append(result, struct {
			Tier    int                 `json:"tier"`
			Points  int                 `json:"points"`
			Rewards []model.QuestReward `json:"rewards"`
		}{
			Tier:    tier,
			Points:  tier,
			Rewards: rw,
		})
	}
	return result
}
