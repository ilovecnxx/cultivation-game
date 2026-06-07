// Package model 定义社交服务的数据模型
package model

import "time"

// ============================================================
// 聊天系统模型
// ============================================================

// ChatChannel 聊天频道类型
type ChatChannel string

const (
	ChannelWorld  ChatChannel = "world"   // 世界频道
	ChannelSect   ChatChannel = "sect"    // 宗门频道
	ChannelPrivate ChatChannel = "private" // 私聊频道
	ChannelSystem ChatChannel = "system"  // 系统频道
)

// ChatMessage 聊天消息
type ChatMessage struct {
	ID        string      `bson:"_id" json:"id"`
	Channel   ChatChannel `bson:"channel" json:"channel"`
	SenderID  string      `bson:"sender_id" json:"sender_id"`
	SenderName string     `bson:"sender_name" json:"sender_name"`
	TargetID  string      `bson:"target_id,omitempty" json:"target_id,omitempty"` // 私聊目标; 宗门ID
	Content   string      `bson:"content" json:"content"`
	IsSystem  bool        `bson:"is_system" json:"is_system"`
	CreatedAt time.Time   `bson:"created_at" json:"created_at"`
}

// ============================================================
// 好友系统模型
// ============================================================

// FriendStatus 好友关系状态
type FriendStatus string

const (
	FriendStatusNormal   FriendStatus = "normal"   // 正常好友
	FriendStatusBlacked  FriendStatus = "blacked"  // 被拉黑
)

