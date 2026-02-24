package xVaild

import (
	"regexp"

	"github.com/go-playground/validator/v10"
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
