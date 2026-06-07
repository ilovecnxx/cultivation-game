-- inventory_cache.lua
-- 背包缓存管理 (使用 Hash)
--
-- 命令模式:
--   1. load       - 批量加载背包物品到缓存
--   2. get        - 获取单个物品
--   3. get_batch  - 批量获取物品
--   4. set        - 设置/更新单个物品
--   5. set_batch  - 批量设置物品
--   6. remove     - 移除物品
--   7. remove_batch - 批量移除物品
--   8. count      - 统计背包物品数量
--   9. version    - 获取/比对版本号
--
-- KEYS[1]: player:{id}:inventory    -- 背包 Hash Key
-- KEYS[2]: player:{id}:inventory:ver -- 版本号 Key (String)
--
-- ARGV[1]: mode
-- ARGV[n]: mode 相关参数

local inventory_key = KEYS[1]
local version_key = KEYS[2]
local mode = ARGV[1]

-- 工具函数: 生成新版本号 (时间戳 + 随机数)
local function new_version()
    local ts = redis.call('TIME')[1]
    local rand = math.random(1000, 9999)
    return tostring(ts) .. '-' .. tostring(rand)
end

if mode == 'load' then
    -- 批量加载物品到缓存
    -- ARGV[2..n]: item_id1, item_data1, item_id2, item_data2, ...
    -- 返回: {ok=1, count=N, version=V}

    local args = {}
    for i = 2, #ARGV do
        args[#args + 1] = ARGV[i]
    end

    if #args == 0 or #args % 2 ~= 0 then
        return {err = "invalid arguments: need pairs of item_id, item_data"}
    end

    -- 批量写入 Hash
    redis.call('HSET', inventory_key, unpack(args))

    -- 设置版本号
    local ver = new_version()
    redis.call('SET', version_key, ver)
    redis.call('EXPIRE', inventory_key, 86400)
    redis.call('EXPIRE', version_key, 86400)

    local count = #args / 2
    return {ok = 1, count = count, version = ver}

elseif mode == 'get' then
    -- 获取单个物品
    -- ARGV[2]: item_id
    -- 返回: {item_id, item_data} 或 nil

    local item_id = ARGV[2]
    if not item_id then
        return {err = "missing item_id"}
    end

    local data = redis.call('HGET', inventory_key, item_id)
    if not data then
        return nil
    end

    return {item_id = item_id, data = data}

elseif mode == 'get_batch' then
    -- 批量获取物品
    -- ARGV[2..n]: item_id1, item_id2, ...
    -- 返回: {items = {id1: data1, id2: data2, ...}, version = V}

    local item_ids = {}
    for i = 2, #ARGV do
        item_ids[#item_ids + 1] = ARGV[i]
    end

    if #item_ids == 0 then
        return {items = {}, version = nil}
    end

    local data_list = redis.call('HMGET', inventory_key, unpack(item_ids))
    local result = {}
    for i = 1, #item_ids do
        if data_list[i] then
            result[item_ids[i]] = data_list[i]
        end
    end

    local version = redis.call('GET', version_key)

    return {items = result, version = version}

elseif mode == 'set' then
    -- 设置/更新单个物品
    -- ARGV[2]: item_id
    -- ARGV[3]: item_data (JSON)
    -- ARGV[4]: check_version (可选，乐观锁版本号)
    -- 返回: {ok=1, version=V} 或 {err="version conflict"}

    local item_id = ARGV[2]
    local item_data = ARGV[3]
    local check_version = ARGV[4]

    if not item_id or not item_data then
        return {err = "missing item_id or item_data"}
    end

    -- 乐观锁: 如果指定了版本号，先检查是否匹配
    if check_version then
        local current_ver = redis.call('GET', version_key)
        if current_ver and current_ver ~= check_version then
            return {err = "version conflict", current_version = current_ver}
        end
    end

    redis.call('HSET', inventory_key, item_id, item_data)

    local ver = new_version()
    redis.call('SET', version_key, ver)
    redis.call('EXPIRE', inventory_key, 86400)
    redis.call('EXPIRE', version_key, 86400)

    return {ok = 1, version = ver}

elseif mode == 'set_batch' then
    -- 批量设置物品
    -- ARGV[2..n]: item_id1, item_data1, item_id2, item_data2, ...
    -- ARGV[n+1]: check_version (可选)
    -- 返回: {ok=1, count=N, version=V}

    local check_version = nil
    local args = {}
    local arg_count = #ARGV

    -- 检查最后一个参数是否为版本号 (非 item_data 格式)
    if arg_count > 2 and arg_count % 2 == 0 then
        -- 奇数个额外参数 -> 最后一个为 check_version
        check_version = ARGV[arg_count]
        arg_count = arg_count - 1
    end

    for i = 2, arg_count do
        args[#args + 1] = ARGV[i]
    end

    if #args == 0 or #args % 2 ~= 0 then
        return {err = "invalid arguments"}
    end

    -- 乐观锁检查
    if check_version then
        local current_ver = redis.call('GET', version_key)
        if current_ver and current_ver ~= check_version then
            return {err = "version conflict", current_version = current_ver}
        end
    end

    redis.call('HSET', inventory_key, unpack(args))

    local ver = new_version()
    redis.call('SET', version_key, ver)
    redis.call('EXPIRE', inventory_key, 86400)
    redis.call('EXPIRE', version_key, 86400)

    local count = #args / 2
    return {ok = 1, count = count, version = ver}

elseif mode == 'remove' then
    -- 移除单个物品
    -- ARGV[2]: item_id
    -- 返回: {ok=1, version=V} 或 {ok=0}

    local item_id = ARGV[2]
    if not item_id then
        return {err = "missing item_id"}
    end

    local removed = redis.call('HDEL', inventory_key, item_id)
    if removed == 1 then
        local ver = new_version()
        redis.call('SET', version_key, ver)
        redis.call('EXPIRE', inventory_key, 86400)
        redis.call('EXPIRE', version_key, 86400)
        return {ok = 1, version = ver}
    else
        return {ok = 0}
    end

elseif mode == 'remove_batch' then
    -- 批量移除物品
    -- ARGV[2..n]: item_id1, item_id2, ...
    -- 返回: {ok=1, removed=N, version=V}

    local item_ids = {}
    for i = 2, #ARGV do
        item_ids[#item_ids + 1] = ARGV[i]
    end

    if #item_ids == 0 then
        return {ok = 1, removed = 0, version = nil}
    end

    local removed = redis.call('HDEL', inventory_key, unpack(item_ids))
    if removed > 0 then
        local ver = new_version()
        redis.call('SET', version_key, ver)
        redis.call('EXPIRE', inventory_key, 86400)
        redis.call('EXPIRE', version_key, 86400)
        return {ok = 1, removed = removed, version = ver}
    else
        return {ok = 1, removed = 0}
    end

elseif mode == 'count' then
    -- 统计背包物品数量
    local count = redis.call('HLEN', inventory_key)
    local version = redis.call('GET', version_key)
    return {count = count, version = version}

elseif mode == 'version' then
    -- 获取当前版本号
    local version = redis.call('GET', version_key)
    return {version = version}

else
    return {err = "unknown mode: " .. mode}
end
