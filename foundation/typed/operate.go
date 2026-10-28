package typed

import (
	"strings"
)

// String 返回底层原生 string。
func (s String) String() string {
	return string(s)
}

// TypedString 返回自身 typed.String，满足 Operable 契约。
func (s String) TypedString() String {
	return s
}

// Bytes 返回底层字符串的字节切片拷贝。
func (s String) Bytes() []byte {
	return []byte(s)
}

// Len 返回字符串的字节长度。
func (s String) Len() int {
	return len(s)
}

// IsEmpty 报告字符串字节长度是否为 0。
func (s String) IsEmpty() bool {
	return len(s) == 0
}

// IsBlank 报告去除首尾空白字符后是否为空。
func (s String) IsBlank() bool {
	return len(strings.TrimSpace(string(s))) == 0
}

// Trim 返回去除首尾空白字符后的 typed.String。
func (s String) Trim() String {
	return String(strings.TrimSpace(string(s)))
}

// Upper 返回所有字符转为大写后的 typed.String。
func (s String) Upper() String {
	return String(strings.ToUpper(string(s)))
}

// Lower 返回所有字符转为小写后的 typed.String。
func (s String) Lower() String {
	return String(strings.ToLower(string(s)))
}

// Equal 比较两个 typed.String 是否完全一致。
func (s String) Equal(other String) bool {
	return s == other
}

// IsValid 校验受控标识符语法规则：
// 1. 长度必须在 1 到 MaxKeyBytes (128) 字节之间；
// 2. 首字符必须是 ASCII 字母 (A-Z 或 a-z)；
// 3. 后续字符仅允许 ASCII 字母、数字 (0-9)、下划线 '_' 与连字符 '-'；
// 4. 分隔符 ('_' 或 '-') 不能出现在末尾；
// 5. 分隔符不能连续出现（例如 '__'、'--'、'_-'、'-_' 均视为非法）。
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
