package toolkit_test

import (
	"fmt"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/toolkit"
)

func ExamplePtr() {
	fmt.Println(*toolkit.Ptr(42))

	// 类型化的 nil 指针返回 nil，Val 对 nil 指针返回零值。
	var p *int
	fmt.Println(toolkit.Ptr(p) == nil, toolkit.Val(p))
	// Output:
	// 42
	// true 0
}

func ExampleVal() {
	s := "hi"
	var p *string
	fmt.Println(toolkit.Val(&s), toolkit.Val(p) == "")
	// Output: hi true
}

func ExampleContains() {
	fmt.Println(toolkit.Contains([]int{1, 2, 3}, 2), toolkit.Contains([]string{"a"}, "b"))
	// Output: true false
}

func ExampleToBool() {
	fmt.Println(toolkit.ToBool("yes", false), toolkit.ToBool("off", true), toolkit.ToBool("unknown", false))
	// Output: true false false
}

func ExampleSHA256() {
	fmt.Println(toolkit.SHA256("bamboo"))
	// Output: add1743eae5bbd8ea17b3d3b3531d0f3de072d10599edc0d8e107af90e52245a
}

func ExampleMD5() {
	fmt.Println(toolkit.MD5("bamboo"))
	// Output: 82960335036ff4ddc124b78af7777ee4
}

type exampleGreeter struct{}

func (exampleGreeter) Greet() string { return "hi" }

func ExampleGetMethodName() {
	fmt.Println(toolkit.GetMethodName(exampleGreeter{}.Greet))
	// Output: Greet
}

func ExampleRandomString() {
	// 输出内容随机，长度是确定的。
	fmt.Println(len(toolkit.RandomString(8)))
	// Output: 8
}

func ExampleGenerateKey() {
	key := toolkit.GenerateKey()
	fmt.Println(len(key), key[:3])
	// Output: 35 cs_
}

func ExampleGenerateLongKey() {
	key := toolkit.GenerateLongKey()
	fmt.Println(len(key), key[:3])
	// Output: 67 cs_
}

func ExampleVerifyKey() {
	fmt.Println(toolkit.VerifyKey(toolkit.GenerateKey()), toolkit.VerifyKey("invalid"))
	// Output: true false
}

func ExampleEncryptPassword() {
	hash, err := toolkit.EncryptPasswordString("s3cret!Pass")
	if err != nil {
		fmt.Println("加密失败:", err)
		return
	}
	fmt.Println(toolkit.IsPasswordValid("s3cret!Pass", hash))
	fmt.Println(toolkit.IsPasswordValid("wrong-pass", hash))
	// Output:
	// true
	// false
}

func ExampleParseInt() {
	fmt.Println(toolkit.ParseInt(42.9))
	fmt.Println(toolkit.ParseInt(" 42 "))
	fmt.Println(toolkit.ParseInt("42.5"))
	fmt.Println(toolkit.ParseInt(nil))
	// Output:
	// 42 true
	// 42 true
	// 0 false
	// 0 false
}

func ExampleParseBool() {
	fmt.Println(toolkit.ParseBool(true))
	fmt.Println(toolkit.ParseBool("YES"))
	fmt.Println(toolkit.ParseBool("unknown"))
	// Output:
	// true true
	// true true
	// false false
}

func ExampleParseString() {
	s, ok := toolkit.ParseString(42)
	fmt.Println(s, ok)
	s, ok = toolkit.ParseString(3.5)
	fmt.Println(s, ok)
	_, ok = toolkit.ParseString(struct{}{})
	fmt.Println(ok)
	// Output:
	// 42 true
	// 3.5 true
	// false
}

func ExampleIsBlank() {
	fmt.Println(toolkit.IsBlank("   "), toolkit.IsBlank("x"))
	// Output: true false
}

func ExampleTruncate() {
	fmt.Println(toolkit.Truncate("hello world", 5))
	// Output: hello
}

func ExampleTruncateWithSuffix() {
	fmt.Println(toolkit.TruncateWithSuffix("hello world", 8, "..."))
	// Output: hello...
}

func ExampleCamelToSnake() {
	fmt.Println(toolkit.CamelToSnake("helloWorld"))
	fmt.Println(toolkit.CamelToSnake("XMLParser"))
	// Output:
	// hello_world
	// x_m_l_parser
}

func ExampleSnakeToCamel() {
	fmt.Println(toolkit.SnakeToCamel("hello_world"))
	// Output: helloWorld
}

func ExampleMask() {
	fmt.Println(toolkit.Mask("13812345678", 3, 4, "*"))
	// Output: 138****5678
}

func ExampleRemoveSpaces() {
	fmt.Println(toolkit.RemoveSpaces("a b  c"))
	// Output: abc
}

func ExampleMustParse() {
	t := toolkit.MustParse(toolkit.TimeFormatDate, "2024-06-15")
	fmt.Println(t.Format(toolkit.TimeFormatDateTime))
	// Output: 2024-06-15 00:00:00
}

func ExampleStartOfDay() {
	t := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	fmt.Println(toolkit.StartOfDay(t).Format(toolkit.TimeFormatDateTime))
	fmt.Println(toolkit.EndOfDay(t).Format(toolkit.TimeFormatTime))
	// Output:
	// 2024-06-15 00:00:00
	// 23:59:59
}

func ExampleStartOfWeek() {
	// 2024-06-15 是周六，所在周从周一开始。
	t := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	fmt.Println(toolkit.StartOfWeek(t).Format(toolkit.TimeFormatDate))
	// Output: 2024-06-10
}

func ExampleDiffDays() {
	t1 := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	t2 := t1.AddDate(0, 0, -3)
	fmt.Println(toolkit.DiffDays(t1, t2))
	// Output: 3
}

func ExampleIsPhone() {
	fmt.Println(toolkit.IsPhone("13812345678"))
	fmt.Println(toolkit.IsPhone("12345678901"))
	// Output:
	// true
	// false
}

func ExampleIsUUID() {
	fmt.Println(toolkit.IsUUID("550e8400-e29b-41d4-a716-446655440000"))
	fmt.Println(toolkit.IsUUID("550e8400-e29b-71d4-a716-446655440000"))
	// Output:
	// true
	// false
}
