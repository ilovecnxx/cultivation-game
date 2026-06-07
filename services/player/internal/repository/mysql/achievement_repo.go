package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// AchievementRepo 成就数据访问
type AchievementRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewAchievementRepo 创建 AchievementRepo
func NewAchievementRepo(db *sql.DB, log *zap.Logger) *AchievementRepo {
	return &AchievementRepo{db: db, log: log}
}

// BatchInit 批量初始化玩家成就记录（首次注册时调用）
func (r *AchievementRepo) BatchInit(playerID uint64, achievementIDs []int) error {
	now := time.Now()
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO player_achievements (player_id, achievement_id, progress, completed, claimed, completed_at) VALUES (?, ?, 0, FALSE, FALSE, ?)`)
	if err != nil {
		return fmt.Errorf("预编译插入语句失败: %w", err)
	}
	defer stmt.Close()

	for _, aid := range achievementIDs {
		if _, err := stmt.Exec(playerID, aid, now); err != nil {
			return fmt.Errorf("初始化成就记录失败 player=%d achievement=%d: %w", playerID, aid, err)
		}
	}

	return tx.Commit()
}

// GetByPlayer 获取玩家所有成就进度
func (r *AchievementRepo) GetByPlayer(playerID uint64) ([]*model.PlayerAchievement, error) {
	query := `SELECT player_id, achievement_id, progress, completed, completed_at, claimed FROM player_achievements WHERE player_id = ?`
	rows, err := r.db.Query(query, playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家成就失败: %w", err)
	}
	defer rows.Close()

	var list []*model.PlayerAchievement
	for rows.Next() {
		pa := &model.PlayerAchievement{}
		if err := rows.Scan(&pa.PlayerID, &pa.AchievementID, &pa.Progress, &pa.Completed, &pa.CompletedAt, &pa.Claimed); err != nil {
			return nil, fmt.Errorf("扫描成就记录失败: %w", err)
		}
		list = append(list, pa)
	}
	return list, rows.Err()
}

// GetOne 获取单个成就进度
func (r *AchievementRepo) GetOne(playerID uint64, achievementID int) (*model.PlayerAchievement, error) {
	query := `SELECT player_id, achievement_id, progress, completed, completed_at, claimed FROM player_achievements WHERE player_id = ? AND achievement_id = ?`
	pa := &model.PlayerAchievement{}
	err := r.db.QueryRow(query, playerID, achievementID).Scan(&pa.PlayerID, &pa.AchievementID, &pa.Progress, &pa.Completed, &pa.CompletedAt, &pa.Claimed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询成就失败: %w", err)
	}
	return pa, nil
}

// UpdateProgress 更新成就进度
func (r *AchievementRepo) UpdateProgress(playerID uint64, achievementID int, progress int, completed bool) error {
	now := time.Now()
	query := `UPDATE player_achievements SET progress = ?, completed = ?, completed_at = ? WHERE player_id = ? AND achievement_id = ?`
	_, err := r.db.Exec(query, progress, completed, now, playerID, achievementID)
	if err != nil {
		return fmt.Errorf("更新成就进度失败: %w", err)
	}
	return nil
}

// MarkClaimed 标记成就已领取
func (r *AchievementRepo) MarkClaimed(playerID uint64, achievementID int) error {
	query := `UPDATE player_achievements SET claimed = TRUE WHERE player_id = ? AND achievement_id = ?`
	_, err := r.db.Exec(query, playerID, achievementID)
	if err != nil {
		return fmt.Errorf("标记成就已领取失败: %w", err)
	}
	return nil
}

// GetTitle 获取玩家当前称号
func (r *AchievementRepo) GetTitle(playerID uint64) (*model.PlayerTitle, error) {
	query := `SELECT player_id, title, updated_at FROM player_titles WHERE player_id = ?`
	pt := &model.PlayerTitle{}
	err := r.db.QueryRow(query, playerID).Scan(&pt.PlayerID, &pt.Title, &pt.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询称号失败: %w", err)
	}
	return pt, nil
}

// SaveTitle 保存/更新玩家称号
func (r *AchievementRepo) SaveTitle(playerID uint64, title string) error {
	query := `INSERT INTO player_titles (player_id, title, updated_at) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE title = ?, updated_at = NOW()`
	_, err := r.db.Exec(query, playerID, title, title)
	if err != nil {
		return fmt.Errorf("保存称号失败: %w", err)
	}
	return nil
}
