package pack

import (
	"testing"
)

// TestEncryption_SHA256 测试 SHA256 哈希计算
func TestEncryption_SHA256(t *testing.T) {
	e := Encryption{}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"hello", "hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
		{"hello world", "hello world", "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"},
		{"number", "123", "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"},
		{"a", "a", "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"},
		{"abc", "abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := e.SHA256(tt.input); got != tt.expected {
				t.Errorf("SHA256() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestEncryption_SHA256_Consistency 测试 SHA256 一致性
func TestEncryption_SHA256_Consistency(t *testing.T) {
	e := Encryption{}
	input := "test"

	hash1 := e.SHA256(input)
	hash2 := e.SHA256(input)

	if hash1 != hash2 {
		t.Error("SHA256() should produce consistent results for same input")
	}
}

// TestEncryption_SHA256_OutputLength 测试 SHA256 输出长度
func TestEncryption_SHA256_OutputLength(t *testing.T) {
	e := Encryption{}

	tests := []string{
		"",
		"a",
		"ab",
		"abc",
		"abcd",
		"long string with many characters to ensure hash length is consistent",
		"特殊字符!@#$%",
		"中文测试",
	}

	for _, input := range tests {
		got := e.SHA256(input)
		if len(got) != 64 {
			t.Errorf("SHA256(%q) length = %d, want 64", input, len(got))
		}
	}
}

// TestEncryption_SHA256Bytes 测试 SHA256 字节切片版本
func TestEncryption_SHA256Bytes(t *testing.T) {
	e := Encryption{}
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"empty", []byte{}, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"hello", []byte("hello"), "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := e.SHA256Bytes(tt.input); got != tt.expected {
				t.Errorf("SHA256Bytes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestEncryption_SHA256Bytes_EquivalentToString 测试字节切片版本与字符串版本等效
func TestEncryption_SHA256Bytes_EquivalentToString(t *testing.T) {
	e := Encryption{}
	input := "test string"

	hashFromStr := e.SHA256(input)
	hashFromBytes := e.SHA256Bytes([]byte(input))

	if hashFromStr != hashFromBytes {
		t.Error("SHA256() and SHA256Bytes() should produce same result for equivalent input")
	}
}

// TestEncryption_MD5 测试 MD5 哈希计算
func TestEncryption_MD5(t *testing.T) {
	e := Encryption{}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "d41d8cd98f00b204e9800998ecf8427e"},
		{"hello", "hello", "5d41402abc4b2a76b9719d911017c592"},
		{"hello world", "hello world", "5eb63bbbe01eeed093cb22bb8f5acdc3"},
		{"number", "123", "202cb962ac59075b964b07152d234b70"},
		{"a", "a", "0cc175b9c0f1b6a831c399e269772661"},
		{"abc", "abc", "900150983cd24fb0d6963f7d28e17f72"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := e.MD5(tt.input); got != tt.expected {
				t.Errorf("MD5() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestEncryption_MD5_Consistency 测试 MD5 一致性
func TestEncryption_MD5_Consistency(t *testing.T) {
	e := Encryption{}
	input := "test"

	hash1 := e.MD5(input)
	hash2 := e.MD5(input)

	if hash1 != hash2 {
		t.Error("MD5() should produce consistent results for same input")
	}
}

// TestEncryption_MD5_OutputLength 测试 MD5 输出长度
func TestEncryption_MD5_OutputLength(t *testing.T) {
	e := Encryption{}

	tests := []string{
		"",
		"a",
		"ab",
		"abc",
		"long string with many characters",
		"特殊字符!@#$%",
		"中文测试",
	}

	for _, input := range tests {
		got := e.MD5(input)
		if len(got) != 32 {
			t.Errorf("MD5(%q) length = %d, want 32", input, len(got))
		}
	}
}

// TestEncryption_MD5_Lowercase 测试 MD5 输出为小写
func TestEncryption_MD5_Lowercase(t *testing.T) {
	e := Encryption{}
	input := "HELLO"

	got := e.MD5(input)
	for _, c := range got {
		if c >= 'A' && c <= 'Z' {
			t.Error("MD5() should return lowercase hex string")
			break
		}
	}
}

// TestEncryption_MD5Bytes 测试 MD5 字节切片版本
func TestEncryption_MD5Bytes(t *testing.T) {
	e := Encryption{}
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"empty", []byte{}, "d41d8cd98f00b204e9800998ecf8427e"},
		{"hello", []byte("hello"), "5d41402abc4b2a76b9719d911017c592"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := e.MD5Bytes(tt.input); got != tt.expected {
				t.Errorf("MD5Bytes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestEncryption_MD5Bytes_EquivalentToString 测试字节切片版本与字符串版本等效
func TestEncryption_MD5Bytes_EquivalentToString(t *testing.T) {
	e := Encryption{}
	input := "test string"

	hashFromStr := e.MD5(input)
	hashFromBytes := e.MD5Bytes([]byte(input))

	if hashFromStr != hashFromBytes {
		t.Error("MD5() and MD5Bytes() should produce same result for equivalent input")
	}
}

// TestEncryption_DifferentInputs 测试不同输入产生不同输出
func TestEncryption_DifferentInputs(t *testing.T) {
	e := Encryption{}

	// SHA256
	if e.SHA256("a") == e.SHA256("b") {
		t.Error("SHA256() should produce different results for different inputs")
	}

	// MD5
	if e.MD5("a") == e.MD5("b") {
		t.Error("MD5() should produce different results for different inputs")
	}
}
