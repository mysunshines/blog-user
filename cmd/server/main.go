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
	"github.com/mysunshines/blog-user/internal/handler"
	"github.com/mysunshines/blog-user/internal/model"
	"github.com/mysunshines/blog-user/internal/repository"
	"github.com/mysunshines/blog-user/internal/service"
	user "github.com/mysunshines/blog-user/proto/pb"

	"github.com/mysunshines/gocommon/cache"
	"github.com/mysunshines/gocommon/constants"
	common_database "github.com/mysunshines/gocommon/database"
	"github.com/mysunshines/gocommon/log"
	"github.com/mysunshines/gocommon/metrics"
	commonmiddleware "github.com/mysunshines/gocommon/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"
)

type Server struct {
	cfg           *config.Config
	httpServer    *http.Server
	grpcServer    *grpc.Server
	userSvc       service.UserService
	userRepo      repository.UserRepository
	tokenRepo     repository.TokenRepository
	blacklistRepo repository.BlacklistRepository
	userHandl     *handler.UserHandler
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
	const migrationLockKey = "migration:lock:user_service"
	const migrationLockTTL = 60 * time.Second
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	acquired, err := cache.TryLock(context.Background(), migrationLockKey, instanceID, migrationLockTTL)
	if err != nil {
		log.Warnf("Failed to acquire migration lock (Redis unavailable): %v, proceeding without lock", err)
	} else if acquired {
		log.Infof("Migration lock acquired by instance %s", instanceID)
		defer func() {
			if unlockErr := cache.Unlock(context.Background(), migrationLockKey, instanceID); unlockErr != nil {
				log.Warnf("Failed to release migration lock: %v", unlockErr)
			}
		}()
	} else {
		log.Info("Migration lock held by another instance, skipping AutoMigrate")
		// 等待持有锁的实例完成迁移（最多等锁过期）
		time.Sleep(2 * time.Second)
	}

	// 只有获取到锁（或 Redis 不可用）时才执行迁移
	if acquired || err != nil {
		if migrateErr := db.AutoMigrate(&model.User{}, &model.Token{}, &model.UserBlacklist{}); migrateErr != nil {
			log.Fatalf("Failed to migrate database: %v", migrateErr)
		}
	}

	// 初始化仓储层
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	blacklistRepo := repository.NewBlacklistRepository(db)

	// 初始化服务层
	userSvc := service.NewUserService(userRepo, cfg)

	// 初始化处理器
	userHandl := handler.NewUserHandler(userSvc)

	return &Server{
		cfg:           cfg,
		userSvc:       userSvc,
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
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

func (s *Server) runHTTPServer() {
	if s.cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(commonmiddleware.RecoveryMiddleware())
	router.Use(commonmiddleware.LoggingMiddleware())
	router.Use(commonmiddleware.CORSMiddleware())
	router.Use(commonmiddleware.MetricsMiddleware(constants.ServiceNameUser))
	router.Use(commonmiddleware.TraceMiddleware())

	// 限流中间件
	router.Use(commonmiddleware.RateLimitMiddleware())

	// 健康检查（带深度检查）
	router.GET("/health", func(c *gin.Context) {
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
	router.GET("/ready", func(c *gin.Context) {
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

	// API 路由
	api := router.Group(constants.APIPathPrefix)
	{
		userGroup := api.Group("/user")
		{
			// 公开接口
			userGroup.POST("/register", s.userHandl.Register)
			userGroup.POST("/login", s.userHandl.Login)
			userGroup.POST("/validate", s.userHandl.ValidateToken)

			// 需要认证的接口
			authGroup := userGroup.Group("")
			authGroup.Use(commonmiddleware.JWTValidMiddleware())
			{
				authGroup.POST("/logout", s.userHandl.Logout)
				authGroup.GET("", s.userHandl.GetUser)
				authGroup.PUT("", s.userHandl.UpdateUser)
				authGroup.DELETE("/:id", s.userHandl.DeleteUser)
				authGroup.POST("/password", s.userHandl.ChangePassword)
				authGroup.POST("/blacklist", s.userHandl.AddToBlacklist)
				authGroup.DELETE("/blacklist", s.userHandl.RemoveFromBlacklist)
			}

			// 公开查询接口
			userGroup.GET("/blacklist/check", s.userHandl.IsInBlacklist)
		}

		// 用户列表（需要认证）
		api.GET("/users", commonmiddleware.JWTValidMiddleware(), s.userHandl.GetUsers)
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
		grpc.UnaryInterceptor(s.grpcUnaryInterceptor),
	}

	s.grpcServer = grpc.NewServer(grpcOpts...)
	user.RegisterUserServiceServer(s.grpcServer, &handler.GrpcUserHandler{
		Svc: s.userSvc,
		Cb:  s.cb,
	})
	reflection.Register(s.grpcServer)

	log.Infof("gRPC server starting on %s", s.cfg.GRPC.Addr())
	if err := s.grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

// grpcUnaryInterceptor gRPC 统一拦截器：超时+熔断
func (s *Server) grpcUnaryInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
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
	// 加载配置（从项目根目录加载）
	configPath := os.Getenv(constants.EnvConfigPath)
	if configPath == "" {
		// 默认从当前目录加载
		configPath = constants.DefaultConfigPath
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	log.Init(cfg.App.LogDir, cfg.App.LogLevel, constants.ServiceNameUser)

	// 初始化指标
	metrics.Init()

	// 创建并运行服务器
	server := NewServer(cfg)
	defer common_database.Close()
	defer cache.Close()
	defer log.StopRotation()
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
