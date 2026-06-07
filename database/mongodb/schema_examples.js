/**
 * ============================================================
 *  修仙游戏 - MongoDB 集合文档示例与索引配置
 *  Cultivation Game - MongoDB Schema Examples & Index Config
 *
 *  使用方式: 在 mongosh 中直接执行本文件
 *  Usage: mongosh < schema_examples.js
 *
 *  约定:
 *  - 所有时间戳使用 ISODate
 *  - player_id 使用 Long 类型 (支持 64 位)
 *  - _id 默认由 MongoDB ObjectId 生成
 *  - 集合名称使用蛇形命名法 (snake_case)
 * ============================================================
 */

// ============================================================
// 切换至目标数据库
// ============================================================
use("cultivation_game");

print("========== 开始创建集合与索引 ==========");

// ============================================================
// 1. cultivation_logs - 修炼日志
//
// 记录玩家每次修炼的完整过程，包括打坐、运功、突破尝试等。
// 用于玩家个人修炼历史查询、突破概率分析、修炼效率计算。
//
// 分片建议: { player_id: 1 } (按玩家分片，适合按玩家查询)
// 数据量预估: 每人每天数十条 ~ 100条
// ============================================================
print("\n--- 1. cultivation_logs ---");

// --- 索引 ---
// 按玩家和时间倒序查询 (修炼历史)
db.cultivation_logs.createIndex({ player_id: 1, timestamp: -1 }, {
  name: "idx_player_timestamp"
});

// 按时间查询 + TTL 自动过期 (90天)
db.cultivation_logs.createIndex({ timestamp: -1 }, {
  name: "idx_timestamp_ttl",
  expireAfterSeconds: 7776000  // 90天 = 90 * 24 * 3600
});

// 按境界查询 (运营统计)
db.cultivation_logs.createIndex({ realm_id: 1, timestamp: -1 }, {
  name: "idx_realm_timestamp"
});

// 突破成功率分析
db.cultivation_logs.createIndex({ technique_id: 1, breakthrough_attempted: 1, timestamp: -1 }, {
  name: "idx_technique_breakthrough"
});

// --- 文档示例 ---
db.cultivation_logs.insertOne({
  player_id: Long("100001"),
  player_name: "青云道人",
  // 当前境界 (1-炼气, 2-筑基, 3-金丹, 4-元婴, 5-化神)
  realm_id: 3,
  realm_name: "金丹期",
  // 功法信息
  technique_id: 101,
  technique_name: "焚天诀",
  technique_grade: "地阶上品",
  technique_level: 7,           // 功法当前等级
  // 经验变化
  exp_before: 15000,
  exp_gained: 500,
  exp_after: 15500,
  // 修炼耗时 (秒)
  duration_seconds: 3600,
  // 修炼模式: 1-普通打坐, 2-丹药辅助, 3-灵脉修炼, 4-闭关
  practice_mode: 2,
  // 灵药/资源消耗
  item_used: {
    item_id: 201,
    item_name: "聚灵丹",
    item_count: 3
  },
  // 突破尝试 (仅突破时有值)
  breakthrough_attempted: true,
  breakthrough_success: false,
  breakthrough_rate: 0.35,       // 基础突破概率
  breakthrough_bonus_rate: 0.10, // 丹药/ buff 加成
  breakthrough_actual_rate: 0.45, // 实际概率
  // 修炼收益倍率
  efficiency_multiplier: 1.5,
  // 修炼地点
  location: {
    region_id: 1,
    region_name: "青云山脉",
    spot_id: 3,
    spot_name: "朝阳洞府",
    location_type: "灵脉",       // 普通/灵脉/秘境
    spirit_power: 85             // 灵气浓度 0-100
  },
  // 客户端 IP (用于风控)
  client_ip: "192.168.1.100",
  timestamp: ISODate("2025-06-01T12:00:00Z")
});

print("  => cultivation_logs 初始化完成");


// ============================================================
// 2. chat_messages - 聊天记录
//
// 全服聊天、世界频道、门派频道、私聊等。
// 按 channel + 时间分区，7天自动过期。
//
// 分片建议:
//   - 按 { channel: 1, _id: 1 } 哈希分片
//   - 或者按 channel 做 zone sharding
// 过期策略: 7天 TTL
// 数据量预估: 大量写入，每天数百万条
// ============================================================
print("\n--- 2. chat_messages ---");

