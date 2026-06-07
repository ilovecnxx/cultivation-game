-- ============================================================
-- 炼丹增强系统数据库迁移 (Alchemy Enhanced System)
-- 创建时间: 2026-06-05
-- ============================================================

-- 1. 丹方表（完整丹方库）
CREATE TABLE IF NOT EXISTS `alchemy_formulas` (
    `id`                 INT(11)        NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '丹方ID',
    `name`               VARCHAR(64)    NOT NULL COMMENT '丹方名称',
    `description`        VARCHAR(512)   DEFAULT '' COMMENT '丹方描述',
    `materials`          TEXT           NOT NULL COMMENT '所需材料ID列表(JSON数组)',
    `base_quality`       TINYINT(1)     NOT NULL DEFAULT 1 COMMENT '基础品质档位 0-5',
    `min_level`          INT(11)        NOT NULL DEFAULT 1 COMMENT '最低炼丹等级',
    `realm_required`     INT(11)        NOT NULL DEFAULT 1 COMMENT '最低境界ID',
    `research_difficulty` DOUBLE        NOT NULL DEFAULT 0 COMMENT '研究难度 0.0-1.0',
    `is_rare`            TINYINT(1)     NOT NULL DEFAULT 0 COMMENT '是否稀有(需Boss材料)',
    `effect`             VARCHAR(256)   DEFAULT '' COMMENT '效果描述',
    `craft_time`         INT(11)        NOT NULL DEFAULT 30 COMMENT '炼制耗时(秒)',
    `exp_value`          INT(11)        NOT NULL DEFAULT 100 COMMENT '基础经验值',
    `created_at`         DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_min_level` (`min_level`),
    INDEX `idx_realm` (`realm_required`),
    INDEX `idx_rare` (`is_rare`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='丹方定义';

-- 2. 玩家已研究丹方表
CREATE TABLE IF NOT EXISTS `player_formulas` (
    `id`             BIGINT(20)     NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '记录ID',
    `player_id`      BIGINT(20)     NOT NULL COMMENT '玩家ID',
    `formula_id`     INT(11)        NOT NULL COMMENT '丹方ID',
    `discovered`     TINYINT(1)     NOT NULL DEFAULT 1 COMMENT '是否已发现',
    `discovered_at`  DATETIME       DEFAULT CURRENT_TIMESTAMP COMMENT '发现时间',
    `craft_count`    INT(11)        NOT NULL DEFAULT 0 COMMENT '炼制次数',
    `updated_at`     DATETIME       DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE KEY `uk_player_formula` (`player_id`, `formula_id`),
    INDEX `idx_player_id` (`player_id`),
    INDEX `idx_formula_id` (`formula_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家已研究丹方';

-- 3. 玩家丹毒表
CREATE TABLE IF NOT EXISTS `player_toxicity` (
    `player_id`   BIGINT(20)     NOT NULL PRIMARY KEY COMMENT '玩家ID',
    `value`       INT(11)        NOT NULL DEFAULT 0 COMMENT '丹毒值 0-100',
    `updated_at`  DATETIME       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_value` (`value`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家丹毒';

-- 4. 玩家丹炉表
CREATE TABLE IF NOT EXISTS `player_furnace` (
    `id`              VARCHAR(64)    NOT NULL PRIMARY KEY COMMENT '丹炉ID',
    `player_id`       BIGINT(20)     NOT NULL COMMENT '玩家ID',
    `quality`         TINYINT(1)     NOT NULL DEFAULT 0 COMMENT '品质 0=青铜/1=白银/2=黄金/3=仙品',
    `durability`      INT(11)        NOT NULL DEFAULT 100 COMMENT '当前耐久度',
    `max_durability`  INT(11)        NOT NULL DEFAULT 100 COMMENT '最大耐久度',
    `created_at`      DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME       DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE KEY `uk_player_id` (`player_id`),
    INDEX `idx_quality` (`quality`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家丹炉';

-- 5. 炼丹会话表（活跃会话）
CREATE TABLE IF NOT EXISTS `alchemy_sessions` (
    `session_id`     VARCHAR(128)   NOT NULL PRIMARY KEY COMMENT '会话ID(playerID:formulaID)',
    `player_id`      BIGINT(20)     NOT NULL COMMENT '玩家ID',
    `formula_id`     INT(11)        NOT NULL COMMENT '丹方ID',
    `furnace_id`     VARCHAR(64)    DEFAULT '' COMMENT '丹炉ID',
    `start_time`     DATETIME       NOT NULL COMMENT '开始时间',
    `heat_zone`      TINYINT(1)     NOT NULL DEFAULT 1 COMMENT '火候区域 0=低/1=中/2=高',
    `heat_timer`     DOUBLE         NOT NULL DEFAULT 0.5 COMMENT '火候进度 0-1',
    `phase`          VARCHAR(16)    NOT NULL DEFAULT 'heating' COMMENT '阶段 heating/adding/condensing/completed',
    `ingredients`    TEXT           DEFAULT NULL COMMENT '已添加材料(JSON数组)',
    `score`          INT(11)        NOT NULL DEFAULT 0 COMMENT '小游戏评分 0-100',
    `base_quality`   TINYINT(1)     NOT NULL DEFAULT 0 COMMENT '基础品质',
    `final_quality`  TINYINT(1)     DEFAULT NULL COMMENT '最终品质',
    `toxicity`       INT(11)        DEFAULT 0 COMMENT '丹毒值',
    `completed`      TINYINT(1)     NOT NULL DEFAULT 0 COMMENT '是否完成',
    `created_at`     DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME       DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_player_id` (`player_id`),
    INDEX `idx_completed` (`completed`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='炼丹会话';

-- 6. 研究记录表
CREATE TABLE IF NOT EXISTS `research_records` (
    `id`             BIGINT(20)     NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '记录ID',
    `player_id`      BIGINT(20)     NOT NULL COMMENT '玩家ID',
    `formula_id`     INT(11)        NOT NULL COMMENT '丹方ID',
    `formula_name`   VARCHAR(64)    DEFAULT '' COMMENT '丹方名称',
    `success`        TINYINT(1)     NOT NULL DEFAULT 0 COMMENT '是否成功',
    `timestamp`      DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '研究时间',
    INDEX `idx_player_id` (`player_id`),
    INDEX `idx_formula_id` (`formula_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='研究记录';

-- 7. 研究尝试计数器表
CREATE TABLE IF NOT EXISTS `research_attempts` (
    `player_id`      BIGINT(20)     NOT NULL PRIMARY KEY COMMENT '玩家ID',
    `daily_count`    INT(11)        NOT NULL DEFAULT 0 COMMENT '今日已用免费次数',
    `last_reset`     DATETIME       DEFAULT CURRENT_TIMESTAMP COMMENT '上次重置时间',
    INDEX `idx_daily` (`daily_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='研究尝试计数器';
