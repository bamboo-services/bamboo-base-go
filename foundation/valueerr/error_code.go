package valueerr

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
//
// 400 组表示输入值本身非法（格式、范围、类型），500 组表示转换目标或
// 接收器等调用契约非法；两类不得互相替代。
var (
	InvalidFormat     = &apperr.ErrorCode{Code: 40013, Output: "VALUE_INVALID_FORMAT", Message: "值格式非法"}        // 值格式非法。
	OutOfRange        = &apperr.ErrorCode{Code: 40014, Output: "VALUE_OUT_OF_RANGE", Message: "值超出范围"}          // 值超出范围。
	InvalidType       = &apperr.ErrorCode{Code: 40015, Output: "VALUE_INVALID_TYPE", Message: "值类型非法"}          // 值类型非法。
	UnsupportedTarget = &apperr.ErrorCode{Code: 50018, Output: "VALUE_UNSUPPORTED_TARGET", Message: "不支持的转换目标"} // 不支持的转换目标。
	InvalidReceiver   = &apperr.ErrorCode{Code: 50019, Output: "VALUE_INVALID_RECEIVER", Message: "值接收器无效"}     // 值接收器无效。
)
