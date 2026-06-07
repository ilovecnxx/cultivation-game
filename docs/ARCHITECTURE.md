# 凡人修仙模拟器 — 系统架构文档

> 版本: 2.0 | 最后更新: 2026-06-05  
> 本文档覆盖微服务拓扑、数据流、数据库设计、通信协议、安全体系、性能指标与扩展性设计。

---

## 目录

1. [总体架构图](#1-总体架构图)
2. [技术选型表](#2-技术选型表)
3. [微服务拓扑](#3-微服务拓扑)
4. [数据流](#4-数据流)
5. [数据库设计](#5-数据库设计)
6. [通信协议](#6-通信协议)
7. [消息格式](#7-消息格式)
8. [安全管理](#8-安全管理)
9. [性能指标](#9-性能指标)
10. [扩展性](#10-扩展性)

---

## 1. 总体架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                             客户端 (Client)                                  │
│    Vue 3 + TypeScript + WebSocket + HTTP                                    │
│    [PC 浏览器 / 移动端 H5 / 未来原生]                                       │
└──────────────────────────┬──────────────────────────────────────────────────┘
                           │
                           │  wss://domain/ws   +   https://domain/api
                           │
┌──────────────────────────▼──────────────────────────────────────────────────┐
│                          Nginx (反向代理 / SSL 终结 / 静态资源)               │
│                          HTTP/2 + WebSocket Proxy                           │
└──────────────────────────┬──────────────────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────────────────┐
│                      Gateway Service (端口 8080/8081)                        │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐              │
│  │ WebSocket       │  │ HTTP Router     │  │ 连接管理器      │              │
│  │ 消息路由        │  │ RESTful API     │  │ 会话/心跳       │              │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘              │
│           │                    │                     │                       │
│  ┌────────▼────────────────────▼─────────────────────▼───────┐               │
│  │              消息分发 / 协议转换 / 限流                      │               │
│  └────────────────────────────┬───────────────────────────────┘               │
└───────────────────────────────┼───────────────────────────────────────────────┘
                                │
   ┌────────────────────────────┼───────────────────────────┐
   │          HTTP/gRPC 内部通信 │   K8s DNS 服务发现        │
   ▼                            ▼                           ▼
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ Auth     │  │ Player   │  │Cultivat. │  │ Combat   │  │ Social   │
│ :8082    │  │ :8083    │  │ :8084    │  │ :8085    │  │ :8086    │
│ 认证/登录│  │ 玩家数据  │  │修炼/突破 │  │ 战斗/PVP │  │聊天/好友 │
│ JWT签发  │  │ 背包/装备 │  │炼丹/功法 │  │ 伤害计算 │  │宗门/邮件 │
└──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘

┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ World    │  │ Trade    │  │ Ranking  │  │ (扩展)   │
│ :8087    │  │ :8088    │  │ :8089    │  │          │
│ 地图/奇遇│  │ 交易行/  │  │ 排行榜/  │  │ 预留     │
│ 采集/NPC │  │ 拍卖     │  │ 赛季    │  │          │
└──────────┘  └──────────┘  └──────────┘  └──────────┘

┌─────────────────────────────────────────────────────────────────┐
│                             数据层                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │ MySQL 8.0    │  │ Redis 7      │  │ MongoDB 7.0  │            │
│  │ 主存储       │  │ 缓存/会话    │  │ 社交/日志    │            │
│  │ 24 张业务表  │  │ 排行榜/限流  │  │ 聊天/回放    │            │
│  └──────────────┘  └──────────────┘  └──────────────┘            │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                       监控 / 运维                                 │
│   Prometheus + Grafana + Loki + Alertmanager                      │
│   容器指标 / 业务指标 / 日志聚合 / 3 级告警 (P1/P2/P3)            │
└──────────────────────────────────────────────────────────────────┘
```

### 架构分层说明

| 层次 | 组件 | 职责 | 关键特性 |
|------|------|------|---------|
| **接入层** | Nginx | SSL 终结、反向代理、静态资源、WS 升级 | HTTP/2, Let's Encrypt |
| **网关层** | Gateway | 连接管理、消息路由、协议转换、限流熔断 | 5 万连接/节点, P99 < 50ms |
| **业务层** | 9 个微服务 | 各自独立业务域，通过 HTTP 内外通信 | 无状态，HPA 自动伸缩 |
| **数据层** | MySQL + Redis + MongoDB | 持久化、缓存、文档存储 | 主从复制、哨兵模式 |
| **监控层** | Prometheus + Grafana + Loki | 指标采集、可视化、日志、告警 | P1 30s 响应 |

---

## 2. 技术选型表

### 2.1 后端技术栈

| 组件 | 技术 | 版本 | 选型理由 |
|------|------|------|---------|
| 开发语言 | Go | 1.22+ | 高并发 goroutine 模型，编译速度快，部署简单，社区生态成熟，单二进制部署 |
| Web 框架 | Gin | 1.9+ | 轻量高性能，中间件丰富（日志/恢复/跨域/限流），路由分组，JSON 绑定验证 |
| 数据库 ORM | GORM | 1.25+ | Go 最流行的 ORM，自动迁移，钩子函数，预加载，事务支持 |
| 配置管理 | 自研 config.Loader | - | 支持 JSON/YAML 文件 + 环境变量覆盖 + fsnotify 热重载 |
| 消息编解码 | Protobuf | proto3 | 高效二进制序列化，跨语言兼容，Schema 强约束 |
| 服务发现 | K8s DNS | - | 原生 Service 发现，无需额外注册中心 |
| 事件总线 | 自研 EventBus | - | 同步/异步发布，通配符订阅，中间件链，超时控制 |
| 插件系统 | 自研 PluginManager | - | 生命周期管理，消息路由注册，事件订阅，模块解耦 |
| 编解码器 | 自研 codec | - | Protobuf 主编码 + JSON 降级，Snappy/Gzip 压缩 |

### 2.2 前端技术栈

| 组件 | 技术 | 版本 | 选型理由 |
|------|------|------|---------|
| 框架 | Vue 3 | 3.4+ | 组合式 API，响应式系统，TypeScript 一等支持 |
| 构建工具 | Vite | 5.1+ | 极速 HMR (<100ms)，原生 ESM，Tree-shaking |
| 状态管理 | Pinia | 2.1+ | Vue 官方推荐，TypeScript 友好，模块化 Store |
| 路由 | Vue Router | 4.3+ | 嵌套路由，导航守卫，懒加载 chunk |
| 样式 | SCSS | 1.71+ | 变量/混合/嵌套，支持 PC/移动端主题切换 |
| 网络层 | 自研 WebSocketClient | - | 自动重连(指数退避1-30s)，心跳保活，消息队列 |
| HTTP 层 | 自研 apiFetch | - | 自动携带 Token，401 跳转，统一错误 Toast |
| 类型检查 | vue-tsc | 2.0+ | 全量类型安全编译检查 |

### 2.3 数据层技术栈

| 组件 | 技术 | 版本 | 核心用途 |
|------|------|------|---------|
| 关系数据库 | MySQL | 8.0 | 玩家数据、物品、境界、交易等需要 ACID 事务保证的数据 |
| 缓存数据库 | Redis | 7-alpine | Session 会话、排行榜(SortedSet)、在线状态、限流 |
| 文档数据库 | MongoDB | 7.0 | 聊天记录、战斗回放、事件日志等灵活 Schema 数据 |
| 消息队列 | Redis Streams (内置) | - | 轻量消息队列，无额外组件依赖 |

### 2.4 部署与监控

| 组件 | 技术 | 版本 | 用途 |
|------|------|------|------|
| 容器运行 | Docker | 24+ | 开发环境容器化，Compose 编排 |
| 容器编排 | Kubernetes | 1.28+ | 生产环境编排，HPA 自动伸缩 |
| 指标采集 | Prometheus | latest | 15s 采集周期，业务 + 系统指标 |
| 可视化 | Grafana | latest | 仪表盘，告警面板，趋势分析 |
| 日志聚合 | Loki + Promtail | latest | 集中式日志，按 label 检索 |
| 告警引擎 | Alertmanager | latest | P1(30s) / P2(5m) / P3(5m) 三级告警 |

---

## 3. 微服务拓扑

系统共包含 **9 个微服务**，每个服务独立部署、独立数据源（或 schema）、独立伸缩、独立故障域。

### 3.1 服务总览

| 序号 | 服务名 | 端口 | 技术栈 | 职责简述 | 资源限制 |
|------|--------|------|--------|---------|---------|
| 1 | **gateway** | 8080(WS) / 8081(HTTP) | Go + Gin | 网关，连接管理，消息路由，限流 | 0.5C / 512M |
| 2 | **auth** | 8082 | Go + Gin + JWT | 认证鉴权，JWT 签发，密码校验 | 0.3C / 256M |
| 3 | **player** | 8083 | Go + Gin + GORM | 玩家数据，背包，装备 | 0.3C / 256M |
| 4 | **cultivation** | 8084 | Go + Gin | 修炼，突破，炼丹，功法 | 0.4C / 512M |
| 5 | **combat** | 8085 | Go + Gin | PVE/PVP 战斗，伤害计算，匹配 | 0.4C / 512M |
| 6 | **social** | 8086 | Go + Gin + Mongo | 聊天，好友，宗门，邮件 | 0.3C / 256M |
| 7 | **world** | 8087 | Go + Gin | 世界地图，探索，采集，奇遇 | 0.3C / 256M |
| 8 | **trade** | 8088 | Go + Gin | 交易行，拍卖，订单 | 0.3C / 256M |
| 9 | **ranking** | 8089 | Go + Gin | 排行榜，赛季，ELO | 0.2C / 256M |

### 3.2 服务间依赖关系

```
Gateway
  ├──→ Auth        (登录/注册/Token 验证)
  ├──→ Player      (玩家信息查询/背包操作/装备管理)
  ├──→ Cultivation (修炼/突破/炼丹/功法)
  ├──→ Combat      (开始战斗/提交行动/查询战报)
  ├──→ Social      (聊天发送/好友操作/宗门管理/邮件)
  ├──→ World       (探索/移动/采集/奇遇)
  ├──→ Trade       (挂单/购买/拍卖)
  └──→ Ranking     (查询排行/提交积分)

Cultivation ──→ Player    (读取玩家属性/更新境界和修为)
Combat       ──→ Player    (读取战斗属性/发放战斗奖励)
Combat       ──→ Cultivation (突破触发战力重新计算)
Social       ──→ Player    (读取玩家昵称/境界显示)
World        ──→ Player    (读取任务进度/更新采集数据)
Trade        ──→ Player    (读取背包/更新灵石)
Ranking      ──→ Player    (读取排行数据)
Ranking      ──→ Combat    (读取 PVP 积分)
```

### 3.3 各服务详细设计

#### 3.3.1 Gateway (网关服务)

**核心职责：**
- WebSocket 连接生命周期管理，单节点支持 5 万并发长连接
- 基于 msg_id 的消息路由分发到对应后端服务
- 客户端 Protobuf 二进制消息与内部 HTTP/JSON 协议的双向转换
- 令牌桶限流（1000 req/s per connection）与熔断保护
- 30 秒心跳探测，连接超时断开回收
- 断线重连支持（30 秒窗口），Session 状态恢复

**关键组件：**
- Hub：全局连接池，维护所有活跃 WebSocket 连接
- Router：消息 ID 到后端服务的映射路由表
- Connection：每个连接独立的读写 goroutine 模型

**技术要点：**
- 使用 goroutine-per-connection 模型，读写分离
- 读写缓冲区 4096 字节，避免频繁系统调用
- 限流使用令牌桶算法，突发容量 2000

#### 3.3.2 Auth (认证服务)

**核心职责：**
- 玩家注册：参数校验、名称去重、灵根分配、bcrypt 密码哈希
- 玩家登录：密码验证、JWT 签发（access_token 15min + refresh_token 7d）
- Token 刷新：验证 refresh_token，轮换签发新 token pair
- Token 失效：服务端 Redis 存储黑名单，支持主动踢下线

**安全措施：**
- bcrypt cost=10，单次验证约 100ms（抗暴力破解）
- JWT 签名使用 HS256，密钥通过环境变量注入
- refresh_token 单次使用即作废（轮换机制）
- 同一账号仅允许一个有效 session（新登录踢旧 session）

#### 3.3.3 Player (玩家服务)

**核心职责：**
- 玩家基本信息 CRUD（昵称、境界、经验、属性、灵根）
- 背包系统：物品增删改查、堆叠、分类过滤、分页查询
- 装备系统：穿戴/卸下/替换、强化等级、宝石镶嵌
- 功法管理：已学功法列表、装备主修/辅修、功法升级
- 属性计算引擎：聚合基础属性 + 装备加成 + 功法加成 + Buff 效果

**数据模型要点：**
- `players` 表使用 JSON 字段存储灵根和辅修功法 ID 列表，灵活扩展
- `inventory` 表使用 extra_data JSON 字段存储耐久度、随机词条等扩展属性
- `equipment` 表使用 base_attr JSON + enhance_level + gems JSON 三字段分离

#### 3.3.4 Cultivation (修炼服务)

**核心职责：**
- 修炼打坐：选择功法投入修炼，根据时长和加成计算修为收益
- 境界突破：大境界突破的成功率计算（受灵根/丹药/法宝/气运影响）
- 天劫系统：化神及以上境界突破需渡天劫，多轮试炼属性判定
- 炼丹系统：丹方配方组合、药材消耗、成丹品质随机、丹药效果
- 功法系统：功法学习（消耗秘籍）、升级（积累熟练度）、主辅切换

**境界体系 (9 大境界, 37 级)：**

| realm_group | 境界名称 | 层数 | 起始修为 | 满级修为 |
|-------------|---------|------|---------|---------|
| 1 | 练气期 | 9 层 | 100 | 5,000 |
| 2 | 筑基期 | 4 境 | 10,000 | 40,000 |
| 3 | 金丹期 | 4 境 | 60,000 | 150,000 |
| 4 | 元婴期 | 4 境 | 200,000 | 500,000 |
| 5 | 化神期 | 4 境 | 650,000 | 1,400,000 |
| 6 | 合体期 | 4 境 | 1,800,000 | 3,600,000 |
| 7 | 大乘期 | 4 境 | 4,500,000 | 8,800,000 |
| 8 | 渡劫期 | 4 境 | 11,000,000 | 18,000,000 |
| 9 | 飞升 | - | 已满 | - |

**突破成功率公式：**
```
final_rate = base_rate + spirit_root_bonus + item_bonus - 
             (current_realm_level * 0.02) + luck_factor
final_rate = min(final_rate, max_rate)
```

#### 3.3.5 Combat (战斗服务)

**核心职责：**
- PVE 战斗引擎：回合制自动战斗，技能选择，战报生成
- PVP 竞技场：在线匹配、异步战报、ELO 积分排名
- 伤害计算：五行克制（金克木 1.3x、木克土 1.3x...）、暴击/闪避/格挡判定
- Buff 系统：增益/减益效果，回合持续，可叠加

**战斗流程：**
```
1. 创建 Fighter 实例 (加载玩家/怪物属性)
2. 速度排序决定行动顺序
3. 回合循环:
   a. 技能选择 (AI / 玩家输入)
   b. 目标选择
   c. 伤害计算:
      base_damage = skill.base_damage + coeff * attacker_attr
      elemental_bonus = 1.0 (无克制) / 1.3 (克制) / 0.7 (被克)
      crit_damage = base_damage * (1.5 + crit_bonus)  [暴击判定]
      final_damage = base_damage * elemental_bonus - target_defense/10
   d. Buff 更新 (添加/移除)
   e. 状态检测 (HP <= 0 则战斗结束)
4. 奖励结算 (经验/灵石/物品掉落)
```

#### 3.3.6 Social (社交服务)

**核心职责：**
- 聊天系统：4 频道（世界/宗门/私聊/系统），敏感词过滤，聊天记录 MongoDB 持久化
- 好友系统：申请/同意/拒绝/删除，在线状态，亲密度
- 宗门系统：创建/加入/退出/踢出，宗主/长老/精英/成员四级职位，贡献度体系
- 邮件系统：系统邮件/玩家邮件，附件发放，已读/领取状态

**数据分层：**
- MySQL：好友关系、宗门信息、邮件正文等结构化数据
- MongoDB：聊天历史、好友操作日志等半结构化数据
- Redis：在线状态、最近聊天缓存（List, capped 100）

#### 3.3.7 World (世界服务)

**核心职责：**
- 世界地图：多区域连接，方向导航，场景描述渲染
- 探索系统：区域探索随机触发奇遇事件，多分支选择，结果判定
- 采集系统：资源点定时刷新，采集限制（每日次数），采集耗时
- NPC 交互：对话分支，任务接取，商店交易

#### 3.3.8 Trade (交易服务)

**核心职责：**
- 交易行：物品上架/下架/搜索/购买，按物品类型和价格排序
- 拍卖系统：竞价拍卖，保留价机制，结束前 5 分钟自动延长
- 灵石资金池：乐观锁（version 字段）防止超卖，事务保证资金安全
- 税率系统：交易金额 5% 系统扣税

#### 3.3.9 Ranking (排行榜服务)

**核心职责：**
- PVP 排行榜：ELO 积分降序，赛季制（30 天）
- 境界排行榜：按 realm_group + realm_level + exp 综合排序
- 战力排行榜：综合战斗力降序
- 财富排行榜：灵石持有量降序

**技术实现：**
- Redis Sorted Set 作为热数据存储，O(log N) 排行查询
- 每 5 分钟定时持久化到 MySQL
- 排行榜缓存 60 秒 TTL，减少查询压力

---

## 4. 数据流

### 4.1 用户注册流程

```
Client                    Gateway                  Auth                  Player                MySQL
  │                         │                       │                     │                     │
  │-- POST /auth/register -->│                       │                     │                     │
  │   {username,password,   │                       │                     │                     │
  │    nickname,spirit_root} │                       │                     │                     │
  │                         │-- HTTP POST ---------->│                     │                     │
  │                         │   /auth/register       │                     │                     │
  │                         │                       │-- 参数校验 ----------│                     │
  │                         │                       │-- 用户名唯一检查 ----│                     │
  │                         │                       │-- bcrypt hash -------│                     │
  │                         │                       │-- INSERT players ----│--------------------->│
  │                         │                       │<-- id ---------------│                     │
  │                         │                       │-- INSERT attributes -│--------------------->│
  │                         │                       │-- INSERT inventory --│--------------------->│
  │                         │                       │-- 生成 JWT Token ----│                     │
  │                         │                       │-- Redis SET session -│-----> Redis          │
  │                         │                       │                     │                     │
  │                         │<-- 200 OK ------------│                     │                     │
  │<-- 200 OK --------------│                       │                     │                     │
  │   {player_id: 42,      │                       │                     │                     │
  │    token: "eyJ...",    │                       │                     │                     │
  │    refresh_token: ".."} │                       │                     │                     │
```

### 4.2 登录 -> 修炼 -> 突破 完整流程

```
Client                  Gateway               Auth              Cultivation           Player/DB
  │                       │                     │                   │                     │
  │== 1. 登录 ==          │                     │                   │                     │
  │-- POST /auth/login -->│                     │                   │                     │
  │                       │-- /auth/verify ---->│                   │                     │
  │                       │                     │-- 密码验证 --------│                     │
  │                       │                     │-- 生成 token ------│                     │
  │<-- token + player_id -│<-- 200 OK ---------│                   │                     │
  │                       │                     │                   │                     │
  │== 2. WebSocket 连接 ==│                     │                   │                     │
  │-- WS /ws?token=xxx -->│                     │                   │                     │
  │                       │-- 验证 token -------│                   │                     │
  │<-- WS 101 connected --│                     │                   │                     │
  │                       │                     │                   │                     │
  │== 3. 修炼 ==           │                     │                   │                     │
  │-- MSG_CULTIVATE ------>│                     │                   │                     │
  │   {technique_id: 2,   │                     │                   │                     │
  │    duration: 20}      │                     │                   │                     │
  │                       │-- HTTP POST ------->│                   │                     │
  │                       │  /cultivate         │                   │                     │
  │                       │                     │-- 校验玩家状态 ----│--------------------->│
  │                       │                     │-- 计算收益:       │                     │
  │                       │                     │   exp = base      │                     │
  │                       │                     │        * duration  │                     │
  │                       │                     │        * speed_buff│                     │
  │                       │                     │        * pill_bonus│                     │
  │                       │                     │-- UPDATE exp ------│--------------------->│
  │                       │                     │-- 检查是否可突破   │                     │
  │                       │                     │                   │                     │
  │<-- MSG_CULTIVATE_RES -│<-- 200 OK ---------│                   │                     │
  │   {exp_gained: 480,  │                     │                   │                     │
  │    exp_per_hour: 1440,│                     │                   │                     │
  │    current_realm:"练气 │                     │                   │                     │
  │     九层",            │                     │                   │                     │
  │    next_realm_exp:    │                     │                   │                     │
  │      5000,            │                     │                   │                     │
  │    triggered_breakth: │                     │                   │                     │
  │      true}            │                     │                   │                     │
  │                       │                     │                   │                     │
  │== 4. 突破 ==           │                     │                   │                     │
  │-- MSG_BREAKTHROUGH --->│                     │                   │                     │
  │   {assist_elixirs:[1],│                     │                   │                     │
  │    protection_talisman│                     │                   │                     │
  │    :0}                │                     │                   │                     │
  │                       │-- HTTP POST ------->│                   │                     │
  │                       │   /breakthrough     │                   │                     │
  │                       │                     │-- 计算基础成功率   │                     │
  │                       │                     │   base_rate=0.3   │                     │
  │                       │                     │-- 灵根加权: +0.15 │                     │
  │                       │                     │-- 丹药加成: +0.10 │                     │
  │                       │                     │-- 最终: 0.55      │                     │
  │                       │                     │-- 随机数 > 0.55   │                     │
  │                       │                     │   => 成功!        │                     │
  │                       │                     │-- UPDATE realm    │--------------------->│
  │                       │                     │-- UPDATE attr     │--------------------->│
  │                       │                     │-- 推送排行榜更新  │                     │
  │                       │                     │                   │                     │
  │<-- MSG_BREAKTHROUGH   │<-- 200 OK ---------│                   │                     │
  │     _RES              │                     │                   │                     │
  │   {success: true,     │                     │                   │                     │
  │    new_realm:"筑基初  │                     │                   │                     │
  │     期",              │                     │                   │                     │
  │    combat_power_inc:  │                     │                   │                     │
  │      1500,            │                     │                   │                     │
  │    description:       │                     │                   │                     │
  │      "筑基成功!..."}  │                     │                   │                     │
```

### 4.3 PVE 战斗流程

```
Client                Gateway                  Combat                   Player/DB
  │                     │                        │                        │
  │-- MSG_COMBAT_START ->│                        │                        │
  │   {instance_id: 3,  │                        │                        │
  │    monster_id: 5}   │-- HTTP POST ---------->│                        │
  │                     │   /combat/start         │                        │
  │                     │                        │-- 加载玩家属性 ---------│
  │                     │                        │-- 加载怪物配置 ---------│
  │                     │                        │-- 创建 Fighter 实例 ----│
  │                     │                        │   (含装备/功法/Buff)    │
  │                     │                        │-- 计算先手顺序(速度) ---│
  │                     │                        │                        │
  │<-- 战斗开始 -------│<-- CombatStartResp ----│                        │
  │                     │                        │                        │
  │== 回合 1 ==         │                        │                        │
  │-- MSG_COMBAT_ACTION>│                        │                        │
  │   {skill_id: 6}     │-- HTTP POST ---------->│                        │
  │                     │   /combat/action        │                        │
  │                     │                        │-- 伤害计算:            │
  │                     │                        │   dmg = 80 + 2.2*120   │
  │                     │                        │        = 344           │
  │                     │                        │-- 暴击判定: 12% > rand │
  │                     │                        │   => 暴击! *1.6=550    │
  │                     │                        │-- 闪避判定: 8% < rand  │
  │                     │                        │   => 命中              │
  │                     │                        │-- 五行: 火克木 *1.3   │
  │                     │                        │   => 715 伤害          │
  │                     │                        │-- 怪物 HP: 2000-715    │
  │                     │                        │   = 1285               │
  │                     │                        │-- 怪物反击...          │
  │                     │                        │                        │
  │<-- 行动结果 -------│<-- CombatActionResp ---│                        │
  │                     │                        │                        │
  │... (重复 N 回合)    │                        │                        │
  │                     │                        │                        │
  │== 战斗结束 ==        │                        │                        │
  │                     │                        │-- 怪物 HP <= 0         │
  │                     │                        │   => 玩家胜利          │
  │                     │                        │-- 经验: 500 * 倍率     │
  │                     │                        │-- 灵石: 200            │
  │                     │                        │-- 掉宝: rand 判定      │
  │                     │                        │   {item_id: 20, qty:1} │
  │                     │                        │-- 更新玩家数据 ---------│
  │                     │                        │                        │
  │<-- MSG_COMBAT_RESULT│<-- CombatResult -------│                        │
  │   {victory: true,   │                        │                        │
  │    exp_reward: 500, │                        │                        │
  │    money: 200,      │                        │                        │
  │    drops: [{"id":20,│                        │                        │
  │      "name":"妖兽    │                        │                        │
  │       内丹","qty":1}]│                        │                        │
  │    rounds: [...]}   │                        │                        │
```

### 4.4 玩家成长闭环

```
                        ┌─────────────────────┐
                        │   注册 / 创建角色    │
                        │   选择灵根 / 昵称   │
                        └──────────┬──────────┘
                                   │
                                   ▼
                        ┌─────────────────────┐
                        │    修炼打坐赚经验   │
                        │    服用丹药加速     │
                        │    功法熟练度提升   │
                        └──────────┬──────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    │                             │
                    ▼                             ▼
            ┌────────────────┐         ┌──────────────────┐
            │  境界突破      │         │  功法/技能升级    │
            │  消耗突破丹药   │         │  消耗熟练度      │
            │  灵根/气运加成  │         │  消耗灵石        │
            │  可能触发天劫   │         └────────┬─────────┘
            └───────┬────────┘                   │
                    │                            │
                    ▼                            ▼
            ┌────────────────┐         ┌──────────────────┐
            │  属性大幅提升  │         │  实力稳步提升    │
            │  解锁新区域    │         │  新技能/新效果   │
            │  战力飙升      │         └────────┬─────────┘
            └───────┬────────┘                   │
                    │                            │
                    └──────────┬─────────────────┘
                               │
                               ▼
                      ┌─────────────────┐
                      │  探索世界 / 打怪  │
                      │  采集资源        │
                      │  触发奇遇事件     │
                      └────────┬────────┘
                               │
                               ▼
                      ┌─────────────────┐
                      │  获取材料/丹药   │
                      │  积累灵石/装备   │
                      └────────┬────────┘
                               │
                    ┌──────────┴──────────┐
                    │                     │
                    ▼                     ▼
            ┌──────────────┐   ┌────────────────┐
            │ 炼丹 / 炼器  │   │ 交易行买卖     │
            │ 消耗材料     │   │ 挂单/竞拍      │
            │ 提升实力     │   │ 赚取灵石      │
            └──────┬───────┘   └───────┬────────┘
                   │                   │
                   └───────┬───────────┘
                           │
                           ▼
                     ┌──────────────┐
                     │  战力提升    │
                     │  挑战更强怪物 │
                     │  PVP 竞技    │
                     └──────┬───────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  重返修炼    │
                     │  突破更高境界  │
                     └──────┬───────┘
                            │
                    (回到循环起点，螺旋上升)
```

---

## 5. 数据库设计

### 5.1 实体关系图 (文字描述)

```
┌──────────────────┐       ┌─────────────────────┐       ┌──────────────────┐
│    realm_config   │       │      players        │       │player_attributes  │
│  (境界配置, 37行) │       │  (玩家核心, 24字段)   │ 1:1  │ (扩展属性, 13字段)│
│  PK: id           │       │  PK: id              │<─────│ PK: player_id    │
│  UK: realm+level  │       │  UK: uid             │       │ FK: -> players   │
└──────────────────┘       │  FK: realm_id         │       └──────────────────┘
         │                 │  (境界/灵根JSON/货币等)│
         │                 └─────────┬─────────────┘
         │                           │
         │        ┌──────────────────┼──────────────────┐
         │        │        1:N       │       1:N        │        1:N
         ▼        ▼                  ▼                  ▼          ▼
  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
  │ inventory   │  │ equipment   │  │ techniques  │  │   skills    │  │  friends    │
  │ (背包)      │  │ (装备)      │  │ (已学功法)   │  │ (已学技能)   │  │ (好友关系)   │
  │ FK: player  │  │ FK: player  │  │ FK: player  │  │ FK: player  │  │ FK: player  │
  │ JSON扩展    │  │ JSON属性    │  │ UK: p+tech  │  │ UK: p+skill │  │ PK: p+friend│
  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘

  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
  │   sects     │  │ sect_members│  │    mail     │  │trade_listing│  │trade_trans. │
  │ (宗门)      │  │ (宗门成员)   │  │ (邮件)      │  │ (交易上架)   │  │ (交易记录)   │
  │ PK: id      │  │ FK: sect    │  │ FK: receiver│  │ FK: seller  │  │ FK: listing │
  │ FK: leader  │  │ FK: player  │  │ 附件系统    │  │ 状态/过期   │  │ buyer/seller│
  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘

  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
  │trade_auct.  │  │pvp_rankings │  │player_buffs │  │world_events │  │daily_resets │
  │ (拍卖)      │  │ (PVP排行)   │  │ (Buff)      │  │ (世界事件)   │  │ (日常重置)   │
  │ PK: id      │  │ PK: p+season│  │ FK: player  │  │ JSON参数    │  │ UK: p+date  │
  │ 竞价/状态   │  │ ELO/赛季    │  │ 时间范围    │  │ 定时触发    │  │ 限次计数    │
  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘

  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
  │player_luck  │  │luck_events  │  │item_templ.  │  │technique_t. │  │skill_templ. │
  │ (玩家气运)   │  │ (气运事件)   │  │ (物品模板)   │  │ (功法模板)   │  │ (技能模板)   │
  │ PK: player  │  │ FK: player  │  │ JSON效果    │  │ 五行属性    │  │ 倍率/效果   │
  │ 可负值      │  │ 全留痕      │  │ 品质/绑定   │  │ 品级/加成   │  │ JSON扩展    │
  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘

  ┌─────────────┐  ┌─────────────┐
  │system_config│  │ admin_users │
  │ (系统配置)   │  │ (管理员)    │
  │ UK: key     │  │ bcrypt密码  │
  │ 热加载      │  │ RBAC权限   │
  └─────────────┘  └─────────────┘
```

### 5.2 表结构说明

系统共使用 **24 张 MySQL 表**，分为六大类。

#### 5.2.1 核心玩家表 (6 张)

| 表名 | 说明 | 关键字段 | 索引 |
|------|------|---------|------|
| **players** | 玩家核心，24 字段 | id, uid(UK), nickname, realm_id, realm_level, exp, spirit_root(JSON), cultivation_technique_id, auxiliary_technique_ids(JSON), money, bind_money, immortal_jade, vip_level, total_play_time | UK(uid), idx(nickname), idx(realm,level) |
| **player_attributes** | 1:1 扩展属性 | player_id(PK,FK), max_hp, max_mp, crit_rate(5.00), crit_dmg(150.00), dodge_rate(5.00), hit_rate(95.00), cultivation_speed(100.00), fortune, comprehension, charm | PK(player_id) |
| **inventory** | 背包物品，JSON 扩展 | id, player_id(FK), item_id, quantity, slot, extra_data(JSON), is_equipped | idx(player), idx(player_slot), idx(item) |
| **equipment** | 已装备 | id, player_id(FK), slot_type(1-10), item_id, base_attr(JSON), enhance_level, gems(JSON) | UK(player_id, slot_type) |
| **techniques** | 已学功法 | id, player_id(FK), technique_id, level, exp, is_equipped(0=未/1=主/2=辅) | UK(player_id, technique_id) |
| **skills** | 已学技能 | id, player_id(FK), skill_id, level, is_equipped | UK(player_id, skill_id) |

#### 5.2.2 社交系统表 (4 张)

| 表名 | 说明 | 关键字段 | 索引 |
|------|------|---------|------|
| **friends** | 好友关系 | player_id, friend_id, status(0申请/1好友/2黑名单) | PK(p,f), idx(status) |
| **sects** | 宗门 | id, name, leader_id(FK), level, member_count, max_members(默认50), notice | idx(name), idx(leader) |
| **sect_members** | 宗门成员 | id, sect_id(FK), player_id(FK), position(1宗主/2长老/3精英/4成员), contribution | UK(sect, player) |
| **mail** | 邮件 | id, sender_id(0=系统), receiver_id(FK), title, content, has_attachment, item_id, item_quantity, is_read, is_claimed | idx(receiver, read), idx(receiver, claimed) |

#### 5.2.3 交易系统表 (4 张)

| 表名 | 说明 | 关键字段 | 索引 |
|------|------|---------|------|
| **trade_listings** | 市场挂单 | id, seller_id(FK), item_id, quantity, unit_price, status(active/sold/cancelled/expired), expires_at | idx(seller), idx(item), idx(status) |
| **trade_transactions** | 交易记录 | id, listing_id, buyer_id, seller_id, item_id, quantity, total_price | idx(buyer), idx(seller), idx(created_at) |
| **trade_auctions** | 拍卖 | id, item_id, seller_id, current_bid, bidder_id, reserve_price, end_time, status | idx(status, end_time) |
| **trade_player_gold** | 灵石 (乐观锁) | player_id(PK), gold, version(乐观锁) | PK |

#### 5.2.4 游戏循环表 (3 张)

| 表名 | 说明 | 关键字段 | 索引 |
|------|------|---------|------|
| **player_buffs** | 玩家 Buff | id, player_id(FK), buff_type, effect_value(0.0000), buff_source, start_time, end_time(NULL=永久) | idx(player, type, end_time) |
| **world_events** | 世界事件 | id, event_type, region_id, title, params(JSON), start_time, end_time, status(scheduled/active/finished/cancelled) | idx(type, status), idx(time_range) |
| **daily_resets** | 日常重置 | id, player_id(FK), reset_date, cultivation_count, quest_count, gather_count, pvp_win/lose, extra_data(JSON) | UK(player_id, reset_date) |

#### 5.2.5 气运系统表 (2 张)

| 表名 | 说明 | 关键字段 | 索引 |
|------|------|---------|------|
| **player_luck** | 玩家气运 | player_id(PK), luck_value(可负), total_earned, total_spent | idx(luck_value) |
| **luck_events** | 气运事件日志 | id, player_id(FK), event_type, amount, description | idx(player_time), idx(player_type) |

#### 5.2.6 配置/模板表 (6 张)

| 表名 | 说明 | 关键字段 | 索引 |
|------|------|---------|------|
| **realm_config** | 境界配置 (37行) | id, realm_name, realm_group(1-9), level, upgrade_exp, base_hp/mp/attack/defense/speed, breakthrough_item_id, description | UK(realm_group, level) |
| **item_templates** | 物品模板 | id, name, item_type(1-6), quality(1-6), use_level, sell_price, bind_type, max_stack, cooldown, extra_attributes(JSON) | idx(type), idx(quality) |
| **technique_templates** | 功法模板 | id, name, grade(1-5), element_type, max_level, cultivation_bonus(%), hp/mp/attack/defense/speed_bonus | idx(grade), idx(element) |
| **skill_templates** | 技能模板 | id, name, skill_type(1-5), element_type, mp_cost, cooldown, target_type, damage_multiplier, extra_effects(JSON) | idx(type), idx(element) |
| **system_config** | 系统配置 | id, config_key(UK), config_value, description | UK(config_key) |
| **admin_users** | 管理员 | id, username(UK), password_hash(bcrypt), role(super_admin/admin/operator/auditor), permissions(JSON) | UK(username) |

### 5.3 Redis 数据结构

| 用途 | 数据结构 | Key 模式 | TTL | 说明 |
|------|---------|----------|-----|------|
| 会话 Session | String | `session:{token}` | 7 天 | 存储 player_id, JSON |
| 在线玩家 | Set | `online:players` | - | 当前在线玩家 ID 集合 |
| 在线心跳 | String | `online:status:{player_id}` | 60s | 心跳刷新，过期自动离线 |
| 排行榜 | SortedSet | `ranking:{season}:{type}` | 永久 | score=rating/exp/combat_power |
| 限流计数器 | String | `ratelimit:{ip}:{endpoint}:{ts}` | 1s | 每秒请求计数 |
| 最近聊天 | List | `chat:recent:{channel}` | - | capped 100 条 |
| 修炼 Session | Hash | `cultivation:session:{pid}` | 30天 | 离线挂机缓存收益 |
| 消息队列 | Stream | `queue:combat` / `queue:social` | - | Redis Streams 消息 |

### 5.4 MongoDB 集合

| 集合 | 说明 | 文档示例 |
|------|------|---------|
| `chat_messages` | 聊天记录 | {channel, sender_id, sender_name, content, timestamp, extra_data} |
| `battle_replays` | 战斗回放 | {combat_id, player_id, monster_id, rounds[{}], exp_reward, drops[], timestamp} |
| `event_logs` | 事件日志 | {player_id, event_type, data(JSON), timestamp} |
| `trade_history` | 历史交易归档 | {listing_id, buyer_id, seller_id, item_id, price, timestamp} |

---

## 6. 通信协议

系统采用 **双协议通信架构**，根据场景选择最合适的通信方式。

### 6.1 协议选择表

| 场景 | 协议 | 理由 |
|------|------|------|
| 游戏实时消息（修炼/战斗/聊天） | WebSocket + Protobuf | 全双工低延迟，二进制高效 |
| 登录/注册/Token 刷新 | HTTP + JSON | 简单请求/响应，无需长连接 |
| 玩家信息/背包查询 | HTTP + JSON | 非实时查询，RESTful 语义清晰 |
| 管理后台操作 | HTTP + JSON | 低频管理操作 |
| 内部服务间调用 | HTTP (gRPC 后续) | 内网通信，JSON 调试方便 |

### 6.2 WebSocket 协议规范

| 项目 | 规格 |
|------|------|
| 协议 | wss:// (生产) / ws:// (开发) |
| 端点 | /ws?token={access_token} |
| 消息格式 | 二进制帧 (Protobuf 编码 GameMessage) |
| 心跳间隔 | 30s (客户端 ping -> 服务端 pong) |
| 重连策略 | 指数退避: 1s -> 2s -> 4s -> 8s -> 16s -> 30s (max 10 次) |
| 请求超时 | 30s (无响应丢弃) |
| 协议版本 | 1.0.0 (连接时 version 字段协商) |
| 断线缓存 | 断开期间消息排队，重连后 flush |

### 6.3 HTTP RESTful API 规范

| 项目 | 规格 |
|------|------|
| 协议 | https:// (生产) / http:// (开发) |
| 基础路径 | /api/v1 |
| 内容类型 | application/json; charset=utf-8 |
| 认证 | Authorization: Bearer {jwt_token} |
| 分页 | Query: ?page=1&page_size=20 (默认 20, 最大 100) |
| 错误响应 | { "code": 4001, "message": "参数错误", "details": [...] } |
| 成功响应 | 直接返回业务数据 (data 字段)，或 status=0 格式 |

### 6.4 HTTP 状态码约定

| 状态码 | 含义 | 处理方式 |
|--------|------|---------|
| 200 | 成功 | 解析响应体 |
| 400 | 参数错误 | 检查请求参数 |
| 401 | 未认证/Token 过期 | 尝试 refresh_token，失败跳转登录 |
| 403 | 无权限 | 提示权限不足 |
| 404 | 资源不存在 | 检查请求路径/ID |
| 429 | 请求频率超限 | 等待后重试 |
| 500 | 服务器错误 | 记录日志，联系运维 |

### 6.5 内部服务通信

| 项目 | 规格 |
|------|------|
| 协议 | HTTP/1.1 (内网) |
| 序列化 | JSON (Protobuf 通信路线) |
| 服务发现 | K8s DNS: service.namespace.svc.cluster.local |
| 超时 | 5s 默认超时，3 次重试 |
| 断路器 | 连续 5 次失败熔断 30s |
| 健康检查 | GET /health, 返回 {"status":"ok","time":1234567890} |

---

## 7. 消息格式

### 7.1 GameMessage 信封

所有 WebSocket 消息包裹在统一的 `GameMessage` 信封（定义在 `common.proto`）中：

```protobuf
message GameMessage {
  uint32 msg_id = 1;      // 消息类型ID (1-999 系统, 1000+ 业务)
  uint64 seq = 2;         // 序列号 (请求-响应配对)
  int64 timestamp = 3;    // 客户端时间戳 (毫秒)
  bytes payload = 4;      // 业务数据 (Protobuf)
  uint32 status = 5;      // 状态码 (0=成功)
  string error_msg = 6;   // 错误信息
  string version = 7;     // 协议版本号
}
```

### 7.2 消息 ID 完整枚举

| 消息 ID | 名称 | 方向 | Payload | 说明 |
|---------|------|------|---------|------|
| 1 | MSG_HEARTBEAT | C->S | HeartbeatReq | 心跳请求 |
| 2 | MSG_HEARTBEAT_ACK | S->C | HeartbeatResp | 心跳响应 |
| 3 | MSG_RECONNECT | C->S | ReconnectReq | 断线重连 |
| 100 | MSG_AUTH_LOGIN | C->S | LoginReq | 登录 |
| 101 | MSG_AUTH_LOGIN_RES | S->C | LoginResp | 登录响应 |
| 102 | MSG_AUTH_REGISTER | C->S | RegisterReq | 注册 |
| 103 | MSG_AUTH_REGISTER_RES | S->C | RegisterResp | 注册响应 |
| 104 | MSG_AUTH_TOKEN_REFRESH | C->S | TokenRefreshReq | Token 刷新 |
| 200 | MSG_PLAYER_INFO | C->S | PlayerInfoReq | 玩家信息 |
| 201 | MSG_PLAYER_INFO_RES | S->C | PlayerInfoResp | 玩家信息响应 |
| 202 | MSG_PLAYER_UPDATE | S->C | PlayerInfo | 属性更新推送 |
| 300 | MSG_CULTIVATE | C->S | CultivateReq | 修炼 |
| 301 | MSG_CULTIVATE_RES | S->C | CultivateResp | 修炼响应 |
| 302 | MSG_BREAKTHROUGH | C->S | BreakthroughReq | 突破 |
| 303 | MSG_BREAKTHROUGH_RES | S->C | BreakthroughResp | 突破响应 |
| 400 | MSG_COMBAT_START | C->S | CombatStartReq | 开始战斗 |
| 401 | MSG_COMBAT_ACTION | C->S | CombatActionReq | 战斗行动 |
| 402 | MSG_COMBAT_RESULT | S->C | CombatActionResp | 战斗结果 |
| 500 | MSG_CHAT_SEND | C->S | ChatSendReq | 发送聊天 |
| 501 | MSG_CHAT_RECV | S->C | ChatMessage | 接收聊天 |
| 502 | MSG_FRIEND_ADD | C->S | FriendAddReq | 添加好友 |
| 600 | MSG_WORLD_EXPLORE | C->S | ExploreReq | 探索 |
| 601 | MSG_WORLD_MOVE | C->S | MoveReq | 移动 |
| 602 | MSG_WORLD_ENCOUNTER | S->C | EncounterInfo | 奇遇 |

### 7.3 状态码枚举

```protobuf
enum StatusCode {
  SUCCESS             = 0;    // 成功
  BAD_REQUEST         = 1001; // 参数错误
  UNAUTHORIZED        = 1002; // 未认证
  FORBIDDEN           = 1003; // 无权限
  NOT_FOUND           = 1004; // 资源不存在
  RATE_LIMITED        = 1005; // 频率超限
  INTERNAL_ERROR      = 2001; // 服务器内部错误
  SERVICE_UNAVAILABLE = 2002; // 服务不可用
}
```

### 7.4 编解码器架构

```
二进制数据流
    │
    ├── detectCompression()
    │   ├── Magic Bytes 0x1F8B → Gzip 解压
    │   ├── Snappy Decode 成功 → Snappy 解压
    │   └── 其他 → 原始数据
    │
    ├── ProtobufCodec (主编码器, 生产环境)
    │   ├── proto.Marshal / proto.Unmarshal
    │   └── Snappy 压缩 (数据 > 64 字节)
    │
    └── JSONCodec (降级编码器, 调试/开发)
        ├── json.Marshal / json.Unmarshal
        ├── pretty-print 支持 (调试模式)
        └── Gzip 压缩 (大数据)
```

---

## 8. 安全管理

系统采用 **4 层纵深防御体系**：网络安全、应用安全、数据安全、反作弊。

### 8.1 第一层：网络安全

| 措施 | 实现方式 | 防护目标 |
|------|---------|---------|
| SSL/TLS | Nginx 终结 HTTPS，证书自动续签 (Let's Encrypt) | 防中间人攻击、数据加密 |
| WebSocket WSS | wss:// 强制加密 | 防 WS 内容嗅探 |
| 端口收敛 | 仅暴露 80(HTTP) / 443(HTTPS) | 减少攻击面 |
| 请求限流 | Nginx limit_conn/limit_req 模块 | 防 DDoS |
| IP 白名单 | 管理后台仅内网/堡垒机访问 | 防未授权管理访问 |
| 防火墙 | iptables 规则：仅开放必要端口 | 网络边界防护 |

### 8.2 第二层：应用安全

| 措施 | 实现方式 | 防护目标 |
|------|---------|---------|
| JWT 认证 | HS256 签名，access_token 短时效 (15min) | 防身份伪造 |
| Token 轮换 | refresh_token 单次使用即作废 | 防 Token 窃取 |
| 权限控制 | 角色 RBAC (super_admin/admin/operator/auditor) | 防越权操作 |
| 请求限流 | 令牌桶算法，1000 req/s/ip，突发 2000 | 防暴力攻击 |
| 参数校验 | 输入长度/类型/范围校验 | 防注入攻击 |
| SQL 防护 | GORM 参数化查询，禁止原生 SQL | 防 SQL 注入 |
| CORS | 严格 origin 白名单 | 防跨域攻击 |
| XSS 防护 | HTML 转义 + Vue 自动转义 | 防 XSS 攻击 |

### 8.3 第三层：数据安全

| 措施 | 实现方式 | 防护目标 |
|------|---------|---------|
| 密码存储 | bcrypt (cost=10, ~100ms/次) | 防密码泄露 |
| 会话管理 | Redis + TTL，服务端主动失效 | 防会话劫持 |
| 日志脱敏 | 密码/Token 不记日志 | 防敏感信息泄露 |
| 事务安全 | MySQL 事务 + 乐观锁 (version) | 防并发超卖 |
| 备份加密 | AES-256 加密备份文件 | 防备份泄露 |
| 内网通信 | gRPC/HTTP 内网，不对外暴露 | 防服务间嗅探 |

### 8.4 第四层：反作弊

| 措施 | 实现方式 | 防护目标 |
|------|---------|---------|
| 服务端权威 | 所有数值计算在服务端执行 | 防客户端篡改 |
| 行为分析 | 异常频率检测（1s 内多次突破） | 防脚本/自动化 |
| 双重校验 | 客户端参数 + 服务端二次校验 | 防参数篡改 |
| 全量审计 | 所有灵石/物品变动记录流水 | 防经济系统破坏 |
| 操作限频 | 修炼/采集/战斗有频率上限 | 防资源刷取 |

---

## 9. 性能指标

### 9.1 目标性能指标

| 指标 | 目标值 | 测量方式 | 达标标准 |
|------|--------|---------|---------|
| 单节点并发连接 | 50,000 | Gateway 连接计数 | 稳定运行 30min |
| 消息延迟 P50 | < 10ms | 端到端 GameMessage | 客户端埋点 |
| 消息延迟 P99 | < 50ms | 端到端 GameMessage | 客户端埋点 |
| Gateway 吞吐 | > 10,000 req/s | Prometheus 计数 | 压测结果 |
| 数据库 P99 | < 20ms | MySQL 慢查询 | 慢查询日志 |
| Redis P99 | < 5ms | Redis MONITOR | 延迟统计 |
| 战斗计算 | < 100ms (10回合) | 服务端计时 | 单元测试 |
| 服务启动 | < 5s (到 ready) | K8s 就绪探测 | 滚动更新 |
| 可用性 SLA | 99.9% | Prometheus Uptime | 月度统计 |

### 9.2 压力测试场景

| 场景 | 并发数 | 持续时间 | 预期指标 |
|------|--------|---------|---------|
| WebSocket 连接风暴 | 10,000/s | 60s | 连接成功率 > 99%, P99 < 100ms |
| 修炼消息 | 50,000/s | 300s | P50 < 10ms, P99 < 50ms |
| 战斗计算 | 1,000/s | 60s | 单次 < 100ms, 无累积延迟 |
| 排行榜查询 | 10,000/s | 60s | 60s 内完成, P99 < 30ms |
| 混合场景 | 30,000 | 600s | 全部指标达标 |

### 9.3 资源配额与伸缩

| 服务 | CPU 限制 | 内存限制 | 最小副本 | 最大副本 | 触发伸缩 (CPU) |
|------|---------|---------|---------|---------|---------------|
| gateway | 0.5 core | 512M | 2 | 10 | > 70% |
| auth | 0.3 core | 256M | 2 | 10 | > 70% |
| player | 0.3 core | 256M | 2 | 10 | > 70% |
| cultivation | 0.4 core | 512M | 2 | 10 | > 70% |
| combat | 0.4 core | 512M | 2 | 10 | > 70% |
| social | 0.3 core | 256M | 2 | 10 | > 70% |
| world | 0.3 core | 256M | 2 | 10 | > 70% |
| trade | 0.3 core | 256M | 1 | 5 | > 70% |
| ranking | 0.2 core | 256M | 1 | 5 | > 70% |
| MySQL | 1.0 core | 1G | 1 | 1 (主) | - |
| Redis | 0.5 core | 512M | 1 | 3 (哨兵) | - |
| MongoDB | 1.0 core | 1G | 1 | 3 (副本集) | - |

---

## 10. 扩展性

### 10.1 插件化架构

系统设计了完整的插件系统 (`shared/plugin/plugin.go`)，所有业务模块以插件形式注册到主进程中。

```go
// GamePlugin 接口 - 每个业务模块实现此接口
type GamePlugin interface {
    Name() string                          // 全局唯一名称
    Version() string                       // 语义版本号
    OnInit(ctx GameContext) error          // 初始化 (加载配置/建立连接)
    OnStart() error                        // 启动 (开始处理消息)
    OnStop() error                         // 停止 (释放资源)
    RegisterHandlers(router *Router)       // 注册消息 ID -> 处理器
    RegisterEvents(bus *EventBus)          // 注册事件监听器
}

// PluginManager 管理插件生命周期
type PluginManager struct { /* ... */ }
func (pm *PluginManager) Register(p GamePlugin) error  // 注册
func (pm *PluginManager) InitAll() error                // 初始化所有
func (pm *PluginManager) StartAll() error               // 启动所有
func (pm *PluginManager) StopAll()                      // 逆序停止
```

**插件化优势：**
- 每个业务模块可独立开发、测试、部署
- 新功能只需注册新插件，无需修改主框架
- 插件可热插拔（停止/启动不影响其他插件）

### 10.2 事件驱动架构

基于自研事件总线 (`shared/eventbus/eventbus.go`) 实现进程内事件驱动。

```go
// 创建事件总线
bus := eventbus.New(logger)
bus.Use(eventbus.Recover(logger))   // panic 恢复中间件
bus.Use(eventbus.Logging(logger))   // 日志中间件
bus.Use(eventbus.Timeout(5*time.Second)) // 超时中间件

// 通配符订阅
bus.Subscribe("player.*", handler)         // 匹配 player.login, player.levelup
bus.Subscribe("combat.**", handler)        // 匹配 combat.start, combat.round.end
bus.Subscribe("*", globalHandler)          // 匹配所有事件

// 发布事件
bus.PublishAsync("player.levelup", data)  // 异步 (非阻塞)
bus.PublishSync("combat.finish", data)    // 同步 (阻塞等待)
```

**核心事件表：**

| 事件主题 | 触发时机 | 数据载荷 | 监听者 | 处理逻辑 |
|---------|---------|---------|--------|---------|
| `player.levelup` | 玩家升级 | {player_id, old_level, new_level} | Achievement | 检查成就进度 |
| `player.breakthrough` | 境界突破 | {player_id, old_realm, new_realm} | Ranking | 更新排行榜 |
| `combat.finish` | 战斗结束 | {combat_id, winner_id, exp_reward} | Quest | 检查任务进度 |
| `combat.finish` | 战斗结束 | {combat_id, winner_id, exp_reward} | Notification | 发送战报推送 |
| `world.encounter` | 触发奇遇 | {player_id, encounter_id, choice_id} | Luck | 计算气运变化 |
| `trade.deal` | 交易成交 | {listing_id, buyer, seller, price} | Social | 发送系统消息 |
| `trade.deal` | 交易成交 | {listing_id, buyer, seller, price} | Ranking | 更新财富排行 |
| `player.login` | 玩家登录 | {player_id, timestamp} | Social | 通知好友上线 |
| `player.logout` | 玩家登出 | {player_id, timestamp} | Social | 通知好友离线 |

### 10.3 数据驱动设计

游戏所有玩法参数通过 **配置表** 控制，而非硬编码。

| 配置类型 | 存储方式 | 热加载 | 变更影响 |
|---------|---------|--------|---------|
| 境界配置 | MySQL `realm_config` | 否 (启动加载) | 新增境界/调整升级曲线 |
| 物品模板 | MySQL `item_templates` | 是 (缓存 60s) | 新增物品/调整价格属性 |
| 功法模板 | MySQL `technique_templates` | 是 (缓存 60s) | 新增功法/调整加成 |
| 技能模板 | MySQL `skill_templates` | 是 (缓存 60s) | 新增技能/调整倍率 |
| 系统参数 | MySQL `system_config` | 是 (缓存 30s) | 调整全局倍率/税率 |
| 服务配置 | JSON/YAML 文件 | 是 (fsnotify) | 调整端口/连接池/日志级别 |
| 环境变量 | OS Env | 否 (重启生效) | 数据库密码/JWT 密钥 |

**配置热重载流程：**
```
文件修改 (vi/vscode 保存)
    │
    ├── fsnotify 捕获 Write/Rename 事件
    │
    ├── 等待 50ms (防抖动，避免多次保存触发多次重载)
    │
    ├── Load(): 重新读取文件
    │   ├── 检测扩展名 (.json/.yaml/.yml)
    │   ├── 解析为 map[string]interface{}
    │   └── 环境变量覆盖同名配置项
    │
    ├── 写锁更新内存 config.cfg
    │
    └── 通知所有 OnChange 回调
        └── 各模块执行刷新逻辑 (重建连接池/刷新缓存)
```

### 10.4 水平扩展

| 维度 | 扩展方式 | 实现 |
|------|---------|------|
| 无状态服务 | K8s HPA 多副本 | 9 个业务服务均可水平扩展 |
| 数据库 | 主从/分片 | MySQL 主从复制，Redis 哨兵，MongoDB 副本集 |
| 计算密集型 | 独立伸缩 | Combat / Cultivation 按需增加副本 |
| I/O 密集型 | 连接池优化 | MySQL 最大连接 200，Redis 连接池 50 |
| 跨实例通信 | 无状态设计 | 服务间通过 HTTP 调用，不共享内存 |

### 10.5 未来可扩展方向

| 方向 | 方案 | 优先级 | 说明 |
|------|------|--------|------|
| 跨服匹配 | 增加 Matchmaker 服务 | 中 | Redis Pub/Sub 跨服同步匹配队列 |
| 联盟/跨服战 | 增加 CrossServer 网关 | 低 | 状态同步 + 战斗广播 |
| AI 机器人 | Combat 集成 AI 决策 | 中 | 简单行为树 -> 神经网络 |
| 观战/直播 | WebSocket 广播流 | 低 | 延迟 30 秒的只读流 |
| 新压缩算法 | Zstandard 支持 | 低 | 比 Gzip 快 5x, 压缩率相近 |
| 国际化 | i18n 资源文件 | 中 | 语言文件 + 中间件注入 |
| 移动端 SDK | Unity / Unreal | 低 | WebSocket + Protobuf 复用 |
| 日志分析 | ELK Stack 集成 | 中 | Filebeat -> Logstash -> ES |

---

> 相关代码路径：
> - 微服务实现: `/root/cultivation-game/services/`
> - 共享核心库: `/root/cultivation-game/shared/` (plugin, eventbus, codec, config, models)
> - 协议定义: `/root/cultivation-game/proto/` (12 个 proto 文件)
> - 数据库脚本: `/root/cultivation-game/database/mysql/` (6 个 SQL 文件, 24 张表)
> - Redis 脚本: `/root/cultivation-game/database/redis/scripts/` (6 个 Lua 脚本)
> - 部署配置: `/root/cultivation-game/deploy/` (Compose + K8s)
> - 监控配置: `/root/cultivation-game/monitoring/` (Prometheus + Loki + Alerts)
