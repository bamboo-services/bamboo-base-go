package typed

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
)

// ErrNilReceiver 表示对 nil 指针接收器调用解码方法时返回的错误。
// [String.UnmarshalText]、[String.UnmarshalBinary]、[String.UnmarshalJSON]、
// [String.UnmarshalXML]、[String.UnmarshalXMLAttr]、[String.Scan] 与
// [String.Set] 在接收器为 nil 时返回以该哨兵为根因的 [*apperr.Error]
// （错误码 50019），调用方可通过 errors.Is 识别。
var ErrNilReceiver = errors.New("typed.String: nil receiver")

// 本文件的十三个标准接口方法保留标准库要求的固定签名，
// 内部先调用强类型核心：核心返回 nil 时适配层返回真正的 nil，
// 核心返回 [*apperr.Error] 时适配层原样交出该强类型错误。
//
// 分类只依据实际校验分支显式给出：
//
//   - 受控键名语法不合法：InvalidFormat（40013）；
//   - 输入源类型不合法（JSON null 与非字符串、XML 嵌套子元素、SQL NULL 与非 string/[]byte）：InvalidType（40015）；
//   - 接收器为 nil：InvalidReceiver（50019）。
//
// 任何失败都不写回接收器原值。

// MarshalText 实现 encoding.TextMarshaler，输出底层字符串内容。
// 输出前先执行 [String.IsValid] 校验，不合法时返回错误（40013）且无部分输出。
func (s String) MarshalText() ([]byte, error) {
	out, err := s.marshalText()
	if err != nil {
		return nil, err
	}
	return out, nil
}

// marshalText 是 [String.MarshalText] 的强类型核心。
func (s String) marshalText() ([]byte, *apperr.Error) {
	if !s.IsValid() {
		return nil, errInvalidFormat("受控键名格式非法")
	}
	return []byte(s), nil
}

// UnmarshalText 实现 encoding.TextUnmarshaler，校验候选文本并写回。
// 接收器为 nil 时返回 50019（根因为 [ErrNilReceiver]）；仅当候选值通过
// [String.IsValid] 校验后才写回，任何失败均不改变接收器原值。
func (s *String) UnmarshalText(text []byte) error {
	candidate, err := s.unmarshalText(text)
	if err != nil {
		return err
	}
	*s = candidate
	return nil
}

// unmarshalText 是 [String.UnmarshalText] 的强类型核心。
func (s *String) unmarshalText(text []byte) (String, *apperr.Error) {
	if s == nil {
		return "", errNilReceiver()
	}
	candidate := String(text)
	if !candidate.IsValid() {
		return "", errInvalidFormat("受控键名格式非法")
	}
	return candidate, nil
}

// MarshalBinary 实现 encoding.BinaryMarshaler，输出底层字符串字节，
// 供 gob 与 Redis 客户端等二进制通道复用。
// 输出前先执行 [String.IsValid] 校验，不合法时返回错误（40013）。
func (s String) MarshalBinary() ([]byte, error) {
	out, err := s.marshalBinary()
	if err != nil {
		return nil, err
	}
	return out, nil
}

// marshalBinary 是 [String.MarshalBinary] 的强类型核心。
func (s String) marshalBinary() ([]byte, *apperr.Error) {
	if !s.IsValid() {
		return nil, errInvalidFormat("受控键名格式非法")
	}
	return []byte(s), nil
}

// UnmarshalBinary 实现 encoding.BinaryUnmarshaler，校验候选字节并写回。
// 接收器为 nil 时返回 50019（根因为 [ErrNilReceiver]）；仅当候选值通过
// [String.IsValid] 校验后才写回，任何失败均不改变接收器原值。
func (s *String) UnmarshalBinary(data []byte) error {
	candidate, err := s.unmarshalBinary(data)
	if err != nil {
		return err
	}
	*s = candidate
	return nil
}

// unmarshalBinary 是 [String.UnmarshalBinary] 的强类型核心。
func (s *String) unmarshalBinary(data []byte) (String, *apperr.Error) {
	if s == nil {
		return "", errNilReceiver()
	}
	candidate := String(data)
	if !candidate.IsValid() {
		return "", errInvalidFormat("受控键名格式非法")
	}
	return candidate, nil
}

