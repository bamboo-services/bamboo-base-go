package fixture

import (
	"errors"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"gorm.io/gorm"
)

func Open() (gorm.Dialector, *apperr.Error) { return nil, errors.New("native failure") }
