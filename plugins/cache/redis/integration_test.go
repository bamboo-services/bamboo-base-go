package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	driver "github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
	goredis "github.com/redis/go-redis/v9"
)

// customItem 用于测试真实结构体序列化/反序列化
type customItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// customStructHash 用于测试 HashCache GetAllStruct / SetAllStruct
type customStructHash struct {
	FieldA string `json:"fieldA"`
	FieldB string `json:"fieldB"`
}

// failingCodec 用于测试编解码错误路径
type failingCodec struct {
	failMarshal   bool
	failUnmarshal bool
}

func (f *failingCodec) Marshal(v any) ([]byte, error) {
	if f.failMarshal {
		return nil, errors.New("forced marshal error")
	}
	return driver.JSONCodec{}.Marshal(v)
}

func (f *failingCodec) Unmarshal(data []byte, v any) error {
	if f.failUnmarshal {
		return errors.New("forced unmarshal error")
	}
	return driver.JSONCodec{}.Unmarshal(data, v)
}

// errorKeyEncoder 用于测试 KeyEncoder 自定义前缀/转换
type errorKeyEncoder struct {
	prefix string
}

func (e *errorKeyEncoder) String(key any) string {
	return fmt.Sprintf("%s:%v", e.prefix, key)
}

// realRedisEnv 获取受限集成环境的 Redis 凭据。
// 遵循约束：必须有 BAMBOO_TEST_REDIS_ADDR，且 BAMBOO_TEST_REDIS_ENABLED 显式为 "true" 或 "1"。
// 绝对不读取 NOSQL_*，绝对不使用默认 127.0.0.1:6379 盲连，无配置时优雅 skip。
func realRedisEnv(t *testing.T) (addr, password string) {
	t.Helper()
	enabled := os.Getenv("BAMBOO_TEST_REDIS_ENABLED")
	if enabled != "true" && enabled != "1" {
		t.Skip("skipping real redis integration test: BAMBOO_TEST_REDIS_ENABLED is not set to true or 1")
	}

	addr = os.Getenv("BAMBOO_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("skipping real redis integration test: BAMBOO_TEST_REDIS_ADDR is empty")
	}

	password = os.Getenv("BAMBOO_TEST_REDIS_PASSWORD")
	return addr, password
}

// newTestRedisClient 创建连接到真实 Redis 夹具的客户端，并在测试结束时关闭连接。
func newTestRedisClient(t *testing.T, addr, password string) *goredis.Client {
	t.Helper()
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		t.Fatalf("failed to ping real redis fixture at %s: %v", addr, err)
	}

	t.Cleanup(func() {
		_ = rdb.Close()
	})
	return rdb
}

// testKeyTracker 记录测试过程中产生的所有精确 key，并在清理时仅删除自己记录的 key。
// 严禁 FLUSHDB/FLUSHALL 或 KEYS * 扫描整个库。
type testKeyTracker struct {
	mu     sync.Mutex
	keys   map[string]struct{}
	prefix string
	rdb    *goredis.Client
}

func newKeyTracker(t *testing.T, rdb *goredis.Client) *testKeyTracker {
	t.Helper()
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("failed to generate random prefix: %v", err)
	}
	prefix := fmt.Sprintf("bb_test_%s_%d", hex.EncodeToString(buf), time.Now().UnixNano())
	tracker := &testKeyTracker{
		keys:   make(map[string]struct{}),
		prefix: prefix,
		rdb:    rdb,
	}

	t.Cleanup(func() {
		tracker.cleanup()
	})
	return tracker
}

func (tk *testKeyTracker) TrackKey(subKey string) string {
	tk.mu.Lock()
	defer tk.mu.Unlock()
	fullKey := fmt.Sprintf("%s:%s", tk.prefix, subKey)
	tk.keys[fullKey] = struct{}{}
	return fullKey
}

