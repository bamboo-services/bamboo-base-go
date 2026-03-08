package pack

import (
	"testing"
)

// TestGenerate_RandomString_Length 测试随机字符串长度
func TestGenerate_RandomString_Length(t *testing.T) {
	g := Generate{}
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
			got := g.RandomString(tt.length)
			if len(got) != tt.length {
				t.Errorf("RandomString() length = %d, want %d", len(got), tt.length)
			}
		})
	}
}

// TestGenerate_RandomString_Charset 测试随机字符串字符集
func TestGenerate_RandomString_Charset(t *testing.T) {
	g := Generate{}
	result := g.RandomString(1000) // 生成较长的字符串来验证字符集

	for _, c := range result {
		// 检查字符是否在允许的字符集中（a-z, A-Z, 0-9）
		valid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !valid {
			t.Errorf("RandomString() contains invalid character: %c", c)
			break
		}
	}
}

// TestGenerate_RandomString_Uniqueness 测试随机字符串唯一性
func TestGenerate_RandomString_Uniqueness(t *testing.T) {
	g := Generate{}
	results := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		s := g.RandomString(32)
		if results[s] {
			t.Errorf("RandomString() generated duplicate: %s", s)
		}
		results[s] = true
	}
}

// TestGenerate_RandomUpperString_Length 测试大写随机字符串长度
func TestGenerate_RandomUpperString_Length(t *testing.T) {
	g := Generate{}
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
			got := g.RandomUpperString(tt.length)
			if len(got) != tt.length {
				t.Errorf("RandomUpperString() length = %d, want %d", len(got), tt.length)
			}
		})
	}
}

// TestGenerate_RandomUpperString_Charset 测试大写随机字符串字符集
func TestGenerate_RandomUpperString_Charset(t *testing.T) {
	g := Generate{}
	result := g.RandomUpperString(1000)

	for _, c := range result {
		// 检查字符是否在允许的字符集中（A-Z, 0-9）
		valid := (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !valid {
			t.Errorf("RandomUpperString() contains invalid character: %c", c)
			break
		}
	}
}

// TestGenerate_RandomUpperString_UppercaseOnly 测试仅包含大写字母和数字
func TestGenerate_RandomUpperString_UppercaseOnly(t *testing.T) {
	g := Generate{}
	result := g.RandomUpperString(1000)

	for _, c := range result {
		// 检查没有小写字母
		if c >= 'a' && c <= 'z' {
			t.Errorf("RandomUpperString() contains lowercase character: %c", c)
			break
		}
	}
}

// TestGenerate_RandomUpperString_Uniqueness 测试大写随机字符串唯一性
func TestGenerate_RandomUpperString_Uniqueness(t *testing.T) {
	g := Generate{}
	results := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		s := g.RandomUpperString(32)
		if results[s] {
			t.Errorf("RandomUpperString() generated duplicate: %s", s)
		}
		results[s] = true
	}
}

// TestGenerate_RandomNumberString_Length 测试数字随机字符串长度
func TestGenerate_RandomNumberString_Length(t *testing.T) {
	g := Generate{}
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
			got := g.RandomNumberString(tt.length)
			if len(got) != tt.length {
				t.Errorf("RandomNumberString() length = %d, want %d", len(got), tt.length)
			}
		})
	}
}

// TestGenerate_RandomNumberString_Charset 测试数字随机字符串字符集
func TestGenerate_RandomNumberString_Charset(t *testing.T) {
	g := Generate{}
	result := g.RandomNumberString(1000)

	for _, c := range result {
		// 检查字符是否在允许的字符集中（0-9）
		if c < '0' || c > '9' {
			t.Errorf("RandomNumberString() contains invalid character: %c", c)
			break
		}
	}
}

// TestGenerate_RandomNumberString_NumbersOnly 测试仅包含数字
func TestGenerate_RandomNumberString_NumbersOnly(t *testing.T) {
	g := Generate{}
	result := g.RandomNumberString(1000)

	for _, c := range result {
		// 检查没有字母
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			t.Errorf("RandomNumberString() contains letter: %c", c)
			break
		}
	}
}

// TestGenerate_RandomNumberString_Uniqueness 测试数字随机字符串唯一性
func TestGenerate_RandomNumberString_Uniqueness(t *testing.T) {
	g := Generate{}
	results := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		s := g.RandomNumberString(16)
		if results[s] {
			t.Errorf("RandomNumberString() generated duplicate: %s", s)
		}
		results[s] = true
	}
}

// TestGenerate_AllMethods_Comparison 测试三种方法的区别
func TestGenerate_AllMethods_Comparison(t *testing.T) {
	g := Generate{}
	length := 16

	randomStr := g.RandomString(length)
	upperStr := g.RandomUpperString(length)
	numberStr := g.RandomNumberString(length)

	// 验证长度相同
	if len(randomStr) != length || len(upperStr) != length || len(numberStr) != length {
		t.Error("All methods should produce strings of the same length")
	}

	// 验证三个结果不相同（高概率）
	if randomStr == upperStr || randomStr == numberStr || upperStr == numberStr {
		t.Log("Warning: Different methods produced the same result (extremely unlikely but possible)")
	}
}
