// Package analytics 游戏分析/埋点系统基础设施。
//
// 网关位于所有流量的入口，是最适合采集分析事件的位置。
// 使用环形缓冲区暂存事件，定时或定量刷新到 MongoDB。
// 刷新策略：每 60 秒或缓冲区满 1000 条事件时自动刷新。
package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ============================================================
// 事件类型常量
//
// 所有预定义的标准事件类型，命名规则：模块_动词。
// ============================================================

const (
	// ---- 认证 ----
	EventPlayerLogin    = "player_login"
	EventPlayerLogout   = "player_logout"
	EventPlayerRegister = "player_register"

	// ---- 修炼 ----
	EventCultivationStart    = "cultivation_start"
	EventCultivationComplete = "cultivation_complete"
	EventBreakthroughAttempt = "breakthrough_attempt"
	EventBreakthroughSuccess = "breakthrough_success"
	EventBreakthroughFail    = "breakthrough_fail"

	// ---- 战斗 ----
	EventCombatStart = "combat_start"
	EventCombatWin   = "combat_win"
	EventCombatLose  = "combat_lose"

	// ---- PVP ----
	EventPVPArenaEnter  = "pvp_arena_enter"
	EventPVPArenaResult = "pvp_arena_result"
	EventPVPArenaRank   = "pvp_arena_rank"
	EventPVPChallenge   = "pvp_challenge"

	// ---- 物品 ----
	EventItemObtain = "item_obtain"
	EventItemUse    = "item_use"
	EventItemSell   = "item_sell"

	// ---- 装备 ----
	EventEquipmentEnhance  = "equipment_enhance"
	EventEquipmentRefine   = "equipment_refine"
	EventEquipmentUpgrade  = "equipment_upgrade"
	EventEquipmentSetBonus = "equipment_set_bonus"

	// ---- 任务 ----
	EventQuestStart   = "quest_start"
	EventQuestComplete = "quest_complete"
	EventQuestAbandon  = "quest_abandon"

	// ---- 副本 ----
	EventDungeonEnter  = "dungeon_enter"
	EventDungeonComplete = "dungeon_complete"
	EventDungeonFail   = "dungeon_fail"

	// ---- 交易 ----
	EventTradeCreate     = "trade_create"
	EventTradeComplete   = "trade_complete"
	EventTradeCancel     = "trade_cancel"
	EventTradeAuction    = "trade_auction"
	EventTradeBid        = "trade_bid"

	// ---- 商店 ----
	EventShopPurchase = "shop_purchase"
	EventShopRefresh  = "shop_refresh"

	// ---- 充值 ----
	EventRechargeOrder   = "recharge_order"
	EventRechargeSuccess = "recharge_success"
	EventRechargeRefund  = "recharge_refund"

	// ---- 社交 ----
	EventSocialFriendAdd   = "social_friend_add"
	EventSocialChatMessage = "social_chat_message"
	EventSocialGuildJoin   = "social_guild_join"
	EventSocialGuildLeave  = "social_guild_leave"

	// ---- 系统 ----
	EventSystemLevelUp = "system_level_up"
	EventSystemRealmUp = "system_realm_up"
	EventSystemAchieve = "system_achievement"
	EventSystemRebirth = "system_rebirth"
)

// ============================================================
// 刷新模式
// ============================================================

// FlushMode 刷新模式。
type FlushMode int

const (
	// FlushModeBatch 批量刷新（默认）。
	FlushModeBatch FlushMode = iota
	// FlushModeImmediate 每次事件立即写入（仅调试使用）。
	FlushModeImmediate
)

// ============================================================
// AnalyticsEvent 分析事件
// ============================================================

