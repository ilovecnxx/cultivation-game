-- 008: 本命法宝系统
-- 金丹期(21层+)解锁，每个玩家绑定1件本命法宝
-- 品质: 1凡品 2灵品 3仙品 4神品 5混沌

CREATE TABLE IF NOT EXISTS player_artifacts (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '法宝ID',
    player_id     BIGINT       NOT NULL UNIQUE COMMENT '玩家ID（唯一，每人1件）',
    name          VARCHAR(64)  NOT NULL COMMENT '法宝名称',
    quality       INT          DEFAULT 1 COMMENT '品质: 1凡品 2灵品 3仙品 4神品 5混沌',
    level         INT          DEFAULT 1 COMMENT '等级 1-100',
    exp           BIGINT       DEFAULT 0 COMMENT '经验值',
    attack_bonus  BIGINT       DEFAULT 0 COMMENT '攻击加成',
    defense_bonus BIGINT       DEFAULT 0 COMMENT '防御加成',
    hp_bonus      BIGINT       DEFAULT 0 COMMENT '气血加成',
    skill_id      INT          DEFAULT 0 COMMENT '法宝技能ID',
    bound_at      DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '绑定时间',
    INDEX idx_player (player_id),
    INDEX idx_quality (quality)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='本命法宝表';
