package test

import (
	"os"
	"testing"

	xUtil "github.com/bamboo-services/bamboo-base-go/env"
)

// Test_GetEnv 测试环境变量获取功能。
func Test_GetEnv(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

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
	os.Setenv("EXISTING_KEY", "existing_value")
	defer os.Unsetenv("EXISTING_KEY")

	// 测试存在的环境变量
	value := xUtil.GetEnvString("EXISTING_KEY", "default")
	if value != "existing_value" {
		t.Errorf("GetEnvString = %s; want existing_value", value)
	}

	// 测试不存在的环境变量
	value = xUtil.GetEnvString("NOT_EXISTS", "default_value")
	if value != "default_value" {
		t.Errorf("GetEnvString = %s; want default_value", value)
	}
}

// Test_GetEnvInt 测试整数环境变量获取。
func Test_GetEnvInt(t *testing.T) {
	// 测试有效整数
	os.Setenv("INT_VALUE", "42")
	defer os.Unsetenv("INT_VALUE")

	value := xUtil.GetEnvInt("INT_VALUE", 0)
	if value != 42 {
		t.Errorf("GetEnvInt = %d; want 42", value)
	}

	// 测试无效整数（应返回默认值）
	os.Setenv("INVALID_INT", "not_a_number")
	defer os.Unsetenv("INVALID_INT")

	value = xUtil.GetEnvInt("INVALID_INT", 100)
	if value != 100 {
		t.Errorf("GetEnvInt 对于无效输入应返回默认值 100, got %d", value)
	}

	// 测试不存在的环境变量
	value = xUtil.GetEnvInt("NOT_EXISTS_INT", 999)
	if value != 999 {
		t.Errorf("GetEnvInt = %d; want 999", value)
	}
}

// Test_GetEnvBool 测试布尔环境变量获取。
func Test_GetEnvBool(t *testing.T) {
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
		{"TRUE", true},
		{"FALSE", false},
		{"Yes", true},
		{"No", false},
	}

	for _, tc := range tests {
		os.Setenv("TEST_BOOL", tc.envValue)
		result := xUtil.GetEnvBool("TEST_BOOL", !tc.expected)
		if result != tc.expected {
			t.Errorf("GetEnvBool(%s) = %v; want %v", tc.envValue, result, tc.expected)
		}
		os.Unsetenv("TEST_BOOL")
	}

	// 测试不存在的环境变量返回默认值
	result := xUtil.GetEnvBool("NOT_EXISTS_BOOL", true)
	if result != true {
		t.Error("GetEnvBool 应该对不存在的键返回默认值 true")
	}

	result = xUtil.GetEnvBool("NOT_EXISTS_BOOL", false)
	if result != false {
		t.Error("GetEnvBool 应该对不存在的键返回默认值 false")
	}
}

// Test_GetEnvFloat 测试浮点数环境变量获取。
func Test_GetEnvFloat(t *testing.T) {
	// 测试有效浮点数
	os.Setenv("FLOAT_VALUE", "3.14")
	defer os.Unsetenv("FLOAT_VALUE")

	value := xUtil.GetEnvFloat("FLOAT_VALUE", 0.0)
	if value != 3.14 {
		t.Errorf("GetEnvFloat = %f; want 3.14", value)
	}

	// 测试整数值（应能解析为浮点数）
	os.Setenv("INT_AS_FLOAT", "42")
	defer os.Unsetenv("INT_AS_FLOAT")

	value = xUtil.GetEnvFloat("INT_AS_FLOAT", 0.0)
	if value != 42.0 {
		t.Errorf("GetEnvFloat = %f; want 42.0", value)
	}

	// 测试无效浮点数
	os.Setenv("INVALID_FLOAT", "not_a_float")
	defer os.Unsetenv("INVALID_FLOAT")

	value = xUtil.GetEnvFloat("INVALID_FLOAT", 1.5)
	if value != 1.5 {
		t.Errorf("GetEnvFloat 对于无效输入应返回默认值 1.5, got %f", value)
	}

	// 测试不存在的环境变量
	value = xUtil.GetEnvFloat("NOT_EXISTS_FLOAT", 9.99)
	if value != 9.99 {
		t.Errorf("GetEnvFloat = %f; want 9.99", value)
	}
}
