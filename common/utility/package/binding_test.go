package pack

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 初始化 Gin 测试模式
func init() {
	gin.SetMode(gin.TestMode)
}

// TestBinding_Data_Success 测试成功绑定 JSON 数据
func TestBinding_Data_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 创建请求体
	type TestRequest struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
		Age   int    `json:"age"`
	}

	requestBody := TestRequest{
		Name:  "Test User",
		Email: "test@example.com",
		Age:   25,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	// 创建 HTTP 请求
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// 执行绑定
	binding := &Binding[TestRequest]{
		Context: c,
		GetData: &TestRequest{},
	}
	result := binding.Data()

	if result == nil {
		t.Error("Data() returned nil for valid input")
		return
	}

	if result.Name != "Test User" {
		t.Errorf("Data() Name = %v, want Test User", result.Name)
	}
	if result.Email != "test@example.com" {
		t.Errorf("Data() Email = %v, want test@example.com", result.Email)
	}
	if result.Age != 25 {
		t.Errorf("Data() Age = %v, want 25", result.Age)
	}
}

// TestBinding_Data_InvalidJSON 测试无效 JSON 数据
func TestBinding_Data_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type TestRequest struct {
		Name string `json:"name" binding:"required"`
	}

	// 创建无效的 JSON
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	binding := &Binding[TestRequest]{
		Context: c,
		GetData: &TestRequest{},
	}
	result := binding.Data()

	if result != nil {
		t.Error("Data() should return nil for invalid JSON")
	}

	// 检查是否中止了请求
	if !c.IsAborted() {
		t.Error("Context should be aborted for invalid JSON")
	}
}

// TestBinding_Data_ValidationFailure 测试验证失败
func TestBinding_Data_ValidationFailure(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type TestRequest struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
	}

	// 缺少必填字段
	requestBody := map[string]interface{}{
		"name": "", // 空字符串，违反 required
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	binding := &Binding[TestRequest]{
		Context: c,
		GetData: &TestRequest{},
	}
	result := binding.Data()

	// 验证失败应该返回 nil
	if result != nil {
		t.Error("Data() should return nil for validation failure")
	}

	// 检查是否中止了请求
	if !c.IsAborted() {
		t.Error("Context should be aborted for validation failure")
	}
}

// TestBinding_Query_Success 测试成功绑定查询参数
func TestBinding_Query_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type TestQuery struct {
		Page     int    `form:"page" binding:"required"`
		PageSize int    `form:"page_size" binding:"required"`
		Keyword  string `form:"keyword"`
	}

	req := httptest.NewRequest(http.MethodGet, "/?page=1&page_size=10&keyword=test", nil)
	c.Request = req

	binding := &Binding[TestQuery]{
		Context: c,
		GetData: &TestQuery{},
	}
	result := binding.Query()

	if result == nil {
		t.Error("Query() returned nil for valid input")
		return
	}

	if result.Page != 1 {
		t.Errorf("Query() Page = %v, want 1", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("Query() PageSize = %v, want 10", result.PageSize)
	}
	if result.Keyword != "test" {
		t.Errorf("Query() Keyword = %v, want test", result.Keyword)
	}
}

// TestBinding_Query_ValidationFailure 测试查询参数验证失败
func TestBinding_Query_ValidationFailure(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type TestQuery struct {
		Page int `form:"page" binding:"required"`
	}

	// 缺少必填的 page 参数
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req

	binding := &Binding[TestQuery]{
		Context: c,
		GetData: &TestQuery{},
	}
	result := binding.Query()

	if result != nil {
		t.Error("Query() should return nil for validation failure")
	}

	if !c.IsAborted() {
		t.Error("Context should be aborted for validation failure")
	}
}

