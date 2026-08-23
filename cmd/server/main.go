package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	v1 "github.com/mysunshines/blog-user/internal/handler/v1"
	"github.com/mysunshines/blog-user/internal/repository"
	"github.com/mysunshines/blog-user/internal/service"
	user "github.com/mysunshines/blog-user/proto/pb"

	"github.com/mysunshines/gocommon/cache"
	goconfig "github.com/mysunshines/gocommon/config"
	"github.com/mysunshines/gocommon/configcenter"
	"github.com/mysunshines/gocommon/constants"
	"github.com/mysunshines/gocommon/consul"
	"github.com/mysunshines/gocommon/database"
	"github.com/mysunshines/gocommon/log"
	"github.com/mysunshines/gocommon/metrics"
	"github.com/mysunshines/gocommon/middleware"
	"github.com/mysunshines/gocommon/observability"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"
)

// Version 由构建脚本通过 -ldflags "-X main.Version=xxx" 注入，未注入时默认 "dev"。
var Version = "dev"

// 进程级资源句柄，供 shutdown/releaseInfra 在退出时统一释放（避免多处 defer 重复释放）。
var (
	metricsCancel context.CancelFunc
	hotCfg        *configcenter.ServiceConfig
	deregister    func() error
)

type Server struct {
	cfg           *goconfig.Config
	httpServer    *http.Server
	grpcServer    *grpc.Server
	userSvc       service.UserService
	userRepo      repository.UserRepository
	blacklistRepo repository.BlacklistRepository

	db *gorm.DB
	cb *gobreaker.CircuitBreaker

	// quitCh 供内部 server goroutine 在监听失败时通知 Run 走正常关闭路径，
	// 避免 log.Fatalf 直接 os.Exit 跳过资源释放。
	quitCh chan struct{}
}

// initInfra 负责所有外部基础设施的初始化（数据库、Redis、表结构迁移）。
// 与 NewServer（纯依赖装配）分离，使 main 的启动顺序清晰可控。
// 初始化失败返回 error（由调用方统一处理，避免直接 os.Exit 导致资源泄漏）。
func initInfra(cfg *goconfig.Config) (*gorm.DB, error) {
	// 初始化数据库
	if err := database.Init(&cfg.Database, cfg.App.Env); err != nil {
		return nil, fmt.Errorf("failed to init database: %v", err)
	}
	db := database.GetDB()

	// 初始化 Redis 缓存
	cacheCfg := cfg.Redis
	cacheCfg.KeyPrefix = constants.RedisKeyPrefixUser
	if err := cache.Init(&cacheCfg); err != nil {
		return nil, fmt.Errorf("failed to init Redis: %v", err)
	}

	return db, nil
}

// NewServer 仅做依赖装配（限流器/JWT/熔断器/仓储/服务/处理器），不做任何 I/O。
func NewServer(cfg *goconfig.Config, db *gorm.DB) *Server {
	// 初始化限流器（类型别名，直接传递）
	middleware.InitRateLimiter(&cfg.RateLimit)

	// 初始化 JWT
	middleware.InitJWT(cfg.JWT.Secret)

	// 初始化熔断器
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        constants.ServiceNameUser,
		MaxRequests: constants.DefaultCBMaxRequests,
		Interval:    constants.DefaultCBInterval * time.Second,
		Timeout:     constants.DefaultCBTimeout * time.Second,
	})

	// 初始化仓储层
	userRepo := repository.NewUserRepository(db)
	blacklistRepo := repository.NewBlacklistRepository(db)

	// 初始化服务层
	userSvc := service.NewUserService(userRepo, cfg)

	return &Server{
		cfg:           cfg,
		userSvc:       userSvc,
		userRepo:      userRepo,
		blacklistRepo: blacklistRepo,
		db:            db,
		cb:            cb,
		quitCh:        make(chan struct{}),
	}
}

