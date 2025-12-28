package xModels

import (
	"time"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
	"gorm.io/gorm"
)

// BaseEntity 通用实体基类
//
// 提供标准的主键、时间戳和软删除字段。
// ID 使用雪花算法自动生成，无需手动设置。
//
// 使用方式:
//
//	type User struct {
//	    xModels.BaseEntity
//	    Username string `gorm:"type:varchar(64);uniqueIndex"`
//	    Email    string `gorm:"type:varchar(128)"`
//	}
type BaseEntity struct {
	ID        xSnowflake.SnowflakeID `gorm:"type:bigint;primaryKey;comment:主键"`
	CreatedAt time.Time              `gorm:"autoCreateTime:milli;not null;comment:创建时间"`
	UpdatedAt time.Time              `gorm:"autoUpdateTime:milli;not null;comment:更新时间"`
	DeletedAt gorm.DeletedAt         `gorm:"type:timestamp;index;comment:删除时间"`
}

// BeforeCreate 创建前钩子，自动生成雪花 ID
//
// 如果 ID 为零值，则使用默认雪花节点生成新 ID。
// 同时设置 CreatedAt 和 UpdatedAt 时间戳。
//
// 参数说明:
//   - tx: GORM 数据库事务
//
// 返回值:
//   - error: 钩子错误
func (e *BaseEntity) BeforeCreate(tx *gorm.DB) error {
	if e.ID.IsZero() {
		e.ID = xSnowflake.GenerateID()
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

// GeneProvider 基因提供者接口
//
// 实现此接口的实体可以在 BeforeCreate 时自动获取正确的基因类型。
// 这允许每个实体类型定义自己的业务基因。
//
// 使用方式:
//
//	type Order struct {
//	    xModels.GeneBaseEntity
//	    OrderNo string `gorm:"type:varchar(64)"`
//	}
//
//	func (o *Order) GetGene() xSnowflake.Gene {
//	    return xSnowflake.GeneOrder
//	}
type GeneProvider interface {
	GetGene() xSnowflake.Gene
}

// GeneBaseEntity 基因实体基类
//
// 与 BaseEntity 类似，但使用基因雪花 ID。
// ID 中嵌入业务类型基因，便于从 ID 识别数据类型。
//
// 使用方式:
//
//	type Order struct {
//	    xModels.GeneBaseEntity
//	    OrderNo     string  `gorm:"type:varchar(64);uniqueIndex"`
//	    TotalAmount float64 `gorm:"type:decimal(10,2)"`
//	}
//
//	// 实现 GeneProvider 接口以指定基因类型
//	func (o *Order) GetGene() xSnowflake.Gene {
//	    return xSnowflake.GeneOrder
//	}
type GeneBaseEntity struct {
	ID        xSnowflake.GeneSnowflakeID `gorm:"type:bigint;primaryKey;comment:主键"`
	CreatedAt time.Time                  `gorm:"autoCreateTime:milli;not null;comment:创建时间"`
	UpdatedAt time.Time                  `gorm:"autoUpdateTime:milli;not null;comment:更新时间"`
	DeletedAt gorm.DeletedAt             `gorm:"type:timestamp;index;comment:删除时间"`
}

// BeforeCreate 创建前钩子，自动生成基因雪花 ID
//
// 如果实体实现了 GeneProvider 接口，则使用其提供的基因类型；
// 否则使用默认基因类型（GeneDefault）。
//
// 参数说明:
//   - tx: GORM 数据库事务
//
// 返回值:
//   - error: 钩子错误
func (e *GeneBaseEntity) BeforeCreate(tx *gorm.DB) error {
	if e.ID.IsZero() {
		// 尝试从实体获取基因类型
		gene := xSnowflake.GeneDefault
		if provider, ok := tx.Statement.Dest.(GeneProvider); ok {
			gene = provider.GetGene()
		}
		e.ID = xSnowflake.MustGenerateGeneID(gene)
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
func (e *GeneBaseEntity) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now()
	return nil
}
