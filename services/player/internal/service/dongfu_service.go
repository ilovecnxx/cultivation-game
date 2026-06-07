package service

import (
	"context"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// 洞府功能常量
const (
	// 化神期41层解锁洞府
	DongFuRequiredLevel = 41

	// 建造消耗基数：灵石 5000×等级
	DongFuBuildCostBase = 5000

	// 建造房间消耗
	RoomBuildCost = 2000

	// 升级房间消耗基数
	RoomUpgradeCostBase = 3000

	// 灵气汇聚：每秒基础修为获取
	GatheringBasePerSecond = 0.5

	// 访客加成基数
	GuestBonusBase = 2.0

	// 洞府等级阈值（所有房间等级之和）
	DongFuLevelThresholdGuests   = 10  // 10级开启访客功能
	DongFuLevelThresholdDecorate = 15  // 15级开启装饰功能
)

// DongFuService 洞府业务逻辑
type DongFuService struct {
	dongfuRepo *mysql.DongFuRepo
	playerRepo *mysql.PlayerRepo
	log        *zap.Logger
}

// NewDongFuService 创建 DongFuService
func NewDongFuService(dongfuRepo *mysql.DongFuRepo, playerRepo *mysql.PlayerRepo, log *zap.Logger) *DongFuService {
	return &DongFuService{
		dongfuRepo: dongfuRepo,
		playerRepo: playerRepo,
		log:        log,
	}
}

// Build 建造洞府
// 条件：化神期41层+，灵石充足，每人最多1座洞府
func (s *DongFuService) Build(ctx context.Context, playerID int64, name string) (*model.DongFu, error) {
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}

	if int(player.Level) < DongFuRequiredLevel {
		return nil, fmt.Errorf("化神期(%d层)以上才能建造洞府，当前等级 %d", DongFuRequiredLevel, player.Level)
	}

	existing, err := s.dongfuRepo.GetDongFuByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("已拥有洞府[%s]，不可重复建造", existing.Name)
	}

	cost := DongFuBuildCostBase * 1
	if player.Gold < int64(cost) {
		return nil, fmt.Errorf("灵石不足，建造需要 %d 灵石，当前 %d", cost, player.Gold)
	}
	player.Gold -= int64(cost)

	dongfu := &model.DongFu{
		PlayerID: playerID,
		Level:    0,
		Name:     name,
	}

	if err := s.dongfuRepo.CreateDongFu(dongfu); err != nil {
		return nil, err
	}

	if err := s.playerRepo.Update(player); err != nil {
		return nil, fmt.Errorf("扣除灵石失败: %w", err)
	}

	s.log.Info("建造洞府成功",
		zap.Int64("player_id", playerID),
		zap.String("name", name),
		zap.Int("cost", cost),
	)
	return dongfu, nil
}

// BuildRoom 建造房间
func (s *DongFuService) BuildRoom(ctx context.Context, playerID int64, roomType int) (*model.Room, error) {
	dongfu, err := s.dongfuRepo.GetDongFuByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		return nil, fmt.Errorf("请先建造洞府")
	}

	roomName, ok := model.RoomTypeNames[roomType]
	if !ok {
		return nil, fmt.Errorf("无效的房间类型: %d", roomType)
	}

	existing, err := s.dongfuRepo.GetRoomByType(dongfu.ID, roomType)
	if err != nil {
		return nil, fmt.Errorf("查询房间失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("已拥有%s，不可重复建造", roomName)
	}

	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}
	if player.Gold < RoomBuildCost {
		return nil, fmt.Errorf("灵石不足，建造房间需要 %d 灵石，当前 %d", RoomBuildCost, player.Gold)
	}

	bonus := model.RoomTypeBonusPerLevel[roomType] * 1
	effectDesc := s.buildEffectDesc(roomType, bonus)

	room := &model.Room{
		DongFuID: dongfu.ID,
		RoomType: roomType,
		Level:    1,
		Name:     roomName,
		Effect:   effectDesc,
		Bonus:    bonus,
	}

	player.Gold -= RoomBuildCost
	if err := s.playerRepo.Update(player); err != nil {
		return nil, fmt.Errorf("扣除灵石失败: %w", err)
	}
	if err := s.dongfuRepo.CreateRoom(room); err != nil {
		return nil, err
	}

	s.recalcDongFuBonuses(dongfu)

	s.log.Info("建造房间成功",
		zap.Int64("player_id", playerID),
		zap.String("room", roomName),
	)
	return room, nil
}

