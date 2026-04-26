package xAsync

import xLog "github.com/bamboo-services/bamboo-base-go/common/log"

// Option 定义异步任务的配置选项。
type Option func(config *Config)

// Config 表示异步任务的运行配置。
type Config struct {
	Name   string
	Debug  bool
	Logger *xLog.LogNamedLogger
}

// WithName 设置异步任务名称，名称会显示在日志中便于追踪任务执行状态。
func WithName(name string) Option {
	return func(config *Config) {
		if name == "" {
			return
		}
		config.Name = name
	}
}

// WithDebug 启用调试日志，启用后会输出任务开始执行和执行完成的日志。
func WithDebug() Option {
	return func(config *Config) {
		config.Debug = true
	}
}

// WithLogger 设置自定义日志器。
func WithLogger(logger *xLog.LogNamedLogger) Option {
	return func(config *Config) {
		if logger == nil {
			return
		}
		config.Logger = logger
	}
}

func defaultConfig() Config {
	return Config{
		Name:   "",
		Debug:  false,
		Logger: nil,
	}
}
