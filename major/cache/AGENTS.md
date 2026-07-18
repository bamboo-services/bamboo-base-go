# cache 知识库

## 概述

泛型缓存系统（`xCache`），提供 Redis 和 Memory 双后端的统一缓存抽象。通过 `Manager` 门面自动分发到对应后端，暴露 `KeyCache` / `HashCache` / `SetCache` / `ListCache` 四种泛型缓存接口。业务侧无需直接依赖 redis 或内存库，切换后端只改一行配置。

## 目录结构

```text
cache/
├── manager.go              # Manager — 缓存门面 + 泛型工厂方法
├── manager_test.go         # Manager 单元测试
├── driver/                 # 缓存接口与类型定义（无后端依赖）
│   ├── interface.go        #   KeyCache / HashCache / SetCache / ListCache 泛型接口
│   ├── type.go             #   CacheType 枚举（Redis / Memory / None）
│   ├── key.go              #   KeyEncoder 接口 + DefaultKeyEncoder
│   └── codec.go            #   Codec 接口 + JSONCodec
├── memory/                 # 内存缓存实现
│   ├── store.go            #   Store — 分片 + TTL + janitor 后台清理
│   ├── key.go              #   memoryKeyCache 实现
│   ├── hash.go             #   memoryHashCache 实现
│   ├── set.go              #   memorySetCache 实现
│   ├── list.go             #   memoryListCache 实现
│   ├── memory_test.go      #   内存缓存功能测试
│   └── concurrent_test.go  #   内存缓存并发安全测试
└── redis/                  # Redis 缓存实现
    ├── key.go              #   redisKeyCache 实现
    ├── hash.go             #   redisHashCache 实现
    ├── set.go              #   redisSetCache 实现
    └── list.go             #   redisListCache 实现
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 创建缓存管理器 | `manager.go` → `NewManager(kind, opts...)` | 通常由 `init.CacheInit` 在启动时构造 |
| 获取键值缓存实例 | `manager.go` → `KeyCacheOf[K, V](manager)` | 返回 `KeyCache[K, V]` 接口 |
| 获取哈希缓存实例 | `manager.go` → `HashCacheOf[K, F, V, S](manager)` | 返回 `HashCache[K, F, V, S]` 接口 |
| 获取集合缓存实例 | `manager.go` → `SetCacheOf[K, V](manager)` | 返回 `SetCache[K, V]` 接口 |
| 获取列表缓存实例 | `manager.go` → `ListCacheOf[K, V](manager)` | 返回 `ListCache[K, V]` 接口 |
| 查看缓存后端类型 | `manager.go` → `Manager.Type()` | 返回 `CacheType` |
| 自定义序列化器 | 实现 `driver/codec.go` → `Codec` 接口 | 通过 `WithCodec()` 注入 |
| 自定义键编码器 | 实现 `driver/key.go` → `KeyEncoder` 接口 | 通过 `WithKeyEncoder()` 注入 |
| 关闭缓存 | `manager.go` → `Manager.Close()` | Memory 后端停止 janitor goroutine |

## 缓存接口速查

| 接口 | 核心方法 |
|------|----------|
| `KeyCache[K, V]` | `Get` / `Set` / `Exists` / `Delete` |
| `HashCache[K, F, V, S]` | `Get` / `Set` / `GetAll` / `GetAllStruct` / `SetAll` / `SetAllStruct` / `Remove` / `Delete` |
| `SetCache[K, V]` | `Add` / `Members` / `IsMember` / `Count` / `Remove` / `Delete` |
| `ListCache[K, V]` | `Prepend` / `Append` / `Range` / `Index` / `Len` / `Pop` / `PopLast` / `Remove` / `Delete` |

## 约定

- **Manager 是统一门面**：通过 `KeyCacheOf` 等工厂方法按 `CacheType` 分发到 redis 或 memory 实现，业务侧无需判断后端。
- **类型别名导出**：`driver` 包的类型通过 `manager.go` 的 type alias 重新导出到 `xCache` 命名空间，业务侧无需直接 import `cache/driver`。
- **默认 JSONCodec**：使用 `encoding/json` 序列化，可通过 `WithCodec` 替换为 gob/protobuf/msgpack。
- **泛型键通过 KeyEncoder 转 string**：`DefaultKeyEncoder` 支持 string / []byte / fmt.Stringer / 数值类型。
- **HashCache 支持结构体映射**：`GetAllStruct` / `SetAllStruct` 通过反射将 hash 字段映射到 Go 结构体。
- **Memory 后端自带 TTL 清理**：通过分片（256 片默认）+ 后台 janitor goroutine 定期清理过期条目。
- **生命周期由 Manager.Close() 管理**：Memory 后端在 Close 时停止 janitor goroutine，Redis 后端 Close 无操作。

## 反模式

- **禁止直接创建 memory 或 redis 实现** — 应通过 `Manager` + 工厂方法获取接口实例，直接创建会绕过 Codec/KeyEncoder 的配置。
- **禁止忽略 Close()** — Memory 后端若不 Close，janitor goroutine 会泄漏。
- **禁止在接口上做类型断言回具体实现** — 这违背了后端透明的设计，切换后端会 panic。
- **禁止在 `ListCache` 中使用负索引越界** — `Range` / `Index` 的实现对越界行为各后端不同（Memory panic，Redis 返回空/零值）。

## 调试路径

1. 缓存操作返回 nil — 检查 `KeyCacheOf` 是否返回了 nil（Manager 为 nil 或类型未注册时）。
2. 序列化/反序列化失败 — 检查 `Codec` 是否能处理你的类型（JSONCodec 要求字段导出）。
3. Memory 缓存内存增长 — 检查 TTL 设置和 janitor 清理间隔，大条目考虑使用 Redis。
4. Redis 操作失败 — 检查 `ContextKey.CacheManagerKey / RedisClientKey` 是否正确注入，网络是否连通。
