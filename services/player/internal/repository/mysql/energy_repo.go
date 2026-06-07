package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// EnergyRepo 玩家能量数据访问
type EnergyRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewEnergyRepo 创建 EnergyRepo
func NewEnergyRepo(db *sql.DB, log *zap.Logger) *EnergyRepo {
	return &EnergyRepo{db: db, log: log}
}

// GetByPlayerID 根据玩家ID查询能量记录
func (r *EnergyRepo) GetByPlayerID(playerID int64) (*model.PlayerEnergy, error) {
	query := `SELECT id, player_id, current_energy, max_energy,
		last_meditation_at, energy_pills_used_today, created_at, updated_at
		FROM player_energy WHERE player_id = ?`

	e := &model.PlayerEnergy{}
	var lastMeditationAt sql.NullTime
	err := r.db.QueryRow(query, playerID).Scan(
		&e.ID, &e.PlayerID, &e.CurrentEnergy, &e.MaxEnergy,
		&lastMeditationAt, &e.EnergyPillsUsedToday, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询玩家能量失败", zap.Error(err))
		return nil, fmt.Errorf("查询玩家能量失败: %w", err)
	}
	if lastMeditationAt.Valid {
		e.LastMeditationAt = &lastMeditationAt.Time
	}
	return e, nil
}

// Create 创建玩家能量记录
func (r *EnergyRepo) Create(e *model.PlayerEnergy) error {
	query := `INSERT INTO player_energy (player_id, current_energy, max_energy,
		last_meditation_at, energy_pills_used_today, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now

	result, err := r.db.Exec(query,
		e.PlayerID, e.CurrentEnergy, e.MaxEnergy,
		e.LastMeditationAt, e.EnergyPillsUsedToday, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		r.log.Error("创建玩家能量记录失败", zap.Error(err))
		return fmt.Errorf("创建玩家能量记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增ID失败: %w", err)
	}
	e.ID = id
	return nil
}

// Update 更新能量记录
func (r *EnergyRepo) Update(e *model.PlayerEnergy) error {
	query := `UPDATE player_energy SET current_energy=?, max_energy=?,
		last_meditation_at=?, energy_pills_used_today=?, updated_at=?
		WHERE player_id=?`

	e.UpdatedAt = time.Now()

	_, err := r.db.Exec(query,
		e.CurrentEnergy, e.MaxEnergy,
		e.LastMeditationAt, e.EnergyPillsUsedToday, e.UpdatedAt,
		e.PlayerID,
	)
	if err != nil {
		r.log.Error("更新玩家能量失败", zap.Error(err))
		return fmt.Errorf("更新玩家能量失败: %w", err)
	}
	return nil
}

// UpdateEnergy 仅更新能量值和最后修炼时间（轻量更新）
func (r *EnergyRepo) UpdateEnergy(playerID int64, currentEnergy int, lastMeditationAt *time.Time) error {
	query := `UPDATE player_energy SET current_energy=?, last_meditation_at=?, updated_at=? WHERE player_id=?`
	_, err := r.db.Exec(query, currentEnergy, lastMeditationAt, time.Now(), playerID)
	if err != nil {
		r.log.Error("更新玩家能量值失败", zap.Error(err))
		return fmt.Errorf("更新玩家能量值失败: %w", err)
	}
	return nil
}

// UseEnergy 扣除玩家能量（原子操作，检查并扣减）
func (r *EnergyRepo) UseEnergy(playerID int64, amount int) (*model.PlayerEnergy, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 锁行查询当前能量
	query := `SELECT id, player_id, current_energy, max_energy,
		last_meditation_at, energy_pills_used_today, created_at, updated_at
		FROM player_energy WHERE player_id = ? FOR UPDATE`

	e := &model.PlayerEnergy{}
	var lastMeditationAt sql.NullTime
	err = tx.QueryRow(query, playerID).Scan(
		&e.ID, &e.PlayerID, &e.CurrentEnergy, &e.MaxEnergy,
		&lastMeditationAt, &e.EnergyPillsUsedToday, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("玩家能量记录不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("查询玩家能量失败: %w", err)
	}
	if lastMeditationAt.Valid {
		e.LastMeditationAt = &lastMeditationAt.Time
	}

	if e.CurrentEnergy < amount {
		return nil, fmt.Errorf("体力不足，当前 %d，需要 %d", e.CurrentEnergy, amount)
	}

	e.CurrentEnergy -= amount
	e.UpdatedAt = time.Now()

	updateQuery := `UPDATE player_energy SET current_energy=?, updated_at=? WHERE player_id=?`
	_, err = tx.Exec(updateQuery, e.CurrentEnergy, e.UpdatedAt, playerID)
	if err != nil {
		return nil, fmt.Errorf("扣除能量失败: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return e, nil
}

// UpdatePillCount 更新丹药使用次数
func (r *EnergyRepo) UpdatePillCount(playerID int64, count int) error {
	query := `UPDATE player_energy SET energy_pills_used_today=?, updated_at=? WHERE player_id=?`
	_, err := r.db.Exec(query, count, time.Now(), playerID)
	if err != nil {
		r.log.Error("更新丹药使用次数失败", zap.Error(err))
		return fmt.Errorf("更新丹药使用次数失败: %w", err)
	}
	return nil
}
