// Package anticheat 反作弊与滥用预防系统。
//
// 入口文件，提供 AntiCheatManager 整合所有检测模块。
package anticheat

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// Options 反作弊系统初始化配置。
type Options struct {
	// Redis 客户端（必填，用于限流和缓存）
	RedisClient *redis.Client
	// MongoDB 集合（可选，用于持久化违规记录）
	MongoCollection *mongo.Collection
	// 高严重度自动临时封禁时长（默认 1 小时）
	BanDuration time.Duration
}

// Manager 反作弊管理器，整合验证器与报告器。
//
// 使用方式：
//
//	mgr := anticheat.NewManager(anticheat.Options{
//	    RedisClient: redisClient,
//	    MongoCollection: mongoColl,
//	})
//
//	// 在消息路由前调用
//	result := mgr.ValidateAction(ctx, playerID, msgID, action, speedStat, ip)
type Manager struct {
	RateLimiter      *ActionRateLimiter
	CombatSpeed      *CombatSpeedCheck
	Economy          *EconomyGuard
	LoginPattern     *LoginPatternCheck
	Reporter         *Reporter
	opts             Options
	logger           *slog.Logger

	// 是否启用各模块（允许按需关闭）
	enableRateLimit  bool
	enableCombatSpeed bool
	enableEconomy    bool
	enableLoginCheck bool

	mu sync.RWMutex
}

// NewManager 创建反作弊管理器，整合所有子模块。
func NewManager(opts Options) *Manager {
	if opts.BanDuration <= 0 {
		opts.BanDuration = DefaultBanDuration()
	}

	m := &Manager{
		RateLimiter:      NewActionRateLimiter(opts.RedisClient),
		CombatSpeed:      NewCombatSpeedCheck(opts.RedisClient),
		Economy:          NewEconomyGuard(opts.RedisClient),
		LoginPattern:     NewLoginPatternCheck(opts.RedisClient),
		Reporter:         NewReporter(opts.MongoCollection, opts.RedisClient, opts.BanDuration),
		opts:             opts,
		logger:           slog.Default().With("module", "anticheat"),
		enableRateLimit:  true,
		enableCombatSpeed: true,
		enableEconomy:    true,
		enableLoginCheck: true,
	}

	m.logger.Info("反作弊系统初始化完成",
		"redis", opts.RedisClient != nil,
		"mongo", opts.MongoCollection != nil,
		"ban_duration", opts.BanDuration,
	)

	// 启动在线时长清理协程（每天凌晨 3 点清理超过 7 天的记录）
	go m.cleanupLoop()

	return m
}

// ValidateActionResult 校验结果。
type ValidateActionResult struct {
	// Allowed 是否允许该操作通过
	Allowed bool
	// RetryAfterSec 建议重试等待秒数（限流相关）
	RetryAfterSec int
	// Blocked 是否应该阻止该操作
	Blocked bool
	// BanInfo 如果触发自动封禁，返回封禁信息
	BanInfo *TempBanInfo
	// Reason 拒绝或违规原因
	Reason string
}

