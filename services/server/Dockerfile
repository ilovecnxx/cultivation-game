# 九转修仙 - 统一单体服务 Dockerfile
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /build

# 先复制所有源码（本地模块需要）
COPY services/ ./services/
COPY shared/ ./shared/

# 下载依赖并构建
WORKDIR /build/services/server
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /build/bin/server ./cmd/server/

# 运行时
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /build/bin/server /app/server
COPY --from=builder /build/services/combat/internal/data/ /app/data/combat/
COPY --from=builder /build/services/cultivation/internal/data/ /app/data/cultivation/
COPY --from=builder /build/services/player/internal/data/ /app/data/player/
COPY --from=builder /build/services/world/internal/data/ /app/data/world/
RUN chown -R app:app /app
USER app
EXPOSE 8080
ENV SERVER_PORT=8080
ENV GIN_MODE=release
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["/app/server"]
