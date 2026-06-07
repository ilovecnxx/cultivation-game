// Package repository 提供 GM 管理后台的 MySQL 持久化访问。
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"cultivation-game/services/auth/internal/model"
)

// GMRepo GM 管理后台数据访问对象。
type GMRepo struct {
	db  *sql.DB
	log *slog.Logger
}

// NewGMRepo 创建 GMRepo。
func NewGMRepo(db *sql.DB, log *slog.Logger) *GMRepo {
	return &GMRepo{db: db, log: log}
}

// ---- 管理员 ----

// GetAdminByUsername 根据用户名查询管理员。
func (r *GMRepo) GetAdminByUsername(ctx context.Context, username string) (*model.GMAdmin, error) {
	query := `SELECT id, username, password_hash, role, status, last_login_at, created_at, updated_at
		FROM gm_admins WHERE username = ?`
	a := &model.GMAdmin{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.Role, &a.Status, &a.LastLoginAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询管理员失败: %w", err)
	}
	return a, nil
}

// GetAdminByID 根据 ID 查询管理员。
func (r *GMRepo) GetAdminByID(ctx context.Context, id uint64) (*model.GMAdmin, error) {
	query := `SELECT id, username, password_hash, role, status, last_login_at, created_at, updated_at
		FROM gm_admins WHERE id = ?`
	a := &model.GMAdmin{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.Role, &a.Status, &a.LastLoginAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询管理员失败: %w", err)
	}
	return a, nil
}

// UpdateAdminLoginTime 更新管理员最后登录时间。
func (r *GMRepo) UpdateAdminLoginTime(ctx context.Context, id uint64) error {
	query := `UPDATE gm_admins SET last_login_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新管理员登录时间失败: %w", err)
	}
	return nil
}

// CreateAdmin 创建管理员（用于种子数据）。
func (r *GMRepo) CreateAdmin(ctx context.Context, a *model.GMAdmin) error {
	query := `INSERT INTO gm_admins (username, password_hash, role, status, last_login_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	result, err := r.db.ExecContext(ctx, query, a.Username, a.PasswordHash, a.Role, a.Status, a.LastLoginAt, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建管理员失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增 ID 失败: %w", err)
	}
	a.ID = uint64(id)
	return nil
}

// ---- 操作日志 ----

// InsertOperationLog 插入操作日志。
func (r *GMRepo) InsertOperationLog(ctx context.Context, log *model.GMOperationLog) error {
	detailBytes := []byte("null")
	if log.Detail != nil {
		detailBytes = log.Detail
	}
	query := `INSERT INTO gm_operation_logs (admin_id, action_type, target_type, target_id, detail, ip_address, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	log.CreatedAt = now
	result, err := r.db.ExecContext(ctx, query,
		log.AdminID, log.ActionType, log.TargetType, log.TargetID, string(detailBytes), log.IPAddress, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("插入操作日志失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增 ID 失败: %w", err)
	}
	log.ID = uint64(id)
	return nil
}

// GetOperationLogs 查询操作日志（分页，按时间倒序）。
func (r *GMRepo) GetOperationLogs(ctx context.Context, page, limit int) ([]*model.GMOperationLog, int64, error) {
	// 查询总数
	var total int64
	countQuery := `SELECT COUNT(*) FROM gm_operation_logs`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询操作日志总数失败: %w", err)
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	query := `SELECT ol.id, ol.admin_id, ol.action_type, ol.target_type, ol.target_id, ol.detail, ol.ip_address, ol.created_at,
		COALESCE(a.username, '') as admin_name
		FROM gm_operation_logs ol
		LEFT JOIN gm_admins a ON ol.admin_id = a.id
		ORDER BY ol.created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询操作日志失败: %w", err)
	}
	defer rows.Close()

	var items []*model.GMOperationLog
	for rows.Next() {
		item := &model.GMOperationLog{}
		var detailStr string
		if err := rows.Scan(&item.ID, &item.AdminID, &item.ActionType, &item.TargetType, &item.TargetID,
			&detailStr, &item.IPAddress, &item.CreatedAt, &item.AdminName); err != nil {
			return nil, 0, fmt.Errorf("扫描操作日志失败: %w", err)
		}
		item.Detail = json.RawMessage(detailStr)
		items = append(items, item)
	}
	if items == nil {
		items = []*model.GMOperationLog{}
	}
	return items, total, nil
}

