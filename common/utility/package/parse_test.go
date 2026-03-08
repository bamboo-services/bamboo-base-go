package pack

import (
	"math"
	"testing"
)

// TestParse_Int_ValidInput 测试 Int 方法处理各种有效输入
func TestParse_Int_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Int(tt.input)
			if ok != tt.ok {
				t.Errorf("Int() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Int() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Int_InvalidInput 测试 Int 方法处理无效输入
func TestParse_Int_InvalidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := p.Int(tt.input)
			if ok {
				t.Errorf("Int() should fail for %v", tt.input)
			}
		})
	}
}

// TestParse_Int8_ValidInput 测试 Int8 方法处理各种有效输入
func TestParse_Int8_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Int8(tt.input)
			if ok != tt.ok {
				t.Errorf("Int8() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Int8() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Int8_OutOfRange 测试 Int8 方法处理超出范围的值
func TestParse_Int8_OutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"overflow", int(128)},
		{"underflow", int(-129)},
		{"uint64 overflow", uint64(128)},
	}

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := p.Int8(tt.input)
			if ok {
				t.Errorf("Int8() should fail for out of range value %v", tt.input)
			}
		})
	}
}

// TestParse_Int16_ValidInput 测试 Int16 方法处理各种有效输入
func TestParse_Int16_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Int16(tt.input)
			if ok != tt.ok {
				t.Errorf("Int16() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Int16() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Int16_OutOfRange 测试 Int16 方法处理超出范围的值
func TestParse_Int16_OutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"overflow", int(32768)},
		{"underflow", int(-32769)},
	}

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := p.Int16(tt.input)
			if ok {
				t.Errorf("Int16() should fail for out of range value %v", tt.input)
			}
		})
	}
}

// TestParse_Int32_ValidInput 测试 Int32 方法处理各种有效输入
func TestParse_Int32_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Int32(tt.input)
			if ok != tt.ok {
				t.Errorf("Int32() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Int32() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Int64_ValidInput 测试 Int64 方法处理各种有效输入
func TestParse_Int64_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Int64(tt.input)
			if ok != tt.ok {
				t.Errorf("Int64() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Int64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Uint_ValidInput 测试 Uint 方法处理各种有效输入
func TestParse_Uint_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Uint(tt.input)
			if ok != tt.ok {
				t.Errorf("Uint() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Uint() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Uint_InvalidInput 测试 Uint 方法处理无效输入
func TestParse_Uint_InvalidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := p.Uint(tt.input)
			if ok {
				t.Errorf("Uint() should fail for %v", tt.input)
			}
		})
	}
}

// TestParse_Uint8_ValidInput 测试 Uint8 方法处理各种有效输入
func TestParse_Uint8_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Uint8(tt.input)
			if ok != tt.ok {
				t.Errorf("Uint8() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Uint8() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Uint8_OutOfRange 测试 Uint8 方法处理超出范围的值
func TestParse_Uint8_OutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"overflow", int(256)},
		{"negative", int(-1)},
		{"uint64 overflow", uint64(256)},
	}

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := p.Uint8(tt.input)
			if ok {
				t.Errorf("Uint8() should fail for out of range value %v", tt.input)
			}
		})
	}
}

// TestParse_Uint16_ValidInput 测试 Uint16 方法处理各种有效输入
func TestParse_Uint16_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Uint16(tt.input)
			if ok != tt.ok {
				t.Errorf("Uint16() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Uint16() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Uint32_ValidInput 测试 Uint32 方法处理各种有效输入
func TestParse_Uint32_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Uint32(tt.input)
			if ok != tt.ok {
				t.Errorf("Uint32() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Uint32() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Uint64_ValidInput 测试 Uint64 方法处理各种有效输入
func TestParse_Uint64_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Uint64(tt.input)
			if ok != tt.ok {
				t.Errorf("Uint64() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Uint64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Float32_ValidInput 测试 Float32 方法处理各种有效输入
func TestParse_Float32_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Float32(tt.input)
			if ok != tt.ok {
				t.Errorf("Float32() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Float32() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Float32_OutOfRange 测试 Float32 方法处理超出范围的值
func TestParse_Float32_OutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"overflow string", "3.4e39"},
		{"overflow float64", float64(math.MaxFloat64)},
	}

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := p.Float32(tt.input)
			if ok {
				t.Errorf("Float32() should fail for out of range value %v", tt.input)
			}
		})
	}
}

// TestParse_Float64_ValidInput 测试 Float64 方法处理各种有效输入
func TestParse_Float64_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Float64(tt.input)
			if ok != tt.ok {
				t.Errorf("Float64() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Float64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Float64_InvalidInput 测试 Float64 方法处理无效输入
func TestParse_Float64_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"empty string", ""},
		{"invalid string", "abc"},
		{"bool", true},
		{"nil", nil},
	}

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := p.Float64(tt.input)
			if ok {
				t.Errorf("Float64() should fail for %v", tt.input)
			}
		})
	}
}

// TestParse_Bool_ValidInput 测试 Bool 方法处理各种有效输入
func TestParse_Bool_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.Bool(tt.input)
			if ok != tt.ok {
				t.Errorf("Bool() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Bool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse_Bool_InvalidInput 测试 Bool 方法处理无效输入
func TestParse_Bool_InvalidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := p.Bool(tt.input)
			if ok {
				t.Errorf("Bool() should fail for %v", tt.input)
			}
		})
	}
}

// TestParse_String_ValidInput 测试 String 方法处理各种有效输入
func TestParse_String_ValidInput(t *testing.T) {
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

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := p.String(tt.input)
			if ok != tt.ok {
				t.Errorf("String() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParse_String_FmtStringer 测试 String 方法处理 fmt.Stringer 接口
func TestParse_String_FmtStringer(t *testing.T) {
	p := Parse{}

	// 使用 time.Time 作为 fmt.Stringer 的例子
	now := parseTestTime
	got, ok := p.String(now)
	if !ok {
		t.Error("String() should succeed for fmt.Stringer")
	}
	if got == "" {
		t.Error("String() should return non-empty string for time.Time")
	}
}

// TestParse_String_Error 测试 String 方法处理 error 接口
func TestParse_String_Error(t *testing.T) {
	p := Parse{}

	err := testError{msg: "test error"}
	got, ok := p.String(err)
	if !ok {
		t.Error("String() should succeed for error")
	}
	if got != "test error" {
		t.Errorf("String() = %q, want %q", got, "test error")
	}
}

// TestParse_String_InvalidInput 测试 String 方法处理无效输入
func TestParse_String_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"nil", nil},
		{"struct", struct{}{}},
		{"slice", []int{1, 2, 3}},
		{"map", map[string]int{"a": 1}},
	}

	p := Parse{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := p.String(tt.input)
			if ok {
				t.Errorf("String() should fail for %v", tt.input)
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
	// 返回一个实现了 fmt.Stringer 的对象用于测试
	return &stringerMock{s: "2024-01-01 00:00:00"}
}

type stringerMock struct {
	s string
}

func (m *stringerMock) String() string {
	return m.s
}
