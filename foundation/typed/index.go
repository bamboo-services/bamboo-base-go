package typed

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
)

// MaxKeyBytes 是受控标识符键名的最大字节长度上限。
const MaxKeyBytes = 128

// String 是以 Go 原生 string 为基底的强类型字符串原语。
// 既可作为受控键名/常量词汇的基底，也可作为环境变量等原始文本的解析载体。
type String string

// Operable 约束所有基于 typed.String 派生的领域词汇必须具备的基础操作能力。
type Operable interface {
	fmt.Stringer
	TypedString() String
	IsEmpty() bool
}

// Convertible 约束支持标准文本、JSON、XML（含属性）、二进制、SQL 与 CLI Flag 双向转换的完整契约。
type Convertible interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	json.Marshaler
	json.Unmarshaler
	xml.Marshaler
	xml.Unmarshaler
	xml.MarshalerAttr
	xml.UnmarshalerAttr
	driver.Valuer
	sql.Scanner
	flag.Value
}

var (
	_ Operable    = String("")
	_ Convertible = (*String)(nil)
)
