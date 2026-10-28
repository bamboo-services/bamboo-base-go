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

// 模型钩子与发号器错误变量。
var (
	ErrNilStatement          = errors.New("models: gorm statement is nil")               // GORM Statement 为 nil 时返回。
	ErrNilContext            = errors.New("models: statement context is nil")            // GORM Statement.Context 为 nil 时返回。
	ErrSnowflakeNodeNotFound = errors.New("models: snowflake node not found in context") // context 中无法获取有效雪花算法节点时返回。
	ErrNilSnowflakeNode      = errors.New("models: snowflake node is nil")               // context 中获取到的雪花节点为 nil 或 typed nil 时返回。
)

// BaseEntity 是通用实体基类（不包含软删除字段）。
//
// 该基类提供标准的雪花主键 ID 和毫秒级创建与更新时间戳字段。
// 当 ID 为零值时，在 BeforeCreate 阶段通过当前上下文的雪花发号器节点自动生成 ID；
// 若 ID 已显式指定非零值，则安全保留原 ID，不依赖雪花节点。
//
// 业务基因特性：若实体类型实现了 [GeneProvider] 接口，则采用其自定义业务基因生成 ID；
// 否则使用默认业务基因 [snowflake.GeneDefault]。
type BaseEntity struct {
	// ID 是实体的雪花主键。在实体创建时若为零值将自动生成。
	ID snowflake.SnowflakeID `json:"id" gorm:"type:bigint;primaryKey;comment:主键"`
	// CreatedAt 是记录创建时间，以毫秒级时间戳持久化。
	CreatedAt time.Time `json:"created_at" gorm:"not null;autoCreateTime:milli;comment:创建时间"`
	// UpdatedAt 是记录最后更新时间，以毫秒级时间戳自动维护。
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;autoUpdateTime:milli;comment:更新时间"`
}

// BeforeCreate 是 GORM 创建记录前的生命周期钩子，负责自动生成雪花 ID 并初始化时间戳。
//
// 当 ID 为零值时，钩子尝试从 tx.Statement.Context 中读取 [*snowflake.Node]；
// 若上下文中缺少节点或节点为 nil，钩子立即终止并返回错误，阻止无主键或零值主键写入；
// 若 ID 已为非零值，则安全保留原 ID，不尝试获取雪花节点。
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

// BeforeUpdate 是 GORM 更新记录前的生命周期钩子，自动刷新更新时间戳 [BaseEntity.UpdatedAt]。
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
