package models_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/models"
)

// User 展示如何将 models.BaseEntity 嵌入业务数据实体。
type User struct {
	models.BaseEntity
	Username string
}

// Order 展示如何实现 models.GeneProvider 自定义业务基因。
type Order struct {
	models.BaseEntityWithSoftDelete
	OrderNo string
}

func (o *Order) GetGene() snowflake.Gene {
	return snowflake.GeneOrder
}

func ExamplePageRequest_Offset() {
	// 构造分页请求：第 2 页，每页 15 条
	req := models.PageRequest{
		Page: 2,
		Size: 15,
		Sort: models.SortDesc,
	}

	// 计算数据库分页的 Offset 与 Limit
	fmt.Println("Offset:", req.Offset())
	fmt.Println("Limit:", req.Limit())
	fmt.Println("OrderDirection:", req.OrderDirection())

	// Output:
	// Offset: 15
	// Limit: 15
	// OrderDirection: desc
}

func ExamplePageRequest_Normalize() {
	// 当请求参数非法（page <= 0, size <= 0, 非法排序）时自动修正为默认值
	req := models.PageRequest{
		Page: 0,
		Size: -5,
		Sort: "invalid",
	}

	normalized := req.Normalize()
	fmt.Println("Page:", normalized.Page)
	fmt.Println("Size:", normalized.Size)
	fmt.Println("Sort:", normalized.Sort)

	// Output:
	// Page: 1
	// Size: 20
	// Sort: asc
}

func ExampleNewPage() {
	// 模拟当前页数据
	items := []string{"item1", "item2"}

	// 构造第 1 页，每页 10 条，总条目数为 25 的分页响应
	resp := models.NewPage(1, 10, 25, items)

	fmt.Println("CurrentPage:", resp.CurrentPage)
	fmt.Println("TotalPages:", resp.TotalPages)
	fmt.Println("TotalItems:", resp.TotalItems)
	fmt.Println("Size:", resp.Size)
	fmt.Println("ItemsCount:", len(resp.Items))

	// Output:
	// CurrentPage: 1
	// TotalPages: 3
	// TotalItems: 25
	// Size: 10
	// ItemsCount: 2
}
