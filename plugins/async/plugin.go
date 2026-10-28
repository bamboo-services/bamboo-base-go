package async

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/panjf2000/ants/v2"
)

// PluginKey 声明 Async 插件在应用拓扑图中的主键。
const PluginKey ctxkey.Key = "plugin_async"

// PoolKey 声明 Async 暴露给应用共享容器的 [Pool] 句柄键。
const PoolKey ctxkey.Key = "context_async_pool"

var (
	// ErrPoolClosed 表示协程池已关闭、正在执行停机流程或应用未处于运行状态。
	// 调用方可通过 [errors.Is] 识别该错误以实施优雅拒绝或降级。
	ErrPoolClosed = errors.New("async: pool is closed or shutting down")

	// ErrPoolFull 表示协程池并发容量已达上限，非阻塞提交被立即拒绝。
	// 调用方可通过 [errors.Is] 捕获该错误以实施削峰重试或业务降级。
	ErrPoolFull = errors.New("async: pool is full")

	// ErrNilTaskFunc 表示向协程池提交的任务执行函数为 nil。
	ErrNilTaskFunc = errors.New("async: task function cannot be nil")

	// ErrNotInitialized 表示插件尚未完成初始化启动，底层协程池未就绪。
	ErrNotInitialized = errors.New("async: plugin not initialized")

	// ErrInvalidPoolCapacity 表示协程池配置容量非法，容量必须大于 0。
	ErrInvalidPoolCapacity = errors.New("async: pool capacity must be positive")

	// ErrInvalidContext 表示提交任务的上下文不满足准入条件（未就绪、跨容器或无授权）。
	ErrInvalidContext = errors.New("async: invalid parent context for pool")
)

// Pool 定义对外暴露的异步任务协程池接口，支持并发安全地提交与观测任务。
// 由插件在启动时以 [PoolKey] 注册到应用共享容器中。
type Pool interface {
	// Submit 提交一个异步任务，从 parentCtx 剥离请求级超时，保留容器与追踪信息。
	// 当应用处于运行状态且并发槽位充足时返回 [Task] 句柄；若提交失败返回相应错误。
	// 本方法并发安全，可被多个 goroutine 同时调用。
	Submit(parentCtx context.Context, fn func(ctx context.Context), opts ...TaskOption) (*Task, error)

	// Running 返回当前池中正在执行的活跃任务数量。
	Running() int

	// Cap 返回协程池的最大并发容量。
	Cap() int

	// Free 返回协程池当前可用的并发槽位数量。
	Free() int
}

// Option 为 Async 插件的全局函数式配置选项。
// 传入 nil 时将被 [New] 安全忽略。
type Option func(*config)

type config struct {
	key      ctxkey.Key
	name     string
	requires []ctxkey.Key
	capacity int
	expiry   time.Duration
	preAlloc bool
	logger   *logging.Logger
}

func defaultConfig() config {
	return config{
		key:      PluginKey,
		name:     "plugin_async",
		capacity: 1000,
		expiry:   time.Second * 10,
	}
}

// WithKey 设置插件在应用拓扑中的组件 Key。
// 若 key 非法，则忽略并保持默认值 [PluginKey]。
func WithKey(key ctxkey.Key) Option {
	return func(c *config) {
		if key.IsValid() {
			c.key = key
		}
	}
}

// WithName 设置插件在应用生命周期中的人类可读名称。
// 若 name 为空字符串，则忽略并保持默认名称 "plugin_async"。
func WithName(name string) Option {
	return func(c *config) {
		if name != "" {
			c.name = name
		}
	}
}

// WithRequires 声明本池内任务使用的下游共享资源键（如 DB、Cache、Email 等），构建 PluginInfo.Requires。
// 宿主应用将依据依赖图确保这些组件在协程池关机完全结束前保持可用。
func WithRequires(keys ...ctxkey.Key) Option {
	return func(c *config) {
		for _, k := range keys {
			if k.IsValid() {
				c.requires = append(c.requires, k)
			}
		}
	}
}

// WithCapacity 设置协程池最大并发运行容量（槽位数，必须大于 0）。
// 若 cap 小于等于 0，则忽略并保持默认容量 1000。
func WithCapacity(cap int) Option {
	return func(c *config) {
		if cap > 0 {
			c.capacity = cap
		}
	}
}

// WithExpiryDuration 设置 worker 协程空闲回收间隔。
// 若 d 小于等于 0，则忽略并保持默认间隔 10 秒。
func WithExpiryDuration(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.expiry = d
		}
	}
}

// WithPreAlloc 设置是否在协程池启动时预先分配 worker 队列内存。
func WithPreAlloc(preAlloc bool) Option {
	return func(c *config) {
		c.preAlloc = preAlloc
	}
}

