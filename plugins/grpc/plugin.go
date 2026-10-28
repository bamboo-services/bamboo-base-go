package xGrpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// DefaultGRPCPluginKey 是 gRPC 插件默认的 Worker 插件标识键。
// 其默认值为 "plugin_grpc"，可在应用容器与依赖分析图中作为唯一键引用。
const DefaultGRPCPluginKey ctxkey.Key = "plugin_grpc"

// gRPC 插件生命周期与状态控制导出的错误变量。
var (
	// ErrServerAlreadyStarted 表示 gRPC 服务已经启动，禁止重复调用 [Plugin.Start]。
	ErrServerAlreadyStarted = errors.New("grpc: 服务已启动")

	// ErrServerClosed 表示 gRPC 服务已经停止或正在关机过程中，拒绝接收入站新请求。
	ErrServerClosed = errors.New("grpc: 服务已停止")

	// ErrNoLifetimeContext 表示缺少可信的生命周期上下文，无法完成服务启动与运行。
	ErrNoLifetimeContext = errors.New("grpc: 缺少 lifetime context")

	// ErrStopTimeout 表示 gRPC 服务在执行优雅停机排空等待时超过了预设的超时预算。
	ErrStopTimeout = errors.New("grpc: 停机等待超时")
)

// ServiceRegistrar 是业务向 [grpc.Server] 挂载 gRPC Service 定义的回调函数。
//
// 该函数在 [Plugin.Start] 阶段被执行，入参 ctx 为启动阶段的上下文，
// s 为已初始化拦截器链的原生 [grpc.Server] 实例。业务应当在此回调中调用
// protobuf 编译器生成的 RegisterXxxServer 方法。回调中不应启动长驻循环或阻塞操作。
type ServiceRegistrar func(ctx context.Context, s *grpc.Server)

// Option 为 gRPC 插件的函数式配置项。
//
// 传入 [New] 用于在构造期定制监听端口、依赖项、拦截器与日志器。
// nil 选项函数在执行时会被安全忽略。
type Option func(*config)

type config struct {
	addr               string
	key                ctxkey.Key
	name               string
	requires           []ctxkey.Key
	registrars         []ServiceRegistrar
	logger             *logging.Logger
	serverOptions      []grpc.ServerOption
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
	middlewareRegistry *xGrpcMiddle.Registry
	reflectionEnabled  bool
	reflectionSet      bool
}

// WithAddr 设置 gRPC 服务的网络监听地址（例如 ":9090" 或 "127.0.0.1:0"）。
//
// 显式传入非空字符串时优先采用；未显式指定或传入空字符串时，
// 默认从环境变量 GRPC_PORT 读取端口数值（缺省或非法端口范围时回退为 1119），
// 并最终拼接为 ":<port>" 格式的监听地址。
func WithAddr(addr string) Option {
	return func(c *config) {
		if addr != "" {
			c.addr = addr
		}
	}
}

// WithKey 设置 gRPC 插件在应用容器中的唯一标识 [ctxkey.Key]。
//
// 仅当传入的 k 通过 [ctxkey.Key.IsValid] 校验时生效；若 k 非法或未调用，
// 插件保持默认的 [DefaultGRPCPluginKey]。
func WithKey(k ctxkey.Key) Option {
	return func(c *config) {
		if k.IsValid() {
			c.key = k
		}
	}
}

// WithName 设置 gRPC 插件的人类可读名称。
//
// 仅当传入非空字符串时生效；若未调用或传入空值，默认为 "GRPCWorker"。
func WithName(name string) Option {
	return func(c *config) {
		if name != "" {
			c.name = name
		}
	}
}

// WithRequires 追加 gRPC 插件所依赖的组件上下文键列表。
//
// 声明的键将被注册到 [application.PluginInfo.Requires] 中，
// 供应用容器在拓扑排序阶段执行依赖就绪检查。
func WithRequires(keys ...ctxkey.Key) Option {
	return func(c *config) {
		c.requires = append(c.requires, keys...)
	}
}

// WithService 注册 gRPC 服务挂载回调。
//
// 业务通过传入 [ServiceRegistrar] 函数将生成的 gRPC 服务绑定到内部 Server 上。
// 传入 nil 时将被安全忽略；支持多次调用以追加多个服务挂载函数。
func WithService(fn ServiceRegistrar) Option {
	return func(c *config) {
		if fn != nil {
			c.registrars = append(c.registrars, fn)
		}
	}
}

