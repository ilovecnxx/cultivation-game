-- ============================================================
-- 体力/能量系统 - 数据表 (v2.0 - 修炼打坐恢复)
-- 核心设计：体力通过修炼打坐恢复和丹药回复，随境界提高而增长
-- ============================================================

-- 1. 玩家能量表
CREATE TABLE IF NOT EXISTS player_energy (
    id                        BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键',
    player_id                 BIGINT NOT NULL                  COMMENT '玩家ID，关联players.id',
    current_energy            INT DEFAULT 100                  COMMENT '当前体力值',
    max_energy                INT DEFAULT 100                  COMMENT '当前体力上限',
    last_meditation_at        TIMESTAMP DEFAULT NULL           COMMENT '上次修炼打坐时间（用于计算离线回复）',
    energy_pills_used_today   INT DEFAULT 0                    COMMENT '今日已使用体力丹药次数',
    created_at                TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at                TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                              ON UPDATE CURRENT_TIMESTAMP      COMMENT '更新时间',

    UNIQUE KEY uk_player_id (player_id) COMMENT '一个玩家只有一条能量记录',
    INDEX idx_player_id (player_id),
    INDEX idx_last_meditation_at (last_meditation_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家体力/能量表（修炼打坐恢复版）';

-- 2. 体力丹药配置表
CREATE TABLE IF NOT EXISTS energy_pill_config (
    id              INT AUTO_INCREMENT PRIMARY KEY COMMENT '配置ID',
    pill_id         INT NOT NULL                   COMMENT '物品ID（关联items.id）',
    name            VARCHAR(64) NOT NULL            COMMENT '丹药名称',
    tier            TINYINT NOT NULL DEFAULT 1      COMMENT '丹药品阶 1-5',
    energy_restore  INT NOT NULL                    COMMENT '回复体力值',
    realm_required  INT NOT NULL DEFAULT 1          COMMENT '所需最低境界',
    description     VARCHAR(255) DEFAULT ''         COMMENT '描述',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    UNIQUE KEY uk_pill_id (pill_id),
    INDEX idx_tier (tier),
    INDEX idx_realm_required (realm_required)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='体力丹药配置表';

-- 3. 功法体力回复加成配置
CREATE TABLE IF NOT EXISTS technique_energy_bonus (
    id              INT AUTO_INCREMENT PRIMARY KEY COMMENT '主键',
    technique_id    INT NOT NULL                   COMMENT '功法ID',
    name            VARCHAR(64) NOT NULL            COMMENT '功法名称',
    regen_bonus     DECIMAL(5,2) NOT NULL DEFAULT 0.0 COMMENT '体力回复加成系数（如0.2=+20%）',
    description     VARCHAR(255) DEFAULT ''         COMMENT '效果描述',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    UNIQUE KEY uk_technique_id (technique_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='功法体力回复加成配置表';
