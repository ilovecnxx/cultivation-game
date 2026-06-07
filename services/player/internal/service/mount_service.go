// Package service 坐骑系统
//
// 坐骑种类：飞剑(筑基)/灵鹤(金丹)/七彩祥云(元婴)/神龙(化神)/凤凰(炼虚)
// 移动速度+10%~50%(按坐骑品质)
// 坐骑可升级(消耗灵石+坐骑饲料)
// 坐骑展示在角色信息页
package service

import (
	"context"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 坐骑相关物品ID
const (
	ItemMountFeed = 201 // 坐骑饲料：增加坐骑经验
)

// MountService 坐骑业务逻辑
type MountService struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewMountService 创建 MountService
func NewMountService(db *gorm.DB, log *zap.Logger) *MountService {
	return &MountService{db: db, log: log}
}

// -------- 内部辅助 --------

// checkRealm 检查玩家境界是否满足坐骑所需
func (s *MountService) checkRealm(ctx context.Context, playerID int64, needRealm int) error {
	var player model.Player
	if err := s.db.WithContext(ctx).First(&player, playerID).Error; err != nil {
		return fmt.Errorf("查询玩家失败: %w", err)
	}
	if int(player.Realm) < needRealm {
		return fmt.Errorf("境界不足，需要 %s 以上", model.RealmNames[int32(needRealm)])
	}
	return nil
}

// -------- 业务方法 --------

// GetMountList 获取玩家所有坐骑
// GET /api/v1/mount/list
func (s *MountService) GetMountList(ctx context.Context, playerID int64) ([]*model.Mount, error) {
	var mounts []*model.Mount
	if err := s.db.WithContext(ctx).Where("player_id = ?", playerID).Order("equipped desc, quality desc, level desc").Find(&mounts).Error; err != nil {
		return nil, fmt.Errorf("查询坐骑列表失败: %w", err)
	}
	return mounts, nil
}

// EquipMount 装备坐骑
// POST /api/v1/mount/equip
func (s *MountService) EquipMount(ctx context.Context, playerID int64, mountID int64) (*model.Mount, error) {
	var mount model.Mount
	if err := s.db.WithContext(ctx).First(&mount, mountID).Error; err != nil {
		return nil, fmt.Errorf("坐骑不存在")
	}
	if mount.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此坐骑")
	}

	// 检查境界
	species := model.MountSpeciesByName(mount.Species)
	if species != nil {
		if err := s.checkRealm(ctx, playerID, species.Realm); err != nil {
			return nil, err
		}
	}

	// 取消当前装备的坐骑
	if err := s.db.WithContext(ctx).Model(&model.Mount{}).
		Where("player_id = ? AND equipped = ?", playerID, true).
		Update("equipped", false).Error; err != nil {
		return nil, fmt.Errorf("取消旧坐骑失败: %w", err)
	}

	// 装备新坐骑
	mount.Equipped = true
	mount.UpdatedAt = time.Now().Unix()
	if err := s.db.WithContext(ctx).Save(&mount).Error; err != nil {
		return nil, fmt.Errorf("装备坐骑失败: %w", err)
	}

	s.log.Info("装备坐骑",
		zap.Int64("player_id", playerID),
		zap.Int64("mount_id", mountID),
		zap.String("name", mount.Name),
	)
	return &mount, nil
}

// UpgradeMount 坐骑升级
// POST /api/v1/mount/upgrade
// 消耗灵石 + 坐骑饲料
func (s *MountService) UpgradeMount(ctx context.Context, playerID int64, mountID int64) (*model.Mount, error) {
	var mount model.Mount
	if err := s.db.WithContext(ctx).First(&mount, mountID).Error; err != nil {
		return nil, fmt.Errorf("坐骑不存在")
	}
	if mount.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此坐骑")
	}
	if mount.Level >= model.MountLevelMax {
		return nil, fmt.Errorf("坐骑已达最高等级(%d级)", model.MountLevelMax)
	}

	// 计算升级所需经验
	needExp := model.MountUpgradeExp(mount.Quality, mount.Level)

	// 模拟消耗：消耗饲料获得经验，消耗灵石作为手续费
	// 每份饲料提供 fixedExp 经验（这里简化处理，实际应从背包扣除）
	fixedExpPerFeed := int64(50 * mount.Level)
	costFeed := (needExp + fixedExpPerFeed - 1) / fixedExpPerFeed
	costSpiritStone := int64(100 * mount.Level)

	// 这里简化处理：直接扣除灵石（实际项目中会调用背包服务）
	// 假设通过 InventoryService 扣除
	// 本实现中直接模拟成功

	mount.Exp += fixedExpPerFeed * costFeed
	leveledUp := 0
	for mount.Level < model.MountLevelMax && mount.Exp >= model.MountUpgradeExp(mount.Quality, mount.Level) {
		mount.Exp -= model.MountUpgradeExp(mount.Quality, mount.Level)
		mount.Level++
		leveledUp++
	}
	mount.UpdatedAt = time.Now().Unix()

	if err := s.db.WithContext(ctx).Save(&mount).Error; err != nil {
		return nil, fmt.Errorf("升级坐骑失败: %w", err)
	}

	s.log.Info("坐骑升级",
		zap.Int64("player_id", playerID),
		zap.Int64("mount_id", mountID),
		zap.Int("new_level", mount.Level),
		zap.Int("levels_gained", leveledUp),
		zap.Int64("cost_feed", costFeed),
		zap.Int64("cost_stone", costSpiritStone),
	)
	return &mount, nil
}

// GetSpeedBonus 获取玩家坐骑速度加成（供其他模块调用）
func (s *MountService) GetSpeedBonus(ctx context.Context, playerID int64) int {
	var mount model.Mount
	if err := s.db.WithContext(ctx).
		Where("player_id = ? AND equipped = ?", playerID, true).
		First(&mount).Error; err != nil {
		return 0 // 无坐骑或出错时加成为0
	}

	baseBonus := model.MountSpeedBonus[mount.Quality]
	// 每级额外增加 0.5% 的速度加成
	levelBonus := mount.Level * 5 / 10
	totalBonus := baseBonus + levelBonus
	if totalBonus > 80 {
		totalBonus = 80
	}
	return totalBonus
}

// InitPlayerMount 新玩家获得初始坐骑（筑基时赠送飞剑）
func (s *MountService) InitPlayerMount(ctx context.Context, playerID int64) error {
	mount := &model.Mount{
		PlayerID:  playerID,
		Name:      "基础飞剑",
		Species:   "flying_sword",
		Quality:   model.MountQualityCommon,
		Level:     1,
		Exp:       0,
		Equipped:  false,
		CreatedAt: time.Now().Unix(),
	}
	return s.db.WithContext(ctx).Create(mount).Error
}
