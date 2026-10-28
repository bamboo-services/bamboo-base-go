package webgin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/header"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/middleware"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DefaultWebGinPluginKey 是 Gin Web 插件默认的 Worker 插件 Key。
const DefaultWebGinPluginKey ctxkey.Key = "plugin_web_gin"

var (
	// ErrServerAlreadyStarted 当重复调用 Start 时返回。
	ErrServerAlreadyStarted = errors.New("webgin: 服务已启动")
	// ErrServerClosed 当服务已停止或正在关闭时返回。
	ErrServerClosed = errors.New("webgin: 服务已停止")
	// ErrNoLifetimeContext 当未设置可信 lifetimeCtx 即尝试运行或处理请求时返回。
	ErrNoLifetimeContext = errors.New("webgin: 缺少 lifetime context")
	// ErrStopTimeout 当服务优雅停机超时时返回。
	ErrStopTimeout = errors.New("webgin: 停机等待超时")
)

// RouteRegistrar 定义路由注册回调函数签名，接收 Start 时的上下文与私有 Gin 引擎。
type RouteRegistrar func(context.Context, *gin.Engine)

// Option 为 Gin Web 插件的配置函数。
type Option func(*config)

type config struct {
	addr                string
	addrExplicit        bool
	key                 ctxkey.Key
	name                string
	requires            []ctxkey.Key
	routes              []RouteRegistrar
	validatorRegistrars []func(*validator.Engine) error
	logger              *logging.Logger
	readTimeout         time.Duration
	writeTimeout        time.Duration
	idleTimeout         time.Duration
	shutdownWait        time.Duration
}

// WithAddr 设置监听地址（如 "127.0.0.1:8080" 或 "127.0.0.1:0"）。
func WithAddr(addr string) Option {
	return func(c *config) {
		if addr != "" {
			c.addr = addr
			c.addrExplicit = true
		}
	}
}

// WithKey 设置插件的唯一标识 Key。
func WithKey(k ctxkey.Key) Option {
	return func(c *config) {
		if k.IsValid() {
			c.key = k
		}
	}
}

// WithName 设置插件的人类可读名称。
func WithName(name string) Option {
	return func(c *config) {
		if name != "" {
			c.name = name
		}
	}
}

// WithRequires 设置插件所依赖的键。
func WithRequires(keys ...ctxkey.Key) Option {
	return func(c *config) {
		c.requires = append(c.requires, keys...)
	}
}

// WithRoute 注册路由挂载回调，回调将在 Start 时依次回调。
func WithRoute(fn RouteRegistrar) Option {
	return func(c *config) {
		if fn != nil {
			c.routes = append(c.routes, fn)
		}
	}
}

// WithValidatorRegistrar 注册自定义验证器扩展回调，回调将在 Start 创建私有验证器引擎后、初始化 Binder 之前执行。
func WithValidatorRegistrar(fn func(*validator.Engine) error) Option {
	return func(c *config) {
		if fn != nil {
			c.validatorRegistrars = append(c.validatorRegistrars, fn)
		}
	}
}

// WithLogger 设置显式传入的 Logger 实例，不使用全局 Default。
func WithLogger(logger *logging.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithTimeouts 设置 HTTP 基础超时。
func WithTimeouts(read, write, idle, shutdown time.Duration) Option {
	return func(c *config) {
		c.readTimeout = read
		c.writeTimeout = write
		c.idleTimeout = idle
		c.shutdownWait = shutdown
	}
}

// requestGate 控制运行期请求准入与飞行中请求的统计追踪。
// enter 与 closeGate/waitDrained 保证同一把锁防竞态。
type requestGate struct {
	mu       sync.Mutex
	closed   bool
	inflight int
	cond     *sync.Cond
}

func newRequestGate() *requestGate {
	g := &requestGate{}
	g.cond = sync.NewCond(&g.mu)
	return g
}

func (g *requestGate) enter() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return false
	}
	g.inflight++
	return true
}

func (g *requestGate) leave() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.inflight--
	if g.inflight <= 0 {
		g.cond.Broadcast()
	}
}

