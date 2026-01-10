package xUtil

import (
	"regexp"
	"strings"
	"unicode"
)

// IsBlank 检查字符串是否为空白（空字符串或只包含空白字符）。
//
// 参数说明:
//   - str: 要检查的字符串
//
// 返回值:
//   - 如果字符串为空白返回 true，否则返回 false
func IsBlank(str string) bool {
	return strings.TrimSpace(str) == ""
}

// IsNotBlank 检查字符串是否不为空白。
//
// 参数说明:
//   - str: 要检查的字符串
//
// 返回值:
//   - 如果字符串不为空白返回 true，否则返回 false
func IsNotBlank(str string) bool {
	return !IsBlank(str)
}

// DefaultIfBlank 如果字符串为空白则返回默认值。
//
// 参数说明:
//   - str: 要检查的字符串
//   - defaultStr: 默认值
//
// 返回值:
//   - 如果字符串为空白返回默认值，否则返回原字符串
func DefaultIfBlank(str, defaultStr string) string {
	if IsBlank(str) {
		return defaultStr
	}
	return str
}

// Truncate 截断字符串到指定长度。
//
// 参数说明:
//   - str: 要截断的字符串
//   - maxLen: 最大长度
//
// 返回值:
//   - 截断后的字符串
func Truncate(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen]
}

// TruncateWithSuffix 截断字符串到指定长度并添加后缀。
//
// 参数说明:
//   - str: 要截断的字符串
//   - maxLen: 最大长度（包含后缀）
//   - suffix: 后缀字符串，默认为 "..."
//
// 返回值:
//   - 截断后的字符串（包含后缀）
func TruncateWithSuffix(str string, maxLen int, suffix string) string {
	if suffix == "" {
		suffix = "..."
	}

	if len(str) <= maxLen {
		return str
	}

	if maxLen <= len(suffix) {
		return suffix[:maxLen]
	}

	return str[:maxLen-len(suffix)] + suffix
}

// CamelToSnake 将驼峰命名转换为蛇形命名。
//
// 参数说明:
//   - str: 驼峰命名字符串
//
// 返回值:
//   - 蛇形命名字符串
func CamelToSnake(str string) string {
	var result strings.Builder

	for i, r := range str {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}

	return result.String()
}

// SnakeToCamel 将蛇形命名转换为驼峰命名。
//
// 参数说明:
//   - str: 蛇形命名字符串
//
// 返回值:
//   - 驼峰命名字符串
func SnakeToCamel(str string) string {
	parts := strings.Split(str, "_")
	var result strings.Builder

	for i, part := range parts {
		if i == 0 {
			result.WriteString(strings.ToLower(part))
		} else {
			result.WriteString(strings.Title(strings.ToLower(part)))
		}
	}

	return result.String()
}

// IsValidEmail 检查字符串是否为有效的邮箱地址。
//
// 参数说明:
//   - email: 要验证的邮箱地址
//
// 返回值:
//   - 如果是有效邮箱返回 true，否则返回 false
func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// MaskString 对字符串进行脱敏处理。
//
// 参数说明:
//   - str: 要脱敏的字符串
//   - start: 保留开头的字符数
//   - end: 保留结尾的字符数
//   - mask: 脱敏字符，默认为 "*"
//
// 返回值:
//   - 脱敏后的字符串
func MaskString(str string, start, end int, mask string) string {
	if mask == "" {
		mask = "*"
	}

	strLen := len(str)
	if strLen <= start+end {
		return strings.Repeat(mask, strLen)
	}

	maskLen := strLen - start - end
	return str[:start] + strings.Repeat(mask, maskLen) + str[strLen-end:]
}

// RemoveSpaces 移除字符串中的所有空白字符。
//
// 参数说明:
//   - str: 要处理的字符串
//
// 返回值:
//   - 移除空白字符后的字符串
func RemoveSpaces(str string) string {
	return strings.ReplaceAll(str, " ", "")
}

// CountWords 统计字符串中的单词数量。
//
// 参数说明:
//   - str: 要统计的字符串
//
// 返回值:
//   - 单词数量
func CountWords(str string) int {
	str = strings.TrimSpace(str)
	if str == "" {
		return 0
	}
	return len(strings.Fields(str))
}
