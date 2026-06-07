-- online_tracker.lua
-- 在线玩家追踪管理
--
-- 数据结构:
--   online:players             -> Sorted Set (score = 最后心跳时间戳)
--   online:players:heartbeat   -> Sorted Set (score = 最后心跳时间戳, 更频繁更新)
--   online:players:count_log   -> List (定时记录在线人数用于统计)
--
-- 命令模式:
--   1. heartbeat  - 更新玩家心跳
--   2. login      - 玩家上线
--   3. logout     - 玩家下线
--   4. cleanup    - 清理超时玩家
--   5. count      - 统计在线人数 (含细分)
--   6. is_online  - 检查玩家是否在线
--   7. list       - 列出在线玩家
--   8. stats      - 获取在线统计历史
--
-- KEYS[1]: online:players          -- 在线玩家 Sorted Set
-- KEYS[2]: online:players:heartbeat -- 心跳 Sorted Set
-- KEYS[3]: online:players:count_log -- 在线人数日志 List
-- KEYS[4]: player:{id}:session      -- 玩家会话 Key (可选, 用于联动)
--
-- ARGV[1]: mode
-- ARGV[2]: player_id          (login/logout/heartbeat/is_online)
-- ARGV[3]: current_timestamp  (所有模式)
-- ARGV[4]: heartbeat_timeout  (秒, 默认 60)
-- ARGV[5]: session_ttl        (秒, 仅 login 模式)
-- ARGV[6]: session_data (JSON, 仅 login 模式)

local online_key = KEYS[1]
local heartbeat_key = KEYS[2]
local count_log_key = KEYS[3]
local session_key = KEYS[4]

local mode = ARGV[1]
local player_id = ARGV[2]
local now = tonumber(ARGV[3]) or redis.call('TIME')[1]
local heartbeat_timeout = tonumber(ARGV[4]) or 60
local session_ttl = tonumber(ARGV[5]) or 1800
local session_data = ARGV[6]

if mode == 'login' then
    -- 玩家上线
    if not player_id then
        return {err = "missing player_id"}
    end

    -- 如果已在线, 先移除旧记录 (防止重复)
    redis.call('ZREM', online_key, player_id)
    redis.call('ZREM', heartbeat_key, player_id)

    -- 记录上线, score = 当前时间戳
    redis.call('ZADD', online_key, now, player_id)
    redis.call('ZADD', heartbeat_key, now, player_id)

    -- 设置会话 (如果提供了 session_key)
    if session_key and session_key ~= '' and session_data then
        redis.call('SET', session_key, session_data, 'EX', session_ttl)
    end

    -- 记录上线事件到日志 (可选)
    redis.call('LPUSH', count_log_key, 'login:' .. player_id .. ':' .. now)
    redis.call('LTRIM', count_log_key, 0, 999)  -- 保留最近 1000 条

    return {ok = 1, action = 'login', player_id = player_id, timestamp = now}

elseif mode == 'logout' then
    -- 玩家下线
    if not player_id then
        return {err = "missing player_id"}
    end

    local existed = redis.call('ZSCORE', online_key, player_id)

    redis.call('ZREM', online_key, player_id)
    redis.call('ZREM', heartbeat_key, player_id)

    -- 清除会话
    if session_key and session_key ~= '' then
        redis.call('DEL', session_key)
    end

    if existed then
        redis.call('LPUSH', count_log_key, 'logout:' .. player_id .. ':' .. now)
        redis.call('LTRIM', count_log_key, 0, 999)
    end

    return {ok = 1, action = 'logout', player_id = player_id, was_online = (existed ~= false)}

elseif mode == 'heartbeat' then
    -- 玩家心跳
    -- 更新心跳时间和在线时间, 不重置整个 session 的 TTL (会话由 session TTL 管理)

    if not player_id then
        return {err = "missing player_id"}
    end

    -- 检查玩家是否在在线集合中
    local is_online = redis.call('ZSCORE', online_key, player_id)
    if not is_online then
        -- 不在在线集合中 -> 需要重新登录
        return {ok = 0, error = "not online", action = 'heartbeat'}
    end

    -- 更新心跳: 使用单独的 Sorted Set 避免频繁更新 online_key
    redis.call('ZADD', heartbeat_key, now, player_id)
    redis.call('EXPIRE', heartbeat_key, heartbeat_timeout + 30)

    -- 同时更新在线时间 (降低频次: 每 30 秒才更新一次 online_key 分数)
    local last_online_update = redis.call('ZSCORE', online_key, player_id)
    if not last_online_update or (now - last_online_update) > 30 then
        redis.call('ZADD', online_key, now, player_id)
    end

    return {ok = 1, action = 'heartbeat', player_id = player_id, timestamp = now}

