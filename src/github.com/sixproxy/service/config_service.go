package service

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"singbox_sub/src/github.com/sixproxy/constant"
	"singbox_sub/src/github.com/sixproxy/logger"
	"singbox_sub/src/github.com/sixproxy/model"
	"strings"
	"time"
)

type ConfigService struct {
	UserConfig *model.UserConfig
}

func (u *ConfigService) LoadConfig(yamlConfigPath string) (*model.UserConfig, error) {
	userConfig := model.UserConfig{}
	// 1. 检查YAML配置文件是否存在
	if _, err := os.Stat(yamlConfigPath); os.IsNotExist(err) {
		logger.ConfigWarn("YAML配置文件不存在: %s，使用模板默认配置", yamlConfigPath)
	} else {
		// 加载YAML配置
		yamlData, err := os.ReadFile(yamlConfigPath)
		if err != nil {
			return nil, fmt.Errorf("读取YAML配置文件失败: %v", err)
		}

		err = yaml.Unmarshal(yamlData, &userConfig)
		if err != nil {
			return nil, fmt.Errorf("解析YAML配置文件失败: %v", err)
		}
		u.UserConfig = &userConfig
	}
	return &userConfig, nil
}

// LoadConfigWithYAML 加载模板配置并用YAML配置覆盖
func (u *ConfigService) LoadTemplate(templatePath string) (*model.Config, error) {

	var templateNewContent string
	var err error
	if u.UserConfig.GitHub != nil {
		templateNewContent, err = u.updateTemplateMirrors(templatePath, u.UserConfig.GitHub)
		if err != nil {
			logger.Warn("更新模板镜像失败: %v", err)
		}
	}

	// 3. 加载（可能已更新的）模板配置
	cfg := &model.Config{}
	err = json.Unmarshal([]byte(templateNewContent), &cfg)
	if err != nil {
		return nil, fmt.Errorf("加载模板配置失败: %v", err)
	}

	// 4. 覆盖其他配置
	u.mergeYAMLConfigToTemplate(cfg)

	return cfg, nil
}

// updateTemplateMirrors 更新模板中的GitHub镜像
func (u *ConfigService) updateTemplateMirrors(templatePath string, githubConfig *model.GitHubConfig) (string, error) {
	logger.Info("🔄 开始更新模板文件中的GitHub镜像地址...")

	// 1.读取模板文件
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("读取模板文件失败: %v", err)
	}

	originalContent := string(content)

	// 2. 确定要使用的镜像地址
	targetMirror, err := selectBestMirror(githubConfig.MirrorURL)
	if err != nil {
		return originalContent, nil
	}

	if targetMirror == "" {
		logger.Info("未配置镜像或镜像不可用，保持原有配置")
		return originalContent, nil
	}

	// 3. 检查模板中是否包含占位符
	if !strings.Contains(originalContent, constant.MIRROR_URL) {
		logger.Info(fmt.Sprintf("✅ 模板未使用%s占位符，无需更新", constant.MIRROR_URL))
		return originalContent, nil
	}

	// 4. 替换占位符
	newContent := replaceMirrorPlaceholder(originalContent, targetMirror)

	logger.Info("✅ 成功更新模板镜像地址")
	logger.Info("   镜像地址: %s", targetMirror)

	return newContent, nil
}

