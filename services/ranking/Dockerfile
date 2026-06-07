# ============================================================
# Stage 1: 构建阶段
# ============================================================
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# 复制依赖文件，利用 Docker 层缓存加速
COPY services/ranking/go.mod services/ranking/go.sum ./

# 下载依赖
RUN go mod download

# 复制源码
COPY services/ranking/ .

# 编译二进制（静态链接，缩小镜像体积）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /build/bin/ranking ./cmd/ranking

# ============================================================
# Stage 2: 运行阶段（最小镜像）
# ============================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# 时区设置（默认中国时区，可通过环境变量覆盖）
ENV TZ=Asia/Shanghai

WORKDIR /app

# 仅复制编译好的二进制
COPY --from=builder /build/bin/ranking /app/ranking

EXPOSE 8088

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8088/api/v1/health || exit 1

ENTRYPOINT ["/app/ranking"]
