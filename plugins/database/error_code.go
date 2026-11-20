package database

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	RecordNotFound    = &apperr.ErrorCode{Code: 40402, Output: "DATABASE_RECORD_NOT_FOUND", Message: "记录未找到"}   // 记录未找到。
	OptimisticLock    = &apperr.ErrorCode{Code: 40902, Output: "DATABASE_OPTIMISTIC_LOCK", Message: "乐观锁冲突"}    // 乐观锁冲突。
	DuplicateEntry    = &apperr.ErrorCode{Code: 40903, Output: "DATABASE_DUPLICATE_ENTRY", Message: "重复条目"}     // 重复条目。
	UniqueConstraint  = &apperr.ErrorCode{Code: 40904, Output: "DATABASE_UNIQUE_CONSTRAINT", Message: "唯一约束冲突"} // 唯一约束冲突。
	ForeignKeyError   = &apperr.ErrorCode{Code: 40905, Output: "DATABASE_FOREIGN_KEY_ERROR", Message: "外键约束错误"} // 外键约束错误。
	IntegrityError    = &apperr.ErrorCode{Code: 40906, Output: "DATABASE_INTEGRITY_ERROR", Message: "完整性约束错误"}  // 完整性约束错误。
	DatabaseError     = &apperr.ErrorCode{Code: 50014, Output: "DATABASE_ERROR", Message: "数据库错误"}              // 数据库错误。
	TransactionFailed = &apperr.ErrorCode{Code: 50015, Output: "DATABASE_TRANSACTION_FAILED", Message: "事务失败"}  // 事务失败。
)
