# ---- 构建阶段 ----
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# 复制所有文件
COPY services/player/ .

# 下载依赖
RUN go mod download

# 编译
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/player-server ./cmd/player/main.go

# ---- 运行阶段 ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/player-server .
COPY --from=builder /app/internal/data/ /internal/data/

EXPOSE 8080

ENTRYPOINT ["/app/player-server"]
