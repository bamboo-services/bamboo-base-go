package webgin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
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
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	v10 "github.com/go-playground/validator/v10"
)

type isolationPayload struct {
	TagVal string `json:"tag_val" binding:"same_rule"`
	Secret string `json:"secret" binding:"required"`
}

type toolPlugin struct {
	resKey ctxkey.Key
	value  string
}

func (p *toolPlugin) Start(_ context.Context) error {
	return nil
}

func (p *toolPlugin) Stop(_ context.Context) error {
	return nil
}

func (p *toolPlugin) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		p.resKey: p.value,
	}
}

type witnessWorker struct {
	ready  chan struct{}
	runCtx context.Context
	once   sync.Once
}

func newWitnessWorker() *witnessWorker {
	return &witnessWorker{
		ready: make(chan struct{}),
	}
}

func (w *witnessWorker) Start(_ context.Context) error {
	return nil
}

func (w *witnessWorker) Run(ctx context.Context) error {
	w.runCtx = ctx
	w.once.Do(func() {
		close(w.ready)
	})
	<-ctx.Done()
	return nil
}

func (w *witnessWorker) RunContext() context.Context {
	return w.runCtx
}

func (w *witnessWorker) Stop(_ context.Context) error {
	return nil
}

func buildApp(
	t *testing.T,
	resourceKey ctxkey.Key,
	toolVal string,
	expectedRuleVal string,
	declaredRequires bool,
) (*application.App, *webgin.Plugin, *witnessWorker, *string) {
	t.Helper()

	app := application.New(
		application.WithGraceTimeout(10*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	toolInfo := application.PluginInfo{
		Key:      ctxkey.Key("tool_plugin_" + toolVal),
		Name:     "tool_plugin_" + toolVal,
		Mode:     application.ModeTool,
		Provides: []ctxkey.Key{resourceKey},
	}
	err := app.RegisterTool(toolInfo, func() application.Plugin {
		return &toolPlugin{resKey: resourceKey, value: toolVal}
	})
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	var capturedStartResource string
	routeRegistrar := func(startCtx context.Context, engine *gin.Engine) {
		res, getErr := application.Get[string](startCtx, resourceKey)
		if getErr != nil {
			if declaredRequires {
				t.Errorf("expected Get in Start to succeed, got: %v", getErr)
			}
		} else {
			capturedStartResource = res
		}

		engine.POST("/test", func(c *gin.Context) {
			b := bind.FromContext(c)
			if b == nil {
				httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "binder not injected", nil))
				return
			}

			var req isolationPayload
			resErr := bind.Bind(c, &req).Data()
			if resErr == nil {
				return
			}

			reqCtx := c.Request.Context()
			reqKey := ctxkey.GetRequestKey(reqCtx)
			if reqKey == "" {
				httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "missing RequestKey", nil))
				return
			}

			svc, hGetErr := application.Get[string](reqCtx, resourceKey)
			if hGetErr != nil || svc == "" {
				httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to get resource in handler", nil))
				return
			}

			cVal := c.Value(resourceKey)
			if cVal == nil {
				httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to get resource via c.Value", nil))
				return
			}
			if strVal, ok := cVal.(string); !ok || strVal != toolVal {
				httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "resource mismatch via c.Value", nil))
				return
			}

			httpresult.SuccessWithData(c, "success", fmt.Sprintf("%s:%s:%s", svc, req.TagVal, reqKey))
		})
	}

	validatorRegistrar := func(vEng *validator.Engine) error {
		return vEng.RegisterValidation("same_rule", func(fl v10.FieldLevel) bool {
			return fl.Field().String() == expectedRuleVal
		})
	}

	opts := []webgin.Option{
		webgin.WithAddr("127.0.0.1:0"),
		webgin.WithRoute(routeRegistrar),
		webgin.WithValidatorRegistrar(validatorRegistrar),
	}
	if declaredRequires {
		opts = append(opts, webgin.WithRequires(resourceKey))
	}

	ginInfo, ginFactory := webgin.New(opts...)
	var pluginInstance *webgin.Plugin
	var pMu sync.Mutex
	wrappedGinFactory := func() application.RunnablePlugin {
		p := ginFactory()
		pMu.Lock()
		pluginInstance = p.(*webgin.Plugin)
		pMu.Unlock()
		return p
	}

	if err := app.RegisterWorker(ginInfo, wrappedGinFactory); err != nil {
		t.Fatalf("RegisterWorker gin failed: %v", err)
	}

	witness := newWitnessWorker()
	witnessInfo := application.PluginInfo{
		Key:      ctxkey.Key("witness_" + toolVal),
		Name:     "witness_" + toolVal,
		Mode:     application.ModeWorker,
		Requires: []ctxkey.Key{ginInfo.Key},
	}
	if err := app.RegisterWorker(witnessInfo, func() application.RunnablePlugin {
		return witness
	}); err != nil {
		t.Fatalf("RegisterWorker witness failed: %v", err)
	}

	return app, pluginInstance, witness, &capturedStartResource
}