// --- 索引 ---
// 按频道+时间查询 (核心查询)
db.chat_messages.createIndex({ channel: 1, timestamp: -1 }, {
  name: "idx_channel_timestamp"
});

// 按发送者查询 (用户历史消息)
db.chat_messages.createIndex({ sender_id: 1, timestamp: -1 }, {
  name: "idx_sender_timestamp"
});

// 私聊双方查询
db.chat_messages.createIndex({
  channel: 1, target_id: 1, timestamp: -1
}, {
  name: "idx_channel_target_timestamp"
});

// TTL 自动过期 (7天)
db.chat_messages.createIndex({ timestamp: 1 }, {
  name: "idx_timestamp_ttl",
  expireAfterSeconds: 604800  // 7天
});

// 敏感词/风控查询 (按内容哈希)
db.chat_messages.createIndex({ content_hash: 1 }, {
  name: "idx_content_hash"
});

// --- 文档示例 ---
db.chat_messages.insertOne({
  // 频道类型: world-世界, guild-门派, team-队伍, whisper-私聊, system-系统
  channel: "world",
  sender_id: Long("100001"),
  sender_name: "青云道人",
  sender_level: 55,
  // 私聊时有 target
  target_id: Long("100002"),
  target_name: "月影仙子",
  // 消息内容
  content: "诸位道友，青云山脉发现异宝，速来！",
  content_length: 18,
  content_hash: "sha256hex...",   // 用于敏感词过滤和去重
  // 消息类型: 0-普通文本, 1-表情, 2-道具链接, 3-语音, 4-系统消息
  message_type: 0,
  // 富媒体 (含道具链接时)
  link_data: null,                 // { type: "item", item_id: 301, item_name: "玄天剑" }
  // 表情 ID
  emoji_id: null,
  // 客户端信息
  client_info: {
    platform: "ios",
    app_version: "2.1.0",
    device_id: "device_uuid_xxx"
  },
  // 消息状态
  is_deleted: false,
  deleted_at: null,
  timestamp: ISODate("2025-06-05T14:30:00Z")
});

print("  => chat_messages 初始化完成");


// ============================================================
// 3. combat_records - 战斗记录
//
// 包含完整战斗回合日志，支持战斗回放。
// PvP / PvE / 帮战 / 世界 Boss 等所有战斗类型。
//
// 分片建议: { player_id: 1 } (玩家自己查询)
//           或 { combat_type: 1, _id: 1 } (按类型全局查询)
// 过期策略: 30天 (玩家回放), 7天 (详细回合日志可归档)
// 数据量预估: 每天数十万 ~ 百万条
// ============================================================
print("\n--- 3. combat_records ---");

// --- 索引 ---
// 按玩家+时间 (玩家战斗历史)
db.combat_records.createIndex({ player_id: 1, timestamp: -1 }, {
  name: "idx_player_timestamp"
});

// 按战斗类型+时间 (运营查询)
db.combat_records.createIndex({ combat_type: 1, timestamp: -1 }, {
  name: "idx_combattype_timestamp"
});

// 按战斗 ID 精确查询 (回放)
db.combat_records.createIndex({ combat_id: 1 }, {
  name: "idx_combat_id",
  unique: true
});

// 按双方玩家查询 (PvP 双方查看)
db.combat_records.createIndex({
  "participants.player_id": 1, timestamp: -1
}, {
  name: "idx_participants_timestamp"
});

// 按胜利方/失败方查询 (排行榜)
db.combat_records.createIndex({ winner_id: 1, combat_type: 1, timestamp: -1 }, {
  name: "idx_winner_combattype"
});

// TTL 过期 (30天)
db.combat_records.createIndex({ timestamp: 1 }, {
  name: "idx_timestamp_ttl",
  expireAfterSeconds: 2592000  // 30天
});

