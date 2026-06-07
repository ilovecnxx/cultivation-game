package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// DongFuRepo 洞府数据访问
type DongFuRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewDongFuRepo 创建 DongFuRepo
func NewDongFuRepo(db *sql.DB, log *zap.Logger) *DongFuRepo {
	return &DongFuRepo{db: db, log: log}
}

// ==================== 洞府 ====================

// CreateDongFu 插入洞府记录
func (r *DongFuRepo) CreateDongFu(d *model.DongFu) error {
	query := `INSERT INTO dongfu (player_id, level, name, cultivation_bonus, alchemy_bonus, storage_bonus,
		combat_exp_per_hour, spirit_stones_per_hour, spirit_energy, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now

	result, err := r.db.Exec(query,
		d.PlayerID, d.Level, d.Name, d.CultivationBonus, d.AlchemyBonus,
		d.StorageBonus, d.CombatExpPerHour, d.SpiritStonesPerHour, d.SpiritEnergy,
		d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		r.log.Error("创建洞府失败", zap.Error(err))
		return fmt.Errorf("创建洞府失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取洞府自增ID失败: %w", err)
	}
	d.ID = id
	return nil
}

// GetDongFuByPlayerID 根据玩家ID查询洞府
func (r *DongFuRepo) GetDongFuByPlayerID(playerID int64) (*model.DongFu, error) {
	query := `SELECT id, player_id, level, name, cultivation_bonus, alchemy_bonus, storage_bonus,
		combat_exp_per_hour, spirit_stones_per_hour, spirit_energy, created_at, updated_at
		FROM dongfu WHERE player_id = ?`

	d := &model.DongFu{}
	err := r.db.QueryRow(query, playerID).Scan(
		&d.ID, &d.PlayerID, &d.Level, &d.Name,
		&d.CultivationBonus, &d.AlchemyBonus, &d.StorageBonus,
		&d.CombatExpPerHour, &d.SpiritStonesPerHour, &d.SpiritEnergy,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询洞府失败", zap.Error(err))
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	return d, nil
}

// GetDongFuByID 根据ID查询洞府
func (r *DongFuRepo) GetDongFuByID(id int64) (*model.DongFu, error) {
	query := `SELECT id, player_id, level, name, cultivation_bonus, alchemy_bonus, storage_bonus,
		combat_exp_per_hour, spirit_stones_per_hour, spirit_energy, created_at, updated_at
		FROM dongfu WHERE id = ?`

	d := &model.DongFu{}
	err := r.db.QueryRow(query, id).Scan(
		&d.ID, &d.PlayerID, &d.Level, &d.Name,
		&d.CultivationBonus, &d.AlchemyBonus, &d.StorageBonus,
		&d.CombatExpPerHour, &d.SpiritStonesPerHour, &d.SpiritEnergy,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询洞府失败", zap.Error(err))
		return nil, fmt.Errorf("查询洞府失败: %w", err)
	}
	return d, nil
}

// UpdateDongFu 更新洞府
func (r *DongFuRepo) UpdateDongFu(d *model.DongFu) error {
	query := `UPDATE dongfu SET level=?, name=?, cultivation_bonus=?, alchemy_bonus=?, storage_bonus=?,
		combat_exp_per_hour=?, spirit_stones_per_hour=?, spirit_energy=?, updated_at=? WHERE id=?`

	d.UpdatedAt = time.Now()
	_, err := r.db.Exec(query,
		d.Level, d.Name, d.CultivationBonus, d.AlchemyBonus, d.StorageBonus,
		d.CombatExpPerHour, d.SpiritStonesPerHour, d.SpiritEnergy, d.UpdatedAt, d.ID,
	)
	if err != nil {
		r.log.Error("更新洞府失败", zap.Error(err))
		return fmt.Errorf("更新洞府失败: %w", err)
	}
	return nil
}

// DeleteDongFu 删除洞府
func (r *DongFuRepo) DeleteDongFu(id int64) error {
	if err := r.DeleteRoomsByDongFuID(id); err != nil {
		return err
	}
	// 删除关联的灵气汇聚、装饰、访客记录
	_, _ = r.db.Exec("DELETE FROM dongfu_spirit_gathering WHERE dongfu_id = ?", id)
	_, _ = r.db.Exec("DELETE FROM dongfu_decorations WHERE dongfu_id = ?", id)
	_, _ = r.db.Exec("DELETE FROM dongfu_guests WHERE dongfu_id = ?", id)

	_, err := r.db.Exec("DELETE FROM dongfu WHERE id = ?", id)
	if err != nil {
		r.log.Error("删除洞府失败", zap.Error(err))
		return fmt.Errorf("删除洞府失败: %w", err)
	}
	return nil
}

// ==================== 房间 ====================

// CreateRoom 插入房间记录
func (r *DongFuRepo) CreateRoom(room *model.Room) error {
	query := `INSERT INTO dongfu_rooms (dongfu_id, room_type, level, name, effect, bonus)
		VALUES (?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query,
		room.DongFuID, room.RoomType, room.Level, room.Name, room.Effect, room.Bonus,
	)
	if err != nil {
		r.log.Error("创建房间失败", zap.Error(err))
		return fmt.Errorf("创建房间失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取房间自增ID失败: %w", err)
	}
	room.ID = id
	return nil
}

