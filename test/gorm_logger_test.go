package test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"gorm.io/gorm"
)

// TestNewSlogLogger_DefaultConfig 测试默认配置
func TestNewSlogLogger_DefaultConfig(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{})

	// 测试普通查询（默认 Info 级别应该输出）
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM users", 10
	}, nil)

	output := buf.String()
	if !strings.Contains(output, "SQL执行") {
		t.Errorf("默认配置应输出 SQL 日志, got: %s", output)
	}
}

// TestSlogLogger_Trace_SlowQuery 测试慢查询识别
func TestSlogLogger_Trace_SlowQuery(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		SlowThreshold: 100, // 100ms
		LogLevel:      xLog.LevelInfo,
	})

	// 模拟慢查询（200ms > 100ms 阈值）
	begin := time.Now().Add(-200 * time.Millisecond)
	gormLogger.Trace(context.Background(), begin, func() (string, int64) {
		return "SELECT * FROM users", 10
	}, nil)

	output := buf.String()
	if !strings.Contains(output, "level=WARN") {
		t.Errorf("慢查询应输出 WARN 级别, got: %s", output)
	}
	if !strings.Contains(output, "SQL慢查询") {
		t.Errorf("慢查询应包含 'SQL慢查询' 消息, got: %s", output)
	}
}

// TestSlogLogger_Trace_NormalQuery 测试普通查询
func TestSlogLogger_Trace_NormalQuery(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		SlowThreshold: 200,
		LogLevel:      xLog.LevelInfo,
	})

	// 普通查询（无延迟）
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM users WHERE id = 1", 1
	}, nil)

	output := buf.String()
	if !strings.Contains(output, "level=INFO") {
		t.Errorf("普通查询应输出 INFO 级别, got: %s", output)
	}
	if !strings.Contains(output, "SQL执行") {
		t.Errorf("普通查询应包含 'SQL执行' 消息, got: %s", output)
	}
}

// TestSlogLogger_Trace_Error 测试错误记录
func TestSlogLogger_Trace_Error(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		LogLevel: xLog.LevelError,
	})

	// 模拟错误
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "INSERT INTO users (name) VALUES ('test')", 0
	}, errors.New("duplicate key"))

	output := buf.String()
	if !strings.Contains(output, "level=ERROR") {
		t.Errorf("错误应输出 ERROR 级别, got: %s", output)
	}
	if !strings.Contains(output, "SQL执行失败") {
		t.Errorf("错误应包含 'SQL执行失败' 消息, got: %s", output)
	}
	if !strings.Contains(output, "duplicate key") {
		t.Errorf("错误应包含错误信息, got: %s", output)
	}
}

// TestSlogLogger_Trace_IgnoreRecordNotFound 测试忽略记录未找到错误
func TestSlogLogger_Trace_IgnoreRecordNotFound(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		LogLevel:                  xLog.LevelError,
		IgnoreRecordNotFoundError: true,
	})

	// 模拟 ErrRecordNotFound
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM users WHERE id = 999", 0
	}, gorm.ErrRecordNotFound)

	output := buf.String()
	if output != "" {
		t.Errorf("应忽略 ErrRecordNotFound, got: %s", output)
	}
}

// TestSlogLogger_Trace_NotIgnoreRecordNotFound 测试不忽略记录未找到错误
func TestSlogLogger_Trace_NotIgnoreRecordNotFound(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		LogLevel:                  xLog.LevelError,
		IgnoreRecordNotFoundError: false, // 不忽略
	})

	// 模拟 ErrRecordNotFound
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM users WHERE id = 999", 0
	}, gorm.ErrRecordNotFound)

	output := buf.String()
	if !strings.Contains(output, "level=ERROR") {
		t.Errorf("不忽略时应输出 ERROR, got: %s", output)
	}
}

// TestSlogLogger_Silent 测试静默模式
func TestSlogLogger_Silent(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		LogLevel: xLog.LevelSilent,
	})

	// Silent 模式应该不输出任何日志
	gormLogger.Info(context.Background(), "test info %s", "message")
	gormLogger.Warn(context.Background(), "test warn %s", "message")
	gormLogger.Error(context.Background(), "test error %s", "message")
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)

	// 即使有错误也不输出
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 1
	}, errors.New("some error"))

	output := buf.String()
	if output != "" {
		t.Errorf("Silent 模式应无输出, got: %s", output)
	}
}

