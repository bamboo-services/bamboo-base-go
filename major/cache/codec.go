package xCache

import "encoding/json"

// Codec 定义缓存值的序列化契约，统一 Redis 与 Memory 两套后端的编解码行为。
//
// 业务侧若需要使用非 JSON 序列化（如 gob、protobuf、msgpack），
// 实现本接口并通过 [WithCodec] 注入 [Manager] 即可。
//
// 注意：同一份缓存数据在 Redis 与 Memory 后端间不保证互通，
// 切换后端时应清空缓存避免反序列化失败。
type Codec interface {
	// Marshal 将值序列化为字节切片，便于在 Redis/内存中以 []byte 形式存储。
	Marshal(v any) ([]byte, error)
	// Unmarshal 将字节切片反序列化回目标值。
	// 实现应使用反射或类型断言把数据写入 v 指向的内存。
	Unmarshal(data []byte, v any) error
}

// JSONCodec 基于 encoding/json 的默认 [Codec] 实现。
//
// 无状态，可作为单例使用：JSONCodec{} 或 (&JSONCodec{}) 均可。
// 性能足以覆盖绝大多数业务场景；若对序列化性能敏感，
// 可替换为 sonic / jsoniter 等实现。
type JSONCodec struct{}

// Marshal 将值序列化为 JSON 字节切片。
func (JSONCodec) Marshal(v any) ([]byte, error) { return json.Marshal(v) }

// Unmarshal 将 JSON 字节切片反序列化回目标值。
func (JSONCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
