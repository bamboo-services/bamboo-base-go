package gormlog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// LogLevel GORM 日志级别类型
type LogLevel int

const (
	// LevelSilent 静默模式，不输出任何日志
	LevelSilent LogLevel = iota + 1
	// LevelError 仅输出错误日志
	LevelError
	// LevelWarn 输出错误和警告日志
	LevelWarn
	// LevelInfo 输出所有级别日志（包含 SQL 语句）
	LevelInfo
)

// Config GORM Logger 配置
type Config struct {
	SlowThreshold             time.Duration // 慢查询阈值，超过以 WARN 记录；<=0 默认 200ms
	LogLevel                  LogLevel      // 日志级别；零值默认 LevelWarn
	IgnoreRecordNotFoundError bool          // 是否忽略 ErrRecordNotFound
}

// Logger GORM logging 适配器，实现 gormlogger.Interface 接口。
type Logger struct {
	logger                    *logging.Logger
	config                    Config
	slowThreshold             time.Duration
	logLevel                  LogLevel
	ignoreRecordNotFoundError bool
}

// New 显式基于 logging.Logger 构造 GORM 日志器。
// 若 logger 为 nil，则返回 gormlogger.Discard，不暗造 Logger 且不依赖 slog.Default。
func New(logger *logging.Logger, cfg Config) gormlogger.Interface {
	if logger == nil {
		return gormlogger.Discard
	}

	slowThreshold := cfg.SlowThreshold
	if slowThreshold <= 0 {
		slowThreshold = 200 * time.Millisecond
	}

	level := cfg.LogLevel
	if level == 0 {
		level = LevelWarn
	}

	return &Logger{
		logger:                    logger,
		config:                    cfg,
		slowThreshold:             slowThreshold,
		logLevel:                  level,
		ignoreRecordNotFoundError: cfg.IgnoreRecordNotFoundError,
	}
}

// LogMode 设置日志级别并返回新实例。
func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := *l
	switch level {
	case gormlogger.Silent:
		clone.logLevel = LevelSilent
	case gormlogger.Error:
		clone.logLevel = LevelError
	case gormlogger.Warn:
		clone.logLevel = LevelWarn
	case gormlogger.Info:
		clone.logLevel = LevelInfo
	default:
		clone.logLevel = LevelWarn
	}
	return &clone
}

// Info 记录 INFO 级别日志。
func (l *Logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= LevelInfo {
		named := l.logger.WithName(logging.NamedREPO)
		named.Info(ctx, fmt.Sprintf(msg, data...))
	}
}

// Warn 记录 WARN 级别日志。
func (l *Logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= LevelWarn {
		named := l.logger.WithName(logging.NamedREPO)
		named.Warn(ctx, fmt.Sprintf(msg, data...))
	}
}

// Error 记录 ERROR 级别日志。
func (l *Logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= LevelError {
		named := l.logger.WithName(logging.NamedREPO)
		named.Error(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace 追踪 SQL 执行耗时与结果。
func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= LevelSilent {
		return
	}

	elapsed := time.Since(begin)
	elapsedMs := float64(elapsed.Nanoseconds()) / 1e6
	sql, rows := fc()

	attrs := []slog.Attr{
		slog.Float64("elapsed_ms", elapsedMs),
		slog.Int64("rows", rows),
		slog.String("sql", sql),
	}

	named := l.logger.WithName(logging.NamedREPO)

	switch {
	case err != nil && l.logLevel >= LevelError:
		if l.ignoreRecordNotFoundError && errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		named.Error(ctx, "SQL执行失败", append(attrs, slog.String("error", err.Error()))...)
	case l.slowThreshold > 0 && elapsed > l.slowThreshold && l.logLevel >= LevelWarn:
		named.Warn(ctx, "发现SQL慢查询", append(attrs, slog.Duration("threshold", l.slowThreshold))...)
	case l.logLevel >= LevelInfo:
		named.Info(ctx, "SQL执行成功", attrs...)
	}
}