// Friend 好友关系
type Friend struct {
	UserID    string       `bson:"user_id" json:"user_id"`
	FriendID  string       `bson:"friend_id" json:"friend_id"`
	Status    FriendStatus `bson:"status" json:"status"`
	CreatedAt time.Time    `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time    `bson:"updated_at" json:"updated_at"`
	Remark    string       `bson:"remark,omitempty" json:"remark,omitempty"` // 备注
}

// FriendApply 好友申请
type FriendApply struct {
	ID         string    `bson:"_id" json:"id"`
	FromID     string    `bson:"from_id" json:"from_id"`
	FromName   string    `bson:"from_name" json:"from_name"`
	ToID       string    `bson:"to_id" json:"to_id"`
	Message    string    `bson:"message,omitempty" json:"message,omitempty"`
	Status     string    `bson:"status" json:"status"` // pending / accepted / rejected
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	HandledAt  time.Time `bson:"handled_at,omitempty" json:"handled_at,omitempty"`
}

// ============================================================
// 邮件系统模型
// ============================================================

// MailType 邮件类型
type MailType string

const (
	MailSystem   MailType = "system"   // 系统邮件
	MailPlayer   MailType = "player"   // 玩家邮件
)

// MailAttachment 邮件附件
type MailAttachment struct {
	ItemID   string `bson:"item_id" json:"item_id"`
	ItemName string `bson:"item_name" json:"item_name"`
	Quantity int64  `bson:"quantity" json:"quantity"`
	// 货币类型附件
	CoinType string `bson:"coin_type,omitempty" json:"coin_type,omitempty"` // spirit_stone / contribution / etc.
	CoinAmount int64 `bson:"coin_amount,omitempty" json:"coin_amount,omitempty"`
}

// Mail 邮件
type Mail struct {
	ID          string         `bson:"_id" json:"id"`
	MailType    MailType       `bson:"mail_type" json:"mail_type"`
	Title       string         `bson:"title" json:"title"`
	Content     string         `bson:"content" json:"content"`
	SenderID    string         `bson:"sender_id" json:"sender_id"`
	SenderName  string         `bson:"sender_name" json:"sender_name"`
	ReceiverID  string         `bson:"receiver_id" json:"receiver_id"`
	Attachments []MailAttachment `bson:"attachments,omitempty" json:"attachments,omitempty"`
	IsRead      bool           `bson:"is_read" json:"is_read"`
	IsClaimed   bool           `bson:"is_claimed" json:"is_claimed"` // 附件已领取
	CreatedAt   time.Time      `bson:"created_at" json:"created_at"`
	ExpireAt    time.Time      `bson:"expire_at,omitempty" json:"expire_at,omitempty"` // 过期时间(系统邮件)
}

// ============================================================
// 宗门系统模型
// ============================================================

// SectRank 宗门职位
type SectRank string

const (
	SectLeader   SectRank = "leader"   // 宗主
	SectElder    SectRank = "elder"    // 长老
	SectElite    SectRank = "elite"    // 精英
	SectRankMember   SectRank = "member"   // 普通成员
)

// Sect 宗门
type Sect struct {
	ID          string    `bson:"_id" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Level       int       `bson:"level" json:"level"`
	Experience  int64     `bson:"experience" json:"experience"`
	Funds       int64     `bson:"funds" json:"funds"` // 宗门资金
	MemberCount int       `bson:"member_count" json:"member_count"`
	MaxMembers  int       `bson:"max_members" json:"max_members"`
	LeaderID    string    `bson:"leader_id" json:"leader_id"`
	LeaderName  string    `bson:"leader_name" json:"leader_name"`
	Notice      string    `bson:"notice,omitempty" json:"notice,omitempty"`       // 公告
	Description string    `bson:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	// 宗门领地相关
	TerritoryID string `bson:"territory_id,omitempty" json:"territory_id,omitempty"`
	Reputation  int64  `bson:"reputation" json:"reputation"` // 宗门声望
}

// SectMember 宗门成员
type SectMember struct {
	SectID    string    `bson:"sect_id" json:"sect_id"`
	UserID    string    `bson:"user_id" json:"user_id"`
	Rank      SectRank  `bson:"rank" json:"rank"`
	Contribution int64  `bson:"contribution" json:"contribution"` // 宗门贡献
	JoinedAt  time.Time `bson:"joined_at" json:"joined_at"`
}

// SectSkill 宗门技能
type SectSkill struct {
	ID          string `bson:"_id" json:"id"`
	SectID      string `bson:"sect_id" json:"sect_id"`
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description" json:"description"`
	Level       int    `bson:"level" json:"level"`
	MaxLevel    int    `bson:"max_level" json:"max_level"`
	CostPerLevel int64 `bson:"cost_per_level" json:"cost_per_level"` // 每级贡献消耗
	EffectType  string `bson:"effect_type" json:"effect_type"`       // 效果类型: cultivation_bonus / combat_bonus / gathering_bonus
	EffectValue float64 `bson:"effect_value" json:"effect_value"`   // 每级效果值
}

// SectMemberSkill 成员已学技能
type SectMemberSkill struct {
	MemberID  string `bson:"member_id" json:"member_id"`
	SkillID   string `bson:"skill_id" json:"skill_id"`
	Level     int    `bson:"level" json:"level"`
}

