-- ============================================================
-- 秘境副本系统
-- ============================================================

-- 秘境副本记录表
CREATE TABLE IF NOT EXISTS dungeon_records (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    dungeon_id      INT             NOT NULL                            COMMENT '秘境ID(1=青竹/2=玄黄/3=九幽/4=天火/5=混沌)',
    current_floor   INT             DEFAULT 1                           COMMENT '当前所在层',
    max_floor       INT             DEFAULT 0                           COMMENT '历史最高通关层数',
    status          VARCHAR(16)     DEFAULT 'entered'                   COMMENT '状态: entered/cleared/claimed/abandoned',
    team_members    JSON            DEFAULT NULL                        COMMENT '组队成员ID列表(JSON数组)',
    daily_attempts  INT             DEFAULT 0                           COMMENT '今日挑战次数',
    attempt_date    DATE            DEFAULT NULL                        COMMENT '最近挑战日期',
    entered_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '进入秘境时间',
    completed_at    DATETIME        DEFAULT NULL                        COMMENT '通关时间',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player_id (player_id),
    INDEX idx_dungeon_id (dungeon_id),
    INDEX idx_status (status),
    INDEX idx_attempt_date (attempt_date),
    INDEX idx_player_dungeon (player_id, dungeon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='秘境副本记录表';

-- 秘境每日挑战计数表
CREATE TABLE IF NOT EXISTS dungeon_daily_attempts (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    dungeon_id      INT             NOT NULL                            COMMENT '秘境ID',
    attempt_date    DATE            NOT NULL                            COMMENT '挑战日期',
    attempt_count   INT             DEFAULT 0                           COMMENT '今日挑战次数',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    UNIQUE KEY uk_player_dungeon_date (player_id, dungeon_id, attempt_date),
    INDEX idx_player_id (player_id),
    INDEX idx_date (attempt_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='秘境每日挑战计数表(用于快速查询次数限制)';

-- 秘境组队邀请表
CREATE TABLE IF NOT EXISTS dungeon_team_invites (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '邀请ID',
    dungeon_id      INT             NOT NULL                            COMMENT '秘境ID',
    inviter_id      BIGINT          NOT NULL                            COMMENT '邀请方玩家ID',
    invitee_id      BIGINT          NOT NULL                            COMMENT '被邀请方玩家ID',
    status          VARCHAR(16)     DEFAULT 'pending'                   COMMENT '状态: pending/accepted/rejected/expired',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    responded_at    DATETIME        DEFAULT NULL                        COMMENT '响应时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_inviter (inviter_id),
    INDEX idx_invitee (invitee_id),
    INDEX idx_status (status),
    INDEX idx_dungeon (dungeon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='秘境组队邀请表';
