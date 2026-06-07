package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// ActivityService 运营活动服务
type ActivityService struct {
	repo         *mysql.ActivityRepo
	checkinRepo  *mysql.CheckinRepo
	playerSvc    *PlayerService
	inventorySvc *InventoryService
	log          *zap.Logger
}

// NewActivityService 创建 ActivityService
func NewActivityService(
	repo *mysql.ActivityRepo,
	checkinRepo *mysql.CheckinRepo,
	playerSvc *PlayerService,
	inventorySvc *InventoryService,
	log *zap.Logger,
) *ActivityService {
	return &ActivityService{
		repo:         repo,
		checkinRepo:  checkinRepo,
		playerSvc:    playerSvc,
		inventorySvc: inventorySvc,
		log:          log,
	}
}

// ============================================================
// 限时活动 (Limited-Time Events)
// ============================================================

// GetActiveEvents 获取当前活跃活动
func (s *ActivityService) GetActiveEvents() ([]*model.LimitedEvent, error) {
	events, err := s.repo.GetActiveEvents()
	if err != nil {
		return nil, err
	}
	// 为每个活动加载奖励和条件
	for _, e := range events {
		rewards, err := s.repo.GetEventRewards(e.ID)
		if err != nil {
			s.log.Warn("加载活动奖励失败", zap.String("event", e.ID), zap.Error(err))
			continue
		}
		e.Rewards = rewards

		conds, err := s.repo.GetEventConditions(e.ID)
		if err != nil {
			s.log.Warn("加载活动条件失败", zap.String("event", e.ID), zap.Error(err))
			continue
		}
		e.Conditions = conds
	}
	return events, nil
}

// GetEventDetail 获取活动详情及玩家进度
func (s *ActivityService) GetEventDetail(eventID string, playerID int64) (*model.EventDetailResponse, error) {
	event, err := s.repo.GetEventByID(eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, fmt.Errorf("活动不存在: %s", eventID)
	}

	// 加载奖励
	rewards, err := s.repo.GetEventRewards(eventID)
	if err != nil {
		return nil, err
	}
	event.Rewards = rewards

	conds, err := s.repo.GetEventConditions(eventID)
	if err != nil {
		return nil, err
	}
	event.Conditions = conds

	// 玩家进度
	progress, _ := s.repo.GetEventProgress(playerID, eventID)
	if progress == nil {
		progress = &model.EventProgress{
			PlayerID: playerID,
			EventID:  eventID,
		}
	}

	return &model.EventDetailResponse{
		Event:    event,
		Progress: progress,
		Rewards:  rewards,
	}, nil
}

// ClaimEventReward 领取活动奖励
func (s *ActivityService) ClaimEventReward(eventID string, playerID int64) ([]*model.EventReward, error) {
	event, err := s.repo.GetEventByID(eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, fmt.Errorf("活动不存在")
	}

	// 检查时间
	now := time.Now()
	if now.Before(event.StartTime) || now.After(event.EndTime) {
		return nil, fmt.Errorf("活动未在进行中")
	}

	// 检查是否已领取
	claimed, err := s.repo.HasClaimedEventReward(playerID, eventID)
	if err != nil {
		return nil, err
	}
	if claimed {
		return nil, fmt.Errorf("活动奖励已领取")
	}

	// 获取奖励
	rewards, err := s.repo.GetEventRewards(eventID)
	if err != nil {
		return nil, err
	}

	// 发放奖励
	ctx := context.Background()
	var granted []*model.EventReward
	for _, r := range rewards {
		if r.Probability > 0 && r.Probability < 1.0 {
			if rand.Float64() > r.Probability {
				continue
			}
		}
		if r.ItemID > 0 && r.Quantity > 0 {
			if _, err := s.inventorySvc.AddItem(ctx, playerID, r.ItemID, r.Quantity); err != nil {
				s.log.Warn("发放活动物品奖励失败", zap.Int64("player", playerID), zap.Int64("item", r.ItemID), zap.Error(err))
				continue
			}
		}
		granted = append(granted, r)
	}

	// 记录领取
	if err := s.repo.ClaimEventReward(playerID, eventID, "all"); err != nil {
		s.log.Warn("记录活动奖励领取失败", zap.Error(err))
	}

	// 更新进度已领取
	progress, _ := s.repo.GetEventProgress(playerID, eventID)
	if progress != nil {
		progress.Claimed = true
		s.repo.UpsertEventProgress(progress)
	}

	s.log.Info("活动奖励已发放",
		zap.Int64("player", playerID),
		zap.String("event", eventID),
		zap.Int("rewards", len(granted)),
	)
	return granted, nil
}

