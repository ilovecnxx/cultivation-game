# 九转修仙 - Cultivation Game

MMORPG 修仙题材网页游戏，采用微服务架构构建的后端 + Vue 3 前端。

## 玩法概述

玩家从一介凡人开始，通过修炼、突破、战斗、探索，逐步登临仙道之巅。

- **修炼突破**：打坐积累修为，冲击境界瓶颈。境界体系从"凡人"到"渡劫"共九个大境界，每个境界九层小境界。
- **战斗系统**：支持 PVE（刷怪、副本）和 PVP（竞技场、匹配对战），五行相克影响战斗结果。
- **世界探索**：开放式地图，包含 NPC 交互、采集资源、奇遇事件。
- **社交系统**：好友、宗门、聊天、邮件，支持双修等交互玩法。
- **交易系统**：玩家间自由交易，灵石作为流通货币。
- **装备系统**：多种品质装备，支持强化、镶嵌。

## 技术栈

| 层次 | 技术 | 说明 |
|------|------|------|
| 后端语言 | Go 1.22 | 高性能编译型语言 |
| 前端框架 | Vue 3 + TypeScript | 组合式 API + Pinia 状态管理 |
| 构建工具 | Vite 5 | 快速 HMR 开发体验 |
| 网关 | Gorilla WebSocket + NATS | 长连接管理 + 消息路由 |
| 服务间通信 | NATS / gRPC | 异步消息 + 同步 RPC |
| 关系数据库 | MySQL 8.0 | 玩家数据、物品等结构化数据 |
| 文档数据库 | MongoDB 7.0 | 社交消息、日志等半结构化数据 |
| 缓存 | Redis 7.2 Cluster | 会话、排行榜、缓存 |
| 容器化 | Docker + Docker Compose | 开发和部署环境一致 |
| CI/CD | GitHub Actions | 自动构建、测试、部署 |

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                       客户端 (Vue 3)                         │
│              WebSocket (长连接) / REST API                   │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                   Gateway 网关服务                           │
│    WebSocket 连接管理 / JWT 鉴权 / 限流 / NATS 路由          │
└──────────┬──────────┬──────────┬──────────┬─────────────────┘
           │          │          │          │
    ┌──────▼──┐ ┌─────▼───┐ ┌───▼────┐ ┌──▼──────┐
    │  Player │ │ Combat  │ │Cultivat.│ │ Social  │
    │ 玩家服务 │ │ 战斗服务 │ │ 修炼服务 │ │ 社交服务 │
    ├─────────┤ ├─────────┤ ├────────┤ ├─────────┤
    │  MySQL   │ │ 内存     │ │ 内存+   │ │MongoDB +│
    │  Redis   │ │ 数据文件 │ │ 数据文件 │ │ Redis   │
    └──────────┘ └─────────┘ └────────┘ └─────────┘
```

## 快速开始

确保已安装 Docker 和 Docker Compose，然后执行：

```bash
# 1. 克隆仓库
git clone <repo-url> && cd cultivation-game

# 2. 启动基础设施 (MySQL + Redis Cluster + MongoDB)
docker compose -f database/redis/docker-compose.redis.yml up -d
docker compose -f database/mysql/docker-compose.yml up -d
docker compose -f database/mongodb/docker-compose.yml up -d

# 3. 启动后端服务 (每个服务一个终端)
go run ./services/gateway/cmd/gateway/main.go
go run ./services/auth/cmd/auth/main.go
go run ./services/player/cmd/player/main.go
go run ./services/combat/cmd/combat/main.go
go run ./services/cultivation/cmd/cultivation/main.go
go run ./services/social/cmd/social/main.go
go run ./services/world/cmd/world/main.go
go run ./services/trade/cmd/trade/main.go
go run ./services/ranking/cmd/ranking/main.go

# 4. 启动前端
cd frontend && npm install && npm run dev

# 5. 打开浏览器访问 http://localhost:3000
```

## 项目结构

```
cultivation-game/
├── .github/workflows/       # CI/CD 配置
├── database/                # 数据层配置和脚本
│   ├── mysql/               #   MySQL 初始化脚本
│   ├── mongodb/             #   MongoDB Schema
│   └── redis/               #   Redis 集群、Sentinel、Lua 脚本
├── frontend/                # Vue 3 前端
│   ├── src/
│   │   ├── components/      #   通用组件
│   │   ├── composables/     #   组合式函数
│   │   ├── core/            #   核心模块 (网络、存储、事件)
│   │   ├── modules/         #   业务模块 (战斗、修炼、背包等)
│   │   ├── styles/          #   样式 (PC/移动端响应式)
│   │   ├── types/           #   TypeScript 类型定义
│   │   └── views/           #   页面视图
│   └── vite.config.ts
├── services/                # Go 微服务 (9个)
│   ├── gateway/             #   网关 (WebSocket/NATS/JWT/限流)
│   ├── auth/                #   认证 (注册/登录/JWT/刷新)
│   ├── player/              #   玩家 (CRUD/背包/装备/洞府/灵兽/法宝)
│   ├── combat/              #   战斗 (PVE/PVP/匹配/副本/心魔塔/世界BOSS)
│   ├── cultivation/         #   修炼 (境界/功法/突破/天劫/炼丹/心魔)
│   ├── social/              #   社交 (聊天/好友/宗门/邮件/道侣/师徒)
│   ├── world/               #   世界 (地图/探索/NPC/奇遇/灵脉)
│   ├── trade/               #   交易 (交易行/拍卖/黑市/灵石)
│   └── ranking/             #   排行 (战力/境界/财富/竞技)
└── shared/                  # 共享库
    ├── config/              #   公共配置
    ├── eventbus/            #   事件总线
    ├── models/              #   公共数据模型
    ├── proto/               #   Protobuf 定义
    └── plugin/              #   插件系统
```

## 开发指南

详见 [docs/DEV_GUIDE.md](docs/DEV_GUIDE.md)。

## 部署指南

详见 [docs/DEPLOY.md](docs/DEPLOY.md)。

## 贡献

详见 [CONTRIBUTING.md](CONTRIBUTING.md)。
