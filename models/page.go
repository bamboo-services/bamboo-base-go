package xModels

import "strings"

// PageSort 表示分页排序方向。
//
// 当前仅支持 asc 和 desc 两个值。
type PageSort string

const (
	// SortAsc 表示升序排序。
	SortAsc PageSort = "asc"
	// SortDesc 表示降序排序。
	SortDesc PageSort = "desc"
)

const (
	// DefaultPageNumber 是默认页码，按约定从 1 开始。
	DefaultPageNumber int64 = 1
	// DefaultPageSize 是默认每页条目数。
	DefaultPageSize int64 = 20
	// DefaultPageMaxSize 是默认允许的最大每页条目数。
	DefaultPageMaxSize int64 = 200
)

// PageConfig 定义分页规范化配置。
//
// 该配置用于统一控制：
//  1. 默认页码
//  2. 默认每页数量
//  3. 每页上限
//  4. 默认排序方向
//
// 建议在服务启动阶段按业务场景配置后复用，避免在业务代码中分散硬编码。
type PageConfig struct {
	DefaultPage int64
	DefaultSize int64
	MaxSize     int64
	DefaultSort PageSort
}

// DefaultPageConfig 是 SDK 默认分页配置。
//
// 该值是可变变量。若需要全局调整分页默认行为，可在初始化阶段覆盖。
var DefaultPageConfig = PageConfig{
	DefaultPage: DefaultPageNumber,
	DefaultSize: DefaultPageSize,
	MaxSize:     DefaultPageMaxSize,
	DefaultSort: SortAsc,
}

// PageRequest 表示分页请求参数。
//
// 该结构体可直接用于 gin 的 query/form/json 绑定与校验。
// 推荐在真正使用前调用 Normalize 或 NormalizeWithConfig，
// 将非法输入（如 page<=0、size<=0、sort 非法）统一修正为可用值。
type PageRequest struct {
	Page int64    `json:"page" form:"page" binding:"omitempty,min=1" label:"页码" description:"当前页码，从 1 开始计数。"`
	Size int64    `json:"size" form:"size" binding:"omitempty,min=1,max=200" label:"每页条目数" description:"每页包含的条目数，最大为 200。"`
	Sort PageSort `json:"sort" form:"sort" binding:"omitempty,enum_string=asc desc" label:"排序方式" description:"排序方式，asc 表示升序，desc 表示降序。"`
}

// PageProvider 定义分页参数提供者接口。
//
// 当业务请求结构体实现该接口时，可通过 GetPageRequest 统一抽取分页参数。
type PageProvider interface {
	// GetPageSettings 返回当前请求的分页设置。
	GetPageSettings() PageRequest
}

// DefaultPageRequest 返回默认分页请求参数。
//
// 返回值来自 DefaultPageConfig，适合作为兜底参数使用。
func DefaultPageRequest() PageRequest {
	return PageRequest{
		Page: DefaultPageConfig.DefaultPage,
		Size: DefaultPageConfig.DefaultSize,
		Sort: DefaultPageConfig.DefaultSort,
	}
}

// GetPageRequest 从任意请求对象中提取分页参数。
//
// 若 req 实现了 PageProvider，则使用其 GetPageSettings 结果并执行 Normalize。
// 若未实现，则返回 DefaultPageRequest 作为兜底。
func GetPageRequest[T any](req T) PageRequest {
	if provider, ok := any(req).(PageProvider); ok {
		return provider.GetPageSettings().Normalize()
	}

	return DefaultPageRequest()
}

// Normalize 使用 DefaultPageConfig 规范化分页请求。
//
// 该方法会保证返回值可直接用于分页查询，不会出现非法页码或非法 size。
func (r PageRequest) Normalize() PageRequest {
	return r.NormalizeWithConfig(DefaultPageConfig)
}

// NormalizeWithConfig 使用指定配置规范化分页请求。
//
// 规范化规则：
//  1. page <= 0 时使用 config.DefaultPage
//  2. size <= 0 时使用 config.DefaultSize
//  3. size > config.MaxSize 时截断到 config.MaxSize
//  4. sort 非 asc/desc 时回退到 config.DefaultSort
func (r PageRequest) NormalizeWithConfig(config PageConfig) PageRequest {
	cfg := config.normalize()

	page := r.Page
	if page <= 0 {
		page = cfg.DefaultPage
	}

	size := r.Size
	if size <= 0 {
		size = cfg.DefaultSize
	}
	if size > cfg.MaxSize {
		size = cfg.MaxSize
	}

	return PageRequest{
		Page: page,
		Size: size,
		Sort: normalizePageSort(r.Sort, cfg.DefaultSort),
	}
}

// Offset 返回数据库分页查询偏移量。
//
// 该方法内部会先执行 Normalize，再按 (page-1)*size 计算。
func (r PageRequest) Offset() int64 {
	normalized := r.Normalize()
	return (normalized.Page - 1) * normalized.Size
}

