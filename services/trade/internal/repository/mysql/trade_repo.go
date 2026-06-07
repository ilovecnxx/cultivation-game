// Package mysql 提供交易数据的 MySQL 持久化访问，包括挂单、交易、拍卖和玩家资产操作。
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"cultivation-game/services/trade/internal/model"
)

// TradeRepo 交易数据访问对象，封装 trade_listings、trade_transactions、trade_auctions、trade_player_gold 表的 CRUD 操作。
type TradeRepo struct {
	db  *sql.DB
	log *slog.Logger
}

// NewTradeRepo 创建 TradeRepo。
func NewTradeRepo(db *sql.DB, log *slog.Logger) *TradeRepo {
	return &TradeRepo{db: db, log: log}
}

// ============================================================================
// 挂单操作
// ============================================================================

// CreateListing 创建挂单记录。
func (r *TradeRepo) CreateListing(ctx context.Context, l *model.Listing) error {
	query := `INSERT INTO trade_listings (seller_id, seller_name, item_id, item_name, quantity, unit_price, currency_type, status, created_at, expires_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	l.CreatedAt = now
	l.UpdatedAt = now
	if l.Status == "" {
		l.Status = model.ListingStatusActive
	}

	result, err := r.db.ExecContext(ctx, query,
		l.SellerID, l.SellerName, l.ItemID, l.ItemName,
		l.Quantity, l.UnitPrice,
		string(l.CurrencyType), string(l.Status), l.CreatedAt, l.ExpiresAt, l.UpdatedAt,
	)
	if err != nil {
		r.log.ErrorContext(ctx, "创建挂单失败", "error", err, "seller_id", l.SellerID)
		return fmt.Errorf("创建挂单失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增 ID 失败: %w", err)
	}
	l.ID = uint64(id)
	return nil
}

// GetListingByID 根据 ID 查询挂单。
func (r *TradeRepo) GetListingByID(ctx context.Context, id uint64) (*model.Listing, error) {
	query := `SELECT id, seller_id, seller_name, item_id, item_name, quantity, unit_price, currency_type, status, created_at, expires_at, updated_at
		FROM trade_listings WHERE id = ?`

	l := &model.Listing{}
	var sellerName, itemName, currencyType, status string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&l.ID, &l.SellerID, &sellerName, &l.ItemID, &itemName,
		&l.Quantity, &l.UnitPrice, &currencyType, &status,
		&l.CreatedAt, &l.ExpiresAt, &l.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.ErrorContext(ctx, "查询挂单失败", "error", err, "listing_id", id)
		return nil, fmt.Errorf("查询挂单失败: %w", err)
	}
	l.SellerName = sellerName
	l.ItemName = itemName
	l.CurrencyType = model.CurrencyType(currencyType)
	l.Status = model.ListingStatus(status)
	return l, nil
}

// UpdateListingStatus 更新挂单状态（用于取消、售出、过期处理）。
func (r *TradeRepo) UpdateListingStatus(ctx context.Context, id uint64, status model.ListingStatus) error {
	query := `UPDATE trade_listings SET status = ?, updated_at = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, string(status), time.Now(), id)
	if err != nil {
		r.log.ErrorContext(ctx, "更新挂单状态失败", "error", err, "listing_id", id, "status", status)
		return fmt.Errorf("更新挂单状态失败: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("挂单不存在: id=%d", id)
	}
	return nil
}

