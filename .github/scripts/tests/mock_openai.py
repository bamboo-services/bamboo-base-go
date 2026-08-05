#!/usr/bin/env python3
# ============================================================
# mock_openai.py
#
# 极简 OpenAI 兼容 mock server，用于本地测试 generate-release-notes.sh。
# 无论收到什么请求都返回固定 content（含 "Breaking Changes" 供断言）。
#
# 用法: python3 mock_openai.py [端口]   (默认 8099)
# ============================================================
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

# 固定响应：content 含 "Breaking Changes"，供冒烟测试断言
RESPONSE = {
    "choices": [{
        "index": 0,
        "message": {
            "role": "assistant",
            "content": "## ⚠️ Breaking Changes\n"
                       "- 数据库驱动插件化 (a2bf56e)\n\n"
                       "### major\n"
                       "- ♻️ 重构 驱动插件化\n",
        },
    }]
}


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        # 读取并丢弃请求体（仅用于核对请求是否成功送达）
        length = int(self.headers.get("Content-Length", 0))
        self.rfile.read(length)
        data = json.dumps(RESPONSE, ensure_ascii=False).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *args):
        pass  # 静默，避免刷屏


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8099
    print(f"mock openai listening on :{port}", file=sys.stderr)
    HTTPServer(("127.0.0.1", port), Handler).serve_forever()
