package test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
)

// Test_SnowflakeID_Generate 测试雪花 ID 生成（普通 ID，Gene=0）
func Test_SnowflakeID_Generate(t *testing.T) {
	node, err := xSnowflake.NewNode(1, 1)
	if err != nil {
		t.Fatalf("创建雪花节点失败: %v", err)
	}

	id := node.MustGenerate()
	if id.IsZero() {
		t.Error("生成的雪花 ID 不应为零值")
	}

	// 验证 ID 组件提取
	if id.DatacenterID() != 1 {
		t.Errorf("DatacenterID = %d; want 1", id.DatacenterID())
	}
	if id.NodeID() != 1 {
		t.Errorf("NodeID = %d; want 1", id.NodeID())
	}
	// 默认 Gene 应为 0
	if id.Gene() != xSnowflake.GeneDefault {
		t.Errorf("Gene = %d; want %d (GeneDefault)", id.Gene(), xSnowflake.GeneDefault)
	}

	// 验证时间戳在合理范围内
	ts := id.Timestamp()
	now := time.Now()
	if ts.After(now) || ts.Before(now.Add(-time.Second)) {
		t.Errorf("Timestamp = %v; should be close to %v", ts, now)
	}
}

// Test_SnowflakeID_GenerateWithGene 测试带基因的雪花 ID 生成
func Test_SnowflakeID_GenerateWithGene(t *testing.T) {
	node, err := xSnowflake.NewNode(1, 1)
	if err != nil {
		t.Fatalf("创建雪花节点失败: %v", err)
	}

	id := node.MustGenerate(xSnowflake.GeneOrder)
	if id.IsZero() {
		t.Error("生成的基因雪花 ID 不应为零值")
	}

	// 验证基因提取
	if id.Gene() != xSnowflake.GeneOrder {
		t.Errorf("Gene = %d; want %d (GeneOrder)", id.Gene(), xSnowflake.GeneOrder)
	}

	// 验证数据中心和节点
	if id.DatacenterID() != 1 {
		t.Errorf("DatacenterID = %d; want 1", id.DatacenterID())
	}
	if id.NodeID() != 1 {
		t.Errorf("NodeID = %d; want 1", id.NodeID())
	}
}

