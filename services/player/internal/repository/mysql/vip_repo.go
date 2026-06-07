package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// VipRepo VIP数据访问
type VipRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewVipRepo 创建 VipRepo
func NewVipRepo(db *sql.DB, log *zap.Logger) *VipRepo {
	return &VipRepo{db: db, log: log}
}

// GetVipPlayer 获取玩家VIP信息，若不存在则创建默认VIP0记录
func (r *VipRepo) GetVipPlayer(playerID int64) (*model.VipPlayer, error) {
	query := `SELECT id, player_id, vip_level, vip_exp, total_recharge,
		monthly_card_expires_at, monthly_card_type, last_daily_claim_date,
		created_at, updated_at
		FROM vip_players WHERE player_id = ?`

	vp := &model.VipPlayer{}
	var monthlyCardExpiresAt sql.NullTime
	var lastClaimDate sql.NullString

	err := r.db.QueryRow(query, playerID).Scan(
		&vp.ID, &vp.PlayerID, &vp.VipLevel, &vp.VipExp, &vp.TotalRecharge,
		&monthlyCardExpiresAt, &vp.MonthlyCardType, &lastClaimDate,
		&vp.CreatedAt, &vp.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return r.createDefault(playerID)
	}
	if err != nil {
		r.log.Error("查询VIP信息失败", zap.Error(err))
		return nil, fmt.Errorf("查询VIP信息失败: %w", err)
	}

	if monthlyCardExpiresAt.Valid {
		vp.MonthlyCardExpiresAt = &monthlyCardExpiresAt.Time
	}
	if lastClaimDate.Valid {
		vp.LastDailyClaimDate = &lastClaimDate.String
	}
	return vp, nil
}

// createDefault 创建默认VIP0记录
func (r *VipRepo) createDefault(playerID int64) (*model.VipPlayer, error) {
	now := time.Now()
	query := `INSERT INTO vip_players (player_id, vip_level, vip_exp, total_recharge,
		monthly_card_expires_at, monthly_card_type, last_daily_claim_date, created_at, updated_at)
		VALUES (?, 0, 0, 0, NULL, 0, NULL, ?, ?)`

	_, err := r.db.Exec(query, playerID, now, now)
	if err != nil {
		r.log.Error("创建默认VIP记录失败", zap.Error(err))
		return nil, fmt.Errorf("创建默认VIP记录失败: %w", err)
	}

	return &model.VipPlayer{
		PlayerID:  playerID,
		VipLevel:  0,
		VipExp:    0,
		TotalRecharge: 0,
		MonthlyCardType: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateVipPlayer 更新VIP信息
func (r *VipRepo) UpdateVipPlayer(vp *model.VipPlayer) error {
	query := `UPDATE vip_players SET vip_level = ?, vip_exp = ?, total_recharge = ?,
		monthly_card_expires_at = ?, monthly_card_type = ?, last_daily_claim_date = ?
		WHERE player_id = ?`

	_, err := r.db.Exec(query,
		vp.VipLevel, vp.VipExp, vp.TotalRecharge,
		vp.MonthlyCardExpiresAt, vp.MonthlyCardType, vp.LastDailyClaimDate,
		vp.PlayerID,
	)
	if err != nil {
		r.log.Error("更新VIP信息失败", zap.Error(err))
		return fmt.Errorf("更新VIP信息失败: %w", err)
	}
	return nil
}

// AddRechargeRecord 创建充值记录
func (r *VipRepo) AddRechargeRecord(record *model.VipRechargeRecord) error {
	query := `INSERT INTO vip_recharge_records (player_id, amount_jade, amount_rmb, order_id, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	now := time.Now()
	record.CreatedAt = now
	_, err := r.db.Exec(query,
		record.PlayerID, record.AmountJade, record.AmountRmb, record.OrderID, record.Status, now,
	)
	if err != nil {
		r.log.Error("创建充值记录失败", zap.Error(err))
		return fmt.Errorf("创建充值记录失败: %w", err)
	}
	return nil
}

// UpdateRechargeStatus 更新充值记录状态
func (r *VipRepo) UpdateRechargeStatus(orderID string, status int8) error {
	query := `UPDATE vip_recharge_records SET status = ? WHERE order_id = ?`
	_, err := r.db.Exec(query, status, orderID)
	if err != nil {
		r.log.Error("更新充值状态失败", zap.Error(err))
		return fmt.Errorf("更新充值状态失败: %w", err)
	}
	return nil
}

// GetRechargeRecordByOrderID 根据订单号查询充值记录
func (r *VipRepo) GetRechargeRecordByOrderID(orderID string) (*model.VipRechargeRecord, error) {
	query := `SELECT id, player_id, amount_jade, amount_rmb, order_id, status, created_at
		FROM vip_recharge_records WHERE order_id = ?`

	record := &model.VipRechargeRecord{}
	err := r.db.QueryRow(query, orderID).Scan(
		&record.ID, &record.PlayerID, &record.AmountJade, &record.AmountRmb,
		&record.OrderID, &record.Status, &record.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询充值记录失败", zap.Error(err))
		return nil, fmt.Errorf("查询充值记录失败: %w", err)
	}
	return record, nil
}

// GetRechargeHistory 获取充值历史（分页）
func (r *VipRepo) GetRechargeHistory(playerID int64, limit, offset int) ([]*model.VipRechargeRecord, error) {
	query := `SELECT id, player_id, amount_jade, amount_rmb, order_id, status, created_at
		FROM vip_recharge_records WHERE player_id = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, playerID, limit, offset)
	if err != nil {
		r.log.Error("查询充值历史失败", zap.Error(err))
		return nil, fmt.Errorf("查询充值历史失败: %w", err)
	}
	defer rows.Close()

	var records []*model.VipRechargeRecord
	for rows.Next() {
		record := &model.VipRechargeRecord{}
		if err := rows.Scan(
			&record.ID, &record.PlayerID, &record.AmountJade, &record.AmountRmb,
			&record.OrderID, &record.Status, &record.CreatedAt,
		); err != nil {
			r.log.Error("扫描充值记录失败", zap.Error(err))
			return nil, fmt.Errorf("扫描充值记录失败: %w", err)
		}
		records = append(records, record)
	}
	if records == nil {
		records = []*model.VipRechargeRecord{}
	}
	return records, rows.Err()
}
