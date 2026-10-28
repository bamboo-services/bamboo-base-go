package typed

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrNilReceiver 表示对 nil 指针接收器调用解码方法时返回的错误。
// [String.UnmarshalText]、[String.UnmarshalBinary]、[String.UnmarshalJSON]、
// [String.UnmarshalXML]、[String.UnmarshalXMLAttr]、[String.Scan] 与
// [String.Set] 在接收器为 nil 时返回该值，调用方可通过 errors.Is 识别。
var ErrNilReceiver = errors.New("typed.String: nil receiver")

// MarshalText 实现 encoding.TextMarshaler，输出底层字符串内容。
// 输出前先执行 [String.IsValid] 校验，不合法时返回错误。
func (s String) MarshalText() ([]byte, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("typed.String: invalid key %q", s)
	}
	return []byte(s), nil
}

// UnmarshalText 实现 encoding.TextUnmarshaler，校验候选文本并写回。
// 接收器为 nil 时返回 [ErrNilReceiver]；仅当候选值通过 [String.IsValid]
// 校验后才写回，任何失败均不改变接收器原值。
func (s *String) UnmarshalText(text []byte) error {
	if s == nil {
		return ErrNilReceiver
	}
	candidate := String(text)
	if !candidate.IsValid() {
		return fmt.Errorf("typed.String: invalid key %q", text)
	}
	*s = candidate
	return nil
}

// MarshalBinary 实现 encoding.BinaryMarshaler，输出底层字符串字节，
// 供 gob 与 Redis 客户端等二进制通道复用。
// 输出前先执行 [String.IsValid] 校验，不合法时返回错误。
func (s String) MarshalBinary() ([]byte, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("typed.String: invalid key %q", s)
	}
	return []byte(s), nil
}

// UnmarshalBinary 实现 encoding.BinaryUnmarshaler，校验候选字节并写回。
// 接收器为 nil 时返回 [ErrNilReceiver]；仅当候选值通过 [String.IsValid]
// 校验后才写回，任何失败均不改变接收器原值。
func (s *String) UnmarshalBinary(data []byte) error {
	if s == nil {
		return ErrNilReceiver
	}
	candidate := String(data)
	if !candidate.IsValid() {
		return fmt.Errorf("typed.String: invalid key %q", data)
	}
	*s = candidate
	return nil
}

// MarshalJSON 实现 json.Marshaler，编码为 JSON 字符串字面量。
// 编码前先执行 [String.IsValid] 校验，不合法时返回错误且无部分输出。
func (s String) MarshalJSON() ([]byte, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("typed.String: invalid key %q", s)
	}
	return json.Marshal(string(s))
}

// UnmarshalJSON 实现 json.Unmarshaler，从 JSON 字符串解码并写回。
// 严格拒绝 JSON null 与非字符串类型；接收器为 nil 时返回 [ErrNilReceiver]；
// 仅当候选值通过 [String.IsValid] 校验后才写回，任何失败均不改变接收器原值。
func (s *String) UnmarshalJSON(raw []byte) error {
	if s == nil {
		return ErrNilReceiver
	}
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return fmt.Errorf("typed.String: cannot unmarshal JSON null")
	}

	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return fmt.Errorf("typed.String: unmarshal JSON: %w", err)
	}

	candidate := String(str)
	if !candidate.IsValid() {
		return fmt.Errorf("typed.String: invalid key %q", candidate)
	}
	*s = candidate
	return nil
}

// MarshalXML 实现 xml.Marshaler，将底层字符串编码为 XML 元素的字符内容。
// 编码前先执行 [String.IsValid] 校验，不合法时返回错误。
func (s String) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if !s.IsValid() {
		return fmt.Errorf("typed.String: invalid key %q", s)
	}
	return e.EncodeElement(string(s), start)
}

