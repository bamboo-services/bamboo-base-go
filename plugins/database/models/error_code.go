package models

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// ModelInvalid 用于模型自身缺少必要上下文或节点实例的契约失败。
var ModelInvalid = &apperr.ErrorCode{Code: 50037, Output: "DATABASE_MODEL_INVALID", Message: "数据库模型契约无效"} // 模型自身调用契约无效。
