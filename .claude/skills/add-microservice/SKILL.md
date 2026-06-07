---
name: add-microservice
description: >
  Generate a new Go microservice skeleton for the 九转修仙 cultivation game project.
  Use this skill whenever the user asks to create a new backend service, add a microservice,
  scaffold a service, or mentions needing a new service for a feature like "创建新服务",
  "添加微服务", "新建服务", "scaffold service". Also use when the user describes a new
  backend domain that doesn't fit existing services (gateway, auth, player, combat,
  cultivation, social, world, trade, ranking).
---

# Add Microservice — 九转修仙项目

为九转修仙游戏项目生成标准的 Go 微服务骨架，严格遵循项目既有的 Handler → Service → Repository 三层架构。

## 前置要求

在开始之前，确认以下信息：
1. **服务名称** (小写英文，如 `auction`, `mail`, `activity`)
2. **服务中文名** (如 `拍卖服务`, `邮件服务`, `活动服务`)
3. **是否需要 MySQL** (大多数服务需要)
4. **是否需要 Redis** (排行榜/缓存类服务需要)
5. **监听端口** (参考 CLAUDE.md 中现有端口分配，避免冲突)

## 生成的文件结构

在 `services/<name>/` 下创建以下完整骨架：

```
services/<name>/
├── cmd/<name>/main.go              # 入口：加载配置、连接DB、启动Gin、优雅关闭
├── internal/
│   ├── config/config.go            # 服务配置结构体 + Load() 函数
│   ├── handler/<name>_handler.go   # HTTP handler (Gin 路由绑定、参数校验)
│   ├── service/<name>_service.go   # 业务逻辑层
│   ├── model/<name>.go             # 数据模型 / 领域类型
│   └── repository/
│       ├── mysql/<name>_repo.go    # MySQL 数据访问 (database/sql 或 GORM)
│       └── redis/<name>_redis.go   # Redis 数据访问 (可选)
├── go.mod                          # module cultivation-game/services/<name>
├── go.sum                          # go mod tidy 自动生成
└── Dockerfile                      # 多阶段构建：golang:1.22-alpine → alpine:3.19
```

### 额外操作

生成文件后，还需：
1. 将新服务添加到 `backend/go.work` 的 `use` 块中
2. 在 `deploy/k8s/` 下创建 `<name>-deployment.yaml` 和 `<name>-service.yaml`
3. 在 `deploy/docker-compose.yml` 中添加服务定义（如适用）

## 代码模板规范

### main.go 模板

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "cultivation-game/services/<name>/internal/config"
    "cultivation-game/services/<name>/internal/handler"
    // 根据需要引入 repository 和 service 包

    "github.com/gin-gonic/gin"
    _ "github.com/go-sql-driver/mysql"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

func main() {
    cfg := config.Load()

    logger, err := zap.NewProduction()
    if err != nil {
        log.Fatalf("初始化日志失败: %v", err)
    }
    defer logger.Sync()

    // MySQL 连接 (如果需要)
    db, err := sql.Open("mysql", cfg.MySQL.DSN)
    if err != nil {
        logger.Fatal("打开数据库连接失败", zap.Error(err))
    }
    defer db.Close()
    db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.MySQL.ConnMaxLifetime)

    if err := db.Ping(); err != nil {
        logger.Fatal("Ping 数据库失败", zap.Error(err))
    }
    logger.Info("MySQL 连接成功")

    // Redis 连接 (如果需要)
    rdb := redis.NewClient(&redis.Options{
        Addr:     cfg.Redis.Addr,
        Password: cfg.Redis.Password,
        DB:       cfg.Redis.DB,
    })
    defer rdb.Close()

    if _, err := rdb.Ping(context.Background()).Result(); err != nil {
        logger.Fatal("Redis 连接失败", zap.Error(err))
    }
    logger.Info("Redis 连接成功")

    // 初始化依赖链: Repository → Service → Handler
    // repo := mysql.New<Name>Repo(db, logger)
    // svc := service.New<Name>Service(repo, logger)
    // h := handler.New<Name>Handler(svc, logger)

    // 路由
    gin.SetMode(gin.ReleaseMode)
    r := gin.Default()
    api := r.Group("/api/v1")
    {
        // api.GET("/<name>/:id", h.Get)
        // api.POST("/<name>", h.Create)
    }

    // HTTP 服务器
    srv := &http.Server{
        Addr:         cfg.Server.Addr,
        Handler:      r,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
    }

    go func() {
        logger.Info("服务启动", zap.String("addr", cfg.Server.Addr))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("服务启动失败", zap.Error(err))
        }
    }()

    // 优雅关闭
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    logger.Info("正在关闭服务...")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        logger.Fatal("服务关闭异常", zap.Error(err))
    }
    logger.Info("服务已关闭")
}
```

### config.go 模板

```go
package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    Server ServerConfig
    MySQL  MySQLConfig
    Redis  RedisConfig
    Log    LogConfig
}

