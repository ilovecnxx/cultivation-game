package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"
	"time"

	"cultivation-game/services/trade/internal/config"
	"cultivation-game/services/trade/internal/model"
)

// ============================================================================
// Mocks
// ============================================================================

type mockTradeRepo struct {
	createListingFn       func(ctx context.Context, l *model.Listing) error
	getListingByIDFn      func(ctx context.Context, id uint64) (*model.Listing, error)
	updateListingStatusFn func(ctx context.Context, id uint64, status model.ListingStatus) error
	listListingsFn        func(ctx context.Context, filter model.ListingFilter) ([]*model.Listing, int, error)
	getPlayerGoldFn       func(ctx context.Context, playerID uint64) (*model.PlayerGold, error)
	withTxFn              func(ctx context.Context, fn func(tx *sql.Tx) error) error
	listTransactionsFn    func(ctx context.Context, buyerID, sellerID uint64, page, pageSize int) ([]*model.Transaction, int, error)
	// Auction methods (unused in market tests)
	createAuctionFn        func(ctx context.Context, a *model.Auction) error
	getAuctionByIDFn       func(ctx context.Context, id uint64) (*model.Auction, error)
	updateAuctionBidFn     func(ctx context.Context, id uint64, newBid uint64, bidderID uint64, oldBid uint64) error
	updateAuctionStatusFn  func(ctx context.Context, id uint64, status model.AuctionStatus) error
	listActiveAuctionsFn   func(ctx context.Context, filter model.AuctionFilter) ([]*model.Auction, int, error)
	findExpiredAuctionsFn  func(ctx context.Context) ([]*model.Auction, error)
}

func (m *mockTradeRepo) CreateListing(ctx context.Context, l *model.Listing) error {
	if m.createListingFn != nil { return m.createListingFn(ctx, l) }
	return nil
}
func (m *mockTradeRepo) GetListingByID(ctx context.Context, id uint64) (*model.Listing, error) {
	if m.getListingByIDFn != nil { return m.getListingByIDFn(ctx, id) }
	return nil, nil
}
func (m *mockTradeRepo) UpdateListingStatus(ctx context.Context, id uint64, status model.ListingStatus) error {
	if m.updateListingStatusFn != nil { return m.updateListingStatusFn(ctx, id, status) }
	return nil
}
func (m *mockTradeRepo) ListListings(ctx context.Context, filter model.ListingFilter) ([]*model.Listing, int, error) {
	if m.listListingsFn != nil { return m.listListingsFn(ctx, filter) }
	return nil, 0, nil
}
func (m *mockTradeRepo) GetPlayerGold(ctx context.Context, playerID uint64) (*model.PlayerGold, error) {
	if m.getPlayerGoldFn != nil { return m.getPlayerGoldFn(ctx, playerID) }
	return &model.PlayerGold{PlayerID: playerID, Gold: 0, Version: 0}, nil
}
func (m *mockTradeRepo) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if m.withTxFn != nil { return m.withTxFn(ctx, fn) }
	return fn(nil)
}
func (m *mockTradeRepo) ListTransactions(ctx context.Context, buyerID, sellerID uint64, page, pageSize int) ([]*model.Transaction, int, error) {
	if m.listTransactionsFn != nil { return m.listTransactionsFn(ctx, buyerID, sellerID, page, pageSize) }
	return nil, 0, nil
}
func (m *mockTradeRepo) CreateAuction(ctx context.Context, a *model.Auction) error {
	if m.createAuctionFn != nil { return m.createAuctionFn(ctx, a) }
	return nil
}
func (m *mockTradeRepo) GetAuctionByID(ctx context.Context, id uint64) (*model.Auction, error) {
	if m.getAuctionByIDFn != nil { return m.getAuctionByIDFn(ctx, id) }
	return nil, nil
}
func (m *mockTradeRepo) UpdateAuctionBid(ctx context.Context, id uint64, newBid uint64, bidderID uint64, oldBid uint64) error {
	if m.updateAuctionBidFn != nil { return m.updateAuctionBidFn(ctx, id, newBid, bidderID, oldBid) }
	return nil
}
func (m *mockTradeRepo) UpdateAuctionStatus(ctx context.Context, id uint64, status model.AuctionStatus) error {
	if m.updateAuctionStatusFn != nil { return m.updateAuctionStatusFn(ctx, id, status) }
	return nil
}
func (m *mockTradeRepo) ListActiveAuctions(ctx context.Context, filter model.AuctionFilter) ([]*model.Auction, int, error) {
	if m.listActiveAuctionsFn != nil { return m.listActiveAuctionsFn(ctx, filter) }
	return nil, 0, nil
}
func (m *mockTradeRepo) FindExpiredAuctions(ctx context.Context) ([]*model.Auction, error) {
	if m.findExpiredAuctionsFn != nil { return m.findExpiredAuctionsFn(ctx) }
	return nil, nil
}

