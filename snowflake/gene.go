package xSnowflake

import "fmt"

// Gene 业务基因类型
//
// 用于标识不同业务类型的数据，嵌入到基因雪花 ID 中。
// 支持 0-63 共 64 种业务类型。
//
// 基因分类:
//   - 系统级别 (0-15): 保留给系统内部使用
//   - 业务级别 (16-63): 供业务自定义使用
type Gene int64

// 系统级别基因常量 (0-15)
//
// 这些基因类型保留给系统内部使用，建议业务扩展时使用 16-63 范围。
const (
	GeneDefault Gene = 0 // 默认/未指定类型
	GeneSystem  Gene = 1 // 系统内部数据
	GeneUser    Gene = 2 // 用户相关数据
	GeneRole    Gene = 3 // 角色权限数据
	GeneLog     Gene = 4 // 日志记录数据
	GeneConfig  Gene = 5 // 配置数据
	GeneFile    Gene = 6 // 文件/附件数据
	GeneSession Gene = 7 // 会话数据
	GeneToken   Gene = 8 // 令牌数据
	GeneCache   Gene = 9 // 缓存数据
)

// 业务级别基因常量 (16-63)
//
// 以下为业务扩展示例，具体项目可根据需求定义自己的业务基因。
// 建议在项目中创建独立的常量文件继承这些基础类型。
const (
	GeneOrder     Gene = 16 // 订单数据
	GeneProduct   Gene = 17 // 商品数据
	GenePayment   Gene = 18 // 支付数据
	GeneInventory Gene = 19 // 库存数据
	GeneCustomer  Gene = 20 // 客户数据
	GeneVendor    Gene = 21 // 供应商数据
	GeneContract  Gene = 22 // 合同数据
	GeneInvoice   Gene = 23 // 发票数据
	GeneShipment  Gene = 24 // 物流数据
	GeneRefund    Gene = 25 // 退款数据
	GeneCoupon    Gene = 26 // 优惠券数据
	GenePromotion Gene = 27 // 促销活动数据
	GeneReview    Gene = 28 // 评价数据
	GeneMessage   Gene = 29 // 消息数据
	GeneNotify    Gene = 30 // 通知数据
	GeneTask      Gene = 31 // 任务数据
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

// String 返回基因类型的字符串描述
//
// 如果是已定义的基因类型，返回对应的名称；
// 否则返回 "Custom(n)" 格式的字符串。
//
// 返回值:
//   - string: 基因类型名称
func (g Gene) String() string {
	if name, ok := geneNames[g]; ok {
		return name
	}
	return fmt.Sprintf("Custom(%d)", g)
}

// IsSystem 判断是否为系统级别基因（0-15）
//
// 返回值:
//   - bool: 如果是系统级别基因返回 true
func (g Gene) IsSystem() bool {
	return g >= 0 && g <= 15
}

// IsBusiness 判断是否为业务级别基因（16-63）
//
// 返回值:
//   - bool: 如果是业务级别基因返回 true
func (g Gene) IsBusiness() bool {
	return g >= 16 && g <= 63
}

// IsValid 判断基因类型是否有效（0-63）
//
// 返回值:
//   - bool: 如果基因类型在有效范围内返回 true
func (g Gene) IsValid() bool {
	return g >= 0 && g <= 63
}

// Int64 返回基因类型的 int64 值
//
// 返回值:
//   - int64: 基因类型的数值
func (g Gene) Int64() int64 {
	return int64(g)
}
