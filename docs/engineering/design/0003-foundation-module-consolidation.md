# v2.0.0 foundation 地基模块合并与包重构设计

> 状态：draft · 日期：2026-10-19 · Scope：foundation
> 承接 [0001 生命周期插件运行时设计](./0001-lifecycle-plugin-runtime.md)、[0002 Context 体系与共享组件传递设计](./0002-context-propagation-and-shared-state.md)

## 目标与边界

在 v1.x 的模块结构中，最底层的两个模块 `defined`（`defined/go.mod`）和 `common`（`common/go.mod`）不仅职责边界模糊，还在包命名与外部依赖上留下了三处明显的技术债：

1. **`x_yyy` 大小写混合包名违背 Go 社区规范**：v1.x 的包声明大量采用 `package xCtx`、`package xError`、`package xLog`、`package xHttp`、`package xVaild`（甚至把 Valid 拼错成 Vaild）这种 `x + 大写字母` 的形式。追根溯源，是因为原目录名直接叫 `context`、`error`、`log`、`http`，与 Go 语言预声明的 `error` 标识符 及标准库 `context`、`log`、`net/http` 迎面撞车，迫使所有调用方在 `import` 时手写别名。
2. **上下文与工具目录首尾分离、套娃严重**：`defined/context` 定义 `ContextKey`，而读取请求元数据的函数却散落在 `common/utility/context`；`common/utility` 底下又套了一层 `common/utility/package`（包名叫 `pack`），导出路径拧巴。
3. **本该属于插件层的 HTTP 与数据库能力混进了底座**：
   - `defined/http/util.go:10` 直接引入了 `*gin.Context`，导致最底层的 `defined/go.mod:6` 挂上了整棵 `github.com/gin-gonic/gin` 依赖树；
   - `common/validator/`（`translator.go`、`custom.go`）里的 `ValidateProvider` 与 8 个 `binding` 标签验证器完全服务于 Gin 的请求体绑定（`gin/binding` 与 `major/validator/response.go`），却塞在 `common` 里并引入了 `go-playground/validator/v10` 等三项依赖；
   - `common/database/dialector.go` 里的 GORM `Dialector` 注册表把 `gorm.io/gorm` 强行拉进了通用底座。

本次重构将原 `defined` 与 `common` 合并为零框架地基模块 **`foundation`**（`github.com/bamboo-services/bamboo-base-go/foundation/v2`）。首批整体 Release 和各独立模块都从 **v2.0.0** 起步；模块声明与 import 使用小写 `/v2`，源码目录不增加 `v2/` 层。后续模块按需发版，整体 Release 记录精确版本清单。包组织执行以下两条规则：

- **废除 `x_yyy` 模式，统一使用“目录名 = `package` 名 = 全小写富含语义词汇”**：用 `apperr`、`ctxkey`、`logging`、`envelope`、`toolkit` 等自解释词汇彻底避开 Go 标准库与关键字冲突，调用方 `import` 后直接按包名调用，无需任何 import 别名。
- **底座彻底剥离 HTTP、Validator 与数据库驱动注册表**：将原 `defined/http` 与原 `common/validator` 移交给后续的 Gin/HTTP 插件实现；将原 `common/database` 移交给后续的数据库插件部分实现。`foundation` 自此不再依赖 `gin`、`gorm` 与 `validator/v10`。

## 工作方式

### 1. 模块合并与顶层依赖方向

合并后，整个 Go Workspace（`go.work`）形成清晰的 **`foundation ← application ← plugins/*`** 三层单向依赖链：

