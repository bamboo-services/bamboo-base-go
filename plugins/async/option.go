package async

// TaskOption 定义针对单个异步任务的函数式配置项。
// 传入 nil 时将被 [Pool.Submit] 安全忽略。
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

// WithTaskName 设置任务的人类可读名称，便于在日志追踪和调试中识别。
// 若 name 为空字符串，则保持默认任务名称 "unnamed_task"。
func WithTaskName(name string) TaskOption {
	return func(c *taskConfig) {
		if name != "" {
			c.name = name
		}
	}
}

// WithTaskDebug 为该任务开启或关闭独立调试日志输出。
// 开启且插件注入了日志器时，将在任务启动与完成时记录详细调试信息。
func WithTaskDebug(debug bool) TaskOption {
	return func(c *taskConfig) {
		c.debug = debug
	}
}
