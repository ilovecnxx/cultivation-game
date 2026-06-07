# 凡人修仙模拟器 — 部署运维文档

> 版本: 2.0 | 最后更新: 2026-06-05  
> 本文档覆盖开发环境 Docker Compose 部署、生产环境 K8s 部署、数据库初始化、Nginx 配置、监控部署、告警规则、备份恢复、灰度发布与回滚方案。

---

## 目录

1. [环境要求](#1-环境要求)
2. [开发环境部署](#2-开发环境部署)
3. [生产环境部署 (K8s)](#3-生产环境部署-k8s)
4. [数据库初始化与迁移](#4-数据库初始化与迁移)
5. [Nginx 配置与 SSL](#5-nginx-配置与-ssl)
6. [监控部署](#6-监控部署)
7. [告警规则](#7-告警规则)
8. [备份与恢复](#8-备份与恢复)
9. [灰度发布流程](#9-灰度发布流程)
10. [回滚方案](#10-回滚方案)

---

## 1. 环境要求

### 1.1 硬件要求

| 环境 | CPU | 内存 | 磁盘 | 网络 |
|------|-----|------|------|------|
| 开发环境 | 2 core+ | 4 GB+ | 50 GB+ (含数据) | 内网 |
| 生产环境 (最小) | 4 core+ | 8 GB+ | 200 GB+ (SSD) | 公网带宽 100Mbps+ |
| 生产环境 (推荐) | 8 core+ | 16 GB+ | 500 GB+ (SSD RAID10) | 公网带宽 500Mbps+ |

### 1.2 软件要求

| 组件 | 开发环境 | 生产环境 |
|------|---------|---------|
| 操作系统 | macOS / Linux / WSL2 | Linux (Ubuntu 22.04+/CentOS 8+) |
| Docker | 24+ | 24+ |
| Docker Compose | v2+ | - |
| Kubernetes | - | 1.28+ (min 3 节点) |
| Helm | - | 3.14+ |
| Go | 1.22+ | - |
| Node.js | 20+ | - |
| MySQL Client | 8.0+ | 8.0+ |
| Redis CLI | 7+ | 7+ |
| MongoDB Shell | 7+ | 7+ |
| kubectl | - | 1.28+ |

### 1.3 网络端口要求 (生产环境)

| 端口 | 协议 | 用途 | 开放范围 |
|------|------|------|---------|
| 80 | TCP | HTTP (重定向到 HTTPS) | 0.0.0.0/0 |
| 443 | TCP | HTTPS / WSS | 0.0.0.0/0 |
| 22 | TCP | SSH (管理) | 堡垒机 IP 白名单 |
| 9090 | TCP | Prometheus | 内网 / 监控节点 |
| 3000 | TCP | Grafana | 内网 / VPN |
| 3306 | TCP | MySQL | 内网 (仅服务) |
| 6379 | TCP | Redis | 内网 (仅服务) |
| 27017 | TCP | MongoDB | 内网 (仅服务) |

---

## 2. 开发环境部署

### 2.1 Docker Compose 一键启动

开发环境使用 `docker-compose.yml` 一键启动所有依赖和业务服务。

```bash
# 1. 克隆项目
git clone <repo-url> /root/cultivation-game
cd /root/cultivation-game/deploy

# 2. 复制环境变量模板并配置
cp .env.example .env
# 编辑 .env 文件，设置必要密钥：
#   MYSQL_ROOT_PASSWORD=your_secure_password
#   JWT_SECRET=your_jwt_secret_key_here

# 3. 一键启动所有服务
docker compose -f docker-compose.yml up -d

# 4. 查看服务状态
docker compose -f docker-compose.yml ps

# 5. 查看日志
docker compose -f docker-compose.yml logs -f gateway
```

### 2.2 Docker Compose 服务清单

| 服务名 | 镜像 | 端口映射 | 依赖 | 说明 |
|--------|------|---------|------|------|
| gateway | cultivation-game/gateway | 8080:8080, 8081:8081 | redis, mysql | WebSocket + HTTP 网关 |
| auth | cultivation-game/auth | 8082:8082 | redis, mysql | 认证服务 |
| player | cultivation-game/player | 8083:8083 | redis, mysql | 玩家服务 |
| cultivation | cultivation-game/cultivation | 8084:8084 | redis, mysql | 修炼服务 |
| combat | cultivation-game/combat | 8085:8085 | redis, mysql | 战斗服务 |
| social | cultivation-game/social | 8086:8086 | redis, mysql, mongo | 社交服务 |
| world | cultivation-game/world | 8087:8087 | redis, mysql | 世界服务 |
| nginx | nginx:1.25-alpine | 80:80, 443:443 | gateway | 反向代理 |
| redis | redis:7-alpine | 6379:6379 | - | 缓存/会话 |
| mysql | mysql:8.0 | 3306:3306 | - | 主数据库 |
| mongo | mongo:7 | 27017:27017 | - | 文档数据库 |

### 2.3 环境变量配置

| 变量名 | 说明 | 默认值 | 必填 |
|--------|------|--------|------|
| GAME_ENV | 环境标识 | development | 否 |
| IMAGE_TAG | 镜像标签 | latest | 否 |
| JWT_SECRET | JWT 签名密钥 | - | **是** |
| MYSQL_ROOT_PASSWORD | MySQL root 密码 | - | **是** |
| MYSQL_DATABASE | 数据库名 | cultivation_game | 否 |
| REDIS_PORT | Redis 端口 | 6379 | 否 |
| NGINX_HTTP_PORT | Nginx HTTP 端口 | 80 | 否 |
| NGINX_HTTPS_PORT | Nginx HTTPS 端口 | 443 | 否 |
| GATEWAY_WS_PORT | Gateway WS 端口 | 8080 | 否 |
| GATEWAY_HTTP_PORT | Gateway HTTP 端口 | 8081 | 否 |

### 2.4 健康检查验证

```bash
# 检查所有服务健康状态
curl http://localhost:80/health                   # Nginx
curl http://localhost:8081/health                 # Gateway
curl http://localhost:8082/health                 # Auth
curl http://localhost:8083/health                 # Player
curl http://localhost:8084/health                 # Cultivation
curl http://localhost:8085/health                 # Combat
curl http://localhost:8086/health                 # Social
curl http://localhost:8087/health                 # World

# 检查数据库
docker compose -f docker-compose.yml exec mysql mysqladmin ping -uroot -p${MYSQL_ROOT_PASSWORD}
docker compose -f docker-compose.yml exec redis redis-cli ping
docker compose -f docker-compose.yml exec mongo mongosh --eval "db.runCommand('ping')"

# 检查 WebSocket 连接 (需要 websocat 或 wscat)
npm install -g wscat
wscat -c ws://localhost:8080/ws?token=test_token
```

### 2.5 单独启动单个服务 (无 Docker)

```bash
# 1. 启动基础设施 (数据库)
docker compose -f docker-compose.yml up -d redis mysql mongo

# 2. 安装 Go 依赖并启动服务
cd ../services/gateway
go mod tidy
go run . --port 8080 --config config.json

# 3. 另一个终端启动前端
cd ../../frontend
npm install
npm run dev
```

---

## 3. 生产环境部署 (K8s)

### 3.1 K8s 集群要求

| 资源 | 要求 | 说明 |
|------|------|------|
| 集群版本 | 1.28+ | 推荐使用 EKS / AKS / GKE |
| 节点数 | 3+ | 高可用需要 3 节点以上 |
| 节点规格 | 4C8G+ | 游戏业务对实时性要求高 |
| Ingress Controller | nginx-ingress | WebSocket 支持 |
| StorageClass | 支持 ReadWriteOnce | 数据库持久卷 |
| Metrics Server | 部署 | HPA 自动伸缩依赖 |

### 3.2 K8s 部署文件清单 (25 个 YAML)

所有 K8s 部署文件位于 `/root/cultivation-game/deploy/k8s/`：

```
deploy/k8s/
├── namespace.yaml              # 命名空间: cultivation-game
├── configmap.yaml              # 通用配置 (非敏感)
├── hpa.yaml                    # HPA 自动伸缩 (7 个服务的 HPA)
├── ingress.yaml                # Ingress 规则 (WS + API + Static)
├── kustomization.yaml          # Kustomize 入口
│
├── gateway-deployment.yaml     # Gateway 部署
├── gateway-service.yaml        # Gateway Service
├── auth-deployment.yaml        # Auth 部署
├── auth-service.yaml           # Auth Service
├── player-deployment.yaml      # Player 部署
├── player-service.yaml         # Player Service
├── cultivation-deployment.yaml # Cultivation 部署
├── cultivation-service.yaml    # Cultivation Service
├── combat-deployment.yaml      # Combat 部署
├── combat-service.yaml         # Combat Service
├── social-deployment.yaml      # Social 部署
├── social-service.yaml         # Social Service
├── world-deployment.yaml       # World 部署
├── world-service.yaml          # World Service
│
├── mysql-statefulset.yaml      # MySQL StatefulSet
├── mysql-service.yaml          # MySQL Service
├── redis-deployment.yaml       # Redis 部署
├── redis-service.yaml          # Redis Service
├── mongo-deployment.yaml       # MongoDB 部署
└── mongo-service.yaml          # MongoDB Service
```

### 3.3 部署步骤

```bash
# 1. 创建命名空间
kubectl apply -f deploy/k8s/namespace.yaml

# 2. 创建 ConfigMap (非敏感配置)
kubectl apply -f deploy/k8s/configmap.yaml

# 3. 创建 Secrets (敏感信息)
kubectl create secret generic game-secrets \
  --namespace=cultivation-game \
  --from-literal=jwt-secret='your-jwt-secret' \
  --from-literal=mysql-root-password='your-mysql-password' \
  --from-literal=redis-password='your-redis-password'

# 4. 部署数据库 (有状态服务)
kubectl apply -f deploy/k8s/mysql-statefulset.yaml
kubectl apply -f deploy/k8s/mysql-service.yaml
kubectl apply -f deploy/k8s/redis-deployment.yaml
kubectl apply -f deploy/k8s/redis-service.yaml
kubectl apply -f deploy/k8s/mongo-deployment.yaml
kubectl apply -f deploy/k8s/mongo-service.yaml

# 5. 等待数据库就绪
kubectl wait --for=condition=ready pod -l app=mysql -n cultivation-game --timeout=300s
kubectl wait --for=condition=ready pod -l app=redis -n cultivation-game --timeout=120s

# 6. 初始化数据库 (见 4.1 节)
# ... 导入 SQL 脚本 ...

# 7. 部署无状态业务服务
kubectl apply -f deploy/k8s/gateway-deployment.yaml
kubectl apply -f deploy/k8s/gateway-service.yaml
kubectl apply -f deploy/k8s/auth-deployment.yaml
kubectl apply -f deploy/k8s/auth-service.yaml
kubectl apply -f deploy/k8s/player-deployment.yaml
kubectl apply -f deploy/k8s/player-service.yaml
kubectl apply -f deploy/k8s/cultivation-deployment.yaml
kubectl apply -f deploy/k8s/cultivation-service.yaml
kubectl apply -f deploy/k8s/combat-deployment.yaml
kubectl apply -f deploy/k8s/combat-service.yaml
kubectl apply -f deploy/k8s/social-deployment.yaml
kubectl apply -f deploy/k8s/social-service.yaml
kubectl apply -f deploy/k8s/world-deployment.yaml
kubectl apply -f deploy/k8s/world-service.yaml

# 8. 部署 HPA (自动伸缩)
kubectl apply -f deploy/k8s/hpa.yaml

# 9. 部署 Ingress
kubectl apply -f deploy/k8s/ingress.yaml

# 10. 验证全部就绪
kubectl get all -n cultivation-game
kubectl get hpa -n cultivation-game
kubectl get ingress -n cultivation-game
```

### 3.4 使用 Kustomize 一键部署

```bash
# 如果所有镜像已构建并推送，可一键部署
kubectl apply -k deploy/k8s/
```

### 3.5 镜像构建与推送

```bash
# 设置镜像仓库
REGISTRY=your-registry.com/cultivation-game
TAG=$(git rev-parse --short HEAD)

# 构建所有服务镜像 (使用 Docker Buildx 并行构建)
for service in gateway auth player cultivation combat social world; do
  docker build \
    -t ${REGISTRY}/${service}:${TAG} \
    -t ${REGISTRY}/${service}:latest \
    -f services/${service}/Dockerfile \
    .
done

# 前端构建
docker build \
  -t ${REGISTRY}/frontend:${TAG} \
  -t ${REGISTRY}/frontend:latest \
  -f frontend/Dockerfile \
  frontend/

# 推送镜像
for image in gateway auth player cultivation combat social world frontend; do
  docker push ${REGISTRY}/${image}:${TAG}
  docker push ${REGISTRY}/${image}:latest
done
```

### 3.6 Ingress 配置说明

Ingress 配置在 `deploy/k8s/ingress.yaml` 中，包含三条路由规则：

| 路径前缀 | 后端服务 | 端口 | 说明 |
|---------|---------|------|------|
| /ws | gateway | 8080 | WebSocket 连接 (支持 WS 协议升级) |
| /api | gateway | 8081 | RESTful API |
| / | frontend | 80 | 前端静态资源 (Vue 构建产物) |

---

## 4. 数据库初始化与迁移

### 4.1 MySQL 初始化

```bash
# 方法 1: Docker Compose 自动初始化 (推荐)
# MySQL 启动时自动执行 /docker-entrypoint-initdb.d/ 下的 SQL 文件
# 挂载目录: ./mysql/init:/docker-entrypoint-initdb.d
# 执行顺序按文件名排序

# 方法 2: 手动导入
# 按以下顺序执行 SQL 脚本:
mysql -h <host> -u root -p < deploy/mysql/001_init.sql       # 核心表结构 (18 张)
mysql -h <host> -u root -p < deploy/mysql/001_cultivation.sql # 修炼系统表 (2 张)
mysql -h <host> -u root -p < deploy/mysql/003_trade.sql      # 交易系统表 (4 张)
mysql -h <host> -u root -p < deploy/mysql/005_game_loop.sql  # 游戏循环表 (3 张)
mysql -h <host> -u root -p < deploy/mysql/006_luck.sql       # 气运系统表 (2 张)
mysql -h <host> -u root -p < database/mysql/002_seed.sql     # 种子数据

# 方法 3: 通过 K8s Job 初始化 (生产环境)
kubectl apply -f deploy/k8s/job-db-init.yaml
```

### 4.2 SQL 脚本说明

| 文件名 | 表数 | 说明 | 是否幂等 |
|--------|------|------|---------|
| 001_init.sql | 18 | 核心表: players, attributes, inventory, equipment, techniques, skills, friends, sects, sect_members, mail, trade_listings, pvp_rankings, admin_users, realm_config, item_templates, technique_templates, skill_templates, system_config | 是 (IF NOT EXISTS) |
| 001_cultivation.sql | 2 | 修炼系统: cultivation_players, cultivation_techniques | 是 |
| 003_trade.sql | 4 | 交易系统: trade_listings(v2), trade_transactions, trade_auctions, trade_player_gold | 是 |
| 005_game_loop.sql | 3 | 游戏循环: player_buffs, world_events, daily_resets | 是 |
| 006_luck.sql | 2 | 气运系统: player_luck, luck_events | 是 |
| 002_seed.sql | - | 种子数据: 37 条境界配置 + 26 条物品 + 4 种功法 + 10 种技能 + 15 条系统配置 + 5 个示例玩家 | 否 (INSERT, 首次执行) |

### 4.3 MongoDB 初始化

```javascript
// database/mongodb/schema_examples.js
// 创建用户和数据库
use cultivation_game;
db.createUser({
  user: "game_service",
  pwd: "your_secure_password",
  roles: [{ role: "readWrite", db: "cultivation_game" }]
});

// 创建集合和索引
db.createCollection("chat_messages");
db.chat_messages.createIndex({ channel: 1, timestamp: -1 });
db.chat_messages.createIndex({ sender_id: 1 });

db.createCollection("battle_replays");
db.battle_replays.createIndex({ player_id: 1, timestamp: -1 });

db.createCollection("event_logs");
db.event_logs.createIndex({ player_id: 1, timestamp: -1 });
db.event_logs.createIndex({ event_type: 1 });

db.createCollection("trade_history");
db.trade_history.createIndex({ seller_id: 1 });
db.trade_history.createIndex({ buyer_id: 1 });
```

### 4.4 数据库迁移策略

| 场景 | 策略 | 步骤 |
|------|------|------|
| 新增表 | 直接执行 CREATE TABLE | 在对应版本 SQL 文件中追加，带 IF NOT EXISTS |
| 新增字段 | ALTER TABLE ADD COLUMN | 新增迁移文件，如 `004_add_buff_columns.sql` |
| 修改字段 | ALTER TABLE MODIFY COLUMN | 需要评估向下兼容性 |
| 删除字段 | 标记废弃，下版本删除 | 先停用代码引用，再删除列 |
| 新增索引 | CREATE INDEX | CREATE INDEX IF NOT EXISTS (MySQL 8.0 支持) |

**迁移版本控制：**
```
database/mysql/
├── 001_init.sql           # v1.0 初始表结构
├── 001_cultivation.sql    # v1.0 修炼系统
├── 002_seed.sql           # v1.0 种子数据
├── 003_trade.sql          # v2.0 交易系统
├── 004_ (预留)
├── 005_game_loop.sql      # v2.0 游戏循环
├── 006_luck.sql           # v2.0 气运系统
└── MIGRATIONS.md          # 迁移历史记录
```

---

## 5. Nginx 配置与 SSL

### 5.1 Nginx 配置 (生产环境推荐)

Nginx 配置位于 `/root/cultivation-game/deploy/nginx/`。

```nginx
# /etc/nginx/nginx.conf
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 4096;      # 单 worker 最大连接数 (65535/worker_processes)
    multi_accept on;
    use epoll;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # 日志格式
    log_format main '$remote_addr - $remote_user [$time_local] '
                    '"$request" $status $body_bytes_sent '
                    '"$http_referer" "$http_user_agent" '
                    '$request_time $upstream_response_time';

    access_log /var/log/nginx/access.log main buffer=32k flush=5s;

    # 基础优化
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    keepalive_requests 1000;
    client_max_body_size 10m;

    # Gzip 压缩
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 4;
    gzip_min_length 1024;
    gzip_types text/plain text/css application/json application/javascript
               image/svg+xml application/protobuf;

    # 限流
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=100r/s;
    limit_conn_zone $binary_remote_addr zone=conn_limit:10m;
    limit_conn_status 429;

    include /etc/nginx/conf.d/*.conf;
}

# WebSocket 支持的 map
map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
}
```

```nginx
# /etc/nginx/conf.d/cultivation-game.conf
server {
    listen 80;
    server_name cultivation-game.local;
    return 301 https://$server_name$request_uri;  # HTTP -> HTTPS 重定向
}

server {
    listen 443 ssl http2;
    server_name cultivation-game.local;

    # SSL 配置
    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # HSTS
    add_header Strict-Transport-Security "max-age=31536000" always;

    # 安全头
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";

    # ---- WebSocket 代理 ----
    location /ws {
        proxy_pass http://gateway:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 3600s;     # 长连接超时 1 小时
        proxy_send_timeout 3600s;
        proxy_buffering off;          # WS 不需要缓冲
    }

    # ---- RESTful API ----
    location /api/ {
        proxy_pass http://gateway:8081/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        limit_req zone=api_limit burst=200 nodelay;  # API 限流
    }

    # ---- 前端静态资源 ----
    location / {
        root /usr/share/nginx/html;
        index index.html;
        try_files $uri $uri/ /index.html;  # SPA 路由支持

        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff2?|ttf|eot)$ {
            expires 365d;
            add_header Cache-Control "public, immutable";
            access_log off;
        }
    }
}
```

### 5.2 SSL 证书 (Let's Encrypt)

```bash
# 安装 certbot
apt-get install -y certbot python3-certbot-nginx

# 申请证书 (自动配置 Nginx)
certbot --nginx -d cultivation-game.local --non-interactive --agree-tos -m admin@example.com

# 查看证书到期时间
certbot certificates

# 设置自动续期 (默认已添加 systemd timer)
certbot renew --dry-run
```

### 5.3 SSL 配置 (Docker Compose 开发环境)

```yaml
# docker-compose.yml 中 nginx 配置
nginx:
  image: nginx:1.25-alpine
  volumes:
    - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
    - ./nginx/conf.d:/etc/nginx/conf.d:ro
    - ../frontend/dist:/usr/share/nginx/html:ro
    - ./ssl:/etc/nginx/ssl:ro              # 本地 SSL 证书目录
  ports:
    - "80:80"
    - "443:443"
```

生成自签名证书（开发环境）：

```bash
mkdir -p deploy/ssl
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout deploy/ssl/privkey.pem \
  -out deploy/ssl/fullchain.pem \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=Dev/CN=cultivation-game.local"
```

---

## 6. 监控部署

### 6.1 Prometheus + Grafana + Loki 架构

```
┌─────────────────┐
│   各业务服务     │  (暴露 /metrics 端点, 端口 9090)
│  gateway:9090   │
│  auth:9090      │
│  player:9090    │
│  cultivation:   │
│    9090         │
│  combat:9090    │
│  social:9090    │
│  world:9090     │
└────────┬────────┘
         │ 拉取 (scrape)
┌────────▼────────┐    ┌────────────┐
│  Prometheus     │    │  Alertmanager│
│  scrape_interval│    │  P1/P2/P3  │
│  = 15s          │    │  告警通知   │
└────────┬────────┘    └────────────┘
         │ 查询
┌────────▼────────┐    ┌────────────┐
│  Grafana        │◄───│  Loki      │
│  仪表盘 / 告警   │    │  日志聚合   │
│  Panel / 图表   │    │  logql     │
└─────────────────┘    └─────┬──────┘
                             │ 推送
                      ┌──────▼──────┐
                      │  Promtail   │
                      │  日志采集   │
                      │  /var/log   │
                      └─────────────┘
```

### 6.2 启动监控栈

```bash
# Docker Compose 方式 (开发环境)
cd /root/cultivation-game/monitoring
docker compose -f docker-compose.monitor.yml up -d

# 访问地址:
#   Prometheus: http://localhost:9090
#   Grafana:    http://localhost:3000 (admin/admin)
#   Loki:       http://localhost:3100
```

### 6.3 Prometheus 配置

Prometheus 配置位于 `/root/cultivation-game/monitoring/prometheus/prometheus.yml`：

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alerts.yml"           # 告警规则

scrape_configs:
  - job_name: 'game-gateway'
    static_configs:
      - targets: ['gateway:9090']

  - job_name: 'game-services'
    static_configs:
      - targets:
          - 'auth:9090'
          - 'player:9090'
          - 'cultivation:9090'
          - 'combat:9090'
          - 'social:9090'
          - 'world:9090'

  - job_name: 'node'       # 节点指标 (CPU/内存/磁盘)
    static_configs:
      - targets: ['node-exporter:9100']

  - job_name: 'redis'      # Redis 指标
    static_configs:
      - targets: ['redis-exporter:9121']

  - job_name: 'mysql'      # MySQL 指标
    static_configs:
      - targets: ['mysql-exporter:9104']
```

### 6.4 暴露的 Prometheus 指标

每个 Go 微服务在 9090 端口暴露以下指标：

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `game_online_players_total` | Gauge | service | 当前在线玩家数 |
| `game_requests_total` | Counter | service, method, status | 请求总数 (含 status=success/error) |
| `game_request_duration_seconds` | Histogram | service, method | 请求延迟分布 |
| `game_message_duration_seconds` | Histogram | msg_id | 消息处理延迟 |
| `game_ws_connections` | Gauge | service | WebSocket 连接数 |
| `game_db_query_duration_seconds` | Histogram | db, query | 数据库查询延迟 |
| `game_cache_hit_ratio` | Gauge | cache_type | 缓存命中率 |
| `game_combat_duration_seconds` | Histogram | - | 战斗计算耗时 |
| `go_goroutines` | Gauge | - | Goroutine 数量 |
| `go_memstats_alloc_bytes` | Gauge | - | Go 内存分配 |

### 6.5 Loki 日志聚合

Loki 配置位于 `/root/cultivation-game/monitoring/loki/loki-config.yml`：
Promtail 配置位于 `/root/cultivation-game/monitoring/promtail/promtail-config.yml`。

```yaml
# promtail-config.yml 关键配置
scrape_configs:
  - job_name: 'game-logs'
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        refresh_interval: 5s
    relabel_configs:
      - source_labels: ['__meta_docker_container_name']
        regex: '/(.*)'
        target_label: 'service'
      - source_labels: ['__meta_docker_container_log_stream']
        target_label: 'log_stream'
```

---

## 7. 告警规则

告警规则定义在 `/root/cultivation-game/monitoring/prometheus/alerts.yml`，分 P1/P2/P3 三级。

### 7.1 P1 告警 (Critical - 立即处理)

| 告警名 | 表达式 | 持续时间 | 说明 |
|--------|--------|---------|------|
| **ServiceDown** | `up == 0` | 30s | 服务宕机，需立即排查 |
| **OnlinePlayerDrop** | 5min 平均 vs 30min 平均下降 > 30% | 1min | 在线玩家骤降，可能服务器故障 |

### 7.2 P2 告警 (Warning - 尽快处理)

| 告警名 | 表达式 | 阈值 | 持续时间 |
|--------|--------|------|---------|
| **HighMessageLatency** | histogram_quantile(0.99, rate(game_message_duration_seconds_bucket[5m])) | > 0.5s | 5min |
| **HighErrorRate** | sum(rate(game_requests_total{status="error"}[5m])) / sum(rate(game_requests_total[5m])) * 100 | > 1% | 5min |
| **RedisMemoryHigh** | redis_memory_used_bytes / redis_memory_max_bytes * 100 | > 80% | 2min |

### 7.3 P3 告警 (Info - 非紧急)

| 告警名 | 表达式 | 阈值 | 持续时间 |
|--------|--------|------|---------|
| **HighCPUUsage** | 100 - avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100 | > 80% | 5min |
| **HighDiskUsage** | (1 - node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes) * 100 | > 80% | 5min |

### 7.4 告警通知配置 (Alertmanager)

```yaml
# alertmanager.yml
route:
  receiver: 'default'
  routes:
    - match:
        severity: P1
      receiver: 'p1-urgent'
      repeat_interval: 5m
    - match:
        severity: P2
      receiver: 'p2-warning'
      repeat_interval: 30m
    - match:
        severity: P3
      receiver: 'p3-info'
      repeat_interval: 4h

receivers:
  - name: 'p1-urgent'
    webhook_configs:
      - url: 'https://hooks.example.com/alert/P1'
    slack_configs:
      - channel: '#game-p1'
        send_resolved: true

  - name: 'p2-warning'
    slack_configs:
      - channel: '#game-p2'
        send_resolved: true

  - name: 'p3-info'
    slack_configs:
      - channel: '#game-p3'
```

---

## 8. 备份与恢复

### 8.1 MySQL 备份

```bash
# ===== 全量备份 =====
# 手动备份
mysqldump -h <host> -u root -p \
  --databases cultivation_game \
  --single-transaction \
  --routines \
  --triggers \
  --events \
  --hex-blob \
  | gzip > /backup/mysql/cultivation_game_$(date +%Y%m%d_%H%M%S).sql.gz

# 自动备份 (crontab -e, 每天凌晨 3 点)
0 3 * * * /root/cultivation-game/scripts/backup-mysql.sh

# 备份保留策略:
#   最近 7 天: 每日全量
#   最近 4 周: 每周全量
#   超过 4 周: 每月全量
#   超过 6 月: 删除
```

```bash
# backup-mysql.sh 脚本内容
#!/bin/bash
BACKUP_DIR="/backup/mysql"
DB_NAME="cultivation_game"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=7
RETENTION_WEEKS=4
RETENTION_MONTHS=6

mkdir -p ${BACKUP_DIR}/{daily,weekly,monthly}

# 执行备份
mysqldump \
  -h ${MYSQL_HOST:-localhost} \
  -u ${MYSQL_USER:-root} \
  -p${MYSQL_PASSWORD} \
  --databases ${DB_NAME} \
  --single-transaction \
  --routines --triggers \
  --hex-blob \
  | gzip > ${BACKUP_DIR}/daily/${DB_NAME}_${DATE}.sql.gz

# 加密备份文件
gpg --symmetric --cipher-algo AES256 \
  --passphrase-file /etc/backup-passphrase \
  ${BACKUP_DIR}/daily/${DB_NAME}_${DATE}.sql.gz

# 清理过期备份
find ${BACKUP_DIR}/daily -name "*.gz.gpg" -mtime +${RETENTION_DAYS} -delete
find ${BACKUP_DIR}/weekly -name "*.gz.gpg" -mtime +$((RETENTION_WEEKS * 7)) -delete
find ${BACKUP_DIR}/monthly -name "*.gz.gpg" -mtime +$((RETENTION_MONTHS * 30)) -delete

# 每周备份提升为 weekly
if [ $(date +%u) -eq 7 ]; then
  cp ${BACKUP_DIR}/daily/${DB_NAME}_${DATE}.sql.gz.gpg ${BACKUP_DIR}/weekly/
fi

# 每月首日备份提升为 monthly
if [ $(date +%d) -eq 1 ]; then
  cp ${BACKUP_DIR}/daily/${DB_NAME}_${DATE}.sql.gz.gpg ${BACKUP_DIR}/monthly/
fi

echo "[$(date)] MySQL 备份完成: ${DB_NAME}_${DATE}.sql.gz.gpg"
```

### 8.2 MySQL 恢复

```bash
# 1. 停止业务服务 (防止写入)
kubectl scale deployment --replicas=0 -n cultivation-game gateway auth player cultivation combat social world trade ranking

# 2. 解密并解压备份
gpg --decrypt --passphrase-file /etc/backup-passphrase \
  /backup/mysql/daily/cultivation_game_20260605_030000.sql.gz.gpg \
  | gunzip > /tmp/restore.sql

# 3. 恢复数据库
mysql -h <host> -u root -p < /tmp/restore.sql

# 4. 恢复完成后重新启动服务
kubectl scale deployment --replicas=2 -n cultivation-game gateway auth player cultivation combat social world trade ranking

# 5. 验证数据完整性
mysql -h <host> -u root -p -e "SELECT COUNT(*) FROM players;" cultivation_game
mysql -h <host> -u root -p -e "SELECT COUNT(*) FROM inventory;" cultivation_game
```

### 8.3 Redis 备份

```bash
# Redis RDB 快照 (默认每 900s/300s/60s 自动保存)
# 配置文件: database/redis/redis.conf
#   save 900 1        # 至少 1 个 key 变化, 900s 保存
#   save 300 10       # 至少 10 个 key 变化, 300s 保存
#   save 60 10000     # 至少 10000 个 key 变化, 60s 保存

# 手动触发 RDB 保存
redis-cli SAVE

# 备份 RDB 文件
cp /data/redis/dump.rdb /backup/redis/dump_$(date +%Y%m%d_%H%M%S).rdb

# Redis AOF 持久化 (可选, 配置文件中设置 appendonly yes)
# AOF 文件: /data/redis/appendonly.aof
```

```bash
# Redis 恢复
# 1. 停止 Redis
# 2. 替换 dump.rdb 文件
cp /backup/redis/dump_20260605_030000.rdb /data/redis/dump.rdb
# 3. 启动 Redis
```

### 8.4 MongoDB 备份

```bash
# MongoDB 全量备份
mongodump \
  --host <host> \
  --port 27017 \
  --username admin \
  --password ${MONGO_PASSWORD} \
  --authenticationDatabase admin \
  --db cultivation_game \
  --out /backup/mongo/mongodump_$(date +%Y%m%d_%H%M%S)

# 压缩备份
tar -czf /backup/mongo/mongodump_$(date +%Y%m%d_%H%M%S).tar.gz \
  /backup/mongo/mongodump_$(date +%Y%m%d_%H%M%S)

# MongoDB 恢复
mongorestore \
  --host <host> \
  --username admin \
  --password ${MONGO_PASSWORD} \
  --authenticationDatabase admin \
  --db cultivation_game \
  /backup/mongo/mongodump_20260605_030000/cultivation_game
```

### 8.5 备份验证

```bash
# 定期验证备份完整性 (每周执行)
./scripts/verify-backups.sh

# verify-backups.sh 内容:
# 1. 从备份中恢复到一个临时数据库
# 2. 执行完整性检查 SQL
# 3. 记录验证结果
# 4. 清理临时数据库
```

---

## 9. 灰度发布流程

### 9.1 发布流程概述

灰度发布采用 **10% -> 50% -> 100%** 渐进式策略，逐阶段验证。

```
阶段 0: 代码审查 + CI 通过
    │
    ▼
阶段 1: 内部测试环境部署 (验证功能正确性)
    │
    ▼
阶段 2: 灰度 10% (金丝雀发布，监控错误率/延迟)
    │
    ▼
阶段 3: 灰度 50% (扩大流量，验证稳定性)
    │
    ▼
阶段 4: 全量 100% (完成发布，监控 24h)
```

### 9.2 GitHub Actions 部署 Workflow

项目 CI/CD 配置位于 `.github/workflows/deploy.yml`：

```yaml
name: Deploy
on:
  workflow_dispatch:
    inputs:
      version:
        description: 'Deploy version (git tag)'
        required: true
      canary_percent:
        description: 'Canary traffic percentage'
        default: '10'
        type: choice
        options: ['10', '50', '100']

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ github.event.inputs.version }}

      - name: Docker Build & Push
        run: |
          for service in gateway auth player cultivation combat social world frontend; do
            docker build -t $REGISTRY/$service:${{ github.event.inputs.version }} -f services/$service/Dockerfile .
            docker push $REGISTRY/$service:${{ github.event.inputs.version }}
          done

  deploy-canary:
    needs: build-and-push
    if: ${{ github.event.inputs.canary_percent == '10' }}
    runs-on: ubuntu-latest
    steps:
      - name: Deploy Canary (10%)
        run: |
          # 更新 Deployment 镜像版本
          kubectl set image deployment/gateway -n cultivation-game \
            gateway=$REGISTRY/gateway:${{ github.event.inputs.version }}
          # ... 其他服务同理 ...

          # 等待 5 分钟监控
          sleep 300

          # 健康检查
          kubectl rollout status deployment/gateway -n cultivation-game --timeout=120s

          # 检查错误率
          # 如果通过则自动触发 50% 阶段

  deploy-50:
    # ... 类似, 扩展到 50% 副本新版本

  deploy-100:
    # ... 全量发布
```

### 9.3 灰度发布检查清单

| 阶段 | 检查项 | 通过标准 | 等待时间 |
|------|--------|---------|---------|
| **10%** | Pod 状态 | 全部 Running & Ready | 5min |
| **10%** | 错误率 | < 0.5% (相比基线) | 5min |
| **10%** | P99 延迟 | < 基线 * 1.2 | 5min |
| **10%** | 在线人数 | 无异常波动 | 5min |
| **50%** | 同 10% 所有项 | 同左 | 10min |
| **50%** | 内存使用 | < 内存限制 * 0.8 | 10min |
| **100%** | 全部指标 | 同 50% | 30min |
| **100%** | 持续观察 | 指标稳定 | 24h |

### 9.4 蓝绿部署 (备选方案)

当灰度发布不适合时，可采用蓝绿部署：

```bash
# 1. 创建新版本 Deployment (green)
kubectl apply -f deploy/k8s/green/gateway-deployment-v2.yaml

# 2. 等待 green 版本就绪
kubectl wait --for=condition=ready pod -l version=v2 -n cultivation-game

# 3. 切换 Service selector 到 v2
kubectl patch service gateway -n cultivation-game \
  -p '{"spec":{"selector":{"version":"v2"}}}'

# 4. 观察 30 分钟，无问题则删除 blue 版本
kubectl delete deployment gateway-v1 -n cultivation-game

# 5. 如有问题，切回 blue
kubectl patch service gateway -n cultivation-game \
  -p '{"spec":{"selector":{"version":"v1"}}}'
```

---

## 10. 回滚方案

### 10.1 回滚触发条件

出现以下任一情况，**立即触发自动回滚**：

- P1 告警触发 (ServiceDown、OnlinePlayerDrop)
- 错误率 > 发布前基线 + 1%
- P99 延迟 > 发布前基线 * 2
- 在线人数下降 > 30%
- 数据库事务失败率 > 5%
- 任何 500 错误持续超过 5 分钟

### 10.2 自动回滚流程

```bash
# 方法 1: kubectl rollout undo (K8s 原生)
kubectl rollout undo deployment/gateway -n cultivation-game
kubectl rollout status deployment/gateway -n cultivation-game --timeout=120s

# 直接回滚到上一版本
kubectl rollout undo deployment/auth -n cultivation-game
kubectl rollout undo deployment/player -n cultivation-game
kubectl rollout undo deployment/cultivation -n cultivation-game
kubectl rollout undo deployment/combat -n cultivation-game
kubectl rollout undo deployment/social -n cultivation-game
kubectl rollout undo deployment/world -n cultivation-game
kubectl rollout undo deployment/trade -n cultivation-game
kubectl rollout undo deployment/ranking -n cultivation-game

# 查看回滚历史
kubectl rollout history deployment/gateway -n cultivation-game
kubectl rollout history deployment/gateway -n cultivation-game --revision=3
```

### 10.3 手动回滚到指定版本

```bash
# 回滚到指定修订版本
kubectl rollout undo deployment/gateway -n cultivation-game --to-revision=3

# 指定镜像版本回滚
kubectl set image deployment/gateway -n cultivation-game \
  gateway=your-registry/gateway:v1.0.0
```

### 10.4 数据库回滚

```bash
# 如果发布包含数据库迁移:
# 1. 先恢复数据库到发布前状态
mysql -h <host> -u root -p cultivation_game < /backup/mysql/pre-release-dump.sql

# 2. 再回滚应用代码
kubectl rollout undo deployment/... -n cultivation-game

# 注意: 数据库回滚是破坏性操作!
# 仅在服务代码回滚后无法兼容新数据时执行
# 优先选择向前兼容的迁移策略
```

### 10.5 回滚后检查清单

| 检查项 | 命令 | 预期结果 |
|--------|------|---------|
| Pod 状态 | `kubectl get pods -n cultivation-game` | 全部 Running |
| 服务健康 | `curl http://localhost:8080/health` | 200 OK |
| 在线人数 | 监控面板 | 恢复到回滚前水平 |
| 错误率 | `sum(rate(game_requests_total{status="error"}[5m]))` | < 1% |
| 消息延迟 | `histogram_quantile(0.99, ...)` | < 50ms |
| 数据库连接 | `kubectl logs pod/mysql-0` | 正常连接 |

### 10.6 PostgreSQL 数据恢复示例

```bash
# 如果仅需恢复某个表
# 1. 从备份中抽取单表
gunzip < /backup/mysql/cultivation_game_20260604_030000.sql.gz \
  | sed -n '/CREATE TABLE.*players/,/CREATE TABLE/p' \
  > /tmp/restore_players.sql

# 2. 导入单表
mysql -h <host> -u root -p cultivation_game < /tmp/restore_players.sql
```

---

> 相关文件路径：
> - Docker Compose: `/root/cultivation-game/deploy/docker-compose.yml`
> - K8s 部署文件: `/root/cultivation-game/deploy/k8s/` (25 个 YAML 文件)
> - Nginx 配置: `/root/cultivation-game/deploy/nginx/`
> - 数据库脚本: `/root/cultivation-game/database/mysql/` (6 个 SQL 文件)
> - Redis 配置: `/root/cultivation-game/database/redis/redis.conf`
> - 监控配置: `/root/cultivation-game/monitoring/` (Prometheus + Loki + Alerts)
> - CI/CD: `/root/cultivation-game/.github/workflows/` (ci.yml + deploy.yml)
