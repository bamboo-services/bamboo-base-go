package xLog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// LogLevel GORM 日志级别类型
// 使用独立定义而非直接引用 gorm/logger 的类型，保持 API 稳定性
type LogLevel int

const (
	// LevelSilent 静默模式，不输出任何日志
	LevelSilent LogLevel = iota + 1
	// LevelError 仅输出错误日志
	LevelError
	// LevelWarn 输出错误和警告日志
	LevelWarn
	// LevelInfo 输出所有级别日志（包括 SQL 语句）
	LevelInfo
)

// GormLoggerConfig GORM Logger 配置
type GormLoggerConfig struct {
	SlowThreshold             int      // 慢查询阈值（毫秒），超过此值以 WARN 级别记录
	LogLevel                  LogLevel // 日志级别
	IgnoreRecordNotFoundError bool     // 是否忽略 ErrRecordNotFound 错误
	Colorful                  bool     // 是否启用彩色输出（预留字段，slog 已自行处理）
}

// SlogLogger GORM slog 适配器
// 实现 gorm.io/gorm/logger.Interface 接口
type SlogLogger struct {
	logger                    *slog.Logger
	config                    GormLoggerConfig
	slowThreshold             time.Duration
	logLevel                  LogLevel
	ignoreRecordNotFoundError bool
}

// NewSlogLogger 创建 GORM slog 日志适配器
//
// 参数说明:
//   - slogger: slog.Logger 实例，推荐使用 slog.Default().WithGroup(xLog.NamedREPO)
//   - config: GORM Logger 配置
//
// 返回值:
//   - logger.Interface: GORM 日志接口实现
//
// 使用示例:
//
//	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
//	    Logger: xLog.NewSlogLogger(slog.Default().WithGroup(xLog.NamedREPO), xLog.GormLoggerConfig{
//	        SlowThreshold:             200,       // 200ms 慢查询阈值
//	        LogLevel:                  xLog.Info, // 日志级别
//	        IgnoreRecordNotFoundError: true,      // 忽略记录未找到错误
//	    }),
//	})
func NewSlogLogger(slogger *slog.Logger, config GormLoggerConfig) logger.Interface {
	// 设置默认值
	if config.SlowThreshold <= 0 {
		config.SlowThreshold = 200 // 默认 200ms
	}
	if config.LogLevel == 0 {
		config.LogLevel = LevelInfo // 默认 Info 级别
	}

	return &SlogLogger{
		logger:                    slogger,
		config:                    config,
		slowThreshold:             time.Duration(config.SlowThreshold) * time.Millisecond,
		logLevel:                  config.LogLevel,
		ignoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
	}
}

// LogMode 设置日志级别，返回新的 Logger 实例（链式调用）
//
// 参数说明:
//   - level: GORM 日志级别
//
// 返回值:
//   - logger.Interface: 新的 Logger 实例
func (l *SlogLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = LogLevel(level)
	return &newLogger
}

// Info 记录 INFO 级别日志
//
// 参数说明:
//   - ctx: 上下文（用于提取 trace ID）
//   - msg: 日志消息格式
//   - data: 格式化参数
func (l *SlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= LevelInfo {
		l.logger.InfoContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Warn 记录 WARN 级别日志
//
// 参数说明:
//   - ctx: 上下文（用于提取 trace ID）
//   - msg: 日志消息格式
//   - data: 格式化参数
func (l *SlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= LevelWarn {
		l.logger.WarnContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Error 记录 ERROR 级别日志
//
// 参数说明:
//   - ctx: 上下文（用于提取 trace ID）
//   - msg: 日志消息格式
//   - data: 格式化参数
func (l *SlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= LevelError {
		l.logger.ErrorContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace 追踪 SQL 执行，记录 SQL 语句、执行时间和受影响行数
//
// 参数说明:
//   - ctx: 上下文（用于提取 trace ID）
//   - begin: SQL 开始执行时间
//   - fc: 获取 SQL 和受影响行数的回调函数
//   - err: SQL 执行错误（如有）
//
// 日志输出规则:
//   - 执行出错: ERROR 级别（IgnoreRecordNotFoundError=true 时忽略 ErrRecordNotFound）
//   - 慢查询: WARN 级别（超过 SlowThreshold）
//   - 普通查询: INFO 级别（仅在 LogLevel >= Info 时输出）
func (l *SlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	// 静默模式不输出任何日志
	if l.logLevel <= LevelSilent {
		return
	}

	elapsed := time.Since(begin)
	elapsedMs := float64(elapsed.Nanoseconds()) / 1e6
	sql, rows := fc()

	// 构建基础日志属性
	attrs := []slog.Attr{
		slog.Float64("elapsed_ms", elapsedMs),
		slog.Int64("rows", rows),
		slog.String("sql", sql),
	}

	switch {
	// 执行出错
	case err != nil && l.logLevel >= LevelError:
		// 判断是否忽略 ErrRecordNotFound
		if l.ignoreRecordNotFoundError && errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		l.logger.LogAttrs(ctx, slog.LevelError, "SQL执行失败", append(attrs, slog.String("error", err.Error()))...)

	// 慢查询
	case l.slowThreshold > 0 && elapsed > l.slowThreshold && l.logLevel >= LevelWarn:
		l.logger.LogAttrs(ctx, slog.LevelWarn, "发现SQL慢查询", append(attrs, slog.Duration("threshold", l.slowThreshold))...)

	// 普通查询
	case l.logLevel >= LevelInfo:
		l.logger.LogAttrs(ctx, slog.LevelInfo, "SQL执行成功", attrs...)
	}
}
