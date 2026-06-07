-- leaderboard.lua
-- 排行榜管理
--
-- 命令模式:
--   1. update  - 更新玩家分数
--   2. rank    - 获取玩家排名
--   3. top     - 获取排行榜 Top N
--   4. around  - 获取玩家前后排名段
--   5. remove  - 移除玩家
--   6. count   - 排行榜总人数
--
-- KEYS[1]: leaderboard:{type}:{season}  -- 排行榜 Sorted Set Key
-- ARGV[1]: mode                         -- 操作模式
-- ARGV[2]: player_id                    -- 玩家 ID (update/rank/remove/around)
-- ARGV[3]: score                        -- 新分数 (update)
-- ARGV[4]: extra_data (JSON)            -- 扩展数据 (update, 可选)
-- ARGV[5]: range_start                  -- 起始排名 (top/around)
-- ARGV[6]: range_end                    -- 结束排名 (top/around)
--
-- 返回值: 根据 mode 不同返回不同结构

local leaderboard_key = KEYS[1]
local mode = ARGV[1]

if mode == 'update' then
    -- 更新玩家分数
    -- ARGV[2]: player_id
    -- ARGV[3]: score
    -- ARGV[4]: extra_data (JSON, 可选) — 存储在 Hash 中用于扩展信息(头像、名称等)

    local player_id = ARGV[2]
    local score = tonumber(ARGV[3])

    if not player_id or not score then
        return {err = "invalid params"}
    end

    -- 更新主排行榜分数
    redis.call('ZADD', leaderboard_key, score, player_id)

    -- 如果有扩展数据，存入关联 Hash
    if ARGV[4] and ARGV[4] ~= '' then
        local extra_key = leaderboard_key .. ':extra'
        redis.call('HSET', extra_key, player_id, ARGV[4])
        -- 扩展数据 TTL 与排行榜一致 (排行榜本身无 TTL，但扩展数据可设长 TTL)
        redis.call('EXPIRE', extra_key, 86400)
    end

    -- 返回当前排名
    local rank = redis.call('ZREVRANK', leaderboard_key, player_id)
    return {ok = 1, player_id = player_id, score = score, rank = (rank or 0) + 1}

elseif mode == 'rank' then
    -- 获取玩家排名
    -- ARGV[2]: player_id
    -- 返回: {rank, score}

    local player_id = ARGV[2]
    if not player_id then
        return {err = "missing player_id"}
    end

    local rank = redis.call('ZREVRANK', leaderboard_key, player_id)
    if rank == false then
        return {rank = -1, score = 0}
    end

    local score = redis.call('ZSCORE', leaderboard_key, player_id)
    return {rank = rank + 1, score = score}

elseif mode == 'top' then
    -- 获取排行榜 Top N
    -- ARGV[5]: range_start (0-based)
    -- ARGV[6]: range_end   (0-based, 例如 99 = Top 100)

    local start_idx = tonumber(ARGV[5]) or 0
    local end_idx = tonumber(ARGV[6]) or 99

    -- 获取排行榜片段 (含分数)
    local list = redis.call('ZREVRANGE', leaderboard_key, start_idx, end_idx, 'WITHSCORES')

    -- 组装返回结果
    local result = {}
    local extra_key = leaderboard_key .. ':extra'
    for i = 1, #list, 2 do
        local player_id = list[i]
        local score = list[i + 1]
        local entry = {player_id = player_id, score = score, rank = (start_idx + (i + 1) / 2)}

        -- 读取扩展数据
        local extra = redis.call('HGET', extra_key, player_id)
        if extra then
            entry.extra = extra
        end

        table.insert(result, entry)
    end

    local total = redis.call('ZCARD', leaderboard_key)
    return {list = result, total = total}

elseif mode == 'around' then
    -- 获取玩家前后排名段
    -- ARGV[2]: player_id
    -- ARGV[5]: range_before (往前取 N 名)
    -- ARGV[6]: range_after  (往后取 N 名)

    local player_id = ARGV[2]
    local before = tonumber(ARGV[5]) or 5
    local after = tonumber(ARGV[6]) or 5

    local rank = redis.call('ZREVRANK', leaderboard_key, player_id)
    if rank == false then
        return {err = "player not ranked"}
    end

    local start_idx = math.max(0, rank - before)
    local end_idx = rank + after

    local list = redis.call('ZREVRANGE', leaderboard_key, start_idx, end_idx, 'WITHSCORES')

    local result = {}
    local extra_key = leaderboard_key .. ':extra'
    for i = 1, #list, 2 do
        local pid = list[i]
        local score = list[i + 1]
        local entry = {player_id = pid, score = score, rank = start_idx + (i + 1) / 2}
        local extra = redis.call('HGET', extra_key, pid)
        if extra then
            entry.extra = extra
        end
        table.insert(result, entry)
    end

    return {list = result, player_rank = rank + 1}

elseif mode == 'remove' then
    -- 移除玩家
    local player_id = ARGV[2]
    if not player_id then
        return {err = "missing player_id"}
    end

    redis.call('ZREM', leaderboard_key, player_id)
    local extra_key = leaderboard_key .. ':extra'
    redis.call('HDEL', extra_key, player_id)

    return {ok = 1, player_id = player_id}

elseif mode == 'count' then
    -- 获取排行榜总人数
    local count = redis.call('ZCARD', leaderboard_key)
    return {count = count}

else
    return {err = "unknown mode: " .. mode}
end