// UpgradeRoom 升级房间（等级上限 10）
func (s *DongFuService) UpgradeRoom(ctx context.Context, playerID int64, roomID int64) (*model.Room, error) {
	room, err := s.dongfuRepo.GetRoomByID(roomID)
	if err != nil {
		return nil, fmt.Errorf("查询房间失败: %w", err)
	}
	if room == nil {
		return nil, fmt.Errorf("房间不存在")
	}

	dongfu, err := s.dongfuRepo.GetDongFuByID(room.DongFuID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil || dongfu.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此房间")
	}

	maxLv, ok := model.RoomTypeMaxLevel[room.RoomType]
	if !ok {
		return nil, fmt.Errorf("未知房间类型")
	}
	if room.Level >= maxLv {
		return nil, fmt.Errorf("%s已达最高等级(%d级)", room.Name, maxLv)
	}

	cost := RoomUpgradeCostBase * int64(room.Level+1)
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}
	if player.Gold < cost {
		return nil, fmt.Errorf("灵石不足，升级需要 %d 灵石，当前 %d", cost, player.Gold)
	}

	room.Level++
	room.Bonus = model.RoomTypeBonusPerLevel[room.RoomType] * float64(room.Level)
	room.Effect = s.buildEffectDesc(room.RoomType, room.Bonus)

	player.Gold -= cost
	if err := s.playerRepo.Update(player); err != nil {
		return nil, fmt.Errorf("扣除灵石失败: %w", err)
	}
	if err := s.dongfuRepo.UpdateRoom(room); err != nil {
		return nil, err
	}

	s.recalcDongFuBonuses(dongfu)

	s.log.Info("升级房间成功",
		zap.Int64("player_id", playerID),
		zap.Int("room_type", room.RoomType),
		zap.Int("new_level", room.Level),
	)
	return room, nil
}

// GetDongFu 获取洞府详情（含房间、装饰、访客、灵气汇聚状态）
func (s *DongFuService) GetDongFu(ctx context.Context, playerID int64) (*model.DongFuResponse, error) {
	resp := &model.DongFuResponse{
		CanBuild:        false,
		RequiredLevel:   DongFuRequiredLevel,
		BuildCost:       DongFuBuildCostBase * 1,
		RoomBuildCost:   RoomBuildCost,
		RoomUpgradeCost: RoomUpgradeCostBase,
	}

	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player != nil && player.Level >= DongFuRequiredLevel {
		resp.CanBuild = true
	}

	dongfu, err := s.dongfuRepo.GetDongFuByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		resp.DongFu = nil
		return resp, nil
	}

	// 查询房间列表
	rooms, err := s.dongfuRepo.GetRoomsByDongFuID(dongfu.ID)
	if err != nil {
		return nil, fmt.Errorf("查询房间列表失败: %w", err)
	}
	dongfu.Rooms = rooms

	// 查询装饰
	decorations, err := s.dongfuRepo.GetDecorationsByDongFuID(dongfu.ID)
	if err == nil {
		dongfu.Decorations = decorations
	}

	// 查询活跃的灵气汇聚
	activeGathering, err := s.dongfuRepo.GetActiveSpiritGathering(playerID)
	if err == nil && activeGathering != nil {
		// 计算实际已过时间
		elapsed := int(time.Since(activeGathering.StartTime).Seconds())
		if elapsed > activeGathering.Duration {
			elapsed = activeGathering.Duration
		}
		activeGathering.ElapsedSeconds = elapsed
		dongfu.ActiveGathering = activeGathering
	}

	// 查询访客
	guests, err := s.dongfuRepo.GetGuestsByDongFuID(dongfu.ID)
	if err == nil {
		// 填充访客名称和等级信息
		for i := range guests {
			if guests[i].GuestPlayerID > 0 {
				gp, _ := s.playerRepo.GetByID(guests[i].GuestPlayerID)
				if gp != nil {
					guests[i].GuestName = gp.Name
					guests[i].GuestLevel = int(gp.Level)
				}
			}
			if guests[i].HostPlayerID > 0 {
				hp, _ := s.playerRepo.GetByID(guests[i].HostPlayerID)
				if hp != nil {
					guests[i].HostName = hp.Name
				}
			}
		}
		dongfu.Guests = guests
	}

	resp.DongFu = dongfu
	return resp, nil
}

