package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type pluginRegistration struct {
	info      PluginInfo
	toolFac   func() Plugin
	workerFac func() RunnablePlugin
	compFac   func() RunnablePlugin
}

type staticInstanceRecord struct {
	info       PluginInfo
	instanceID string
	plugin     Plugin
	runnable   RunnablePlugin
	handle     *runHandleImpl

	// 状态与错误追踪
	started  bool
	startErr error

	expectedStop atomic.Bool // 统一标记 expectedStop，避免误报崩溃
	runExited    atomic.Bool
	runErr       error

	quiesced   bool
	quiesceErr error

	stopped atomic.Bool
	stopErr error

	// 停止过程并发控制
	stopMu      sync.Mutex
	stopStarted bool
	stopDone    chan struct{}
	runClaimed  bool
	runDone     chan struct{}

	// Start 生命周期独立监督屏障
	startMu      sync.Mutex
	startEntered bool
	startDone    chan struct{}
	startCancel  context.CancelFunc

	// panic 记录
	stopPanicked atomic.Bool
	stopPanicErr *HookPanicError
}

// App 是应用级生命周期管理器与运行时核心。
//
// App 承载整个应用的组件依赖拓扑、状态机迁移、看门狗监督与逆拓扑平滑关机。
// 实例通过 [New] 创建，在调用 [App.Run] 前由调用方通过 [App.RegisterTool]、
// [App.RegisterWorker] 和 [App.RegisterComponent] 注册各类组件定义。
//
// # 并发语义
//
// [App.State] 与 [App.Shutdown] 方法并发安全，支持多个 goroutine 同时调用。
// 各类组件注册方法（如 [App.RegisterTool]）内部受互斥锁保护，但必须在单线程装配
// 阶段或 [App.Run] 启动之前调用；在应用启动后调用注册方法将返回 [ErrRegistrationClosed]。
type App struct {
	cfg    Config
	cfgErr error

	stateMu     sync.RWMutex
	state       AppState
	runConsumed bool

	regMu sync.Mutex
	defs  []pluginRegistration

	appID string

	// 运行时组件
	container       *sharedContainer
	coordinator     *shutdownCoordinator
	supervisor      *supervisor
	componentEngine *componentEngine

	// 图与拓扑
	graph *dependencyGraph

	// 静态运行实例字典
	staticMu        sync.RWMutex
	staticInstances map[ctxkey.Key]*staticInstanceRecord

	// 运行结果
	runErrMu sync.Mutex
	runErrs  []error

	// 强制取消与生命周期根
	graceCancelFunc context.CancelFunc
	forceCtx        context.Context

	// 信号监听通道
	sigChan chan os.Signal
}

// New 创建并返回一个新的 [App] 实例。
//
// New 根据传入的 [Option] 选项解析并计算应用超时预算 [Config]。
// 若选项校验失败（例如非正数超时或 GraceTimeout 不小于 ShutdownTimeout），
// 错误将被安全封存，并在后续调用 [App.Run] 或 [App.Shutdown] 时返回。
//
// 每次调用 New 均分配全局唯一的 appID，初始化共享组件容器 [sharedContainer]，
// 并构造关联的关机协调器、Worker 协程看门狗与动态组件激活引擎。
// opts 中的 nil 选项将被自动跳过。
func New(opts ...Option) *App {
	cfg, cfgErr := resolveConfig(opts...)

	appID := fmt.Sprintf("app_%d", time.Now().UnixNano())
	forceCtx, graceCancel := context.WithCancel(context.Background())

	app := &App{
		cfg:             cfg,
		cfgErr:          cfgErr,
		state:           StateNew,
		defs:            make([]pluginRegistration, 0),
		appID:           appID,
		container:       newSharedContainer(appID),
		staticInstances: make(map[ctxkey.Key]*staticInstanceRecord),
		forceCtx:        forceCtx,
		graceCancelFunc: graceCancel,
		sigChan:         make(chan os.Signal, 2),
	}
	app.container.stateReader = app.State
	app.coordinator = newShutdownCoordinator(app)
	app.supervisor = newSupervisor(app)
	app.componentEngine = newComponentEngine(app)
	return app
}

// State 返回当前 [App] 的生命周期状态 [AppState]。
//
// 该方法为只读线程安全操作，内部使用读写锁 [sync.RWMutex] 保护。
// 返回的状态值表示内核全局状态（如 [StateNew]、[StateRunning]、[StateStopping] 等），
// 不代表具体某个任务组件的业务执行结果。
func (a *App) State() AppState {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.state
}

// RegisterTool 注册一个工具态单实例插件（[ModeTool]）。
//
// 工具态插件仅实现 [Plugin] 接口（[Plugin.Start] 与 [Plugin.Stop]），
// 在应用启动时按照拓扑排序优先初始化，为后续启动的 Worker 或 Component
// 提供只读/共享基础设施资源（如数据库连接、缓存客户端或算法节点）。
//
// 参数 info 为组件元信息，方法内部会自动将其 Mode 强制设置为 [ModeTool]
// 并深拷贝存储；factory 为无参工厂函数，在启动阶段实例化组件。
//
// # 错误处理与时序
//
//   - 若 factory 为 nil，返回 [ErrNilFactory]；
//   - 若 App 当前状态不为 [StateNew] 或已调用过 [App.Run]，返回 [ErrRegistrationClosed]。
//
// 注册阶段不执行工厂调用或拓扑校验，实际依赖解析与实例化延迟至 [App.Run] 执行。
func (a *App) RegisterTool(info PluginInfo, factory func() Plugin) error {
	if factory == nil {
		return ErrNilFactory
	}
	info.Mode = ModeTool

	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if a.state != StateNew || a.runConsumed {
		return ErrRegistrationClosed
	}

	a.regMu.Lock()
	defer a.regMu.Unlock()
	a.defs = append(a.defs, pluginRegistration{
		info:    clonePluginInfo(info),
		toolFac: factory,
	})
	return nil
}

