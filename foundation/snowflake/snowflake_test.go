package snowflake

import (
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test_SnowflakeID_Generate(t *testing.T) {
	node, err := NewNode(1, 1)
	if err != nil {
		t.Fatalf("创建雪花节点失败: %v", err)
	}

	id := node.MustGenerate()
	if id.IsZero() {
		t.Error("生成的雪花 ID 不应为零值")
	}

	if id.DatacenterID() != 1 {
		t.Errorf("DatacenterID = %d; want 1", id.DatacenterID())
	}
	if id.NodeID() != 1 {
		t.Errorf("NodeID = %d; want 1", id.NodeID())
	}
	if id.Gene() != GeneDefault {
		t.Errorf("Gene = %d; want %d (GeneDefault)", id.Gene(), GeneDefault)
	}

	ts := id.Timestamp()
	now := time.Now()
	if ts.After(now.Add(time.Second)) || ts.Before(now.Add(-time.Second)) {
		t.Errorf("Timestamp = %v; should be close to %v", ts, now)
	}
}

func Test_SnowflakeID_GenerateWithGene(t *testing.T) {
	node, err := NewNode(1, 1)
	if err != nil {
		t.Fatalf("创建雪花节点失败: %v", err)
	}

	id := node.MustGenerate(GeneOrder)
	if id.IsZero() {
		t.Error("生成的基因雪花 ID 不应为零值")
	}

	if id.Gene() != GeneOrder {
		t.Errorf("Gene = %d; want %d (GeneOrder)", id.Gene(), GeneOrder)
	}
	if id.DatacenterID() != 1 {
		t.Errorf("DatacenterID = %d; want 1", id.DatacenterID())
	}
	if id.NodeID() != 1 {
		t.Errorf("NodeID = %d; want 1", id.NodeID())
	}
}

func Test_SnowflakeID_Uniqueness(t *testing.T) {
	node, _ := NewNode(0, 0)
	const count = 10000
	ids := make(map[int64]bool, count)

	for i := 0; i < count; i++ {
		id := node.MustGenerate()
		if ids[id.Int64()] {
			t.Fatalf("发现重复 ID: %s", id.String())
		}
		ids[id.Int64()] = true
	}
}

func Test_SnowflakeID_Concurrent(t *testing.T) {
	node, _ := NewNode(0, 0)
	const goroutines = 10
	const idsPerGoroutine = 1000

	var wg sync.WaitGroup
	idsChan := make(chan int64, goroutines*idsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id := node.MustGenerate()
				idsChan <- id.Int64()
			}
		}()
	}

	wg.Wait()
	close(idsChan)

	ids := make(map[int64]bool)
	for id := range idsChan {
		if ids[id] {
			t.Fatalf("并发测试发现重复 ID: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != goroutines*idsPerGoroutine {
		t.Errorf("生成的 ID 数量 = %d; want %d", len(ids), goroutines*idsPerGoroutine)
	}
}

func Test_SnowflakeID_BitLayout(t *testing.T) {
	node, err := NewNode(3, 5)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	fixedTime := int64(1700000000000)
	node.clockFunc = func() int64 {
		return fixedTime
	}

	id, err := node.Generate(Gene(42))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if id.DatacenterID() != 3 {
		t.Errorf("DatacenterID = %d; want 3", id.DatacenterID())
	}
	if id.NodeID() != 5 {
		t.Errorf("NodeID = %d; want 5", id.NodeID())
	}
	if id.Gene() != Gene(42) {
		t.Errorf("Gene = %d; want 42", id.Gene())
	}
	if id.Sequence() != 0 {
		t.Errorf("Sequence = %d; want 0", id.Sequence())
	}
	if id.Timestamp().UnixMilli() != fixedTime {
		t.Errorf("Timestamp ms = %d; want %d", id.Timestamp().UnixMilli(), fixedTime)
	}
}

func Test_NewNode_Boundaries(t *testing.T) {
	for dc := int64(0); dc <= 7; dc++ {
		for nid := int64(0); nid <= 7; nid++ {
			n, err := NewNode(dc, nid)
			if err != nil {
				t.Fatalf("NewNode(%d, %d) failed: %v", dc, nid, err)
			}
			if n.DatacenterID() != dc || n.NodeID() != nid {
				t.Fatalf("Node getters mismatch")
			}
		}
	}

	badCases := [][2]int64{
		{-1, 0}, {8, 0}, {0, -1}, {0, 8}, {-1, -1}, {99, 99},
	}
	for _, bc := range badCases {
		if _, err := NewNode(bc[0], bc[1]); err == nil {
			t.Errorf("NewNode(%d, %d) should return error", bc[0], bc[1])
		}
	}
}

func Test_NilNode_Generate(t *testing.T) {
	var n *Node
	_, err := n.Generate()
	if !errors.Is(err, ErrNilNode) {
		t.Errorf("expected ErrNilNode, got %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("MustGenerate on nil node should panic")
		}
	}()
	_ = n.MustGenerate()
}

func Test_SequenceExhaustion_AdvancesTime(t *testing.T) {
	node, _ := NewNode(1, 1)
	var current int64 = 1700000000000
	node.clockFunc = func() int64 {
		return atomic.LoadInt64(&current)
	}

	// 生成 1024 个同一毫秒内
	for i := 0; i < 1024; i++ {
		id, err := node.Generate()
		if err != nil {
			t.Fatalf("Generate failed at %d: %v", i, err)
		}
		if id.Sequence() != int64(i) {
			t.Fatalf("Sequence = %d, want %d", id.Sequence(), i)
		}
	}

	// 模拟等待下一毫秒
	go func() {
		time.Sleep(10 * time.Millisecond)
		atomic.StoreInt64(&current, 1700000000001)
	}()

	nextID, err := node.Generate()
	if err != nil {
		t.Fatalf("Generate failed on exhausted sequence: %v", err)
	}
	if nextID.Sequence() != 0 {
		t.Errorf("Next sequence should reset to 0, got %d", nextID.Sequence())
	}
	if nextID.Timestamp().UnixMilli() != 1700000000001 {
		t.Errorf("Timestamp ms = %d, want 1700000000001", nextID.Timestamp().UnixMilli())
	}
}

func Test_ClockBackwards_Error(t *testing.T) {
	node, _ := NewNode(1, 1)
	var current int64 = 1700000001000
	node.clockFunc = func() int64 {
		return current
	}

	_, err := node.Generate()
	if err != nil {
		t.Fatalf("first generate failed: %v", err)
	}

	// 时钟回退
	current = 1700000000000
	_, err = node.Generate()
	if !errors.Is(err, ErrClockBackwards) {
		t.Fatalf("expected ErrClockBackwards, got %v", err)
	}
}

func Test_SnowflakeID_JSON(t *testing.T) {
	id := SnowflakeID(1234567890123456789)

	data, err := json.Marshal(&id)
	if err != nil {
		t.Fatalf("JSON Marshal 失败: %v", err)
	}
	expected := `"1234567890123456789"`
	if string(data) != expected {
		t.Errorf("JSON = %s; want %s", string(data), expected)
	}

	var id2 SnowflakeID
	if err := json.Unmarshal(data, &id2); err != nil {
		t.Fatalf("JSON Unmarshal 失败: %v", err)
	}
	if id2 != id {
		t.Errorf("反序列化结果 = %d; want %d", id2, id)
	}

	// 兼容数字
	var id3 SnowflakeID
	if err := json.Unmarshal([]byte(`123456789`), &id3); err != nil {
		t.Fatalf("JSON Unmarshal 数字格式失败: %v", err)
	}
	if id3 != 123456789 {
		t.Errorf("数字格式反序列化结果 = %d; want 123456789", id3)
	}

	// nil 接收器不崩溃
	var nilPtr *SnowflakeID
	if err := nilPtr.UnmarshalJSON(data); !errors.Is(err, ErrNilReceiver) {
		t.Errorf("expected ErrNilReceiver, got %v", err)
	}

	// 非法值不污染
	idKeep := SnowflakeID(999)
	if err := idKeep.UnmarshalJSON([]byte(`"not-a-number"`)); err == nil {
		t.Errorf("UnmarshalJSON should fail for invalid string")
	}
	if idKeep != 999 {
		t.Errorf("idKeep should remain 999, got %d", idKeep)
	}
}

func Test_SnowflakeID_Binary(t *testing.T) {
	var _ encoding.BinaryMarshaler = SnowflakeID(0)
	var _ encoding.BinaryUnmarshaler = (*SnowflakeID)(nil)

	id := SnowflakeID(9876543210)
	data, err := id.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	var parsed SnowflakeID
	if err := parsed.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}
	if parsed != id {
		t.Errorf("got %d, want %d", parsed, id)
	}

	var nilPtr *SnowflakeID
	if err := nilPtr.UnmarshalBinary(data); !errors.Is(err, ErrNilReceiver) {
		t.Errorf("expected ErrNilReceiver, got %v", err)
	}

	target := SnowflakeID(888)
	if err := target.UnmarshalBinary([]byte("invalid")); err == nil {
		t.Errorf("expected error on invalid binary")
	}
	if target != 888 {
		t.Errorf("target should not be mutated on error, got %d", target)
	}
}

