// Package service 实现拍卖业务逻辑：发起拍卖、出价验证、过期处理。
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"cultivation-game/services/trade/internal/config"
	"cultivation-game/services/trade/internal/model"
)

// 拍卖相关错误定义。
var (
	ErrAuctionNotFound         = errors.New("拍卖不存在")
	ErrAuctionNotActive        = errors.New("拍卖已结束或已取消")
	ErrAuctionAlreadyEnded     = errors.New("拍卖已结束")
	ErrBidTooLow               = errors.New("出价必须高于当前最高出价")
	ErrBidBelowReserve         = errors.New("出价未达到保留价")
	ErrBidBelowMinIncrement    = errors.New("出价未达到最小加价幅度")
	ErrBidOwnAuction           = errors.New("不能对自己发起的拍卖出价")
	ErrAuctionSellerMismatch   = errors.New("卖家不匹配")
	ErrAuctionReserveTooLow    = errors.New("保留价必须大于 0")
	ErrAuctionDurationTooShort = errors.New("拍卖持续时间过短")
	ErrAuctionDurationTooLong  = errors.New("拍卖持续时间过长")
)

// 出价规则常量。
const (
	minBidIncrement   = 1    // 最小加价幅度（1 灵石）
	maxAuctionHours   = 168  // 最长拍卖时间（7 天）
	minAuctionMinutes = 5    // 最短拍卖时间（5 分钟）
)

// AuctionService 拍卖服务，处理拍卖创建、出价和过期处理。
type AuctionService struct {
	repo  TradeRepository
	cache CacheRepository
	cfg   *config.Config
	log   *slog.Logger
}

// NewAuctionService 创建 AuctionService。
func NewAuctionService(repo TradeRepository, cache CacheRepository, cfg *config.Config, log *slog.Logger) *AuctionService {
	return &AuctionService{
		repo:  repo,
		cache: cache,
		cfg:   cfg,
		log:   log,
	}
}

// ============================================================================
// 发起拍卖
// ============================================================================

// StartAuction 发起拍卖。
// 流程：参数校验 -> 设置结束时间 -> 写入 MySQL -> 写入 Redis 缓存。
func (s *AuctionService) StartAuction(ctx context.Context, itemID uint32, sellerID uint64, reservePrice uint64, durationSeconds uint32) (*model.Auction, error) {
	// 参数校验
	if sellerID == 0 {
		return nil, errors.New("卖家 ID 不能为空")
	}
	if itemID == 0 {
		return nil, errors.New("物品 ID 不能为空")
	}
	if reservePrice == 0 {
		return nil, ErrAuctionReserveTooLow
	}

	// 计算结束时间
	var endTime time.Time
	if durationSeconds <= 0 {
		endTime = time.Now().Add(s.cfg.AuctionDefaultDuration)
	} else {
		duration := time.Duration(durationSeconds) * time.Second
		if duration < minAuctionMinutes*time.Minute {
			return nil, ErrAuctionDurationTooShort
		}
		if duration > maxAuctionHours*time.Hour {
			return nil, ErrAuctionDurationTooLong
		}
		endTime = time.Now().Add(duration)
	}

	auction := &model.Auction{
		ItemID:       itemID,
		SellerID:     sellerID,
		CurrentBid:   0,
		BidderID:     0,
		ReservePrice: reservePrice,
		EndTime:      endTime,
		Status:       model.AuctionStatusActive,
	}

	// 持久化到 MySQL
	if err := s.repo.CreateAuction(ctx, auction); err != nil {
		s.log.ErrorContext(ctx, "创建拍卖失败", "error", err, "seller_id", sellerID)
		return nil, fmt.Errorf("创建拍卖失败: %w", err)
	}

	// 缓存到 Redis（异步风格，失败不影响主流程）
	if err := s.cache.CacheAuction(ctx, auction); err != nil {
		s.log.WarnContext(ctx, "缓存拍卖失败", "error", err, "auction_id", auction.ID)
	}
	if err := s.cache.AddActiveAuction(ctx, auction.ID, auction.EndTime); err != nil {
		s.log.WarnContext(ctx, "添加到活跃拍卖列表失败", "error", err, "auction_id", auction.ID)
	}

	s.log.InfoContext(ctx, "拍卖创建成功",
		"auction_id", auction.ID,
		"seller_id", sellerID,
		"item_id", itemID,
		"reserve_price", reservePrice,
		"end_time", endTime,
	)
	return auction, nil
}

// ============================================================================
// 出价
// ============================================================================

