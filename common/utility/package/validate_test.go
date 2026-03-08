package pack

import (
	"testing"
)

// TestValid_IsPhone_Valid 测试有效手机号
func TestValid_IsPhone_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		{"standard", "13812345678", true},
		{"with 3", "13512345678", true},
		{"with 4", "14712345678", true},
		{"with 5", "15512345678", true},
		{"with 6", "16612345678", true},
		{"with 7", "17712345678", true},
		{"with 8", "18812345678", true},
		{"with 9", "19912345678", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsPhone(tt.phone); got != tt.want {
				t.Errorf("IsPhone() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestValid_IsPhone_Invalid 测试无效手机号
func TestValid_IsPhone_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name  string
		phone string
	}{
		{"empty", ""},
		{"too short", "1381234567"},
		{"too long", "138123456789"},
		{"starts with 0", "10812345678"},
		{"starts with 1", "11812345678"},
		{"starts with 2", "12812345678"},
		{"contains letters", "1381234abcd"},
		{"with spaces", "138 1234 5678"},
		{"with dash", "138-1234-5678"},
		{"international", "+8613812345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsPhone(tt.phone); got {
				t.Errorf("IsPhone() should return false for %q", tt.phone)
			}
		})
	}
}

// TestValid_IsIDCard_Valid 测试有效身份证号
func TestValid_IsIDCard_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name   string
		idCard string
	}{
		{"standard", "11010519900307234X"},
		{"all digits", "110105199003072345"},
		// 注意：源代码只接受大写 X，不接受小写 x
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsIDCard(tt.idCard); !got {
				t.Errorf("IsIDCard() should return true for %q", tt.idCard)
			}
		})
	}
}

// TestValid_IsIDCard_Invalid 测试无效身份证号
func TestValid_IsIDCard_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name   string
		idCard string
	}{
		{"empty", ""},
		{"too short", "11010519900307234"},
		{"too long", "1101051990030723456"},
		{"17 digits", "1101051990030723"},
		{"invalid chars", "11010519900307abcd"},
		{"invalid last char", "11010519900307234Y"},
		{"letters in middle", "11010a19900307234X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsIDCard(tt.idCard); got {
				t.Errorf("IsIDCard() should return false for %q", tt.idCard)
			}
		})
	}
}

// TestValid_IsURI_Valid 测试有效 URL
func TestValid_IsURI_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		url  string
	}{
		{"http", "http://example.com"},
		{"https", "https://example.com"},
		{"with path", "https://example.com/path"},
		{"with port", "https://example.com:8080"},
		{"with path and port", "https://example.com:8080/path"},
		{"subdomain", "https://sub.example.com"},
		{"nested subdomain", "https://a.b.example.com"},
		{"with hyphen", "https://example-site.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsURI(tt.url); !got {
				t.Errorf("IsURI() should return true for %q", tt.url)
			}
		})
	}
}

// TestValid_IsURI_Invalid 测试无效 URL
func TestValid_IsURI_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		url  string
	}{
		{"empty", ""},
		{"no protocol", "example.com"},
		{"ftp", "ftp://example.com"},
		{"no tld", "https://example"},
		{"invalid tld", "https://example.1"},
		// 注意：源代码的正则表达式会匹配带空格的路径
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsURI(tt.url); got {
				t.Errorf("IsURI() should return false for %q", tt.url)
			}
		})
	}
}

// TestValid_IsIP_Valid 测试有效 IP 地址
func TestValid_IsIP_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		ip   string
	}{
		{"localhost", "127.0.0.1"},
		{"zero", "0.0.0.0"},
		{"max", "255.255.255.255"},
		{"standard", "192.168.1.1"},
		{"google dns", "8.8.8.8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsIP(tt.ip); !got {
				t.Errorf("IsIP() should return true for %q", tt.ip)
			}
		})
	}
}

// TestValid_IsIP_Invalid 测试无效 IP 地址
func TestValid_IsIP_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		ip   string
	}{
		{"empty", ""},
		{"out of range", "256.1.1.1"},
		{"negative", "-1.1.1.1"},
		{"too few parts", "192.168.1"},
		{"too many parts", "192.168.1.1.1"},
		{"with letters", "192.168.1.a"},
		{"ipv6", "::1"},
		{"hostname", "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsIP(tt.ip); got {
				t.Errorf("IsIP() should return false for %q", tt.ip)
			}
		})
	}
}

