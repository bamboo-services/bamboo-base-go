package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/gormlog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// PluginKey 是数据库插件在应用组件容器中的权威标识。
const PluginKey ctxkey.Key = "plugin_database"

// 数据库插件启动阶段的配置类失败哨兵。
//
// 哨兵作为错误链根因保留，调用方仍可通过 errors.Is 精确判定失败语义；
// 对外暴露的失败结果统一为携带 [DatabaseError] 的 [*apperr.Error]。
var (
	// errAdapterMissing 表示既未显式传入适配器，也未启用 FromEnv 环境变量解析。
	errAdapterMissing = errors.New("database: 未指定数据库适配器，请传入 WithAdapter 或使用 FromEnv")

	// errDriverUnset 表示启用 FromEnv 但环境变量 DATABASE_DRIVER 缺失或为空。
	errDriverUnset = errors.New("database: 环境变量 DATABASE_DRIVER 未配置")
)

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
//
// 失败结果为 [*apperr.Error]：成功必须返回真正的 nil 指针；
// 取消与超时类原因经 [foundation/v2/control] 归一，原生 context 根因保留；
// 其余失败按所属域错误码 [DatabaseError] 兜底，GORM 与驱动原生错误继续作为根因保留。
func (p *Plugin) Start(ctx context.Context) (retErr *apperr.Error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	adapter := p.cfg.adapter
	if adapter == nil {
		if !p.cfg.fromEnv {
			return lifecycleErr("database: 缺少数据库适配器", errAdapterMissing)
		}
		rawDriver, ok := env.Lookup(env.DatabaseDriver)
		if !ok || strings.TrimSpace(rawDriver.String()) == "" {
			return lifecycleErr("database: 缺少数据库驱动名称", errDriverUnset)
		}
		driverName := dialect.Driver(strings.TrimSpace(rawDriver.String()))
		adp, err := dialect.Lookup(driverName)
		if err != nil {
			return lifecycleErr("database: 查找数据库方言适配器失败", fmt.Errorf("数据库驱动 [%s] 未加载，请传入 WithAdapter(%s.FromEnv()) 或匿名导入 plugins/database/adapter/%s/v2: %w", driverName, driverName, driverName, err))
		}
		adapter = adp
	}

	dialector, err := adapter.Open()
	if err != nil {
		return lifecycleErr("database: 构造方言 Dialector 失败", err)
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
		return lifecycleErr("database: gorm.Open 失败", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return lifecycleErr("database: 获取底层 sql.DB 失败", err)
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
		return lifecycleErr("database: 验证数据库连通性失败", err)
	}

	return nil
}

// Stop 在受控关闭阶段安全释放底层 [*sql.DB] 连接池资源。
//
// 该方法具备互斥锁保护与幂等性，多次调用不会引发重复关闭或错误。
//
// 失败结果为 [*apperr.Error]：成功必须返回真正的 nil 指针；
// 关闭失败按所属域错误码 [DatabaseError] 归一，底层 sql.DB 原生错误作为根因保留。
func (p *Plugin) Stop(_ context.Context) *apperr.Error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeSQLDB()
}

// closeSQLDB 关闭底层连接池并清空句柄，关闭失败归一为 [*apperr.Error]。
func (p *Plugin) closeSQLDB() *apperr.Error {
	if p.sqlDB != nil {
		err := p.sqlDB.Close()
		p.sqlDB = nil
		p.db = nil
		if err != nil {
			return lifecycleErr("database: 关闭数据库连接池失败", err)
		}
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