elseif mode == 'cleanup' then
    -- 清理超时玩家 (由定时任务调用)
    -- 清理条件: 心跳超过 heartbeat_timeout 秒未更新

    local cutoff = now - heartbeat_timeout

    -- 从 heartbeat_key 中找出超时玩家
    local expired = redis.call('ZRANGEBYSCORE', heartbeat_key, 0, cutoff)

    local removed_count = 0
    if #expired > 0 then
        -- 从 online_key 和 heartbeat_key 中移除
        redis.call('ZREM', online_key, unpack(expired))
        redis.call('ZREM', heartbeat_key, unpack(expired))

        removed_count = #expired

        -- 记录清理事件
        for _, pid in ipairs(expired) do
            redis.call('LPUSH', count_log_key, 'timeout:' .. pid .. ':' .. now)
        end
        redis.call('LTRIM', count_log_key, 0, 999)

        -- 清理对应的 session (可选, 如果 session_key 模式传入)
        -- 这里可以批量删除 session, 但为了避免阻塞, 建议由外部程序处理
    end

    -- 获取当前在线人数 (用于记录统计)
    local online_count = redis.call('ZCARD', online_key)

    -- 每 5 分钟记录一次在线人数到日志 - 仅在分钟能被 5 整除时记录
    local minute = math.floor(now / 60)
    if minute % 5 == 0 then
        local log_entry = 'snapshot:' .. now .. ':' .. online_count
        -- 检查最近是否已记录 (避免重复)
        local last_entry = redis.call('LINDEX', count_log_key, 0)
        if last_entry ~= log_entry then
            redis.call('LPUSH', count_log_key, log_entry)
            redis.call('LTRIM', count_log_key, 0, 9999)  -- 保留更多历史
        end
    end

    return {ok = 1, removed = removed_count, online_count = online_count, cutoff = cutoff}

elseif mode == 'count' then
    -- 统计在线人数
    local total = redis.call('ZCARD', online_key)

    -- 细分: 按最近活跃时间分组
    local recent_60s = redis.call('ZCOUNT', heartbeat_key, now - 60, now)
    local recent_300s = redis.call('ZCOUNT', heartbeat_key, now - 300, now)

    return {
        total = total,
        active_60s = recent_60s,
        active_300s = recent_300s,
        timestamp = now
    }

elseif mode == 'is_online' then
    -- 检查单个玩家是否在线
    if not player_id then
        return {err = "missing player_id"}
    end

    local score = redis.call('ZSCORE', online_key, player_id)
    if score then
        local last_active = now - score
        return {online = true, player_id = player_id, last_active = last_active, idle_seconds = last_active}
    else
        return {online = false, player_id = player_id}
    end

elseif mode == 'list' then
    -- 列出所有在线玩家 (分页)
    -- ARGV[6]: offset (默认 0)
    -- ARGV[7]: limit  (默认 100)

    local offset = tonumber(ARGV[6]) or 0
    local limit = tonumber(ARGV[7]) or 100

    local total = redis.call('ZCARD', online_key)

    -- ZRANGE 按分数升序 (最早在线 -> 最新在线)
    local players = redis.call('ZRANGE', online_key, offset, offset + limit - 1, 'WITHSCORES')

    local result = {}
    for i = 1, #players, 2 do
        local pid = players[i]
        local score = tonumber(players[i + 1])
        result[#result + 1] = {
            player_id = pid,
            online_since = score,
            idle_seconds = now - score
        }
    end

    return {total = total, players = result, offset = offset, limit = limit}

elseif mode == 'stats' then
    -- 获取在线统计历史
    -- ARGV[6]: limit (返回最近多少条, 默认 100)
    local log_limit = tonumber(ARGV[6]) or 100

    local logs = redis.call('LRANGE', count_log_key, 0, log_limit - 1)
    local snapshots = {}
    local events = {}

    for _, entry in ipairs(logs) do
        if entry:sub(1, 8) == 'snapshot:' then
            local _, ts, count = entry:match('([^:]+):([^:]+):([^:]+)')
            snapshots[#snapshots + 1] = {timestamp = tonumber(ts), count = tonumber(count)}
        else
            events[#events + 1] = entry
        end
    end

    return {snapshots = snapshots, recent_events = events}

else
    return {err = "unknown mode: " .. mode}
end
