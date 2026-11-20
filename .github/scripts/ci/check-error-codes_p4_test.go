package main

import "testing"

func TestP4DatabaseSubdomainOwnership(t *testing.T) {
	entries := []errorEntry{
		{50000, "FRAMEWORK_INTERNAL_ERROR", "foundation/apperr", "FrameworkInternalError", "框架内部错误", "active"},
		{50001, "DATABASE_DIALECT_DRIVER_INVALID", "plugins/database/dialect", "DriverInvalid", "数据库驱动声明非法", "active"},
		{50002, "DATABASE_MODEL_INVALID", "plugins/database/models", "ModelInvalid", "数据库模型契约无效", "active"},
	}
	root := createFixture(t, entries, `var FrameworkInternalError = &ErrorCode{Code:50000,Output:"FRAMEWORK_INTERNAL_ERROR",Message:"框架内部错误"}`)
	writeFixture(t, root, "plugins/database/dialect/error_code.go", `package dialect
 import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
 var DriverInvalid = &apperr.ErrorCode{Code:50001,Output:"DATABASE_DIALECT_DRIVER_INVALID",Message:"数据库驱动声明非法"}`)
	writeFixture(t, root, "plugins/database/models/error_code.go", `package models
 import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
 var ModelInvalid = &apperr.ErrorCode{Code:50002,Output:"DATABASE_MODEL_INVALID",Message:"数据库模型契约无效"}`)
	if err := checkRepository(root); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		`import d "github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"; var Alias = d.DriverInvalid`,
		`import . "github.com/bamboo-services/bamboo-base-go/plugins/database/v2/models"; var Alias = ModelInvalid`,
	} {
		writeFixture(t, root, "plugins/cache/extra.go", "package cache\n"+source)
		if err := checkRepository(root); err == nil {
			t.Fatal("subdomain bypass accepted")
		}
	}
	for _, entry := range []errorEntry{
		{50000, "DATABASE_DIALECT_DRIVER_INVALID", "plugins/database", "DriverInvalid", "错误", "active"},
		{50000, "DATABASE_ERROR", "plugins/database/dialect", "DatabaseError", "错误", "active"},
		{50000, "DATABASE_DIALECT_MODEL_INVALID", "plugins/database/models", "ModelInvalid", "错误", "active"},
	} {
		if err := validateVocabulary([]errorEntry{entry}); err == nil {
			t.Fatalf("borrowed prefix accepted: %+v", entry)
		}
	}
}
