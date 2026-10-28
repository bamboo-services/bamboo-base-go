package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/gormlog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// PluginKey 是数据库插件在应用组件容器中的权威标识。
const PluginKey ctxkey.Key = "plugin_database"

// Plugin 是数据库插件运行时实现，提供数据库连接池管理与生命周期协调。
type Plugin struct {
	cfg   config
	db    *gorm.DB
	sqlDB *sql.DB
	mu    sync.Mutex
}

// New 接收配置选项构造数据库插件元信息 [application.PluginInfo] 与工厂函数。
//
// 该函数与返回的工厂函数均为纯构造函数，不触发环境变量解析、不产生任何网络 I/O，
// 且不提前建立物理数据库连接。实际连接、连接池配置与连通性验证推迟至生命周期的 Start 阶段执行。
func New(opts ...Option) (application.PluginInfo, func() application.Plugin) {
	var cfg config
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	info := application.PluginInfo{
		Key:      PluginKey,
		Name:     "database",
		Mode:     application.ModeTool,
		Provides: []ctxkey.Key{ctxkey.DatabaseKey},
		Requires: nil,
	}

	factory := func() application.Plugin {
		return &Plugin{
			cfg: cfg,
		}
	}

	return info, factory
}

// Start 在受控启动上下文中初始化数据库方言、创建 GORM 实例、配置连接池并执行连通性探测。
//
// 若未显式传入适配器且启用了 FromEnv，Start 将从环境变量 DATABASE_DRIVER 中读取驱动名称
// 并从全局方言注册表中查找对应适配器。连接池成功创建后将调用底层 sql.DB.PingContext(ctx)
// 验证连通性；若探测失败或启动超时，Start 会自动关闭已初始化的底层连接并返回错误。
func (p *Plugin) Start(ctx context.Context) (retErr error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	adapter := p.cfg.adapter
	if adapter == nil {
		if !p.cfg.fromEnv {
			return errors.New("database: 未指定数据库适配器，请传入 WithAdapter 或使用 FromEnv")
		}
		rawDriver, ok := env.Lookup(env.DatabaseDriver)
		if !ok || strings.TrimSpace(rawDriver.String()) == "" {
			return errors.New("database: 环境变量 DATABASE_DRIVER 未配置")
		}
		driverName := dialect.Driver(strings.TrimSpace(rawDriver.String()))
		adp, err := dialect.Lookup(driverName)
		if err != nil {
			return fmt.Errorf("数据库驱动 [%s] 未加载，请传入 WithAdapter(%s.FromEnv()) 或匿名导入 plugins/database/adapter/%s/v2", driverName, driverName, driverName)
		}
		adapter = adp
	}

	dialector, err := adapter.Open()
	if err != nil {
		return fmt.Errorf("database: 构造方言 Dialector 失败: %w", err)
	}

	var gormCfg *gorm.Config
	if p.cfg.gormConfig != nil {
		clone := *p.cfg.gormConfig
		gormCfg = &clone
	} else {
		gormCfg = &gorm.Config{}
	}

	gormCfg.DisableAutomaticPing = true
	if p.cfg.logger != nil {
		gormCfg.Logger = gormlog.New(p.cfg.logger, p.cfg.logConfig)
	} else if gormCfg.Logger == nil {
		gormCfg.Logger = gormlogger.Discard
	}

	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return fmt.Errorf("database: gorm.Open 失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: 获取底层 sql.DB 失败: %w", err)
	}
	p.sqlDB = sqlDB
	p.db = db

	defer func() {
		if retErr != nil {
			_ = p.closeSQLDB()
		}
	}()

	if p.cfg.maxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(p.cfg.maxOpenConns)
	}
	if p.cfg.maxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(p.cfg.maxIdleConns)
	}
	if p.cfg.connMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(p.cfg.connMaxLifetime)
	}
	if p.cfg.connMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(p.cfg.connMaxIdleTime)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database: 验证数据库连通性失败: %w", err)
	}

	return nil
}

// Stop 在受控关闭阶段安全释放底层 [*sql.DB] 连接池资源。
//
// 该方法具备互斥锁保护与幂等性，多次调用不会引发重复关闭或错误。
func (p *Plugin) Stop(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeSQLDB()
}

func (p *Plugin) closeSQLDB() error {
	if p.sqlDB != nil {
		err := p.sqlDB.Close()
		p.sqlDB = nil
		p.db = nil
		return err
	}
	return nil
}

// Expose 交付数据库插件向容器导出的共享对象映射表。
//
// 插件启动成功后交付 [*gorm.DB] 句柄，导出键为 [ctxkey.DatabaseKey]。
func (p *Plugin) Expose() map[ctxkey.Key]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	res := make(map[ctxkey.Key]any)
	if p.db != nil {
		res[ctxkey.DatabaseKey] = p.db
	}
	return res
}
