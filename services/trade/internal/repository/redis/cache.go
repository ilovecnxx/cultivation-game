// Package redis 提供交易数据的 Redis 缓存访问，缓存热门挂单和活跃拍卖数据。
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	rd "github.com/redis/go-redis/v9"

	"cultivation-game/services/trade/internal/model"
)

// CacheRepo 交易缓存数据访问对象，使用 Redis 缓存热门数据以减轻 MySQL 压力。
type CacheRepo struct {
	rdb *rd.Client
	log *slog.Logger
}

// Redis 键格式常量。
const (
	listingKey       = "trade:listing:%d"           // trade:listing:<id> 挂单 JSON 缓存
	hotListingsKey   = "trade:hot_listings"          // trade:hot_listings ZSET 热门挂单 ID 列表
	auctionKey       = "trade:auction:%d"            // trade:auction:<id> 拍卖 JSON 缓存
	activeAuctionsKey = "trade:active_auctions"       // trade:active_auctions ZSET 活跃拍卖 ID 列表
)

// 缓存过期时间。
const (
	listingCacheTTL  = 5 * time.Minute // 挂单缓存过期时间
	auctionCacheTTL  = 3 * time.Minute // 拍卖缓存过期时间
	hotListingsMax   = 200             // 热门挂单列表最大数量
	activeAuctionsMax = 100            // 活跃拍卖列表最大数量
)

// NewCacheRepo 创建 CacheRepo。
func NewCacheRepo(rdb *rd.Client, log *slog.Logger) *CacheRepo {
	return &CacheRepo{rdb: rdb, log: log}
}

// ============================================================================
// 挂单缓存操作
// ============================================================================

// CacheListing 缓存单条挂单信息。
func (r *CacheRepo) CacheListing(ctx context.Context, listing *model.Listing) error {
	data, err := json.Marshal(listing)
	if err != nil {
		r.log.ErrorContext(ctx, "序列化挂单缓存失败", "error", err, "listing_id", listing.ID)
		return fmt.Errorf("序列化挂单失败: %w", err)
	}

	key := fmt.Sprintf(listingKey, listing.ID)
	if err := r.rdb.Set(ctx, key, data, listingCacheTTL).Err(); err != nil {
		r.log.WarnContext(ctx, "缓存挂单失败", "error", err, "listing_id", listing.ID)
		return fmt.Errorf("缓存挂单失败: %w", err)
	}
	return nil
}

// GetCachedListing 从缓存获取挂单信息。
func (r *CacheRepo) GetCachedListing(ctx context.Context, listingID uint64) (*model.Listing, error) {
	key := fmt.Sprintf(listingKey, listingID)
	data, err := r.rdb.Get(ctx, key).Bytes()
	if err == rd.Nil {
		return nil, nil // 缓存未命中
	}
	if err != nil {
		return nil, fmt.Errorf("读取挂单缓存失败: %w", err)
	}

	listing := &model.Listing{}
	if err := json.Unmarshal(data, listing); err != nil {
		r.log.ErrorContext(ctx, "反序列化挂单缓存失败", "error", err, "listing_id", listingID)
		return nil, fmt.Errorf("反序列化挂单失败: %w", err)
	}
	return listing, nil
}

// DeleteCachedListing 删除缓存的挂单（挂单状态变更时调用）。
func (r *CacheRepo) DeleteCachedListing(ctx context.Context, listingID uint64) error {
	key := fmt.Sprintf(listingKey, listingID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		r.log.WarnContext(ctx, "删除挂单缓存失败", "error", err, "listing_id", listingID)
		return fmt.Errorf("删除挂单缓存失败: %w", err)
	}
	return nil
}

// AddHotListing 将挂单加入热门列表（按创建时间排序）。
func (r *CacheRepo) AddHotListing(ctx context.Context, listingID uint64, score float64) error {
	if err := r.rdb.ZAdd(ctx, hotListingsKey, rd.Z{
		Score:  score,
		Member: listingID,
	}).Err(); err != nil {
		r.log.WarnContext(ctx, "添加热门挂单失败", "error", err, "listing_id", listingID)
		return fmt.Errorf("添加热门挂单失败: %w", err)
	}

	// 限制列表大小，移除最旧的条目
	r.rdb.ZRemRangeByRank(ctx, hotListingsKey, 0, -(hotListingsMax + 1))
	return nil
}

