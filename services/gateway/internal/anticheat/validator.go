// Package anticheat 反作弊与滥用预防系统。
//
// 提供多层反作弊检测：
//   - ActionRateLimiter: 基于 Redis Sorted Set 的滑动窗口限流（按玩家、按行为）
//   - CombatSpeedCheck: 战斗动作频率校验（防止加速器）
//   - EconomyGuard: 经济交易异常检测
//   - LoginPatternCheck: 异常登录模式检测（挂机、脚本）
package anticheat

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ============================================================
// 行为限流配置
// ============================================================

// ActionLimitConfig 行为限流配置项。
type ActionLimitConfig struct {
	Action    string        // 行为名称，如 "breakthrough"、"combat_attack"
	MaxCount  int           // 时间窗口内最大次数
	Window    time.Duration // 滑动窗口时长
}

// defaultActionLimits 默认限流规则。
var defaultActionLimits = []ActionLimitConfig{
	{Action: "breakthrough", MaxCount: 10, Window: time.Hour},          // 突破：每小时最多10次
	{Action: "combat_attack", MaxCount: 60, Window: time.Minute},       // 战斗攻击：每分钟最多60次
	{Action: "combat_skill", MaxCount: 30, Window: time.Minute},        // 技能释放：每分钟最多30次
	{Action: "shop_buy", MaxCount: 50, Window: time.Minute},           // 商店购买：每分钟最多50次
	{Action: "shop_sell", MaxCount: 30, Window: time.Minute},          // 商店出售：每分钟最多30次
	{Action: "trade_create", MaxCount: 20, Window: time.Minute},       // 创建交易：每分钟最多20次
	{Action: "chat_message", MaxCount: 30, Window: time.Minute},       // 聊天消息：每分钟最多30条
	{Action: "mail_send", MaxCount: 10, Window: time.Minute},          // 发送邮件：每分钟最多10封
	{Action: "dungeon_enter", MaxCount: 20, Window: time.Hour},        // 副本进入：每小时最多20次
	{Action: "cultivation_meditate", MaxCount: 1440, Window: 24 * time.Hour}, // 打坐修炼：每天最多1440次（每分钟1次）
}

// ============================================================
// ActionRateLimiter 滑动窗口限流器
// ============================================================

// rateLimitScript Lua 脚本：原子化滑动窗口限流。
// KEYS[1] = redis key
// ARGV[1] = 当前时间戳（毫秒）
// ARGV[2] = 窗口起始时间戳（毫秒）
// ARGV[3] = 最大允许次数
// ARGV[4] = 窗口时长（毫秒）
// 返回 {allowed, retry_after} 其中 allowed 为 0 或 1。
const rateLimitScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window_start = tonumber(ARGV[2])
local max_count = tonumber(ARGV[3])
local window_ms = tonumber(ARGV[4])

-- 移除窗口外过期条目
redis.call('ZREMRANGEBYSCORE', key, 0, window_start)

-- 统计当前窗口内条目数
local count = redis.call('ZCARD', key)

if count >= max_count then
	-- 获取最早条目的时间戳计算重试等待秒数
	local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
	local retry_after = 0
	if #oldest >= 2 then
		retry_after = math.ceil((tonumber(oldest[2]) + window_ms - now) / 1000)
		if retry_after < 0 then retry_after = 0 end
	end
	return {0, retry_after}
end

