package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Logger 是基础日志器包装，持有底层的 *slog.Logger 实例。
// 每个应用/App 拥有独立的 Logger 实例，不替换 slog.Default，无全局单例。
type Logger struct {
	logger *slog.Logger
}

// New 创建一个新的 Logger 实例。
// 若传入的 handler 为 nil，则拒绝并返回错误。
func New(handler slog.Handler) (*Logger, error) {
	if handler == nil {
		return nil, errors.New("logging.New: handler cannot be nil")
	}
	return &Logger{
		logger: slog.New(handler),
	}, nil
}

// LogNamedLogger 带名称的日志器。
// 名称会显示在日志输出中，如 [CORE]、[INIT] 等。
// optionName 会按顺序追加到 logger 名称后显示，如 [CORE] [DB]。
type LogNamedLogger struct {
	logger *slog.Logger
}

// WithName 从 Logger 创建带名称与可选附加标签的日志器。
// 推荐使用 Named* 常量作为名称。
// optionName 会通过 WithAttrs 写入 option_name_<number>。
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

// Info 记录 INFO 级别日志。
func (l *LogNamedLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// Debug 记录 DEBUG 级别日志。
func (l *LogNamedLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// Warn 记录 WARN 级别日志。
func (l *LogNamedLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// Error 记录 ERROR 级别日志。
func (l *LogNamedLogger) Error(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}

// Notice 记录 NOTICE 级别日志（介于 INFO 和 WARN 之间）。
func (l *LogNamedLogger) Notice(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelInfo+1, msg, attrs...)
}

// Panic 记录 ERROR 级别日志并触发 panic。
func (l *LogNamedLogger) Panic(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
	panic(msg)
}

// SugarInfo 记录 INFO 级别日志（key-value 语法糖）。
func (l *LogNamedLogger) SugarInfo(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelInfo, msg, args...)
}

// SugarDebug 记录 DEBUG 级别日志（key-value 语法糖）。
func (l *LogNamedLogger) SugarDebug(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelDebug, msg, args...)
}

// SugarWarn 记录 WARN 级别日志（key-value 语法糖）。
func (l *LogNamedLogger) SugarWarn(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelWarn, msg, args...)
}

// SugarError 记录 ERROR 级别日志（key-value 语法糖）。
func (l *LogNamedLogger) SugarError(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelError, msg, args...)
}

// SugarNotice 记录 NOTICE 级别日志（key-value 语法糖）。
func (l *LogNamedLogger) SugarNotice(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelInfo+1, msg, args...)
}

// SugarPanic 记录 ERROR 级别日志并触发 panic（key-value 语法糖）。
func (l *LogNamedLogger) SugarPanic(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelError, msg, args...)
	panic(msg)
}
