package xUtil

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
	"strings"

	"github.com/google/uuid"
)

// GenerateSecurityKey 生成一个唯一的安全密钥字符串。
//
// 该函数通过生成两个 UUID 字符串，并将其组合成一个新的字符串，最终返回一个前缀为 "cs_" 的安全密钥。
//
// 安全密钥不包含任何特殊符号，且长度足够长以保证唯一性，适用于会话或认证等场景。
//
// 返回值:
//   - 返回一个字符串类型的安全密钥，确保其唯一性。
func GenerateSecurityKey() string {
	getKeyValue := uuid.NewString() + uuid.NewString()
	return "cs_" + strings.ReplaceAll(getKeyValue, "-", "")
}

// GenerateRandomString 根据指定长度生成一个包含字母和数字的随机字符串。
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// GenerateRandomUpperString 根据指定长度生成一个仅包含大写字母和数字的随机字符串。
func GenerateRandomUpperString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// GenerateMD5 根据输入字符串生成其对应的 MD5 哈希值并以十六进制字符串的形式返回。
func GenerateMD5(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}