type ServerConfig struct {
    Addr         string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
}

type MySQLConfig struct {
    DSN             string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}

type RedisConfig struct {
    Addr     string
    Password string
    DB       int
}

type LogConfig struct {
    Level string
}

func Load() *Config {
    return &Config{
        Server: ServerConfig{
            Addr:         getEnv("SERVER_ADDR", ":8080"),
            ReadTimeout:  time.Duration(getEnvInt("SERVER_READ_TIMEOUT", 30)) * time.Second,
            WriteTimeout: time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT", 30)) * time.Second,
        },
        MySQL: MySQLConfig{
            DSN:             getEnv("MYSQL_DSN", "root:root@tcp(localhost:3306)/cultivation?charset=utf8mb4&parseTime=True"),
            MaxOpenConns:    getEnvInt("MYSQL_MAX_OPEN_CONNS", 25),
            MaxIdleConns:    getEnvInt("MYSQL_MAX_IDLE_CONNS", 10),
            ConnMaxLifetime: time.Duration(getEnvInt("MYSQL_CONN_MAX_LIFETIME", 300)) * time.Second,
        },
        Redis: RedisConfig{
            Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
            Password: getEnv("REDIS_PASSWORD", ""),
            DB:       getEnvInt("REDIS_DB", 0),
        },
    }
}

func getEnv(key, defaultVal string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
    if v := os.Getenv(key); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    return defaultVal
}
```

### handler 模板

```go
package handler

import (
    "net/http"
    "strconv"

    "cultivation-game/services/<name>/internal/service"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type <Name>Handler struct {
    svc *service.<Name>Service
    log *zap.Logger
}

func New<Name>Handler(svc *service.<Name>Service, log *zap.Logger) *<Name>Handler {
    return &<Name>Handler{svc: svc, log: log}
}

// Get 获取单条记录
// GET /api/v1/<name>/:id
func (h *<Name>Handler) Get(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的ID"})
        return
    }

    result, err := h.svc.Get(c.Request.Context(), id)
    if err != nil {
        h.log.Error("查询失败", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": result})
}

// Create 创建新记录
// POST /api/v1/<name>
func (h *<Name>Handler) Create(c *gin.Context) {
    var req model.Create<Name>Request
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
        return
    }

    result, err := h.svc.Create(c.Request.Context(), &req)
    if err != nil {
        h.log.Error("创建失败", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建失败"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": result})
}
```

**命名约定**：将 `<name>` 替换为服务名（小写），将 `<Name>` 替换为 PascalCase 服务名。

### service 模板

```go
package service

import (
    "context"
    "fmt"

    "cultivation-game/services/<name>/internal/model"
    mysqlRepo "cultivation-game/services/<name>/internal/repository/mysql"

    "go.uber.org/zap"
)

type <Name>Service struct {
    repo *mysqlRepo.<Name>Repo
    log  *zap.Logger
}

func New<Name>Service(repo *mysqlRepo.<Name>Repo, log *zap.Logger) *<Name>Service {
    return &<Name>Service{repo: repo, log: log}
}

func (s *<Name>Service) Get(ctx context.Context, id uint64) (*model.<Name>, error) {
    result, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("查询<中文名>失败: %w", err)
    }
    return result, nil
}

