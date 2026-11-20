package main

import "testing"

func TestP5PluginDomainOwnership(t *testing.T) {
	entries := []errorEntry{
		{42900, "ASYNC_POOL_FULL", "plugins/async", "PoolFull", "异步任务池已满", "active"},
		{50000, "FRAMEWORK_INTERNAL_ERROR", "foundation/apperr", "FrameworkInternalError", "框架内部错误", "active"},
		{50001, "CACHE_ERROR", "plugins/cache", "CacheError", "缓存错误", "active"},
		{50002, "CACHE_DRIVER_CODEC_FAILED", "plugins/cache/driver", "CodecFailed", "缓存编解码失败", "active"},
		{50003, "CRON_SCHEDULER_ERROR", "plugins/cron", "SchedulerError", "定时调度器错误", "active"},
		{50004, "ASYNC_POOL_ERROR", "plugins/async", "PoolError", "异步任务池错误", "active"},
	}
	root := createFixture(t, entries, `var FrameworkInternalError = &ErrorCode{Code:50000, Output:"FRAMEWORK_INTERNAL_ERROR", Message:"框架内部错误"}`)
	writeFixture(t, root, "plugins/cache/error_code.go", `package cache
import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
var CacheError = &apperr.ErrorCode{Code:50001, Output:"CACHE_ERROR", Message:"缓存错误"}`)
	writeFixture(t, root, "plugins/cache/driver/error_code.go", `package driver
import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
var CodecFailed = &apperr.ErrorCode{Code:50002, Output:"CACHE_DRIVER_CODEC_FAILED", Message:"缓存编解码失败"}`)
	writeFixture(t, root, "plugins/cron/error_code.go", `package cron
import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
var SchedulerError = &apperr.ErrorCode{Code:50003, Output:"CRON_SCHEDULER_ERROR", Message:"定时调度器错误"}`)
	writeFixture(t, root, "plugins/async/error_code.go", `package async
import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
var PoolFull = &apperr.ErrorCode{Code:42900, Output:"ASYNC_POOL_FULL", Message:"异步任务池已满"}
var PoolError = &apperr.ErrorCode{Code:50004, Output:"ASYNC_POOL_ERROR", Message:"异步任务池错误"}`)
	if err := checkRepository(root); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		`import d "github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
var Alias = d.CodecFailed`,
		`import . "github.com/bamboo-services/bamboo-base-go/plugins/cron/v2"
var Alias = SchedulerError`,
		`import a "github.com/bamboo-services/bamboo-base-go/plugins/async/v2"
var Alias = a.PoolFull`,
	} {
		writeFixture(t, root, "plugins/email/extra.go", "package email\n"+source)
		if err := checkRepository(root); err == nil {
			t.Fatal("p5 subdomain or plugin alias bypass accepted")
		}
	}
	for _, entry := range []errorEntry{
		{50000, "CACHE_DRIVER_CODEC_FAILED", "plugins/cache", "CodecFailed", "错误", "active"},
		{50000, "CACHE_ERROR", "plugins/cache/driver", "OperationFailed", "错误", "active"},
		{50000, "ASYNC_SCHEDULER_ERROR", "plugins/cron", "SchedulerError", "错误", "active"},
		{42900, "CRON_POOL_FULL", "plugins/async", "PoolFull", "错误", "active"},
	} {
		if err := validateVocabulary([]errorEntry{entry}); err == nil {
			t.Fatalf("borrowed prefix accepted: %+v", entry)
		}
	}
}
