package xUtil

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Hash 接收一个字符串并返回其 SHA-256 哈希值的十六进制字符串表示。
//
// 参数说明:
//   - data: 要计算哈希值的字符串
//
// 返回值:
//   - SHA-256 哈希值（64位小写十六进制字符串）
func Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// HashBytes 接收一个字节切片并返回其 SHA-256 哈希值的十六进制字符串表示。
//
// 参数说明:
//   - data: 要计算哈希值的字节切片
//
// 返回值:
//   - SHA-256 哈希值（64位小写十六进制字符串）
func HashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// MD5Hash 计算字符串的 MD5 哈希值。
//
// 参数说明:
//   - str: 要计算哈希的字符串
//
// 返回值:
//   - MD5 哈希值（32位小写十六进制字符串）
func MD5Hash(str string) string {
	hash := md5.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

// MD5HashBytes 计算字节切片的 MD5 哈希值。
//
// 参数说明:
//   - data: 要计算哈希的字节切片
//
// 返回值:
//   - MD5 哈希值（32位小写十六进制字符串）
func MD5HashBytes(data []byte) string {
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}
