# ============================================================
# 第一阶段: 构建阶段
# ============================================================
FROM golang:1.24-alpine AS builder

WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git ca-certificates

# 先复制依赖文件，利用Docker缓存
COPY services/world/go.mod services/world/go.sum ./
RUN go mod download

# 复制源代码
COPY services/world/cmd/ ./cmd/
COPY services/world/internal/ ./internal/

# 多阶段构建: 为不同架构编译
# 默认构建 amd64
ARG TARGETARCH=amd64
ARG TARGETOS=linux

# 编译世界服务
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /build/world-server ./cmd/world/

# ============================================================
# 第二阶段: 运行阶段
# ============================================================
FROM alpine:3.19

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非root用户
RUN adduser -D -h /app appuser

# 从构建阶段复制编译产物
COPY --from=builder /build/world-server /app/world-server

# 复制数据文件
COPY --from=builder /build/internal/data/ /app/internal/data/

# 设置权限
RUN chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8086

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8086/api/v1/health || exit 1

# 启动世界服务
ENTRYPOINT ["/app/world-server"]
