package toolkit

import (
	"math"
	"testing"
)

// TestParseInt_ValidInput 测试 ParseInt 函数处理各种有效输入
func TestParseInt_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
		ok    bool
	}{
		// int 类型
		{"int positive", int(42), 42, true},
		{"int negative", int(-42), -42, true},
		{"int zero", int(0), 0, true},
		// int8 类型
		{"int8", int8(42), 42, true},
		// int16 类型
		{"int16", int16(42), 42, true},
		// int32 类型
		{"int32", int32(42), 42, true},
		// int64 类型
		{"int64", int64(42), 42, true},
		// uint 类型
		{"uint", uint(42), 42, true},
		// uint8 类型
		{"uint8", uint8(42), 42, true},
		// uint16 类型
		{"uint16", uint16(42), 42, true},
		// uint32 类型
		{"uint32", uint32(42), 42, true},
		// uint64 类型
		{"uint64", uint64(42), 42, true},
		// float32 类型
		{"float32", float32(42.9), 42, true},
		// float64 类型
		{"float64", float64(42.9), 42, true},
		// string 类型
		{"string positive", "42", 42, true},
		{"string negative", "-42", -42, true},
		{"string with spaces", " 42 ", 42, true},
		{"string zero", "0", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseInt(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseInt() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseInt_InvalidInput 测试 ParseInt 函数处理无效输入
func TestParseInt_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"empty string", ""},
		{"invalid string", "abc"},
		{"float string", "42.5"},
		{"bool", true},
		{"nil", nil},
		{"struct", struct{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseInt(tt.input)
			if ok {
				t.Errorf("ParseInt() should fail for %v", tt.input)
			}
		})
	}
}

// TestParseInt8_ValidInput 测试 ParseInt8 函数处理各种有效输入
func TestParseInt8_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int8
		ok    bool
	}{
		{"int", int(42), 42, true},
		{"int8 max", int8(127), 127, true},
		{"int8 min", int8(-128), -128, true},
		{"string", "42", 42, true},
		{"float32", float32(42.9), 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseInt8(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseInt8() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseInt8() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseInt8_OutOfRange 测试 ParseInt8 函数处理超出范围的值
func TestParseInt8_OutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"overflow", int(128)},
		{"underflow", int(-129)},
		{"uint64 overflow", uint64(128)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseInt8(tt.input)
			if ok {
				t.Errorf("ParseInt8() should fail for out of range value %v", tt.input)
			}
		})
	}
}

// TestParseInt16_ValidInput 测试 ParseInt16 函数处理各种有效输入
func TestParseInt16_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int16
		ok    bool
	}{
		{"int", int(42), 42, true},
		{"int16 max", int16(32767), 32767, true},
		{"int16 min", int16(-32768), -32768, true},
		{"string", "42", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseInt16(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseInt16() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseInt16() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseInt16_OutOfRange 测试 ParseInt16 函数处理超出范围的值
func TestParseInt16_OutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"overflow", int(32768)},
		{"underflow", int(-32769)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseInt16(tt.input)
			if ok {
				t.Errorf("ParseInt16() should fail for out of range value %v", tt.input)
			}
		})
	}
}

