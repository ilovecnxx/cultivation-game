-- ============================================================
-- 灵脉争夺系统数据库迁移 (Spirit Vein Contest System)
-- 创建时间: 2026-06-05
-- ============================================================

-- 1. 灵脉资源点表
CREATE TABLE IF NOT EXISTS `spirit_veins` (
    `id`                VARCHAR(64)    NOT NULL PRIMARY KEY COMMENT '灵脉ID',
    `name`              VARCHAR(128)   NOT NULL COMMENT '灵脉名称',
    `quality`           TINYINT(1)     NOT NULL DEFAULT 1 COMMENT '品质 1-5星',
    `region_id`         VARCHAR(64)    NOT NULL COMMENT '所属区域ID',
    `region_name`       VARCHAR(64)    DEFAULT '' COMMENT '区域名称',
    `position_x`        DOUBLE         NOT NULL DEFAULT 0 COMMENT '地图坐标X',
    `position_y`        DOUBLE         NOT NULL DEFAULT 0 COMMENT '地图坐标Y',
    `owner_type`        VARCHAR(16)    NOT NULL DEFAULT 'none' COMMENT '拥有者类型: none/player/sect',
    `owner_id`          VARCHAR(64)    DEFAULT '' COMMENT '拥有者ID',
    `owner_name`        VARCHAR(64)    DEFAULT '' COMMENT '拥有者名称',
    `occupied_since`    DATETIME       DEFAULT NULL COMMENT '占领时间',
    `last_yield_time`   DATETIME       DEFAULT NULL COMMENT '上次产出时间',
    `yield_interval`    INT(11)        NOT NULL DEFAULT 3600 COMMENT '产出间隔(秒)',
    `yield_amount`      BIGINT(20)     NOT NULL DEFAULT 0 COMMENT '每次产出灵石数量',
    `cultivation_bonus` DOUBLE         NOT NULL DEFAULT 0 COMMENT '修炼速度加成(%)',
    `status`            VARCHAR(16)    NOT NULL DEFAULT 'idle' COMMENT '状态: idle/contested/occupied',
    `discovered`        TINYINT(1)     NOT NULL DEFAULT 0 COMMENT '是否已被发现',
    `upgrade_level`     TINYINT(1)     NOT NULL DEFAULT 0 COMMENT '强化等级',
    `description`       TEXT           DEFAULT NULL COMMENT '灵脉描述',
    `created_at`        DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`        DATETIME       DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_region` (`region_id`),
    INDEX `idx_owner` (`owner_type`, `owner_id`),
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='灵脉资源点';

-- 2. 灵脉争夺战表
CREATE TABLE IF NOT EXISTS `vein_contests` (
    `id`              VARCHAR(64)    NOT NULL PRIMARY KEY COMMENT '争夺ID',
    `vein_id`         VARCHAR(64)    NOT NULL COMMENT '灵脉ID',
    `vein_name`       VARCHAR(128)   DEFAULT '' COMMENT '灵脉名称',
    `attacker_id`     VARCHAR(64)    NOT NULL COMMENT '攻击方玩家ID',
    `attacker_name`   VARCHAR(64)    DEFAULT '' COMMENT '攻击方名称',
    `defender_id`     VARCHAR(64)    NOT NULL COMMENT '防守方玩家ID',
    `defender_name`   VARCHAR(64)    DEFAULT '' COMMENT '防守方名称',
    `start_time`      DATETIME       NOT NULL COMMENT '开始时间',
    `end_time`        DATETIME       NOT NULL COMMENT '结束时间(30分钟限时)',
    `attacker_hp`     BIGINT(20)     NOT NULL DEFAULT 10000 COMMENT '攻击方HP',
    `attacker_max_hp` BIGINT(20)     NOT NULL DEFAULT 10000 COMMENT '攻击方最大HP',
    `defender_hp`     BIGINT(20)     NOT NULL DEFAULT 11000 COMMENT '防守方HP(含主场加成)',
    `defender_max_hp` BIGINT(20)     NOT NULL DEFAULT 11000 COMMENT '防守方最大HP',
    `status`          VARCHAR(16)    NOT NULL DEFAULT 'active' COMMENT '状态: active/attacker_win/defender_win/timeout',
    `spectators`      TEXT           DEFAULT NULL COMMENT '观战者列表(JSON)',
    `created_at`      DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME       DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_vein_id` (`vein_id`),
    INDEX `idx_status` (`status`),
    INDEX `idx_attacker` (`attacker_id`),
    INDEX `idx_defender` (`defender_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='灵脉争夺战';

-- 3. 灵脉升级记录表
CREATE TABLE IF NOT EXISTS `vein_upgrades` (
    `id`            VARCHAR(64)    NOT NULL PRIMARY KEY COMMENT '升级记录ID',
    `vein_id`       VARCHAR(64)    NOT NULL COMMENT '灵脉ID',
    `owner_id`      VARCHAR(64)    NOT NULL COMMENT '拥有者ID',
    `owner_type`    VARCHAR(16)    NOT NULL COMMENT '拥有者类型',
    `old_quality`   TINYINT(1)     NOT NULL COMMENT '升级前品质',
    `new_quality`   TINYINT(1)     NOT NULL COMMENT '升级后品质',
    `cost_stones`   BIGINT(20)     NOT NULL DEFAULT 0 COMMENT '消耗灵石',
    `duration`      INT(11)        NOT NULL DEFAULT 0 COMMENT '升级耗时(秒)',
    `start_time`    DATETIME       NOT NULL COMMENT '开始时间',
    `end_time`      DATETIME       NOT NULL COMMENT '结束时间',
    `status`        VARCHAR(16)    NOT NULL DEFAULT 'in_progress' COMMENT '状态: in_progress/completed/cancelled',
    `created_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_vein_id` (`vein_id`),
    INDEX `idx_owner` (`owner_id`),
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='灵脉升级记录';

-- 4. 灵脉发现记录表
CREATE TABLE IF NOT EXISTS `vein_discoveries` (
    `id`            VARCHAR(64)    NOT NULL PRIMARY KEY COMMENT '发现记录ID',
    `user_id`       VARCHAR(64)    NOT NULL COMMENT '玩家ID',
    `vein_id`       VARCHAR(64)    NOT NULL COMMENT '灵脉ID',
    `method`        VARCHAR(32)    NOT NULL COMMENT '发现方式: explore/divination/map_purchase',
    `reward_stones` BIGINT(20)     NOT NULL DEFAULT 0 COMMENT '发现奖励灵石',
    `discovered_at` DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发现时间',
    INDEX `idx_user_id` (`user_id`),
    INDEX `idx_vein_id` (`vein_id`),
    UNIQUE KEY `uk_user_vein` (`user_id`, `vein_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='灵脉发现记录';

-- 5. 灵脉占领历史表
CREATE TABLE IF NOT EXISTS `vein_occupation_history` (
    `id`            VARCHAR(64)    NOT NULL PRIMARY KEY COMMENT '历史记录ID',
    `vein_id`       VARCHAR(64)    NOT NULL COMMENT '灵脉ID',
    `vein_name`     VARCHAR(128)   DEFAULT '' COMMENT '灵脉名称',
    `vein_quality`  TINYINT(1)     DEFAULT 0 COMMENT '灵脉品质',
    `owner_type`    VARCHAR(16)    NOT NULL COMMENT '拥有者类型',
    `owner_id`      VARCHAR(64)    NOT NULL COMMENT '拥有者ID',
    `owner_name`    VARCHAR(64)    DEFAULT '' COMMENT '拥有者名称',
    `occupied_at`   DATETIME       NOT NULL COMMENT '占领时间',
    `lost_at`       DATETIME       DEFAULT NULL COMMENT '失去时间',
    `duration`      INT(11)        DEFAULT 0 COMMENT '占领持续时长(小时)',
    `created_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_vein_id` (`vein_id`),
    INDEX `idx_owner` (`owner_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='灵脉占领历史';

-- ============================================================
-- 初始数据：20条灵脉
-- ============================================================
INSERT INTO `spirit_veins` (`id`, `name`, `quality`, `region_id`, `region_name`, `position_x`, `position_y`, `yield_amount`, `cultivation_bonus`, `discovered`, `description`) VALUES
('vein_01', '九天灵脉', 5, 'secret_realm_04', '虚空裂隙', 120.5, 230.0, 2500, 25.0, 0, '品质5星的九天灵脉，每小时产出2500灵石，修炼速度+25%'),
('vein_02', '地脉灵泉', 5, 'danger_land_03', '葬神渊', 180.3, 280.5, 2500, 25.0, 0, '品质5星的地脉灵泉，每小时产出2500灵石，修炼速度+25%'),
('vein_03', '天罡灵脉', 4, 'secret_realm_03', '天池秘境', 250.0, 320.0, 2000, 20.0, 0, '品质4星的天罡灵脉，每小时产出2000灵石，修炼速度+20%'),
('vein_04', '玄黄灵脉', 4, 'secret_realm_03', '天池秘境', 310.0, 350.0, 2000, 20.0, 0, '品质4星的玄黄灵脉，每小时产出2000灵石，修炼速度+20%'),
('vein_05', '紫霄灵脉', 4, 'danger_land_02', '毒瘴沼泽', 400.0, 380.0, 2000, 20.0, 0, '品质4星的紫霄灵脉，每小时产出2000灵石，修炼速度+20%'),
('vein_06', '青木灵脉', 4, 'danger_land_02', '毒瘴沼泽', 450.0, 410.0, 2000, 20.0, 0, '品质4星的青木灵脉，每小时产出2000灵石，修炼速度+20%'),
('vein_07', '赤炎灵脉', 3, 'danger_land_01', '苍狼山', 500.0, 440.0, 1500, 15.0, 0, '品质3星的赤炎灵脉，每小时产出1500灵石，修炼速度+15%'),
('vein_08', '玄冰灵脉', 3, 'danger_land_01', '苍狼山', 550.0, 470.0, 1500, 15.0, 0, '品质3星的玄冰灵脉，每小时产出1500灵石，修炼速度+15%'),
('vein_09', '厚土灵脉', 3, 'secret_realm_01', '紫府秘境', 600.0, 500.0, 1500, 15.0, 0, '品质3星的厚土灵脉，每小时产出1500灵石，修炼速度+15%'),
('vein_10', '庚金灵脉', 3, 'secret_realm_01', '紫府秘境', 650.0, 530.0, 1500, 15.0, 0, '品质3星的庚金灵脉，每小时产出1500灵石，修炼速度+15%'),
('vein_11', '星辰灵脉', 3, 'secret_realm_02', '万妖窟', 700.0, 560.0, 1500, 15.0, 0, '品质3星的星辰灵脉，每小时产出1500灵石，修炼速度+15%'),
('vein_12', '月华灵脉', 3, 'secret_realm_02', '万妖窟', 750.0, 590.0, 1500, 15.0, 0, '品质3星的月华灵脉，每小时产出1500灵石，修炼速度+15%'),
('vein_13', '日曜灵脉', 2, 'town_02', '天机城', 800.0, 620.0, 1000, 10.0, 1, '品质2星的日曜灵脉，每小时产出1000灵石，修炼速度+10%'),
('vein_14', '雷音灵脉', 2, 'town_02', '天机城', 850.0, 650.0, 1000, 10.0, 1, '品质2星的雷音灵脉，每小时产出1000灵石，修炼速度+10%'),
('vein_15', '风啸灵脉', 2, 'town_01', '青云镇', 900.0, 680.0, 1000, 10.0, 1, '品质2星的风啸灵脉，每小时产出1000灵石，修炼速度+10%'),
('vein_16', '云梦灵脉', 2, 'town_01', '青云镇', 950.0, 710.0, 1000, 10.0, 1, '品质2星的云梦灵脉，每小时产出1000灵石，修炼速度+10%'),
('vein_17', '龙脉灵泉', 2, 'secret_realm_04', '虚空裂隙', 1000.0, 740.0, 1000, 10.0, 1, '品质2星的龙脉灵泉，每小时产出1000灵石，修炼速度+10%'),
('vein_18', '凤鸣灵脉', 1, 'newbie_village_01', '落日森林', 1050.0, 770.0, 500, 5.0, 1, '品质1星的凤鸣灵脉，每小时产出500灵石，修炼速度+5%'),
('vein_19', '麒麟灵脉', 1, 'newbie_village_01', '落日森林', 1100.0, 800.0, 500, 5.0, 1, '品质1星的麒麟灵脉，每小时产出500灵石，修炼速度+5%'),
('vein_20', '玄武灵脉', 1, 'newbie_village_01', '落日森林', 1150.0, 830.0, 500, 5.0, 1, '品质1星的玄武灵脉，每小时产出500灵石，修炼速度+5%');