// --- 文档示例 ---
db.combat_records.insertOne({
  // 战斗唯一标识
  combat_id: "combat_20250605_abc123",
  // 战斗类型: pvp-玩家对战, pve-打怪, guildwar-帮战, boss-世界Boss, arena-竞技场
  combat_type: "pvp",
  // 战斗场景
  scene_id: 5,
  scene_name: "昆仑擂台",
  // 参与方
  participants: [
    {
      player_id: Long("100001"),
      player_name: "青云道人",
      realm_id: 3,
      realm_name: "金丹期",
      level: 55,
      faction: "天剑门",
      // 战斗前属性
      hp_before: 12000,
      hp_after: 3500,
      mp_before: 8000,
      mp_after: 1200,
      // 使用的技能
      skills_used: [101, 203, 305],
      // 总输出
      total_damage_dealt: 9500,
      total_damage_taken: 8500,
      is_winner: true,
      team_id: 1
    },
    {
      player_id: Long("100002"),
      player_name: "月影仙子",
      realm_id: 3,
      realm_name: "金丹期",
      level: 54,
      faction: "瑶池宫",
      hp_before: 11500,
      hp_after: 0,
      mp_before: 9000,
      mp_after: 500,
      skills_used: [105, 208, 310],
      total_damage_dealt: 8500,
      total_damage_taken: 9500,
      is_winner: false,
      team_id: 2
    }
  ],
  // 战斗统计
  stats: {
    total_rounds: 12,
    total_duration_ms: 45200,
    total_damage: 18000,
    total_healing: 3000,
    crit_count: 4,
    miss_count: 2
  },
  // 完整回合日志 (战斗回放核心数据)
  rounds: [
    {
      round: 1,
      actions: [
        {
          actor_id: Long("100001"),
          action_type: "skill",     // skill / item / defend / flee
          skill_id: 101,
          skill_name: "焚天剑气",
          target_id: Long("100002"),
          damage: 1200,
          is_crit: false,
          is_miss: false,
          buff_applied: null,
          hp_snapshot: {
            actor: 12000,
            target: 10300
          }
        },
        {
          actor_id: Long("100002"),
          action_type: "skill",
          skill_id: 105,
          skill_name: "冰心诀",
          target_id: Long("100001"),
          damage: 900,
          is_crit: false,
          is_miss: false,
          buff_applied: {
            buff_id: 50,
            buff_name: "寒冰护盾",
            duration_rounds: 3
          },
          hp_snapshot: {
            actor: 10300,
            target: 11100
          }
        }
      ]
    }
    // ... 后续回合 (完整结构同上, 此处省略以保持示例简洁)
  ],
  // 奖励
  rewards: {
    exp_gained: 2000,
    spirit_stones: 500,
    items_dropped: [
      { item_id: 301, item_name: "灵石碎片", count: 3 }
    ],
    // 排名变化 (仅竞技场)
    rank_change: null
  },
  summary: "青云道人在第12回合击败了月影仙子",
  timestamp: ISODate("2025-06-05T15:00:00Z")
});

print("  => combat_records 初始化完成");


// ============================================================
// 4. encounter_logs - 奇遇日志
//
// 记录玩家触发的随机奇遇事件。
// 用于奇遇统计、成就判定、后续奇遇链触发。
//
// 分片建议: { player_id: 1 }
// 过期策略: 60天 (仅保留近期奇遇)
// 数据量: 每人每天较少，但奇遇链需完整保留
// ============================================================
print("\n--- 4. encounter_logs ---");

// --- 索引 ---
// 按玩家+时间 (玩家奇遇历史)
db.encounter_logs.createIndex({ player_id: 1, timestamp: -1 }, {
  name: "idx_player_timestamp"
});

// 按奇遇类型查询 (运营分析)
db.encounter_logs.createIndex({ encounter_type: 1, timestamp: -1 }, {
  name: "idx_encountertype_timestamp"
});

// 奇遇链查询 (连续奇遇)
db.encounter_logs.createIndex({ chain_id: 1, step: 1 }, {
  name: "idx_chain_step"
});

// 按玩家+区域查询 (区域奇遇触发率分析)
db.encounter_logs.createIndex({ player_id: 1, "location.region_id": 1 }, {
  name: "idx_player_region"
});

// TTL 过期 (60天)
db.encounter_logs.createIndex({ timestamp: 1 }, {
  name: "idx_timestamp_ttl",
  expireAfterSeconds: 5184000  // 60天
});

