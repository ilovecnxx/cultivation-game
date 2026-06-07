-- ===================================================================
-- 修仙游戏装备套装/附魔/觉醒系统 - 表结构
-- ===================================================================

-- 1. 装备套装定义表（静态配置可缓存）
CREATE TABLE equipment_set_definitions (
    id                      VARCHAR(32)     NOT NULL        PRIMARY KEY         COMMENT '套装ID(如azure_dragon)',
    name                    VARCHAR(64)     NOT NULL                            COMMENT '套装名称(如青龙套装)',
    quality                 TINYINT         DEFAULT 5                           COMMENT '套装品质(1-5星)',
    element                 VARCHAR(16)     DEFAULT NULL                        COMMENT '五行属性',
    icon                    VARCHAR(16)     DEFAULT NULL                        COMMENT '图标字符',
    lore                    TEXT            DEFAULT NULL                        COMMENT '背景故事',
    pieces                  JSON            NOT NULL                            COMMENT '包含部件item_id数组',
    bonuses                 JSON            NOT NULL                            COMMENT '套装效果配置JSON',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='装备套装定义表';

-- 2. 玩家附魔记录表
CREATE TABLE player_enchantments (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id               BIGINT          NOT NULL                            COMMENT '玩家ID',
    equipment_id            BIGINT          NOT NULL                            COMMENT '装备记录ID(eqeuipment.id)',
    enchant_id              VARCHAR(32)     NOT NULL                            COMMENT '附魔类型ID',
    level                   INT             DEFAULT 1                           COMMENT '附魔等级',
    slot_index              TINYINT         DEFAULT 0                           COMMENT '附魔槽位索引(0-2,觉醒可增加)',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player (player_id),
    INDEX idx_equipment (equipment_id),
    INDEX idx_enchant (enchant_id),
    CONSTRAINT fk_ench_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家附魔记录表';

-- 3. 装备觉醒记录表
CREATE TABLE equipment_awakenings (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id               BIGINT          NOT NULL                            COMMENT '玩家ID',
    equipment_id            BIGINT          NOT NULL                            COMMENT '装备记录ID(equipment.id)',
    awaken_level            TINYINT         DEFAULT 1                           COMMENT '觉醒等级(1-3)',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '觉醒时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player (player_id),
    UNIQUE KEY uk_equipment (equipment_id),
    INDEX idx_awaken_level (awaken_level),
    CONSTRAINT fk_awaken_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='装备觉醒记录表';

-- 4. 装备表扩展字段（原equipment表新增索引）
-- 如果equipment表没有max_level字段，添加它
ALTER TABLE equipment
    ADD COLUMN max_level INT DEFAULT 20 COMMENT '最大强化等级(基础20,每觉醒+20)' AFTER level,
    ADD COLUMN awaken_level TINYINT DEFAULT 0 COMMENT '觉醒等级(0-3)' AFTER max_level,
    ADD INDEX idx_max_level (max_level),
    ADD INDEX idx_awaken (awaken_level);

-- ===================================================================
-- 插入默认套装配置数据
-- ===================================================================

INSERT INTO equipment_set_definitions (id, name, quality, element, icon, lore, pieces, bonuses) VALUES
('azure_dragon', '青龙套装', 5, '木', '🐉', '东方青龙，四象之首，蕴含无穷生机与力量。传说为天帝镇守东方的神兽之鳞所铸。',
 '[2001,2002,2003,2004,2005]',
 '[{"pieces_required":2,"description":"攻击力+10%","effects":[{"stat":"attack","value":10,"is_percent":true}]},{"pieces_required":3,"description":"暴击伤害+15%","effects":[{"stat":"crit_dmg","value":15,"is_percent":true}]},{"pieces_required":5,"description":"青龙之魂：攻击时10%几率吸取造成伤害的50%为生命值","effects":[{"stat":"lifesteal","value":10,"is_percent":true,"special":"青龙之魂"}]}]'),

('vermillion_bird', '朱雀套装', 5, '火', '🦅', '南方朱雀，浴火重生。蕴含南明离火之力的上古神装，得之可焚尽万物。',
 '[2006,2007,2008,2009,2010]',
 '[{"pieces_required":2,"description":"速度+10%","effects":[{"stat":"speed","value":10,"is_percent":true}]},{"pieces_required":3,"description":"火焰伤害+15%","effects":[{"stat":"fire_dmg","value":15,"is_percent":true,"special":"朱雀之炎"}]},{"pieces_required":5,"description":"涅槃：每场战斗可复活一次，恢复50%生命值","effects":[{"stat":"hp","value":50,"is_percent":true,"special":"涅槃重生"}]}]'),

('black_tortoise', '玄武套装', 5, '水', '🐢', '北方玄武，以防御称著。玄冰之甲，坚不可摧，乃天地间最坚固的防护。',
 '[2011,2012,2013,2014,2015]',
 '[{"pieces_required":2,"description":"防御力+15%","effects":[{"stat":"defense","value":15,"is_percent":true}]},{"pieces_required":3,"description":"生命值上限+20%","effects":[{"stat":"hp","value":20,"is_percent":true}]},{"pieces_required":5,"description":"玄武盾：战斗开始获得30%伤害减免，持续3回合","effects":[{"stat":"damage_reduction","value":30,"is_percent":true,"special":"玄武盾"}]}]'),

('white_tiger', '白虎套装', 5, '金', '🐯', '西方白虎，杀伐之气冠绝四方。庚金之气凝聚而成，锐不可当。',
 '[2016,2017,2018,2019,2020]',
 '[{"pieces_required":2,"description":"暴击率+10%","effects":[{"stat":"crit_rate","value":10,"is_percent":true}]},{"pieces_required":3,"description":"暴击伤害+20%","effects":[{"stat":"crit_dmg","value":20,"is_percent":true}]},{"pieces_required":5,"description":"虎威：暴击时有30%几率眩晕敌人一回合","effects":[{"stat":"crit_rate","value":0,"is_percent":false,"special":"虎威震慑"}]}]'),

('qilin', '麒麟套装', 5, '土', '🦄', '祥瑞之兽麒麟所化，蕴含大地之力与无尽祥瑞之气。得之者福缘深厚。',
 '[2021,2022,2023,2024,2025]',
 '[{"pieces_required":2,"description":"全属性+8%","effects":[{"stat":"all_stats","value":8,"is_percent":true,"special":"祥瑞之兆"}]},{"pieces_required":3,"description":"突破成功率+15%","effects":[{"stat":"breakthrough","value":15,"is_percent":true}]},{"pieces_required":5,"description":"祥瑞：机缘值+50","effects":[{"stat":"luck","value":50,"is_percent":false,"special":"麒麟祥瑞"}]}]'),

('chaos', '混沌套装', 5, '无', '🌀', '混沌初开，鸿蒙未判之时诞生的原始力量。蕴藏无尽的毁灭与创造之力。',
 '[2026,2027,2028,2029,2030]',
 '[{"pieces_required":2,"description":"伤害+5%","effects":[{"stat":"damage_bonus","value":5,"is_percent":true}]},{"pieces_required":3,"description":"护甲穿透+10%","effects":[{"stat":"armor_pen","value":10,"is_percent":true}]},{"pieces_required":5,"description":"混沌之力：每回合有20%几率发动一次额外攻击","effects":[{"stat":"damage_bonus","value":0,"is_percent":false,"special":"混沌之力"}]}]'),

('void', '太虚套装', 5, '无', '🌌', '太虚之境，无形无相。蕴含空间法则的至宝，能令佩戴者穿梭虚空。',
 '[2031,2032,2033,2034,2035]',
 '[{"pieces_required":2,"description":"闪避率+12%","effects":[{"stat":"dodge_rate","value":12,"is_percent":true}]},{"pieces_required":3,"description":"每回合额外行动1次","effects":[{"stat":"extra_actions","value":1,"is_percent":false,"special":"太虚步"}]},{"pieces_required":5,"description":"虚空：免疫受到的第一次攻击","effects":[{"stat":"dodge_rate","value":0,"is_percent":false,"special":"虚空之体"}]}]'),

('primordial', '鸿蒙套装', 5, '全', '☯️', '鸿蒙初开，大道之始。传说为开天辟地之前就已存在的至高神器。',
 '[2036,2037,2038,2039,2040]',
 '[{"pieces_required":2,"description":"全属性+15%","effects":[{"stat":"all_stats","value":15,"is_percent":true,"special":"鸿蒙之气"}]},{"pieces_required":3,"description":"修炼经验获取+25%","effects":[{"stat":"exp_bonus","value":25,"is_percent":true}]},{"pieces_required":5,"description":"鸿蒙初开：所有套装效果翻倍","effects":[{"stat":"all_stats","value":0,"is_percent":false,"special":"鸿蒙初开"}]}]');
