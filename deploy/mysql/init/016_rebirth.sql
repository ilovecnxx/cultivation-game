-- ============================================================
-- 轮回重生系统 (化神期解锁 / 飞升后使用)
-- ============================================================

-- 轮回记录表
CREATE TABLE IF NOT EXISTS rebirth_records (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '轮回ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    rebirth_count   INT             DEFAULT 1                           COMMENT '第几次轮回',
    prev_level      INT             NOT NULL                            COMMENT '轮回前等级',
    prev_realm      VARCHAR(32)     NOT NULL                            COMMENT '轮回前境界',
    prev_power      BIGINT          DEFAULT 0                           COMMENT '轮回前战力',
    new_level       INT             DEFAULT 1                           COMMENT '轮回后等级(初始等级)',
    new_realm       VARCHAR(32)     DEFAULT '练气'                      COMMENT '轮回后境界',
    talent_bonus    FLOAT           DEFAULT 0.0                         COMMENT '获得的天赋加成(0~0.5)',
    kept_items      JSON            DEFAULT NULL                        COMMENT '保留的物品列表(JSON数组)',
    kept_skills     JSON            DEFAULT NULL                        COMMENT '保留的技能列表(JSON数组)',
    status          VARCHAR(16)     DEFAULT 'completed'                 COMMENT '状态: completed/pending_review',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '轮回时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player_id (player_id),
    INDEX idx_rebirth_count (player_id, rebirth_count),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='轮回记录表';

-- 轮回天赋表
CREATE TABLE IF NOT EXISTS rebirth_talents (
    id              INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '天赋ID',
    name            VARCHAR(32)     NOT NULL                            COMMENT '天赋名称',
    description     TEXT                                                COMMENT '天赋描述',
    type            VARCHAR(16)     NOT NULL                            COMMENT '类型: combat/growth/fortune/special',
    effect_type     VARCHAR(32)     NOT NULL                            COMMENT '效果类型: attack_bonus/exp_bonus/drop_bonus/cultivation_speed',
    effect_value    FLOAT           NOT NULL                            COMMENT '效果数值(百分比)',
    min_rebirth     INT             DEFAULT 1                           COMMENT '最少轮回次数解锁',
    max_level       INT             DEFAULT 5                           COMMENT '天赋最大等级',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_type (type),
    INDEX idx_min_rebirth (min_rebirth)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='轮回天赋表';

-- 玩家已获取的轮回天赋
CREATE TABLE IF NOT EXISTS player_rebirth_talents (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    talent_id       INT             NOT NULL                            COMMENT '天赋ID',
    level           INT             DEFAULT 1                           COMMENT '当前等级',
    source_rebirth  INT             NOT NULL                            COMMENT '获取时第几次轮回',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '获取时间',

    UNIQUE KEY uk_player_talent (player_id, talent_id),
    INDEX idx_player_id (player_id),
    INDEX idx_talent_id (talent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家轮回天赋表';

-- 轮回商店兑换表(用轮回积分换物品)
CREATE TABLE IF NOT EXISTS rebirth_shop_items (
    id              INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '商品ID',
    item_id         INT             NOT NULL                            COMMENT '兑换物品ID',
    item_name       VARCHAR(32)     NOT NULL                            COMMENT '物品名称',
    item_type       VARCHAR(16)     DEFAULT 'item'                      COMMENT '物品类型: item/skill/title/pet',
    cost_rebirth_points INT         NOT NULL                            COMMENT '消耗轮回积分',
    min_rebirth     INT             DEFAULT 1                           COMMENT '最少轮回次数要求',
    stock           INT             DEFAULT -1                          COMMENT '库存(-1=不限)',
    refresh_type    VARCHAR(16)     DEFAULT 'none'                      COMMENT '刷新类型: none/daily/weekly/monthly',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_refresh_type (refresh_type),
    INDEX idx_min_rebirth (min_rebirth)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='轮回商店兑换表';

-- ============================================================
-- 示例数据
-- ============================================================

INSERT INTO rebirth_talents (id, name, description, type, effect_type, effect_value, min_rebirth, max_level) VALUES
(1, '转世灵童',    '每次修炼额外获得经验',          'growth', 'exp_bonus',          0.10, 1, 5),
(2, '战斗本能',    '永久提升攻击力',                'combat', 'attack_bonus',       0.05, 1, 5),
(3, '铁骨铮铮',    '永久提升防御力',                'combat', 'defense_bonus',      0.05, 1, 5),
(4, '福缘深厚',    '提升掉落率',                    'fortune','drop_bonus',         0.08, 2, 3),
(5, '天资聪颖',    '提升修炼速度',                  'growth', 'cultivation_speed',  0.10, 3, 5),
(6, '百战之躯',    '永久提升生命值',                'combat', 'hp_bonus',           0.08, 2, 5);

INSERT INTO rebirth_records (id, player_id, rebirth_count, prev_level, prev_realm, prev_power, new_level, new_realm, talent_bonus, kept_items, kept_skills, status) VALUES
(1, 1, 1, 80, '化神',    150000, 1,  '练气', 0.05, '[101,102,103]',          '[1,5,10]',   'completed'),
(2, 1, 2, 85, '化神后期',180000, 1,  '练气', 0.10, '[101,102,103,201]',      '[1,5,10,12]', 'completed'),
(3, 2, 1, 75, '元婴',    90000,  1,  '练气', 0.03, '[101,102]',               '[3,7]',       'completed');

INSERT INTO rebirth_shop_items (id, item_id, item_name, item_type, cost_rebirth_points, min_rebirth, stock, refresh_type) VALUES
(1, 501, '轮回·先天灵根',   'item',   500,  1, 1,  'none'),
(2, 502, '轮回·星辰仙衣',   'item',   300,  1, 3,  'none'),
(3, 503, '天赋重置令',       'item',   200,  2, -1, 'weekly'),
(4, 201, '轮回称号·重生者', 'title',  1000, 1, 1,  'none'),
(5, 301, '绝技·轮回斩',     'skill',  800,  2, 1,  'none');

INSERT INTO player_rebirth_talents (id, player_id, talent_id, level, source_rebirth, created_at) VALUES
(1, 1, 1, 3, 1, '2026-05-01 10:00:00'),
(2, 1, 2, 2, 1, '2026-05-01 10:00:00'),
(3, 1, 3, 1, 2, '2026-05-20 10:00:00');
