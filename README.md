# User Service - 用户服务

## 一、服务概述

用户服务是博客系统的核心服务之一，负责用户注册、登录、认证和用户信息管理。

**端口配置**:
- HTTP: 8081
- gRPC: 9101
- Metrics: 9091

## 二、技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| RPC框架 | gRPC + Protobuf（**业务层纯 gRPC**，HTTP 仅保留 `/health`、`/ready`、`/version` 探活端口，无 gin 业务路由） |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis |
| 监控 | Prometheus |
| 注册中心 | Consul |
| 加密 | bcrypt (密码), JWT (认证) |

## 三、项目结构

```
user-service/
├── cmd/server/
│   └── main.go                 # 服务启动（gRPC 业务端口 + 探活 HTTP 端口）
├── internal/
│   ├── config/config.go        # 配置管理
│   ├── handler/
│   │   └── v1/grpc_handler.go  # gRPC 处理器（含 requireGRPCAdmin 与审计）
│   ├── service/user_service.go # 业务逻辑层
│   ├── repository/
│   │   ├── user_repo.go        # 用户数据访问
│   │   ├── token_repo.go       # Token数据访问
│   │   └── blacklist_repo.go   # 黑名单数据访问
│   ├── model/user.go           # 数据模型
│   └── middleware/middleware.go # 中间件
├── pkg/
│   ├── errors/errors.go        # 错误处理
│   └── response/response.go    # 响应封装
├── proto/
│   └── user.proto              # gRPC定义
├── config.yaml                 # 配置文件
└── Dockerfile
```

## 四、API 列表

### 4.1 gRPC API（经网关 `/api/v1` 与 `/admin-api` 反射代理）

> 业务层纯 gRPC，下表路径为网关反射代理入口（`/api/v1/user/<snake_method>` / `/admin-api/users/<snake_method>`），实际对应 `user.v1.UserService` 的 gRPC 方法。

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|

#### 4.1.1 POST `/api/v1/user/register` - 用户注册

**请求体 (JSON)**:
```json
{
    "username": "string",    // string, 必填, 用户名, 长度3-64字符
    "email": "string",        // string, 必填, 邮箱, 邮箱格式
    "password": "string",    // string, 必填, 密码, 长度6-32字符
    "nickname": "string"     // string, 选填, 昵称
}
```

**响应**:
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "token": "string",
        "user": { ... }
    }
}
```

#### 4.1.2 POST `/api/v1/user/login` - 用户登录

**请求体 (JSON)**:
```json
{
    "username": "string",     // string, 必填, 用户名
    "password": "string"      // string, 必填, 密码
}
```

**响应**:
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "token": "string",
        "user": { ... }
    }
}
```

#### 4.1.3 POST `/api/v1/user/logout` - 用户登出

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

#### 4.1.4 GET `/api/v1/user` - 获取用户信息

**查询参数 (Query Parameters)**:
```
id: int,         // int, 选填, 用户ID (与username二选一)
username: string // string, 选填, 用户名 (与id二选一)
```

#### 4.1.5 PUT `/api/v1/user` - 更新用户信息

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**请求体 (JSON)**:
```json
{
    "user_id": "uint",           // uint, 必填, 用户ID (从Token解析)
    "nickname": "string",        // string, 选填, 昵称
    "avatar": "string",          // string, 选填, 头像URL
    "bio": "string"              // string, 选填, 个人简介
}
```

#### 4.1.6 DELETE `/api/v1/user/:id` - 删除用户

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**路径参数**:
```
id: int,     // int, 必填, 用户ID
```

#### 4.1.7 GET `/api/v1/users` - 获取用户列表

**查询参数 (Query Parameters)**:
```
page: int,         // int, 选填, 页码, 默认1, 最小值1
page_size: int,    // int, 选填, 每页数量, 默认20, 最小值1, 最大值100
role: uint8,       // uint8, 选填, 角色筛选 (1=普通用户, 2=管理员)
status: uint8      // uint8, 选填, 状态筛选 (1=正常, 0=禁用)
```

#### 4.1.8 POST `/api/v1/user/password` - 修改密码

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**请求体 (JSON)**:
```json
{
    "user_id": "uint",            // uint, 必填, 用户ID (从Token解析)
    "old_password": "string",    // string, 必填, 旧密码
    "new_password": "string"     // string, 必填, 新密码, 长度6-32字符
}
```

#### 4.1.9 POST `/api/v1/user/validate` - 验证Token

**请求体 (JSON)**:
```json
{
    "token": "string"             // string, 必填, JWT Token
}
```

#### 4.1.10 POST `/api/v1/user/blacklist` - 添加黑名单

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**请求体 (JSON)**:
```json
{
    "user_id": "uint",            // uint, 必填, 用户ID (当前登录用户)
    "target_user_id": "uint",     // uint, 必填, 目标用户ID (被拉黑的用户)
    "reason": "string"            // string, 选填, 拉黑原因
}
```

#### 4.1.11 DELETE `/api/v1/user/blacklist` - 移除黑名单

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**请求体 (JSON)**:
```json
{
    "user_id": "uint",            // uint, 必填, 用户ID
    "target_user_id": "uint"      // uint, 必填, 目标用户ID
}
```

#### 4.1.12 GET `/api/v1/user/blacklist/check` - 检查黑名单

**查询参数 (Query Parameters)**:
```
user_id: int,           // int, 必填, 用户ID
target_user_id: int     // int, 必填, 目标用户ID
```

#### 4.1.13 GET `/health` - 健康检查

**响应**: 返回服务健康状态

#### 4.1.14 GET `/ready` - 就绪探针

**响应**: 返回服务就绪状态

### 4.2 gRPC API

