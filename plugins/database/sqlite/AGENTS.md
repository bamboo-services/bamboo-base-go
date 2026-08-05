# sqlite 数据库驱动插件

## 概述

SQLite 数据库驱动插件（`github.com/bamboo-services/bamboo-base-go/plugins/database/sqlite`）。在 `init()` 中将 `DriverSQLite` → `sqlite.Open` 工厂注册到 common 层驱动注册表，使 major 框架的 `FromEnv` / `WithDriver` 可解析 SQLite 驱动。底层为 `github.com/libtnb/sqlite`（基于 modernc.org/sqlite 的纯 Go 实现，无 CGO）。

业务端使用（blank import 触发注册）：

```go
import _ "github.com/bamboo-services/bamboo-base-go/plugins/database/sqlite"

// 显式指定（:memory: 仅用于开发/测试）
xOption.WithDatabase(xOptDatabase.WithDriver(xOptDatabase.DriverSQLite, "app.db"))
// 或从环境变量装配（DATABASE_DRIVER=sqlite，DATABASE_PATH 默认 :memory:）
xOption.WithDatabase(xOptDatabase.FromEnv())
```

## 约定

- 依赖 `common`（驱动注册表）+ `libtnb/sqlite`，**不依赖 `major`**。
- import 本插件即把 SQLite 驱动编入二进制；未使用请不要引入。
