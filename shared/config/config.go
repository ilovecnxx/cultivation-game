// Package config 提供游戏配置的加载、解析和热重载功能。
// 支持 JSON、YAML 和环境变量三种来源，环境变量优先级最高。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// Loader 配置加载器，支持从文件加载 JSON/YAML 配置，并通过环境变量覆盖。
// 同时支持通过 fsnotify 监听文件变化实现热重载。
type Loader struct {
	mu       sync.RWMutex
	cfg      map[string]interface{} // 合并后的配置
	filePath string                 // 配置文件路径
	watcher  *fsnotify.Watcher      // 文件监听器（热重载用）
	onChange []func(newCfg map[string]interface{}) // 变更回调列表
	closed   bool
}

// New 创建一个配置加载器，从指定路径加载配置文件。
// filePath 支持 .json 和 .yaml/.yml 格式。
func New(filePath string) (*Loader, error) {
	l := &Loader{
		cfg:      make(map[string]interface{}),
		filePath: filePath,
	}
	if err := l.Load(); err != nil {
		return nil, err
	}
	return l, nil
}

// Load 从文件重新加载配置，合并环境变量覆盖。
func (l *Loader) Load() error {
	raw, err := os.ReadFile(l.filePath)
	if err != nil {
		return fmt.Errorf("config: 读取文件失败 %s: %w", l.filePath, err)
	}

	ext := strings.ToLower(filepath.Ext(l.filePath))
	parsed := make(map[string]interface{})

	switch ext {
	case ".json":
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return fmt.Errorf("config: JSON 解析失败: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			return fmt.Errorf("config: YAML 解析失败: %w", err)
		}
	default:
		return fmt.Errorf("config: 不支持的文件格式 %s (仅支持 .json/.yaml/.yml)", ext)
	}

	// 环境变量覆盖
	applyEnvOverrides(parsed, "")

	l.mu.Lock()
	l.cfg = parsed
	l.mu.Unlock()
	return nil
}

// Get 按 key 获取配置值（支持点号分隔的嵌套键，如 "server.port"）。
// key 不存在时返回 nil。
func (l *Loader) Get(key string) interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return getNested(l.cfg, key)
}

// GetString 获取字符串配置值。若 key 不存在或类型不匹配返回空字符串。
func (l *Loader) GetString(key string) string {
	v := l.Get(key)
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// GetInt 获取整数配置值。
func (l *Loader) GetInt(key string) int {
	v := l.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		n, _ := strconv.Atoi(val)
		return n
	default:
		return 0
	}
}

// GetBool 获取布尔配置值。
func (l *Loader) GetBool(key string) bool {
	v := l.Get(key)
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	default:
		return false
	}
}

// GetFloat 获取浮点数配置值。
func (l *Loader) GetFloat(key string) float64 {
	v := l.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

// GetDuration 获取时间间隔配置值（支持 "5s", "100ms" 等格式）。
func (l *Loader) GetDuration(key string) time.Duration {
	v := l.GetString(key)
	if v == "" {
		return 0
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0
	}
	return d
}

// GetSlice 获取切片配置值。
func (l *Loader) GetSlice(key string) []interface{} {
	v := l.Get(key)
	if v == nil {
		return nil
	}
	if slice, ok := v.([]interface{}); ok {
		return slice
	}
	return nil
}

// GetMap 获取 map 配置值。
func (l *Loader) GetMap(key string) map[string]interface{} {
	v := l.Get(key)
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// All 返回完整配置的只读快照。
func (l *Loader) All() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]interface{}, len(l.cfg))
	for k, v := range l.cfg {
		out[k] = v
	}
	return out
}

// Dump 将当前配置序列化为 JSON 字符串。
func (l *Loader) Dump() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	data, _ := json.MarshalIndent(l.cfg, "", "  ")
	return string(data)
}

// ---- 热重载 ----

// Watch 启动文件监听，当配置文件发生变更时自动重载并通知回调。
// 需要先调用 OnChange 注册回调。返回的 error 表示监听器是否成功启动。
func (l *Loader) Watch() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("config: 创建文件监听器失败: %w", err)
	}
	l.watcher = w

	// 监听配置文件所在目录（fsnotify 需要监听目录而非文件）
	dir := filepath.Dir(l.filePath)
	if err := w.Add(dir); err != nil {
		w.Close()
		return fmt.Errorf("config: 监听目录 %s 失败: %w", dir, err)
	}

	go l.watchLoop()
	return nil
}