```protobuf
service UserService {
    rpc Register(RegisterRequest) returns (RegisterResponse);
    rpc Login(LoginRequest) returns (LoginResponse);
    rpc Logout(LogoutRequest) returns (LogoutResponse);
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
    rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
    rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
    rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
    rpc GetUsers(GetUsersRequest) returns (GetUsersResponse);
    rpc ChangePassword(ChangePasswordRequest) returns (ChangePasswordResponse);
    rpc AddToBlacklist(BlacklistRequest) returns (BlacklistResponse);
    rpc RemoveFromBlacklist(BlacklistRequest) returns (BlacklistResponse);
    rpc IsInBlacklist(IsBlacklistRequest) returns (IsBlacklistResponse);
}
```

## 五、API 流程图

### 5.1 用户注册流程

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Client │────▶│  Register   │────▶│   Validate   │────▶│   Check     │
│         │     │  Handler    │     │   Input      │     │  Existence  │
└─────────┘     └─────────────┘     └──────────────┘     └──────┬──────┘
                                                                   │
                        ┌─────────────────────────────────────────┘
                        │ (用户已存在)
                        ▼
                   ┌─────────────┐
                   │   Return    │──▶ 400: 用户已存在
                   │   Error     │
                   └─────────────┘

                        ┌─────────────────────────────────────────┐
                        │ (用户不存在)                            │
                        ▼
                   ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
                   │   Hash      │────▶│   Create     │────▶│  Generate   │
                   │   Password  │     │   User       │     │    JWT      │
                   └─────────────┘     └──────────────┘     └──────┬──────┘
                                                                     │
                        ┌────────────────────────────────────────────┘
                        ▼
                   ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
                   │   Save      │────▶│   Add to     │────▶│   Return    │
                   │   Token     │     │   BloomFilter│     │   Success   │
                   └─────────────┘     └──────────────┘     └─────────────┘
                                                              │
                   ┌──────────────────────────────────────────┘
                   ▼
            ┌─────────────┐
            │  Response:  │
            │  {user,    │
            │   token,   │
            │   csrf_   │
            │   token}   │
            └─────────────┘
```

> **注意**：`RegisterResponse` 和 `LoginResponse` 均包含 `csrf_token` 字段（`user.proto` 字段 5），客户端需将其存入 `localStorage.csrf_token`，后续写请求需携带 `X-CSRF-Token` 请求头。

### 5.2 用户登录流程

```
┌─────────┐     ┌───────────┐     ┌──────────────┐     ┌───────────────┐
│  Client │────▶│   Login   │────▶│   Validate   │────▶│    Get        │
│         │     │  Handler  │     │   Input      │     │    User       │
└─────────┘     └───────────┘     └──────────────┘     └───────┬───────┘
                                                               │
                        ┌──────────────────────────────────────┘
                        ▼
                   ┌─────────────┐     ┌──────────────┐     ┌───────────────┐
                   │   Check     │────▶│   Verify      │────▶│   Generate    │
                   │   Status   │     │   Password    │     │     JWT       │
                   └─────────────┘     └──────────────┘     └───────┬───────┘
                                                                      │
                        ┌─────────────────────────────────────────────┘
                        ▼
                   ┌─────────────┐     ┌──────────────┐     ┌───────────────┐
                   │   Save      │────▶│   Add to     │────▶│   Return      │
                   │   Token     │     │   BloomFilter│     │   Success     │
                   └─────────────┘     └──────────────┘     └───────┬───────┘
                                                                    │
                        ┌───────────────────────────────────────────┘
                        ▼
                 ┌─────────────┐
                 │  Response:  │
                 │  {user,    │
                 │   token,   │
                 │   csrf_   │
                 │   token}   │
                 └─────────────┘
```

## 六、Prometheus Metrics

| 指标名称 | 类型 | 标签 | 描述 |
|----------|------|------|------|
| `http_requests_total` | Counter | method, endpoint, status | HTTP请求总数 |
| `http_request_duration_seconds` | Histogram | method, endpoint | HTTP请求延迟 |
| `rpc_requests_total` | Counter | service, method, status | RPC请求总数 |
| `rpc_request_duration_seconds` | Histogram | service, method | RPC请求延迟 |
| `requests_in_flight` | Gauge | - | 当前处理中的请求数 |
| `request_duration_seconds` | Histogram | method, endpoint | 请求总延迟 |
| `errors_total` | Counter | type | 错误总数 |
| `cpu_usage_percent` | Gauge | - | CPU使用率 |
| `memory_usage_bytes` | Gauge | - | 内存使用量 |
| `goroutine_count` | Gauge | - | Goroutine数量 |
| `panic_counter_total` | Counter | service | Panic次数 |
| `mysql_slow_queries_total` | Counter | - | MySQL慢查询数 |
| `redis_cache_hits_total` | Counter | - | Redis缓存命中次数 |
| `redis_cache_misses_total` | Counter | - | Redis缓存未命中次数 |
| Redis命中率(PromQL) | - | - | `sum(rate(redis_cache_hits_total[5m])) / clamp_min(sum(rate(redis_cache_hits_total[5m])) + sum(rate(redis_cache_misses_total[5m])), 0)` |
| `redis_hot_keys_total` | Counter | key | 热键访问次数 |
| `cache_operations_total` | Counter | operation, status | 缓存操作数 |
| `db_operations_total` | Counter | operation, status | 数据库操作数 |
| `service_health` | Gauge | service | 服务健康状态 |

## 七、高并发特性

### 7.1 熔断器 (Circuit Breaker)

#### 原理详解

熔断器模式源自电路保险丝，当电流过大时自动断开以保护电路。分布式系统中的熔断器保护的是服务调用链。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           熔断器状态机                                       │
│                                                                             │
│    ┌──────────┐     失败率>阈值      ┌─────────┐     超时后探测    ┌─────────┐
│    │  CLOSED  │ ─────────────────▶ │  OPEN   │ ───────────────▶ │ HALF-   │
│    │  关闭    │                    │  熔断   │                  │ OPEN    │
│    │  正常    │                    │  拒绝   │                  │ 半开    │
│    └──────────┘                    │  请求   │                  └────┬────┘
│           ▲                        └─────────┘                       │     │
│           │                             │                             │     │
│           │      成功后转为关闭          │                             │     │
│           └─────────────────────────────┘                             │     │
│                                                                            │     │
│                                  失败率>阈值 ◀─────────────────────────────┘
│                                                                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

**状态说明**：

| 状态 | 行为 | 说明 |
|------|------|------|
| **CLOSED (关闭)** | 正常执行请求 | 当失败率低于阈值时，服务正常运行 |
| **OPEN (打开)** | 快速失败，拒绝请求 | 当失败率超过阈值时，立即拒绝请求，返回错误 |
| **HALF-OPEN (半开)** | 允许有限请求通过 | 探测下游服务是否恢复，决定切换到 CLOSED 或 OPEN |

**gobreaker 工作原理**：

```go
// gobreaker 内部状态机
type State int