func (g *requestGate) closeGate() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.closed = true
}

func (g *requestGate) waitDrained(ctx context.Context) error {
	g.mu.Lock()
	g.closed = true
	if g.inflight == 0 {
		g.mu.Unlock()
		return nil
	}
	g.mu.Unlock()

	done := make(chan struct{})
	go func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		for g.inflight > 0 {
			g.cond.Wait()
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Plugin 实现 application.RunnablePlugin，作为 Gin Web Worker 插件。
type Plugin struct {
	cfg config

	stateMu  sync.Mutex
	starting bool
	started  bool

	engine   *gin.Engine
	server   *http.Server
	listener net.Listener
	binder   *bind.Binder

	gate *requestGate

	forceCtx    context.Context
	cancelForce context.CancelFunc

	lifetimeCtx context.Context
	lifetimeMu  sync.RWMutex

	stopOnce sync.Once
	stopErr  error
	doneChan chan struct{}

	namedLog *logging.LogNamedLogger
}

// New 返回 Worker 的应用注册信息以及 RunnablePlugin 构造工厂（构造期不进行任何 I/O）。
func New(opts ...Option) (application.PluginInfo, func() application.RunnablePlugin) {
	cfg := config{
		key:          DefaultWebGinPluginKey,
		name:         "WebGinWorker",
		readTimeout:  30 * time.Second,
		writeTimeout: 30 * time.Second,
		idleTimeout:  60 * time.Second,
		shutdownWait: 15 * time.Second,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	requiresCopy := make([]ctxkey.Key, len(cfg.requires))
	copy(requiresCopy, cfg.requires)

	routesCopy := make([]RouteRegistrar, len(cfg.routes))
	copy(routesCopy, cfg.routes)

	validatorRegistrarsCopy := make([]func(*validator.Engine) error, len(cfg.validatorRegistrars))
	copy(validatorRegistrarsCopy, cfg.validatorRegistrars)

	cfg.requires = requiresCopy
	cfg.routes = routesCopy
	cfg.validatorRegistrars = validatorRegistrarsCopy

	infoRequiresCopy := make([]ctxkey.Key, len(cfg.requires))
	copy(infoRequiresCopy, cfg.requires)

	info := application.PluginInfo{
		Key:      cfg.key,
		Name:     cfg.name,
		Mode:     application.ModeWorker,
		Requires: infoRequiresCopy,
	}

	factory := func() application.RunnablePlugin {
		p := &Plugin{
			cfg:      cfg,
			gate:     newRequestGate(),
			doneChan: make(chan struct{}),
		}
		if cfg.logger != nil {
			p.namedLog = cfg.logger.WithName("GINW")
		}
		return p
	}

	return info, factory
}

// resolveAddr 解析未显式配置的监听地址。
// 若未显式调用 WithAddr 设置，则尝试从环境变量 HOST / PORT 读取，默认回退为 localhost:1118。
func resolveAddr(c *config) string {
	if c.addrExplicit && c.addr != "" {
		return c.addr
	}
	host := "localhost"
	if h, ok := env.Lookup(env.Host); ok && !h.IsEmpty() {
		host = h.String()
	}
	port := "1118"
	if p, ok := env.Lookup(env.Port); ok && !p.IsEmpty() {
		port = p.String()
	}
	return host + ":" + port
}

// Addr 返回已监听的实际网络地址（未 Start 则返回配置地址或解析默认地址）。
func (p *Plugin) Addr() string {
	p.stateMu.Lock()
	ln := p.listener
	p.stateMu.Unlock()
	if ln != nil {
		return ln.Addr().String()
	}
	return resolveAddr(&p.cfg)
}

// Start 完成 Gin 实例创建（私有新实例、无全局 gin.SetMode 污染）、独立 Validator 与 Binder 初始化、中间件与路由装配及端口监听。
func (p *Plugin) Start(ctx context.Context) error {
	p.stateMu.Lock()
	if p.starting || p.started {
		p.stateMu.Unlock()
		return ErrServerAlreadyStarted
	}
	p.starting = true
	p.stateMu.Unlock()

	// 从 application.ForceContext(ctx) 派生实例私有的强制取消分支
	forceCtx, cancelForce := context.WithCancel(application.ForceContext(ctx))

	var success bool
	defer func() {
		if !success {
			cancelForce()
			p.stateMu.Lock()
			p.starting = false
			p.stateMu.Unlock()
		}
	}()

	// 每实例独立的 gin.New()，不动全局 mode
	engine := gin.New()
	engine.ContextWithFallback = true

	vEngine, err := validator.New()
	if err != nil {
		cancelForce()
		p.stateMu.Lock()
		p.starting = false
		p.stateMu.Unlock()
		return fmt.Errorf("webgin: 初始化 validator 引擎失败: %w", err)
	}

	for _, vr := range p.cfg.validatorRegistrars {
		if vr != nil {
			if err := vr(vEngine); err != nil {
				cancelForce()
				p.stateMu.Lock()
				p.starting = false
				p.stateMu.Unlock()
				return fmt.Errorf("webgin: 注册自定义验证器失败: %w", err)
			}
		}
	}

	b, err := bind.New(vEngine)
	if err != nil {
		cancelForce()
		p.stateMu.Lock()
		p.starting = false
		p.stateMu.Unlock()
		return fmt.Errorf("webgin: 初始化 binder 失败: %w", err)
	}

	// 挂载核心请求处理管道与基础中间件
	engine.Use(
		middleware.Recovery(p.namedLog),
		middleware.Response(),
		middleware.CORS(),
		p.requestBridgeMiddleware(),
	)

	// 调用业务路由注册回调
	for _, r := range p.cfg.routes {
		if r != nil {
			r(ctx, engine)
		}
	}

	listenAddr := resolveAddr(&p.cfg)

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      engine,
		ReadTimeout:  p.cfg.readTimeout,
		WriteTimeout: p.cfg.writeTimeout,
		IdleTimeout:  p.cfg.idleTimeout,
		BaseContext: func(l net.Listener) context.Context {
			return forceCtx
		},
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", listenAddr)
	if err != nil {
		cancelForce()
		p.stateMu.Lock()
		p.starting = false
		p.stateMu.Unlock()
		return fmt.Errorf("webgin: 监听地址 %s 失败: %w", listenAddr, err)
	}

	p.stateMu.Lock()
	p.forceCtx = forceCtx
	p.cancelForce = cancelForce
	p.binder = b
	p.engine = engine
	p.server = server
	p.listener = ln
	p.started = true
	p.starting = false
	p.stateMu.Unlock()
	success = true
	return nil
}

// Run 封存可信运行 lifetimeCtx，并开始阻塞监听服务。
func (p *Plugin) Run(lifetimeCtx context.Context) error {
	p.stateMu.Lock()
	srv := p.server
	ln := p.listener
	p.stateMu.Unlock()

	if srv == nil || ln == nil {
		return errors.New("webgin: 服务未完成 Start 即调用 Run")
	}

	p.lifetimeMu.Lock()
	p.lifetimeCtx = lifetimeCtx
	p.lifetimeMu.Unlock()

	err := srv.Serve(ln)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Quiesce 入口关闸，关闭新请求准入。不取消业务 Handler，不调用 Stop。
func (p *Plugin) Quiesce(ctx context.Context) error {
	p.gate.closeGate()
	return nil
}

// Stop 真实优雅关机。先关闭准入，触发 http.Server.Shutdown；外层超时可中断网络等待，但必须继续等飞行中 Handler 真正退完。
// 监听 App Force 信号或外层 ctx 超时，必要时调用 Server.Close 推进底层传输中断。
// 重复调用 Stop 共享同一次结果。
func (p *Plugin) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		p.stopErr = p.doStop(ctx)
		close(p.doneChan)
	})
	<-p.doneChan
	return p.stopErr
}

func (p *Plugin) doStop(ctx context.Context) error {
	p.stateMu.Lock()
	srv := p.server
	p.stateMu.Unlock()

	if srv == nil {
		return nil
	}

	// 1. Quiesce: 先关闭新请求准入
	p.gate.closeGate()

	// 2. 异步启动 http.Server.Shutdown
	shutdownDone := make(chan error, 1)
	go func() {
		// 使用独立的 Context 还是传入的 ctx？传入 ctx 作为 Shutdown 的等待上限
		shutdownDone <- srv.Shutdown(ctx)
	}()

	var stopReason error

	// 3. 监控超时或取消通知
	select {
	case err := <-shutdownDone:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			stopReason = err
		}
	case <-ctx.Done():
		stopReason = ctx.Err()
		// 超时/取消推进传输关闭
		_ = srv.Close()
		if err := <-shutdownDone; err != nil && !errors.Is(err, http.ErrServerClosed) {
			if stopReason == nil {
				stopReason = err
			} else if !errors.Is(err, stopReason) && !errors.Is(stopReason, err) {
				stopReason = errors.Join(stopReason, err)
			}
		}
	case <-p.forceCtx.Done():
		// App Force 信号通知关闭底层传输，不把正常 AppForce 信号当作 Stop 错误
		_ = srv.Close()
		if err := <-shutdownDone; err != nil && !errors.Is(err, http.ErrServerClosed) {
			stopReason = err
		}
	}

	// 4. 触发实例 force 取消，让所有接纳了 force 信号的业务 Handler 收到取消
	if p.cancelForce != nil {
		p.cancelForce()
	}

	// 5. 核心：必须真正等待所有普通 Handler 退出（active 归零），绝不因 ctx deadline 提前释放或返回假 Done
	drainErr := p.gate.waitDrained(context.Background())
	if drainErr != nil && stopReason == nil {
		stopReason = drainErr
	}

	return stopReason
}

