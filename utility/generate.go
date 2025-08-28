package xUtil

import (
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
