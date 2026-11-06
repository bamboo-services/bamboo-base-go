// check-error-codes 校验统一词表及各包预置错误码，不导入 SDK 模块。
package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const vocabularyPath = "docs/engineering/spec/0002-foundation-error-code-vocabulary.md"

var ownerPrefixes = map[string]string{
	"foundation/apperr":          "FRAMEWORK_",
	"plugins/web-gin/httpresult": "HTTP_",
	"plugins/web-gin/bind":       "HTTP_BIND_",
	"plugins/web-gin/validator":  "HTTP_VALIDATOR_",
	"plugins/grpc/grpcresult":    "GRPC_",
	"plugins/database":           "DATABASE_",
	"plugins/cache":              "CACHE_",
	"plugins/email":              "EMAIL_",
}

var outputPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+$`)

type errorEntry struct {
	code    int
	output  string
	owner   string
	name    string
	message string
	state   string
}

func (e errorEntry) key() string {
	return e.owner + "." + e.name
}

func main() {
	if err := checkRepository("."); err != nil {
		fmt.Fprintln(os.Stderr, "错误码门禁:", err)
		os.Exit(1)
	}
	fmt.Println("错误码词表、全库唯一性及源码归属检查通过。")
}

func checkRepository(root string) error {
	source, err := os.ReadFile(filepath.Join(root, vocabularyPath))
	if err != nil {
		return err
	}
	entries, err := parseVocabulary(string(source))
	if err != nil {
		return err
	}
	if err := validateVocabulary(entries); err != nil {
		return err
	}
	reserved := make(map[string]bool, len(entries))
	for _, entry := range entries {
		reserved[entry.key()] = true
	}
	actual, err := scanDefinitions(root, reserved)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		got, found := actual[entry.key()]
		if entry.state == "retired" {
			if found {
				return fmt.Errorf("退役项仍有源码定义: %s", entry.key())
			}
			continue
		}
		if !found {
			return fmt.Errorf("词表项缺少源码定义: %s", entry.key())
		}
		if got.code != entry.code || got.output != entry.output || got.message != entry.message {
			return fmt.Errorf("词表与源码三元组不一致: %s", entry.key())
		}
		delete(actual, entry.key())
	}
	if len(actual) != 0 {
		for key := range actual {
			return fmt.Errorf("源码定义未登记词表: %s", key)
		}
	}
	return nil
}

func parseVocabulary(source string) ([]errorEntry, error) {
	const start = "<!-- error-code-vocabulary:start -->"
	const end = "<!-- error-code-vocabulary:end -->"
	if strings.Count(source, start) != 1 || strings.Count(source, end) != 1 {
		return nil, fmt.Errorf("词表起止标记必须各有一个")
	}
	_, body, _ := strings.Cut(source, start)
	body, _, found := strings.Cut(body, end)
	if !found {
		return nil, fmt.Errorf("词表起止标记顺序错误")
	}
	var entries []errorEntry
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "| Code") || strings.HasPrefix(line, "| ---") {
			continue
		}
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			return nil, fmt.Errorf("词表行不是表格: %s", line)
		}
		fields := strings.Split(strings.Trim(line, "|"), "|")
		if len(fields) != 6 {
			return nil, fmt.Errorf("词表行必须恰好六列: %s", line)
		}
		for i := range fields {
			fields[i] = strings.Trim(fields[i], " `")
		}
		code, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("词表编号非法: %s", fields[0])
		}
		entries = append(entries, errorEntry{code, fields[1], fields[2], fields[3], fields[4], fields[5]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("词表不能为空")
	}
	return entries, nil
}