```text
┌─────────────────────────────────────────────────────────────┐
│                     业务服务 (下游应用)                      │
└──────────────┬─────────────────────────────┬────────────────┘
               │                             │
    ┌──────────▼──────────┐       ┌──────────▼──────────┐
    │   application 模块   │◄──────┤    plugins/* 模块   │
    │ (生命周期内核/容器/  │       │ (gin/grpc/cron/     │
    │  三模式调度/句柄)    │       │  async/email/db/*)  │
    └──────────┬──────────┘       └──────────┬──────────┘
               │                             │
               └──────────────┬──────────────┘
                              │
                   ┌──────────▼──────────┐
                   │   foundation 模块    │
                   │ (typed/apperr/ctxkey│
                   │  /env/envelope/     │
                   │  logging/snowflake/ │
                   │  toolkit)           │
                   │ ← 纯净底座，零框架   │
                   └─────────────────────┘
```

| 当前工作树                                           | 本次设计                                    |
| ---------------------------------------------------- | ------------------------------------------- |
| defined 与 common 分开，common 仍依赖 GORM/validator | 合并成 foundation，框架能力移入插件         |
| ctxkey/env/日志提取各有旧入口                        | 统一键原语与直接 Context 读取，不保留兼容壳 |

### 2. `foundation` 八大子包结构与重命名映射

v2 工作树将移除原 `defined/` 与 `common/`，在根目录下建立 `foundation/`，内部收敛为 8 个全小写子包。这里描述的是待实施的重组；已发布的 v1 版本仍保留其原有代码，不会被这次目录删除追溯改写。

```text
foundation/                  # module .../bamboo-base-go/foundation/v2
├── go.mod                   # 仅依赖 uuid、x/crypto、godotenv，零 gin/gorm/validator
├── typed/                   # package typed     — 强类型字符串原语 typed.String (详见 0004)
├── apperr/                  # package apperr    — 错误码体系与统一错误对象
├── ctxkey/                  # package ctxkey    — 上下文键 Key、Exposer 与请求元数据
├── env/                     # package env       — 环境变量键 Key 与 typed.String 读取器
├── envelope/                # package envelope  — 协议中立的标准响应信封结构
├── logging/                 # package logging   — 结构化 slog 双写与切割日志系统
├── snowflake/               # package snowflake — 64 基因位雪花 ID 算法
└── toolkit/                 # package toolkit   — 纯 Go 通用工具集（拍平原 utility/package）
```

各子包的迁移、改名与剥离对照表如下：

| 原路径（v1.x）与旧包名                                               | v2.0.0 归宿与新包名                                             | 为什么这么改？                                                                                                                               |
| -------------------------------------------------------------------- | --------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| _(新增)_                                                             | `foundation/typed`（`package typed`）                           | 提供 `typed.String` 原语与六通道编解码/原生类型解析（详见 [0004 设计](./0004-typed-string-primitive.md)）。                                  |
| `common/error`（`xError`）                                           | `foundation/apperr`（`package apperr`）                         | 避开 Go 预声明的 `error` 接口名，直呼 `apperr.New` / `apperr.Code` / `apperr.Error`。                                                        |
| `defined/context` + `common/utility/context`（`xCtx` / `xCtxUtil`）  | `foundation/ctxkey`（`package ctxkey`）                         | 避开标准库 `context`；合并键值定义与请求元数据读取，废除 `ContextNodeList` 与全局 Extractor。                                                |
| `defined/env`（`xEnv`）                                              | `foundation/env`（`package env`）                               | 标准库无 `env` 包，直接用 `package env`；读取出口对接 `typed.String`。                                                                       |
| `common/base_response.go`（`xBase`）                                 | `foundation/envelope`（`package envelope`）                     | 与 `apperr` 同属底座**标准契约定义**；类型命名为 `envelope.Response`，只定义响应信封结构，不含任何协议输出逻辑。                             |
| `common/log`（`xLog`）                                               | `foundation/logging`（`package logging`）                       | 避开标准库 `log`，并把简短的 `log` 标识符留给业务函数内的局部变量（`log := logging.WithName(...)`）。                                        |
| `common/snowflake`（`xSnowflake`）                                   | `foundation/snowflake`（`package snowflake`）                   | `snowflake` 本身即为高辨识度领域词，直接去掉 `x` 前缀转小写。                                                                                |
| `common/utility` + `utility/package`（`xUtil` / `pack`）             | `foundation/toolkit`（`package toolkit`）                       | 拍平内部 `package` 套娃目录，将加密、哈希、密码、随机串、时间等纯函数统一收归 `toolkit`。                                                    |
| `defined/http`（`xHttp`）                                            | **移出 `foundation`** → 交由后续 Gin/HTTP 插件实现              | `foundation` 不再包含任何 HTTP 专属头常量与 Token 提取逻辑，彻底斩断对 Gin 的依赖。                                                          |
| `common/validator`（`xVaild`）                                       | **移出 `foundation`** → 交由后续 Gin/HTTP 插件实现              | 原验证器与翻译器完全围绕 `gin/binding` 请求校验工作；移出后底座不再背负 `validator/v10` 依赖。                                               |
| `common/database`（`xDB`）                                           | **移出 `foundation`** → 交由后续数据库部分实现                  | 底座不内置 GORM `Dialector` 注册表，彻底清退 `gorm.io/gorm` 依赖。                                                                           |
| `major/result`（`xResult`）与 `plugins/grpc/result`（`xGrpcResult`） | **移出底座/核心层** → 交由插件实现 `httpresult` 与 `grpcresult` | 响应如何写回网络属于协议插件职责；HTTP 与 gRPC 插件各自提供 `httpresult` 与 `grpcresult`，统一依据 `apperr` 的错误信息决策最终报错显示内容。 |

