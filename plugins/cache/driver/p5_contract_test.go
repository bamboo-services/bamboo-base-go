package driver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
)

// 期望签名接口：编译期断言 driver 暴露的方法集已整体切换为强类型错误。
type p5ExpectedCodec interface {
	Marshal(v any) ([]byte, *apperr.Error)
	Unmarshal(data []byte, v any) *apperr.Error
}

type p5ExpectedKeyCache interface {
	Get(ctx context.Context, key string) (*int, bool, *apperr.Error)
	Set(ctx context.Context, key string, value *int, opts ...driver.SetOption) *apperr.Error
	Exists(ctx context.Context, key string) (bool, *apperr.Error)
	Delete(ctx context.Context, key string) *apperr.Error
}

type p5ExpectedHashCache interface {
	Get(ctx context.Context, key string, field string) (*int, bool, *apperr.Error)
	Set(ctx context.Context, key string, field string, value *int, opts ...driver.SetOption) *apperr.Error
	GetAll(ctx context.Context, key string) (map[string]int, *apperr.Error)
	GetAllStruct(ctx context.Context, key string) (map[string]int, *apperr.Error)
	SetAll(ctx context.Context, key string, fields map[string]*int, opts ...driver.SetOption) *apperr.Error
	SetAllStruct(ctx context.Context, key string, value map[string]int, opts ...driver.SetOption) *apperr.Error
	Exists(ctx context.Context, key string, field string) (bool, *apperr.Error)
	Remove(ctx context.Context, key string, fields ...string) *apperr.Error
	Delete(ctx context.Context, key string) *apperr.Error
}

type p5ExpectedSetCache interface {
	Add(ctx context.Context, key string, members []int, opts ...driver.SetOption) *apperr.Error
	Members(ctx context.Context, key string) ([]int, *apperr.Error)
	IsMember(ctx context.Context, key string, member int) (bool, *apperr.Error)
	Count(ctx context.Context, key string) (int64, *apperr.Error)
	Remove(ctx context.Context, key string, members ...int) *apperr.Error
	Delete(ctx context.Context, key string) *apperr.Error
}

type p5ExpectedListCache interface {
	Prepend(ctx context.Context, key string, values []int, opts ...driver.SetOption) *apperr.Error
	Append(ctx context.Context, key string, values []int, opts ...driver.SetOption) *apperr.Error
	Range(ctx context.Context, key string, start int64, end int64) ([]int, *apperr.Error)
	Index(ctx context.Context, key string, index int64) (*int, *apperr.Error)
	Len(ctx context.Context, key string) (int64, *apperr.Error)
	Pop(ctx context.Context, key string) (*int, *apperr.Error)
	PopLast(ctx context.Context, key string) (*int, *apperr.Error)
	Remove(ctx context.Context, key string, count int64, value int) *apperr.Error
	Delete(ctx context.Context, key string) *apperr.Error
}

var (
	_ p5ExpectedCodec     = driver.JSONCodec{}
	_ p5ExpectedKeyCache  = (driver.KeyCache[string, int])(nil)
	_ p5ExpectedHashCache = (driver.HashCache[string, string, int, map[string]int])(nil)
	_ p5ExpectedSetCache  = (driver.SetCache[string, int])(nil)
	_ p5ExpectedListCache = (driver.ListCache[string, int])(nil)

	// 方法值签名断言：确保实现侧的返回类型与接口一致。
	_ func(any) ([]byte, *apperr.Error) = driver.JSONCodec{}.Marshal
	_ func([]byte, any) *apperr.Error   = driver.JSONCodec{}.Unmarshal
)

// TestP5DriverErrorCodes 断言 P5.0 冻结的两个错误码编号与输出标识。
func TestP5DriverErrorCodes(t *testing.T) {
	if driver.CodecFailed == nil || driver.CodecFailed.Code != 50038 {
		t.Fatalf("CodecFailed 编号错误: %+v", driver.CodecFailed)
	}
	if driver.CodecFailed.Output != "CACHE_DRIVER_CODEC_FAILED" {
		t.Fatalf("CodecFailed 输出标识错误: %s", driver.CodecFailed.Output)
	}
	if driver.OperationFailed == nil || driver.OperationFailed.Code != 50039 {
		t.Fatalf("OperationFailed 编号错误: %+v", driver.OperationFailed)
	}
	if driver.OperationFailed.Output != "CACHE_DRIVER_OPERATION_FAILED" {
		t.Fatalf("OperationFailed 输出标识错误: %s", driver.OperationFailed.Output)
	}
}

// TestP5JSONCodecMarshalFailureUsesCodecFailed 断言序列化失败归一为 50038 并保留原生根因。
func TestP5JSONCodecMarshalFailureUsesCodecFailed(t *testing.T) {
	codec := driver.JSONCodec{}
	_, err := codec.Marshal(func() {})
	if err == nil {
		t.Fatal("期望序列化失败返回错误，实际得到 nil")
	}
	if err.Code.Code != driver.CodecFailed.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, driver.CodecFailed.Code)
	}
	if err.Err == nil {
		t.Fatal("期望保留 encoding/json 原生根因，实际根因为空")
	}
}

// TestP5JSONCodecUnmarshalFailureUsesCodecFailed 断言反序列化失败归一为 50038 并保留原生根因。
func TestP5JSONCodecUnmarshalFailureUsesCodecFailed(t *testing.T) {
	codec := driver.JSONCodec{}
	var target int
	err := codec.Unmarshal([]byte("not-json"), &target)
	if err == nil {
		t.Fatal("期望反序列化失败返回错误，实际得到 nil")
	}
	if err.Code.Code != driver.CodecFailed.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, driver.CodecFailed.Code)
	}
	if err.Err == nil {
		t.Fatal("期望保留 encoding/json 原生根因，实际根因为空")
	}
	if !errors.Is(err, err.Err) {
		t.Fatal("期望原生根因可通过 errors.Is 命中")
	}
}

// TestP5JSONCodecSuccessReturnsTrueNil 断言成功路径返回真正的 nil 指针。
func TestP5JSONCodecSuccessReturnsTrueNil(t *testing.T) {
	codec := driver.JSONCodec{}
	data, err := codec.Marshal(map[string]int{"a": 1})
	if err != nil {
		t.Fatalf("期望成功返回 nil 错误，实际: %v", err)
	}
	if !apperr.IsNil(err) {
		t.Fatal("期望成功路径错误为空指针（真 nil）")
	}
	var target map[string]int
	if err := codec.Unmarshal(data, &target); err != nil {
		t.Fatalf("期望成功返回 nil 错误，实际: %v", err)
	}
	if target["a"] != 1 {
		t.Fatalf("反序列化结果错误: %+v", target)
	}
}
