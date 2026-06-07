# =============================================================================
# 修仙游戏认证服务 - 多阶段构建 Dockerfile
# =============================================================================
# 阶段一：编译构建
FROM golang:1.24-alpine AS builder

# 安装编译依赖
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# 保持目录结构以支持 go.mod replace 指令 (auth → ../../shared)
COPY services/auth/ services/auth/
COPY shared/ shared/

WORKDIR /build/services/auth

RUN go mod download

# 编译可执行文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /go/bin/auth ./cmd/auth

# =============================================================================
# 阶段二：运行镜像
FROM alpine:3.20

# 安装运行时证书和时区数据
RUN apk add --no-cache ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN adduser -D -g '' app

# 复制编译产物
COPY --from=builder /go/bin/auth /app/auth

# 使用非 root 用户运行
USER app
WORKDIR /app

# 暴露 gRPC 端口
EXPOSE 50060

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/auth", "-health"] || exit 1

# 启动认证服务
ENTRYPOINT ["/app/auth"]