type mockCacheRepo struct {
	cacheListingFn         func(ctx context.Context, listing *model.Listing) error
	deleteCachedListingFn  func(ctx context.Context, listingID uint64) error
	addHotListingFn        func(ctx context.Context, listingID uint64, score float64) error
	removeHotListingFn     func(ctx context.Context, listingID uint64) error
	cacheAuctionFn         func(ctx context.Context, auction *model.Auction) error
	deleteCachedAuctionFn  func(ctx context.Context, auctionID uint64) error
	addActiveAuctionFn     func(ctx context.Context, auctionID uint64, endTime time.Time) error
	removeActiveAuctionFn  func(ctx context.Context, auctionID uint64) error
}

func (m *mockCacheRepo) CacheListing(ctx context.Context, listing *model.Listing) error {
	if m.cacheListingFn != nil { return m.cacheListingFn(ctx, listing) }
	return nil
}
func (m *mockCacheRepo) DeleteCachedListing(ctx context.Context, listingID uint64) error {
	if m.deleteCachedListingFn != nil { return m.deleteCachedListingFn(ctx, listingID) }
	return nil
}
func (m *mockCacheRepo) AddHotListing(ctx context.Context, listingID uint64, score float64) error {
	if m.addHotListingFn != nil { return m.addHotListingFn(ctx, listingID, score) }
	return nil
}
func (m *mockCacheRepo) RemoveHotListing(ctx context.Context, listingID uint64) error {
	if m.removeHotListingFn != nil { return m.removeHotListingFn(ctx, listingID) }
	return nil
}
func (m *mockCacheRepo) CacheAuction(ctx context.Context, auction *model.Auction) error {
	if m.cacheAuctionFn != nil { return m.cacheAuctionFn(ctx, auction) }
	return nil
}
func (m *mockCacheRepo) DeleteCachedAuction(ctx context.Context, auctionID uint64) error {
	if m.deleteCachedAuctionFn != nil { return m.deleteCachedAuctionFn(ctx, auctionID) }
	return nil
}
func (m *mockCacheRepo) AddActiveAuction(ctx context.Context, auctionID uint64, endTime time.Time) error {
	if m.addActiveAuctionFn != nil { return m.addActiveAuctionFn(ctx, auctionID, endTime) }
	return nil
}
func (m *mockCacheRepo) RemoveActiveAuction(ctx context.Context, auctionID uint64) error {
	if m.removeActiveAuctionFn != nil { return m.removeActiveAuctionFn(ctx, auctionID) }
	return nil
}

// ============================================================================
// Test helpers
// ============================================================================

func newMarketTestConfig() *config.Config {
	return &config.Config{
		ListingDefaultDuration: 72 * time.Hour,
		MinBuyQuantity:         1,
		MaxBuyQuantity:         9999,
		AuctionDefaultDuration: 24 * time.Hour,
		AuctionCheckInterval:   30 * time.Second,
	}
}

func newMarketService(repo TradeRepository, cache CacheRepository) *MarketService {
	return &MarketService{
		repo:  repo,
		cache: cache,
		cfg:   newMarketTestConfig(),
		log:   slog.Default(),
	}
}

