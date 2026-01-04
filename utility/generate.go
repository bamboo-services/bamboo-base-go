package xUtil

import (
	"math/rand"
)

// GenerateRandomString 根据指定长度生成一个包含字母和数字的随机字符串。
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// GenerateRandomUpperString 根据指定长度生成一个仅包含大写字母和数字的随机字符串。
func GenerateRandomUpperString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// GenerateRandomNumberString 根据指定的长度生成由数字组成的随机字符串。
func GenerateRandomNumberString(length int) string {
	const charset = "0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}