-- 允许：将当前请求加入窗口
redis.call('ZADD', key, now, now .. ':' .. math.random())
redis.call('EXPIRE', key, math.ceil(window_ms / 1000) + 60)
return {1, 0}
`

// ActionRateLimiter 基于 Redis Sorted Set 的滑动窗口限流器。
// 按 player_id + action 维度进行精确限流，支持自定义限流规则。
type ActionRateLimiter struct {
	rdb    *redis.Client
	limits map[string]ActionLimitConfig
	script *redis.Script
	logger *slog.Logger
}

// NewActionRateLimiter 创建滑动窗口限流器。
// rdb 为 Redis 客户端；若为 nil 则跳过 Redis 校验（降级为允许所有）。
func NewActionRateLimiter(rdb *redis.Client) *ActionRateLimiter {
	limits := make(map[string]ActionLimitConfig, len(defaultActionLimits))
	for _, l := range defaultActionLimits {
		limits[l.Action] = l
	}
	return &ActionRateLimiter{
		rdb:    rdb,
		limits: limits,
		script: redis.NewScript(rateLimitScript),
		logger: slog.Default().With("module", "anticheat.ratelimiter"),
	}
}

// SetLimit 动态设置或覆盖某个行为的限流规则。
func (arl *ActionRateLimiter) SetLimit(action string, maxCount int, window time.Duration) {
	arl.limits[action] = ActionLimitConfig{
		Action:   action,
		MaxCount: maxCount,
		Window:   window,
	}
}

// redisKey 生成 Redis 键名。
func (arl *ActionRateLimiter) redisKey(action string, playerID uint64) string {
	return fmt.Sprintf("anticheat:rl:%s:%d", action, playerID)
}

// Allow 检查是否允许执行指定行为。
// 返回 (allowed, retryAfterSeconds)。
// allowed=true 表示通过；retryAfterSeconds 为建议重试前等待秒数。
func (arl *ActionRateLimiter) Allow(ctx context.Context, action string, playerID uint64) (bool, int) {
	// 无 Redis 时降级为允许
	if arl.rdb == nil {
		return true, 0
	}

	limit, ok := arl.limits[action]
	if !ok {
		return true, 0 // 未配置限流规则的行为默认允许
	}

	key := arl.redisKey(action, playerID)
	now := time.Now().UnixMilli()
	windowStart := now - limit.Window.Milliseconds()

	result, err := arl.script.Run(ctx, arl.rdb, []string{key},
		now, windowStart, limit.MaxCount, limit.Window.Milliseconds(),
	).Result()
	if err != nil {
		arl.logger.Warn("Redis 限流脚本执行失败，放行请求",
			"error", err, "action", action, "player_id", playerID,
		)
		return true, 0
	}

	results, ok := result.([]interface{})
	if !ok || len(results) < 2 {
		return true, 0
	}

	allowed := results[0].(int64) == 1
	retryAfter := int(results[1].(int64))
	return allowed, retryAfter
}

// Reset 手动清除某个玩家的某个行为限流记录。
func (arl *ActionRateLimiter) Reset(ctx context.Context, action string, playerID uint64) error {
	if arl.rdb == nil {
		return nil
	}
	return arl.rdb.Del(ctx, arl.redisKey(action, playerID)).Err()
}

// GetLimitConfig 获取指定行为的限流配置。
func (arl *ActionRateLimiter) GetLimitConfig(action string) (ActionLimitConfig, bool) {
	limit, ok := arl.limits[action]
	return limit, ok
}

// ============================================================
// CombatSpeedCheck 战斗速度校验
// ============================================================

// lastActionEntry 玩家某次战斗的最后动作时间记录。
type lastActionEntry struct {
	timestamp int64 // UnixMilli
	actionID  uint32
}

// CombatSpeedCheck 校验战斗动作频率是否快于玩家属性允许的速度。
// 防止加速器/变速齿轮类外挂。
type CombatSpeedCheck struct {
	// 内存中记录每个玩家每种战斗动作的最后执行时间
	// map[playerID]map[combatID]*lastActionEntry
	lastActions sync.Map
	rdb         *redis.Client
	logger      *slog.Logger
}

// NewCombatSpeedCheck 创建战斗速度校验器。
func NewCombatSpeedCheck(rdb *redis.Client) *CombatSpeedCheck {
	return &CombatSpeedCheck{
		rdb:    rdb,
		logger: slog.Default().With("module", "anticheat.combat_speed"),
	}
}

// Validate 校验战斗动作是否合法。
// playerID: 玩家 ID
// combatID: 战斗实例 ID（同一场战斗使用相同 ID）
// actionID: 动作 ID（技能/攻击）
// speedStat: 玩家速度属性值（由属性系统提供）
// now: 当前时间戳（毫秒）
// 返回 (valid, cooldownMs, reason)。
// valid=false 时 reason 描述违规原因。
func (csc *CombatSpeedCheck) Validate(playerID uint64, combatID string, actionID uint32, speedStat float64, now int64) (bool, int64, string) {
	// 基于速度属性计算基础冷却（毫秒）
	// 速度越快，动作间最小间隔越短
	// 公式: cooldown = max(100, 1000 / speedStat)，确保至少 100ms
	minCooldown := int64(math.Max(100, 1000.0/speedStat))

	playerKey := fmt.Sprintf("%d:%s", playerID, combatID)

	// 从内存加载上次动作时间
	entryRaw, loaded := csc.lastActions.Load(playerKey)
	var lastEntry *lastActionEntry
	if loaded {
		lastEntry = entryRaw.(*lastActionEntry)
	}

	if lastEntry != nil {
		elapsed := now - lastEntry.timestamp
		if elapsed < minCooldown && elapsed >= 0 {
			return false, minCooldown - elapsed,
				fmt.Sprintf("战斗动作过快: 间隔 %dms < 最小 %dms (速度: %.1f)", elapsed, minCooldown, speedStat)
		}
	}

	// 更新内存记录
	csc.lastActions.Store(playerKey, &lastActionEntry{
		timestamp: now,
		actionID:  actionID,
	})

	return true, 0, ""
}

// Cleanup 清除玩家的战斗速度记录（战斗结束或玩家断线时调用）。
func (csc *CombatSpeedCheck) Cleanup(playerID uint64, combatID string) {
	playerKey := fmt.Sprintf("%d:%s", playerID, combatID)
	csc.lastActions.Delete(playerKey)
}

// CleanupPlayer 清除玩家所有战斗速度记录。
func (csc *CombatSpeedCheck) CleanupPlayer(playerID uint64) {
	csc.lastActions.Range(func(key, value interface{}) bool {
		k := key.(string)
		// 键格式 "playerID:combatID"，匹配 playerID 前缀
		var pid uint64
		if _, err := fmt.Sscanf(k, "%d:", &pid); err == nil && pid == playerID {
			csc.lastActions.Delete(k)
		}
		return true
	})
}

// ============================================================
// EconomyGuard 经济行为异常检测
// ============================================================

// EconomyConfig 经济检测配置。
type EconomyConfig struct {
	// 单笔交易超过此倍数（相对该境界中位数）即标记异常
	PriceDeviationThreshold float64
	// 快速买卖窗口秒数（在此时间内买入后卖出即标记）
	RapidTradeWindowSec int64
	// 单位时间内异常积累次数阈值
	AccumulationRateThreshold int
	// 积累率检测窗口
	AccumulationWindow time.Duration
}

// DefaultEconomyConfig 默认经济检测配置。
func DefaultEconomyConfig() EconomyConfig {
	return EconomyConfig{
		PriceDeviationThreshold:  10.0,                     // 超过中位数10倍
		RapidTradeWindowSec:      60,                       // 60秒内买卖
		AccumulationRateThreshold: 100,                     // 窗口内积累次数
		AccumulationWindow:       time.Hour,                // 1小时窗口
	}
}

// EconomyRecord 单笔交易记录。
type EconomyRecord struct {
	PlayerID   uint64
	Action     string // "buy" / "sell"
	ItemID     uint32
	Amount     int64
	Price      float64
	Realm      string
	Timestamp  int64
}

// EconomyGuard 经济交易异常检测。
// 检测维度：
//   - 单笔价格偏离境界中位数
//   - 快速买卖套利模式
//   - 单位时间内异常积累
type EconomyGuard struct {
	rdb         *redis.Client
	config      EconomyConfig
	realmMedians map[string]float64 // 各境界物品价格中位数缓存
	mu          sync.RWMutex
	logger      *slog.Logger
}

// NewEconomyGuard 创建经济异常检测器。
func NewEconomyGuard(rdb *redis.Client) *EconomyGuard {
	return &EconomyGuard{
		rdb:          rdb,
		config:       DefaultEconomyConfig(),
		realmMedians: make(map[string]float64),
		logger:       slog.Default().With("module", "anticheat.economy"),
	}
}

// UpdateConfig 动态更新经济检测配置。
func (eg *EconomyGuard) UpdateConfig(cfg EconomyConfig) {
	eg.mu.Lock()
	defer eg.mu.Unlock()
	eg.config = cfg
}

// SetRealmMedian 设置某境界的物品价格中位数。
func (eg *EconomyGuard) SetRealmMedian(realm string, medianPrice float64) {
	eg.mu.Lock()
	defer eg.mu.Unlock()
	eg.realmMedians[realm] = medianPrice
}

// SetRealmMediansBulk 批量设置各境界中位数。
func (eg *EconomyGuard) SetRealmMediansBulk(medians map[string]float64) {
	eg.mu.Lock()
	defer eg.mu.Unlock()
	for realm, median := range medians {
		eg.realmMedians[realm] = median
	}
}

// getRealmMedian 获取境界中位数价格。
func (eg *EconomyGuard) getRealmMedian(realm string) (float64, bool) {
	eg.mu.RLock()
	defer eg.mu.RUnlock()
	median, ok := eg.realmMedians[realm]
	return median, ok
}

// getConfig 安全读取配置。
func (eg *EconomyGuard) getConfig() EconomyConfig {
	eg.mu.RLock()
	defer eg.mu.RUnlock()
	return eg.config
}

// CheckTransaction 校验一笔交易是否异常。
// 返回 (isSuspicious, severity, reason, evidenceMap)。
// 异常但非高严重度时 caller 可以选择仅记录日志而不阻止交易。
func (eg *EconomyGuard) CheckTransaction(ctx context.Context, record EconomyRecord) (bool, string, string, map[string]interface{}) {
	evidence := make(map[string]interface{})
	evidence["player_id"] = record.PlayerID
	evidence["action"] = record.Action
	evidence["item_id"] = record.ItemID
	evidence["amount"] = record.Amount
	evidence["price"] = record.Price
	evidence["realm"] = record.Realm

	cfg := eg.getConfig()

	// 1. 价格偏离检测
	if median, ok := eg.getRealmMedian(record.Realm); ok && median > 0 {
		ratio := record.Price / median
		evidence["median_price"] = median
		evidence["price_ratio"] = ratio

		if ratio > cfg.PriceDeviationThreshold {
			return true, "high",
				fmt.Sprintf("交易价格异常: %s %.0f，超过境界中位数 %.0f 的 %.1f 倍",
					record.Realm, record.Price, median, ratio),
				evidence
		}
	}

	// 2. 快速买卖检测
	if eg.rdb != nil {
		rapidKey := fmt.Sprintf("anticheat:rapid_trade:%d:%d", record.PlayerID, record.ItemID)
		now := time.Now().Unix()

		if record.Action == "buy" {
			// 记录买入时间
			eg.rdb.Set(ctx, rapidKey, now, time.Duration(cfg.RapidTradeWindowSec)*time.Second)
		} else if record.Action == "sell" {
			// 检查是否在买入后不久就卖出
			buyTime, err := eg.rdb.Get(ctx, rapidKey).Int64()
			if err == nil && (now-buyTime) < cfg.RapidTradeWindowSec {
				evidence["buy_timestamp"] = buyTime
				evidence["sell_timestamp"] = now
				evidence["rapid_window_sec"] = cfg.RapidTradeWindowSec
				return true, "medium",
					fmt.Sprintf("快速买卖: %ds 内买入后又卖出 (物品 %d)", now-buyTime, record.ItemID),
					evidence
			}
		}
	}

	// 3. 积累率检测
	accKey := fmt.Sprintf("anticheat:accumulate:%s:%d:%s", record.Realm, record.PlayerID, record.Action)
	if eg.rdb != nil && record.Price > 0 {
		accNow := time.Now().UnixMilli()
		accWindowStart := accNow - cfg.AccumulationWindow.Milliseconds()
		pipe := eg.rdb.Pipeline()

		// 清除窗口外记录
		pipe.ZRemRangeByScore(ctx, accKey, "0", fmt.Sprintf("%d", accWindowStart))
		// 添加当前记录
		pipe.ZAdd(ctx, accKey, redis.Z{Score: float64(accNow), Member: accNow})
		pipe.Expire(ctx, accKey, cfg.AccumulationWindow+time.Hour)
		// 统计窗口内次数
		countCmd := pipe.ZCard(ctx, accKey)

		_, err := pipe.Exec(ctx)
		if err == nil {
			count, _ := countCmd.Result()
			evidence["accumulation_count"] = count
			if int(count) > cfg.AccumulationRateThreshold {
				evidence["accumulation_threshold"] = cfg.AccumulationRateThreshold
				return true, "medium",
					fmt.Sprintf("异常积累: %s 动作在 %s 内执行 %d 次",
						record.Action, cfg.AccumulationWindow, count),
					evidence
			}
		}
	}

	return false, "", "", nil
}

// ============================================================
// LoginPatternCheck 登录模式异常检测
// ============================================================

// LoginPatternConfig 登录模式检测配置。
type LoginPatternConfig struct {
	// 每天异常在线时长阈值（小时）
	MaxDailyOnlineHours float64
	// 持续异常天数阈值
	ConsecutiveDaysThreshold int
	// 重复动作检测：同一动作每秒最大次数
	MaxRepeatedActionsPerSecond int
	// 重复动作检测窗口（秒）
	RepeatedActionWindowSec int
}

// DefaultLoginPatternConfig 默认登录模式检测配置。
func DefaultLoginPatternConfig() LoginPatternConfig {
	return LoginPatternConfig{
		MaxDailyOnlineHours:         16,                             // 每天最多16小时
		ConsecutiveDaysThreshold:    7,                              // 连续7天
		MaxRepeatedActionsPerSecond: 5,                              // 每秒最多5次相同动作
		RepeatedActionWindowSec:     60,                             // 60秒内检测
	}
}

// LoginPatternCheck 检测异常登录模式。
// 检测维度：
//   - 每天在线时间过长（>16h，连续7天+）
//   - 短时间内大量重复相同动作（脚本检测）
type LoginPatternCheck struct {
	rdb    *redis.Client
	config LoginPatternConfig
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewLoginPatternCheck 创建登录模式检测器。
func NewLoginPatternCheck(rdb *redis.Client) *LoginPatternCheck {
	return &LoginPatternCheck{
		rdb:    rdb,
		config: DefaultLoginPatternConfig(),
		logger: slog.Default().With("module", "anticheat.login_pattern"),
	}
}

// UpdateConfig 动态更新配置。
func (lpc *LoginPatternCheck) UpdateConfig(cfg LoginPatternConfig) {
	lpc.mu.Lock()
	defer lpc.mu.Unlock()
	lpc.config = cfg
}

// getConfig 安全读取配置。
func (lpc *LoginPatternCheck) getConfig() LoginPatternConfig {
	lpc.mu.RLock()
	defer lpc.mu.RUnlock()
	return lpc.config
}

// RecordOnlineDuration 记录玩家每日在线时长。
// 每次心跳或关键操作时调用，累计当天的在线时长。
func (lpc *LoginPatternCheck) RecordOnlineDuration(ctx context.Context, playerID uint64, durationSeconds float64) error {
	if lpc.rdb == nil {
		return nil
	}

	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("anticheat:online:%s", today)
	field := fmt.Sprintf("%d", playerID)

	// HINCRBYFLOAT 累加在线时长
	return lpc.rdb.HIncrByFloat(ctx, key, field, durationSeconds).Err()
}

// CheckAbnormalOnline 检查玩家是否存在异常在线模式。
// 返回 (isSuspicious, severity, reason, evidence)。
func (lpc *LoginPatternCheck) CheckAbnormalOnline(ctx context.Context, playerID uint64) (bool, string, string, map[string]interface{}) {
	if lpc.rdb == nil {
		return false, "", "", nil
	}

	cfg := lpc.getConfig()
	evidence := make(map[string]interface{})
	evidence["player_id"] = playerID
	field := fmt.Sprintf("%d", playerID)

	// 检查最近 N 天的在线时长
	var dailyHours []float64
	now := time.Now()
	for i := 0; i < cfg.ConsecutiveDaysThreshold; i++ {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		key := fmt.Sprintf("anticheat:online:%s", day)
		hours, err := lpc.rdb.HGet(ctx, key, field).Float64()
		if err != nil {
			continue // 可能当天无记录
		}
		dailyHours = append(dailyHours, hours)
	}

	evidence["daily_hours"] = dailyHours
	evidence["max_daily_threshold"] = cfg.MaxDailyOnlineHours
	evidence["consecutive_days_threshold"] = cfg.ConsecutiveDaysThreshold

	// 统计持续超时天数
	exceedDays := 0
	for _, hours := range dailyHours {
		if hours >= cfg.MaxDailyOnlineHours {
			exceedDays++
		}
	}

	if exceedDays >= cfg.ConsecutiveDaysThreshold {
		return true, "high",
			fmt.Sprintf("异常在线模式: 连续%d天在线时长超过%.0f小时", exceedDays, cfg.MaxDailyOnlineHours),
			evidence
	}

	// 如果最近几天有超时但还没达到阈值，标记为低严重度
	if exceedDays >= 3 {
		return true, "low",
			fmt.Sprintf("在线时长偏高: %d天超过%.0f小时", exceedDays, cfg.MaxDailyOnlineHours),
			evidence
	}

	return false, "", "", nil
}

// CheckRepeatedActions 检测短时间内大量重复相同动作（脚本/宏检测）。
// 返回 (isSuspicious, severity, reason, evidence)。
func (lpc *LoginPatternCheck) CheckRepeatedActions(ctx context.Context, playerID uint64, action string, msgID uint32) (bool, string, string, map[string]interface{}) {
	if lpc.rdb == nil {
		return false, "", "", nil
	}

	cfg := lpc.getConfig()
	key := fmt.Sprintf("anticheat:repeated:%d", playerID)
	now := time.Now().Unix()
	windowStart := now - int64(cfg.RepeatedActionWindowSec)

	pipe := lpc.rdb.Pipeline()

	// 清理窗口外记录
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// 添加动作记录
	member := fmt.Sprintf("%d:%d:%s", now, msgID, action)
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: member,
	})
	pipe.Expire(ctx, key, time.Duration(cfg.RepeatedActionWindowSec)*time.Second+60)

	// 统计该窗口内总的操作次数
	countCmd := pipe.ZCard(ctx, key)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, "", "", nil
	}

	totalActions, _ := countCmd.Result()
	actionsPerSec := float64(totalActions) / float64(cfg.RepeatedActionWindowSec)

	evidence := map[string]interface{}{
		"player_id":                     playerID,
		"total_actions_in_window":       totalActions,
		"actions_per_second":            actionsPerSec,
		"window_seconds":                cfg.RepeatedActionWindowSec,
		"max_per_second":                cfg.MaxRepeatedActionsPerSecond,
		"latest_action":                 action,
		"latest_msg_id":                 msgID,
	}

	if actionsPerSec > float64(cfg.MaxRepeatedActionsPerSecond) {
		return true, "medium",
			fmt.Sprintf("重复动作异常: 每分钟 %.0f 次操作，超过阈值 %d 次/秒",
				actionsPerSec, cfg.MaxRepeatedActionsPerSecond),
			evidence
	}

	return false, "", "", nil
}

// GetDailyOnline 获取玩家指定日期的在线时长。
func (lpc *LoginPatternCheck) GetDailyOnline(ctx context.Context, playerID uint64, date string) (float64, error) {
	if lpc.rdb == nil {
		return 0, nil
	}
	key := fmt.Sprintf("anticheat:online:%s", date)
	field := fmt.Sprintf("%d", playerID)
	return lpc.rdb.HGet(ctx, key, field).Float64()
}
