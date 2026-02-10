package xUtil

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type parse struct{}

// Parse 返回解析工具实例。
//
// 可用于链式调用解析方法，例如：
//
//	xUtil.Parse().Int("123")
//
// 返回值:
//   - *parse: 解析工具实例
func Parse() *parse {
	return &parse{}
}

// Int 将输入值转换为 int 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - int: 转换后的整数值
//   - bool: 是否转换成功（类型不支持或超出范围时返回 false）
func (parse) Int(value any) (int, bool) {
	number, ok := parseIntRange(value, math.MinInt, math.MaxInt, strconv.IntSize)
	if !ok {
		return 0, false
	}
	return int(number), true
}

// Int8 将输入值转换为 int8 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - int8: 转换后的整数值
//   - bool: 是否转换成功（类型不支持或超出范围时返回 false）
func (parse) Int8(value any) (int8, bool) {
	number, ok := parseIntRange(value, math.MinInt8, math.MaxInt8, 8)
	if !ok {
		return 0, false
	}
	return int8(number), true
}

// Int16 将输入值转换为 int16 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - int16: 转换后的整数值
//   - bool: 是否转换成功（类型不支持或超出范围时返回 false）
func (parse) Int16(value any) (int16, bool) {
	number, ok := parseIntRange(value, math.MinInt16, math.MaxInt16, 16)
	if !ok {
		return 0, false
	}
	return int16(number), true
}

// Int32 将输入值转换为 int32 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - int32: 转换后的整数值
//   - bool: 是否转换成功（类型不支持或超出范围时返回 false）
func (parse) Int32(value any) (int32, bool) {
	number, ok := parseIntRange(value, math.MinInt32, math.MaxInt32, 32)
	if !ok {
		return 0, false
	}
	return int32(number), true
}

// Int64 将输入值转换为 int64 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - int64: 转换后的整数值
//   - bool: 是否转换成功（类型不支持或超出范围时返回 false）
func (parse) Int64(value any) (int64, bool) {
	return parseIntRange(value, math.MinInt64, math.MaxInt64, 64)
}

// Uint 将输入值转换为 uint 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 负数或超出范围时会返回失败。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - uint: 转换后的无符号整数值
//   - bool: 是否转换成功（类型不支持、负数或超出范围时返回 false）
func (parse) Uint(value any) (uint, bool) {
	number, ok := parseUintRange(value, math.MaxUint, strconv.IntSize)
	if !ok {
		return 0, false
	}
	return uint(number), true
}

// Uint8 将输入值转换为 uint8 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 负数或超出范围时会返回失败。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - uint8: 转换后的无符号整数值
//   - bool: 是否转换成功（类型不支持、负数或超出范围时返回 false）
func (parse) Uint8(value any) (uint8, bool) {
	number, ok := parseUintRange(value, math.MaxUint8, 8)
	if !ok {
		return 0, false
	}
	return uint8(number), true
}

// Uint16 将输入值转换为 uint16 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 负数或超出范围时会返回失败。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - uint16: 转换后的无符号整数值
//   - bool: 是否转换成功（类型不支持、负数或超出范围时返回 false）
func (parse) Uint16(value any) (uint16, bool) {
	number, ok := parseUintRange(value, math.MaxUint16, 16)
	if !ok {
		return 0, false
	}
	return uint16(number), true
}

// Uint32 将输入值转换为 uint32 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 负数或超出范围时会返回失败。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - uint32: 转换后的无符号整数值
//   - bool: 是否转换成功（类型不支持、负数或超出范围时返回 false）
func (parse) Uint32(value any) (uint32, bool) {
	number, ok := parseUintRange(value, math.MaxUint32, 32)
	if !ok {
		return 0, false
	}
	return uint32(number), true
}

// Uint64 将输入值转换为 uint64 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 负数或超出范围时会返回失败。
// 浮点数会按 Go 默认类型转换规则截断小数部分。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - uint64: 转换后的无符号整数值
//   - bool: 是否转换成功（类型不支持、负数或超出范围时返回 false）
func (parse) Uint64(value any) (uint64, bool) {
	return parseUintRange(value, math.MaxUint64, 64)
}

// Float32 将输入值转换为 float32 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
// 当数值超出 float32 可表示范围时返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - float32: 转换后的浮点数值
//   - bool: 是否转换成功（类型不支持或超出范围时返回 false）
func (parse) Float32(value any) (float32, bool) {
	number, ok := parseFloatValue(value, 32)
	if !ok {
		return 0, false
	}
	if number > math.MaxFloat32 || number < -math.MaxFloat32 {
		return 0, false
	}
	return float32(number), true
}

// Float64 将输入值转换为 float64 类型。
//
// 支持的类型包括所有整数、浮点数以及 string。
// 字符串会自动去除首尾空格，并按十进制解析。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - float64: 转换后的浮点数值
//   - bool: 是否转换成功（类型不支持或解析失败时返回 false）
func (parse) Float64(value any) (float64, bool) {
	return parseFloatValue(value, 64)
}

