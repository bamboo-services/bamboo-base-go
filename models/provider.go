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
	GetGene() xSnowflake.Gene
	CalcGene() xSnowflake.GeneCalc
}
