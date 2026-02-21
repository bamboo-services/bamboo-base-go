package xGrpcIStream

import (
	"context"
	"strings"
	"testing"
	"time"

	xCtx "github.com/bamboo-services/bamboo-base-go/context"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// mockServerStream 实现 grpc.ServerStream 接口用于测试。
type mockServerStream struct {
	ctx     context.Context
	header  metadata.MD
	trailer metadata.MD
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SetHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	return nil
}

func (m *mockServerStream) SendHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	return nil
}

func (m *mockServerStream) SetTrailer(md metadata.MD) {
	m.trailer = metadata.Join(m.trailer, md)
}

func (m *mockServerStream) SendMsg(interface{}) error { return nil }
func (m *mockServerStream) RecvMsg(interface{}) error { return nil }

func TestTraceGenerateRequestUUID(t *testing.T) {
	ctx := context.Background()
	ss := &mockServerStream{ctx: ctx}
	info := &grpc.StreamServerInfo{FullMethod: "/x.Base/Test"}
	requestUUID := ""

	interceptor := Trace()
	err := interceptor(nil, ss, info, func(srv interface{}, stream grpc.ServerStream) error {
		requestUUID = xCtxUtil.GetRequestKey(stream.Context())
		if requestUUID == "" {
			t.Fatalf("request uuid should not be empty")
		}

		startTimeValue := stream.Context().Value(xCtx.UserStartTimeKey)
		startTime, ok := startTimeValue.(time.Time)
		if !ok {
			t.Fatalf("start time should be injected")
		}
		if startTime.IsZero() {
			t.Fatalf("start time should not be zero")
		}

		headerValues := ss.header.Get("x-request-uuid")
		if len(headerValues) == 0 || headerValues[0] != requestUUID {
			t.Fatalf("header x-request-uuid should equal context request uuid")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("trace interceptor should not return error: %v", err)
	}

	trailerValues := ss.trailer.Get("x-request-uuid")
	if len(trailerValues) == 0 || trailerValues[0] != requestUUID {
		t.Fatalf("trailer x-request-uuid should equal context request uuid")
	}
}

func TestTraceReuseIncomingRequestUUID(t *testing.T) {
	expected := "custom-request-id"
	incomingMD := metadata.Pairs("x-request-uuid", expected)
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)
	ss := &mockServerStream{ctx: ctx}
	info := &grpc.StreamServerInfo{FullMethod: "/x.Base/Test"}

	interceptor := Trace()
	err := interceptor(nil, ss, info, func(srv interface{}, stream grpc.ServerStream) error {
		requestUUID := xCtxUtil.GetRequestKey(stream.Context())
		if requestUUID != expected {
			t.Fatalf("request uuid should reuse incoming constant value")
		}
		if strings.TrimSpace(requestUUID) == "" {
			t.Fatalf("request uuid should not be blank")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("trace interceptor should not return error: %v", err)
	}
}

func TestInitContext(t *testing.T) {
	ctxNodeList := xCtx.ContextNodeList{
		{Key: xCtx.DatabaseKey, Value: "test-db"},
	}
	mainCtx := context.WithValue(context.Background(), xCtx.RegNodeKey, ctxNodeList)

	ss := &mockServerStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{FullMethod: "/x.Base/Test"}

	interceptor := InitContext(mainCtx)
	err := interceptor(nil, ss, info, func(srv interface{}, stream grpc.ServerStream) error {
		val := stream.Context().Value(xCtx.RegNodeKey)
		if val == nil {
			t.Fatalf("RegNodeKey should be injected")
		}
		nodeList, ok := val.(xCtx.ContextNodeList)
		if !ok {
			t.Fatalf("RegNodeKey value should be ContextNodeList")
		}
		if len(nodeList) != 1 {
			t.Fatalf("injected nodeList should have 1 element")
		}
		if nodeList[0].Key != xCtx.DatabaseKey || nodeList[0].Value != "test-db" {
			t.Fatalf("injected nodeList should contain test-db")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("init context interceptor should not return error: %v", err)
	}
}
