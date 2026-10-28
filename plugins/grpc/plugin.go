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

// DefaultGRPCPluginKey 是 gRPC 插件默认的 Worker 插件 Key。
const DefaultGRPCPluginKey ctxkey.Key = "plugin_grpc"

var (
	// ErrServerAlreadyStarted 当重复调用 Start 时返回。
	ErrServerAlreadyStarted = errors.New("grpc: 服务已启动")
	// ErrServerClosed 当服务已停止或正在关闭时返回。
	ErrServerClosed = errors.New("grpc: 服务已停止")
	// ErrNoLifetimeContext 当未设置可信 lifetimeCtx 即尝试运行或处理请求时返回。
	ErrNoLifetimeContext = errors.New("grpc: 缺少 lifetime context")
	// ErrStopTimeout 当服务优雅停机超时时返回。
	ErrStopTimeout = errors.New("grpc: 停机等待超时")
)

// ServiceRegistrar 是业务向 grpc.Server 注册 Service 的回调函数。
type ServiceRegistrar func(ctx context.Context, s *grpc.Server)

// Option 为 gRPC 插件的配置函数。
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

// WithAddr 设置监听地址（如 ":9090" 或 "127.0.0.1:0"）。
// 显式传入非空地址时优先采用；未显式指定时，默认从环境变量 GRPC_PORT 读取端口（缺省或无效为 1119），监听地址为 ":<port>"。
func WithAddr(addr string) Option {
	return func(c *config) {
		if addr != "" {
			c.addr = addr
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

// WithService 注册 gRPC 服务挂载回调。
func WithService(fn ServiceRegistrar) Option {
	return func(c *config) {
		if fn != nil {
			c.registrars = append(c.registrars, fn)
		}
	}
}

// WithLogger 设置显式传入的 Logger 实例。
func WithLogger(logger *logging.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithServerOptions 追加原生 grpc.ServerOption。
func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(c *config) {
		c.serverOptions = append(c.serverOptions, opts...)
	}
}

// WithUnaryInterceptors 追加一元拦截器。
func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(c *config) {
		c.unaryInterceptors = append(c.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors 追加流式拦截器。
func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(c *config) {
		c.streamInterceptors = append(c.streamInterceptors, interceptors...)
	}
}

// WithMiddlewareRegistry 设置实例独享的服务中间件注册表。
func WithMiddlewareRegistry(reg *xGrpcMiddle.Registry) Option {
	return func(c *config) {
		c.middlewareRegistry = reg
	}
}

// WithReflection 配置是否启用 gRPC 反射服务。
// 显式传入参数时记录配置偏好；未显式调用时后续逻辑可根据环境默认策略处理。
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

// Plugin 实现 application.RunnablePlugin，作为 gRPC Worker 插件。
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

// New 返回 Worker 的应用注册信息以及 RunnablePlugin 构造工厂（构造期不进行任何 I/O 与后台操作）。
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

// Addr 返回已分配或配置的监听地址字符串（若已成功启动，返回实际绑定的 socket 地址）。
// 未 Start 时返回解析后的监听地址。
func (p *Plugin) Addr() string {
	if p.listener != nil {
		return p.listener.Addr().String()
	}
	return resolveGRPCAddr(p.cfg.addr)
}

// Start 完成端口监听分配、Server 初始化、私有拦截器装配与业务 Service 挂载。
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

// Run 封存可信运行 lifetimeCtx，并开始阻塞 Serve。
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

// Quiesce 入站关闸：仅关闭新请求准入闸门，不调用 Stop、不取消 Handlers、不分配新预算。
func (p *Plugin) Quiesce(ctx context.Context) error {
	p.gate.closeGate()
	return nil
}

// Stop 真实优雅关机。在私有 goroutine 运行 GracefulStop；
// 同时监听 ctx 超时/取消及 App Force 信号，必要时通过 Server.Stop 中断传输；
// 严格关准入并等待飞行中 Handler 真退（不因 ctx 截止假返回）。
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
