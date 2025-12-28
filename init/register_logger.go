package xInit

import (
	"os"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerInit 初始化日志记录器。
//
// 该方法配置并初始化全局日志记录器，通过 zap.L() 访问。
// 它将 JSON 格式的日志输出到文件，同时将自定义格式的彩色日志输出到控制台。
//
// 日志格式:
//   - 文件: JSON 格式，Info 级别及以上。
//   - 控制台: <时间戳> [<日志等级>] [<日志类型>] <输出内容>，Debug 级别及以上，带颜色。
func (r *Reg) LoggerInit() {
	// --- 1. 文件日志核心 (JSON格式) ---
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	// 使用纳秒级的时间戳
	fileEncoderConfig.EncodeTime = zapcore.EpochNanosTimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	// 打开日志文件
	err := os.Mkdir("logs", os.ModePerm)
	if err != nil && !os.IsExist(err) {
		panic("[INIT] 日志目录创建失败: " + err.Error())
	}
	file, err := os.OpenFile("logs/log.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("[INIT] 日志文件打开失败: " + err.Error())
	}
	fileWriter := zapcore.AddSync(file)

	// 创建文件日志核心，只记录 Info 及以上级别的日志
	fileCore := zapcore.NewCore(jsonEncoder, fileWriter, zapcore.InfoLevel)

	// --- 2. 控制台日志核心 (自定义彩色格式) ---

	// 创建控制台日志核心，记录 Debug 及以上级别的日志
	logLevel := zapcore.InfoLevel
	if isDebugMode() {
		logLevel = zapcore.DebugLevel
	}
	consoleCore := xLog.NewXlfCore(
		xLog.NewXlfConsoleEncoder(),
		zapcore.AddSync(os.Stdout),
		logLevel,
	)

	// --- 3. 合并核心并创建 Logger ---
	// 使用 NewTee 将多个核心合并，实现日志同时输出到不同地方
	core := zapcore.NewTee(
		fileCore,
		consoleCore,
	)

	// 创建最终的日志记录器，AddCaller() 用于记录调用位置
	logger := zap.New(core, zap.AddCaller())

	// 注册为全局 logger，通过 zap.L() 访问
	zap.ReplaceGlobals(logger)
}

// isDebugMode 判断是否处于调试模式。
func isDebugMode() bool {
	debug := strings.ToLower(os.Getenv("XLF_DEBUG"))
	return debug == "true" || debug == "1" || debug == "yes" || debug == "on"
}
