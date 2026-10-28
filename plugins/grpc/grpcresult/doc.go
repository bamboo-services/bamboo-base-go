// Package grpcresult 提供面向 gRPC 服务的统一响应构造、Protobuf 信封注入与错误安全归一契约。
//
// 本包承担微服务 RPC 调用结果的标准输出收口职责，与 HTTP 的 envelope 和 httpresult
// 保持统一设计规范。主要提供成功信封构造函数 [Success]、强类型 Protobuf 响应泛型构造器
// [SuccessWith] 以及业务错误至 gRPC 状态转换器 [ToStatusError]。
//
// # 成功响应构造与泛型注入
//
// 业务在实现 Protobuf 生成的 gRPC Service 时，通过 [SuccessWith] 返回携带
// 标准响应信封的消息指针：
//
//	func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
//		// 业务逻辑查询...
//		resp := grpcresult.SuccessWith[*pb.GetUserResponse](ctx, "获取用户成功")
//		resp.User = &pb.UserInfo{Id: 1001, Name: "Konata"}
//		return resp, nil
//	}
//
// [SuccessWith] 会自动通过 Protobuf 反射实例化消息对象，并查找顶层类型为
// [generate.BaseResponse] 的唯一字段，调用 [Success] 完成追踪 ID、耗时及状态注入。
//
// # 泛型类型参数 T 的契约约束
//
// 为保证框架行为确定性并杜绝隐式隐患，[SuccessWith] 对类型参数 T 执行严格断言：
//
//   - T 必须为具体结构体指针类型，禁止传入接口或非指针类型；
//   - T 必须实现 proto.Message 接口；
//   - T 的 Protobuf 顶层字段中必须恰好有且仅有一个消息字段类型为 [generate.BaseResponse]；
//   - 违背上述约束直接触发 panic 快速失败，不对任意类型做无条件承诺。
//
// # 业务错误转换与安全脱敏
//
// 当业务需要返回错误时，调用 [ToStatusError] 将 [*apperr.Error] 转换为
// 标准 gRPC status 错误：
//
//   - 严禁将内部底层的原始 SQL 错误、私密系统信息或堆栈输出到 status 响应中；
//   - 外部调用方仅接收安全的业务错误文案，并通过 status.Details 解码
//     [generate.BaseResponse] 结构获取标准错误码与追踪上下文；
//   - 错误状态码由业务错误码前 3 位（HTTP 状态组）映射至对应的 gRPC 状态码。
//
// # 并发安全
//
// 本包所有导出函数均为无外部可变状态的纯函数或构造函数，输入独立的 ctx 即可并发安全调用。
package grpcresult