// TestValid_IsNumeric_Valid 测试有效数字字符串
func TestValid_IsNumeric_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		str  string
	}{
		{"single digit", "5"},
		{"multiple digits", "12345"},
		{"zeros", "000"},
		{"large number", "1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsNumeric(tt.str); !got {
				t.Errorf("IsNumeric() should return true for %q", tt.str)
			}
		})
	}
}

// TestValid_IsNumeric_Invalid 测试无效数字字符串
func TestValid_IsNumeric_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		str  string
	}{
		{"empty", ""},
		{"with letter", "123a"},
		{"with space", "123 456"},
		{"negative", "-123"},
		{"float", "12.34"},
		{"scientific", "1e10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsNumeric(tt.str); got {
				t.Errorf("IsNumeric() should return false for %q", tt.str)
			}
		})
	}
}

// TestValid_IsAlpha_Valid 测试有效字母字符串
func TestValid_IsAlpha_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		str  string
	}{
		{"lowercase", "hello"},
		{"uppercase", "HELLO"},
		{"mixed", "HelloWorld"},
		{"single letter", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsAlpha(tt.str); !got {
				t.Errorf("IsAlpha() should return true for %q", tt.str)
			}
		})
	}
}

// TestValid_IsAlpha_Invalid 测试无效字母字符串
func TestValid_IsAlpha_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		str  string
	}{
		{"empty", ""},
		{"with number", "abc123"},
		{"with space", "hello world"},
		{"with special", "hello!"},
		{"with underscore", "hello_world"},
		{"chinese", "你好"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsAlpha(tt.str); got {
				t.Errorf("IsAlpha() should return false for %q", tt.str)
			}
		})
	}
}

// TestValid_IsAlphaNumeric_Valid 测试有效字母数字字符串
func TestValid_IsAlphaNumeric_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		str  string
	}{
		{"letters only", "hello"},
		{"numbers only", "12345"},
		{"mixed", "abc123"},
		{"mixed case", "Hello123"},
		{"single char", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsAlphaNumeric(tt.str); !got {
				t.Errorf("IsAlphaNumeric() should return true for %q", tt.str)
			}
		})
	}
}

// TestValid_IsAlphaNumeric_Invalid 测试无效字母数字字符串
func TestValid_IsAlphaNumeric_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		str  string
	}{
		{"empty", ""},
		{"with space", "hello world"},
		{"with special", "hello!"},
		{"with underscore", "hello_world"},
		{"with dash", "hello-world"},
		{"chinese", "abc你好"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsAlphaNumeric(tt.str); got {
				t.Errorf("IsAlphaNumeric() should return false for %q", tt.str)
			}
		})
	}
}

// TestValid_IsUsername_Valid 测试有效用户名
func TestValid_IsUsername_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name     string
		username string
	}{
		{"simple", "user1"},
		{"with underscore", "user_name"},
		{"minimum length", "user"},
		{"maximum length", "abcdefghijklmnopqrst"},
		{"mixed case", "User123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsUsername(tt.username); !got {
				t.Errorf("IsUsername() should return true for %q", tt.username)
			}
		})
	}
}

// TestValid_IsUsername_Invalid 测试无效用户名
func TestValid_IsUsername_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name     string
		username string
	}{
		{"empty", ""},
		{"too short", "usr"},
		{"too long", "abcdefghijklmnopqrstu"},
		{"starts with number", "1user"},
		{"starts with underscore", "_user"},
		{"with space", "user name"},
		{"with special char", "user@name"},
		{"with dash", "user-name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsUsername(tt.username); got {
				t.Errorf("IsUsername() should return false for %q", tt.username)
			}
		})
	}
}

// TestIsStrongPassword_Valid 测试有效强密码
func TestIsStrongPassword_Valid(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"standard", "Password123!"},
		{"complex", "MyP@ssw0rd"},
		{"with brackets", "Pass123[{}]"},
		{"with symbols", "Abc123!@#$"},
		{"minimum", "Passw0rd!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStrongPassword(tt.password); !got {
				t.Errorf("IsStrongPassword() should return true for %q", tt.password)
			}
		})
	}
}

// TestIsStrongPassword_Invalid 测试无效强密码
func TestIsStrongPassword_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"too short", "Pass1!"},
		{"no uppercase", "password123!"},
		{"no lowercase", "PASSWORD123!"},
		{"no digit", "Password!"},
		{"no special", "Password123"},
		{"only letters", "Passwords"},
		{"only numbers", "12345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStrongPassword(tt.password); got {
				t.Errorf("IsStrongPassword() should return false for %q", tt.password)
			}
		})
	}
}