// GetHotListingIDs 获取热门挂单 ID 列表（按创建时间倒序）。
func (r *CacheRepo) GetHotListingIDs(ctx context.Context, count int) ([]uint64, error) {
	if count <= 0 || count > hotListingsMax {
		count = hotListingsMax
	}

	members, err := r.rdb.ZRevRange(ctx, hotListingsKey, 0, int64(count-1)).Result()
	if err != nil {
		r.log.WarnContext(ctx, "获取热门挂单列表失败", "error", err)
		return nil, fmt.Errorf("获取热门挂单列表失败: %w", err)
	}

	ids := make([]uint64, 0, len(members))
	for _, m := range members {
		var id uint64
		if _, err := fmt.Sscanf(m, "%d", &id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// RemoveHotListing 从热门列表中移除挂单（状态变更时）。
func (r *CacheRepo) RemoveHotListing(ctx context.Context, listingID uint64) error {
	if err := r.rdb.ZRem(ctx, hotListingsKey, listingID).Err(); err != nil {
		r.log.WarnContext(ctx, "移除热门挂单失败", "error", err, "listing_id", listingID)
		return fmt.Errorf("移除热门挂单失败: %w", err)
	}
	return nil
}

// ============================================================================
// 拍卖缓存操作
// ============================================================================

// CacheAuction 缓存单条拍卖信息。
func (r *CacheRepo) CacheAuction(ctx context.Context, auction *model.Auction) error {
	data, err := json.Marshal(auction)
	if err != nil {
		r.log.ErrorContext(ctx, "序列化拍卖缓存失败", "error", err, "auction_id", auction.ID)
		return fmt.Errorf("序列化拍卖失败: %w", err)
	}

	key := fmt.Sprintf(auctionKey, auction.ID)
	if err := r.rdb.Set(ctx, key, data, auctionCacheTTL).Err(); err != nil {
		r.log.WarnContext(ctx, "缓存拍卖失败", "error", err, "auction_id", auction.ID)
		return fmt.Errorf("缓存拍卖失败: %w", err)
	}
	return nil
}

// GetCachedAuction 从缓存获取拍卖信息。
func (r *CacheRepo) GetCachedAuction(ctx context.Context, auctionID uint64) (*model.Auction, error) {
	key := fmt.Sprintf(auctionKey, auctionID)
	data, err := r.rdb.Get(ctx, key).Bytes()
	if err == rd.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取拍卖缓存失败: %w", err)
	}

	auction := &model.Auction{}
	if err := json.Unmarshal(data, auction); err != nil {
		r.log.ErrorContext(ctx, "反序列化拍卖缓存失败", "error", err, "auction_id", auctionID)
		return nil, fmt.Errorf("反序列化拍卖失败: %w", err)
	}
	return auction, nil
}

// DeleteCachedAuction 删除缓存的拍卖信息。
func (r *CacheRepo) DeleteCachedAuction(ctx context.Context, auctionID uint64) error {
	key := fmt.Sprintf(auctionKey, auctionID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		r.log.WarnContext(ctx, "删除拍卖缓存失败", "error", err, "auction_id", auctionID)
		return fmt.Errorf("删除拍卖缓存失败: %w", err)
	}
	return nil
}

// AddActiveAuction 将拍卖加入活跃列表（按结束时间排序）。
func (r *CacheRepo) AddActiveAuction(ctx context.Context, auctionID uint64, endTime time.Time) error {
	score := float64(endTime.Unix())
	if err := r.rdb.ZAdd(ctx, activeAuctionsKey, rd.Z{
		Score:  score,
		Member: auctionID,
	}).Err(); err != nil {
		r.log.WarnContext(ctx, "添加活跃拍卖失败", "error", err, "auction_id", auctionID)
		return fmt.Errorf("添加活跃拍卖失败: %w", err)
	}

	r.rdb.ZRemRangeByRank(ctx, activeAuctionsKey, 0, -(activeAuctionsMax + 1))
	return nil
}

// GetActiveAuctionIDs 获取活跃拍卖 ID 列表（按结束时间升序，即将结束的在前）。
func (r *CacheRepo) GetActiveAuctionIDs(ctx context.Context, count int) ([]uint64, error) {
	if count <= 0 || count > activeAuctionsMax {
		count = activeAuctionsMax
	}

	members, err := r.rdb.ZRange(ctx, activeAuctionsKey, 0, int64(count-1)).Result()
	if err != nil {
		r.log.WarnContext(ctx, "获取活跃拍卖列表失败", "error", err)
		return nil, fmt.Errorf("获取活跃拍卖列表失败: %w", err)
	}

	ids := make([]uint64, 0, len(members))
	for _, m := range members {
		var id uint64
		if _, err := fmt.Sscanf(m, "%d", &id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// RemoveActiveAuction 从活跃列表中移除拍卖。
func (r *CacheRepo) RemoveActiveAuction(ctx context.Context, auctionID uint64) error {
	if err := r.rdb.ZRem(ctx, activeAuctionsKey, auctionID).Err(); err != nil {
		r.log.WarnContext(ctx, "移除活跃拍卖失败", "error", err, "auction_id", auctionID)
		return fmt.Errorf("移除活跃拍卖失败: %w", err)
	}
	return nil
}
