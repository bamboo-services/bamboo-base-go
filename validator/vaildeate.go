package xVaild

import (
	"github.com/go-playground/validator/v10"
	"regexp"
)

// ValidateURL 验证URL格式
//
// 该函数使用正则表达式验证输入的URL是否符合标准的HTTP或HTTPS格式。
// 支持的格式包括：
//   - http://example.com
//   - https://example.com
//   - http://example.com/path/to/resource
//   - https://example.com/path/to/resource
//
// 注意：此验证器不检查URL的实际可访问性或有效性，仅验证格式。
func ValidateURL(fl validator.FieldLevel) bool {
	return regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?$`).MatchString(fl.Field().String())
}

// ValidateUUID 验证UUID格式
//
// 该函数使用正则表达式验证输入的UUID是否符合标准的UUID格式。
//
// 支持的格式：
//   - 123e4567-e89b-12d3-a456-426614000000
//
// 注意：此验证器仅检查UUID的格式是否正确，不检查UUID的实际存在性或
func ValidateUUID(fl validator.FieldLevel) bool {
	return regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(fl.Field().String())
}

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