// Test_SnowflakeID_Uniqueness 测试雪花 ID 唯一性
func Test_SnowflakeID_Uniqueness(t *testing.T) {
	node, _ := xSnowflake.NewNode(0, 0)

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

// Test_SnowflakeID_Concurrent 测试雪花 ID 并发安全性
func Test_SnowflakeID_Concurrent(t *testing.T) {
	node, _ := xSnowflake.NewNode(0, 0)

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

	// 检查唯一性
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

// Test_SnowflakeID_JSON 测试雪花 ID JSON 序列化
func Test_SnowflakeID_JSON(t *testing.T) {
	id := xSnowflake.SnowflakeID(1234567890123456789)

	// 序列化（使用指针以调用 MarshalJSON 方法）
	data, err := json.Marshal(&id)
	if err != nil {
		t.Fatalf("JSON Marshal 失败: %v", err)
	}

	// 应该序列化为字符串
	expected := `"1234567890123456789"`
	if string(data) != expected {
		t.Errorf("JSON = %s; want %s", string(data), expected)
	}

	// 反序列化字符串格式
	var id2 xSnowflake.SnowflakeID
	if err := json.Unmarshal(data, &id2); err != nil {
		t.Fatalf("JSON Unmarshal 失败: %v", err)
	}
	if id2 != id {
		t.Errorf("反序列化结果 = %d; want %d", id2, id)
	}

	// 反序列化数字格式（兼容性测试）
	var id3 xSnowflake.SnowflakeID
	if err := json.Unmarshal([]byte(`123456789`), &id3); err != nil {
		t.Fatalf("JSON Unmarshal 数字格式失败: %v", err)
	}
	if id3 != 123456789 {
		t.Errorf("数字格式反序列化结果 = %d; want 123456789", id3)
	}
}

// Test_SnowflakeID_String 测试雪花 ID 字符串转换
func Test_SnowflakeID_String(t *testing.T) {
	id := xSnowflake.SnowflakeID(1234567890)
	if id.String() != "1234567890" {
		t.Errorf("String() = %s; want 1234567890", id.String())
	}
}

// Test_SnowflakeID_GeneExtraction 测试基因提取
func Test_SnowflakeID_GeneExtraction(t *testing.T) {
	node, _ := xSnowflake.NewNode(0, 0)

	testCases := []xSnowflake.Gene{
		xSnowflake.GeneDefault,
		xSnowflake.GeneUser,
		xSnowflake.GeneOrder,
		xSnowflake.GeneProduct,
		xSnowflake.Gene(63), // 最大值
	}

	for _, gene := range testCases {
		id := node.MustGenerate(gene)
		extracted := id.Gene()
		if extracted != gene {
			t.Errorf("Gene extraction failed: got %d, want %d", extracted, gene)
		}
	}
}

// Test_Gene_Methods 测试基因类型方法
func Test_Gene_Methods(t *testing.T) {
	// 系统级基因
	if !xSnowflake.GeneUser.IsSystem() {
		t.Error("GeneUser 应该是系统级基因")
	}
	if xSnowflake.GeneUser.IsBusiness() {
		t.Error("GeneUser 不应该是业务级基因")
	}

	// 业务级基因
	if xSnowflake.GeneOrder.IsSystem() {
		t.Error("GeneOrder 不应该是系统级基因")
	}
	if !xSnowflake.GeneOrder.IsBusiness() {
		t.Error("GeneOrder 应该是业务级基因")
	}

	// 有效性检查
	if !xSnowflake.Gene(0).IsValid() {
		t.Error("Gene(0) 应该是有效的")
	}
	if !xSnowflake.Gene(63).IsValid() {
		t.Error("Gene(63) 应该是有效的")
	}
	if xSnowflake.Gene(64).IsValid() {
		t.Error("Gene(64) 不应该是有效的")
	}
	if xSnowflake.Gene(-1).IsValid() {
		t.Error("Gene(-1) 不应该是有效的")
	}
}

// Test_Gene_String 测试基因类型字符串
func Test_Gene_String(t *testing.T) {
	if xSnowflake.GeneUser.String() != "User" {
		t.Errorf("GeneUser.String() = %s; want User", xSnowflake.GeneUser.String())
	}
	if xSnowflake.GeneOrder.String() != "Order" {
		t.Errorf("GeneOrder.String() = %s; want Order", xSnowflake.GeneOrder.String())
	}

	// 自定义基因
	custom := xSnowflake.Gene(50)
	if custom.String() != "Custom(50)" {
		t.Errorf("Gene(50).String() = %s; want Custom(50)", custom.String())
	}
}

// Test_NewNode_InvalidParams 测试无效参数
func Test_NewNode_InvalidParams(t *testing.T) {
	// 无效的数据中心 ID
	_, err := xSnowflake.NewNode(-1, 0)
	if err == nil {
		t.Error("NewNode 应该拒绝负数数据中心 ID")
	}

	_, err = xSnowflake.NewNode(8, 0)
	if err == nil {
		t.Error("NewNode 应该拒绝超出范围的数据中心 ID (max=7)")
	}

	// 无效的节点 ID
	_, err = xSnowflake.NewNode(0, -1)
	if err == nil {
		t.Error("NewNode 应该拒绝负数节点 ID")
	}

	_, err = xSnowflake.NewNode(0, 8)
	if err == nil {
		t.Error("NewNode 应该拒绝超出范围的节点 ID (max=7)")
	}
}

// Test_GlobalFunctions 测试全局便捷函数
func Test_GlobalFunctions(t *testing.T) {
	// 测试 GenerateID（无基因）
	id := xSnowflake.GenerateID()
	if id.IsZero() {
		t.Error("GenerateID 不应返回零值")
	}
	if id.Gene() != xSnowflake.GeneDefault {
		t.Errorf("GenerateID 应生成 Gene=0 的 ID, got %d", id.Gene())
	}

	// 测试 GenerateID（带基因）
	geneID := xSnowflake.GenerateID(xSnowflake.GeneUser)
	if geneID.IsZero() {
		t.Error("GenerateID(GeneUser) 不应返回零值")
	}
	if geneID.Gene() != xSnowflake.GeneUser {
		t.Errorf("基因类型 = %d; want %d", geneID.Gene(), xSnowflake.GeneUser)
	}

	// 测试 GenerateID（带 GeneOrder）
	geneID2 := xSnowflake.GenerateID(xSnowflake.GeneOrder)
	if geneID2.Gene() != xSnowflake.GeneOrder {
		t.Errorf("基因类型 = %d; want %d", geneID2.Gene(), xSnowflake.GeneOrder)
	}
}

// Test_ParseSnowflakeID 测试解析雪花 ID
func Test_ParseSnowflakeID(t *testing.T) {
	original := xSnowflake.GenerateID()

	// 解析有效字符串
	parsed, err := xSnowflake.ParseSnowflakeID(original.String())
	if err != nil {
		t.Fatalf("ParseSnowflakeID 失败: %v", err)
	}
	if parsed != original {
		t.Errorf("解析结果 = %d; want %d", parsed, original)
	}

	// 解析无效字符串
	_, err = xSnowflake.ParseSnowflakeID("invalid")
	if err == nil {
		t.Error("ParseSnowflakeID 应该拒绝无效字符串")
	}
}

// Test_ParseSnowflakeID_WithGene 测试解析带基因的雪花 ID
func Test_ParseSnowflakeID_WithGene(t *testing.T) {
	original := xSnowflake.GenerateID(xSnowflake.GeneProduct)

	// 解析有效字符串
	parsed, err := xSnowflake.ParseSnowflakeID(original.String())
	if err != nil {
		t.Fatalf("ParseSnowflakeID 失败: %v", err)
	}
	if parsed != original {
		t.Errorf("解析结果 = %d; want %d", parsed, original)
	}
	if parsed.Gene() != xSnowflake.GeneProduct {
		t.Errorf("基因类型 = %d; want %d", parsed.Gene(), xSnowflake.GeneProduct)
	}
}

// Benchmark_SnowflakeID_Generate 基准测试：雪花 ID 生成（无基因）
func Benchmark_SnowflakeID_Generate(b *testing.B) {
	node, _ := xSnowflake.NewNode(0, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.MustGenerate()
	}
}

// Benchmark_SnowflakeID_GenerateWithGene 基准测试：雪花 ID 生成（带基因）
func Benchmark_SnowflakeID_GenerateWithGene(b *testing.B) {
	node, _ := xSnowflake.NewNode(0, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.MustGenerate(xSnowflake.GeneOrder)
	}
}
