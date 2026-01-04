package xVaild

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// ValidateAlphanumUnderscore 验证字母数字和下划线格式
//
// 该函数使用正则表达式验证输入的字符串是否仅包含字母、数字和下划线。
//
// 支持的格式：
//   - 字母（a-z, A-Z）
//   - 数字（0-9）
//   - 下划线（_）
//
// 示例：
//   - "valid_string_123" -> true
//   - "invalid string!" -> false
//   - "anotherValid123" -> true
//   - "xiao_lfeng" -> true
//
// 注意：此验证器不允许其他特殊字符或空格，仅限字母、数字和下划线。
func ValidateAlphanumUnderscore(fl validator.FieldLevel) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(fl.Field().String())
}
