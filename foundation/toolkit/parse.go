package toolkit

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseInt 将输入值转换为 int 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。数值输入需落在 int 范围内；浮点数向零截断小数部分。
// 字符串去除首尾空白后按十进制解析，可带正负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - int: 转换后的整数值。
//   - bool: 是否转换成功（类型不支持、格式非法或超出范围时
//     返回 false，此时数值为零值）。
func ParseInt(value any) (int, bool) {
	number, ok := parseIntRange(value, math.MinInt, math.MaxInt, strconv.IntSize)
	if !ok {
		return 0, false
	}
	return int(number), true
}

// ParseInt8 将输入值转换为 int8 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。数值输入需落在 int8 范围内；浮点数向零截断小数部分。
// 字符串去除首尾空白后按十进制解析，可带正负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - int8: 转换后的整数值。
//   - bool: 是否转换成功（类型不支持、格式非法或超出范围时
//     返回 false，此时数值为零值）。
func ParseInt8(value any) (int8, bool) {
	number, ok := parseIntRange(value, math.MinInt8, math.MaxInt8, 8)
	if !ok {
		return 0, false
	}
	return int8(number), true
}

// ParseInt16 将输入值转换为 int16 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。数值输入需落在 int16 范围内；浮点数向零截断小数部分。
// 字符串去除首尾空白后按十进制解析，可带正负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - int16: 转换后的整数值。
//   - bool: 是否转换成功（类型不支持、格式非法或超出范围时
//     返回 false，此时数值为零值）。
func ParseInt16(value any) (int16, bool) {
	number, ok := parseIntRange(value, math.MinInt16, math.MaxInt16, 16)
	if !ok {
		return 0, false
	}
	return int16(number), true
}

// ParseInt32 将输入值转换为 int32 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。数值输入需落在 int32 范围内；浮点数向零截断小数部分。
// 字符串去除首尾空白后按十进制解析，可带正负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - int32: 转换后的整数值。
//   - bool: 是否转换成功（类型不支持、格式非法或超出范围时
//     返回 false，此时数值为零值）。
func ParseInt32(value any) (int32, bool) {
	number, ok := parseIntRange(value, math.MinInt32, math.MaxInt32, 32)
	if !ok {
		return 0, false
	}
	return int32(number), true
}

// ParseInt64 将输入值转换为 int64 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。数值输入需落在 int64 范围内；浮点数向零截断小数部分。
// 字符串去除首尾空白后按十进制解析，可带正负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - int64: 转换后的整数值。
//   - bool: 是否转换成功（类型不支持、格式非法或超出范围时
//     返回 false，此时数值为零值）。
func ParseInt64(value any) (int64, bool) {
	return parseIntRange(value, math.MinInt64, math.MaxInt64, 64)
}

// ParseUint 将输入值转换为 uint 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。负数或超出 uint 范围时返回失败；
// 浮点数向零截断小数部分且不得为负。
// 字符串去除首尾空白后按十进制解析，不支持负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - uint: 转换后的无符号整数值。
//   - bool: 是否转换成功（类型不支持、格式非法、为负或超出范围时
//     返回 false，此时数值为零值）。
func ParseUint(value any) (uint, bool) {
	number, ok := parseUintRange(value, math.MaxUint, strconv.IntSize)
	if !ok {
		return 0, false
	}
	return uint(number), true
}

// ParseUint8 将输入值转换为 uint8 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。负数或超出 uint8 范围时返回失败；
// 浮点数向零截断小数部分且不得为负。
// 字符串去除首尾空白后按十进制解析，不支持负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - uint8: 转换后的无符号整数值。
//   - bool: 是否转换成功（类型不支持、格式非法、为负或超出范围时
//     返回 false，此时数值为零值）。
func ParseUint8(value any) (uint8, bool) {
	number, ok := parseUintRange(value, math.MaxUint8, 8)
	if !ok {
		return 0, false
	}
	return uint8(number), true
}