// WithLogger 设置显式传入的日志记录器实例。
//
// 插件将基于该日志器派生名为 "GRPC" 的子日志器，用于输出未预期 Panic
// 或内部系统错误的结构化日志。若未注入或传入 nil，插件在异常时静默不记录日志。
func WithLogger(logger *logging.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithServerOptions 追加原生 [grpc.ServerOption] 选项。
//
// 追加的选项将在创建 [grpc.Server] 时生效，排在框架内置拦截器链选项之后。
func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(c *config) {
		c.serverOptions = append(c.serverOptions, opts...)
	}
}

// WithUnaryInterceptors 追加自定义一元拦截器。
//
// 追加的拦截器将被编排在框架入口桥接拦截器（负责准入、追踪、桥接与 Panic 恢复）之后，
// 依次形成洋葱模型调用链。
func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(c *config) {
		c.unaryInterceptors = append(c.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors 追加自定义流式拦截器。
//
// 追加的拦截器将被编排在框架流式入口桥接拦截器之后，依次形成洋葱模型调用链。
func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(c *config) {
		c.streamInterceptors = append(c.streamInterceptors, interceptors...)
	}
}

// WithMiddlewareRegistry 设置实例独享的服务级中间件注册表。
//
// 传入的 [xGrpcMiddle.Registry] 会在构造工厂中执行深拷贝快照，
// 确保每个插件实例拥有独立的中间件存储，避免并发修改冲突。若传入 nil，
// 插件将自动创建一份空的注册表。
func WithMiddlewareRegistry(reg *xGrpcMiddle.Registry) Option {
	return func(c *config) {
		c.middlewareRegistry = reg
	}
}

// WithReflection 配置是否启用 gRPC 反射服务。
//
// 显式传入 true 时在服务启动后通过 [reflection.Register] 注册反射服务；
// 若未显式调用本选项，插件在启动时将尝试从环境变量 GRPC_REFLECTION 解析布尔值配置，
// 若环境变量未配置或解析失败则默认关闭反射服务。
func WithReflection(enabled bool) Option {
	return func(c *config) {
		c.reflectionEnabled = enabled
		c.reflectionSet = true
	}
}

// requestGate 控制运行期请求准入与处理中调用的统计追踪。
// 同一锁内判定准入并累加计数，严防并发 Add/Wait 竞态。
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
	if g.inflight <= 0 {
		g.cond.Broadcast()
	}
}

