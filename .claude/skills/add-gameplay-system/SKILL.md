---
name: add-gameplay-system
description: >
  Full-stack gameplay system generator for the 九转修仙 cultivation game.
  Use this skill when the user asks to add a complete new gameplay feature
  spanning backend + frontend + database, such as "添加新玩法", "新增系统",
  "add gameplay system", "实现XX功能(前后端都要)", "做一个新系统", "新建玩法模块".
  Also use when the user describes a feature that needs a new MySQL table,
  a new frontend view, and API endpoints — even if they don't explicitly say
  "full-stack". This skill ensures consistency across all layers.
---

# Add Gameplay System — 九转修仙项目

为九转修仙游戏项目全栈添加新玩法功能，涵盖后端 Go 微服务代码、前端 Vue 3 模块、MySQL 迁移文件、Proto 定义更新。严格遵循项目 CONTRIBUTING.md 代码规范。

## 工作流程概览

```
1. 需求分析  →  2. 数据建模  →  3. 后端实现  →  4. 前端实现  →  5. 集成注册
   玩法描述      MySQL 迁移      Handler         Vue 组件       路由/Store
   数值设计      Proto 定义      Service         Pinia Store    go.work
   交互流程      Model 定义      Repository      API 调用       CLAUDE.md
```

---

## 第一步：需求分析

在开始编码前，确认以下信息：

1. **玩法名称**（如 `灵兽孵化`, `宗门大战`, `秘境探宝`）
2. **玩法描述**（核心机制和流程）
3. **归属服务**：是新服务还是归属现有服务？
   - 新服务 → 也要触发 `add-microservice` 技能
   - 现有服务 → 在现有服务 `internal/` 下添加新模块
4. **数据存储**：需要哪些新表？（MySQL）/ 需要缓存结构？（Redis）
5. **前端位置**：`frontend/src/modules/<name>/` 下创建新模块
6. **是否需要 Proto 更新**：新增的消息类型需要定义

---

## 第二步：数据建模

### MySQL 迁移文件

创建 `database/mysql/020_<feature_name>.sql`，遵循以下规范：

```sql
-- ===================================================================
-- <玩法中文名> - 数据表
-- 版本: v1.0.0
-- ===================================================================

-- 1. 核心表
CREATE TABLE <table_name> (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键',
    player_id       BIGINT NOT NULL                COMMENT '玩家ID',
    -- 业务字段...
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_player_id (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='<玩法中文名>表';

-- 2. 配置表（如适用）
CREATE TABLE <table_name>_config (
    id              INT AUTO_INCREMENT PRIMARY KEY COMMENT '配置ID',
    -- 配置字段...
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='<玩法中文名>配置表';
```

**命名规约**：
- 表名：蛇形命名，如 `pet_eggs`, `sect_wars`, `dungeon_records`
- 字段：蛇形命名，`player_id` 作为外键
- 必须包含 `created_at` 和 `updated_at` 时间戳
- 必须包含 `COMMENT` 注释
- 必须包含合理索引
- 文件编号取 `database/mysql/` 下最大编号 + 1

### Proto 定义（如需要新消息类型）

在 `services/gateway/api/proto/gateway.proto` 中添加新消息：

```protobuf
// <玩法名>请求
message <Name>Request {
  uint64 player_id = 1;
  // 其他字段...
}

// <玩法名>响应
message <Name>Response {
  int32  code = 1;
  string msg  = 2;
  // 数据字段...
}
```

---

## 第三步：后端实现

按照项目既有的 Handler → Service → Repository 三层架构实现。

### 文件清单

在目标服务 `services/<svc>/internal/` 下创建：

```
internal/
├── handler/<feature>_handler.go      # HTTP handler
├── service/<feature>_service.go      # 业务逻辑
├── model/<feature>.go                # 数据模型
└── repository/
    └── mysql/<feature>_repo.go       # 数据访问
```

### Handler 代码规范

```go
package handler

// <Name>Handler <玩法名>处理器
// 所有公共类型和函数必须有文档注释
type <Name>Handler struct {
    svc *service.<Name>Service
    log *zap.Logger
}

// New<Name>Handler 创建 <玩法名>Handler
func New<Name>Handler(svc *service.<Name>Service, log *zap.Logger) *<Name>Handler {
    return &<Name>Handler{svc: svc, log: log}
}
```

- 每个 Handler 方法注释包含路由路径和用途
- 参数校验使用 Gin 的 `ShouldBindJSON` 或手动校验
- 错误统一返回 `{"code": <err_code>, "msg": "<描述>"}`
- 成功返回 `{"code": 0, "msg": "success", "data": <结果>}`

### Service 代码规范

```go
package service

// <Name>Service <玩法名>业务逻辑
// 负责业务规则校验、数据组装、跨模块调用
type <Name>Service struct {
    repo *mysqlRepo.<Name>Repo
    log  *zap.Logger
}
```

