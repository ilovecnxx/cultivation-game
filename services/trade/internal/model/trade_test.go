package model

import (
	"testing"
	"time"
)

// ============================================================================
// Listing status transitions
// ============================================================================

func TestListingStatusTransitions(t *testing.T) {
	tests := []struct {
		name     string
		status   ListingStatus
		expected bool // IsActive result
		desc     string
	}{
		{name: "active is active", status: ListingStatusActive, expected: true, desc: "should be active"},
		{name: "sold is not active", status: ListingStatusSold, expected: false, desc: "should NOT be active"},
		{name: "cancelled is not active", status: ListingStatusCancelled, expected: false, desc: "should NOT be active"},
		{name: "expired is not active", status: ListingStatusExpired, expected: false, desc: "should NOT be active"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Listing{
				Status:    tt.status,
				ExpiresAt: time.Now().Add(24 * time.Hour), // far in the future
			}
			active := l.IsActive()
			if active != tt.expected {
				t.Errorf("Listing{Status: %q}.IsActive() = %v; want %v (%s)", tt.status, active, tt.expected, tt.desc)
			}
		})
	}
}

func TestListingIsActiveWithExpiry(t *testing.T) {
	t.Run("active but expired returns false", func(t *testing.T) {
		l := &Listing{
			Status:    ListingStatusActive,
			ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
		}
		if l.IsActive() {
			t.Error("expected expired listing to be inactive")
		}
	})

	t.Run("active with future expiry returns true", func(t *testing.T) {
		l := &Listing{
			Status:    ListingStatusActive,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		if !l.IsActive() {
			t.Error("expected active listing with future expiry to be active")
		}
	})
}

// ============================================================================
// Listing.TotalPrice
// ============================================================================

func TestListingTotalPrice(t *testing.T) {
	tests := []struct {
		name     string
		unitPrice uint64
		quantity uint32
		want     uint64
	}{
		{name: "single item", unitPrice: 100, quantity: 1, want: 100},
		{name: "multiple items", unitPrice: 50, quantity: 10, want: 500},
		{name: "zero price", unitPrice: 0, quantity: 5, want: 0},
		{name: "large price", unitPrice: 999999, quantity: 9999, want: 999999 * 9999},
		{name: "single quantity", unitPrice: 1, quantity: 1, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Listing{UnitPrice: tt.unitPrice}
			got := l.TotalPrice(tt.quantity)
			if got != tt.want {
				t.Errorf("TotalPrice(%d) = %d; want %d", tt.quantity, got, tt.want)
			}
		})
	}
}

// ============================================================================
// Auction state machine
// ============================================================================

func TestAuctionIsActive(t *testing.T) {
	tests := []struct {
		name   string
		status AuctionStatus
		active bool
	}{
		{name: "active is active", status: AuctionStatusActive, active: true},
		{name: "completed is not active", status: AuctionStatusCompleted, active: false},
		{name: "cancelled is not active", status: AuctionStatusCancelled, active: false},
		{name: "expired is not active", status: AuctionStatusExpired, active: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Auction{
				Status:  tt.status,
				EndTime: time.Now().Add(24 * time.Hour),
			}
			if got := a.IsActive(); got != tt.active {
				t.Errorf("Auction{Status: %q}.IsActive() = %v; want %v", tt.status, got, tt.active)
			}
		})
	}
}

func TestAuctionIsActiveEndTime(t *testing.T) {
	t.Run("active status but past end time", func(t *testing.T) {
		a := &Auction{
			Status:  AuctionStatusActive,
			EndTime: time.Now().Add(-1 * time.Hour),
		}
		if a.IsActive() {
			t.Error("auction past end time should not be active")
		}
	})

	t.Run("active status with future end time", func(t *testing.T) {
		a := &Auction{
			Status:  AuctionStatusActive,
			EndTime: time.Now().Add(24 * time.Hour),
		}
		if !a.IsActive() {
			t.Error("auction with future end time should be active")
		}
	})
}

// ============================================================================
// Auction.IsReserveMet
// ============================================================================