// ==================== 灵气汇聚 ====================

// StartGathering 开始灵气汇聚（挂机修炼）
func (s *DongFuService) StartGathering(ctx context.Context, playerID int64, duration int) (*model.SpiritGathering, error) {
	dongfu, err := s.dongfuRepo.GetDongFuByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		return nil, fmt.Errorf("请先建造洞府")
	}

	// 检查是否已有活跃的汇聚
	active, err := s.dongfuRepo.GetActiveSpiritGathering(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询灵气汇聚失败: %w", err)
	}
	if active != nil {
		return nil, fmt.Errorf("当前已有进行中的灵气汇聚，请先领取")
	}

	// 至少1分钟，最多24小时
	if duration < 60 {
		duration = 60
	}
	if duration > 86400 {
		duration = 86400
	}

	gathering := &model.SpiritGathering{
		DongFuID:   dongfu.ID,
		PlayerID:   playerID,
		Status:     model.GatheringStatusActive,
		StartTime:  time.Now(),
		Duration:   duration,
		BonusCultivation: 0,
		ElapsedSeconds: 0,
	}

	if err := s.dongfuRepo.CreateSpiritGathering(gathering); err != nil {
		return nil, err
	}

	s.log.Info("开始灵气汇聚",
		zap.Int64("player_id", playerID),
		zap.Int("duration", duration),
	)
	return gathering, nil
}

// CollectGathering 领取灵气汇聚收益
func (s *DongFuService) CollectGathering(ctx context.Context, playerID int64) (*model.SpiritGathering, error) {
	gathering, err := s.dongfuRepo.GetActiveSpiritGathering(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询灵气汇聚失败: %w", err)
	}
	if gathering == nil {
		return nil, fmt.Errorf("没有进行中的灵气汇聚")
	}

	dongfu, err := s.dongfuRepo.GetDongFuByID(gathering.DongFuID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		return nil, fmt.Errorf("洞府不存在")
	}

	// 计算收益
	elapsed := int(time.Since(gathering.StartTime).Seconds())
	if elapsed > gathering.Duration {
		elapsed = gathering.Duration
	}

	// 基础修为 = 时间(秒) * 基础值 * (1 + 修炼加成/100)
	cultivationBonus := 1.0 + dongfu.CultivationBonus/100.0
	earnedCultivation := float64(elapsed) * GatheringBasePerSecond * cultivationBonus

	gathering.ElapsedSeconds = elapsed
	gathering.BonusCultivation += earnedCultivation
	gathering.Status = model.GatheringStatusIdle

	// 给玩家加修为
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}

	// 将修为转化为玩家经验
	player.Experience += int64(earnedCultivation)

	if err := s.dongfuRepo.UpdateSpiritGathering(gathering); err != nil {
		return nil, err
	}
	if err := s.playerRepo.Update(player); err != nil {
		return nil, fmt.Errorf("增加修为失败: %w", err)
	}

	s.log.Info("领取灵气汇聚",
		zap.Int64("player_id", playerID),
		zap.Float64("earned", earnedCultivation),
		zap.Int("elapsed_seconds", elapsed),
	)
	return gathering, nil
}

// GetGatheringStatus 获取灵气汇聚状态
func (s *DongFuService) GetGatheringStatus(ctx context.Context, playerID int64) (*model.SpiritGathering, error) {
	gathering, err := s.dongfuRepo.GetActiveSpiritGathering(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询灵气汇聚失败: %w", err)
	}
	if gathering == nil {
		return nil, nil
	}

	elapsed := int(time.Since(gathering.StartTime).Seconds())
	if elapsed > gathering.Duration {
		elapsed = gathering.Duration
	}
	gathering.ElapsedSeconds = elapsed

	return gathering, nil
}

// ==================== 装饰系统 ====================

