package test

import (
	"testing"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
)

// Test_GeneCalc_Hash 测试基于 SnowflakeID 计算基因
func Test_GeneCalc_Hash(t *testing.T) {
	calc := xSnowflake.GeneCalc{}

	// 测试不同的 ID 产生不同的基因
	id1 := xSnowflake.SnowflakeID(1234567890)
	id2 := xSnowflake.SnowflakeID(9876543210)

	gene1 := calc.Hash(id1)
	gene2 := calc.Hash(id2)

	if gene1 == gene2 {
		t.Errorf("不同 ID 应产生不同基因: gene1=%d, gene2=%d", gene1, gene2)
	}

	// 验证基因值在有效范围内
	if !gene1.IsValid() {
		t.Errorf("基因值应在 0-63 范围内: got %d", gene1)
	}
	if !gene2.IsValid() {
		t.Errorf("基因值应在 0-63 范围内: got %d", gene2)
	}

	// 验证同一 ID 产生相同基因
	gene1Again := calc.Hash(id1)
	if gene1 != gene1Again {
		t.Errorf("同一 ID 应产生相同基因: first=%d, second=%d", gene1, gene1Again)
	}
}

// Test_GeneCalc_HashMulti 测试基于多个 ID 计算组合基因
func Test_GeneCalc_HashMulti(t *testing.T) {
	calc := xSnowflake.GeneCalc{}

	id1 := xSnowflake.SnowflakeID(111)
	id2 := xSnowflake.SnowflakeID(222)
	id3 := xSnowflake.SnowflakeID(333)

	// 测试两个 ID 的组合
	gene12 := calc.HashMulti(id1, id2)
	gene21 := calc.HashMulti(id2, id1) // 顺序不同

	// 顺序不同应产生不同基因
	if gene12 == gene21 {
		t.Errorf("不同顺序的 ID 组合应产生不同基因")
	}

	// 测试三个 ID 的组合
	gene123 := calc.HashMulti(id1, id2, id3)

	if !gene123.IsValid() {
		t.Errorf("基因值应在 0-63 范围内: got %d", gene123)
	}
}

// Test_GeneCalc_HashString 测试基于字符串计算基因
func Test_GeneCalc_HashString(t *testing.T) {
	calc := xSnowflake.GeneCalc{}

	// 测试不同字符串
	gene1 := calc.HashString("order")
	gene2 := calc.HashString("payment")

	if gene1 == gene2 {
		t.Errorf("不同字符串应产生不同基因")
	}

	// 验证基因值在有效范围内
	if !gene1.IsValid() {
		t.Errorf("基因值应在 0-63 范围内: got %d", gene1)
	}

	// 验证同一字符串产生相同基因
	gene1Again := calc.HashString("order")
	if gene1 != gene1Again {
		t.Errorf("同一字符串应产生相同基因")
	}

	// 测试空字符串
	geneEmpty := calc.HashString("")
	if !geneEmpty.IsValid() {
		t.Errorf("空字符串也应产生有效基因: got %d", geneEmpty)
	}
}

// Test_GeneCalc_Distribution 测试基因分布均匀性
func Test_GeneCalc_Distribution(t *testing.T) {
	calc := xSnowflake.GeneCalc{}

	// 生成 1000 个基因，统计分布
	const count = 1000
	buckets := make(map[int64]int)

	for i := 0; i < count; i++ {
		id := xSnowflake.SnowflakeID(i*12345 + 67890) // 生成不同的 ID
		gene := calc.Hash(id)
		buckets[int64(gene)]++
	}

	// 验证至少有 50 个不同的基因值（说明分布比较均匀）
	uniqueCount := len(buckets)
	if uniqueCount < 50 {
		t.Errorf("基因分布不够均匀: only %d unique values out of 64 possible", uniqueCount)
	}

	// 验证没有基因值超过总数的 5%（避免过度集中）
	maxCount := 0
	for _, count := range buckets {
		if count > maxCount {
			maxCount = count
		}
	}
	if maxCount > count*5/100 {
		t.Errorf("基因过度集中: max bucket has %d out of %d", maxCount, count)
	}
}

// Test_GeneCalc_Consistency 测试基因计算一致性
func Test_GeneCalc_Consistency(t *testing.T) {
	calc := xSnowflake.GeneCalc{}

	// 使用固定的 ID 验证一致性
	fixedID := xSnowflake.SnowflakeID(1234567890123456789)

	// 多次计算应得到相同结果
	results := make([]xSnowflake.Gene, 10)
	for i := range results {
		results[i] = calc.Hash(fixedID)
	}

	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("基因计算不一致: result[0]=%d, result[%d]=%d", results[0], i, results[i])
		}
	}
}

// Benchmark_GeneCalc_Hash 基准测试：Hash 方法
func Benchmark_GeneCalc_Hash(b *testing.B) {
	calc := xSnowflake.GeneCalc{}
	id := xSnowflake.SnowflakeID(1234567890)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calc.Hash(id)
	}
}

// Benchmark_GeneCalc_HashMulti 基准测试：HashMulti 方法
func Benchmark_GeneCalc_HashMulti(b *testing.B) {
	calc := xSnowflake.GeneCalc{}
	id1 := xSnowflake.SnowflakeID(111)
	id2 := xSnowflake.SnowflakeID(222)
	id3 := xSnowflake.SnowflakeID(333)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calc.HashMulti(id1, id2, id3)
	}
}

// Benchmark_GeneCalc_HashString 基准测试：HashString 方法
func Benchmark_GeneCalc_HashString(b *testing.B) {
	calc := xSnowflake.GeneCalc{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calc.HashString("order-12345")
	}
}
