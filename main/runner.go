package xMain

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	xEnv "github.com/bamboo-services/bamboo-base-go/env"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xReg "github.com/bamboo-services/bamboo-base-go/register"
)

// Runner 启动应用程序的主入口，协调 HTTP 服务与后台协程的运行、信号处理及优雅关闭。
//
// 该函数首先验证 reg 参数及其核心组件的有效性，随后设置信号监听器以捕获中断信号。
// 它会从环境变量读取主机和端口配置以启动 HTTP 服务器，并支持传入自定义路由注册函数和额外的后台协程函数。
// 在接收到退出信号时，该函数负责取消上下文、通知所有后台协程停止，并在超时时间内强制关闭 HTTP 服务，
// 最后阻塞等待所有相关资源清理完毕后才返回。
//
// 参数 reg 携带 Gin 引擎、上下文及依赖注入的核心注册信息，必须非空且包含有效组件。。
//
// 环境变量 XLF_HOST 和 XLF_PORT 分别用于指定监听地址和端口，默认为 localhost:1118。
func Runner(reg *xReg.Reg, log *xLog.LogNamedLogger, routeFunc func(reg *xReg.Reg), goroutineFunc ...func(ctx context.Context, option ...any)) {
	if reg == nil || reg.Init == nil || reg.Serve == nil {
		log.Panic(context.Background(), "Runner 初始化参数异常: reg/init/serve 不能为空")
		return
	}

	_, cancel := context.WithCancel(reg.Init.Ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if routeFunc != nil {
		routeFunc(reg)
	}

	getHost := xEnv.GetEnvString(xEnv.Host, "localhost")
	getPort := xEnv.GetEnvString(xEnv.Port, "1118")
	server := &http.Server{
		Addr:    getHost + ":" + getPort,
		Handler: reg.Serve,
	}

	engineSync := sync.WaitGroup{}
	engineSync.Add(1)
	shutdownNotify := make(chan struct{})

	go func() {
		defer engineSync.Done()
		log.Info(reg.Init.Ctx, "服务器已成功启动", slog.String("addr", "http(s)://"+server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(reg.Init.Ctx, err.Error())
		}
	}()

	for _, goroutineExec := range goroutineFunc {
		if goroutineExec == nil {
			continue
		}

		engineSync.Add(1)
		doneOnce := sync.Once{}
		doneFunc := func() {
			doneOnce.Do(engineSync.Done)
		}
		funcDone := make(chan struct{})

		go func(execFunc func(context.Context, ...any), ctx context.Context, done chan<- struct{}, finish func()) {
			defer close(done)
			defer finish()
			execFunc(ctx)
		}(goroutineExec, reg.Init.Ctx, funcDone, doneFunc)

		go func(done <-chan struct{}, finish func()) {
			select {
			case <-shutdownNotify:
				finish()
			case <-done:
			}
		}(funcDone, doneFunc)
	}

	go func() {
		<-sigChan
		cancel()
		close(shutdownNotify)

		log.Warn(reg.Init.Ctx, "正在关闭 HTTP 服务器...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error(reg.Init.Ctx, err.Error())
		}
	}()

	engineSync.Wait()
	log.Info(reg.Init.Ctx, "所有服务已安全退出")
	return
}
