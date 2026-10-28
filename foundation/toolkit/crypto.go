package toolkit

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// SHA256 接收一个字符串并返回其 SHA-256 哈希的十六进制表示。
//
// 相同输入始终得到相同结果，适合用于内容指纹、完整性比对等
// 安全摘要场景。
//
// 参数说明:
//   - data: 要计算哈希值的字符串。
//
// 返回值:
//   - SHA-256 哈希值，固定为 64 个小写十六进制字符。
func SHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SHA256Bytes 接收一个字节切片并返回其 SHA-256 哈希的十六进制表示。
//
// 与 [SHA256] 语义一致，区别仅在输入为字节切片，适合二进制内容的
// 摘要计算。
//
// 参数说明:
//   - data: 要计算哈希值的字节切片。
//
// 返回值:
//   - SHA-256 哈希值，固定为 64 个小写十六进制字符。
func SHA256Bytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// MD5 计算字符串的 MD5 哈希值。
//
// MD5 已不具备抗碰撞安全性，仅适用于缓存键、指纹比对等
// 非安全场景；安全摘要请改用 [SHA256]。
//
// 参数说明:
//   - str: 要计算哈希的字符串。
//
// 返回值:
//   - MD5 哈希值，固定为 32 个小写十六进制字符。
func MD5(str string) string {
	hash := md5.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

// MD5Bytes 计算字节切片的 MD5 哈希值。
//
// 与 [MD5] 语义一致，区别仅在输入为字节切片；同样仅适用于
// 非安全场景，安全摘要请改用 [SHA256Bytes]。
//
// 参数说明:
//   - data: 要计算哈希的字节切片。
//
// 返回值:
//   - MD5 哈希值，固定为 32 个小写十六进制字符。
func MD5Bytes(data []byte) string {
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}
