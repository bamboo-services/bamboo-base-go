package redis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	driver "github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
	goredis "github.com/redis/go-redis/v9"
)

// p5ExpectedKeyCache 等接口用于编译期断言 Redis 实现的方法集已切换为强类型错误。
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

// p5FakeRedis 是最小 RESP 服务器，只实现回归所需的命令语义。
//
// 它让「键不存在」「SET NX 条件不满足」等 goredis.Nil 路径可以在没有外部
// Redis 服务的情况下被真实调用链覆盖。
type p5FakeRedis struct {
	ln     net.Listener
	mu     sync.Mutex
	values map[string]string
	closed chan struct{}
}

// newP5FakeRedis 启动本地假 Redis 并在测试结束时关闭。
func newP5FakeRedis(t *testing.T) *p5FakeRedis {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("启动本地假 Redis 失败: %v", err)
	}
	srv := &p5FakeRedis{ln: ln, values: make(map[string]string), closed: make(chan struct{})}
	go srv.serve()
	t.Cleanup(func() {
		close(srv.closed)
		_ = ln.Close()
	})
	return srv
}

// addr 返回假 Redis 的监听地址。
func (s *p5FakeRedis) addr() string { return s.ln.Addr().String() }

// serve 接受连接并逐条应答 RESP 命令。
func (s *p5FakeRedis) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
			}
			return
		}
		go s.handle(conn)
	}
}

// handle 处理单条连接上的命令流。
func (s *p5FakeRedis) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	reader := bufio.NewReader(conn)
	for {
		args, err := p5ReadCommand(reader)
		if err != nil {
			return
		}
		if _, err := conn.Write([]byte(s.reply(args))); err != nil {
			return
		}
	}
}

// reply 根据命令生成 RESP 应答。
func (s *p5FakeRedis) reply(args []string) string {
	if len(args) == 0 {
		return "+OK\r\n"
	}
	switch strings.ToLower(args[0]) {
	case "hello":
		// RESP3 HELLO 应答：7 组键值对，客户端据此进入 RESP3 模式
		return "%7\r\n" +
			"$6\r\nserver\r\n$5\r\nredis\r\n" +
			"$7\r\nversion\r\n$5\r\n7.0.0\r\n" +
			"$5\r\nproto\r\n:3\r\n" +
			"$2\r\nid\r\n:1\r\n" +
			"$4\r\nmode\r\n$10\r\nstandalone\r\n" +
			"$4\r\nrole\r\n$6\r\nmaster\r\n" +
			"$7\r\nmodules\r\n*0\r\n"
	case "ping":
		return "+PONG\r\n"
	case "get":
		s.mu.Lock()
		value, ok := s.values[args[1]]
		s.mu.Unlock()
		if !ok {
			return "$-1\r\n"
		}
		return fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
	case "set":
		if len(args) < 3 {
			return "+OK\r\n"
		}
		key, value := args[1], args[2]
		conditional := false
		for _, arg := range args[3:] {
			if strings.EqualFold(arg, "NX") || strings.EqualFold(arg, "XX") {
				conditional = true
			}
		}
		s.mu.Lock()
		_, exists := s.values[key]
		if conditional && exists {
			s.mu.Unlock()
			// NX/XX 条件不满足：Redis 返回 nil bulk string，客户端读到 goredis.Nil
			return "$-1\r\n"
		}
		s.values[key] = value
		s.mu.Unlock()
		return "+OK\r\n"
	case "del":
		s.mu.Lock()
		delete(s.values, args[1])
		s.mu.Unlock()
		return ":1\r\n"
	case "exists":
		s.mu.Lock()
		_, ok := s.values[args[1]]
		s.mu.Unlock()
		if ok {
			return ":1\r\n"
		}
		return ":0\r\n"
	default:
		return "+OK\r\n"
	}
}

// p5ReadCommand 读取一条 RESP 数组命令。
func p5ReadCommand(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return nil, errors.New("empty command")
	}
	if line[0] != '*' {
		return nil, fmt.Errorf("unexpected command: %q", line)
	}
	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		header, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		header = strings.TrimRight(header, "\r\n")
		if header == "" || header[0] != '$' {
			return nil, fmt.Errorf("unexpected header: %q", header)
		}
		size, err := strconv.Atoi(header[1:])
		if err != nil {
			return nil, err
		}
		payload := make([]byte, size+2)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, err
		}
		args = append(args, string(payload[:size]))
	}
	return args, nil
}