- 错误处理：使用 `fmt.Errorf("操作描述: %w", err)` 包装底层错误
- 事务操作在 Service 层控制
- 复杂业务逻辑拆分为多个私有方法

### Repository 代码规范

- 使用 `database/sql`（原生 SQL，与项目一致）或 GORM
- 查询方法返回 `(*model.Xxx, error)`，列表返回 `([]*model.Xxx, error)`
- 使用参数化查询防止注入
- 批量操作使用事务

### 主文件注册

在 `services/<svc>/cmd/<svc>/main.go` 中添加：

```go
// 在依赖初始化处添加
<feature>Repo := mysqlRepo.New<Name>Repo(db, logger)
<feature>Svc := service.New<Name>Service(<feature>Repo, logger)
<feature>Handler := handler.New<Name>Handler(<feature>Svc, logger)

// 在路由注册处添加
<feature>Group := api.Group("/<feature>")
{
    <feature>Group.GET("/:id", <feature>Handler.Get)
    <feature>Group.POST("", <feature>Handler.Create)
}
```

---

## 第四步：前端实现

### 文件清单

在 `frontend/src/` 下创建：

```
src/
├── modules/<feature>/
│   ├── <Name>View.vue             # 主视图组件
│   ├── <Name>Panel.vue            # 子面板（可选）
│   └── components/                # 玩法专用子组件（可选）
├── core/store/<feature>.ts        # Pinia Store
└── types/<feature>.ts             # TypeScript 类型定义
```

### Vue 组件规范

```vue
<script setup lang="ts">
/**
 * <Name>View - <玩法中文名>主视图
 *
 * 描述玩法的主要交互界面
 */
import { ref, onMounted } from 'vue'
import { use<Name>Store } from '@/core/store/<feature>'

const store = use<Name>Store()

onMounted(() => {
  store.fetchData()
})
</script>

<template>
  <div class="<feature>-view">
    <!-- 使用项目 SCSS 变量：$primary, $bg-dark, $text-light 等 -->
  </div>
</template>

<style lang="scss" scoped>
@use '@/styles/variables' as *;

.<feature>-view {
  // 响应式设计：同时考虑 PC 和移动端
}
</style>
```

**必须遵守**：
- 使用 `<script setup lang="ts">` 组合式 API
- 组件文件名使用 PascalCase
- CSS 使用 SCSS，引用 `styles/variables.scss` 中的变量
- 状态管理使用 Pinia，不在组件间直接 props 穿透超过两层
- 每个组件文件顶部必须有 JSDoc 注释

### Pinia Store 规范

```typescript
// src/core/store/<feature>.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/core/api'
import type { <Name>Data } from '@/types/<feature>'

/**
 * <玩法中文名>状态管理
 */
export const use<Name>Store = defineStore('<feature>', () => {
  const data = ref<<Name>Data | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isReady = computed(() => data.value !== null)

  async function fetchData(id: number) {
    loading.value = true
    error.value = null
    try {
      const res = await api.get(`/<feature>/${id}`)
      data.value = res.data
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, isReady, fetchData }
})
```

### 路由注册

在 `frontend/src/router/index.ts` 中添加：

```typescript
{
  path: '<feature>',
  name: '<Feature>',
  component: () => import('@/modules/<feature>/<Name>View.vue'),
  meta: { title: '<玩法中文名>', requiresAuth: true }
}
```

---

## 第五步：集成注册

完成以下项目级更新：

1. **更新 `backend/go.work`**：如果是新服务，添加到 `use` 块
2. **更新 CLAUDE.md**：
   - 服务拓扑图添加新服务/模块
   - 前端模块列表添加新模块
3. **更新 `deploy/k8s/`**：添加新服务的 Deployment 和 Service YAML
4. **更新 `docs/API.md`**：记录新增的 API 端点
5. **创建 MySQL 迁移编号**：取 `database/mysql/` 下最大编号 + 1
6. **运行测试**：`cd backend && go test ./...` 确保无破坏

## Commit 信息格式

```
feat(<scope>): 新增<玩法中文名>系统

- 后端: <service名>/internal/{handler,service,model,repository}
- 前端: modules/<feature>/<Name>View.vue + store/<feature>.ts
- 数据库: database/mysql/020_<feature>.sql
- Proto: 新增 <Name>Request/<Name>Response 消息

Closes #<issue号>
```

## 反模式（禁止）

- 不要跳过 Handler 直接在 main.go 中写业务逻辑
- 不要在 Vue 组件中直接调用 fetch，必须通过 Pinia Store
- 不要硬编码配置值，使用 config.go 和 env 变量
- 不要遗漏 MySQL 迁移文件的 COMMENT 和索引
- 不要忘记在 CLAUDE.md 中记录新增模块
