// Package client 提供可靠的跨服务 HTTP 通信客户端。
//
// 特性：
//   - 自动重试（指数退避）
//   - 超时控制
//   - 断路器模式
//   - 请求追踪
//   - 优雅降级
//   - 与服务发现集成
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

// CircuitState 断路器状态
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // 正常状态
	CircuitHalfOpen                      // 半开状态（尝试恢复）
	CircuitOpen                          // 断开状态（快速失败）
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitHalfOpen:
		return "half-open"
	case CircuitOpen:
		return "open"
	default:
		return "unknown"
	}
}

// Config 客户端配置
type Config struct {
	// 超时设置
	RequestTimeout time.Duration // 单次请求超时（默认 5s）
	RetryMaxWait   time.Duration // 最大重试等待时间（默认 30s）

	// 重试设置
	MaxRetries      int           // 最大重试次数（默认 3）
	RetryBackoffMin time.Duration // 退避起始时间（默认 100ms）
	RetryBackoffMax time.Duration // 退避最大时间（默认 5s）

	// 断路器设置
	CircuitThreshold   int           // 连续失败多少次后断开（默认 5）
	CircuitTimeout     time.Duration // 断开后多久尝试恢复（默认 30s）
	HalfOpenMaxReqs    int           // 半开状态下允许的试探请求数（默认 3）

	// 服务发现（可选）
	Discoverer ServiceDiscoverer
}

// ServiceDiscoverer 服务发现接口
type ServiceDiscoverer interface {
	PickAddr(ctx context.Context, name string) (string, error)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		RequestTimeout:   5 * time.Second,
		RetryMaxWait:     30 * time.Second,
		MaxRetries:       3,
		RetryBackoffMin:  100 * time.Millisecond,
		RetryBackoffMax:  5 * time.Second,
		CircuitThreshold: 5,
		CircuitTimeout:   30 * time.Second,
		HalfOpenMaxReqs:  3,
	}
}

// circuitBreaker 断路器实现
type circuitBreaker struct {
	mu            sync.Mutex
	state         CircuitState
	failureCount  int
	lastFailTime  time.Time
	halfOpenReqs  int
	threshold     int
	timeout       time.Duration
	halfOpenMax   int
}

func newCircuitBreaker(threshold int, timeout time.Duration, halfOpenMax int) *circuitBreaker {
	return &circuitBreaker{
		state:       CircuitClosed,
		threshold:   threshold,
		timeout:     timeout,
		halfOpenMax: halfOpenMax,
	}
}

// allow 检查请求是否被允许通过
func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenReqs = 0
			log.Printf("[circuit-breaker] 断路器进入半开状态，尝试恢复")
			return true
		}
		return false
	case CircuitHalfOpen:
		if cb.halfOpenReqs < cb.halfOpenMax {
			cb.halfOpenReqs++
			return true
		}
		return false
	default:
		return false
	}
}

// recordSuccess 记录成功
func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		cb.halfOpenReqs = 0
		log.Printf("[circuit-breaker] 断路器已恢复，状态: closed")
	}
}

// recordFailure 记录失败
func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailTime = time.Now()

	if cb.state == CircuitClosed && cb.failureCount >= cb.threshold {
		cb.state = CircuitOpen
		log.Printf("[circuit-breaker] 断路器断开！连续失败 %d 次，将于 %s 后尝试恢复",
			cb.failureCount, cb.timeout)
	}

	if cb.state == CircuitHalfOpen {
		cb.state = CircuitOpen
		log.Printf("[circuit-breaker] 半开状态试探失败，断路器重新断开")
	}
}

// ServiceClient 跨服务 HTTP 客户端
type ServiceClient struct {
	config   *Config
	breakers map[string]*circuitBreaker
	mu       sync.RWMutex
}

// New 创建服务客户端
func New(cfg *Config) *ServiceClient {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &ServiceClient{
		config:   cfg,
		breakers: make(map[string]*circuitBreaker),
	}
}

