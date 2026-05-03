package xEmail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewMessage 测试创建空邮件消息
func TestNewMessage(t *testing.T) {
	msg := NewMessage()
	if msg == nil {
		t.Fatal("NewMessage() 返回 nil")
	}
	if len(msg.To) != 0 {
		t.Errorf("期望 To 为空切片, 实际长度: %d", len(msg.To))
	}
	if msg.Subject != "" {
		t.Errorf("期望 Subject 为空, 实际: %s", msg.Subject)
	}
	if len(msg.Attachments) != 0 {
		t.Errorf("期望 Attachments 为空切片, 实际长度: %d", len(msg.Attachments))
	}
}

// TestMessageAttachReader 测试从 Reader 添加附件
func TestMessageAttachReader(t *testing.T) {
	msg := NewMessage()
	data := strings.NewReader("test content")
	result := msg.AttachReader("test.txt", data)

	if result != msg {
		t.Fatal("AttachReader 应该返回 Message 指针以支持链式调用")
	}
	if len(msg.Attachments) != 1 {
		t.Fatalf("期望 1 个附件, 实际: %d", len(msg.Attachments))
	}
	if msg.Attachments[0].Filename != "test.txt" {
		t.Errorf("期望附件名 test.txt, 实际: %s", msg.Attachments[0].Filename)
	}
}

// TestTemplateManagerRender 测试模板渲染
func TestTemplateManagerRender(t *testing.T) {
	tmpl, err := newTemplateManager("")
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}

	// 测试渲染 welcome 模板（通过 Render 方法）
	data := map[string]string{
		"Username": "test_user",
	}
	html, err := tmpl.Render("welcome", data)
	if err != nil {
		t.Fatalf("渲染 welcome 模板失败: %v", err)
	}

	// 验证 base 布局
	if !strings.Contains(html, "Bamboo Service") {
		t.Error("渲染结果应包含 Bamboo Service 标题")
	}
	if !strings.Contains(html, "此邮件由 Bamboo Service 自动发送") {
		t.Error("渲染结果应包含页脚")
	}
	// 验证内容模板
	if !strings.Contains(html, "test_user") {
		t.Error("渲染结果应包含用户名 test_user")
	}
	if !strings.Contains(html, "欢迎加入") {
		t.Error("渲染结果应包含欢迎标题")
	}
}

// TestRenderVerification 测试渲染验证码模板
func TestRenderVerification(t *testing.T) {
	tmpl, err := newTemplateManager("")
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}

	html, err := tmpl.Render("verification", map[string]string{
		"Code":   "123456",
		"Expire": "5分钟",
	})
	if err != nil {
		t.Fatalf("渲染 verification 模板失败: %v", err)
	}

	if !strings.Contains(html, "Bamboo Service") {
		t.Error("渲染结果应包含 Bamboo Service 标题")
	}
	if !strings.Contains(html, "123456") {
		t.Error("渲染结果应包含验证码 123456")
	}
	if !strings.Contains(html, "5分钟") {
		t.Error("渲染结果应包含过期时间 5分钟")
	}
	if !strings.Contains(html, "验证码") {
		t.Error("渲染结果应包含验证码标题")
	}
}

// TestRenderResetPassword 测试渲染重置密码模板
func TestRenderResetPassword(t *testing.T) {
	tmpl, err := newTemplateManager("")
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}

	html, err := tmpl.Render("reset_password", map[string]string{
		"ResetURL": "https://example.com/reset?token=abc123",
		"Expire":   "30分钟",
	})
	if err != nil {
		t.Fatalf("渲染 reset_password 模板失败: %v", err)
	}

	if !strings.Contains(html, "Bamboo Service") {
		t.Error("渲染结果应包含 Bamboo Service 标题")
	}
	if !strings.Contains(html, "https://example.com/reset?token=abc123") {
		t.Error("渲染结果应包含重置链接")
	}
	if !strings.Contains(html, "重置密码") {
		t.Error("渲染结果应包含重置密码标题")
	}
	if !strings.Contains(html, "30分钟") {
		t.Error("渲染结果应包含过期时间 30分钟")
	}
}

// TestRenderExternalTemplate 测试渲染外部模板
func TestRenderExternalTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	customHTML := `{{define "custom_report"}}<p>报告内容: {{.Title}}</p>{{end}}`
	os.WriteFile(filepath.Join(tmpDir, "custom_report.html"), []byte(customHTML), 0o644)

	tmpl, err := newTemplateManager(tmpDir)
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}

	html, err := tmpl.Render("custom_report", map[string]string{
		"Title": "月度报告",
	})
	if err != nil {
		t.Fatalf("渲染外部模板失败: %v", err)
	}

	if !strings.Contains(html, "月度报告") {
		t.Errorf("渲染结果应包含 月度报告, 实际: %s", html)
	}
	if !strings.Contains(html, "Bamboo Service") {
		t.Error("外部模板渲染结果应包含 Bamboo Service 标题")
	}
}

// TestTemplateManagerListTemplates 测试获取模板列表
func TestTemplateManagerListTemplates(t *testing.T) {
	tmpl, err := newTemplateManager("")
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}

	names := tmpl.ListTemplates()
	if len(names) == 0 {
		t.Fatal("应该返回至少一个内置模板")
	}

	// 验证包含 verification 模板
	found := false
	for _, name := range names {
		if name == "verification" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("模板列表应包含 verification, 实际: %v", names)
	}
}

// TestTemplateManagerRenderNotFound 测试渲染不存在的模板
func TestTemplateManagerRenderNotFound(t *testing.T) {
	tmpl, err := newTemplateManager("")
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}

	_, err = tmpl.Render("nonexistent", nil)
	if err == nil {
		t.Fatal("渲染不存在的模板应返回错误")
	}
}
