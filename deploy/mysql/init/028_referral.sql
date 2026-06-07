-- ============================================================
-- 玩家邀请/推荐系统 - 数据表
-- 版本: v1.0.0
-- 允许玩家生成邀请码拉新，邀请者在被邀请者达到一定境界时获得奖励
-- ============================================================

-- 1. 玩家邀请码表
-- 每个活跃玩家最多可拥有一个邀请码（唯一），用于邀请新玩家注册
CREATE TABLE IF NOT EXISTS player_invite_codes (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    player_id   BIGINT NOT NULL                  COMMENT '玩家ID，关联players.id',
    invite_code VARCHAR(16) NOT NULL             COMMENT '8位字母数字邀请码，全局唯一',
    times_used  INT DEFAULT 0                    COMMENT '已使用次数',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    UNIQUE KEY uk_player_id (player_id)          COMMENT '一个玩家只有一个邀请码',
    UNIQUE KEY uk_invite_code (invite_code)      COMMENT '邀请码全局唯一',
    INDEX idx_player_id (player_id)              COMMENT '玩家ID索引',
    INDEX idx_invite_code (invite_code)          COMMENT '邀请码查询索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家邀请码表';

-- 2. 推荐记录表
-- 记录每个被邀请者的归属关系及其境界进度，用于结算邀请奖励
CREATE TABLE IF NOT EXISTS referral_records (
    id                    BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    inviter_id            BIGINT NOT NULL           COMMENT '邀请者玩家ID',
    invitee_id            BIGINT NOT NULL           COMMENT '被邀请者玩家ID',
    invitee_realm_reached TINYINT DEFAULT 0         COMMENT '被邀请者已达成的境界档位(里程碑)，bitmask方式: bit0=筑基, bit1=元婴, bit2=化神, bit3=大乘',
    reward_claimed        TINYINT DEFAULT 0         COMMENT '已领取的奖励档位，同bitmask方式',
    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                          ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',

    UNIQUE KEY uk_invitee_id (invitee_id)          COMMENT '每个被邀请者只有一条记录',
    INDEX idx_inviter_id (inviter_id)              COMMENT '邀请者索引（查询我的邀请列表）',
    INDEX idx_invitee_id (invitee_id)              COMMENT '被邀请者索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='推荐记录表';
