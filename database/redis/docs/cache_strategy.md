# 修仙游戏 Redis 缓存策略

## 1. 三级缓存架构

```
┌─────────────────────────────────────────────────────┐
│                    应用层 (Application)               │
├─────────────────────────────────────────────────────┤
│  L1: 进程内存缓存 (Caffeine / LRU Map)               │
│  ├─ 热点数据: 玩家基本信息、当前场景、技能CD            │
│  ├─ TTL: 5~30 秒, 最大条目: 10,000                   │
│  └─ 淘汰: W-TinyLFU                                  │
├─────────────────────────────────────────────────────┤
│  L2: Redis 分布式缓存                                │
│  ├─ 会话数据、排行榜、背包、限流计数器、在线状态         │
│  ├─ TTL: 依据数据类型 5 分钟~24 小时                   │
│  └─ 淘汰: allkeys-lru (配置见 redis.conf)             │
├─────────────────────────────────────────────────────┤
│  L3: MySQL / MongoDB 持久化层                        │
│  └─ 所有数据的最终一致存储                            │
└─────────────────────────────────────────────────────┘
```

### 读取流程

```
请求数据 → L1 命中? → 返回
           ↓ 未命中
         L2 命中? → 回填 L1 → 返回
           ↓ 未命中
         L3 查询 → 回填 L2 + L1 → 返回
```

### 写入流程

```
写入请求 → 更新 L1 (即时) → 异步写入 L2 → 异步写入 L3
                              ↓
                    定期校验: L2 vs L3 一致性
```

---

## 2. 缓存粒度定义

| 缓存键模式 | 粒度 | 数据类型 | TTL |
|---|---|---|---|
| `player:{id}:profile` | 玩家基本信息 | String (JSON) | 30m |
| `player:{id}:session` | 会话 | String (JSON) | 取决于 session 有效期 |
| `player:{id}:inventory` | 背包 | Hash | 24h |
| `player:{id}:equip` | 装备 | Hash | 24h |
| `player:{id}:skills` | 技能 | Hash | 24h |
| `player:{id}:pet:{pet_id}` | 灵宠 | Hash | 24h |
| `player:{id}:quest:{quest_id}` | 任务进度 | String | 2h |
| `leaderboard:{type}:{season}` | 排行榜 | Sorted Set | 永久 (按赛季淘汰) |
| `email_code:{email}` | 验证码 | String | 5m |
| `rate_limit:{ip}:{action}` | 限流计数 | String | 1m~1h |
| `online:players` | 在线玩家 | Sorted Set | 永久 (带过期分数) |
| `guild:{id}:info` | 宗门信息 | String (JSON) | 1h |
| `guild:{id}:members` | 宗门成员 | Set | 1h |
| `market:listings:{page}` | 交易行 | String (JSON) | 30s |
| `world:boss:{id}:status` | 世界 BOSS | String | 10s~1m |

---

## 3. 过期策略 (TTL 设置)

### TTL 分类原则

| 类别 | TTL | 典型数据 | 说明 |
|---|---|---|---|
| 短时 (volatile) | 5s~1m | 世界 BOSS 血量、场景动态信息 | 高频变动 |
| 会话 (session) | session 有效期 | 玩家登录态、临时 Token | 绑定用户生命周期 |
| 常规 (normal) | 30m~2h | 玩家资料、技能数据、任务进度 | 低频修改 |
| 长时 (long) | 12h~24h | 背包、装备、灵宠 | 数据量大、修改不频繁 |
| 永久 (permanent) | 无 TTL | 排行榜(赛季内)、配置数据 | 由业务逻辑显式淘汰 |
| 限流 (rate-limit) | 窗口期+1s | 令牌桶计数器 | 与限流窗口匹配 |

### 主动淘汰策略

- **定时巡检**: 每 15 分钟扫描一批近过期的 Key，提前重新加载 L3 数据刷新 TTL，避免大量 Key 同时过期。
- **只缓存热数据**: 超过 24 小时未访问的数据从缓存移除 (通过访问计数判断)。

---

## 4. 缓存淘汰策略

### Redis 配置: `allkeys-lru`

```
maxmemory 4gb
maxmemory-policy allkeys-lru
```

选择 `allkeys-lru` 的原因:
- 修仙游戏数据量大但访问有热点集中效应（同服玩家刷同一副本、查看同一排行榜）。
- LRU 自动淘汰长期不访问的冷数据，为热数据腾出空间。
- 不需要区分设置了过期和未设置过期的 Key，统一管理更简洁。

### L1 进程内存淘汰

使用 Caffeine 的 W-TinyLFU 策略:
```java
// 示例 (Java)
Cache<Key, Value> cache = Caffeine.newBuilder()
    .maximumSize(10_000)
    .expireAfterWrite(Duration.ofSeconds(30))
    .recordStats()
    .build();
```

---

## 5. 缓存穿透/击穿/雪崩防护方案

### 5.1 缓存穿透 (查询不存在的数据)

**现象**: 频繁查询 DB 中不存在的数据，绕过缓存直击 L3。

**防护**:

1. **布隆过滤器 (Bloom Filter)**:
   - 预加载所有合法 PlayerID、ItemID 到布隆过滤器。
   - 查询前先检查布隆过滤器，不存在则直接返回空。
   - 重建策略: 每日凌晨重建一次，误判率控制在 1% 以内。

2. **空值缓存**:
   - 对 L3 查询为空的 Key，在 L2 缓存空值标记，TTL 设置为 30~60 秒。
   - Lua 脚本实现: `get_or_nil` 返回 nil 时，写入 `nil:{key}` 标记。

