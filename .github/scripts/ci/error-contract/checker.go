package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Violation 描述一处违反强类型错误契约（R01/R04/R10）的声明或实现。
type Violation struct {
	Location string
	PkgPath  string
	Kind     EntryKind
	Symbol   string
	Detail   string
}

// Report 汇总类型守卫的扫描结果。
type Report struct {
	ModuleCount      int
	PackageCount     int
	Violations       []Violation
	UnusedWhitelist  []WhitelistEntry
	MatchedWhitelist int
}

// CheckRepo 加载并校验给定仓库根目录中 manifest 列出的全部模块。
func CheckRepo(repoRoot string, entries []WhitelistEntry) (*Report, error) {
	if err := ValidateWhitelist(entries); err != nil {
		return nil, err
	}

	modPaths, err := readModuleManifest(filepath.Join(repoRoot, ".github/scripts/manifest/v2-modules.txt"))
	if err != nil {
		return nil, err
	}

	return CheckPatterns(repoRoot, modPaths, entries)
}

// CheckPatterns 使用 go/packages 对指定模块目录列表（相对 repoRoot）进行真实类型检查。
func CheckPatterns(repoRoot string, relDirs []string, entries []WhitelistEntry) (*Report, error) {
	if err := ValidateWhitelist(entries); err != nil {
		return nil, err
	}

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}

	patterns := make([]string, 0, len(relDirs))
	for _, d := range relDirs {
		patterns = append(patterns, "./"+filepath.ToSlash(d)+"/...")
	}

	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GOWORK=") {
			continue
		}
		env = append(env, kv)
	}
	goWorkPath := filepath.Join(absRoot, "go.work")
	if _, statErr := os.Stat(goWorkPath); statErr == nil {
		env = append(env, "GOWORK="+goWorkPath)
	} else {
		env = append(env, "GOWORK=off")
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedModule,
		Dir:   absRoot,
		Env:   env,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("packages.Load 失败: %w", err)
	}

	var loadErrs []string
	for _, p := range pkgs {
		for _, pe := range p.Errors {
			loadErrs = append(loadErrs, fmt.Sprintf("%s: %s", p.PkgPath, pe.Error()))
		}
	}
	if len(loadErrs) > 0 {
		return nil, fmt.Errorf("模块类型加载存在编译错误:\n  - %s", strings.Join(loadErrs, "\n  - "))
	}

	wlMap := make(map[string]WhitelistEntry, len(entries))
	used := make(map[string]bool, len(entries))
	for _, e := range entries {
		wlMap[e.Key()] = e
	}

	errorType := types.Universe.Lookup("error").Type()

	report := &Report{
		ModuleCount:  len(relDirs),
		PackageCount: len(pkgs),
	}

	for _, pkg := range pkgs {
		relPkg := trimModulePrefix(pkg.PkgPath)
		if isExcludedPkg(relPkg) {
			continue
		}
		checkPackage(pkg, relPkg, errorType, wlMap, used, report)
	}

	for _, e := range entries {
		if used[e.Key()] {
			report.MatchedWhitelist++
		} else {
			report.UnusedWhitelist = append(report.UnusedWhitelist, e)
		}
	}

	sort.Slice(report.Violations, func(i, j int) bool {
		if report.Violations[i].Location != report.Violations[j].Location {
			return report.Violations[i].Location < report.Violations[j].Location
		}
		return report.Violations[i].Symbol < report.Violations[j].Symbol
	})

	return report, nil
}

func readModuleManifest(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("读取模块清单 %s 失败: %w", path, err)
	}
	defer func() {
		_ = f.Close()
	}()

	var dirs []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			dirs = append(dirs, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return dirs, nil
}

