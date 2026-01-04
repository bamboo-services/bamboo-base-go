package xVaild

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	xError "github.com/bamboo-services/bamboo-base-go/error"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationErrorDetail 验证错误详情
type ValidationErrorDetail struct {
	Field   string      `json:"field"`   // 字段名（英文）
	Tag     string      `json:"tag"`     // 验证标签
	Message string      `json:"message"` // 错误消息
	Value   interface{} `json:"value"`   // 字段值
}

// HandleValidationError 处理验证错误并返回友好的响应
//
// 该函数将验证错误转换为用户友好的错误响应，支持中文翻译。
// 对于标准的验证错误，会提取所有字段的错误详情；
// 对于非标准验证错误（如 JSON 解析错误），会添加通用错误详情。
//
// 参数：
//   - ctx: Gin 上下文
//   - bindErr: 绑定或验证时产生的错误
//
// 响应格式：
//
//	{
//	    "code": 40014,
//	    "message": "请求体错误",
//	    "error_message": "用户名格式不正确",
//	    "data": [
//	        {
//	            "field": "username",
//	            "tag": "regexp",
//	            "message": "用户名格式不正确",
//	            "value": "abc"
//	        }
//	    ]
//	}
func HandleValidationError(ctx *gin.Context, bindErr error) {
	var errorDetails []ValidationErrorDetail
	var firstErrorMessage string

	// 尝试解析为标准的 ValidationErrors
	var validationErrors validator.ValidationErrors
	if errors.As(bindErr, &validationErrors) {
		// 使用翻译器翻译错误
		translatedErrors := TranslateError(bindErr)

		for i, fe := range validationErrors {
			// 优先使用翻译后的错误消息
			errorMessage := translatedErrors[fe.Field()]
			if errorMessage == "" {
				// 如果翻译失败，使用旧的方法
				errorMessage = GetValidationErrorMessage(fe)
			}

			// 记录第一个错误作为主要错误消息
			if i == 0 {
				firstErrorMessage = errorMessage
			}

			errorDetails = append(errorDetails, ValidationErrorDetail{
				Field:   fe.Field(),
				Tag:     fe.Tag(),
				Message: errorMessage,
				Value:   fe.Value(),
			})
		}
	} else {
		// 处理非标准验证错误（如 JSON 解析错误、未注册的验证器等）
		log := xLog.WithName(xLog.NamedVALD)
		log.Warn(ctx.Request.Context(),
			"验证错误无法解析为 ValidationErrors",
			slog.String("error", bindErr.Error()),
			slog.String("error_type", fmt.Sprintf("%T", bindErr)),
		)

		// 解析绑定错误，提取字段名、值和友好消息
		detail := parseBindingErrorDetail(ctx, bindErr)
		firstErrorMessage = detail.Message

		// 添加错误详情
		errorDetails = append(errorDetails, detail)
	}

	// 如果没有详细错误，使用默认消息
	if firstErrorMessage == "" {
		firstErrorMessage = "请求数据验证失败"
	}

	// 创建错误响应
	// 注意：使用 errorDetails... 展开切片，避免嵌套数组
	_ = ctx.Error(xError.NewErrorHasData(
		ctx,
		xError.BodyError,
		xError.ErrMessage(firstErrorMessage),
		false,
		bindErr,
		interfaceSlice(errorDetails)...,
	))
}

// interfaceSlice 将 ValidationErrorDetail 切片转换为 interface{} 切片
// 这是为了配合 NewErrorHasData 的可变参数 ...interface{}
func interfaceSlice(details []ValidationErrorDetail) []interface{} {
	result := make([]interface{}, len(details))
	for i, detail := range details {
		result[i] = detail
	}
	return result
}

