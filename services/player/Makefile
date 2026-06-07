APP_NAME = player-server
BUILD_DIR = ./build
CMD_DIR = ./cmd/player

.PHONY: build run clean test lint docker-build docker-run

# 构建二进制
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)/main.go
	@echo "构建完成: $(BUILD_DIR)/$(APP_NAME)"

# 运行服务（开发）
run:
	go run $(CMD_DIR)/main.go

# 清理构建产物
clean:
	rm -rf $(BUILD_DIR)
	@echo "清理完成"

# 运行测试
test:
	go test ./... -v -race -cover

# 代码检查
lint:
	golangci-lint run ./...

# 构建 Docker 镜像
docker-build:
	docker build -t cultivation/$(APP_NAME):latest .

# 运行 Docker 容器（开发模式）
docker-run:
	docker run --rm -p 8080:8080 \
		-e MYSQL_DSN="root:password@tcp(host.docker.internal:3306)/cultivation?charset=utf8mb4&parseTime=True&loc=Local" \
		-e REDIS_ADDR="host.docker.internal:6379" \
		--name $(APP_NAME) \
		cultivation/$(APP_NAME):latest

# 初始化数据库表结构
db-init:
	mysql -u root -p < schema.sql

# 生成 API 文档（如果安装了 swag）
swag:
	swag init -g $(CMD_DIR)/main.go -o ./docs
