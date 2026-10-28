package database_test

import (
	"database/sql"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// fakeDialector 最小 test-only GORM Dialector，封装已有的 *sql.DB 用于测试连接池、Ping、Close、Logger 等行为。
type fakeDialector struct {
	name string
	conn *sql.DB
}

func newFakeDialector(name string, conn *sql.DB) gorm.Dialector {
	return &fakeDialector{
		name: name,
		conn: conn,
	}
}

func (d *fakeDialector) Name() string {
	if d.name != "" {
		return d.name
	}
	return "fake"
}

func (d *fakeDialector) Initialize(db *gorm.DB) error {
	if d.conn != nil {
		db.ConnPool = d.conn
	}
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}

func (d *fakeDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return nil
}

func (d *fakeDialector) DataTypeOf(field *schema.Field) string {
	return ""
}

func (d *fakeDialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (d *fakeDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	_ = writer.WriteByte('?')
}

func (d *fakeDialector) QuoteTo(writer clause.Writer, str string) {
	_, _ = writer.WriteString(str)
}

func (d *fakeDialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, "'", vars...)
}
