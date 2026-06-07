// Package ratelimit 提供令牌桶限流器，支持按连接粒度限流。
package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket 令牌桶，每个连接独立持有。
type TokenBucket struct {
	rate       float64   // 每秒令牌填充速率
	capacity   float64   // 桶容量（最大令牌数）
	tokens     float64   // 当前令牌数
	lastRefill time.Time // 上次填充时间
	mu         sync.Mutex
}

// NewTokenBucket 创建令牌桶。
func NewTokenBucket(rate float64, capacity int) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   float64(capacity),
		tokens:     float64(capacity),
		lastRefill: time.Now(),
	}
}

// Allow 消费 1 个令牌，是否允许通过。
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 消费 N 个令牌，是否允许通过。
func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 时间差补填令牌
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

// Available 返回当前可用令牌数。
func (tb *TokenBucket) Available() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokens := tb.tokens + elapsed*tb.rate
	if tokens > tb.capacity {
		tokens = tb.capacity
	}
	return tokens
}

// RateLimiter 限流管理器，管理按 key 分桶的多个令牌桶。
type RateLimiter struct {
	buckets  map[string]*TokenBucket
	rate     float64
	capacity int
	mu       sync.RWMutex
}

// NewRateLimiter 创建限流管理器。
func NewRateLimiter(rate float64, capacity int) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*TokenBucket),
		rate:     rate,
		capacity: capacity,
	}
}

// GetBucket 获取或创建指定 key 的令牌桶。
func (rl *RateLimiter) GetBucket(key string) *TokenBucket {
	rl.mu.RLock()
	bucket, ok := rl.buckets[key]
	rl.mu.RUnlock()
	if ok {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()
	// double-check
	if bucket, ok = rl.buckets[key]; ok {
		return bucket
	}
	bucket = NewTokenBucket(rl.rate, rl.capacity)
	rl.buckets[key] = bucket
	return bucket
}

// Allow 对指定 key 消费 1 个令牌。
func (rl *RateLimiter) Allow(key string) bool {
	return rl.GetBucket(key).Allow()
}

// AllowN 对指定 key 消费 N 个令牌。
func (rl *RateLimiter) AllowN(key string, n int) bool {
	return rl.GetBucket(key).AllowN(n)
}

// RemoveBucket 移除指定 key 的令牌桶（连接断开时清理）。
func (rl *RateLimiter) RemoveBucket(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.buckets, key)
}
