-- ============================================================
-- 组队副本系统 - 数据表
-- 玩家组队(3-5人)进入独立副本, 依次挑战怪物波次+最终Boss
-- ============================================================

-- 组队副本配置表
CREATE TABLE IF NOT EXISTS team_dungeon_configs (
    id                  INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '配置ID',
    name                VARCHAR(64)     NOT NULL                            COMMENT '副本名称',
    realm_required      INT             NOT NULL DEFAULT 0                  COMMENT '要求境界等级',
    min_players         TINYINT         NOT NULL DEFAULT 3                  COMMENT '最小人数',
    max_players         TINYINT         NOT NULL DEFAULT 5                  COMMENT '最大人数',
    waves               JSON            NOT NULL                            COMMENT '波次配置(每波怪物组+boss)',
    boss_id             INT             DEFAULT NULL                        COMMENT '最终BossID',
    time_limit_minutes  INT             NOT NULL DEFAULT 15                 COMMENT '时间限制(分钟)',
    rewards             JSON            NOT NULL                            COMMENT '奖励配置(通关奖励+特殊目标奖励)',
    created_at          TIMESTAMP       DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_realm_required (realm_required)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组队副本配置表';

-- 组队副本队伍表
CREATE TABLE IF NOT EXISTS dungeon_teams (
    id                  BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '队伍ID',
    dungeon_config_id   INT             NOT NULL                            COMMENT '副本配置ID',
    leader_id           BIGINT          NOT NULL                            COMMENT '队长玩家ID',
    status              TINYINT         NOT NULL DEFAULT 0                  COMMENT '状态:0=招募中 1=已就绪 2=进行中 3=已完成 4=已失败',
    current_wave        INT             NOT NULL DEFAULT 0                  COMMENT '当前波次(0=未开始)',
    started_at          DATETIME        DEFAULT NULL                        COMMENT '开始时间',
    completed_at        DATETIME        DEFAULT NULL                        COMMENT '完成时间',
    total_damage        BIGINT          NOT NULL DEFAULT 0                  COMMENT '全队总伤害',
    time_limit_sec      INT             NOT NULL DEFAULT 900                COMMENT '时间限制(秒)',
    created_at          TIMESTAMP       DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_dungeon_config_id (dungeon_config_id),
    INDEX idx_leader_id (leader_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (dungeon_config_id) REFERENCES team_dungeon_configs(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组队副本队伍表';

-- 组队副本成员表
CREATE TABLE IF NOT EXISTS dungeon_team_members (
    id                  BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '成员记录ID',
    team_id             BIGINT          NOT NULL                            COMMENT '队伍ID',
    player_id           BIGINT          NOT NULL                            COMMENT '玩家ID',
    position            TINYINT         NOT NULL DEFAULT 2                  COMMENT '位置:1=坦克 2=输出 3=辅助',
    ready               TINYINT         NOT NULL DEFAULT 0                  COMMENT '是否就绪:0=未就绪 1=已就绪',
    damage_dealt        BIGINT          NOT NULL DEFAULT 0                  COMMENT '造成伤害',
    healing_done        BIGINT          NOT NULL DEFAULT 0                  COMMENT '治疗量',
    support_provided    BIGINT          NOT NULL DEFAULT 0                  COMMENT '辅助贡献值',
    rewards_claimed     TINYINT         NOT NULL DEFAULT 0                  COMMENT '奖励已领取:0=未领取 1=已领取',
    joined_at           TIMESTAMP       DEFAULT CURRENT_TIMESTAMP           COMMENT '加入时间',

    INDEX idx_team_id (team_id),
    INDEX idx_player_id (player_id),
    INDEX idx_team_player (team_id, player_id),
    FOREIGN KEY (team_id) REFERENCES dungeon_teams(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组队副本成员表';

-- 组队副本邀请表
CREATE TABLE IF NOT EXISTS dungeon_team_invites (
    id                  BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '邀请ID',
    team_id             BIGINT          NOT NULL                            COMMENT '队伍ID',
    inviter_id          BIGINT          NOT NULL                            COMMENT '邀请人玩家ID',
    invitee_id          BIGINT          NOT NULL                            COMMENT '被邀请人玩家ID',
    status              TINYINT         NOT NULL DEFAULT 0                  COMMENT '状态:0=待处理 1=已接受 2=已拒绝 3=已过期',
    created_at          TIMESTAMP       DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_team_id (team_id),
    INDEX idx_invitee_id (invitee_id),
    INDEX idx_status (status),
    FOREIGN KEY (team_id) REFERENCES dungeon_teams(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组队副本邀请表';
