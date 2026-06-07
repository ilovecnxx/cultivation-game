// MongoDB initialization for cultivation-game
db = db.getSiblingDB('cultivation_game');

// Create collections with indexes
db.createCollection('chat_messages');
db.createCollection('system_logs');
db.createCollection('player_events');
db.createCollection('game_analytics');

// Indexes
db.chat_messages.createIndex({ "channel_id": 1, "created_at": -1 });
db.chat_messages.createIndex({ "sender_id": 1 });
db.chat_messages.createIndex({ "created_at": -1 });

db.system_logs.createIndex({ "service": 1, "level": 1, "created_at": -1 });
db.system_logs.createIndex({ "created_at": -1 });

db.player_events.createIndex({ "player_id": 1, "created_at": -1 });
db.player_events.createIndex({ "event_type": 1 });

db.game_analytics.createIndex({ "event": 1, "created_at": -1 });
db.game_analytics.createIndex({ "player_id": 1 });

print("MongoDB initialization completed for cultivation_game");