// p5TestClient 构造连接本地假 Redis 的客户端。
func p5TestClient(t *testing.T, addr string) *goredis.Client {
	t.Helper()
	rdb := goredis.NewClient(&goredis.Options{
		Addr:            addr,
		DisableIdentity: true,
		MaxRetries:      -1,
		DialTimeout:     time.Second,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
	})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

// TestP5RedisNilIsNotFailure 断言 goredis.Nil 不是失败：未命中返回 (nil, false, nil)。
func TestP5RedisNilIsNotFailure(t *testing.T) {
	srv := newP5FakeRedis(t)
	rdb := p5TestClient(t, srv.addr())
	kc := NewKeyCache[string, string](rdb, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()

	value, ok, err := kc.Get(ctx, "absent")
	if err != nil {
		t.Fatalf("期望未命中不是失败，实际: %v", err)
	}
	if !apperr.IsNil(err) {
		t.Fatal("期望未命中返回空指针（真 nil）")
	}
	if ok || value != nil {
		t.Fatalf("期望未命中返回 (nil, false)，实际: (%v, %v)", value, ok)
	}

	exists, err := kc.Exists(ctx, "absent")
	if err != nil {
		t.Fatalf("期望 Exists 未命中不是失败，实际: %v", err)
	}
	if exists {
		t.Fatal("期望 Exists 未命中返回 false")
	}

	if err := kc.Delete(ctx, "absent"); err != nil {
		t.Fatalf("期望删除不存在的键幂等成功，实际: %v", err)
	}
}

// TestP5RedisSetNXConditionMissIsNotFailure 断言 SET NX 条件不满足（goredis.Nil）不是失败。
func TestP5RedisSetNXConditionMissIsNotFailure(t *testing.T) {
	srv := newP5FakeRedis(t)
	rdb := p5TestClient(t, srv.addr())
	kc := NewKeyCache[string, string](rdb, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()

	first := "v1"
	if err := kc.Set(ctx, "key", &first); err != nil {
		t.Fatalf("首次写入失败: %v", err)
	}
	second := "v2"
	if err := kc.Set(ctx, "key", &second, driver.WithNX()); err != nil {
		t.Fatalf("期望 NX 条件不满足返回 nil 错误，实际: %v", err)
	}
	value, ok, err := kc.Get(ctx, "key")
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if !ok || value == nil || *value != "v1" {
		t.Fatalf("期望 NX 写入被跳过且原值保留，实际: (%v, %v)", value, ok)
	}
}

// TestP5RedisContextCancelledKeepsControlCode 断言 ctx 取消归一为 49900 且保留原生哨兵。
func TestP5RedisContextCancelledKeepsControlCode(t *testing.T) {
	srv := newP5FakeRedis(t)
	rdb := p5TestClient(t, srv.addr())
	kc := NewKeyCache[string, string](rdb, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := kc.Get(ctx, "key")
	if err == nil {
		t.Fatal("期望 ctx 取消返回失败，实际得到 nil")
	}
	if err.Code.Code != control.Canceled.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, control.Canceled.Code)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("期望保留原生 context.Canceled 根因，实际: %v", err)
	}
}

// TestP5RedisDeadlineExceededKeepsControlCode 断言内部超时归一为 50401 且保留原生哨兵。
func TestP5RedisDeadlineExceededKeepsControlCode(t *testing.T) {
	srv := newP5FakeRedis(t)
	rdb := p5TestClient(t, srv.addr())
	kc := NewKeyCache[string, string](rdb, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, _, err := kc.Get(ctx, "key")
	if err == nil {
		t.Fatal("期望超时返回失败，实际得到 nil")
	}
	if err.Code.Code != control.DeadlineExceeded.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, control.DeadlineExceeded.Code)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("期望保留原生 context.DeadlineExceeded 根因，实际: %v", err)
	}
}

// TestP5RedisUnreachableUsesOperationFailed 断言 Redis 运行故障归一为 50039 并保留原生根因。
func TestP5RedisUnreachableUsesOperationFailed(t *testing.T) {
	rdb := p5TestClient(t, "127.0.0.1:1")
	kc := NewKeyCache[string, string](rdb, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)

	_, _, err := kc.Get(context.Background(), "key")
	if err == nil {
		t.Fatal("期望 Redis 不可达时返回失败，实际得到 nil")
	}
	if err.Code.Code != driver.OperationFailed.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, driver.OperationFailed.Code)
	}
	if err.Err == nil {
		t.Fatal("期望保留 Redis 原生根因，实际根因为空")
	}
}

// TestP5RedisNilReceiverDoesNotPanic 断言 nil 接收器与未注入客户端不触发 panic。
func TestP5RedisNilReceiverDoesNotPanic(t *testing.T) {
	ctx := context.Background()

	var kc *KeyCache[string, string]
	if _, _, err := kc.Get(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if err := kc.Set(ctx, "k", nil); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if _, err := kc.Exists(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if err := kc.Delete(ctx, "k"); err == nil {
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

	var lc *ListCache[string, string]
	if _, err := lc.Pop(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
	if _, err := lc.Len(ctx, "k"); err == nil {
		t.Fatal("期望 nil 接收器返回错误，实际得到 nil")
	}
}

// TestP5RedisHashGetAllContractFailureUsesOperationFailed 断言 F 非 string 时不静默返回空 map。
func TestP5RedisHashGetAllContractFailureUsesOperationFailed(t *testing.T) {
	srv := newP5FakeRedis(t)
	rdb := p5TestClient(t, srv.addr())
	hc := NewHashCache[string, int, string, struct{}](rdb, driver.JSONCodec{}, driver.DefaultKeyEncoder{}, 0)

	all, err := hc.GetAll(context.Background(), "key")
	if err == nil {
		t.Fatal("期望 F 非 string 时返回失败，实际得到 nil")
	}
	if err.Code.Code != driver.OperationFailed.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, driver.OperationFailed.Code)
	}
	if all != nil {
		t.Fatalf("期望失败时不返回空 map，实际: %+v", all)
	}
}