func Test_SnowflakeID_SQL_Scan_Value(t *testing.T) {
	var _ driver.Valuer = SnowflakeID(0)

	id := SnowflakeID(1122334455)
	val, err := id.Value()
	if err != nil || val != int64(1122334455) {
		t.Errorf("Value() = %v, %v; want int64", val, err)
	}

	// Scan(nil) 置 0
	var scanned SnowflakeID = 123
	if err := scanned.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) failed: %v", err)
	}
	if scanned != 0 {
		t.Errorf("Scan(nil) should set to 0, got %d", scanned)
	}

	// Scan int64
	if err := scanned.Scan(int64(456)); err != nil || scanned != 456 {
		t.Errorf("Scan(int64) = %d, err %v", scanned, err)
	}

	// Scan []byte
	if err := scanned.Scan([]byte("789")); err != nil || scanned != 789 {
		t.Errorf("Scan([]byte) = %d, err %v", scanned, err)
	}

	// Scan string
	if err := scanned.Scan("101112"); err != nil || scanned != 101112 {
		t.Errorf("Scan(string) = %d, err %v", scanned, err)
	}

	// nil receiver
	var nilScan *SnowflakeID
	if err := nilScan.Scan(int64(1)); !errors.Is(err, ErrNilReceiver) {
		t.Errorf("expected ErrNilReceiver on nil receiver, got %v", err)
	}

	// 错误输入不污染
	preserve := SnowflakeID(777)
	if err := preserve.Scan("bad-number"); err == nil {
		t.Errorf("expected error on bad string")
	}
	if preserve != 777 {
		t.Errorf("preserve should remain 777, got %d", preserve)
	}
}

