// Package xGrpcConst 定义 gRPC 传输层常用的强类型元数据（Metadata 与 Trailer）键常量。
//
// 本包属于纯常量与基础类型契约包，为插件内部拦截器、中间件以及业务鉴权、
// 链路追踪等提供统一、强类型、抗拼写错误的键名定义。
//
// # 元数据分类与约定
//
// gRPC 元数据分为请求入站/出站头（Metadata）与请求尾部（Trailer）：
//
//   - [Metadata]：用于 HTTP/2 请求 Header 阶段传递的元数据，如应用凭据与追踪 ID；
//   - [Trailer]：用于 HTTP/2 响应流结束阶段传递的元数据，如请求处理完成后的追踪回写。
//
// # 典型用法
//
// 读取传入元数据中的追踪 ID 或应用凭证：
//
//	if md, ok := metadata.FromIncomingContext(ctx); ok {
//		if vals := md.Get(xGrpcConst.MetadataRequestUUID.String()); len(vals) > 0 {
//			traceID := vals[0]
//			_ = traceID
//		}
//	}
//
// # 并发与零值
//
// [Metadata] 与 [Trailer] 基于 string 类型定义，为不可变值类型，
// 天然具备并发安全性。[Metadata.String] 与 [Trailer.String] 实现了 [fmt.Stringer] 接口。
package xGrpcConst
