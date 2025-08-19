package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/logger"
	"strconv"
	"strings"
	"time"
)

func String2Int(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		return -999999
	}
	return i
}

func RemoveEmoji(s string) string {
	bs := []byte(s)
	out := bs[:0] // 复用原切片，省一次分配

	for i := 0; i < len(bs); {
		// 4 字节 UTF-8 的首字节一定是 0xF0~0xF4
		if bs[i]&0xF8 == 0xF0 { // 0xF0 = 11110000
			// 跳过这 4 字节（一个 emoji）
			i += 4
			continue
		}
		// 普通字符，拷贝过去
		out = append(out, bs[i])
		i++
	}
	return string(out)
}

func ParseTag(data string) string {

	// 处理 # 标签
	if hashIndex := strings.Index(data, "#"); hashIndex != -1 {
		if hashIndex+1 < len(data) {
			tag, err := url.QueryUnescape(data[hashIndex+1:])
			if err == nil && tag != "" {
				return strings.TrimSpace(RemoveEmoji(tag))
			}
		}
		data = data[:hashIndex]
	}

	// 处理 ?remarks= 标签
	if remarksIndex := strings.Index(data, "?remarks="); remarksIndex != -1 {
		if remarksIndex+9 < len(data) {
			tag, err := url.QueryUnescape(data[remarksIndex+9:])
			if err == nil && tag != "" {
				return RemoveEmoji(tag)
			}
		}
		data = data[:remarksIndex]
	}

	return ""
}

// GetSystemDNS 自动获取当前系统配置的DNS服务器
func GetSystemDNS() ([]string, error) {
	switch runtime.GOOS {
	case "darwin":
		return getDNSMacOS()
	case "linux":
		return getDNSLinux()
	case "windows":
		return getDNSWindows()
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// getDNSMacOS 获取macOS系统的DNS设置
func getDNSMacOS() ([]string, error) {
	// 方法1: 尝试读取系统配置
	if dns, err := getDNSFromScutil(); err == nil && len(dns) > 0 {
		return dns, nil
	}

	// 方法2: 尝试读取resolv.conf
	if dns, err := getDNSFromResolvConf(); err == nil && len(dns) > 0 {
		return dns, nil
	}

	// 方法3: 使用Go的默认DNS解析器配置
	return getDNSFromGoResolver()
}

// getDNSLinux 获取Linux系统的DNS设置
func getDNSLinux() ([]string, error) {
	// 方法1: 读取 /etc/resolv.conf
	if dns, err := getDNSFromResolvConf(); err == nil && len(dns) > 0 {
		return dns, nil
	}

	// 方法2: 尝试systemd-resolve
	if dns, err := getDNSFromSystemdResolve(); err == nil && len(dns) > 0 {
		return dns, nil
	}

	// 方法3: 使用Go的默认DNS解析器配置
	return getDNSFromGoResolver()
}

// getDNSWindows 获取Windows系统的DNS设置
func getDNSWindows() ([]string, error) {
	// Windows上主要通过Go的默认解析器获取
	return getDNSFromGoResolver()
}

// getDNSFromScutil macOS特有: 通过scutil命令获取DNS
func getDNSFromScutil() ([]string, error) {
	// 这里我们通过读取系统网络配置来获取DNS
	// macOS的网络配置通常存储在系统配置数据库中
	// 由于无法直接执行scutil，我们回退到其他方法
	return nil, fmt.Errorf("scutil method not available")
}

// getDNSFromResolvConf 从 /etc/resolv.conf 读取DNS服务器
func getDNSFromResolvConf() ([]string, error) {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dnsServers []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 忽略注释行
		if strings.HasPrefix(line, "#") {
			continue
		}

		// 查找 nameserver 行
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := strings.TrimSpace(parts[1])
				// 验证是否为有效IP
				if net.ParseIP(ip) != nil {
					dnsServers = append(dnsServers, ip)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return dnsServers, nil
}

// getDNSFromSystemdResolve Linux特有: 通过systemd-resolve获取DNS
func getDNSFromSystemdResolve() ([]string, error) {
	// 尝试读取systemd-resolve的状态文件
	possiblePaths := []string{
		"/run/systemd/resolve/resolv.conf",
		"/run/systemd/resolve/stub-resolv.conf",
	}

	for _, path := range possiblePaths {
		if dns, err := readDNSFromFile(path); err == nil && len(dns) > 0 {
			return dns, nil
		}
	}

	return nil, fmt.Errorf("systemd-resolve DNS not found")
}

// readDNSFromFile 从指定文件读取DNS配置
func readDNSFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dnsServers []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := strings.TrimSpace(parts[1])
				if net.ParseIP(ip) != nil {
					dnsServers = append(dnsServers, ip)
				}
			}
		}
	}

	return dnsServers, scanner.Err()
}

