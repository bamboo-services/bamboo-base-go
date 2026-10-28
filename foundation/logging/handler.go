package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// LogHandler 是自定义的 slog.Handler 实现，支持彩色控制台输出与
// JSON 文件输出双写。
//
// 零值不可用，须通过 [NewLogHandler] 构造。Handle 全程持锁串行执行，
// 因此实例可被多个 goroutine 并发使用；经 [LogHandler.WithAttrs] 与
// [LogHandler.WithGroup] 派生的 Handler 与原 Handler 共享同一把
// 互斥锁与底层 Writer。
type LogHandler struct {
	opts        slog.HandlerOptions
	mu          *sync.Mutex
	console     io.Writer
	file        io.Writer
	group       string // logger 名称（通过 WithGroup 设置）
	attrs       []slog.Attr
	isDebugMode bool
}

// HandlerConfig 是 [NewLogHandler] 的构造配置。
type HandlerConfig struct {
	Console     io.Writer  // 控制台输出目标；为 nil 时默认使用 os.Stdout。
	File        io.Writer  // 文件输出目标；为 nil 时不输出文件，仅输出控制台。
	Level       slog.Level // 最低输出级别，零值表示 slog.LevelInfo。
	IsDebugMode bool       // 调试模式标记；当前版本仅随实例保存，不影响输出行为。
}

// NewLogHandler 根据配置创建控制台与文件双写的 slog.Handler。
//
// config.Console 为 nil 时默认写入 os.Stdout；config.File 为 nil 时
// 仅输出控制台。返回值可直接传给 [New] 或 slog.New。
func NewLogHandler(config HandlerConfig) slog.Handler {
	console := config.Console
	if console == nil {
		console = os.Stdout
	}

	return &LogHandler{
		opts: slog.HandlerOptions{
			Level:     config.Level,
			AddSource: false,
		},
		mu:          &sync.Mutex{},
		console:     console,
		file:        config.File,
		isDebugMode: config.IsDebugMode,
		attrs:       []slog.Attr{},
	}
}

