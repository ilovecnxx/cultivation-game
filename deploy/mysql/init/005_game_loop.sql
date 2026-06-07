-- ===================================================================
-- 修仙游戏 - 游戏循环系统表
-- InnoDB / utf8mb4 / 合理索引
-- ===================================================================

-- 1. 玩家 Buff 表
CREATE TABLE IF NOT EXISTS player_buffs (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT 'Buff记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    buff_type       VARCHAR(32)     NOT NULL                            COMMENT 'Buff类型(cultivation_speed/meditation_efficiency/breakthrough_bonus/attack/defense)',
    effect_value    DECIMAL(10,4)   NOT NULL DEFAULT 0.0000             COMMENT '效果数值(倍率/百分比/绝对值)',
    buff_source     VARCHAR(64)     DEFAULT ''                          COMMENT 'Buff来源(pill/artifact/equipment/event)',
    source_id       VARCHAR(64)     DEFAULT NULL                        COMMENT '来源ID(丹药ID/法宝ID/事件ID)',
    start_time      DATETIME        NOT NULL                            COMMENT 'Buff开始时间',
    end_time        DATETIME        DEFAULT NULL                        COMMENT 'Buff结束时间(NULL=永久)',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player (player_id),
    INDEX idx_type (buff_type),
    INDEX idx_end_time (end_time),
    INDEX idx_player_active (player_id, buff_type, end_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家Buff表(丹药/法宝/事件等临时效果)';


-- 2. 世界事件表
CREATE TABLE IF NOT EXISTS world_events (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '事件ID',
    event_type      VARCHAR(32)     NOT NULL                            COMMENT '事件类型(world_boss/treasure_rain/qi_tide/mystic_mist/sect_war)',
    region_id       VARCHAR(64)     DEFAULT NULL                        COMMENT '关联区域ID(NULL=全服)',
    title           VARCHAR(128)    NOT NULL                            COMMENT '事件标题',
    description     TEXT            DEFAULT NULL                        COMMENT '事件描述',
    params          JSON            DEFAULT NULL                        COMMENT '事件参数(如BossID/物品ID/倍率等)',
    start_time      DATETIME        NOT NULL                            COMMENT '事件开始时间',
    end_time        DATETIME        DEFAULT NULL                        COMMENT '事件结束时间',
    status          VARCHAR(16)     DEFAULT 'scheduled'                 COMMENT '状态(scheduled/active/finished/cancelled)',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_type (event_type),
    INDEX idx_status (status),
    INDEX idx_region (region_id),
    INDEX idx_time_range (start_time, end_time),
    INDEX idx_type_status (event_type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='世界事件表(定时触发/随机触发的全服事件)';


-- 3. 日常重置表
CREATE TABLE IF NOT EXISTS daily_resets (
    id                  BIGINT          AUTO_INCREMENT  PRIMARY KEY     COMMENT '记录ID',
    player_id           BIGINT          NOT NULL                        COMMENT '玩家ID',
    reset_date          DATE            NOT NULL                        COMMENT '重置日期(YYYY-MM-DD)',
    cultivation_count   INT             DEFAULT 0                       COMMENT '今日修炼次数',
    quest_count         INT             DEFAULT 0                       COMMENT '今日完成任务数',
    gather_count        INT             DEFAULT 0                       COMMENT '今日采集次数',
    pvp_win_count       INT             DEFAULT 0                       COMMENT '今日PVP胜场',
    pvp_lose_count      INT             DEFAULT 0                       COMMENT '今日PVP负场',
    extra_data          JSON            DEFAULT NULL                    COMMENT '扩展数据(各系统独立计数)',
    created_at          DATETIME        DEFAULT CURRENT_TIMESTAMP       COMMENT '创建时间',
    updated_at          DATETIME        DEFAULT CURRENT_TIMESTAMP
                                        ON UPDATE CURRENT_TIMESTAMP     COMMENT '更新时间',

    UNIQUE KEY uk_player_date (player_id, reset_date),
    INDEX idx_player (player_id),
    INDEX idx_date (reset_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='日常重置表(每日任务/次数限制)';
