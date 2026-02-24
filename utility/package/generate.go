package pack

import (
	"math/rand"
)

// Generate 生成工具结构体，提供各类随机字符串生成方法。
//
// 使用方式：
//
//	xUtil.Generate().RandomString(32)
//	xUtil.Generate().RandomUpperString(16)
type Generate struct{}

// RandomString 根据指定长度生成一个包含字母和数字的随机字符串。
func (Generate) RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// RandomUpperString 根据指定长度生成一个仅包含大写字母和数字的随机字符串。
func (Generate) RandomUpperString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}

// RandomNumberString 根据指定的长度生成由数字组成的随机字符串。
func (Generate) RandomNumberString(length int) string {
	const charset = "0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}
	return result
}
