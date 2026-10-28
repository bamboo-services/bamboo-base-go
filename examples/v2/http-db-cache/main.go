//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	cache "github.com/bamboo-services/bamboo-base-go/plugins/cache/v2"
	sqlite "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2"
	database "github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
	webgin "github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DemoItem 数据库示例记录实体。
type DemoItem struct {
	ID   uint   `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"size:64;not null"`
}

func main() {
	app := application.New()

	dbInfo, dbFac := database.New(
		database.WithAdapter(sqlite.New("file:bamboo-example?mode=memory&cache=shared")),
	)
	if err := app.RegisterTool(dbInfo, dbFac); err != nil {
		fmt.Fprintln(os.Stderr, "register database error:", err)
		os.Exit(1)
	}

	cacheInfo, cacheFac := cache.New(cache.WithMemory())
	if err := app.RegisterTool(cacheInfo, cacheFac); err != nil {
		fmt.Fprintln(os.Stderr, "register cache error:", err)
		os.Exit(1)
	}

	webInfo, webFac := webgin.New(
		webgin.WithAddr("127.0.0.1:8080"),
		webgin.WithRequires(ctxkey.DatabaseKey, ctxkey.CacheManagerKey),
		webgin.WithRoute(func(startCtx context.Context, engine *gin.Engine) {
			db, getDBErr := application.Get[*gorm.DB](startCtx, ctxkey.DatabaseKey)
			if getDBErr != nil {
				fmt.Fprintln(os.Stderr, "get db failed in startCtx:", getDBErr)
				return
			}
			mgr, getCacheErr := application.Get[*cache.Manager](startCtx, ctxkey.CacheManagerKey)
			if getCacheErr != nil {
				fmt.Fprintln(os.Stderr, "get cache manager failed in startCtx:", getCacheErr)
				return
			}

			if migrateErr := db.AutoMigrate(&DemoItem{}); migrateErr != nil {
				fmt.Fprintln(os.Stderr, "migrate table error:", migrateErr)
				return
			}

			engine.GET("/example", func(c *gin.Context) {
				reqCtx := c.Request.Context()

				item := DemoItem{Name: "bamboo-item"}
				if dbErr := db.WithContext(reqCtx).Create(&item).Error; dbErr != nil {
					httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to insert db item", dbErr))
					return
				}

				var count int64
				if dbErr := db.WithContext(reqCtx).Model(&DemoItem{}).Count(&count).Error; dbErr != nil {
					httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to count db items", dbErr))
					return
				}

				keyCache := cache.KeyCacheOf[string, string](mgr)
				cacheVal := fmt.Sprintf("total-%d", count)
				if setErr := keyCache.Set(reqCtx, "item_count", &cacheVal); setErr != nil {
					httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to set cache", setErr))
					return
				}

				valPtr, found, getErr := keyCache.Get(reqCtx, "item_count")
				if getErr != nil {
					httpresult.AbortError(c, apperr.New(apperr.ServerInternalError, "failed to get cache", getErr))
					return
				}

				var cachedResult string
				if found && valPtr != nil {
					cachedResult = *valPtr
				}

				httpresult.SuccessWithData(c, "ok", map[string]any{
					"latest_id":    item.ID,
					"total_count":  count,
					"cached_value": cachedResult,
				})
			})
		}),
	)
	if err := app.RegisterWorker(webInfo, webFac); err != nil {
		fmt.Fprintln(os.Stderr, "register webgin error:", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "app run error:", err)
		os.Exit(1)
	}
}
