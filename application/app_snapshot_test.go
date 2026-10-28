package application

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
)

func TestSnapshot_NewEnvAndOptionIsolation(t *testing.T) {
	key := string(env.StartTimeout)
	origEnv, existed := os.LookupEnv(key)
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, origEnv)
		} else {
			_ = os.Unsetenv(key)
		}
	})

	_ = os.Setenv(key, "5s")
	app := New()

	if app.cfg.StartTimeout != 5*time.Second {
		t.Fatalf("expected 5s, got %v", app.cfg.StartTimeout)
	}

	// 改变环境变量，已创建的 app.cfg 不应改变
	_ = os.Setenv(key, "10s")
	if app.cfg.StartTimeout != 5*time.Second {
		t.Fatalf("expected snapshot cfg to remain 5s, got %v", app.cfg.StartTimeout)
	}

	// Option 快照独立性
	customOptVal := 3 * time.Second
	optFunc := func(c *optionCollector) {
		c.startTimeout = customOptVal
		c.hasStartTimeout = true
	}
	appOpt := New(optFunc)
	if appOpt.cfg.StartTimeout != 3*time.Second {
		t.Fatalf("expected 3s, got %v", appOpt.cfg.StartTimeout)
	}
	customOptVal = 9 * time.Second
	if appOpt.cfg.StartTimeout != 3*time.Second {
		t.Fatalf("expected option snapshot to remain 3s, got %v", appOpt.cfg.StartTimeout)
	}
}

func TestSnapshot_RunConsumedRepeatAndClosed(t *testing.T) {
	// 构造 cfgErr 的 App
	appCfgErr := New(WithStartTimeout(-1 * time.Second))
	if appCfgErr.cfgErr == nil {
		t.Fatalf("expected cfgErr on invalid option")
	}

	// 首次 Run 失败（配置错误），消耗了 runConsumed
	err1 := appCfgErr.Run()
	if err1 == nil {
		t.Fatalf("expected run error due to cfgErr")
	}

	// 再次 Run 返回 ErrAlreadyRun
	err2 := appCfgErr.Run()
	if !errors.Is(err2, ErrAlreadyRun) {
		t.Fatalf("expected ErrAlreadyRun after consumed, got %v", err2)
	}

	// 注册也应当冻结
	errReg := appCfgErr.RegisterTool(PluginInfo{Key: "tool.test"}, func() Plugin { return nil })
	if !errors.Is(errReg, ErrRegistrationClosed) {
		t.Fatalf("expected ErrRegistrationClosed, got %v", errReg)
	}

	// 未 Run 封存后 Run 返回 ErrAppClosed
	appShutdown := New()
	if err := appShutdown.Shutdown(); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
	errRunAfterShutdown := appShutdown.Run()
	if !errors.Is(errRunAfterShutdown, ErrAppClosed) {
		t.Fatalf("expected ErrAppClosed, got %v", errRunAfterShutdown)
	}
}

func TestSnapshot_ErrorLedgerConcurrent(t *testing.T) {
	app := New()

	errA := errors.New("error A")
	errB := errors.New("error B")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			app.recordRunErr(fmt.Errorf("%w: %d", errA, idx))
		}(i)
		go func(idx int) {
			defer wg.Done()
			app.recordRunErr(fmt.Errorf("%w: %d", errB, idx))
		}(i)
	}
	wg.Wait()

	joined := app.getRunErr()
	if !errors.Is(joined, errA) || !errors.Is(joined, errB) {
		t.Fatalf("expected joined error to contain errA and errB, got %v", joined)
	}
}
