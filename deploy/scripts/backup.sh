#!/usr/bin/env bash
set -euo pipefail

# ============================================
# cultivation-game 数据库备份脚本
# 支持 MySQL / Redis / MongoDB 备份
# 保留最近 7 天，可上传至 S3/OSS
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_DIR="$(cd "$DEPLOY_DIR/.." && pwd)"

# 加载环境变量
if [ -f "${DEPLOY_DIR}/.env" ]; then
    set -a
    source "${DEPLOY_DIR}/.env"
    set +a
fi

# ============================================
# 配置
# ============================================
BACKUP_DIR="${BACKUP_DIR:-${DEPLOY_DIR}/backups}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-7}"
DATE_TAG="$(date +%Y%m%d_%H%M%S)"
TIMESTAMP="$(date +%s)"

# MySQL
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-game}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
MYSQL_DATABASE="${MYSQL_DATABASE:-cultivation_game}"

# Redis
REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${REDIS_PORT:-6379}"

# MongoDB
MONGO_HOST="${MONGO_HOST:-127.0.0.1}"
MONGO_PORT="${MONGO_PORT:-27017}"
MONGO_USER="${MONGO_USER:-admin}"
MONGO_PASSWORD="${MONGO_PASSWORD:-}"
MONGO_DATABASE="${MONGO_DATABASE:-cultivation_game}"

# S3/OSS
S3_ENABLED="${S3_ENABLED:-false}"
S3_BUCKET="${S3_BUCKET:-}"
S3_ENDPOINT="${S3_ENDPOINT:-}"
S3_REGION="${S3_REGION:-}"
AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-}"
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-}"

# Docker Compose 项目名
COMPOSE_PROJECT="${COMPOSE_PROJECT_NAME:-cultivation-game}"

