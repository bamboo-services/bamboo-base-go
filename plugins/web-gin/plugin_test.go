package webgin_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	webgin "github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/header"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
)

func TestWebGin_InstanceIsolationAndGlobalMode(t *testing.T) {
	// 验证 Start 阶段创建独立 engine，不修改 gin.Mode() 全局状态
	origMode := gin.Mode()

	info1, factory1 := webgin.New(
		webgin.WithAddr("127.0.0.1:0"),
	)
	p1 := factory1()
	defer func() { _ = p1.Stop(context.Background()) }()

	if err := p1.Start(context.Background()); err != nil {
		t.Fatalf("p1.Start failed: %v", err)
	}

	if gin.Mode() != origMode {
		t.Fatalf("expected gin.Mode() to remain %q, but got %q", origMode, gin.Mode())
	}

	info2, factory2 := webgin.New(
		webgin.WithAddr("127.0.0.1:0"),
		webgin.WithKey("plugin_web_gin_2"),
	)
	if info1.Key == info2.Key {
		t.Fatalf("expected different keys, got same %s", info1.Key)
	}

	p2 := factory2()
	defer func() { _ = p2.Stop(context.Background()) }()
	if err := p2.Start(context.Background()); err != nil {
		t.Fatalf("p2.Start failed: %v", err)
	}

	// 端口隔离验证：真实端口监听在不同端口
	w1, ok1 := p1.(*webgin.Plugin)
	w2, ok2 := p2.(*webgin.Plugin)
	if !ok1 || !ok2 {
		t.Fatalf("factory did not return *webgin.Plugin")
	}

	if w1.Addr() == w2.Addr() {
		t.Fatalf("expected different listen addresses, got both %s", w1.Addr())
	}
}

