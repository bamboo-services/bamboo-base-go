package xEmail

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"strings"
)

//go:embed template/*.html
var templateFS embed.FS

// TemplateManager 邮件模板管理器
type TemplateManager struct {
	templates *template.Template
	names     []string
}

// newTemplateManager 创建模板管理器
//
// 如果 externalDir 不为空，则加载外部模板目录中的模板，
// 同名模板会覆盖内置模板。
func newTemplateManager(externalDir string) (*TemplateManager, error) {
	tmpl, err := template.New("").ParseFS(templateFS, "template/*.html")
	if err != nil {
		return nil, fmt.Errorf("解析内置模板失败: %w", err)
	}

	tm := &TemplateManager{
		templates: tmpl,
		names:     extractTemplateNames(tmpl),
	}

	if externalDir != "" {
		if err := tm.AddDir(externalDir); err != nil {
			return nil, fmt.Errorf("加载外部模板目录失败: %w", err)
		}
	}

	return tm, nil
}

// Render 渲染指定模板
//
// name 为模板名称（不含 .html 后缀），data 为模板数据。
func (t *TemplateManager) Render(name string, data any) (string, error) {
	var buf strings.Builder
	tmplName := "template/" + name + ".html"
	if err := t.templates.ExecuteTemplate(&buf, tmplName, data); err != nil {
		return "", fmt.Errorf("渲染模板 %s 失败: %w", name, err)
	}
	return buf.String(), nil
}

// ListTemplates 返回可用模板名称列表
func (t *TemplateManager) ListTemplates() []string {
	return t.names
}

// AddDir 添加外部模板目录
//
// 外部目录中的同名模板会覆盖内置模板。
func (t *TemplateManager) AddDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("模板目录不存在: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("路径不是目录: %s", dir)
	}

	externalTmpl, err := t.templates.Clone()
	if err != nil {
		return fmt.Errorf("克隆模板失败: %w", err)
	}

	pattern := dir + "/*.html"
	if _, err = externalTmpl.ParseGlob(pattern); err != nil {
		return fmt.Errorf("解析外部模板失败: %w", err)
	}

	t.templates = externalTmpl
	t.names = extractTemplateNames(externalTmpl)
	return nil
}

// extractTemplateNames 从模板中提取模板名称
func extractTemplateNames(tmpl *template.Template) []string {
	var names []string
	for _, t := range tmpl.Templates() {
		name := t.Name()
		if name == "" {
			continue
		}
		// 提取 "template/verification.html" -> "verification"
		if cleanName, ok := strings.CutSuffix(name, ".html"); ok {
			cleanName, ok = strings.CutPrefix(cleanName, "template/")
			if ok || !strings.Contains(name, "/") {
				if cleanName != "_base" {
					names = append(names, cleanName)
				}
			}
		}
	}
	return names
}
