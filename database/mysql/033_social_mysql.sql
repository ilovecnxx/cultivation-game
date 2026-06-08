-- ===================================================================
-- 社交系统 MySQL 表 — 替代 MongoDB，单机部署简化
-- ===================================================================

-- 聊天消息表
CREATE TABLE IF NOT EXISTS chat_messages (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY    COMMENT '消息ID',
    channel         VARCHAR(20) NOT NULL                COMMENT '频道: world/sect/private/system',
    sender_id       VARCHAR(32) NOT NULL                COMMENT '发送者ID',
    sender_name     VARCHAR(64) NOT NULL                COMMENT '发送者昵称',
    target_id       VARCHAR(32) DEFAULT ''              COMMENT '目标ID(私聊对方/宗门ID)',
    content         VARCHAR(1024) NOT NULL              COMMENT '消息内容',
    is_system       TINYINT DEFAULT 0                   COMMENT '是否系统消息',
    created_at      DATETIME(3) NOT NULL                COMMENT '发送时间',

    INDEX idx_channel_time (channel, created_at),
    INDEX idx_private (sender_id, target_id, created_at),
    INDEX idx_target (target_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='聊天消息表';

-- 道侣关系表
CREATE TABLE IF NOT EXISTS daolv_relations (
    id              VARCHAR(64) NOT NULL PRIMARY KEY    COMMENT '关系ID',
    player_a        BIGINT NOT NULL                     COMMENT '玩家A',
    player_b        BIGINT NOT NULL                     COMMENT '玩家B',
    intimacy        INT DEFAULT 0                       COMMENT '亲密度',
    compatibility   DOUBLE DEFAULT 0                    COMMENT '契合度(0-1)',
    level           VARCHAR(20) DEFAULT '初识'          COMMENT '等级: 初识/知己/情深/同心/仙侣',
    skills          JSON DEFAULT NULL                   COMMENT '已解锁技能(JSON数组)',
    daily_cultivated BIGINT DEFAULT 0                   COMMENT '今日双修时间(秒)',
    daily_cultivate_date VARCHAR(10) DEFAULT ''         COMMENT '记录日期',
    gift_item_a     VARCHAR(64) DEFAULT ''              COMMENT 'A的定情信物',
    gift_item_b     VARCHAR(64) DEFAULT ''              COMMENT 'B的定情信物',
    last_propose_at DATETIME DEFAULT NULL               COMMENT '上次求婚时间',
    started_at      DATETIME NOT NULL                   COMMENT '建立时间',
    updated_at      DATETIME NOT NULL                   COMMENT '更新时间',
    status          VARCHAR(20) DEFAULT 'normal'        COMMENT 'normal/divorced',

    INDEX idx_player_a (player_a, status),
    INDEX idx_player_b (player_b, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='道侣关系表';

-- 道侣申请(求婚)表
CREATE TABLE IF NOT EXISTS daolv_proposals (
    id              VARCHAR(64) NOT NULL PRIMARY KEY    COMMENT '申请ID',
    from_id         BIGINT NOT NULL                     COMMENT '发起方ID',
    from_name       VARCHAR(64) DEFAULT ''              COMMENT '发起方昵称',
    to_id           BIGINT NOT NULL                     COMMENT '目标ID',
    to_name         VARCHAR(64) DEFAULT ''              COMMENT '目标昵称',
    message         VARCHAR(512) DEFAULT ''             COMMENT '留言',
    gift_item_id    VARCHAR(64) DEFAULT ''              COMMENT '信物ID',
    gift_item_name  VARCHAR(64) DEFAULT ''              COMMENT '信物名称',
    status          VARCHAR(20) DEFAULT 'pending'       COMMENT 'pending/accepted/rejected',
    created_at      DATETIME NOT NULL                   COMMENT '创建时间',
    handled_at      DATETIME DEFAULT NULL               COMMENT '处理时间',

    INDEX idx_from (from_id, status),
    INDEX idx_to (to_id, status),
    INDEX idx_pair (from_id, to_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='道侣申请(求婚)表';

-- 道侣任务表
CREATE TABLE IF NOT EXISTS daolv_tasks (
    id              VARCHAR(64) NOT NULL PRIMARY KEY    COMMENT '任务ID',
    relation_id     VARCHAR(64) NOT NULL                COMMENT '道侣关系ID',
    period          VARCHAR(20) DEFAULT ''              COMMENT '周期标识',
    progress        BIGINT DEFAULT 0                    COMMENT '当前进度',
    completed       TINYINT DEFAULT 0                   COMMENT '是否完成',
    created_at      DATETIME NOT NULL                   COMMENT '创建时间',

    INDEX idx_relation (relation_id, period)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='道侣任务表';

-- 邮件表
CREATE TABLE IF NOT EXISTS social_mail (
    id              VARCHAR(64) NOT NULL PRIMARY KEY    COMMENT '邮件ID',
    mail_type       VARCHAR(20) NOT NULL                COMMENT '类型: system/player',
    title           VARCHAR(128) NOT NULL               COMMENT '标题',
    content         TEXT NOT NULL                       COMMENT '内容',
    sender_id       VARCHAR(32) DEFAULT ''              COMMENT '发送者ID',
    sender_name     VARCHAR(64) DEFAULT ''              COMMENT '发送者昵称',
    receiver_id     VARCHAR(32) NOT NULL                COMMENT '接收者ID',
    attachments     JSON DEFAULT NULL                   COMMENT '附件(JSON)',
    is_read         TINYINT DEFAULT 0                   COMMENT '是否已读',
    is_claimed      TINYINT DEFAULT 0                   COMMENT '附件是否已领取',
    created_at      DATETIME NOT NULL                   COMMENT '创建时间',
    expire_at       DATETIME DEFAULT NULL               COMMENT '过期时间',

    INDEX idx_receiver (receiver_id, is_read, created_at),
    INDEX idx_receiver_claimed (receiver_id, is_claimed)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='社交邮件表';

-- 好友申请表
CREATE TABLE IF NOT EXISTS friend_applies (
    id              VARCHAR(64) NOT NULL PRIMARY KEY    COMMENT '申请ID',
    from_id         VARCHAR(32) NOT NULL                COMMENT '发起方ID',
    from_name       VARCHAR(64) DEFAULT ''              COMMENT '发起方昵称',
    to_id           VARCHAR(32) NOT NULL                COMMENT '目标ID',
    message         VARCHAR(256) DEFAULT ''             COMMENT '留言',
    status          VARCHAR(20) DEFAULT 'pending'       COMMENT 'pending/accepted/rejected',
    created_at      DATETIME NOT NULL                   COMMENT '创建时间',
    handled_at      DATETIME DEFAULT NULL               COMMENT '处理时间',

    INDEX idx_to_status (to_id, status),
    INDEX idx_from_to (from_id, to_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友申请表';

-- 宗门申请表
CREATE TABLE IF NOT EXISTS sect_applies (
    id              VARCHAR(64) NOT NULL PRIMARY KEY    COMMENT '申请ID',
    sect_id         VARCHAR(64) NOT NULL                COMMENT '宗门ID',
    user_id         VARCHAR(32) NOT NULL                COMMENT '用户ID',
    user_name       VARCHAR(64) DEFAULT ''              COMMENT '用户昵称',
    message         VARCHAR(256) DEFAULT ''             COMMENT '留言',
    status          VARCHAR(20) DEFAULT 'pending'       COMMENT 'pending/accepted/rejected',
    created_at      DATETIME NOT NULL                   COMMENT '创建时间',

    INDEX idx_sect_status (sect_id, status),
    INDEX idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门申请表';

-- 宗门技能表
CREATE TABLE IF NOT EXISTS sect_skills (
    id              VARCHAR(64) NOT NULL PRIMARY KEY    COMMENT '技能ID',
    sect_id         VARCHAR(64) NOT NULL                COMMENT '宗门ID',
    name            VARCHAR(64) NOT NULL                COMMENT '名称',
    description     VARCHAR(256) DEFAULT ''             COMMENT '描述',
    level           INT DEFAULT 1                       COMMENT '当前等级',
    max_level       INT DEFAULT 10                      COMMENT '最大等级',
    cost_per_level  BIGINT DEFAULT 0                    COMMENT '每级贡献消耗',
    effect_type     VARCHAR(64) DEFAULT ''              COMMENT '效果类型',
    effect_value    DOUBLE DEFAULT 0                    COMMENT '每级效果值',

    INDEX idx_sect (sect_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门技能表';

-- 宗门成员技能表
CREATE TABLE IF NOT EXISTS sect_member_skills (
    member_id       VARCHAR(64) NOT NULL                COMMENT '成员ID',
    skill_id        VARCHAR(64) NOT NULL                COMMENT '技能ID',
    level           INT DEFAULT 1                       COMMENT '当前等级',

    PRIMARY KEY (member_id, skill_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门成员技能表';

-- 宗门任务表
CREATE TABLE IF NOT EXISTS sect_missions (
    id                  VARCHAR(64) NOT NULL PRIMARY KEY    COMMENT '任务ID',
    sect_id             VARCHAR(64) NOT NULL                COMMENT '宗门ID',
    mission_type        VARCHAR(20) NOT NULL                COMMENT '类型: gathering/combat/donation/cultivate',
    description         VARCHAR(256) DEFAULT ''             COMMENT '描述',
    requirement         INT DEFAULT 0                       COMMENT '要求数量',
    reward_contribution BIGINT DEFAULT 0                    COMMENT '贡献奖励',
    reward_exp          BIGINT DEFAULT 0                    COMMENT '经验奖励',
    reward_funds        BIGINT DEFAULT 0                    COMMENT '宗门资金奖励',
    date                VARCHAR(10) NOT NULL                COMMENT '日期 yyyy-mm-dd',

    INDEX idx_sect_date (sect_id, date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门任务表';

-- 宗门成员任务进度表
CREATE TABLE IF NOT EXISTS sect_member_missions (
    id          VARCHAR(64) NOT NULL PRIMARY KEY    COMMENT '记录ID',
    member_id   VARCHAR(64) NOT NULL                COMMENT '成员ID',
    mission_id  VARCHAR(64) NOT NULL                COMMENT '任务ID',
    progress    INT DEFAULT 0                       COMMENT '当前进度',
    completed   TINYINT DEFAULT 0                   COMMENT '是否完成',
    claimed     TINYINT DEFAULT 0                   COMMENT '是否已领取',
    date        VARCHAR(10) NOT NULL                COMMENT '日期',

    INDEX idx_member_date (member_id, date),
    INDEX idx_mission (mission_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宗门成员任务进度表';
