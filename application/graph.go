package application

import (
	"errors"
	"sort"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// 拓扑图构建与依赖校验错误哨兵。
//
// 哨兵作为错误链根因保留，对外失败结果统一为 [*apperr.Error]，
// 错误码由 [sentinelCodes] 登记。
var (
	ErrEmptyKey               = errors.New("application: empty key")                            // 插件主键或暴露/依赖的资源键为空。
	ErrInvalidKey             = errors.New("application: invalid key")                          // 键格式不符合规范（例如未通过 ctxkey.Key.IsValid 校验）。
	ErrRequestKeyShared       = errors.New("application: request-level key cannot be shared")   // 请求级上下文键（如 ctxkey.RequestKey）被错误地声明为共享插件键或暴露键。
	ErrSelfProvidedKey        = errors.New("application: key listed in its own provides")       // 插件在其 [PluginInfo.Provides] 中列出了其自身的插件主键。
	ErrMissingDependency      = errors.New("application: missing dependency")                   // 插件声明的 [PluginInfo.Requires] 依赖键在整个应用中未找到对应的提供者。
	ErrSelfDependency         = errors.New("application: self dependency")                      // 插件将其自身的主键或自身暴露的资源键作为依赖项。
	ErrDependencyCycle        = errors.New("application: dependency cycle detected")            // 插件之间存在相互依赖构成的有向循环圈，拓扑排序无法收敛。
	ErrComponentHasProvides   = errors.New("application: component cannot have provides")       // 按需激发的 [ModeComponent] 插件错误地声明了静态 [PluginInfo.Provides]。
	ErrInvalidOverlapKind     = errors.New("application: invalid overlap kind")                 // Component 声明了未知的重入策略类型 [OverlapKind]。
	ErrInvalidOverlapCapacity = errors.New("application: invalid overlap capacity")             // 重入策略容量配额不合法（如并发数或队列长度小于 1）。
	ErrInvalidOverlapConfig   = errors.New("application: invalid overlap config")               // 重入策略配置冲突（如非队列策略配置了队列长度，或非并发策略配置了并发数）。
	ErrNonComponentHasOverlap = errors.New("application: non-component has overlap configured") // 非 Component 插件（如 Tool 或 Worker）错误地配置了 [OverlapPolicy]。
	ErrUnknownMode            = errors.New("application: unknown mode")                         // 插件元信息中声明了未知的插件运行形态 [Mode]。
)

// dependencyGraph 拓扑依赖图。
type dependencyGraph struct {
	layers        [][]ctxkey.Key
	providerByKey map[ctxkey.Key]ctxkey.Key
	infoByKey     map[ctxkey.Key]PluginInfo
}

// buildGraph 构建并校验插件依赖图。
func buildGraph(infos []PluginInfo) (*dependencyGraph, *apperr.Error) {
	graph := &dependencyGraph{
		layers:        make([][]ctxkey.Key, 0),
		providerByKey: make(map[ctxkey.Key]ctxkey.Key),
		infoByKey:     make(map[ctxkey.Key]PluginInfo, len(infos)),
	}

	// 1. 深拷贝输入元数据，校验 Key、Mode、Provides、Overlap 与命名空间查重
	for _, raw := range infos {
		info := clonePluginInfo(raw)

		if info.Key.IsEmpty() {
			return nil, newAppErr(ErrEmptyKey, "插件主键或资源键为空")
		}
		if !info.Key.IsValid() {
			return nil, newAppErr(ErrInvalidKey, "键格式不符合规范")
		}
		if info.Key == ctxkey.RequestKey {
			return nil, newAppErr(ErrRequestKeyShared, "请求级上下文键不得作为共享插件键或暴露键")
		}

		switch info.Mode {
		case ModeTool, ModeWorker:
			if info.Overlap != (OverlapPolicy{}) {
				return nil, newAppErr(ErrNonComponentHasOverlap, "非 Component 插件不得配置重入策略")
			}
		case ModeComponent:
			if len(info.Provides) > 0 {
				return nil, newAppErr(ErrComponentHasProvides, "Component 不得声明静态 Provides")
			}
			if err := validateComponentOverlap(info.Overlap); err != nil {
				return nil, err
			}
		default:
			return nil, newAppErr(ErrUnknownMode, "插件元信息声明了未知的运行形态")
		}

		if _, exists := graph.providerByKey[info.Key]; exists {
			return nil, newAppErr(ErrDuplicateExposedKey, "插件主键 "+string(info.Key)+" 重复注册")
		}
		graph.providerByKey[info.Key] = info.Key

		for _, p := range info.Provides {
			if p.IsEmpty() {
				return nil, newAppErr(ErrEmptyKey, "插件主键或资源键为空")
			}
			if !p.IsValid() {
				return nil, newAppErr(ErrInvalidKey, "键格式不符合规范")
			}
			if p == ctxkey.RequestKey {
				return nil, newAppErr(ErrRequestKeyShared, "请求级上下文键不得作为共享插件键或暴露键")
			}
			if p == info.Key {
				return nil, newAppErr(ErrSelfProvidedKey, "插件 "+string(info.Key)+" 在 Provides 中声明了自身主键")
			}
			if _, exists := graph.providerByKey[p]; exists {
				return nil, newAppErr(ErrDuplicateExposedKey, "资源键 "+string(p)+" 重复注册")
			}
			graph.providerByKey[p] = info.Key
		}

		graph.infoByKey[info.Key] = info
	}

	// 2. 解析依赖，语义去重提供者边，保留原 Requires 白名单
	depEdges := make(map[ctxkey.Key]map[ctxkey.Key]struct{}, len(graph.infoByKey))
	inDegree := make(map[ctxkey.Key]int, len(graph.infoByKey))
	for k := range graph.infoByKey {
		inDegree[k] = 0
	}

	for consumerKey, info := range graph.infoByKey {
		depEdges[consumerKey] = make(map[ctxkey.Key]struct{})
		for _, req := range info.Requires {
			if req.IsEmpty() {
				return nil, newAppErr(ErrEmptyKey, "插件主键或资源键为空")
			}
			if !req.IsValid() {
				return nil, newAppErr(ErrInvalidKey, "键格式不符合规范")
			}
			providerKey, ok := graph.providerByKey[req]
			if !ok {
				return nil, newAppErr(ErrMissingDependency, "依赖键 "+string(req)+" 在应用中无对应提供者（被 "+string(consumerKey)+" 依赖）")
			}
			if providerKey == consumerKey {
				return nil, newAppErr(ErrSelfDependency, "插件 "+string(consumerKey)+" 依赖了自身提供的 "+string(req))
			}
			providerInfo := graph.infoByKey[providerKey]
			if providerInfo.Mode == ModeComponent {
				return nil, newAppErr(ErrInvalidDependencyScope, "Component "+string(providerKey)+" 不得被 "+string(consumerKey)+" 静态依赖")
			}
			depEdges[consumerKey][providerKey] = struct{}{}
		}
	}

	// 3. 构建提供者到消费者的出边图，并统计入度
	dependents := make(map[ctxkey.Key][]ctxkey.Key, len(graph.infoByKey))
	for consumer, providers := range depEdges {
		inDegree[consumer] = len(providers)
		for provider := range providers {
			dependents[provider] = append(dependents[provider], consumer)
		}
	}

	// 4. Kahn 稳定分层拓扑排序（同层按 Key 字符串排序）
	var currentLayer []ctxkey.Key
	for k, deg := range inDegree {
		if deg == 0 {
			currentLayer = append(currentLayer, k)
		}
	}
	sortKeys(currentLayer)

	processedCount := 0
	for len(currentLayer) > 0 {
		graph.layers = append(graph.layers, currentLayer)
		processedCount += len(currentLayer)

		nextLayerCandidates := make(map[ctxkey.Key]struct{})
		for _, node := range currentLayer {
			for _, dep := range dependents[node] {
				inDegree[dep]--
				if inDegree[dep] == 0 {
					nextLayerCandidates[dep] = struct{}{}
				}
			}
		}

		nextLayer := make([]ctxkey.Key, 0, len(nextLayerCandidates))
		for k := range nextLayerCandidates {
			nextLayer = append(nextLayer, k)
		}
		sortKeys(nextLayer)
		currentLayer = nextLayer
	}

	if processedCount < len(graph.infoByKey) {
		return nil, newAppErr(ErrDependencyCycle, "插件依赖构成有向循环，拓扑排序无法收敛")
	}

	return graph, nil
}

func validateComponentOverlap(p OverlapPolicy) *apperr.Error {
	switch p.Kind {
	case 0:
		if p.MaxConcurrency != 0 || p.MaxQueue != 0 {
			return newAppErr(ErrInvalidOverlapCapacity, "重入策略容量配额不合法")
		}
		return nil
	case OverlapSkip:
		if p.MaxConcurrency != 0 || p.MaxQueue != 0 {
			return newAppErr(ErrInvalidOverlapCapacity, "重入策略容量配额不合法")
		}
		return nil
	case OverlapAllow:
		if p.MaxConcurrency < 1 {
			return newAppErr(ErrInvalidOverlapCapacity, "重入策略容量配额不合法")
		}
		if p.MaxQueue != 0 {
			return newAppErr(ErrInvalidOverlapConfig, "重入策略配置冲突")
		}
		return nil
	case OverlapQueue:
		if p.MaxQueue < 1 {
			return newAppErr(ErrInvalidOverlapCapacity, "重入策略容量配额不合法")
		}
		if p.MaxConcurrency != 0 {
			return newAppErr(ErrInvalidOverlapConfig, "重入策略配置冲突")
		}
		return nil
	default:
		return newAppErr(ErrInvalidOverlapKind, "未知的重入策略类型")
	}
}

func clonePluginInfo(raw PluginInfo) PluginInfo {
	cloned := raw
	if raw.Provides != nil {
		cloned.Provides = make([]ctxkey.Key, len(raw.Provides))
		copy(cloned.Provides, raw.Provides)
	}
	if raw.Requires != nil {
		cloned.Requires = make([]ctxkey.Key, len(raw.Requires))
		copy(cloned.Requires, raw.Requires)
	}
	return cloned
}

func sortKeys(keys []ctxkey.Key) {
	sort.Slice(keys, func(i, j int) bool {
		return string(keys[i]) < string(keys[j])
	})
}
