package toolkit

import (
	"testing"
)

// TestEncryptPassword 测试密码加密功能
func TestEncryptPassword(t *testing.T) {
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
			hash, err := EncryptPassword(tt.password)
			if err != nil {
				t.Errorf("EncryptPassword() error = %v", err)
				return
			}
			if len(hash) == 0 {
				t.Error("EncryptPassword() returned empty hash")
			}
			// bcrypt 哈希应该是 60 个字符
			if len(hash) != 60 {
				t.Errorf("EncryptPassword() hash length = %d, want 60", len(hash))
			}
		})
	}
}

// TestEncryptPassword_Uniqueness 测试相同密码产生不同哈希
func TestEncryptPassword_Uniqueness(t *testing.T) {
	password := "same_password"

	hash1, err := EncryptPassword(password)
	if err != nil {
		t.Fatalf("First encrypt failed: %v", err)
	}

	hash2, err := EncryptPassword(password)
	if err != nil {
		t.Fatalf("Second encrypt failed: %v", err)
	}

	// bcrypt 每次应该产生不同的哈希（因为使用了随机盐）
	if string(hash1) == string(hash2) {
		t.Error("EncryptPassword() should produce different hashes for same password")
	}
}

// TestEncryptPasswordString 测试加密返回字符串版本
func TestEncryptPasswordString(t *testing.T) {
	hash, err := EncryptPasswordString("password123")
	if err != nil {
		t.Errorf("EncryptPasswordString() error = %v", err)
		return
	}
	if len(hash) != 60 {
		t.Errorf("EncryptPasswordString() hash length = %d, want 60", len(hash))
	}
}

// TestMustEncryptPassword 测试 MustEncryptPassword 成功场景
func TestMustEncryptPassword(t *testing.T) {
	hash := MustEncryptPassword("password123")
	if len(hash) != 60 {
		t.Errorf("MustEncryptPassword() hash length = %d, want 60", len(hash))
	}
}

// TestMustEncryptPasswordString 测试 MustEncryptPasswordString 成功场景
func TestMustEncryptPasswordString(t *testing.T) {
	hash := MustEncryptPasswordString("password123")
	if len(hash) != 60 {
		t.Errorf("MustEncryptPasswordString() hash length = %d, want 60", len(hash))
	}
}

// TestVerifyPassword_CorrectPassword 测试验证正确密码
func TestVerifyPassword_CorrectPassword(t *testing.T) {
	password := "test_password_123"

	// 加密密码
	hash, err := EncryptPasswordString(password)
	if err != nil {
		t.Fatalf("EncryptPasswordString() error = %v", err)
	}

	// 验证正确密码
	err = VerifyPassword(password, hash)
	if err != nil {
		t.Errorf("VerifyPassword() failed for correct password: %v", err)
	}
}

// TestVerifyPassword_IncorrectPassword 测试验证错误密码
func TestVerifyPassword_IncorrectPassword(t *testing.T) {
	password := "correct_password"

	// 加密密码
	hash, err := EncryptPasswordString(password)
	if err != nil {
		t.Fatalf("EncryptPasswordString() error = %v", err)
	}

	// 验证错误密码
	err = VerifyPassword("wrong_password", hash)
	if err == nil {
		t.Error("VerifyPassword() should fail for incorrect password")
	}
}

// TestVerifyPassword_InvalidHash 测试验证无效哈希
func TestVerifyPassword_InvalidHash(t *testing.T) {
	// 使用无效的哈希值
	err := VerifyPassword("password", "invalid_hash")
	if err == nil {
		t.Error("VerifyPassword() should fail for invalid hash")
	}
}

// TestIsPasswordValid_CorrectPassword 测试 IsPasswordValid 正确密码
func TestIsPasswordValid_CorrectPassword(t *testing.T) {
	password := "test_password"

	hash := MustEncryptPasswordString(password)

	if !IsPasswordValid(password, hash) {
		t.Error("IsPasswordValid() should return true for correct password")
	}
}

// TestIsPasswordValid_IncorrectPassword 测试 IsPasswordValid 错误密码
func TestIsPasswordValid_IncorrectPassword(t *testing.T) {
	password := "correct_password"

	hash := MustEncryptPasswordString(password)

	if IsPasswordValid("wrong_password", hash) {
		t.Error("IsPasswordValid() should return false for incorrect password")
	}
}

// TestIsPasswordValid_InvalidHash 测试 IsPasswordValid 无效哈希
func TestIsPasswordValid_InvalidHash(t *testing.T) {
	if IsPasswordValid("password", "invalid_hash") {
		t.Error("IsPasswordValid() should return false for invalid hash")
	}
}

// TestPassword_FullFlow 测试完整流程
func TestPassword_FullFlow(t *testing.T) {
	password := "MySecurePassword123!"

	// 1. 加密密码
	hash, err := EncryptPassword(password)
	if err != nil {
		t.Fatalf("EncryptPassword() error = %v", err)
	}

	// 2. 验证正确密码
	if !IsPasswordValid(password, string(hash)) {
		t.Error("IsPasswordValid() should return true for correct password")
	}

	// 3. 验证错误密码
	if IsPasswordValid("WrongPassword", string(hash)) {
		t.Error("IsPasswordValid() should return false for incorrect password")
	}

	// 4. 使用 EncryptPasswordString 和 VerifyPassword
	hashStr, err := EncryptPasswordString(password)
	if err != nil {
		t.Fatalf("EncryptPasswordString() error = %v", err)
	}

	err = VerifyPassword(password, hashStr)
	if err != nil {
		t.Errorf("VerifyPassword() error = %v", err)
	}
}

// TestPassword_SpecialCharacters 测试特殊字符密码
func TestPassword_SpecialCharacters(t *testing.T) {
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
			hash := MustEncryptPasswordString(tt.password)
			if !IsPasswordValid(tt.password, hash) {
				t.Errorf("IsPasswordValid() failed for password with %s", tt.name)
			}
		})
	}
}

// TestPassword_Base64Encoding 测试 Base64 编码一致性
func TestPassword_Base64Encoding(t *testing.T) {
	password := "test"

	// EncryptPassword 内部会对密码进行 Base64 编码
	// VerifyPassword 需要使用相同的编码方式
	hash := MustEncryptPasswordString(password)

	// 验证能成功
	if !IsPasswordValid(password, hash) {
		t.Error("IsPasswordValid() should work with internal Base64 encoding")
	}
}
