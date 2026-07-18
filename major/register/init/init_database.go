package xInit

import (
	"context"
	"fmt"
	"log/slog"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xGormLog "github.com/bamboo-services/bamboo-base-go/major/log"
	xOptionDB "github.com/bamboo-services/bamboo-base-go/major/option/database"
	xRegNode "github.com/bamboo-services/bamboo-base-go/major/register/node"
	"github.com/libtnb/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DatabaseInit 根据传入的 [xOptionDB.Config] 构造数据库初始化节点。
//
// 返回的 [xRegNode.Node] 会根据 Config.Driver 选择对应的 GORM 驱动
// （mysql / postgres / sqlite），用项目自带的 [xGormLog.SlogLogger] 作为 GORM
// 日志适配器，并按 Common() 中的连接池参数配置底层 *sql.DB。
//
// 调用方：Runner 在 UseAfterExec 阶段按 option 决定是否装配此节点。
// 若 Driver 为 DriverNone，调用方应跳过此工厂。
func DatabaseInit(cfg xOptionDB.Config) xRegNode.Node {
	return func(ctx context.Context) (any, error) {
		log := xLog.WithName(xLog.NamedINIT)
		log.Debug(ctx, "正在连接数据库", slog.String("driver", string(cfg.Driver())))

		gormCfg := &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   cfg.TablePrefix(),
				SingularTable: true,
			},
			Logger: xGormLog.NewSlogLogger(slog.Default().WithGroup(xLog.NamedREPO), xGormLog.GormLoggerConfig{
				SlowThreshold:             200,
				LogLevel:                  xGormLog.LevelInfo,
				Colorful:                  false,
				IgnoreRecordNotFoundError: true,
			}),
		}

		db, err := openByDriver(cfg, gormCfg)
		if err != nil {
			return nil, fmt.Errorf("连接数据库失败: %w", err)
		}

		if err = applyPool(db, cfg.Common()); err != nil {
			return nil, fmt.Errorf("配置数据库连接池失败: %w", err)
		}

		log.Info(ctx, "数据库连接成功", slog.String("driver", string(cfg.Driver())))
		return db, nil
	}
}

// openByDriver 按 DatabaseConfig.Driver 选择对应的 GORM 驱动打开连接。
func openByDriver(cfg xOptionDB.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
	dsn := cfg.DSN()
	switch cfg.Driver() {
	case xOptionDB.DriverMySQL:
		return gorm.Open(mysql.Open(dsn), gormCfg)
	case xOptionDB.DriverPostgres:
		return gorm.Open(postgres.Open(dsn), gormCfg)
	case xOptionDB.DriverSQLite:
		return gorm.Open(sqlite.Open(dsn), gormCfg)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Driver())
	}
}

// applyPool 将 DBCommonOptions 应用到底层 *sql.DB 连接池。
//
// 零值字段表示使用默认值，不强制覆盖。
func applyPool(db *gorm.DB, c xOptionDB.CommonOptions) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if c.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	}
	if c.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(c.ConnMaxLifetime)
	}
	if c.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(c.ConnMaxIdleTime)
	}
	return nil
}