func Benchmark_SnowflakeID_Generate(b *testing.B) {
	node, _ := NewNode(0, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.MustGenerate()
	}
}

func Test_SnowflakeID_SequenceOverflowClockBackwardsRollback(t *testing.T) {
	node, err := NewNode(0, 0)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}

	const fixedT = int64(1700000000000)
	currentTime := fixedT
	node.clockFunc = func() int64 {
		return atomic.LoadInt64(&currentTime)
	}

	history := make(map[SnowflakeID]struct{}, 1024)
	for i := 0; i < 1024; i++ {
		id, genErr := node.Generate()
		if genErr != nil {
			t.Fatalf("unexpected error generating at fixedT: %v", genErr)
		}
		if _, exists := history[id]; exists {
			t.Fatalf("duplicate ID generated within initial 1024: %d", id)
		}
		history[id] = struct{}{}
	}

	// 第 1025 次调用触发序列号溢出进入等待下一毫秒。
	// 通过控制时钟返回序列：第 1 次调 currentTime() 仍是 fixedT（触发 now == n.lastTime 分支），
	// 在循环等待下一毫秒时返回 fixedT - 1（触发回退错误）。
	clockSeq := []int64{fixedT, fixedT - 1}
	var callCount int64
	node.clockFunc = func() int64 {
		idx := atomic.AddInt64(&callCount, 1) - 1
		if int(idx) < len(clockSeq) {
			return clockSeq[idx]
		}
		return fixedT - 1
	}

	_, err = node.Generate()
	if !errors.Is(err, ErrClockBackwards) {
		t.Fatalf("expected ErrClockBackwards, got: %v", err)
	}

	// 接下来受控时钟顺序给出 fixedT、fixedT+1
	seqAfter := []int64{fixedT, fixedT + 1}
	var afterCount int64
	node.clockFunc = func() int64 {
		idx := atomic.AddInt64(&afterCount, 1) - 1
		if int(idx) < len(seqAfter) {
			return seqAfter[idx]
		}
		return seqAfter[len(seqAfter)-1]
	}

	// 在 fixedT 再次调用：若节点未受污染，序列号应已在 fixedT 耗尽或直接成功
	// 如果在 fixedT 尝试生成，若能生成必须不能与 history 重复！
	// 原实现如果污染了 sequence 为 0，在 fixedT 再次生成就会发 sequence 1，导致 ID 重复！
	idAtT, err := node.Generate()
	if err == nil {
		if _, exists := history[idAtT]; exists {
			t.Fatalf("duplicate ID generated after clock returned to fixedT: %d", idAtT)
		}
		history[idAtT] = struct{}{}
	} else if !errors.Is(err, ErrClockBackwards) {
		t.Fatalf("unexpected error at fixedT: %v", err)
	}

	// 在 fixedT+1 调用，确保后续成功 ID 不在历史集合
	idNext, err := node.Generate()
	if err != nil {
		t.Fatalf("unexpected error at fixedT+1: %v", err)
	}
	if _, exists := history[idNext]; exists {
		t.Fatalf("duplicate ID generated at fixedT+1: %d", idNext)
	}
}