// PlaceDecoration 摆放装饰
func (s *DongFuService) PlaceDecoration(ctx context.Context, playerID int64, req model.PlaceDecorationRequest) (*model.Decoration, error) {
	dongfu, err := s.dongfuRepo.GetDongFuByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		return nil, fmt.Errorf("请先建造洞府")
	}

	// 检查洞府等级是否达到解锁装饰的条件
	if dongfu.Level < DongFuLevelThresholdDecorate {
		return nil, fmt.Errorf("洞府等级%d以上解锁装饰功能，当前%d级", DongFuLevelThresholdDecorate, dongfu.Level)
	}

	// 装饰可带来微小的加成
	decoration := &model.Decoration{
		DongFuID:       dongfu.ID,
		PlayerID:       playerID,
		ItemID:         req.ItemID,
		Name:           req.Name,
		DecorationType: req.DecorationType,
		BonusType:      s.decorationBonusType(req.DecorationType),
		BonusValue:     s.decorationBonusValue(req.DecorationType),
		Description:    s.decorationDescription(req.DecorationType),
		IsPlaced:       true,
		PositionX:      req.PositionX,
		PositionY:      req.PositionY,
	}

	if err := s.dongfuRepo.CreateDecoration(decoration); err != nil {
		return nil, err
	}

	s.log.Info("摆放装饰成功",
		zap.Int64("player_id", playerID),
		zap.String("name", req.Name),
	)
	return decoration, nil
}

// RemoveDecoration 移除装饰
func (s *DongFuService) RemoveDecoration(ctx context.Context, playerID int64, decorationID int64) error {
	decoration, err := s.dongfuRepo.GetDecorationByID(decorationID)
	if err != nil {
		return fmt.Errorf("查询装饰失败: %w", err)
	}
	if decoration == nil {
		return fmt.Errorf("装饰不存在")
	}

	dongfu, err := s.dongfuRepo.GetDongFuByID(decoration.DongFuID)
	if err != nil {
		return fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil || dongfu.PlayerID != playerID {
		return fmt.Errorf("无权操作此装饰")
	}

	if err := s.dongfuRepo.RemoveDecoration(decorationID); err != nil {
		return err
	}

	s.log.Info("移除装饰成功",
		zap.Int64("player_id", playerID),
		zap.Int64("decoration_id", decorationID),
	)
	return nil
}

// ListDecorations 获取装饰列表
func (s *DongFuService) ListDecorations(ctx context.Context, playerID int64) ([]model.Decoration, error) {
	dongfu, err := s.dongfuRepo.GetDongFuByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		return nil, fmt.Errorf("请先建造洞府")
	}

	decorations, err := s.dongfuRepo.GetDecorationsByDongFuID(dongfu.ID)
	if err != nil {
		return nil, fmt.Errorf("查询装饰列表失败: %w", err)
	}
	return decorations, nil
}

// ==================== 访客系统 ====================

