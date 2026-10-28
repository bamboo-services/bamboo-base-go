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

// LogLevel 定义 GORM 日志输出级别。
type LogLevel int

const (
	// LevelSilent 静默模式，不输出任何 SQL 或错误日志。
	LevelSilent LogLevel = iota + 1
	// LevelError 仅在 SQL 执行发生严重错误时输出日志。
	LevelError
	// LevelWarn 输出错误日志以及超出慢查询阈值的告警日志。
	LevelWarn
	// LevelInfo 输出所有级别日志，包含每次成功执行的 SQL 详情。
	LevelInfo
)

// Config 定义 GORM 日志适配器的运行参数。
type Config struct {
	// SlowThreshold 是慢查询告警阈值。执行耗时超过该值时以 WARN 级别记录；<= 0 时默认采用 200ms。
	SlowThreshold time.Duration
	// LogLevel 是日志输出级别；零值时默认采用 [LevelWarn]。
	LogLevel LogLevel
	// IgnoreRecordNotFoundError 控制是否忽略 [gorm.ErrRecordNotFound] 错误。
	IgnoreRecordNotFoundError bool
}

// Logger 是 GORM 日志适配器，实现了 [gormlogger.Interface] 接口。
type Logger struct {
	logger                    *logging.Logger
	config                    Config
	slowThreshold             time.Duration
	logLevel                  LogLevel
	ignoreRecordNotFoundError bool
}

// New 显式基于 [*logging.Logger] 构造 GORM 日志适配器。
//
// 若传入的 logger 为 nil，则直接返回 [gormlogger.Discard]，
// 不产生任何隐式日志器且不依赖全局 [slog.Default]。
// 配置中的零值将自动应用默认安全策略：[Config.SlowThreshold] <= 0 默认采用 200ms，
// [Config.LogLevel] 默认采用 [LevelWarn]。
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

// LogMode 调整当前日志器的输出级别，并返回新构造的浅拷贝实例。
// 原日志器实例的配置保持不变，支持在链式调用中按需局部调整级别。
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

// Info 记录 INFO 级别的提示信息。
// 仅在当前级别满足 [LevelInfo] 时输出，使用 [logging.NamedREPO] 命名空间。
func (l *Logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= LevelInfo {
		named := l.logger.WithName(logging.NamedREPO)
		named.Info(ctx, fmt.Sprintf(msg, data...))
	}
}

// Warn 记录 WARN 级别的告警信息。
// 仅在当前级别满足 [LevelWarn] 时输出，使用 [logging.NamedREPO] 命名空间。
func (l *Logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= LevelWarn {
		named := l.logger.WithName(logging.NamedREPO)
		named.Warn(ctx, fmt.Sprintf(msg, data...))
	}
}

// Error 记录 ERROR 级别的错误信息。
// 仅在当前级别满足 [LevelError] 时输出，使用 [logging.NamedREPO] 命名空间。
func (l *Logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= LevelError {
		named := l.logger.WithName(logging.NamedREPO)
		named.Error(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace 追踪 SQL 执行耗时、受影响行数与执行结果。
//
// 记录以结构化属性（elapsed_ms, rows, sql, error, threshold）呈现。
// 若当前配置启用了 IgnoreRecordNotFoundError 且 err 为 [gorm.ErrRecordNotFound]，
// 则不记录 ERROR 日志。超出 SlowThreshold 的查询将输出 WARN 级别的慢查询日志。
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
