package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// ReferralRepo 推荐系统数据访问
type ReferralRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewReferralRepo 创建 ReferralRepo
func NewReferralRepo(db *sql.DB, log *zap.Logger) *ReferralRepo {
	return &ReferralRepo{db: db, log: log}
}

// CreateInviteCode 创建邀请码
func (r *ReferralRepo) CreateInviteCode(code *model.InviteCode) error {
	query := `INSERT INTO player_invite_codes (player_id, invite_code, times_used, created_at) VALUES (?, ?, ?, ?)`
	now := time.Now()
	code.CreatedAt = now

	result, err := r.db.Exec(query, code.PlayerID, code.InviteCode, code.TimesUsed, code.CreatedAt)
	if err != nil {
		r.log.Error("创建邀请码失败", zap.Error(err))
		return fmt.Errorf("创建邀请码失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增ID失败: %w", err)
	}
	code.ID = id
	return nil
}

// GetInviteCodeByPlayerID 根据玩家ID查询邀请码
func (r *ReferralRepo) GetInviteCodeByPlayerID(playerID int64) (*model.InviteCode, error) {
	query := `SELECT id, player_id, invite_code, times_used, created_at FROM player_invite_codes WHERE player_id = ?`
	code := &model.InviteCode{}
	err := r.db.QueryRow(query, playerID).Scan(&code.ID, &code.PlayerID, &code.InviteCode, &code.TimesUsed, &code.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询邀请码失败: %w", err)
	}
	return code, nil
}

// GetInviteCodeByCode 根据邀请码字符串查询
func (r *ReferralRepo) GetInviteCodeByCode(code string) (*model.InviteCode, error) {
	query := `SELECT id, player_id, invite_code, times_used, created_at FROM player_invite_codes WHERE invite_code = ?`
	c := &model.InviteCode{}
	err := r.db.QueryRow(query, code).Scan(&c.ID, &c.PlayerID, &c.InviteCode, &c.TimesUsed, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询邀请码失败: %w", err)
	}
	return c, nil
}

// IncrementInviteCodeUsage 增加邀请码使用次数
func (r *ReferralRepo) IncrementInviteCodeUsage(codeID int64) error {
	query := `UPDATE player_invite_codes SET times_used = times_used + 1 WHERE id = ?`
	_, err := r.db.Exec(query, codeID)
	if err != nil {
		return fmt.Errorf("增加邀请码使用次数失败: %w", err)
	}
	return nil
}

// CreateReferralRecord 创建推荐记录
func (r *ReferralRepo) CreateReferralRecord(record *model.ReferralRecord) error {
	query := `INSERT INTO referral_records (inviter_id, invitee_id, invitee_realm_reached, reward_claimed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now()
	record.CreatedAt = now
	record.UpdatedAt = now

	_, err := r.db.Exec(query, record.InviterID, record.InviteeID, record.InviteeRealmReached, record.RewardClaimed, record.CreatedAt, record.UpdatedAt)
	if err != nil {
		r.log.Error("创建推荐记录失败", zap.Error(err))
		return fmt.Errorf("创建推荐记录失败: %w", err)
	}
	return nil
}

// GetReferralRecordByInviteeID 根据被邀请者ID查询推荐记录
func (r *ReferralRepo) GetReferralRecordByInviteeID(inviteeID int64) (*model.ReferralRecord, error) {
	query := `SELECT id, inviter_id, invitee_id, invitee_realm_reached, reward_claimed, created_at, updated_at
		FROM referral_records WHERE invitee_id = ?`
	record := &model.ReferralRecord{}
	err := r.db.QueryRow(query, inviteeID).Scan(
		&record.ID, &record.InviterID, &record.InviteeID,
		&record.InviteeRealmReached, &record.RewardClaimed,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询推荐记录失败: %w", err)
	}
	return record, nil
}

// GetReferralRecordsByInviterID 查询邀请者的所有推荐记录
func (r *ReferralRepo) GetReferralRecordsByInviterID(inviterID int64) ([]*model.ReferralRecord, error) {
	query := `SELECT id, inviter_id, invitee_id, invitee_realm_reached, reward_claimed, created_at, updated_at
		FROM referral_records WHERE inviter_id = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, inviterID)
	if err != nil {
		return nil, fmt.Errorf("查询推荐列表失败: %w", err)
	}
	defer rows.Close()

	var records []*model.ReferralRecord
	for rows.Next() {
		record := &model.ReferralRecord{}
		err := rows.Scan(
			&record.ID, &record.InviterID, &record.InviteeID,
			&record.InviteeRealmReached, &record.RewardClaimed,
			&record.CreatedAt, &record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描推荐记录失败: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// UpdateReferralRealmReached 更新被邀请者已达成的境界里程碑
func (r *ReferralRepo) UpdateReferralRealmReached(recordID int64, realmReachedBits int8) error {
	query := `UPDATE referral_records SET invitee_realm_reached = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, realmReachedBits, time.Now(), recordID)
	if err != nil {
		return fmt.Errorf("更新推荐记录境界里程碑失败: %w", err)
	}
	return nil
}

// UpdateRewardClaimed 更新已领取奖励的里程碑位
func (r *ReferralRepo) UpdateRewardClaimed(recordID int64, claimedBits int8) error {
	query := `UPDATE referral_records SET reward_claimed = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, claimedBits, time.Now(), recordID)
	if err != nil {
		return fmt.Errorf("更新奖励领取状态失败: %w", err)
	}
	return nil
}