// MarshalJSON 实现 json.Marshaler，编码为 JSON 字符串字面量。
// 编码前先执行 [String.IsValid] 校验，不合法时返回错误（40013）且无部分输出。
func (s String) MarshalJSON() ([]byte, error) {
	out, err := s.marshalJSON()
	if err != nil {
		return nil, err
	}
	return out, nil
}

// marshalJSON 是 [String.MarshalJSON] 的强类型核心。
func (s String) marshalJSON() ([]byte, *apperr.Error) {
	if !s.IsValid() {
		return nil, errInvalidFormat("受控键名格式非法")
	}
	out, err := json.Marshal(string(s))
	if err != nil {
		return nil, apperr.From(apperr.FrameworkInternalError, "JSON 编码失败", err)
	}
	return out, nil
}

// UnmarshalJSON 实现 json.Unmarshaler，从 JSON 字符串解码并写回。
// 严格拒绝 JSON null 与非字符串类型（40015）；接收器为 nil 时返回 50019；
// 仅当候选值通过 [String.IsValid] 校验后才写回，任何失败均不改变接收器原值。
func (s *String) UnmarshalJSON(raw []byte) error {
	candidate, err := s.unmarshalJSON(raw)
	if err != nil {
		return err
	}
	*s = candidate
	return nil
}

// unmarshalJSON 是 [String.UnmarshalJSON] 的强类型核心。
func (s *String) unmarshalJSON(raw []byte) (String, *apperr.Error) {
	if s == nil {
		return "", errNilReceiver()
	}
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return "", errInvalidType("不接受 JSON null")
	}

	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return "", valueerr.From(valueerr.KindInvalidType, "JSON 内容不是字符串", err)
	}

	candidate := String(str)
	if !candidate.IsValid() {
		return "", errInvalidFormat("受控键名格式非法")
	}
	return candidate, nil
}

// MarshalXML 实现 xml.Marshaler，将底层字符串编码为 XML 元素的字符内容。
// 编码前先执行 [String.IsValid] 校验，不合法时返回错误（40013）。
func (s String) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if err := s.marshalXML(e, start); err != nil {
		return err
	}
	return nil
}

// marshalXML 是 [String.MarshalXML] 的强类型核心。
func (s String) marshalXML(e *xml.Encoder, start xml.StartElement) *apperr.Error {
	if !s.IsValid() {
		return errInvalidFormat("受控键名格式非法")
	}
	if err := e.EncodeElement(string(s), start); err != nil {
		return apperr.From(apperr.FrameworkInternalError, "XML 编码失败", err)
	}
	return nil
}

// UnmarshalXML 实现 xml.Unmarshaler，从 XML 元素的字符内容解码并写回。
// 拒绝空元素内容与嵌套子元素（40015）；接收器为 nil 时返回 50019；
// 仅当候选值通过 [String.IsValid] 校验后才写回，任何失败均不改变接收器原值。
func (s *String) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	candidate, err := s.unmarshalXML(d, start)
	if err != nil {
		return err
	}
	*s = candidate
	return nil
}

// unmarshalXML 是 [String.UnmarshalXML] 的强类型核心。
func (s *String) unmarshalXML(d *xml.Decoder, start xml.StartElement) (String, *apperr.Error) {
	if s == nil {
		return "", errNilReceiver()
	}

	var buf strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", valueerr.From(valueerr.KindInvalidFormat, "XML 内容解析失败", err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			buf.Write(t)
		case xml.StartElement:
			return "", errInvalidType("不接受嵌套的 XML 子元素")
		case xml.EndElement:
			if t.Name == start.Name {
				candidate := String(buf.String())
				if !candidate.IsValid() {
					return "", errInvalidFormat("受控键名格式非法")
				}
				return candidate, nil
			}
		}
	}

	candidate := String(buf.String())
	if !candidate.IsValid() {
		return "", errInvalidFormat("受控键名格式非法")
	}
	return candidate, nil
}

