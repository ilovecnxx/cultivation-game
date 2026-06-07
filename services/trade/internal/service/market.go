// Package service 实现交易市场业务逻辑：挂单管理、购买流程、价格验证。
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"cultivation-game/services/trade/internal/config"
	"cultivation-game/services/trade/internal/model"
)

// 错误定义。
var (
	ErrListingNotFound      = errors.New("挂单不存在")
	ErrListingNotActive     = errors.New("挂单已下架或已售出")
	ErrListingExpired       = errors.New("挂单已过期")
	ErrNotListingOwner      = errors.New("只能操作自己的挂单")
	ErrInsufficientGold     = errors.New("灵石不足")
	ErrInvalidQuantity      = errors.New("数量不合法")
	ErrInvalidPrice         = errors.New("价格必须大于 0")
	ErrBuyOwnListing        = errors.New("不能购买自己的挂单")
	ErrConcurrentConflict   = errors.New("并发冲突，请重试")
	ErrListingAlreadySold   = errors.New("挂单已售出")
	ErrListingAlreadyCancelled = errors.New("挂单已取消")
)

// MarketService 市场交易服务，处理挂单创建、取消、购买和查询。
type MarketService struct {
	repo  TradeRepository
	cache CacheRepository
	cfg   *config.Config
	log   *slog.Logger
}

// NewMarketService 创建 MarketService。
func NewMarketService(repo TradeRepository, cache CacheRepository, cfg *config.Config, log *slog.Logger) *MarketService {
	return &MarketService{
		repo:  repo,
		cache: cache,
		cfg:   cfg,
		log:   log,
	}
}

// ============================================================================
// 挂单创建
// ============================================================================

// CreateListing 创建市场挂单。
// 流程：参数校验 -> 设置过期时间 -> 写入 MySQL -> 写入 Redis 缓存。
func (s *MarketService) CreateListing(ctx context.Context, sellerID uint64, sellerName string, itemID uint32, itemName string, quantity uint32, unitPrice uint64, currencyType model.CurrencyType, expiresAt time.Time) (*model.Listing, error) {
	// 参数校验
	if sellerID == 0 {
		return nil, errors.New("卖家 ID 不能为空")
	}
	if itemID == 0 {
		return nil, errors.New("物品 ID 不能为空")
	}
	if quantity == 0 {
		return nil, ErrInvalidQuantity
	}
	if unitPrice == 0 {
		return nil, ErrInvalidPrice
	}
	if currencyType == "" {
		currencyType = model.CurrencySpiritStone
	}

	// 如果未指定过期时间，使用系统默认
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(s.cfg.ListingDefaultDuration)
	}

	listing := &model.Listing{
		SellerID:     sellerID,
		SellerName:   sellerName,
		ItemID:       itemID,
		ItemName:     itemName,
		Quantity:     quantity,
		UnitPrice:    unitPrice,
		CurrencyType: currencyType,
		Status:       model.ListingStatusActive,
		ExpiresAt:    expiresAt,
	}

	// 持久化到 MySQL
	if err := s.repo.CreateListing(ctx, listing); err != nil {
		s.log.ErrorContext(ctx, "创建挂单失败", "error", err, "seller_id", sellerID)
		return nil, fmt.Errorf("创建挂单失败: %w", err)
	}

	// 缓存到 Redis（异步风格，失败不影响主流程）
	if err := s.cache.CacheListing(ctx, listing); err != nil {
		s.log.WarnContext(ctx, "缓存挂单失败", "error", err, "listing_id", listing.ID)
	}
	if err := s.cache.AddHotListing(ctx, listing.ID, float64(listing.CreatedAt.Unix())); err != nil {
		s.log.WarnContext(ctx, "添加到热门列表失败", "error", err, "listing_id", listing.ID)
	}

	s.log.InfoContext(ctx, "挂单创建成功",
		"listing_id", listing.ID,
		"seller_id", sellerID,
		"item_id", itemID,
		"quantity", quantity,
		"unit_price", unitPrice,
	)
	return listing, nil
}

