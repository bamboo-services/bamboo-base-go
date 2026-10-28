package logging

// Named* 常量定义统一的日志器命名规范，供 [Logger.WithName]
// 作为名称使用；名称固定为 4 个字符，控制台输出时按职责类别
// 着色，未在着色表中的名称以紫色显示。
const (
	NamedCONT  = "CONT"  // 控制层日志。
	NamedSERV  = "SERV"  // 服务层日志。
	NamedROUT  = "ROUT"  // 路由层日志。
	NamedREPO  = "REPO"  // 数据层日志。
	NamedBUSI  = "BUSI"  // 业务层日志。
	NamedLOGC  = "LOGC"  // 逻辑处理日志。
	NamedPROC  = "PROC"  // 流程处理日志。
	NamedFLOW  = "FLOW"  // 工作流日志。
	NamedTASK  = "TASK"  // 任务处理日志。
	NamedJOBS  = "JOBS"  // 作业调度日志。
	NamedCORE  = "CORE"  // 核心服务日志。
	NamedBASE  = "BASE"  // 基础组件日志。
	NamedMAIN  = "MAIN"  // 主程序日志。
	NamedHTTP  = "HTTP"  // HTTP 服务日志。
	NamedGRPC  = "GRPC"  // GRPC 服务日志。
	NamedSOCK  = "SOCK"  // Socket 连接日志。
	NamedCONN  = "CONN"  // 连接管理日志。
	NamedLINK  = "LINK"  // 链路层日志。
	NamedAUTH  = "AUTH"  // 认证层日志。
	NamedUSER  = "USER"  // 用户管理日志。
	NamedPERM  = "PERM"  // 权限管理日志。
	NamedROLE  = "ROLE"  // 角色管理日志。
	NamedTOKN  = "TOKN"  // 令牌管理日志。
	NamedSIGN  = "SIGN"  // 签名验证日志。
	NamedUTIL  = "UTIL"  // 工具层日志。
	NamedFILT  = "FILT"  // 过滤器层日志。
	NamedMIDE  = "MIDE"  // 中间件层日志。
	NamedVALD  = "VALD"  // 验证器层日志。
	NamedINIT  = "INIT"  // 初始化日志。
	NamedTHOW  = "THOW"  // 抛出错误日志。
	NamedRESU  = "RESU"  // 响应结果日志。
	NamedRECO  = "RECO"  // 恢复层系统日志。
	NamedCRON  = "CRON"  // 定时任务日志。
	NamedEMAIL = "EMAIL" // 邮件服务日志。
)
