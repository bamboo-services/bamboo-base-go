package pack

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Security 安全密钥工具结构体，提供安全密钥的生成与验证方法。
//
// 使用方式：
//
//	xUtil.Security().GenerateLongKey()
//	xUtil.Security().GenerateKey()
//	xUtil.Security().VerifyKey("cs_xxx")
type Security struct{}

// GenerateLongKey 生成一个唯一的安全密钥字符串。
//
// 该函数通过生成两个 UUID 字符串，并将其组合成一个新的字符串，最终返回一个前缀为 "cs_" 的安全密钥。
//
// 安全密钥不包含任何特殊符号，且长度足够长以保证唯一性，适用于会话或认证等场景。
//
// 返回值:
//   - 返回一个字符串类型的安全密钥，确保其唯一性。
func (Security) GenerateLongKey() string {
	getKeyValue := uuid.NewString() + uuid.NewString()
	return "cs_" + strings.ReplaceAll(getKeyValue, "-", "")
}

// GenerateKey 生成一个唯一的安全密钥字符串，用于标识或加密操作。
func (Security) GenerateKey() string {
	getKeyValue := uuid.NewString()
	return "cs_" + strings.ReplaceAll(getKeyValue, "-", "")
}

// VerifyKey 验证输入字符串是否符合特定的安全密钥格式。
//
// 输入必须以 "cs_" 开头，后跟 64 或 32 个十六进制字符组成。
// 返回 true 表示输入符合要求的格式，false 表示格式不匹配或发生错误。
func (Security) VerifyKey(input string) bool {
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
