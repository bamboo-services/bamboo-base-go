// Package envelope 定义跨协议复用的标准响应信封契约。
//
// [Response] 描述 Bamboo 服务对外统一返回的字段结构，与 apperr
// 错误码体系配套使用。本包只定义契约与 JSON 序列化格式，不绑定
// HTTP 或 gRPC 的写出行为；协议插件负责把信封编码为各自的传输格式。
//
// # 字段与序列化
//
// Code、Output 与 Message 恒定输出，错误场景通常取自 apperr
// 预定义错误码，成功场景取值由协议层决定；Context、ErrorMessage、
// Overhead 与 Data 为可选项，零值时 JSON 输出省略对应字段。
//
// # 典型用法
//
// 以下片段构造一个携带业务数据的错误信封并序列化为 JSON：
//
//	resp := envelope.Response{
//		Context:      "req-123456",
//		Code:         apperr.BadRequest.Code,
//		Output:       apperr.BadRequest.Output,
//		Message:      apperr.BadRequest.Message,
//		ErrorMessage: "field 'username' is invalid",
//		Data:         map[string]any{"field": "username"},
//	}
//	data, err := json.Marshal(resp)
//
// # 使用边界
//
// [Response] 是纯数据契约，不提供构造函数与校验方法；错误场景的
// 信封组装由协议插件依据 apperr 的错误值完成。字段可自由读写，
// 并发安全由使用方自行保证。
package envelope
