-- ===================================================================
-- 修仙游戏 - 气运系统表
-- InnoDB / utf8mb4 / 合理索引
-- ===================================================================

-- 1. 玩家气运表
CREATE TABLE IF NOT EXISTS player_luck (
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    luck_value      DECIMAL(12,4)   NOT NULL DEFAULT 0.0000             COMMENT '当前气运值(可负值,影响随机事件品质)',
    total_earned    DECIMAL(14,4)   NOT NULL DEFAULT 0.0000             COMMENT '累计获得气运(仅正向累加)',
    total_spent     DECIMAL(14,4)   NOT NULL DEFAULT 0.0000             COMMENT '累计消耗气运(负向累加,记录绝对值)',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    PRIMARY KEY (player_id),
    INDEX idx_luck_value (luck_value),
    INDEX idx_total_earned (total_earned)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家气运表(影响奇遇/突破/掉落等随机判定)';


-- 2. 气运事件记录表
CREATE TABLE IF NOT EXISTS luck_events (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '事件ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    event_type      VARCHAR(32)     NOT NULL                            COMMENT '事件类型(breakthrough/treasure/combat/gather/meditate/quest/penalty/other)',
    amount          DECIMAL(12,4)   NOT NULL                            COMMENT '气运变化量(正=获得,负=消耗)',
    description     VARCHAR(255)    DEFAULT ''                          COMMENT '事件描述',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_player (player_id),
    INDEX idx_type (event_type),
    INDEX idx_created (created_at),
    INDEX idx_player_time (player_id, created_at),
    INDEX idx_player_type (player_id, event_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='气运事件记录表(每次气运变动均记录留痕)';
