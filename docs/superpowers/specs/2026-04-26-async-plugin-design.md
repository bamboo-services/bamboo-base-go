# Async Plugin Design — 异步执行插件

## 概述

为 bamboo-base 新增 `plugins/async` 插件，提供轻量级的异步任务执行能力。核心功能是从当前请求上下文中提取组件引用（RegNodeKey），注入到独立的 goroutine 上下文中，使异步任务能访问数据库、Redis 等组件，且不受父上下文取消的影响。

## 模块结构

```
plugins/async/
├── async.go          # 核心：Async/Cancel/Wait 函数
├── task.go           # Task 类型定义
├── context.go        # 上下文分离与注入
├── go.mod            # 模块定义
├── go.sum
└── version           # 子版本号: 0.0
```

**模块路径**: `github.com/bamboo-services/bamboo-base-go/plugins/async`
**包导入别名**: `xAsync`
**依赖链**: `plugins/async → defined + common`（与 grpc 插件一致）

## 核心 API

### Task 类型

```go
// Task 代表一个异步任务的句柄
type Task struct {
    ctx    context.Context
    cancel context.CancelFunc
    done   chan struct{}
}
```

### Async — 发起异步任务

```go
func Async(parentCtx context.Context, fn func(ctx context.Context)) *Task
```

从 `parentCtx` 提取 `RegNodeKey`（ContextNodeList），创建基于 `context.Background()` 的新上下文并注入组件引用。在新 goroutine 中执行 `fn`，返回 `*Task` 句柄。

### Cancel — 强制终止

```go
func Cancel(task *Task)
```

调用 `task.cancel()` 取消上下文。异步任务通过 `ctx.Done()` 感知取消信号并主动退出。不阻塞等待。

### Wait — 等待完成

```go
func Wait(task *Task)
```

阻塞直到异步任务执行完毕（`task.done` channel 关闭）。如果任务 Panic，Wait 正常返回。

## 上下文复制策略

只复制 `RegNodeKey`（组件容器），不复制请求级数据（`RequestKey`、`UserStartTimeKey` 等）：

1. 创建 `context.Background()` 作为基础上下文
2. 从 parentCtx 提取 `RegNodeKey` → `ContextNodeList`
3. 将 `ContextNodeList` 注入新上下文的 `RegNodeKey`
4. 包装 `context.WithCancel` 以支持 Cancel 调用

这确保 `xCtxUtil.MustGetDB(ctx)`、`xCtxUtil.MustGetRDB(ctx)` 等函数在异步任务中正常工作。

## Panic 恢复

异步任务内部自动 Recover，使用全局 `slog` 记录 Panic 信息和堆栈。无论是否 Panic，`task.done` 都会关闭。

## 使用示例

```go
// 在 HTTP Handler 中发起异步任务
func (h *Handler) CreateUser(c *gin.Context) {
    var req CreateRequest
    // ... 绑定和验证 ...

    // 异步发送通知
    xAsync.Async(c.Request.Context(), func(ctx context.Context) {
        db := xCtxUtil.MustGetDB(ctx)
        // 发送邮件、写日志等异步操作
    })

    xResult.Success(c, "创建成功")
}
```

```go
// 主协程退出时的两种策略

// 策略 1：强制终止（不等待）
xAsync.Cancel(task)

// 策略 2：等待完成
xAsync.Wait(task)
```

## 设计决策

1. **基于 `context.Background()` 而非 parent** — 确保父 ctx 终止不影响异步任务
2. **只复制 `RegNodeKey`** — 异步任务是独立执行单元，不需要请求级追踪信息
3. **不使用 errgroup** — errgroup 要求父取消时子任务也取消，与需求冲突
4. **不引入新错误码** — 保持插件轻量，业务错误由调用方处理
5. **不维护全局任务池** — 简单模式，每次调用独立管理

## go.work 集成

在 `go.work` 的 `use` 块中添加 `./plugins/async`。在 Makefile 的 `PLUGINS` 变量中添加 `async`。
