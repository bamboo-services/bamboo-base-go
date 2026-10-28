package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
	v10 "github.com/go-playground/validator/v10"
)

// ValidationErrorDetail 单个验证错误的详情
type ValidationErrorDetail struct {
	Field   string `json:"field"`           // 字段名
	Tag     string `json:"tag"`             // 验证标签
	Message string `json:"message"`         // 错误消息
	Value   any    `json:"value,omitempty"` // 触发错误的字段值（安全默认置空避免泄漏敏感输入）
}

// handleErrorInternal 将验证或绑定错误统一安全转换为 apperr.Error 并使用 httpresult 写出。
// 严禁将底层内部 err.Err 暴露至响应，不循环引用 bind。
func (e *Engine) handleErrorInternal(ctx *gin.Context, bindErr error) {
	if ctx == nil || ctx.Writer.Written() {
		return
	}

	var (
		errorDetails      []ValidationErrorDetail
		firstErrorMessage string
	)

	var valErrs v10.ValidationErrors
	if errors.As(bindErr, &valErrs) {
		translated := e.TranslateError(bindErr)
		for i, fe := range valErrs {
			msg := translated[fe.Field()]
			if msg == "" {
				msg = formatFallbackMessage(fe)
			}
			if i == 0 {
				firstErrorMessage = msg
			}
			errorDetails = append(errorDetails, ValidationErrorDetail{
				Field:   fe.Field(),
				Tag:     fe.Tag(),
				Message: msg,
				Value:   nil,
			})
		}
	} else {
		detail := parseBindingErrorDetail(bindErr)
		firstErrorMessage = detail.Message
		errorDetails = append(errorDetails, detail)
	}

	if firstErrorMessage == "" {
		firstErrorMessage = "请求数据验证失败"
	}

	// 构造统一 apperr.Error。注意：底层根因保留在 Err 内部供日志与 unwrap 使用，
	// httpresult.Error 内部 buildEnvelope 绝不会把 Err 序列化泄漏给客户端。
	appErr := apperr.New(apperr.BodyError, apperr.ErrMessage(firstErrorMessage), bindErr)
	appErr.Data = errorDetails

	httpresult.Error(ctx, appErr)
}

func formatFallbackMessage(fe v10.FieldError) string {
	field := fe.Field()
	tag := fe.Tag()
	switch tag {
	case "required":
		return field + " 为必填项"
	case "min":
		return fmt.Sprintf("%s 长度不能少于 %s", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s 长度不能超过 %s", field, fe.Param())
	case "oneof", "enum_int", "enum_string", "enum_float":
		return fmt.Sprintf("%s 必须是以下值之一: %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s 验证失败", field)
	}
}

// parseBindingErrorDetail 解析非 ValidationErrors 的反序列化错误。
func parseBindingErrorDetail(err error) ValidationErrorDetail {
	detail := ValidationErrorDetail{
		Field:   "unknown",
		Tag:     "binding",
		Message: "请求体解析失败",
		Value:   nil,
	}
	if err == nil {
		return detail
	}

	errMsg := err.Error()

	var unmarshalTypeErr *json.UnmarshalTypeError
	curr := err
	for curr != nil {
		if errors.As(curr, &unmarshalTypeErr) {
			detail.Field = unmarshalTypeErr.Field
			break
		}
		if unwrapper, ok := curr.(interface{ Unwrap() error }); ok {
			curr = unwrapper.Unwrap()
		} else {
			break
		}
	}

	if detail.Field == "unknown" || detail.Field == "" {
		if strings.Contains(errMsg, "Go struct field") {
			if parts := strings.Split(errMsg, "Go struct field "); len(parts) > 1 {
				if fieldParts := strings.Split(parts[1], " of type"); len(fieldParts) > 0 {
					path := strings.TrimSpace(fieldParts[0])
					if dotIdx := strings.LastIndex(path, "."); dotIdx != -1 {
						detail.Field = path[dotIdx+1:]
					} else {
						detail.Field = path
					}
				}
			}
		}
	}

	if strings.Contains(errMsg, "parsing time") {
		detail.Message = "时间格式不正确，请使用标准时间格式"
		if detail.Field != "unknown" && detail.Field != "" {
			detail.Message = detail.Field + " " + detail.Message
		}
		return detail
	}

	if strings.Contains(errMsg, "invalid character") ||
		strings.Contains(errMsg, "unexpected end of JSON") ||
		strings.Contains(errMsg, "looking for beginning of value") {
		detail.Message = "JSON 格式错误，请检查请求体格式"
		return detail
	}

	if strings.Contains(errMsg, "cannot unmarshal") {
		detail.Message = "字段类型不匹配，请检查请求参数类型"
		if detail.Field != "unknown" && detail.Field != "" {
			detail.Message = detail.Field + " " + detail.Message
		}
		return detail
	}

	if strings.Contains(errMsg, "required") {
		detail.Message = "缺少必填字段"
		return detail
	}

	detail.Message = "请求体解析失败"
	return detail
}
