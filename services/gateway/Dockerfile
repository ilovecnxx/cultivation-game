# =========================================================
# Stage 1: 构建
# =========================================================
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# 保持目录结构以支持 go.mod replace 指令 (gateway → ../auth)
COPY services/gateway/ services/gateway/
COPY services/auth/ services/auth/
COPY shared/ shared/

WORKDIR /build/services/gateway

RUN go mod download

# 编译网关
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /build/bin/gateway \
    ./cmd/gateway

# =========================================================
# Stage 2: 运行
# =========================================================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN adduser -D -g '' gateway
USER gateway

WORKDIR /app

COPY --from=builder /build/bin/gateway .

# 健康检查
EXPOSE 8080

ENTRYPOINT ["/app/gateway"]