// RegisterWorker 注册一个常驻型后台服务插件（[ModeWorker]）。
//
// Worker 插件实现 [RunnablePlugin] 接口，具备常驻事件循环或网络监听能力
// （如 HTTP Web 服务器、gRPC 服务端、异步任务队列消费者等）。
// 在应用启动且共享容器冻结后，内核为每个 Worker 启动独立协程调用其 [RunnablePlugin.Run]，
// 并由看门狗协程进行全周期监督。
//
// 参数 info 为组件元信息，方法内部会自动将其 Mode 强制设置为 [ModeWorker]
// 并深拷贝存储；factory 为无参工厂函数，在启动阶段实例化组件。
//
// # 错误处理与时序
//
//   - 若 factory 为 nil，返回 [ErrNilFactory]；
//   - 若 App 当前状态不为 [StateNew] 或已调用过 [App.Run]，返回 [ErrRegistrationClosed]。
//
// 注册阶段不执行工厂调用或拓扑校验，实际依赖解析与实例化延迟至 [App.Run] 执行。
func (a *App) RegisterWorker(info PluginInfo, factory func() RunnablePlugin) error {
	if factory == nil {
		return ErrNilFactory
	}
	info.Mode = ModeWorker

	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if a.state != StateNew || a.runConsumed {
		return ErrRegistrationClosed
	}

	a.regMu.Lock()
	defer a.regMu.Unlock()
	a.defs = append(a.defs, pluginRegistration{
		info:      clonePluginInfo(info),
		workerFac: factory,
	})
	return nil
}

// RegisterComponent 注册一个按需激发的任务型组件插件（[ModeComponent]）。
//
// Component 插件实现 [RunnablePlugin] 接口，不在应用启动时自动实例化，
// 而是由业务或其他插件通过 [Activator.Activate] 在运行时按需动态激发。
// 每次激发均调用 factory 构造独立实例，并受 [PluginInfo.Overlap] 并发配额约束。
//
// 参数 info 为组件元信息，方法内部会自动将其 Mode 强制设置为 [ModeComponent]
// 并深拷贝存储；factory 为无参工厂函数，在每次激活时调用。
// 按照设计规范，Component 不允许静态声明 Provides，若提供将在构建拓扑图时报错。
//
// # 错误处理与时序
//
//   - 若 factory 为 nil，返回 [ErrNilFactory]；
//   - 若 App 当前状态不为 [StateNew] 或已调用过 [App.Run]，返回 [ErrRegistrationClosed]。
//
// 注册阶段仅登记定义与重入策略，实际依赖解析延迟至 [App.Run] 执行。
func (a *App) RegisterComponent(info PluginInfo, factory func() RunnablePlugin) error {
	if factory == nil {
		return ErrNilFactory
	}
	info.Mode = ModeComponent

	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if a.state != StateNew || a.runConsumed {
		return ErrRegistrationClosed
	}

	a.regMu.Lock()
	defer a.regMu.Unlock()
	a.defs = append(a.defs, pluginRegistration{
		info:    clonePluginInfo(info),
		compFac: factory,
	})
	return nil
}

// Activator 返回当前 [App] 绑定的组件激活器接口 [Activator]。
//
// 返回的激活器用于在应用运行期间动态激发已注册的 [ModeComponent] 插件。
// 即使在应用未启动时也可获取该对象，但实际调用 [Activator.Activate]
// 必须在 App 处于 [StateRunning] 状态下进行，且调用方上下文须具备可信父级身份。
func (a *App) Activator() Activator {
	return a.componentEngine
}

