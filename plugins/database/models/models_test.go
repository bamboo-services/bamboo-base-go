package models

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
	"gorm.io/gorm"
)

type customOrder struct {
	BaseEntity
	OrderNo string
}

func (c *customOrder) GetGene() snowflake.Gene {
	return snowflake.GeneOrder
}

type invalidGeneEntity struct {
	BaseEntity
}

func (i *invalidGeneEntity) GetGene() snowflake.Gene {
	return snowflake.Gene(99)
}

type customSoftDeleteEntity struct {
	BaseEntityWithSoftDelete
}

func (c *customSoftDeleteEntity) GetGene() snowflake.Gene {
	return snowflake.GeneUser
}

type testSnowflakeTool struct {
	node *snowflake.Node
}

func (t *testSnowflakeTool) Start(ctx context.Context) error {
	return nil
}

func (t *testSnowflakeTool) Stop(ctx context.Context) error {
	return nil
}

func (t *testSnowflakeTool) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		ctxkey.SnowflakeNodeKey: t.node,
	}
}

type testWorker struct {
	ctxChan chan<- context.Context
}

func (w *testWorker) Start(ctx context.Context) error {
	return nil
}

func (w *testWorker) Stop(ctx context.Context) error {
	return nil
}

func (w *testWorker) Run(ctx context.Context) error {
	select {
	case w.ctxChan <- ctx:
	default:
	}
	<-ctx.Done()
	return nil
}

func runInTestApp(t *testing.T, node *snowflake.Node, testFn func(ctx context.Context)) {
	t.Helper()
	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	toolKey := ctxkey.Key("test_tool_snowflake")
	workerKey := ctxkey.Key("test_worker")

	err := app.RegisterTool(application.PluginInfo{
		Key:      toolKey,
		Provides: []ctxkey.Key{ctxkey.SnowflakeNodeKey},
	}, func() application.Plugin {
		return &testSnowflakeTool{node: node}
	})
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	workerCtxChan := make(chan context.Context, 1)

	err = app.RegisterWorker(application.PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{ctxkey.SnowflakeNodeKey},
	}, func() application.RunnablePlugin {
		return &testWorker{
			ctxChan: workerCtxChan,
		}
	})
	if err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrChan := make(chan error, 1)
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

	var workerCtx context.Context
	select {
	case err := <-runErrChan:
		t.Fatalf("app run exited prematurely: %v", err)
	case workerCtx = <-workerCtxChan:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for worker context")
	}

	testFn(workerCtx)
}

func Test_BaseEntity_BeforeCreate_Success(t *testing.T) {
	node, err := snowflake.NewNode(1, 2)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	runInTestApp(t, node, func(ctx context.Context) {
		ent := &BaseEntity{}
		stmt := &gorm.Statement{
			Context: ctx,
			Dest:    ent,
		}
		tx := &gorm.DB{Statement: stmt}

		if err := ent.BeforeCreate(tx); err != nil {
			t.Fatalf("BeforeCreate failed: %v", err)
		}
		if ent.ID.IsZero() {
			t.Fatalf("expected non-zero ID")
		}
		if ent.ID.Gene() != snowflake.GeneDefault {
			t.Fatalf("expected gene default, got %v", ent.ID.Gene())
		}
		if ent.CreatedAt.IsZero() || ent.UpdatedAt.IsZero() {
			t.Fatalf("timestamps should not be zero")
		}
	})
}

func Test_BaseEntity_BeforeCreate_GeneProvider(t *testing.T) {
	node, err := snowflake.NewNode(2, 3)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	runInTestApp(t, node, func(ctx context.Context) {
		order := &customOrder{OrderNo: "ORD123"}
		stmt := &gorm.Statement{
			Context: ctx,
			Dest:    order,
		}
		tx := &gorm.DB{Statement: stmt}

		if err := order.BeforeCreate(tx); err != nil {
			t.Fatalf("BeforeCreate failed: %v", err)
		}
		if order.ID.IsZero() {
			t.Fatalf("expected non-zero ID")
		}
		if order.ID.Gene() != snowflake.GeneOrder {
			t.Fatalf("expected GeneOrder, got %v", order.ID.Gene())
		}
	})
}

func Test_BaseEntity_BeforeCreate_PreserveExistingID(t *testing.T) {
	ent := &BaseEntity{ID: snowflake.SnowflakeID(123456789)}
	// 没有 context 和 node 也不应该报错，因为已有显式非零 ID
	stmt := &gorm.Statement{
		Context: context.Background(),
		Dest:    ent,
	}
	tx := &gorm.DB{Statement: stmt}

	if err := ent.BeforeCreate(tx); err != nil {
		t.Fatalf("BeforeCreate failed: %v", err)
	}
	if ent.ID != snowflake.SnowflakeID(123456789) {
		t.Fatalf("expected ID preserved 123456789, got %v", ent.ID)
	}
}

