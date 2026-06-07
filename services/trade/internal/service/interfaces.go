// Package service 定义交易服务的业务逻辑接口，支持test mock。
package service

import (
	"context"
	"database/sql"
	"time"

	"cultivation-game/services/trade/internal/model"
)

// TradeRepository 交易服务需要的数据访问方法。
// 生产环境由 *mysql.TradeRepo 实现，test中可使用 mock。
type TradeRepository interface {
	// 挂单操作
	CreateListing(ctx context.Context, l *model.Listing) error
	GetListingByID(ctx context.Context, id uint64) (*model.Listing, error)
	UpdateListingStatus(ctx context.Context, id uint64, status model.ListingStatus) error
	ListListings(ctx context.Context, filter model.ListingFilter) ([]*model.Listing, int, error)
	// 玩家资产
	GetPlayerGold(ctx context.Context, playerID uint64) (*model.PlayerGold, error)
	// 事务支持
	WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error
	// 交易记录
	ListTransactions(ctx context.Context, buyerID, sellerID uint64, page, pageSize int) ([]*model.Transaction, int, error)
	// 拍卖操作
	CreateAuction(ctx context.Context, a *model.Auction) error
	GetAuctionByID(ctx context.Context, id uint64) (*model.Auction, error)
	UpdateAuctionBid(ctx context.Context, id uint64, newBid uint64, bidderID uint64, oldBid uint64) error
	UpdateAuctionStatus(ctx context.Context, id uint64, status model.AuctionStatus) error
	ListActiveAuctions(ctx context.Context, filter model.AuctionFilter) ([]*model.Auction, int, error)
	FindExpiredAuctions(ctx context.Context) ([]*model.Auction, error)
}

// CacheRepository 交易服务需要的缓存操作。
// 生产环境由 *redis.CacheRepo 实现，test中可使用 mock。
type CacheRepository interface {
	// 挂单缓存
	CacheListing(ctx context.Context, listing *model.Listing) error
	DeleteCachedListing(ctx context.Context, listingID uint64) error
	AddHotListing(ctx context.Context, listingID uint64, score float64) error
	RemoveHotListing(ctx context.Context, listingID uint64) error
	// 拍卖缓存
	CacheAuction(ctx context.Context, auction *model.Auction) error
	DeleteCachedAuction(ctx context.Context, auctionID uint64) error
	AddActiveAuction(ctx context.Context, auctionID uint64, endTime time.Time) error
	RemoveActiveAuction(ctx context.Context, auctionID uint64) error
}