// Run 启动应用拓扑并阻塞等待运行结束或关机。
//
// Run 负责整个应用的确定性启动编排与运行时监督，主要流程如下：
//
//  1. 状态流转与单次运行检查：将状态切换为 [StateStarting]。同一 App 实例仅允许
//     运行一次，重复调用将返回 [ErrAlreadyRun]；在已关闭或停机中的 App 上调用
//     将返回 [ErrAppClosed]；
//  2. 拓扑计算与校验：根据全部已注册组件的 Requires 与 Provides 构建依赖图，
//     并使用 Kahn 算法计算稳定分层拓扑；存在循环依赖时返回 [ErrDependencyCycle]；
//  3. 信号监听注册：自动注册监听 [syscall.SIGINT] 与 [syscall.SIGTERM] 信号，
//     捕获后自动触发 [App.Shutdown]；
//  4. 分层静态启动：按依赖层级从底向顶依次调用 Tool 与 Worker 的工厂函数及 [Plugin.Start]。
//     启动期间上下文挂载受限白名单视图 [startScopeView]，越权读取未声明依赖将失败；
//     若插件实现了 [ctxkey.Exposer]，内核将校验并原子发布其暴露的资源键；
//     启动失败或超时将触发自动停机回滚，并返回包含启动错误的合并错误；
//  5. 发布就绪状态：冻结共享组件容器为只读，并将状态置为 [StateRunning]；
//  6. 启动 Worker 看门狗：为所有 Worker 启动独立协程运行 [RunnablePlugin.Run]；
//     在 Running 期间若任何 Worker 提前退出（包括返回 nil），看门狗将其判定为异常崩溃，
//     触发应用级平滑关机并记录 [ErrWorkerUnexpectedExit]；
//  7. 阻塞等待：等待关机协调器完成或 Worker 看门狗错误触发停机，并返回合并后的错误。
//
// # 并发与时序语义
//
// Run 本身不是线程安全的，不可被多个 goroutine 同时并发执行。
// 启动完成后，外部 goroutine 可安全调用 [App.Shutdown] 触发优雅关机。
func (a *App) Run() error {
	// 1. 状态流转与单次运行权限检查
	a.stateMu.Lock()
	if a.runConsumed {
		a.stateMu.Unlock()
		return ErrAlreadyRun
	}
	if a.state == StateStopped || a.state == StateStopping || a.state == StateIncomplete {
		a.stateMu.Unlock()
		return ErrAppClosed
	}
	a.runConsumed = true
	a.state = StateStarting
	// 快照并冻结注册定义
	a.regMu.Lock()
	defsSnapshot := make([]pluginRegistration, len(a.defs))
	copy(defsSnapshot, a.defs)
	a.regMu.Unlock()
	a.stateMu.Unlock()

	// 2. 读取 New 已解析配置与错误（不重读 env）
	if a.cfgErr != nil {
		a.stateMu.Lock()
		a.state = StateStopped
		a.stateMu.Unlock()
		return a.cfgErr
	}
	cfg := a.cfg

	// 3. 构建拓扑图（在任何工厂调用前完成）
	infos := make([]PluginInfo, 0, len(defsSnapshot))
	defMap := make(map[ctxkey.Key]pluginRegistration, len(defsSnapshot))
	for _, def := range defsSnapshot {
		infos = append(infos, def.info)
		defMap[def.info.Key] = def
	}

	graph, err := buildGraph(infos)
	if err != nil {
		a.stateMu.Lock()
		a.state = StateStopped
		a.stateMu.Unlock()
		return err
	}
	a.graph = graph

	// 登记 ModeComponent 规范化元信息至 componentEngine
	for _, def := range defsSnapshot {
		if def.info.Mode == ModeComponent {
			normalizedInfo := graph.infoByKey[def.info.Key]
			a.componentEngine.RegisterComponentDef(normalizedInfo, def.compFac)
		}
	}

	// 4. 初始化信号监听（forceCtx/graceCancelFunc 由 New 初始化，此处不重写）
	signal.Notify(a.sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(a.sigChan)

	// 监听系统信号自动触发关机
	go func() {
		select {
		case <-a.sigChan:
			_ = a.Shutdown()
		case <-a.coordinator.done:
		}
	}()

	// 5. 启动流水线：按拓扑分层启动 Tool 和 Worker
	startErr := a.startStaticLayers(cfg, defMap)
	if startErr != nil {
		// 启动失败，Shutdown 进行回滚
		shutdownErr := a.Shutdown()
		a.stateMu.RLock()
		st := a.state
		a.stateMu.RUnlock()
		if st == StateStopped || st == StateIncomplete {
			// 如果是在 Starting 收到外部中断，返回 ErrStartupInterrupted 并保留同期错误
			if errors.Is(startErr, ErrStartupInterrupted) {
				return errors.Join(ErrStartupInterrupted, shutdownErr)
			}
		}
		return errors.Join(startErr, shutdownErr)
	}

	// 6. 检查是否在启动期间已经收到了关机请求
	a.stateMu.Lock()
	if a.state == StateStopping || a.state == StateStopped || a.state == StateIncomplete {
		a.stateMu.Unlock()
		shutdownErr := a.Shutdown()
		return errors.Join(ErrStartupInterrupted, shutdownErr)
	}
	// 冻结共享容器为只读，发布 StateRunning
	a.container.freeze()
	a.state = StateRunning
	a.stateMu.Unlock()

	// 7. 启动常驻 Worker.Run 并监督看门狗
	watchdogErrChan := make(chan error, len(defsSnapshot)+1)
	a.launchWorkers(watchdogErrChan)

	// 8. 等待运行结束（Worker 异常退出、信号触发、或外部主动调用 Shutdown）
	select {
	case workerErr := <-watchdogErrChan:
		// Worker 早退或 panic 触发失败关机
		if workerErr != nil {
			a.recordRunErr(workerErr)
		}
		shutdownErr := a.Shutdown()
		return errors.Join(a.getRunErr(), shutdownErr)
	case <-a.coordinator.done:
		// 关机协调器完成
		return errors.Join(a.getRunErr(), a.coordinator.res.err)
	}
}

func (a *App) recordRunErr(err error) {
	if err == nil {
		return
	}
	a.runErrMu.Lock()
	defer a.runErrMu.Unlock()
	a.runErrs = append(a.runErrs, err)
}

func (a *App) getRunErr() error {
	a.runErrMu.Lock()
	defer a.runErrMu.Unlock()
	return errors.Join(a.runErrs...)
}

func (a *App) startStaticLayers(cfg Config, defMap map[ctxkey.Key]pluginRegistration) (retErr error) {
	for _, layer := range a.graph.layers {
		for _, key := range layer {
			def := defMap[key]
			if def.info.Mode == ModeComponent {
				// Component 注册仅保存定义，静态启动时不实例化
				continue
			}

			// 检查 Starting 期间是否已被中断
			a.stateMu.RLock()
			st := a.state
			a.stateMu.RUnlock()
			if st != StateStarting {
				return ErrStartupInterrupted
			}

			// 1. 调用工厂函数实例化（不在锁内调用）
			inst, instID, facErr := a.createPluginInstance(def)
			if facErr != nil {
				return facErr
			}

			rec := &staticInstanceRecord{
				info:       def.info,
				instanceID: instID,
				plugin:     inst,
				handle:     newRunHandle(instID, key),
			}
			if runnable, ok := inst.(RunnablePlugin); ok {
				rec.runnable = runnable
			}

			// 2. 构造 startCtx（受限视图 + 独立超时）
			allowedMap := make(map[ctxkey.Key]struct{}, len(def.info.Requires))
			for _, r := range def.info.Requires {
				allowedMap[r] = struct{}{}
			}
			scopeView := &startScopeView{
				container:   a.container,
				ownerKey:    key,
				allowedKeys: allowedMap,
			}

			startTimeout := cfg.StartTimeout
			if startTimeout <= 0 {
				startTimeout = 15 * time.Second
			}
			startCtx, startCancel := context.WithTimeout(context.Background(), startTimeout)

			// 检查进入 Start 前 App 状态并原子登记已 claimStart 的 staticInstanceRecord
			a.stateMu.Lock()
			if a.state != StateStarting {
				a.stateMu.Unlock()
				startCancel()
				return ErrStartupInterrupted
			}

			// 在 record 独立互斥锁下安全封存 startCancel/startDone 并设置 started=true
			startDone := make(chan struct{})
			rec.startMu.Lock()
			rec.startEntered = true
			rec.startDone = startDone
			rec.startCancel = startCancel
			rec.started = true
			rec.startMu.Unlock()

			a.staticMu.Lock()
			a.staticInstances[key] = rec
			a.staticMu.Unlock()
			a.stateMu.Unlock()

			// 挂载受限视图与身份
			ctxWithView := context.WithValue(startCtx, scopeViewCtxKey, scopeView)
			ctxWithView = context.WithValue(ctxWithView, containerCtxKey, a.container)
			ctxWithView = context.WithValue(ctxWithView, forceCtxKey, a.forceCtx)
			ctxWithView = context.WithValue(ctxWithView, metaCtxKey, InstanceMeta{
				InstanceID: instID,
				PluginKey:  key,
				TraceID:    fmt.Sprintf("trace_%s", instID),
			})

			// 3. 将 safeCallStart 放入内核明确持有的 goroutine 中异步执行
			// 启动驱动 select 真实完成与 startCtx.Done，使忽略 ctx 的 Start 也能在超时/取消后让 Run 失败关机
			startResCh := make(chan error, 1)

			go func() {
				resErr := a.safeCallStart(rec, ctxWithView)
				rec.startMu.Lock()
				rec.startErr = resErr
				rec.startCancel = nil
				rec.startMu.Unlock()

				close(startDone)
				startResCh <- resErr
				startCancel()
			}()

			var err error
			select {
			case err = <-startResCh:
			case <-startCtx.Done():
				// 非阻塞重查 startResCh，若真实结果已就绪优先消费（正常成功不假超时）
				select {
				case err = <-startResCh:
				default:
					a.stateMu.RLock()
					isStopping := (a.state == StateStopping || a.state == StateStopped || a.state == StateIncomplete)
					a.stateMu.RUnlock()

					var retErr error
					if isStopping {
						retErr = errors.Join(ErrStartupInterrupted, startCtx.Err())
					} else {
						retErr = fmt.Errorf("plugin %s start timed out: %w", key, startCtx.Err())
					}

					// 仅启动一条内核收尾 goroutine 等待 hook 真退后幂等 safeCallStop，绝不落入下面 err 分支同步 Stop/裸写 rec.startErr
					go func() {
						<-startDone
						_ = a.safeCallStop(rec, cfg)
					}()
					return retErr
				}
			}

			// 检查 Start 真实返回后 App 是否已处于 Stopping 状态
			a.stateMu.RLock()
			afterStopping := (a.state == StateStopping || a.state == StateStopped || a.state == StateIncomplete)
			a.stateMu.RUnlock()

			if err != nil {
				// Start 失败：Start 已真退（startDone 已 close），先为当前失败实例调用唯一一次 Stop，再退出以触发逆序回滚
				stopErr := a.safeCallStop(rec, cfg)
				if afterStopping && errors.Is(err, context.Canceled) {
					return errors.Join(ErrStartupInterrupted, err, stopErr)
				}
				return errors.Join(err, stopErr)
			}

			if afterStopping {
				// Start 已真退且 App 已切入 Stopping：先执行幂等 Stop 兜底清理，再退出交由关机协调
				stopErr := a.safeCallStop(rec, cfg)
				return errors.Join(ErrStartupInterrupted, stopErr)
			}

			// 4. 交付校验 Exposer 与发布共享条目
			exposeErr := a.publishExposedKeys(def.info, inst)
			if exposeErr != nil {
				stopErr := a.safeCallStop(rec, cfg)
				return errors.Join(exposeErr, stopErr)
			}
		}
	}
	return nil
}

func (a *App) createPluginInstance(def pluginRegistration) (Plugin, string, error) {
	instID := fmt.Sprintf("%s_%d", def.info.Key, time.Now().UnixNano())
	var inst Plugin
	var facErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				facErr = &HookPanicError{
					PluginKey:  def.info.Key,
					InstanceID: instID,
					Stage:      "factory",
					Value:      r,
					Stack:      debug.Stack(),
				}
			}
		}()
		switch def.info.Mode {
		case ModeTool:
			if def.toolFac != nil {
				inst = def.toolFac()
			}
		case ModeWorker:
			if def.workerFac != nil {
				inst = def.workerFac()
			}
		}
	}()

	if facErr != nil {
		return nil, "", facErr
	}
	if inst == nil || isTypedNil(inst) {
		return nil, "", fmt.Errorf("%w: plugin %s", ErrNilPlugin, def.info.Key)
	}
	return inst, instID, nil
}