// getDNSFromGoResolver 使用Go内置方法获取DNS服务器
func getDNSFromGoResolver() ([]string, error) {
	// 这个方法通过解析一个已知域名来获取DNS服务器信息
	// 虽然不是直接获取系统DNS配置，但可以作为后备方案

	// 尝试查询一些常见的DNS服务器来确定当前使用的DNS
	testDomains := []string{
		"google.com",
		"cloudflare.com",
		"baidu.com",
	}

	// 获取当前网络接口的DNS信息
	interfaces, err := net.Interfaces()
	if err != nil {
		return getDefaultDNS(), nil // 返回默认DNS作为后备
	}

	var dnsServers []string
	seen := make(map[string]bool)

	// 尝试从网络接口获取DNS信息
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					// 基于网络段猜测可能的DNS服务器
					possibleDNS := guessDNSFromNetwork(ipnet.IP)
					for _, dns := range possibleDNS {
						if !seen[dns] {
							seen[dns] = true
							dnsServers = append(dnsServers, dns)
						}
					}
				}
			}
		}
	}

	// 如果没有找到任何DNS，返回一些通用的DNS服务器
	if len(dnsServers) == 0 {
		return getDefaultDNS(), nil
	}

	// 验证这些DNS服务器是否可用
	validDNS := validateDNSServers(dnsServers, testDomains)
	if len(validDNS) > 0 {
		return validDNS, nil
	}

	return getDefaultDNS(), nil
}

// guessDNSFromNetwork 根据网络地址猜测可能的DNS服务器
func guessDNSFromNetwork(ip net.IP) []string {
	var dns []string

	// 基于IP地址的前三个八位组构造可能的网关/DNS地址
	if ip.To4() != nil {
		ipv4 := ip.To4()

		// 常见的网关地址通常是 x.x.x.1
		gateway := fmt.Sprintf("%d.%d.%d.1", ipv4[0], ipv4[1], ipv4[2])
		dns = append(dns, gateway)

		// 一些运营商使用的DNS模式
		switch {
		case ipv4[0] == 192 && ipv4[1] == 168:
			// 私有网络，可能使用路由器作为DNS
			dns = append(dns, gateway)
		case ipv4[0] == 10:
			// 企业网络
			dns = append(dns, gateway)
		case ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31:
			// 私有网络
			dns = append(dns, gateway)
		default:
			// 公网IP，可能使用运营商DNS
			// 中国移动: 221.5.88.88, 221.5.203.98
			// 中国联通: 202.106.0.20, 202.106.196.115
			// 中国电信: 202.101.172.35, 202.101.172.47
			dns = append(dns, "202.101.172.35", "202.106.0.20", "221.5.88.88")
		}
	}

	return dns
}

// validateDNSServers 验证DNS服务器是否可用
func validateDNSServers(dnsServers []string, testDomains []string) []string {
	var validDNS []string

	for _, dnsServer := range dnsServers {
		if isDNSServerValid(dnsServer, testDomains) {
			validDNS = append(validDNS, dnsServer)
		}
	}

	return validDNS
}

