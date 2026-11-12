package models

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
	"gorm.io/gorm"
)

// 编译期断言：四个模型钩子的签名必须保持 func(tx *gorm.DB) error，
// 强类型错误只在钩子内部转换为 error 接口，失败时才发生非 nil 转换。
var (
	_ func(*BaseEntity, *gorm.DB) error               = (*BaseEntity).BeforeCreate
	_ func(*BaseEntity, *gorm.DB) error               = (*BaseEntity).BeforeUpdate
	_ func(*BaseEntityWithSoftDelete, *gorm.DB) error = (*BaseEntityWithSoftDelete).BeforeCreate
	_ func(*BaseEntityWithSoftDelete, *gorm.DB) error = (*BaseEntityWithSoftDelete).BeforeUpdate
)

// otherKeyTool 是仅暴露无关组件键的测试插件，用于构造"容器已挂载但目标组件缺失"的上下文。
type otherKeyTool struct{}

// Start 实现 application.Plugin 生命周期钩子。
func (o *otherKeyTool) Start(ctx context.Context) *apperr.Error {
	return nil
}

// Stop 实现 application.Plugin 生命周期钩子。
func (o *otherKeyTool) Stop(ctx context.Context) *apperr.Error {
	return nil
}

// Expose 暴露与雪花节点无关的组件键。
func (o *otherKeyTool) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		ctxkey.Key("test_unrelated_key"): "unrelated",
	}
}

// otherKeyWorker 是持有运行期容器视图上下文的测试后台服务。
type otherKeyWorker struct {
	ctxChan chan<- context.Context
}

// Start 实现 application.RunnablePlugin 生命周期钩子。
func (w *otherKeyWorker) Start(ctx context.Context) *apperr.Error {
	return nil
}

// Stop 实现 application.RunnablePlugin 生命周期钩子。
func (w *otherKeyWorker) Stop(ctx context.Context) *apperr.Error {
	return nil
}

// Run 阻塞等待停机信号，并向测试暴露运行期容器视图上下文。
func (w *otherKeyWorker) Run(ctx context.Context) *apperr.Error {
	select {
	case w.ctxChan <- ctx:
	default:
	}
	<-ctx.Done()
	return nil
}

// runInAppWithoutSnowflake 启动一个已挂载容器但未暴露雪花节点的应用，
// 用于验证组件缺失时 application 层错误码必须原样保留。
func runInAppWithoutSnowflake(t *testing.T, testFn func(ctx context.Context)) {
	t.Helper()
	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	toolKey := ctxkey.Key("test_tool_unrelated")
	workerKey := ctxkey.Key("test_worker_unrelated")

	if err := app.RegisterTool(application.PluginInfo{
		Key:      toolKey,
		Provides: []ctxkey.Key{ctxkey.Key("test_unrelated_key")},
	}, func() application.Plugin {
		return &otherKeyTool{}
	}); err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	workerCtxChan := make(chan context.Context, 1)
	if err := app.RegisterWorker(application.PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{ctxkey.Key("test_unrelated_key")},
	}, func() application.RunnablePlugin {
		return &otherKeyWorker{ctxChan: workerCtxChan}
	}); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrChan := make(chan *apperr.Error, 1)
	go func() {
		runErrChan <- app.Run()
	}()
	t.Cleanup(func() {
		if shutdownErr := app.Shutdown(); shutdownErr != nil {
			t.Errorf("app shutdown failed: %v", shutdownErr)
		}
		select {
		case err := <-runErrChan:
			if err != nil {
				t.Errorf("app run exited with error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("timeout waiting for app run to exit after shutdown")
		}
	})

	select {
	case err := <-runErrChan:
		t.Fatalf("app run exited prematurely: %v", err)
	case workerCtx := <-workerCtxChan:
		testFn(workerCtx)
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for worker context")
	}
}

// TestGenerateEntityID_ContractViolationUsesModelInvalid 验证模型自身契约失败
// （tx/Statement/Context 缺失）统一使用 ModelInvalid 错误码，并保留模型哨兵。
func TestGenerateEntityID_ContractViolationUsesModelInvalid(t *testing.T) {
	cases := []struct {
		name     string
		tx       *gorm.DB
		sentinel error
	}{
		{name: "nil tx", tx: nil, sentinel: ErrNilStatement},
		{name: "nil statement", tx: &gorm.DB{}, sentinel: ErrNilStatement},
		{name: "nil context", tx: &gorm.DB{Statement: &gorm.Statement{}}, sentinel: ErrNilContext},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			id, err := generateEntityID(tt.tx)
			if err == nil {
				t.Fatal("期望返回强类型错误，实际为 nil")
			}
			if id != 0 {
				t.Fatalf("失败路径不应返回 ID，实际为 %d", id)
			}
			if err.Code.Code != ModelInvalid.Code || err.Code.Output != ModelInvalid.Output {
				t.Fatalf("错误码 = %+v, 期望 %+v", err.Code, *ModelInvalid)
			}
			if !errors.Is(err, tt.sentinel) {
				t.Fatalf("errors.Is(err, %v) = false, err = %v", tt.sentinel, err)
			}
			var appErr *apperr.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("errors.As 未能提取 *apperr.Error: %v", err)
			}
			if appErr.Data != nil {
				t.Fatalf("模型哨兵不应写入 Data，实际为 %v", appErr.Data)
			}
		})
	}
}

