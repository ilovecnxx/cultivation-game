-- ===================================================================
-- 修仙游戏核心数据库 - 表结构
-- InnoDB / utf8mb4 / 合理索引 / JSON 灵活属性 / 时间戳
-- ===================================================================

-- 建议先创建数据库（如果不存在）：
-- CREATE DATABASE IF NOT EXISTS cultivation_game
--     DEFAULT CHARACTER SET utf8mb4
--     DEFAULT COLLATE utf8mb4_unicode_ci;
-- USE cultivation_game;

-- ===================================================================
-- 一、核心玩家表
-- ===================================================================

-- 1. 玩家核心表
CREATE TABLE players (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '玩家ID',
    uid                     VARCHAR(32)     NOT NULL                            COMMENT '唯一标识(接入平台/登录用)',
    nickname                VARCHAR(64)     NOT NULL                            COMMENT '昵称',
    realm_id                INT             DEFAULT 1                           COMMENT '境界ID(realm_config.id)',
    realm_level             INT             DEFAULT 1                           COMMENT '境界内层级(如练气第几层)',
    exp                     BIGINT          DEFAULT 0                           COMMENT '修为/当前经验值',
    spirit_root             VARCHAR(64)     DEFAULT NULL                        COMMENT '灵根(JSON, 如{"roots":["火灵根"],"quality":"上品"})',
    hp                      BIGINT          DEFAULT 100                         COMMENT '当前气血',
    mp                      BIGINT          DEFAULT 100                         COMMENT '当前真元',
    attack                  BIGINT          DEFAULT 10                          COMMENT '攻击力',
    defense                 BIGINT          DEFAULT 10                          COMMENT '防御力',
    speed                   INT             DEFAULT 100                         COMMENT '速度(影响出手顺序)',
    cultivation_technique_id INT            DEFAULT NULL                        COMMENT '主修功法ID(technique_templates.id)',
    auxiliary_technique_ids VARCHAR(128)    DEFAULT NULL                        COMMENT '辅修功法ID列表(JSON数组, 如[2,3,5])',
    money                   BIGINT          DEFAULT 0                           COMMENT '灵石(非绑定流通货币)',
    bind_money              BIGINT          DEFAULT 0                           COMMENT '绑定灵石(不可交易)',
    immortal_jade           INT             DEFAULT 0                           COMMENT '仙玉(充值货币)',
    vip_level               INT             DEFAULT 0                           COMMENT 'VIP等级',
    total_play_time         BIGINT          DEFAULT 0                           COMMENT '累计在线时长(秒)',
    last_login_at           DATETIME        DEFAULT NULL                        COMMENT '最后登录时间',
    last_logout_at          DATETIME        DEFAULT NULL                        COMMENT '最后登出时间',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    UNIQUE KEY uk_uid (uid),
    INDEX idx_nickname (nickname),
    INDEX idx_realm (realm_id, realm_level),
    INDEX idx_vip (vip_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家核心表';


-- 2. 玩家详细属性表
CREATE TABLE player_attributes (
    player_id               BIGINT          NOT NULL        PRIMARY KEY         COMMENT '玩家ID(与players一一对应)',
    max_hp                  BIGINT          DEFAULT 1000                        COMMENT '最大气血上限',
    max_mp                  BIGINT          DEFAULT 1000                        COMMENT '最大真元上限',
    hp_regen                BIGINT          DEFAULT 5                           COMMENT '气血自动回复(每 tick)',
    mp_regen                BIGINT          DEFAULT 5                           COMMENT '真元自动回复(每 tick)',
    crit_rate               DECIMAL(5,2)    DEFAULT 5.00                        COMMENT '暴击率(%)',
    crit_dmg                DECIMAL(5,2)    DEFAULT 150.00                      COMMENT '暴击伤害加成(%)',
    dodge_rate              DECIMAL(5,2)    DEFAULT 5.00                        COMMENT '闪避率(%)',
    hit_rate                DECIMAL(5,2)    DEFAULT 95.00                       COMMENT '命中率(%)',
    damage_reduction        DECIMAL(5,2)    DEFAULT 0.00                        COMMENT '伤害减免(%)',
    cultivation_speed       DECIMAL(6,2)    DEFAULT 100.00                      COMMENT '修炼速度倍率(%)',
    fortune                 INT             DEFAULT 0                           COMMENT '机缘(影响奇遇/掉落)',
    comprehension           INT             DEFAULT 0                           COMMENT '悟性(影响技能学习速度)',
    charm                   INT             DEFAULT 0                           COMMENT '魅力(影响NPC好感/交易)',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    CONSTRAINT fk_attr_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家详细属性表(1:1扩展)';


-- 3. 背包表
CREATE TABLE inventory (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id               BIGINT          NOT NULL                            COMMENT '玩家ID',
    item_id                 INT             NOT NULL                            COMMENT '物品模板ID(item_templates.id)',
    quantity                INT             DEFAULT 1                           COMMENT '数量',
    slot                    INT             DEFAULT 0                           COMMENT '背包格子序号(0表示自动分配)',
    extra_data              JSON            DEFAULT NULL                        COMMENT '额外数据(耐久度/随机词条等)',
    is_equipped             TINYINT         DEFAULT 0                           COMMENT '是否已装备(0否/1是)',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player (player_id),
    INDEX idx_player_slot (player_id, slot),
    INDEX idx_item (item_id),
    CONSTRAINT fk_inv_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='背包表(含仓库)';


-- 4. 装备表
CREATE TABLE equipment (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '装备记录ID',
    player_id               BIGINT          NOT NULL                            COMMENT '玩家ID',
    slot_type               TINYINT         NOT NULL                            COMMENT '装备位:1武器/2头盔/3衣服/4护腕/5腰带/6裤子/7靴子/8项链/9戒指/10法宝',
    item_id                 INT             NOT NULL                            COMMENT '物品模板ID(item_templates.id)',
    base_attr               JSON            DEFAULT NULL                        COMMENT '基础属性KV(如{"attack":25,"defense":10})',
    enhance_level           INT             DEFAULT 0                           COMMENT '强化等级',
    gems                    JSON            DEFAULT NULL                        COMMENT '镶嵌宝石ID数组(如[1,3,5])',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player (player_id),
    UNIQUE KEY uk_player_slot (player_id, slot_type),
    CONSTRAINT fk_equip_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='装备表(已装备的武器/防具/饰品等)';


-- 5. 玩家已学功法表
CREATE TABLE techniques (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id               BIGINT          NOT NULL                            COMMENT '玩家ID',
    technique_id            INT             NOT NULL                            COMMENT '功法模板ID(technique_templates.id)',
    level                   INT             DEFAULT 1                           COMMENT '当前功法等级',
    exp                     BIGINT          DEFAULT 0                           COMMENT '当前功法经验',
    is_equipped             TINYINT         DEFAULT 0                           COMMENT '是否已装备(主修=1/辅修=2/未装备=0)',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player (player_id),
    UNIQUE KEY uk_player_technique (player_id, technique_id),
    CONSTRAINT fk_tech_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家已学功法表';


-- 6. 玩家已学技能表
CREATE TABLE skills (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    player_id               BIGINT          NOT NULL                            COMMENT '玩家ID',
    skill_id                INT             NOT NULL                            COMMENT '技能模板ID(skill_templates.id)',
    level                   INT             DEFAULT 1                           COMMENT '技能等级',
    is_equipped             TINYINT         DEFAULT 0                           COMMENT '是否已装备到技能栏(0否/1是)',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_player (player_id),
    UNIQUE KEY uk_player_skill (player_id, skill_id),
    CONSTRAINT fk_skill_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家已学技能表';


-- ===================================================================
-- 二、社交系统表
-- ===================================================================

-- 7. 好友关系表
CREATE TABLE friends (
    player_id               BIGINT          NOT NULL                            COMMENT '玩家ID(主动方)',
    friend_id               BIGINT          NOT NULL                            COMMENT '好友ID(被动方)',
    status                  TINYINT         DEFAULT 0                           COMMENT '关系状态:0申请中/1已好友/2黑名单',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    PRIMARY KEY (player_id, friend_id),
    INDEX idx_player (player_id),
    INDEX idx_friend (friend_id),
    INDEX idx_status (status),
    CONSTRAINT fk_friend_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    CONSTRAINT fk_friend_target FOREIGN KEY (friend_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友关系表';


-- 8. 宗门表
CREATE TABLE sects (
    id                      INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '宗门ID',
    name                    VARCHAR(64)     NOT NULL                            COMMENT '宗门名称',
    leader_id               BIGINT          NOT NULL                            COMMENT '宗主玩家ID',
    level                   INT             DEFAULT 1                           COMMENT '宗门等级',
    member_count            INT             DEFAULT 1                           COMMENT '当前成员数',
    max_members             INT             DEFAULT 50                          COMMENT '最大成员数上限',
    notice                  TEXT            DEFAULT NULL                        COMMENT '宗门公告',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_name (name),
    INDEX idx_leader (leader_id),
    CONSTRAINT fk_sect_leader FOREIGN KEY (leader_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门表';


-- 9. 宗门成员表
CREATE TABLE sect_members (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '记录ID',
    sect_id                 INT             NOT NULL                            COMMENT '宗门ID',
    player_id               BIGINT          NOT NULL                            COMMENT '玩家ID',
    position                TINYINT         DEFAULT 4                           COMMENT '职位:1宗主/2长老/3精英/4成员',
    contribution            BIGINT          DEFAULT 0                           COMMENT '宗门贡献度',
    joined_at               DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '加入时间',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_sect (sect_id),
    INDEX idx_player (player_id),
    UNIQUE KEY uk_sect_player (sect_id, player_id),
    CONSTRAINT fk_sm_sect FOREIGN KEY (sect_id) REFERENCES sects(id) ON DELETE CASCADE,
    CONSTRAINT fk_sm_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门成员表';


-- ===================================================================
-- 三、邮件/交易系统表
-- ===================================================================

-- 10. 邮件表
CREATE TABLE mail (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '邮件ID',
    sender_id               BIGINT          DEFAULT 0                           COMMENT '发件人ID(0=系统,>0=玩家ID)',
    receiver_id             BIGINT          NOT NULL                            COMMENT '收件人玩家ID',
    title                   VARCHAR(128)    NOT NULL                            COMMENT '邮件标题',
    content                 TEXT            DEFAULT NULL                        COMMENT '邮件正文',
    has_attachment          TINYINT         DEFAULT 0                           COMMENT '是否有附件(0无/1有)',
    item_id                 INT             DEFAULT NULL                        COMMENT '附件物品模板ID',
    item_quantity           INT             DEFAULT 0                           COMMENT '附件物品数量',
    is_read                 TINYINT         DEFAULT 0                           COMMENT '是否已读(0未读/1已读)',
    is_claimed              TINYINT         DEFAULT 0                           COMMENT '附件是否已领取(0未领/1已领)',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '发送时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_receiver (receiver_id, is_read),
    INDEX idx_sender (sender_id),
    INDEX idx_claimed (receiver_id, is_claimed),
    CONSTRAINT fk_mail_receiver FOREIGN KEY (receiver_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='游戏邮件表(含系统邮件+附件)';


-- 11. 交易行上架表
CREATE TABLE trade_listings (
    id                      BIGINT          AUTO_INCREMENT  PRIMARY KEY         COMMENT '上架记录ID',
    seller_id               BIGINT          NOT NULL                            COMMENT '卖家玩家ID',
    item_id                 INT             NOT NULL                            COMMENT '物品模板ID',
    quantity                INT             DEFAULT 1                           COMMENT '出售数量',
    unit_price              BIGINT          NOT NULL                            COMMENT '单价',
    currency_type           TINYINT         DEFAULT 0                           COMMENT '货币类型:0灵石/1仙玉',
    status                  TINYINT         DEFAULT 0                           COMMENT '状态:0上架中/1已售出/2已下架',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_seller (seller_id),
    INDEX idx_item (item_id),
    INDEX idx_status (status, created_at),
    CONSTRAINT fk_trade_seller FOREIGN KEY (seller_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='交易行上架表(玩家间交易)';


-- ===================================================================
-- 四、PVP / 排行表
-- ===================================================================

-- 12. PVP 排行榜
CREATE TABLE pvp_rankings (
    player_id               BIGINT          NOT NULL                            COMMENT '玩家ID',
    season_id               INT             NOT NULL                            COMMENT '赛季ID',
    `rank`                  INT             NOT NULL                            COMMENT '当前排名',
    rating                  INT             DEFAULT 1000                        COMMENT '积分(ELO)',
    win_count               INT             DEFAULT 0                           COMMENT '胜场数',
    lose_count              INT             DEFAULT 0                           COMMENT '负场数',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '首次进入排行时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    PRIMARY KEY (player_id, season_id),
    INDEX idx_rank (season_id, `rank`),
    INDEX idx_rating (season_id, rating DESC),
    CONSTRAINT fk_pvp_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='PVP排行榜(按赛季)';


-- ===================================================================
-- 五、管理后台表
-- ===================================================================

-- 13. 管理员账号表
CREATE TABLE admin_users (
    id                      INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '管理员ID',
    username                VARCHAR(64)     NOT NULL                            COMMENT '管理员用户名',
    password_hash           VARCHAR(256)    NOT NULL                            COMMENT '密码bcrypt哈希',
    role                    VARCHAR(32)     DEFAULT 'operator'                  COMMENT '角色:super_admin/admin/operator/auditor',
    permissions             JSON            DEFAULT NULL                        COMMENT '细粒度权限(JSON数组或对象)',
    last_login_at           DATETIME        DEFAULT NULL                        COMMENT '最后登录时间',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    UNIQUE KEY uk_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='管理员账号表';


-- ===================================================================
-- 六、配置/模板表（支撑数据）
-- ===================================================================

-- 14. 境界配置表
CREATE TABLE realm_config (
    id                      INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '境界等级ID',
    realm_name              VARCHAR(32)     NOT NULL                            COMMENT '境界完整名称(如:练气一层/筑基初期)',
    realm_group             TINYINT         NOT NULL                            COMMENT '境界大类:1练气/2筑基/3金丹/4元婴/5化神/6合体/7大乘/8渡劫/9飞升',
    level                   INT             DEFAULT 1                           COMMENT '当前大境界内层级(从1开始)',
    upgrade_exp             BIGINT          NOT NULL                            COMMENT '升至下一级所需修为(0=已满级)',
    base_hp                 BIGINT          DEFAULT 0                           COMMENT '晋升后气血基础加成',
    base_mp                 BIGINT          DEFAULT 0                           COMMENT '晋升后真元基础加成',
    base_attack             BIGINT          DEFAULT 0                           COMMENT '晋升后攻击基础加成',
    base_defense            BIGINT          DEFAULT 0                           COMMENT '晋升后防御基础加成',
    base_speed              INT             DEFAULT 0                           COMMENT '晋升后速度基础加成',
    breakthrough_item_id    INT             DEFAULT NULL                        COMMENT '突破所需物品ID(如筑基丹)',
    description             VARCHAR(255)    DEFAULT NULL                        COMMENT '境界描述',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',

    UNIQUE KEY uk_realm_level (realm_group, level),
    INDEX idx_realm_group (realm_group)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='境界配置表(定义各大境界与各层属性)';


-- 15. 物品模板表
CREATE TABLE item_templates (
    id                      INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '物品ID',
    name                    VARCHAR(64)     NOT NULL                            COMMENT '物品名称',
    item_type               TINYINT         NOT NULL                            COMMENT '类型:1消耗品/2装备/3材料/4功法书/5任务/6其他',
    quality                 TINYINT         DEFAULT 1                           COMMENT '品质:1凡品/2下品/3中品/4上品/5极品/6仙品',
    use_level               INT             DEFAULT 1                           COMMENT '使用所需境界等级',
    sell_price              BIGINT          DEFAULT 0                           COMMENT '商店出售价格(灵石)',
    bind_type               TINYINT         DEFAULT 0                           COMMENT '绑定类型:0不绑定/1拾取绑定/2使用绑定/3装备绑定',
    max_stack               INT             DEFAULT 1                           COMMENT '最大堆叠数量',
    cooldown                INT             DEFAULT 0                           COMMENT '使用冷却时间(秒)',
    description             TEXT            DEFAULT NULL                        COMMENT '物品描述',
    extra_attributes        JSON            DEFAULT NULL                        COMMENT '额外属性(效果值/装备基础属性等KV)',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_type (item_type),
    INDEX idx_quality (quality),
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='物品模板表(所有物品的定义)';


-- 16. 功法模板表
CREATE TABLE technique_templates (
    id                      INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '功法ID',
    name                    VARCHAR(64)     NOT NULL                            COMMENT '功法名称',
    grade                   TINYINT         DEFAULT 1                           COMMENT '功法品级:1黄阶/2玄阶/3地阶/4天阶/5仙阶',
    element_type            TINYINT         DEFAULT NULL                        COMMENT '五行属性:1金/2木/3水/4火/5土/0无',
    max_level               INT             DEFAULT 9                           COMMENT '功法最高等级',
    level_required          INT             DEFAULT 1                           COMMENT '修炼所需境界等级',
    cultivation_bonus       DECIMAL(6,2)    DEFAULT 0.00                        COMMENT '修炼速度加成(%)',
    hp_bonus                BIGINT          DEFAULT 0                           COMMENT '气血加成',
    mp_bonus                BIGINT          DEFAULT 0                           COMMENT '真元加成',
    attack_bonus            BIGINT          DEFAULT 0                           COMMENT '攻击加成',
    defense_bonus           BIGINT          DEFAULT 0                           COMMENT '防御加成',
    speed_bonus             INT             DEFAULT 0                           COMMENT '速度加成',
    description             TEXT            DEFAULT NULL                        COMMENT '功法描述',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_grade (grade),
    INDEX idx_element (element_type),
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='功法模板表(修炼功法定义)';


-- 17. 技能模板表
CREATE TABLE skill_templates (
    id                      INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '技能ID',
    name                    VARCHAR(64)     NOT NULL                            COMMENT '技能名称',
    skill_type              TINYINT         NOT NULL                            COMMENT '技能类型:1攻击/2防御/3辅助/4身法/5被动',
    element_type            TINYINT         DEFAULT NULL                        COMMENT '五行属性:1金/2木/3水/4火/5土/0无',
    level_required          INT             DEFAULT 1                           COMMENT '所需境界等级',
    mp_cost                 BIGINT          DEFAULT 0                           COMMENT '真元消耗',
    cooldown                INT             DEFAULT 0                           COMMENT '冷却时间(秒)',
    target_type             TINYINT         DEFAULT 0                           COMMENT '目标类型:0敌方单体/1敌方全体/2己方单体/3己方全体/4自身',
    damage_multiplier       DECIMAL(5,2)    DEFAULT 1.00                        COMMENT '伤害倍率(攻击系数)',
    description             TEXT            DEFAULT NULL                        COMMENT '技能描述',
    extra_effects           JSON            DEFAULT NULL                        COMMENT '额外效果(如dot/减益/增益等)',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    INDEX idx_type (skill_type),
    INDEX idx_element (element_type),
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='技能模板表(技能/法术定义)';


-- 18. 系统配置表
CREATE TABLE system_config (
    id                      INT             AUTO_INCREMENT  PRIMARY KEY         COMMENT '配置ID',
    config_key              VARCHAR(64)     NOT NULL                            COMMENT '配置键(唯一)',
    config_value            TEXT            NOT NULL                            COMMENT '配置值',
    description             VARCHAR(255)    DEFAULT NULL                        COMMENT '配置说明',
    created_at              DATETIME        DEFAULT CURRENT_TIMESTAMP           COMMENT '创建时间',
    updated_at              DATETIME        DEFAULT CURRENT_TIMESTAMP
                                            ON UPDATE CURRENT_TIMESTAMP         COMMENT '更新时间',

    UNIQUE KEY uk_config_key (config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置表(服务器参数/倍率等)';