func (g *requestGate) waitDrained(ctx context.Context) error {
	g.mu.Lock()
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

// wrappedServerStream 封装 grpc.ServerStream，重写 Context() 返回桥接上下文。
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// Plugin 实现 [application.RunnablePlugin] 接口，作为长驻的 gRPC Worker 插件。
//
// 负责维护 gRPC 服务端的生命周期管理、端口监听、请求准入控制以及优雅关机。
// 实例由 [New] 导出的工厂函数创建，其生命周期受应用状态机编排控制。
type Plugin struct {
	cfg config

	server   *grpc.Server
	listener net.Listener

	gate        *requestGate
	forceCtx    context.Context
	cancelForce context.CancelFunc

	lifetimeCtx context.Context
	lifetimeMu  sync.RWMutex

	stopOnce sync.Once
	stopErr  error
	doneChan chan struct{}

	namedLog *logging.LogNamedLogger

	// 私有中间件注册表快照
	mdRegistry *xGrpcMiddle.Registry
}

// resolveGRPCAddr 解析 gRPC 监听地址。
// 若显式配置了非空地址则原样返回；否则读取环境变量 env.GrpcPort，
// 默认端口为 1119，若无法解析或非有效端口范围 (<=0 或 >65535) 则回退为 1119，最终返回 ":<port>"。
func resolveGRPCAddr(configured string) string {
	if configured != "" {
		return configured
	}
	port := 1119
	if val, ok := env.Lookup(env.GrpcPort); ok {
		if parsedPort, err := val.Int(); err == nil && parsedPort >= 1 && parsedPort <= 65535 {
			port = parsedPort
		}
	}
	return ":" + strconv.Itoa(port)
}

// New 创建 gRPC 插件并返回其注册信息 [application.PluginInfo] 与构造工厂函数。
//
// 函数在构造期仅完成配置项合并、校验与切片深拷贝快照，不执行任何端口绑定、
// 网络监听或后台任务。返回的工厂函数每次调用都会创建全新的 [Plugin] 实例，
// 确保中间件注册表及准入控制状态完全隔离。
func New(opts ...Option) (application.PluginInfo, func() application.RunnablePlugin) {
	cfg := config{
		addr: "",
		key:  DefaultGRPCPluginKey,
		name: "GRPCWorker",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	// 选项切片深拷贝快照，防止外部修改复用
	cfgSnapshot := config{
		addr:               cfg.addr,
		key:                cfg.key,
		name:               cfg.name,
		requires:           append([]ctxkey.Key(nil), cfg.requires...),
		registrars:         append([]ServiceRegistrar(nil), cfg.registrars...),
		logger:             cfg.logger,
		serverOptions:      append([]grpc.ServerOption(nil), cfg.serverOptions...),
		unaryInterceptors:  append([]grpc.UnaryServerInterceptor(nil), cfg.unaryInterceptors...),
		streamInterceptors: append([]grpc.StreamServerInterceptor(nil), cfg.streamInterceptors...),
		reflectionEnabled:  cfg.reflectionEnabled,
		reflectionSet:      cfg.reflectionSet,
	}
	if cfg.middlewareRegistry != nil {
		cfgSnapshot.middlewareRegistry = cfg.middlewareRegistry.Clone()
	} else {
		cfgSnapshot.middlewareRegistry = xGrpcMiddle.NewRegistry()
	}

	info := application.PluginInfo{
		Key:      cfgSnapshot.key,
		Name:     cfgSnapshot.name,
		Mode:     application.ModeWorker,
		Requires: cfgSnapshot.requires,
	}

	factory := func() application.RunnablePlugin {
		p := &Plugin{
			cfg:      cfgSnapshot,
			gate:     newRequestGate(),
			doneChan: make(chan struct{}),
		}
		if cfgSnapshot.middlewareRegistry != nil {
			p.mdRegistry = cfgSnapshot.middlewareRegistry.Clone()
		} else {
			p.mdRegistry = xGrpcMiddle.NewRegistry()
		}
		if cfgSnapshot.logger != nil {
			p.namedLog = cfgSnapshot.logger.WithName("GRPC")
		}
		return p
	}

	return info, factory
}

// Addr 返回已分配或配置的监听地址字符串。
//
// 若服务已通过 [Plugin.Start] 成功绑定端口，返回底层 socket 的实际地址（如 "127.0.0.1:9090"）；
// 若服务尚未启动，返回经过默认规则解析后的监听地址配置（例如 ":1119"）。
// 该方法可并发安全调用。
func (p *Plugin) Addr() string {
	if p.listener != nil {
		return p.listener.Addr().String()
	}
	return resolveGRPCAddr(p.cfg.addr)
}

// Start 完成端口监听分配、Server 初始化、私有拦截器装配与业务 Service 挂载。
//
// 启动逻辑依循以下顺序执行：
//  1. 状态检查：若底层 listener 已分配，说明服务重复启动，返回 [ErrServerAlreadyStarted]；
//  2. 强制上下文派生：从传入的 ctx 中提取应用级 Force 取消分支，用于超时强杀联动；
//  3. 端口监听：调用底层网络库在目标地址启动 TCP 监听；
//  4. 拦截器组装：依次将框架内置入口桥接拦截器与自定义拦截器拼接成链；
//  5. Server 初始化：创建原生 [grpc.Server] 并依次执行已注册的 [ServiceRegistrar]；
//  6. 反射注册：若配置或环境变量开启反射，则通过 [reflection.Register] 挂载反射服务。
//
// 若启动过程中遇到端口占用或网络监听失败，将立即触发内部清理并返回带有地址说明的包装错误。
func (p *Plugin) Start(ctx context.Context) error {
	if p.listener != nil {
		return ErrServerAlreadyStarted
	}

	// 派生实例私有的强制取消分支
	p.forceCtx, p.cancelForce = context.WithCancel(application.ForceContext(ctx))

	targetAddr := resolveGRPCAddr(p.cfg.addr)

	// 监听端口
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", targetAddr)
	if err != nil {
		if p.cancelForce != nil {
			p.cancelForce()
		}
		return fmt.Errorf("grpc: 监听地址 %s 失败: %w", targetAddr, err)
	}
	p.listener = ln

	// 构建实例私有的一元与流式拦截器链
	unaryChain := []grpc.UnaryServerInterceptor{
		p.unaryBridgeInterceptor(),
	}
	unaryChain = append(unaryChain, p.cfg.unaryInterceptors...)

	streamChain := []grpc.StreamServerInterceptor{
		p.streamBridgeInterceptor(),
	}
	streamChain = append(streamChain, p.cfg.streamInterceptors...)

	serverOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryChain...),
		grpc.ChainStreamInterceptor(streamChain...),
	}
	serverOpts = append(serverOpts, p.cfg.serverOptions...)

	p.server = grpc.NewServer(serverOpts...)

	// 注册业务 Service
	for _, reg := range p.cfg.registrars {
		if reg != nil {
			reg(ctx, p.server)
		}
	}

	enabled := p.cfg.reflectionEnabled
	if !p.cfg.reflectionSet {
		if val, ok := env.Lookup(env.GrpcReflection); ok {
			if b, err := val.Bool(); err == nil {
				enabled = b
			} else {
				enabled = false
			}
		} else {
			enabled = false
		}
	}
	if enabled {
		reflection.Register(p.server)
	}

	return nil
}

