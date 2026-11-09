package logging

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	HandlerInvalid = &apperr.ErrorCode{Code: 50028, Output: "LOGGING_HANDLER_INVALID", Message: "日志处理器缺失"} // 日志处理器缺失。
	ConfigInvalid  = &apperr.ErrorCode{Code: 50029, Output: "LOGGING_CONFIG_INVALID", Message: "日志配置非法"}   // 日志配置非法。
	WriteFailed    = &apperr.ErrorCode{Code: 50030, Output: "LOGGING_WRITE_FAILED", Message: "日志写入失败"}     // 日志写入失败。
	RotationFailed = &apperr.ErrorCode{Code: 50031, Output: "LOGGING_ROTATION_FAILED", Message: "日志轮转失败"}  // 日志轮转失败。
)