### 3. 调用方代码体验对比

取消 `x_yyy` 模式后，业务代码与插件代码不再需要维护冗长的 import 别名表，阅读体验立竿见影：

```go
// v2.0.0 标准导入方式：零别名、全小写、不与标准库 context/log/error 冲突
import (
    "context"

    "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/toolkit"
)
```

### 4. 子包内部的关键清理动作

1. **`foundation/ctxkey` 合流与去全局化**：
   - 将原 `ContextKey` 简化命名为 `ctxkey.Key`（底层基于 `typed.String` 派生：`type Key typed.String`），调用时写 `ctxkey.Key`、`ctxkey.RequestKey`、`ctxkey.DatabaseKey`，自然顺口；
   - 删除 v1.x 遗留的 `defined/context/reg_node.go`（`ContextNodeList` 裸切片）；
   - 将原 `common/utility/context/common.go` 中的 `IsDebugMode`、`CalcOverheadTime`、`GetRequestKey`、`GetErrorMessage` 合入 `ctxkey/meta.go`，直接从标准 `ctx.Value(...)` 读取，彻底删除 `globalContextExtractor`。
2. **`foundation/logging` 移除全局 `LogContextExtractor`**：
   - 配合 [0002 Context 体系设计](./0002-context-propagation-and-shared-state.md) 的请求桥接，`logging/handler.go` 直接通过 `ctx.Value(ctxkey.RequestKey)` 提取请求 ID，移除现位于 `common/log/handler.go` 的 `globalLogExtractor` 与 `SetLogContextExtractor`。
3. **`foundation/toolkit` 拍平目录**：
   - 废弃原 `common/utility/package/` 二级目录与 `util.Encrypt.*` 这种伪命名空间结构体中转，直接在 `foundation/toolkit/` 下按 `crypto.go`、`generate.go`、`password.go`、`string.go`、`time.go`、`valid.go` 导出清晰的函数。

## 数据与接口

### 1. `foundation/ctxkey`（`package ctxkey`）核心签名

