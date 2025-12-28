package xModels

// Config 表示应用程序的完整配置结构。
//
// 该类型包含了应用程序运行所需的所有配置项，包括调试模式、数据库连接和 NoSQL 连接配置。
// 配置值从环境变量加载，支持 .env 文件。
//
// 注意: 使用该类型时，如启用调试模式，可能会输出额外的日志信息。
type Config struct {
	Xlf      Xlf      // 应用基础配置
	Database Database // 数据库配置
	Nosql    Nosql    // NoSQL 配置
}

// Xlf 表示应用基础配置项。
type Xlf struct {
	Debug bool   // 是否启用调试模式
	Host  string // 程序监听主机地址
	Port  int    // 程序监听端口
}

// Database 表示数据库连接的配置项。
type Database struct {
	Host   string // 数据库主机地址 [必填]
	Port   int    // 数据库端口
	User   string // 数据库用户名 [必填]
	Pass   string // 数据库密码 [必填]
	Name   string // 数据库名称 [必填]
	Prefix string // 数据库表前缀
}

// Nosql 表示 NoSQL (Redis) 连接的配置项。
type Nosql struct {
	Host     string // Redis 主机地址
	Port     int    // Redis 端口
	User     string // Redis 用户名
	Pass     string // Redis 密码
	Database int    // Redis 数据库编号
	Prefix   string // Redis 键前缀
}