// TestGenerateEntityID_PreservesContainerErrorCode 验证容器未挂载时
// application 层错误码必须原样保留，且不得被洗成 ModelInvalid。
func TestGenerateEntityID_PreservesContainerErrorCode(t *testing.T) {
	_, err := generateEntityID(&gorm.DB{Statement: &gorm.Statement{Context: context.Background()}})
	if err == nil {
		t.Fatal("期望返回强类型错误，实际为 nil")
	}
	if err.Code.Code != apperr.FrameworkContainerNotMounted.Code {
		t.Fatalf("错误码 = %+v, 期望保留容器错误码 %+v", err.Code, *apperr.FrameworkContainerNotMounted)
	}
	if err.Code.Code == ModelInvalid.Code {
		t.Fatal("容器错误不得被洗成 ModelInvalid")
	}
	if !errors.Is(err, ErrSnowflakeNodeNotFound) {
		t.Fatalf("errors.Is(err, ErrSnowflakeNodeNotFound) = false, err = %v", err)
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("errors.As 未能提取 *apperr.Error: %v", err)
	}
	if appErr.Data != nil {
		t.Fatalf("模型哨兵不应写入 Data，实际为 %v", appErr.Data)
	}
}

// TestGenerateEntityID_PreservesComponentNotFoundCode 验证组件缺失时
// application 层组件错误码必须原样保留，并同时携带 ErrSnowflakeNodeNotFound 哨兵。
func TestGenerateEntityID_PreservesComponentNotFoundCode(t *testing.T) {
	runInAppWithoutSnowflake(t, func(ctx context.Context) {
		_, err := generateEntityID(&gorm.DB{Statement: &gorm.Statement{Context: ctx}})
		if err == nil {
			t.Fatal("期望返回强类型错误，实际为 nil")
		}
		if err.Code.Code != apperr.FrameworkComponentNotFound.Code {
			t.Fatalf("错误码 = %+v, 期望保留组件错误码 %+v", err.Code, *apperr.FrameworkComponentNotFound)
		}
		if err.Code.Code == ModelInvalid.Code {
			t.Fatal("组件缺失错误不得被洗成 ModelInvalid")
		}
		if !errors.Is(err, ErrSnowflakeNodeNotFound) {
			t.Fatalf("errors.Is(err, ErrSnowflakeNodeNotFound) = false, err = %v", err)
		}
	})
}

