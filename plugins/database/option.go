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

// Option 定义数据库插件的函数式配置选项。
type Option func(*config)

// WithAdapter 显式指定具体数据库驱动适配器实例。
func WithAdapter(adapter dialect.Adapter) Option {
	return func(c *config) {
		c.adapter = adapter
	}
}

// FromEnv 声明在插件启动阶段从环境变量动态解析驱动方言与数据库连接参数。
func FromEnv() Option {
	return func(c *config) {
		c.fromEnv = true
	}
}

// WithLogger 配置数据库插件关联的结构化日志记录器 [*logging.Logger]。
func WithLogger(logger *logging.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithLogConfig 配置 GORM 日志行为参数（如慢查询阈值与日志输出级别）。
func WithLogConfig(cfg gormlog.Config) Option {
	return func(c *config) {
		c.logConfig = cfg
	}
}

// WithGormConfig 允许注入自定义的基础 [*gorm.Config] 实例。
func WithGormConfig(cfg *gorm.Config) Option {
	return func(c *config) {
		c.gormConfig = cfg
	}
}

// WithMaxOpenConns 配置底层连接池最大打开物理连接数。
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

// WithConnMaxLifetime 配置底层物理连接可被复用的最大存活时长。
func WithConnMaxLifetime(d time.Duration) Option {
	return func(c *config) {
		c.connMaxLifetime = d
	}
}

// WithConnMaxIdleTime 配置底层连接池中连接的最大空闲时长。
func WithConnMaxIdleTime(d time.Duration) Option {
	return func(c *config) {
		c.connMaxIdleTime = d
	}
}
