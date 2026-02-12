package xLog

import (
	"context"
	"fmt"
	"log/slog"
)

// ==================== 日志命令函数 ====================
// 提供简洁的日志记录 API，自动从 context 提取 trace ID

// LogNamedLogger 带名称的日志器
// 名称会显示在日志输出中，如 [CORE]、[INIT] 等
// optionName 会按顺序追加到 logger 名称后显示，如 [CORE] [DB]
type LogNamedLogger struct {
	logger *slog.Logger
}

// WithName 创建带名称的日志器
// 推荐使用 context.Log* 常量作为名称
// optionName 会通过 WithAttrs 写入 option_name_<number>
func WithName(name string, optionName ...string) *LogNamedLogger {
	handler := slog.Default().Handler().WithGroup(name)
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

// Info 记录 INFO 级别日志
func (l *LogNamedLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// Debug 记录 DEBUG 级别日志
func (l *LogNamedLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// Warn 记录 WARN 级别日志
func (l *LogNamedLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// Error 记录 ERROR 级别日志
func (l *LogNamedLogger) Error(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}

// Notice 记录 NOTICE 级别日志（介于 INFO 和 WARN 之间）
func (l *LogNamedLogger) Notice(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelInfo+1, msg, attrs...)
}

// Panic 记录 PANIC 级别日志并触发 panic
func (l *LogNamedLogger) Panic(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
	panic(msg)
}

// ==================== Sugar 语法糖方法 ====================
// 支持 key-value 形式的便捷写法，无需构造 slog.Attr

// SugarInfo 记录 INFO 级别日志（语法糖）
func (l *LogNamedLogger) SugarInfo(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelInfo, msg, args...)
}

// SugarDebug 记录 DEBUG 级别日志（语法糖）
func (l *LogNamedLogger) SugarDebug(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelDebug, msg, args...)
}

// SugarWarn 记录 WARN 级别日志（语法糖）
func (l *LogNamedLogger) SugarWarn(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelWarn, msg, args...)
}

// SugarError 记录 ERROR 级别日志（语法糖）
func (l *LogNamedLogger) SugarError(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelError, msg, args...)
}

// SugarNotice 记录 NOTICE 级别日志（语法糖）
func (l *LogNamedLogger) SugarNotice(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelInfo+1, msg, args...)
}

// SugarPanic 记录 PANIC 级别日志并触发 panic（语法糖）
func (l *LogNamedLogger) SugarPanic(ctx context.Context, msg string, args ...any) {
	l.logger.Log(ctx, slog.LevelError, msg, args...)
	panic(msg)
}

// ==================== 全局便捷函数 ====================

// Info 记录 INFO 级别日志
func Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// Debug 记录 DEBUG 级别日志
func Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// Warn 记录 WARN 级别日志
func Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// Error 记录 ERROR 级别日志
func Error(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}

// Notice 记录 NOTICE 级别日志（介于 INFO 和 WARN 之间）
func Notice(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelInfo+1, msg, attrs...)
}

// Panic 记录 PANIC 级别日志并触发 panic
func Panic(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
	panic(msg)
}

// ==================== 全局 Sugar 语法糖函数 ====================
// 支持 key-value 形式的便捷写法，无需构造 slog.Attr

// SugarInfo 记录 INFO 级别日志（语法糖）
func SugarInfo(ctx context.Context, msg string, args ...any) {
	slog.Log(ctx, slog.LevelInfo, msg, args...)
}

// SugarDebug 记录 DEBUG 级别日志（语法糖）
func SugarDebug(ctx context.Context, msg string, args ...any) {
	slog.Log(ctx, slog.LevelDebug, msg, args...)
}

// SugarWarn 记录 WARN 级别日志（语法糖）
func SugarWarn(ctx context.Context, msg string, args ...any) {
	slog.Log(ctx, slog.LevelWarn, msg, args...)
}

// SugarError 记录 ERROR 级别日志（语法糖）
func SugarError(ctx context.Context, msg string, args ...any) {
	slog.Log(ctx, slog.LevelError, msg, args...)
}

// SugarNotice 记录 NOTICE 级别日志（语法糖）
func SugarNotice(ctx context.Context, msg string, args ...any) {
	slog.Log(ctx, slog.LevelInfo+1, msg, args...)
}

// SugarPanic 记录 PANIC 级别日志并触发 panic（语法糖）
func SugarPanic(ctx context.Context, msg string, args ...any) {
	slog.Log(ctx, slog.LevelError, msg, args...)
	panic(msg)
}
