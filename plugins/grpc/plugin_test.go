package xGrpc_test

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	xGrpc "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant"
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// 手写极简真实 gRPC ServiceDesc 与 Handler，避免引入外部生成代码
var testServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.TestService",
	HandlerType: (*testServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UnarySuccess",
			Handler:    unarySuccessHandler,
		},
		{
			MethodName: "UnaryPanic",
			Handler:    unaryPanicHandler,
		},
		{
			MethodName: "UnaryAppError",
			Handler:    unaryAppErrorHandler,
		},
		{
			MethodName: "UnaryBlockUntilCancel",
			Handler:    unaryBlockUntilCancelHandler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamPingPong",
			Handler:       streamPingPongHandler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "StreamPanic",
			Handler:       streamPanicHandler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
}

type testServer interface{}

func unarySuccessHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	req := new(wrapperspb.StringValue)
	if err := dec(req); err != nil {
		return nil, err
	}
	h := func(ctx context.Context, req any) (any, error) {
		reqKey := ctxkey.GetRequestKey(ctx)
		return wrapperspb.String("ok:" + reqKey), nil
	}
	if interceptor != nil {
		return interceptor(ctx, req, &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/test.TestService/UnarySuccess",
		}, h)
	}
	return h(ctx, req)
}

func unaryPanicHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	req := new(wrapperspb.StringValue)
	if err := dec(req); err != nil {
		return nil, err
	}
	h := func(ctx context.Context, req any) (any, error) {
		panic("database connection leaked sensitive password")
	}
	if interceptor != nil {
		return interceptor(ctx, req, &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/test.TestService/UnaryPanic",
		}, h)
	}
	return h(ctx, req)
}

func unaryAppErrorHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	req := new(wrapperspb.StringValue)
	if err := dec(req); err != nil {
		return nil, err
	}
	h := func(ctx context.Context, req any) (any, error) {
		return nil, apperr.New(apperr.ParameterError, "参数校验失败", errors.New("underlying private db err"))
	}
	if interceptor != nil {
		return interceptor(ctx, req, &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/test.TestService/UnaryAppError",
		}, h)
	}
	return h(ctx, req)
}

func unaryBlockUntilCancelHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	req := new(wrapperspb.StringValue)
	if err := dec(req); err != nil {
		return nil, err
	}
	h := func(ctx context.Context, req any) (any, error) {
		<-ctx.Done()
		return wrapperspb.String("canceled"), nil
	}
	if interceptor != nil {
		return interceptor(ctx, nil, &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/test.TestService/UnaryBlockUntilCancel",
		}, h)
	}
	return h(ctx, nil)
}

func streamPingPongHandler(srv any, stream grpc.ServerStream) error {
	ctx := stream.Context()
	reqKey := ctxkey.GetRequestKey(ctx)
	if reqKey == "" {
		return status.Error(codes.Internal, "missing request key")
	}
	// Echo back a trailer header
	return nil
}

func streamPanicHandler(srv any, stream grpc.ServerStream) error {
	panic("stream fatal panic")
}

func buildTestClient(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	cc, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("连接测试 gRPC 服务失败: %v", err)
	}
	return cc
}