func isTypedNil(val any) bool {
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

func (a *App) safeCallStart(rec *staticInstanceRecord, ctx context.Context) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = &HookPanicError{
				PluginKey:  rec.info.Key,
				InstanceID: rec.instanceID,
				Stage:      "start",
				Value:      r,
				Stack:      debug.Stack(),
			}
		}
	}()
	return rec.plugin.Start(ctx)
}

func (a *App) safeCallStop(rec *staticInstanceRecord, cfg Config, budget ...*budgetMetadata) (retErr error) {
	rec.stopMu.Lock()
	if rec.stopDone == nil {
		rec.stopDone = make(chan struct{})
	}
	if rec.stopStarted {
		done := rec.stopDone
		rec.stopMu.Unlock()
		<-done
		if rec.stopPanicked.Load() && rec.stopPanicErr != nil {
			return rec.stopPanicErr
		}
		return rec.stopErr
	}
	rec.stopStarted = true
	done := rec.stopDone
	rec.stopMu.Unlock()

	defer func() {
		close(done)
	}()

	defer func() {
		if r := recover(); r != nil {
			rec.stopPanicked.Store(true)
			panicErr := &HookPanicError{
				PluginKey:  rec.info.Key,
				InstanceID: rec.instanceID,
				Stage:      "stop",
				Value:      r,
				Stack:      debug.Stack(),
			}
			rec.stopPanicErr = panicErr
			retErr = panicErr

			// Stop panic: 标记为 EntryUnavailable 并保留依赖
			a.container.setOwnerState(rec.info.Key, EntryUnavailable)
			a.coordinator.recordStuck(StuckInstance{
				PluginKey:    rec.info.Key,
				InstanceID:   rec.instanceID,
				Stage:        "stop-panic",
				RetainedDeps: rec.info.Requires,
				Err:          panicErr,
			})
		}
	}()

	var stopCtx context.Context
	var cancel context.CancelFunc

	if len(budget) > 0 && budget[0] != nil {
		meta := budget[0]
		// 实际 Stop 收到以固定 totalDeadline 为 deadline 并携带 budgetCtxKey metadata 的 Context
		if !meta.totalDeadline.IsZero() {
			stopCtx, cancel = context.WithDeadline(context.Background(), meta.totalDeadline)
		} else {
			cleanupLimit := cfg.CleanupTimeout
			if cleanupLimit <= 0 {
				cleanupLimit = 10 * time.Second
			}
			stopCtx, cancel = context.WithTimeout(context.Background(), cleanupLimit)
		}
		stopCtx = context.WithValue(stopCtx, budgetCtxKey, meta)
	} else {
		// 无 meta 的启动失败旧调用暂保原策略并明确仍待移入协调
		cleanupLimit := cfg.CleanupTimeout
		if cleanupLimit <= 0 {
			cleanupLimit = 10 * time.Second
		}
		stopCtx, cancel = context.WithTimeout(context.Background(), cleanupLimit)
	}
	defer cancel()

	stopErr := rec.plugin.Stop(stopCtx)
	rec.stopErr = stopErr
	if stopErr != nil {
		retErr = stopErr
	}

	rec.stopMu.Lock()
	claimed := rec.runClaimed
	rDone := rec.runDone
	rec.stopMu.Unlock()

	if claimed && rDone != nil {
		<-rDone
	}

	// 只有正常真返回后才能 stopped.Store(true)
	rec.stopped.Store(true)

	// 正常返回（无论是否有释放错误）：标记条目 EntryStopped
	a.container.setOwnerState(rec.info.Key, EntryStopped)
	return retErr
}

