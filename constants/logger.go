package xConsts

const (
	// 表示四层架构相关日志
	LogCONT = "CONT" // 控制层日志
	LogSERV = "SERV" // 服务层日志
	LogROUT = "ROUT" // 路由层日志
	LogREPO = "REPO" // 数据层日志

	// 业务相关日志
	LogBUSI = "BUSI" // 业务层日志
	LogLOGC = "LOGC" // 逻辑处理日志
	LogPROC = "PROC" // 流程处理日志
	LogFLOW = "FLOW" // 工作流日志
	LogTASK = "TASK" // 任务处理日志
	LogJOBS = "JOBS" // 作业调度日志

	// 核心服务与组件
	LogCORE = "CORE" // 核心服务日志
	LogBASE = "BASE" // 基础组件日志
	LogMAIN = "MAIN" // 主程序日志

	// 网络与路由
	LogHTTP = "HTTP" // HTTP服务日志
	LogGRPC = "GRPC" // GRPC服务日志
	LogSOCK = "SOCK" // Socket连接日志
	LogCONN = "CONN" // 连接管理日志
	LogLINK = "LINK" // 链路层日志

	// 安全认证相关
	LogAUTH = "AUTH" // 认证层日志
	LogUSER = "USER" // 用户管理日志
	LogPERM = "PERM" // 权限管理日志
	LogROLE = "ROLE" // 角色管理日志
	LogTOKN = "TOKN" // 令牌管理日志
	LogSIGN = "SIGN" // 签名验证日志

	// 系统监控相关
	LogLOGS = "LOGS" // 日志系统日志
	LogMETR = "METR" // 指标监控日志
	LogMONI = "MONI" // 监控系统日志
	LogPERF = "PERF" // 性能监控日志
	LogSTAT = "STAT" // 统计分析日志
	LogHEAL = "HEAL" // 健康检查日志

	// 其他通用类
	LogUTIL = "UTIL" // 工具层日志
	LogFILT = "FILT" // 过滤器层日志
	LogMIDE = "MIDE" // 中间件层日志
	LogINIT = "INIT" // 初始化日志
	LogTHOW = "THOW" // 抛出错误日志
	LogRESU = "RESU" // 响应结果日志
	LogRECO = "RECO" // 恢复层系统日志
)