func (p *Plugin) getLifetimeCtx() context.Context {
	p.lifetimeMu.RLock()
	defer p.lifetimeMu.RUnlock()
	return p.lifetimeCtx
}

// requestBridgeMiddleware 拦截请求，维护准入门禁，绑定 Binder，并注入可信 Trace 上下文。
func (p *Plugin) requestBridgeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 同一把锁先判断准入与递增 inflight
		if !p.gate.enter() {
			httpresult.AbortError(c, apperr.New(apperr.ServiceUnavailable, "服务已停止或正在关机", ErrServerClosed))
			return
		}
		defer p.gate.leave()

		// 2. Bind.Inject 先于业务绑定动作注入
		bind.Inject(c, p.binder)

		// 3. 检查 lifetimeCtx，缺失则拒绝请求（严禁 fallback 到背景 context）
		lifetime := p.getLifetimeCtx()
		if lifetime == nil {
			httpresult.AbortError(c, apperr.New(apperr.ServiceUnavailable, "服务未完全就绪，缺少 lifetimeContext", ErrNoLifetimeContext))
			return
		}

		// 4. 请求级上下文构建：生成 requestID 并回写响应头
		reqID := uuid.NewString()
		c.Header(header.HeaderRequestUUID.String(), reqID)

		// 派生请求上下文：接入 forceCtx 取消通知，使原请求 cancel 与 App Force 均能到达 Handler
		reqCtx, cancelReq := context.WithCancel(c.Request.Context())
		defer cancelReq()

		stopAfterFunc := context.AfterFunc(p.forceCtx, cancelReq)
		defer stopAfterFunc()

		bridgedCtx := BridgeRequest(reqCtx, lifetime, reqID)
		c.Request = c.Request.WithContext(bridgedCtx)

		c.Next()
	}
}

// BridgeRequest 派生请求上下文：委托 application.BridgeRequest 挂载内核容器与可信父实例身份，
// 并挂载起始时间 UserStartTimeKey（time.Time 格式供耗时统计）与 RequestKey。
func BridgeRequest(reqCtx context.Context, lifetimeCtx context.Context, id string) context.Context {
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	// 挂载请求开始时间
	reqCtx = context.WithValue(reqCtx, ctxkey.UserStartTimeKey, time.Now())

	// 委托 application.BridgeRequest 注入真正的应用容器与实例凭据
	return application.BridgeRequest(reqCtx, lifetimeCtx, id)
}
