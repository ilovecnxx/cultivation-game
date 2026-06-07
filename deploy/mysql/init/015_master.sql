-- ============================================================
-- 师徒系统 (元婴期解锁)
-- ============================================================

-- 师徒关系表
CREATE TABLE IF NOT EXISTS master_relations (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '关系ID',
    master_id       BIGINT          NOT NULL                            COMMENT '师父玩家ID',
    apprentice_id   BIGINT          NOT NULL                            COMMENT '徒弟玩家ID',
    started_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '拜师时间',
    graduated       TINYINT(1)      DEFAULT 0                           COMMENT '是否已出师(0=未出师,1=已出师)',
    graduated_at    DATETIME        DEFAULT NULL                        COMMENT '出师时间',
    status          VARCHAR(16)     DEFAULT 'active'                    COMMENT '状态: active/graduated/expelled/left',
    master_reputation INT           DEFAULT 0                           COMMENT '师父获得的名望值',
    apprentice_gift  TINYINT(1)     DEFAULT 0                           COMMENT '出师时徒弟是否已领取奖励',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    UNIQUE KEY uk_master_apprentice (master_id, apprentice_id),
    INDEX idx_master_id (master_id),
    INDEX idx_apprentice_id (apprentice_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='师徒关系表';

-- 师父配置表(每个玩家作为师父时的配置)
CREATE TABLE IF NOT EXISTS master_configs (
    player_id       BIGINT          NOT NULL PRIMARY KEY                COMMENT '玩家ID',
    open            TINYINT(1)      DEFAULT 1                           COMMENT '是否开放收徒(0=关闭,1=开放)',
    max_apprentice  INT             DEFAULT 3                           COMMENT '最大徒弟数量',
    requirement     JSON            DEFAULT NULL                        COMMENT '收徒要求{min_level,min_realm,...}',
    description     TEXT                                                COMMENT '收徒宣言',
    reputation      INT             DEFAULT 0                           COMMENT '累积名望值',
    total_graduates INT             DEFAULT 0                           COMMENT '累计出师人数',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_open (open)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='师父配置表';

-- 拜师申请表
CREATE TABLE IF NOT EXISTS master_applications (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '申请ID',
    apprentice_id   BIGINT          NOT NULL                            COMMENT '申请方(徒弟)玩家ID',
    master_id       BIGINT          NOT NULL                            COMMENT '被申请方(师父)玩家ID',
    message         TEXT                                                COMMENT '申请留言',
    status          VARCHAR(16)     DEFAULT 'pending'                   COMMENT '状态: pending/accepted/rejected/canceled',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '申请时间',
    handled_at      DATETIME        DEFAULT NULL                        COMMENT '处理时间',

    INDEX idx_apprentice_id (apprentice_id),
    INDEX idx_master_id (master_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拜师申请表';

-- 师徒任务表(徒弟突破任务)
CREATE TABLE IF NOT EXISTS master_tasks (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '任务ID',
    relation_id     BIGINT          NOT NULL                            COMMENT '师徒关系ID',
    task_type       VARCHAR(32)     NOT NULL                            COMMENT '任务类型: level_up/realm_break/dungeon_clear/quest_complete',
    target_value    INT             NOT NULL                            COMMENT '目标值',
    current_value   INT             DEFAULT 0                           COMMENT '当前进度',
    reward_claim    TINYINT(1)      DEFAULT 0                           COMMENT '师父是否已领取奖励',
    completed       TINYINT(1)      DEFAULT 0                           COMMENT '是否完成',
    completed_at    DATETIME        DEFAULT NULL                        COMMENT '完成时间',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_relation_id (relation_id),
    INDEX idx_task_type (task_type),
    INDEX idx_completed (completed)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='师徒任务表';

-- ============================================================
-- 示例数据
-- ============================================================

INSERT INTO master_configs (player_id, open, max_apprentice, requirement, description, reputation, total_graduates) VALUES
(1, 1, 5, '{"min_level":10,"min_realm":"筑基"}',  '传道授业，共登仙途',  1200, 3),
(2, 1, 3, '{"min_level":5,"min_realm":"炼气"}',   '耐心指导，包教包会',  500,  1);

INSERT INTO master_relations (id, master_id, apprentice_id, started_at, graduated, graduated_at, status, master_reputation, apprentice_gift) VALUES
(1, 1, 3, '2026-04-01 10:00:00', 1, '2026-05-15 14:30:00', 'graduated', 500, 1),
(2, 1, 4, '2026-05-20 09:00:00', 0, NULL,                 'active',    200, 0),
(3, 2, 5, '2026-06-01 11:00:00', 0, NULL,                 'active',    100, 0);

INSERT INTO master_applications (id, apprentice_id, master_id, message, status, created_at, handled_at) VALUES
(1, 6, 1, '久仰大名，恳请前辈收我为徒！',    'accepted', '2026-05-19 08:00:00', '2026-05-19 20:00:00'),
(2, 7, 1, '想跟您学习炼丹之术',               'pending',  '2026-06-04 15:00:00', NULL);

INSERT INTO master_tasks (id, relation_id, task_type, target_value, current_value, reward_claim, completed, completed_at) VALUES
(1, 1, 'level_up',        20, 20, 1, 1, '2026-05-10 12:00:00'),
(2, 1, 'realm_break',     1,  1,  1, 1, '2026-05-15 14:30:00'),
(3, 1, 'dungeon_clear',   5,  5,  1, 1, '2026-05-13 16:00:00'),
(4, 2, 'level_up',        15, 12, 0, 0, NULL);
