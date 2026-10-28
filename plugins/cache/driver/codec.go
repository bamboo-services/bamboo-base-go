package driver

import "encoding/json"

// Codec 定义缓存值的序列化契约，统一多后端的编解码行为。
//
// 业务侧若需要使用非 JSON 序列化（如 gob、protobuf、msgpack），
// 实现本接口并注入缓存管理器即可。
type Codec interface {
	// Marshal 将值序列化为字节切片，便于以 []byte 形式存储。
	Marshal(v any) ([]byte, error)
	// Unmarshal 将字节切片反序列化回目标值。
	// 实现应使用反射或类型断言把数据写入 v 指向的内存。
	Unmarshal(data []byte, v any) error
}

// JSONCodec 基于 encoding/json 的默认 [Codec] 实现。
//
// 无状态，可作为单例使用：JSONCodec{} 或 (&JSONCodec{}) 均可。
type JSONCodec struct{}

// Marshal 将值序列化为 JSON 字节切片。
func (JSONCodec) Marshal(v any) ([]byte, error) { return json.Marshal(v) }

// Unmarshal 将 JSON 字节切片反序列化回目标值。
func (JSONCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