// TestSlogLogger_LogLevel_Error 测试仅 Error 级别
func TestSlogLogger_LogLevel_Error(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		LogLevel:      xLog.LevelError,
		SlowThreshold: 10, // 10ms
	})

	// 普通查询不应输出
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM users", 10
	}, nil)

	// 慢查询也不应输出（因为级别是 Error）
	begin := time.Now().Add(-200 * time.Millisecond)
	gormLogger.Trace(context.Background(), begin, func() (string, int64) {
		return "SELECT * FROM users", 10
	}, nil)

	output := buf.String()
	if output != "" {
		t.Errorf("Error 级别不应输出普通/慢查询日志, got: %s", output)
	}

	// 错误应该输出
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "INSERT INTO users", 0
	}, errors.New("some error"))

	output = buf.String()
	if !strings.Contains(output, "level=ERROR") {
		t.Errorf("Error 级别应输出错误日志, got: %s", output)
	}
}

// TestSlogLogger_LogLevel_Warn 测试 Warn 级别
func TestSlogLogger_LogLevel_Warn(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		LogLevel:      xLog.LevelWarn,
		SlowThreshold: 10, // 10ms
	})

	// 普通查询不应输出
	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM users", 10
	}, nil)

	output := buf.String()
	if output != "" {
		t.Errorf("Warn 级别不应输出普通查询日志, got: %s", output)
	}

	// 慢查询应该输出
	begin := time.Now().Add(-200 * time.Millisecond)
	gormLogger.Trace(context.Background(), begin, func() (string, int64) {
		return "SELECT * FROM orders", 100
	}, nil)

	output = buf.String()
	if !strings.Contains(output, "level=WARN") {
		t.Errorf("Warn 级别应输出慢查询日志, got: %s", output)
	}
}

// TestSlogLogger_InfoWarnError 测试 Info/Warn/Error 方法
func TestSlogLogger_InfoWarnError(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		LogLevel: xLog.LevelInfo,
	})

	gormLogger.Info(context.Background(), "info message: %s", "test")
	if !strings.Contains(buf.String(), "info message: test") {
		t.Errorf("Info 方法应输出格式化消息")
	}

	buf.Reset()
	gormLogger.Warn(context.Background(), "warn message: %d", 123)
	if !strings.Contains(buf.String(), "warn message: 123") {
		t.Errorf("Warn 方法应输出格式化消息")
	}

	buf.Reset()
	gormLogger.Error(context.Background(), "error message: %v", errors.New("test error"))
	if !strings.Contains(buf.String(), "error message: test error") {
		t.Errorf("Error 方法应输出格式化消息")
	}
}

// TestSlogLogger_LogMode 测试 LogMode 方法
func TestSlogLogger_LogMode(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		LogLevel: xLog.LevelInfo,
	})

	// 使用 LogMode 切换到 Silent
	silentLogger := gormLogger.LogMode(1) // 1 = Silent

	silentLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)

	output := buf.String()
	if output != "" {
		t.Errorf("LogMode(Silent) 后应无输出, got: %s", output)
	}
}

// TestSlogLogger_SQLAttributes 测试 SQL 属性输出
func TestSlogLogger_SQLAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLogger := xLog.NewSlogLogger(logger, xLog.GormLoggerConfig{
		LogLevel: xLog.LevelInfo,
	})

	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM users WHERE id = 1", 1
	}, nil)

	output := buf.String()

	// 验证日志属性
	if !strings.Contains(output, "elapsed_ms") {
		t.Errorf("应包含 elapsed_ms 属性, got: %s", output)
	}
	if !strings.Contains(output, "rows=1") {
		t.Errorf("应包含 rows 属性, got: %s", output)
	}
	if !strings.Contains(output, "SELECT * FROM users WHERE id = 1") {
		t.Errorf("应包含 SQL 语句, got: %s", output)
	}
}
