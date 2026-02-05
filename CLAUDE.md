# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

本文件为 Claude Code (claude.ai/code) 提供在此代码库中工作的指导说明。

## ⚠️ 重要架构变更（v2.0）

从 v2.0 版本开始，bamboo-base 进行了重大架构升级：

### 核心变更

1. **注册系统节点化**
   - 从固定初始化流程改为可扩展的节点注册系统
   - 支持依赖注入：每个节点可访问之前初始化的组件
   - 节点函数签名：`func(ctx context.Context) (any, error)`

2. **上下文管理标准化**
   - 所有工具函数从 `gin.Context` 迁移到标准 `context.Context`
   - 实现框架解耦，业务逻辑不再依赖 Gin
   - 新增 `Must` 和 `Error` 两种版本的访问函数

3. **初始化流程优化**
   - 配置和日志初始化改为私有方法
   - 雪花算法等内置组件自动注册
   - 支持自定义节点动态注册

### 迁移指南

**旧版本 (v1.x)**：
```go
reg := xReg.Register()
db := xCtxUtil.GetDB(ginCtx)
```

**新版本 (v2.0+)**：
```go
reg := xReg.Register(ctx, nodeList)
db := xCtxUtil.MustGetDB(ginCtx.Request.Context())
// 或
db, err := xCtxUtil.GetDB(ginCtx.Request.Context())
```

## 项目概述

这是 `bamboo-base-go`，一个为 bamboo 服务提供基础组件的 Go 语言库。它被设计为一个可重用的基础库，用于构建基于 Gin 框架的 Web API，提供标准化的错误处理、日志记录、配置管理和响应格式化功能。

## 架构设计

### 核心组件

- **注册系统** (`register/`): 集中式注册系统，负责引导应用程序启动 (包名: `xReg`)
  - `register.go` 包含主要的 `Register()` 函数，初始化所有组件
  - `node/node.go` 提供节点化管理系统，支持依赖注入和顺序初始化
  - `init/` 目录包含内置组件的初始化函数（如雪花算法）
  - 处理配置加载、日志器设置、Gin 引擎初始化和系统上下文设置
  - **重要特性**：支持自定义节点注册，每个节点可访问之前初始化的组件

- **环境变量管理** (`env/`): 类型安全的环境变量管理系统 (包名: `xEnv`)
  - `env.go` 定义 `EnvKey` 类型和所有环境变量常量
  - `utility.go` 提供类型安全的获取函数 (`GetEnvString`、`GetEnvInt`、`GetEnvBool` 等)

- **上下文管理** (`context/`): 上下文键常量定义 (包名: `xCtx`)
  - `context.go` 定义 `ContextKey` 类型和所有上下文键常量
  - 新增 `Nil` 和 `Exec` 特殊键，用于节点注册系统
  - 提供 `IsNil()` 和 `IsExec()` 辅助方法

- **错误处理** (`error/`): 全面的错误管理系统 (包名: `xError`)
  - `ErrorInterface` 定义标准错误接口合约
  - `Error` 结构体实现包含代码、消息和数据的结构化错误
  - `ErrorCode` 提供预定义的错误常量

- **响应系统**: 标准化 API 响应结构
  - `BaseResponse` 在 `base_response.go` 中定义通用响应格式
  - `result/` 包处理响应格式化 (包名: `xResult`)
  - `middleware/response.go` 提供统一响应中间件 (包名: `xMiddle`)

- **路由处理** (`route/`): 路由相关处理器 (包名: `xRoute`)
  - `no_route.go` 处理未定义路由的 404 响应
  - `no_method.go` 处理不支持的 HTTP 方法的 405 响应

- **HTTP 常量** (`http/`): HTTP 相关常量定义 (包名: `xHttp`)
  - `header.go` 定义 HTTP 请求头常量 (`HeaderRequestUUID`、`HeaderAuthorization`)

- **配置管理**: 基于环境变量的配置系统
  - 使用 `godotenv` 加载 `.env` 文件
  - 配置通过 `xEnv.GetEnvXxx()` 系列函数获取

- **工具库** (`utility/`): 丰富的通用辅助函数和上下文工具 (包名: `xUtil`)
  - `ctxutil/` 提供与数据库、日志和通用操作相关的上下文工具 (包名: `xCtxUtil`)
  - 基础工具：`Ptr()`、`Val()`、`Contains()`、`ToBool()` 等指针和类型转换
  - 字符串处理：命名转换、数据脱敏、格式验证等字符串操作工具
  - 时间处理：格式化、解析、计算等时间操作工具
  - 数据验证：手机号、身份证、URL、密码强度等验证工具

