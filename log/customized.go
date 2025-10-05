package xLog

import (
	"github.com/bamboo-services/bamboo-base-go/constants"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// ConsoleLevelEncoder 是一个自定义的日志等级编码器，
// 它会为控制台输出添加颜色和方括号。
func ConsoleLevelEncoder(level zapcore.Level, enc *buffer.Buffer) {
	switch level {
	case zapcore.DebugLevel:
		enc.AppendString(" \u001B[36m[DEBU]\u001B[0m") // 青色
	case zapcore.WarnLevel:
		enc.AppendString(" \u001B[33m[WARN]\u001B[0m") // 黄色
	case zapcore.ErrorLevel:
		enc.AppendString(" \u001B[31m[ERRO]\u001B[0m") // 红色
	case zapcore.DPanicLevel:
		enc.AppendString(" \u001B[31m[DPAN]\u001B[0m") // 紫色
	case zapcore.PanicLevel:
		enc.AppendString(" \u001B[31m[PANI]\u001B[0m") // 紫色
	case zapcore.FatalLevel:
		enc.AppendString(" \u001B[31m[FATA]\u001B[0m") // 紫色
	default:
		enc.AppendString(" \u001B[32m[INFO]\u001B[0m") // 绿色
	}
}

// ConsoleNameEncoder 根据提供的名称对控制台标识符进行编码并添加颜色格式化字符串至缓冲区。
func ConsoleNameEncoder(name string, enc *buffer.Buffer) {
	// 检查是否为四字符名称
	if len(name) != 4 {
		enc.AppendString(" \u001B[96m[" + name + "]\u001B[0m ") // 亮青色：非四字符名称
		return
	}

	switch name {
	// 核心服务类 - 蓝色
	case xConsts.LogCONT, xConsts.LogSERV, xConsts.LogREPO, xConsts.LogCORE, xConsts.LogBASE, xConsts.LogMAIN:
		enc.AppendString(" \u001B[34m[" + name + "]\u001B[0m ")
	// 路由网络类 - 黄色
	case xConsts.LogROUT, xConsts.LogHTTP, xConsts.LogGRPC, xConsts.LogSOCK, xConsts.LogCONN, xConsts.LogLINK:
		enc.AppendString(" \u001B[33m[" + name + "]\u001B[0m ")
	// 安全认证类 - 红色
	case xConsts.LogAUTH, xConsts.LogUSER, xConsts.LogPERM, xConsts.LogROLE, xConsts.LogTOKN, xConsts.LogSIGN:
		enc.AppendString(" \u001B[31m[" + name + "]\u001B[0m ")
	// 系统监控类 - 绿色
	case xConsts.LogLOGS, xConsts.LogMETR, xConsts.LogMONI, xConsts.LogPERF, xConsts.LogSTAT, xConsts.LogHEAL:
		enc.AppendString(" \u001B[32m[" + name + "]\u001B[0m ")
	// 业务逻辑类 - 白色
	case xConsts.LogBUSI, xConsts.LogLOGC, xConsts.LogPROC, xConsts.LogFLOW, xConsts.LogTASK, xConsts.LogJOBS:
		enc.AppendString(" \u001B[37m[" + name + "]\u001B[0m ")
	// 其他已定义的常量 - 橙色
	case xConsts.LogRECO, xConsts.LogUTIL, xConsts.LogFILT, xConsts.LogMIDE, xConsts.LogINIT, xConsts.LogTHOW, xConsts.LogRESU:
		enc.AppendString(" \u001B[93m[" + name + "]\u001B[0m ")
	// 未分类的四字符名称 - 紫色（默认）
	default:
		enc.AppendString(" \u001B[35m[" + name + "]\u001B[0m ")
	}
}
