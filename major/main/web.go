package xMain

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// initWeb 启动 HTTP 服务并注册关闭协程。
//
// 从环境变量 XLF_HOST / XLF_PORT 读取监听地址（默认 localhost:1118），
// 创建 *http.Server 并启动两个协程：
//
//   - 服务协程：调用 ListenAndServe，退出时 Done WaitGroup 并 close serverFailed。
//     非 ErrServerClosed 的错误记录到日志；ErrServerClosed（由关闭协程触发）属正常退出。
//
//   - 关闭协程：select 同时监听 sigChan（SIGINT/SIGTERM）与 serverFailed（服务自己挂了），
//     任一触发都执行关闭流程：取消运行期上下文 → close shutdownNotify 通知附加协程停止
//     → 30s 超时内 server.Shutdown 优雅关闭 HTTP。
//
// serverFailed 的引入解决了端口占用等场景下服务协程提前退出、关闭协程却永久阻塞
// 在 sigChan 的协程泄漏问题。
//
// shutdownNotify 的 close 在关闭协程中完成，先于 server.Shutdown，
// 确保附加协程能及时收到通知并开始退出。
func (runner *mainRunner) initWeb() {
	getHost := xEnv.GetEnvString(xEnv.Host, "localhost")
	getPort := xEnv.GetEnvString(xEnv.Port, "1118")
	server := &http.Server{
		Addr:    getHost + ":" + getPort,
		Handler: runner.reg.Serve,
	}
	serverFailed := make(chan struct{})

	go func() {
		defer runner.sync.engineSync.Done()
		defer close(serverFailed)
		runner.log.Info(runner.runCtx, "服务器已成功启动", slog.String("addr", "http(s)://"+server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			runner.log.Error(runner.runCtx, err.Error())
		}
	}()

	go func() {
		select {
		case <-runner.sigChan:
		case <-serverFailed:
		}
		runner.ctxCancel()
		close(runner.sync.shutdownNotify)

		runner.log.Warn(runner.runCtx, "正在关闭 HTTP 服务器...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			runner.log.Error(runner.runCtx, err.Error())
		}
	}()
}
