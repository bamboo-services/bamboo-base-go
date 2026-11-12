package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/models"
	"gorm.io/gorm"
)

// gormBaseEntity 是嵌入通用实体基类的真实持久化测试模型。
type gormBaseEntity struct {
	models.BaseEntity
	Name string `gorm:"size:128"`
}

// gormSoftDeleteEntity 是嵌入带软删除实体基类的真实持久化测试模型。
type gormSoftDeleteEntity struct {
	models.BaseEntityWithSoftDelete
	Name string `gorm:"size:128"`
}

// invalidGeneGORMEntity 通过公开 GeneProvider 契约触发发号器范围失败。
type invalidGeneGORMEntity struct {
	models.BaseEntity
}

func (*invalidGeneGORMEntity) GetGene() snowflake.Gene {
	return snowflake.Gene(99)
}

// snowflakeTool 是向容器暴露雪花发号节点的测试插件。
type snowflakeTool struct {
	node *snowflake.Node
}

// Start 实现 application.Plugin 生命周期钩子。
func (s *snowflakeTool) Start(ctx context.Context) *apperr.Error {
	return nil
}

// Stop 实现 application.Plugin 生命周期钩子。
func (s *snowflakeTool) Stop(ctx context.Context) *apperr.Error {
	return nil
}

// Expose 暴露雪花发号节点。
func (s *snowflakeTool) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		ctxkey.SnowflakeNodeKey: s.node,
	}
}

// containerWorker 是持有运行期容器视图上下文的测试后台服务。
type containerWorker struct {
	ctxChan chan<- context.Context
}

// Start 实现 application.RunnablePlugin 生命周期钩子。
func (c *containerWorker) Start(ctx context.Context) *apperr.Error {
	return nil
}

// Stop 实现 application.RunnablePlugin 生命周期钩子。
func (c *containerWorker) Stop(ctx context.Context) *apperr.Error {
	return nil
}

// Run 阻塞等待停机信号，并向测试暴露运行期容器视图上下文。
func (c *containerWorker) Run(ctx context.Context) *apperr.Error {
	select {
	case c.ctxChan <- ctx:
	default:
	}
	<-ctx.Done()
	return nil
}

