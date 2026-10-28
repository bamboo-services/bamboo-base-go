package env_test

import (
	"fmt"
	"os"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
)

// 演示环境变量读取：区分未设置与空串，并链式解析为原生类型。
func ExampleLookup() {
	const demoKey = "EXAMPLE_BAMBOO_DEMO_PORT"
	_ = os.Setenv(demoKey, "8080")
	defer func() { _ = os.Unsetenv(demoKey) }()

	raw, ok := env.Lookup(demoKey)
	fmt.Println("set:", ok, raw.IsEmpty())
	port, err := raw.Int()
	fmt.Println("port:", port, err)

	_, ok = env.Lookup("EXAMPLE_BAMBOO_MISSING")
	fmt.Println("missing:", ok)

	// Output:
	// set: true false
	// port: 8080 <nil>
	// missing: false
}

// 演示预置目录常量键与文本通道解码。
func ExampleKey() {
	fmt.Println(env.Port.String(), env.Port.IsValid())

	var decoded env.Key
	if err := decoded.UnmarshalText([]byte("APP_NAME")); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(decoded, decoded == env.AppName)

	// Output:
	// PORT true
	// APP_NAME true
}
