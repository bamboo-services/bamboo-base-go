package xModels

import xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"

// GeneProvider 基因提供者接口
//
// 实现此接口的实体可以在 BeforeCreate 时自动获取正确的基因类型。
// 这允许每个实体类型定义自己的业务基因。
//
// 使用方式:
//
//	type Order struct {
//	    xModels.BaseEntityWithSoftDelete
//	    OrderNo string `gorm:"type:varchar(64)"`
//	}
//
//	func (o *Order) GetGene() xSnowflake.Gene {
//	    return xSnowflake.GeneOrder
//	}
type GeneProvider interface {

	// GetGene 返回实体的基因类型，用于生成雪花 ID 时指定业务基因。
	//
	// 返回值:
	//   - xSnowflake.Gene: 实体定义的基因类型，用于区分不同的业务逻辑。
	GetGene() xSnowflake.Gene
}
