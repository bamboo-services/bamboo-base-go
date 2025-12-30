package xReg

import "github.com/joho/godotenv"

// ConfigInit 初始化配置。
//
// 该方法从 .env 文件加载环境变量（如果存在）。
// 配置项直接通过 xEnv.GetXxx() 获取。
func (r *Reg) ConfigInit() {
	// 加载 .env 文件到环境变量（忽略不存在的错误）
	_ = godotenv.Load()
}
