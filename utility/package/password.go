package pack

import (
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// Password 密码工具结构体，提供密码加密与验证方法。
//
// 使用方式：
//
//	xUtil.Password().Encrypt("password")
//	xUtil.Password().IsValid("input", "hash")
type Password struct{}

// Encrypt 将密码加密并返回加密后的字节切片及可能发生的错误。
//
// 该函数首先对输入密码进行 Base64 编码，然后使用 bcrypt 进行哈希处理。
//
// 参数说明:
//   - pass: 需要加密的明文密码
//
// 返回值:
//   - []byte: bcrypt 哈希后的密码
//   - error: 加密过程中可能出现的错误
func (Password) Encrypt(pass string) ([]byte, error) {
	// 先进行 Base64 编码
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(pass)))
	base64.StdEncoding.Encode(encoded, []byte(pass))

	// 然后使用 bcrypt 进行哈希
	return bcrypt.GenerateFromPassword(encoded, bcrypt.DefaultCost)
}

// MustEncrypt 加密密码并返回加密后的字节切片。
//
// 这是 Encrypt 的便捷版本，直接返回加密后的字节切片。
//
// 参数说明:
//   - pass: 需要加密的明文密码
//
// 返回值:
//   - []byte: bcrypt 哈希后的密码
func (p Password) MustEncrypt(pass string) []byte {
	hash, err := p.Encrypt(pass)
	if err != nil {
		panic(err)
	}
	return hash
}

// Verify 验证用户输入的密码是否与加密后的密码匹配。
//
// 参数说明:
//   - inputPass: 用户输入的明文密码
//   - hashPass: 存储的加密密码哈希值
//
// 返回值:
//   - error: 如果密码匹配返回 nil，否则返回错误
func (Password) Verify(inputPass, hashPass string) error {
	// 对输入密码进行 Base64 编码（与加密时保持一致）
	encodedInput := make([]byte, base64.StdEncoding.EncodedLen(len(inputPass)))
	base64.StdEncoding.Encode(encodedInput, []byte(inputPass))

	// 使用 bcrypt 验证密码
	return bcrypt.CompareHashAndPassword([]byte(hashPass), encodedInput)
}

// EncryptString 将密码加密并返回加密后的字符串。
//
// 这是 Encrypt 的便捷版本，直接返回字符串形式的哈希值。
//
// 参数说明:
//   - pass: 需要加密的明文密码
//
// 返回值:
//   - string: 加密后的密码字符串
//   - error: 加密过程中可能出现的错误
func (p Password) EncryptString(pass string) (string, error) {
	hash, err := p.Encrypt(pass)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// MustEncryptString 加密密码并返回加密后的字符串。
//
// 这是 Encrypt 的便捷版本，直接返回字符串形式的哈希值。
//
// 参数说明:
//   - pass: 需要加密的明文密码
//
// 返回值:
//   - string: 加密后的密码字符串
func (p Password) MustEncryptString(pass string) string {
	hash, err := p.Encrypt(pass)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

// IsValid 检查密码是否匹配，返回布尔值。
//
// 这是 Verify 的便捷版本，直接返回是否匹配的布尔值。
//
// 参数说明:
//   - inputPass: 用户输入的明文密码
//   - hashPass: 存储的加密密码哈希值
//
// 返回值:
//   - bool: 密码匹配返回 true，否则返回 false
func (p Password) IsValid(inputPass, hashPass string) bool {
	return p.Verify(inputPass, hashPass) == nil
}
