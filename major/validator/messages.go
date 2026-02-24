package xVaild

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
)

// ValidationErrorMessages 定义验证错误消息的映射
//
// 该映射包含了常用验证规则的中文错误消息，用于在翻译器初始化失败时提供 fallback。
// 对于带参数的验证规则（如 min、max、oneof），消息中可以使用 %s 占位符。
//
// 注意：
//   - 优先使用翻译器（translator）提供的错误消息
//   - 此映射仅在翻译器不可用时作为后备方案
//   - 建议在 translator.go 中注册新的验证规则翻译，而不是仅依赖此映射
var ValidationErrorMessages = map[string]string{
	// 基础验证规则
	"required": "为必填项",
	"min":      "长度不能少于 %s 个字符",
	"max":      "长度不能超过 %s 个字符",

	// 格式验证规则
	"url":      "必须是有效的URL",
	"uuid":     "必须是有效的UUID格式",
	"email":    "必须是有效的邮箱地址",
	"alphanum": "只能包含字母和数字",
	"alpha":    "只能包含字母",
	"numeric":  "只能包含数字",

	// 自定义验证规则
	"strict_url":          "必须是有效的 HTTP 或 HTTPS URL",
	"strict_uuid":         "必须是标准的 UUID 格式",
	"alphanum_underscore": "只能包含字母、数字和下划线",
	"regexp":              "格式不正确",
	"enum_int":            "必须是以下值之一: %s",
	"enum_string":         "必须是以下值之一: %s",
	"enum_float":          "必须是以下值之一: %s",

	// 条件验证规则
	"oneof": "必须是以下值之一: %s",
}

// GetValidationErrorMessage 获取验证错误的中文消息
func GetValidationErrorMessage(fe validator.FieldError) string {
	tag := fe.Tag()
	field := fe.Field()

	if msg, exists := ValidationErrorMessages[tag]; exists {
		switch tag {
		case "min", "max", "oneof", "enum_int", "enum_string", "enum_float":
			return fmt.Sprintf("%s%s", field, fmt.Sprintf(msg, fe.Param()))
		default:
			return fmt.Sprintf("%s %s", field, msg)
		}
	}

	// 默认错误消息
	return fmt.Sprintf("%s 验证失败", field)
}

// FormatValidationErrors 格式化验证错误为用户友好的消息
func FormatValidationErrors(err error) []string {
	var messages []string

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fe := range validationErrors {
			messages = append(messages, GetValidationErrorMessage(fe))
		}
	}

	return messages
}