// UpdateEventProgress 更新玩家活动进度
func (s *ActivityService) UpdateEventProgress(playerID int64, eventID string, delta int64) error {
	progress, err := s.repo.GetEventProgress(playerID, eventID)
	if err != nil {
		return err
	}
	if progress == nil {
		progress = &model.EventProgress{
			PlayerID: playerID,
			EventID:  eventID,
		}
	}
	progress.Progress += delta
	return s.repo.UpsertEventProgress(progress)
}

// ============================================================
// 战令系统 (Battle Pass)
// ============================================================

// GetBattlePass 获取战令状态
func (s *ActivityService) GetBattlePass(playerID int64) (*model.BattlePassStatus, error) {
	season, err := s.repo.GetActiveSeason()
	if err != nil {
		return nil, err
	}
	if season == nil {
		return nil, fmt.Errorf("当前无活跃战令赛季")
	}

	tiers, err := s.repo.GetBPTiers(season.SeasonID)
	if err != nil {
		return nil, err
	}

	progress, _ := s.repo.GetBPProgress(playerID, season.SeasonID)
	if progress == nil {
		progress = &model.BPProgress{
			PlayerID:    playerID,
			SeasonID:    season.SeasonID,
			CurrentLevel: 1,
			CurrentExp:   0,
			HasPremium:   false,
		}
	}

	// 分离免费和高级
	var freeTiers, premiumTiers []*model.BPTier
	for _, t := range tiers {
		if t.IsPremium {
			premiumTiers = append(premiumTiers, t)
		} else {
			freeTiers = append(freeTiers, t)
		}
	}

	return &model.BattlePassStatus{
		Season:       season,
		Progress:     progress,
		FreeTiers:    freeTiers,
		PremiumTiers: premiumTiers,
	}, nil
}

// BuyPremiumBP 购买高级战令
func (s *ActivityService) BuyPremiumBP(playerID int64) error {
	season, err := s.repo.GetActiveSeason()
	if err != nil {
		return err
	}
	if season == nil {
		return fmt.Errorf("当前无活跃战令赛季")
	}

	progress, _ := s.repo.GetBPProgress(playerID, season.SeasonID)
	if progress == nil {
		progress = &model.BPProgress{
			PlayerID:    playerID,
			SeasonID:    season.SeasonID,
			CurrentLevel: 1,
			CurrentExp:   0,
		}
	}
	if progress.HasPremium {
		return fmt.Errorf("已拥有高级战令，无需重复购买")
	}

	// 扣费 (灵玉)
	ctx := context.Background()
	if _, err := s.playerSvc.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{
		Jade: -season.PremiumCost,
	}); err != nil {
		return fmt.Errorf("购买高级战令失败，仙玉不足: %w", err)
	}

	progress.HasPremium = true
	if err := s.repo.UpsertBPProgress(progress); err != nil {
		return err
	}

	s.log.Info("玩家购买高级战令", zap.Int64("player", playerID), zap.String("season", season.SeasonID))
	return nil
}