// --- 文档示例 ---
db.encounter_logs.insertOne({
  player_id: Long("100001"),
  player_name: "青云道人",
  // 奇遇类型: 1-秘境发现, 2-前辈传承, 3-天降异宝, 4-妖兽突袭,
  //           5-隐世高人, 6-灵草现世, 7-古宝出世, 8-天劫预警
  encounter_type: 2,
  encounter_type_name: "前辈传承",
  encounter_id: "enc_20250604_789xyz",
  // 奇遇等级 (决定奖励品质)
  encounter_grade: 4,           // 1-普通, 2-稀有, 3-史诗, 4-传说
  // 奇遇名称 (用于前端展示)
  encounter_name: "古剑仙遗刻",
  // 奇遇详细描述
  description: "在青云山脉深处发现一处上古剑仙洞府，墙上有遗刻剑诀。",
  // 触发条件
  trigger_condition: {
    type: "area_explore",       // 探索触发
    region_id: 1,
    min_realm: 2,
    required_item: null,
    probability: 0.001
  },
  // 奇遇地点
  location: {
    region_id: 1,
    region_name: "青云山脉",
    coordinates: { x: 1250, y: 3800 }
  },
  // 奇遇结果与奖励
  result: {
    is_success: true,
    rewards: {
      exp: 5000,
      spirit_stones: 2000,
      items: [
        { item_id: 401, item_name: "残破剑谱·上卷", count: 1, quality: "传说" }
      ],
      // 领悟技能
      skill_unlocked: {
        skill_id: 501,
        skill_name: "天外飞仙",
        skill_grade: "天阶下品"
      }
    },
    // 后续奇遇链 (触发下一个奇遇)
    next_encounter: {
      chain_id: "chain_20250604_001",
      step: 2,
      next_encounter_id: "enc_20250604_790xyz",
      next_encounter_name: "古剑仙遗刻·续",
      chain_expire_at: ISODate("2025-06-11T12:00:00Z")  // 7天内有效
    }
  },
  timestamp: ISODate("2025-06-04T08:30:00Z")
});

print("  => encounter_logs 初始化完成");


// ============================================================
// 5. game_events - 全服事件
//
// 世界 Boss 刷新、天降异宝、仙魔大战、全服活动等。
// 所有玩家可见的系统级事件广播。
//
// 分片建议: { event_type: 1, status: 1 }
// 过期策略: TTL 基于 event_end_time 自动过期 (结束后7天)
// 数据量: 较少，但事件期间有大量关联数据
// ============================================================
print("\n--- 5. game_events ---");

// --- 索引 ---
// 按事件状态查询 (当前活跃事件)
db.game_events.createIndex({ status: 1, event_end_time: 1 }, {
  name: "idx_status_endtime"
});

// 按事件类型+时间
db.game_events.createIndex({ event_type: 1, event_start_time: -1 }, {
  name: "idx_eventtype_starttime"
});

// TTL: 事件结束后7天自动清理
db.game_events.createIndex({ event_end_time: 1 }, {
  name: "idx_endtime_ttl",
  expireAfterSeconds: 604800  // 事件结束后7天删除
});

// 全服通知推送 (当前事件列表给客户端)
db.game_events.createIndex({
  status: 1, event_type: 1, event_start_time: -1
}, {
  name: "idx_status_type_starttime"
});

