package xGrpcMiddle_test

import (
	"context"
	"fmt"

	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware"
	"google.golang.org/grpc"
)

// ExampleExtractServiceName 演示从 gRPC FullMethod 中提取服务限定名。
func ExampleExtractServiceName() {
	s1 := xGrpcMiddle.ExtractServiceName("/user.v1.UserService/GetUser")
	s2 := xGrpcMiddle.ExtractServiceName("/OrderService/Create")
	s3 := xGrpcMiddle.ExtractServiceName("DirectService")

	fmt.Println(s1)
	fmt.Println(s2)
	fmt.Println(s3)
	// Output:
	// user.v1.UserService
	// OrderService
	// DirectService
}

// ExampleRegistry 演示中间件注册表的创建、挂载与按服务查找。
func ExampleRegistry() {
	reg := xGrpcMiddle.NewRegistry()

	// 模拟自定义一元中间件
	dummyMiddleware := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(ctx, req)
	}

	serviceDesc := grpc.ServiceDesc{
		ServiceName: "account.AccountService",
	}

	reg.UseUnary(serviceDesc, dummyMiddleware)

	// 按服务名查询中间件链
	mws := reg.LookupUnary("account.AccountService")
	fmt.Println(len(mws))

	// 查询未注册服务返回 nil
	notFound := reg.LookupUnary("unknown.Service")
	fmt.Println(len(notFound))
	// Output:
	// 1
	// 0
}
