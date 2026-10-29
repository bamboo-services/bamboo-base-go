package application

import (
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// waitForStaticStopDone 阻塞等待指定静态实例 Stop 收尾完成信号。
//
// stopDone 通道在实例登记时即创建，任意时刻等待均安全；
// Stop 真实返回（含 Stop panic 路径）后关闭且仅关闭一次。
// 时间敏感测试应通过本助手同步内核异步收尾（如 Start 超时后脱离
// 关机协调的晚 Stop），禁止使用 time.Sleep 抢跑或释放屏障后立即
// 读计数等负载敏感模式。
func waitForStaticStopDone(t *testing.T, app *App, key ctxkey.Key, timeout time.Duration) {
	t.Helper()
	app.staticMu.RLock()
	rec, ok := app.staticInstances[key]
	app.staticMu.RUnlock()
	if !ok {
		t.Fatalf("static instance %v not registered", key)
	}
	rec.stopMu.Lock()
	done := rec.stopDone
	rec.stopMu.Unlock()
	if done == nil {
		t.Fatalf("static instance %v has no stopDone signal", key)
	}
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf("timed out (%v) waiting for stop completion of static instance %v", timeout, key)
	}
}
