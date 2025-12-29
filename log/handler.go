package xLog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	"github.com/gin-gonic/gin"
)

// XlfHandler 自定义 slog Handler，支持彩色控制台输出和 JSON 文件输出
type XlfHandler struct {
	opts        slog.HandlerOptions
	mu          *sync.Mutex
	console     io.Writer
	file        io.Writer
	group       string // logger 名称（通过 WithGroup 设置）
	attrs       []slog.Attr
	isDebugMode bool
}

// HandlerConfig Handler 配置选项
type HandlerConfig struct {
	Console     io.Writer  // 控制台输出（可选，默认 os.Stdout）
	File        io.Writer  // 文件输出（可选）
	Level       slog.Level // 日志级别
	IsDebugMode bool       // 是否调试模式
	AddSource   bool       // 是否添加调用位置
}

// NewXlfHandler 创建自定义 slog Handler
//
// 参数说明:
//   - config: Handler 配置选项
//
// 返回值:
//   - slog.Handler: 自定义 Handler 实例
func NewXlfHandler(config HandlerConfig) slog.Handler {
	console := config.Console
	if console == nil {
		console = os.Stdout
	}

	return &XlfHandler{
		opts: slog.HandlerOptions{
			Level:     config.Level,
			AddSource: config.AddSource,
		},
		mu:          &sync.Mutex{},
		console:     console,
		file:        config.File,
		isDebugMode: config.IsDebugMode,
		attrs:       []slog.Attr{},
	}
}

