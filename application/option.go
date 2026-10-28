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

// Config 定义内核全局超时预算。
type Config struct {
	StartTimeout    time.Duration
	GraceTimeout    time.Duration
	CleanupTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// Option 供外部配置 App 超时预算。
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

// WithStartTimeout 设置单个插件或组件 Start 阶段的协作超时，默认 15s。
func WithStartTimeout(v time.Duration) Option {
	return func(c *optionCollector) {
		c.startTimeout = v
		c.hasStartTimeout = true
	}
}

// WithGraceTimeout 设置应用关闭时的自然排空窗口，默认 20s，自关机开始计时。
func WithGraceTimeout(v time.Duration) Option {
	return func(c *optionCollector) {
		c.graceTimeout = v
		c.hasGraceTimeout = true
	}
}

// WithCleanupTimeout 设置每次纯资源清理上限，默认 10s。
func WithCleanupTimeout(v time.Duration) Option {
	return func(c *optionCollector) {
		c.cleanupTimeout = v
		c.hasCleanupTimeout = true
	}
}

// WithShutdownTimeout 设置应用关机整体协调上限，默认 60s。
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