// ListListings 查询挂单列表，支持分页和筛选。
// 返回挂单列表和总数量。
func (r *TradeRepo) ListListings(ctx context.Context, filter model.ListingFilter) ([]*model.Listing, int, error) {
	// 构建动态 WHERE 条件
	where := "WHERE 1=1"
	args := make([]interface{}, 0)

	if filter.SellerID > 0 {
		where += " AND seller_id = ?"
		args = append(args, filter.SellerID)
	}
	if filter.ItemID > 0 {
		where += " AND item_id = ?"
		args = append(args, filter.ItemID)
	}
	if filter.Status != "" {
		where += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	if filter.CurrencyType != "" {
		where += " AND currency_type = ?"
		args = append(args, string(filter.CurrencyType))
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM trade_listings %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		r.log.ErrorContext(ctx, "查询挂单总数失败", "error", err)
		return nil, 0, fmt.Errorf("查询挂单总数失败: %w", err)
	}

	if total == 0 {
		return []*model.Listing{}, 0, nil
	}

	// 分页查询
	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := fmt.Sprintf(
		"SELECT id, seller_id, seller_name, item_id, item_name, quantity, unit_price, currency_type, status, created_at, expires_at, updated_at FROM trade_listings %s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		where,
	)
	args = append(args, filter.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		r.log.ErrorContext(ctx, "查询挂单列表失败", "error", err)
		return nil, 0, fmt.Errorf("查询挂单列表失败: %w", err)
	}
	defer rows.Close()

	listings := make([]*model.Listing, 0, filter.PageSize)
	for rows.Next() {
		l := &model.Listing{}
		var sellerName, itemName, currencyType, status string
		if err := rows.Scan(
			&l.ID, &l.SellerID, &sellerName, &l.ItemID, &itemName,
			&l.Quantity, &l.UnitPrice, &currencyType, &status,
			&l.CreatedAt, &l.ExpiresAt, &l.UpdatedAt,
		); err != nil {
			r.log.ErrorContext(ctx, "扫描挂单行失败", "error", err)
			return nil, 0, fmt.Errorf("扫描挂单数据失败: %w", err)
		}
		l.SellerName = sellerName
		l.ItemName = itemName
		l.CurrencyType = model.CurrencyType(currencyType)
		l.Status = model.ListingStatus(status)
		listings = append(listings, l)
	}

	return listings, total, nil
}

// ============================================================================
// 交易记录操作
// ============================================================================

// CreateTransaction 创建交易记录。
func (r *TradeRepo) CreateTransaction(ctx context.Context, t *model.Transaction) error {
	query := `INSERT INTO trade_transactions (listing_id, buyer_id, seller_id, item_id, quantity, unit_price, total_price, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	t.CreatedAt = now

	result, err := r.db.ExecContext(ctx, query,
		t.ListingID, t.BuyerID, t.SellerID, t.ItemID,
		t.Quantity, t.UnitPrice, t.TotalPrice, t.CreatedAt,
	)
	if err != nil {
		r.log.ErrorContext(ctx, "创建交易记录失败", "error", err, "listing_id", t.ListingID)
		return fmt.Errorf("创建交易记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取交易自增 ID 失败: %w", err)
	}
	t.ID = uint64(id)
	return nil
}

// ListTransactionsByBuyer 查询买家交易记录。
func (r *TradeRepo) ListTransactionsByBuyer(ctx context.Context, buyerID uint64, page, pageSize int) ([]*model.Transaction, int, error) {
	countQuery := `SELECT COUNT(*) FROM trade_transactions WHERE buyer_id = ?`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, buyerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询交易总数失败: %w", err)
	}

	offset := (page - 1) * pageSize
	dataQuery := `SELECT id, listing_id, buyer_id, seller_id, item_id, quantity, unit_price, total_price, created_at
		FROM trade_transactions WHERE buyer_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, dataQuery, buyerID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询交易记录失败: %w", err)
	}
	defer rows.Close()

	transactions := make([]*model.Transaction, 0, pageSize)
	for rows.Next() {
		t := &model.Transaction{}
		if err := rows.Scan(&t.ID, &t.ListingID, &t.BuyerID, &t.SellerID, &t.ItemID,
			&t.Quantity, &t.UnitPrice, &t.TotalPrice, &t.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描交易行失败: %w", err)
		}
		transactions = append(transactions, t)
	}
	return transactions, total, nil
}

// ListTransactions 通用查询交易记录，支持按买家和卖家筛选，支持分页。
func (r *TradeRepo) ListTransactions(ctx context.Context, buyerID, sellerID uint64, page, pageSize int) ([]*model.Transaction, int, error) {
	where := "WHERE 1=1"
	args := make([]interface{}, 0)

	if buyerID > 0 {
		where += " AND buyer_id = ?"
		args = append(args, buyerID)
	}
	if sellerID > 0 {
		where += " AND seller_id = ?"
		args = append(args, sellerID)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM trade_transactions %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询交易总数失败: %w", err)
	}
	if total == 0 {
		return []*model.Transaction{}, 0, nil
	}

	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(
		"SELECT id, listing_id, buyer_id, seller_id, item_id, quantity, unit_price, total_price, created_at FROM trade_transactions %s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		where,
	)
	queryArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询交易记录失败: %w", err)
	}
	defer rows.Close()

	transactions := make([]*model.Transaction, 0, pageSize)
	for rows.Next() {
		t := &model.Transaction{}
		if err := rows.Scan(&t.ID, &t.ListingID, &t.BuyerID, &t.SellerID, &t.ItemID,
			&t.Quantity, &t.UnitPrice, &t.TotalPrice, &t.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描交易行失败: %w", err)
		}
		transactions = append(transactions, t)
	}
	return transactions, total, nil
}

// ============================================================================
// 拍卖操作
// ============================================================================