func (s *<Name>Service) Create(ctx context.Context, req *model.Create<Name>Request) (*model.<Name>, error) {
    // 业务校验逻辑
    // ...
    result, err := s.repo.Create(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("创建<中文名>失败: %w", err)
    }
    return result, nil
}
```

### model 模板

```go
package model

import "time"

type <Name> struct {
    ID        uint64    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Create<Name>Request struct {
    Name string `json:"name" binding:"required"`
}
```

### repository (MySQL) 模板

```go
package mysql

import (
    "database/sql"
    "fmt"

    "cultivation-game/services/<name>/internal/model"

    "go.uber.org/zap"
)

type <Name>Repo struct {
    db  *sql.DB
    log *zap.Logger
}

func New<Name>Repo(db *sql.DB, log *zap.Logger) *<Name>Repo {
    return &<Name>Repo{db: db, log: log}
}

func (r *<Name>Repo) GetByID(ctx context.Context, id uint64) (*model.<Name>, error) {
    query := `SELECT id, name, created_at, updated_at FROM <table_name> WHERE id = ?`
    row := r.db.QueryRowContext(ctx, query, id)
    var m model.<Name>
    if err := row.Scan(&m.ID, &m.Name, &m.CreatedAt, &m.UpdatedAt); err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("记录不存在 id=%d", id)
        }
        return nil, fmt.Errorf("查询失败: %w", err)
    }
    return &m, nil
}

func (r *<Name>Repo) Create(ctx context.Context, req *model.Create<Name>Request) (*model.<Name>, error) {
    query := `INSERT INTO <table_name> (name) VALUES (?)`
    result, err := r.db.ExecContext(ctx, query, req.Name)
    if err != nil {
        return nil, fmt.Errorf("插入失败: %w", err)
    }
    id, _ := result.LastInsertId()
    return &model.<Name>{ID: uint64(id), Name: req.Name}, nil
}
```

### go.mod 模板

```
module cultivation-game/services/<name>

go 1.22

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/go-sql-driver/mysql v1.8.1
    github.com/redis/go-redis/v9 v9.5.1
    go.uber.org/zap v1.27.0
)
```

依赖版本号要与项目中其他服务的 go.mod 保持一致。生成后执行 `cd backend && go mod tidy -C ../services/<name>` 来自动补全间接依赖。

### Dockerfile 模板

```dockerfile
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY . .
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/<name>-server ./cmd/<name>/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/<name>-server .
EXPOSE 8080
ENTRYPOINT ["/app/<name>-server"]
```

## 工作流程

1. **询问服务名称、中文名和功能描述**
2. **确认是否需要 MySQL / Redis**
3. **分配端口号**（检查 CLAUDE.md 中已有服务端口，避免冲突；推荐从 8090 开始递增）
4. **创建完整目录结构**
5. **使用上述模板生成所有文件**，将 `<name>` / `<Name>` 替换为实际服务名
6. **更新 `backend/go.work`**，在 `use` 块中添加 `../services/<name>`
7. **运行 `cd backend && go mod tidy`** 为新服务拉取依赖
8. **创建 K8s 部署文件** (`deploy/k8s/<name>-deployment.yaml`, `deploy/k8s/<name>-service.yaml`)
9. **更新 CLAUDE.md** 中的服务拓扑图，添加新服务
10. **输出摘要**：列出创建的文件和新服务的访问地址

## 常见变体

- **无状态服务** (如 gateway)：跳过 MySQL/Redis 相关代码
- **纯计算服务** (如 ranking 部分功能)：使用 Redis 代替 MySQL
- **含静态数据的服务** (如 combat 怪物配置)：创建 `internal/data/` 目录存放 JSON 配置文件

每次生成后提醒用户检查端口分配和服务间依赖关系是否与 CLAUDE.md 中的架构图一致。