// OnChange 注册配置变更回调。支持多次调用注册多个回调。
func (l *Loader) OnChange(fn func(newCfg map[string]interface{})) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onChange = append(l.onChange, fn)
}

// Close 关闭文件监听器，释放资源。
func (l *Loader) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	if l.watcher != nil {
		return l.watcher.Close()
	}
	return nil
}

// watchLoop 内部事件循环，监听文件变化并触发重载。
func (l *Loader) watchLoop() {
	for {
		l.mu.RLock()
		if l.closed {
			l.mu.RUnlock()
			return
		}
		l.mu.RUnlock()

		select {
		case event, ok := <-l.watcher.Events:
			if !ok {
				return
			}
			// 只关心写入和重命名事件（编辑器保存文件时的常见操作）
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
				// 检查是否是我们关注的文件
				if _, err := os.Stat(l.filePath); err != nil {
					continue
				}
				// 短延迟等待文件写入完成
				time.Sleep(50 * time.Millisecond)
				if err := l.Load(); err != nil {
					fmt.Fprintf(os.Stderr, "config: 热重载失败: %v\n", err)
					continue
				}
				// 通知回调
				l.mu.RLock()
				callbacks := make([]func(map[string]interface{}), len(l.onChange))
				copy(callbacks, l.onChange)
				snapshot := make(map[string]interface{})
				for k, v := range l.cfg {
					snapshot[k] = v
				}
				l.mu.RUnlock()

				for _, cb := range callbacks {
					cb(snapshot)
				}
			}
		case err, ok := <-l.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "config: 文件监听错误: %v\n", err)
		}
	}
}

// ---- 辅助函数 ----

// getNested 从嵌套 map 中按点号分隔的 key 取值。
func getNested(m map[string]interface{}, key string) interface{} {
	if key == "" {
		return m
	}
	parts := strings.Split(key, ".")
	current := interface{}(m)
	for _, part := range parts {
		switch node := current.(type) {
		case map[string]interface{}:
			current = node[part]
		default:
			return nil
		}
	}
	return current
}

// applyEnvOverrides 递归遍历配置 map，用同名环境变量覆盖值。
// 环境变量命名规则：配置键的全路径转为大写且用下划线连接，
// 如 "server.port" 对应 "SERVER_PORT"。
func applyEnvOverrides(m map[string]interface{}, prefix string) {
	for key, val := range m {
		envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		if prefix != "" {
			envKey = prefix + "_" + envKey
		}
		if envVal, ok := os.LookupEnv(envKey); ok {
			// 尝试将环境变量字符串转为合适的类型
			m[key] = parseEnvValue(envVal, val)
		}
		// 递归处理子 map
		if subMap, ok := val.(map[string]interface{}); ok {
			applyEnvOverrides(subMap, envKey)
		}
	}
}

// parseEnvValue 将环境变量字符串转换为与原始值类型一致的值。
func parseEnvValue(envVal string, origVal interface{}) interface{} {
	if origVal == nil {
		return envVal
	}
	switch origVal.(type) {
	case bool:
		if b, err := strconv.ParseBool(envVal); err == nil {
			return b
		}
		return origVal
	case float64:
		if f, err := strconv.ParseFloat(envVal, 64); err == nil {
			return f
		}
		return origVal
	case string:
		return envVal
	default:
		// 对于 int、数组等复杂类型，保留原始值的类型推断
		// 这里简单处理：检查是否为整数
		if _, err := strconv.Atoi(envVal); err == nil {
			// 原值如果是 float64，则返回 float64 以保持 JSON 解析一致
			if _, ok := origVal.(float64); ok {
				f, _ := strconv.ParseFloat(envVal, 64)
				return f
			}
		}
		return envVal
	}
}

// UnmarshalKey 将指定配置键的值反序列化到目标结构体（通过 JSON 标签映射）。
func (l *Loader) UnmarshalKey(key string, out interface{}) error {
	val := l.Get(key)
	if val == nil {
		return fmt.Errorf("config: 键 %q 不存在", key)
	}
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("config: 序列化中间值失败: %w", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("config: 反序列化到目标类型失败: %w", err)
	}
	return nil
}

// MustLoad 加载配置，失败则 panic（适用于初始化阶段）。
func MustLoad(filePath string) *Loader {
	l, err := New(filePath)
	if err != nil {
		panic(fmt.Sprintf("config: 加载配置异常: %v", err))
	}
	return l
}

// Ensure Loader implements a simple string representation.
func (l *Loader) String() string {
	return fmt.Sprintf("Loader{file=%s, keys=%d}", l.filePath, len(l.cfg))
}

