# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

本文件为 Claude Code (claude.ai/code) 提供在此代码库中工作的指导说明。

## 项目概述

这是 `bamboo-base-go`，一个为 bamboo 服务提供基础组件的 Go 语言库。它被设计为一个可重用的基础库，用于构建基于 Gin 框架的 Web API，提供标准化的错误处理、日志记录、配置管理和响应格式化功能。

## 架构设计

### 核心组件

- **初始化系统** (`init/`): 集中式注册系统，负责引导应用程序启动
  - `register.go` 包含主要的 `Register()` 函数，初始化所有组件
  - 处理配置加载、日志器设置、Gin 引擎初始化和系统上下文设置

- **错误处理** (`error/`): 全面的错误管理系统
  - `ErrorInterface` 定义标准错误接口合约
  - `Error` 结构体实现包含代码、消息和数据的结构化错误
  - `ErrorCode` 提供预定义的错误常量

- **响应系统**: 标准化 API 响应结构
  - `BaseResponse` 在 `base_response.go` 中定义通用响应格式
  - `result/` 包处理响应格式化
  - `middleware/response.go` 提供统一响应中间件

- **配置管理**: 基于环境变量的配置系统
  - 使用 `godotenv` 加载 `.env` 文件
  - 配置直接通过 `xLog.GetXxx()` 获取

- **工具库** (`utility/`): 丰富的通用辅助函数和上下文工具
  - `ctxutil/` 提供与数据库、日志和通用操作相关的上下文工具
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
├── log/                      # slog 自定义 Handler (彩色控制台 + JSON 文件)
├── constants/               # 应用常量 (上下文键、请求头、日志器名称)
├── error/                   # 错误处理系统 (接口、错误码、构造函数)
├── go.mod                   # 模块定义和依赖管理
├── handler/                 # HTTP 处理器 (当前为空)
├── helper/                  # 辅助工具 (恐慌恢复等)
├── init/                    # 初始化和注册系统
├── middleware/              # Gin 中间件 (响应处理)
├── models/                  # 数据模型 (实体基类)
├── result/                  # 响应结果格式化
├── snowflake/               # 雪花算法 (标准雪花、基因雪花)
├── test/                    # 测试文件
├── utility/                 # 通用工具和上下文辅助工具
└── validator/               # 自定义验证逻辑和消息
```

### 详细模块说明

- **log/**: slog 自定义 Handler 模块
  - `handler.go`: 自定义 slog.Handler 实现，支持彩色控制台输出和 JSON 文件记录，自动从 context 提取 trace ID

- **constants/**: 系统常量定义
  - `context.go`: 上下文键常量 (xConsts:124)
  - `header.go`: HTTP 请求头常量
  - `logger.go`: 日志器名称常量

- **error/**: 错误处理核心模块
  - `error.go`: 错误接口和结构体定义 (xError:72)
  - `error_code.go`: 预定义错误码常量 (包含 40+ 种错误类型)
  - `error_new.go`: 错误构造函数

- **init/**: 应用初始化模块
  - `register.go`: 主注册函数，返回 Reg 结构体 (xInit:50)
  - `register_config.go`: 配置初始化
  - `register_context.go`: 上下文初始化
  - `register_gin.go`: Gin 引擎初始化
  - `register_logger.go`: 日志器初始化
  - `register_snowflake.go`: 雪花算法初始化

- **models/**: 数据模型模块
  - `base_entity.go`: GORM 实体基类 (BaseEntity、GeneBaseEntity)

- **snowflake/**: 雪花算法模块
  - `snowflake.go`: 标准雪花 ID (41位时间戳 + 5位数据中心 + 5位节点 + 12位序列)
  - `gene.go`: 基因类型定义 (64种业务类型)
  - `gene_snowflake.go`: 基因雪花 ID (41位时间戳 + 6位基因 + 3位数据中心 + 3位节点 + 10位序列)
  - `global.go`: 全局节点管理 (默认节点初始化)

- **middleware/**: 中间件模块
  - `response.go`: 统一响应中间件，处理错误和成功响应 (xMiddle:64)

- **result/**: 响应处理模块
  - `result.go`: 提供 Success、SuccessHasData、Error 三种响应方法 (xResult:79)

- **utility/**: 工具库模块
  - `common.go`: 基础工具函数 (`Ptr()`、`Val()`、`Contains()`、`ToBool()`)
  - `string.go`: 字符串处理工具 (命名转换、脱敏、验证等)
  - `time.go`: 时间处理工具 (格式化、解析、计算等)
  - `validate.go`: 数据验证工具 (手机号、身份证、URL 等验证)
  - `password.go`: 密码加密工具 (bcrypt 加密、验证等)
  - `generate.go`: 生成工具函数 (`GenerateSecurityKey()`)
  - `ctxutil/`: 上下文工具子模块
    - `common.go`: 上下文通用工具 (调试模式、请求信息等)
    - `database.go`: 数据库上下文工具
    - `nosql.go`: Redis 上下文工具
    - `snowflake.go`: 雪花算法上下文工具

- **validator/**: 验证模块
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

## 使用模式

### 初始化应用程序
```go
// 创建注册实例并初始化所有组件
reg := xInit.Register()

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

应用程序通过环境变量配置，支持 `.env` 文件加载。直接使用 `xEnv.GetXxx()` 获取配置：

```go
import "os"

// 获取配置
debug := os.Getenv("XLF_DEBUG")
dbHost := os.Getenv("DATABASE_HOST")
```

**常用环境变量：**
| 环境变量 | 说明 |
|----------|------|
| `XLF_DEBUG` | 调试模式 (true/false) |
| `XLF_HOST` | 监听地址 |
| `XLF_PORT` | 监听端口 |
| `DATABASE_HOST` | 数据库主机 |
| `DATABASE_PORT` | 数据库端口 |
| `DATABASE_USER` | 数据库用户名 |
| `DATABASE_PASS` | 数据库密码 |
| `DATABASE_NAME` | 数据库名称 |
| `SNOWFLAKE_DATACENTER_ID` | 雪花算法数据中心 ID (0-31) |
| `SNOWFLAKE_NODE_ID` | 雪花算法节点 ID (0-31) |

**使用方式：**
1. 复制 `.env.example` 为 `.env`
2. 根据需要修改配置项

### 中间件使用
```go
// 应用响应中间件 (通常在初始化时自动应用)
engine.Use(xMiddle.ResponseMiddleware)
```

### 上下文工具使用
```go
// 检查是否为调试模式
if xCtxUtil.IsDebugMode(ctx) {
    // 调试逻辑
}

// 计算请求处理时间
overhead := xCtxUtil.CalcOverheadTime(ctx)

// 获取系统组件
db := xCtxUtil.GetDB(ctx)       // 数据库连接
rdb := xCtxUtil.GetRDB(ctx)     // Redis 客户端

// 获取请求信息
requestKey := xCtxUtil.GetRequestKey(ctx)
errorCode := xCtxUtil.GetErrorCode(ctx)

// 获取雪花算法节点
snowflakeNode := xCtxUtil.GetSnowflakeNode(ctx)
geneNode := xCtxUtil.GetGeneSnowflakeNode(ctx)
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

### 代码组织
1. **包导入约定**: 使用项目统一的包别名 (`xInit`、`xError`、`xResult` 等)
2. **上下文传递**: 始终通过 `gin.Context` 传递请求上下文
3. **工具函数**: 将通用逻辑封装为 `utility` 包中的工具函数

## 扩展指南

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