const (
    StateClosed   State = iota  // 0: 关闭状态
    StateOpen                   // 1: 打开状态(熔断中)
    StateHalfOpen               // 2: 半开状态(探测恢复)
)

// 熔断器核心计数器
type counts struct {
    requests             uint32        // 当前窗口请求数
    success              uint32        // 成功请求数
    consecutiveSuccesses uint32        // 连续成功次数
    consecutiveFailures  uint32        // 连续失败次数
}
```

**熔断触发条件**：

```go
// gobreaker.Settings 配置项
Settings{
    Name:        "user-service",           // 熔断器名称
    MaxRequests: 3,                         // 半开状态允许的最大请求数
    Interval:    10 * time.Second,          // 统计窗口周期
    Timeout:     30 * time.Second,          // 熔断持续时间
}

// 熔断逻辑伪代码
func shouldTrip(failureRatio float64) bool {
    // 条件1: 连续失败 >= MinRequests
    // 条件2: 失败率 > failureRatio (默认50%)
    return consecutiveFailures >= MinRequests && 
           float64(consecutiveFailures)/float64(requests) > failureRatio
}
```

**熔断器解决的问题**：

1. **防止雪崩效应**：当下游服务不可用时，快速失败避免请求堆积
2. **资源保护**：限制对故障服务的请求，释放线程池资源
3. **故障隔离**：单个服务故障不会扩散到整个系统

#### 配置示例

```go
cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "user-service",
    MaxRequests: 3,              // 半开状态最多放行3个请求进行探测
    Interval:    10 * time.Second, // 每10秒重置计数器
    Timeout:     30 * time.Second, // 熔断30秒后进入半开状态
})

// 熔断器调用方式
result, err := cb.Execute(func() (interface{}, error) {
    // 执行可能失败的操作
    return userService.GetUser(id)
})
```

### 7.2 限流 (Rate Limiting)

#### 原理详解

限流是保护系统的第一道防线，通过控制请求速率防止系统过载。本服务使用 **令牌桶算法**。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         令牌桶算法原理                                        │
│                                                                             │
│                         ┌─────────────┐                                     │
│                         │   令牌桶    │  容量 = Burst = 10                  │
│                         │  ┌──┬──┬──┐ │                                     │
│                         │  │  │  │  │ │ ← 桶中存储令牌                      │
│                         │  │  │  │  │ │                                     │
│                         │  └──┴──┴──┘ │                                     │
│                         └──────┬──────┘                                     │
│                                │                                            │
│            每秒补充 QPS 个令牌   │   每次请求消耗1个令牌                       │
│                   ▲             │             ▼                             │
│                   │             │             │                             │
│  ┌────────┐      │      ┌──────┴──────┐      │      ┌────────────┐         │
│  │ 定时器 │───────┼─────▶│   令牌补充   │──────┼─────▶│  桶满丢弃  │         │
│  └────────┘      │      │  rate.Add()  │      │      │  桶空拒绝  │         │
│                  │      └──────────────┘      │      └─────┬──────┘         │
│                  │                             │            │                │
│                  │                             │      桶中有令牌              │
│                  │                             │            ▼                │
│                  │                             │      ┌────────────┐         │
│                  │                             └─────▶│   通过     │         │
│                  │                                   └────────────┘         │
│                  │                                                               │
│  假设 QPS=5, Burst=10:                                                        │
│  - 系统启动时桶满(10个令牌)                                                     │
│  - 瞬间可处理10个请求(突发能力)                                                 │
│  - 之后每秒补充5个令牌                                                          │
└─────────────────────────────────────────────────────────────────────────────┘
```

**算法对比**：

| 算法 | 令牌桶 | 滑动窗口 | 固定窗口 | 漏桶 |
|------|--------|----------|----------|------|
| **突发能力** | ✅ 支持 | ❌ 不支持 | ✅ 支持 | ❌ 不支持 |
| **平滑限流** | ✅ 平滑 | ✅ 平滑 | ❌ 临界突变 | ✅ 平滑 |
| **实现复杂度** | 中等 | 高 | 低 | 中等 |

**Go 实现原理**：