// --- 文档示例 ---
db.game_events.insertOne({
  // 事件唯一 ID
  event_id: "event_20250605_worldboss_01",
  // 事件类型: worldboss-世界Boss, treasure-天降异宝,
  //           war-仙魔大战, activity-全服活动, server-服务器公告
  event_type: "worldboss",
  event_type_name: "世界 Boss",
  // 事件标题与描述
  title: "远古凶兽·饕餮降临",
  content: "远古凶兽饕餮突破封印，降临青云山脉！请各路道友速速前往讨伐！",
  // 事件状态: pending-预告, active-进行中, finished-已结束, cancelled-已取消
  status: "active",
  // 时间
  announce_time: ISODate("2025-06-05T10:00:00Z"),  // 预告时间
  event_start_time: ISODate("2025-06-05T12:00:00Z"), // 开始时间
  event_end_time: ISODate("2025-06-05T14:00:00Z"),   // 结束时间
  // 事件地点
  location: {
    region_id: 1,
    region_name: "青云山脉",
    specific_area: "妖兽森林"
  },
  // Boss 信息 (worldboss 类型)
  boss_data: {
    boss_id: 1001,
    boss_name: "饕餮",
    boss_level: 80,
    boss_realm: "元婴期",
    total_hp: 5000000,
    current_hp: 3500000,
    // 掉落的奖励池
    reward_pool: {
      min_rank: 1,
      max_rank: 100,
      items: [
        { item_id: 501, item_name: "饕餮鳞片", count: 1, drop_rate: 0.3 },
        { item_id: 502, item_name: "太古精血", count: 1, drop_rate: 0.05 }
      ]
    },
    // 参与人数统计(实时更新)
    participant_count: 156,
    total_damage_dealt: 1500000
  },
  // 天降异宝类型 (treasure 类型)
  treasure_data: null,
  // 事件标签
  tags: ["限时", "世界Boss", "组队"],
  // 奖励结算状态
  reward_claimed: false,
  timestamp: ISODate("2025-06-05T12:00:00Z")
});

print("  => game_events 初始化完成");


// ============================================================
// 6. operation_logs - GM 操作审计日志
//
// 记录 GM (Game Master) 和管理员的所有操作。
// 仅追加 (append-only)，不可删除，不可修改。
// 用于合规审计和问题追溯。
//
// 分片建议: { operator_id: 1, timestamp: -1 }
// 过期策略: 无 TTL (永久保存)
// 数据量: 较少，但需长期积累
// ============================================================
print("\n--- 6. operation_logs ---");

// --- 索引 ---
// 按操作者+时间查询 (审计追溯)
db.operation_logs.createIndex({ operator_id: 1, timestamp: -1 }, {
  name: "idx_operator_timestamp"
});

// 按玩家 ID 查询 (对某玩家的所有操作)
db.operation_logs.createIndex({ target_player_id: 1, timestamp: -1 }, {
  name: "idx_targetplayer_timestamp"
});

// 按操作类型查询
db.operation_logs.createIndex({ operation_type: 1, timestamp: -1 }, {
  name: "idx_operationtype_timestamp"
});

// IP + 操作者 (安全审计)
db.operation_logs.createIndex({ client_ip: 1, operator_id: 1 }, {
  name: "idx_ip_operator"
});

// 操作时间范围查询 (日报/周报)
db.operation_logs.createIndex({ timestamp: -1 }, {
  name: "idx_timestamp"
});

// --- 文档示例 ---
db.operation_logs.insertOne({
  // 操作唯一 ID
  log_id: "gm_20250605_op_001",
  // 操作者信息
  operator_id: Long("900001"),
  operator_name: "admin_zhang",
  operator_role: "super_gm",    // super_gm / gm / cs(客服) / developer
  // 操作类型
  operation_type: "item_grant",
  operation_category: "player_management",  // 大类: system/player/item/event/notice
  // 操作详情
  detail: {
    action: "发放道具",
    reason: "玩家误操作提交工单补偿",        // 操作原因
    ticket_id: "CS-20250605-001",           // 关联工单
    // 涉及玩家
    target_player_id: Long("100001"),
    target_player_name: "青云道人",
    // 操作前后快照 (用于回滚和验证)
    before_snapshot: {
      spirit_stones: 5000,
      items: [{ item_id: 301, count: 10 }]
    },
    after_snapshot: {
      spirit_stones: 8000,
      items: [{ item_id: 301, count: 10 }, { item_id: 302, count: 5 }]
    },
    // 发放的道具列表
    changes: [
      { field: "spirit_stones", delta: 3000, type: "add" },
      { field: "inventory.item_302", delta: 5, type: "add" }
    ]
  },
  // 审批信息 (敏感操作需审批)
  approval: {
    required: true,
    approved_by: Long("900002"),
    approved_at: ISODate("2025-06-05T10:05:00Z"),
    status: "approved"            // pending / approved / rejected
  },
  // 源信息
  source: {
    ip: "10.0.1.100",
    platform: "gm_web",           // gm_web / api / console
    user_agent: "Mozilla/5.0 ..."
  },
  // 数据完整性校验: 该文档不可被删除或修改
  immutable: true,
  // 数据签名 (用于防篡改验证)
  signature: "sha256_hmac_signature_value",
  timestamp: ISODate("2025-06-05T10:00:00Z")
});