func validateVocabulary(entries []errorEntry) error {
	codes, outputs, symbols := map[int]bool{}, map[string]bool{}, map[string]bool{}
	next := map[int]int{}
	previous := 0
	for _, entry := range entries {
		if codes[entry.code] || outputs[entry.output] || symbols[entry.key()] {
			return fmt.Errorf("重复Code、Output或所属包符号: %+v", entry)
		}
		codes[entry.code], outputs[entry.output], symbols[entry.key()] = true, true, true
		group, sequence := entry.code/100, entry.code%100
		if group < 400 || group > 599 || sequence < 0 || sequence > 49 {
			return fmt.Errorf("SDK编号必须为400..599分组的00..49序号: %d", entry.code)
		}
		if entry.code <= previous || sequence != next[group] {
			return fmt.Errorf("词表必须数值升序且分组序号连续（含退役项）: %d", entry.code)
		}
		previous, next[group] = entry.code, sequence+1
		prefix, ok := ownerPrefixes[entry.owner]
		if !ok || !outputPattern.MatchString(entry.output) || !strings.HasPrefix(entry.output, prefix) || len(entry.output) == len(prefix) {
			return fmt.Errorf("未知归属或Output前缀不符: %s/%s", entry.owner, entry.output)
		}
		for owner, otherPrefix := range ownerPrefixes {
			if len(otherPrefix) > len(prefix) && strings.HasPrefix(entry.output, otherPrefix) && owner != entry.owner {
				return fmt.Errorf("Output借用了其他包前缀: %s", entry.output)
			}
		}
		if entry.owner == "foundation/apperr" && (group != 500 || !strings.HasPrefix(entry.name, "Framework")) {
			return fmt.Errorf("foundation只允许Framework内部500分组码: %s", entry.key())
		}
		if !token.IsIdentifier(entry.name) || !ast.IsExported(entry.name) || strings.TrimSpace(entry.message) == "" || (entry.state != "active" && entry.state != "retired") {
			return fmt.Errorf("词表符号、文案或状态非法: %s", entry.key())
		}
	}
	return nil
}

func scanDefinitions(root string, reserved map[string]bool) (map[string]errorEntry, error) {
	actual := map[string]errorEntry{}
	for _, directory := range []string{"application", "foundation", "plugins"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, item fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if item.IsDir() {
				if item.Name() == "testdata" || item.Name() == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			owner, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			return collectDefinitions(filepath.ToSlash(owner), filepath.Base(path), file, actual, reserved)
		})
		if err != nil {
			return nil, err
		}
	}
	return actual, nil
}

