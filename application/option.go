package application

import (
	"errors"
	"fmt"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// 默认超时预算。
const (
	defaultStartTimeout    = 15 * time.Second
	defaultGraceTimeout    = 20 * time.Second
	defaultCleanupTimeout  = 10 * time.Second
	defaultShutdownTimeout = 60 * time.Second
)

// Config 定义内核全局超时预算与时间窗口。
//
// 该配置由 [New] 在初始化时通过选项 [Option]、环境变量或内建默认值解析生成，
// 决定插件启动、关机排空、清理及整体协调的截止时限。
type Config struct {
	// StartTimeout 是单个插件或动态组件执行 Start 钩子的协作超时时间，默认 15s。
	StartTimeout time.Duration

	// GraceTimeout 是应用关机时的自然排空窗口，自关机触发时开始计时，默认 20s。
	// 该时间窗口到期后将触发强制取消根通知，且必须严格小于 ShutdownTimeout。
	GraceTimeout time.Duration

	// CleanupTimeout 是单个插件执行 Stop 钩子的纯资源清理时间上限，默认 10s。
	CleanupTimeout time.Duration

	// ShutdownTimeout 是应用平滑关机的全局整体协调时限，默认 60s。
	// 超过此时间若仍有实例未退，应用将以 [StateIncomplete] 结束停机。
	ShutdownTimeout time.Duration
}

// Option 供外部配置 [App] 的超时预算。
//
// 选项函数在调用 [New] 时传入，未显式配置的字段将尝试从环境变量读取；
// 若环境变量未配置，则回退至内建默认值。所有时长参数必须为正数。
type Option func(*optionCollector)

type optionCollector struct {
	startTimeout       time.Duration
	hasStartTimeout    bool
	graceTimeout       time.Duration
	hasGraceTimeout    bool
	cleanupTimeout     time.Duration
	hasCleanupTimeout  bool
	shutdownTimeout    time.Duration
	hasShutdownTimeout bool
}

// WithStartTimeout 设置单个插件或组件 Start 阶段的协作超时时间。
//
// 默认值为 15s。若传入时长小于或等于 0，将在配置解析时报错。
func WithStartTimeout(v time.Duration) Option {
	return func(c *optionCollector) {
		c.startTimeout = v
		c.hasStartTimeout = true
	}
}

// WithGraceTimeout 设置应用关机时的自然排空窗口时间。
//
// 默认值为 20s，自关机协调触发时开始单调计时。排空窗口到期后将触发
// [ForceContext] 强制取消通知。传入时长必须为正数且严格小于 [WithShutdownTimeout]
// 设置的关机总超时，否则在配置解析时报错。
func WithGraceTimeout(v time.Duration) Option {
	return func(c *optionCollector) {
		c.graceTimeout = v
		c.hasGraceTimeout = true
	}
}

// WithCleanupTimeout 设置单个插件或组件执行 Stop 钩子的纯资源清理时间上限。
//
// 默认值为 10s。传入时长必须大于 0，否则在配置解析时报错。
func WithCleanupTimeout(v time.Duration) Option {
	return func(c *optionCollector) {
		c.cleanupTimeout = v
		c.hasCleanupTimeout = true
	}
}

// WithShutdownTimeout 设置应用关机整体协调的总截止超时时间。
//
// 默认值为 60s。关机开始后若超过该时长仍有组件未能确认停止，
// 停机流程将提前返回并将应用置为 [StateIncomplete]。
// 传入时长必须大于 0，且必须严格大于 [WithGraceTimeout] 的排空窗口。
func WithShutdownTimeout(v time.Duration) Option {
	return func(c *optionCollector) {
		c.shutdownTimeout = v
		c.hasShutdownTimeout = true
	}
}

// resolveConfig 解析配置，优先级：显式 Option > 环境变量 > 默认值。
func resolveConfig(opts ...Option) (Config, error) {
	var collector optionCollector
	for _, opt := range opts {
		if opt != nil {
			opt(&collector)
		}
	}

	start, err := resolveField(
		collector.hasStartTimeout,
		collector.startTimeout,
		env.StartTimeout,
		defaultStartTimeout,
	)
	if err != nil {
		return Config{}, fmt.Errorf("invalid StartTimeout: %w", err)
	}

	grace, err := resolveField(
		collector.hasGraceTimeout,
		collector.graceTimeout,
		env.GraceTimeout,
		defaultGraceTimeout,
	)
	if err != nil {
		return Config{}, fmt.Errorf("invalid GraceTimeout: %w", err)
	}

	cleanup, err := resolveField(
		collector.hasCleanupTimeout,
		collector.cleanupTimeout,
		env.CleanupTimeout,
		defaultCleanupTimeout,
	)
	if err != nil {
		return Config{}, fmt.Errorf("invalid CleanupTimeout: %w", err)
	}

	shutdown, err := resolveField(
		collector.hasShutdownTimeout,
		collector.shutdownTimeout,
		env.ShutdownTimeout,
		defaultShutdownTimeout,
	)
	if err != nil {
		return Config{}, fmt.Errorf("invalid ShutdownTimeout: %w", err)
	}

	if grace >= shutdown {
		return Config{}, fmt.Errorf("invalid timeout budgets: GraceTimeout (%v) must be strictly less than ShutdownTimeout (%v)", grace, shutdown)
	}

	return Config{
		StartTimeout:    start,
		GraceTimeout:    grace,
		CleanupTimeout:  cleanup,
		ShutdownTimeout: shutdown,
	}, nil
}

func resolveField(explicit bool, explicitVal time.Duration, envKey env.Key, defVal time.Duration) (time.Duration, error) {
	if explicit {
		if explicitVal <= 0 {
			return 0, fmt.Errorf("duration must be positive, got %v", explicitVal)
		}
		return explicitVal, nil
	}

	strVal, ok := env.Lookup(envKey)
	if ok {
		raw := string(strVal)
		if raw == "" {
			return 0, errors.New("environment variable is set but empty")
		}
		// typed.String.Duration 不接受裸数字，纯数字没有单位时会失败
		d, err := typed.String(raw).Duration()
		if err != nil {
			return 0, fmt.Errorf("failed to parse environment variable %s: %w", envKey, err)
		}
		if d <= 0 {
			return 0, fmt.Errorf("parsed duration from %s must be positive, got %v", envKey, d)
		}
		return d, nil
	}

	return defVal, nil
}