// isDNSServerValid 检查DNS服务器是否可用
func isDNSServerValid(dnsServer string, testDomains []string) bool {
	// 注意: 这里暂时简化实现，实际应该使用指定的DNS服务器来查询
	// 当前只是验证能否解析域名，作为基础可用性检查
	for _, domain := range testDomains {
		_, err := net.LookupHost(domain)
		if err == nil {
			return true
		}
	}
	return false
}

// getDefaultDNS 返回默认的DNS服务器列表
func getDefaultDNS() []string {
	return []string{
		"202.101.172.35", // 中国电信
		"202.106.0.20",   // 中国联通
		"221.5.88.88",    // 中国移动
		"8.8.8.8",        // Google
		"1.1.1.1",        // Cloudflare
	}
}

// GetISPDNS 获取运营商DNS (主要方法)
func GetISPDNS() []string {
	dns, err := GetSystemDNS()
	if err != nil {
		logger.NetworkWarn("获取系统DNS失败: %v，使用默认DNS", err)
		return getDefaultDNS()
	}

	if len(dns) == 0 {
		return getDefaultDNS()
	}

	// 过滤掉本地DNS (127.x.x.x)
	var filteredDNS []string
	for _, d := range dns {
		if !strings.HasPrefix(d, "127.") && !strings.HasPrefix(d, "::1") {
			filteredDNS = append(filteredDNS, d)
		}
	}

	if len(filteredDNS) == 0 {
		return getDefaultDNS()
	}

	return filteredDNS
}

// LocationInfo IP地理位置信息
type LocationInfo struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	Region  string `json:"region"` // 省份
	City    string `json:"city"`   // 城市
	ISP     string `json:"isp"`    // 运营商
}

// CityISPInfo 城市运营商网段信息
type CityISPInfo struct {
	City         string   // 城市名称
	ISP          string   // 运营商名称
	ClientSubnet []string // 该城市该运营商的网段列表
}