// InviteGuest 邀请访客
func (s *DongFuService) InviteGuest(ctx context.Context, hostPlayerID int64, guestPlayerID int64) (*model.Guest, error) {
	dongfu, err := s.dongfuRepo.GetDongFuByPlayerID(hostPlayerID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		return nil, fmt.Errorf("请先建造洞府")
	}

	// 检查洞府等级
	if dongfu.Level < DongFuLevelThresholdGuests {
		return nil, fmt.Errorf("洞府等级%d以上解锁访客功能，当前%d级", DongFuLevelThresholdGuests, dongfu.Level)
	}

	// 不能邀请自己
	if hostPlayerID == guestPlayerID {
		return nil, fmt.Errorf("不能邀请自己")
	}

	// 检查访客是否存在
	guestPlayer, err := s.playerRepo.GetByID(guestPlayerID)
	if err != nil {
		return nil, fmt.Errorf("查询访客玩家失败: %w", err)
	}
	if guestPlayer == nil {
		return nil, fmt.Errorf("访客玩家不存在")
	}

	// 检查是否已有邀请
	existing, err := s.dongfuRepo.GetGuestByPlayers(dongfu.ID, guestPlayerID)
	if err != nil {
		return nil, fmt.Errorf("查询已有邀请失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("已向该玩家发送过邀请")
	}

	// 计算互惠加成
	hostBonusType := "cultivation"
	hostBonusValue := GuestBonusBase + float64(dongfu.Level)*0.5
	guestBonusType := "cultivation"
	guestBonusValue := GuestBonusBase

	guest := &model.Guest{
		DongFuID:       dongfu.ID,
		GuestPlayerID:  guestPlayerID,
		HostPlayerID:   hostPlayerID,
		Status:         "pending",
		HostBonusType:  hostBonusType,
		HostBonusValue: hostBonusValue,
		GuestBonusType: guestBonusType,
		GuestBonusValue: guestBonusValue,
	}

	if err := s.dongfuRepo.CreateGuest(guest); err != nil {
		return nil, err
	}

	s.log.Info("邀请访客成功",
		zap.Int64("host_id", hostPlayerID),
		zap.Int64("guest_id", guestPlayerID),
	)
	return guest, nil
}

// GuestAction 处理访客邀请（接受/拒绝/结束拜访）
func (s *DongFuService) GuestAction(ctx context.Context, playerID int64, req model.GuestActionRequest) (*model.Guest, error) {
	guest, err := s.dongfuRepo.GetGuestByID(req.GuestID)
	if err != nil {
		return nil, fmt.Errorf("查询访客记录失败: %w", err)
	}
	if guest == nil {
		return nil, fmt.Errorf("访客记录不存在")
	}

	now := time.Now()

	switch req.Action {
	case "accept":
		if guest.GuestPlayerID != playerID {
			return nil, fmt.Errorf("无权操作")
		}
		if guest.Status != "pending" {
			return nil, fmt.Errorf("该邀请已处理")
		}
		guest.Status = "visiting"
		guest.VisitStart = &now

	case "reject":
		if guest.GuestPlayerID != playerID {
			return nil, fmt.Errorf("无权操作")
		}
		if guest.Status != "pending" {
			return nil, fmt.Errorf("该邀请已处理")
		}
		guest.Status = "completed"

	case "complete":
		// 房主或访客均可结束拜访
		if guest.HostPlayerID != playerID && guest.GuestPlayerID != playerID {
			return nil, fmt.Errorf("无权操作")
		}
		if guest.Status != "visiting" {
			return nil, fmt.Errorf("当前没有进行中的拜访")
		}
		guest.Status = "completed"
		guest.VisitEnd = &now

		// 结束拜访时双方获得加成奖励
		s.applyGuestVisitReward(guest)

	default:
		return nil, fmt.Errorf("未知操作: %s", req.Action)
	}

	if err := s.dongfuRepo.UpdateGuest(guest); err != nil {
		return nil, err
	}

	return guest, nil
}

// applyGuestVisitReward 处理拜访结束时的双方奖励
func (s *DongFuService) applyGuestVisitReward(g *model.Guest) {
	// 房主获得灵气值
	dongfu, err := s.dongfuRepo.GetDongFuByID(g.DongFuID)
	if err == nil && dongfu != nil {
		dongfu.SpiritEnergy += g.HostBonusValue
		_ = s.dongfuRepo.UpdateDongFu(dongfu)
	}

	// 访客获得灵石奖励
	guestPlayer, err := s.playerRepo.GetByID(g.GuestPlayerID)
	if err == nil && guestPlayer != nil {
		goldReward := int64(g.GuestBonusValue * 10)
		guestPlayer.Gold += goldReward
		_ = s.playerRepo.Update(guestPlayer)
	}
}

// GetGuests 获取洞府访客列表
func (s *DongFuService) GetGuests(ctx context.Context, playerID int64) ([]model.Guest, error) {
	dongfu, err := s.dongfuRepo.GetDongFuByPlayerID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		return nil, fmt.Errorf("请先建造洞府")
	}

	guests, err := s.dongfuRepo.GetGuestsByDongFuID(dongfu.ID)
	if err != nil {
		return nil, fmt.Errorf("查询访客列表失败: %w", err)
	}

	// 填充名称
	for i := range guests {
		if guests[i].GuestPlayerID > 0 {
			gp, _ := s.playerRepo.GetByID(guests[i].GuestPlayerID)
			if gp != nil {
				guests[i].GuestName = gp.Name
				guests[i].GuestLevel = int(gp.Level)
			}
		}
		if guests[i].HostPlayerID > 0 {
			hp, _ := s.playerRepo.GetByID(guests[i].HostPlayerID)
			if hp != nil {
				guests[i].HostName = hp.Name
			}
		}
	}

	return guests, nil
}

