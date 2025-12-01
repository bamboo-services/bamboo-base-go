package xUtil

import (
	"os"
	"strconv"
	"strings"
)

// EnvPrefix 环境变量前缀常量。
//
// 所有配置相关的环境变量都需要使用此前缀，以避免与系统环境变量冲突。
const EnvPrefix = "BAMBOO_"

// GetEnv 获取带前缀的环境变量值。
//
// 该函数会自动为键名添加 BAMBOO_ 前缀，并将键名转换为大写。
//
// 参数说明:
//   - key: 环境变量键名（不含前缀），如 "XLF_DEBUG"
//
// 返回值:
//   - value: 环境变量值
//   - exists: 是否存在该环境变量
func GetEnv(key string) (value string, exists bool) {
	fullKey := EnvPrefix + strings.ToUpper(key)
	return os.LookupEnv(fullKey)
}

// GetEnvOrDefault 获取环境变量值，如果不存在则返回默认值。
//
// 该函数会自动为键名添加 BAMBOO_ 前缀，并将键名转换为大写。
//
// 参数说明:
//   - key: 环境变量键名（不含前缀），如 "XLF_DEBUG"
//   - defaultValue: 环境变量不存在时返回的默认值
//
// 返回值:
//   - 环境变量值，或默认值（当环境变量不存在时）
func GetEnvOrDefault(key string, defaultValue string) string {
	fullKey := EnvPrefix + strings.ToUpper(key)
	if value, exists := os.LookupEnv(fullKey); exists {
		return value
	}
	return defaultValue
}

// YamlPathToEnvKey 将 YAML 路径转换为环境变量键名。
//
// 该函数将点号 `.` 替换为下划线 `_`，并将结果转换为大写。
// 注意：返回的键名不包含 BAMBOO_ 前缀。
//
// 参数说明:
//   - yamlPath: YAML 配置路径，如 "database.host"
//
// 返回值:
//   - 环境变量键名（不含前缀），如 "DATABASE_HOST"
func YamlPathToEnvKey(yamlPath string) string {
	return strings.ToUpper(strings.ReplaceAll(yamlPath, ".", "_"))
}

// ApplyEnvOverrides 将环境变量覆盖应用到配置映射。
//
// 该函数递归遍历配置结构，检查对应的环境变量是否存在，
// 如果存在则使用环境变量值覆盖原有配置（包括空字符串）。
//
// 环境变量命名规则：BAMBOO_{PATH}，其中 PATH 为配置路径的大写形式，点号替换为下划线。
// 例如：xlf.debug -> BAMBOO_XLF_DEBUG
//
// 参数说明:
//   - config: 配置映射 map[string]interface{}
//   - prefix: 当前路径前缀（用于递归，初始调用时传入空字符串）
//
// 返回值:
//   - 应用环境变量覆盖后的配置映射
func ApplyEnvOverrides(config map[string]interface{}, prefix string) map[string]interface{} {
	for key, value := range config {
		// 构建当前配置项的完整路径
		currentPath := key
		if prefix != "" {
			currentPath = prefix + "_" + key
		}

		// 将路径转换为环境变量键名（大写）
		envKey := strings.ToUpper(currentPath)

		switch v := value.(type) {
		case map[string]interface{}:
			// 递归处理嵌套结构
			config[key] = ApplyEnvOverrides(v, currentPath)
		default:
			// 检查是否存在对应的环境变量
			if envValue, exists := GetEnv(envKey); exists {
				// 根据原始值类型进行类型转换
				config[key] = convertEnvValue(envValue, value)
			}
		}
	}

	return config
}

// convertEnvValue 根据原始值类型将环境变量字符串转换为对应类型。
//
// 该函数根据原始配置值的类型，将环境变量字符串转换为相同类型。
// 如果转换失败，则返回该类型的零值。
//
// 参数说明:
//   - envValue: 环境变量字符串值
//   - originalValue: 原始配置值（用于类型推断）
//
// 返回值:
//   - 转换后的值，类型与原始值相同
func convertEnvValue(envValue string, originalValue interface{}) interface{} {
	if originalValue == nil {
		return envValue
	}

	switch originalValue.(type) {
	case bool:
		return ToBool(envValue, false)

	case int:
		if intVal, err := strconv.Atoi(envValue); err == nil {
			return intVal
		}
		return 0

	case int64:
		if intVal, err := strconv.ParseInt(envValue, 10, 64); err == nil {
			return intVal
		}
		return int64(0)

	case float64:
		// YAML 解析数字时通常使用 float64
		if floatVal, err := strconv.ParseFloat(envValue, 64); err == nil {
			return floatVal
		}
		return float64(0)

	case string:
		return envValue

	default:
		return envValue
	}
}
