// Package model 定义 GM 管理后台的数据模型。
package model

import (
	"encoding/json"
	"time"
)

// ---- GM 管理员 ----

// GMAdminRole GM 管理员角色。
type GMAdminRole int8

const (
	GMAdminRoleSuperAdmin GMAdminRole = 1 // 超级管理员
	GMAdminRoleOperator   GMAdminRole = 2 // 运营
	GMAdminRoleViewer     GMAdminRole = 3 // 观察者
)

// GMAdminStatus GM 管理员状态。
type GMAdminStatus int8

const (
	GMAdminStatusDisabled GMAdminStatus = 0 // 禁用
	GMAdminStatusEnabled  GMAdminStatus = 1 // 启用
)

// GMAdmin 管理员账号实体，对应数据库 gm_admins 表。
type GMAdmin struct {
	ID           uint64        `json:"id"`
	Username     string        `json:"username"`
	PasswordHash string        `json:"-"`
	Role         GMAdminRole   `json:"role"`
	Status       GMAdminStatus `json:"status"`
	LastLoginAt  *time.Time    `json:"last_login_at"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// IsEnabled 检查管理员是否启用。
func (a *GMAdmin) IsEnabled() bool {
	return a.Status == GMAdminStatusEnabled
}

// IsSuperAdmin 检查是否为超级管理员。
func (a *GMAdmin) IsSuperAdmin() bool {
	return a.Role == GMAdminRoleSuperAdmin
}

// CanWrite 检查是否具有写权限（超级管理员或运营）。
func (a *GMAdmin) CanWrite() bool {
	return a.Role == GMAdminRoleSuperAdmin || a.Role == GMAdminRoleOperator
}

// ---- GM 操作日志 ----

// GMActionType 操作类型。
type GMActionType string

const (
	GMActionLogin          GMActionType = "gm_login"
	GMActionBanPlayer      GMActionType = "ban_player"
	GMActionUnbanPlayer    GMActionType = "unban_player"
	GMActionEditAttribute  GMActionType = "edit_attribute"
	GMActionSendItem       GMActionType = "send_item"
	GMActionAnnouncement   GMActionType = "send_announcement"
	GMActionViewPlayer     GMActionType = "view_player"
	GMActionSearchPlayer   GMActionType = "search_player"
)

// GMAuditTargetType 操作目标类型。
type GMAuditTargetType string

const (
	GMTargetPlayer      GMAuditTargetType = "player"
	GMTargetAnnouncement GMAuditTargetType = "announcement"
	GMTargetSystem      GMAuditTargetType = "system"
)

// GMOperationLog 操作日志实体，对应数据库 gm_operation_logs 表。
type GMOperationLog struct {
	ID         uint64            `json:"id"`
	AdminID    uint64            `json:"admin_id"`
	ActionType GMActionType      `json:"action_type"`
	TargetType GMAuditTargetType `json:"target_type"`
	TargetID   uint64            `json:"target_id"`
	Detail     json.RawMessage   `json:"detail"`
	IPAddress  string            `json:"ip_address"`
	CreatedAt  time.Time         `json:"created_at"`
	AdminName  string            `json:"admin_name,omitempty"` // 关联查询填充
}

// ---- GM 公告 ----

// GMAnnouncementType 公告类型。
type GMAnnouncementType int8

const (
	GMAnnouncementSystem  GMAnnouncementType = 1 // 系统公告
	GMAnnouncementWorld   GMAnnouncementType = 2 // 世界公告
	GMAnnouncementPersonal GMAnnouncementType = 3 // 个人消息
)

// GMAnnouncement 公告实体，对应数据库 gm_announcements 表。
type GMAnnouncement struct {
	ID             uint64              `json:"id"`
	AdminID        uint64              `json:"admin_id"`
	Title          string              `json:"title"`
	Content        string              `json:"content"`
	Type           GMAnnouncementType  `json:"type"`
	TargetPlayerID *uint64             `json:"target_player_id,omitempty"`
	SentAt         time.Time           `json:"sent_at"`
	ExpireAt       *time.Time          `json:"expire_at,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	AdminName      string              `json:"admin_name,omitempty"` // 关联查询填充
}

// ---- GM 封禁 ----

// GMBanType 封禁类型。
type GMBanType int8

