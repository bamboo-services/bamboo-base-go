package grpcresult_test

import (
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	"google.golang.org/grpc/status"
)

// 编译期固定公开分类原语的参数与强类型结果；不把分类器当作原生状态转换器。
var _ func(*status.Status, control.Phase) *apperr.Error = grpcresult.ClassifyStatus
