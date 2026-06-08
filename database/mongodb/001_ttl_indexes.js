// ===================================================================
// MongoDB TTL 索引 — 自动过期历史数据，防止磁盘无限增长
// ===================================================================
// 使用方式:
//   mongosh --host localhost --port 27017 cultivation_game 001_ttl_indexes.js
// ===================================================================

// 聊天记录：30 天后自动删除
db.chat_messages.createIndex(
  { "timestamp": 1 },
  { expireAfterSeconds: 2592000, name: "ttl_chat_messages" }
);
print("Created TTL index on chat_messages.timestamp (30d)");

// 战斗回放：30 天后自动删除
db.battle_replays.createIndex(
  { "timestamp": 1 },
  { expireAfterSeconds: 2592000, name: "ttl_battle_replays" }
);
print("Created TTL index on battle_replays.timestamp (30d)");

// 事件日志：30 天后自动删除
db.event_logs.createIndex(
  { "timestamp": 1 },
  { expireAfterSeconds: 2592000, name: "ttl_event_logs" }
);
print("Created TTL index on event_logs.timestamp (30d)");

// 交易历史归档：90 天后自动删除
db.trade_history.createIndex(
  { "timestamp": 1 },
  { expireAfterSeconds: 7776000, name: "ttl_trade_history" }
);
print("Created TTL index on trade_history.timestamp (90d)");

print("=== All TTL indexes created ===");
