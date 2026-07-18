# init 知识库

## 概述

内置初始化节点集合，存放 `Register()` 注册的系统级组件初始化函数。当前包含雪花算法、缓存（Redis/Memory）和数据库（MySQL/Postgres/SQLite）三种初始化节点。

## 目录结构

```text
init/
├── init_snowflake.go    # SnowflakeInit() — 雪花算法默认节点初始化与验证
├── init_cache.go        # CacheInit() — 缓存管理器初始化（Redis/Memory 双后端）
└── init_database.go     # DatabaseInit() — 数据库初始化（MySQL/Postgres/SQLite 三驱动）
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 雪花算法初始化 | `init_snowflake.go` → `SnowflakeInit()` | 注册到 `SnowflakeNodeKey`，生成测试 ID 验证可用性 |
| 缓存管理器初始化 | `init_cache.go` → `CacheInit(cfg)` | 根据 `CacheConfig.Type()` 选择 Redis/Memory 后端，返回 `*xCache.Manager` |
| 数据库初始化 | `init_database.go` → `DatabaseInit(cfg)` | 根据 `Config.Driver()` 选择 MySQL/Postgres/SQLite 驱动，返回 `*gorm.DB` |
| Redis 兼容性桥接 | `init_cache.go` → `RedisClientFromManager()` | 从 Manager 提取 `*redis.Client` 补注册到 `RedisClientKey`，兼容历史代码 |
| 添加新的内置节点 | 在此目录新建文件，遵循 `XxxInit(...)` 签名 | 然后在 `register.go` 的 `Register()` 中通过 `reg.Init.Use()` 注册 |

## 约定

- **函数签名灵活**：`SnowflakeInit()` 为 `Node func(ctx) (any, error)` 的标准签名；`CacheInit` 和 `DatabaseInit` 为工厂函数，接收配置参数返回 `Node`。
- **返回值会被存入 context**：返回的实例通过 `context.WithValue(ctx, key, val)` 供后续节点和请求中间件使用。
- **初始化阶段用 `xLog.NamedINIT` 命名日志器**：`xLog.WithName(xLog.NamedINIT)`，日志会标记 `[INIT]` 前缀。
- **验证后才返回**：雪花算法初始化时会生成测试 ID 验证节点可用性；Redis 初始化会 Ping 验证连通性，不验证直接返回可能掩盖问题。
- **ContextKey 由调用方决定**：`CacheInit` / `DatabaseInit` 只返回实例，由 Runner 侧的 `UseAfterExec` 决定注册到哪个 `ContextKey`（`CacheManagerKey` / `DatabaseKey`）。
- **RedisClientFromManager 是兼容性节点**：仅供 Redis 后端用于把 `*redis.Client` 补注册到 `RedisClientKey`，保持 `GetRDB/MustGetRDB` 的兼容性。若 Manager 不存在或后端非 Redis，返回 nil。

## 反模式

- **禁止在此目录放非初始化逻辑** — 此目录的职责是"组件初始化"，不是业务逻辑或工具函数。
- **禁止返回未经验证的实例** — 初始化失败要返回 error，不要返回 nil 实例。
- **禁止在 `CacheTypeNone` 时调用 `CacheInit`** — 调用方（Runner）应先在 Option 中判断，未启用则跳过此工厂。
- **禁止在 `DriverNone` 时调用 `DatabaseInit`** — 同上，调用方应提前跳过。

## 调试路径

1. 雪花算法初始化失败 — 检查 `SNOWFLAKE_DATACENTER_ID` 和 `SNOWFLAKE_NODE_ID` 环境变量是否在 0-31 范围内。
2. 生成的 ID 重复 — 确认多实例部署时 datacenter_id 和 node_id 组合唯一。
3. Redis 初始化失败（Ping 不通） — 检查 `NOSQL_HOST` / `NOSQL_PORT` / `NOSQL_DATABASE` 环境变量和网络连通性。
4. 数据库初始化失败 — 检查 `DATABASE_DRIVER` + DSN 拼装是否正确，连接池参数是否合理。
5. `RedisClientFromManager` 返回 nil — 确认 Manager 已成功注入到 `CacheManagerKey`，且后端确实为 Redis。