// SectApply 宗门申请
type SectApply struct {
	ID        string    `bson:"_id" json:"id"`
	SectID    string    `bson:"sect_id" json:"sect_id"`
	UserID    string    `bson:"user_id" json:"user_id"`
	UserName  string    `bson:"user_name" json:"user_name"`
	Message   string    `bson:"message,omitempty" json:"message,omitempty"`
	Status    string    `bson:"status" json:"status"` // pending / accepted / rejected
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// ============================================================
// 双修系统模型
// ============================================================

// DualCultivation 双修关系
type DualCultivation struct {
	ID        string    `bson:"_id" json:"id"`
	UserA     string    `bson:"user_a" json:"user_a"`
	UserB     string    `bson:"user_b" json:"user_b"`
	StartedAt time.Time `bson:"started_at" json:"started_at"`
	// 当前状态: idle / cultivating
	Status string `bson:"status" json:"status"`
	// 效率加成(双方修为不同时): 双方修为差越小加成越高
	EfficiencyBonus float64 `bson:"efficiency_bonus" json:"efficiency_bonus"`
}

// DualCultivationSession 双修会话记录
type DualCultivationSession struct {
	ID        string    `bson:"_id" json:"id"`
	UserA     string    `bson:"user_a" json:"user_a"`
	UserB     string    `bson:"user_b" json:"user_b"`
	Duration  int64     `bson:"duration" json:"duration"` // 修炼时长(秒)
	BonusA    float64   `bson:"bonus_a" json:"bonus_a"`
	BonusB    float64   `bson:"bonus_b" json:"bonus_b"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// ============================================================
// 道侣系统模型 (金丹期解锁)
// ============================================================

// Item 简单物品表示
type Item struct {
	ItemID   uint64 `json:"item_id"`
	Name     string `json:"name,omitempty"`
	Quantity int    `json:"quantity"`
}

// DaolvRelation 道侣关系
type DaolvRelation struct {
	ID            string    `bson:"_id" json:"id"`
	PlayerA       uint64    `bson:"player_a" json:"player_a"`
	PlayerB       uint64    `bson:"player_b" json:"player_b"`
	Intimacy      int       `bson:"intimacy" json:"intimacy"`                      // 亲密度
	Compatibility float64   `bson:"compatibility" json:"compatibility"`            // 契合度 (0-1)
	Level         string    `bson:"level" json:"level"`                            // 等级: 初识/知己/情深/同心/仙侣
	Skills        []string  `bson:"skills" json:"skills"`                          // 已解锁技能
	DailyCultivated int64   `bson:"daily_cultivated" json:"daily_cultivated"`      // 今日双修时间(秒)
	DailyCultivateDate string `bson:"daily_cultivate_date" json:"daily_cultivate_date"` // 记录日期
	GiftItemA     string    `bson:"gift_item_a" json:"gift_item_a"`                // A的定情信物
	GiftItemB     string    `bson:"gift_item_b" json:"gift_item_b"`                // B的定情信物
	LastProposeAt time.Time `bson:"last_propose_at,omitempty" json:"last_propose_at,omitempty"` // 上次求婚时间(冷却)
	StartedAt     time.Time `bson:"started_at" json:"started_at"`
	UpdatedAt     time.Time `bson:"updated_at" json:"updated_at"`
	Status        string    `bson:"status" json:"status"` // normal / divorced
}

// DaolvProposal 道侣申请(求婚)
type DaolvProposal struct {
	ID          string    `bson:"_id" json:"id"`
	FromID      uint64    `bson:"from_id" json:"from_id"`
	FromName    string    `bson:"from_name,omitempty" json:"from_name,omitempty"`
	ToID        uint64    `bson:"to_id" json:"to_id"`
	ToName      string    `bson:"to_name,omitempty" json:"to_name,omitempty"`
	Message     string    `bson:"message,omitempty" json:"message,omitempty"`
	GiftItemID  string    `bson:"gift_item_id,omitempty" json:"gift_item_id,omitempty"`
	GiftItemName string   `bson:"gift_item_name,omitempty" json:"gift_item_name,omitempty"`
	Status      string    `bson:"status" json:"status"` // pending / accepted / rejected
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	HandledAt   time.Time `bson:"handled_at,omitempty" json:"handled_at,omitempty"`
}

// ============================================================
// 宗门任务系统模型
// ============================================================

// MissionType 任务类型
type MissionType string

const (
	MissionGathering MissionType = "gathering" // 采集类
	MissionCombat    MissionType = "combat"    // 战斗类
	MissionDonation  MissionType = "donation"  // 贡献类(捐献灵石)
	MissionCultivate MissionType = "cultivate" // 修炼类
)

// SectMission 宗门任务定义(每日刷新)
type SectMission struct {
	ID                 string      `bson:"_id" json:"id"`
	SectID             string      `bson:"sect_id" json:"sect_id"`
	MissionType        MissionType `bson:"mission_type" json:"mission_type"`
	Description        string      `bson:"description" json:"description"`
	Requirement        int32       `bson:"requirement" json:"requirement"`             // 要求数量
	RewardContribution int64       `bson:"reward_contribution" json:"reward_contribution"`
	RewardExp          int64       `bson:"reward_exp" json:"reward_exp"`
	RewardFunds        int64       `bson:"reward_funds" json:"reward_funds"` // 奖励宗门资金
	Date               string      `bson:"date" json:"date"`                 // 日期 yyyy-mm-dd
}

// MemberMission 成员个人任务进度
type MemberMission struct {
	ID         string       `bson:"_id" json:"id"`
	MemberID   string       `bson:"member_id" json:"member_id"`
	MissionID  string       `bson:"mission_id" json:"mission_id"`
	Progress   int32        `bson:"progress" json:"progress"`     // 当前进度
	Completed  bool         `bson:"completed" json:"completed"`   // 是否完成
	Claimed    bool         `bson:"claimed" json:"claimed"`       // 是否已领取奖励
	Date       string       `bson:"date" json:"date"`             // 日期 yyyy-mm-dd
}

// ============================================================
// 宗门战系统模型
// ============================================================

// WarStatus 宗门战状态
type WarStatus string

const (
	WarPending  WarStatus = "pending"
	WarActive   WarStatus = "active"
	WarFinished WarStatus = "finished"
)

// WarParticipant 宗门战参战代表
type WarParticipant struct {
	UserID    string `bson:"user_id" json:"user_id"`
	UserName  string `bson:"user_name" json:"user_name"`
	SectID    string `bson:"sect_id" json:"sect_id"`
	Score     int    `bson:"score" json:"score"`         // 个人积分
	RemainHp  int    `bson:"remain_hp" json:"remain_hp"` // 车轮战剩余血量
}

// SectWar 宗门战记录
type SectWar struct {
	ID          string           `bson:"_id" json:"id"`
	Season      int              `bson:"season" json:"season"`
	Round       int              `bson:"round" json:"round"`         // 第几轮
	SectA       string           `bson:"sect_a" json:"sect_a"`
	SectAName   string           `bson:"sect_a_name" json:"sect_a_name"`
	SectB       string           `bson:"sect_b" json:"sect_b"`
	SectBName   string           `bson:"sect_b_name" json:"sect_b_name"`
	SectAPlayers []WarParticipant `bson:"sect_a_players" json:"sect_a_players"`
	SectBPlayers []WarParticipant `bson:"sect_b_players" json:"sect_b_players"`
	SectAScore  int              `bson:"sect_a_score" json:"sect_a_score"`
	SectBScore  int              `bson:"sect_b_score" json:"sect_b_score"`
	WinnerSect  string           `bson:"winner_sect,omitempty" json:"winner_sect,omitempty"`
	Status      WarStatus        `bson:"status" json:"status"`
	ScheduledAt time.Time        `bson:"scheduled_at" json:"scheduled_at"`
	FinishedAt  time.Time        `bson:"finished_at,omitempty" json:"finished_at,omitempty"`
	CreatedAt   time.Time        `bson:"created_at" json:"created_at"`
}

// LeagueRank 宗门联赛排名条目
type LeagueRank struct {
	SectID       string `bson:"sect_id" json:"sect_id"`
	SectName     string `bson:"sect_name" json:"sect_name"`
	Score        int    `bson:"score" json:"score"`
	WinCount     int    `bson:"win_count" json:"win_count"`
	TotalMatches int    `bson:"total_matches" json:"total_matches"`
	Rank         int    `bson:"rank" json:"rank"`
}

// SectLeague 宗门联赛赛季
type SectLeague struct {
	Season    int           `bson:"season" json:"season"`
	StartDate time.Time     `bson:"start_date" json:"start_date"`
	EndDate   time.Time     `bson:"end_date" json:"end_date"`
	Rankings  []LeagueRank  `bson:"rankings" json:"rankings"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}
