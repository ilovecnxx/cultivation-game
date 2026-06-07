-- ============================================================
-- 每日任务系统 - 数据表
-- 玩家每日任务进度追踪，活跃度积分与宝箱奖励系统
-- ============================================================

-- 每日任务进度表
CREATE TABLE IF NOT EXISTS daily_task_progress (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    task_date       DATE            NOT NULL                            COMMENT '任务日期',
    task_id         VARCHAR(64)     NOT NULL                            COMMENT '任务ID',
    task_type       VARCHAR(32)     NOT NULL                            COMMENT '任务类型(如 cultivate_time/kill_monster)',
    current_count   INT             NOT NULL DEFAULT 0                  COMMENT '当前进度',
    required_count  INT             NOT NULL DEFAULT 1                  COMMENT '目标数量',
    status          TINYINT         NOT NULL DEFAULT 0                  COMMENT '状态:0=进行中 1=已完成 2=已领取',
    completed_at    TIMESTAMP       NULL        DEFAULT NULL            COMMENT '完成时间',
    claimed_at      TIMESTAMP       NULL        DEFAULT NULL            COMMENT '领取时间',
    created_at      TIMESTAMP       DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      TIMESTAMP       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_player_id (player_id),
    INDEX idx_task_date (task_date),
    INDEX idx_player_date (player_id, task_date),
    INDEX idx_player_date_status (player_id, task_date, status),
    UNIQUE KEY uk_player_date_task (player_id, task_date, task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='每日任务进度表';

-- 每日活跃度积分表
CREATE TABLE IF NOT EXISTS daily_activity_points (
    id                  BIGINT          AUTO_INCREMENT  PRIMARY KEY     COMMENT '记录ID',
    player_id           BIGINT          NOT NULL                        COMMENT '玩家ID',
    date                DATE            NOT NULL                        COMMENT '日期',
    total_points        INT             NOT NULL DEFAULT 0              COMMENT '当前活跃度总分',
    chest_25_claimed    TINYINT         NOT NULL DEFAULT 0              COMMENT '25分宝箱是否已领取:0=未领 1=已领',
    chest_50_claimed    TINYINT         NOT NULL DEFAULT 0              COMMENT '50分宝箱是否已领取:0=未领 1=已领',
    chest_75_claimed    TINYINT         NOT NULL DEFAULT 0              COMMENT '75分宝箱是否已领取:0=未领 1=已领',
    chest_100_claimed   TINYINT         NOT NULL DEFAULT 0              COMMENT '100分宝箱是否已领取:0=未领 1=已领',
    created_at          TIMESTAMP       DEFAULT CURRENT_TIMESTAMP       COMMENT '创建时间',
    updated_at          TIMESTAMP       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_player_id (player_id),
    UNIQUE KEY uk_player_date (player_id, date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='每日活跃度积分表';
