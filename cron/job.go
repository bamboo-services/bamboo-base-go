package xCron

import (
	"context"
	"fmt"
	"reflect"
)

// Job 定义定时任务结构
type Job struct {
	Spec string // cron 表达式，如 "*/5 * * * *" 或 "@every 1m"
	Func any    // 支持 func() 或 func(context.Context)
}

// NewJob 创建定时任务
// 支持的函数签名:
//   - func()
//   - func(context.Context)
//
// 示例:
//
//	// 无参数函数
//	xCron.NewJob("@every 1m", func() {
//	    log.Println("每分钟执行")
//	})
//
//	// 带上下文的函数
//	xCron.NewJob("*/5 * * * *", func(ctx context.Context) {
//	    log.Println("每5分钟执行")
//	})
func NewJob(spec string, fn any) Job {
	return Job{
		Spec: spec,
		Func: fn,
	}
}

// jobFunc 内部统一的任务函数签名
type jobFunc func(ctx context.Context)

// AdaptJob 适配任务函数，支持多种签名
// 支持的函数签名:
//   - func()
//   - func(context.Context)
func AdaptJob(fn any) (jobFunc, error) {
	if fn == nil {
		return nil, fmt.Errorf("cron job func 不能为 nil")
	}

	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return nil, fmt.Errorf("cron job func 必须是函数类型")
	}

	t := v.Type()
	switch t.NumIn() {
	case 0:
		// func() -> 包装为 func(context.Context)
		return func(ctx context.Context) {
			v.Call(nil)
		}, nil
	case 1:
		// func(context.Context) -> 直接使用
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
		if t.In(0) != contextType {
			return nil, fmt.Errorf("参数类型必须是 context.Context")
		}
		return func(ctx context.Context) {
			v.Call([]reflect.Value{reflect.ValueOf(ctx)})
		}, nil
	default:
		return nil, fmt.Errorf("cron job func 最多接受一个 context.Context 参数")
	}
}