// ---- 公告 ----

// InsertAnnouncement 插入公告。
func (r *GMRepo) InsertAnnouncement(ctx context.Context, a *model.GMAnnouncement) error {
	query := `INSERT INTO gm_announcements (admin_id, title, content, type, target_player_id, sent_at, expire_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	a.SentAt = now
	a.CreatedAt = now
	result, err := r.db.ExecContext(ctx, query,
		a.AdminID, a.Title, a.Content, a.Type, a.TargetPlayerID, a.SentAt, a.ExpireAt, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("插入公告失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增 ID 失败: %w", err)
	}
	a.ID = uint64(id)
	return nil
}

// GetAnnouncements 获取公告列表（按时间倒序）。
func (r *GMRepo) GetAnnouncements(ctx context.Context, limit int) ([]*model.GMAnnouncement, error) {
	query := `SELECT a.id, a.admin_id, a.title, a.content, a.type, a.target_player_id, a.sent_at, a.expire_at, a.created_at,
		COALESCE(adm.username, '') as admin_name
		FROM gm_announcements a
		LEFT JOIN gm_admins adm ON a.admin_id = adm.id
		ORDER BY a.sent_at DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("查询公告失败: %w", err)
	}
	defer rows.Close()

	var items []*model.GMAnnouncement
	for rows.Next() {
		item := &model.GMAnnouncement{}
		if err := rows.Scan(&item.ID, &item.AdminID, &item.Title, &item.Content, &item.Type,
			&item.TargetPlayerID, &item.SentAt, &item.ExpireAt, &item.CreatedAt, &item.AdminName); err != nil {
			return nil, fmt.Errorf("扫描公告失败: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []*model.GMAnnouncement{}
	}
	return items, nil
}

// ---- 封禁 ----

// InsertBan 插入封禁记录。
func (r *GMRepo) InsertBan(ctx context.Context, b *model.GMBan) error {
	query := `INSERT INTO gm_bans (player_id, admin_id, reason, ban_type, starts_at, ends_at, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	b.StartsAt = now
	b.CreatedAt = now
	result, err := r.db.ExecContext(ctx, query,
		b.PlayerID, b.AdminID, b.Reason, b.BanType, b.StartsAt, b.EndsAt, b.Status, b.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("插入封禁记录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增 ID 失败: %w", err)
	}
	b.ID = uint64(id)
	return nil
}

// GetActiveBanByPlayer 查询玩家当前生效的封禁记录。
func (r *GMRepo) GetActiveBanByPlayer(ctx context.Context, playerID uint64) (*model.GMBan, error) {
	query := `SELECT id, player_id, admin_id, reason, ban_type, starts_at, ends_at, status, created_at
		FROM gm_bans
		WHERE player_id = ? AND status = 1
		ORDER BY id DESC
		LIMIT 1`
	b := &model.GMBan{}
	err := r.db.QueryRowContext(ctx, query, playerID).Scan(
		&b.ID, &b.PlayerID, &b.AdminID, &b.Reason, &b.BanType, &b.StartsAt, &b.EndsAt, &b.Status, &b.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询封禁记录失败: %w", err)
	}
	return b, nil
}

// DeactivateBan 解除封禁。
func (r *GMRepo) DeactivateBan(ctx context.Context, playerID uint64) error {
	query := `UPDATE gm_bans SET status = 0 WHERE player_id = ? AND status = 1`
	_, err := r.db.ExecContext(ctx, query, playerID)
	if err != nil {
		return fmt.Errorf("解除封禁失败: %w", err)
	}
	return nil
}
