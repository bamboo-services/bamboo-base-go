package ctxkey_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type exampleExposer struct {
	m map[ctxkey.Key]any
}

func (e exampleExposer) Expose() map[ctxkey.Key]any {
	return e.m
}

// ExampleExposer 演示插件通过 Expose 交付组件句柄映射。
func ExampleExposer() {
	var exp ctxkey.Exposer = exampleExposer{
		m: map[ctxkey.Key]any{
			ctxkey.DatabaseKey: "db-instance",
		},
	}
	res := exp.Expose()
	fmt.Println(res[ctxkey.DatabaseKey])
	// Output:
	// db-instance
}

// ExampleKey 演示从文本解码上下文键并校验。
func ExampleKey() {
	var k ctxkey.Key
	if err := k.UnmarshalText([]byte("context_database")); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(k.String(), k.IsEmpty(), k.IsValid())
	// Output:
	// context_database false true
}

// ExampleGetRequestKey 演示读取请求唯一标识及其缺省回退。
func ExampleGetRequestKey() {
	ctx := context.WithValue(context.Background(), ctxkey.RequestKey, "req-42")
	fmt.Println(ctxkey.GetRequestKey(ctx))
	fmt.Println(ctxkey.GetRequestKey(context.Background()) == "")
	// Output:
	// req-42
	// true
}

// ExampleIsDebugMode 演示调试模式的读取。
func ExampleIsDebugMode() {
	if err := os.Setenv("APP_DEBUG", "true"); err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = os.Unsetenv("APP_DEBUG") }()
	fmt.Println(ctxkey.IsDebugMode())
	// Output:
	// true
}

// ExampleCalcOverheadTime 演示非调试模式下耗时计算稳定返回 0。
func ExampleCalcOverheadTime() {
	if err := os.Setenv("APP_DEBUG", "false"); err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = os.Unsetenv("APP_DEBUG") }()

	ctx := context.WithValue(context.Background(), ctxkey.UserStartTimeKey, time.Now())
	fmt.Println(ctxkey.CalcOverheadTime(ctx))
	// Output:
	// 0
}
