package model

import (
	"fmt"
	"os"
	"singbox_sub/src/github.com/sixproxy/logger"
	"singbox_sub/src/github.com/sixproxy/util"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// CityMappingConfig 城市映射配置结构
type CityMappingConfig struct {
	// 城市名称映射（英文 -> 中文）
	CityNameMapping map[string]string `yaml:"city_name_mapping"`

	// 省份到城市的推导映射
	RegionToCityMapping map[string]string `yaml:"region_to_city_mapping"`

	// 运营商名称标准化映射
	ISPNameMapping []ISPMappingRule `yaml:"isp_name_mapping"`

	// 城市和运营商对应的网段数据库
	CityISPDatabase map[string]map[string][]string `yaml:"city_isp_database"`

	// 地区默认网段映射
	RegionalDefaults map[string]string `yaml:"regional_defaults"`

	// 运营商默认城市
	ISPDefaultCities map[string]string `yaml:"isp_default_cities"`

	// 全局默认配置
	Defaults DefaultConfig `yaml:"defaults"`
}

// ISPMappingRule 运营商映射规则
type ISPMappingRule struct {
	Keywords   []string `yaml:"keywords"`
	Normalized string   `yaml:"normalized"`
}

// DefaultConfig 默认配置
type DefaultConfig struct {
	City         string `yaml:"city"`
	ISP          string `yaml:"isp"`
	ClientSubnet string `yaml:"client_subnet"`
}

// 全局配置实例
var (
	cityMappingConfig *CityMappingConfig
	configMutex       sync.RWMutex
	configLoaded      bool
)

// LoadCityMappingConfig 加载城市映射配置
func LoadCityMappingConfig(configPath string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析YAML
	var config CityMappingConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析YAML配置失败: %v", err)
	}

	// 验证配置完整性
	if err := validateConfig(&config); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}

	cityMappingConfig = &config
	configLoaded = true

	logger.ConfigInfo("城市映射配置加载成功: %d个城市映射, %d个ISP规则",
		len(config.CityNameMapping), len(config.ISPNameMapping))

	return nil
}

// validateConfig 验证配置完整性
func validateConfig(config *CityMappingConfig) error {
	if len(config.CityNameMapping) == 0 {
		return fmt.Errorf("城市名称映射不能为空")
	}

	if len(config.ISPNameMapping) == 0 {
		return fmt.Errorf("运营商映射规则不能为空")
	}

	if len(config.CityISPDatabase) == 0 {
		return fmt.Errorf("城市运营商数据库不能为空")
	}

	if config.Defaults.City == "" || config.Defaults.ISP == "" || config.Defaults.ClientSubnet == "" {
		return fmt.Errorf("默认配置不完整")
	}

	return nil
}

// ensureConfigLoaded 确保配置已加载
func ensureConfigLoaded() {
	configMutex.RLock()
	loaded := configLoaded
	configMutex.RUnlock()

	if !loaded {
		// 尝试加载默认配置
		if err := LoadCityMappingConfig("config/city_mapping.yaml"); err != nil {
			logger.ConfigWarn("加载城市映射配置失败: %v，使用内置默认值", err)
			// 初始化一个基本的默认配置
			initFallbackConfig()
		}
	}
}

// initFallbackConfig 初始化备用配置
func initFallbackConfig() {
	configMutex.Lock()
	defer configMutex.Unlock()

	cityMappingConfig = &CityMappingConfig{
		CityNameMapping: map[string]string{
			"Beijing":   "北京",
			"Shanghai":  "上海",
			"Guangzhou": "广州",
			"Shenzhen":  "深圳",
			"Hangzhou":  "杭州",
			"Nanjing":   "南京",
			"Fuzhou":    "福州",
			"Quanzhou":  "泉州",
			"Taipei":    "福州",
			"Hong Kong": "深圳",
			"Macau":     "广州",
		},
		ISPNameMapping: []ISPMappingRule{
			{Keywords: []string{"telecom", "电信", "chinanet"}, Normalized: "电信"},
			{Keywords: []string{"unicom", "联通"}, Normalized: "联通"},
			{Keywords: []string{"mobile", "移动", "cmcc"}, Normalized: "移动"},
			{Keywords: []string{"cernet", "教育", "edu"}, Normalized: "教育网"},
		},
		CityISPDatabase: map[string]map[string][]string{
			"北京": {"电信": []string{"202.101.170.0/24"}},
			"上海": {"电信": []string{"202.96.209.0/24"}},
			"福州": {"电信": []string{"27.155.96.0/24"}},
		},
		Defaults: DefaultConfig{
			City:         "北京",
			ISP:          "电信",
			ClientSubnet: "202.101.170.0/24",
		},
	}
	configLoaded = true
	logger.ConfigWarn("使用内置默认城市映射配置")
}

