package toolkit

import (
	"testing"
)

// TestRandomString_Length 测试随机字符串长度
func TestRandomString_Length(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero", 0},
		{"one", 1},
		{"ten", 10},
		{"thirty two", 32},
		{"sixty four", 64},
		{"large", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RandomString(tt.length)
			if len(got) != tt.length {
				t.Errorf("RandomString() length = %d, want %d", len(got), tt.length)
			}
		})
	}
}

// TestRandomString_Charset 测试随机字符串字符集
func TestRandomString_Charset(t *testing.T) {
	result := RandomString(1000) // 生成较长的字符串来验证字符集

	for _, c := range result {
		// 检查字符是否在允许的字符集中（a-z, A-Z, 0-9）
		valid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !valid {
			t.Errorf("RandomString() contains invalid character: %c", c)
			break
		}
	}
}

// TestRandomString_Uniqueness 测试随机字符串唯一性
func TestRandomString_Uniqueness(t *testing.T) {
	results := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		s := RandomString(32)
		if results[s] {
			t.Errorf("RandomString() generated duplicate: %s", s)
		}
		results[s] = true
	}
}

// TestRandomUpperString_Length 测试大写随机字符串长度
func TestRandomUpperString_Length(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero", 0},
		{"one", 1},
		{"ten", 10},
		{"thirty two", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RandomUpperString(tt.length)
			if len(got) != tt.length {
				t.Errorf("RandomUpperString() length = %d, want %d", len(got), tt.length)
			}
		})
	}
}

// TestRandomUpperString_Charset 测试大写随机字符串字符集
func TestRandomUpperString_Charset(t *testing.T) {
	result := RandomUpperString(1000)

	for _, c := range result {
		// 检查字符是否在允许的字符集中（A-Z, 0-9）
		valid := (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !valid {
			t.Errorf("RandomUpperString() contains invalid character: %c", c)
			break
		}
	}
}

// TestRandomUpperString_UppercaseOnly 测试仅包含大写字母和数字
func TestRandomUpperString_UppercaseOnly(t *testing.T) {
	result := RandomUpperString(1000)

	for _, c := range result {
		// 检查没有小写字母
		if c >= 'a' && c <= 'z' {
			t.Errorf("RandomUpperString() contains lowercase character: %c", c)
			break
		}
	}
}

// TestRandomUpperString_Uniqueness 测试大写随机字符串唯一性
func TestRandomUpperString_Uniqueness(t *testing.T) {
	results := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		s := RandomUpperString(32)
		if results[s] {
			t.Errorf("RandomUpperString() generated duplicate: %s", s)
		}
		results[s] = true
	}
}

// TestRandomNumberString_Length 测试数字随机字符串长度
func TestRandomNumberString_Length(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero", 0},
		{"one", 1},
		{"ten", 10},
		{"thirty two", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RandomNumberString(tt.length)
			if len(got) != tt.length {
				t.Errorf("RandomNumberString() length = %d, want %d", len(got), tt.length)
			}
		})
	}
}

// TestRandomNumberString_Charset 测试数字随机字符串字符集
func TestRandomNumberString_Charset(t *testing.T) {
	result := RandomNumberString(1000)

	for _, c := range result {
		// 检查字符是否在允许的字符集中（0-9）
		if c < '0' || c > '9' {
			t.Errorf("RandomNumberString() contains invalid character: %c", c)
			break
		}
	}
}

// TestRandomNumberString_NumbersOnly 测试仅包含数字
func TestRandomNumberString_NumbersOnly(t *testing.T) {
	result := RandomNumberString(1000)

	for _, c := range result {
		// 检查没有字母
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			t.Errorf("RandomNumberString() contains letter: %c", c)
			break
		}
	}
}

// TestRandomNumberString_Uniqueness 测试数字随机字符串唯一性
func TestRandomNumberString_Uniqueness(t *testing.T) {
	results := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		s := RandomNumberString(16)
		if results[s] {
			t.Errorf("RandomNumberString() generated duplicate: %s", s)
		}
		results[s] = true
	}
}

// TestGenerate_AllMethods_Comparison 测试三种方法的区别
func TestGenerate_AllMethods_Comparison(t *testing.T) {
	length := 16

	randomStr := RandomString(length)
	upperStr := RandomUpperString(length)
	numberStr := RandomNumberString(length)

	// 验证长度相同
	if len(randomStr) != length || len(upperStr) != length || len(numberStr) != length {
		t.Error("All methods should produce strings of the same length")
	}

	// 验证三个结果不相同（高概率）
	if randomStr == upperStr || randomStr == numberStr || upperStr == numberStr {
		t.Log("Warning: Different methods produced the same result (extremely unlikely but possible)")
	}
}
