package driver

import "fmt"

// KeyEncoder 定义缓存键到字符串的转换契约。
//
// 当业务侧的泛型 K 非 string 时（如自定义类型、结构体），需要通过 KeyEncoder
// 将其转换为稳定的字符串表示。
type KeyEncoder interface {
	// String 把任意键转换为字符串。
	// 实现必须保证：相同输入产生相同输出（可用作 map key）。
	String(key any) string
}

// DefaultKeyEncoder 是 [KeyEncoder] 的默认实现。
//
// 转换规则按优先级：
//  1. nil → 空串
//  2. string → 原值
//  3. []byte → string(b)
//  4. fmt.Stringer → key.String()
//  5. 其他 → fmt.Sprint(key)
type DefaultKeyEncoder struct{}

// String 把任意键转换为字符串，实现 [KeyEncoder] 接口。
func (DefaultKeyEncoder) String(key any) string {
	switch k := key.(type) {
	case nil:
		return ""
	case string:
		return k
	case []byte:
		return string(k)
	case fmt.Stringer:
		return k.String()
	default:
		return fmt.Sprint(key)
	}
}

// EncodeKey 使用传入 KeyEncoder 把任意键转换为字符串。
// 未注入时回退到 [DefaultKeyEncoder]。
func EncodeKey(enc KeyEncoder, key any) string {
	if enc == nil {
		return DefaultKeyEncoder{}.String(key)
	}
	return enc.String(key)
}