func TestCreateListing(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		cacheCalled := false
		hotListCalled := false

		repo := &mockTradeRepo{
			createListingFn: func(_ context.Context, l *model.Listing) error {
				l.ID = 1
				return nil
			},
		}
		cache := &mockCacheRepo{
			cacheListingFn: func(_ context.Context, _ *model.Listing) error {
				cacheCalled = true
				return nil
			},
			addHotListingFn: func(_ context.Context, _ uint64, _ float64) error {
				hotListCalled = true
				return nil
			},
		}
		svc := newMarketService(repo, cache)
		expiresAt := time.Now().Add(48 * time.Hour)

		listing, err := svc.CreateListing(ctx, 1001, "玩家A", 2001, "灵剑", 5, 500, model.CurrencySpiritStone, expiresAt)
		if err != nil {
			t.Fatal(err)
		}
		if listing.ID != 1 {
			t.Errorf("ID = %d; want 1", listing.ID)
		}
		if listing.SellerID != 1001 {
			t.Errorf("SellerID = %d; want 1001", listing.SellerID)
		}
		if listing.ItemID != 2001 {
			t.Errorf("ItemID = %d; want 2001", listing.ItemID)
		}
		if listing.Quantity != 5 {
			t.Errorf("Quantity = %d; want 5", listing.Quantity)
		}
		if listing.UnitPrice != 500 {
			t.Errorf("UnitPrice = %d; want 500", listing.UnitPrice)
		}
		if listing.Status != model.ListingStatusActive {
			t.Errorf("Status = %q; want active", listing.Status)
		}
		if !cacheCalled {
			t.Error("CacheListing was not called")
		}
		if !hotListCalled {
			t.Error("AddHotListing was not called")
		}
	})

	t.Run("seller_id is zero", func(t *testing.T) {
		svc := newMarketService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.CreateListing(ctx, 0, "test", 1, "item", 1, 100, model.CurrencySpiritStone, time.Time{})
		if err == nil {
			t.Fatal("expected error for empty seller ID")
		}
	})

	t.Run("item_id is zero", func(t *testing.T) {
		svc := newMarketService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.CreateListing(ctx, 1, "test", 0, "item", 1, 100, model.CurrencySpiritStone, time.Time{})
		if err == nil {
			t.Fatal("expected error for empty item ID")
		}
	})

	t.Run("quantity is zero", func(t *testing.T) {
		svc := newMarketService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.CreateListing(ctx, 1, "test", 1, "item", 0, 100, model.CurrencySpiritStone, time.Time{})
		if err != ErrInvalidQuantity {
			t.Errorf("expected ErrInvalidQuantity, got %v", err)
		}
	})

	t.Run("unit price is zero", func(t *testing.T) {
		svc := newMarketService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.CreateListing(ctx, 1, "test", 1, "item", 1, 0, model.CurrencySpiritStone, time.Time{})
		if err != ErrInvalidPrice {
			t.Errorf("expected ErrInvalidPrice, got %v", err)
		}
	})

	t.Run("default currency type", func(t *testing.T) {
		var captured *model.Listing
		repo := &mockTradeRepo{
			createListingFn: func(_ context.Context, l *model.Listing) error {
				captured = l
				l.ID = 1
				return nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.CreateListing(ctx, 1, "test", 1, "item", 1, 100, "", time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		if captured.CurrencyType != model.CurrencySpiritStone {
			t.Errorf("CurrencyType = %q; want %q", captured.CurrencyType, model.CurrencySpiritStone)
		}
	})

	t.Run("default expiry when zero", func(t *testing.T) {
		var captured *model.Listing
		repo := &mockTradeRepo{
			createListingFn: func(_ context.Context, l *model.Listing) error {
				captured = l
				l.ID = 1
				return nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.CreateListing(ctx, 1, "test", 1, "item", 1, 100, model.CurrencySpiritStone, time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		expectedExpiry := time.Now().Add(72 * time.Hour)
		if captured.ExpiresAt.Before(expectedExpiry.Add(-time.Minute)) || captured.ExpiresAt.After(expectedExpiry.Add(time.Minute)) {
			t.Errorf("ExpiresAt = %v; want ~%v", captured.ExpiresAt, expectedExpiry)
		}
	})

	t.Run("repo error propagates", func(t *testing.T) {
		repo := &mockTradeRepo{
			createListingFn: func(_ context.Context, _ *model.Listing) error {
				return errors.New("db error")
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.CreateListing(ctx, 1, "test", 1, "item", 1, 100, model.CurrencySpiritStone, time.Now().Add(24*time.Hour))
		if err == nil {
			t.Fatal("expected error from repo")
		}
	})
}

func TestCancelListing(t *testing.T) {
	ctx := context.Background()

	makeListing := func(sellerID uint64, status model.ListingStatus) *model.Listing {
		return &model.Listing{
			ID:        1,
			SellerID:  sellerID,
			ItemID:    100,
			ItemName:  "灵剑",
			Quantity:  1,
			UnitPrice: 500,
			Status:    status,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
	}

	t.Run("success", func(t *testing.T) {
		var updatedStatus model.ListingStatus
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusActive), nil
			},
			updateListingStatusFn: func(_ context.Context, _ uint64, status model.ListingStatus) error {
				updatedStatus = status
				return nil
			},
		}
		cacheDelCalled := false
		cacheRemCalled := false
		cache := &mockCacheRepo{
			deleteCachedListingFn: func(_ context.Context, _ uint64) error {
				cacheDelCalled = true
				return nil
			},
			removeHotListingFn: func(_ context.Context, _ uint64) error {
				cacheRemCalled = true
				return nil
			},
		}
		svc := newMarketService(repo, cache)
		listing, err := svc.CancelListing(ctx, 1, 100)
		if err != nil {
			t.Fatal(err)
		}
		if listing.Status != model.ListingStatusCancelled {
			t.Errorf("Status = %q; want cancelled", listing.Status)
		}
		if updatedStatus != model.ListingStatusCancelled {
			t.Errorf("UpdateListingStatus called with %q; want cancelled", updatedStatus)
		}
		if !cacheDelCalled {
			t.Error("DeleteCachedListing was not called")
		}
		if !cacheRemCalled {
			t.Error("RemoveHotListing was not called")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return nil, nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.CancelListing(ctx, 999, 100)
		if err != ErrListingNotFound {
			t.Errorf("expected ErrListingNotFound, got %v", err)
		}
	})

	t.Run("not owner", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusActive), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.CancelListing(ctx, 1, 999) // sellerID 999 != 100
		if err != ErrNotListingOwner {
			t.Errorf("expected ErrNotListingOwner, got %v", err)
		}
	})

	t.Run("already sold", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusSold), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.CancelListing(ctx, 1, 100)
		if err != ErrListingAlreadySold {
			t.Errorf("expected ErrListingAlreadySold, got %v", err)
		}
	})

	t.Run("already cancelled", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusCancelled), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.CancelListing(ctx, 1, 100)
		if err != ErrListingAlreadyCancelled {
			t.Errorf("expected ErrListingAlreadyCancelled, got %v", err)
		}
	})

	t.Run("already expired", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusExpired), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.CancelListing(ctx, 1, 100)
		if err != ErrListingExpired {
			t.Errorf("expected ErrListingExpired, got %v", err)
		}
	})
}

