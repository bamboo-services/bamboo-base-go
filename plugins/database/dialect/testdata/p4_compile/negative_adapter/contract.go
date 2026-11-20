package fixture

import (
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"gorm.io/gorm"
)

type adapter struct{}

func (*adapter) Name() string                  { return "legacy" }
func (*adapter) Open() (gorm.Dialector, error) { return nil, nil }

var _ dialect.Adapter = (*adapter)(nil)
