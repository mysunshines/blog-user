# 构建阶段
FROM golang:1.25.0-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装依赖
RUN apk add --no-cache git ca-certificates

# 容器内默认 GOPROXY 为 proxy.golang.org，国内网络通常不可达，
# 显式设置为本机一致的国内镜像，避免 go mod download 失败。
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ENV GOSUMDB=off

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-service ./cmd/server

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/user-service .
COPY --from=builder /app/config.yaml .

# 创建非 root 用户
RUN adduser -D -g '' appuser
USER appuser

# 暴露端口 (gRPC=9090, metrics=9091)
EXPOSE 9090 9091

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# 启动命令
CMD ["./user-service"]
