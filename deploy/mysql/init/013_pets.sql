-- ============================================================
-- 灵兽宠物系统 (金丹期解锁)
-- ============================================================

-- 玩家灵兽表
CREATE TABLE IF NOT EXISTS player_pets (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '灵兽ID',
    player_id       BIGINT          NOT NULL                            COMMENT '所属玩家ID',
    pet_id          INT             NOT NULL                            COMMENT '灵兽模板ID(1=青云雀/2=玄龟/3=火凤/4=冰蚕/5=雷狼)',
    name            VARCHAR(32)     DEFAULT NULL                        COMMENT '自定义名称',
    level           INT             DEFAULT 1                           COMMENT '等级',
    exp             BIGINT          DEFAULT 0                           COMMENT '当前经验',
    growth          FLOAT           DEFAULT 1.0                         COMMENT '成长率(1.0~2.0)',
    quality         VARCHAR(16)     DEFAULT 'common'                    COMMENT '品质: common/uncommon/rare/epic/legendary',
    hp              BIGINT          DEFAULT 100                         COMMENT '生命',
    attack          BIGINT          DEFAULT 10                          COMMENT '攻击',
    defense         BIGINT          DEFAULT 10                          COMMENT '防御',
    speed           INT             DEFAULT 100                         COMMENT '速度',
    skill_ids       JSON            DEFAULT NULL                        COMMENT '已学技能ID列表(JSON数组)',
    equip_slot      INT             DEFAULT 0                           COMMENT '当前出战位(0=未出战,1~3)',
    bond_value      INT             DEFAULT 0                           COMMENT '亲密度',
    status          VARCHAR(16)     DEFAULT 'idle'                      COMMENT '状态: idle/fighting/exploring/breeding',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '获取时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player_id (player_id),
    INDEX idx_quality (quality),
    INDEX idx_status (status),
    INDEX idx_equip_slot (player_id, equip_slot)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家灵兽表';

-- 灵兽技能表
CREATE TABLE IF NOT EXISTS pet_skills (
    id              INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '技能ID',
    name            VARCHAR(32)     NOT NULL                            COMMENT '技能名称',
    description     TEXT                                                COMMENT '技能描述',
    type            VARCHAR(16)     NOT NULL                            COMMENT '类型: passive/active/ultimate',
    element         VARCHAR(16)     DEFAULT 'none'                      COMMENT '属性: none/fire/water/wind/thunder/ice',
    target          VARCHAR(16)     DEFAULT 'single'                    COMMENT '目标: self/single/ally_all/enemy_all',
    effect_type     VARCHAR(32)     DEFAULT NULL                        COMMENT '效果类型: damage/heal/buff/debuff',
    base_value      BIGINT          DEFAULT 0                           COMMENT '基础效果值',
    scaling         FLOAT           DEFAULT 0.0                         COMMENT '属性倍率(0~1)',
    cooldown        INT             DEFAULT 0                           COMMENT '冷却回合数',
    max_level       INT             DEFAULT 5                           COMMENT '最大等级',
    unlock_pet_id   INT             DEFAULT NULL                        COMMENT '对应灵兽模板ID(NULL=通用)',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_type (type),
    INDEX idx_element (element),
    INDEX idx_unlock_pet (unlock_pet_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='灵兽技能表';

-- 灵兽孵化表
CREATE TABLE IF NOT EXISTS pet_eggs (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '蛋ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    egg_template_id INT             NOT NULL                            COMMENT '蛋模板ID',
    incubating      TINYINT(1)      DEFAULT 0                           COMMENT '是否正在孵化(0=未孵化,1=孵化中)',
    incubate_start  DATETIME        DEFAULT NULL                        COMMENT '孵化开始时间',
    incubate_end    DATETIME        DEFAULT NULL                        COMMENT '孵化完成时间',
    quality_hint    VARCHAR(16)     DEFAULT 'common'                    COMMENT '品质提示',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '获取时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player_id (player_id),
    INDEX idx_incubating (player_id, incubating)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='灵兽孵化表';

-- ============================================================
-- 示例数据
-- ============================================================

INSERT INTO pet_skills (id, name, description, type, element, target, effect_type, base_value, scaling, cooldown, max_level, unlock_pet_id) VALUES
(1, '疾风斩',  '以风刃攻击单个敌人',           'active',  'wind',  'single',    'damage', 50,  0.3, 0, 5, 1),
(2, '青云护体','提升自身防御力',               'active',  'wind',  'self',      'buff',   20,  0.2, 3, 5, 1),
(3, '龟甲防御','大幅提升自身防御',             'active',  'water', 'self',      'buff',   40,  0.4, 3, 5, 2),
(4, '水疗术',  '恢复自身生命值',               'active',  'water', 'self',      'heal',   80,  0.2, 2, 5, 2),
(5, '烈焰冲击','以火焰攻击全体敌人',           'active',  'fire',  'enemy_all', 'damage', 60,  0.3, 4, 5, 3),
(6, '涅槃重生','战斗中有几率复活',             'passive', 'fire',  'self',      'buff',   20,  0.0, 0, 3, 3),
(7, '寒冰吐息','冰冻单个敌人',                 'active',  'ice',   'single',    'debuff', 30,  0.2, 2, 5, 4),
(8, '灵丝缠绕','降低敌人速度',                 'active',  'ice',   'single',    'debuff', 15,  0.1, 1, 5, 4),
(9, '雷狼怒吼','提升全体友方攻击力',           'active',  'thunder','ally_all', 'buff',   25,  0.2, 3, 5, 5),
(10,'闪电突袭','对单个敌人造成雷属性伤害',     'active',  'thunder','single',    'damage', 90,  0.4, 2, 5, 5);

INSERT INTO player_pets (id, player_id, pet_id, name, level, exp, growth, quality, hp, attack, defense, speed, skill_ids, equip_slot, bond_value, status) VALUES
(1, 1, 1, '小云雀',  10, 500,  1.2, 'uncommon',   1200, 85,  65,  120, '[1,2]',  1, 300, 'idle'),
(2, 1, 2, '铁甲龟',  8,  300,  1.5, 'rare',       2000, 40,  120, 40,  '[3,4]',  2, 150, 'idle'),
(3, 2, 5, '苍雷',    12, 800,  1.8, 'epic',       1800, 120, 70,  140, '[9,10]', 1, 500, 'fighting');

INSERT INTO pet_eggs (id, player_id, egg_template_id, incubating, incubate_start, incubate_end, quality_hint) VALUES
(1, 1, 3, 1, '2026-06-01 08:00:00', '2026-06-03 08:00:00', 'rare'),
(2, 2, 1, 0, NULL, NULL, 'common');
