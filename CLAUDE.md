# CLAUDE.md

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

- **配置管理** (`models/config.go`): 基于 YAML 的配置系统
  - `xConfig` 结构支持数据库、NoSQL (Redis) 和调试设置
  - 以嵌套 YAML 配置形式组织

- **工具库** (`utility/`): 通用辅助函数和上下文工具
  - `ctxutil/` 提供与数据库、日志和通用操作相关的上下文工具
  - 通用工具函数如 `Ptr()` 用于指针处理

### 核心依赖

- **Gin 框架**: 用于 HTTP 路由和中间件的 Web 框架
- **Zap 日志器**: 支持多种输出格式的结构化日志记录
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
├── config/                   # 日志配置 (核心配置、编码器配置)
├── constants/               # 应用常量 (上下文键、请求头、日志器名称)
├── error/                   # 错误处理系统 (接口、错误码、构造函数)
├── go.mod                   # 模块定义和依赖管理
├── handler/                 # HTTP 处理器 (当前为空)
├── helper/                  # 辅助工具 (恐慌恢复等)
├── init/                    # 初始化和注册系统
├── middleware/              # Gin 中间件 (响应处理)
├── models/                  # 数据模型 (配置结构体)
├── result/                  # 响应结果格式化
├── test/                    # 测试文件
├── utility/                 # 通用工具和上下文辅助工具
└── validator/               # 自定义验证逻辑和消息
```

### 详细模块说明

- **config/**: 日志配置模块
  - `logger_core.go`: 日志核心配置
  - `logger_encoder.go`: 日志编码器配置

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

- **middleware/**: 中间件模块
  - `response.go`: 统一响应中间件，处理错误和成功响应 (xMiddle:64)

- **result/**: 响应处理模块
  - `result.go`: 提供 Success、SuccessHasData、Error 三种响应方法 (xResult:79)

- **utility/**: 工具库模块
  - `common.go`: 通用工具函数
  - `generate.go`: 生成工具函数
  - `ctxutil/`: 上下文工具子模块
    - `common.go`: 通用上下文工具 (调试模式判断、时间计算)
    - `database.go`: 数据库上下文工具
    - `logger.go`: 日志器上下文工具

- **validator/**: 验证模块
  - `custom.go`: 自定义验证规则
  - `messages.go`: 验证错误消息
  - `response.go`: 验证响应处理
  - `vaildeate.go`: 验证逻辑实现

## 使用模式

### 初始化应用程序
```go
// 创建注册实例并初始化所有组件
reg := xInit.Register()

// 访问初始化后的组件
engine := reg.Serve      // *gin.Engine - HTTP 服务引擎
config := reg.Config     // *xModels.xConfig - 应用配置
logger := reg.Logger     // *zap.Logger - 日志记录器

// 启动服���器
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
应用程序期望的 YAML 配置结构:
```yaml
xlf:
  debug: true

database:
  host: "localhost"
  port: 3306
  user: "root"
  pass: "password"
  name: "bamboo_db"

nosql:
  host: "localhost"
  port: 6379
  user: ""
  pass: ""
  database: 0
  prefix: "bamboo:"
```

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

// 获取日志器
logger := xCtxUtil.GetLogger(ctx)
sugarLogger := xCtxUtil.GetSugarLogger(ctx)
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
1. **使用上下文日志器**: 通过 `xCtxUtil.GetLogger(ctx)` 获取与请求绑定的日志器
2. **结构化日志**: 利用 Zap 的结构化日志功能，避免字符串拼接
3. **调试模式**: 在调试模式下记录详细信息，生产环境保持简洁

### 响应格式
1. **统一响应结构**: 所有 API 响应均使用 `BaseResponse` 结构
2. **标准化方法**: 使用 `xResult.Success`、`xResult.SuccessHasData`、`xResult.Error` 方法
3. **自动时间计算**: 在调试模式下自动计算并返回请求处理时间

### 配置管理
1. **环境分离**: 为不同环境准备独立的配置文件
2. **敏感信息**: 使用环境变量或密钥管理系统保护敏感配置
3. **配置验证**: 在应用启动时验证必需的配置项

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
1. **配置文件未找到**: 确认 YAML 配置文件路径和格式正确
2. **依赖包版本冲突**: 运行 `go mod tidy` 清理依赖
3. **上下文键不存在**: 检查是否正确初始化了系统上下文
4. **响应中间件未生效**: 确认中间件已正确注册到 Gin 引擎

### 调试建议
1. **启用调试模式**: 在配置中设置 `xlf.debug: true`
2. **查看详细日志**: 调试模式下会输出详细的日志信息
3. **检查响应时间**: 利用自动计算的 `overhead` 字段分析性能
4. **验证错误码**: 确保自定义错误码不与预定义码冲突