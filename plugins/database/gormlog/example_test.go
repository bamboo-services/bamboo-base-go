package gormlog_test

import (
	"fmt"
	"time"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/gormlog"
	gormlogger "gorm.io/gorm/logger"
)

func ExampleNew_nilLogger() {
	// 当传入的 logger 为 nil 时，函数返回 gormlogger.Discard，不依赖全局 slog
	l := gormlog.New(nil, gormlog.Config{
		SlowThreshold: 100 * time.Millisecond,
		LogLevel:      gormlog.LevelWarn,
	})

	fmt.Println(l == gormlogger.Discard)

	// Output:
	// true
}

func ExampleLogLevel() {
	// 查看日志级别常量的定义
	fmt.Println(gormlog.LevelSilent)
	fmt.Println(gormlog.LevelError)
	fmt.Println(gormlog.LevelWarn)
	fmt.Println(gormlog.LevelInfo)

	// Output:
	// 1
	// 2
	// 3
	// 4
}