// getBreaker 获取或创建服务的断路器
func (c *ServiceClient) getBreaker(service string) *circuitBreaker {
	c.mu.RLock()
	cb, ok := c.breakers[service]
	c.mu.RUnlock()
	if ok {
		return cb
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// 双重检查
	if cb, ok = c.breakers[service]; ok {
		return cb
	}
	cb = newCircuitBreaker(c.config.CircuitThreshold, c.config.CircuitTimeout, c.config.HalfOpenMaxReqs)
	c.breakers[service] = cb
	return cb
}

// Call 发起跨服务调用
// service: 目标服务名称（如 "player", "cultivation"）
// method: HTTP 方法
// path: API 路径（如 "/api/v1/player/exp"）
// body: 请求体（可选）
func (c *ServiceClient) Call(ctx context.Context, service, method, path string, body interface{}) (*http.Response, error) {
	cb := c.getBreaker(service)

	if !cb.allow() {
		return nil, fmt.Errorf("circuit breaker open for service %s", service)
	}

	var lastErr error
	backoff := c.config.RetryBackoffMin

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避
			wait := time.Duration(float64(backoff) * math.Pow(2, float64(attempt-1)))
			if wait > c.config.RetryBackoffMax {
				wait = c.config.RetryBackoffMax
			}
			log.Printf("[client] 重试 %s (%d/%d)，等待 %v", service, attempt, c.config.MaxRetries, wait)

			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := c.doRequest(ctx, service, method, path, body)
		if err == nil && resp.StatusCode < 500 {
			cb.recordSuccess()
			return resp, nil
		}

		lastErr = err
		if err == nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
		}

		cb.recordFailure()

		// 如果已超过最大重试等待，不再重试
		if time.Duration(attempt)*c.config.RetryBackoffMax > c.config.RetryMaxWait {
			break
		}
	}

	return nil, fmt.Errorf("call to %s%s failed after %d retries: %w", service, path, c.config.MaxRetries, lastErr)
}

// CallJSON 调用并解析 JSON 响应
func (c *ServiceClient) CallJSON(ctx context.Context, service, method, path string, body, result interface{}) error {
	resp, err := c.Call(ctx, service, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("service %s returned %d: %s", service, resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response from %s: %w", service, err)
		}
	}

	return nil
}

// CallFireAndForget 异步 fire-and-forget 调用（不等待结果但不丢失）
// 返回一个 error channel，调用方可选择性等待
func (c *ServiceClient) CallFireAndForget(ctx context.Context, service, method, path string, body interface{}) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		_, err := c.Call(ctx, service, method, path, body)
		if err != nil {
			log.Printf("[client] fire-and-forget call to %s%s failed: %v", service, path, err)
		}
		errCh <- err
	}()

	return errCh
}

// doRequest 执行单次 HTTP 请求
func (c *ServiceClient) doRequest(ctx context.Context, service, method, path string, body interface{}) (*http.Response, error) {
	// 解析目标地址
	addr, err := c.resolveAddr(ctx, service)
	if err != nil {
		return nil, err
	}

	url := addr + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	reqCtx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "cultivation-game/1.0")
	req.Header.Set("X-Service", "cultivation-game")

	return http.DefaultClient.Do(req)
}

// resolveAddr 解析服务地址
func (c *ServiceClient) resolveAddr(ctx context.Context, service string) (string, error) {
	if c.config.Discoverer != nil {
		addr, err := c.config.Discoverer.PickAddr(ctx, service)
		if err == nil {
			return addr, nil
		}
		log.Printf("[client] 服务发现失败，使用默认地址: %v", err)
	}
	// 如果没有服务发现，使用标准地址格式
	return fmt.Sprintf("http://%s", service), nil
}

// GetStats 获取断路器状态统计
func (c *ServiceClient) GetStats() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]string)
	for name, cb := range c.breakers {
		cb.mu.Lock()
		stats[name] = fmt.Sprintf("state=%s failures=%d lastFail=%s",
			cb.state, cb.failureCount, cb.lastFailTime.Format(time.RFC3339))
		cb.mu.Unlock()
	}
	return stats
}