// AnalyticsEvent 分析事件结构体。
type AnalyticsEvent struct {
	// EventType 事件类型，使用 Event* 常量。
	EventType string `bson:"event_type" json:"event_type"`
	// PlayerID 玩家 ID（0 表示系统事件）。
	PlayerID uint64 `bson:"player_id" json:"player_id"`
	// Realm 玩家当前境界名称（可为空）。
	Realm string `bson:"realm,omitempty" json:"realm,omitempty"`
	// PlayerLevel 玩家等级（0 表示未知）。
	PlayerLevel int `bson:"player_level,omitempty" json:"player_level,omitempty"`
	// Timestamp 事件发生时间（服务端时间）。
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	// EventDate 事件日期字符串 YYYY-MM-DD，用于 MongoDB 分区和 TTL。
	EventDate string `bson:"event_date" json:"event_date"`
	// Properties 自定义属性映射，存放与具体事件相关的附加数据。
	Properties map[string]interface{} `bson:"properties,omitempty" json:"properties,omitempty"`
	// SessionID 玩家会话 ID（用于关联登录/登出）。
	SessionID string `bson:"session_id,omitempty" json:"session_id,omitempty"`
}

// ============================================================
// RingBuffer 线程安全环形缓冲区
// ============================================================

// RingBuffer 线程安全环形缓冲区，用于暂存分析事件。
// 当元素个数达到容量或刷新定时器到期时触发刷新。
type RingBuffer struct {
	buf    []*AnalyticsEvent
	cap    int
	head   int
	tail   int
	count  int
	mu     sync.Mutex
	notify chan struct{} // 非阻塞通知通道
}

// NewRingBuffer 创建指定容量的环形缓冲区。
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buf:    make([]*AnalyticsEvent, capacity),
		cap:    capacity,
		notify: make(chan struct{}, 1),
	}
}

// Push 向缓冲区尾部追加一个事件。
// 返回是否触发刷新信号（缓冲区满）。
func (rb *RingBuffer) Push(event *AnalyticsEvent) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == rb.cap {
		// 缓冲区已满，覆盖最旧的事件并移动 head
		slog.Warn("analytics ring buffer full, overwriting oldest event",
			"capacity", rb.cap,
			"event_type", event.EventType,
			"player_id", event.PlayerID,
		)
		rb.buf[rb.head] = event
		rb.head = (rb.head + 1) % rb.cap
		rb.tail = (rb.tail + 1) % rb.cap
		// 通知消费者刷新
		select {
		case rb.notify <- struct{}{}:
		default:
		}
		return true
	}

	rb.buf[rb.tail] = event
	rb.tail = (rb.tail + 1) % rb.cap
	rb.count++

	full := rb.count == rb.cap
	if full {
		select {
		case rb.notify <- struct{}{}:
		default:
		}
	}
	return full
}

// PopAll 弹出缓冲区中所有事件并清空缓冲区。
func (rb *RingBuffer) PopAll() []*AnalyticsEvent {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return nil
	}

	result := make([]*AnalyticsEvent, 0, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.head + i) % rb.cap
		result = append(result, rb.buf[idx])
		rb.buf[idx] = nil // 释放引用
	}
	rb.head = 0
	rb.tail = 0
	rb.count = 0
	return result
}

// Len 返回缓冲区当前元素个数。
func (rb *RingBuffer) Len() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}

// Cap 返回缓冲区容量。
func (rb *RingBuffer) Cap() int {
	return rb.cap
}

// ============================================================
// Analytics 分析引擎
// ============================================================

// Analytics 分析引擎，管理事件采集、缓冲和 MongoDB 刷新。
type Analytics struct {
	buffer    *RingBuffer
	collection *mongo.Collection
	mode      FlushMode
	playerID  uint64 // 预留：当前玩家 ID（网关上下文）

	flushInterval time.Duration
	flushBatch    int

	stopCh   chan struct{}
	closed   bool
	mu       sync.RWMutex
	logger   *slog.Logger

	// 统计
	eventsTracked    int64
	eventsFlushed    int64
	flushCount       int64
	lastFlushTime    time.Time
	muStats          sync.Mutex
}

// Options 分析引擎配置项。
type Options struct {
	// MongoDB 连接 URI（为空时不启动 MongoDB 刷新）。
	MongoURI string
	// MongoDB 数据库名。
	MongoDatabase string
	// MongoDB 集合名。
	MongoCollection string
	// 缓冲区容量。
	BufferCapacity int
	// 刷新间隔。
	FlushInterval time.Duration
	// 批量刷新阈值。
	FlushBatchSize int
	// 刷新模式。
	Mode FlushMode
}

