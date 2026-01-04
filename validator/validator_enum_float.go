package xVaild

import (
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidateEnumFloat 验证浮点数枚举值
//
// 该函数验证字段值是否在指定的浮点数枚举列表中。
// 支持所有浮点数类型（float32、float64）以及自定义浮点数类型。
//
// Tag 格式：
//
//	enum_float=0.5 1.0 1.5 2.0   - 允许的值为 0.5, 1.0, 1.5, 2.0
//	enum_float=1.5 2.0 2.5       - 支持任意浮点数
//
// 使用示例：
//
//	type Rating float64
//	type Product struct {
//	    Rating  Rating  `binding:"enum_float=0.5 1.0 1.5 2.0 2.5 3.0" label:"评分"`
//	    Discount float32 `binding:"enum_float=0.1 0.2 0.5" label:"折扣"`
//	}
//
// 支持的类型：
//   - float32, float64
//   - 基于以上类型的自定义类型（如 type Rating float64）
//
// 注意：
//   - 必须在 binding tag 中提供至少一个有效的浮点数值
//   - 参数使用空格分隔
//   - 浮点数比较使用一定的精度容差（epsilon），避免浮点数精度问题
//   - 如果参数解析失败或字段类型不是浮点数，验证将返回 false
func ValidateEnumFloat(fl validator.FieldLevel) bool {
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

	// 将允许的值转换为 float64 切片
	allowedFloats := make([]float64, 0, len(allowedValues))
	for _, valStr := range allowedValues {
		val, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64)
		if err != nil {
			// 参数格式错误，验证失败
			return false
		}
		allowedFloats = append(allowedFloats, val)
	}

	// 获取字段值（使用反射安全地获取浮点数值）
	field := fl.Field()

	// 根据字段的 Kind 类型获取浮点数值
	var fieldValue float64
	switch field.Kind() {
	case reflect.Float32, reflect.Float64:
		fieldValue = field.Float()
	default:
		// 不支持的类型，验证失败
		return false
	}

	// 定义浮点数比较的精度容差（epsilon）
	const epsilon = 1e-9

	// 检查值是否在允许的枚举列表中（使用 epsilon 容差）
	for _, allowed := range allowedFloats {
		if math.Abs(fieldValue-allowed) < epsilon {
			return true
		}
	}

	return false
}
