-- ============================================================
-- VIP系统 - 数据表
-- 版本: v1.0.0
-- 玩家VIP等级、充值记录与月卡系统
-- ============================================================

-- 1. VIP玩家表
-- 每个玩家有且仅有一条VIP记录，存储VIP等级、经验、累计充值、月卡信息
CREATE TABLE IF NOT EXISTS vip_players (
    id                      BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    player_id               BIGINT NOT NULL                 COMMENT '玩家ID，关联players.id',
    vip_level               INT NOT NULL DEFAULT 0          COMMENT 'VIP等级(0-9)',
    vip_exp                 BIGINT NOT NULL DEFAULT 0        COMMENT '当前VIP经验值',
    total_recharge          BIGINT NOT NULL DEFAULT 0        COMMENT '累计充值金额(仙玉)',
    monthly_card_expires_at TIMESTAMP NULL DEFAULT NULL      COMMENT '月卡到期时间',
    monthly_card_type       TINYINT NOT NULL DEFAULT 0       COMMENT '月卡类型:0=无 1=小月卡 2=大月卡',
    last_daily_claim_date   DATE NULL DEFAULT NULL           COMMENT '上次领取每日VIP奖励日期',
    created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    UNIQUE KEY uk_player_id (player_id)                     COMMENT '一个玩家只有一条VIP记录',
    INDEX idx_vip_level (vip_level)                         COMMENT 'VIP等级索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家VIP信息表';

-- 2. 充值记录表
-- 记录每笔充值订单的状态，用于订单验证和充值历史查询
CREATE TABLE IF NOT EXISTS vip_recharge_records (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    player_id       BIGINT NOT NULL                 COMMENT '玩家ID',
    amount_jade     INT NOT NULL                    COMMENT '获得仙玉数量',
    amount_rmb      INT NOT NULL                    COMMENT '充值金额(单位:分 100分=1元)',
    order_id        VARCHAR(64) NOT NULL            COMMENT '订单号(全局唯一)',
    status          TINYINT NOT NULL DEFAULT 0      COMMENT '状态:0=待支付 1=已完成 2=已失败',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    UNIQUE KEY uk_order_id (order_id)               COMMENT '订单号全局唯一',
    INDEX idx_player_id (player_id)                 COMMENT '玩家ID索引',
    INDEX idx_status (status)                       COMMENT '状态索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='充值记录表';
