package xReg

import (
	"log/slog"
	"os"

	xLog "github.com/bamboo-services/bamboo-base-go/log"
)

// LoggerInit 初始化日志记录器。
//
// 该方法配置并初始化全局日志记录器，通过 slog.Default() 访问。
// 它将 JSON 格式的日志输出到文件，同时将自定义格式的彩色日志输出到控制台。
//
// 日志格式:
//   - 文件: JSON 格式，Info 级别及以上，支持自动切割和归档。
//   - 控制台: <时间戳> [<日志等级>] [<trace>] [<日志类型>] <输出内容>，带颜色。
//
// 日志切割:
//   - 按大小切割: 单文件超过 10MB 自动切割为 log.0.log, log.1.log ...
//   - 按时间归档: 每天 00:00:05 将前一天日志打包为 logger-yyyy-MM-dd.tar.gz
func (r *Reg) LoggerInit() {
	// 创建日志切割写入器
	rotator, err := xLog.NewRotatingWriter(xLog.RotatorConfig{
		Dir:      ".logs",
		BaseName: "log",
		Ext:      ".log",
		MaxSize:  10 * 1024 * 1024, // 10MB
	})
	if err != nil {
		panic("[INIT] 日志切割器创建失败: " + err.Error())
	}

	// 确定日志级别
	logLevel := slog.LevelInfo
	debugMode := isDebugMode()
	if debugMode {
		logLevel = slog.LevelDebug
	}

	// 创建自定义 Handler
	handler := xLog.NewLogHandler(xLog.HandlerConfig{
		Console:     os.Stdout,
		File:        rotator,
		Level:       logLevel,
		IsDebugMode: debugMode,
	})

	// 设置为全局默认 logger
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
