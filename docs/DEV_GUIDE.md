# 凡人修仙模拟器 — 开发指南

> 版本: 2.0 | 最后更新: 2026-06-05  
> 本文档覆盖环境搭建、项目结构、添加新功能、代码规范、测试指南与调试技巧。

---

## 目录

1. [环境搭建](#1-环境搭建)
2. [项目结构说明](#2-项目结构说明)
3. [如何添加新功能](#3-如何添加新功能)
4. [代码规范](#4-代码规范)
5. [测试指南](#5-测试指南)
6. [调试技巧](#6-调试技巧)
7. [常用命令](#7-常用命令)

---

## 1. 环境搭建

### 1.1 必备工具

| 工具 | 最低版本 | 安装验证 | 用途 |
|------|---------|---------|------|
| Go | 1.22 | `go version` | 后端微服务开发 |
| Node.js | 20 LTS | `node --version` | 前端 Vue 开发 |
| npm | 10+ | `npm --version` | 前端包管理 |
| Docker | 24+ | `docker --version` | 容器化运行 |
| Docker Compose | v2+ | `docker compose version` | 多服务编排 |
| MySQL Client | 8.0+ | `mysql --version` | 数据库管理 |
| Redis CLI | 7+ | `redis-cli --version` | 缓存管理 |
| MongoDB Shell | 7+ | `mongosh --version` | 文档数据库管理 |
| Git | 2.40+ | `git --version` | 版本控制 |

### 1.2 安装指南

```bash
# Go 安装 (推荐使用官方安装包)
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Node.js 安装 (推荐使用 nvm)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
nvm install 20
nvm use 20

# Docker 安装
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 工具安装
sudo apt-get install mysql-client redis-tools -y
```

### 1.3 快速启动 (5 分钟)

```bash
# 1. 克隆仓库
git clone <repo-url> /root/cultivation-game
cd /root/cultivation-game

# 2. 启动基础设施 (数据库)
docker compose -f deploy/docker-compose.yml up -d redis mysql mongo

# 3. 安装前端依赖并启动
cd frontend
npm install
npm run dev &
cd ..

# 4. 启动后端服务 (每个服务一个新终端)
cd services/gateway && go run . &
cd services/auth && go run . &
cd services/player && go run . &
cd services/cultivation && go run . &
cd services/combat && go run . &

# 5. 验证
curl http://localhost:8080/health
curl http://localhost:8082/health
```

---

## 2. 项目结构说明

### 2.1 顶层目录结构

```
/root/cultivation-game/
├── backend/                   # 后端代码 (如有独立入口)
├── build/                     # 构建脚本
├── database/                  # 数据库脚本
│   ├── mysql/                 # MySQL 建表 + 种子数据
│   ├── mongodb/               # MongoDB Schema 示例
│   └── redis/                 # Redis 配置 + Lua 脚本
├── deploy/                    # 部署配置
│   ├── docker-compose.yml     # Docker Compose 编排
│   ├── k8s/                   # K8s 生产部署 (25 YAML)
│   └── nginx/                 # Nginx 配置
├── docs/                      # 文档 (本文档所在目录)
├── frontend/                  # Vue 3 前端
│   ├── src/                   # 源代码
│   ├── vite.config.ts         # Vite 构建配置
│   └── package.json           # 依赖清单
├── monitoring/                # 监控配置
│   ├── prometheus/            # Prometheus 指标 + 告警
│   ├── loki/                  # Loki 日志聚合
│   └── promtail/              # Promtail 日志采集
├── proto/                     # Protobuf 协议定义 (12 个 .proto)
├── services/                  # Go 微服务 (9 个)
│   ├── gateway/               # #1 网关服务
│   ├── auth/                  # #2 认证服务
│   ├── player/                # #3 玩家服务
│   ├── cultivation/           # #4 修炼服务
│   ├── combat/                # #5 战斗服务
│   ├── social/                # #6 社交服务
│   ├── world/                 # #7 世界服务
│   ├── trade/                 # #8 交易服务
│   └── ranking/               # #9 排行榜服务
├── shared/                    # Go 共享核心库
│   ├── config/                # 配置热重载
│   ├── eventbus/              # 事件总线
│   ├── models/                # 数据模型
│   ├── codec/                 # 编解码器
│   └── plugin/                # 插件接口
├── .github/workflows/         # CI/CD Workflow
├── README.md                  # 项目介绍
├── CONTRIBUTING.md            # 贡献指南
└── CHANGELOG.md               # 变更日志
```

### 2.2 单个服务内部结构 (以 Combat 为例)

```
services/combat/
├── Dockerfile                 # 容器镜像构建
├── go.mod                     # Go 模块定义
├── go.sum                     # 依赖哈希
├── config.json                # 服务配置 (端口/数据库连接)
├── main.go                    # 入口: 启动 HTTP 服务器
├── internal/                  # 内部包 (不对外暴露)
│   ├── handler/               # HTTP 处理器 (路由绑定)
│   │   └── combat_handler.go  # 战斗相关 API 处理器
│   ├── service/               # 业务逻辑层
│   │   └── combat_service.go  # 战斗业务流程编排
│   ├── engine/                # 核心引擎
│   │   ├── battle.go          # 战斗引擎 (回合循环)
│   │   └── damage.go          # 伤害计算逻辑
│   ├── model/                 # 内部数据模型
│   │   ├── fighter.go         # 战斗单元定义
│   │   └── skill.go           # 技能定义
│   └── repository/            # 数据访问层 (可选)
├── api/                       # gRPC 生成代码 (如有)
│   ├── combat.pb.go           # Protobuf 消息结构
│   └── combat_grpc.pb.go      # gRPC 客户端/服务端
└── combat_test.go             # 测试文件
```

### 2.3 前端结构

```
frontend/
├── src/
│   ├── main.ts                # Vue 应用入口
│   ├── App.vue                # 根组件
│   ├── router/
│   │   └── index.ts           # 路由配置 (10 条路由)
│   ├── core/                  # 核心模块
│   │   ├── api.ts             # HTTP API 封装 (带 Token)
│   │   ├── network/           # WebSocket 网络层
│   │   │   ├── WebSocketClient.ts  # WS 客户端 (自动重连)
│   │   │   ├── MessageCodec.ts     # 消息编解码
│   │   │   └── index.ts
│   │   ├── event/
│   │   │   └── EventBus.ts    # 前端事件总线
│   │   └── store/             # Pinia 状态管理
│   │       ├── player.ts      # 玩家状态
│   │       ├── combat.ts      # 战斗状态
│   │       └── world.ts       # 世界状态
│   ├── views/                 # 页面组件
│   │   ├── Home.vue           # 落地页
│   │   ├── LoginView.vue      # 登录页
│   │   ├── RegisterView.vue   # 注册页
│   │   ├── GameLayout.vue     # 游戏主布局 (含导航)
│   │   ├── HomeView.vue       # 游戏首页
│   │   └── AdminView.vue      # 管理后台
│   ├── modules/               # 业务模块视图
│   │   ├── social/            # 社交 (聊天/好友/宗门)
│   │   ├── alchemy/           # 炼丹
│   │   ├── trade/             # 交易行
│   │   ├── inventory/         # 背包
│   │   └── combat/            # 战斗
│   ├── components/            # 通用组件
│   │   ├── navigation/NavBar.vue   # 底部导航栏
│   │   ├── CombatLog.vue           # 战斗日志组件
│   │   ├── VirtualList.vue         # 虚拟滚动列表
│   │   └── ThemeSwitcher.vue       # 主题切换
│   ├── composables/           # 组合式函数
│   │   ├── useVirtualScroll.ts     # 虚拟滚动逻辑
│   │   └── useNumberScroll.ts      # 数字滚动动画
│   ├── styles/                # 样式
│   │   ├── variables.scss     # 变量定义
│   │   ├── mixins.scss        # 混合宏
│   │   ├── themes.scss        # 主题系统
│   │   ├── pc.scss            # PC 端适配
│   │   └── mobile.scss        # 移动端适配
│   └── types/                 # TypeScript 类型定义
│       ├── game.ts            # 游戏核心类型
│       ├── game.d.ts          # 类型声明
│       └── index.ts           # 类型导出
├── package.json
├── tsconfig.json
├── vite.config.ts
└── env.d.ts
```

---

## 3. 如何添加新功能

### 3.1 添加新境界

境界配置存储在 MySQL `realm_config` 表和 Go 代码的 `shared/models/realm.go` 中。

**步骤：**

```bash
# 1. 向 realm_config 表插入新境界数据
mysql -h localhost -u root -p cultivation_game
```

```sql
-- realm_group: 9=飞升 (第 9 大境界)
INSERT INTO realm_config (id, realm_name, realm_group, level, upgrade_exp,
    base_hp, base_mp, base_attack, base_defense, base_speed, description)
VALUES
(38, '飞升初期', 9, 1, 25000000, 2000000, 2000000, 120000, 95000, 2600, '超凡入圣'),
(39, '飞升中期', 9, 2, 35000000, 2800000, 2800000, 160000, 125000, 3100, '飞升中期'),
(40, '飞升后期', 9, 3, 0,         3800000, 3800000, 220000, 170000, 3700, '飞升后期，至此已是凡间极致');
```

```go
// 2. 在 shared/models/realm.go 中添加 RealmType 枚举 (如果使用枚举)
const (
    RealmAscension  RealmType = 8  // 原大乘改名为飞升
)

// 3. 配置文件中增加境界加成倍率 (如 cultivaiton/config.json)
{
  "realm_multipliers": {
    "9": { "exp_rate": 0.5, "breakthrough_base": 0.05 }
  }
}
```

```typescript
// 4. 前端添加境界展示
// frontend/src/types/game.ts
export const REALM_NAMES: Record<number, string> = {
  1: '练气期', 2: '筑基期', 3: '金丹期',
  4: '元婴期', 5: '化神期', 6: '合体期',
  7: '大乘期', 8: '渡劫期', 9: '飞升期',
};
```

### 3.2 添加新怪物

怪物配置存储在 Combat 服务的 JSON 文件中。

**步骤：**

```json
// 1. services/combat/config.json 或独立的 monsters.json 中添加怪物
{
  "monsters": {
    "38": {
      "id": 38,
      "name": "远古蛟龙",
      "level_required": 35,
      "realm_group": 8,
      "element": "water",
      "hp": 500000,
      "mp": 200000,
      "attack": 12000,
      "defense": 8000,
      "speed": 500,
      "crit_rate": 15.0,
      "dodge_rate": 10.0,
      "skills": [7, 4, 10],
      "exp_reward": 50000,
      "money_reward": 20000,
      "drops": [
        {"item_id": 22, "min_qty": 1, "max_qty": 3, "rate": 0.8},
        {"item_id": 20, "min_qty": 1, "max_qty": 5, "rate": 0.5}
      ],
      "description": "修炼万年的远古蛟龙，盘踞在深潭之中，实力堪比渡劫修士"
    }
  }
}
```

```go
// 2. 在 Combat 服务中添加怪物 ID 到 Fighters 的转换
// services/combat/internal/service/combat_service.go
func (s *CombatService) loadMonster(monsterID uint32) (*model.Fighter, error) {
    monsterCfg := s.config.GetMonster(monsterID)
    if monsterCfg == nil {
        return nil, fmt.Errorf("monster %d not found", monsterID)
    }
    return &model.Fighter{
        ID:     uint64(monsterID) + 1000000, // 怪物 ID 偏移
        Name:   monsterCfg.Name,
        Attr:   monsterCfg.ToAttributes(),
        HP:     monsterCfg.HP,
        MaxHP:  monsterCfg.HP,
        Skills: monsterCfg.LoadSkills(),
    }, nil
}
```

### 3.3 添加新任务

任务系统集成在 World 服务中。

**步骤：**

```sql
-- 1. 数据库添加任务模板 (新增 quest_templates 表)
CREATE TABLE IF NOT EXISTS quest_templates (
    id            INT           AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(64)   NOT NULL,
    quest_type    TINYINT       NOT NULL COMMENT '1主线/2支线/3日常/4宗门',
    level_required INT         DEFAULT 1,
    realm_required INT         DEFAULT NULL,
    npc_id        INT           DEFAULT NULL,
    description   TEXT          DEFAULT NULL,
    objectives    JSON          NOT NULL COMMENT '[{"type":"kill","target":38,"count":3},{"type":"collect","item_id":20,"count":5}]',
    rewards       JSON          NOT NULL COMMENT '{"exp":10000,"money":5000,"items":[{"id":2,"qty":5}]}',
    created_at    DATETIME      DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_type (quest_type),
    INDEX idx_realm (realm_required)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务模板表';
```

```go
// 2. 实现任务进度追踪
// services/world/internal/service/quest_service.go
type QuestService struct {
    db    *gorm.DB
    cache *redis.Client
}

// AcceptQuest 接取任务
func (s *QuestService) AcceptQuest(playerID uint64, questID uint32) error {
    // 校验前置条件 (等级/境界/前置任务)
    // 创建任务进度记录
    // 返回任务详情
}

// UpdateQuestProgress 更新任务进度 (由事件驱动)
func (s *QuestService) UpdateQuestProgress(playerID uint64, eventType string, targetID uint32, count int) {
    // 查询玩家进行中的任务
    // 匹配 objectives
    // 更新进度
    // 如果全部完成，标记可提交
}
```

```vue
<!-- 3. 前端任务界面 -->
<template>
  <div class="quest-panel">
    <div v-for="quest in quests" :key="quest.id" class="quest-card">
      <div class="quest-header">
        <span class="quest-name">{{ quest.name }}</span>
        <span class="quest-type" :class="quest.type">{{ typeLabel(quest.type) }}</span>
      </div>
      <div class="quest-progress">
        <div v-for="obj in quest.objectives" :key="obj.id" class="objective">
          <span>{{ obj.description }}</span>
          <span class="progress">{{ obj.current }}/{{ obj.target }}</span>
        </div>
      </div>
      <button v-if="quest.canSubmit" @click="submitQuest(quest.id)">提交</button>
    </div>
  </div>
</template>
```

### 3.4 添加新 API

**步骤（后端）：**

```go
// 1. 在 handler 中定义处理函数
// services/player/internal/handler/player_handler.go

func (h *PlayerHandler) GetPlayerEquipment(c *gin.Context) {
    playerID := c.Param("id")
    
    // 调用 service 层
    equipment, err := h.playerService.GetEquipment(playerID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, equipment)
}

// 2. 注册路由
// services/player/main.go (或 routes.go)

func setupRoutes(r *gin.Engine, h *PlayerHandler) {
    v1 := r.Group("/api/v1/player")
    {
        v1.GET("/:id", h.GetPlayerInfo)
        v1.GET("/:id/inventory", h.GetInventory)
        v1.GET("/:id/equipment", h.GetPlayerEquipment)  // 新增路由
        // ...
    }
}
```

**步骤（前端）：**

```typescript
// 3. 在 api.ts 中添加 API 调用方法
// frontend/src/core/api.ts

export async function getPlayerEquipment(playerID: number): Promise<Equipment[]> {
    return apiFetch(`/api/v1/player/${playerID}/equipment`)
}

// 4. 如果通过 WebSocket 通信，在 MessageCodec 中注册消息 ID
// frontend/src/core/network/MessageCodec.ts

const MSG_PLAYER_EQUIPMENT = 203;  // 新增消息 ID

// 5. 在 Vue 组件中调用
// frontend/src/modules/inventory/InventoryView.vue

const equipment = ref<Equipment[]>([])
onMounted(async () => {
    const pid = localStorage.getItem('player_id')
    if (pid) {
        equipment.value = await getPlayerEquipment(Number(pid))
    }
})
```

### 3.5 添加新 Protobuf 消息

```protobuf
// 1. 在 proto/ 目录下定义新消息
// proto/player.proto

message EquipItemReq {
  uint64 player_id = 1;
  uint32 item_id = 2;
  string slot = 3;
}

message EquipItemResp {
  bool success = 1;
  int64 combat_power_change = 2;
}

// 2. 在 common.proto 中添加消息 ID 枚举
enum MsgId {
  // ...
  MSG_PLAYER_EQUIP = 203;
  MSG_PLAYER_EQUIP_RES = 204;
}

// 3. 编译生成 Go 代码
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/player.proto

// 4. 注册消息处理器
// services/gateway/main.go 或 router
router.Handle(203, playerPlugin.EquipItem)
```

### 3.6 添加新 Pinia Store

```typescript
// frontend/src/core/store/trade.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '@/core/api'

export const useTradeStore = defineStore('trade', () => {
    const listings = ref<Listing[]>([])
    const myOrders = ref<Listing[]>([])
    const isLoading = ref(false)

    async function fetchListings(filter?: { itemType?: number, page?: number }) {
        isLoading.value = true
        try {
            const data = await apiFetch('/api/v1/trade/listings?' + new URLSearchParams(filter as any))
            listings.value = data.items
        } finally {
            isLoading.value = false
        }
    }

    async function placeOrder(itemId: number, quantity: number, price: number) {
        return apiFetch('/api/v1/trade/sell', {
            method: 'POST',
            body: JSON.stringify({ item_id: itemId, quantity, unit_price: price })
        })
    }

    return { listings, myOrders, isLoading, fetchListings, placeOrder }
})
```

---

## 4. 代码规范

### 4.1 Go 代码规范

| 规则 | 规范 | 示例 |
|------|------|------|
| 命名 | 驼峰式 | `playerService`, `GetPlayerInfo` |
| 包名 | 小写单数 | `package service` |
| 文件命名 | 蛇形 | `player_handler.go` |
| 接口命名 | -er 结尾 | `type Storage interface` |
| 错误处理 | 必须处理 error | `if err != nil { return err }` |
| 错误信息 | 小写开头, 不带标点 | `fmt.Errorf("player %d not found", id)` |
| 导入分组 | std -> 第三方 -> 内部 | 空行分隔 |
| 注释 | 英文, 完整句 | `// GetPlayerInfo returns player details.` |
| 导出 | 大写开头 | `func NewService() *Service` |
| 未导出 | 小写开头 | `func calculateExp() uint64` |
| 长度限制 | 行 < 120 字符 | 超过则换行 |
| 资源关闭 | defer | `defer resp.Body.Close()` |
| 并发安全 | 明确注释 | `// mu guards the players map` |

**Import 分组示例：**

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "cultivation-game/shared/config"
    "cultivation-game/shared/models"
)
```

**错误处理模式：**

```go
// 推荐: 尽早返回
value, err := fetchData(id)
if err != nil {
    return fmt.Errorf("fetchData failed: %w", err)
}
result, err := processValue(value)
if err != nil {
    return fmt.Errorf("processValue failed: %w", err)
}
return result, nil

// 不推荐: 深层嵌套
if value, err := fetchData(id); err == nil {
    // ...
}
```

### 4.2 TypeScript / Vue 规范

| 规则 | 规范 | 示例 |
|------|------|------|
| 命名 | 驼峰式 | `playerStore`, `getInventory()` |
| 组件名 | PascalCase | `InventoryView.vue` |
| 类型定义 | PascalCase | `interface PlayerInfo` |
| 文件名 | PascalCase (组件) / camelCase (工具) | `PlayerInfo.vue` / `api.ts` |
| 模板 | 组合式 API + `<script setup>` | `defineComponent` |
| Prop 定义 | 指定类型和默认值 | `props: { count: { type: Number, default: 0 } }` |
| 状态管理 | Pinia + setup store | `export const useStore = defineStore('name', () => {})` |
| 类型声明 | 优先 interface 而非 type | `interface PlayerInfo { ... }` |
| 可选链 | 使用 `?.` 和 `??` | `data?.items ?? []` |
| 异步 | async/await | `const data = await apiFetch(...)` |
| 样式 | SCSS scoped | `<style scoped lang="scss">` |

**Vue 组件模板：**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { PlayerInfo } from '@/types'

const props = defineProps<{
  playerId: number
}>()

const emit = defineEmits<{
  (e: 'update', value: PlayerInfo): void
}>()

const player = ref<PlayerInfo | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    player.value = await apiFetch(`/api/v1/player/${props.playerId}`)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div v-if="loading">加载中...</div>
  <div v-else-if="player" class="player-card">
    <h2>{{ player.nickname }}</h2>
    <p>境界: {{ player.realm }}</p>
  </div>
</template>

<style scoped lang="scss">
.player-card {
  padding: 16px;
  border: 1px solid var(--border-color);
}
</style>
```

### 4.3 Git 提交规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

| Type | 用途 |
|------|------|
| feat | 新功能 |
| fix | 修复 Bug |
| refactor | 重构 |
| docs | 文档 |
| test | 测试 |
| chore | 构建/工具 |
| perf | 性能优化 |
| style | 格式调整 |

**示例：**

```
feat(cultivation): add tribulation system for ascension realm

Implement 4-round tribulation trials for breakthrough from Epocenter to Ascension realm.
Each round tests a random attribute against a difficulty threshold.
Player must pass at least 3 of 4 rounds to succeed.

Closes #123
Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
```

---

## 5. 测试指南

### 5.1 测试分类

| 类型 | 覆盖范围 | 运行时间 | 运行频率 |
|------|---------|---------|---------|
| 单元测试 | 核心业务逻辑 | < 1ms/个 | 每次提交 |
| 集成测试 | Handler + DB | < 100ms/个 | 每次提交 |
| 端到端测试 | 完整流程 | < 5s/个 | CI (PR) |
| 压力测试 | 性能指标 | < 5min | 发布前 |

### 5.2 Go 测试

```bash
# 运行所有测试 (含竞态检测)
go test ./... -race -count=1 -timeout=60s

# 运行指定包测试 (详细输出)
go test ./services/combat/... -v

# 运行指定测试函数
go test ./services/combat/... -run TestBattleStart -v

# 生成覆盖率报告
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -html=coverage.out -o coverage.html

# 查看包覆盖率
go test ./... -cover
```

**单元测试示例：**

```go
// services/combat/internal/engine/battle_test.go
package engine

import (
    "testing"
    "cultivation-game/shared/models"
)

func setupTestFighters() (*models.Fighter, *models.Fighter) {
    player := &models.Fighter{
        ID:     1,
        Name:   "玩家",
        IsPlayer: true,
        Attr: models.PlayerAttribute{
            Health: 1000, Mana: 500, Strength: 100,
            Agility: 50, Spirit: 80, Defense: 50,
            Critical: 1000, Dodge: 500,
        },
        HP: 1000, MaxHP: 1000, MP: 500, MaxMP: 500,
        Skills: []models.Skill{
            {ID: 6, Name: "飞剑术", BaseDamage: 80, Coefficient: 2.2,
             AttrScale: "strength", DamageType: models.DamageTypePhysical,
             CostMana: 40, Cooldown: 0},
        },
    }
    monster := &models.Fighter{
        ID: 2, Name: "风狼",
        Attr: models.PlayerAttribute{
            Health: 200, Strength: 30, Agility: 60,
            Spirit: 10, Defense: 20, Critical: 500, Dodge: 800,
        },
        HP: 200, MaxHP: 200, MP: 50, MaxMP: 50,
    }
    return player, monster
}

func TestBattleStart(t *testing.T) {
    player, monster := setupTestFighters()
    engine := NewBattle([]*models.Fighter{player}, []*models.Fighter{monster})

    if engine.State != BattleStateRunning {
        t.Errorf("新战斗状态应为 Running, 得到 %s", engine.State)
    }
}

func TestBattlePlayerWin(t *testing.T) {
    player, monster := setupTestFighters()
    player.Attr.Strength = 10000 // 一击秒杀
    engine := NewBattle([]*models.Fighter{player}, []*models.Fighter{monster})
    result := engine.Start()

    if result.WinnerID != player.ID {
        t.Errorf("玩家应获胜, 但获胜者是 %d", result.WinnerID)
    }
    if result.TotalRounds != 1 {
        t.Errorf("预期 1 回合, 实际 %d 回合", result.TotalRounds)
    }
}

func TestDamageCalculation(t *testing.T) {
    attacker := &models.Fighter{
        Attr: models.PlayerAttribute{Strength: 100, Defense: 20},
    }
    defender := &models.Fighter{
        Attr: models.PlayerAttribute{Defense: 30},
        HP: 1000, MaxHP: 1000,
    }
    skill := &models.Skill{
        BaseDamage: 50, Coefficient: 1.5, AttrScale: "strength",
    }

    damage := CalculateDamage(attacker, defender, skill, models.DamageTypePhysical)
    // expected: (50 + 1.5*100) - 30/10 = 200 - 3 = 197
    if damage != 197 {
        t.Errorf("预期伤害 197, 得到 %d", damage)
    }
}
```

### 5.3 前端测试

```bash
# 类型检查
cd frontend
npx vue-tsc --noEmit

# Lint 检查
# TODO: 集成 ESLint + Prettier

# 组件测试 (TODO: 集成 Vitest)
# npx vitest run
```

### 5.4 端到端测试方案

```typescript
// TODO: e2e/test-flow.ts (使用 Playwright)
// 1. 用户注册
// 2. 用户登录
// 3. 进入游戏首页
// 4. 进行修炼
// 5. 查看背包
// 6. 开始战斗
// 7. 查看战斗结果
// 8. 退出登录
```

---

## 6. 调试技巧

### 6.1 后端调试

```bash
# 1. 查看服务日志
docker compose logs -f gateway         # 实时日志
docker compose logs gateway --tail=100 # 最后 100 行

# 2. Go 测试调试
go test -v -run TestBattlePlayerWin ./services/combat/...

# 3. 使用 Delve 调试 (安装: go install github.com/go-delve/delve/cmd/dlv@latest)
cd services/gateway
dlv debug . -- --port 8080

# 4. 在代码中添加调试日志
// services/combat/internal/engine/damage.go
import "log"
log.Printf("[DEBUG] damage calculation: base=%d, coeff=%.2f, attr=%d, result=%d",
    skill.BaseDamage, skill.Coefficient, attrValue, finalDamage)

# 5. 性能 Profiling
# 添加 pprof 端点
# import _ "net/http/pprof"
# go tool pprof http://localhost:8082/debug/pprof/heap
```

### 6.2 数据库调试

```bash
# 1. 慢查询日志
# MySQL 配置 (my.cnf):
# slow_query_log = 1
# long_query_time = 0.5
# 查看慢查询日志:
tail -f /var/log/mysql/mysql-slow.log

# 2. Redis 调试
redis-cli MONITOR                          # 实时命令监控
redis-cli --bigkeys                        # 大 key 扫描
redis-cli SLOWLOG GET 10                   # 慢查询
redis-cli INFO keyspace                    # key 统计

# 3. 数据库连接检查
# MySQL 当前连接:
mysql -e "SHOW FULL PROCESSLIST\G"
# Redis 当前连接:
redis-cli CLIENT LIST
```

### 6.3 网络调试

```bash
# 1. WebSocket 测试
npm install -g wscat
wscat -c ws://localhost:8080/ws?token=test

# 2. HTTP API 测试
curl -v http://localhost:8082/health                  # 查看完整请求/响应
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}' \
  -w "\nTime: %{time_total}s\n"                       # 计时

# 3. 网络延迟检查
ping localhost
tcping localhost 8080                                 # TCP 端口延迟
```

### 6.4 常见问题排查

| 问题 | 可能原因 | 排查命令 |
|------|---------|---------|
| 服务启动失败 | 端口被占用 | `lsof -i :8080` |
| 数据库连接拒绝 | DSN 错误 / 服务未启动 | `mysqladmin ping -h localhost -u root -p` |
| WebSocket 连接失败 | Token 无效 / Nginx 未配置 WS | `curl -H "Connection: Upgrade" -H "Upgrade: websocket" http://localhost:8080/ws` |
| 高 Goroutine 数 | goroutine 泄漏 | `curl http://localhost:8080/debug/pprof/goroutine?debug=2` |
| Redis 慢查询 | 大 key / 复杂命令 | `redis-cli SLOWLOG GET 20` |
| 前端白屏 | 构建错误 / 路由问题 | 浏览器 F12 -> Console / Network |
| 跨域请求失败 | CORS 配置不正确 | 检查 Nginx `Access-Control-Allow-Origin` |
| 消息延迟高 | 服务过载 / 网络瓶颈 | `kubectl top pods -n cultivation-game` |

### 6.5 热重载

```bash
# Go 热重载 (安装 air)
go install github.com/air-verse/air@latest
cd services/gateway && air

# 前端热重载 (Vite 自带)
cd frontend && npm run dev
# 修改 .vue 文件 > 1s 内浏览器自动刷新

# 配置热重载 (自研 config Loader)
# 修改 config.json 文件 -> 50ms 后自动生效
```

---

## 7. 常用命令

### 7.1 后端命令

```bash
# 初始化新服务
mkdir -p services/myservice/{cmd/myservice,internal/{handler,service,model,repository}}
cd services/myservice
go mod init cultivation-game/services/myservice
go mod tidy

# 编译
go build ./services/gateway/...

# 格式检查
gofmt -l -s ./services/...

# Go 依赖管理
go mod tidy                          # 清理无用依赖
go mod vendor                        # 导出 vendor
go list -m all                       # 列出所有依赖

# 代码检查 (安装: go install golang.org/x/tools/go/analysis/passes/...)
go vet ./...

# Protobuf 编译
protoc --go_out=. --go_opt=paths=source_relative proto/*.proto
```

### 7.2 前端命令

```bash
cd frontend

# 安装依赖
npm install

# 开发服务器 (HMR)
npm run dev                        # http://localhost:5173

# 生产构建
npm run build                      # 输出到 dist/

# 预览构建结果
npm run preview

# 类型检查
npx vue-tsc --noEmit

# 依赖更新
npm outdated                        # 查看过时依赖
npm update                          # 更新依赖
```

### 7.3 Docker 命令

```bash
# 构建
docker build -t cultivation-game/gateway:latest -f services/gateway/Dockerfile .

# 运行
docker run -d --name gateway -p 8080:8080 \
  -e JWT_SECRET=dev-secret \
  -e REDIS_URL=redis://host.docker.internal:6379 \
  cultivation-game/gateway:latest

# 查看日志
docker logs -f gateway

# 进入容器
docker exec -it gateway sh

# 多服务管理 (Compose)
docker compose -f deploy/docker-compose.yml up -d          # 启动
docker compose -f deploy/docker-compose.yml down           # 停止
docker compose -f deploy/docker-compose.yml restart gateway # 重启单个
docker compose -f deploy/docker-compose.yml ps             # 状态
docker compose -f deploy/docker-compose.yml logs -f -t     # 日志时间戳
```

### 7.4 K8s 命令

```bash
# 查看状态
kubectl get all -n cultivation-game
kubectl get pods -n cultivation-game -o wide
kubectl get hpa -n cultivation-game

# 查看日志
kubectl logs -n cultivation-game deployment/gateway -f
kubectl logs -n cultivation-game -l app=gateway --tail=100

# 进入 Pod
kubectl exec -it -n cultivation-game deployment/gateway -- sh

# 端口转发 (调试)
kubectl port-forward -n cultivation-game service/gateway 8080:8080

# 伸缩
kubectl scale deployment/gateway -n cultivation-game --replicas=5

# 滚动更新状态
kubectl rollout status deployment/gateway -n cultivation-game

# 回滚
kubectl rollout undo deployment/gateway -n cultivation-game

# 查看资源使用
kubectl top pods -n cultivation-game
kubectl top nodes
```

### 7.5 数据库命令

```bash
# MySQL
mysql -h localhost -u root -p cultivation_game
mysql> SHOW TABLES;
mysql> DESCRIBE players;
mysql> SELECT COUNT(*) FROM players;
mysql> EXPLAIN SELECT * FROM players WHERE realm_id = 10;

# Redis
redis-cli
redis:6379> INFO stats
redis:6379> KEYS session:*
redis:6379> TYPE online:players
redis:6379> SMEMBERS online:players
redis:6379> ZREVRANGE ranking:1:pvp 0 9 WITHSCORES

# MongoDB
mongosh
test> use cultivation_game
cultivation_game> db.chat_messages.countDocuments()
cultivation_game> db.chat_messages.find().sort({timestamp:-1}).limit(10)
cultivation_game> db.battle_replays.createIndex({player_id:1})
```

### 7.6 Git 命令

```bash
# 分支管理
git checkout -b feat/cultivation-tribulation   # 创建功能分支
git branch -d old-branch                        # 删除本地分支

# 提交
git add -p                                       # 交互式暂存
git commit -m "feat(cultivation): add tribulation"
git push origin feat/cultivation-tribulation

# 查看历史
git log --oneline --graph --all                  # 分支图
git log --author="name" --since="2026-01-01"     # 按作者时间筛选
git show <commit> --stat                         # 查看提交的文件变更

# 合并
git checkout main
git merge --no-ff feat/cultivation-tribulation   # 保留分支历史
```

---

> 相关文件：
> - 后端共享库: `/root/cultivation-game/shared/`
> - 微服务代码: `/root/cultivation-game/services/`
> - 前端代码: `/root/cultivation-game/frontend/src/`
> - 协议定义: `/root/cultivation-game/proto/`
> - 数据库脚本: `/root/cultivation-game/database/`
> - 测试文件: 各服务目录下的 `*_test.go` 文件