// CreateAuction 创建拍卖记录。
func (r *TradeRepo) CreateAuction(ctx context.Context, a *model.Auction) error {
	query := `INSERT INTO trade_auctions (item_id, seller_id, current_bid, bidder_id, reserve_price, end_time, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.Status == "" {
		a.Status = model.AuctionStatusActive
	}

	result, err := r.db.ExecContext(ctx, query,
		a.ItemID, a.SellerID, a.CurrentBid, a.BidderID,
		a.ReservePrice, a.EndTime, string(a.Status), a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		r.log.ErrorContext(ctx, "创建拍卖失败", "error", err, "seller_id", a.SellerID)
		return fmt.Errorf("创建拍卖失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取拍卖自增 ID 失败: %w", err)
	}
	a.ID = uint64(id)
	return nil
}

// GetAuctionByID 根据 ID 查询拍卖。
func (r *TradeRepo) GetAuctionByID(ctx context.Context, id uint64) (*model.Auction, error) {
	query := `SELECT id, item_id, seller_id, current_bid, bidder_id, reserve_price, end_time, status, created_at, updated_at
		FROM trade_auctions WHERE id = ?`

	a := &model.Auction{}
	var status string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID, &a.ItemID, &a.SellerID, &a.CurrentBid, &a.BidderID,
		&a.ReservePrice, &a.EndTime, &status, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.ErrorContext(ctx, "查询拍卖失败", "error", err, "auction_id", id)
		return nil, fmt.Errorf("查询拍卖失败: %w", err)
	}
	a.Status = model.AuctionStatus(status)
	return a, nil
}

// UpdateAuctionBid 更新拍卖当前出价和出价者（使用乐观锁确保并发安全）。
func (r *TradeRepo) UpdateAuctionBid(ctx context.Context, id uint64, newBid uint64, bidderID uint64, oldBid uint64) error {
	query := `UPDATE trade_auctions SET current_bid = ?, bidder_id = ?, updated_at = ? WHERE id = ? AND current_bid = ?`
	result, err := r.db.ExecContext(ctx, query, newBid, bidderID, time.Now(), id, oldBid)
	if err != nil {
		r.log.ErrorContext(ctx, "更新拍卖出价失败", "error", err, "auction_id", id)
		return fmt.Errorf("更新拍卖出价失败: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("出价并发冲突或拍卖已变更: auction_id=%d", id)
	}
	return nil
}

// UpdateAuctionStatus 更新拍卖状态。
func (r *TradeRepo) UpdateAuctionStatus(ctx context.Context, id uint64, status model.AuctionStatus) error {
	query := `UPDATE trade_auctions SET status = ?, updated_at = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, string(status), time.Now(), id)
	if err != nil {
		r.log.ErrorContext(ctx, "更新拍卖状态失败", "error", err, "auction_id", id, "status", status)
		return fmt.Errorf("更新拍卖状态失败: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("拍卖不存在: id=%d", id)
	}
	return nil
}

// ListActiveAuctions 查询活跃拍卖列表，支持分页和物品筛选。
func (r *TradeRepo) ListActiveAuctions(ctx context.Context, filter model.AuctionFilter) ([]*model.Auction, int, error) {
	where := "WHERE status = ?"
	args := []interface{}{string(model.AuctionStatusActive)}

	if filter.ItemID > 0 {
		where += " AND item_id = ?"
		args = append(args, filter.ItemID)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM trade_auctions %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询拍卖总数失败: %w", err)
	}
	if total == 0 {
		return []*model.Auction{}, 0, nil
	}

	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := fmt.Sprintf(
		"SELECT id, item_id, seller_id, current_bid, bidder_id, reserve_price, end_time, status, created_at, updated_at FROM trade_auctions %s ORDER BY end_time ASC LIMIT ? OFFSET ?",
		where,
	)
	queryArgs := append(args, filter.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, queryArgs...)
	if err != nil {
		r.log.ErrorContext(ctx, "查询活跃拍卖列表失败", "error", err)
		return nil, 0, fmt.Errorf("查询活跃拍卖列表失败: %w", err)
	}
	defer rows.Close()

	auctions := make([]*model.Auction, 0, filter.PageSize)
	for rows.Next() {
		a := &model.Auction{}
		var status string
		if err := rows.Scan(
			&a.ID, &a.ItemID, &a.SellerID, &a.CurrentBid, &a.BidderID,
			&a.ReservePrice, &a.EndTime, &status, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描拍卖行失败: %w", err)
		}
		a.Status = model.AuctionStatus(status)
		auctions = append(auctions, a)
	}
	return auctions, total, nil
}

