package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mysunshines/blog-user/internal/config"
	v1 "github.com/mysunshines/blog-user/internal/handler/v1"
	"github.com/mysunshines/blog-user/internal/model"
	"github.com/mysunshines/blog-user/internal/repository"
	"github.com/mysunshines/blog-user/internal/service"
	user "github.com/mysunshines/blog-user/proto/pb"

	"github.com/mysunshines/gocommon/cache"
	"github.com/mysunshines/gocommon/configcenter"
	"github.com/mysunshines/gocommon/constants"
	common_database "github.com/mysunshines/gocommon/database"
	"github.com/mysunshines/gocommon/log"
	"github.com/mysunshines/gocommon/metrics"
	commonmiddleware "github.com/mysunshines/gocommon/middleware"
	"github.com/mysunshines/gocommon/consul"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"
)

// Version 由构建脚本通过 -ldflags "-X main.Version=xxx" 注入，未注入时默认 "dev"。
var Version = "dev"

type Server struct {
	cfg           *config.Config
	httpServer    *http.Server
	grpcServer    *grpc.Server
	userSvc       service.UserService
	userRepo      repository.UserRepository
	blacklistRepo repository.BlacklistRepository
	userHandl     *v1.UserHandler

	db            *gorm.DB
	cb            *gobreaker.CircuitBreaker
}

func NewServer(cfg *config.Config) *Server {
	// 初始化数据库（类型别名，直接传递）
	if err := common_database.Init(&cfg.Database, cfg.App.Env); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	db := common_database.GetDB()

	// 初始化 Redis 缓存
	cacheCfg := cfg.Redis
	cacheCfg.KeyPrefix = constants.RedisKeyPrefixUser
	if err := cache.Init(&cacheCfg); err != nil {
		log.Warnf("Warning: Failed to init Redis: %v", err)
	}

	// 初始化限流器（类型别名，直接传递）
	commonmiddleware.InitRateLimiter(&cfg.RateLimit)

	// 初始化 JWT
	commonmiddleware.InitJWT(cfg.JWT.Secret)

	// 初始化熔断器
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        constants.ServiceNameUser,
		MaxRequests: constants.DefaultCBMaxRequests,
		Interval:    constants.DefaultCBInterval * time.Second,
		Timeout:     constants.DefaultCBTimeout * time.Second,
	})

	// 自动迁移（分布式锁保护，多实例只有一个执行）
	runDBMigration(db, "migration:lock:user_service", &model.User{}, &model.Token{}, &model.UserBlacklist{}, &model.OperationLog{})

	// 初始化仓储层
	userRepo := repository.NewUserRepository(db)
	blacklistRepo := repository.NewBlacklistRepository(db)

	// 初始化服务层
	userSvc := service.NewUserService(userRepo, cfg)

	// 初始化处理器
	userHandl := v1.NewUserHandler(userSvc, db)

	return &Server{
		cfg:           cfg,
		userSvc:       userSvc,
		userRepo:      userRepo,
		blacklistRepo: blacklistRepo,
		userHandl:     userHandl,
		db:            db,
		cb:            cb,
	}
}

