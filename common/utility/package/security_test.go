package pack

import (
	"strings"
	"testing"
)

// TestSecurity_GenerateLongKey 测试生成长密钥
func TestSecurity_GenerateLongKey(t *testing.T) {
	s := Security{}
	key := s.GenerateLongKey()

	t.Logf("生成的长密钥: %s", key)

	// 验证前缀
	if !strings.HasPrefix(key, "cs_") {
		t.Errorf("GenerateLongKey() should start with 'cs_', got: %s", key[:3])
	}

	// 验证长度：cs_(3) + 32(UUID去掉4个-) + 32(UUID去掉4个-) = 67
	if len(key) != 67 {
		t.Errorf("GenerateLongKey() length = %d, want 67", len(key))
	}

	// 验证字符集（cs_ 后面应该是小写十六进制字符）
	keyPart := key[3:]
	for _, c := range keyPart {
		if !isLowerHexChar(c) {
			t.Errorf("GenerateLongKey() contains invalid character: %c", c)
			break
		}
	}
}

// TestSecurity_GenerateLongKey_Uniqueness 测试长密钥唯一性
func TestSecurity_GenerateLongKey_Uniqueness(t *testing.T) {
	s := Security{}
	results := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		key := s.GenerateLongKey()
		if results[key] {
			t.Errorf("GenerateLongKey() generated duplicate: %s", key)
		}
		results[key] = true
	}
}

// TestSecurity_GenerateKey 测试生成短密钥
func TestSecurity_GenerateKey(t *testing.T) {
	s := Security{}
	key := s.GenerateKey()

	t.Logf("生成的短密钥: %s", key)

	// 验证前缀
	if !strings.HasPrefix(key, "cs_") {
		t.Errorf("GenerateKey() should start with 'cs_', got: %s", key[:3])
	}

	// 验证长度：cs_(3) + 32(UUID去掉4个-) = 35
	if len(key) != 35 {
		t.Errorf("GenerateKey() length = %d, want 35", len(key))
	}

	// 验证字符集
	keyPart := key[3:]
	for _, c := range keyPart {
		if !isLowerHexChar(c) {
			t.Errorf("GenerateKey() contains invalid character: %c", c)
			break
		}
	}
}

// TestSecurity_GenerateKey_Uniqueness 测试短密钥唯一性
func TestSecurity_GenerateKey_Uniqueness(t *testing.T) {
	s := Security{}
	results := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		key := s.GenerateKey()
		if results[key] {
			t.Errorf("GenerateKey() generated duplicate: %s", key)
		}
		results[key] = true
	}
}

// TestSecurity_VerifyKey_ValidLongKey 测试验证有效的长密钥
func TestSecurity_VerifyKey_ValidLongKey(t *testing.T) {
	s := Security{}

	// 生成并验证长密钥
	key := s.GenerateLongKey()
	if !s.VerifyKey(key) {
		t.Errorf("VerifyKey() should return true for valid long key: %s", key)
	}
}

// TestSecurity_VerifyKey_ValidShortKey 测试验证有效的短密钥
func TestSecurity_VerifyKey_ValidShortKey(t *testing.T) {
	s := Security{}

	// 生成并验证短密钥
	key := s.GenerateKey()
	if !s.VerifyKey(key) {
		t.Errorf("VerifyKey() should return true for valid short key: %s", key)
	}
}

// TestSecurity_VerifyKey_InvalidKeys 测试验证无效密钥
func TestSecurity_VerifyKey_InvalidKeys(t *testing.T) {
	s := Security{}
	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"no prefix", "abc123"},
		{"wrong prefix", "pk_abc123"},
		{"uppercase", "CS_abc123"},
		{"too short", "cs_abc"},
		{"invalid chars", "cs_ghijklmnopqrstuvwxyz"},
		{"wrong length", "cs_" + strings.Repeat("a", 50)},
		{"spaces", "cs_ abc123"},
		{"special chars", "cs_abc-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if s.VerifyKey(tt.key) {
				t.Errorf("VerifyKey() should return false for invalid key: %s", tt.key)
			}
		})
	}
}

// TestSecurity_VerifyKey_ManualValidKeys 测试手动构造的有效密钥
func TestSecurity_VerifyKey_ManualValidKeys(t *testing.T) {
	s := Security{}
	tests := []struct {
		name string
		key  string
	}{
		{"32 hex chars", "cs_" + strings.Repeat("a", 32)},
		{"64 hex chars", "cs_" + strings.Repeat("a", 64)},
		{"mixed case (should fail)", "cs_" + strings.Repeat("A", 32)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.VerifyKey(tt.key)
			// 只有前两个应该通过
			if tt.name == "32 hex chars" || tt.name == "64 hex chars" {
				if !got {
					t.Errorf("VerifyKey() should return true for: %s", tt.key)
				}
			} else {
				if got {
					t.Errorf("VerifyKey() should return false for: %s", tt.key)
				}
			}
		})
	}
}

// TestSecurity_FullWorkflow 测试完整工作流
func TestSecurity_FullWorkflow(t *testing.T) {
	s := Security{}

	// 1. 生成长密钥
	longKey := s.GenerateLongKey()
	if len(longKey) != 67 {
		t.Errorf("GenerateLongKey() length = %d, want 67", len(longKey))
	}

	// 2. 验证长密钥
	if !s.VerifyKey(longKey) {
		t.Error("VerifyKey() should validate generated long key")
	}

	// 3. 生成短密钥
	shortKey := s.GenerateKey()
	if len(shortKey) != 35 {
		t.Errorf("GenerateKey() length = %d, want 35", len(shortKey))
	}

	// 4. 验证短密钥
	if !s.VerifyKey(shortKey) {
		t.Error("VerifyKey() should validate generated short key")
	}

	// 5. 验证长密钥和短密钥不同
	if longKey == shortKey {
		t.Error("Long key and short key should be different")
	}

	t.Logf("长密钥验证通过: %s", longKey)
	t.Logf("短密钥验证通过: %s", shortKey)
}

// 辅助函数：检查是否是小写十六进制字符
func isLowerHexChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}
