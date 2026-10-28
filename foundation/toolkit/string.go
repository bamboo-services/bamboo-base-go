package toolkit

import (
	"regexp"
	"strings"
	"unicode"
)

// IsBlank 检查字符串是否为空白（空字符串或只包含空白字符）。
//
// 空白字符按 strings.TrimSpace 的定义，包含空格、制表符、
// 换行符等 Unicode 空白。
//
// 参数说明:
//   - str: 要检查的字符串。
//
// 返回值:
//   - 如果字符串为空白返回 true，否则返回 false。
func IsBlank(str string) bool {
	return strings.TrimSpace(str) == ""
}

// IsNotBlank 检查字符串是否不为空白。
//
// 语义与 [IsBlank] 完全相反，便于在过滤链中直接引用。
//
// 参数说明:
//   - str: 要检查的字符串。
//
// 返回值:
//   - 如果字符串不为空白返回 true，否则返回 false。
func IsNotBlank(str string) bool {
	return !IsBlank(str)
}

// DefaultIfBlank 如果字符串为空白则返回默认值。
//
// 空白判断复用 [IsBlank]；字符串非空白时原样返回，不做修剪。
//
// 参数说明:
//   - str: 要检查的字符串。
//   - defaultStr: 默认值。
//
// 返回值:
//   - 如果字符串为空白返回默认值，否则返回原字符串。
func DefaultIfBlank(str, defaultStr string) string {
	if IsBlank(str) {
		return defaultStr
	}
	return str
}

// Truncate 截断字符串到指定字节长度。
//
// 长度按字节而非字符计，截断发生在字节边界，多字节字符可能被截成
// 无效 UTF-8 序列。maxLen 为负数时会触发切片越界 panic。
//
// 参数说明:
//   - str: 要截断的字符串。
//   - maxLen: 最大字节长度，不应为负。
//
// 返回值:
//   - 截断后的字符串；长度不超限时返回原字符串。
func Truncate(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen]
}

// TruncateWithSuffix 截断字符串到指定字节长度并添加后缀。
//
// 长度按字节计且包含后缀本身。suffix 为空字符串时默认使用 "..."；
// maxLen 不大于后缀长度时仅返回后缀的前 maxLen 字节；
// 字符串不超限时原样返回且不追加后缀。maxLen 为负数时会触发
// 切片越界 panic，多字节字符可能被截成无效 UTF-8 序列。
//
// 参数说明:
//   - str: 要截断的字符串。
//   - maxLen: 最大字节长度（含后缀），不应为负。
//   - suffix: 截断后追加的后缀，空字符串时使用 "..."。
//
// 返回值:
//   - 截断后的字符串（不超限时为原字符串）。
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
// 在每个大写字母（首字符除外）前插入下划线，并将全部字母转为小写。
// 连续大写字母会被逐字拆开，例如 "XMLParser" 转换为 "x_m_l_parser"。
//
// 参数说明:
//   - str: 驼峰命名字符串。
//
// 返回值:
//   - 蛇形命名字符串。
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

// SnakeToCamel 将蛇形命名转换为小驼峰命名。
//
// 按下划线分割后，首段转为小写，其余段首字母大写，
// 例如 "hello_world" 转换为 "helloWorld"；连续下划线产生的空段
// 会被忽略；全大写输入同样转换为小驼峰。
//
// 参数说明:
//   - str: 蛇形命名字符串。
//
// 返回值:
//   - 小驼峰命名字符串。
func SnakeToCamel(str string) string {
	parts := strings.Split(str, "_")
	var result strings.Builder

	for i, part := range parts {
		if i == 0 {
			result.WriteString(strings.ToLower(part))
		} else {
			result.WriteString(titleWord(strings.ToLower(part)))
		}
	}

	return result.String()
}

// titleWord 实现了 strings.Title 的转换语义，避免直接调用已弃用的 strings.Title。
func titleWord(s string) string {
	prev := ' '
	return strings.Map(
		func(r rune) rune {
			if isWordSeparator(prev) {
				prev = r
				return unicode.ToTitle(r)
			}
			prev = r
			return r
		},
		s,
	)
}

func isWordSeparator(r rune) bool {
	// ASCII 字母数字与下划线不是分隔符
	if r <= 0x7F {
		switch {
		case '0' <= r && r <= '9':
			return false
		case 'a' <= r && r <= 'z':
			return false
		case 'A' <= r && r <= 'Z':
			return false
		case r == '_':
			return false
		}
		return true
	}
	// 字母与数字不是分隔符
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return false
	}
	// 空白字符为分隔符
	return unicode.IsSpace(r)
}

// IsValidEmail 检查字符串是否为有效的邮箱地址。
//
// 基于简化的正则规则校验（本地部分允许字母、数字及 ._%+-，
// 域名须含至少一个点号且后缀为两个以上字母），并非完整的
// RFC 5322 校验，极端合法地址可能被误判。
//
// 参数说明:
//   - email: 要验证的邮箱地址。
//
// 返回值:
//   - 如果是有效邮箱返回 true，否则返回 false。
func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// Mask 对字符串进行脱敏处理。
//
// 保留开头 start 字节与结尾 end 字节，中间以 mask 字符逐字节填充；
// mask 为空字符串时默认使用 "*"。start 与 end 之和不小于字符串
// 字节长度时整串脱敏，长度保持不变。start 或 end 为负数时会触发
// 切片越界 panic；多字节字符可能被截成无效 UTF-8 序列。
//
// 参数说明:
//   - str: 要脱敏的字符串。
//   - start: 保留开头的字节数。
//   - end: 保留结尾的字节数。
//   - mask: 脱敏字符，默认为 "*"。
//
// 返回值:
//   - 脱敏后的字符串，长度与原字符串一致。
func Mask(str string, start, end int, mask string) string {
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

// RemoveSpaces 移除字符串中的所有半角空格。
//
// 仅移除 ASCII 空格 " "，不移除制表符、换行符等其他空白字符；
// 需要移除全部空白时请自行组合 strings.TrimSpace 等方法。
//
// 参数说明:
//   - str: 要处理的字符串。
//
// 返回值:
//   - 移除半角空格后的字符串。
func RemoveSpaces(str string) string {
	return strings.ReplaceAll(str, " ", "")
}

// CountWords 统计字符串中的单词数量。
//
// 语义与 strings.Fields 一致：以任意 Unicode 空白为分隔，
// 连续空白视为单个分隔符；无空白的中文整句计为一个词。
//
// 参数说明:
//   - str: 要统计的字符串。
//
// 返回值:
//   - 单词数量；空白字符串返回 0。
func CountWords(str string) int {
	str = strings.TrimSpace(str)
	if str == "" {
		return 0
	}
	return len(strings.Fields(str))
}
