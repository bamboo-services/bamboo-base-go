// Package generate 包含由 Protocol Buffers 编译器自动生成的 gRPC 基础协议代码。
//
// 本包定义并维护 Bamboo 服务的通用响应协议结构，其源码来源为 proto/base.proto。
// 主要承载 [BaseResponse] 统一响应元信息信封，供 gRPC 插件内核、grpcresult 响应构造器
// 以及外部客户端解析协议信封使用。
//
// # 核心类型
//
// [BaseResponse] 是微服务 RPC 调用的标准化统一信封，包含以下关键字段：
//
//   - Context：请求链路追踪唯一标识（UUID），与 HTTP 头的 X-Request-UUID 对应；
//   - Output：业务状态语义标签（如 "Success" 或错误枚举名称）；
//   - Code：数值型业务状态码（200 表示成功，其余为领域业务错误码）；
//   - Message：面向调用方的安全描述文案；
//   - ErrorMessage：可选的内部详细错误描述（仅在错误场景下有选择地提供，避免泄露内部敏感信息）；
//   - Overhead：可选的请求处理耗时（微秒整数），仅在应用开启 Debug 模式时填充。
//
// # 维护与代码生成
//
// 本包内的 Go 源码由 Buf 与 protoc-gen-go 生成，禁止手动修改 *.pb.go 文件。
// 若协议字段定义发生演进，请在 proto/ 目录下修改对应 .proto 定义后通过
// make proto 重新生成并执行 make proto-check 校验一致性。
package generate