```go
// golang.org/x/time/rate 令牌桶实现
type Limiter struct {
    limit Limit      // 每秒产生的令牌数 (QPS)
    burst int        // 桶的容量 (Burst)
    
    mu     sync.Mutex
    tokens float64   // 当前桶中令牌数
    last   time.Time // 上次更新时间
    lastEvent int    // 上次拒绝事件
}

// Allow() 判断是否允许通过
func (lim *Limiter) Allow() bool {
    return lim.AllowN(time.Now(), 1)
}

func (lim *Limiter) AllowN(now time.Time, n int) bool {
    lim.mu.Lock()
    defer lim.mu.Unlock()
    
    // 1. 先补充令牌 (基于时间流逝)
    last := now
    lim.mu.b = last
    tokens := lim.AllowN(last, 1) // 计算补充的令牌数
    // 补充公式: tokens += (now - last).Seconds() * float64(lim.limit)
    
    // 2. 检查令牌是否足够
    if tokens >= float64(n) {
        lim.tokens -= float64(n)
        return true  // 允许通过
    }
    return false  // 拒绝
}
```

**IP 级别限流实现**：

```go
// 为每个 IP 维护独立的限流器
type IPRateLimiter struct {
    limiters sync.Map  // map[string]*rate.Limiter
    qps      rate.Limit
    burst    int
}

func (m *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
    limiter, ok := m.limiters.Load(ip)
    if !ok {
        // 首次访问，为该 IP 创建限流器
        limiter = rate.NewLimiter(m.qps, m.burst)
        m.limiters.Store(ip, limiter)
    }
    return limiter.(*rate.Limiter)
}

// 中间件中使用
func RateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        limiter := ipLimiter.getLimiter(ip)
        
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "请求过于频繁"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**限流策略选择**：

| 场景 | 推荐算法 | 原因 |
|------|----------|------|
| API 限流 | 令牌桶 | 支持突发流量，用户体验好 |
| 登录接口 | 滑动窗口 | 更精确的恶意请求限制 |
| 文件上传 | 漏桶 | 恒定速率，保护带宽 |
| 支付接口 | 固定窗口+计数器 | 实现简单，够用 |

#### 配置示例

```go
commonmiddleware.InitRateLimiter(&commonconfig.RateLimitConfig{
    Enabled: cfg.RateLimit.Enabled,
    QPS:     int(cfg.RateLimit.QPS),   // 每秒允许的请求数
    Burst:   cfg.RateLimit.Burst,       // 突发容量
})
```

### 7.3 HTTP Server 超时配置

#### 原理详解

HTTP Server 超时配置是防止资源泄漏和连接堆积的关键。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         HTTP 连接生命周期                                    │
│                                                                             │
│  连接建立                                                                    │
│      │                                                                      │
│      ▼                                                                      │
│  ┌─────────────────┐                                                       │
│  │ ReadHeaderTime  │ ◀── 读取请求头超时 (10s)                               │
│  └────────┬────────┘                                                       │
│           │                                                                │
│           ▼                                                                │
│  ┌─────────────────┐                                                       │
│  │  ReadTimeout    │ ◀── 读取请求体超时 (30s)                               │
│  │  (包含Header)    │     (上传大文件时尤为重要)                             │
│  └────────┬────────┘                                                       │
│           │                                                                │
│           ▼                                                                │
│  ┌─────────────────┐                                                       │
│  │   Handler执行    │ ◀── 请求处理时间                                      │
│  └────────┬────────┘                                                       │
│           │                                                                │
│           ▼                                                                │
│  ┌─────────────────┐                                                       │
│  │  WriteTimeout   │ ◀── 写入响应超时 (30s)                                │
│  │  (包含Header)    │     (慢客户端防护)                                     │
│  └────────┬────────┘                                                       │
│           │                                                                │
│           ▼                                                                │
│  ┌─────────────────┐                                                       │
│  │   IdleTimeout   │ ◀── 空闲连接超时 (120s)                                │
│  │  (Keep-Alive)    │     (释放空闲连接资源)                                 │
│  └────────┬────────┘                                                       │
│           │                                                                │
│           ▼                                                                │
│      连接关闭                                                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**超时设置原则**：

| 超时类型 | 设置值 | 原因 |
|----------|--------|------|
| `ReadHeaderTimeout` | 10s | 请求头通常很小，10s足够 |
| `ReadTimeout` | 30s | 考虑大请求体(上传文件) |
| `WriteTimeout` | 30s | 考虑慢客户端和网络延迟 |
| `IdleTimeout` | 120s | 减少连接维护开销 |

**为什么需要 ReadHeaderTimeout**：
- `ReadTimeout` 包括了读取 headers 的时间
- 如果只设置 ReadTimeout，攻击者可以**缓慢发送请求头**，占用连接直到超时
- `ReadHeaderTimeout` 专门限制 headers 读取时间

#### 配置示例

```go
s.httpServer = &http.Server{
    ReadTimeout:       30 * time.Second,   // 读取请求超时
    ReadHeaderTimeout: 10 * time.Second,   // 读取请求头超时
    WriteTimeout:      30 * time.Second,   // 写入响应超时
    IdleTimeout:       120 * time.Second,  // 空闲连接超时
    MaxHeaderBytes:    1 << 20,            // 1MB最大请求头
}
```

### 7.4 gRPC Keepalive 原理

#### 原理详解

gRPC Keepalive 用于检测连接健康状态和防止中间设备关闭空闲连接。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         gRPC Keepalive 工作机制                               │
│                                                                             │
│   Client                         Server                                      │
│     │                              │                                         │
│     │  PING frame (Probe)          │                                         │
│     │─────────────────────────────▶│                                         │
│     │                              │                                         │
│     │  PING ACK                    │                                         │
│     │◀─────────────────────────────│                                         │
│     │                              │                                         │
│     │  [数据传输]                   │                                         │
│     │─────────────────────────────▶│                                         │
│     │                              │                                         │
│     │  [数据传输]                   │                                         │
│     │◀─────────────────────────────│                                         │
│     │                              │                                         │
│     │                              │                                         │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    Keepalive 参数说明                                  │  │
│  │                                                                       │  │
│  │  MaxConnectionIdle    │ 空闲多久关闭连接 (5min)                       │  │
│  │  MaxConnectionAge     │ 连接最大存活时间 (10min)                      │  │
│  │  Time                 │ 客户端发送 PING 间隔 (2h)                      │  │
│  │  Timeout             │ PING 响应超时 (20s)                             │  │
│  │  MinTime             │ 服务端允许的最小 PING 间隔 (5min)               │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Keepalive 参数作用**：

| 参数 | 方向 | 作用 |
|------|------|------|
| `MaxConnectionIdle` | Server | 空闲连接超时后关闭 |
| `MaxConnectionAge` | Server | 强制关闭存活过长的连接 |
| `Time` | Client→Server | 探测连接的存活状态 |
| `Timeout` | Client→Server | 等待 ACK 的超时时间 |
| `PermitWithoutStream` | Server | 允许在没有流时响应 PING |

**为什么需要 Keepalive**：

1. **NAT 超时**：网络地址转换会清理长时间空闲的映射
2. **防火墙超时**：防火墙会关闭长时间空闲的 TCP 连接
3. **负载均衡器**：LB 也会清理空闲连接
4. **检测死连接**：发现对端崩溃的连接

#### 配置示例

```go
grpcOpts := []grpc.ServerOption{
    grpc.KeepaliveParams(keepalive.ServerParameters{
        MaxConnectionIdle:     5 * time.Minute,   // 空闲5分钟后关闭
        MaxConnectionAge:      10 * time.Minute,  // 10分钟后强制重置
        MaxConnectionAgeGrace: 30 * time.Second,   // 优雅关闭宽限期
        Time:                  2 * time.Hour,     // 客户端2小时探测一次
        Timeout:               20 * time.Second,   // PING响应超时
    }),
    grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
        MinTime:             5 * time.Minute,     // 客户端至少5分钟发一次
        PermitWithoutStream: true,                 // 无流时也响应PING
    }),
    grpc.UnaryInterceptor(s.grpcUnaryInterceptor),
}
```

## 八、中间件链（gRPC 拦截器 + 探活 HTTP）

业务层纯 gRPC，HTTP 仅保留探活端口（无 gin 业务路由）。

```text
gRPC 拦截器链（grpc.ChainUnaryInterceptor）：
  grpcUnaryInterceptor       10s 超时 + gobreaker 熔断
  → GRPCAuthInterceptor()    从 metadata 校验 JWT，注入 user_id/role/username 到 ctx
  → GRPCMetricsInterceptor() RPC 级 QPS/延迟/错误
  → GRPCLoggingInterceptor() 记录 RPC 方法/对端/耗时/错误

