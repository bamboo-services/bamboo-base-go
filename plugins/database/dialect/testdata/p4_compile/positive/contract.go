package fixture

import (
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/models"
	"gorm.io/gorm"
)

type adapter struct{}

func (*adapter) Name() string                          { return "fixture" }
func (*adapter) Open() (gorm.Dialector, *apperr.Error) { return nil, nil }

var (
	_ dialect.Adapter                                        = (*adapter)(nil)
	_ func(dialect.Driver) (dialect.Adapter, *apperr.Error)  = dialect.Lookup
	_ func(dialect.Adapter) (gorm.Dialector, *apperr.Error)  = dialect.Adapter.Open
	_ func(*models.BaseEntity, *gorm.DB) error               = (*models.BaseEntity).BeforeCreate
	_ func(*models.BaseEntity, *gorm.DB) error               = (*models.BaseEntity).BeforeUpdate
	_ func(*models.BaseEntityWithSoftDelete, *gorm.DB) error = (*models.BaseEntityWithSoftDelete).BeforeCreate
	_ func(*models.BaseEntityWithSoftDelete, *gorm.DB) error = (*models.BaseEntityWithSoftDelete).BeforeUpdate
)
