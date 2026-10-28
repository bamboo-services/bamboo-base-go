package logging_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
)

// ExampleNewLogHandler 展示装配双写日志器、派生命名日志器并发送
// 一条 INFO 日志；控制台输出包含时间戳，因此仅断言关键内容存在。
func ExampleNewLogHandler() {
	var console bytes.Buffer

	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console: &console,
		Level:   slog.LevelInfo,
	})
	logger, err := logging.New(handler)
	if err != nil {
		log.Fatal(err)
	}

	named := logger.WithName(logging.NamedCORE, "BOOT")
	named.Info(context.Background(), "service started")

	fmt.Println(strings.Contains(console.String(), "service started"))
	// Output: true
}

// ExampleNewRotatingWriter 展示轮转写入器的创建、写入与释放：
// 使用结束后必须调用 Close 等待归档并关闭文件句柄。
// 输出包含临时目录路径，因此不校验具体输出。
func ExampleNewRotatingWriter() {
	dir, err := os.MkdirTemp("", "logging-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintln(os.Stderr, "cleanup failed:", err)
		}
	}()

	writer, err := logging.NewRotatingWriter(logging.RotatorConfig{
		Dir:      dir,
		BaseName: "app",
		Ext:      ".log",
		MaxSize:  1024 * 1024,
	})
	if err != nil {
		log.Fatal(err)
	}

	if _, err := writer.Write([]byte("hello bamboo logging\n")); err != nil {
		log.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		log.Fatal(err)
	}
}
