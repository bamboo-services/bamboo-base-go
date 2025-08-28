package xLog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// LogEncoder 是自定义的编码器，用于控制台输出格式
type LogEncoder struct {
	zapcore.Encoder
	fields []zapcore.Field
	pool   buffer.Pool
}

// NewXlfConsoleEncoder 创建一个自定义的控制台编码器。
//
// 该函数返回一个 `zapcore.Encoder` 实例，使用了自定义的控制台输出格式。
// 编码器可以为日志增加格式化能力，包括时间、日志级别和字段渲染逻辑等。
//
// 返回值:
//   - `zapcore.Encoder`: 一个自定义实现的控制台编码器实例。
func NewXlfConsoleEncoder() zapcore.Encoder {
	config := zap.NewProductionEncoderConfig()
	return &LogEncoder{
		Encoder: zapcore.NewConsoleEncoder(config),
		pool:    buffer.NewPool(),
		fields:  make([]zapcore.Field, 0),
	}
}

// Clone 创建并返回当前 `LogEncoder` 的深拷贝实例。
//
// 该方法用于生成编码器的独立副本，包括基础 `Encoder` 的拷贝和新的缓冲区池实例。
// 通过这种方式，可以在多个场景中并发使用相同的编码器配置，而无需干扰彼此状态。
//
// 注意: 返回的新实例拥有独立的缓冲池 `buffer.Pool`，以确保线程安全。
func (e *LogEncoder) Clone() zapcore.Encoder {
	return &LogEncoder{
		Encoder: e.Encoder.Clone(),
		pool:    buffer.NewPool(),
	}

}

// EncodeEntry 自定义日志条目的编码方法。
//
// 该方法使用自定义格式将日志条目和字段编码为缓冲区，特别适用于控制台输出。
// 它包括时间、日志等级、Logger 名称、消息内容及错误堆栈（若存在）。
//
// 参数说明:
//   - entry: 包含日志条目的核心信息，例如时间、等级、消息和堆栈。
//   - fields: 附加的日志字段集合，支持格式化特定键值。
//
// 返回值:
//   - 缓冲区指针，包含已编码的日志内容。
//   - 错误值，如果在编码过程中遇到任何问题，则返回非 nil 的错误。
//
// 注意: 此方法生成的输出经过了颜色编码，主要用于提升控制台可读性，不适用于纯文本日志文件。
func (e *LogEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf := e.pool.Get()

	// 定义时间格式
	buf.AppendString("\u001B[90m" + entry.Time.Format("2006-01-02 15:04:05.000") + "\u001B[0m")

	// 定义唯一键
	if len(fields) > 0 {
		for _, field := range fields {
			if field.Key == "trace" {
				// 如果字段是 trace，则格式化为 JSON
				buf.AppendString(" \u001B[34m[" + field.String + "]\u001B[0m")
			}
		}
	}

	// 添加日志等级和名称
	ConsoleLevelEncoder(entry.Level, buf)
	ConsoleNameEncoder(entry.LoggerName, buf)
	buf.AppendString(entry.Message)

	// 添加额外的字段
	buf.AppendString("\n")

	// 检查是否是 ERROR 以上级别
	if entry.Level >= zapcore.ErrorLevel {
		buf.AppendString(" \u001B[31m" + entry.Stack + "\u001B[0m ")
		buf.AppendString("\n")
	}
	return buf, nil
}
