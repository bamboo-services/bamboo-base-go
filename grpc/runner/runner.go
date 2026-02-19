package xGrpcRunner

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"time"

	xEnv "github.com/bamboo-services/bamboo-base-go/env"
	xGrpcInterface "github.com/bamboo-services/bamboo-base-go/grpc/interceptor"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// RegisterServiceFunc 定义 gRPC 服务注册函数。
//
// server 参数使用 grpc.ServiceRegistrar，便于直接调用生成代码中的 RegisterXxxServer。
type RegisterServiceFunc func(ctx context.Context, server grpc.ServiceRegistrar)

// Option 定义 gRPC Runner 的配置选项。
type Option func(config *Config)

// Config 表示 gRPC Runner 的运行配置。
type Config struct {
	GracefulStopTimeout time.Duration
	Logger              *xLog.LogNamedLogger
	RegisterServices    []RegisterServiceFunc
	UnaryInterceptors   []grpc.UnaryServerInterceptor
	StreamInterceptors  []grpc.StreamServerInterceptor
	ServerOptions       []grpc.ServerOption
}

// New 返回一个可直接挂载到 xMain.Runner 附加协程中的 gRPC 启动函数。
//
// 返回值函数签名与 xMain.Runner 的 goroutineFunc 保持一致：
//   - func(ctx context.Context, option ...any)
//
// 运行时 option 支持：
//   - Option / []Option：动态覆盖配置
func New(options ...Option) func(ctx context.Context, option ...any) {
	baseConfig := defaultConfig()
	for _, option := range options {
		if option != nil {
			option(&baseConfig)
		}
	}

	return func(ctx context.Context, option ...any) {
		runtimeConfig := cloneConfig(baseConfig)
		applyRuntimeOption(&runtimeConfig, option...)
		run(ctx, runtimeConfig)
	}
}

// WithGracefulStopTimeout 设置优雅关闭超时时间。
func WithGracefulStopTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		if timeout <= 0 {
			return
		}
		config.GracefulStopTimeout = timeout
	}
}

// WithLogger 设置 gRPC Runner 日志器。
func WithLogger(logger *xLog.LogNamedLogger) Option {
	return func(config *Config) {
		if logger == nil {
			return
		}
		config.Logger = logger
	}
}

// WithRegisterService 追加服务注册函数。
func WithRegisterService(registerService RegisterServiceFunc) Option {
	return func(config *Config) {
		if registerService == nil {
			return
		}
		config.RegisterServices = append(config.RegisterServices, registerService)
	}
}

// WithUnaryInterceptors 追加一元拦截器（支持后续扩展）。
func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(config *Config) {
		for _, interceptor := range interceptors {
			if interceptor == nil {
				continue
			}
			config.UnaryInterceptors = append(config.UnaryInterceptors, interceptor)
		}
	}
}

// WithStreamInterceptors 追加流式拦截器（支持后续扩展）。
func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(config *Config) {
		for _, interceptor := range interceptors {
			if interceptor == nil {
				continue
			}
			config.StreamInterceptors = append(config.StreamInterceptors, interceptor)
		}
	}
}

// WithServerOptions 追加原生 grpc.ServerOption。
func WithServerOptions(options ...grpc.ServerOption) Option {
	return func(config *Config) {
		for _, option := range options {
			if option == nil {
				continue
			}
			config.ServerOptions = append(config.ServerOptions, option)
		}
	}
}

func defaultConfig() Config {
	return Config{
		GracefulStopTimeout: 30 * time.Second,
		Logger:              xLog.WithName(xLog.NamedGRPC),
		RegisterServices:    make([]RegisterServiceFunc, 0),
		UnaryInterceptors:   make([]grpc.UnaryServerInterceptor, 0),
		StreamInterceptors:  make([]grpc.StreamServerInterceptor, 0),
		ServerOptions:       make([]grpc.ServerOption, 0),
	}
}

func cloneConfig(config Config) Config {
	cloned := config
	cloned.RegisterServices = append([]RegisterServiceFunc(nil), config.RegisterServices...)
	cloned.UnaryInterceptors = append([]grpc.UnaryServerInterceptor(nil), config.UnaryInterceptors...)
	cloned.StreamInterceptors = append([]grpc.StreamServerInterceptor(nil), config.StreamInterceptors...)
	cloned.ServerOptions = append([]grpc.ServerOption(nil), config.ServerOptions...)
	return cloned
}

func applyRuntimeOption(config *Config, option ...any) {
	for _, item := range option {
		switch value := item.(type) {
		case Option:
			if value != nil {
				value(config)
			}
		case []Option:
			for _, opt := range value {
				if opt != nil {
					opt(config)
				}
			}
		}
	}
}

func run(ctx context.Context, config Config) {
	address := resolveGrpcAddress()
	reflectionEnabled := xEnv.GetEnvBool(xEnv.GrpcReflection, false)

	log := config.Logger
	if log == nil {
		log = xLog.WithName(xLog.NamedGRPC)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Panic(ctx, "gRPC 服务监听失败",
			slog.String("addr", address),
			slog.String("error", err.Error()),
		)
		return
	}
	defer func() {
		_ = listener.Close()
	}()

	serverOptionList := make([]grpc.ServerOption, 0, len(config.ServerOptions)+2)
	serverOptionList = append(serverOptionList, config.ServerOptions...)
	unaryInterceptorList := make([]grpc.UnaryServerInterceptor, 0, len(config.UnaryInterceptors)+1)
	unaryInterceptorList = append(unaryInterceptorList, xGrpcInterface.Trace())
	unaryInterceptorList = append(unaryInterceptorList, config.UnaryInterceptors...)
	if len(unaryInterceptorList) > 0 {
		serverOptionList = append(serverOptionList, grpc.ChainUnaryInterceptor(unaryInterceptorList...))
	}
	if len(config.StreamInterceptors) > 0 {
		serverOptionList = append(serverOptionList, grpc.ChainStreamInterceptor(config.StreamInterceptors...))
	}

	grpcServer := grpc.NewServer(serverOptionList...)
	for _, registerService := range config.RegisterServices {
		if registerService == nil {
			continue
		}
		registerService(ctx, grpcServer)
	}
	if reflectionEnabled {
		reflection.Register(grpcServer)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- grpcServer.Serve(listener)
	}()

	log.Info(ctx, "gRPC 服务已启动", slog.String("addr", address), slog.Bool("reflection", reflectionEnabled))

	select {
	case <-ctx.Done():
		gracefulStop(grpcServer, config.GracefulStopTimeout)
		if serveErr := <-errChan; serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			log.Error(ctx, "gRPC 服务退出异常", slog.String("error", serveErr.Error()))
		}
		log.Info(ctx, "gRPC 服务已退出", slog.String("addr", address))
	case serveErr := <-errChan:
		if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			log.Panic(ctx, "gRPC 服务运行失败", slog.String("error", serveErr.Error()))
			return
		}
		log.Info(ctx, "gRPC 服务已退出", slog.String("addr", address))
	}
}

func resolveGrpcAddress() string {
	port := xEnv.GetEnvInt(xEnv.GrpcPort, 1119)
	if port <= 0 || port > 65535 {
		port = 1119
	}
	return ":" + strconv.Itoa(port)
}

func gracefulStop(server *grpc.Server, timeout time.Duration) {
	if server == nil {
		return
	}
	if timeout <= 0 {
		server.GracefulStop()
		return
	}

	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		server.Stop()
		return
	}
}
