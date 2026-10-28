package typed

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Int 将原始文本去除首尾空白后按十进制解析为 Go 原生 int。
// bitSize 传 0 以遵循当前平台的位宽（32 位或 64 位），超出范围返回溢出错误。
func (s String) Int() (int, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(string(s)), 10, 0)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

// Int8 将原始文本去除首尾空白后按十进制严格解析为 Go 原生 int8。
// 超出 [-128, 127] 范围返回溢出错误。
func (s String) Int8() (int8, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(string(s)), 10, 8)
	if err != nil {
		return 0, err
	}
	return int8(v), nil
}

// Int16 将原始文本去除首尾空白后按十进制严格解析为 Go 原生 int16。
// 超出 [-32768, 32767] 范围返回溢出错误。
func (s String) Int16() (int16, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(string(s)), 10, 16)
	if err != nil {
		return 0, err
	}
	return int16(v), nil
}

// Int32 将原始文本去除首尾空白后按十进制严格解析为 Go 原生 int32。
// 超出 [-2147483648, 2147483647] 范围返回溢出错误。
func (s String) Int32() (int32, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(string(s)), 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

// Int64 将原始文本去除首尾空白后按十进制严格解析为 Go 原生 int64。
// 超出 int64 范围返回溢出错误。
func (s String) Int64() (int64, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(string(s)), 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// Uint 将原始文本去除首尾空白后按十进制解析为 Go 原生 uint。
// bitSize 传 0 以遵循当前平台的位宽（32 位或 64 位），绝不先按 uint64 解析再截断。
func (s String) Uint() (uint, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(string(s)), 10, 0)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}

// Uint8 将原始文本去除首尾空白后按十进制严格解析为 Go 原生 uint8。
// 超出 [0, 255] 范围返回溢出错误。
func (s String) Uint8() (uint8, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(string(s)), 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(v), nil
}

// Uint16 将原始文本去除首尾空白后按十进制严格解析为 Go 原生 uint16。
// 超出 [0, 65535] 范围返回溢出错误。
func (s String) Uint16() (uint16, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(string(s)), 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}

// Uint32 将原始文本去除首尾空白后按十进制严格解析为 Go 原生 uint32。
// 超出 uint32 范围返回溢出错误。
func (s String) Uint32() (uint32, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(string(s)), 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// Uint64 将原始文本去除首尾空白后按十进制严格解析为 Go 原生 uint64。
// 超出 uint64 范围返回溢出错误。
func (s String) Uint64() (uint64, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(string(s)), 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// Float32 将原始文本去除首尾空白后解析为 Go 原生 float32。
// 严格拒绝 NaN 与 ±Inf（含 Infinity）。
func (s String) Float32() (float32, error) {
	trimmed := strings.TrimSpace(string(s))
	v, err := strconv.ParseFloat(trimmed, 32)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("typed.String: float value %q is not finite", trimmed)
	}
	return float32(v), nil
}

// Float64 将原始文本去除首尾空白后解析为 Go 原生 float64。
// 严格拒绝 NaN 与 ±Inf（含 Infinity）。
func (s String) Float64() (float64, error) {
	trimmed := strings.TrimSpace(string(s))
	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("typed.String: float value %q is not finite", trimmed)
	}
	return v, nil
}

// Bool 将原始文本去除首尾空白并忽略大小写后解析为 Go 原生 bool。
// 真值仅接受："true"、"1"、"yes"、"on"、"enabled"；
// 假值仅接受："false"、"0"、"no"、"off"、"disabled"；
// 其余输入一律返回错误。
func (s String) Bool() (bool, error) {
	trimmed := strings.ToLower(strings.TrimSpace(string(s)))
	switch trimmed {
	case "true", "1", "yes", "on", "enabled":
		return true, nil
	case "false", "0", "no", "off", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("typed.String: invalid boolean value %q", s)
	}
}

// Duration 将原始文本去除首尾空白后委托 time.ParseDuration 解析为 time.Duration。
// 支持 "30s"、"500ms"、"1h30m" 等标准时长写法。
func (s String) Duration() (time.Duration, error) {
	return time.ParseDuration(strings.TrimSpace(string(s)))
}
