// Package xGrpcUtil 提供 gRPC 运行时的辅助工具函数。
//
// 主要提供从 gRPC 传入上下文（Incoming Metadata）安全提取元数据键值的契约封装，
// 辅助拦截器、鉴权中间件及业务 Handler 快速解析外部请求元数据。
//
// # 元数据提取与空白修剪
//
// gRPC 的元数据基于 HTTP/2 Headers 实现，同一键名可能关联多个切片条目。
// [ExtractMetadata] 会安全检查上下文，遍历指定键下的所有切片值，
// 并返回首个去除首尾空白字符后非空的字符串。
//
// # 典型用法
//
// 在中间件中读取客户端传入的应用访问凭证或追踪标识：
//
//	accessID, err := xGrpcUtil.ExtractMetadata(ctx, xGrpcConst.MetadataAppAccessID)
//	if err != nil {
//		// 缺少凭证处理
//		return nil, grpcresult.ToStatusError(ctx, err)
//	}
//
// # 错误安全与脱敏
//
// 当上下文缺失元数据或未包含指定键时，[ExtractMetadata] 返回标准结构化
// 业务错误 [*apperr.Error]（包含 [apperr.NotExist] 错误码）。
// 错误信息仅说明键名缺失，不打印任何外部传入的脏数据或敏感内容。
package xGrpcUtil
