-- ===================================================================
-- 修仙游戏 V4 - 渡劫系统
-- 交互式雷劫系统 + 护法机制
-- InnoDB / utf8mb4 / 合理索引
-- ===================================================================

-- 1. 渡劫会话表
CREATE TABLE IF NOT EXISTS tribulation_sessions (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    player_name     VARCHAR(64)     NOT NULL DEFAULT ''                 COMMENT '玩家名称',
    session_id      VARCHAR(64)     NOT NULL                            COMMENT '会话唯一标识',
    tribulation_type TINYINT       NOT NULL DEFAULT 0                   COMMENT '渡劫类型: 0=三九雷劫, 1=六九雷劫, 2=九九雷劫',
    type_name       VARCHAR(32)     NOT NULL DEFAULT ''                 COMMENT '渡劫类型名称',
    total_waves     INT             NOT NULL DEFAULT 0                  COMMENT '总波次',
    current_wave    INT             NOT NULL DEFAULT 1                  COMMENT '当前波次',
    strikes_per_wave INT            NOT NULL DEFAULT 3                  COMMENT '每波雷击数',
    player_hp       BIGINT          NOT NULL DEFAULT 0                  COMMENT '渡劫时玩家HP',
    max_hp          BIGINT          NOT NULL DEFAULT 0                  COMMENT '玩家最大HP',
    damage_taken    BIGINT          NOT NULL DEFAULT 0                  COMMENT '已受伤害',
    status          VARCHAR(16)     NOT NULL DEFAULT 'active'           COMMENT '状态: active/success/failed',
    guardians       JSON            DEFAULT NULL                        COMMENT '护法列表(JSON数组)',
    realm_id        INT             NOT NULL DEFAULT 0                  COMMENT '渡劫时目标大境界ID',
    realm_level     INT             NOT NULL DEFAULT 0                  COMMENT '渡劫时目标小境界等级',
    bonus_data      JSON            DEFAULT NULL                        COMMENT '渡劫成功奖励属性(JSON)',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    finished_at     DATETIME        DEFAULT NULL                        COMMENT '完成/失败时间',

    UNIQUE KEY uk_session_id (session_id),
    INDEX idx_player (player_id),
    INDEX idx_player_status (player_id, status),
    INDEX idx_status (status),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='渡劫会话表(交互式渡劫系统)';

-- 2. 渡劫伤害记录表
CREATE TABLE IF NOT EXISTS tribulation_wave_logs (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    session_id      VARCHAR(64)     NOT NULL                            COMMENT '关联会话ID',
    wave_number     INT             NOT NULL DEFAULT 0                  COMMENT '波次编号',
    action_taken    VARCHAR(16)     NOT NULL DEFAULT ''                 COMMENT '玩家选择: endure/dodge/artifact',
    damage_before   BIGINT          NOT NULL DEFAULT 0                  COMMENT '减免前伤害',
    damage_after    BIGINT          NOT NULL DEFAULT 0                  COMMENT '减免后伤害',
    damage_reduced  BIGINT          NOT NULL DEFAULT 0                  COMMENT '减免的伤害',
    dodged          TINYINT(1)      NOT NULL DEFAULT 0                  COMMENT '是否完全闪避',
    hp_remaining    BIGINT          NOT NULL DEFAULT 0                  COMMENT '剩余HP',
    survived        TINYINT(1)      NOT NULL DEFAULT 0                  COMMENT '是否存活',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_session (session_id),
    INDEX idx_session_wave (session_id, wave_number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='渡劫波次伤害记录';

-- 3. 玩家渡劫统计表
CREATE TABLE IF NOT EXISTS player_tribulation_stats (
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    total_attempts  INT             NOT NULL DEFAULT 0                  COMMENT '总渡劫次数',
    success_count   INT             NOT NULL DEFAULT 0                  COMMENT '成功次数',
    failed_count    INT             NOT NULL DEFAULT 0                  COMMENT '失败次数',
    best_wave       INT             NOT NULL DEFAULT 0                  COMMENT '最高通过波次',
    total_damage_taken BIGINT      NOT NULL DEFAULT 0                  COMMENT '累计承受伤害',
    last_tribulation_time DATETIME  DEFAULT NULL                        COMMENT '最近渡劫时间',

    PRIMARY KEY (player_id),
    INDEX idx_success_rate (success_count, total_attempts)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家渡劫统计';