func TestPlugin_BasicLifecycleAndUnary(t *testing.T) {
	_, factory := xGrpc.New(
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithService(func(_ context.Context, s *grpc.Server) {
			s.RegisterService(&testServiceDesc, struct{}{})
		}),
	)

	plugin := factory().(*xGrpc.Plugin)

	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStart()

	if err := plugin.Start(startCtx); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	addr := plugin.Addr()
	if addr == "" || addr == "127.0.0.1:0" {
		t.Fatalf("Addr() 返回了未解析的地址: %s", addr)
	}

	runDone := make(chan error, 1)
	lifetimeCtx, cancelLifetime := context.WithCancel(context.Background())
	defer cancelLifetime()

	go func() {
		runDone <- plugin.Run(lifetimeCtx)
	}()

	time.Sleep(50 * time.Millisecond)

	cc := buildTestClient(t, addr)
	defer func() { _ = cc.Close() }()

	// 1. Unary 成功调用与 RequestKey、Trailer 验证
	var trailerMD metadata.MD
	var out wrapperspb.StringValue
	reqMD := metadata.Pairs(xGrpcConst.MetadataRequestUUID.String(), "custom-trace-uuid-12345")
	callCtx := metadata.NewOutgoingContext(context.Background(), reqMD)

	err := cc.Invoke(callCtx, "/test.TestService/UnarySuccess", wrapperspb.String("hi"), &out, grpc.Trailer(&trailerMD))
	if err != nil {
		t.Fatalf("UnarySuccess 调用失败: %v", err)
	}
	if out.GetValue() != "ok:custom-trace-uuid-12345" {
		t.Fatalf("UnarySuccess 返回结果不符合预期: %s", out.GetValue())
	}
	trailerVals := trailerMD.Get(xGrpcConst.TrailerRequestUUID.String())
	if len(trailerVals) == 0 || trailerVals[0] != "custom-trace-uuid-12345" {
		t.Fatalf("TrailerRequestUUID 未写回: %v", trailerVals)
	}

	// 2. Unary Panic 捕获与堆栈不泄露验证
	err = cc.Invoke(context.Background(), "/test.TestService/UnaryPanic", wrapperspb.String(""), &out)
	if err == nil {
		t.Fatalf("预期 UnaryPanic 报错，实际成功")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("返回非 gRPC status 错误: %v", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("panic 映射 code 错误: 期望 %v, 得到 %v", codes.Internal, st.Code())
	}
	// status message 绝不能泄露敏感堆栈
	if st.Message() == "database connection leaked sensitive password" {
		t.Fatalf("错误泄露了原始堆栈文案: %s", st.Message())
	}

	// 3. Unary apperr 业务错误映射
	err = cc.Invoke(context.Background(), "/test.TestService/UnaryAppError", wrapperspb.String(""), &out)
	if err == nil {
		t.Fatalf("预期 UnaryAppError 报错，实际成功")
	}
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("apperr 映射 code 错误: 期望 InvalidArgument, 得到 %v", st.Code())
	}

	// 优雅关机
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStop()
	if err := plugin.Stop(stopCtx); err != nil {
		t.Fatalf("Stop 失败: %v", err)
	}

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run 退出返回错误: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Run 未在 Stop 后退出")
	}
}

func TestPlugin_StreamAndPanicSafety(t *testing.T) {
	_, factory := xGrpc.New(
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithService(func(_ context.Context, s *grpc.Server) {
			s.RegisterService(&testServiceDesc, struct{}{})
		}),
	)

	plugin := factory().(*xGrpc.Plugin)

	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStart()
	if err := plugin.Start(startCtx); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	go func() {
		_ = plugin.Run(context.Background())
	}()

	time.Sleep(50 * time.Millisecond)

	cc := buildTestClient(t, plugin.Addr())
	defer func() { _ = cc.Close() }()

	// 1. 测试正常 Stream 调用与 RequestKey 自动生成
	var trailerMD metadata.MD
	stream, err := cc.NewStream(context.Background(), &testServiceDesc.Streams[0], "/test.TestService/StreamPingPong", grpc.Trailer(&trailerMD))
	if err != nil {
		t.Fatalf("NewStream 失败: %v", err)
	}
	_ = stream.CloseSend()
	for {
		var dummy wrapperspb.StringValue
		recvErr := stream.RecvMsg(&dummy)
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			t.Fatalf("RecvMsg 失败: %v", recvErr)
		}
	}
	trailers := trailerMD.Get(xGrpcConst.TrailerRequestUUID.String())
	if len(trailers) == 0 || trailers[0] == "" {
		t.Fatalf("流式调用未回写 RequestUUID Trailer")
	}

	// 2. 测试 Stream Panic 安全处理
	streamPanic, err := cc.NewStream(context.Background(), &testServiceDesc.Streams[1], "/test.TestService/StreamPanic")
	if err != nil {
		t.Fatalf("NewStream StreamPanic 失败: %v", err)
	}
	_ = streamPanic.CloseSend()
	var dummy wrapperspb.StringValue
	recvErr := streamPanic.RecvMsg(&dummy)
	if recvErr == nil {
		t.Fatalf("预期 StreamPanic 报错，实际成功")
	}
	st, ok := status.FromError(recvErr)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("StreamPanic 映射状态码错误: %v", recvErr)
	}

	_ = plugin.Stop(context.Background())
}

