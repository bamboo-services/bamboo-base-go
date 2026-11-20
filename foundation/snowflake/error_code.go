package snowflake

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	NodeInvalid    = &apperr.ErrorCode{Code: 50026, Output: "SNOWFLAKE_NODE_INVALID", Message: "数据中心或节点编号非法"} // 数据中心或节点编号非法。
	ClockBackwards = &apperr.ErrorCode{Code: 50027, Output: "SNOWFLAKE_CLOCK_BACKWARDS", Message: "系统时钟回退"}   // 系统时钟回退。
)
