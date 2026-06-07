package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// ReferralRepository 推荐系统数据存储接口
type ReferralRepository interface {
	CreateInviteCode(code *model.InviteCode) error
	GetInviteCodeByPlayerID(playerID int64) (*model.InviteCode, error)
	GetInviteCodeByCode(code string) (*model.InviteCode, error)
	IncrementInviteCodeUsage(codeID int64) error
	CreateReferralRecord(record *model.ReferralRecord) error
	GetReferralRecordByInviteeID(inviteeID int64) (*model.ReferralRecord, error)
	GetReferralRecordsByInviterID(inviterID int64) ([]*model.ReferralRecord, error)
	UpdateReferralRealmReached(recordID int64, realmReachedBits int8) error
	UpdateRewardClaimed(recordID int64, claimedBits int8) error
}

// ReferralService 邀请/推荐系统业务逻辑
type ReferralService struct {
	repo          ReferralRepository
	playerService *PlayerService
	inventorySvc  *InventoryService
	log           *zap.Logger
}

// NewReferralService 创建 ReferralService
func NewReferralService(
	repo ReferralRepository,
	playerService *PlayerService,
	inventorySvc *InventoryService,
	log *zap.Logger,
) *ReferralService {
	return &ReferralService{
		repo:          repo,
		playerService: playerService,
		inventorySvc:  inventorySvc,
		log:           log,
	}
}

// inviteCodeChars 邀请码字符集（排除易混淆字符 0/O/I/l）
const inviteCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"

// GenerateInviteCode 为玩家生成唯一邀请码
func (s *ReferralService) GenerateInviteCode(ctx context.Context, playerID int64) (string, error) {
	// 检查玩家是否已有邀请码
	existing, err := s.repo.GetInviteCodeByPlayerID(playerID)
	if err != nil {
		return "", fmt.Errorf("检查现有邀请码失败: %w", err)
	}
	if existing != nil {
		return existing.InviteCode, nil
	}

	// 验证玩家存在
	_, err = s.playerService.GetPlayer(ctx, playerID)
	if err != nil {
		return "", fmt.Errorf("获取玩家信息失败: %w", err)
	}

	// 生成唯一 8 位邀请码（最多重试 10 次）
	var code string
	for i := 0; i < 10; i++ {
		candidate, err := generateRandomCode(8)
		if err != nil {
			return "", fmt.Errorf("生成邀请码失败: %w", err)
		}
		// 检查是否已存在
		existingCode, err := s.repo.GetInviteCodeByCode(candidate)
		if err != nil {
			return "", fmt.Errorf("检查邀请码冲突失败: %w", err)
		}
		if existingCode == nil {
			code = candidate
			break
		}
	}
	if code == "" {
		return "", fmt.Errorf("无法生成唯一邀请码（请稍后重试）")
	}

	inviteCode := &model.InviteCode{
		PlayerID:   playerID,
		InviteCode: code,
		TimesUsed:  0,
	}
	if err := s.repo.CreateInviteCode(inviteCode); err != nil {
		return "", err
	}

	return code, nil
}

// generateRandomCode 生成 n 位随机字母数字编码
func generateRandomCode(length int) (string, error) {
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
		if err != nil {
			return "", err
		}
		sb.WriteByte(inviteCodeChars[idx.Int64()])
	}
	return sb.String(), nil
}

