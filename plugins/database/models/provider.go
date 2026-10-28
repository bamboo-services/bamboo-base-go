package models

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"

// GeneProvider 是业务基因提供者接口。
//
// 业务实体通过实现此接口，可以在 [BaseEntity.BeforeCreate] 或
// [BaseEntityWithSoftDelete.BeforeCreate] 时自动提供对应的业务基因类型 [snowflake.Gene]。
// 这允许不同的业务实体（如用户、订单、支付流水）嵌入专属的业务基因，
// 从而使生成的雪花 ID 在全局具备业务语义可区分性。
type GeneProvider interface {
	// GetGene 返回实体的业务基因类型，用于生成雪花 ID 时指定业务基因。
	GetGene() snowflake.Gene
}
