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

// LoggerPluginKey 是内置日志受管插件在拓扑图中的全局唯一标识。
//
// 作为 [application.PluginInfo.Key] 传递给 [application.App.RegisterTool]，
// 用于在依赖拓扑排序与生命周期管理中唯一标识内置日志组件。
const LoggerPluginKey ctxkey.Key = "plugin_logging"

// LoggerKey 是日志器实例导出至容器上下文的键。
//
// 内置日志组件启动成功后，向共享容器导出 [*logging.Logger] 实例，
// 下游 Worker 或 Tool 可通过 [application.Get] 传入此键安全提取。
// 注意：[foundation/ctxkey] 基础包中不包含 LoggerKey，本键为本包独有定义。
const LoggerKey ctxkey.Key = "context_logger"

// LoggerConfig 定义内置日志插件的受管配置选项。
//
// 零值时使用各字段的零值配置；若 Rotator 的配置为空，底层日志写入器
// 将使用其内部默认值（目录默认 ".logs"，文件名前缀默认 "log"）。
type LoggerConfig struct {
	// Rotator 是文件日志轮转配置，包含目标目录、文件名格式与单文件大小限制。
	Rotator logging.RotatorConfig

	// Console 是控制台日志输出目标。
	// 为 nil 时由底层处理器默认输出至标准输出，也可传入 [io.Discard] 禁用控制台输出。
	Console io.Writer

	// Level 是日志输出的最低级别（如 [slog.LevelInfo] 或 [slog.LevelDebug]）。
	Level slog.Level

	// Debug 标识是否开启调试模式。
	Debug bool
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

// NewLogger 构造内置日志受管插件的元信息与工厂闭包。
//
// 返回的 [application.PluginInfo] 声明组件为 [application.ModeTool] 工具模式，
// 插件标识为 [LoggerPluginKey]，并对外提供 [LoggerKey] 上下文服务。
//
// # 行为契约
//
//   - 纯构造语义：NewLogger 及其返回的工厂函数均为纯构造，不进行任何目录创建、
//     文件句柄打开、协程启动或磁盘 I/O。
//   - 延迟初始化：所有实质性资源分配（创建日志目录、初始化 [logging.RotatingWriter]、
//     组装 [*logging.Logger]）均推迟至组件生命周期的 Start 阶段执行。
//   - 资源释放与幂等：在组件生命周期的 Stop 阶段，将真实等待底层日志写入器
//     刷新并关闭归档文件。Stop 具备并发安全性与幂等性（由 [sync.Once] 保护），
//     且无论传入的 context 是否超时或已取消，均会保证底层文件安全关闭。
//   - 实例隔离：构造的日志器实例完全归属于当前 [application.App]，
//     绝不修改全局 [slog.Default] 指针，支持多应用实例并行无干扰运行。
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
