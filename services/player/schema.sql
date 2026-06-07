-- 修仙游戏 玩家服务 数据库表结构
-- Database: cultivation

CREATE DATABASE IF NOT EXISTS cultivation DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE cultivation;

-- 玩家表
CREATE TABLE IF NOT EXISTS players (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id       VARCHAR(64)  NOT NULL COMMENT '用户ID（来自账号服务）',
    name          VARCHAR(32)  NOT NULL COMMENT '角色名',
    level         INT          NOT NULL DEFAULT 1 COMMENT '等级',
    realm         INT          NOT NULL DEFAULT 1 COMMENT '境界: 1凡人 2练气 3筑基 4金丹 5元婴 6化神 7合体 8大乘 9渡劫',
    spirit_root   INT          NOT NULL DEFAULT 0 COMMENT '灵根: 0无 1金 2木 3水 4火 5土 6风 7雷 8冰',
    hp            BIGINT       NOT NULL DEFAULT 100 COMMENT '当前气血',
    max_hp        BIGINT       NOT NULL DEFAULT 100 COMMENT '最大气血',
    mp            BIGINT       NOT NULL DEFAULT 50 COMMENT '当前真元',
    max_mp        BIGINT       NOT NULL DEFAULT 50 COMMENT '最大真元',
    attack        BIGINT       NOT NULL DEFAULT 10 COMMENT '攻击力',
    defense       BIGINT       NOT NULL DEFAULT 5 COMMENT '防御力',
    spirit_power  BIGINT       NOT NULL DEFAULT 0 COMMENT '修为',
    experience    BIGINT       NOT NULL DEFAULT 0 COMMENT '经验',
    gold          BIGINT       NOT NULL DEFAULT 0 COMMENT '灵石',
    bound_gold    BIGINT       NOT NULL DEFAULT 0 COMMENT '绑定灵石',
    jade          BIGINT       NOT NULL DEFAULT 0 COMMENT '仙玉',
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_id (user_id),
    UNIQUE KEY uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家角色';

-- 物品模板表（静态配置）
CREATE TABLE IF NOT EXISTS items (
    id               BIGINT AUTO_INCREMENT PRIMARY KEY,
    name             VARCHAR(64)  NOT NULL COMMENT '物品名称',
    type             INT          NOT NULL COMMENT '类型: 1武器 2头盔 3衣服 4护腕 5腰带 6裤子 7鞋子 8项链 9戒指 10丹药 11材料 12功法 13消耗品',
    quality          INT          NOT NULL DEFAULT 1 COMMENT '品质: 1凡品 2下品 3中品 4上品 5极品 6仙品',
    required_level   INT          NOT NULL DEFAULT 1 COMMENT '需求等级',
    required_realm   INT          NOT NULL DEFAULT 1 COMMENT '需求境界',
    description      VARCHAR(256) DEFAULT '' COMMENT '描述',
    max_stack        INT          NOT NULL DEFAULT 1 COMMENT '最大堆叠数',
    base_attack      BIGINT       NOT NULL DEFAULT 0 COMMENT '基础攻击',
    base_defense     BIGINT       NOT NULL DEFAULT 0 COMMENT '基础防御',
    base_hp          BIGINT       NOT NULL DEFAULT 0 COMMENT '基础气血加成',
    base_mp          BIGINT       NOT NULL DEFAULT 0 COMMENT '基础真元加成',
    use_effect       VARCHAR(128) DEFAULT '' COMMENT '使用效果(JSON或 hp:50,mp:30 格式)',
    sell_price       BIGINT       NOT NULL DEFAULT 0 COMMENT '出售价格(灵石)',
    sell_price_bound BIGINT       NOT NULL DEFAULT 0 COMMENT '出售价格(绑定灵石)',
    created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_type (type),
    INDEX idx_quality (quality)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='物品模板';

-- 背包物品表
CREATE TABLE IF NOT EXISTS inventory_items (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    player_id   BIGINT   NOT NULL COMMENT '玩家ID',
    item_id     BIGINT   NOT NULL COMMENT '物品模板ID',
    quantity    INT      NOT NULL DEFAULT 1 COMMENT '数量',
    slot_index  INT      NOT NULL DEFAULT 0 COMMENT '背包格子索引',
    is_equipped TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已装备',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_player (player_id),
    INDEX idx_item (item_id),
    INDEX idx_player_item (player_id, item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='背包物品';

-- 装备表（强化信息）
CREATE TABLE IF NOT EXISTS equipments (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY,
    player_id         BIGINT   NOT NULL COMMENT '玩家ID',
    slot              INT      NOT NULL COMMENT '槽位: 1武器 2头盔 3衣服 4护腕 5腰带 6裤子 7鞋子 8项链 9戒指',
    inventory_item_id BIGINT   NOT NULL COMMENT '关联的背包物品ID',
    item_id           BIGINT   NOT NULL COMMENT '物品模板ID',
    level             INT      NOT NULL DEFAULT 0 COMMENT '强化等级',
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_player (player_id),
    UNIQUE KEY uk_inventory_item (inventory_item_id),
    UNIQUE KEY uk_player_slot (player_id, slot)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='已装备';

-- 插入示例物品模板
INSERT INTO items (id, name, type, quality, required_level, required_realm, description, max_stack, base_attack, base_defense, base_hp, base_mp, use_effect, sell_price) VALUES
-- 武器
(1001, '竹剑',       1, 1, 1, 1, '入门级法器', 1, 15, 0, 0, 0, '', 10),
(1002, '青锋剑',     1, 2, 5, 1, '精铁打造的剑', 1, 30, 0, 0, 0, '', 50),
(1003, '玄铁重剑',   1, 3, 10, 2, '玄铁铸造', 1, 60, 5, 0, 0, '', 200),
-- 防具
(2001, '布衣',       3, 1, 1, 1, '普通布衣', 1, 0, 5, 10, 0, '', 5),
(2002, '皮甲',       3, 2, 5, 1, '兽皮制作', 1, 0, 12, 25, 0, '', 40),
-- 丹药
(3001, '回血丹',     10, 1, 1, 1, '恢复50点气血', 99, 0, 0, 50, 0, 'hp:50', 5),
(3002, '回元丹',     10, 1, 1, 1, '恢复30点真元', 99, 0, 0, 0, 30, 'mp:30', 5),
(3003, '修为丹',     10, 2, 5, 1, '增加10修为', 99, 10, 0, 0, 0, 'spirit:10', 20),
-- 材料
(4001, '强化石',     11, 2, 1, 1, '装备强化材料（1-9级）', 999, 0, 0, 0, 0, '', 10),
(4002, '高级强化石', 11, 4, 30, 3, '装备强化材料（10级+）', 999, 0, 0, 0, 0, '', 50);
