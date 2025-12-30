package xEnv

import (
	"os"
	"strconv"
	"strings"
)

// GetEnv 获取环境变量值。
//
// 参数说明:
//   - key: 环境变量键名
//
// 返回值:
//   - value: 环境变量值
//   - exists: 是否存在该环境变量
func GetEnv(key string) (value string, exists bool) {
	return os.LookupEnv(key)
}

// GetEnvOrDefault 获取环境变量值，如果不存在则返回默认值。
//
// 参数说明:
//   - key: 环境变量键名
//   - defaultValue: 环境变量不存在时返回的默认值
//
// 返回值:
//   - 环境变量值，或默认值（当环境变量不存在时）
func GetEnvOrDefault(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetEnvInt 获取整数类型的环境变量值。
//
// 参数说明:
//   - key: 环境变量键名
//   - defaultValue: 环境变量不存在或解析失败时返回的默认值
//
// 返回值:
//   - 环境变量的整数值，或默认值
func GetEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvBool 获取布尔类型的环境变量值。
//
// 支持的真值: true, 1, yes, on
// 支持的假值: false, 0, no, off
//
// 参数说明:
//   - key: 环境变量键名
//   - defaultValue: 环境变量不存在或无法识别时返回的默认值
//
// 返回值:
//   - 环境变量的布尔值，或默认值
func GetEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		lower := strings.ToLower(value)
		switch lower {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultValue
}

// GetEnvFloat 获取浮点数类型的环境变量值。
//
// 参数说明:
//   - key: 环境变量键名
//   - defaultValue: 环境变量不存在或解析失败时返回的默认值
//
// 返回值:
//   - 环境变量的浮点数值，或默认值
func GetEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

// GetEnvInt64 获取 int64 类型的环境变量值。
//
// 参数说明:
//   - key: 环境变量键名
//   - defaultValue: 环境变量不存在或解析失败时返回的默认值
//
// 返回值:
//   - 环境变量的 int64 值，或默认值
func GetEnvInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}
