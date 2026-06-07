package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"cultivation-game/services/trade/internal/config"
	"cultivation-game/services/trade/internal/model"
)

// ============================================================================
// Test helpers
// ============================================================================

func newAuctionTestConfig() *config.Config {
	return &config.Config{
		ListingDefaultDuration: 72 * time.Hour,
		AuctionDefaultDuration: 24 * time.Hour,
		AuctionCheckInterval:   30 * time.Second,
		MinBuyQuantity:         1,
		MaxBuyQuantity:         9999,
	}
}

func newAuctionService(repo TradeRepository, cache CacheRepository) *AuctionService {
	return &AuctionService{
		repo:  repo,
		cache: cache,
		cfg:   newAuctionTestConfig(),
		log:   slog.Default(),
	}
}

// ============================================================================
// StartAuction
// ============================================================================

func TestStartAuction(t *testing.T) {
	ctx := context.Background()

	t.Run("success with explicit duration", func(t *testing.T) {
		cacheCalled := false
		addActiveCalled := false

		repo := &mockTradeRepo{
			createAuctionFn: func(_ context.Context, a *model.Auction) error {
				a.ID = 1
				return nil
			},
		}
		cache := &mockCacheRepo{
			cacheAuctionFn: func(_ context.Context, _ *model.Auction) error {
				cacheCalled = true
				return nil
			},
			addActiveAuctionFn: func(_ context.Context, _ uint64, _ time.Time) error {
				addActiveCalled = true
				return nil
			},
		}
		svc := newAuctionService(repo, cache)
		auction, err := svc.StartAuction(ctx, 2001, 100, 50000, 3600) // 1 hour
		if err != nil {
			t.Fatal(err)
		}
		if auction.ID != 1 {
			t.Errorf("ID = %d; want 1", auction.ID)
		}
		if auction.SellerID != 100 {
			t.Errorf("SellerID = %d; want 100", auction.SellerID)
		}
		if auction.ItemID != 2001 {
			t.Errorf("ItemID = %d; want 2001", auction.ItemID)
		}
		if auction.ReservePrice != 50000 {
			t.Errorf("ReservePrice = %d; want 50000", auction.ReservePrice)
		}
		if auction.Status != model.AuctionStatusActive {
			t.Errorf("Status = %q; want active", auction.Status)
		}
		if auction.CurrentBid != 0 {
			t.Errorf("CurrentBid = %d; want 0", auction.CurrentBid)
		}
		if auction.BidderID != 0 {
			t.Errorf("BidderID = %d; want 0", auction.BidderID)
		}
		expectedEnd := time.Now().Add(3600 * time.Second)
		if auction.EndTime.Before(expectedEnd.Add(-time.Second)) || auction.EndTime.After(expectedEnd.Add(time.Second)) {
			t.Errorf("EndTime = %v; want ~%v", auction.EndTime, expectedEnd)
		}
		if !cacheCalled {
			t.Error("CacheAuction was not called")
		}
		if !addActiveCalled {
			t.Error("AddActiveAuction was not called")
		}
	})

	t.Run("success with default duration", func(t *testing.T) {
		repo := &mockTradeRepo{
			createAuctionFn: func(_ context.Context, a *model.Auction) error {
				a.ID = 2
				return nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		auction, err := svc.StartAuction(ctx, 100, 200, 10000, 0) // durationSeconds=0 -> use default
		if err != nil {
			t.Fatal(err)
		}
		if auction.ID != 2 {
			t.Errorf("ID = %d; want 2", auction.ID)
		}
		expectedEnd := time.Now().Add(24 * time.Hour)
		if auction.EndTime.Before(expectedEnd.Add(-time.Minute)) || auction.EndTime.After(expectedEnd.Add(time.Minute)) {
			t.Errorf("EndTime = %v; want ~%v", auction.EndTime, expectedEnd)
		}
	})

	t.Run("seller id is zero", func(t *testing.T) {
		svc := newAuctionService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.StartAuction(ctx, 100, 0, 1000, 3600)
		if err == nil {
			t.Fatal("expected error for empty seller ID")
		}
	})

	t.Run("item id is zero", func(t *testing.T) {
		svc := newAuctionService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.StartAuction(ctx, 0, 100, 1000, 3600)
		if err == nil {
			t.Fatal("expected error for empty item ID")
		}
	})

	t.Run("reserve price is zero", func(t *testing.T) {
		svc := newAuctionService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.StartAuction(ctx, 100, 200, 0, 3600)
		if err != ErrAuctionReserveTooLow {
			t.Errorf("expected ErrAuctionReserveTooLow, got %v", err)
		}
	})

	t.Run("duration too short", func(t *testing.T) {
		svc := newAuctionService(&mockTradeRepo{}, &mockCacheRepo{})
		// 4 minutes < 5 min minimum
		_, err := svc.StartAuction(ctx, 100, 200, 10000, 240)
		if err != ErrAuctionDurationTooShort {
			t.Errorf("expected ErrAuctionDurationTooShort, got %v", err)
		}
	})

	t.Run("duration equal to minimum is valid", func(t *testing.T) {
		repo := &mockTradeRepo{
			createAuctionFn: func(_ context.Context, a *model.Auction) error {
				a.ID = 3
				return nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.StartAuction(ctx, 100, 200, 10000, 300) // 5 minutes = minimum
		if err != nil {
			t.Fatalf("5 minute duration should be valid: %v", err)
		}
	})

	t.Run("duration too long", func(t *testing.T) {
		svc := newAuctionService(&mockTradeRepo{}, &mockCacheRepo{})
		// 8 days > 7 day maximum
		_, err := svc.StartAuction(ctx, 100, 200, 10000, 8*24*3600)
		if err != ErrAuctionDurationTooLong {
			t.Errorf("expected ErrAuctionDurationTooLong, got %v", err)
		}
	})

	t.Run("duration equal to maximum is valid", func(t *testing.T) {
		repo := &mockTradeRepo{
			createAuctionFn: func(_ context.Context, a *model.Auction) error {
				a.ID = 4
				return nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.StartAuction(ctx, 100, 200, 10000, 168*3600) // 7 days = max
		if err != nil {
			t.Fatalf("7 day duration should be valid: %v", err)
		}
	})
}

// ============================================================================
// PlaceBid
// ============================================================================

func TestPlaceBid(t *testing.T) {
	ctx := context.Background()

	makeActiveAuction := func(currentBid, reservePrice uint64, bidderID uint64) *model.Auction {
		return &model.Auction{
			ID:           1,
			ItemID:       100,
			SellerID:     200,
			CurrentBid:   currentBid,
			BidderID:     bidderID,
			ReservePrice: reservePrice,
			EndTime:      time.Now().Add(24 * time.Hour),
			Status:       model.AuctionStatusActive,
		}
	}

	t.Run("successful bid", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				return makeActiveAuction(1000, 5000, 0), nil
			},
			updateAuctionBidFn: func(_ context.Context, _ uint64, newBid, _ uint64, oldBid uint64) error {
				if newBid != 5000 {
					t.Errorf("newBid = %d; want 5000", newBid)
				}
				if oldBid != 1000 {
					t.Errorf("oldBid = %d; want 1000", oldBid)
				}
				return nil
			},
		}
		cacheCalled := false
		cache := &mockCacheRepo{
			cacheAuctionFn: func(_ context.Context, _ *model.Auction) error {
				cacheCalled = true
				return nil
			},
		}
		svc := newAuctionService(repo, cache)
		auction, err := svc.PlaceBid(ctx, 1, 300, 5000)
		if err != nil {
			t.Fatal(err)
		}
		if auction.CurrentBid != 5000 {
			t.Errorf("CurrentBid = %d; want 5000", auction.CurrentBid)
		}
		if auction.BidderID != 300 {
			t.Errorf("BidderID = %d; want 300", auction.BidderID)
		}
		if !cacheCalled {
			t.Error("CacheAuction was not called")
		}
	})

	t.Run("bidder id is zero", func(t *testing.T) {
		svc := newAuctionService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 1, 0, 1000)
		if err == nil {
			t.Fatal("expected error for empty bidder ID")
		}
	})

	t.Run("bid amount is zero", func(t *testing.T) {
		svc := newAuctionService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 1, 300, 0)
		if err != ErrBidTooLow {
			t.Errorf("expected ErrBidTooLow, got %v", err)
		}
	})

	t.Run("auction not found", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				return nil, nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 999, 300, 1000)
		if err != ErrAuctionNotFound {
			t.Errorf("expected ErrAuctionNotFound, got %v", err)
		}
	})

	t.Run("auction already completed", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				a := makeActiveAuction(1000, 5000, 0)
				a.Status = model.AuctionStatusCompleted
				return a, nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 1, 300, 2000)
		if err != ErrAuctionAlreadyEnded {
			t.Errorf("expected ErrAuctionAlreadyEnded, got %v", err)
		}
	})

	t.Run("auction already cancelled", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				a := makeActiveAuction(1000, 5000, 0)
				a.Status = model.AuctionStatusCancelled
				return a, nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 1, 300, 2000)
		if err != ErrAuctionNotActive {
			t.Errorf("expected ErrAuctionNotActive, got %v", err)
		}
	})

	t.Run("auction already expired", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				a := makeActiveAuction(1000, 5000, 0)
				a.Status = model.AuctionStatusExpired
				return a, nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 1, 300, 2000)
		if err != ErrAuctionAlreadyEnded {
			t.Errorf("expected ErrAuctionAlreadyEnded, got %v", err)
		}
	})

	t.Run("auction past end time", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				a := makeActiveAuction(1000, 5000, 0)
				a.EndTime = time.Now().Add(-1 * time.Hour) // ended, but status still active
				return a, nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 1, 300, 2000)
		// When the auction status is still "active" but the end time has passed,
		// IsActive() returns false, and the default switch case returns ErrAuctionNotActive.
		if err != ErrAuctionNotActive {
			t.Errorf("expected ErrAuctionNotActive, got %v", err)
		}
	})

	t.Run("bid on own auction", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				return makeActiveAuction(1000, 5000, 0), nil // sellerID = 200
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 1, 200, 2000) // bidderID == sellerID
		if err != ErrBidOwnAuction {
			t.Errorf("expected ErrBidOwnAuction, got %v", err)
		}
	})

	t.Run("bid not higher than current", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				return makeActiveAuction(1000, 5000, 0), nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 1, 300, 1000) // equal to current bid
		if err != ErrBidTooLow {
			t.Errorf("expected ErrBidTooLow, got %v", err)
		}
	})

	t.Run("bid below minimum increment", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				return makeActiveAuction(1000, 5000, 0), nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		// minBid = 1000 + 1 = 1001. 1000 + 0 = 1000 < 1001
		_, err := svc.PlaceBid(ctx, 1, 300, 1000)
		if err != ErrBidTooLow {
			t.Errorf("expected ErrBidTooLow, got %v", err)
		}
	})

	t.Run("concurrent bid fails", func(t *testing.T) {
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				return makeActiveAuction(1000, 5000, 0), nil
			},
			updateAuctionBidFn: func(_ context.Context, _ uint64, _, _ uint64, _ uint64) error {
				return errors.New("concurrent update conflict")
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, err := svc.PlaceBid(ctx, 1, 300, 2000)
		if err == nil {
			t.Fatal("expected error from concurrent bid failure")
		}
	})
}

