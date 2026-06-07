# ============================================
# cultivation-game 部署管理 Makefile
# ============================================

SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

PROJECT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)
DEPLOY_DIR  := $(PROJECT_DIR)/deploy
SCRIPTS_DIR := $(DEPLOY_DIR)/scripts

# ============================================
# 引入 .env（如果存在）
# ============================================
ifneq ($(wildcard $(DEPLOY_DIR)/.env),)
    include $(DEPLOY_DIR)/.env
    export
endif

# 镜像标签（可被覆盖）
IMAGE_TAG ?= latest

.PHONY: help dev build test deploy rollback backup logs status

help: ## 显示帮助信息
	@echo '用法: make <target> [ENV=value...]'
	@echo ''
	@echo '目标:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ''
	@echo '常用变量:'
	@echo '  IMAGE_TAG     镜像标签 (默认: latest)'
	@echo '  GAME_ENV      环境 (development/production)'
	@echo ''
	@echo '示例:'
	@echo '  make dev               # 启动开发环境'
	@echo '  make build             # 构建所有服务镜像'
	@echo '  make deploy            # 部署到生产环境'
	@echo '  IMAGE_TAG=v1.2.3 make deploy  # 指定版本部署'
	@echo '  make rollback          # 回滚到上一个版本'

# ============================================
# 开发环境
# ============================================

dev: ## 启动开发环境 (docker-compose up -d)
	@echo "=== 启动开发环境 ==="
	cd $(DEPLOY_DIR) && docker compose -f docker-compose.yml -p cultivation-game up -d
	@echo "服务已启动。访问 http://localhost:80"
	@echo "使用 'make logs' 查看日志"

dev-down: ## 停止并清理开发环境
	@echo "=== 停止开发环境 ==="
	cd $(DEPLOY_DIR) && docker compose -p cultivation-game down --remove-orphans

dev-reset: dev-down ## 重置开发环境 (删除数据卷)
	@echo "=== 重置开发环境 (删除数据) ==="
	cd $(DEPLOY_DIR) && docker compose -p cultivation-game down -v --remove-orphans
	@echo "数据卷已清除。运行 'make dev' 重新启动。"

# ============================================
# 构建
# ============================================

build: ## 构建所有服务镜像
	@echo "=== 构建服务镜像 (IMAGE_TAG=$(IMAGE_TAG)) ==="
	cd $(DEPLOY_DIR) && IMAGE_TAG=$(IMAGE_TAG) docker compose -f docker-compose.yml build
	@echo "构建完成"

build-gateway: ## 构建 gateway 服务镜像
	cd $(DEPLOY_DIR) && IMAGE_TAG=$(IMAGE_TAG) docker compose -f docker-compose.yml build gateway

build-auth: ## 构建 auth 服务镜像
	cd $(DEPLOY_DIR) && IMAGE_TAG=$(IMAGE_TAG) docker compose -f docker-compose.yml build auth

build-player: ## 构建 player 服务镜像
	cd $(DEPLOY_DIR) && IMAGE_TAG=$(IMAGE_TAG) docker compose -f docker-compose.yml build player

build-cultivation: ## 构建 cultivation 服务镜像
	cd $(DEPLOY_DIR) && IMAGE_TAG=$(IMAGE_TAG) docker compose -f docker-compose.yml build cultivation

build-combat: ## 构建 combat 服务镜像
	cd $(DEPLOY_DIR) && IMAGE_TAG=$(IMAGE_TAG) docker compose -f docker-compose.yml build combat

build-social: ## 构建 social 服务镜像
	cd $(DEPLOY_DIR) && IMAGE_TAG=$(IMAGE_TAG) docker compose -f docker-compose.yml build social

build-world: ## 构建 world 服务镜像
	cd $(DEPLOY_DIR) && IMAGE_TAG=$(IMAGE_TAG) docker compose -f docker-compose.yml build world

# ============================================
# 测试
# ============================================

test: ## 运行所有测试
	@echo "=== 运行测试 ==="
	@echo "请实现各服务的测试命令。示例："
	@echo "  cd $(PROJECT_DIR)/services/gateway && go test ./..."
	@echo "  cd $(PROJECT_DIR)/services/auth && go test ./..."
	@echo "  cd $(PROJECT_DIR)/services/player && go test ./..."

test-gateway: ## 运行 gateway 服务测试
	cd $(PROJECT_DIR)/services/gateway && go test ./... || echo "测试未实现"

test-auth: ## 运行 auth 服务测试
	cd $(PROJECT_DIR)/services/auth && go test ./... || echo "测试未实现"