// DefaultOptions 返回默认配置。
func DefaultOptions() Options {
	return Options{
		MongoURI:        "",
		MongoDatabase:   "cultivation_game",
		MongoCollection: "analytics_events",
		BufferCapacity:  1000,
		FlushInterval:   60 * time.Second,
		FlushBatchSize:  1000,
		Mode:            FlushModeBatch,
	}
}

// NewAnalytics 创建分析引擎。
// 如果提供了 MongoDB URI，会尝试连接；连接失败仅记录警告，不会阻止引擎运行。
func NewAnalytics(opts Options) *Analytics {
	a := &Analytics{
		buffer:        NewRingBuffer(opts.BufferCapacity),
		mode:          opts.Mode,
		flushInterval: opts.FlushInterval,
		flushBatch:    opts.FlushBatchSize,
		stopCh:        make(chan struct{}),
		lastFlushTime: time.Now(),
		logger:        slog.Default().With("module", "analytics"),
	}

	// 如果提供了 MongoDB URI，尝试连接
	if opts.MongoURI != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(opts.MongoURI))
		if err != nil {
			a.logger.Warn("analytics MongoDB 连接失败，将使用本地缓冲模式",
				"error", err,
			)
		} else {
			// 验证连接
			if err := client.Ping(ctx, nil); err != nil {
				a.logger.Warn("analytics MongoDB Ping 失败，将使用本地缓冲模式",
					"error", err,
				)
			} else {
				a.logger.Info("analytics MongoDB 连接成功",
					"database", opts.MongoDatabase,
					"collection", opts.MongoCollection,
				)
				a.collection = client.Database(opts.MongoDatabase).Collection(opts.MongoCollection)

				// 在后台 goroutine 中注册集合关闭
				go func() {
					<-a.stopCh
					if err := client.Disconnect(context.Background()); err != nil {
						a.logger.Warn("analytics MongoDB 断开连接失败", "error", err)
					}
				}()
			}
		}
	}

	// 启动自动刷新协程
	if a.mode == FlushModeBatch {
		go a.flushLoop()
	}

	a.logger.Info("analytics engine initialized",
		"buffer_capacity", opts.BufferCapacity,
		"flush_interval", a.flushInterval,
		"flush_batch", a.flushBatch,
		"mode", func() string {
			if a.mode == FlushModeImmediate {
				return "immediate"
			}
			return "batch"
		}(),
	)

	return a
}

// Track 采集一个分析事件。
// 如果启用了消息推送模式，会同时通过 NATS 发布事件。
func (a *Analytics) Track(event *AnalyticsEvent) {
	if event == nil {
		return
	}

	// 填充默认值
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.EventDate == "" {
		event.EventDate = event.Timestamp.Format("2006-01-02")
	}

	a.mu.RLock()
	closed := a.closed
	a.mu.RUnlock()
	if closed {
		return
	}

	a.muStats.Lock()
	a.eventsTracked++
	a.muStats.Unlock()

	if a.mode == FlushModeImmediate && a.collection != nil {
		// 立即模式：直接写入 MongoDB
		a.flushSingle(event)
		return
	}

	// 批量模式：推入环形缓冲区
	full := a.buffer.Push(event)
	if full {
		// 缓冲区满，尝试异步刷新（不阻塞调用方）
		go a.Flush()
	}

	a.logger.Debug("analytics event tracked",
		"event_type", event.EventType,
		"player_id", event.PlayerID,
		"timestamp", event.Timestamp,
	)
}

// EventDateString 返回当前日期的 YYYY-MM-DD 字符串。
func EventDateString(t time.Time) string {
	return t.Format("2006-01-02")
}

// ============================================================
// 便捷事件采集方法
// ============================================================

// TrackLogin 记录玩家登录事件。
func (a *Analytics) TrackLogin(playerID uint64, realm string, level int, sessionID string, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType:  EventPlayerLogin,
		PlayerID:   playerID,
		Realm:      realm,
		PlayerLevel: level,
		Timestamp:  time.Now(),
		EventDate:  EventDateString(time.Now()),
		SessionID:  sessionID,
		Properties: props,
	})
}

