package xModels

// Config 表示应用程序的完整配置结构。
//
// 该类型包含了应用程序运行所需的所有配置项，包括调试模式、数据库连接和 NoSQL 连接配置。
//
// 注意: 使用该类型时，如启用调试模式，可能会输出额外的日志信息。
type Config struct {
	Xlf      xlf      `yaml:"xlf"`      // 唤醒功能开关
	Database database `yaml:"database"` // 数据库配置
	Nosql    nosql    `yaml:"nosql"`    // NoSQL 配置
}

// xlf 表示唤醒功能的配置项。
type xlf struct {
	Debug bool   `yaml:"debug"` // 是否启用调试模式
	Host  string `yaml:"host"`  // 程序监听主机地址
	Port  *int   `yaml:"port"`  // 程序监听端口
}

// database 表示数据库连接的配置项。
type database struct {
	Host   string `yaml:"host"`   // 数据库主机地址
	Port   int    `yaml:"port"`   // 数据库端口
	User   string `yaml:"user"`   // 数据库用户名
	Pass   string `yaml:"pass"`   // 数据库密码
	Name   string `yaml:"name"`   // 数据库名称
	Prefix string `yaml:"prefix"` // 数据库表前缀
}

// nosql 表示 NoSQL (Redis) 连接的配置项。
type nosql struct {
	Host     string `yaml:"host"`     // Redis 主机地址 (默认 localhost)
	Port     int    `yaml:"port"`     // Redis 端口 (默认6379)
	User     string `yaml:"user"`     // Redis 用户名 (可选)
	Pass     string `yaml:"pass"`     // Redis 密码 (可选)
	Database int    `yaml:"database"` // Redis 数据库编号 (0 为默认数据库)
	Prefix   string `yaml:"prefix"`   // Redis 键前缀 (可选，默认为空字符串)
}