func (a *App) publishExposedKeys(info PluginInfo, inst Plugin) error {
	// 1. 先在锁外安全执行 Expose() 与 recover，不持有任何容器锁
	exposer, hasExposer := inst.(ctxkey.Exposer)
	if len(info.Provides) > 0 && !hasExposer {
		return fmt.Errorf("%w: plugin %s declared provides but does not implement ctxkey.Exposer", ErrExposedKeysMismatch, info.Key)
	}

	var exposedMap map[ctxkey.Key]any
	if hasExposer {
		var exposePanic error
		func() {
			defer func() {
				if r := recover(); r != nil {
					exposePanic = &HookPanicError{
						PluginKey: info.Key,
						Stage:     "expose",
						Value:     r,
						Stack:     debug.Stack(),
					}
				}
			}()
			exposedMap = exposer.Expose()
		}()

		if exposePanic != nil {
			return exposePanic
		}

		// 严格校验暴露键集合：不能为 nil、漏键、多键
		if exposedMap == nil {
			return fmt.Errorf("%w: plugin %s exposed nil map", ErrExposedKeysMismatch, info.Key)
		}
		if len(exposedMap) != len(info.Provides) {
			return fmt.Errorf("%w: exposed count %d does not match declared provides count %d", ErrExposedKeysMismatch, len(exposedMap), len(info.Provides))
		}

		declaredSet := make(map[ctxkey.Key]struct{}, len(info.Provides))
		for _, p := range info.Provides {
			declaredSet[p] = struct{}{}
		}

		for k, v := range exposedMap {
			if _, ok := declaredSet[k]; !ok {
				return fmt.Errorf("%w: exposed key %s was not declared in provides", ErrExposedKeysMismatch, k)
			}
			if k == info.Key {
				return fmt.Errorf("%w: exposed key cannot overwrite plugin key %s", ErrDuplicateExposedKey, k)
			}
			if v == nil || isTypedNil(v) {
				return fmt.Errorf("%w: exposed nil value for key %s", ErrExposedKeysMismatch, k)
			}
		}
	}

	// 2. 组装待原子发布的 batch（主键 + 所有资源键）
	batch := make([]publishItem, 0, 1+len(exposedMap))
	batch = append(batch, publishItem{
		key:      info.Key,
		ownerKey: info.Key,
		val:      inst,
		state:    EntryReady,
	})
	for k, v := range exposedMap {
		batch = append(batch, publishItem{
			key:      k,
			ownerKey: info.Key,
			val:      v,
			state:    EntryReady,
		})
	}

	// 3. 在容器锁内执行单次原子发布：拒绝覆盖已存在键、拒绝冻结后写入；失败时一个键都不入库
	return a.container.publishBatch(batch)
}

func (a *App) launchWorkers(watchdogErrChan chan error) {
	a.staticMu.RLock()
	defer a.staticMu.RUnlock()

	for _, rec := range a.staticInstances {
		if rec.info.Mode != ModeWorker || rec.runnable == nil {
			continue
		}

		currentRec := rec
		go func() {
			a.stateMu.RLock()
			currentRec.stopMu.Lock()
			if a.state != StateRunning || currentRec.stopStarted {
				currentRec.stopMu.Unlock()
				a.stateMu.RUnlock()
				return
			}
			currentRec.runClaimed = true
			done := make(chan struct{})
			currentRec.runDone = done
			currentRec.stopMu.Unlock()
			a.stateMu.RUnlock()

			defer close(done)

			workerCtx, workerCancel := context.WithCancel(context.Background())
			defer workerCancel()

			// 挂载运行期完整容器、元数据与可信凭据
			lifetimeCtx := context.WithValue(workerCtx, containerCtxKey, a.container)
			lifetimeCtx = context.WithValue(lifetimeCtx, forceCtxKey, a.forceCtx)
			lifetimeCtx = context.WithValue(lifetimeCtx, metaCtxKey, InstanceMeta{
				InstanceID: currentRec.instanceID,
				PluginKey:  currentRec.info.Key,
				TraceID:    fmt.Sprintf("trace_%s", currentRec.instanceID),
			})
			lifetimeCtx = context.WithValue(lifetimeCtx, trustedAuthKey, trustedAuthToken{
				appID:      a.appID,
				instanceID: currentRec.instanceID,
				pluginKey:  currentRec.info.Key,
			})

			// 当全局 forceCtx 触发取消时，向 worker 传达
			stopWatch := make(chan struct{})
			go func() {
				select {
				case <-a.forceCtx.Done():
					workerCancel()
				case <-stopWatch:
				}
			}()
			defer close(stopWatch)

			if a.componentEngine != nil {
				a.componentEngine.registerWorker(currentRec.instanceID, currentRec.info.Key)
			}

			var runErr error
			func() {
				defer func() {
					if r := recover(); r != nil {
						runErr = &HookPanicError{
							PluginKey:  currentRec.info.Key,
							InstanceID: currentRec.instanceID,
							Stage:      "run",
							Value:      r,
							Stack:      debug.Stack(),
						}
					}
				}()
				runErr = currentRec.runnable.Run(lifetimeCtx)
			}()

			if a.componentEngine != nil {
				closeCause := runErr
				if closeCause == nil {
					closeCause = ErrParentClosing
				}
				a.componentEngine.closeParent(currentRec.instanceID, closeCause)
				_ = a.componentEngine.waitParent(context.Background(), currentRec.instanceID)
			}

			currentRec.runExited.Store(true)
			currentRec.runErr = runErr

			// 看门狗：完成事件与进入 Stopping 使用同一同步边界判定
			a.stateMu.Lock()
			currentState := a.state
			isExpected := currentRec.expectedStop.Load()
			a.stateMu.Unlock()

			if currentState == StateRunning && !isExpected {
				// 在 Running 状态下提前退出，无论 nil 还是错误，均触发失败关机
				if runErr == nil {
					watchdogErrChan <- fmt.Errorf("%w: worker %s exited unexpectedly with nil error", ErrWorkerUnexpectedExit, currentRec.info.Key)
				} else {
					watchdogErrChan <- fmt.Errorf("%w: worker %s: %v", ErrWorkerUnexpectedExit, currentRec.info.Key, runErr)
				}
			} else {
				// 正常关机或预期停止
				if runErr != nil && !errors.Is(runErr, context.Canceled) {
					// 真实非取消错误并入账本
					a.recordRunErr(runErr)
				}
			}
		}()
	}
}