// TrackLogout 记录玩家登出事件。
func (a *Analytics) TrackLogout(playerID uint64, realm string, level int, sessionID string, durationSeconds int, props map[string]interface{}) {
	if props == nil {
		props = make(map[string]interface{})
	}
	props["session_duration_seconds"] = durationSeconds

	a.Track(&AnalyticsEvent{
		EventType:  EventPlayerLogout,
		PlayerID:   playerID,
		Realm:      realm,
		PlayerLevel: level,
		Timestamp:  time.Now(),
		EventDate:  EventDateString(time.Now()),
		SessionID:  sessionID,
		Properties: props,
	})
}

// TrackRegister 记录玩家注册事件。
func (a *Analytics) TrackRegister(playerID uint64, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType: EventPlayerRegister,
		PlayerID:  playerID,
		Timestamp: time.Now(),
		EventDate: EventDateString(time.Now()),
		Properties: props,
	})
}

// TrackCultivation 记录修炼相关事件。
func (a *Analytics) TrackCultivation(eventType string, playerID uint64, realm string, level int, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType:   eventType,
		PlayerID:    playerID,
		Realm:       realm,
		PlayerLevel: level,
		Timestamp:   time.Now(),
		EventDate:   EventDateString(time.Now()),
		Properties:  props,
	})
}

// TrackCombat 记录战斗相关事件。
func (a *Analytics) TrackCombat(eventType string, playerID uint64, realm string, level int, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType:   eventType,
		PlayerID:    playerID,
		Realm:       realm,
		PlayerLevel: level,
		Timestamp:   time.Now(),
		EventDate:   EventDateString(time.Now()),
		Properties:  props,
	})
}

// TrackItem 记录物品相关事件。
func (a *Analytics) TrackItem(eventType string, playerID uint64, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType:  eventType,
		PlayerID:   playerID,
		Timestamp:  time.Now(),
		EventDate:  EventDateString(time.Now()),
		Properties: props,
	})
}

// TrackRecharge 记录充值事件。
func (a *Analytics) TrackRecharge(playerID uint64, realm string, level int, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType:   EventRechargeSuccess,
		PlayerID:    playerID,
		Realm:       realm,
		PlayerLevel: level,
		Timestamp:   time.Now(),
		EventDate:   EventDateString(time.Now()),
		Properties:  props,
	})
}

// TrackQuest 记录任务相关事件。
func (a *Analytics) TrackQuest(eventType string, playerID uint64, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType:  eventType,
		PlayerID:   playerID,
		Timestamp:  time.Now(),
		EventDate:  EventDateString(time.Now()),
		Properties: props,
	})
}

// TrackDungeon 记录副本相关事件。
func (a *Analytics) TrackDungeon(eventType string, playerID uint64, realm string, level int, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType:   eventType,
		PlayerID:    playerID,
		Realm:       realm,
		PlayerLevel: level,
		Timestamp:   time.Now(),
		EventDate:   EventDateString(time.Now()),
		Properties:  props,
	})
}

// TrackTrade 记录交易相关事件。
func (a *Analytics) TrackTrade(eventType string, playerID uint64, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType:  eventType,
		PlayerID:   playerID,
		Timestamp:  time.Now(),
		EventDate:  EventDateString(time.Now()),
		Properties: props,
	})
}

// TrackSocial 记录社交相关事件。
func (a *Analytics) TrackSocial(eventType string, playerID uint64, props map[string]interface{}) {
	a.Track(&AnalyticsEvent{
		EventType:  eventType,
		PlayerID:   playerID,
		Timestamp:  time.Now(),
		EventDate:  EventDateString(time.Now()),
		Properties: props,
	})
}

// ============================================================
// 刷新机制
// ============================================================

