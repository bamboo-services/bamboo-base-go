package oracle_test

import (
	"testing"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/oracle/v2"
	gormoracle "github.com/oracle-samples/gorm-oracle/oracle"
)

// TestNew_AcceptedFormats 锁定 Oracle 的宽松接受集合：URL、logfmt 与 Easy Connect 简写均被接受且原样透传。
func TestNew_AcceptedFormats(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantDSN string
	}{
		{
			name:    "oracle URL",
			dsn:     "oracle://scott:tiger@dbhost:1521/orclpdb1",
			wantDSN: "oracle://scott:tiger@dbhost:1521/orclpdb1",
		},
		{
			name:    "oracledb URL",
			dsn:     "oracledb://scott:tiger@dbhost:1521/orclpdb1",
			wantDSN: "oracledb://scott:tiger@dbhost:1521/orclpdb1",
		},
		{
			name:    "godror logfmt",
			dsn:     `user="scott" password="tiger" connectString="dbhost:1521/orclpdb1"`,
			wantDSN: `user="scott" password="tiger" connectString="dbhost:1521/orclpdb1"`,
		},
		{
			name:    "logfmt with libDir",
			dsn:     `user="scott" password="tiger" connectString="dbhost:1521/orclpdb1" libDir="/opt/instantclient"`,
			wantDSN: `user="scott" password="tiger" connectString="dbhost:1521/orclpdb1" libDir="/opt/instantclient"`,
		},
		{
			name:    "Easy Connect shorthand",
			dsn:     "scott/tiger@dbhost:1521/orclpdb1",
			wantDSN: "scott/tiger@dbhost:1521/orclpdb1",
		},
		{
			name:    "surrounding whitespace is trimmed",
			dsn:     "  oracle://scott:tiger@dbhost:1521/orclpdb1  ",
			wantDSN: "oracle://scott:tiger@dbhost:1521/orclpdb1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialector, err := oracle.New(tt.dsn).Open()
			if err != nil {
				t.Fatalf("unexpected Open() error: %v", err)
			}
			od, ok := dialector.(*gormoracle.Dialector)
			if !ok {
				t.Fatalf("expected *gormoracle.Dialector, got %T", dialector)
			}
			if od.DataSourceName != tt.wantDSN {
				t.Fatalf("DSN mismatch: got %q, want %q", od.DataSourceName, tt.wantDSN)
			}
		})
	}
}

// TestNew_LogfmtCredentialsEscaped 验证 logfmt 凭据使用 %q 安全转义且可还原。
func TestNew_LogfmtCredentialsEscaped(t *testing.T) {
	dsn := `user="user\"quote" password="pass\\slash\"quote" connectString="localhost:1521/ORCLPDB1"`

	dialector, err := oracle.New(dsn).Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	od, ok := dialector.(*gormoracle.Dialector)
	if !ok {
		t.Fatalf("expected *gormoracle.Dialector, got %T", dialector)
	}
	if od.DataSourceName != dsn {
		t.Fatalf("DSN mismatch: got %q, want %q", od.DataSourceName, dsn)
	}
}