3. **参数校验**:
   - 应用层拦截明显不合法的请求（如负数 ID、超长字符串）。

### 5.2 缓存击穿 (热点 Key 过期)

**现象**: 某个极高并发的 Key 在过期瞬间，大量请求穿透到 L3。

**防护**:

1. **互斥锁 (Mutex Lock)**:
   ```lua
   -- 尝试获取重建锁
   local locked = redis.call('SET', lock_key, 1, 'NX', 'EX', 5)
   if locked then
       -- 只有拿到锁的请求去查 L3 并回填缓存
       -- 其他请求等待或返回旧缓存
   else
       -- 降级: 返回旧数据或等待重试
   end
   ```

2. **逻辑过期 (Active Refresh)**:
   - 缓存永不真正过期，但存储逻辑过期时间。
   - 后台线程扫描即将逻辑过期的 Key，主动刷新。
   - 高并发读取时若发现逻辑过期，一个线程去刷新，其余返回旧值。

3. **多级缓存**:
   - L1 进程内存的 TTL 短于 L2 Redis，L2 失效时 L1 仍可用做缓冲。

### 5.3 缓存雪崩 (大量 Key 同时过期)

**现象**: 大范围 Key 在同一时段过期，导致请求全部落到 L3。

**防护**:

1. **过期时间打散**:
   - 基础 TTL + 随机偏移 10%~30%:
   ```
   TTL = base_TTL + random(0, base_TTL * 0.3)
   ```

2. **多级缓存**:
   - 同击穿防护，L1 做最后屏障。

3. **降级 / 限流**:
   - L3 负载过高时，对缓存未命中的请求直接返回降级数据（略旧但可用）。
   - Redis 缓存未命中时触发本地限流，保护 DB。

4. **主从/集群高可用**:
   - Redis Cluster 3主3从 (见 docker-compose.redis.yml)。
   - 从库可提供只读缓存，主库故障时自动切换。

---

## 6. 数据一致性保证

### 策略: Write-Behind + 定期校验

```
应用层 → [L1] → [L2 Redis] → [异步队列] → [L3 DB]
                            ↕
                      [定期校验服务]
```

### 6.1 写入路径

1. **立即写入 L1**: 保证当前请求立刻读到最新数据。
2. **异步写入 L2**: 请求返回后，通过消息队列批量写入 Redis。
   - 使用 Lua 脚本配合乐观锁 (版本号) 防止并发覆盖。
3. **异步写入 L3**: 同样通过消息队列，合并写入 MySQL/MongoDB。
   - 使用 CAS (Compare And Swap) 思想: 只有在版本号匹配时才写入。

### 6.2 最终一致性窗口

| 层级 | 预计延迟 | 说明 |
|---|---|---|
| L1 → L2 | ~50ms | 消息队列 + Redis Lua 脚本 |
| L2 → L3 | ~200ms~1s | 消息队列 + 批量写入 DB |

### 6.3 定期校验

```python
# 伪代码: 校验服务
def reconcile():
    while True:
        # 抽样 0.1% 的缓存 Key
        for key in scan_cursor("player:*"):
            l2_data = redis.get(key)
            l3_data = db.query(key)
            if l2_data.version < l3_data.version:
                rpush("reconcile_queue", key)  # 重新加载缓存
        sleep(300)  # 每 5 分钟执行一次
```

### 6.4 最终一致性保证是强还是弱

**弱一致性 (最终一致)**:
- 写入 L3 可能在 1s 之后才完成。
- 核心场景（充值、交易）需等待 L3 写入成功才返回，此时使用同步双写。
- 非核心场景（查看排行榜、背包）允许短暂的不一致。

### 6.5 缓存淘汰通知

- 玩家主动删除数据（如出售道具）时，使用 Redis Pub/Sub 或 Keyspace Notification 通知所有相关服务实例清空 L1 缓存。
- 配置: `notify-keyspace-events Ex` 监听 Key 过期事件。

---

## 7. 监控与告警

### Redis 监控指标

| 指标 | 阈值 | 动作 |
|---|---|---|
| 内存使用率 > 80% | Warning | 扩容或检查淘汰策略 |
| 缓存命中率 < 85% | Warning | 检查 TTL 设置是否合理 |
| 阻塞命令数 > 0 | Critical | 检查慢查询日志 (SLOWLOG) |
| 主从延迟 > 2s | Critical | 检查网络或从库负载 |
| OOM 计数 > 0 | Critical | 重启 + 检查 maxmemory 配置 |
| `keyspace_misses` 突增 | Warning | 可能发生穿透/击穿 |

### 告警通知

- 通过 Grafana + Prometheus 收集 Redis Exporter 数据。
- 企业微信 / 钉钉 / Slack 机器人推送告警。

---

## 8. 最佳实践

1. **批量操作优先 Pipeline/Lua**: 减少网络 RTT。
2. **避免大 Key**: 单个 Hash 不超过 10,000 个 field，单个 String 不超过 10MB。
3. **冷热分离**: 超过 7 天的排行历史数据迁移到 MongoDB，Redis 只保留当前赛季。
4. **命名空间规范**: `业务:对象:ID:字段`，全部小写。
5. **禁用高危命令**: 线上禁用 `KEYS`、`FLUSHALL`、`FLUSHDB` (Rename 处理)。
6. **慢查询阈值**: `slowlog-log-slower-than 10000` (10ms)。
