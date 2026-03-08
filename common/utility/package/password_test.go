package pack

import (
	"testing"
)

// TestPassword_Encrypt 测试密码加密功能
func TestPassword_Encrypt(t *testing.T) {
	p := Password{}

	tests := []struct {
		name     string
		password string
	}{
		{"simple password", "password123"},
		{"empty password", ""},
		{"long password", "this_is_a_very_long_password_that_should_still_work"},
		{"special chars", "p@$$w0rd!@#$%^&*()"},
		{"unicode password", "密码测试123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := p.Encrypt(tt.password)
			if err != nil {
				t.Errorf("Encrypt() error = %v", err)
				return
			}
			if len(hash) == 0 {
				t.Error("Encrypt() returned empty hash")
			}
			// bcrypt 哈希应该是 60 个字符
			if len(hash) != 60 {
				t.Errorf("Encrypt() hash length = %d, want 60", len(hash))
			}
		})
	}
}

// TestPassword_Encrypt_Uniqueness 测试相同密码产生不同哈希
func TestPassword_Encrypt_Uniqueness(t *testing.T) {
	p := Password{}
	password := "same_password"

	hash1, err := p.Encrypt(password)
	if err != nil {
		t.Fatalf("First encrypt failed: %v", err)
	}

	hash2, err := p.Encrypt(password)
	if err != nil {
		t.Fatalf("Second encrypt failed: %v", err)
	}

	// bcrypt 每次应该产生不同的哈希（因为使用了随机盐）
	if string(hash1) == string(hash2) {
		t.Error("Encrypt() should produce different hashes for same password")
	}
}

// TestPassword_EncryptString 测试加密返回字符串版本
func TestPassword_EncryptString(t *testing.T) {
	p := Password{}

	hash, err := p.EncryptString("password123")
	if err != nil {
		t.Errorf("EncryptString() error = %v", err)
		return
	}
	if len(hash) != 60 {
		t.Errorf("EncryptString() hash length = %d, want 60", len(hash))
	}
}

// TestPassword_MustEncrypt 测试 MustEncrypt 成功场景
func TestPassword_MustEncrypt(t *testing.T) {
	p := Password{}

	hash := p.MustEncrypt("password123")
	if len(hash) != 60 {
		t.Errorf("MustEncrypt() hash length = %d, want 60", len(hash))
	}
}

// TestPassword_MustEncryptString 测试 MustEncryptString 成功场景
func TestPassword_MustEncryptString(t *testing.T) {
	p := Password{}

	hash := p.MustEncryptString("password123")
	if len(hash) != 60 {
		t.Errorf("MustEncryptString() hash length = %d, want 60", len(hash))
	}
}

// TestPassword_Verify_CorrectPassword 测试验证正确密码
func TestPassword_Verify_CorrectPassword(t *testing.T) {
	p := Password{}
	password := "test_password_123"

	// 加密密码
	hash, err := p.EncryptString(password)
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}

	// 验证正确密码
	err = p.Verify(password, hash)
	if err != nil {
		t.Errorf("Verify() failed for correct password: %v", err)
	}
}

// TestPassword_Verify_IncorrectPassword 测试验证错误密码
func TestPassword_Verify_IncorrectPassword(t *testing.T) {
	p := Password{}
	password := "correct_password"

	// 加密密码
	hash, err := p.EncryptString(password)
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}

	// 验证错误密码
	err = p.Verify("wrong_password", hash)
	if err == nil {
		t.Error("Verify() should fail for incorrect password")
	}
}

// TestPassword_Verify_InvalidHash 测试验证无效哈希
func TestPassword_Verify_InvalidHash(t *testing.T) {
	p := Password{}

	// 使用无效的哈希值
	err := p.Verify("password", "invalid_hash")
	if err == nil {
		t.Error("Verify() should fail for invalid hash")
	}
}

// TestPassword_IsValid_CorrectPassword 测试 IsValid 正确密码
func TestPassword_IsValid_CorrectPassword(t *testing.T) {
	p := Password{}
	password := "test_password"

	hash := p.MustEncryptString(password)

	if !p.IsValid(password, hash) {
		t.Error("IsValid() should return true for correct password")
	}
}

// TestPassword_IsValid_IncorrectPassword 测试 IsValid 错误密码
func TestPassword_IsValid_IncorrectPassword(t *testing.T) {
	p := Password{}
	password := "correct_password"

	hash := p.MustEncryptString(password)

	if p.IsValid("wrong_password", hash) {
		t.Error("IsValid() should return false for incorrect password")
	}
}

// TestPassword_IsValid_InvalidHash 测试 IsValid 无效哈希
func TestPassword_IsValid_InvalidHash(t *testing.T) {
	p := Password{}

	if p.IsValid("password", "invalid_hash") {
		t.Error("IsValid() should return false for invalid hash")
	}
}

// TestPassword_FullFlow 测试完整流程
func TestPassword_FullFlow(t *testing.T) {
	p := Password{}
	password := "MySecurePassword123!"

	// 1. 加密密码
	hash, err := p.Encrypt(password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// 2. 验证正确密码
	if !p.IsValid(password, string(hash)) {
		t.Error("IsValid() should return true for correct password")
	}

	// 3. 验证错误密码
	if p.IsValid("WrongPassword", string(hash)) {
		t.Error("IsValid() should return false for incorrect password")
	}

	// 4. 使用 EncryptString 和 Verify
	hashStr, err := p.EncryptString(password)
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}

	err = p.Verify(password, hashStr)
	if err != nil {
		t.Errorf("Verify() error = %v", err)
	}
}

// TestPassword_SpecialCharacters 测试特殊字符密码
func TestPassword_SpecialCharacters(t *testing.T) {
	p := Password{}

	tests := []struct {
		name     string
		password string
	}{
		{"emoji", "🔐password🔐"},
		{"spaces", "pass word 123"},
		{"newlines", "pass\nword"},
		{"tabs", "pass\tword"},
		{"quotes", `"pass'word"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := p.MustEncryptString(tt.password)
			if !p.IsValid(tt.password, hash) {
				t.Errorf("IsValid() failed for password with %s", tt.name)
			}
		})
	}
}

// TestPassword_Base64Encoding 测试 Base64 编码一致性
func TestPassword_Base64Encoding(t *testing.T) {
	p := Password{}
	password := "test"

	// Encrypt 内部会对密码进行 Base64 编码
	// Verify 需要使用相同的编码方式
	hash := p.MustEncryptString(password)

	// 验证能成功
	if !p.IsValid(password, hash) {
		t.Error("IsValid() should work with internal Base64 encoding")
	}
}
