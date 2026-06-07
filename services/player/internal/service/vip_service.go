package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// VipService VIP业务逻辑
type VipService struct {
	repo         *mysql.VipRepo
	playerSvc    *PlayerService
	inventorySvc *InventoryService
	configs      []*model.VipLevelConfig // VIP等级配置，按等级索引
	configMu     sync.RWMutex
	log          *zap.Logger
}

// NewVipService 创建 VipService
func NewVipService(repo *mysql.VipRepo, playerSvc *PlayerService, inventorySvc *InventoryService, log *zap.Logger) *VipService {
	s := &VipService{
		repo:         repo,
		playerSvc:    playerSvc,
		inventorySvc: inventorySvc,
		configs:      make([]*model.VipLevelConfig, 0),
		log:          log,
	}
	s.LoadVipConfig()
	return s
}

// LoadVipConfig 从 JSON 文件加载VIP等级配置
func (s *VipService) LoadVipConfig() {
	data, err := os.ReadFile("internal/data/vip.json")
	if err != nil {
		s.log.Warn("读取VIP配置文件失败, 使用空配置", zap.Error(err))
		return
	}
	var list []*model.VipLevelConfig
	if err := json.Unmarshal(data, &list); err != nil {
		s.log.Warn("解析VIP配置文件失败", zap.Error(err))
		return
	}
	s.configMu.Lock()
	s.configs = list
	s.configMu.Unlock()
	s.log.Info("VIP配置加载完成", zap.Int("count", len(list)))
}

// getConfig 根据VIP等级获取配置
func (s *VipService) getConfig(level int) *model.VipLevelConfig {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	for _, c := range s.configs {
		if c.Level == level {
			return c
		}
	}
	return nil
}

// getMaxLevel 获取最高VIP等级
func (s *VipService) getMaxLevel() int {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	maxLevel := 0
	for _, c := range s.configs {
		if c.Level > maxLevel {
			maxLevel = c.Level
		}
	}
	return maxLevel
}

// GetVipInfo 获取玩家VIP完整信息（含当前等级权益）
func (s *VipService) GetVipInfo(ctx context.Context, playerID int64) (*model.VipInfoResponse, error) {
	vp, err := s.repo.GetVipPlayer(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取VIP信息失败: %w", err)
	}

	cfg := s.getConfig(vp.VipLevel)
	if cfg == nil {
		return nil, fmt.Errorf("VIP等级配置不存在: %d", vp.VipLevel)
	}

	maxLevel := s.getMaxLevel()
	var nextLevelExp int64
	if vp.VipLevel < maxLevel {
		nextCfg := s.getConfig(vp.VipLevel + 1)
		if nextCfg != nil {
			nextLevelExp = nextCfg.RequiredExp
		}
	}

	// 检查每日奖励是否可领取
	canClaimDaily := false
	today := time.Now().Format("2006-01-02")
	if vp.LastDailyClaimDate == nil || *vp.LastDailyClaimDate != today {
		canClaimDaily = vp.VipLevel > 0
	}

	// 检查月卡状态
	monthlyCardActive := false
	if vp.MonthlyCardExpiresAt != nil && vp.MonthlyCardExpiresAt.After(time.Now()) {
		monthlyCardActive = true
	}

	var expiresAtStr *string
	if vp.MonthlyCardExpiresAt != nil {
		formatted := vp.MonthlyCardExpiresAt.Format("2006-01-02 15:04:05")
		expiresAtStr = &formatted
	}

	return &model.VipInfoResponse{
		PlayerID:            vp.PlayerID,
		VipLevel:            vp.VipLevel,
		VipExp:              vp.VipExp,
		TotalRecharge:       vp.TotalRecharge,
		NextLevelExp:        nextLevelExp,
		SpeedBonus:          cfg.SpeedBonus,
		AuctionFeeDiscount:  cfg.AuctionFeeDiscount,
		ExtraSweepTickets:   cfg.ExtraSweepTickets,
		ExtraDungeonAttempts: cfg.ExtraDungeonAttempts,
		DailyRewardItems:    cfg.DailyRewardItems,
		CanClaimDaily:       canClaimDaily,
		MonthlyCardType:     vp.MonthlyCardType,
		MonthlyCardActive:   monthlyCardActive,
		MonthlyCardExpiresAt: expiresAtStr,
	}, nil
}

