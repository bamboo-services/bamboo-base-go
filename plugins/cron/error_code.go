package cron

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	SchedulerError = &apperr.ErrorCode{Code: 50040, Output: "CRON_SCHEDULER_ERROR", Message: "定时调度器错误"} // 定时调度器配置或运行失败。
)
