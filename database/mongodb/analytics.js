/**
 * ============================================================
 *  修仙游戏 - 运营分析集合初始化脚本
 *  Cultivation Game - Analytics Events Collection
 *
 *  使用方式: mongosh < analytics.js
 *  或: mongosh cultivation_game analytics.js
 *
 *  功能:
 *  - 创建 analytics_events 集合
 *  - 设置 90 天 TTL 过期索引
 *  - 业务查询索引 (event_type, player_id, timestamp)
 *  - 预置 daily_stats 聚合管道 (用于运营报表)
 * ============================================================
 */

// ============================================================
// 切换至目标数据库
// ============================================================
use("cultivation_game");

print("========== 开始初始化 analytics_events 集合 ==========");

// ============================================================
// 1. analytics_events - 运营分析事件
//
// 记录玩家的全部行为数据，用于数据分析和 BI 报表。
// 通过网关分析引擎自动采集，前端 SDK 也可直接上报。
//
// 分片建议: { event_type: 1, event_date: 1 }
// 过期策略: 90 天 TTL (expireAfterSeconds: 7776000)
// 数据量: 极大，每天数百万条
// ============================================================
print("\n--- 1. analytics_events ---");

// ============================================================
// 核心业务索引
// ============================================================

// 1-a) 按事件类型+日期查询 (核心分析查询：DAU、漏斗等)
db.analytics_events.createIndex(
  { event_type: 1, event_date: -1, timestamp: -1 },
  { name: "idx_type_date_timestamp" }
);

// 1-b) 按玩家查询 (用户行为画像)
db.analytics_events.createIndex(
  { player_id: 1, event_date: -1, event_type: 1 },
  { name: "idx_player_date_type" }
);

// 1-c) 按时间范围查询 (运营报表)
db.analytics_events.createIndex(
  { timestamp: -1 },
  { name: "idx_timestamp" }
);

// 1-d) 按事件类型+玩家去重 (DAU/MAU 统计)
db.analytics_events.createIndex(
  { event_type: 1, event_date: -1, player_id: 1 },
  {
    name: "idx_type_date_player",
    partialFilterExpression: { event_type: { $in: ["player_login", "player_register"] } }
  }
);

// 1-e) 充值分析索引
db.analytics_events.createIndex(
  { event_type: 1, event_date: -1, "properties.amount": 1 },
  {
    name: "idx_type_date_amount",
    partialFilterExpression: { event_type: { $in: ["recharge_success", "shop_purchase"] } }
  }
);

// 1-f) 玩家等级分布索引
db.analytics_events.createIndex(
  { event_type: 1, event_date: -1, player_level: 1 },
  {
    name: "idx_type_date_level",
    partialFilterExpression: { event_type: "system_level_up" }
  }
);

// 1-g) 事件类型+境界查询 (修炼相关分析)
db.analytics_events.createIndex(
  { event_type: 1, realm: 1, timestamp: -1 },
  { name: "idx_type_realm_timestamp" }
);

// 1-h) 会话关联查询 (登录->登出关联)
db.analytics_events.createIndex(
  { session_id: 1, event_type: 1 },
  {
    name: "idx_session_event",
    partialFilterExpression: {
      event_type: { $in: ["player_login", "player_logout"] }
    }
  }
);

// ============================================================
// TTL 过期索引 (90 天自动清理)
// ============================================================
db.analytics_events.createIndex(
  { timestamp: 1 },
  {
    name: "idx_timestamp_ttl",
    expireAfterSeconds: 7776000  // 90 天 = 90 * 24 * 3600
  }
);

print("  => analytics_events 索引创建完成");


// ============================================================
// 2. daily_stats - 日统计聚合结果 (物化视图)
//
// 由 daily_stats 聚合管道定时写入，缓存每日统计结果。
// 管理员通过管理后台查询。
// ============================================================
print("\n--- 2. daily_stats ---");

// 2-a) 按日期查询
db.daily_stats.createIndex(
  { date: -1 },
  { name: "idx_date", unique: true }
);

// 2-b) 按指标类型查询
db.daily_stats.createIndex(
  { "metrics.name": 1, date: -1 },
  { name: "idx_metric_date" }
);

// TTL: 保留 365 天
db.daily_stats.createIndex(
  { date: 1 },
  {
    name: "idx_date_ttl",
    expireAfterSeconds: 31536000  // 365 天
  }
);

print("  => daily_stats 索引创建完成");


// ============================================================
// 3. 预置每天运行的 dailyStats 聚合管道
//
// 此管道计算每日核心运营指标，结果写入 daily_stats 集合。
// 建议通过 cron 或调度器每天凌晨 00:30 执行一次。
//
// 用法:
//   db.analytics_events.aggregate(dailyStatsPipeline, { allowDiskUse: true })
// ============================================================
print("\n--- 3. 预置 daily_stats 聚合管道 ---");

