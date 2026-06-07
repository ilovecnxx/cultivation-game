-- ============================================================
-- 商城系统
-- ============================================================

-- 商品模板表
CREATE TABLE IF NOT EXISTS shop_items (
    id              INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '商品ID',
    name            VARCHAR(64)     NOT NULL                            COMMENT '商品名称',
    description     TEXT                                                COMMENT '商品描述',
    item_type       VARCHAR(32)     NOT NULL                            COMMENT '物品类型: item/equip/pet_egg/skill/scroll/title',
    item_ref_id     INT             NOT NULL                            COMMENT '对应模板ID(道具/装备/技能等ID)',
    currency_type   VARCHAR(16)     NOT NULL DEFAULT 'coin'             COMMENT '货币类型: coin/spirit_stone/bound_stone/contribution/rebirth_point',
    price           BIGINT          NOT NULL                            COMMENT '价格',
    discount        FLOAT           DEFAULT 1.0                         COMMENT '折扣率(0.1~1.0)',
    stock           INT             DEFAULT -1                          COMMENT '库存(-1=不限)',
    limit_type      VARCHAR(16)     DEFAULT 'none'                      COMMENT '限购类型: none/daily/weekly/monthly/permanent',
    limit_count     INT             DEFAULT 0                           COMMENT '限购数量(0=不限)',
    required_level  INT             DEFAULT 1                           COMMENT '购买所需等级',
    required_realm  VARCHAR(32)     DEFAULT NULL                        COMMENT '购买所需境界',
    tab             VARCHAR(16)     DEFAULT 'general'                   COMMENT '分页: general/limited/gift/vip/rebirth',
    sort_order      INT             DEFAULT 0                           COMMENT '排序权重(越小越靠前)',
    active          TINYINT(1)      DEFAULT 1                           COMMENT '是否上架',
    start_time      DATETIME        DEFAULT NULL                        COMMENT '上架时间',
    end_time        DATETIME        DEFAULT NULL                        COMMENT '下架时间',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_tab (tab),
    INDEX idx_active (active),
    INDEX idx_currency_type (currency_type),
    INDEX idx_limit_type (limit_type),
    INDEX idx_required_level (required_level),
    INDEX idx_required_realm (required_realm)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品模板表';

-- 玩家购买记录表
CREATE TABLE IF NOT EXISTS player_purchase_records (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    shop_item_id    INT             NOT NULL                            COMMENT '商品ID',
    quantity        INT             DEFAULT 1                           COMMENT '购买数量',
    unit_price      BIGINT          NOT NULL                            COMMENT '实际单价',
    total_price     BIGINT          NOT NULL                            COMMENT '总价',
    currency_type   VARCHAR(16)     NOT NULL                            COMMENT '支付货币类型',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '购买时间',

    INDEX idx_player_id (player_id),
    INDEX idx_shop_item_id (shop_item_id),
    INDEX idx_created_at (created_at),
    INDEX idx_player_item (player_id, shop_item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家购买记录表';

-- 限购消费统计表
CREATE TABLE IF NOT EXISTS player_purchase_limits (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    shop_item_id    INT             NOT NULL                            COMMENT '商品ID',
    period_type     VARCHAR(16)     NOT NULL                            COMMENT '周期: daily/weekly/monthly/permanent',
    period_start    DATE            NOT NULL                            COMMENT '周期开始日期',
    period_end      DATE            NOT NULL                            COMMENT '周期结束日期',
    bought_count    INT             DEFAULT 0                           COMMENT '已购买数量',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    UNIQUE KEY uk_player_item_period (player_id, shop_item_id, period_type, period_start),
    INDEX idx_player_id (player_id),
    INDEX idx_period_end (period_end)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='限购消费统计表';

-- ============================================================
-- 示例数据
-- ============================================================

INSERT INTO shop_items (id, name, description, item_type, item_ref_id, currency_type, price, discount, stock, limit_type, limit_count, required_level, required_realm, tab, sort_order, active) VALUES
(1,  '灵石*100',      '一百枚下品灵石',     'item',    1001, 'coin',          500,   1.0, -1,  'none',     0,  1,  NULL,  'general', 1, 1),
(2,  '灵石*1000',     '一千枚下品灵石',     'item',    1002, 'coin',          4500,  0.9, -1,  'none',     0,  1,  NULL,  'general', 2, 1),
(3,  '筑基丹',        '突破筑基期必备丹药', 'item',    2001, 'coin',          10000, 1.0, 100, 'none',     0,  10, '炼气', 'general', 3, 1),
(4,  '结金丹',        '突破金丹期必备丹药', 'item',    2002, 'spirit_stone',  50,    1.0, 50,  'none',     0,  30, '筑基', 'general', 4, 1),
(5,  '随机灵兽蛋',    '随机孵化一只灵兽',   'pet_egg', 3001, 'spirit_stone',  200,   1.0, -1,  'none',     0,  20, '筑基', 'general', 5, 1),
(6,  '每日特惠礼包',  '每日限购一次的礼包', 'item',    4001, 'coin',          680,   0.5, -1,  'daily',    1,  1,  NULL,  'limited', 1, 1),
(7,  '周限灵石包',    '每周限购的灵石包',   'item',    4002, 'coin',          3000,  0.7, -1,  'weekly',   3,  5,  NULL,  'limited', 2, 1),
(8,  '仙缘月卡',      '购买后30天每日领取奖励', 'item',5001, 'spirit_stone', 300,   1.0, -1,  'permanent',0,  15, '炼气', 'gift',   1, 1),
(9,  '高级月卡',      '包含仙缘月卡全部内容',   'item',5002, 'spirit_stone', 680,   1.0, -1,  'permanent',0,  30, '筑基', 'gift',   2, 1),
(10, '战力至尊礼包',  'VIP专属礼包',        'item',    6001, 'bound_stone',   1280,  0.6, 5,   'permanent',1,  50, '金丹', 'vip',    1, 1);

INSERT INTO player_purchase_records (id, player_id, shop_item_id, quantity, unit_price, total_price, currency_type, created_at) VALUES
(1, 1, 1, 5,  500,   2500,  'coin',         '2026-06-01 10:00:00'),
(2, 1, 3, 1,  10000, 10000, 'coin',         '2026-06-01 10:30:00'),
(3, 2, 5, 2,  200,   400,   'spirit_stone', '2026-06-02 14:00:00'),
(4, 1, 8, 1,  300,   300,   'spirit_stone', '2026-06-03 08:00:00');

INSERT INTO player_purchase_limits (id, player_id, shop_item_id, period_type, period_start, period_end, bought_count) VALUES
(1, 1, 6, 'daily',   '2026-06-05', '2026-06-05', 1),
(2, 1, 7, 'weekly',  '2026-06-01', '2026-06-07', 2);
