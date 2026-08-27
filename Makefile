.PHONY: all build run test clean deps update proto docker docker-run lint fmt help

# 服务名称（各服务按需修改）
SERVICE_SHORT_NAME=user
SERVICE_NAME=$(SERVICE_SHORT_NAME)-service
# 二进制文件
BINARY_NAME=$(SERVICE_NAME)
# 源码目录
SRC_DIR=cmd/server
# 构建输出目录
BIN_DIR=bin
# 容器端口映射（docker-run 使用，按服务实际情况修改）
PORTS=8081:8081 9001:9001 9091:9091

# 版本号：优先取 git describe，失败回退 dev
GIT_VERSION      := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
VERSION_LDFLAGS := -X main.Version=$(GIT_VERSION)

# Proto 生成配置
# MODULE：Go module 路径，须与 proto 的 go_package 前缀一致
MODULE := github.com/mysunshines/blog-$(SERVICE_SHORT_NAME)
# PROTO_DIR：proto 源文件目录
PROTO_DIR := proto
# PROTO_OUT：生成代码输出目录
PROTO_OUT := $(PROTO_DIR)/pb
PROTOC_OPTS := --go_out=. --go_opt=module=$(MODULE) --go-grpc_out=. --go-grpc_opt=module=$(MODULE)

# 默认目标
all: build

# 依赖管理（整理并下载）
deps:
	go mod tidy
	go mod download

# 依赖更新
update:
	go get -u ./...

# Proto 生成（无 proto 目录时自动跳过）
# 说明：仅暴露纯 gRPC，proto 不再包含 google.api.http 注解，
#       因此只生成 --go_out（message）与 --go-grpc_out（service），不生成 grpc-gateway 的 .pb.gw.go。
# 从模块根调用 protoc：--go_out=. 输出根即模块根，module= 将 go_package 映射到 proto/pb/，
# 避免产生冗余的 proto/proto 或 github.com 目录。
proto:
	@if [ -d $(PROTO_DIR) ]; then \
		mkdir -p $(PROTO_OUT) && \
		protoc -I $(PROTO_DIR) $(PROTOC_OPTS) $(PROTO_DIR)/$(SERVICE_SHORT_NAME).proto; \
	else \
		echo "==> No proto directory"; \
	fi

# 代码检查
lint:
	golangci-lint run

# 格式化代码（自动分组 import：标准库 → 项目包 → 第三方包）
fmt:
	go fmt ./...
	@which goimports > /dev/null 2>&1 || GO111MODULE=on go install golang.org/x/tools/cmd/goimports@latest
	goimports -local common -w .

# 编译服务
build: deps proto fmt
	mkdir -p $(BIN_DIR)
	go build -ldflags "$(VERSION_LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(SRC_DIR)/main.go

# 运行服务
run:
	go run $(SRC_DIR)/main.go

# 运行测试
test:
	go test -v ./...

# 清理构建文件
clean:
	rm -f $(BIN_DIR)/*

# Docker 构建（构建上下文为当前服务目录）
docker-build:
	docker build --build-arg GIT_VERSION=$(GIT_VERSION) -t $(SERVICE_NAME):latest .

# Docker 运行
docker-run:
	docker run -p $(PORTS) $(SERVICE_NAME):latest
