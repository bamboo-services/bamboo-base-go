package builtin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
)

// LoggerPluginKey 声明日志插件全局唯一标识。
const LoggerPluginKey ctxkey.Key = "plugin_logging"

// LoggerKey 是日志器实例导出至容器的上下文键。
const LoggerKey ctxkey.Key = "context_logger"

// LoggerConfig 定义内置日志插件的受管配置。
type LoggerConfig struct {
	Rotator logging.RotatorConfig
	Console io.Writer
	Level   slog.Level
	Debug   bool
}

type loggerPlugin struct {
	cfg LoggerConfig

	mu      sync.Mutex
	started bool
	writer  *logging.RotatingWriter
	logger  *logging.Logger

	stopOnce sync.Once
	stopErr  error
}

// NewLogger 构造日志插件元信息与工厂函数。
// factory 纯构造，不进行任何文件创建或 IO/协程启动。
func NewLogger(cfg LoggerConfig) (application.PluginInfo, func() application.Plugin) {
	info := application.PluginInfo{
		Key:      LoggerPluginKey,
		Name:     "plugin_logging",
		Mode:     application.ModeTool,
		Provides: []ctxkey.Key{LoggerKey},
	}

	factory := func() application.Plugin {
		return &loggerPlugin{
			cfg: cfg,
		}
	}

	return info, factory
}

// Start 启动受管日志组件，完成 RotatingWriter、Handler 与 Logger 构造。
// 启动局部失败时，真实关闭已创建的 writer 并合并错误返回。
func (p *loggerPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return errors.New("builtin logger: already started")
	}

	writer, err := logging.NewRotatingWriter(p.cfg.Rotator)
	if err != nil {
		return err
	}

	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console:     p.cfg.Console,
		File:        writer,
		Level:       p.cfg.Level,
		IsDebugMode: p.cfg.Debug,
	})

	instLogger, err := logging.New(handler)
	if err != nil {
		closeErr := writer.Close()
		return errors.Join(err, closeErr)
	}

	p.writer = writer
	p.logger = instLogger
	p.started = true
	return nil
}

// Stop 停止受管日志组件并释放资源。
// 无论传入 ctx 是否已过期，均真实等待底层的 writer.Close 退出，不提前返回。
func (p *loggerPlugin) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		writer := p.writer
		p.mu.Unlock()

		if writer != nil {
			p.stopErr = writer.Close()
		}
	})

	return p.stopErr
}

// Expose 准确交付容器导出实例。
func (p *loggerPlugin) Expose() map[ctxkey.Key]any {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make(map[ctxkey.Key]any)
	if p.logger != nil {
		out[LoggerKey] = p.logger
	}
	return out
}