func (s *Server) Run() error {
	// 启动 HTTP 服务器
	go s.runHTTPServer()

	// 启动 gRPC 服务器
	go s.runGRPCServer()

	// 启动 Prometheus 指标服务器
	if goconfig.Get().Metrics.Enabled {
		go s.runMetricsServer()
	}

	// 等待信号或内部 server 监听失败
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
	case <-s.quitCh:
		log.Errorf("server goroutine failed, initiating shutdown")
	}

	log.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Errorf("HTTP server shutdown error: %v", err)
	}

	s.grpcServer.GracefulStop()

	log.Info("Server exited")
	return nil
}

// loadConfig 解析配置路径并加载配置，加载失败时返回 error（由调用方统一处理）。
func loadConfig() (*goconfig.Config, error) {
	cfg, err := goconfig.LoadByEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}
	return cfg, nil
}

// registerToConsul 向 Consul 注册本服务实例，返回取消注册函数。
// 注册失败视为启动失败，返回 error。
func registerToConsul(cfg *goconfig.Config) (func() error, error) {
	deregister, err := consul.Register(consul.Registration{
		Name:               cfg.App.Name,
		ConsulAddress:      cfg.Consul.Address,
		GRPCPort:           cfg.GRPC.Port,
		HTTPPort:           cfg.HTTP.Port,
		CheckInterval:      cfg.Consul.CheckInterval,
		DeregisterCritical: cfg.Consul.DeregisterCritical,
		Version:            consul.VersionFromEnv(Version),
		Canary:             consul.CanaryFromEnv(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register to consul: %v", err)
	}
	return deregister, nil
}

// runHTTPServer 仅承载运维探活端点（/health、/ready、/version），不暴露任何业务路由。
// 业务流量（公开/用户/admin 接口）一律走 gRPC（见 runGRPCServer），由 Gateway 经 gRPC 反射代理
// 转发，本服务不再保留任何 HTTP 业务入口（不再有 gin handler 双实现）。
// 头像上传（multipart）已上提至 Gateway（/api/v1/user/avatar 走 MinIO），不再由本服务接收。
func (s *Server) runHTTPServer() {
	rootMux := http.NewServeMux()
	// 健康检查（深度检查 db/redis）
	rootMux.HandleFunc(constants.HealthCheckPath, func(w http.ResponseWriter, r *http.Request) {
		if sqlDB, _ := s.db.DB(); sqlDB != nil {
			if err := sqlDB.Ping(); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"unhealthy","reason":"db"}`))
				return
			}
		}
		if err := cache.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unhealthy","reason":"redis"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	// 就绪探针
	rootMux.HandleFunc(constants.ReadinessPath, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	// 版本信息
	rootMux.HandleFunc(constants.VersionPath, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"` + Version + `"}`))
	})

	addr := goconfig.Get().HTTP.Addr()
	h := goconfig.Get().Server.HTTP
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           rootMux,
		ReadTimeout:       time.Duration(h.ReadTimeoutSec) * time.Second,
		ReadHeaderTimeout: time.Duration(h.ReadHeaderTimeoutSec) * time.Second,
		WriteTimeout:      time.Duration(h.WriteTimeoutSec) * time.Second,
		IdleTimeout:       time.Duration(h.IdleTimeoutSec) * time.Second,
		MaxHeaderBytes:    constants.MaxHeaderBytes,
	}

	log.Infof("HTTP server (probe-only) starting on %s", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Errorf("Failed to start HTTP server: %v", err)
		close(s.quitCh)
	}
}