// TestParseInt32_ValidInput 测试 ParseInt32 函数处理各种有效输入
func TestParseInt32_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int32
		ok    bool
	}{
		{"int", int(42), 42, true},
		{"int32 max", int32(2147483647), 2147483647, true},
		{"int32 min", int32(-2147483648), -2147483648, true},
		{"string", "42", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseInt32(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseInt32() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseInt32() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseInt64_ValidInput 测试 ParseInt64 函数处理各种有效输入
func TestParseInt64_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int64
		ok    bool
	}{
		{"int", int(42), 42, true},
		{"int64", int64(9223372036854775807), 9223372036854775807, true},
		{"int64 negative", int64(-9223372036854775808), -9223372036854775808, true},
		{"string", "42", 42, true},
		{"string large", "9223372036854775807", 9223372036854775807, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseInt64(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseInt64() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseUint_ValidInput 测试 ParseUint 函数处理各种有效输入
func TestParseUint_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  uint
		ok    bool
	}{
		{"int positive", int(42), 42, true},
		{"uint", uint(42), 42, true},
		{"uint8", uint8(42), 42, true},
		{"uint16", uint16(42), 42, true},
		{"uint32", uint32(42), 42, true},
		{"uint64", uint64(42), 42, true},
		{"string", "42", 42, true},
		{"float64", float64(42.9), 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseUint(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseUint() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseUint() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseUint_InvalidInput 测试 ParseUint 函数处理无效输入
func TestParseUint_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"negative int", int(-1)},
		{"negative int64", int64(-1)},
		{"negative string", "-42"},
		{"empty string", ""},
		{"invalid string", "abc"},
		{"bool", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseUint(tt.input)
			if ok {
				t.Errorf("ParseUint() should fail for %v", tt.input)
			}
		})
	}
}

// TestParseUint8_ValidInput 测试 ParseUint8 函数处理各种有效输入
func TestParseUint8_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  uint8
		ok    bool
	}{
		{"int", int(42), 42, true},
		{"uint8 max", uint8(255), 255, true},
		{"uint8 zero", uint8(0), 0, true},
		{"string", "42", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseUint8(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseUint8() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseUint8() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseUint8_OutOfRange 测试 ParseUint8 函数处理超出范围的值
func TestParseUint8_OutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"overflow", int(256)},
		{"negative", int(-1)},
		{"uint64 overflow", uint64(256)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseUint8(tt.input)
			if ok {
				t.Errorf("ParseUint8() should fail for out of range value %v", tt.input)
			}
		})
	}
}

// TestParseUint16_ValidInput 测试 ParseUint16 函数处理各种有效输入
func TestParseUint16_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  uint16
		ok    bool
	}{
		{"int", int(42), 42, true},
		{"uint16 max", uint16(65535), 65535, true},
		{"string", "42", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseUint16(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseUint16() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseUint16() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseUint32_ValidInput 测试 ParseUint32 函数处理各种有效输入
func TestParseUint32_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  uint32
		ok    bool
	}{
		{"int", int(42), 42, true},
		{"uint32 max", uint32(4294967295), 4294967295, true},
		{"string", "42", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseUint32(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseUint32() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseUint32() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseUint64_ValidInput 测试 ParseUint64 函数处理各种有效输入
func TestParseUint64_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  uint64
		ok    bool
	}{
		{"int", int(42), 42, true},
		{"uint64", uint64(18446744073709551615), 18446744073709551615, true},
		{"string", "42", 42, true},
		{"string max", "18446744073709551615", 18446744073709551615, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseUint64(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseUint64() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseUint64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseFloat32_ValidInput 测试 ParseFloat32 函数处理各种有效输入
func TestParseFloat32_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  float32
		ok    bool
	}{
		{"float32", float32(42.5), 42.5, true},
		{"float64", float64(42.5), 42.5, true},
		{"int", int(42), 42, true},
		{"int64", int64(42), 42, true},
		{"string", "42.5", 42.5, true},
		{"string with spaces", " 42.5 ", 42.5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseFloat32(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseFloat32() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseFloat32() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseFloat32_OutOfRange 测试 ParseFloat32 函数处理超出范围的值
func TestParseFloat32_OutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"overflow string", "3.4e39"},
		{"overflow float64", float64(math.MaxFloat64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseFloat32(tt.input)
			if ok {
				t.Errorf("ParseFloat32() should fail for out of range value %v", tt.input)
			}
		})
	}
}

// TestParseFloat64_ValidInput 测试 ParseFloat64 函数处理各种有效输入
func TestParseFloat64_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  float64
		ok    bool
	}{
		{"float64", float64(42.5), 42.5, true},
		{"float32", float32(42.5), 42.5, true},
		{"int", int(42), 42, true},
		{"int64", int64(42), 42, true},
		{"string", "42.5", 42.5, true},
		{"negative string", "-42.5", -42.5, true},
		{"scientific notation", "1.5e10", 1.5e10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseFloat64() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseFloat64_InvalidInput 测试 ParseFloat64 函数处理无效输入
func TestParseFloat64_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"empty string", ""},
		{"invalid string", "abc"},
		{"bool", true},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseFloat64(tt.input)
			if ok {
				t.Errorf("ParseFloat64() should fail for %v", tt.input)
			}
		})
	}
}

// TestParseBool_ValidInput 测试 ParseBool 函数处理各种有效输入
func TestParseBool_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  bool
		ok    bool
	}{
		// bool 类型
		{"bool true", true, true, true},
		{"bool false", false, false, true},
		// 整数类型（非零为 true）
		{"int positive", int(1), true, true},
		{"int zero", int(0), false, true},
		{"int negative", int(-1), true, true},
		{"int64", int64(1), true, true},
		{"uint", uint(1), true, true},
		// 浮点类型
		{"float64 positive", float64(1.5), true, true},
		{"float64 zero", float64(0), false, true},
		// 字符串类型 - true
		{"string true", "true", true, true},
		{"string TRUE", "TRUE", true, true},
		{"string 1", "1", true, true},
		{"string yes", "yes", true, true},
		{"string YES", "YES", true, true},
		{"string on", "on", true, true},
		{"string enabled", "enabled", true, true},
		// 字符串类型 - false
		{"string false", "false", false, true},
		{"string FALSE", "FALSE", false, true},
		{"string 0", "0", false, true},
		{"string no", "no", false, true},
		{"string off", "off", false, true},
		{"string disabled", "disabled", false, true},
		// 带空格的字符串
		{"string with spaces", " true ", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseBool(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseBool() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseBool_InvalidInput 测试 ParseBool 函数处理无效输入
func TestParseBool_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"empty string", ""},
		{"invalid string", "maybe"},
		{"struct", struct{}{}},
		{"nil", nil},
		{"slice", []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseBool(tt.input)
			if ok {
				t.Errorf("ParseBool() should fail for %v", tt.input)
			}
		})
	}
}

