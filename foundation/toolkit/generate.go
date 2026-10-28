package toolkit

import (
	"math/rand"
)

// RandomString 根据指定长度生成一个包含字母和数字的随机字符串。
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// RandomUpperString 根据指定长度生成一个仅包含大写字母和数字的随机字符串。
func RandomUpperString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// RandomNumberString 根据指定的长度生成由数字组成的随机字符串。
func RandomNumberString(length int) string {
	const charset = "0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}