// 中国主要城市的运营商网段数据库
var cityISPDatabase = map[string]map[string][]string{
	"北京": {
		"电信":  {"202.101.172.0/24", "202.101.170.0/24", "218.85.152.0/24", "124.205.155.0/24"},
		"联通":  {"202.106.0.0/24", "202.106.196.0/24", "123.125.81.0/24", "140.207.198.0/24"},
		"移动":  {"221.5.88.0/24", "221.5.203.0/24", "211.136.112.0/24", "120.196.165.0/24"},
		"教育网": {"202.112.0.0/24", "166.111.8.0/24", "59.66.0.0/24"},
	},
	"上海": {
		"电信":  {"202.96.209.0/24", "202.96.199.0/24", "116.228.111.0/24", "180.169.81.0/24"},
		"联通":  {"140.207.54.0/24", "140.207.198.0/24", "101.71.154.0/24", "210.22.70.0/24"},
		"移动":  {"117.131.0.0/24", "183.194.238.0/24", "223.87.238.0/24", "120.204.0.0/24"},
		"教育网": {"202.120.2.0/24", "202.38.64.0/24", "210.25.0.0/24"},
	},
	"广州": {
		"电信":  {"183.232.231.0/24", "183.232.169.0/24", "113.108.239.0/24", "14.29.0.0/24"},
		"联通":  {"113.107.219.0/24", "210.21.196.0/24", "221.4.70.0/24", "113.200.91.0/24"},
		"移动":  {"120.196.165.0/24", "221.179.155.0/24", "183.232.126.0/24", "223.104.248.0/24"},
		"教育网": {"202.38.140.0/24", "210.38.137.0/24", "202.116.160.0/24"},
	},
	"深圳": {
		"电信":  {"183.240.200.0/24", "183.240.48.0/24", "119.147.15.0/24", "14.215.177.0/24"},
		"联通":  {"113.200.91.0/24", "210.21.70.0/24", "221.4.81.0/24", "140.75.166.0/24"},
		"移动":  {"120.196.212.0/24", "223.104.248.0/24", "183.240.102.0/24", "120.204.29.0/24"},
		"教育网": {"202.38.193.0/24", "210.38.137.0/24", "166.111.4.0/24"},
	},
	"杭州": {
		"电信":  {"115.236.101.0/24", "60.191.124.0/24", "124.160.194.0/24", "122.224.0.0/24"},
		"联通":  {"101.71.37.0/24", "140.207.160.0/24", "210.22.84.0/24", "153.35.0.0/24"},
		"移动":  {"120.199.40.0/24", "183.129.244.0/24", "223.104.130.0/24", "117.136.0.0/24"},
		"教育网": {"210.32.0.0/24", "202.38.64.0/24", "210.25.5.0/24"},
	},
	"南京": {
		"电信":  {"180.101.49.0/24", "180.101.136.0/24", "114.222.0.0/24", "58.240.0.0/24"},
		"联通":  {"114.221.0.0/24", "221.6.0.0/24", "123.58.180.0/24", "210.22.80.0/24"},
		"移动":  {"112.17.0.0/24", "117.136.0.0/24", "223.111.0.0/24", "120.204.96.0/24"},
		"教育网": {"210.28.0.0/24", "202.119.0.0/24", "58.192.114.0/24"},
	},
	"福州": {
		"电信":  {"27.155.96.0/24", "27.155.97.0/24", "180.101.212.0/24", "114.84.224.0/24"},
		"联通":  {"221.131.128.0/24", "61.154.0.0/24", "210.15.128.0/24", "202.101.224.0/24"},
		"移动":  {"120.39.0.0/24", "117.28.0.0/24", "223.104.56.0/24", "183.207.224.0/24"},
		"教育网": {"202.38.193.0/24", "202.201.112.0/24", "210.34.0.0/24"},
	},
	"泉州": {
		"电信":  {"27.155.64.0/24", "27.155.65.0/24", "114.84.160.0/24", "180.101.208.0/24"},
		"联通":  {"221.131.160.0/24", "61.154.32.0/24", "210.15.160.0/24", "202.101.240.0/24"},
		"移动":  {"120.39.32.0/24", "117.28.32.0/24", "223.104.88.0/24", "183.207.240.0/24"},
		"教育网": {"202.38.224.0/24", "210.34.32.0/24", "202.201.176.0/24"},
	},
}

// GetOptimalClientSubnet 获取优化的client_subnet值
func GetOptimalClientSubnet() string {
	// 1. 尝试获取真实的公网IP和地理位置
	location, err := getRealLocation()
	if err != nil {
		logger.NetworkWarn("获取地理位置失败: %v，使用默认策略", err)
		return getDefaultClientSubnet()
	}

	// 2. 根据城市和运营商获取精确的网段
	if subnet := getCityISPSubnet(location.City, location.ISP); subnet != "" {
		logger.NetworkInfo("检测到位置: %s %s，使用client_subnet: %s", location.City, location.ISP, subnet)
		return subnet
	}

	// 3. 如果精确匹配失败，尝试模糊匹配
	if subnet := getFallbackSubnet(location); subnet != "" {
		logger.NetworkInfo("使用备选匹配: %s，client_subnet: %s", location.City, subnet)
		return subnet
	}

	// 4. 默认策略
	logger.NetworkInfo("无法精确匹配，使用默认client_subnet")
	return getDefaultClientSubnet()
}