// Run 封存应用生命周期上下文并阻塞运行 gRPC 服务监听。
//
// 调用方传入由应用容器注入的可信 lifetimeCtx。若在未成功执行 [Plugin.Start]
// 之前直接调用本方法，将返回明确的编程未就绪错误。
//
// 该方法会持续阻塞在底层 [grpc.Server.Serve] 上，直到服务被正常关闭或外部强制终止。
// 若服务因 [grpc.ErrServerStopped] 退出，本方法正常返回 nil，不视作异常。
func (p *Plugin) Run(lifetimeCtx context.Context) error {
	if p.server == nil || p.listener == nil {
		return errors.New("grpc: 服务未完成 Start 即调用 Run")
	}

	p.lifetimeMu.Lock()
	p.lifetimeCtx = lifetimeCtx
	p.lifetimeMu.Unlock()

	err := p.server.Serve(p.listener)
	if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}

// Quiesce 执行入站关闸操作。
//
// 该方法通过内部准入闸门关闭新请求准入（随后到达的 RPC 请求将立即被桥接拦截器拒绝并返回
// 包含 [apperr.ServiceUnavailable] 的状态错误），但不会调用 Stop，不会打断正在处理中的请求，
// 也不分配新的停机超时预算。此方法可并发幂等调用。
func (p *Plugin) Quiesce(ctx context.Context) error {
	p.gate.closeGate()
	return nil
}

// Stop 执行真实的 gRPC 优雅关机流程。
//
// 该方法具备并发幂等性（内部由 [sync.Once] 保护）：
//  1. 关闭新请求准入闸门；
//  2. 在独立的私有 goroutine 中触发原生 [grpc.Server.GracefulStop]；
//  3. 同时监控传入 ctx 的超时/取消信号与应用的强制停止信号（forceCtx）：若超时或收到 Force，
//     立即调用 [grpc.Server.Stop] 强制中断底层网络连接；
//  4. 核心保障：无超时等待飞行中的所有 Handler 与 Stream 完全排空（waitDrained），
//     绝不因外部 ctx 截止而假返回或提前释放底层资源；
//  5. 显式关闭 listener 并返回汇总的关闭错误。
func (p *Plugin) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		p.stopErr = p.doStop(ctx)
		close(p.doneChan)
	})
	<-p.doneChan
	return p.stopErr
}

