package xGrpcIStream

import (
	"context"
	"errors"
	"testing"

	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/grpc/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// mockServerStream 实现 grpc.ServerStream 接口用于测试。
type testServerStream struct {
	ctx context.Context
}

func (m *testServerStream) Context() context.Context     { return m.ctx }
func (m *testServerStream) SetHeader(metadata.MD) error  { return nil }
func (m *testServerStream) SendHeader(metadata.MD) error { return nil }
func (m *testServerStream) SetTrailer(metadata.MD)       {}
func (m *testServerStream) SendMsg(interface{}) error    { return nil }
func (m *testServerStream) RecvMsg(interface{}) error    { return nil }

// tagStreamInterceptor 创建一个记录进出顺序的拦截器。
func tagStreamInterceptor(tag string, recorder *[]string) xGrpcMiddle.StreamMiddlewareFunc {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		*recorder = append(*recorder, tag+"-enter")
		err := handler(srv, ss)
		*recorder = append(*recorder, tag+"-exit")
		return err
	}
}

func TestMiddlewareNoMatchPassthrough(t *testing.T) {
	// 注册一个不相关的服务
	xGrpcMiddle.UseStream(grpc.ServiceDesc{ServiceName: "pkg.Other"}, tagStreamInterceptor("A", &[]string{}))

	interceptor := Middleware()

	called := false
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		called = true
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/pkg.Unmatched/Method"}
	ss := &testServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("未匹配服务不应产生错误: %v", err)
	}
	if !called {
		t.Error("handler 应被直接调用")
	}
}

func TestMiddlewareOnionOrder(t *testing.T) {
	var recorder []string
	xGrpcMiddle.UseStream(grpc.ServiceDesc{ServiceName: "test.onion.stream.Svc"},
		tagStreamInterceptor("A", &recorder),
		tagStreamInterceptor("B", &recorder),
		tagStreamInterceptor("C", &recorder),
	)

	interceptor := Middleware()
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		recorder = append(recorder, "handler")
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.onion.stream.Svc/DoSomething"}
	ss := &testServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("洋葱模型执行不应产生错误: %v", err)
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
	xGrpcMiddle.UseStream(grpc.ServiceDesc{ServiceName: "test.error.stream.Svc"},
		tagStreamInterceptor("A", &recorder),
	)

	interceptor := Middleware()
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		recorder = append(recorder, "handler")
		return expectedErr
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.error.stream.Svc/Method"}
	ss := &testServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, info, handler)
	if !errors.Is(err, expectedErr) {
		t.Errorf("错误应透传, 实际 err = %v", err)
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
	xGrpcMiddle.UseStream(grpc.ServiceDesc{ServiceName: "test.ctx.stream.Svc"},
		func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			wrapped := &wrappedServerStream{ServerStream: ss, ctx: context.WithValue(ss.Context(), ctxKey{}, "injected")}
			return handler(srv, wrapped)
		},
	)

	interceptor := Middleware()
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		val, ok := ss.Context().Value(ctxKey{}).(string)
		if !ok || val != "injected" {
			return errors.New("上下文未正确传播")
		}
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.ctx.stream.Svc/Method"}
	ss := &testServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("上下文传播失败: %v", err)
	}
}

func TestMiddlewareSingleMiddleware(t *testing.T) {
	var recorder []string
	xGrpcMiddle.UseStream(grpc.ServiceDesc{ServiceName: "test.single.stream.Svc"},
		tagStreamInterceptor("Only", &recorder),
	)

	interceptor := Middleware()
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		recorder = append(recorder, "handler")
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.single.stream.Svc/Method"}
	ss := &testServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("单中间件执行不应产生错误: %v", err)
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