// ============================================================================
// 挂单取消
// ============================================================================

// CancelListing 取消挂单。
// 流程：查询挂单 -> 验证所有权 -> 验证状态 -> 更新状态为取消 -> 清除缓存。
func (s *MarketService) CancelListing(ctx context.Context, listingID uint64, sellerID uint64) (*model.Listing, error) {
	listing, err := s.repo.GetListingByID(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("查询挂单失败: %w", err)
	}
	if listing == nil {
		return nil, ErrListingNotFound
	}

	// 验证所有权
	if listing.SellerID != sellerID {
		return nil, ErrNotListingOwner
	}

	// 验证状态
	switch listing.Status {
	case model.ListingStatusSold:
		return nil, ErrListingAlreadySold
	case model.ListingStatusCancelled:
		return nil, ErrListingAlreadyCancelled
	case model.ListingStatusExpired:
		return nil, ErrListingExpired
	}

	// 更新状态
	if err := s.repo.UpdateListingStatus(ctx, listingID, model.ListingStatusCancelled); err != nil {
		return nil, fmt.Errorf("取消挂单失败: %w", err)
	}
	listing.Status = model.ListingStatusCancelled

	// 清除缓存
	if err := s.cache.DeleteCachedListing(ctx, listingID); err != nil {
		s.log.WarnContext(ctx, "删除挂单缓存失败", "error", err, "listing_id", listingID)
	}
	if err := s.cache.RemoveHotListing(ctx, listingID); err != nil {
		s.log.WarnContext(ctx, "移除热门挂单失败", "error", err, "listing_id", listingID)
	}

	s.log.InfoContext(ctx, "挂单已取消", "listing_id", listingID, "seller_id", sellerID)
	return listing, nil
}

// ============================================================================
// 购买流程
// ============================================================================