print("  => operation_logs 初始化完成");


// ============================================================
// 7. analytics_events - 运营分析事件
//
// 记录玩家行为数据: 登录/登出/充值/消费/升级等。
// 按日期分区存储，用于数据分析和 BI 报表。
//
// 分片建议: { event_type: 1, event_date: 1 }
// 过期策略: 按 event_date 分区删除 (保留 180 天)
// 数据量: 极大，每天数千万条
// ============================================================
print("\n--- 7. analytics_events ---");

// --- 索引 ---
// 按事件类型+日期查询 (核心分析查询)
db.analytics_events.createIndex({ event_type: 1, event_date: -1, timestamp: -1 }, {
  name: "idx_type_date_timestamp"
});

// 按玩家查询 (用户行为画像)
db.analytics_events.createIndex({ player_id: 1, event_date: -1, event_type: 1 }, {
  name: "idx_player_date_type"
});

// DAU/MAU 统计 (按日期去重)
db.analytics_events.createIndex({
  event_type: 1, event_date: -1, player_id: 1
}, {
  name: "idx_type_date_player",
  partialFilterExpression: { event_type: { $in: ["login", "logout"] } }
});

// 充值分析
db.analytics_events.createIndex({
  event_type: 1, event_date: -1, "revenue.amount": 1
}, {
  name: "idx_type_date_revenue",
  partialFilterExpression: { event_type: "purchase" }
});

// 等级分布
db.analytics_events.createIndex({
  event_type: 1, event_date: -1, player_level: 1
}, {
  name: "idx_type_date_level",
  partialFilterExpression: { event_type: "level_up" }
});

// 过期索引 (用于分区删除, 保留 180 天)
db.analytics_events.createIndex({ event_date: 1 }, {
  name: "idx_eventdate_ttl",
  expireAfterSeconds: 15552000  // 180天
});

// --- 文档示例 ---

// (a) 登录事件
db.analytics_events.insertOne({
  event_id: "ae_20250605_login_100001_001",
  event_type: "login",
  event_date: "2025-06-05",           // YYYY-MM-DD 字符串，用于分区
  player_id: Long("100001"),
  player_name: "青云道人",
  player_level: 55,
  player_realm: 3,                    // 金丹期
  player_vip_level: 3,
  player_faction: "天剑门",
  // 设备信息
  device: {
    platform: "ios",
    device_model: "iPhone 16 Pro",
    os_version: "iOS 19.0",
    device_id: "device_uuid_xxx"
  },
  // 渠道信息
  channel: "app_store",               // 用户获取渠道
  // 登录方式: account / wechat / apple / guest
  login_method: "wechat",
  // 连续登录天数
  consecutive_login_days: 15,
  // 客户端版本
  client_version: "2.1.0",
  // IP 属地
  ip: "114.114.114.114",
  geo: {
    country: "中国",
    province: "广东省",
    city: "深圳市"
  },
  // 会话 ID (关联登录和登出)
  session_id: "session_uuid_xxx",
  // 客户端时间 (用于校准)
  client_timestamp: ISODate("2025-06-05T19:00:00Z"),
  timestamp: ISODate("2025-06-05T19:00:00Z")
});

// (b) 登出事件
db.analytics_events.insertOne({
  event_id: "ae_20250605_logout_100001_001",
  event_type: "logout",
  event_date: "2025-06-05",
  player_id: Long("100001"),
  player_name: "青云道人",
  player_level: 55,
  player_realm: 3,
  player_vip_level: 3,
  // 会话时长 (秒)
  session_duration_seconds: 7200,
  session_id: "session_uuid_xxx",
  // 登出时统计
  session_stats: {
    exp_gained: 8500,
    spirit_stones_gained: 1200,
    items_acquired: 3,
    combats_fought: 5,
    combats_won: 4,
    encounters_triggered: 1,
    level_up: true,
    level_before: 54,
    level_after: 55
  },
  // 登出方式: normal / timeout / force_kick / crash
  logout_reason: "normal",
  timestamp: ISODate("2025-06-05T21:00:00Z")
});

