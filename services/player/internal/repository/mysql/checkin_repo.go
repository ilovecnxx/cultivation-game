package mysql

import (
	"database/sql"
	"fmt"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// CheckinRepo 签到数据访问
type CheckinRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewCheckinRepo 创建 CheckinRepo
func NewCheckinRepo(db *sql.DB, log *zap.Logger) *CheckinRepo {
	return &CheckinRepo{db: db, log: log}
}

// GetByPlayerID 查询玩家签到记录
func (r *CheckinRepo) GetByPlayerID(playerID int64) (*model.CheckinRecord, error) {
	query := `SELECT player_id, last_checkin_date, consecutive_days,
		week_start_date, week_claimed_mask, month_total, month_str,
		month_reward_claimed, makeup_date, makeup_used_today, created_at, updated_at
		FROM checkin_records WHERE player_id = ?`

	rec := &model.CheckinRecord{}
	err := r.db.QueryRow(query, playerID).Scan(
		&rec.PlayerID, &rec.LastCheckinDate, &rec.ConsecutiveDays,
		&rec.WeekStartDate, &rec.WeekClaimedMask, &rec.MonthTotal, &rec.MonthStr,
		&rec.MonthRewardClaimed, &rec.MakeupDate, &rec.MakeupUsedToday, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询签到记录失败", zap.Error(err))
		return nil, fmt.Errorf("查询签到记录失败: %w", err)
	}
	return rec, nil
}

// Upsert 插入或更新签到记录
func (r *CheckinRepo) Upsert(rec *model.CheckinRecord) error {
	query := `INSERT INTO checkin_records (player_id, last_checkin_date, consecutive_days,
		week_start_date, week_claimed_mask, month_total, month_str,
		month_reward_claimed, makeup_date, makeup_used_today, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			last_checkin_date = VALUES(last_checkin_date),
			consecutive_days = VALUES(consecutive_days),
			week_start_date = VALUES(week_start_date),
			week_claimed_mask = VALUES(week_claimed_mask),
			month_total = VALUES(month_total),
			month_str = VALUES(month_str),
			month_reward_claimed = VALUES(month_reward_claimed),
			makeup_date = VALUES(makeup_date),
			makeup_used_today = VALUES(makeup_used_today),
			updated_at = NOW()`

	_, err := r.db.Exec(query,
		rec.PlayerID, rec.LastCheckinDate, rec.ConsecutiveDays,
		rec.WeekStartDate, rec.WeekClaimedMask, rec.MonthTotal, rec.MonthStr,
		rec.MonthRewardClaimed, rec.MakeupDate, rec.MakeupUsedToday,
	)
	if err != nil {
		r.log.Error("保存签到记录失败", zap.Error(err))
		return fmt.Errorf("保存签到记录失败: %w", err)
	}
	return nil
}

// InitRecord 初始化玩家签到记录(首次签到时创建)
func (r *CheckinRepo) InitRecord(playerID int64) error {
	query := `INSERT IGNORE INTO checkin_records (player_id, created_at, updated_at) VALUES (?, NOW(), NOW())`
	_, err := r.db.Exec(query, playerID)
	if err != nil {
		r.log.Error("初始化签到记录失败", zap.Error(err))
		return fmt.Errorf("初始化签到记录失败: %w", err)
	}
	return nil
}
