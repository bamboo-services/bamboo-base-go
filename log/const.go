package xLog

const (
	// 表示四层架构相关日志
	NamedCONT = "CONT" // 控制层日志
	NamedSERV = "SERV" // 服务层日志
	NamedROUT = "ROUT" // 路由层日志
	NamedREPO = "REPO" // 数据层日志

	// 业务相关日志
	NamedBUSI = "BUSI" // 业务层日志
	NamedLOGC = "LOGC" // 逻辑处理日志
	NamedPROC = "PROC" // 流程处理日志
	NamedFLOW = "FLOW" // 工作流日志
	NamedTASK = "TASK" // 任务处理日志
	NamedJOBS = "JOBS" // 作业调度日志

	// 核心服务与组件
	NamedCORE = "CORE" // 核心服务日志
	NamedBASE = "BASE" // 基础组件日志
	NamedMAIN = "MAIN" // 主程序日志

	// 网络与路由
	NamedHTTP = "HTTP" // HTTP服务日志
	NamedGRPC = "GRPC" // GRPC服务日志
	NamedSOCK = "SOCK" // Socket连接日志
	NamedCONN = "CONN" // 连接管理日志
	NamedLINK = "LINK" // 链路层日志

	// 安全认证相关
	NamedAUTH = "AUTH" // 认证层日志
	NamedUSER = "USER" // 用户管理日志
	NamedPERM = "PERM" // 权限管理日志
	NamedROLE = "ROLE" // 角色管理日志
	NamedTOKN = "TOKN" // 令牌管理日志
	NamedSIGN = "SIGN" // 签名验证日志

	// 其他通用类
	NamedUTIL = "UTIL" // 工具层日志
	NamedFILT = "FILT" // 过滤器层日志
	NamedMIDE = "MIDE" // 中间件层日志
	NamedVALD = "VALD" // 验证器层日志
	NamedINIT = "INIT" // 初始化日志
	NamedTHOW = "THOW" // 抛出错误日志
	NamedRESU = "RESU" // 响应结果日志
	NamedRECO = "RECO" // 恢复层系统日志
)
