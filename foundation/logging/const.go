package logging

// Named* 常量定义统一的日志器命名规范，供 [Logger.WithName]
// 作为名称使用；名称固定为 4 个字符，控制台输出时按职责类别
// 着色，未在着色表中的名称以紫色显示。
const (
	// NamedCONT 表示控制层日志。
	NamedCONT = "CONT"
	// NamedSERV 表示服务层日志。
	NamedSERV = "SERV"
	// NamedROUT 表示路由层日志。
	NamedROUT = "ROUT"
	// NamedREPO 表示数据层日志。
	NamedREPO = "REPO"

	// NamedBUSI 表示业务层日志。
	NamedBUSI = "BUSI"
	// NamedLOGC 表示逻辑处理日志。
	NamedLOGC = "LOGC"
	// NamedPROC 表示流程处理日志。
	NamedPROC = "PROC"
	// NamedFLOW 表示工作流日志。
	NamedFLOW = "FLOW"
	// NamedTASK 表示任务处理日志。
	NamedTASK = "TASK"
	// NamedJOBS 表示作业调度日志。
	NamedJOBS = "JOBS"

	// NamedCORE 表示核心服务日志。
	NamedCORE = "CORE"
	// NamedBASE 表示基础组件日志。
	NamedBASE = "BASE"
	// NamedMAIN 表示主程序日志。
	NamedMAIN = "MAIN"

	// NamedHTTP 表示 HTTP 服务日志。
	NamedHTTP = "HTTP"
	// NamedGRPC 表示 GRPC 服务日志。
	NamedGRPC = "GRPC"
	// NamedSOCK 表示 Socket 连接日志。
	NamedSOCK = "SOCK"
	// NamedCONN 表示连接管理日志。
	NamedCONN = "CONN"
	// NamedLINK 表示链路层日志。
	NamedLINK = "LINK"

	// NamedAUTH 表示认证层日志。
	NamedAUTH = "AUTH"
	// NamedUSER 表示用户管理日志。
	NamedUSER = "USER"
	// NamedPERM 表示权限管理日志。
	NamedPERM = "PERM"
	// NamedROLE 表示角色管理日志。
	NamedROLE = "ROLE"
	// NamedTOKN 表示令牌管理日志。
	NamedTOKN = "TOKN"
	// NamedSIGN 表示签名验证日志。
	NamedSIGN = "SIGN"

	// NamedUTIL 表示工具层日志。
	NamedUTIL = "UTIL"
	// NamedFILT 表示过滤器层日志。
	NamedFILT = "FILT"
	// NamedMIDE 表示中间件层日志。
	NamedMIDE = "MIDE"
	// NamedVALD 表示验证器层日志。
	NamedVALD = "VALD"
	// NamedINIT 表示初始化日志。
	NamedINIT = "INIT"
	// NamedTHOW 表示抛出错误日志。
	NamedTHOW = "THOW"
	// NamedRESU 表示响应结果日志。
	NamedRESU = "RESU"
	// NamedRECO 表示恢复层系统日志。
	NamedRECO = "RECO"
	// NamedCRON 表示定时任务日志。
	NamedCRON = "CRON"
	// NamedEMAIL 表示邮件服务日志。
	NamedEMAIL = "EMAIL"
)
