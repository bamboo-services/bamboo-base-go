package xGrpc_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	xGrpc "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestP6_GRPCPlugin_RealInterceptorChain_R12(t *testing.T) {
	unknownDetail := &anypb.Any{
		TypeUrl: "type.googleapis.com/thirdparty.AuditTag",
		Value:   []byte{0x08, 0x2a},
	}
	existingBR, _ := anypb.New(&generate.BaseResponse{
		Context: "caller-fixed-trace",
		Output:  string(grpcresult.PermissionDenied.Output),
		Code:    uint64(grpcresult.PermissionDenied.Code),
		Message: "caller explicit br",
	})

	svcDesc := grpc.ServiceDesc{
		ServiceName: "test.P6R12Service",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "UnaryOKStatus",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					return interceptor(ctx, wrapperspb.String("req"), &grpc.UnaryServerInfo{FullMethod: "/test.P6R12Service/UnaryOKStatus"}, func(ctx context.Context, req any) (any, error) {
						return wrapperspb.String("ok-resp"), status.Error(codes.OK, "ignored")
					})
				},
			},
			{
				MethodName: "UnaryTypedNilErr",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					return interceptor(ctx, wrapperspb.String("req"), &grpc.UnaryServerInfo{FullMethod: "/test.P6R12Service/UnaryTypedNilErr"}, func(ctx context.Context, req any) (any, error) {
						var typedNil *apperr.Error
						return wrapperspb.String("ok-typed-nil"), typedNil
					})
				},
			},
			{
				MethodName: "UnaryNativeAbortedMissingBR",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					return interceptor(ctx, wrapperspb.String("req"), &grpc.UnaryServerInfo{FullMethod: "/test.P6R12Service/UnaryNativeAbortedMissingBR"}, func(ctx context.Context, req any) (any, error) {
						raw := &spb.Status{
							Code:    int32(codes.Aborted),
							Message: "tx conflict",
							Details: []*anypb.Any{unknownDetail},
						}
						return nil, fmt.Errorf("secret_wrapper_abort: %w", status.FromProto(raw).Err())
					})
				},
			},
			{
				MethodName: "UnaryNativeExistingBR",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					return interceptor(ctx, wrapperspb.String("req"), &grpc.UnaryServerInfo{FullMethod: "/test.P6R12Service/UnaryNativeExistingBR"}, func(ctx context.Context, req any) (any, error) {
						raw := &spb.Status{
							Code:    int32(codes.PermissionDenied),
							Message: "denied",
							Details: []*anypb.Any{existingBR},
						}
						return nil, status.FromProto(raw).Err()
					})
				},
			},
			{
				MethodName: "UnaryWrappedContextCancel",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					return interceptor(ctx, wrapperspb.String("req"), &grpc.UnaryServerInfo{FullMethod: "/test.P6R12Service/UnaryWrappedContextCancel"}, func(ctx context.Context, req any) (any, error) {
						return nil, fmt.Errorf("secret_db_pwd=123: %w", context.Canceled)
					})
				},
			},
			{
				MethodName: "UnaryPlainUnknown",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					return interceptor(ctx, wrapperspb.String("req"), &grpc.UnaryServerInfo{FullMethod: "/test.P6R12Service/UnaryPlainUnknown"}, func(ctx context.Context, req any) (any, error) {
						return nil, errors.New("secret_db_pwd=123 plain error")
					})
				},
			},
		},
		Streams: []grpc.StreamDesc{
			{
				StreamName: "StreamNativeDataLoss",
				Handler: func(srv any, stream grpc.ServerStream) error {
					return status.Error(codes.DataLoss, "stream data lost")
				},
				ServerStreams: true,
			},
		},
		Metadata: "test_p6_r12.proto",
	}

	_, factory := xGrpc.New(
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithService(func(_ context.Context, s *grpc.Server) {
			s.RegisterService(&svcDesc, struct{}{})
		}),
	)

	plugin := factory().(*xGrpc.Plugin)
	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStart()
	if err := plugin.Start(startCtx); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	addr := plugin.Addr()
	runDone := make(chan *apperr.Error, 1)
	lifetimeCtx, cancelLifetime := context.WithCancel(context.Background())
	defer cancelLifetime()
	go func() {
		runDone <- plugin.Run(lifetimeCtx)
	}()

	t.Cleanup(func() {
		stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelStop()
		_ = plugin.Stop(stopCtx)
		select {
		case <-runDone:
		case <-time.After(3 * time.Second):
			t.Error("Run 未在 Stop 后及时退出")
		}
	})

	// 通过 TCP 握手就绪确认替代固定 sleep
	deadline := time.Now().Add(3 * time.Second)
	for {
		conn, dialErr := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待 gRPC 监听就绪超时: %v", dialErr)
		}
	}

	cc := buildTestClient(t, addr)
	defer func() { _ = cc.Close() }()

	// 1. codes.OK 与 typed-nil *apperr.Error 返回真正成功
	var out wrapperspb.StringValue
	if err := cc.Invoke(context.Background(), "/test.P6R12Service/UnaryOKStatus", wrapperspb.String("hi"), &out); err != nil {
		t.Fatalf("UnaryOKStatus 失败: %v", err)
	}
	if out.GetValue() != "ok-resp" {
		t.Fatalf("UnaryOKStatus 响应不匹配: %q", out.GetValue())
	}
	if err := cc.Invoke(context.Background(), "/test.P6R12Service/UnaryTypedNilErr", wrapperspb.String("hi"), &out); err != nil {
		t.Fatalf("UnaryTypedNilErr 失败: %v", err)
	}
	if out.GetValue() != "ok-typed-nil" {
		t.Fatalf("UnaryTypedNilErr 响应不匹配: %q", out.GetValue())
	}

	// 2. 原生 Aborted 缺 BaseResponse -> 补齐 40907 信封且保持 Aborted 与未知 Any 顺序、不泄外层文本
	callCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(xGrpcConst.MetadataRequestUUID.String(), "p6-req-uuid"))
	err := cc.Invoke(callCtx, "/test.P6R12Service/UnaryNativeAbortedMissingBR", wrapperspb.String("hi"), &out)
	st := status.Convert(err)
	if st.Code() != codes.Aborted || st.Message() != "tx conflict" {
		t.Fatalf("Aborted 原状态或文案被改写: code=%v msg=%q", st.Code(), st.Message())
	}
	if strings.Contains(st.Message(), "secret_wrapper_abort") {
		t.Fatalf("泄露外层包装文本: %q", st.Message())
	}
	details := st.Proto().GetDetails()
	if len(details) != 2 || !proto.Equal(details[0], unknownDetail) {
		t.Fatalf("未知 Any 未原样保序保留: len=%d", len(details))
	}
	var br generate.BaseResponse
	if uErr := details[1].UnmarshalTo(&br); uErr != nil {
		t.Fatalf("解码补齐 BaseResponse 失败: %v", uErr)
	}
	if br.Code != uint64(grpcresult.Aborted.Code) || br.Context != "p6-req-uuid" {
		t.Fatalf("补齐 BaseResponse 错误: %+v", &br)
	}

	// 3. 已有 BaseResponse -> 不覆盖 Context、不重复追加
	err = cc.Invoke(callCtx, "/test.P6R12Service/UnaryNativeExistingBR", wrapperspb.String("hi"), &out)
	st = status.Convert(err)
	if st.Code() != codes.PermissionDenied || len(st.Proto().GetDetails()) != 1 {
		t.Fatalf("已有 BaseResponse 状态或条数错误: code=%v len=%d", st.Code(), len(st.Proto().GetDetails()))
	}
	var gotBR generate.BaseResponse
	if uErr := st.Proto().GetDetails()[0].UnmarshalTo(&gotBR); uErr != nil {
		t.Fatal(uErr)
	}
	if gotBR.Context != "caller-fixed-trace" {
		t.Fatalf("已有 BaseResponse Context 被改写: %q", gotBR.Context)
	}

	// 4. context.Canceled 包装 -> 49900 信封，不泄秘密
	err = cc.Invoke(callCtx, "/test.P6R12Service/UnaryWrappedContextCancel", wrapperspb.String("hi"), &out)
	st = status.Convert(err)
	if st.Code() != codes.Canceled || strings.Contains(st.Message(), "secret_db_pwd") {
		t.Fatalf("Canceled 状态或文案异常: code=%v msg=%q", st.Code(), st.Message())
	}
	if len(st.Proto().GetDetails()) != 1 {
		t.Fatalf("Canceled 缺少 BaseResponse: len=%d", len(st.Proto().GetDetails()))
	}
	if uErr := st.Proto().GetDetails()[0].UnmarshalTo(&gotBR); uErr != nil {
		t.Fatal(uErr)
	}
	if gotBR.Code != uint64(control.Canceled.Code) || gotBR.Context != "p6-req-uuid" {
		t.Fatalf("Canceled BaseResponse 错误: %+v", &gotBR)
	}

	// 5. 未知普通错误 -> 50011 安全脱敏
	err = cc.Invoke(callCtx, "/test.P6R12Service/UnaryPlainUnknown", wrapperspb.String("hi"), &out)
	st = status.Convert(err)
	if st.Code() != codes.Internal || strings.Contains(st.Message(), "secret_db_pwd") {
		t.Fatalf("未知错误状态或脱敏失败: code=%v msg=%q", st.Code(), st.Message())
	}

	// 6. Stream 拦截器原生 DataLoss -> 保持 DataLoss 且补齐 50025 信封
	stream, err := cc.NewStream(callCtx, &svcDesc.Streams[0], "/test.P6R12Service/StreamNativeDataLoss")
	if err != nil {
		t.Fatalf("NewStream 失败: %v", err)
	}
	var streamOut wrapperspb.StringValue
	recvErr := stream.RecvMsg(&streamOut)
	st = status.Convert(recvErr)
	if st.Code() != codes.DataLoss || st.Message() != "stream data lost" {
		t.Fatalf("Stream DataLoss 被改写: code=%v msg=%q", st.Code(), st.Message())
	}
	if len(st.Proto().GetDetails()) != 1 {
		t.Fatalf("Stream DataLoss 未补齐 BaseResponse")
	}
	if uErr := st.Proto().GetDetails()[0].UnmarshalTo(&gotBR); uErr != nil {
		t.Fatal(uErr)
	}
	if gotBR.Code != uint64(grpcresult.DataLoss.Code) || gotBR.Context != "p6-req-uuid" {
		t.Fatalf("Stream DataLoss BaseResponse 错误: %+v", &gotBR)
	}
}
