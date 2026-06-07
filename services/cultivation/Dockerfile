# ===================== 第一阶段：构建 =====================
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# 先复制依赖文件，利用Docker缓存层
COPY services/cultivation/go.mod services/cultivation/go.sum ./
RUN go mod download

# 复制源码
COPY services/cultivation/ .

# 编译为静态链接的二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /build/cultivation ./cmd/cultivation/

# ===================== 第二阶段：运行 =====================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# 创建非root用户
RUN adduser -D -u 1000 app

WORKDIR /app

# 复制编译产物
COPY --from=builder /build/cultivation .

# 复制数据配置（JSON文件）
COPY --from=builder /build/internal/data ./internal/data

# 使用非root用户运行
USER app

EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

ENV PORT=8080

ENTRYPOINT ["/app/cultivation"]