// UnmarshalXML 实现 xml.Unmarshaler，从 XML 元素的字符内容解码并写回。
// 拒绝空元素内容与嵌套子元素，元素内容按原文作为候选值、不做空白修剪；
// 接收器为 nil 时返回 [ErrNilReceiver]；仅当候选值通过 [String.IsValid]
// 校验后才写回，任何失败均不改变接收器原值。
func (s *String) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if s == nil {
		return ErrNilReceiver
	}

	var buf strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("typed.String: unmarshal XML: %w", err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			buf.Write(t)
		case xml.StartElement:
			return fmt.Errorf("typed.String: nested XML elements not allowed: <%s>", t.Name.Local)
		case xml.EndElement:
			if t.Name == start.Name {
				candidate := String(buf.String())
				if !candidate.IsValid() {
					return fmt.Errorf("typed.String: invalid XML element content %q", buf.String())
				}
				*s = candidate
				return nil
			}
		}
	}

	candidate := String(buf.String())
	if !candidate.IsValid() {
		return fmt.Errorf("typed.String: invalid XML element content %q", buf.String())
	}
	*s = candidate
	return nil
}

// MarshalXMLAttr 实现 xml.MarshalerAttr，将底层字符串编码为 XML 属性。
// 编码前先执行 [String.IsValid] 校验，不合法时返回零值 xml.Attr 与错误。
func (s String) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if !s.IsValid() {
		return xml.Attr{}, fmt.Errorf("typed.String: invalid key %q", s)
	}
	return xml.Attr{
		Name:  name,
		Value: string(s),
	}, nil
}

// UnmarshalXMLAttr 实现 xml.UnmarshalerAttr，从 XML 属性值解码并写回。
// 接收器为 nil 时返回 [ErrNilReceiver]；仅当候选值通过 [String.IsValid]
// 校验后才写回，任何失败均不改变接收器原值。
func (s *String) UnmarshalXMLAttr(attr xml.Attr) error {
	if s == nil {
		return ErrNilReceiver
	}
	candidate := String(attr.Value)
	if !candidate.IsValid() {
		return fmt.Errorf("typed.String: invalid XML attribute value %q", attr.Value)
	}
	*s = candidate
	return nil
}

// Value 实现 driver.Valuer，将底层字符串输出为 SQL 字符串值。
// 输出前先执行 [String.IsValid] 校验，不合法时返回错误。
func (s String) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("typed.String: invalid key %q", s)
	}
	return string(s), nil
}

// Scan 实现 sql.Scanner，从 SQL 列值解码并写回。
// 仅接受 string 与 []byte，拒绝 SQL NULL 与其它类型；
// 接收器为 nil 时返回 [ErrNilReceiver]；仅当候选值通过 [String.IsValid]
// 校验后才写回，任何失败均不改变接收器原值。
func (s *String) Scan(src any) error {
	if s == nil {
		return ErrNilReceiver
	}
	if src == nil {
		return fmt.Errorf("typed.String: cannot scan SQL NULL")
	}

	var candidate String
	switch v := src.(type) {
	case string:
		candidate = String(v)
	case []byte:
		candidate = String(v)
	default:
		return fmt.Errorf("typed.String: cannot scan type %T into typed.String (must be string or []byte)", src)
	}

	if !candidate.IsValid() {
		return fmt.Errorf("typed.String: invalid key %q", candidate)
	}
	*s = candidate
	return nil
}

// Set 实现 flag.Value，解析命令行参数值并写回。
// 配合 [String.String] 使 *String 完整满足 [flag.Value]；
// 接收器为 nil 时返回 [ErrNilReceiver]；仅当候选值通过 [String.IsValid]
// 校验后才写回，任何失败均不改变接收器原值。
func (s *String) Set(raw string) error {
	if s == nil {
		return ErrNilReceiver
	}
	candidate := String(raw)
	if !candidate.IsValid() {
		return fmt.Errorf("typed.String: invalid flag value %q", raw)
	}
	*s = candidate
	return nil
}
