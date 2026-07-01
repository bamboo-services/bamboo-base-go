# init 知识库

## 概述

内置初始化节点集合，存放 `Register()` 强制注册的系统级组件初始化函数。当前仅包含雪花算法节点初始化。

## 目录结构

```text
init/
└── init_snowflake.go    # SnowflakeInit() — 雪花算法默认节点初始化与验证
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 雪花算法初始化 | `init_snowflake.go` → `SnowflakeInit()` | 被注册到 `SnowflakeNodeKey` |
| 添加新的内置节点 | 在此目录新建文件，遵循 `XxxInit(ctx) (any, error)` 签名 | 然后在 `register.go` 的 `Register()` 中通过 `reg.Init.Use()` 注册 |

## 约定

- **函数签名统一**：`func(ctx context.Context) (any, error)`，与 `xRegNode.Node` 类型一致。
- **返回值会被存入 context**：返回的实例通过 `context.WithValue(ctx, key, val)` 供后续节点和请求中间件使用。
- **初始化阶段用 `xLog.NamedINIT` 命名日志器**：`xLog.WithName(xLog.NamedINIT)`，日志会标记 `[INIT]` 前缀。
- **验证后才返回**：雪花算法初始化时会生成测试 ID 验证节点可用性，不验证直接返回可能掩盖问题。

## 反模式

- **禁止在此目录放非初始化逻辑** — 此目录的职责是"组件初始化"，不是业务逻辑或工具函数。
- **禁止返回未经验证的实例** — 初始化失败要返回 error，不要返回 nil 实例。

## 调试路径

1. 雪花算法初始化失败 — 检查 `SNOWFLAKE_DATACENTER_ID` 和 `SNOWFLAKE_NODE_ID` 环境变量是否在 0-31 范围内。
2. 生成的 ID 重复 — 确认多实例部署时 datacenter_id 和 node_id 组合唯一。
