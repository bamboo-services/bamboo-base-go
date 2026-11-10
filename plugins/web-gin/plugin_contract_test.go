package webgin_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"syscall"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	webgin "github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
)

// 生命周期契约断言：四个生命周期方法必须返回统一的 [*apperr.Error]，
// 且插件必须继续满足内核冻结后的 [application.Plugin]、[application.RunnablePlugin]
// 与 [application.Quiescer] 契约。
var (
	_ func(*webgin.Plugin, context.Context) *apperr.Error = (*webgin.Plugin).Start
	_ func(*webgin.Plugin, context.Context) *apperr.Error = (*webgin.Plugin).Stop
	_ func(*webgin.Plugin, context.Context) *apperr.Error = (*webgin.Plugin).Run
	_ func(*webgin.Plugin, context.Context) *apperr.Error = (*webgin.Plugin).Quiesce
	_ application.Plugin                                  = (*webgin.Plugin)(nil)
	_ application.RunnablePlugin                          = (*webgin.Plugin)(nil)
	_ application.Quiescer                                = (*webgin.Plugin)(nil)
)

// TestWebGin_LifecycleSuccessReturnsTrueNil 验证成功路径返回真正的 nil 指针，
// 不得返回装入 error 接口的 nil 载体，也不得返回带空码的假失败。
func TestWebGin_LifecycleSuccessReturnsTrueNil(t *testing.T) {
	_, factory := webgin.New(webgin.WithAddr("127.0.0.1:0"))
	p := factory().(*webgin.Plugin)

	startErr := p.Start(context.Background())
	if startErr != nil {
		t.Fatalf("Start 失败: %v", startErr)
	}
	if !apperr.IsNil(startErr) {
		t.Fatalf("Start 成功时必须返回真正的 nil 指针，实际得到: %+v", startErr)
	}

	quiesceErr := p.Quiesce(context.Background())
	if quiesceErr != nil {
		t.Fatalf("Quiesce 失败: %v", quiesceErr)
	}
	if !apperr.IsNil(quiesceErr) {
		t.Fatalf("Quiesce 成功时必须返回真正的 nil 指针，实际得到: %+v", quiesceErr)
	}

	stopErr := p.Stop(context.Background())
	if stopErr != nil {
		t.Fatalf("Stop 失败: %v", stopErr)
	}
	if !apperr.IsNil(stopErr) {
		t.Fatalf("Stop 成功时必须返回真正的 nil 指针，实际得到: %+v", stopErr)
	}
}

// TestWebGin_DuplicateStartReturnsInvalidState 验证重复启动返回既有语义码
// [httpresult.InvalidState]，并保留 [webgin.ErrServerAlreadyStarted] 作为根因。
func TestWebGin_DuplicateStartReturnsInvalidState(t *testing.T) {
	_, factory := webgin.New(webgin.WithAddr("127.0.0.1:0"))
	p := factory().(*webgin.Plugin)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	defer func() { _ = p.Stop(context.Background()) }()

	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("重复启动必须失败")
	}
	if err.Code.Code != httpresult.InvalidState.Code {
		t.Fatalf("重复启动错误码错误: 期望 %d, 得到 %d", httpresult.InvalidState.Code, err.Code.Code)
	}
	if !errors.Is(err, webgin.ErrServerAlreadyStarted) {
		t.Fatalf("重复启动必须保留 ErrServerAlreadyStarted 根因, 得到: %v", err)
	}
}

// TestWebGin_RunBeforeStartReturnsServiceUnavailable 验证未启动即调用 Run
// 返回既有语义码 [httpresult.ServiceUnavailable]，并保留 [webgin.ErrNotStarted] 根因。
func TestWebGin_RunBeforeStartReturnsServiceUnavailable(t *testing.T) {
	_, factory := webgin.New(webgin.WithAddr("127.0.0.1:0"))
	p := factory().(*webgin.Plugin)

	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("未启动即调用 Run 必须失败")
	}
	if err.Code.Code != httpresult.ServiceUnavailable.Code {
		t.Fatalf("未启动即 Run 的错误码错误: 期望 %d, 得到 %d", httpresult.ServiceUnavailable.Code, err.Code.Code)
	}
	if !errors.Is(err, webgin.ErrNotStarted) {
		t.Fatalf("未启动即 Run 必须保留 ErrNotStarted 根因, 得到: %v", err)
	}
}

// TestWebGin_StartListenFailureKeepsRootCause 验证监听失败归一为模块内部错误码
// [httpresult.InternalError]，且底层地址占用根因不丢失。
func TestWebGin_StartListenFailureKeepsRootCause(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("准备占用端口失败: %v", err)
	}
	defer func() { _ = occupied.Close() }()

	_, factory := webgin.New(webgin.WithAddr(occupied.Addr().String()))
	p := factory().(*webgin.Plugin)

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("监听失败必须返回错误")
	}
	if startErr.Code.Code != httpresult.InternalError.Code {
		t.Fatalf("监听失败错误码错误: 期望 %d, 得到 %d", httpresult.InternalError.Code, startErr.Code.Code)
	}
	if startErr.Err == nil {
		t.Fatal("监听失败必须保留底层根因")
	}
	if !errors.Is(startErr, syscall.EADDRINUSE) {
		t.Fatalf("监听失败必须保留地址占用根因, 得到: %v", startErr)
	}
}

// TestWebGin_StopWithCanceledContextClassifiedAsControlError 验证停机预算上下文取消
// 归一为控制类错误码，而不是被吞掉或误判为内部错误。
func TestWebGin_StopWithCanceledContextClassifiedAsControlError(t *testing.T) {
	handlerEntered := make(chan struct{})
	route := func(_ context.Context, r *gin.Engine) {
		r.GET("/block", func(c *gin.Context) {
			close(handlerEntered)
			<-c.Request.Context().Done()
			httpresult.Success(c, "canceled")
		})
	}

	_, factory := webgin.New(webgin.WithAddr("127.0.0.1:0"), webgin.WithRoute(route))
	p := factory().(*webgin.Plugin)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	runDone := make(chan *apperr.Error, 1)
	go func() {
		runDone <- p.Run(context.Background())
	}()

	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		resp, err := http.Get("http://" + p.Addr() + "/block")
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	<-handlerEntered

	stopCtx, cancelStop := context.WithCancel(context.Background())
	cancelStop()

	stopErrCh := make(chan *apperr.Error, 1)
	go func() {
		stopErrCh <- p.Stop(stopCtx)
	}()

	stopErr := <-stopErrCh
	if stopErr == nil {
		t.Fatal("停机上下文取消时 Stop 必须返回失败")
	}
	if stopErr.Code.Code != control.Canceled.Code {
		t.Fatalf("停机上下文取消的错误码错误: 期望 %d, 得到 %d", control.Canceled.Code, stopErr.Code.Code)
	}
	if !errors.Is(stopErr, context.Canceled) {
		t.Fatalf("停机上下文取消必须保留原生 context.Canceled 根因, 得到: %v", stopErr)
	}

	<-requestDone
	<-runDone
}
