package env

import (
	"database/sql/driver"
	"encoding/xml"
	"fmt"
	"os"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// Key 是基于 [typed.String] 派生的强类型环境变量键。
//
// Key 与 [typed.String] 是不同的命名类型，相互赋值需显式转换；
// 零值为空字符串，[Key.IsValid] 返回 false。
// 校验与编解码行为与 [typed.String] 对应方法一致：编码前先校验键名，
// 解码仅在候选值校验成功后写回，失败不改变原值。
type Key typed.String

var (
	_ typed.Operable    = Key("")
	_ typed.Convertible = (*Key)(nil)
)

// Lookup 读取指定环境变量键的当前值，返回封装为 [typed.String] 的
// 原始文本与是否存在的布尔值。
//
//   - 未设置：返回 ("", false)；
//   - 设置为空串：返回 ("", true)，两种状态严格区分。
//
// 返回值完全保留用户输入，不做受控键名语法过滤，
// 可直接经 [typed.String.Int]、[typed.String.Bool]、
// [typed.String.Duration] 等方法继续解析。
func Lookup(key Key) (typed.String, bool) {
	val, ok := os.LookupEnv(string(key))
	if !ok {
		return "", false
	}
	return typed.String(val), true
}

// String 返回键的底层原生 string，使 Key 满足 fmt.Stringer。
func (k Key) String() string {
	return string(k)
}

// TypedString 返回键对应的 [typed.String]，供其方法族统一复用。
func (k Key) TypedString() typed.String {
	return typed.String(k)
}

// IsEmpty 报告键的字节长度是否为 0。
func (k Key) IsEmpty() bool {
	return len(k) == 0
}

// IsValid 校验键名是否满足受控标识符语法，
// 规则与 [typed.String.IsValid] 完全一致。
func (k Key) IsValid() bool {
	return typed.String(k).IsValid()
}

// MarshalText 实现 encoding.TextMarshaler，
// 行为与 [typed.String.MarshalText] 一致：输出前先校验键名，
// 不合法时返回错误。
func (k Key) MarshalText() ([]byte, error) {
	return k.TypedString().MarshalText()
}

// UnmarshalText 实现 encoding.TextUnmarshaler。
// 接收器为 nil 时返回 "env.Key: nil receiver" 错误；
// 行为与 [typed.String.UnmarshalText] 一致，
// 仅当候选值校验成功后写回，失败不改变原值。
func (k *Key) UnmarshalText(text []byte) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalText(text); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalBinary 实现 encoding.BinaryMarshaler，
// 行为与 [typed.String.MarshalBinary] 一致：输出前先校验键名，
// 不合法时返回错误。
func (k Key) MarshalBinary() ([]byte, error) {
	return k.TypedString().MarshalBinary()
}

// UnmarshalBinary 实现 encoding.BinaryUnmarshaler。
// 接收器为 nil 时返回 "env.Key: nil receiver" 错误；
// 行为与 [typed.String.UnmarshalBinary] 一致，
// 仅当候选值校验成功后写回，失败不改变原值。
func (k *Key) UnmarshalBinary(data []byte) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalBinary(data); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalJSON 实现 json.Marshaler，
// 行为与 [typed.String.MarshalJSON] 一致：编码为 JSON 字符串，
// 输出前先校验键名，不合法时返回错误。
func (k Key) MarshalJSON() ([]byte, error) {
	return k.TypedString().MarshalJSON()
}

// UnmarshalJSON 实现 json.Unmarshaler。
// 接收器为 nil 时返回 "env.Key: nil receiver" 错误；
// 行为与 [typed.String.UnmarshalJSON] 一致：拒绝 JSON null 与非字符串，
// 仅当候选值校验成功后写回，失败不改变原值。
func (k *Key) UnmarshalJSON(raw []byte) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalJSON(raw); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalXML 实现 xml.Marshaler，
// 行为与 [typed.String.MarshalXML] 一致：编码为 XML 元素字符内容，
// 输出前先校验键名，不合法时返回错误。
func (k Key) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return k.TypedString().MarshalXML(e, start)
}

// UnmarshalXML 实现 xml.Unmarshaler。
// 接收器为 nil 时返回 "env.Key: nil receiver" 错误；
// 行为与 [typed.String.UnmarshalXML] 一致：拒绝空内容与嵌套子元素，
// 仅当候选值校验成功后写回，失败不改变原值。
func (k *Key) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalXML(d, start); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalXMLAttr 实现 xml.MarshalerAttr，
// 行为与 [typed.String.MarshalXMLAttr] 一致：编码为 XML 属性，
// 输出前先校验键名，不合法时返回错误。
func (k Key) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return k.TypedString().MarshalXMLAttr(name)
}

// UnmarshalXMLAttr 实现 xml.UnmarshalerAttr。
// 接收器为 nil 时返回 "env.Key: nil receiver" 错误；
// 行为与 [typed.String.UnmarshalXMLAttr] 一致，
// 仅当候选值校验成功后写回，失败不改变原值。
func (k *Key) UnmarshalXMLAttr(attr xml.Attr) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalXMLAttr(attr); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// Value 实现 driver.Valuer，
// 行为与 [typed.String.Value] 一致：输出为 SQL 字符串，
// 输出前先校验键名，不合法时返回错误。
func (k Key) Value() (driver.Value, error) {
	return k.TypedString().Value()
}

// Scan 实现 sql.Scanner。
// 接收器为 nil 时返回 "env.Key: nil receiver" 错误；
// 行为与 [typed.String.Scan] 一致：仅接受 string 与 []byte、拒绝 NULL，
// 仅当候选值校验成功后写回，失败不改变原值。
func (k *Key) Scan(src any) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.Scan(src); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// Set 实现 flag.Value，解析命令行参数值并写回。
// 接收器为 nil 时返回 "env.Key: nil receiver" 错误；
// 行为与 [typed.String.Set] 一致，配合 [Key.String] 使 *Key
// 完整满足 [flag.Value]；仅当候选值校验成功后写回，失败不改变原值。
func (k *Key) Set(raw string) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.Set(raw); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}