func (s *Server) Run() error {
	// 启动 HTTP 服务器
	go s.runHTTPServer()

	// 启动 gRPC 服务器
	go s.runGRPCServer()

	// 启动 Prometheus 指标服务器
	if s.cfg.Metrics.Enabled {
		go s.runMetricsServer()
	}

	// 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

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

// loadConfig 解析配置路径并加载配置，加载失败时直接终止进程。
func loadConfig() *config.Config {
	// 按 APP_ENV 自动选择配置：test → config_test.yaml  production → config_production.yaml
	// 显式 CONFIG_PATH 优先，未设 APP_ENV 则默认 config.yaml
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		env := os.Getenv("APP_ENV")
		if env != "" && env != "development" {
			configPath = fmt.Sprintf("config/config_%s.yaml", env)
		} else {
			configPath = "config/config.yaml"
		}
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	return cfg
}

// registerToConsul 向 Consul 注册本服务实例，返回取消注册函数。
// 注册失败不致命（降级运行），返回 nil。
func registerToConsul(cfg *config.Config) func() error {
	deregister, err := consul.Register(consul.Registration{
		Name:               cfg.App.Name,
		ConsulAddress:      cfg.Consul.Address,
		GRPCPort:           cfg.GRPC.Port,
		HTTPPort:           cfg.HTTP.Port,
		CheckInterval:      cfg.Consul.CheckInterval,
		DeregisterCritical: cfg.Consul.DeregisterCritical,
	})
	if err != nil {
		log.Warnf("failed to register to consul: %v", err)
		return nil
	}
	return deregister
}

// runDBMigration 在分布式锁保护下执行 GORM AutoMigrate。
// 多实例部署时仅一个实例执行建表/补列，避免并发 ALTER 产生元数据争用。
// Redis 不可用时降级为直接迁移（GORM AutoMigrate 本身幂等）。
func runDBMigration(db interface{ AutoMigrate(dst ...any) error }, lockKey string, models ...any) {
	const migrationLockTTL = 60 * time.Second
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	acquired, err := cache.TryLock(context.Background(), lockKey, instanceID, migrationLockTTL)
	if err != nil {
		log.Warnf("Failed to acquire migration lock (Redis unavailable): %v, proceeding without lock", err)
	} else if acquired {
		log.Infof("Migration lock acquired by instance %s", instanceID)
		defer func() {
			if unlockErr := cache.Unlock(context.Background(), lockKey, instanceID); unlockErr != nil {
				log.Warnf("Failed to release migration lock: %v", unlockErr)
			}
		}()
	} else {
		log.Info("Migration lock held by another instance, skipping AutoMigrate")
		time.Sleep(2 * time.Second)
	}

	if acquired || err != nil {
		if migrateErr := db.AutoMigrate(models...); migrateErr != nil {
			log.Fatalf("Failed to migrate database: %v", migrateErr)
		}
	}
}

func (s *Server) runHTTPServer() {
	if s.cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(commonmiddleware.RecoveryMiddleware())
	// TraceMiddleware 必须在 LoggingMiddleware 之前，确保访问日志能取到 X-Trace-ID
	router.Use(commonmiddleware.TraceMiddleware())
	router.Use(commonmiddleware.LoggingMiddleware())
	router.Use(commonmiddleware.CORSMiddleware())
	// 限制请求体大小，防大请求体 DoS
	router.Use(commonmiddleware.ValidateRequestMiddleware())
	router.Use(commonmiddleware.CSRFMiddleware())
	router.Use(commonmiddleware.MetricsMiddleware(constants.ServiceNameUser))

	// 限流中间件
	router.Use(commonmiddleware.RateLimitMiddleware())

	// 健康检查（带深度检查）
	router.GET(constants.HealthCheckPath, func(c *gin.Context) {
		// 检查数据库连接
		if sqlDB, _ := s.db.DB(); sqlDB != nil {
			if err := sqlDB.Ping(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "reason": "db"})
				return
			}
		}

		// 检查 Redis 连接
		if err := cache.Ping(context.Background()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "reason": "redis"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 就绪探针
	router.GET(constants.ReadinessPath, func(c *gin.Context) {
		sqlDB, _ := s.db.DB()
		if sqlDB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db not ready"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db ping failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// 版本信息
	router.GET(constants.VersionPath, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": Version})
	})

	// API 路由
	api := router.Group(constants.APIPathPrefix)
	{
		userGroup := api.Group("/user")
		{
			// 仅保留无 gRPC 等价方法的 HTTP 端点：
			//   register  — 需要 verify_code 字段（proto RegisterRequest 不含此字段）
			//   send-code — 发送验证码（proto 无该方法）
			// 其余接口（login/logout/get_user 等）已迁移至 gRPC，经 Gateway 代理访问。
			userGroup.POST("/register", s.userHandl.Register)
			userGroup.POST("/send-code", s.userHandl.SendVerificationCode)
		}

		// 管理员接口（需要认证 + 管理员角色），由 Gateway DynamicAdminProxy 透传。
		//
		// URL 约定：
		//   前端:  /admin-api/user/list
		//   Nginx: /admin-api/user/list → Gateway
		//   Gateway: deriveServiceName("user") → user-service
		//   Gateway: 重写路径 → /api/v1/admin/user/list → HTTP 反向代理到此处
		adminGroup := api.Group("/admin/user")
		adminGroup.Use(commonmiddleware.JWTValidMiddleware(), commonmiddleware.ContextMiddleware(), commonmiddleware.AdminOnlyMiddleware())
		{
			adminGroup.GET("", s.userHandl.AdminGetUsers)
			adminGroup.PUT("/:id", s.userHandl.AdminUpdateUser)
			adminGroup.DELETE("/:id", s.userHandl.DeleteUser)
			adminGroup.GET("/operation-logs", s.userHandl.ListOperationLogs)
		}
	}

	addr := s.cfg.HTTP.Addr()
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       constants.DefaultReadTimeout * time.Second,
		ReadHeaderTimeout: constants.DefaultReadHeaderTimeout * time.Second,
		WriteTimeout:      constants.DefaultWriteTimeout * time.Second,
		IdleTimeout:       constants.DefaultIdleTimeout * time.Second,
		MaxHeaderBytes:    constants.MaxHeaderBytes,
	}

	log.Infof("HTTP server starting on %s", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func (s *Server) runGRPCServer() {
	lis, err := net.Listen("tcp", s.cfg.GRPC.Addr())
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// gRPC 限流和超时配置
	grpcOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      10 * time.Minute,
			MaxConnectionAgeGrace: 30 * time.Second,
			Time:                  2 * time.Hour,
			Timeout:               20 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Minute,
			PermitWithoutStream: true,
		}),
		grpc.ChainUnaryInterceptor(s.grpcUnaryInterceptor, commonmiddleware.GRPCAuthInterceptor(), commonmiddleware.GRPCMetricsInterceptor(constants.ServiceNameUser), commonmiddleware.GRPCLoggingInterceptor()),
	}

	s.grpcServer = grpc.NewServer(grpcOpts...)
	userHandler := &v1.GrpcUserHandler{
		Svc: s.userSvc,
		Cb:  s.cb,
		DB:  s.db,
	}
	user.RegisterUserServiceServer(s.grpcServer, userHandler)
	user.RegisterAuditServiceServer(s.grpcServer, userHandler)
	reflection.Register(s.grpcServer)

	log.Infof("gRPC server starting on %s", s.cfg.GRPC.Addr())
	if err := s.grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

// grpcUnaryInterceptor gRPC 统一拦截器：超时+熔断
func (s *Server) grpcUnaryInterceptor(ctx context.Context, req any,
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultGRPCUnaryTimeout*time.Second)
	defer cancel()

	if s.cb != nil {
		result, err := s.cb.Execute(func() (interface{}, error) {
			return handler(ctx, req)
		})
		return result, err
	}
	return handler(ctx, req)
}

func (s *Server) runMetricsServer() {
	addr := fmt.Sprintf(":%d", s.cfg.Metrics.Port)
	http.Handle(s.cfg.Metrics.Path, promhttp.Handler())

	log.Infof("Metrics server starting on %s%s", addr, s.cfg.Metrics.Path)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Errorf("Metrics server error: %v", err)
	}
}

func main() {
	cfg := loadConfig()

	log.Init(cfg.App.LogDir, cfg.App.LogLevel, constants.ServiceNameUser)
	metrics.Init()

	// 配置中心：从 Consul KV 拉取热更配置（限流阈值/日志级别等），缺失时降级到默认值。
	hotCfg := configcenter.Init(cfg.Consul.Address, cfg.App.Name, cfg.App.Env)
	if err := hotCfg.Load(); err != nil && err != configcenter.ErrNotFound {
		log.Warnf("load hot config failed: %v", err)
	}
	// 配置中心热更（日志级别/限流阈值/JWT时效）由 configcenter.apply 自动生效，
	// 其中限流器实例刷新已在 apply 内调用 middleware.UpdateRateLimiter 完成，无需额外回调。
	go hotCfg.Watch()
	defer hotCfg.Stop()

	server := NewServer(cfg)

	deregister := registerToConsul(cfg)
	if deregister != nil {
		defer deregister()
	}

	defer common_database.Close()
	defer cache.Close()
	defer log.StopRotation()
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