func (tk *testKeyTracker) cleanup() {
	tk.mu.Lock()
	defer tk.mu.Unlock()
	if len(tk.keys) == 0 {
		return
	}
	keys := make([]string, 0, len(tk.keys))
	for k := range tk.keys {
		keys = append(keys, k)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tk.rdb.Del(ctx, keys...).Err()
}

func TestIntegration_RedisKeyCache(t *testing.T) {
	addr, password := realRedisEnv(t)
	rdb := newTestRedisClient(t, addr, password)
	tracker := newKeyTracker(t, rdb)
	ctx := context.Background()

	// 1. 基本读写删与结构体
	rawKey := tracker.TrackKey("key_basic")
	cache := NewKeyCache[string, customItem](rdb, nil, nil, 10*time.Minute)

	// Exists 不存在
	exists, err := cache.Exists(ctx, rawKey)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatalf("expected key not exists initially")
	}

	// Get 不存在返回 nil, false, nil
	got, found, err := cache.Get(ctx, rawKey)
	if err != nil {
		t.Fatalf("Get unexpected err: %v", err)
	}
	if found || got != nil {
		t.Fatalf("expected not found, got %v", got)
	}

	// Set 正常写入
	item := customItem{ID: 101, Title: "BambooBase"}
	if err := cache.Set(ctx, rawKey, &item); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Exists 应该为 true
	exists, err = cache.Exists(ctx, rawKey)
	if err != nil {
		t.Fatalf("Exists after Set failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected key exists after Set")
	}

	// Get 应该读到相同数据
	got, found, err = cache.Get(ctx, rawKey)
	if err != nil {
		t.Fatalf("Get after Set failed: %v", err)
	}
	if !found || got == nil || *got != item {
		t.Fatalf("expected %v, got %v", item, got)
	}

	// 原生 TTL 校验（确认存在且大于 0，不是断言 elapsed）
	ttlCmd := rdb.TTL(ctx, rawKey)
	if ttl := ttlCmd.Val(); ttl <= 0 || ttl > 10*time.Minute {
		t.Fatalf("expected positive TTL <= 10m, got %v", ttl)
	}

	// Set value=nil 等价于 Delete
	if err := cache.Set(ctx, rawKey, nil); err != nil {
		t.Fatalf("Set(nil) delete equivalent failed: %v", err)
	}
	exists, err = cache.Exists(ctx, rawKey)
	if err != nil || exists {
		t.Fatalf("expected key removed after Set(nil), exists=%v, err=%v", exists, err)
	}

	// 2. Options: WithNX / WithXX / WithTTL / WithKeepTTL
	nxKey := tracker.TrackKey("key_nx")
	val1 := customItem{ID: 1, Title: "V1"}
	val2 := customItem{ID: 2, Title: "V2"}

	// NX 在不存在时写入
	if err := cache.Set(ctx, nxKey, &val1, driver.WithNX(), driver.WithTTL(5*time.Minute)); err != nil {
		t.Fatalf("Set WithNX failed: %v", err)
	}
	// NX 在已存在时应跳过写入（返回 nil 非错误）
	if err := cache.Set(ctx, nxKey, &val2, driver.WithNX()); err != nil {
		t.Fatalf("Set WithNX on existing key should return nil err, got: %v", err)
	}
	got, _, _ = cache.Get(ctx, nxKey)
	if got == nil || got.ID != 1 {
		t.Fatalf("expected val1 retained under NX, got %v", got)
	}

	// XX 在存在时写入
	xxVal := customItem{ID: 3, Title: "V3"}
	if err := cache.Set(ctx, nxKey, &xxVal, driver.WithXX(), driver.WithKeepTTL()); err != nil {
		t.Fatalf("Set WithXX WithKeepTTL failed: %v", err)
	}
	got, _, _ = cache.Get(ctx, nxKey)
	if got == nil || got.ID != 3 {
		t.Fatalf("expected val3 updated under XX, got %v", got)
	}

	// Delete 正常删除
	if err := cache.Delete(ctx, nxKey); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	// XX 在不存在时跳过写入
	if err := cache.Set(ctx, nxKey, &xxVal, driver.WithXX()); err != nil {
		t.Fatalf("Set WithXX on non-existent key should return nil err, got: %v", err)
	}
	exists, _ = cache.Exists(ctx, nxKey)
	if exists {
		t.Fatalf("expected key not created under XX when absent")
	}

	// 3. TTL 过期测试：短 TTL 写入，原生 TTL 查询并确认最终过期
	expireKey := tracker.TrackKey("key_expire")
	if err := cache.Set(ctx, expireKey, &val1, driver.WithTTL(1*time.Second)); err != nil {
		t.Fatalf("Set with short TTL failed: %v", err)
	}
	initialTTL := rdb.TTL(ctx, expireKey).Val()
	if initialTTL <= 0 || initialTTL > 2*time.Second {
		t.Fatalf("expected initial TTL <= 2s, got %v", initialTTL)
	}
	// 验证原生 TTL 为正值后，小幅等待 TTL 到期（1s + 100ms）
	time.Sleep(1100 * time.Millisecond)
	got, found, err = cache.Get(ctx, expireKey)
	if err != nil {
		t.Fatalf("Get after expiration failed: %v", err)
	}
	if found || got != nil {
		t.Fatalf("expected key expired, got found=%v, val=%v", found, got)
	}

	// 4. Codec 错误路径与 KeyEncoder
	failCodec := &failingCodec{failMarshal: true}
	errCache := NewKeyCache[string, customItem](rdb, failCodec, nil, time.Minute)
	if err := errCache.Set(ctx, tracker.TrackKey("key_codec_err"), &val1); err == nil {
		t.Fatalf("expected marshal error from failing codec")
	}

	// Unmarshal 错误路径
	corruptKey := tracker.TrackKey("key_corrupt")
	_ = rdb.Set(ctx, corruptKey, "invalid-json-data", time.Minute).Err()
	_, _, err = cache.Get(ctx, corruptKey)
	if err == nil {
		t.Fatalf("expected unmarshal error on corrupt json payload")
	}

	// 自定义 KeyEncoder 保持正常功能
	customEnc := &errorKeyEncoder{prefix: "enc_prefix"}
	encKeyCache := NewKeyCache[int, customItem](rdb, nil, customEnc, time.Minute)
	intKey := 99999
	expectedRedisKey := tracker.TrackKey(customEnc.String(intKey))
	// 手动注册到 tracker 以防遗漏
	_ = tracker.TrackKey(fmt.Sprintf("%s:%d", "enc_prefix", intKey))
	if err := encKeyCache.Set(ctx, intKey, &val1); err != nil {
		t.Fatalf("encKeyCache Set failed: %v", err)
	}
	gotEnc, foundEnc, err := encKeyCache.Get(ctx, intKey)
	if err != nil || !foundEnc || gotEnc == nil || *gotEnc != val1 {
		t.Fatalf("encKeyCache Get failed: got=%v, found=%v, err=%v", gotEnc, foundEnc, err)
	}
	_ = rdb.Del(ctx, expectedRedisKey).Err()
}

