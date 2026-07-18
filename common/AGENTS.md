# common 知识库

## 概述

通用层模块（`github.com/bamboo-services/bamboo-base-go/common`），提供错误处理、日志系统、雪花算法 ID 生成、请求验证器和工具函数。是 `major` 层和 `plugins/*` 插件的共同依赖，被依赖链中的第二层（`defined → common → major`）。**common 层已完全解耦 gin 依赖，保持纯 Go 依赖。**

## 目录结构

```text
common/
├── base_response.go           # BaseResponse — 标准 API 响应结构体
├── error/
│   ├── error.go               # IError 接口 + Error 结构体
│   ├── error_code.go          # ErrorCode 结构体 + 全量预定义错误码常量
│   ├── error_message.go       # ErrMessage 类型（string 别名）
│   ├── error_new.go           # NewError / NewErrorHasData 构造函数
│   └── error_library.go       # 错误库辅助函数
├── log/
│   ├── command.go             # LogNamedLogger + 全局日志函数（Info/Warn/Error/Panic）
│   ├── const.go               # 日志器名称常量（NamedMAIN / NamedINIT / NamedRESU 等）
│   ├── handler.go             # 自定义 slog Handler（控制台 + 文件双写 + LogContextExtractor）
│   └── rotator.go             # 按日期 + 大小的日志文件切割器
├── snowflake/
│   ├── gene.go                # Gene 类型 + 预定义基因常量（系统级 0-15 / 业务级 16-63）
│   ├── gene_calc.go           # 基因 ID 计算（位运算嵌入 gene）
│   ├── snowflake.go           # Node 结构体 + ID 生成
│   ├── global.go              # 默认节点单例（InitDefaultNode / GetDefaultNode）
│   └── snowflake_test.go      # 雪花算法测试
├── validator/
│   ├── custom.go              # RegisterCustomValidators — 8 个自定义验证器注册入口
│   ├── translator.go          # 验证错误翻译器（中文）+ ValidateProvider 接口
│   ├── messages.go            # 验证错误消息模板
│   ├── validator_url.go       # strict_url — HTTP/HTTPS URL 验证
│   ├── validator_uuid.go      # strict_uuid — UUID 格式验证
│   ├── validator_snowflake.go # snowflake — 雪花 ID 验证
│   ├── validator_regexp.go    # regexp — 自定义正则验证
│   ├── validator_alphanum.go  # alphanum_underscore — 字母数字下划线
│   ├── validator_enum_*.go    # enum_int / enum_string / enum_float — 枚举验证
│   └── *_test.go              # 各验证器测试
├── utility/
│   ├── common.go              # Ptr / Val / Contains / ToBool 泛型工具
│   ├── utility.go             # Encryption / Generate / Password / Str / Timer / Parse / Valid / Function — 工具入口
│   ├── context/               # 上下文工具（仅通用工具，不依赖框架）
│   │   └── common.go          # IsDebugMode / CalcOverheadTime / GetRequestKey / GetErrorMessage
│   └── package/               # 工具实现（pack）
│       ├── password.go        # bcrypt 密码加密与验证
│       ├── encryption.go      # SHA256 / MD5 哈希
│       ├── security.go        # 安全密钥生成
│       ├── generate.go        # 随机字符串生成
│       ├── string.go          # 字符串工具（IsBlank / Mask / ...）
│       ├── time.go            # 时间工具
│       ├── parse.go           # 类型解析
│       ├── validate.go        # 格式验证（手机号 / 身份证 / IP ...）
│       ├── function.go        # 反射工具（函数名获取）
│       └── *_test.go          # 各工具测试
└── go.mod                     # 独立模块定义
```

