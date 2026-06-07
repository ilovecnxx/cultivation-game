# 贡献指南

感谢你参与九转修仙项目的开发。请遵循以下规范以确保代码质量和协作效率。

## PR 流程

1. **创建 Issue**：在开始工作前，先创建 Issue 描述你要解决的问题或新增的功能。
2. **分支策略**：
   - 从 `main` 分支创建功能分支：`feature/issue-xxx-short-desc`
   - 修复分支：`fix/issue-xxx-short-desc`
3. **本地开发**：在分支上完成开发，确保所有测试通过。
4. **提交 PR**：
   - PR 标题应简要描述变更内容。
   - PR 描述应关联 Issue（`Closes #xxx`），并说明改动点和测试方式。
5. **Code Review**：至少一位维护者审核通过后方可合并。
6. **合并**：使用 Squash Merge 将分支合并到 `main`。

## 代码规范

### Go

- 使用 `go fmt` / `gofumpt` 格式化代码。
- 遵循 [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)。
- 错误处理：不要忽略错误，使用 `%w` 包装错误以保留上下文。
- 日志：使用 `slog` 或 `zerolog` 结构化日志，避免 `fmt.Println`。
- 命名：包名小写单数，导出类型使用驼峰。

### TypeScript / Vue

- 使用 `vue-tsc` 进行类型检查，确保无类型错误。
- 组件使用 `<script setup lang="ts">` 组合式 API。
- 文件名使用 PascalCase（组件）和 camelCase（工具函数）。
- CSS 使用 SCSS，变量定义在 `styles/variables.scss` 中。
- 状态管理使用 Pinia，避免组件间直接 props 穿透超过两层。

### 通用

- 新增文件必须包含包注释或文件头注释。
- 公共 API 必须有注释，说明参数和返回值。
- 禁止提交 `node_modules/`、`.env`、编译产物。
- 配置和环境变量遵循 `config.go` 中定义的结构，禁止硬编码。

## Commit Message 规范

提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <description>

[optional body]
[optional footer]
```

### Type 类型

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | 修复 Bug |
| `refactor` | 重构（既不修复 Bug 也不添加功能） |
| `perf` | 性能优化 |
| `test` | 添加或修改测试 |
| `docs` | 文档变更 |
| `style` | 代码格式（不改变逻辑） |
| `chore` | 构建、CI、依赖等杂项 |
| `ci` | CI 配置变更 |

### Scope 范围

| Scope | 说明 |
|-------|------|
| `gateway` | 网关服务 |
| `player` | 玩家服务 |
| `combat` | 战斗服务 |
| `cultivation` | 修炼服务 |
| `social` | 社交服务 |
| `world` | 世界服务 |
| `frontend` | 前端 |
| `shared` | 共享库 |
| `db` | 数据库 |

### 示例

```
feat(combat): 新增五行相克计算逻辑

- 金克木、木克土、土克水、水克火、火克金
- 克制时伤害 +30%，被克制时伤害 -20%

Closes #42
```

```
fix(player): 修复装备强化后属性未及时刷新

装备强化后未清除缓存导致玩家面板属性显示错误。
现已在强化接口中主动失效 Redis 缓存。

Closes #87
```

```
docs(readme): 更新快速启动命令
```
