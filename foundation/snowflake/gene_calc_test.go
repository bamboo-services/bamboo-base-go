package snowflake

import (
	"testing"
)

func Test_GeneCalc_Hash(t *testing.T) {
	calc := GeneCalc{}

	id1 := SnowflakeID(1234567890)
	id2 := SnowflakeID(9876543210)

	gene1 := calc.Hash(id1)
	gene2 := calc.Hash(id2)

	if gene1 == gene2 {
		t.Errorf("不同 ID 应产生不同基因: gene1=%d, gene2=%d", gene1, gene2)
	}

	if !gene1.IsValid() {
		t.Errorf("基因值应在 0-63 范围内: got %d", gene1)
	}
	if !gene2.IsValid() {
		t.Errorf("基因值应在 0-63 范围内: got %d", gene2)
	}

	gene1Again := calc.Hash(id1)
	if gene1 != gene1Again {
		t.Errorf("同一 ID 应产生相同基因: first=%d, second=%d", gene1, gene1Again)
	}

	// zero id returns GeneDefault
	if calc.Hash(SnowflakeID(0)) != GeneDefault {
		t.Errorf("Zero id should return GeneDefault")
	}
}

func Test_GeneCalc_HashMulti(t *testing.T) {
	calc := GeneCalc{}

	id1 := SnowflakeID(111)
	id2 := SnowflakeID(222)
	id3 := SnowflakeID(333)

	gene12 := calc.HashMulti(id1, id2)
	gene21 := calc.HashMulti(id2, id1)

	if gene12 == gene21 {
		t.Errorf("不同顺序的 ID 组合应产生不同基因")
	}

	gene123 := calc.HashMulti(id1, id2, id3)
	if !gene123.IsValid() {
		t.Errorf("基因值应在 0-63 范围内: got %d", gene123)
	}

	if calc.HashMulti() != GeneDefault {
		t.Errorf("Empty IDs should return GeneDefault")
	}
}

func Test_GeneCalc_HashString(t *testing.T) {
	calc := GeneCalc{}

	gene1 := calc.HashString("order")
	gene2 := calc.HashString("payment")

	if gene1 == gene2 {
		t.Errorf("不同字符串应产生不同基因")
	}

	if !gene1.IsValid() {
		t.Errorf("基因值应在 0-63 范围内: got %d", gene1)
	}

	gene1Again := calc.HashString("order")
	if gene1 != gene1Again {
		t.Errorf("同一字符串应产生相同基因")
	}

	geneEmpty := calc.HashString("")
	if geneEmpty != GeneDefault {
		t.Errorf("空字符串应返回 GeneDefault: got %d", geneEmpty)
	}
}

func Test_GeneCalc_Distribution(t *testing.T) {
	calc := GeneCalc{}

	const count = 1000
	buckets := make(map[int64]int)

	for i := 0; i < count; i++ {
		id := SnowflakeID(i*12345 + 67890)
		gene := calc.Hash(id)
		buckets[int64(gene)]++
	}

	uniqueCount := len(buckets)
	if uniqueCount < 50 {
		t.Errorf("基因分布不够均匀: only %d unique values out of 64 possible", uniqueCount)
	}

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

func Test_Gene_Methods(t *testing.T) {
	if !GeneUser.IsSystem() {
		t.Error("GeneUser 应该是系统级基因")
	}
	if GeneUser.IsBusiness() {
		t.Error("GeneUser 不应该是业务级基因")
	}

	if GeneOrder.IsSystem() {
		t.Error("GeneOrder 不应该是系统级基因")
	}
	if !GeneOrder.IsBusiness() {
		t.Error("GeneOrder 应该是业务级基因")
	}

	if !Gene(0).IsValid() {
		t.Error("Gene(0) 应该是有效的")
	}
	if !Gene(63).IsValid() {
		t.Error("Gene(63) 应该是有效的")
	}
	if Gene(64).IsValid() {
		t.Error("Gene(64) 不应该是有效的")
	}
	if Gene(-1).IsValid() {
		t.Error("Gene(-1) 不应该是有效的")
	}

	if GeneUser.String() != "User" {
		t.Errorf("GeneUser.String() = %s; want User", GeneUser.String())
	}
	if GeneOrder.String() != "Order" {
		t.Errorf("GeneOrder.String() = %s; want Order", GeneOrder.String())
	}
	if Gene(50).String() != "Custom(50)" {
		t.Errorf("Gene(50).String() = %s; want Custom(50)", Gene(50).String())
	}
	if GeneUser.Int64() != 2 {
		t.Errorf("GeneUser.Int64() = %d; want 2", GeneUser.Int64())
	}
}
