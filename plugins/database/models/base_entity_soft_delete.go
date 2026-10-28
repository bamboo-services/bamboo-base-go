package models

import (
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
	"gorm.io/gorm"
)

// BaseEntityWithSoftDelete 通用实体基类（带软删除）。
//
// 提供标准的主键、时间戳和软删除字段。
// 当 ID 为零值时，在 BeforeCreate 阶段通过当前上下文的 Snowflake Node 自动生成雪花 ID；
// 若 ID 已显式指定非零值，则安全保留原 ID，不依赖雪花节点。
//
// 业务基因特性：若实体类型实现了 GeneProvider 接口，则使用其定义的业务基因；
// 否则使用默认基因 snowflake.GeneDefault。
//
// 使用示例:
//
//	// 普通实体（使用默认基因）
//	type User struct {
//	    models.BaseEntityWithSoftDelete
//	    Username string `gorm:"type:varchar(64);uniqueIndex"`
//	    Email    string `gorm:"type:varchar(128)"`
//	}
//
//	// 带指定基因的实体
//	type Order struct {
//	    models.BaseEntityWithSoftDelete
//	    OrderNo string `gorm:"type:varchar(64);uniqueIndex"`
//	}
//
//	func (o *Order) GetGene() snowflake.Gene {
//	    return snowflake.GeneOrder
//	}
type BaseEntityWithSoftDelete struct {
	ID        snowflake.SnowflakeID `json:"id" gorm:"type:bigint;primaryKey;comment:主键"`
	CreatedAt time.Time             `json:"created_at" gorm:"not null;autoCreateTime:milli;comment:创建时间"`
	UpdatedAt time.Time             `json:"updated_at" gorm:"not null;autoUpdateTime:milli;comment:更新时间"`
	DeletedAt gorm.DeletedAt        `json:"-" gorm:"index;comment:删除时间"`
}

// BeforeCreate 创建前钩子，自动生成雪花 ID 并初始化时间戳。
//
// 当 ID 为零值时，会从 tx.Statement.Context 尝试读取 *snowflake.Node；
// 若缺少节点或生成失败，会立即中止并返回错误；
// 若 ID 已非零，则不依赖节点直接保留原 ID。
func (e *BaseEntityWithSoftDelete) BeforeCreate(tx *gorm.DB) error {
	if e.ID.IsZero() {
		id, err := generateEntityID(tx)
		if err != nil {
			return err
		}
		e.ID = id
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	return nil
}

// BeforeUpdate 更新前钩子，自动刷新更新时间戳。
func (e *BaseEntityWithSoftDelete) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now()
	return nil
}