func TestAuctionIsReserveMet(t *testing.T) {
	tests := []struct {
		name        string
		currentBid  uint64
		reservePrice uint64
		met         bool
	}{
		{name: "bid equals reserve", currentBid: 1000, reservePrice: 1000, met: true},
		{name: "bid exceeds reserve", currentBid: 1500, reservePrice: 1000, met: true},
		{name: "bid below reserve", currentBid: 500, reservePrice: 1000, met: false},
		{name: "no bids yet", currentBid: 0, reservePrice: 1000, met: false},
		{name: "zero reserve", currentBid: 0, reservePrice: 0, met: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Auction{
				CurrentBid:   tt.currentBid,
				ReservePrice: tt.reservePrice,
			}
			if got := a.IsReserveMet(); got != tt.met {
				t.Errorf("IsReserveMet() = %v; want %v", got, tt.met)
			}
		})
	}
}

// ============================================================================
// Currency validation
// ============================================================================

func TestCurrencyTypes(t *testing.T) {
	if CurrencySpiritStone != "spirit_stone" {
		t.Errorf("CurrencySpiritStone = %q; want %q", CurrencySpiritStone, "spirit_stone")
	}
}

// ============================================================================
// Listing status constants
// ============================================================================

func TestListingStatusConstants(t *testing.T) {
	statuses := map[ListingStatus]string{
		ListingStatusActive:    "active",
		ListingStatusSold:      "sold",
		ListingStatusCancelled: "cancelled",
		ListingStatusExpired:   "expired",
	}

	for status, expected := range statuses {
		if string(status) != expected {
			t.Errorf("ListingStatus(%q) = %q; want %q", string(status), string(status), expected)
		}
	}
}

// ============================================================================
// Auction status constants
// ============================================================================

func TestAuctionStatusConstants(t *testing.T) {
	statuses := map[AuctionStatus]string{
		AuctionStatusActive:    "active",
		AuctionStatusCompleted: "completed",
		AuctionStatusCancelled: "cancelled",
		AuctionStatusExpired:   "expired",
	}

	for status, expected := range statuses {
		if string(status) != expected {
			t.Errorf("AuctionStatus(%q) = %q; want %q", string(status), string(status), expected)
		}
	}
}

// ============================================================================
// ListingFilter
// ============================================================================

func TestListingFilterDefaults(t *testing.T) {
	f := ListingFilter{}
	if f.Page != 0 || f.PageSize != 0 {
		t.Error("default filter should have zero page/pageSize")
	}
}

// ============================================================================
// AuctionFilter
// ============================================================================

func TestAuctionFilterDefaults(t *testing.T) {
	f := AuctionFilter{}
	if f.Page != 0 || f.PageSize != 0 {
		t.Error("default auction filter should have zero page/pageSize")
	}
}

// ============================================================================
// Transaction structure
// ============================================================================

func TestTransactionFields(t *testing.T) {
	now := time.Now()
	trans := &Transaction{
		ID:         1,
		ListingID:  100,
		BuyerID:    200,
		SellerID:   300,
		ItemID:     400,
		Quantity:   5,
		UnitPrice:  1000,
		TotalPrice: 5000,
		CreatedAt:  now,
	}

	if trans.ID != 1 { t.Errorf("ID = %d; want 1", trans.ID) }
	if trans.ListingID != 100 { t.Errorf("ListingID = %d; want 100", trans.ListingID) }
	if trans.BuyerID != 200 { t.Errorf("BuyerID = %d; want 200", trans.BuyerID) }
	if trans.SellerID != 300 { t.Errorf("SellerID = %d; want 300", trans.SellerID) }
	if trans.ItemID != 400 { t.Errorf("ItemID = %d; want 400", trans.ItemID) }
	if trans.Quantity != 5 { t.Errorf("Quantity = %d; want 5", trans.Quantity) }
	if trans.UnitPrice != 1000 { t.Errorf("UnitPrice = %d; want 1000", trans.UnitPrice) }
	if trans.TotalPrice != 5000 { t.Errorf("TotalPrice = %d; want 5000", trans.TotalPrice) }
	if !trans.CreatedAt.Equal(now) { t.Error("CreatedAt mismatch") }
}

// ============================================================================
// PlayerGold structure
// ============================================================================

func TestPlayerGoldFields(t *testing.T) {
	now := time.Now()
	pg := &PlayerGold{
		PlayerID:  42,
		Gold:      100000,
		Version:   3,
		UpdatedAt: now,
	}
	if pg.PlayerID != 42 { t.Errorf("PlayerID = %d; want 42", pg.PlayerID) }
	if pg.Gold != 100000 { t.Errorf("Gold = %d; want 100000", pg.Gold) }
	if pg.Version != 3 { t.Errorf("Version = %d; want 3", pg.Version) }
}