// AddVipExp 增加VIP经验，自动检测是否升级
func (s *VipService) AddVipExp(ctx context.Context, playerID int64, amount int64) (int, int64, error) {
	vp, err := s.repo.GetVipPlayer(playerID)
	if err != nil {
		return 0, 0, fmt.Errorf("获取VIP信息失败: %w", err)
	}

	vp.VipExp += amount

	maxLevel := s.getMaxLevel()
	oldLevel := vp.VipLevel

	// 循环检测升级
	for vp.VipLevel < maxLevel {
		nextCfg := s.getConfig(vp.VipLevel + 1)
		if nextCfg == nil {
			break
		}
		if vp.VipExp >= nextCfg.RequiredExp {
			vp.VipLevel++
		} else {
			break
		}
	}

	if vp.VipLevel != oldLevel {
		s.log.Info("VIP等级提升",
			zap.Int64("player", playerID),
			zap.Int("oldLevel", oldLevel),
			zap.Int("newLevel", vp.VipLevel),
		)
	}

	if err := s.repo.UpdateVipPlayer(vp); err != nil {
		return 0, 0, fmt.Errorf("更新VIP经验失败: %w", err)
	}

	return vp.VipLevel, vp.VipExp, nil
}

// ProcessRecharge 处理充值
// 流程：校验订单 -> 创建记录 -> 发放仙玉 -> 增加VIP经验 -> 更新状态
func (s *VipService) ProcessRecharge(ctx context.Context, playerID int64, amountRmb int, orderID string) (*model.RechargeResponse, error) {
	// 1. 校验订单号唯一性
	existing, err := s.repo.GetRechargeRecordByOrderID(orderID)
	if err != nil {
		return nil, fmt.Errorf("检查订单号失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("订单号已存在")
	}

	// 2. 计算仙玉数量（100分=1元=10仙玉）
	amountJade := amountRmb / 10

	// 3. 创建待支付充值记录
	record := &model.VipRechargeRecord{
		PlayerID:   playerID,
		AmountJade: amountJade,
		AmountRmb:  amountRmb,
		OrderID:    orderID,
		Status:     model.RechargeStatusPending,
	}
	if err := s.repo.AddRechargeRecord(record); err != nil {
		return nil, fmt.Errorf("创建充值记录失败: %w", err)
	}

	// 4. 发放仙玉到玩家账户
	_, err = s.playerSvc.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{Jade: int64(amountJade)})
	if err != nil {
		// 仙玉发放失败，标记充值记录失败
		_ = s.repo.UpdateRechargeStatus(orderID, model.RechargeStatusFailed)
		return nil, fmt.Errorf("发放仙玉失败: %w", err)
	}

	// 5. 增加VIP经验（1:1 仙玉转VIP经验）
	vipExpAmount := int64(amountJade)
	vipLevel, _, err := s.AddVipExp(ctx, playerID, vipExpAmount)
	if err != nil {
		_ = s.repo.UpdateRechargeStatus(orderID, model.RechargeStatusFailed)
		return nil, fmt.Errorf("增加VIP经验失败: %w", err)
	}

	// 6. 更新累计充值金额
	vp, err := s.repo.GetVipPlayer(playerID)
	if err == nil {
		vp.TotalRecharge += int64(amountJade)
		_ = s.repo.UpdateVipPlayer(vp)
	}

	// 7. 标记充值记录已完成
	if err := s.repo.UpdateRechargeStatus(orderID, model.RechargeStatusCompleted); err != nil {
		s.log.Warn("更新充值状态失败", zap.String("orderID", orderID), zap.Error(err))
	}

	// 获取玩家最新仙玉数量
	player, err := s.playerSvc.GetPlayer(ctx, playerID)
	if err != nil {
		s.log.Warn("获取玩家信息失败", zap.Error(err))
	}

	response := &model.RechargeResponse{
		OrderID:    orderID,
		AmountJade: amountJade,
		AddedExp:   vipExpAmount,
		VipLevel:   vipLevel,
	}
	if player != nil {
		response.NewJade = player.Jade
	}

	return response, nil
}