// ParseUint16 将输入值转换为 uint16 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。负数或超出 uint16 范围时返回失败；
// 浮点数向零截断小数部分且不得为负。
// 字符串去除首尾空白后按十进制解析，不支持负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - uint16: 转换后的无符号整数值。
//   - bool: 是否转换成功（类型不支持、格式非法、为负或超出范围时
//     返回 false，此时数值为零值）。
func ParseUint16(value any) (uint16, bool) {
	number, ok := parseUintRange(value, math.MaxUint16, 16)
	if !ok {
		return 0, false
	}
	return uint16(number), true
}

// ParseUint32 将输入值转换为 uint32 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。负数或超出 uint32 范围时返回失败；
// 浮点数向零截断小数部分且不得为负。
// 字符串去除首尾空白后按十进制解析，不支持负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - uint32: 转换后的无符号整数值。
//   - bool: 是否转换成功（类型不支持、格式非法、为负或超出范围时
//     返回 false，此时数值为零值）。
func ParseUint32(value any) (uint32, bool) {
	number, ok := parseUintRange(value, math.MaxUint32, 32)
	if !ok {
		return 0, false
	}
	return uint32(number), true
}

// ParseUint64 将输入值转换为 uint64 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。负数或超出 uint64 范围时返回失败；
// 浮点数向零截断小数部分且不得为负。
// 字符串去除首尾空白后按十进制解析，不支持负号，
// 小数、十六进制等字面量返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - uint64: 转换后的无符号整数值。
//   - bool: 是否转换成功（类型不支持、格式非法、为负或超出范围时
//     返回 false，此时数值为零值）。
func ParseUint64(value any) (uint64, bool) {
	return parseUintRange(value, math.MaxUint64, 64)
}

// ParseFloat32 将输入值转换为 float32 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。字符串去除首尾空白后按 Go 浮点语法解析
// （支持科学计数法）。字符串按 float32 精度舍入；
// 数值超出 float32 可表示范围时返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - float32: 转换后的浮点数值。
//   - bool: 是否转换成功（类型不支持、格式非法或超出范围时
//     返回 false，此时数值为零值）。
func ParseFloat32(value any) (float32, bool) {
	number, ok := parseFloatValue(value, 32)
	if !ok {
		return 0, false
	}
	if number > math.MaxFloat32 || number < -math.MaxFloat32 {
		return 0, false
	}
	return float32(number), true
}

// ParseFloat64 将输入值转换为 float64 类型。
//
// 支持的类型包括所有整数、浮点数以及 string，其余类型（含 nil）
// 返回失败。字符串去除首尾空白后按 Go 浮点语法解析
// （支持科学计数法），整数输入无损转换为浮点数。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - float64: 转换后的浮点数值。
//   - bool: 是否转换成功（类型不支持或格式非法时返回 false，
//     此时数值为零值）。
func ParseFloat64(value any) (float64, bool) {
	return parseFloatValue(value, 64)
}

// ParseBool 将输入值转换为 bool 类型。
//
// 支持的类型包括 bool、所有整数、浮点数以及 string，
// 其余类型（含 nil）返回失败。数值类型中非 0 视为 true，
// 0 视为 false。字符串去除首尾空白且不区分大小写，
// 支持 true/false、1/0、yes/no、on/off、enabled/disabled；
// 空字符串或无法识别的取值返回失败。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - bool: 转换后的布尔值。
//   - bool: 是否转换成功（类型不支持或字符串不可识别时
//     返回 false，此时布尔值为 false）。
func ParseBool(value any) (bool, bool) {
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

// ParseString 将输入值转换为 string 类型。
//
// 支持的类型包括 string、[]byte、fmt.Stringer、error、bool 以及
// 所有整数与浮点数，其余类型（含 nil）返回失败。实现了
// fmt.Stringer 的值优先调用 String 方法；error 调用 Error 方法。
// 整数按十进制输出；浮点数采用无指数的最短十进制表示。
//
// 参数说明:
//   - value: 需要解析的输入值。
//
// 返回值:
//   - string: 转换后的字符串。
//   - bool: 是否转换成功（类型不支持时返回 false，此时字符串为空）。
func ParseString(value any) (string, bool) {
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