// flushLoop 自动刷新循环。
func (a *Analytics) flushLoop() {
	ticker := time.NewTicker(a.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 定时刷新
			if a.buffer.Len() > 0 {
				if err := a.Flush(); err != nil {
					a.logger.Warn("analytics timer flush failed", "error", err)
				}
			}
		case <-a.buffer.notify:
			// 缓冲区满通知
			// 小延迟等待更多事件聚合
			time.Sleep(100 * time.Millisecond)
			if a.buffer.Len() >= a.flushBatch {
				if err := a.Flush(); err != nil {
					a.logger.Warn("analytics buffer-full flush failed", "error", err)
				}
			}
		case <-a.stopCh:
			// 退出前刷新剩余事件
			if a.buffer.Len() > 0 {
				if err := a.Flush(); err != nil {
					a.logger.Warn("analytics final flush failed", "error", err)
				}
			}
			return
		}
	}
}

// Flush 将缓冲区中的事件刷新到 MongoDB。
func (a *Analytics) Flush() error {
	events := a.buffer.PopAll()
	if len(events) == 0 {
		return nil
	}

	a.muStats.Lock()
	a.eventsFlushed += int64(len(events))
	a.flushCount++
	a.lastFlushTime = time.Now()
	a.muStats.Unlock()

	if a.collection == nil {
		a.logger.Debug("analytics MongoDB not connected, discarding events",
			"count", len(events),
		)
		return nil
	}

	// 批量写入 MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	docs := make([]interface{}, len(events))
	for i, e := range events {
		docs[i] = e
	}

	result, err := a.collection.InsertMany(ctx, docs)
	if err != nil {
		a.logger.Warn("analytics MongoDB insert failed",
			"error", err,
			"count", len(events),
		)
		return fmt.Errorf("analytics flush to mongodb: %w", err)
	}

	a.logger.Info("analytics events flushed to MongoDB",
		"count", len(events),
		"inserted", len(result.InsertedIDs),
	)
	return nil
}

// flushSingle 立即写入单个事件（即时模式）。
func (a *Analytics) flushSingle(event *AnalyticsEvent) {
	if a.collection == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := a.collection.InsertOne(ctx, event); err != nil {
		a.logger.Warn("analytics immediate insert failed",
			"error", err,
			"event_type", event.EventType,
		)
	}
}

// ============================================================
// 统计与状态
// ============================================================

// Stats 返回分析引擎的统计信息。
type Stats struct {
	EventsTracked int64     `json:"events_tracked"`
	EventsFlushed int64     `json:"events_flushed"`
	FlushCount    int64     `json:"flush_count"`
	BufferLen     int       `json:"buffer_len"`
	BufferCap     int       `json:"buffer_cap"`
	LastFlushTime time.Time `json:"last_flush_time"`
	MongoConnected bool    `json:"mongo_connected"`
}

// GetStats 返回当前统计信息。
func (a *Analytics) GetStats() Stats {
	a.muStats.Lock()
	tracked := a.eventsTracked
	flushed := a.eventsFlushed
	flushCount := a.flushCount
	lastFlush := a.lastFlushTime
	a.muStats.Unlock()

	return Stats{
		EventsTracked:  tracked,
		EventsFlushed:  flushed,
		FlushCount:     flushCount,
		BufferLen:      a.buffer.Len(),
		BufferCap:      a.buffer.Cap(),
		LastFlushTime:  lastFlush,
		MongoConnected: a.collection != nil,
	}
}

// IsMongoConnected 返回 MongoDB 是否连接。
func (a *Analytics) IsMongoConnected() bool {
	return a.collection != nil
}

// ============================================================
// 生命周期
// ============================================================

// Close 关闭分析引擎，刷新剩余事件并释放资源。
func (a *Analytics) Close() {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return
	}
	a.closed = true
	a.mu.Unlock()

	a.logger.Info("analytics engine closing")

	// 停止刷新循环
	close(a.stopCh)

	// 执行最后一次刷新（flushLoop 中已处理，但以防协程未启动）
	if a.buffer.Len() > 0 {
		if err := a.Flush(); err != nil {
			a.logger.Warn("analytics close flush failed", "error", err)
		}
	}

	stats := a.GetStats()
	a.logger.Info("analytics engine stopped",
		"events_tracked", stats.EventsTracked,
		"events_flushed", stats.EventsFlushed,
		"flush_count", stats.FlushCount,
	)
}
