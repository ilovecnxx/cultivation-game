-- ============================================================
-- 反作弊与滥用预防系统 - 数据表
-- 配合 gateway 的 anticheat 模块使用
-- ============================================================

-- 反作弊违规记录表
CREATE TABLE IF NOT EXISTS anticheat_violations (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY      COMMENT '主键',
    player_id       BIGINT NOT NULL                        COMMENT '违规玩家ID，关联players.id',
    violation_type  VARCHAR(64) NOT NULL                   COMMENT '违规类型：rate_limit_exceeded/combat_speed_abnormal/economy_price_deviation/economy_rapid_trade/economy_abnormal_accumulation/login_abnormal_online/login_repeated_actions',
    severity        ENUM('low','medium','high') NOT NULL   COMMENT '严重度：low-低/medium-中/high-高',
    evidence        JSON DEFAULT NULL                      COMMENT '违规证据，JSON格式存储详细上下文',
    ip              VARCHAR(45) DEFAULT NULL               COMMENT '违规时的玩家IP地址',
    action          VARCHAR(64) DEFAULT NULL               COMMENT '违规时的具体动作名称',
    msg_id          INT UNSIGNED DEFAULT NULL               COMMENT '违规时的消息ID',
    description     VARCHAR(512) DEFAULT NULL              COMMENT '人工可读的违规描述',
    reported_by     VARCHAR(64) NOT NULL DEFAULT 'system'  COMMENT '报告来源：system/manual',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP    COMMENT '违规记录创建时间',

    INDEX idx_player_id (player_id)                        COMMENT '按玩家ID查询加速',
    INDEX idx_violation_type (violation_type)              COMMENT '按违规类型查询加速',
    INDEX idx_severity (severity)                          COMMENT '按严重度查询加速',
    INDEX idx_created_at (created_at)                      COMMENT '按时间查询加速',
    INDEX idx_player_severity (player_id, severity)        COMMENT '复合索引：玩家+严重度'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='反作弊违规记录表';


-- 反作弊封禁记录表
CREATE TABLE IF NOT EXISTS anticheat_bans (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY    COMMENT '主键',
    player_id         BIGINT NOT NULL                      COMMENT '被封禁玩家ID，关联players.id',
    reason            VARCHAR(512) NOT NULL                COMMENT '封禁原因描述',
    banned_by         VARCHAR(64) NOT NULL DEFAULT 'system' COMMENT '封禁执行者：system-系统自动/manual-管理员手动/GM账号名',
    ban_type          ENUM('temporary','permanent') NOT NULL DEFAULT 'temporary' COMMENT '封禁类型：temporary-临时/permanent-永久',
    duration_seconds  INT NOT NULL DEFAULT 3600            COMMENT '封禁时长(秒)，永久封禁为0',
    violation_id      BIGINT DEFAULT NULL                  COMMENT '关联的违规记录ID（anticheat_violations.id）',
    started_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP  COMMENT '封禁开始时间',
    expires_at        TIMESTAMP NOT NULL                   COMMENT '封禁到期时间（临时封禁）或'9999-12-31 23:59:59'（永久封禁）',
    active            BOOLEAN NOT NULL DEFAULT TRUE        COMMENT '是否仍处于封禁状态：TRUE-有效/TALSE-已解除',
    revoked_at        TIMESTAMP NULL DEFAULT NULL          COMMENT '封禁解除时间（手动解封时设置）',
    revoked_by        VARCHAR(64) DEFAULT NULL             COMMENT '封禁解除者',
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP  COMMENT '记录创建时间',
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                      ON UPDATE CURRENT_TIMESTAMP          COMMENT '记录更新时间',

    INDEX idx_player_id (player_id)                        COMMENT '按玩家ID查询加速',
    INDEX idx_active (active)                              COMMENT '按封禁状态查询加速',
    INDEX idx_expires_at (expires_at)                      COMMENT '按到期时间查询加速（用于定时清理过期封禁）',
    INDEX idx_player_active (player_id, active)            COMMENT '复合索引：玩家+状态',
    INDEX idx_banned_by (banned_by)                        COMMENT '按封禁执行者查询加速'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='反作弊封禁记录表';