// WithLogger 注入应用统一结构化日志器，保持 App 隔离。
// 为 nil 时插件仅在控制台或标准输出产生最小诊断，不记录任务级调试与错误日志。
func WithLogger(logger *logging.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// Plugin 实现 [application.Plugin] 与 [ctxkey.Exposer] 契约（Tool 模式）。
// 由宿主生命周期机制调度，不直接由业务业务方构造。
type Plugin struct {
	cfg config

	pool     *ants.Pool
	poolMu   sync.RWMutex
	namedLog *logging.LogNamedLogger

	// 任务计数与准入控制
	admitMu  sync.RWMutex
	closed   bool
	active   int64
	zeroCond *sync.Cond
	zeroMu   sync.Mutex
	stopOnce sync.Once
	stopDone chan struct{}
	stopErr  error
}

// New 构造 Async Tool 插件定义与工厂，主键为 [PluginKey]，Provides 包含 [PoolKey]。
// 构造过程为纯配置构造，不创建后台协程池或预分配 worker 内存。
func New(opts ...Option) (application.PluginInfo, func() application.Plugin) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	info := application.PluginInfo{
		Key:      cfg.key,
		Name:     cfg.name,
		Mode:     application.ModeTool,
		Provides: []ctxkey.Key{PoolKey},
		Requires: cfg.requires,
	}

	factory := func() application.Plugin {
		p := &Plugin{
			cfg:      cfg,
			stopDone: make(chan struct{}),
		}
		p.zeroCond = sync.NewCond(&p.zeroMu)
		if cfg.logger != nil {
			p.namedLog = cfg.logger.WithName(logging.NamedTASK)
		}
		return p
	}

	return info, factory
}

// Start 在生命周期启动阶段创建私有底层的并发协程池。
// 若已启动则幂等返回 nil；若容量参数非法返回 [ErrInvalidPoolCapacity]。
func (p *Plugin) Start(_ context.Context) error {
	p.poolMu.Lock()
	defer p.poolMu.Unlock()

	if p.pool != nil {
		return nil
	}

	if p.cfg.capacity <= 0 {
		return ErrInvalidPoolCapacity
	}

	antsOpts := []ants.Option{
		ants.WithNonblocking(true), // 有界非阻塞并发
	}
	if p.cfg.expiry > 0 {
		antsOpts = append(antsOpts, ants.WithExpiryDuration(p.cfg.expiry))
	}
	if p.cfg.preAlloc {
		antsOpts = append(antsOpts, ants.WithPreAlloc(p.cfg.preAlloc))
	}

	pool, err := ants.NewPool(p.cfg.capacity, antsOpts...)
	if err != nil {
		return fmt.Errorf("async: 创建协程池失败: %w", err)
	}

	p.pool = pool
	return nil
}

// Expose 交付 [application.PluginInfo.Provides] 中声明的 [PoolKey] 底层接口句柄。
func (p *Plugin) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		PoolKey: p,
	}
}

// Stop 关闭准入门禁，阻塞等待在途活跃任务真实归零后释放底层协程池。
// 即使传入的 ctx 超时，本方法也绝不虚假提前返回；多次重复调用共享相同退出结果与等待同一完成。
func (p *Plugin) Stop(ctx context.Context) error {
	if p == nil {
		return nil
	}

	p.stopOnce.Do(func() {
		p.stopErr = p.doStop(ctx)
		close(p.stopDone)
	})

	<-p.stopDone
	return p.stopErr
}

func (p *Plugin) doStop(_ context.Context) error {
	// 1. 关闭准入，后续所有 Submit 立即失败
	p.admitMu.Lock()
	p.closed = true
	p.admitMu.Unlock()

	// 2. 等待在途活跃任务计数真实归零（Cond.Wait 释放并重新获得锁）
	// 注意：不以调用方上下文的超时虚构关闭成功，必须等待真实归零
	p.zeroMu.Lock()
	for atomic.LoadInt64(&p.active) > 0 {
		p.zeroCond.Wait()
	}
	p.zeroMu.Unlock()

	// 3. 真正归零后，清理并释放 ants.Pool
	p.poolMu.Lock()
	defer p.poolMu.Unlock()
	if p.pool != nil {
		if err := p.pool.ReleaseContext(context.Background()); err != nil {
			return err
		}
		p.pool = nil
	}

	return nil
}

// Running 返回当前池中正在执行的活跃任务数量。
// 本方法并发安全。
func (p *Plugin) Running() int {
	return int(atomic.LoadInt64(&p.active))
}

