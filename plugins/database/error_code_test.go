package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
)

func TestDatabaseErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code *apperr.ErrorCode
		want apperr.ErrorCode
	}{
		{"RecordNotFound", database.RecordNotFound, apperr.ErrorCode{Code: 40402, Output: "DATABASE_RECORD_NOT_FOUND", Message: "记录未找到"}},
		{"OptimisticLock", database.OptimisticLock, apperr.ErrorCode{Code: 40902, Output: "DATABASE_OPTIMISTIC_LOCK", Message: "乐观锁冲突"}},
		{"DuplicateEntry", database.DuplicateEntry, apperr.ErrorCode{Code: 40903, Output: "DATABASE_DUPLICATE_ENTRY", Message: "重复条目"}},
		{"UniqueConstraint", database.UniqueConstraint, apperr.ErrorCode{Code: 40904, Output: "DATABASE_UNIQUE_CONSTRAINT", Message: "唯一约束冲突"}},
		{"ForeignKeyError", database.ForeignKeyError, apperr.ErrorCode{Code: 40905, Output: "DATABASE_FOREIGN_KEY_ERROR", Message: "外键约束错误"}},
		{"IntegrityError", database.IntegrityError, apperr.ErrorCode{Code: 40906, Output: "DATABASE_INTEGRITY_ERROR", Message: "完整性约束错误"}},
		{"DatabaseError", database.DatabaseError, apperr.ErrorCode{Code: 50014, Output: "DATABASE_ERROR", Message: "数据库错误"}},
		{"TransactionFailed", database.TransactionFailed, apperr.ErrorCode{Code: 50015, Output: "DATABASE_TRANSACTION_FAILED", Message: "事务失败"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.code == nil || *tc.code != tc.want {
				t.Fatalf("database error code changed: got %+v, want %+v", tc.code, tc.want)
			}
			cause := errors.New("query failed")
			err := apperr.New(tc.code, "查询失败", cause)
			if err.Code != tc.want || !errors.Is(err, cause) {
				t.Fatalf("database error code or cause was lost: %+v", err)
			}
		})
	}
}

func TestGetPreservesApplicationError(t *testing.T) {
	for _, ctx := range []context.Context{nil, context.Background()} {
		db, err := database.Get(ctx)
		if db != nil || err == nil {
			t.Fatalf("expected missing-container error, got db=%v err=%v", db, err)
		}
		if err.Code != *apperr.FrameworkContainerNotMounted || !errors.Is(err, application.ErrContainerNotMounted) {
			t.Fatalf("application error was replaced by a database error: %+v", err)
		}
	}
}
