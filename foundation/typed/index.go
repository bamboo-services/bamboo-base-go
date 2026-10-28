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

// MaxKeyBytes 是受控标识符键名的最大字节长度上限，当前为 128。
// [String.IsValid] 与六通道编解码的长度校验均以此为准。
const MaxKeyBytes = 128

// String 是以 Go 原生 string 为基底的强类型字符串原语。
//
// String 承担两种角色：
//
//   - 受控机器键与常量词汇：以 [String.IsValid] 的语法规则为准，
//     经 [Convertible] 汇总的六条标准通道双向编解码；
//   - 原始文本值载体：经 [String.Int]、[String.Bool]、[String.Duration]
//     等方法解析为 Go 原生类型，不强制键名语法。
//
// String 是不可变值类型，零值为空字符串；零值无法通过 [String.IsValid]，
// 直接编码会返回错误。未被并发写回的值可在多个 goroutine 间安全共享。
type String string

// Operable 约束所有基于 [String] 派生的领域词汇必须具备的基础操作能力，
// 使派生类型能够转换为通用 [String] 并复用本包的校验与转换方法。
type Operable interface {
	fmt.Stringer
	// TypedString 返回转换后的通用 String。
	TypedString() String
	// IsEmpty 报告底层值字节长度是否为 0。
	IsEmpty() bool
}

// Convertible 汇总 [String] 在六条标准通道上的双向编解码契约：
// Text、Binary、JSON、XML（含属性）、SQL 与 CLI Flag。
// *String 满足该接口；派生领域词汇实现这组契约后，
// 即可与标准库的序列化、数据库与命令行设施直接协作。
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
