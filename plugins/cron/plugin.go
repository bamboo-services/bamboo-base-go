package cron

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/go-co-op/gocron/v2"
)

// DefaultCronPluginKey 是 Cron 插件默认的 Worker 插件 Key。
const DefaultCronPluginKey ctxkey.Key = "plugin_cron"

var (
	// ErrNoActivator 当未注入 application.Activator 时在 Start 返回。
	ErrNoActivator = errors.New("cron: 未显式注入 application.Activator，禁止隐式启动")
	// ErrInvalidJobTarget 当任务目标 Component Key 无效时返回。
	ErrInvalidJobTarget = errors.New("cron: 任务目标 Component Key 无效")
	// ErrInvalidJobSchedule 当任务调度表达式或间隔非法时返回。
	ErrInvalidJobSchedule = errors.New("cron: 任务调度表达式或间隔非法")
)

type jobKind uint8

const (
	jobKindCron jobKind = iota + 1
	jobKindInterval
)

type jobDef struct {
	kind      jobKind
	spec      string
	interval  time.Duration
	targetKey ctxkey.Key
}

// Option 为 Cron 插件配置项。
type Option func(*config)

type config struct {
	key       ctxkey.Key
	name      string
	requires  []ctxkey.Key
	activator application.Activator
	logger    *logging.Logger
	jobs      []jobDef
	location  *time.Location
}

// WithKey 设置插件 Key。
func WithKey(key ctxkey.Key) Option {
	return func(c *config) {
		if key.IsValid() {
			c.key = key
		}
	}
}

// WithName 设置插件人类可读名称。
func WithName(name string) Option {
	return func(c *config) {
		if name != "" {
			c.name = name
		}
	}
}

// WithRequires 显式声明插件额外依赖的共享资源（目标 Component 动态 Key 绝不在此声明）。
func WithRequires(keys ...ctxkey.Key) Option {
	return func(c *config) {
		for _, k := range keys {
			if k.IsValid() {
				c.requires = append(c.requires, k)
			}
		}
	}
}

// WithActivator 显式注入 application.Activator，不使用 global 或虚构 getter。
func WithActivator(act application.Activator) Option {
	return func(c *config) {
		c.activator = act
	}
}

