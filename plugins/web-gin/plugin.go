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

// DefaultWebGinPluginKey 是 Gin Web Worker 插件注册到应用上下文的默认唯一键。
const DefaultWebGinPluginKey ctxkey.Key = "plugin_web_gin"

var (
	// ErrServerAlreadyStarted 表示服务已在启动流程中或已成功启动，禁止重复调用。
	ErrServerAlreadyStarted = errors.New("webgin: 服务已启动")
	// ErrServerClosed 表示服务处于已关闭或正在关闭状态，拒绝接纳新请求。
	ErrServerClosed = errors.New("webgin: 服务已停止")
	// ErrNoLifetimeContext 表示缺少有效的应用生命周期上下文，请求无法被安全调度。
	ErrNoLifetimeContext = errors.New("webgin: 缺少 lifetime context")
	// ErrStopTimeout 表示服务停机等待已超出预设的超时时限。
	ErrStopTimeout = errors.New("webgin: 停机等待超时")
)

// RouteRegistrar 定义路由注册回调函数签名。
//
// 回调在插件启动流程中执行，接收当前启动上下文与私有的 [*gin.Engine] 实例。
// 业务组件可通过此回调向 Gin 引擎注册路由分组、处理函数与局部中间件。
type RouteRegistrar func(context.Context, *gin.Engine)

// Option 定义配置 Gin Web 插件选项的函数类型。
//
// 传入 nil 的选项函数在配置解析阶段将被静默忽略。
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

// WithAddr 设置 HTTP 服务的 TCP 监听地址（例如 "127.0.0.1:8080" 或 "127.0.0.1:0"）。
//
// 若传入空字符串，该配置项将被忽略；若未显式调用本选项，将尝试从环境变量 HOST 与
// PORT 读取监听地址，两者均未配置时默认回退为 "localhost:1118"。
func WithAddr(addr string) Option {
	return func(c *config) {
		if addr != "" {
			c.addr = addr
			c.addrExplicit = true
		}
	}
}

// WithKey 设置插件在应用容器中的唯一标识键 [ctxkey.Key]。
//
// 若传入的键不合法（[ctxkey.Key.IsValid] 返回 false），则保留原有的键设置，
// 默认键为 [DefaultWebGinPluginKey]。
func WithKey(k ctxkey.Key) Option {
	return func(c *config) {
		if k.IsValid() {
			c.key = k
		}
	}
}

// WithName 设置插件的人类可读名称，主要用于日志记录和拓扑诊断。
//
// 若传入空字符串，该配置项将被忽略，默认名称为 "WebGinWorker"。
func WithName(name string) Option {
	return func(c *config) {
		if name != "" {
			c.name = name
		}
	}
}

// WithRequires 设置当前插件在启动前所依赖的其他共享组件键列表。
//
// 框架拓扑排序将依据此依赖关系确定启动次序。多次调用将累加依赖项。
func WithRequires(keys ...ctxkey.Key) Option {
	return func(c *config) {
		c.requires = append(c.requires, keys...)
	}
}

// WithRoute 注册业务路由挂载回调 [RouteRegistrar]。
//
// 注册的回调函数将在插件执行 [Plugin.Start] 时依次回调。若传入 nil，
// 该回调将被忽略。可多次调用以注册多个独立的路由模块。
func WithRoute(fn RouteRegistrar) Option {
	return func(c *config) {
		if fn != nil {
			c.routes = append(c.routes, fn)
		}
	}
}

// WithValidatorRegistrar 注册自定义验证器扩展回调函数。
//
// 回调函数在插件执行 [Plugin.Start] 并创建私有 [*validator.Engine] 实例后、
// 初始化绑定器之前执行。业务可通过此函数注册领域特定的参数校验规则。
// 若回调返回错误，[Plugin.Start] 流程将立即终止并返回该错误。传入 nil 时忽略。
func WithValidatorRegistrar(fn func(*validator.Engine) error) Option {
	return func(c *config) {
		if fn != nil {
			c.validatorRegistrars = append(c.validatorRegistrars, fn)
		}
	}
}