func (p *Plugin) doStop(ctx context.Context) (stopReason error) {
	defer func() {
		if p.cancelForce != nil {
			p.cancelForce()
		}
	}()

	if p.server == nil {
		if p.listener != nil {
			if err := p.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				stopReason = errors.Join(stopReason, err)
			}
		}
		return stopReason
	}

	// 1. 关闭新请求准入
	p.gate.closeGate()

	// 2. 在独立私有 goroutine 中触发 GracefulStop
	gracefulDone := make(chan struct{})
	go func() {
		p.server.GracefulStop()
		close(gracefulDone)
	}()

	// 3. 监控超时或取消通知
	select {
	case <-gracefulDone:
		// 正常排空结束
	case <-ctx.Done():
		stopReason = ctx.Err()
		p.server.Stop()
	case <-p.forceCtx.Done():
		// App 正常 Force 信号不作为 stopReason 失败错误
		p.server.Stop()
	}

	// 4. 触发强制取消，确保接入此 forceCtx 的底层调用收到取消
	if p.cancelForce != nil {
		p.cancelForce()
	}

	// 等待 GracefulStop 流程真正退出（若被 Stop 打断，GracefulStop 会随之返回）
	<-gracefulDone

	// 5. 关键：真实等待所有飞行中 Handler/Stream 退出，不因 ctx 截止假返回释放
	// 使用无超时的等待，确保 Handler 完全结束才释放底层资源
	drainErr := p.gate.waitDrained(context.Background())
	if drainErr != nil && stopReason == nil {
		stopReason = drainErr
	}

	// 6. 显式关闭 listener（未 Serve 的 listener 不被 Server.GracefulStop 管理，幂等忽略 net.ErrClosed）
	if p.listener != nil {
		if err := p.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			stopReason = errors.Join(stopReason, err)
		}
	}

	return stopReason
}

func (p *Plugin) getLifetimeCtx() context.Context {
	p.lifetimeMu.RLock()
	defer p.lifetimeMu.RUnlock()
	return p.lifetimeCtx
}

// extractRequestUUID 从元数据提取或生成 UUID。
func extractRequestUUID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(xGrpcConst.MetadataRequestUUID.String()); len(vals) > 0 && vals[0] != "" {
			return vals[0]
		}
	}
	return uuid.NewString()
}

// normalizeRPCError 对 RPC 调用过程产生的 error 进行标准化安全归一：
// 1. nil 原样返回 nil；
// 2. errors.As(*apperr.Error) 转为 grpcresult.ToStatusError；
// 3. context.Canceled / context.DeadlineExceeded 转换为对应标准 gRPC status 与固定 sentinel 文案（避免包装错泄漏 secret）；
// 4. 显式带有 gRPC status 的调用错误保持既有状态与语义不变；
// 5. 其余普通未知 error 用 namedLog 记录内部原错，并返回包含 ServerInternalError 的安全 50000 status 错误。
func (p *Plugin) normalizeRPCError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return grpcresult.ToStatusError(ctx, appErr)
	}

	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "上下文已取消")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "请求超时")
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	if p.namedLog != nil {
		p.namedLog.Error(ctx, fmt.Sprintf("gRPC 调用未预期内部错误: %v", err))
	}

	internalErr := apperr.New(apperr.ServerInternalError, "系统内部异常", err)
	return grpcresult.ToStatusError(ctx, internalErr)
}

