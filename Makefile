.PHONY: all build run clean proto test docker

# 变量定义
BINARY_NAME=im-server
MAIN_PATH=cmd/server
CONFIG_PATH=config/config.yaml

# 默认目标
all: build

# 编译
build:
	@echo "Building..."
	@go build -o bin/$(BINARY_NAME) $(MAIN_PATH)/main.go $(MAIN_PATH)/config.go
	@echo "Build complete: bin/$(BINARY_NAME)"

# 运行
run: build
	@echo "Running..."
	@./bin/$(BINARY_NAME) -config=$(CONFIG_PATH)

# 清理
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f logs/*.log
	@echo "Clean complete"

# 生成protobuf代码
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		api/proto/im_protocol.proto
	@echo "Protobuf generation complete"

# 运行测试
test:
	@echo "Running tests..."
	@go test -v ./...

# 下载依赖
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# 格式化代码
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# 代码检查
lint:
	@echo "Linting code..."
	@golangci-lint run

# 构建Docker镜像
docker-build:
	@echo "Building Docker image..."
	@docker build -t im-server:latest .

# 运行Docker容器
docker-run:
	@echo "Running Docker container..."
	@docker-compose up -d

# 停止Docker容器
docker-stop:
	@echo "Stopping Docker container..."
	@docker-compose down

# 查看日志
logs:
	@tail -f logs/im-server.log

# 帮助
help:
	@echo "Available targets:"
	@echo "  all          - Build the project (default)"
	@echo "  build        - Build the binary"
	@echo "  run          - Build and run the server"
	@echo "  clean        - Remove build artifacts"
	@echo "  proto        - Generate protobuf code"
	@echo "  test         - Run tests"
	@echo "  deps         - Download dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  docker-stop  - Stop Docker containers"
	@echo "  logs         - View server logs"
	@echo "  help         - Show this help message"

