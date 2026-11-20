package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// ==============================
// 编译期契约守卫
// ==============================

// 下列赋值在编译期固定内核自有失败出口的强类型签名：
// 一旦任一自有出口退回原生 error，编译立即失败。
var (
	_ func() *apperr.Error                                  = New().Run
	_ func() *apperr.Error                                  = New().Shutdown
	_ func(PluginInfo, func() Plugin) *apperr.Error         = New().RegisterTool
	_ func(PluginInfo, func() RunnablePlugin) *apperr.Error = New().RegisterWorker
	_ func(PluginInfo, func() RunnablePlugin) *apperr.Error = New().RegisterComponent
)

// assertFunc 在编译期断言目标值的静态类型是否满足期望签名。
//
// 显式类型实参让断言在编译期完成：一旦内核契约退回原生 error，
// 该调用立即编译失败，测试代码无需运行即可暴露契约回退。
func assertFunc[T any](_ T) {}

// contractPlugin 是用于契约断言的最小插件实现。
type contractPlugin struct {
	started chan struct{}
}

func (p *contractPlugin) Start(ctx context.Context) *apperr.Error {
	if p.started != nil {
		close(p.started)
	}
	return nil
}

func (p *contractPlugin) Stop(ctx context.Context) *apperr.Error { return nil }

func (p *contractPlugin) Run(ctx context.Context) *apperr.Error { return nil }

func (p *contractPlugin) Quiesce(ctx context.Context) *apperr.Error { return nil }

// TestCoreContract_SelfOwnedFailuresAreStrongTyped 验证插件生命周期契约、
// 动态激活契约与运行句柄等待契约全部以 [*apperr.Error] 表达失败。
func TestCoreContract_SelfOwnedFailuresAreStrongTyped(t *testing.T) {
	var plugin Plugin = &contractPlugin{}
	var runnable RunnablePlugin = &contractPlugin{}
	var quiescer Quiescer = &contractPlugin{}

	// 编译期契约守卫：任一钩子退回原生 error 都会在此编译失败
	assertFunc[func(context.Context) *apperr.Error](plugin.Start)
	assertFunc[func(context.Context) *apperr.Error](plugin.Stop)
	assertFunc[func(context.Context) *apperr.Error](runnable.Run)
	assertFunc[func(context.Context) *apperr.Error](quiescer.Quiesce)

	// 动态激活是强类型失败出口。
	handle, activateErr := New().Activator().Activate(context.Background(), ctxkey.Key("missing"))
	if activateErr == nil {
		t.Fatal("未注册组件的激活必须返回非 nil 统一错误")
	}
	if handle != nil {
		t.Fatalf("激活失败时不得返回句柄, got %v", handle)
	}
	if activateErr.Code.Code/100 != 500 {
		t.Fatalf("激活失败必须落在 500 分组, got %d", activateErr.Code.Code)
	}
}

// TestCoreContract_RunReturnsRealNilOnSuccess 验证成功路径返回真正的 nil 指针，
// 而不是装入 error 接口后的 typed-nil 载体。
func TestCoreContract_RunReturnsRealNilOnSuccess(t *testing.T) {
	app := New(
		WithGraceTimeout(50*time.Millisecond),
		WithCleanupTimeout(50*time.Millisecond),
		WithShutdownTimeout(500*time.Millisecond),
	)

	started := make(chan struct{})
	workerKey := ctxkey.Key("contract_worker")
	if err := app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &contractWorker{started: started}
	}); err != nil {
		t.Fatalf("注册 Worker 失败: %v", err)
	}

	runErrCh := make(chan *apperr.Error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	<-started
	shutdownErr := app.Shutdown()
	if shutdownErr != nil {
		t.Fatalf("正常关机不应返回错误: %v", shutdownErr)
	}

	runErr := <-runErrCh
	if runErr != nil {
		t.Fatalf("正常结束必须返回真正的 nil 统一错误, got %v", runErr)
	}
}

