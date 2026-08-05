package xInit

import (
	"context"
	"fmt"
	"log/slog"

	xDB "github.com/bamboo-services/bamboo-base-go/common/database"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xGormLog "github.com/bamboo-services/bamboo-base-go/major/log"
	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
	xRegNode "github.com/bamboo-services/bamboo-base-go/major/register/node"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DatabaseInit 根据传入的 [xOptDatabase.DatabaseConfig] 构造数据库初始化节点。
//
// 返回的 [xRegNode.Node] 不内置任何数据库驱动，驱动来源按优先级取自：
//  1. [xOptDatabase.DatabaseConfig.Dialector]（[xOptDatabase.WithDialector] 显式注入的自定义驱动）；
//  2. common 层驱动注册表（[xDB.GetDialectorFactory]），按 Driver 枚举解析对应驱动插件
//     （plugins/database/*）在 init() 中注册的 Dialector 工厂。
//
// 连接打开后用项目自带的 [xGormLog.SlogLogger] 作为 GORM 日志适配器，并按 Common() 中的
// 连接池参数配置底层 *sql.DB。
//
// 调用方：Register 在 Use 阶段按 option 决定是否装配此节点。
// 若 Driver 为 DriverNone 且未注入 Dialector，调用方应跳过此工厂。
func DatabaseInit(cfg xOptDatabase.DatabaseConfig) xRegNode.Node {
	return func(ctx context.Context) (any, error) {
		log := xLog.WithName(xLog.NamedINIT)
		log.Debug(ctx, "正在连接数据库", slog.String("driver", driverLabel(cfg)))

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

		// AutoMigrate：声明的表自动建表
		if tables := cfg.AutoMigrateTables(); len(tables) > 0 {
			if err := db.AutoMigrate(tables...); err != nil {
				return nil, fmt.Errorf("数据库表自动迁移失败: %w", err)
			}
			log.Debug(ctx, "数据库表自动迁移完成", slog.Int("tables", len(tables)))
		}
		// Prepare：建表后数据初始化，按注册顺序执行
		for i, fn := range cfg.Prepares() {
			if err := fn(ctx, db); err != nil {
				return nil, fmt.Errorf("数据初始化失败(第%d个): %w", i+1, err)
			}
		}

		log.Info(ctx, "数据库连接成功", slog.String("driver", driverLabel(cfg)))
		return db, nil
	}
}

// driverLabel 计算用于日志展示的驱动标识。
//
// 显式注入 Dialector 但未设置驱动枚举时（[xOptDatabase.WithDialector] 自定义驱动场景），
// driver 为空串，回退使用 Dialector.Name() 作为展示名。
func driverLabel(cfg xOptDatabase.DatabaseConfig) string {
	if driver := string(cfg.Driver()); driver != "" {
		return driver
	}
	if d := cfg.Dialector(); d != nil {
		return d.Name()
	}
	return ""
}

// openByDriver 打开 GORM 连接，按优先级选择驱动来源：
//  1. [xOptDatabase.DatabaseConfig.Dialector] 非 nil（[xOptDatabase.WithDialector] 显式注入）；
//  2. 否则按 Driver 枚举从 common 层驱动注册表解析 Dialector 工厂
//     （[xOptDatabase.WithDriver] / [xOptDatabase.FromEnv] 路径，需业务端 import 对应驱动插件触发注册）。
func openByDriver(cfg xOptDatabase.DatabaseConfig, gormCfg *gorm.Config) (*gorm.DB, error) {
	if d := cfg.Dialector(); d != nil {
		return gorm.Open(d, gormCfg)
	}
	factory, ok := xDB.GetDialectorFactory(cfg.Driver())
	if !ok {
		return nil, fmt.Errorf(
			"不支持的数据库驱动: %s（请 import 对应驱动插件，如 github.com/bamboo-services/bamboo-base-go/plugins/database/mysql）",
			cfg.Driver(),
		)
	}
	return gorm.Open(factory(cfg.DSN()), gormCfg)
}

// applyPool 将 DBCommonOptions 应用到底层 *sql.DB 连接池。
//
// 零值字段表示使用默认值，不强制覆盖。
func applyPool(db *gorm.DB, c xOptDatabase.CommonOptions) error {
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
