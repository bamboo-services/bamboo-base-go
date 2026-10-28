package toolkit

import (
	"math/rand"
)

// RandomString 根据指定长度生成包含大小写字母和数字的随机字符串。
//
// 字符集为 a-z、A-Z、0-9 共 62 个字符；length 为 0 或负数时返回空字符串。
// 结果基于 math/rand 全局源，适用于验证码、临时编号等非安全场景，
// 不应用于密码、令牌等密码学安全需求（此类场景请使用 crypto/rand）。
//
// 参数说明:
//   - length: 目标字符串长度（字符数）。
//
// 返回值:
//   - 指定长度的随机字符串。
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// RandomUpperString 根据指定长度生成仅包含大写字母和数字的随机字符串。
//
// 字符集为 A-Z、0-9 共 36 个字符；length 为 0 或负数时返回空字符串。
// 与 [RandomString] 相同，基于 math/rand 全局源，仅适用于
// 非安全场景。
//
// 参数说明:
//   - length: 目标字符串长度（字符数）。
//
// 返回值:
//   - 指定长度的随机字符串。
func RandomUpperString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// RandomNumberString 根据指定长度生成由数字组成的随机字符串。
//
// 字符集为 0-9 共 10 个字符；length 为 0 或负数时返回空字符串。
// 与 [RandomString] 相同，基于 math/rand 全局源，仅适用于
// 非安全场景。
//
// 参数说明:
//   - length: 目标字符串长度（字符数）。
//
// 返回值:
//   - 指定长度的数字随机字符串。
func RandomNumberString(length int) string {
	const charset = "0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}
