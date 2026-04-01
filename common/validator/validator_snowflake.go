package xVaild

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

// ValidateSnowflake 验证字符串是否为合法的 Snowflake ID
//
// 该函数验证字段值是否为合法的 Snowflake ID 格式（纯数字、长度 1~20）。
// Snowflake ID 是 Twitter 推出的分布式 ID 生成方案，在 bamboo-base-go 中
// 通过 snowflake 包实现。
//
// Tag 格式：
//
//	binding:"snowflake"
//
// 使用示例：
//
//	type CreateRequest struct {
//	    ParentID string `binding:"omitempty,snowflake" label:"父分类ID"`
//	}
//
// 支持的类型：
//   - string
//   - 基于 string 的自定义类型（如 type SnowflakeID string）
//
// 注意：
//   - 空字符串被视为无效（使用 omitempty 来允许空值）
//   - 长度超过 20 位的数字被视为无效（Snowflake ID 最大为 64 位整数）
//   - 仅支持纯数字（0-9）
func ValidateSnowflake(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	value := field.String()
	if value == "" {
		return false
	}
	// Snowflake ID 为纯数字，长度通常在 1~20 位之间
	// 64 位整数的最大值是 9223372036854775807（19 位），留出余量到 20 位
	if len(value) > 20 {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