// BuyItem 购买物品。使用数据库事务确保数据一致性。
// 完整流程：
//   1. 查询并验证挂单
//   2. 验证买家不能是自己
//   3. 验证数量合法性
//   4. 在事务中：扣除买家灵石 -> 增加卖家灵石 -> 更新挂单状态 -> 创建交易记录
//   5. 清除 Redis 缓存
func (s *MarketService) BuyItem(ctx context.Context, listingID uint64, buyerID uint64, quantity uint32) (*model.Transaction, error) {
	// ---- 第 1 步：查询并验证挂单 ----
	listing, err := s.repo.GetListingByID(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("查询挂单失败: %w", err)
	}
	if listing == nil {
		return nil, ErrListingNotFound
	}

	// 验证挂单状态
	switch listing.Status {
	case model.ListingStatusSold:
		return nil, ErrListingAlreadySold
	case model.ListingStatusCancelled:
		return nil, ErrListingAlreadyCancelled
	case model.ListingStatusExpired:
		return nil, ErrListingExpired
	}
	if !listing.IsActive() {
		return nil, ErrListingNotActive
	}

	// 验证买卖家不同
	if listing.SellerID == buyerID {
		return nil, ErrBuyOwnListing
	}

	// 验证数量合法性
	if quantity == 0 || quantity > listing.Quantity {
		return nil, ErrInvalidQuantity
	}
	if quantity < s.cfg.MinBuyQuantity || quantity > s.cfg.MaxBuyQuantity {
		return nil, fmt.Errorf("购买数量应在 %d 到 %d 之间", s.cfg.MinBuyQuantity, s.cfg.MaxBuyQuantity)
	}

	totalPrice := listing.TotalPrice(quantity)

	// 验证买家灵石余额
	buyerGold, err := s.repo.GetPlayerGold(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("查询买家资产失败: %w", err)
	}
	if buyerGold.Gold < totalPrice {
		return nil, ErrInsufficientGold
	}

	// ---- 第 2 步：在事务中执行核心操作 ----
	var transaction *model.Transaction

	err = s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		// 在事务内再次查询挂单，确保状态未被其他事务修改
		txListing, err := s.getListingTx(ctx, tx, listingID)
		if err != nil {
			return err
		}
		if txListing == nil {
			return ErrListingNotFound
		}
		if txListing.Status != model.ListingStatusActive {
			return ErrListingNotActive
		}

		// 扣除买家灵石（带乐观锁）
		if err := s.deductGoldTx(ctx, tx, buyerID, totalPrice, buyerGold.Version); err != nil {
			return err
		}

		// 增加卖家灵石
		if err := s.addGoldTx(ctx, tx, listing.SellerID, totalPrice); err != nil {
			// 回滚买家扣款（调用者会回滚整个事务）
			return fmt.Errorf("转账给卖家失败: %w", err)
		}

		// 更新挂单状态为已售出
		if err := s.updateListingStatusTx(ctx, tx, listingID, string(model.ListingStatusSold)); err != nil {
			return fmt.Errorf("更新挂单状态失败: %w", err)
		}

		// 创建交易记录
		trans := &model.Transaction{
			ListingID:  listingID,
			BuyerID:    buyerID,
			SellerID:   listing.SellerID,
			ItemID:     listing.ItemID,
			Quantity:   quantity,
			UnitPrice:  listing.UnitPrice,
			TotalPrice: totalPrice,
		}

		if err := s.createTransactionTx(ctx, tx, trans); err != nil {
			return fmt.Errorf("创建交易记录失败: %w", err)
		}

		transaction = trans
		return nil
	})

	if err != nil {
		s.log.ErrorContext(ctx, "购买失败",
			"error", err,
			"listing_id", listingID,
			"buyer_id", buyerID,
			"quantity", quantity,
		)
		return nil, err
	}

	// ---- 第 3 步：清除缓存（在事务外执行） ----
	if err := s.cache.DeleteCachedListing(ctx, listingID); err != nil {
		s.log.WarnContext(ctx, "删除挂单缓存失败", "error", err, "listing_id", listingID)
	}
	if err := s.cache.RemoveHotListing(ctx, listingID); err != nil {
		s.log.WarnContext(ctx, "移除热门挂单失败", "error", err, "listing_id", listingID)
	}

	s.log.InfoContext(ctx, "交易成功",
		"transaction_id", transaction.ID,
		"listing_id", listingID,
		"buyer_id", buyerID,
		"seller_id", listing.SellerID,
		"item_id", listing.ItemID,
		"quantity", quantity,
		"total_price", totalPrice,
	)
	return transaction, nil
}

// ============================================================================
// 挂单查询
// ============================================================================

// GetListings 查询挂单列表，支持分页和筛选。
// 对于活跃挂单优先尝试从缓存读取。
func (s *MarketService) GetListings(ctx context.Context, filter model.ListingFilter) ([]*model.Listing, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	listings, total, err := s.repo.ListListings(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("查询挂单列表失败: %w", err)
	}

	// 如果查询的是活跃状态挂单，异步写入缓存加速后续请求
	if filter.Status == model.ListingStatusActive || filter.Status == "" {
		for _, l := range listings {
			if l.Status == model.ListingStatusActive {
				if cacheErr := s.cache.CacheListing(ctx, l); cacheErr != nil {
					s.log.DebugContext(ctx, "缓存挂单失败", "error", cacheErr, "listing_id", l.ID)
				}
			}
		}
	}

	return listings, total, nil
}

// ============================================================================
// 交易记录查询
// ============================================================================

// GetTransactions 查询交易记录，支持按买家和卖家筛选。
func (s *MarketService) GetTransactions(ctx context.Context, buyerID, sellerID uint64, page, pageSize int) ([]*model.Transaction, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return s.repo.ListTransactions(ctx, buyerID, sellerID, page, pageSize)
}

// ============================================================================
// 事务内部辅助方法
// 这些方法使用 *sql.Tx 而非 *sql.DB，确保在同一个事务中执行。
// ============================================================================

