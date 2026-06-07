// Package repository 提供用户数据的 MySQL 持久化访问。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"cultivation-game/services/auth/internal/model"
)

// UserRepo 用户数据访问对象，封装 users 表的 CRUD 操作。
type UserRepo struct {
	db  *sql.DB
	log *slog.Logger
}

// NewUserRepo 创建 UserRepo。
func NewUserRepo(db *sql.DB, log *slog.Logger) *UserRepo {
	return &UserRepo{db: db, log: log}
}

// Create 插入新用户记录。
func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	query := `INSERT INTO users (username, password_hash, player_id, email, status, last_login_at, last_ip, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, query,
		u.Username, u.PasswordHash, u.PlayerID, u.Email, u.Status,
		u.LastLoginAt, u.LastIP, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		r.log.ErrorContext(ctx, "创建用户失败", "error", err, "username", u.Username)
		return fmt.Errorf("创建用户失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增 ID 失败: %w", err)
	}
	u.ID = uint64(id)
	return nil
}

// GetByID 根据用户 ID 查询用户。
func (r *UserRepo) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	query := `SELECT id, username, password_hash, player_id, email, status, last_login_at, last_ip, created_at, updated_at
		FROM users WHERE id = ?`

	u := &model.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.PlayerID, &u.Email,
		&u.Status, &u.LastLoginAt, &u.LastIP, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.ErrorContext(ctx, "查询用户失败", "error", err, "user_id", id)
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return u, nil
}

// GetByUsername 根据用户名查询用户（登录时使用）。
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `SELECT id, username, password_hash, player_id, email, status, last_login_at, last_ip, created_at, updated_at
		FROM users WHERE username = ?`

	u := &model.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.PlayerID, &u.Email,
		&u.Status, &u.LastLoginAt, &u.LastIP, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.ErrorContext(ctx, "按用户名查询用户失败", "error", err, "username", username)
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return u, nil
}

// UpdateLoginTime 更新用户最后登录时间和 IP。
func (r *UserRepo) UpdateLoginTime(ctx context.Context, id uint64, ip string) error {
	query := `UPDATE users SET last_login_at = ?, last_ip = ?, updated_at = ? WHERE id = ?`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, now, ip, now, id)
	if err != nil {
		r.log.ErrorContext(ctx, "更新登录时间失败", "error", err, "user_id", id)
		return fmt.Errorf("更新登录时间失败: %w", err)
	}
	return nil
}

// UpdatePlayerID 更新用户关联的玩家 ID（注册创建角色后回写）。
func (r *UserRepo) UpdatePlayerID(ctx context.Context, userID, playerID uint64) error {
	query := `UPDATE users SET player_id = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, playerID, time.Now(), userID)
	if err != nil {
		r.log.ErrorContext(ctx, "更新玩家 ID 失败", "error", err, "user_id", userID, "player_id", playerID)
		return fmt.Errorf("更新玩家 ID 失败: %w", err)
	}
	return nil
}

// UpdateStatus 更新用户账号状态（封禁/解封等）。
func (r *UserRepo) UpdateStatus(ctx context.Context, id uint64, status model.UserStatus) error {
	query := `UPDATE users SET status = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		r.log.ErrorContext(ctx, "更新用户状态失败", "error", err, "user_id", id, "status", status)
		return fmt.Errorf("更新用户状态失败: %w", err)
	}
	return nil
}
