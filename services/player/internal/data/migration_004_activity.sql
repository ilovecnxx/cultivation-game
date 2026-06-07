-- ============================================================
-- 运营活动框架 - 数据库迁移
-- Version: 004
-- Description: 限时活动、战令、签到增强、成就称号增强
-- ============================================================

-- 限时活动表
CREATE TABLE IF NOT EXISTS limited_events (
    id VARCHAR(64) PRIMARY KEY COMMENT '活动ID',
    name VARCHAR(64) NOT NULL COMMENT '活动名称',
    type VARCHAR(32) NOT NULL COMMENT '活动类型: exp_boost/drop_boost/special_boss/collection/ranking/recharge/fortune',
    description VARCHAR(512) DEFAULT '' COMMENT '活动描述',
    start_time DATETIME NOT NULL COMMENT '开始时间',
    end_time DATETIME NOT NULL COMMENT '结束时间',
    min_realm INT DEFAULT 1 COMMENT '最低参与境界',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_time (start_time, end_time),
    INDEX idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='限时活动定义';

-- 活动奖励表
CREATE TABLE IF NOT EXISTS event_rewards (
    id VARCHAR(64) PRIMARY KEY COMMENT '奖励ID',
    event_id VARCHAR(64) NOT NULL COMMENT '关联活动ID',
    item_id BIGINT NOT NULL COMMENT '物品ID',
    item_name VARCHAR(64) DEFAULT '' COMMENT '物品名称',
    quantity INT DEFAULT 1 COMMENT '数量',
    probability DECIMAL(5,4) DEFAULT 1.0000 COMMENT '概率(0-1)',
    is_guaranteed TINYINT(1) DEFAULT 1 COMMENT '是否必定获得',
    FOREIGN KEY (event_id) REFERENCES limited_events(id) ON DELETE CASCADE,
    INDEX idx_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='活动奖励配置';

-- 活动条件表
CREATE TABLE IF NOT EXISTS event_conditions (
    id VARCHAR(64) PRIMARY KEY COMMENT '条件ID',
    event_id VARCHAR(64) NOT NULL COMMENT '关联活动ID',
    type VARCHAR(32) NOT NULL COMMENT '条件类型: kill_monsters/collect_items/cultivate_time/spend_stones',
    target BIGINT NOT NULL DEFAULT 0 COMMENT '目标值',
    progress BIGINT NOT NULL DEFAULT 0 COMMENT '当前进度',
    priority INT DEFAULT 0 COMMENT '优先级',
    FOREIGN KEY (event_id) REFERENCES limited_events(id) ON DELETE CASCADE,
    INDEX idx_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='活动条件/任务';

-- 玩家活动进度表
CREATE TABLE IF NOT EXISTS event_progress (
    id VARCHAR(64) PRIMARY KEY COMMENT '进度ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    event_id VARCHAR(64) NOT NULL COMMENT '活动ID',
    progress BIGINT DEFAULT 0 COMMENT '当前进度',
    claimed TINYINT(1) DEFAULT 0 COMMENT '是否已领取奖励',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_player_event (player_id, event_id),
    INDEX idx_player (player_id),
    INDEX idx_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家活动进度';

-- 活动奖励领取记录表
CREATE TABLE IF NOT EXISTS event_reward_records (
    id VARCHAR(64) PRIMARY KEY COMMENT '记录ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    event_id VARCHAR(64) NOT NULL COMMENT '活动ID',
    reward_id VARCHAR(64) DEFAULT '' COMMENT '奖励ID',
    claimed_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '领取时间',
    INDEX idx_player_event (player_id, event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='活动奖励领取记录';

-- ============================================================
-- 战令系统
-- ============================================================

-- 战令赛季表
CREATE TABLE IF NOT EXISTS battle_pass_seasons (
    season_id VARCHAR(64) PRIMARY KEY COMMENT '赛季ID',
    season_name VARCHAR(64) NOT NULL COMMENT '赛季名称',
    start_time DATETIME NOT NULL COMMENT '开始时间',
    end_time DATETIME NOT NULL COMMENT '结束时间',
    premium_cost BIGINT NOT NULL DEFAULT 0 COMMENT '高级战令费用(灵玉)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_time (start_time, end_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='战令赛季';

-- 战令等级定义表
CREATE TABLE IF NOT EXISTS bp_tiers (
    id VARCHAR(64) PRIMARY KEY COMMENT '等级ID',
    season_id VARCHAR(64) NOT NULL COMMENT '赛季ID',
    level INT NOT NULL COMMENT '等级(1-60)',
    exp_required BIGINT NOT NULL COMMENT '升级所需经验',
    is_premium TINYINT(1) DEFAULT 0 COMMENT '是否高级战令奖励',
    reward_item_id BIGINT DEFAULT 0 COMMENT '奖励物品ID',
    reward_name VARCHAR(64) DEFAULT '' COMMENT '奖励名称',
    reward_quantity INT DEFAULT 1 COMMENT '奖励数量',
    reward_type VARCHAR(32) DEFAULT 'item' COMMENT '奖励类型: item/title/outfit/mount/artifact',
    FOREIGN KEY (season_id) REFERENCES battle_pass_seasons(season_id) ON DELETE CASCADE,
    INDEX idx_season_level (season_id, level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='战令等级奖励';

-- 玩家战令进度表
CREATE TABLE IF NOT EXISTS bp_progress (
    id VARCHAR(64) PRIMARY KEY COMMENT '进度ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    season_id VARCHAR(64) NOT NULL COMMENT '赛季ID',
    current_level INT DEFAULT 1 COMMENT '当前等级',
    current_exp BIGINT DEFAULT 0 COMMENT '当前经验',
    has_premium TINYINT(1) DEFAULT 0 COMMENT '是否已购买高级战令',
    claimed_levels VARCHAR(512) DEFAULT '' COMMENT '已领取等级(CSV)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_player_season (player_id, season_id),
    INDEX idx_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家战令进度';

-- 战令奖励领取日志表
CREATE TABLE IF NOT EXISTS bp_reward_claim_log (
    id VARCHAR(64) PRIMARY KEY COMMENT '日志ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    season_id VARCHAR(64) NOT NULL COMMENT '赛季ID',
    level INT NOT NULL COMMENT '领取等级',
    claimed_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '领取时间',
    INDEX idx_player_season (player_id, season_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='战令奖励领取日志';

-- ============================================================
-- 签到增强
-- ============================================================

-- 每月签到明细表
CREATE TABLE IF NOT EXISTS monthly_checkin (
    id VARCHAR(64) PRIMARY KEY COMMENT '记录ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    month_str VARCHAR(7) NOT NULL COMMENT '月份 YYYY-MM',
    checkin_day INT NOT NULL COMMENT '签到日(1-28)',
    is_makeup TINYINT(1) DEFAULT 0 COMMENT '是否补签',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '签到时间',
    UNIQUE KEY uk_player_month_day (player_id, month_str, checkin_day),
    INDEX idx_player_month (player_id, month_str)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='每月签到明细';

-- 每月签到里程碑领取表
CREATE TABLE IF NOT EXISTS monthly_checkin_milestones (
    id VARCHAR(64) PRIMARY KEY COMMENT '记录ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    month_str VARCHAR(7) NOT NULL COMMENT '月份',
    milestone_day INT NOT NULL COMMENT '里程碑日(7/14/21/28)',
    claimed_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '领取时间',
    UNIQUE KEY uk_player_month_milestone (player_id, month_str, milestone_day),
    INDEX idx_player_month (player_id, month_str)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='签到里程碑领取记录';

-- ============================================================
-- 成就系统增强
-- ============================================================

-- 成就定义表
CREATE TABLE IF NOT EXISTS achievements (
    id VARCHAR(64) PRIMARY KEY COMMENT '成就ID',
    category VARCHAR(32) NOT NULL COMMENT '分类: cultivation/combat/social/collection/explore/wealth/activity/hidden',
    name VARCHAR(64) NOT NULL COMMENT '成就名称',
    description VARCHAR(256) DEFAULT '' COMMENT '成就描述',
    is_hidden TINYINT(1) DEFAULT 0 COMMENT '是否隐藏成就',
    hint VARCHAR(128) DEFAULT '' COMMENT '隐藏成就提示',
    icon VARCHAR(32) DEFAULT '' COMMENT '图标',
    sort_order INT DEFAULT 0 COMMENT '排序',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_category (category),
    INDEX idx_sort (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='成就定义';

-- 成就等级表
CREATE TABLE IF NOT EXISTS achievement_tiers (
    id VARCHAR(64) PRIMARY KEY COMMENT '等级ID',
    achievement_id VARCHAR(64) NOT NULL COMMENT '成就ID',
    level INT NOT NULL COMMENT '等级(1-4): 初窥门径/登堂入室/炉火纯青/出神入化',
    name VARCHAR(32) NOT NULL COMMENT '等级名称',
    condition BIGINT NOT NULL COMMENT '达成条件值',
    title_id VARCHAR(64) DEFAULT '' COMMENT '可解锁称号ID',
    reward_exp BIGINT DEFAULT 0 COMMENT '经验奖励',
    reward_money BIGINT DEFAULT 0 COMMENT '灵石奖励',
    FOREIGN KEY (achievement_id) REFERENCES achievements(id) ON DELETE CASCADE,
    INDEX idx_achievement (achievement_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='成就等级配置';

-- 玩家成就进度表
CREATE TABLE IF NOT EXISTS player_achievements_tiers (
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    achievement_id VARCHAR(64) NOT NULL COMMENT '成就ID',
    current_tier INT DEFAULT 0 COMMENT '当前最高等级',
    progress BIGINT DEFAULT 0 COMMENT '当前进度',
    completed TINYINT(1) DEFAULT 0 COMMENT '是否完成',
    claimed_tiers VARCHAR(32) DEFAULT '' COMMENT '已领取等级(CSV)',
    completed_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '完成时间',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (player_id, achievement_id),
    INDEX idx_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家成就等级进度';

-- ============================================================
-- 称号系统增强
-- ============================================================

-- 称号定义表
CREATE TABLE IF NOT EXISTS titles (
    id VARCHAR(64) PRIMARY KEY COMMENT '称号ID',
    name VARCHAR(32) NOT NULL COMMENT '称号名称',
    description VARCHAR(128) DEFAULT '' COMMENT '称号描述',
    color VARCHAR(16) DEFAULT '#FFFFFF' COMMENT '显示颜色',
    source VARCHAR(64) DEFAULT '' COMMENT '获取来源',
    stat_bonus_hp BIGINT DEFAULT 0 COMMENT 'HP加成',
    stat_bonus_attack BIGINT DEFAULT 0 COMMENT '攻击加成',
    stat_bonus_defense BIGINT DEFAULT 0 COMMENT '防御加成',
    stat_bonus_speed DECIMAL(5,2) DEFAULT 0 COMMENT '速度加成',
    rarity INT DEFAULT 1 COMMENT '稀有度(1-5)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_rarity (rarity)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='称号定义';

-- 玩家称号表
CREATE TABLE IF NOT EXISTS player_titles_enhanced (
    id VARCHAR(64) PRIMARY KEY COMMENT '记录ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    title_id VARCHAR(64) NOT NULL COMMENT '称号ID',
    is_equipped TINYINT(1) DEFAULT 0 COMMENT '是否装备',
    obtained_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '获得时间',
    UNIQUE KEY uk_player_title (player_id, title_id),
    INDEX idx_player (player_id),
    INDEX idx_equipped (player_id, is_equipped)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家称号记录';