const (
	GMBanChatMute    GMBanType = 1 // 禁言
	GMBanTemp        GMBanType = 2 // 临时封号
	GMBanPermanent   GMBanType = 3 // 永久封号
)

// GMBanStatus 封禁状态。
type GMBanStatus int8

const (
	GMBanInactive GMBanStatus = 0 // 已解封
	GMBanActive   GMBanStatus = 1 // 生效中
)

// GMBan 封禁记录实体，对应数据库 gm_bans 表。
type GMBan struct {
	ID        uint64      `json:"id"`
	PlayerID  uint64      `json:"player_id"`
	AdminID   uint64      `json:"admin_id"`
	Reason    string      `json:"reason"`
	BanType   GMBanType   `json:"ban_type"`
	StartsAt  time.Time   `json:"starts_at"`
	EndsAt    *time.Time  `json:"ends_at,omitempty"`
	Status    GMBanStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	AdminName string      `json:"admin_name,omitempty"`  // 关联查询填充
}

// IsActive 检查封禁是否仍生效。
func (b *GMBan) IsActive() bool {
	if b.Status != GMBanActive {
		return false
	}
	if b.BanType == GMBanPermanent {
		return true
	}
	if b.EndsAt != nil && time.Now().After(*b.EndsAt) {
		return false
	}
	return true
}

// ---- GM 会话 ----

// GMClaims JWT 载荷（GM 管理员专用）。
type GMClaims struct {
	AdminID   uint64 `json:"admin_id"`
	Username  string `json:"username"`
	Role      int8   `json:"role"`
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

// ---- GM 请求/响应 ----

// GMLoginRequest GM 登录请求。
type GMLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// GMLoginResponse GM 登录响应。
type GMLoginResponse struct {
	Token     string      `json:"token"`
	AdminID   uint64      `json:"admin_id"`
	Username  string      `json:"username"`
	Role      GMAdminRole `json:"role"`
	ExpiresAt int64       `json:"expires_at"`
}

// GMPlayerSearchRequest 玩家查询请求。
type GMPlayerSearchRequest struct {
	Search string `json:"search" form:"search"`
	Page   int    `json:"page" form:"page"`
	Limit  int    `json:"limit" form:"limit"`
}

// GMPlayerSearchResponse 玩家查询响应。
type GMPlayerSearchResponse struct {
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Items []*GMPLayer `json:"items"`
}

// GMPLayer GM 视角的玩家摘要信息。
type GMPLayer struct {
	ID           uint64 `json:"id"`
	Username     string `json:"username,omitempty"`
	Nickname     string `json:"nickname"`
	Realm        string `json:"realm"`
	Level        int    `json:"level"`
	Power        int64  `json:"power"`
	SpiritStones int64  `json:"spirit_stones"`
	VipLevel     int    `json:"vip_level"`
	Status       int8   `json:"status"`
	Online       bool   `json:"online"`
	LastLoginAt  string `json:"last_login_at"`
	CreatedAt    string `json:"created_at"`
}

// GMEditAttributeRequest 编辑属性请求。
type GMEditAttributeRequest struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// GMBanRequest 封禁请求。
type GMBanRequest struct {
	Reason   string `json:"reason"`
	BanType  int8   `json:"ban_type"`
	Duration int    `json:"duration"` // 封禁时长（分钟），0=永久
}

// GMAnnouncementRequest 发送公告请求。
type GMAnnouncementRequest struct {
	Title          string `json:"title"`
	Content        string `json:"content"`
	Type           int8   `json:"type"`
	TargetPlayerID uint64 `json:"target_player_id,omitempty"`
}

// GMSendItemRequest 发送物品请求。
type GMSendItemRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

// GMServerStats 服务器统计信息。
type GMServerStats struct {
	OnlinePlayers int    `json:"online_players"`
	TotalPlayers  int64  `json:"total_players"`
	TotalUsers    int64  `json:"total_users"`
	TodayDAU      int64  `json:"today_dau"`
	PeakDAU       int64  `json:"peak_dau"`
	Uptime        string `json:"uptime"`
	Version       string `json:"version"`
	MySQLStatus   string `json:"mysql_status"`
	RedisStatus   string `json:"redis_status"`
}

// GMOperationLogResponse 操作日志响应。
type GMOperationLogResponse struct {
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Items []*GMOperationLog `json:"items"`
}

// GMAPIResponse 统一 GM API 响应格式。
type GMAPIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