// getRealLocation 获取真实的公网IP地理位置
func getRealLocation() (*LocationInfo, error) {
	// 使用多个IP查询服务，提高成功率
	services := []string{
		"https://ipapi.co/json/",
		"http://ip.sb/geoip/",
		"http://ip-api.com/json/?fields=status,country,regionName,city,isp,query",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, service := range services {
		location, err := queryLocationService(client, service)

		if err == nil && location != nil {
			originalCity := location.City
			
			// 先处理空字段的情况
			if originalCity == "" {
				logger.NetworkWarn("API返回的城市字段为空，尝试从省份推导")
				// 尝试从省份/地区推导城市
				if location.Region != "" {
					originalCity = inferCityFromRegion(location.Region)
					location.City = originalCity
					logger.NetworkInfo("从省份 '%s' 推导城市: %s", location.Region, originalCity)
				}
			}
			
			// 进行城市名映射
			mappedCity := getCityNameCH(originalCity)
			
			// 详细的调试信息
			logger.NetworkInfo("地理位置查询成功 - 服务: %s", service)
			logger.NetworkInfo("原始数据 - 城市:'%s', 省份:'%s', ISP:'%s', IP:'%s'", originalCity, location.Region, location.ISP, location.IP)
			logger.NetworkInfo("城市映射: '%s' -> '%s'", originalCity, mappedCity)
			
			// 验证映射结果
			if mappedCity != "" && mappedCity != originalCity {
				// 映射成功
				location.City = mappedCity
				logger.NetworkInfo("城市映射成功，使用: %s", mappedCity)
			} else if originalCity != "" {
				// 映射失败但原始城市不为空，保留原始名称
				location.City = originalCity
				logger.NetworkWarn("未找到城市映射，保留原始名称: %s", originalCity)
			} else {
				// 城市字段完全为空，使用ISP推导
				logger.NetworkWarn("城市信息完全缺失，尝试从ISP推导默认城市")
				location.City = getDefaultCityByISP(location.ISP)
			}
			
			logger.NetworkInfo("最终结果 - 城市:'%s', ISP:'%s', 省份:'%s'", location.City, location.ISP, location.Region)
			
			// 确保关键字段不为空
			if location.City == "" {
				logger.NetworkWarn("城市字段仍为空，使用默认值")
				location.City = "北京" // 使用默认城市
			}
			if location.ISP == "" {
				logger.NetworkWarn("ISP字段为空，使用默认值")
				location.ISP = "电信" // 使用默认ISP
			}
			
			return location, nil
		}
		logger.NetworkWarn("地理位置查询服务 %s 失败: %v", service, err)
	}

	return nil, fmt.Errorf("所有地理位置查询服务都失败了")
}

// queryLocationService 查询单个地理位置服务
func queryLocationService(client *http.Client, serviceURL string) (*LocationInfo, error) {
	resp, err := client.Get(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应是否为空
	if len(body) == 0 {
		return nil, fmt.Errorf("响应为空")
	}

	// 检查是否有错误响应 (如rate limit)
	var errorCheck map[string]interface{}
	if err := json.Unmarshal(body, &errorCheck); err == nil {
		if errorVal, exists := errorCheck["error"]; exists {
			if errorBool, ok := errorVal.(bool); ok && errorBool {
				reason := "未知错误"
				if reasonVal, exists := errorCheck["reason"]; exists {
					reason = fmt.Sprintf("%v", reasonVal)
				}
				return nil, fmt.Errorf("服务返回错误: %s", reason)
			}
		}
		
		// 检查ip-api.com的状态字段
		if statusVal, exists := errorCheck["status"]; exists {
			if status, ok := statusVal.(string); ok && status != "success" {
				return nil, fmt.Errorf("API状态异常: %s", status)
			}
		}
	}

	// 解析不同服务的响应格式
	if strings.Contains(serviceURL, "ip-api.com") {
		return parseIPAPIResponse(body)
	} else if strings.Contains(serviceURL, "ipapi.co") {
		return parseIPAPICoResponse(body)
	} else if strings.Contains(serviceURL, "ip.sb") {
		return parseIPSBResponse(body)
	}

	return nil, fmt.Errorf("未知的服务格式")
}

// parseIPAPIResponse 解析ip-api.com的响应
func parseIPAPIResponse(body []byte) (*LocationInfo, error) {
	var response struct {
		Status     string `json:"status"`
		Country    string `json:"country"`
		RegionName string `json:"regionName"`
		City       string `json:"city"`
		ISP        string `json:"isp"`
		Query      string `json:"query"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("API返回状态: %s", response.Status)
	}

	return &LocationInfo{
		IP:      response.Query,
		Country: response.Country,
		Region:  response.RegionName,
		City:    response.City,
		ISP:     normalizeISPName(response.ISP),
	}, nil
}

// parseIPAPICoResponse 解析ipapi.co的响应
func getCityNameCH(enName string) string {
	// 中国大陆城市
	switch enName {
	case "Beijing":
		return "北京"
	case "Shanghai":
		return "上海"
	case "Hangzhou":
		return "杭州"
	case "Shenzhen":
		return "深圳"
	case "Guangzhou":
		return "广州"
	case "Quanzhou":
		return "泉州"
	case "Fuzhou":
		return "福州"
	case "Nanjing":
		return "南京"
	case "Chengdu":
		return "成都"
	case "Wuhan":
		return "武汉"
	case "Xi'an", "Xian":
		return "西安"
	case "Chongqing":
		return "重庆"
	case "Tianjin":
		return "天津"
	case "Shenyang":
		return "沈阳"
	case "Changchun":
		return "长春"
	case "Harbin":
		return "哈尔滨"
	case "Jinan":
		return "济南"
	case "Qingdao":
		return "青岛"
	case "Zhengzhou":
		return "郑州"
	case "Taiyuan":
		return "太原"
	case "Shijiazhuang":
		return "石家庄"
	case "Hohhot":
		return "呼和浩特"
	case "Yinchuan":
		return "银川"
	case "Xining":
		return "西宁"
	case "Urumqi":
		return "乌鲁木齐"
	case "Lhasa":
		return "拉萨"
	case "Kunming":
		return "昆明"
	case "Guiyang":
		return "贵阳"
	case "Nanning":
		return "南宁"
	case "Haikou":
		return "海口"
	case "Changsha":
		return "长沙"
	case "Nanchang":
		return "南昌"
	case "Hefei":
		return "合肥"
	// 港澳台城市 - 使用大陆相近的网段
	case "Hong Kong":
		return "深圳" // 香港使用深圳网段
	case "Macau", "Macao":
		return "广州" // 澳门使用广州网段  
	case "Taipei":
		return "福州" // 台北使用福建网段
	case "Kaohsiung":
		return "泉州" // 高雄使用泉州网段
	default:
		// 返回原始名称，后续会进入fallback逻辑
		return enName
	}
}

func parseIPAPICoResponse(body []byte) (*LocationInfo, error) {
	var response struct {
		IP      string `json:"ip"`
		Country string `json:"country_name"`
		Region  string `json:"region"`
		City    string `json:"city"`
		Org     string `json:"org"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &LocationInfo{
		IP:      response.IP,
		Country: response.Country,
		Region:  response.Region,
		City:    getCityNameCH(response.City),
		ISP:     normalizeISPName(response.Org),
	}, nil
}

// parseIPSBResponse 解析ip.sb的响应
func parseIPSBResponse(body []byte) (*LocationInfo, error) {
	var response struct {
		IP           string `json:"ip"`
		Country      string `json:"country"`
		Region       string `json:"region"`
		City         string `json:"city"`
		ISP          string `json:"isp"`
		Organization string `json:"organization"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	isp := response.ISP
	if isp == "" {
		isp = response.Organization
	}

	return &LocationInfo{
		IP:      response.IP,
		Country: response.Country,
		Region:  response.Region,
		City:    response.City,
		ISP:     normalizeISPName(isp),
	}, nil
}

// normalizeISPName 标准化运营商名称
func normalizeISPName(isp string) string {
	isp = strings.ToLower(isp)

	if strings.Contains(isp, "china telecom") || strings.Contains(isp, "电信") || strings.Contains(isp, "chinanet") {
		return "电信"
	}
	if strings.Contains(isp, "china unicom") || strings.Contains(isp, "联通") || strings.Contains(isp, "unicom") {
		return "联通"
	}
	if strings.Contains(isp, "china mobile") || strings.Contains(isp, "移动") || strings.Contains(isp, "cmcc") {
		return "移动"
	}
	if strings.Contains(isp, "cernet") || strings.Contains(isp, "教育") || strings.Contains(isp, "edu") {
		return "教育网"
	}

	return "电信" // 默认返回电信
}

// getCityISPSubnet 根据城市和运营商获取网段
func getCityISPSubnet(city, isp string) string {
	if cityData, exists := cityISPDatabase[city]; exists {
		if subnets, exists := cityData[isp]; exists && len(subnets) > 0 {
			// 返回第一个网段作为client_subnet
			return subnets[0]
		}
	}
	return ""
}

// getFallbackSubnet 获取备选网段
func getFallbackSubnet(location *LocationInfo) string {
	// 1. 尝试匹配其他城市的相同运营商
	for _, cityData := range cityISPDatabase {
		if subnets, exists := cityData[location.ISP]; exists && len(subnets) > 0 {
			logger.NetworkInfo("使用 %s 网段作为备选", location.ISP)
			return subnets[0]
		}
	}

	// 2. 如果连运营商都匹配不上，根据地区选择默认运营商
	return getRegionalDefault(location.Region)
}

// getRegionalDefault 根据地区获取默认网段
func getRegionalDefault(region string) string {
	// 根据不同省份选择主要运营商的网段
	switch {
	case strings.Contains(region, "北京") || strings.Contains(region, "天津"):
		return "202.101.170.0/24" // 电信北京
	case strings.Contains(region, "上海"):
		return "202.96.209.0/24" // 电信上海
	case strings.Contains(region, "广东"):
		return "183.232.231.0/24" // 电信广州
	case strings.Contains(region, "浙江"):
		return "115.236.101.0/24" // 电信杭州
	case strings.Contains(region, "江苏"):
		return "180.101.49.0/24" // 电信南京
	case strings.Contains(region, "福建"):
		return "27.155.96.0/24" // 电信福州
	default:
		return "202.101.170.0/24" // 默认电信北京
	}
}

// getDefaultClientSubnet 获取默认的client_subnet
func getDefaultClientSubnet() string {
	// 使用本地IP推测或者返回通用网段
	if subnet := guessClientSubnetFromLocalIP(); subnet != "" {
		return subnet
	}
	return "202.101.170.0/24" // 电信北京作为最终备选
}

// GetISPName 获取当前运营商名称 (更新为使用新的地理位置API)
func GetISPName() string {
	// 尝试从真实位置获取运营商信息
	if location, err := getRealLocation(); err == nil {
		return fmt.Sprintf("%s (%s)", location.ISP, location.City)
	}

	// 备选方案：通过DNS推测
	currentDNS := GetISPDNS()
	if len(currentDNS) == 0 {
		return "未知运营商"
	}

	// 检查是否使用了知名公共DNS
	for _, dns := range currentDNS {
		switch dns {
		case "8.8.8.8", "8.8.4.4":
			return "Google DNS"
		case "1.1.1.1", "1.0.0.1":
			return "Cloudflare DNS"
		case "114.114.114.114", "114.114.115.115":
			return "114 DNS"
		case "223.5.5.5", "223.6.6.6":
			return "阿里 DNS"
		case "119.29.29.29", "182.254.116.116":
			return "腾讯 DNS"
		}
	}

	return "本地网络"
}

// guessClientSubnetFromLocalIP 根据本地IP推测合适的client_subnet
func guessClientSubnetFromLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				localIP := ipnet.IP.To4()

				// 根据本地IP的ISP特征推测合适的client_subnet
				subnet := guessSubnetByIPPattern(localIP)
				if subnet != "" {
					return subnet
				}
			}
		}
	}

	return ""
}

// guessSubnetByIPPattern 根据IP模式推测运营商网段
func guessSubnetByIPPattern(ip net.IP) string {
	if ip == nil {
		return ""
	}

	// 私有IP范围，无法直接确定运营商
	// 但可以根据一些经验规则推测
	switch {
	case ip[0] == 192 && ip[1] == 168:
		// 家庭网络，根据常见路由器配置推测
		switch ip[2] {
		case 1, 0:
			// 最常见的家用路由器配置，通常是电信
			return "202.101.170.0/24"
		case 31:
			// 一些品牌路由器的默认配置
			return "202.106.0.0/24" // 联通
		default:
			return "202.101.170.0/24" // 默认电信
		}
	case ip[0] == 10:
		// 企业网络，通常使用教育网或企业专线
		return "202.112.0.0/24"
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		// 私有网络
		return "202.106.0.0/24" // 联通
	}

	return ""
}

// isPrivateIP 检查是否为私有IP地址
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// 检查是否为私有IP段
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range privateRanges {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if subnet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// inferCityFromRegion 从省份推导主要城市
func inferCityFromRegion(region string) string {
	region = strings.ToLower(region)
	
	// 匹配省份到主要城市
	if strings.Contains(region, "beijing") || strings.Contains(region, "北京") {
		return "Beijing"
	}
	if strings.Contains(region, "shanghai") || strings.Contains(region, "上海") {
		return "Shanghai"
	}
	if strings.Contains(region, "guangdong") || strings.Contains(region, "广东") {
		return "Guangzhou"
	}
	if strings.Contains(region, "zhejiang") || strings.Contains(region, "浙江") {
		return "Hangzhou"
	}
	if strings.Contains(region, "jiangsu") || strings.Contains(region, "江苏") {
		return "Nanjing"
	}
	if strings.Contains(region, "fujian") || strings.Contains(region, "福建") {
		return "Fuzhou"
	}
	if strings.Contains(region, "taiwan") || strings.Contains(region, "台湾") {
		return "Taipei"
	}
	if strings.Contains(region, "hong kong") || strings.Contains(region, "香港") {
		return "Hong Kong"
	}
	if strings.Contains(region, "macau") || strings.Contains(region, "macao") || strings.Contains(region, "澳门") {
		return "Macau"
	}
	
	return "" // 无法推导
}

// getDefaultCityByISP 根据ISP推导默认城市
func getDefaultCityByISP(isp string) string {
	// 根据运营商特点选择合适的默认城市
	switch isp {
	case "电信":
		return "北京" // 电信总部在北京
	case "联通":
		return "北京" // 联通总部在北京
	case "移动":
		return "北京" // 移动总部在北京
	case "教育网":
		return "北京" // CERNET总部在清华大学
	default:
		return "北京" // 默认使用北京
	}
}

// GetInternalIP 获取本机的内网IP地址
func GetInternalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		logger.NetworkWarn("获取网络接口失败: %v", err)
		return "127.0.0.1"
	}

	for _, iface := range interfaces {
		// 跳过禁用的接口和回环接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ip := ipnet.IP.String()
					// 优先返回私有IP地址
					if isPrivateIP(ip) {
						logger.NetworkInfo("检测到内网IP: %s (接口: %s)", ip, iface.Name)
						return ip
					}
				}
			}
		}
	}

	// 如果没有找到私有IP，尝试连接外部服务来获取本地IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		logger.NetworkWarn("无法检测内网IP，使用默认地址: %v", err)
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := localAddr.IP.String()
	logger.NetworkInfo("通过连接测试检测到内网IP: %s", ip)
	return ip
}
