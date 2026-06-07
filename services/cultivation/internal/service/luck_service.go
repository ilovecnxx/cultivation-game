package service
import (
	"fmt"
	"log/slog"
	"time"
	"cultivation-game/services/cultivation/internal/model"
)
// LuckService 气运系统服务
// 管理玩家的气运值获取、消耗以及气运与业力的抵消机制。
type LuckService struct {
	logger *slog.Logger
	repo PlayerRepository
}
// NewLuckService 创建气运服务实例
func NewLuckService(logger *slog.Logger, repo PlayerRepository) *LuckService {
	return &LuckService{logger: logger, repo: repo}
}
// GetLuck 获取玩家当前气运值
// 返回 -1 表示玩家不存在
func (s *LuckService) GetLuck(playerID uint64) (int64, error) {
	player, err := s.getPlayer(playerID)
	if err != nil {
		return 0, err
	}
	if player == nil {
		return -1, fmt.Errorf("玩家 %d 不存在", playerID)
	}
	return player.Luck, nil
}
// AddLuck 增加玩家气运值
// amount 可以是负值（等价于 SpendLuck）
func (s *LuckService) AddLuck(playerID uint64, amount int64) error {
	player, err := s.getPlayer(playerID)
	if err != nil {
		return err
	}
	if player == nil {
		return fmt.Errorf("玩家 %d 不存在", playerID)
	}
	player.Luck += amount
	if player.Luck < 0 {
		player.Luck = 0
	}
	s.logger.Info("玩家气运变化", "player_id", playerID, "delta", amount, "current_luck", player.Luck)
	return s.repo.SavePlayer(player)
}
// SpendLuck 消耗玩家气运值
// 不会使气运降为负数
func (s *LuckService) SpendLuck(playerID uint64, amount int64) error {
	player, err := s.getPlayer(playerID)
	if err != nil {
		return err
	}
	if player == nil {
		return fmt.Errorf("玩家 %d 不存在", playerID)
	}
	if player.Luck < amount {
		player.Luck = 0
	} else {
		player.Luck -= amount
	}
	s.logger.Info("玩家消耗气运", "player_id", playerID, "amount", amount, "current_luck", player.Luck)
	return s.repo.SavePlayer(player)
}
// DailyCheckIn 每日签到
// 每天首次签到获得 +10 气运，返回本次增加的气运值
// 已签到过返回 0
func (s *LuckService) DailyCheckIn(playerID uint64) (int64, error) {
	player, err := s.getPlayer(playerID)
	if err != nil {
		return 0, err
	}
	if player == nil {
		return 0, fmt.Errorf("玩家 %d 不存在", playerID)
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	if player.DailyCheckInTime >= todayStart {
		return 0, nil // 今日已签到
	}
	player.DailyCheckInTime = now.Unix()
	player.Luck += 10
	s.logger.Info("玩家每日签到", "player_id", playerID, "luck_gained", 10, "current_luck", player.Luck)
	if err := s.repo.SavePlayer(player); err != nil {
		return 0, fmt.Errorf("保存签到结果失败: %w", err)
	}
	return 10, nil
}
// OnKarmaAbsorb 业力吸收处理
// 玩家获得业力时，气运可以抵消部分业力影响
// 规则：每 1 点气运抵消 1 点业力，气运消耗后业力减少
// 返回实际增加的业力值（抵消后）
func (s *LuckService) OnKarmaAbsorb(playerID uint64, karma int64) (int64, error) {
	player, err := s.getPlayer(playerID)
	if err != nil {
		return 0, err
	}
	if player == nil {
		return 0, fmt.Errorf("玩家 %d 不存在", playerID)
	}
	if karma <= 0 {
		return 0, nil
	}
	// 气运抵消：消耗气运减少业力
	offset := karma
	if player.Luck >= karma {
		offset = 0
		player.Luck -= karma
	} else {
		offset = karma - player.Luck
		player.Luck = 0
	}
	player.Karma += offset
	s.logger.Info("玩家业力吸收", "player_id", playerID, "raw_karma", karma, "luck_offset", karma-offset, "actual_karma", offset, "current_luck", player.Luck, "current_karma", player.Karma)
	if err := s.repo.SavePlayer(player); err != nil {
		return 0, fmt.Errorf("保存业力结果失败: %w", err)
	}
	return offset, nil
}
// getPlayer 根据ID获取玩家
func (s *LuckService) getPlayer(playerID uint64) (*model.Player, error) {
	return s.repo.GetPlayer(playerID)
}