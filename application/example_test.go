package application_test

import (
	"context"
	"fmt"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type sampleService struct {
	name string
}

func (s *sampleService) Start(ctx context.Context) error {
	return nil
}

func (s *sampleService) Stop(ctx context.Context) error {
	return nil
}

type sampleWorker struct {
	app *application.App
}

func (w *sampleWorker) Start(ctx context.Context) error {
	return nil
}

func (w *sampleWorker) Run(ctx context.Context) error {
	// 从运行期上下文中安全提取依赖的 Tool 共享服务
	svc, err := application.Get[*sampleService](ctx, ctxkey.Key("sample_service"))
	if err != nil {
		return err.Err
	}
	fmt.Println("Loaded Service:", svc.name)

	// 在收到外部请求时，融合请求上下文与运行时上下文
	reqCtx := context.Background()
	bridged := application.BridgeRequest(reqCtx, ctx, "req_sample_001")
	fmt.Println("Bridged Request ID:", bridged.Value(ctxkey.RequestKey))

	// 异步调用 Shutdown 触发优雅关机
	go func() {
		_ = w.app.Shutdown()
	}()

	// 阻塞等待外部关机或上下文取消
	<-ctx.Done()
	return nil
}

func (w *sampleWorker) Stop(ctx context.Context) error {
	return nil
}

// Example 展示完整的 App 装配、插件注册、拓扑启动、组件提取、请求桥接与平滑关机全流程。
func Example() {
	app := application.New(
		application.WithGraceTimeout(2*time.Second),
		application.WithShutdownTimeout(5*time.Second),
	)

	service := &sampleService{name: "BambooDatabase"}
	_ = app.RegisterTool(application.PluginInfo{
		Key: ctxkey.Key("sample_service"),
	}, func() application.Plugin {
		return service
	})

	worker := &sampleWorker{app: app}
	_ = app.RegisterWorker(application.PluginInfo{
		Key:      ctxkey.Key("sample_worker"),
		Requires: []ctxkey.Key{ctxkey.Key("sample_service")},
	}, func() application.RunnablePlugin {
		return worker
	})

	if err := app.Run(); err != nil {
		fmt.Printf("app run failed: %v\n", err)
		return
	}

	fmt.Println("Final State Stopped:", app.State() == application.StateStopped)

	// Output:
	// Loaded Service: BambooDatabase
	// Bridged Request ID: req_sample_001
	// Final State Stopped: true
}

// ExampleGet 展示从上下文提取组件时的防御性错误处理。
func ExampleGet() {
	// 当直接向未挂载共享容器的原生上下文调用 Get 时，
	// 函数返回类型零值并返回携带 PARAMETER_EMPTY 的错误
	ctx := context.Background()
	_, appErr := application.Get[string](ctx, ctxkey.Key("unmounted_key"))
	if appErr != nil {
		fmt.Println("Error Output:", appErr.Code.Output)
	}

	// Output:
	// Error Output: PARAMETER_EMPTY
}

// ExampleBridgeRequest 展示在未挂载合法运行期容器时 BridgeRequest 的防御性安全降级行为。
func ExampleBridgeRequest() {
	reqCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 当 appLifetimeCtx 为 nil 或未挂载有效容器时，BridgeRequest 安全降级返回原生 reqCtx
	fallbackCtx := application.BridgeRequest(reqCtx, nil, "trace_req_001")
	fmt.Println("Fallback to native reqCtx:", fallbackCtx == reqCtx)

	// Output:
	// Fallback to native reqCtx: true
}

// ExampleDetach 展示在后台异步协程中剥离请求上下文并保留追踪链路标识。
func ExampleDetach() {
	reqCtx := context.WithValue(context.Background(), ctxkey.RequestKey, "trace_async_002")

	detachedCtx, cancel := application.Detach(reqCtx)
	defer cancel()

	fmt.Println("Detached Trace ID:", detachedCtx.Value(ctxkey.RequestKey))

	// Output:
	// Detached Trace ID: trace_async_002
}
