package database

import (
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/gormlog"
	"gorm.io/gorm"
)

type config struct {
	adapter         dialect.Adapter
	fromEnv         bool
	logger          *logging.Logger
	logConfig       gormlog.Config
	gormConfig      *gorm.Config
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
}

// Option 定义 Database 插件的配置函数。
type Option func(*config)

// WithAdapter 显式注入具体数据库驱动适配器。
func WithAdapter(adapter dialect.Adapter) Option {
	return func(c *config) {
		c.adapter = adapter
	}
}

// FromEnv 声明在启动期从环境变量解析数据库方言与配置。
func FromEnv() Option {
	return func(c *config) {
		c.fromEnv = true
	}
}

// WithLogger 显式配置 logging.Logger 实例。
func WithLogger(logger *logging.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithLogConfig 配置 GORM 日志参数。
func WithLogConfig(cfg gormlog.Config) Option {
	return func(c *config) {
		c.logConfig = cfg
	}
}

// WithGormConfig 允许注入自定义基础 gorm.Config。
func WithGormConfig(cfg *gorm.Config) Option {
	return func(c *config) {
		c.gormConfig = cfg
	}
}

// WithMaxOpenConns 配置底层连接池最大打开连接数。
func WithMaxOpenConns(n int) Option {
	return func(c *config) {
		c.maxOpenConns = n
	}
}

// WithMaxIdleConns 配置底层连接池最大空闲连接数。
func WithMaxIdleConns(n int) Option {
	return func(c *config) {
		c.maxIdleConns = n
	}
}

// WithConnMaxLifetime 配置底层连接最大存活时间。
func WithConnMaxLifetime(d time.Duration) Option {
	return func(c *config) {
		c.connMaxLifetime = d
	}
}

// WithConnMaxIdleTime 配置底层连接最大空闲时间。
func WithConnMaxIdleTime(d time.Duration) Option {
	return func(c *config) {
		c.connMaxIdleTime = d
	}
}
