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

	xConsts "github.com/bamboo-services/bamboo-base-go/context"
	"github.com/gin-gonic/gin"
)

// LogHandler 自定义 slog Handler，支持彩色控制台输出和 JSON 文件输出
type LogHandler struct {
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
}

// NewLogHandler 创建自定义 slog Handler
//
// 参数说明:
//   - config: Handler 配置选项
//
// 返回值:
//   - slog.Handler: 自定义 Handler 实例
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

// Enabled 判断指定级别是否启用
func (h *LogHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle 处理日志记录
func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 提取 contextUUID ID
	contextUUID := h.extractContextUUID(ctx)

	// 写入控制台（彩色格式）
	if h.console != nil {
		h.writeConsole(r, contextUUID)
	}

	// 写入文件（JSON 格式）
	if h.file != nil {
		h.writeFile(r, contextUUID)
	}

	return nil
}

// WithAttrs 添加属性
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

// WithGroup 设置日志组名称（用作 logger name）
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

// extractContextUUID 从 context 中提取 trace ID
//
// 支持的 context 类型:
//   - gin.Context: 从 Gin 请求上下文中提取
//   - gorm.DB context: 从 GORM 数据库操作上下文中提取
//   - 标准 context.Context: 从任意标准 context 中提取
func (h *LogHandler) extractContextUUID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// 1. 尝试从 gin.Context 指针中提取 (HTTP 请求场景)
	if ginCtx, ok := ctx.(*gin.Context); ok {
		// 先尝试从 gin.Context 自身的存储中获取
		if contextUUID, exists := ginCtx.Get(string(xConsts.RequestKey)); exists {
			if traceStr, ok := contextUUID.(string); ok {
				return traceStr
			}
		}
		// 再尝试从 Request.Context() 中获取
		if contextUUID := ginCtx.Request.Context().Value(xConsts.RequestKey); contextUUID != nil {
			if traceStr, ok := contextUUID.(string); ok {
				return traceStr
			}
		}
	}

	// 2. 从标准 context.Context 中提取 (包括 GORM 数据库操作场景)
	// GORM 使用标准 context，通过 db.WithContext(ctx) 传递
	// context.Value() 会自动沿着 context 链向上查找
	if contextUUID := ctx.Value(xConsts.RequestKey); contextUUID != nil {
		if traceStr, ok := contextUUID.(string); ok {
			return traceStr
		}
	}

	return ""
}

// writeConsole 写入控制台（彩色格式）
// 格式: 时间 [LEVEL] [trace] [NAME] 消息
//
//	变量（换行棕色显示）
func (h *LogHandler) writeConsole(r slog.Record, contextUuid string) {
	var buf strings.Builder

	// 时间戳（灰色）
	buf.WriteString("\033[90m")
	buf.WriteString(r.Time.Format("2006-01-02 15:04:05.000"))
	buf.WriteString("\033[0m")

	// 日志级别
	buf.WriteString(h.colorLevel(r.Level))

	// Trace ID（如果有）
	if contextUuid != "" {
		buf.WriteString(" \033[34m[")
		buf.WriteString(contextUuid)
		buf.WriteString("]\033[0m")
	}

	// Logger 名称
	if h.group != "" {
		buf.WriteString(h.colorName(h.group))
	}

	// 消息
	buf.WriteString(" ")
	buf.WriteString(r.Message)

	// 收集所有属性
	var hasAttrs bool

	// 额外属性（棕色，换行显示）
	r.Attrs(func(a slog.Attr) bool {
		if !hasAttrs {
			hasAttrs = true
		}
		buf.WriteString("\n    \033[38;5;130m")
		buf.WriteString(a.Key)
		buf.WriteString("\033[0m=\033[38;5;180m")
		buf.WriteString(fmt.Sprintf("%v", a.Value.Any()))
		buf.WriteString("\033[0m")
		return true
	})

	// 预设属性（棕色，换行显示）
	for _, a := range h.attrs {
		if !hasAttrs {
			hasAttrs = true
		}
		buf.WriteString("\n    \033[38;5;130m")
		buf.WriteString(a.Key)
		buf.WriteString("\033[0m=\033[38;5;180m")
		buf.WriteString(fmt.Sprintf("%v", a.Value.Any()))
		buf.WriteString("\033[0m")
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
func (h *LogHandler) writeFile(r slog.Record, trace string) {
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
	if err == nil {
		data = append(data, '\n')
		_, _ = h.file.Write(data)
	}
}

// colorLevel 返回带颜色的日志级别
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

// colorName 返回带颜色的 logger 名称
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

// getStack 获取堆栈信息
func (h *LogHandler) getStack() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return "\033[31m" + string(buf[:n]) + "\033[0m"
}