// TestBinding_URI_Success 测试成功绑定 URI 参数
func TestBinding_URI_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type TestURI struct {
		ID   string `uri:"id" binding:"required"`
		Name string `uri:"name" binding:"required"`
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req

	// 设置 URI 参数
	c.Params = gin.Params{
		{Key: "id", Value: "123"},
		{Key: "name", Value: "test"},
	}

	binding := &Binding[TestURI]{
		Context: c,
		GetData: &TestURI{},
	}
	result := binding.URI()

	if result == nil {
		t.Error("URI() returned nil for valid input")
		return
	}

	if result.ID != "123" {
		t.Errorf("URI() ID = %v, want 123", result.ID)
	}
	if result.Name != "test" {
		t.Errorf("URI() Name = %v, want test", result.Name)
	}
}

// TestBinding_URI_ValidationFailure 测试 URI 参数验证失败
func TestBinding_URI_ValidationFailure(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type TestURI struct {
		ID string `uri:"id" binding:"required,uuid"`
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req

	// 设置无效的 UUID
	c.Params = gin.Params{
		{Key: "id", Value: "not-a-uuid"},
	}

	binding := &Binding[TestURI]{
		Context: c,
		GetData: &TestURI{},
	}
	result := binding.URI()

	if result != nil {
		t.Error("URI() should return nil for validation failure")
	}

	if !c.IsAborted() {
		t.Error("Context should be aborted for validation failure")
	}
}

// TestBinding_Header_Success 测试成功绑定 Header
func TestBinding_Header_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type TestHeader struct {
		Authorization string `header:"Authorization" binding:"required"`
		ContentType   string `header:"Content-Type"`
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	binding := &Binding[TestHeader]{
		Context: c,
		GetData: &TestHeader{},
	}
	result := binding.Header()

	if result == nil {
		t.Error("Header() returned nil for valid input")
		return
	}

	if result.Authorization != "Bearer token123" {
		t.Errorf("Header() Authorization = %v, want Bearer token123", result.Authorization)
	}
	if result.ContentType != "application/json" {
		t.Errorf("Header() ContentType = %v, want application/json", result.ContentType)
	}
}

// TestBinding_Header_ValidationFailure 测试 Header 验证失败
func TestBinding_Header_ValidationFailure(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type TestHeader struct {
		Authorization string `header:"Authorization" binding:"required"`
	}

	// 缺少 Authorization header
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req

	binding := &Binding[TestHeader]{
		Context: c,
		GetData: &TestHeader{},
	}
	result := binding.Header()

	if result != nil {
		t.Error("Header() should return nil for validation failure")
	}

	if !c.IsAborted() {
		t.Error("Context should be aborted for validation failure")
	}
}

// TestBinding_NestedStruct 测试嵌套结构体绑定
func TestBinding_NestedStruct(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type Address struct {
		City    string `json:"city" binding:"required"`
		Street  string `json:"street"`
		ZipCode string `json:"zip_code"`
	}

	type UserRequest struct {
		Name    string  `json:"name" binding:"required"`
		Email   string  `json:"email" binding:"required,email"`
		Address Address `json:"address"`
	}

	requestBody := UserRequest{
		Name:  "Test User",
		Email: "test@example.com",
		Address: Address{
			City:    "Beijing",
			Street:  "Main Street",
			ZipCode: "100000",
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	binding := &Binding[UserRequest]{
		Context: c,
		GetData: &UserRequest{},
	}
	result := binding.Data()

	if result == nil {
		t.Error("Data() returned nil for valid nested input")
		return
	}

	if result.Address.City != "Beijing" {
		t.Errorf("Data() Address.City = %v, want Beijing", result.Address.City)
	}
}

// TestBinding_ArrayData 测试数组数据绑定
func TestBinding_ArrayData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type ArrayRequest struct {
		IDs    []int    `json:"ids" binding:"required"`
		Names  []string `json:"names"`
		Active bool     `json:"active"`
	}

	requestBody := ArrayRequest{
		IDs:    []int{1, 2, 3},
		Names:  []string{"a", "b", "c"},
		Active: true,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	binding := &Binding[ArrayRequest]{
		Context: c,
		GetData: &ArrayRequest{},
	}
	result := binding.Data()

	if result == nil {
		t.Error("Data() returned nil for valid array input")
		return
	}

	if len(result.IDs) != 3 {
		t.Errorf("Data() len(IDs) = %v, want 3", len(result.IDs))
	}
}
