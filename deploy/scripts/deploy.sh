#!/usr/bin/env bash
set -euo pipefail

# ============================================
# cultivation-game 部署脚本
# 支持零停机滚动部署、健康检查、回滚
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_DIR="$(cd "$DEPLOY_DIR/.." && pwd)"

source "${DEPLOY_DIR}/.env" 2>/dev/null || true

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ============================================
# 配置
# ============================================
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.yml"
BACKUP_SCRIPT="${DEPLOY_DIR}/scripts/backup.sh"
IMAGE_TAG="${IMAGE_TAG:-latest}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-cultivation-game}"
HEALTHCHECK_INTERVAL=10
HEALTHCHECK_RETRIES=30
ROLLBACK_TAG_FILE="${DEPLOY_DIR}/.rollback_tag"

# ============================================
# Helper Functions
# ============================================
log_info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

die() {
    log_error "$*"
    exit 1
}

docker_compose() {
    docker compose \
        -f "${COMPOSE_FILE}" \
        -p "${COMPOSE_PROJECT_NAME}" \
        "$@"
}

# ============================================
# 健康检查
# ============================================
wait_for_service() {
    local service_name="$1"
    local retries="${2:-$HEALTHCHECK_RETRIES}"
    local interval="${3:-$HEALTHCHECK_INTERVAL}"

    log_info "等待服务 ${service_name} 健康..."

    for i in $(seq 1 "$retries"); do
        local status
        status=$(docker_compose ps --format json "$service_name" 2>/dev/null | \
            python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('Health',''))" 2>/dev/null || echo "")

        if [ "$status" = "healthy" ]; then
            log_ok "服务 ${service_name} 已健康 (尝试 ${i}/${retries})"
            return 0
        fi
        sleep "$interval"
    done

    log_error "服务 ${service_name} 在 ${retries} 次检查后仍未健康"
    return 1
}

wait_for_all_services() {
    local services=("$@")
    for svc in "${services[@]}"; do
        wait_for_service "$svc" || return 1
    done
}

