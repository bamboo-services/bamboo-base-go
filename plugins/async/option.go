package async

// TaskOption 定义针对具体任务的配置项。
type TaskOption func(*taskConfig)

type taskConfig struct {
	name  string
	debug bool
}

func defaultTaskConfig() taskConfig {
	return taskConfig{
		name:  "unnamed_task",
		debug: false,
	}
}

// WithTaskName 设置任务的人类可读名称。
func WithTaskName(name string) TaskOption {
	return func(c *taskConfig) {
		if name != "" {
			c.name = name
		}
	}
}

// WithTaskDebug 为该任务开启调试日志。
func WithTaskDebug(debug bool) TaskOption {
	return func(c *taskConfig) {
		c.debug = debug
	}
}
