package toolkit

import (
	"regexp"
	"strconv"
	"strings"
)

// IsPhone 检查是否为有效的中国大陆手机号码。
//
// 规则为 11 位数字，以 1 开头且第二位为 3-9；
// 不支持国际区号前缀与分隔符。
//
// 参数说明:
//   - phone: 要验证的手机号码。
//
// 返回值:
//   - 如果是有效手机号返回 true，否则返回 false。
func IsPhone(phone string) bool {
	// 中国大陆手机号码正则：1开头，第二位是3-9，总共11位
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

// IsIDCard 检查是否为有效的中国大陆居民身份证号码。
//
// 仅做格式检查：长度为 18 位，前 17 位为数字，末位为数字或大写 X
// （小写 x 不被接受）；不校验 GB 11643 校验码、出生日期与
// 行政区划的语义正确性。
//
// 参数说明:
//   - idCard: 要验证的身份证号码。
//
// 返回值:
//   - 如果格式合法返回 true，否则返回 false。
func IsIDCard(idCard string) bool {
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

// IsURI 检查是否为有效的 HTTP/HTTPS URL。
//
// 仅接受 http 或 https 协议，域名段由字母、数字、连字符与点号组成，
// 须以两个以上字母的顶级后缀结尾，端口与路径可选。
// 基于简化正则，非完整 URL 规范校验（路径部分允许任意字符）。
//
// 参数说明:
//   - url: 要验证的 URL。
//
// 返回值:
//   - 如果是有效 URL 返回 true，否则返回 false。
func IsURI(url string) bool {
	pattern := `^https?://[a-zA-Z0-9\-\.]+\.[a-zA-Z]{2,}(:[0-9]+)?(/.*)?$`
	matched, _ := regexp.MatchString(pattern, url)
	return matched
}

// IsIP 检查是否为有效的 IPv4 地址。
//
// 仅支持点分十进制 IPv4，每段取值 0-255，不校验前导零
// （如 "01.2.3.4" 也会通过），不支持 IPv6。
//
// 参数说明:
//   - ip: 要验证的 IP 地址。
//
// 返回值:
//   - 如果是有效 IPv4 地址返回 true，否则返回 false。
func IsIP(ip string) bool {
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

// IsNumeric 检查字符串是否只包含 ASCII 数字 0-9。
//
// 空字符串返回 false；不支持负号、小数点与其他 Unicode 数字。
//
// 参数说明:
//   - str: 要检查的字符串。
//
// 返回值:
//   - 如果只包含数字返回 true，否则返回 false。
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

// IsAlpha 检查字符串是否只包含 ASCII 字母。
//
// 空字符串返回 false；不包含数字、下划线与非 ASCII 字母。
//
// 参数说明:
//   - str: 要检查的字符串。
//
// 返回值:
//   - 如果只包含字母返回 true，否则返回 false。
func IsAlpha(str string) bool {
	if str == "" {
		return false
	}

	pattern := `^[a-zA-Z]+$`
	matched, _ := regexp.MatchString(pattern, str)
	return matched
}

// IsAlphaNumeric 检查字符串是否只包含 ASCII 字母和数字。
//
// 空字符串返回 false；不包含下划线与其他符号。
//
// 参数说明:
//   - str: 要检查的字符串。
//
// 返回值:
//   - 如果只包含字母和数字返回 true，否则返回 false。
func IsAlphaNumeric(str string) bool {
	if str == "" {
		return false
	}

	pattern := `^[a-zA-Z0-9]+$`
	matched, _ := regexp.MatchString(pattern, str)
	return matched
}

// IsUsername 检查是否为有效的用户名。
//
// 规则：长度 4-20（按字节计），只能包含 ASCII 字母、数字、下划线，
// 且必须以字母开头。
//
// 参数说明:
//   - username: 要验证的用户名。
//
// 返回值:
//   - 如果是有效用户名返回 true，否则返回 false。
func IsUsername(username string) bool {
	if len(username) < 4 || len(username) > 20 {
		return false
	}

	pattern := `^[a-zA-Z][a-zA-Z0-9_]*$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

// IsStrongPassword 检查是否为强密码。
//
// 规则：长度至少 8（按字节计），且同时包含大写字母、小写字母、
// 数字与特殊字符，特殊字符集合为常见 ASCII 标点
// （!@#$%^&*()_+-=[]{};':"\|,.<>/?）。
//
// 参数说明:
//   - password: 要验证的密码。
//
// 返回值:
//   - 如果是强密码返回 true，否则返回 false。
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
// 采用闭区间判断，边界值（等于 min 或 max）视为在范围内。
//
// 参数说明:
//   - value: 要检查的数值。
//   - min: 最小值（含）。
//   - max: 最大值（含）。
//
// 返回值:
//   - 如果在范围内返回 true，否则返回 false。
func InRange(value, min, max float64) bool {
	return value >= min && value <= max
}

// IsLength 检查字符串字节长度是否在指定范围内。
//
// 长度按字节而非字符计（多字节字符占多个字节），
// 边界值采用闭区间判断。
//
// 参数说明:
//   - str: 要检查的字符串。
//   - minLen: 最小长度（含）。
//   - maxLen: 最大长度（含）。
//
// 返回值:
//   - 如果长度在范围内返回 true，否则返回 false。
func IsLength(str string, minLen, maxLen int) bool {
	length := len(str)
	return length >= minLen && length <= maxLen
}

// IsUUID 检查是否为有效的 UUID 字符串。
//
// 要求标准 8-4-4-4-12 连字符格式，十六进制大小写均可；
// 版本位限定为 1-5 且变体位为 8/9/a/b（大小写均可），
// 因此 v6、v7、v8 等新版本 UUID 不会被接受。
//
// 参数说明:
//   - uuid: 要验证的 UUID 字符串。
//
// 返回值:
//   - 如果是有效 UUID 返回 true，否则返回 false。
func IsUUID(uuid string) bool {
	pattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
	matched, _ := regexp.MatchString(pattern, uuid)
	return matched
}

// IsJSON 检查字符串是否为有效的 JSON 格式（粗略检查）。
//
// 仅去除首尾空白后检查是否以 { 和 } 或 [ 和 ] 成对首尾包裹，
// 不校验内部语法（如 "{invalid}" 也会返回 true）；
// 需严格校验时请使用 encoding/json 反序列化。
//
// 参数说明:
//   - jsonStr: 要验证的 JSON 字符串。
//
// 返回值:
//   - 如果形似 JSON 返回 true，否则返回 false。
func IsJSON(jsonStr string) bool {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return false
	}

	// 简单检查：JSON 应该以 { 或 [ 开头，以 } 或 ] 结尾
	return (strings.HasPrefix(jsonStr, "{") && strings.HasSuffix(jsonStr, "}")) ||
		(strings.HasPrefix(jsonStr, "[") && strings.HasSuffix(jsonStr, "]"))
}
