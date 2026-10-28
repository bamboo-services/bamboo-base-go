package cron_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/plugins/cron/v2"
)

// ExampleNew 展示如何通过函数式选项配置并构造 Cron 定时调度 Worker 插件定义。
func ExampleNew() {
	info, _ := cron.New(
		cron.WithName("order_scheduler"),
		cron.WithCronJob("0 * * * *", ctxkey.Key("hourly_order_sync")),
	)

	fmt.Println(info.Key)
	fmt.Println(info.Name)
	fmt.Println(info.Mode == application.ModeWorker)
	// Output:
	// plugin_cron
	// order_scheduler
	// true
}

// ExampleWithCronJob 展示如何配置基于标准 Cron 表达式的定时触发选项。
func ExampleWithCronJob() {
	opt := cron.WithCronJob("0 2 * * *", ctxkey.Key("daily_report_job"))
	fmt.Printf("%T\n", opt)
	// Output:
	// cron.Option
}