func trimModulePrefix(pkgPath string) string {
	// 对仓库自身模块 github.com/bamboo-services/bamboo-base-go/<mod>/v2[/sub] 归一化为 <mod>[/sub]
	if !strings.HasPrefix(pkgPath, modulePrefix) {
		return pkgPath
	}
	rest := strings.TrimPrefix(pkgPath, modulePrefix)
	parts := strings.Split(rest, "/")
	out := make([]string, 0, len(parts))
	removedV2 := false
	for _, p := range parts {
		if !removedV2 && p == "v2" {
			removedV2 = true
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, "/")
}

func isExcludedPkg(relPkg string) bool {
	// internaltest 为测试辅助桩包
	return strings.HasSuffix(relPkg, "/internaltest") || strings.Contains(relPkg, "/internaltest/")
}

func checkPackage(
	pkg *packages.Package,
	relPkg string,
	errorType types.Type,
	wlMap map[string]WhitelistEntry,
	used map[string]bool,
	report *Report,
) {
	if pkg.TypesInfo == nil {
		return
	}

	allowOrRecord := func(pos token.Pos, kind EntryKind, symbol, detail string) {
		k := fmt.Sprintf("%s|%s|%s", kind, relPkg, symbol)
		if _, ok := wlMap[k]; ok {
			used[k] = true
			return
		}
		p := pkg.Fset.Position(pos)
		report.Violations = append(report.Violations, Violation{
			Location: fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column),
			PkgPath:  relPkg,
			Kind:     kind,
			Symbol:   symbol,
			Detail:   detail,
		})
	}

	for _, file := range pkg.Syntax {
		if isGeneratedFile(file) {
			continue
		}

		// 1. 顶层声明：函数/方法、类型声明、包级变量
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				obj := pkg.TypesInfo.Defs[d.Name]
				if obj == nil {
					continue
				}
				fn, ok := obj.(*types.Func)
				if !ok {
					continue
				}
				sig, ok := fn.Type().(*types.Signature)
				if !ok {
					continue
				}
				symbol, kind := formatFuncSymbol(fn)
				if reason := inspectSignatureReturn(sig, errorType); reason != "" {
					allowOrRecord(d.Pos(), kind, symbol, reason)
				}
				// 检查自有函数/方法的回调参数中是否包含返回 error 的函数签名
				if reason := inspectParamCallbacks(sig, errorType); reason != "" {
					allowOrRecord(d.Pos(), kind, symbol+":param-callback", reason)
				}
				// 检查函数体内的匿名函数字面量
				if d.Body != nil {
					inspectFuncLits(d.Body, symbol, pkg, errorType, allowOrRecord)
				}

			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						obj := pkg.TypesInfo.Defs[s.Name]
						if obj == nil {
							continue
						}
						t := obj.Type()
						if s.Assign.IsValid() {
							// 类型别名（如 type myErr = error 或 type fn = func() error / = grpc.UnaryServerInterceptor）
							if containsErrorType(t, errorType) {
								kind := KindFuncType
								if _, isSig := t.Underlying().(*types.Signature); !isSig {
									kind = KindInterface
								}
								allowOrRecord(s.Pos(), kind, s.Name.Name, fmt.Sprintf("类型别名暴露了原生 error: %s", types.TypeString(t, nil)))
							} else if sig, isSig := t.Underlying().(*types.Signature); isSig {
								if reason := inspectSignatureReturn(sig, errorType); reason != "" {
									allowOrRecord(s.Pos(), KindFuncType, s.Name.Name, fmt.Sprintf("类型别名暴露了原生 error: %s (%s)", types.TypeString(t, nil), reason))
								}
							}
							continue
						}

						switch under := t.Underlying().(type) {
						case *types.Signature:
							if reason := inspectSignatureReturn(under, errorType); reason != "" {
								allowOrRecord(s.Pos(), KindFuncType, s.Name.Name, reason)
							}
							if reason := inspectParamCallbacks(under, errorType); reason != "" {
								allowOrRecord(s.Pos(), KindFuncType, s.Name.Name+":param-callback", reason)
							}
						case *types.Interface:
							// 检查自定义 error 接口或包含返回 error 方法的接口（含嵌入第三方接口的方法）
							if isCustomErrorInterface(under, errorType) {
								allowOrRecord(s.Pos(), KindInterface, s.Name.Name, "禁止定义自定义通用 error 接口替代 *apperr.Error")
								continue
							}
							explicit := make(map[*types.Func]bool, under.NumExplicitMethods())
							for i := 0; i < under.NumExplicitMethods(); i++ {
								explicit[under.ExplicitMethod(i)] = true
							}
							for i := 0; i < under.NumMethods(); i++ {
								m := under.Method(i)
								sig, ok := m.Type().(*types.Signature)
								if !ok {
									continue
								}
								pos := m.Pos()
								if !explicit[m] {
									pos = s.Pos()
								}
								if reason := inspectSignatureReturn(sig, errorType); reason != "" {
									allowOrRecord(pos, KindInterface, s.Name.Name+"."+m.Name(), reason)
								}
								if reason := inspectParamCallbacks(sig, errorType); reason != "" {
									allowOrRecord(pos, KindInterface, s.Name.Name+"."+m.Name()+":param-callback", reason)
								}
							}
						case *types.Struct:
							for i := 0; i < under.NumFields(); i++ {
								f := under.Field(i)
								if containsErrorType(f.Type(), errorType) {
									allowOrRecord(f.Pos(), KindField, s.Name.Name+"."+f.Name(), fmt.Sprintf("结构体字段含原生 error 类型: %s", types.TypeString(f.Type(), nil)))
								}
							}
						}

					case *ast.ValueSpec:
						for idx, name := range s.Names {
							if name.Name == "_" {
								continue
							}
							obj := pkg.TypesInfo.Defs[name]
							if obj == nil {
								continue
							}
							if sig, ok := obj.Type().Underlying().(*types.Signature); ok {
								if reason := inspectSignatureReturn(sig, errorType); reason != "" {
									allowOrRecord(name.Pos(), KindVar, name.Name, reason)
								}
							}
							if idx < len(s.Values) {
								inspectFuncLits(s.Values[idx], name.Name, pkg, errorType, allowOrRecord)
							}
						}
					}
				}
			}
		}
	}
}

