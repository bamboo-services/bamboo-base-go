package apperr_test

import (
	"os/exec"
	"strings"
	"testing"
)

// compileCase 描述一个持久编译夹具的期望结果。
//
// 夹具源码保存在本包 testdata 下，随仓库持久存在；
// 负例必须在指定的强类型位置因类型不匹配而编译失败。
type compileCase struct {
	dir       string
	wantTexts []string
}

// strongTypeCompileCases 覆盖强类型失败出口的三种典型位置：返回、字段与回调。
var strongTypeCompileCases = []compileCase{
	{dir: "./testdata/compile/positive_return"},
	{dir: "./testdata/compile/positive_api"},
	{dir: "./testdata/compile/negative_join", wantTexts: []string{"cannot use", "*apperr.Error"}},
	{dir: "./testdata/compile/positive_field"},
	{dir: "./testdata/compile/positive_callback"},
	{dir: "./testdata/compile/negative_return", wantTexts: []string{"cannot use", "*apperr.Error"}},
	{dir: "./testdata/compile/negative_field", wantTexts: []string{"cannot use", "*apperr.Error"}},
	{dir: "./testdata/compile/negative_callback", wantTexts: []string{"cannot use", "FailureHandler"}},
}

// envFailureMarkers 标识编译失败来自依赖下载或环境问题，而不是类型不匹配。
var envFailureMarkers = []string{
	"dial tcp",
	"connection refused",
	"i/o timeout",
	"module lookup disabled",
	"cannot find module",
	"missing go.sum entry",
	"no required module provides",
	"proxy.golang.org",
	"toolchain not available",
	"cannot find GOROOT directory",
	"malformed module path",
}

func TestStrongTypeCompileContract(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Fatalf("环境缺少 go 工具链，无法执行编译契约测试: %v", err)
	}

	// 先编译正例：正例失败说明工具链或依赖不可用，必须与环境失败区分开。
	for _, item := range strongTypeCompileCases {
		if len(item.wantTexts) > 0 {
			continue
		}
		output, err := compileFixture(item.dir)
		if err != nil {
			t.Fatalf("正例 %s 编译失败（工具链或依赖环境问题）: %v\n%s", item.dir, err, output)
		}
	}

	// 再编译负例：必须在强类型位置因类型不匹配失败。
	for _, item := range strongTypeCompileCases {
		if len(item.wantTexts) == 0 {
			continue
		}
		output, err := compileFixture(item.dir)
		if err == nil {
			t.Fatalf("负例 %s 预期因类型不匹配编译失败，但编译通过", item.dir)
		}
		if marker := findEnvFailure(output); marker != "" {
			t.Fatalf("负例 %s 的编译失败来自环境或依赖问题（%s），不能当作类型不匹配通过:\n%s", item.dir, marker, output)
		}
		for _, want := range item.wantTexts {
			if !strings.Contains(output, want) {
				t.Fatalf("负例 %s 未在指定类型位置因类型不匹配失败（缺少 %q）:\n%s", item.dir, want, output)
			}
		}
	}
}

// compileFixture 在当前模块上下文中编译指定夹具目录，返回合并输出与错误。
func compileFixture(dir string) (string, error) {
	cmd := exec.Command("go", "build", dir)
	var builder strings.Builder
	cmd.Stdout = &builder
	cmd.Stderr = &builder
	err := cmd.Run()
	return builder.String(), err
}

// findEnvFailure 返回命中的环境问题特征；没有命中时返回空字符串。
func findEnvFailure(output string) string {
	for _, marker := range envFailureMarkers {
		if strings.Contains(output, marker) {
			return marker
		}
	}
	return ""
}
