-- 010: 心魔系统持久化表
-- 五大心魔（贪嗔痴疑慢）持久化存储
-- 突破失败/渡劫失败时生成，可通过幻境挑战或道具压制

CREATE TABLE IF NOT EXISTS heart_demons (
    id                VARCHAR(64)  PRIMARY KEY COMMENT '心魔唯一ID',
    player_id         BIGINT       NOT NULL COMMENT '玩家ID',
    demon_type        VARCHAR(16)  NOT NULL COMMENT '心魔类型: greed/wrath/ignor/doubt/sloth',
    level             INT          DEFAULT 1 COMMENT '心魔等级 1-10',
    debuff_value      DECIMAL(4,2) DEFAULT 0.05 COMMENT 'debuff百分比 0.05~0.50',
    created_at        BIGINT       NOT NULL COMMENT '创建时间戳',
    created_from      VARCHAR(32)  DEFAULT '' COMMENT '来源: breakthrough_failure/tribulation_failure/curse',
    defeated          TINYINT(1)   DEFAULT 0 COMMENT '是否已被净化',
    defeated_at       BIGINT       DEFAULT 0 COMMENT '净化时间戳',
    INDEX idx_player (player_id),
    INDEX idx_player_type (player_id, demon_type),
    INDEX idx_active (player_id, defeated)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='心魔持久化表';

CREATE TABLE IF NOT EXISTS demon_illusion_records (
    id                VARCHAR(64)  PRIMARY KEY COMMENT '记录ID',
    player_id         BIGINT       NOT NULL COMMENT '玩家ID',
    demon_type        VARCHAR(16)  NOT NULL COMMENT '心魔类型',
    challenged_at     BIGINT       NOT NULL COMMENT '挑战时间戳',
    won               TINYINT(1)   DEFAULT 0 COMMENT '是否胜利',
    level_before      INT          DEFAULT 0 COMMENT '挑战前等级',
    level_after       INT          DEFAULT 0 COMMENT '挑战后等级',
    INDEX idx_player (player_id),
    INDEX idx_player_time (player_id, challenged_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='心魔幻境挑战记录表';

-- 玩家心魔统计视图（各类型活跃心魔数量）
CREATE OR REPLACE VIEW v_player_active_demons AS
SELECT
    player_id,
    demon_type,
    COUNT(*) AS active_count,
    SUM(level) AS total_level,
    MAX(level) AS max_level
FROM heart_demons
WHERE defeated = 0
GROUP BY player_id, demon_type;