// mergeYAMLConfig 将YAML配置覆盖到模板配置中
func (u *ConfigService) mergeYAMLConfigToTemplate(template *model.Config) {
	// 初始化service的城市映射提供者

	// 1. 覆盖订阅配置
	if len(u.UserConfig.Subs) > 0 {
		template.Subs = u.UserConfig.Subs
	}

	// 2. 自动设置experimental功能配置 - 使用内网IP:9095
	if template.Experimental.ClashAPI.ExternalController == "" {
		internalIP := GetInternalIP()
		template.Experimental.ClashAPI.ExternalController = fmt.Sprintf("%s:9095", internalIP)
		logger.ConfigInfo("自动设置 external_controller: %s", template.Experimental.ClashAPI.ExternalController)
	}

	// 3. 如果用户在YAML中手动配置了experimental，优先使用用户配置
	if u.UserConfig.Experimental != nil && u.UserConfig.Experimental.ClashAPI.ExternalController != "" {
		template.Experimental.ClashAPI.ExternalController = u.UserConfig.Experimental.ClashAPI.ExternalController
		logger.ConfigInfo("使用用户配置的 external_controller: %s", template.Experimental.ClashAPI.ExternalController)
	}

	// 4. 覆盖DNS配置 (通过重新构造)
	if u.UserConfig.DNS != nil {

		// 配置自动优化，就使用自动优化设置
		if u.UserConfig.DNS.AutoOptimize == true {
			client_subnet := GetOptimalClientSubnet()
			template.DNS.ClientSubnet = client_subnet
			template.Route.DefaultDomainResolver.ClientSubnet = client_subnet
			for i, server := range template.DNS.Servers {
				if server.Tag == constant.DNS_LOCAL {
					dns_local := GetISPDNS()[0]
					logger.ConfigInfo("自动获取本地运营商DNS: %s", dns_local)
					template.DNS.Servers[i].Server = dns_local
				}
			}
		} else {
			switch {
			case u.UserConfig.DNS.ClientSubnet != "":
				template.DNS.ClientSubnet = u.UserConfig.DNS.ClientSubnet
			case u.UserConfig.DNS.Strategy != "":
				template.DNS.Strategy = u.UserConfig.DNS.Strategy
			case u.UserConfig.DNS.Final != "":
				template.DNS.Final = u.UserConfig.DNS.Final
			}
		}

	}

	// 5. 处理GitHub配置
	if u.UserConfig.GitHub != nil {
		logger.ConfigInfo("已加载GitHub镜像配置")
		if u.UserConfig.GitHub.MirrorURL != "" {
			logger.ConfigInfo("主要镜像: %s", u.UserConfig.GitHub.MirrorURL)
		}
		if len(u.UserConfig.GitHub.FallbackMirrors) > 0 {
			logger.ConfigInfo("备用镜像数量: %d", len(u.UserConfig.GitHub.FallbackMirrors))
		}
	}
}

// selectBestMirror 选择最佳镜像地址
func selectBestMirror(userMirror string) (string, error) {
	// 如果用户没有配置镜像，直接返回空
	if userMirror == "" {
		logger.Info("用户未配置GitHub镜像，保持原始GitHub地址")
		return "", nil
	}

	logger.Info("🧪 测试用户配置的镜像: %s", userMirror)
	if testMirrorConnectivity(userMirror) {
		logger.Info("✅ 用户镜像可用")
		return strings.TrimSuffix(userMirror, "/"), nil
	} else {
		return "", fmt.Errorf("用户配置的GitHub镜像 %s 不可用，请检查网络连接或更换镜像地址", userMirror)
	}
}

// replaceMirrorPlaceholder 替换模板中的{{mirror_url}}占位符
func replaceMirrorPlaceholder(content, mirrorURL string) string {
	// 确保镜像URL末尾没有斜杠（模板中已经包含了斜杠）
	cleanMirrorURL := strings.TrimSuffix(mirrorURL, "/")

	// 简单的字符串替换
	return strings.ReplaceAll(content, constant.MIRROR_URL, cleanMirrorURL)
}

// testMirrorConnectivity 测试镜像连通性
func testMirrorConnectivity(mirrorURL string) bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 构造测试URL - 使用具体的GitHub文件URL来测试
	var testURL string
	if strings.HasSuffix(mirrorURL, "/") {
		testURL = mirrorURL + "https://raw.githubusercontent.com/sixproxy/singbox_sub/main/README.md"
	} else {
		testURL = mirrorURL + "/https://raw.githubusercontent.com/sixproxy/singbox_sub/main/README.md"
	}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		logger.Debug("创建请求失败 %s: %v", mirrorURL, err)
		return false
	}

	// 设置User-Agent以避免被拦截
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("连接失败 %s: %v", mirrorURL, err)
		return false
	}
	defer resp.Body.Close()

	// GitHub镜像检测逻辑：
	// 200: 完全正常，内容已获取
	// 304: 内容未修改（缓存有效），也是可用的
	isAvailable := resp.StatusCode == 200 || resp.StatusCode == 304
	logger.Debug("镜像 %s 测试结果: %d (%t)", mirrorURL, resp.StatusCode, isAvailable)
	return isAvailable
}
