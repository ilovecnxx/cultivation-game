-- 009: 洞府建造系统
-- 化神期(41层+)解锁，每个玩家最多1座洞府
-- 洞府等级: 1-10
-- 房间类型: 1修炼室 2炼丹房 3藏宝阁 4灵兽园 5阵法室
-- 房间等级: 1-5

CREATE TABLE IF NOT EXISTS dongfu (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '洞府ID',
    player_id         BIGINT       NOT NULL UNIQUE COMMENT '玩家ID（唯一，每人1座）',
    level             INT          DEFAULT 1 COMMENT '洞府等级 1-10',
    name              VARCHAR(32)  NOT NULL DEFAULT '洞府' COMMENT '洞府名称',
    cultivation_bonus DECIMAL(10,2) DEFAULT 0.00 COMMENT '修炼加成%',
    defense_bonus     DECIMAL(10,2) DEFAULT 0.00 COMMENT '防御加成%',
    created_at        DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at        DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='洞府表';

CREATE TABLE IF NOT EXISTS dongfu_rooms (
    id        BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '房间ID',
    dongfu_id BIGINT       NOT NULL COMMENT '所属洞府ID',
    room_type INT          NOT NULL COMMENT '房间类型: 1修炼室 2炼丹房 3藏宝阁 4灵兽园 5阵法室',
    level     INT          DEFAULT 1 COMMENT '房间等级 1-5',
    name      VARCHAR(32)  NOT NULL COMMENT '房间名称',
    effect    VARCHAR(128) DEFAULT '' COMMENT '效果描述',
    bonus     DECIMAL(10,2) DEFAULT 0.00 COMMENT '当前加成值',
    INDEX idx_dongfu (dongfu_id),
    CONSTRAINT fk_rooms_dongfu FOREIGN KEY (dongfu_id) REFERENCES dongfu(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='洞府房间表';
