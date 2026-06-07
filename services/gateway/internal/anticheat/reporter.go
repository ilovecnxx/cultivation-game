// Package anticheat 反作弊与滥用预防系统 - 报告与自动封禁模块。
package anticheat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// ============================================================
// 严重度级别
// ============================================================

// Severity 事件严重度级别。
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// ============================================================
// 反作弊事件类型
// ============================================================

const (
	// ViolationRateLimit 超过动作频率限制（配合 validator.go）
	ViolationRateLimit = "rate_limit_exceeded"
	// ViolationCombatSpeed 战斗速度异常
	ViolationCombatSpeed = "combat_speed_abnormal"
	// ViolationEconomyPrice 交易价格偏离
	ViolationEconomyPrice = "economy_price_deviation"
	// ViolationEconomyRapid 快速买卖套利
	ViolationEconomyRapid = "economy_rapid_trade"
	// ViolationEconomyAccumulation 异常积累
	ViolationEconomyAccumulation = "economy_abnormal_accumulation"
	// ViolationLoginOnline 异常在线时长
	ViolationLoginOnline = "login_abnormal_online"
	// ViolationLoginRepeated 重复动作（脚本）
	ViolationLoginRepeated = "login_repeated_actions"
)

// ============================================================
// SuspiciousActivity 可疑活动记录
// ============================================================

// SuspiciousActivity 可疑活动记录结构体。
// 同时用于内存传输、MongoDB 序列化和 JSON 序列化。
type SuspiciousActivity struct {
	// PlayerID 玩家 ID
	PlayerID uint64 `bson:"player_id" json:"player_id"`
	// Type 违规类型（使用 Violation* 常量）
	Type string `bson:"type" json:"type"`
	// Severity 严重度
	Severity Severity `bson:"severity" json:"severity"`
	// Evidence 违规证据 JSON
	Evidence json.RawMessage `bson:"evidence" json:"evidence"`
	// Timestamp 事件时间
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	// IP 玩家 IP 地址
	IP string `bson:"ip,omitempty" json:"ip,omitempty"`
	// Action 具体动作名称
	Action string `bson:"action,omitempty" json:"action,omitempty"`
	// MsgID 消息 ID
	MsgID uint32 `bson:"msg_id,omitempty" json:"msg_id,omitempty"`
	// Description 人工可读的描述
	Description string `bson:"description,omitempty" json:"description,omitempty"`
}

// ============================================================
// TempBanInfo 临时封禁信息
// ============================================================

