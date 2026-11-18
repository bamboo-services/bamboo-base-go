package driver

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	CodecFailed     = &apperr.ErrorCode{Code: 50038, Output: "CACHE_DRIVER_CODEC_FAILED", Message: "缓存编解码失败"}    // 缓存值序列化或反序列化失败。
	OperationFailed = &apperr.ErrorCode{Code: 50039, Output: "CACHE_DRIVER_OPERATION_FAILED", Message: "缓存操作失败"} // 缓存数据面操作或后端执行失败。
)
