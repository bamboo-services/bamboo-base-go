// Package cache 提供通用泛型缓存抽象、多后端支持与受管生命周期编排。
//
// 本包为 Bamboo Base Go 框架的缓存插件核心，基于类型安全泛型设计，统一了
// 进程内分片内存存储（Memory）与分布式键值存储（Redis）两类后端的访问形态。
// 业务应用既可以作为 [application.ModeTool] 受管组件接入整个应用生命周期，
// 也可以独立构造 [Manager] 在局部逻辑中灵活使用。
//
// # 核心类型与架构
//
//   - [Manager]：缓存统一管理器门面，持有底层真实存储实例（[*memory.Store] 或 [*goredis.Client]）。
//     通过泛型工厂函数 [KeyCacheOf]、[HashCacheOf]、[SetCacheOf] 与 [ListCacheOf] 构造对应的
//     4 种数据结构操作接口；
//   - [KeyCache]：基于字符串模型的单键值对缓存；
//   - [HashCache]：基于哈希模型的二维键-字段-值缓存；
//   - [SetCache]：基于无序唯一集合模型的缓存；
//   - [ListCache]：基于有序双端列表模型的缓存；
//   - [New]：构造受管缓存插件，输出 [application.PluginInfo] 与工厂回调，交由 [application.App] 编排。
//
// # 双后端配置与切换规则
//
// 插件支持函数式选项配置与环境变量声明式装配：
//
//   - 内存模式：通过 [WithMemory] 配置程序内分片内存，支持 [WithMemoryDefaultTTL]、
//     [WithMemoryMaxEntries]、[WithMemoryShardCount] 等选项；
//   - Redis 模式：通过 [WithRedis] 配置主机地址与连接池选项，支持 [WithRedisUsername]、
//     [WithRedisPassword]、[WithRedisDB]、[WithRedisPoolSize]、[WithRedisDialTimeout] 等；
//   - 切换规则：同时调用 [WithMemory] 与 [WithRedis] 时，以最后一个生效的选项为准；
//   - 环境变量模式：通过 [FromEnv] 从环境变量自动识别 NOSQL_DRIVER（redis/memory/none）
//     并填充连接与存储参数；
//   - 零值与未选择安全：若未配置任何后端（类型为 [CacheTypeNone] 或空），[New] 将在 [application.Plugin.Start]
//     阶段明确返回初始化失败错误，绝不静默伪装已成功或伪称提供能力。
//
// # 容器依赖注入与提取
//
// 缓存插件作为 Tool 注册到应用容器后，根据所选后端在生命周期启动阶段向容器暴露能力：
//
//   - Memory 模式：提供 [ctxkey.CacheManagerKey]，类型为 [*Manager]；
//   - Redis 模式：同时提供 [ctxkey.CacheManagerKey]（[*Manager]）与
//     [ctxkey.RedisClientKey]（[*goredis.Client]），方便需要调用底层 Redis 原生命令的高级场景；
//   - 业务侧提取：通过类型安全函数 [Get] 从 [context.Context] 提取 [*Manager]；
//     若在不可恢复的关键入口，可通过 [MustGet] 提取（未注册或类型错误时 panic）。
//
// # 典型用法
//
// 1. 作为受管插件注册到 [application.App]：
//
//	app := application.New()
//	cacheInfo, cacheFactory := cache.New(
//		cache.WithMemory(
//			cache.WithMemoryDefaultTTL(30 * time.Minute),
//			cache.WithMemoryMaxEntries(10000),
//		),
//	)
//	if err := app.RegisterTool(cacheInfo, cacheFactory); err != nil {
//		return err
//	}
//
// 2. 从请求上下文获取 Manager 并进行泛型键值存取：
//
//	func handleUser(ctx context.Context, userID int64) (*User, error) {
//		mgr, err := cache.Get(ctx)
//		if err != nil {
//			return nil, err
//		}
//		userCache := cache.KeyCacheOf[int64, User](mgr)
//		if user, ok, err := userCache.Get(ctx, userID); err != nil {
//			return nil, err
//		} else if ok {
//			return user, nil
//		}
//		// 缓存未命中，回源加载...
//		return nil, nil
//	}
//
// # 写入选项与条件控制
//
// 所有的写入操作均支持函数式选项 [SetOption]：
//
//   - [WithTTL]：为单次写入指定专属 TTL，覆盖管理器默认值；
//   - [WithNX]：仅在键不存在时写入；
//   - [WithXX]：仅在键已存在时写入；
//   - [WithKeepTTL]：更新数据内容但保留原有过期时间；
//   - [WithNoSlide]：追加集合或哈希元素时不滑动延长整体过期时间。
//
// # 生命周期、并发与资源释放
//
// [*Manager] 本身完全并发安全，可跨多个协程共享。
// 独立运行的 Manager 可调用 [Manager.Close] 释放底层资源；受管模式下的插件
// 在应用收到终止信号执行逆拓扑平滑关机时，由应用协调器统一调用其 Stop 方法，
// 确保后台淘汰协程（Janitor）完全退出，并安全关闭持有的 Redis 连接池。
package cache
