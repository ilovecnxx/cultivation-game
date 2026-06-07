-- ===================================================================
-- 灵鱼垂钓系统 - 数据表
-- 版本: v1.0.0
-- 修仙者可在各地灵池/仙湖垂钓，捕获灵鱼仙鱼
-- ===================================================================

-- 1. 钓鱼点配置表
CREATE TABLE fishing_spots (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键',
    region_id       VARCHAR(64) NOT NULL             COMMENT '所属区域ID',
    name            VARCHAR(64) NOT NULL             COMMENT '钓鱼点名称',
    description     VARCHAR(512) DEFAULT ''          COMMENT '钓鱼点描述',
    min_realm       INT NOT NULL DEFAULT 1           COMMENT '最低修为等级要求',
    max_realm       INT NOT NULL DEFAULT 100         COMMENT '最高修为等级要求',
    fish_ids        JSON NOT NULL                    COMMENT '可钓到的鱼ID列表JSON数组',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_region_id (region_id),
    INDEX idx_realm_range (min_realm, max_realm)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='钓鱼点配置表';

-- 2. 鱼类配置表
CREATE TABLE fish_types (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键',
    name            VARCHAR(64) NOT NULL             COMMENT '鱼类名称',
    rarity          TINYINT NOT NULL DEFAULT 1       COMMENT '稀有度(1-5, 1普通/2稀有/3珍稀/4仙品/5神品)',
    min_realm       INT NOT NULL DEFAULT 1           COMMENT '最低修为等级要求',
    base_weight     DECIMAL(6,2) NOT NULL DEFAULT 1.00 COMMENT '基础重量(斤)',
    exp_reward      INT NOT NULL DEFAULT 10          COMMENT '钓鱼经验奖励',
    spirit_stone_value INT NOT NULL DEFAULT 0        COMMENT '灵石价值',
    description     VARCHAR(256) DEFAULT ''          COMMENT '鱼类描述',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    INDEX idx_rarity (rarity),
    INDEX idx_min_realm (min_realm)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='鱼类配置表';

-- 3. 玩家钓鱼信息表
CREATE TABLE player_fishing (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键',
    player_id       BIGINT NOT NULL                 COMMENT '玩家ID',
    fishing_skill_level INT NOT NULL DEFAULT 1      COMMENT '钓鱼技能等级',
    fishing_exp     INT NOT NULL DEFAULT 0           COMMENT '钓鱼经验',
    total_caught    INT NOT NULL DEFAULT 0           COMMENT '总捕获数',
    best_catch_id   BIGINT DEFAULT NULL              COMMENT '最佳捕获鱼类ID',
    best_catch_weight DECIMAL(8,2) DEFAULT 0.00      COMMENT '最佳捕获重量',
    bait_count      INT NOT NULL DEFAULT 10          COMMENT '鱼饵数量',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    UNIQUE INDEX idx_player_id (player_id),
    INDEX idx_skill_level (fishing_skill_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家钓鱼信息表';

-- 4. 钓鱼记录表
CREATE TABLE fishing_records (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键',
    player_id       BIGINT NOT NULL                 COMMENT '玩家ID',
    fish_id         BIGINT NOT NULL                 COMMENT '钓到的鱼类ID',
    spot_id         BIGINT NOT NULL                 COMMENT '钓鱼点ID',
    weight          DECIMAL(6,2) NOT NULL           COMMENT '鱼重量(斤)',
    exp_gained      INT NOT NULL DEFAULT 0          COMMENT '获得经验',
    caught_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '捕获时间',

    INDEX idx_player_id (player_id),
    INDEX idx_fish_id (fish_id),
    INDEX idx_spot_id (spot_id),
    INDEX idx_caught_at (caught_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='钓鱼记录表';