// WithLogger 显式注入 Logger 实例，可选，无 Default。
func WithLogger(logger *logging.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithLocation 设置调度器时区。
func WithLocation(loc *time.Location) Option {
	return func(c *config) {
		c.location = loc
	}
}

// WithCronJob 配置基于标准 5 段 cron 表达式的目标 Component 触发任务。
func WithCronJob(spec string, targetKey ctxkey.Key) Option {
	return func(c *config) {
		c.jobs = append(c.jobs, jobDef{
			kind:      jobKindCron,
			spec:      spec,
			targetKey: targetKey,
		})
	}
}

// WithIntervalJob 配置基于固定 Duration 间隔的目标 Component 触发任务。
func WithIntervalJob(d time.Duration, targetKey ctxkey.Key) Option {
	return func(c *config) {
		c.jobs = append(c.jobs, jobDef{
			kind:      jobKindInterval,
			interval:  d,
			targetKey: targetKey,
		})
	}
}

// Plugin 实现 application.RunnablePlugin，为 ModeWorker 调度器。
type Plugin struct {
	cfg config

	scheduler gocron.Scheduler
	startOnce sync.Once
	startErr  error

	stopOnce sync.Once
	stopErr  error
	doneChan chan struct{}

	namedLog *logging.LogNamedLogger

	lifetimeCtx context.Context
	lifetimeMu  sync.RWMutex

	// activeTriggers 追踪飞行中到点触发的 activator.Activate 调用
	activeTriggers sync.WaitGroup
	gateMu         sync.Mutex
	quiesced       bool
	stopped        bool
}

// New 构造 Cron 插件信息与工厂函数。
// 工厂为纯构造，不创建后台调度器、不产生任何 I/O。
func New(opts ...Option) (application.PluginInfo, func() application.RunnablePlugin) {
	cfg := config{
		key:  DefaultCronPluginKey,
		name: "plugin_cron",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	info := application.PluginInfo{
		Key:      cfg.key,
		Name:     cfg.name,
		Mode:     application.ModeWorker,
		Requires: cfg.requires, // 严禁将 jobs 的 targetKey 放入静态 Requires
	}

	factory := func() application.RunnablePlugin {
		p := &Plugin{
			cfg:      cfg,
			doneChan: make(chan struct{}),
		}
		if cfg.logger != nil {
			p.namedLog = cfg.logger.WithName("CRON")
		}
		return p
	}

	return info, factory
}

// Start 验证表达式、校验激活器并在本阶段构建 gocron.Scheduler。
func (p *Plugin) Start(_ context.Context) error {
	p.startOnce.Do(func() {
		p.startErr = p.doStart()
	})
	return p.startErr
}

func (p *Plugin) doStart() error {
	if p.cfg.activator == nil {
		return ErrNoActivator
	}

	var schedOpts []gocron.SchedulerOption
	if p.cfg.location != nil {
		schedOpts = append(schedOpts, gocron.WithLocation(p.cfg.location))
	}

	s, err := gocron.NewScheduler(schedOpts...)
	if err != nil {
		return fmt.Errorf("cron: 创建 gocron 调度器失败: %w", err)
	}

	for idx, j := range p.cfg.jobs {
		if !j.targetKey.IsValid() {
			_ = s.Shutdown()
			return fmt.Errorf("%w: job[%d] key 为空或无效", ErrInvalidJobTarget, idx)
		}

		target := j.targetKey
		jobAction := func() {
			p.dispatchTick(target)
		}

		switch j.kind {
		case jobKindCron:
			if j.spec == "" {
				_ = s.Shutdown()
				return fmt.Errorf("%w: job[%d] cron 表达式为空", ErrInvalidJobSchedule, idx)
			}
			_, err = s.NewJob(
				gocron.CronJob(j.spec, false), // 5 段标准 cron，不启用秒级
				gocron.NewTask(jobAction),
			)
			if err != nil {
				_ = s.Shutdown()
				return fmt.Errorf("%w: job[%d] 表达式 [%s] 无效: %v", ErrInvalidJobSchedule, idx, j.spec, err)
			}
		case jobKindInterval:
			if j.interval <= 0 {
				_ = s.Shutdown()
				return fmt.Errorf("%w: job[%d] 间隔必须大于 0", ErrInvalidJobSchedule, idx)
			}
			_, err = s.NewJob(
				gocron.DurationJob(j.interval),
				gocron.NewTask(jobAction),
			)
			if err != nil {
				_ = s.Shutdown()
				return fmt.Errorf("%w: job[%d] 间隔 [%v] 注册失败: %v", ErrInvalidJobSchedule, idx, j.interval, err)
			}
		default:
			_ = s.Shutdown()
			return fmt.Errorf("%w: job[%d] 未知任务类型", ErrInvalidJobSchedule, idx)
		}
	}

	p.scheduler = s
	return nil
}

// Run 启动调度器并在 lifetimeCtx 结束时退出。
func (p *Plugin) Run(lifetimeCtx context.Context) error {
	if p.scheduler == nil {
		return errors.New("cron: 调度器尚未 Start 或 Start 失败")
	}

	p.lifetimeMu.Lock()
	p.lifetimeCtx = lifetimeCtx
	p.lifetimeMu.Unlock()

	p.scheduler.Start()

	<-lifetimeCtx.Done()
	return nil
}

// Quiesce 先原子关闭 tick 准入，不再发起新的组件激活。
func (p *Plugin) Quiesce(_ context.Context) error {
	p.gateMu.Lock()
	p.quiesced = true
	p.gateMu.Unlock()
	return nil
}

// Stop 停止调度器、等待所有已发起的激活调用返回后收尾退出。重复 Stop 共享结果。
func (p *Plugin) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		p.stopErr = p.doStop(ctx)
		close(p.doneChan)
	})
	<-p.doneChan
	return p.stopErr
}

func (p *Plugin) doStop(ctx context.Context) error {
	p.gateMu.Lock()
	p.quiesced = true
	p.stopped = true
	p.gateMu.Unlock()

	var schedErr error
	if p.scheduler != nil {
		schedErr = p.scheduler.Shutdown()
	}

	p.activeTriggers.Wait()

	if ctxErr := ctx.Err(); ctxErr != nil {
		if schedErr != nil {
			return errors.Join(schedErr, ctxErr)
		}
		return ctxErr
	}
	return schedErr
}

func (p *Plugin) dispatchTick(target ctxkey.Key) {
	p.gateMu.Lock()
	if p.quiesced || p.stopped {
		p.gateMu.Unlock()
		return
	}
	p.activeTriggers.Add(1)
	p.gateMu.Unlock()
	defer p.activeTriggers.Done()

	p.lifetimeMu.RLock()
	lCtx := p.lifetimeCtx
	p.lifetimeMu.RUnlock()

	if lCtx == nil {
		lCtx = context.Background()
	}

	tickTime := time.Now()
	// 到点仅调用 activator.Activate，不裸跑用户业务 func、不等待 Task.Run 结束
	_, err := p.cfg.activator.Activate(lCtx, target, application.WithPayload(tickTime))
	if err != nil && p.namedLog != nil {
		p.namedLog.SugarWarn(lCtx, "cron: 触发 component 激活失败", "target", string(target), "error", err)
	}
}
