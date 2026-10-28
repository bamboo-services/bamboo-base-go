package snowflake

import "fmt"

// Gene 代表嵌入雪花 ID 中的业务基因强类型（底层为 int64，有效取值 0~63）。
//
// 基因设计用于在无额外数据库查询的情况下识别全局 ID 的业务属性，并支持分库分表路由。
// 按取值划分为两个主要区间：
//   - 系统级别（0 至 15）：框架内部及公共基座保留使用；
//   - 业务级别（16 至 63）：业务系统自主定义及扩展使用。
//
// 零值 Gene(0) 等价于 [GeneDefault]，属于系统级别。
type Gene int64

// 系统级基因常量（取值 0 至 15），保留给框架内核及公共服务使用。
const (
	GeneDefault Gene = 0 // GeneDefault 表示默认或未指定的业务类型。
	GeneSystem  Gene = 1 // GeneSystem 表示系统内部数据。
	GeneUser    Gene = 2 // GeneUser 表示用户相关数据。
	GeneRole    Gene = 3 // GeneRole 表示角色与权限数据。
	GeneLog     Gene = 4 // GeneLog 表示日志记录数据。
	GeneConfig  Gene = 5 // GeneConfig 表示配置元数据。
	GeneFile    Gene = 6 // GeneFile 表示文件与附件数据。
	GeneSession Gene = 7 // GeneSession 表示用户会话数据。
	GeneToken   Gene = 8 // GeneToken 表示认证令牌数据。
	GeneCache   Gene = 9 // GeneCache 表示缓存与临时数据。
)

// 业务级基因常量（取值 16 至 63），供各业务领域按需引用或在项目中继承扩展。
const (
	GeneOrder     Gene = 16 // GeneOrder 表示订单业务数据。
	GeneProduct   Gene = 17 // GeneProduct 表示商品业务数据。
	GenePayment   Gene = 18 // GenePayment 表示支付结算数据。
	GeneInventory Gene = 19 // GeneInventory 表示库存仓储数据。
	GeneCustomer  Gene = 20 // GeneCustomer 表示客户关系数据。
	GeneVendor    Gene = 21 // GeneVendor 表示供应商数据。
	GeneContract  Gene = 22 // GeneContract 表示合同协议数据。
	GeneInvoice   Gene = 23 // GeneInvoice 表示发票与税务数据。
	GeneShipment  Gene = 24 // GeneShipment 表示物流履约数据。
	GeneRefund    Gene = 25 // GeneRefund 表示售后退款数据。
	GeneCoupon    Gene = 26 // GeneCoupon 表示优惠券数据。
	GenePromotion Gene = 27 // GenePromotion 表示营销活动数据。
	GeneReview    Gene = 28 // GeneReview 表示用户评价数据。
	GeneMessage   Gene = 29 // GeneMessage 表示站内消息数据。
	GeneNotify    Gene = 30 // GeneNotify 表示系统通知数据。
	GeneTask      Gene = 31 // GeneTask 表示异步作业与任务数据。
)

// geneNames 基因类型名称映射
var geneNames = map[Gene]string{
	GeneDefault:   "Default",
	GeneSystem:    "System",
	GeneUser:      "User",
	GeneRole:      "Role",
	GeneLog:       "Log",
	GeneConfig:    "Config",
	GeneFile:      "File",
	GeneSession:   "Session",
	GeneToken:     "Token",
	GeneCache:     "Cache",
	GeneOrder:     "Order",
	GeneProduct:   "Product",
	GenePayment:   "Payment",
	GeneInventory: "Inventory",
	GeneCustomer:  "Customer",
	GeneVendor:    "Vendor",
	GeneContract:  "Contract",
	GeneInvoice:   "Invoice",
	GeneShipment:  "Shipment",
	GeneRefund:    "Refund",
	GeneCoupon:    "Coupon",
	GenePromotion: "Promotion",
	GeneReview:    "Review",
	GeneMessage:   "Message",
	GeneNotify:    "Notify",
	GeneTask:      "Task",
}

// String 返回业务基因的名称。
//
// 预定义的系统级与业务级常量将返回清晰的英文名称；
// 未预定义的自定义或扩展基因将返回 "Custom(n)" 格式的文本。
func (g Gene) String() string {
	if name, ok := geneNames[g]; ok {
		return name
	}
	return fmt.Sprintf("Custom(%d)", g)
}

// IsSystem 报告业务基因是否属于系统保留级别（取值范围 0 至 15）。
func (g Gene) IsSystem() bool {
	return g >= 0 && g <= 15
}

// IsBusiness 报告业务基因是否属于业务自定义级别（取值范围 16 至 63）。
func (g Gene) IsBusiness() bool {
	return g >= 16 && g <= 63
}

// IsValid 报告业务基因数值是否在 6 位位宽的合法范围 [0, 63] 内。
func (g Gene) IsValid() bool {
	return g >= 0 && g <= 63
}

// Int64 将业务基因显式转换为原生的 int64 数值。
func (g Gene) Int64() int64 {
	return int64(g)
}
