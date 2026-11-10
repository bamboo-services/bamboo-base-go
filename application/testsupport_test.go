package application

import (
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// ctxErrToAppErr 将上下文取消或超时原因转换为统一错误，供测试替身复用。
//
// 测试替身必须遵守新的强类型契约，因此不能直接返回原生 context 错误；
// 该助手保留原始哨兵作为根因，取消与超时仍可经 errors.Is 精确判定。
func ctxErrToAppErr(err error) *apperr.Error {
	if err == nil {
		return nil
	}
	return normalizeErr(apperr.FrameworkInternalError, "上下文已结束", err)
}

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
