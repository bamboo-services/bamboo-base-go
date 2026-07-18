# option 知识库

## 概述

Runner 启动阶段的声明式配置层，采用**函数式选项模式（Functional Options）**。业务侧用一行 `WithXxx()` 选择内置组件的实现（Redis/MySQL/Postgres/SQLite/路由），Runner 自动完成装配，无需手写初始化节点。

## 目录结构

```text
option/
├── option.go              # Option 类型 + Config 聚合体 + Apply 入口
├── cache.go               # CacheConfig + RedisOptions + MemoryOptions + 二级选项
├── database.go            # 数据库桥接层：WithDatabase / WithMySQL / WithPostgres / WithSQLite / WithDatabaseFromEnv
├── router.go              # RouteRegistrar + WithRoute + WithRouteGroup
└── database/              # 数据库配置子包（独立，避免循环依赖）
    ├── database.go        #   Driver + Config + CommonOptions + FromEnv
    ├── mysql.go           #   MySQL() + MySQLFromEnv()
    ├── postgres.go        #   Postgres() + PostgresFromEnv()
    └── sqlite.go          #   SQLite() + SQLiteFromEnv()
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 启用 Redis 缓存 | `cache.go` → `WithRedis(addr, opts...)` | Runner 自动装配 Redis 缓存管理器 |
| 启用内存缓存 | `cache.go` → `WithMemory(opts...)` | Runner 自动装配内存缓存管理器 |
| 启用 MySQL | `database.go` → `WithMySQL(dsn, opts...)` | Runner 自动装配 MySQL + GORM |
| 启用 Postgres | `database.go` → `WithPostgres(dsn, opts...)` | Runner 自动装配 Postgres + GORM |
| 启用 SQLite | `database.go` → `WithSQLite(dsn, opts...)` | Runner 自动装配 SQLite + GORM |
| 从环境变量装配数据库 | `database.go` → `WithDatabaseFromEnv(opts...)` | 自动读取 `DATABASE_DRIVER` + DSN 拼装 |
| 注册 HTTP 路由 | `router.go` → `WithRoute(func(serve *gin.Engine))` | 可叠加多个注册器，按顺序执行 |
| 注册路由组 | `router.go` → `WithRouteGroup(prefix, func(*gin.RouterGroup))` | `WithRoute` 的语法糖 |

## 核心类型

### Option 类型

```go
type Option func(*Config)
```

`Config` 是聚合体，字段全小写，仅通过 getter 暴露：

| 方法 | 返回 |
|------|------|
| `Config.Cache()` | `CacheConfig` |
| `Config.Database()` | `xOptionDB.Config` |
| `Config.Routes()` | `[]RouteRegistrar` |

### CacheConfig

| 选项函数 | 说明 |
|----------|------|
| `WithRedis(addr, opts...)` | 启用 Redis，可级联 `WithRedisPassword` / `WithRedisDB` 等二级选项 |
| `WithMemory(opts...)` | 启用内存，可级联 `WithMemoryDefaultTTL` / `WithMemoryMaxEntries` / `WithMemoryShardCount` |

Redis 二级选项：`WithRedisUsername` / `WithRedisPassword` / `WithRedisDB` / `WithRedisPoolSize` / `WithRedisMinIdleConns` / `WithRedisDialTimeout` / `WithRedisReadTimeout` / `WithRedisWriteTimeout`

Memory 二级选项：`WithMemoryDefaultTTL(d)` / `WithMemoryMaxEntries(n)` / `WithMemoryShardCount(n)`

### DatabaseConfig（database 子包）

| 构造函数 | 说明 |
|----------|------|
| `database.MySQL(dsn, opts...)` | 构造 MySQL Config |
| `database.Postgres(dsn, opts...)` | 构造 Postgres Config |
| `database.SQLite(dsn, opts...)` | 构造 SQLite Config |
| `database.FromEnv(opts...)` | 从 `DATABASE_DRIVER` + 分项 env 自动装配 |

连接池二级选项（`CommonOption`）：`WithMaxOpenConns` / `WithMaxIdleConns` / `WithConnMaxLifetime` / `WithConnMaxIdleTime`

## 约定

- **字段全小写只读**：`Config` / `CacheConfig` / `database.Config` 的字段均为小写，通过 getter 对外暴露，避免外部误改。
- **nil Option 安全**：`Apply` 和 `WithRoute` 都会跳过 nil，支持条件构造（如 `cond && WithRedis(...)`）。
- **零值 = 未启用**：未设置时 Type/Driver 为空串，等价于 `"none"`，Runner 据此跳过装配。
- **二级选项模式**：所有参数较多的配置拆分为一级选项 + 二级选项（`RedisOption` / `MemoryOption` / `CommonOption`），避免参数列表爆炸。
- **database 是独立子包**：不 import option 父包，避免循环依赖。

## 反模式

- **禁止直接修改 Config 字段** — 字段小写不可导出，应通过 `WithXxx()` 选项函数构造。
- **禁止手写 DSN 字符串绕过 FromEnv** — 使用 `WithDatabaseFromEnv` 从环境变量装配，便于配置管理和多环境切换。
- **禁止在 `WithRoute` / `WithRouteGroup` 中注册非 HTTP 逻辑** — 路由注册器应专注于 Gin 路由绑定。
- **禁止混用 `WithRedis` 和 `WithMemory`** — 同一 Runner 只能启用一种缓存后端。

## 调试路径

1. Runner 没有装配缓存/数据库 — 检查 Option 是否正确传入 `Runner()`，以及 `CacheConfig.Enabled()` / `Config.Enabled()` 是否返回 true。
2. 启动时 panic "不支持的缓存类型" / "不支持的数据库驱动" — 检查传入的 Type/Driver 值和常量定义是否匹配。
3. DSN 拼装错误 — 使用 `FromEnv` 时检查所有相关环境变量（`DATABASE_HOST` / `DATABASE_USER` 等）是否有值。
4. 路由未生效 — 确认 `WithRoute` 注册的函数被执行（Runner 按序调用），检查 Gin 路由路径是否重复。