handler 内鉴权：
  Register/SendCode/Login                 公开（注册/发验证码/登录），RequireGRPCAuth 不强制
  其余用户方法                            RequireGRPCAuth(ctx)  // 已登录用户
  AdminGetUsers/AdminUpdateUser/AdminDeleteUser/ListOperationLogs
                                        requireGRPCAdmin()  // 仅管理员（RoleAdmin）
```

探活 HTTP（`runHTTPServer`，`net/http` mux，无业务路由）：`/health`、`/ready`、`/version`。

## 九、数据库模型

### 9.1 users 表

| 字段 | 类型 | 约束 | 描述 |
|------|------|------|------|
| id | INT UNSIGNED | PRIMARY KEY, AUTO_INCREMENT | 用户ID |
| username | VARCHAR(64) | NOT NULL, UNIQUE INDEX | 用户名 |
| email | VARCHAR(128) | NOT NULL, UNIQUE INDEX | 邮箱 |
| password | VARCHAR(256) | NOT NULL | 密码(bcrypt加密) |
| nickname | VARCHAR(64) | - | 昵称 |
| avatar | VARCHAR(256) | - | 头像URL |
| bio | TEXT | - | 个人简介 |
| role | TINYINT UNSIGNED | DEFAULT 1 | 角色(1=用户, 2=编辑, 99=管理员) |
| status | TINYINT UNSIGNED | DEFAULT 1 | 状态(1=正常, 0=禁用) |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | 更新时间 |

**索引信息**:
| 索引名 | 类型 | 列 | 唯一 | 说明 |
|--------|------|-----|------|------|
| PRIMARY | 主键 | id | - | 主键索引 |
| idx_username | 普通 | username | 否 | 用户名查询 |
| idx_email | 普通 | email | 否 | 邮箱查询 |
| idx_created_at | 普通 | created_at | 否 | 创建时间排序 |
| - | 唯一 | username | 是 | GORM自动创建 |
| - | 唯一 | email | 是 | GORM自动创建 |

### 9.2 tokens 表

| 字段 | 类型 | 约束 | 描述 |
|------|------|------|------|
| id | INT UNSIGNED | PRIMARY KEY, AUTO_INCREMENT | 记录ID |
| user_id | INT UNSIGNED | NOT NULL, INDEX | 用户ID |
| token | VARCHAR(512) | NOT NULL, UNIQUE | JWT Token |
| expires_at | TIMESTAMP | NOT NULL, INDEX | 过期时间 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 创建时间 |

**索引信息**:
| 索引名 | 类型 | 列 | 唯一 | 说明 |
|--------|------|-----|------|------|
| PRIMARY | 主键 | id | - | 主键索引 |
| idx_token | 普通 | token | - | Token查询 |
| idx_user_id | 普通 | user_id | 否 | 用户ID查询(登出时删除用户所有Token) |
| idx_expires_at | 普通 | expires_at | 否 | 过期Token清理 |
| - | 唯一 | token | 是 | Token唯一性 |

### 9.3 user_blacklists 表

| 字段 | 类型 | 约束 | 描述 |
|------|------|------|------|
| id | INT UNSIGNED | PRIMARY KEY, AUTO_INCREMENT | 记录ID |
| user_id | INT UNSIGNED | NOT NULL, INDEX | 用户ID |
| blocked_user_id | INT UNSIGNED | NOT NULL, INDEX | 被拉黑用户ID |
| reason | VARCHAR(256) | - | 拉黑原因 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 创建时间 |

**索引信息**:
| 索引名 | 类型 | 列 | 唯一 | 说明 |
|--------|------|-----|------|------|
| PRIMARY | 主键 | id | - | 主键索引 |
| idx_user_id | 普通 | user_id | 否 | 用户ID查询(获取用户的黑名单列表) |
| idx_blocked_user_id | 普通 | blocked_user_id | 否 | 被拉黑用户ID查询 |
| uk_user_blocked | 唯一 | (user_id, blocked_user_id) | 是 | 防止重复添加同一用户到黑名单 |

## 十、GORM 原理

### 10.1 GORM 架构概览

GORM 是 Go 语言中最流行的 ORM 库之一，采用分层架构设计，将用户操作转换为 SQL 执行。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              GORM 架构分层                                   │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        应用层 (Application Layer)                       │ │
│  │                                                                        │ │
│  │   user := &User{}                                                      │ │
│  │   db.First(user, 1)                                                    │ │
│  │   db.Create(&User{Name: "test"})                                       │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                    链式 API (Chainable API)                            │ │
│  │                                                                        │ │
│  │   db.Where("name = ?", "test").                                       │ │
│  │      Order("created_at desc").                                         │ │
│  │      Preload("Orders").                                               │ │
│  │      Find(&users)                                                      │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                     核心层 (Core Layer)                                 │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │ │
│  │  │    Dialector   │  │    Clause      │  │    Callback     │         │ │
│  │  │   数据库适配器   │  │    SQL构建器    │  │    钩子函数      │         │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘         │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                      数据库驱动层 (Driver Layer)                        │ │
│  │                                                                        │ │
│  │   MySQL: go-sql-driver/mysql                                           │ │
│  │   PostgreSQL: lib/pq                                                   │ │
│  │   SQLite: modernc.org/sqlite                                           │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                          数据库 (Database)                              │ │
│  │                                                                        │ │
│  │                        MySQL / PostgreSQL / SQLite                      │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.2 SQL 生成原理

#### 10.2.1 链式调用原理

GORM 的链式 API 基于 Go 的方法链模式，每个方法返回 `*gorm.DB`，可以继续调用其他方法。

```go
// GORM DB 结构
type DB struct {
    Error        error
    RowsAffected int64
    
    // 内部状态
    Statement    *Statement
    Config       *Config
    Clone        *bool
    // ... 其他字段
}

