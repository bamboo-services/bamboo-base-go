package xGrpc_test

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	xGrpc "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
)

// 生命周期契约断言：四个生命周期方法必须返回统一的 [*apperr.Error]，
// 且插件必须继续满足内核冻结后的 [application.Plugin]、[application.RunnablePlugin]
// 与 [application.Quiescer] 契约。
var (
	_ func(*xGrpc.Plugin, context.Context) *apperr.Error = (*xGrpc.Plugin).Start
	_ func(*xGrpc.Plugin, context.Context) *apperr.Error = (*xGrpc.Plugin).Stop
	_ func(*xGrpc.Plugin, context.Context) *apperr.Error = (*xGrpc.Plugin).Run
	_ func(*xGrpc.Plugin, context.Context) *apperr.Error = (*xGrpc.Plugin).Quiesce
	_ application.Plugin                                 = (*xGrpc.Plugin)(nil)
	_ application.RunnablePlugin                         = (*xGrpc.Plugin)(nil)
	_ application.Quiescer                               = (*xGrpc.Plugin)(nil)
)

// contractServer 是契约测试用的极简服务接口。
type contractServer interface{}

// contractService 是承载“阻塞至取消”语义的测试服务实例。
type contractService struct {
	entered chan struct{}
}

// contractServiceDesc 声明契约测试使用的极简 gRPC 服务。
var contractServiceDesc = grpc.ServiceDesc{
	ServiceName: "contract.TestService",
	HandlerType: (*contractServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "BlockUntilCancel",
			Handler:    contractBlockUntilCancelHandler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "contract.proto",
}

// contractBlockUntilCancelHandler 在进入后阻塞至上下文取消，用于验证停机预算语义。
func contractBlockUntilCancelHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	req := new(wrapperspb.StringValue)
	if err := dec(req); err != nil {
		return nil, err
	}
	h := func(ctx context.Context, req any) (any, error) {
		if svcImpl, ok := srv.(*contractService); ok && svcImpl.entered != nil {
			close(svcImpl.entered)
		}
		<-ctx.Done()
		return wrapperspb.String("canceled"), nil
	}
	if interceptor != nil {
		return interceptor(ctx, req, &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/contract.TestService/BlockUntilCancel",
		}, h)
	}
	return h(ctx, req)
}

// TestPlugin_LifecycleSuccessReturnsTrueNil 验证成功路径返回真正的 nil 指针，
// 不得返回装入 error 接口的 nil 载体，也不得返回带空码的假失败。
func TestPlugin_LifecycleSuccessReturnsTrueNil(t *testing.T) {
	_, factory := xGrpc.New(xGrpc.WithAddr("127.0.0.1:0"))
	p := factory().(*xGrpc.Plugin)

	startErr := p.Start(context.Background())
	if startErr != nil {
		t.Fatalf("Start 失败: %v", startErr)
	}
	if !apperr.IsNil(startErr) {
		t.Fatalf("Start 成功时必须返回真正的 nil 指针，实际得到: %+v", startErr)
	}

	quiesceErr := p.Quiesce(context.Background())
	if quiesceErr != nil {
		t.Fatalf("Quiesce 失败: %v", quiesceErr)
	}
	if !apperr.IsNil(quiesceErr) {
		t.Fatalf("Quiesce 成功时必须返回真正的 nil 指针，实际得到: %+v", quiesceErr)
	}

	stopErr := p.Stop(context.Background())
	if stopErr != nil {
		t.Fatalf("Stop 失败: %v", stopErr)
	}
	if !apperr.IsNil(stopErr) {
		t.Fatalf("Stop 成功时必须返回真正的 nil 指针，实际得到: %+v", stopErr)
	}
}

// TestPlugin_DuplicateStartReturnsInvalidState 验证重复启动返回既有语义码
// [grpcresult.InvalidState]，并保留 [xGrpc.ErrServerAlreadyStarted] 作为根因。
func TestPlugin_DuplicateStartReturnsInvalidState(t *testing.T) {
	_, factory := xGrpc.New(xGrpc.WithAddr("127.0.0.1:0"))
	p := factory().(*xGrpc.Plugin)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	defer func() { _ = p.Stop(context.Background()) }()

	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("重复启动必须失败")
	}
	if err.Code.Code != grpcresult.InvalidState.Code {
		t.Fatalf("重复启动错误码错误: 期望 %d, 得到 %d", grpcresult.InvalidState.Code, err.Code.Code)
	}
	if !errors.Is(err, xGrpc.ErrServerAlreadyStarted) {
		t.Fatalf("重复启动必须保留 ErrServerAlreadyStarted 根因, 得到: %v", err)
	}
}

