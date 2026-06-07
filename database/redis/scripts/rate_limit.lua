-- rate_limit.lua
-- 令牌桶限流 (基于滑动窗口)
--
-- 三级限流:
--   Level 1: 每秒限流 (requests/second)
--   Level 2: 每分钟限流 (requests/minute)
--   Level 3: 每小时限流 (requests/hour)
--
-- 算法: 滑动窗口 + 计数器 (非严格令牌桶，更适合游戏场景)
--
-- KEYS[1]: rate_limit:{key}:s  -- 秒级窗口
-- KEYS[2]: rate_limit:{key}:m  -- 分钟级窗口
-- KEYS[3]: rate_limit:{key}:h  -- 小时级窗口
--
-- ARGV[1]: max_requests_per_second
-- ARGV[2]: max_requests_per_minute
-- ARGV[3]: max_requests_per_hour
-- ARGV[4]: now_timestamp  (秒级 Unix 时间戳)
-- ARGV[5]: weight        (本次请求消耗的令牌数, 默认 1)
--
-- 返回值: JSON 字符串
--   {
--     "allowed": true/false,
--     "remaining": { "s": N, "m": N, "h": N },
--     "reset_after": { "s": N, "m": N, "h": N }
--   }

-- 参数解析
local key_s = KEYS[1]
local key_m = KEYS[2]
local key_h = KEYS[3]

local max_s = tonumber(ARGV[1]) or 10    -- 默认每秒 10 次
local max_m = tonumber(ARGV[2]) or 300   -- 默认每分钟 300 次
local max_h = tonumber(ARGV[3]) or 5000  -- 默认每小时 5000 次
local now = tonumber(ARGV[4])
local weight = tonumber(ARGV[5]) or 1

if not now or now < 1 then
    now = redis.call('TIME')[1]
end

-- 当前秒窗口: 使用当前秒作为窗口 ID
local current_second = math.floor(now)
-- 当前分钟窗口: 使用分钟起始时间戳
local current_minute = math.floor(now / 60) * 60
-- 当前小时窗口: 使用小时起始时间戳
local current_hour = math.floor(now / 3600) * 3600

-- 秒级限流检查
local s_count = 0
local s_remain = 0
local s_reset = 0

-- 清理旧的秒窗口 (保留最近 3 秒，用于带宽计算)
redis.call('ZREMRANGEBYSCORE', key_s, 0, current_second - 3)

-- 获取当前秒内请求数
s_count = redis.call('ZCOUNT', key_s, current_second, current_second + 0.999)
local s_allowed = (s_count + weight) <= max_s

if s_allowed then
    -- 记录本次请求
    redis.call('ZADD', key_s, now, now .. ':' .. math.random())
    redis.call('EXPIRE', key_s, 3)
    s_count = s_count + weight
end
s_remain = math.max(0, max_s - (s_count + weight))
-- 重置时间: 当前秒结束
s_reset = math.ceil(now) - now + 1

-- 分钟级限流检查
local m_count = 0
local m_remain = 0
local m_reset = 0

redis.call('ZREMRANGEBYSCORE', key_m, 0, current_minute - 60)

m_count = redis.call('ZCARD', key_m)
local m_allowed = (m_count + weight) <= max_m

if m_allowed then
    redis.call('ZADD', key_m, now, now .. ':' .. math.random())
    redis.call('EXPIRE', key_m, 120)
    m_count = m_count + weight
end
m_remain = math.max(0, max_m - (m_count + weight))
m_reset = 60 - (now - current_minute)

-- 小时级限流检查
local h_count = 0
local h_remain = 0
local h_reset = 0

redis.call('ZREMRANGEBYSCORE', key_h, 0, current_hour - 3600)

h_count = redis.call('ZCARD', key_h)
local h_allowed = (h_count + weight) <= max_h

if h_allowed then
    redis.call('ZADD', key_h, now, now .. ':' .. math.random())
    redis.call('EXPIRE', key_h, 3600)
    h_count = h_count + weight
end
h_remain = math.max(0, max_h - (h_count + weight))
h_reset = 3600 - (now - current_hour)

-- 最终结果: 所有级别都通过才允许
local allowed = s_allowed and m_allowed and h_allowed

local result = {
    allowed = allowed,
    remaining = {
        s = s_remain,
        m = m_remain,
        h = h_remain
    },
    reset_after = {
        s = s_reset,
        m = m_reset,
        h = h_reset
    }
}

return cjson.encode(result)
