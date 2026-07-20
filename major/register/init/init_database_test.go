package xInit_test

import (
	"context"
	"testing"

	"gorm.io/gorm"

	xOption "github.com/bamboo-services/bamboo-base-go/major/option"
	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
	xInit "github.com/bamboo-services/bamboo-base-go/major/register/init"
)

// TestEntity 测试用实体，用于验证 AutoMigrate 建表和 Prepare 数据插入。
type TestEntity struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:100"`
}

func TestDatabaseInit_AutoMigrateAndPrepare(t *testing.T) {
	// 通过 option.Apply + WithDatabase 构造 DatabaseConfig，完整模拟 Runner 装配链路。
	// SQLite 内存数据库 + 迁移 TestEntity 表 + 插入一条种子数据。
	cfg := xOption.Apply(
		xOption.WithDatabase(
			xOptDatabase.SQLite(":memory:"),
			xOptDatabase.WithAutoMigrate(&TestEntity{}),
			xOptDatabase.WithPrepare(xOptDatabase.PrepareFunc(func(ctx context.Context, db *gorm.DB) error {
				return db.Create(&TestEntity{Name: "seed"}).Error
			})),
		),
	).Database()

	node := xInit.DatabaseInit(cfg)
	result, err := node(context.Background())
	if err != nil {
		t.Fatalf("DatabaseInit 失败: %v", err)
	}

	db, ok := result.(*gorm.DB)
	if !ok {
		t.Fatalf("返回类型非 *gorm.DB: %T", result)
	}

	// 验证表已存在（AutoMigrate 成功）
	if !db.Migrator().HasTable(&TestEntity{}) {
		t.Error("AutoMigrate 后表未创建")
	}

	// 验证种子数据存在（Prepare 成功）
	var entity TestEntity
	if err := db.Where("name = ?", "seed").First(&entity).Error; err != nil {
		t.Errorf("种子数据未找到: %v", err)
	}
	if entity.Name != "seed" {
		t.Errorf("种子数据 Name 不匹配: got=%q want=%q", entity.Name, "seed")
	}

	t.Log("✅ AutoMigrate 建表 + Prepare 种子数据全部通过")
}

func TestDatabaseInit_MultiPrepare(t *testing.T) {
	// 验证 WithPrepare 可变参数：一次传入多个回调，两条种子数据均应写入。
	cfg := xOption.Apply(
		xOption.WithDatabase(
			xOptDatabase.SQLite(":memory:"),
			xOptDatabase.WithAutoMigrate(&TestEntity{}),
			xOptDatabase.WithPrepare(
				xOptDatabase.PrepareFunc(func(ctx context.Context, db *gorm.DB) error {
					return db.Create(&TestEntity{Name: "seed-a"}).Error
				}),
				xOptDatabase.PrepareFunc(func(ctx context.Context, db *gorm.DB) error {
					return db.Create(&TestEntity{Name: "seed-b"}).Error
				}),
			),
		),
	).Database()

	node := xInit.DatabaseInit(cfg)
	result, err := node(context.Background())
	if err != nil {
		t.Fatalf("DatabaseInit 失败: %v", err)
	}

	db, ok := result.(*gorm.DB)
	if !ok {
		t.Fatalf("返回类型非 *gorm.DB: %T", result)
	}

	// 验证两条种子数据均已写入
	var count int64
	db.Model(&TestEntity{}).Where("name IN ?", []string{"seed-a", "seed-b"}).Count(&count)
	if count != 2 {
		t.Errorf("期望 2 条种子数据，实际 %d 条", count)
	}

	t.Log("✅ WithPrepare 可变参数多回调均生效")
}