// GetInvitations 获取玩家收到的洞府邀请
func (s *DongFuService) GetInvitations(ctx context.Context, playerID int64) ([]model.Guest, error) {
	invitations, err := s.dongfuRepo.GetGuestInvitations(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询邀请列表失败: %w", err)
	}

	for i := range invitations {
		if invitations[i].HostPlayerID > 0 {
			hp, _ := s.playerRepo.GetByID(invitations[i].HostPlayerID)
			if hp != nil {
				invitations[i].HostName = hp.Name
			}
		}
	}

	return invitations, nil
}

// ==================== 洞府被动收益计算 ====================

// GetPassiveRewards 计算洞府被动收益（演武场战斗经验 + 灵泉灵石）
// 按小时产出计算
func (s *DongFuService) GetPassiveRewards(ctx context.Context, playerID int64) (combatExpPerHour float64, stonesPerHour float64, err error) {
	dongfu, err := s.dongfuRepo.GetDongFuByPlayerID(playerID)
	if err != nil {
		return 0, 0, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		return 0, 0, fmt.Errorf("请先建造洞府")
	}

	return dongfu.CombatExpPerHour, dongfu.SpiritStonesPerHour, nil
}

// CollectPassiveRewards 领取被动收益（按实际时间计算）
func (s *DongFuService) CollectPassiveRewards(ctx context.Context, playerID int64) (combatExp int64, stones int64, err error) {
	dongfu, err := s.dongfuRepo.GetDongFuByPlayerID(playerID)
	if err != nil {
		return 0, 0, fmt.Errorf("查询洞府失败: %w", err)
	}
	if dongfu == nil {
		return 0, 0, fmt.Errorf("请先建造洞府")
	}

	// 基于上次领取时间（简化：按小时产出折算）
	// 实际生产中需要 last_collected_at 字段
	hours := 1.0 // 简化处理，每小时可领一次
	combatExp = int64(dongfu.CombatExpPerHour * hours)
	stones = int64(dongfu.SpiritStonesPerHour * hours)

	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return 0, 0, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return 0, 0, fmt.Errorf("玩家不存在")
	}

	player.Experience += combatExp
	player.Gold += stones
	if err := s.playerRepo.Update(player); err != nil {
		return 0, 0, fmt.Errorf("领取被动收益失败: %w", err)
	}

	return combatExp, stones, nil
}

// ==================== 辅助方法 ====================

// recalcDongFuBonuses 重新计算洞府总加成
func (s *DongFuService) recalcDongFuBonuses(dongfu *model.DongFu) {
	rooms, err := s.dongfuRepo.GetRoomsByDongFuID(dongfu.ID)
	if err != nil {
		s.log.Error("查询房间列表失败", zap.Error(err))
		return
	}

	var cultBonus, alchemyBonus, storageBonus, combatExp, spiritStones float64
	totalLevel := 0

	for _, r := range rooms {
		totalLevel += r.Level
		switch r.RoomType {
		case model.RoomTypeTraining:
			cultBonus += r.Bonus
		case model.RoomTypeAlchemy:
			alchemyBonus += r.Bonus
		case model.RoomTypeTreasure:
			storageBonus += r.Bonus
		case model.RoomTypeArena:
			combatExp += r.Bonus
		case model.RoomTypeSpring:
			spiritStones += r.Bonus
		}
	}

	dongfu.CultivationBonus = cultBonus
	dongfu.AlchemyBonus = alchemyBonus
	dongfu.StorageBonus = storageBonus
	dongfu.CombatExpPerHour = combatExp
	dongfu.SpiritStonesPerHour = spiritStones
	dongfu.Level = totalLevel

	if err := s.dongfuRepo.UpdateDongFu(dongfu); err != nil {
		s.log.Error("更新洞府加成失败", zap.Error(err))
	}
}

// buildEffectDesc 生成房间效果描述
func (s *DongFuService) buildEffectDesc(roomType int, bonus float64) string {
	tmpl, ok := model.RoomTypeEffectDesc[roomType]
	if !ok {
		return ""
	}
	return fmt.Sprintf(tmpl, bonus)
}

