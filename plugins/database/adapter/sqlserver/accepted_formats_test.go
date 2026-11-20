package sqlserver_test

import (
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlserver/v2"
	gormsqlserver "gorm.io/driver/sqlserver"
)

// TestNew_AcceptedFormats 锁定 SQL Server 的接受集合：合法 URL 必须原样透传给驱动。
func TestNew_AcceptedFormats(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantDSN string
	}{
		{
			name:    "plain URL",
			dsn:     "sqlserver://sa:pass@localhost:1433?database=master",
			wantDSN: "sqlserver://sa:pass@localhost:1433?database=master",
		},
		{
			name:    "URL with database query",
			dsn:     "sqlserver://sa:pass@db.internal:14330?database=app",
			wantDSN: "sqlserver://sa:pass@db.internal:14330?database=app",
		},
		{
			name:    "URL without database query",
			dsn:     "sqlserver://sa:pass@localhost:1433",
			wantDSN: "sqlserver://sa:pass@localhost:1433",
		},
		{
			name:    "IPv6 host",
			dsn:     "sqlserver://sa:pass@[::1]:1433?database=master",
			wantDSN: "sqlserver://sa:pass@[::1]:1433?database=master",
		},
		{
			name:    "surrounding whitespace is trimmed",
			dsn:     "  sqlserver://sa:pass@localhost:1433?database=master  ",
			wantDSN: "sqlserver://sa:pass@localhost:1433?database=master",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialector, err := sqlserver.New(tt.dsn).Open()
			if err != nil {
				t.Fatalf("unexpected Open() error: %v", err)
			}
			sd, ok := dialector.(*gormsqlserver.Dialector)
			if !ok {
				t.Fatalf("expected *gormsqlserver.Dialector, got %T", dialector)
			}
			if sd.DSN != tt.wantDSN {
				t.Fatalf("DSN mismatch: got %q, want %q", sd.DSN, tt.wantDSN)
			}
		})
	}
}

// TestNew_CredentialsAreEscaped 验证凭据按 URL 标准编码转义且不丢失原值。
func TestNew_CredentialsAreEscaped(t *testing.T) {
	dsn := "sqlserver://user%40corp:p%40ss%2Fword@localhost:1433?database=master"

	dialector, err := sqlserver.New(dsn).Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	sd, ok := dialector.(*gormsqlserver.Dialector)
	if !ok {
		t.Fatalf("expected *gormsqlserver.Dialector, got %T", dialector)
	}
	if !strings.Contains(sd.DSN, "user%40corp") || !strings.Contains(sd.DSN, "p%40ss%2Fword") {
		t.Fatalf("expected escaped credentials in DSN, got %q", sd.DSN)
	}
}
