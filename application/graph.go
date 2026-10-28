package application

import (
	"errors"
	"fmt"
	"sort"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// 拓扑图构建与依赖校验错误哨兵。
var (
	// ErrEmptyKey 表示插件主键或暴露/依赖的资源键为空。
	ErrEmptyKey = errors.New("application: empty key")

	// ErrInvalidKey 表示键格式不符合规范（例如未通过 ctxkey.Key.IsValid 校验）。
	ErrInvalidKey = errors.New("application: invalid key")

	// ErrRequestKeyShared 表示请求级上下文键（如 ctxkey.RequestKey）被错误地声明为共享插件键或暴露键。
	ErrRequestKeyShared = errors.New("application: request-level key cannot be shared")

	// ErrSelfProvidedKey 表示插件在其 [PluginInfo.Provides] 中列出了其自身的插件主键。
	ErrSelfProvidedKey = errors.New("application: key listed in its own provides")

	// ErrMissingDependency 表示插件声明的 [PluginInfo.Requires] 依赖键在整个应用中未找到对应的提供者。
	ErrMissingDependency = errors.New("application: missing dependency")

	// ErrSelfDependency 表示插件将其自身的主键或自身暴露的资源键作为依赖项。
	ErrSelfDependency = errors.New("application: self dependency")

	// ErrDependencyCycle 表示插件之间存在相互依赖构成的有向循环圈，拓扑排序无法收敛。
	ErrDependencyCycle = errors.New("application: dependency cycle detected")

	// ErrComponentHasProvides 表示按需激发的 [ModeComponent] 插件错误地声明了静态 [PluginInfo.Provides]。
	ErrComponentHasProvides = errors.New("application: component cannot have provides")

	// ErrInvalidOverlapKind 表示 Component 声明了未知的重入策略类型 [OverlapKind]。
	ErrInvalidOverlapKind = errors.New("application: invalid overlap kind")

	// ErrInvalidOverlapCapacity 表示重入策略容量配额不合法（如并发数或队列长度小于 1）。
	ErrInvalidOverlapCapacity = errors.New("application: invalid overlap capacity")

	// ErrInvalidOverlapConfig 表示重入策略配置冲突（如非队列策略配置了队列长度，或非并发策略配置了并发数）。
	ErrInvalidOverlapConfig = errors.New("application: invalid overlap config")

	// ErrNonComponentHasOverlap 表示非 Component 插件（如 Tool 或 Worker）错误地配置了 [OverlapPolicy]。
	ErrNonComponentHasOverlap = errors.New("application: non-component has overlap configured")

	// ErrUnknownMode 表示插件元信息中声明了未知的插件运行形态 [Mode]。
	ErrUnknownMode = errors.New("application: unknown mode")
)

// dependencyGraph 拓扑依赖图。
type dependencyGraph struct {
	layers        [][]ctxkey.Key
	providerByKey map[ctxkey.Key]ctxkey.Key
	infoByKey     map[ctxkey.Key]PluginInfo
}

// buildGraph 构建并校验插件依赖图。
func buildGraph(infos []PluginInfo) (*dependencyGraph, error) {
	graph := &dependencyGraph{
		layers:        make([][]ctxkey.Key, 0),
		providerByKey: make(map[ctxkey.Key]ctxkey.Key),
		infoByKey:     make(map[ctxkey.Key]PluginInfo, len(infos)),
	}

	// 1. 深拷贝输入元数据，校验 Key、Mode、Provides、Overlap 与命名空间查重
	for _, raw := range infos {
		info := clonePluginInfo(raw)

		if info.Key.IsEmpty() {
			return nil, ErrEmptyKey
		}
		if !info.Key.IsValid() {
			return nil, ErrInvalidKey
		}
		if info.Key == ctxkey.RequestKey {
			return nil, ErrRequestKeyShared
		}

		switch info.Mode {
		case ModeTool, ModeWorker:
			if info.Overlap != (OverlapPolicy{}) {
				return nil, ErrNonComponentHasOverlap
			}
		case ModeComponent:
			if len(info.Provides) > 0 {
				return nil, ErrComponentHasProvides
			}
			if err := validateComponentOverlap(info.Overlap); err != nil {
				return nil, err
			}
		default:
			return nil, ErrUnknownMode
		}

		if _, exists := graph.providerByKey[info.Key]; exists {
			return nil, fmt.Errorf("%w: plugin key %s", ErrDuplicateExposedKey, info.Key)
		}
		graph.providerByKey[info.Key] = info.Key

		for _, p := range info.Provides {
			if p.IsEmpty() {
				return nil, ErrEmptyKey
			}
			if !p.IsValid() {
				return nil, ErrInvalidKey
			}
			if p == ctxkey.RequestKey {
				return nil, ErrRequestKeyShared
			}
			if p == info.Key {
				return nil, fmt.Errorf("%w: plugin %s provides itself", ErrSelfProvidedKey, info.Key)
			}
			if _, exists := graph.providerByKey[p]; exists {
				return nil, fmt.Errorf("%w: resource key %s", ErrDuplicateExposedKey, p)
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
				return nil, ErrEmptyKey
			}
			if !req.IsValid() {
				return nil, ErrInvalidKey
			}
			providerKey, ok := graph.providerByKey[req]
			if !ok {
				return nil, fmt.Errorf("%w: key %s required by %s", ErrMissingDependency, req, consumerKey)
			}
			if providerKey == consumerKey {
				return nil, fmt.Errorf("%w: %s requires %s provided by itself", ErrSelfDependency, consumerKey, req)
			}
			providerInfo := graph.infoByKey[providerKey]
			if providerInfo.Mode == ModeComponent {
				return nil, fmt.Errorf("%w: component %s cannot be required statically by %s", ErrInvalidDependencyScope, providerKey, consumerKey)
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
		return nil, ErrDependencyCycle
	}

	return graph, nil
}

func validateComponentOverlap(p OverlapPolicy) error {
	switch p.Kind {
	case 0:
		if p.MaxConcurrency != 0 || p.MaxQueue != 0 {
			return ErrInvalidOverlapCapacity
		}
		return nil
	case OverlapSkip:
		if p.MaxConcurrency != 0 || p.MaxQueue != 0 {
			return ErrInvalidOverlapCapacity
		}
		return nil
	case OverlapAllow:
		if p.MaxConcurrency < 1 {
			return ErrInvalidOverlapCapacity
		}
		if p.MaxQueue != 0 {
			return ErrInvalidOverlapConfig
		}
		return nil
	case OverlapQueue:
		if p.MaxQueue < 1 {
			return ErrInvalidOverlapCapacity
		}
		if p.MaxConcurrency != 0 {
			return ErrInvalidOverlapConfig
		}
		return nil
	default:
		return ErrInvalidOverlapKind
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