// GetRoomsByDongFuID 查询洞府的所有房间
func (r *DongFuRepo) GetRoomsByDongFuID(dongfuID int64) ([]model.Room, error) {
	query := `SELECT id, dongfu_id, room_type, level, name, effect, bonus
		FROM dongfu_rooms WHERE dongfu_id = ? ORDER BY room_type`

	rows, err := r.db.Query(query, dongfuID)
	if err != nil {
		r.log.Error("查询房间列表失败", zap.Error(err))
		return nil, fmt.Errorf("查询房间列表失败: %w", err)
	}
	defer rows.Close()

	var rooms []model.Room
	for rows.Next() {
		var room model.Room
		if err := rows.Scan(&room.ID, &room.DongFuID, &room.RoomType, &room.Level,
			&room.Name, &room.Effect, &room.Bonus); err != nil {
			r.log.Error("扫描房间行失败", zap.Error(err))
			return nil, fmt.Errorf("扫描房间行失败: %w", err)
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

// GetRoomByID 根据ID查询房间
func (r *DongFuRepo) GetRoomByID(roomID int64) (*model.Room, error) {
	query := `SELECT id, dongfu_id, room_type, level, name, effect, bonus
		FROM dongfu_rooms WHERE id = ?`

	room := &model.Room{}
	err := r.db.QueryRow(query, roomID).Scan(
		&room.ID, &room.DongFuID, &room.RoomType, &room.Level,
		&room.Name, &room.Effect, &room.Bonus,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询房间失败", zap.Error(err))
		return nil, fmt.Errorf("查询房间失败: %w", err)
	}
	return room, nil
}

// GetRoomByType 根据洞府ID和房间类型查询
func (r *DongFuRepo) GetRoomByType(dongfuID int64, roomType int) (*model.Room, error) {
	query := `SELECT id, dongfu_id, room_type, level, name, effect, bonus
		FROM dongfu_rooms WHERE dongfu_id = ? AND room_type = ?`

	room := &model.Room{}
	err := r.db.QueryRow(query, dongfuID, roomType).Scan(
		&room.ID, &room.DongFuID, &room.RoomType, &room.Level,
		&room.Name, &room.Effect, &room.Bonus,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询房间类型失败", zap.Error(err))
		return nil, fmt.Errorf("查询房间失败: %w", err)
	}
	return room, nil
}

// UpdateRoom 更新房间
func (r *DongFuRepo) UpdateRoom(room *model.Room) error {
	query := `UPDATE dongfu_rooms SET level=?, name=?, effect=?, bonus=? WHERE id=?`
	_, err := r.db.Exec(query,
		room.Level, room.Name, room.Effect, room.Bonus, room.ID,
	)
	if err != nil {
		r.log.Error("更新房间失败", zap.Error(err))
		return fmt.Errorf("更新房间失败: %w", err)
	}
	return nil
}

// DeleteRoomsByDongFuID 删除洞府的所有房间
func (r *DongFuRepo) DeleteRoomsByDongFuID(dongfuID int64) error {
	_, err := r.db.Exec("DELETE FROM dongfu_rooms WHERE dongfu_id = ?", dongfuID)
	if err != nil {
		r.log.Error("删除房间失败", zap.Error(err))
		return fmt.Errorf("删除房间失败: %w", err)
	}
	return nil
}

// ==================== 灵气汇聚 ====================

// CreateSpiritGathering 创建灵气汇聚记录
func (r *DongFuRepo) CreateSpiritGathering(sg *model.SpiritGathering) error {
	query := `INSERT INTO dongfu_spirit_gathering (dongfu_id, player_id, status, start_time, duration, bonus_cultivation, elapsed_seconds, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	sg.CreatedAt = now
	sg.UpdatedAt = now

	result, err := r.db.Exec(query,
		sg.DongFuID, sg.PlayerID, sg.Status, sg.StartTime, sg.Duration,
		sg.BonusCultivation, sg.ElapsedSeconds, sg.CreatedAt, sg.UpdatedAt,
	)
	if err != nil {
		r.log.Error("创建灵气汇聚失败", zap.Error(err))
		return fmt.Errorf("创建灵气汇聚失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取灵气汇聚ID失败: %w", err)
	}
	sg.ID = id
	return nil
}

// GetActiveSpiritGathering 获取活跃的灵气汇聚（按玩家）
func (r *DongFuRepo) GetActiveSpiritGathering(playerID int64) (*model.SpiritGathering, error) {
	query := `SELECT id, dongfu_id, player_id, status, start_time, duration, bonus_cultivation, elapsed_seconds, created_at, updated_at
		FROM dongfu_spirit_gathering WHERE player_id = ? AND status = ? ORDER BY id DESC LIMIT 1`

	sg := &model.SpiritGathering{}
	err := r.db.QueryRow(query, playerID, model.GatheringStatusActive).Scan(
		&sg.ID, &sg.DongFuID, &sg.PlayerID, &sg.Status, &sg.StartTime, &sg.Duration,
		&sg.BonusCultivation, &sg.ElapsedSeconds, &sg.CreatedAt, &sg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询活跃灵气汇聚失败", zap.Error(err))
		return nil, fmt.Errorf("查询灵气汇聚失败: %w", err)
	}
	return sg, nil
}

// UpdateSpiritGathering 更新灵气汇聚
func (r *DongFuRepo) UpdateSpiritGathering(sg *model.SpiritGathering) error {
	query := `UPDATE dongfu_spirit_gathering SET status=?, bonus_cultivation=?, elapsed_seconds=?, updated_at=? WHERE id=?`
	sg.UpdatedAt = time.Now()
	_, err := r.db.Exec(query, sg.Status, sg.BonusCultivation, sg.ElapsedSeconds, sg.UpdatedAt, sg.ID)
	if err != nil {
		r.log.Error("更新灵气汇聚失败", zap.Error(err))
		return fmt.Errorf("更新灵气汇聚失败: %w", err)
	}
	return nil
}

// GetSpiritGatheringByID 根据ID查询灵气汇聚
func (r *DongFuRepo) GetSpiritGatheringByID(id int64) (*model.SpiritGathering, error) {
	query := `SELECT id, dongfu_id, player_id, status, start_time, duration, bonus_cultivation, elapsed_seconds, created_at, updated_at
		FROM dongfu_spirit_gathering WHERE id = ?`

	sg := &model.SpiritGathering{}
	err := r.db.QueryRow(query, id).Scan(
		&sg.ID, &sg.DongFuID, &sg.PlayerID, &sg.Status, &sg.StartTime, &sg.Duration,
		&sg.BonusCultivation, &sg.ElapsedSeconds, &sg.CreatedAt, &sg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询灵气汇聚失败: %w", err)
	}
	return sg, nil
}

// ==================== 装饰 ====================

// CreateDecoration 创建装饰
func (r *DongFuRepo) CreateDecoration(d *model.Decoration) error {
	query := `INSERT INTO dongfu_decorations (dongfu_id, player_id, item_id, name, decoration_type, bonus_type, bonus_value, description, is_placed, position_x, position_y, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	d.CreatedAt = time.Now()
	result, err := r.db.Exec(query,
		d.DongFuID, d.PlayerID, d.ItemID, d.Name, d.DecorationType,
		d.BonusType, d.BonusValue, d.Description, d.IsPlaced,
		d.PositionX, d.PositionY, d.CreatedAt,
	)
	if err != nil {
		r.log.Error("创建装饰失败", zap.Error(err))
		return fmt.Errorf("创建装饰失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取装饰ID失败: %w", err)
	}
	d.ID = id
	return nil
}

// GetDecorationsByDongFuID 获取洞府所有装饰
func (r *DongFuRepo) GetDecorationsByDongFuID(dongfuID int64) ([]model.Decoration, error) {
	query := `SELECT id, dongfu_id, player_id, item_id, name, decoration_type, bonus_type, bonus_value, description, is_placed, position_x, position_y, created_at
		FROM dongfu_decorations WHERE dongfu_id = ? AND is_placed = 1 ORDER BY decoration_type, created_at`

	rows, err := r.db.Query(query, dongfuID)
	if err != nil {
		r.log.Error("查询装饰列表失败", zap.Error(err))
		return nil, fmt.Errorf("查询装饰列表失败: %w", err)
	}
	defer rows.Close()

	var decorations []model.Decoration
	for rows.Next() {
		var d model.Decoration
		if err := rows.Scan(&d.ID, &d.DongFuID, &d.PlayerID, &d.ItemID, &d.Name,
			&d.DecorationType, &d.BonusType, &d.BonusValue, &d.Description,
			&d.IsPlaced, &d.PositionX, &d.PositionY, &d.CreatedAt); err != nil {
			r.log.Error("扫描装饰行失败", zap.Error(err))
			return nil, fmt.Errorf("扫描装饰行失败: %w", err)
		}
		decorations = append(decorations, d)
	}
	return decorations, rows.Err()
}

// GetDecorationByID 根据ID获取装饰
func (r *DongFuRepo) GetDecorationByID(id int64) (*model.Decoration, error) {
	query := `SELECT id, dongfu_id, player_id, item_id, name, decoration_type, bonus_type, bonus_value, description, is_placed, position_x, position_y, created_at
		FROM dongfu_decorations WHERE id = ?`

	d := &model.Decoration{}
	err := r.db.QueryRow(query, id).Scan(
		&d.ID, &d.DongFuID, &d.PlayerID, &d.ItemID, &d.Name,
		&d.DecorationType, &d.BonusType, &d.BonusValue, &d.Description,
		&d.IsPlaced, &d.PositionX, &d.PositionY, &d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询装饰失败: %w", err)
	}
	return d, nil
}

// RemoveDecoration 移除装饰（软删除 - 设为未摆放）
func (r *DongFuRepo) RemoveDecoration(id int64) error {
	_, err := r.db.Exec("UPDATE dongfu_decorations SET is_placed = 0 WHERE id = ?", id)
	if err != nil {
		r.log.Error("移除装饰失败", zap.Error(err))
		return fmt.Errorf("移除装饰失败: %w", err)
	}
	return nil
}

// ==================== 访客 ====================

// CreateGuest 创建访客邀请
func (r *DongFuRepo) CreateGuest(g *model.Guest) error {
	query := `INSERT INTO dongfu_guests (dongfu_id, guest_player_id, host_player_id, status, host_bonus_type, host_bonus_value, guest_bonus_type, guest_bonus_value, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now

	result, err := r.db.Exec(query,
		g.DongFuID, g.GuestPlayerID, g.HostPlayerID, g.Status,
		g.HostBonusType, g.HostBonusValue, g.GuestBonusType, g.GuestBonusValue,
		g.CreatedAt, g.UpdatedAt,
	)
	if err != nil {
		r.log.Error("创建访客邀请失败", zap.Error(err))
		return fmt.Errorf("创建访客邀请失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取访客ID失败: %w", err)
	}
	g.ID = id
	return nil
}

// GetGuestsByDongFuID 获取洞府的所有访客
func (r *DongFuRepo) GetGuestsByDongFuID(dongfuID int64) ([]model.Guest, error) {
	query := `SELECT id, dongfu_id, guest_player_id, host_player_id, status, visit_start, visit_end,
		host_bonus_type, host_bonus_value, guest_bonus_type, guest_bonus_value, created_at, updated_at
		FROM dongfu_guests WHERE dongfu_id = ? AND status != 'completed' ORDER BY created_at DESC`

	rows, err := r.db.Query(query, dongfuID)
	if err != nil {
		r.log.Error("查询访客列表失败", zap.Error(err))
		return nil, fmt.Errorf("查询访客列表失败: %w", err)
	}
	defer rows.Close()

	var guests []model.Guest
	for rows.Next() {
		var g model.Guest
		if err := rows.Scan(&g.ID, &g.DongFuID, &g.GuestPlayerID, &g.HostPlayerID,
			&g.Status, &g.VisitStart, &g.VisitEnd,
			&g.HostBonusType, &g.HostBonusValue, &g.GuestBonusType, &g.GuestBonusValue,
			&g.CreatedAt, &g.UpdatedAt); err != nil {
			r.log.Error("扫描访客行失败", zap.Error(err))
			return nil, fmt.Errorf("扫描访客行失败: %w", err)
		}
		guests = append(guests, g)
	}
	return guests, rows.Err()
}

// GetGuestByID 根据ID获取访客
func (r *DongFuRepo) GetGuestByID(id int64) (*model.Guest, error) {
	query := `SELECT id, dongfu_id, guest_player_id, host_player_id, status, visit_start, visit_end,
		host_bonus_type, host_bonus_value, guest_bonus_type, guest_bonus_value, created_at, updated_at
		FROM dongfu_guests WHERE id = ?`

	g := &model.Guest{}
	err := r.db.QueryRow(query, id).Scan(
		&g.ID, &g.DongFuID, &g.GuestPlayerID, &g.HostPlayerID,
		&g.Status, &g.VisitStart, &g.VisitEnd,
		&g.HostBonusType, &g.HostBonusValue, &g.GuestBonusType, &g.GuestBonusValue,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询访客失败: %w", err)
	}
	return g, nil
}

// GetGuestByPlayers 查询两个玩家之间的访客关系
func (r *DongFuRepo) GetGuestByPlayers(dongfuID, guestPlayerID int64) (*model.Guest, error) {
	query := `SELECT id, dongfu_id, guest_player_id, host_player_id, status, visit_start, visit_end,
		host_bonus_type, host_bonus_value, guest_bonus_type, guest_bonus_value, created_at, updated_at
		FROM dongfu_guests WHERE dongfu_id = ? AND guest_player_id = ? AND status != 'completed' ORDER BY id DESC LIMIT 1`

	g := &model.Guest{}
	err := r.db.QueryRow(query, dongfuID, guestPlayerID).Scan(
		&g.ID, &g.DongFuID, &g.GuestPlayerID, &g.HostPlayerID,
		&g.Status, &g.VisitStart, &g.VisitEnd,
		&g.HostBonusType, &g.HostBonusValue, &g.GuestBonusType, &g.GuestBonusValue,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询访客关系失败: %w", err)
	}
	return g, nil
}

// UpdateGuest 更新访客
func (r *DongFuRepo) UpdateGuest(g *model.Guest) error {
	query := `UPDATE dongfu_guests SET status=?, visit_start=?, visit_end=?, host_bonus_type=?, host_bonus_value=?,
		guest_bonus_type=?, guest_bonus_value=?, updated_at=? WHERE id=?`
	g.UpdatedAt = time.Now()
	_, err := r.db.Exec(query,
		g.Status, g.VisitStart, g.VisitEnd,
		g.HostBonusType, g.HostBonusValue, g.GuestBonusType, g.GuestBonusValue,
		g.UpdatedAt, g.ID,
	)
	if err != nil {
		r.log.Error("更新访客失败", zap.Error(err))
		return fmt.Errorf("更新访客失败: %w", err)
	}
	return nil
}

// GetGuestInvitations 获取玩家收到的邀请（作为访客）
func (r *DongFuRepo) GetGuestInvitations(guestPlayerID int64) ([]model.Guest, error) {
	query := `SELECT id, dongfu_id, guest_player_id, host_player_id, status, visit_start, visit_end,
		host_bonus_type, host_bonus_value, guest_bonus_type, guest_bonus_value, created_at, updated_at
		FROM dongfu_guests WHERE guest_player_id = ? AND status = 'pending' ORDER BY created_at DESC`

	rows, err := r.db.Query(query, guestPlayerID)
	if err != nil {
		r.log.Error("查询邀请列表失败", zap.Error(err))
		return nil, fmt.Errorf("查询邀请列表失败: %w", err)
	}
	defer rows.Close()

	var guests []model.Guest
	for rows.Next() {
		var g model.Guest
		if err := rows.Scan(&g.ID, &g.DongFuID, &g.GuestPlayerID, &g.HostPlayerID,
			&g.Status, &g.VisitStart, &g.VisitEnd,
			&g.HostBonusType, &g.HostBonusValue, &g.GuestBonusType, &g.GuestBonusValue,
			&g.CreatedAt, &g.UpdatedAt); err != nil {
			r.log.Error("扫描邀请行失败", zap.Error(err))
			return nil, fmt.Errorf("扫描邀请行失败: %w", err)
		}
		guests = append(guests, g)
	}
	return guests, rows.Err()
}
