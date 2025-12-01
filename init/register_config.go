package xInit

import (
	"os"

	xUtil "github.com/bamboo-services/bamboo-base-go/utility"
	"gopkg.in/yaml.v3"
)

// ConfigInit 初始化配置文件的加载和解析。
//
// 该方法尝试从以下路径依次读取配置文件：
// 1. config.yaml
// 2. configs/config.yaml
//
// 加载 YAML 配置后，会自动应用环境变量覆盖。
// 环境变量命名规则：BAMBOO_{YAML_PATH}，其中 YAML 路径的点号替换为下划线并转为大写。
// 例如：xlf.debug -> BAMBOO_XLF_DEBUG
//
// 环境变量优先级高于 YAML 配置文件，空字符串也会覆盖原有配置。
//
// 如果配置文件读取或解析失败，程序将终止执行。
func (r *Reg) ConfigInit() {
	configPaths := []string{"config.yaml", "configs/config.yaml"}

	var configData []byte
	var err error

	for _, path := range configPaths {
		configData, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		panic("[CONF] 配置文件读取失败: " + err.Error())
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		panic("[CONF] 配置文件解析失败: " + err.Error())
	}

	// 设置 xlf.debug 默认值
	if config["xlf"] == nil {
		config["xlf"] = map[string]interface{}{
			"debug": false,
		}
	} else if xlfConfig, ok := config["xlf"].(map[string]interface{}); ok {
		if xlfConfig["debug"] == nil {
			xlfConfig["debug"] = false
		}
	}

	// 应用环境变量覆盖
	config = xUtil.ApplyEnvOverrides(config, "")

	r.Config = &config
}
