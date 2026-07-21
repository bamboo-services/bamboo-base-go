package xMain

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xReg "github.com/bamboo-services/bamboo-base-go/major/register"
)

// mainRunner 应用主运行器，聚合 Runner 启动期所需的所有状态与资源。
//
// 该结构体将原单函数 Runner 拆解为按生命周期阶段组织的方法集合：
// initContext → initSignal → initSync → initGoroutine → initWeb。
// 各阶段方法分布在同包的不同文件中（goroutine.go / web.go），
// 便于按职责维护。生命周期清理（ctxCancel / signal.Stop）由 [Runner] 函数
// 在所有阶段完成后统一 defer 执行，避免子方法提前释放资源。
type mainRunner struct {
	reg       *xReg.Reg            // 注册中心，携带 Gin 引擎与初始化节点
	log       *xLog.LogNamedLogger // 主入口命名日志器
	runCtx    context.Context      // 运行期上下文，附加协程与 HTTP 共享
	ctxCancel context.CancelFunc   // 运行期上下文取消函数，用于优雅关闭
	sigChan   chan os.Signal       // 信号通道，接收 SIGINT/SIGTERM
	sync      *syncGroup           // 同步原组，协调协程退出
}

// syncGroup 协程同步原组，聚合 WaitGroup 与关闭通知通道。
//
// engineSync 统一计数 HTTP 服务协程与所有附加协程；shutdownNotify 在
// 收到退出信号时关闭，用于通知附加协程主动停止。
type syncGroup struct {
	engineSync     sync.WaitGroup // 协程退出同步
	shutdownNotify chan struct{}  // 关闭通知广播
}

// newMainRunner 创建主运行器实例，初始化同步原组。
//
// 该函数仅完成字段赋值，不执行任何初始化逻辑；实际的生命周期阶段
// 由 [Runner] 函数按顺序调用各 initXxx 方法完成。
func newMainRunner(
	reg *xReg.Reg,
	log *xLog.LogNamedLogger,
) *mainRunner {
	return &mainRunner{
		reg:  reg,
		log:  log,
		sync: &syncGroup{},
	}
}

// Runner 启动应用程序的主入口，协调 HTTP 服务与后台协程的运行、信号处理及优雅关闭。
//
// 该函数首先验证 reg 参数及其核心组件的有效性，随后按生命周期阶段顺序执行：
// initContext → initSignal → initSync → initGoroutine → initWeb。
// 各阶段分布在同包的 goroutine.go / web.go 中，按职责拆分。
//
// 在接收到退出信号时，initWeb 启动的关闭协程会取消上下文、通知所有后台协程停止，
// 并在超时时间内强制关闭 HTTP 服务。函数最后阻塞等待所有相关资源清理完毕后才返回。
//
// 参数:
//   - reg 携带 Gin 引擎、上下文及依赖注入的核心注册信息，必须非空且包含有效组件。
//     组件装配（数据库/缓存/路由等）由 [xReg.Register] 完成，Runner 不再参与装配。
//   - log 主入口日志器，用于输出启动、关闭与异常信息。
//   - goroutineFunc 附加后台协程函数，每个函数接收运行期上下文。
//     Runner 会在收到退出信号后取消上下文并等待所有协程退出。
//
// 环境变量 XLF_HOST 和 XLF_PORT 分别用于指定监听地址和端口，默认为 localhost:1118。
func Runner(
	reg *xReg.Reg,
	log *xLog.LogNamedLogger,
	goroutineFunc ...func(ctx context.Context, extra ...any),
) {
	if reg == nil || reg.Init == nil || reg.Serve == nil {
		log.Panic(context.Background(), "Runner 初始化参数异常: reg/init/serve 不能为空")
		return
	}

	runner := newMainRunner(reg, log)

	runner.initContext()
	defer runner.ctxCancel()

	runner.initSignal()
	defer signal.Stop(runner.sigChan)

	runner.initSync()
	runner.initGoroutine(goroutineFunc...)
	runner.initWeb()
	runner.sync.engineSync.Wait()

	log.Info(runner.runCtx, "所有服务已安全退出")
	return
}

// initContext 创建运行期上下文并替换 reg.Init.Ctx。
//
// 该方法仅完成上下文创建，不注册 defer 清理；ctxCancel 由 [Runner] 函数
// 统一 defer 执行，确保在所有协程退出后才取消上下文。
func (runner *mainRunner) initContext() {
	runner.runCtx, runner.ctxCancel = context.WithCancel(runner.reg.Init.Ctx)
	runner.reg.Init.Ctx = runner.runCtx
}

// initSignal 注册 SIGINT/SIGTERM 信号监听器到 sigChan。
//
// 该方法仅完成信号注册，不注册 defer 清理；signal.Stop 由 [Runner] 函数
// 统一 defer 执行，确保在所有阶段完成后才停止信号监听。
func (runner *mainRunner) initSignal() {
	runner.sigChan = make(chan os.Signal, 1)
	signal.Notify(runner.sigChan, syscall.SIGINT, syscall.SIGTERM)
}

// initSync 初始化协程同步原组。
//
// engineSync 预置 1 个计数（对应 HTTP 服务协程），附加协程在 initGoroutine 中按需 Add。
// shutdownNotify 创建为开放通道，在收到退出信号时由关闭协程 close 广播。
func (runner *mainRunner) initSync() {
	runner.sync.engineSync = sync.WaitGroup{}
	runner.sync.engineSync.Add(1)
	runner.sync.shutdownNotify = make(chan struct{})
}
