package models

import (
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
	"gorm.io/gorm"
)

// BaseEntityWithSoftDelete 是带软删除特性的通用实体基类。
//
// 该基类在 [BaseEntity] 基础上增加了 GORM 的软删除字段 [BaseEntityWithSoftDelete.DeletedAt]。
// 当 ID 为零值时，在 BeforeCreate 阶段通过当前上下文的雪花发号器节点自动生成 ID；
// 若 ID 已显式指定非零值，则安全保留原 ID，不依赖雪花节点。
//
// 业务基因特性：若实体类型实现了 [GeneProvider] 接口，则采用其自定义业务基因生成 ID；
// 否则使用默认业务基因 [snowflake.GeneDefault]。
type BaseEntityWithSoftDelete struct {
	// ID 是实体的雪花主键。在实体创建时若为零值将自动生成。
	ID snowflake.SnowflakeID `json:"id" gorm:"type:bigint;primaryKey;comment:主键"`
	// CreatedAt 是记录创建时间，以毫秒级时间戳持久化。
	CreatedAt time.Time `json:"created_at" gorm:"not null;autoCreateTime:milli;comment:创建时间"`
	// UpdatedAt 是记录最后更新时间，以毫秒级时间戳自动维护。
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;autoUpdateTime:milli;comment:更新时间"`
	// DeletedAt 是 GORM 软删除标记字段，非空时表示记录已被软删除。
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// BeforeCreate 是 GORM 创建记录前的生命周期钩子，负责自动生成雪花 ID 并初始化时间戳。
//
// 当 ID 为零值时，钩子尝试从 tx.Statement.Context 中读取 [*snowflake.Node]；
// 若上下文中缺少节点或节点为 nil，钩子立即终止并返回错误，阻止无主键或零值主键写入；
// 若 ID 已为非零值，则安全保留原 ID，不尝试获取雪花节点。
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

// BeforeUpdate 是 GORM 更新记录前的生命周期钩子，自动刷新更新时间戳 [BaseEntityWithSoftDelete.UpdatedAt]。
func (e *BaseEntityWithSoftDelete) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now()
	return nil
}
