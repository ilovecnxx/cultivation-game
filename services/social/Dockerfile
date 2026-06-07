# ============================================================
# 修仙游戏 - 社交服务 Dockerfile
# 多阶段构建
# ============================================================

# ---- 构建阶段 ----
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 安装依赖
COPY services/social/go.mod services/social/go.sum ./
RUN go mod download

# 编译
COPY services/social/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/social ./cmd/social/

# ---- 运行阶段 ----
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/social .

EXPOSE 8082

CMD ["./social", "-config", "config.json"]
