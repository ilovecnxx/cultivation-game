// Package model 定义交易服务的数据模型，包括挂单、交易记录和拍卖实体。
package model

import (
	"time"
)

// ============================================================================
// 查询过滤条件
// ============================================================================

// ListingFilter 挂单列表查询过滤条件。
type ListingFilter struct {
	SellerID     uint64        `json:"seller_id,omitempty"`     // 卖家 ID（可选）
	ItemID       uint32        `json:"item_id,omitempty"`       // 物品 ID（可选）
	Status       ListingStatus `json:"status,omitempty"`        // 状态（可选）
	CurrencyType CurrencyType  `json:"currency_type,omitempty"` // 货币类型（可选）
	Page         int           `json:"page"`                    // 页码（从 1 开始）
	PageSize     int           `json:"page_size"`               // 每页数量
}

// AuctionFilter 拍卖列表查询过滤条件。
type AuctionFilter struct {
	ItemID   uint32 `json:"item_id,omitempty"` // 物品 ID（可选）
	Page     int    `json:"page"`              // 页码
	PageSize int    `json:"page_size"`         // 每页数量
}

// ============================================================================
// 枚举定义
// ============================================================================

// ListingStatus 挂单状态。
type ListingStatus string

const (
	ListingStatusActive    ListingStatus = "active"    // 上架中
	ListingStatusSold      ListingStatus = "sold"      // 已售出
	ListingStatusCancelled ListingStatus = "cancelled" // 已取消
	ListingStatusExpired   ListingStatus = "expired"   // 已过期
)

// AuctionStatus 拍卖状态。
type AuctionStatus string

const (
	AuctionStatusActive    AuctionStatus = "active"    // 进行中
	AuctionStatusCompleted AuctionStatus = "completed" // 已成交
	AuctionStatusCancelled AuctionStatus = "cancelled" // 已取消
	AuctionStatusExpired   AuctionStatus = "expired"   // 流拍
)

// CurrencyType 货币类型。
type CurrencyType string

const (
	CurrencySpiritStone CurrencyType = "spirit_stone" // 灵石
)

// ============================================================================
// 实体定义
// ============================================================================

// Listing 市场挂单，对应数据库 trade_listings 表。
type Listing struct {
	ID           uint64        `json:"id"`             // 挂单 ID
	SellerID     uint64        `json:"seller_id"`      // 卖家 ID
	SellerName   string        `json:"seller_name"`    // 卖家名称
	ItemID       uint32        `json:"item_id"`        // 物品模板 ID
	ItemName     string        `json:"item_name"`      // 物品名称
	Quantity     uint32        `json:"quantity"`       // 数量
	UnitPrice    uint64        `json:"unit_price"`     // 单价（灵石）
	CurrencyType CurrencyType  `json:"currency_type"`  // 货币类型
	Status       ListingStatus `json:"status"`         // 挂单状态
	CreatedAt    time.Time     `json:"created_at"`     // 创建时间
	ExpiresAt    time.Time     `json:"expires_at"`     // 过期时间
	UpdatedAt    time.Time     `json:"updated_at"`     // 更新时间
}

// IsActive 检查挂单是否处于可交易状态。
func (l *Listing) IsActive() bool {
	return l.Status == ListingStatusActive && l.ExpiresAt.After(time.Now())
}

// TotalPrice 计算指定数量的总价。
func (l *Listing) TotalPrice(quantity uint32) uint64 {
	return l.UnitPrice * uint64(quantity)
}

// Transaction 交易记录，对应数据库 trade_transactions 表。
type Transaction struct {
	ID         uint64    `json:"id"`          // 交易 ID
	ListingID  uint64    `json:"listing_id"`  // 关联挂单 ID
	BuyerID    uint64    `json:"buyer_id"`    // 买家 ID
	SellerID   uint64    `json:"seller_id"`   // 卖家 ID
	ItemID     uint32    `json:"item_id"`     // 物品模板 ID
	Quantity   uint32    `json:"quantity"`    // 数量
	UnitPrice  uint64    `json:"unit_price"`  // 成交单价
	TotalPrice uint64    `json:"total_price"` // 总价
	CreatedAt  time.Time `json:"created_at"`  // 创建时间
}

// Auction 拍卖，对应数据库 trade_auctions 表。
type Auction struct {
	ID           uint64        `json:"id"`             // 拍卖 ID
	ItemID       uint32        `json:"item_id"`        // 物品模板 ID
	SellerID     uint64        `json:"seller_id"`      // 卖家 ID
	CurrentBid   uint64        `json:"current_bid"`    // 当前最高出价
	BidderID     uint64        `json:"bidder_id"`      // 当前最高出价者 ID
	ReservePrice uint64        `json:"reserve_price"`  // 保留价
	EndTime      time.Time     `json:"end_time"`       // 结束时间
	Status       AuctionStatus `json:"status"`         // 拍卖状态
	CreatedAt    time.Time     `json:"created_at"`     // 创建时间
	UpdatedAt    time.Time     `json:"updated_at"`     // 更新时间
}

// IsActive 检查拍卖是否处于进行中状态。
func (a *Auction) IsActive() bool {
	return a.Status == AuctionStatusActive && a.EndTime.After(time.Now())
}

// IsReserveMet 检查是否达到保留价。
func (a *Auction) IsReserveMet() bool {
	return a.CurrentBid >= a.ReservePrice
}

// ============================================================================
// 玩家资产（灵石）
// ============================================================================

// PlayerGold 玩家灵石资产，对应数据库 trade_player_gold 表。
type PlayerGold struct {
	PlayerID  uint64    `json:"player_id"`  // 玩家 ID
	Gold      uint64    `json:"gold"`       // 灵石数量
	Version   uint32    `json:"version"`    // 乐观锁版本号
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}
