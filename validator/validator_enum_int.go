package xVaild

import (
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidateEnumInt 验证整数枚举值
//
// 该函数验证字段值是否在指定的整数枚举列表中。
// 支持所有整数类型（int、int8、int16、int32、int64、uint 等）以及自定义数值类型。
//
// Tag 格式：
//
//	enum_int=0 1 2         - 允许的值为 0, 1, 2
//	enum_int=-1 0 1 2      - 允许负数
//	enum_int=100 200 300   - 支持任意整数
//
// 使用示例：
//
//	type UserGender int8
//	type User struct {
//	    Gender UserGender `binding:"enum_int=0 1 2" label:"性别"`
//	    Status int        `binding:"enum_int=-1 0 1" label:"状态"`
//	}
//
// 支持的类型：
//   - int, int8, int16, int32, int64
//   - uint, uint8, uint16, uint32, uint64
//   - 基于以上类型的自定义类型（如 type UserGender int8）
//
// 注意：
//   - 必须在 binding tag 中提供至少一个有效的整数值
//   - 参数使用空格分隔
//   - 如果参数解析失败或字段类型不是整数，验证将返回 false
func ValidateEnumInt(fl validator.FieldLevel) bool {
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

	// 将允许的值转换为 int64 集合（用于快速查找）
	allowedSet := make(map[int64]bool, len(allowedValues))
	for _, valStr := range allowedValues {
		val, err := strconv.ParseInt(strings.TrimSpace(valStr), 10, 64)
		if err != nil {
			// 参数格式错误，验证失败
			return false
		}
		allowedSet[val] = true
	}

	// 获取字段值（使用反射安全地获取整数值）
	field := fl.Field()

	// 根据字段的 Kind 类型获取整数值
	var fieldValue int64
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue = field.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// 对于 uint 类型，需要检查是否在 int64 范围内
		uintVal := field.Uint()
		if uintVal > math.MaxInt64 {
			// 超出 int64 范围，验证失败
			return false
		}
		fieldValue = int64(uintVal)
	default:
		// 不支持的类型，验证失败
		return false
	}

	// 检查值是否在允许的枚举列表中
	return allowedSet[fieldValue]
}
