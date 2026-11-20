package typed_test

import (
	"errors"
	"flag"
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// printCode 打印统一错误的错误码与安全描述，演示调用方如何提取强类型错误。
func printCode(err error) {
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		fmt.Println(appErr.Code.Code, appErr.ErrorMessage)
		return
	}
	fmt.Println("非统一错误:", err)
}

// 演示受控键的文本编码与非法输入的错误处理。
func ExampleString_MarshalText() {
	data, err := typed.String("trace_id").MarshalText()
	fmt.Println(string(data), err == nil)

	_, err = typed.String("has space").MarshalText()
	printCode(err)

	// Output:
	// trace_id true
	// 40013 受控键名格式非法
}

// 演示原始文本解析为整数，含首尾空白与失败分支。
func ExampleString_Int() {
	port, err := typed.String(" 8080 ").Int()
	fmt.Println(port, err == nil)

	_, err = typed.String("eighty").Int()
	printCode(err)

	// Output:
	// 8080 true
	// 40013 解析为 int 失败
}

// 演示布尔词汇解析与非法输入报错。
func ExampleString_Bool() {
	enabled, err := typed.String("enabled").Bool()
	fmt.Println(enabled, err == nil)

	_, err = typed.String("maybe").Bool()
	printCode(err)

	// Output:
	// true true
	// 40013 布尔值格式非法
}

// 演示时长解析，输入可含首尾空白。
func ExampleString_Duration() {
	d, err := typed.String(" 1h30m ").Duration()
	fmt.Println(d, err == nil)

	// Output:
	// 1h30m0s true
}

// 演示 SQL 通道往返与 NULL 拒绝后的原值保留。
func ExampleString_Scan() {
	value := typed.String("order_no")

	sqlValue, err := value.Value()
	if err != nil {
		fmt.Println("value:", err)
		return
	}

	var scanned typed.String
	if err := scanned.Scan(sqlValue); err != nil {
		fmt.Println("scan:", err)
		return
	}
	fmt.Println(scanned, scanned.Equal(value))

	if err := scanned.Scan(nil); err != nil {
		printCode(err)
		fmt.Println(scanned)
	}

	// Output:
	// order_no true
	// 40015 不接受 SQL NULL 值
	// order_no
}

// 演示 *typed.String 作为 flag.Value 在命令行解析中使用。
func ExampleString_Set() {
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)

	var key typed.String
	fs.Var(&key, "key", "controlled key name")

	if err := fs.Parse([]string{"-key", "cache_slot"}); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(key)

	// Output:
	// cache_slot
}
