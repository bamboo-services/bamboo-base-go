package toolkit

import (
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// TestPasswordFailedErrorCodeRegistry 固化密码失败错误码的数值与输出标识。
func TestPasswordFailedErrorCodeRegistry(t *testing.T) {
	if PasswordFailed == nil {
		t.Fatal("错误码不得为 nil")
	}
	if PasswordFailed.GetCode() != 50032 {
		t.Errorf("错误码数值 = %d; want 50032", PasswordFailed.GetCode())
	}
	if PasswordFailed.GetOutput() != "TOOLKIT_PASSWORD_FAILED" {
		t.Errorf("输出标识 = %s; want TOOLKIT_PASSWORD_FAILED", PasswordFailed.GetOutput())
	}
}

// TestEncryptPassword_TooLong_UsesPasswordFailedCode 固化加密失败归 50032 且保留根因。
func TestEncryptPassword_TooLong_UsesPasswordFailedCode(t *testing.T) {
	const secret = "super-secret-password-that-is-long-enough-to-break-bcrypt"
	_, err := EncryptPassword(secret)
	if err == nil {
		t.Fatal("超长密码必须返回错误")
	}
	if err.Code.Code != 50032 {
		t.Errorf("错误码 = %d; want 50032", err.Code.Code)
	}
	if err.Code.Output != "TOOLKIT_PASSWORD_FAILED" {
		t.Errorf("输出标识 = %s; want TOOLKIT_PASSWORD_FAILED", err.Code.Output)
	}
	if !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Error("必须保留 bcrypt 原始错误为根因")
	}
	if err.Err == nil {
		t.Error("必须保留根因")
	}
	if strings.Contains(err.Error(), secret) {
		t.Error("错误描述不得包含密码内容")
	}
	if !strings.Contains(err.Error(), "密码加密失败") {
		t.Errorf("错误描述 = %s; want 包含 密码加密失败", err.Error())
	}
}

// TestEncryptPassword_Success_ReturnsRealNil 固化成功路径返回真正的 nil 错误。
func TestEncryptPassword_Success_ReturnsRealNil(t *testing.T) {
	hash, err := EncryptPassword("password123")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("必须返回哈希结果")
	}
}

// TestEncryptPasswordString_Success_ReturnsRealNil 固化字符串版本成功路径返回真正的 nil 错误。
func TestEncryptPasswordString_Success_ReturnsRealNil(t *testing.T) {
	hash, err := EncryptPasswordString("password123")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if len(hash) != 60 {
		t.Errorf("哈希长度 = %d; want 60", len(hash))
	}
}

// TestVerifyPassword_Mismatch_UsesPasswordFailedCode 固化校验失败归 50032 且保留 bcrypt 根因。
func TestVerifyPassword_Mismatch_UsesPasswordFailedCode(t *testing.T) {
	hash, err := EncryptPasswordString("right-password")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	err = VerifyPassword("wrong-password", hash)
	if err == nil {
		t.Fatal("错误密码必须返回错误")
	}
	if err.Code.Code != 50032 {
		t.Errorf("错误码 = %d; want 50032", err.Code.Code)
	}
	if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		t.Error("必须保留 bcrypt 原始错误为根因")
	}
	if strings.Contains(err.Error(), "wrong-password") {
		t.Error("错误描述不得包含密码内容")
	}

	// 哈希格式非法同样归 50032。
	invalid := VerifyPassword("right-password", "not-a-bcrypt-hash")
	if invalid == nil {
		t.Fatal("非法哈希必须返回错误")
	}
	if invalid.Code.Code != 50032 {
		t.Errorf("错误码 = %d; want 50032", invalid.Code.Code)
	}
}

// TestVerifyPassword_Success_ReturnsRealNil 固化校验成功路径返回真正的 nil 错误。
func TestVerifyPassword_Success_ReturnsRealNil(t *testing.T) {
	hash, err := EncryptPasswordString("right-password")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if err := VerifyPassword("right-password", hash); err != nil {
		t.Fatalf("校验失败: %v", err)
	}
	if !IsPasswordValid("right-password", hash) {
		t.Error("正确密码必须校验通过")
	}
}

// TestParse_Failure_UsesInvalidFormatCode 固化时间解析失败归 40013 且保留根因。
func TestParse_Failure_UsesInvalidFormatCode(t *testing.T) {
	parsed, err := Parse(TimeFormatDate, "not-a-date")
	if err == nil {
		t.Fatal("非法时间必须返回错误")
	}
	if !parsed.IsZero() {
		t.Error("解析失败必须返回零值时间")
	}
	if err.Code.Code != 40013 {
		t.Errorf("错误码 = %d; want 40013", err.Code.Code)
	}
	if err.Code.Output != "VALUE_INVALID_FORMAT" {
		t.Errorf("输出标识 = %s; want VALUE_INVALID_FORMAT", err.Code.Output)
	}
	if err.Err == nil {
		t.Error("必须保留原始解析错误为根因")
	}
	var parseErr *time.ParseError
	if !errors.As(err, &parseErr) {
		t.Error("必须可提取 *time.ParseError 根因")
	}
}

// TestParse_Success_ReturnsRealNil 固化时间解析成功路径返回真正的 nil 错误。
func TestParse_Success_ReturnsRealNil(t *testing.T) {
	parsed, err := Parse(TimeFormatDate, "2024-06-15")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if parsed.Year() != 2024 || parsed.Month() != time.June || parsed.Day() != 15 {
		t.Errorf("解析结果 = %v; want 2024-06-15", parsed)
	}
}

// TestMustParse_PanicRetained 固化 Must 前缀仍保持 panic 行为。
func TestMustParse_PanicRetained(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustParse 解析失败必须 panic")
		}
	}()
	MustParse(TimeFormatDate, "invalid")
}

// TestMustEncryptPassword_PanicRetained 固化 Must 前缀密码函数仍保持 panic 行为。
func TestMustEncryptPassword_PanicRetained(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustEncryptPassword 加密失败必须 panic")
		}
	}()
	MustEncryptPassword(strings.Repeat("x", 100))
}

// TestStrongErrorImplementsErrorInterface 固化统一错误可被标准库错误机制识别。
func TestStrongErrorImplementsErrorInterface(t *testing.T) {
	var err error
	_, appErr := EncryptPassword(strings.Repeat("x", 100))
	if appErr == nil {
		t.Fatal("必须返回错误")
	}
	err = appErr
	var target *apperr.Error
	if !errors.As(err, &target) {
		t.Error("必须可经 errors.As 提取统一错误")
	}
	if target.Code.Code != 50032 {
		t.Errorf("错误码 = %d; want 50032", target.Code.Code)
	}
}