func inspectFuncLits(
	root ast.Node,
	enclosingSymbol string,
	pkg *packages.Package,
	errorType types.Type,
	allowOrRecord func(token.Pos, EntryKind, string, string),
) {
	litIdx := 0
	ast.Inspect(root, func(n ast.Node) bool {
		fl, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		t := pkg.TypesInfo.TypeOf(fl)
		sig, ok := t.(*types.Signature)
		if !ok {
			return true
		}
		if reason := inspectSignatureReturn(sig, errorType); reason != "" {
			litIdx++
			litSymbol := fmt.Sprintf("%s/lit#%d", enclosingSymbol, litIdx)
			allowOrRecord(fl.Pos(), KindFuncLit, litSymbol, reason)
		}
		return true
	})
}

func formatFuncSymbol(fn *types.Func) (string, EntryKind) {
	sig := fn.Type().(*types.Signature)
	recv := sig.Recv()
	if recv == nil {
		return fn.Name(), KindFunc
	}
	rt := recv.Type()
	isPtr := false
	if p, ok := rt.(*types.Pointer); ok {
		isPtr = true
		rt = p.Elem()
	}
	typeName := rt.String()
	if named, ok := rt.(*types.Named); ok {
		typeName = named.Obj().Name()
	}
	if isPtr {
		return fmt.Sprintf("(*%s).%s", typeName, fn.Name()), KindMethod
	}
	return fmt.Sprintf("(%s).%s", typeName, fn.Name()), KindMethod
}

func inspectSignatureReturn(sig *types.Signature, errorType types.Type) string {
	res := sig.Results()
	for i := 0; i < res.Len(); i++ {
		rt := res.At(i).Type()
		if containsErrorType(rt, errorType) {
			return fmt.Sprintf("返回值第 %d 项包含原生 error 类型 (%s)，应改为 *apperr.Error", i+1, types.TypeString(rt, nil))
		}
	}
	return ""
}

func inspectParamCallbacks(sig *types.Signature, errorType types.Type) string {
	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		pt := params.At(i).Type()
		if containsCallbackReturningError(pt, errorType) {
			return fmt.Sprintf("参数第 %d 项回调签名的返回位置包含原生 error (%s)", i+1, types.TypeString(pt, nil))
		}
	}
	return ""
}

func containsCallbackReturningError(t types.Type, errorType types.Type) bool {
	return walkType(t, map[types.Type]bool{}, func(cur types.Type) bool {
		if sig, ok := cur.(*types.Signature); ok {
			return inspectSignatureReturn(sig, errorType) != ""
		}
		return false
	})
}

func containsErrorType(t types.Type, errorType types.Type) bool {
	return walkType(t, map[types.Type]bool{}, func(cur types.Type) bool {
		if types.Identical(cur, errorType) {
			return true
		}
		if iface, ok := cur.(*types.Interface); ok && isCustomErrorInterface(iface, errorType) {
			return true
		}
		if sig, ok := cur.(*types.Signature); ok {
			return inspectSignatureReturn(sig, errorType) != ""
		}
		return false
	})
}

func walkType(t types.Type, visited map[types.Type]bool, match func(types.Type) bool) bool {
	if t == nil || visited[t] {
		return false
	}
	visited[t] = true
	if match(t) {
		return true
	}
	switch u := t.(type) {
	case *types.Alias:
		return walkType(u.Rhs(), visited, match)
	case *types.Pointer:
		return walkType(u.Elem(), visited, match)
	case *types.Slice:
		return walkType(u.Elem(), visited, match)
	case *types.Array:
		return walkType(u.Elem(), visited, match)
	case *types.Chan:
		return walkType(u.Elem(), visited, match)
	case *types.Map:
		return walkType(u.Key(), visited, match) || walkType(u.Elem(), visited, match)
	case *types.Named:
		if args := u.TypeArgs(); args != nil {
			for i := 0; i < args.Len(); i++ {
				if walkType(args.At(i), visited, match) {
					return true
				}
			}
		}
		// 仅展开本库自有非结构体命名类型；命名结构体字段由 TypeSpec 统一检查并受白名单约束
		if u.Obj() == nil || u.Obj().Pkg() == nil || !strings.HasPrefix(u.Obj().Pkg().Path(), modulePrefix) {
			return false
		}
		if _, isStruct := u.Underlying().(*types.Struct); isStruct {
			return false
		}
		return walkType(u.Underlying(), visited, match)
	case *types.Struct:
		// 匿名结构体（未包装为命名类型）需检查其字段是否夹带原生 error
		for i := 0; i < u.NumFields(); i++ {
			if walkType(u.Field(i).Type(), visited, match) {
				return true
			}
		}
		return false
	case *types.Signature:
		return false
	}
	return false
}

func isCustomErrorInterface(iface *types.Interface, errorType types.Type) bool {
	errIface := errorType.Underlying().(*types.Interface)
	if iface.NumMethods() == 0 {
		return false
	}
	return types.Implements(errorType, iface) && types.Implements(iface, errIface) && iface.NumMethods() == 1
}

func isGeneratedFile(file *ast.File) bool {
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "Code generated") && strings.Contains(c.Text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}