// MarshalXMLAttr 实现 xml.MarshalerAttr，将底层字符串编码为 XML 属性。
// 编码前先执行 [String.IsValid] 校验，不合法时返回零值 xml.Attr 与错误（40013）。
func (s String) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	attr, err := s.marshalXMLAttr(name)
	if err != nil {
		return xml.Attr{}, err
	}
	return attr, nil
}

// marshalXMLAttr 是 [String.MarshalXMLAttr] 的强类型核心。
func (s String) marshalXMLAttr(name xml.Name) (xml.Attr, *apperr.Error) {
	if !s.IsValid() {
		return xml.Attr{}, errInvalidFormat("受控键名格式非法")
	}
	return xml.Attr{
		Name:  name,
		Value: string(s),
	}, nil
}

// UnmarshalXMLAttr 实现 xml.UnmarshalerAttr，从 XML 属性值解码并写回。
// 接收器为 nil 时返回 50019；仅当候选值通过 [String.IsValid] 校验后才写回，
// 任何失败均不改变接收器原值。
func (s *String) UnmarshalXMLAttr(attr xml.Attr) error {
	candidate, err := s.unmarshalXMLAttr(attr)
	if err != nil {
		return err
	}
	*s = candidate
	return nil
}

// unmarshalXMLAttr 是 [String.UnmarshalXMLAttr] 的强类型核心。
func (s *String) unmarshalXMLAttr(attr xml.Attr) (String, *apperr.Error) {
	if s == nil {
		return "", errNilReceiver()
	}
	candidate := String(attr.Value)
	if !candidate.IsValid() {
		return "", errInvalidFormat("受控键名格式非法")
	}
	return candidate, nil
}

// Value 实现 driver.Valuer，将底层字符串输出为 SQL 字符串值。
// 输出前先执行 [String.IsValid] 校验，不合法时返回错误（40013）。
func (s String) Value() (driver.Value, error) {
	out, err := s.value()
	if err != nil {
		return nil, err
	}
	return out, nil
}

// value 是 [String.Value] 的强类型核心。
func (s String) value() (driver.Value, *apperr.Error) {
	if !s.IsValid() {
		return nil, errInvalidFormat("受控键名格式非法")
	}
	return string(s), nil
}

// Scan 实现 sql.Scanner，从 SQL 列值解码并写回。
// 仅接受 string 与 []byte，拒绝 SQL NULL 与其它类型（40015）；
// 接收器为 nil 时返回 50019；仅当候选值通过 [String.IsValid] 校验后才写回，
// 任何失败均不改变接收器原值。
func (s *String) Scan(src any) error {
	candidate, err := s.scan(src)
	if err != nil {
		return err
	}
	*s = candidate
	return nil
}

// scan 是 [String.Scan] 的强类型核心。
func (s *String) scan(src any) (String, *apperr.Error) {
	if s == nil {
		return "", errNilReceiver()
	}
	if src == nil {
		return "", errInvalidType("不接受 SQL NULL 值")
	}

	var candidate String
	switch v := src.(type) {
	case string:
		candidate = String(v)
	case []byte:
		candidate = String(v)
	default:
		return "", errInvalidType("SQL 列值类型不受支持，仅接受 string 或 []byte")
	}

	if !candidate.IsValid() {
		return "", errInvalidFormat("受控键名格式非法")
	}
	return candidate, nil
}

// Set 实现 flag.Value，解析命令行参数值并写回。
// 配合 [String.String] 使 *String 完整满足 [flag.Value]；
// 接收器为 nil 时返回 50019；仅当候选值通过 [String.IsValid] 校验后才写回，
// 任何失败均不改变接收器原值。
func (s *String) Set(raw string) error {
	candidate, err := s.set(raw)
	if err != nil {
		return err
	}
	*s = candidate
	return nil
}

// set 是 [String.Set] 的强类型核心。
func (s *String) set(raw string) (String, *apperr.Error) {
	if s == nil {
		return "", errNilReceiver()
	}
	candidate := String(raw)
	if !candidate.IsValid() {
		return "", errInvalidFormat("受控键名格式非法")
	}
	return candidate, nil
}