// ValidateAction 对指定玩家的行为进行全面反作弊校验。
// 这是网关消息路由前调用的主入口。
//
// 参数说明：
//   - ctx: 上下文
//   - playerID: 玩家 ID
//   - msgID: 消息 ID（参考 protocol 中的消息类型）
//   - action: 行为名称（如 "breakthrough"、"combat_attack"）
//   - speedStat: 玩家速度属性值（战斗相关校验使用，非战斗场景传 0）
//   - combatID: 战斗实例 ID（非战斗场景传 ""）
//   - playerRealm: 玩家当前境界（经济校验使用，非交易场景传 ""）
//   - ip: 玩家 IP 地址
func (m *Manager) ValidateAction(ctx context.Context, playerID uint64, msgID uint32, action string, speedStat float64, combatID string, playerRealm string, ip string) ValidateActionResult {
	result := ValidateActionResult{Allowed: true}

	// 1. 检查是否被封禁
	if banned, banInfo := m.Reporter.IsBanned(playerID); banned {
		result.Allowed = false
		result.Blocked = true
		result.BanInfo = banInfo
		result.Reason = "账号已被暂时封禁"
		return result
	}

	// 2. 动作级滑动窗口限流
	m.mu.RLock()
	enableRL := m.enableRateLimit
	m.mu.RUnlock()
	if enableRL {
		allowed, retryAfter := m.RateLimiter.Allow(ctx, action, playerID)
		if !allowed {
			// 记录违规
			evidence := map[string]interface{}{
				"action":       action,
				"msg_id":       msgID,
				"retry_after":  retryAfter,
			}
			m.Reporter.ReportWithEvidence(ctx, playerID, ViolationRateLimit, SeverityLow, evidence,
				"行为频率超过限制: "+action)

			result.Allowed = false
			result.RetryAfterSec = retryAfter
			result.Reason = "操作过于频繁"
			return result
		}
	}

	// 3. 战斗速度校验（仅战斗相关消息）
	m.mu.RLock()
	enableCS := m.enableCombatSpeed
	m.mu.RUnlock()
	if enableCS && combatID != "" && speedStat > 0 {
		valid, _, reason := m.CombatSpeed.Validate(playerID, combatID, msgID, speedStat, time.Now().UnixMilli())
		if !valid {
			evidence := map[string]interface{}{
				"combat_id":  combatID,
				"speed_stat": speedStat,
				"msg_id":     msgID,
				"reason":     reason,
			}
			banInfo := m.Reporter.ReportWithEvidence(ctx, playerID, ViolationCombatSpeed, SeverityMedium, evidence, reason)

			result.Allowed = false
			result.BanInfo = banInfo
			result.Reason = reason
			return result
		}
	}

	return result
}

// ValidateEconomy 经济交易校验（独立方法，因为需要更详细的参数）。
func (m *Manager) ValidateEconomy(ctx context.Context, record EconomyRecord, ip string) ValidateActionResult {
	result := ValidateActionResult{Allowed: true}

	// 检查封禁
	if banned, banInfo := m.Reporter.IsBanned(record.PlayerID); banned {
		result.Allowed = false
		result.Blocked = true
		result.BanInfo = banInfo
		result.Reason = "账号已被暂时封禁"
		return result
	}

	m.mu.RLock()
	enableEcon := m.enableEconomy
	m.mu.RUnlock()
	if !enableEcon {
		return result
	}

	suspicious, severity, reason, evidence := m.Economy.CheckTransaction(ctx, record)
	if suspicious {
		banInfo := m.Reporter.ReportWithEvidence(ctx, record.PlayerID, ViolationEconomyPrice, Severity(severity), evidence, reason)
		result.Allowed = false
		result.BanInfo = banInfo
		result.Reason = reason
		return result
	}

	return result
}

// CheckLoginPattern 执行登录模式检测。
func (m *Manager) CheckLoginPattern(ctx context.Context, playerID uint64, msgID uint32, action string) ValidateActionResult {
	result := ValidateActionResult{Allowed: true}

	// 记录在线时长
	m.mu.RLock()
	enableLC := m.enableLoginCheck
	m.mu.RUnlock()

	if !enableLC {
		// 即使禁用检测，也记录在线时长供其他系统使用
		m.LoginPattern.RecordOnlineDuration(ctx, playerID, 0)
		return result
	}

	// 检查重复动作
	suspicious, severity, reason, evidence := m.LoginPattern.CheckRepeatedActions(ctx, playerID, action, msgID)
	if suspicious {
		banInfo := m.Reporter.ReportWithEvidence(ctx, playerID, ViolationLoginRepeated, Severity(severity), evidence, reason)
		result.Allowed = false
		result.BanInfo = banInfo
		result.Reason = reason
		return result
	}

	return result
}

