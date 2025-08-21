package xConfig

import (
	"go.uber.org/zap/zapcore"
	"runtime/debug"
)

type CustomCore struct {
	zapcore.Core
	encoder zapcore.Encoder
	fields  []zapcore.Field // 存储 With 添加的字段
}

// NewXlfCore 创建一个自定义的 `zapcore.Core` 实例，用于日志记录。
//
// 该函数通过组合传入的 `zapcore.Encoder`、`zapcore.WriteSyncer` 和 `zapcore.LevelEnabler` 创建
// 基于 `zapcore.NewCore` 的核心，并将其包装为 `CustomCore`，以支持扩展功能。
//
// 参数说明:
//   - encoder: 用于格式化日志条目的编码器，例如 JSON 编码器。
//   - syncer: 用于写入日志条目的输出目标，例如文件或标准输出。
//   - level: 日志级别使能器，用于控制哪些级别的日志可以被记录。
//
// 返回值:
//   - 返回一个实现了 `zapcore.Core` 的自定义核心实例，支持额外的字段管理功能。
func NewXlfCore(encoder zapcore.Encoder, syncer zapcore.WriteSyncer, level zapcore.LevelEnabler) zapcore.Core {
	return &CustomCore{
		Core:    zapcore.NewCore(encoder, syncer, level),
		encoder: encoder,
		fields:  []zapcore.Field{},
	}
}

func (c *CustomCore) With(fields []zapcore.Field) zapcore.Core {
	allFields := make([]zapcore.Field, len(c.fields)+len(fields))
	copy(allFields, c.fields)
	copy(allFields[len(c.fields):], fields)

	return &CustomCore{
		Core:    c.Core.With(fields),
		encoder: c.encoder,
		fields:  allFields,
	}
}

func (c *CustomCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	allFields := make([]zapcore.Field, len(c.fields)+len(fields))
	copy(allFields, c.fields)
	copy(allFields[len(c.fields):], fields)

	// 如果是 ERROR 级别以上且 Stack 为空，手动获取堆栈
	if entry.Level == zapcore.ErrorLevel && entry.Stack == "" {
		entry.Stack = string(debug.Stack())
	}

	return c.Core.Write(entry, allFields)
}

func (c *CustomCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}