// runWithSnowflakeContext 启动一个暴露雪花发号节点的应用，
// 并在应用运行期容器视图上下文中执行测试逻辑。
func runWithSnowflakeContext(t *testing.T, node *snowflake.Node, testFn func(ctx context.Context)) {
	t.Helper()
	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	toolKey := ctxkey.Key("sqlite_test_tool_snowflake")
	workerKey := ctxkey.Key("sqlite_test_worker")

	if err := app.RegisterTool(application.PluginInfo{
		Key:      toolKey,
		Provides: []ctxkey.Key{ctxkey.SnowflakeNodeKey},
	}, func() application.Plugin {
		return &snowflakeTool{node: node}
	}); err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	workerCtxChan := make(chan context.Context, 1)
	if err := app.RegisterWorker(application.PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{ctxkey.SnowflakeNodeKey},
	}, func() application.RunnablePlugin {
		return &containerWorker{ctxChan: workerCtxChan}
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

// openRealSQLiteDB 基于真实 SQLite 文件打开数据库连接，并注册两个模型的表结构。
func openRealSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "models_gorm_test.db")

	adp := sqlite.New(dbFile)
	dialector, openErr := adp.Open()
	if openErr != nil {
		t.Fatalf("Open 失败: %v", openErr)
	}

	db, gormErr := gorm.Open(dialector, &gorm.Config{})
	if gormErr != nil {
		t.Fatalf("gorm.Open 失败: %v", gormErr)
	}

	sqlDB, sqlErr := db.DB()
	if sqlErr != nil {
		t.Fatalf("获取底层 sql.DB 失败: %v", sqlErr)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if migrateErr := db.AutoMigrate(&gormBaseEntity{}, &gormSoftDeleteEntity{}); migrateErr != nil {
		t.Fatalf("AutoMigrate 失败: %v", migrateErr)
	}
	return db
}

// TestModels_RealSQLite_CreateInvokesBeforeCreateHook 验证真实 SQLite 写入路径上
// BeforeCreate 钩子确实被调用：ID 由钩子生成并落库，时间戳由钩子写入。
func TestModels_RealSQLite_CreateInvokesBeforeCreateHook(t *testing.T) {
	node, nodeErr := snowflake.NewNode(1, 2)
	if nodeErr != nil {
		t.Fatalf("failed to create node: %v", nodeErr)
	}

	db := openRealSQLiteDB(t)
	before := time.Now()

	runWithSnowflakeContext(t, node, func(ctx context.Context) {
		base := &gormBaseEntity{Name: "base"}
		if createErr := db.WithContext(ctx).Create(base).Error; createErr != nil {
			t.Fatalf("Create 失败: %v", createErr)
		}
		if base.ID.IsZero() {
			t.Fatal("期望钩子生成非零 ID")
		}
		// 雪花 ID 可反解出时间戳，若 ID 来自 GORM 自增主键则无法通过该校验。
		if base.ID.Gene() != snowflake.GeneDefault {
			t.Fatalf("期望业务基因为 %v, 实际为 %v", snowflake.GeneDefault, base.ID.Gene())
		}
		idTime := base.ID.Timestamp()
		if idTime.Before(before.Add(-time.Minute)) || idTime.After(time.Now().Add(time.Minute)) {
			t.Fatalf("ID 反解时间戳 %v 超出合理区间", idTime)
		}
		if base.CreatedAt.IsZero() || base.UpdatedAt.IsZero() {
			t.Fatalf("期望钩子写入时间戳: created=%v updated=%v", base.CreatedAt, base.UpdatedAt)
		}

		var persisted gormBaseEntity
		if firstErr := db.WithContext(ctx).First(&persisted, "name = ?", "base").Error; firstErr != nil {
			t.Fatalf("First 失败: %v", firstErr)
		}
		if persisted.ID != base.ID {
			t.Fatalf("落库 ID = %d, 期望 %d", persisted.ID, base.ID)
		}

		soft := &gormSoftDeleteEntity{Name: "soft"}
		if createErr := db.WithContext(ctx).Create(soft).Error; createErr != nil {
			t.Fatalf("Create 失败: %v", createErr)
		}
		if soft.ID.IsZero() {
			t.Fatal("期望钩子生成非零 ID")
		}
		if soft.ID.Gene() != snowflake.GeneDefault {
			t.Fatalf("期望业务基因为 %v, 实际为 %v", snowflake.GeneDefault, soft.ID.Gene())
		}
		if soft.CreatedAt.IsZero() || soft.UpdatedAt.IsZero() {
			t.Fatalf("期望钩子写入时间戳: created=%v updated=%v", soft.CreatedAt, soft.UpdatedAt)
		}

		var persistedSoft gormSoftDeleteEntity
		if firstErr := db.WithContext(ctx).First(&persistedSoft, "name = ?", "soft").Error; firstErr != nil {
			t.Fatalf("First 失败: %v", firstErr)
		}
		if persistedSoft.ID != soft.ID {
			t.Fatalf("落库 ID = %d, 期望 %d", persistedSoft.ID, soft.ID)
		}
	})
}

// TestModels_RealSQLite_UpdateInvokesBeforeUpdateHook 验证真实 SQLite 更新路径上
// BeforeUpdate 钩子确实被调用。
//
// 观测手段：在 GORM 更新回调链的 "gorm:update" 之前注册观察回调，
// 直接读取模型实体上被模型钩子写回的 UpdatedAt。
// 该位置既晚于模型钩子（"gorm:before_update" 内执行），又早于 GORM 自身的
// autoUpdateTime 赋值（发生在 "gorm:update" 内），是唯一能区分
// “模型钩子写回”与“框架自动维护”的观察点。
// SkipHooks 会话作为对照组：模型钩子被跳过时，UpdatedAt 保持哨兵值不变。
func TestModels_RealSQLite_UpdateInvokesBeforeUpdateHook(t *testing.T) {
	db := openRealSQLiteDB(t)
	sentinel := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	seed := func(t *testing.T, name string, id snowflake.SnowflakeID) gormBaseEntity {
		t.Helper()
		ent := gormBaseEntity{
			BaseEntity: models.BaseEntity{ID: id},
			Name:       name,
		}
		if createErr := db.Create(&ent).Error; createErr != nil {
			t.Fatalf("Create 失败: %v", createErr)
		}
		return ent
	}

	withHooks := seed(t, "with_hooks", snowflake.SnowflakeID(1000))
	skipHooks := seed(t, "skip_hooks", snowflake.SnowflakeID(1001))

	// 观察回调通过通道回传模型钩子执行后实体上的 UpdatedAt。
	observed := make(chan time.Time, 4)
	if regErr := db.Callback().Update().Before("gorm:update").
		Register("test:observe_before_update", func(tx *gorm.DB) {
			switch model := tx.Statement.Model.(type) {
			case *gormBaseEntity:
				observed <- model.UpdatedAt
			case *gormSoftDeleteEntity:
				observed <- model.UpdatedAt
			}
		}); regErr != nil {
		t.Fatalf("注册观察回调失败: %v", regErr)
	}

	// 对照组：SkipHooks 会话下模型钩子被跳过，UpdatedAt 保持哨兵值。
	skipHooks.UpdatedAt = sentinel
	if updateErr := db.Model(&skipHooks).Session(&gorm.Session{SkipHooks: true}).
		Update("name", "skip_hooks_updated").Error; updateErr != nil {
		t.Fatalf("Update 失败: %v", updateErr)
	}
	select {
	case got := <-observed:
		if got.Year() != sentinel.Year() {
			t.Fatalf("SkipHooks 会话下模型钩子不应执行，UpdatedAt = %v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for update observation")
	}

	// 实验组：模型钩子必须把 UpdatedAt 刷新为当前时间。
	withHooks.UpdatedAt = sentinel
	if updateErr := db.Model(&withHooks).Update("name", "with_hooks_updated").Error; updateErr != nil {
		t.Fatalf("Update 失败: %v", updateErr)
	}
	select {
	case got := <-observed:
		if got.Year() == sentinel.Year() {
			t.Fatalf("期望 BeforeUpdate 钩子刷新时间戳，实际仍为哨兵值 %v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for update observation")
	}

	var found gormBaseEntity
	if firstErr := db.First(&found, "id = ?", withHooks.ID).Error; firstErr != nil {
		t.Fatalf("First 失败: %v", firstErr)
	}
	if found.Name != "with_hooks_updated" {
		t.Fatalf("更新未落库: name = %q", found.Name)
	}
	if found.UpdatedAt.Year() == sentinel.Year() {
		t.Fatalf("期望落库时间戳被刷新，实际仍为哨兵值 %v", found.UpdatedAt)
	}
}

// TestModels_RealSQLite_SoftDeleteUpdateInvokesBeforeUpdateHook 验证带软删除基类的模型
// 在真实 SQLite 更新路径上同样执行 BeforeUpdate 钩子。
func TestModels_RealSQLite_SoftDeleteUpdateInvokesBeforeUpdateHook(t *testing.T) {
	db := openRealSQLiteDB(t)
	sentinel := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	ent := gormSoftDeleteEntity{
		BaseEntityWithSoftDelete: models.BaseEntityWithSoftDelete{ID: snowflake.SnowflakeID(2000)},
		Name:                     "soft_update",
	}
	if createErr := db.Create(&ent).Error; createErr != nil {
		t.Fatalf("Create 失败: %v", createErr)
	}

	observed := make(chan time.Time, 2)
	if regErr := db.Callback().Update().Before("gorm:update").
		Register("test:observe_before_update_soft", func(tx *gorm.DB) {
			if model, ok := tx.Statement.Model.(*gormSoftDeleteEntity); ok {
				observed <- model.UpdatedAt
			}
		}); regErr != nil {
		t.Fatalf("注册观察回调失败: %v", regErr)
	}

	ent.UpdatedAt = sentinel
	if updateErr := db.Model(&ent).Update("name", "soft_updated").Error; updateErr != nil {
		t.Fatalf("Update 失败: %v", updateErr)
	}
	select {
	case got := <-observed:
		if got.Year() == sentinel.Year() {
			t.Fatalf("期望 BeforeUpdate 钩子刷新时间戳，实际仍为哨兵值 %v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for update observation")
	}

	var found gormSoftDeleteEntity
	if firstErr := db.First(&found, "id = ?", ent.ID).Error; firstErr != nil {
		t.Fatalf("First 失败: %v", firstErr)
	}
	if found.Name != "soft_updated" {
		t.Fatalf("更新未落库: name = %q", found.Name)
	}
}

// TestModels_RealSQLite_PreservesNonZeroID 验证非零 ID 在真实写入路径上被完整保留，
// 且不依赖雪花节点上下文。
func TestModels_RealSQLite_PreservesNonZeroID(t *testing.T) {
	db := openRealSQLiteDB(t)

	base := gormBaseEntity{
		BaseEntity: models.BaseEntity{ID: snowflake.SnowflakeID(424242)},
		Name:       "explicit",
	}
	if createErr := db.Create(&base).Error; createErr != nil {
		t.Fatalf("Create 失败: %v", createErr)
	}
	if base.ID != snowflake.SnowflakeID(424242) {
		t.Fatalf("期望保留 ID 424242，实际为 %d", base.ID)
	}

	var found gormBaseEntity
	if firstErr := db.First(&found, "id = ?", snowflake.SnowflakeID(424242)).Error; firstErr != nil {
		t.Fatalf("First 失败: %v", firstErr)
	}
	if found.ID != snowflake.SnowflakeID(424242) {
		t.Fatalf("落库 ID = %d, 期望 424242", found.ID)
	}

	soft := gormSoftDeleteEntity{
		BaseEntityWithSoftDelete: models.BaseEntityWithSoftDelete{ID: snowflake.SnowflakeID(525252)},
		Name:                     "explicit_soft",
	}
	if createErr := db.Create(&soft).Error; createErr != nil {
		t.Fatalf("Create 失败: %v", createErr)
	}
	if soft.ID != snowflake.SnowflakeID(525252) {
		t.Fatalf("期望保留 ID 525252，实际为 %d", soft.ID)
	}
}

// TestModels_RealSQLite_FailedCreateWritesNoRow 验证钩子失败时写入被整体拒绝，
// 不会留下半成品记录。
func TestModels_RealSQLite_FailedCreateWritesNoRow(t *testing.T) {
	db := openRealSQLiteDB(t)

	ent := &gormBaseEntity{Name: "should_not_exist"}
	createErr := db.WithContext(context.Background()).Create(ent).Error
	if createErr == nil {
		t.Fatal("期望创建失败，实际返回 nil")
	}

	var appErr *apperr.Error
	if !errors.As(createErr, &appErr) {
		t.Fatalf("errors.As 未能提取 *apperr.Error: %v", createErr)
	}
	if !ent.ID.IsZero() {
		t.Fatalf("失败路径不应写入 ID，实际为 %d", ent.ID)
	}

	var count int64
	if countErr := db.Model(&gormBaseEntity{}).Count(&count).Error; countErr != nil {
		t.Fatalf("Count 失败: %v", countErr)
	}
	if count != 0 {
		t.Fatalf("期望未写入任何记录，实际写入 %d 条", count)
	}
}

// TestModels_RealSQLite_InvalidGenePreservesCause 验证发号错误经过真实GORM钩子仍保原码且不落库。
func TestModels_RealSQLite_InvalidGenePreservesCause(t *testing.T) {
	db := openRealSQLiteDB(t)
	if err := db.AutoMigrate(&invalidGeneGORMEntity{}); err != nil {
		t.Fatal(err)
	}
	node, nodeErr := snowflake.NewNode(1, 2)
	if nodeErr != nil {
		t.Fatal(nodeErr)
	}
	runWithSnowflakeContext(t, node, func(ctx context.Context) {
		entity := &invalidGeneGORMEntity{}
		err := db.WithContext(ctx).Create(entity).Error
		var appErr *apperr.Error
		if !errors.As(err, &appErr) || appErr == nil || appErr.Code != *valueerr.OutOfRange || appErr.Err == nil {
			t.Fatalf("发号错误被丢失或重新归类: %v", err)
		}
		if !entity.ID.IsZero() {
			t.Fatalf("失败时写回了ID: %v", entity.ID)
		}
		var count int64
		if countErr := db.Model(&invalidGeneGORMEntity{}).Count(&count).Error; countErr != nil {
			t.Fatal(countErr)
		}
		if count != 0 {
			t.Fatalf("发号失败后存在半成品记录: %d", count)
		}
	})
}
