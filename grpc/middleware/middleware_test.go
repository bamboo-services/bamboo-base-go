package xGrpcMiddle

import (
	"context"
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
	t.Cleanup(reset)

	var recorder []string
	UseUnary(grpc.ServiceDesc{ServiceName: "pkg.Svc"},
		dummyUnaryInterceptor("A", &recorder),
		nil,
		dummyUnaryInterceptor("B", &recorder),
	)

	mws := LookupUnary("pkg.Svc")
	if len(mws) != 2 {
		t.Errorf("有效中间件数量 = %d, 期望 2", len(mws))
	}
}

func TestUseUnaryAppend(t *testing.T) {
	t.Cleanup(reset)

	var recorder []string
	desc := grpc.ServiceDesc{ServiceName: "pkg.Svc"}
	UseUnary(desc, dummyUnaryInterceptor("A", &recorder))
	UseUnary(desc, dummyUnaryInterceptor("B", &recorder))

	mws := LookupUnary("pkg.Svc")
	if len(mws) != 2 {
		t.Errorf("追加后中间件数量 = %d, 期望 2", len(mws))
	}
}

func TestUseStreamBasic(t *testing.T) {
	t.Cleanup(reset)

	var recorder []string
	UseStream(grpc.ServiceDesc{ServiceName: "pkg.StreamSvc"},
		dummyStreamInterceptor("A", &recorder),
		nil,
		dummyStreamInterceptor("B", &recorder),
	)

	mws := LookupStream("pkg.StreamSvc")
	if len(mws) != 2 {
		t.Errorf("有效中间件数量 = %d, 期望 2", len(mws))
	}
}

func TestUseStreamAppend(t *testing.T) {
	t.Cleanup(reset)

	var recorder []string
	desc := grpc.ServiceDesc{ServiceName: "pkg.StreamSvc"}
	UseStream(desc, dummyStreamInterceptor("A", &recorder))
	UseStream(desc, dummyStreamInterceptor("B", &recorder))

	mws := LookupStream("pkg.StreamSvc")
	if len(mws) != 2 {
		t.Errorf("追加后中间件数量 = %d, 期望 2", len(mws))
	}
}

func TestUseMultipleServices(t *testing.T) {
	t.Cleanup(reset)

	var recorder []string
	UseUnary(grpc.ServiceDesc{ServiceName: "pkg.A"}, dummyUnaryInterceptor("A1", &recorder))
	UseUnary(grpc.ServiceDesc{ServiceName: "pkg.B"}, dummyUnaryInterceptor("B1", &recorder), dummyUnaryInterceptor("B2", &recorder))
	UseStream(grpc.ServiceDesc{ServiceName: "pkg.StreamA"}, dummyStreamInterceptor("SA1", &recorder))

	if mws := LookupUnary("pkg.A"); len(mws) != 1 {
		t.Errorf("pkg.A 中间件数量 = %d, 期望 1", len(mws))
	}
	if mws := LookupUnary("pkg.B"); len(mws) != 2 {
		t.Errorf("pkg.B 中间件数量 = %d, 期望 2", len(mws))
	}
	if mws := LookupUnary("pkg.C"); mws != nil {
		t.Errorf("未注册的 pkg.C 应返回 nil, 实际 = %v", mws)
	}
	if mws := LookupStream("pkg.StreamA"); len(mws) != 1 {
		t.Errorf("pkg.StreamA 中间件数量 = %d, 期望 1", len(mws))
	}
	if mws := LookupStream("pkg.StreamB"); mws != nil {
		t.Errorf("未注册的 pkg.StreamB 应返回 nil, 实际 = %v", mws)
	}
}

func TestUseAllNilSkipped(t *testing.T) {
	t.Cleanup(reset)

	UseUnary(grpc.ServiceDesc{ServiceName: "pkg.Empty"}, nil, nil)
	if mws := LookupUnary("pkg.Empty"); mws != nil {
		t.Errorf("全 nil 注册后 LookupUnary 应返回 nil, 实际长度 = %d", len(mws))
	}

	UseStream(grpc.ServiceDesc{ServiceName: "pkg.StreamEmpty"}, nil, nil)
	if mws := LookupStream("pkg.StreamEmpty"); mws != nil {
		t.Errorf("全 nil 注册后 LookupStream 应返回 nil, 实际长度 = %d", len(mws))
	}
}

func TestLookupEmpty(t *testing.T) {
	t.Cleanup(reset)

	if mws := LookupUnary("any.Service"); mws != nil {
		t.Errorf("空注册表 LookupUnary 应返回 nil, 实际 = %v", mws)
	}
	if mws := LookupStream("any.Service"); mws != nil {
		t.Errorf("空注册表 LookupStream 应返回 nil, 实际 = %v", mws)
	}
}

func TestConcurrentSafe(t *testing.T) {
	t.Cleanup(reset)

	var wg sync.WaitGroup

	// 并发写入 Unary
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var recorder []string
			UseUnary(
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
			UseStream(
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
			_ = LookupUnary("pkg.Concurrent")
			_ = LookupStream("pkg.StreamConcurrent")
		}()
	}

	wg.Wait()

	mws := LookupUnary("pkg.Concurrent")
	if len(mws) != 100 {
		t.Errorf("并发注册后 Unary 中间件数量 = %d, 期望 100", len(mws))
	}

	smws := LookupStream("pkg.StreamConcurrent")
	if len(smws) != 100 {
		t.Errorf("并发注册后 Stream 中间件数量 = %d, 期望 100", len(smws))
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
