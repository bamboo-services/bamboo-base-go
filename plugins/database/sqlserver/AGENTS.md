# sqlserver 数据库驱动插件

## 概述

SQL Server 数据库驱动插件（`github.com/bamboo-services/bamboo-base-go/plugins/database/sqlserver`）。在 `init()` 中将 `DriverSQLServer` → `sqlserver.Open` 工厂注册到 common 层驱动注册表，使 major 框架的 `FromEnv` / `WithDriver` 可解析 SQL Server 驱动。底层为 `gorm.io/driver/sqlserver`（依赖 microsoft/go-mssqldb）。

业务端使用（blank import 触发注册）：

```go
import _ "github.com/bamboo-services/bamboo-base-go/plugins/database/sqlserver"

xOption.WithDatabase(xOptDatabase.WithDriver(xOptDatabase.DriverSQLServer, "sqlserver://sa:pass@host:1433?database=db"))
// 或从环境变量装配（DATABASE_DRIVER=sqlserver）
xOption.WithDatabase(xOptDatabase.FromEnv())
```

## 约定

- 仅含 `init()` 注册逻辑，不导出业务符号（对齐 database/sql 驱动惯例）。
- 依赖 `common`（驱动注册表）+ `gorm.io/driver/sqlserver`，不依赖 `major`。
- import 本插件即把 SQL Server 驱动编入二进制；未使用请不要引入。
