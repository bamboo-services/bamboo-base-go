package snowflake

import "hash/fnv"

// GeneCalc 提供基于 FNV-1a 哈希算法的业务基因映射与计算工具。
//
// 本类型为无状态的轻量值对象，并发安全，常用于依据关联主实体 ID 或业务特征字符串
// 计算子实体的基因分类，实现分库分表等路由对齐。
//
// 典型用法：
//
//	calc := snowflake.CalcGene()
//	userGene := calc.Hash(userID)
//	orderID, err := node.Generate(userGene)
type GeneCalc struct{}

// CalcGene 创建并返回一个无状态的基因计算工具 [GeneCalc] 实例。
//
// 内部计算采用 FNV-1a 64 位哈希并将结果截断至低 6 位（掩码 0x3F），
// 输出的基因范围恒定处于 [0, 63] 区间内。
func CalcGene() GeneCalc {
	return GeneCalc{}
}

// Hash 依据单个 [SnowflakeID] 计算并返回对应的 6 位业务基因。
//
// 算法将 ID 的十进制字符串经过 FNV-1a 哈希后取低 6 位作为基因值。
// 若输入的 id 为零值（[SnowflakeID.IsZero] 为 true），则直接返回 [GeneDefault]。
func (gc GeneCalc) Hash(id SnowflakeID) Gene {
	if id.IsZero() {
		return GeneDefault
	}
	h := fnv.New64a()
	if _, err := h.Write([]byte(id.String())); err != nil {
		return GeneDefault
	}
	return Gene(h.Sum64() & 0x3F) // 取低 6 位
}

// HashMulti 依据多个 [SnowflakeID] 组合计算并返回对应的 6 位业务基因。
//
// 算法按参数传递顺序将多个 ID 的十进制字符串依序写入 FNV-1a 哈希器。
// 入参顺序敏感，不同顺序将产生不同的计算结果；若未传入任何 ID 参数则返回 [GeneDefault]。
func (gc GeneCalc) HashMulti(ids ...SnowflakeID) Gene {
	if len(ids) == 0 {
		return GeneDefault
	}
	h := fnv.New64a()
	for _, id := range ids {
		if _, err := h.Write([]byte(id.String())); err != nil {
			return GeneDefault
		}
	}
	return Gene(h.Sum64() & 0x3F) // 取低 6 位
}

// HashString 依据任意字符串内容计算并返回对应的 6 位业务基因。
//
// 适用于根据非雪花 ID 字段（如用户账号名、地区编码、业务代号等）计算基因特征的场景。
// 若输入字符串 s 为空字符串，则直接返回 [GeneDefault]。
func (gc GeneCalc) HashString(s string) Gene {
	if s == "" {
		return GeneDefault
	}
	h := fnv.New64a()
	if _, err := h.Write([]byte(s)); err != nil {
		return GeneDefault
	}
	return Gene(h.Sum64() & 0x3F) // 取低 6 位
}
