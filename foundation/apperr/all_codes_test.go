package apperr_test

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

var allExportedCodes = []*apperr.ErrorCode{
	apperr.FrameworkInternalError,
	apperr.FrameworkContainerNotMounted,
	apperr.FrameworkDependencyUndeclared,
	apperr.FrameworkComponentNotFound,
	apperr.FrameworkComponentTypeMismatch,
	apperr.FrameworkComponentStopped,
	apperr.FrameworkComponentUnavailable,
	apperr.FrameworkComponentInvalidState,
}
