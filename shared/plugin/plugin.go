// Package plugin 定义游戏插件系统接口，所有业务模块通过插件机制注册到主进程中。
package plugin

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

// 插件系统预定义错误。
var (
	ErrHandlerNotFound = errors.New("plugin: 未找到消息处理器")
)

// ErrPluginExists 插件名称冲突。
func ErrPluginExists(name string) error {
	return fmt.Errorf("plugin: 插件 %q 已存在", name)
}

// ErrPluginInitFailed 插件初始化失败。
func ErrPluginInitFailed(name string, err error) error {
	return fmt.Errorf("plugin: 插件 %q 初始化失败: %w", name, err)
}

// ErrPluginStartFailed 插件启动失败。
func ErrPluginStartFailed(name string, err error) error {
	return fmt.Errorf("plugin: 插件 %q 启动失败: %w", name, err)
}

// GamePlugin 是游戏插件必须实现的接口。
type GamePlugin interface {
	Name() string            // 插件名称，全局唯一
	Version() string         // 插件版本号
	OnInit(ctx GameContext) error    // 初始化，加载配置、建立连接等
	OnStart() error          // 启动，开始处理消息
	OnStop() error           // 停止，释放资源
	RegisterHandlers(router *Router) // 注册消息处理器到路由
	RegisterEvents(bus *EventBus)    // 注册事件监听器到事件总线
}

// GameContext 为插件提供游戏运行时的上下文访问能力。
type GameContext interface {
	GetConfig(key string) string // 获取配置项
	GetLogger() *log.Logger      // 获取日志器
}

// HandlerFunc 消息处理函数签名。
type HandlerFunc func(msgID uint32, seq uint64, payload []byte) ([]byte, error)

// Router 消息路由注册表，维护消息ID到处理函数的映射。
// 线程安全，支持运行时注册。
type Router struct {
	mu       sync.RWMutex
	handlers map[uint32]HandlerFunc
}

// NewRouter 创建一个新的路由表。
func NewRouter() *Router {
	return &Router{handlers: make(map[uint32]HandlerFunc)}
}

// Handle 注册指定消息ID的处理函数。如果已存在则覆盖。
func (r *Router) Handle(msgID uint32, fn HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[msgID] = fn
}

// Dispatch 查找并调用消息ID对应的处理函数。返回 (nil, false) 表示未找到处理器。
func (r *Router) Dispatch(msgID uint32, seq uint64, payload []byte) ([]byte, error) {
	r.mu.RLock()
	fn, ok := r.handlers[msgID]
	r.mu.RUnlock()
	if !ok {
		return nil, ErrHandlerNotFound
	}
	return fn(msgID, seq, payload)
}

// PluginManager 管理所有已注册插件的生命周期。
type PluginManager struct {
	mu        sync.RWMutex
	plugins   map[string]GamePlugin // 插件名 -> 插件实例
	loadOrder []string              // 按加载顺序记录的插件名
	ctx       GameContext           // 共享的运行时上下文
}

// NewPluginManager 创建插件管理器，需要传入一个上下文实现。
func NewPluginManager(ctx GameContext) *PluginManager {
	return &PluginManager{
		plugins: make(map[string]GamePlugin),
		ctx:     ctx,
	}
}

// Register 注册一个插件。如果名称重复则返回错误。
func (pm *PluginManager) Register(p GamePlugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	name := p.Name()
	if _, exists := pm.plugins[name]; exists {
		return ErrPluginExists(name)
	}
	pm.plugins[name] = p
	pm.loadOrder = append(pm.loadOrder, name)
	return nil
}

// InitAll 按注册顺序依次初始化所有插件。
func (pm *PluginManager) InitAll() error {
	pm.mu.RLock()
	order := pm.loadOrder
	pm.mu.RUnlock()

	for _, name := range order {
		pm.mu.RLock()
		p := pm.plugins[name]
		pm.mu.RUnlock()

		if err := p.OnInit(pm.ctx); err != nil {
			return ErrPluginInitFailed(name, err)
		}
	}
	return nil
}

// StartAll 按注册顺序启动所有插件。
func (pm *PluginManager) StartAll() error {
	pm.mu.RLock()
	order := pm.loadOrder
	pm.mu.RUnlock()

	for _, name := range order {
		pm.mu.RLock()
		p := pm.plugins[name]
		pm.mu.RUnlock()

		if err := p.OnStart(); err != nil {
			return ErrPluginStartFailed(name, err)
		}
	}
	return nil
}

// StopAll 逆序停止所有插件。
func (pm *PluginManager) StopAll() {
	pm.mu.RLock()
	order := make([]string, len(pm.loadOrder))
	copy(order, pm.loadOrder)
	pm.mu.RUnlock()

	// 逆序停止
	for i := len(order) - 1; i >= 0; i-- {
		pm.mu.RLock()
		p := pm.plugins[order[i]]
		pm.mu.RUnlock()

		if err := p.OnStop(); err != nil {
			pm.ctx.GetLogger().Printf("停止插件 %s 失败: %v", order[i], err)
		}
	}
}

// GetPlugin 按名称获取已注册的插件实例。
func (pm *PluginManager) GetPlugin(name string) (GamePlugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	p, ok := pm.plugins[name]
	return p, ok
}

// EventBus 事件总线，插件通过它注册和分发事件。
type EventBus struct {
	mu     sync.RWMutex
	subs   map[string][]EventHandler // 主题 -> 处理器列表
}

// EventHandler 事件处理函数。第一个返回值决定是否继续传播给下一个处理器。
type EventHandler func(topic string, data interface{}) (propagate bool)

// NewEventBus 创建事件总线。
func NewEventBus() *EventBus {
	return &EventBus{subs: make(map[string][]EventHandler)}
}

// Subscribe 订阅指定主题。支持通配符（如 "player.*"）。
func (eb *EventBus) Subscribe(topic string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subs[topic] = append(eb.subs[topic], handler)
}

// Publish 向指定主题发布事件，同步调用所有匹配的处理器。
// 任何处理器返回 propagate=false 则停止传播。
func (eb *EventBus) Publish(topic string, data interface{}) {
	eb.mu.RLock()
	// 先找精确匹配
	handlers := append([]EventHandler{}, eb.subs[topic]...)
	// 再找通配符匹配
	for pattern, hlist := range eb.subs {
		if matchWildcard(pattern, topic) {
			handlers = append(handlers, hlist...)
		}
	}
	eb.mu.RUnlock()

	for _, h := range handlers {
		if !h(topic, data) {
			break
		}
	}
}

// matchWildcard 简单的通配符匹配，仅支持末尾星号（如 "player.*" 匹配 "player.login"）。
func matchWildcard(pattern, topic string) bool {
	if pattern == "*" {
		return true
	}
	n := len(pattern)
	if n > 0 && pattern[n-1] == '*' {
		prefix := pattern[:n-1]
		return len(topic) >= len(prefix) && topic[:len(prefix)] == prefix
	}
	return pattern == topic
}
