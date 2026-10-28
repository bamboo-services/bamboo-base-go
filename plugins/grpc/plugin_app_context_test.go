package xGrpc_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	xGrpc "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type customCtxKeyType struct{}

var customCtxKey customCtxKeyType

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

var errSentinelBiz = errors.New("sentinel_biz_error")

type testAppContextServiceServer interface {
	unaryContextCheckHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error)
	unaryBlockUntilCancelHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error)
	unaryErrorCheckHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error)
	streamContextCheckHandler(srv any, ss grpc.ServerStream) error
	streamErrorCheckHandler(srv any, ss grpc.ServerStream) error
}

type testServerAppFixture struct {
	t        *testing.T
	app      *application.App
	resKey   ctxkey.Key
	compKey  ctxkey.Key
	grpcKey  ctxkey.Key
	actCh    chan application.InstanceMeta
	cancelCh chan struct{}
}

func newTestAppServiceDesc(srv *testServerAppFixture) grpc.ServiceDesc {
	return grpc.ServiceDesc{
		ServiceName: "test.AppContextService",
		HandlerType: (*testAppContextServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "UnaryContextCheck",
				Handler:    srv.unaryContextCheckHandler,
			},
			{
				MethodName: "UnaryBlockUntilCancel",
				Handler:    srv.unaryBlockUntilCancelHandler,
			},
			{
				MethodName: "UnaryErrorCheck",
				Handler:    srv.unaryErrorCheckHandler,
			},
			{
				MethodName: "UnaryNoOutput",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					req := new(wrapperspb.StringValue)
					if err := dec(req); err != nil {
						return nil, err
					}
					h := func(context.Context, any) (any, error) { return nil, nil }
					if interceptor == nil {
						return h(ctx, req)
					}
					return interceptor(ctx, req, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/test.AppContextService/UnaryNoOutput"}, h)
				},
			},
		},
		Streams: []grpc.StreamDesc{
			{
				StreamName:    "StreamContextCheck",
				Handler:       srv.streamContextCheckHandler,
				ServerStreams: true,
				ClientStreams: true,
			},
			{
				StreamName:    "StreamErrorCheck",
				Handler:       srv.streamErrorCheckHandler,
				ServerStreams: true,
				ClientStreams: true,
			},
		},
	}
}

func (s *testServerAppFixture) unaryContextCheckHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	req := new(wrapperspb.StringValue)
	if err := dec(req); err != nil {
		return nil, err
	}
	h := func(ctx context.Context, req any) (any, error) {
		resVal, appErr := application.Get[string](ctx, s.resKey)
		if appErr != nil {
			return nil, appErr
		}

		st, hasState := application.StateFrom(ctx)
		if !hasState || st != application.StateRunning {
			return nil, fmt.Errorf("unexpected StateFrom: %v, hasState=%v", st, hasState)
		}

		reqKey := ctxkey.GetRequestKey(ctx)
		if reqKey == "" {
			return nil, errors.New("missing request key")
		}

		meta, hasMeta := application.InstanceMetaFrom(ctx)
		if !hasMeta {
			return nil, errors.New("missing instance meta from unary ctx")
		}
		if meta.PluginKey != s.grpcKey {
			return nil, fmt.Errorf("expected parent PluginKey %s, got %s", s.grpcKey, meta.PluginKey)
		}

		act := s.app.Activator()
		if act != nil {
			h, actErr := act.Activate(ctx, s.compKey)
			if actErr != nil {
				return nil, fmt.Errorf("activate component failed: %w", actErr)
			}
			select {
			case childMeta := <-s.actCh:
				if childMeta.ParentInstanceID != meta.InstanceID {
					return nil, fmt.Errorf("child ParentInstanceID mismatch: expected %s, got %s", meta.InstanceID, childMeta.ParentInstanceID)
				}
				if childMeta.ParentKey != s.grpcKey {
					return nil, fmt.Errorf("child ParentKey mismatch: expected %s, got %s", s.grpcKey, childMeta.ParentKey)
				}
			case <-time.After(2 * time.Second):
				return nil, errors.New("timed out waiting for child component execution")
			}
			outcome, waitErr := h.Wait(ctx)
			if waitErr != nil {
				return nil, fmt.Errorf("wait child component failed: %w", waitErr)
			}
			if outcome.Outcome != application.OutcomeSuccess {
				return nil, fmt.Errorf("child component outcome not success: %v", outcome.Outcome)
			}
		}

		return wrapperspb.String(resVal + ":" + reqKey), nil
	}

	if interceptor != nil {
		return interceptor(ctx, req, &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/test.AppContextService/UnaryContextCheck",
		}, h)
	}
	return h(ctx, req)
}

