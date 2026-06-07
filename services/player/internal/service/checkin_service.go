// Package service 签到福利业务逻辑
package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// CheckinService 签到服务
type CheckinService struct {
	checkinRepo  *mysql.CheckinRepo
	playerSvc    *PlayerService
	inventorySvc *InventoryService
	log          *zap.Logger
}

// NewCheckinService 创建 CheckinService
func NewCheckinService(
	checkinRepo *mysql.CheckinRepo,
	playerSvc *PlayerService,
	inventorySvc *InventoryService,
	log *zap.Logger,
) *CheckinService {
	return &CheckinService{
		checkinRepo:  checkinRepo,
		playerSvc:    playerSvc,
		inventorySvc: inventorySvc,
		log:          log,
	}
}

// GetStatus 获取玩家签到状态
func (s *CheckinService) GetStatus(playerID int64) (*model.CheckinStatus, error) {
	rec, err := s.checkinRepo.GetByPlayerID(playerID)
	if err != nil {
		return nil, err
	}

	// 没有记录 => 从未签到
	if rec == nil {
		return &model.CheckinStatus{
			CheckedInToday:  false,
			ConsecutiveDays: 0,
			WeekClaimed:     []bool{false, false, false, false, false, false, false},
			MonthTotal:      0,
			CanMakeup:       false,
			MakeupCost:      model.MakeupCheckinCost,
			FullMonthReward: false,
		}, nil
	}

	today := time.Now().Format("2006-01-02")
	monthStr := time.Now().Format("2006-01")

	checkedInToday := rec.LastCheckinDate == today

	weekClaimed := make([]bool, 7)
	// 检查本周记录是否过期(跨周重置)
	if rec.WeekStartDate == weekStartDate(time.Now()) {
		for i := 0; i < 7; i++ {
			weekClaimed[i] = (rec.WeekClaimedMask>>i)&1 == 1
		}
	}

	// 补签判定: 最后签到不是今天 且 今天没用过补签
	canMakeup := !checkedInToday && rec.MakeupDate != today

	// 检查本月满签奖励
	monthChanged := rec.MonthStr != monthStr
	fullMonth := !monthChanged && rec.MonthTotal >= model.MonthlyFullCheckinDays

	status := &model.CheckinStatus{
		CheckedInToday:  checkedInToday,
		ConsecutiveDays: rec.ConsecutiveDays,
		WeekClaimed:     weekClaimed,
		MonthTotal:      rec.MonthTotal,
		CanMakeup:       canMakeup,
		MakeupCost:      model.MakeupCheckinCost,
		FullMonthReward: fullMonth,
	}

	// 如果跨月则重置显示数据
	if monthChanged {
		status.MonthTotal = 0
		status.WeekClaimed = []bool{false, false, false, false, false, false, false}
	}

	return status, nil
}

// Checkin 执行签到
func (s *CheckinService) Checkin(playerID int64) (*model.CheckinResult, error) {
	rec, err := s.checkinRepo.GetByPlayerID(playerID)
	if err != nil {
		return nil, err
	}

	// 首次签到 => 初始化记录
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
	weekStart := weekStartDate(now)
	weekday := (int(now.Weekday()) + 6) % 7 // Mon=0..Sun=6

	// 检查是否已签到
	if rec.LastCheckinDate == today {
		return nil, fmt.Errorf("今日已签到，请明天再来")
	}

	// 判断连续签到是否中断
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if rec.LastCheckinDate == yesterday {
		rec.ConsecutiveDays++
	} else {
		rec.ConsecutiveDays = 1
	}
	// 封顶7天(奖励按7天循环)
	if rec.ConsecutiveDays > 7 {
		rec.ConsecutiveDays = 1
	}

	rec.LastCheckinDate = today

	// 本周记录
	if rec.WeekStartDate != weekStart {
		rec.WeekStartDate = weekStart
		rec.WeekClaimedMask = 0
	}
	rec.WeekClaimedMask |= 1 << weekday

	// 本月记录
	if rec.MonthStr != monthStr {
		rec.MonthStr = monthStr
		rec.MonthTotal = 0
		rec.MonthRewardClaimed = false
	}
	rec.MonthTotal++

	// 获取本次奖励
	rewardDay := int(rec.ConsecutiveDays)
	reward := pickRewardByDay(rewardDay)

	// 发放奖励
	if err := s.grantReward(playerID, reward); err != nil {
		return nil, fmt.Errorf("发放签到奖励失败: %w", err)
	}

	// 检查月满签奖励(每月仅首次满28天时发放)
	fullMonth := false
	if rec.MonthTotal >= model.MonthlyFullCheckinDays && !rec.MonthRewardClaimed {
		if err := s.grantMonthlyFullReward(playerID); err != nil {
			s.log.Warn("发放月满签奖励失败", zap.Int64("player", playerID), zap.Error(err))
		} else {
			rec.MonthRewardClaimed = true
			fullMonth = true
		}
	}

	// 持久化
	if err := s.checkinRepo.Upsert(rec); err != nil {
		return nil, err
	}

	return &model.CheckinResult{
		Reward:     reward,
		Makeup:     false,
		CostGold:   0,
		FullMonth:  fullMonth,
		MonthTotal: rec.MonthTotal,
		Streak:     rec.ConsecutiveDays,
	}, nil
}

