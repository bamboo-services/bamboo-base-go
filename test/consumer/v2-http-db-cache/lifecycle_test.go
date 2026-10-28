package consumer_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	cache "github.com/bamboo-services/bamboo-base-go/plugins/cache/v2"
	sqlite "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2"
	database "github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
	webgin "github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/header"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DemoRecord struct {
	ID   uint   `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"size:64;not null"`
}

type CreateItemDTO struct {
	Name string `json:"name" binding:"required,min=3"`
}

type readyWeb struct {
	*webgin.Plugin
	startedAddr chan<- string
}

func (w *readyWeb) Start(ctx context.Context) error {
	if err := w.Plugin.Start(ctx); err != nil {
		return err
	}
	w.startedAddr <- w.Plugin.Addr()
	return nil
}

func TestHTTPDBCacheLifecycle_EndToEnd(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "consumer_v2_http.db")
	dbInfo, dbFac := database.New(database.WithAdapter(sqlite.New(dbPath)))
	cacheInfo, cacheFac := cache.New(cache.WithMemory())

	startedAddrCh := make(chan string, 1)
	var rawSQLDB *sql.DB

	webInfo, realWebFac := webgin.New(
		webgin.WithAddr("127.0.0.1:0"),
		webgin.WithRequires(ctxkey.DatabaseKey, ctxkey.CacheManagerKey),
		webgin.WithRoute(func(startCtx context.Context, engine *gin.Engine) {
			db, getDBErr := application.Get[*gorm.DB](startCtx, ctxkey.DatabaseKey)
			if getDBErr != nil {
				t.Errorf("startCtx get DB failed: %v", getDBErr)
				return
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Errorf("startCtx db.DB() failed: %v", err)
				return
			}
			rawSQLDB = sqlDB

			mgr, getCacheErr := application.Get[*cache.Manager](startCtx, ctxkey.CacheManagerKey)
			if getCacheErr != nil {
				t.Errorf("startCtx get CacheManager failed: %v", getCacheErr)
				return
			}

			if err := db.AutoMigrate(&DemoRecord{}); err != nil {
				t.Errorf("AutoMigrate failed: %v", err)
				return
			}

			engine.GET("/health", func(c *gin.Context) {
				httpresult.Success(c, "ready")
			})

			engine.POST("/items", func(c *gin.Context) {
				var dto CreateItemDTO
				if bound := bind.Bind[CreateItemDTO](c, &dto).Data(); bound == nil {
					return
				}

				reqCtx := c.Request.Context()
				item := DemoRecord{Name: dto.Name}
				if err := db.WithContext(reqCtx).Create(&item).Error; err != nil {
					httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to insert item", err))
					return
				}

				var count int64
				if err := db.WithContext(reqCtx).Model(&DemoRecord{}).Count(&count).Error; err != nil {
					httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to count items", err))
					return
				}

				keyCache := cache.KeyCacheOf[string, string](mgr)
				cacheVal := fmt.Sprintf("val-%s-%d", dto.Name, count)
				if err := keyCache.Set(reqCtx, "last_item", &cacheVal); err != nil {
					httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to set cache", err))
					return
				}

				valPtr, found, err := keyCache.Get(reqCtx, "last_item")
				if err != nil {
					httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to get cache", err))
					return
				}
				var cachedStr string
				if found && valPtr != nil {
					cachedStr = *valPtr
				}

				httpresult.SuccessWithData(c, "ok", map[string]any{"id": item.ID, "count": count, "cache": cachedStr})
			})
		}),
	)

	webFacWrapper := func() application.RunnablePlugin {
		rp := realWebFac()
		p, ok := rp.(*webgin.Plugin)
		if !ok {
			t.Fatalf("expected *webgin.Plugin, got %T", rp)
		}
		return &readyWeb{
			Plugin:      p,
			startedAddr: startedAddrCh,
		}
	}

	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	if err := app.RegisterTool(dbInfo, dbFac); err != nil {
		t.Fatalf("RegisterTool db: %v", err)
	}
	if err := app.RegisterTool(cacheInfo, cacheFac); err != nil {
		t.Fatalf("RegisterTool cache: %v", err)
	}
	if err := app.RegisterWorker(webInfo, webFacWrapper); err != nil {
		t.Fatalf("RegisterWorker webgin: %v", err)
	}

	appDone := make(chan struct{})
	var appRunErr error
	var runOnce sync.Once
	shutdownApp := func() {
		runOnce.Do(func() {
			_ = app.Shutdown()
			select {
			case <-appDone:
			case <-time.After(1 * time.Second):
				t.Error("timeout waiting for app to finish")
			}
		})
	}
	t.Cleanup(shutdownApp)

	go func() {
		defer close(appDone)
		appRunErr = app.Run()
	}()

	var serverBase string
	select {
	case addr := <-startedAddrCh:
		serverBase = "http://" + addr
	case <-time.After(2 * time.Second):
		t.Fatalf("service failed to start within 2s")
	}

	client := &http.Client{Timeout: 2 * time.Second}

	// 1. 无效输入：缺少 name 或 name < 3，期望 400 且不写入 DB
	req1, err := http.NewRequestWithContext(context.Background(), http.MethodPost, serverBase+"/items", bytes.NewReader([]byte(`{"name":"ab"}`)))
	if err != nil {
		t.Fatalf("create request 1 failed: %v", err)
	}
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("request 1 failed: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got: %d", resp1.StatusCode)
	}

	var countBefore int64
	if err := rawSQLDB.QueryRow("SELECT COUNT(*) FROM demo_records;").Scan(&countBefore); err != nil {
		t.Fatalf("query count failed: %v", err)
	}
	if countBefore != 0 {
		t.Fatalf("expected 0 records in DB after 400, got %d", countBefore)
	}

	// 2. 合法请求 1：固定 UUID A
	uuidA := "11111111-1111-4111-8111-111111111111"
	req2, err := http.NewRequestWithContext(context.Background(), http.MethodPost, serverBase+"/items", bytes.NewReader([]byte(`{"name":"item-one"}`)))
	if err != nil {
		t.Fatalf("create request 2 failed: %v", err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(header.HeaderRequestUUID.String(), uuidA)
	req2.Header.Set("X-Request-Id", uuidA)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("request 2 failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp2.StatusCode)
	}
	respBytes2, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("read resp2 body failed: %v", err)
	}
	var env2 envelope.Response
	if err := json.Unmarshal(respBytes2, &env2); err != nil {
		t.Fatalf("unmarshal resp2 failed: %v", err)
	}
	headerUUID2 := resp2.Header.Get(header.HeaderRequestUUID.String())
	if headerUUID2 == "" {
		t.Fatalf("expected non-empty %s header in resp2", header.HeaderRequestUUID)
	}
	if _, err := uuid.Parse(headerUUID2); err != nil {
		t.Fatalf("expected valid UUID in resp2 header, got %q: %v", headerUUID2, err)
	}
	if env2.Context != headerUUID2 {
		t.Fatalf("expected env2.Context == headerUUID2, got context %q != header %q", env2.Context, headerUUID2)
	}
	if env2.Context == uuidA {
		t.Fatalf("expected client uuidA (%q) to be ignored, but server echoed it", uuidA)
	}
	dataMap2, ok := env2.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected env2.Data to be map[string]any, got %T", env2.Data)
	}
	id2, ok := dataMap2["id"].(float64)
	if !ok || id2 <= 0 {
		t.Fatalf("expected id > 0, got %v", dataMap2["id"])
	}
	if dataMap2["count"] != float64(1) {
		t.Fatalf("expected count 1, got %v", dataMap2["count"])
	}
	if dataMap2["cache"] != "val-item-one-1" {
		t.Fatalf("expected cache val-item-one-1, got %v", dataMap2["cache"])
	}

	// 3. 合法请求 2：固定 UUID B（确保不同 UUID 不串且 DB/缓存正确累加）
	uuidB := "22222222-2222-4222-8222-222222222222"
	req3, err := http.NewRequestWithContext(context.Background(), http.MethodPost, serverBase+"/items", bytes.NewReader([]byte(`{"name":"item-two"}`)))
	if err != nil {
		t.Fatalf("create request 3 failed: %v", err)
	}
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set(header.HeaderRequestUUID.String(), uuidB)
	req3.Header.Set("X-Request-Id", uuidB)
	resp3, err := client.Do(req3)
	if err != nil {
		t.Fatalf("request 3 failed: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp3.StatusCode)
	}
	respBytes3, err := io.ReadAll(resp3.Body)
	if err != nil {
		t.Fatalf("read resp3 body failed: %v", err)
	}
	var env3 envelope.Response
	if err := json.Unmarshal(respBytes3, &env3); err != nil {
		t.Fatalf("unmarshal resp3 failed: %v", err)
	}
	headerUUID3 := resp3.Header.Get(header.HeaderRequestUUID.String())
	if headerUUID3 == "" {
		t.Fatalf("expected non-empty %s header in resp3", header.HeaderRequestUUID)
	}
	if _, err := uuid.Parse(headerUUID3); err != nil {
		t.Fatalf("expected valid UUID in resp3 header, got %q: %v", headerUUID3, err)
	}
	if env3.Context != headerUUID3 {
		t.Fatalf("expected env3.Context == headerUUID3, got context %q != header %q", env3.Context, headerUUID3)
	}
	if env3.Context == uuidB {
		t.Fatalf("expected client uuidB (%q) to be ignored, but server echoed it", uuidB)
	}
	if env2.Context == env3.Context {
		t.Fatalf("expected different UUIDs, both were %q", env2.Context)
	}
	dataMap3, ok := env3.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected env3.Data to be map[string]any, got %T", env3.Data)
	}
	id3, ok := dataMap3["id"].(float64)
	if !ok || id3 <= 0 {
		t.Fatalf("expected id > 0, got %v", dataMap3["id"])
	}
	if dataMap3["count"] != float64(2) {
		t.Fatalf("expected count 2, got %v", dataMap3["count"])
	}
	if dataMap3["cache"] != "val-item-two-2" {
		t.Fatalf("expected cache val-item-two-2, got %v", dataMap3["cache"])
	}

	var countAfter int64
	if err := rawSQLDB.QueryRow("SELECT COUNT(*) FROM demo_records;").Scan(&countAfter); err != nil || countAfter != 2 {
		t.Fatalf("expected 2 records in DB, got %d (err: %v)", countAfter, err)
	}

	// 4. 执行有序关机并验证资源真实关闭
	shutdownApp()
	if appRunErr != nil && !errors.Is(appRunErr, context.Canceled) {
		t.Fatalf("app.Run() failed: %v", appRunErr)
	}
	if pingErr := rawSQLDB.Ping(); pingErr == nil {
		t.Errorf("expected rawSQLDB.Ping to fail after shutdown, but it succeeded")
	}
	reqRefused, err := http.NewRequestWithContext(context.Background(), http.MethodGet, serverBase+"/health", nil)
	if err != nil {
		t.Fatalf("create reqRefused failed: %v", err)
	}
	if resp, err := client.Do(reqRefused); err == nil {
		_ = resp.Body.Close()
		t.Errorf("expected http request to fail after shutdown, but got response")
	}
}
