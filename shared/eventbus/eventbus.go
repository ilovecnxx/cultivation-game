// Package eventbus 提供游戏内事件总线，支持同步/异步事件分发、通配符订阅和超时控制。
package eventbus

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// Event 封装一个事件的数据。
type Event struct {
	Topic   string      // 事件主题
	Data    interface{} // 事件载荷
	Context context.Context // 上下文，支持超时和取消
}

// Handler 事件处理函数。
// 返回 true 表示继续传播给下一个处理器；返回 false 则停止。
type Handler func(e *Event) (propagate bool)

// Middleware 事件中间件，可以在事件到达处理器前后执行逻辑。
type Middleware func(next Handler) Handler

// Bus 是事件总线的核心结构，线程安全。
type Bus struct {
	mu         sync.RWMutex
	handlers   map[string][]Handler // 主题 -> 处理器列表
	middleware []Middleware         // 全局中间件链
	logger     *log.Logger
}

// New 创建一个新的事件总线。
// logger 可为 nil，此时不记录日志。
func New(logger *log.Logger) *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
		logger:   logger,
	}
}

// Use 添加全局中间件。中间件按添加顺序组成链。
func (b *Bus) Use(mw Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middleware = append(b.middleware, mw)
}

// Subscribe 订阅指定主题。支持通配符模式：
//   - "player.*"      匹配 "player.login"、"player.levelup"
//   - "combat.**"     匹配 "combat.start"、"combat.round.end"
//   - "*"             匹配所有
func (b *Bus) Subscribe(topic string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
}

// PublishSync 同步发布事件：阻塞等待所有处理器执行完毕。
// 如果任一处理器返回 false，则停止后续处理器。
func (b *Bus) PublishSync(topic string, data interface{}) {
	b.publish(context.Background(), topic, data, false)
}

// PublishAsync 异步发布事件：在新 goroutine 中执行，不阻塞调用者。
func (b *Bus) PublishAsync(topic string, data interface{}) {
	b.publish(context.Background(), topic, data, true)
}

// PublishWithContext 带上下文发布事件。可通过 context.WithTimeout 实现超时控制。
// 仅在同步模式下超时有效；异步模式在后台 goroutine 中执行，忽略超时。
func (b *Bus) PublishWithContext(ctx context.Context, topic string, data interface{}) {
	b.publish(ctx, topic, data, false)
}

// publish 内部发布逻辑。
func (b *Bus) publish(ctx context.Context, topic string, data interface{}, async bool) {
	matched := b.collectHandlers(topic)
	if len(matched) == 0 {
		return
	}

	// 应用中间件链
	chain := b.buildChain(matched)

	e := &Event{
		Topic:   topic,
		Data:    data,
		Context: ctx,
	}

	if async {
		go b.executeAsync(chain, e)
		return
	}
	b.executeSync(ctx, chain, e)
}

// collectHandlers 收集匹配主题的所有处理器（精确匹配 + 通配符）。
func (b *Bus) collectHandlers(topic string) []Handler {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []Handler

	// 精确匹配
	result = append(result, b.handlers[topic]...)

	// 通配符匹配
	for pattern, handlers := range b.handlers {
		if pattern != topic && match(pattern, topic) {
			result = append(result, handlers...)
		}
	}

	return result
}

// buildChain 将中间件依次包裹到处理器链上。
func (b *Bus) buildChain(handlers []Handler) []Handler {
	b.mu.RLock()
	mws := make([]Middleware, len(b.middleware))
	copy(mws, b.middleware)
	b.mu.RUnlock()

	// 如果没有中间件，直接返回原始处理器列表
	if len(mws) == 0 {
		return handlers
	}

	// 为每个处理器应用中间件链
	wrapped := make([]Handler, len(handlers))
	for i, h := range handlers {
		chain := h
		// 逆序应用，使第一个中间件最先执行
		for j := len(mws) - 1; j >= 0; j-- {
			chain = mws[j](chain)
		}
		wrapped[i] = chain
	}
	return wrapped
}

// executeSync 同步执行处理器链。
func (b *Bus) executeSync(ctx context.Context, handlers []Handler, e *Event) {
	for _, h := range handlers {
		// 检查上下文是否已超时或取消
		if err := ctx.Err(); err != nil {
			if b.logger != nil {
				b.logger.Printf("[eventbus] 事件 %q 执行被中止: %v", e.Topic, err)
			}
			return
		}
		if !h(e) {
			return // 停止传播
		}
	}
}

// executeAsync 在后台执行处理器链。
func (b *Bus) executeAsync(handlers []Handler, e *Event) {
	for _, h := range handlers {
		if !h(e) {
			return
		}
	}
}

// Reset 清空所有订阅和中间件。
func (b *Bus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = make(map[string][]Handler)
	b.middleware = nil
}

// HandlerCount 返回指定主题的处理器数量（包含通配符匹配）。
func (b *Bus) HandlerCount(topic string) int {
	return len(b.collectHandlers(topic))
}

// match 判断 pattern 是否匹配 topic。
// 支持的语法：
//   - "*"  匹配单层，如 "player.*" 匹配 "player.login"
//   - "**" 匹配多层，如 "combat.**" 匹配 "combat.round.end"
//   - 普通字符串做精确匹配（外部已做精确匹配，这里只处理通配）
func match(pattern, topic string) bool {
	// "*" 匹配所有
	if pattern == "*" {
		return true
	}
	// "**" 也匹配所有
	if pattern == "**" {
		return true
	}

	pp := strings.Split(pattern, ".")
	tp := strings.Split(topic, ".")

	pi, ti := 0, 0
	for pi < len(pp) && ti < len(tp) {
		if pp[pi] == "**" {
			// "**" 匹配剩余所有段
			return true
		}
		if pp[pi] == "*" {
			// "*" 匹配当前段，继续下一段
			pi++
			ti++
			continue
		}
		if pp[pi] != tp[ti] {
			return false
		}
		pi++
		ti++
	}

	// 剩余的模式段如果全是 "**" 也算匹配
	for pi < len(pp) {
		if pp[pi] != "**" {
			return false
		}
		pi++
	}
	return pi == len(pp) && ti == len(tp)
}

// ---- 辅助函数 ----

// Timeout 返回一个中间件，为每个事件设置超时。
// 超时后上下文取消，后续处理器将收到 context.DeadlineExceeded 错误。
func Timeout(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(e *Event) bool {
			ctx, cancel := context.WithTimeout(e.Context, timeout)
			defer cancel()
			e.Context = ctx
			return next(e)
		}
	}
}

// Logging 返回一个日志中间件，记录事件分发情况。
func Logging(logger *log.Logger) Middleware {
	return func(next Handler) Handler {
		return func(e *Event) bool {
			start := time.Now()
			result := next(e)
			logger.Printf("[eventbus] topic=%s duration=%v", e.Topic, time.Since(start))
			return result
		}
	}
}

// Recover 返回一个恢复中间件，捕获处理器中的 panic 并记录，防止崩溃。
func Recover(logger *log.Logger) Middleware {
	return func(next Handler) Handler {
		return func(e *Event) (propagate bool) {
			defer func() {
				if r := recover(); r != nil {
					logger.Printf("[eventbus] panic 恢复 topic=%s recover=%v", e.Topic, r)
					propagate = true // panic 后继续传播
				}
			}()
			return next(e)
		}
	}
}

// Ensure type Bus implements a simple stringer for debugging.
func (b *Bus) String() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return fmt.Sprintf("EventBus{topics=%d, middleware=%d}", len(b.handlers), len(b.middleware))
}
