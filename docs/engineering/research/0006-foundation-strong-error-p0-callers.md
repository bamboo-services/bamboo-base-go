# V2 强类型错误 P0 仓内引用与实现者台账

> 调研日期：2026-11-08 · Scope：foundation
> 承接：[P0逐符号分类](./0005-foundation-strong-error-p0-inventory.md) · [plan0002](../../plan/0002-foundation-strong-error-migration.md)

## 背景

本文件保存基线 `aaf73c684d2d108c0916bba1733ef8c608024831` 的2277处精确符号引用和133条接口实现关系，供原子切换前核对调用方。源码、测试、3个示例和2个消费测试使用同一类型快照；没有扫描仓外工程，没有运行新候选或外部服务。

`p0.consumer0` 至 `p0.consumer4` 是为3个示例和2个消费测试创建的临时类型加载包名，不是真实发布module；实现者身份以表内原始文件路径为准，不能把这些临时名称填入仓外消费者版本清单。

## 发现

### 引用与接口分发不能混同

引用来自 `go/types.Info.Uses`，包含调用、方法值/函数值、字段读写和类型引用，不把所有标识符都伪称函数调用；继承方法和匿名回调没有独立定义标识符可引用时仍保留本ID。测试变体按位置去重。接口分发只知道具体接收者类型时才可解析；以下“实现者”是方法集兼容候选，不是断言运行时每条调用必达。未找到直接引用的公开符号仍可能被仓外使用，Go/第三方反射调用的GORM钩子也不能因此删除。

### 自有接口的实现者/测试替身

下表以 `go/types.Implements` 检验真实方法集；泛型缓存接口用同一组受约束形参实例化后匹配。只列错误相关自有接口，忽略不含错误载体的Operable等普通接口。父接口与子接口会产生多条关系，不能把关系数当不同类型数；方法来源可以是继承/提升而不全在类型声明同一文件。`typed.Convertible` 的标准嵌入方法保留外部签名，不属于P3生命周期原子变更。

| 自有接口                                                                        | 实现者类型                                                                                           | 类型声明                                                                                                                       | 类别          | 涉及错误方法的定义位置                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| ------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Activator`            | `*github.com/bamboo-services/bamboo-base-go/application/v2.componentEngine`                          | [application/activator.go:96](../../../application/activator.go#L96)                                                           | SDK生产       | [Activate@565](../../../application/activator.go#L565)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.acceptanceWorkerFixture`                  | [application/activator_acceptance_test.go:15](../../../application/activator_acceptance_test.go#L15)                           | 单测替身/断言 | [Start@21](../../../application/activator_acceptance_test.go#L21)；[Stop@35](../../../application/activator_acceptance_test.go#L35)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.acceptanceComponentFixture`               | [application/activator_acceptance_test.go:42](../../../application/activator_acceptance_test.go#L42)                           | 单测替身/断言 | [Start@48](../../../application/activator_acceptance_test.go#L48)；[Stop@62](../../../application/activator_acceptance_test.go#L62)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.mockTool`                                 | [application/app_test.go:15](../../../application/app_test.go#L15)                                                             | 单测替身/断言 | [Start@20](../../../application/app_test.go#L20)；[Stop@27](../../../application/app_test.go#L27)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.mockExposingTool`                         | [application/app_test.go:35](../../../application/app_test.go#L35)                                                             | 单测替身/断言 | [Start@20](../../../application/app_test.go#L20)；[Stop@27](../../../application/app_test.go#L27)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.mockWorker`                               | [application/app_test.go:45](../../../application/app_test.go#L45)                                                             | 单测替身/断言 | [Start@20](../../../application/app_test.go#L20)；[Stop@27](../../../application/app_test.go#L27)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2/builtin.loggerPlugin`                     | [application/builtin/logging.go:84](../../../application/builtin/logging.go#L84)                                               | SDK生产       | [Start@98](../../../application/builtin/logging.go#L98)；[Stop@132](../../../application/builtin/logging.go#L132)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2/builtin.snowflakePlugin`                  | [application/builtin/snowflake.go:67](../../../application/builtin/snowflake.go#L67)                                           | SDK生产       | [Start@77](../../../application/builtin/snowflake.go#L77)；[Stop@96](../../../application/builtin/snowflake.go#L96)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2_test.testWorker`                          | [application/builtin_lifecycle_test.go:26](../../../application/builtin_lifecycle_test.go#L26)                                 | 单测替身/断言 | [Start@37](../../../application/builtin_lifecycle_test.go#L37)；[Stop@73](../../../application/builtin_lifecycle_test.go#L73)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.testComponentFixture`                     | [application/component_engine_test.go:15](../../../application/component_engine_test.go#L15)                                   | 单测替身/断言 | [Start@21](../../../application/component_engine_test.go#L21)；[Stop@35](../../../application/component_engine_test.go#L35)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.dynamicRetentionProvider`                 | [application/dynamic_retention_test.go:47](../../../application/dynamic_retention_test.go#L47)                                 | 单测替身/断言 | [Start@54](../../../application/dynamic_retention_test.go#L54)；[Stop@59](../../../application/dynamic_retention_test.go#L59)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2_test.sampleService`                       | [application/example_test.go:12](../../../application/example_test.go#L12)                                                     | 单测替身/断言 | [Start@16](../../../application/example_test.go#L16)；[Stop@20](../../../application/example_test.go#L20)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2_test.sampleWorker`                        | [application/example_test.go:24](../../../application/example_test.go#L24)                                                     | 单测替身/断言 | [Start@28](../../../application/example_test.go#L28)；[Stop@55](../../../application/example_test.go#L55)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.missingKeysPlugin`                        | [application/expose_atomic_test.go:16](../../../application/expose_atomic_test.go#L16)                                         | 单测替身/断言 | [Start@18](../../../application/expose_atomic_test.go#L18)；[Stop@19](../../../application/expose_atomic_test.go#L19)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.extraKeysPlugin`                          | [application/expose_atomic_test.go:25](../../../application/expose_atomic_test.go#L25)                                         | 单测替身/断言 | [Start@27](../../../application/expose_atomic_test.go#L27)；[Stop@28](../../../application/expose_atomic_test.go#L28)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.typedNilPlugin`                           | [application/expose_atomic_test.go:37](../../../application/expose_atomic_test.go#L37)                                         | 单测替身/断言 | [Start@39](../../../application/expose_atomic_test.go#L39)；[Stop@40](../../../application/expose_atomic_test.go#L40)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.panicExposePlugin`                        | [application/expose_atomic_test.go:49](../../../application/expose_atomic_test.go#L49)                                         | 单测替身/断言 | [Start@51](../../../application/expose_atomic_test.go#L51)；[Stop@52](../../../application/expose_atomic_test.go#L52)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.blockingExposePlugin`                     | [application/expose_atomic_test.go:58](../../../application/expose_atomic_test.go#L58)                                         | 单测替身/断言 | [Start@63](../../../application/expose_atomic_test.go#L63)；[Stop@64](../../../application/expose_atomic_test.go#L64)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.successPlugin`                            | [application/expose_atomic_test.go:74](../../../application/expose_atomic_test.go#L74)                                         | 单测替身/断言 | [Start@76](../../../application/expose_atomic_test.go#L76)；[Stop@77](../../../application/expose_atomic_test.go#L77)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.realExitMockDBProvider`                   | [application/kernel_real_exit_test.go:209](../../../application/kernel_real_exit_test.go#L209)                                 | 单测替身/断言 | [Start@214](../../../application/kernel_real_exit_test.go#L214)；[Stop@218](../../../application/kernel_real_exit_test.go#L218)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.realExitStubWorker`                       | [application/kernel_real_exit_test.go:227](../../../application/kernel_real_exit_test.go#L227)                                 | 单测替身/断言 | [Start@235](../../../application/kernel_real_exit_test.go#L235)；[Stop@257](../../../application/kernel_real_exit_test.go#L257)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.realExitComponentPlugin`                  | [application/kernel_real_exit_test.go:389](../../../application/kernel_real_exit_test.go#L389)                                 | 单测替身/断言 | [Start@394](../../../application/kernel_real_exit_test.go#L394)；[Stop@408](../../../application/kernel_real_exit_test.go#L408)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.realExitProviderTool`                     | [application/kernel_real_exit_test.go:50](../../../application/kernel_real_exit_test.go#L50)                                   | 单测替身/断言 | [Start@56](../../../application/kernel_real_exit_test.go#L56)；[Stop@61](../../../application/kernel_real_exit_test.go#L61)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.realExitConsumerTool`                     | [application/kernel_real_exit_test.go:72](../../../application/kernel_real_exit_test.go#L72)                                   | 单测替身/断言 | [Start@79](../../../application/kernel_real_exit_test.go#L79)；[Stop@91](../../../application/kernel_real_exit_test.go#L91)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.ignoreCtxWorker`                          | [application/kernel_shutdown_races_test.go:113](../../../application/kernel_shutdown_races_test.go#L113)                       | 单测替身/断言 | [Start@119](../../../application/kernel_shutdown_races_test.go#L119)；[Stop@120](../../../application/kernel_shutdown_races_test.go#L120)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.blockStartPlugin`                         | [application/kernel_shutdown_races_test.go:13](../../../application/kernel_shutdown_races_test.go#L13)                         | 单测替身/断言 | [Start@20](../../../application/kernel_shutdown_races_test.go#L20)；[Stop@30](../../../application/kernel_shutdown_races_test.go#L30)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.stubWorker`                               | [application/kernel_shutdown_races_test.go:244](../../../application/kernel_shutdown_races_test.go#L244)                       | 单测替身/断言 | [Start@248](../../../application/kernel_shutdown_races_test.go#L248)；[Stop@249](../../../application/kernel_shutdown_races_test.go#L249)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.stubRunnable`                             | [application/kernel_shutdown_races_test.go:361](../../../application/kernel_shutdown_races_test.go#L361)                       | 单测替身/断言 | [Start@365](../../../application/kernel_shutdown_races_test.go#L365)；[Stop@366](../../../application/kernel_shutdown_races_test.go#L366)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.quiesceTestProviderTool`                  | [application/quiesce_budget_test.go:18](../../../application/quiesce_budget_test.go#L18)                                       | 单测替身/断言 | [Start@23](../../../application/quiesce_budget_test.go#L23)；[Stop@27](../../../application/quiesce_budget_test.go#L27)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.startFailProviderTool`                    | [application/quiesce_budget_test.go:212](../../../application/quiesce_budget_test.go#L212)                                     | 单测替身/断言 | [Start@218](../../../application/quiesce_budget_test.go#L218)；[Stop@223](../../../application/quiesce_budget_test.go#L223)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.startFailConsumerTool`                    | [application/quiesce_budget_test.go:234](../../../application/quiesce_budget_test.go#L234)                                     | 单测替身/断言 | [Start@239](../../../application/quiesce_budget_test.go#L239)；[Stop@247](../../../application/quiesce_budget_test.go#L247)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.quiesceBlockingWorker`                    | [application/quiesce_budget_test.go:38](../../../application/quiesce_budget_test.go#L38)                                       | 单测替身/断言 | [Start@44](../../../application/quiesce_budget_test.go#L44)；[Stop@64](../../../application/quiesce_budget_test.go#L64)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.mockRetainedPlugin`                       | [application/retained_dependency_test.go:14](../../../application/retained_dependency_test.go#L14)                             | 单测替身/断言 | [Start@19](../../../application/retained_dependency_test.go#L19)；[Stop@26](../../../application/retained_dependency_test.go#L26)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.mockRetainedExposer`                      | [application/retained_dependency_test.go:33](../../../application/retained_dependency_test.go#L33)                             | 单测替身/断言 | [Start@19](../../../application/retained_dependency_test.go#L19)；[Stop@26](../../../application/retained_dependency_test.go#L26)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.witnessWorker`                            | [application/stop_budget_test.go:14](../../../application/stop_budget_test.go#L14)                                             | 单测替身/断言 | [Start@18](../../../application/stop_budget_test.go#L18)；[Stop@28](../../../application/stop_budget_test.go#L28)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.blockingStopPlugin`                       | [application/stop_budget_test.go:32](../../../application/stop_budget_test.go#L32)                                             | 单测替身/断言 | [Start@37](../../../application/stop_budget_test.go#L37)；[Stop@41](../../../application/stop_budget_test.go#L41)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.dummyProviderPlugin`                      | [application/stop_budget_test.go:47](../../../application/stop_budget_test.go#L47)                                             | 单测替身/断言 | [Start@53](../../../application/stop_budget_test.go#L53)；[Stop@58](../../../application/stop_budget_test.go#L58)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.blockingMockPlugin`                       | [application/stop_completion_test.go:14](../../../application/stop_completion_test.go#L14)                                     | 单测替身/断言 | [Start@21](../../../application/stop_completion_test.go#L21)；[Stop@25](../../../application/stop_completion_test.go#L25)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.dummyTypedNilPlugin`                      | [application/typed_nil_test.go:12](../../../application/typed_nil_test.go#L12)                                                 | 单测替身/断言 | [Start@17](../../../application/typed_nil_test.go#L17)；[Stop@22](../../../application/typed_nil_test.go#L22)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/application/v2.typedNilExposePlugin`                     | [application/typed_nil_test.go:27](../../../application/typed_nil_test.go#L27)                                                 | 单测替身/断言 | [Start@31](../../../application/typed_nil_test.go#L31)；[Stop@32](../../../application/typed_nil_test.go#L32)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*p0.consumer2.cronTargetComponent`                                                                  | [examples/v2/grpc-cron-async/main.go:23](../../../examples/v2/grpc-cron-async/main.go#L23)                                     | 消费者        | [Start@27](../../../examples/v2/grpc-cron-async/main.go#L27)；[Stop@53](../../../examples/v2/grpc-cron-async/main.go#L53)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*p0.consumer0.ConfigTool`                                                                           | [examples/v2/worker-tool/main.go:32](../../../examples/v2/worker-tool/main.go#L32)                                             | 消费者        | [Start@37](../../../examples/v2/worker-tool/main.go#L37)；[Stop@42](../../../examples/v2/worker-tool/main.go#L42)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*p0.consumer0.PrinterWorker`                                                                        | [examples/v2/worker-tool/main.go:71](../../../examples/v2/worker-tool/main.go#L71)                                             | 消费者        | [Start@78](../../../examples/v2/worker-tool/main.go#L78)；[Stop@118](../../../examples/v2/worker-tool/main.go#L118)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.testResourceProviderTool`               | [plugins/async/dependency_lifecycle_test.go:45](../../../plugins/async/dependency_lifecycle_test.go#L45)                       | 单测替身/断言 | [Start@51](../../../plugins/async/dependency_lifecycle_test.go#L51)；[Stop@56](../../../plugins/async/dependency_lifecycle_test.go#L56)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.testLifecycleWorker`                    | [plugins/async/dependency_lifecycle_test.go:67](../../../plugins/async/dependency_lifecycle_test.go#L67)                       | 单测替身/断言 | [Start@71](../../../plugins/async/dependency_lifecycle_test.go#L71)；[Stop@83](../../../plugins/async/dependency_lifecycle_test.go#L83)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.Plugin`                                 | [plugins/async/plugin.go:149](../../../plugins/async/plugin.go#L149)                                                           | SDK生产       | [Start@202](../../../plugins/async/plugin.go#L202)；[Stop@242](../../../plugins/async/plugin.go#L242)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.testWorker`                             | [plugins/async/plugin_test.go:13](../../../plugins/async/plugin_test.go#L13)                                                   | 单测替身/断言 | [Start@18](../../../plugins/async/plugin_test.go#L18)；[Stop@28](../../../plugins/async/plugin_test.go#L28)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2.pluginImpl`                             | [plugins/cache/plugin.go:20](../../../plugins/cache/plugin.go#L20)                                                             | SDK生产       | [Start@94](../../../plugins/cache/plugin.go#L94)；[Stop@166](../../../plugins/cache/plugin.go#L166)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2.testLifetimeWorker`                     | [plugins/cache/plugin_test.go:104](../../../plugins/cache/plugin_test.go#L104)                                                 | 单测替身/断言 | [Start@109](../../../plugins/cache/plugin_test.go#L109)；[Stop@120](../../../plugins/cache/plugin_test.go#L120)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/cron/v2.Plugin`                                  | [plugins/cron/plugin.go:138](../../../plugins/cron/plugin.go#L138)                                                             | SDK生产       | [Start@198](../../../plugins/cron/plugin.go#L198)；[Stop@296](../../../plugins/cron/plugin.go#L296)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/cron/v2.realAppTestComponent`                    | [plugins/cron/real_app_test.go:53](../../../plugins/cron/real_app_test.go#L53)                                                 | 单测替身/断言 | [Start@58](../../../plugins/cron/real_app_test.go#L58)；[Stop@73](../../../plugins/cron/real_app_test.go#L73)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/cron/v2_test.barrierComponent`                   | [plugins/cron/real_queue_test.go:47](../../../plugins/cron/real_queue_test.go#L47)                                             | 单测替身/断言 | [Start@55](../../../plugins/cron/real_queue_test.go#L55)；[Stop@70](../../../plugins/cron/real_queue_test.go#L70)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/mysql/v2_test.witnessWorker`    | [plugins/database/adapter/mysql/integration_test.go:31](../../../plugins/database/adapter/mysql/integration_test.go#L31)       | 单测替身/断言 | [Start@48](../../../plugins/database/adapter/mysql/integration_test.go#L48)；[Stop@67](../../../plugins/database/adapter/mysql/integration_test.go#L67)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/postgres/v2_test.witnessWorker` | [plugins/database/adapter/postgres/integration_test.go:18](../../../plugins/database/adapter/postgres/integration_test.go#L18) | 单测替身/断言 | [Start@23](../../../plugins/database/adapter/postgres/integration_test.go#L23)；[Stop@37](../../../plugins/database/adapter/postgres/integration_test.go#L37)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/database/v2/models.testSnowflakeTool`            | [plugins/database/models/models_test.go:41](../../../plugins/database/models/models_test.go#L41)                               | 单测替身/断言 | [Start@45](../../../plugins/database/models/models_test.go#L45)；[Stop@49](../../../plugins/database/models/models_test.go#L49)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/database/v2/models.testWorker`                   | [plugins/database/models/models_test.go:59](../../../plugins/database/models/models_test.go#L59)                               | 单测替身/断言 | [Start@63](../../../plugins/database/models/models_test.go#L63)；[Stop@67](../../../plugins/database/models/models_test.go#L67)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/database/v2.Plugin`                              | [plugins/database/plugin.go:24](../../../plugins/database/plugin.go#L24)                                                       | SDK生产       | [Start@65](../../../plugins/database/plugin.go#L65)；[Stop@147](../../../plugins/database/plugin.go#L147)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/email/v2.Plugin`                                 | [plugins/email/client.go:125](../../../plugins/email/client.go#L125)                                                           | SDK生产       | [Start@174](../../../plugins/email/client.go#L174)；[Stop@302](../../../plugins/email/client.go#L302)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/email/v2_test.emailWorker`                       | [plugins/email/integration_smtp_test.go:46](../../../plugins/email/integration_smtp_test.go#L46)                               | 单测替身/断言 | [Start@51](../../../plugins/email/integration_smtp_test.go#L51)；[Stop@62](../../../plugins/email/integration_smtp_test.go#L62)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2.Plugin`                                  | [plugins/grpc/plugin.go:270](../../../plugins/grpc/plugin.go#L270)                                                             | SDK生产       | [Start@396](../../../plugins/grpc/plugin.go#L396)；[Stop@505](../../../plugins/grpc/plugin.go#L505)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2_test.toolResourcePlugin`                 | [plugins/grpc/plugin_app_context_test.go:295](../../../plugins/grpc/plugin_app_context_test.go#L295)                           | 单测替身/断言 | [Start@300](../../../plugins/grpc/plugin_app_context_test.go#L300)；[Stop@304](../../../plugins/grpc/plugin_app_context_test.go#L304)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2_test.witnessPlugin`                      | [plugins/grpc/plugin_app_context_test.go:314](../../../plugins/grpc/plugin_app_context_test.go#L314)                           | 单测替身/断言 | [Start@318](../../../plugins/grpc/plugin_app_context_test.go#L318)；[Stop@328](../../../plugins/grpc/plugin_app_context_test.go#L328)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2_test.dummyComponent`                     | [plugins/grpc/plugin_app_context_test.go:332](../../../plugins/grpc/plugin_app_context_test.go#L332)                           | 单测替身/断言 | [Start@336](../../../plugins/grpc/plugin_app_context_test.go#L336)；[Stop@349](../../../plugins/grpc/plugin_app_context_test.go#L349)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2.Plugin`                               | [plugins/web-gin/plugin.go:231](../../../plugins/web-gin/plugin.go#L231)                                                       | SDK生产       | [Start@358](../../../plugins/web-gin/plugin.go#L358)；[Stop@516](../../../plugins/web-gin/plugin.go#L516)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2_test.toolPlugin`                      | [plugins/web-gin/plugin_isolation_test.go:37](../../../plugins/web-gin/plugin_isolation_test.go#L37)                           | 单测替身/断言 | [Start@42](../../../plugins/web-gin/plugin_isolation_test.go#L42)；[Stop@46](../../../plugins/web-gin/plugin_isolation_test.go#L46)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2_test.witnessWorker`                   | [plugins/web-gin/plugin_isolation_test.go:56](../../../plugins/web-gin/plugin_isolation_test.go#L56)                           | 单测替身/断言 | [Start@68](../../../plugins/web-gin/plugin_isolation_test.go#L68)；[Stop@85](../../../plugins/web-gin/plugin_isolation_test.go#L85)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2_test.testWorkerLifetime`              | [plugins/web-gin/plugin_test.go:602](../../../plugins/web-gin/plugin_test.go#L602)                                             | 单测替身/断言 | [Start@607](../../../plugins/web-gin/plugin_test.go#L607)；[Stop@613](../../../plugins/web-gin/plugin_test.go#L613)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*p0.consumer3_test.readyWeb`                                                                        | [test/consumer/v2-http-db-cache/lifecycle_test.go:43](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L43)           | 消费者        | [Start@48](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L48)；[Stop@516](../../../plugins/web-gin/plugin.go#L516)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`               | `*p0.consumer4_test.lifecycleTestWorker`                                                             | [test/consumer/v2-sqlite-async/lifecycle_test.go:21](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L21)             | 消费者        | [Start@25](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L25)；[Stop@26](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L26)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`             | `*github.com/bamboo-services/bamboo-base-go/application/v2_test.testWorker`                          | [application/builtin_lifecycle_test.go:26](../../../application/builtin_lifecycle_test.go#L26)                                 | 单测替身/断言 | [Quiesce@83](../../../application/builtin_lifecycle_test.go#L83)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`             | `*github.com/bamboo-services/bamboo-base-go/application/v2.quiesceBlockingWorker`                    | [application/quiesce_budget_test.go:38](../../../application/quiesce_budget_test.go#L38)                                       | 单测替身/断言 | [Quiesce@58](../../../application/quiesce_budget_test.go#L58)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`             | `*p0.consumer2.cronTargetComponent`                                                                  | [examples/v2/grpc-cron-async/main.go:23](../../../examples/v2/grpc-cron-async/main.go#L23)                                     | 消费者        | [Quiesce@49](../../../examples/v2/grpc-cron-async/main.go#L49)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`             | `*github.com/bamboo-services/bamboo-base-go/plugins/cron/v2.Plugin`                                  | [plugins/cron/plugin.go:138](../../../plugins/cron/plugin.go#L138)                                                             | SDK生产       | [Quiesce@286](../../../plugins/cron/plugin.go#L286)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`             | `*github.com/bamboo-services/bamboo-base-go/plugins/cron/v2.realAppTestComponent`                    | [plugins/cron/real_app_test.go:53](../../../plugins/cron/real_app_test.go#L53)                                                 | 单测替身/断言 | [Quiesce@69](../../../plugins/cron/real_app_test.go#L69)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`             | `*github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2.Plugin`                                  | [plugins/grpc/plugin.go:270](../../../plugins/grpc/plugin.go#L270)                                                             | SDK生产       | [Quiesce@490](../../../plugins/grpc/plugin.go#L490)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`             | `*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2.Plugin`                               | [plugins/web-gin/plugin.go:231](../../../plugins/web-gin/plugin.go#L231)                                                       | SDK生产       | [Quiesce@500](../../../plugins/web-gin/plugin.go#L500)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`             | `p0.consumer3_test.readyWeb`                                                                         | [test/consumer/v2-http-db-cache/lifecycle_test.go:43](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L43)           | 消费者        | [Quiesce@500](../../../plugins/web-gin/plugin.go#L500)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunHandle`            | `*github.com/bamboo-services/bamboo-base-go/application/v2.runHandleImpl`                            | [application/supervisor.go:12](../../../application/supervisor.go#L12)                                                         | SDK生产       | [Wait@35](../../../application/supervisor.go#L35)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.acceptanceWorkerFixture`                  | [application/activator_acceptance_test.go:15](../../../application/activator_acceptance_test.go#L15)                           | 单测替身/断言 | [Run@28](../../../application/activator_acceptance_test.go#L28)；[Start@21](../../../application/activator_acceptance_test.go#L21)；[Stop@35](../../../application/activator_acceptance_test.go#L35)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.acceptanceComponentFixture`               | [application/activator_acceptance_test.go:42](../../../application/activator_acceptance_test.go#L42)                           | 单测替身/断言 | [Run@55](../../../application/activator_acceptance_test.go#L55)；[Start@48](../../../application/activator_acceptance_test.go#L48)；[Stop@62](../../../application/activator_acceptance_test.go#L62)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.mockWorker`                               | [application/app_test.go:45](../../../application/app_test.go#L45)                                                             | 单测替身/断言 | [Run@50](../../../application/app_test.go#L50)；[Start@20](../../../application/app_test.go#L20)；[Stop@27](../../../application/app_test.go#L27)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2_test.testWorker`                          | [application/builtin_lifecycle_test.go:26](../../../application/builtin_lifecycle_test.go#L26)                                 | 单测替身/断言 | [Run@54](../../../application/builtin_lifecycle_test.go#L54)；[Start@37](../../../application/builtin_lifecycle_test.go#L37)；[Stop@73](../../../application/builtin_lifecycle_test.go#L73)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.testComponentFixture`                     | [application/component_engine_test.go:15](../../../application/component_engine_test.go#L15)                                   | 单测替身/断言 | [Run@28](../../../application/component_engine_test.go#L28)；[Start@21](../../../application/component_engine_test.go#L21)；[Stop@35](../../../application/component_engine_test.go#L35)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2_test.sampleWorker`                        | [application/example_test.go:24](../../../application/example_test.go#L24)                                                     | 单测替身/断言 | [Run@32](../../../application/example_test.go#L32)；[Start@28](../../../application/example_test.go#L28)；[Stop@55](../../../application/example_test.go#L55)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.realExitStubWorker`                       | [application/kernel_real_exit_test.go:227](../../../application/kernel_real_exit_test.go#L227)                                 | 单测替身/断言 | [Run@239](../../../application/kernel_real_exit_test.go#L239)；[Start@235](../../../application/kernel_real_exit_test.go#L235)；[Stop@257](../../../application/kernel_real_exit_test.go#L257)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.realExitComponentPlugin`                  | [application/kernel_real_exit_test.go:389](../../../application/kernel_real_exit_test.go#L389)                                 | 单测替身/断言 | [Run@398](../../../application/kernel_real_exit_test.go#L398)；[Start@394](../../../application/kernel_real_exit_test.go#L394)；[Stop@408](../../../application/kernel_real_exit_test.go#L408)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.ignoreCtxWorker`                          | [application/kernel_shutdown_races_test.go:113](../../../application/kernel_shutdown_races_test.go#L113)                       | 单测替身/断言 | [Run@130](../../../application/kernel_shutdown_races_test.go#L130)；[Start@119](../../../application/kernel_shutdown_races_test.go#L119)；[Stop@120](../../../application/kernel_shutdown_races_test.go#L120)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.stubWorker`                               | [application/kernel_shutdown_races_test.go:244](../../../application/kernel_shutdown_races_test.go#L244)                       | 单测替身/断言 | [Run@250](../../../application/kernel_shutdown_races_test.go#L250)；[Start@248](../../../application/kernel_shutdown_races_test.go#L248)；[Stop@249](../../../application/kernel_shutdown_races_test.go#L249)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.stubRunnable`                             | [application/kernel_shutdown_races_test.go:361](../../../application/kernel_shutdown_races_test.go#L361)                       | 单测替身/断言 | [Run@367](../../../application/kernel_shutdown_races_test.go#L367)；[Start@365](../../../application/kernel_shutdown_races_test.go#L365)；[Stop@366](../../../application/kernel_shutdown_races_test.go#L366)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.quiesceBlockingWorker`                    | [application/quiesce_budget_test.go:38](../../../application/quiesce_budget_test.go#L38)                                       | 单测替身/断言 | [Run@48](../../../application/quiesce_budget_test.go#L48)；[Start@44](../../../application/quiesce_budget_test.go#L44)；[Stop@64](../../../application/quiesce_budget_test.go#L64)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/application/v2.witnessWorker`                            | [application/stop_budget_test.go:14](../../../application/stop_budget_test.go#L14)                                             | 单测替身/断言 | [Run@22](../../../application/stop_budget_test.go#L22)；[Start@18](../../../application/stop_budget_test.go#L18)；[Stop@28](../../../application/stop_budget_test.go#L28)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*p0.consumer2.cronTargetComponent`                                                                  | [examples/v2/grpc-cron-async/main.go:23](../../../examples/v2/grpc-cron-async/main.go#L23)                                     | 消费者        | [Run@36](../../../examples/v2/grpc-cron-async/main.go#L36)；[Start@27](../../../examples/v2/grpc-cron-async/main.go#L27)；[Stop@53](../../../examples/v2/grpc-cron-async/main.go#L53)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*p0.consumer0.PrinterWorker`                                                                        | [examples/v2/worker-tool/main.go:71](../../../examples/v2/worker-tool/main.go#L71)                                             | 消费者        | [Run@100](../../../examples/v2/worker-tool/main.go#L100)；[Start@78](../../../examples/v2/worker-tool/main.go#L78)；[Stop@118](../../../examples/v2/worker-tool/main.go#L118)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.testLifecycleWorker`                    | [plugins/async/dependency_lifecycle_test.go:67](../../../plugins/async/dependency_lifecycle_test.go#L67)                       | 单测替身/断言 | [Run@75](../../../plugins/async/dependency_lifecycle_test.go#L75)；[Start@71](../../../plugins/async/dependency_lifecycle_test.go#L71)；[Stop@83](../../../plugins/async/dependency_lifecycle_test.go#L83)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.testWorker`                             | [plugins/async/plugin_test.go:13](../../../plugins/async/plugin_test.go#L13)                                                   | 单测替身/断言 | [Run@22](../../../plugins/async/plugin_test.go#L22)；[Start@18](../../../plugins/async/plugin_test.go#L18)；[Stop@28](../../../plugins/async/plugin_test.go#L28)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2.testLifetimeWorker`                     | [plugins/cache/plugin_test.go:104](../../../plugins/cache/plugin_test.go#L104)                                                 | 单测替身/断言 | [Run@113](../../../plugins/cache/plugin_test.go#L113)；[Start@109](../../../plugins/cache/plugin_test.go#L109)；[Stop@120](../../../plugins/cache/plugin_test.go#L120)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/cron/v2.Plugin`                                  | [plugins/cron/plugin.go:138](../../../plugins/cron/plugin.go#L138)                                                             | SDK生产       | [Run@270](../../../plugins/cron/plugin.go#L270)；[Start@198](../../../plugins/cron/plugin.go#L198)；[Stop@296](../../../plugins/cron/plugin.go#L296)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/cron/v2.realAppTestComponent`                    | [plugins/cron/real_app_test.go:53](../../../plugins/cron/real_app_test.go#L53)                                                 | 单测替身/断言 | [Run@62](../../../plugins/cron/real_app_test.go#L62)；[Start@58](../../../plugins/cron/real_app_test.go#L58)；[Stop@73](../../../plugins/cron/real_app_test.go#L73)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/cron/v2_test.barrierComponent`                   | [plugins/cron/real_queue_test.go:47](../../../plugins/cron/real_queue_test.go#L47)                                             | 单测替身/断言 | [Run@57](../../../plugins/cron/real_queue_test.go#L57)；[Start@55](../../../plugins/cron/real_queue_test.go#L55)；[Stop@70](../../../plugins/cron/real_queue_test.go#L70)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/mysql/v2_test.witnessWorker`    | [plugins/database/adapter/mysql/integration_test.go:31](../../../plugins/database/adapter/mysql/integration_test.go#L31)       | 单测替身/断言 | [Run@54](../../../plugins/database/adapter/mysql/integration_test.go#L54)；[Start@48](../../../plugins/database/adapter/mysql/integration_test.go#L48)；[Stop@67](../../../plugins/database/adapter/mysql/integration_test.go#L67)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/postgres/v2_test.witnessWorker` | [plugins/database/adapter/postgres/integration_test.go:18](../../../plugins/database/adapter/postgres/integration_test.go#L18) | 单测替身/断言 | [Run@27](../../../plugins/database/adapter/postgres/integration_test.go#L27)；[Start@23](../../../plugins/database/adapter/postgres/integration_test.go#L23)；[Stop@37](../../../plugins/database/adapter/postgres/integration_test.go#L37)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/database/v2/models.testWorker`                   | [plugins/database/models/models_test.go:59](../../../plugins/database/models/models_test.go#L59)                               | 单测替身/断言 | [Run@71](../../../plugins/database/models/models_test.go#L71)；[Start@63](../../../plugins/database/models/models_test.go#L63)；[Stop@67](../../../plugins/database/models/models_test.go#L67)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/email/v2_test.emailWorker`                       | [plugins/email/integration_smtp_test.go:46](../../../plugins/email/integration_smtp_test.go#L46)                               | 单测替身/断言 | [Run@55](../../../plugins/email/integration_smtp_test.go#L55)；[Start@51](../../../plugins/email/integration_smtp_test.go#L51)；[Stop@62](../../../plugins/email/integration_smtp_test.go#L62)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2.Plugin`                                  | [plugins/grpc/plugin.go:270](../../../plugins/grpc/plugin.go#L270)                                                             | SDK生产       | [Run@469](../../../plugins/grpc/plugin.go#L469)；[Start@396](../../../plugins/grpc/plugin.go#L396)；[Stop@505](../../../plugins/grpc/plugin.go#L505)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2_test.witnessPlugin`                      | [plugins/grpc/plugin_app_context_test.go:314](../../../plugins/grpc/plugin_app_context_test.go#L314)                           | 单测替身/断言 | [Run@322](../../../plugins/grpc/plugin_app_context_test.go#L322)；[Start@318](../../../plugins/grpc/plugin_app_context_test.go#L318)；[Stop@328](../../../plugins/grpc/plugin_app_context_test.go#L328)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2_test.dummyComponent`                     | [plugins/grpc/plugin_app_context_test.go:332](../../../plugins/grpc/plugin_app_context_test.go#L332)                           | 单测替身/断言 | [Run@340](../../../plugins/grpc/plugin_app_context_test.go#L340)；[Start@336](../../../plugins/grpc/plugin_app_context_test.go#L336)；[Stop@349](../../../plugins/grpc/plugin_app_context_test.go#L349)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2.Plugin`                               | [plugins/web-gin/plugin.go:231](../../../plugins/web-gin/plugin.go#L231)                                                       | SDK生产       | [Run@474](../../../plugins/web-gin/plugin.go#L474)；[Start@358](../../../plugins/web-gin/plugin.go#L358)；[Stop@516](../../../plugins/web-gin/plugin.go#L516)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2_test.witnessWorker`                   | [plugins/web-gin/plugin_isolation_test.go:56](../../../plugins/web-gin/plugin_isolation_test.go#L56)                           | 单测替身/断言 | [Run@72](../../../plugins/web-gin/plugin_isolation_test.go#L72)；[Start@68](../../../plugins/web-gin/plugin_isolation_test.go#L68)；[Stop@85](../../../plugins/web-gin/plugin_isolation_test.go#L85)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2_test.testWorkerLifetime`              | [plugins/web-gin/plugin_test.go:602](../../../plugins/web-gin/plugin_test.go#L602)                                             | 单测替身/断言 | [Run@608](../../../plugins/web-gin/plugin_test.go#L608)；[Start@607](../../../plugins/web-gin/plugin_test.go#L607)；[Stop@613](../../../plugins/web-gin/plugin_test.go#L613)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*p0.consumer3_test.readyWeb`                                                                        | [test/consumer/v2-http-db-cache/lifecycle_test.go:43](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L43)           | 消费者        | [Run@474](../../../plugins/web-gin/plugin.go#L474)；[Start@48](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L48)；[Stop@516](../../../plugins/web-gin/plugin.go#L516)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`       | `*p0.consumer4_test.lifecycleTestWorker`                                                             | [test/consumer/v2-sqlite-async/lifecycle_test.go:21](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L21)             | 消费者        | [Run@27](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L27)；[Start@25](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L25)；[Stop@26](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L26)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`     | `*github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey.Key`                                | [foundation/ctxkey/key.go:17](../../../foundation/ctxkey/key.go#L17)                                                           | SDK生产       | [MarshalBinary@94](../../../foundation/ctxkey/key.go#L94)；[MarshalJSON@115](../../../foundation/ctxkey/key.go#L115)；[MarshalText@72](../../../foundation/ctxkey/key.go#L72)；[MarshalXML@136](../../../foundation/ctxkey/key.go#L136)；[MarshalXMLAttr@157](../../../foundation/ctxkey/key.go#L157)；[Scan@185](../../../foundation/ctxkey/key.go#L185)；[Set@200](../../../foundation/ctxkey/key.go#L200)；[UnmarshalBinary@101](../../../foundation/ctxkey/key.go#L101)；[UnmarshalJSON@122](../../../foundation/ctxkey/key.go#L122)；[UnmarshalText@79](../../../foundation/ctxkey/key.go#L79)；[UnmarshalXML@143](../../../foundation/ctxkey/key.go#L143)；[UnmarshalXMLAttr@164](../../../foundation/ctxkey/key.go#L164)；[Value@178](../../../foundation/ctxkey/key.go#L178)                                                                       |
| `github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`     | `*github.com/bamboo-services/bamboo-base-go/foundation/v2/env.Key`                                   | [foundation/env/key.go:18](../../../foundation/env/key.go#L18)                                                                 | SDK生产       | [MarshalBinary@89](../../../foundation/env/key.go#L89)；[MarshalJSON@112](../../../foundation/env/key.go#L112)；[MarshalText@66](../../../foundation/env/key.go#L66)；[MarshalXML@135](../../../foundation/env/key.go#L135)；[MarshalXMLAttr@158](../../../foundation/env/key.go#L158)；[Scan@189](../../../foundation/env/key.go#L189)；[Set@205](../../../foundation/env/key.go#L205)；[UnmarshalBinary@97](../../../foundation/env/key.go#L97)；[UnmarshalJSON@120](../../../foundation/env/key.go#L120)；[UnmarshalText@74](../../../foundation/env/key.go#L74)；[UnmarshalXML@143](../../../foundation/env/key.go#L143)；[UnmarshalXMLAttr@166](../../../foundation/env/key.go#L166)；[Value@181](../../../foundation/env/key.go#L181)                                                                                                                |
| `github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`     | `*github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.String`                              | [foundation/typed/index.go:28](../../../foundation/typed/index.go#L28)                                                         | SDK生产       | [MarshalBinary@47](../../../foundation/typed/convert.go#L47)；[MarshalJSON@71](../../../foundation/typed/convert.go#L71)；[MarshalText@22](../../../foundation/typed/convert.go#L22)；[MarshalXML@105](../../../foundation/typed/convert.go#L105)；[MarshalXMLAttr@157](../../../foundation/typed/convert.go#L157)；[Scan@195](../../../foundation/typed/convert.go#L195)；[Set@224](../../../foundation/typed/convert.go#L224)；[UnmarshalBinary@57](../../../foundation/typed/convert.go#L57)；[UnmarshalJSON@81](../../../foundation/typed/convert.go#L81)；[UnmarshalText@32](../../../foundation/typed/convert.go#L32)；[UnmarshalXML@116](../../../foundation/typed/convert.go#L116)；[UnmarshalXMLAttr@170](../../../foundation/typed/convert.go#L170)；[Value@184](../../../foundation/typed/convert.go#L184)                                      |
| `github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`     | `*github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.mockDomainKey`                       | [foundation/typed/index_test.go:25](../../../foundation/typed/index_test.go#L25)                                               | 单测替身/断言 | [MarshalBinary@64](../../../foundation/typed/index_test.go#L64)；[MarshalJSON@80](../../../foundation/typed/index_test.go#L80)；[MarshalText@48](../../../foundation/typed/index_test.go#L48)；[MarshalXML@96](../../../foundation/typed/index_test.go#L96)；[MarshalXMLAttr@112](../../../foundation/typed/index_test.go#L112)；[Scan@132](../../../foundation/typed/index_test.go#L132)；[Set@144](../../../foundation/typed/index_test.go#L144)；[UnmarshalBinary@68](../../../foundation/typed/index_test.go#L68)；[UnmarshalJSON@84](../../../foundation/typed/index_test.go#L84)；[UnmarshalText@52](../../../foundation/typed/index_test.go#L52)；[UnmarshalXML@100](../../../foundation/typed/index_test.go#L100)；[UnmarshalXMLAttr@116](../../../foundation/typed/index_test.go#L116)；[Value@128](../../../foundation/typed/index_test.go#L128) |
| `github.com/bamboo-services/bamboo-base-go/plugins/async/v2.Pool`               | `*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.Plugin`                                 | [plugins/async/plugin.go:149](../../../plugins/async/plugin.go#L149)                                                           | SDK生产       | [Submit@315](../../../plugins/async/plugin.go#L315)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.Codec`       | `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.JSONCodec`                        | [plugins/cache/driver/codec.go:20](../../../plugins/cache/driver/codec.go#L20)                                                 | SDK生产       | [Marshal@23](../../../plugins/cache/driver/codec.go#L23)；[Unmarshal@26](../../../plugins/cache/driver/codec.go#L26)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.Codec`       | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.failingCodec`                     | [plugins/cache/redis/integration_test.go:32](../../../plugins/cache/redis/integration_test.go#L32)                             | 单测替身/断言 | [Marshal@37](../../../plugins/cache/redis/integration_test.go#L37)；[Unmarshal@44](../../../plugins/cache/redis/integration_test.go#L44)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`   | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K, F, V, S]`           | [plugins/cache/memory/hash.go:18](../../../plugins/cache/memory/hash.go#L18)                                                   | SDK生产       | [Delete@300](../../../plugins/cache/memory/hash.go#L300)；[Exists@261](../../../plugins/cache/memory/hash.go#L261)；[Get@34](../../../plugins/cache/memory/hash.go#L34)；[GetAll@110](../../../plugins/cache/memory/hash.go#L110)；[GetAllStruct@134](../../../plugins/cache/memory/hash.go#L134)；[Remove@278](../../../plugins/cache/memory/hash.go#L278)；[Set@66](../../../plugins/cache/memory/hash.go#L66)；[SetAll@161](../../../plugins/cache/memory/hash.go#L161)；[SetAllStruct@243](../../../plugins/cache/memory/hash.go#L243)                                                                                                                                                                                                                                                                                                                 |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`   | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K, F, V, S]`            | [plugins/cache/redis/hash.go:17](../../../plugins/cache/redis/hash.go#L17)                                                     | SDK生产       | [Delete@244](../../../plugins/cache/redis/hash.go#L244)；[Exists@223](../../../plugins/cache/redis/hash.go#L223)；[Get@40](../../../plugins/cache/redis/hash.go#L40)；[GetAll@108](../../../plugins/cache/redis/hash.go#L108)；[GetAllStruct@134](../../../plugins/cache/redis/hash.go#L134)；[Remove@232](../../../plugins/cache/redis/hash.go#L232)；[Set@63](../../../plugins/cache/redis/hash.go#L63)；[SetAll@157](../../../plugins/cache/redis/hash.go#L157)；[SetAllStruct@191](../../../plugins/cache/redis/hash.go#L191)                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`    | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.KeyCache[K, V]`                  | [plugins/cache/memory/key.go:11](../../../plugins/cache/memory/key.go#L11)                                                     | SDK生产       | [Delete@84](../../../plugins/cache/memory/key.go#L84)；[Exists@79](../../../plugins/cache/memory/key.go#L79)；[Get@33](../../../plugins/cache/memory/key.go#L33)；[Set@59](../../../plugins/cache/memory/key.go#L59)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`    | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.KeyCache[K, V]`                   | [plugins/cache/redis/key.go:13](../../../plugins/cache/redis/key.go#L13)                                                       | SDK生产       | [Delete@105](../../../plugins/cache/redis/key.go#L105)；[Exists@96](../../../plugins/cache/redis/key.go#L96)；[Get@34](../../../plugins/cache/redis/key.go#L34)；[Set@58](../../../plugins/cache/redis/key.go#L58)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`   | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V]`                 | [plugins/cache/memory/list.go:13](../../../plugins/cache/memory/list.go#L13)                                                   | SDK生产       | [Append@109](../../../plugins/cache/memory/list.go#L109)；[Delete@325](../../../plugins/cache/memory/list.go#L325)；[Index@182](../../../plugins/cache/memory/list.go#L182)；[Len@204](../../../plugins/cache/memory/list.go#L204)；[Pop@220](../../../plugins/cache/memory/list.go#L220)；[PopLast@251](../../../plugins/cache/memory/list.go#L251)；[Prepend@53](../../../plugins/cache/memory/list.go#L53)；[Range@154](../../../plugins/cache/memory/list.go#L154)；[Remove@280](../../../plugins/cache/memory/list.go#L280)                                                                                                                                                                                                                                                                                                                           |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`   | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V]`                  | [plugins/cache/redis/list.go:13](../../../plugins/cache/redis/list.go#L13)                                                     | SDK生产       | [Append@99](../../../plugins/cache/redis/list.go#L99)；[Delete@215](../../../plugins/cache/redis/list.go#L215)；[Index@150](../../../plugins/cache/redis/list.go#L150)；[Len@166](../../../plugins/cache/redis/list.go#L166)；[Pop@171](../../../plugins/cache/redis/list.go#L171)；[PopLast@187](../../../plugins/cache/redis/list.go#L187)；[Prepend@56](../../../plugins/cache/redis/list.go#L56)；[Range@133](../../../plugins/cache/redis/list.go#L133)；[Remove@206](../../../plugins/cache/redis/list.go#L206)                                                                                                                                                                                                                                                                                                                                      |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`    | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.SetCache[K, V]`                  | [plugins/cache/memory/set.go:14](../../../plugins/cache/memory/set.go#L14)                                                     | SDK生产       | [Add@48](../../../plugins/cache/memory/set.go#L48)；[Count@141](../../../plugins/cache/memory/set.go#L141)；[Delete@187](../../../plugins/cache/memory/set.go#L187)；[IsMember@122](../../../plugins/cache/memory/set.go#L122)；[Members@100](../../../plugins/cache/memory/set.go#L100)；[Remove@157](../../../plugins/cache/memory/set.go#L157)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`    | `*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.SetCache[K, V]`                   | [plugins/cache/redis/set.go:12](../../../plugins/cache/redis/set.go#L12)                                                       | SDK生产       | [Add@53](../../../plugins/cache/redis/set.go#L53)；[Count@113](../../../plugins/cache/redis/set.go#L113)；[Delete@130](../../../plugins/cache/redis/set.go#L130)；[IsMember@104](../../../plugins/cache/redis/set.go#L104)；[Members@87](../../../plugins/cache/redis/set.go#L87)；[Remove@118](../../../plugins/cache/redis/set.go#L118)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter` | `*github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/mysql/v2.adapter`               | [plugins/database/adapter/mysql/mysql.go:18](../../../plugins/database/adapter/mysql/mysql.go#L18)                             | SDK生产       | [Open@133](../../../plugins/database/adapter/mysql/mysql.go#L133)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter` | `*github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/oracle/v2.adapter`              | [plugins/database/adapter/oracle/oracle.go:18](../../../plugins/database/adapter/oracle/oracle.go#L18)                         | SDK生产       | [Open@133](../../../plugins/database/adapter/oracle/oracle.go#L133)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter` | `*github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/postgres/v2.adapter`            | [plugins/database/adapter/postgres/postgres.go:18](../../../plugins/database/adapter/postgres/postgres.go#L18)                 | SDK生产       | [Open@127](../../../plugins/database/adapter/postgres/postgres.go#L127)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter` | `*github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2.adapter`              | [plugins/database/adapter/sqlite/sqlite.go:20](../../../plugins/database/adapter/sqlite/sqlite.go#L20)                         | SDK生产       | [Open@79](../../../plugins/database/adapter/sqlite/sqlite.go#L79)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter` | `*github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlserver/v2.adapter`           | [plugins/database/adapter/sqlserver/sqlserver.go:20](../../../plugins/database/adapter/sqlserver/sqlserver.go#L20)             | SDK生产       | [Open@128](../../../plugins/database/adapter/sqlserver/sqlserver.go#L128)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter` | `*github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.dummyAdapter`                | [plugins/database/dialect/dialect_test.go:14](../../../plugins/database/dialect/dialect_test.go#L14)                           | 单测替身/断言 | [Open@22](../../../plugins/database/dialect/dialect_test.go#L22)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter` | `*github.com/bamboo-services/bamboo-base-go/plugins/database/v2_test.fakeAdapter`                    | [plugins/database/plugin_test.go:109](../../../plugins/database/plugin_test.go#L109)                                           | 单测替身/断言 | [Open@119](../../../plugins/database/plugin_test.go#L119)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |

### 逐符号引用位置

下面每个P编号与分类表一一对应；所有位置均列全，不截取top-N。匿名回调的外部注册/返回位置见定义行及分类表，不能把“无独立标识符引用”解释为死代码。变量/字段/诊断类型同样是引用账，不虚称为函数调用。

#### P0001

- 定义：[application/activator.go:31](../../../application/activator.go#L31) `componentQueueItem.cancelErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                   | 所在函数/声明            | 引用来源 |
| ---------------------------------------------------------------------- | ------------------------ | -------- |
| [application/activator.go:403](../../../application/activator.go#L403) | `cancelUndispatchedItem` | SDK生产  |
| [application/component.go:156](../../../application/component.go#L156) | `executePipeline`        | SDK生产  |
| [application/component.go:285](../../../application/component.go#L285) | `executePipeline`        | SDK生产  |
| [application/component.go:368](../../../application/component.go#L368) | `executePipeline`        | SDK生产  |
| [application/component.go:369](../../../application/component.go#L369) | `executePipeline`        | SDK生产  |

#### P0002

- 定义：[application/activator.go:54](../../../application/activator.go#L54) `parentState.closeCause`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                   | 所在函数/声明     | 引用来源 |
| ---------------------------------------------------------------------- | ----------------- | -------- |
| [application/activator.go:312](../../../application/activator.go#L312) | `closeParent`     | SDK生产  |
| [application/activator.go:382](../../../application/activator.go#L382) | `requestStopAll`  | SDK生产  |
| [application/activator.go:465](../../../application/activator.go#L465) | `onSlotReleased`  | SDK生产  |
| [application/component.go:139](../../../application/component.go#L139) | `executePipeline` | SDK生产  |
| [application/component.go:271](../../../application/component.go#L271) | `executePipeline` | SDK生产  |

#### P0003

- 定义：[application/activator.go:101](../../../application/activator.go#L101) `componentEngine.stopErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                   | 所在函数/声明     | 引用来源 |
| ---------------------------------------------------------------------- | ----------------- | -------- |
| [application/activator.go:377](../../../application/activator.go#L377) | `requestStopAll`  | SDK生产  |
| [application/activator.go:457](../../../application/activator.go#L457) | `onSlotReleased`  | SDK生产  |
| [application/component.go:131](../../../application/component.go#L131) | `executePipeline` | SDK生产  |
| [application/component.go:263](../../../application/component.go#L263) | `executePipeline` | SDK生产  |

#### P0004

- 定义：[application/activator.go:298](../../../application/activator.go#L298) `componentEngine.closeParent`；分类N；阶段P1/P3。
- 完整当前类型：`func(instanceID string, cause error)`。

| 位置                                                                                           | 所在函数/声明                                            | 引用来源 |
| ---------------------------------------------------------------------------------------------- | -------------------------------------------------------- | -------- |
| [application/app.go:930](../../../application/app.go#L930)                                     | `launchWorkers`                                          | SDK生产  |
| [application/component.go:357](../../../application/component.go#L357)                         | `executePipeline`                                        | SDK生产  |
| [application/component_engine_test.go:414](../../../application/component_engine_test.go#L414) | `TestComponentEngine_UndispatchedCancelledOnParentClose` | 单测     |
| [application/component_engine_test.go:580](../../../application/component_engine_test.go#L580) | `TestComponentEngine_IdentityAndStateDefense`            | 单测     |

#### P0005

- 定义：[application/activator.go:343](../../../application/activator.go#L343) `componentEngine.waitParent`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context, parentID string) error`。

| 位置                                                                   | 所在函数/声明     | 引用来源 |
| ---------------------------------------------------------------------- | ----------------- | -------- |
| [application/app.go:931](../../../application/app.go#L931)             | `launchWorkers`   | SDK生产  |
| [application/component.go:358](../../../application/component.go#L358) | `executePipeline` | SDK生产  |

#### P0006

- 定义：[application/activator.go:364](../../../application/activator.go#L364) `componentEngine.requestStopAll`；分类N；阶段P1/P3。
- 完整当前类型：`func(cause error)`。

| 位置                                                         | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------ | ------------- | -------- |
| [application/app.go:1029](../../../application/app.go#L1029) | `Shutdown`    | SDK生产  |

#### P0007

- 定义：[application/activator.go:396](../../../application/activator.go#L396) `componentEngine.cancelUndispatchedItem`；分类N；阶段P1/P3。
- 完整当前类型：`func(it *github.com/bamboo-services/bamboo-base-go/application/v2.componentQueueItem, cause error)`。

| 位置                                                                   | 所在函数/声明    | 引用来源 |
| ---------------------------------------------------------------------- | ---------------- | -------- |
| [application/activator.go:333](../../../application/activator.go#L333) | `closeParent`    | SDK生产  |
| [application/activator.go:392](../../../application/activator.go#L392) | `requestStopAll` | SDK生产  |
| [application/activator.go:508](../../../application/activator.go#L508) | `onSlotReleased` | SDK生产  |

#### P0008

- 定义：[application/activator.go:435](../../../application/activator.go#L435) `onSlotReleased.rejectCauses`；分类S；阶段P1/P3。
- 完整当前类型：`[]error`。

| 位置                                                                   | 所在函数/声明    | 引用来源 |
| ---------------------------------------------------------------------- | ---------------- | -------- |
| [application/activator.go:479](../../../application/activator.go#L479) | `onSlotReleased` | SDK生产  |
| [application/activator.go:479](../../../application/activator.go#L479) | `onSlotReleased` | SDK生产  |
| [application/activator.go:508](../../../application/activator.go#L508) | `onSlotReleased` | SDK生产  |

#### P0009

- 定义：[application/activator.go:565](../../../application/activator.go#L565) `componentEngine.Activate`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context, compKey github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey.Key, opts ...github.com/bamboo-services/bamboo-base-go/application/v2.ActivateOption) (github.com/bamboo-services/bamboo-base-go/application/v2.RunHandle, error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Activator`。

| 位置                                                                                                     | 所在函数/声明                                                       | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- | -------- |
| [application/component_engine_test.go:128](../../../application/component_engine_test.go#L128)           | `TestComponentEngine_Skip_BarrierStopsUntilStopReturned`            | 单测     |
| [application/component_engine_test.go:136](../../../application/component_engine_test.go#L136)           | `TestComponentEngine_Skip_BarrierStopsUntilStopReturned`            | 单测     |
| [application/component_engine_test.go:146](../../../application/component_engine_test.go#L146)           | `TestComponentEngine_Skip_BarrierStopsUntilStopReturned`            | 单测     |
| [application/component_engine_test.go:159](../../../application/component_engine_test.go#L159)           | `TestComponentEngine_Skip_BarrierStopsUntilStopReturned`            | 单测     |
| [application/component_engine_test.go:210](../../../application/component_engine_test.go#L210)           | `TestComponentEngine_Allow_ConcurrencyLimit`                        | 单测     |
| [application/component_engine_test.go:211](../../../application/component_engine_test.go#L211)           | `TestComponentEngine_Allow_ConcurrencyLimit`                        | 单测     |
| [application/component_engine_test.go:220](../../../application/component_engine_test.go#L220)           | `TestComponentEngine_Allow_ConcurrencyLimit`                        | 单测     |
| [application/component_engine_test.go:230](../../../application/component_engine_test.go#L230)           | `TestComponentEngine_Allow_ConcurrencyLimit`                        | 单测     |
| [application/component_engine_test.go:286](../../../application/component_engine_test.go#L286)           | `TestComponentEngine_Queue_SharedFIFO_MaxQueue_ContextIndependence` | 单测     |
| [application/component_engine_test.go:297](../../../application/component_engine_test.go#L297)           | `TestComponentEngine_Queue_SharedFIFO_MaxQueue_ContextIndependence` | 单测     |
| [application/component_engine_test.go:310](../../../application/component_engine_test.go#L310)           | `TestComponentEngine_Queue_SharedFIFO_MaxQueue_ContextIndependence` | 单测     |
| [application/component_engine_test.go:316](../../../application/component_engine_test.go#L316)           | `TestComponentEngine_Queue_SharedFIFO_MaxQueue_ContextIndependence` | 单测     |
| [application/component_engine_test.go:395](../../../application/component_engine_test.go#L395)           | `TestComponentEngine_UndispatchedCancelledOnParentClose`            | 单测     |
| [application/component_engine_test.go:402](../../../application/component_engine_test.go#L402)           | `TestComponentEngine_UndispatchedCancelledOnParentClose`            | 单测     |
| [application/component_engine_test.go:501](../../../application/component_engine_test.go#L501)           | `TestComponentEngine_NestedQueue_WouldBlock_And_Cycle`              | 单测     |
| [application/component_engine_test.go:507](../../../application/component_engine_test.go#L507)           | `TestComponentEngine_NestedQueue_WouldBlock_And_Cycle`              | 单测     |
| [application/component_engine_test.go:521](../../../application/component_engine_test.go#L521)           | `TestComponentEngine_NestedQueue_WouldBlock_And_Cycle`              | 单测     |
| [application/component_engine_test.go:528](../../../application/component_engine_test.go#L528)           | `TestComponentEngine_NestedQueue_WouldBlock_And_Cycle`              | 单测     |
| [application/component_engine_test.go:560](../../../application/component_engine_test.go#L560)           | `TestComponentEngine_IdentityAndStateDefense`                       | 单测     |
| [application/component_engine_test.go:571](../../../application/component_engine_test.go#L571)           | `TestComponentEngine_IdentityAndStateDefense`                       | 单测     |
| [application/component_engine_test.go:583](../../../application/component_engine_test.go#L583)           | `TestComponentEngine_IdentityAndStateDefense`                       | 单测     |
| [application/component_engine_test.go:598](../../../application/component_engine_test.go#L598)           | `TestComponentEngine_IdentityAndStateDefense`                       | 单测     |
| [application/component_engine_test.go:630](../../../application/component_engine_test.go#L630)           | `TestComponentEngine_StopPanic_RetainsDepsAndTriggersAppShutdown`   | 单测     |
| [application/component_engine_test.go:690](../../../application/component_engine_test.go#L690)           | `TestComponentEngine_Wait_ReturnsImpliesSlotReleased`               | 单测     |
| [application/component_engine_test.go:697](../../../application/component_engine_test.go#L697)           | `TestComponentEngine_Wait_ReturnsImpliesSlotReleased`               | 单测     |
| [application/component_engine_test.go:706](../../../application/component_engine_test.go#L706)           | `TestComponentEngine_Wait_ReturnsImpliesSlotReleased`               | 单测     |
| [application/component_engine_test.go:725](../../../application/component_engine_test.go#L725)           | `TestComponentEngine_Wait_ReturnsImpliesSlotReleased`               | 单测     |
| [application/kernel_shutdown_races_test.go:298](../../../application/kernel_shutdown_races_test.go#L298) | `TestF3_AppStoppingCancelsQueueImmediately`                         | 单测     |
| [application/kernel_shutdown_races_test.go:307](../../../application/kernel_shutdown_races_test.go#L307) | `TestF3_AppStoppingCancelsQueueImmediately`                         | 单测     |

#### P0010

- 定义：[application/app.go:39](../../../application/app.go#L39) `staticInstanceRecord.startErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                       | 所在函数/声明       | 引用来源 |
| ---------------------------------------------------------- | ------------------- | -------- |
| [application/app.go:546](../../../application/app.go#L546) | `startStaticLayers` | SDK生产  |

#### P0011

- 定义：[application/app.go:43](../../../application/app.go#L43) `staticInstanceRecord.runErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                       | 所在函数/声明   | 引用来源 |
| ---------------------------------------------------------- | --------------- | -------- |
| [application/app.go:935](../../../application/app.go#L935) | `launchWorkers` | SDK生产  |

#### P0012

- 定义：[application/app.go:46](../../../application/app.go#L46) `staticInstanceRecord.quiesceErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                         | 所在函数/声明              | 引用来源 |
| ------------------------------------------------------------ | -------------------------- | -------- |
| [application/app.go:1169](../../../application/app.go#L1169) | `stopStaticInstanceRecord` | SDK生产  |

#### P0013

- 定义：[application/app.go:49](../../../application/app.go#L49) `staticInstanceRecord.stopErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                       | 所在函数/声明  | 引用来源 |
| ---------------------------------------------------------- | -------------- | -------- |
| [application/app.go:692](../../../application/app.go#L692) | `safeCallStop` | SDK生产  |
| [application/app.go:754](../../../application/app.go#L754) | `safeCallStop` | SDK生产  |

#### P0014

- 定义：[application/app.go:67](../../../application/app.go#L67) `staticInstanceRecord.stopPanicErr`；分类S；阶段P1/P3。
- 完整当前类型：`*github.com/bamboo-services/bamboo-base-go/application/v2.HookPanicError`。

| 位置                                                       | 所在函数/声明  | 引用来源 |
| ---------------------------------------------------------- | -------------- | -------- |
| [application/app.go:689](../../../application/app.go#L689) | `safeCallStop` | SDK生产  |
| [application/app.go:690](../../../application/app.go#L690) | `safeCallStop` | SDK生产  |
| [application/app.go:712](../../../application/app.go#L712) | `safeCallStop` | SDK生产  |

#### P0015

- 定义：[application/app.go:87](../../../application/app.go#L87) `App.cfgErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                         | 所在函数/声明                                | 引用来源 |
| -------------------------------------------------------------------------------------------- | -------------------------------------------- | -------- |
| [application/app.go:144](../../../application/app.go#L144)                                   | `New`                                        | SDK生产  |
| [application/app.go:341](../../../application/app.go#L341)                                   | `Run`                                        | SDK生产  |
| [application/app.go:345](../../../application/app.go#L345)                                   | `Run`                                        | SDK生产  |
| [application/app.go:1003](../../../application/app.go#L1003)                                 | `Shutdown`                                   | SDK生产  |
| [application/app.go:1033](../../../application/app.go#L1033)                                 | `Shutdown`                                   | SDK生产  |
| [application/app.go:1037](../../../application/app.go#L1037)                                 | `Shutdown`                                   | SDK生产  |
| [application/app.go:1039](../../../application/app.go#L1039)                                 | `Shutdown`                                   | SDK生产  |
| [application/app_snapshot_test.go:57](../../../application/app_snapshot_test.go#L57)         | `TestSnapshot_RunConsumedRepeatAndClosed`    | 单测     |
| [application/shutdown_cfgerr_test.go:19](../../../application/shutdown_cfgerr_test.go#L19)   | `TestShutdown_ConfigErr_BoundedChannelRepro` | 单测     |
| [application/shutdown_cfgerr_test.go:31](../../../application/shutdown_cfgerr_test.go#L31)   | `TestShutdown_ConfigErr_BoundedChannelRepro` | 单测     |
| [application/shutdown_cfgerr_test.go:32](../../../application/shutdown_cfgerr_test.go#L32)   | `TestShutdown_ConfigErr_BoundedChannelRepro` | 单测     |
| [application/shutdown_cfgerr_test.go:46](../../../application/shutdown_cfgerr_test.go#L46)   | `TestShutdown_ConfigErr_BoundedChannelRepro` | 单测     |
| [application/shutdown_cfgerr_test.go:67](../../../application/shutdown_cfgerr_test.go#L67)   | `TestShutdown_ConfigErr_BoundedChannelRepro` | 单测     |
| [application/shutdown_cfgerr_test.go:86](../../../application/shutdown_cfgerr_test.go#L86)   | `TestShutdown_ConfigErr_AfterRunRepro`       | 单测     |
| [application/shutdown_cfgerr_test.go:91](../../../application/shutdown_cfgerr_test.go#L91)   | `TestShutdown_ConfigErr_AfterRunRepro`       | 单测     |
| [application/shutdown_cfgerr_test.go:102](../../../application/shutdown_cfgerr_test.go#L102) | `TestShutdown_ConfigErr_AfterRunRepro`       | 单测     |
| [application/shutdown_cfgerr_test.go:110](../../../application/shutdown_cfgerr_test.go#L110) | `TestShutdown_ConfigErr_AfterRunRepro`       | 单测     |

#### P0016

- 定义：[application/app.go:113](../../../application/app.go#L113) `App.runErrs`；分类S；阶段P1/P3。
- 完整当前类型：`[]error`。

| 位置                                                       | 所在函数/声明  | 引用来源 |
| ---------------------------------------------------------- | -------------- | -------- |
| [application/app.go:445](../../../application/app.go#L445) | `recordRunErr` | SDK生产  |
| [application/app.go:445](../../../application/app.go#L445) | `recordRunErr` | SDK生产  |
| [application/app.go:451](../../../application/app.go#L451) | `getRunErr`    | SDK生产  |

#### P0017

- 定义：[application/app.go:191](../../../application/app.go#L191) `App.RegisterTool`；分类S；阶段P1/P3。
- 完整当前类型：`func(info github.com/bamboo-services/bamboo-base-go/application/v2.PluginInfo, factory func() github.com/bamboo-services/bamboo-base-go/application/v2.Plugin) error`。

| 位置                                                                                                                           | 所在函数/声明                                                                     | 引用来源   |
| ------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------- | ---------- |
| [application/app_snapshot_test.go:74](../../../application/app_snapshot_test.go#L74)                                           | `TestSnapshot_RunConsumedRepeatAndClosed`                                         | 单测       |
| [application/app_test.go:66](../../../application/app_test.go#L66)                                                             | `TestApp_DualAppIsolation`                                                        | 单测       |
| [application/app_test.go:71](../../../application/app_test.go#L71)                                                             | `TestApp_DualAppIsolation`                                                        | 单测       |
| [application/app_test.go:123](../../../application/app_test.go#L123)                                                           | `TestApp_Run_OnceAndAlreadyRun`                                                   | 单测       |
| [application/app_test.go:160](../../../application/app_test.go#L160)                                                           | `TestApp_StartFailureRollback`                                                    | 单测       |
| [application/app_test.go:171](../../../application/app_test.go#L171)                                                           | `TestApp_StartFailureRollback`                                                    | 单测       |
| [application/app_test.go:223](../../../application/app_test.go#L223)                                                           | `TestApp_ExposeValidationFailure`                                                 | 单测       |
| [application/app_test.go:269](../../../application/app_test.go#L269)                                                           | `TestApp_StopPanic_RetainsDependency`                                             | 单测       |
| [application/app_test.go:273](../../../application/app_test.go#L273)                                                           | `TestApp_StopPanic_RetainsDependency`                                             | 单测       |
| [application/app_test.go:401](../../../application/app_test.go#L401)                                                           | `TestApp_Context_RequiresStartScopeWhitelist`                                     | 单测       |
| [application/app_test.go:406](../../../application/app_test.go#L406)                                                           | `TestApp_Context_RequiresStartScopeWhitelist`                                     | 单测       |
| [application/builtin_lifecycle_test.go:126](../../../application/builtin_lifecycle_test.go#L126)                               | `setupApp`                                                                        | 单测       |
| [application/builtin_lifecycle_test.go:129](../../../application/builtin_lifecycle_test.go#L129)                               | `setupApp`                                                                        | 单测       |
| [application/builtin_lifecycle_test.go:311](../../../application/builtin_lifecycle_test.go#L311)                               | `TestAppLifecycle_IllegalSnowflakeNodeFailsRun`                                   | 单测       |
| [application/builtin_lifecycle_test.go:312](../../../application/builtin_lifecycle_test.go#L312)                               | `TestAppLifecycle_IllegalSnowflakeNodeFailsRun`                                   | 单测       |
| [application/dynamic_retention_test.go:109](../../../application/dynamic_retention_test.go#L109)                               | `runSubtestRegressionA`                                                           | 单测       |
| [application/dynamic_retention_test.go:281](../../../application/dynamic_retention_test.go#L281)                               | `TestDynamicRetention_RegressionB_StuckInRun_RetainsDependency`                   | 单测       |
| [application/dynamic_retention_test.go:465](../../../application/dynamic_retention_test.go#L465)                               | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`                      | 单测       |
| [application/dynamic_retention_test.go:659](../../../application/dynamic_retention_test.go#L659)                               | `TestDynamicRetention_RegressionC_StartTimeout_NoAppShutdown`                     | 单测       |
| [application/dynamic_retention_test.go:801](../../../application/dynamic_retention_test.go#L801)                               | `TestDynamicRetention_RegressionD_StartError_StopPanic_CoordinatesAppFailure`     | 单测       |
| [application/example_test.go:67](../../../application/example_test.go#L67)                                                     | `Example`                                                                         | 单测       |
| [application/kernel_real_exit_test.go:126](../../../application/kernel_real_exit_test.go#L126)                                 | `TestKernelStartTimeout_NoExternalShutdown_RealExit`                              | 单测       |
| [application/kernel_real_exit_test.go:136](../../../application/kernel_real_exit_test.go#L136)                                 | `TestKernelStartTimeout_NoExternalShutdown_RealExit`                              | 单测       |
| [application/kernel_real_exit_test.go:297](../../../application/kernel_real_exit_test.go#L297)                                 | `TestKernelRealExit_WorkerIgnoresCancel_StopReturnsNil_ShutdownIncomplete`        | 单测       |
| [application/kernel_shutdown_races_test.go:51](../../../application/kernel_shutdown_races_test.go#L51)                         | `TestF2_ShutdownDuringStartClaim`                                                 | 单测       |
| [application/quiesce_budget_test.go:97](../../../application/quiesce_budget_test.go#L97)                                       | `TestQuiesceBudget_RetainedDepsAndStuckInstance`                                  | 单测       |
| [application/quiesce_budget_test.go:270](../../../application/quiesce_budget_test.go#L270)                                     | `TestStartFailureCleanup_ErrorsJoinAndReverseCleanup`                             | 单测       |
| [application/quiesce_budget_test.go:280](../../../application/quiesce_budget_test.go#L280)                                     | `TestStartFailureCleanup_ErrorsJoinAndReverseCleanup`                             | 单测       |
| [application/retained_dependency_test.go:65](../../../application/retained_dependency_test.go#L65)                             | `TestApp_RetainedDependency_PropagationAndClosure`                                | 单测       |
| [application/retained_dependency_test.go:75](../../../application/retained_dependency_test.go#L75)                             | `TestApp_RetainedDependency_PropagationAndClosure`                                | 单测       |
| [application/retained_dependency_test.go:92](../../../application/retained_dependency_test.go#L92)                             | `TestApp_RetainedDependency_PropagationAndClosure`                                | 单测       |
| [application/retained_dependency_test.go:105](../../../application/retained_dependency_test.go#L105)                           | `TestApp_RetainedDependency_PropagationAndClosure`                                | 单测       |
| [application/stop_budget_test.go:100](../../../application/stop_budget_test.go#L100)                                           | `TestApp_StopBudget_TimeoutRetainsDepsAndUnblocks`                                | 单测       |
| [application/stop_budget_test.go:108](../../../application/stop_budget_test.go#L108)                                           | `TestApp_StopBudget_TimeoutRetainsDepsAndUnblocks`                                | 单测       |
| [application/typed_nil_test.go:86](../../../application/typed_nil_test.go#L86)                                                 | `TestTypedNil_FactoryReturnsTypedNil`                                             | 单测       |
| [application/typed_nil_test.go:119](../../../application/typed_nil_test.go#L119)                                               | `TestTypedNil_ExposeReturnsTypedNil`                                              | 单测       |
| [examples/v2/grpc-cron-async/main.go:64](../../../examples/v2/grpc-cron-async/main.go#L64)                                     | `main`                                                                            | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:34](../../../examples/v2/http-db-cache/main.go#L34)                                         | `main`                                                                            | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:40](../../../examples/v2/http-db-cache/main.go#L40)                                         | `main`                                                                            | 仓内消费者 |
| [examples/v2/worker-tool/main.go:150](../../../examples/v2/worker-tool/main.go#L150)                                           | `main`                                                                            | 仓内消费者 |
| [examples/v2/worker-tool/main.go:156](../../../examples/v2/worker-tool/main.go#L156)                                           | `main`                                                                            | 仓内消费者 |
| [examples/v2/worker-tool/main.go:162](../../../examples/v2/worker-tool/main.go#L162)                                           | `main`                                                                            | 仓内消费者 |
| [plugins/async/dependency_lifecycle_test.go:112](../../../plugins/async/dependency_lifecycle_test.go#L112)                     | `TestAsyncDependencyLifecycle_CaseA_AppForceCancelsTask_OrderlyShutdown`          | 单测       |
| [plugins/async/dependency_lifecycle_test.go:122](../../../plugins/async/dependency_lifecycle_test.go#L122)                     | `TestAsyncDependencyLifecycle_CaseA_AppForceCancelsTask_OrderlyShutdown`          | 单测       |
| [plugins/async/dependency_lifecycle_test.go:226](../../../plugins/async/dependency_lifecycle_test.go#L226)                     | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete` | 单测       |
| [plugins/async/dependency_lifecycle_test.go:236](../../../plugins/async/dependency_lifecycle_test.go#L236)                     | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete` | 单测       |
| [plugins/async/dependency_lifecycle_test.go:379](../../../plugins/async/dependency_lifecycle_test.go#L379)                     | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated`   | 单测       |
| [plugins/async/dependency_lifecycle_test.go:389](../../../plugins/async/dependency_lifecycle_test.go#L389)                     | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated`   | 单测       |
| [plugins/async/plugin_test.go:37](../../../plugins/async/plugin_test.go#L37)                                                   | `setupTestApp`                                                                    | 单测       |
| [plugins/cache/example_test.go:88](../../../plugins/cache/example_test.go#L88)                                                 | `ExampleNew`                                                                      | 单测       |
| [plugins/cache/example_test.go:108](../../../plugins/cache/example_test.go#L108)                                               | `ExampleWithRedis`                                                                | 单测       |
| [plugins/cache/plugin_test.go:132](../../../plugins/cache/plugin_test.go#L132)                                                 | `runWithCacheApp`                                                                 | 单测       |
| [plugins/database/adapter/mysql/integration_test.go:101](../../../plugins/database/adapter/mysql/integration_test.go#L101)     | `TestIntegration_MySQL`                                                           | 单测       |
| [plugins/database/adapter/mysql/integration_test.go:290](../../../plugins/database/adapter/mysql/integration_test.go#L290)     | `TestIntegration_MySQL_StartConnectionFailure`                                    | 单测       |
| [plugins/database/adapter/postgres/integration_test.go:62](../../../plugins/database/adapter/postgres/integration_test.go#L62) | `TestIntegration_PostgresReal`                                                    | 单测       |
| [plugins/database/models/models_test.go:91](../../../plugins/database/models/models_test.go#L91)                               | `runInTestApp`                                                                    | 单测       |
| [plugins/email/integration_smtp_test.go:113](../../../plugins/email/integration_smtp_test.go#L113)                             | `TestIntegration_SMTPSendAndLifecycle`                                            | 单测       |
| [plugins/email/integration_smtp_test.go:293](../../../plugins/email/integration_smtp_test.go#L293)                             | `TestIntegration_StopDrainsInflightPhysicalConn`                                  | 单测       |
| [plugins/grpc/plugin_app_context_test.go:390](../../../plugins/grpc/plugin_app_context_test.go#L390)                           | `createTestApp`                                                                   | 单测       |
| [plugins/grpc/plugin_app_context_test.go:786](../../../plugins/grpc/plugin_app_context_test.go#L786)                           | `TestPlugin_AppContext_StartACLUndeclared`                                        | 单测       |
| [plugins/grpc/plugin_app_context_test.go:793](../../../plugins/grpc/plugin_app_context_test.go#L793)                           | `TestPlugin_AppContext_StartACLUndeclared`                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:110](../../../plugins/web-gin/plugin_isolation_test.go#L110)                         | `buildApp`                                                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:448](../../../plugins/web-gin/plugin_isolation_test.go#L448)                         | `TestWebGin_UndeclaredDependency_RejectedInStart`                                 | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:154](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L154)         | `TestHTTPDBCacheLifecycle_EndToEnd`                                               | 仓内消费者 |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:157](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L157)         | `TestHTTPDBCacheLifecycle_EndToEnd`                                               | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:46](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L46)             | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`             | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:49](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L49)             | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`             | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:196](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L196)           | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`     | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:199](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L199)           | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`     | 仓内消费者 |

#### P0018

- 定义：[application/app.go:228](../../../application/app.go#L228) `App.RegisterWorker`；分类S；阶段P1/P3。
- 完整当前类型：`func(info github.com/bamboo-services/bamboo-base-go/application/v2.PluginInfo, factory func() github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin) error`。

| 位置                                                                                                                           | 所在函数/声明                                                                             | 引用来源   |
| ------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------- | ---------- |
| [application/activator_acceptance_test.go:116](../../../application/activator_acceptance_test.go#L116)                         | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned`                            | 单测       |
| [application/activator_acceptance_test.go:213](../../../application/activator_acceptance_test.go#L213)                         | `TestActivatorAcceptance_OverlapAllow_LimitExceeded`                                      | 单测       |
| [application/activator_acceptance_test.go:310](../../../application/activator_acceptance_test.go#L310)                         | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测       |
| [application/activator_acceptance_test.go:321](../../../application/activator_acceptance_test.go#L321)                         | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测       |
| [application/activator_acceptance_test.go:458](../../../application/activator_acceptance_test.go#L458)                         | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`                      | 单测       |
| [application/activator_acceptance_test.go:598](../../../application/activator_acceptance_test.go#L598)                         | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测       |
| [application/app_test.go:100](../../../application/app_test.go#L100)                                                           | `TestApp_Run_OnceAndAlreadyRun`                                                           | 单测       |
| [application/app_test.go:200](../../../application/app_test.go#L200)                                                           | `TestApp_WorkerUnexpectedExit`                                                            | 单测       |
| [application/app_test.go:312](../../../application/app_test.go#L312)                                                           | `TestApp_StopPanic_RetainsDependency`                                                     | 单测       |
| [application/builtin_lifecycle_test.go:132](../../../application/builtin_lifecycle_test.go#L132)                               | `setupApp`                                                                                | 单测       |
| [application/builtin_lifecycle_test.go:313](../../../application/builtin_lifecycle_test.go#L313)                               | `TestAppLifecycle_IllegalSnowflakeNodeFailsRun`                                           | 单测       |
| [application/dynamic_retention_test.go:160](../../../application/dynamic_retention_test.go#L160)                               | `runSubtestRegressionA`                                                                   | 单测       |
| [application/dynamic_retention_test.go:329](../../../application/dynamic_retention_test.go#L329)                               | `TestDynamicRetention_RegressionB_StuckInRun_RetainsDependency`                           | 单测       |
| [application/dynamic_retention_test.go:524](../../../application/dynamic_retention_test.go#L524)                               | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`                              | 单测       |
| [application/dynamic_retention_test.go:706](../../../application/dynamic_retention_test.go#L706)                               | `TestDynamicRetention_RegressionC_StartTimeout_NoAppShutdown`                             | 单测       |
| [application/dynamic_retention_test.go:838](../../../application/dynamic_retention_test.go#L838)                               | `TestDynamicRetention_RegressionD_StartError_StopPanic_CoordinatesAppFailure`             | 单测       |
| [application/example_test.go:74](../../../application/example_test.go#L74)                                                     | `Example`                                                                                 | 单测       |
| [application/kernel_real_exit_test.go:307](../../../application/kernel_real_exit_test.go#L307)                                 | `TestKernelRealExit_WorkerIgnoresCancel_StopReturnsNil_ShutdownIncomplete`                | 单测       |
| [application/kernel_real_exit_test.go:468](../../../application/kernel_real_exit_test.go#L468)                                 | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly`                     | 单测       |
| [application/kernel_shutdown_races_test.go:157](../../../application/kernel_shutdown_races_test.go#L157)                       | `TestF4_StopNilWhileRunBlocks`                                                            | 单测       |
| [application/kernel_shutdown_races_test.go:294](../../../application/kernel_shutdown_races_test.go#L294)                       | `TestF3_AppStoppingCancelsQueueImmediately`                                               | 单测       |
| [application/quiesce_budget_test.go:107](../../../application/quiesce_budget_test.go#L107)                                     | `TestQuiesceBudget_RetainedDepsAndStuckInstance`                                          | 单测       |
| [application/retained_dependency_test.go:117](../../../application/retained_dependency_test.go#L117)                           | `TestApp_RetainedDependency_PropagationAndClosure`                                        | 单测       |
| [application/stop_budget_test.go:116](../../../application/stop_budget_test.go#L116)                                           | `TestApp_StopBudget_TimeoutRetainsDepsAndUnblocks`                                        | 单测       |
| [examples/v2/grpc-cron-async/main.go:84](../../../examples/v2/grpc-cron-async/main.go#L84)                                     | `main`                                                                                    | 仓内消费者 |
| [examples/v2/grpc-cron-async/main.go:96](../../../examples/v2/grpc-cron-async/main.go#L96)                                     | `main`                                                                                    | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:107](../../../examples/v2/http-db-cache/main.go#L107)                                       | `main`                                                                                    | 仓内消费者 |
| [examples/v2/worker-tool/main.go:168](../../../examples/v2/worker-tool/main.go#L168)                                           | `main`                                                                                    | 仓内消费者 |
| [plugins/async/dependency_lifecycle_test.go:140](../../../plugins/async/dependency_lifecycle_test.go#L140)                     | `TestAsyncDependencyLifecycle_CaseA_AppForceCancelsTask_OrderlyShutdown`                  | 单测       |
| [plugins/async/dependency_lifecycle_test.go:260](../../../plugins/async/dependency_lifecycle_test.go#L260)                     | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete`         | 单测       |
| [plugins/async/dependency_lifecycle_test.go:408](../../../plugins/async/dependency_lifecycle_test.go#L408)                     | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated`           | 单测       |
| [plugins/async/plugin_test.go:52](../../../plugins/async/plugin_test.go#L52)                                                   | `setupTestApp`                                                                            | 单测       |
| [plugins/cache/plugin_test.go:145](../../../plugins/cache/plugin_test.go#L145)                                                 | `runWithCacheApp`                                                                         | 单测       |
| [plugins/cron/real_app_test.go:158](../../../plugins/cron/real_app_test.go#L158)                                               | `TestRealApp_CronSkip_RealComponentLifecycle`                                             | 单测       |
| [plugins/cron/real_queue_test.go:190](../../../plugins/cron/real_queue_test.go#L190)                                           | `TestCron_RealApp_QueueFIFOAndFullBehavior`                                               | 单测       |
| [plugins/database/adapter/mysql/integration_test.go:223](../../../plugins/database/adapter/mysql/integration_test.go#L223)     | `TestIntegration_MySQL`                                                                   | 单测       |
| [plugins/database/adapter/mysql/integration_test.go:304](../../../plugins/database/adapter/mysql/integration_test.go#L304)     | `TestIntegration_MySQL_StartConnectionFailure`                                            | 单测       |
| [plugins/database/adapter/postgres/integration_test.go:76](../../../plugins/database/adapter/postgres/integration_test.go#L76) | `TestIntegration_PostgresReal`                                                            | 单测       |
| [plugins/database/models/models_test.go:103](../../../plugins/database/models/models_test.go#L103)                             | `runInTestApp`                                                                            | 单测       |
| [plugins/email/integration_smtp_test.go:127](../../../plugins/email/integration_smtp_test.go#L127)                             | `TestIntegration_SMTPSendAndLifecycle`                                                    | 单测       |
| [plugins/email/integration_smtp_test.go:307](../../../plugins/email/integration_smtp_test.go#L307)                             | `TestIntegration_StopDrainsInflightPhysicalConn`                                          | 单测       |
| [plugins/grpc/plugin_app_context_test.go:422](../../../plugins/grpc/plugin_app_context_test.go#L422)                           | `createTestApp`                                                                           | 单测       |
| [plugins/grpc/plugin_app_context_test.go:429](../../../plugins/grpc/plugin_app_context_test.go#L429)                           | `createTestApp`                                                                           | 单测       |
| [plugins/grpc/plugin_app_context_test.go:825](../../../plugins/grpc/plugin_app_context_test.go#L825)                           | `TestPlugin_AppContext_StartACLUndeclared`                                                | 单测       |
| [plugins/grpc/safety_regression_test.go:53](../../../plugins/grpc/safety_regression_test.go#L53)                               | `TestSafetyRegression_StartRegistrarPanic_ReleasesListener`                               | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:194](../../../plugins/web-gin/plugin_isolation_test.go#L194)                         | `buildApp`                                                                                | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:205](../../../plugins/web-gin/plugin_isolation_test.go#L205)                         | `buildApp`                                                                                | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:467](../../../plugins/web-gin/plugin_isolation_test.go#L467)                         | `TestWebGin_UndeclaredDependency_RejectedInStart`                                         | 单测       |
| [plugins/web-gin/plugin_test.go:120](../../../plugins/web-gin/plugin_test.go#L120)                                             | `TestWebGin_ContainerBridgeAndRequestKey`                                                 | 单测       |
| [plugins/web-gin/plugin_test.go:257](../../../plugins/web-gin/plugin_test.go#L257)                                             | `TestWebGin_FullNetworkLifecycle`                                                         | 单测       |
| [plugins/web-gin/plugin_test.go:518](../../../plugins/web-gin/plugin_test.go#L518)                                             | `TestWebGin_ForceContextCancellation`                                                     | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:160](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L160)         | `TestHTTPDBCacheLifecycle_EndToEnd`                                                       | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:97](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L97)             | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`                     | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:275](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L275)           | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`             | 仓内消费者 |

#### P0019

- 定义：[application/app.go:265](../../../application/app.go#L265) `App.RegisterComponent`；分类S；阶段P1/P3。
- 完整当前类型：`func(info github.com/bamboo-services/bamboo-base-go/application/v2.PluginInfo, factory func() github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin) error`。

| 位置                                                                                                     | 所在函数/声明                                                                             | 引用来源   |
| -------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ---------- |
| [application/activator_acceptance_test.go:94](../../../application/activator_acceptance_test.go#L94)     | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned`                            | 单测       |
| [application/activator_acceptance_test.go:195](../../../application/activator_acceptance_test.go#L195)   | `TestActivatorAcceptance_OverlapAllow_LimitExceeded`                                      | 单测       |
| [application/activator_acceptance_test.go:284](../../../application/activator_acceptance_test.go#L284)   | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测       |
| [application/activator_acceptance_test.go:429](../../../application/activator_acceptance_test.go#L429)   | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`                      | 单测       |
| [application/activator_acceptance_test.go:548](../../../application/activator_acceptance_test.go#L548)   | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测       |
| [application/activator_acceptance_test.go:569](../../../application/activator_acceptance_test.go#L569)   | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测       |
| [application/dynamic_retention_test.go:125](../../../application/dynamic_retention_test.go#L125)         | `runSubtestRegressionA`                                                                   | 单测       |
| [application/dynamic_retention_test.go:302](../../../application/dynamic_retention_test.go#L302)         | `TestDynamicRetention_RegressionB_StuckInRun_RetainsDependency`                           | 单测       |
| [application/dynamic_retention_test.go:489](../../../application/dynamic_retention_test.go#L489)         | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`                              | 单测       |
| [application/dynamic_retention_test.go:678](../../../application/dynamic_retention_test.go#L678)         | `TestDynamicRetention_RegressionC_StartTimeout_NoAppShutdown`                             | 单测       |
| [application/dynamic_retention_test.go:813](../../../application/dynamic_retention_test.go#L813)         | `TestDynamicRetention_RegressionD_StartError_StopPanic_CoordinatesAppFailure`             | 单测       |
| [application/kernel_real_exit_test.go:436](../../../application/kernel_real_exit_test.go#L436)           | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly`                     | 单测       |
| [application/kernel_shutdown_races_test.go:271](../../../application/kernel_shutdown_races_test.go#L271) | `TestF3_AppStoppingCancelsQueueImmediately`                                               | 单测       |
| [examples/v2/grpc-cron-async/main.go:74](../../../examples/v2/grpc-cron-async/main.go#L74)               | `main`                                                                                    | 仓内消费者 |
| [plugins/cron/real_app_test.go:108](../../../plugins/cron/real_app_test.go#L108)                         | `TestRealApp_CronSkip_RealComponentLifecycle`                                             | 单测       |
| [plugins/cron/real_queue_test.go:149](../../../plugins/cron/real_queue_test.go#L149)                     | `TestCron_RealApp_QueueFIFOAndFullBehavior`                                               | 单测       |
| [plugins/grpc/plugin_app_context_test.go:398](../../../plugins/grpc/plugin_app_context_test.go#L398)     | `createTestApp`                                                                           | 单测       |

#### P0020

- 定义：[application/app.go:320](../../../application/app.go#L320) `App.Run`；分类S；阶段P1/P3。
- 完整当前类型：`func() error`。

| 位置                                                                                                                           | 所在函数/声明                                                                             | 引用来源   |
| ------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------- | ---------- |
| [application/activator_acceptance_test.go:172](../../../application/activator_acceptance_test.go#L172)                         | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned`                            | 单测       |
| [application/activator_acceptance_test.go:250](../../../application/activator_acceptance_test.go#L250)                         | `TestActivatorAcceptance_OverlapAllow_LimitExceeded`                                      | 单测       |
| [application/activator_acceptance_test.go:334](../../../application/activator_acceptance_test.go#L334)                         | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测       |
| [application/activator_acceptance_test.go:490](../../../application/activator_acceptance_test.go#L490)                         | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`                      | 单测       |
| [application/activator_acceptance_test.go:632](../../../application/activator_acceptance_test.go#L632)                         | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测       |
| [application/app_snapshot_test.go:62](../../../application/app_snapshot_test.go#L62)                                           | `TestSnapshot_RunConsumedRepeatAndClosed`                                                 | 单测       |
| [application/app_snapshot_test.go:68](../../../application/app_snapshot_test.go#L68)                                           | `TestSnapshot_RunConsumedRepeatAndClosed`                                                 | 单测       |
| [application/app_snapshot_test.go:84](../../../application/app_snapshot_test.go#L84)                                           | `TestSnapshot_RunConsumedRepeatAndClosed`                                                 | 单测       |
| [application/app_test.go:112](../../../application/app_test.go#L112)                                                           | `TestApp_Run_OnceAndAlreadyRun`                                                           | 单测       |
| [application/app_test.go:117](../../../application/app_test.go#L117)                                                           | `TestApp_Run_OnceAndAlreadyRun`                                                           | 单测       |
| [application/app_test.go:145](../../../application/app_test.go#L145)                                                           | `TestApp_ShutdownBeforeRun_SealsApp`                                                      | 单测       |
| [application/app_test.go:183](../../../application/app_test.go#L183)                                                           | `TestApp_StartFailureRollback`                                                            | 单测       |
| [application/app_test.go:209](../../../application/app_test.go#L209)                                                           | `TestApp_WorkerUnexpectedExit`                                                            | 单测       |
| [application/app_test.go:236](../../../application/app_test.go#L236)                                                           | `TestApp_ExposeValidationFailure`                                                         | 单测       |
| [application/app_test.go:318](../../../application/app_test.go#L318)                                                           | `TestApp_StopPanic_RetainsDependency`                                                     | 单测       |
| [application/app_test.go:419](../../../application/app_test.go#L419)                                                           | `TestApp_Context_RequiresStartScopeWhitelist`                                             | 单测       |
| [application/builtin_lifecycle_test.go:160](../../../application/builtin_lifecycle_test.go#L160)                               | `TestAppLifecycle_BuiltinLoggingAndSnowflake`                                             | 单测       |
| [application/builtin_lifecycle_test.go:192](../../../application/builtin_lifecycle_test.go#L192)                               | `TestAppLifecycle_BuiltinLoggingAndSnowflake`                                             | 单测       |
| [application/builtin_lifecycle_test.go:326](../../../application/builtin_lifecycle_test.go#L326)                               | `TestAppLifecycle_IllegalSnowflakeNodeFailsRun`                                           | 单测       |
| [application/dynamic_retention_test.go:188](../../../application/dynamic_retention_test.go#L188)                               | `runSubtestRegressionA`                                                                   | 单测       |
| [application/dynamic_retention_test.go:357](../../../application/dynamic_retention_test.go#L357)                               | `TestDynamicRetention_RegressionB_StuckInRun_RetainsDependency`                           | 单测       |
| [application/dynamic_retention_test.go:551](../../../application/dynamic_retention_test.go#L551)                               | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`                              | 单测       |
| [application/dynamic_retention_test.go:730](../../../application/dynamic_retention_test.go#L730)                               | `TestDynamicRetention_RegressionC_StartTimeout_NoAppShutdown`                             | 单测       |
| [application/dynamic_retention_test.go:865](../../../application/dynamic_retention_test.go#L865)                               | `TestDynamicRetention_RegressionD_StartError_StopPanic_CoordinatesAppFailure`             | 单测       |
| [application/example_test.go:81](../../../application/example_test.go#L81)                                                     | `Example`                                                                                 | 单测       |
| [application/kernel_real_exit_test.go:148](../../../application/kernel_real_exit_test.go#L148)                                 | `TestKernelStartTimeout_NoExternalShutdown_RealExit`                                      | 单测       |
| [application/kernel_real_exit_test.go:319](../../../application/kernel_real_exit_test.go#L319)                                 | `TestKernelRealExit_WorkerIgnoresCancel_StopReturnsNil_ShutdownIncomplete`                | 单测       |
| [application/kernel_real_exit_test.go:508](../../../application/kernel_real_exit_test.go#L508)                                 | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly`                     | 单测       |
| [application/kernel_shutdown_races_test.go:60](../../../application/kernel_shutdown_races_test.go#L60)                         | `TestF2_ShutdownDuringStartClaim`                                                         | 单测       |
| [application/kernel_shutdown_races_test.go:165](../../../application/kernel_shutdown_races_test.go#L165)                       | `TestF4_StopNilWhileRunBlocks`                                                            | 单测       |
| [application/kernel_shutdown_races_test.go:343](../../../application/kernel_shutdown_races_test.go#L343)                       | `TestF3_AppStoppingCancelsQueueImmediately`                                               | 单测       |
| [application/quiesce_budget_test.go:119](../../../application/quiesce_budget_test.go#L119)                                     | `TestQuiesceBudget_RetainedDepsAndStuckInstance`                                          | 单测       |
| [application/quiesce_budget_test.go:290](../../../application/quiesce_budget_test.go#L290)                                     | `TestStartFailureCleanup_ErrorsJoinAndReverseCleanup`                                     | 单测       |
| [application/retained_dependency_test.go:130](../../../application/retained_dependency_test.go#L130)                           | `TestApp_RetainedDependency_PropagationAndClosure`                                        | 单测       |
| [application/shutdown_cfgerr_test.go:90](../../../application/shutdown_cfgerr_test.go#L90)                                     | `TestShutdown_ConfigErr_AfterRunRepro`                                                    | 单测       |
| [application/stop_budget_test.go:126](../../../application/stop_budget_test.go#L126)                                           | `TestApp_StopBudget_TimeoutRetainsDepsAndUnblocks`                                        | 单测       |
| [application/typed_nil_test.go:96](../../../application/typed_nil_test.go#L96)                                                 | `TestTypedNil_FactoryReturnsTypedNil`                                                     | 单测       |
| [application/typed_nil_test.go:134](../../../application/typed_nil_test.go#L134)                                               | `TestTypedNil_ExposeReturnsTypedNil`                                                      | 单测       |
| [examples/v2/grpc-cron-async/main.go:100](../../../examples/v2/grpc-cron-async/main.go#L100)                                   | `main`                                                                                    | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:112](../../../examples/v2/http-db-cache/main.go#L112)                                       | `main`                                                                                    | 仓内消费者 |
| [examples/v2/worker-tool/main.go:173](../../../examples/v2/worker-tool/main.go#L173)                                           | `main`                                                                                    | 仓内消费者 |
| [plugins/async/dependency_lifecycle_test.go:152](../../../plugins/async/dependency_lifecycle_test.go#L152)                     | `TestAsyncDependencyLifecycle_CaseA_AppForceCancelsTask_OrderlyShutdown`                  | 单测       |
| [plugins/async/dependency_lifecycle_test.go:271](../../../plugins/async/dependency_lifecycle_test.go#L271)                     | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete`         | 单测       |
| [plugins/async/dependency_lifecycle_test.go:419](../../../plugins/async/dependency_lifecycle_test.go#L419)                     | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated`           | 单测       |
| [plugins/async/plugin_test.go:58](../../../plugins/async/plugin_test.go#L58)                                                   | `setupTestApp`                                                                            | 单测       |
| [plugins/cache/plugin_test.go:151](../../../plugins/cache/plugin_test.go#L151)                                                 | `runWithCacheApp`                                                                         | 单测       |
| [plugins/cron/real_app_test.go:164](../../../plugins/cron/real_app_test.go#L164)                                               | `TestRealApp_CronSkip_RealComponentLifecycle`                                             | 单测       |
| [plugins/cron/real_queue_test.go:195](../../../plugins/cron/real_queue_test.go#L195)                                           | `TestCron_RealApp_QueueFIFOAndFullBehavior`                                               | 单测       |
| [plugins/database/adapter/mysql/integration_test.go:232](../../../plugins/database/adapter/mysql/integration_test.go#L232)     | `TestIntegration_MySQL`                                                                   | 单测       |
| [plugins/database/adapter/mysql/integration_test.go:311](../../../plugins/database/adapter/mysql/integration_test.go#L311)     | `TestIntegration_MySQL_StartConnectionFailure`                                            | 单测       |
| [plugins/database/adapter/postgres/integration_test.go:84](../../../plugins/database/adapter/postgres/integration_test.go#L84) | `TestIntegration_PostgresReal`                                                            | 单测       |
| [plugins/database/models/models_test.go:117](../../../plugins/database/models/models_test.go#L117)                             | `runInTestApp`                                                                            | 单测       |
| [plugins/email/integration_smtp_test.go:133](../../../plugins/email/integration_smtp_test.go#L133)                             | `TestIntegration_SMTPSendAndLifecycle`                                                    | 单测       |
| [plugins/email/integration_smtp_test.go:313](../../../plugins/email/integration_smtp_test.go#L313)                             | `TestIntegration_StopDrainsInflightPhysicalConn`                                          | 单测       |
| [plugins/grpc/plugin_app_context_test.go:439](../../../plugins/grpc/plugin_app_context_test.go#L439)                           | `createTestApp`                                                                           | 单测       |
| [plugins/grpc/plugin_app_context_test.go:843](../../../plugins/grpc/plugin_app_context_test.go#L843)                           | `TestPlugin_AppContext_StartACLUndeclared`                                                | 单测       |
| [plugins/grpc/safety_regression_test.go:65](../../../plugins/grpc/safety_regression_test.go#L65)                               | `TestSafetyRegression_StartRegistrarPanic_ReleasesListener`                               | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:270](../../../plugins/web-gin/plugin_isolation_test.go#L270)                         | `TestWebGin_TwoApps_FullIsolation`                                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:282](../../../plugins/web-gin/plugin_isolation_test.go#L282)                         | `TestWebGin_TwoApps_FullIsolation`                                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:474](../../../plugins/web-gin/plugin_isolation_test.go#L474)                         | `TestWebGin_UndeclaredDependency_RejectedInStart`                                         | 单测       |
| [plugins/web-gin/plugin_test.go:126](../../../plugins/web-gin/plugin_test.go#L126)                                             | `TestWebGin_ContainerBridgeAndRequestKey`                                                 | 单测       |
| [plugins/web-gin/plugin_test.go:263](../../../plugins/web-gin/plugin_test.go#L263)                                             | `TestWebGin_FullNetworkLifecycle`                                                         | 单测       |
| [plugins/web-gin/plugin_test.go:521](../../../plugins/web-gin/plugin_test.go#L521)                                             | `TestWebGin_ForceContextCancellation`                                                     | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:181](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L181)         | `TestHTTPDBCacheLifecycle_EndToEnd`                                                       | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:120](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L120)           | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`                     | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:287](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L287)           | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`             | 仓内消费者 |

#### P0021

- 定义：[application/app.go:417](../../../application/app.go#L417) `Run.watchdogErrChan`；分类S；阶段P1/P3。
- 完整当前类型：`chan error`。

| 位置                                                       | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------- | ------------- | -------- |
| [application/app.go:418](../../../application/app.go#L418) | `Run`         | SDK生产  |
| [application/app.go:422](../../../application/app.go#L422) | `Run`         | SDK生产  |

#### P0022

- 定义：[application/app.go:439](../../../application/app.go#L439) `App.recordRunErr`；分类S；阶段P1/P3。
- 完整当前类型：`func(err error)`。

| 位置                                                                                   | 所在函数/声明                        | 引用来源 |
| -------------------------------------------------------------------------------------- | ------------------------------------ | -------- |
| [application/app.go:425](../../../application/app.go#L425)                             | `Run`                                | SDK生产  |
| [application/app.go:954](../../../application/app.go#L954)                             | `launchWorkers`                      | SDK生产  |
| [application/app_snapshot_test.go:101](../../../application/app_snapshot_test.go#L101) | `TestSnapshot_ErrorLedgerConcurrent` | 单测     |
| [application/app_snapshot_test.go:105](../../../application/app_snapshot_test.go#L105) | `TestSnapshot_ErrorLedgerConcurrent` | 单测     |

#### P0023

- 定义：[application/app.go:448](../../../application/app.go#L448) `App.getRunErr`；分类S；阶段P1/P3。
- 完整当前类型：`func() error`。

| 位置                                                                                   | 所在函数/声明                        | 引用来源 |
| -------------------------------------------------------------------------------------- | ------------------------------------ | -------- |
| [application/app.go:428](../../../application/app.go#L428)                             | `Run`                                | SDK生产  |
| [application/app.go:431](../../../application/app.go#L431)                             | `Run`                                | SDK生产  |
| [application/app_snapshot_test.go:110](../../../application/app_snapshot_test.go#L110) | `TestSnapshot_ErrorLedgerConcurrent` | 单测     |

#### P0024

- 定义：[application/app.go:454](../../../application/app.go#L454) `App.startStaticLayers`；分类S；阶段P1/P3。
- 完整当前类型：`func(cfg github.com/bamboo-services/bamboo-base-go/application/v2.Config, defMap map[github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey.Key]github.com/bamboo-services/bamboo-base-go/application/v2.pluginRegistration) (retErr error)`。

| 位置                                                       | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------- | ------------- | -------- |
| [application/app.go:388](../../../application/app.go#L388) | `Run`         | SDK生产  |

#### P0025

- 定义：[application/app.go:541](../../../application/app.go#L541) `startStaticLayers.startResCh`；分类S；阶段P1/P3。
- 完整当前类型：`chan error`。

| 位置                                                       | 所在函数/声明       | 引用来源 |
| ---------------------------------------------------------- | ------------------- | -------- |
| [application/app.go:551](../../../application/app.go#L551) | `startStaticLayers` | SDK生产  |
| [application/app.go:557](../../../application/app.go#L557) | `startStaticLayers` | SDK生产  |
| [application/app.go:561](../../../application/app.go#L561) | `startStaticLayers` | SDK生产  |

#### P0026

- 定义：[application/app.go:614](../../../application/app.go#L614) `App.createPluginInstance`；分类S；阶段P1/P3。
- 完整当前类型：`func(def github.com/bamboo-services/bamboo-base-go/application/v2.pluginRegistration) (github.com/bamboo-services/bamboo-base-go/application/v2.Plugin, string, error)`。

| 位置                                                       | 所在函数/声明       | 引用来源 |
| ---------------------------------------------------------- | ------------------- | -------- |
| [application/app.go:472](../../../application/app.go#L472) | `startStaticLayers` | SDK生产  |

#### P0027

- 定义：[application/app.go:665](../../../application/app.go#L665) `App.safeCallStart`；分类S；阶段P1/P3。
- 完整当前类型：`func(rec *github.com/bamboo-services/bamboo-base-go/application/v2.staticInstanceRecord, ctx context.Context) (retErr error)`。

| 位置                                                       | 所在函数/声明       | 引用来源 |
| ---------------------------------------------------------- | ------------------- | -------- |
| [application/app.go:544](../../../application/app.go#L544) | `startStaticLayers` | SDK生产  |

#### P0028

- 定义：[application/app.go:680](../../../application/app.go#L680) `App.safeCallStop`；分类S；阶段P1/P3。
- 完整当前类型：`func(rec *github.com/bamboo-services/bamboo-base-go/application/v2.staticInstanceRecord, cfg github.com/bamboo-services/bamboo-base-go/application/v2.Config, budget ...*github.com/bamboo-services/bamboo-base-go/application/v2.budgetMetadata) (retErr error)`。

| 位置                                                                                                     | 所在函数/声明                                         | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | ----------------------------------------------------- | -------- |
| [application/app.go:577](../../../application/app.go#L577)                                               | `startStaticLayers`                                   | SDK生产  |
| [application/app.go:590](../../../application/app.go#L590)                                               | `startStaticLayers`                                   | SDK生产  |
| [application/app.go:599](../../../application/app.go#L599)                                               | `startStaticLayers`                                   | SDK生产  |
| [application/app.go:606](../../../application/app.go#L606)                                               | `startStaticLayers`                                   | SDK生产  |
| [application/app.go:1196](../../../application/app.go#L1196)                                             | `stopStaticInstanceRecord`                            | SDK生产  |
| [application/app.go:1305](../../../application/app.go#L1305)                                             | `executeShutdownTopology`                             | SDK生产  |
| [application/kernel_shutdown_races_test.go:221](../../../application/kernel_shutdown_races_test.go#L221) | `TestF4_StopWinsBeforeRun`                            | 单测     |
| [application/stop_completion_test.go:72](../../../application/stop_completion_test.go#L72)               | `TestStopCompletion_ConcurrentWaitAndSingleExecution` | 单测     |
| [application/stop_completion_test.go:77](../../../application/stop_completion_test.go#L77)               | `TestStopCompletion_ConcurrentWaitAndSingleExecution` | 单测     |
| [application/stop_completion_test.go:127](../../../application/stop_completion_test.go#L127)             | `TestStopCompletion_PanicHandlingAndSecondCall`       | 单测     |
| [application/stop_completion_test.go:151](../../../application/stop_completion_test.go#L151)             | `TestStopCompletion_PanicHandlingAndSecondCall`       | 单测     |

#### P0029

- 定义：[application/app.go:776](../../../application/app.go#L776) `App.publishExposedKeys`；分类S；阶段P1/P3。
- 完整当前类型：`func(info github.com/bamboo-services/bamboo-base-go/application/v2.PluginInfo, inst github.com/bamboo-services/bamboo-base-go/application/v2.Plugin) error`。

| 位置                                                                                     | 所在函数/声明                                         | 引用来源 |
| ---------------------------------------------------------------------------------------- | ----------------------------------------------------- | -------- |
| [application/app.go:604](../../../application/app.go#L604)                               | `startStaticLayers`                                   | SDK生产  |
| [application/expose_atomic_test.go:124](../../../application/expose_atomic_test.go#L124) | `TestExposeAtomic_ValidationFailuresNoPartialPublish` | 单测     |
| [application/expose_atomic_test.go:144](../../../application/expose_atomic_test.go#L144) | `TestExposeAtomic_SuccessAllReadable`                 | 单测     |
| [application/expose_atomic_test.go:167](../../../application/expose_atomic_test.go#L167) | `TestExposeAtomic_DuplicateKeyRejection`              | 单测     |
| [application/expose_atomic_test.go:173](../../../application/expose_atomic_test.go#L173) | `TestExposeAtomic_DuplicateKeyRejection`              | 单测     |
| [application/expose_atomic_test.go:192](../../../application/expose_atomic_test.go#L192) | `TestExposeAtomic_FreezeRejection`                    | 单测     |
| [application/expose_atomic_test.go:217](../../../application/expose_atomic_test.go#L217) | `TestExposeAtomic_UserCodeDoesNotHoldLock`            | 单测     |

#### P0030

- 定义：[application/app.go:851](../../../application/app.go#L851) `App.launchWorkers`；分类S；阶段P1/P3。
- 完整当前类型：`func(watchdogErrChan chan error)`。

| 位置                                                                                                     | 所在函数/声明              | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | -------------------------- | -------- |
| [application/app.go:418](../../../application/app.go#L418)                                               | `Run`                      | SDK生产  |
| [application/kernel_shutdown_races_test.go:235](../../../application/kernel_shutdown_races_test.go#L235) | `TestF4_StopWinsBeforeRun` | 单测     |

#### P0031

- 定义：[application/app.go:986](../../../application/app.go#L986) `App.Shutdown`；分类S；阶段P1/P3。
- 完整当前类型：`func() error`。

| 位置                                                                                                                             | 所在函数/声明                                                                             | 引用来源   |
| -------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ---------- |
| [application/activator_acceptance_test.go:76](../../../application/activator_acceptance_test.go#L76)                             | `requireActivator`                                                                        | 单测       |
| [application/activator_acceptance_test.go:181](../../../application/activator_acceptance_test.go#L181)                           | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned`                            | 单测       |
| [application/activator_acceptance_test.go:259](../../../application/activator_acceptance_test.go#L259)                           | `TestActivatorAcceptance_OverlapAllow_LimitExceeded`                                      | 单测       |
| [application/activator_acceptance_test.go:410](../../../application/activator_acceptance_test.go#L410)                           | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测       |
| [application/activator_acceptance_test.go:505](../../../application/activator_acceptance_test.go#L505)                           | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`                      | 单测       |
| [application/activator_acceptance_test.go:641](../../../application/activator_acceptance_test.go#L641)                           | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测       |
| [application/app.go:382](../../../application/app.go#L382)                                                                       | `Run`                                                                                     | SDK生产    |
| [application/app.go:391](../../../application/app.go#L391)                                                                       | `Run`                                                                                     | SDK生产    |
| [application/app.go:408](../../../application/app.go#L408)                                                                       | `Run`                                                                                     | SDK生产    |
| [application/app.go:427](../../../application/app.go#L427)                                                                       | `Run`                                                                                     | SDK生产    |
| [application/app_snapshot_test.go:81](../../../application/app_snapshot_test.go#L81)                                             | `TestSnapshot_RunConsumedRepeatAndClosed`                                                 | 单测       |
| [application/app_test.go:128](../../../application/app_test.go#L128)                                                             | `TestApp_Run_OnceAndAlreadyRun`                                                           | 单测       |
| [application/app_test.go:138](../../../application/app_test.go#L138)                                                             | `TestApp_ShutdownBeforeRun_SealsApp`                                                      | 单测       |
| [application/app_test.go:329](../../../application/app_test.go#L329)                                                             | `TestApp_StopPanic_RetainsDependency`                                                     | 单测       |
| [application/builtin_lifecycle_test.go:152](../../../application/builtin_lifecycle_test.go#L152)                                 | `TestAppLifecycle_BuiltinLoggingAndSnowflake`                                             | 单测       |
| [application/builtin_lifecycle_test.go:184](../../../application/builtin_lifecycle_test.go#L184)                                 | `TestAppLifecycle_BuiltinLoggingAndSnowflake`                                             | 单测       |
| [application/builtin_lifecycle_test.go:218](../../../application/builtin_lifecycle_test.go#L218)                                 | `TestAppLifecycle_BuiltinLoggingAndSnowflake`                                             | 单测       |
| [application/builtin_lifecycle_test.go:221](../../../application/builtin_lifecycle_test.go#L221)                                 | `TestAppLifecycle_BuiltinLoggingAndSnowflake`                                             | 单测       |
| [application/builtin_lifecycle_test.go:256](../../../application/builtin_lifecycle_test.go#L256)                                 | `TestAppLifecycle_BuiltinLoggingAndSnowflake`                                             | 单测       |
| [application/builtin_lifecycle_test.go:259](../../../application/builtin_lifecycle_test.go#L259)                                 | `TestAppLifecycle_BuiltinLoggingAndSnowflake`                                             | 单测       |
| [application/builtin_lifecycle_test.go:318](../../../application/builtin_lifecycle_test.go#L318)                                 | `TestAppLifecycle_IllegalSnowflakeNodeFailsRun`                                           | 单测       |
| [application/component.go:478](../../../application/component.go#L478)                                                           | `handleStopPanic`                                                                         | SDK生产    |
| [application/dynamic_retention_test.go:206](../../../application/dynamic_retention_test.go#L206)                                 | `runSubtestRegressionA`                                                                   | 单测       |
| [application/dynamic_retention_test.go:373](../../../application/dynamic_retention_test.go#L373)                                 | `TestDynamicRetention_RegressionB_StuckInRun_RetainsDependency`                           | 单测       |
| [application/dynamic_retention_test.go:567](../../../application/dynamic_retention_test.go#L567)                                 | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`                              | 单测       |
| [application/dynamic_retention_test.go:765](../../../application/dynamic_retention_test.go#L765)                                 | `TestDynamicRetention_RegressionC_StartTimeout_NoAppShutdown`                             | 单测       |
| [application/example_test.go:47](../../../application/example_test.go#L47)                                                       | `Run`                                                                                     | 单测       |
| [application/kernel_real_exit_test.go:334](../../../application/kernel_real_exit_test.go#L334)                                   | `TestKernelRealExit_WorkerIgnoresCancel_StopReturnsNil_ShutdownIncomplete`                | 单测       |
| [application/kernel_real_exit_test.go:368](../../../application/kernel_real_exit_test.go#L368)                                   | `TestKernelRealExit_WorkerIgnoresCancel_StopReturnsNil_ShutdownIncomplete`                | 单测       |
| [application/kernel_real_exit_test.go:530](../../../application/kernel_real_exit_test.go#L530)                                   | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly`                     | 单测       |
| [application/kernel_shutdown_races_test.go:73](../../../application/kernel_shutdown_races_test.go#L73)                           | `TestF2_ShutdownDuringStartClaim`                                                         | 单测       |
| [application/kernel_shutdown_races_test.go:177](../../../application/kernel_shutdown_races_test.go#L177)                         | `TestF4_StopNilWhileRunBlocks`                                                            | 单测       |
| [application/kernel_shutdown_races_test.go:314](../../../application/kernel_shutdown_races_test.go#L314)                         | `TestF3_AppStoppingCancelsQueueImmediately`                                               | 单测       |
| [application/quiesce_budget_test.go:131](../../../application/quiesce_budget_test.go#L131)                                       | `TestQuiesceBudget_RetainedDepsAndStuckInstance`                                          | 单测       |
| [application/quiesce_budget_test.go:182](../../../application/quiesce_budget_test.go#L182)                                       | `TestQuiesceBudget_RetainedDepsAndStuckInstance`                                          | 单测       |
| [application/retained_dependency_test.go:140](../../../application/retained_dependency_test.go#L140)                             | `TestApp_RetainedDependency_PropagationAndClosure`                                        | 单测       |
| [application/retained_dependency_test.go:146](../../../application/retained_dependency_test.go#L146)                             | `TestApp_RetainedDependency_PropagationAndClosure`                                        | 单测       |
| [application/shutdown_cfgerr_test.go:26](../../../application/shutdown_cfgerr_test.go#L26)                                       | `TestShutdown_ConfigErr_BoundedChannelRepro`                                              | 单测       |
| [application/shutdown_cfgerr_test.go:41](../../../application/shutdown_cfgerr_test.go#L41)                                       | `TestShutdown_ConfigErr_BoundedChannelRepro`                                              | 单测       |
| [application/shutdown_cfgerr_test.go:62](../../../application/shutdown_cfgerr_test.go#L62)                                       | `TestShutdown_ConfigErr_BoundedChannelRepro`                                              | 单测       |
| [application/shutdown_cfgerr_test.go:97](../../../application/shutdown_cfgerr_test.go#L97)                                       | `TestShutdown_ConfigErr_AfterRunRepro`                                                    | 单测       |
| [application/shutdown_cfgerr_test.go:110](../../../application/shutdown_cfgerr_test.go#L110)                                     | `TestShutdown_ConfigErr_AfterRunRepro`                                                    | 单测       |
| [application/stop_budget_test.go:133](../../../application/stop_budget_test.go#L133)                                             | `TestApp_StopBudget_TimeoutRetainsDepsAndUnblocks`                                        | 单测       |
| [application/stop_budget_test.go:163](../../../application/stop_budget_test.go#L163)                                             | `TestApp_StopBudget_TimeoutRetainsDepsAndUnblocks`                                        | 单测       |
| [plugins/async/dependency_lifecycle_test.go:109](../../../plugins/async/dependency_lifecycle_test.go#L109)                       | `TestAsyncDependencyLifecycle_CaseA_AppForceCancelsTask_OrderlyShutdown`                  | 单测       |
| [plugins/async/dependency_lifecycle_test.go:161](../../../plugins/async/dependency_lifecycle_test.go#L161)                       | `TestAsyncDependencyLifecycle_CaseA_AppForceCancelsTask_OrderlyShutdown`                  | 单测       |
| [plugins/async/dependency_lifecycle_test.go:222](../../../plugins/async/dependency_lifecycle_test.go#L222)                       | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete`         | 单测       |
| [plugins/async/dependency_lifecycle_test.go:296](../../../plugins/async/dependency_lifecycle_test.go#L296)                       | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete`         | 单测       |
| [plugins/async/dependency_lifecycle_test.go:329](../../../plugins/async/dependency_lifecycle_test.go#L329)                       | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete`         | 单测       |
| [plugins/async/dependency_lifecycle_test.go:375](../../../plugins/async/dependency_lifecycle_test.go#L375)                       | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated`           | 单测       |
| [plugins/async/plugin_test.go:103](../../../plugins/async/plugin_test.go#L103)                                                   | `TestPlugin_BasicExecution`                                                               | 单测       |
| [plugins/async/plugin_test.go:135](../../../plugins/async/plugin_test.go#L135)                                                   | `TestPlugin_ContextDetachedFromParentCancel`                                              | 单测       |
| [plugins/async/plugin_test.go:170](../../../plugins/async/plugin_test.go#L170)                                                   | `TestPlugin_RequestKeyCopied`                                                             | 单测       |
| [plugins/async/plugin_test.go:200](../../../plugins/async/plugin_test.go#L200)                                                   | `TestPlugin_TaskCancel`                                                                   | 单测       |
| [plugins/async/plugin_test.go:224](../../../plugins/async/plugin_test.go#L224)                                                   | `TestPlugin_WaitWithTimeoutDoesNotCancelTask`                                             | 单测       |
| [plugins/async/plugin_test.go:263](../../../plugins/async/plugin_test.go#L263)                                                   | `TestPlugin_PanicRecovery`                                                                | 单测       |
| [plugins/async/plugin_test.go:295](../../../plugins/async/plugin_test.go#L295)                                                   | `TestPlugin_PoolFullRejection`                                                            | 单测       |
| [plugins/async/plugin_test.go:353](../../../plugins/async/plugin_test.go#L353)                                                   | `TestPlugin_RejectDuringStoppingAndClosed`                                                | 单测       |
| [plugins/async/plugin_test.go:385](../../../plugins/async/plugin_test.go#L385)                                                   | `TestPlugin_ForceCancelReceivedByTask`                                                    | 单测       |
| [plugins/async/plugin_test.go:459](../../../plugins/async/plugin_test.go#L459)                                                   | `TestPlugin_IgnoreCancelTaskMakesStopWait`                                                | 单测       |
| [plugins/async/plugin_test.go:465](../../../plugins/async/plugin_test.go#L465)                                                   | `TestPlugin_SubmitRollbackOnFailure`                                                      | 单测       |
| [plugins/async/plugin_test.go:506](../../../plugins/async/plugin_test.go#L506)                                                   | `TestPlugin_CrossAppAndBackgroundRejection`                                               | 单测       |
| [plugins/async/plugin_test.go:511](../../../plugins/async/plugin_test.go#L511)                                                   | `TestPlugin_CrossAppAndBackgroundRejection`                                               | 单测       |
| [plugins/cache/plugin_test.go:162](../../../plugins/cache/plugin_test.go#L162)                                                   | `runWithCacheApp`                                                                         | 单测       |
| [plugins/cron/real_app_test.go:171](../../../plugins/cron/real_app_test.go#L171)                                                 | `TestRealApp_CronSkip_RealComponentLifecycle`                                             | 单测       |
| [plugins/cron/real_queue_test.go:123](../../../plugins/cron/real_queue_test.go#L123)                                             | `TestCron_RealApp_QueueFIFOAndFullBehavior`                                               | 单测       |
| [plugins/cron/real_queue_test.go:283](../../../plugins/cron/real_queue_test.go#L283)                                             | `TestCron_RealApp_QueueFIFOAndFullBehavior`                                               | 单测       |
| [plugins/database/adapter/mysql/integration_test.go:89](../../../plugins/database/adapter/mysql/integration_test.go#L89)         | `TestIntegration_MySQL`                                                                   | 单测       |
| [plugins/database/adapter/mysql/integration_test.go:246](../../../plugins/database/adapter/mysql/integration_test.go#L246)       | `TestIntegration_MySQL`                                                                   | 单测       |
| [plugins/database/adapter/postgres/integration_test.go:198](../../../plugins/database/adapter/postgres/integration_test.go#L198) | `TestIntegration_PostgresReal`                                                            | 单测       |
| [plugins/database/models/models_test.go:121](../../../plugins/database/models/models_test.go#L121)                               | `runInTestApp`                                                                            | 单测       |
| [plugins/email/integration_smtp_test.go:247](../../../plugins/email/integration_smtp_test.go#L247)                               | `TestIntegration_SMTPSendAndLifecycle`                                                    | 单测       |
| [plugins/email/integration_smtp_test.go:343](../../../plugins/email/integration_smtp_test.go#L343)                               | `TestIntegration_StopDrainsInflightPhysicalConn`                                          | 单测       |
| [plugins/grpc/plugin_app_context_test.go:538](../../../plugins/grpc/plugin_app_context_test.go#L538)                             | `TestPlugin_AppContext_RealTCP_IsolationAndRegistry`                                      | 单测       |
| [plugins/grpc/plugin_app_context_test.go:547](../../../plugins/grpc/plugin_app_context_test.go#L547)                             | `TestPlugin_AppContext_RealTCP_IsolationAndRegistry`                                      | 单测       |
| [plugins/grpc/plugin_app_context_test.go:831](../../../plugins/grpc/plugin_app_context_test.go#L831)                             | `TestPlugin_AppContext_StartACLUndeclared`                                                | 单测       |
| [plugins/grpc/plugin_app_context_test.go:858](../../../plugins/grpc/plugin_app_context_test.go#L858)                             | `TestPlugin_AppContext_StartACLUndeclared`                                                | 单测       |
| [plugins/grpc/safety_regression_test.go:72](../../../plugins/grpc/safety_regression_test.go#L72)                                 | `TestSafetyRegression_StartRegistrarPanic_ReleasesListener`                               | 单测       |
| [plugins/grpc/safety_regression_test.go:107](../../../plugins/grpc/safety_regression_test.go#L107)                               | `TestSafetyRegression_StartRegistrarPanic_ReleasesListener`                               | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:273](../../../plugins/web-gin/plugin_isolation_test.go#L273)                           | `TestWebGin_TwoApps_FullIsolation`                                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:285](../../../plugins/web-gin/plugin_isolation_test.go#L285)                           | `TestWebGin_TwoApps_FullIsolation`                                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:407](../../../plugins/web-gin/plugin_isolation_test.go#L407)                           | `TestWebGin_TwoApps_FullIsolation`                                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:410](../../../plugins/web-gin/plugin_isolation_test.go#L410)                           | `TestWebGin_TwoApps_FullIsolation`                                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:478](../../../plugins/web-gin/plugin_isolation_test.go#L478)                           | `TestWebGin_UndeclaredDependency_RejectedInStart`                                         | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:503](../../../plugins/web-gin/plugin_isolation_test.go#L503)                           | `TestWebGin_UndeclaredDependency_RejectedInStart`                                         | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:169](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L169)           | `TestHTTPDBCacheLifecycle_EndToEnd`                                                       | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:110](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L110)             | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`                     | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:136](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L136)             | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`                     | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:161](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L161)             | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`                     | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:217](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L217)             | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`             | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:302](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L302)             | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`             | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:339](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L339)             | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`             | 仓内消费者 |

#### P0032

- 定义：[application/app.go:1120](../../../application/app.go#L1120) `App.stopStaticInstanceRecord`；分类S；阶段P1/P3。
- 完整当前类型：`func(rec *github.com/bamboo-services/bamboo-base-go/application/v2.staticInstanceRecord, cfg github.com/bamboo-services/bamboo-base-go/application/v2.Config, totalDeadline time.Time, graceDeadline time.Time, retainedProviders map[github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey.Key]struct{}) error`。

| 位置                                                         | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------ | ------------------------- | -------- |
| [application/app.go:1382](../../../application/app.go#L1382) | `executeShutdownTopology` | SDK生产  |

#### P0033

- 定义：[application/app.go:1140](../../../application/app.go#L1140) `stopStaticInstanceRecord.errs`；分类S；阶段P1/P3。
- 完整当前类型：`[]error`。

| 位置                                                         | 所在函数/声明              | 引用来源 |
| ------------------------------------------------------------ | -------------------------- | -------- |
| [application/app.go:1170](../../../application/app.go#L1170) | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1170](../../../application/app.go#L1170) | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1183](../../../application/app.go#L1183) | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1218](../../../application/app.go#L1218) | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1218](../../../application/app.go#L1218) | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1236](../../../application/app.go#L1236) | `stopStaticInstanceRecord` | SDK生产  |

#### P0034

- 定义：[application/app.go:1145](../../../application/app.go#L1145) `stopStaticInstanceRecord.quiesceDoneChan`；分类S；阶段P1/P3。
- 完整当前类型：`chan error`。

| 位置                                                         | 所在函数/声明              | 引用来源 |
| ------------------------------------------------------------ | -------------------------- | -------- |
| [application/app.go:1147](../../../application/app.go#L1147) | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1164](../../../application/app.go#L1164) | `stopStaticInstanceRecord` | SDK生产  |

#### P0035

- 定义：[application/app.go:1194](../../../application/app.go#L1194) `stopStaticInstanceRecord.stopDoneChan`；分类S；阶段P1/P3。
- 完整当前类型：`chan error`。

| 位置                                                         | 所在函数/声明              | 引用来源 |
| ------------------------------------------------------------ | -------------------------- | -------- |
| [application/app.go:1196](../../../application/app.go#L1196) | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1213](../../../application/app.go#L1213) | `stopStaticInstanceRecord` | SDK生产  |

#### P0036

- 定义：[application/app.go:1287](../../../application/app.go#L1287) `App.executeShutdownTopology`；分类S；阶段P1/P3。
- 完整当前类型：`func(cfg github.com/bamboo-services/bamboo-base-go/application/v2.Config, totalDeadline time.Time, graceDeadline time.Time) error`。

| 位置                                                         | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------ | ------------- | -------- |
| [application/app.go:1057](../../../application/app.go#L1057) | `Shutdown`    | SDK生产  |

#### P0037

- 定义：[application/app.go:1288](../../../application/app.go#L1288) `executeShutdownTopology.errs`；分类S；阶段P1/P3。
- 完整当前类型：`[]error`。

| 位置                                                         | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------ | ------------------------- | -------- |
| [application/app.go:1306](../../../application/app.go#L1306) | `executeShutdownTopology` | SDK生产  |
| [application/app.go:1306](../../../application/app.go#L1306) | `executeShutdownTopology` | SDK生产  |
| [application/app.go:1310](../../../application/app.go#L1310) | `executeShutdownTopology` | SDK生产  |
| [application/app.go:1383](../../../application/app.go#L1383) | `executeShutdownTopology` | SDK生产  |
| [application/app.go:1383](../../../application/app.go#L1383) | `executeShutdownTopology` | SDK生产  |
| [application/app.go:1456](../../../application/app.go#L1456) | `executeShutdownTopology` | SDK生产  |

#### P0038

- 定义：[application/app.go:1459](../../../application/app.go#L1459) `App.safeCallQuiesce`；分类S；阶段P1/P3。
- 完整当前类型：`func(rec *github.com/bamboo-services/bamboo-base-go/application/v2.staticInstanceRecord, q github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer, totalDeadline time.Time, graceDeadline time.Time, cleanupLimit time.Duration) (retErr error)`。

| 位置                                                         | 所在函数/声明              | 引用来源 |
| ------------------------------------------------------------ | -------------------------- | -------- |
| [application/app.go:1147](../../../application/app.go#L1147) | `stopStaticInstanceRecord` | SDK生产  |

#### P0039

- 定义：[application/builtin/logging.go:93](../../../application/builtin/logging.go#L93) `loggerPlugin.stopErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                               | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------- | ------------- | -------- |
| [application/builtin/logging.go:139](../../../application/builtin/logging.go#L139) | `Stop`        | SDK生产  |
| [application/builtin/logging.go:143](../../../application/builtin/logging.go#L143) | `Stop`        | SDK生产  |

#### P0040

- 定义：[application/builtin/logging.go:98](../../../application/builtin/logging.go#L98) `loggerPlugin.Start`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0041

- 定义：[application/builtin/logging.go:132](../../../application/builtin/logging.go#L132) `loggerPlugin.Stop`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0042

- 定义：[application/builtin/snowflake.go:77](../../../application/builtin/snowflake.go#L77) `snowflakePlugin.Start`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0043

- 定义：[application/builtin/snowflake.go:96](../../../application/builtin/snowflake.go#L96) `snowflakePlugin.Stop`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0044

- 定义：[application/component.go:29](../../../application/component.go#L29) `componentInstance.stopErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                   | 所在函数/声明           | 引用来源 |
| ---------------------------------------------------------------------- | ----------------------- | -------- |
| [application/component.go:418](../../../application/component.go#L418) | `safeCallComponentStop` | SDK生产  |
| [application/component.go:431](../../../application/component.go#L431) | `safeCallComponentStop` | SDK生产  |

#### P0045

- 定义：[application/component.go:396](../../../application/component.go#L396) `isContextCanceledError`；分类N；阶段P1/P3。
- 完整当前类型：`func(err error) bool`。

| 位置                                                                   | 所在函数/声明     | 引用来源 |
| ---------------------------------------------------------------------- | ----------------- | -------- |
| [application/component.go:384](../../../application/component.go#L384) | `executePipeline` | SDK生产  |

#### P0046

- 定义：[application/component.go:403](../../../application/component.go#L403) `componentEngine.safeCallComponentStop`；分类S；阶段P1/P3。
- 完整当前类型：`func(ci *github.com/bamboo-services/bamboo-base-go/application/v2.componentInstance, def github.com/bamboo-services/bamboo-base-go/application/v2.PluginInfo) (retErr error)`。

| 位置                                                                   | 所在函数/声明     | 引用来源 |
| ---------------------------------------------------------------------- | ----------------- | -------- |
| [application/component.go:236](../../../application/component.go#L236) | `executePipeline` | SDK生产  |
| [application/component.go:293](../../../application/component.go#L293) | `executePipeline` | SDK生产  |
| [application/component.go:362](../../../application/component.go#L362) | `executePipeline` | SDK生产  |

#### P0047

- 定义：[application/component.go:457](../../../application/component.go#L457) `componentEngine.handleStopPanic`；分类S；阶段P1/P3。
- 完整当前类型：`func(ci *github.com/bamboo-services/bamboo-base-go/application/v2.componentInstance, def github.com/bamboo-services/bamboo-base-go/application/v2.PluginInfo, stopErr error)`。

| 位置                                                                   | 所在函数/声明     | 引用来源 |
| ---------------------------------------------------------------------- | ----------------- | -------- |
| [application/component.go:242](../../../application/component.go#L242) | `executePipeline` | SDK生产  |
| [application/component.go:299](../../../application/component.go#L299) | `executePipeline` | SDK生产  |
| [application/component.go:379](../../../application/component.go#L379) | `executePipeline` | SDK生产  |

#### P0048

- 定义：[application/container.go:46](../../../application/container.go#L46) `sharedContainer.publishBatch`；分类S；阶段P1/P3。
- 完整当前类型：`func(items []github.com/bamboo-services/bamboo-base-go/application/v2.publishItem) error`。

| 位置                                                       | 所在函数/声明        | 引用来源 |
| ---------------------------------------------------------- | -------------------- | -------- |
| [application/app.go:848](../../../application/app.go#L848) | `publishExposedKeys` | SDK生产  |

#### P0049

- 定义：[application/container.go:122](../../../application/container.go#L122) `startScopeView.getEntry`；分类S；阶段P1/P3。
- 完整当前类型：`func(key github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey.Key) (*github.com/bamboo-services/bamboo-base-go/application/v2.containerEntry, error)`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [application/context.go:450](../../../application/context.go#L450) | `Get`         | SDK生产  |

#### P0050

- 定义：[application/context.go:190](../../../application/context.go#L190) `bridgedContext.Err`；分类F；阶段P1/P3。
- 完整当前类型：`func() error`。
- 方法集：`context.Context`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0051

- 定义：[application/context.go:404](../../../application/context.go#L404) `newAppErr`；分类A；阶段P1/P2。
- 完整当前类型：`func(code *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.ErrorCode, msg string, cause error) *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [application/context.go:443](../../../application/context.go#L443) | `Get`         | SDK生产  |
| [application/context.go:453](../../../application/context.go#L453) | `Get`         | SDK生产  |
| [application/context.go:456](../../../application/context.go#L456) | `Get`         | SDK生产  |
| [application/context.go:458](../../../application/context.go#L458) | `Get`         | SDK生产  |
| [application/context.go:467](../../../application/context.go#L467) | `Get`         | SDK生产  |
| [application/context.go:471](../../../application/context.go#L471) | `Get`         | SDK生产  |
| [application/context.go:485](../../../application/context.go#L485) | `Get`         | SDK生产  |
| [application/context.go:489](../../../application/context.go#L489) | `Get`         | SDK生产  |
| [application/context.go:500](../../../application/context.go#L500) | `castEntry`   | SDK生产  |
| [application/context.go:502](../../../application/context.go#L502) | `castEntry`   | SDK生产  |
| [application/context.go:506](../../../application/context.go#L506) | `castEntry`   | SDK生产  |
| [application/context.go:512](../../../application/context.go#L512) | `castEntry`   | SDK生产  |
| [application/context.go:520](../../../application/context.go#L520) | `castEntry`   | SDK生产  |

#### P0052

- 定义：[application/context.go:440](../../../application/context.go#L440) `Get`；分类A；阶段P1/P2。
- 完整当前类型：`func[T any](ctx context.Context, key github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey.Key) (T, *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error)`。

| 位置                                                                                                                 | 所在函数/声明                                                                     | 引用来源   |
| -------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | ---------- |
| [application/app_test.go:83](../../../application/app_test.go#L83)                                                   | `TestApp_DualAppIsolation`                                                        | 单测       |
| [application/app_test.go:88](../../../application/app_test.go#L88)                                                   | `TestApp_DualAppIsolation`                                                        | 单测       |
| [application/app_test.go:378](../../../application/app_test.go#L378)                                                 | `TestApp_StopPanic_RetainsDependency`                                             | 单测       |
| [application/app_test.go:383](../../../application/app_test.go#L383)                                                 | `TestApp_StopPanic_RetainsDependency`                                             | 单测       |
| [application/app_test.go:409](../../../application/app_test.go#L409)                                                 | `TestApp_Context_RequiresStartScopeWhitelist`                                     | 单测       |
| [application/app_test.go:455](../../../application/app_test.go#L455)                                                 | `TestApp_BridgeRequestAndDetach`                                                  | 单测       |
| [application/app_test.go:465](../../../application/app_test.go#L465)                                                 | `TestApp_BridgeRequestAndDetach`                                                  | 单测       |
| [application/builtin_lifecycle_test.go:39](../../../application/builtin_lifecycle_test.go#L39)                       | `Start`                                                                           | 单测       |
| [application/builtin_lifecycle_test.go:45](../../../application/builtin_lifecycle_test.go#L45)                       | `Start`                                                                           | 单测       |
| [application/context.go:533](../../../application/context.go#L533)                                                   | `MustGet`                                                                         | SDK生产    |
| [application/example_test.go:34](../../../application/example_test.go#L34)                                           | `Run`                                                                             | 单测       |
| [application/example_test.go:99](../../../application/example_test.go#L99)                                           | `ExampleGet`                                                                      | 单测       |
| [application/framework_error_code_test.go:46](../../../application/framework_error_code_test.go#L46)                 | `TestGetFrameworkErrorCodesAndCauses`                                             | 单测       |
| [examples/v2/grpc-cron-async/main.go:28](../../../examples/v2/grpc-cron-async/main.go#L28)                           | `Start`                                                                           | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:50](../../../examples/v2/http-db-cache/main.go#L50)                               | `main`                                                                            | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:55](../../../examples/v2/http-db-cache/main.go#L55)                               | `main`                                                                            | 仓内消费者 |
| [examples/v2/worker-tool/main.go:79](../../../examples/v2/worker-tool/main.go#L79)                                   | `Start`                                                                           | 仓内消费者 |
| [examples/v2/worker-tool/main.go:85](../../../examples/v2/worker-tool/main.go#L85)                                   | `Start`                                                                           | 仓内消费者 |
| [examples/v2/worker-tool/main.go:91](../../../examples/v2/worker-tool/main.go#L91)                                   | `Start`                                                                           | 仓内消费者 |
| [plugins/async/dependency_lifecycle_test.go:242](../../../plugins/async/dependency_lifecycle_test.go#L242)           | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete` | 单测       |
| [plugins/async/dependency_lifecycle_test.go:395](../../../plugins/async/dependency_lifecycle_test.go#L395)           | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated`   | 单测       |
| [plugins/async/plugin.go:333](../../../plugins/async/plugin.go#L333)                                                 | `Submit`                                                                          | SDK生产    |
| [plugins/async/plugin.go:421](../../../plugins/async/plugin.go#L421)                                                 | `Submit`                                                                          | SDK生产    |
| [plugins/async/plugin_test.go:388](../../../plugins/async/plugin_test.go#L388)                                       | `TestPlugin_ForceCancelReceivedByTask`                                            | 单测       |
| [plugins/async/plugin_test.go:416](../../../plugins/async/plugin_test.go#L416)                                       | `TestPlugin_IgnoreCancelTaskMakesStopWait`                                        | 单测       |
| [plugins/async/plugin_test.go:468](../../../plugins/async/plugin_test.go#L468)                                       | `TestPlugin_SubmitRollbackOnFailure`                                              | 单测       |
| [plugins/async/plugin_test.go:514](../../../plugins/async/plugin_test.go#L514)                                       | `TestPlugin_CrossAppAndBackgroundRejection`                                       | 单测       |
| [plugins/async/plugin_test.go:518](../../../plugins/async/plugin_test.go#L518)                                       | `TestPlugin_CrossAppAndBackgroundRejection`                                       | 单测       |
| [plugins/cache/config.go:244](../../../plugins/cache/config.go#L244)                                                 | `Get`                                                                             | SDK生产    |
| [plugins/cache/plugin_test.go:311](../../../plugins/cache/plugin_test.go#L311)                                       | `TestExpose_ExposesActualInstances`                                               | 单测       |
| [plugins/cache/plugin_test.go:317](../../../plugins/cache/plugin_test.go#L317)                                       | `TestExpose_ExposesActualInstances`                                               | 单测       |
| [plugins/database/context.go:18](../../../plugins/database/context.go#L18)                                           | `Get`                                                                             | SDK生产    |
| [plugins/database/models/base_entity.go:74](../../../plugins/database/models/base_entity.go#L74)                     | `generateEntityID`                                                                | SDK生产    |
| [plugins/email/integration_smtp_test.go:145](../../../plugins/email/integration_smtp_test.go#L145)                   | `TestIntegration_SMTPSendAndLifecycle`                                            | 单测       |
| [plugins/email/integration_smtp_test.go:318](../../../plugins/email/integration_smtp_test.go#L318)                   | `TestIntegration_StopDrainsInflightPhysicalConn`                                  | 单测       |
| [plugins/grpc/plugin_app_context_test.go:115](../../../plugins/grpc/plugin_app_context_test.go#L115)                 | `unaryContextCheckHandler`                                                        | 单测       |
| [plugins/grpc/plugin_app_context_test.go:221](../../../plugins/grpc/plugin_app_context_test.go#L221)                 | `streamContextCheckHandler`                                                       | 单测       |
| [plugins/grpc/plugin_app_context_test.go:413](../../../plugins/grpc/plugin_app_context_test.go#L413)                 | `createTestApp`                                                                   | 单测       |
| [plugins/grpc/plugin_app_context_test.go:806](../../../plugins/grpc/plugin_app_context_test.go#L806)                 | `TestPlugin_AppContext_StartACLUndeclared`                                        | 单测       |
| [plugins/grpc/plugin_app_context_test.go:812](../../../plugins/grpc/plugin_app_context_test.go#L812)                 | `TestPlugin_AppContext_StartACLUndeclared`                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:119](../../../plugins/web-gin/plugin_isolation_test.go#L119)               | `buildApp`                                                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:148](../../../plugins/web-gin/plugin_isolation_test.go#L148)               | `buildApp`                                                                        | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:304](../../../plugins/web-gin/plugin_isolation_test.go#L304)               | `TestWebGin_TwoApps_FullIsolation`                                                | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:311](../../../plugins/web-gin/plugin_isolation_test.go#L311)               | `TestWebGin_TwoApps_FullIsolation`                                                | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:457](../../../plugins/web-gin/plugin_isolation_test.go#L457)               | `TestWebGin_UndeclaredDependency_RejectedInStart`                                 | 单测       |
| [plugins/web-gin/plugin_test.go:95](../../../plugins/web-gin/plugin_test.go#L95)                                     | `TestWebGin_ContainerBridgeAndRequestKey`                                         | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:68](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L68) | `TestHTTPDBCacheLifecycle_EndToEnd`                                               | 仓内消费者 |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:80](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L80) | `TestHTTPDBCacheLifecycle_EndToEnd`                                               | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:61](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L61)   | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`             | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:71](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L71)   | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`             | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:238](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L238) | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`     | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:248](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L248) | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`     | 仓内消费者 |

#### P0053

- 定义：[application/context.go:495](../../../application/context.go#L495) `castEntry`；分类A；阶段P1/P2。
- 完整当前类型：`func[T any](key github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey.Key, entry *github.com/bamboo-services/bamboo-base-go/application/v2.containerEntry) (T, *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error)`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [application/context.go:460](../../../application/context.go#L460) | `Get`         | SDK生产  |
| [application/context.go:492](../../../application/context.go#L492) | `Get`         | SDK生产  |

#### P0054

- 定义：[application/errors.go:13](../../../application/errors.go#L13) `ErrAlreadyRun`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                 | 所在函数/声明                             | 引用来源 |
| ------------------------------------------------------------------------------------ | ----------------------------------------- | -------- |
| [application/app.go:325](../../../application/app.go#L325)                           | `Run`                                     | SDK生产  |
| [application/app_snapshot_test.go:69](../../../application/app_snapshot_test.go#L69) | `TestSnapshot_RunConsumedRepeatAndClosed` | 单测     |
| [application/app_test.go:118](../../../application/app_test.go#L118)                 | `TestApp_Run_OnceAndAlreadyRun`           | 单测     |

#### P0055

- 定义：[application/errors.go:14](../../../application/errors.go#L14) `ErrAppClosed`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                           | 所在函数/声明                                 | 引用来源 |
| ---------------------------------------------------------------------------------------------- | --------------------------------------------- | -------- |
| [application/activator.go:579](../../../application/activator.go#L579)                         | `Activate`                                    | SDK生产  |
| [application/activator.go:610](../../../application/activator.go#L610)                         | `Activate`                                    | SDK生产  |
| [application/app.go:329](../../../application/app.go#L329)                                     | `Run`                                         | SDK生产  |
| [application/app_snapshot_test.go:85](../../../application/app_snapshot_test.go#L85)           | `TestSnapshot_RunConsumedRepeatAndClosed`     | 单测     |
| [application/app_test.go:146](../../../application/app_test.go#L146)                           | `TestApp_ShutdownBeforeRun_SealsApp`          | 单测     |
| [application/component_engine_test.go:599](../../../application/component_engine_test.go#L599) | `TestComponentEngine_IdentityAndStateDefense` | 单测     |

#### P0056

- 定义：[application/errors.go:15](../../../application/errors.go#L15) `ErrRegistrationClosed`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                 | 所在函数/声明                             | 引用来源 |
| ------------------------------------------------------------------------------------ | ----------------------------------------- | -------- |
| [application/app.go:200](../../../application/app.go#L200)                           | `RegisterTool`                            | SDK生产  |
| [application/app.go:237](../../../application/app.go#L237)                           | `RegisterWorker`                          | SDK生产  |
| [application/app.go:274](../../../application/app.go#L274)                           | `RegisterComponent`                       | SDK生产  |
| [application/app_snapshot_test.go:75](../../../application/app_snapshot_test.go#L75) | `TestSnapshot_RunConsumedRepeatAndClosed` | 单测     |
| [application/app_test.go:124](../../../application/app_test.go#L124)                 | `TestApp_Run_OnceAndAlreadyRun`           | 单测     |
| [application/container.go:51](../../../application/container.go#L51)                 | `publishBatch`                            | SDK生产  |

#### P0057

- 定义：[application/errors.go:16](../../../application/errors.go#L16) `ErrStartupInterrupted`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                              | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------ | -------- |
| [application/app.go:397](../../../application/app.go#L397)                                             | `Run`                                      | SDK生产  |
| [application/app.go:398](../../../application/app.go#L398)                                             | `Run`                                      | SDK生产  |
| [application/app.go:409](../../../application/app.go#L409)                                             | `Run`                                      | SDK生产  |
| [application/app.go:468](../../../application/app.go#L468)                                             | `startStaticLayers`                        | SDK生产  |
| [application/app.go:512](../../../application/app.go#L512)                                             | `startStaticLayers`                        | SDK生产  |
| [application/app.go:569](../../../application/app.go#L569)                                             | `startStaticLayers`                        | SDK生产  |
| [application/app.go:592](../../../application/app.go#L592)                                             | `startStaticLayers`                        | SDK生产  |
| [application/app.go:600](../../../application/app.go#L600)                                             | `startStaticLayers`                        | SDK生产  |
| [application/kernel_shutdown_races_test.go:92](../../../application/kernel_shutdown_races_test.go#L92) | `TestF2_ShutdownDuringStartClaim`          | 单测     |
| [plugins/grpc/plugin_app_context_test.go:834](../../../plugins/grpc/plugin_app_context_test.go#L834)   | `TestPlugin_AppContext_StartACLUndeclared` | 单测     |
| [plugins/grpc/plugin_app_context_test.go:861](../../../plugins/grpc/plugin_app_context_test.go#L861)   | `TestPlugin_AppContext_StartACLUndeclared` | 单测     |

#### P0058

- 定义：[application/errors.go:17](../../../application/errors.go#L17) `ErrWorkerUnexpectedExit`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明                  | 引用来源 |
| -------------------------------------------------------------------- | ------------------------------ | -------- |
| [application/app.go:946](../../../application/app.go#L946)           | `launchWorkers`                | SDK生产  |
| [application/app.go:948](../../../application/app.go#L948)           | `launchWorkers`                | SDK生产  |
| [application/app_test.go:210](../../../application/app_test.go#L210) | `TestApp_WorkerUnexpectedExit` | 单测     |

#### P0059

- 定义：[application/errors.go:18](../../../application/errors.go#L18) `ErrNilFactory`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                       | 所在函数/声明       | 引用来源 |
| ---------------------------------------------------------- | ------------------- | -------- |
| [application/app.go:193](../../../application/app.go#L193) | `RegisterTool`      | SDK生产  |
| [application/app.go:230](../../../application/app.go#L230) | `RegisterWorker`    | SDK生产  |
| [application/app.go:267](../../../application/app.go#L267) | `RegisterComponent` | SDK生产  |

#### P0060

- 定义：[application/errors.go:19](../../../application/errors.go#L19) `ErrNilPlugin`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                             | 所在函数/声明                         | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [application/app.go:647](../../../application/app.go#L647)                       | `createPluginInstance`                | SDK生产  |
| [application/component.go:116](../../../application/component.go#L116)           | `executePipeline`                     | SDK生产  |
| [application/typed_nil_test.go:100](../../../application/typed_nil_test.go#L100) | `TestTypedNil_FactoryReturnsTypedNil` | 单测     |

#### P0061

- 定义：[application/errors.go:20](../../../application/errors.go#L20) `ErrInvalidActivationParent`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                           | 所在函数/声明                                 | 引用来源 |
| ---------------------------------------------------------------------------------------------- | --------------------------------------------- | -------- |
| [application/activator.go:473](../../../application/activator.go#L473)                         | `onSlotReleased`                              | SDK生产  |
| [application/activator.go:568](../../../application/activator.go#L568)                         | `Activate`                                    | SDK生产  |
| [application/activator.go:581](../../../application/activator.go#L581)                         | `Activate`                                    | SDK生产  |
| [application/activator.go:588](../../../application/activator.go#L588)                         | `Activate`                                    | SDK生产  |
| [application/activator.go:592](../../../application/activator.go#L592)                         | `Activate`                                    | SDK生产  |
| [application/activator.go:595](../../../application/activator.go#L595)                         | `Activate`                                    | SDK生产  |
| [application/activator.go:616](../../../application/activator.go#L616)                         | `Activate`                                    | SDK生产  |
| [application/activator.go:623](../../../application/activator.go#L623)                         | `Activate`                                    | SDK生产  |
| [application/component_engine_test.go:561](../../../application/component_engine_test.go#L561) | `TestComponentEngine_IdentityAndStateDefense` | 单测     |
| [application/component_engine_test.go:572](../../../application/component_engine_test.go#L572) | `TestComponentEngine_IdentityAndStateDefense` | 单测     |
| [application/component_engine_test.go:584](../../../application/component_engine_test.go#L584) | `TestComponentEngine_IdentityAndStateDefense` | 单测     |

#### P0062

- 定义：[application/errors.go:21](../../../application/errors.go#L21) `ErrComponentSkipped`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                                  | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------- | -------- |
| [application/activator.go:666](../../../application/activator.go#L666)                                 | `Activate`                                                     | SDK生产  |
| [application/activator_acceptance_test.go:135](../../../application/activator_acceptance_test.go#L135) | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned` | 单测     |
| [application/activator_acceptance_test.go:145](../../../application/activator_acceptance_test.go#L145) | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned` | 单测     |
| [application/component_engine_test.go:137](../../../application/component_engine_test.go#L137)         | `TestComponentEngine_Skip_BarrierStopsUntilStopReturned`       | 单测     |
| [application/component_engine_test.go:147](../../../application/component_engine_test.go#L147)         | `TestComponentEngine_Skip_BarrierStopsUntilStopReturned`       | 单测     |
| [plugins/cron/real_app_test.go:193](../../../plugins/cron/real_app_test.go#L193)                       | `TestRealApp_CronSkip_RealComponentLifecycle`                  | 单测     |

#### P0063

- 定义：[application/errors.go:22](../../../application/errors.go#L22) `ErrConcurrencyLimitExceeded`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                        | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | ---------------------------------------------------- | -------- |
| [application/activator.go:679](../../../application/activator.go#L679)                                 | `Activate`                                           | SDK生产  |
| [application/activator_acceptance_test.go:233](../../../application/activator_acceptance_test.go#L233) | `TestActivatorAcceptance_OverlapAllow_LimitExceeded` | 单测     |
| [application/component_engine_test.go:221](../../../application/component_engine_test.go#L221)         | `TestComponentEngine_Allow_ConcurrencyLimit`         | 单测     |

#### P0064

- 定义：[application/errors.go:23](../../../application/errors.go#L23) `ErrQueueFull`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                                                             | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------- | -------- |
| [application/activator.go:717](../../../application/activator.go#L717)                                 | `Activate`                                                                                | SDK生产  |
| [application/activator_acceptance_test.go:376](../../../application/activator_acceptance_test.go#L376) | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测     |
| [application/component_engine_test.go:317](../../../application/component_engine_test.go#L317)         | `TestComponentEngine_Queue_SharedFIFO_MaxQueue_ContextIndependence`                       | 单测     |
| [plugins/cron/real_queue_test.go:243](../../../plugins/cron/real_queue_test.go#L243)                   | `TestCron_RealApp_QueueFIFOAndFullBehavior`                                               | 单测     |

#### P0065

- 定义：[application/errors.go:24](../../../application/errors.go#L24) `ErrNestedQueueWouldBlock`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                                        | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------- | -------- |
| [application/activator.go:701](../../../application/activator.go#L701)                                 | `Activate`                                                           | SDK生产  |
| [application/activator_acceptance_test.go:588](../../../application/activator_acceptance_test.go#L588) | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle` | 单测     |
| [application/component_engine_test.go:508](../../../application/component_engine_test.go#L508)         | `TestComponentEngine_NestedQueue_WouldBlock_And_Cycle`               | 单测     |

#### P0066

- 定义：[application/errors.go:25](../../../application/errors.go#L25) `ErrActivationCycle`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                                        | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------- | -------- |
| [application/activator.go:630](../../../application/activator.go#L630)                                 | `Activate`                                                           | SDK生产  |
| [application/activator_acceptance_test.go:581](../../../application/activator_acceptance_test.go#L581) | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle` | 单测     |
| [application/component_engine_test.go:502](../../../application/component_engine_test.go#L502)         | `TestComponentEngine_NestedQueue_WouldBlock_And_Cycle`               | 单测     |

#### P0067

- 定义：[application/errors.go:26](../../../application/errors.go#L26) `ErrContainerNotMounted`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                 | 所在函数/声明                             | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ----------------------------------------- | -------- |
| [application/context.go:443](../../../application/context.go#L443)                                   | `Get`                                     | SDK生产  |
| [application/context.go:467](../../../application/context.go#L467)                                   | `Get`                                     | SDK生产  |
| [application/context.go:471](../../../application/context.go#L471)                                   | `Get`                                     | SDK生产  |
| [application/framework_error_code_test.go:30](../../../application/framework_error_code_test.go#L30) | `TestGetFrameworkErrorCodesAndCauses`     | 单测     |
| [application/framework_error_code_test.go:31](../../../application/framework_error_code_test.go#L31) | `TestGetFrameworkErrorCodesAndCauses`     | 单测     |
| [application/framework_error_code_test.go:32](../../../application/framework_error_code_test.go#L32) | `TestGetFrameworkErrorCodesAndCauses`     | 单测     |
| [application/framework_error_code_test.go:33](../../../application/framework_error_code_test.go#L33) | `TestGetFrameworkErrorCodesAndCauses`     | 单测     |
| [plugins/database/error_code_test.go:48](../../../plugins/database/error_code_test.go#L48)           | `TestGetPreservesApplicationError`        | 单测     |
| [plugins/web-gin/plugin_test.go:96](../../../plugins/web-gin/plugin_test.go#L96)                     | `TestWebGin_ContainerBridgeAndRequestKey` | 单测     |

#### P0068

- 定义：[application/errors.go:27](../../../application/errors.go#L27) `ErrUndeclaredDependency`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                     | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------- | -------- |
| [application/app_test.go:420](../../../application/app_test.go#L420)                                   | `TestApp_Context_RequiresStartScopeWhitelist`     | 单测     |
| [application/container.go:124](../../../application/container.go#L124)                                 | `getEntry`                                        | SDK生产  |
| [application/context.go:452](../../../application/context.go#L452)                                     | `Get`                                             | SDK生产  |
| [application/context.go:453](../../../application/context.go#L453)                                     | `Get`                                             | SDK生产  |
| [application/framework_error_code_test.go:34](../../../application/framework_error_code_test.go#L34)   | `TestGetFrameworkErrorCodesAndCauses`             | 单测     |
| [plugins/grpc/plugin_app_context_test.go:817](../../../plugins/grpc/plugin_app_context_test.go#L817)   | `TestPlugin_AppContext_StartACLUndeclared`        | 单测     |
| [plugins/web-gin/plugin_isolation_test.go:495](../../../plugins/web-gin/plugin_isolation_test.go#L495) | `TestWebGin_UndeclaredDependency_RejectedInStart` | 单测     |

#### P0069

- 定义：[application/errors.go:28](../../../application/errors.go#L28) `ErrComponentStopped`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                 | 所在函数/声明                         | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [application/context.go:500](../../../application/context.go#L500)                                   | `castEntry`                           | SDK生产  |
| [application/framework_error_code_test.go:40](../../../application/framework_error_code_test.go#L40) | `TestGetFrameworkErrorCodesAndCauses` | 单测     |
| [plugins/async/plugin.go:423](../../../plugins/async/plugin.go#L423)                                 | `Submit`                              | SDK生产  |

#### P0070

- 定义：[application/errors.go:29](../../../application/errors.go#L29) `ErrComponentTypeMismatch`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                 | 所在函数/声明                         | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [application/context.go:485](../../../application/context.go#L485)                                   | `Get`                                 | SDK生产  |
| [application/context.go:512](../../../application/context.go#L512)                                   | `castEntry`                           | SDK生产  |
| [application/context.go:520](../../../application/context.go#L520)                                   | `castEntry`                           | SDK生产  |
| [application/framework_error_code_test.go:37](../../../application/framework_error_code_test.go#L37) | `TestGetFrameworkErrorCodesAndCauses` | 单测     |
| [application/framework_error_code_test.go:38](../../../application/framework_error_code_test.go#L38) | `TestGetFrameworkErrorCodesAndCauses` | 单测     |
| [application/framework_error_code_test.go:39](../../../application/framework_error_code_test.go#L39) | `TestGetFrameworkErrorCodesAndCauses` | 单测     |

#### P0071

- 定义：[application/errors.go:30](../../../application/errors.go#L30) `ErrComponentNotFound`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                 | 所在函数/声明                         | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [application/activator.go:638](../../../application/activator.go#L638)                               | `Activate`                            | SDK生产  |
| [application/container.go:128](../../../application/container.go#L128)                               | `getEntry`                            | SDK生产  |
| [application/context.go:455](../../../application/context.go#L455)                                   | `Get`                                 | SDK生产  |
| [application/context.go:456](../../../application/context.go#L456)                                   | `Get`                                 | SDK生产  |
| [application/context.go:489](../../../application/context.go#L489)                                   | `Get`                                 | SDK生产  |
| [application/framework_error_code_test.go:35](../../../application/framework_error_code_test.go#L35) | `TestGetFrameworkErrorCodesAndCauses` | 单测     |
| [application/framework_error_code_test.go:36](../../../application/framework_error_code_test.go#L36) | `TestGetFrameworkErrorCodesAndCauses` | 单测     |

#### P0072

- 定义：[application/errors.go:31](../../../application/errors.go#L31) `ErrComponentUnavailable`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                 | 所在函数/声明                         | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [application/app_test.go:379](../../../application/app_test.go#L379)                                 | `TestApp_StopPanic_RetainsDependency` | 单测     |
| [application/app_test.go:390](../../../application/app_test.go#L390)                                 | `TestApp_StopPanic_RetainsDependency` | 单测     |
| [application/context.go:502](../../../application/context.go#L502)                                   | `castEntry`                           | SDK生产  |
| [application/context.go:506](../../../application/context.go#L506)                                   | `castEntry`                           | SDK生产  |
| [application/framework_error_code_test.go:41](../../../application/framework_error_code_test.go#L41) | `TestGetFrameworkErrorCodesAndCauses` | 单测     |
| [application/framework_error_code_test.go:42](../../../application/framework_error_code_test.go#L42) | `TestGetFrameworkErrorCodesAndCauses` | 单测     |
| [plugins/async/plugin.go:423](../../../plugins/async/plugin.go#L423)                                 | `Submit`                              | SDK生产  |

#### P0073

- 定义：[application/errors.go:32](../../../application/errors.go#L32) `ErrDuplicateExposedKey`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                     | 所在函数/声明                            | 引用来源 |
| ---------------------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [application/app.go:822](../../../application/app.go#L822)                               | `publishExposedKeys`                     | SDK生产  |
| [application/container.go:57](../../../application/container.go#L57)                     | `publishBatch`                           | SDK生产  |
| [application/container.go:65](../../../application/container.go#L65)                     | `publishBatch`                           | SDK生产  |
| [application/expose_atomic_test.go:174](../../../application/expose_atomic_test.go#L174) | `TestExposeAtomic_DuplicateKeyRejection` | 单测     |
| [application/graph.go:74](../../../application/graph.go#L74)                             | `buildGraph`                             | SDK生产  |
| [application/graph.go:92](../../../application/graph.go#L92)                             | `buildGraph`                             | SDK生产  |
| [application/graph_test.go:122](../../../application/graph_test.go#L122)                 | `TestBuildGraph_ValidationFailures`      | 单测     |

#### P0074

- 定义：[application/errors.go:33](../../../application/errors.go#L33) `ErrExposedKeysMismatch`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                     | 所在函数/声明                                         | 引用来源 |
| ---------------------------------------------------------------------------------------- | ----------------------------------------------------- | -------- |
| [application/app.go:780](../../../application/app.go#L780)                               | `publishExposedKeys`                                  | SDK生产  |
| [application/app.go:806](../../../application/app.go#L806)                               | `publishExposedKeys`                                  | SDK生产  |
| [application/app.go:809](../../../application/app.go#L809)                               | `publishExposedKeys`                                  | SDK生产  |
| [application/app.go:819](../../../application/app.go#L819)                               | `publishExposedKeys`                                  | SDK生产  |
| [application/app.go:825](../../../application/app.go#L825)                               | `publishExposedKeys`                                  | SDK生产  |
| [application/app_test.go:237](../../../application/app_test.go#L237)                     | `TestApp_ExposeValidationFailure`                     | 单测     |
| [application/expose_atomic_test.go:96](../../../application/expose_atomic_test.go#L96)   | `TestExposeAtomic_ValidationFailuresNoPartialPublish` | 单测     |
| [application/expose_atomic_test.go:102](../../../application/expose_atomic_test.go#L102) | `TestExposeAtomic_ValidationFailuresNoPartialPublish` | 单测     |
| [application/expose_atomic_test.go:108](../../../application/expose_atomic_test.go#L108) | `TestExposeAtomic_ValidationFailuresNoPartialPublish` | 单测     |
| [application/typed_nil_test.go:138](../../../application/typed_nil_test.go#L138)         | `TestTypedNil_ExposeReturnsTypedNil`                  | 单测     |

#### P0075

- 定义：[application/errors.go:34](../../../application/errors.go#L34) `ErrInvalidDependencyScope`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:125](../../../application/graph.go#L125)           | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:153](../../../application/graph_test.go#L153) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0076

- 定义：[application/errors.go:39](../../../application/errors.go#L39) `ErrParentClosing`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                     | 所在函数/声明                                            | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- | -------- |
| [application/activator.go:307](../../../application/activator.go#L307)                                   | `closeParent`                                            | SDK生产  |
| [application/activator.go:467](../../../application/activator.go#L467)                                   | `onSlotReleased`                                         | SDK生产  |
| [application/app.go:928](../../../application/app.go#L928)                                               | `launchWorkers`                                          | SDK生产  |
| [application/component.go:141](../../../application/component.go#L141)                                   | `executePipeline`                                        | SDK生产  |
| [application/component.go:273](../../../application/component.go#L273)                                   | `executePipeline`                                        | SDK生产  |
| [application/component.go:357](../../../application/component.go#L357)                                   | `executePipeline`                                        | SDK生产  |
| [application/component_engine_test.go:414](../../../application/component_engine_test.go#L414)           | `TestComponentEngine_UndispatchedCancelledOnParentClose` | 单测     |
| [application/component_engine_test.go:580](../../../application/component_engine_test.go#L580)           | `TestComponentEngine_IdentityAndStateDefense`            | 单测     |
| [application/kernel_shutdown_races_test.go:348](../../../application/kernel_shutdown_races_test.go#L348) | `TestF3_AppStoppingCancelsQueueImmediately`              | 单测     |
| [application/option_test.go:46](../../../application/option_test.go#L46)                                 | `TestErrorsAndSentinels`                                 | 单测     |
| [application/option_test.go:52](../../../application/option_test.go#L52)                                 | `TestErrorsAndSentinels`                                 | 单测     |
| [application/option_test.go:52](../../../application/option_test.go#L52)                                 | `TestErrorsAndSentinels`                                 | 单测     |

#### P0077

- 定义：[application/errors.go:40](../../../application/errors.go#L40) `ErrShuttingDown`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明            | 引用来源 |
| ------------------------------------------------------------------------ | ------------------------ | -------- |
| [application/activator.go:366](../../../application/activator.go#L366)   | `requestStopAll`         | SDK生产  |
| [application/activator.go:459](../../../application/activator.go#L459)   | `onSlotReleased`         | SDK生产  |
| [application/app.go:1029](../../../application/app.go#L1029)             | `Shutdown`               | SDK生产  |
| [application/component.go:133](../../../application/component.go#L133)   | `executePipeline`        | SDK生产  |
| [application/component.go:265](../../../application/component.go#L265)   | `executePipeline`        | SDK生产  |
| [application/option_test.go:49](../../../application/option_test.go#L49) | `TestErrorsAndSentinels` | 单测     |

#### P0078

- 定义：[application/errors.go:43](../../../application/errors.go#L43) `canceledSentinelError`；分类D；阶段P1/P3。
- 完整当前类型：`*github.com/bamboo-services/bamboo-base-go/application/v2.canceledSentinelError`。

| 位置                                                           | 所在函数/声明   | 引用来源 |
| -------------------------------------------------------------- | --------------- | -------- |
| [application/errors.go:39](../../../application/errors.go#L39) | `package-level` | SDK生产  |
| [application/errors.go:40](../../../application/errors.go#L40) | `package-level` | SDK生产  |
| [application/errors.go:47](../../../application/errors.go#L47) | `Error`         | SDK生产  |
| [application/errors.go:51](../../../application/errors.go#L51) | `Is`            | SDK生产  |

#### P0079

- 定义：[application/errors.go:51](../../../application/errors.go#L51) `canceledSentinelError.Is`；分类N；阶段P1/P3。
- 完整当前类型：`func(target error) bool`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0080

- 定义：[application/errors.go:59](../../../application/errors.go#L59) `HookPanicError`；分类D；阶段P1/P3。
- 完整当前类型：`*github.com/bamboo-services/bamboo-base-go/application/v2.HookPanicError`。

| 位置                                                                                               | 所在函数/声明                                               | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- | -------- |
| [application/app.go:67](../../../application/app.go#L67)                                           | `package-level`                                             | SDK生产  |
| [application/app.go:622](../../../application/app.go#L622)                                         | `createPluginInstance`                                      | SDK生产  |
| [application/app.go:668](../../../application/app.go#L668)                                         | `safeCallStart`                                             | SDK生产  |
| [application/app.go:705](../../../application/app.go#L705)                                         | `safeCallStop`                                              | SDK生产  |
| [application/app.go:789](../../../application/app.go#L789)                                         | `publishExposedKeys`                                        | SDK生产  |
| [application/app.go:913](../../../application/app.go#L913)                                         | `launchWorkers`                                             | SDK生产  |
| [application/app.go:1462](../../../application/app.go#L1462)                                       | `safeCallQuiesce`                                           | SDK生产  |
| [application/component.go:94](../../../application/component.go#L94)                               | `executePipeline`                                           | SDK生产  |
| [application/component.go:212](../../../application/component.go#L212)                             | `executePipeline`                                           | SDK生产  |
| [application/component.go:333](../../../application/component.go#L333)                             | `executePipeline`                                           | SDK生产  |
| [application/component.go:411](../../../application/component.go#L411)                             | `safeCallComponentStop`                                     | SDK生产  |
| [application/errors.go:68](../../../application/errors.go#L68)                                     | `Error`                                                     | SDK生产  |
| [application/errors.go:80](../../../application/errors.go#L80)                                     | `Unwrap`                                                    | SDK生产  |
| [application/expose_atomic_test.go:115](../../../application/expose_atomic_test.go#L115)           | `TestExposeAtomic_ValidationFailuresNoPartialPublish`       | 单测     |
| [application/option_test.go:57](../../../application/option_test.go#L57)                           | `TestErrorsAndSentinels`                                    | 单测     |
| [application/option_test.go:73](../../../application/option_test.go#L73)                           | `TestErrorsAndSentinels`                                    | 单测     |
| [application/stop_completion_test.go:131](../../../application/stop_completion_test.go#L131)       | `TestStopCompletion_PanicHandlingAndSecondCall`             | 单测     |
| [plugins/grpc/safety_regression_test.go:102](../../../plugins/grpc/safety_regression_test.go#L102) | `TestSafetyRegression_StartRegistrarPanic_ReleasesListener` | 单测     |

#### P0081

- 定义：[application/errors.go:80](../../../application/errors.go#L80) `HookPanicError.Unwrap`；分类F；阶段P1/P3。
- 完整当前类型：`func() error`。

| 位置                                                                     | 所在函数/声明            | 引用来源 |
| ------------------------------------------------------------------------ | ------------------------ | -------- |
| [application/option_test.go:78](../../../application/option_test.go#L78) | `TestErrorsAndSentinels` | 单测     |

#### P0082

- 定义：[application/graph.go:13](../../../application/graph.go#L13) `ErrEmptyKey`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                   | 所在函数/声明                       | 引用来源 |
| ---------------------------------------------------------------------- | ----------------------------------- | -------- |
| [application/graph.go:48](../../../application/graph.go#L48)           | `buildGraph`                        | SDK生产  |
| [application/graph.go:80](../../../application/graph.go#L80)           | `buildGraph`                        | SDK生产  |
| [application/graph.go:111](../../../application/graph.go#L111)         | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:99](../../../application/graph_test.go#L99) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0083

- 定义：[application/graph.go:14](../../../application/graph.go#L14) `ErrInvalidKey`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:51](../../../application/graph.go#L51)             | `buildGraph`                        | SDK生产  |
| [application/graph.go:83](../../../application/graph.go#L83)             | `buildGraph`                        | SDK生产  |
| [application/graph.go:114](../../../application/graph.go#L114)           | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:104](../../../application/graph_test.go#L104) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0084

- 定义：[application/graph.go:15](../../../application/graph.go#L15) `ErrRequestKeyShared`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:54](../../../application/graph.go#L54)             | `buildGraph`                        | SDK生产  |
| [application/graph.go:86](../../../application/graph.go#L86)             | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:109](../../../application/graph_test.go#L109) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0085

- 定义：[application/graph.go:16](../../../application/graph.go#L16) `ErrSelfProvidedKey`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:89](../../../application/graph.go#L89)             | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:114](../../../application/graph_test.go#L114) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0086

- 定义：[application/graph.go:17](../../../application/graph.go#L17) `ErrMissingDependency`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:118](../../../application/graph.go#L118)           | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:127](../../../application/graph_test.go#L127) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0087

- 定义：[application/graph.go:18](../../../application/graph.go#L18) `ErrSelfDependency`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:121](../../../application/graph.go#L121)           | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:132](../../../application/graph_test.go#L132) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0088

- 定义：[application/graph.go:19](../../../application/graph.go#L19) `ErrDependencyCycle`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:173](../../../application/graph.go#L173)           | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:140](../../../application/graph_test.go#L140) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0089

- 定义：[application/graph.go:20](../../../application/graph.go#L20) `ErrComponentHasProvides`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:64](../../../application/graph.go#L64)             | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:145](../../../application/graph_test.go#L145) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0090

- 定义：[application/graph.go:21](../../../application/graph.go#L21) `ErrInvalidOverlapKind`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                           | 所在函数/声明              | 引用来源 |
| -------------------------------------------------------------- | -------------------------- | -------- |
| [application/graph.go:208](../../../application/graph.go#L208) | `validateComponentOverlap` | SDK生产  |

#### P0091

- 定义：[application/graph.go:22](../../../application/graph.go#L22) `ErrInvalidOverlapCapacity`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:183](../../../application/graph.go#L183)           | `validateComponentOverlap`          | SDK生产  |
| [application/graph.go:188](../../../application/graph.go#L188)           | `validateComponentOverlap`          | SDK生产  |
| [application/graph.go:193](../../../application/graph.go#L193)           | `validateComponentOverlap`          | SDK生产  |
| [application/graph.go:201](../../../application/graph.go#L201)           | `validateComponentOverlap`          | SDK生产  |
| [application/graph_test.go:163](../../../application/graph_test.go#L163) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0092

- 定义：[application/graph.go:23](../../../application/graph.go#L23) `ErrInvalidOverlapConfig`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:196](../../../application/graph.go#L196)           | `validateComponentOverlap`          | SDK生产  |
| [application/graph.go:204](../../../application/graph.go#L204)           | `validateComponentOverlap`          | SDK生产  |
| [application/graph_test.go:168](../../../application/graph_test.go#L168) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0093

- 定义：[application/graph.go:24](../../../application/graph.go#L24) `ErrNonComponentHasOverlap`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/graph.go:60](../../../application/graph.go#L60)             | `buildGraph`                        | SDK生产  |
| [application/graph_test.go:158](../../../application/graph_test.go#L158) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0094

- 定义：[application/graph.go:25](../../../application/graph.go#L25) `ErrUnknownMode`；分类N；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                         | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------ | ------------- | -------- |
| [application/graph.go:70](../../../application/graph.go#L70) | `buildGraph`  | SDK生产  |

#### P0095

- 定义：[application/graph.go:36](../../../application/graph.go#L36) `buildGraph`；分类S；阶段P1/P3。
- 完整当前类型：`func(infos []github.com/bamboo-services/bamboo-base-go/application/v2.PluginInfo) (*github.com/bamboo-services/bamboo-base-go/application/v2.dependencyGraph, error)`。

| 位置                                                                     | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [application/app.go:357](../../../application/app.go#L357)               | `Run`                               | SDK生产  |
| [application/graph_test.go:45](../../../application/graph_test.go#L45)   | `TestBuildGraph_BasicAndLayering`   | 单测     |
| [application/graph_test.go:75](../../../application/graph_test.go#L75)   | `TestBuildGraph_InputImmutability`  | 单测     |
| [application/graph_test.go:174](../../../application/graph_test.go#L174) | `TestBuildGraph_ValidationFailures` | 单测     |

#### P0096

- 定义：[application/graph.go:179](../../../application/graph.go#L179) `validateComponentOverlap`；分类S；阶段P1/P3。
- 完整当前类型：`func(p github.com/bamboo-services/bamboo-base-go/application/v2.OverlapPolicy) error`。

| 位置                                                         | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------ | ------------- | -------- |
| [application/graph.go:66](../../../application/graph.go#L66) | `buildGraph`  | SDK生产  |

#### P0097

- 定义：[application/option.go:102](../../../application/option.go#L102) `resolveConfig`；分类S；阶段P1/P3。
- 完整当前类型：`func(opts ...github.com/bamboo-services/bamboo-base-go/application/v2.Option) (github.com/bamboo-services/bamboo-base-go/application/v2.Config, error)`。

| 位置                                                                       | 所在函数/声明       | 引用来源 |
| -------------------------------------------------------------------------- | ------------------- | -------- |
| [application/app.go:137](../../../application/app.go#L137)                 | `New`               | SDK生产  |
| [application/option_test.go:120](../../../application/option_test.go#L120) | `TestResolveConfig` | 单测     |
| [application/option_test.go:135](../../../application/option_test.go#L135) | `TestResolveConfig` | 单测     |
| [application/option_test.go:145](../../../application/option_test.go#L145) | `TestResolveConfig` | 单测     |
| [application/option_test.go:158](../../../application/option_test.go#L158) | `TestResolveConfig` | 单测     |
| [application/option_test.go:163](../../../application/option_test.go#L163) | `TestResolveConfig` | 单测     |
| [application/option_test.go:171](../../../application/option_test.go#L171) | `TestResolveConfig` | 单测     |
| [application/option_test.go:176](../../../application/option_test.go#L176) | `TestResolveConfig` | 单测     |
| [application/option_test.go:186](../../../application/option_test.go#L186) | `TestResolveConfig` | 单测     |
| [application/option_test.go:193](../../../application/option_test.go#L193) | `TestResolveConfig` | 单测     |
| [application/option_test.go:200](../../../application/option_test.go#L200) | `TestResolveConfig` | 单测     |
| [application/option_test.go:207](../../../application/option_test.go#L207) | `TestResolveConfig` | 单测     |
| [application/option_test.go:216](../../../application/option_test.go#L216) | `TestResolveConfig` | 单测     |
| [application/option_test.go:217](../../../application/option_test.go#L217) | `TestResolveConfig` | 单测     |

#### P0098

- 定义：[application/option.go:162](../../../application/option.go#L162) `resolveField`；分类S；阶段P1/P3。
- 完整当前类型：`func(explicit bool, explicitVal time.Duration, envKey github.com/bamboo-services/bamboo-base-go/foundation/v2/env.Key, defVal time.Duration) (time.Duration, error)`。

| 位置                                                             | 所在函数/声明   | 引用来源 |
| ---------------------------------------------------------------- | --------------- | -------- |
| [application/option.go:110](../../../application/option.go#L110) | `resolveConfig` | SDK生产  |
| [application/option.go:120](../../../application/option.go#L120) | `resolveConfig` | SDK生产  |
| [application/option.go:130](../../../application/option.go#L130) | `resolveConfig` | SDK生产  |
| [application/option.go:140](../../../application/option.go#L140) | `resolveConfig` | SDK生产  |

#### P0099

- 定义：[application/plugin.go:119](../../../application/plugin.go#L119) `Plugin.Start`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。

| 位置                                                                                         | 所在函数/声明                                          | 引用来源 |
| -------------------------------------------------------------------------------------------- | ------------------------------------------------------ | -------- |
| [application/app.go:677](../../../application/app.go#L677)                                   | `safeCallStart`                                        | SDK生产  |
| [application/builtin/builtin_test.go:42](../../../application/builtin/builtin_test.go#L42)   | `TestBuiltinLogger_NewFactoryNoIO_StartCreatesDir`     | 单测     |
| [application/builtin/builtin_test.go:70](../../../application/builtin/builtin_test.go#L70)   | `TestBuiltinLogger_NewFactoryNoIO_StartCreatesDir`     | 单测     |
| [application/builtin/builtin_test.go:101](../../../application/builtin/builtin_test.go#L101) | `TestBuiltinLogger_IsolationAndGlobalDefaultUntouched` | 单测     |
| [application/builtin/builtin_test.go:104](../../../application/builtin/builtin_test.go#L104) | `TestBuiltinLogger_IsolationAndGlobalDefaultUntouched` | 单测     |
| [application/builtin/builtin_test.go:139](../../../application/builtin/builtin_test.go#L139) | `TestBuiltinLogger_RealLogCloseAndContent`             | 单测     |
| [application/builtin/builtin_test.go:189](../../../application/builtin/builtin_test.go#L189) | `TestBuiltinSnowflake_LifecycleAndIsolation`           | 单测     |
| [application/builtin/builtin_test.go:206](../../../application/builtin/builtin_test.go#L206) | `TestBuiltinSnowflake_LifecycleAndIsolation`           | 单测     |
| [application/builtin/builtin_test.go:209](../../../application/builtin/builtin_test.go#L209) | `TestBuiltinSnowflake_LifecycleAndIsolation`           | 单测     |
| [application/builtin/builtin_test.go:214](../../../application/builtin/builtin_test.go#L214) | `TestBuiltinSnowflake_LifecycleAndIsolation`           | 单测     |
| [application/component.go:221](../../../application/component.go#L221)                       | `executePipeline`                                      | SDK生产  |
| [plugins/cache/plugin_test.go:90](../../../plugins/cache/plugin_test.go#L90)                 | `TestNew_NoneReturnsErrorOnStart`                      | 单测     |
| [plugins/cache/plugin_test.go:97](../../../plugins/cache/plugin_test.go#L97)                 | `TestNew_NoneReturnsErrorOnStart`                      | 单测     |
| [plugins/cache/plugin_test.go:206](../../../plugins/cache/plugin_test.go#L206)               | `TestStop_IdempotentAndShared`                         | 单测     |
| [plugins/cache/plugin_test.go:230](../../../plugins/cache/plugin_test.go#L230)               | `TestStop_NilOrPartialStartSafe`                       | 单测     |
| [plugins/cache/plugin_test.go:240](../../../plugins/cache/plugin_test.go#L240)               | `TestStop_IgnoresContextDone`                          | 单测     |
| [plugins/cache/plugin_test.go:262](../../../plugins/cache/plugin_test.go#L262)               | `TestFromEnv_ValidationErrors`                         | 单测     |
| [plugins/cache/plugin_test.go:277](../../../plugins/cache/plugin_test.go#L277)               | `TestRedis_ConnectionFailureCleansUp`                  | 单测     |
| [plugins/cache/plugin_test.go:332](../../../plugins/cache/plugin_test.go#L332)               | `TestWithLogger`                                       | 单测     |
| [plugins/cron/plugin_test.go:73](../../../plugins/cron/plugin_test.go#L73)                   | `TestPlugin_NoActivatorFailsStart`                     | 单测     |
| [plugins/cron/plugin_test.go:86](../../../plugins/cron/plugin_test.go#L86)                   | `TestPlugin_InvalidSpecFailsStart`                     | 单测     |
| [plugins/cron/plugin_test.go:137](../../../plugins/cron/plugin_test.go#L137)                 | `TestPlugin_IntervalJobTriggerAndCleanStop`            | 单测     |
| [plugins/database/plugin_test.go:169](../../../plugins/database/plugin_test.go#L169)         | `TestPlugin_StartPingFailureCleansUp`                  | 单测     |
| [plugins/database/plugin_test.go:198](../../../plugins/database/plugin_test.go#L198)         | `TestPlugin_StartPingCancelled`                        | 单测     |
| [plugins/database/plugin_test.go:217](../../../plugins/database/plugin_test.go#L217)         | `TestPlugin_StopIdempotentAndRealClose`                | 单测     |
| [plugins/database/plugin_test.go:246](../../../plugins/database/plugin_test.go#L246)         | `TestPlugin_LoggerDiscardWhenNil`                      | 单测     |
| [plugins/database/plugin_test.go:265](../../../plugins/database/plugin_test.go#L265)         | `TestPlugin_FromEnvDriverNotLoadedError`               | 单测     |
| [plugins/email/client_test.go:380](../../../plugins/email/client_test.go#L380)               | `TestPlugin_StartValidation`                           | 单测     |
| [plugins/email/client_test.go:437](../../../plugins/email/client_test.go#L437)               | `TestPlugin_StopIdempotentAndNil`                      | 单测     |
| [plugins/email/client_test.go:466](../../../plugins/email/client_test.go#L466)               | `TestClient_SendSuccessViaFakeSMTP`                    | 单测     |
| [plugins/web-gin/plugin_test.go:40](../../../plugins/web-gin/plugin_test.go#L40)             | `TestWebGin_InstanceIsolationAndGlobalMode`            | 单测     |
| [plugins/web-gin/plugin_test.go:58](../../../plugins/web-gin/plugin_test.go#L58)             | `TestWebGin_InstanceIsolationAndGlobalMode`            | 单测     |

#### P0100

- 定义：[application/plugin.go:126](../../../application/plugin.go#L126) `Plugin.Stop`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。

| 位置                                                                                         | 所在函数/声明                                          | 引用来源 |
| -------------------------------------------------------------------------------------------- | ------------------------------------------------------ | -------- |
| [application/app.go:753](../../../application/app.go#L753)                                   | `safeCallStop`                                         | SDK生产  |
| [application/builtin/builtin_test.go:46](../../../application/builtin/builtin_test.go#L46)   | `TestBuiltinLogger_NewFactoryNoIO_StartCreatesDir`     | 单测     |
| [application/builtin/builtin_test.go:97](../../../application/builtin/builtin_test.go#L97)   | `TestBuiltinLogger_IsolationAndGlobalDefaultUntouched` | 单测     |
| [application/builtin/builtin_test.go:98](../../../application/builtin/builtin_test.go#L98)   | `TestBuiltinLogger_IsolationAndGlobalDefaultUntouched` | 单测     |
| [application/builtin/builtin_test.go:159](../../../application/builtin/builtin_test.go#L159) | `TestBuiltinLogger_RealLogCloseAndContent`             | 单测     |
| [application/builtin/builtin_test.go:162](../../../application/builtin/builtin_test.go#L162) | `TestBuiltinLogger_RealLogCloseAndContent`             | 单测     |
| [application/builtin/builtin_test.go:202](../../../application/builtin/builtin_test.go#L202) | `TestBuiltinSnowflake_LifecycleAndIsolation`           | 单测     |
| [application/builtin/builtin_test.go:203](../../../application/builtin/builtin_test.go#L203) | `TestBuiltinSnowflake_LifecycleAndIsolation`           | 单测     |
| [application/builtin/builtin_test.go:238](../../../application/builtin/builtin_test.go#L238) | `TestBuiltinSnowflake_LifecycleAndIsolation`           | 单测     |
| [application/component.go:430](../../../application/component.go#L430)                       | `safeCallComponentStop`                                | SDK生产  |
| [plugins/cache/plugin_test.go:210](../../../plugins/cache/plugin_test.go#L210)               | `TestStop_IdempotentAndShared`                         | 单测     |
| [plugins/cache/plugin_test.go:211](../../../plugins/cache/plugin_test.go#L211)               | `TestStop_IdempotentAndShared`                         | 单测     |
| [plugins/cache/plugin_test.go:223](../../../plugins/cache/plugin_test.go#L223)               | `TestStop_NilOrPartialStartSafe`                       | 单测     |
| [plugins/cache/plugin_test.go:231](../../../plugins/cache/plugin_test.go#L231)               | `TestStop_NilOrPartialStartSafe`                       | 单测     |
| [plugins/cache/plugin_test.go:245](../../../plugins/cache/plugin_test.go#L245)               | `TestStop_IgnoresContextDone`                          | 单测     |
| [plugins/cache/plugin_test.go:289](../../../plugins/cache/plugin_test.go#L289)               | `TestRedis_ConnectionFailureCleansUp`                  | 单测     |
| [plugins/cache/plugin_test.go:333](../../../plugins/cache/plugin_test.go#L333)               | `TestWithLogger`                                       | 单测     |
| [plugins/cron/plugin_test.go:158](../../../plugins/cron/plugin_test.go#L158)                 | `TestPlugin_IntervalJobTriggerAndCleanStop`            | 单测     |
| [plugins/cron/plugin_test.go:178](../../../plugins/cron/plugin_test.go#L178)                 | `TestPlugin_IntervalJobTriggerAndCleanStop`            | 单测     |
| [plugins/database/plugin_test.go:184](../../../plugins/database/plugin_test.go#L184)         | `TestPlugin_StartPingFailureCleansUp`                  | 单测     |
| [plugins/database/plugin_test.go:228](../../../plugins/database/plugin_test.go#L228)         | `TestPlugin_StopIdempotentAndRealClose`                | 单测     |
| [plugins/database/plugin_test.go:231](../../../plugins/database/plugin_test.go#L231)         | `TestPlugin_StopIdempotentAndRealClose`                | 单测     |
| [plugins/database/plugin_test.go:249](../../../plugins/database/plugin_test.go#L249)         | `TestPlugin_LoggerDiscardWhenNil`                      | 单测     |
| [plugins/email/client_test.go:385](../../../plugins/email/client_test.go#L385)               | `TestPlugin_StartValidation`                           | 单测     |
| [plugins/email/client_test.go:442](../../../plugins/email/client_test.go#L442)               | `TestPlugin_StopIdempotentAndNil`                      | 单测     |
| [plugins/email/client_test.go:446](../../../plugins/email/client_test.go#L446)               | `TestPlugin_StopIdempotentAndNil`                      | 单测     |
| [plugins/email/client_test.go:469](../../../plugins/email/client_test.go#L469)               | `TestClient_SendSuccessViaFakeSMTP`                    | 单测     |
| [plugins/web-gin/plugin_test.go:38](../../../plugins/web-gin/plugin_test.go#L38)             | `TestWebGin_InstanceIsolationAndGlobalMode`            | 单测     |
| [plugins/web-gin/plugin_test.go:57](../../../plugins/web-gin/plugin_test.go#L57)             | `TestWebGin_InstanceIsolationAndGlobalMode`            | 单测     |

#### P0101

- 定义：[application/plugin.go:132](../../../application/plugin.go#L132) `RunnablePlugin.Start`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0102

- 定义：[application/plugin.go:132](../../../application/plugin.go#L132) `RunnablePlugin.Stop`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0103

- 定义：[application/plugin.go:141](../../../application/plugin.go#L141) `RunnablePlugin.Run`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。

| 位置                                                                         | 所在函数/声明                               | 引用来源 |
| ---------------------------------------------------------------------------- | ------------------------------------------- | -------- |
| [application/app.go:922](../../../application/app.go#L922)                   | `launchWorkers`                             | SDK生产  |
| [application/component.go:342](../../../application/component.go#L342)       | `executePipeline`                           | SDK生产  |
| [plugins/cron/plugin_test.go:144](../../../plugins/cron/plugin_test.go#L144) | `TestPlugin_IntervalJobTriggerAndCleanStop` | 单测     |

#### P0104

- 定义：[application/plugin.go:154](../../../application/plugin.go#L154) `Quiescer.Quiesce`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) error`。

| 位置                                                         | 所在函数/声明     | 引用来源 |
| ------------------------------------------------------------ | ----------------- | -------- |
| [application/app.go:1490](../../../application/app.go#L1490) | `safeCallQuiesce` | SDK生产  |

#### P0105

- 定义：[application/plugin.go:182](../../../application/plugin.go#L182) `RunResult.CancelErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                     | 所在函数/声明                                                         | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- | -------- |
| [application/activator.go:427](../../../application/activator.go#L427)                                   | `cancelUndispatchedItem`                                              | SDK生产  |
| [application/activator_acceptance_test.go:522](../../../application/activator_acceptance_test.go#L522)   | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`  | 单测     |
| [application/activator_acceptance_test.go:523](../../../application/activator_acceptance_test.go#L523)   | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`  | 单测     |
| [application/component.go:161](../../../application/component.go#L161)                                   | `executePipeline`                                                     | SDK生产  |
| [application/component.go:289](../../../application/component.go#L289)                                   | `executePipeline`                                                     | SDK生产  |
| [application/component.go:369](../../../application/component.go#L369)                                   | `executePipeline`                                                     | SDK生产  |
| [application/component_engine_test.go:434](../../../application/component_engine_test.go#L434)           | `TestComponentEngine_UndispatchedCancelledOnParentClose`              | 单测     |
| [application/component_engine_test.go:435](../../../application/component_engine_test.go#L435)           | `TestComponentEngine_UndispatchedCancelledOnParentClose`              | 单测     |
| [application/kernel_real_exit_test.go:547](../../../application/kernel_real_exit_test.go#L547)           | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly` | 单测     |
| [application/kernel_real_exit_test.go:549](../../../application/kernel_real_exit_test.go#L549)           | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly` | 单测     |
| [application/kernel_shutdown_races_test.go:322](../../../application/kernel_shutdown_races_test.go#L322) | `TestF3_AppStoppingCancelsQueueImmediately`                           | 单测     |
| [application/supervisor.go:65](../../../application/supervisor.go#L65)                                   | `computeWaitError`                                                    | SDK生产  |
| [application/supervisor.go:66](../../../application/supervisor.go#L66)                                   | `computeWaitError`                                                    | SDK生产  |
| [plugins/cron/real_queue_test.go:324](../../../plugins/cron/real_queue_test.go#L324)                     | `TestCron_RealApp_QueueFIFOAndFullBehavior`                           | 单测     |
| [plugins/cron/real_queue_test.go:327](../../../plugins/cron/real_queue_test.go#L327)                     | `TestCron_RealApp_QueueFIFOAndFullBehavior`                           | 单测     |
| [plugins/cron/real_queue_test.go:328](../../../plugins/cron/real_queue_test.go#L328)                     | `TestCron_RealApp_QueueFIFOAndFullBehavior`                           | 单测     |

#### P0106

- 定义：[application/plugin.go:183](../../../application/plugin.go#L183) `RunResult.FactoryErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                                        | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------- | -------- |
| [application/activator_acceptance_test.go:516](../../../application/activator_acceptance_test.go#L516) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown` | 单测     |
| [application/activator_acceptance_test.go:517](../../../application/activator_acceptance_test.go#L517) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown` | 单测     |
| [application/component.go:110](../../../application/component.go#L110)                                 | `executePipeline`                                                    | SDK生产  |
| [application/component.go:116](../../../application/component.go#L116)                                 | `executePipeline`                                                    | SDK生产  |
| [application/supervisor.go:62](../../../application/supervisor.go#L62)                                 | `computeWaitError`                                                   | SDK生产  |
| [application/supervisor.go:63](../../../application/supervisor.go#L63)                                 | `computeWaitError`                                                   | SDK生产  |

#### P0107

- 定义：[application/plugin.go:184](../../../application/plugin.go#L184) `RunResult.StartErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                                        | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------- | -------- |
| [application/activator_acceptance_test.go:516](../../../application/activator_acceptance_test.go#L516) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown` | 单测     |
| [application/activator_acceptance_test.go:517](../../../application/activator_acceptance_test.go#L517) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown` | 单测     |
| [application/component.go:233](../../../application/component.go#L233)                                 | `executePipeline`                                                    | SDK生产  |
| [application/component.go:383](../../../application/component.go#L383)                                 | `executePipeline`                                                    | SDK生产  |
| [application/dynamic_retention_test.go:629](../../../application/dynamic_retention_test.go#L629)       | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`         | 单测     |
| [application/dynamic_retention_test.go:630](../../../application/dynamic_retention_test.go#L630)       | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`         | 单测     |
| [application/supervisor.go:70](../../../application/supervisor.go#L70)                                 | `computeWaitError`                                                   | SDK生产  |
| [application/supervisor.go:71](../../../application/supervisor.go#L71)                                 | `computeWaitError`                                                   | SDK生产  |

#### P0108

- 定义：[application/plugin.go:185](../../../application/plugin.go#L185) `RunResult.ExposeErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                   | 所在函数/声明      | 引用来源 |
| ---------------------------------------------------------------------- | ------------------ | -------- |
| [application/supervisor.go:73](../../../application/supervisor.go#L73) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:74](../../../application/supervisor.go#L74) | `computeWaitError` | SDK生产  |

#### P0109

- 定义：[application/plugin.go:186](../../../application/plugin.go#L186) `RunResult.QuiesceErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                   | 所在函数/声明      | 引用来源 |
| ---------------------------------------------------------------------- | ------------------ | -------- |
| [application/supervisor.go:76](../../../application/supervisor.go#L76) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:77](../../../application/supervisor.go#L77) | `computeWaitError` | SDK生产  |

#### P0110

- 定义：[application/plugin.go:187](../../../application/plugin.go#L187) `RunResult.RunErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                                         | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------- | -------- |
| [application/activator_acceptance_test.go:513](../../../application/activator_acceptance_test.go#L513) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`  | 单测     |
| [application/activator_acceptance_test.go:514](../../../application/activator_acceptance_test.go#L514) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`  | 单测     |
| [application/component.go:345](../../../application/component.go#L345)                                 | `executePipeline`                                                     | SDK生产  |
| [application/component.go:383](../../../application/component.go#L383)                                 | `executePipeline`                                                     | SDK生产  |
| [application/component.go:384](../../../application/component.go#L384)                                 | `executePipeline`                                                     | SDK生产  |
| [application/component_engine_test.go:428](../../../application/component_engine_test.go#L428)         | `TestComponentEngine_UndispatchedCancelledOnParentClose`              | 单测     |
| [application/component_engine_test.go:429](../../../application/component_engine_test.go#L429)         | `TestComponentEngine_UndispatchedCancelledOnParentClose`              | 单测     |
| [application/dynamic_retention_test.go:635](../../../application/dynamic_retention_test.go#L635)       | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`          | 单测     |
| [application/dynamic_retention_test.go:636](../../../application/dynamic_retention_test.go#L636)       | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`          | 单测     |
| [application/dynamic_retention_test.go:755](../../../application/dynamic_retention_test.go#L755)       | `TestDynamicRetention_RegressionC_StartTimeout_NoAppShutdown`         | 单测     |
| [application/kernel_real_exit_test.go:544](../../../application/kernel_real_exit_test.go#L544)         | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly` | 单测     |
| [application/kernel_real_exit_test.go:545](../../../application/kernel_real_exit_test.go#L545)         | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly` | 单测     |
| [application/supervisor.go:79](../../../application/supervisor.go#L79)                                 | `computeWaitError`                                                    | SDK生产  |
| [application/supervisor.go:80](../../../application/supervisor.go#L80)                                 | `computeWaitError`                                                    | SDK生产  |
| [plugins/cron/real_queue_test.go:330](../../../plugins/cron/real_queue_test.go#L330)                   | `TestCron_RealApp_QueueFIFOAndFullBehavior`                           | 单测     |
| [plugins/cron/real_queue_test.go:331](../../../plugins/cron/real_queue_test.go#L331)                   | `TestCron_RealApp_QueueFIFOAndFullBehavior`                           | 单测     |

#### P0111

- 定义：[application/plugin.go:188](../../../application/plugin.go#L188) `RunResult.StopErr`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                                        | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------- | -------- |
| [application/activator_acceptance_test.go:516](../../../application/activator_acceptance_test.go#L516) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown` | 单测     |
| [application/activator_acceptance_test.go:517](../../../application/activator_acceptance_test.go#L517) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown` | 单测     |
| [application/component.go:237](../../../application/component.go#L237)                                 | `executePipeline`                                                    | SDK生产  |
| [application/component.go:242](../../../application/component.go#L242)                                 | `executePipeline`                                                    | SDK生产  |
| [application/component.go:294](../../../application/component.go#L294)                                 | `executePipeline`                                                    | SDK生产  |
| [application/component.go:299](../../../application/component.go#L299)                                 | `executePipeline`                                                    | SDK生产  |
| [application/component.go:363](../../../application/component.go#L363)                                 | `executePipeline`                                                    | SDK生产  |
| [application/component.go:379](../../../application/component.go#L379)                                 | `executePipeline`                                                    | SDK生产  |
| [application/component.go:383](../../../application/component.go#L383)                                 | `executePipeline`                                                    | SDK生产  |
| [application/component.go:384](../../../application/component.go#L384)                                 | `executePipeline`                                                    | SDK生产  |
| [application/supervisor.go:82](../../../application/supervisor.go#L82)                                 | `computeWaitError`                                                   | SDK生产  |
| [application/supervisor.go:83](../../../application/supervisor.go#L83)                                 | `computeWaitError`                                                   | SDK生产  |

#### P0112

- 定义：[application/plugin.go:207](../../../application/plugin.go#L207) `RunHandle.Wait`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) (github.com/bamboo-services/bamboo-base-go/application/v2.RunResult, error)`。

| 位置                                                                                                     | 所在函数/声明                                                                             | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | -------- |
| [application/activator_acceptance_test.go:152](../../../application/activator_acceptance_test.go#L152)   | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned`                            | 单测     |
| [application/activator_acceptance_test.go:238](../../../application/activator_acceptance_test.go#L238)   | `TestActivatorAcceptance_OverlapAllow_LimitExceeded`                                      | 单测     |
| [application/activator_acceptance_test.go:239](../../../application/activator_acceptance_test.go#L239)   | `TestActivatorAcceptance_OverlapAllow_LimitExceeded`                                      | 单测     |
| [application/activator_acceptance_test.go:383](../../../application/activator_acceptance_test.go#L383)   | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测     |
| [application/activator_acceptance_test.go:397](../../../application/activator_acceptance_test.go#L397)   | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测     |
| [application/activator_acceptance_test.go:405](../../../application/activator_acceptance_test.go#L405)   | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测     |
| [application/activator_acceptance_test.go:501](../../../application/activator_acceptance_test.go#L501)   | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`                      | 单测     |
| [application/activator_acceptance_test.go:621](../../../application/activator_acceptance_test.go#L621)   | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测     |
| [application/activator_acceptance_test.go:622](../../../application/activator_acceptance_test.go#L622)   | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测     |
| [application/component_engine_test.go:153](../../../application/component_engine_test.go#L153)           | `TestComponentEngine_Skip_BarrierStopsUntilStopReturned`                                  | 单测     |
| [application/component_engine_test.go:168](../../../application/component_engine_test.go#L168)           | `TestComponentEngine_Skip_BarrierStopsUntilStopReturned`                                  | 单测     |
| [application/component_engine_test.go:226](../../../application/component_engine_test.go#L226)           | `TestComponentEngine_Allow_ConcurrencyLimit`                                              | 单测     |
| [application/component_engine_test.go:227](../../../application/component_engine_test.go#L227)           | `TestComponentEngine_Allow_ConcurrencyLimit`                                              | 单测     |
| [application/component_engine_test.go:234](../../../application/component_engine_test.go#L234)           | `TestComponentEngine_Allow_ConcurrencyLimit`                                              | 单测     |
| [application/component_engine_test.go:324](../../../application/component_engine_test.go#L324)           | `TestComponentEngine_Queue_SharedFIFO_MaxQueue_ContextIndependence`                       | 单测     |
| [application/component_engine_test.go:338](../../../application/component_engine_test.go#L338)           | `TestComponentEngine_Queue_SharedFIFO_MaxQueue_ContextIndependence`                       | 单测     |
| [application/component_engine_test.go:349](../../../application/component_engine_test.go#L349)           | `TestComponentEngine_Queue_SharedFIFO_MaxQueue_ContextIndependence`                       | 单测     |
| [application/component_engine_test.go:418](../../../application/component_engine_test.go#L418)           | `TestComponentEngine_UndispatchedCancelledOnParentClose`                                  | 单测     |
| [application/component_engine_test.go:421](../../../application/component_engine_test.go#L421)           | `TestComponentEngine_UndispatchedCancelledOnParentClose`                                  | 单测     |
| [application/component_engine_test.go:540](../../../application/component_engine_test.go#L540)           | `TestComponentEngine_NestedQueue_WouldBlock_And_Cycle`                                    | 单测     |
| [application/component_engine_test.go:541](../../../application/component_engine_test.go#L541)           | `TestComponentEngine_NestedQueue_WouldBlock_And_Cycle`                                    | 单测     |
| [application/component_engine_test.go:638](../../../application/component_engine_test.go#L638)           | `TestComponentEngine_StopPanic_RetainsDepsAndTriggersAppShutdown`                         | 单测     |
| [application/component_engine_test.go:694](../../../application/component_engine_test.go#L694)           | `TestComponentEngine_Wait_ReturnsImpliesSlotReleased`                                     | 单测     |
| [application/component_engine_test.go:701](../../../application/component_engine_test.go#L701)           | `TestComponentEngine_Wait_ReturnsImpliesSlotReleased`                                     | 单测     |
| [application/component_engine_test.go:710](../../../application/component_engine_test.go#L710)           | `TestComponentEngine_Wait_ReturnsImpliesSlotReleased`                                     | 单测     |
| [application/component_engine_test.go:729](../../../application/component_engine_test.go#L729)           | `TestComponentEngine_Wait_ReturnsImpliesSlotReleased`                                     | 单测     |
| [application/dynamic_retention_test.go:238](../../../application/dynamic_retention_test.go#L238)         | `runSubtestRegressionA`                                                                   | 单测     |
| [application/dynamic_retention_test.go:422](../../../application/dynamic_retention_test.go#L422)         | `TestDynamicRetention_RegressionB_StuckInRun_RetainsDependency`                           | 单测     |
| [application/dynamic_retention_test.go:614](../../../application/dynamic_retention_test.go#L614)         | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`                              | 单测     |
| [application/dynamic_retention_test.go:754](../../../application/dynamic_retention_test.go#L754)         | `TestDynamicRetention_RegressionC_StartTimeout_NoAppShutdown`                             | 单测     |
| [application/kernel_real_exit_test.go:536](../../../application/kernel_real_exit_test.go#L536)           | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly`                     | 单测     |
| [application/kernel_shutdown_races_test.go:318](../../../application/kernel_shutdown_races_test.go#L318) | `TestF3_AppStoppingCancelsQueueImmediately`                                               | 单测     |
| [plugins/cron/real_queue_test.go:320](../../../plugins/cron/real_queue_test.go#L320)                     | `TestCron_RealApp_QueueFIFOAndFullBehavior`                                               | 单测     |
| [plugins/grpc/plugin_app_context_test.go:155](../../../plugins/grpc/plugin_app_context_test.go#L155)     | `unaryContextCheckHandler`                                                                | 单测     |
| [plugins/grpc/plugin_app_context_test.go:258](../../../plugins/grpc/plugin_app_context_test.go#L258)     | `streamContextCheckHandler`                                                               | 单测     |

#### P0113

- 定义：[application/plugin.go:250](../../../application/plugin.go#L250) `Activator.Activate`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context, compKey github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey.Key, opts ...github.com/bamboo-services/bamboo-base-go/application/v2.ActivateOption) (github.com/bamboo-services/bamboo-base-go/application/v2.RunHandle, error)`。

| 位置                                                                                                   | 所在函数/声明                                                                             | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------- | -------- |
| [application/activator_acceptance_test.go:124](../../../application/activator_acceptance_test.go#L124) | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned`                            | 单测     |
| [application/activator_acceptance_test.go:134](../../../application/activator_acceptance_test.go#L134) | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned`                            | 单测     |
| [application/activator_acceptance_test.go:144](../../../application/activator_acceptance_test.go#L144) | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned`                            | 单测     |
| [application/activator_acceptance_test.go:158](../../../application/activator_acceptance_test.go#L158) | `TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned`                            | 单测     |
| [application/activator_acceptance_test.go:219](../../../application/activator_acceptance_test.go#L219) | `TestActivatorAcceptance_OverlapAllow_LimitExceeded`                                      | 单测     |
| [application/activator_acceptance_test.go:223](../../../application/activator_acceptance_test.go#L223) | `TestActivatorAcceptance_OverlapAllow_LimitExceeded`                                      | 单测     |
| [application/activator_acceptance_test.go:232](../../../application/activator_acceptance_test.go#L232) | `TestActivatorAcceptance_OverlapAllow_LimitExceeded`                                      | 单测     |
| [application/activator_acceptance_test.go:343](../../../application/activator_acceptance_test.go#L343) | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测     |
| [application/activator_acceptance_test.go:356](../../../application/activator_acceptance_test.go#L356) | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测     |
| [application/activator_acceptance_test.go:369](../../../application/activator_acceptance_test.go#L369) | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测     |
| [application/activator_acceptance_test.go:375](../../../application/activator_acceptance_test.go#L375) | `TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence` | 单测     |
| [application/activator_acceptance_test.go:465](../../../application/activator_acceptance_test.go#L465) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`                      | 单测     |
| [application/activator_acceptance_test.go:474](../../../application/activator_acceptance_test.go#L474) | `TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown`                      | 单测     |
| [application/activator_acceptance_test.go:580](../../../application/activator_acceptance_test.go#L580) | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测     |
| [application/activator_acceptance_test.go:587](../../../application/activator_acceptance_test.go#L587) | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测     |
| [application/activator_acceptance_test.go:604](../../../application/activator_acceptance_test.go#L604) | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测     |
| [application/activator_acceptance_test.go:612](../../../application/activator_acceptance_test.go#L612) | `TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle`                      | 单测     |
| [application/dynamic_retention_test.go:167](../../../application/dynamic_retention_test.go#L167)       | `runSubtestRegressionA`                                                                   | 单测     |
| [application/dynamic_retention_test.go:335](../../../application/dynamic_retention_test.go#L335)       | `TestDynamicRetention_RegressionB_StuckInRun_RetainsDependency`                           | 单测     |
| [application/dynamic_retention_test.go:530](../../../application/dynamic_retention_test.go#L530)       | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`                              | 单测     |
| [application/dynamic_retention_test.go:712](../../../application/dynamic_retention_test.go#L712)       | `TestDynamicRetention_RegressionC_StartTimeout_NoAppShutdown`                             | 单测     |
| [application/dynamic_retention_test.go:844](../../../application/dynamic_retention_test.go#L844)       | `TestDynamicRetention_RegressionD_StartError_StopPanic_CoordinatesAppFailure`             | 单测     |
| [application/kernel_real_exit_test.go:477](../../../application/kernel_real_exit_test.go#L477)         | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly`                     | 单测     |
| [application/kernel_real_exit_test.go:490](../../../application/kernel_real_exit_test.go#L490)         | `TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly`                     | 单测     |
| [plugins/cron/plugin.go:347](../../../plugins/cron/plugin.go#L347)                                     | `dispatchTick`                                                                            | SDK生产  |
| [plugins/cron/real_app_test.go:36](../../../plugins/cron/real_app_test.go#L36)                         | `Activate`                                                                                | 单测     |
| [plugins/cron/real_queue_test.go:31](../../../plugins/cron/real_queue_test.go#L31)                     | `Activate`                                                                                | 单测     |
| [plugins/grpc/plugin_app_context_test.go:140](../../../plugins/grpc/plugin_app_context_test.go#L140)   | `unaryContextCheckHandler`                                                                | 单测     |
| [plugins/grpc/plugin_app_context_test.go:246](../../../plugins/grpc/plugin_app_context_test.go#L246)   | `streamContextCheckHandler`                                                               | 单测     |

#### P0114

- 定义：[application/shutdown.go:19](../../../application/shutdown.go#L19) `StuckInstance.Err`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                                   | 所在函数/声明              | 引用来源 |
| ---------------------------------------------------------------------- | -------------------------- | -------- |
| [application/activator.go:291](../../../application/activator.go#L291) | `snapshotLiveTasks`        | SDK生产  |
| [application/app.go:722](../../../application/app.go#L722)             | `safeCallStop`             | SDK生产  |
| [application/app.go:1133](../../../application/app.go#L1133)           | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1178](../../../application/app.go#L1178)           | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1226](../../../application/app.go#L1226)           | `stopStaticInstanceRecord` | SDK生产  |
| [application/app.go:1262](../../../application/app.go#L1262)           | `waitForStartDone`         | SDK生产  |
| [application/app.go:1280](../../../application/app.go#L1280)           | `waitForStartDone`         | SDK生产  |
| [application/app.go:1448](../../../application/app.go#L1448)           | `executeShutdownTopology`  | SDK生产  |
| [application/component.go:471](../../../application/component.go#L471) | `handleStopPanic`          | SDK生产  |

#### P0115

- 定义：[application/shutdown.go:26](../../../application/shutdown.go#L26) `StopIncompleteError`；分类D；阶段P1/P3。
- 完整当前类型：`*github.com/bamboo-services/bamboo-base-go/application/v2.StopIncompleteError`。

| 位置                                                                                                                 | 所在函数/声明                                                                     | 引用来源   |
| -------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | ---------- |
| [application/app.go:1064](../../../application/app.go#L1064)                                                         | `Shutdown`                                                                        | SDK生产    |
| [application/app_test.go:345](../../../application/app_test.go#L345)                                                 | `TestApp_StopPanic_RetainsDependency`                                             | 单测       |
| [application/dynamic_retention_test.go:378](../../../application/dynamic_retention_test.go#L378)                     | `TestDynamicRetention_RegressionB_StuckInRun_RetainsDependency`                   | 单测       |
| [application/dynamic_retention_test.go:572](../../../application/dynamic_retention_test.go#L572)                     | `TestDynamicRetention_RegressionC_StartTimeout_WithShutdown`                      | 单测       |
| [application/dynamic_retention_test.go:896](../../../application/dynamic_retention_test.go#L896)                     | `TestDynamicRetention_RegressionD_StartError_StopPanic_CoordinatesAppFailure`     | 单测       |
| [application/kernel_real_exit_test.go:174](../../../application/kernel_real_exit_test.go#L174)                       | `TestKernelStartTimeout_NoExternalShutdown_RealExit`                              | 单测       |
| [application/kernel_real_exit_test.go:351](../../../application/kernel_real_exit_test.go#L351)                       | `TestKernelRealExit_WorkerIgnoresCancel_StopReturnsNil_ShutdownIncomplete`        | 单测       |
| [application/kernel_shutdown_races_test.go:186](../../../application/kernel_shutdown_races_test.go#L186)             | `TestF4_StopNilWhileRunBlocks`                                                    | 单测       |
| [application/quiesce_budget_test.go:141](../../../application/quiesce_budget_test.go#L141)                           | `TestQuiesceBudget_RetainedDepsAndStuckInstance`                                  | 单测       |
| [application/retained_dependency_test.go:153](../../../application/retained_dependency_test.go#L153)                 | `TestApp_RetainedDependency_PropagationAndClosure`                                | 单测       |
| [application/shutdown.go:31](../../../application/shutdown.go#L31)                                                   | `Error`                                                                           | SDK生产    |
| [application/stop_budget_test.go:143](../../../application/stop_budget_test.go#L143)                                 | `TestApp_StopBudget_TimeoutRetainsDepsAndUnblocks`                                | 单测       |
| [plugins/async/dependency_lifecycle_test.go:300](../../../plugins/async/dependency_lifecycle_test.go#L300)           | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete` | 单测       |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:306](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L306) | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`     | 仓内消费者 |

#### P0116

- 定义：[application/shutdown.go:40](../../../application/shutdown.go#L40) `shutdownResult.err`；分类S；阶段P1/P3。
- 完整当前类型：`error`。

| 位置                                                         | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------ | ------------- | -------- |
| [application/app.go:431](../../../application/app.go#L431)   | `Run`         | SDK生产  |
| [application/app.go:991](../../../application/app.go#L991)   | `Shutdown`    | SDK生产  |
| [application/app.go:1004](../../../application/app.go#L1004) | `Shutdown`    | SDK生产  |
| [application/app.go:1037](../../../application/app.go#L1037) | `Shutdown`    | SDK生产  |
| [application/app.go:1071](../../../application/app.go#L1071) | `Shutdown`    | SDK生产  |

#### P0117

- 定义：[application/supervisor.go:35](../../../application/supervisor.go#L35) `runHandleImpl.Wait`；分类S；阶段P1/P3。
- 完整当前类型：`func(ctx context.Context) (github.com/bamboo-services/bamboo-base-go/application/v2.RunResult, error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.RunHandle`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0118

- 定义：[application/supervisor.go:61](../../../application/supervisor.go#L61) `RunResult.computeWaitError`；分类S；阶段P1/P3。
- 完整当前类型：`func() error`。

| 位置                                                                   | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------- | ------------- | -------- |
| [application/supervisor.go:44](../../../application/supervisor.go#L44) | `Wait`        | SDK生产  |

#### P0119

- 定义：[application/supervisor.go:69](../../../application/supervisor.go#L69) `computeWaitError.errs`；分类S；阶段P1/P3。
- 完整当前类型：`[]error`。

| 位置                                                                   | 所在函数/声明      | 引用来源 |
| ---------------------------------------------------------------------- | ------------------ | -------- |
| [application/supervisor.go:71](../../../application/supervisor.go#L71) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:71](../../../application/supervisor.go#L71) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:74](../../../application/supervisor.go#L74) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:74](../../../application/supervisor.go#L74) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:77](../../../application/supervisor.go#L77) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:77](../../../application/supervisor.go#L77) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:80](../../../application/supervisor.go#L80) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:80](../../../application/supervisor.go#L80) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:83](../../../application/supervisor.go#L83) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:83](../../../application/supervisor.go#L83) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:86](../../../application/supervisor.go#L86) | `computeWaitError` | SDK生产  |
| [application/supervisor.go:87](../../../application/supervisor.go#L87) | `computeWaitError` | SDK生产  |

#### P0120

- 定义：[foundation/apperr/error.go:10](../../../foundation/apperr/error.go#L10) `Error`；分类D；阶段P1。
- 完整当前类型：`*github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error`。

| 位置                                                                                                                 | 所在函数/声明                                  | 引用来源 |
| -------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- | -------- |
| [application/builtin_lifecycle_test.go:38](../../../application/builtin_lifecycle_test.go#L38)                       | `Start`                                        | 单测     |
| [application/builtin_lifecycle_test.go:44](../../../application/builtin_lifecycle_test.go#L44)                       | `Start`                                        | 单测     |
| [application/context.go:404](../../../application/context.go#L404)                                                   | `newAppErr`                                    | SDK生产  |
| [application/context.go:409](../../../application/context.go#L409)                                                   | `newAppErr`                                    | SDK生产  |
| [application/context.go:440](../../../application/context.go#L440)                                                   | `Get`                                          | SDK生产  |
| [application/context.go:495](../../../application/context.go#L495)                                                   | `castEntry`                                    | SDK生产  |
| [foundation/apperr/error.go:32](../../../foundation/apperr/error.go#L32)                                             | `Error`                                        | SDK生产  |
| [foundation/apperr/error.go:63](../../../foundation/apperr/error.go#L63)                                             | `Unwrap`                                       | SDK生产  |
| [foundation/apperr/error_test.go:68](../../../foundation/apperr/error_test.go#L68)                                   | `TestErrorUnwrapAndIsAs`                       | 单测     |
| [foundation/apperr/error_test.go:86](../../../foundation/apperr/error_test.go#L86)                                   | `TestErrorUnwrapAndIsAs`                       | 单测     |
| [foundation/apperr/error_test.go:97](../../../foundation/apperr/error_test.go#L97)                                   | `TestErrorJSONSerializationNoLeak`             | 单测     |
| [foundation/apperr/error_test.go:128](../../../foundation/apperr/error_test.go#L128)                                 | `TestErrorJSONSerializationNoLeak`             | 单测     |
| [foundation/apperr/new.go:21](../../../foundation/apperr/new.go#L21)                                                 | `New`                                          | SDK生产  |
| [foundation/apperr/new.go:44](../../../foundation/apperr/new.go#L44)                                                 | `New`                                          | SDK生产  |
| [plugins/cache/config.go:243](../../../plugins/cache/config.go#L243)                                                 | `Get`                                          | SDK生产  |
| [plugins/database/context.go:17](../../../plugins/database/context.go#L17)                                           | `Get`                                          | SDK生产  |
| [plugins/grpc/grpcresult/response.go:128](../../../plugins/grpc/grpcresult/response.go#L128)                         | `ToStatusError`                                | SDK生产  |
| [plugins/grpc/grpcresult/response_test.go:113](../../../plugins/grpc/grpcresult/response_test.go#L113)               | `TestToStatusError_GroupMapping`               | 单测     |
| [plugins/grpc/grpcresult/response_test.go:136](../../../plugins/grpc/grpcresult/response_test.go#L136)               | `TestToStatusError_SecurityRawErrorNotLeaked`  | 单测     |
| [plugins/grpc/plugin.go:604](../../../plugins/grpc/plugin.go#L604)                                                   | `normalizeRPCError`                            | SDK生产  |
| [plugins/grpc/utility/grpc_util.go:27](../../../plugins/grpc/utility/grpc_util.go#L27)                               | `ExtractMetadata`                              | SDK生产  |
| [plugins/web-gin/httpresult/result.go:53](../../../plugins/web-gin/httpresult/result.go#L53)                         | `buildEnvelope`                                | SDK生产  |
| [plugins/web-gin/httpresult/result.go:129](../../../plugins/web-gin/httpresult/result.go#L129)                       | `Error`                                        | SDK生产  |
| [plugins/web-gin/httpresult/result.go:141](../../../plugins/web-gin/httpresult/result.go#L141)                       | `AbortError`                                   | SDK生产  |
| [plugins/web-gin/httpresult/status_test.go:51](../../../plugins/web-gin/httpresult/status_test.go#L51)               | `TestError_FallbackAndEnvelope`                | 单测     |
| [plugins/web-gin/middleware/middleware.go:73](../../../plugins/web-gin/middleware/middleware.go#L73)                 | `Response`                                     | SDK生产  |
| [plugins/web-gin/validator/error_security_test.go:81](../../../plugins/web-gin/validator/error_security_test.go#L81) | `TestSecurity_UnderlyingNetworkErrorNotLeaked` | 单测     |

#### P0121

- 定义：[foundation/apperr/error.go:23](../../../foundation/apperr/error.go#L23) `Error.Err`；分类N；阶段P1。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明                                 | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | --------------------------------------------- | -------- |
| [application/app_test.go:411](../../../application/app_test.go#L411)                                   | `TestApp_Context_RequiresStartScopeWhitelist` | 单测     |
| [application/app_test.go:412](../../../application/app_test.go#L412)                                   | `TestApp_Context_RequiresStartScopeWhitelist` | 单测     |
| [application/context.go:412](../../../application/context.go#L412)                                     | `newAppErr`                                   | SDK生产  |
| [application/example_test.go:36](../../../application/example_test.go#L36)                             | `Run`                                         | 单测     |
| [foundation/apperr/error.go:47](../../../foundation/apperr/error.go#L47)                               | `Error`                                       | SDK生产  |
| [foundation/apperr/error.go:49](../../../foundation/apperr/error.go#L49)                               | `Error`                                       | SDK生产  |
| [foundation/apperr/error.go:51](../../../foundation/apperr/error.go#L51)                               | `Error`                                       | SDK生产  |
| [foundation/apperr/error.go:67](../../../foundation/apperr/error.go#L67)                               | `Unwrap`                                      | SDK生产  |
| [foundation/apperr/error_test.go:71](../../../foundation/apperr/error_test.go#L71)                     | `TestErrorUnwrapAndIsAs`                      | 单测     |
| [foundation/apperr/error_test.go:88](../../../foundation/apperr/error_test.go#L88)                     | `TestErrorUnwrapAndIsAs`                      | 单测     |
| [foundation/apperr/error_test.go:101](../../../foundation/apperr/error_test.go#L101)                   | `TestErrorJSONSerializationNoLeak`            | 单测     |
| [foundation/apperr/new.go:47](../../../foundation/apperr/new.go#L47)                                   | `New`                                         | SDK生产  |
| [foundation/apperr/new_test.go:56](../../../foundation/apperr/new_test.go#L56)                         | `TestNew_CauseHandlingAndUnwrap`              | 单测     |
| [foundation/apperr/new_test.go:57](../../../foundation/apperr/new_test.go#L57)                         | `TestNew_CauseHandlingAndUnwrap`              | 单测     |
| [foundation/apperr/new_test.go:61](../../../foundation/apperr/new_test.go#L61)                         | `TestNew_CauseHandlingAndUnwrap`              | 单测     |
| [foundation/apperr/new_test.go:62](../../../foundation/apperr/new_test.go#L62)                         | `TestNew_CauseHandlingAndUnwrap`              | 单测     |
| [plugins/grpc/grpcresult/response_test.go:143](../../../plugins/grpc/grpcresult/response_test.go#L143) | `TestToStatusError_SecurityRawErrorNotLeaked` | 单测     |
| [plugins/web-gin/plugin_test.go:96](../../../plugins/web-gin/plugin_test.go#L96)                       | `TestWebGin_ContainerBridgeAndRequestKey`     | 单测     |

#### P0122

- 定义：[foundation/apperr/error.go:63](../../../foundation/apperr/error.go#L63) `Error.Unwrap`；分类F；阶段P1。
- 完整当前类型：`func() error`。

| 位置                                                                           | 所在函数/声明                    | 引用来源 |
| ------------------------------------------------------------------------------ | -------------------------------- | -------- |
| [foundation/apperr/new_test.go:56](../../../foundation/apperr/new_test.go#L56) | `TestNew_CauseHandlingAndUnwrap` | 单测     |
| [foundation/apperr/new_test.go:61](../../../foundation/apperr/new_test.go#L61) | `TestNew_CauseHandlingAndUnwrap` | 单测     |

#### P0123

- 定义：[foundation/apperr/new.go:21](../../../foundation/apperr/new.go#L21) `New`；分类A；阶段P1。
- 完整当前类型：`func(code *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.ErrorCode, msg github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.ErrMessage, errs ...error) *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error`。

| 位置                                                                                                                   | 所在函数/声明                              | 引用来源   |
| ---------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ | ---------- |
| [examples/v2/http-db-cache/main.go:71](../../../examples/v2/http-db-cache/main.go#L71)                                 | `main`                                     | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:77](../../../examples/v2/http-db-cache/main.go#L77)                                 | `main`                                     | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:84](../../../examples/v2/http-db-cache/main.go#L84)                                 | `main`                                     | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:90](../../../examples/v2/http-db-cache/main.go#L90)                                 | `main`                                     | 仓内消费者 |
| [foundation/apperr/example_test.go:14](../../../foundation/apperr/example_test.go#L14)                                 | `ExampleNew`                               | 单测       |
| [foundation/apperr/example_test.go:24](../../../foundation/apperr/example_test.go#L24)                                 | `ExampleNew_withCause`                     | 单测       |
| [foundation/apperr/example_test.go:39](../../../foundation/apperr/example_test.go#L39)                                 | `ExampleNew_customCode`                    | 单测       |
| [foundation/apperr/example_test.go:49](../../../foundation/apperr/example_test.go#L49)                                 | `ExampleNew_nilCode`                       | 单测       |
| [foundation/apperr/example_test.go:60](../../../foundation/apperr/example_test.go#L60)                                 | `ExampleError`                             | 单测       |
| [foundation/apperr/new_test.go:28](../../../foundation/apperr/new_test.go#L28)                                         | `TestNew_CodeSnapshot`                     | 单测       |
| [foundation/apperr/new_test.go:44](../../../foundation/apperr/new_test.go#L44)                                         | `TestNew_NilCodeDefault`                   | 单测       |
| [foundation/apperr/new_test.go:55](../../../foundation/apperr/new_test.go#L55)                                         | `TestNew_CauseHandlingAndUnwrap`           | 单测       |
| [foundation/apperr/new_test.go:60](../../../foundation/apperr/new_test.go#L60)                                         | `TestNew_CauseHandlingAndUnwrap`           | 单测       |
| [foundation/apperr/new_test.go:70](../../../foundation/apperr/new_test.go#L70)                                         | `TestNew_CauseHandlingAndUnwrap`           | 单测       |
| [foundation/apperr/new_test.go:75](../../../foundation/apperr/new_test.go#L75)                                         | `TestNew_CauseHandlingAndUnwrap`           | 单测       |
| [foundation/apperr/new_test.go:86](../../../foundation/apperr/new_test.go#L86)                                         | `TestNew_CauseHandlingAndUnwrap`           | 单测       |
| [foundation/apperr/new_test.go:100](../../../foundation/apperr/new_test.go#L100)                                       | `TestNew_JSONSerializationDoesNotLeakErr`  | 单测       |
| [foundation/apperr/new_test.go:119](../../../foundation/apperr/new_test.go#L119)                                       | `TestNew_DataAssignment`                   | 单测       |
| [foundation/apperr/new_test.go:136](../../../foundation/apperr/new_test.go#L136)                                       | `TestNew_NoLoggingSideEffect`              | 单测       |
| [plugins/cache/error_code_test.go:18](../../../plugins/cache/error_code_test.go#L18)                                   | `TestCacheErrorCode`                       | 单测       |
| [plugins/cache/error_code_test.go:27](../../../plugins/cache/error_code_test.go#L27)                                   | `ExampleCacheError`                        | 单测       |
| [plugins/database/context.go:23](../../../plugins/database/context.go#L23)                                             | `Get`                                      | SDK生产    |
| [plugins/database/error_code_test.go:34](../../../plugins/database/error_code_test.go#L34)                             | `TestDatabaseErrorCodes`                   | 单测       |
| [plugins/email/error_code_test.go:18](../../../plugins/email/error_code_test.go#L18)                                   | `TestEmailErrorCode`                       | 单测       |
| [plugins/email/error_code_test.go:27](../../../plugins/email/error_code_test.go#L27)                                   | `ExampleEmailError`                        | 单测       |
| [plugins/grpc/grpcresult/error_code_test.go:55](../../../plugins/grpc/grpcresult/error_code_test.go#L55)               | `TestProtocolErrorCodes`                   | 单测       |
| [plugins/grpc/grpcresult/example_test.go:48](../../../plugins/grpc/grpcresult/example_test.go#L48)                     | `ExampleToStatusError`                     | 单测       |
| [plugins/grpc/plugin.go:624](../../../plugins/grpc/plugin.go#L624)                                                     | `normalizeRPCError`                        | SDK生产    |
| [plugins/grpc/plugin.go:633](../../../plugins/grpc/plugin.go#L633)                                                     | `unaryBridgeInterceptor`                   | SDK生产    |
| [plugins/grpc/plugin.go:666](../../../plugins/grpc/plugin.go#L666)                                                     | `unaryBridgeInterceptor`                   | SDK生产    |
| [plugins/grpc/plugin.go:694](../../../plugins/grpc/plugin.go#L694)                                                     | `unaryBridgeInterceptor`                   | SDK生产    |
| [plugins/grpc/plugin.go:706](../../../plugins/grpc/plugin.go#L706)                                                     | `streamBridgeInterceptor`                  | SDK生产    |
| [plugins/grpc/plugin.go:738](../../../plugins/grpc/plugin.go#L738)                                                     | `streamBridgeInterceptor`                  | SDK生产    |
| [plugins/grpc/plugin_test.go:107](../../../plugins/grpc/plugin_test.go#L107)                                           | `unaryAppErrorHandler`                     | 单测       |
| [plugins/grpc/utility/grpc_util.go:29](../../../plugins/grpc/utility/grpc_util.go#L29)                                 | `ExtractMetadata`                          | SDK生产    |
| [plugins/grpc/utility/grpc_util.go:34](../../../plugins/grpc/utility/grpc_util.go#L34)                                 | `ExtractMetadata`                          | SDK生产    |
| [plugins/grpc/utility/grpc_util.go:43](../../../plugins/grpc/utility/grpc_util.go#L43)                                 | `ExtractMetadata`                          | SDK生产    |
| [plugins/web-gin/bind/binding.go:229](../../../plugins/web-gin/bind/binding.go#L229)                                   | `handleBindError`                          | SDK生产    |
| [plugins/web-gin/bind/error_code_test.go:25](../../../plugins/web-gin/bind/error_code_test.go#L25)                     | `TestProtocolErrorCodes`                   | 单测       |
| [plugins/web-gin/httpresult/error_code_test.go:62](../../../plugins/web-gin/httpresult/error_code_test.go#L62)         | `TestProtocolErrorCodes`                   | 单测       |
| [plugins/web-gin/httpresult/example_test.go:59](../../../plugins/web-gin/httpresult/example_test.go#L59)               | `ExampleError`                             | 单测       |
| [plugins/web-gin/httpresult/status_test.go:63](../../../plugins/web-gin/httpresult/status_test.go#L63)                 | `TestError_FallbackAndEnvelope`            | 单测       |
| [plugins/web-gin/httpresult/status_test.go:73](../../../plugins/web-gin/httpresult/status_test.go#L73)                 | `TestError_FallbackAndEnvelope`            | 单测       |
| [plugins/web-gin/httpresult/status_test.go:83](../../../plugins/web-gin/httpresult/status_test.go#L83)                 | `TestError_FallbackAndEnvelope`            | 单测       |
| [plugins/web-gin/httpresult/status_test.go:93](../../../plugins/web-gin/httpresult/status_test.go#L93)                 | `TestError_FallbackAndEnvelope`            | 单测       |
| [plugins/web-gin/httpresult/status_test.go:103](../../../plugins/web-gin/httpresult/status_test.go#L103)               | `TestError_FallbackAndEnvelope`            | 单测       |
| [plugins/web-gin/httpresult/status_test.go:113](../../../plugins/web-gin/httpresult/status_test.go#L113)               | `TestError_FallbackAndEnvelope`            | 单测       |
| [plugins/web-gin/httpresult/status_test.go:153](../../../plugins/web-gin/httpresult/status_test.go#L153)               | `TestError_NoUnderlyingErrLeakEvenInDebug` | 单测       |
| [plugins/web-gin/httpresult/status_test.go:185](../../../plugins/web-gin/httpresult/status_test.go#L185)               | `TestAbortError_StopsSubsequentHandlers`   | 单测       |
| [plugins/web-gin/middleware/middleware.go:51](../../../plugins/web-gin/middleware/middleware.go#L51)                   | `Recovery`                                 | SDK生产    |
| [plugins/web-gin/middleware/middleware.go:77](../../../plugins/web-gin/middleware/middleware.go#L77)                   | `Response`                                 | SDK生产    |
| [plugins/web-gin/middleware/middleware.go:86](../../../plugins/web-gin/middleware/middleware.go#L86)                   | `Response`                                 | SDK生产    |
| [plugins/web-gin/plugin.go:596](../../../plugins/web-gin/plugin.go#L596)                                               | `requestBridgeMiddleware`                  | SDK生产    |
| [plugins/web-gin/plugin.go:607](../../../plugins/web-gin/plugin.go#L607)                                               | `requestBridgeMiddleware`                  | SDK生产    |
| [plugins/web-gin/plugin_isolation_test.go:131](../../../plugins/web-gin/plugin_isolation_test.go#L131)                 | `buildApp`                                 | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:144](../../../plugins/web-gin/plugin_isolation_test.go#L144)                 | `buildApp`                                 | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:150](../../../plugins/web-gin/plugin_isolation_test.go#L150)                 | `buildApp`                                 | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:156](../../../plugins/web-gin/plugin_isolation_test.go#L156)                 | `buildApp`                                 | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:160](../../../plugins/web-gin/plugin_isolation_test.go#L160)                 | `buildApp`                                 | 单测       |
| [plugins/web-gin/plugin_test.go:208](../../../plugins/web-gin/plugin_test.go#L208)                                     | `TestWebGin_FullNetworkLifecycle`          | 单测       |
| [plugins/web-gin/validator/error.go:65](../../../plugins/web-gin/validator/error.go#L65)                               | `handleErrorInternal`                      | SDK生产    |
| [plugins/web-gin/validator/error_code_test.go:26](../../../plugins/web-gin/validator/error_code_test.go#L26)           | `TestProtocolErrorCodes`                   | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:104](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L104) | `TestHTTPDBCacheLifecycle_EndToEnd`        | 仓内消费者 |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:110](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L110) | `TestHTTPDBCacheLifecycle_EndToEnd`        | 仓内消费者 |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:117](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L117) | `TestHTTPDBCacheLifecycle_EndToEnd`        | 仓内消费者 |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:123](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L123) | `TestHTTPDBCacheLifecycle_EndToEnd`        | 仓内消费者 |

#### P0124

- 定义：[foundation/apperr/new.go:27](../../../foundation/apperr/new.go#L27) `New.nonNilErrs`；分类N；阶段P1。
- 完整当前类型：`[]error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [foundation/apperr/new.go:30](../../../foundation/apperr/new.go#L30) | `New`         | SDK生产  |
| [foundation/apperr/new.go:30](../../../foundation/apperr/new.go#L30) | `New`         | SDK生产  |
| [foundation/apperr/new.go:35](../../../foundation/apperr/new.go#L35) | `New`         | SDK生产  |
| [foundation/apperr/new.go:39](../../../foundation/apperr/new.go#L39) | `New`         | SDK生产  |
| [foundation/apperr/new.go:41](../../../foundation/apperr/new.go#L41) | `New`         | SDK生产  |

#### P0125

- 定义：[foundation/ctxkey/key.go:72](../../../foundation/ctxkey/key.go#L72) `Key.MarshalText`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding.TextMarshaler`。

| 位置                                                                             | 所在函数/声明                         | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [foundation/ctxkey/key_test.go:72](../../../foundation/ctxkey/key_test.go#L72)   | `TestKey_SixChannels_RoundTrip`       | 单测     |
| [foundation/ctxkey/key_test.go:268](../../../foundation/ctxkey/key_test.go#L268) | `TestKey_MarshalInvalid_ReturnsError` | 单测     |

#### P0126

- 定义：[foundation/ctxkey/key.go:79](../../../foundation/ctxkey/key.go#L79) `Key.UnmarshalText`；分类F；阶段P1/P2。
- 完整当前类型：`func(text []byte) error`。
- 方法集：`encoding.TextUnmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                                   | 所在函数/声明                            | 引用来源 |
| -------------------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [foundation/ctxkey/example_test.go:36](../../../foundation/ctxkey/example_test.go#L36) | `ExampleKey`                             | 单测     |
| [foundation/ctxkey/key_test.go:77](../../../foundation/ctxkey/key_test.go#L77)         | `TestKey_SixChannels_RoundTrip`          | 单测     |
| [foundation/ctxkey/key_test.go:157](../../../foundation/ctxkey/key_test.go#L157)       | `TestKey_NilReceiver_Safety`             | 单测     |
| [foundation/ctxkey/key_test.go:185](../../../foundation/ctxkey/key_test.go#L185)       | `TestKey_InvalidInput_PreservesOriginal` | 单测     |

#### P0127

- 定义：[foundation/ctxkey/key.go:94](../../../foundation/ctxkey/key.go#L94) `Key.MarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding.BinaryMarshaler`。

| 位置                                                                             | 所在函数/声明                         | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [foundation/ctxkey/key_test.go:271](../../../foundation/ctxkey/key_test.go#L271) | `TestKey_MarshalInvalid_ReturnsError` | 单测     |

#### P0128

- 定义：[foundation/ctxkey/key.go:101](../../../foundation/ctxkey/key.go#L101) `Key.UnmarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func(data []byte) error`。
- 方法集：`encoding.BinaryUnmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                             | 所在函数/声明                            | 引用来源 |
| -------------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [foundation/ctxkey/key_test.go:160](../../../foundation/ctxkey/key_test.go#L160) | `TestKey_NilReceiver_Safety`             | 单测     |
| [foundation/ctxkey/key_test.go:194](../../../foundation/ctxkey/key_test.go#L194) | `TestKey_InvalidInput_PreservesOriginal` | 单测     |

#### P0129

- 定义：[foundation/ctxkey/key.go:115](../../../foundation/ctxkey/key.go#L115) `Key.MarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding/json.Marshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0130

- 定义：[foundation/ctxkey/key.go:122](../../../foundation/ctxkey/key.go#L122) `Key.UnmarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func(raw []byte) error`。
- 方法集：`encoding/json.Unmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                             | 所在函数/声明                | 引用来源 |
| -------------------------------------------------------------------------------- | ---------------------------- | -------- |
| [foundation/ctxkey/key_test.go:163](../../../foundation/ctxkey/key_test.go#L163) | `TestKey_NilReceiver_Safety` | 单测     |

#### P0131

- 定义：[foundation/ctxkey/key.go:136](../../../foundation/ctxkey/key.go#L136) `Key.MarshalXML`；分类F；阶段P1/P2。
- 完整当前类型：`func(e *encoding/xml.Encoder, start encoding/xml.StartElement) error`。
- 方法集：`encoding/xml.Marshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0132

- 定义：[foundation/ctxkey/key.go:143](../../../foundation/ctxkey/key.go#L143) `Key.UnmarshalXML`；分类F；阶段P1/P2。
- 完整当前类型：`func(d *encoding/xml.Decoder, start encoding/xml.StartElement) error`。
- 方法集：`encoding/xml.Unmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                             | 所在函数/声明                | 引用来源 |
| -------------------------------------------------------------------------------- | ---------------------------- | -------- |
| [foundation/ctxkey/key_test.go:166](../../../foundation/ctxkey/key_test.go#L166) | `TestKey_NilReceiver_Safety` | 单测     |

#### P0133

- 定义：[foundation/ctxkey/key.go:157](../../../foundation/ctxkey/key.go#L157) `Key.MarshalXMLAttr`；分类F；阶段P1/P2。
- 完整当前类型：`func(name encoding/xml.Name) (encoding/xml.Attr, error)`。
- 方法集：`encoding/xml.MarshalerAttr`。

| 位置                                                                             | 所在函数/声明                         | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [foundation/ctxkey/key_test.go:277](../../../foundation/ctxkey/key_test.go#L277) | `TestKey_MarshalInvalid_ReturnsError` | 单测     |

#### P0134

- 定义：[foundation/ctxkey/key.go:164](../../../foundation/ctxkey/key.go#L164) `Key.UnmarshalXMLAttr`；分类F；阶段P1/P2。
- 完整当前类型：`func(attr encoding/xml.Attr) error`。
- 方法集：`encoding/xml.UnmarshalerAttr`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                             | 所在函数/声明                | 引用来源 |
| -------------------------------------------------------------------------------- | ---------------------------- | -------- |
| [foundation/ctxkey/key_test.go:169](../../../foundation/ctxkey/key_test.go#L169) | `TestKey_NilReceiver_Safety` | 单测     |

#### P0135

- 定义：[foundation/ctxkey/key.go:178](../../../foundation/ctxkey/key.go#L178) `Key.Value`；分类F；阶段P1/P2。
- 完整当前类型：`func() (database/sql/driver.Value, error)`。
- 方法集：`database/sql/driver.Valuer`。

| 位置                                                                             | 所在函数/声明                         | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [foundation/ctxkey/key_test.go:132](../../../foundation/ctxkey/key_test.go#L132) | `TestKey_SixChannels_RoundTrip`       | 单测     |
| [foundation/ctxkey/key_test.go:280](../../../foundation/ctxkey/key_test.go#L280) | `TestKey_MarshalInvalid_ReturnsError` | 单测     |

#### P0136

- 定义：[foundation/ctxkey/key.go:185](../../../foundation/ctxkey/key.go#L185) `Key.Scan`；分类F；阶段P1/P2。
- 完整当前类型：`func(src any) error`。
- 方法集：`database/sql.Scanner`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                             | 所在函数/声明                            | 引用来源 |
| -------------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [foundation/ctxkey/key_test.go:137](../../../foundation/ctxkey/key_test.go#L137) | `TestKey_SixChannels_RoundTrip`          | 单测     |
| [foundation/ctxkey/key_test.go:172](../../../foundation/ctxkey/key_test.go#L172) | `TestKey_NilReceiver_Safety`             | 单测     |
| [foundation/ctxkey/key_test.go:242](../../../foundation/ctxkey/key_test.go#L242) | `TestKey_InvalidInput_PreservesOriginal` | 单测     |
| [foundation/ctxkey/key_test.go:248](../../../foundation/ctxkey/key_test.go#L248) | `TestKey_InvalidInput_PreservesOriginal` | 单测     |

#### P0137

- 定义：[foundation/ctxkey/key.go:200](../../../foundation/ctxkey/key.go#L200) `Key.Set`；分类F；阶段P1/P2。
- 完整当前类型：`func(raw string) error`。
- 方法集：`flag.Value`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                             | 所在函数/声明                            | 引用来源 |
| -------------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [foundation/ctxkey/key_test.go:146](../../../foundation/ctxkey/key_test.go#L146) | `TestKey_SixChannels_RoundTrip`          | 单测     |
| [foundation/ctxkey/key_test.go:175](../../../foundation/ctxkey/key_test.go#L175) | `TestKey_NilReceiver_Safety`             | 单测     |
| [foundation/ctxkey/key_test.go:257](../../../foundation/ctxkey/key_test.go#L257) | `TestKey_InvalidInput_PreservesOriginal` | 单测     |

#### P0138

- 定义：[foundation/env/key.go:66](../../../foundation/env/key.go#L66) `Key.MarshalText`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding.TextMarshaler`。

| 位置                                                                       | 所在函数/声明                         | 引用来源 |
| -------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [foundation/env/key_test.go:145](../../../foundation/env/key_test.go#L145) | `TestKey_SixChannels_RoundTrip`       | 单测     |
| [foundation/env/key_test.go:341](../../../foundation/env/key_test.go#L341) | `TestKey_MarshalInvalid_ReturnsError` | 单测     |

#### P0139

- 定义：[foundation/env/key.go:74](../../../foundation/env/key.go#L74) `Key.UnmarshalText`；分类F；阶段P1/P2。
- 完整当前类型：`func(text []byte) error`。
- 方法集：`encoding.TextUnmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                             | 所在函数/声明                            | 引用来源 |
| -------------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [foundation/env/example_test.go:35](../../../foundation/env/example_test.go#L35) | `ExampleKey`                             | 单测     |
| [foundation/env/key_test.go:150](../../../foundation/env/key_test.go#L150)       | `TestKey_SixChannels_RoundTrip`          | 单测     |
| [foundation/env/key_test.go:230](../../../foundation/env/key_test.go#L230)       | `TestKey_NilReceiver_Safety`             | 单测     |
| [foundation/env/key_test.go:258](../../../foundation/env/key_test.go#L258)       | `TestKey_InvalidInput_PreservesOriginal` | 单测     |

#### P0140

- 定义：[foundation/env/key.go:89](../../../foundation/env/key.go#L89) `Key.MarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding.BinaryMarshaler`。

| 位置                                                                       | 所在函数/声明                         | 引用来源 |
| -------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [foundation/env/key_test.go:344](../../../foundation/env/key_test.go#L344) | `TestKey_MarshalInvalid_ReturnsError` | 单测     |

#### P0141

- 定义：[foundation/env/key.go:97](../../../foundation/env/key.go#L97) `Key.UnmarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func(data []byte) error`。
- 方法集：`encoding.BinaryUnmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                       | 所在函数/声明                            | 引用来源 |
| -------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [foundation/env/key_test.go:233](../../../foundation/env/key_test.go#L233) | `TestKey_NilReceiver_Safety`             | 单测     |
| [foundation/env/key_test.go:267](../../../foundation/env/key_test.go#L267) | `TestKey_InvalidInput_PreservesOriginal` | 单测     |

#### P0142

- 定义：[foundation/env/key.go:112](../../../foundation/env/key.go#L112) `Key.MarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding/json.Marshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0143

- 定义：[foundation/env/key.go:120](../../../foundation/env/key.go#L120) `Key.UnmarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func(raw []byte) error`。
- 方法集：`encoding/json.Unmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                       | 所在函数/声明                | 引用来源 |
| -------------------------------------------------------------------------- | ---------------------------- | -------- |
| [foundation/env/key_test.go:236](../../../foundation/env/key_test.go#L236) | `TestKey_NilReceiver_Safety` | 单测     |

#### P0144

- 定义：[foundation/env/key.go:135](../../../foundation/env/key.go#L135) `Key.MarshalXML`；分类F；阶段P1/P2。
- 完整当前类型：`func(e *encoding/xml.Encoder, start encoding/xml.StartElement) error`。
- 方法集：`encoding/xml.Marshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0145

- 定义：[foundation/env/key.go:143](../../../foundation/env/key.go#L143) `Key.UnmarshalXML`；分类F；阶段P1/P2。
- 完整当前类型：`func(d *encoding/xml.Decoder, start encoding/xml.StartElement) error`。
- 方法集：`encoding/xml.Unmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                       | 所在函数/声明                | 引用来源 |
| -------------------------------------------------------------------------- | ---------------------------- | -------- |
| [foundation/env/key_test.go:239](../../../foundation/env/key_test.go#L239) | `TestKey_NilReceiver_Safety` | 单测     |

#### P0146

- 定义：[foundation/env/key.go:158](../../../foundation/env/key.go#L158) `Key.MarshalXMLAttr`；分类F；阶段P1/P2。
- 完整当前类型：`func(name encoding/xml.Name) (encoding/xml.Attr, error)`。
- 方法集：`encoding/xml.MarshalerAttr`。

| 位置                                                                       | 所在函数/声明                         | 引用来源 |
| -------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [foundation/env/key_test.go:350](../../../foundation/env/key_test.go#L350) | `TestKey_MarshalInvalid_ReturnsError` | 单测     |

#### P0147

- 定义：[foundation/env/key.go:166](../../../foundation/env/key.go#L166) `Key.UnmarshalXMLAttr`；分类F；阶段P1/P2。
- 完整当前类型：`func(attr encoding/xml.Attr) error`。
- 方法集：`encoding/xml.UnmarshalerAttr`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                       | 所在函数/声明                | 引用来源 |
| -------------------------------------------------------------------------- | ---------------------------- | -------- |
| [foundation/env/key_test.go:242](../../../foundation/env/key_test.go#L242) | `TestKey_NilReceiver_Safety` | 单测     |

#### P0148

- 定义：[foundation/env/key.go:181](../../../foundation/env/key.go#L181) `Key.Value`；分类F；阶段P1/P2。
- 完整当前类型：`func() (database/sql/driver.Value, error)`。
- 方法集：`database/sql/driver.Valuer`。

| 位置                                                                       | 所在函数/声明                         | 引用来源 |
| -------------------------------------------------------------------------- | ------------------------------------- | -------- |
| [foundation/env/key_test.go:205](../../../foundation/env/key_test.go#L205) | `TestKey_SixChannels_RoundTrip`       | 单测     |
| [foundation/env/key_test.go:353](../../../foundation/env/key_test.go#L353) | `TestKey_MarshalInvalid_ReturnsError` | 单测     |

#### P0149

- 定义：[foundation/env/key.go:189](../../../foundation/env/key.go#L189) `Key.Scan`；分类F；阶段P1/P2。
- 完整当前类型：`func(src any) error`。
- 方法集：`database/sql.Scanner`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                       | 所在函数/声明                            | 引用来源 |
| -------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [foundation/env/key_test.go:210](../../../foundation/env/key_test.go#L210) | `TestKey_SixChannels_RoundTrip`          | 单测     |
| [foundation/env/key_test.go:245](../../../foundation/env/key_test.go#L245) | `TestKey_NilReceiver_Safety`             | 单测     |
| [foundation/env/key_test.go:315](../../../foundation/env/key_test.go#L315) | `TestKey_InvalidInput_PreservesOriginal` | 单测     |
| [foundation/env/key_test.go:321](../../../foundation/env/key_test.go#L321) | `TestKey_InvalidInput_PreservesOriginal` | 单测     |

#### P0150

- 定义：[foundation/env/key.go:205](../../../foundation/env/key.go#L205) `Key.Set`；分类F；阶段P1/P2。
- 完整当前类型：`func(raw string) error`。
- 方法集：`flag.Value`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                       | 所在函数/声明                            | 引用来源 |
| -------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [foundation/env/key_test.go:219](../../../foundation/env/key_test.go#L219) | `TestKey_SixChannels_RoundTrip`          | 单测     |
| [foundation/env/key_test.go:248](../../../foundation/env/key_test.go#L248) | `TestKey_NilReceiver_Safety`             | 单测     |
| [foundation/env/key_test.go:330](../../../foundation/env/key_test.go#L330) | `TestKey_InvalidInput_PreservesOriginal` | 单测     |

#### P0151

- 定义：[foundation/logging/command.go:25](../../../foundation/logging/command.go#L25) `New`；分类S；阶段P1/P2。
- 完整当前类型：`func(handler log/slog.Handler) (*github.com/bamboo-services/bamboo-base-go/foundation/v2/logging.Logger, error)`。

| 位置                                                                                       | 所在函数/声明                      | 引用来源 |
| ------------------------------------------------------------------------------------------ | ---------------------------------- | -------- |
| [application/builtin/logging.go:118](../../../application/builtin/logging.go#L118)         | `Start`                            | SDK生产  |
| [foundation/logging/example_test.go:24](../../../foundation/logging/example_test.go#L24)   | `ExampleNewLogHandler`             | 单测     |
| [foundation/logging/handler_test.go:26](../../../foundation/logging/handler_test.go#L26)   | `TestNew_RejectsNilHandler`        | 单测     |
| [foundation/logging/handler_test.go:43](../../../foundation/logging/handler_test.go#L43)   | `TestSlogDefault_RemainsUnchanged` | 单测     |
| [foundation/logging/handler_test.go:68](../../../foundation/logging/handler_test.go#L68)   | `TestTwoLoggers_OutputIsolation`   | 单测     |
| [foundation/logging/handler_test.go:72](../../../foundation/logging/handler_test.go#L72)   | `TestTwoLoggers_OutputIsolation`   | 单测     |
| [foundation/logging/handler_test.go:102](../../../foundation/logging/handler_test.go#L102) | `TestTraceID_Extraction`           | 单测     |
| [foundation/logging/handler_test.go:142](../../../foundation/logging/handler_test.go#L142) | `TestDualWrite_AndLevels`          | 单测     |
| [foundation/logging/handler_test.go:189](../../../foundation/logging/handler_test.go#L189) | `TestSugarMethods_AndNotice`       | 单测     |
| [foundation/logging/handler_test.go:217](../../../foundation/logging/handler_test.go#L217) | `TestPanicMethods_TriggerPanic`    | 单测     |
| [plugins/cache/plugin_test.go:326](../../../plugins/cache/plugin_test.go#L326)             | `TestWithLogger`                   | 单测     |
| [plugins/email/client_test.go:632](../../../plugins/email/client_test.go#L632)             | `TestClient_WithLogger`            | 单测     |

#### P0152

- 定义：[foundation/logging/handler.go:85](../../../foundation/logging/handler.go#L85) `LogHandler.Handle`；分类F；阶段P1/P2。
- 完整当前类型：`func(ctx context.Context, r log/slog.Record) error`。
- 方法集：`log/slog.Handler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0153

- 定义：[foundation/logging/handler.go:92](../../../foundation/logging/handler.go#L92) `Handle.errs`；分类S；阶段P1/P2。
- 完整当前类型：`[]error`。

| 位置                                                                             | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------- | ------------- | -------- |
| [foundation/logging/handler.go:97](../../../foundation/logging/handler.go#L97)   | `Handle`      | SDK生产  |
| [foundation/logging/handler.go:97](../../../foundation/logging/handler.go#L97)   | `Handle`      | SDK生产  |
| [foundation/logging/handler.go:104](../../../foundation/logging/handler.go#L104) | `Handle`      | SDK生产  |
| [foundation/logging/handler.go:104](../../../foundation/logging/handler.go#L104) | `Handle`      | SDK生产  |
| [foundation/logging/handler.go:108](../../../foundation/logging/handler.go#L108) | `Handle`      | SDK生产  |

#### P0154

- 定义：[foundation/logging/handler.go:174](../../../foundation/logging/handler.go#L174) `LogHandler.writeConsole`；分类S；阶段P1/P2。
- 完整当前类型：`func(r log/slog.Record, contextUUID string) error`。

| 位置                                                                           | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------------ | ------------- | -------- |
| [foundation/logging/handler.go:96](../../../foundation/logging/handler.go#L96) | `Handle`      | SDK生产  |

#### P0155

- 定义：[foundation/logging/handler.go:269](../../../foundation/logging/handler.go#L269) `LogHandler.writeFile`；分类S；阶段P1/P2。
- 完整当前类型：`func(r log/slog.Record, trace string) error`。

| 位置                                                                             | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------- | ------------- | -------- |
| [foundation/logging/handler.go:103](../../../foundation/logging/handler.go#L103) | `Handle`      | SDK生产  |

#### P0156

- 定义：[foundation/logging/rotator.go:23](../../../foundation/logging/rotator.go#L23) `ErrClosed`；分类N；阶段P1/P2。
- 完整当前类型：`error`。

| 位置                                                                                         | 所在函数/声明                                | 引用来源 |
| -------------------------------------------------------------------------------------------- | -------------------------------------------- | -------- |
| [application/builtin/builtin_test.go:167](../../../application/builtin/builtin_test.go#L167) | `TestBuiltinLogger_RealLogCloseAndContent`   | 单测     |
| [foundation/logging/rotator.go:129](../../../foundation/logging/rotator.go#L129)             | `Write`                                      | SDK生产  |
| [foundation/logging/rotator.go:140](../../../foundation/logging/rotator.go#L140)             | `Write`                                      | SDK生产  |
| [foundation/logging/rotator.go:158](../../../foundation/logging/rotator.go#L158)             | `Write`                                      | SDK生产  |
| [foundation/logging/rotator_test.go:181](../../../foundation/logging/rotator_test.go#L181)   | `TestRotatingWriter_ConcurrentWriteAndClose` | 单测     |
| [foundation/logging/rotator_test.go:208](../../../foundation/logging/rotator_test.go#L208)   | `TestRotatingWriter_ConcurrentWriteAndClose` | 单测     |

#### P0157

- 定义：[foundation/logging/rotator.go:61](../../../foundation/logging/rotator.go#L61) `RotatingWriter.closeErr`；分类S；阶段P1/P2。
- 完整当前类型：`error`。

| 位置                                                                             | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------- | ------------- | -------- |
| [foundation/logging/rotator.go:179](../../../foundation/logging/rotator.go#L179) | `Close`       | SDK生产  |
| [foundation/logging/rotator.go:209](../../../foundation/logging/rotator.go#L209) | `Close`       | SDK生产  |
| [foundation/logging/rotator.go:211](../../../foundation/logging/rotator.go#L211) | `Close`       | SDK生产  |

#### P0158

- 定义：[foundation/logging/rotator.go:68](../../../foundation/logging/rotator.go#L68) `RotatingWriter.bgErrors`；分类S；阶段P1/P2。
- 完整当前类型：`[]error`。

| 位置                                                                             | 所在函数/声明   | 引用来源 |
| -------------------------------------------------------------------------------- | --------------- | -------- |
| [foundation/logging/rotator.go:208](../../../foundation/logging/rotator.go#L208) | `Close`         | SDK生产  |
| [foundation/logging/rotator.go:568](../../../foundation/logging/rotator.go#L568) | `recordBgError` | SDK生产  |
| [foundation/logging/rotator.go:568](../../../foundation/logging/rotator.go#L568) | `recordBgError` | SDK生产  |

#### P0159

- 定义：[foundation/logging/rotator.go:81](../../../foundation/logging/rotator.go#L81) `NewRotatingWriter`；分类S；阶段P1/P2。
- 完整当前类型：`func(config github.com/bamboo-services/bamboo-base-go/foundation/v2/logging.RotatorConfig) (*github.com/bamboo-services/bamboo-base-go/foundation/v2/logging.RotatingWriter, error)`。

| 位置                                                                                       | 所在函数/声明                                     | 引用来源 |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------- | -------- |
| [application/builtin/logging.go:106](../../../application/builtin/logging.go#L106)         | `Start`                                           | SDK生产  |
| [foundation/logging/example_test.go:50](../../../foundation/logging/example_test.go#L50)   | `ExampleNewRotatingWriter`                        | 单测     |
| [foundation/logging/rotator_test.go:18](../../../foundation/logging/rotator_test.go#L18)   | `TestRotatingWriter_BasicWriteAndContent`         | 单测     |
| [foundation/logging/rotator_test.go:49](../../../foundation/logging/rotator_test.go#L49)   | `TestRotatingWriter_SizeRotation`                 | 单测     |
| [foundation/logging/rotator_test.go:92](../../../foundation/logging/rotator_test.go#L92)   | `TestRotatingWriter_ArchiveTarGzContent`          | 单测     |
| [foundation/logging/rotator_test.go:155](../../../foundation/logging/rotator_test.go#L155) | `TestRotatingWriter_ConcurrentWriteAndClose`      | 单测     |
| [foundation/logging/rotator_test.go:226](../../../foundation/logging/rotator_test.go#L226) | `TestRotatingWriter_CloseWaitsForArchivalBarrier` | 单测     |

#### P0160

- 定义：[foundation/logging/rotator.go:125](../../../foundation/logging/rotator.go#L125) `RotatingWriter.Write`；分类F；阶段P1/P2。
- 完整当前类型：`func(p []byte) (n int, err error)`。
- 方法集：`io.Writer`。

| 位置                                                                                         | 所在函数/声明                                | 引用来源 |
| -------------------------------------------------------------------------------------------- | -------------------------------------------- | -------- |
| [application/builtin/builtin_test.go:167](../../../application/builtin/builtin_test.go#L167) | `TestBuiltinLogger_RealLogCloseAndContent`   | 单测     |
| [foundation/logging/example_test.go:60](../../../foundation/logging/example_test.go#L60)     | `ExampleNewRotatingWriter`                   | 单测     |
| [foundation/logging/rotator_test.go:29](../../../foundation/logging/rotator_test.go#L29)     | `TestRotatingWriter_BasicWriteAndContent`    | 单测     |
| [foundation/logging/rotator_test.go:61](../../../foundation/logging/rotator_test.go#L61)     | `TestRotatingWriter_SizeRotation`            | 单测     |
| [foundation/logging/rotator_test.go:65](../../../foundation/logging/rotator_test.go#L65)     | `TestRotatingWriter_SizeRotation`            | 单测     |
| [foundation/logging/rotator_test.go:179](../../../foundation/logging/rotator_test.go#L179)   | `TestRotatingWriter_ConcurrentWriteAndClose` | 单测     |
| [foundation/logging/rotator_test.go:207](../../../foundation/logging/rotator_test.go#L207)   | `TestRotatingWriter_ConcurrentWriteAndClose` | 单测     |

#### P0161

- 定义：[foundation/logging/rotator.go:172](../../../foundation/logging/rotator.go#L172) `RotatingWriter.Close`；分类F；阶段P1/P2。
- 完整当前类型：`func() error`。
- 方法集：`io.Closer`。

| 位置                                                                                       | 所在函数/声明                                     | 引用来源 |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------- | -------- |
| [application/builtin/logging.go:120](../../../application/builtin/logging.go#L120)         | `Start`                                           | SDK生产  |
| [application/builtin/logging.go:139](../../../application/builtin/logging.go#L139)         | `Stop`                                            | SDK生产  |
| [foundation/logging/example_test.go:64](../../../foundation/logging/example_test.go#L64)   | `ExampleNewRotatingWriter`                        | 单测     |
| [foundation/logging/rotator_test.go:34](../../../foundation/logging/rotator_test.go#L34)   | `TestRotatingWriter_BasicWriteAndContent`         | 单测     |
| [foundation/logging/rotator_test.go:69](../../../foundation/logging/rotator_test.go#L69)   | `TestRotatingWriter_SizeRotation`                 | 单测     |
| [foundation/logging/rotator_test.go:105](../../../foundation/logging/rotator_test.go#L105) | `TestRotatingWriter_ArchiveTarGzContent`          | 单测     |
| [foundation/logging/rotator_test.go:199](../../../foundation/logging/rotator_test.go#L199) | `TestRotatingWriter_ConcurrentWriteAndClose`      | 单测     |
| [foundation/logging/rotator_test.go:213](../../../foundation/logging/rotator_test.go#L213) | `TestRotatingWriter_ConcurrentWriteAndClose`      | 单测     |
| [foundation/logging/rotator_test.go:254](../../../foundation/logging/rotator_test.go#L254) | `TestRotatingWriter_CloseWaitsForArchivalBarrier` | 单测     |
| [foundation/logging/rotator_test.go:268](../../../foundation/logging/rotator_test.go#L268) | `TestRotatingWriter_CloseWaitsForArchivalBarrier` | 单测     |

#### P0162

- 定义：[foundation/logging/rotator.go:204](../../../foundation/logging/rotator.go#L204) `Close.allErrs`；分类S；阶段P1/P2。
- 完整当前类型：`[]error`。

| 位置                                                                             | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------- | ------------- | -------- |
| [foundation/logging/rotator.go:206](../../../foundation/logging/rotator.go#L206) | `Close`       | SDK生产  |
| [foundation/logging/rotator.go:206](../../../foundation/logging/rotator.go#L206) | `Close`       | SDK生产  |
| [foundation/logging/rotator.go:208](../../../foundation/logging/rotator.go#L208) | `Close`       | SDK生产  |
| [foundation/logging/rotator.go:208](../../../foundation/logging/rotator.go#L208) | `Close`       | SDK生产  |
| [foundation/logging/rotator.go:209](../../../foundation/logging/rotator.go#L209) | `Close`       | SDK生产  |

#### P0163

- 定义：[foundation/logging/rotator.go:217](../../../foundation/logging/rotator.go#L217) `RotatingWriter.openFileLocked`；分类S；阶段P1/P2。
- 完整当前类型：`func() error`。

| 位置                                                                             | 所在函数/声明           | 引用来源 |
| -------------------------------------------------------------------------------- | ----------------------- | -------- |
| [foundation/logging/rotator.go:304](../../../foundation/logging/rotator.go#L304) | `rotateLocked`          | SDK生产  |
| [foundation/logging/rotator.go:337](../../../foundation/logging/rotator.go#L337) | `rotateForNewDayLocked` | SDK生产  |
| [foundation/logging/rotator.go:525](../../../foundation/logging/rotator.go#L525) | `checkStartupState`     | SDK生产  |
| [foundation/logging/rotator.go:542](../../../foundation/logging/rotator.go#L542) | `checkStartupState`     | SDK生产  |
| [foundation/logging/rotator.go:551](../../../foundation/logging/rotator.go#L551) | `checkStartupState`     | SDK生产  |

#### P0164

- 定义：[foundation/logging/rotator.go:251](../../../foundation/logging/rotator.go#L251) `RotatingWriter.renameCurrentLogByDateLocked`；分类S；阶段P1/P2。
- 完整当前类型：`func(date string) error`。

| 位置                                                                             | 所在函数/声明       | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------- | -------- |
| [foundation/logging/rotator.go:539](../../../foundation/logging/rotator.go#L539) | `checkStartupState` | SDK生产  |

#### P0165

- 定义：[foundation/logging/rotator.go:285](../../../foundation/logging/rotator.go#L285) `RotatingWriter.rotateLocked`；分类S；阶段P1/P2。
- 完整当前类型：`func() error`。

| 位置                                                                             | 所在函数/声明       | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------- | -------- |
| [foundation/logging/rotator.go:152](../../../foundation/logging/rotator.go#L152) | `Write`             | SDK生产  |
| [foundation/logging/rotator.go:547](../../../foundation/logging/rotator.go#L547) | `checkStartupState` | SDK生产  |

#### P0166

- 定义：[foundation/logging/rotator.go:307](../../../foundation/logging/rotator.go#L307) `RotatingWriter.rotateForNewDayLocked`；分类S；阶段P1/P2。
- 完整当前类型：`func(newDate string) error`。

| 位置                                                                             | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------- | ------------- | -------- |
| [foundation/logging/rotator.go:146](../../../foundation/logging/rotator.go#L146) | `Write`       | SDK生产  |

#### P0167

- 定义：[foundation/logging/rotator.go:451](../../../foundation/logging/rotator.go#L451) `RotatingWriter.createTarGz`；分类S；阶段P1/P2。
- 完整当前类型：`func(archivePath string, files []string) (err error)`。

| 位置                                                                             | 所在函数/声明      | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------ | -------- |
| [foundation/logging/rotator.go:420](../../../foundation/logging/rotator.go#L420) | `archiveYesterday` | SDK生产  |
| [foundation/logging/rotator.go:621](../../../foundation/logging/rotator.go#L621) | `archiveOldFiles`  | SDK生产  |

#### P0168

- 定义：[foundation/logging/rotator.go:488](../../../foundation/logging/rotator.go#L488) `RotatingWriter.addFileToTar`；分类S；阶段P1/P2。
- 完整当前类型：`func(tarWriter *archive/tar.Writer, filePath string) (err error)`。

| 位置                                                                             | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------- | ------------- | -------- |
| [foundation/logging/rotator.go:480](../../../foundation/logging/rotator.go#L480) | `createTarGz` | SDK生产  |

#### P0169

- 定义：[foundation/logging/rotator.go:518](../../../foundation/logging/rotator.go#L518) `RotatingWriter.checkStartupState`；分类S；阶段P1/P2。
- 完整当前类型：`func() error`。

| 位置                                                                             | 所在函数/声明       | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------- | -------- |
| [foundation/logging/rotator.go:110](../../../foundation/logging/rotator.go#L110) | `NewRotatingWriter` | SDK生产  |

#### P0170

- 定义：[foundation/logging/rotator.go:563](../../../foundation/logging/rotator.go#L563) `RotatingWriter.recordBgError`；分类S；阶段P1/P2。
- 完整当前类型：`func(err error)`。

| 位置                                                                             | 所在函数/声明       | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------- | -------- |
| [foundation/logging/rotator.go:421](../../../foundation/logging/rotator.go#L421) | `archiveYesterday`  | SDK生产  |
| [foundation/logging/rotator.go:558](../../../foundation/logging/rotator.go#L558) | `checkStartupState` | SDK生产  |
| [foundation/logging/rotator.go:622](../../../foundation/logging/rotator.go#L622) | `archiveOldFiles`   | SDK生产  |

#### P0171

- 定义：[foundation/logging/rotator.go:572](../../../foundation/logging/rotator.go#L572) `RotatingWriter.archiveOldFiles`；分类S；阶段P1/P2。
- 完整当前类型：`func() error`。

| 位置                                                                             | 所在函数/声明       | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------- | -------- |
| [foundation/logging/rotator.go:557](../../../foundation/logging/rotator.go#L557) | `checkStartupState` | SDK生产  |

#### P0172

- 定义：[foundation/snowflake/snowflake.go:41](../../../foundation/snowflake/snowflake.go#L41) `ErrNilNode`；分类N；阶段P1/P2。
- 完整当前类型：`error`。

| 位置                                                                                               | 所在函数/声明           | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ----------------------- | -------- |
| [foundation/snowflake/snowflake.go:96](../../../foundation/snowflake/snowflake.go#L96)             | `Generate`              | SDK生产  |
| [foundation/snowflake/snowflake_test.go:171](../../../foundation/snowflake/snowflake_test.go#L171) | `Test_NilNode_Generate` | 单测     |

#### P0173

- 定义：[foundation/snowflake/snowflake.go:42](../../../foundation/snowflake/snowflake.go#L42) `ErrNilReceiver`；分类N；阶段P1/P2。
- 完整当前类型：`error`。

| 位置                                                                                               | 所在函数/声明                     | 引用来源 |
| -------------------------------------------------------------------------------------------------- | --------------------------------- | -------- |
| [foundation/snowflake/snowflake.go:206](../../../foundation/snowflake/snowflake.go#L206)           | `UnmarshalJSON`                   | SDK生产  |
| [foundation/snowflake/snowflake.go:238](../../../foundation/snowflake/snowflake.go#L238)           | `UnmarshalBinary`                 | SDK生产  |
| [foundation/snowflake/snowflake.go:262](../../../foundation/snowflake/snowflake.go#L262)           | `Scan`                            | SDK生产  |
| [foundation/snowflake/snowflake_test.go:271](../../../foundation/snowflake/snowflake_test.go#L271) | `Test_SnowflakeID_JSON`           | 单测     |
| [foundation/snowflake/snowflake_test.go:304](../../../foundation/snowflake/snowflake_test.go#L304) | `Test_SnowflakeID_Binary`         | 单测     |
| [foundation/snowflake/snowflake_test.go:352](../../../foundation/snowflake/snowflake_test.go#L352) | `Test_SnowflakeID_SQL_Scan_Value` | 单测     |

#### P0174

- 定义：[foundation/snowflake/snowflake.go:43](../../../foundation/snowflake/snowflake.go#L43) `ErrClockBackwards`；分类N；阶段P1/P2。
- 完整当前类型：`error`。

| 位置                                                                                               | 所在函数/声明                                             | 引用来源 |
| -------------------------------------------------------------------------------------------------- | --------------------------------------------------------- | -------- |
| [foundation/snowflake/snowflake.go:114](../../../foundation/snowflake/snowflake.go#L114)           | `Generate`                                                | SDK生产  |
| [foundation/snowflake/snowflake.go:125](../../../foundation/snowflake/snowflake.go#L125)           | `Generate`                                                | SDK生产  |
| [foundation/snowflake/snowflake_test.go:235](../../../foundation/snowflake/snowflake_test.go#L235) | `Test_ClockBackwards_Error`                               | 单测     |
| [foundation/snowflake/snowflake_test.go:412](../../../foundation/snowflake/snowflake_test.go#L412) | `Test_SnowflakeID_SequenceOverflowClockBackwardsRollback` | 单测     |
| [foundation/snowflake/snowflake_test.go:436](../../../foundation/snowflake/snowflake_test.go#L436) | `Test_SnowflakeID_SequenceOverflowClockBackwardsRollback` | 单测     |

#### P0175

- 定义：[foundation/snowflake/snowflake.go:66](../../../foundation/snowflake/snowflake.go#L66) `NewNode`；分类S；阶段P1/P2。
- 完整当前类型：`func(datacenterID int64, nodeID int64) (*github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake.Node, error)`。

| 位置                                                                                               | 所在函数/声明                                             | 引用来源 |
| -------------------------------------------------------------------------------------------------- | --------------------------------------------------------- | -------- |
| [application/builtin/snowflake.go:85](../../../application/builtin/snowflake.go#L85)               | `Start`                                                   | SDK生产  |
| [foundation/snowflake/example_test.go:12](../../../foundation/snowflake/example_test.go#L12)       | `ExampleNewNode`                                          | 单测     |
| [foundation/snowflake/example_test.go:20](../../../foundation/snowflake/example_test.go#L20)       | `ExampleNewNode`                                          | 单测     |
| [foundation/snowflake/example_test.go:31](../../../foundation/snowflake/example_test.go#L31)       | `ExampleNode_Generate`                                    | 单测     |
| [foundation/snowflake/snowflake_test.go:15](../../../foundation/snowflake/snowflake_test.go#L15)   | `Test_SnowflakeID_Generate`                               | 单测     |
| [foundation/snowflake/snowflake_test.go:43](../../../foundation/snowflake/snowflake_test.go#L43)   | `Test_SnowflakeID_GenerateWithGene`                       | 单测     |
| [foundation/snowflake/snowflake_test.go:65](../../../foundation/snowflake/snowflake_test.go#L65)   | `Test_SnowflakeID_Uniqueness`                             | 单测     |
| [foundation/snowflake/snowflake_test.go:79](../../../foundation/snowflake/snowflake_test.go#L79)   | `Test_SnowflakeID_Concurrent`                             | 单测     |
| [foundation/snowflake/snowflake_test.go:114](../../../foundation/snowflake/snowflake_test.go#L114) | `Test_SnowflakeID_BitLayout`                              | 单测     |
| [foundation/snowflake/snowflake_test.go:148](../../../foundation/snowflake/snowflake_test.go#L148) | `Test_NewNode_Boundaries`                                 | 单测     |
| [foundation/snowflake/snowflake_test.go:162](../../../foundation/snowflake/snowflake_test.go#L162) | `Test_NewNode_Boundaries`                                 | 单测     |
| [foundation/snowflake/snowflake_test.go:185](../../../foundation/snowflake/snowflake_test.go#L185) | `Test_SequenceExhaustion_AdvancesTime`                    | 单测     |
| [foundation/snowflake/snowflake_test.go:221](../../../foundation/snowflake/snowflake_test.go#L221) | `Test_ClockBackwards_Error`                               | 单测     |
| [foundation/snowflake/snowflake_test.go:367](../../../foundation/snowflake/snowflake_test.go#L367) | `Benchmark_SnowflakeID_Generate`                          | 单测     |
| [foundation/snowflake/snowflake_test.go:375](../../../foundation/snowflake/snowflake_test.go#L375) | `Test_SnowflakeID_SequenceOverflowClockBackwardsRollback` | 单测     |
| [plugins/database/models/models_test.go:147](../../../plugins/database/models/models_test.go#L147) | `Test_BaseEntity_BeforeCreate_Success`                    | 单测     |
| [plugins/database/models/models_test.go:176](../../../plugins/database/models/models_test.go#L176) | `Test_BaseEntity_BeforeCreate_GeneProvider`               | 单测     |
| [plugins/database/models/models_test.go:252](../../../plugins/database/models/models_test.go#L252) | `Test_BaseEntity_BeforeCreate_InvalidGene`                | 单测     |
| [plugins/database/models/models_test.go:289](../../../plugins/database/models/models_test.go#L289) | `Test_BaseEntityWithSoftDelete_BeforeCreate_AndUpdate`    | 单测     |

#### P0176

- 定义：[foundation/snowflake/snowflake.go:94](../../../foundation/snowflake/snowflake.go#L94) `Node.Generate`；分类S；阶段P1/P2。
- 完整当前类型：`func(gene ...github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake.Gene) (github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake.SnowflakeID, error)`。

| 位置                                                                                               | 所在函数/声明                                             | 引用来源   |
| -------------------------------------------------------------------------------------------------- | --------------------------------------------------------- | ---------- |
| [application/builtin/builtin_test.go:225](../../../application/builtin/builtin_test.go#L225)       | `TestBuiltinSnowflake_LifecycleAndIsolation`              | 单测       |
| [application/builtin/builtin_test.go:229](../../../application/builtin/builtin_test.go#L229)       | `TestBuiltinSnowflake_LifecycleAndIsolation`              | 单测       |
| [application/builtin_lifecycle_test.go:55](../../../application/builtin_lifecycle_test.go#L55)     | `Run`                                                     | 单测       |
| [application/builtin_lifecycle_test.go:245](../../../application/builtin_lifecycle_test.go#L245)   | `TestAppLifecycle_BuiltinLoggingAndSnowflake`             | 单测       |
| [examples/v2/worker-tool/main.go:101](../../../examples/v2/worker-tool/main.go#L101)               | `Run`                                                     | 仓内消费者 |
| [foundation/snowflake/example_test.go:37](../../../foundation/snowflake/example_test.go#L37)       | `ExampleNode_Generate`                                    | 单测       |
| [foundation/snowflake/snowflake.go:152](../../../foundation/snowflake/snowflake.go#L152)           | `MustGenerate`                                            | SDK生产    |
| [foundation/snowflake/snowflake_test.go:123](../../../foundation/snowflake/snowflake_test.go#L123) | `Test_SnowflakeID_BitLayout`                              | 单测       |
| [foundation/snowflake/snowflake_test.go:170](../../../foundation/snowflake/snowflake_test.go#L170) | `Test_NilNode_Generate`                                   | 单测       |
| [foundation/snowflake/snowflake_test.go:193](../../../foundation/snowflake/snowflake_test.go#L193) | `Test_SequenceExhaustion_AdvancesTime`                    | 单测       |
| [foundation/snowflake/snowflake_test.go:208](../../../foundation/snowflake/snowflake_test.go#L208) | `Test_SequenceExhaustion_AdvancesTime`                    | 单测       |
| [foundation/snowflake/snowflake_test.go:227](../../../foundation/snowflake/snowflake_test.go#L227) | `Test_ClockBackwards_Error`                               | 单测       |
| [foundation/snowflake/snowflake_test.go:234](../../../foundation/snowflake/snowflake_test.go#L234) | `Test_ClockBackwards_Error`                               | 单测       |
| [foundation/snowflake/snowflake_test.go:388](../../../foundation/snowflake/snowflake_test.go#L388) | `Test_SnowflakeID_SequenceOverflowClockBackwardsRollback` | 单测       |
| [foundation/snowflake/snowflake_test.go:411](../../../foundation/snowflake/snowflake_test.go#L411) | `Test_SnowflakeID_SequenceOverflowClockBackwardsRollback` | 单测       |
| [foundation/snowflake/snowflake_test.go:430](../../../foundation/snowflake/snowflake_test.go#L430) | `Test_SnowflakeID_SequenceOverflowClockBackwardsRollback` | 单测       |
| [foundation/snowflake/snowflake_test.go:441](../../../foundation/snowflake/snowflake_test.go#L441) | `Test_SnowflakeID_SequenceOverflowClockBackwardsRollback` | 单测       |
| [plugins/database/models/base_entity.go:89](../../../plugins/database/models/base_entity.go#L89)   | `generateEntityID`                                        | SDK生产    |

#### P0177

- 定义：[foundation/snowflake/snowflake.go:196](../../../foundation/snowflake/snowflake.go#L196) `SnowflakeID.MarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding/json.Marshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0178

- 定义：[foundation/snowflake/snowflake.go:204](../../../foundation/snowflake/snowflake.go#L204) `SnowflakeID.UnmarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func(data []byte) error`。
- 方法集：`encoding/json.Unmarshaler`。

| 位置                                                                                               | 所在函数/声明                      | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ---------------------------------- | -------- |
| [foundation/snowflake/example_test.go:88](../../../foundation/snowflake/example_test.go#L88)       | `ExampleSnowflakeID_UnmarshalJSON` | 单测     |
| [foundation/snowflake/snowflake_test.go:271](../../../foundation/snowflake/snowflake_test.go#L271) | `Test_SnowflakeID_JSON`            | 单测     |
| [foundation/snowflake/snowflake_test.go:277](../../../foundation/snowflake/snowflake_test.go#L277) | `Test_SnowflakeID_JSON`            | 单测     |

#### P0179

- 定义：[foundation/snowflake/snowflake.go:229](../../../foundation/snowflake/snowflake.go#L229) `SnowflakeID.MarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding.BinaryMarshaler`。

| 位置                                                                                               | 所在函数/声明             | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ------------------------- | -------- |
| [foundation/snowflake/snowflake_test.go:290](../../../foundation/snowflake/snowflake_test.go#L290) | `Test_SnowflakeID_Binary` | 单测     |

#### P0180

- 定义：[foundation/snowflake/snowflake.go:236](../../../foundation/snowflake/snowflake.go#L236) `SnowflakeID.UnmarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func(data []byte) error`。
- 方法集：`encoding.BinaryUnmarshaler`。

| 位置                                                                                               | 所在函数/声明             | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ------------------------- | -------- |
| [foundation/snowflake/snowflake_test.go:296](../../../foundation/snowflake/snowflake_test.go#L296) | `Test_SnowflakeID_Binary` | 单测     |
| [foundation/snowflake/snowflake_test.go:304](../../../foundation/snowflake/snowflake_test.go#L304) | `Test_SnowflakeID_Binary` | 单测     |
| [foundation/snowflake/snowflake_test.go:309](../../../foundation/snowflake/snowflake_test.go#L309) | `Test_SnowflakeID_Binary` | 单测     |

#### P0181

- 定义：[foundation/snowflake/snowflake.go:251](../../../foundation/snowflake/snowflake.go#L251) `SnowflakeID.Value`；分类F；阶段P1/P2。
- 完整当前类型：`func() (database/sql/driver.Value, error)`。
- 方法集：`database/sql/driver.Valuer`。

| 位置                                                                                               | 所在函数/声明                     | 引用来源 |
| -------------------------------------------------------------------------------------------------- | --------------------------------- | -------- |
| [foundation/snowflake/snowflake_test.go:321](../../../foundation/snowflake/snowflake_test.go#L321) | `Test_SnowflakeID_SQL_Scan_Value` | 单测     |

#### P0182

- 定义：[foundation/snowflake/snowflake.go:260](../../../foundation/snowflake/snowflake.go#L260) `SnowflakeID.Scan`；分类F；阶段P1/P2。
- 完整当前类型：`func(value interface{}) error`。
- 方法集：`database/sql.Scanner`。

| 位置                                                                                               | 所在函数/声明                     | 引用来源 |
| -------------------------------------------------------------------------------------------------- | --------------------------------- | -------- |
| [foundation/snowflake/snowflake_test.go:328](../../../foundation/snowflake/snowflake_test.go#L328) | `Test_SnowflakeID_SQL_Scan_Value` | 单测     |
| [foundation/snowflake/snowflake_test.go:336](../../../foundation/snowflake/snowflake_test.go#L336) | `Test_SnowflakeID_SQL_Scan_Value` | 单测     |
| [foundation/snowflake/snowflake_test.go:341](../../../foundation/snowflake/snowflake_test.go#L341) | `Test_SnowflakeID_SQL_Scan_Value` | 单测     |
| [foundation/snowflake/snowflake_test.go:346](../../../foundation/snowflake/snowflake_test.go#L346) | `Test_SnowflakeID_SQL_Scan_Value` | 单测     |
| [foundation/snowflake/snowflake_test.go:352](../../../foundation/snowflake/snowflake_test.go#L352) | `Test_SnowflakeID_SQL_Scan_Value` | 单测     |
| [foundation/snowflake/snowflake_test.go:358](../../../foundation/snowflake/snowflake_test.go#L358) | `Test_SnowflakeID_SQL_Scan_Value` | 单测     |

#### P0183

- 定义：[foundation/toolkit/password.go:23](../../../foundation/toolkit/password.go#L23) `EncryptPassword`；分类S；阶段P1/P2。
- 完整当前类型：`func(pass string) ([]byte, error)`。

| 位置                                                                                         | 所在函数/声明                    | 引用来源 |
| -------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [foundation/toolkit/password.go:44](../../../foundation/toolkit/password.go#L44)             | `MustEncryptPassword`            | SDK生产  |
| [foundation/toolkit/password.go:85](../../../foundation/toolkit/password.go#L85)             | `EncryptPasswordString`          | SDK生产  |
| [foundation/toolkit/password.go:103](../../../foundation/toolkit/password.go#L103)           | `MustEncryptPasswordString`      | SDK生产  |
| [foundation/toolkit/password_test.go:22](../../../foundation/toolkit/password_test.go#L22)   | `TestEncryptPassword`            | 单测     |
| [foundation/toolkit/password_test.go:42](../../../foundation/toolkit/password_test.go#L42)   | `TestEncryptPassword_Uniqueness` | 单测     |
| [foundation/toolkit/password_test.go:47](../../../foundation/toolkit/password_test.go#L47)   | `TestEncryptPassword_Uniqueness` | 单测     |
| [foundation/toolkit/password_test.go:163](../../../foundation/toolkit/password_test.go#L163) | `TestPassword_FullFlow`          | 单测     |

#### P0184

- 定义：[foundation/toolkit/password.go:64](../../../foundation/toolkit/password.go#L64) `VerifyPassword`；分类S；阶段P1/P2。
- 完整当前类型：`func(inputPass string, hashPass string) error`。

| 位置                                                                                         | 所在函数/声明                          | 引用来源 |
| -------------------------------------------------------------------------------------------- | -------------------------------------- | -------- |
| [foundation/toolkit/password.go:122](../../../foundation/toolkit/password.go#L122)           | `IsPasswordValid`                      | SDK生产  |
| [foundation/toolkit/password_test.go:97](../../../foundation/toolkit/password_test.go#L97)   | `TestVerifyPassword_CorrectPassword`   | 单测     |
| [foundation/toolkit/password_test.go:114](../../../foundation/toolkit/password_test.go#L114) | `TestVerifyPassword_IncorrectPassword` | 单测     |
| [foundation/toolkit/password_test.go:123](../../../foundation/toolkit/password_test.go#L123) | `TestVerifyPassword_InvalidHash`       | 单测     |
| [foundation/toolkit/password_test.go:184](../../../foundation/toolkit/password_test.go#L184) | `TestPassword_FullFlow`                | 单测     |

#### P0185

- 定义：[foundation/toolkit/password.go:84](../../../foundation/toolkit/password.go#L84) `EncryptPasswordString`；分类S；阶段P1/P2。
- 完整当前类型：`func(pass string) (string, error)`。

| 位置                                                                                         | 所在函数/声明                          | 引用来源 |
| -------------------------------------------------------------------------------------------- | -------------------------------------- | -------- |
| [foundation/toolkit/example_test.go:81](../../../foundation/toolkit/example_test.go#L81)     | `ExampleEncryptPassword`               | 单测     |
| [foundation/toolkit/password_test.go:60](../../../foundation/toolkit/password_test.go#L60)   | `TestEncryptPasswordString`            | 单测     |
| [foundation/toolkit/password_test.go:91](../../../foundation/toolkit/password_test.go#L91)   | `TestVerifyPassword_CorrectPassword`   | 单测     |
| [foundation/toolkit/password_test.go:108](../../../foundation/toolkit/password_test.go#L108) | `TestVerifyPassword_IncorrectPassword` | 单测     |
| [foundation/toolkit/password_test.go:179](../../../foundation/toolkit/password_test.go#L179) | `TestPassword_FullFlow`                | 单测     |

#### P0186

- 定义：[foundation/toolkit/time.go:88](../../../foundation/toolkit/time.go#L88) `Parse`；分类S；阶段P1/P2。
- 完整当前类型：`func(layout string, value string) (time.Time, error)`。

| 位置                                                                               | 所在函数/声明     | 引用来源 |
| ---------------------------------------------------------------------------------- | ----------------- | -------- |
| [foundation/toolkit/time_test.go:93](../../../foundation/toolkit/time_test.go#L93) | `TestTimer_Parse` | 单测     |

#### P0187

- 定义：[foundation/typed/convert.go:18](../../../foundation/typed/convert.go#L18) `ErrNilReceiver`；分类N；阶段P1/P2。
- 完整当前类型：`error`。

| 位置                                                                                   | 所在函数/声明              | 引用来源 |
| -------------------------------------------------------------------------------------- | -------------------------- | -------- |
| [foundation/typed/convert.go:34](../../../foundation/typed/convert.go#L34)             | `UnmarshalText`            | SDK生产  |
| [foundation/typed/convert.go:59](../../../foundation/typed/convert.go#L59)             | `UnmarshalBinary`          | SDK生产  |
| [foundation/typed/convert.go:83](../../../foundation/typed/convert.go#L83)             | `UnmarshalJSON`            | SDK生产  |
| [foundation/typed/convert.go:118](../../../foundation/typed/convert.go#L118)           | `UnmarshalXML`             | SDK生产  |
| [foundation/typed/convert.go:172](../../../foundation/typed/convert.go#L172)           | `UnmarshalXMLAttr`         | SDK生产  |
| [foundation/typed/convert.go:197](../../../foundation/typed/convert.go#L197)           | `Scan`                     | SDK生产  |
| [foundation/typed/convert.go:226](../../../foundation/typed/convert.go#L226)           | `Set`                      | SDK生产  |
| [foundation/typed/convert_test.go:39](../../../foundation/typed/convert_test.go#L39)   | `TestConvert_Text`         | 单测     |
| [foundation/typed/convert_test.go:67](../../../foundation/typed/convert_test.go#L67)   | `TestConvert_BinaryAndGob` | 单测     |
| [foundation/typed/convert_test.go:120](../../../foundation/typed/convert_test.go#L120) | `TestConvert_JSON`         | 单测     |
| [foundation/typed/convert_test.go:165](../../../foundation/typed/convert_test.go#L165) | `TestConvert_XML`          | 单测     |
| [foundation/typed/convert_test.go:168](../../../foundation/typed/convert_test.go#L168) | `TestConvert_XML`          | 单测     |
| [foundation/typed/convert_test.go:209](../../../foundation/typed/convert_test.go#L209) | `TestConvert_SQL`          | 单测     |
| [foundation/typed/convert_test.go:231](../../../foundation/typed/convert_test.go#L231) | `TestConvert_Flag`         | 单测     |

#### P0188

- 定义：[foundation/typed/convert.go:22](../../../foundation/typed/convert.go#L22) `String.MarshalText`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding.TextMarshaler`。

| 位置                                                                                 | 所在函数/声明               | 引用来源 |
| ------------------------------------------------------------------------------------ | --------------------------- | -------- |
| [foundation/ctxkey/key.go:73](../../../foundation/ctxkey/key.go#L73)                 | `MarshalText`               | SDK生产  |
| [foundation/env/key.go:67](../../../foundation/env/key.go#L67)                       | `MarshalText`               | SDK生产  |
| [foundation/typed/convert_test.go:17](../../../foundation/typed/convert_test.go#L17) | `TestConvert_Text`          | 单测     |
| [foundation/typed/convert_test.go:23](../../../foundation/typed/convert_test.go#L23) | `TestConvert_Text`          | 单测     |
| [foundation/typed/example_test.go:12](../../../foundation/typed/example_test.go#L12) | `ExampleString_MarshalText` | 单测     |
| [foundation/typed/example_test.go:15](../../../foundation/typed/example_test.go#L15) | `ExampleString_MarshalText` | 单测     |
| [foundation/typed/index_test.go:49](../../../foundation/typed/index_test.go#L49)     | `MarshalText`               | 单测     |

#### P0189

- 定义：[foundation/typed/convert.go:32](../../../foundation/typed/convert.go#L32) `String.UnmarshalText`；分类F；阶段P1/P2。
- 完整当前类型：`func(text []byte) error`。
- 方法集：`encoding.TextUnmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                                 | 所在函数/声明      | 引用来源 |
| ------------------------------------------------------------------------------------ | ------------------ | -------- |
| [foundation/ctxkey/key.go:84](../../../foundation/ctxkey/key.go#L84)                 | `UnmarshalText`    | SDK生产  |
| [foundation/env/key.go:79](../../../foundation/env/key.go#L79)                       | `UnmarshalText`    | SDK生产  |
| [foundation/typed/convert_test.go:28](../../../foundation/typed/convert_test.go#L28) | `TestConvert_Text` | 单测     |
| [foundation/typed/convert_test.go:33](../../../foundation/typed/convert_test.go#L33) | `TestConvert_Text` | 单测     |
| [foundation/typed/convert_test.go:39](../../../foundation/typed/convert_test.go#L39) | `TestConvert_Text` | 单测     |
| [foundation/typed/index_test.go:57](../../../foundation/typed/index_test.go#L57)     | `UnmarshalText`    | 单测     |

#### P0190

- 定义：[foundation/typed/convert.go:47](../../../foundation/typed/convert.go#L47) `String.MarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding.BinaryMarshaler`。

| 位置                                                                                 | 所在函数/声明              | 引用来源 |
| ------------------------------------------------------------------------------------ | -------------------------- | -------- |
| [foundation/ctxkey/key.go:95](../../../foundation/ctxkey/key.go#L95)                 | `MarshalBinary`            | SDK生产  |
| [foundation/env/key.go:90](../../../foundation/env/key.go#L90)                       | `MarshalBinary`            | SDK生产  |
| [foundation/typed/convert_test.go:47](../../../foundation/typed/convert_test.go#L47) | `TestConvert_BinaryAndGob` | 单测     |
| [foundation/typed/convert_test.go:53](../../../foundation/typed/convert_test.go#L53) | `TestConvert_BinaryAndGob` | 单测     |
| [foundation/typed/index_test.go:65](../../../foundation/typed/index_test.go#L65)     | `MarshalBinary`            | 单测     |

#### P0191

- 定义：[foundation/typed/convert.go:57](../../../foundation/typed/convert.go#L57) `String.UnmarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func(data []byte) error`。
- 方法集：`encoding.BinaryUnmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                                 | 所在函数/声明              | 引用来源 |
| ------------------------------------------------------------------------------------ | -------------------------- | -------- |
| [foundation/ctxkey/key.go:106](../../../foundation/ctxkey/key.go#L106)               | `UnmarshalBinary`          | SDK生产  |
| [foundation/env/key.go:102](../../../foundation/env/key.go#L102)                     | `UnmarshalBinary`          | SDK生产  |
| [foundation/typed/convert_test.go:58](../../../foundation/typed/convert_test.go#L58) | `TestConvert_BinaryAndGob` | 单测     |
| [foundation/typed/convert_test.go:62](../../../foundation/typed/convert_test.go#L62) | `TestConvert_BinaryAndGob` | 单测     |
| [foundation/typed/convert_test.go:67](../../../foundation/typed/convert_test.go#L67) | `TestConvert_BinaryAndGob` | 单测     |
| [foundation/typed/index_test.go:73](../../../foundation/typed/index_test.go#L73)     | `UnmarshalBinary`          | 单测     |

#### P0192

- 定义：[foundation/typed/convert.go:71](../../../foundation/typed/convert.go#L71) `String.MarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding/json.Marshaler`。

| 位置                                                                             | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------- | ------------- | -------- |
| [foundation/ctxkey/key.go:116](../../../foundation/ctxkey/key.go#L116)           | `MarshalJSON` | SDK生产  |
| [foundation/env/key.go:113](../../../foundation/env/key.go#L113)                 | `MarshalJSON` | SDK生产  |
| [foundation/typed/index_test.go:81](../../../foundation/typed/index_test.go#L81) | `MarshalJSON` | 单测     |

#### P0193

- 定义：[foundation/typed/convert.go:81](../../../foundation/typed/convert.go#L81) `String.UnmarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func(raw []byte) error`。
- 方法集：`encoding/json.Unmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                                   | 所在函数/声明      | 引用来源 |
| -------------------------------------------------------------------------------------- | ------------------ | -------- |
| [foundation/ctxkey/key.go:127](../../../foundation/ctxkey/key.go#L127)                 | `UnmarshalJSON`    | SDK生产  |
| [foundation/env/key.go:125](../../../foundation/env/key.go#L125)                       | `UnmarshalJSON`    | SDK生产  |
| [foundation/typed/convert_test.go:120](../../../foundation/typed/convert_test.go#L120) | `TestConvert_JSON` | 单测     |
| [foundation/typed/index_test.go:89](../../../foundation/typed/index_test.go#L89)       | `UnmarshalJSON`    | 单测     |

#### P0194

- 定义：[foundation/typed/convert.go:105](../../../foundation/typed/convert.go#L105) `String.MarshalXML`；分类F；阶段P1/P2。
- 完整当前类型：`func(e *encoding/xml.Encoder, start encoding/xml.StartElement) error`。
- 方法集：`encoding/xml.Marshaler`。

| 位置                                                                             | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------- | ------------- | -------- |
| [foundation/ctxkey/key.go:137](../../../foundation/ctxkey/key.go#L137)           | `MarshalXML`  | SDK生产  |
| [foundation/env/key.go:136](../../../foundation/env/key.go#L136)                 | `MarshalXML`  | SDK生产  |
| [foundation/typed/index_test.go:97](../../../foundation/typed/index_test.go#L97) | `MarshalXML`  | 单测     |

#### P0195

- 定义：[foundation/typed/convert.go:116](../../../foundation/typed/convert.go#L116) `String.UnmarshalXML`；分类F；阶段P1/P2。
- 完整当前类型：`func(d *encoding/xml.Decoder, start encoding/xml.StartElement) error`。
- 方法集：`encoding/xml.Unmarshaler`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                                   | 所在函数/声明     | 引用来源 |
| -------------------------------------------------------------------------------------- | ----------------- | -------- |
| [foundation/ctxkey/key.go:148](../../../foundation/ctxkey/key.go#L148)                 | `UnmarshalXML`    | SDK生产  |
| [foundation/env/key.go:148](../../../foundation/env/key.go#L148)                       | `UnmarshalXML`    | SDK生产  |
| [foundation/typed/convert_test.go:165](../../../foundation/typed/convert_test.go#L165) | `TestConvert_XML` | 单测     |
| [foundation/typed/index_test.go:105](../../../foundation/typed/index_test.go#L105)     | `UnmarshalXML`    | 单测     |

#### P0196

- 定义：[foundation/typed/convert.go:157](../../../foundation/typed/convert.go#L157) `String.MarshalXMLAttr`；分类F；阶段P1/P2。
- 完整当前类型：`func(name encoding/xml.Name) (encoding/xml.Attr, error)`。
- 方法集：`encoding/xml.MarshalerAttr`。

| 位置                                                                               | 所在函数/声明    | 引用来源 |
| ---------------------------------------------------------------------------------- | ---------------- | -------- |
| [foundation/ctxkey/key.go:158](../../../foundation/ctxkey/key.go#L158)             | `MarshalXMLAttr` | SDK生产  |
| [foundation/env/key.go:159](../../../foundation/env/key.go#L159)                   | `MarshalXMLAttr` | SDK生产  |
| [foundation/typed/index_test.go:113](../../../foundation/typed/index_test.go#L113) | `MarshalXMLAttr` | 单测     |

#### P0197

- 定义：[foundation/typed/convert.go:170](../../../foundation/typed/convert.go#L170) `String.UnmarshalXMLAttr`；分类F；阶段P1/P2。
- 完整当前类型：`func(attr encoding/xml.Attr) error`。
- 方法集：`encoding/xml.UnmarshalerAttr`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                                   | 所在函数/声明      | 引用来源 |
| -------------------------------------------------------------------------------------- | ------------------ | -------- |
| [foundation/ctxkey/key.go:169](../../../foundation/ctxkey/key.go#L169)                 | `UnmarshalXMLAttr` | SDK生产  |
| [foundation/env/key.go:171](../../../foundation/env/key.go#L171)                       | `UnmarshalXMLAttr` | SDK生产  |
| [foundation/typed/convert_test.go:159](../../../foundation/typed/convert_test.go#L159) | `TestConvert_XML`  | 单测     |
| [foundation/typed/convert_test.go:168](../../../foundation/typed/convert_test.go#L168) | `TestConvert_XML`  | 单测     |
| [foundation/typed/index_test.go:121](../../../foundation/typed/index_test.go#L121)     | `UnmarshalXMLAttr` | 单测     |

#### P0198

- 定义：[foundation/typed/convert.go:184](../../../foundation/typed/convert.go#L184) `String.Value`；分类F；阶段P1/P2。
- 完整当前类型：`func() (database/sql/driver.Value, error)`。
- 方法集：`database/sql/driver.Valuer`。

| 位置                                                                                   | 所在函数/声明        | 引用来源 |
| -------------------------------------------------------------------------------------- | -------------------- | -------- |
| [foundation/ctxkey/key.go:179](../../../foundation/ctxkey/key.go#L179)                 | `Value`              | SDK生产  |
| [foundation/env/key.go:182](../../../foundation/env/key.go#L182)                       | `Value`              | SDK生产  |
| [foundation/typed/convert_test.go:176](../../../foundation/typed/convert_test.go#L176) | `TestConvert_SQL`    | 单测     |
| [foundation/typed/convert_test.go:182](../../../foundation/typed/convert_test.go#L182) | `TestConvert_SQL`    | 单测     |
| [foundation/typed/example_test.go:62](../../../foundation/typed/example_test.go#L62)   | `ExampleString_Scan` | 单测     |
| [foundation/typed/index_test.go:129](../../../foundation/typed/index_test.go#L129)     | `Value`              | 单测     |

#### P0199

- 定义：[foundation/typed/convert.go:195](../../../foundation/typed/convert.go#L195) `String.Scan`；分类F；阶段P1/P2。
- 完整当前类型：`func(src any) error`。
- 方法集：`database/sql.Scanner`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                                   | 所在函数/声明        | 引用来源 |
| -------------------------------------------------------------------------------------- | -------------------- | -------- |
| [foundation/ctxkey/key.go:190](../../../foundation/ctxkey/key.go#L190)                 | `Scan`               | SDK生产  |
| [foundation/env/key.go:194](../../../foundation/env/key.go#L194)                       | `Scan`               | SDK生产  |
| [foundation/typed/convert_test.go:188](../../../foundation/typed/convert_test.go#L188) | `TestConvert_SQL`    | 单测     |
| [foundation/typed/convert_test.go:193](../../../foundation/typed/convert_test.go#L193) | `TestConvert_SQL`    | 单测     |
| [foundation/typed/convert_test.go:198](../../../foundation/typed/convert_test.go#L198) | `TestConvert_SQL`    | 单测     |
| [foundation/typed/convert_test.go:203](../../../foundation/typed/convert_test.go#L203) | `TestConvert_SQL`    | 单测     |
| [foundation/typed/convert_test.go:209](../../../foundation/typed/convert_test.go#L209) | `TestConvert_SQL`    | 单测     |
| [foundation/typed/example_test.go:69](../../../foundation/typed/example_test.go#L69)   | `ExampleString_Scan` | 单测     |
| [foundation/typed/example_test.go:75](../../../foundation/typed/example_test.go#L75)   | `ExampleString_Scan` | 单测     |
| [foundation/typed/index_test.go:137](../../../foundation/typed/index_test.go#L137)     | `Scan`               | 单测     |

#### P0200

- 定义：[foundation/typed/convert.go:224](../../../foundation/typed/convert.go#L224) `String.Set`；分类F；阶段P1/P2。
- 完整当前类型：`func(raw string) error`。
- 方法集：`flag.Value`；`github.com/bamboo-services/bamboo-base-go/foundation/v2/typed.Convertible`。

| 位置                                                                                   | 所在函数/声明      | 引用来源 |
| -------------------------------------------------------------------------------------- | ------------------ | -------- |
| [foundation/ctxkey/key.go:205](../../../foundation/ctxkey/key.go#L205)                 | `Set`              | SDK生产  |
| [foundation/env/key.go:210](../../../foundation/env/key.go#L210)                       | `Set`              | SDK生产  |
| [foundation/typed/convert_test.go:231](../../../foundation/typed/convert_test.go#L231) | `TestConvert_Flag` | 单测     |
| [foundation/typed/index_test.go:149](../../../foundation/typed/index_test.go#L149)     | `Set`              | 单测     |

#### P0201

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.MarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func() (data []byte, err error)`。
- 方法集：`encoding.BinaryMarshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0202

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.MarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func() ([]byte, error)`。
- 方法集：`encoding/json.Marshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0203

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.MarshalText`；分类F；阶段P1/P2。
- 完整当前类型：`func() (text []byte, err error)`。
- 方法集：`encoding.TextMarshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0204

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.MarshalXML`；分类F；阶段P1/P2。
- 完整当前类型：`func(e *encoding/xml.Encoder, start encoding/xml.StartElement) error`。
- 方法集：`encoding/xml.Marshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0205

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.MarshalXMLAttr`；分类F；阶段P1/P2。
- 完整当前类型：`func(name encoding/xml.Name) (encoding/xml.Attr, error)`。
- 方法集：`encoding/xml.MarshalerAttr`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0206

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.Scan`；分类F；阶段P1/P2。
- 完整当前类型：`func(src any) error`。
- 方法集：`database/sql.Scanner`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0207

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.Set`；分类F；阶段P1/P2。
- 完整当前类型：`func(string) error`。
- 方法集：`flag.Value`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0208

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.UnmarshalBinary`；分类F；阶段P1/P2。
- 完整当前类型：`func(data []byte) error`。
- 方法集：`encoding.BinaryUnmarshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0209

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.UnmarshalJSON`；分类F；阶段P1/P2。
- 完整当前类型：`func([]byte) error`。
- 方法集：`encoding/json.Unmarshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0210

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.UnmarshalText`；分类F；阶段P1/P2。
- 完整当前类型：`func(text []byte) error`。
- 方法集：`encoding.TextUnmarshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0211

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.UnmarshalXML`；分类F；阶段P1/P2。
- 完整当前类型：`func(d *encoding/xml.Decoder, start encoding/xml.StartElement) error`。
- 方法集：`encoding/xml.Unmarshaler`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0212

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.UnmarshalXMLAttr`；分类F；阶段P1/P2。
- 完整当前类型：`func(attr encoding/xml.Attr) error`。
- 方法集：`encoding/xml.UnmarshalerAttr`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0213

- 定义：[foundation/typed/index.go:44](../../../foundation/typed/index.go#L44) `Convertible.Value`；分类F；阶段P1/P2。
- 完整当前类型：`func() (database/sql/driver.Value, error)`。
- 方法集：`database/sql/driver.Valuer`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0214

- 定义：[foundation/typed/parse.go:14](../../../foundation/typed/parse.go#L14) `String.Int`；分类S；阶段P1/P2。
- 完整当前类型：`func() (int, error)`。

| 位置                                                                                 | 所在函数/声明              | 引用来源 |
| ------------------------------------------------------------------------------------ | -------------------------- | -------- |
| [foundation/env/example_test.go:18](../../../foundation/env/example_test.go#L18)     | `ExampleLookup`            | 单测     |
| [foundation/env/key_test.go:77](../../../foundation/env/key_test.go#L77)             | `TestLookup`               | 单测     |
| [foundation/typed/doc_test.go:16](../../../foundation/typed/doc_test.go#L16)         | `ExampleString`            | 单测     |
| [foundation/typed/example_test.go:25](../../../foundation/typed/example_test.go#L25) | `ExampleString_Int`        | 单测     |
| [foundation/typed/example_test.go:28](../../../foundation/typed/example_test.go#L28) | `ExampleString_Int`        | 单测     |
| [foundation/typed/parse_test.go:32](../../../foundation/typed/parse_test.go#L32)     | `TestStringParseIntFamily` | 单测     |
| [plugins/cache/config.go:189](../../../plugins/cache/config.go#L189)                 | `applyFromEnv`             | SDK生产  |
| [plugins/cache/config.go:197](../../../plugins/cache/config.go#L197)                 | `applyFromEnv`             | SDK生产  |
| [plugins/cache/config.go:218](../../../plugins/cache/config.go#L218)                 | `applyFromEnv`             | SDK生产  |
| [plugins/cache/config.go:226](../../../plugins/cache/config.go#L226)                 | `applyFromEnv`             | SDK生产  |
| [plugins/email/client.go:190](../../../plugins/email/client.go#L190)                 | `Start`                    | SDK生产  |
| [plugins/grpc/plugin.go:302](../../../plugins/grpc/plugin.go#L302)                   | `resolveGRPCAddr`          | SDK生产  |

#### P0215

- 定义：[foundation/typed/parse.go:24](../../../foundation/typed/parse.go#L24) `String.Int8`；分类S；阶段P1/P2。
- 完整当前类型：`func() (int8, error)`。

| 位置                                                                             | 所在函数/声明              | 引用来源 |
| -------------------------------------------------------------------------------- | -------------------------- | -------- |
| [foundation/typed/parse_test.go:58](../../../foundation/typed/parse_test.go#L58) | `TestStringParseIntFamily` | 单测     |

#### P0216

- 定义：[foundation/typed/parse.go:34](../../../foundation/typed/parse.go#L34) `String.Int16`；分类S；阶段P1/P2。
- 完整当前类型：`func() (int16, error)`。

| 位置                                                                             | 所在函数/声明              | 引用来源 |
| -------------------------------------------------------------------------------- | -------------------------- | -------- |
| [foundation/typed/parse_test.go:83](../../../foundation/typed/parse_test.go#L83) | `TestStringParseIntFamily` | 单测     |

#### P0217

- 定义：[foundation/typed/parse.go:44](../../../foundation/typed/parse.go#L44) `String.Int32`；分类S；阶段P1/P2。
- 完整当前类型：`func() (int32, error)`。

| 位置                                                                               | 所在函数/声明              | 引用来源 |
| ---------------------------------------------------------------------------------- | -------------------------- | -------- |
| [foundation/typed/parse_test.go:108](../../../foundation/typed/parse_test.go#L108) | `TestStringParseIntFamily` | 单测     |

#### P0218

- 定义：[foundation/typed/parse.go:54](../../../foundation/typed/parse.go#L54) `String.Int64`；分类S；阶段P1/P2。
- 完整当前类型：`func() (int64, error)`。

| 位置                                                                               | 所在函数/声明              | 引用来源 |
| ---------------------------------------------------------------------------------- | -------------------------- | -------- |
| [foundation/typed/parse_test.go:133](../../../foundation/typed/parse_test.go#L133) | `TestStringParseIntFamily` | 单测     |

#### P0219

- 定义：[foundation/typed/parse.go:65](../../../foundation/typed/parse.go#L65) `String.Uint`；分类S；阶段P1/P2。
- 完整当前类型：`func() (uint, error)`。

| 位置                                                                               | 所在函数/声明               | 引用来源 |
| ---------------------------------------------------------------------------------- | --------------------------- | -------- |
| [foundation/typed/parse_test.go:163](../../../foundation/typed/parse_test.go#L163) | `TestStringParseUintFamily` | 单测     |

#### P0220

- 定义：[foundation/typed/parse.go:75](../../../foundation/typed/parse.go#L75) `String.Uint8`；分类S；阶段P1/P2。
- 完整当前类型：`func() (uint8, error)`。

| 位置                                                                               | 所在函数/声明               | 引用来源 |
| ---------------------------------------------------------------------------------- | --------------------------- | -------- |
| [foundation/typed/parse_test.go:188](../../../foundation/typed/parse_test.go#L188) | `TestStringParseUintFamily` | 单测     |

#### P0221

- 定义：[foundation/typed/parse.go:85](../../../foundation/typed/parse.go#L85) `String.Uint16`；分类S；阶段P1/P2。
- 完整当前类型：`func() (uint16, error)`。

| 位置                                                                               | 所在函数/声明               | 引用来源 |
| ---------------------------------------------------------------------------------- | --------------------------- | -------- |
| [foundation/typed/parse_test.go:212](../../../foundation/typed/parse_test.go#L212) | `TestStringParseUintFamily` | 单测     |

#### P0222

- 定义：[foundation/typed/parse.go:95](../../../foundation/typed/parse.go#L95) `String.Uint32`；分类S；阶段P1/P2。
- 完整当前类型：`func() (uint32, error)`。

| 位置                                                                               | 所在函数/声明               | 引用来源 |
| ---------------------------------------------------------------------------------- | --------------------------- | -------- |
| [foundation/typed/parse_test.go:236](../../../foundation/typed/parse_test.go#L236) | `TestStringParseUintFamily` | 单测     |

#### P0223

- 定义：[foundation/typed/parse.go:105](../../../foundation/typed/parse.go#L105) `String.Uint64`；分类S；阶段P1/P2。
- 完整当前类型：`func() (uint64, error)`。

| 位置                                                                               | 所在函数/声明               | 引用来源 |
| ---------------------------------------------------------------------------------- | --------------------------- | -------- |
| [foundation/typed/parse_test.go:260](../../../foundation/typed/parse_test.go#L260) | `TestStringParseUintFamily` | 单测     |

#### P0224

- 定义：[foundation/typed/parse.go:116](../../../foundation/typed/parse.go#L116) `String.Float32`；分类S；阶段P1/P2。
- 完整当前类型：`func() (float32, error)`。

| 位置                                                                               | 所在函数/声明                | 引用来源 |
| ---------------------------------------------------------------------------------- | ---------------------------- | -------- |
| [foundation/typed/parse_test.go:295](../../../foundation/typed/parse_test.go#L295) | `TestStringParseFloatFamily` | 单测     |

#### P0225

- 定义：[foundation/typed/parse.go:131](../../../foundation/typed/parse.go#L131) `String.Float64`；分类S；阶段P1/P2。
- 完整当前类型：`func() (float64, error)`。

| 位置                                                                               | 所在函数/声明                | 引用来源 |
| ---------------------------------------------------------------------------------- | ---------------------------- | -------- |
| [foundation/typed/parse_test.go:326](../../../foundation/typed/parse_test.go#L326) | `TestStringParseFloatFamily` | 单测     |

#### P0226

- 定义：[foundation/typed/parse.go:149](../../../foundation/typed/parse.go#L149) `String.Bool`；分类S；阶段P1/P2。
- 完整当前类型：`func() (bool, error)`。

| 位置                                                                                 | 所在函数/声明         | 引用来源 |
| ------------------------------------------------------------------------------------ | --------------------- | -------- |
| [foundation/ctxkey/meta.go:19](../../../foundation/ctxkey/meta.go#L19)               | `IsDebugMode`         | SDK生产  |
| [foundation/env/key_test.go:97](../../../foundation/env/key_test.go#L97)             | `TestLookup`          | 单测     |
| [foundation/typed/example_test.go:38](../../../foundation/typed/example_test.go#L38) | `ExampleString_Bool`  | 单测     |
| [foundation/typed/example_test.go:41](../../../foundation/typed/example_test.go#L41) | `ExampleString_Bool`  | 单测     |
| [foundation/typed/parse_test.go:379](../../../foundation/typed/parse_test.go#L379)   | `TestStringParseBool` | 单测     |
| [plugins/grpc/plugin.go:446](../../../plugins/grpc/plugin.go#L446)                   | `Start`               | SDK生产  |

#### P0227

- 定义：[foundation/typed/parse.go:164](../../../foundation/typed/parse.go#L164) `String.Duration`；分类S；阶段P1/P2。
- 完整当前类型：`func() (time.Duration, error)`。

| 位置                                                                                 | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------------------------------ | ------------------------- | -------- |
| [application/option.go:177](../../../application/option.go#L177)                     | `resolveField`            | SDK生产  |
| [foundation/env/key_test.go:87](../../../foundation/env/key_test.go#L87)             | `TestLookup`              | 单测     |
| [foundation/typed/example_test.go:51](../../../foundation/typed/example_test.go#L51) | `ExampleString_Duration`  | 单测     |
| [foundation/typed/parse_test.go:411](../../../foundation/typed/parse_test.go#L411)   | `TestStringParseDuration` | 单测     |

#### P0228

- 定义：[plugins/async/plugin.go:27](../../../plugins/async/plugin.go#L27) `ErrPoolClosed`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                           | 所在函数/声明                              | 引用来源 |
| ------------------------------------------------------------------------------ | ------------------------------------------ | -------- |
| [plugins/async/plugin.go:324](../../../plugins/async/plugin.go#L324)           | `Submit`                                   | SDK生产  |
| [plugins/async/plugin.go:401](../../../plugins/async/plugin.go#L401)           | `Submit`                                   | SDK生产  |
| [plugins/async/plugin.go:424](../../../plugins/async/plugin.go#L424)           | `Submit`                                   | SDK生产  |
| [plugins/async/plugin_test.go:363](../../../plugins/async/plugin_test.go#L363) | `TestPlugin_RejectDuringStoppingAndClosed` | 单测     |
| [plugins/async/plugin_test.go:377](../../../plugins/async/plugin_test.go#L377) | `TestPlugin_RejectDuringStoppingAndClosed` | 单测     |

#### P0229

- 定义：[plugins/async/plugin.go:31](../../../plugins/async/plugin.go#L31) `ErrPoolFull`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                           | 所在函数/声明                        | 引用来源 |
| ------------------------------------------------------------------------------ | ------------------------------------ | -------- |
| [plugins/async/plugin.go:398](../../../plugins/async/plugin.go#L398)           | `Submit`                             | SDK生产  |
| [plugins/async/plugin_test.go:314](../../../plugins/async/plugin_test.go#L314) | `TestPlugin_PoolFullRejection`       | 单测     |
| [plugins/async/plugin_test.go:492](../../../plugins/async/plugin_test.go#L492) | `TestPlugin_SubmitRollbackOnFailure` | 单测     |

#### P0230

- 定义：[plugins/async/plugin.go:33](../../../plugins/async/plugin.go#L33) `ErrNilTaskFunc`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                           | 所在函数/声明                        | 引用来源 |
| ------------------------------------------------------------------------------ | ------------------------------------ | -------- |
| [plugins/async/plugin.go:317](../../../plugins/async/plugin.go#L317)           | `Submit`                             | SDK生产  |
| [plugins/async/plugin_test.go:475](../../../plugins/async/plugin_test.go#L475) | `TestPlugin_SubmitRollbackOnFailure` | 单测     |

#### P0231

- 定义：[plugins/async/plugin.go:34](../../../plugins/async/plugin.go#L34) `ErrNotInitialized`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/async/plugin.go:375](../../../plugins/async/plugin.go#L375) | `Submit`      | SDK生产  |
| [plugins/async/plugin.go:429](../../../plugins/async/plugin.go#L429) | `Submit`      | SDK生产  |

#### P0232

- 定义：[plugins/async/plugin.go:35](../../../plugins/async/plugin.go#L35) `ErrInvalidPoolCapacity`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/async/plugin.go:211](../../../plugins/async/plugin.go#L211) | `Start`       | SDK生产  |

#### P0233

- 定义：[plugins/async/plugin.go:36](../../../plugins/async/plugin.go#L36) `ErrInvalidContext`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                           | 所在函数/声明                               | 引用来源 |
| ------------------------------------------------------------------------------ | ------------------------------------------- | -------- |
| [plugins/async/plugin.go:330](../../../plugins/async/plugin.go#L330)           | `Submit`                                    | SDK生产  |
| [plugins/async/plugin.go:336](../../../plugins/async/plugin.go#L336)           | `Submit`                                    | SDK生产  |
| [plugins/async/plugin_test.go:548](../../../plugins/async/plugin_test.go#L548) | `TestPlugin_CrossAppAndBackgroundRejection` | 单测     |
| [plugins/async/plugin_test.go:559](../../../plugins/async/plugin_test.go#L559) | `TestPlugin_CrossAppAndBackgroundRejection` | 单测     |
| [plugins/async/plugin_test.go:571](../../../plugins/async/plugin_test.go#L571) | `TestPlugin_CrossAppAndBackgroundRejection` | 单测     |

#### P0234

- 定义：[plugins/async/plugin.go:45](../../../plugins/async/plugin.go#L45) `Pool.Submit`；分类S；阶段P5。
- 完整当前类型：`func(parentCtx context.Context, fn func(ctx context.Context), opts ...github.com/bamboo-services/bamboo-base-go/plugins/async/v2.TaskOption) (*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.Task, error)`。

| 位置                                                                                                                 | 所在函数/声明                                                                   | 引用来源   |
| -------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | ---------- |
| [examples/v2/grpc-cron-async/main.go:37](../../../examples/v2/grpc-cron-async/main.go#L37)                           | `Run`                                                                           | 仓内消费者 |
| [plugins/async/dependency_lifecycle_test.go:438](../../../plugins/async/dependency_lifecycle_test.go#L438)           | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated` | 单测       |
| [plugins/async/dependency_lifecycle_test.go:461](../../../plugins/async/dependency_lifecycle_test.go#L461)           | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated` | 单测       |
| [plugins/async/dependency_lifecycle_test.go:485](../../../plugins/async/dependency_lifecycle_test.go#L485)           | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated` | 单测       |
| [plugins/async/plugin.go:431](../../../plugins/async/plugin.go#L431)                                                 | `Submit`                                                                        | SDK生产    |
| [plugins/async/plugin_test.go:394](../../../plugins/async/plugin_test.go#L394)                                       | `TestPlugin_ForceCancelReceivedByTask`                                          | 单测       |
| [plugins/async/plugin_test.go:424](../../../plugins/async/plugin_test.go#L424)                                       | `TestPlugin_IgnoreCancelTaskMakesStopWait`                                      | 单测       |
| [plugins/async/plugin_test.go:474](../../../plugins/async/plugin_test.go#L474)                                       | `TestPlugin_SubmitRollbackOnFailure`                                            | 单测       |
| [plugins/async/plugin_test.go:484](../../../plugins/async/plugin_test.go#L484)                                       | `TestPlugin_SubmitRollbackOnFailure`                                            | 单测       |
| [plugins/async/plugin_test.go:491](../../../plugins/async/plugin_test.go#L491)                                       | `TestPlugin_SubmitRollbackOnFailure`                                            | 单测       |
| [plugins/async/plugin_test.go:525](../../../plugins/async/plugin_test.go#L525)                                       | `TestPlugin_CrossAppAndBackgroundRejection`                                     | 单测       |
| [plugins/async/plugin_test.go:545](../../../plugins/async/plugin_test.go#L545)                                       | `TestPlugin_CrossAppAndBackgroundRejection`                                     | 单测       |
| [plugins/async/plugin_test.go:556](../../../plugins/async/plugin_test.go#L556)                                       | `TestPlugin_CrossAppAndBackgroundRejection`                                     | 单测       |
| [plugins/async/plugin_test.go:568](../../../plugins/async/plugin_test.go#L568)                                       | `TestPlugin_CrossAppAndBackgroundRejection`                                     | 单测       |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:79](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L79)   | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`           | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:256](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L256) | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`   | 仓内消费者 |

#### P0235

- 定义：[plugins/async/plugin.go:164](../../../plugins/async/plugin.go#L164) `Plugin.stopErr`；分类S；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/async/plugin.go:248](../../../plugins/async/plugin.go#L248) | `Stop`        | SDK生产  |
| [plugins/async/plugin.go:253](../../../plugins/async/plugin.go#L253) | `Stop`        | SDK生产  |

#### P0236

- 定义：[plugins/async/plugin.go:202](../../../plugins/async/plugin.go#L202) `Plugin.Start`；分类S；阶段P3/P5。
- 完整当前类型：`func(_ context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0237

- 定义：[plugins/async/plugin.go:242](../../../plugins/async/plugin.go#L242) `Plugin.Stop`；分类S；阶段P3/P5。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。

| 位置                                                                           | 所在函数/声明                              | 引用来源 |
| ------------------------------------------------------------------------------ | ------------------------------------------ | -------- |
| [plugins/async/plugin_test.go:435](../../../plugins/async/plugin_test.go#L435) | `TestPlugin_IgnoreCancelTaskMakesStopWait` | 单测     |
| [plugins/async/plugin_test.go:456](../../../plugins/async/plugin_test.go#L456) | `TestPlugin_IgnoreCancelTaskMakesStopWait` | 单测     |

#### P0238

- 定义：[plugins/async/plugin.go:256](../../../plugins/async/plugin.go#L256) `Plugin.doStop`；分类S；阶段P5。
- 完整当前类型：`func(_ context.Context) error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/async/plugin.go:248](../../../plugins/async/plugin.go#L248) | `Stop`        | SDK生产  |

#### P0239

- 定义：[plugins/async/plugin.go:315](../../../plugins/async/plugin.go#L315) `Plugin.Submit`；分类S；阶段P5。
- 完整当前类型：`func(parentCtx context.Context, fn func(ctx context.Context), opts ...github.com/bamboo-services/bamboo-base-go/plugins/async/v2.TaskOption) (*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.Task, error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/async/v2.Pool`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0240

- 定义：[plugins/async/plugin.go:420](../../../plugins/async/plugin.go#L420) `Submit`；分类S；阶段P5。
- 完整当前类型：`func(parentCtx context.Context, fn func(ctx context.Context), opts ...github.com/bamboo-services/bamboo-base-go/plugins/async/v2.TaskOption) (*github.com/bamboo-services/bamboo-base-go/plugins/async/v2.Task, error)`。

| 位置                                                                                                       | 所在函数/声明                                                                     | 引用来源 |
| ---------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | -------- |
| [plugins/async/dependency_lifecycle_test.go:128](../../../plugins/async/dependency_lifecycle_test.go#L128) | `TestAsyncDependencyLifecycle_CaseA_AppForceCancelsTask_OrderlyShutdown`          | 单测     |
| [plugins/async/dependency_lifecycle_test.go:246](../../../plugins/async/dependency_lifecycle_test.go#L246) | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete` | 单测     |
| [plugins/async/plugin_test.go:107](../../../plugins/async/plugin_test.go#L107)                             | `TestPlugin_BasicExecution`                                                       | 单测     |
| [plugins/async/plugin_test.go:141](../../../plugins/async/plugin_test.go#L141)                             | `TestPlugin_ContextDetachedFromParentCancel`                                      | 单测     |
| [plugins/async/plugin_test.go:174](../../../plugins/async/plugin_test.go#L174)                             | `TestPlugin_RequestKeyCopied`                                                     | 单测     |
| [plugins/async/plugin_test.go:204](../../../plugins/async/plugin_test.go#L204)                             | `TestPlugin_TaskCancel`                                                           | 单测     |
| [plugins/async/plugin_test.go:228](../../../plugins/async/plugin_test.go#L228)                             | `TestPlugin_WaitWithTimeoutDoesNotCancelTask`                                     | 单测     |
| [plugins/async/plugin_test.go:266](../../../plugins/async/plugin_test.go#L266)                             | `TestPlugin_PanicRecovery`                                                        | 单测     |
| [plugins/async/plugin_test.go:302](../../../plugins/async/plugin_test.go#L302)                             | `TestPlugin_PoolFullRejection`                                                    | 单测     |
| [plugins/async/plugin_test.go:313](../../../plugins/async/plugin_test.go#L313)                             | `TestPlugin_PoolFullRejection`                                                    | 单测     |
| [plugins/async/plugin_test.go:324](../../../plugins/async/plugin_test.go#L324)                             | `TestPlugin_PoolFullRejection`                                                    | 单测     |
| [plugins/async/plugin_test.go:342](../../../plugins/async/plugin_test.go#L342)                             | `TestPlugin_RejectDuringStoppingAndClosed`                                        | 单测     |
| [plugins/async/plugin_test.go:362](../../../plugins/async/plugin_test.go#L362)                             | `TestPlugin_RejectDuringStoppingAndClosed`                                        | 单测     |
| [plugins/async/plugin_test.go:376](../../../plugins/async/plugin_test.go#L376)                             | `TestPlugin_RejectDuringStoppingAndClosed`                                        | 单测     |

#### P0241

- 定义：[plugins/async/task.go:106](../../../plugins/async/task.go#L106) `Task.Wait`；分类S；阶段P5。
- 完整当前类型：`func(ctx ...context.Context) error`。

| 位置                                                                                                                 | 所在函数/声明                                                                     | 引用来源   |
| -------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | ---------- |
| [examples/v2/grpc-cron-async/main.go:43](../../../examples/v2/grpc-cron-async/main.go#L43)                           | `Run`                                                                             | 仓内消费者 |
| [plugins/async/dependency_lifecycle_test.go:136](../../../plugins/async/dependency_lifecycle_test.go#L136)           | `TestAsyncDependencyLifecycle_CaseA_AppForceCancelsTask_OrderlyShutdown`          | 单测       |
| [plugins/async/dependency_lifecycle_test.go:289](../../../plugins/async/dependency_lifecycle_test.go#L289)           | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete` | 单测       |
| [plugins/async/dependency_lifecycle_test.go:341](../../../plugins/async/dependency_lifecycle_test.go#L341)           | `TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete` | 单测       |
| [plugins/async/dependency_lifecycle_test.go:456](../../../plugins/async/dependency_lifecycle_test.go#L456)           | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated`   | 单测       |
| [plugins/async/dependency_lifecycle_test.go:472](../../../plugins/async/dependency_lifecycle_test.go#L472)           | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated`   | 单测       |
| [plugins/async/dependency_lifecycle_test.go:496](../../../plugins/async/dependency_lifecycle_test.go#L496)           | `TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated`   | 单测       |
| [plugins/async/plugin_test.go:124](../../../plugins/async/plugin_test.go#L124)                                       | `TestPlugin_BasicExecution`                                                       | 单测       |
| [plugins/async/plugin_test.go:162](../../../plugins/async/plugin_test.go#L162)                                       | `TestPlugin_ContextDetachedFromParentCancel`                                      | 单测       |
| [plugins/async/plugin_test.go:243](../../../plugins/async/plugin_test.go#L243)                                       | `TestPlugin_WaitWithTimeoutDoesNotCancelTask`                                     | 单测       |
| [plugins/async/plugin_test.go:255](../../../plugins/async/plugin_test.go#L255)                                       | `TestPlugin_WaitWithTimeoutDoesNotCancelTask`                                     | 单测       |
| [plugins/async/plugin_test.go:273](../../../plugins/async/plugin_test.go#L273)                                       | `TestPlugin_PanicRecovery`                                                        | 单测       |
| [plugins/async/plugin_test.go:320](../../../plugins/async/plugin_test.go#L320)                                       | `TestPlugin_PoolFullRejection`                                                    | 单测       |
| [plugins/async/plugin_test.go:330](../../../plugins/async/plugin_test.go#L330)                                       | `TestPlugin_PoolFullRejection`                                                    | 单测       |
| [plugins/async/plugin_test.go:369](../../../plugins/async/plugin_test.go#L369)                                       | `TestPlugin_RejectDuringStoppingAndClosed`                                        | 单测       |
| [plugins/async/plugin_test.go:410](../../../plugins/async/plugin_test.go#L410)                                       | `TestPlugin_ForceCancelReceivedByTask`                                            | 单测       |
| [plugins/async/plugin_test.go:445](../../../plugins/async/plugin_test.go#L445)                                       | `TestPlugin_IgnoreCancelTaskMakesStopWait`                                        | 单测       |
| [plugins/async/plugin_test.go:500](../../../plugins/async/plugin_test.go#L500)                                       | `TestPlugin_SubmitRollbackOnFailure`                                              | 单测       |
| [plugins/async/plugin_test.go:532](../../../plugins/async/plugin_test.go#L532)                                       | `TestPlugin_CrossAppAndBackgroundRejection`                                       | 单测       |
| [plugins/async/task.go:125](../../../plugins/async/task.go#L125)                                                     | `WaitFor`                                                                         | SDK生产    |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:93](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L93)   | `TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown`             | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:221](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L221) | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`     | 仓内消费者 |
| [test/consumer/v2-sqlite-async/lifecycle_test.go:349](../../../test/consumer/v2-sqlite-async/lifecycle_test.go#L349) | `TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete`     | 仓内消费者 |

#### P0242

- 定义：[plugins/async/task.go:122](../../../plugins/async/task.go#L122) `Task.WaitFor`；分类S；阶段P5。
- 完整当前类型：`func(timeout time.Duration) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0243

- 定义：[plugins/cache/config.go:24](../../../plugins/cache/config.go#L24) `Config.validationErr`；分类S；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明  | 引用来源 |
| -------------------------------------------------------------------- | -------------- | -------- |
| [plugins/cache/config.go:191](../../../plugins/cache/config.go#L191) | `applyFromEnv` | SDK生产  |
| [plugins/cache/config.go:191](../../../plugins/cache/config.go#L191) | `applyFromEnv` | SDK生产  |
| [plugins/cache/config.go:199](../../../plugins/cache/config.go#L199) | `applyFromEnv` | SDK生产  |
| [plugins/cache/config.go:199](../../../plugins/cache/config.go#L199) | `applyFromEnv` | SDK生产  |
| [plugins/cache/config.go:214](../../../plugins/cache/config.go#L214) | `applyFromEnv` | SDK生产  |
| [plugins/cache/config.go:214](../../../plugins/cache/config.go#L214) | `applyFromEnv` | SDK生产  |
| [plugins/cache/config.go:220](../../../plugins/cache/config.go#L220) | `applyFromEnv` | SDK生产  |
| [plugins/cache/config.go:220](../../../plugins/cache/config.go#L220) | `applyFromEnv` | SDK生产  |
| [plugins/cache/config.go:228](../../../plugins/cache/config.go#L228) | `applyFromEnv` | SDK生产  |
| [plugins/cache/config.go:228](../../../plugins/cache/config.go#L228) | `applyFromEnv` | SDK生产  |
| [plugins/cache/plugin.go:51](../../../plugins/cache/plugin.go#L51)   | `New`          | SDK生产  |
| [plugins/cache/plugin.go:52](../../../plugins/cache/plugin.go#L52)   | `New`          | SDK生产  |

#### P0244

- 定义：[plugins/cache/config.go:243](../../../plugins/cache/config.go#L243) `Get`；分类A；阶段P5。
- 完整当前类型：`func(ctx context.Context) (*github.com/bamboo-services/bamboo-base-go/plugins/cache/v2.Manager, *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error)`。

| 位置                                                                           | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [plugins/cache/config.go:251](../../../plugins/cache/config.go#L251)           | `MustGet`                           | SDK生产  |
| [plugins/cache/plugin_test.go:175](../../../plugins/cache/plugin_test.go#L175) | `TestNew_TwoMemoryPluginsIsolation` | 单测     |
| [plugins/cache/plugin_test.go:179](../../../plugins/cache/plugin_test.go#L179) | `TestNew_TwoMemoryPluginsIsolation` | 单测     |

#### P0245

- 定义：[plugins/cache/driver/codec.go:11](../../../plugins/cache/driver/codec.go#L11) `Codec.Marshal`；分类S；阶段P5。
- 完整当前类型：`func(v any) ([]byte, error)`。

| 位置                                                                           | 所在函数/声明   | 引用来源 |
| ------------------------------------------------------------------------------ | --------------- | -------- |
| [plugins/cache/memory/hash.go:70](../../../plugins/cache/memory/hash.go#L70)   | `Set`           | SDK生产  |
| [plugins/cache/memory/hash.go:140](../../../plugins/cache/memory/hash.go#L140) | `GetAllStruct`  | SDK生产  |
| [plugins/cache/memory/hash.go:173](../../../plugins/cache/memory/hash.go#L173) | `SetAll`        | SDK生产  |
| [plugins/cache/memory/hash.go:244](../../../plugins/cache/memory/hash.go#L244) | `SetAllStruct`  | SDK生产  |
| [plugins/cache/memory/key.go:64](../../../plugins/cache/memory/key.go#L64)     | `Set`           | SDK生产  |
| [plugins/cache/memory/list.go:59](../../../plugins/cache/memory/list.go#L59)   | `Prepend`       | SDK生产  |
| [plugins/cache/memory/list.go:115](../../../plugins/cache/memory/list.go#L115) | `Append`        | SDK生产  |
| [plugins/cache/memory/list.go:281](../../../plugins/cache/memory/list.go#L281) | `Remove`        | SDK生产  |
| [plugins/cache/memory/set.go:31](../../../plugins/cache/memory/set.go#L31)     | `encodeMember`  | SDK生产  |
| [plugins/cache/memory/set.go:55](../../../plugins/cache/memory/set.go#L55)     | `Add`           | SDK生产  |
| [plugins/cache/memory/set.go:163](../../../plugins/cache/memory/set.go#L163)   | `Remove`        | SDK生产  |
| [plugins/cache/redis/hash.go:67](../../../plugins/cache/redis/hash.go#L67)     | `Set`           | SDK生产  |
| [plugins/cache/redis/hash.go:143](../../../plugins/cache/redis/hash.go#L143)   | `GetAllStruct`  | SDK生产  |
| [plugins/cache/redis/hash.go:167](../../../plugins/cache/redis/hash.go#L167)   | `SetAll`        | SDK生产  |
| [plugins/cache/redis/hash.go:192](../../../plugins/cache/redis/hash.go#L192)   | `SetAllStruct`  | SDK生产  |
| [plugins/cache/redis/hash.go:203](../../../plugins/cache/redis/hash.go#L203)   | `SetAllStruct`  | SDK生产  |
| [plugins/cache/redis/key.go:62](../../../plugins/cache/redis/key.go#L62)       | `Set`           | SDK生产  |
| [plugins/cache/redis/list.go:39](../../../plugins/cache/redis/list.go#L39)     | `encodeValues`  | SDK生产  |
| [plugins/cache/redis/list.go:207](../../../plugins/cache/redis/list.go#L207)   | `Remove`        | SDK生产  |
| [plugins/cache/redis/set.go:38](../../../plugins/cache/redis/set.go#L38)       | `encodeMembers` | SDK生产  |
| [plugins/cache/redis/set.go:105](../../../plugins/cache/redis/set.go#L105)     | `IsMember`      | SDK生产  |

#### P0246

- 定义：[plugins/cache/driver/codec.go:14](../../../plugins/cache/driver/codec.go#L14) `Codec.Unmarshal`；分类S；阶段P5。
- 完整当前类型：`func(data []byte, v any) error`。

| 位置                                                                           | 所在函数/声明  | 引用来源 |
| ------------------------------------------------------------------------------ | -------------- | -------- |
| [plugins/cache/memory/hash.go:49](../../../plugins/cache/memory/hash.go#L49)   | `Get`          | SDK生产  |
| [plugins/cache/memory/hash.go:123](../../../plugins/cache/memory/hash.go#L123) | `GetAll`       | SDK生产  |
| [plugins/cache/memory/hash.go:144](../../../plugins/cache/memory/hash.go#L144) | `GetAllStruct` | SDK生产  |
| [plugins/cache/memory/hash.go:249](../../../plugins/cache/memory/hash.go#L249) | `SetAllStruct` | SDK生产  |
| [plugins/cache/memory/key.go:44](../../../plugins/cache/memory/key.go#L44)     | `Get`          | SDK生产  |
| [plugins/cache/memory/list.go:173](../../../plugins/cache/memory/list.go#L173) | `Range`        | SDK生产  |
| [plugins/cache/memory/list.go:197](../../../plugins/cache/memory/list.go#L197) | `Index`        | SDK生产  |
| [plugins/cache/memory/list.go:234](../../../plugins/cache/memory/list.go#L234) | `Pop`          | SDK生产  |
| [plugins/cache/memory/list.go:263](../../../plugins/cache/memory/list.go#L263) | `PopLast`      | SDK生产  |
| [plugins/cache/memory/set.go:113](../../../plugins/cache/memory/set.go#L113)   | `Members`      | SDK生产  |
| [plugins/cache/redis/hash.go:50](../../../plugins/cache/redis/hash.go#L50)     | `Get`          | SDK生产  |
| [plugins/cache/redis/hash.go:121](../../../plugins/cache/redis/hash.go#L121)   | `GetAll`       | SDK生产  |
| [plugins/cache/redis/hash.go:147](../../../plugins/cache/redis/hash.go#L147)   | `GetAllStruct` | SDK生产  |
| [plugins/cache/redis/hash.go:197](../../../plugins/cache/redis/hash.go#L197)   | `SetAllStruct` | SDK生产  |
| [plugins/cache/redis/key.go:44](../../../plugins/cache/redis/key.go#L44)       | `Get`          | SDK生产  |
| [plugins/cache/redis/list.go:141](../../../plugins/cache/redis/list.go#L141)   | `Range`        | SDK生产  |
| [plugins/cache/redis/list.go:159](../../../plugins/cache/redis/list.go#L159)   | `Index`        | SDK生产  |
| [plugins/cache/redis/list.go:180](../../../plugins/cache/redis/list.go#L180)   | `Pop`          | SDK生产  |
| [plugins/cache/redis/list.go:196](../../../plugins/cache/redis/list.go#L196)   | `PopLast`      | SDK生产  |
| [plugins/cache/redis/set.go:95](../../../plugins/cache/redis/set.go#L95)       | `Members`      | SDK生产  |

#### P0247

- 定义：[plugins/cache/driver/codec.go:23](../../../plugins/cache/driver/codec.go#L23) `JSONCodec.Marshal`；分类S；阶段P5。
- 完整当前类型：`func(v any) ([]byte, error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.Codec`。

| 位置                                                                                               | 所在函数/声明      | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ------------------ | -------- |
| [plugins/cache/driver/driver_test.go:30](../../../plugins/cache/driver/driver_test.go#L30)         | `TestJSONCodec`    | 单测     |
| [plugins/cache/driver/example_test.go:62](../../../plugins/cache/driver/example_test.go#L62)       | `ExampleJSONCodec` | 单测     |
| [plugins/cache/redis/integration_test.go:41](../../../plugins/cache/redis/integration_test.go#L41) | `Marshal`          | 单测     |

#### P0248

- 定义：[plugins/cache/driver/codec.go:26](../../../plugins/cache/driver/codec.go#L26) `JSONCodec.Unmarshal`；分类S；阶段P5。
- 完整当前类型：`func(data []byte, v any) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.Codec`。

| 位置                                                                                               | 所在函数/声明      | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ------------------ | -------- |
| [plugins/cache/driver/driver_test.go:44](../../../plugins/cache/driver/driver_test.go#L44)         | `TestJSONCodec`    | 单测     |
| [plugins/cache/driver/driver_test.go:52](../../../plugins/cache/driver/driver_test.go#L52)         | `TestJSONCodec`    | 单测     |
| [plugins/cache/driver/example_test.go:68](../../../plugins/cache/driver/example_test.go#L68)       | `ExampleJSONCodec` | 单测     |
| [plugins/cache/redis/integration_test.go:48](../../../plugins/cache/redis/integration_test.go#L48) | `Unmarshal`        | 单测     |

#### P0249

- 定义：[plugins/cache/driver/interface.go:17](../../../plugins/cache/driver/interface.go#L17) `KeyCache.Get`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (*V, bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache[K, V any]`。

| 位置                                                                                                                   | 所在函数/声明                       | 引用来源   |
| ---------------------------------------------------------------------------------------------------------------------- | ----------------------------------- | ---------- |
| [examples/v2/http-db-cache/main.go:88](../../../examples/v2/http-db-cache/main.go#L88)                                 | `main`                              | 仓内消费者 |
| [plugins/cache/example_test.go:33](../../../plugins/cache/example_test.go#L33)                                         | `ExampleKeyCacheOf`                 | 单测       |
| [plugins/cache/memory/concurrent_test.go:139](../../../plugins/cache/memory/concurrent_test.go#L139)                   | `TestConcurrentKeyCacheGetSet`      | 单测       |
| [plugins/cache/memory/example_test.go:45](../../../plugins/cache/memory/example_test.go#L45)                           | `ExampleNewKeyCache`                | 单测       |
| [plugins/cache/memory/memory_test.go:21](../../../plugins/cache/memory/memory_test.go#L21)                             | `TestMemoryKeyCache`                | 单测       |
| [plugins/cache/memory/memory_test.go:34](../../../plugins/cache/memory/memory_test.go#L34)                             | `TestMemoryKeyCache`                | 单测       |
| [plugins/cache/memory/memory_test.go:42](../../../plugins/cache/memory/memory_test.go#L42)                             | `TestMemoryKeyCache`                | 单测       |
| [plugins/cache/plugin_test.go:194](../../../plugins/cache/plugin_test.go#L194)                                         | `TestNew_TwoMemoryPluginsIsolation` | 单测       |
| [plugins/cache/redis/example_test.go:27](../../../plugins/cache/redis/example_test.go#L27)                             | `ExampleNewKeyCache`                | 单测       |
| [plugins/cache/redis/integration_test.go:176](../../../plugins/cache/redis/integration_test.go#L176)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:200](../../../plugins/cache/redis/integration_test.go#L200)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:236](../../../plugins/cache/redis/integration_test.go#L236)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:246](../../../plugins/cache/redis/integration_test.go#L246)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:275](../../../plugins/cache/redis/integration_test.go#L275)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:293](../../../plugins/cache/redis/integration_test.go#L293)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:308](../../../plugins/cache/redis/integration_test.go#L308)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:121](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L121) | `TestHTTPDBCacheLifecycle_EndToEnd` | 仓内消费者 |

#### P0250

- 定义：[plugins/cache/driver/interface.go:23](../../../plugins/cache/driver/interface.go#L23) `KeyCache.Set`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, value *V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache[K, V any]`。

| 位置                                                                                                                   | 所在函数/声明                       | 引用来源   |
| ---------------------------------------------------------------------------------------------------------------------- | ----------------------------------- | ---------- |
| [examples/v2/http-db-cache/main.go:83](../../../examples/v2/http-db-cache/main.go#L83)                                 | `main`                              | 仓内消费者 |
| [plugins/cache/example_test.go:29](../../../plugins/cache/example_test.go#L29)                                         | `ExampleKeyCacheOf`                 | 单测       |
| [plugins/cache/memory/concurrent_test.go:137](../../../plugins/cache/memory/concurrent_test.go#L137)                   | `TestConcurrentKeyCacheGetSet`      | 单测       |
| [plugins/cache/memory/example_test.go:41](../../../plugins/cache/memory/example_test.go#L41)                           | `ExampleNewKeyCache`                | 单测       |
| [plugins/cache/memory/memory_test.go:18](../../../plugins/cache/memory/memory_test.go#L18)                             | `TestMemoryKeyCache`                | 单测       |
| [plugins/cache/memory/memory_test.go:40](../../../plugins/cache/memory/memory_test.go#L40)                             | `TestMemoryKeyCache`                | 单测       |
| [plugins/cache/plugin_test.go:193](../../../plugins/cache/plugin_test.go#L193)                                         | `TestNew_TwoMemoryPluginsIsolation` | 单测       |
| [plugins/cache/redis/example_test.go:26](../../../plugins/cache/redis/example_test.go#L26)                             | `ExampleNewKeyCache`                | 单测       |
| [plugins/cache/redis/integration_test.go:186](../../../plugins/cache/redis/integration_test.go#L186)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:215](../../../plugins/cache/redis/integration_test.go#L215)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:229](../../../plugins/cache/redis/integration_test.go#L229)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:233](../../../plugins/cache/redis/integration_test.go#L233)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:243](../../../plugins/cache/redis/integration_test.go#L243)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:256](../../../plugins/cache/redis/integration_test.go#L256)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:266](../../../plugins/cache/redis/integration_test.go#L266)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:286](../../../plugins/cache/redis/integration_test.go#L286)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [plugins/cache/redis/integration_test.go:305](../../../plugins/cache/redis/integration_test.go#L305)                   | `TestIntegration_RedisKeyCache`     | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:116](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L116) | `TestHTTPDBCacheLifecycle_EndToEnd` | 仓内消费者 |

#### P0251

- 定义：[plugins/cache/driver/interface.go:26](../../../plugins/cache/driver/interface.go#L26) `KeyCache.Exists`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                   | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:26](../../../plugins/cache/memory/memory_test.go#L26)           | `TestMemoryKeyCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:167](../../../plugins/cache/redis/integration_test.go#L167) | `TestIntegration_RedisKeyCache` | 单测     |
| [plugins/cache/redis/integration_test.go:191](../../../plugins/cache/redis/integration_test.go#L191) | `TestIntegration_RedisKeyCache` | 单测     |
| [plugins/cache/redis/integration_test.go:218](../../../plugins/cache/redis/integration_test.go#L218) | `TestIntegration_RedisKeyCache` | 单测     |
| [plugins/cache/redis/integration_test.go:259](../../../plugins/cache/redis/integration_test.go#L259) | `TestIntegration_RedisKeyCache` | 单测     |

#### P0252

- 定义：[plugins/cache/driver/interface.go:29](../../../plugins/cache/driver/interface.go#L29) `KeyCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                   | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:31](../../../plugins/cache/memory/memory_test.go#L31)           | `TestMemoryKeyCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:252](../../../plugins/cache/redis/integration_test.go#L252) | `TestIntegration_RedisKeyCache` | 单测     |

#### P0253

- 定义：[plugins/cache/driver/interface.go:45](../../../plugins/cache/driver/interface.go#L45) `HashCache.Get`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, field F) (*V, bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache[K any, F comparable, V, S any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/example_test.go:63](../../../plugins/cache/example_test.go#L63)                       | `ExampleHashCacheOf`             | 单测     |
| [plugins/cache/memory/example_test.go:70](../../../plugins/cache/memory/example_test.go#L70)         | `ExampleNewHashCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:58](../../../plugins/cache/memory/memory_test.go#L58)           | `TestMemoryHashCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:74](../../../plugins/cache/memory/memory_test.go#L74)           | `TestMemoryHashCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:328](../../../plugins/cache/redis/integration_test.go#L328) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:398](../../../plugins/cache/redis/integration_test.go#L398) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:407](../../../plugins/cache/redis/integration_test.go#L407) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:439](../../../plugins/cache/redis/integration_test.go#L439) | `TestIntegration_RedisHashCache` | 单测     |

#### P0254

- 定义：[plugins/cache/driver/interface.go:51](../../../plugins/cache/driver/interface.go#L51) `HashCache.Set`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, field F, value *V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache[K any, F comparable, V, S any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/example_test.go:59](../../../plugins/cache/example_test.go#L59)                       | `ExampleHashCacheOf`             | 单测     |
| [plugins/cache/memory/concurrent_test.go:34](../../../plugins/cache/memory/concurrent_test.go#L34)   | `TestConcurrentHashCacheWrite`   | 单测     |
| [plugins/cache/memory/example_test.go:66](../../../plugins/cache/memory/example_test.go#L66)         | `ExampleNewHashCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:55](../../../plugins/cache/memory/memory_test.go#L55)           | `TestMemoryHashCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:56](../../../plugins/cache/memory/memory_test.go#L56)           | `TestMemoryHashCache`            | 单测     |
| [plugins/cache/redis/example_test.go:46](../../../plugins/cache/redis/example_test.go#L46)           | `ExampleNewHashCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:325](../../../plugins/cache/redis/integration_test.go#L325) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:338](../../../plugins/cache/redis/integration_test.go#L338) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:391](../../../plugins/cache/redis/integration_test.go#L391) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:395](../../../plugins/cache/redis/integration_test.go#L395) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:404](../../../plugins/cache/redis/integration_test.go#L404) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:413](../../../plugins/cache/redis/integration_test.go#L413) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:432](../../../plugins/cache/redis/integration_test.go#L432) | `TestIntegration_RedisHashCache` | 单测     |

#### P0255

- 定义：[plugins/cache/driver/interface.go:56](../../../plugins/cache/driver/interface.go#L56) `HashCache.GetAll`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (map[F]V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache[K any, F comparable, V, S any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/concurrent_test.go:43](../../../plugins/cache/memory/concurrent_test.go#L43)   | `TestConcurrentHashCacheWrite`   | 单测     |
| [plugins/cache/memory/memory_test.go:63](../../../plugins/cache/memory/memory_test.go#L63)           | `TestMemoryHashCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:80](../../../plugins/cache/memory/memory_test.go#L80)           | `TestMemoryHashCache`            | 单测     |
| [plugins/cache/redis/example_test.go:47](../../../plugins/cache/redis/example_test.go#L47)           | `ExampleNewHashCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:356](../../../plugins/cache/redis/integration_test.go#L356) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:368](../../../plugins/cache/redis/integration_test.go#L368) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:446](../../../plugins/cache/redis/integration_test.go#L446) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/redis_test.go:63](../../../plugins/cache/redis/redis_test.go#L63)               | `TestHashCacheGetAllTypeCheck`   | 单测     |

#### P0256

- 定义：[plugins/cache/driver/interface.go:59](../../../plugins/cache/driver/interface.go#L59) `HashCache.GetAllStruct`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (S, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache[K any, F comparable, V, S any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/redis/integration_test.go:380](../../../plugins/cache/redis/integration_test.go#L380) | `TestIntegration_RedisHashCache` | 单测     |

#### P0257

- 定义：[plugins/cache/driver/interface.go:64](../../../plugins/cache/driver/interface.go#L64) `HashCache.SetAll`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, fields map[F]*V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache[K any, F comparable, V, S any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/redis/integration_test.go:352](../../../plugins/cache/redis/integration_test.go#L352) | `TestIntegration_RedisHashCache` | 单测     |

#### P0258

- 定义：[plugins/cache/driver/interface.go:67](../../../plugins/cache/driver/interface.go#L67) `HashCache.SetAllStruct`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, value S, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache[K any, F comparable, V, S any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/redis/integration_test.go:376](../../../plugins/cache/redis/integration_test.go#L376) | `TestIntegration_RedisHashCache` | 单测     |

#### P0259

- 定义：[plugins/cache/driver/interface.go:70](../../../plugins/cache/driver/interface.go#L70) `HashCache.Exists`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, field F) (bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache[K any, F comparable, V, S any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:68](../../../plugins/cache/memory/memory_test.go#L68)           | `TestMemoryHashCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:332](../../../plugins/cache/redis/integration_test.go#L332) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:341](../../../plugins/cache/redis/integration_test.go#L341) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:416](../../../plugins/cache/redis/integration_test.go#L416) | `TestIntegration_RedisHashCache` | 单测     |
| [plugins/cache/redis/integration_test.go:425](../../../plugins/cache/redis/integration_test.go#L425) | `TestIntegration_RedisHashCache` | 单测     |

#### P0260

- 定义：[plugins/cache/driver/interface.go:73](../../../plugins/cache/driver/interface.go#L73) `HashCache.Remove`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, fields ...F) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache[K any, F comparable, V, S any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:73](../../../plugins/cache/memory/memory_test.go#L73)           | `TestMemoryHashCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:365](../../../plugins/cache/redis/integration_test.go#L365) | `TestIntegration_RedisHashCache` | 单测     |

#### P0261

- 定义：[plugins/cache/driver/interface.go:76](../../../plugins/cache/driver/interface.go#L76) `HashCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache[K any, F comparable, V, S any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:79](../../../plugins/cache/memory/memory_test.go#L79)           | `TestMemoryHashCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:422](../../../plugins/cache/redis/integration_test.go#L422) | `TestIntegration_RedisHashCache` | 单测     |

#### P0262

- 定义：[plugins/cache/driver/interface.go:90](../../../plugins/cache/driver/interface.go#L90) `SetCache.Add`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, members []V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                   | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| [plugins/cache/memory/concurrent_test.go:70](../../../plugins/cache/memory/concurrent_test.go#L70)   | `TestConcurrentSetCacheAdd`     | 单测     |
| [plugins/cache/memory/example_test.go:90](../../../plugins/cache/memory/example_test.go#L90)         | `ExampleNewSetCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:93](../../../plugins/cache/memory/memory_test.go#L93)           | `TestMemorySetCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:463](../../../plugins/cache/redis/integration_test.go#L463) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:499](../../../plugins/cache/redis/integration_test.go#L499) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:503](../../../plugins/cache/redis/integration_test.go#L503) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:512](../../../plugins/cache/redis/integration_test.go#L512) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:522](../../../plugins/cache/redis/integration_test.go#L522) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:546](../../../plugins/cache/redis/integration_test.go#L546) | `TestIntegration_RedisSetCache` | 单测     |

#### P0263

- 定义：[plugins/cache/driver/interface.go:93](../../../plugins/cache/driver/interface.go#L93) `SetCache.Members`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) ([]V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                   | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:104](../../../plugins/cache/memory/memory_test.go#L104)         | `TestMemorySetCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:482](../../../plugins/cache/redis/integration_test.go#L482) | `TestIntegration_RedisSetCache` | 单测     |

#### P0264

- 定义：[plugins/cache/driver/interface.go:96](../../../plugins/cache/driver/interface.go#L96) `SetCache.IsMember`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, member V) (bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                   | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| [plugins/cache/memory/example_test.go:94](../../../plugins/cache/memory/example_test.go#L94)         | `ExampleNewSetCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:99](../../../plugins/cache/memory/memory_test.go#L99)           | `TestMemorySetCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:472](../../../plugins/cache/redis/integration_test.go#L472) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:477](../../../plugins/cache/redis/integration_test.go#L477) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:506](../../../plugins/cache/redis/integration_test.go#L506) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:515](../../../plugins/cache/redis/integration_test.go#L515) | `TestIntegration_RedisSetCache` | 单测     |

#### P0265

- 定义：[plugins/cache/driver/interface.go:99](../../../plugins/cache/driver/interface.go#L99) `SetCache.Count`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (int64, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                   | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| [plugins/cache/memory/concurrent_test.go:79](../../../plugins/cache/memory/concurrent_test.go#L79)   | `TestConcurrentSetCacheAdd`     | 单测     |
| [plugins/cache/memory/example_test.go:98](../../../plugins/cache/memory/example_test.go#L98)         | `ExampleNewSetCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:94](../../../plugins/cache/memory/memory_test.go#L94)           | `TestMemorySetCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:110](../../../plugins/cache/memory/memory_test.go#L110)         | `TestMemorySetCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:467](../../../plugins/cache/redis/integration_test.go#L467) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:491](../../../plugins/cache/redis/integration_test.go#L491) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:529](../../../plugins/cache/redis/integration_test.go#L529) | `TestIntegration_RedisSetCache` | 单测     |
| [plugins/cache/redis/integration_test.go:538](../../../plugins/cache/redis/integration_test.go#L538) | `TestIntegration_RedisSetCache` | 单测     |

#### P0266

- 定义：[plugins/cache/driver/interface.go:102](../../../plugins/cache/driver/interface.go#L102) `SetCache.Remove`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, members ...V) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                   | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:109](../../../plugins/cache/memory/memory_test.go#L109)         | `TestMemorySetCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:488](../../../plugins/cache/redis/integration_test.go#L488) | `TestIntegration_RedisSetCache` | 单测     |

#### P0267

- 定义：[plugins/cache/driver/interface.go:105](../../../plugins/cache/driver/interface.go#L105) `SetCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                   | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| [plugins/cache/redis/integration_test.go:535](../../../plugins/cache/redis/integration_test.go#L535) | `TestIntegration_RedisSetCache` | 单测     |

#### P0268

- 定义：[plugins/cache/driver/interface.go:119](../../../plugins/cache/driver/interface.go#L119) `ListCache.Prepend`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, values []V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:135](../../../plugins/cache/memory/memory_test.go#L135)         | `TestMemoryListCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:566](../../../plugins/cache/redis/integration_test.go#L566) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:641](../../../plugins/cache/redis/integration_test.go#L641) | `TestIntegration_RedisListCache` | 单测     |

#### P0269

- 定义：[plugins/cache/driver/interface.go:122](../../../plugins/cache/driver/interface.go#L122) `ListCache.Append`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, values []V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/concurrent_test.go:103](../../../plugins/cache/memory/concurrent_test.go#L103) | `TestConcurrentListCacheAppend`  | 单测     |
| [plugins/cache/memory/example_test.go:117](../../../plugins/cache/memory/example_test.go#L117)       | `ExampleNewListCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:123](../../../plugins/cache/memory/memory_test.go#L123)         | `TestMemoryListCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:562](../../../plugins/cache/redis/integration_test.go#L562) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:614](../../../plugins/cache/redis/integration_test.go#L614) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:628](../../../plugins/cache/redis/integration_test.go#L628) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:632](../../../plugins/cache/redis/integration_test.go#L632) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:651](../../../plugins/cache/redis/integration_test.go#L651) | `TestIntegration_RedisListCache` | 单测     |

#### P0270

- 定义：[plugins/cache/driver/interface.go:127](../../../plugins/cache/driver/interface.go#L127) `ListCache.Range`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, start int64, end int64) ([]V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/example_test.go:136](../../../plugins/cache/memory/example_test.go#L136)       | `ExampleNewListCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:129](../../../plugins/cache/memory/memory_test.go#L129)         | `TestMemoryListCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:136](../../../plugins/cache/memory/memory_test.go#L136)         | `TestMemoryListCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:576](../../../plugins/cache/redis/integration_test.go#L576) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:619](../../../plugins/cache/redis/integration_test.go#L619) | `TestIntegration_RedisListCache` | 单测     |

#### P0271

- 定义：[plugins/cache/driver/interface.go:130](../../../plugins/cache/driver/interface.go#L130) `ListCache.Index`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, index int64) (*V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/redis/integration_test.go:586](../../../plugins/cache/redis/integration_test.go#L586) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:590](../../../plugins/cache/redis/integration_test.go#L590) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:594](../../../plugins/cache/redis/integration_test.go#L594) | `TestIntegration_RedisListCache` | 单测     |

#### P0272

- 定义：[plugins/cache/driver/interface.go:133](../../../plugins/cache/driver/interface.go#L133) `ListCache.Len`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (int64, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/concurrent_test.go:112](../../../plugins/cache/memory/concurrent_test.go#L112) | `TestConcurrentListCacheAppend`  | 单测     |
| [plugins/cache/memory/example_test.go:126](../../../plugins/cache/memory/example_test.go#L126)       | `ExampleNewListCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:124](../../../plugins/cache/memory/memory_test.go#L124)         | `TestMemoryListCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:152](../../../plugins/cache/memory/memory_test.go#L152)         | `TestMemoryListCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:570](../../../plugins/cache/redis/integration_test.go#L570) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:608](../../../plugins/cache/redis/integration_test.go#L608) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:635](../../../plugins/cache/redis/integration_test.go#L635) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:644](../../../plugins/cache/redis/integration_test.go#L644) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:658](../../../plugins/cache/redis/integration_test.go#L658) | `TestIntegration_RedisListCache` | 单测     |
| [plugins/cache/redis/integration_test.go:667](../../../plugins/cache/redis/integration_test.go#L667) | `TestIntegration_RedisListCache` | 单测     |

#### P0273

- 定义：[plugins/cache/driver/interface.go:136](../../../plugins/cache/driver/interface.go#L136) `ListCache.Pop`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (*V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/example_test.go:122](../../../plugins/cache/memory/example_test.go#L122)       | `ExampleNewListCache`            | 单测     |
| [plugins/cache/memory/memory_test.go:141](../../../plugins/cache/memory/memory_test.go#L141)         | `TestMemoryListCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:600](../../../plugins/cache/redis/integration_test.go#L600) | `TestIntegration_RedisListCache` | 单测     |

#### P0274

- 定义：[plugins/cache/driver/interface.go:139](../../../plugins/cache/driver/interface.go#L139) `ListCache.PopLast`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (*V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:146](../../../plugins/cache/memory/memory_test.go#L146)         | `TestMemoryListCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:604](../../../plugins/cache/redis/integration_test.go#L604) | `TestIntegration_RedisListCache` | 单测     |

#### P0275

- 定义：[plugins/cache/driver/interface.go:146](../../../plugins/cache/driver/interface.go#L146) `ListCache.Remove`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, count int64, value V) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/memory/memory_test.go:151](../../../plugins/cache/memory/memory_test.go#L151)         | `TestMemoryListCache`            | 单测     |
| [plugins/cache/redis/integration_test.go:616](../../../plugins/cache/redis/integration_test.go#L616) | `TestIntegration_RedisListCache` | 单测     |

#### P0276

- 定义：[plugins/cache/driver/interface.go:149](../../../plugins/cache/driver/interface.go#L149) `ListCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache[K, V any]`。

| 位置                                                                                                 | 所在函数/声明                    | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------- |
| [plugins/cache/redis/integration_test.go:664](../../../plugins/cache/redis/integration_test.go#L664) | `TestIntegration_RedisListCache` | 单测     |

#### P0277

- 定义：[plugins/cache/memory/hash.go:34](../../../plugins/cache/memory/hash.go#L34) `HashCache.Get`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, field F) (*V, bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0278

- 定义：[plugins/cache/memory/hash.go:66](../../../plugins/cache/memory/hash.go#L66) `HashCache.Set`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, field F, value *V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0279

- 定义：[plugins/cache/memory/hash.go:110](../../../plugins/cache/memory/hash.go#L110) `HashCache.GetAll`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (map[F]V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。

| 位置                                                                           | 所在函数/声明  | 引用来源 |
| ------------------------------------------------------------------------------ | -------------- | -------- |
| [plugins/cache/memory/hash.go:136](../../../plugins/cache/memory/hash.go#L136) | `GetAllStruct` | SDK生产  |

#### P0280

- 定义：[plugins/cache/memory/hash.go:134](../../../plugins/cache/memory/hash.go#L134) `HashCache.GetAllStruct`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (S, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0281

- 定义：[plugins/cache/memory/hash.go:161](../../../plugins/cache/memory/hash.go#L161) `HashCache.SetAll`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, fields map[F]*V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。

| 位置                                                                           | 所在函数/声明  | 引用来源 |
| ------------------------------------------------------------------------------ | -------------- | -------- |
| [plugins/cache/memory/hash.go:257](../../../plugins/cache/memory/hash.go#L257) | `SetAllStruct` | SDK生产  |

#### P0282

- 定义：[plugins/cache/memory/hash.go:243](../../../plugins/cache/memory/hash.go#L243) `HashCache.SetAllStruct`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, value S, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0283

- 定义：[plugins/cache/memory/hash.go:261](../../../plugins/cache/memory/hash.go#L261) `HashCache.Exists`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, field F) (bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0284

- 定义：[plugins/cache/memory/hash.go:278](../../../plugins/cache/memory/hash.go#L278) `HashCache.Remove`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, fields ...F) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。

| 位置                                                                         | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/cache/memory/hash.go:68](../../../plugins/cache/memory/hash.go#L68) | `Set`         | SDK生产  |

#### P0285

- 定义：[plugins/cache/memory/hash.go:300](../../../plugins/cache/memory/hash.go#L300) `HashCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0286

- 定义：[plugins/cache/memory/key.go:33](../../../plugins/cache/memory/key.go#L33) `KeyCache.Get`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (*V, bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.KeyCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0287

- 定义：[plugins/cache/memory/key.go:59](../../../plugins/cache/memory/key.go#L59) `KeyCache.Set`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, value *V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.KeyCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0288

- 定义：[plugins/cache/memory/key.go:79](../../../plugins/cache/memory/key.go#L79) `KeyCache.Exists`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.KeyCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0289

- 定义：[plugins/cache/memory/key.go:84](../../../plugins/cache/memory/key.go#L84) `KeyCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.KeyCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`。

| 位置                                                                       | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/cache/memory/key.go:61](../../../plugins/cache/memory/key.go#L61) | `Set`         | SDK生产  |

#### P0290

- 定义：[plugins/cache/memory/list.go:53](../../../plugins/cache/memory/list.go#L53) `ListCache.Prepend`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, values []V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0291

- 定义：[plugins/cache/memory/list.go:109](../../../plugins/cache/memory/list.go#L109) `ListCache.Append`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, values []V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0292

- 定义：[plugins/cache/memory/list.go:154](../../../plugins/cache/memory/list.go#L154) `ListCache.Range`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, start int64, end int64) ([]V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0293

- 定义：[plugins/cache/memory/list.go:182](../../../plugins/cache/memory/list.go#L182) `ListCache.Index`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, index int64) (*V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0294

- 定义：[plugins/cache/memory/list.go:204](../../../plugins/cache/memory/list.go#L204) `ListCache.Len`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (int64, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0295

- 定义：[plugins/cache/memory/list.go:220](../../../plugins/cache/memory/list.go#L220) `ListCache.Pop`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (*V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0296

- 定义：[plugins/cache/memory/list.go:251](../../../plugins/cache/memory/list.go#L251) `ListCache.PopLast`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (*V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0297

- 定义：[plugins/cache/memory/list.go:280](../../../plugins/cache/memory/list.go#L280) `ListCache.Remove`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, count int64, value V) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0298

- 定义：[plugins/cache/memory/list.go:325](../../../plugins/cache/memory/list.go#L325) `ListCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0299

- 定义：[plugins/cache/memory/set.go:30](../../../plugins/cache/memory/set.go#L30) `SetCache.encodeMember`；分类S；阶段P5。
- 完整当前类型：`func(member V) (string, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.SetCache[K, V any]`。

| 位置                                                                         | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/cache/memory/set.go:132](../../../plugins/cache/memory/set.go#L132) | `IsMember`    | SDK生产  |

#### P0300

- 定义：[plugins/cache/memory/set.go:48](../../../plugins/cache/memory/set.go#L48) `SetCache.Add`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, members []V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0301

- 定义：[plugins/cache/memory/set.go:100](../../../plugins/cache/memory/set.go#L100) `SetCache.Members`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) ([]V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0302

- 定义：[plugins/cache/memory/set.go:122](../../../plugins/cache/memory/set.go#L122) `SetCache.IsMember`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, member V) (bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0303

- 定义：[plugins/cache/memory/set.go:141](../../../plugins/cache/memory/set.go#L141) `SetCache.Count`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (int64, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0304

- 定义：[plugins/cache/memory/set.go:157](../../../plugins/cache/memory/set.go#L157) `SetCache.Remove`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, members ...V) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0305

- 定义：[plugins/cache/memory/set.go:187](../../../plugins/cache/memory/set.go#L187) `SetCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0306

- 定义：[plugins/cache/plugin.go:23](../../../plugins/cache/plugin.go#L23) `pluginImpl.initErr`；分类S；阶段P5。
- 完整当前类型：`error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/cache/plugin.go:86](../../../plugins/cache/plugin.go#L86) | `New`         | SDK生产  |
| [plugins/cache/plugin.go:98](../../../plugins/cache/plugin.go#L98) | `Start`       | SDK生产  |
| [plugins/cache/plugin.go:99](../../../plugins/cache/plugin.go#L99) | `Start`       | SDK生产  |

#### P0307

- 定义：[plugins/cache/plugin.go:32](../../../plugins/cache/plugin.go#L32) `pluginImpl.stopErr`；分类S；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/cache/plugin.go:191](../../../plugins/cache/plugin.go#L191) | `Stop`        | SDK生产  |
| [plugins/cache/plugin.go:194](../../../plugins/cache/plugin.go#L194) | `Stop`        | SDK生产  |

#### P0308

- 定义：[plugins/cache/plugin.go:94](../../../plugins/cache/plugin.go#L94) `pluginImpl.Start`；分类S；阶段P3/P5。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0309

- 定义：[plugins/cache/plugin.go:166](../../../plugins/cache/plugin.go#L166) `pluginImpl.Stop`；分类S；阶段P3/P5。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0310

- 定义：[plugins/cache/plugin.go:177](../../../plugins/cache/plugin.go#L177) `Stop.errs`；分类S；阶段P5。
- 完整当前类型：`[]error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/cache/plugin.go:187](../../../plugins/cache/plugin.go#L187) | `Stop`        | SDK生产  |
| [plugins/cache/plugin.go:187](../../../plugins/cache/plugin.go#L187) | `Stop`        | SDK生产  |
| [plugins/cache/plugin.go:191](../../../plugins/cache/plugin.go#L191) | `Stop`        | SDK生产  |

#### P0311

- 定义：[plugins/cache/redis/hash.go:40](../../../plugins/cache/redis/hash.go#L40) `HashCache.Get`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, field F) (*V, bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0312

- 定义：[plugins/cache/redis/hash.go:63](../../../plugins/cache/redis/hash.go#L63) `HashCache.Set`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, field F, value *V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0313

- 定义：[plugins/cache/redis/hash.go:108](../../../plugins/cache/redis/hash.go#L108) `HashCache.GetAll`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (map[F]V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。

| 位置                                                                         | 所在函数/声明  | 引用来源 |
| ---------------------------------------------------------------------------- | -------------- | -------- |
| [plugins/cache/redis/hash.go:136](../../../plugins/cache/redis/hash.go#L136) | `GetAllStruct` | SDK生产  |

#### P0314

- 定义：[plugins/cache/redis/hash.go:134](../../../plugins/cache/redis/hash.go#L134) `HashCache.GetAllStruct`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (S, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0315

- 定义：[plugins/cache/redis/hash.go:157](../../../plugins/cache/redis/hash.go#L157) `HashCache.SetAll`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, fields map[F]*V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0316

- 定义：[plugins/cache/redis/hash.go:191](../../../plugins/cache/redis/hash.go#L191) `HashCache.SetAllStruct`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, value S, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0317

- 定义：[plugins/cache/redis/hash.go:223](../../../plugins/cache/redis/hash.go#L223) `HashCache.Exists`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, field F) (bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0318

- 定义：[plugins/cache/redis/hash.go:232](../../../plugins/cache/redis/hash.go#L232) `HashCache.Remove`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, fields ...F) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。

| 位置                                                                       | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/cache/redis/hash.go:65](../../../plugins/cache/redis/hash.go#L65) | `Set`         | SDK生产  |

#### P0319

- 定义：[plugins/cache/redis/hash.go:244](../../../plugins/cache/redis/hash.go#L244) `HashCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.HashCache[K any, F comparable, V, S any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.HashCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0320

- 定义：[plugins/cache/redis/key.go:34](../../../plugins/cache/redis/key.go#L34) `KeyCache.Get`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (*V, bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.KeyCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0321

- 定义：[plugins/cache/redis/key.go:58](../../../plugins/cache/redis/key.go#L58) `KeyCache.Set`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, value *V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.KeyCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0322

- 定义：[plugins/cache/redis/key.go:96](../../../plugins/cache/redis/key.go#L96) `KeyCache.Exists`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.KeyCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0323

- 定义：[plugins/cache/redis/key.go:105](../../../plugins/cache/redis/key.go#L105) `KeyCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.KeyCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.KeyCache`。

| 位置                                                                     | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/cache/redis/key.go:60](../../../plugins/cache/redis/key.go#L60) | `Set`         | SDK生产  |

#### P0324

- 定义：[plugins/cache/redis/list.go:36](../../../plugins/cache/redis/list.go#L36) `ListCache.encodeValues`；分类S；阶段P5。
- 完整当前类型：`func(values []V) ([]any, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。

| 位置                                                                         | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/cache/redis/list.go:79](../../../plugins/cache/redis/list.go#L79)   | `Prepend`     | SDK生产  |
| [plugins/cache/redis/list.go:118](../../../plugins/cache/redis/list.go#L118) | `Append`      | SDK生产  |

#### P0325

- 定义：[plugins/cache/redis/list.go:56](../../../plugins/cache/redis/list.go#L56) `ListCache.Prepend`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, values []V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0326

- 定义：[plugins/cache/redis/list.go:99](../../../plugins/cache/redis/list.go#L99) `ListCache.Append`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, values []V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0327

- 定义：[plugins/cache/redis/list.go:133](../../../plugins/cache/redis/list.go#L133) `ListCache.Range`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, start int64, end int64) ([]V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0328

- 定义：[plugins/cache/redis/list.go:150](../../../plugins/cache/redis/list.go#L150) `ListCache.Index`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, index int64) (*V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0329

- 定义：[plugins/cache/redis/list.go:166](../../../plugins/cache/redis/list.go#L166) `ListCache.Len`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (int64, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0330

- 定义：[plugins/cache/redis/list.go:171](../../../plugins/cache/redis/list.go#L171) `ListCache.Pop`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (*V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0331

- 定义：[plugins/cache/redis/list.go:187](../../../plugins/cache/redis/list.go#L187) `ListCache.PopLast`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (*V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0332

- 定义：[plugins/cache/redis/list.go:206](../../../plugins/cache/redis/list.go#L206) `ListCache.Remove`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, count int64, value V) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0333

- 定义：[plugins/cache/redis/list.go:215](../../../plugins/cache/redis/list.go#L215) `ListCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.ListCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.ListCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0334

- 定义：[plugins/cache/redis/set.go:35](../../../plugins/cache/redis/set.go#L35) `SetCache.encodeMembers`；分类S；阶段P5。
- 完整当前类型：`func(members []V) ([]any, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.SetCache[K, V any]`。

| 位置                                                                       | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/cache/redis/set.go:72](../../../plugins/cache/redis/set.go#L72)   | `Add`         | SDK生产  |
| [plugins/cache/redis/set.go:122](../../../plugins/cache/redis/set.go#L122) | `Remove`      | SDK生产  |

#### P0335

- 定义：[plugins/cache/redis/set.go:53](../../../plugins/cache/redis/set.go#L53) `SetCache.Add`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, members []V, opts ...github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetOption) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0336

- 定义：[plugins/cache/redis/set.go:87](../../../plugins/cache/redis/set.go#L87) `SetCache.Members`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) ([]V, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0337

- 定义：[plugins/cache/redis/set.go:104](../../../plugins/cache/redis/set.go#L104) `SetCache.IsMember`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, member V) (bool, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0338

- 定义：[plugins/cache/redis/set.go:113](../../../plugins/cache/redis/set.go#L113) `SetCache.Count`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) (int64, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0339

- 定义：[plugins/cache/redis/set.go:118](../../../plugins/cache/redis/set.go#L118) `SetCache.Remove`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K, members ...V) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0340

- 定义：[plugins/cache/redis/set.go:130](../../../plugins/cache/redis/set.go#L130) `SetCache.Delete`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, key K) error`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis.SetCache[K, V any]`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver.SetCache`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0341

- 定义：[plugins/cron/plugin.go:23](../../../plugins/cron/plugin.go#L23) `ErrNoActivator`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                       | 所在函数/声明                      | 引用来源 |
| -------------------------------------------------------------------------- | ---------------------------------- | -------- |
| [plugins/cron/plugin.go:207](../../../plugins/cron/plugin.go#L207)         | `doStart`                          | SDK生产  |
| [plugins/cron/plugin_test.go:74](../../../plugins/cron/plugin_test.go#L74) | `TestPlugin_NoActivatorFailsStart` | 单测     |

#### P0342

- 定义：[plugins/cron/plugin.go:25](../../../plugins/cron/plugin.go#L25) `ErrInvalidJobTarget`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/cron/plugin.go:223](../../../plugins/cron/plugin.go#L223) | `doStart`     | SDK生产  |

#### P0343

- 定义：[plugins/cron/plugin.go:26](../../../plugins/cron/plugin.go#L26) `ErrInvalidJobSchedule`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                       | 所在函数/声明                      | 引用来源 |
| -------------------------------------------------------------------------- | ---------------------------------- | -------- |
| [plugins/cron/plugin.go:235](../../../plugins/cron/plugin.go#L235)         | `doStart`                          | SDK生产  |
| [plugins/cron/plugin.go:243](../../../plugins/cron/plugin.go#L243)         | `doStart`                          | SDK生产  |
| [plugins/cron/plugin.go:248](../../../plugins/cron/plugin.go#L248)         | `doStart`                          | SDK生产  |
| [plugins/cron/plugin.go:256](../../../plugins/cron/plugin.go#L256)         | `doStart`                          | SDK生产  |
| [plugins/cron/plugin.go:260](../../../plugins/cron/plugin.go#L260)         | `doStart`                          | SDK生产  |
| [plugins/cron/plugin_test.go:87](../../../plugins/cron/plugin_test.go#L87) | `TestPlugin_InvalidSpecFailsStart` | 单测     |

#### P0344

- 定义：[plugins/cron/plugin.go:143](../../../plugins/cron/plugin.go#L143) `Plugin.startErr`；分类S；阶段P5。
- 完整当前类型：`error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/cron/plugin.go:200](../../../plugins/cron/plugin.go#L200) | `Start`       | SDK生产  |
| [plugins/cron/plugin.go:202](../../../plugins/cron/plugin.go#L202) | `Start`       | SDK生产  |

#### P0345

- 定义：[plugins/cron/plugin.go:146](../../../plugins/cron/plugin.go#L146) `Plugin.stopErr`；分类S；阶段P5。
- 完整当前类型：`error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/cron/plugin.go:298](../../../plugins/cron/plugin.go#L298) | `Stop`        | SDK生产  |
| [plugins/cron/plugin.go:302](../../../plugins/cron/plugin.go#L302) | `Stop`        | SDK生产  |

#### P0346

- 定义：[plugins/cron/plugin.go:198](../../../plugins/cron/plugin.go#L198) `Plugin.Start`；分类S；阶段P3/P5。
- 完整当前类型：`func(_ context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`；`github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`。

| 位置                                                                                       | 所在函数/声明                                | 引用来源 |
| ------------------------------------------------------------------------------------------ | -------------------------------------------- | -------- |
| [plugins/cron/stop_retention_test.go:92](../../../plugins/cron/stop_retention_test.go#L92) | `TestStopActiveTriggersRetentionAndDeadline` | 单测     |

#### P0347

- 定义：[plugins/cron/plugin.go:205](../../../plugins/cron/plugin.go#L205) `Plugin.doStart`；分类S；阶段P5。
- 完整当前类型：`func() error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/cron/plugin.go:200](../../../plugins/cron/plugin.go#L200) | `Start`       | SDK生产  |

#### P0348

- 定义：[plugins/cron/plugin.go:210](../../../plugins/cron/plugin.go#L210) `doStart.schedOpts`；分类N；阶段P5。
- 完整当前类型：`[]github.com/go-co-op/gocron/v2.SchedulerOption`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/cron/plugin.go:212](../../../plugins/cron/plugin.go#L212) | `doStart`     | SDK生产  |
| [plugins/cron/plugin.go:212](../../../plugins/cron/plugin.go#L212) | `doStart`     | SDK生产  |
| [plugins/cron/plugin.go:215](../../../plugins/cron/plugin.go#L215) | `doStart`     | SDK生产  |

#### P0349

- 定义：[plugins/cron/plugin.go:270](../../../plugins/cron/plugin.go#L270) `Plugin.Run`；分类S；阶段P3/P5。
- 完整当前类型：`func(lifetimeCtx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`。

| 位置                                                                                         | 所在函数/声明                                | 引用来源 |
| -------------------------------------------------------------------------------------------- | -------------------------------------------- | -------- |
| [plugins/cron/stop_retention_test.go:101](../../../plugins/cron/stop_retention_test.go#L101) | `TestStopActiveTriggersRetentionAndDeadline` | 单测     |

#### P0350

- 定义：[plugins/cron/plugin.go:286](../../../plugins/cron/plugin.go#L286) `Plugin.Quiesce`；分类S；阶段P3/P5。
- 完整当前类型：`func(_ context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`。

| 位置                                                                                         | 所在函数/声明                                | 引用来源 |
| -------------------------------------------------------------------------------------------- | -------------------------------------------- | -------- |
| [plugins/cron/stop_retention_test.go:135](../../../plugins/cron/stop_retention_test.go#L135) | `TestStopActiveTriggersRetentionAndDeadline` | 单测     |

#### P0351

- 定义：[plugins/cron/plugin.go:296](../../../plugins/cron/plugin.go#L296) `Plugin.Stop`；分类S；阶段P3/P5。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`；`github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`。

| 位置                                                                                         | 所在函数/声明                                | 引用来源 |
| -------------------------------------------------------------------------------------------- | -------------------------------------------- | -------- |
| [plugins/cron/stop_retention_test.go:119](../../../plugins/cron/stop_retention_test.go#L119) | `TestStopActiveTriggersRetentionAndDeadline` | 单测     |
| [plugins/cron/stop_retention_test.go:161](../../../plugins/cron/stop_retention_test.go#L161) | `TestStopActiveTriggersRetentionAndDeadline` | 单测     |

#### P0352

- 定义：[plugins/cron/plugin.go:305](../../../plugins/cron/plugin.go#L305) `Plugin.doStop`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context) error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/cron/plugin.go:298](../../../plugins/cron/plugin.go#L298) | `Stop`        | SDK生产  |

#### P0353

- 定义：[plugins/database/adapter/mysql/mysql.go:20](../../../plugins/database/adapter/mysql/mysql.go#L20) `adapter.err`；分类S；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                                 | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/database/adapter/mysql/mysql.go:36](../../../plugins/database/adapter/mysql/mysql.go#L36)   | `New`         | SDK生产  |
| [plugins/database/adapter/mysql/mysql.go:42](../../../plugins/database/adapter/mysql/mysql.go#L42)   | `New`         | SDK生产  |
| [plugins/database/adapter/mysql/mysql.go:64](../../../plugins/database/adapter/mysql/mysql.go#L64)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/mysql/mysql.go:100](../../../plugins/database/adapter/mysql/mysql.go#L100) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/mysql/mysql.go:104](../../../plugins/database/adapter/mysql/mysql.go#L104) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/mysql/mysql.go:118](../../../plugins/database/adapter/mysql/mysql.go#L118) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/mysql/mysql.go:134](../../../plugins/database/adapter/mysql/mysql.go#L134) | `Open`        | SDK生产  |
| [plugins/database/adapter/mysql/mysql.go:135](../../../plugins/database/adapter/mysql/mysql.go#L135) | `Open`        | SDK生产  |

#### P0354

- 定义：[plugins/database/adapter/mysql/mysql.go:133](../../../plugins/database/adapter/mysql/mysql.go#L133) `adapter.Open`；分类S；阶段P4。
- 完整当前类型：`func() (gorm.io/gorm.Dialector, error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0355

- 定义：[plugins/database/adapter/mysql/mysql.go:141](../../../plugins/database/adapter/mysql/mysql.go#L141) `validateDSN`；分类S；阶段P4。
- 完整当前类型：`func(raw string) error`。

| 位置                                                                                                 | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/database/adapter/mysql/mysql.go:40](../../../plugins/database/adapter/mysql/mysql.go#L40)   | `New`         | SDK生产  |
| [plugins/database/adapter/mysql/mysql.go:63](../../../plugins/database/adapter/mysql/mysql.go#L63)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/mysql/mysql.go:117](../../../plugins/database/adapter/mysql/mysql.go#L117) | `FromEnv`     | SDK生产  |

#### P0356

- 定义：[plugins/database/adapter/oracle/oracle.go:20](../../../plugins/database/adapter/oracle/oracle.go#L20) `adapter.err`；分类S；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                                     | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/database/adapter/oracle/oracle.go:36](../../../plugins/database/adapter/oracle/oracle.go#L36)   | `New`         | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:42](../../../plugins/database/adapter/oracle/oracle.go#L42)   | `New`         | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:70](../../../plugins/database/adapter/oracle/oracle.go#L70)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:101](../../../plugins/database/adapter/oracle/oracle.go#L101) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:104](../../../plugins/database/adapter/oracle/oracle.go#L104) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:107](../../../plugins/database/adapter/oracle/oracle.go#L107) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:118](../../../plugins/database/adapter/oracle/oracle.go#L118) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:134](../../../plugins/database/adapter/oracle/oracle.go#L134) | `Open`        | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:135](../../../plugins/database/adapter/oracle/oracle.go#L135) | `Open`        | SDK生产  |

#### P0357

- 定义：[plugins/database/adapter/oracle/oracle.go:133](../../../plugins/database/adapter/oracle/oracle.go#L133) `adapter.Open`；分类S；阶段P4。
- 完整当前类型：`func() (gorm.io/gorm.Dialector, error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0358

- 定义：[plugins/database/adapter/oracle/oracle.go:141](../../../plugins/database/adapter/oracle/oracle.go#L141) `validateDSN`；分类S；阶段P4。
- 完整当前类型：`func(raw string) error`。

| 位置                                                                                                     | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/database/adapter/oracle/oracle.go:40](../../../plugins/database/adapter/oracle/oracle.go#L40)   | `New`         | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:69](../../../plugins/database/adapter/oracle/oracle.go#L69)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/oracle/oracle.go:117](../../../plugins/database/adapter/oracle/oracle.go#L117) | `FromEnv`     | SDK生产  |

#### P0359

- 定义：[plugins/database/adapter/postgres/postgres.go:20](../../../plugins/database/adapter/postgres/postgres.go#L20) `adapter.err`；分类S；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                                             | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/database/adapter/postgres/postgres.go:36](../../../plugins/database/adapter/postgres/postgres.go#L36)   | `New`         | SDK生产  |
| [plugins/database/adapter/postgres/postgres.go:42](../../../plugins/database/adapter/postgres/postgres.go#L42)   | `New`         | SDK生产  |
| [plugins/database/adapter/postgres/postgres.go:63](../../../plugins/database/adapter/postgres/postgres.go#L63)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/postgres/postgres.go:99](../../../plugins/database/adapter/postgres/postgres.go#L99)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/postgres/postgres.go:113](../../../plugins/database/adapter/postgres/postgres.go#L113) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/postgres/postgres.go:128](../../../plugins/database/adapter/postgres/postgres.go#L128) | `Open`        | SDK生产  |
| [plugins/database/adapter/postgres/postgres.go:129](../../../plugins/database/adapter/postgres/postgres.go#L129) | `Open`        | SDK生产  |

#### P0360

- 定义：[plugins/database/adapter/postgres/postgres.go:127](../../../plugins/database/adapter/postgres/postgres.go#L127) `adapter.Open`；分类S；阶段P4。
- 完整当前类型：`func() (gorm.io/gorm.Dialector, error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0361

- 定义：[plugins/database/adapter/postgres/postgres.go:148](../../../plugins/database/adapter/postgres/postgres.go#L148) `validateDSN`；分类S；阶段P4。
- 完整当前类型：`func(raw string) error`。

| 位置                                                                                                             | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/database/adapter/postgres/postgres.go:40](../../../plugins/database/adapter/postgres/postgres.go#L40)   | `New`         | SDK生产  |
| [plugins/database/adapter/postgres/postgres.go:62](../../../plugins/database/adapter/postgres/postgres.go#L62)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/postgres/postgres.go:112](../../../plugins/database/adapter/postgres/postgres.go#L112) | `FromEnv`     | SDK生产  |

#### P0362

- 定义：[plugins/database/adapter/sqlite/sqlite.go:22](../../../plugins/database/adapter/sqlite/sqlite.go#L22) `adapter.err`；分类S；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                                   | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/database/adapter/sqlite/sqlite.go:33](../../../plugins/database/adapter/sqlite/sqlite.go#L33) | `New`         | SDK生产  |
| [plugins/database/adapter/sqlite/sqlite.go:61](../../../plugins/database/adapter/sqlite/sqlite.go#L61) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/sqlite/sqlite.go:80](../../../plugins/database/adapter/sqlite/sqlite.go#L80) | `Open`        | SDK生产  |
| [plugins/database/adapter/sqlite/sqlite.go:81](../../../plugins/database/adapter/sqlite/sqlite.go#L81) | `Open`        | SDK生产  |

#### P0363

- 定义：[plugins/database/adapter/sqlite/sqlite.go:79](../../../plugins/database/adapter/sqlite/sqlite.go#L79) `adapter.Open`；分类S；阶段P4。
- 完整当前类型：`func() (gorm.io/gorm.Dialector, error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0364

- 定义：[plugins/database/adapter/sqlserver/sqlserver.go:22](../../../plugins/database/adapter/sqlserver/sqlserver.go#L22) `adapter.err`；分类S；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/database/adapter/sqlserver/sqlserver.go:38](../../../plugins/database/adapter/sqlserver/sqlserver.go#L38)   | `New`         | SDK生产  |
| [plugins/database/adapter/sqlserver/sqlserver.go:44](../../../plugins/database/adapter/sqlserver/sqlserver.go#L44)   | `New`         | SDK生产  |
| [plugins/database/adapter/sqlserver/sqlserver.go:64](../../../plugins/database/adapter/sqlserver/sqlserver.go#L64)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/sqlserver/sqlserver.go:96](../../../plugins/database/adapter/sqlserver/sqlserver.go#L96)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/sqlserver/sqlserver.go:100](../../../plugins/database/adapter/sqlserver/sqlserver.go#L100) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/sqlserver/sqlserver.go:114](../../../plugins/database/adapter/sqlserver/sqlserver.go#L114) | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/sqlserver/sqlserver.go:129](../../../plugins/database/adapter/sqlserver/sqlserver.go#L129) | `Open`        | SDK生产  |
| [plugins/database/adapter/sqlserver/sqlserver.go:130](../../../plugins/database/adapter/sqlserver/sqlserver.go#L130) | `Open`        | SDK生产  |

#### P0365

- 定义：[plugins/database/adapter/sqlserver/sqlserver.go:128](../../../plugins/database/adapter/sqlserver/sqlserver.go#L128) `adapter.Open`；分类S；阶段P4。
- 完整当前类型：`func() (gorm.io/gorm.Dialector, error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0366

- 定义：[plugins/database/adapter/sqlserver/sqlserver.go:136](../../../plugins/database/adapter/sqlserver/sqlserver.go#L136) `validateDSN`；分类S；阶段P4。
- 完整当前类型：`func(raw string) error`。

| 位置                                                                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/database/adapter/sqlserver/sqlserver.go:42](../../../plugins/database/adapter/sqlserver/sqlserver.go#L42)   | `New`         | SDK生产  |
| [plugins/database/adapter/sqlserver/sqlserver.go:63](../../../plugins/database/adapter/sqlserver/sqlserver.go#L63)   | `FromEnv`     | SDK生产  |
| [plugins/database/adapter/sqlserver/sqlserver.go:113](../../../plugins/database/adapter/sqlserver/sqlserver.go#L113) | `FromEnv`     | SDK生产  |

#### P0367

- 定义：[plugins/database/context.go:17](../../../plugins/database/context.go#L17) `Get`；分类A；阶段P4。
- 完整当前类型：`func(ctx context.Context) (*gorm.io/gorm.DB, *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error)`。

| 位置                                                                                                                             | 所在函数/声明                      | 引用来源 |
| -------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- | -------- |
| [plugins/database/adapter/mysql/integration_test.go:124](../../../plugins/database/adapter/mysql/integration_test.go#L124)       | `TestIntegration_MySQL`            | 单测     |
| [plugins/database/adapter/postgres/integration_test.go:102](../../../plugins/database/adapter/postgres/integration_test.go#L102) | `TestIntegration_PostgresReal`     | 单测     |
| [plugins/database/context.go:35](../../../plugins/database/context.go#L35)                                                       | `MustGet`                          | SDK生产  |
| [plugins/database/error_code_test.go:44](../../../plugins/database/error_code_test.go#L44)                                       | `TestGetPreservesApplicationError` | 单测     |
| [plugins/database/plugin_test.go:279](../../../plugins/database/plugin_test.go#L279)                                             | `TestGetAndMustGet`                | 单测     |

#### P0368

- 定义：[plugins/database/dialect/dialect.go:27](../../../plugins/database/dialect/dialect.go#L27) `ErrDriverNotLoaded`；分类N；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                                 | 所在函数/声明                   | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| [plugins/database/dialect/dialect.go:108](../../../plugins/database/dialect/dialect.go#L108)         | `lookup`                        | SDK生产  |
| [plugins/database/dialect/dialect_test.go:42](../../../plugins/database/dialect/dialect_test.go#L42) | `TestRegistry_NormalAndUnknown` | 单测     |
| [plugins/database/dialect/example_test.go:29](../../../plugins/database/dialect/example_test.go#L29) | `ExampleLookup`                 | 单测     |

#### P0369

- 定义：[plugins/database/dialect/dialect.go:28](../../../plugins/database/dialect/dialect.go#L28) `ErrNilAdapter`；分类N；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                                 | 所在函数/声明                 | 引用来源 |
| ---------------------------------------------------------------------------------------------------- | ----------------------------- | -------- |
| [plugins/database/dialect/dialect.go:113](../../../plugins/database/dialect/dialect.go#L113)         | `lookup`                      | SDK生产  |
| [plugins/database/dialect/dialect_test.go:55](../../../plugins/database/dialect/dialect_test.go#L55) | `TestRegistry_TypedNilAndNil` | 单测     |
| [plugins/database/dialect/dialect_test.go:65](../../../plugins/database/dialect/dialect_test.go#L65) | `TestRegistry_TypedNilAndNil` | 单测     |

#### P0370

- 定义：[plugins/database/dialect/dialect.go:39](../../../plugins/database/dialect/dialect.go#L39) `Adapter.Open`；分类S；阶段P4。
- 完整当前类型：`func() (gorm.io/gorm.Dialector, error)`。

| 位置                                                                                                                           | 所在函数/声明                                   | 引用来源 |
| ------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------- | -------- |
| [plugins/database/adapter/mysql/mysql_test.go:37](../../../plugins/database/adapter/mysql/mysql_test.go#L37)                   | `TestAdapter_Open_Type`                         | 单测     |
| [plugins/database/adapter/mysql/mysql_test.go:64](../../../plugins/database/adapter/mysql/mysql_test.go#L64)                   | `TestAdapter_FromEnv_Assembly`                  | 单测     |
| [plugins/database/adapter/mysql/mysql_test.go:90](../../../plugins/database/adapter/mysql/mysql_test.go#L90)                   | `TestAdapter_FromEnv_Defaults`                  | 单测     |
| [plugins/database/adapter/mysql/mysql_test.go:111](../../../plugins/database/adapter/mysql/mysql_test.go#L111)                 | `TestAdapter_FromEnv_DirectDSN`                 | 单测     |
| [plugins/database/adapter/mysql/mysql_test.go:176](../../../plugins/database/adapter/mysql/mysql_test.go#L176)                 | `TestAdapter_DSNValidationAndNoPasswordLeakage` | 单测     |
| [plugins/database/adapter/mysql/mysql_test.go:212](../../../plugins/database/adapter/mysql/mysql_test.go#L212)                 | `TestAdapter_InitRegistration`                  | 单测     |
| [plugins/database/adapter/mysql/mysql_test.go:234](../../../plugins/database/adapter/mysql/mysql_test.go#L234)                 | `TestNoNetworkOrConnectionOnConstruction`       | 单测     |
| [plugins/database/adapter/mysql/mysql_test.go:256](../../../plugins/database/adapter/mysql/mysql_test.go#L256)                 | `TestRealEngine_MySQL`                          | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:37](../../../plugins/database/adapter/oracle/oracle_test.go#L37)               | `TestAdapter_Open_Type`                         | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:64](../../../plugins/database/adapter/oracle/oracle_test.go#L64)               | `TestAdapter_FromEnv_Assembly`                  | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:90](../../../plugins/database/adapter/oracle/oracle_test.go#L90)               | `TestAdapter_FromEnv_Defaults`                  | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:114](../../../plugins/database/adapter/oracle/oracle_test.go#L114)             | `TestAdapter_FromEnv_DirectDSN`                 | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:140](../../../plugins/database/adapter/oracle/oracle_test.go#L140)             | `TestAdapter_FromEnv_Escaping`                  | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:206](../../../plugins/database/adapter/oracle/oracle_test.go#L206)             | `TestAdapter_DSNValidationAndNoPasswordLeakage` | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:242](../../../plugins/database/adapter/oracle/oracle_test.go#L242)             | `TestAdapter_InitRegistration`                  | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:264](../../../plugins/database/adapter/oracle/oracle_test.go#L264)             | `TestNoNetworkOrConnectionOnConstruction`       | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:286](../../../plugins/database/adapter/oracle/oracle_test.go#L286)             | `TestRealEngine_Oracle`                         | 单测     |
| [plugins/database/adapter/postgres/postgres_test.go:37](../../../plugins/database/adapter/postgres/postgres_test.go#L37)       | `TestAdapter_Open_Type`                         | 单测     |
| [plugins/database/adapter/postgres/postgres_test.go:64](../../../plugins/database/adapter/postgres/postgres_test.go#L64)       | `TestAdapter_FromEnv_Assembly`                  | 单测     |
| [plugins/database/adapter/postgres/postgres_test.go:90](../../../plugins/database/adapter/postgres/postgres_test.go#L90)       | `TestAdapter_FromEnv_Defaults`                  | 单测     |
| [plugins/database/adapter/postgres/postgres_test.go:116](../../../plugins/database/adapter/postgres/postgres_test.go#L116)     | `TestAdapter_FromEnv_Escaping`                  | 单测     |
| [plugins/database/adapter/postgres/postgres_test.go:137](../../../plugins/database/adapter/postgres/postgres_test.go#L137)     | `TestAdapter_FromEnv_DirectDSN`                 | 单测     |
| [plugins/database/adapter/postgres/postgres_test.go:185](../../../plugins/database/adapter/postgres/postgres_test.go#L185)     | `TestAdapter_DSNValidationAndNoPasswordLeakage` | 单测     |
| [plugins/database/adapter/postgres/postgres_test.go:221](../../../plugins/database/adapter/postgres/postgres_test.go#L221)     | `TestAdapter_InitRegistration`                  | 单测     |
| [plugins/database/adapter/postgres/postgres_test.go:245](../../../plugins/database/adapter/postgres/postgres_test.go#L245)     | `TestRealEngine_Postgres`                       | 单测     |
| [plugins/database/adapter/sqlite/sqlite_test.go:28](../../../plugins/database/adapter/sqlite/sqlite_test.go#L28)               | `TestAdapter_InterfaceAndRegistration`          | 单测     |
| [plugins/database/adapter/sqlite/sqlite_test.go:76](../../../plugins/database/adapter/sqlite/sqlite_test.go#L76)               | `TestNew_ExplicitDSNAndErrors`                  | 单测     |
| [plugins/database/adapter/sqlite/sqlite_test.go:103](../../../plugins/database/adapter/sqlite/sqlite_test.go#L103)             | `TestFromEnv_FieldPrecedenceAndDefaults`        | 单测     |
| [plugins/database/adapter/sqlite/sqlite_test.go:115](../../../plugins/database/adapter/sqlite/sqlite_test.go#L115)             | `TestFromEnv_FieldPrecedenceAndDefaults`        | 单测     |
| [plugins/database/adapter/sqlite/sqlite_test.go:127](../../../plugins/database/adapter/sqlite/sqlite_test.go#L127)             | `TestFromEnv_FieldPrecedenceAndDefaults`        | 单测     |
| [plugins/database/adapter/sqlite/sqlite_test.go:139](../../../plugins/database/adapter/sqlite/sqlite_test.go#L139)             | `TestFromEnv_FieldPrecedenceAndDefaults`        | 单测     |
| [plugins/database/adapter/sqlite/sqlite_test.go:155](../../../plugins/database/adapter/sqlite/sqlite_test.go#L155)             | `TestNoNetworkOrConnectionOnConstruction`       | 单测     |
| [plugins/database/adapter/sqlite/sqlite_test.go:175](../../../plugins/database/adapter/sqlite/sqlite_test.go#L175)             | `TestSQLite_RealReadWrite_TempFile`             | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:62](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L62)   | `TestAdapter_Open_Type`                         | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:89](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L89)   | `TestAdapter_FromEnv_Assembly`                  | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:118](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L118) | `TestAdapter_FromEnv_SpecialCredentials`        | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:158](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L158) | `TestAdapter_FromEnv_IPv6`                      | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:192](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L192) | `TestAdapter_FromEnv_Defaults`                  | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:214](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L214) | `TestAdapter_FromEnv_DirectDSN`                 | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:263](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L263) | `TestAdapter_DSNValidationAndNoPasswordLeakage` | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:299](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L299) | `TestAdapter_InitRegistration`                  | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:323](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L323) | `TestRealEngine_SQLServer`                      | 单测     |
| [plugins/database/plugin.go:86](../../../plugins/database/plugin.go#L86)                                                       | `Start`                                         | SDK生产  |

#### P0371

- 定义：[plugins/database/dialect/dialect.go:72](../../../plugins/database/dialect/dialect.go#L72) `validateDriver`；分类S；阶段P4。
- 完整当前类型：`func(driver github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Driver) error`。

| 位置                                                                                       | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/database/dialect/dialect.go:85](../../../plugins/database/dialect/dialect.go#L85) | `register`    | SDK生产  |

#### P0372

- 定义：[plugins/database/dialect/dialect.go:102](../../../plugins/database/dialect/dialect.go#L102) `registry.lookup`；分类S；阶段P4。
- 完整当前类型：`func(driver github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Driver) (github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter, error)`。

| 位置                                                                                                   | 所在函数/声明                                   | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | ----------------------------------------------- | -------- |
| [plugins/database/dialect/dialect.go:135](../../../plugins/database/dialect/dialect.go#L135)           | `Lookup`                                        | SDK生产  |
| [plugins/database/dialect/dialect_test.go:33](../../../plugins/database/dialect/dialect_test.go#L33)   | `TestRegistry_NormalAndUnknown`                 | 单测     |
| [plugins/database/dialect/dialect_test.go:41](../../../plugins/database/dialect/dialect_test.go#L41)   | `TestRegistry_NormalAndUnknown`                 | 单测     |
| [plugins/database/dialect/dialect_test.go:54](../../../plugins/database/dialect/dialect_test.go#L54)   | `TestRegistry_TypedNilAndNil`                   | 单测     |
| [plugins/database/dialect/dialect_test.go:64](../../../plugins/database/dialect/dialect_test.go#L64)   | `TestRegistry_TypedNilAndNil`                   | 单测     |
| [plugins/database/dialect/dialect_test.go:146](../../../plugins/database/dialect/dialect_test.go#L146) | `TestRegistry_Concurrency`                      | 单测     |
| [plugins/database/dialect/dialect_test.go:179](../../../plugins/database/dialect/dialect_test.go#L179) | `TestRegistry_DoesNotHoldLockDuringFactoryCall` | 单测     |
| [plugins/database/dialect/dialect_test.go:189](../../../plugins/database/dialect/dialect_test.go#L189) | `TestRegistry_DoesNotHoldLockDuringFactoryCall` | 单测     |

#### P0373

- 定义：[plugins/database/dialect/dialect.go:134](../../../plugins/database/dialect/dialect.go#L134) `Lookup`；分类S；阶段P4。
- 完整当前类型：`func(driver github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Driver) (github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect.Adapter, error)`。

| 位置                                                                                                                           | 所在函数/声明                          | 引用来源 |
| ------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------- | -------- |
| [plugins/database/adapter/mysql/mysql_test.go:201](../../../plugins/database/adapter/mysql/mysql_test.go#L201)                 | `TestAdapter_InitRegistration`         | 单测     |
| [plugins/database/adapter/oracle/oracle_test.go:231](../../../plugins/database/adapter/oracle/oracle_test.go#L231)             | `TestAdapter_InitRegistration`         | 单测     |
| [plugins/database/adapter/postgres/postgres_test.go:210](../../../plugins/database/adapter/postgres/postgres_test.go#L210)     | `TestAdapter_InitRegistration`         | 单测     |
| [plugins/database/adapter/sqlite/sqlite_test.go:17](../../../plugins/database/adapter/sqlite/sqlite_test.go#L17)               | `TestAdapter_InterfaceAndRegistration` | 单测     |
| [plugins/database/adapter/sqlserver/sqlserver_test.go:288](../../../plugins/database/adapter/sqlserver/sqlserver_test.go#L288) | `TestAdapter_InitRegistration`         | 单测     |
| [plugins/database/dialect/dialect_test.go:215](../../../plugins/database/dialect/dialect_test.go#L215)                         | `TestPackageLevelRegisterAndLookup`    | 单测     |
| [plugins/database/dialect/example_test.go:28](../../../plugins/database/dialect/example_test.go#L28)                           | `ExampleLookup`                        | 单测     |
| [plugins/database/plugin.go:79](../../../plugins/database/plugin.go#L79)                                                       | `Start`                                | SDK生产  |

#### P0374

- 定义：[plugins/database/gormlog/gormlog.go:126](../../../plugins/database/gormlog/gormlog.go#L126) `Logger.Trace`；分类F；阶段P4。
- 完整当前类型：`func(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)`。
- 方法集：`gorm.io/gorm/logger.Interface`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0375

- 定义：[plugins/database/models/base_entity.go:17](../../../plugins/database/models/base_entity.go#L17) `ErrNilStatement`；分类N；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                               | 所在函数/声明                                        | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ---------------------------------------------------- | -------- |
| [plugins/database/models/base_entity.go:67](../../../plugins/database/models/base_entity.go#L67)   | `generateEntityID`                                   | SDK生产  |
| [plugins/database/models/models_test.go:240](../../../plugins/database/models/models_test.go#L240) | `Test_BaseEntity_BeforeCreate_NilStatementOrContext` | 单测     |
| [plugins/database/models/models_test.go:243](../../../plugins/database/models/models_test.go#L243) | `Test_BaseEntity_BeforeCreate_NilStatementOrContext` | 单测     |

#### P0376

- 定义：[plugins/database/models/base_entity.go:18](../../../plugins/database/models/base_entity.go#L18) `ErrNilContext`；分类N；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                               | 所在函数/声明                                        | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ---------------------------------------------------- | -------- |
| [plugins/database/models/base_entity.go:71](../../../plugins/database/models/base_entity.go#L71)   | `generateEntityID`                                   | SDK生产  |
| [plugins/database/models/models_test.go:246](../../../plugins/database/models/models_test.go#L246) | `Test_BaseEntity_BeforeCreate_NilStatementOrContext` | 单测     |

#### P0377

- 定义：[plugins/database/models/base_entity.go:19](../../../plugins/database/models/base_entity.go#L19) `ErrSnowflakeNodeNotFound`；分类N；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                               | 所在函数/声明                              | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ------------------------------------------ | -------- |
| [plugins/database/models/base_entity.go:76](../../../plugins/database/models/base_entity.go#L76)   | `generateEntityID`                         | SDK生产  |
| [plugins/database/models/models_test.go:230](../../../plugins/database/models/models_test.go#L230) | `Test_BaseEntity_BeforeCreate_MissingNode` | 单测     |

#### P0378

- 定义：[plugins/database/models/base_entity.go:20](../../../plugins/database/models/base_entity.go#L20) `ErrNilSnowflakeNode`；分类N；阶段P4。
- 完整当前类型：`error`。

| 位置                                                                                             | 所在函数/声明      | 引用来源 |
| ------------------------------------------------------------------------------------------------ | ------------------ | -------- |
| [plugins/database/models/base_entity.go:79](../../../plugins/database/models/base_entity.go#L79) | `generateEntityID` | SDK生产  |

#### P0379

- 定义：[plugins/database/models/base_entity.go:45](../../../plugins/database/models/base_entity.go#L45) `BaseEntity.BeforeCreate`；分类F；阶段P4。
- 完整当前类型：`func(tx *gorm.io/gorm.DB) error`。
- 方法集：`gorm.io/gorm/callbacks.BeforeCreateInterface`。

| 位置                                                                                               | 所在函数/声明                                        | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ---------------------------------------------------- | -------- |
| [plugins/database/models/models_test.go:160](../../../plugins/database/models/models_test.go#L160) | `Test_BaseEntity_BeforeCreate_Success`               | 单测     |
| [plugins/database/models/models_test.go:189](../../../plugins/database/models/models_test.go#L189) | `Test_BaseEntity_BeforeCreate_GeneProvider`          | 单测     |
| [plugins/database/models/models_test.go:210](../../../plugins/database/models/models_test.go#L210) | `Test_BaseEntity_BeforeCreate_PreserveExistingID`    | 单测     |
| [plugins/database/models/models_test.go:226](../../../plugins/database/models/models_test.go#L226) | `Test_BaseEntity_BeforeCreate_MissingNode`           | 单测     |
| [plugins/database/models/models_test.go:240](../../../plugins/database/models/models_test.go#L240) | `Test_BaseEntity_BeforeCreate_NilStatementOrContext` | 单测     |
| [plugins/database/models/models_test.go:243](../../../plugins/database/models/models_test.go#L243) | `Test_BaseEntity_BeforeCreate_NilStatementOrContext` | 单测     |
| [plugins/database/models/models_test.go:246](../../../plugins/database/models/models_test.go#L246) | `Test_BaseEntity_BeforeCreate_NilStatementOrContext` | 单测     |
| [plugins/database/models/models_test.go:265](../../../plugins/database/models/models_test.go#L265) | `Test_BaseEntity_BeforeCreate_InvalidGene`           | 单测     |

#### P0380

- 定义：[plugins/database/models/base_entity.go:60](../../../plugins/database/models/base_entity.go#L60) `BaseEntity.BeforeUpdate`；分类F；阶段P4。
- 完整当前类型：`func(tx *gorm.io/gorm.DB) error`。
- 方法集：`gorm.io/gorm/callbacks.BeforeUpdateInterface`。

| 位置                                                                                               | 所在函数/声明                  | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ------------------------------ | -------- |
| [plugins/database/models/models_test.go:280](../../../plugins/database/models/models_test.go#L280) | `Test_BaseEntity_BeforeUpdate` | 单测     |

#### P0381

- 定义：[plugins/database/models/base_entity.go:65](../../../plugins/database/models/base_entity.go#L65) `generateEntityID`；分类S；阶段P4。
- 完整当前类型：`func(tx *gorm.io/gorm.DB) (github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake.SnowflakeID, error)`。

| 位置                                                                                                                     | 所在函数/声明  | 引用来源 |
| ------------------------------------------------------------------------------------------------------------------------ | -------------- | -------- |
| [plugins/database/models/base_entity.go:47](../../../plugins/database/models/base_entity.go#L47)                         | `BeforeCreate` | SDK生产  |
| [plugins/database/models/base_entity_soft_delete.go:36](../../../plugins/database/models/base_entity_soft_delete.go#L36) | `BeforeCreate` | SDK生产  |

#### P0382

- 定义：[plugins/database/models/base_entity_soft_delete.go:34](../../../plugins/database/models/base_entity_soft_delete.go#L34) `BaseEntityWithSoftDelete.BeforeCreate`；分类F；阶段P4。
- 完整当前类型：`func(tx *gorm.io/gorm.DB) error`。
- 方法集：`gorm.io/gorm/callbacks.BeforeCreateInterface`。

| 位置                                                                                               | 所在函数/声明                                          | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ------------------------------------------------------ | -------- |
| [plugins/database/models/models_test.go:302](../../../plugins/database/models/models_test.go#L302) | `Test_BaseEntityWithSoftDelete_BeforeCreate_AndUpdate` | 单测     |

#### P0383

- 定义：[plugins/database/models/base_entity_soft_delete.go:49](../../../plugins/database/models/base_entity_soft_delete.go#L49) `BaseEntityWithSoftDelete.BeforeUpdate`；分类F；阶段P4。
- 完整当前类型：`func(tx *gorm.io/gorm.DB) error`。
- 方法集：`gorm.io/gorm/callbacks.BeforeUpdateInterface`。

| 位置                                                                                               | 所在函数/声明                                          | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ------------------------------------------------------ | -------- |
| [plugins/database/models/models_test.go:317](../../../plugins/database/models/models_test.go#L317) | `Test_BaseEntityWithSoftDelete_BeforeCreate_AndUpdate` | 单测     |

#### P0384

- 定义：[plugins/database/plugin.go:65](../../../plugins/database/plugin.go#L65) `Plugin.Start`；分类S；阶段P3/P4。
- 完整当前类型：`func(ctx context.Context) (retErr error)`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0385

- 定义：[plugins/database/plugin.go:147](../../../plugins/database/plugin.go#L147) `Plugin.Stop`；分类S；阶段P3/P4。
- 完整当前类型：`func(_ context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0386

- 定义：[plugins/database/plugin.go:153](../../../plugins/database/plugin.go#L153) `Plugin.closeSQLDB`；分类S；阶段P4。
- 完整当前类型：`func() error`。

| 位置                                                                       | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/database/plugin.go:120](../../../plugins/database/plugin.go#L120) | `Start`       | SDK生产  |
| [plugins/database/plugin.go:150](../../../plugins/database/plugin.go#L150) | `Stop`        | SDK生产  |

#### P0387

- 定义：[plugins/email/client.go:24](../../../plugins/email/client.go#L24) `ErrStopped`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                           | 所在函数/声明                           | 引用来源 |
| ------------------------------------------------------------------------------ | --------------------------------------- | -------- |
| [plugins/email/client.go:344](../../../plugins/email/client.go#L344)           | `acquire`                               | SDK生产  |
| [plugins/email/client_test.go:625](../../../plugins/email/client_test.go#L625) | `TestClient_ConcurrencyAndStopDraining` | 单测     |

#### P0388

- 定义：[plugins/email/client.go:25](../../../plugins/email/client.go#L25) `ErrRecipientEmpty`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/email/client.go:442](../../../plugins/email/client.go#L442) | `Send`        | SDK生产  |

#### P0389

- 定义：[plugins/email/client.go:26](../../../plugins/email/client.go#L26) `ErrSubjectEmpty`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/email/client.go:445](../../../plugins/email/client.go#L445) | `Send`        | SDK生产  |

#### P0390

- 定义：[plugins/email/client.go:27](../../../plugins/email/client.go#L27) `ErrTemplateEmpty`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明  | 引用来源 |
| -------------------------------------------------------------------- | -------------- | -------- |
| [plugins/email/client.go:621](../../../plugins/email/client.go#L621) | `SendTemplate` | SDK生产  |

#### P0391

- 定义：[plugins/email/client.go:28](../../../plugins/email/client.go#L28) `ErrNotInitialized`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明    | 引用来源 |
| -------------------------------------------------------------------- | ---------------- | -------- |
| [plugins/email/client.go:176](../../../plugins/email/client.go#L176) | `Start`          | SDK生产  |
| [plugins/email/client.go:436](../../../plugins/email/client.go#L436) | `Send`           | SDK生产  |
| [plugins/email/client.go:615](../../../plugins/email/client.go#L615) | `SendTemplate`   | SDK生产  |
| [plugins/email/client.go:645](../../../plugins/email/client.go#L645) | `AddTemplateDir` | SDK生产  |
| [plugins/email/client.go:660](../../../plugins/email/client.go#L660) | `AddTemplateFS`  | SDK生产  |

#### P0392

- 定义：[plugins/email/client.go:29](../../../plugins/email/client.go#L29) `ErrMissingHost`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                           | 所在函数/声明                | 引用来源 |
| ------------------------------------------------------------------------------ | ---------------------------- | -------- |
| [plugins/email/client.go:233](../../../plugins/email/client.go#L233)           | `Start`                      | SDK生产  |
| [plugins/email/client_test.go:357](../../../plugins/email/client_test.go#L357) | `TestPlugin_StartValidation` | 单测     |

#### P0393

- 定义：[plugins/email/client.go:30](../../../plugins/email/client.go#L30) `ErrMissingUser`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                           | 所在函数/声明                | 引用来源 |
| ------------------------------------------------------------------------------ | ---------------------------- | -------- |
| [plugins/email/client.go:236](../../../plugins/email/client.go#L236)           | `Start`                      | SDK生产  |
| [plugins/email/client_test.go:362](../../../plugins/email/client_test.go#L362) | `TestPlugin_StartValidation` | 单测     |

#### P0394

- 定义：[plugins/email/client.go:31](../../../plugins/email/client.go#L31) `ErrMissingPass`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                           | 所在函数/声明                | 引用来源 |
| ------------------------------------------------------------------------------ | ---------------------------- | -------- |
| [plugins/email/client.go:239](../../../plugins/email/client.go#L239)           | `Start`                      | SDK生产  |
| [plugins/email/client_test.go:367](../../../plugins/email/client_test.go#L367) | `TestPlugin_StartValidation` | 单测     |

#### P0395

- 定义：[plugins/email/client.go:32](../../../plugins/email/client.go#L32) `ErrMissingFrom`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                           | 所在函数/声明                | 引用来源 |
| ------------------------------------------------------------------------------ | ---------------------------- | -------- |
| [plugins/email/client.go:242](../../../plugins/email/client.go#L242)           | `Start`                      | SDK生产  |
| [plugins/email/client_test.go:372](../../../plugins/email/client_test.go#L372) | `TestPlugin_StartValidation` | 单测     |

#### P0396

- 定义：[plugins/email/client.go:33](../../../plugins/email/client.go#L33) `ErrInvalidPort`；分类N；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/email/client.go:192](../../../plugins/email/client.go#L192) | `Start`       | SDK生产  |

#### P0397

- 定义：[plugins/email/client.go:174](../../../plugins/email/client.go#L174) `Plugin.Start`；分类S；阶段P3/P5。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。

| 位置                                                                           | 所在函数/声明                                | 引用来源 |
| ------------------------------------------------------------------------------ | -------------------------------------------- | -------- |
| [plugins/email/client_test.go:405](../../../plugins/email/client_test.go#L405) | `TestPlugin_FromEnv`                         | 单测     |
| [plugins/email/client_test.go:534](../../../plugins/email/client_test.go#L534) | `TestClient_ContextCancelAndTimeout`         | 单测     |
| [plugins/email/client_test.go:575](../../../plugins/email/client_test.go#L575) | `TestClient_ConcurrencyAndStopDraining`      | 单测     |
| [plugins/email/client_test.go:652](../../../plugins/email/client_test.go#L652) | `TestClient_WithLogger`                      | 单测     |
| [plugins/email/client_test.go:705](../../../plugins/email/client_test.go#L705) | `TestClient_ContextCancelDuringSend`         | 单测     |
| [plugins/email/client_test.go:746](../../../plugins/email/client_test.go#L746) | `TestClient_StopDrainingWaitingInflight`     | 单测     |
| [plugins/email/client_test.go:822](../../../plugins/email/client_test.go#L822) | `TestClient_ContextCancelRaceAndDoubleClose` | 单测     |

#### P0398

- 定义：[plugins/email/client.go:302](../../../plugins/email/client.go#L302) `Plugin.Stop`；分类S；阶段P3/P5。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`。

| 位置                                                                           | 所在函数/声明                                | 引用来源 |
| ------------------------------------------------------------------------------ | -------------------------------------------- | -------- |
| [plugins/email/client_test.go:409](../../../plugins/email/client_test.go#L409) | `TestPlugin_FromEnv`                         | 单测     |
| [plugins/email/client_test.go:423](../../../plugins/email/client_test.go#L423) | `TestPlugin_StopIdempotentAndNil`            | 单测     |
| [plugins/email/client_test.go:537](../../../plugins/email/client_test.go#L537) | `TestClient_ContextCancelAndTimeout`         | 单测     |
| [plugins/email/client_test.go:609](../../../plugins/email/client_test.go#L609) | `TestClient_ConcurrencyAndStopDraining`      | 单测     |
| [plugins/email/client_test.go:655](../../../plugins/email/client_test.go#L655) | `TestClient_WithLogger`                      | 单测     |
| [plugins/email/client_test.go:708](../../../plugins/email/client_test.go#L708) | `TestClient_ContextCancelDuringSend`         | 单测     |
| [plugins/email/client_test.go:749](../../../plugins/email/client_test.go#L749) | `TestClient_StopDrainingWaitingInflight`     | 单测     |
| [plugins/email/client_test.go:779](../../../plugins/email/client_test.go#L779) | `TestClient_StopDrainingWaitingInflight`     | 单测     |
| [plugins/email/client_test.go:825](../../../plugins/email/client_test.go#L825) | `TestClient_ContextCancelRaceAndDoubleClose` | 单测     |

#### P0399

- 定义：[plugins/email/client.go:339](../../../plugins/email/client.go#L339) `Plugin.acquire`；分类S；阶段P5。
- 完整当前类型：`func() error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/email/client.go:457](../../../plugins/email/client.go#L457) | `Send`        | SDK生产  |

#### P0400

- 定义：[plugins/email/client.go:373](../../../plugins/email/client.go#L373) `Plugin.buildMailClient`；分类S；阶段P5。
- 完整当前类型：`func(extraOpts ...github.com/wneessen/go-mail.Option) (*github.com/wneessen/go-mail.Client, error)`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/email/client.go:514](../../../plugins/email/client.go#L514) | `Send`        | SDK生产  |

#### P0401

- 定义：[plugins/email/client.go:378](../../../plugins/email/client.go#L378) `buildMailClient.opts`；分类N；阶段P5。
- 完整当前类型：`[]github.com/wneessen/go-mail.Option`。

| 位置                                                                 | 所在函数/声明     | 引用来源 |
| -------------------------------------------------------------------- | ----------------- | -------- |
| [plugins/email/client.go:387](../../../plugins/email/client.go#L387) | `buildMailClient` | SDK生产  |
| [plugins/email/client.go:387](../../../plugins/email/client.go#L387) | `buildMailClient` | SDK生产  |
| [plugins/email/client.go:389](../../../plugins/email/client.go#L389) | `buildMailClient` | SDK生产  |
| [plugins/email/client.go:389](../../../plugins/email/client.go#L389) | `buildMailClient` | SDK生产  |
| [plugins/email/client.go:392](../../../plugins/email/client.go#L392) | `buildMailClient` | SDK生产  |
| [plugins/email/client.go:392](../../../plugins/email/client.go#L392) | `buildMailClient` | SDK生产  |
| [plugins/email/client.go:396](../../../plugins/email/client.go#L396) | `buildMailClient` | SDK生产  |
| [plugins/email/client.go:396](../../../plugins/email/client.go#L396) | `buildMailClient` | SDK生产  |
| [plugins/email/client.go:399](../../../plugins/email/client.go#L399) | `buildMailClient` | SDK生产  |

#### P0402

- 定义：[plugins/email/client.go:413](../../../plugins/email/client.go#L413) `watchedConn.closeErr`；分类S；阶段P5。
- 完整当前类型：`error`。

| 位置                                                                 | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------- | ------------- | -------- |
| [plugins/email/client.go:425](../../../plugins/email/client.go#L425) | `Close`       | SDK生产  |
| [plugins/email/client.go:427](../../../plugins/email/client.go#L427) | `Close`       | SDK生产  |

#### P0403

- 定义：[plugins/email/client.go:416](../../../plugins/email/client.go#L416) `watchedConn.Close`；分类F；阶段P5。
- 完整当前类型：`func() error`。
- 方法集：`io.Closer`；`net.Conn`。

| 位置                                                                           | 所在函数/声明                                | 引用来源 |
| ------------------------------------------------------------------------------ | -------------------------------------------- | -------- |
| [plugins/email/client.go:504](../../../plugins/email/client.go#L504)           | `Send`                                       | SDK生产  |
| [plugins/email/client_test.go:883](../../../plugins/email/client_test.go#L883) | `TestClient_ContextCancelRaceAndDoubleClose` | 单测     |
| [plugins/email/client_test.go:888](../../../plugins/email/client_test.go#L888) | `TestClient_ContextCancelRaceAndDoubleClose` | 单测     |

#### P0404

- 定义：[plugins/email/client.go:434](../../../plugins/email/client.go#L434) `Client.Send`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, msg *github.com/bamboo-services/bamboo-base-go/plugins/email/v2.Message) error`。

| 位置                                                                                               | 所在函数/声明                                    | 引用来源 |
| -------------------------------------------------------------------------------------------------- | ------------------------------------------------ | -------- |
| [plugins/email/client.go:607](../../../plugins/email/client.go#L607)                               | `SendHTML`                                       | SDK生产  |
| [plugins/email/client.go:638](../../../plugins/email/client.go#L638)                               | `SendTemplate`                                   | SDK生产  |
| [plugins/email/client_test.go:496](../../../plugins/email/client_test.go#L496)                     | `TestClient_SendSuccessViaFakeSMTP`              | 单测     |
| [plugins/email/client_test.go:550](../../../plugins/email/client_test.go#L550)                     | `TestClient_ContextCancelAndTimeout`             | 单测     |
| [plugins/email/client_test.go:593](../../../plugins/email/client_test.go#L593)                     | `TestClient_ConcurrencyAndStopDraining`          | 单测     |
| [plugins/email/client_test.go:624](../../../plugins/email/client_test.go#L624)                     | `TestClient_ConcurrencyAndStopDraining`          | 单测     |
| [plugins/email/client_test.go:667](../../../plugins/email/client_test.go#L667)                     | `TestClient_WithLogger`                          | 单测     |
| [plugins/email/client_test.go:721](../../../plugins/email/client_test.go#L721)                     | `TestClient_ContextCancelDuringSend`             | 单测     |
| [plugins/email/client_test.go:763](../../../plugins/email/client_test.go#L763)                     | `TestClient_StopDrainingWaitingInflight`         | 单测     |
| [plugins/email/client_test.go:837](../../../plugins/email/client_test.go#L837)                     | `TestClient_ContextCancelRaceAndDoubleClose`     | 单测     |
| [plugins/email/client_test.go:859](../../../plugins/email/client_test.go#L859)                     | `TestClient_ContextCancelRaceAndDoubleClose`     | 单测     |
| [plugins/email/integration_smtp_test.go:168](../../../plugins/email/integration_smtp_test.go#L168) | `TestIntegration_SMTPSendAndLifecycle`           | 单测     |
| [plugins/email/integration_smtp_test.go:267](../../../plugins/email/integration_smtp_test.go#L267) | `TestIntegration_SMTPSendAndLifecycle`           | 单测     |
| [plugins/email/integration_smtp_test.go:337](../../../plugins/email/integration_smtp_test.go#L337) | `TestIntegration_StopDrainsInflightPhysicalConn` | 单测     |
| [plugins/email/integration_smtp_test.go:353](../../../plugins/email/integration_smtp_test.go#L353) | `TestIntegration_StopDrainsInflightPhysicalConn` | 单测     |

#### P0405

- 定义：[plugins/email/client.go:481](../../../plugins/email/client.go#L481) `Send::<anonymous@481>`；分类F；阶段P5。
- 完整当前类型：`func(dialCtx context.Context, network string, address string) (net.Conn, error)`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0406

- 定义：[plugins/email/client.go:606](../../../plugins/email/client.go#L606) `Client.SendHTML`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, msg *github.com/bamboo-services/bamboo-base-go/plugins/email/v2.Message) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0407

- 定义：[plugins/email/client.go:613](../../../plugins/email/client.go#L613) `Client.SendTemplate`；分类S；阶段P5。
- 完整当前类型：`func(ctx context.Context, msg *github.com/bamboo-services/bamboo-base-go/plugins/email/v2.Message) error`。

| 位置                                                                           | 所在函数/声明                       | 引用来源 |
| ------------------------------------------------------------------------------ | ----------------------------------- | -------- |
| [plugins/email/client_test.go:507](../../../plugins/email/client_test.go#L507) | `TestClient_SendSuccessViaFakeSMTP` | 单测     |

#### P0408

- 定义：[plugins/email/client.go:643](../../../plugins/email/client.go#L643) `Client.AddTemplateDir`；分类S；阶段P5。
- 完整当前类型：`func(dir string) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0409

- 定义：[plugins/email/client.go:658](../../../plugins/email/client.go#L658) `Client.AddTemplateFS`；分类S；阶段P5。
- 完整当前类型：`func(fsys io/fs.FS, pattern string) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0410

- 定义：[plugins/email/message.go:60](../../../plugins/email/message.go#L60) `Message.AttachFile`；分类S；阶段P5。
- 完整当前类型：`func(filename string) error`。

| 位置                                                                           | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------------------------ | ------------------------- | -------- |
| [plugins/email/client_test.go:207](../../../plugins/email/client_test.go#L207) | `TestMessageAttachReader` | 单测     |
| [plugins/email/client_test.go:214](../../../plugins/email/client_test.go#L214) | `TestMessageAttachReader` | 单测     |

#### P0411

- 定义：[plugins/email/template.go:24](../../../plugins/email/template.go#L24) `newTemplateManager`；分类S；阶段P5。
- 完整当前类型：`func(externalDir string) (*github.com/bamboo-services/bamboo-base-go/plugins/email/v2.TemplateManager, error)`。

| 位置                                                                           | 所在函数/声明                      | 引用来源 |
| ------------------------------------------------------------------------------ | ---------------------------------- | -------- |
| [plugins/email/client.go:253](../../../plugins/email/client.go#L253)           | `Start`                            | SDK生产  |
| [plugins/email/client_test.go:221](../../../plugins/email/client_test.go#L221) | `TestTemplateManagerRender`        | 单测     |
| [plugins/email/client_test.go:274](../../../plugins/email/client_test.go#L274) | `TestRenderExternalTemplate`       | 单测     |
| [plugins/email/client_test.go:292](../../../plugins/email/client_test.go#L292) | `TestTemplateManagerListTemplates` | 单测     |

#### P0412

- 定义：[plugins/email/template.go:47](../../../plugins/email/template.go#L47) `TemplateManager.Render`；分类S；阶段P5。
- 完整当前类型：`func(name string, data any) (string, error)`。

| 位置                                                                           | 所在函数/声明                      | 引用来源 |
| ------------------------------------------------------------------------------ | ---------------------------------- | -------- |
| [plugins/email/client.go:632](../../../plugins/email/client.go#L632)           | `SendTemplate`                     | SDK生产  |
| [plugins/email/client_test.go:230](../../../plugins/email/client_test.go#L230) | `TestTemplateManagerRender`        | 单测     |
| [plugins/email/client_test.go:242](../../../plugins/email/client_test.go#L242) | `TestTemplateManagerRender`        | 单测     |
| [plugins/email/client_test.go:254](../../../plugins/email/client_test.go#L254) | `TestTemplateManagerRender`        | 单测     |
| [plugins/email/client_test.go:279](../../../plugins/email/client_test.go#L279) | `TestRenderExternalTemplate`       | 单测     |
| [plugins/email/client_test.go:313](../../../plugins/email/client_test.go#L313) | `TestTemplateManagerListTemplates` | 单测     |

#### P0413

- 定义：[plugins/email/template.go:86](../../../plugins/email/template.go#L86) `TemplateManager.AddDir`；分类S；阶段P5。
- 完整当前类型：`func(dir string) error`。

| 位置                                                                   | 所在函数/声明        | 引用来源 |
| ---------------------------------------------------------------------- | -------------------- | -------- |
| [plugins/email/client.go:259](../../../plugins/email/client.go#L259)   | `Start`              | SDK生产  |
| [plugins/email/client.go:653](../../../plugins/email/client.go#L653)   | `AddTemplateDir`     | SDK生产  |
| [plugins/email/template.go:36](../../../plugins/email/template.go#L36) | `newTemplateManager` | SDK生产  |

#### P0414

- 定义：[plugins/email/template.go:115](../../../plugins/email/template.go#L115) `TemplateManager.AddFS`；分类S；阶段P5。
- 完整当前类型：`func(fsys io/fs.FS, pattern string) error`。

| 位置                                                                 | 所在函数/声明   | 引用来源 |
| -------------------------------------------------------------------- | --------------- | -------- |
| [plugins/email/client.go:264](../../../plugins/email/client.go#L264) | `Start`         | SDK生产  |
| [plugins/email/client.go:668](../../../plugins/email/client.go#L668) | `AddTemplateFS` | SDK生产  |

#### P0415

- 定义：[plugins/grpc/grpcresult/response.go:128](../../../plugins/grpc/grpcresult/response.go#L128) `ToStatusError`；分类F；阶段P6。
- 完整当前类型：`func(ctx context.Context, err *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error) error`。

| 位置                                                                                                     | 所在函数/声明                                 | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | --------------------------------------------- | -------- |
| [plugins/grpc/grpcresult/error_code_test.go:60](../../../plugins/grpc/grpcresult/error_code_test.go#L60) | `TestProtocolErrorCodes`                      | 单测     |
| [plugins/grpc/grpcresult/example_test.go:50](../../../plugins/grpc/grpcresult/example_test.go#L50)       | `ExampleToStatusError`                        | 单测     |
| [plugins/grpc/grpcresult/response_test.go:62](../../../plugins/grpc/grpcresult/response_test.go#L62)     | `TestToStatusError_NilError`                  | 单测     |
| [plugins/grpc/grpcresult/response_test.go:120](../../../plugins/grpc/grpcresult/response_test.go#L120)   | `TestToStatusError_GroupMapping`              | 单测     |
| [plugins/grpc/grpcresult/response_test.go:148](../../../plugins/grpc/grpcresult/response_test.go#L148)   | `TestToStatusError_SecurityRawErrorNotLeaked` | 单测     |
| [plugins/grpc/plugin.go:606](../../../plugins/grpc/plugin.go#L606)                                       | `normalizeRPCError`                           | SDK生产  |
| [plugins/grpc/plugin.go:625](../../../plugins/grpc/plugin.go#L625)                                       | `normalizeRPCError`                           | SDK生产  |
| [plugins/grpc/plugin.go:633](../../../plugins/grpc/plugin.go#L633)                                       | `unaryBridgeInterceptor`                      | SDK生产  |
| [plugins/grpc/plugin.go:668](../../../plugins/grpc/plugin.go#L668)                                       | `unaryBridgeInterceptor`                      | SDK生产  |
| [plugins/grpc/plugin.go:694](../../../plugins/grpc/plugin.go#L694)                                       | `unaryBridgeInterceptor`                      | SDK生产  |
| [plugins/grpc/plugin.go:706](../../../plugins/grpc/plugin.go#L706)                                       | `streamBridgeInterceptor`                     | SDK生产  |
| [plugins/grpc/plugin.go:739](../../../plugins/grpc/plugin.go#L739)                                       | `streamBridgeInterceptor`                     | SDK生产  |

#### P0416

- 定义：[plugins/grpc/middleware/middleware.go:13](../../../plugins/grpc/middleware/middleware.go#L13) `UnaryMiddlewareFunc`；分类F；阶段P6。
- 完整当前类型：`func(ctx context.Context, req any, info *google.golang.org/grpc.UnaryServerInfo, handler google.golang.org/grpc.UnaryHandler) (resp any, err error)`。

| 位置                                                                                                     | 所在函数/声明           | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | ----------------------- | -------- |
| [plugins/grpc/middleware/middleware.go:27](../../../plugins/grpc/middleware/middleware.go#L27)           | `package-level`         | SDK生产  |
| [plugins/grpc/middleware/middleware.go:34](../../../plugins/grpc/middleware/middleware.go#L34)           | `NewRegistry`           | SDK生产  |
| [plugins/grpc/middleware/middleware.go:52](../../../plugins/grpc/middleware/middleware.go#L52)           | `Clone`                 | SDK生产  |
| [plugins/grpc/middleware/middleware.go:65](../../../plugins/grpc/middleware/middleware.go#L65)           | `UseUnary`              | SDK生产  |
| [plugins/grpc/middleware/middleware.go:94](../../../plugins/grpc/middleware/middleware.go#L94)           | `LookupUnary`           | SDK生产  |
| [plugins/grpc/middleware/middleware.go:128](../../../plugins/grpc/middleware/middleware.go#L128)         | `filterNilUnary`        | SDK生产  |
| [plugins/grpc/middleware/middleware.go:128](../../../plugins/grpc/middleware/middleware.go#L128)         | `filterNilUnary`        | SDK生产  |
| [plugins/grpc/middleware/middleware.go:129](../../../plugins/grpc/middleware/middleware.go#L129)         | `filterNilUnary`        | SDK生产  |
| [plugins/grpc/middleware/middleware_test.go:14](../../../plugins/grpc/middleware/middleware_test.go#L14) | `dummyUnaryInterceptor` | 单测     |

#### P0417

- 定义：[plugins/grpc/middleware/middleware.go:18](../../../plugins/grpc/middleware/middleware.go#L18) `StreamMiddlewareFunc`；分类F；阶段P6。
- 完整当前类型：`func(srv any, ss google.golang.org/grpc.ServerStream, info *google.golang.org/grpc.StreamServerInfo, handler google.golang.org/grpc.StreamHandler) error`。

| 位置                                                                                                     | 所在函数/声明            | 引用来源 |
| -------------------------------------------------------------------------------------------------------- | ------------------------ | -------- |
| [plugins/grpc/middleware/middleware.go:28](../../../plugins/grpc/middleware/middleware.go#L28)           | `package-level`          | SDK生产  |
| [plugins/grpc/middleware/middleware.go:35](../../../plugins/grpc/middleware/middleware.go#L35)           | `NewRegistry`            | SDK生产  |
| [plugins/grpc/middleware/middleware.go:55](../../../plugins/grpc/middleware/middleware.go#L55)           | `Clone`                  | SDK生产  |
| [plugins/grpc/middleware/middleware.go:80](../../../plugins/grpc/middleware/middleware.go#L80)           | `UseStream`              | SDK生产  |
| [plugins/grpc/middleware/middleware.go:104](../../../plugins/grpc/middleware/middleware.go#L104)         | `LookupStream`           | SDK生产  |
| [plugins/grpc/middleware/middleware.go:139](../../../plugins/grpc/middleware/middleware.go#L139)         | `filterNilStream`        | SDK生产  |
| [plugins/grpc/middleware/middleware.go:139](../../../plugins/grpc/middleware/middleware.go#L139)         | `filterNilStream`        | SDK生产  |
| [plugins/grpc/middleware/middleware.go:140](../../../plugins/grpc/middleware/middleware.go#L140)         | `filterNilStream`        | SDK生产  |
| [plugins/grpc/middleware/middleware_test.go:24](../../../plugins/grpc/middleware/middleware_test.go#L24) | `dummyStreamInterceptor` | 单测     |

#### P0418

- 定义：[plugins/grpc/middleware/middleware.go:27](../../../plugins/grpc/middleware/middleware.go#L27) `Registry.unaryStore`；分类F；阶段P6。
- 完整当前类型：`map[string][]github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.UnaryMiddlewareFunc`。

| 位置                                                                                           | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/grpc/middleware/middleware.go:34](../../../plugins/grpc/middleware/middleware.go#L34) | `NewRegistry` | SDK生产  |
| [plugins/grpc/middleware/middleware.go:51](../../../plugins/grpc/middleware/middleware.go#L51) | `Clone`       | SDK生产  |
| [plugins/grpc/middleware/middleware.go:52](../../../plugins/grpc/middleware/middleware.go#L52) | `Clone`       | SDK生产  |
| [plugins/grpc/middleware/middleware.go:72](../../../plugins/grpc/middleware/middleware.go#L72) | `UseUnary`    | SDK生产  |
| [plugins/grpc/middleware/middleware.go:72](../../../plugins/grpc/middleware/middleware.go#L72) | `UseUnary`    | SDK生产  |
| [plugins/grpc/middleware/middleware.go:97](../../../plugins/grpc/middleware/middleware.go#L97) | `LookupUnary` | SDK生产  |

#### P0419

- 定义：[plugins/grpc/middleware/middleware.go:28](../../../plugins/grpc/middleware/middleware.go#L28) `Registry.streamStore`；分类F；阶段P6。
- 完整当前类型：`map[string][]github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.StreamMiddlewareFunc`。

| 位置                                                                                             | 所在函数/声明  | 引用来源 |
| ------------------------------------------------------------------------------------------------ | -------------- | -------- |
| [plugins/grpc/middleware/middleware.go:35](../../../plugins/grpc/middleware/middleware.go#L35)   | `NewRegistry`  | SDK生产  |
| [plugins/grpc/middleware/middleware.go:54](../../../plugins/grpc/middleware/middleware.go#L54)   | `Clone`        | SDK生产  |
| [plugins/grpc/middleware/middleware.go:55](../../../plugins/grpc/middleware/middleware.go#L55)   | `Clone`        | SDK生产  |
| [plugins/grpc/middleware/middleware.go:87](../../../plugins/grpc/middleware/middleware.go#L87)   | `UseStream`    | SDK生产  |
| [plugins/grpc/middleware/middleware.go:87](../../../plugins/grpc/middleware/middleware.go#L87)   | `UseStream`    | SDK生产  |
| [plugins/grpc/middleware/middleware.go:107](../../../plugins/grpc/middleware/middleware.go#L107) | `LookupStream` | SDK生产  |

#### P0420

- 定义：[plugins/grpc/middleware/middleware.go:65](../../../plugins/grpc/middleware/middleware.go#L65) `Registry.UseUnary`；分类F；阶段P6。
- 完整当前类型：`func(desc google.golang.org/grpc.ServiceDesc, middlewares ...github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.UnaryMiddlewareFunc)`。

| 位置                                                                                                       | 所在函数/声明                                        | 引用来源 |
| ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------- | -------- |
| [plugins/grpc/middleware/example_test.go:39](../../../plugins/grpc/middleware/example_test.go#L39)         | `ExampleRegistry`                                    | 单测     |
| [plugins/grpc/middleware/middleware_test.go:36](../../../plugins/grpc/middleware/middleware_test.go#L36)   | `TestUseUnaryBasic`                                  | 单测     |
| [plugins/grpc/middleware/middleware_test.go:52](../../../plugins/grpc/middleware/middleware_test.go#L52)   | `TestUseUnaryAppend`                                 | 单测     |
| [plugins/grpc/middleware/middleware_test.go:53](../../../plugins/grpc/middleware/middleware_test.go#L53)   | `TestUseUnaryAppend`                                 | 单测     |
| [plugins/grpc/middleware/middleware_test.go:92](../../../plugins/grpc/middleware/middleware_test.go#L92)   | `TestUseMultipleServices`                            | 单测     |
| [plugins/grpc/middleware/middleware_test.go:93](../../../plugins/grpc/middleware/middleware_test.go#L93)   | `TestUseMultipleServices`                            | 单测     |
| [plugins/grpc/middleware/middleware_test.go:116](../../../plugins/grpc/middleware/middleware_test.go#L116) | `TestUseAllNilSkipped`                               | 单测     |
| [plugins/grpc/middleware/middleware_test.go:148](../../../plugins/grpc/middleware/middleware_test.go#L148) | `TestConcurrentSafe`                                 | 单测     |
| [plugins/grpc/middleware/middleware_test.go:196](../../../plugins/grpc/middleware/middleware_test.go#L196) | `TestRegistryIsolation`                              | 单测     |
| [plugins/grpc/middleware/middleware_test.go:197](../../../plugins/grpc/middleware/middleware_test.go#L197) | `TestRegistryIsolation`                              | 单测     |
| [plugins/grpc/middleware/middleware_test.go:239](../../../plugins/grpc/middleware/middleware_test.go#L239) | `TestMiddlewareExecutionChainAndErrorPropagation`    | 单测     |
| [plugins/grpc/plugin_app_context_test.go:462](../../../plugins/grpc/plugin_app_context_test.go#L462)       | `TestPlugin_AppContext_RealTCP_IsolationAndRegistry` | 单测     |
| [plugins/grpc/plugin_app_context_test.go:473](../../../plugins/grpc/plugin_app_context_test.go#L473)       | `TestPlugin_AppContext_RealTCP_IsolationAndRegistry` | 单测     |
| [plugins/grpc/plugin_app_context_test.go:492](../../../plugins/grpc/plugin_app_context_test.go#L492)       | `TestPlugin_AppContext_RealTCP_IsolationAndRegistry` | 单测     |
| [plugins/grpc/plugin_test.go:329](../../../plugins/grpc/plugin_test.go#L329)                               | `TestPlugin_MultiInstanceIsolation`                  | 单测     |
| [plugins/grpc/plugin_test.go:336](../../../plugins/grpc/plugin_test.go#L336)                               | `TestPlugin_MultiInstanceIsolation`                  | 单测     |

#### P0421

- 定义：[plugins/grpc/middleware/middleware.go:66](../../../plugins/grpc/middleware/middleware.go#L66) `UseUnary.filtered`；分类F；阶段P6。
- 完整当前类型：`[]github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.UnaryMiddlewareFunc`。

| 位置                                                                                           | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/grpc/middleware/middleware.go:67](../../../plugins/grpc/middleware/middleware.go#L67) | `UseUnary`    | SDK生产  |
| [plugins/grpc/middleware/middleware.go:72](../../../plugins/grpc/middleware/middleware.go#L72) | `UseUnary`    | SDK生产  |

#### P0422

- 定义：[plugins/grpc/middleware/middleware.go:80](../../../plugins/grpc/middleware/middleware.go#L80) `Registry.UseStream`；分类F；阶段P6。
- 完整当前类型：`func(desc google.golang.org/grpc.ServiceDesc, middlewares ...github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.StreamMiddlewareFunc)`。

| 位置                                                                                                       | 所在函数/声明                                        | 引用来源 |
| ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------- | -------- |
| [plugins/grpc/middleware/middleware_test.go:64](../../../plugins/grpc/middleware/middleware_test.go#L64)   | `TestUseStreamBasic`                                 | 单测     |
| [plugins/grpc/middleware/middleware_test.go:80](../../../plugins/grpc/middleware/middleware_test.go#L80)   | `TestUseStreamAppend`                                | 单测     |
| [plugins/grpc/middleware/middleware_test.go:81](../../../plugins/grpc/middleware/middleware_test.go#L81)   | `TestUseStreamAppend`                                | 单测     |
| [plugins/grpc/middleware/middleware_test.go:94](../../../plugins/grpc/middleware/middleware_test.go#L94)   | `TestUseMultipleServices`                            | 单测     |
| [plugins/grpc/middleware/middleware_test.go:121](../../../plugins/grpc/middleware/middleware_test.go#L121) | `TestUseAllNilSkipped`                               | 单测     |
| [plugins/grpc/middleware/middleware_test.go:161](../../../plugins/grpc/middleware/middleware_test.go#L161) | `TestConcurrentSafe`                                 | 单测     |
| [plugins/grpc/plugin_app_context_test.go:508](../../../plugins/grpc/plugin_app_context_test.go#L508)       | `TestPlugin_AppContext_RealTCP_IsolationAndRegistry` | 单测     |
| [plugins/grpc/plugin_app_context_test.go:522](../../../plugins/grpc/plugin_app_context_test.go#L522)       | `TestPlugin_AppContext_RealTCP_IsolationAndRegistry` | 单测     |

#### P0423

- 定义：[plugins/grpc/middleware/middleware.go:81](../../../plugins/grpc/middleware/middleware.go#L81) `UseStream.filtered`；分类F；阶段P6。
- 完整当前类型：`[]github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.StreamMiddlewareFunc`。

| 位置                                                                                           | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/grpc/middleware/middleware.go:82](../../../plugins/grpc/middleware/middleware.go#L82) | `UseStream`   | SDK生产  |
| [plugins/grpc/middleware/middleware.go:87](../../../plugins/grpc/middleware/middleware.go#L87) | `UseStream`   | SDK生产  |

#### P0424

- 定义：[plugins/grpc/middleware/middleware.go:94](../../../plugins/grpc/middleware/middleware.go#L94) `Registry.LookupUnary`；分类F；阶段P6。
- 完整当前类型：`func(serviceName string) []github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.UnaryMiddlewareFunc`。

| 位置                                                                                                       | 所在函数/声明                                     | 引用来源 |
| ---------------------------------------------------------------------------------------------------------- | ------------------------------------------------- | -------- |
| [plugins/grpc/middleware/example_test.go:42](../../../plugins/grpc/middleware/example_test.go#L42)         | `ExampleRegistry`                                 | 单测     |
| [plugins/grpc/middleware/example_test.go:46](../../../plugins/grpc/middleware/example_test.go#L46)         | `ExampleRegistry`                                 | 单测     |
| [plugins/grpc/middleware/middleware_test.go:42](../../../plugins/grpc/middleware/middleware_test.go#L42)   | `TestUseUnaryBasic`                               | 单测     |
| [plugins/grpc/middleware/middleware_test.go:55](../../../plugins/grpc/middleware/middleware_test.go#L55)   | `TestUseUnaryAppend`                              | 单测     |
| [plugins/grpc/middleware/middleware_test.go:96](../../../plugins/grpc/middleware/middleware_test.go#L96)   | `TestUseMultipleServices`                         | 单测     |
| [plugins/grpc/middleware/middleware_test.go:99](../../../plugins/grpc/middleware/middleware_test.go#L99)   | `TestUseMultipleServices`                         | 单测     |
| [plugins/grpc/middleware/middleware_test.go:102](../../../plugins/grpc/middleware/middleware_test.go#L102) | `TestUseMultipleServices`                         | 单测     |
| [plugins/grpc/middleware/middleware_test.go:117](../../../plugins/grpc/middleware/middleware_test.go#L117) | `TestUseAllNilSkipped`                            | 单测     |
| [plugins/grpc/middleware/middleware_test.go:130](../../../plugins/grpc/middleware/middleware_test.go#L130) | `TestLookupEmpty`                                 | 单测     |
| [plugins/grpc/middleware/middleware_test.go:173](../../../plugins/grpc/middleware/middleware_test.go#L173) | `TestConcurrentSafe`                              | 单测     |
| [plugins/grpc/middleware/middleware_test.go:180](../../../plugins/grpc/middleware/middleware_test.go#L180) | `TestConcurrentSafe`                              | 单测     |
| [plugins/grpc/middleware/middleware_test.go:199](../../../plugins/grpc/middleware/middleware_test.go#L199) | `TestRegistryIsolation`                           | 单测     |
| [plugins/grpc/middleware/middleware_test.go:200](../../../plugins/grpc/middleware/middleware_test.go#L200) | `TestRegistryIsolation`                           | 单测     |
| [plugins/grpc/middleware/middleware_test.go:240](../../../plugins/grpc/middleware/middleware_test.go#L240) | `TestMiddlewareExecutionChainAndErrorPropagation` | 单测     |
| [plugins/grpc/plugin.go:674](../../../plugins/grpc/plugin.go#L674)                                         | `unaryBridgeInterceptor`                          | SDK生产  |

#### P0425

- 定义：[plugins/grpc/middleware/middleware.go:104](../../../plugins/grpc/middleware/middleware.go#L104) `Registry.LookupStream`；分类F；阶段P6。
- 完整当前类型：`func(serviceName string) []github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.StreamMiddlewareFunc`。

| 位置                                                                                                       | 所在函数/声明             | 引用来源 |
| ---------------------------------------------------------------------------------------------------------- | ------------------------- | -------- |
| [plugins/grpc/middleware/middleware_test.go:70](../../../plugins/grpc/middleware/middleware_test.go#L70)   | `TestUseStreamBasic`      | 单测     |
| [plugins/grpc/middleware/middleware_test.go:83](../../../plugins/grpc/middleware/middleware_test.go#L83)   | `TestUseStreamAppend`     | 单测     |
| [plugins/grpc/middleware/middleware_test.go:105](../../../plugins/grpc/middleware/middleware_test.go#L105) | `TestUseMultipleServices` | 单测     |
| [plugins/grpc/middleware/middleware_test.go:108](../../../plugins/grpc/middleware/middleware_test.go#L108) | `TestUseMultipleServices` | 单测     |
| [plugins/grpc/middleware/middleware_test.go:122](../../../plugins/grpc/middleware/middleware_test.go#L122) | `TestUseAllNilSkipped`    | 单测     |
| [plugins/grpc/middleware/middleware_test.go:133](../../../plugins/grpc/middleware/middleware_test.go#L133) | `TestLookupEmpty`         | 单测     |
| [plugins/grpc/middleware/middleware_test.go:174](../../../plugins/grpc/middleware/middleware_test.go#L174) | `TestConcurrentSafe`      | 单测     |
| [plugins/grpc/middleware/middleware_test.go:185](../../../plugins/grpc/middleware/middleware_test.go#L185) | `TestConcurrentSafe`      | 单测     |
| [plugins/grpc/plugin.go:744](../../../plugins/grpc/plugin.go#L744)                                         | `streamBridgeInterceptor` | SDK生产  |

#### P0426

- 定义：[plugins/grpc/middleware/middleware.go:128](../../../plugins/grpc/middleware/middleware.go#L128) `filterNilUnary`；分类F；阶段P6。
- 完整当前类型：`func(middlewares []github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.UnaryMiddlewareFunc) []github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.UnaryMiddlewareFunc`。

| 位置                                                                                           | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/grpc/middleware/middleware.go:66](../../../plugins/grpc/middleware/middleware.go#L66) | `UseUnary`    | SDK生产  |

#### P0427

- 定义：[plugins/grpc/middleware/middleware.go:129](../../../plugins/grpc/middleware/middleware.go#L129) `filterNilUnary.filtered`；分类F；阶段P6。
- 完整当前类型：`[]github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.UnaryMiddlewareFunc`。

| 位置                                                                                             | 所在函数/声明    | 引用来源 |
| ------------------------------------------------------------------------------------------------ | ---------------- | -------- |
| [plugins/grpc/middleware/middleware.go:132](../../../plugins/grpc/middleware/middleware.go#L132) | `filterNilUnary` | SDK生产  |
| [plugins/grpc/middleware/middleware.go:132](../../../plugins/grpc/middleware/middleware.go#L132) | `filterNilUnary` | SDK生产  |
| [plugins/grpc/middleware/middleware.go:135](../../../plugins/grpc/middleware/middleware.go#L135) | `filterNilUnary` | SDK生产  |

#### P0428

- 定义：[plugins/grpc/middleware/middleware.go:139](../../../plugins/grpc/middleware/middleware.go#L139) `filterNilStream`；分类F；阶段P6。
- 完整当前类型：`func(middlewares []github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.StreamMiddlewareFunc) []github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.StreamMiddlewareFunc`。

| 位置                                                                                           | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/grpc/middleware/middleware.go:81](../../../plugins/grpc/middleware/middleware.go#L81) | `UseStream`   | SDK生产  |

#### P0429

- 定义：[plugins/grpc/middleware/middleware.go:140](../../../plugins/grpc/middleware/middleware.go#L140) `filterNilStream.filtered`；分类F；阶段P6。
- 完整当前类型：`[]github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.StreamMiddlewareFunc`。

| 位置                                                                                             | 所在函数/声明     | 引用来源 |
| ------------------------------------------------------------------------------------------------ | ----------------- | -------- |
| [plugins/grpc/middleware/middleware.go:143](../../../plugins/grpc/middleware/middleware.go#L143) | `filterNilStream` | SDK生产  |
| [plugins/grpc/middleware/middleware.go:143](../../../plugins/grpc/middleware/middleware.go#L143) | `filterNilStream` | SDK生产  |
| [plugins/grpc/middleware/middleware.go:146](../../../plugins/grpc/middleware/middleware.go#L146) | `filterNilStream` | SDK生产  |

#### P0430

- 定义：[plugins/grpc/plugin.go:34](../../../plugins/grpc/plugin.go#L34) `ErrServerAlreadyStarted`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/grpc/plugin.go:398](../../../plugins/grpc/plugin.go#L398) | `Start`       | SDK生产  |

#### P0431

- 定义：[plugins/grpc/plugin.go:35](../../../plugins/grpc/plugin.go#L35) `ErrServerClosed`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                               | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------------ | ------------------------- | -------- |
| [plugins/grpc/plugin.go:633](../../../plugins/grpc/plugin.go#L633) | `unaryBridgeInterceptor`  | SDK生产  |
| [plugins/grpc/plugin.go:706](../../../plugins/grpc/plugin.go#L706) | `streamBridgeInterceptor` | SDK生产  |

#### P0432

- 定义：[plugins/grpc/plugin.go:36](../../../plugins/grpc/plugin.go#L36) `ErrNoLifetimeContext`；分类N；阶段P6。
- 完整当前类型：`error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0433

- 定义：[plugins/grpc/plugin.go:37](../../../plugins/grpc/plugin.go#L37) `ErrStopTimeout`；分类N；阶段P6。
- 完整当前类型：`error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0434

- 定义：[plugins/grpc/plugin.go:61](../../../plugins/grpc/plugin.go#L61) `config.unaryInterceptors`；分类F；阶段P6。
- 完整当前类型：`[]google.golang.org/grpc.UnaryServerInterceptor`。

| 位置                                                               | 所在函数/声明           | 引用来源 |
| ------------------------------------------------------------------ | ----------------------- | -------- |
| [plugins/grpc/plugin.go:151](../../../plugins/grpc/plugin.go#L151) | `WithUnaryInterceptors` | SDK生产  |
| [plugins/grpc/plugin.go:151](../../../plugins/grpc/plugin.go#L151) | `WithUnaryInterceptors` | SDK生产  |
| [plugins/grpc/plugin.go:335](../../../plugins/grpc/plugin.go#L335) | `New`                   | SDK生产  |
| [plugins/grpc/plugin.go:335](../../../plugins/grpc/plugin.go#L335) | `New`                   | SDK生产  |
| [plugins/grpc/plugin.go:421](../../../plugins/grpc/plugin.go#L421) | `Start`                 | SDK生产  |

#### P0435

- 定义：[plugins/grpc/plugin.go:62](../../../plugins/grpc/plugin.go#L62) `config.streamInterceptors`；分类F；阶段P6。
- 完整当前类型：`[]google.golang.org/grpc.StreamServerInterceptor`。

| 位置                                                               | 所在函数/声明            | 引用来源 |
| ------------------------------------------------------------------ | ------------------------ | -------- |
| [plugins/grpc/plugin.go:160](../../../plugins/grpc/plugin.go#L160) | `WithStreamInterceptors` | SDK生产  |
| [plugins/grpc/plugin.go:160](../../../plugins/grpc/plugin.go#L160) | `WithStreamInterceptors` | SDK生产  |
| [plugins/grpc/plugin.go:336](../../../plugins/grpc/plugin.go#L336) | `New`                    | SDK生产  |
| [plugins/grpc/plugin.go:336](../../../plugins/grpc/plugin.go#L336) | `New`                    | SDK生产  |
| [plugins/grpc/plugin.go:426](../../../plugins/grpc/plugin.go#L426) | `Start`                  | SDK生产  |

#### P0436

- 定义：[plugins/grpc/plugin.go:149](../../../plugins/grpc/plugin.go#L149) `WithUnaryInterceptors`；分类F；阶段P6。
- 完整当前类型：`func(interceptors ...google.golang.org/grpc.UnaryServerInterceptor) github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2.Option`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0437

- 定义：[plugins/grpc/plugin.go:158](../../../plugins/grpc/plugin.go#L158) `WithStreamInterceptors`；分类F；阶段P6。
- 完整当前类型：`func(interceptors ...google.golang.org/grpc.StreamServerInterceptor) github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2.Option`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0438

- 定义：[plugins/grpc/plugin.go:230](../../../plugins/grpc/plugin.go#L230) `requestGate.waitDrained`；分类S；阶段P6。
- 完整当前类型：`func(ctx context.Context) error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/grpc/plugin.go:562](../../../plugins/grpc/plugin.go#L562) | `doStop`      | SDK生产  |

#### P0439

- 定义：[plugins/grpc/plugin.go:284](../../../plugins/grpc/plugin.go#L284) `Plugin.stopErr`；分类S；阶段P6。
- 完整当前类型：`error`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/grpc/plugin.go:507](../../../plugins/grpc/plugin.go#L507) | `Stop`        | SDK生产  |
| [plugins/grpc/plugin.go:511](../../../plugins/grpc/plugin.go#L511) | `Stop`        | SDK生产  |

#### P0440

- 定义：[plugins/grpc/plugin.go:396](../../../plugins/grpc/plugin.go#L396) `Plugin.Start`；分类S；阶段P3/P6。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`；`github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`。

| 位置                                                                         | 所在函数/声明                           | 引用来源 |
| ---------------------------------------------------------------------------- | --------------------------------------- | -------- |
| [plugins/grpc/plugin_test.go:175](../../../plugins/grpc/plugin_test.go#L175) | `TestPlugin_BasicLifecycleAndUnary`     | 单测     |
| [plugins/grpc/plugin_test.go:271](../../../plugins/grpc/plugin_test.go#L271) | `TestPlugin_StreamAndPanicSafety`       | 单测     |
| [plugins/grpc/plugin_test.go:359](../../../plugins/grpc/plugin_test.go#L359) | `TestPlugin_MultiInstanceIsolation`     | 单测     |
| [plugins/grpc/plugin_test.go:364](../../../plugins/grpc/plugin_test.go#L364) | `TestPlugin_MultiInstanceIsolation`     | 单测     |
| [plugins/grpc/plugin_test.go:405](../../../plugins/grpc/plugin_test.go#L405) | `TestPlugin_GracefulStopAndForceCancel` | 单测     |

#### P0441

- 定义：[plugins/grpc/plugin.go:418](../../../plugins/grpc/plugin.go#L418) `Start.unaryChain`；分类F；阶段P6。
- 完整当前类型：`[]google.golang.org/grpc.UnaryServerInterceptor`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/grpc/plugin.go:421](../../../plugins/grpc/plugin.go#L421) | `Start`       | SDK生产  |
| [plugins/grpc/plugin.go:421](../../../plugins/grpc/plugin.go#L421) | `Start`       | SDK生产  |
| [plugins/grpc/plugin.go:429](../../../plugins/grpc/plugin.go#L429) | `Start`       | SDK生产  |

#### P0442

- 定义：[plugins/grpc/plugin.go:423](../../../plugins/grpc/plugin.go#L423) `Start.streamChain`；分类F；阶段P6。
- 完整当前类型：`[]google.golang.org/grpc.StreamServerInterceptor`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/grpc/plugin.go:426](../../../plugins/grpc/plugin.go#L426) | `Start`       | SDK生产  |
| [plugins/grpc/plugin.go:426](../../../plugins/grpc/plugin.go#L426) | `Start`       | SDK生产  |
| [plugins/grpc/plugin.go:430](../../../plugins/grpc/plugin.go#L430) | `Start`       | SDK生产  |

#### P0443

- 定义：[plugins/grpc/plugin.go:469](../../../plugins/grpc/plugin.go#L469) `Plugin.Run`；分类S；阶段P3/P6。
- 完整当前类型：`func(lifetimeCtx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`。

| 位置                                                                         | 所在函数/声明                           | 引用来源 |
| ---------------------------------------------------------------------------- | --------------------------------------- | -------- |
| [plugins/grpc/plugin_test.go:189](../../../plugins/grpc/plugin_test.go#L189) | `TestPlugin_BasicLifecycleAndUnary`     | 单测     |
| [plugins/grpc/plugin_test.go:276](../../../plugins/grpc/plugin_test.go#L276) | `TestPlugin_StreamAndPanicSafety`       | 单测     |
| [plugins/grpc/plugin_test.go:369](../../../plugins/grpc/plugin_test.go#L369) | `TestPlugin_MultiInstanceIsolation`     | 单测     |
| [plugins/grpc/plugin_test.go:370](../../../plugins/grpc/plugin_test.go#L370) | `TestPlugin_MultiInstanceIsolation`     | 单测     |
| [plugins/grpc/plugin_test.go:408](../../../plugins/grpc/plugin_test.go#L408) | `TestPlugin_GracefulStopAndForceCancel` | 单测     |

#### P0444

- 定义：[plugins/grpc/plugin.go:490](../../../plugins/grpc/plugin.go#L490) `Plugin.Quiesce`；分类S；阶段P3/P6。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0445

- 定义：[plugins/grpc/plugin.go:505](../../../plugins/grpc/plugin.go#L505) `Plugin.Stop`；分类S；阶段P3/P6。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`；`github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`。

| 位置                                                                         | 所在函数/声明                           | 引用来源 |
| ---------------------------------------------------------------------------- | --------------------------------------- | -------- |
| [plugins/grpc/plugin_test.go:245](../../../plugins/grpc/plugin_test.go#L245) | `TestPlugin_BasicLifecycleAndUnary`     | 单测     |
| [plugins/grpc/plugin_test.go:322](../../../plugins/grpc/plugin_test.go#L322) | `TestPlugin_StreamAndPanicSafety`       | 单测     |
| [plugins/grpc/plugin_test.go:362](../../../plugins/grpc/plugin_test.go#L362) | `TestPlugin_MultiInstanceIsolation`     | 单测     |
| [plugins/grpc/plugin_test.go:367](../../../plugins/grpc/plugin_test.go#L367) | `TestPlugin_MultiInstanceIsolation`     | 单测     |
| [plugins/grpc/plugin_test.go:436](../../../plugins/grpc/plugin_test.go#L436) | `TestPlugin_GracefulStopAndForceCancel` | 单测     |

#### P0446

- 定义：[plugins/grpc/plugin.go:514](../../../plugins/grpc/plugin.go#L514) `Plugin.doStop`；分类S；阶段P6。
- 完整当前类型：`func(ctx context.Context) (stopReason error)`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/grpc/plugin.go:507](../../../plugins/grpc/plugin.go#L507) | `Stop`        | SDK生产  |

#### P0447

- 定义：[plugins/grpc/plugin.go:599](../../../plugins/grpc/plugin.go#L599) `Plugin.normalizeRPCError`；分类X；阶段P6。
- 完整当前类型：`func(ctx context.Context, err error) error`。

| 位置                                                               | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------------ | ------------------------- | -------- |
| [plugins/grpc/plugin.go:691](../../../plugins/grpc/plugin.go#L691) | `unaryBridgeInterceptor`  | SDK生产  |
| [plugins/grpc/plugin.go:760](../../../plugins/grpc/plugin.go#L760) | `streamBridgeInterceptor` | SDK生产  |

#### P0448

- 定义：[plugins/grpc/plugin.go:630](../../../plugins/grpc/plugin.go#L630) `Plugin.unaryBridgeInterceptor`；分类F；阶段P6。
- 完整当前类型：`func() google.golang.org/grpc.UnaryServerInterceptor`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/grpc/plugin.go:419](../../../plugins/grpc/plugin.go#L419) | `Start`       | SDK生产  |

#### P0449

- 定义：[plugins/grpc/plugin.go:631](../../../plugins/grpc/plugin.go#L631) `unaryBridgeInterceptor::<anonymous@631>`；分类F；阶段P6。
- 完整当前类型：`func(ctx context.Context, req any, info *google.golang.org/grpc.UnaryServerInfo, handler google.golang.org/grpc.UnaryHandler) (resp any, err error)`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0450

- 定义：[plugins/grpc/plugin.go:674](../../../plugins/grpc/plugin.go#L674) `unaryBridgeInterceptor.mws`；分类F；阶段P6。
- 完整当前类型：`[]github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.UnaryMiddlewareFunc`。

| 位置                                                               | 所在函数/声明            | 引用来源 |
| ------------------------------------------------------------------ | ------------------------ | -------- |
| [plugins/grpc/plugin.go:676](../../../plugins/grpc/plugin.go#L676) | `unaryBridgeInterceptor` | SDK生产  |
| [plugins/grpc/plugin.go:678](../../../plugins/grpc/plugin.go#L678) | `unaryBridgeInterceptor` | SDK生产  |
| [plugins/grpc/plugin.go:679](../../../plugins/grpc/plugin.go#L679) | `unaryBridgeInterceptor` | SDK生产  |

#### P0451

- 定义：[plugins/grpc/plugin.go:681](../../../plugins/grpc/plugin.go#L681) `unaryBridgeInterceptor::<anonymous@681>`；分类F；阶段P6。
- 完整当前类型：`func(c context.Context, r any) (any, error)`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0452

- 定义：[plugins/grpc/plugin.go:703](../../../plugins/grpc/plugin.go#L703) `Plugin.streamBridgeInterceptor`；分类F；阶段P6。
- 完整当前类型：`func() google.golang.org/grpc.StreamServerInterceptor`。

| 位置                                                               | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------ | ------------- | -------- |
| [plugins/grpc/plugin.go:424](../../../plugins/grpc/plugin.go#L424) | `Start`       | SDK生产  |

#### P0453

- 定义：[plugins/grpc/plugin.go:704](../../../plugins/grpc/plugin.go#L704) `streamBridgeInterceptor::<anonymous@704>`；分类F；阶段P6。
- 完整当前类型：`func(srv any, ss google.golang.org/grpc.ServerStream, info *google.golang.org/grpc.StreamServerInfo, handler google.golang.org/grpc.StreamHandler) (err error)`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0454

- 定义：[plugins/grpc/plugin.go:744](../../../plugins/grpc/plugin.go#L744) `streamBridgeInterceptor.mws`；分类F；阶段P6。
- 完整当前类型：`[]github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/middleware.StreamMiddlewareFunc`。

| 位置                                                               | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------------ | ------------------------- | -------- |
| [plugins/grpc/plugin.go:746](../../../plugins/grpc/plugin.go#L746) | `streamBridgeInterceptor` | SDK生产  |
| [plugins/grpc/plugin.go:748](../../../plugins/grpc/plugin.go#L748) | `streamBridgeInterceptor` | SDK生产  |
| [plugins/grpc/plugin.go:749](../../../plugins/grpc/plugin.go#L749) | `streamBridgeInterceptor` | SDK生产  |

#### P0455

- 定义：[plugins/grpc/plugin.go:751](../../../plugins/grpc/plugin.go#L751) `streamBridgeInterceptor::<anonymous@751>`；分类F；阶段P6。
- 完整当前类型：`func(s any, stream google.golang.org/grpc.ServerStream) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0456

- 定义：[plugins/grpc/utility/grpc_util.go:27](../../../plugins/grpc/utility/grpc_util.go#L27) `ExtractMetadata`；分类A；阶段P6。
- 完整当前类型：`func(ctx context.Context, key github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant.Metadata) (string, *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error)`。

| 位置                                                                                             | 所在函数/声明            | 引用来源 |
| ------------------------------------------------------------------------------------------------ | ------------------------ | -------- |
| [plugins/grpc/utility/example_test.go:22](../../../plugins/grpc/utility/example_test.go#L22)     | `ExampleExtractMetadata` | 单测     |
| [plugins/grpc/utility/example_test.go:27](../../../plugins/grpc/utility/example_test.go#L27)     | `ExampleExtractMetadata` | 单测     |
| [plugins/grpc/utility/example_test.go:31](../../../plugins/grpc/utility/example_test.go#L31)     | `ExampleExtractMetadata` | 单测     |
| [plugins/grpc/utility/grpc_util_test.go:84](../../../plugins/grpc/utility/grpc_util_test.go#L84) | `TestExtractMetadata`    | 单测     |

#### P0457

- 定义：[plugins/web-gin/bind/binder.go:15](../../../plugins/web-gin/bind/binder.go#L15) `ErrNoBinderInContext`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                                                 | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/bind/binding.go:213](../../../plugins/web-gin/bind/binding.go#L213) | `ensureReady` | SDK生产  |

#### P0458

- 定义：[plugins/web-gin/bind/binder.go:16](../../../plugins/web-gin/bind/binder.go#L16) `ErrNilEngine`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                                             | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/web-gin/bind/binder.go:33](../../../plugins/web-gin/bind/binder.go#L33) | `New`         | SDK生产  |

#### P0459

- 定义：[plugins/web-gin/bind/binder.go:17](../../../plugins/web-gin/bind/binder.go#L17) `ErrNilTarget`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                                                 | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/bind/binding.go:209](../../../plugins/web-gin/bind/binding.go#L209) | `ensureReady` | SDK生产  |

#### P0460

- 定义：[plugins/web-gin/bind/binder.go:31](../../../plugins/web-gin/bind/binder.go#L31) `New`；分类S；阶段P6。
- 完整当前类型：`func(engine *github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator.Engine) (*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind.Binder, error)`。

| 位置                                                                                           | 所在函数/声明                       | 引用来源 |
| ---------------------------------------------------------------------------------------------- | ----------------------------------- | -------- |
| [plugins/web-gin/bind/binder_test.go:67](../../../plugins/web-gin/bind/binder_test.go#L67)     | `TestBindWithNoGlobalValidatorLeak` | 单测     |
| [plugins/web-gin/bind/example_test.go:36](../../../plugins/web-gin/bind/example_test.go#L36)   | `ExampleBind`                       | 单测     |
| [plugins/web-gin/bind/example_test.go:66](../../../plugins/web-gin/bind/example_test.go#L66)   | `ExampleBind_query`                 | 单测     |
| [plugins/web-gin/bind/key_case_test.go:55](../../../plugins/web-gin/bind/key_case_test.go#L55) | `initTestBinder`                    | 单测     |
| [plugins/web-gin/plugin.go:405](../../../plugins/web-gin/plugin.go#L405)                       | `Start`                             | SDK生产  |

#### P0461

- 定义：[plugins/web-gin/bind/binding.go:219](../../../plugins/web-gin/bind/binding.go#L219) `Binding.handleBindError`；分类N；阶段P6。
- 完整当前类型：`func(err error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind.Binding[T any]`。

| 位置                                                                                 | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/bind/binding.go:41](../../../plugins/web-gin/bind/binding.go#L41)   | `Data`        | SDK生产  |
| [plugins/web-gin/bind/binding.go:46](../../../plugins/web-gin/bind/binding.go#L46)   | `Data`        | SDK生产  |
| [plugins/web-gin/bind/binding.go:51](../../../plugins/web-gin/bind/binding.go#L51)   | `Data`        | SDK生产  |
| [plugins/web-gin/bind/binding.go:75](../../../plugins/web-gin/bind/binding.go#L75)   | `XML`         | SDK生产  |
| [plugins/web-gin/bind/binding.go:80](../../../plugins/web-gin/bind/binding.go#L80)   | `XML`         | SDK生产  |
| [plugins/web-gin/bind/binding.go:105](../../../plugins/web-gin/bind/binding.go#L105) | `Query`       | SDK生产  |
| [plugins/web-gin/bind/binding.go:134](../../../plugins/web-gin/bind/binding.go#L134) | `URI`         | SDK生产  |
| [plugins/web-gin/bind/binding.go:158](../../../plugins/web-gin/bind/binding.go#L158) | `Header`      | SDK生产  |
| [plugins/web-gin/bind/binding.go:181](../../../plugins/web-gin/bind/binding.go#L181) | `Form`        | SDK生产  |
| [plugins/web-gin/bind/binding.go:191](../../../plugins/web-gin/bind/binding.go#L191) | `Form`        | SDK生产  |
| [plugins/web-gin/bind/binding.go:209](../../../plugins/web-gin/bind/binding.go#L209) | `ensureReady` | SDK生产  |
| [plugins/web-gin/bind/binding.go:213](../../../plugins/web-gin/bind/binding.go#L213) | `ensureReady` | SDK生产  |

#### P0462

- 定义：[plugins/web-gin/bind/binding.go:233](../../../plugins/web-gin/bind/binding.go#L233) `Binding.readBodyBytes`；分类S；阶段P6。
- 完整当前类型：`func() ([]byte, error)`<br>泛型属主：`github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind.Binding[T any]`。

| 位置                                                                               | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/web-gin/bind/binding.go:39](../../../plugins/web-gin/bind/binding.go#L39) | `Data`        | SDK生产  |
| [plugins/web-gin/bind/binding.go:73](../../../plugins/web-gin/bind/binding.go#L73) | `XML`         | SDK生产  |

#### P0463

- 定义：[plugins/web-gin/bind/binding.go:246](../../../plugins/web-gin/bind/binding.go#L246) `mapValuesToStruct`；分类S；阶段P6。
- 完整当前类型：`func(values map[string][]string, target any, headerMode bool, tagNames ...string) error`。

| 位置                                                                                 | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/bind/binding.go:104](../../../plugins/web-gin/bind/binding.go#L104) | `Query`       | SDK生产  |
| [plugins/web-gin/bind/binding.go:133](../../../plugins/web-gin/bind/binding.go#L133) | `URI`         | SDK生产  |
| [plugins/web-gin/bind/binding.go:157](../../../plugins/web-gin/bind/binding.go#L157) | `Header`      | SDK生产  |
| [plugins/web-gin/bind/binding.go:190](../../../plugins/web-gin/bind/binding.go#L190) | `Form`        | SDK生产  |

#### P0464

- 定义：[plugins/web-gin/bind/binding.go:258](../../../plugins/web-gin/bind/binding.go#L258) `fillStructFields`；分类S；阶段P6。
- 完整当前类型：`func(values map[string][]string, structVal reflect.Value, headerMode bool, tagNames ...string) error`。

| 位置                                                                                 | 所在函数/声明       | 引用来源 |
| ------------------------------------------------------------------------------------ | ------------------- | -------- |
| [plugins/web-gin/bind/binding.go:255](../../../plugins/web-gin/bind/binding.go#L255) | `mapValuesToStruct` | SDK生产  |
| [plugins/web-gin/bind/binding.go:270](../../../plugins/web-gin/bind/binding.go#L270) | `fillStructFields`  | SDK生产  |

#### P0465

- 定义：[plugins/web-gin/bind/binding.go:335](../../../plugins/web-gin/bind/binding.go#L335) `setFieldValue`；分类S；阶段P6。
- 完整当前类型：`func(fieldVal reflect.Value, rawVals []string, timeFormat string, loc *time.Location) error`。

| 位置                                                                                 | 所在函数/声明      | 引用来源 |
| ------------------------------------------------------------------------------------ | ------------------ | -------- |
| [plugins/web-gin/bind/binding.go:293](../../../plugins/web-gin/bind/binding.go#L293) | `fillStructFields` | SDK生产  |

#### P0466

- 定义：[plugins/web-gin/bind/binding.go:383](../../../plugins/web-gin/bind/binding.go#L383) `setSingleScalar`；分类S；阶段P6。
- 完整当前类型：`func(v reflect.Value, raw string, timeFormat string, loc *time.Location) error`。

| 位置                                                                                 | 所在函数/声明   | 引用来源 |
| ------------------------------------------------------------------------------------ | --------------- | -------- |
| [plugins/web-gin/bind/binding.go:366](../../../plugins/web-gin/bind/binding.go#L366) | `setFieldValue` | SDK生产  |
| [plugins/web-gin/bind/binding.go:379](../../../plugins/web-gin/bind/binding.go#L379) | `setFieldValue` | SDK生产  |

#### P0467

- 定义：[plugins/web-gin/bind/binding.go:426](../../../plugins/web-gin/bind/binding.go#L426) `parseTimeValue`；分类S；阶段P6。
- 完整当前类型：`func(raw string, format string, loc *time.Location) (time.Time, error)`。

| 位置                                                                                 | 所在函数/声明     | 引用来源 |
| ------------------------------------------------------------------------------------ | ----------------- | -------- |
| [plugins/web-gin/bind/binding.go:346](../../../plugins/web-gin/bind/binding.go#L346) | `setFieldValue`   | SDK生产  |
| [plugins/web-gin/bind/binding.go:385](../../../plugins/web-gin/bind/binding.go#L385) | `setSingleScalar` | SDK生产  |

#### P0468

- 定义：[plugins/web-gin/header/util.go:12](../../../plugins/web-gin/header/util.go#L12) `ErrMissingAuthorization`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                                             | 所在函数/声明      | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------ | -------- |
| [plugins/web-gin/header/util.go:24](../../../plugins/web-gin/header/util.go#L24) | `GetAuthorization` | SDK生产  |
| [plugins/web-gin/header/util.go:28](../../../plugins/web-gin/header/util.go#L28) | `GetAuthorization` | SDK生产  |

#### P0469

- 定义：[plugins/web-gin/header/util.go:13](../../../plugins/web-gin/header/util.go#L13) `ErrInvalidAuthorizationFormat`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                                             | 所在函数/声明      | 引用来源 |
| -------------------------------------------------------------------------------- | ------------------ | -------- |
| [plugins/web-gin/header/util.go:33](../../../plugins/web-gin/header/util.go#L33) | `GetAuthorization` | SDK生产  |

#### P0470

- 定义：[plugins/web-gin/header/util.go:22](../../../plugins/web-gin/header/util.go#L22) `GetAuthorization`；分类S；阶段P6。
- 完整当前类型：`func(ctx *github.com/gin-gonic/gin.Context) (string, error)`。

| 位置                                                                                             | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------------------------------------------ | ------------------------- | -------- |
| [plugins/web-gin/header/example_test.go:24](../../../plugins/web-gin/header/example_test.go#L24) | `ExampleGetAuthorization` | 单测     |
| [plugins/web-gin/header/header_test.go:39](../../../plugins/web-gin/header/header_test.go#L39)   | `TestGetAuthorization`    | 单测     |
| [plugins/web-gin/header/header_test.go:45](../../../plugins/web-gin/header/header_test.go#L45)   | `TestGetAuthorization`    | 单测     |
| [plugins/web-gin/header/header_test.go:51](../../../plugins/web-gin/header/header_test.go#L51)   | `TestGetAuthorization`    | 单测     |
| [plugins/web-gin/header/header_test.go:57](../../../plugins/web-gin/header/header_test.go#L57)   | `TestGetAuthorization`    | 单测     |

#### P0471

- 定义：[plugins/web-gin/httpresult/result.go:53](../../../plugins/web-gin/httpresult/result.go#L53) `buildEnvelope`；分类Z；阶段P6。
- 完整当前类型：`func(ctx *github.com/gin-gonic/gin.Context, err *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error) (int, github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope.Response)`。

| 位置                                                                                           | 所在函数/声明 | 引用来源 |
| ---------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/web-gin/httpresult/result.go:133](../../../plugins/web-gin/httpresult/result.go#L133) | `Error`       | SDK生产  |
| [plugins/web-gin/httpresult/result.go:148](../../../plugins/web-gin/httpresult/result.go#L148) | `AbortError`  | SDK生产  |

#### P0472

- 定义：[plugins/web-gin/httpresult/result.go:129](../../../plugins/web-gin/httpresult/result.go#L129) `Error`；分类Z；阶段P6。
- 完整当前类型：`func(ctx *github.com/gin-gonic/gin.Context, err *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error)`。

| 位置                                                                                                           | 所在函数/声明                              | 引用来源 |
| -------------------------------------------------------------------------------------------------------------- | ------------------------------------------ | -------- |
| [plugins/web-gin/httpresult/error_code_test.go:69](../../../plugins/web-gin/httpresult/error_code_test.go#L69) | `TestProtocolErrorCodes`                   | 单测     |
| [plugins/web-gin/httpresult/example_test.go:60](../../../plugins/web-gin/httpresult/example_test.go#L60)       | `ExampleError`                             | 单测     |
| [plugins/web-gin/httpresult/status_test.go:130](../../../plugins/web-gin/httpresult/status_test.go#L130)       | `TestError_FallbackAndEnvelope`            | 单测     |
| [plugins/web-gin/httpresult/status_test.go:161](../../../plugins/web-gin/httpresult/status_test.go#L161)       | `TestError_NoUnderlyingErrLeakEvenInDebug` | 单测     |
| [plugins/web-gin/middleware/middleware.go:75](../../../plugins/web-gin/middleware/middleware.go#L75)           | `Response`                                 | SDK生产  |
| [plugins/web-gin/middleware/middleware.go:77](../../../plugins/web-gin/middleware/middleware.go#L77)           | `Response`                                 | SDK生产  |
| [plugins/web-gin/middleware/middleware.go:86](../../../plugins/web-gin/middleware/middleware.go#L86)           | `Response`                                 | SDK生产  |
| [plugins/web-gin/validator/error.go:68](../../../plugins/web-gin/validator/error.go#L68)                       | `handleErrorInternal`                      | SDK生产  |

#### P0473

- 定义：[plugins/web-gin/httpresult/result.go:141](../../../plugins/web-gin/httpresult/result.go#L141) `AbortError`；分类Z；阶段P6。
- 完整当前类型：`func(ctx *github.com/gin-gonic/gin.Context, err *github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr.Error)`。

| 位置                                                                                                                   | 所在函数/声明                            | 引用来源   |
| ---------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- | ---------- |
| [examples/v2/http-db-cache/main.go:71](../../../examples/v2/http-db-cache/main.go#L71)                                 | `main`                                   | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:77](../../../examples/v2/http-db-cache/main.go#L77)                                 | `main`                                   | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:84](../../../examples/v2/http-db-cache/main.go#L84)                                 | `main`                                   | 仓内消费者 |
| [examples/v2/http-db-cache/main.go:90](../../../examples/v2/http-db-cache/main.go#L90)                                 | `main`                                   | 仓内消费者 |
| [plugins/web-gin/bind/binding.go:230](../../../plugins/web-gin/bind/binding.go#L230)                                   | `handleBindError`                        | SDK生产    |
| [plugins/web-gin/httpresult/status_test.go:190](../../../plugins/web-gin/httpresult/status_test.go#L190)               | `TestAbortError_StopsSubsequentHandlers` | 单测       |
| [plugins/web-gin/middleware/middleware.go:52](../../../plugins/web-gin/middleware/middleware.go#L52)                   | `Recovery`                               | SDK生产    |
| [plugins/web-gin/plugin.go:596](../../../plugins/web-gin/plugin.go#L596)                                               | `requestBridgeMiddleware`                | SDK生产    |
| [plugins/web-gin/plugin.go:607](../../../plugins/web-gin/plugin.go#L607)                                               | `requestBridgeMiddleware`                | SDK生产    |
| [plugins/web-gin/plugin_isolation_test.go:131](../../../plugins/web-gin/plugin_isolation_test.go#L131)                 | `buildApp`                               | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:144](../../../plugins/web-gin/plugin_isolation_test.go#L144)                 | `buildApp`                               | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:150](../../../plugins/web-gin/plugin_isolation_test.go#L150)                 | `buildApp`                               | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:156](../../../plugins/web-gin/plugin_isolation_test.go#L156)                 | `buildApp`                               | 单测       |
| [plugins/web-gin/plugin_isolation_test.go:160](../../../plugins/web-gin/plugin_isolation_test.go#L160)                 | `buildApp`                               | 单测       |
| [plugins/web-gin/plugin_test.go:208](../../../plugins/web-gin/plugin_test.go#L208)                                     | `TestWebGin_FullNetworkLifecycle`        | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:104](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L104) | `TestHTTPDBCacheLifecycle_EndToEnd`      | 仓内消费者 |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:110](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L110) | `TestHTTPDBCacheLifecycle_EndToEnd`      | 仓内消费者 |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:117](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L117) | `TestHTTPDBCacheLifecycle_EndToEnd`      | 仓内消费者 |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:123](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L123) | `TestHTTPDBCacheLifecycle_EndToEnd`      | 仓内消费者 |

#### P0474

- 定义：[plugins/web-gin/plugin.go:31](../../../plugins/web-gin/plugin.go#L31) `ErrServerAlreadyStarted`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/plugin.go:362](../../../plugins/web-gin/plugin.go#L362) | `Start`       | SDK生产  |

#### P0475

- 定义：[plugins/web-gin/plugin.go:32](../../../plugins/web-gin/plugin.go#L32) `ErrServerClosed`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------------------ | ------------------------- | -------- |
| [plugins/web-gin/plugin.go:596](../../../plugins/web-gin/plugin.go#L596) | `requestBridgeMiddleware` | SDK生产  |

#### P0476

- 定义：[plugins/web-gin/plugin.go:33](../../../plugins/web-gin/plugin.go#L33) `ErrNoLifetimeContext`；分类N；阶段P6。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明             | 引用来源 |
| ------------------------------------------------------------------------ | ------------------------- | -------- |
| [plugins/web-gin/plugin.go:607](../../../plugins/web-gin/plugin.go#L607) | `requestBridgeMiddleware` | SDK生产  |

#### P0477

- 定义：[plugins/web-gin/plugin.go:34](../../../plugins/web-gin/plugin.go#L34) `ErrStopTimeout`；分类N；阶段P6。
- 完整当前类型：`error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0478

- 定义：[plugins/web-gin/plugin.go:55](../../../plugins/web-gin/plugin.go#L55) `config.validatorRegistrars`；分类S；阶段P6。
- 完整当前类型：`[]func(*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator.Engine) error`。

| 位置                                                                     | 所在函数/声明            | 引用来源 |
| ------------------------------------------------------------------------ | ------------------------ | -------- |
| [plugins/web-gin/plugin.go:128](../../../plugins/web-gin/plugin.go#L128) | `WithValidatorRegistrar` | SDK生产  |
| [plugins/web-gin/plugin.go:128](../../../plugins/web-gin/plugin.go#L128) | `WithValidatorRegistrar` | SDK生产  |
| [plugins/web-gin/plugin.go:283](../../../plugins/web-gin/plugin.go#L283) | `New`                    | SDK生产  |
| [plugins/web-gin/plugin.go:284](../../../plugins/web-gin/plugin.go#L284) | `New`                    | SDK生产  |
| [plugins/web-gin/plugin.go:288](../../../plugins/web-gin/plugin.go#L288) | `New`                    | SDK生产  |
| [plugins/web-gin/plugin.go:393](../../../plugins/web-gin/plugin.go#L393) | `Start`                  | SDK生产  |

#### P0479

- 定义：[plugins/web-gin/plugin.go:125](../../../plugins/web-gin/plugin.go#L125) `WithValidatorRegistrar`；分类S；阶段P6。
- 完整当前类型：`func(fn func(*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator.Engine) error) github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2.Option`。

| 位置                                                                                                   | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/plugin_isolation_test.go:177](../../../plugins/web-gin/plugin_isolation_test.go#L177) | `buildApp`    | 单测     |

#### P0480

- 定义：[plugins/web-gin/plugin.go:200](../../../plugins/web-gin/plugin.go#L200) `requestGate.waitDrained`；分类S；阶段P6。
- 完整当前类型：`func(ctx context.Context) error`。

| 位置                                                                     | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/plugin.go:577](../../../plugins/web-gin/plugin.go#L577) | `doStop`      | SDK生产  |

#### P0481

- 定义：[plugins/web-gin/plugin.go:252](../../../plugins/web-gin/plugin.go#L252) `Plugin.stopErr`；分类S；阶段P6。
- 完整当前类型：`error`。

| 位置                                                                     | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/plugin.go:518](../../../plugins/web-gin/plugin.go#L518) | `Stop`        | SDK生产  |
| [plugins/web-gin/plugin.go:522](../../../plugins/web-gin/plugin.go#L522) | `Stop`        | SDK生产  |

#### P0482

- 定义：[plugins/web-gin/plugin.go:283](../../../plugins/web-gin/plugin.go#L283) `New.validatorRegistrarsCopy`；分类S；阶段P6。
- 完整当前类型：`[]func(*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator.Engine) error`。

| 位置                                                                     | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/plugin.go:284](../../../plugins/web-gin/plugin.go#L284) | `New`         | SDK生产  |
| [plugins/web-gin/plugin.go:288](../../../plugins/web-gin/plugin.go#L288) | `New`         | SDK生产  |

#### P0483

- 定义：[plugins/web-gin/plugin.go:358](../../../plugins/web-gin/plugin.go#L358) `Plugin.Start`；分类S；阶段P3/P6。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`；`github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`。

| 位置                                                                                                                 | 所在函数/声明                          | 引用来源   |
| -------------------------------------------------------------------------------------------------------------------- | -------------------------------------- | ---------- |
| [plugins/web-gin/address_test.go:45](../../../plugins/web-gin/address_test.go#L45)                                   | `TestResolveAddr_StartListenRandom`    | 单测       |
| [plugins/web-gin/address_test.go:71](../../../plugins/web-gin/address_test.go#L71)                                   | `TestResolveAddr_InvalidPortError`     | 单测       |
| [plugins/web-gin/plugin_test.go:507](../../../plugins/web-gin/plugin_test.go#L507)                                   | `TestWebGin_ForceContextCancellation`  | 单测       |
| [plugins/web-gin/plugin_test.go:575](../../../plugins/web-gin/plugin_test.go#L575)                                   | `TestWebGin_MissingLifetimeCtxRejects` | 单测       |
| [test/consumer/v2-http-db-cache/lifecycle_test.go:49](../../../test/consumer/v2-http-db-cache/lifecycle_test.go#L49) | `Start`                                | 仓内消费者 |

#### P0484

- 定义：[plugins/web-gin/plugin.go:474](../../../plugins/web-gin/plugin.go#L474) `Plugin.Run`；分类S；阶段P3/P6。
- 完整当前类型：`func(lifetimeCtx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`。

| 位置                                                                               | 所在函数/声明                          | 引用来源 |
| ---------------------------------------------------------------------------------- | -------------------------------------- | -------- |
| [plugins/web-gin/plugin_test.go:532](../../../plugins/web-gin/plugin_test.go#L532) | `TestWebGin_ForceContextCancellation`  | 单测     |
| [plugins/web-gin/plugin_test.go:585](../../../plugins/web-gin/plugin_test.go#L585) | `TestWebGin_MissingLifetimeCtxRejects` | 单测     |

#### P0485

- 定义：[plugins/web-gin/plugin.go:500](../../../plugins/web-gin/plugin.go#L500) `Plugin.Quiesce`；分类S；阶段P3/P6。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Quiescer`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0486

- 定义：[plugins/web-gin/plugin.go:516](../../../plugins/web-gin/plugin.go#L516) `Plugin.Stop`；分类S；阶段P3/P6。
- 完整当前类型：`func(ctx context.Context) error`。
- 方法集：`github.com/bamboo-services/bamboo-base-go/application/v2.Plugin`；`github.com/bamboo-services/bamboo-base-go/application/v2.RunnablePlugin`。

| 位置                                                                               | 所在函数/声明                             | 引用来源 |
| ---------------------------------------------------------------------------------- | ----------------------------------------- | -------- |
| [plugins/web-gin/address_test.go:48](../../../plugins/web-gin/address_test.go#L48) | `TestResolveAddr_StartListenRandom`       | 单测     |
| [plugins/web-gin/address_test.go:73](../../../plugins/web-gin/address_test.go#L73) | `TestResolveAddr_InvalidPortError`        | 单测     |
| [plugins/web-gin/plugin_test.go:178](../../../plugins/web-gin/plugin_test.go#L178) | `TestWebGin_ContainerBridgeAndRequestKey` | 单测     |
| [plugins/web-gin/plugin_test.go:426](../../../plugins/web-gin/plugin_test.go#L426) | `TestWebGin_FullNetworkLifecycle`         | 单测     |
| [plugins/web-gin/plugin_test.go:471](../../../plugins/web-gin/plugin_test.go#L471) | `TestWebGin_FullNetworkLifecycle`         | 单测     |
| [plugins/web-gin/plugin_test.go:510](../../../plugins/web-gin/plugin_test.go#L510) | `TestWebGin_ForceContextCancellation`     | 单测     |
| [plugins/web-gin/plugin_test.go:549](../../../plugins/web-gin/plugin_test.go#L549) | `TestWebGin_ForceContextCancellation`     | 单测     |
| [plugins/web-gin/plugin_test.go:578](../../../plugins/web-gin/plugin_test.go#L578) | `TestWebGin_MissingLifetimeCtxRejects`    | 单测     |

#### P0487

- 定义：[plugins/web-gin/plugin.go:525](../../../plugins/web-gin/plugin.go#L525) `Plugin.doStop`；分类S；阶段P6。
- 完整当前类型：`func(ctx context.Context) error`。

| 位置                                                                     | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/plugin.go:518](../../../plugins/web-gin/plugin.go#L518) | `Stop`        | SDK生产  |

#### P0488

- 定义：[plugins/web-gin/plugin.go:538](../../../plugins/web-gin/plugin.go#L538) `doStop.shutdownDone`；分类S；阶段P6。
- 完整当前类型：`chan error`。

| 位置                                                                     | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/plugin.go:541](../../../plugins/web-gin/plugin.go#L541) | `doStop`      | SDK生产  |
| [plugins/web-gin/plugin.go:548](../../../plugins/web-gin/plugin.go#L548) | `doStop`      | SDK生产  |
| [plugins/web-gin/plugin.go:556](../../../plugins/web-gin/plugin.go#L556) | `doStop`      | SDK生产  |
| [plugins/web-gin/plugin.go:566](../../../plugins/web-gin/plugin.go#L566) | `doStop`      | SDK生产  |

#### P0489

- 定义：[plugins/web-gin/validator/custom.go:186](../../../plugins/web-gin/validator/custom.go#L186) `Engine.registerBuiltinCustomRules`；分类S；阶段P6。
- 完整当前类型：`func() error`。

| 位置                                                                                       | 所在函数/声明 | 引用来源 |
| ------------------------------------------------------------------------------------------ | ------------- | -------- |
| [plugins/web-gin/validator/engine.go:62](../../../plugins/web-gin/validator/engine.go#L62) | `New`         | SDK生产  |

#### P0490

- 定义：[plugins/web-gin/validator/custom.go:212](../../../plugins/web-gin/validator/custom.go#L212) `registerBuiltinTranslations::<anonymous@212>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0491

- 定义：[plugins/web-gin/validator/custom.go:215](../../../plugins/web-gin/validator/custom.go#L215) `registerBuiltinTranslations::<anonymous@215>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator, fe github.com/go-playground/validator/v10.FieldError) string`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0492

- 定义：[plugins/web-gin/validator/custom.go:223](../../../plugins/web-gin/validator/custom.go#L223) `registerBuiltinTranslations::<anonymous@223>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0493

- 定义：[plugins/web-gin/validator/custom.go:226](../../../plugins/web-gin/validator/custom.go#L226) `registerBuiltinTranslations::<anonymous@226>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator, fe github.com/go-playground/validator/v10.FieldError) string`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0494

- 定义：[plugins/web-gin/validator/custom.go:234](../../../plugins/web-gin/validator/custom.go#L234) `registerBuiltinTranslations::<anonymous@234>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0495

- 定义：[plugins/web-gin/validator/custom.go:237](../../../plugins/web-gin/validator/custom.go#L237) `registerBuiltinTranslations::<anonymous@237>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator, fe github.com/go-playground/validator/v10.FieldError) string`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0496

- 定义：[plugins/web-gin/validator/custom.go:245](../../../plugins/web-gin/validator/custom.go#L245) `registerBuiltinTranslations::<anonymous@245>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0497

- 定义：[plugins/web-gin/validator/custom.go:248](../../../plugins/web-gin/validator/custom.go#L248) `registerBuiltinTranslations::<anonymous@248>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator, fe github.com/go-playground/validator/v10.FieldError) string`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0498

- 定义：[plugins/web-gin/validator/custom.go:256](../../../plugins/web-gin/validator/custom.go#L256) `registerBuiltinTranslations::<anonymous@256>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0499

- 定义：[plugins/web-gin/validator/custom.go:259](../../../plugins/web-gin/validator/custom.go#L259) `registerBuiltinTranslations::<anonymous@259>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator, fe github.com/go-playground/validator/v10.FieldError) string`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0500

- 定义：[plugins/web-gin/validator/custom.go:267](../../../plugins/web-gin/validator/custom.go#L267) `registerBuiltinTranslations::<anonymous@267>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0501

- 定义：[plugins/web-gin/validator/custom.go:270](../../../plugins/web-gin/validator/custom.go#L270) `registerBuiltinTranslations::<anonymous@270>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator, fe github.com/go-playground/validator/v10.FieldError) string`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0502

- 定义：[plugins/web-gin/validator/custom.go:278](../../../plugins/web-gin/validator/custom.go#L278) `registerBuiltinTranslations::<anonymous@278>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0503

- 定义：[plugins/web-gin/validator/custom.go:281](../../../plugins/web-gin/validator/custom.go#L281) `registerBuiltinTranslations::<anonymous@281>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator, fe github.com/go-playground/validator/v10.FieldError) string`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0504

- 定义：[plugins/web-gin/validator/custom.go:289](../../../plugins/web-gin/validator/custom.go#L289) `registerBuiltinTranslations::<anonymous@289>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0505

- 定义：[plugins/web-gin/validator/custom.go:292](../../../plugins/web-gin/validator/custom.go#L292) `registerBuiltinTranslations::<anonymous@292>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator, fe github.com/go-playground/validator/v10.FieldError) string`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0506

- 定义：[plugins/web-gin/validator/custom.go:300](../../../plugins/web-gin/validator/custom.go#L300) `registerBuiltinTranslations::<anonymous@300>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator) error`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0507

- 定义：[plugins/web-gin/validator/custom.go:303](../../../plugins/web-gin/validator/custom.go#L303) `registerBuiltinTranslations::<anonymous@303>`；分类F；阶段P6。
- 完整当前类型：`func(ut github.com/go-playground/universal-translator.Translator, fe github.com/go-playground/validator/v10.FieldError) string`。
- 精确引用：0；不等于无调用，接口候选/匿名注册位置与仓外使用仍需分别核对。

#### P0508

- 定义：[plugins/web-gin/validator/engine.go:29](../../../plugins/web-gin/validator/engine.go#L29) `New`；分类S；阶段P6。
- 完整当前类型：`func() (*github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator.Engine, error)`。

| 位置                                                                                                                 | 所在函数/声明                                  | 引用来源 |
| -------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- | -------- |
| [plugins/web-gin/bind/binder_test.go:62](../../../plugins/web-gin/bind/binder_test.go#L62)                           | `TestBindWithNoGlobalValidatorLeak`            | 单测     |
| [plugins/web-gin/bind/example_test.go:32](../../../plugins/web-gin/bind/example_test.go#L32)                         | `ExampleBind`                                  | 单测     |
| [plugins/web-gin/bind/example_test.go:62](../../../plugins/web-gin/bind/example_test.go#L62)                         | `ExampleBind_query`                            | 单测     |
| [plugins/web-gin/bind/key_case_test.go:51](../../../plugins/web-gin/bind/key_case_test.go#L51)                       | `initTestBinder`                               | 单测     |
| [plugins/web-gin/plugin.go:384](../../../plugins/web-gin/plugin.go#L384)                                             | `Start`                                        | SDK生产  |
| [plugins/web-gin/validator/error_security_test.go:27](../../../plugins/web-gin/validator/error_security_test.go#L27) | `TestSecurity_ValidationValueNotLeaked`        | 单测     |
| [plugins/web-gin/validator/error_security_test.go:73](../../../plugins/web-gin/validator/error_security_test.go#L73) | `TestSecurity_UnderlyingNetworkErrorNotLeaked` | 单测     |
| [plugins/web-gin/validator/example_test.go:18](../../../plugins/web-gin/validator/example_test.go#L18)               | `ExampleEngine_Validate`                       | 单测     |
| [plugins/web-gin/validator/example_test.go:44](../../../plugins/web-gin/validator/example_test.go#L44)               | `ExampleEngine_TranslateError`                 | 单测     |
| [plugins/web-gin/validator/validator_test.go:24](../../../plugins/web-gin/validator/validator_test.go#L24)           | `TestDualInstanceOppositeRulesIsolation`       | 单测     |
| [plugins/web-gin/validator/validator_test.go:28](../../../plugins/web-gin/validator/validator_test.go#L28)           | `TestDualInstanceOppositeRulesIsolation`       | 单测     |
| [plugins/web-gin/validator/validator_test.go:74](../../../plugins/web-gin/validator/validator_test.go#L74)           | `TestBuiltinRulesAndChineseTranslations`       | 单测     |
| [plugins/web-gin/validator/validator_test.go:153](../../../plugins/web-gin/validator/validator_test.go#L153)         | `TestHandleErrorSafety`                        | 单测     |
| [plugins/web-gin/validator/validator_test.go:209](../../../plugins/web-gin/validator/validator_test.go#L209)         | `TestConcurrentValidateRace`                   | 单测     |

#### P0509

- 定义：[plugins/web-gin/validator/engine.go:74](../../../plugins/web-gin/validator/engine.go#L74) `Engine.Validate`；分类S；阶段P6。
- 完整当前类型：`func(obj any) error`。

| 位置                                                                                                                 | 所在函数/声明                            | 引用来源 |
| -------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [plugins/web-gin/bind/binding.go:55](../../../plugins/web-gin/bind/binding.go#L55)                                   | `Data`                                   | SDK生产  |
| [plugins/web-gin/bind/binding.go:84](../../../plugins/web-gin/bind/binding.go#L84)                                   | `XML`                                    | SDK生产  |
| [plugins/web-gin/bind/binding.go:109](../../../plugins/web-gin/bind/binding.go#L109)                                 | `Query`                                  | SDK生产  |
| [plugins/web-gin/bind/binding.go:138](../../../plugins/web-gin/bind/binding.go#L138)                                 | `URI`                                    | SDK生产  |
| [plugins/web-gin/bind/binding.go:162](../../../plugins/web-gin/bind/binding.go#L162)                                 | `Header`                                 | SDK生产  |
| [plugins/web-gin/bind/binding.go:195](../../../plugins/web-gin/bind/binding.go#L195)                                 | `Form`                                   | SDK生产  |
| [plugins/web-gin/validator/error_security_test.go:39](../../../plugins/web-gin/validator/error_security_test.go#L39) | `TestSecurity_ValidationValueNotLeaked`  | 单测     |
| [plugins/web-gin/validator/example_test.go:28](../../../plugins/web-gin/validator/example_test.go#L28)               | `ExampleEngine_Validate`                 | 单测     |
| [plugins/web-gin/validator/example_test.go:35](../../../plugins/web-gin/validator/example_test.go#L35)               | `ExampleEngine_Validate`                 | 单测     |
| [plugins/web-gin/validator/example_test.go:54](../../../plugins/web-gin/validator/example_test.go#L54)               | `ExampleEngine_TranslateError`           | 单测     |
| [plugins/web-gin/validator/validator_test.go:56](../../../plugins/web-gin/validator/validator_test.go#L56)           | `TestDualInstanceOppositeRulesIsolation` | 单测     |
| [plugins/web-gin/validator/validator_test.go:59](../../../plugins/web-gin/validator/validator_test.go#L59)           | `TestDualInstanceOppositeRulesIsolation` | 单测     |
| [plugins/web-gin/validator/validator_test.go:64](../../../plugins/web-gin/validator/validator_test.go#L64)           | `TestDualInstanceOppositeRulesIsolation` | 单测     |
| [plugins/web-gin/validator/validator_test.go:67](../../../plugins/web-gin/validator/validator_test.go#L67)           | `TestDualInstanceOppositeRulesIsolation` | 单测     |
| [plugins/web-gin/validator/validator_test.go:101](../../../plugins/web-gin/validator/validator_test.go#L101)         | `TestBuiltinRulesAndChineseTranslations` | 单测     |
| [plugins/web-gin/validator/validator_test.go:117](../../../plugins/web-gin/validator/validator_test.go#L117)         | `TestBuiltinRulesAndChineseTranslations` | 单测     |
| [plugins/web-gin/validator/validator_test.go:162](../../../plugins/web-gin/validator/validator_test.go#L162)         | `TestHandleErrorSafety`                  | 单测     |
| [plugins/web-gin/validator/validator_test.go:237](../../../plugins/web-gin/validator/validator_test.go#L237)         | `TestConcurrentValidateRace`             | 单测     |
| [plugins/web-gin/validator/validator_test.go:247](../../../plugins/web-gin/validator/validator_test.go#L247)         | `TestConcurrentValidateRace`             | 单测     |

#### P0510

- 定义：[plugins/web-gin/validator/engine.go:86](../../../plugins/web-gin/validator/engine.go#L86) `Engine.RegisterValidation`；分类S；阶段P6。
- 完整当前类型：`func(tag string, fn github.com/go-playground/validator/v10.Func) error`。

| 位置                                                                                                       | 所在函数/声明                            | 引用来源 |
| ---------------------------------------------------------------------------------------------------------- | ---------------------------------------- | -------- |
| [plugins/web-gin/plugin_isolation_test.go:169](../../../plugins/web-gin/plugin_isolation_test.go#L169)     | `buildApp`                               | 单测     |
| [plugins/web-gin/validator/validator_test.go:34](../../../plugins/web-gin/validator/validator_test.go#L34) | `TestDualInstanceOppositeRulesIsolation` | 单测     |
| [plugins/web-gin/validator/validator_test.go:41](../../../plugins/web-gin/validator/validator_test.go#L41) | `TestDualInstanceOppositeRulesIsolation` | 单测     |

#### P0511

- 定义：[plugins/web-gin/validator/engine.go:96](../../../plugins/web-gin/validator/engine.go#L96) `Engine.TranslateError`；分类N；阶段P6。
- 完整当前类型：`func(err error) map[string]string`。

| 位置                                                                                                         | 所在函数/声明                            | 引用来源 |
| ------------------------------------------------------------------------------------------------------------ | ---------------------------------------- | -------- |
| [plugins/web-gin/validator/error.go:37](../../../plugins/web-gin/validator/error.go#L37)                     | `handleErrorInternal`                    | SDK生产  |
| [plugins/web-gin/validator/example_test.go:56](../../../plugins/web-gin/validator/example_test.go#L56)       | `ExampleEngine_TranslateError`           | 单测     |
| [plugins/web-gin/validator/validator_test.go:122](../../../plugins/web-gin/validator/validator_test.go#L122) | `TestBuiltinRulesAndChineseTranslations` | 单测     |
| [plugins/web-gin/validator/validator_test.go:251](../../../plugins/web-gin/validator/validator_test.go#L251) | `TestConcurrentValidateRace`             | 单测     |

#### P0512

- 定义：[plugins/web-gin/validator/engine.go:102](../../../plugins/web-gin/validator/engine.go#L102) `TranslateError.valErrs`；分类N；阶段P6。
- 完整当前类型：`github.com/go-playground/validator/v10.ValidationErrors`。

| 位置                                                                                         | 所在函数/声明    | 引用来源 |
| -------------------------------------------------------------------------------------------- | ---------------- | -------- |
| [plugins/web-gin/validator/engine.go:103](../../../plugins/web-gin/validator/engine.go#L103) | `TranslateError` | SDK生产  |
| [plugins/web-gin/validator/engine.go:107](../../../plugins/web-gin/validator/engine.go#L107) | `TranslateError` | SDK生产  |

#### P0513

- 定义：[plugins/web-gin/validator/engine.go:117](../../../plugins/web-gin/validator/engine.go#L117) `Engine.HandleError`；分类N；阶段P6。
- 完整当前类型：`func(ctx *github.com/gin-gonic/gin.Context, err error)`。

| 位置                                                                                                                 | 所在函数/声明                                  | 引用来源 |
| -------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- | -------- |
| [plugins/web-gin/bind/binding.go:56](../../../plugins/web-gin/bind/binding.go#L56)                                   | `Data`                                         | SDK生产  |
| [plugins/web-gin/bind/binding.go:85](../../../plugins/web-gin/bind/binding.go#L85)                                   | `XML`                                          | SDK生产  |
| [plugins/web-gin/bind/binding.go:110](../../../plugins/web-gin/bind/binding.go#L110)                                 | `Query`                                        | SDK生产  |
| [plugins/web-gin/bind/binding.go:139](../../../plugins/web-gin/bind/binding.go#L139)                                 | `URI`                                          | SDK生产  |
| [plugins/web-gin/bind/binding.go:163](../../../plugins/web-gin/bind/binding.go#L163)                                 | `Header`                                       | SDK生产  |
| [plugins/web-gin/bind/binding.go:196](../../../plugins/web-gin/bind/binding.go#L196)                                 | `Form`                                         | SDK生产  |
| [plugins/web-gin/bind/binding.go:224](../../../plugins/web-gin/bind/binding.go#L224)                                 | `handleBindError`                              | SDK生产  |
| [plugins/web-gin/validator/error_security_test.go:36](../../../plugins/web-gin/validator/error_security_test.go#L36) | `TestSecurity_ValidationValueNotLeaked`        | 单测     |
| [plugins/web-gin/validator/error_security_test.go:40](../../../plugins/web-gin/validator/error_security_test.go#L40) | `TestSecurity_ValidationValueNotLeaked`        | 单测     |
| [plugins/web-gin/validator/error_security_test.go:83](../../../plugins/web-gin/validator/error_security_test.go#L83) | `TestSecurity_UnderlyingNetworkErrorNotLeaked` | 单测     |
| [plugins/web-gin/validator/validator_test.go:172](../../../plugins/web-gin/validator/validator_test.go#L172)         | `TestHandleErrorSafety`                        | 单测     |
| [plugins/web-gin/validator/validator_test.go:201](../../../plugins/web-gin/validator/validator_test.go#L201)         | `TestHandleErrorSafety`                        | 单测     |

#### P0514

- 定义：[plugins/web-gin/validator/error.go:25](../../../plugins/web-gin/validator/error.go#L25) `Engine.handleErrorInternal`；分类N；阶段P6。
- 完整当前类型：`func(ctx *github.com/gin-gonic/gin.Context, bindErr error)`。

| 位置                                                                                         | 所在函数/声明 | 引用来源 |
| -------------------------------------------------------------------------------------------- | ------------- | -------- |
| [plugins/web-gin/validator/engine.go:118](../../../plugins/web-gin/validator/engine.go#L118) | `HandleError` | SDK生产  |

#### P0515

- 定义：[plugins/web-gin/validator/error.go:35](../../../plugins/web-gin/validator/error.go#L35) `handleErrorInternal.valErrs`；分类N；阶段P6。
- 完整当前类型：`github.com/go-playground/validator/v10.ValidationErrors`。

| 位置                                                                                     | 所在函数/声明         | 引用来源 |
| ---------------------------------------------------------------------------------------- | --------------------- | -------- |
| [plugins/web-gin/validator/error.go:36](../../../plugins/web-gin/validator/error.go#L36) | `handleErrorInternal` | SDK生产  |
| [plugins/web-gin/validator/error.go:38](../../../plugins/web-gin/validator/error.go#L38) | `handleErrorInternal` | SDK生产  |

#### P0516

- 定义：[plugins/web-gin/validator/error.go:71](../../../plugins/web-gin/validator/error.go#L71) `formatFallbackMessage`；分类N；阶段P6。
- 完整当前类型：`func(fe github.com/go-playground/validator/v10.FieldError) string`。

| 位置                                                                                     | 所在函数/声明         | 引用来源 |
| ---------------------------------------------------------------------------------------- | --------------------- | -------- |
| [plugins/web-gin/validator/error.go:41](../../../plugins/web-gin/validator/error.go#L41) | `handleErrorInternal` | SDK生产  |

#### P0517

- 定义：[plugins/web-gin/validator/error.go:89](../../../plugins/web-gin/validator/error.go#L89) `parseBindingErrorDetail`；分类N；阶段P6。
- 完整当前类型：`func(err error) github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator.ValidationErrorDetail`。

| 位置                                                                                     | 所在函数/声明         | 引用来源 |
| ---------------------------------------------------------------------------------------- | --------------------- | -------- |
| [plugins/web-gin/validator/error.go:54](../../../plugins/web-gin/validator/error.go#L54) | `handleErrorInternal` | SDK生产  |

## 结论

本台账供后续迁移逐点复核，不是CI守卫或运行覆盖率。精确引用和方法集关系已经区分；全局动态调用、反射与仓外消费者仍需实现阶段和人工补证。P0没有修改任何实现、签名、依赖或接口方法集，不能把这份现状表当成强类型契约已完成。

## 参考

- [P0分类、固定签名和消费者模板](./0005-foundation-strong-error-p0-inventory.md)
- [强类型规范spec0003](../spec/0003-foundation-strong-error-contract.md)
- [迁移计划plan0002](../../plan/0002-foundation-strong-error-migration.md)