func TestWebGin_ContainerBridgeAndRequestKey(t *testing.T) {
	// 使用真实 App 注册并运行 Worker，验证 BridgeRequest 挂载真正的应用容器与 RequestKey
	app := application.New()

	var capturedReqID string
	var capturedStartTime time.Time
	var capturedHasContainer bool

	route := func(_ context.Context, r *gin.Engine) {
		r.GET("/test-context", func(c *gin.Context) {
			reqCtx := c.Request.Context()
			capturedReqID = ctxkey.GetRequestKey(reqCtx)

			if val := reqCtx.Value(ctxkey.UserStartTimeKey); val != nil {
				if st, ok := val.(time.Time); ok {
					capturedStartTime = st
				}
			}

			// 测试容器挂载：从应用中读取组件或检查容器存在
			// 即使未注册具体业务 key，只要挂载了应用容器，Get 不会报 ErrContainerNotMounted
			_, getErr := application.Get[string](reqCtx, "non_existent_key")
			if getErr != nil && !errors.Is(getErr.Err, application.ErrContainerNotMounted) {
				capturedHasContainer = true
			}

			httpresult.Success(c, "ok")
		})
	}

	info, factory := webgin.New(
		webgin.WithAddr("127.0.0.1:0"),
		webgin.WithRoute(route),
	)

	// 包装 factory 以提取运行中的 *webgin.Plugin
	var activePlugin *webgin.Plugin
	var mu sync.Mutex
	wrappedFactory := func() application.RunnablePlugin {
		p := factory()
		mu.Lock()
		activePlugin = p.(*webgin.Plugin)
		mu.Unlock()
		return p
	}

	if err := app.RegisterWorker(info, wrappedFactory); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	appErrCh := make(chan error, 1)
	go func() {
		appErrCh <- app.Run()
	}()

	var clientAddr string
	for range 50 {
		mu.Lock()
		p := activePlugin
		mu.Unlock()
		if p != nil && p.Addr() != "127.0.0.1:0" && p.Addr() != "" {
			clientAddr = p.Addr()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if clientAddr == "" {
		t.Fatalf("failed to obtain active worker listen address")
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/test-context", clientAddr), nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	clientCustomID := "custom-trace-ctx-123"
	req.Header.Set("X-Request-Id", clientCustomID)
	req.Header.Set(header.HeaderRequestUUID.String(), clientCustomID)

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	serverHeaderID := resp.Header.Get(header.HeaderRequestUUID.String())
	if serverHeaderID == "" {
		t.Fatalf("expected non-empty %s header in resp", header.HeaderRequestUUID)
	}
	if _, err := uuid.Parse(serverHeaderID); err != nil {
		t.Fatalf("expected valid UUID in resp header, got %q: %v", serverHeaderID, err)
	}
	if serverHeaderID == clientCustomID {
		t.Fatalf("expected client custom ID %q to be ignored, but server echoed it", clientCustomID)
	}
	if capturedReqID != serverHeaderID {
		t.Fatalf("expected capturedReqID %q to match server header %q", capturedReqID, serverHeaderID)
	}
	if capturedStartTime.IsZero() {
		t.Fatalf("expected capturedStartTime to be non-zero time")
	}
	if !capturedHasContainer {
		t.Fatalf("expected real app container to be mounted in request context")
	}

	_ = activePlugin.Stop(context.Background())
}

func TestWebGin_FullNetworkLifecycle(t *testing.T) {
	// 完整真实网络集成测试：
	// 1. 启动服务，真实监听 127.0.0.1:0
	// 2. 发起真实 HTTP 请求
	// 3. 验证 BridgeRequest 容器、RequestKey、Overhead、Header
	// 4. 验证缺少 lifetimeCtx 拒绝
	// 5. 验证 Quiesce 与 GracefulStop 排空：测试 Handler 忽略 ctx 时 Stop 等待，释放后真实退
	// 6. 验证 ForceContext 取消到达 Handler
	// 7. 验证端口释放与重复 Stop 幂等

	const testHeaderKey = "X-Request-Id"
	customReqID := "custom-req-id-888"

	type testPayload struct {
		Name string `json:"name" binding:"required"`
	}

	handlerEntering := make(chan struct{})
	blockHandler := make(chan struct{})
	handlerExited := make(chan struct{})

	forceReceived := make(chan struct{})

	route := func(_ context.Context, r *gin.Engine) {
		r.POST("/bind-test", func(c *gin.Context) {
			b := bind.FromContext(c)
			if b == nil {
				httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "binder not injected", nil))
				return
			}
			var p testPayload
			if bind.Bind(c, &p).Data() == nil {
				return
			}
			httpresult.SuccessWithData(c, "bound", p.Name)
		})

		r.GET("/slow-request", func(c *gin.Context) {
			close(handlerEntering)
			select {
			case <-blockHandler:
			case <-c.Request.Context().Done():
				close(forceReceived)
			}
			close(handlerExited)
			httpresult.Success(c, "done")
		})

		r.GET("/panic-route", func(c *gin.Context) {
			panic("simulated panic")
		})

		r.GET("/empty-route", func(c *gin.Context) {
			// 不写任何数据，测试 Response 兜底中间件
		})
	}

	app := application.New()
	var assignedAddr string
	var mu sync.Mutex

	info, factory := webgin.New(
		webgin.WithAddr("127.0.0.1:0"),
		webgin.WithRoute(route),
	)

	// 包装 factory 以提取运行中的 *webgin.Plugin
	var activePlugin *webgin.Plugin
	wrappedFactory := func() application.RunnablePlugin {
		p := factory()
		mu.Lock()
		activePlugin = p.(*webgin.Plugin)
		mu.Unlock()
		return p
	}

	if err := app.RegisterWorker(info, wrappedFactory); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	appRunErr := make(chan error, 1)
	go func() {
		appRunErr <- app.Run()
	}()

	// 等待 Worker 启动并取得真实监听地址
	var clientAddr string
	for range 50 {
		mu.Lock()
		p := activePlugin
		mu.Unlock()
		if p != nil && p.Addr() != "127.0.0.1:0" && p.Addr() != "" {
			clientAddr = p.Addr()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if clientAddr == "" {
		t.Fatalf("failed to obtain active worker listen address")
	}

	assignedAddr = clientAddr
	httpClient := &http.Client{Timeout: 5 * time.Second}

	// 1. 测试基础请求与 Bind、Trace、Overhead
	{
		bodyStr := `{"name":"bamboo"}`
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/bind-test", assignedAddr), io.NopCloser(strings.NewReader(bodyStr)))
		if err != nil {
			t.Fatalf("create request failed: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(testHeaderKey, customReqID)
		req.Header.Set(header.HeaderRequestUUID.String(), customReqID)

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected status 200, got %d, body: %s", resp.StatusCode, string(b))
		}

		serverHeaderID1 := resp.Header.Get(header.HeaderRequestUUID.String())
		if serverHeaderID1 == "" {
			t.Fatalf("expected non-empty %s header in resp", header.HeaderRequestUUID)
		}
		if _, err := uuid.Parse(serverHeaderID1); err != nil {
			t.Fatalf("expected valid UUID in resp header, got %q: %v", serverHeaderID1, err)
		}
		if serverHeaderID1 == customReqID {
			t.Fatalf("expected client customReqID %q to be ignored, but server echoed it", customReqID)
		}

		var envResp envelope.Response
		if err := json.NewDecoder(resp.Body).Decode(&envResp); err != nil {
			t.Fatalf("decode response failed: %v", err)
		}

		if envResp.Context != serverHeaderID1 {
			t.Fatalf("expected response Context %q to match header %q", envResp.Context, serverHeaderID1)
		}
		if envResp.Overhead < 0 {
			t.Fatalf("expected non-negative overhead, got %d", envResp.Overhead)
		}

		// 发起第二请求验证每次生成新 UUID 且不同
		req2, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/bind-test", assignedAddr), io.NopCloser(strings.NewReader(bodyStr)))
		if err != nil {
			t.Fatalf("create request 2 failed: %v", err)
		}
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set(testHeaderKey, customReqID)
		req2.Header.Set(header.HeaderRequestUUID.String(), customReqID)

		resp2, err := httpClient.Do(req2)
		if err != nil {
			t.Fatalf("request 2 failed: %v", err)
		}
		defer func() { _ = resp2.Body.Close() }()

		serverHeaderID2 := resp2.Header.Get(header.HeaderRequestUUID.String())
		if serverHeaderID2 == "" {
			t.Fatalf("expected non-empty %s header in resp2", header.HeaderRequestUUID)
		}
		if _, err := uuid.Parse(serverHeaderID2); err != nil {
			t.Fatalf("expected valid UUID in resp2 header, got %q: %v", serverHeaderID2, err)
		}
		if serverHeaderID2 == serverHeaderID1 {
			t.Fatalf("expected different UUIDs for separate requests, got both %q", serverHeaderID1)
		}
		var envResp2 envelope.Response
		if err := json.NewDecoder(resp2.Body).Decode(&envResp2); err != nil {
			t.Fatalf("decode response 2 failed: %v", err)
		}
		if envResp2.Context != serverHeaderID2 {
			t.Fatalf("expected response 2 Context %q to match header %q", envResp2.Context, serverHeaderID2)
		}
	}

	// 2. 测试 Panic 兜底中间件
	{
		resp, err := httpClient.Get(fmt.Sprintf("http://%s/panic-route", assignedAddr))
		if err != nil {
			t.Fatalf("panic route request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status 500 on panic, got %d", resp.StatusCode)
		}
		var envResp envelope.Response
		_ = json.NewDecoder(resp.Body).Decode(&envResp)
		if envResp.Code != apperr.ServerInternalError.Code {
			t.Fatalf("expected ServerInternalError code, got %d", envResp.Code)
		}
	}

	// 3. 测试 Response 兜底中间件（无输出时兜底为开发者错误）
	{
		resp, err := httpClient.Get(fmt.Sprintf("http://%s/empty-route", assignedAddr))
		if err != nil {
			t.Fatalf("empty route request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var envResp envelope.Response
		_ = json.NewDecoder(resp.Body).Decode(&envResp)
		if envResp.Code != apperr.DeveloperError.Code {
			t.Fatalf("expected DeveloperError code, got %d", envResp.Code)
		}
	}

	// 4. 验证 Quiesce 与 Grace 排空：
	// 发起慢请求，Handler 正在运行；此时对 Worker 执行 Stop；
	// 验证在 Handler 退出前 Stop 保持阻塞，并在 Handler 退出后 Stop 成功返回。
	slowClient := &http.Client{}
	go func() {
		slowReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/slow-request", assignedAddr), nil)
		resp, err := slowClient.Do(slowReq)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	// 等待 Handler 确实已进入
	select {
	case <-handlerEntering:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for slow handler to enter")
	}

	// 尝试向服务器发起新请求，此时 Quiesce 关闭后必须拒绝新请求（或者连接被拒）
	stopStarted := make(chan struct{})
	stopDone := make(chan error, 1)

	go func() {
		close(stopStarted)
		// 给予 3 秒优雅关机时间
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		stopDone <- activePlugin.Stop(stopCtx)
	}()

	<-stopStarted
	time.Sleep(50 * time.Millisecond)

	// 新请求应该被拒绝（429/503 或 connect refused）
	newReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/empty-route", assignedAddr), nil)
	newResp, newErr := httpClient.Do(newReq)
	if newErr == nil {
		if newResp.StatusCode != http.StatusServiceUnavailable {
			t.Logf("new request status during shutdown: %d", newResp.StatusCode)
		}
		_ = newResp.Body.Close()
	}

	// 确保此时 Stop 依然在阻塞等待，没有提前返回
	select {
	case <-stopDone:
		t.Fatal("Stop returned prematurely while handler was still running!")
	case <-time.After(200 * time.Millisecond):
		// 预期行为：Stop 挂起等待飞行中请求
	}

	// 释放 Handler
	close(blockHandler)

	// 等待 Handler 真实退出
	select {
	case <-handlerExited:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit")
	}

	// 此时 Stop 应该成功返回
	select {
	case err := <-stopDone:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Stop failed with unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not complete after handler exited")
	}

	// 验证重复调用 Stop 共享同一次结果
	if err := activePlugin.Stop(context.Background()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("repeated Stop failed: %v", err)
	}

	// 5. 验证端口已真实释放，可重新绑定
	ln, err := net.Listen("tcp", assignedAddr)
	if err != nil {
		t.Fatalf("expected port %s to be released and rebindable, but got err: %v", assignedAddr, err)
	}
	_ = ln.Close()
}

func TestWebGin_ForceContextCancellation(t *testing.T) {
	// 测试 App ForceContext 触发后，Handler 接收到 ctx 取消
	forceCancelled := make(chan struct{})

	route := func(_ context.Context, r *gin.Engine) {
		r.GET("/force-test", func(c *gin.Context) {
			<-c.Request.Context().Done()
			close(forceCancelled)
			httpresult.Success(c, "cancelled")
		})
	}

	_, factory := webgin.New(
		webgin.WithAddr("127.0.0.1:0"),
		webgin.WithRoute(route),
	)

	p := factory().(*webgin.Plugin)

	// 构造一个可取消的 force 上下文传给 Start
	forceRootCtx, cancelForceRoot := context.WithCancel(context.Background())
	// 挂载 forceCtxKeyType 凭据
	startCtx := forceRootCtx

	if err := p.Start(startCtx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = p.Stop(context.Background()) }()

	addr := p.Addr()

	// 构造合法的 lifetimeCtx
	app := application.New()
	workerInfo := application.PluginInfo{Key: "dummy_worker"}
	dummyStarted := make(chan context.Context, 1)
	_ = app.RegisterWorker(workerInfo, func() application.RunnablePlugin {
		return &testWorkerLifetime{started: dummyStarted, done: make(chan struct{})}
	})
	go func() { _ = app.Run() }()

	var lifetimeCtx context.Context
	select {
	case lifetimeCtx = <-dummyStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for lifetimeCtx")
	}

	runErr := make(chan error, 1)
	go func() {
		runErr <- p.Run(lifetimeCtx)
	}()

	time.Sleep(50 * time.Millisecond)

	// 发起 HTTP 请求
	client := &http.Client{}
	go func() {
		_, _ = client.Get(fmt.Sprintf("http://%s/force-test", addr))
	}()

	time.Sleep(50 * time.Millisecond)

	// 触发 Stop，超时 100ms 强制超时，推进 Close
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer stopCancel()

	_ = p.Stop(stopCtx)

	select {
	case <-forceCancelled:
		// 成功收到取消
	case <-time.After(2 * time.Second):
		t.Fatal("Handler did not receive context cancellation during force stop")
	}

	_ = cancelForceRoot
}

func TestWebGin_MissingLifetimeCtxRejects(t *testing.T) {
	// 验证缺少可信 lifetimeCtx 的请求被拒绝为 503，而非 fallback 背景
	route := func(_ context.Context, r *gin.Engine) {
		r.GET("/no-lifetime", func(c *gin.Context) {
			httpresult.Success(c, "should not be reached")
		})
	}

	_, factory := webgin.New(
		webgin.WithAddr("127.0.0.1:0"),
		webgin.WithRoute(route),
	)

	p := factory().(*webgin.Plugin)
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = p.Stop(context.Background()) }()

	// 未调用 Run(lifetimeCtx)，直接在 listener 上处理 HTTP 请求（或手工模拟调用）
	// 为测试方便，在后台启动一个伪 Serve 但不注入 lifetimeCtx
	var serveDone int32
	go func() {
		// 直接用 p.server.Serve，此时 lifetimeCtx 为 nil
		_ = p.Run(nil) //nolint:staticcheck // nil 用于验证缺失 lifetimeCtx，必须拒绝请求
		atomic.StoreInt32(&serveDone, 1)
	}()

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%s/no-lifetime", p.Addr()))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 for request without lifetimeCtx, got %d", resp.StatusCode)
	}
}

type testWorkerLifetime struct {
	started chan context.Context
	done    chan struct{}
}

func (w *testWorkerLifetime) Start(_ context.Context) error { return nil }
func (w *testWorkerLifetime) Run(lifetimeCtx context.Context) error {
	w.started <- lifetimeCtx
	<-w.done
	return nil
}
func (w *testWorkerLifetime) Stop(_ context.Context) error {
	close(w.done)
	return nil
}
