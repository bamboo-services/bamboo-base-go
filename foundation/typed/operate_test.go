package typed

import (
	"bytes"
	"strings"
	"testing"
)

// TestStringOperate 测试 typed.String 的常用操作方法。
func TestStringOperate(t *testing.T) {
	tests := []struct {
		name       string
		in         String
		wantStr    string
		wantBytes  []byte
		wantLen    int
		wantEmpty  bool
		wantBlank  bool
		wantTrim   String
		wantUpper  String
		wantLower  String
		equalOther String
		wantEqual  bool
	}{
		{
			name:       "空字符串",
			in:         "",
			wantStr:    "",
			wantBytes:  []byte{},
			wantLen:    0,
			wantEmpty:  true,
			wantBlank:  true,
			wantTrim:   "",
			wantUpper:  "",
			wantLower:  "",
			equalOther: "",
			wantEqual:  true,
		},
		{
			name:       "纯空白字符串",
			in:         "  \t \n ",
			wantStr:    "  \t \n ",
			wantBytes:  []byte("  \t \n "),
			wantLen:    6,
			wantEmpty:  false,
			wantBlank:  true,
			wantTrim:   "",
			wantUpper:  "  \t \n ",
			wantLower:  "  \t \n ",
			equalOther: "  \t \n ",
			wantEqual:  true,
		},
		{
			name:       "混合大小写与首尾空白",
			in:         "  Hello_World-123  ",
			wantStr:    "  Hello_World-123  ",
			wantBytes:  []byte("  Hello_World-123  "),
			wantLen:    19,
			wantEmpty:  false,
			wantBlank:  false,
			wantTrim:   "Hello_World-123",
			wantUpper:  "  HELLO_WORLD-123  ",
			wantLower:  "  hello_world-123  ",
			equalOther: "Hello_World-123",
			wantEqual:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.String(); got != tt.wantStr {
				t.Errorf("String() = %q, want %q", got, tt.wantStr)
			}
			if got := tt.in.TypedString(); got != tt.in {
				t.Errorf("TypedString() = %q, want %q", got, tt.in)
			}
			if got := tt.in.Bytes(); !bytes.Equal(got, tt.wantBytes) {
				t.Errorf("Bytes() = %v, want %v", got, tt.wantBytes)
			}
			if got := tt.in.Len(); got != tt.wantLen {
				t.Errorf("Len() = %d, want %d", got, tt.wantLen)
			}
			if got := tt.in.IsEmpty(); got != tt.wantEmpty {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.wantEmpty)
			}
			if got := tt.in.IsBlank(); got != tt.wantBlank {
				t.Errorf("IsBlank() = %v, want %v", got, tt.wantBlank)
			}
			if got := tt.in.Trim(); got != tt.wantTrim {
				t.Errorf("Trim() = %q, want %q", got, tt.wantTrim)
			}
			if got := tt.in.Upper(); got != tt.wantUpper {
				t.Errorf("Upper() = %q, want %q", got, tt.wantUpper)
			}
			if got := tt.in.Lower(); got != tt.wantLower {
				t.Errorf("Lower() = %q, want %q", got, tt.wantLower)
			}
			if got := tt.in.Equal(tt.equalOther); got != tt.wantEqual {
				t.Errorf("Equal(%q) = %v, want %v", tt.equalOther, got, tt.wantEqual)
			}
		})
	}
}

// TestStringIsValid 测试受控标识符语法校验规则。
func TestStringIsValid(t *testing.T) {
	exact128 := "A" + strings.Repeat("a", 127)
	exact129 := "A" + strings.Repeat("a", 128)

	tests := []struct {
		name string
		in   String
		want bool
	}{
		{name: "空字符串", in: "", want: false},
		{name: "ASCII首字母单字符合法", in: "a", want: true},
		{name: "ASCII大写首字母合法", in: "Z", want: true},
		{name: "数字开头非法", in: "1abc", want: false},
		{name: "下划线开头非法", in: "_abc", want: false},
		{name: "连字符开头非法", in: "-abc", want: false},
		{name: "下划线末尾非法", in: "abc_", want: false},
		{name: "连字符末尾非法", in: "abc-", want: false},
		{name: "连续下划线非法", in: "abc__def", want: false},
		{name: "连续连字符非法", in: "abc--def", want: false},
		{name: "下划线后连字符非法", in: "abc_-def", want: false},
		{name: "连字符后下划线非法", in: "abc-_def", want: false},
		{name: "含数字与合法分隔符", in: "app-v2_service1", want: true},
		{name: "恰好128字节合法", in: String(exact128), want: true},
		{name: "超过128字节(129)非法", in: String(exact129), want: false},
		{name: "含中文非ASCII非法", in: "a_中文", want: false},
		{name: "中文开头非法", in: "中_abc", want: false},
		{name: "含特殊符号非法", in: "abc@def", want: false},
		{name: "含空格非法", in: "abc def", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.IsValid(); got != tt.want {
				t.Errorf("IsValid() for %q = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
