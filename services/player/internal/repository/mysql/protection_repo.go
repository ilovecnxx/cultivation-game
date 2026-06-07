package mysql

import (
	"database/sql"
	"fmt"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// ProtectionRepo 新手保护数据访问
type ProtectionRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewProtectionRepo 创建 ProtectionRepo
func NewProtectionRepo(db *sql.DB, log *zap.Logger) *ProtectionRepo {
	return &ProtectionRepo{db: db, log: log}
}

// GetByPlayerID 查询玩家保护记录
func (r *ProtectionRepo) GetByPlayerID(playerID int64) (*model.PlayerProtection, error) {
	query := `SELECT id, player_id, protection_until, pvp_protection_until,
		breakthrough_grace_count, free_resurrection_count, created_at, updated_at
		FROM player_protection WHERE player_id = ?`

	rec := &model.PlayerProtection{}
	var protectionUntil, pvpProtectionUntil sql.NullTime
	err := r.db.QueryRow(query, playerID).Scan(
		&rec.ID, &rec.PlayerID,
		&protectionUntil, &pvpProtectionUntil,
		&rec.BreakthroughGraceCount, &rec.FreeResurrectionCount,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询保护记录失败", zap.Error(err))
		return nil, fmt.Errorf("查询保护记录失败: %w", err)
	}
	if protectionUntil.Valid {
		rec.ProtectionUntil = &protectionUntil.Time
	}
	if pvpProtectionUntil.Valid {
		rec.PvpProtectionUntil = &pvpProtectionUntil.Time
	}
	return rec, nil
}

// Create 创建玩家保护记录
func (r *ProtectionRepo) Create(rec *model.PlayerProtection) error {
	query := `INSERT INTO player_protection (player_id, protection_until, pvp_protection_until,
		breakthrough_grace_count, free_resurrection_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())`

	var protectionUntil, pvpProtectionUntil interface{}
	if rec.ProtectionUntil != nil {
		protectionUntil = *rec.ProtectionUntil
	}
	if rec.PvpProtectionUntil != nil {
		pvpProtectionUntil = *rec.PvpProtectionUntil
	}

	_, err := r.db.Exec(query,
		rec.PlayerID, protectionUntil, pvpProtectionUntil,
		rec.BreakthroughGraceCount, rec.FreeResurrectionCount,
	)
	if err != nil {
		r.log.Error("创建保护记录失败", zap.Error(err))
		return fmt.Errorf("创建保护记录失败: %w", err)
	}
	return nil
}

// UpdateGraceCount 更新突破免罚次数
func (r *ProtectionRepo) UpdateGraceCount(playerID int64, count int32) error {
	query := `UPDATE player_protection SET breakthrough_grace_count = ?, updated_at = NOW() WHERE player_id = ?`
	_, err := r.db.Exec(query, count, playerID)
	if err != nil {
		r.log.Error("更新突破免罚次数失败", zap.Error(err))
		return fmt.Errorf("更新突破免罚次数失败: %w", err)
	}
	return nil
}

// UpdateResurrectionCount 更新免费复活次数
func (r *ProtectionRepo) UpdateResurrectionCount(playerID int64, count int32) error {
	query := `UPDATE player_protection SET free_resurrection_count = ?, updated_at = NOW() WHERE player_id = ?`
	_, err := r.db.Exec(query, count, playerID)
	if err != nil {
		r.log.Error("更新免费复活次数失败", zap.Error(err))
		return fmt.Errorf("更新免费复活次数失败: %w", err)
	}
	return nil
}

// ExpireGeneralProtection 使通用保护过期
func (r *ProtectionRepo) ExpireGeneralProtection(playerID int64) error {
	query := `UPDATE player_protection SET protection_until = NULL, updated_at = NOW() WHERE player_id = ?`
	_, err := r.db.Exec(query, playerID)
	if err != nil {
		r.log.Error("过期通用保护失败", zap.Error(err))
		return fmt.Errorf("过期通用保护失败: %w", err)
	}
	return nil
}

// ExpirePvpProtection 使 PVP 保护过期
func (r *ProtectionRepo) ExpirePvpProtection(playerID int64) error {
	query := `UPDATE player_protection SET pvp_protection_until = NULL, updated_at = NOW() WHERE player_id = ?`
	_, err := r.db.Exec(query, playerID)
	if err != nil {
		r.log.Error("过期PVP保护失败", zap.Error(err))
		return fmt.Errorf("过期PVP保护失败: %w", err)
	}
	return nil
}
