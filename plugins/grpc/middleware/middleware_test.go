package xGrpcMiddle

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"google.golang.org/grpc"
)

// dummyUnaryInterceptor 创建一个标记一元拦截器，用于测试中间件链注册和查找。
func dummyUnaryInterceptor(tag string, recorder *[]string) UnaryMiddlewareFunc {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		*recorder = append(*recorder, tag+"-enter")
		resp, err := handler(ctx, req)
		*recorder = append(*recorder, tag+"-exit")
		return resp, err
	}
}

// dummyStreamInterceptor 创建一个标记流式拦截器，用于测试中间件链注册和查找。
func dummyStreamInterceptor(tag string, recorder *[]string) StreamMiddlewareFunc {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		*recorder = append(*recorder, tag+"-enter")
		err := handler(srv, ss)
		*recorder = append(*recorder, tag+"-exit")
		return err
	}
}

func TestUseUnaryBasic(t *testing.T) {
	reg := NewRegistry()
	var recorder []string
	reg.UseUnary(grpc.ServiceDesc{ServiceName: "pkg.Svc"},
		dummyUnaryInterceptor("A", &recorder),
		nil,
		dummyUnaryInterceptor("B", &recorder),
	)

	mws := reg.LookupUnary("pkg.Svc")
	if len(mws) != 2 {
		t.Errorf("有效中间件数量 = %d, 期望 2", len(mws))
	}
}

func TestUseUnaryAppend(t *testing.T) {
	reg := NewRegistry()
	var recorder []string
	desc := grpc.ServiceDesc{ServiceName: "pkg.Svc"}
	reg.UseUnary(desc, dummyUnaryInterceptor("A", &recorder))
	reg.UseUnary(desc, dummyUnaryInterceptor("B", &recorder))

	mws := reg.LookupUnary("pkg.Svc")
	if len(mws) != 2 {
		t.Errorf("追加后中间件数量 = %d, 期望 2", len(mws))
	}
}

func TestUseStreamBasic(t *testing.T) {
	reg := NewRegistry()
	var recorder []string
	reg.UseStream(grpc.ServiceDesc{ServiceName: "pkg.StreamSvc"},
		dummyStreamInterceptor("A", &recorder),
		nil,
		dummyStreamInterceptor("B", &recorder),
	)

	mws := reg.LookupStream("pkg.StreamSvc")
	if len(mws) != 2 {
		t.Errorf("有效中间件数量 = %d, 期望 2", len(mws))
	}
}

func TestUseStreamAppend(t *testing.T) {
	reg := NewRegistry()
	var recorder []string
	desc := grpc.ServiceDesc{ServiceName: "pkg.StreamSvc"}
	reg.UseStream(desc, dummyStreamInterceptor("A", &recorder))
	reg.UseStream(desc, dummyStreamInterceptor("B", &recorder))

	mws := reg.LookupStream("pkg.StreamSvc")
	if len(mws) != 2 {
		t.Errorf("追加后中间件数量 = %d, 期望 2", len(mws))
	}
}

func TestUseMultipleServices(t *testing.T) {
	reg := NewRegistry()
	var recorder []string
	reg.UseUnary(grpc.ServiceDesc{ServiceName: "pkg.A"}, dummyUnaryInterceptor("A1", &recorder))
	reg.UseUnary(grpc.ServiceDesc{ServiceName: "pkg.B"}, dummyUnaryInterceptor("B1", &recorder), dummyUnaryInterceptor("B2", &recorder))
	reg.UseStream(grpc.ServiceDesc{ServiceName: "pkg.StreamA"}, dummyStreamInterceptor("SA1", &recorder))

	if mws := reg.LookupUnary("pkg.A"); len(mws) != 1 {
		t.Errorf("pkg.A 中间件数量 = %d, 期望 1", len(mws))
	}
	if mws := reg.LookupUnary("pkg.B"); len(mws) != 2 {
		t.Errorf("pkg.B 中间件数量 = %d, 期望 2", len(mws))
	}
	if mws := reg.LookupUnary("pkg.C"); mws != nil {
		t.Errorf("未注册的 pkg.C 应返回 nil, 实际 = %v", mws)
	}
	if mws := reg.LookupStream("pkg.StreamA"); len(mws) != 1 {
		t.Errorf("pkg.StreamA 中间件数量 = %d, 期望 1", len(mws))
	}
	if mws := reg.LookupStream("pkg.StreamB"); mws != nil {
		t.Errorf("未注册的 pkg.StreamB 应返回 nil, 实际 = %v", mws)
	}
}

func TestUseAllNilSkipped(t *testing.T) {
	reg := NewRegistry()

	reg.UseUnary(grpc.ServiceDesc{ServiceName: "pkg.Empty"}, nil, nil)
	if mws := reg.LookupUnary("pkg.Empty"); mws != nil {
		t.Errorf("全 nil 注册后 LookupUnary 应返回 nil, 实际长度 = %d", len(mws))
	}

	reg.UseStream(grpc.ServiceDesc{ServiceName: "pkg.StreamEmpty"}, nil, nil)
	if mws := reg.LookupStream("pkg.StreamEmpty"); mws != nil {
		t.Errorf("全 nil 注册后 LookupStream 应返回 nil, 实际长度 = %d", len(mws))
	}
}

