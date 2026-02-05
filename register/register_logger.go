package xReg

import (
	"fmt"
	"log/slog"
	"os"

	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
)

// loggerInit 初始化并设置全局日志记录器。
//
// 该方法根据当前运行模式（调试/发布）配置日志级别，并创建一个支持控制台输出与文件切割归档的日志记录器。
// 文件日志按日期归档，单个文件大小限制为 10MB。初始化失败会触发 panic。
func (r *Reg) loggerInit() {
	// 创建日志切割写入器
	rotator, err := xLog.NewRotatingWriter(xLog.RotatorConfig{
		Dir:      ".logs",
		BaseName: "log",
		Ext:      ".log",
		MaxSize:  10 * 1024 * 1024, // 10MB
	})
	if err != nil {
		panic(fmt.Sprintf("日志写入器创建失败: %v", err))
	}

	// 确定日志级别
	logLevel := slog.LevelInfo
	debugMode := xCtxUtil.IsDebugMode()
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
