-- ============================================================
-- 宗门战 / 跨服战系统 (金丹期解锁)
-- ============================================================

-- 宗门战赛季表
CREATE TABLE IF NOT EXISTS war_seasons (
    id              INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '赛季ID',
    season_name     VARCHAR(64)     NOT NULL                            COMMENT '赛季名称',
    start_time      DATETIME        NOT NULL                            COMMENT '赛季开始时间',
    end_time        DATETIME        NOT NULL                            COMMENT '赛季结束时间',
    status          VARCHAR(16)     DEFAULT 'upcoming'                  COMMENT '状态: upcoming/active/ended/settled',
    rules           JSON            DEFAULT NULL                        COMMENT '赛季规则配置',
    rewards         JSON            DEFAULT NULL                        COMMENT '赛季结算奖励',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_status (status),
    INDEX idx_start_time (start_time),
    INDEX idx_end_time (end_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门战赛季表';

-- 参战宗门记录表
CREATE TABLE IF NOT EXISTS war_clans (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    season_id       INT             NOT NULL                            COMMENT '赛季ID',
    clan_id         BIGINT          NOT NULL                            COMMENT '宗门ID',
    clan_name       VARCHAR(64)     NOT NULL                            COMMENT '宗门名称',
    member_count    INT             DEFAULT 0                           COMMENT '参战人数',
    total_power     BIGINT          DEFAULT 0                           COMMENT '宗门总战力',
    score           INT             DEFAULT 0                           COMMENT '赛季积分',
    rank            INT             DEFAULT 0                           COMMENT '排名',
    wins            INT             DEFAULT 0                           COMMENT '胜场数',
    losses          INT             DEFAULT 0                           COMMENT '负场数',
    draws           INT             DEFAULT 0                           COMMENT '平局数',
    joined_at       DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '加入赛季时间',
    eliminated_at   DATETIME        DEFAULT NULL                        COMMENT '淘汰时间',
    final_rank      INT             DEFAULT NULL                        COMMENT '最终排名',

    UNIQUE KEY uk_season_clan (season_id, clan_id),
    INDEX idx_season_id (season_id),
    INDEX idx_score (season_id, score),
    INDEX idx_rank (season_id, rank)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='参战宗门记录表';

-- 战斗场次表
CREATE TABLE IF NOT EXISTS war_battles (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '战斗ID',
    season_id       INT             NOT NULL                            COMMENT '赛季ID',
    round           INT             NOT NULL                            COMMENT '回合/轮次',
    bracket         VARCHAR(16)     DEFAULT 'group'                     COMMENT '赛段: group/quarterfinal/semifinal/final',
    clan_a_id       BIGINT          NOT NULL                            COMMENT '甲方宗门ID',
    clan_b_id       BIGINT          NOT NULL                            COMMENT '乙方宗门ID',
    clan_a_score    INT             DEFAULT 0                           COMMENT '甲方得分',
    clan_b_score    INT             DEFAULT 0                           COMMENT '乙方得分',
    winner_clan_id  BIGINT          DEFAULT NULL                        COMMENT '胜利方宗门ID',
    battle_data     JSON            DEFAULT NULL                        COMMENT '战斗详情JSON(每场对阵胜负记录)',
    status          VARCHAR(16)     DEFAULT 'scheduled'                 COMMENT '状态: scheduled/ongoing/finished/canceled',
    scheduled_at    DATETIME        DEFAULT NULL                        COMMENT '预定开战时间',
    started_at      DATETIME        DEFAULT NULL                        COMMENT '实际开始时间',
    finished_at     DATETIME        DEFAULT NULL                        COMMENT '结束时间',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_season_round (season_id, round),
    INDEX idx_bracket (season_id, bracket),
    INDEX idx_clan_a (clan_a_id),
    INDEX idx_clan_b (clan_b_id),
    INDEX idx_winner (winner_clan_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='战斗场次表';

-- 玩家个人战报表
CREATE TABLE IF NOT EXISTS war_player_records (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    season_id       INT             NOT NULL                            COMMENT '赛季ID',
    battle_id       BIGINT          NOT NULL                            COMMENT '战斗ID',
    player_id       BIGINT          NOT NULL                            COMMENT '玩家ID',
    clan_id         BIGINT          NOT NULL                            COMMENT '所属宗门ID',
    kill_count      INT             DEFAULT 0                           COMMENT '击败人数',
    death_count     INT             DEFAULT 0                           COMMENT '死亡次数',
    damage_dealt    BIGINT          DEFAULT 0                           COMMENT '造成伤害',
    damage_taken    BIGINT          DEFAULT 0                           COMMENT '承受伤害',
    heal_done       BIGINT          DEFAULT 0                           COMMENT '治疗量',
    mvp             TINYINT(1)      DEFAULT 0                           COMMENT '是否本场MVP',
    score_gained    INT             DEFAULT 0                           COMMENT '获得个人积分',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '记录时间',

    INDEX idx_season_id (season_id),
    INDEX idx_battle_id (battle_id),
    INDEX idx_player_id (player_id),
    INDEX idx_clan_id (clan_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家个人战报表';

-- ============================================================
-- 示例数据
-- ============================================================

INSERT INTO war_seasons (id, season_name, start_time, end_time, status, rules, rewards) VALUES
(1, 'S1·九州争霸',     '2026-07-01 00:00:00', '2026-07-28 23:59:59', 'upcoming',
    '{"group_size":8,"playoff_size":4,"match_interval_hours":24,"max_members_per_clan":50}',
    '{"rank_1":{"title":"九州霸主","spirit_stone":5000,"item_101":10},"rank_2":{"spirit_stone":3000,"item_101":5},"rank_3_4":{"spirit_stone":1000,"item_101":2}}'),
(2, 'S2·万宗会武',     '2026-08-01 00:00:00', '2026-08-28 23:59:59', 'upcoming',
    '{"group_size":16,"playoff_size":8,"match_interval_hours":24,"max_members_per_clan":50}',
    '{"rank_1":{"title":"万宗至尊","spirit_stone":10000,"item_101":20},"rank_2":{"spirit_stone":5000,"item_101":10},"rank_3_4":{"spirit_stone":2000,"item_101":5}}');

INSERT INTO war_clans (id, season_id, clan_id, clan_name, member_count, total_power, score, rank, wins, losses, draws, joined_at, final_rank) VALUES
(1, 1, 1, '青云宗', 45, 850000, 120, 1, 5, 1, 0, '2026-07-01 00:00:00', NULL),
(2, 1, 2, '玄天盟', 42, 780000, 100, 2, 4, 2, 0, '2026-07-01 00:00:00', NULL),
(3, 1, 3, '万剑门', 38, 720000, 85,  3, 4, 1, 1, '2026-07-01 00:00:00', NULL);

INSERT INTO war_battles (id, season_id, round, bracket, clan_a_id, clan_b_id, clan_a_score, clan_b_score, winner_clan_id, status, scheduled_at, started_at, finished_at) VALUES
(1, 1, 1, 'group',          1, 2, 3, 2, 1, 'finished',  '2026-07-03 20:00:00', '2026-07-03 20:00:00', '2026-07-03 20:30:00'),
(2, 1, 1, 'group',          2, 3, 2, 2, 2, 'finished',  '2026-07-05 20:00:00', '2026-07-05 20:00:00', '2026-07-05 20:25:00');

INSERT INTO war_player_records (id, season_id, battle_id, player_id, clan_id, kill_count, death_count, damage_dealt, damage_taken, heal_done, mvp, score_gained) VALUES
(1, 1, 1, 1, 1, 5, 0, 150000, 20000, 5000,  1, 50),
(2, 1, 1, 2, 1, 3, 1, 80000,  45000, 12000, 0, 30),
(3, 1, 1, 3, 2, 2, 2, 60000,  60000, 8000,  0, 20),
(4, 1, 2, 2, 2, 4, 1, 110000, 35000, 3000,  1, 45);

-- ============================================================
-- 战斗详情记录表(每场比赛的每一轮详细对战记录)
-- ============================================================
CREATE TABLE IF NOT EXISTS war_matches (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '比赛ID',
    battle_id       BIGINT          NOT NULL                            COMMENT '关联战斗场次ID',
    season_id       INT             NOT NULL                            COMMENT '赛季ID',
    round_num       INT             NOT NULL    DEFAULT 1               COMMENT '第几轮(1-3)',
    sect_a_id       BIGINT          NOT NULL                            COMMENT '甲方宗门ID',
    sect_b_id       BIGINT          NOT NULL                            COMMENT '乙方宗门ID',
    sect_a_player   BIGINT          DEFAULT NULL                        COMMENT '甲方出战玩家ID',
    sect_b_player   BIGINT          DEFAULT NULL                        COMMENT '乙方出战玩家ID',
    sect_a_role     VARCHAR(8)      DEFAULT 'attack'                    COMMENT '甲方本回合角色: attack/defend',
    sect_b_role     VARCHAR(8)      DEFAULT 'defend'                    COMMENT '乙方本回合角色: attack/defend',
    winner_id       BIGINT          DEFAULT NULL                        COMMENT '本回合胜者玩家ID',
    winning_sect_id BIGINT          DEFAULT NULL                        COMMENT '本回合胜方宗门ID',
    winner_score    INT             DEFAULT 0                           COMMENT '胜者得分',
    loser_score     INT             DEFAULT 0                           COMMENT '败者得分',
    battle_data     JSON            DEFAULT NULL                        COMMENT '战斗详细数据(技能/伤害/回合)',
    status          VARCHAR(16)     DEFAULT 'pending'                   COMMENT '状态: pending/fighting/finished',
    started_at      DATETIME        DEFAULT NULL                        COMMENT '开始时间',
    finished_at     DATETIME        DEFAULT NULL                        COMMENT '结束时间',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    INDEX idx_battle_id (battle_id),
    INDEX idx_season_id (season_id),
    INDEX idx_sect_a (sect_a_id),
    INDEX idx_sect_b (sect_b_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='战斗详情记录表(每轮对战)';

-- ============================================================
-- 灵脉福地表(世界地图上的修炼资源点)
-- ============================================================
CREATE TABLE IF NOT EXISTS spirit_veins (
    id              INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '灵脉ID',
    name            VARCHAR(64)     NOT NULL                            COMMENT '灵脉名称',
    quality         TINYINT         NOT NULL    DEFAULT 1               COMMENT '品质(1-5星)',
    region_id       VARCHAR(32)     NOT NULL                            COMMENT '所在区域ID',
    region_name     VARCHAR(64)     DEFAULT NULL                        COMMENT '区域名称',
    owner_sect_id   BIGINT          DEFAULT NULL                        COMMENT '当前占领宗门ID',
    owner_sect_name VARCHAR(64)     DEFAULT NULL                        COMMENT '当前占领宗门名称',
    occupied_at     DATETIME        DEFAULT NULL                        COMMENT '占领时间',
    cultivation_bonus DECIMAL(5,2)  DEFAULT 0.00                       COMMENT '修炼速度加成(%)',
    breakthrough_bonus DECIMAL(5,2) DEFAULT 0.00                       COMMENT '突破概率加成(%)',
    daily_yield     BIGINT          DEFAULT 0                           COMMENT '每日灵石产量',
    position_x      INT             DEFAULT 0                           COMMENT '地图X坐标',
    position_y      INT             DEFAULT 0                           COMMENT '地图Y坐标',
    description     VARCHAR(256)    DEFAULT NULL                        COMMENT '灵脉描述',
    status          VARCHAR(16)     DEFAULT 'idle'                      COMMENT '状态: idle/contested/occupied',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP
                                    ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    UNIQUE KEY uk_name (name),
    INDEX idx_quality (quality),
    INDEX idx_region (region_id),
    INDEX idx_owner (owner_sect_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='灵脉福地表(修炼资源点)';

-- ============================================================
-- 灵脉占领历史记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS vein_occupation_history (
    id              BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    vein_id         INT             NOT NULL                            COMMENT '灵脉ID',
    vein_name       VARCHAR(64)     NOT NULL                            COMMENT '灵脉名称',
    sect_id         BIGINT          NOT NULL                            COMMENT '占领宗门ID',
    sect_name       VARCHAR(64)     NOT NULL                            COMMENT '占领宗门名称',
    season_id       INT             DEFAULT NULL                        COMMENT '占领时的赛季ID',
    occupied_at     DATETIME        NOT NULL                            COMMENT '占领开始时间',
    lost_at         DATETIME        DEFAULT NULL                        COMMENT '失去占领时间',
    duration_hours  INT             DEFAULT 0                           COMMENT '持续占领时长(小时)',
    total_yield     BIGINT          DEFAULT 0                           COMMENT '占领期间总产量',

    INDEX idx_vein_id (vein_id),
    INDEX idx_sect_id (sect_id),
    INDEX idx_season_id (season_id),
    INDEX idx_occupied_at (occupied_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='灵脉占领历史记录表';

-- ============================================================
-- 示例灵脉数据(6条分布在不同区域)
-- ============================================================
INSERT INTO spirit_veins (id, name, quality, region_id, region_name, cultivation_bonus, breakthrough_bonus, daily_yield, position_x, position_y, description, status) VALUES
(1, '青龙灵脉', 5, 'region_east', '东海之滨', 25.00, 10.00, 5000, 850, 300, '上古青龙陨落之地，灵力充沛，修炼一日千里', 'idle'),
(2, '玄武灵脉', 4, 'region_north', '北极冰原', 20.00, 6.00, 3000, 200, 150, '冰川之下隐藏的灵脉，蕴含冰属性灵力', 'idle'),
(3, '朱雀灵脉', 4, 'region_south', '南疆火域', 20.00, 6.00, 3000, 600, 800, '地火交融之处，火属性灵力旺盛', 'idle'),
(4, '白虎灵脉', 3, 'region_west', '西荒戈壁', 15.00, 3.00, 2000, 100, 650, '荒漠中的灵脉绿洲，金灵气充盈', 'idle'),
(5, '青木灵脉', 2, 'region_center', '中州森林', 10.00, 0.00, 1000, 450, 450, '古老森林中的灵脉，生机勃勃', 'idle'),
(6, '地泉灵脉', 1, 'region_east', '东海之滨', 5.00, 0.00, 500, 800, 250, '小型地泉灵脉，适合新晋宗门', 'idle');