func TestBuyItem(t *testing.T) {
	ctx := context.Background()

	makeListing := func(sellerID uint64, status model.ListingStatus) *model.Listing {
		return &model.Listing{
			ID:        1,
			SellerID:  sellerID,
			SellerName: "卖家",
			ItemID:    100,
			ItemName:  "灵剑",
			Quantity:  5,
			UnitPrice: 1000,
			Status:    status,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
	}

	t.Run("not found", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return nil, nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.BuyItem(ctx, 999, 200, 1)
		if err != ErrListingNotFound {
			t.Errorf("expected ErrListingNotFound, got %v", err)
		}
	})

	t.Run("already sold", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusSold), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.BuyItem(ctx, 1, 200, 1)
		if err != ErrListingAlreadySold {
			t.Errorf("expected ErrListingAlreadySold, got %v", err)
		}
	})

	t.Run("already cancelled", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusCancelled), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.BuyItem(ctx, 1, 200, 1)
		if err != ErrListingAlreadyCancelled {
			t.Errorf("expected ErrListingAlreadyCancelled, got %v", err)
		}
	})

	t.Run("already expired", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusExpired), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.BuyItem(ctx, 1, 200, 1)
		if err != ErrListingExpired {
			t.Errorf("expected ErrListingExpired, got %v", err)
		}
	})

	t.Run("buyer is seller", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusActive), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.BuyItem(ctx, 1, 100, 1) // buyerID == sellerID
		if err != ErrBuyOwnListing {
			t.Errorf("expected ErrBuyOwnListing, got %v", err)
		}
	})

	t.Run("quantity zero", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusActive), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.BuyItem(ctx, 1, 200, 0)
		if err != ErrInvalidQuantity {
			t.Errorf("expected ErrInvalidQuantity, got %v", err)
		}
	})

	t.Run("quantity exceeds listing", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusActive), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.BuyItem(ctx, 1, 200, 10) // listing has 5
		if err != ErrInvalidQuantity {
			t.Errorf("expected ErrInvalidQuantity, got %v", err)
		}
	})

	t.Run("quantity below min", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusActive), nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		// Use an atypical config with MinBuyQuantity > 1
		svc.cfg.MinBuyQuantity = 2
		_, err := svc.BuyItem(ctx, 1, 200, 1)
		if err == nil {
			t.Fatal("expected error for quantity below min")
		}
	})

	t.Run("insufficient gold", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusActive), nil
			},
			getPlayerGoldFn: func(_ context.Context, _ uint64) (*model.PlayerGold, error) {
				return &model.PlayerGold{PlayerID: 200, Gold: 100, Version: 1}, nil // only 100 gold
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		// Total price = 1 * 1000 = 1000
		_, err := svc.BuyItem(ctx, 1, 200, 1)
		if err != ErrInsufficientGold {
			t.Errorf("expected ErrInsufficientGold, got %v", err)
		}
	})

	// Note: BuyItem success path (transaction execution) is not unit-tested at the
	// service level because WithTx passes a *sql.Tx to the callback and we cannot mock
	// the internal SQL operations without go-sqlmock. The transaction body logic
	// (deductGoldTx, addGoldTx, updateListingStatusTx, createTransactionTx) is covered
	// by repository integration tests.

	t.Run("withTx error propagates", func(t *testing.T) {
		repo := &mockTradeRepo{
			getListingByIDFn: func(_ context.Context, _ uint64) (*model.Listing, error) {
				return makeListing(100, model.ListingStatusActive), nil
			},
			getPlayerGoldFn: func(_ context.Context, _ uint64) (*model.PlayerGold, error) {
				return &model.PlayerGold{PlayerID: 200, Gold: 100000, Version: 1}, nil
			},
			withTxFn: func(_ context.Context, _ func(tx *sql.Tx) error) error {
				return errors.New("transaction failed")
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, err := svc.BuyItem(ctx, 1, 200, 1)
		if err == nil {
			t.Fatal("expected error from WithTx")
		}
	})
}

func TestGetListings(t *testing.T) {
	ctx := context.Background()

	t.Run("basic query", func(t *testing.T) {
		repo := &mockTradeRepo{
			listListingsFn: func(_ context.Context, filter model.ListingFilter) ([]*model.Listing, int, error) {
				if filter.Page != 1 || filter.PageSize != 20 {
					t.Errorf("expected page=1, size=20; got page=%d, size=%d", filter.Page, filter.PageSize)
				}
				return []*model.Listing{
					{ID: 1, ItemName: "灵剑", Status: model.ListingStatusActive},
				}, 1, nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		listings, total, err := svc.GetListings(ctx, model.ListingFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if total != 1 {
			t.Errorf("total = %d; want 1", total)
		}
		if len(listings) != 1 {
			t.Errorf("got %d listings; want 1", len(listings))
		}
		if listings[0].ID != 1 {
			t.Errorf("first listing ID = %d; want 1", listings[0].ID)
		}
	})

	t.Run("with filters", func(t *testing.T) {
		var captured model.ListingFilter
		repo := &mockTradeRepo{
			listListingsFn: func(_ context.Context, filter model.ListingFilter) ([]*model.Listing, int, error) {
				captured = filter
				return []*model.Listing{}, 0, nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, _, _ = svc.GetListings(ctx, model.ListingFilter{
			SellerID:     10,
			ItemID:       200,
			Status:       model.ListingStatusActive,
			CurrencyType: model.CurrencySpiritStone,
			Page:         3,
			PageSize:     15,
		})
		if captured.SellerID != 10 {
			t.Errorf("SellerID = %d; want 10", captured.SellerID)
		}
		if captured.ItemID != 200 {
			t.Errorf("ItemID = %d; want 200", captured.ItemID)
		}
		if captured.Status != model.ListingStatusActive {
			t.Errorf("Status = %q; want active", captured.Status)
		}
		if captured.Page != 3 {
			t.Errorf("Page = %d; want 3", captured.Page)
		}
	})

	t.Run("page clamping", func(t *testing.T) {
		var captured model.ListingFilter
		repo := &mockTradeRepo{
			listListingsFn: func(_ context.Context, filter model.ListingFilter) ([]*model.Listing, int, error) {
				captured = filter
				return []*model.Listing{}, 0, nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, _, _ = svc.GetListings(ctx, model.ListingFilter{Page: 0, PageSize: 0})
		if captured.Page != 1 {
			t.Errorf("Page after normalization = %d; want 1", captured.Page)
		}
		if captured.PageSize != 20 {
			t.Errorf("PageSize after normalization = %d; want 20", captured.PageSize)
		}
	})

	t.Run("page size max 100", func(t *testing.T) {
		var captured model.ListingFilter
		repo := &mockTradeRepo{
			listListingsFn: func(_ context.Context, filter model.ListingFilter) ([]*model.Listing, int, error) {
				captured = filter
				return []*model.Listing{}, 0, nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, _, _ = svc.GetListings(ctx, model.ListingFilter{Page: 1, PageSize: 200})
		if captured.PageSize != 100 {
			t.Errorf("PageSize clamped to %d; want 100", captured.PageSize)
		}
	})

	t.Run("active listings are cached", func(t *testing.T) {
		cacheCallCount := 0
		repo := &mockTradeRepo{
			listListingsFn: func(_ context.Context, _ model.ListingFilter) ([]*model.Listing, int, error) {
				return []*model.Listing{
					{ID: 1, Status: model.ListingStatusActive, ItemName: "剑"},
					{ID: 2, Status: model.ListingStatusSold, ItemName: "甲"},
				}, 2, nil
			},
		}
		cache := &mockCacheRepo{
			cacheListingFn: func(_ context.Context, _ *model.Listing) error {
				cacheCallCount++
				return nil
			},
		}
		svc := newMarketService(repo, cache)
		_, _, err := svc.GetListings(ctx, model.ListingFilter{Status: model.ListingStatusActive})
		if err != nil {
			t.Fatal(err)
		}
		// Only active listings should be cached, not sold ones
		if cacheCallCount != 1 {
			t.Errorf("expected 1 cache call (only active), got %d", cacheCallCount)
		}
	})
}

func TestGetTransactions(t *testing.T) {
	ctx := context.Background()

	t.Run("basic query", func(t *testing.T) {
		repo := &mockTradeRepo{
			listTransactionsFn: func(_ context.Context, buyerID, sellerID uint64, page, pageSize int) ([]*model.Transaction, int, error) {
				return []*model.Transaction{
					{ID: 1, ListingID: 100, BuyerID: 200, SellerID: 300, TotalPrice: 5000},
				}, 1, nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		trans, total, err := svc.GetTransactions(ctx, 200, 300, 1, 20)
		if err != nil {
			t.Fatal(err)
		}
		if total != 1 {
			t.Errorf("total = %d; want 1", total)
		}
		if len(trans) != 1 {
			t.Errorf("got %d transactions; want 1", len(trans))
		}
	})

	t.Run("page clamping", func(t *testing.T) {
		var (
			capturedPage     int
			capturedPageSize int
		)
		repo := &mockTradeRepo{
			listTransactionsFn: func(_ context.Context, _, _ uint64, page, pageSize int) ([]*model.Transaction, int, error) {
				capturedPage = page
				capturedPageSize = pageSize
				return nil, 0, nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, _, _ = svc.GetTransactions(ctx, 0, 0, 0, 0)
		if capturedPage != 1 {
			t.Errorf("page = %d; want 1", capturedPage)
		}
		if capturedPageSize != 20 {
			t.Errorf("pageSize = %d; want 20", capturedPageSize)
		}
	})

	t.Run("page size capped at 100", func(t *testing.T) {
		var capturedPageSize int
		repo := &mockTradeRepo{
			listTransactionsFn: func(_ context.Context, _, _ uint64, _ int, pageSize int) ([]*model.Transaction, int, error) {
				capturedPageSize = pageSize
				return nil, 0, nil
			},
		}
		svc := newMarketService(repo, &mockCacheRepo{})
		_, _, _ = svc.GetTransactions(ctx, 0, 0, 1, 999)
		if capturedPageSize != 100 {
			t.Errorf("pageSize = %d; want 100", capturedPageSize)
		}
	})
}
