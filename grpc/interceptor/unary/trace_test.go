package xGrpcIUnary

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

type mockServerTransportStream struct {
	method  string
	header  metadata.MD
	trailer metadata.MD
}

func (m *mockServerTransportStream) Method() string {
	return m.method
}

func (m *mockServerTransportStream) SetHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	return nil
}

func (m *mockServerTransportStream) SendHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	return nil
}

func (m *mockServerTransportStream) SetTrailer(md metadata.MD) error {
	m.trailer = metadata.Join(m.trailer, md)
	return nil
}

func TestTraceGenerateRequestUUID(t *testing.T) {
	stream := &mockServerTransportStream{method: "/x.Base/Test"}
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), stream)
	requestUUID := ""

	interceptor := Trace()
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: stream.method}, func(handlerCtx context.Context, req interface{}) (interface{}, error) {
		requestUUID = xCtxUtil.GetRequestKey(handlerCtx)
		if requestUUID == "" {
			t.Fatalf("request uuid should not be empty")
		}

		startTimeValue := handlerCtx.Value(xCtx.UserStartTimeKey)
		startTime, ok := startTimeValue.(time.Time)
		if !ok {
			t.Fatalf("start time should be injected")
		}
		if startTime.IsZero() {
			t.Fatalf("start time should not be zero")
		}

		headerValues := stream.header.Get("x-request-uuid")
		if len(headerValues) == 0 || headerValues[0] != requestUUID {
			t.Fatalf("header x-request-uuid should equal context request uuid")
		}

		return nil, nil
	})
	if err != nil {
		t.Fatalf("trace interceptor should not return error: %v", err)
	}

	trailerValues := stream.trailer.Get("x-request-uuid")
	if len(trailerValues) == 0 || trailerValues[0] != requestUUID {
		t.Fatalf("trailer x-request-uuid should equal context request uuid")
	}
}

func TestTraceReuseIncomingRequestUUID(t *testing.T) {
	expected := "custom-request-id"
	stream := &mockServerTransportStream{method: "/x.Base/Test"}

	incomingMD := metadata.Pairs("x-request-uuid", expected)
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)
	ctx = grpc.NewContextWithServerTransportStream(ctx, stream)

	interceptor := Trace()
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: stream.method}, func(handlerCtx context.Context, req interface{}) (interface{}, error) {
		requestUUID := xCtxUtil.GetRequestKey(handlerCtx)
		if requestUUID != expected {
			t.Fatalf("request uuid should reuse incoming constant value")
		}
		if strings.TrimSpace(requestUUID) == "" {
			t.Fatalf("request uuid should not be blank")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("trace interceptor should not return error: %v", err)
	}
}
