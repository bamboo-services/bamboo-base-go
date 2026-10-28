package typed

import (
	"math"
	"strconv"
	"testing"
	"time"
)

// TestStringParseIntFamily 测试 Int/Int8/Int16/Int32/Int64 整数族解析。
func TestStringParseIntFamily(t *testing.T) {
	t.Run("Int", func(t *testing.T) {
		maxInt := strconv.FormatInt(math.MaxInt, 10)
		minInt := strconv.FormatInt(math.MinInt, 10)
		tests := []struct {
			name    string
			in      String
			want    int
			wantErr bool
		}{
			{name: "合法正整数带空格", in: "  42  ", want: 42, wantErr: false},
			{name: "合法负整数", in: "-100", want: -100, wantErr: false},
			{name: "当前平台MaxInt", in: String(maxInt), want: math.MaxInt, wantErr: false},
			{name: "当前平台MinInt", in: String(minInt), want: math.MinInt, wantErr: false},
			{name: "上溢出", in: "9223372036854775808", wantErr: true},
			{name: "下溢出", in: "-9223372036854775809", wantErr: true},
			{name: "非十进制十六进制非法", in: "0x1f", wantErr: true},
			{name: "非数字字符非法", in: "42a", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Int()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Int() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Int() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("Int8", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    int8
			wantErr bool
		}{
			{name: "正常解析带空格", in: " \t 127 \n", want: 127, wantErr: false},
			{name: "下界-128", in: "-128", want: -128, wantErr: false},
			{name: "上界127", in: "127", want: 127, wantErr: false},
			{name: "上溢128", in: "128", wantErr: true},
			{name: "下溢-129", in: "-129", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Int8()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Int8() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Int8() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("Int16", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    int16
			wantErr bool
		}{
			{name: "下界-32768", in: "-32768", want: -32768, wantErr: false},
			{name: "上界32767", in: "32767", want: 32767, wantErr: false},
			{name: "上溢32768", in: "32768", wantErr: true},
			{name: "下溢-32769", in: "-32769", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Int16()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Int16() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Int16() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("Int32", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    int32
			wantErr bool
		}{
			{name: "下界-2147483648", in: "-2147483648", want: math.MinInt32, wantErr: false},
			{name: "上界2147483647", in: "2147483647", want: math.MaxInt32, wantErr: false},
			{name: "上溢2147483648", in: "2147483648", wantErr: true},
			{name: "下溢-2147483649", in: "-2147483649", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Int32()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Int32() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Int32() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("Int64", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    int64
			wantErr bool
		}{
			{name: "下界-9223372036854775808", in: "-9223372036854775808", want: math.MinInt64, wantErr: false},
			{name: "上界9223372036854775807", in: "9223372036854775807", want: math.MaxInt64, wantErr: false},
			{name: "上溢", in: "9223372036854775808", wantErr: true},
			{name: "下溢", in: "-9223372036854775809", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Int64()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Int64() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Int64() = %d, want %d", got, tt.want)
				}
			})
		}
	})
}

// TestStringParseUintFamily 测试 Uint/Uint8/Uint16/Uint32/Uint64 无符号整数族解析。
func TestStringParseUintFamily(t *testing.T) {
	t.Run("Uint", func(t *testing.T) {
		maxUint := strconv.FormatUint(math.MaxUint, 10)
		tests := []struct {
			name    string
			in      String
			want    uint
			wantErr bool
		}{
			{name: "合法无符号数带空格", in: "  88  ", want: 88, wantErr: false},
			{name: "当前平台MaxUint", in: String(maxUint), want: math.MaxUint, wantErr: false},
			{name: "负数非法", in: "-1", wantErr: true},
			{name: "溢出", in: "18446744073709551616", wantErr: true},
			{name: "非十进制八进制非法", in: "077", want: 77, wantErr: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Uint()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Uint() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Uint() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("Uint8", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    uint8
			wantErr bool
		}{
			{name: "下界0", in: "0", want: 0, wantErr: false},
			{name: "上界255", in: "255", want: 255, wantErr: false},
			{name: "上溢256", in: "256", wantErr: true},
			{name: "负数非法", in: "-1", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Uint8()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Uint8() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Uint8() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("Uint16", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    uint16
			wantErr bool
		}{
			{name: "下界0", in: "0", want: 0, wantErr: false},
			{name: "上界65535", in: "65535", want: math.MaxUint16, wantErr: false},
			{name: "上溢65536", in: "65536", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Uint16()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Uint16() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Uint16() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("Uint32", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    uint32
			wantErr bool
		}{
			{name: "下界0", in: "0", want: 0, wantErr: false},
			{name: "上界4294967295", in: "4294967295", want: math.MaxUint32, wantErr: false},
			{name: "上溢4294967296", in: "4294967296", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Uint32()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Uint32() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Uint32() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("Uint64", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    uint64
			wantErr bool
		}{
			{name: "下界0", in: "0", want: 0, wantErr: false},
			{name: "上界18446744073709551615", in: "18446744073709551615", want: math.MaxUint64, wantErr: false},
			{name: "上溢18446744073709551616", in: "18446744073709551616", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Uint64()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Uint64() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Uint64() = %d, want %d", got, tt.want)
				}
			})
		}
	})
}

// TestStringParseFloatFamily 测试 Float32/Float64 解析及其对 NaN、Inf、溢出的严格限制。
func TestStringParseFloatFamily(t *testing.T) {
	t.Run("Float32", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    float32
			wantErr bool
		}{
			{name: "合法普通浮点", in: " 3.14159 ", want: 3.14159, wantErr: false},
			{name: "合法负数浮点", in: "-12.5", want: -12.5, wantErr: false},
			{name: "拒绝NaN小写", in: "nan", wantErr: true},
			{name: "拒绝NaN大写", in: "NAN", wantErr: true},
			{name: "拒绝Inf", in: "inf", wantErr: true},
			{name: "拒绝+Inf", in: "+inf", wantErr: true},
			{name: "拒绝-Inf", in: "-Inf", wantErr: true},
			{name: "拒绝Infinity", in: "Infinity", wantErr: true},
			{name: "拒绝-Infinity", in: "-Infinity", wantErr: true},
			{name: "超出float32范围溢出为Inf", in: "1e39", wantErr: true},
			{name: "非数字字符非法", in: "abc", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Float32()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Float32() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Float32() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Float64", func(t *testing.T) {
		tests := []struct {
			name    string
			in      String
			want    float64
			wantErr bool
		}{
			{name: "合法普通浮点带空格", in: "  2.718281828459045  ", want: 2.718281828459045, wantErr: false},
			{name: "合法负浮点", in: "-0.0001", want: -0.0001, wantErr: false},
			{name: "拒绝NaN", in: "NaN", wantErr: true},
			{name: "拒绝Inf", in: "Inf", wantErr: true},
			{name: "拒绝+Inf", in: "+Inf", wantErr: true},
			{name: "拒绝-Inf", in: "-Inf", wantErr: true},
			{name: "拒绝Infinity", in: "Infinity", wantErr: true},
			{name: "拒绝-Infinity", in: "-Infinity", wantErr: true},
			{name: "超出float64范围溢出为Inf", in: "1e309", wantErr: true},
			{name: "非数字字符非法", in: "not-a-number", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Float64()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Float64() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("Float64() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

// TestStringParseBool 测试 10 种布尔词的大小写支持与非法词校验。
func TestStringParseBool(t *testing.T) {
	tests := []struct {
		name    string
		in      String
		want    bool
		wantErr bool
	}{
		// 5 种真值
		{name: "true小写", in: "true", want: true, wantErr: false},
		{name: "true大写带空格", in: "  TRUE  ", want: true, wantErr: false},
		{name: "1", in: "1", want: true, wantErr: false},
		{name: "yes小写", in: "yes", want: true, wantErr: false},
		{name: "yes混合大小写", in: "Yes", want: true, wantErr: false},
		{name: "on小写", in: "on", want: true, wantErr: false},
		{name: "on大写", in: "ON", want: true, wantErr: false},
		{name: "enabled小写", in: "enabled", want: true, wantErr: false},
		{name: "enabled混合大小写", in: "Enabled", want: true, wantErr: false},

		// 5 种假值
		{name: "false小写", in: "false", want: false, wantErr: false},
		{name: "false大写带空格", in: " FALSE \n", want: false, wantErr: false},
		{name: "0", in: "0", want: false, wantErr: false},
		{name: "no小写", in: "no", want: false, wantErr: false},
		{name: "no大写", in: "NO", want: false, wantErr: false},
		{name: "off小写", in: "off", want: false, wantErr: false},
		{name: "off大写", in: "OFF", want: false, wantErr: false},
		{name: "disabled小写", in: "disabled", want: false, wantErr: false},
		{name: "disabled大写", in: "DISABLED", want: false, wantErr: false},

		// 非法输入
		{name: "空字符串非法", in: "", wantErr: true},
		{name: "纯空白非法", in: "   ", wantErr: true},
		{name: "非布尔词非法", in: "maybe", wantErr: true},
		{name: "2非法", in: "2", wantErr: true},
		{name: "-1非法", in: "-1", wantErr: true},
		{name: "ok非法", in: "ok", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.Bool()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Bool() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Bool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStringParseDuration 测试合法与非法时间间隔解析。
func TestStringParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		in      String
		want    time.Duration
		wantErr bool
	}{
		{name: "合法秒数带空格", in: "  30s  ", want: 30 * time.Second, wantErr: false},
		{name: "合法毫秒", in: "500ms", want: 500 * time.Millisecond, wantErr: false},
		{name: "合法复合时长", in: "1h30m", want: 90 * time.Minute, wantErr: false},
		{name: "合法微秒", in: "100us", want: 100 * time.Microsecond, wantErr: false},
		{name: "合法纳秒", in: "250ns", want: 250 * time.Nanosecond, wantErr: false},
		{name: "空字符串非法", in: "", wantErr: true},
		{name: "缺失单位非法", in: "30", wantErr: true},
		{name: "非法单位", in: "30days", wantErr: true},
		{name: "纯文本非法", in: "duration", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.Duration()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Duration() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Duration() = %v, want %v", got, tt.want)
			}
		})
	}
}
