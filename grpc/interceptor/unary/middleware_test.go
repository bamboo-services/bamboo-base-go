package xGrpcIUnary

import (
	"context"
	"errors"
	"testing"

	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/grpc/middleware"
	"google.golang.org/grpc"
)

// tagInterceptor 创建一个记录进出顺序的拦截器。
func tagInterceptor(tag string, recorder *[]string) xGrpcMiddle.UnaryMiddlewareFunc {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		*recorder = append(*recorder, tag+"-enter")
		resp, err := handler(ctx, req)
		*recorder = append(*recorder, tag+"-exit")
		return resp, err
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
			result := extractServiceName(tt.fullMethod)
			if result != tt.expected {
				t.Errorf("extractServiceName(%q) = %q, 期望 %q", tt.fullMethod, result, tt.expected)
			}
		})
	}
}

func TestMiddlewareNoMatchPassthrough(t *testing.T) {
	// 注册一个不相关的服务
	xGrpcMiddle.UseUnary(grpc.ServiceDesc{ServiceName: "pkg.Other"}, tagInterceptor("A", &[]string{}))

	interceptor := Middleware()

	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "passthrough", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Unmatched/Method"}

	resp, err := interceptor(context.Background(), "req", info, handler)
	if err != nil {
		t.Fatalf("未匹配服务不应产生错误: %v", err)
	}
	if !called {
		t.Error("handler 应被直接调用")
	}
	if resp != "passthrough" {
		t.Errorf("resp = %v, 期望 %q", resp, "passthrough")
	}
}

func TestMiddlewareOnionOrder(t *testing.T) {
	var recorder []string
	xGrpcMiddle.UseUnary(grpc.ServiceDesc{ServiceName: "test.onion.unary.Svc"},
		tagInterceptor("A", &recorder),
		tagInterceptor("B", &recorder),
		tagInterceptor("C", &recorder),
	)

	interceptor := Middleware()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		recorder = append(recorder, "handler")
		return "result", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.onion.unary.Svc/DoSomething"}

	resp, err := interceptor(context.Background(), "req", info, handler)
	if err != nil {
		t.Fatalf("洋葱模型执行不应产生错误: %v", err)
	}
	if resp != "result" {
		t.Errorf("resp = %v, 期望 %q", resp, "result")
	}

	expected := []string{"A-enter", "B-enter", "C-enter", "handler", "C-exit", "B-exit", "A-exit"}
	if len(recorder) != len(expected) {
		t.Fatalf("执行记录长度 = %d, 期望 %d\n记录: %v", len(recorder), len(expected), recorder)
	}
	for i, v := range expected {
		if recorder[i] != v {
			t.Errorf("recorder[%d] = %q, 期望 %q\n完整记录: %v", i, recorder[i], v, recorder)
			break
		}
	}
}

func TestMiddlewareErrorPropagation(t *testing.T) {
	var recorder []string
	expectedErr := errors.New("handler error")
	xGrpcMiddle.UseUnary(grpc.ServiceDesc{ServiceName: "test.error.unary.Svc"},
		tagInterceptor("A", &recorder),
	)

	interceptor := Middleware()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		recorder = append(recorder, "handler")
		return nil, expectedErr
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.error.unary.Svc/Method"}

	resp, err := interceptor(context.Background(), "req", info, handler)
	if !errors.Is(err, expectedErr) {
		t.Errorf("错误应透传, 实际 err = %v", err)
	}
	if resp != nil {
		t.Errorf("resp 应为 nil, 实际 = %v", resp)
	}

	expected := []string{"A-enter", "handler", "A-exit"}
	if len(recorder) != len(expected) {
		t.Fatalf("执行记录长度 = %d, 期望 %d\n记录: %v", len(recorder), len(expected), recorder)
	}
	for i, v := range expected {
		if recorder[i] != v {
			t.Errorf("recorder[%d] = %q, 期望 %q", i, recorder[i], v)
			break
		}
	}
}

func TestMiddlewareContextPropagation(t *testing.T) {
	type ctxKey struct{}
	xGrpcMiddle.UseUnary(grpc.ServiceDesc{ServiceName: "test.ctx.unary.Svc"},
		func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(context.WithValue(ctx, ctxKey{}, "injected"), req)
		},
	)

	interceptor := Middleware()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		val, ok := ctx.Value(ctxKey{}).(string)
		if !ok || val != "injected" {
			return nil, errors.New("上下文未正确传播")
		}
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.ctx.unary.Svc/Method"}

	resp, err := interceptor(context.Background(), "req", info, handler)
	if err != nil {
		t.Fatalf("上下文传播失败: %v", err)
	}
	if resp != "ok" {
		t.Errorf("resp = %v, 期望 %q", resp, "ok")
	}
}

func TestMiddlewareSingleMiddleware(t *testing.T) {
	var recorder []string
	xGrpcMiddle.UseUnary(grpc.ServiceDesc{ServiceName: "test.single.unary.Svc"},
		tagInterceptor("Only", &recorder),
	)

	interceptor := Middleware()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		recorder = append(recorder, "handler")
		return "done", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.single.unary.Svc/Method"}

	resp, err := interceptor(context.Background(), "req", info, handler)
	if err != nil {
		t.Fatalf("单中间件执行不应产生错误: %v", err)
	}
	if resp != "done" {
		t.Errorf("resp = %v, 期望 %q", resp, "done")
	}

	expected := []string{"Only-enter", "handler", "Only-exit"}
	if len(recorder) != len(expected) {
		t.Fatalf("执行记录长度 = %d, 期望 %d\n记录: %v", len(recorder), len(expected), recorder)
	}
	for i, v := range expected {
		if recorder[i] != v {
			t.Errorf("recorder[%d] = %q, 期望 %q", i, recorder[i], v)
			break
		}
	}
}