// ============================================================================
// GetActiveAuctions
// ============================================================================

func TestGetActiveAuctions(t *testing.T) {
	ctx := context.Background()

	t.Run("basic query", func(t *testing.T) {
		repo := &mockTradeRepo{
			listActiveAuctionsFn: func(_ context.Context, filter model.AuctionFilter) ([]*model.Auction, int, error) {
				if filter.Page != 1 || filter.PageSize != 20 {
					t.Errorf("expected page=1 size=20; got page=%d size=%d", filter.Page, filter.PageSize)
				}
				return []*model.Auction{
					{ID: 1, ItemID: 100, ReservePrice: 50000},
				}, 1, nil
			},
		}
		cacheCalled := false
		cache := &mockCacheRepo{
			cacheAuctionFn: func(_ context.Context, _ *model.Auction) error {
				cacheCalled = true
				return nil
			},
		}
		svc := newAuctionService(repo, cache)
		auctions, total, err := svc.GetActiveAuctions(ctx, model.AuctionFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if total != 1 {
			t.Errorf("total = %d; want 1", total)
		}
		if len(auctions) != 1 {
			t.Errorf("got %d auctions; want 1", len(auctions))
		}
		if !cacheCalled {
			t.Error("CacheAuction was not called")
		}
	})

	t.Run("pagination clamping", func(t *testing.T) {
		var captured model.AuctionFilter
		repo := &mockTradeRepo{
			listActiveAuctionsFn: func(_ context.Context, filter model.AuctionFilter) ([]*model.Auction, int, error) {
				captured = filter
				return []*model.Auction{}, 0, nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, _, _ = svc.GetActiveAuctions(ctx, model.AuctionFilter{Page: 0, PageSize: 0})
		if captured.Page != 1 {
			t.Errorf("Page = %d; want 1", captured.Page)
		}
		if captured.PageSize != 20 {
			t.Errorf("PageSize = %d; want 20", captured.PageSize)
		}
	})

	t.Run("page size capped at 100", func(t *testing.T) {
		var captured model.AuctionFilter
		repo := &mockTradeRepo{
			listActiveAuctionsFn: func(_ context.Context, filter model.AuctionFilter) ([]*model.Auction, int, error) {
				captured = filter
				return []*model.Auction{}, 0, nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, _, _ = svc.GetActiveAuctions(ctx, model.AuctionFilter{Page: 1, PageSize: 999})
		if captured.PageSize != 100 {
			t.Errorf("PageSize = %d; want 100", captured.PageSize)
		}
	})

	t.Run("filter by item id", func(t *testing.T) {
		var captured model.AuctionFilter
		repo := &mockTradeRepo{
			listActiveAuctionsFn: func(_ context.Context, filter model.AuctionFilter) ([]*model.Auction, int, error) {
				captured = filter
				return []*model.Auction{}, 0, nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		_, _, _ = svc.GetActiveAuctions(ctx, model.AuctionFilter{ItemID: 42})
		if captured.ItemID != 42 {
			t.Errorf("ItemID = %d; want 42", captured.ItemID)
		}
	})
}

// ============================================================================
// ProcessExpiredAuctions
// ============================================================================

func TestProcessExpiredAuctions(t *testing.T) {
	ctx := context.Background()

	makeExpired := func(id uint64, currentBid, reservePrice uint64) *model.Auction {
		return &model.Auction{
			ID:           id,
			ItemID:       100,
			SellerID:     200,
			CurrentBid:   currentBid,
			BidderID:     300,
			ReservePrice: reservePrice,
			EndTime:      time.Now().Add(-1 * time.Hour), // expired
			Status:       model.AuctionStatusActive,
		}
	}

	t.Run("completed when reserve met", func(t *testing.T) {
		var updatedStatus model.AuctionStatus
		repo := &mockTradeRepo{
			findExpiredAuctionsFn: func(_ context.Context) ([]*model.Auction, error) {
				return []*model.Auction{
					makeExpired(1, 10000, 5000), // currentBid >= reserve
				}, nil
			},
			updateAuctionStatusFn: func(_ context.Context, _ uint64, status model.AuctionStatus) error {
				updatedStatus = status
				return nil
			},
		}
		cacheDelCalled := false
		cacheRemCalled := false
		cache := &mockCacheRepo{
			deleteCachedAuctionFn: func(_ context.Context, _ uint64) error {
				cacheDelCalled = true
				return nil
			},
			removeActiveAuctionFn: func(_ context.Context, _ uint64) error {
				cacheRemCalled = true
				return nil
			},
		}
		svc := newAuctionService(repo, cache)
		count, err := svc.ProcessExpiredAuctions(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("processed count = %d; want 1", count)
		}
		if updatedStatus != model.AuctionStatusCompleted {
			t.Errorf("status = %q; want completed", updatedStatus)
		}
		if !cacheDelCalled {
			t.Error("DeleteCachedAuction was not called")
		}
		if !cacheRemCalled {
			t.Error("RemoveActiveAuction was not called")
		}
	})

	t.Run("expired when reserve not met", func(t *testing.T) {
		var updatedStatus model.AuctionStatus
		repo := &mockTradeRepo{
			findExpiredAuctionsFn: func(_ context.Context) ([]*model.Auction, error) {
				return []*model.Auction{
					makeExpired(2, 3000, 5000), // currentBid < reserve
				}, nil
			},
			updateAuctionStatusFn: func(_ context.Context, _ uint64, status model.AuctionStatus) error {
				updatedStatus = status
				return nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		count, err := svc.ProcessExpiredAuctions(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("processed count = %d; want 1", count)
		}
		if updatedStatus != model.AuctionStatusExpired {
			t.Errorf("status = %q; want expired", updatedStatus)
		}
	})

	t.Run("mixed expired and completed", func(t *testing.T) {
		var statusUpdates []model.AuctionStatus
		repo := &mockTradeRepo{
			findExpiredAuctionsFn: func(_ context.Context) ([]*model.Auction, error) {
				return []*model.Auction{
					makeExpired(3, 10000, 5000), // completed
					makeExpired(4, 1000, 5000),  // expired
				}, nil
			},
			updateAuctionStatusFn: func(_ context.Context, _ uint64, status model.AuctionStatus) error {
				statusUpdates = append(statusUpdates, status)
				return nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		count, err := svc.ProcessExpiredAuctions(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Errorf("processed count = %d; want 2", count)
		}
		if len(statusUpdates) != 2 {
			t.Fatalf("expected 2 status updates, got %d", len(statusUpdates))
		}
		if statusUpdates[0] != model.AuctionStatusCompleted {
			t.Errorf("first = %q; want completed", statusUpdates[0])
		}
		if statusUpdates[1] != model.AuctionStatusExpired {
			t.Errorf("second = %q; want expired", statusUpdates[1])
		}
	})

	t.Run("no expired auctions", func(t *testing.T) {
		repo := &mockTradeRepo{
			findExpiredAuctionsFn: func(_ context.Context) ([]*model.Auction, error) {
				return []*model.Auction{}, nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		count, err := svc.ProcessExpiredAuctions(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("processed count = %d; want 0", count)
		}
	})
}

// ============================================================================
// Price / reserve validation
// ============================================================================

func TestAuctionReservePriceValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("reserve price must be > 0", func(t *testing.T) {
		svc := newAuctionService(&mockTradeRepo{}, &mockCacheRepo{})
		_, err := svc.StartAuction(ctx, 100, 200, 0, 3600)
		if err != ErrAuctionReserveTooLow {
			t.Errorf("expected ErrAuctionReserveTooLow, got %v", err)
		}
	})

	t.Run("reserve price of 1 is valid", func(t *testing.T) {
		repo := &mockTradeRepo{
			createAuctionFn: func(_ context.Context, a *model.Auction) error {
				a.ID = 1
				return nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		auction, err := svc.StartAuction(ctx, 100, 200, 1, 3600)
		if err != nil {
			t.Fatalf("reserve=1 should be valid: %v", err)
		}
		if auction.ReservePrice != 1 {
			t.Errorf("ReservePrice = %d; want 1", auction.ReservePrice)
		}
	})
}

func TestBidIncrementRules(t *testing.T) {
	ctx := context.Background()

	t.Run("minimum increment is 1", func(t *testing.T) {
		// currentBid=1000, minBid=1001
		repo := &mockTradeRepo{
			getAuctionByIDFn: func(_ context.Context, _ uint64) (*model.Auction, error) {
				return &model.Auction{
					ID: 1, SellerID: 200, CurrentBid: 1000, ReservePrice: 5000,
					EndTime: time.Now().Add(24 * time.Hour), Status: model.AuctionStatusActive,
				}, nil
			},
			updateAuctionBidFn: func(_ context.Context, _ uint64, _, _ uint64, _ uint64) error {
				return nil
			},
		}
		svc := newAuctionService(repo, &mockCacheRepo{})
		// 1001 = currentBid + 1, should succeed
		_, err := svc.PlaceBid(ctx, 1, 300, 1001)
		if err != nil {
			t.Errorf("bid of currentBid+1 should succeed: %v", err)
		}
	})
}
