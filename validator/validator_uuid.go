package xVaild

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// ValidateUUID 验证UUID格式
//
// 该函数使用正则表达式验证输入的UUID是否符合标准的UUID格式。
//
// 支持的格式：
//   - 123e4567-e89b-12d3-a456-426614000000
//
// 注意：此验证器仅检查UUID的格式是否正确，不检查UUID的实际存在性或有效性。
func ValidateUUID(fl validator.FieldLevel) bool {
	return regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(fl.Field().String())
}