const dailyStatsPipeline = [
  // 匹配当天的事件
  {
    $match: {
      event_date: "YYYY-MM-DD",  // 替换为目标日期
      timestamp: {
        $gte: new Date("YYYY-MM-DDT00:00:00Z"),
        $lt: new Date("YYYY-MM-DDT23:59:59Z")
      }
    }
  },
  // 按事件类型分组统计
  {
    $group: {
      _id: "$event_type",
      total_events: { $sum: 1 },
      unique_players: { $addToSet: "$player_id" }
    }
  },
  // 计算各事件的独立玩家数
  {
    $project: {
      _id: 0,
      event_type: "$_id",
      total_events: 1,
      unique_players: { $size: "$unique_players" }
    }
  },
  // 按事件数降序排列
  { $sort: { total_events: -1 } }
];

// ============================================================
// 4. 预置 revenueStats 聚合管道 (收入统计)
// ============================================================
print("\n--- 4. 预置 revenue_stats 聚合管道 ---");

const revenueStatsPipeline = [
  {
    $match: {
      event_type: { $in: ["recharge_success", "shop_purchase"] },
      event_date: "YYYY-MM-DD"
    }
  },
  {
    $group: {
      _id: "$event_type",
      total_amount: { $sum: { $ifNull: ["$properties.amount", 0] } },
      total_count: { $sum: 1 },
      unique_payers: { $addToSet: "$player_id" }
    }
  },
  {
    $project: {
      _id: 0,
      event_type: "$_id",
      total_amount: 1,
      total_count: 1,
      unique_payers: { $size: "$unique_payers" },
      avg_amount: {
        $cond: [
          { $gt: ["$total_count", 0] },
          { $divide: ["$total_amount", "$total_count"] },
          0
        ]
      }
    }
  }
];

// ============================================================
// 5. 预置 retentionStats 聚合管道 (留存统计)
// ============================================================
print("\n--- 5. 预置 retention_stats 聚合管道 ---");

const retentionStatsPipeline = (registerDate, targetDate) => {
  return [
    // 获取基准日期注册的用户
    {
      $match: {
        event_type: "player_register",
        event_date: registerDate
      }
    },
    {
      $group: {
        _id: null,
        new_users: { $addToSet: "$player_id" }
      }
    },
    // 查询目标日期这些用户是否登录
    {
      $lookup: {
        from: "analytics_events",
        let: { newUsers: "$new_users" },
        pipeline: [
          {
            $match: {
              $expr: {
                $and: [
                  { $eq: ["$event_type", "player_login"] },
                  { $eq: ["$event_date", targetDate] },
                  { $in: ["$player_id", "$$newUsers"] }
                ]
              }
            }
          },
          {
            $group: {
              _id: null,
              retained_users: { $addToSet: "$player_id" }
            }
          }
        ],
        as: "retention_data"
      }
    },
    {
      $project: {
        _id: 0,
        register_date: registerDate,
        target_date: targetDate,
        new_users: { $size: "$new_users" },
        retained_users: {
          $ifNull: [
            { $arrayElemAt: ["$retention_data.retained_users", 0] },
            []
          ]
        },
        retention_rate: {
          $round: [
            {
              $multiply: [
                {
                  $divide: [
                    {
                      $size: {
                        $ifNull: [
                          { $arrayElemAt: ["$retention_data.retained_users", 0] },
                          []
                        ]
                      }
                    },
                    { $size: "$new_users" }
                  ]
                },
                100
              ]
            },
            2
          ]
        }
      }
    }
  ];
};

// ============================================================
// 汇总信息
// ============================================================
print("\n========== analytics_events 初始化完成 ==========");
print("集合:");
print("  1. analytics_events - 运营分析事件 (TTL: 90 天)");
print("  2. daily_stats      - 日统计聚合 (TTL: 365 天)");
print("索引:");
print("  1. idx_type_date_timestamp  (event_type + event_date + timestamp)");
print("  2. idx_player_date_type     (player_id + event_date + event_type)");
print("  3. idx_timestamp            (timestamp)");
print("  4. idx_type_date_player     (event_type + event_date + player_id, partial)");
print("  5. idx_type_date_amount     (event_type + event_date + amount, partial)");
print("  6. idx_type_date_level      (event_type + event_date + level, partial)");
print("  7. idx_type_realm_timestamp (event_type + realm + timestamp)");
print("  8. idx_session_event        (session_id + event_type, partial)");
print("  9. idx_timestamp_ttl        (timestamp TTL 90天)");
print("聚合管道:");
print("  1. dailyStatsPipeline   - 日统计汇总");
print("  2. revenueStatsPipeline - 收入统计");
print("  3. retentionStatsPipeline - 留存统计");
print("===========================================");
