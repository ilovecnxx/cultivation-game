// Package sd 提供服务注册与发现功能。
//
// 基于 Redis 实现，支持：
//   - 服务启动时自动注册
//   - 定期心跳保活（TTL 15s，心跳间隔 5s）
//   - 服务发现（按名称获取健康实例列表）
//   - 优雅下线（主动注销）
//   - 健康检查和自动摘除
package sd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultTTL      = 15 * time.Second
	defaultInterval  = 5 * time.Second
	redisKeyPrefix  = "sd:service:"
	redisKeyWorkers = "sd:workers"
)

// Instance 表示一个服务实例
type Instance struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Addr      string            `json:"addr"`
	Port      int               `json:"port"`
	Metadata  map[string]string `json:"metadata"`
	StartTime time.Time         `json:"start_time"`
	LastSeen  time.Time         `json:"last_seen"`
}

// Registrar 服务注册器
type Registrar struct {
	client   *redis.Client
	instance Instance
	ttl      time.Duration
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewRegistrar 创建服务注册器
func NewRegistrar(client *redis.Client, name, addr string, port int, meta map[string]string) *Registrar {
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%s-%d-%d", name, hostname, port, rand.Intn(10000))

	return &Registrar{
		client: client,
		instance: Instance{
			ID:        instanceID,
			Name:      name,
			Addr:      addr,
			Port:      port,
			Metadata:  meta,
			StartTime: time.Now(),
			LastSeen:  time.Now(),
		},
		ttl:      defaultTTL,
		interval: defaultInterval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// SetTTL 设置心跳 TTL 和间隔（用于测试）
func (r *Registrar) SetTTL(ttl, interval time.Duration) {
	r.ttl = ttl
	r.interval = interval
}

// Register 注册服务并开始心跳
func (r *Registrar) Register(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("sd: already registered")
	}

	// 注册实例
	if err := r.heartbeat(ctx); err != nil {
		return fmt.Errorf("sd: failed to register: %w", err)
	}

	r.running = true
	go r.loop(ctx)

	log.Printf("[sd] 服务 %s (%s) 已注册: %s:%d", r.instance.Name, r.instance.ID, r.instance.Addr, r.instance.Port)
	return nil
}

// Deregister 注销服务
func (r *Registrar) Deregister(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}

	r.running = false
	select {
	case r.stopCh <- struct{}{}:
	default:
	}
	<-r.doneCh

	key := redisKeyPrefix + r.instance.Name
	err := r.client.ZRem(ctx, key, r.instance.ID).Err()
	if err != nil {
		return fmt.Errorf("sd: failed to deregister: %w", err)
	}

	log.Printf("[sd] 服务 %s (%s) 已注销", r.instance.Name, r.instance.ID)
	return nil
}

func (r *Registrar) loop(ctx context.Context) {
	defer close(r.doneCh)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.heartbeat(ctx); err != nil {
				log.Printf("[sd] 心跳失败: %v", err)
			}
		case <-r.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (r *Registrar) heartbeat(ctx context.Context) error {
	r.instance.LastSeen = time.Now()
	data, err := json.Marshal(r.instance)
	if err != nil {
		return err
	}

	key := redisKeyPrefix + r.instance.Name
	pipe := r.client.Pipeline()

	// 添加到 sorted set，score 为最后心跳时间
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: r.instance.ID,
	})

	// 存储实例详细信息
	pipe.HSet(ctx, redisKeyWorkers, r.instance.ID, string(data))

	// 设置 sorted set 的过期时间（防止僵尸数据）
	pipe.Expire(ctx, key, r.ttl*3)

	_, err = pipe.Exec(ctx)
	return err
}

// Discoverer 服务发现器
type Discoverer struct {
	client *redis.Client
	mu     sync.RWMutex
	cache  map[string][]Instance // 本地缓存
	ttl    time.Duration
}

// NewDiscoverer 创建服务发现器
func NewDiscoverer(client *redis.Client) *Discoverer {
	return &Discoverer{
		client: client,
		cache:  make(map[string][]Instance),
		ttl:    30 * time.Second,
	}
}

// Discover 获取指定服务的所有健康实例
func (d *Discoverer) Discover(ctx context.Context, name string) ([]Instance, error) {
	// 先查本地缓存
	d.mu.RLock()
	if cached, ok := d.cache[name]; ok {
		d.mu.RUnlock()
		return cached, nil
	}
	d.mu.RUnlock()

	return d.refresh(ctx, name)
}

// refresh 从 Redis 刷新服务实例列表
func (d *Discoverer) refresh(ctx context.Context, name string) ([]Instance, error) {
	key := redisKeyPrefix + name

	// 获取未过期的实例 ID（score > 过期阈值）
	threshold := time.Now().Add(-defaultTTL).Unix()
	ids, err := d.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", threshold),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("sd: failed to discover service %s: %w", name, err)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("sd: no healthy instances found for service %s", name)
	}

	// 获取实例详细信息
	raw, err := d.client.HMGet(ctx, redisKeyWorkers, ids...).Result()
	if err != nil {
		return nil, err
	}

	instances := make([]Instance, 0, len(raw))
	for _, r := range raw {
		if r == nil {
			continue
		}
		var inst Instance
		if err := json.Unmarshal([]byte(r.(string)), &inst); err != nil {
			continue
		}
		instances = append(instances, inst)
	}

	// 更新本地缓存
	d.mu.Lock()
	d.cache[name] = instances
	d.mu.Unlock()

	return instances, nil
}

// Pick 从健康实例中随机选择一个（简单负载均衡）
func (d *Discoverer) Pick(ctx context.Context, name string) (*Instance, error) {
	instances, err := d.Discover(ctx, name)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("sd: no instances available for service %s", name)
	}

	idx := rand.Intn(len(instances))
	return &instances[idx], nil
}

// PickAddr 获取服务地址（格式：addr:port）
func (d *Discoverer) PickAddr(ctx context.Context, name string) (string, error) {
	inst, err := d.Pick(ctx, name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:%d", inst.Addr, inst.Port), nil
}

// InvalidateCache 使本地缓存失效（用于后台定时刷新）
func (d *Discoverer) InvalidateCache() {
	d.mu.Lock()
	d.cache = make(map[string][]Instance)
	d.mu.Unlock()
}

// StartCacheRefresh 启动后台缓存刷新
func (d *Discoverer) StartCacheRefresh(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(d.ttl)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				d.InvalidateCache()
			case <-ctx.Done():
				return
			}
		}
	}()
}
