package async

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	PoolFull  = &apperr.ErrorCode{Code: 42903, Output: "ASYNC_POOL_FULL", Message: "异步任务池已满"}  // 非阻塞任务提交因并发容量已满被拒绝。
	PoolError = &apperr.ErrorCode{Code: 50041, Output: "ASYNC_POOL_ERROR", Message: "异步任务池错误"} // 异步任务池配置、生命周期或准入失败。
)
