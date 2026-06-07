# 凡人修仙模拟器 — API 文档

> 版本: 2.0 | 最后更新: 2026-06-05  
> 本文档覆盖系统全部 API 端点，按服务分类，包含 WebSocket 消息和 RESTful API。

---

## 目录

1. [通用规范](#1-通用规范)
2. [Gateway 服务](#2-gateway-服务)
3. [Auth 服务](#3-auth-服务)
4. [Player 服务](#4-player-服务)
5. [Cultivation 服务](#5-cultivation-服务)
6. [Combat 服务](#6-combat-服务)
7. [Social 服务](#7-social-服务)
8. [World 服务](#8-world-服务)
9. [Trade 服务](#9-trade-服务)
10. [Ranking 服务](#10-ranking-服务)

---

## 1. 通用规范

### 1.1 通信方式

| 类型 | 端点 | 序列化 | 认证 |
|------|------|--------|------|
| WebSocket | `ws://host:8080/ws?token={jwt}` | Protobuf (GameMessage 信封) | URL Query token |
| HTTP REST | `http://host:8081/api/v1/...` | JSON | Authorization: Bearer {jwt} |
| HTTP 健康检查 | `http://host:xxxx/health` | JSON | 无 |

### 1.2 通用错误响应

```json
// HTTP 错误
{
  "code": 1001,
  "message": "参数错误",
  "details": [
    { "field": "username", "reason": "长度应在 3-32 个字符之间" }
  ]
}
```

### 1.3 HTTP 状态码含义

| 状态码 | 含义 | 处理方式 |
|--------|------|---------|
| 200 | 成功 | 正常解析响应体 |
| 400 | 参数错误 | 检查请求参数格式 |
| 401 | 未认证 / Token 过期 | 尝试 refresh_token，失败跳转登录页 |
| 403 | 权限不足 | 提示用户无此权限 |
| 404 | 资源不存在 | 检查请求路径或 ID |
| 429 | 请求频率超限 | 等待后重试（限流） |
| 500 | 服务器内部错误 | 记录日志，联系运维 |

### 1.4 WebSocket 消息 ID 前缀

| ID 范围 | 服务 |
|---------|------|
| 1-99 | 系统消息 (心跳/重连) |
| 100-199 | Auth |
| 200-299 | Player |
| 300-399 | Cultivation |
| 400-499 | Combat |
| 500-599 | Social |
| 600-699 | World |
| 700-799 | Trade (预留) |
| 800-899 | Ranking (预留) |
| 900-999 | 管理/系统 |

---

## 2. Gateway 服务

### 2.1 WebSocket 连接

```
URL:  ws://<host>:8080/ws?token=<access_token>
       wss://<host>/ws?token=<access_token>  (生产环境)
```

**连接流程：**
1. 客户端携带 JWT access_token 发起 WebSocket 连接
2. 服务端验证 token 有效性，解析 player_id
3. 建立长连接，开始心跳保活
4. 如验证失败，返回 401 并关闭连接

### 2.2 心跳

```protobuf
// Message ID: 1 (MSG_HEARTBEAT) / 2 (MSG_HEARTBEAT_ACK)

// Client -> Server (每 30 秒)
message HeartbeatReq {
  int64 client_time = 1;  // 客户端时间戳 (ms)
}

// Server -> Client
message HeartbeatResp {
  int64 server_time = 1;     // 服务端时间戳 (ms)
  int64 interval_ms = 2;     // 建议心跳间隔 (ms)
}
```

```
示例 (JSON 表示):
C->S: {"msg_id":1, "seq":0, "timestamp":1712345678000, "payload":{"client_time":1712345678000}}
S->C: {"msg_id":2, "seq":0, "timestamp":1712345678000, "payload":{"server_time":1712345678000,"interval_ms":30000}}
```

### 2.3 断线重连

```protobuf
// Message ID: 3 (MSG_RECONNECT)

// Client -> Server
message ReconnectReq {
  string session_id = 1;     // 原 session ID
  uint64 player_id = 2;      // 玩家 ID
  string token = 3;          // 当前 access_token
}

// Server -> Client
message ReconnectResp {
  bool success = 1;
  string new_session_id = 2;
  PlayerStateSnapshot snapshot = 3;  // 断线前的状态快照
}

message PlayerStateSnapshot {
  uint64 player_id = 1;
  string scene_id = 2;
  int32 hp = 3;
  int32 mp = 4;
  int64 last_action_time = 5;
}
```

### 2.4 健康检查

```
GET /health

Response 200:
{
  "status": "ok",
  "service": "gateway",
  "version": "2.0.0",
  "time": 1712345678,
  "uptime_seconds": 3600,
  "connections": 1250,
  "goroutines": 87
}
```

---

## 3. Auth 服务

基础路径: `/auth`

### 3.1 用户注册

```
POST /auth/register

Request Body:
{
  "username": "xiuxian_zhang",
  "password": "password123",
  "nickname": "修仙张三",
  "spirit_root": ""         // 可选，空字符串则随机分配
}

Response 200:
{
  "player_id": 42,
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}

Response 400:
{
  "code": 1001,
  "message": "参数错误",
  "details": [{ "field": "username", "reason": "该用户名已被注册" }]
}
```

### 3.2 用户登录

```
POST /auth/login

Request Body:
{
  "username": "xiuxian_zhang",
  "password": "password123",
  "device_id": "device_abc123",     // 可选
  "client_version": "2.0.0"         // 可选
}

Response 200:
{
  "player_id": 42,
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": 1712950478,
  "is_new_player": false
}

Response 401:
{
  "code": 1002,
  "message": "用户名或密码错误"
}
```

### 3.3 Token 刷新

```
POST /auth/refresh

Request Body:
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "player_id": 42
}

Response 200:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": 1712950478
}

Response 401:
{
  "code": 1002,
  "message": "refresh_token 已过期，请重新登录"
}
```

### 3.4 WebSocket 登录

```
// Message ID: 100 (MSG_AUTH_LOGIN) / 101 (MSG_AUTH_LOGIN_RES)
// 此消息通过 WebSocket 发送，与 HTTP 登录等效

// Client -> Server
message LoginReq {
  string username = 1;
  string password = 2;
  string device_id = 3;
  string client_version = 4;
}

// Server -> Client
message LoginResp {
  uint64 player_id = 1;
  string token = 2;
  string refresh_token = 3;
  int64 expires_at = 4;
  bool is_new_player = 5;
}
```

### 3.5 WebSocket 注册

```
// Message ID: 102 (MSG_AUTH_REGISTER) / 103 (MSG_AUTH_REGISTER_RES)

// Client -> Server
message RegisterReq {
  string username = 1;
  string password = 2;
  string nickname = 3;
  string spirit_root = 4;
}

// Server -> Client
message RegisterResp {
  uint64 player_id = 1;
  string token = 2;
  string refresh_token = 3;
}
```

---

## 4. Player 服务

基础路径: `/api/v1/player`

### 4.1 获取玩家信息

```
GET /api/v1/player/:id

Response 200:
{
  "player_id": 42,
  "nickname": "修仙张三",
  "realm_id": 10,
  "realm_name": "筑基初期",
  "realm_level": 1,
  "exp": 15000,
  "max_exp": 18000,
  "spirit_root": {
    "roots": ["火灵根", "木灵根"],
    "quality": "中品"
  },
  "hp": 1200,
  "max_hp": 2000,
  "mp": 800,
  "max_mp": 1500,
  "attack": 120,
  "defense": 90,
  "speed": 115,
  "combat_power": 4850,
  "cultivation_technique_id": 2,
  "money": 50000,
  "bind_money": 20000,
  "immortal_jade": 500,
  "vip_level": 2,
  "level": 25,
  "created_at": "2026-01-15T10:30:00Z",
  "last_login_at": "2026-06-05T10:00:00Z"
}
```

### 4.2 WebSocket 获取玩家信息

```
// Message ID: 200 (MSG_PLAYER_INFO) / 201 (MSG_PLAYER_INFO_RES)

// Client -> Server
message PlayerInfoReq {
  uint64 player_id = 1;
}

// Server -> Client
message PlayerInfoResp {
  uint64 player_id = 1;
  string nickname = 2;
  uint32 realm_id = 3;
  uint32 realm_level = 4;
  uint64 exp = 5;
  uint64 max_exp = 6;
  string spirit_root = 7;
  int32 hp = 8;
  int32 max_hp = 9;
  int32 mp = 10;
  int32 max_mp = 11;
  int64 attack = 12;
  int64 defense = 13;
  int32 speed = 14;
  int64 combat_power = 15;
  uint32 cultivation_technique_id = 16;
  int64 money = 17;
  int64 bind_money = 18;
  int32 immortal_jade = 19;
  int32 vip_level = 20;
  int32 level = 21;
}
```

### 4.3 获取背包

```
GET /api/v1/player/:id/inventory?page=1&page_size=20&filter_type=-1

Query Parameters:
  page:        页码 (默认 1)
  page_size:   每页数量 (默认 20, 最大 100)
  filter_type: 物品类型筛选 (-1=全部, 1=消耗品, 2=装备, 3=材料, 4=功法, 5=任务)

Response 200:
{
  "items": [
    {
      "instance_id": 1001,
      "item_id": 2,
      "name": "聚气丹",
      "type": 1,
      "quality": 1,
      "quantity": 20,
      "slot": 1,
      "is_equipped": false,
      "extra_data": null,
      "description": "回复少量真元，适合练气期使用"
    },
    {
      "instance_id": 1005,
      "item_id": 9,
      "name": "青锋剑",
      "type": 2,
      "quality": 2,
      "quantity": 1,
      "slot": 5,
      "is_equipped": true,
      "extra_data": { "durability": 95 },
      "description": "精铁所铸的制式长剑，锋利坚韧"
    }
  ],
  "total": 15,
  "page": 1,
  "page_size": 20,
  "max_slots": 100,
  "used_slots": 7
}
```

### 4.4 使用物品

```
POST /api/v1/player/:id/inventory/use

Request Body:
{
  "item_instance_id": 1001,
  "quantity": 1
}

Response 200:
{
  "code": 0,
  "message": "成功使用聚气丹 x1",
  "effect": {
    "type": "mp_restore",
    "value": 30,
    "description": "真元恢复 30 点"
  }
}

Response 400:
{
  "code": 1001,
  "message": "物品数量不足"
}
```

### 4.5 装备物品

```
POST /api/v1/player/:id/equipment/equip

Request Body:
{
  "item_instance_id": 1008,
  "slot": "weapon"    // weapon/armor/helmet/necklace/ring/treasure
}

Response 200:
{
  "success": true,
  "combat_power_change": 150,
  "slot": 1
}

Response 400:
{
  "code": 1001,
  "message": "该物品不能装备到此槽位"
}
```

### 4.6 卸下装备

```
POST /api/v1/player/:id/equipment/unequip

Request Body:
{
  "slot": "weapon"    // weapon/armor/helmet/necklace/ring/treasure
}

Response 200:
{
  "success": true,
  "combat_power_change": -150,
  "item_instance_id": 1008
}
```

### 4.7 获取装备列表

```
GET /api/v1/player/:id/equipment

Response 200:
{
  "equipment": {
    "weapon": {
      "instance_id": 1005,
      "item_id": 9,
      "name": "青锋剑",
      "base_attr": { "attack": 15, "defense": 2 },
      "enhance_level": 3,
      "gems": [1, 2]
    },
    "helmet": {
      "instance_id": 1006,
      "item_id": 11,
      "name": "紫金冠",
      "base_attr": { "defense": 12, "hp": 100 },
      "enhance_level": 2,
      "gems": null
    }
  },
  "total_combat_power": 4850
}
```

### 4.8 强化装备

```
POST /api/v1/player/:id/equipment/strengthen

Request Body:
{
  "slot": "weapon"
}

Response 200:
{
  "success": true,
  "new_level": 4,
  "cost_money": 500,
  "new_attributes": { "attack": 20, "defense": 3 }
}

Response 400:
{
  "code": 1001,
  "message": "灵石不足，强化需要 500 灵石"
}
```

### 4.9 获取已学功法

```
GET /api/v1/player/:id/techniques

Response 200:
{
  "techniques": [
    {
      "instance_id": 1,
      "technique_id": 2,
      "name": "长青功",
      "grade": 3,
      "level": 5,
      "exp": 800,
      "max_level": 12,
      "is_equipped": 1,       // 0=未装备, 1=主修, 2=辅修
      "cultivation_bonus": 15.0,
      "hp_bonus": 500,
      "attack_bonus": 20,
      "defense_bonus": 30
    },
    {
      "instance_id": 2,
      "technique_id": 3,
      "name": "紫霞功",
      "grade": 4,
      "level": 2,
      "exp": 200,
      "max_level": 15,
      "is_equipped": 2,
      "cultivation_bonus": 25.0,
      "mp_bonus": 500,
      "attack_bonus": 60,
      "defense_bonus": 40
    }
  ],
  "max_main": 1,
  "max_auxiliary": 2
}
```

### 4.10 玩家属性更新推送

```
// Message ID: 202 (MSG_PLAYER_UPDATE) - 服务端主动推送
// 当玩家属性发生变化时，服务端推送更新

// Server -> Client
message PlayerInfo {
  uint64 id = 1;
  string nickname = 2;
  uint32 realm_id = 3;
  uint32 realm_level = 4;
  uint64 exp = 5;
  int32 hp = 6;
  int32 mp = 7;
  int64 attack = 8;
  int64 defense = 9;
  int32 speed = 10;
  int64 combat_power = 11;
}
```

---

## 5. Cultivation 服务

基础路径: `/api/v1`

### 5.1 修炼

```
POST /api/v1/cultivate

Request Body:
{
  "player_id": 42,
  "technique_id": 2,
  "duration_minutes": 30,
  "use_elixir": true,
  "elixir_id": 5
}

Response 200:
{
  "exp_gained": 1440,
  "exp_per_hour": 2880,
  "total_exp": 16440,
  "current_realm_id": 10,
  "current_realm_name": "筑基初期",
  "current_realm_level": 1,
  "next_realm_exp": 18000,
  "triggered_breakthrough": false,
  "exp_bonuses": {
    "technique": 1.15,
    "elixir": 1.20,
    "realm": 1.0,
    "total_multiplier": 1.38
  }
}
```

**经验计算公式：**
```
exp_per_hour = base_exp * technique_bonus * realm_bonus * pill_bonus * artifact_bonus * global_rate
exp_gained = exp_per_hour * (duration_minutes / 60)
```

### 5.2 WebSocket 修炼

```
// Message ID: 300 (MSG_CULTIVATE) / 301 (MSG_CULTIVATE_RES)

// Client -> Server
message CultivateReq {
  uint32 technique_id = 1;
  uint32 duration_minutes = 2;
  bool use_elixir = 3;
  uint32 elixir_id = 4;
}

// Server -> Client
message CultivateResp {
  uint64 exp_gained = 1;
  uint32 exp_per_hour = 2;
  uint64 total_exp = 3;
  uint32 current_realm_id = 4;
  uint32 current_realm_level = 5;
  uint64 next_realm_exp = 6;
  bool triggered_breakthrough = 7;
}
```

### 5.3 境界突破

```
POST /api/v1/breakthrough

Request Body:
{
  "player_id": 42,
  "technique_id": 2,
  "assist_elixirs": [1],      // 辅助丹药 ID 列表
  "assist_treasure_id": 0,    // 辅助法宝 ID
  "protection_talisman_id": 0 // 护身符 ID
}

Response 200 (成功):
{
  "success": true,
  "success_rate": 0.30,
  "final_rate": 0.55,
  "new_realm_id": 14,
  "new_realm_name": "金丹初期",
  "new_realm_level": 1,
  "combat_power_increase": 5000,
  "triggered_tribulation": false,
  "description": "历经千辛万苦，道友终于凝聚金丹成功！寿元大增，实力飞跃！"
}

Response 200 (失败):
{
  "success": false,
  "success_rate": 0.30,
  "final_rate": 0.55,
  "failed_drop_levels": 1,
  "cooldown_seconds": 3600,
  "description": "突破失败，修为受损，境界退回练气八层。道友还需继续积累，不可操之过急。"
}
```

**成功率计算公式：**
```
final_rate = base_rate(0.30)
           + spirit_root_bonus(0.10)   // 灵根亲和加成
           + item_bonus(0.10)          // 辅助物品加成
           + luck_bonus(0.05)          // 气运加成
           - realm_penalty(0.05)       // 境界惩罚
           = 0.50
final_rate = min(final_rate, max_rate)  // 上限 0.85
```

### 5.4 WebSocket 突破

```
// Message ID: 302 (MSG_BREAKTHROUGH) / 303 (MSG_BREAKTHROUGH_RES)

// Client -> Server
message BreakthroughReq {
  uint32 technique_id = 1;
  repeated uint32 assist_elixirs = 2;
  uint32 assist_treasure_id = 3;
  uint32 protection_talisman_id = 4;
}

// Server -> Client
message BreakthroughResp {
  bool success = 1;
  float success_rate = 2;
  float final_rate = 3;
  uint32 new_realm_id = 4;
  uint32 new_realm_level = 5;
  int64 combat_power_increase = 6;
  bool triggered_tribulation = 7;
  TribulationInfo tribulation = 8;
  string description = 9;
}

message TribulationInfo {
  uint32 tribulation_type = 1;
  int32 difficulty = 2;
  repeated TribulationRound rounds = 3;
}

message TribulationRound {
  uint32 round_num = 1;
  string trial_description = 2;
  int32 required_attribute = 3;
  int32 player_value = 4;
  int32 threshold = 5;
  bool passed = 6;
  string outcome = 7;
}
```

### 5.5 天劫信息查询

```
GET /api/v1/tribulation/info?player_id=42

Response 200:
{
  "tribulation_type": 1,
  "difficulty": 8500,
  "player_combat_power": 4850,
  "tribulation_power": 7200,
  "recommended": "道友当前实力不足以渡劫，建议提升至 8000 战力以上再尝试",
  "rounds": [
    { "round": 1, "trial": "天雷淬体", "attribute": "defense", "threshold": 150, "player_value": 90 },
    { "round": 2, "trial": "心魔试炼", "attribute": "spirit", "threshold": 120, "player_value": 80 },
    { "round": 3, "trial": "神识风暴", "attribute": "agility", "threshold": 100, "player_value": 50 },
    { "round": 4, "trial": "天道问心", "attribute": "comprehension", "threshold": 30, "player_value": 30 }
  ]
}
```

### 5.6 功法列表

```
GET /api/v1/techniques?player_id=42

Response 200:
{
  "techniques": [
    {
      "technique_id": 1,
      "name": "基础吐纳法",
      "grade": 1,
      "grade_name": "黄阶",
      "element": 0,
      "max_level": 9,
      "level_required": 1,
      "cultivation_bonus": 5.0,
      "hp_bonus": 50,
      "attack_bonus": 5,
      "defense_bonus": 5,
      "is_learned": true,
      "is_equipped": false,
      "description": "修仙界最基础的呼吸吐纳之法，人人可学"
    },
    {
      "technique_id": 2,
      "name": "长青功",
      "grade": 3,
      "grade_name": "地阶",
      "element": 2,
      "max_level": 12,
      "level_required": 5,
      "cultivation_bonus": 15.0,
      "hp_bonus": 500,
      "attack_bonus": 20,
      "defense_bonus": 30,
      "is_learned": true,
      "is_equipped": true,
      "description": "乙木灵气淬体，生生不息，擅长回复与防御"
    }
  ],
  "can_learn": [
    {
      "technique_id": 4,
      "name": "九转金身诀",
      "grade": 5,
      "grade_name": "仙阶",
      "element": 1,
      "require_item_id": 26,
      "require_item_name": "九转金身诀残卷",
      "description": "传说级炼体功法，九转大成则肉身不灭"
    }
  ]
}
```

### 5.7 学习功法

```
POST /api/v1/technique/learn

Request Body:
{
  "player_id": 42,
  "technique_id": 4,
  "item_instance_id": 2001  // 功法秘籍的物品实例 ID
}

Response 200:
{
  "success": true,
  "technique_name": "九转金身诀",
  "message": "成功学习功法【九转金身诀】！"
}

Response 400:
{
  "code": 1001,
  "message": "境界不足，需要金丹期以上才能修炼此功法"
}
```

### 5.8 装备 / 切换功法

```
POST /api/v1/technique/equip

Request Body:
{
  "player_id": 42,
  "technique_id": 3,
  "slot": "main"     // "main" = 主修, "auxiliary" = 辅修
}

Response 200:
{
  "success": true,
  "slot": "main",
  "technique_name": "紫霞功"
}
```

### 5.9 开始闭关 (冥想)

```
POST /api/v1/meditate/start

Request Body:
{
  "player_id": 42,
  "technique_id": 2,
  "duration_minutes": 120  // 最长 1440 (24h)
}

Response 200:
{
  "success": true,
  "start_time": 1712345678,
  "estimated_exp": 5760,
  "end_time": 1712352878,
  "message": "道友开始闭关，预计 2 小时后出关"
}
```

### 5.10 领取闭关收益

```
POST /api/v1/meditate/claim

Request Body:
{
  "player_id": 42
}

Response 200:
{
  "success": true,
  "actual_duration_minutes": 120,
  "exp_gained": 5760,
  "bonus_exp": 576,      // 额外奖励 (满时长 10%)
  "total_exp": 6336,
  "triggered_breakthrough": false
}
```

### 5.11 炼丹

```
POST /api/v1/alchemy/refine

Request Body:
{
  "player_id": 42,
  "recipe_id": 1,
  "materials": [
    { "item_id": 20, "quantity": 3 },     // 妖兽内丹 x3
    { "item_id": 21, "quantity": 5 }      // 玄铁 x5
  ],
  "technique_id": 2   // 用炼丹术功法
}

Response 200:
{
  "success": true,
  "product_item_id": 1,
  "product_name": "筑基丹",
  "product_quantity": 2,
  "quality": 4,          // 上品
  "quality_name": "上品",
  "exp_gained": 500
}

Response 400:
{
  "success": false,
  "message": "炼丹失败，材料化为灰烬",
  "lost_materials": true
}
```

---

## 6. Combat 服务

基础路径: `/api/v1`

### 6.1 怪物列表

```
GET /api/v1/pve/monsters

Response 200:
{
  "monsters": [
    {
      "id": 1,
      "name": "风狼",
      "level": 1,
      "realm_group": 1,
      "element": "wind",
      "hp": 200,
      "attack": 30,
      "defense": 15,
      "speed": 120,
      "exp_reward": 50,
      "money_reward": 20,
      "drops": [ { "item_id": 19, "name": "灵石碎片", "rate": 0.5 } ],
      "description": "迅捷如风的一阶妖兽，常在低阶区域出没"
    },
    {
      "id": 38,
      "name": "远古蛟龙",
      "level": 35,
      "realm_group": 8,
      "element": "water",
      "hp": 500000,
      "attack": 12000,
      "defense": 8000,
      "speed": 500,
      "exp_reward": 50000,
      "money_reward": 20000,
      "drops": [ { "item_id": 22, "name": "千年灵木", "rate": 0.8 } ],
      "description": "渡劫期远古蛟龙，盘踞深海"
    }
  ]
}
```

### 6.2 副本列表

```
GET /api/v1/pve/instances

Response 200:
{
  "instances": [
    {
      "id": 1,
      "name": "灵雾森林",
      "level_required": 1,
      "realm_required": 1,
      "monsters": [1, 2, 3],
      "boss_id": 4,
      "exp_reward": 500,
      "money_reward": 200,
      "daily_limit": 5,
      "description": "练气期修炼秘境，妖兽众多"
    },
    {
      "id": 5,
      "name": "天劫炼狱",
      "level_required": 30,
      "realm_required": 8,
      "monsters": [35, 36, 37],
      "boss_id": 38,
      "exp_reward": 100000,
      "money_reward": 50000,
      "daily_limit": 1,
      "description": "渡劫期终极试炼"
    }
  ]
}
```

### 6.3 开始 PVE 战斗

```
POST /api/v1/pve/battle
或 WebSocket Message ID: 400 (MSG_COMBAT_START)

Request Body:
{
  "player_id": 42,
  "instance_id": 1,
  "monster_id": 3,        // 直接指定怪物 (跳过副本)
  "target_player_id": 0   // PVP 时使用
}

// WebSocket 等效消息
message CombatStartReq {
  uint32 instance_id = 1;
  uint32 monster_id = 2;
  uint64 target_player_id = 3;
}

Response 200:
{
  "combat_id": "battle_uuid_xxxx",
  "combat_type": "pve",
  "player_state": {
    "entity_id": 42,
    "name": "修仙张三",
    "hp": 1800,
    "max_hp": 2000,
    "mp": 1400,
    "max_mp": 1500,
    "shield": 0,
    "speed": 115,
    "buffs": []
  },
  "enemy_state": {
    "entity_id": 2000003,
    "name": "赤焰蛇",
    "hp": 500,
    "max_hp": 500,
    "mp": 100,
    "max_mp": 100,
    "shield": 0,
    "speed": 90,
    "buffs": []
  },
  "total_rounds_estimate": 5,
  "description": "你进入灵雾森林深处，一条赤焰蛇拦住了去路！"
}
```

### 6.4 提交战斗行动

```
POST /api/v1/pve/action
或 WebSocket Message ID: 401 (MSG_COMBAT_ACTION)

Request Body:
{
  "combat_id": "battle_uuid_xxxx",
  "skill_id": 6,         // 使用技能 ID
  "action_type": "skill", // skill/defend/use_item/flee
  "item_id": 0
}

// WebSocket 等效消息
message CombatActionReq {
  uint64 combat_id = 1;
  uint32 skill_id = 2;
  string action_type = 3;
  uint32 item_id = 4;
}

Response 200:
{
  "rounds": [
    {
      "round_num": 1,
      "attacker_name": "修仙张三",
      "skill_name": "飞剑术",
      "description": "你御使飞剑化作一道流光，直刺赤焰蛇七寸之处！",
      "damage": 344,
      "is_critical": false,
      "is_miss": false,
      "is_blocked": false,
      "hp_change": -344,
      "element_effect": {
        "element_type": "metal",
        "multiplier": 1.3,
        "description": "金克火，伤害提升30%！"
      }
    },
    {
      "round_num": 1,
      "attacker_name": "赤焰蛇",
      "skill_name": "毒牙",
      "description": "赤焰蛇张开血盆大口，向你咬来！",
      "damage": 86,
      "is_critical": false,
      "is_miss": true,
      "is_blocked": false,
      "hp_change": 0,
      "element_effect": null
    }
  ],
  "updated_player": {
    "entity_id": 42,
    "name": "修仙张三",
    "hp": 1800,
    "max_hp": 2000,
    "mp": 1360,
    "max_mp": 1500,
    "shield": 0,
    "speed": 115,
    "buffs": []
  },
  "updated_enemy": {
    "entity_id": 2000003,
    "name": "赤焰蛇",
    "hp": 156,
    "max_hp": 500,
    "mp": 100,
    "max_mp": 100,
    "shield": 0,
    "speed": 90,
    "buffs": [ { "buff_id": 1, "name": "灼烧", "remaining_rounds": 2, "effect_description": "每回合损失 20 点气血" } ]
  },
  "combat_ended": false,
  "victory": false,
  "reward": null,
  "narrative": "飞剑精准命中！金克火效果显著！赤焰蛇身受重伤，但仍未放弃战斗。"
}
```

**伤害计算规则：**
```
1. 基础伤害 = skill.base_damage + skill.coefficient * attacker.strength
2. 暴击判定: random(0, 1) < crit_rate     -> 伤害 *= (1.5 + crit_bonus)
3. 闪避判定: random(0, 1) < target.dodge_rate -> 伤害 = 0 (闪避)
4. 五行加成: 金克木/木克土/土克水/水克火/火克金 -> 伤害 *= 1.3
             被克 -> 伤害 *= 0.7
5. 防御减伤: final = damage - target.defense / 10 (至少 1 点)
```

### 6.5 战斗结果 (服务端主动推送)

```
// 战斗结束时，通过 WebSocket 推送结果
// Message ID: 402 (MSG_COMBAT_RESULT)

// Server -> Client
message CombatActionResp {
  repeated CombatRound rounds = 1;
  CombatState updated_player = 2;
  CombatState updated_enemy = 3;
  bool combat_ended = 4;
  bool victory = 5;
  CombatReward reward = 6;
  string narrative = 7;
}

message CombatReward {
  uint64 exp = 1;
  int64 money = 2;
  repeated ItemDrop items = 3;
  uint32 reputation = 4;
}

message ItemDrop {
  uint32 item_id = 1;
  string item_name = 2;
  uint32 quantity = 3;
  uint32 quality = 4;
  bool is_rare = 5;
}
```

### 6.6 PVP 匹配

```
POST /api/v1/pvp/join

Request Body:
{
  "player_id": 42,
  "match_type": "ranked"    // ranked / casual
}

Response 200 (已匹配):
{
  "matched": true,
  "combat_id": "pvp_uuid_xxxx",
  "opponent_id": 58,
  "opponent_name": "剑心",
  "opponent_realm_id": 5,
  "opponent_level": 5,
  "estimated_wait": 0
}

Response 200 (排队中):
{
  "matched": false,
  "estimated_wait": 15,
  "queue_position": 3,
  "message": "你已加入匹配队列，预计等待 15 秒"
}
```

### 6.7 查询匹配状态

```
GET /api/v1/pvp/queue-status?player_id=42

Response 200:
{
  "in_queue": true,
  "queue_position": 3,
  "estimated_wait_seconds": 15
}
```

### 6.8 匹配 WebSocket 消息

```
// Message ID: 403 (MSG_PVP_MATCH) / 404 (MSG_PVP_MATCH_RES)

message PvpMatchReq {
  uint64 player_id = 1;
  string match_type = 2;   // "ranked" / "casual"
}

message PvpMatchResp {
  bool matched = 1;
  uint64 combat_id = 2;
  uint64 opponent_id = 3;
  string opponent_name = 4;
  int32 opponent_realm_id = 5;
  int32 opponent_level = 6;
  int32 estimated_wait = 7;
}
```

---

## 7. Social 服务

基础路径: `/api/v1`

### 7.1 发送聊天消息

```
POST /api/v1/chat/send
或 WebSocket Message ID: 500 (MSG_CHAT_SEND)

Request Body:
{
  "player_id": 42,
  "channel": 0,           // 0=世界, 1=宗门, 2=私聊, 3=系统
  "content": "各位道友，今日可有人同闯秘境？",
  "target_id": 0,         // 私聊目标 ID
  "extra_data": ""        // JSON: 物品链接等
}

// WebSocket 等效消息
message ChatSendReq {
  uint32 channel = 1;
  string content = 2;
  uint64 target_id = 3;
  string extra_data = 4;
}

Response 200:
{
  "success": true,
  "msg_id": 5001,
  "timestamp": 1712345678
}

Response 400:
{
  "code": 1001,
  "message": "消息包含敏感词"
}
```

### 7.2 接收聊天消息 (WebSocket 推送)

```
// Server -> Client (WebSocket 主动推送)
// Message ID: 501 (MSG_CHAT_RECV)

message ChatMessage {
  uint64 msg_id = 1;
  uint32 channel = 2;
  uint64 sender_id = 3;
  string sender_name = 4;
  uint32 sender_level = 5;
  uint32 sender_realm = 6;
  string content = 7;
  int64 timestamp = 8;
  string extra_data = 9;
}

// 示例 JSON
{
  "msg_id": 5001,
  "channel": 0,
  "sender_id": 42,
  "sender_name": "修仙张三",
  "sender_level": 25,
  "sender_realm": 2,
  "content": "各位道友，今日可有人同闯秘境？",
  "timestamp": 1712345678000,
  "extra_data": ""
}
```

### 7.3 获取聊天历史

```
GET /api/v1/chat/history?channel=0&before_id=5001&limit=50

Query Parameters:
  channel:   频道 (0=世界, 1=宗门, 2=私聊, 3=系统)
  before_id: 分页起始消息 ID (返回比此 ID 更早的消息)
  limit:     数量 (默认 50, 最大 100)

Response 200:
{
  "messages": [
    {
      "msg_id": 4900,
      "channel": 0,
      "sender_id": 35,
      "sender_name": "月影仙子",
      "content": "刚突破元婴期，庆祝一下！",
      "timestamp": 1712345000000,
      "extra_data": ""
    }
  ],
  "has_more": true
}
```

### 7.4 获取好友列表

```
GET /api/v1/friends?player_id=42

Response 200:
{
  "friends": [
    {
      "player_id": 58,
      "nickname": "剑心",
      "realm_id": 5,
      "level": 5,
      "is_online": true,
      "last_online": 1712345678,
      "friendship": 85
    }
  ],
  "pending": [
    {
      "player_id": 60,
      "nickname": "玄天散人",
      "realm_id": 1,
      "level": 1,
      "is_online": false,
      "last_online": 1712340000,
      "friendship": 0
    }
  ]
}
```

### 7.5 发送好友申请

```
POST /api/v1/friends/apply
或 WebSocket Message ID: 502 (MSG_FRIEND_ADD)

Request Body:
{
  "player_id": 42,
  "target_id": 60,
  "message": "道友，可愿结为好友？"
}

Response 200:
{
  "success": true,
  "message": "好友申请已发送"
}
```

### 7.6 处理好友申请

```
POST /api/v1/friends/respond

Request Body:
{
  "player_id": 42,
  "target_id": 60,
  "accept": true
}

Response 200:
{
  "success": true,
  "message": "已添加 玄天散人 为好友"
}
```

### 7.7 删除好友

```
POST /api/v1/friends/remove

Request Body:
{
  "player_id": 42,
  "target_id": 60
}

Response 200:
{
  "success": true,
  "message": "已删除好友"
}
```

### 7.8 创建宗门

```
POST /api/v1/sect/create

Request Body:
{
  "name": "青云宗",
  "leader_id": 42,
  "notice": "天地有道，万物有灵。青云宗门，广纳贤才。"
}

Response 200:
{
  "success": true,
  "sect_id": 1,
  "message": "青云宗创立成功！"
}

Response 400:
{
  "code": 1001,
  "message": "灵石不足，创建宗门需要 10000 灵石"
}
```

### 7.9 获取宗门信息

```
GET /api/v1/sect/info?sect_id=1

Response 200:
{
  "sect_id": 1,
  "name": "青云宗",
  "level": 5,
  "leader_id": 42,
  "leader_name": "修仙张三",
  "member_count": 4,
  "max_members": 50,
  "notice": "天地有道，万物有灵。青云宗门，广纳贤才。",
  "total_contribution": 146000,
  "members": [
    {
      "player_id": 42,
      "nickname": "修仙张三",
      "position": 1,
      "position_name": "宗主",
      "contribution": 50000,
      "is_online": true
    },
    {
      "player_id": 58,
      "nickname": "剑心",
      "position": 3,
      "position_name": "精英",
      "contribution": 15000,
      "is_online": true
    }
  ]
}
```

### 7.10 申请加入宗门

```
POST /api/v1/sect/join

Request Body:
{
  "player_id": 58,
  "sect_id": 1
}

Response 200:
{
  "success": true,
  "message": "申请已发送，等待宗主审批"
}
```

### 7.11 审批入宗申请

```
POST /api/v1/sect/approve

Request Body:
{
  "sect_id": 1,
  "player_id": 60,
  "approve": true,
  "admin_id": 42  // 宗主或长老
}

Response 200:
{
  "success": true,
  "message": "已同意 玄天散人 加入宗门"
}
```

### 7.12 邮件列表

```
GET /api/v1/mail/inbox?player_id=42&page=1&page_size=20

Response 200:
{
  "mails": [
    {
      "id": 1001,
      "sender_id": 0,
      "sender_name": "系统",
      "title": "欢迎来到修仙大陆",
      "content": "道友初入仙途，特赠薄礼以助修行...",
      "has_attachment": true,
      "attachment": { "item_id": 2, "item_name": "聚气丹", "quantity": 10 },
      "is_read": false,
      "is_claimed": false,
      "created_at": "2026-06-01T00:00:00Z"
    },
    {
      "id": 1002,
      "sender_id": 58,
      "sender_name": "剑心",
      "title": "明日秘境探索",
      "content": "青云兄，明日宗门秘境探索，可否同往？",
      "has_attachment": false,
      "attachment": null,
      "is_read": true,
      "is_claimed": false,
      "created_at": "2026-06-04T14:30:00Z"
    }
  ],
  "total": 6,
  "unread": 3,
  "page": 1,
  "page_size": 20
}
```

### 7.13 领取邮件附件

```
POST /api/v1/mail/claim

Request Body:
{
  "player_id": 42,
  "mail_id": 1001
}

Response 200:
{
  "success": true,
  "items": [{ "item_id": 2, "name": "聚气丹", "quantity": 10 }],
  "message": "领取成功"
}
```

### 7.14 发送系统邮件 (管理)

```
POST /api/v1/mail/system
需要管理员权限

Request Body:
{
  "receiver_id": 42,
  "title": "活动奖励",
  "content": "恭喜道友在宗门大比中获得第一名！",
  "items": [{ "item_id": 22, "quantity": 3 }]
}

Response 200:
{
  "success": true,
  "mail_id": 2001
}
```

---

## 8. World 服务

基础路径: `/api/v1`

### 8.1 探索区域

```
POST /api/v1/world/explore
或 WebSocket Message ID: 600 (MSG_WORLD_EXPLORE)

Request Body:
{
  "player_id": 42,
  "region_id": 3,
  "direction": "north"
}

// WebSocket 等效消息
message ExploreReq {
  uint64 player_id = 1;
  uint32 region_id = 2;
  string direction = 3;
}

Response 200:
{
  "region_id": 5,
  "region_name": "青木林深处",
  "description": "林木愈发茂密，空气中弥漫着浓郁的灵气。几株千年灵芝若隐若现。",
  "available_directions": ["north", "south", "east"],
  "npcs": [
    {
      "npc_id": 5,
      "name": "药老",
      "title": "采药老人",
      "description": "一位鹤发童颜的老者，背着一个大药篓。",
      "available_actions": ["对话", "交易"],
      "avatar": "npc/yao_lao.png"
    }
  ],
  "resources": [
    {
      "resource_id": 3,
      "name": "灵芝",
      "quantity": 5,
      "gather_time": 10,
      "can_gather": true
    }
  ],
  "has_encounter": true,
  "encounter": {
    "encounter_id": 12,
    "title": "迷雾迷踪",
    "description": "一阵诡异的雾气突然从四面八方涌来，你迷失了方向...",
    "choices": [
      {
        "choice_id": 1,
        "label": "运功驱散雾气",
        "description": "耗费法力强行驱散迷雾",
        "requirement_type": 1,
        "requirement_value": 500,
        "requirement_text": "需要灵力 > 500"
      },
      {
        "choice_id": 2,
        "label": "原地等待",
        "description": "静待雾气自然散去",
        "requirement_type": 0,
        "requirement_value": 0,
        "requirement_text": "无要求"
      }
    ]
  }
}
```

### 8.2 奇遇选择

```
POST /api/v1/world/encounter/act
或 WebSocket Message ID: 602 (MSG_WORLD_ENCOUNTER)

Request Body:
{
  "player_id": 42,
  "encounter_id": 12,
  "choice_id": 1
}

// WebSocket 等效消息
message EncounterActReq {
  uint64 player_id = 1;
  uint32 encounter_id = 2;
  uint32 choice_id = 3;
}

Response 200:
{
  "success": true,
  "outcome_description": "你运足灵力，大喝一声！浓郁的雾气被震散开来，一缕金色阳光洒落，你发现了一株千年灵芝！",
  "exp_reward": 2000,
  "money_reward": 500,
  "item_reward_id": 22,
  "item_reward_name": "千年灵木",
  "item_reward_quantity": 1,
  "reputation_change": 10,
  "narrative": "迷雾深处暗藏机缘，这株千年灵芝足以让你的炼丹术更进一步。"
}
```

### 8.3 采集资源

```
POST /api/v1/world/gather

Request Body:
{
  "player_id": 42,
  "resource_id": 3,
  "region_id": 5
}

Response 200:
{
  "success": true,
  "item_id": 22,
  "item_name": "千年灵木",
  "quantity": 1,
  "remaining": 4,
  "next_refresh_seconds": 300
}

Response 400:
{
  "code": 1001,
  "message": "今日采集已达上限 (5/5)"
}
```

### 8.4 移动

```
POST /api/v1/world/move
或 WebSocket Message ID: 601 (MSG_WORLD_MOVE)

Request Body:
{
  "player_id": 42,
  "target_region_id": 8
}

// WebSocket 等效消息
message MoveReq {
  uint64 player_id = 1;
  uint32 target_region_id = 2;
}

Response 200:
{
  "success": true,
  "region_name": "灵药谷",
  "description": "四面环山的幽谷，灵气充沛，各种珍稀灵药遍地生长。",
  "travel_time_seconds": 30,
  "encounter_on_way": null
}
```

### 8.5 NPC 对话

```
POST /api/v1/world/npc/talk

Request Body:
{
  "player_id": 42,
  "npc_id": 5,
  "action": "对话"
}

Response 200:
{
  "npc_name": "药老",
  "dialogues": [
    { "speaker": "药老", "text": "年轻人，你也来采药？" },
    { "speaker": "药老", "text": "这青木林深处有不少好东西，但也不是随便什么人都能拿到的。" }
  ],
  "available_actions": ["交易", "接取任务", "离开"]
}
```

---

## 9. Trade 服务

基础路径: `/api/v1/trade`

### 9.1 交易行列表

```
GET /api/v1/trade/listings?page=1&page_size=20&item_type=0&sort=price_asc

Query Parameters:
  page:       页码 (默认 1)
  page_size:  每页数量 (默认 20, 最大 100)
  item_type:  物品类型 (0=全部, 1=消耗品, 2=装备, 3=材料, 4=功法)
  sort:       排序 (price_asc / price_desc / newest / oldest)
  search:     搜索关键词 (物品名)

Response 200:
{
  "listings": [
    {
      "id": 1001,
      "seller_id": 58,
      "seller_name": "剑心",
      "item_id": 20,
      "item_name": "妖兽内丹",
      "quantity": 5,
      "unit_price": 1500,
      "total_price": 7500,
      "created_at": "2026-06-04T10:00:00Z",
      "expires_at": "2026-06-11T10:00:00Z"
    },
    {
      "id": 1002,
      "seller_id": 42,
      "seller_name": "修仙张三",
      "item_id": 22,
      "item_name": "千年灵木",
      "quantity": 2,
      "unit_price": 5500,
      "total_price": 11000,
      "created_at": "2026-06-03T15:00:00Z",
      "expires_at": "2026-06-10T15:00:00Z"
    }
  ],
  "total": 8,
  "page": 1,
  "page_size": 20,
  "filters": { "item_types": [...], "quality_range": {...} }
}
```

### 9.2 上架物品

```
POST /api/v1/trade/sell

Request Body:
{
  "player_id": 42,
  "item_instance_id": 2001,
  "quantity": 3,
  "unit_price": 1500,
  "duration_hours": 168    // 7 天 (默认), 最长 30 天
}

Response 200:
{
  "success": true,
  "listing_id": 2001,
  "total_price": 4500,
  "fee": 225,               // 5% 上架费 (立即扣除)
  "expires_at": "2026-06-12T10:00:00Z"
}

Response 400:
{
  "code": 1001,
  "message": "灵石不足，上架需要 225 灵石手续费"
}
```

### 9.3 购买物品

```
POST /api/v1/trade/buy

Request Body:
{
  "buyer_id": 58,
  "listing_id": 1001,
  "quantity": 2
}

Response 200:
{
  "success": true,
  "item_id": 20,
  "item_name": "妖兽内丹",
  "quantity": 2,
  "total_cost": 3000,
  "tax": 150,              // 5% 交易税
  "remaining": 3            // 该挂单剩余数量
}

Response 400:
{
  "code": 1001,
  "message": "灵石不足"
}
```

### 9.4 下架物品

```
POST /api/v1/trade/cancel

Request Body:
{
  "player_id": 42,
  "listing_id": 1002
}

Response 200:
{
  "success": true,
  "item_id": 22,
  "item_name": "千年灵木",
  "quantity": 2,
  "message": "已从交易行下架"
}
```

### 9.5 我的订单

```
GET /api/v1/trade/my-orders?player_id=42&status=active

Query Parameters:
  status: active / sold / cancelled / expired (默认 active)

Response 200:
{
  "active": [
    {
      "listing_id": 1002,
      "item_name": "千年灵木",
      "quantity": 2,
      "unit_price": 5500,
      "total_price": 11000,
      "created_at": "2026-06-03T15:00:00Z",
      "expires_at": "2026-06-10T15:00:00Z"
    }
  ],
  "sold": [...],
  "total_earnings": 35000
}
```

### 9.6 拍卖列表

```
GET /api/v1/trade/auctions?page=1&page_size=20

Response 200:
{
  "auctions": [
    {
      "id": 1,
      "item_id": 18,
      "item_name": "玉清剑",
      "quality": 6,
      "quality_name": "神话",
      "seller_id": 5,
      "seller_name": "龙傲天",
      "current_bid": 80000,
      "bid_count": 5,
      "bidder_id": 3,
      "bidder_name": "月影仙子",
      "reserve_price": 50000,
      "end_time": "2026-06-06T20:00:00Z",
      "remaining_seconds": 36000,
      "status": "active"
    }
  ],
  "total": 1
}
```

### 9.7 出价拍卖

```
POST /api/v1/trade/auction/bid

Request Body:
{
  "player_id": 3,
  "auction_id": 1,
  "bid_amount": 85000
}

Response 200:
{
  "success": true,
  "current_bid": 85000,
  "message": "出价成功，你是当前最高出价者"
}

Response 400:
{
  "code": 1001,
  "message": "出价必须高于当前最高价"
}
```

### 9.8 交易记录

```
GET /api/v1/trade/transactions?player_id=42&page=1&page_size=20

Response 200:
{
  "transactions": [
    {
      "id": 5001,
      "type": "buy",
      "item_name": "妖兽内丹",
      "quantity": 2,
      "unit_price": 1500,
      "total": 3000,
      "tax": 150,
      "other_party": "剑心",
      "created_at": "2026-06-04T10:05:00Z"
    },
    {
      "id": 5000,
      "type": "sell",
      "item_name": "千年灵木",
      "quantity": 1,
      "unit_price": 6000,
      "total": 6000,
      "tax": 300,
      "other_party": "月影仙子",
      "created_at": "2026-06-03T16:00:00Z"
    }
  ],
  "total": 15,
  "total_bought": 15000,
  "total_sold": 35000,
  "total_tax_paid": 2500
}
```

---

## 10. Ranking 服务

基础路径: `/api/v1/ranking`

### 10.1 PVP 排行榜

```
GET /api/v1/ranking/pvp?season=1&page=1&page_size=20

Response 200:
{
  "season": 1,
  "season_name": "S1 紫气东来",
  "start_time": "2026-06-01T00:00:00Z",
  "end_time": "2026-06-30T23:59:59Z",
  "rankings": [
    {
      "rank": 1,
      "player_id": 5,
      "nickname": "龙傲天",
      "rating": 2500,
      "win_count": 180,
      "lose_count": 20,
      "win_rate": "90.0%",
      "realm_name": "化神初期",
      "combat_power": 52000,
      "sect_name": "天剑阁"
    },
    {
      "rank": 2,
      "player_id": 3,
      "nickname": "月影仙子",
      "rating": 2200,
      "win_count": 150,
      "lose_count": 35,
      "win_rate": "81.1%",
      "realm_name": "化神初期",
      "combat_power": 48000,
      "sect_name": "青云宗"
    },
    {
      "rank": 3,
      "player_id": 1,
      "nickname": "青云道人",
      "rating": 1800,
      "win_count": 100,
      "lose_count": 50,
      "win_rate": "66.7%",
      "realm_name": "筑基初期",
      "combat_power": 4850,
      "sect_name": "青云宗"
    }
  ],
  "total_players": 50,
  "player_rank": {
    "player_id": 42,
    "rank": 15,
    "rating": 1250
  }
}
```

### 10.2 境界排行榜

```
GET /api/v1/ranking/realm?page=1&page_size=20

Response 200:
{
  "rankings": [
    {
      "rank": 1,
      "player_id": 5,
      "nickname": "龙傲天",
      "realm_name": "化神初期",
      "realm_level": 1,
      "exp": 700000,
      "combat_power": 52000,
      "sect_name": "天剑阁"
    },
    {
      "rank": 2,
      "player_id": 3,
      "nickname": "月影仙子",
      "realm_name": "金丹初期",
      "realm_level": 1,
      "exp": 75000,
      "combat_power": 48000,
      "sect_name": "青云宗"
    }
  ],
  "total_players": 50
}
```

### 10.3 战力排行榜

```
GET /api/v1/ranking/combat-power?page=1&page_size=20

Response 200:
{
  "rankings": [
    {
      "rank": 1,
      "player_id": 5,
      "nickname": "龙傲天",
      "combat_power": 52000,
      "realm_name": "化神初期",
      "level": 37,
      "sect_name": "天剑阁"
    }
  ]
}
```

### 10.4 财富排行榜

```
GET /api/v1/ranking/wealth?page=1&page_size=20

Response 200:
{
  "rankings": [
    {
      "rank": 1,
      "player_id": 5,
      "nickname": "龙傲天",
      "wealth": 5000000,
      "realm_name": "化神初期",
      "sect_name": "天剑阁"
    }
  ]
}
```

### 10.5 玩家个人排行

```
GET /api/v1/ranking/player?player_id=42

Response 200:
{
  "player_id": 42,
  "nickname": "修仙张三",
  "realm_name": "筑基初期",
  "rankings": {
    "pvp": { "rank": 15, "rating": 1250, "season": 1 },
    "realm": { "rank": 8 },
    "combat_power": { "rank": 10, "value": 4850 },
    "wealth": { "rank": 20, "value": 50000 }
  }
}
```

---

> 相关文件：
> - Protobuf 协议定义: `/root/cultivation-game/proto/` (auth.proto, player.proto, cultivation.proto, combat.proto, social.proto, world.proto, gateway.proto, common.proto)
> - 数据库表定义: `/root/cultivation-game/database/mysql/` (6 个 SQL 文件, 24 张表)
> - 前端 API 封装: `/root/cultivation-game/frontend/src/core/api.ts`
> - WebSocket 客户端: `/root/cultivation-game/frontend/src/core/network/WebSocketClient.ts`
> - 消息编解码: `/root/cultivation-game/frontend/src/core/network/MessageCodec.ts`