// TestParseString_ValidInput 测试 ParseString 函数处理各种有效输入
func TestParseString_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
		ok    bool
	}{
		{"string", "hello", "hello", true},
		{"empty string", "", "", true},
		{"[]byte", []byte("hello"), "hello", true},
		{"bool true", true, "true", true},
		{"bool false", false, "false", true},
		{"int", int(42), "42", true},
		{"int negative", int(-42), "-42", true},
		{"int8", int8(42), "42", true},
		{"int16", int16(42), "42", true},
		{"int32", int32(42), "42", true},
		{"int64", int64(42), "42", true},
		{"uint", uint(42), "42", true},
		{"uint8", uint8(42), "42", true},
		{"uint16", uint16(42), "42", true},
		{"uint32", uint32(42), "42", true},
		{"uint64", uint64(42), "42", true},
		{"float32", float32(42.5), "42.5", true},
		{"float64", float64(42.5), "42.5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseString(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseString() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseString() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseString_FmtStringer 测试 ParseString 函数处理 fmt.Stringer 接口
func TestParseString_FmtStringer(t *testing.T) {
	now := parseTestTime
	got, ok := ParseString(now)
	if !ok {
		t.Error("ParseString() should succeed for fmt.Stringer")
	}
	if got == "" {
		t.Error("ParseString() should return non-empty string for time.Time")
	}
}

// TestParseString_Error 测试 ParseString 函数处理 error 接口
func TestParseString_Error(t *testing.T) {
	err := testError{msg: "test error"}
	got, ok := ParseString(err)
	if !ok {
		t.Error("ParseString() should succeed for error")
	}
	if got != "test error" {
		t.Errorf("ParseString() = %q, want %q", got, "test error")
	}
}

// TestParseString_InvalidInput 测试 ParseString 函数处理无效输入
func TestParseString_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"nil", nil},
		{"struct", struct{}{}},
		{"slice", []int{1, 2, 3}},
		{"map", map[string]int{"a": 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseString(tt.input)
			if ok {
				t.Errorf("ParseString() should fail for %v", tt.input)
			}
		})
	}
}

// 测试辅助类型
type testError struct {
	msg string
}

func (e testError) Error() string {
	return e.msg
}

// 用于测试的固定时间
var parseTestTime = parseTime()

func parseTime() (t interface{ String() string }) {
	return &stringerMock{s: "2024-01-01 00:00:00"}
}

type stringerMock struct {
	s string
}

func (m *stringerMock) String() string {
	return m.s
}
