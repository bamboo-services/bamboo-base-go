package xInit

import (
	"fmt"
	xModels "github.com/bamboo-services/bamboo-base-go/models"
	"gopkg.in/yaml.v3"
	"os"
)

// ConfigInit 初始化配置文件的加载和解析。
//
// 该方法尝试从以下路径依次读取配置文件：
// 1. config.yaml
// 2. configs/config.yaml
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

	var config *xModels.AwakenConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		panic("[CONF] 配置文件解析失败: " + err.Error())
	}

	fmt.Println(config)

	r.Config = config
}