// ClaimBPReward 领取战令等级奖励
func (s *ActivityService) ClaimBPReward(playerID int64, level int) (*model.BPTier, error) {
	season, err := s.repo.GetActiveSeason()
	if err != nil {
		return nil, err
	}
	if season == nil {
		return nil, fmt.Errorf("当前无活跃战令赛季")
	}

	// 获取该等级配置
	tiers, err := s.repo.GetBPTiers(season.SeasonID)
	if err != nil {
		return nil, err
	}

	var targetTier *model.BPTier
	for _, t := range tiers {
		if t.Level == level {
			targetTier = t
			break
		}
	}
	if targetTier == nil {
		return nil, fmt.Errorf("战令等级 %d 不存在", level)
	}

	// 获取玩家进度
	progress, err := s.repo.GetBPProgress(playerID, season.SeasonID)
	if err != nil {
		return nil, err
	}
	if progress == nil {
		return nil, fmt.Errorf("未参与战令，请先进行任务获取经验")
	}

	// 检查等级是否已解锁
	if progress.CurrentLevel < level {
		return nil, fmt.Errorf("战令等级 %d 未解锁，当前等级 %d", level, progress.CurrentLevel)
	}

	// 高级战令检查
	if targetTier.IsPremium && !progress.HasPremium {
		return nil, fmt.Errorf("高级战令未购买，无法领取高级奖励")
	}

	// 检查是否已领取
	claimed, err := s.repo.HasClaimedBPReward(playerID, season.SeasonID, level)
	if err != nil {
		return nil, err
	}
	if claimed {
		return nil, fmt.Errorf("战令等级 %d 奖励已领取", level)
	}

	// 发放奖励
	ctx := context.Background()
	if targetTier.RewardItemID > 0 && targetTier.RewardQuantity > 0 {
		if _, err := s.inventorySvc.AddItem(ctx, playerID, targetTier.RewardItemID, targetTier.RewardQuantity); err != nil {
			return nil, fmt.Errorf("发放战令奖励失败: %w", err)
		}
	}

	// 处理特殊奖励类型
	switch targetTier.RewardType {
	case "title":
		// 添加称号
		if err := s.repo.AddPlayerTitle(playerID, fmt.Sprintf("%d", targetTier.RewardItemID)); err != nil {
			s.log.Warn("添加战令称号失败", zap.Error(err))
		}
	}

	// 记录领取
	if err := s.repo.ClaimBPReward(playerID, season.SeasonID, level); err != nil {
		return nil, err
	}

	// 更新已领取列表
	claimedLevels := mysql.UnmarshalClaimedLevels(progress.ClaimedLevels)
	claimedLevels = append(claimedLevels, level)
	progress.ClaimedLevels = mysql.MarshalClaimedLevels(claimedLevels)
	if err := s.repo.UpsertBPProgress(progress); err != nil {
		return nil, err
	}

	s.log.Info("战令奖励已领取",
		zap.Int64("player", playerID),
		zap.Int("level", level),
		zap.String("reward", targetTier.RewardName),
	)
	return targetTier, nil
}

// AddBPExp 为玩家添加战令经验（由外部事件触发）
func (s *ActivityService) AddBPExp(playerID int64, exp int64) error {
	season, err := s.repo.GetActiveSeason()
	if err != nil {
		return err
	}
	if season == nil {
		return nil // 无赛季时静默忽略
	}

	progress, err := s.repo.GetBPProgress(playerID, season.SeasonID)
	if err != nil {
		return err
	}
	if progress == nil {
		progress = &model.BPProgress{
			PlayerID:    playerID,
			SeasonID:    season.SeasonID,
			CurrentLevel: 1,
			CurrentExp:   0,
		}
	}

	progress.CurrentExp += exp

	// 升级计算: 每级需要 100 + (level-1)*50 经验
	maxLevel := 60
	for progress.CurrentLevel < maxLevel {
		expNeeded := int64(100 + (progress.CurrentLevel-1)*50)
		if progress.CurrentExp >= expNeeded {
			progress.CurrentExp -= expNeeded
			progress.CurrentLevel++
		} else {
			break
		}
	}
	// 满级后经验保留但不再升级
	if progress.CurrentLevel >= maxLevel {
		progress.CurrentLevel = maxLevel
		if progress.CurrentExp > 0 {
			progress.CurrentExp = 0
		}
	}

	return s.repo.UpsertBPProgress(progress)
}

// ============================================================
// 签到增强 (Enhanced Check-in)
// ============================================================

