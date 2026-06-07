-- 019: 洞府系统增强
-- 增加房间类型（1-5，等级上限改为10级）
-- 新增灵气汇聚、装饰、访客功能
-- 洞府等级 = 所有房间等级之和

-- 1. 洞府表新增字段（alchemy_bonus, storage_bonus, combat_exp_per_hour, spirit_stones_per_hour, spirit_energy）
ALTER TABLE dongfu
    ADD COLUMN alchemy_bonus     DECIMAL(10,2) DEFAULT 0.00 COMMENT '炼丹加成%' AFTER cultivation_bonus,
    ADD COLUMN storage_bonus     DECIMAL(10,2) DEFAULT 0.00 COMMENT '存储加成' AFTER alchemy_bonus,
    ADD COLUMN combat_exp_per_hour DECIMAL(10,2) DEFAULT 0.00 COMMENT '战斗经验/小时' AFTER storage_bonus,
    ADD COLUMN spirit_stones_per_hour DECIMAL(10,2) DEFAULT 0.00 COMMENT '灵石产出/小时' AFTER combat_exp_per_hour,
    ADD COLUMN spirit_energy     DECIMAL(10,2) DEFAULT 0.00 COMMENT '洞府灵气值' AFTER spirit_stones_per_hour;

-- 2. 灵气汇聚表
CREATE TABLE IF NOT EXISTS dongfu_spirit_gathering (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '记录ID',
    dongfu_id         BIGINT       NOT NULL COMMENT '洞府ID',
    player_id         BIGINT       NOT NULL COMMENT '玩家ID',
    status            TINYINT      DEFAULT 0 COMMENT '状态 0=空闲 1=汇聚中',
    start_time        DATETIME     NOT NULL COMMENT '开始时间',
    duration          INT          DEFAULT 0 COMMENT '计划时长(秒)',
    bonus_cultivation DECIMAL(10,2) DEFAULT 0.00 COMMENT '已累积修为',
    elapsed_seconds   INT          DEFAULT 0 COMMENT '已持续秒数',
    created_at        DATETIME     DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_dongfu (dongfu_id),
    INDEX idx_player (player_id),
    CONSTRAINT fk_gathering_dongfu FOREIGN KEY (dongfu_id) REFERENCES dongfu(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='洞府灵气汇聚表';

-- 3. 装饰表
CREATE TABLE IF NOT EXISTS dongfu_decorations (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '装饰ID',
    dongfu_id       BIGINT       NOT NULL COMMENT '洞府ID',
    player_id       BIGINT       NOT NULL COMMENT '玩家ID',
    item_id         VARCHAR(64)  NOT NULL COMMENT '物品标识',
    name            VARCHAR(64)  NOT NULL COMMENT '装饰名称',
    decoration_type INT          DEFAULT 0 COMMENT '类型 0=装饰 1=家具 2=盆景 3=挂画 4=奇石',
    bonus_type      VARCHAR(32)  DEFAULT '' COMMENT '加成类型',
    bonus_value     DECIMAL(10,2) DEFAULT 0.00 COMMENT '加成值',
    description     VARCHAR(256) DEFAULT '' COMMENT '描述',
    is_placed       TINYINT(1)   DEFAULT 1 COMMENT '是否已摆放',
    position_x      INT          DEFAULT 0 COMMENT '位置X',
    position_y      INT          DEFAULT 0 COMMENT '位置Y',
    created_at      DATETIME     DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_dongfu (dongfu_id),
    INDEX idx_player (player_id),
    CONSTRAINT fk_decoration_dongfu FOREIGN KEY (dongfu_id) REFERENCES dongfu(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='洞府装饰表';

-- 4. 访客表
CREATE TABLE IF NOT EXISTS dongfu_guests (
    id               BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '记录ID',
    dongfu_id        BIGINT       NOT NULL COMMENT '洞府ID',
    guest_player_id  BIGINT       NOT NULL COMMENT '访客玩家ID',
    host_player_id   BIGINT       NOT NULL COMMENT '房主玩家ID',
    status           VARCHAR(16)  DEFAULT 'pending' COMMENT '状态: pending/visiting/completed',
    visit_start      DATETIME     DEFAULT NULL COMMENT '开始拜访时间',
    visit_end        DATETIME     DEFAULT NULL COMMENT '结束拜访时间',
    host_bonus_type  VARCHAR(32)  DEFAULT '' COMMENT '房主加成类型',
    host_bonus_value DECIMAL(10,2) DEFAULT 0.00 COMMENT '房主加成值',
    guest_bonus_type VARCHAR(32)  DEFAULT '' COMMENT '访客加成类型',
    guest_bonus_value DECIMAL(10,2) DEFAULT 0.00 COMMENT '访客加成值',
    created_at       DATETIME     DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_dongfu (dongfu_id),
    INDEX idx_guest (guest_player_id),
    INDEX idx_host (host_player_id),
    CONSTRAINT fk_guest_dongfu FOREIGN KEY (dongfu_id) REFERENCES dongfu(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='洞府访客表';