// Shutdown 执行关机协调流程，平滑释放应用静态与动态资源。
//
// 关机流程严格遵循逆拓扑时序与超时预算控制，主要执行步骤如下：
//
//  1. 协调并发控制与状态流转：首次调用将 App 状态原子切换至 [StateStopping]，
//     后续并发或重复调用共享单次协调的唯一结果通道，等待首次关机执行完毕；
//  2. 触发取消通知：向正在执行 Start 钩子的插件发送取消信号，并通知动态组件引擎
//     全局关闸拒绝新激活；同时开启基于 [Config.GraceTimeout] 的强制取消定时器；
//  3. 逆拓扑分层停机：从顶层消费者到底层提供者逆序遍历所有已启动的静态实例。
//     若插件实现了 [Quiescer]，先调用 [Quiescer.Quiesce] 拒绝新请求并排空在途流量；
//     之后调用 [Plugin.Stop] 进行实际资源释放与连接关闭；
//  4. 动态依赖保持：若底层提供者仍被活跃的动态 Component 任务占用，停机流程将
//     暂缓停机并等待动态任务完成或关机总超时 [Config.ShutdownTimeout]；
//  5. 判定最终状态：若所有实例均正常退出，应用状态置为 [StateStopped]；
//     若存在超时未退或 Stop 发生 panic 的实例，将其作为 [StuckInstance]
//     安全隔离并保留其底层依赖，应用状态置为 [StateIncomplete]，
//     并返回合并了 [StopIncompleteError] 的错误。
//
// # 幂等性与并发安全
//
// 多次并发或串行调用 Shutdown 是幂等且协程安全的，所有调用方均获得相同的结果错误。
func (a *App) Shutdown() error {
	a.coordinator.mu.Lock()
	if a.coordinator.started {
		a.coordinator.mu.Unlock()
		<-a.coordinator.done
		return a.coordinator.res.err
	}
	a.coordinator.started = true
	a.coordinator.mu.Unlock()

	// 1. 同步原子切换状态至 Stopping，并统一标记所有已创建 Worker expectedStop
	a.stateMu.Lock()
	oldState := a.state
	if oldState == StateNew {
		// New 状态下直接封存空 App
		a.state = StateStopped
		a.stateMu.Unlock()
		shutErr := a.cfgErr
		a.coordinator.res = shutdownResult{err: shutErr}
		close(a.coordinator.done)
		return shutErr
	}
	a.state = StateStopping

	var cancelsToCall []context.CancelFunc
	a.staticMu.RLock()
	for _, rec := range a.staticInstances {
		rec.expectedStop.Store(true)
		rec.startMu.Lock()
		if rec.startCancel != nil {
			cancelsToCall = append(cancelsToCall, rec.startCancel)
		}
		rec.startMu.Unlock()
	}
	a.staticMu.RUnlock()
	a.stateMu.Unlock()

	// 锁外调用正在执行 Start 的 cancel
	for _, cancelFn := range cancelsToCall {
		cancelFn()
	}

	if a.componentEngine != nil {
		a.componentEngine.requestStopAll(ErrShuttingDown)
	}

	// 2. 读取配置，若配置有误直接返回错误（无默认静默降级）
	if a.cfgErr != nil {
		a.stateMu.Lock()
		a.state = StateStopped
		a.stateMu.Unlock()
		a.coordinator.res = shutdownResult{err: a.cfgErr}
		close(a.coordinator.done)
		return a.cfgErr
	}
	cfg := a.cfg

	// 3. 开启自然排空定时器与总截止时间
	startTime := time.Now()
	totalDeadline := startTime.Add(cfg.ShutdownTimeout)
	graceDeadline := startTime.Add(cfg.GraceTimeout)

	// 独立 Grace 取消定时器：20s 到期后独立触发强制取消根通知
	graceTimer := time.AfterFunc(cfg.GraceTimeout, func() {
		if a.graceCancelFunc != nil {
			a.graceCancelFunc()
		}
	})
	defer graceTimer.Stop()

	// 4. 逆序拓扑执行 Quiesce 与 Stop
	shutdownErr := a.executeShutdownTopology(cfg, totalDeadline, graceDeadline)

	// 5. 判定最终状态：Stopped 或 Incomplete
	a.stateMu.Lock()
	stuckList := a.coordinator.getStuckInstances()
	if len(stuckList) > 0 {
		a.state = StateIncomplete
		incompleteErr := &StopIncompleteError{StuckInstances: stuckList}
		shutdownErr = errors.Join(shutdownErr, incompleteErr)
	} else {
		a.state = StateStopped
	}
	a.stateMu.Unlock()

	a.coordinator.res = shutdownResult{err: shutdownErr}
	close(a.coordinator.done)
	return shutdownErr
}

func (a *App) markRetainedTransitiveClosure(pluginKey ctxkey.Key, retained map[ctxkey.Key]struct{}) {
	if a.graph == nil {
		return
	}
	info, ok := a.graph.infoByKey[pluginKey]
	if !ok {
		return
	}
	for _, req := range info.Requires {
		providerKey, exists := a.graph.providerByKey[req]
		if !exists {
			providerKey = req
		}
		if _, seen := retained[providerKey]; !seen {
			retained[providerKey] = struct{}{}
			a.markRetainedTransitiveClosure(providerKey, retained)
		}
	}
}