func TestLookupEmpty(t *testing.T) {
	reg := NewRegistry()

	if mws := reg.LookupUnary("any.Service"); mws != nil {
		t.Errorf("空注册表 LookupUnary 应返回 nil, 实际 = %v", mws)
	}
	if mws := reg.LookupStream("any.Service"); mws != nil {
		t.Errorf("空注册表 LookupStream 应返回 nil, 实际 = %v", mws)
	}
}

func TestConcurrentSafe(t *testing.T) {
	reg := NewRegistry()
	var wg sync.WaitGroup

	// 并发写入 Unary
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var recorder []string
			reg.UseUnary(
				grpc.ServiceDesc{ServiceName: "pkg.Concurrent"},
				dummyUnaryInterceptor("mw", &recorder),
			)
		}()
	}

	// 并发写入 Stream
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var recorder []string
			reg.UseStream(
				grpc.ServiceDesc{ServiceName: "pkg.StreamConcurrent"},
				dummyStreamInterceptor("smw", &recorder),
			)
		}()
	}

	// 并发读取
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.LookupUnary("pkg.Concurrent")
			_ = reg.LookupStream("pkg.StreamConcurrent")
		}()
	}

	wg.Wait()

	mws := reg.LookupUnary("pkg.Concurrent")
	if len(mws) != 100 {
		t.Errorf("并发注册后 Unary 中间件数量 = %d, 期望 100", len(mws))
	}

	smws := reg.LookupStream("pkg.StreamConcurrent")
	if len(smws) != 100 {
		t.Errorf("并发注册后 Stream 中间件数量 = %d, 期望 100", len(smws))
	}
}

func TestRegistryIsolation(t *testing.T) {
	regA := NewRegistry()
	regB := NewRegistry()

	var recA, recB []string
	regA.UseUnary(grpc.ServiceDesc{ServiceName: "pkg.Isolated"}, dummyUnaryInterceptor("A", &recA))
	regB.UseUnary(grpc.ServiceDesc{ServiceName: "pkg.Isolated"}, dummyUnaryInterceptor("B", &recB))

	mwsA := regA.LookupUnary("pkg.Isolated")
	mwsB := regB.LookupUnary("pkg.Isolated")

	if len(mwsA) != 1 || len(mwsB) != 1 {
		t.Fatalf("两注册表长度应各为 1, 实际 A=%d, B=%d", len(mwsA), len(mwsB))
	}

	// 执行验证中间件互不串
	_, _ = mwsA[0](context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/pkg.Isolated/Call"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "okA", nil
	})
	if len(recA) != 2 || len(recB) != 0 {
		t.Errorf("regA 执行不应影响 regB，recA=%v, recB=%v", recA, recB)
	}

	_, _ = mwsB[0](context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/pkg.Isolated/Call"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "okB", nil
	})
	if len(recB) != 2 {
		t.Errorf("regB 执行后 recB 应有记录, recB=%v", recB)
	}
}

func TestMiddlewareExecutionChainAndErrorPropagation(t *testing.T) {
	reg := NewRegistry()
	type ctxKey struct{}

	mw1 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = context.WithValue(ctx, ctxKey{}, "value1")
		return handler(ctx, req)
	}
	expectedErr := errors.New("boom")
	mw2 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		val := ctx.Value(ctxKey{})
		if val != "value1" {
			return nil, fmt.Errorf("context not propagated: %v", val)
		}
		return nil, expectedErr
	}

	reg.UseUnary(grpc.ServiceDesc{ServiceName: "pkg.Chain"}, mw1, mw2)
	mws := reg.LookupUnary("pkg.Chain")
	if len(mws) != 2 {
		t.Fatalf("期望 2 个中间件, 实际 %d", len(mws))
	}

	// 链式调用
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "handler-reached", nil
	}
	chain := func(ctx context.Context, req interface{}) (interface{}, error) {
		return mws[0](ctx, req, &grpc.UnaryServerInfo{}, func(ctx2 context.Context, req2 interface{}) (interface{}, error) {
			return mws[1](ctx2, req2, &grpc.UnaryServerInfo{}, handler)
		})
	}

	_, err := chain(context.Background(), "input")
	if !errors.Is(err, expectedErr) {
		t.Errorf("错误传播失败, 期望 %v, 实际 %v", expectedErr, err)
	}
}

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		fullMethod string
		expected   string
	}{
		{"/pkg.Svc/Method", "pkg.Svc"},
		{"/com.example.UserService/GetUser", "com.example.UserService"},
		{"pkg.Svc/Method", "pkg.Svc"},
		{"/Svc/Method", "Svc"},
		{"", ""},
		{"/", ""},
		{"NoSlash", "NoSlash"},
	}

	for _, tt := range tests {
		t.Run(tt.fullMethod, func(t *testing.T) {
			result := ExtractServiceName(tt.fullMethod)
			if result != tt.expected {
				t.Errorf("ExtractServiceName(%q) = %q, 期望 %q", tt.fullMethod, result, tt.expected)
			}
		})
	}
}