// Cap 返回协程池的最大并发容量。
// 若协程池尚未完成 Start 初始化，返回配置容量。本方法并发安全。
func (p *Plugin) Cap() int {
	p.poolMu.RLock()
	defer p.poolMu.RUnlock()
	if p.pool == nil {
		return p.cfg.capacity
	}
	return p.pool.Cap()
}

// Free 返回协程池当前可用的并发槽位数量。
// 若协程池尚未完成 Start 初始化，返回 0。本方法并发安全。
func (p *Plugin) Free() int {
	p.poolMu.RLock()
	defer p.poolMu.RUnlock()
	if p.pool == nil {
		return 0
	}
	return p.pool.Free()
}

// Submit 提交异步任务，执行运行态准入检查、任务计数登记、Detach 上下文、接入 Force 取消，并在协程池中执行。
// 若任务函数 fn 为 nil，返回 [ErrNilTaskFunc]；若应用处于关闭状态返回 [ErrPoolClosed]；
// 若池满返回 [ErrPoolFull]；若上下文跨容器或未就绪返回 [ErrInvalidContext]。
// 本方法并发安全。
func (p *Plugin) Submit(parentCtx context.Context, fn func(ctx context.Context), opts ...TaskOption) (*Task, error) {
	if fn == nil {
		return nil, ErrNilTaskFunc
	}

	// 1. 准入锁内校验 App 状态与所属容器契约
	p.admitMu.RLock()
	if p.closed {
		p.admitMu.RUnlock()
		return nil, ErrPoolClosed
	}

	st, ok := application.StateFrom(parentCtx)
	if !ok || st != application.StateRunning {
		p.admitMu.RUnlock()
		return nil, ErrInvalidContext
	}

	callerPool, err := application.Get[Pool](parentCtx, PoolKey)
	if err != nil || callerPool != p {
		p.admitMu.RUnlock()
		return nil, ErrInvalidContext
	}

	// 登记活跃任务数
	atomic.AddInt64(&p.active, 1)
	p.admitMu.RUnlock()

	taskCfg := defaultTaskConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&taskCfg)
		}
	}

	// 2. 从 parentCtx 剥离取消与临时数据，接入全局 Force 取消通知
	detachedCtx, cancel := application.Detach(parentCtx)
	task := newTask(detachedCtx, cancel, taskCfg.name)

	// 监听全局 Force 取消
	forceCtx := application.ForceContext(parentCtx)
	var stopForceWatch func() bool
	if forceCtx != nil {
		stopForceWatch = context.AfterFunc(forceCtx, func() {
			task.Cancel()
		})
	}

	// 3. 提交到 ants 协程池
	p.poolMu.RLock()
	pool := p.pool
	p.poolMu.RUnlock()

	if pool == nil {
		// 池未就绪，回滚任务计数与取消挂钩
		if stopForceWatch != nil {
			stopForceWatch()
		}
		cancel()
		p.releaseTask()
		return nil, ErrNotInitialized
	}

	submitErr := pool.Submit(func() {
		defer func() {
			if stopForceWatch != nil {
				stopForceWatch()
			}
			p.releaseTask()
		}()

		task.execute(fn, p.namedLog, taskCfg.debug)
	})

	if submitErr != nil {
		// ants 提交失败（如池满 ErrPoolOverload），精准回滚登记计数与取消挂钩，不遗留 Task 占用
		if stopForceWatch != nil {
			stopForceWatch()
		}
		cancel()
		p.releaseTask()

		if errors.Is(submitErr, ants.ErrPoolOverload) {
			return nil, ErrPoolFull
		}
		if errors.Is(submitErr, ants.ErrPoolClosed) {
			return nil, ErrPoolClosed
		}
		return nil, submitErr
	}

	return task, nil
}

func (p *Plugin) releaseTask() {
	if atomic.AddInt64(&p.active, -1) == 0 {
		p.zeroMu.Lock()
		p.zeroCond.Broadcast()
		p.zeroMu.Unlock()
	}
}

// Submit 从 parentCtx 的应用共享容器中获取 [Pool] 实例并提交异步任务。
// 若 parentCtx 未挂载容器或池已停用，返回对应错误（包括 [ErrPoolClosed]、[ErrNotInitialized] 等）。
// 提交成功后返回关联的 [Task] 句柄。本函数并发安全。
func Submit(parentCtx context.Context, fn func(ctx context.Context), opts ...TaskOption) (*Task, error) {
	pool, err := application.Get[Pool](parentCtx, PoolKey)
	if err != nil {
		if errors.Is(err, application.ErrComponentStopped) || errors.Is(err, application.ErrComponentUnavailable) {
			return nil, errors.Join(ErrPoolClosed, err)
		}
		return nil, err
	}
	if pool == nil {
		return nil, ErrNotInitialized
	}
	return pool.Submit(parentCtx, fn, opts...)
}