// CheckAbnormalOnline 主动检查玩家的异常在线模式（可周期性调用）。
func (m *Manager) CheckAbnormalOnline(ctx context.Context, playerID uint64) ValidateActionResult {
	result := ValidateActionResult{Allowed: true}

	suspicious, severity, reason, evidence := m.LoginPattern.CheckAbnormalOnline(ctx, playerID)
	if suspicious {
		m.Reporter.ReportWithEvidence(ctx, playerID, ViolationLoginOnline, Severity(severity), evidence, reason)
		if Severity(severity) == SeverityHigh {
			result.Allowed = false
			result.Reason = reason
			return result
		}
	}

	return result
}

// RecordOnlineDuration 记录玩家在线时长增量。
func (m *Manager) RecordOnlineDuration(ctx context.Context, playerID uint64, durationSeconds float64) {
	m.LoginPattern.RecordOnlineDuration(ctx, playerID, durationSeconds)
}

// ============================================================
// 配置管理
// ============================================================

// SetRateLimitRule 动态设置限流规则。
func (m *Manager) SetRateLimitRule(action string, maxCount int, window time.Duration) {
	m.RateLimiter.SetLimit(action, maxCount, window)
}

// SetRealmMedians 批量设置各境界价格中位数。
func (m *Manager) SetRealmMedians(medians map[string]float64) {
	m.Economy.SetRealmMediansBulk(medians)
}

// SetEconomyConfig 动态设置经济检测配置。
func (m *Manager) SetEconomyConfig(cfg EconomyConfig) {
	m.Economy.UpdateConfig(cfg)
}

// SetLoginPatternConfig 动态设置登录模式检测配置。
func (m *Manager) SetLoginPatternConfig(cfg LoginPatternConfig) {
	m.LoginPattern.UpdateConfig(cfg)
}

// EnableModule 启用/禁用某个检测模块。
func (m *Manager) EnableModule(module string, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch module {
	case "rate_limit":
		m.enableRateLimit = enabled
	case "combat_speed":
		m.enableCombatSpeed = enabled
	case "economy":
		m.enableEconomy = enabled
	case "login_pattern":
		m.enableLoginCheck = enabled
	}
	m.logger.Info("反作弊模块状态变更", "module", module, "enabled", enabled)
}

// cleanupLoop 定期清理过期数据。
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupExpiredBans(context.Background())
	}
}

func (m *Manager) cleanupExpiredBans(ctx context.Context) {
	if m.opts.RedisClient == nil {
		return
	}

	key := "anticheat:active_bans"
	data, err := m.opts.RedisClient.HGetAll(ctx, key).Result()
	if err != nil {
		m.logger.Warn("清理过期封禁失败", "error", err)
		return
	}

	now := time.Now()
	for playerIDStr, banJSON := range data {
		var ban TempBanInfo
		if err := json.Unmarshal([]byte(banJSON), &ban); err != nil {
			continue
		}
		if now.After(ban.ExpiresAt) {
			m.opts.RedisClient.HDel(ctx, key, playerIDStr)
			m.Reporter.activeBans.Delete(ban.PlayerID)
			m.logger.Info("自动清理过期封禁", "player_id", ban.PlayerID)
		}
	}
}

// Stats 返回所有模块的综合统计信息。
func (m *Manager) Stats() map[string]interface{} {
	reporterStats := m.Reporter.Stats()
	return map[string]interface{}{
		"violation_count":   reporterStats.ViolationCount,
		"ban_count":         reporterStats.BanCount,
		"active_bans":       reporterStats.ActiveBansCount,
		"modules_enabled": map[string]bool{
			"rate_limit":    func() bool { m.mu.RLock(); defer m.mu.RUnlock(); return m.enableRateLimit }(),
			"combat_speed":  func() bool { m.mu.RLock(); defer m.mu.RUnlock(); return m.enableCombatSpeed }(),
			"economy":       func() bool { m.mu.RLock(); defer m.mu.RUnlock(); return m.enableEconomy }(),
			"login_pattern": func() bool { m.mu.RLock(); defer m.mu.RUnlock(); return m.enableLoginCheck }(),
		},
	}
}