// (c) 充值事件
db.analytics_events.insertOne({
  event_id: "ae_20250605_purchase_100001_001",
  event_type: "purchase",
  event_date: "2025-06-05",
  player_id: Long("100001"),
  player_name: "青云道人",
  player_level: 55,
  player_realm: 3,
  player_vip_level: 3,
  // 订单信息
  order: {
    order_id: "order_20250605_001",
    // 充值金额 (元)
    amount: 648.00,
    currency: "CNY",
    // 获得灵玉
    lingyu_gained: 6480,
    bonus_lingyu: 648,                // 首充赠送
    total_lingyu: 7128,
    // 支付渠道
    payment_channel: "wechat_pay",
    payment_method: "wechat_wallet",
    // 商品 ID (商城配置)
    product_id: "lingyu_pack_648",
    product_name: "648灵玉礼包",
    // 支付状态
    status: "success",                 // pending / success / failed / refunded
    paid_at: ISODate("2025-06-05T20:15:00Z")
  },
  // 是否首充
  is_first_purchase: false,
  // 累计充值 (充值后)
  total_purchase_amount: 3240.00,
  // 风控标记
  risk_flag: false,
  timestamp: ISODate("2025-06-05T20:15:00Z")
});

// (d) 消费事件 (灵玉消费)
db.analytics_events.insertOne({
  event_id: "ae_20250605_spend_100001_001",
  event_type: "spend",
  event_date: "2025-06-05",
  player_id: Long("100001"),
  player_name: "青云道人",
  player_level: 55,
  player_realm: 3,
  player_vip_level: 3,
  // 消费信息
  spend: {
    // 消费类型: shop-商城购买, gacha-抽卡, refine-炼器, auction-拍卖行,
    //           teleport-传送, revive-复活, buff-购买buff
    spend_type: "gacha",
    spend_type_name: "灵玉抽卡",
    // 消耗的货币类型: lingyu-灵玉, spirit_stones-灵石
    currency_type: "lingyu",
    amount: 280,
    currency_name: "灵玉",
    // 获得的物品
    items_obtained: [
      { item_id: 601, item_name: "紫霞仙衣", quality: "史诗", count: 1 }
    ],
    // 商城商品 ID (商城购买时)
    product_id: "gacha_10x_advanced",
    // 抽卡池信息
    gacha_pool: {
      pool_id: 3,
      pool_name: "仙衣阁",
      pity_count: 45,                  // 保底计数
      guaranteed_ssr: false
    }
  },
  // 消费后余额
  balance_after: {
    lingyu: 5840,
    spirit_stones: 3200
  },
  timestamp: ISODate("2025-06-05T20:20:00Z")
});

// (e) 升级事件
db.analytics_events.insertOne({
  event_id: "ae_20250605_levelup_100001_002",
  event_type: "level_up",
  event_date: "2025-06-05",
  player_id: Long("100001"),
  player_name: "青云道人",
  player_level: 56,                    // 升级后等级
  player_realm: 3,
  player_vip_level: 3,
  // 升级详情
  level_up: {
    level_before: 55,
    level_after: 56,
    // 突破情况
    realm_before: 3,
    realm_after: 3,                    // 未突破大境界
    // 升级途径: practice-修炼, combat-战斗, quest-任务, item-道具
    source: "practice",
    // 升级耗时
    total_exp_required: 50000,
    exp_accumulated: 52300,
    days_since_last_level: 3,
    // 是否是大境界突破
    is_realm_breakthrough: false
  },
  timestamp: ISODate("2025-06-05T22:00:00Z")
});

print("  => analytics_events 初始化完成");


// ============================================================
// 汇总信息
// ============================================================
print("\n========== 所有集合已初始化 ==========");
print("集合列表:");
print("  1. cultivation_logs  - 修炼日志 (TTL: 90天)");
print("  2. chat_messages     - 聊天记录 (TTL: 7天)");
print("  3. combat_records    - 战斗记录 (TTL: 30天)");
print("  4. encounter_logs    - 奇遇日志 (TTL: 60天)");
print("  5. game_events       - 全服事件 (TTL: 结束后7天)");
print("  6. operation_logs    - GM审计日志 (永久保留)");
print("  7. analytics_events  - 运营分析事件 (TTL: 180天)");
print("===========================================");
