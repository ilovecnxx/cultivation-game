-- player_session.lua
-- 玩家会话管理
--
-- KEYS[1]: session:{player_id}        -- 会话 Key
-- ARGV[1]: session_data (JSON)       -- 会话数据
-- ARGV[2]: ttl_seconds               -- 过期时间 (秒)
-- ARGV[3]: online_set_key            -- 在线玩家集合 Key (Sorted Set)
-- ARGV[4]: current_timestamp         -- 当前时间戳 (用于 ZADD 分数)
-- ARGV[5]: heartbeat_ttl             -- 心跳超时时间 (秒)
--
-- 返回值: 1 = 成功, 0 = 失败

-- 参数校验
local player_key = KEYS[1]
local session_data = ARGV[1]
local ttl = tonumber(ARGV[2])
local online_key = ARGV[3]
local timestamp = tonumber(ARGV[4])
local heartbeat_ttl = tonumber(ARGV[5])

if not player_key or not session_data then
    return 0
end

if not ttl or ttl < 1 then
    ttl = 1800  -- 默认 30 分钟
end

if not timestamp or timestamp < 1 then
    timestamp = redis.call('TIME')[1]
end

if not heartbeat_ttl or heartbeat_ttl < 1 then
    heartbeat_ttl = 60  -- 默认 60 秒心跳超时
end

-- 检查是否已有旧会话 (主动踢下线)
local existing = redis.call('GET', player_key)
if existing then
    -- 如果已有会话且数据不同，记录被踢日志 (可选)
    -- 这里直接覆盖
end

-- 设置会话数据
redis.call('SET', player_key, session_data, 'EX', ttl)

-- 更新在线集合: 分数为当前时间戳，按分数排序可以快速清理过期玩家
redis.call('ZADD', online_key, timestamp, player_key)

-- 设置在线集合的过期哨兵: 确保即使会话被删，在线集合也有心跳超时保护
-- 使用第二个 Sorted Set 存储心跳时间
local heartbeat_key = online_key .. ':heartbeat'
redis.call('ZADD', heartbeat_key, timestamp, player_key)
redis.call('EXPIRE', heartbeat_key, heartbeat_ttl + 10)

return 1
