.PHONY: build run test clean proto docker

# 服务名称
SERVICE_NAME=user-service
# 二进制文件
BINARY_NAME=$(SERVICE_NAME)
# 源码目录
SRC_DIR=cmd/server

# 默认目标
all: build

# 编译服务
build:
	@echo "Building $(SERVICE_NAME)..."
	mkdir bin
	go build -o bin/$(BINARY_NAME) $(SRC_DIR)/main.go

# 运行服务
run:
	@echo "Running $(SERVICE_NAME)..."
	go run $(SRC_DIR)/main.go

# 运行测试
test:
	@echo "Running tests..."
	go test -v ./...

# 清理构建文件
clean:
	@echo "Cleaning..."
	rm -f bin/*

# 依赖管理
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# 依赖更新
update:
	@echo "Updating dependencies..."
	go get -u ./...

# Proto 生成
proto:
	@echo "Generating protobuf code..."
	mkdir -p proto/pb
	cd proto && protoc --go_out=pb --go_opt=paths=source_relative \
		--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
		*.proto

# Docker 构建
docker:
	@echo "Building Docker image..."
	docker build -t $(SERVICE_NAME):latest .

# Docker 运行
docker-run:
	@echo "Running Docker container..."
	docker run -p 8081:8081 -p 9001:9001 -p 9091:9091 $(SERVICE_NAME):latest

# 代码检查
lint:
	@echo "Running linter..."
	golangci-lint run

# 格式化代码（自动分组 import：标准库 → 项目包 → 第三方包）
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@which goimports > /dev/null || GO111MODULE=on go install golang.org/x/tools/cmd/goimports@latest
	goimports -local common -w .

# 帮助
help:
	@echo "Available targets:"
	@echo "  build      - Build the service"
	@echo "  run        - Run the service"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build files"
	@echo "  deps       - Install dependencies"
	@echo "  update     - Update dependencies"
	@echo "  proto      - Generate protobuf code"
	@echo "  docker     - Build Docker image"
	@echo "  docker-run - Run Docker container"
	@echo "  lint       - Run linter"
	@echo "  fmt        - Format code"
