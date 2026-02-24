package xUtil

import (
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// EncryptPassword 将密码加密并返回加密后的字节切片及可能发生的错误。
//
// 该函数首先对输入密码进行 Base64 编码，然后使用 bcrypt 进行哈希处理。
//
// 参数说明:
//   - pass: 需要加密的明文密码
//
// 返回值:
//   - []byte: bcrypt 哈希后的密码
//   - error: 加密过程中可能出现的错误
func EncryptPassword(pass string) ([]byte, error) {
	// 先进行 Base64 编码
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(pass)))
	base64.StdEncoding.Encode(encoded, []byte(pass))

	// 然后使用 bcrypt 进行哈希
	return bcrypt.GenerateFromPassword(encoded, bcrypt.DefaultCost)
}

// MustEncryptPassword 加密密码并返回加密后的字节切片。
//
// 这是 EncryptPassword 的便捷版本，直接返回加密后的字节切片。
//
// 参数说明:
//   - pass: 需要加密的明文密码
//
// 返回值:
//   - []byte: bcrypt 哈希后的密码
func MustEncryptPassword(pass string) []byte {
	hash, err := EncryptPassword(pass)
	if err != nil {
		panic(err)
	}
	return hash
}

// VerifyPassword 验证用户输入的密码是否与加密后的密码匹配。
//
// 参数说明:
//   - inputPass: 用户输入的明文密码
//   - hashPass: 存储的加密密码哈希值
//
// 返回值:
//   - error: 如果密码匹配返回 nil，否则返回错误
func VerifyPassword(inputPass, hashPass string) error {
	// 对输入密码进行 Base64 编码（与加密时保持一致）
	encodedInput := make([]byte, base64.StdEncoding.EncodedLen(len(inputPass)))
	base64.StdEncoding.Encode(encodedInput, []byte(inputPass))

	// 使用 bcrypt 验证密码
	return bcrypt.CompareHashAndPassword([]byte(hashPass), encodedInput)
}

// EncryptPasswordString 将密码加密并返回加密后的字符串。
//
// 这是 EncryptPassword 的便捷版本，直接返回字符串形式的哈希值。
//
// 参数说明:
//   - pass: 需要加密的明文密码
//
// 返回值:
//   - string: 加密后的密码字符串
//   - error: 加密过程中可能出现的错误
func EncryptPasswordString(pass string) (string, error) {
	hash, err := EncryptPassword(pass)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// MustEncryptPasswordString 加密密码并返回加密后的字符串。
//
// 这是 EncryptPassword 的便捷版本，直接返回字符串形式的哈希值。
//
// 参数说明:
//   - pass: 需要加密的明文密码
//
// 返回值:
//   - string: 加密后的密码字符串
func MustEncryptPasswordString(pass string) string {
	hash, err := EncryptPassword(pass)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

// IsPasswordValid 检查密码是否匹配，返回布尔值。
//
// 这是 VerifyPassword 的便捷版本，直接返回是否匹配的布尔值。
//
// 参数说明:
//   - inputPass: 用户输入的明文密码
//   - hashPass: 存储的加密密码哈希值
//
// 返回值:
//   - bool: 密码匹配返回 true，否则返回 false
func IsPasswordValid(inputPass, hashPass string) bool {
	return VerifyPassword(inputPass, hashPass) == nil
}
