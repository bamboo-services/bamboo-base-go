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

// PluginKey 声明 Async 插件主键。
const PluginKey ctxkey.Key = "plugin_async"

// PoolKey 声明 Async 暴露给应用共享容器的 Pool 句柄键。
const PoolKey ctxkey.Key = "context_async_pool"

var (
	// ErrPoolClosed 当池已关闭或应用处于关机状态时返回。
	ErrPoolClosed = errors.New("async: pool is closed or shutting down")
	// ErrPoolFull 当池已满且为非阻塞模式时返回。
	ErrPoolFull = errors.New("async: pool is full")
	// ErrNilTaskFunc 当提交的任务函数为 nil 时返回。
	ErrNilTaskFunc = errors.New("async: task function cannot be nil")
	// ErrNotInitialized 当插件尚未初始化或启动时返回。
	ErrNotInitialized = errors.New("async: plugin not initialized")
	// ErrInvalidPoolCapacity 当容量小于等于 0 时返回。
	ErrInvalidPoolCapacity = errors.New("async: pool capacity must be positive")
	// ErrInvalidContext 当提交任务的上下文不满足准入条件（未就绪、跨容器或无授权）时返回。
	ErrInvalidContext = errors.New("async: invalid parent context for pool")
)

// Pool 定义对外暴露的异步池服务接口。
type Pool interface {
	// Submit 提交一个异步任务，从 parentCtx 剥离请求级超时，保留容器与追踪信息。
	Submit(parentCtx context.Context, fn func(ctx context.Context), opts ...TaskOption) (*Task, error)
	// Running 返回当前正在运行的任务数。
	Running() int
	// Cap 返回池容量。
	Cap() int
	// Free 返回池可用容量。
	Free() int
}

// Option 为 Async 插件配置项。
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

// WithKey 设置插件 Key。
func WithKey(key ctxkey.Key) Option {
	return func(c *config) {
		if key.IsValid() {
			c.key = key
		}
	}
}

// WithName 设置插件名称。
func WithName(name string) Option {
	return func(c *config) {
		if name != "" {
			c.name = name
		}
	}
}

// WithRequires 声明本池内任务使用的资源（DB/Cache/Email 等），构建 PluginInfo.Requires。
func WithRequires(keys ...ctxkey.Key) Option {
	return func(c *config) {
		for _, k := range keys {
			if k.IsValid() {
				c.requires = append(c.requires, k)
			}
		}
	}
}

// WithCapacity 设置协程池容量（最大并发并发数，必须大于 0）。
func WithCapacity(cap int) Option {
	return func(c *config) {
		if cap > 0 {
			c.capacity = cap
		}
	}
}

// WithExpiryDuration 设置 worker 清理空闲间隔。
func WithExpiryDuration(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.expiry = d
		}
	}
}

// WithPreAlloc 设置是否预先分配 worker 队列内存。
func WithPreAlloc(preAlloc bool) Option {
	return func(c *config) {
		c.preAlloc = preAlloc
	}
}

// WithLogger 注入应用统一日志器，保持 App 隔离。
func WithLogger(logger *logging.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// Plugin 实现 application.Plugin 与 ctxkey.Exposer 契约（Tool 模式）。
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

// New 构造 Async Tool 定义与工厂，主键为 PluginKey，Provides 包含 PoolKey。
// 构造过程为纯构造，不创建后台协程池或分配 worker 内存。
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

// Start 在生命周期启动阶段创建私有 ants.Pool。
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

// Expose 交付 PluginInfo.Provides 中声明的 PoolKey 底层句柄。
func (p *Plugin) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		PoolKey: p,
	}
}

// Stop 关准入，等待在途任务真实归零，再释放 ants 协程池。
// 多次重复调用 Stop 共享相同结果与等待同一完成。
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

// Running 返回当前正在运行的任务数。
func (p *Plugin) Running() int {
	return int(atomic.LoadInt64(&p.active))
}

// Cap 返回池容量。
func (p *Plugin) Cap() int {
	p.poolMu.RLock()
	defer p.poolMu.RUnlock()
	if p.pool == nil {
		return p.cfg.capacity
	}
	return p.pool.Cap()
}

// Free 返回可用容量。
func (p *Plugin) Free() int {
	p.poolMu.RLock()
	defer p.poolMu.RUnlock()
	if p.pool == nil {
		return 0
	}
	return p.pool.Free()
}

// Submit 核心提交实现：准入检查、任务计数登记、Detach 上下文、接入 Force 取消，并在 ants 中执行。
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

// Submit 从 parentCtx 的应用共享容器中获取 Pool 实例并提交任务。
// 若 parentCtx 未挂载容器或池不存在，返回相应错误。
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