func TestIntegration_RedisHashCache(t *testing.T) {
	addr, password := realRedisEnv(t)
	rdb := newTestRedisClient(t, addr, password)
	tracker := newKeyTracker(t, rdb)
	ctx := context.Background()

	hashKey := tracker.TrackKey("hash_main")
	cache := NewHashCache[string, string, string, customStructHash](rdb, nil, nil, 10*time.Minute)

	// 1. Get/Set/Exists/Remove 单字段与 TTL
	if err := cache.Set(ctx, hashKey, "field1", ptr("val1")); err != nil {
		t.Fatalf("HashSet field1 failed: %v", err)
	}
	val, found, err := cache.Get(ctx, hashKey, "field1")
	if err != nil || !found || val == nil || *val != "val1" {
		t.Fatalf("HashGet field1 failed: val=%v, found=%v, err=%v", val, found, err)
	}
	exists, err := cache.Exists(ctx, hashKey, "field1")
	if err != nil || !exists {
		t.Fatalf("HashExists field1 expected true, got %v, err=%v", exists, err)
	}

	// Set nil 等价于 Remove
	if err := cache.Set(ctx, hashKey, "field1", nil); err != nil {
		t.Fatalf("HashSet field1 nil failed: %v", err)
	}
	exists, _ = cache.Exists(ctx, hashKey, "field1")
	if exists {
		t.Fatalf("field1 should be removed after Set nil")
	}

	// 2. SetAll 批量写入
	fields := map[string]*string{
		"fA": ptr("valueA"),
		"fB": ptr("valueB"),
		"fC": ptr("valueC"),
	}
	if err := cache.SetAll(ctx, hashKey, fields, driver.WithTTL(5*time.Minute)); err != nil {
		t.Fatalf("SetAll failed: %v", err)
	}

	allMap, err := cache.GetAll(ctx, hashKey)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(allMap) != 3 || allMap["fA"] != "valueA" || allMap["fB"] != "valueB" || allMap["fC"] != "valueC" {
		t.Fatalf("unexpected GetAll result: %v", allMap)
	}

	// Remove 批量字段
	if err := cache.Remove(ctx, hashKey, "fA", "fB"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	allMap, _ = cache.GetAll(ctx, hashKey)
	if len(allMap) != 1 || allMap["fC"] != "valueC" {
		t.Fatalf("expected only fC left, got %v", allMap)
	}

	// 3. GetAllStruct 与 SetAllStruct
	structKey := tracker.TrackKey("hash_struct")
	sVal := customStructHash{FieldA: "Alpha", FieldB: "Beta"}
	if err := cache.SetAllStruct(ctx, structKey, sVal); err != nil {
		t.Fatalf("SetAllStruct failed: %v", err)
	}

	gotStruct, err := cache.GetAllStruct(ctx, structKey)
	if err != nil {
		t.Fatalf("GetAllStruct failed: %v", err)
	}
	if gotStruct != sVal {
		t.Fatalf("expected %v, got %v", sVal, gotStruct)
	}

	// 4. NX / XX / KeepTTL / NoSlide 条件
	condKey := tracker.TrackKey("hash_cond")
	// NX: 字段不存在时写入
	if err := cache.Set(ctx, condKey, "nx_field", ptr("10"), driver.WithNX()); err != nil {
		t.Fatalf("Set WithNX failed: %v", err)
	}
	// NX: 字段已存在时跳过
	if err := cache.Set(ctx, condKey, "nx_field", ptr("20"), driver.WithNX()); err != nil {
		t.Fatalf("Set WithNX on existing field failed: %v", err)
	}
	v, _, _ := cache.Get(ctx, condKey, "nx_field")
	if v == nil || *v != "10" {
		t.Fatalf("expected nx_field to remain 10, got %v", v)
	}

	// XX: 字段存在时更新
	if err := cache.Set(ctx, condKey, "nx_field", ptr("30"), driver.WithXX(), driver.WithNoSlide()); err != nil {
		t.Fatalf("Set WithXX failed: %v", err)
	}
	v, _, _ = cache.Get(ctx, condKey, "nx_field")
	if v == nil || *v != "30" {
		t.Fatalf("expected nx_field updated to 30, got %v", v)
	}

	// XX: 字段不存在时跳过
	if err := cache.Set(ctx, condKey, "absent_field", ptr("40"), driver.WithXX()); err != nil {
		t.Fatalf("Set WithXX on absent field failed: %v", err)
	}
	ex, _ := cache.Exists(ctx, condKey, "absent_field")
	if ex {
		t.Fatalf("expected absent_field not written with XX")
	}

	// 5. Delete 删除整个 hash
	if err := cache.Delete(ctx, condKey); err != nil {
		t.Fatalf("Delete hash failed: %v", err)
	}
	ex, _ = cache.Exists(ctx, condKey, "nx_field")
	if ex {
		t.Fatalf("expected hash deleted")
	}

	// 6. 原生 TTL 验证
	ttlKey := tracker.TrackKey("hash_ttl")
	if err := cache.Set(ctx, ttlKey, "k1", ptr("v1"), driver.WithTTL(1*time.Second)); err != nil {
		t.Fatalf("Set WithTTL failed: %v", err)
	}
	if ttl := rdb.TTL(ctx, ttlKey).Val(); ttl <= 0 || ttl > 2*time.Second {
		t.Fatalf("expected valid TTL, got %v", ttl)
	}
	time.Sleep(1100 * time.Millisecond)
	v, found, err = cache.Get(ctx, ttlKey, "k1")
	if err != nil || found || v != nil {
		t.Fatalf("expected expired hash field, found=%v, val=%v, err=%v", found, v, err)
	}

	// 7. 错误路径：GetAll 非 string field 类型检查报错
	intFieldCache := NewHashCache[string, int, string, struct{}](rdb, nil, nil, time.Minute)
	_, err = intFieldCache.GetAll(ctx, hashKey)
	if err == nil {
		t.Fatalf("expected error when F is int in GetAll")
	}
}

func TestIntegration_RedisSetCache(t *testing.T) {
	addr, password := realRedisEnv(t)
	rdb := newTestRedisClient(t, addr, password)
	tracker := newKeyTracker(t, rdb)
	ctx := context.Background()

	setKey := tracker.TrackKey("set_main")
	cache := NewSetCache[string, string](rdb, nil, nil, 10*time.Minute)

	// 1. Add / Count / Members / IsMember / Remove
	members := []string{"member_a", "member_b", "member_c"}
	if err := cache.Add(ctx, setKey, members); err != nil {
		t.Fatalf("Add members failed: %v", err)
	}

	count, err := cache.Count(ctx, setKey)
	if err != nil || count != 3 {
		t.Fatalf("Count expected 3, got %d, err=%v", count, err)
	}

	isMem, err := cache.IsMember(ctx, setKey, "member_b")
	if err != nil || !isMem {
		t.Fatalf("IsMember expected true for member_b, got %v, err=%v", isMem, err)
	}

	isNotMem, err := cache.IsMember(ctx, setKey, "member_x")
	if err != nil || isNotMem {
		t.Fatalf("IsMember expected false for member_x, got %v, err=%v", isNotMem, err)
	}

	all, err := cache.Members(ctx, setKey)
	if err != nil || len(all) != 3 {
		t.Fatalf("Members expected len 3, got %v, err=%v", all, err)
	}

	// 移除部分成员
	if err := cache.Remove(ctx, setKey, "member_b", "member_c"); err != nil {
		t.Fatalf("Remove members failed: %v", err)
	}
	count, _ = cache.Count(ctx, setKey)
	if count != 1 {
		t.Fatalf("Count expected 1 after Remove, got %d", count)
	}

	// 2. Options: WithNX / WithXX
	condKey := tracker.TrackKey("set_cond")
	// NX 在 key 不存在时写入
	if err := cache.Add(ctx, condKey, []string{"nx1"}, driver.WithNX()); err != nil {
		t.Fatalf("Add WithNX failed: %v", err)
	}
	// NX 在 key 已存在时跳过
	if err := cache.Add(ctx, condKey, []string{"nx2"}, driver.WithNX()); err != nil {
		t.Fatalf("Add WithNX on existing key failed: %v", err)
	}
	isNx2, _ := cache.IsMember(ctx, condKey, "nx2")
	if isNx2 {
		t.Fatalf("expected nx2 skipped under NX")
	}

	// XX 在 key 存在时写入
	if err := cache.Add(ctx, condKey, []string{"xx1"}, driver.WithXX(), driver.WithNoSlide()); err != nil {
		t.Fatalf("Add WithXX failed: %v", err)
	}
	isXx1, _ := cache.IsMember(ctx, condKey, "xx1")
	if !isXx1 {
		t.Fatalf("expected xx1 added under XX")
	}

	// 3. 原生 TTL 与过期
	ttlKey := tracker.TrackKey("set_ttl")
	if err := cache.Add(ctx, ttlKey, []string{"m1"}, driver.WithTTL(1*time.Second)); err != nil {
		t.Fatalf("Add WithTTL failed: %v", err)
	}
	if ttl := rdb.TTL(ctx, ttlKey).Val(); ttl <= 0 || ttl > 2*time.Second {
		t.Fatalf("expected valid TTL, got %v", ttl)
	}
	time.Sleep(1100 * time.Millisecond)
	count, err = cache.Count(ctx, ttlKey)
	if err != nil || count != 0 {
		t.Fatalf("expected set expired, count=%d, err=%v", count, err)
	}

	// 4. Delete 删除整个 set
	if err := cache.Delete(ctx, condKey); err != nil {
		t.Fatalf("Delete set failed: %v", err)
	}
	cnt, _ := cache.Count(ctx, condKey)
	if cnt != 0 {
		t.Fatalf("expected set count 0 after delete")
	}

	// 5. Codec 错误
	failCodec := &failingCodec{failMarshal: true}
	errCache := NewSetCache[string, string](rdb, failCodec, nil, time.Minute)
	if err := errCache.Add(ctx, tracker.TrackKey("set_err"), []string{"val"}); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestIntegration_RedisListCache(t *testing.T) {
	addr, password := realRedisEnv(t)
	rdb := newTestRedisClient(t, addr, password)
	tracker := newKeyTracker(t, rdb)
	ctx := context.Background()

	listKey := tracker.TrackKey("list_main")
	cache := NewListCache[string, string](rdb, nil, nil, 10*time.Minute)

	// 1. Append / Prepend / Range / Index / Len
	// Append 追加: "b", "c" -> [b, c]
	if err := cache.Append(ctx, listKey, []string{"b", "c"}); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	// Prepend 插入头部: "a" -> [a, b, c]
	if err := cache.Prepend(ctx, listKey, []string{"a"}); err != nil {
		t.Fatalf("Prepend failed: %v", err)
	}

	length, err := cache.Len(ctx, listKey)
	if err != nil || length != 3 {
		t.Fatalf("Len expected 3, got %d, err=%v", length, err)
	}

	// Range 全部
	elems, err := cache.Range(ctx, listKey, 0, -1)
	if err != nil || len(elems) != 3 {
		t.Fatalf("Range expected 3 items, got %v, err=%v", elems, err)
	}
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(elems, expected) {
		t.Fatalf("Range expected %v, got %v", expected, elems)
	}

	// Index 索引访问（含负数）
	idx0, err := cache.Index(ctx, listKey, 0)
	if err != nil || idx0 == nil || *idx0 != "a" {
		t.Fatalf("Index(0) expected 'a', got %v, err=%v", idx0, err)
	}
	idxLast, err := cache.Index(ctx, listKey, -1)
	if err != nil || idxLast == nil || *idxLast != "c" {
		t.Fatalf("Index(-1) expected 'c', got %v, err=%v", idxLast, err)
	}
	idxOutOfBounds, err := cache.Index(ctx, listKey, 100)
	if err != nil || idxOutOfBounds != nil {
		t.Fatalf("Index(100) expected nil, nil, got %v, err=%v", idxOutOfBounds, err)
	}

	// 2. Pop / PopLast
	popHead, err := cache.Pop(ctx, listKey)
	if err != nil || popHead == nil || *popHead != "a" {
		t.Fatalf("Pop expected 'a', got %v, err=%v", popHead, err)
	}
	popTail, err := cache.PopLast(ctx, listKey)
	if err != nil || popTail == nil || *popTail != "c" {
		t.Fatalf("PopLast expected 'c', got %v, err=%v", popTail, err)
	}
	length, _ = cache.Len(ctx, listKey)
	if length != 1 {
		t.Fatalf("Len expected 1 after pop head and tail, got %d", length)
	}

	// 3. Remove 元素
	_ = cache.Append(ctx, listKey, []string{"b", "x", "b"})
	// 目前列表为 [b, b, x, b]
	if err := cache.Remove(ctx, listKey, 2, "b"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	remaining, _ := cache.Range(ctx, listKey, 0, -1)
	// 移除了前2个 "b"，剩下 [x, b]
	if !reflect.DeepEqual(remaining, []string{"x", "b"}) {
		t.Fatalf("expected [x, b] after Remove, got %v", remaining)
	}

	// 4. Options: WithNX / WithXX
	condKey := tracker.TrackKey("list_cond")
	// NX: key 不存在时写入
	if err := cache.Append(ctx, condKey, []string{"nx1"}, driver.WithNX()); err != nil {
		t.Fatalf("Append WithNX failed: %v", err)
	}
	// NX: key 存在时跳过
	if err := cache.Append(ctx, condKey, []string{"nx2"}, driver.WithNX()); err != nil {
		t.Fatalf("Append WithNX existing failed: %v", err)
	}
	l, _ := cache.Len(ctx, condKey)
	if l != 1 {
		t.Fatalf("expected len 1, got %d", l)
	}

	// XX: key 存在时写入
	if err := cache.Prepend(ctx, condKey, []string{"xx1"}, driver.WithXX(), driver.WithNoSlide()); err != nil {
		t.Fatalf("Prepend WithXX failed: %v", err)
	}
	l, _ = cache.Len(ctx, condKey)
	if l != 2 {
		t.Fatalf("expected len 2 after Prepend WithXX, got %d", l)
	}

	// 5. 原生 TTL 与过期
	ttlKey := tracker.TrackKey("list_ttl")
	if err := cache.Append(ctx, ttlKey, []string{"item"}, driver.WithTTL(1*time.Second)); err != nil {
		t.Fatalf("Append WithTTL failed: %v", err)
	}
	if ttl := rdb.TTL(ctx, ttlKey).Val(); ttl <= 0 || ttl > 2*time.Second {
		t.Fatalf("expected valid TTL, got %v", ttl)
	}
	time.Sleep(1100 * time.Millisecond)
	l, err = cache.Len(ctx, ttlKey)
	if err != nil || l != 0 {
		t.Fatalf("expected list expired, len=%d, err=%v", l, err)
	}

	// 6. Delete
	if err := cache.Delete(ctx, condKey); err != nil {
		t.Fatalf("Delete list failed: %v", err)
	}
	l, _ = cache.Len(ctx, condKey)
	if l != 0 {
		t.Fatalf("expected list deleted, len=%d", l)
	}
}

func TestIntegration_RealRedisClientClosePingFails(t *testing.T) {
	// 验证对真实 Redis 客户端关闭后，Ping 真实失败（无 mock/miniredis）
	addr, password := realRedisEnv(t)
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		t.Fatalf("initial ping failed: %v", err)
	}

	// 关闭客户端
	if err := rdb.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// 关 Client 真实 Ping 必须失败
	if err := rdb.Ping(ctx).Err(); err == nil {
		t.Fatalf("expected ping to fail after client is closed")
	}
}

func ptr[T any](v T) *T {
	return &v
}