// getListingTx 在事务中根据 ID 查询挂单（使用 FOR UPDATE 锁定行）。
func (s *MarketService) getListingTx(ctx context.Context, tx *sql.Tx, id uint64) (*model.Listing, error) {
	query := `SELECT id, seller_id, seller_name, item_id, item_name, quantity, unit_price, currency_type, status, created_at, expires_at, updated_at
		FROM trade_listings WHERE id = ? FOR UPDATE`

	l := &model.Listing{}
	var sellerName, itemName, currencyType, status string
	err := tx.QueryRowContext(ctx, query, id).Scan(
		&l.ID, &l.SellerID, &sellerName, &l.ItemID, &itemName,
		&l.Quantity, &l.UnitPrice, &currencyType, &status,
		&l.CreatedAt, &l.ExpiresAt, &l.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("事务中查询挂单失败: %w", err)
	}
	l.SellerName = sellerName
	l.ItemName = itemName
	l.CurrencyType = model.CurrencyType(currencyType)
	l.Status = model.ListingStatus(status)
	return l, nil
}

// deductGoldTx 在事务中扣除玩家灵石（带版本号乐观锁）。
func (s *MarketService) deductGoldTx(ctx context.Context, tx *sql.Tx, playerID uint64, amount uint64, oldVersion uint32) error {
	query := `UPDATE trade_player_gold SET gold = gold - ?, version = version + 1, updated_at = ? WHERE player_id = ? AND version = ? AND gold >= ?`
	result, err := tx.ExecContext(ctx, query, amount, time.Now(), playerID, oldVersion, amount)
	if err != nil {
		return fmt.Errorf("事务中扣除灵石失败: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		// 验证具体原因
		var currentGold uint64
		var currentVersion uint32
		err := tx.QueryRowContext(ctx, "SELECT gold, version FROM trade_player_gold WHERE player_id = ? FOR UPDATE", playerID).Scan(&currentGold, &currentVersion)
		if err != nil {
			return fmt.Errorf("验证余额失败: %w", err)
		}
		if currentVersion != oldVersion {
			return ErrConcurrentConflict
		}
		if currentGold < amount {
			return ErrInsufficientGold
		}
		return fmt.Errorf("扣除灵石失败: 未知原因")
	}
	return nil
}

// addGoldTx 在事务中增加玩家灵石。
func (s *MarketService) addGoldTx(ctx context.Context, tx *sql.Tx, playerID uint64, amount uint64) error {
	query := `INSERT INTO trade_player_gold (player_id, gold, version, updated_at) VALUES (?, ?, 1, ?)
		ON DUPLICATE KEY UPDATE gold = gold + ?, version = version + 1, updated_at = ?`
	_, err := tx.ExecContext(ctx, query, playerID, amount, time.Now(), amount, time.Now())
	if err != nil {
		return fmt.Errorf("事务中增加灵石失败: %w", err)
	}
	return nil
}

// updateListingStatusTx 在事务中更新挂单状态。
func (s *MarketService) updateListingStatusTx(ctx context.Context, tx *sql.Tx, id uint64, status string) error {
	query := `UPDATE trade_listings SET status = ?, updated_at = ? WHERE id = ?`
	_, err := tx.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("事务中更新挂单状态失败: %w", err)
	}
	return nil
}

// createTransactionTx 在事务中创建交易记录。
func (s *MarketService) createTransactionTx(ctx context.Context, tx *sql.Tx, t *model.Transaction) error {
	query := `INSERT INTO trade_transactions (listing_id, buyer_id, seller_id, item_id, quantity, unit_price, total_price, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := tx.ExecContext(ctx, query,
		t.ListingID, t.BuyerID, t.SellerID, t.ItemID,
		t.Quantity, t.UnitPrice, t.TotalPrice, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("事务中创建交易记录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = uint64(id)
	return nil
}
