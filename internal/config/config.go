package config

import (
	"fmt"
	"os"
	"strconv"

	commonconfig "github.com/mysunshines/gocommon/config"
	"github.com/mysunshines/gocommon/constants"
	"gopkg.in/yaml.v3"
)

// 共享类型别名 — 直接复用 common/config，零映射开销
type (
	AppConfig       = commonconfig.AppConfig
	DatabaseConfig  = commonconfig.DatabaseConfig
	RedisConfig     = commonconfig.RedisConfig
	JWTConfig       = commonconfig.JWTConfig
	GRPCConfig      = commonconfig.GRPCConfig
	HTTPConfig      = commonconfig.HTTPConfig
	ConsulConfig    = commonconfig.ConsulConfig
	MetricsConfig   = commonconfig.MetricsConfig
	RateLimitConfig = commonconfig.RateLimitConfig
)

// MailConfig 邮件配置
type MailConfig struct {
	SMTPHost     string `yaml:"smtp_host"`
	SMTPPort     int    `yaml:"smtp_port"`
	SMTPUsername string `yaml:"smtp_username"`
	SMTPPassword string `yaml:"smtp_password"`
	FromAddress  string `yaml:"from_address"`
	UseTLS       bool   `yaml:"use_tls"`
}

// Config 用户服务配置
type Config struct {
	App       AppConfig       `yaml:"app"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	HTTP      HTTPConfig      `yaml:"http"`
	GRPC      GRPCConfig      `yaml:"grpc"`
	JWT       JWTConfig       `yaml:"jwt"`
	Consul    ConsulConfig    `yaml:"consul"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Mail      MailConfig      `yaml:"mail"`
}

var globalConfig *Config

// Load 加载配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	setDefaults(&c)
	applyEnvOverrides(&c)
	globalConfig = &c
	return &c, nil
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

func setDefaults(c *Config) {
	if c.App.Env == "" {
		c.App.Env = constants.EnvDevelopment
	}
	if c.App.Name == "" {
		c.App.Name = "user-service"
	}
	if c.App.LogDir == "" {
		c.App.LogDir = "/var/log"
	}
	if c.App.LogLevel == "" {
		c.App.LogLevel = "info"
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 100
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 10
	}
	if c.Database.ConnMaxLifetime == 0 {
		c.Database.ConnMaxLifetime = 3600
	}
	if c.Redis.PoolSize == 0 {
		c.Redis.PoolSize = 100
	}
	if c.HTTP.Host == "" {
		c.HTTP.Host = "0.0.0.0"
	}
	if c.HTTP.Port == 0 {
		c.HTTP.Port = 8081
	}
	if c.GRPC.Host == "" {
		c.GRPC.Host = "0.0.0.0"
	}
	if c.GRPC.Port == 0 {
		c.GRPC.Port = 9090
	}
	if c.JWT.ExpireTime == 0 {
		c.JWT.ExpireTime = 86400 * 7 // 7天
	}
	if c.Metrics.Port == 0 {
		c.Metrics.Port = 9091
	}
	if c.Metrics.Path == "" {
		c.Metrics.Path = constants.MetricsPath
	}
	if c.RateLimit.QPS == 0 {
		c.RateLimit.QPS = 100
	}
	if c.RateLimit.Burst == 0 {
		c.RateLimit.Burst = 200
	}
	if c.Mail.SMTPPort == 0 {
		c.Mail.SMTPPort = 587
	}
	if c.Mail.FromAddress == "" {
		c.Mail.FromAddress = "noreply@blog.local"
	}
}

func applyEnvOverrides(c *Config) {
	if v := os.Getenv(constants.EnvDBHost); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv(constants.EnvDBPort); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Database.Port = port
		}
	}
	if v := os.Getenv(constants.EnvDBUser); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv(constants.EnvDBPassword); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv(constants.EnvDBName); v != "" {
		c.Database.Name = v
	}
	if v := os.Getenv(constants.EnvRedisHost); v != "" {
		c.Redis.Host = v
	}
	if v := os.Getenv(constants.EnvRedisPort); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Redis.Port = port
		}
	}
	if v := os.Getenv(constants.EnvRedisPassword); v != "" {
		c.Redis.Password = v
	}
	if v := os.Getenv(constants.EnvConsulAddress); v != "" {
		c.Consul.Address = v
	}
	if v := os.Getenv(constants.EnvGRPCPort); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.GRPC.Port = port
		}
	}
	if v := os.Getenv(constants.EnvSMTPHost); v != "" {
		c.Mail.SMTPHost = v
	}
	if v := os.Getenv(constants.EnvSMTPPort); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Mail.SMTPPort = port
		}
	}
	if v := os.Getenv(constants.EnvSMTPUsername); v != "" {
		c.Mail.SMTPUsername = v
	}
	if v := os.Getenv(constants.EnvSMTPPassword); v != "" {
		c.Mail.SMTPPassword = v
	}
	if v := os.Getenv(constants.EnvSMTPFrom); v != "" {
		c.Mail.FromAddress = v
	}
}
