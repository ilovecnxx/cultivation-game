# =============================================================================
# 修仙游戏交易服务 - 多阶段构建 Dockerfile
# =============================================================================
# 阶段一：编译构建
FROM golang:1.24-alpine AS builder

# 安装编译依赖
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# 保持目录结构以支持 go.mod replace 指令 (trade → ../../shared)
COPY services/trade/ services/trade/
COPY shared/ shared/

WORKDIR /build/services/trade

RUN go mod download

# 编译可执行文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /go/bin/trade ./cmd/trade

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
COPY --from=builder /go/bin/trade /app/trade

# 使用非 root 用户运行
USER app
WORKDIR /app

# 暴露 HTTP 端口
EXPOSE 8087

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8087/api/v1/trade/listings?page=1\&page_size=1 || exit 1

# 启动交易服务
ENTRYPOINT ["/app/trade"]