// TempBanInfo 临时封禁信息。
type TempBanInfo struct {
	PlayerID  uint64    `json:"player_id"`
	Reason    string    `json:"reason"`
	BanType   string    `json:"ban_type"` // "temporary" / "permanent"
	Duration  int       `json:"duration_seconds"`
	BannedAt  time.Time `json:"banned_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ============================================================
// Reporter 反作弊报告器
// ============================================================

// Reporter 负责记录可疑活动、自动临时封禁、以及持久化存储到 MongoDB。
// 高严重度事件自动触发临时封禁（默认 1 小时）。
type Reporter struct {
	mongoColl   *mongo.Collection
	rdb         *redis.Client
	banDuration time.Duration // 临时封禁时长

	// 内存中的活跃封禁缓存
	activeBans sync.Map // map[uint64]*TempBanInfo

	// 统计
	violationCount int64
	banCount       int64

	logger *slog.Logger
}

// NewReporter 创建反作弊报告器。
//
// mongoColl: MongoDB anticheat_reports 集合的引用（可为 nil，降级为纯内存模式）。
// rdb: Redis 客户端（可为 nil，降级）。
// banDuration: 高严重度自动临时封禁时长。
func NewReporter(mongoColl *mongo.Collection, rdb *redis.Client, banDuration time.Duration) *Reporter {
	if banDuration <= 0 {
		banDuration = time.Hour // 默认 1 小时
	}

	r := &Reporter{
		mongoColl:   mongoColl,
		rdb:         rdb,
		banDuration: banDuration,
		logger:      slog.Default().With("module", "anticheat.reporter"),
	}

	// 启动时从 Redis 加载活跃封禁缓存
	if rdb != nil {
		r.loadActiveBans(context.Background())
	}

	return r
}

// DefaultBanDuration 返回默认临时封禁时长（1 小时）。
func DefaultBanDuration() time.Duration {
	return time.Hour
}

// loadActiveBans 从 Redis 加载当前活跃封禁到内存缓存。
func (r *Reporter) loadActiveBans(ctx context.Context) {
	key := "anticheat:active_bans"
	data, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		r.logger.Warn("加载活跃封禁缓存失败", "error", err)
		return
	}

	for playerIDStr, jsonData := range data {
		var ban TempBanInfo
		if err := json.Unmarshal([]byte(jsonData), &ban); err != nil {
			continue
		}
		if time.Now().Before(ban.ExpiresAt) {
			// 只加载未过期的封禁
			var pid uint64
			fmt.Sscanf(playerIDStr, "%d", &pid)
			r.activeBans.Store(pid, &ban)
		} else {
			// 清除已过期的封禁
			r.rdb.HDel(ctx, key, playerIDStr)
		}
	}
	r.logger.Info("活跃封禁缓存加载完成", "count", func() int {
		count := 0
		r.activeBans.Range(func(_, _ interface{}) bool {
			count++
			return true
		})
		return count
	}())
}

// IsBanned 检查玩家是否被临时封禁。
// 返回 (被封禁, 封禁信息指针)。
// 同时会检查封禁是否已过期，过期则自动解除。
func (r *Reporter) IsBanned(playerID uint64) (bool, *TempBanInfo) {
	banRaw, ok := r.activeBans.Load(playerID)
	if !ok {
		// 检查 Redis（如果内存缓存未命中）
		if r.rdb != nil {
			key := "anticheat:active_bans"
			field := fmt.Sprintf("%d", playerID)
			data, err := r.rdb.HGet(context.Background(), key, field).Result()
			if err == nil && data != "" {
				var ban TempBanInfo
				if err := json.Unmarshal([]byte(data), &ban); err == nil {
					if time.Now().Before(ban.ExpiresAt) {
						r.activeBans.Store(playerID, &ban)
						return true, &ban
					}
					// 过期，清理
					r.rdb.HDel(context.Background(), key, field)
				}
			}
		}
		return false, nil
	}

	ban := banRaw.(*TempBanInfo)
	if time.Now().After(ban.ExpiresAt) {
		// 过期自动解除
		r.activeBans.Delete(playerID)
		if r.rdb != nil {
			key := "anticheat:active_bans"
			field := fmt.Sprintf("%d", playerID)
			r.rdb.HDel(context.Background(), key, field)
		}
		r.logger.Info("临时封禁已自动解除", "player_id", playerID, "reason", ban.Reason)
		return false, nil
	}

	return true, ban
}

// Report 记录并处理一条可疑活动记录。
// 返回创建的封禁信息（如果触发了自动封禁），否则返回 nil。
func (r *Reporter) Report(ctx context.Context, activity *SuspiciousActivity) *TempBanInfo {
	atomic.AddInt64(&r.violationCount, 1)

	if activity.Timestamp.IsZero() {
		activity.Timestamp = time.Now()
	}

	// 持久化到 MongoDB
	if r.mongoColl != nil {
		if err := r.insertToMongo(ctx, activity); err != nil {
			r.logger.Warn("写入 MongoDB 反作弊记录失败",
				"error", err,
				"player_id", activity.PlayerID,
				"type", activity.Type,
			)
		}
	}

	// 记录日志
	logArgs := []interface{}{
		"player_id", activity.PlayerID,
		"type", activity.Type,
		"severity", activity.Severity,
		"description", activity.Description,
	}
	r.logger.Warn("反作弊事件", logArgs...)

	// 高严重度自动触发临时封禁
	if activity.Severity == SeverityHigh {
		return r.applyTempBan(ctx, activity)
	}

	return nil
}

// ReportWithEvidence 便捷方法：直接传入 evidence map 创建报告。
func (r *Reporter) ReportWithEvidence(ctx context.Context, playerID uint64, violationType string, severity Severity, evidence map[string]interface{}, description string) *TempBanInfo {
	evidenceJSON, _ := json.Marshal(evidence)
	return r.Report(ctx, &SuspiciousActivity{
		PlayerID:    playerID,
		Type:        violationType,
		Severity:    severity,
		Evidence:    evidenceJSON,
		Timestamp:   time.Now(),
		Description: description,
	})
}

// ReportAction 便捷方法：记录某次违规动作。
func (r *Reporter) ReportAction(ctx context.Context, playerID uint64, violationType string, severity Severity, evidence map[string]interface{}, ip, action string, msgID uint32, description string) *TempBanInfo {
	evidenceJSON, _ := json.Marshal(evidence)
	return r.Report(ctx, &SuspiciousActivity{
		PlayerID:    playerID,
		Type:        violationType,
		Severity:    severity,
		Evidence:    evidenceJSON,
		Timestamp:   time.Now(),
		IP:          ip,
		Action:      action,
		MsgID:       msgID,
		Description: description,
	})
}

// applyTempBan 执行临时封禁。
func (r *Reporter) applyTempBan(ctx context.Context, activity *SuspiciousActivity) *TempBanInfo {
	now := time.Now()
	banInfo := &TempBanInfo{
		PlayerID:  activity.PlayerID,
		Reason:    fmt.Sprintf("反作弊自动封禁: [%s] %s", activity.Type, activity.Description),
		BanType:   "temporary",
		Duration:  int(r.banDuration.Seconds()),
		BannedAt:  now,
		ExpiresAt: now.Add(r.banDuration),
	}

	// 写入内存缓存
	r.activeBans.Store(activity.PlayerID, banInfo)
	atomic.AddInt64(&r.banCount, 1)

	// 持久化到 Redis（用于跨节点共享 + 重启恢复）
	if r.rdb != nil {
		banJSON, _ := json.Marshal(banInfo)
		key := "anticheat:active_bans"
		field := fmt.Sprintf("%d", activity.PlayerID)
		r.rdb.HSet(ctx, key, field, string(banJSON))
		// 封禁记录也会在到期后自动过期清理
		r.rdb.Expire(ctx, key, r.banDuration+24*time.Hour)
	}

	r.logger.Warn("触发自动临时封禁",
		"player_id", activity.PlayerID,
		"duration", r.banDuration,
		"reason", banInfo.Reason,
	)

	return banInfo
}

// RevokeBan 手动解除某个玩家的封禁。
func (r *Reporter) RevokeBan(ctx context.Context, playerID uint64) error {
	r.activeBans.Delete(playerID)
	if r.rdb != nil {
		key := "anticheat:active_bans"
		field := fmt.Sprintf("%d", playerID)
		if err := r.rdb.HDel(ctx, key, field).Err(); err != nil {
			return fmt.Errorf("Redis 清除封禁记录失败: %w", err)
		}
	}
	r.logger.Info("手动解除封禁", "player_id", playerID)
	return nil
}

// insertToMongo 将可疑活动记录插入 MongoDB。
func (r *Reporter) insertToMongo(ctx context.Context, activity *SuspiciousActivity) error {
	_, err := r.mongoColl.InsertOne(ctx, activity)
	return err
}

// ============================================================
// 统计与状态
// ============================================================

// ReporterStats 报告器统计信息。
type ReporterStats struct {
	ViolationCount  int64 `json:"violation_count"`
	BanCount        int64 `json:"ban_count"`
	ActiveBansCount int   `json:"active_bans_count"`
}

// Stats 返回报告器统计信息。
func (r *Reporter) Stats() ReporterStats {
	activeCount := 0
	r.activeBans.Range(func(_, _ interface{}) bool {
		activeCount++
		return true
	})

	return ReporterStats{
		ViolationCount:  atomic.LoadInt64(&r.violationCount),
		BanCount:        atomic.LoadInt64(&r.banCount),
		ActiveBansCount: activeCount,
	}
}
