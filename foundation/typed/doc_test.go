package typed

import (
	"encoding/json"
	"fmt"
)

// 演示 typed.String 作为受控键名与原始文本解析载体的基础用法。
func ExampleString() {
	// 角色 A：作为受控机器键 / 词汇
	key := String("app_port")
	fmt.Printf("key valid: %t\n", key.IsValid())

	// 角色 B：作为原始环境变量等文本载体
	raw := String("  8080  ")
	port, err := raw.Int()
	if err != nil {
		panic(err)
	}
	fmt.Printf("parsed port: %d\n", port)

	// Output:
	// key valid: true
	// parsed port: 8080
}

// 演示受控标识符的语法校验。
func ExampleString_IsValid() {
	validKey := String("redis_host")
	invalidKey := String("redis__host")

	fmt.Printf("validKey: %t\n", validKey.IsValid())
	fmt.Printf("invalidKey: %t\n", invalidKey.IsValid())

	// Output:
	// validKey: true
	// invalidKey: false
}

// 演示 JSON 序列化与反序列化。
func ExampleString_json() {
	type Config struct {
		Key String `json:"key"`
	}

	cfg := Config{Key: String("user_id")}
	data, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("marshaled: %s\n", data)

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		panic(err)
	}
	fmt.Printf("decoded: %s\n", decoded.Key)

	// Output:
	// marshaled: {"key":"user_id"}
	// decoded: user_id
}
