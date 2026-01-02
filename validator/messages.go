package xVaild

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
)

// ValidationErrorMessages 定义验证错误消息的映射
var ValidationErrorMessages = map[string]string{
	"required":    "为必填项",
	"min":         "长度不能少于 %s 个字符",
	"max":         "长度不能超过 %s 个字符",
	"url":         "必须是有效的URL",
	"uuid":        "必须是有效的UUID格式",
	"strict_url":  "必须是有效的 HTTP 或 HTTPS URL",
	"strict_uuid": "必须是标准的 UUID 格式",
	"oneof":       "必须是以下值之一: %s",
}

// GetValidationErrorMessage 获取验证错误的中文消息
func GetValidationErrorMessage(fe validator.FieldError) string {
	tag := fe.Tag()
	field := fe.Field()

	if msg, exists := ValidationErrorMessages[tag]; exists {
		switch tag {
		case "min", "max", "oneof":
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