// GetCityNameCH 获取城市的中文名称（新的配置文件版本）
func GetCityNameCH(enName string) string {
	ensureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if cityMappingConfig == nil {
		return enName
	}

	// 查找直接映射
	if zhName, exists := cityMappingConfig.CityNameMapping[enName]; exists {
		return zhName
	}

	// 返回原始名称，由上层逻辑处理fallback
	return enName
}

// InferCityFromRegion 从省份推导城市（新的配置文件版本）
func InferCityFromRegion(region string) string {
	ensureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if cityMappingConfig == nil {
		return ""
	}

	region = strings.ToLower(region)

	// 查找精确匹配
	if city, exists := cityMappingConfig.RegionToCityMapping[region]; exists {
		return city
	}

	// 查找包含匹配
	for regionKey, city := range cityMappingConfig.RegionToCityMapping {
		if strings.Contains(region, regionKey) {
			return city
		}
	}

	return ""
}

// NormalizeISPName 标准化运营商名称（新的配置文件版本）
func NormalizeISPName(isp string) string {
	ensureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if cityMappingConfig == nil {
		return "电信" // 默认返回电信
	}

	isp = strings.ToLower(isp)

	// 遍历映射规则
	for _, rule := range cityMappingConfig.ISPNameMapping {
		for _, keyword := range rule.Keywords {
			if strings.Contains(isp, strings.ToLower(keyword)) {
				return rule.Normalized
			}
		}
	}

	return "电信" // 默认返回电信
}

// GetCityISPSubnet 根据城市和运营商获取网段（新的配置文件版本）
func GetCityISPSubnet(city, isp string) string {
	ensureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if cityMappingConfig == nil {
		return ""
	}

	if cityData, exists := cityMappingConfig.CityISPDatabase[city]; exists {
		if subnets, exists := cityData[isp]; exists && len(subnets) > 0 {
			return subnets[0]
		}
	}

	return ""
}

// GetFallbackSubnet 获取备选网段（新的配置文件版本）
func GetFallbackSubnet(location *util.LocationInfo) string {
	ensureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if cityMappingConfig == nil {
		return "202.101.170.0/24"
	}

	// 1. 尝试匹配其他城市的相同运营商
	for _, cityData := range cityMappingConfig.CityISPDatabase {
		if subnets, exists := cityData[location.ISP]; exists && len(subnets) > 0 {
			logger.NetworkInfo("使用 %s 网段作为备选", location.ISP)
			return subnets[0]
		}
	}

	// 2. 根据地区选择默认网段
	if subnet, exists := cityMappingConfig.RegionalDefaults[location.Region]; exists {
		return subnet
	}

	// 3. 使用全局默认
	return cityMappingConfig.Defaults.ClientSubnet
}

// GetDefaultCityByISP 根据ISP获取默认城市（新的配置文件版本）
func GetDefaultCityByISP(isp string) string {
	ensureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if cityMappingConfig == nil {
		return "北京"
	}

	if city, exists := cityMappingConfig.ISPDefaultCities[isp]; exists {
		return city
	}

	return cityMappingConfig.Defaults.City
}

// GetDefaultClientSubnet 获取默认client_subnet（新的配置文件版本）
func GetDefaultClientSubnet() string {
	ensureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if cityMappingConfig == nil {
		return "202.101.170.0/24"
	}

	return cityMappingConfig.Defaults.ClientSubnet
}

// GetConfigStats 获取配置统计信息
func GetConfigStats() map[string]int {
	ensureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if cityMappingConfig == nil {
		return map[string]int{}
	}

	stats := map[string]int{
		"城市映射数量":  len(cityMappingConfig.CityNameMapping),
		"省份映射数量":  len(cityMappingConfig.RegionToCityMapping),
		"ISP规则数量": len(cityMappingConfig.ISPNameMapping),
		"城市数据库数量": len(cityMappingConfig.CityISPDatabase),
	}

	// 统计总网段数量
	totalSubnets := 0
	for _, cityData := range cityMappingConfig.CityISPDatabase {
		for _, subnets := range cityData {
			totalSubnets += len(subnets)
		}
	}
	stats["总网段数量"] = totalSubnets

	return stats
}