func (s *testServerAppFixture) unaryBlockUntilCancelHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	req := new(wrapperspb.StringValue)
	if err := dec(req); err != nil {
		return nil, err
	}
	h := func(ctx context.Context, req any) (any, error) {
		close(s.cancelCh)
		<-ctx.Done()
		return wrapperspb.String("canceled"), ctx.Err()
	}
	if interceptor != nil {
		return interceptor(ctx, req, &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/test.AppContextService/UnaryBlockUntilCancel",
		}, h)
	}
	return h(ctx, req)
}

func (s *testServerAppFixture) unaryErrorCheckHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	req := new(wrapperspb.StringValue)
	if err := dec(req); err != nil {
		return nil, err
	}
	h := func(ctx context.Context, req any) (any, error) {
		r := req.(*wrapperspb.StringValue).GetValue()
		if r == "sentinel" {
			return nil, errSentinelBiz
		}
		if r == "status" {
			return nil, status.Error(codes.FailedPrecondition, "custom_failed_precondition")
		}
		return wrapperspb.String("ok"), nil
	}
	if interceptor != nil {
		return interceptor(ctx, req, &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/test.AppContextService/UnaryErrorCheck",
		}, h)
	}
	return h(ctx, req)
}

func (s *testServerAppFixture) streamContextCheckHandler(srv any, ss grpc.ServerStream) error {
	ctx := ss.Context()
	resVal, appErr := application.Get[string](ctx, s.resKey)
	if appErr != nil {
		return appErr
	}

	st, hasState := application.StateFrom(ctx)
	if !hasState || st != application.StateRunning {
		return fmt.Errorf("unexpected StateFrom in stream: %v", st)
	}

	reqKey := ctxkey.GetRequestKey(ctx)
	if reqKey == "" {
		return errors.New("missing request key in stream")
	}

	meta, hasMeta := application.InstanceMetaFrom(ctx)
	if !hasMeta {
		return errors.New("missing instance meta in stream")
	}
	if meta.PluginKey != s.grpcKey {
		return fmt.Errorf("expected parent PluginKey %s, got %s", s.grpcKey, meta.PluginKey)
	}

	act := s.app.Activator()
	if act != nil {
		h, actErr := act.Activate(ctx, s.compKey)
		if actErr != nil {
			return fmt.Errorf("stream activate component failed: %w", actErr)
		}
		select {
		case childMeta := <-s.actCh:
			if childMeta.ParentInstanceID != meta.InstanceID {
				return fmt.Errorf("stream child ParentInstanceID mismatch: expected %s, got %s", meta.InstanceID, childMeta.ParentInstanceID)
			}
		case <-time.After(2 * time.Second):
			return errors.New("timed out waiting for child component in stream")
		}
		outcome, waitErr := h.Wait(ctx)
		if waitErr != nil {
			return fmt.Errorf("stream wait child component failed: %w", waitErr)
		}
		if outcome.Outcome != application.OutcomeSuccess {
			return fmt.Errorf("stream child component outcome not success: %v", outcome.Outcome)
		}
	}

	for {
		var in wrapperspb.StringValue
		err := ss.RecvMsg(&in)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = ss.SendMsg(wrapperspb.String(resVal + ":" + in.GetValue() + ":" + reqKey))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *testServerAppFixture) streamErrorCheckHandler(srv any, ss grpc.ServerStream) error {
	var in wrapperspb.StringValue
	if err := ss.RecvMsg(&in); err != nil {
		return err
	}
	if in.GetValue() == "sentinel" {
		return errSentinelBiz
	}
	return status.Error(codes.FailedPrecondition, "custom_stream_failed_precondition")
}

type toolResourcePlugin struct {
	resKey ctxkey.Key
	val    string
}

func (t *toolResourcePlugin) Start(ctx context.Context) error {
	return nil
}

func (t *toolResourcePlugin) Stop(ctx context.Context) error {
	return nil
}

func (t *toolResourcePlugin) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		t.resKey: t.val,
	}
}