// Bool 将输入值转换为 bool 类型。
//
// 支持的类型包括 bool、所有整数、浮点数以及 string。
// 数值类型中非 0 视为 true；0 视为 false。
// 字符串支持 true/false、1/0、yes/no、on/off、enabled/disabled（不区分大小写）。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - bool: 转换后的布尔值
//   - bool: 是否转换成功（类型不支持或字符串不可识别时返回 false）
func (parse) Bool(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case int:
		return v != 0, true
	case int8:
		return v != 0, true
	case int16:
		return v != 0, true
	case int32:
		return v != 0, true
	case int64:
		return v != 0, true
	case uint:
		return v != 0, true
	case uint8:
		return v != 0, true
	case uint16:
		return v != 0, true
	case uint32:
		return v != 0, true
	case uint64:
		return v != 0, true
	case float32:
		return v != 0, true
	case float64:
		return v != 0, true
	case string:
		trimmed := strings.ToLower(strings.TrimSpace(v))
		if trimmed == "" {
			return false, false
		}
		switch trimmed {
		case "true", "1", "yes", "on", "enabled":
			return true, true
		case "false", "0", "no", "off", "disabled":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

// String 将输入值转换为 string 类型。
//
// 支持的类型包括 string、[]byte、fmt.Stringer、error 以及基础数值类型。
// 数值类型按十进制格式输出，浮点数采用 Go 默认格式化规则。
//
// 参数说明:
//   - value: 需要解析的输入值
//
// 返回值:
//   - string: 转换后的字符串
//   - bool: 是否转换成功（类型不支持时返回 false）
func (parse) String(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	case fmt.Stringer:
		return v.String(), true
	case error:
		return v.Error(), true
	case bool:
		return strconv.FormatBool(v), true
	case int:
		return strconv.FormatInt(int64(v), 10), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	default:
		return "", false
	}
}

// parseIntRange 在指定范围内解析为 int64。
//
// bitSize 用于字符串解析时的位数控制，min/max 用于范围校验。
func parseIntRange(value any, min int64, max int64, bitSize int) (int64, bool) {
	switch v := value.(type) {
	case int:
		number := int64(v)
		if number < min || number > max {
			return 0, false
		}
		return number, true
	case int8:
		number := int64(v)
		if number < min || number > max {
			return 0, false
		}
		return number, true
	case int16:
		number := int64(v)
		if number < min || number > max {
			return 0, false
		}
		return number, true
	case int32:
		number := int64(v)
		if number < min || number > max {
			return 0, false
		}
		return number, true
	case int64:
		if v < min || v > max {
			return 0, false
		}
		return v, true
	case uint:
		number := uint64(v)
		if number > uint64(max) {
			return 0, false
		}
		return int64(number), true
	case uint8:
		number := uint64(v)
		if number > uint64(max) {
			return 0, false
		}
		return int64(number), true
	case uint16:
		number := uint64(v)
		if number > uint64(max) {
			return 0, false
		}
		return int64(number), true
	case uint32:
		number := uint64(v)
		if number > uint64(max) {
			return 0, false
		}
		return int64(number), true
	case uint64:
		if v > uint64(max) {
			return 0, false
		}
		return int64(v), true
	case float32:
		number := float64(v)
		if number < float64(min) || number > float64(max) {
			return 0, false
		}
		return int64(number), true
	case float64:
		if v < float64(min) || v > float64(max) {
			return 0, false
		}
		return int64(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		number, err := strconv.ParseInt(trimmed, 10, bitSize)
		if err != nil {
			return 0, false
		}
		return number, true
	default:
		return 0, false
	}
}

// parseUintRange 在指定范围内解析为 uint64。
//
// bitSize 用于字符串解析时的位数控制，max 用于范围校验。
func parseUintRange(value any, max uint64, bitSize int) (uint64, bool) {
	switch v := value.(type) {
	case uint:
		number := uint64(v)
		if number > max {
			return 0, false
		}
		return number, true
	case uint8:
		number := uint64(v)
		if number > max {
			return 0, false
		}
		return number, true
	case uint16:
		number := uint64(v)
		if number > max {
			return 0, false
		}
		return number, true
	case uint32:
		number := uint64(v)
		if number > max {
			return 0, false
		}
		return number, true
	case uint64:
		if v > max {
			return 0, false
		}
		return v, true
	case int:
		if v < 0 {
			return 0, false
		}
		number := uint64(v)
		if number > max {
			return 0, false
		}
		return number, true
	case int8:
		if v < 0 {
			return 0, false
		}
		number := uint64(v)
		if number > max {
			return 0, false
		}
		return number, true
	case int16:
		if v < 0 {
			return 0, false
		}
		number := uint64(v)
		if number > max {
			return 0, false
		}
		return number, true
	case int32:
		if v < 0 {
			return 0, false
		}
		number := uint64(v)
		if number > max {
			return 0, false
		}
		return number, true
	case int64:
		if v < 0 {
			return 0, false
		}
		number := uint64(v)
		if number > max {
			return 0, false
		}
		return number, true
	case float32:
		number := float64(v)
		if number < 0 || number > float64(max) {
			return 0, false
		}
		return uint64(number), true
	case float64:
		if v < 0 || v > float64(max) {
			return 0, false
		}
		return uint64(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		number, err := strconv.ParseUint(trimmed, 10, bitSize)
		if err != nil {
			return 0, false
		}
		return number, true
	default:
		return 0, false
	}
}

// parseFloatValue 解析为 float64。
//
// bitSize 仅用于字符串解析时的精度控制。
func parseFloatValue(value any, bitSize int) (float64, bool) {
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		number, err := strconv.ParseFloat(trimmed, bitSize)
		if err != nil {
			return 0, false
		}
		return number, true
	default:
		return 0, false
	}
}