### 核心依赖

- **Gin 框架**: 用于 HTTP 路由和中间件的 Web 框架
- **slog (Go 标准库)**: Go 1.21+ 标准库结构化日志，支持彩色控制台输出和 JSON 文件记录，自动从 context 提取 trace ID
- **GORM**: 数据库操作的 ORM 工具
- **Validator**: 使用 go-playground/validator 进行请求验证
- **UUID**: Google UUID 用于唯一标识符生成

## 开发命令

### 构建和测试
```bash
# 运行测试
go test ./...

# 运行特定测试
go test ./test -v

# 构建模块
go build

# 格式化代码
go fmt ./...

# 检查代码中的常见问题
go vet ./...

# 运行测试并生成覆盖率报告
go test -cover ./...
```

### 模块管理
```bash
# 整理依赖
go mod tidy

# 验证依赖
go mod verify

# 下载依赖
go mod download
```

## 项目结构

```
bamboo-base/
├── base_response.go          # 标准 API 响应结构
├── context/                 # 上下文键常量定义 (xCtx)
├── env/                     # 环境变量管理 (xEnv)
├── error/                   # 错误处理系统 (xError)
├── go.mod                   # 模块定义和依赖管理 (Go 1.24.6)
├── helper/                  # 辅助工具 (恐慌恢复等)
├── hook/                    # Redis 钩子
├── http/                    # HTTP 常量定义 (xHttp)
├── log/                     # slog 自定义 Handler (xLog)
├── middleware/              # Gin 中间件 (xMiddle)
├── models/                  # 数据模型 (xModels)
├── register/                # 注册和初始化系统 (xReg)
├── result/                  # 响应结果格式化 (xResult)
├── route/                   # 路由处理 (xRoute)
├── snowflake/               # 雪花算法 (xSnowflake)
├── test/                    # 测试文件
├── utility/                 # 通用工具 (xUtil) 和上下文辅助工具 (xCtxUtil)
└── validator/               # 自定义验证逻辑和消息 (xValidator)
```

### 详细模块说明