type witnessPlugin struct {
	readyCh chan struct{}
}

func (w *witnessPlugin) Start(ctx context.Context) error {
	return nil
}

func (w *witnessPlugin) Run(ctx context.Context) error {
	close(w.readyCh)
	<-ctx.Done()
	return nil
}

func (w *witnessPlugin) Stop(ctx context.Context) error {
	return nil
}

type dummyComponent struct {
	actCh chan application.InstanceMeta
}

func (d *dummyComponent) Start(ctx context.Context) error {
	return nil
}

func (d *dummyComponent) Run(ctx context.Context) error {
	meta, _ := application.InstanceMetaFrom(ctx)
	select {
	case d.actCh <- meta:
	default:
	}
	return nil
}

func (d *dummyComponent) Stop(ctx context.Context) error {
	return nil
}

func createTestApp(
	t *testing.T,
	appSuffix string,
	toolVal string,
	reg *xGrpcMiddle.Registry,
) (
	app *application.App,
	grpcPlugin *xGrpc.Plugin,
	fix *testServerAppFixture,
	resKey ctxkey.Key,
	grpcKey ctxkey.Key,
	readyCh chan struct{},
	errCh chan error,
) {
	app = application.New(
		application.WithGraceTimeout(10*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	resKey = ctxkey.Key("res_" + appSuffix)
	toolKey := ctxkey.Key("tool_" + appSuffix)
	grpcKey = ctxkey.Key("grpc_worker_" + appSuffix)
	witnessKey := ctxkey.Key("witness_" + appSuffix)
	compKey := ctxkey.Key("comp_" + appSuffix)

	actCh := make(chan application.InstanceMeta, 10)
	fix = &testServerAppFixture{
		t:        t,
		app:      app,
		resKey:   resKey,
		compKey:  compKey,
		grpcKey:  grpcKey,
		actCh:    actCh,
		cancelCh: make(chan struct{}),
	}

	_ = app.RegisterTool(application.PluginInfo{
		Key:      toolKey,
		Name:     "Tool_" + appSuffix,
		Provides: []ctxkey.Key{resKey},
	}, func() application.Plugin {
		return &toolResourcePlugin{resKey: resKey, val: toolVal}
	})

	_ = app.RegisterComponent(application.PluginInfo{
		Key:      compKey,
		Name:     "Comp_" + appSuffix,
		Requires: []ctxkey.Key{resKey},
	}, func() application.RunnablePlugin {
		return &dummyComponent{actCh: actCh}
	})

	var localPlugin *xGrpc.Plugin
	grpcInfo, grpcFac := xGrpc.New(
		xGrpc.WithKey(grpcKey),
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithRequires(resKey),
		xGrpc.WithMiddlewareRegistry(reg),
		xGrpc.WithService(func(startCtx context.Context, server *grpc.Server) {
			_, err := application.Get[string](startCtx, resKey)
			if err != nil {
				panic(fmt.Sprintf("Start callback Get %s failed: %v", resKey, err))
			}
			desc := newTestAppServiceDesc(fix)
			server.RegisterService(&desc, fix)
		}),
	)

	_ = app.RegisterWorker(grpcInfo, func() application.RunnablePlugin {
		p := grpcFac().(*xGrpc.Plugin)
		localPlugin = p
		return p
	})

	readyCh = make(chan struct{})
	_ = app.RegisterWorker(application.PluginInfo{
		Key:      witnessKey,
		Name:     "Witness_" + appSuffix,
		Requires: []ctxkey.Key{grpcKey},
	}, func() application.RunnablePlugin {
		return &witnessPlugin{readyCh: readyCh}
	})

	errCh = make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	select {
	case <-readyCh:
	case err := <-errCh:
		t.Fatalf("App %s failed to run: %v", appSuffix, err)
	case <-time.After(5 * time.Second):
		t.Fatalf("App %s start timeout waiting for witness ready", appSuffix)
	}

	return app, localPlugin, fix, resKey, grpcKey, readyCh, errCh
}

func TestPlugin_AppContext_RealTCP_IsolationAndRegistry(t *testing.T) {
	var (
		mu1          sync.Mutex
		callOrder1   []string
		customValue1 string
	)
	reg1 := xGrpcMiddle.NewRegistry()
	desc := grpc.ServiceDesc{ServiceName: "test.AppContextService"}

	reg1.UseUnary(desc, func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		mu1.Lock()
		callOrder1 = append(callOrder1, "u1_outer_before")
		mu1.Unlock()
		ctx = context.WithValue(ctx, customCtxKey, "val_appA")
		r, e := handler(ctx, req)
		mu1.Lock()
		callOrder1 = append(callOrder1, "u1_outer_after")
		mu1.Unlock()
		return r, e
	})
	reg1.UseUnary(desc, func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		mu1.Lock()
		callOrder1 = append(callOrder1, "u1_inner_before")
		if v, ok := ctx.Value(customCtxKey).(string); ok {
			customValue1 = v
		}
		mu1.Unlock()
		r, e := handler(ctx, req)
		mu1.Lock()
		callOrder1 = append(callOrder1, "u1_inner_after")
		mu1.Unlock()
		return r, e
	})

	var (
		mu2        sync.Mutex
		callOrder2 []string
	)
	reg2 := xGrpcMiddle.NewRegistry()
	reg2.UseUnary(desc, func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		mu2.Lock()
		callOrder2 = append(callOrder2, "u2_before")
		mu2.Unlock()
		r, e := handler(ctx, req)
		mu2.Lock()
		callOrder2 = append(callOrder2, "u2_after")
		mu2.Unlock()
		return r, e
	})

	var (
		muStream1  sync.Mutex
		sOrder1    []string
		sValueVal1 string
	)
	reg1.UseStream(desc, func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		muStream1.Lock()
		sOrder1 = append(sOrder1, "s1_outer_before")
		muStream1.Unlock()
		wrappedSS := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), customCtxKey, "stream_appA"),
		}
		err := handler(srv, wrappedSS)
		muStream1.Lock()
		sOrder1 = append(sOrder1, "s1_outer_after")
		muStream1.Unlock()
		return err
	})
	reg1.UseStream(desc, func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		muStream1.Lock()
		sOrder1 = append(sOrder1, "s1_inner_before")
		if v, ok := ss.Context().Value(customCtxKey).(string); ok {
			sValueVal1 = v
		}
		muStream1.Unlock()
		err := handler(srv, ss)
		muStream1.Lock()
		sOrder1 = append(sOrder1, "s1_inner_after")
		muStream1.Unlock()
		return err
	})

	app1, p1, fix1, _, _, _, runDone1 := createTestApp(t, "A", "AppA", reg1)
	defer func() {
		_ = app1.Shutdown()
		select {
		case <-runDone1:
		case <-time.After(2 * time.Second):
		}
	}()

	app2, p2, _, _, _, _, runDone2 := createTestApp(t, "B", "AppB", reg2)
	defer func() {
		_ = app2.Shutdown()
		select {
		case <-runDone2:
		case <-time.After(2 * time.Second):
		}
	}()

	addr1 := p1.Addr()
	addr2 := p2.Addr()
	if addr1 == addr2 {
		t.Fatalf("Both Apps got same addr: %s", addr1)
	}

	cc1, err := grpc.NewClient(addr1, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient A failed: %v", err)
	}
	defer func() { _ = cc1.Close() }()

	cc2, err := grpc.NewClient(addr2, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient B failed: %v", err)
	}
	defer func() { _ = cc2.Close() }()

	// 1. Unary 传入已存在 requestUUID，校验 Header 与 Trailer 均正确传递且等于该值
	var header1, trailer1 metadata.MD
	var out1 wrapperspb.StringValue
	callCtx1 := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(xGrpcConst.MetadataRequestUUID.String(), "custom-uuid-111"))
	err = cc1.Invoke(callCtx1, "/test.AppContextService/UnaryContextCheck", wrapperspb.String("hello"), &out1, grpc.Header(&header1), grpc.Trailer(&trailer1))
	if err != nil {
		t.Fatalf("Invoke AppA Unary failed: %v", err)
	}
	if out1.GetValue() != "AppA:custom-uuid-111" {
		t.Fatalf("Unexpected AppA response: %s", out1.GetValue())
	}
	hVals1 := header1.Get(xGrpcConst.TrailerRequestUUID.String())
	if len(hVals1) == 0 || hVals1[0] != "custom-uuid-111" {
		t.Fatalf("Header mismatch in AppA: %v", hVals1)
	}
	tVals1 := trailer1.Get(xGrpcConst.TrailerRequestUUID.String())
	if len(tVals1) == 0 || tVals1[0] != "custom-uuid-111" {
		t.Fatalf("Trailer mismatch in AppA: %v", tVals1)
	}

	// 2. Unary 未传入 requestUUID (自动生成，并保证 header、trailer 与 context 一致)
	var header2, trailer2 metadata.MD
	var out2 wrapperspb.StringValue
	err = cc2.Invoke(context.Background(), "/test.AppContextService/UnaryContextCheck", wrapperspb.String("hello"), &out2, grpc.Header(&header2), grpc.Trailer(&trailer2))
	if err != nil {
		t.Fatalf("Invoke AppB Unary failed: %v", err)
	}
	hVals2 := header2.Get(xGrpcConst.TrailerRequestUUID.String())
	if len(hVals2) == 0 || hVals2[0] == "" {
		t.Fatalf("Header empty in AppB: %v", hVals2)
	}
	tVals2 := trailer2.Get(xGrpcConst.TrailerRequestUUID.String())
	if len(tVals2) == 0 || tVals2[0] == "" {
		t.Fatalf("Trailer empty in AppB: %v", tVals2)
	}
	if hVals2[0] != tVals2[0] {
		t.Fatalf("Header and trailer UUID mismatch: %s != %s", hVals2[0], tVals2[0])
	}
	if out2.GetValue() != "AppB:"+tVals2[0] {
		t.Fatalf("AppB context requestKey and trailer mismatch: body=%s, trailer=%s", out2.GetValue(), tVals2[0])
	}

	// 3. 校验两套 App 中间件不串扰与洋葱圈顺序
	mu1.Lock()
	if len(callOrder1) != 4 ||
		callOrder1[0] != "u1_outer_before" ||
		callOrder1[1] != "u1_inner_before" ||
		callOrder1[2] != "u1_inner_after" ||
		callOrder1[3] != "u1_outer_after" {
		t.Fatalf("AppA middleware order mismatch: %v", callOrder1)
	}
	if customValue1 != "val_appA" {
		t.Fatalf("AppA typed request-level ctx value missing or incorrect: %s", customValue1)
	}
	mu1.Unlock()

	mu2.Lock()
	if len(callOrder2) != 2 || callOrder2[0] != "u2_before" || callOrder2[1] != "u2_after" {
		t.Fatalf("AppB middleware order mismatch: %v", callOrder2)
	}
	mu2.Unlock()

	// 4. 普通错误安全兜底、显式status保持语义
	var outErr wrapperspb.StringValue
	err = cc1.Invoke(context.Background(), "/test.AppContextService/UnaryErrorCheck", wrapperspb.String("sentinel"), &outErr)
	if err == nil {
		t.Fatalf("Expected sentinel error, got nil")
	}
	stErr, ok := status.FromError(err)
	if !ok || stErr.Code() != codes.Internal {
		t.Fatalf("Sentinel error should be normalized to gRPC status internal: %v", err)
	}
	if strings.Contains(stErr.Message(), errSentinelBiz.Error()) {
		t.Fatalf("Sentinel error message should not leak: %s", stErr.Message())
	}
	details := stErr.Details()
	if len(details) == 0 {
		t.Fatalf("Expected status details containing BaseResponse")
	}
	detail, ok := details[0].(*generate.BaseResponse)
	if !ok {
		t.Fatalf("Expected *generate.BaseResponse, got %T", details[0])
	}
	if detail.Code != 50000 {
		t.Fatalf("Expected BaseResponse code 50000, got %d", detail.Code)
	}
	if detail.Context == "" {
		t.Fatalf("Expected BaseResponse context tracking to be non-empty")
	}

	err = cc1.Invoke(context.Background(), "/test.AppContextService/UnaryErrorCheck", wrapperspb.String("status"), &outErr)
	if err == nil {
		t.Fatalf("Expected status error, got nil")
	}
	stErr, ok = status.FromError(err)
	if !ok || stErr.Code() != codes.FailedPrecondition {
		t.Fatalf("Expected FailedPrecondition, got: %v", err)
	}
	if stErr.Message() != "custom_failed_precondition" {
		t.Fatalf("Expected message custom_failed_precondition, got: %s", stErr.Message())
	}

	// 5. Stream 调用、两套 App 隔离与中间件顺序验证
	streamDesc := newTestAppServiceDesc(fix1).Streams[0]
	streamCallCtx, cancelStreamCtx := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelStreamCtx()
	streamCallCtx = metadata.NewOutgoingContext(streamCallCtx, metadata.Pairs(xGrpcConst.MetadataRequestUUID.String(), "stream-uuid-999"))
	streamCli, err := cc1.NewStream(streamCallCtx, &streamDesc, "/test.AppContextService/StreamContextCheck")
	if err != nil {
		t.Fatalf("NewStream failed: %v", err)
	}
	if err = streamCli.SendMsg(wrapperspb.String("stream-msg")); err != nil {
		t.Fatalf("SendMsg failed: %v", err)
	}
	var streamResp wrapperspb.StringValue
	if err = streamCli.RecvMsg(&streamResp); err != nil {
		t.Fatalf("RecvMsg failed: %v", err)
	}
	if streamResp.GetValue() != "AppA:stream-msg:stream-uuid-999" {
		t.Fatalf("Stream response mismatch: %s", streamResp.GetValue())
	}
	streamHeader, err := streamCli.Header()
	if err != nil {
		t.Fatalf("Stream Header() failed: %v", err)
	}
	sHeaders := streamHeader.Get(xGrpcConst.TrailerRequestUUID.String())
	if len(sHeaders) == 0 || sHeaders[0] != "stream-uuid-999" {
		t.Fatalf("Stream header mismatch: %v", sHeaders)
	}
	_ = streamCli.CloseSend()
	for {
		var dummy wrapperspb.StringValue
		recvErr := streamCli.RecvMsg(&dummy)
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			t.Fatalf("Drain stream failed: %v", recvErr)
		}
	}
	sTrailers := streamCli.Trailer().Get(xGrpcConst.TrailerRequestUUID.String())
	if len(sTrailers) == 0 || sTrailers[0] != "stream-uuid-999" {
		t.Fatalf("Stream trailer mismatch: %v", sTrailers)
	}

	muStream1.Lock()
	if len(sOrder1) != 4 ||
		sOrder1[0] != "s1_outer_before" ||
		sOrder1[1] != "s1_inner_before" ||
		sOrder1[2] != "s1_inner_after" ||
		sOrder1[3] != "s1_outer_after" {
		t.Fatalf("Stream middleware order mismatch: %v", sOrder1)
	}
	if sValueVal1 != "stream_appA" {
		t.Fatalf("Stream middleware ctx value mismatch: %s", sValueVal1)
	}
	muStream1.Unlock()

	// 6. 原请求取消可达 Handler，且不把 Start 短 ctx 继承为运行取消
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	var cancelOut wrapperspb.StringValue
	cancelDone := make(chan error, 1)
	go func() {
		cancelDone <- cc1.Invoke(cancelCtx, "/test.AppContextService/UnaryBlockUntilCancel", wrapperspb.String("wait"), &cancelOut)
	}()

	select {
	case <-fix1.cancelCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("Handler did not enter block until cancel")
	}

	cancelFunc()

	select {
	case invokeErr := <-cancelDone:
		if invokeErr == nil {
			t.Fatalf("Expected canceled invoke error, got nil")
		}
		cSt, ok := status.FromError(invokeErr)
		if !ok || cSt.Code() != codes.Canceled {
			t.Fatalf("Expected Canceled status code, got %v", invokeErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Invoke did not return upon cancel")
	}

	// 7. Unary handler err==nil && resp==nil 桥接开发者错误兜底
	noOutCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(xGrpcConst.TrailerRequestUUID.String(), "e23b44d1-b287-42e6-ab37-94e4f5287ef4"))
	err = cc1.Invoke(noOutCtx, "/test.AppContextService/UnaryNoOutput", wrapperspb.String("nil"), new(wrapperspb.StringValue))
	stNoOut, ok := status.FromError(err)
	if !ok || stNoOut.Code() != codes.InvalidArgument {
		t.Fatalf("Expected InvalidArgument for UnaryNoOutput, got err=%v", err)
	}
	detailsNoOut := stNoOut.Details()
	if len(detailsNoOut) == 0 {
		t.Fatalf("Expected status details containing BaseResponse for UnaryNoOutput")
	}
	detailNoOut, ok := detailsNoOut[0].(*generate.BaseResponse)
	if !ok || detailNoOut.Code != 40024 || detailNoOut.Context != "e23b44d1-b287-42e6-ab37-94e4f5287ef4" {
		t.Fatalf("Unexpected BaseResponse in UnaryNoOutput details: %+v", detailNoOut)
	}
}

