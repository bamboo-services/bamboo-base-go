package env

import (
	"database/sql/driver"
	"encoding/xml"
	"fmt"
	"os"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// Key 是基于 typed.String 派生的强类型环境变量键。
type Key typed.String

var (
	_ typed.Operable    = Key("")
	_ typed.Convertible = (*Key)(nil)
)

// Lookup 查询指定环境变量键的值。
// 返回原始字符串封装的 typed.String 以及是否存在布尔值。
// 如果环境变量未设置，返回 ("", false)；如果设置为空串，返回 ("", true)。
// 原始环境变量值不进行受控键名语法过滤。
func Lookup(key Key) (typed.String, bool) {
	val, ok := os.LookupEnv(string(key))
	if !ok {
		return "", false
	}
	return typed.String(val), true
}

// String 返回底层原生 string。
func (k Key) String() string {
	return string(k)
}

// TypedString 返回对应的 typed.String。
func (k Key) TypedString() typed.String {
	return typed.String(k)
}

// IsEmpty 报告键是否为空。
func (k Key) IsEmpty() bool {
	return len(k) == 0
}

// IsValid 校验受控键名语法是否合法。
func (k Key) IsValid() bool {
	return typed.String(k).IsValid()
}

// MarshalText 实现 encoding.TextMarshaler 接口。
func (k Key) MarshalText() ([]byte, error) {
	return k.TypedString().MarshalText()
}

// UnmarshalText 实现 encoding.TextUnmarshaler 接口。
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

// MarshalBinary 实现 encoding.BinaryMarshaler 接口。
func (k Key) MarshalBinary() ([]byte, error) {
	return k.TypedString().MarshalBinary()
}

// UnmarshalBinary 实现 encoding.BinaryUnmarshaler 接口。
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

// MarshalJSON 实现 json.Marshaler 接口。
func (k Key) MarshalJSON() ([]byte, error) {
	return k.TypedString().MarshalJSON()
}

// UnmarshalJSON 实现 json.Unmarshaler 接口。
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

// MarshalXML 实现 xml.Marshaler 接口。
func (k Key) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return k.TypedString().MarshalXML(e, start)
}

// UnmarshalXML 实现 xml.Unmarshaler 接口。
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

// MarshalXMLAttr 实现 xml.MarshalerAttr 接口。
func (k Key) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return k.TypedString().MarshalXMLAttr(name)
}

// UnmarshalXMLAttr 实现 xml.UnmarshalerAttr 接口。
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

// Value 实现 driver.Valuer 接口。
func (k Key) Value() (driver.Value, error) {
	return k.TypedString().Value()
}

// Scan 实现 sql.Scanner 接口。
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

// Set 实现 flag.Value 接口。
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