func TestPlugin_MultiInstanceIsolation(t *testing.T) {
	// 创建实例 1 与 实例 2，验证端口、中间件注册表独立隔离
	reg1 := xGrpcMiddle.NewRegistry()
	var rec1 []string
	reg1.UseUnary(testServiceDesc, func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		rec1 = append(rec1, "inst1")
		return handler(ctx, req)
	})

	reg2 := xGrpcMiddle.NewRegistry()
	var rec2 []string
	reg2.UseUnary(testServiceDesc, func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		rec2 = append(rec2, "inst2")
		return handler(ctx, req)
	})

	_, f1 := xGrpc.New(
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithMiddlewareRegistry(reg1),
		xGrpc.WithService(func(_ context.Context, s *grpc.Server) {
			s.RegisterService(&testServiceDesc, struct{}{})
		}),
	)
	p1 := f1().(*xGrpc.Plugin)

	_, f2 := xGrpc.New(
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithMiddlewareRegistry(reg2),
		xGrpc.WithService(func(_ context.Context, s *grpc.Server) {
			s.RegisterService(&testServiceDesc, struct{}{})
		}),
	)
	p2 := f2().(*xGrpc.Plugin)

	if err := p1.Start(context.Background()); err != nil {
		t.Fatalf("p1 Start 失败: %v", err)
	}
	defer func() { _ = p1.Stop(context.Background()) }()

	if err := p2.Start(context.Background()); err != nil {
		t.Fatalf("p2 Start 失败: %v", err)
	}
	defer func() { _ = p2.Stop(context.Background()) }()

	go func() { _ = p1.Run(context.Background()) }()
	go func() { _ = p2.Run(context.Background()) }()

	time.Sleep(50 * time.Millisecond)

	if p1.Addr() == p2.Addr() {
		t.Fatalf("两实例分配到了相同端口: %s", p1.Addr())
	}

	cc1 := buildTestClient(t, p1.Addr())
	defer func() { _ = cc1.Close() }()
	cc2 := buildTestClient(t, p2.Addr())
	defer func() { _ = cc2.Close() }()

	var out wrapperspb.StringValue
	_ = cc1.Invoke(context.Background(), "/test.TestService/UnarySuccess", wrapperspb.String(""), &out)
	_ = cc2.Invoke(context.Background(), "/test.TestService/UnarySuccess", wrapperspb.String(""), &out)

	if len(rec1) != 1 || rec1[0] != "inst1" {
		t.Fatalf("实例 1 中间件未执行或被串扰: %v", rec1)
	}
	if len(rec2) != 1 || rec2[0] != "inst2" {
		t.Fatalf("实例 2 中间件未执行或被串扰: %v", rec2)
	}
}

func TestPlugin_GracefulStopAndForceCancel(t *testing.T) {
	// 测试在存在阻塞请求时，Stop 的排空与 Force 取消行为
	_, factory := xGrpc.New(
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithService(func(_ context.Context, s *grpc.Server) {
			s.RegisterService(&testServiceDesc, struct{}{})
		}),
	)
	plugin := factory().(*xGrpc.Plugin)

	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	go func() { _ = plugin.Run(context.Background()) }()
	time.Sleep(50 * time.Millisecond)

	addr := plugin.Addr()
	cc := buildTestClient(t, addr)
	defer func() { _ = cc.Close() }()

	// 发起阻塞请求
	handlerStarted := make(chan struct{})
	handlerDone := make(chan struct{})

	// 注册一个等待被中断的挂起调用
	var once sync.Once
	go func() {
		var out wrapperspb.StringValue
		once.Do(func() { close(handlerStarted) })
		_ = cc.Invoke(context.Background(), "/test.TestService/UnaryBlockUntilCancel", wrapperspb.String(""), &out)
		close(handlerDone)
	}()

	<-handlerStarted
	time.Sleep(50 * time.Millisecond)

	// Stop 传入短超时触发 force 中断
	stopCtx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	stopStart := time.Now()
	err := plugin.Stop(stopCtx)
	stopCost := time.Since(stopStart)

	if err == nil && stopCost < 100*time.Millisecond {
		t.Logf("Stop 正常返回: %v", err)
	}

	// 验证 Handler 真正被 cancel 打退并完成计数退还
	select {
	case <-handlerDone:
		// 正常被取消结束
	case <-time.After(2 * time.Second):
		t.Fatalf("Handler 未在 Stop/Force 后退出")
	}

	// 端口验证：确认 socket 已彻底关闭释放，能够重新监听
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("Stop 后原 socket 未释放，端口重开失败: %v", err)
	}
	_ = ln.Close()
}