func (s *Server) runGRPCServer() {
	lis, err := net.Listen("tcp", goconfig.Get().GRPC.Addr())
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// gRPC 限流和超时配置（keepalive/并发流取自 config.Server.GRPC，仅启动期生效）
	g := goconfig.Get().Server.GRPC
	grpcOpts := []grpc.ServerOption{
		// 连接超时
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     time.Duration(g.MaxConnectionIdle) * time.Second,
			MaxConnectionAge:      time.Duration(g.MaxConnectionAge) * time.Second,
			MaxConnectionAgeGrace: time.Duration(g.MaxConnectionAgeGrace) * time.Second,
		}),
		// 超时配置
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             time.Duration(g.MinPingInterval) * time.Second,
			PermitWithoutStream: true,
		}),
		// 最大并发连接数
		grpc.MaxConcurrentStreams(g.MaxConcurrentStreams),
		// 添加 unary 拦截器（超时+熔断），并叠加 gRPC 鉴权/指标/日志拦截器
		grpc.ChainUnaryInterceptor(
			s.grpcUnaryInterceptor,
			middleware.GRPCAuthInterceptor(),
			middleware.GRPCMetricsInterceptor(constants.ServiceNameUser),
			middleware.GRPCLoggingInterceptor(),
		),
	}
	// 链路追踪：服务端基于入站 W3C traceparent 生成子 span（想法 3 · 方案 B）。
	grpcOpts = append(grpcOpts, observability.GRPCServerOptions()...)

	s.grpcServer = grpc.NewServer(grpcOpts...)
	// 显式注册 gRPC 业务服务：标准 protobuf 生成的 RegisterXxxServiceServer 不会自动生效，
	// 必须在此调用一次，否则客户端（gateway 经 Consul 转发）调用会报 unknown method。
	// GrpcUserHandler 实现了 user.UserServiceServer 接口。
	userHandler := &v1.GrpcUserHandler{
		Svc: s.userSvc,
		Cb:  s.cb,
		DB:  s.db,
	}
	user.RegisterUserServiceServer(s.grpcServer, userHandler)
	reflection.Register(s.grpcServer)

	log.Infof("gRPC server starting on %s", goconfig.Get().GRPC.Addr())
	if err := s.grpcServer.Serve(lis); err != nil {
		log.Errorf("Failed to serve gRPC: %v", err)
		close(s.quitCh)
	}
}

// grpcUnaryInterceptor gRPC 统一拦截器（超时+熔断）。
// 利用 info.FullMethod 实现按方法差异化超时：列表/搜索等重接口给予更长超时，
// 避免被统一的短超时误杀；轻量单条查询仍走默认超时。
func (s *Server) grpcUnaryInterceptor(ctx context.Context, req any,
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, middleware.GRPCMethodTimeout(info.FullMethod))
	defer cancel()

	if s.cb != nil {
		result, err := s.cb.Execute(func() (interface{}, error) {
			return handler(ctx, req)
		})
		return result, err
	}
	return handler(ctx, req)
}

// runMetricsServer 运行指标服务器
func (s *Server) runMetricsServer() {
	addr := fmt.Sprintf(":%d", goconfig.Get().Metrics.Port)
	http.Handle(goconfig.Get().Metrics.Path, promhttp.Handler())

	log.Infof("Metrics server starting on %s%s", addr, goconfig.Get().Metrics.Path)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Errorf("Metrics server error: %v", err)
	}
}

func main() {
	// 顶层兜底：panic 与 run 返回 err 两条路径收敛到同一个出口，
	// 自然走到 defer 统一释放资源（避免中途 log.Fatalf/os.Exit 跳过 defer）。
	var runErr error
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic recovered in main: %v\n%s", r, debug.Stack())
			runErr = fmt.Errorf("panic: %v", r)
		}
		if runErr != nil {
			log.Errorf("%s exited: %v", constants.ServiceNameUser, runErr)
		}
		releaseInfra()
		if runErr != nil {
			os.Exit(1)
		}
	}()

	runErr = run()
}

