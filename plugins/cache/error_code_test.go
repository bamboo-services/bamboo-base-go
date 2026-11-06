package cache_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2"
)

func TestCacheErrorCode(t *testing.T) {
	want := apperr.ErrorCode{Code: 50016, Output: "CACHE_ERROR", Message: "缓存错误"}
	if cache.CacheError == nil || *cache.CacheError != want {
		t.Fatalf("plugin error code changed: got %+v, want %+v", cache.CacheError, want)
	}
	cause := errors.New("operation failed")
	err := apperr.New(cache.CacheError, "操作失败", cause)
	if err.Code != want || !errors.Is(err, cause) {
		t.Fatalf("plugin error code or cause was lost: %+v", err)
	}
}

// ExampleCacheError 演示调用方按需包装插件错误，保留原始根因。
func ExampleCacheError() {
	cause := errors.New("operation failed")
	err := apperr.New(cache.CacheError, "操作失败", cause)
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())
	fmt.Println(errors.Is(err, cause))
	// Output:
	// 50016 CACHE_ERROR
	// true
}