func Test_BaseEntity_BeforeCreate_MissingNode(t *testing.T) {
	ent := &BaseEntity{}
	stmt := &gorm.Statement{
		Context: context.Background(),
		Dest:    ent,
	}
	tx := &gorm.DB{Statement: stmt}

	err := ent.BeforeCreate(tx)
	if err == nil {
		t.Fatalf("expected error when snowflake node missing")
	}
	if !errors.Is(err, ErrSnowflakeNodeNotFound) {
		t.Fatalf("expected ErrSnowflakeNodeNotFound, got %v", err)
	}
	if !ent.ID.IsZero() {
		t.Fatalf("expected ID to remain zero on error, got %v", ent.ID)
	}
}

func Test_BaseEntity_BeforeCreate_NilStatementOrContext(t *testing.T) {
	ent := &BaseEntity{}
	if err := ent.BeforeCreate(nil); !errors.Is(err, ErrNilStatement) {
		t.Fatalf("expected ErrNilStatement, got %v", err)
	}
	if err := ent.BeforeCreate(&gorm.DB{Statement: nil}); !errors.Is(err, ErrNilStatement) {
		t.Fatalf("expected ErrNilStatement, got %v", err)
	}
	if err := ent.BeforeCreate(&gorm.DB{Statement: &gorm.Statement{Context: nil}}); !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

func Test_BaseEntity_BeforeCreate_InvalidGene(t *testing.T) {
	node, err := snowflake.NewNode(1, 1)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	runInTestApp(t, node, func(ctx context.Context) {
		bad := &invalidGeneEntity{}
		stmt := &gorm.Statement{
			Context: ctx,
			Dest:    bad,
		}
		tx := &gorm.DB{Statement: stmt}

		err = bad.BeforeCreate(tx)
		if err == nil {
			t.Fatalf("expected error on invalid gene")
		}
		if !bad.ID.IsZero() {
			t.Fatalf("expected ID to remain zero on error, got %v", bad.ID)
		}
	})
}

func Test_BaseEntity_BeforeUpdate(t *testing.T) {
	ent := &BaseEntity{}
	before := time.Now().Add(-1 * time.Second)
	ent.UpdatedAt = before

	if err := ent.BeforeUpdate(nil); err != nil {
		t.Fatalf("BeforeUpdate failed: %v", err)
	}
	if !ent.UpdatedAt.After(before) {
		t.Fatalf("expected UpdatedAt updated")
	}
}

func Test_BaseEntityWithSoftDelete_BeforeCreate_AndUpdate(t *testing.T) {
	node, err := snowflake.NewNode(3, 4)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	runInTestApp(t, node, func(ctx context.Context) {
		ent := &customSoftDeleteEntity{}
		stmt := &gorm.Statement{
			Context: ctx,
			Dest:    ent,
		}
		tx := &gorm.DB{Statement: stmt}

		if err := ent.BeforeCreate(tx); err != nil {
			t.Fatalf("BeforeCreate failed: %v", err)
		}
		if ent.ID.IsZero() {
			t.Fatalf("expected non-zero ID")
		}
		if ent.ID.Gene() != snowflake.GeneUser {
			t.Fatalf("expected GeneUser, got %v", ent.ID.Gene())
		}
		if ent.DeletedAt.Valid {
			t.Fatalf("expected DeletedAt to be invalid initially")
		}

		before := ent.UpdatedAt
		time.Sleep(10 * time.Millisecond)
		if err := ent.BeforeUpdate(tx); err != nil {
			t.Fatalf("BeforeUpdate failed: %v", err)
		}
		if !ent.UpdatedAt.After(before) {
			t.Fatalf("expected UpdatedAt updated")
		}
	})
}

func Test_PageRequest_And_PageResponse(t *testing.T) {
	req := PageRequest{Page: 2, Size: 10, Sort: SortDesc}
	if req.Offset() != 10 {
		t.Fatalf("expected offset 10, got %d", req.Offset())
	}
	if req.Limit() != 10 {
		t.Fatalf("expected limit 10, got %d", req.Limit())
	}
	if req.OrderDirection() != "desc" {
		t.Fatalf("expected order direction desc, got %s", req.OrderDirection())
	}

	// 边界测试
	boundaryReq := PageRequest{Page: -5, Size: -20, Sort: "invalid"}
	norm := boundaryReq.Normalize()
	if norm.Page != DefaultPageNumber || norm.Size != DefaultPageSize || norm.Sort != SortAsc {
		t.Fatalf("unexpected normalized request: %+v", norm)
	}

	// PageResponse 计算
	page := NewPage(2, 10, 25, []string{"a", "b"})
	if page.CurrentPage != 2 || page.TotalPages != 3 || page.TotalItems != 25 || page.Size != 10 {
		t.Fatalf("unexpected page: %+v", page)
	}

	pageZero := NewPage(-1, 0, -5, []string{})
	if pageZero.CurrentPage != DefaultPageNumber || pageZero.Size != DefaultPageSize || pageZero.TotalItems != 0 || pageZero.TotalPages != 0 {
		t.Fatalf("unexpected pageZero: %+v", pageZero)
	}

	data, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("failed to marshal page: %v", err)
	}
	expectedJSON := `{"current_page":2,"total_pages":3,"total_items":25,"size":10,"items":["a","b"]}`
	if string(data) != expectedJSON {
		t.Fatalf("JSON golden mismatch:\nexpected: %s\ngot:      %s", expectedJSON, string(data))
	}
}
