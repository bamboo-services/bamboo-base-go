package dialect

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；仅处理驱动声明、适配器契约与部署配置失败。
var (
	DriverInvalid        = &apperr.ErrorCode{Code: 50033, Output: "DATABASE_DIALECT_DRIVER_INVALID", Message: "数据库驱动声明非法"}          // 驱动声明非法。
	DriverNotLoaded      = &apperr.ErrorCode{Code: 50034, Output: "DATABASE_DIALECT_DRIVER_NOT_LOADED", Message: "数据库驱动未加载"}        // 驱动未加载。
	AdapterInvalid       = &apperr.ErrorCode{Code: 50035, Output: "DATABASE_DIALECT_ADAPTER_INVALID", Message: "数据库适配器无效"}          // 适配器对象契约无效。
	AdapterConfigInvalid = &apperr.ErrorCode{Code: 50036, Output: "DATABASE_DIALECT_ADAPTER_CONFIG_INVALID", Message: "数据库适配器配置非法"} // 适配器启动配置非法。
)