// GetEnhancedCheckinStatus 获取增强签到状态
func (s *ActivityService) GetEnhancedCheckinStatus(playerID int64) (*model.EnhancedCheckinStatus, error) {
	// 先从现有签到服务获取基础数据
	status := &model.EnhancedCheckinStatus{
		MonthDays:        make([]bool, 28),
		MilestoneClaimed: make([]bool, 4), // [7,14,21,28]
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	monthStr := now.Format("2006-01")

	// 获取签到记录
	rec, err := s.checkinRepo.GetByPlayerID(playerID)
	if err != nil {
		return nil, err
	}

	if rec == nil {
		status.CheckedInToday = false
		status.ConsecutiveDays = 0
		status.MonthTotal = 0
		status.CanMakeup = false
		status.MakeupCount = 0
		status.MakeupCost = model.GetMakeupCost(0)
		status.StreakBonus = 1.0
		return status, nil
	}

	checkedInToday := rec.LastCheckinDate == today
	canMakeup := !checkedInToday

	// 获取月签到天数
	days, makeupCount, err := s.repo.GetMonthlyCheckinDays(playerID, monthStr)
	if err != nil {
		s.log.Warn("获取月签到天数失败", zap.Error(err))
		days = nil
	}

	// 构建月签到数组
	for _, d := range days {
		if d >= 1 && d <= 28 {
			status.MonthDays[d-1] = true
		}
	}

	// 获取里程碑奖励状态
	milestoneDays := []int{7, 14, 21, 28}
	for i, md := range milestoneDays {
		claimed, _ := s.repo.HasMilestoneClaimed(playerID, monthStr, md)
		status.MilestoneClaimed[i] = claimed
	}

	status.CheckedInToday = checkedInToday
	status.ConsecutiveDays = rec.ConsecutiveDays
	status.MonthTotal = int32(len(days))
	status.CanMakeup = canMakeup
	status.MakeupCount = makeupCount
	status.MakeupCost = model.GetMakeupCost(makeupCount)
	status.StreakBonus = s.repo.GetStreakBonus(rec.ConsecutiveDays)

	return status, nil
}

// DoEnhancedCheckin 增强签到
func (s *ActivityService) DoEnhancedCheckin(playerID int64) (*model.CheckinResult, error) {
	// 复用现有签到逻辑
	rec, err := s.checkinRepo.GetByPlayerID(playerID)
	if err != nil {
		return nil, err
	}

	if rec == nil {
		rec = &model.CheckinRecord{
			PlayerID:        playerID,
			LastCheckinDate: "",
			ConsecutiveDays: 0,
			WeekStartDate:   "",
			WeekClaimedMask: 0,
			MonthTotal:      0,
		}
	}

	today := time.Now().Format("2006-01-02")
	now := time.Now()
	monthStr := now.Format("2006-01")
	dayOfMonth := now.Day()

	if rec.LastCheckinDate == today {
		return nil, fmt.Errorf("今日已签到，请明天再来")
	}

	// 连续签到
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	if rec.LastCheckinDate == yesterday {
		rec.ConsecutiveDays++
	} else {
		rec.ConsecutiveDays = 1
	}

	rec.LastCheckinDate = today

	// 月签到记录
	if rec.MonthStr != monthStr {
		rec.MonthStr = monthStr
		rec.MonthTotal = 0
		rec.MonthRewardClaimed = false
	}
	rec.MonthTotal++

	if err := s.repo.InsertMonthlyCheckin(playerID, monthStr, dayOfMonth, false); err != nil {
		s.log.Warn("插入月签到记录失败", zap.Error(err))
	}

	// 更新基础签到记录
	if err := s.checkinRepo.Upsert(rec); err != nil {
		return nil, err
	}

	// 获取里程碑奖励 (在 day >= 7/14/21/28 时发放里程碑奖励物品)
	if dayOfMonth == 28 && rec.MonthTotal >= 28 && !rec.MonthRewardClaimed {
		rec.MonthRewardClaimed = true
		s.checkinRepo.Upsert(rec)
	}

	reward := &model.CheckinResult{
		Reward: &model.CheckinReward{
			Day:      int(rec.ConsecutiveDays),
			Gold:     100,
			ItemName: "灵石",
			Quantity: 100,
		},
		Makeup:     false,
		CostGold:   0,
		FullMonth:  rec.MonthTotal >= 28,
		MonthTotal: rec.MonthTotal,
		Streak:     rec.ConsecutiveDays,
	}

	// 计算连续签到倍率
	bonus := s.repo.GetStreakBonus(rec.ConsecutiveDays)
	if bonus > 1.0 {
		reward.Reward.Gold = int64(float64(reward.Reward.Gold) * bonus)
	}

	return reward, nil
}

// DoMakeupCheckinEnhanced 增强补签(递增花费)
func (s *ActivityService) DoMakeupCheckinEnhanced(playerID int64) (*model.CheckinResult, error) {
	rec, err := s.checkinRepo.GetByPlayerID(playerID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("尚无签到记录，请先正常签到")
	}

	today := time.Now().Format("2006-01-02")
	now := time.Now()
	monthStr := now.Format("2006-01")
	dayOfMonth := now.Day()

	if rec.LastCheckinDate == today {
		return nil, fmt.Errorf("今日已签到，无需补签")
	}


	_, makeupCount, _ := s.repo.GetMonthlyCheckinDays(playerID, monthStr)
	cost := model.GetMakeupCost(makeupCount)

	// 扣费
	ctx := context.Background()
	if _, err := s.playerSvc.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{
		Gold: -cost,
	}); err != nil {
		return nil, fmt.Errorf("灵石不足，补签需要 %d 灵石: %w", cost, err)
	}

	// 补签逻辑
	rec.LastCheckinDate = today
	rec.ConsecutiveDays++
	rec.MonthTotal++

	// 月补签记录
	if err := s.repo.InsertMonthlyCheckin(playerID, monthStr, dayOfMonth, true); err != nil {
		s.log.Warn("插入月补签记录失败", zap.Error(err))
	}

	if err := s.checkinRepo.Upsert(rec); err != nil {
		return nil, err
	}

	return &model.CheckinResult{
		Reward: &model.CheckinReward{
			Day:      int(rec.ConsecutiveDays),
			Gold:     50,
			ItemName: "灵石",
			Quantity: 50,
		},
		Makeup:     true,
		CostGold:   cost,
		FullMonth:  rec.MonthTotal >= 28,
		MonthTotal: rec.MonthTotal,
		Streak:     rec.ConsecutiveDays,
	}, nil
}