func executeRequest(t *testing.T, addr string, body string) (*http.Response, envelope.Response, string) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/test", addr), bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	clientTraceID := "trace-" + time.Now().Format("150405.000000")
	req.Header.Set("X-Request-Id", clientTraceID)
	req.Header.Set(header.HeaderRequestUUID.String(), clientTraceID)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request to %s failed: %v", addr, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	var env envelope.Response
	if err := json.Unmarshal(respBytes, &env); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	serverReqUUID := resp.Header.Get(header.HeaderRequestUUID.String())
	if serverReqUUID == "" {
		t.Fatalf("expected non-empty %s header in resp", header.HeaderRequestUUID)
	}
	if _, err := uuid.Parse(serverReqUUID); err != nil {
		t.Fatalf("expected valid UUID in response header %q: %v", serverReqUUID, err)
	}
	if serverReqUUID == clientTraceID {
		t.Fatalf("expected client trace ID %q to be ignored, but server echoed it", clientTraceID)
	}
	if env.Context != serverReqUUID {
		t.Fatalf("expected env.Context %q == header %q", env.Context, serverReqUUID)
	}

	return resp, env, string(respBytes)
}

func TestWebGin_TwoApps_FullIsolation(t *testing.T) {
	origGinMode := gin.Mode()
	origValidator := binding.Validator

	const resourceKey ctxkey.Key = "shared_tool_key"

	appA, _, witnessA, capturedA := buildApp(t, resourceKey, "VALUE_A", "ONLY_A", true)
	appB, _, witnessB, capturedB := buildApp(t, resourceKey, "VALUE_B", "ONLY_B", true)

	runErrChA := make(chan error, 1)
	go func() {
		runErrChA <- appA.Run()
	}()
	t.Cleanup(func() {
		_ = appA.Shutdown()
		select {
		case <-runErrChA:
		case <-time.After(2 * time.Second):
		}
	})

	runErrChB := make(chan error, 1)
	go func() {
		runErrChB <- appB.Run()
	}()
	t.Cleanup(func() {
		_ = appB.Shutdown()
		select {
		case <-runErrChB:
		case <-time.After(2 * time.Second):
		}
	})

	select {
	case <-witnessA.ready:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for appA witness ready")
	}

	select {
	case <-witnessB.ready:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for appB witness ready")
	}

	pAVal, errA := application.Get[application.RunnablePlugin](witnessA.RunContext(), webgin.DefaultWebGinPluginKey)
	var pluginA *webgin.Plugin
	if errA == nil {
		pluginA, _ = pAVal.(*webgin.Plugin)
	}
	_ = errA

	pBVal, errB := application.Get[application.RunnablePlugin](witnessB.RunContext(), webgin.DefaultWebGinPluginKey)
	var pluginB *webgin.Plugin
	if errB == nil {
		pluginB, _ = pBVal.(*webgin.Plugin)
	}
	_ = errB

	addrA := pluginA.Addr()
	addrB := pluginB.Addr()

	if addrA == "" || addrA == "127.0.0.1:0" {
		t.Fatalf("invalid addrA: %s", addrA)
	}
	if addrB == "" || addrB == "127.0.0.1:0" {
		t.Fatalf("invalid addrB: %s", addrB)
	}
	if addrA == addrB {
		t.Fatalf("addrA and addrB must be distinct ports: got both %s", addrA)
	}

	if *capturedA != "VALUE_A" {
		t.Fatalf("expected capturedA to be 'VALUE_A', got %q", *capturedA)
	}
	if *capturedB != "VALUE_B" {
		t.Fatalf("expected capturedB to be 'VALUE_B', got %q", *capturedB)
	}

	const sensitiveValue = "SECRET_CREDENTIAL_XYZ_999"

	// 1. AppA 验证：发送符合 A 规则的 payload -> 200
	var serverUUIDA1 string
	{
		payloadA := fmt.Sprintf(`{"tag_val":"ONLY_A","secret":"%s"}`, sensitiveValue)
		respA1, envA1, _ := executeRequest(t, addrA, payloadA)
		if respA1.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for appA with payloadA, got %d", respA1.StatusCode)
		}
		serverUUIDA1 = respA1.Header.Get(header.HeaderRequestUUID.String())
		expectedDataA := fmt.Sprintf("VALUE_A:ONLY_A:%s", serverUUIDA1)
		dataStr, _ := envA1.Data.(string)
		if dataStr != expectedDataA {
			t.Fatalf("expected appA success data %q, got %q", expectedDataA, dataStr)
		}
	}

	// 2. AppA 验证：发送符合 B 规则的 payload -> 400
	{
		payloadB := fmt.Sprintf(`{"tag_val":"ONLY_B","secret":"%s"}`, sensitiveValue)
		respA2, _, rawA2 := executeRequest(t, addrA, payloadB)
		if respA2.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for appA with payloadB, got %d", respA2.StatusCode)
		}
		if strings.Contains(rawA2, sensitiveValue) {
			t.Fatalf("security violation: raw error response leaked sensitive value %q", sensitiveValue)
		}
	}

	// 3. AppB 验证：发送符合 B 规则的 payload -> 200
	{
		payloadB := fmt.Sprintf(`{"tag_val":"ONLY_B","secret":"%s"}`, sensitiveValue)
		respB1, envB1, _ := executeRequest(t, addrB, payloadB)
		if respB1.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for appB with payloadB, got %d", respB1.StatusCode)
		}
		serverUUIDB1 := respB1.Header.Get(header.HeaderRequestUUID.String())
		if serverUUIDB1 == serverUUIDA1 {
			t.Fatalf("expected appA and appB to have distinct UUIDs, both got %q", serverUUIDA1)
		}
		expectedDataB := fmt.Sprintf("VALUE_B:ONLY_B:%s", serverUUIDB1)
		dataStr, _ := envB1.Data.(string)
		if dataStr != expectedDataB {
			t.Fatalf("expected appB success data %q, got %q", expectedDataB, dataStr)
		}
	}

	// 4. AppB 验证：发送符合 A 规则的 payload -> 400
	{
		payloadA := fmt.Sprintf(`{"tag_val":"ONLY_A","secret":"%s"}`, sensitiveValue)
		respB2, _, rawB2 := executeRequest(t, addrB, payloadA)
		if respB2.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for appB with payloadA, got %d", respB2.StatusCode)
		}
		if strings.Contains(rawB2, sensitiveValue) {
			t.Fatalf("security violation: raw error response leaked sensitive value %q", sensitiveValue)
		}
	}

	// 验证全局状态未被篡改
	if gin.Mode() != origGinMode {
		t.Fatalf("expected gin.Mode() to remain unchanged (%q), got %q", origGinMode, gin.Mode())
	}
	if binding.Validator != origValidator {
		t.Fatalf("expected binding.Validator to remain unchanged, but was modified")
	}

	// 停机与等待真退
	if err := appA.Shutdown(); err != nil {
		t.Fatalf("appA Shutdown failed: %v", err)
	}
	if err := appB.Shutdown(); err != nil {
		t.Fatalf("appB Shutdown failed: %v", err)
	}

	select {
	case err := <-runErrChA:
		if err != nil {
			t.Fatalf("appA Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for appA Run to exit")
	}

	select {
	case err := <-runErrChB:
		if err != nil {
			t.Fatalf("appB Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for appB Run to exit")
	}
}

func TestWebGin_UndeclaredDependency_RejectedInStart(t *testing.T) {
	const resourceKey ctxkey.Key = "undeclared_tool_key"

	app := application.New(
		application.WithGraceTimeout(10*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	toolInfo := application.PluginInfo{
		Key:      ctxkey.Key("tool_plugin_undeclared"),
		Name:     "tool_undeclared",
		Mode:     application.ModeTool,
		Provides: []ctxkey.Key{resourceKey},
	}
	err := app.RegisterTool(toolInfo, func() application.Plugin {
		return &toolPlugin{resKey: resourceKey, value: "TOOL_VAL"}
	})
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	startAccessErrCh := make(chan error, 1)
	routeRegistrar := func(startCtx context.Context, engine *gin.Engine) {
		_, getErr := application.Get[string](startCtx, resourceKey)
		startAccessErrCh <- getErr
	}

	ginInfo, ginFactory := webgin.New(
		webgin.WithAddr("127.0.0.1:0"),
		webgin.WithRoute(routeRegistrar),
		// 故意不写 webgin.WithRequires(resourceKey)
	)

	if err := app.RegisterWorker(ginInfo, ginFactory); err != nil {
		t.Fatalf("RegisterWorker gin failed: %v", err)
	}

	runDoneCh := make(chan struct{})
	var runErr error
	go func() {
		runErr = app.Run()
		close(runDoneCh)
	}()
	t.Cleanup(func() {
		_ = app.Shutdown()
		select {
		case <-runDoneCh:
		case <-time.After(2 * time.Second):
		}
	})

	var startAccessErr error
	select {
	case startAccessErr = <-startAccessErrCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for routeRegistrar startAccessErr")
	}

	if startAccessErr == nil {
		t.Fatalf("expected undeclared dependency access error during Start, but got nil")
	}
	if !errors.Is(startAccessErr, application.ErrUndeclaredDependency) {
		t.Fatalf("expected ErrUndeclaredDependency, got: %v", startAccessErr)
	}

	select {
	case <-runDoneCh:
		_ = runErr
	case <-time.After(2 * time.Second):
		_ = app.Shutdown()
		<-runDoneCh
	}
}