// TestGenerateEntityID_NilNodeIsUnreachableThroughContainer 验证容器不允许发布空的雪花节点，
// 因此 generateEntityID 中的空节点分支属于防御性守卫，无法通过公开 API 触发。
func TestGenerateEntityID_NilNodeIsUnreachableThroughContainer(t *testing.T) {
	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	if err := app.RegisterTool(application.PluginInfo{
		Key:      ctxkey.Key("test_tool_nil_node"),
		Provides: []ctxkey.Key{ctxkey.SnowflakeNodeKey},
	}, func() application.Plugin {
		return &testSnowflakeTool{node: nil}
	}); err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	runErrChan := make(chan *apperr.Error, 1)
	go func() {
		runErrChan <- app.Run()
	}()
	t.Cleanup(func() {
		_ = app.Shutdown()
	})

	select {
	case err := <-runErrChan:
		if err == nil {
			t.Fatal("期望空节点暴露被拒绝，实际应用启动成功")
		}
		if !errors.Is(err, application.ErrExposedKeysMismatch) {
			t.Fatalf("期望暴露键校验失败，实际为 %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for app run result")
	}
}

// TestGenerateEntityID_PreservesGenerateError 验证雪花发号失败时保留发号器自身错误码，
// 不得统一洗成 ModelInvalid。
func TestGenerateEntityID_PreservesGenerateError(t *testing.T) {
	node, err := snowflake.NewNode(2, 3)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	runInTestApp(t, node, func(ctx context.Context) {
		bad := &invalidGeneEntity{}
		_, genErr := generateEntityID(&gorm.DB{Statement: &gorm.Statement{Context: ctx, Dest: bad}})
		if genErr == nil {
			t.Fatal("期望返回强类型错误，实际为 nil")
		}
		if genErr.Code.Code != valueerr.OutOfRange.Code {
			t.Fatalf("错误码 = %+v, 期望保留发号器错误码 %+v", genErr.Code, *valueerr.OutOfRange)
		}
		if genErr.Code.Code == ModelInvalid.Code {
			t.Fatal("发号器错误不得被洗成 ModelInvalid")
		}
	})
}

// TestBeforeCreate_HookErrorContract 验证两个模型的 BeforeCreate 在失败时返回可识别的
// 强类型错误，成功时返回真正的 nil（而不是装着 nil 指针的 error 接口）。
func TestBeforeCreate_HookErrorContract(t *testing.T) {
	t.Run("BaseEntity", func(t *testing.T) {
		ent := &BaseEntity{}
		hookErr := ent.BeforeCreate(&gorm.DB{Statement: &gorm.Statement{Context: context.Background()}})
		if hookErr == nil {
			t.Fatal("期望返回错误，实际为 nil")
		}
		var appErr *apperr.Error
		if !errors.As(hookErr, &appErr) {
			t.Fatalf("errors.As 未能提取 *apperr.Error: %v", hookErr)
		}
		if !errors.Is(hookErr, ErrSnowflakeNodeNotFound) {
			t.Fatalf("errors.Is(hookErr, ErrSnowflakeNodeNotFound) = false, err = %v", hookErr)
		}
	})

	t.Run("BaseEntityWithSoftDelete", func(t *testing.T) {
		ent := &BaseEntityWithSoftDelete{}
		hookErr := ent.BeforeCreate(&gorm.DB{Statement: &gorm.Statement{Context: context.Background()}})
		if hookErr == nil {
			t.Fatal("期望返回错误，实际为 nil")
		}
		var appErr *apperr.Error
		if !errors.As(hookErr, &appErr) {
			t.Fatalf("errors.As 未能提取 *apperr.Error: %v", hookErr)
		}
		if !errors.Is(hookErr, ErrSnowflakeNodeNotFound) {
			t.Fatalf("errors.Is(hookErr, ErrSnowflakeNodeNotFound) = false, err = %v", hookErr)
		}
	})
}

// TestBeforeCreate_SuccessReturnsLiteralNil 验证成功路径返回真正的 nil 错误。
func TestBeforeCreate_SuccessReturnsLiteralNil(t *testing.T) {
	node, err := snowflake.NewNode(4, 5)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	runInTestApp(t, node, func(ctx context.Context) {
		base := &BaseEntity{}
		if hookErr := base.BeforeCreate(&gorm.DB{Statement: &gorm.Statement{Context: ctx, Dest: base}}); hookErr != nil {
			t.Fatalf("BeforeCreate 期望返回 nil，实际为 %v", hookErr)
		}
		if base.ID.IsZero() {
			t.Fatal("期望生成非零 ID")
		}

		soft := &BaseEntityWithSoftDelete{}
		if hookErr := soft.BeforeCreate(&gorm.DB{Statement: &gorm.Statement{Context: ctx, Dest: soft}}); hookErr != nil {
			t.Fatalf("BeforeCreate 期望返回 nil，实际为 %v", hookErr)
		}
		if soft.ID.IsZero() {
			t.Fatal("期望生成非零 ID")
		}
	})
}

// TestBeforeCreate_PreservesNonZeroIDWithoutNode 验证非零 ID 在没有雪花节点时
// 不强制要求节点，且原值被完整保留。
func TestBeforeCreate_PreservesNonZeroIDWithoutNode(t *testing.T) {
	base := &BaseEntity{ID: snowflake.SnowflakeID(987654321)}
	if hookErr := base.BeforeCreate(&gorm.DB{Statement: &gorm.Statement{Context: context.Background()}}); hookErr != nil {
		t.Fatalf("BeforeCreate 期望返回 nil，实际为 %v", hookErr)
	}
	if base.ID != snowflake.SnowflakeID(987654321) {
		t.Fatalf("期望保留 ID 987654321，实际为 %d", base.ID)
	}

	soft := &BaseEntityWithSoftDelete{ID: snowflake.SnowflakeID(123456789)}
	if hookErr := soft.BeforeCreate(&gorm.DB{Statement: &gorm.Statement{Context: context.Background()}}); hookErr != nil {
		t.Fatalf("BeforeCreate 期望返回 nil，实际为 %v", hookErr)
	}
	if soft.ID != snowflake.SnowflakeID(123456789) {
		t.Fatalf("期望保留 ID 123456789，实际为 %d", soft.ID)
	}
}

// TestBeforeUpdate_AcceptsNilTransaction 验证 BeforeUpdate 不额外校验 tx 与上下文。
func TestBeforeUpdate_AcceptsNilTransaction(t *testing.T) {
	base := &BaseEntity{}
	if hookErr := base.BeforeUpdate(nil); hookErr != nil {
		t.Fatalf("BeforeUpdate 期望返回 nil，实际为 %v", hookErr)
	}

	soft := &BaseEntityWithSoftDelete{}
	if hookErr := soft.BeforeUpdate(nil); hookErr != nil {
		t.Fatalf("BeforeUpdate 期望返回 nil，实际为 %v", hookErr)
	}
}