// ToRoomDetail 构建房间详情（含前端所需信息）
func (s *DongFuService) ToRoomDetail(room *model.Room) *model.RoomDetail {
	if room == nil {
		return nil
	}

	maxLv := model.RoomTypeMaxLevel[room.RoomType]
	bonusPerLv := model.RoomTypeBonusPerLevel[room.RoomType]
	nextBonus := bonusPerLv * float64(room.Level+1)

	if room.Level >= maxLv {
		nextBonus = room.Bonus
	}

	return &model.RoomDetail{
		Room:         *room,
		Icon:         model.RoomTypeIcon[room.RoomType],
		Description:  model.RoomTypeDetailDesc[room.RoomType],
		MaxLevel:     maxLv,
		BonusPerLevel: bonusPerLv,
		NextBonus:    nextBonus,
		BuildCost:    RoomBuildCost,
		UpgradeCost:  RoomUpgradeCostBase * int64(room.Level+1),
		IsMaxLevel:   room.Level >= maxLv,
	}
}

// GetRoomUpgradeCost 获取升级到下一级所需灵石
func (s *DongFuService) GetRoomUpgradeCost(room *model.Room) int64 {
	return RoomUpgradeCostBase * int64(room.Level+1)
}

// decorationBonusType 根据装饰类型返回加成类型
func (s *DongFuService) decorationBonusType(dType int) string {
	switch dType {
	case 0:
		return "cultivation"
	case 1:
		return "alchemy"
	case 2:
		return "defense"
	case 3:
		return "cultivation"
	case 4:
		return "spirit_energy"
	default:
		return "cultivation"
	}
}

// decorationBonusValue 根据装饰类型返回加成值
func (s *DongFuService) decorationBonusValue(dType int) float64 {
	switch dType {
	case 0:
		return 0.5 // 装饰：修炼+0.5%
	case 1:
		return 0.5 // 家具：炼丹+0.5%
	case 2:
		return 0.3 // 盆景：防御+0.3%
	case 3:
		return 0.4 // 挂画：修炼+0.4%
	case 4:
		return 1.0 // 奇石：灵气+1
	default:
		return 0.2
	}
}

// decorationDescription 装饰描述
func (s *DongFuService) decorationDescription(dType int) string {
	switch dType {
	case 0:
		return "普通装饰物，为洞府增添雅致"
	case 1:
		return "精美家具，提升炼丹效率"
	case 2:
		return "珍稀盆景，涵养灵气"
	case 3:
		return "名家挂画，蕴藏天道至理"
	case 4:
		return "天地奇石，汇聚八方灵气"
	default:
		return "未知装饰"
	}
}

// CalculateGatheringReward 计算灵气汇聚当前预收益（不实际发放）
func (s *DongFuService) CalculateGatheringReward(gathering *model.SpiritGathering, cultBonus float64) float64 {
	elapsed := int(time.Since(gathering.StartTime).Seconds())
	if elapsed > gathering.Duration {
		elapsed = gathering.Duration
	}
	bonusFactor := 1.0 + cultBonus/100.0
	return float64(elapsed) * GatheringBasePerSecond * bonusFactor
}

// GetRequiredLevel 获取洞府解锁等级要求
func (s *DongFuService) GetRequiredLevel() int {
	return DongFuRequiredLevel
}

// GetBuildCost returns the base build cost.
func (s *DongFuService) GetBuildCost() int64 {
	return DongFuBuildCostBase
}

// GetRoomBuildCost returns the room build cost.
func (s *DongFuService) GetRoomBuildCost() int64 {
	return RoomBuildCost
}

// GetRoomUpgradeCostBase returns the base upgrade cost multiplier.
func (s *DongFuService) GetRoomUpgradeCostBase() int64 {
	return RoomUpgradeCostBase
}

// GetDongFuLevelThresholds returns the level thresholds for feature unlocks.
func (s *DongFuService) GetDongFuLevelThresholds() map[string]int {
	return map[string]int{
		"guests":   DongFuLevelThresholdGuests,
		"decorate": DongFuLevelThresholdDecorate,
	}
}

// FormatDuration formats seconds into a human-readable string.
func FormatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%d分%d秒", seconds/60, seconds%60)
	} else if seconds < 86400 {
		h := seconds / 3600
		m := (seconds % 3600) / 60
		return fmt.Sprintf("%d时%d分", h, m)
	}
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	return fmt.Sprintf("%d天%d时", d, h)
}
