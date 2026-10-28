package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// responsePayload 为固定 mock 响应，含 "Breaking Changes" 供 generate-release-notes.sh 冒烟测试断言。
const responsePayload = `{"choices": [{"index": 0, "message": {"role": "assistant", "content": "## ⚠️ Breaking Changes\n- 数据库驱动插件化 (a2bf56e)\n\n### major\n- ♻️ 重构 驱动插件化\n"}}]}`

func main() {
	port := "8099"
	if len(os.Args) > 1 && os.Args[1] != "" {
		port = os.Args[1]
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// 读取并丢弃请求体（仅用于核对请求送达）
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()

			data := []byte(responsePayload)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	fmt.Fprintf(os.Stderr, "mock openai listening on :%s\n", port)
	if err := http.ListenAndServe("127.0.0.1:"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "mock server error: %v\n", err)
		os.Exit(1)
	}
}