// FindExpiredAuctions 查找所有已过期但状态仍为 active 的拍卖。
func (r *TradeRepo) FindExpiredAuctions(ctx context.Context) ([]*model.Auction, error) {
	query := `SELECT id, item_id, seller_id, current_bid, bidder_id, reserve_price, end_time, status, created_at, updated_at
		FROM trade_auctions WHERE status = ? AND end_time <= ?`

	rows, err := r.db.QueryContext(ctx, query, string(model.AuctionStatusActive), time.Now())
	if err != nil {
		r.log.ErrorContext(ctx, "查询过期拍卖失败", "error", err)
		return nil, fmt.Errorf("查询过期拍卖失败: %w", err)
	}
	defer rows.Close()

	auctions := make([]*model.Auction, 0)
	for rows.Next() {
		a := &model.Auction{}
		var status string
		if err := rows.Scan(
			&a.ID, &a.ItemID, &a.SellerID, &a.CurrentBid, &a.BidderID,
			&a.ReservePrice, &a.EndTime, &status, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描过期拍卖行失败: %w", err)
		}
		a.Status = model.AuctionStatus(status)
		auctions = append(auctions, a)
	}
	return auctions, nil
}

// ============================================================================
// 玩家资产（灵石）操作
// ============================================================================

// GetPlayerGold 查询玩家灵石数量。
func (r *TradeRepo) GetPlayerGold(ctx context.Context, playerID uint64) (*model.PlayerGold, error) {
	query := `SELECT player_id, gold, version, updated_at FROM trade_player_gold WHERE player_id = ?`
	pg := &model.PlayerGold{}
	err := r.db.QueryRowContext(ctx, query, playerID).Scan(&pg.PlayerID, &pg.Gold, &pg.Version, &pg.UpdatedAt)
	if err == sql.ErrNoRows {
		// 玩家没有资产记录，默认为 0
		return &model.PlayerGold{PlayerID: playerID, Gold: 0, Version: 0}, nil
	}
	if err != nil {
		r.log.ErrorContext(ctx, "查询玩家灵石失败", "error", err, "player_id", playerID)
		return nil, fmt.Errorf("查询玩家灵石失败: %w", err)
	}
	return pg, nil
}

// DeductGold 扣除玩家灵石（使用乐观锁防止超卖）。
func (r *TradeRepo) DeductGold(ctx context.Context, playerID uint64, amount uint64, oldVersion uint32) error {
	query := `UPDATE trade_player_gold SET gold = gold - ?, version = version + 1, updated_at = ? WHERE player_id = ? AND version = ? AND gold >= ?`
	result, err := r.db.ExecContext(ctx, query, amount, time.Now(), playerID, oldVersion, amount)
	if err != nil {
		r.log.ErrorContext(ctx, "扣除灵石失败", "error", err, "player_id", playerID, "amount", amount)
		return fmt.Errorf("扣除灵石失败: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		// 可能是版本冲突或余额不足
		pg, err := r.GetPlayerGold(ctx, playerID)
		if err != nil {
			return fmt.Errorf("验证余额失败: %w", err)
		}
		if pg.Version != oldVersion {
			return fmt.Errorf("并发操作导致版本冲突")
		}
		return fmt.Errorf("灵石不足: 持有 %d, 需要 %d", pg.Gold, amount)
	}
	return nil
}

// AddGold 增加玩家灵石。
func (r *TradeRepo) AddGold(ctx context.Context, playerID uint64, amount uint64) error {
	// 使用 INSERT ... ON DUPLICATE KEY UPDATE 处理首次加灵石的情况
	query := `INSERT INTO trade_player_gold (player_id, gold, version, updated_at) VALUES (?, ?, 1, ?)
		ON DUPLICATE KEY UPDATE gold = gold + ?, version = version + 1, updated_at = ?`
	_, err := r.db.ExecContext(ctx, query, playerID, amount, time.Now(), amount, time.Now())
	if err != nil {
		r.log.ErrorContext(ctx, "增加灵石失败", "error", err, "player_id", playerID, "amount", amount)
		return fmt.Errorf("增加灵石失败: %w", err)
	}
	return nil
}

// EnsurePlayerGold 确保玩家存在灵石记录，不存在则创建。
func (r *TradeRepo) EnsurePlayerGold(ctx context.Context, playerID uint64) error {
	query := `INSERT IGNORE INTO trade_player_gold (player_id, gold, version, updated_at) VALUES (?, 0, 0, ?)`
	_, err := r.db.ExecContext(ctx, query, playerID, time.Now())
	return err
}

// ============================================================================
// 事务支持
// ============================================================================

// WithTx 在数据库事务中执行函数。用于需要原子性的操作（如购买流程）。
func (r *TradeRepo) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			r.log.ErrorContext(ctx, "事务回滚失败", "error", rbErr)
		}
		return err
	}

	return tx.Commit()
}
