-- 010: 成就称号系统
-- 成就分类: cultivation/combat/explore/social/wealth
-- 称号通过完成带称号奖励的成就获得，自动装备

CREATE TABLE IF NOT EXISTS player_achievements (
    player_id        BIGINT       NOT NULL COMMENT '玩家ID',
    achievement_id   INT          NOT NULL COMMENT '成就ID',
    progress         INT          NOT NULL DEFAULT 0 COMMENT '当前进度',
    completed        TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '是否已完成',
    completed_at     DATETIME     NULL COMMENT '完成时间',
    claimed          TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '奖励是否已领取',
    PRIMARY KEY (player_id, achievement_id),
    INDEX idx_player_id (player_id),
    INDEX idx_completed (completed)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家成就记录表';

CREATE TABLE IF NOT EXISTS player_titles (
    player_id    BIGINT       NOT NULL PRIMARY KEY COMMENT '玩家ID',
    title        VARCHAR(32)  NOT NULL DEFAULT '' COMMENT '当前称号',
    updated_at   DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家称号表';
