package xVaild

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators 注册自定义验证器
//
// 该函数将所有自定义验证器注册到 validator 引擎中。
// 如果任何验证器注册失败，将返回错误。
//
// 已注册的验证器：
//   - strict_url: 严格的 URL 验证（仅支持 HTTP/HTTPS）
//   - strict_uuid: 严格的 UUID 格式验证
//   - alphanum_underscore: 字母数字下划线验证
//   - regexp: 正则表达式验证（支持自定义正则）
//   - enum_int: 整数枚举值验证（支持所有整数类型及自定义数值类型）
//   - enum_string: 字符串枚举值验证（支持字符串及自定义字符串类型）
//   - enum_float: 浮点数枚举值验证（支持浮点数及自定义浮点数类型）
//
// 使用示例：
//
//	validate := validator.New()
//	if err := RegisterCustomValidators(validate); err != nil {
//	    log.Fatal("验证器注册失败:", err)
//	}
//
// 返回值：
//   - error: 如果注册失败，返回错误信息；成功则返回 nil
func RegisterCustomValidators(validate *validator.Validate) error {
	// 定义所有自定义验证器
	validators := map[string]validator.Func{
		"strict_url":          ValidateURL,
		"strict_uuid":         ValidateUUID,
		"alphanum_underscore": ValidateAlphanumUnderscore,
		"regexp":              ValidateRegexp,
		"enum_int":            ValidateEnumInt,
		"enum_string":         ValidateEnumString,
		"enum_float":          ValidateEnumFloat,
	}

	// 注册所有验证器
	for name, fn := range validators {
		if err := validate.RegisterValidation(name, fn); err != nil {
			return fmt.Errorf("注册 %s 验证器失败: %w", name, err)
		}
	}

	return nil
}