// Enabled 判断指定级别是否达到输出阈值。
//
// 构造时未显式设置级别（零值）按 slog.LevelInfo 处理。
func (h *LogHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle 处理一条日志记录，执行控制台与文件双写输出。
//
// 控制台输出彩色文本，文件输出单行 JSON，两路输出均携带从
// context 的 [ctxkey.RequestKey] 提取的 TraceID；ERROR 级别的
// 控制台输出附加当前 goroutine 堆栈。若任一底层 Writer 写入失败，
// 将保留并以 errors.Join 汇总返回该错误，避免静默吞错，
// 另一路输出仍会执行。
func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 提取 context 中的 RequestKey (TraceID)
	contextUUID := h.extractContextUUID(ctx)

	var errs []error

	// 写入控制台（彩色格式）
	if h.console != nil {
		if err := h.writeConsole(r, contextUUID); err != nil {
			errs = append(errs, fmt.Errorf("logging: console write failed: %w", err))
		}
	}

	// 写入文件（JSON 格式）
	if h.file != nil {
		if err := h.writeFile(r, contextUUID); err != nil {
			errs = append(errs, fmt.Errorf("logging: file write failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

// WithAttrs 返回追加了属性的 Handler 派生副本。
//
// 派生副本与原 Handler 共享互斥锁与底层 Writer，原 Handler 不受影响。
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &LogHandler{
		opts:        h.opts,
		mu:          h.mu,
		console:     h.console,
		file:        h.file,
		group:       h.group,
		attrs:       newAttrs,
		isDebugMode: h.isDebugMode,
	}
}

// WithGroup 返回携带日志组名称的 Handler 派生副本。
//
// 组名称在控制台输出中作为 logger 名称渲染（如 [CORE]），
// 在 JSON 输出中写入 "logger" 字段；派生副本与原 Handler
// 共享互斥锁与底层 Writer。
func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{
		opts:        h.opts,
		mu:          h.mu,
		console:     h.console,
		file:        h.file,
		group:       name,
		attrs:       h.attrs,
		isDebugMode: h.isDebugMode,
	}
}

// extractContextUUID 从 context 中读取 ctxkey.RequestKey 作为 TraceID。
//
// 直接从 context.Value(ctxkey.RequestKey) 提取，支持 string 与 fmt.Stringer 类型。
func (h *LogHandler) extractContextUUID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	val := ctx.Value(ctxkey.RequestKey)
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// writeConsole 写入控制台（彩色格式）
// 格式: 时间 [LEVEL] [trace] [NAME] 消息
//
//	变量（换行棕色显示）
func (h *LogHandler) writeConsole(r slog.Record, contextUUID string) error {
	var buf strings.Builder

	// 时间戳（灰色）
	buf.WriteString("\033[90m")
	buf.WriteString(r.Time.Format("2006-01-02 15:04:05.000"))
	buf.WriteString("\033[0m")

	// 日志级别
	buf.WriteString(h.colorLevel(r.Level))

	// Trace ID（如果有）
	if contextUUID != "" {
		buf.WriteString(" \033[34m[")
		buf.WriteString(contextUUID)
		buf.WriteString("]\033[0m")
	}

	// Logger 名称
	if h.group != "" {
		buf.WriteString(h.colorName(h.group))
	}

	// Option 名称
	optionNames := make([]string, 0)
	for _, attr := range h.attrs {
		if !strings.HasPrefix(attr.Key, "option_name_") {
			continue
		}
		if attr.Value.Kind() == slog.KindString {
			optionNames = append(optionNames, attr.Value.String())
			continue
		}
		if value, ok := attr.Value.Any().(string); ok {
			optionNames = append(optionNames, value)
		}
	}
	for _, option := range optionNames {
		buf.WriteString(" \033[96m[")
		buf.WriteString(option)
		buf.WriteString("]\033[0m")
		buf.WriteString("\t")
	}

	// 消息
	buf.WriteString(" >> ")
	buf.WriteString(r.Message)

	// 收集所有属性
	var hasAttrs bool

	// 额外属性（棕色，换行显示）
	r.Attrs(func(a slog.Attr) bool {
		if strings.HasPrefix(a.Key, "option_name_") {
			return true
		}
		if !hasAttrs {
			hasAttrs = true
		}
		buf.WriteString("\n    \033[38;5;130m")
		buf.WriteString(a.Key)
		buf.WriteString("\033[0m=\033[38;5;180m")
		_, _ = fmt.Fprintf(&buf, "%v", a.Value.Any())
		buf.WriteString("\033[0m")
		return true
	})

	// 预设属性（棕色，换行显示）
	for _, a := range h.attrs {
		if strings.HasPrefix(a.Key, "option_name_") {
			continue
		}
		if !hasAttrs {
			hasAttrs = true
		}
		buf.WriteString("\n    \033[38;5;130m")
		buf.WriteString(a.Key)
		buf.WriteString("\033[0m=\033[38;5;180m")
		_, _ = fmt.Fprintf(&buf, "%v", a.Value.Any())
		buf.WriteString("\033[0m")
	}

	buf.WriteString("\n")

	// 错误级别添加堆栈
	if r.Level >= slog.LevelError {
		buf.WriteString(h.getStack())
		buf.WriteString("\n")
	}

	_, err := io.WriteString(h.console, buf.String())
	return err
}

// writeFile 写入文件（JSON 格式）
func (h *LogHandler) writeFile(r slog.Record, trace string) error {
	entry := map[string]interface{}{
		"time":    r.Time.Format(time.RFC3339Nano),
		"level":   r.Level.String(),
		"message": r.Message,
	}

	if trace != "" {
		entry["trace"] = trace
	}
	if h.group != "" {
		entry["logger"] = h.group
	}

	// 添加额外属性
	r.Attrs(func(a slog.Attr) bool {
		entry[a.Key] = a.Value.Any()
		return true
	})

	// 添加预设属性
	for _, a := range h.attrs {
		entry[a.Key] = a.Value.Any()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = h.file.Write(data)
	return err
}

// colorLevel 返回带颜色的日志级别。
func (h *LogHandler) colorLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return " \033[36m[DEBU]\033[0m" // 青色
	case slog.LevelInfo:
		return " \033[32m[INFO]\033[0m" // 绿色
	case slog.LevelWarn:
		return " \033[33m[WARN]\033[0m" // 黄色
	case slog.LevelError:
		return " \033[31m[ERRO]\033[0m" // 红色
	default:
		return " \033[32m[INFO]\033[0m"
	}
}

// colorName 返回带颜色的 logger 名称。
func (h *LogHandler) colorName(name string) string {
	if len(name) != 4 {
		return fmt.Sprintf(" \033[96m[%s]\033[0m", name) // 亮青色
	}

	switch name {
	// 核心服务类 - 蓝色
	case NamedCONT, NamedSERV, NamedLOGC, NamedREPO, NamedCORE, NamedBASE, NamedMAIN:
		return fmt.Sprintf(" \033[34m[%s]\033[0m", name)
	// 路由网络类 - 黄色
	case NamedROUT, NamedHTTP, NamedGRPC, NamedSOCK, NamedCONN, NamedLINK:
		return fmt.Sprintf(" \033[33m[%s]\033[0m", name)
	// 安全认证类 - 红色
	case NamedAUTH, NamedUSER, NamedPERM, NamedROLE, NamedTOKN, NamedSIGN:
		return fmt.Sprintf(" \033[31m[%s]\033[0m", name)
	// 业务逻辑类 - 白色
	case NamedBUSI, NamedPROC, NamedFLOW, NamedTASK, NamedJOBS:
		return fmt.Sprintf(" \033[37m[%s]\033[0m", name)
	// 其他已定义的常量 - 橙色
	case NamedRECO, NamedUTIL, NamedFILT, NamedMIDE, NamedVALD, NamedINIT, NamedTHOW, NamedRESU:
		return fmt.Sprintf(" \033[93m[%s]\033[0m", name)
	default:
		return fmt.Sprintf(" \033[35m[%s]\033[0m", name) // 紫色
	}
}

// getStack 获取堆栈信息。
func (h *LogHandler) getStack() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return "\033[31m" + string(buf[:n]) + "\033[0m"
}