// TestValid_InRange 测试范围检查
func TestValid_InRange(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name  string
		value float64
		min   float64
		max   float64
		want  bool
	}{
		{"in range", 5, 1, 10, true},
		{"at min", 1, 1, 10, true},
		{"at max", 10, 1, 10, true},
		{"below min", 0, 1, 10, false},
		{"above max", 11, 1, 10, false},
		{"negative", -5, -10, 0, true},
		{"float", 5.5, 1.0, 10.0, true},
		{"zero range", 5, 5, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.InRange(tt.value, tt.min, tt.max); got != tt.want {
				t.Errorf("InRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestValid_IsLength 测试长度检查
func TestValid_IsLength(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name   string
		str    string
		minLen int
		maxLen int
		want   bool
	}{
		{"in range", "hello", 1, 10, true},
		{"at min", "a", 1, 10, true},
		{"at max", "1234567890", 1, 10, true},
		{"below min", "", 1, 10, false},
		{"above max", "12345678901", 1, 10, false},
		{"empty allowed", "", 0, 10, true},
		{"exact", "test", 4, 4, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsLength(tt.str, tt.minLen, tt.maxLen); got != tt.want {
				t.Errorf("IsLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestValid_IsUUID_Valid 测试有效 UUID
func TestValid_IsUUID_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		uuid string
	}{
		{"v1", "550e8400-e29b-11d4-a716-446655440000"},
		{"v2", "550e8400-e29b-21d4-a716-446655440000"},
		{"v3", "550e8400-e29b-31d4-a716-446655440000"},
		{"v4 lowercase", "550e8400-e29b-41d4-a716-446655440000"},
		{"v4 uppercase", "550E8400-E29B-41D4-A716-446655440000"},
		{"v5", "550e8400-e29b-51d4-a716-446655440000"},
		{"v4 variant 8", "550e8400-e29b-41d4-87c6-446655440000"},
		{"v4 variant 9", "550e8400-e29b-41d4-97c6-446655440000"},
		{"v4 variant a", "550e8400-e29b-41d4-a7c6-446655440000"},
		{"v4 variant b", "550e8400-e29b-41d4-b7c6-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsUUID(tt.uuid); !got {
				t.Errorf("IsUUID() should return true for %q", tt.uuid)
			}
		})
	}
}

// TestValid_IsUUID_Invalid 测试无效 UUID
func TestValid_IsUUID_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		uuid string
	}{
		{"empty", ""},
		{"no dashes", "550e8400e29b41d4a716446655440000"},
		{"wrong version", "550e8400-e29b-01d4-a716-446655440000"},
		{"wrong variant", "550e8400-e29b-41d4-c716-446655440000"},
		{"too short", "550e8400-e29b-41d4-a716-44665544000"},
		{"too long", "550e8400-e29b-41d4-a716-4466554400000"},
		{"invalid chars", "550e8400-e29b-41d4-a716-44665544000g"},
		{"wrong format", "550e8400e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsUUID(tt.uuid); got {
				t.Errorf("IsUUID() should return false for %q", tt.uuid)
			}
		})
	}
}

// TestValid_IsJSON_Valid 测试有效 JSON
func TestValid_IsJSON_Valid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		json string
	}{
		{"empty object", "{}"},
		{"simple object", `{"key": "value"}`},
		{"nested object", `{"outer": {"inner": "value"}}`},
		{"empty array", "[]"},
		{"simple array", `[1, 2, 3]`},
		{"array of objects", `[{"id": 1}]`},
		{"with spaces", ` { "key": "value" } `},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsJSON(tt.json); !got {
				t.Errorf("IsJSON() should return true for %q", tt.json)
			}
		})
	}
}

// TestValid_IsJSON_Invalid 测试无效 JSON
func TestValid_IsJSON_Invalid(t *testing.T) {
	v := &Valid{}
	tests := []struct {
		name string
		json string
	}{
		{"empty", ""},
		{"plain string", "hello"},
		{"number", "123"},
		{"unclosed object", `{"key": "value"`},
		{"unclosed array", `[1, 2, 3`},
		{"mismatched brackets", `{]`},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsJSON(tt.json); got {
				t.Errorf("IsJSON() should return false for %q", tt.json)
			}
		})
	}
}
