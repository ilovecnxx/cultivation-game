-- ============================================================
-- 阵法系统 v2 (金丹期解锁)
-- 8 大阵法类型：五行阵/八卦阵/诛仙阵/护山大阵/聚灵阵/迷魂阵/传送阵/天罡北斗阵
-- 新增：阵法熟练度、守护灵兽、阵法联动、阵法相克
-- ============================================================

-- 阵法模板表
CREATE TABLE IF NOT EXISTS formation_templates (
    id              INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '阵法ID',
    name            VARCHAR(32)     NOT NULL                            COMMENT '阵法名称',
    type            TINYINT         NOT NULL DEFAULT 1                  COMMENT '阵法类型 1-8',
    quality         TINYINT         NOT NULL DEFAULT 3                  COMMENT '基础品质 1-5',
    description     TEXT                                                COMMENT '阵法描述',
    effects_json    JSON            DEFAULT NULL                        COMMENT '效果配置[{type,value}]',
    learn_cost      BIGINT          DEFAULT 0                           COMMENT '学习消耗灵石',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='阵法模板表';

-- 玩家已习得阵法表
CREATE TABLE IF NOT EXISTS player_formations (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    tmpl_id         INT             NOT NULL                            COMMENT '模板ID',
    name            VARCHAR(32)     NOT NULL                            COMMENT '阵法名称',
    type            TINYINT         NOT NULL DEFAULT 1                  COMMENT '阵法类型',
    level           INT             DEFAULT 1                           COMMENT '阵法等级 1-10',
    quality         TINYINT         DEFAULT 1                           COMMENT '品质 1-5',
    deployed        TINYINT(1)      DEFAULT 0                           COMMENT '是否部署',
    guardian        TINYINT(1)      DEFAULT 0                           COMMENT '是否护法',
    exp             BIGINT          DEFAULT 0                           COMMENT '升级经验',
    effects_json    JSON            DEFAULT NULL                        COMMENT '当前效果快照',
    learned_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '习得时间',

    -- === v2 新增字段 ===
    mastery_exp     BIGINT          DEFAULT 0                           COMMENT '熟练度经验',
    mastery_level   INT             DEFAULT 0                           COMMENT '熟练度等级 0-10',
    guardian_pet_id BIGINT          DEFAULT NULL                        COMMENT '守护灵兽ID',
    link_group      INT             DEFAULT 0                           COMMENT '联动组编号(0=未联动)',

    INDEX idx_player_id (player_id),
    INDEX idx_type (type),
    INDEX idx_guardian_pet (guardian_pet_id),
    INDEX idx_link_group (link_group)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家阵法表';

-- 护法记录表
CREATE TABLE IF NOT EXISTS guardian_tasks (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    guardian_id     BIGINT          NOT NULL                            COMMENT '护法玩家ID',
    beneficiary_id  BIGINT          NOT NULL                            COMMENT '受益玩家ID',
    formation_id    BIGINT          NOT NULL                            COMMENT '阵法ID',
    bonus_rate      DOUBLE          DEFAULT 0.0                         COMMENT '实际加成比例',
    success         TINYINT(1)      DEFAULT 0                           COMMENT '突破是否成功',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_guardian (guardian_id),
    INDEX idx_beneficiary (beneficiary_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='护法记录表';

-- ============================================================
-- 插入 8 种基础阵法模板
-- ============================================================
INSERT INTO formation_templates (id, name, type, quality, description, effects_json, learn_cost) VALUES
(1, '五行阵',   1, 3, '引金木水火土五行之力，大幅提升元素伤害',       '[{"type":"element_dmg","value":0.20}]',              5000),
(2, '八卦阵',   2, 3, '以八卦方位运转灵力，身法缥缈难以捉摸',         '[{"type":"dodge","value":0.15},{"type":"counter_rate","value":0.10}]', 5000),
(3, '诛仙阵',   3, 4, '上古杀伐之阵，对单体敌人造成毁灭性打击',       '[{"type":"single_target_dmg","value":0.35}]',         8000),
(4, '护山大阵', 4, 3, '汇聚天地灵气形成护盾，大幅提升全队防御',       '[{"type":"party_def","value":0.25}]',                 5000),
(5, '聚灵阵',   5, 2, '汇聚四方天地灵气，大幅加快修炼速度',           '[{"type":"cultivation_speed","value":0.10}]',         3000),
(6, '迷魂阵',   6, 3, '以幻术迷惑敌人心神，大幅降低其命中率',        '[{"type":"enemy_accuracy","value":-0.20}]',           4000),
(7, '传送阵',   7, 2, '打通空间壁垒，可在不同区域间瞬间传送',         '[{"type":"travel_speed","value":0.50}]',              3000),
(8, '天罡北斗阵',8,5, '引北斗七星之力加身，全面强化所有属性',        '[{"type":"all_stats","value":0.10}]',                 10000);

-- ============================================================
-- 示例数据
-- ============================================================
INSERT INTO player_formations (id, player_id, tmpl_id, name, type, level, quality, deployed, guardian, exp, effects_json, mastery_exp, mastery_level) VALUES
(1, 1, 1, '五行阵',   1, 3, 3, 1, 0, 1200, '[{"type":"element_dmg","value":0.26}]', 800,  2),
(2, 1, 2, '八卦阵',   2, 1, 3, 1, 0, 300,  '[{"type":"dodge","value":0.17},{"type":"counter_rate","value":0.11}]', 200, 1),
(3, 1, 8, '天罡北斗阵',8, 1, 5, 1, 0, 0,    '[{"type":"all_stats","value":0.11}]',      0,   0);