// run 承载全部初始化与运行逻辑。初始化失败统一返回 error（不再直接 os.Exit），
// 资源释放统一交给 main 的顶层 defer（自然走到 releaseInfra），run 自身不负责释放。
func run() error {
	// ① 加载配置
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// ② 初始化日志
	log.Init(cfg.App.LogDir, cfg.App.LogLevel, constants.ServiceNameUser)

	// ②.1 启用 Loki 集中日志（想法 3 · 方案 A）；未配置时降级为仅本地日志。
	log.EnableLokiFromConfig(cfg.Loki, constants.ServiceNameUser)
	// ②.2 启用 OpenTelemetry 链路追踪（想法 3 · 方案 B）；未配置时降级为不采集。
	observability.InitAndRegister(constants.ServiceNameUser, cfg.OTel)

	// ③ 初始化指标
	metrics.Init(constants.ServiceNameUser)
	// 周期性刷新运行时指标（内存/goroutine）并上报服务健康状态，消除 dashboard 长期 0 / No data。
	metricsCtx, metricsCancelFn := context.WithCancel(context.Background())
	metricsCancel = metricsCancelFn
	metrics.StartRuntimeMetrics(metricsCtx, 15*time.Second)
	metrics.StartHealthReporter(metricsCtx, constants.ServiceNameUser, 10*time.Second, database.Ping, cache.Ping)

	// ④ 配置中心热更：从 Consul KV 拉取热更配置（限流阈值/日志级别等），缺失时降级到默认值。
	hotCfg = configcenter.Init(cfg.Consul.Address, cfg.App.Name, cfg.App.Env)
	if err := hotCfg.Load(); err != nil && err != configcenter.ErrNotFound {
		log.Warnf("load hot config failed: %v", err)
	}
	// 配置中心热更（日志级别/限流阈值/JWT时效）由 configcenter.apply 自动生效，
	// 其中限流器实例刷新已在 apply 内调用 middleware.UpdateRateLimiter 完成，无需额外回调。
	go hotCfg.Watch()

	// ⑤ 初始化基础设施（数据库 / Redis / 表迁移）
	db, err := initInfra(cfg)
	if err != nil {
		return err
	}

	// ⑥ 启用 Consul 服务发现（供本服务调用下游时解析实例）
	consul.UseConsulDiscovery(cfg.Consul.Address)

	// ⑦ 注册本服务到 Consul
	deregister, err = registerToConsul(cfg)
	if err != nil {
		return err
	}

	// ⑧ 装配并启动服务（Run 内部监听信号并优雅关闭 HTTP/gRPC）
	server := NewServer(cfg, db)
	if err := server.Run(); err != nil {
		return fmt.Errorf("server error: %v", err)
	}

	// ⑨ 统一释放资源（顺序：先摘流量→关连接→停热更/指标→停日志）
	shutdown()
	return nil
}

// shutdown 正常退出路径释放：先摘流量，再交由 releaseInfra 释放全局资源。
func shutdown() {
	// 1. 先从 Consul 注销，摘除流量（让网关停止转发新请求）
	if deregister != nil {
		if err := deregister(); err != nil {
			log.Warnf("consul deregister: %v", err)
		}
	}
	// 2. 释放其余全局资源（热更/指标/连接池/日志）
	releaseInfra()
}

// releaseInfra 释放所有"已初始化的全局资源"，幂等可重复调用。
// 正常退出由 shutdown 调用；异常退出（panic 兜底、初始化失败 return err 的 defer）也调用，
// 保证无论哪条路径都不会泄漏 Redis/DB 连接、日志句柄或后台指标 goroutine。
func releaseInfra() {
	// 停止配置中心热更监听（关闭 fsnotify watcher）
	if hotCfg != nil {
		hotCfg.Stop()
	}

	// 取消指标采集上下文，停止后台 goroutine
	if metricsCancel != nil {
		metricsCancel()
	}

	// 释放 Redis 连接池
	if err := cache.Close(); err != nil {
		log.Warnf("cache close: %v", err)
	}

	// 关闭数据库连接池
	if err := database.Close(); err != nil {
		log.Warnf("database close: %v", err)
	}

	// 最后停止日志轮转（flush 并关闭日志文件）
	observability.ShutdownGlobal(context.Background())
	log.StopRotation()
}
