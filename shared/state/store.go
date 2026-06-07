// Package state 提供 Redis 支持的游戏状态持久化。
//
// 用于 combat 和 world 等服务的状态存储，
// 支持定期快照和数据恢复，防止重启导致数据丢失。
package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix    = "state:"
	saveInterval = 30 * time.Second // 定期保存间隔
)

// Store 状态存储
type Store struct {
	client   *redis.Client
	mu       sync.RWMutex
	cache    map[string]interface{} // 本地写入缓冲
	dirty    map[string]bool        // 脏标记
	batch    int                    // 攒批大小
	stopCh   chan struct{}
	running  bool
}

// NewStore 创建状态存储
func NewStore(client *redis.Client) *Store {
	return &Store{
		client: client,
		cache:  make(map[string]interface{}),
		dirty:  make(map[string]bool),
		batch:  100,
		stopCh: make(chan struct{}),
	}
}

// StartAutoSave 启动定期自动保存
func (s *Store) StartAutoSave(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(saveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.Flush(ctx); err != nil {
					log.Printf("[state] 自动保存失败: %v", err)
				}
			case <-s.stopCh:
				// 停止前最后一次保存
				if err := s.Flush(context.Background()); err != nil {
					log.Printf("[state] 最终保存失败: %v", err)
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop 停止自动保存
func (s *Store) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

// Set 设置状态（先缓存，定期刷入 Redis）
func (s *Store) Set(ctx context.Context, key string, value interface{}) error {
	s.mu.Lock()
	s.cache[key] = value
	s.dirty[key] = true
	needFlush := len(s.dirty) >= s.batch
	s.mu.Unlock()

	if needFlush {
		return s.Flush(ctx)
	}
	return nil
}

// SetImmediate 立即写入 Redis（用于关键数据）
func (s *Store) SetImmediate(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("state: marshal error: %w", err)
	}

	fullKey := keyPrefix + key
	return s.client.Set(ctx, fullKey, string(data), 24*time.Hour).Err()
}

// Get 获取状态（先查缓存，再查 Redis）
func (s *Store) Get(ctx context.Context, key string, target interface{}) error {
	// 先查本地缓存
	s.mu.RLock()
	if val, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		return copyValue(val, target)
	}
	s.mu.RUnlock()

	// 查 Redis
	fullKey := keyPrefix + key
	data, err := s.client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		return fmt.Errorf("state: key %s not found", key)
	}
	if err != nil {
		return fmt.Errorf("state: get error: %w", err)
	}

	return json.Unmarshal([]byte(data), target)
}

// Delete 删除状态
func (s *Store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.cache, key)
	delete(s.dirty, key)
	s.mu.Unlock()

	fullKey := keyPrefix + key
	return s.client.Del(ctx, fullKey).Err()
}

// Flush 将所有脏数据刷入 Redis
func (s *Store) Flush(ctx context.Context) error {
	s.mu.Lock()
	dirtyKeys := make([]string, 0, len(s.dirty))
	for k := range s.dirty {
		dirtyKeys = append(dirtyKeys, k)
	}
	s.mu.Unlock()

	if len(dirtyKeys) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	flushed := make([]string, 0, len(dirtyKeys))

	s.mu.RLock()
	for _, k := range dirtyKeys {
		if val, ok := s.cache[k]; ok {
			data, err := json.Marshal(val)
			if err != nil {
				continue
			}
			fullKey := keyPrefix + k
			pipe.Set(ctx, fullKey, string(data), 24*time.Hour)
			flushed = append(flushed, k)
		}
	}
	s.mu.RUnlock()

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("state: flush error: %w", err)
	}

	// 清除脏标记
	s.mu.Lock()
	for _, k := range flushed {
		delete(s.dirty, k)
	}
	s.mu.Unlock()

	if len(flushed) > 0 {
		log.Printf("[state] 已刷入 %d 条状态数据到 Redis", len(flushed))
	}
	return nil
}

// copyValue 将一个值拷贝到目标指针
func copyValue(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
