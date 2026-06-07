// Package model 定义认证服务的数据模型，包括用户实体和账号状态。
package model

import "time"

// UserStatus 用户账号状态。
type UserStatus int8

const (
	UserStatusNormal  UserStatus = 0 // 正常
	UserStatusBanned  UserStatus = 1 // 封禁
	UserStatusFrozen  UserStatus = 2 // 冻结
	UserStatusDeleted UserStatus = 3 // 已删除
)

// User 用户账号实体，对应数据库 users 表。
type User struct {
	ID           uint64     `json:"id"`            // 用户唯一 ID
	Username     string     `json:"username"`      // 用户名（唯一）
	PasswordHash string     `json:"-"`             // 密码哈希（序列化时隐藏）
	PlayerID     uint64     `json:"player_id"`     // 关联的玩家角色 ID
	Email        string     `json:"email"`         // 电子邮箱
	Status       UserStatus `json:"status"`        // 账号状态
	LastLoginAt  *time.Time `json:"last_login_at"` // 最后登录时间
	LastIP       string     `json:"last_ip"`       // 最后登录 IP
	CreatedAt    time.Time  `json:"created_at"`    // 创建时间
	UpdatedAt    time.Time  `json:"updated_at"`    // 更新时间
}

// IsBanned 检查账号是否被封禁。
func (u *User) IsBanned() bool {
	return u.Status == UserStatusBanned
}

// IsActive 检查账号是否处于正常可用状态。
func (u *User) IsActive() bool {
	return u.Status == UserStatusNormal
}

// SessionInfo 会话信息，存储在 Redis 中。
type SessionInfo struct {
	UserID       uint64 `json:"user_id"`       // 用户 ID
	PlayerID     uint64 `json:"player_id"`     // 玩家 ID
	Username     string `json:"username"`      // 用户名
	AccessToken  string `json:"access_token"`  // 当前访问令牌
	RefreshToken string `json:"refresh_token"` // 当前刷新令牌
	DeviceID     string `json:"device_id"`     // 设备标识
}
