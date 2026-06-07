-- ============================================================
-- 宗门副本系统 - 数据表
-- 宗门成员共同挑战首领，按伤害排名发放奖励
-- ============================================================

-- 宗门副本配置表
CREATE TABLE IF NOT EXISTS sect_dungeon_configs (
    id              INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '配置ID',
    name            VARCHAR(64)     NOT NULL                            COMMENT '副本名称',
    realm_required  INT             NOT NULL DEFAULT 0                  COMMENT '宗门要求境界等级',
    boss_hp         BIGINT          NOT NULL                            COMMENT '首领血量',
    boss_atk        INT             NOT NULL DEFAULT 0                  COMMENT '首领攻击力',
    boss_def        INT             NOT NULL DEFAULT 0                  COMMENT '首领防御力',
    boss_skills     JSON            DEFAULT NULL                        COMMENT '首领技能配置',
    max_participants INT            NOT NULL DEFAULT 10                 COMMENT '最大参与人数',
    duration_minutes INT            NOT NULL DEFAULT 30                 COMMENT '副本持续时间(分钟)',
    unlock_cost     INT             NOT NULL DEFAULT 0                  COMMENT '解锁消耗(宗门资金)',
    rewards         JSON            NOT NULL                            COMMENT '奖励配置(按排名区间)',
    created_at      TIMESTAMP       DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_realm_required (realm_required)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门副本配置表';

-- 宗门副本会话表
CREATE TABLE IF NOT EXISTS sect_dungeon_sessions (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '会话ID',
    sect_id         BIGINT          NOT NULL                            COMMENT '宗门ID',
    dungeon_config_id INT           NOT NULL                            COMMENT '副本配置ID',
    status          TINYINT         NOT NULL DEFAULT 0                  COMMENT '状态:0=准备中 1=进行中 2=已完成 3=已失败',
    started_at      DATETIME        DEFAULT NULL                        COMMENT '开始时间',
    ended_at        DATETIME        DEFAULT NULL                        COMMENT '结束时间',
    total_damage    BIGINT          NOT NULL DEFAULT 0                  COMMENT '总伤害',
    created_at      TIMESTAMP       DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_sect_id (sect_id),
    INDEX idx_status (status),
    INDEX idx_sect_status (sect_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门副本会话表';

-- 宗门副本参与者表
CREATE TABLE IF NOT EXISTS sect_dungeon_participants (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '参与记录ID',
    session_id      BIGINT          NOT NULL                            COMMENT '会话ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    damage_dealt    BIGINT          NOT NULL DEFAULT 0                  COMMENT '造成伤害',
    rank            INT             DEFAULT NULL                        COMMENT '排名',
    rewards_claimed TINYINT         NOT NULL DEFAULT 0                  COMMENT '奖励是否已领取 0=未领取 1=已领取',
    created_at      TIMESTAMP       DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_session_id (session_id),
    INDEX idx_player_id (player_id),
    INDEX idx_session_player (session_id, player_id),
    FOREIGN KEY (session_id) REFERENCES sect_dungeon_sessions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门副本参与者表';
