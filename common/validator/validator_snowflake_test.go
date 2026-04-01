package xVaild

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// 自定义字符串类型
type SnowflakeID string

// 测试结构体
type SnowflakeTestStruct struct {
	ID       string      `json:"id" binding:"snowflake" label:"ID"`
	ParentID *string     `json:"parent_id,omitempty" binding:"omitempty,snowflake" label:"父分类ID"`
	GeneID   SnowflakeID `json:"gene_id" binding:"snowflake" label:"基因ID"`
}

// Test_ValidateSnowflake_ValidValues 测试合法的 Snowflake ID
func Test_ValidateSnowflake_ValidValues(t *testing.T) {
	initValidator(t)

	testCases := []struct {
		name     string
		input    SnowflakeTestStruct
		wantPass bool
	}{
		{
			name: "标准 Snowflake ID",
			input: SnowflakeTestStruct{
				ID:     "1234567890123456789",
				GeneID: "9876543210987654321",
			},
			wantPass: true,
		},
		{
			name: "最短 ID（1位）",
			input: SnowflakeTestStruct{
				ID:     "0",
				GeneID: "1",
			},
			wantPass: true,
		},
		{
			name: "20位边界值",
			input: SnowflakeTestStruct{
				ID:     "9223372036854775807", // int64 最大值
				GeneID: "9999999999999999999",
			},
			wantPass: true,
		},
		{
			name: "带前导零",
			input: SnowflakeTestStruct{
				ID:     "0012345",
				GeneID: "0000001",
			},
			wantPass: true,
		},
		{
			name: "omitempty 允许空指针",
			input: SnowflakeTestStruct{
				ID:       "123456789",
				ParentID: nil,
				GeneID:   "987654321",
			},
			wantPass: true,
		},
		{
			name: "omitempty 允许空字符串指针",
			input: SnowflakeTestStruct{
				ID:       "123456789",
				ParentID: strPtr(""),
				GeneID:   "987654321",
			},
			wantPass: false, // 空字符串不通过 snowflake 验证
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

// Test_ValidateSnowflake_InvalidValues 测试非法的 Snowflake ID
func Test_ValidateSnowflake_InvalidValues(t *testing.T) {
	initValidator(t)

	testCases := []struct {
		name          string
		input         SnowflakeTestStruct
		expectedField string
	}{
		{
			name: "包含字母",
			input: SnowflakeTestStruct{
				ID:     "123abc456",
				GeneID: "987654321",
			},
			expectedField: "ID",
		},
		{
			name: "包含特殊字符",
			input: SnowflakeTestStruct{
				ID:     "123-456-789",
				GeneID: "987654321",
			},
			expectedField: "ID",
		},
		{
			name: "负数",
			input: SnowflakeTestStruct{
				ID:     "-123456789",
				GeneID: "987654321",
			},
			expectedField: "ID",
		},
		{
			name: "小数",
			input: SnowflakeTestStruct{
				ID:     "123456.789",
				GeneID: "987654321",
			},
			expectedField: "ID",
		},
		{
			name: "超过20位",
			input: SnowflakeTestStruct{
				ID:     "123456789012345678901",
				GeneID: "987654321",
			},
			expectedField: "ID",
		},
		{
			name: "空字符串",
			input: SnowflakeTestStruct{
				ID:     "",
				GeneID: "987654321",
			},
			expectedField: "ID",
		},
		{
			name: "空格",
			input: SnowflakeTestStruct{
				ID:     "123 456",
				GeneID: "987654321",
			},
			expectedField: "ID",
		},
		{
			name: "自定义类型包含字母",
			input: SnowflakeTestStruct{
				ID:     "123456789",
				GeneID: "abc123",
			},
			expectedField: "基因ID",
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

			errors := TranslateError(err)
			t.Logf("验证错误: %+v", errors)

			if _, exists := errors[tc.expectedField]; !exists {
				t.Errorf("期望字段 %s 有错误，但未找到", tc.expectedField)
			}
		})
	}
}

// Test_ValidateSnowflake_ErrorMessages 测试错误消息翻译
func Test_ValidateSnowflake_ErrorMessages(t *testing.T) {
	initValidator(t)

	invalidInput := SnowflakeTestStruct{
		ID:     "invalid_id_123",
		GeneID: "987654321",
	}

	v := binding.Validator.Engine().(*validator.Validate)
	err := v.Struct(invalidInput)

	if err == nil {
		t.Fatal("期望验证失败")
	}

	errors := TranslateError(err)
	t.Logf("中文错误消息: %+v", errors)

	idError := errors["ID"]
	if idError == "" {
		t.Error("ID 字段应该有错误消息")
	}

	if !containsChinese(idError) {
		t.Errorf("错误消息应该是中文，实际是: %s", idError)
	}

	t.Logf("ID 字段错误消息: %s", idError)
}

// strPtr 返回字符串指针辅助函数
func strPtr(s string) *string {
	return &s
}
