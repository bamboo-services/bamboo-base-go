package driver_test

import (
	"fmt"
	"time"

	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
)

func ExampleDefaultKeyEncoder() {
	var enc driver.DefaultKeyEncoder

	// 各种不同类型的键统一转换为稳定字符串
	fmt.Println("string:", enc.String("user:1001"))
	fmt.Println("bytes:", enc.String([]byte("session:token")))
	fmt.Println("int:", enc.String(42))
	fmt.Println("nil:", enc.String(nil))

	// Output:
	// string: user:1001
	// bytes: session:token
	// int: 42
	// nil:
}

func ExampleApplySet() {
	baseTTL := 10 * time.Minute

	// 1. 无选项，沿用基准 TTL
	cfg1 := driver.ApplySet(baseTTL, nil)
	fmt.Printf("cfg1: ttl=%v, nx=%v\n", cfg1.TTL, cfg1.NX)

	// 2. 覆盖 TTL 并设置 NX
	cfg2 := driver.ApplySet(baseTTL, []driver.SetOption{
		driver.WithTTL(5 * time.Minute),
		driver.WithNX(),
	})
	fmt.Printf("cfg2: ttl=%v, nx=%v\n", cfg2.TTL, cfg2.NX)

	// 3. 同时传入 NX 与 XX 时，NX 优先
	cfg3 := driver.ApplySet(baseTTL, []driver.SetOption{
		driver.WithNX(),
		driver.WithXX(),
	})
	fmt.Printf("cfg3: nx=%v, xx=%v\n", cfg3.NX, cfg3.XX)

	// Output:
	// cfg1: ttl=10m0s, nx=false
	// cfg2: ttl=5m0s, nx=true
	// cfg3: nx=true, xx=false
}

func ExampleJSONCodec() {
	var codec driver.JSONCodec

	type Profile struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}

	p := Profile{Name: "Konata", Role: "Admin"}
	data, err := codec.Marshal(p)
	if err != nil {
		panic(err)
	}

	var decoded Profile
	if err := codec.Unmarshal(data, &decoded); err != nil {
		panic(err)
	}

	fmt.Printf("%s (%s)\n", decoded.Name, decoded.Role)

	// Output:
	// Konata (Admin)
}