// WithLogger 设置插件专用的日志记录器实例。
//
// 若未显式配置，插件内部日志输出将被禁用，绝不退回或污染全局日志实例。
func WithLogger(logger *logging.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithTimeouts 设置 HTTP 服务的基础超时时间参数。
//
// 各参数含义如下：
//   - read：从连接读取完整请求（含请求体）的最大允许持续时间；
//   - write：向连接写入完整响应的最大允许持续时间；
//   - idle：启用 Keep-Alive 时等待下一个新请求的最大空闲时间；
//   - shutdown：调用停机时等待现有网络连接与服务退出的最长等待时长。
//
// 默认值分别为：读超时 30 秒、写超时 30 秒、空闲超时 60 秒、停机等待 15 秒。
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

// Plugin 实现 [application.RunnablePlugin] 接口，代表由应用框架托管的 Gin Web Worker 插件。
//
// 实例通过构造工厂创建，生命周期由应用框架统一编排，内部状态受互斥锁保护，
// 支持多协程安全并发查询状态。
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

// New 构造 Gin Web 插件的注册元信息 [application.PluginInfo] 与延迟工厂函数。
//
// 构造过程仅完成配置解析与防御性副本复制，不产生任何网络监听、I/O 或资源分配副作用。
// 返回的工厂函数每次调用均生成相互独立的 [*Plugin] 实例。
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

// Addr 返回底层网络监听器绑定的实际地址字符串（例如 "127.0.0.1:8080"）。
//
// 若插件尚未调用 [Plugin.Start] 绑定监听器，则返回配置中指定的地址或经环境变量解析的预设地址。
// 该方法可被多个 goroutine 并发安全调用。
func (p *Plugin) Addr() string {
	p.stateMu.Lock()
	ln := p.listener
	p.stateMu.Unlock()
	if ln != nil {
		return ln.Addr().String()
	}
	return resolveAddr(&p.cfg)
}

// Start 执行插件启动前的资源准备与服务初始化。
//
// 该方法依次执行以下关键动作：
//   - 校验启动状态，防止并发或重复启动，重复启动返回 [ErrServerAlreadyStarted]；
//   - 派生实例独立的强制取消上下文，关联框架强制停止信号；
//   - 创建全新的私有 [*gin.Engine] 实例并开启 ContextWithFallback；
//   - 初始化私有 [*validator.Engine] 并执行用户注册的自定义验证器扩展；
//   - 构造私有 [bind.Binder] 并挂载核心基础中间件（异常恢复、响应兜底、CORS 与请求桥接）；
//   - 调用全部业务注册的路由回调函数挂载路由；
//   - 绑定并启动底层 TCP 监听器（若监听失败，自动释放所有已申请的资源并返回错误）。
//
// 该方法不阻塞等待网络请求，真正的服务监听循环由 [Plugin.Run] 接管。
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

// Run 封存应用生命周期上下文并启动阻塞式的 HTTP 服务事件循环。
//
// 参数 lifetimeCtx 为应用生命周期上下文，包含宿主应用的共享依赖容器与上下文凭据。
// 若尚未成功执行 [Plugin.Start] 即调用此方法，将直接返回错误。
//
// 当底层 HTTP 服务由于 [Plugin.Stop]、[http.Server.Shutdown] 或 [http.Server.Close] 退出时，
// 若退出原因为普通的 [http.ErrServerClosed]，本方法返回 nil 表示正常退出；
// 其余非预期的网络中断或运行时异常将原样返回。
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

// Quiesce 关闭 HTTP 请求准入门禁，进入静默就绪状态。
//
// 关闸后到达的新 HTTP 请求将被内部准入中间件拦截并直接返回 503 Service Unavailable 错误；
// 当前已接纳且正在处理中的存量请求不受影响，继续正常执行。
// 该方法不会主动中断网络连接，也不会调用 [Plugin.Stop]。
func (p *Plugin) Quiesce(ctx context.Context) error {
	p.gate.closeGate()
	return nil
}

// Stop 执行平滑优雅关机流程。
//
// 该方法包含以下协作阶段：
//   - 确保入口门禁关闭，拒绝新请求进入；
//   - 启动 [http.Server.Shutdown] 停止接收新连接并尝试排空空闲连接；
//   - 同时监听停机超时上下文 ctx 与应用层强制取消信号，一旦超时或收到强制信号，
//     立即调用底层传输层的 Close 方法中断网络传输；
//   - 触发内部强制取消分支，通知当前所有执行中的业务 Handler 协同退出；
//   - 阻塞等待准入门禁中所有处于飞行中的请求彻底执行完毕并退出，绝不因超时虚假返回。
//
// 多次重复调用此方法并发安全，且共享同一次执行结果。
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

// BridgeRequest 派生请求上下文，将底层 HTTP 请求上下文与框架应用上下文进行深度桥接。
//
// 该函数在请求上下文上注入以下元数据：
//   - 注入当前请求起始时间戳（以 [ctxkey.UserStartTimeKey] 保存），供下游耗时统计；
//   - 委托 [application.BridgeRequest] 挂载应用级共享容器、追踪键及父实例信任凭据。
//
// 若传入的 reqCtx 为 nil，内部将自动回退为 [context.Background]。
func BridgeRequest(reqCtx context.Context, lifetimeCtx context.Context, id string) context.Context {
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	// 挂载请求开始时间
	reqCtx = context.WithValue(reqCtx, ctxkey.UserStartTimeKey, time.Now())

	// 委托 application.BridgeRequest 注入真正的应用容器与实例凭据
	return application.BridgeRequest(reqCtx, lifetimeCtx, id)
}
