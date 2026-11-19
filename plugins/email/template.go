package email

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

//go:embed template/*.html
var defaultTemplateFS embed.FS

// TemplateManager 负责加载、管理与渲染基于 Go html/template 的邮件正文模板。
// 默认内置包含了 welcome、verification 与 reset_password 等常用模板及通用布局框架。
type TemplateManager struct {
	templates *template.Template
	names     []string
}

// newTemplateManager 创建模板管理器。
// 如果 externalDir 不为空，则加载外部模板目录中的模板，同名模板覆盖内置模板。
//
// 失败结果为 [*apperr.Error]：成功必须返回真正的 nil 指针；
// 模板解析与加载失败按所属域错误码 [EmailError] 归一并保留底层原因。
func newTemplateManager(externalDir string) (*TemplateManager, *apperr.Error) {
	tmpl, err := template.New("").ParseFS(defaultTemplateFS, "template/*.html")
	if err != nil {
		return nil, opErr("email: 解析内置模板失败", err)
	}

	tm := &TemplateManager{
		templates: tmpl,
		names:     extractTemplateNames(tmpl),
	}

	if externalDir != "" {
		if dirErr := tm.AddDir(externalDir); dirErr != nil {
			return nil, opErr("email: 加载外部模板目录失败", dirErr)
		}
	}

	return tm, nil
}

// Render 渲染指定的命名模板。
// 参数 name 为模板文件名（不含 .html 后缀，如 "welcome"）；参数 data 为传递给模板的上下文数据。
// 若模板管理器未初始化、模板不存在或渲染过程中语法报错，返回相应错误。
//
// 失败结果为 [*apperr.Error]：成功必须返回真正的 nil 指针；
// 失败按所属域错误码 [EmailError] 归一并保留底层原因。
func (t *TemplateManager) Render(name string, data any) (string, *apperr.Error) {
	if t == nil || t.templates == nil {
		return "", opErr("email: 模板管理器未初始化", ErrNotInitialized)
	}

	contentTmpl := t.templates.Lookup(name)
	if contentTmpl == nil {
		return "", opErr("email: 邮件模板不存在", fmt.Errorf("email: 模板 %s 不存在", name))
	}

	tmp, err := t.templates.Clone()
	if err != nil {
		return "", opErr("email: 克隆模板失败", err)
	}
	tmp, err = tmp.AddParseTree("content", contentTmpl.Tree)
	if err != nil {
		return "", opErr("email: 组合模板失败", err)
	}

	var buf strings.Builder
	if err := tmp.ExecuteTemplate(&buf, "base", data); err != nil {
		return "", opErr("email: 渲染模板失败", fmt.Errorf("email: 渲染模板 %s 失败: %w", name, err))
	}
	return buf.String(), nil
}

// ListTemplates 返回当前已加载的所有可用模板名称切片（不含 .html 后缀与基础布局模板）。
// 若模板管理器为 nil，返回 nil。
func (t *TemplateManager) ListTemplates() []string {
	if t == nil {
		return nil
	}
	names := make([]string, len(t.names))
	copy(names, t.names)
	return names
}

// AddDir 加载外部模板目录中的所有 *.html 文件。
// 外部目录中的同名模板将自动覆盖默认内置模板。若路径不存在或非目录，返回错误。
//
// 失败结果为 [*apperr.Error]：成功必须返回真正的 nil 指针；
// 失败按所属域错误码 [EmailError] 归一并保留底层原因。
func (t *TemplateManager) AddDir(dir string) *apperr.Error {
	if t == nil || t.templates == nil {
		return opErr("email: 模板管理器未初始化", ErrNotInitialized)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return opErr("email: 模板目录不存在", err)
	}
	if !info.IsDir() {
		return opErr("email: 模板路径不是目录", fmt.Errorf("email: 路径不是目录: %s", dir))
	}

	externalTmpl, err := t.templates.Clone()
	if err != nil {
		return opErr("email: 克隆模板失败", err)
	}

	pattern := strings.TrimRight(dir, "/\\") + "/*.html"
	if _, err = externalTmpl.ParseGlob(pattern); err != nil {
		return opErr("email: 解析外部模板失败", err)
	}

	t.templates = externalTmpl
	t.names = extractTemplateNames(externalTmpl)
	return nil
}

// AddFS 从内嵌文件系统 [fs.FS] 加载匹配 pattern 的外部模板文件。
// 同名模板将覆盖原有的内置模板，适用于单二进制分发场景。
//
// 失败结果为 [*apperr.Error]：成功必须返回真正的 nil 指针；
// 文件系统为空、模板解析失败按所属域错误码 [EmailError] 归一并保留底层原因。
func (t *TemplateManager) AddFS(fsys fs.FS, pattern string) *apperr.Error {
	if t == nil || t.templates == nil {
		return opErr("email: 模板管理器未初始化", ErrNotInitialized)
	}
	if fsys == nil {
		return apperr.New(EmailError, "email: 内嵌模板文件系统为空")
	}
	if pattern == "" {
		return apperr.New(EmailError, "email: 内嵌模板匹配模式为空")
	}

	externalTmpl, err := t.templates.Clone()
	if err != nil {
		return opErr("email: 克隆模板失败", err)
	}

	if _, err = externalTmpl.ParseFS(fsys, pattern); err != nil {
		return opErr("email: 解析内嵌模板失败", err)
	}

	t.templates = externalTmpl
	t.names = extractTemplateNames(externalTmpl)
	return nil
}

func extractTemplateNames(tmpl *template.Template) []string {
	if tmpl == nil {
		return nil
	}
	var names []string
	for _, t := range tmpl.Templates() {
		name := t.Name()
		if name == "" || name == "base" || name == "content" {
			continue
		}
		if cleanName, ok := strings.CutSuffix(name, ".html"); ok {
			cleanName, _ = strings.CutPrefix(cleanName, "template/")
			if cleanName != "_base" {
				names = append(names, cleanName)
			}
		}
	}
	return names
}
