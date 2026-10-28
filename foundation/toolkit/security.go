package toolkit

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// GenerateLongKey 生成一个唯一的长安全密钥字符串。
//
// 该函数拼接两个 UUIDv4 字符串并去除连字符，加上 "cs_" 前缀，
// 结果固定为 67 个字符，由小写十六进制字符组成，不含其他特殊符号。
// 随机源来自 crypto/rand，适用于会话密钥、认证凭据等安全场景。
//
// 返回值:
//   - 形如 "cs_" + 64 位小写十六进制的密钥字符串。
func GenerateLongKey() string {
	getKeyValue := uuid.NewString() + uuid.NewString()
	return "cs_" + strings.ReplaceAll(getKeyValue, "-", "")
}

// GenerateKey 生成一个唯一的短安全密钥字符串，用于标识或加密操作。
//
// 该函数取单个 UUIDv4 字符串并去除连字符，加上 "cs_" 前缀，
// 结果固定为 35 个字符，由小写十六进制字符组成；随机量约为
// [GenerateLongKey] 的一半。随机源来自 crypto/rand。
//
// 返回值:
//   - 形如 "cs_" + 32 位小写十六进制的密钥字符串。
func GenerateKey() string {
	getKeyValue := uuid.NewString()
	return "cs_" + strings.ReplaceAll(getKeyValue, "-", "")
}

// VerifyKey 验证输入字符串是否符合本包生成的安全密钥格式。
//
// 合法格式为 "cs_" 前缀后跟 64 位（[GenerateLongKey]）或
// 32 位（[GenerateKey]）小写十六进制字符，区分大小写。
// 该检查仅验证格式，不能证明密钥本身有效。
//
// 参数说明:
//   - input: 待验证的密钥字符串。
//
// 返回值:
//   - 符合格式返回 true，否则返回 false。
func VerifyKey(input string) bool {
	// 正则表达式验证：cs_ 后面跟着64个十六进制字符
	patternFirst := `^cs_[a-f0-9]{64}$`
	patternSecond := `^cs_[a-f0-9]{32}$`
	matched, err := regexp.MatchString(patternFirst, input)
	isValid := err == nil && matched
	if !isValid {
		matched, err = regexp.MatchString(patternSecond, input)
	}
	if err != nil || !matched {
		return false
	}
	return true
}
