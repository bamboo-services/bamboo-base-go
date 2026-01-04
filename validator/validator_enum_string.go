package xVaild

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidateEnumString 验证字符串枚举值
//
// 该函数验证字段值是否在指定的字符串枚举列表中。
// 支持字符串类型以及基于字符串的自定义类型。
//
// Tag 格式：
//
//	enum_string=active pending inactive   - 允许的值为 "active", "pending", "inactive"
//	enum_string=admin user guest          - 支持任意字符串
//
// 使用示例：
//
//	type UserRole string
//	type User struct {
//	    Role   UserRole `binding:"enum_string=admin user guest" label:"角色"`
//	    Status string   `binding:"enum_string=active pending inactive" label:"状态"`
//	}
//
// 支持的类型：
//   - string
//   - 基于 string 的自定义类型（如 type UserRole string）
//
// 注意：
//   - 必须在 binding tag 中提供至少一个有效的字符串值
//   - 参数使用空格分隔
//   - 验证是大小写敏感的
//   - 如果参数为空或字段类型不是字符串，验证将返回 false
func ValidateEnumString(fl validator.FieldLevel) bool {
	// 获取枚举值参数
	enumParam := fl.Param()
	if enumParam == "" {
		return false
	}

	// 解析允许的枚举值列表
	allowedValues := strings.Fields(enumParam) // 使用 strings.Fields 自动处理多余空格
	if len(allowedValues) == 0 {
		return false
	}

	// 将允许的值转换为 map 集合（用于快速查找）
	allowedSet := make(map[string]bool, len(allowedValues))
	for _, val := range allowedValues {
		allowedSet[strings.TrimSpace(val)] = true
	}

	// 获取字段值
	field := fl.Field()

	// 检查字段类型是否为字符串
	if field.Kind() != reflect.String {
		return false
	}

	fieldValue := field.String()

	// 检查值是否在允许的枚举列表中
	return allowedSet[fieldValue]
}
