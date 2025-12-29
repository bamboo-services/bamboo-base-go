package xReg

import (
	"log/slog"
	"os"
	"strings"

	xConstEnv "github.com/bamboo-services/bamboo-base-go/constants/env"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
)

// LoggerInit 初始化日志记录器。
//
// 该方法配置并初始化全局日志记录器，通过 slog.Default() 访问。
// 它将 JSON 格式的日志输出到文件，同时将自定义格式的彩色日志输出到控制台。
//
// 日志格式:
//   - 文件: JSON 格式，Info 级别及以上。
//   - 控制台: <时间戳> [<日志等级>] [<trace>] [<日志类型>] <输出内容>，带颜色。
func (r *Reg) LoggerInit() {
	// 创建日志目录
	err := os.Mkdir(".logs", os.ModePerm)
	if err != nil && !os.IsExist(err) {
		panic("[INIT] 日志目录创建失败: " + err.Error())
	}

	// 打开日志文件
	file, err := os.OpenFile(".logs/log.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("[INIT] 日志文件打开失败: " + err.Error())
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
		File:        file,
		Level:       logLevel,
		IsDebugMode: debugMode,
		AddSource:   true,
	})
	handler.WithGroup("DEFU")

	// 设置为全局默认 logger
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// isDebugMode 判断是否处于调试模式。
func isDebugMode() bool {
	debug := strings.ToLower(os.Getenv(xConstEnv.Debug.String()))
	return debug == "true" || debug == "1" || debug == "yes" || debug == "on"
}
