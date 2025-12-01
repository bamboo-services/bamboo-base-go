package test

import (
	"os"
	"testing"

	xUtil "github.com/bamboo-services/bamboo-base-go/utility"
)

// Test_YamlPathToEnvKey 测试 YAML 路径到环境变量键名的转换。
func Test_YamlPathToEnvKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"xlf.debug", "XLF_DEBUG"},
		{"database.host", "DATABASE_HOST"},
		{"nosql.database", "NOSQL_DATABASE"},
		{"xlf.port", "XLF_PORT"},
		{"database.prefix", "DATABASE_PREFIX"},
	}

	for _, tc := range tests {
		result := xUtil.YamlPathToEnvKey(tc.input)
		if result != tc.expected {
			t.Errorf("YamlPathToEnvKey(%s) = %s; want %s", tc.input, result, tc.expected)
		}
	}
}

// Test_GetEnv 测试环境变量获取功能。
func Test_GetEnv(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("BAMBOO_TEST_KEY", "test_value")
	defer os.Unsetenv("BAMBOO_TEST_KEY")

	// 测试存在的环境变量
	value, exists := xUtil.GetEnv("TEST_KEY")
	if !exists {
		t.Error("GetEnv 应该返回 exists=true 对于已设置的环境变量")
	}
	if value != "test_value" {
		t.Errorf("GetEnv = %s; want test_value", value)
	}

	// 测试不存在的环境变量
	_, exists = xUtil.GetEnv("NOT_EXISTS_KEY")
	if exists {
		t.Error("GetEnv 应该返回 exists=false 对于未设置的环境变量")
	}
}

// Test_GetEnvOrDefault 测试获取环境变量或默认值。
func Test_GetEnvOrDefault(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("BAMBOO_EXISTING_KEY", "existing_value")
	defer os.Unsetenv("BAMBOO_EXISTING_KEY")

	// 测试存在的环境变量
	value := xUtil.GetEnvOrDefault("EXISTING_KEY", "default")
	if value != "existing_value" {
		t.Errorf("GetEnvOrDefault = %s; want existing_value", value)
	}

	// 测试不存在的环境变量
	value = xUtil.GetEnvOrDefault("NOT_EXISTS", "default_value")
	if value != "default_value" {
		t.Errorf("GetEnvOrDefault = %s; want default_value", value)
	}
}

// Test_ApplyEnvOverrides 测试环境变量覆盖配置功能。
func Test_ApplyEnvOverrides(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("BAMBOO_XLF_DEBUG", "true")
	os.Setenv("BAMBOO_DATABASE_PORT", "3307")
	os.Setenv("BAMBOO_DATABASE_HOST", "") // 测试空字符串覆盖
	defer func() {
		os.Unsetenv("BAMBOO_XLF_DEBUG")
		os.Unsetenv("BAMBOO_DATABASE_PORT")
		os.Unsetenv("BAMBOO_DATABASE_HOST")
	}()

	config := map[string]interface{}{
		"xlf": map[string]interface{}{
			"debug": false,
			"host":  "0.0.0.0",
		},
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
	}

	result := xUtil.ApplyEnvOverrides(config, "")

	// 验证 xlf.debug 被覆盖为 true
	xlfConfig := result["xlf"].(map[string]interface{})
	if xlfConfig["debug"] != true {
		t.Errorf("xlf.debug 应该被覆盖为 true，实际值: %v", xlfConfig["debug"])
	}

	// 验证 xlf.host 保持不变（未设置环境变量）
	if xlfConfig["host"] != "0.0.0.0" {
		t.Errorf("xlf.host 应该保持 0.0.0.0，实际值: %v", xlfConfig["host"])
	}

	// 验证 database.port 被覆盖
	dbConfig := result["database"].(map[string]interface{})
	// 注意：原始值是 int，环境变量覆盖后应该也是 int
	if port, ok := dbConfig["port"].(int); ok {
		if port != 3307 {
			t.Errorf("database.port 应该被覆盖为 3307，实际值: %d", port)
		}
	} else {
		t.Errorf("database.port 类型应该是 int，实际类型: %T", dbConfig["port"])
	}

	// 验证 database.host 被空字符串覆盖
	if dbConfig["host"] != "" {
		t.Errorf("database.host 应该被覆盖为空字符串，实际值: %v", dbConfig["host"])
	}
}

// Test_ApplyEnvOverrides_BoolConversion 测试布尔值转换。
func Test_ApplyEnvOverrides_BoolConversion(t *testing.T) {
	tests := []struct {
		envValue string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
		{"yes", true},
		{"no", false},
		{"on", true},
		{"off", false},
	}

	for _, tc := range tests {
		os.Setenv("BAMBOO_TEST_BOOL", tc.envValue)

		config := map[string]interface{}{
			"test": map[string]interface{}{
				"bool": false,
			},
		}

		result := xUtil.ApplyEnvOverrides(config, "")
		testConfig := result["test"].(map[string]interface{})

		if testConfig["bool"] != tc.expected {
			t.Errorf("布尔值转换失败: %s -> %v; want %v", tc.envValue, testConfig["bool"], tc.expected)
		}

		os.Unsetenv("BAMBOO_TEST_BOOL")
	}
}

// Test_ApplyEnvOverrides_FloatConversion 测试浮点数转换（YAML 默认解析数字为 float64）。
func Test_ApplyEnvOverrides_FloatConversion(t *testing.T) {
	os.Setenv("BAMBOO_CONFIG_VALUE", "3.14")
	defer os.Unsetenv("BAMBOO_CONFIG_VALUE")

	config := map[string]interface{}{
		"config": map[string]interface{}{
			"value": float64(1.0),
		},
	}

	result := xUtil.ApplyEnvOverrides(config, "")
	configMap := result["config"].(map[string]interface{})

	if val, ok := configMap["value"].(float64); ok {
		if val != 3.14 {
			t.Errorf("浮点数转换失败: want 3.14, got %v", val)
		}
	} else {
		t.Errorf("浮点数类型转换失败，实际类型: %T", configMap["value"])
	}
}

// Test_ApplyEnvOverrides_NestedConfig 测试嵌套配置覆盖。
func Test_ApplyEnvOverrides_NestedConfig(t *testing.T) {
	os.Setenv("BAMBOO_LEVEL1_LEVEL2_VALUE", "nested_value")
	defer os.Unsetenv("BAMBOO_LEVEL1_LEVEL2_VALUE")

	config := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"value": "original",
			},
		},
	}

	result := xUtil.ApplyEnvOverrides(config, "")
	level1 := result["level1"].(map[string]interface{})
	level2 := level1["level2"].(map[string]interface{})

	if level2["value"] != "nested_value" {
		t.Errorf("嵌套配置覆盖失败: want nested_value, got %v", level2["value"])
	}
}