> **架构变更说明**：
> - `HandleValidationError`（验证错误响应）已从 `common/validator/response.go` 迁移到 `major/validator/response.go`，通过 `ValidateProvider` 接口解耦 gin 依赖。
> - `Bind` 绑定工具已从 `common/utility/` 迁移到 `major/utility/bind.go`。
> - `GetDB` / `GetRDB` / `GetEmailClient` / `GetSnowflakeNode` 等上下文提取函数已从 `common/utility/context/` 迁移到 `major/utility/context/`，通过 `ContextExtractor` 接口解耦 gin 依赖。
> - GORM 日志适配器已从 `common/log/gorm.go` 迁移到 `major/log/gorm.go`。
> - `common/validator/translator.go` 不再直接依赖 `gin/binding`，通过 `ValidateProvider` 接口由 major 层注入。

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 返回标准错误响应 | `error/error_new.go` → `NewError()` | 传入 ErrorCode + ErrMessage |
| 查找预定义错误码 | `error/error_code.go` | 400xx-504xx 分段定义 |
| 创建带名称的日志器 | `log/command.go` → `WithName(name)` | 使用 `xLog.Named*` 常量 |
| 选择日志器名称 | `log/const.go` | NamedMAIN / NamedINIT / NamedRESU / NamedHTTP / NamedGRPC 等 |
| 生成雪花 ID | `snowflake/snowflake.go` → `GenerateID(gene)` | 传入 `xSnowflake.Gene*` 基因类型 |
| 选择基因类型 | `snowflake/gene.go` | 系统级 0-15 / 业务级 16-63 |
| 注册自定义验证器 | `validator/custom.go` → `RegisterCustomValidators()` | 在 Gin 引擎初始化时调用 |
| 翻译验证错误 | `validator/translator.go` → `TranslateError(err)` | 返回 `map[field]message` 中文翻译 |
| 加密密码 | `utility/package/password.go` → `Encrypt()` | base64 + bcrypt |
| 构建 API 响应体 | `base_response.go` → `BaseResponse` | HTTP 与 gRPC 共用结构 |
| 设置 ValidateProvider | `validator/translator.go` → `SetValidateProvider()` | major 层在初始化时注入 |

## 约定

- **错误码分段体系**：错误码遵循 HTTP 状态码分段（400xx / 401xx / 403xx / 404xx / 500xx ...），`ErrorCode.Code` 前 3 位决定 HTTP 响应状态码（`Code/100`）。
- **ErrMessage 使用 `xError.ErrMessage("...")` 构造**：是 `string` 的类型别名，提供 `.String()` 方法。
- **日志必须从 context 取 trace ID**：`Info/Warn/Error` 系列函数的第一个参数始终是 `ctx context.Context`，Handler 通过 `LogContextExtractor` 从上下文提取追踪标识。
- **日志器命名使用 4 字母大写常量**：`WithName(xLog.NamedMAIN)` 会在日志中标记 `[MAIN]` 分组前缀，便于过滤。
- **LogContextExtractor 解耦**：`log/handler.go` 中的 LogHandler 通过 `SetLogContextExtractor` 注入提取器，major 层注入 gin 实现。common 层不直接依赖 gin。
- **雪花 ID 必须指定基因**：`GenerateID(xSnowflake.GeneDefault)` 是兜底选项，业务实体应实现 `GeneProvider` 接口指定自己的基因类型。
- **工具函数返回实例而非指针**：`Encryption()` / `Generate()` / `Password()` 等返回结构体值，方法通过值接收者调用。
- **ValidateProvider 接口**：`validator/translator.go` 定义了 `ValidateProvider` 接口，由 major 层注入 gin 实现。common 层不直接依赖 gin。
- **验证器名称在 binding tag 中直接使用**：`binding:"enum_string=asc desc"` / `binding:"snowflake"` / `binding:"strict_url"`。

## 反模式

- **禁止用 `fmt.Errorf` 或 `errors.New` 替代 `xError.NewError`** — 标准 error 没有错误码，无法走统一错误处理中间件。
- **禁止在业务代码中直接调用 `slog.*`** — 应使用 `xLog.WithName(...)` 创建命名日志器，便于日志分组和过滤。
- **禁止硬编码日志器名称字符串** — 使用 `log/const.go` 中的 `Named*` 常量。
- **禁止在 `ErrorCode` 中使用非标准 HTTP 段错误码** — 前三位必须对应有效 HTTP 状态码（400/401/403/404/500/502/503/504）。
- **禁止在 common 层引入 gin 依赖** — 已通过 ContextExtractor / ValidateProvider / LogContextExtractor 等接口解耦。
- **禁止使用已迁移的上下文提取函数** — `GetDB/GetRDB/GetEmailClient/GetSnowflakeNode/Bind` 等函数的新位置在 `major/utility/` 中。

## 调试路径

1. 请求返回错误但日志没有业务错误记录 — 检查 `NewError()` 的 `throw` 参数是否为 `true`。
2. 日志缺少 trace ID — 确认 `LogContextExtractor` 已注入（major 层在 `register_logger.go` 中注入）。
3. 验证器报错信息是英文 — 确认 `RegisterTranslator()` 在 `RegisterCustomValidators()` 之前调用，且 `ValidateProvider` 已注入。
4. 雪花 ID 冲突 — 检查多实例的 `SNOWFLAKE_DATACENTER_ID` + `SNOWFLAKE_NODE_ID` 唯一性。