func TestPlugin_AppContext_StartACLUndeclared(t *testing.T) {
	app := application.New(
		application.WithGraceTimeout(10*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	resKeyDeclared := ctxkey.Key("res_declared")
	resKeyUndeclared := ctxkey.Key("res_undeclared")

	_ = app.RegisterTool(application.PluginInfo{
		Key:      "tool_declared",
		Provides: []ctxkey.Key{resKeyDeclared},
	}, func() application.Plugin {
		return &toolResourcePlugin{resKey: resKeyDeclared, val: "declared_val"}
	})

	_ = app.RegisterTool(application.PluginInfo{
		Key:      "tool_undeclared",
		Provides: []ctxkey.Key{resKeyUndeclared},
	}, func() application.Plugin {
		return &toolResourcePlugin{resKey: resKeyUndeclared, val: "undeclared_val"}
	})

	aclResultCh := make(chan error, 1)
	grpcInfo, grpcFac := xGrpc.New(
		xGrpc.WithKey("grpc_worker_acl"),
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithRequires(resKeyDeclared),
		xGrpc.WithService(func(startCtx context.Context, server *grpc.Server) {
			_, err := application.Get[string](startCtx, resKeyDeclared)
			if err != nil {
				aclResultCh <- fmt.Errorf("declared get should succeed: %w", err)
				return
			}

			_, undeclaredErr := application.Get[string](startCtx, resKeyUndeclared)
			if undeclaredErr == nil {
				aclResultCh <- errors.New("undeclared get should fail with ACL check")
				return
			}
			if !errors.Is(undeclaredErr, application.ErrUndeclaredDependency) {
				aclResultCh <- fmt.Errorf("expected ErrUndeclaredDependency, got %w", undeclaredErr)
				return
			}
			aclResultCh <- nil
		}),
	)

	_ = app.RegisterWorker(grpcInfo, grpcFac)

	var runErr error
	runDone := make(chan struct{})

	t.Cleanup(func() {
		_ = app.Shutdown()
		select {
		case <-runDone:
			if runErr != nil && !errors.Is(runErr, application.ErrStartupInterrupted) {
				t.Errorf("unexpected run error during cleanup: %v", runErr)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("timed out waiting for app.Run to terminate during cleanup")
		}
	})

	go func() {
		runErr = app.Run()
		close(runDone)
	}()

	select {
	case aclErr := <-aclResultCh:
		if aclErr != nil {
			t.Fatalf("Start ACL check failed: %v", aclErr)
		}
	case <-runDone:
		t.Fatalf("App exited before ACL check returned: %v", runErr)
	case <-time.After(3 * time.Second):
		t.Fatalf("Timed out waiting for Start ACL callback execution")
	}

	_ = app.Shutdown()
	select {
	case <-runDone:
		if runErr != nil && !errors.Is(runErr, application.ErrStartupInterrupted) {
			t.Fatalf("App run returned unexpected error on shutdown: %v", runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Timed out waiting for App run to terminate after shutdown")
	}
}
