package model

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"singbox_sub/src/github.com/sixproxy/constant"
	"singbox_sub/src/github.com/sixproxy/logger"
	"singbox_sub/src/github.com/sixproxy/util"
)

// UserConfig 用户配置结构 (对应 config.yaml)
type UserConfig struct {
	Subs         []Sub         `yaml:"subs,omitempty"`
	Experimental *Experimental `yaml:"experimental,omitempty"`
	DNS          *UserDNS      `yaml:"dns,omitempty"`
}

// UserDNS 用户DNS配置
type UserDNS struct {
	ClientSubnet string `yaml:"client_subnet,omitempty"`
	Strategy     string `yaml:"strategy,omitempty"`
	Final        string `yaml:"final,omitempty"`
	AutoOptimize bool   `yaml:"auto_optimize,omitempty"` // 是否自动优化client_subnet
}

// LoadConfig 加载JSON配置文件
func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	return &cfg, json.Unmarshal(b, &cfg)
}

// LoadConfigWithYAML 加载模板配置并用YAML配置覆盖
func LoadConfigWithYAML(templatePath, yamlConfigPath string) (*Config, error) {
	// 1. 加载模板配置
	template, err := LoadConfig(templatePath)
	if err != nil {
		return nil, fmt.Errorf("加载模板配置失败: %v", err)
	}

	// 2. 检查YAML配置文件是否存在
	if _, err := os.Stat(yamlConfigPath); os.IsNotExist(err) {
		logger.ConfigWarn("YAML配置文件不存在: %s，使用模板默认配置", yamlConfigPath)
		return template, nil
	}

	// 3. 加载YAML配置
	yamlData, err := os.ReadFile(yamlConfigPath)
	if err != nil {
		return nil, fmt.Errorf("读取YAML配置文件失败: %v", err)
	}

	var userConfig UserConfig
	err = yaml.Unmarshal(yamlData, &userConfig)
	if err != nil {
		return nil, fmt.Errorf("解析YAML配置文件失败: %v", err)
	}

	// 4. 覆盖配置
	mergeYAMLConfig(template, &userConfig)

	return template, nil
}

// mergeYAMLConfig 将YAML配置覆盖到模板配置中
func mergeYAMLConfig(template *Config, userConfig *UserConfig) {
	// 1. 覆盖订阅配置
	if len(userConfig.Subs) > 0 {
		template.Subs = userConfig.Subs
	}

	// 2. 自动设置experimental功能配置 - 使用内网IP:9095
	if template.Experimental.ClashAPI.ExternalController == "" {
		internalIP := util.GetInternalIP()
		template.Experimental.ClashAPI.ExternalController = fmt.Sprintf("%s:9095", internalIP)
		logger.ConfigInfo("自动设置 external_controller: %s", template.Experimental.ClashAPI.ExternalController)
	}

	// 3. 如果用户在YAML中手动配置了experimental，优先使用用户配置
	if userConfig.Experimental != nil && userConfig.Experimental.ClashAPI.ExternalController != "" {
		template.Experimental.ClashAPI.ExternalController = userConfig.Experimental.ClashAPI.ExternalController
		logger.ConfigInfo("使用用户配置的 external_controller: %s", template.Experimental.ClashAPI.ExternalController)
	}

	// 4. 覆盖DNS配置 (通过重新构造)
	if userConfig.DNS != nil {

		// 配置自动优化，就使用自动优化设置
		if userConfig.DNS.AutoOptimize == true {
			client_subnet := util.GetOptimalClientSubnet()
			template.DNS.ClientSubnet = client_subnet
			template.Route.DefaultDomainResolver.ClientSubnet = client_subnet
			for i, server := range template.DNS.Servers {
				if server.Tag == constant.DNS_LOCAL {
					dns_local := util.GetISPDNS()[0]
					logger.ConfigInfo("自动获取本地运营商DNS: %s", dns_local)
					template.DNS.Servers[i].Server = dns_local
				}
			}
		} else {
			switch {
			case userConfig.DNS.ClientSubnet != "":
				template.DNS.ClientSubnet = userConfig.DNS.ClientSubnet
			case userConfig.DNS.Strategy != "":
				template.DNS.Strategy = userConfig.DNS.Strategy
			case userConfig.DNS.Final != "":
				template.DNS.Final = userConfig.DNS.Final
			}
		}

	}
}