// computeDynamicHoldProviders 计算当前动态组件占用的 providerKey 及其闭包
func (a *App) computeDynamicHoldProviders(activeDeps map[ctxkey.Key]int) map[ctxkey.Key]struct{} {
	holds := make(map[ctxkey.Key]struct{})
	if a.graph == nil || len(activeDeps) == 0 {
		return holds
	}
	for reqKey, count := range activeDeps {
		if count <= 0 {
			continue
		}
		providerKey, exists := a.graph.providerByKey[reqKey]
		if !exists {
			providerKey = reqKey
		}
		holds[providerKey] = struct{}{}
		a.markRetainedTransitiveClosure(providerKey, holds)
	}
	return holds
}

func (a *App) stopStaticInstanceRecord(rec *staticInstanceRecord, cfg Config, totalDeadline, graceDeadline time.Time, retainedProviders map[ctxkey.Key]struct{}) error {
	// 关机路径调用 Stop 前确认 Start 真实返回；未退等待到 totalDeadline，超时报 start 残留并保依赖
	if !a.waitForStartDone(rec, totalDeadline, retainedProviders) {
		return nil
	}

	// 检查是否已经超出总截止时间
	if time.Now().After(totalDeadline) {
		a.coordinator.recordStuck(StuckInstance{
			PluginKey:    rec.info.Key,
			InstanceID:   rec.instanceID,
			Stage:        "timeout",
			RetainedDeps: rec.info.Requires,
			Err:          context.DeadlineExceeded,
		})
		retainedProviders[rec.info.Key] = struct{}{}
		a.markRetainedTransitiveClosure(rec.info.Key, retainedProviders)
		return nil
	}

	var errs []error
	// 1. 若实现了 Quiescer，先执行 Quiesce（内核持有 goroutine + buffered 结果 chan，外层 select totalDeadline）
	if quiescer, ok := rec.plugin.(Quiescer); ok && !rec.quiesced {
		rec.quiesced = true

		quiesceDoneChan := make(chan error, 1)
		go func(targetRec *staticInstanceRecord, q Quiescer) {
			quiesceDoneChan <- a.safeCallQuiesce(targetRec, q, totalDeadline, graceDeadline, cfg.CleanupTimeout)
		}(rec, quiescer)

		var quiesceTimer *time.Timer
		var quiesceTimeoutChan <-chan time.Time
		qRem := time.Until(totalDeadline)
		if qRem > 0 {
			quiesceTimer = time.NewTimer(qRem)
			defer quiesceTimer.Stop()
			quiesceTimeoutChan = quiesceTimer.C
		} else {
			ch := make(chan time.Time, 1)
			ch <- time.Now()
			quiesceTimeoutChan = ch
		}

		select {
		case quiesceErr := <-quiesceDoneChan:
			if quiesceTimer != nil {
				quiesceTimer.Stop()
			}
			if quiesceErr != nil {
				rec.quiesceErr = quiesceErr
				errs = append(errs, quiesceErr)
			}
		case <-quiesceTimeoutChan:
			a.coordinator.recordStuck(StuckInstance{
				PluginKey:    rec.info.Key,
				InstanceID:   rec.instanceID,
				Stage:        "quiesce",
				RetainedDeps: rec.info.Requires,
				Err:          context.DeadlineExceeded,
			})
			retainedProviders[rec.info.Key] = struct{}{}
			a.markRetainedTransitiveClosure(rec.info.Key, retainedProviders)
			// 超时保护闭包：不调用该实例 Stop、不置 stopped/retired、不重复 Quiesce、不假释放，直接返回
			return errors.Join(errs...)
		}
	}

	// 2. 执行 Stop 真实收尾：锁外受管异步调用，用有缓冲完成 chan 在外层 select
	budget := &budgetMetadata{
		graceDeadline: graceDeadline,
		totalDeadline: totalDeadline,
		cleanupLimit:  cfg.CleanupTimeout,
	}

	stopDoneChan := make(chan error, 1)
	go func(targetRec *staticInstanceRecord) {
		stopDoneChan <- a.safeCallStop(targetRec, cfg, budget)
	}(rec)

	var stopTimer *time.Timer
	var stopTimeoutChan <-chan time.Time
	rem := time.Until(totalDeadline)
	if rem > 0 {
		stopTimer = time.NewTimer(rem)
		defer stopTimer.Stop()
		stopTimeoutChan = stopTimer.C
	} else {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		stopTimeoutChan = ch
	}

	select {
	case stopErr := <-stopDoneChan:
		if stopTimer != nil {
			stopTimer.Stop()
		}
		if stopErr != nil {
			errs = append(errs, stopErr)
		}
	case <-stopTimeoutChan:
		a.coordinator.recordStuck(StuckInstance{
			PluginKey:    rec.info.Key,
			InstanceID:   rec.instanceID,
			Stage:        "stop",
			RetainedDeps: rec.info.Requires,
			Err:          context.DeadlineExceeded,
		})
	}

	// 检查该实例是否未正常退出
	if !rec.stopped.Load() {
		retainedProviders[rec.info.Key] = struct{}{}
		a.markRetainedTransitiveClosure(rec.info.Key, retainedProviders)
	}

	return errors.Join(errs...)
}

func (a *App) waitForStartDone(rec *staticInstanceRecord, totalDeadline time.Time, retainedProviders map[ctxkey.Key]struct{}) bool {
	rec.startMu.Lock()
	done := rec.startDone
	entered := rec.startEntered
	rec.startMu.Unlock()

	if !entered || done == nil {
		return true
	}

	select {
	case <-done:
		return true
	default:
	}

	rem := time.Until(totalDeadline)
	if rem <= 0 {
		a.coordinator.recordStuck(StuckInstance{
			PluginKey:    rec.info.Key,
			InstanceID:   rec.instanceID,
			Stage:        "start",
			RetainedDeps: rec.info.Requires,
			Err:          context.DeadlineExceeded,
		})
		a.markRetainedTransitiveClosure(rec.info.Key, retainedProviders)
		return false
	}

	timer := time.NewTimer(rem)
	defer timer.Stop()

	select {
	case <-done:
		return true
	case <-timer.C:
		a.coordinator.recordStuck(StuckInstance{
			PluginKey:    rec.info.Key,
			InstanceID:   rec.instanceID,
			Stage:        "start",
			RetainedDeps: rec.info.Requires,
			Err:          context.DeadlineExceeded,
		})
		a.markRetainedTransitiveClosure(rec.info.Key, retainedProviders)
		return false
	}
}