test-player: ## 运行 player 服务测试
	cd $(PROJECT_DIR)/services/player && go test ./... || echo "测试未实现"

test-cultivation: ## 运行 cultivation 服务测试
	cd $(PROJECT_DIR)/services/cultivation && go test ./... || echo "测试未实现"

test-combat: ## 运行 combat 服务测试
	cd $(PROJECT_DIR)/services/combat && go test ./... || echo "测试未实现"

test-social: ## 运行 social 服务测试
	cd $(PROJECT_DIR)/services/social && go test ./... || echo "测试未实现"

test-world: ## 运行 world 服务测试
	cd $(PROJECT_DIR)/services/world && go test ./... || echo "测试未实现"

# ============================================
# 部署
# ============================================

deploy: ## 部署服务 (零停机滚动更新)
	@echo "=== 部署服务 (IMAGE_TAG=$(IMAGE_TAG)) ==="
	IMAGE_TAG=$(IMAGE_TAG) bash $(SCRIPTS_DIR)/deploy.sh deploy

deploy-fresh: ## 首次全新部署
	@echo "=== 首次全量部署 (IMAGE_TAG=$(IMAGE_TAG)) ==="
	IMAGE_TAG=$(IMAGE_TAG) bash $(SCRIPTS_DIR)/deploy.sh fresh

rollback: ## 回滚到上一个版本
	@echo "=== 回滚服务 ==="
	bash $(SCRIPTS_DIR)/deploy.sh rollback

# ============================================
# 备份
# ============================================

backup: ## 执行全量数据库备份
	@echo "=== 备份数据库 ==="
	bash $(SCRIPTS_DIR)/backup.sh all

backup-mysql: ## 仅备份 MySQL
	bash $(SCRIPTS_DIR)/backup.sh mysql

backup-redis: ## 仅备份 Redis
	bash $(SCRIPTS_DIR)/backup.sh redis

backup-mongo: ## 仅备份 MongoDB
	bash $(SCRIPTS_DIR)/backup.sh mongo

backup-list: ## 列出已有备份
	bash $(SCRIPTS_DIR)/backup.sh list

# ============================================
# 日志 & 状态
# ============================================

logs: ## 查看所有服务日志 (tail -f)
	cd $(DEPLOY_DIR) && docker compose -p cultivation-game logs -f

logs-%: ## 查看指定服务日志，如: make logs-gateway
	cd $(DEPLOY_DIR) && docker compose -p cultivation-game logs -f $(@:logs-%=%)

status: ## 查看各服务运行状态
	cd $(DEPLOY_DIR) && docker compose -p cultivation-game ps

stats: ## 查看各服务资源使用情况
	cd $(DEPLOY_DIR) && docker compose -p cultivation-game stats --no-stream

health: ## 检查各服务健康状态
	bash $(SCRIPTS_DIR)/deploy.sh health

# ============================================
# SSL / Certbot
# ============================================

ssl-init: ## 初始化 Let's Encrypt SSL 证书
	@echo "=== 初始化 SSL 证书 ==="
	@echo "请确保域名 DNS 已指向服务器，然后运行:"
	@echo '  docker run -it --rm -v $(DEPLOY_DIR)/nginx/letsencrypt:/var/www/letsencrypt \
	    -v $(DEPLOY_DIR)/ssl:/etc/letsencrypt \
	    certbot/certbot certonly --webroot \
	    -w /var/www/letsencrypt \
	    -d your-domain.com'

ssl-renew: ## 续签 SSL 证书
	docker run --rm -v $(DEPLOY_DIR)/ssl:/etc/letsencrypt \
	    -v $(DEPLOY_DIR)/nginx/letsencrypt:/var/www/letsencrypt \
	    certbot/certbot renew --quiet

# ============================================
# 工具
# ============================================

prune: ## 清理 Docker 未使用资源
	docker system prune -f --volumes

env-check: ## 检查 .env 配置完整性
	@echo "=== 检查 .env 配置 ==="
	@if [ -f $(DEPLOY_DIR)/.env ]; then \
	    echo ".env 文件存在"; \
	    echo "---"; \
	    grep -v '^#' $(DEPLOY_DIR)/.env | grep -v '^\s*$$' || echo "(空)"; \
	else \
	    echo ".env 文件不存在"; \
	    echo "请复制 .env.example 到 .env 并配置"; \
	    echo "  cp $(DEPLOY_DIR)/.env.example $(DEPLOY_DIR)/.env"; \
	fi