// ApplyInviteCode 应用邀请码（邀请者使用被邀请者的邀请码建立关联）
// 在被邀请者创建角色后调用
func (s *ReferralService) ApplyInviteCode(ctx context.Context, inviteeID int64, code string) error {
	// 1. 查找邀请码
	inviteCode, err := s.repo.GetInviteCodeByCode(code)
	if err != nil {
		return fmt.Errorf("查询邀请码失败: %w", err)
	}
	if inviteCode == nil {
		return fmt.Errorf("邀请码无效或不存在")
	}

	// 2. 不可自己邀请自己
	if inviteCode.PlayerID == inviteeID {
		return fmt.Errorf("不可使用自己的邀请码")
	}

	// 3. 检查被邀请者是否已被邀请
	existing, err := s.repo.GetReferralRecordByInviteeID(inviteeID)
	if err != nil {
		return fmt.Errorf("检查推荐记录失败: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("已被其他玩家邀请，不可重复使用邀请码")
	}

	// 4. 验证被邀请者存在
	_, err = s.playerService.GetPlayer(ctx, inviteeID)
	if err != nil {
		return fmt.Errorf("获取被邀请者信息失败: %w", err)
	}

	// 5. 创建推荐记录
	record := &model.ReferralRecord{
		InviterID:          inviteCode.PlayerID,
		InviteeID:          inviteeID,
		InviteeRealmReached: 0,
		RewardClaimed:      0,
	}
	if err := s.repo.CreateReferralRecord(record); err != nil {
		return err
	}

	// 6. 增加邀请码使用次数
	if err := s.repo.IncrementInviteCodeUsage(inviteCode.ID); err != nil {
		s.log.Warn("增加邀请码使用次数失败", zap.Int64("codeID", inviteCode.ID), zap.Error(err))
	}

	return nil
}

// GetReferralInfo 获取邀请者完整的邀请信息
func (s *ReferralService) GetReferralInfo(ctx context.Context, playerID int64) (*model.ReferralInfo, error) {
	inviteCode, err := s.repo.GetInviteCodeByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询邀请码失败: %w", err)
	}
	if inviteCode == nil {
		// 尚无邀请码，自动生成
		code, err := s.GenerateInviteCode(ctx, playerID)
		if err != nil {
			return nil, err
		}
		return &model.ReferralInfo{
			InviteCode: code,
			TimesUsed:  0,
			Referrals:  []*model.ReferralDetail{},
		}, nil
	}

	// 查询推荐列表
	records, err := s.repo.GetReferralRecordsByInviterID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询推荐列表失败: %w", err)
	}

	details, err := s.buildReferralDetails(ctx, records)
	if err != nil {
		return nil, err
	}

	pendingCount := 0
	for _, d := range details {
		if d.ClaimableBits > 0 {
			pendingCount++
		}
	}

	return &model.ReferralInfo{
		InviteCode:   inviteCode.InviteCode,
		TimesUsed:    inviteCode.TimesUsed,
		Referrals:    details,
		PendingCount: pendingCount,
	}, nil
}

// buildReferralDetails 构建被邀请者的详细里程碑信息
func (s *ReferralService) buildReferralDetails(ctx context.Context, records []*model.ReferralRecord) ([]*model.ReferralDetail, error) {
	if len(records) == 0 {
		return []*model.ReferralDetail{}, nil
	}

	details := make([]*model.ReferralDetail, 0, len(records))
	for _, record := range records {
		// 查询被邀请者当前境界
		invitee, err := s.playerService.GetPlayer(ctx, record.InviteeID)
		if err != nil {
			// 可能已删除，跳过
			s.log.Warn("查询被邀请者信息失败，跳过", zap.Int64("inviteeID", record.InviteeID), zap.Error(err))
			continue
		}

		// 计算已达成的里程碑位
		reachedBits := calcReachedBits(invitee.Realm)

		// 如果被邀请者达到了新的里程碑，更新数据库
		if reachedBits != record.InviteeRealmReached {
			if err := s.repo.UpdateReferralRealmReached(record.ID, reachedBits); err != nil {
				s.log.Warn("更新推荐记录境界里程碑失败", zap.Int64("recordID", record.ID), zap.Error(err))
			}
			record.InviteeRealmReached = reachedBits
		}

		// 可领取位 = 已达成的位 & ^已领取的位
		claimableBits := reachedBits & ^record.RewardClaimed

		realmName := realmName(invitee.Realm)

		milestones := buildMilestoneStatus(reachedBits, record.RewardClaimed)

		detail := &model.ReferralDetail{
			InviteeID:         record.InviteeID,
			InviteeName:       invitee.Name,
			InviteeRealm:      invitee.Realm,
			InviteeRealmName:  realmName,
			RealmReachedBits:  reachedBits,
			RewardClaimedBits: record.RewardClaimed,
			ClaimableBits:     claimableBits,
			Milestones:        milestones,
		}
		details = append(details, detail)
	}
	return details, nil
}

// calcReachedBits 根据玩家境界计算已达成的里程碑位
func calcReachedBits(realm int32) int8 {
	var bits int8
	if realm >= model.MilestoneRealmBase {
		bits |= model.MilestoneBitBase
	}
	if realm >= model.MilestoneRealmNascent {
		bits |= model.MilestoneBitNascent
	}
	if realm >= model.MilestoneRealmSpirit {
		bits |= model.MilestoneBitSpirit
	}
	if realm >= model.MilestoneRealmAscend {
		bits |= model.MilestoneBitAscend
	}
	return bits
}