// TestCoreContract_StopIncompleteMembersReachable 验证关机残留清单中
// 每个成员的结构化错误都逐个可达，且原始错误码与根因不丢失。
func TestCoreContract_StopIncompleteMembersReachable(t *testing.T) {
	app := New(
		WithGraceTimeout(50*time.Millisecond),
		WithCleanupTimeout(50*time.Millisecond),
		WithShutdownTimeout(500*time.Millisecond),
	)

	providerKey := ctxkey.Key("contract_provider")
	panickingKey := ctxkey.Key("contract_panicking")
	started := make(chan struct{})
	if err := app.RegisterTool(PluginInfo{Key: providerKey}, func() Plugin {
		return &contractPlugin{}
	}); err != nil {
		t.Fatalf("注册 provider 失败: %v", err)
	}
	if err := app.RegisterTool(PluginInfo{
		Key:      panickingKey,
		Requires: []ctxkey.Key{providerKey},
	}, func() Plugin {
		return &panicStopPlugin{started: started}
	}); err != nil {
		t.Fatalf("注册 panic 插件失败: %v", err)
	}

	runErrCh := make(chan *apperr.Error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	// 确定性屏障：等待目标插件真正进入 Start，随后触发关机
	<-started
	shutdownErr := app.Shutdown()
	if shutdownErr == nil {
		t.Fatal("Stop panic 必须导致关机返回失败")
	}

	runErr := <-runErrCh
	if runErr == nil {
		t.Fatal("Stop panic 必须导致 Run 返回失败")
	}

	var incomplete *StopIncompleteError
	if !errors.As(runErr, &incomplete) {
		t.Fatalf("必须可经 errors.As 命中 StopIncompleteError, got %v", runErr)
	}
	if len(incomplete.StuckInstances) == 0 {
		t.Fatal("残留清单不得为空")
	}

	for _, stuck := range incomplete.StuckInstances {
		if stuck.Err == nil {
			t.Fatalf("残留成员 %s 不得丢失结构化错误", stuck.InstanceID)
		}
		if stuck.Err.Code.Code != apperr.FrameworkHookPanic.GetCode() {
			t.Fatalf("残留成员错误码不符: got %d", stuck.Err.Code.Code)
		}
		var panicDiag *HookPanicError
		if !errors.As(stuck.Err, &panicDiag) {
			t.Fatalf("残留成员必须可经 errors.As 命中 HookPanicError, got %v", stuck.Err)
		}
		if panicDiag.Stage != "stop" {
			t.Fatalf("panic 阶段不符: got %s", panicDiag.Stage)
		}
		if !errors.Is(stuck.Err, errContractStopPanic) {
			t.Fatalf("残留成员必须保留原始 panic 根因, got %v", stuck.Err)
		}
	}

	// 关机错误本身同样可命中诊断对象与成员
	var shutdownIncomplete *StopIncompleteError
	if !errors.As(shutdownErr, &shutdownIncomplete) {
		t.Fatalf("关机错误必须可经 errors.As 命中 StopIncompleteError, got %v", shutdownErr)
	}

	// 整个失败链上仍可命中诊断对象
	var diag *HookPanicError
	if !errors.As(runErr, &diag) {
		t.Fatalf("失败链必须可经 errors.As 命中 HookPanicError, got %v", runErr)
	}
}

// TestCoreContract_ControlAttribution 验证取消与超时经 control 归一为控制类错误码，
// 同时保留原生 context 哨兵作为根因。
func TestCoreContract_ControlAttribution(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	canceledErr := normalizeErr(apperr.FrameworkInternalError, "等待被取消", canceledCtx.Err())
	if canceledErr == nil {
		t.Fatal("取消原因必须归一为非 nil 统一错误")
	}
	if canceledErr.Code.Code != 49900 {
		t.Fatalf("取消必须归一为 49900, got %d", canceledErr.Code.Code)
	}
	if !errors.Is(canceledErr, context.Canceled) {
		t.Fatalf("必须保留原生 context.Canceled 根因, got %v", canceledErr)
	}

	deadlineCtx, deadlineCancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer deadlineCancel()
	<-deadlineCtx.Done()

	deadlineErr := normalizeErr(apperr.FrameworkInternalError, "等待超时", deadlineCtx.Err())
	if deadlineErr == nil {
		t.Fatal("超时原因必须归一为非 nil 统一错误")
	}
	if deadlineErr.Code.Code != 50401 {
		t.Fatalf("内部操作超时必须归一为 50401, got %d", deadlineErr.Code.Code)
	}
	if !errors.Is(deadlineErr, context.DeadlineExceeded) {
		t.Fatalf("必须保留原生 context.DeadlineExceeded 根因, got %v", deadlineErr)
	}
}

// ==============================
// 契约夹具
// ==============================

// errContractStopPanic 是契约夹具中 Stop panic 的原始根因。
var errContractStopPanic = errors.New("contract stop panic")

// contractWorker 是可确定性控制的契约 Worker。
type contractWorker struct {
	started chan struct{}
}

func (w *contractWorker) Start(ctx context.Context) *apperr.Error { return nil }

func (w *contractWorker) Stop(ctx context.Context) *apperr.Error { return nil }

func (w *contractWorker) Run(ctx context.Context) *apperr.Error {
	close(w.started)
	<-ctx.Done()
	return nil
}

// panicStopPlugin 在 Stop 阶段发生 panic 的契约插件。
type panicStopPlugin struct {
	started chan struct{}
}

func (p *panicStopPlugin) Start(ctx context.Context) *apperr.Error {
	close(p.started)
	return nil
}

func (p *panicStopPlugin) Stop(ctx context.Context) *apperr.Error {
	panic(errContractStopPanic)
}
