package driver

import (
	"encoding/json"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// Codec 定义缓存值的序列化契约，统一多后端的编解码行为。
//
// 业务侧若需要使用非 JSON 序列化（如 gob、protobuf、msgpack），
// 实现本接口并注入缓存管理器即可。
//
// 失败语义：
//   - 失败必须返回非 nil 的 [*apperr.Error]，建议使用 [CodecFailed]（50038）；
//   - 成功必须返回真正的 nil 指针，禁止把 nil 指针装入 error 接口冒充失败；
//   - 原生编解码错误（如 encoding/json 的语法错误）应作为根因保留，
//     便于调用方经 errors.Is / errors.As 精确判定。
type Codec interface {
	// Marshal 将值序列化为字节切片，便于以 []byte 形式存储。
	//
	// 失败时返回 [CodecFailed]（50038）并保留原生根因。
	Marshal(v any) ([]byte, *apperr.Error)
	// Unmarshal 将字节切片反序列化回目标值。
	//
	// 实现应使用反射或类型断言把数据写入 v 指向的内存。
	//
	// 失败时返回 [CodecFailed]（50038）并保留原生根因。
	Unmarshal(data []byte, v any) *apperr.Error
}

// JSONCodec 基于 encoding/json 的默认 [Codec] 实现。
//
// 无状态，可作为单例使用：JSONCodec{} 或 (&JSONCodec{}) 均可。
type JSONCodec struct{}

// Marshal 将值序列化为 JSON 字节切片。
//
// encoding/json 的原生失败被包装为 [CodecFailed]（50038），原生错误作为根因保留；
// 成功时返回真正的 nil 错误指针。
func (JSONCodec) Marshal(v any) ([]byte, *apperr.Error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, apperr.New(CodecFailed, "缓存值序列化失败", err)
	}
	return data, nil
}

// Unmarshal 将 JSON 字节切片反序列化回目标值。
//
// encoding/json 的原生失败被包装为 [CodecFailed]（50038），原生错误作为根因保留；
// 成功时返回真正的 nil 错误指针。
func (JSONCodec) Unmarshal(data []byte, v any) *apperr.Error {
	if err := json.Unmarshal(data, v); err != nil {
		return apperr.New(CodecFailed, "缓存值反序列化失败", err)
	}
	return nil
}