// unaryBridgeInterceptor 实现一元调用的入口桥接：准入关卡、生命周期挂载、时间/追踪注入、
// ForceContext 取消通知级联、安全 panic 捕获与 grpcresult 错误转换。
func (p *Plugin) unaryBridgeInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if !p.gate.enter() {
			return nil, grpcresult.ToStatusError(ctx, apperr.New(apperr.ServiceUnavailable, "服务不可用或正在关机", ErrServerClosed))
		}
		defer p.gate.leave()

		// 派生可被本插件 Force 取消的 RPC 上下文分支
		rpcCtx, cancelRPC := context.WithCancel(ctx)
		defer cancelRPC()

		stopAfterFunc := context.AfterFunc(p.forceCtx, cancelRPC)
		defer stopAfterFunc()

		// 注入请求起始时间
		rpcCtx = context.WithValue(rpcCtx, ctxkey.UserStartTimeKey, time.Now())

		// 提取/生成 RequestID
		reqUUID := extractRequestUUID(rpcCtx)
		rpcCtx = context.WithValue(rpcCtx, ctxkey.RequestKey, reqUUID)

		// 写回响应 Header/Trailer
		trailerMD := metadata.Pairs(xGrpcConst.TrailerRequestUUID.String(), reqUUID)
		_ = grpc.SetHeader(rpcCtx, trailerMD)
		_ = grpc.SetTrailer(rpcCtx, trailerMD)

		// 挂入应用容器与可信 Worker 身份
		lifetime := p.getLifetimeCtx()
		bridgedCtx := application.BridgeRequest(rpcCtx, lifetime, reqUUID)

		// 拦截 panic，安全转换为 gRPC status 错误，绝不泄露敏感堆栈
		defer func() {
			if r := recover(); r != nil {
				if p.namedLog != nil {
					p.namedLog.Error(bridgedCtx, fmt.Sprintf("gRPC 一元调用发生 panic: %v", r))
				}
				panicErr := apperr.New(apperr.ServerInternalError, "系统内部异常", fmt.Errorf("panic: %v", r))
				resp = nil
				err = grpcresult.ToStatusError(bridgedCtx, panicErr)
			}
		}()

		// 检查私有中间件
		serviceName := xGrpcMiddle.ExtractServiceName(info.FullMethod)
		mws := p.mdRegistry.LookupUnary(serviceName)

		if len(mws) > 0 {
			curr := handler
			for i := len(mws) - 1; i >= 0; i-- {
				mw := mws[i]
				next := curr
				curr = func(c context.Context, r any) (any, error) {
					return mw(c, r, info, next)
				}
			}
			resp, err = curr(bridgedCtx, req)
		} else {
			resp, err = handler(bridgedCtx, req)
		}

		if err != nil {
			return nil, p.normalizeRPCError(bridgedCtx, err)
		}
		if resp == nil {
			return nil, grpcresult.ToStatusError(bridgedCtx, apperr.New(apperr.DeveloperError, "没有正常输出信息或报错信息，请检查代码逻辑「开发者错误」"))
		}

		return resp, nil
	}
}

// streamBridgeInterceptor 实现流式调用的入口桥接：准入关卡、生命周期挂载、时间/追踪注入、
// ForceContext 取消通知级联、安全 panic 捕获与 ServerStream.Context() 封装。
func (p *Plugin) streamBridgeInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		if !p.gate.enter() {
			return grpcresult.ToStatusError(ss.Context(), apperr.New(apperr.ServiceUnavailable, "服务不可用或正在关机", ErrServerClosed))
		}
		defer p.gate.leave()

		rawCtx := ss.Context()
		rpcCtx, cancelRPC := context.WithCancel(rawCtx)
		defer cancelRPC()

		stopAfterFunc := context.AfterFunc(p.forceCtx, cancelRPC)
		defer stopAfterFunc()

		rpcCtx = context.WithValue(rpcCtx, ctxkey.UserStartTimeKey, time.Now())
		reqUUID := extractRequestUUID(rpcCtx)
		rpcCtx = context.WithValue(rpcCtx, ctxkey.RequestKey, reqUUID)

		trailerMD := metadata.Pairs(xGrpcConst.TrailerRequestUUID.String(), reqUUID)
		_ = ss.SetHeader(trailerMD)
		ss.SetTrailer(trailerMD)

		lifetime := p.getLifetimeCtx()
		bridgedCtx := application.BridgeRequest(rpcCtx, lifetime, reqUUID)

		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          bridgedCtx,
		}

		defer func() {
			if r := recover(); r != nil {
				if p.namedLog != nil {
					p.namedLog.Error(bridgedCtx, fmt.Sprintf("gRPC 流式调用发生 panic: %v", r))
				}
				panicErr := apperr.New(apperr.ServerInternalError, "系统内部异常", fmt.Errorf("panic: %v", r))
				err = grpcresult.ToStatusError(bridgedCtx, panicErr)
			}
		}()

		serviceName := xGrpcMiddle.ExtractServiceName(info.FullMethod)
		mws := p.mdRegistry.LookupStream(serviceName)

		if len(mws) > 0 {
			curr := handler
			for i := len(mws) - 1; i >= 0; i-- {
				mw := mws[i]
				next := curr
				curr = func(s any, stream grpc.ServerStream) error {
					return mw(s, stream, info, next)
				}
			}
			err = curr(srv, wrapped)
		} else {
			err = handler(srv, wrapped)
		}

		return p.normalizeRPCError(bridgedCtx, err)
	}
}