// MakeupCheckin 补签(消耗灵石, 每日限1次)
func (s *CheckinService) MakeupCheckin(playerID int64) (*model.CheckinResult, error) {
	rec, err := s.checkinRepo.GetByPlayerID(playerID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("尚无签到记录，请先正常签到")
	}

	today := time.Now().Format("2006-01-02")

	// 今日已签到则无需补签
	if rec.LastCheckinDate == today {
		return nil, fmt.Errorf("今日已签到，无需补签")
	}

	// 检查补签次数
	if rec.MakeupDate == today && rec.MakeupUsedToday >= 1 {
		return nil, fmt.Errorf("今日补签次数已用完")
	}

	// 扣减灵石
	ctx := context.Background()
	if _, err := s.playerSvc.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{
		Gold: -model.MakeupCheckinCost,
	}); err != nil {
		return nil, fmt.Errorf("灵石不足，补签需要 %d 灵石: %w", model.MakeupCheckinCost, err)
	}

	// 补签成功, 更新记录
	rec.LastCheckinDate = today
	rec.MakeupDate = today
	rec.MakeupUsedToday = 1

	// 连续天数+1 (补签不算中断)
	rec.ConsecutiveDays++
	if rec.ConsecutiveDays > 7 {
		rec.ConsecutiveDays = 1
	}

	// 本周标记
	weekStart := weekStartDate(time.Now())
	if rec.WeekStartDate != weekStart {
		rec.WeekStartDate = weekStart
		rec.WeekClaimedMask = 0
	}
	weekday := (int(time.Now().Weekday()) + 6) % 7
	rec.WeekClaimedMask |= 1 << weekday

	// 本月累计
	monthStr := time.Now().Format("2006-01")
	if rec.MonthStr != monthStr {
		rec.MonthStr = monthStr
		rec.MonthTotal = 0
		rec.MonthRewardClaimed = false
	}
	rec.MonthTotal++

	// 获取补签奖励(按连续天数)
	rewardDay := int(rec.ConsecutiveDays)
	reward := pickRewardByDay(rewardDay)

	// 发放奖励
	if err := s.grantReward(playerID, reward); err != nil {
		return nil, fmt.Errorf("发放补签奖励失败: %w", err)
	}

	// 月满签检查(补签也可能触发)
	fullMonth := false
	if rec.MonthTotal >= model.MonthlyFullCheckinDays && !rec.MonthRewardClaimed {
		if err := s.grantMonthlyFullReward(playerID); err != nil {
			s.log.Warn("发放月满签奖励失败", zap.Int64("player", playerID), zap.Error(err))
		} else {
			rec.MonthRewardClaimed = true
			fullMonth = true
		}
	}

	// 持久化
	if err := s.checkinRepo.Upsert(rec); err != nil {
		return nil, err
	}

	return &model.CheckinResult{
		Reward:     reward,
		Makeup:     true,
		CostGold:   model.MakeupCheckinCost,
		FullMonth:  fullMonth,
		MonthTotal: rec.MonthTotal,
		Streak:     rec.ConsecutiveDays,
	}, nil
}

// grantReward 发放签到奖励(灵石/物品/气运)
func (s *CheckinService) grantReward(playerID int64, reward *model.CheckinReward) error {
	ctx := context.Background()
	// 灵石
	if reward.Gold > 0 {
		_, err := s.playerSvc.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{Gold: reward.Gold})
		if err != nil {
			return err
		}
	}
	// 物品
	if reward.ItemID > 0 && reward.Quantity > 0 {
		_, err := s.inventorySvc.AddItem(ctx, playerID, reward.ItemID, reward.Quantity)
		if err != nil {
			// 物品发放失败记录日志但不上报(签到本身成功)
			s.log.Warn("签到物品发放失败",
				zap.Int64("player", playerID),
				zap.Int64("item", reward.ItemID),
				zap.Error(err),
			)
		}
	}
	// 气运(暂记录日志, 后续扩展)
	if reward.Luck > 0 {
		s.log.Info("签到气运加成",
			zap.Int64("player", playerID),
			zap.Int32("luck", reward.Luck),
		)
	}
	return nil
}

// grantMonthlyFullReward 月满签奖励: 随机法宝碎片
func (s *CheckinService) grantMonthlyFullReward(playerID int64) error {
	// 随机法宝碎片ID列表(法宝碎片ID范围 2001-2010)
	fragmentIDs := []int64{2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010}
	idx := rand.Intn(len(fragmentIDs))
	fragmentID := fragmentIDs[idx]

	s.log.Info("月满签奖励: 法宝碎片",
		zap.Int64("player", playerID),
		zap.Int64("fragment_id", fragmentID),
	)
	ctx := context.Background()
	_, err := s.inventorySvc.AddItem(ctx, playerID, fragmentID, 1)
	if err != nil {
		return fmt.Errorf("发放月满签法宝碎片失败: %w", err)
	}
	return nil
}

// pickRewardByDay 根据连续天数获取奖励配置
func pickRewardByDay(day int) *model.CheckinReward {
	for i := range model.CheckinRewards {
		if model.CheckinRewards[i].Day == day {
			return &model.CheckinRewards[i]
		}
	}
	return &model.CheckinRewards[0]
}

// weekStartDate 获取本周一的日期字符串
func weekStartDate(now time.Time) string {
	weekday := now.Weekday()
	offset := int(weekday)
	if weekday == time.Sunday {
		offset = 6
	} else {
		offset = int(weekday) - 1
	}
	monday := now.AddDate(0, 0, -offset)
	return monday.Format("2006-01-02")
}
