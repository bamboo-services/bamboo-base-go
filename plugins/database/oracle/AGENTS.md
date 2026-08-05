# oracle 数据库驱动插件

## 概述

Oracle 数据库驱动插件（`github.com/bamboo-services/bamboo-base-go/plugins/database/oracle`）。在 `init()` 中将 `DriverOracle` → `oracle.Open` 工厂注册到 common 层驱动注册表，使 major 框架的 `FromEnv` / `WithDriver` 可解析 Oracle 驱动。底层为 `github.com/oracle-samples/gorm-oracle`（依赖 godror，需 **CGO + Oracle Instant Client**）。

业务端使用（blank import 触发注册）：

```go
import _ "github.com/bamboo-services/bamboo-base-go/plugins/database/oracle"

xOption.WithDatabase(xOptDatabase.WithDriver(xOptDatabase.DriverOracle, `user="scott" password="tiger" connectString="..."`))
// 或从环境变量装配（DATABASE_DRIVER=oracle）
xOption.WithDatabase(xOptDatabase.FromEnv())
```

## 约定

- 仅含 `init()` 注册逻辑，不导出业务符号。
- 依赖 `common`（驱动注册表）+ `gorm-oracle`，不依赖 `major`，**且不得被其他插件依赖**——否则会把 godror/CGO 传染给所有驱动使用者。
- 依赖 gorm-oracle v1.1.3，其 go.mod 声明 `go 1.25.11`（MVS 强制本插件 go 版本不低于此），构建环境需 Go 1.25.11+ 且安装 Oracle Instant Client（CGO）。
- 仅当业务端真正需要 Oracle 时才应引入本插件，避免将 CGO 工具链强加给其他应用。
