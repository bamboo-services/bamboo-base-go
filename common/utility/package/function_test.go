package pack

import (
	"runtime"
	"testing"
)

// 测试用辅助函数
func testFunctionForName() {}

type testStructForMethod struct{}

func (t testStructForMethod) testMethod()         {}
func (t *testStructForMethod) testPointerMethod() {}

// TestFunction_GetFunctionName 测试获取函数名
func TestFunction_GetFunctionName(t *testing.T) {
	f := Function{}
	tests := []struct {
		name     string
		fn       interface{}
		wantName string
		wantOk   bool
	}{
		{
			name:     "regular function",
			fn:       testFunctionForName,
			wantName: "github.com/bamboo-services/bamboo-base-go/common/utility/package/testFunctionForName",
			wantOk:   true,
		},
		{
			name:     "anonymous function",
			fn:       func() {},
			wantName: "", // 匿名函数名称可能包含特殊格式
			wantOk:   true,
		},
		{
			name:   "not a function (int)",
			fn:     42,
			wantOk: false,
		},
		{
			name:   "not a function (string)",
			fn:     "hello",
			wantOk: false,
		},
		{
			name:   "not a function (nil)",
			fn:     nil,
			wantOk: false,
		},
		{
			name:   "not a function (struct)",
			fn:     struct{}{},
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.GetFunctionName(tt.fn)

			if tt.wantOk {
				if got == "" {
					t.Error("GetFunctionName() returned empty string for valid function")
				}
				// 对于具名函数，验证名称包含函数名
				if tt.name == "regular function" && !containsFuncName(got, "testFunctionForName") {
					t.Errorf("GetFunctionName() = %v, should contain 'testFunctionForName'", got)
				}
			} else {
				if got != "" {
					t.Errorf("GetFunctionName() = %v, want empty string for non-function", got)
				}
			}
		})
	}
}

// TestFunction_GetFunctionName_Builtin 测试内置函数
func TestFunction_GetFunctionName_Builtin(t *testing.T) {
	f := Function{}

	// 内置函数不能直接作为参数，所以我们测试其他情况
	// 测试闭包
	closure := func(x int) int { return x * 2 }
	got := f.GetFunctionName(closure)
	if got == "" {
		t.Error("GetFunctionName() should return name for closure")
	}
	t.Logf("Closure name: %s", got)
}

// TestFunction_GetMethodName 测试获取方法名
func TestFunction_GetMethodName(t *testing.T) {
	f := Function{}
	s := testStructForMethod{}

	tests := []struct {
		name         string
		method       interface{}
		expectedName string
	}{
		{
			name:         "value receiver method",
			method:       s.testMethod,
			expectedName: "testMethod",
		},
		{
			name:         "pointer receiver method",
			method:       (&s).testPointerMethod,
			expectedName: "testPointerMethod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.GetMethodName(tt.method)
			if got != tt.expectedName {
				t.Errorf("GetMethodName() = %v, want %v", got, tt.expectedName)
			}
		})
	}
}

// TestFunction_GetMethodName_NilReceiver 测试 nil 接收者
// 注意：在 Go 中，对 nil 指针调用方法会 panic
// 这个测试验证的是 nil 指针方法引用仍然可以获取名称
func TestFunction_GetMethodName_NilReceiver(t *testing.T) {
	f := Function{}
	var s *testStructForMethod = nil

	// 方法表达式：获取方法名称不依赖于接收者是否为 nil
	// 注意：这会 panic，因为 nil 方法值会触发解引用
	// 所以我们跳过这个测试，只测试有效的方法引用
	t.Skip("Skipping nil receiver test - method value on nil causes panic")
	_ = f.GetMethodName(s.testMethod)
}

// TestFunction_GetMethodName_VerifyNoPackagePath 测试方法名不包含包路径
func TestFunction_GetMethodName_VerifyNoPackagePath(t *testing.T) {
	f := Function{}
	s := testStructForMethod{}

	got := f.GetMethodName(s.testMethod)

	// 确保返回的方法名不包含包路径或接收者类型
	if containsFuncName(got, ".") || containsFuncName(got, "/") {
		t.Errorf("GetMethodName() = %v, should not contain package path", got)
	}
}

// TestFunction_GetMethodName_CompareWithFunctionName 测试方法名和函数名的区别
func TestFunction_GetMethodName_CompareWithFunctionName(t *testing.T) {
	f := Function{}
	s := testStructForMethod{}

	// 获取完整函数名
	fullName := f.GetFunctionName(s.testMethod)
	methodName := f.GetMethodName(s.testMethod)

	t.Logf("Full name: %s", fullName)
	t.Logf("Method name: %s", methodName)

	// 方法名应该是完整名称的后缀
	if fullName != "" && methodName != "" {
		if !containsFuncName(fullName, methodName) {
			t.Errorf("Full name '%s' should contain method name '%s'", fullName, methodName)
		}
	}
}

// TestFunction_GetFunctionName_Closures 测试闭包函数
func TestFunction_GetFunctionName_Closures(t *testing.T) {
	f := Function{}

	// 测试多个闭包
	closure1 := func() {}
	closure2 := func() {}
	closure3 := func(x int) int { return x + 1 }

	name1 := f.GetFunctionName(closure1)
	name2 := f.GetFunctionName(closure2)
	name3 := f.GetFunctionName(closure3)

	if name1 == "" || name2 == "" || name3 == "" {
		t.Error("GetFunctionName() should return names for closures")
	}

	t.Logf("Closure1: %s", name1)
	t.Logf("Closure2: %s", name2)
	t.Logf("Closure3: %s", name3)

	// 闭包名称通常包含 "func" 字样
	// 注意：不同的 Go 版本可能有不同的命名格式
}

// TestFunction_GetMethodName_FunctionAsInput 测试将普通函数传给 GetMethodName
func TestFunction_GetMethodName_FunctionAsInput(t *testing.T) {
	f := Function{}

	// GetMethodName 不验证输入是否为方法
	// 它只是尝试获取函数名称
	got := f.GetMethodName(testFunctionForName)
	// 应该能获取到名称，但可能不是预期的格式
	t.Logf("Function passed to GetMethodName: %s", got)
}

// TestFunction_EdgeCases 测试边界情况
func TestFunction_EdgeCases(t *testing.T) {
	f := Function{}

	t.Run("nil input to GetFunctionName", func(t *testing.T) {
		got := f.GetFunctionName(nil)
		if got != "" {
			t.Errorf("GetFunctionName(nil) = %v, want empty string", got)
		}
	})

	t.Run("nil input to GetMethodName", func(t *testing.T) {
		// 这可能会 panic，所以我们使用 recover
		defer func() {
			if r := recover(); r != nil {
				t.Logf("GetMethodName(nil) panicked (expected): %v", r)
			}
		}()
		f.GetMethodName(nil)
	})
}

// TestFunction_GetFunctionName_PackageFunction 测试包级函数
func TestFunction_GetFunctionName_PackageFunction(t *testing.T) {
	f := Function{}

	// 测试标准库函数
	// 注意：标准库函数作为参数时可能需要特殊处理
	got := f.GetFunctionName(t.Log)
	if got == "" {
		// 某些情况下可能无法获取标准库函数名称
		t.Log("Could not get name for standard library function (may be expected)")
	} else {
		t.Logf("Standard library function name: %s", got)
	}
}

// 辅助函数：检查字符串是否包含子串
func containsFuncName(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// 编译时验证接口
var _ = runtime.FuncForPC