// 链式调用示例
func (db *DB) Where(query interface{}, args ...interface{}) *DB {
    return db.clone().Session(&gorm.Session{}).Where(query, args...)
}

func (db *DB) Order(value interface{}) *DB {
    return db.clone().Session(&gorm.Session{}).Order(value)
}

func (db *DB) Find(dest interface{}, conds ...interface{}) *DB {
    return db.session(func(tx *DB) *DB {
        return tx.CallFunction(func(tx *DB) *DB {
            return tx.NewRecord(dest)
        }, dest, conds...)
    })
}
```

#### 10.2.2 SQL 构建流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GORM SQL 构建流程                                    │
│                                                                             │
│  1. 用户调用                                                                 │
│  ──────────                                                                 │
│                                                                             │
│  db.Select("id, name").                                                     │
│     Where("age > ?", 18).                                                   │
│     Order("created_at desc").                                               │
│     Limit(10).                                                              │
│     Find(&users)                                                            │
│                                                                             │
│                                    │                                        │
│                                    ▼                                        │
│  2. 构建 Statement                                                           │
│  ────────────────                                                            │
│                                                                             │
│  Statement {                                                                │
│      Schema:     User{}         // 模型结构体                                │
│      Selects:    ["id", "name"] // SELECT 字段                              │
│      Clause:     {where: [...], order: [...], limit: [10]}                 │
│      TableOpts:  {}              // 表选项                                  │
│  }                                                                           │
│                                                                             │
│                                    │                                        │
│                                    ▼                                        │
│  3. 注册 Callback                                                            │
│  ────────────────                                                            │
│                                                                             │
│  callbacks.Create() → callbacks.Query() → callbacks.Update() → ...          │
│                                                                             │
│                                    │                                        │
│                                    ▼                                        │
│  4. 执行 SQL                                                                 │
│  ──────────                                                                 │
│                                                                             │
│  SELECT id, name FROM users WHERE age > 18 ORDER BY created_at DESC LIMIT 10 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.3 模型定义与映射

#### 10.3.1 Tag 解析机制

GORM 通过反射解析结构体的字段和 Tag，生成数据库表的映射关系。

```go
// 用户模型定义
type User struct {
    ID        uint      `gorm:"primaryKey;autoIncrement"`
    Username  string    `gorm:"type:varchar(64);uniqueIndex;not null"`
    Email     string    `gorm:"type:varchar(128);uniqueIndex"`
    Password  string    `gorm:"not null"`
    Nickname  string    `gorm:"type:varchar(64)"`
    Role      uint8     `gorm:"default:1;comment:角色"`
    Status    uint8     `gorm:"default:1;comment:状态"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
}
```

#### 10.3.2 Tag 解析流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GORM Tag 解析流程                                    │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        反射获取结构体                                   │ │
│  │                                                                        │ │
│  │   t := reflect.TypeOf(User{})                                          │ │
│  │   for i := 0; i < t.NumField(); i++ {                                  │ │
│  │       field := t.Field(i)                                               │ │
│  │       tag := field.Tag.Get("gorm")                                      │ │
│  │   }                                                                     │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        Tag 解析器                                      │ │
│  │                                                                        │ │
│  │   // Tag 格式: "key1:value1;key2:value2"                               │ │
│  │   // 例如: "type:varchar(64);uniqueIndex;not null"                     │ │
│  │                                                                        │ │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                    │ │
│  │   │  type       │  │ uniqueIndex │  │  not null   │                    │ │
│  │   │  varchar    │  │  创建唯一索引 │  │  非空约束    │                    │ │
│  │   └─────────────┘  └─────────────┘  └─────────────┘                    │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        生成 Schema                                     │ │
│  │                                                                        │ │
│  │   Schema {                                                              │ │
│  │       Table: "users",                                                   │ │
│  │       Fields: [                                                         │ │
│  │           {Name: "ID", DBName: "id", Type: "uint", PK: true},          │ │
│  │           {Name: "Username", DBName: "username", Type: "varchar(64)"},│ │
│  │           ...                                                           │ │
│  │       ],                                                                │ │
│  │       Relations: {}                                                     │ │
│  │   }                                                                     │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.4 数据库操作原理

#### 10.4.1 查询操作 (Find)

```go
// 用户代码
var users []User
db.Where("status = ?", 1).Find(&users)

// 内部执行流程
func (db *DB) Find(dest interface{}, conds ...interface{}) *DB {
    return db.Session(&gorm.Session{}). callback.Find → Generate SQL
}
```

#### 10.4.2 创建操作 (Create)

```go
// 用户代码
user := User{Username: "test", Email: "test@example.com"}
db.Create(&user)

// 内部执行流程
func (db *DB) Create(value interface{}, opts ...Option) *DB {
    return db.Session(&gorm.Session{}). callback.Create → Generate INSERT SQL
}
```

#### 10.4.3 更新操作 (Update)

```go
// 用户代码
db.Model(&user).Where("status = ?", 1).Update("name", "new_name")

// 内部执行流程
func (db *DB) Update(column string, value interface{}) *DB {
    return db.Session(&gorm.Session{}). callback.Update → Generate UPDATE SQL
}
```

### 10.5 钩子函数 (Hooks) 原理

#### 10.5.1 钩子类型

GORM 支持在 CRUD 操作前后执行钩子函数：

```go
type User struct {
    Name string
}

// 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
    // 修改值
    u.Name = "prefix_" + u.Name
    return nil
}

// 创建后钩子
func (u *User) AfterCreate(tx *gorm.DB) error {
    // 发送通知等
    return nil
}
```

#### 10.5.2 钩子执行时机

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         钩子函数执行时机                                     │
│                                                                             │
│  CREATE 操作:                                                                │
│  ──────────                                                                 │
│  BeforeSave → BeforeCreate → [INSERT] → AfterCreate → AfterSave             │
│                                                                             │
│  UPDATE 操作:                                                                │
│  ──────────                                                                 │
│  BeforeSave → BeforeUpdate → [UPDATE] → AfterUpdate → AfterSave            │
│                                                                             │
│  DELETE 操作:                                                                │
│  ──────────                                                                 │
│  BeforeDelete → [DELETE] → AfterDelete                                       │
│                                                                             │
│  QUERY 操作:                                                                 │
│  ──────────                                                                 │
│  [SELECT] → AfterFind                                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.6 关联处理原理

#### 10.6.1 预加载 (Preload)

```go
// 预加载用户的文章
var users []User
db.Preload("Articles").Find(&users)

// 生成的 SQL:
// SELECT * FROM users;
// SELECT * FROM articles WHERE user_id IN (1, 2, 3, ...);
```

#### 10.6.2 关联模式

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GORM 关联处理                                        │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        一对多 (HasMany)                                │ │
│  │                                                                        │ │
│  │   User ──────< Article                                                │ │
│  │   db.Preload("Articles").Find(&users)                                 │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        多对多 (Many2Many)                              │ │
│  │                                                                        │ │
│  │   Article ───── ArticleTag ───── Tag                                  │ │
│  │   db.Preload("Tags").Find(&articles)                                 │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.7 事务处理

#### 10.7.1 事务 API

```go
// 方式1: Transaction 方法
db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    if err := tx.Create(&profile).Error; err != nil {
        return err
    }
    return nil
})

// 方式2: 手动控制
tx := db.Begin()
err := tx.Create(&user).Error
if err != nil {
    tx.Rollback()
    return err
}
tx.Commit()
```

#### 10.7.2 事务传播

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         事务传播机制                                         │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                                                                        │ │
│  │   外层事务                                                             │ │
│  │   ┌─────────────────────────────────────────────────────────────────┐  │ │
│  │   │                                                                  │  │ │
│  │   │   tx := db.Begin()                                              │  │ │
│  │   │                                                                  │  │ │
│  │   │   ┌─────────────────────────────────────────────────────────┐   │  │ │
│  │   │   │  内层事务 (使用同一个 tx)                                  │   │  │ │
│  │   │   │  如果内层回滚: 只回滚内层操作                              │   │  │ │
│  │   │   │  如果内层提交: 加入外层事务                               │   │  │ │
│  │   │   └─────────────────────────────────────────────────────────┘   │  │ │
│  │   │                                                                  │  │ │
│  │   │   tx.Commit()                                                   │  │ │
│  │   │                                                                  │  │ │
│  │   └─────────────────────────────────────────────────────────────────┘  │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.8 性能优化建议

| 优化项 | 说明 | 示例 |
|--------|------|------|
| 使用 `Select` 限制字段 | 减少数据传输 | `db.Select("id, name").Find(&users)` |
| 使用 `Limit` 限制数量 | 避免全表扫描 | `db.Limit(100).Find(&users)` |
| 使用索引列查询 | 确保查询走索引 | `Where("email = ?", email)` |
| 使用 `Preload` 代替 `Joins` | 避免 N+1 问题 | `Preload("Articles")` |
| 使用 `First`/`Take` 代替 `Find` | 单条查询更高效 | `db.First(&user, id)` |
| 批量插入使用 `CreateInBatches` | 减少 SQL 执行次数 | `db.CreateInBatches(users, 100)` |
| 使用 `Unscoped` 删除大量数据 | 避免逐条删除 | `db.Unscoped().Where("status = ?", 0).Delete(&users)` |

## 十一、API SQL 与索引分析

### 11.1 用户注册 (Register)

**执行的SQL**:
```sql
-- 检查用户是否存在 (使用 OR 条件)
SELECT COUNT(*) FROM users WHERE username = ? OR email = ? LIMIT 1
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `username = ?` | idx_username | 使用用户名索引 |
| `email = ?` | idx_email | 使用邮箱索引 |

**说明**: MySQL 会选择其中一个索引，由于使用了 OR 条件，无法同时利用两个索引。

---

### 11.2 用户登录 (Login)

**执行的SQL**:
```sql
-- 方式1: 根据用户名查询
SELECT * FROM users WHERE username = ? LIMIT 1

-- 方式2: 根据邮箱查询 (如果用户名查询失败)
SELECT * FROM users WHERE email = ? LIMIT 1
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `username = ?` | idx_username | 使用用户名索引 |
| `email = ?` | idx_email | 使用邮箱索引 |

**说明**: 两次查询都是精确匹配，索引效率高。

---

### 11.3 获取用户信息 (GetUser)

**执行的SQL**:
```sql
-- 根据ID查询
SELECT * FROM users WHERE id = ? LIMIT 1

-- 或根据用户名查询
SELECT * FROM users WHERE username = ? LIMIT 1
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `id = ?` | PRIMARY (id) | 主键索引，最高效 |
| `username = ?` | idx_username | 用户名索引 |

---

### 11.4 获取用户列表 (GetUsers)

**执行的SQL**:
```sql
-- 统计总数 (带角色筛选)
SELECT COUNT(*) FROM users WHERE role = ?

-- 分页查询
SELECT * FROM users WHERE role = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `role = ?` | idx_created_at (辅助) | role无索引，MySQL可能全表扫描 |
| `ORDER BY created_at` | idx_created_at | 使用创建时间索引优化排序 |

**优化建议**: 如果 role 筛选是高频场景，建议添加 `INDEX idx_role (role)`。

---

### 11.5 Token 操作

**登录时创建Token**:
```sql
INSERT INTO tokens (user_id, token, expires_at) VALUES (?, ?, ?)
```

**验证Token**:
```sql
SELECT * FROM tokens WHERE token = ? LIMIT 1
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `token = ?` | idx_token | Token索引，唯一查找 |

---

### 11.6 黑名单操作

**检查是否在黑名单**:
```sql
SELECT COUNT(*) FROM user_blacklists WHERE user_id = ? AND blocked_user_id = ?
```

**添加黑名单**:
```sql
INSERT INTO user_blacklists (user_id, blocked_user_id, reason) VALUES (?, ?, ?)
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `user_id = ? AND blocked_user_id = ?` | uk_user_blocked | 唯一索引，两个等值查询 |
| - | idx_user_id | 辅助索引 |
| - | idx_blocked_user_id | 辅助索引 |

**说明**: `uk_user_blocked` 是复合唯一索引，两个等值条件查询可以直接命中。

## 进程退出与资源释放

服务在 `cmd/server/main.go` 中统一处理退出流程：监听 `SIGINT`/`SIGTERM`，由 `Server.Run()` 优雅关闭 gRPC 与探活 HTTP server（10s 超时），随后调用 `shutdown()` 集合方法按固定顺序释放其余资源：

1. **摘除流量**：从 Consul 注销（`deregister`），让网关停止转发新请求；
2. **释放连接**：关闭 Redis 连接池（`cache.Close`）→ 关闭数据库连接池（`database.Close`）；
3. **停热更/指标**：停止配置中心热更监听（`HotConfig.Stop`）→ 取消指标采集 context（`metricsCancel`）；
4. **停日志**：最后 `log.StopRotation()` flush 并关闭日志文件。

> 所有释放集中在 `shutdown()` 一处便于审计，新增需释放的资源只需在此追加，避免分散 `defer` 导致顺序混乱或重复释放。

### 异常退出兜底（panic / 初始化失败）

除上述正常 `SIGINT/SIGTERM` 路径外，本服务对两类异常退出也做了资源兜底：

- **初始化失败**：`loadConfig` / `initInfra` / `registerToConsul` 等不再直接 `log.Fatalf`（内部 `os.Exit`），而是返回 `error` 由 `run()` 统一处理；`run()` 通过 `defer releaseInfra()` 释放已初始化的全局资源后回到 `main` 上报错误。
- **运行期 panic**：`main` 顶层 `defer recover` 捕获 panic，`log.Errorf` 打印堆栈后调用 `releaseInfra()` 兜底释放再 `os.Exit(1)`。
- HTTP/gRPC server 监听失败也不再 `os.Exit`，而是通过 `Server.quitCh` 通知 `Run` 走正常 `shutdown()` 路径。

> `releaseInfra()` 幂等可重复调用，由正常 `shutdown`、panic 兜底、初始化失败 `defer` 三处共用，保证任何退出路径都不会泄漏 Redis/DB 连接、日志句柄或后台指标 goroutine。
