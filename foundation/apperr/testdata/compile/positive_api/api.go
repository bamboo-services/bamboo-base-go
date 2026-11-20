// Package positiveapi 锁定P1已交付公开原语的精确静态契约。
package positiveapi

import (
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
)

var (
	_ func(error) bool                                                           = apperr.IsNil
	_ func(error) *apperr.Error                                                  = apperr.Find
	_ func(*apperr.ErrorCode, apperr.ErrMessage, error) *apperr.Error            = apperr.From
	_ func(*apperr.ErrorCode, apperr.ErrMessage, ...*apperr.Error) *apperr.Error = apperr.Join
	_ func(*apperr.ErrorCode, apperr.ErrMessage, ...error) *apperr.Error         = apperr.New
	_ func(error, control.Phase) *apperr.Error                                   = control.Classify
	_ func(valueerr.Kind, apperr.ErrMessage, error) *apperr.Error                = valueerr.From
)