// buildMilestoneStatus 构建里程碑状态列表
func buildMilestoneStatus(reachedBits, claimedBits int8) []*model.MilestoneStatus {
	statuses := make([]*model.MilestoneStatus, len(model.Milestones))
	for i, m := range model.Milestones {
		reached := (reachedBits & m.Bit) != 0
		claimed := (claimedBits & m.Bit) != 0
		statuses[i] = &model.MilestoneStatus{
			Bit:      m.Bit,
			Name:     m.Name,
			Reached:  reached,
			Claimed:  claimed,
			CanClaim: reached && !claimed,
		}
	}
	return statuses
}

// ClaimReferralReward 领取推荐里程碑奖励
func (s *ReferralService) ClaimReferralReward(ctx context.Context, playerID int64, inviteeID int64) error {
	// 1. 查找推荐记录
	record, err := s.repo.GetReferralRecordByInviteeID(inviteeID)
	if err != nil {
		return fmt.Errorf("查询推荐记录失败: %w", err)
	}
	if record == nil {
		return fmt.Errorf("推荐记录不存在")
	}
	if record.InviterID != playerID {
		return fmt.Errorf("无权领取此奖励")
	}

	// 2. 计算最新可达里程碑位
	invitee, err := s.playerService.GetPlayer(ctx, inviteeID)
	if err != nil {
		return fmt.Errorf("获取被邀请者信息失败: %w", err)
	}
	reachedBits := calcReachedBits(invitee.Realm)

	// 3. 更新数据库中的里程碑位
	if reachedBits != record.InviteeRealmReached {
		if err := s.repo.UpdateReferralRealmReached(record.ID, reachedBits); err != nil {
			return fmt.Errorf("更新里程碑状态失败: %w", err)
		}
		record.InviteeRealmReached = reachedBits
	}

	// 4. 计算可领取的新里程碑位
	claimable := reachedBits & ^record.RewardClaimed
	if claimable == 0 {
		return fmt.Errorf("当前没有可领取的奖励")
	}

	// 5. 发放每个里程碑的奖励
	for _, m := range model.Milestones {
		if (claimable & m.Bit) == 0 {
			continue
		}
		if err := s.grantMilestoneReward(ctx, playerID, m.Bit); err != nil {
			return fmt.Errorf("发放%s奖励失败: %w", m.Name, err)
		}
	}

	// 6. 更新已领取状态
	newClaimedBits := record.RewardClaimed | claimable
	if err := s.repo.UpdateRewardClaimed(record.ID, newClaimedBits); err != nil {
		return fmt.Errorf("更新奖励领取状态失败: %w", err)
	}

	return nil
}

// grantMilestoneReward 发放单个里程碑奖励
func (s *ReferralService) grantMilestoneReward(ctx context.Context, playerID int64, bit int8) error {
	switch bit {
	case model.MilestoneBitBase:
		// 筑基里程碑：100 灵石
		_, err := s.playerService.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{Gold: 100})
		return err

	case model.MilestoneBitNascent:
		// 元婴里程碑：500 灵石 + 丹药
		_, err := s.playerService.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{Gold: 500})
		if err != nil {
			return err
		}
		// 发放丹药（假设道具ID=10001 为修炼丹，根据实际配置调整）
		if s.inventorySvc != nil {
			_, err = s.inventorySvc.AddItem(ctx, playerID, 10001, 1)
			if err != nil {
				s.log.Warn("发放元婴里程碑丹药失败", zap.Int64("playerID", playerID), zap.Error(err))
			}
		}
		return nil

	case model.MilestoneBitSpirit:
		// 化神里程碑：2000 灵石 + 稀有物品
		_, err := s.playerService.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{Gold: 2000})
		if err != nil {
			return err
		}
		// 发放稀有物品（假设道具ID=20001 为稀有材料）
		if s.inventorySvc != nil {
			_, err = s.inventorySvc.AddItem(ctx, playerID, 20001, 1)
			if err != nil {
				s.log.Warn("发放化神里程碑稀有物品失败", zap.Int64("playerID", playerID), zap.Error(err))
			}
		}
		return nil

	case model.MilestoneBitAscend:
		// 大乘里程碑：100 仙玉
		_, err := s.playerService.UpdateCurrency(ctx, playerID, &model.CurrencyChangeRequest{Jade: 100})
		return err
	}
	return nil
}