// ============================================================
// 成就系统增强 (Enhanced Achievement)
// ============================================================

// GetAllAchievementDefinitions 获取所有成就定义(带等级)
func (s *ActivityService) GetAllAchievementDefinitions() ([]*model.AchievementReq, error) {
	return s.repo.GetAllAchievements()
}

// GetPlayerAchievements 获取玩家成就进度(带等级)
func (s *ActivityService) GetPlayerAchievements(playerID int64) ([]*model.AchievementReq, []*model.PlayerAchievementTier, error) {
	achievements, err := s.repo.GetAllAchievements()
	if err != nil {
		return nil, nil, err
	}

	// 获取所有等级配置
	tierMap := make(map[string][]*model.AchievementTier)
	for _, ach := range achievements {
		tiers, err := s.repo.GetAchievementTiers(ach.ID)
		if err == nil && len(tiers) > 0 {
			tierMap[ach.ID] = tiers
		}
	}

	// 获取玩家进度
	var playerAchievements []*model.PlayerAchievementTier
	for _, ach := range achievements {
		pa, _ := s.repo.GetPlayerAchievement(playerID, ach.ID)
		if pa == nil {
			pa = &model.PlayerAchievementTier{
				PlayerID:    playerID,
				AchievementID: ach.ID,
				CurrentTier: 0,
				Progress:    0,
				Completed:   false,
			}
		}
		playerAchievements = append(playerAchievements, pa)
	}

	return achievements, playerAchievements, nil
}

