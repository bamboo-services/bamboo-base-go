package models

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
	"gorm.io/gorm"
)

var (
	// ErrNilStatement 当 GORM Statement 为 nil 时返回。
	ErrNilStatement = errors.New("models: gorm statement is nil")
	// ErrNilContext 当 GORM Statement.Context 为 nil 时返回。
	ErrNilContext = errors.New("models: statement context is nil")
	// ErrSnowflakeNodeNotFound 当 context 中无法获取有效雪花算法节点时返回。
	ErrSnowflakeNodeNotFound = errors.New("models: snowflake node not found in context")
	// ErrNilSnowflakeNode 当 context 中获取到的雪花节点为 nil 时返回。
	ErrNilSnowflakeNode = errors.New("models: snowflake node is nil")
)

// BaseEntity 通用实体基类（不带软删除）。
//
// 提供标准的主键和时间戳字段。
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
//	    models.BaseEntity
//	    Username string `gorm:"type:varchar(64);uniqueIndex"`
//	    Email    string `gorm:"type:varchar(128)"`
//	}
//
//	// 带指定基因的实体
//	type Order struct {
//	    models.BaseEntity
//	    OrderNo string `gorm:"type:varchar(64);uniqueIndex"`
//	}
//
//	func (o *Order) GetGene() snowflake.Gene {
//	    return snowflake.GeneOrder
//	}
type BaseEntity struct {
	ID        snowflake.SnowflakeID `json:"id" gorm:"type:bigint;primaryKey;comment:主键"`
	CreatedAt time.Time             `json:"created_at" gorm:"not null;autoCreateTime:milli;comment:创建时间"`
	UpdatedAt time.Time             `json:"updated_at" gorm:"not null;autoUpdateTime:milli;comment:更新时间"`
}

// BeforeCreate 创建前钩子，自动生成雪花 ID 并初始化时间戳。
//
// 当 ID 为零值时，会从 tx.Statement.Context 尝试读取 *snowflake.Node；
// 若缺少节点或生成失败，会立即中止并返回错误；
// 若 ID 已非零，则不依赖节点直接保留原 ID。
func (e *BaseEntity) BeforeCreate(tx *gorm.DB) error {
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
func (e *BaseEntity) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now()
	return nil
}

func generateEntityID(tx *gorm.DB) (snowflake.SnowflakeID, error) {
	if tx == nil || tx.Statement == nil {
		return 0, ErrNilStatement
	}
	ctx := tx.Statement.Context
	if ctx == nil {
		return 0, ErrNilContext
	}

	node, appErr := application.Get[*snowflake.Node](ctx, ctxkey.SnowflakeNodeKey)
	if appErr != nil {
		return 0, fmt.Errorf("%w: %w", ErrSnowflakeNodeNotFound, appErr)
	}
	if node == nil || isTypedNil(node) {
		return 0, ErrNilSnowflakeNode
	}

	gene := snowflake.GeneDefault
	if tx.Statement.Dest != nil {
		if provider, ok := tx.Statement.Dest.(GeneProvider); ok {
			gene = provider.GetGene()
		}
	}

	id, err := node.Generate(gene)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func isTypedNil(v any) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}
