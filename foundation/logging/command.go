package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Logger 是基于显式 slog.Handler 的基础日志器包装，持有底层的
// *slog.Logger 实例。
//
// 每个应用持有独立的 Logger 实例，构造与使用均不替换 slog.Default，
// 无进程级全局单例。零值不可用，须通过 [New] 构造；日志输出经由
// [Logger.WithName] 派生的 [LogNamedLogger] 完成。
type Logger struct {
	logger *slog.Logger
}

// New 创建一个新的 Logger 实例。
//
// handler 为 nil 时拒绝并返回错误，调用方应改用有效的 slog.Handler
// 重试；handler 通常是 [NewLogHandler] 的返回值，也可以是任意
// 第三方 Handler。
func New(handler slog.Handler) (*Logger, error) {
	if handler == nil {
		return nil, errors.New("logging.New: handler cannot be nil")
	}
	return &Logger{
		logger: slog.New(handler),
	}, nil
}

// LogNamedLogger 是带名称的日志器，由 [Logger.WithName] 派生。
//
// 名称显示在日志输出中，如 [CORE]、[INIT]；附加的 optionName 按顺序
// 追加到名称后显示，如 [CORE] [DB]。各输出方法把 ctx 中携带的
// [ctxkey.RequestKey] 作为 TraceID 交给底层 Handler；级别低于
// Handler 阈值的记录不产生输出。方法本身无状态，可并发调用。
type LogNamedLogger struct {
	logger *slog.Logger
}

// WithName 从 Logger 创建带名称与可选附加标签的日志器。
//
// 推荐使用 Named* 常量作为名称。optionName 依次写入
// option_name_<序号> 属性，控制台输出时按顺序渲染为名称后的
// 附加标签。返回的日志器与原 Logger 共享底层 Handler。
func (l *Logger) WithName(name string, optionName ...string) *LogNamedLogger {
	handler := l.logger.Handler().WithGroup(name)
	if len(optionName) > 0 {
		attrs := make([]slog.Attr, 0, len(optionName))
		for index, option := range optionName {
			attrs = append(attrs, slog.String(fmt.Sprintf("option_name_%d", index+1), option))
		}
		handler = handler.WithAttrs(attrs)
	}
	return &LogNamedLogger{
		logger: slog.New(handler),
	}
}

// Info 记录 INFO 级别日志，附加属性以 slog.Attr 形式传入。
func (l *LogNamedLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// Debug 记录 DEBUG 级别日志，附加属性以 slog.Attr 形式传入。
func (l *LogNamedLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// Warn 记录 WARN 级别日志，附加属性以 slog.Attr 形式传入。
func (l *LogNamedLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// Error 记录 ERROR 级别日志，附加属性以 slog.Attr 形式传入；
// 控制台输出会附加当前 goroutine 堆栈。
func (l *LogNamedLogger) Error(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}

// Notice 记录 NOTICE 级别日志（介于 INFO 和 WARN 之间），
// 附加属性以 slog.Attr 形式传入。
func (l *LogNamedLogger) Notice(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelInfo+1, msg, attrs...)
}

// Panic 先以 ERROR 级别记录日志，再以 msg 为值触发 panic；
// 调用方需自行 recover。附加属性以 slog.Attr 形式传入。
func (l *LogNamedLogger) Panic(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
	panic(msg)
}

// SugarInfo 记录 INFO 级别日志，附加属性采用 key-value
// 交替传参的语法糖形式。
func (l *LogNamedLogger) SugarInfo(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelInfo, msg, args...)
}

// SugarDebug 记录 DEBUG 级别日志，附加属性采用 key-value
// 交替传参的语法糖形式。
func (l *LogNamedLogger) SugarDebug(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelDebug, msg, args...)
}

// SugarWarn 记录 WARN 级别日志，附加属性采用 key-value
// 交替传参的语法糖形式。
func (l *LogNamedLogger) SugarWarn(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelWarn, msg, args...)
}

// SugarError 记录 ERROR 级别日志，附加属性采用 key-value 交替传参
// 的语法糖形式；控制台输出会附加当前 goroutine 堆栈。
func (l *LogNamedLogger) SugarError(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelError, msg, args...)
}

// SugarNotice 记录 NOTICE 级别日志（介于 INFO 和 WARN 之间），
// 附加属性采用 key-value 交替传参的语法糖形式。
func (l *LogNamedLogger) SugarNotice(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelInfo+1, msg, args...)
}

// SugarPanic 先以 ERROR 级别记录日志，再以 msg 为值触发 panic；
// 调用方需自行 recover。附加属性采用 key-value 交替传参的
// 语法糖形式。
func (l *LogNamedLogger) SugarPanic(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelError, msg, args...)
	panic(msg)
}
