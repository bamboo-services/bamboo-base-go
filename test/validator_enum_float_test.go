package test

import (
	"testing"

	xVaild "github.com/bamboo-services/bamboo-base-go/validator"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// 自定义浮点数类型
type Rating float64
type Temperature float32

// 测试结构体
type EnumFloatTestStruct struct {
	Rating      Rating      `json:"rating" binding:"enum_float=0.5 1.0 1.5 2.0 2.5 3.0" label:"评分"`
	Temperature Temperature `json:"temperature" binding:"enum_float=36.5 37.0 37.5 38.0" label:"温度"`
	Discount    float64     `json:"discount" binding:"enum_float=0.1 0.2 0.5" label:"折扣"`
}

// Test_ValidateEnumFloat_ValidValues 测试合法枚举值
func Test_ValidateEnumFloat_ValidValues(t *testing.T) {
	initValidator(t)

	testCases := []struct {
		name     string
		input    EnumFloatTestStruct
		wantPass bool
	}{
		{
			name: "所有字段都是合法枚举值",
			input: EnumFloatTestStruct{
				Rating:      1.5,
				Temperature: 37.0,
				Discount:    0.2,
			},
			wantPass: true,
		},
		{
			name: "边界值测试 - 最小值",
			input: EnumFloatTestStruct{
				Rating:      0.5,
				Temperature: 36.5,
				Discount:    0.1,
			},
			wantPass: true,
		},
		{
			name: "边界值测试 - 最大值",
			input: EnumFloatTestStruct{
				Rating:      3.0,
				Temperature: 38.0,
				Discount:    0.5,
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

// Test_ValidateEnumFloat_InvalidValues 测试非法枚举值
func Test_ValidateEnumFloat_InvalidValues(t *testing.T) {
	initValidator(t)

	testCases := []struct {
		name          string
		input         EnumFloatTestStruct
		expectedField string
	}{
		{
			name: "Rating 值超出枚举范围",
			input: EnumFloatTestStruct{
				Rating:      4.0,
				Temperature: 37.0,
				Discount:    0.2,
			},
			expectedField: "评分",
		},
		{
			name: "Temperature 值超出枚举范围",
			input: EnumFloatTestStruct{
				Rating:      1.5,
				Temperature: 39.0,
				Discount:    0.2,
			},
			expectedField: "温度",
		},
		{
			name: "Discount 值超出枚举范围",
			input: EnumFloatTestStruct{
				Rating:      1.5,
				Temperature: 37.0,
				Discount:    0.3,
			},
			expectedField: "折扣",
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

// Test_ValidateEnumFloat_ErrorMessages 测试错误消息翻译
func Test_ValidateEnumFloat_ErrorMessages(t *testing.T) {
	initValidator(t)

	invalidInput := EnumFloatTestStruct{
		Rating:      99.9,
		Temperature: 37.0,
		Discount:    0.2,
	}

	v := binding.Validator.Engine().(*validator.Validate)
	err := v.Struct(invalidInput)

	if err == nil {
		t.Fatal("期望验证失败")
	}

	errors := xVaild.TranslateError(err)
	t.Logf("中文错误消息: %+v", errors)

	ratingError := errors["评分"]
	if ratingError == "" {
		t.Error("评分字段应该有错误消息")
	}

	if !containsChinese(ratingError) {
		t.Errorf("错误消息应该是中文，实际是: %s", ratingError)
	}

	t.Logf("评分字段错误消息: %s", ratingError)
}
