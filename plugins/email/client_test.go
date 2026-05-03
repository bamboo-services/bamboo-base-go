package xEmail

import (
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

	// 内置模板使用 {{define "content"}} 定义内容块，
	// "base" 模板通过 {{template "content" .}} 包含内容。
	// 按 ParseFS 的字典序，welcome.html 最后解析，其 "content" 定义为最终版本。
	data := map[string]string{
		"Username": "test_user",
	}

	var buf strings.Builder
	if err := tmpl.templates.ExecuteTemplate(&buf, "base", data); err != nil {
		t.Fatalf("渲染 base 模板失败: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "Bamboo Service") {
		t.Error("渲染结果应包含 Bamboo Service 标题")
	}
	if !strings.Contains(html, "test_user") {
		t.Error("渲染结果应包含用户名 test_user")
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
