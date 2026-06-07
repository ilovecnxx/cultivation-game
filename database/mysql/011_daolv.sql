-- ============================================================
-- 道侣双修系统 (金丹期解锁)
-- ============================================================

-- 道侣关系表
CREATE TABLE IF NOT EXISTS daolv_relations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '关系ID',
    player_a BIGINT NOT NULL COMMENT '玩家A ID',
    player_b BIGINT NOT NULL COMMENT '玩家B ID',
    intimacy INT DEFAULT 0 COMMENT '亲密度',
    compatibility FLOAT DEFAULT 0.5 COMMENT '契合度 (0~1)',
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '结为道侣时间',
    status VARCHAR(16) DEFAULT 'normal' COMMENT '状态: normal/divorced',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE KEY uk_players (player_a, player_b),
    INDEX idx_player_a (player_a),
    INDEX idx_player_b (player_b)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='道侣关系表';

-- 道侣求婚申请表
CREATE TABLE IF NOT EXISTS daolv_proposals (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '申请ID',
    from_id BIGINT NOT NULL COMMENT '求婚方ID',
    to_id BIGINT NOT NULL COMMENT '被求婚方ID',
    gift_json TEXT COMMENT '彩礼 JSON',
    message TEXT COMMENT '求婚留言',
    status VARCHAR(16) DEFAULT 'pending' COMMENT '状态: pending/accepted/rejected',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '申请时间',
    handled_at DATETIME COMMENT '处理时间',
    INDEX idx_from_id (from_id),
    INDEX idx_to_id (to_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='道侣求婚申请表';

-- 双修记录表
CREATE TABLE IF NOT EXISTS daolv_cultivate_records (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '记录ID',
    relation_id BIGINT NOT NULL COMMENT '道侣关系ID',
    player_a BIGINT NOT NULL COMMENT '玩家A ID',
    player_b BIGINT NOT NULL COMMENT '玩家B ID',
    duration INT NOT NULL COMMENT '修炼时长(秒)',
    gain_a BIGINT DEFAULT 0 COMMENT 'A获得修为',
    gain_b BIGINT DEFAULT 0 COMMENT 'B获得修为',
    intimacy_gain INT DEFAULT 0 COMMENT '亲密度增加',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '修炼时间',
    INDEX idx_relation_id (relation_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='双修记录表';

-- 道侣礼物赠送记录
CREATE TABLE IF NOT EXISTS daolv_gift_records (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '记录ID',
    relation_id BIGINT NOT NULL COMMENT '道侣关系ID',
    from_id BIGINT NOT NULL COMMENT '赠送方ID',
    to_id BIGINT NOT NULL COMMENT '接收方ID',
    item_id BIGINT NOT NULL COMMENT '物品ID',
    quantity INT NOT NULL DEFAULT 1 COMMENT '数量',
    intimacy_gain INT DEFAULT 0 COMMENT '亲密度增加',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '赠送时间',
    INDEX idx_relation_id (relation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='道侣礼物赠送记录';
