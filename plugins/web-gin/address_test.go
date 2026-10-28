package webgin

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
)

func TestResolveAddr_DefaultAndEnv(t *testing.T) {
	// 1. 无环境变量且未显式配置 -> localhost:1118
	t.Setenv(string(env.Host), "")
	t.Setenv(string(env.Port), "")

	_, factory := New()
	p := factory().(*Plugin)
	if got := p.Addr(); got != "localhost:1118" {
		t.Fatalf("expected default addr %q, got %q", "localhost:1118", got)
	}

	// 2. 设置 HOST / PORT 环境变量 -> 采用环境变量值
	t.Setenv(string(env.Host), "127.0.0.1")
	t.Setenv(string(env.Port), "9999")
	_, factoryEnv := New()
	pEnv := factoryEnv().(*Plugin)
	if got := pEnv.Addr(); got != "127.0.0.1:9999" {
		t.Fatalf("expected env addr %q, got %q", "127.0.0.1:9999", got)
	}

	// 3. 显式 WithAddr 明确优先于环境变量
	_, factoryExplicit := New(WithAddr("0.0.0.0:1234"))
	pExplicit := factoryExplicit().(*Plugin)
	if got := pExplicit.Addr(); got != "0.0.0.0:1234" {
		t.Fatalf("expected explicit addr %q, got %q", "0.0.0.0:1234", got)
	}
}

func TestResolveAddr_StartListenRandom(t *testing.T) {
	// 验证 Start 阶段实际监听且不占用生产 1118
	_, factory := New(WithAddr("127.0.0.1:0"))
	p := factory().(*Plugin)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = p.Stop(context.Background()) }()

	addr := p.Addr()
	if !strings.HasPrefix(addr, "127.0.0.1:") || addr == "127.0.0.1:0" {
		t.Fatalf("unexpected bound addr %q", addr)
	}

	// 验证实际端口有效可连接
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to dial bound addr: %v", err)
	}
	_ = conn.Close()
}

func TestResolveAddr_InvalidPortError(t *testing.T) {
	// 验证非法端口环境变量在 Start 时由底层 Listen 真实报错，不虚假默认
	t.Setenv(string(env.Host), "127.0.0.1")
	t.Setenv(string(env.Port), "invalid_port")

	_, factory := New()
	p := factory().(*Plugin)

	err := p.Start(context.Background())
	if err == nil {
		_ = p.Stop(context.Background())
		t.Fatalf("expected Start to fail with invalid port, but got nil")
	}
}
