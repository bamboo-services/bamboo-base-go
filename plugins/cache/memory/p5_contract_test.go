package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
)

// p5FailingUnmarshalCodec 模拟写入成功但读取失败的编解码器。
type p5FailingUnmarshalCodec struct {
	driver.JSONCodec
}

func (p5FailingUnmarshalCodec) Unmarshal(data []byte, v any) *apperr.Error {
	return apperr.New(driver.CodecFailed, "强制反序列化失败", errors.New("forced unmarshal error"))
}

// 期望签名接口：编译期断言内存实现的方法集已整体切换为强类型错误。
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
	_ p5ExpectedKeyCache  = (driver.KeyCache[string, int])(nil)
	_ p5ExpectedHashCache = (driver.HashCache[string, string, int, map[string]int])(nil)
	_ p5ExpectedSetCache  = (driver.SetCache[string, int])(nil)
	_ p5ExpectedListCache = (driver.ListCache[string, int])(nil)

	_ p5ExpectedKeyCache  = (*KeyCache[string, int])(nil)
	_ p5ExpectedHashCache = (*HashCache[string, string, int, map[string]int])(nil)
	_ p5ExpectedSetCache  = (*SetCache[string, int])(nil)
	_ p5ExpectedListCache = (*ListCache[string, int])(nil)
)

// TestP5MemoryPopDecodeFailureKeepsList 断言 Pop 解码失败时返回错误且不弹出元素。
func TestP5MemoryPopDecodeFailureKeepsList(t *testing.T) {
	store := NewStore(0, 0, 0)
	defer store.Close()

	lc := NewListCache[string, string](store, p5FailingUnmarshalCodec{}, driver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()
	if err := lc.Append(ctx, "queue", []string{"a", "b"}); err != nil {
		t.Fatalf("Append 失败: %v", err)
	}

	value, err := lc.Pop(ctx, "queue")
	if err == nil {
		t.Fatal("期望 Pop 在解码失败时返回错误，实际得到 nil")
	}
	if err.Code.Code != driver.CodecFailed.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, driver.CodecFailed.Code)
	}
	if value != nil {
		t.Fatalf("期望解码失败时不返回元素，实际: %v", *value)
	}
	length, lenErr := lc.Len(ctx, "queue")
	if lenErr != nil {
		t.Fatalf("Len 失败: %v", lenErr)
	}
	if length != 2 {
		t.Fatalf("期望解码失败时列表保持不变（长度 2），实际: %d", length)
	}
}

// TestP5MemoryPopLastDecodeFailureKeepsList 断言 PopLast 解码失败时返回错误且不弹出元素。
func TestP5MemoryPopLastDecodeFailureKeepsList(t *testing.T) {
	store := NewStore(0, 0, 0)
	defer store.Close()

	lc := NewListCache[string, string](store, p5FailingUnmarshalCodec{}, driver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()
	if err := lc.Append(ctx, "queue", []string{"a", "b"}); err != nil {
		t.Fatalf("Append 失败: %v", err)
	}

	value, err := lc.PopLast(ctx, "queue")
	if err == nil {
		t.Fatal("期望 PopLast 在解码失败时返回错误，实际得到 nil")
	}
	if err.Code.Code != driver.CodecFailed.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, driver.CodecFailed.Code)
	}
	if value != nil {
		t.Fatalf("期望解码失败时不返回元素，实际: %v", *value)
	}
	length, lenErr := lc.Len(ctx, "queue")
	if lenErr != nil {
		t.Fatalf("Len 失败: %v", lenErr)
	}
	if length != 2 {
		t.Fatalf("期望解码失败时列表保持不变（长度 2），实际: %d", length)
	}
}

