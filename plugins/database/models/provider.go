package models

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"

// GeneProvider 基因提供者接口。
//
// 实现此接口的实体可以在 BeforeCreate 时自动提供对应的业务基因类型。
// 这允许每个业务实体定义自己的业务基因，进而嵌入生成的雪花 ID 中。
//
// 使用示例:
//
//	type Order struct {
//	    models.BaseEntityWithSoftDelete
//	    OrderNo string `gorm:"type:varchar(64)"`
//	}
//
//	func (o *Order) GetGene() snowflake.Gene {
//	    return snowflake.GeneOrder
//	}
type GeneProvider interface {
	// GetGene 返回实体的业务基因类型，用于生成雪花 ID 时指定业务基因。
	GetGene() snowflake.Gene
}