```go
package ctxkey

import (
    "context"

    "github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// Key 是基于 typed.String 派生的强类型上下文键与插件唯一标识。
type Key typed.String

var _ typed.Operable = Key("")
// 下文只列核心签名；完整编解码委托与 Convertible 断言按 0004 实现。

const (
    Nil              Key = ""
    RequestKey       Key = "context_request_key"
    ErrorCodeKey     Key = "context_error_code"
    ErrorMessageKey  Key = "context_error_message"
    UserStartTimeKey Key = "context_user_start_time"
    DatabaseKey      Key = "context_database"
    RedisClientKey   Key = "context_redis_client"
    SnowflakeNodeKey Key = "context_snowflake_node"
    EmailClientKey   Key = "context_email_client"
    CacheManagerKey  Key = "context_cache_manager"
)

func (k Key) String() string            { return string(k) }
func (k Key) TypedString() typed.String { return typed.String(k) }
func (k Key) IsEmpty() bool             { return len(k) == 0 }
func (k Key) IsValid() bool             { return typed.String(k).IsValid() }

// Marshal/Unmarshal、Value/Scan 与 Set 显式委托给 typed.String，详见 0004。
// 完整实现须满足：var _ typed.Convertible = (*Key)(nil)

// Exposer 交付 PluginInfo.Provides 中声明的底层句柄；集合校验由 application 负责。
type Exposer interface {
    Expose() map[Key]any
}

// 以下函数由原 common/utility/context 合入，直接读取标准 ctx.Value，无全局 Extractor。
func IsDebugMode() bool
func CalcOverheadTime(ctx context.Context) int64
func GetRequestKey(ctx context.Context) string
func GetErrorMessage(ctx context.Context) string
```

### 2. `foundation/apperr` 与 `foundation/envelope` 核心签名

```go
package apperr

import "context"

// Code 表示业务错误码（前 3 位对应状态分组，如 400xx / 500xx）。
type Code int
type Output string
type Message string
type ErrMessage string

// ErrorCode 保留稳定的机器码、输出标识与对外文案。
type ErrorCode struct {
    Code    Code
    Output  Output
    Message Message
}

// Error 是框架统一错误对象（原 xError.Error）。
type Error struct {
    Code         ErrorCode `json:"code"`
    ErrorMessage ErrMessage `json:"error_message,omitempty"`
    Data         any       `json:"data,omitempty"`
    Err          error     `json:"-"`
}

func New(
    ctx context.Context,
    code *ErrorCode,
    msg ErrMessage,
    isLog bool,
    errs ...error,
) *Error

// Error 实现标准 error；Unwrap 暴露根因供 errors.Is/As 使用，不等于对外输出根因。
func (e *Error) Error() string
func (e *Error) Unwrap() error
```

```go
package envelope

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// Response 是与 apperr 并列的标准响应信封契约定义（原 xBase.BaseResponse）。
// 它只描述跨协议的标准返回字段，不绑定 HTTP 或 gRPC 写出行为。
type Response struct {
    Context      string            `json:"context,omitempty"`
    Code         apperr.Code       `json:"code"`
    Output       apperr.Output     `json:"output"`
    Message      apperr.Message    `json:"message"`
    ErrorMessage apperr.ErrMessage `json:"error_message,omitempty"`
    Overhead     int64             `json:"overhead,omitempty"`
    Data         any               `json:"data,omitempty"`
}
```

> **标准契约（`apperr` + `envelope.Response`）与协议结果输出（`httpresult` / `grpcresult`）的边界**：
> `foundation` 只负责定义“错误长什么样（`apperr.Error`）”和“响应信封长什么样（`envelope.Response`）”。
> 至于怎么把结果写回给客户端，原 `major/result`（`xResult`）从底座与核心层彻底移除，交由各协议插件在自身目录内实现 **`httpresult`**（HTTP/Gin 插件下）和 **`grpcresult`**（`plugins/grpc` 下）——两者统一消费 `apperr.Error` 中的错误码、对外消息与调试详情，各自决策在 HTTP JSON 或 gRPC Status/Protobuf 中如何展示报错内容。

### 3. foundation/env 的边界

