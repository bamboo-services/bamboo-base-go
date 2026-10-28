// Package internaltest 提供专供 grpcresult 包单元测试与契约校验使用的 Protobuf 测试桩消息。
//
// 本包为内部测试辅助包，由 testdata.proto 编译生成，不对生产业务提供服务能力。
// 内部包含了满足不同反射约束的测试消息类型：
//
//   - [ValidAlphaResponse]：顶层包含唯一名称为 header 的 BaseResponse 字段，用于验证单 BaseResponse 的合法注入；
//   - [ValidBetaResponse]：顶层包含唯一名称为 custom_base 的 BaseResponse 字段，用于验证任意字段名的反射发现；
//   - [MissingBaseResponse]：顶层不包含任何 BaseResponse 字段，用于验证缺失信封时的 panic 拦截；
//   - [MultipleBaseResponse]：顶层包含多个 BaseResponse 字段，用于验证字段歧义时的 panic 拦截。
//
// # 维护规则
//
// 本包属于测试辅助工具，禁止生产业务代码引用。生成的 testdata.pb.go 同样由 protoc 自动维护。
package internaltest
