# ===== 构建阶段 =====
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# 先复制依赖文件以利用缓存
COPY services/combat/go.mod services/combat/go.sum ./
RUN go mod download

# 复制源码
COPY services/combat/ .

# 构建二进制
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o combat ./cmd/combat

# ===== 运行阶段 =====
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制
COPY --from=builder /build/combat .

# 复制数据文件
COPY --from=builder /build/internal/data/ ./internal/data/

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# 默认配置(可通过环境变量 COMBAT_CONFIG_PATH 覆盖)
ENV COMBAT_CONFIG_PATH=/app/config.json

ENTRYPOINT ["/app/combat"]