// TestPlugin_RunBeforeStartReturnsServiceUnavailable 验证未启动即调用 Run
// 返回既有语义码 [grpcresult.ServiceUnavailable]，并保留 [xGrpc.ErrNotStarted] 根因。
func TestPlugin_RunBeforeStartReturnsServiceUnavailable(t *testing.T) {
	_, factory := xGrpc.New(xGrpc.WithAddr("127.0.0.1:0"))
	p := factory().(*xGrpc.Plugin)

	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("未启动即调用 Run 必须失败")
	}
	if err.Code.Code != grpcresult.ServiceUnavailable.Code {
		t.Fatalf("未启动即 Run 的错误码错误: 期望 %d, 得到 %d", grpcresult.ServiceUnavailable.Code, err.Code.Code)
	}
	if !errors.Is(err, xGrpc.ErrNotStarted) {
		t.Fatalf("未启动即 Run 必须保留 ErrNotStarted 根因, 得到: %v", err)
	}
}

// TestPlugin_StartListenFailureKeepsRootCause 验证监听失败归一为模块内部错误码
// [grpcresult.InternalError]，且底层地址占用根因不丢失。
func TestPlugin_StartListenFailureKeepsRootCause(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("准备占用端口失败: %v", err)
	}
	defer func() { _ = occupied.Close() }()

	_, factory := xGrpc.New(xGrpc.WithAddr(occupied.Addr().String()))
	p := factory().(*xGrpc.Plugin)

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("监听失败必须返回错误")
	}
	if startErr.Code.Code != grpcresult.InternalError.Code {
		t.Fatalf("监听失败错误码错误: 期望 %d, 得到 %d", grpcresult.InternalError.Code, startErr.Code.Code)
	}
	if startErr.Err == nil {
		t.Fatal("监听失败必须保留底层根因")
	}
	if !errors.Is(startErr, syscall.EADDRINUSE) {
		t.Fatalf("监听失败必须保留地址占用根因, 得到: %v", startErr)
	}
}

// TestPlugin_StopWithCanceledContextClassifiedAsControlError 验证停机预算上下文取消
// 归一为控制类错误码，而不是被吞掉或误判为内部错误。
func TestPlugin_StopWithCanceledContextClassifiedAsControlError(t *testing.T) {
	entered := make(chan struct{})
	_, factory := xGrpc.New(
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithService(func(_ context.Context, s *grpc.Server) {
			s.RegisterService(&contractServiceDesc, &contractService{entered: entered})
		}),
	)
	p := factory().(*xGrpc.Plugin)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	runDone := make(chan *apperr.Error, 1)
	go func() {
		runDone <- p.Run(context.Background())
	}()

	cc := buildTestClient(t, p.Addr())
	defer func() { _ = cc.Close() }()

	invokeDone := make(chan struct{})
	go func() {
		defer close(invokeDone)
		var out wrapperspb.StringValue
		_ = cc.Invoke(context.Background(), "/contract.TestService/BlockUntilCancel", wrapperspb.String("wait"), &out)
	}()

	<-entered

	stopCtx, cancelStop := context.WithCancel(context.Background())
	cancelStop()

	stopErrCh := make(chan *apperr.Error, 1)
	go func() {
		stopErrCh <- p.Stop(stopCtx)
	}()

	stopErr := <-stopErrCh
	if stopErr == nil {
		t.Fatal("停机上下文取消时 Stop 必须返回失败")
	}
	if stopErr.Code.Code != control.Canceled.Code {
		t.Fatalf("停机上下文取消的错误码错误: 期望 %d, 得到 %d", control.Canceled.Code, stopErr.Code.Code)
	}
	if !errors.Is(stopErr, context.Canceled) {
		t.Fatalf("停机上下文取消必须保留原生 context.Canceled 根因, 得到: %v", stopErr)
	}

	<-invokeDone
	<-runDone
}
