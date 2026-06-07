-- ===================================================================
-- 修仙游戏 V3 - 修炼系统重构
-- 突破节点小游戏 + 道心系统 + 隐藏灵脉
-- InnoDB / utf8mb4 / 合理索引
-- ===================================================================

-- 1. 突破节点小游戏会话表
CREATE TABLE IF NOT EXISTS breakthrough_sessions (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    session_id      VARCHAR(64)     NOT NULL                            COMMENT '会话唯一标识',
    realm_id        INT             NOT NULL DEFAULT 0                  COMMENT '突破時的大境界ID',
    realm_level     INT             NOT NULL DEFAULT 0                  COMMENT '突破時的小境界等级',
    total_nodes     INT             NOT NULL DEFAULT 0                  COMMENT '需要收集的总节点数',
    collected       INT             NOT NULL DEFAULT 0                  COMMENT '已收集节点数',
    time_limit_sec  INT             NOT NULL DEFAULT 0                  COMMENT '总时限(秒)',
    status          VARCHAR(16)     NOT NULL DEFAULT 'active'           COMMENT '状态: active/success/failed',
    pill_buffs      JSON            DEFAULT NULL                        COMMENT '使用的丹药加成(JSON: {time_bonus, range_bonus})',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    finished_at     DATETIME        DEFAULT NULL                        COMMENT '完成/失败时间',

    UNIQUE KEY uk_session_id (session_id),
    INDEX idx_player (player_id),
    INDEX idx_player_status (player_id, status),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='突破节点小游戏会话表';

-- 2. 道心表
CREATE TABLE IF NOT EXISTS dao_xin (
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    stacks          TINYINT         NOT NULL DEFAULT 0                  COMMENT '道心层数(0~3)',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    PRIMARY KEY (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='道心系统表(突破失败累积,每层节点生成速度-5%)';

-- 3. 隐藏灵脉表
CREATE TABLE IF NOT EXISTS hidden_spirit_veins (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    vein_id         VARCHAR(64)     NOT NULL                            COMMENT '灵脉唯一标识',
    region_id       VARCHAR(64)     NOT NULL                            COMMENT '所在区域ID',
    density         DECIMAL(4,2)    NOT NULL DEFAULT 0.00               COMMENT '灵气浓度加成值',
    owner_id        BIGINT          NOT NULL DEFAULT 0                  COMMENT '发现者玩家ID(0=公共)',
    vein_type       VARCHAR(16)     NOT NULL DEFAULT 'hidden'           COMMENT '类型: hidden(隐藏)/public(公共)',
    expires_at      DATETIME        NOT NULL                            COMMENT '过期时间',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    UNIQUE KEY uk_vein_id (vein_id),
    INDEX idx_region (region_id),
    INDEX idx_owner (owner_id),
    INDEX idx_expires (expires_at),
    INDEX idx_type_expires (vein_type, expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='隐藏灵脉表(临时灵气加成,隐藏持续2h,公共有数量上限)';

-- 4. 修改玩家表新增字段（如果采用关系型玩家表）
-- 注意：如果 Player 数据在 MongoDB 等 NoSQL 中，以下仅作参考
-- ALTER TABLE players ADD COLUMN spirit_density DECIMAL(4,2) DEFAULT 0.00 COMMENT '当前区域灵气浓度';
-- ALTER TABLE players ADD COLUMN dao_xin_stacks TINYINT DEFAULT 0 COMMENT '道心层数';
