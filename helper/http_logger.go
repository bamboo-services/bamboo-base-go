package xHelper

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	"github.com/gin-gonic/gin"
)

// HttpLogger 提供自定义的 HTTP 请求日志中间件。
//
// 该方法返回一个 Gin 中间件，用于记录每个 HTTP 请求的详细信息，
// 包括请求开始和请求完成两个阶段的日志。
//
// 请求开始日志记录：
//   - 基础信息：method, path, client_ip
//   - 调试模式额外信息：query, headers, body
//
// 请求完成日志记录：
//   - 响应状态码、耗时等结果信息
//
// 日志级别根据响应状态码自动调整：
//   - 2xx/3xx: INFO 级别
//   - 4xx: WARN 级别
//   - 5xx: ERROR 级别
//
// 日志使用项目标准 slog 库，并自动从 context 提取 trace ID，
// 使用 xLog.NamedHTTP 作为日志名称。
//
// 参数说明: 无。
//
// 返回值:
//   - 返回一个 `gin.HandlerFunc` 类型的函数，用于注册到 Gin 中间件链中。
//
// 注意: 建议将此中间件放置在 PanicRecovery 之后，确保能记录到 panic 恢复后的状态。
func HttpLogger() gin.HandlerFunc {
	// 创建带名称的日志器（复用实例，避免每次请求都创建）
	log := xLog.WithName(xLog.NamedHTTP)

	return func(c *gin.Context) {
		// 记录请求开始时间
		startTime := time.Now()

		// 获取请求基本信息
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		// ========== 请求开始日志 ==========
		// 基础日志属性
		args := []any{
			"method", method,
			"path", path,
			"client_ip", clientIP,
		}

		// 调试模式下添加详细信息
		if xCtxUtil.IsDebugMode() {
			// 添加查询参数
			if c.Request.URL.RawQuery != "" {
				args = append(args, "query", c.Request.URL.RawQuery)
			}

			// 添加请求头（脱敏）
			args = append(args, "headers", sanitizeHeaders(c.Request.Header))

			// 添加请求体（脱敏，根据请求方法判断）
			if shouldLogBody(c) {
				body := readRequestBody(c)
				sanitizedBody := sanitizeBody(body, c.Request.Header)
				if sanitizedBody != "" {
					args = append(args, "body", sanitizedBody)
				}
			}
		}

		log.SugarInfo(c.Request.Context(), "HTTP 请求开始", args...)

		// 放行请求
		c.Next()

		// ========== 请求完成日志 ==========
		// 请求处理完成后记录日志
		statusCode := c.Writer.Status()
		latency := time.Since(startTime)

		// 基础日志属性
		responseArgs := []any{
			"method", method,
			"path", path,
			"status", statusCode,
			"latency_ms", latency.Milliseconds(),
			"client_ip", clientIP,
		}

		// 根据状态码选择日志级别
		switch {
		case statusCode >= 500:
			log.SugarError(c.Request.Context(), "HTTP 请求完成", responseArgs...)
		case statusCode >= 400:
			log.SugarWarn(c.Request.Context(), "HTTP 请求完成", responseArgs...)
		default:
			log.SugarInfo(c.Request.Context(), "HTTP 请求完成", responseArgs...)
		}
	}
}

// maskSensitive 对敏感值进行脱敏处理。
//
// 该函数对敏感值进行脱敏处理，保留左/右各最多3个字符，中间用...替换。
// 短值（≤6字符）全部替换为 ******。
//
// 参数说明:
//   - value: 原始敏感值
//
// 返回值:
//   - 脱敏后的值
//
// 脱敏示例:
//   - "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." → "Bea...J9..."
//   - "mySecretToken123" → "myS...123"
//   - "abc" → "******"（≤6字符全部隐藏）
func maskSensitive(value string) string {
	if len(value) <= 6 {
		return "******"
	}

	leftLen := 3
	rightLen := 3
	if len(value) < leftLen+rightLen {
		leftLen = len(value) / 2
		rightLen = len(value) - leftLen
	}

	left := value[:leftLen]
	right := value[len(value)-rightLen:]
	return left + "..." + right
}