// ClaimDailyReward 领取VIP每日奖励
func (s *VipService) ClaimDailyReward(ctx context.Context, playerID int64) ([]model.VipDailyReward, error) {
	vp, err := s.repo.GetVipPlayer(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取VIP信息失败: %w", err)
	}

	if vp.VipLevel <= 0 {
		return nil, fmt.Errorf("VIP0无每日奖励")
	}

	cfg := s.getConfig(vp.VipLevel)
	if cfg == nil {
		return nil, fmt.Errorf("VIP等级配置不存在: %d", vp.VipLevel)
	}

	// 检查是否已领取
	today := time.Now().Format("2006-01-02")
	if vp.LastDailyClaimDate != nil && *vp.LastDailyClaimDate == today {
		return nil, fmt.Errorf("今日VIP奖励已领取")
	}

	// 发放物品到背包
	for _, reward := range cfg.DailyRewardItems {
		if s.inventorySvc != nil {
			_, err := s.inventorySvc.AddItem(ctx, playerID, int64(reward.ItemID), int32(reward.Quantity))
			if err != nil {
				s.log.Warn("发放VIP每日奖励物品失败",
					zap.Int64("player", playerID),
					zap.Int("itemID", reward.ItemID),
					zap.Error(err),
				)
			}
		}
	}

	// 更新领取日期
	vp.LastDailyClaimDate = &today
	if err := s.repo.UpdateVipPlayer(vp); err != nil {
		return nil, fmt.Errorf("更新VIP领取日期失败: %w", err)
	}

	s.log.Info("VIP每日奖励领取成功",
		zap.Int64("player", playerID),
		zap.Int("level", vp.VipLevel),
	)

	return cfg.DailyRewardItems, nil
}

// GetMonthlyCardStatus 获取月卡状态
func (s *VipService) GetMonthlyCardStatus(ctx context.Context, playerID int64) (*model.MonthlyCardStatusResponse, error) {
	vp, err := s.repo.GetVipPlayer(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取VIP信息失败: %w", err)
	}

	resp := &model.MonthlyCardStatusResponse{
		CardType: vp.MonthlyCardType,
		Active:   false,
	}

	if vp.MonthlyCardExpiresAt != nil && vp.MonthlyCardExpiresAt.After(time.Now()) {
		resp.Active = true
		resp.ExpiresAt = vp.MonthlyCardExpiresAt.Format("2006-01-02 15:04:05")
		resp.RemainingDays = int(time.Until(*vp.MonthlyCardExpiresAt).Hours() / 24)
	}

	return resp, nil
}

// ActivateMonthlyCard 激活月卡
// cardType: 1=小月卡 2=大月卡
func (s *VipService) ActivateMonthlyCard(ctx context.Context, playerID int64, cardType int8) (*model.MonthlyCardStatusResponse, error) {
	vp, err := s.repo.GetVipPlayer(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取VIP信息失败: %w", err)
	}

	now := time.Now()
	var expiresAt time.Time

	// 如果已有生效月卡，则续期（延长30天）
	if vp.MonthlyCardExpiresAt != nil && vp.MonthlyCardExpiresAt.After(now) {
		expiresAt = vp.MonthlyCardExpiresAt.Add(30 * 24 * time.Hour)
	} else {
		expiresAt = now.Add(30 * 24 * time.Hour)
	}

	vp.MonthlyCardType = cardType
	vp.MonthlyCardExpiresAt = &expiresAt

	if err := s.repo.UpdateVipPlayer(vp); err != nil {
		return nil, fmt.Errorf("激活月卡失败: %w", err)
	}

	s.log.Info("月卡激活成功",
		zap.Int64("player", playerID),
		zap.Int8("cardType", cardType),
		zap.Time("expiresAt", expiresAt),
	)

	return &model.MonthlyCardStatusResponse{
		CardType:      cardType,
		Active:        true,
		ExpiresAt:     expiresAt.Format("2006-01-02 15:04:05"),
		RemainingDays: 30,
	}, nil
}

// GetRechargeHistory 获取充值历史
func (s *VipService) GetRechargeHistory(ctx context.Context, playerID int64, limit, offset int) ([]*model.VipRechargeRecord, error) {
	records, err := s.repo.GetRechargeHistory(playerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("获取充值历史失败: %w", err)
	}
	return records, nil
}