`env.Lookup(key) (typed.String, bool)` 区分“未设置”和“已设置为空”，读取结果仍是原始文本。Key 的编解码走受控键名校验，环境变量值的 Duration/Int/Bool 解析不走键名语法。

系统／应用级环境键以 `APP_` 起手，不再沿用 `XLF_` 前缀；例如 `APP_DEBUG`、`APP_HOST`、`APP_PORT`。数据库、邮件和具体协议插件仍使用各自领域前缀，不因这条系统命名规则混入 `APP_`。本轮只规定新设计，不改写现有 v1 代码。

四个应用时间键命名为 APP_START_TIMEOUT、APP_GRACE_TIMEOUT、APP_CLEANUP_TIMEOUT、APP_SHUTDOWN_TIMEOUT。env 只声明键与读取工具；默认值、Option 优先级、启动快照和跨字段校验由 [0001](./0001-lifecycle-plugin-runtime.md#环境变量只在启动装配时读取) 负责，底座不依赖 application。

### 4. `foundation/go.mod` 依赖边界

```text
module github.com/bamboo-services/bamboo-base-go/foundation/v2

go 1.25.0

require (
    github.com/google/uuid v1.6.0
    github.com/joho/godotenv v1.5.1
    golang.org/x/crypto v0.54.0
)
```

> 对比 v1.x：彻底清退了 `github.com/gin-gonic/gin`、`gorm.io/gorm`、`github.com/go-playground/validator/v10`、`locales` 与 `universal-translator`。

## 异常与兼容

1. **源码边界一次切清**：v2 工作树不保留旧 `defined/common/major` 的入口与访问器兼容壳。业务主动迁移到新模块路径与包名；旧项目如暂不迁移，继续固定已发布的 v1 模块版本。
2. **插件能力有明确归宿**：原 `defined/http`、`common/validator` 交给 `plugins/web-gin`；原 `common/database` 交给 `plugins/database/dialect`。其结构由 [0005](./0005-plugin-architecture-and-adapter-pattern.md) 承接，不再描述为“以后再找地方安放”。

### 源码目录不搬，模块身份升级到 v2

Go 从 v2 起要求模块路径包含主版本。这里采用标准的语义导入版本规则，不改磁盘布局：`plugins/grpc/go.mod` 声明 `module github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2`，下游 import 相应带 `/v2`，但文件仍放在 `plugins/grpc/`。go.work 也仍写 `use ./plugins/grpc`。

首批 14 个非根模块都发布 v2.0.0。下表的模块路径统一省略前缀 `github.com/bamboo-services/bamboo-base-go/`；tag 按源码目录生成，不把模块路径末尾的 /v2 再当成一层目录：

| 源码目录          | module 路径尾部     | 首批 tag                |
| ----------------- | ------------------- | ----------------------- |
| foundation/       | foundation/v2       | foundation/v2.0.0       |
| application/      | application/v2      | application/v2.0.0      |
| plugins/web-gin/  | plugins/web-gin/v2  | plugins/web-gin/v2.0.0  |
| plugins/grpc/     | plugins/grpc/v2     | plugins/grpc/v2.0.0     |
| plugins/cron/     | plugins/cron/v2     | plugins/cron/v2.0.0     |
| plugins/async/    | plugins/async/v2    | plugins/async/v2.0.0    |
| plugins/database/ | plugins/database/v2 | plugins/database/v2.0.0 |
| plugins/cache/    | plugins/cache/v2    | plugins/cache/v2.0.0    |
| plugins/email/    | plugins/email/v2    | plugins/email/v2.0.0    |

驱动表三列均以 `plugins/database/adapter/` 为公共前缀；module 列还需补仓库前缀：

| 源码目录尾部 | module 路径尾部 | 首批 tag 尾部    |
| ------------ | --------------- | ---------------- |
| mysql/       | mysql/v2        | mysql/v2.0.0     |
| postgres/    | postgres/v2     | postgres/v2.0.0  |
| sqlite/      | sqlite/v2       | sqlite/v2.0.0    |
| oracle/      | oracle/v2       | oracle/v2.0.0    |
| sqlserver/   | sqlserver/v2    | sqlserver/v2.0.0 |

根 go.mod 若继续作为安装锚点保留，也应声明 `module github.com/bamboo-services/bamboo-base-go/v2`，对应根 tag `v2.0.0`。它不自动聚合所有子模块，安装根模块不等于安装整套插件。旧 defined/common/major 不进入新发布清单，已发布的 v1 tag 不移动、不覆盖。

特别注意：驱动模块的 /v2 在自己的模块末尾，而不是父模块后面。驱动声明 `.../plugins/database/adapter/postgres/v2`，导入公共契约则用 `.../plugins/database/v2/dialect`；父模块与嵌套 Adapter 仍是两个 go.mod。

### 整体发布记全量清单，模块只在需要时升级

首批都叫 v2.0.0，不等于以后每次都要一起加版本号。一次整体 Release 是一组经过验证的模块版本，不是把 Release 号无脑写进所有 require：

| 整体 Release（示例） | foundation  | application | email  | 其它模块           |
| -------------------- | ----------- | ----------- | ------ | ------------------ |
| v2.0.0               | v2.0.0      | v2.0.0      | v2.0.0 | 全部 v2.0.0        |
| v2.0.1               | 沿用 v2.0.0 | v2.0.1      | v2.0.1 | 沿用各自已验证版本 |

这是节奏示例，不预先宣称下一次一定修改哪两个模块。实际有代码/API 变动、依赖下限需要提高，或测试证明需要更新配套依赖时，才给该模块发新版本；没有变的模块不重打 tag。仅根级文档或清单变更，可以发布新的整体 Release 而不新发代码模块。

新增兼容 API 升模块次版本，兼容修复升补丁版本；未来某模块再次破坏 API 时，应开启新的主版本路径，而不是以整体 Release 标题替代 Go 的主版本隔离。

### 一份清单说清楚到底发布了什么

每个整体 Release 附机器可读的 `release-manifest.json`。根级记录 `release`、`sourceCommit` 与 `modules`；每个模块记录 `directory`、`module`、`version`、`tag`、`sourceCommit`。清单列出本代全部模块，不只列本次变动项；后续新增模块也要纳入。未改模块的 tag/源码提交沿用历史值，不能把本次根提交冒充它的原始发布提交。

业务希望可复现，就按清单使用明确版本；希望升级到某模块当前主版本的最新稳定版本，可以这样写：

```sh
go get github.com/bamboo-services/bamboo-base-go/application/v2@v2.0.0
go get github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2@latest
```

@latest 只负责选择这个模块路径的最新合法版本，不保证它恰好等于某个历史整体 Release 的组合。真正落地依赖由 Go 的最小版本选择规则决定，go.mod/go.sum 应记录消费端实际选出的结果；需要严格复现 Release 组合时，须校验有效模块图，而不只看命令参数。

### 发布先准备并验证，再把 tag 交出去

当前发布脚本会把全部 require 改成同一版本，这与新规则冲突，后续实施时应替换为以下流程：

1. 先确定变动模块及各自目标版本，按模块依赖关系补齐需要发布的提供者。只在确实需要提高依赖下限时改 require，不把整体 Release 号传播到未变模块。
2. 形成候选发布提交与全量清单。新增版本对应本次候选提交，沿用模块保留其原始 tag/提交，目录与 module 路径按上表校验。
3. Workspace 仅用于本地联调；发布门禁使用 GOWORK=off，禁止依赖工作区覆盖或本地 replace 偶然通过。尚未公开的候选依赖可放进隔离的临时文件模块代理供校验，不能把替换配置写入正式 go.mod。
4. 对精确候选模块图完成测试后，才创建子模块 tag。Go 源码及清单必须来自对应提交，不能“先打 tag，再补 require”。已有 tag 永不改指向。
5. 最后创建整体根 tag 与 GitHub Release，附清单；从公开可取的模块版本再做一次独立消费验证。模块代理缓存可能延迟，因此公开验证失败时不宣称安装完成。

Git 与模块代理之间没有跨系统发布事务。若中途只有部分新 tag 对外可见，保留已公开 tag，不回写它们；依据同一候选提交和清单幂等补齐，发现内容错误则发新版本。整体 Release 在清单全部可取、独立消费验证通过前不进入正式可用状态。

本轮只冻结文档中的版本与目录对应方式，不修改真实 go.mod、go.work、Makefile 或 GitHub Action，也不执行打 tag、push 或 Release。脚本实现、模块测试与公开取包是后续实施验收，不能算作本次已完成能力。规则依据见 [Go 主版本升级](https://go.dev/doc/modules/major-version) 与 [多模块仓库的版本管理](https://go.dev/ref/mod#vcs-version)。

## 备选与取舍

保留 defined/common 两层也能维持单向依赖，并且改动更少。本次合并不是因为“两层天然错误”，而是实际的键、元数据读取、错误和日志已经频繁跨模块协作，分开发布的边界收益不高；把 Gin/GORM 能力搬出去后，一份底座模块更容易看清依赖。

继续保留旧入口壳可以降低短期迁移量，但需要同时维护键类型、错误类型和启动模型的转换。用户已经选择主动迁移，因此不把这份复杂度留在新底座。

若今后八个子包需要明显不同的版本或安全更新节奏，或者业务只用 typed 却持续被迫跟随其它子包大版本迁移，应重评是否拆出独立模块。合并不是永远不能拆，边界要跟维护事实走。

版本上曾考虑保留原 module 路径，只把 GitHub Release 标成 v2；那能保住旧 import，却不能让 Go 模块本身合法发布 v2.0.0。用户现在接受 module/import 的 /v2、而不要求新增磁盘目录，因此不再保留这个分离方案。今后模块按需发版是否增加过多清单维护成本，可在连续三次发布都不得不同步升级全部模块时重评，但不能因此改写已公开版本。

## 验证

当前没有 foundation 实现。落地后在该模块目录检查：

1. `GOWORK=off go list -deps ./...` 成功，结果没有 gin-gonic、gorm.io、validator/v10；命令失败不能被“grep 没输出”冒充通过。
2. `GOWORK=off go list -f '{{.Name}} {{.ImportPath}}' ./...` 的八个顶层子包名与目录相符，不引入旧 x 前缀包名。
3. `GOWORK=off go test -race ./...` 覆盖键原语委托、日志 Context 提取、错误 Unwrap、环境值解析及 envelope 序列化。错误内部根因不直接写入对外响应。
4. foundation/env 只提供配置键和读取工具，不导入 application；应用预算默认值和校验测试放在 application。
5. 首批所有模块的 module 路径以小写 /v2 结尾，源码目录无新增 v2/ 层；检查 14 个非根模块及根占位模块（若保留）的 tag 推导，禁止出现 plugins/grpc/v2/v2.0.0。
6. 用隔离临时代理验证候选清单，再用公开版本的独立消费项目执行 GOWORK=off go test ./... 与 go list -m all；拒绝本地 replace 与未记录的依赖漂移。后续发布中未变模块保持原 tag/提交，候选失败不改写已公开 tag。

## 待确认

八大子包、主动迁移、源码目录不增 v2/ 层、module/import 使用 /v2、首批全模块 v2.0.0、后续按需发版与全量 Release 清单已经确认。本篇无剩余阻塞性版本方案选择。

真实发布脚本、独立模块构建及公开下载尚未实现和验收；这是后续实施工作，不再用“版本映射待定”掩盖已经达成的决定。
