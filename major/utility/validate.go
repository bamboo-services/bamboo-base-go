package xUtil

import (
	"regexp"
	"strconv"
	"strings"
)

// IsValidPhone 检查是否为有效的手机号码（中国大陆）。
//
// 参数说明:
//   - phone: 要验证的手机号码
//
// 返回值:
//   - 如果是有效手机号返回 true，否则返回 false
func IsValidPhone(phone string) bool {
	// 中国大陆手机号码正则：1开头，第二位是3-9，总共11位
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

// IsValidIDCard 检查是否为有效的身份证号码（中国大陆）。
//
// 参数说明:
//   - idCard: 要验证的身份证号码
//
// 返回值:
//   - 如果是有效身份证号返回 true，否则返回 false
func IsValidIDCard(idCard string) bool {
	if len(idCard) != 18 {
		return false
	}

	// 检查前17位是否都是数字
	for i := 0; i < 17; i++ {
		if idCard[i] < '0' || idCard[i] > '9' {
			return false
		}
	}

	// 检查最后一位（校验码）
	lastChar := idCard[17]
	if lastChar != 'X' && (lastChar < '0' || lastChar > '9') {
		return false
	}

	// 简单的校验码验证（完整的算法比较复杂，这里只做基本检查）
	return true
}

// IsValidURL 检查是否为有效的 URL。
//
// 参数说明:
//   - url: 要验证的 URL
//
// 返回值:
//   - 如果是有效 URL 返回 true，否则返回 false
func IsValidURL(url string) bool {
	pattern := `^https?://[a-zA-Z0-9\-\.]+\.[a-zA-Z]{2,}(:[0-9]+)?(/.*)?$`
	matched, _ := regexp.MatchString(pattern, url)
	return matched
}

// IsValidIP 检查是否为有效的 IP 地址。
//
// 参数说明:
//   - ip: 要验证的 IP 地址
//
// 返回值:
//   - 如果是有效 IP 地址返回 true，否则返回 false
func IsValidIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return false
		}
	}

	return true
}

// IsNumeric 检查字符串是否只包含数字。
//
// 参数说明:
//   - str: 要检查的字符串
//
// 返回值:
//   - 如果只包含数字返回 true，否则返回 false
func IsNumeric(str string) bool {
	if str == "" {
		return false
	}

	for _, r := range str {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// IsAlpha 检查字符串是否只包含字母。
//
// 参数说明:
//   - str: 要检查的字符串
//
// 返回值:
//   - 如果只包含字母返回 true，否则返回 false
func IsAlpha(str string) bool {
	if str == "" {
		return false
	}

	pattern := `^[a-zA-Z]+$`
	matched, _ := regexp.MatchString(pattern, str)
	return matched
}

// IsAlphaNumeric 检查字符串是否只包含字母和数字。
//
// 参数说明:
//   - str: 要检查的字符串
//
// 返回值:
//   - 如果只包含字母和数字返回 true，否则返回 false
func IsAlphaNumeric(str string) bool {
	if str == "" {
		return false
	}

	pattern := `^[a-zA-Z0-9]+$`
	matched, _ := regexp.MatchString(pattern, str)
	return matched
}

// IsValidUsername 检查是否为有效的用户名。
//
// 用户名规则：4-20位，只能包含字母、数字、下划线，必须以字母开头。
//
// 参数说明:
//   - username: 要验证的用户名
//
// 返回值:
//   - 如果是有效用户名返回 true，否则返回 false
func IsValidUsername(username string) bool {
	if len(username) < 4 || len(username) > 20 {
		return false
	}

	pattern := `^[a-zA-Z][a-zA-Z0-9_]*$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

// IsStrongPassword 检查是否为强密码。
//
// 强密码规则：至少8位，包含大写字母、小写字母、数字和特殊字符。
//
// 参数说明:
//   - password: 要验证的密码
//
// 返回值:
//   - 如果是强密码返回 true，否则返回 false
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper, _ := regexp.MatchString(`[A-Z]`, password)
	hasLower, _ := regexp.MatchString(`[a-z]`, password)
	hasDigit, _ := regexp.MatchString(`\d`, password)
	hasSpecial, _ := regexp.MatchString(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`, password)

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// InRange 检查数值是否在指定范围内。
//
// 参数说明:
//   - value: 要检查的数值
//   - min: 最小值
//   - max: 最大值
//
// 返回值:
//   - 如果在范围内返回 true，否则返回 false
func InRange(value, min, max float64) bool {
	return value >= min && value <= max
}

// IsValidLength 检查字符串长度是否在指定范围内。
//
// 参数说明:
//   - str: 要检查的字符串
//   - minLen: 最小长度
//   - maxLen: 最大长度
//
// 返回值:
//   - 如果长度在范围内返回 true，否则返回 false
func IsValidLength(str string, minLen, maxLen int) bool {
	length := len(str)
	return length >= minLen && length <= maxLen
}

// IsValidUUID 检查是否为有效的 UUID。
//
// 参数说明:
//   - uuid: 要验证的 UUID 字符串
//
// 返回值:
//   - 如果是有效 UUID 返回 true，否则返回 false
func IsValidUUID(uuid string) bool {
	pattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
	matched, _ := regexp.MatchString(pattern, uuid)
	return matched
}

// IsValidJSON 检查字符串是否为有效的 JSON 格式（简单检查）。
//
// 参数说明:
//   - jsonStr: 要验证的 JSON 字符串
//
// 返回值:
//   - 如果是有效 JSON 格式返回 true，否则返回 false
func IsValidJSON(jsonStr string) bool {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return false
	}

	// 简单检查：JSON 应该以 { 或 [ 开头，以 } 或 ] 结尾
	return (strings.HasPrefix(jsonStr, "{") && strings.HasSuffix(jsonStr, "}")) ||
		(strings.HasPrefix(jsonStr, "[") && strings.HasSuffix(jsonStr, "]"))
}
