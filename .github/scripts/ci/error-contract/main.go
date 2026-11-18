package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	root := flag.String("root", ".", "仓库根目录路径")
	jsonOut := flag.Bool("json", false, "以 JSON 输出检查结果")
	allowUnused := flag.Bool("allow-unused", false, "允许存在未命中的白名单项")
	flag.Parse()

	report, err := CheckRepo(*root, DefaultWhitelist())
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ error-contract 检查执行失败: %v\n", err)
		os.Exit(2)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
	} else {
		fmt.Printf("🔍 error-contract 扫描完成: %d 模块 / %d 包 | 命中白名单: %d | 未使用白名单: %d | 违规: %d\n",
			report.ModuleCount, report.PackageCount, report.MatchedWhitelist, len(report.UnusedWhitelist), len(report.Violations))
		for _, v := range report.Violations {
			fmt.Printf("  ❌ [%s] %s (%s) @ %s — %s\n", v.Kind, v.Symbol, v.PkgPath, v.Location, v.Detail)
		}
		for _, u := range report.UnusedWhitelist {
			fmt.Printf("  ⚠️  [UNUSED %s] %s (%s) — %s\n", u.ID, u.Symbol, u.PkgPath, u.Reason)
		}
	}

	if len(report.Violations) > 0 {
		os.Exit(1)
	}
	if !*allowUnused && len(report.UnusedWhitelist) > 0 {
		os.Exit(1)
	}
	if !*jsonOut {
		fmt.Println("✅ 全部 14 个工作区模块的强类型错误契约与白名单台账均已通过校验！")
	}
}