// ClaimAchievementTier 领取成就等级奖励
func (s *ActivityService) ClaimAchievementTier(playerID int64, achievementID string, tier int) (*model.AchievementTier, error) {
	// 获取成就定义
	ach, err := s.repo.GetAchievementByID(achievementID)
	if err != nil {
		return nil, err
	}
	if ach == nil {
		return nil, fmt.Errorf("成就不存在")
	}

	// 获取等级配置
	tiers, err := s.repo.GetAchievementTiers(achievementID)
	if err != nil {
		return nil, err
	}

	var targetTier *model.AchievementTier
	for _, t := range tiers {
		if t.Level == tier {
			targetTier = t
			break
		}
	}
	if targetTier == nil {
		return nil, fmt.Errorf("成就等级 %d 不存在", tier)
	}

	// 获取玩家进度
	pa, err := s.repo.GetPlayerAchievement(playerID, achievementID)
	if err != nil {
		return nil, err
	}
	if pa == nil {
		return nil, fmt.Errorf("成就记录未初始化")
	}
	if !pa.Completed && pa.Progress < targetTier.Condition {
		return nil, fmt.Errorf("未达到成就条件")
	}

	// 检查该等级是否已领取
	claimedTiers := parseTierList(pa.ClaimedTiers)
	for _, ct := range claimedTiers {
		if ct == tier {
			return nil, fmt.Errorf("该等级奖励已领取")
		}
	}

	// 发放奖励
	ctx := context.Background()
		if targetTier.RewardExp > 0 {
			// 发放经验
			if _, err := s.playerSvc.AddExp(ctx, playerID, targetTier.RewardExp); err != nil {
				s.log.Warn("发放成就经验失败", zap.Error(err))
			}
		}
	if targetTier.RewardMoney > 0 {
		if _, err := s.playerSvc.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{
			Gold: targetTier.RewardMoney,
		}); err != nil {
			s.log.Warn("发放成就灵石失败", zap.Error(err))
		}
	}

	// 称号解锁
	if targetTier.TitleID != "" {
		if err := s.repo.AddPlayerTitle(playerID, targetTier.TitleID); err != nil {
			s.log.Warn("添加成就称号失败", zap.Error(err))
		}
	}

	// 更新领取状态
	claimedTiers = append(claimedTiers, tier)
	pa.ClaimedTiers = formatTierList(claimedTiers)
	if err := s.repo.UpsertPlayerAchievement(pa); err != nil {
		return nil, err
	}

	s.log.Info("成就等级奖励已领取",
		zap.Int64("player", playerID),
		zap.String("achievement", achievementID),
		zap.Int("tier", tier),
	)
	return targetTier, nil
}

// UpdateAchievementProgress 更新成就进度(由外部事件调用)
func (s *ActivityService) UpdateAchievementProgress(playerID int64, achievementID string, delta int64) error {
	pa, err := s.repo.GetPlayerAchievement(playerID, achievementID)
	if err != nil {
		return err
	}
	if pa == nil {
		// 自动初始化
		pa = &model.PlayerAchievementTier{
			PlayerID:    playerID,
			AchievementID: achievementID,
			CurrentTier: 0,
			Progress:    0,
		}
	}

	pa.Progress += delta

	// 检查是否达成更高等级
	tiers, err := s.repo.GetAchievementTiers(achievementID)
	if err == nil {
		for _, t := range tiers {
			if pa.Progress >= t.Condition && t.Level > pa.CurrentTier {
				pa.CurrentTier = t.Level
				pa.Completed = true
			}
		}
	}

	return s.repo.UpsertPlayerAchievement(pa)
}

// ============================================================
// 称号系统 (Title System)
// ============================================================

// GetPlayerTitles 获取玩家称号列表
func (s *ActivityService) GetPlayerTitles(playerID int64) (*model.PlayerTitleResponse, error) {
	// 获取玩家已拥有的称号记录
	playerTitles, err := s.repo.GetPlayerTitles(playerID)
	if err != nil {
		return nil, err
	}

	// 收集称号ID
	var titleIDs []string
	var equippedID string
	for _, pt := range playerTitles {
		titleIDs = append(titleIDs, pt.TitleID)
		if pt.IsEquipped {
			equippedID = pt.TitleID
		}
	}

	// 获取称号定义
	titleMap, err := s.repo.GetTitlesByIDs(titleIDs)
	if err != nil {
		return nil, err
	}

	// 构建响应
	titles := make([]*model.Title, 0, len(titleIDs))
	for _, tid := range titleIDs {
		if t, ok := titleMap[tid]; ok {
			titles = append(titles, t)
		}
	}

	// 排序: 已装备 > 稀有度降序
	sort.Slice(titles, func(i, j int) bool {
		if titles[i].ID == equippedID {
			return true
		}
		if titles[j].ID == equippedID {
			return false
		}
		return titles[i].Rarity > titles[j].Rarity
	})

	var currentTitle *model.Title
	if equippedID != "" {
		currentTitle = titleMap[equippedID]
	}

	// 计算成就点
	points, _ := s.repo.GetPlayerAchievementPoints(playerID)

	return &model.PlayerTitleResponse{
		PlayerID:    playerID,
		CurrentTitle: currentTitle,
		Titles:      titles,
		TotalPoints: points,
	}, nil
}

