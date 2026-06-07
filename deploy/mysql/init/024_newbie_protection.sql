-- ============================================================
-- 新手保护系统 - 数据表
-- 为新玩家提供时间限制和次数限制的保护机制
-- ============================================================

-- 玩家保护记录表
CREATE TABLE IF NOT EXISTS player_protection (
    id                      BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键',
    player_id               BIGINT NOT NULL                  COMMENT '玩家ID，关联players.id',
    protection_until        TIMESTAMP DEFAULT NULL            COMMENT '通用保护截止时间（绝对时间）',
    pvp_protection_until    TIMESTAMP DEFAULT NULL            COMMENT 'PVP保护截止时间（绝对时间）',
    breakthrough_grace_count INT NOT NULL DEFAULT 3           COMMENT '突破免罚剩余次数',
    free_resurrection_count INT NOT NULL DEFAULT 5            COMMENT '免费复活剩余次数',
    created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                            ON UPDATE CURRENT_TIMESTAMP       COMMENT '更新时间',

    UNIQUE KEY uk_player_id (player_id) COMMENT '一个玩家只有一条保护记录',
    INDEX idx_player_id (player_id),
    INDEX idx_protection_until (protection_until),
    INDEX idx_pvp_protection_until (pvp_protection_until)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家新手保护记录表';