// Limit 返回数据库分页查询条数。
//
// 该方法内部会先执行 Normalize，确保返回值始终为合法正数。
func (r PageRequest) Limit() int64 {
	return r.Normalize().Size
}

// OrderDirection 返回标准化后的排序方向字符串。
//
// 返回值严格为 "asc" 或 "desc"，可直接拼接到 SQL/GORM 的排序表达式中。
func (r PageRequest) OrderDirection() string {
	if r.Normalize().Sort == SortDesc {
		return string(SortDesc)
	}

	return string(SortAsc)
}

// PageResponse 是通用分页响应结构。
//
// T 通常为切片类型（例如 []UserDTO），也可以是任意分页数据容器。
type PageResponse[T any] struct {
	CurrentPage int64 `json:"current_page" label:"当前页码" description:"当前页码，从 1 开始计数。"`
	TotalPages  int64 `json:"total_pages" label:"总页数" description:"总页数，根据总条目数和每页条目数计算得出。"`
	TotalItems  int64 `json:"total_items" label:"总条目数" description:"列表总条目数。"`
	Size        int64 `json:"size" label:"每页条目数" description:"每页包含的条目数。"`
	Items       T     `json:"items" label:"当前页条目" description:"当前页包含的条目数据。"`
}

// NewPage 构造分页响应。
//
// 参数说明：
//   - currentPage: 当前页码
//   - perPage: 每页条目数
//   - totalItems: 总条目数
//   - items: 当前页数据
//
// 该方法会自动处理非法输入：
//   - currentPage/perPage 非法时回退默认值
//   - totalItems 为负时按 0 处理
func NewPage[T any](currentPage, perPage, totalItems int64, items T) *PageResponse[T] {
	currentPage, perPage = normalizePageMeta(currentPage, perPage)

	return &PageResponse[T]{
		CurrentPage: currentPage,
		TotalPages:  calcTotalPages(totalItems, perPage),
		TotalItems:  normalizeTotalItems(totalItems),
		Size:        perPage,
		Items:       items,
	}
}

// NewPageFromRequest 使用 PageRequest 构造分页响应。
//
// 该方法会先规范化 req，再委托 NewPage 构造最终响应。
func NewPageFromRequest[T any](req PageRequest, totalItems int64, items T) *PageResponse[T] {
	normalized := req.Normalize()
	return NewPage(normalized.Page, normalized.Size, totalItems, items)
}

// NewPageNoData 构造不携带数据的分页响应。
//
// 常用于先返回分页元信息，后续再补充数据，或在无结果时返回标准分页结构。
func NewPageNoData[T any](currentPage, perPage, totalItems int64) *PageResponse[T] {
	var zero T
	return NewPage(currentPage, perPage, totalItems, zero)
}

// NewPageNoDataFromRequest 使用 PageRequest 构造不携带数据的分页响应。
func NewPageNoDataFromRequest[T any](req PageRequest, totalItems int64) *PageResponse[T] {
	var zero T
	return NewPageFromRequest(req, totalItems, zero)
}

// SetData 设置分页响应中的 Items 字段。
//
// 当分页元信息已构建完成，但数据需要延迟填充时可使用该方法。
func (p *PageResponse[T]) SetData(items T) {
	p.Items = items
}

func (c PageConfig) normalize() PageConfig {
	if c.DefaultPage <= 0 {
		c.DefaultPage = DefaultPageNumber
	}

	if c.DefaultSize <= 0 {
		c.DefaultSize = DefaultPageSize
	}

	if c.MaxSize <= 0 {
		c.MaxSize = DefaultPageMaxSize
	}

	if c.DefaultSize > c.MaxSize {
		c.DefaultSize = c.MaxSize
	}

	c.DefaultSort = normalizePageSort(c.DefaultSort, SortAsc)
	return c
}

func normalizePageSort(sort PageSort, fallback PageSort) PageSort {
	value := strings.TrimSpace(strings.ToLower(string(sort)))
	if value == string(SortAsc) {
		return SortAsc
	}
	if value == string(SortDesc) {
		return SortDesc
	}

	fallbackValue := strings.TrimSpace(strings.ToLower(string(fallback)))
	if fallbackValue == string(SortDesc) {
		return SortDesc
	}

	return SortAsc
}

func normalizePageMeta(currentPage, perPage int64) (int64, int64) {
	if currentPage <= 0 {
		currentPage = DefaultPageNumber
	}

	if perPage <= 0 {
		perPage = DefaultPageSize
	}

	return currentPage, perPage
}

func normalizeTotalItems(totalItems int64) int64 {
	if totalItems < 0 {
		return 0
	}

	return totalItems
}

func calcTotalPages(totalItems, perPage int64) int64 {
	totalItems = normalizeTotalItems(totalItems)
	if totalItems == 0 {
		return 0
	}

	if perPage <= 0 {
		perPage = 1
	}

	return (totalItems + perPage - 1) / perPage
}
