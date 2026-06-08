-- ===================================================================
-- 装备系统 — 10境界 × 5品阶 × 6部位
-- ===================================================================

CREATE TABLE IF NOT EXISTS player_equipment (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY           COMMENT '装备唯一ID',
    player_id       BIGINT NOT NULL                             COMMENT '所属玩家ID',
    slot            VARCHAR(16) NOT NULL                        COMMENT '部位: weapon/robe/headgear/boots/necklace/ring',
    name            VARCHAR(64) NOT NULL                        COMMENT '装备名称(如 玄·寒铁甲)',
    realm           INT NOT NULL DEFAULT 1                      COMMENT '装备境界(1-10)',
    tier            VARCHAR(16) NOT NULL DEFAULT 'human'        COMMENT '品阶: human/yellow/dark/earth/heaven',
    tier_mult       DOUBLE NOT NULL DEFAULT 1.0                 COMMENT '品阶倍率(0.5/1.0/1.8/3.0/5.0)',
    element         VARCHAR(8) DEFAULT ''                       COMMENT '五行: 金/木/水/火/土',

    -- 主属性
    attack          INT DEFAULT 0                               COMMENT '攻击力',
    defense         INT DEFAULT 0                               COMMENT '防御力',
    hp              INT DEFAULT 0                               COMMENT '生命值',
    mp              INT DEFAULT 0                               COMMENT '灵力值',
    speed           INT DEFAULT 0                               COMMENT '速度',
    crit_rate       INT DEFAULT 0                               COMMENT '暴击率',
    crit_dmg        INT DEFAULT 0                               COMMENT '暴击伤害',
    dodge           INT DEFAULT 0                               COMMENT '闪避率',
    hit             INT DEFAULT 0                               COMMENT '命中率',
    mp_regen        INT DEFAULT 0                               COMMENT '回蓝效率',

    -- 词缀
    substats        JSON DEFAULT NULL                           COMMENT '随机词缀(JSON数组)',

    -- 装备状态
    is_equipped     TINYINT DEFAULT 0                           COMMENT '是否已装备(1=是)',
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP          COMMENT '获得时间',
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_player_slot (player_id, slot),
    INDEX idx_player_equipped (player_id, is_equipped),
    INDEX idx_realm_tier (realm, tier)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家装备表';
