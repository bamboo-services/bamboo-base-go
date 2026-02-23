package xCronRunner

import (
	"context"
	"log/slog"
	"sync"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xCron "github.com/bamboo-services/bamboo-base-go/plugins/cron"
	"github.com/robfig/cron/v3"
)

// Option 定义配置选项
type Option func(config *Config)

// Config 运行配置
type Config struct {
	GracefulStopTimeout time.Duration
	Logger              *xLog.LogNamedLogger
	Jobs                []xCron.Job
	WithSeconds         bool
	Location            *time.Location
}

// cronLogger 适配 xLog.LogNamedLogger 到 cron.Logger 接口
type cronLogger struct {
	log   *xLog.LogNamedLogger
	mu    sync.Mutex
	cache map[string]bool
}

// New 返回一个可直接挂载到 xMain.Runner 附加协程中的 Cron 启动函数。
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

// WithGracefulStopTimeout 设置优雅关闭超时时间
func WithGracefulStopTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		if timeout <= 0 {
			return
		}
		config.GracefulStopTimeout = timeout
	}
}

// WithLogger 设置 Cron Runner 日志器
func WithLogger(logger *xLog.LogNamedLogger) Option {
	return func(config *Config) {
		if logger == nil {
			return
		}
		config.Logger = logger
	}
}

// WithRegister 批量注册定时任务
func WithRegister(jobs ...xCron.Job) Option {
	return func(config *Config) {
		for _, job := range jobs {
			if job.Spec != "" && job.Func != nil {
				config.Jobs = append(config.Jobs, job)
			}
		}
	}
}

// WithSeconds 启用秒级 cron 支持
func WithSeconds() Option {
	return func(config *Config) {
		config.WithSeconds = true
	}
}

// WithLocation 设置时区
func WithLocation(loc *time.Location) Option {
	return func(config *Config) {
		if loc != nil {
			config.Location = loc
		}
	}
}

func defaultConfig() Config {
	return Config{
		GracefulStopTimeout: 30 * time.Second,
		Logger:              xLog.WithName(xLog.NamedCRON),
		Jobs:                make([]xCron.Job, 0),
		WithSeconds:         false,
		Location:            nil,
	}
}

func cloneConfig(config Config) Config {
	cloned := config
	cloned.Jobs = append([]xCron.Job(nil), config.Jobs...)
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
	log := config.Logger
	if log == nil {
		log = xLog.WithName(xLog.NamedCRON)
	}

	// 构建 cron 选项
	cronOpts := make([]cron.Option, 0)
	if config.WithSeconds {
		cronOpts = append(cronOpts, cron.WithSeconds())
	}
	if config.Location != nil {
		cronOpts = append(cronOpts, cron.WithLocation(config.Location))
	}
	// 添加自定义日志
	cronOpts = append(cronOpts, cron.WithLogger(newCronLogger(log)))

	// 创建 cron 管理器
	c := cron.New(cronOpts...)

	// 注册所有任务
	registeredCount := 0
	for _, job := range config.Jobs {
		jobFn, err := xCron.AdaptJob(job.Func)
		if err != nil {
			log.Warn(ctx, "任务适配失败", slog.String("spec", job.Spec), slog.String("error", err.Error()))
			continue
		}
		_, err = c.AddFunc(job.Spec, func() {
			jobFn(ctx)
		})
		if err != nil {
			log.Error(ctx, "任务注册失败", slog.String("spec", job.Spec), slog.String("error", err.Error()))
			continue
		}
		registeredCount++
	}

	// 启动 cron
	c.Start()
	log.Info(ctx, "Cron 服务已启动", slog.Int("jobs", registeredCount))

	// 等待上下文取消
	<-ctx.Done()

	// 优雅关闭
	log.Info(ctx, "Cron 服务正在关闭...")
	stopCron(c, config.GracefulStopTimeout)
	log.Info(ctx, "Cron 服务已退出")
}

func stopCron(c *cron.Cron, timeout time.Duration) {
	if c == nil {
		return
	}

	if timeout <= 0 {
		c.Stop()
		return
	}

	done := make(chan struct{})
	go func() {
		c.Stop()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		return
	}
}

// newCronLogger 创建 cron 日志适配器
func newCronLogger(log *xLog.LogNamedLogger) *cronLogger {
	return &cronLogger{
		log:   log,
		cache: make(map[string]bool),
	}
}

// Info 实现 cron.Logger 接口
func (l *cronLogger) Info(msg string, keysAndValues ...any) {
	ctx := context.Background()
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.cache[msg] {
		l.log.SugarInfo(ctx, msg, keysAndValues...)
		l.cache[msg] = true
	}
}

// Error 实现 cron.Logger 接口
func (l *cronLogger) Error(err error, msg string, keysAndValues ...any) {
	ctx := context.Background()
	l.log.SugarError(ctx, msg, append([]any{"error", err}, keysAndValues...)...)
}
