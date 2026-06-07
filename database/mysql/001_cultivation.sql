-- ===================================================================
-- 修仙游戏 - 修炼系统持久化表
-- InnoDB / utf8mb4 / 合理索引
-- ===================================================================

-- 1. 修炼玩家表
CREATE TABLE IF NOT EXISTS cultivation_players (
    player_id        BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '玩家ID(自增)',
    nickname         VARCHAR(64)     NOT NULL                            COMMENT '角色名',
    realm_id         INT             DEFAULT 1                           COMMENT '当前大境界ID',
    realm_level      INT             DEFAULT 1                           COMMENT '当前小境界等级',
    exp              BIGINT          DEFAULT 0                           COMMENT '当前修为值',
    spirit_root      JSON            DEFAULT NULL                        COMMENT '灵根属性(JSON, 如{"金":0.4,"木":0.6})',
    base_attack      BIGINT          DEFAULT 0                           COMMENT '基础攻击力',
    base_defense     BIGINT          DEFAULT 0                           COMMENT '基础防御力',
    base_hp          BIGINT          DEFAULT 0                           COMMENT '基础生命值',
    technique_id     INT             DEFAULT 0                           COMMENT '已装备功法ID',
    technique_level  INT             DEFAULT 0                           COMMENT '功法等级',
    is_meditating    TINYINT(1)      DEFAULT 0                           COMMENT '是否在闭关中',
    meditation_start BIGINT          DEFAULT 0                           COMMENT '闭关开始时间(unix时间戳)',
    accumulated_exp  BIGINT          DEFAULT 0                           COMMENT '闭关期间累计修为',
    pill_bonuses     JSON            DEFAULT NULL                        COMMENT '丹药加成(JSON map)',
    artifact_bonuses JSON            DEFAULT NULL                        COMMENT '法宝加成(JSON map)',
    created_at       DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at       DATETIME        DEFAULT CURRENT_TIMESTAMP
                                     ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_realm (realm_id, realm_level),
    INDEX idx_nickname (nickname)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='修炼系统玩家数据表';

-- 2. 玩家功法表
CREATE TABLE IF NOT EXISTS cultivation_techniques (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    technique_id    INT             NOT NULL                            COMMENT '功法ID',
    level           INT             DEFAULT 1                           COMMENT '功法等级',
    is_equipped     TINYINT(1)      DEFAULT 0                           COMMENT '是否已装备(1主修/0未装备)',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player (player_id),
    UNIQUE KEY uk_player_technique (player_id, technique_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='修炼系统玩家功法表';