# ============================================
# Helper Functions
# ============================================
log_info()  { echo "[INFO]  $(date '+%Y-%m-%d %H:%M:%S')  $*"; }
log_ok()    { echo "[OK]    $(date '+%Y-%m-%d %H:%M:%S')  $*"; }
log_warn()  { echo "[WARN]  $(date '+%Y-%m-%d %H:%M:%S')  $*"; }
log_error() { echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S')  $*"; }

ensure_backup_dir() {
    mkdir -p "${BACKUP_DIR}/{mysql,redis,mongo}"
}

cleanup_old_backups() {
    local dir="$1"
    log_info "清理 ${dir} 中超过 ${RETENTION_DAYS} 天的备份..."
    find "${dir}" -type f -name "*.gz" -mtime +"${RETENTION_DAYS}" -delete 2>/dev/null || true
    find "${dir}" -type f -name "*.rdb" -mtime +"${RETENTION_DAYS}" -delete 2>/dev/null || true
    log_ok "清理完成"
}

upload_to_s3() {
    if [ "$S3_ENABLED" != "true" ] || [ -z "$S3_BUCKET" ]; then
        return 0
    fi

    local file_path="$1"
    local dest_path="$2"

    log_info "上传 ${file_path} 到 S3://${S3_BUCKET}/${dest_path}..."

    if command -v aws &>/dev/null; then
        aws s3 cp "$file_path" "s3://${S3_BUCKET}/${dest_path}" \
            --endpoint-url "${S3_ENDPOINT:-}" \
            --region "${S3_REGION:-}" 2>&1 || log_warn "S3 上传失败"
    elif command -v rclone &>/dev/null; then
        rclone copy "$file_path" ":s3,env_auth,bucket=${S3_BUCKET}/${dest_path}" 2>&1 || log_warn "rclone 上传失败"
    elif command -v mc &>/dev/null; then
        mc cp "$file_path" "backup/${S3_BUCKET}/${dest_path}" 2>&1 || log_warn "minio client 上传失败"
    else
        log_warn "未安装 aws/rclone/mc，跳过 S3 上传"
    fi

    log_ok "上传完成"
}

# ============================================
# MySQL 备份 (mysqldump)
# ============================================
backup_mysql() {
    log_info "===== MySQL 备份 ====="
    local backup_file="${BACKUP_DIR}/mysql/cultivation_game_${DATE_TAG}.sql.gz"

    if [ -z "$MYSQL_PASSWORD" ]; then
        log_warn "MYSQL_PASSWORD 未设置，跳过 MySQL 备份"
        return 1
    fi

    # 尝试从 Docker 容器中执行
    local container_name
    container_name="$(docker ps --format '{{.Names}}' 2>/dev/null | grep "${COMPOSE_PROJECT}" | grep mysql | head -1)" || true

    if [ -n "$container_name" ]; then
        log_info "从 Docker 容器 ${container_name} 备份 MySQL..."
        docker exec "$container_name" \
            mysqldump \
                --single-transaction \
                --routines \
                --triggers \
                --events \
                --hex-blob \
                --skip-lock-tables \
                --quick \
                -h127.0.0.1 \
                -u"${MYSQL_USER}" \
                -p"${MYSQL_PASSWORD}" \
                "${MYSQL_DATABASE}" | gzip > "$backup_file"
    else
        log_info "从外部连接备份 MySQL (${MYSQL_HOST}:${MYSQL_PORT})..."
        mysqldump \
            --single-transaction \
            --routines \
            --triggers \
            --events \
            --hex-blob \
            --skip-lock-tables \
            --quick \
            -h"${MYSQL_HOST}" \
            -P"${MYSQL_PORT}" \
            -u"${MYSQL_USER}" \
            -p"${MYSQL_PASSWORD}" \
            "${MYSQL_DATABASE}" | gzip > "$backup_file"
    fi

    if [ -f "$backup_file" ] && [ -s "$backup_file" ]; then
        local size
        size=$(du -h "$backup_file" | cut -f1)
        log_ok "MySQL 备份完成: ${backup_file} (${size})"
        upload_to_s3 "$backup_file" "mysql/${DATE_TAG}_cultivation_game.sql.gz"
    else
        log_error "MySQL 备份文件为空或不存在"
        return 1
    fi
}

# ============================================
# Redis RDB 备份
# ============================================
backup_redis() {
    log_info "===== Redis 备份 ====="

    local container_name
    container_name="$(docker ps --format '{{.Names}}' 2>/dev/null | grep "${COMPOSE_PROJECT}" | grep redis | head -1)" || true

    if [ -z "$container_name" ]; then
        log_warn "未找到 Redis 容器，跳过 Redis 备份"
        return 1
    fi

    # 触发 SAVE (阻塞式) 或 BGSAVE
    log_info "触发 Redis BGSAVE..."
    docker exec "$container_name" redis-cli BGSAVE 2>/dev/null || true
    sleep 2

    # 确认 RDB 文件路径
    local rdb_dir
    rdb_dir=$(docker exec "$container_name" redis-cli CONFIG GET dir 2>/dev/null | tail -1)
    local rdb_name
    rdb_name=$(docker exec "$container_name" redis-cli CONFIG GET dbfilename 2>/dev/null | tail -1)
    local rdb_path="${rdb_dir}/${rdb_name}"

    local backup_file="${BACKUP_DIR}/redis/redis_${DATE_TAG}.rdb.gz"

    # 从容器复制 RDB 文件
    if docker cp "${container_name}:${rdb_path}" - 2>/dev/null | gzip > "$backup_file"; then
        local size
        size=$(du -h "$backup_file" | cut -f1)
        log_ok "Redis 备份完成: ${backup_file} (${size})"
        upload_to_s3 "$backup_file" "redis/${DATE_TAG}.rdb.gz"
    else
        log_error "Redis RDB 备份失败"
        return 1
    fi
}

# ============================================
# MongoDB 备份 (mongodump)
# ============================================
backup_mongo() {
    log_info "===== MongoDB 备份 ====="

    local container_name
    container_name="$(docker ps --format '{{.Names}}' 2>/dev/null | grep "${COMPOSE_PROJECT}" | grep mongo | grep -v mongo-express | head -1)" || true

    local backup_file="${BACKUP_DIR}/mongo/mongo_${DATE_TAG}.gz"

    local auth_params=""
    if [ -n "$MONGO_USER" ] && [ -n "$MONGO_PASSWORD" ]; then
        auth_params="--username=${MONGO_USER} --password=${MONGO_PASSWORD} --authenticationDatabase=admin"
    fi

    if [ -n "$container_name" ]; then
        log_info "从 Docker 容器 ${container_name} 备份 MongoDB..."
        docker exec "$container_name" \
            mongodump \
                $auth_params \
                --db="${MONGO_DATABASE}" \
                --archive \
                --gzip 2>/dev/null > "$backup_file" || {
            log_error "MongoDB 容器内备份失败"
            return 1
        }
    else
        log_info "从外部连接备份 MongoDB (${MONGO_HOST}:${MONGO_PORT})..."
        mongodump \
            $auth_params \
            --host="${MONGO_HOST}" \
            --port="${MONGO_PORT}" \
            --db="${MONGO_DATABASE}" \
            --archive \
            --gzip > "$backup_file" 2>/dev/null || {
            log_error "MongoDB 外部备份失败"
            return 1
        }
    fi

    if [ -f "$backup_file" ] && [ -s "$backup_file" ]; then
        local size
        size=$(du -h "$backup_file" | cut -f1)
        log_ok "MongoDB 备份完成: ${backup_file} (${size})"
        upload_to_s3 "$backup_file" "mongo/${DATE_TAG}.gz"
    else
        log_error "MongoDB 备份文件为空或不存在"
        return 1
    fi
}

# ============================================
# 全量备份
# ============================================
backup_all() {
    log_info "========== 开始全量数据库备份 =========="
    log_info "备份目录: ${BACKUP_DIR}"
    log_info "保留天数: ${RETENTION_DAYS}"

    ensure_backup_dir

    backup_mysql || log_warn "MySQL 备份跳过或失败"
    backup_redis || log_warn "Redis 备份跳过或失败"
    backup_mongo || log_warn "MongoDB 备份跳过或失败"

    # 清理旧备份
    cleanup_old_backups "${BACKUP_DIR}/mysql"
    cleanup_old_backups "${BACKUP_DIR}/redis"
    cleanup_old_backups "${BACKUP_DIR}/mongo"

    log_ok "========== 备份流程全部完成 =========="
}

# ============================================
# 列出备份
# ============================================
list_backups() {
    echo "========== 备份文件清单 =========="
    echo ""
    echo "--- MySQL ---"
    ls -lh "${BACKUP_DIR}/mysql/" 2>/dev/null || echo "(无)"
    echo ""
    echo "--- Redis ---"
    ls -lh "${BACKUP_DIR}/redis/" 2>/dev/null || echo "(无)"
    echo ""
    echo "--- MongoDB ---"
    ls -lh "${BACKUP_DIR}/mongo/" 2>/dev/null || echo "(无)"
}

# ============================================
# Main
# ============================================
main() {
    local cmd="${1:-all}"

    case "$cmd" in
        all|full)
            backup_all
            ;;
        mysql)
            backup_mysql
            ;;
        redis)
            backup_redis
            ;;
        mongo)
            backup_mongo
            ;;
        list)
            list_backups
            ;;
        pre-deploy)
            # deploy.sh 在部署前调用的快速备份
            backup_all
            ;;
        *)
            echo "用法: $0 <command>"
            echo ""
            echo "命令:"
            echo "  all         全量备份 (默认)"
            echo "  mysql       仅备份 MySQL"
            echo "  redis       仅备份 Redis"
            echo "  mongo       仅备份 MongoDB"
            echo "  list        列出已有备份"
            echo ""
            echo "环境变量:"
            echo "  BACKUP_DIR              备份存储目录 (默认: ./backups)"
            echo "  BACKUP_RETENTION_DAYS   保留天数 (默认: 7)"
            echo "  S3_ENABLED              是否上传到 S3/OSS (true/false)"
            echo "  S3_BUCKET               S3 存储桶名"
            echo ""
            echo "示例:"
            echo "  $0 all"
            echo "  $0 mysql"
            echo "  S3_ENABLED=true S3_BUCKET=my-bucket $0 all"
            ;;
    esac
}

main "$@"
