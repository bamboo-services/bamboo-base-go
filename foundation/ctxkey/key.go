package ctxkey

import (
	"database/sql/driver"
	"encoding/xml"
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// Key 是基于 [typed.String] 派生的强类型上下文与插件标识键。
//
// 底层复用 [typed.String] 的受控键名语法校验与六大通道编解码：
// 合法键由 ASCII 字母开头，仅含字母、数字、下划线与连字符，
// 长度为 1 至 128 字节；零值为空键，等价于 [Nil]，
// 参与编码与校验时会得到否定结果。
type Key typed.String

var (
	_ typed.Operable    = Key("")
	_ typed.Convertible = (*Key)(nil)
)

// 内置上下文键目录。
// 元数据键由协议层在入口写入基础信息；组件键由上层容器
// 在装配共享组件时注入实例，取值类型以各键说明为准。
const (
	Nil              Key = ""                        // 空上下文键，等价于 [Key] 的零值。
	RequestKey       Key = "context_request_key"     // 请求唯一标识的上下文键，约定携带 string 值，可经 [GetRequestKey] 读取。
	ErrorCodeKey     Key = "context_error_code"      // 错误码对象的上下文键。
	ErrorMessageKey  Key = "context_error_message"   // 错误消息的上下文键，约定携带 string 值，可经 [GetErrorMessage] 读取。
	UserStartTimeKey Key = "context_user_start_time" // 请求起始时间的上下文键，约定携带 time.Time 值，供 [CalcOverheadTime] 计算请求耗时。
	DatabaseKey      Key = "context_database"        // 数据库连接实例的上下文键。
	RedisClientKey   Key = "context_redis_client"    // Redis 客户端实例的上下文键。
	SnowflakeNodeKey Key = "context_snowflake_node"  // 雪花算法节点实例的上下文键。
	EmailClientKey   Key = "context_email_client"    // 邮件客户端实例的上下文键。
	CacheManagerKey  Key = "context_cache_manager"   // 缓存管理器实例的上下文键。
)

// Exposer 交付插件在 PluginInfo.Provides 中声明的底层句柄。
//
// 实现方将键到组件实例的映射交给上层容器完成注入；
// 返回映射中实例的构造与生命周期仍由实现方维护。
type Exposer interface {
	// Expose 返回键到组件实例的映射。
	Expose() map[Key]any
}

// String 返回键的底层原生字符串。
func (k Key) String() string {
	return string(k)
}

// TypedString 返回与键等值的 [typed.String]，
// 用于复用其校验与解析能力。
func (k Key) TypedString() typed.String {
	return typed.String(k)
}

// IsEmpty 报告键是否为空；零值与 [Nil] 返回 true。
func (k Key) IsEmpty() bool {
	return len(k) == 0
}

// IsValid 校验键是否满足受控键名语法，委托 [typed.String] 执行。
func (k Key) IsValid() bool {
	return typed.String(k).IsValid()
}

// MarshalText 实现 encoding.TextMarshaler 接口。
// 编码前先校验键名，空键或非法键返回错误。
func (k Key) MarshalText() ([]byte, error) {
	return k.TypedString().MarshalText()
}

// UnmarshalText 实现 encoding.TextUnmarshaler 接口。
// 仅在校验成功后写回键值，校验失败时返回错误且原键值保持不变；
// nil 接收器返回错误。
func (k *Key) UnmarshalText(text []byte) error {
	if k == nil {
		return fmt.Errorf("ctxkey.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalText(text); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalBinary 实现 encoding.BinaryMarshaler 接口，
// 供 gob 等二进制通道复用；编码前先校验键名，
// 空键或非法键返回错误。
func (k Key) MarshalBinary() ([]byte, error) {
	return k.TypedString().MarshalBinary()
}

// UnmarshalBinary 实现 encoding.BinaryUnmarshaler 接口。
// 仅在校验成功后写回键值，校验失败时返回错误且原键值保持不变；
// nil 接收器返回错误。
func (k *Key) UnmarshalBinary(data []byte) error {
	if k == nil {
		return fmt.Errorf("ctxkey.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalBinary(data); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口，编码为 JSON 字符串。
// 编码前先校验键名，空键或非法键返回错误。
func (k Key) MarshalJSON() ([]byte, error) {
	return k.TypedString().MarshalJSON()
}

// UnmarshalJSON 实现 json.Unmarshaler 接口。
// 仅接受 JSON 字符串，严格拒绝 null 与非字符串；
// 校验失败时返回错误且原键值保持不变，nil 接收器返回错误。
func (k *Key) UnmarshalJSON(raw []byte) error {
	if k == nil {
		return fmt.Errorf("ctxkey.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalJSON(raw); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalXML 实现 xml.Marshaler 接口，将键编码为 XML 元素内容。
// 编码前先校验键名，空键或非法键返回错误。
func (k Key) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return k.TypedString().MarshalXML(e, start)
}

// UnmarshalXML 实现 xml.Unmarshaler 接口。
// 仅在校验成功后写回键值，校验失败时返回错误且原键值保持不变；
// nil 接收器返回错误。
func (k *Key) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if k == nil {
		return fmt.Errorf("ctxkey.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalXML(d, start); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalXMLAttr 实现 xml.MarshalerAttr 接口，将键编码为 XML 属性。
// 编码前先校验键名，空键或非法键返回错误。
func (k Key) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return k.TypedString().MarshalXMLAttr(name)
}

// UnmarshalXMLAttr 实现 xml.UnmarshalerAttr 接口。
// 仅在校验成功后写回键值，校验失败时返回错误且原键值保持不变；
// nil 接收器返回错误。
func (k *Key) UnmarshalXMLAttr(attr xml.Attr) error {
	if k == nil {
		return fmt.Errorf("ctxkey.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalXMLAttr(attr); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// Value 实现 driver.Valuer 接口，输出字符串驱动值。
// 输出前先校验键名，空键或非法键返回错误。
func (k Key) Value() (driver.Value, error) {
	return k.TypedString().Value()
}

// Scan 实现 sql.Scanner 接口。
// 仅接受 string 与 []byte，拒绝 SQL NULL；仅在校验成功后写回键值，
// 失败时原键值保持不变；nil 接收器返回错误。
func (k *Key) Scan(src any) error {
	if k == nil {
		return fmt.Errorf("ctxkey.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.Scan(src); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// Set 实现 flag.Value 接口，配合 [Key.String] 使 *Key 可直接用作
// 命令行标志。仅在校验成功后写回键值，失败时原键值保持不变；
// nil 接收器返回错误。
func (k *Key) Set(raw string) error {
	if k == nil {
		return fmt.Errorf("ctxkey.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.Set(raw); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}