func collectDefinitions(owner, filename string, file *ast.File, actual map[string]errorEntry, reserved map[string]bool) error {
	qualifier := ""
	imports := map[string]string{}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return err
		}
		importOwner := strings.TrimPrefix(importPath, "github.com/bamboo-services/bamboo-base-go/")
		importOwner = strings.Replace(importOwner, "/v2", "", 1)
		name := filepath.Base(importOwner)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if _, controlled := ownerPrefixes[importOwner]; controlled && name == "." {
			return fmt.Errorf("禁止点导入受控错误码包: %s/%s", owner, filename)
		}
		imports[name] = importOwner
		if importOwner == "foundation/apperr" {
			qualifier = name
		}
	}
	isErrorCode := func(expr ast.Expr) bool {
		switch value := expr.(type) {
		case *ast.Ident:
			return value.Name == "ErrorCode" && owner == "foundation/apperr"
		case *ast.SelectorExpr:
			pkg, ok := value.X.(*ast.Ident)
			return ok && qualifier != "" && pkg.Name == qualifier && value.Sel.Name == "ErrorCode"
		}
		return false
	}
	catalog := filename == "error_code.go"
	seen := map[*ast.CompositeLit]bool{}
	for _, declaration := range file.Decls {
		decl, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range decl.Specs {
			if alias, ok := spec.(*ast.TypeSpec); ok && isErrorCode(alias.Type) {
				return fmt.Errorf("禁止用类型别名绕过错误码词表: %s/%s", owner, filename)
			}
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if !catalog {
				typeExpr := value.Type
				if pointer, ok := typeExpr.(*ast.StarExpr); ok {
					typeExpr = pointer.X
				}
				if isErrorCode(typeExpr) {
					return fmt.Errorf("目录外不允许包级ErrorCode变量: %s/%s", owner, filename)
				}
				for _, initializer := range value.Values {
					if referencesCode(initializer, owner, imports, reserved) {
						return fmt.Errorf("目录外不允许错误码别名: %s/%s", owner, filename)
					}
				}
				continue
			}
			if decl.Tok != token.VAR || len(value.Names) != 1 || len(value.Values) != 1 || !ast.IsExported(value.Names[0].Name) {
				return fmt.Errorf("目录只能包含单项导出var声明: %s/%s", owner, filename)
			}
			pointer, ok := value.Values[0].(*ast.UnaryExpr)
			if !ok || pointer.Op != token.AND {
				return fmt.Errorf("禁止错误码别名或非字面量定义: %s", value.Names[0].Name)
			}
			literal, ok := pointer.X.(*ast.CompositeLit)
			if !ok || !isErrorCode(literal.Type) {
				return fmt.Errorf("目录声明必须为&apperr.ErrorCode具名字面量: %s", value.Names[0].Name)
			}
			entry, err := readLiteral(literal)
			if err != nil {
				return err
			}
			entry.owner, entry.name = owner, value.Names[0].Name
			if _, exists := actual[entry.key()]; exists {
				return fmt.Errorf("重复源码符号: %s", entry.key())
			}
			actual[entry.key()] = entry
			seen[literal] = true
		}
	}
	var violation error
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if ok && isErrorCode(literal.Type) && !seen[literal] {
			violation = fmt.Errorf("预置错误必须集中于error_code.go: %s/%s", owner, filename)
		}
		return violation == nil
	})
	return violation
}

func referencesCode(expr ast.Expr, owner string, imports map[string]string, reserved map[string]bool) bool {
	switch value := expr.(type) {
	case *ast.Ident:
		return reserved[owner+"."+value.Name]
	case *ast.SelectorExpr:
		if qualifier, ok := value.X.(*ast.Ident); ok {
			return reserved[imports[qualifier.Name]+"."+value.Sel.Name]
		}
	case *ast.ParenExpr:
		return referencesCode(value.X, owner, imports, reserved)
	case *ast.StarExpr:
		return referencesCode(value.X, owner, imports, reserved)
	case *ast.UnaryExpr:
		return referencesCode(value.X, owner, imports, reserved)
	}
	return false
}

func readLiteral(literal *ast.CompositeLit) (errorEntry, error) {
	fields := map[string]string{}
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return errorEntry{}, fmt.Errorf("错误码必须使用Code/Output/Message具名字段")
		}
		key, keyOK := pair.Key.(*ast.Ident)
		value, valueOK := pair.Value.(*ast.BasicLit)
		if !keyOK || !valueOK || (key.Name != "Code" && key.Name != "Output" && key.Name != "Message") {
			return errorEntry{}, fmt.Errorf("错误码字段必须为静态整数/字符串字面量")
		}
		if _, exists := fields[key.Name]; exists {
			return errorEntry{}, fmt.Errorf("重复错误码字段: %s", key.Name)
		}
		fields[key.Name] = value.Value
	}
	if len(fields["Code"]) != 5 || fields["Code"][0] == '0' {
		return errorEntry{}, fmt.Errorf("Code必须为5位规范十进制字面量，禁止八进制、进制前缀及数字分隔符")
	}
	code, codeErr := strconv.Atoi(fields["Code"])
	output, outputErr := strconv.Unquote(fields["Output"])
	message, messageErr := strconv.Unquote(fields["Message"])
	if len(fields) != 3 || codeErr != nil || outputErr != nil || messageErr != nil {
		return errorEntry{}, fmt.Errorf("错误码必须包含静态Code/Output/Message三元组")
	}
	return errorEntry{code: code, output: output, message: message}, nil
}
