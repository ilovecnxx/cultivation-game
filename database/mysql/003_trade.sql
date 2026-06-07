-- ===================================================================
-- 修仙游戏交易服务 - 数据库表结构
-- ===================================================================
-- 版本: 003
-- 说明: 交易服务专用表，包括市场挂单、交易记录、拍卖和玩家灵石
-- ===================================================================

-- 1. 市场挂单表
CREATE TABLE IF NOT EXISTS trade_listings (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY          COMMENT '挂单唯一 ID',
    seller_id     BIGINT UNSIGNED NOT NULL                            COMMENT '卖家 ID',
    seller_name   VARCHAR(64) NOT NULL DEFAULT ''                     COMMENT '卖家名称',
    item_id       INT UNSIGNED NOT NULL                               COMMENT '物品模板 ID',
    item_name     VARCHAR(128) NOT NULL DEFAULT ''                    COMMENT '物品名称',
    quantity      INT UNSIGNED NOT NULL                               COMMENT '数量',
    unit_price    BIGINT UNSIGNED NOT NULL                            COMMENT '单价（灵石）',
    currency_type VARCHAR(32) NOT NULL DEFAULT 'spirit_stone'         COMMENT '货币类型',
    status        VARCHAR(32) NOT NULL DEFAULT 'active'               COMMENT '状态：active/sold/cancelled/expired',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP         COMMENT '创建时间',
    expires_at    DATETIME NOT NULL                                   COMMENT '过期时间',
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_seller_id (seller_id),
    INDEX idx_item_id (item_id),
    INDEX idx_status (status),
    INDEX idx_expires_at (expires_at),
    INDEX idx_seller_status (seller_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='市场挂单表';

-- 2. 交易记录表
CREATE TABLE IF NOT EXISTS trade_transactions (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY            COMMENT '交易唯一 ID',
    listing_id  BIGINT UNSIGNED NOT NULL                              COMMENT '关联挂单 ID',
    buyer_id    BIGINT UNSIGNED NOT NULL                              COMMENT '买家 ID',
    seller_id   BIGINT UNSIGNED NOT NULL                              COMMENT '卖家 ID',
    item_id     INT UNSIGNED NOT NULL                                 COMMENT '物品模板 ID',
    quantity    INT UNSIGNED NOT NULL                                 COMMENT '数量',
    unit_price  BIGINT UNSIGNED NOT NULL                              COMMENT '成交单价',
    total_price BIGINT UNSIGNED NOT NULL                              COMMENT '成交总价',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP           COMMENT '成交时间',
    INDEX idx_listing_id (listing_id),
    INDEX idx_buyer_id (buyer_id),
    INDEX idx_seller_id (seller_id),
    INDEX idx_item_id (item_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='交易记录表';

-- 3. 拍卖表
CREATE TABLE IF NOT EXISTS trade_auctions (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY          COMMENT '拍卖唯一 ID',
    item_id       INT UNSIGNED NOT NULL                               COMMENT '物品模板 ID',
    seller_id     BIGINT UNSIGNED NOT NULL                            COMMENT '卖家 ID',
    current_bid   BIGINT UNSIGNED NOT NULL DEFAULT 0                  COMMENT '当前最高出价',
    bidder_id     BIGINT UNSIGNED NOT NULL DEFAULT 0                  COMMENT '当前最高出价者 ID',
    reserve_price BIGINT UNSIGNED NOT NULL DEFAULT 0                  COMMENT '保留价',
    end_time      DATETIME NOT NULL                                   COMMENT '结束时间',
    status        VARCHAR(32) NOT NULL DEFAULT 'active'               COMMENT '状态：active/completed/cancelled/expired',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP         COMMENT '创建时间',
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_seller_id (seller_id),
    INDEX idx_status (status),
    INDEX idx_end_time (end_time),
    INDEX idx_status_end (status, end_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拍卖表';

-- 4. 玩家灵石表（用于交易时的资产扣减和增加）
CREATE TABLE IF NOT EXISTS trade_player_gold (
    player_id  BIGINT UNSIGNED PRIMARY KEY                           COMMENT '玩家 ID',
    gold       BIGINT UNSIGNED NOT NULL DEFAULT 0                    COMMENT '灵石数量',
    version    INT UNSIGNED NOT NULL DEFAULT 0                       COMMENT '乐观锁版本号',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家灵石表';