// TestP5MemoryMissReturnsTrueNil 断言键/字段/索引未命中与删除幂等均返回真正的 nil。
func TestP5MemoryMissReturnsTrueNil(t *testing.T) {
	store := NewStore(0, 0, 0)
	defer store.Close()
	ctx := context.Background()

	kc := NewKeyCache[string, string](store, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)
	value, ok, err := kc.Get(ctx, "absent")
	if err != nil {
		t.Fatalf("期望未命中返回 nil 错误，实际: %v", err)
	}
	if !apperr.IsNil(err) {
		t.Fatal("期望未命中返回空指针（真 nil）")
	}
	if ok || value != nil {
		t.Fatalf("期望未命中返回 (nil, false)，实际: (%v, %v)", value, ok)
	}
	if err := kc.Delete(ctx, "absent"); err != nil {
		t.Fatalf("期望删除不存在的键幂等成功，实际: %v", err)
	}

	hc := NewHashCache[string, string, string, map[string]string](store, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)
	fieldValue, fieldOK, err := hc.Get(ctx, "absent", "f")
	if err != nil || fieldOK || fieldValue != nil {
		t.Fatalf("期望哈希字段未命中返回 (nil, false, nil)，实际: (%v, %v, %v)", fieldValue, fieldOK, err)
	}
	if err := hc.Remove(ctx, "absent", "f"); err != nil {
		t.Fatalf("期望移除不存在的字段幂等成功，实际: %v", err)
	}

	sc := NewSetCache[string, string](store, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)
	members, err := sc.Members(ctx, "absent")
	if err != nil || members != nil {
		t.Fatalf("期望集合未命中返回 (nil, nil)，实际: (%v, %v)", members, err)
	}
	if err := sc.Remove(ctx, "absent", "m"); err != nil {
		t.Fatalf("期望移除不存在的成员幂等成功，实际: %v", err)
	}

	lc := NewListCache[string, string](store, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)
	element, err := lc.Index(ctx, "absent", 0)
	if err != nil || element != nil {
		t.Fatalf("期望列表未命中返回 (nil, nil)，实际: (%v, %v)", element, err)
	}
	popped, err := lc.Pop(ctx, "absent")
	if err != nil || popped != nil {
		t.Fatalf("期望空列表弹出返回 (nil, nil)，实际: (%v, %v)", popped, err)
	}
	if err := lc.Delete(ctx, "absent"); err != nil {
		t.Fatalf("期望删除不存在的列表幂等成功，实际: %v", err)
	}
}

// TestP5MemoryCodecFailurePropagates 断言编解码失败在内存后端保留 50038 原码。
func TestP5MemoryCodecFailurePropagates(t *testing.T) {
	store := NewStore(0, 0, 0)
	defer store.Close()
	ctx := context.Background()

	kc := NewKeyCache[string, string](store, p5FailingUnmarshalCodec{}, driver.DefaultKeyEncoder{}, 0)
	if err := kc.Set(ctx, "k", strPtr("v")); err != nil {
		t.Fatalf("Set 失败: %v", err)
	}
	_, _, err := kc.Get(ctx, "k")
	if err == nil {
		t.Fatal("期望解码失败返回错误，实际得到 nil")
	}
	if err.Code.Code != driver.CodecFailed.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, driver.CodecFailed.Code)
	}
	if err.Err == nil {
		t.Fatal("期望保留原生根因，实际根因为空")
	}
}

// TestP5MemoryNilReceiverDoesNotPanic 断言 nil 接收器不触发 panic。
func TestP5MemoryNilReceiverDoesNotPanic(t *testing.T) {
	ctx := context.Background()

	var kc *KeyCache[string, string]
	if _, _, err := kc.Get(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if err := kc.Set(ctx, "k", strPtr("v")); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if _, err := kc.Exists(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if err := kc.Delete(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}

	var lc *ListCache[string, string]
	if _, err := lc.Pop(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if _, err := lc.Len(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}

	var hc *HashCache[string, string, string, map[string]string]
	if _, _, err := hc.Get(ctx, "k", "f"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if _, err := hc.GetAll(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}

	var sc *SetCache[string, string]
	if _, err := sc.Members(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if _, err := sc.Count(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
}

// TestP5MemoryTTLStillExpires 断言迁移未破坏 TTL 语义。
func TestP5MemoryTTLStillExpires(t *testing.T) {
	store := NewStore(0, 0, 50*time.Millisecond)
	defer store.Close()

	kc := NewKeyCache[string, string](store, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 50*time.Millisecond)
	ctx := context.Background()
	if err := kc.Set(ctx, "k", strPtr("v")); err != nil {
		t.Fatalf("Set 失败: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	_, ok, err := kc.Get(ctx, "k")
	if err != nil {
		t.Fatalf("期望过期后返回 nil 错误，实际: %v", err)
	}
	if ok {
		t.Fatal("期望 TTL 过期后键未命中")
	}
}
