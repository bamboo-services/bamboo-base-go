package xModels

import (
	"time"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
	"gorm.io/gorm"
)

// BaseEntity 通用实体基类（不带软删除）
//
// 提供标准的主键和时间戳字段。
// ID 使用雪花算法自动生成，无需手动设置。
//
// 支持基因功能：如果实体实现了 GeneProvider 接口，
// 会自动使用指定的基因类型生成 ID；否则使用默认基因（GeneDefault=0）。
//
// 使用方式:
//
//	// 普通实体（Gene=0）
//	type User struct {
//	    xModels.BaseEntity
//	    Username string `gorm:"type:varchar(64);uniqueIndex"`
//	    Email    string `gorm:"type:varchar(128)"`
//	}
//
//	// 带基因的实体
//	type Order struct {
//	    xModels.BaseEntity
//	    OrderNo string `gorm:"type:varchar(64);uniqueIndex"`
//	}
//
//	func (o *Order) GetGene() xSnowflake.Gene {
//	    return xSnowflake.GeneOrder
//	}
type BaseEntity struct {
	ID        xSnowflake.SnowflakeID `json:"id" gorm:"type:bigint;primaryKey;comment:主键"`
	CreatedAt time.Time              `json:"-" gorm:"autoCreateTime:milli;not null;comment:创建时间"`
	UpdatedAt time.Time              `json:"updated_at" gorm:"autoUpdateTime:milli;not null;comment:更新时间"`
}

// BeforeCreate 创建前钩子，自动生成雪花 ID
//
// 如果实体实现了 GeneProvider 接口，则使用其提供的基因类型；
// 否则使用默认基因类型（GeneDefault）。
// 同时设置 CreatedAt 和 UpdatedAt 时间戳。
//
// 参数说明:
//   - tx: GORM 数据库事务
//
// 返回值:
//   - error: 钩子错误
func (e *BaseEntity) BeforeCreate(tx *gorm.DB) error {
	if e.ID.IsZero() {
		// 尝试从实体获取基因类型
		gene := xSnowflake.GeneDefault
		if provider, ok := tx.Statement.Dest.(GeneProvider); ok {
			gene = provider.GetGene()
		}
		e.ID = xSnowflake.GenerateID(gene)
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	return nil
}

// BeforeUpdate 更新前钩子，自动更新时间戳
//
// 参数说明:
//   - tx: GORM 数据库事务
//
// 返回值:
//   - error: 钩子错误
func (e *BaseEntity) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now()
	return nil
}
