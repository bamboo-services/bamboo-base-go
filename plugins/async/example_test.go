package async_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/async/v2"
)

// ExampleNew 展示如何通过函数式选项配置并构造 Async 异步协程池插件定义。
func ExampleNew() {
	info, _ := async.New(
		async.WithName("async_worker_pool"),
		async.WithCapacity(500),
	)

	fmt.Println(info.Key)
	fmt.Println(info.Name)
	fmt.Println(info.Mode == application.ModeTool)
	fmt.Println(info.Provides[0])
	// Output:
	// plugin_async
	// async_worker_pool
	// true
	// context_async_pool
}

// ExampleWithTaskName 展示如何为异步任务指定可读名称。
func ExampleWithTaskName() {
	opt := async.WithTaskName("email_notification")
	fmt.Printf("%T\n", opt)
	// Output:
	// async.TaskOption
}
