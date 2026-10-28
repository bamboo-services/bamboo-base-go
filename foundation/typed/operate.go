package typed

import (
	"strings"
)

// String 返回底层原生 string，使 String 满足 fmt.Stringer。
func (s String) String() string {
	return string(s)
}

// TypedString 返回自身值，满足 [Operable] 契约，供接口层统一复用。
func (s String) TypedString() String {
	return s
}

// Bytes 返回底层字符串内容的字节切片副本。
// 每次调用分配新切片，修改返回值不影响原字符串。
func (s String) Bytes() []byte {
	return []byte(s)
}

// Len 返回底层字符串的字节长度，非 Unicode 字符数。
func (s String) Len() int {
	return len(s)
}

// IsEmpty 报告底层字符串字节长度是否为 0。
func (s String) IsEmpty() bool {
	return len(s) == 0
}

// IsBlank 报告去除首尾 Unicode 空白后是否为空。
// 空字符串与纯空白字符串（如 " \t "）均返回 true。
func (s String) IsBlank() bool {
	return len(strings.TrimSpace(string(s))) == 0
}

// Trim 返回去除首尾 Unicode 空白后的新值，规则同 strings.TrimSpace。
func (s String) Trim() String {
	return String(strings.TrimSpace(string(s)))
}

// Upper 返回所有 Unicode 字符转为大写后的新值。
func (s String) Upper() String {
	return String(strings.ToUpper(string(s)))
}

// Lower 返回所有 Unicode 字符转为小写后的新值。
func (s String) Lower() String {
	return String(strings.ToLower(string(s)))
}

// Equal 按字节精确比较两个值是否一致，大小写敏感。
func (s String) Equal(other String) bool {
	return s == other
}

// IsValid 按受控标识符语法规则校验底层字符串，全部条件满足才返回 true：
//
//   - 长度：1 至 [MaxKeyBytes] 字节；
//   - 首字符：ASCII 字母（A-Z 或 a-z）；
//   - 其余字符：ASCII 字母、数字（0-9）、下划线 '_' 或连字符 '-'；
//   - 末字符：不能是下划线或连字符；
//   - 分隔符：不能连续出现（"__"、"--"、"_-"、"-_" 均非法）。
//
// 校验按原始字节执行，不忽略首尾空白，非 ASCII 内容一律不合法。
func (s String) IsValid() bool {
	n := len(s)
	if n < 1 || n > MaxKeyBytes {
		return false
	}

	first := s[0]
	if !isASCIILetter(first) {
		return false
	}

	last := s[n-1]
	if isSeparator(last) {
		return false
	}

	for i := 1; i < n; i++ {
		b := s[i]
		if isSeparator(b) {
			if isSeparator(s[i-1]) {
				return false
			}
			continue
		}
		if !isASCIILetter(b) && !isASCIIDigit(b) {
			return false
		}
	}

	return true
}

func isASCIILetter(b byte) bool {
	return ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}

func isASCIIDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

func isSeparator(b byte) bool {
	return b == '_' || b == '-'
}
