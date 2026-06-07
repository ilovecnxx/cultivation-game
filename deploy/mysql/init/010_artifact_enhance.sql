-- 010: 法宝系统增强 - 类型、进化、觉醒、器灵、共鸣、试炼

-- ===== 1. 玩家法宝表增强 =====
ALTER TABLE player_artifacts
    ADD COLUMN `type`          INT    DEFAULT 1 COMMENT '法宝类型: 1飞剑 2护盾 3宝塔 4灵珠 5羽翼 6印玺' AFTER name,
    ADD COLUMN `mp_bonus`      BIGINT DEFAULT 0 COMMENT '灵力加成' AFTER hp_bonus,
    ADD COLUMN `speed_bonus`   BIGINT DEFAULT 0 COMMENT '速度加成' AFTER mp_bonus,
    ADD COLUMN `dodge_bonus`   BIGINT DEFAULT 0 COMMENT '闪避加成' AFTER speed_bonus,
    ADD COLUMN `awaken_skills` TEXT COMMENT '已解锁觉醒技能ID列表(JSON数组)' AFTER skill_id,
    ADD COLUMN `potential`     INT    DEFAULT 0 COMMENT '潜力点数' AFTER skill_id,
    ADD COLUMN `spirit_id`     BIGINT DEFAULT 0 COMMENT '关联器灵ID' AFTER skill_id,
    ADD COLUMN `power_bonus`   BIGINT DEFAULT 0 COMMENT '战力加成' AFTER dodge_bonus;

-- ===== 2. 器灵表 =====
CREATE TABLE IF NOT EXISTS artifact_spirits (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '器灵ID',
    artifact_id     BIGINT       NOT NULL COMMENT '所属法宝ID',
    player_id       BIGINT       NOT NULL COMMENT '玩家ID',
    name            VARCHAR(64)  NOT NULL COMMENT '器灵名称',
    personality     INT          DEFAULT 1 COMMENT '性格: 1内敛 2热忱 3高傲 4活泼 5阴沉 6古板',
    bond_level      INT          DEFAULT 1 COMMENT '好感等级 1-100',
    bond_exp        BIGINT       DEFAULT 0 COMMENT '好感经验',
    bond_unlocked   INT          DEFAULT 0 COMMENT '已解锁好感事件数',
    last_dialogue   TEXT         COMMENT '最近一句对话',
    last_event      VARCHAR(32)  DEFAULT '' COMMENT '最近触发事件',
    created_at      DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '激活时间',
    UNIQUE KEY uk_artifact (artifact_id),
    INDEX idx_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='法宝器灵表';

-- ===== 3. 法宝试炼进度表 =====
CREATE TABLE IF NOT EXISTS artifact_trial_progress (
    id                   BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '记录ID',
    player_id            BIGINT       NOT NULL COMMENT '玩家ID',
    artifact_id          BIGINT       NOT NULL COMMENT '法宝ID',
    completed_stages     TEXT         COMMENT '已完成试炼ID列表(JSON数组)',
    last_completed_stage INT          DEFAULT 0 COMMENT '最后完成试炼ID',
    today_attempts       INT          DEFAULT 0 COMMENT '今日尝试次数',
    last_attempt_date    VARCHAR(10)  DEFAULT '' COMMENT '最后尝试日期 YYYY-MM-DD',
    created_at           DATETIME     DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_artifact (artifact_id),
    INDEX idx_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='法宝试炼进度表';

-- ===== 4. 物品表中增加进化材料 =====
INSERT IGNORE INTO items (id, name, type, quality, required_level, required_realm, description, max_stack, sell_price) VALUES
(1001, '碧落石',     11, 2, 1, 1, '法宝进化材料·初级', 999, 20),
(1002, '天星砂',     11, 3, 20, 2, '法宝进化材料·中级', 999, 100),
(1003, '混沌石',     11, 4, 40, 3, '法宝进化材料·高级', 999, 500),
(1004, '鸿蒙紫气',   11, 5, 60, 4, '法宝进化材料·极品', 999, 2000),
(1005, '天道碎片',   11, 5, 80, 5, '法宝进化材料·传说', 999, 10000);

-- 更新 player_artifacts 已有记录，设置缺省值
UPDATE player_artifacts SET type=1 WHERE type IS NULL;
UPDATE player_artifacts SET mp_bonus=0 WHERE mp_bonus IS NULL;
UPDATE player_artifacts SET speed_bonus=0 WHERE speed_bonus IS NULL;
UPDATE player_artifacts SET dodge_bonus=0 WHERE dodge_bonus IS NULL;
UPDATE player_artifacts SET potential=0 WHERE potential IS NULL;
UPDATE player_artifacts SET spirit_id=0 WHERE spirit_id IS NULL;
UPDATE player_artifacts SET awaken_skills='[]' WHERE awaken_skills IS NULL;
UPDATE player_artifacts SET power_bonus=0 WHERE power_bonus IS NULL;
