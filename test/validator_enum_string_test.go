package test

import (
	"testing"

	xVaild "github.com/bamboo-services/bamboo-base-go/validator"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// 自定义字符串类型
type UserRole string
type TaskStatus string

// 测试结构体
type EnumStringTestStruct struct {
	Role   UserRole   `json:"role" binding:"enum_string=admin user guest" label:"角色"`
	Status TaskStatus `json:"status" binding:"enum_string=active pending inactive" label:"状态"`
	Type   string     `json:"type" binding:"enum_string=A B C" label:"类型"`
}

// Test_ValidateEnumString_ValidValues 测试合法枚举值
func Test_ValidateEnumString_ValidValues(t *testing.T) {
	initValidator(t)

	testCases := []struct {
		name     string
		input    EnumStringTestStruct
		wantPass bool
	}{
		{
			name: "所有字段都是合法枚举值",
			input: EnumStringTestStruct{
				Role:   "admin",
				Status: "active",
				Type:   "A",
			},
			wantPass: true,
		},
		{
			name: "边界值测试",
			input: EnumStringTestStruct{
				Role:   "guest",
				Status: "inactive",
				Type:   "C",
			},
			wantPass: true,
		},
	}

	v := binding.Validator.Engine().(*validator.Validate)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Struct(tc.input)
			if tc.wantPass && err != nil {
				t.Errorf("期望验证通过，但失败了: %v", err)
			}
			if !tc.wantPass && err == nil {
				t.Errorf("期望验证失败，但通过了")
			}
		})
	}
}

// Test_ValidateEnumString_InvalidValues 测试非法枚举值
func Test_ValidateEnumString_InvalidValues(t *testing.T) {
	initValidator(t)

	testCases := []struct {
		name          string
		input         EnumStringTestStruct
		expectedField string
	}{
		{
			name: "Role 值超出枚举范围",
			input: EnumStringTestStruct{
				Role:   "superadmin",
				Status: "active",
				Type:   "A",
			},
			expectedField: "角色",
		},
		{
			name: "Status 值超出枚举范围",
			input: EnumStringTestStruct{
				Role:   "admin",
				Status: "deleted",
				Type:   "A",
			},
			expectedField: "状态",
		},
		{
			name: "大小写敏感测试",
			input: EnumStringTestStruct{
				Role:   "ADMIN",
				Status: "active",
				Type:   "A",
			},
			expectedField: "角色",
		},
	}

	v := binding.Validator.Engine().(*validator.Validate)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Struct(tc.input)
			if err == nil {
				t.Errorf("期望验证失败，但通过了")
				return
			}

			errors := xVaild.TranslateError(err)
			t.Logf("验证错误: %+v", errors)

			if _, exists := errors[tc.expectedField]; !exists {
				t.Errorf("期望字段 %s 有错误，但未找到", tc.expectedField)
			}
		})
	}
}

// Test_ValidateEnumString_ErrorMessages 测试错误消息翻译
func Test_ValidateEnumString_ErrorMessages(t *testing.T) {
	initValidator(t)

	invalidInput := EnumStringTestStruct{
		Role:   "invalid_role",
		Status: "active",
		Type:   "A",
	}

	v := binding.Validator.Engine().(*validator.Validate)
	err := v.Struct(invalidInput)

	if err == nil {
		t.Fatal("期望验证失败")
	}

	errors := xVaild.TranslateError(err)
	t.Logf("中文错误消息: %+v", errors)

	roleError := errors["角色"]
	if roleError == "" {
		t.Error("角色字段应该有错误消息")
	}

	if !containsChinese(roleError) {
		t.Errorf("错误消息应该是中文，实际是: %s", roleError)
	}

	t.Logf("角色字段错误消息: %s", roleError)
}