// shouldLogBody 判断是否应该记录请求体。
//
// 该函数根据请求方法和内容类型判断是否应该记录请求体。
// 只有调试模式且非 GET/HEAD 请求才记录，并且 Content-Type 必须是可解析的类型。
//
// 参数说明:
//   - c: Gin 上下文对象
//
// 返回值:
//   - true 表示应该记录请求体，false 表示不应该记录
func shouldLogBody(c *gin.Context) bool {
	// 只有调试模式才记录
	if !xCtxUtil.IsDebugMode() {
		return false
	}

	// GET/HEAD 请求没有请求体
	method := c.Request.Method
	if method == "GET" || method == "HEAD" {
		return false
	}

	// 检查 Content-Type
	contentType := c.Request.Header.Get("Content-Type")
	validTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"text/xml",
		"application/xml",
	}

	for _, validType := range validTypes {
		if strings.Contains(contentType, validType) {
			return true
		}
	}

	return false
}

// readRequestBody 读取请求体内容。
//
// 该函数读取请求体内容，并在读取后恢复请求体，以便后续处理器使用。
//
// 参数说明:
//   - c: Gin 上下文对象
//
// 返回值:
//   - 请求体内容（字符串），如果读取失败则返回空字符串
//
// 注意: 读取后需要恢复请求体（重要！），否则后续处理器无法读取请求体。
func readRequestBody(c *gin.Context) string {
	// 读取请求体
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}

	// 恢复请求体（重要！）
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes)
}

// sanitizeBody 脱敏处理请求体。
//
// 该函数根据 Content-Type 选择脱敏方式对请求体进行脱敏处理。
// 目前仅支持 JSON 格式的脱敏。
//
// 参数说明:
//   - body: 原始请求体内容
//   - header: 请求头，用于获取 Content-Type
//
// 返回值:
//   - 脱敏后的请求体内容，如果无法解析则返回原始内容
func sanitizeBody(body string, header map[string][]string) string {
	if body == "" {
		return body
	}

	contentType := ""
	for k, v := range header {
		if strings.ToLower(k) == "content-type" && len(v) > 0 {
			contentType = v[0]
			break
		}
	}

	// JSON 格式脱敏
	if strings.Contains(contentType, "application/json") {
		return sanitizeJSONBody(body)
	}

	// 其他格式暂不处理
	return body
}

// sanitizeJSONBody 脱敏 JSON 请求体中的敏感字段。
//
// 该函数解析 JSON 请求体，对敏感字段值进行脱敏处理，然后重新序列化。
//
// 参数说明:
//   - body: 原始 JSON 请求体内容
//
// 返回值:
//   - 脱敏后的 JSON 请求体内容，如果解析失败则返回原始内容
func sanitizeJSONBody(body string) string {
	// 敏感字段列表（小写）
	sensitiveFields := map[string]bool{
		"password":         true,
		"passwd":           true,
		"token":            true,
		"secret":           true,
		"apikey":           true,
		"api_key":          true,
		"x-api-key":        true,
		"access_token":     true,
		"refresh_token":    true,
		"old_password":     true,
		"new_password":     true,
		"confirm_password": true,
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return body
	}

	for key, value := range data {
		lowerKey := strings.ToLower(key)
		if sensitiveFields[lowerKey] {
			if strValue, ok := value.(string); ok {
				data[key] = maskSensitive(strValue)
			}
		}
	}

	sanitized, err := json.Marshal(data)
	if err != nil {
		return body
	}

	return string(sanitized)
}

// sanitizeHeaders 脱敏处理请求头信息。
//
// 该函数用于在调试模式下记录请求头时，对敏感字段进行脱敏处理，
// 防止密码、token 等敏感信息泄露到日志中。
//
// 脱敏规则：
//   - 完全移除: Set-Cookie
//   - 中间隐藏脱敏: Authorization, Cookie, Proxy-Authorization, X-API-Key, Access-Token
//
// 参数说明:
//   - headers: 原始的 http.Header 对象
//
// 返回值:
//   - 脱敏后的请求头 map
func sanitizeHeaders(headers map[string][]string) map[string]string {
	// 完全移除的黑名单
	blacklist := map[string]bool{
		"set-cookie": true,
	}

	// 需要脱敏的字段（中间隐藏）
	sensitiveMask := map[string]bool{
		"authorization":       true,
		"cookie":              true,
		"proxy-authorization": true,
		"x-api-key":           true,
		"access-token":        true,
	}

	sanitized := make(map[string]string)

	for key, values := range headers {
		lowerKey := strings.ToLower(key)

		// 检查是否在黑名单中（完全移除）
		if blacklist[lowerKey] {
			continue
		}

		// 检查是否需要脱敏
		if sensitiveMask[lowerKey] && len(values) > 0 {
			sanitized[key] = maskSensitive(values[0])
		} else {
			// 保留第一个值
			if len(values) > 0 {
				sanitized[key] = values[0]
			}
		}
	}

	return sanitized
}