check_current_health() {
    local unhealthy
    unhealthy=$(docker_compose ps --format json 2>/dev/null | \
        python3 -c "
import sys,json
for line in sys.stdin:
    line=line.strip()
    if not line: continue
    d=json.loads(line)
    if d.get('Health') == 'unhealthy':
        print(d.get('Service',''))
" 2>/dev/null || true)

    if [ -n "$unhealthy" ]; then
        log_warn "当前不健康服务:"
        echo "$unhealthy" | while IFS= read -r s; do echo "  - $s"; done
        return 1
    fi
    return 0
}

# ============================================
# 备份数据库
# ============================================
backup_databases() {
    log_info "部署前备份数据库..."
    if [ -f "$BACKUP_SCRIPT" ]; then
        bash "$BACKUP_SCRIPT" pre-deploy || log_warn "备份未完成，继续部署..."
    else
        log_warn "未找到备份脚本: ${BACKUP_SCRIPT}"
    fi
}

# ============================================
# 部署
# ============================================
save_rollback_tag() {
    echo "${IMAGE_TAG}" > "$ROLLBACK_TAG_FILE"
    log_info "保存回滚标签: ${IMAGE_TAG} -> ${ROLLBACK_TAG_FILE}"
}

pull_images() {
    log_info "拉取镜像 (IMAGE_TAG=${IMAGE_TAG})..."
    docker_compose pull --quiet 2>&1 || log_warn "拉取镜像失败，将尝试使用本地镜像构建"
    docker_compose build 2>&1 || true
}

deploy_rolling() {
    local services=("$@")
    log_info "开始滚动更新 ${#services[@]} 个服务..."

    for svc in "${services[@]}"; do
        log_info "更新服务: ${svc}"
        docker_compose up -d --no-deps --force-recreate "$svc" || {
            log_error "服务 ${svc} 启动失败，触发回滚"
            rollback
            return 1
        }

        if ! wait_for_service "$svc"; then
            log_error "服务 ${svc} 健康检查失败，触发回滚"
            rollback
            return 1
        fi

        log_ok "服务 ${svc} 更新成功"
    done
}

deploy_all() {
    log_info "========== 开始部署 cultivation-game =========="
    echo "  镜像标签: ${IMAGE_TAG}"
    echo "  项目:     ${COMPOSE_PROJECT_NAME}"
    echo ""

    # 1. 备份
    backup_databases

    # 2. 保存回滚信息
    save_rollback_tag

    # 3. 拉取/构建镜像
    pull_images

    # 4. 启动依赖服务
    log_info "启动基础设施服务..."
    docker_compose up -d redis mysql mongo nginx || die "基础设施启动失败"

    wait_for_service redis
    wait_for_service mysql
    log_info "基础设施服务已就绪"

    # 5. 滚动更新业务服务
    local biz_services=(gateway auth player cultivation combat social world)
    deploy_rolling "${biz_services[@]}" || return 1

    # 6. 最终健康检查
    log_info "执行最终健康检查..."
    sleep 5
    if ! check_current_health; then
        log_error "最终健康检查失败"
        return 1
    fi

    # 7. 清理旧镜像
    log_info "清理未使用的 Docker 资源..."
    docker image prune -f --filter "until=24h" 2>/dev/null || true

    log_ok "========== 部署完成 =========="
    docker_compose ps
}

# ============================================
# 回滚
# ============================================
rollback() {
    log_info "========== 开始回滚 =========="

    local rollback_tag
    if [ -f "$ROLLBACK_TAG_FILE" ]; then
        rollback_tag=$(cat "$ROLLBACK_TAG_FILE")
        log_info "回滚至标签: ${rollback_tag}"
    else
        log_warn "未找到回滚标签文件，使用 IMAGE_TAG=${IMAGE_TAG}"
        rollback_tag="${IMAGE_TAG}"
    fi

    # 重新拉取旧标签镜像
    IMAGE_TAG="$rollback_tag" docker_compose pull --quiet 2>/dev/null || true

    # 重新部署所有服务
    docker_compose up -d --force-recreate 2>&1 || die "回滚启动失败"

    # 健康检查
    for svc in gateway auth player cultivation combat social world; do
        wait_for_service "$svc" || log_warn "服务 ${svc} 回滚后仍未健康"
    done

    log_ok "========== 回滚完成 =========="
}

# ============================================
# 停机部署 (首次部署)
# ============================================
deploy_fresh() {
    log_info "========== 首次全新部署 =========="

    backup_databases
    save_rollback_tag
    pull_images

    # 一次性启动所有服务
    docker_compose up -d || die "服务启动失败"

    # 等待所有服务就绪
    for svc in redis mysql mongo nginx gateway auth player cultivation combat social world; do
        wait_for_service "$svc" || log_warn "服务 ${svc} 可能未就绪"
    done

    log_ok "========== 首次部署完成 =========="
    docker_compose ps
}

# ============================================
# 状态
# ============================================
show_status() {
    echo "========== 服务状态 =========="
    docker_compose ps
    echo ""
    echo "========== 资源使用 =========="
    docker_compose stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null || \
        docker_compose top
}

show_logs() {
    local service="${1:-}"
    if [ -n "$service" ]; then
        docker_compose logs -f "$service"
    else
        docker_compose logs -f
    fi
}

# ============================================
# Main
# ============================================
main() {
    local cmd="${1:-help}"
    shift || true

    case "$cmd" in
        deploy)
            # 判断是否已有运行中的服务
            if docker_compose ps --format json 2>/dev/null | grep -q '"Name"'; then
                deploy_all "$@"
            else
                deploy_fresh "$@"
            fi
            ;;
        rollback)
            rollback "$@"
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$@"
            ;;
        health)
            check_current_health && log_ok "所有服务健康" || die "存在不健康服务"
            ;;
        help)
            echo "用法: $0 <command>"
            echo ""
            echo "命令:"
            echo "  deploy      部署或更新服务 (零停机)"
            echo "  rollback    回滚至上一个版本"
            echo "  status      查看服务状态"
            echo "  logs        查看日志 (可选: 指定服务名)"
            echo "  health      健康检查"
            echo ""
            echo "环境变量:"
            echo "  IMAGE_TAG   镜像标签 (默认: latest)"
            echo ""
            echo "示例:"
            echo "  IMAGE_TAG=v1.2.3 $0 deploy"
            echo "  $0 status"
            ;;
        *)
            die "未知命令: ${cmd}。使用 '$0 help' 查看帮助"
            ;;
    esac
}

main "$@"
