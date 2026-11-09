package main

import "testing"

func TestP1FoundationDomainDefinitions(t *testing.T) {
	entries := []errorEntry{
		{40000, "VALUE_INVALID_FORMAT", "foundation/valueerr", "InvalidFormat", "值格式非法", "active"},
		{49900, "CONTROL_CANCELED", "foundation/control", "Canceled", "操作已取消", "active"},
		{50000, "FRAMEWORK_INTERNAL_ERROR", "foundation/apperr", "FrameworkInternalError", "框架内部错误", "active"},
		{50001, "VALUE_UNSUPPORTED_TARGET", "foundation/valueerr", "UnsupportedTarget", "不支持的转换目标", "active"},
	}
	root := createFixture(t, entries, `var FrameworkInternalError = &ErrorCode{Code:50000, Output:"FRAMEWORK_INTERNAL_ERROR", Message:"框架内部错误"}`)
	writeFixture(t, root, "foundation/control/error_code.go", `package control
import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
var Canceled = &apperr.ErrorCode{Code:49900, Output:"CONTROL_CANCELED", Message:"操作已取消"}`)
	writeFixture(t, root, "foundation/valueerr/error_code.go", `package valueerr
import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
var InvalidFormat = &apperr.ErrorCode{Code:40000, Output:"VALUE_INVALID_FORMAT", Message:"值格式非法"}
var UnsupportedTarget = &apperr.ErrorCode{Code:50001, Output:"VALUE_UNSUPPORTED_TARGET", Message:"不支持的转换目标"}`)
	if err := checkRepository(root); err != nil {
		t.Fatalf("registered control/value domains must pass: %v", err)
	}
	for _, tc := range []struct {
		name   string
		source string
	}{
		{"named alias", `import ctr "github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
var Exported = ctr.Canceled`},
		{"dot import", `import . "github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
var Exported = InvalidFormat`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeFixture(t, root, "plugins/cache/extra.go", "package cache\n"+tc.source)
			if err := checkRepository(root); err == nil {
				t.Fatal("new domains must not bypass imported-code guards")
			}
		})
	}
}

func TestP1DomainsCannotBorrowPrefixes(t *testing.T) {
	for _, entry := range []errorEntry{
		{40000, "FRAMEWORK_INVALID_FORMAT", "foundation/valueerr", "InvalidFormat", "值格式非法", "active"},
		{49900, "VALUE_CANCELED", "foundation/control", "Canceled", "操作已取消", "active"},
		{40000, "CONTROL_INVALID_FORMAT", "foundation/valueerr", "InvalidFormat", "值格式非法", "active"},
	} {
		if err := validateVocabulary([]errorEntry{entry}); err == nil {
			t.Fatalf("borrowed domain prefix accepted: %+v", entry)
		}
	}
}