func (a *App) executeShutdownTopology(cfg Config, totalDeadline, graceDeadline time.Time) error {
	var errs []error

	if a.graph == nil || len(a.graph.layers) == 0 {
		// 没有拓扑图，按已创建实例逆序清理
		a.staticMu.RLock()
		var instances []*staticInstanceRecord
		for _, rec := range a.staticInstances {
			instances = append(instances, rec)
		}
		a.staticMu.RUnlock()

		for i := len(instances) - 1; i >= 0; i-- {
			rec := instances[i]
			if rec.started && !rec.stopped.Load() {
				if !a.waitForStartDone(rec, totalDeadline, nil) {
					continue
				}
				if err := a.safeCallStop(rec, cfg); err != nil {
					errs = append(errs, err)
				}
			}
		}
		return errors.Join(errs...)
	}

	retainedProviders := make(map[ctxkey.Key]struct{})

	// 初始保留：从 coordinator 快照（例如启动失败或 Shutdown 前已发生的 Stop panic）恢复保护
	for _, stuck := range a.coordinator.getStuckInstances() {
		retainedProviders[stuck.PluginKey] = struct{}{}
		a.markRetainedTransitiveClosure(stuck.PluginKey, retainedProviders)
	}

	// 收集所有需要停机的静态实例，并建立待停机项清单
	type stopCandidate struct {
		key ctxkey.Key
		rec *staticInstanceRecord
	}
	var pendingList []*stopCandidate

	// 按分层依赖逆序展开（从顶层消费者到底层提供者）
	layers := a.graph.layers
	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]
		for _, key := range layer {
			a.staticMu.RLock()
			rec, ok := a.staticInstances[key]
			a.staticMu.RUnlock()

			if ok && rec.started && !rec.stopped.Load() {
				pendingList = append(pendingList, &stopCandidate{
					key: key,
					rec: rec,
				})
			}
		}
	}

	for len(pendingList) > 0 {
		var activeDeps map[ctxkey.Key]int
		var depCh <-chan struct{}
		if a.componentEngine != nil {
			activeDeps, depCh = a.componentEngine.dependencySnapshot()
		}
		dynamicHolds := a.computeDynamicHoldProviders(activeDeps)

		madeProgress := false
		var remaining []*stopCandidate

		for _, item := range pendingList {
			key := item.key
			rec := item.rec

			// 已停止则跳过
			if rec.stopped.Load() {
				madeProgress = true
				continue
			}

			// 如果已被硬故障/panic/stuck 保护，保留运行，不调用 Stop
			if _, isRetained := retainedProviders[key]; isRetained {
				madeProgress = true
				continue
			}

			// 检查是否仍被动态组件占用
			if _, isHeld := dynamicHolds[key]; isHeld {
				// 动态占用暂挂，留待后续轮次复访
				remaining = append(remaining, item)
				continue
			}

			// 允许停机
			madeProgress = true
			if stopErr := a.stopStaticInstanceRecord(rec, cfg, totalDeadline, graceDeadline, retainedProviders); stopErr != nil {
				errs = append(errs, stopErr)
			}
		}

		pendingList = remaining
		if len(pendingList) == 0 {
			break
		}

		if !madeProgress {
			// 没有可推进项，无轮询等待依赖变化通知或总截止时间
			rem := time.Until(totalDeadline)
			if rem <= 0 {
				break
			}
			timer := time.NewTimer(rem)
			if depCh != nil {
				select {
				case <-depCh:
					timer.Stop()
				case <-timer.C:
					break
				}
			} else {
				<-timer.C
			}
			if time.Now().After(totalDeadline) {
				break
			}
		}
	}

	// 若超时退出后仍有未停止项或动态未完成任务，从 snapshotLiveTasks 并入 coordinator
	if a.componentEngine != nil {
		liveTasks := a.componentEngine.snapshotLiveTasks()
		existingStuck := a.coordinator.getStuckInstances()
		existingIDs := make(map[string]struct{}, len(existingStuck))
		for _, s := range existingStuck {
			existingIDs[s.InstanceID] = struct{}{}
		}
		for _, task := range liveTasks {
			if _, exists := existingIDs[task.InstanceID]; !exists {
				a.coordinator.recordStuck(task)
				existingIDs[task.InstanceID] = struct{}{}
				for _, reqKey := range task.RetainedDeps {
					pKey, exists := a.graph.providerByKey[reqKey]
					if !exists {
						pKey = reqKey
					}
					retainedProviders[pKey] = struct{}{}
					a.markRetainedTransitiveClosure(pKey, retainedProviders)
				}
			}
		}
	}

	// 记录所有仍未完成停机的 pending static 项
	for _, item := range pendingList {
		if !item.rec.stopped.Load() {
			if _, isRetained := retainedProviders[item.key]; !isRetained {
				a.coordinator.recordStuck(StuckInstance{
					PluginKey:    item.rec.info.Key,
					InstanceID:   item.rec.instanceID,
					Stage:        "timeout",
					RetainedDeps: item.rec.info.Requires,
					Err:          context.DeadlineExceeded,
				})
				retainedProviders[item.key] = struct{}{}
				a.markRetainedTransitiveClosure(item.key, retainedProviders)
			}
		}
	}

	return errors.Join(errs...)
}

func (a *App) safeCallQuiesce(rec *staticInstanceRecord, q Quiescer, totalDeadline, graceDeadline time.Time, cleanupLimit time.Duration) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = &HookPanicError{
				PluginKey:  rec.info.Key,
				InstanceID: rec.instanceID,
				Stage:      "quiesce",
				Value:      r,
				Stack:      debug.Stack(),
			}
		}
	}()

	budget := &budgetMetadata{
		graceDeadline: graceDeadline,
		totalDeadline: totalDeadline,
		cleanupLimit:  cleanupLimit,
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if !totalDeadline.IsZero() {
		ctx, cancel = context.WithDeadline(context.Background(), totalDeadline)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	ctx = context.WithValue(ctx, budgetCtxKey, budget)
	ctx = context.WithValue(ctx, containerCtxKey, a.container)

	return q.Quiesce(ctx)
}