// parseBindingErrorDetail 解析绑定错误并提取字段名、值和友好的中文错误消息
//
// 该函数处理常见的 JSON 绑定错误，包括：
//   - 时间格式解析错误：提取时间值，提示正确的时间格式
//   - JSON 语法错误：提示 JSON 格式错误
//   - 类型不匹配：提取字段名和值，提示字段类型错误
//   - 其他错误：尽可能提取字段信息
//
// 参数：
//   - ctx: Gin 上下文（用于读取请求体）
//   - err: 绑定过程中产生的错误
//
// 返回值：
//   - ValidationErrorDetail: 包含字段名、标签、消息和值的错误详情
func parseBindingErrorDetail(ctx *gin.Context, err error) ValidationErrorDetail {
	errMsg := err.Error()
	detail := ValidationErrorDetail{
		Field:   "unknown",
		Tag:     "binding",
		Message: errMsg,
		Value:   nil,
	}

	// 尝试从 json.UnmarshalTypeError 中获取字段名
	// 递归展开错误链
	var unmarshalTypeErr *json.UnmarshalTypeError
	currentErr := err
	for currentErr != nil {
		if errors.As(currentErr, &unmarshalTypeErr) {
			detail.Field = unmarshalTypeErr.Field
			break
		}
		// 尝试展开 Unwrap
		if unwrapper, ok := currentErr.(interface{ Unwrap() error }); ok {
			currentErr = unwrapper.Unwrap()
		} else {
			break
		}
	}

	// 尝试从 JSON unmarshal 错误消息中提取字段名
	// 格式1: json: cannot unmarshal string into Go struct field RegisterRequest.birthday of type time.Time
	if detail.Field == "unknown" || detail.Field == "" {
		if strings.Contains(errMsg, "Go struct field") {
			if parts := strings.Split(errMsg, "Go struct field "); len(parts) > 1 {
				if fieldParts := strings.Split(parts[1], " of type"); len(fieldParts) > 0 {
					// 提取字段路径，如 "RegisterRequest.birthday"
					fieldPath := strings.TrimSpace(fieldParts[0])
					// 取最后一段作为字段名
					if dotIdx := strings.LastIndex(fieldPath, "."); dotIdx != -1 {
						detail.Field = fieldPath[dotIdx+1:]
					} else {
						detail.Field = fieldPath
					}
				}
			}
		}
	}

	// 格式2: 尝试从更详细的错误消息中提取
	// 有些错误可能包含字段路径，如 "Time.birthday: parsing time..."
	if detail.Field == "unknown" || detail.Field == "" {
		if idx := strings.Index(errMsg, ": parsing time"); idx > 0 {
			fieldPath := errMsg[:idx]
			if dotIdx := strings.LastIndex(fieldPath, "."); dotIdx != -1 {
				detail.Field = fieldPath[dotIdx+1:]
			}
		}
	}

	// 检测时间格式解析错误
	// 格式: parsing time "2025-11-05" as "2006-01-02T15:04:05Z07:00": cannot parse "" as "T"
	if strings.Contains(errMsg, "parsing time") {
		// 提取时间值（在引号之间）
		if detail.Value == nil {
			if startIdx := strings.Index(errMsg, "\""); startIdx != -1 {
				if endIdx := strings.Index(errMsg[startIdx+1:], "\""); endIdx != -1 {
					detail.Value = errMsg[startIdx+1 : startIdx+1+endIdx]
				}
			}
		}

		// 如果还是没有字段名，尝试从请求体中智能匹配
		if (detail.Field == "unknown" || detail.Field == "") && detail.Value != nil {
			if field := findFieldByValueInRequest(ctx, detail.Value); field != "" {
				detail.Field = field
			}
		}

		// 设置友好的错误消息
		if strings.Contains(errMsg, "as \"2006-01-02T15:04:05Z07:00\"") {
			detail.Message = "时间格式不正确，请使用 ISO8601 格式（如：2025-11-05T00:00:00Z）"
		} else {
			detail.Message = "时间格式不正确"
		}

		// 如果已经提取到字段名，补充到消息中
		if detail.Field != "unknown" && detail.Field != "" {
			detail.Message = detail.Field + " " + detail.Message
		}
		return detail
	}

	// 检测 JSON 语法错误
	if strings.Contains(errMsg, "invalid character") ||
		strings.Contains(errMsg, "unexpected end of JSON") ||
		strings.Contains(errMsg, "looking for beginning of value") {
		detail.Message = "JSON 格式错误，请检查请求体格式"
		return detail
	}

	// 检测类型不匹配错误
	if strings.Contains(errMsg, "cannot unmarshal") {
		// 尝试提取值（在引号之间）
		if detail.Value == nil {
			if startIdx := strings.Index(errMsg, "\""); startIdx != -1 {
				if endIdx := strings.Index(errMsg[startIdx+1:], "\""); endIdx != -1 {
					detail.Value = errMsg[startIdx+1 : startIdx+1+endIdx]
				}
			}
		}

		if strings.Contains(errMsg, "into Go value of type") {
			detail.Message = "字段类型不匹配，请检查请求参数类型"
		} else {
			detail.Message = "数据类型错误"
		}

		// 如果已经提取到字段名，补充到消息中
		if detail.Field != "unknown" && detail.Field != "" {
			detail.Message = detail.Field + " " + detail.Message
		}
		return detail
	}

	// 检测必填字段缺失
	if strings.Contains(errMsg, "required") {
		detail.Message = "缺少必填字段"
		return detail
	}

	// 返回原始错误消息（作为兜底）
	return detail
}

// findFieldByValueInRequest 尝试从请求体 JSON 中查找包含指定值的字段名
//
// 该函数读取请求体，解析为 map，然后查找值匹配的字段。
// 注意：这个方法只在其他方法都失败时使用，作为最后的尝试。
//
// 参数：
//   - ctx: Gin 上下文
//   - value: 要查找的值
//
// 返回值：
//   - 字段名（JSON key），如果找不到则返回空字符串
func findFieldByValueInRequest(ctx *gin.Context, value interface{}) string {
	// 尝试从 ctx.Keys 中获取缓存的请求体
	if cachedBody, exists := ctx.Get("cached_request_body"); exists {
		if bodyBytes, ok := cachedBody.([]byte); ok {
			return findFieldInJSON(bodyBytes, value)
		}
	}

	// 如果没有缓存，尝试读取请求体（这可能会失败，因为请求体已经被读取了）
	// Gin 的 ShouldBindJSON 会读取请求体，但我们可以尝试从 Request.Body 获取
	// 注意：这通常不会成功，除非使用了缓存中间件
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil || len(bodyBytes) == 0 {
		return ""
	}

	// 重置请求体，以便后续可以再次读取（虽然可能已经太晚了）
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return findFieldInJSON(bodyBytes, value)
}

// findFieldInJSON 在 JSON 数据中查找值匹配的字段名
func findFieldInJSON(jsonData []byte, targetValue interface{}) string {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return ""
	}

	// 将目标值转换为字符串进行比较
	targetStr := fmt.Sprintf("%v", targetValue)

	for key, val := range data {
		valStr := fmt.Sprintf("%v", val)
		if valStr == targetStr {
			return key
		}
	}

	return ""
}
