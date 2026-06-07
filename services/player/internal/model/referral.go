package model

import "time"

// InviteCode 玩家邀请码
type InviteCode struct {
	ID         int64     `json:"id"`
	PlayerID   int64     `json:"player_id"`
	InviteCode string    `json:"invite_code"`
	TimesUsed  int       `json:"times_used"`
	CreatedAt  time.Time `json:"created_at"`
}

// ReferralRecord 推荐记录
type ReferralRecord struct {
	ID                 int64     `json:"id"`
	InviterID          int64     `json:"inviter_id"`
	InviteeID          int64     `json:"invitee_id"`
	InviteeRealmReached int8     `json:"invitee_realm_reached"` // bitmask: bit0=筑基, bit1=元婴, bit2=化神, bit3=大乘
	RewardClaimed      int8     `json:"reward_claimed"`         // bitmask: 已领取奖励档位
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ============================================================
// 里程碑常量 — 境界奖励档位（bitmask 位定义）
// ============================================================

const (
	// MilestoneBitBase   里程碑位: 筑基
	MilestoneBitBase = 1 << iota // 1
	// MilestoneBitNascent 里程碑位: 元婴
	MilestoneBitNascent // 2
	// MilestoneBitSpirit 里程碑位: 化神
	MilestoneBitSpirit // 4
	// MilestoneBitAscend 里程碑位: 大乘
	MilestoneBitAscend // 8
)

// MilestoneRealmThreshold 里程碑对应的境界要求（参考 player.Realm 字段）
const (
	MilestoneRealmBase   = 3  // 筑基
	MilestoneRealmNascent = 5 // 元婴
	MilestoneRealmSpirit  = 6 // 化神
	MilestoneRealmAscend  = 8 // 大乘
)

// MilestoneInfo 里程碑信息（用于前端展示）
type MilestoneInfo struct {
	Bit          int8   `json:"bit"`
	RealmID      int32  `json:"realm_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	RewardDesc   string `json:"reward_desc"`
}

// Milestones 所有里程碑定义
var Milestones = []MilestoneInfo{
	{Bit: MilestoneBitBase, RealmID: MilestoneRealmBase, Name: "筑基", Description: "被邀请者达到筑基期", RewardDesc: "100灵石"},
	{Bit: MilestoneBitNascent, RealmID: MilestoneRealmNascent, Name: "元婴", Description: "被邀请者达到元婴期", RewardDesc: "500灵石 + 丹药"},
	{Bit: MilestoneBitSpirit, RealmID: MilestoneRealmSpirit, Name: "化神", Description: "被邀请者达到化神期", RewardDesc: "2000灵石 + 稀有物品"},
	{Bit: MilestoneBitAscend, RealmID: MilestoneRealmAscend, Name: "大乘", Description: "被邀请者达到大乘期", RewardDesc: "100仙玉"},
}

// ReferralInfo 邀请信息聚合（用于 GET /referral/info 响应）
type ReferralInfo struct {
	InviteCode   string            `json:"invite_code"`
	TimesUsed    int               `json:"times_used"`
	Referrals    []*ReferralDetail `json:"referrals"`
	PendingCount int               `json:"pending_count"`
}

// ReferralDetail 单个被邀请者详情
type ReferralDetail struct {
	InviteeID          int64  `json:"invitee_id"`
	InviteeName        string `json:"invitee_name"`
	InviteeRealm       int32  `json:"invitee_realm"`
	InviteeRealmName   string `json:"invitee_realm_name"`
	RealmReachedBits   int8   `json:"realm_reached_bits"`   // 已达成的里程碑位
	RewardClaimedBits  int8   `json:"reward_claimed_bits"`  // 已领取的里程碑位
	ClaimableBits      int8   `json:"claimable_bits"`       // 可领取但未领取的里程碑位
	Milestones         []*MilestoneStatus `json:"milestones"`
}

// MilestoneStatus 单个里程碑状态
type MilestoneStatus struct {
	Bit     int8   `json:"bit"`
	Name    string `json:"name"`
	Reached bool   `json:"reached"`
	Claimed bool   `json:"claimed"`
	CanClaim bool  `json:"can_claim"`
}

// ApplyInviteCodeRequest 申请邀请码请求
type ApplyInviteCodeRequest struct {
	InviteeID int64  `json:"invitee_id" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

// ClaimRewardRequest 领取奖励请求
type ClaimRewardRequest struct {
	PlayerID  int64 `json:"player_id" binding:"required"`
	InviteeID int64 `json:"invitee_id" binding:"required"`
}
