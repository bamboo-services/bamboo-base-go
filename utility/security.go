package xUtil

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// GenerateLongSecurityKey 生成一个唯一的安全密钥字符串。
//
// 该函数通过生成两个 UUID 字符串，并将其组合成一个新的字符串，最终返回一个前缀为 "cs_" 的安全密钥。
//
// 安全密钥不包含任何特殊符号，且长度足够长以保证唯一性，适用于会话或认证等场景。
//
// 返回值:
//   - 返回一个字符串类型的安全密钥，确保其唯一性。
func GenerateLongSecurityKey() string {
	getKeyValue := uuid.NewString() + uuid.NewString()
	return "cs_" + strings.ReplaceAll(getKeyValue, "-", "")
}

// GenerateSecurityKey 生成一个唯一的安全密钥字符串，用于标识或加密操作。
func GenerateSecurityKey() string {
	getKeyValue := uuid.NewString()
	return "cs_" + strings.ReplaceAll(getKeyValue, "-", "")
}

// VerifySecurityKey 验证输入字符串是否符合特定的安全密钥格式。
//
// 输入必须以 "cs_" 开头，后跟 64 或 32 个十六进制字符组成。
// 返回 true 表示输入符合要求的格式，false 表示格式不匹配或发生错误。
func VerifySecurityKey(input string) bool {
	// 正则表达式验证：cs_ 后面跟着64个十六进制字符
	patternFirst := `^cs_[a-f0-9]{64}$`
	patternSecond := `^cs_[a-f0-9]{32}$`
	matched, err := regexp.MatchString(patternFirst, input)
	isValid := false
	if err == nil && matched {
		isValid = true
	}
	if !isValid {
		matched, err = regexp.MatchString(patternSecond, input)
	}
	if err != nil || !matched {
		return false
	}
	return true
}
