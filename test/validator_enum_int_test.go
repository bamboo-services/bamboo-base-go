package test

import (
	"testing"

	xVaild "github.com/bamboo-services/bamboo-base-go/validator"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// 自定义数值类型
type UserGender int8
type OrderStatus int
type ProductType int64

// 测试结构体
type EnumTestStruct struct {
	Gender      UserGender  `json:"gender" binding:"enum_int=0 1 2" label:"性别"`
	Status      OrderStatus `json:"status" binding:"enum_int=-1 0 1 2" label:"订单状态"`
	ProductType ProductType `json:"product_type" binding:"enum_int=100 200 300" label:"商品类型"`
	SimpleInt   int         `json:"simple_int" binding:"enum_int=1 2 3" label:"简单整数"`
	UnsignedInt uint8       `json:"unsigned_int" binding:"enum_int=0 1 2" label:"无符号整数"`
}

// 初始化验证器
func initValidator(t *testing.T) {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := xVaild.RegisterCustomValidators(v); err != nil {
			t.Fatalf("注册自定义验证器失败: %v", err)
		}
		if err := xVaild.RegisterTranslator(v); err != nil {
			t.Fatalf("注册翻译器失败: %v", err)
		}
	}
}

// Test_ValidateEnumInt_ValidValues 测试合法枚举值
func Test_ValidateEnumInt_ValidValues(t *testing.T) {
	initValidator(t)

	testCases := []struct {
		name     string
		input    EnumTestStruct
		wantPass bool
	}{
		{
			name: "所有字段都是合法枚举值",
			input: EnumTestStruct{
				Gender:      1,
				Status:      -1,
				ProductType: 100,
				SimpleInt:   2,
				UnsignedInt: 1,
			},
			wantPass: true,
		},
		{
			name: "边界值测试 - 最小值",
			input: EnumTestStruct{
				Gender:      0,
				Status:      -1,
				ProductType: 100,
				SimpleInt:   1,
				UnsignedInt: 0,
			},
			wantPass: true,
		},
		{
			name: "边界值测试 - 最大值",
			input: EnumTestStruct{
				Gender:      2,
				Status:      2,
				ProductType: 300,
				SimpleInt:   3,
				UnsignedInt: 2,
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

// Test_ValidateEnumInt_InvalidValues 测试非法枚举值
func Test_ValidateEnumInt_InvalidValues(t *testing.T) {
	initValidator(t)

	testCases := []struct {
		name          string
		input         EnumTestStruct
		expectedField string // 使用 label 而不是 json tag
	}{
		{
			name: "Gender 值超出枚举范围",
			input: EnumTestStruct{
				Gender:      99, // 不在 [0, 1, 2] 中
				Status:      0,
				ProductType: 100,
				SimpleInt:   1,
				UnsignedInt: 0,
			},
			expectedField: "性别", // 使用 label 的值
		},
		{
			name: "Status 值超出枚举范围",
			input: EnumTestStruct{
				Gender:      1,
				Status:      10, // 不在 [-1, 0, 1, 2] 中
				ProductType: 100,
				SimpleInt:   1,
				UnsignedInt: 0,
			},
			expectedField: "订单状态", // 使用 label 的值
		},
		{
			name: "ProductType 值超出枚举范围",
			input: EnumTestStruct{
				Gender:      1,
				Status:      0,
				ProductType: 999, // 不在 [100, 200, 300] 中
				SimpleInt:   1,
				UnsignedInt: 0,
			},
			expectedField: "商品类型", // 使用 label 的值
		},
		{
			name: "负数测试 - Gender 不允许负数",
			input: EnumTestStruct{
				Gender:      -1, // Gender 的枚举是 [0, 1, 2]，不包含 -1
				Status:      0,
				ProductType: 100,
				SimpleInt:   1,
				UnsignedInt: 0,
			},
			expectedField: "性别", // 使用 label 的值
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

			// 检查错误消息
			errors := xVaild.TranslateError(err)
			t.Logf("验证错误: %+v", errors)

			if _, exists := errors[tc.expectedField]; !exists {
				t.Errorf("期望字段 %s 有错误，但未找到", tc.expectedField)
			}
		})
	}
}

// Test_ValidateEnumInt_ErrorMessages 测试错误消息翻译
func Test_ValidateEnumInt_ErrorMessages(t *testing.T) {
	initValidator(t)

	invalidInput := EnumTestStruct{
		Gender:      99,
		Status:      0,
		ProductType: 100,
		SimpleInt:   1,
		UnsignedInt: 0,
	}

	v := binding.Validator.Engine().(*validator.Validate)
	err := v.Struct(invalidInput)

	if err == nil {
		t.Fatal("期望验证失败")
	}

	// 获取翻译后的错误消息
	errors := xVaild.TranslateError(err)
	t.Logf("中文错误消息: %+v", errors)

	// 使用 label 的值（"性别"）而不是 json tag（"gender"）
	genderError := errors["性别"]
	if genderError == "" {
		t.Error("性别字段应该有错误消息")
	}

	// 检查是否包含中文
	if !containsChinese(genderError) {
		t.Errorf("错误消息应该是中文，实际是: %s", genderError)
	}

	t.Logf("性别字段错误消息: %s", genderError)
}

// containsChinese 检查字符串是否包含中文字符
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fa5 {
			return true
		}
	}
	return false
}