// EquipTitle 装备/卸下称号
func (s *ActivityService) EquipTitle(playerID int64, titleID string, equip bool) error {
	// 检查称号是否存在
	title, err := s.repo.GetTitleByID(titleID)
	if err != nil {
		return err
	}
	if title == nil {
		return fmt.Errorf("称号不存在: %s", titleID)
	}

	// 检查玩家是否拥有此称号
	playerTitles, err := s.repo.GetPlayerTitles(playerID)
	if err != nil {
		return err
	}
	hasTitle := false
	for _, pt := range playerTitles {
		if pt.TitleID == titleID {
			hasTitle = true
			break
		}
	}
	if !hasTitle {
		return fmt.Errorf("未获得称号: %s", title.Name)
	}

	if err := s.repo.EquipPlayerTitle(playerID, titleID, equip); err != nil {
		return err
	}

	action := "装备"
	if !equip {
		action = "卸下"
	}
	s.log.Info("玩家"+action+"称号", zap.Int64("player", playerID), zap.String("title", titleID))
	return nil
}

// GetEquippedTitle 获取当前装备的称号
func (s *ActivityService) GetEquippedTitle(playerID int64) (*model.Title, error) {
	pt, err := s.repo.GetEquippedTitle(playerID)
	if err != nil {
		return nil, err
	}
	if pt == nil {
		return nil, nil
	}
	return s.repo.GetTitleByID(pt.TitleID)
}

// ============================================================
// 初始化数据
// ============================================================

// InitPlayerActivityData 初始化新玩家的活动数据
func (s *ActivityService) InitPlayerActivityData(ctx context.Context, playerID int64) error {
	// 初始化成就
	achievements, err := s.repo.GetAllAchievements()
	if err != nil {
		return err
	}
	ids := make([]string, len(achievements))
	for i, a := range achievements {
		ids[i] = a.ID
	}
	if len(ids) > 0 {
		if err := s.repo.BatchInitAchievements(playerID, ids); err != nil {
			return err
		}
	}
	return nil
}

// GetAchievementTiers 获取成就等级配置
func (s *ActivityService) GetAchievementTiers(achievementID string) ([]*model.AchievementTier, error) {
	return s.repo.GetAchievementTiers(achievementID)
}

// ============================================================
// 工具函数
// ============================================================

func parseTierList(data string) []int {
	if data == "" {
		return nil
	}
	parts := strings.Split(data, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		var v int
		if _, err := fmt.Sscanf(p, "%d", &v); err == nil {
			result = append(result, v)
		}
	}
	return result
}

func formatTierList(list []int) string {
	parts := make([]string, len(list))
	for i, v := range list {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}

// CalculateExpForLevel 计算某级所需经验
func CalculateExpForLevel(level int) int64 {
	return int64(100 + (level-1)*50)
}

// CalculateBPLevel 根据经验计算战令等级
func CalculateBPLevel(exp int64) (level int, remaining int64) {
	level = 1
	remaining = exp
	for level < 60 {
		needed := CalculateExpForLevel(level)
		if remaining >= needed {
			remaining -= needed
			level++
		} else {
			break
		}
	}
	if level > 60 {
		level = 60
		remaining = 0
	}
	return
}

// GetRemainingEventTime 获取活动剩余时间描述
func GetRemainingEventTime(endTime time.Time) string {
	remaining := time.Until(endTime)
	if remaining <= 0 {
		return "已结束"
	}

	days := int(math.Floor(remaining.Hours() / 24))
	hours := int(math.Floor(remaining.Hours())) % 24
	minutes := int(math.Floor(remaining.Minutes())) % 60

	if days > 0 {
		return fmt.Sprintf("%d天%d小时", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}
