package apperr

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
// 仅包含框架内部故障；HTTP/RPC及领域错误由对应插件维护。
var (
	FrameworkInternalError         = &ErrorCode{Code: 50000, Output: "FRAMEWORK_INTERNAL_ERROR", Message: "框架内部错误"}           // 框架内部错误。
	FrameworkContainerNotMounted   = &ErrorCode{Code: 50001, Output: "FRAMEWORK_CONTAINER_NOT_MOUNTED", Message: "应用容器未挂载"}   // 应用容器未挂载。
	FrameworkDependencyUndeclared  = &ErrorCode{Code: 50002, Output: "FRAMEWORK_DEPENDENCY_UNDECLARED", Message: "组件依赖未声明"}   // 组件依赖未声明。
	FrameworkComponentNotFound     = &ErrorCode{Code: 50003, Output: "FRAMEWORK_COMPONENT_NOT_FOUND", Message: "组件未找到"}       // 组件未找到。
	FrameworkComponentTypeMismatch = &ErrorCode{Code: 50004, Output: "FRAMEWORK_COMPONENT_TYPE_MISMATCH", Message: "组件类型不匹配"} // 组件类型不匹配。
	FrameworkComponentStopped      = &ErrorCode{Code: 50005, Output: "FRAMEWORK_COMPONENT_STOPPED", Message: "组件已停止"}         // 组件已停止。
	FrameworkComponentUnavailable  = &ErrorCode{Code: 50006, Output: "FRAMEWORK_COMPONENT_UNAVAILABLE", Message: "组件不可用"}     // 组件不可用。
	FrameworkComponentInvalidState = &ErrorCode{Code: 50007, Output: "FRAMEWORK_COMPONENT_INVALID_STATE", Message: "组件状态无效"}  // 组件状态无效。
	FrameworkMultipleFailures      = &ErrorCode{Code: 50020, Output: "FRAMEWORK_MULTIPLE_FAILURES", Message: "多个框架阶段失败"}      // 多个框架阶段失败。
	FrameworkHookPanic             = &ErrorCode{Code: 50021, Output: "FRAMEWORK_HOOK_PANIC", Message: "框架生命周期钩子异常"}           // 框架生命周期钩子异常。
	FrameworkShutdownIncomplete    = &ErrorCode{Code: 50022, Output: "FRAMEWORK_SHUTDOWN_INCOMPLETE", Message: "框架关闭未完成"}     // 框架关闭未完成。
	FrameworkWorkerUnexpectedExit  = &ErrorCode{Code: 50023, Output: "FRAMEWORK_WORKER_UNEXPECTED_EXIT", Message: "后台服务意外退出"} // 后台服务意外退出。
)
