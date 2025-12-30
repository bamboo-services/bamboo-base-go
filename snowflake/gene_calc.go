package xSnowflake

import "hash/fnv"

// GeneCalc 基因计算工具
//
// 提供基于 FNV-1a 哈希算法的基因计算方法，用于动态生成基因类型。
// 适用于需要基于关联 ID（如 UserID）计算基因的场景，实现分片定位。
//
// 使用方式:
//
//	type Order struct {
//	    xModels.BaseEntity
//	    UserID  xSnowflake.SnowflakeID
//	    OrderNo string
//	}
//
//	func (o *Order) GetGene() xSnowflake.Gene {
//	    return xSnowflake.CalcGene().Hash(o.UserID)
//	}
type GeneCalc struct{}

// CalcGene 获取 GeneCalc 实例
//
// 基因计算基于 FNV-1a 哈希算法，支持基于 ID 或字符串的基因值动态生成。
// 常用于分片定位和关联 ID 的基因计算场景。
//
// 注意：这里的计算的基因范围为 0-63，即 6 位二进制数。（不会受到原始雪花算法基因的影响）
func CalcGene() GeneCalc {
	return GeneCalc{}
}

// Hash 基于 SnowflakeID 计算基因（FNV-1a 哈希）
//
// 使用 FNV-1a 64位哈希算法，对 ID 字符串进行哈希，取低 6 位作为基因值（0-63）。
//
// 参数说明:
//   - id: 雪花 ID
//
// 返回值:
//   - Gene: 计算出的基因值（0-63），如果计算失败返回 GeneDefault
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

// HashMulti 基于多个 SnowflakeID 计算组合基因
//
// 对多个 ID 进行组合哈希，适用于需要基于多个关联实体计算基因的场景。
// 哈希顺序影响结果，ID 的顺序不同会产生不同的基因值。
//
// 参数说明:
//   - ids: 雪花 ID 列表
//
// 返回值:
//   - Gene: 计算出的组合基因值（0-63），如果计算失败或 IDs 为空返回 GeneDefault
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

// HashString 基于字符串计算基因
//
// 对任意字符串进行哈希，适用于基于非 ID 字段（如地区编码、分类等）计算基因。
//
// 参数说明:
//   - s: 输入字符串
//
// 返回值:
//   - Gene: 计算出的基因值（0-63），如果字符串为空或计算失败返回 GeneDefault
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
