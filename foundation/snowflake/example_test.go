package snowflake_test

import (
	"encoding/json"
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
)

func ExampleNewNode() {
	// 创建合法节点（数据中心 1，节点 2）
	node, err := snowflake.NewNode(1, 2)
	if err != nil {
		fmt.Printf("创建失败: %v\n", err)
		return
	}
	fmt.Printf("数据中心 ID: %d, 节点 ID: %d\n", node.DatacenterID(), node.NodeID())

	// 传入超出合法范围 [0, 7] 的参数将返回错误
	_, err = snowflake.NewNode(8, 0)
	if err != nil {
		fmt.Println("边界校验生效: 参数超出范围")
	}

	// Output:
	// 数据中心 ID: 1, 节点 ID: 2
	// 边界校验生效: 参数超出范围
}

func ExampleNode_Generate() {
	node, err := snowflake.NewNode(2, 3)
	if err != nil {
		panic(err)
	}

	// 生成带有订单业务基因的 ID
	id, err := node.Generate(snowflake.GeneOrder)
	if err != nil {
		panic(err)
	}

	// 验证从生成 ID 中提取出的位段信息（基因、数据中心、节点 ID 是完全确定的）
	fmt.Println("业务基因:", id.Gene())
	fmt.Println("数据中心 ID:", id.DatacenterID())
	fmt.Println("节点 ID:", id.NodeID())

	// Output:
	// 业务基因: Order
	// 数据中心 ID: 2
	// 节点 ID: 3
}

func ExampleSnowflakeID_MarshalJSON() {
	// 构造一个确定性雪花 ID
	id := snowflake.SnowflakeID(1234567890123456789)

	data, err := json.Marshal(id)
	if err != nil {
		panic(err)
	}

	// 输出为带双引号的 JSON 字符串，防止前端 JavaScript 丢失精度
	fmt.Println(string(data))

	// Output:
	// "1234567890123456789"
}

func ExampleSnowflakeID_UnmarshalJSON() {
	var fromString snowflake.SnowflakeID
	var fromNumber snowflake.SnowflakeID

	// 支持字符串格式
	if err := json.Unmarshal([]byte(`"9876543210"`), &fromString); err != nil {
		panic(err)
	}

	// 向前兼容纯数字格式
	if err := json.Unmarshal([]byte(`9876543210`), &fromNumber); err != nil {
		panic(err)
	}

	fmt.Printf("字符串解析结果: %d\n", fromString.Int64())
	fmt.Printf("数字解析结果: %d\n", fromNumber.Int64())

	// 非法内容解析失败且不污染接收器
	invalidData := []byte(`"not-a-number"`)
	if err := fromString.UnmarshalJSON(invalidData); err != nil {
		fmt.Println("格式错误时拦截成功")
	}
	fmt.Printf("原有值未受污染: %d\n", fromString.Int64())

	// Output:
	// 字符串解析结果: 9876543210
	// 数字解析结果: 9876543210
	// 格式错误时拦截成功
	// 原有值未受污染: 9876543210
}

func ExampleSnowflakeID_Gene() {
	// 构造已知位段的确定性雪花 ID：
	// 时间戳偏移 1000ms, 业务基因 16(GeneOrder), 数据中心 2, 节点 3, 序列号 4
	// (1000 << 22) | (16 << 16) | (2 << 13) | (3 << 10) | 4 = 4195372036
	const sampleID = snowflake.SnowflakeID(4195372036)

	fmt.Println("业务基因:", sampleID.Gene())
	fmt.Println("数据中心 ID:", sampleID.DatacenterID())
	fmt.Println("节点 ID:", sampleID.NodeID())
	fmt.Println("序列号:", sampleID.Sequence())

	// Output:
	// 业务基因: Order
	// 数据中心 ID: 2
	// 节点 ID: 3
	// 序列号: 4
}

func ExampleCalcGene() {
	calc := snowflake.CalcGene()

	// 零值 ID 返回默认基因 GeneDefault (0)
	zeroGene := calc.Hash(0)
	fmt.Println("零值 ID 基因:", zeroGene)

	// 空字符串返回默认基因 GeneDefault (0)
	emptyGene := calc.HashString("")
	fmt.Println("空字符串基因:", emptyGene)

	// 非空字符串计算输出合法 6 位基因（范围 [0, 63]）
	orderGene := calc.HashString("order-service")
	fmt.Printf("哈希基因有效性: %t\n", orderGene.IsValid())

	// Output:
	// 零值 ID 基因: Default
	// 空字符串基因: Default
	// 哈希基因有效性: true
}

func ExampleGene() {
	userGene := snowflake.GeneUser
	orderGene := snowflake.GeneOrder
	customGene := snowflake.Gene(50)

	fmt.Printf("用户基因: %s (系统级: %t, 业务级: %t)\n", userGene, userGene.IsSystem(), userGene.IsBusiness())
	fmt.Printf("订单基因: %s (系统级: %t, 业务级: %t)\n", orderGene, orderGene.IsSystem(), orderGene.IsBusiness())
	fmt.Printf("自定义基因: %s (有效性: %t)\n", customGene, customGene.IsValid())

	// Output:
	// 用户基因: User (系统级: true, 业务级: false)
	// 订单基因: Order (系统级: false, 业务级: true)
	// 自定义基因: Custom(50) (有效性: true)
}
