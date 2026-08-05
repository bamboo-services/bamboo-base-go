# postgres 数据库驱动插件

## 概述

PostgreSQL 数据库驱动插件（`github.com/bamboo-services/bamboo-base-go/plugins/database/postgres`）。在 `init()` 中将 `DriverPostgres` → `postgres.Open` 工厂注册到 common 层驱动注册表，使 major 框架的 `FromEnv` / `WithDriver` 可解析 PostgreSQL 驱动。

业务端使用（blank import 触发注册）：

```go
import _ "github.com/bamboo-services/bamboo-base-go/plugins/database/postgres"

xOption.WithDatabase(xOptDatabase.WithDriver(xOptDatabase.DriverPostgres, "host=... dbname=..."))
// 或从环境变量装配（DATABASE_DRIVER=postgres）
xOption.WithDatabase(xOptDatabase.FromEnv())
```

## 约定

- 仅含 `init()` 注册逻辑，不导出业务符号（对齐 database/sql 驱动惯例）。
- 依赖 `common`（驱动注册表）+ `gorm.io/driver/postgres`，不依赖 `major`。
- import 本插件即把 PostgreSQL 驱动编入二进制；未使用请不要引入。
