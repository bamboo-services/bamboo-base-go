package application

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

func TestBuildGraph_BasicAndLayering(t *testing.T) {
	dbKey := ctxkey.Key("db_plugin")
	dbRes := ctxkey.Key("context_db_res")
	cacheKey := ctxkey.Key("cache_plugin")
	workerKey := ctxkey.Key("worker_plugin")
	compKey := ctxkey.Key("comp_plugin")

	infos := []PluginInfo{
		{
			Key:      workerKey,
			Name:     "worker",
			Mode:     ModeWorker,
			Requires: []ctxkey.Key{dbRes, cacheKey, dbKey}, // 多键指向同一提供者，应去重为单边
		},
		{
			Key:      dbKey,
			Name:     "db",
			Mode:     ModeTool,
			Provides: []ctxkey.Key{dbRes},
		},
		{
			Key:  cacheKey,
			Name: "cache",
			Mode: ModeTool,
		},
		{
			Key:      compKey,
			Name:     "comp",
			Mode:     ModeComponent,
			Requires: []ctxkey.Key{workerKey},
			Overlap:  OverlapPolicy{Kind: OverlapQueue, MaxQueue: 5},
		},
	}

	g, err := buildGraph(infos)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// 稳定排序验证：Layer 0: cacheKey, dbKey (c < d); Layer 1: workerKey; Layer 2: compKey
	expectedLayers := [][]ctxkey.Key{
		{cacheKey, dbKey},
		{workerKey},
		{compKey},
	}
	if !reflect.DeepEqual(g.layers, expectedLayers) {
		t.Fatalf("layers mismatch: got %v, want %v", g.layers, expectedLayers)
	}

	// 原 Requires 白名单保持
	workerInfo := g.infoByKey[workerKey]
	if len(workerInfo.Requires) != 3 {
		t.Fatalf("worker requires len mismatch: got %d, want 3", len(workerInfo.Requires))
	}
}

func TestBuildGraph_InputImmutability(t *testing.T) {
	key := ctxkey.Key("tool_a")
	reqKey := ctxkey.Key("tool_b")
	infos := []PluginInfo{
		{Key: key, Mode: ModeTool, Requires: []ctxkey.Key{reqKey}},
		{Key: reqKey, Mode: ModeTool},
	}

	g, err := buildGraph(infos)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	infos[0].Requires[0] = ctxkey.Key("mutated")
	if g.infoByKey[key].Requires[0] != reqKey {
		t.Fatalf("input mutation leaked into graph metadata")
	}
}

func TestBuildGraph_ValidationFailures(t *testing.T) {
	k1 := ctxkey.Key("plugin_one")
	k2 := ctxkey.Key("plugin_two")
	res := ctxkey.Key("shared_res")

	tests := []struct {
		name    string
		infos   []PluginInfo
		wantErr error
	}{
		{
			name:    "empty key",
			infos:   []PluginInfo{{Key: "", Mode: ModeTool}},
			wantErr: ErrEmptyKey,
		},
		{
			name:    "invalid key",
			infos:   []PluginInfo{{Key: "1_invalid_key", Mode: ModeTool}},
			wantErr: ErrInvalidKey,
		},
		{
			name:    "request key shared",
			infos:   []PluginInfo{{Key: ctxkey.RequestKey, Mode: ModeTool}},
			wantErr: ErrRequestKeyShared,
		},
		{
			name:    "self provided key",
			infos:   []PluginInfo{{Key: k1, Mode: ModeTool, Provides: []ctxkey.Key{k1}}},
			wantErr: ErrSelfProvidedKey,
		},
		{
			name: "duplicate exposed key across plugins",
			infos: []PluginInfo{
				{Key: k1, Mode: ModeTool, Provides: []ctxkey.Key{res}},
				{Key: k2, Mode: ModeTool, Provides: []ctxkey.Key{res}},
			},
			wantErr: ErrDuplicateExposedKey,
		},
		{
			name:    "missing dependency",
			infos:   []PluginInfo{{Key: k1, Mode: ModeTool, Requires: []ctxkey.Key{"non_exist"}}},
			wantErr: ErrMissingDependency,
		},
		{
			name:    "self dependency",
			infos:   []PluginInfo{{Key: k1, Mode: ModeTool, Requires: []ctxkey.Key{k1}}},
			wantErr: ErrSelfDependency,
		},
		{
			name: "cycle detected",
			infos: []PluginInfo{
				{Key: k1, Mode: ModeTool, Requires: []ctxkey.Key{k2}},
				{Key: k2, Mode: ModeTool, Requires: []ctxkey.Key{k1}},
			},
			wantErr: ErrDependencyCycle,
		},
		{
			name:    "component has provides",
			infos:   []PluginInfo{{Key: k1, Mode: ModeComponent, Provides: []ctxkey.Key{res}}},
			wantErr: ErrComponentHasProvides,
		},
		{
			name: "component statically required",
			infos: []PluginInfo{
				{Key: k1, Mode: ModeComponent},
				{Key: k2, Mode: ModeWorker, Requires: []ctxkey.Key{k1}},
			},
			wantErr: ErrInvalidDependencyScope,
		},
		{
			name:    "non component has overlap",
			infos:   []PluginInfo{{Key: k1, Mode: ModeTool, Overlap: OverlapPolicy{Kind: OverlapSkip}}},
			wantErr: ErrNonComponentHasOverlap,
		},
		{
			name:    "invalid overlap capacity - allow with 0",
			infos:   []PluginInfo{{Key: k1, Mode: ModeComponent, Overlap: OverlapPolicy{Kind: OverlapAllow, MaxConcurrency: 0}}},
			wantErr: ErrInvalidOverlapCapacity,
		},
		{
			name:    "invalid overlap config - queue with concurrency",
			infos:   []PluginInfo{{Key: k1, Mode: ModeComponent, Overlap: OverlapPolicy{Kind: OverlapQueue, MaxQueue: 1, MaxConcurrency: 1}}},
			wantErr: ErrInvalidOverlapConfig,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildGraph(tc.infos)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error is %v, got %v", tc.wantErr, err)
			}
		})
	}
}