// Enabled 判断指定级别是否启用
func (h *XlfHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle 处理日志记录
func (h *XlfHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 提取 trace ID
	trace := h.extractTrace(ctx)

	// 获取调用位置
	var caller string
	if h.opts.AddSource {
		caller = h.getCaller(r)
	}

	// 写入控制台（彩色格式）
	if h.console != nil {
		h.writeConsole(r, trace, caller)
	}

	// 写入文件（JSON 格式）
	if h.file != nil {
		h.writeFile(r, trace, caller)
	}

	return nil
}

// WithAttrs 添加属性
func (h *XlfHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &XlfHandler{
		opts:        h.opts,
		mu:          h.mu,
		console:     h.console,
		file:        h.file,
		group:       h.group,
		attrs:       newAttrs,
		isDebugMode: h.isDebugMode,
	}
}

// WithGroup 设置日志组名称（用作 logger name）
func (h *XlfHandler) WithGroup(name string) slog.Handler {
	return &XlfHandler{
		opts:        h.opts,
		mu:          h.mu,
		console:     h.console,
		file:        h.file,
		group:       name,
		attrs:       h.attrs,
		isDebugMode: h.isDebugMode,
	}
}

// extractTrace 从 context 中提取 trace ID
func (h *XlfHandler) extractTrace(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// 尝试从 gin.Context 获取
	if ginCtx, ok := ctx.(*gin.Context); ok {
		if trace, exists := ginCtx.Get(xConsts.ContextRequestKey.String()); exists {
			if traceStr, ok := trace.(string); ok {
				return traceStr
			}
		}
	}

	// 尝试从标准 context 获取
	if trace := ctx.Value(xConsts.ContextRequestKey); trace != nil {
		if traceStr, ok := trace.(string); ok {
			return traceStr
		}
	}

	// 尝试从 gin context key 获取
	if trace := ctx.Value(xConsts.ContextRequestKey.String()); trace != nil {
		if traceStr, ok := trace.(string); ok {
			return traceStr
		}
	}

	return ""
}

// getCaller 获取调用位置
func (h *XlfHandler) getCaller(r slog.Record) string {
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()
	if f.File != "" {
		// 只保留文件名和行号
		idx := strings.LastIndex(f.File, "/")
		if idx >= 0 {
			return fmt.Sprintf("%s:%d", f.File[idx+1:], f.Line)
		}
		return fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return ""
}

// writeConsole 写入控制台（彩色格式）
// 格式: 时间 [LEVEL] [CORE] [trace] [NAME] 消息
func (h *XlfHandler) writeConsole(r slog.Record, trace, caller string) {
	var buf strings.Builder

	// 时间戳（灰色）
	buf.WriteString("\033[90m")
	buf.WriteString(r.Time.Format("2006-01-02 15:04:05.000"))
	buf.WriteString("\033[0m")

	// 日志级别
	buf.WriteString(h.colorLevel(r.Level))

	// CORE 标识
	buf.WriteString(" \033[94m[CORE]\033[0m")

	// Trace ID（如果有）
	if trace != "" {
		buf.WriteString(" \033[34m[")
		buf.WriteString(trace)
		buf.WriteString("]\033[0m")
	}

	// Logger 名称
	if h.group != "" {
		buf.WriteString(h.colorName(h.group))
	}

	// 调用位置（如果启用）
	if caller != "" {
		buf.WriteString(" \033[90m")
		buf.WriteString(caller)
		buf.WriteString("\033[0m")
	}

	// 消息
	buf.WriteString(" ")
	buf.WriteString(r.Message)

	// 额外属性
	r.Attrs(func(a slog.Attr) bool {
		buf.WriteString(" ")
		buf.WriteString(a.Key)
		buf.WriteString("=")
		buf.WriteString(fmt.Sprintf("%v", a.Value.Any()))
		return true
	})

	// 预设属性
	for _, a := range h.attrs {
		buf.WriteString(" ")
		buf.WriteString(a.Key)
		buf.WriteString("=")
		buf.WriteString(fmt.Sprintf("%v", a.Value.Any()))
	}

	buf.WriteString("\n")

	// 错误级别添加堆栈
	if r.Level >= slog.LevelError {
		buf.WriteString(h.getStack())
		buf.WriteString("\n")
	}

	_, _ = io.WriteString(h.console, buf.String())
}

// writeFile 写入文件（JSON 格式）
func (h *XlfHandler) writeFile(r slog.Record, trace, caller string) {
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
	if caller != "" {
		entry["caller"] = caller
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
	if err == nil {
		data = append(data, '\n')
		_, _ = h.file.Write(data)
	}
}

// colorLevel 返回带颜色的日志级别
func (h *XlfHandler) colorLevel(level slog.Level) string {
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

// colorName 返回带颜色的 logger 名称
func (h *XlfHandler) colorName(name string) string {
	if len(name) != 4 {
		return fmt.Sprintf(" \033[96m[%s]\033[0m", name) // 亮青色
	}

	switch name {
	// 核心服务类 - 蓝色
	case xConsts.LogCONT, xConsts.LogSERV, xConsts.LogREPO, xConsts.LogCORE, xConsts.LogBASE, xConsts.LogMAIN:
		return fmt.Sprintf(" \033[34m[%s]\033[0m", name)
	// 路由网络类 - 黄色
	case xConsts.LogROUT, xConsts.LogHTTP, xConsts.LogGRPC, xConsts.LogSOCK, xConsts.LogCONN, xConsts.LogLINK:
		return fmt.Sprintf(" \033[33m[%s]\033[0m", name)
	// 安全认证类 - 红色
	case xConsts.LogAUTH, xConsts.LogUSER, xConsts.LogPERM, xConsts.LogROLE, xConsts.LogTOKN, xConsts.LogSIGN:
		return fmt.Sprintf(" \033[31m[%s]\033[0m", name)
	// 系统监控类 - 绿色
	case xConsts.LogLOGS, xConsts.LogMETR, xConsts.LogMONI, xConsts.LogPERF, xConsts.LogSTAT, xConsts.LogHEAL:
		return fmt.Sprintf(" \033[32m[%s]\033[0m", name)
	// 业务逻辑类 - 白色
	case xConsts.LogBUSI, xConsts.LogLOGC, xConsts.LogPROC, xConsts.LogFLOW, xConsts.LogTASK, xConsts.LogJOBS:
		return fmt.Sprintf(" \033[37m[%s]\033[0m", name)
	// 其他已定义的常量 - 橙色
	case xConsts.LogRECO, xConsts.LogUTIL, xConsts.LogFILT, xConsts.LogMIDE, xConsts.LogINIT, xConsts.LogTHOW, xConsts.LogRESU:
		return fmt.Sprintf(" \033[93m[%s]\033[0m", name)
	default:
		return fmt.Sprintf(" \033[35m[%s]\033[0m", name) // 紫色
	}
}

// getStack 获取堆栈信息
func (h *XlfHandler) getStack() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return "\033[31m" + string(buf[:n]) + "\033[0m"
}
