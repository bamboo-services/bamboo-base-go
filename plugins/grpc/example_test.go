package xGrpc_test

import (
	"context"
	"fmt"

	xGrpc "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// ExampleNew 演示如何使用函数式选项构造 gRPC Worker 插件并挂载服务。
func ExampleNew() {
	// 构造 gRPC Worker 插件注册信息与工厂函数
	info, factory := xGrpc.New(
		xGrpc.WithAddr(":9090"),
		xGrpc.WithName("UserServiceWorker"),
		xGrpc.WithService(func(ctx context.Context, s *grpc.Server) {
			// 在此注册由 protoc 生成的 gRPC 服务定义
			// pb.RegisterUserServiceServer(s, &userServiceServer{})
		}),
		xGrpc.WithReflection(true),
	)

	_ = info
	_ = factory
	// 编译型示例，不执行真实网络监听
}

// ExampleToGrpcStatusCode 演示如何将 5 位业务错误码映射为标准 gRPC 状态码。
func ExampleToGrpcStatusCode() {
	c1 := xGrpc.ToGrpcStatusCode(40001) // 400 状态组 -> InvalidArgument
	c2 := xGrpc.ToGrpcStatusCode(40402) // 404 状态组 -> NotFound
	c3 := xGrpc.ToGrpcStatusCode(50000) // 500 状态组 -> Internal
	c4 := xGrpc.ToGrpcStatusCode(99999) // 未识别状态组 -> Unknown

	fmt.Println(c1 == codes.InvalidArgument)
	fmt.Println(c2 == codes.NotFound)
	fmt.Println(c3 == codes.Internal)
	fmt.Println(c4 == codes.Unknown)

	// Output:
	// true
	// true
	// true
	// true
}