// PlaceBid 对拍卖出价。
// 流程：查询拍卖 -> 验证拍卖状态 -> 验证出价者 -> 验证加价幅度 -> 乐观锁更新 -> 缓存更新。
func (s *AuctionService) PlaceBid(ctx context.Context, auctionID uint64, bidderID uint64, bidAmount uint64) (*model.Auction, error) {
	if bidderID == 0 {
		return nil, errors.New("出价者 ID 不能为空")
	}
	if bidAmount == 0 {
		return nil, ErrBidTooLow
	}

	// 查询拍卖
	auction, err := s.repo.GetAuctionByID(ctx, auctionID)
	if err != nil {
		return nil, fmt.Errorf("查询拍卖失败: %w", err)
	}
	if auction == nil {
		return nil, ErrAuctionNotFound
	}

	// 验证拍卖状态
	if !auction.IsActive() {
		switch auction.Status {
		case model.AuctionStatusCompleted:
			return nil, ErrAuctionAlreadyEnded
		case model.AuctionStatusCancelled:
			return nil, ErrAuctionNotActive
		case model.AuctionStatusExpired:
			return nil, ErrAuctionAlreadyEnded
		default:
			return nil, ErrAuctionNotActive
		}
	}
	if time.Now().After(auction.EndTime) {
		return nil, ErrAuctionAlreadyEnded
	}

	// 验证不能对自己的拍卖出价
	if auction.SellerID == bidderID {
		return nil, ErrBidOwnAuction
	}

	// 验证加价幅度：必须高于当前出价
	if bidAmount <= auction.CurrentBid {
		return nil, ErrBidTooLow
	}

	// 验证最小加价幅度
	minBid := auction.CurrentBid + minBidIncrement
	if bidAmount < minBid {
		return nil, ErrBidBelowMinIncrement
	}

	// 使用乐观锁更新出价（如果 current_bid 在此期间被其他出价者改变，更新会失败）
	if err := s.repo.UpdateAuctionBid(ctx, auctionID, bidAmount, bidderID, auction.CurrentBid); err != nil {
		// 检查是否是并发冲突
		s.log.WarnContext(ctx, "出价更新失败，可能已被其他出价者领先",
			"error", err,
			"auction_id", auctionID,
			"bidder_id", bidderID,
			"bid_amount", bidAmount,
		)
		return nil, fmt.Errorf("出价失败，请刷新后重试: %w", err)
	}

	// 更新本地模型状态
	auction.CurrentBid = bidAmount
	auction.BidderID = bidderID

	// 更新缓存
	if err := s.cache.CacheAuction(ctx, auction); err != nil {
		s.log.WarnContext(ctx, "更新拍卖缓存失败", "error", err, "auction_id", auctionID)
	}

	s.log.InfoContext(ctx, "出价成功",
		"auction_id", auctionID,
		"bidder_id", bidderID,
		"bid_amount", bidAmount,
		"previous_bid", auction.CurrentBid,
	)
	return auction, nil
}

// ============================================================================
// 查询活跃拍卖
// ============================================================================

// GetActiveAuctions 获取活跃拍卖列表，支持分页和物品筛选。
func (s *AuctionService) GetActiveAuctions(ctx context.Context, filter model.AuctionFilter) ([]*model.Auction, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	auctions, total, err := s.repo.ListActiveAuctions(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("查询活跃拍卖列表失败: %w", err)
	}

	// 异步缓存查询结果
	for _, a := range auctions {
		if cacheErr := s.cache.CacheAuction(ctx, a); cacheErr != nil {
			s.log.DebugContext(ctx, "缓存拍卖失败", "error", cacheErr, "auction_id", a.ID)
		}
	}

	return auctions, total, nil
}

// ============================================================================
// 拍卖过期处理
// ============================================================================

// ProcessExpiredAuctions 处理所有已过期的拍卖。
// 对于达到保留价的拍卖标记为 completed，未达到的标记为 expired。
// 此方法应由后台定时任务调用。
func (s *AuctionService) ProcessExpiredAuctions(ctx context.Context) (int, error) {
	expired, err := s.repo.FindExpiredAuctions(ctx)
	if err != nil {
		return 0, fmt.Errorf("查询过期拍卖失败: %w", err)
	}

	if len(expired) == 0 {
		return 0, nil
	}

	completedCount := 0
	expiredCount := 0

	for _, a := range expired {
		if a.IsReserveMet() {
			// 达到保留价，成交
			if err := s.repo.UpdateAuctionStatus(ctx, a.ID, model.AuctionStatusCompleted); err != nil {
				s.log.ErrorContext(ctx, "更新拍卖为成交状态失败",
					"error", err, "auction_id", a.ID)
				continue
			}
			completedCount++
			s.log.InfoContext(ctx, "拍卖成交",
				"auction_id", a.ID,
				"bidder_id", a.BidderID,
				"price", a.CurrentBid,
			)
		} else {
			// 未达到保留价，流拍
			if err := s.repo.UpdateAuctionStatus(ctx, a.ID, model.AuctionStatusExpired); err != nil {
				s.log.ErrorContext(ctx, "更新拍卖为过期状态失败",
					"error", err, "auction_id", a.ID)
				continue
			}
			expiredCount++
			s.log.InfoContext(ctx, "拍卖流拍",
				"auction_id", a.ID,
				"current_bid", a.CurrentBid,
				"reserve_price", a.ReservePrice,
			)
		}

		// 清除缓存
		if err := s.cache.DeleteCachedAuction(ctx, a.ID); err != nil {
			s.log.WarnContext(ctx, "删除拍卖缓存失败", "error", err, "auction_id", a.ID)
		}
		if err := s.cache.RemoveActiveAuction(ctx, a.ID); err != nil {
			s.log.WarnContext(ctx, "移除活跃拍卖缓存失败", "error", err, "auction_id", a.ID)
		}
	}

	s.log.InfoContext(ctx, "拍卖过期处理完成",
		"total", len(expired),
		"completed", completedCount,
		"expired", expiredCount,
	)
	return len(expired), nil
}

// StartAuctionExpiryLoop 启动拍卖过期检查后台循环。
// 在独立 goroutine 中运行，按配置的间隔检查过期拍卖。
func (s *AuctionService) StartAuctionExpiryLoop(ctx context.Context) {
	s.log.InfoContext(ctx, "拍卖过期检查循环已启动",
		"interval", s.cfg.AuctionCheckInterval)
	ticker := time.NewTicker(s.cfg.AuctionCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.InfoContext(ctx, "拍卖过期检查循环已停止")
			return
		case <-ticker.C:
			checkCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			count, err := s.ProcessExpiredAuctions(checkCtx)
			cancel()
			if err != nil {
				s.log.ErrorContext(ctx, "拍卖过期检查失败", "error", err)
			} else if count > 0 {
				s.log.InfoContext(ctx, "拍卖过期检查完成", "processed", count)
			}
		}
	}
}
