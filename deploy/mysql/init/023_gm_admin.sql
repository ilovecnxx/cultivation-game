-- ============================================================
-- GM 管理后台系统 - 数据表
-- 版本: v1.0.0
-- 管理员管理 / 操作日志 / 公告 / 封禁
-- ============================================================

-- 1. GM 管理员表
CREATE TABLE IF NOT EXISTS gm_admins (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY         COMMENT '管理员唯一ID',
    username        VARCHAR(64) NOT NULL                      COMMENT '管理员用户名（唯一）',
    password_hash   VARCHAR(255) NOT NULL                     COMMENT '密码哈希（bcrypt）',
    role            TINYINT NOT NULL DEFAULT 3                COMMENT '角色：1=超级管理员 2=运营 3=观察者',
    status          TINYINT NOT NULL DEFAULT 1                COMMENT '状态：1=启用 0=禁用',
    last_login_at   TIMESTAMP NULL DEFAULT NULL               COMMENT '最后登录时间',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
                    ON UPDATE CURRENT_TIMESTAMP               COMMENT '更新时间',

    UNIQUE KEY uk_username (username),
    INDEX idx_role (role),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 管理员账号表';

-- 2. GM 操作日志表
CREATE TABLE IF NOT EXISTS gm_operation_logs (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY         COMMENT '日志ID',
    admin_id        BIGINT NOT NULL                           COMMENT '操作管理员ID',
    action_type     VARCHAR(64) NOT NULL                      COMMENT '操作类型（如 ban_player, send_item, edit_attribute）',
    target_type     VARCHAR(64) NOT NULL DEFAULT ''           COMMENT '操作目标类型（如 player, announcement, system）',
    target_id       BIGINT NOT NULL DEFAULT 0                 COMMENT '操作目标ID',
    detail          JSON DEFAULT NULL                         COMMENT '操作详情（JSON格式）',
    ip_address      VARCHAR(45) NOT NULL DEFAULT ''           COMMENT '操作IP地址',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '操作时间',

    INDEX idx_admin_id (admin_id),
    INDEX idx_action_type (action_type),
    INDEX idx_created_at (created_at),
    INDEX idx_target_type_target_id (target_type, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 操作日志表';

-- 3. GM 公告表
CREATE TABLE IF NOT EXISTS gm_announcements (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY         COMMENT '公告ID',
    admin_id        BIGINT NOT NULL                           COMMENT '发布管理员ID',
    title           VARCHAR(128) NOT NULL                     COMMENT '公告标题',
    content         TEXT NOT NULL                             COMMENT '公告内容',
    type            TINYINT NOT NULL DEFAULT 1                COMMENT '公告类型：1=系统公告 2=世界公告 3=个人消息',
    target_player_id BIGINT NULL DEFAULT NULL                 COMMENT '目标玩家ID（个人消息时使用）',
    sent_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发送时间',
    expire_at       TIMESTAMP NULL DEFAULT NULL               COMMENT '过期时间（NULL表示永不过期）',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    INDEX idx_admin_id (admin_id),
    INDEX idx_type (type),
    INDEX idx_target_player_id (target_player_id),
    INDEX idx_sent_at (sent_at),
    INDEX idx_expire_at (expire_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 公告表';

-- 4. GM 封禁表
CREATE TABLE IF NOT EXISTS gm_bans (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY         COMMENT '封禁记录ID',
    player_id       BIGINT NOT NULL                           COMMENT '被封禁玩家ID',
    admin_id        BIGINT NOT NULL                           COMMENT '操作管理员ID',
    reason          TEXT NOT NULL                             COMMENT '封禁原因',
    ban_type        TINYINT NOT NULL DEFAULT 1                COMMENT '封禁类型：1=禁言 2=临时封号 3=永久封号',
    starts_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '封禁开始时间',
    ends_at         TIMESTAMP NULL DEFAULT NULL               COMMENT '封禁结束时间（NULL表示永久）',
    status          TINYINT NOT NULL DEFAULT 1                COMMENT '状态：1=生效中 0=已解封',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    INDEX idx_player_id (player_id),
    INDEX idx_admin_id (admin_id),
    INDEX idx_status (status),
    INDEX idx_ban_type (ban_type),
    INDEX idx_starts_at (starts_at),
    INDEX idx_ends_at (ends_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 封禁记录表';