- **register/**: 应用注册初始化模块 (包名: `xReg`)
  - `register.go`: 主注册函数 `Register(ctx, nodeList)`，返回 `Reg` 结构体
  - `node/node.go`: 节点管理系统，提供 `RegNode` 类型和 `Use()`、`Exec()` 方法
  - `init/init_snowflake.go`: 雪花算法初始化节点
  - `register_config.go`: 配置初始化（私有方法）
  - `register_gin.go`: Gin 引擎初始化（私有方法），包含上下文注入中间件
  - `register_logger.go`: 日志器初始化（私有方法）
  - **核心特性**: 节点化管理、依赖注入、顺序执行、错误传播

- **env/**: 环境变量管理模块 (包名: `xEnv`)
  - `env.go`: 定义 `EnvKey` 类型和所有环境变量常量 (系统、数据库、Redis、雪花算法、日志、第三方服务等)
  - `utility.go`: 类型安全的获取函数 (`GetEnv`、`GetEnvString`、`GetEnvInt`、`GetEnvBool`、`GetEnvFloat`、`GetEnvInt64`、`GetEnvDuration`)

- **context/**: 上下文键常量模块 (包名: `xCtx`)
  - `context.go`: 定义 `ContextKey` 类型和上下文键常量 (`RequestKey`、`ErrorCodeKey`、`DatabaseKey`、`RedisClientKey`、`SnowflakeNodeKey` 等)

- **log/**: slog 自定义 Handler 模块 (包名: `xLog`)
  - `handler.go`: 自定义 slog.Handler 实现，支持彩色控制台输出和 JSON 文件记录，自动从 context 提取 trace ID

- **http/**: HTTP 常量模块 (包名: `xHttp`)
  - `header.go`: 定义 `HttpHeader` 类型和 HTTP 请求头常量 (`HeaderRequestUUID`、`HeaderAuthorization`)

- **route/**: 路由处理模块 (包名: `xRoute`)
  - `no_route.go`: 处理未定义路由的 404 响应
  - `no_method.go`: 处理不支持的 HTTP 方法的 405 响应

- **error/**: 错误处理核心模块 (包名: `xError`)
  - `error.go`: 错误接口和结构体定义
  - `error_code.go`: 预定义错误码常量 (包含 40+ 种错误类型)
  - `error_new.go`: 错误构造函数

- **models/**: 数据模型模块 (包名: `xModels`)
  - `base_entity.go`: GORM 实体基类 (`BaseEntity`、`GeneBaseEntity`)

- **snowflake/**: 雪花算法模块 (包名: `xSnowflake`)
  - `snowflake.go`: 标准雪花 ID (41位时间戳 + 5位数据中心 + 5位节点 + 12位序列)
  - `gene.go`: 基因类型定义 (64种业务类型)
  - `gene_snowflake.go`: 基因雪花 ID (41位时间戳 + 6位基因 + 3位数据中心 + 3位节点 + 10位序列)
  - `global.go`: 全局节点管理 (默认节点初始化)

- **middleware/**: 中间件模块 (包名: `xMiddle`)
  - `response.go`: 统一响应中间件，处理错误和成功响应

- **result/**: 响应处理模块 (包名: `xResult`)
  - `result.go`: 提供 `Success`、`SuccessHasData`、`Error` 三种响应方法

- **utility/**: 工具库模块 (包名: `xUtil`)
  - `common.go`: 基础工具函数 (`Ptr()`、`Val()`、`Contains()`、`ToBool()`)
  - `string.go`: 字符串处理工具 (命名转换、脱敏、验证等)
  - `time.go`: 时间处理工具 (格式化、解析、计算等)
  - `validate.go`: 数据验证工具 (手机号、身份证、URL 等验证)
  - `password.go`: 密码加密工具 (bcrypt 加密、验证等)
  - `generate.go`: 生成工具函数 (`GenerateSecurityKey()`)
  - `ctxutil/`: 上下文工具子模块 (包名: `xCtxUtil`)
    - `common.go`: 上下文通用工具 (调试模式、请求信息等)
    - `database.go`: 数据库上下文工具
    - `nosql.go`: Redis 上下文工具
    - `snowflake.go`: 雪花算法上下文工具

- **validator/**: 验证模块 (包名: `xValidator`)
  - `custom.go`: 自定义验证规则（7个验证器：strict_url、strict_uuid、alphanum_underscore、regexp、enum_int、enum_string、enum_float）
  - `messages.go`: 验证错误消息
  - `response.go`: 验证响应处理
  - `translator.go`: 中文翻译器
  - `validator_enum_int.go`: 整数枚举验证器
  - `validator_enum_string.go`: 字符串枚举验证器
  - `validator_enum_float.go`: 浮点数枚举验证器
  - `validator_url.go`: URL 验证器
  - `validator_uuid.go`: UUID 验证器
  - `validator_alphanum.go`: 字母数字验证器
  - `validator_regexp.go`: 正则表达式验证器

- **hook/**: 钩子模块
  - Redis 钩子相关功能

## 使用模式

### 节点注册系统（重要！）

bamboo-base 采用节点化的初始化系统，支持依赖注入和顺序执行。每个节点都是一个初始化函数，可以访问之前已初始化的组件。

#### 节点函数签名

```go
type Node func(ctx context.Context) (any, error)
```

**参数说明**：
- `ctx`: 包含已注册依赖的上下文，可通过 `ctx.Value(key)` 获取其他组件实例
- 返回值：初始化成功的组件实例和可能的错误

#### 注册流程

```go
// 1. 创建上下文
ctx := context.Background()

// 2. 注册自定义节点
nodeList := []xRegNode.RegNodeList{
    // 数据库初始化节点
    {
        Key: xCtx.DatabaseKey,
        Node: func(ctx context.Context) (any, error) {
            // 可以从 ctx 获取配置等已初始化的组件
            dsn := "user:pass@tcp(127.0.0.1:3306)/dbname"
            db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
            if err != nil {
                return nil, err
            }
            return db, nil
        },
    },
    // Redis 初始化节点
    {
        Key: xCtx.RedisClientKey,
        Node: func(ctx context.Context) (any, error) {
            rdb := redis.NewClient(&redis.Options{
                Addr: "localhost:6379",
            })
            return rdb, nil
        },
    },
}

// 3. 执行注册
reg := xReg.Register(ctx, nodeList)

// 4. 使用初始化后的组件
engine := reg.Serve
engine.Run(":8080")
```

#### 内置节点

系统自动注册以下内置节点：
- **雪花算法节点** (`xCtx.SnowflakeNodeKey`): 自动初始化，无需手动注册

#### 节点执行顺序

节点按注册顺序依次执行：
1. 配置加载 (自动)
2. 日志器初始化 (自动)
3. 雪花算法初始化 (自动)
4. 用户自定义节点 (按 nodeList 顺序)
5. Gin 引擎初始化 (自动)

**重要**：后注册的节点可以通过 `ctx.Value()` 访问先注册节点的实例。

#### 特殊执行节点

如果需要执行某些逻辑但不需要将结果存入上下文，可以使用 `xCtx.Exec` 键：

```go
{
    Key: xCtx.Exec,
    Node: func(ctx context.Context) (any, error) {
        // 执行一些初始化逻辑，但不存储结果
        log.Println("执行特殊初始化...")
        return nil, nil
    },
}
```

### 初始化应用程序
```go
// 创建注册实例并初始化所有组件
ctx := context.Background()
reg := xReg.Register(ctx, nil)  // nil 表示不注册额外的自定义节点

// 或者注册自定义节点
reg := xReg.Register(ctx, []xRegNode.RegNodeList{
    {Key: xCtx.DatabaseKey, Node: myDatabaseInit},
    {Key: xCtx.RedisClientKey, Node: myRedisInit},
})

// 访问初始化后的组件
engine := reg.Serve      // *gin.Engine - HTTP 服务引擎

// 启动服务器
engine.Run(":8080")
```

### 错误处理模式
```go
// 使用预定义错误码创建错误
err := xError.New(xError.ParameterError, "用户ID不能为空", nil)

// 在处理器中返回错误 (由响应中间件自动格式化)
ctx.Error(err)
return
```

### 响应处理模式
```go
// 成功响应
xResult.Success(ctx, "操作成功")

// 带数据的成功响应
xResult.SuccessHasData(ctx, "获取数据成功", userData)

// 错误响应
xResult.Error(ctx, xError.NotFound, "用户不存在", nil)
```

### 配置管理

应用程序通过环境变量配置，支持 `.env` 文件加载。使用 `xEnv` 包的类型安全函数获取配置：

```go
import xEnv "github.com/bamboo-services/bamboo-base-go/env"

// 获取配置 (类型安全)
debug := xEnv.GetEnvBool(xEnv.Debug, false)           // 布尔值
dbHost := xEnv.GetEnvString(xEnv.DatabaseHost, "localhost") // 字符串
dbPort := xEnv.GetEnvInt(xEnv.DatabasePort, 3306)    // 整数
timeout := xEnv.GetEnvDuration(xEnv.SomeTimeout, 30) // 时间

// 检查环境变量是否存在
if value, exists := xEnv.GetEnv(xEnv.DatabaseHost); exists {
    // 使用 value
}
```

**常用环境变量 (定义在 `xEnv` 包中)：**

| 环境变量常量 | 环境变量名 | 说明 |
|-------------|-----------|------|
| `xEnv.Debug` | `XLF_DEBUG` | 调试模式 (true/false) |
| `xEnv.Host` | `XLF_HOST` | 监听地址 |
| `xEnv.Port` | `XLF_PORT` | 监听端口 |
| `xEnv.DatabaseHost` | `DATABASE_HOST` | 数据库主机 |
| `xEnv.DatabasePort` | `DATABASE_PORT` | 数据库端口 |
| `xEnv.DatabaseUser` | `DATABASE_USER` | 数据库用户名 |
| `xEnv.DatabasePass` | `DATABASE_PASS` | 数据库密码 |
| `xEnv.DatabaseName` | `DATABASE_NAME` | 数据库名称 |
| `xEnv.NoSqlHost` | `NOSQL_HOST` | Redis 主机地址 |
| `xEnv.NoSqlPort` | `NOSQL_PORT` | Redis 端口 |
| `xEnv.NoSqlPass` | `NOSQL_PASS` | Redis 密码 |
| `xEnv.NoSqlDB` | `NOSQL_DB` | Redis 数据库索引 |
| `xEnv.SnowflakeDatacenterID` | `SNOWFLAKE_DATACENTER_ID` | 雪花算法数据中心 ID (0-31) |
| `xEnv.SnowflakeNodeID` | `SNOWFLAKE_NODE_ID` | 雪花算法节点 ID (0-31) |

**使用方式：**
1. 复制 `.env.example` 为 `.env`
2. 根据需要修改配置项

### 中间件使用
```go
// 应用响应中间件 (通常在初始化时自动应用)
engine.Use(xMiddle.ResponseMiddleware)
```

### 上下文工具使用

**重要变化**：从 v2.0 开始，所有上下文工具函数从 `gin.Context` 迁移到标准 `context.Context`，实现框架解耦。

#### 基础工具

```go
// 检查是否为调试模式（无需上下文）
if xCtxUtil.IsDebugMode() {
    // 调试逻辑
}

// 计算请求处理时间
// 注意：gin.Context 实现了 context.Context 接口，可以直接传入
overhead := xCtxUtil.CalcOverheadTime(ctx)
overhead := xCtxUtil.CalcOverheadTime(ginCtx.Request.Context())  // 从 gin.Context 获取

// 获取请求信息
requestKey := xCtxUtil.GetRequestKey(ctx)
errorMsg := xCtxUtil.GetErrorMessage(ctx)
```

#### 数据库访问（Must vs Error 版本）

系统提供两种版本的数据库访问函数：

```go
// Must 版本：失败时 panic（适合启动阶段或必须成功的场景）
db := xCtxUtil.MustGetDB(ctx)
db.Create(&user)

// Error 版本：返回错误（适合运行时或可容错的场景）
db, err := xCtxUtil.GetDB(ctx)
if err != nil {
    return err
}
db.Create(&user)
```

**在 Gin Handler 中使用**：

```go
func CreateUser(c *gin.Context) {
    // 方式 1: 使用 Must 版本（推荐，简洁）
    db := xCtxUtil.MustGetDB(c.Request.Context())

    // 方式 2: 使用 Error 版本（更安全）
    db, err := xCtxUtil.GetDB(c.Request.Context())
    if err != nil {
        xResult.Error(c, err.ErrorCode, err.ErrorMessage, nil)
        return
    }

    // 使用数据库
    db.Create(&user)
}
```

#### Redis 访问

```go
// Must 版本
rdb := xCtxUtil.MustGetRDB(ctx)
rdb.Set(ctx, "key", "value", 0)

// Error 版本
rdb, err := xCtxUtil.GetRDB(ctx)
if err != nil {
    return err
}
rdb.Set(ctx, "key", "value", 0)
```

#### 雪花 ID 生成

```go
// 获取雪花算法节点（自动回退到默认节点）
snowflakeNode := xCtxUtil.GetSnowflakeNode(ctx)

// 生成标准雪花 ID
id := xCtxUtil.MustGenerateSnowflakeID(ctx)           // panic 版本
id, err := xCtxUtil.GenerateSnowflakeID(ctx)          // error 版本

// 生成带基因的雪花 ID
geneID := xCtxUtil.MustGenerateGeneSnowflakeID(ctx, xSnowflake.GeneOrder)  // panic 版本
geneID, err := xCtxUtil.GenerateGeneSnowflakeID(ctx, xSnowflake.GeneOrder) // error 版本
```

#### 上下文传递最佳实践

```go
// ✅ 推荐：在 Gin Handler 中使用 Request.Context()
func MyHandler(c *gin.Context) {
    ctx := c.Request.Context()  // 获取标准 context.Context

    // 使用上下文工具
    db := xCtxUtil.MustGetDB(ctx)
    id := xCtxUtil.MustGenerateSnowflakeID(ctx)

    // 传递给业务逻辑
    result, err := myService.DoSomething(ctx, id)
}

// ✅ 推荐：业务逻辑使用标准 context.Context
func (s *MyService) DoSomething(ctx context.Context, id int64) error {
    db := xCtxUtil.MustGetDB(ctx)
    rdb := xCtxUtil.MustGetRDB(ctx)
    // ... 业务逻辑
}

// ❌ 不推荐：直接传递 gin.Context 到业务层
func (s *MyService) DoSomething(c *gin.Context, id int64) error {
    // 业务层不应依赖 Gin 框架
}
```

### 日志记录 (slog)
```go
import "log/slog"

// 直接使用 slog，trace ID 会自动从 context 提取
slog.InfoContext(ctx.Request.Context(), "用户登录成功",
    "user_id", userID,
    "ip", ctx.ClientIP(),
)

slog.WarnContext(ctx.Request.Context(), "请求参数异常",
    "param", paramName,
    "error", err,
)

slog.ErrorContext(ctx.Request.Context(), "数据库操作失败",
    "table", "users",
    "error", err,
)

// 控制台输出格式 (彩色):
// 2024-01-15 10:30:45 [INFO] [CORE] [abc123] 用户登录成功 user_id=1001 ip=192.168.1.1

// 文件输出格式 (JSON):
// {"time":"2024-01-15T10:30:45","level":"INFO","trace":"abc123","msg":"用户登录成功","user_id":1001,"ip":"192.168.1.1"}
```

### 雪花算法使用
```go
// 全局函数生成 ID
id := xSnowflake.GenerateID()                           // 标准雪花 ID
geneID := xSnowflake.MustGenerateGeneID(xSnowflake.GeneUser) // 基因雪花 ID

// 使用节点生成
node, _ := xSnowflake.NewNode(1, 1)                     // 数据中心 1, 节点 1
id := node.Generate()

// 使用上下文生成
id := xCtxUtil.GenerateSnowflakeID(ctx)
geneID, _ := xCtxUtil.GenerateGeneSnowflakeID(ctx, xSnowflake.GeneOrder)

// ID 组件提取
timestamp := id.Timestamp()     // 时间戳
datacenter := id.DatacenterID() // 数据中心 ID
nodeID := id.NodeID()           // 节点 ID
sequence := id.Sequence()       // 序列号

// 基因提取
gene := geneID.Gene()           // 业务类型基因
```

### GORM 实体使用
```go
// 使用标准雪花 ID 的实体
type User struct {
    xModels.BaseEntity
    Username string `gorm:"type:varchar(64);uniqueIndex"`
    Email    string `gorm:"type:varchar(128)"`
}

// 使用基因雪花 ID 的实体
type Order struct {
    xModels.GeneBaseEntity
    OrderNo     string  `gorm:"type:varchar(64);uniqueIndex"`
    TotalAmount float64 `gorm:"type:decimal(10,2)"`
}

// 实现 GeneProvider 接口指定基因类型
func (o *Order) GetGene() xSnowflake.Gene {
    return xSnowflake.GeneOrder
}

// 创建记录时自动生成 ID
user := &User{Username: "test"}
db.Create(user) // user.ID 自动生成
```

### 枚举值验证

bamboo-base 提供三种枚举验证器，支持不同的数据类型：

#### 1. enum_int - 整数枚举

对于自定义数值类型的枚举验证：

```go
// 定义自定义数值类型
type UserGender int8

const (
    GenderUnknown UserGender = 0
    GenderMale    UserGender = 1
    GenderFemale  UserGender = 2
)

// 在结构体中使用 enum_int 验证器
type CreateUserRequest struct {
    Username string     `json:"username" binding:"required,min=3,max=64" label:"用户名"`
    Gender   UserGender `json:"gender" binding:"enum_int=0 1 2" label:"性别"`
}

// 支持负数枚举
type OrderStatus int

const (
    StatusCanceled OrderStatus = -1
    StatusPending  OrderStatus = 0
    StatusPaid     OrderStatus = 1
    StatusShipped  OrderStatus = 2
)

type Order struct {
    Status OrderStatus `json:"status" binding:"enum_int=-1 0 1 2" label:"订单状态"`
}
```

**注意事项**:
- `enum_int` 支持所有整数类型（int、int8、int16、int32、int64、uint 等）
- 支持基于整数的自定义类型
- 枚举值使用空格分隔
- 支持负数
- 建议配合 `label` tag 提供友好的字段名称

**错误消息示例**:
```json
{
  "gender": "性别必须是以下值之一: 0 1 2"
}
```

#### 2. enum_string - 字符串枚举

对于字符串类型的枚举验证：

```go
// 定义自定义字符串类型
type UserRole string

const (
    RoleAdmin UserRole = "admin"
    RoleUser  UserRole = "user"
    RoleGuest UserRole = "guest"
)

type User struct {
    Role   UserRole `json:"role" binding:"enum_string=admin user guest" label:"角色"`
    Status string   `json:"status" binding:"enum_string=active pending inactive" label:"状态"`
}
```

**注意事项**:
- `enum_string` 支持字符串及自定义字符串类型
- 验证是大小写敏感的
- 枚举值使用空格分隔

**错误消息示例**:
```json
{
  "role": "角色必须是以下值之一: admin user guest"
}
```

#### 3. enum_float - 浮点数枚举

对于浮点数类型的枚举验证：

```go
// 定义自定义浮点数类型
type Rating float64

const (
    RatingStar0_5 Rating = 0.5
    RatingStar1_0 Rating = 1.0
    RatingStar1_5 Rating = 1.5
    RatingStar2_0 Rating = 2.0
)

type Product struct {
    Rating   Rating  `json:"rating" binding:"enum_float=0.5 1.0 1.5 2.0 2.5 3.0" label:"评分"`
    Discount float64 `json:"discount" binding:"enum_float=0.1 0.2 0.5" label:"折扣"`
}
```

**注意事项**:
- `enum_float` 支持 float32、float64 及自定义浮点数类型
- 使用精度容差（epsilon）进行浮点数比较，避免精度问题
- 枚举值使用空格分隔

**错误消息示例**:
```json
{
  "rating": "评分必须是以下值之一: 0.5 1.0 1.5 2.0 2.5 3.0"
}
```

### 工具函数使用
```go
// 基础工具
userPtr := xUtil.Ptr(user)              // 获取指针
userName := xUtil.Val(userPtr)          // 安全解引用
hasRole := xUtil.Contains(roles, "admin") // 切片包含检查
enabled := xUtil.ToBool("yes", false)   // 智能布尔转换

// 字符串处理
masked := xUtil.MaskString("13812345678", 3, 4, "*")  // 138****5678
snake := xUtil.CamelToSnake("UserName")               // user_name
camel := xUtil.SnakeToCamel("user_name")             // userName
hash := xUtil.MD5Hash("password123")                  // MD5哈希
clean := xUtil.DefaultIfBlank(input, "默认值")       // 空白处理

// 时间处理
now := xUtil.Now()                                    // 当前时间
unix := xUtil.NowUnix()                              // Unix时间戳
formatted := xUtil.FormatNow("2006-01-02 15:04:05") // 格式化
age := xUtil.Age(birthday)                           // 计算年龄
isToday := xUtil.IsToday(someTime)                   // 是否今天

// 数据验证
validPhone := xUtil.IsValidPhone("13812345678")      // 手机号验证
validEmail := xUtil.IsValidEmail("user@example.com") // 邮箱验证
strongPwd := xUtil.IsStrongPassword("MyP@ssw0rd!")   // 强密码验证
validURL := xUtil.IsValidURL("https://example.com")  // URL验证

// 密码处理
hashBytes, err := xUtil.EncryptPassword("mypassword") // 加密密码(字节)
hashStr, err := xUtil.EncryptPasswordString("mypass") // 加密密码(字符串)
err = xUtil.VerifyPassword("input", hashStr)          // 验证密码
isValid := xUtil.IsPasswordValid("input", hashStr)    // 密码是否有效

// 生成工具
securityKey := xUtil.GenerateSecurityKey()           // 生成安全密钥
```

## 测试

项目使用 Go 标准测试框架。测试文件位于 `test/` 目录中。现有测试 (`util_test.go`) 展示了使用适当断言模式测试工具函数的方法。

### 运行测试
```bash
# 运行所有测试
go test ./...

# 运行特定测试包并显示详细输出
go test ./test -v

# 运行测试并生成覆盖率报告
go test -cover ./...
```

## 最佳实践

### 节点注册
1. **按依赖顺序注册**: 被依赖的组件先注册（如：配置 → 日志 → 数据库 → 业务服务）
2. **错误处理**: 节点函数返回 error 时会导致整个初始化流程中断并 panic
3. **避免重复注册**: 相同的 ContextKey 不能注册多次
4. **使用 Exec 键**: 对于不需要存储结果的初始化逻辑，使用 `xCtx.Exec` 键

### 上下文传递
1. **框架解耦**: 业务逻辑层使用 `context.Context` 而非 `gin.Context`
2. **从 Gin 获取**: 在 Handler 中使用 `c.Request.Context()` 获取标准上下文
3. **Must vs Error**: 启动阶段用 Must 版本，运行时用 Error 版本
4. **上下文注入**: 系统自动将初始化上下文注入到每个 HTTP 请求中

### 错误处理
1. **使用预定义错误码**: 优先使用 `error/error_code.go` 中的预定义错误码
2. **结构化错误**: 通过 `xError.New()` 创建包含上下文信息的结构化错误
3. **中间件自动处理**: 依赖响应中间件自动格式化错误响应，避免手动处理

### 日志记录
1. **使用 slog 标准库**: 直接使用 `slog.InfoContext(ctx, ...)` 进行日志记录
2. **自动 trace 注入**: 自定义 Handler 会自动从 gin.Context 提取 trace ID
3. **结构化日志**: 使用 key-value 对传递日志属性，避免字符串拼接
4. **调试模式**: DEBUG 级别日志仅在调试模式下输出

### 响应格式
1. **统一响应结构**: 所有 API 响应均使用 `BaseResponse` 结构
2. **标准化方法**: 使用 `xResult.Success`、`xResult.SuccessHasData`、`xResult.Error` 方法
3. **自动时间计算**: 在调试模式下自动计算并返回请求处理时间

### 配置管理
1. **环境分离**: 为不同环境准备独立的 `.env` 文件（如 `.env.development`、`.env.production`）
2. **敏感信息**: 使用环境变量保护敏感配置，不要将 `.env` 文件提交到版本控制
3. **配置验证**: 必填配置项缺失时应用会自动 panic，确保启动前所有必填项已设置

### 上下文注入机制
1. **自动注入**: 系统通过 `injectContext` 中间件自动将初始化上下文注入到每个 HTTP 请求
2. **访问方式**: 在 Handler 中使用 `c.Request.Context()` 获取包含所有初始化组件的上下文
3. **生命周期**: 初始化上下文在应用启动时创建，在每个请求中可用
4. **组件访问**: 通过 `xCtxUtil` 工具函数访问数据库、Redis、雪花节点等组件

### 代码组织
1. **包导入约定**: 使用项目统一的包别名:
   - `xReg` - register 包 (注册初始化)
   - `xEnv` - env 包 (环境变量)
   - `xCtx` - context 包 (上下文键)
   - `xError` - error 包 (错误处理)
   - `xResult` - result 包 (响应处理)
   - `xMiddle` - middleware 包 (中间件)
   - `xRoute` - route 包 (路由处理)
   - `xHttp` - http 包 (HTTP 常量)
   - `xLog` - log 包 (日志)
   - `xModels` - models 包 (数据模型)
   - `xSnowflake` - snowflake 包 (雪花算法)
   - `xUtil` - utility 包 (工具函数)
   - `xCtxUtil` - utility/ctxutil 包 (上下文工具)
   - `xValidator` - validator 包 (验证器)
2. **上下文传递**: 始终通过 `gin.Context` 传递请求上下文
3. **工具函数**: 将通用逻辑封装为 `utility` 包中的工具函数

## 扩展指南

### 添加自定义初始化节点

创建自定义节点来初始化数据库、Redis、消息队列等组件：

```go
// 1. 定义初始化函数
func InitDatabase(ctx context.Context) (any, error) {
    // 可以从 ctx 获取其他已初始化的组件
    log := ctx.Value(xCtx.LoggerKey).(*slog.Logger)

    // 初始化数据库
    dsn := xEnv.GetEnvString(xEnv.DatabaseHost, "localhost")
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: xLog.NewGormLogger(log),
    })
    if err != nil {
        return nil, fmt.Errorf("数据库连接失败: %w", err)
    }

    log.Info("数据库初始化成功")
    return db, nil
}

// 2. 注册节点
nodeList := []xRegNode.RegNodeList{
    {Key: xCtx.DatabaseKey, Node: InitDatabase},
}

// 3. 执行注册
reg := xReg.Register(ctx, nodeList)
```

**节点开发建议**：
1. 使用环境变量配置，通过 `xEnv.GetEnvXxx()` 获取
2. 返回明确的错误信息，便于调试
3. 记录初始化日志，方便追踪
4. 考虑依赖关系，确保依赖的组件已初始化

### 添加新的错误码
1. 在 `error/error_code.go` 中定义新的错误常量
2. 遵循现有命名约定和错误码编号规则
3. 添加详细的中文注释说明

### 创建自定义中间件
1. 在 `middleware/` 目录下创建新的中间件文件
2. 实现 `gin.HandlerFunc` 接口
3. 在初始化系统中注册中间件

### 扩展验证器
1. 在 `validator/custom.go` 中添加新的验证函数
2. 在 `RegisterCustomValidators` 中注册新验证器
3. 更新 `validator/messages.go` 中的错误消息

### 添加工具函数
1. 根据功能类型选择合适的工具包 (`utility/` 或 `utility/ctxutil/`)
2. 编写详细的函数注释
3. 在 `test/` 目录中添加对应的单元测试

## 故障排查

### 常见问题
1. **配置未生效**: 确认 `.env` 文件在项目根目录，或环境变量已正确设置
2. **依赖包版本冲突**: 运行 `go mod tidy` 清理依赖
3. **上下文键不存在**: 检查是否正确初始化了系统上下文
4. **响应中间件未生效**: 确认中间件已正确注册到 Gin 引擎

### 调试建议
1. **启用调试模式**: 设置环境变量 `XLF_DEBUG=true`
2. **查看详细日志**: 调试模式下会输出详细的日志信息
3. **检查响应时间**: 利用自动计算的 `overhead` 字段分析性能
4. **验证错误码**: 确保自定义错误码不与预定义码冲突